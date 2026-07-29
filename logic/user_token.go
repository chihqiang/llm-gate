package logic

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type TokenLogic struct {
	db *gorm.DB
}

func NewTokenLogic(db *gorm.DB) *TokenLogic {
	return &TokenLogic{db: db}
}

type TokenListRequest struct {
	Page      int   `form:"page" binding:"required,min=1"`
	Size      int   `form:"size" binding:"required,min=1,max=1000"`
	AccountID int64 `form:"account_id"`
}

type TokenVO struct {
	ID        int64      `json:"id"`
	AccountID int64      `json:"account_id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"`
	KeyMasked string     `json:"key_masked"`
	Quota     int64      `json:"quota"`
	Status    bool       `json:"status"`
	ExpiredAt *time.Time `json:"expired_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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

func toTokenVO(t model.UserToken) TokenVO {
	return TokenVO{
		ID:        t.ID,
		AccountID: t.AccountID,
		Name:      t.Name,
		Key:       t.Key,
		KeyMasked: maskKey(t.Key),
		Quota:     t.Quota,
		Status:    t.Status,
		ExpiredAt: t.ExpiredAt,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func (s *TokenLogic) List(req *TokenListRequest) (*TokenListResponse, error) {
	var tokens []model.UserToken
	var total int64

	query := s.db.Model(&model.UserToken{})
	if req.AccountID > 0 {
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
		data[i] = toTokenVO(t)
	}

	return &TokenListResponse{Data: data, Total: total}, nil
}

func (s *TokenLogic) GetByID(id int64) (*TokenVO, error) {
	var token model.UserToken
	if err := s.db.First(&token, id).Error; err != nil {
		return nil, err
	}
	vo := toTokenVO(token)
	return &vo, nil
}

func (s *TokenLogic) GetByKey(key string) (*model.UserToken, error) {
	var token model.UserToken
	if err := s.db.Where("key = ? AND status = ?", key, true).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

type TokenCreateRequest struct {
	AccountID int64  `json:"account_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Quota     int64  `json:"quota"`
}

func generateTokenKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("sk-%s", hex.EncodeToString(b)), nil
}

func (s *TokenLogic) Create(req *TokenCreateRequest) (*TokenVO, error) {
	key, err := generateTokenKey()
	if err != nil {
		return nil, err
	}

	token := model.UserToken{
		AccountID: req.AccountID,
		Name:      req.Name,
		Key:       key,
		Quota:     req.Quota,
		Status:    true,
	}
	if err := s.db.Create(&token).Error; err != nil {
		return nil, err
	}
	vo := toTokenVO(token)
	return &vo, nil
}

type TokenUpdateRequest struct {
	ID     int64  `json:"id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Quota  int64  `json:"quota"`
	Status bool   `json:"status"`
}

func (s *TokenLogic) Update(req *TokenUpdateRequest) (*TokenVO, error) {
	updates := map[string]interface{}{
		"name":   req.Name,
		"quota":  req.Quota,
		"status": req.Status,
	}
	if err := s.db.Model(&model.UserToken{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(req.ID)
}

func (s *TokenLogic) Delete(id int64) error {
	return s.db.Delete(&model.UserToken{}, id).Error
}

func (s *TokenLogic) RevealKey(id, accountID int64) (string, error) {
	var token model.UserToken
	if err := s.db.First(&token, id).Error; err != nil {
		return "", fmt.Errorf("token not found")
	}
	if token.AccountID != accountID {
		return "", fmt.Errorf("access denied")
	}
	return token.Key, nil
}

func (s *TokenLogic) DeductQuota(id int64, amount int64) error {
	return s.db.Model(&model.UserToken{}).Where("id = ? AND quota >= ?", id, amount).
		UpdateColumn("quota", gorm.Expr("quota - ?", amount)).Error
}

func (s *TokenLogic) RefundQuota(id int64, amount int64) error {
	return s.db.Model(&model.UserToken{}).Where("id = ?", id).
		UpdateColumn("quota", gorm.Expr("quota + ?", amount)).Error
}

func (s *TokenLogic) CleanupExpired() error {
	return s.db.Where("expired_at IS NOT NULL AND expired_at < ?", time.Now()).Delete(&model.UserToken{}).Error
}
