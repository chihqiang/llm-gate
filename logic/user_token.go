package logic

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/model"
	"chihqiang/llm-gate/security"

	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

type TokenLogic struct {
	db        *gorm.DB
	authCache cache.Cache
	cipher    *security.Cipher
}

func NewTokenLogic(db *gorm.DB, authCache cache.Cache, cipher *security.Cipher) *TokenLogic {
	return &TokenLogic{db: db, authCache: authCache, cipher: cipher}
}

type TokenListRequest struct {
	Page             int   `form:"page" binding:"required,min=1"`
	Size             int   `form:"size" binding:"required,min=1,max=1000"`
	AccountID        int64 `form:"account_id"`
	CurrentAccountID int64 `form:"-"`
}

type TokenVO struct {
	ID          int64      `json:"id"`
	AccountID   int64      `json:"account_id"`
	Name        string     `json:"name"`
	Key         string     `json:"-"`
	KeyMasked   string     `json:"key_masked"`
	Quota       int64      `json:"quota"`
	SpentCents  int64      `json:"spent_cents"`
	ModelIDs    []int64    `json:"model_ids"`
	Status      bool       `json:"status"`
	ExpiredAt   *time.Time `json:"expired_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TokenCreateResponse struct {
	TokenVO
	Key string `json:"key"`
}

type TokenListResponse struct {
	Data  []TokenVO `json:"data"`
	Total int64     `json:"total"`
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:6] + "****" + key[len(key)-4:]
}

func parseModelIDs(raw string) []int64 {
	if raw == "" {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		logger.Warn("token: parse model_ids failed", logger.String("raw", raw), logger.Err(err))
	}
	return ids
}

func marshalModelIDs(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

func (s *TokenLogic) toTokenVO(t model.UserToken) TokenVO {
	plain, _ := s.cipher.Decrypt(t.Key)
	return TokenVO{
		ID:         t.ID,
		AccountID:  t.AccountID,
		Name:       t.Name,
		Key:        plain,
		KeyMasked:  maskKey(plain),
		Quota:      t.Quota,
		SpentCents: t.SpentCents,
		ModelIDs:   parseModelIDs(t.ModelIDs),
		Status:     t.Status,
		ExpiredAt:  t.ExpiredAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func (s *TokenLogic) List(req *TokenListRequest) (*TokenListResponse, error) {
	var tokens []model.UserToken
	var total int64

	query := s.db.Model(&model.UserToken{})
	if req.CurrentAccountID > 0 {
		query = query.Where("account_id = ?", req.CurrentAccountID)
	} else if req.AccountID > 0 {
		query = query.Where("account_id = ?", req.AccountID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("id DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}

	data := make([]TokenVO, len(tokens))
	for i, t := range tokens {
		data[i] = s.toTokenVO(t)
	}

	return &TokenListResponse{Data: data, Total: total}, nil
}

func (s *TokenLogic) GetByID(id int64) (*TokenVO, error) {
	var token model.UserToken
	if err := s.db.First(&token, id).Error; err != nil {
		return nil, err
	}
	vo := s.toTokenVO(token)
	return &vo, nil
}

type TokenCreateRequest struct {
	AccountID int64   `json:"account_id" binding:"required"`
	Name      string  `json:"name" binding:"required"`
	Quota     int64   `json:"quota"`
	ModelIDs  []int64 `json:"model_ids"`
	ExpiredAt *time.Time `json:"expired_at"`
}

func generateTokenKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("sk-%s", hex.EncodeToString(b)), nil
}

func (s *TokenLogic) Create(req *TokenCreateRequest) (*TokenCreateResponse, error) {
	key, err := generateTokenKey()
	if err != nil {
		return nil, err
	}
	encrypted, err := s.cipher.Encrypt(key)
	if err != nil {
		return nil, err
	}

	token := model.UserToken{
		AccountID: req.AccountID,
		Name:      req.Name,
		Key:       encrypted,
		KeyHash:   security.SHA256Hex(key),
		Quota:     req.Quota,
		ModelIDs:  marshalModelIDs(req.ModelIDs),
		Status:    true,
		ExpiredAt: req.ExpiredAt,
	}
	if err := s.db.Create(&token).Error; err != nil {
		return nil, err
	}
	resp := &TokenCreateResponse{Key: key}
	vo := s.toTokenVO(token)
	resp.TokenVO = vo
	return resp, nil
}

type TokenUpdateRequest struct {
	ID        int64      `json:"id" binding:"required"`
	Name      string     `json:"name" binding:"required"`
	Quota     int64      `json:"quota"`
	Status    bool       `json:"status"`
	ModelIDs  []int64    `json:"model_ids"`
	ExpiredAt *time.Time `json:"expired_at"`
}

func (s *TokenLogic) Update(req *TokenUpdateRequest) (*TokenVO, error) {
	updates := map[string]interface{}{
		"name":      req.Name,
		"quota":     req.Quota,
		"status":    req.Status,
		"model_ids": marshalModelIDs(req.ModelIDs),
		"expired_at": req.ExpiredAt,
	}
	if err := s.db.Model(&model.UserToken{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	// 失效认证缓存，禁用/过期/改预算后立即生效
	token, _ := s.GetByID(req.ID)
	if token != nil && token.Key != "" {
		s.authCache.Del(fmt.Sprintf("auth:%s", token.Key))
	}
	return s.GetByID(req.ID)
}

func (s *TokenLogic) Delete(id int64) error {
	token, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(&model.UserToken{}, id).Error; err != nil {
		return err
	}
	if token != nil && token.Key != "" {
		s.authCache.Del(fmt.Sprintf("auth:%s", token.Key))
	}
	return nil
}

// RevealKey 解密密钥返回给管理员，并记录审计日志。
func (s *TokenLogic) RevealKey(id, accountID int64) (string, error) {
	var token model.UserToken
	if err := s.db.First(&token, id).Error; err != nil {
		return "", fmt.Errorf("token not found")
	}
	if token.AccountID != accountID {
		return "", fmt.Errorf("access denied")
	}
	plain, err := s.cipher.Decrypt(token.Key)
	if err != nil {
		return "", err
	}
	logger.Info("token key revealed", logger.Int64("token_id", id), logger.Int64("operator", accountID))
	return plain, nil
}
