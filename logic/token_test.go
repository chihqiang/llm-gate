package logic

import (
	"testing"

	"chihqiang/llm-gate/config"
)

func newTestTokenManager(t *testing.T) *TokenManager {
	t.Helper()
	m, err := NewTokenManager(config.JWTConfig{Secret: "test-secret-123", Issuer: "llm-gate"})
	if err != nil {
		t.Fatalf("new token manager: %v", err)
	}
	return m
}

func TestTokenManagerIssueAndParse(t *testing.T) {
	m := newTestTokenManager(t)
	pair, err := m.GenerateTokenPair(Claims{
		ClaimKeyUserID:   int64(7),
		ClaimKeyUsername: "admin@example.com",
	})
	if err != nil {
		t.Fatalf("generate pair: %v", err)
	}

	claims, err := m.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if got, _ := claims[ClaimKeyUserID].(float64); got != 7 {
		t.Fatalf("user_id = %v, want 7", claims[ClaimKeyUserID])
	}
	if got, _ := claims[ClaimKeyUsername].(string); got != "admin@example.com" {
		t.Fatalf("username = %q", got)
	}

	// 刷新令牌不能当作访问令牌使用（签名密钥不同）
	if _, err := m.ParseAccessToken(pair.RefreshToken); err == nil {
		t.Fatal("refresh token accepted as access token")
	}
	// 访问令牌不能当作刷新令牌使用
	if _, err := m.ParseRefreshToken(pair.AccessToken); err == nil {
		t.Fatal("access token accepted as refresh token")
	}
}

func TestTokenManagerRefreshRotates(t *testing.T) {
	m := newTestTokenManager(t)
	pair, err := m.GenerateTokenPair(Claims{ClaimKeyUserID: int64(9)})
	if err != nil {
		t.Fatalf("generate pair: %v", err)
	}

	next, err := m.RefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	claims, err := m.ParseAccessToken(next.AccessToken)
	if err != nil {
		t.Fatalf("parse refreshed access: %v", err)
	}
	if got, _ := claims[ClaimKeyUserID].(float64); got != 9 {
		t.Fatalf("refreshed user_id = %v, want 9", claims[ClaimKeyUserID])
	}
}

func TestTokenManagerRejectsBadConfig(t *testing.T) {
	if _, err := NewTokenManager(config.JWTConfig{Secret: "short"}); err == nil {
		t.Fatal("short secret must be rejected")
	}
	if _, err := NewTokenManager(config.JWTConfig{Secret: "test-secret-123", Algorithm: "HS512"}); err == nil {
		t.Fatal("non-HS256 algorithm must be rejected")
	}
}

func TestTokenManagerIssuerMismatch(t *testing.T) {
	m := newTestTokenManager(t)
	pair, err := m.GenerateTokenPair(Claims{ClaimKeyUserID: int64(1)})
	if err != nil {
		t.Fatalf("generate pair: %v", err)
	}
	other, err := NewTokenManager(config.JWTConfig{Secret: "test-secret-123", Issuer: "another"})
	if err != nil {
		t.Fatalf("new other manager: %v", err)
	}
	if _, err := other.ParseAccessToken(pair.AccessToken); err == nil {
		t.Fatal("token with a different issuer must be rejected")
	}
}
