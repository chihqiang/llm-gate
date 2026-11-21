package logic

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/chihqiang/ten-go/rest/token"

	"chihqiang/llm-gate/config"
)

// 令牌类型、标准声明与业务声明键。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"

	ClaimKeyUserID   = "user_id"
	ClaimKeyUsername = "username"

	claimSubject   = "sub"
	claimIssuer    = "iss"
	claimAudience  = "aud"
	claimIssuedAt  = "iat"
	claimExpiresAt = "exp"
	claimTokenType = "token_type"

	minSecretLength = 8
)

// 令牌校验错误。
var (
	ErrInvalidToken    = errors.New("token: invalid token")
	ErrExpiredToken    = errors.New("token: token is expired")
	ErrNotRefreshToken = errors.New("token: token is not a refresh token")
)

// Claims 是校验后的 JWT 载荷（ten-go rest/token 的 MapClaims）。
type Claims = jwtlib.MapClaims

// TokenPair 访问令牌 + 刷新令牌。
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// TokenManager 基于 ten-go rest/token 的令牌签发与校验。
//
// ten-go 的 Parser 只负责 HS256 签名/验签与 exp 校验，业务声明（user_id、
// username、token_type）由这里拼装；access 与 refresh 使用由主密钥派生的
// 不同签名密钥，因此刷新令牌不能当作访问令牌使用。
type TokenManager struct {
	access        *token.Parser
	refresh       *token.Parser
	accessSecret  string
	issuer        string
	audience      []string
	accessExpire  time.Duration
	refreshExpire time.Duration
}

// NewTokenManager 按配置创建令牌管理器。
func NewTokenManager(cfg config.JWTConfig) (*TokenManager, error) {
	if len(cfg.Secret) < minSecretLength {
		return nil, fmt.Errorf("token: secret must be at least %d characters", minSecretLength)
	}
	if cfg.Algorithm != "" && cfg.Algorithm != "HS256" {
		return nil, fmt.Errorf("token: ten-go rest/token only supports HS256, got %q", cfg.Algorithm)
	}
	accessExpire := cfg.AccessTokenExpire
	if accessExpire <= 0 {
		accessExpire = 2 * time.Hour
	}
	refreshExpire := cfg.RefreshTokenExpire
	if refreshExpire <= 0 {
		refreshExpire = 168 * time.Hour
	}
	accessSecret := deriveSecret(cfg.Secret, TokenTypeAccess)
	return &TokenManager{
		access:        token.NewParser(token.Config{Secret: accessSecret}),
		refresh:       token.NewParser(token.Config{Secret: deriveSecret(cfg.Secret, TokenTypeRefresh)}),
		accessSecret:  accessSecret,
		issuer:        cfg.Issuer,
		audience:      cfg.Audience,
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}, nil
}

// AccessSecret 返回访问令牌签名密钥，供 rest/handler.AuthHandler 校验签名。
func (m *TokenManager) AccessSecret() string { return m.accessSecret }

// AccessExpire 返回访问令牌有效期。
func (m *TokenManager) AccessExpire() time.Duration { return m.accessExpire }

// RefreshExpire 返回刷新令牌有效期。
func (m *TokenManager) RefreshExpire() time.Duration { return m.refreshExpire }

// GenerateAccessToken 签发访问令牌。
func (m *TokenManager) GenerateAccessToken(claims Claims) (string, error) {
	return m.access.Generate(m.buildClaims(claims, m.accessExpire, TokenTypeAccess))
}

// GenerateRefreshToken 签发刷新令牌。
func (m *TokenManager) GenerateRefreshToken(claims Claims) (string, error) {
	return m.refresh.Generate(m.buildClaims(claims, m.refreshExpire, TokenTypeRefresh))
}

// GenerateTokenPair 同时签发访问令牌与刷新令牌。
func (m *TokenManager) GenerateTokenPair(claims Claims) (*TokenPair, error) {
	now := time.Now()
	access, err := m.GenerateAccessToken(claims)
	if err != nil {
		return nil, fmt.Errorf("token: generate access token: %w", err)
	}
	refresh, err := m.GenerateRefreshToken(claims)
	if err != nil {
		return nil, fmt.Errorf("token: generate refresh token: %w", err)
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    now.Add(m.accessExpire).Unix(),
	}, nil
}

// ParseAccessToken 校验访问令牌。
func (m *TokenManager) ParseAccessToken(raw string) (Claims, error) {
	claims, err := m.parse(m.access, raw)
	if err != nil {
		return nil, err
	}
	if typ, _ := claims[claimTokenType].(string); typ != TokenTypeAccess {
		return nil, fmt.Errorf("%w: expected access token, got %q", ErrInvalidToken, typ)
	}
	return claims, nil
}

// ParseRefreshToken 校验刷新令牌。
func (m *TokenManager) ParseRefreshToken(raw string) (Claims, error) {
	claims, err := m.parse(m.refresh, raw)
	if err != nil {
		return nil, err
	}
	if typ, _ := claims[claimTokenType].(string); typ != TokenTypeRefresh {
		return nil, ErrNotRefreshToken
	}
	return claims, nil
}

// RefreshToken 用刷新令牌换取新的令牌对，业务声明从旧令牌继承。
func (m *TokenManager) RefreshToken(raw string) (*TokenPair, error) {
	claims, err := m.ParseRefreshToken(raw)
	if err != nil {
		return nil, err
	}
	return m.GenerateTokenPair(extractBusinessClaims(claims))
}

// buildClaims 复制业务声明并补齐标准声明（sub/iat/exp/iss/aud/token_type）。
func (m *TokenManager) buildClaims(src Claims, expire time.Duration, tokenType string) jwtlib.MapClaims {
	now := time.Now()
	claims := make(Claims, len(src)+6)
	for k, v := range src {
		claims[k] = v
	}
	if sub, _ := claims[claimSubject].(string); sub == "" {
		claims[claimSubject] = subjectOf(claims)
	}
	claims[claimIssuedAt] = now.Unix()
	claims[claimExpiresAt] = now.Add(expire).Unix()
	claims[claimTokenType] = tokenType
	if m.issuer != "" {
		claims[claimIssuer] = m.issuer
	}
	if len(m.audience) > 0 {
		claims[claimAudience] = m.audience
	}
	return claims
}

// parse 校验签名/有效期，并补做 ten-go Parser 未覆盖的 iss/aud 校验。
func (m *TokenManager) parse(parser *token.Parser, raw string) (Claims, error) {
	claims, err := parser.Parse(raw)
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if m.issuer != "" {
		if iss, _ := claims[claimIssuer].(string); iss != m.issuer {
			return nil, ErrInvalidToken
		}
	}
	if len(m.audience) > 0 && !hasAudience(claims[claimAudience], m.audience) {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func subjectOf(claims Claims) string {
	switch v := claims[ClaimKeyUserID].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case uint:
		return fmt.Sprintf("%d", v)
	default:
		if name, _ := claims[ClaimKeyUsername].(string); name != "" {
			return name
		}
		return "unknown"
	}
}

func hasAudience(claim any, want []string) bool {
	for _, target := range want {
		switch aud := claim.(type) {
		case string:
			if aud == target {
				return true
			}
		case []string:
			for _, v := range aud {
				if v == target {
					return true
				}
			}
		case []any:
			for _, item := range aud {
				if v, _ := item.(string); v == target {
					return true
				}
			}
		}
	}
	return false
}

// standardClaimKeys 是标准声明与 token_type，刷新换发时会被剔除。
var standardClaimKeys = map[string]bool{
	claimSubject: true, claimIssuer: true, claimAudience: true,
	claimIssuedAt: true, claimExpiresAt: true, claimTokenType: true,
	"nbf": true, "jti": true,
}

func extractBusinessClaims(claims Claims) Claims {
	out := make(Claims)
	for k, v := range claims {
		if !standardClaimKeys[k] {
			out[k] = v
		}
	}
	return out
}

func deriveSecret(secret, scope string) string {
	sum := sha256.Sum256([]byte(secret + ":" + scope))
	return hex.EncodeToString(sum[:])
}
