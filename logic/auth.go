package logic

import (
	"errors"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/hash"
	"github.com/chihqiang/infra-go/jwt"
	"gorm.io/gorm"
)

type AuthLogic struct {
	db *gorm.DB
	j  *jwt.JWT
}

func NewAuthLogic(db *gorm.DB, j *jwt.JWT) *AuthLogic {
	return &AuthLogic{db: db, j: j}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	ID           int64  `json:"id"`
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (s *AuthLogic) Login(req *LoginRequest) (*LoginResponse, error) {
	var account model.Account
	if err := s.db.Where("email = ?", req.Email).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("邮箱或密码错误")
		}
		return nil, err
	}

	if !account.Status {
		return nil, errors.New("账号已被禁用")
	}

	if err := hash.BcryptCompare(account.Password, req.Password); err != nil {
		return nil, errors.New("邮箱或密码错误")
	}

	claims := jwt.Claims{
		jwt.ClaimKeyUserID:   account.ID,
		jwt.ClaimKeyUsername: account.Email,
	}

	tokenPair, err := s.j.GenerateTokenPair(claims)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		ID:           account.ID,
		AccessToken:  tokenPair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenPair.ExpiresAt,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

type ProfileResponse struct {
	ID     int64        `json:"id"`
	Name   string       `json:"name"`
	Email  string       `json:"email"`
	Status bool         `json:"status"`
	Menus  []model.Menu `json:"menus"`
}

func (s *AuthLogic) GetProfile(accountID int64) (*ProfileResponse, error) {
	var account model.Account
	if err := s.db.Preload("Roles", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).Preload("Roles.Menus", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).First(&account, accountID).Error; err != nil {
		return nil, err
	}

	seen := make(map[int64]bool)
	var menus []model.Menu
	for _, role := range account.Roles {
		for _, menu := range role.Menus {
			if !seen[menu.ID] {
				seen[menu.ID] = true
				menus = append(menus, menu)
			}
		}
	}

	return &ProfileResponse{
		ID:     account.ID,
		Name:   account.Name,
		Email:  account.Email,
		Status: account.Status,
		Menus:  menus,
	}, nil
}

func (s *AuthLogic) GetAccountByID(accountID int64) (*model.Account, error) {
	var account model.Account
	if err := s.db.Preload("Roles", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).Preload("Roles.Menus", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).First(&account, accountID).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
