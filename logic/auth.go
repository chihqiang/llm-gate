package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/hash"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

type AuthLogic struct {
	db           *gorm.DB
	j            *jwt.JWT
	accountCache cache.Cache
}

func NewAuthLogic(db *gorm.DB, j *jwt.JWT, accountCache cache.Cache) *AuthLogic {
	return &AuthLogic{
		db:           db,
		j:            j,
		accountCache: accountCache,
	}
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

func (s *AuthLogic) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	var account model.Account
	if err := s.db.WithContext(ctx).Where("email = ?", req.Email).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WarnCtx(ctx, "auth login failed: email not found", logger.String("email", req.Email))
			return nil, errors.New("邮箱或密码错误")
		}
		return nil, err
	}

	if !account.Status {
		logger.WarnCtx(ctx, "auth login failed: account disabled", logger.Int64("account_id", account.ID))
		return nil, errors.New("账号已被禁用")
	}

	if err := hash.BcryptCompare(account.Password, req.Password); err != nil {
		logger.WarnCtx(ctx, "auth login failed: wrong password", logger.Int64("account_id", account.ID))
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

	logger.InfoCtx(ctx, "auth login ok", logger.Int64("account_id", account.ID), logger.String("email", account.Email))
	return &LoginResponse{
		ID:           account.ID,
		AccessToken:  tokenPair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenPair.ExpiresAt,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (s *AuthLogic) Refresh(ctx context.Context, req *RefreshRequest) (*LoginResponse, error) {
	tokenPair, err := s.j.RefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("刷新令牌无效或已过期")
	}

	// 从刷新令牌中提取用户 ID
	claims, err := s.j.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("刷新令牌无效或已过期")
	}

	userID, ok := claims[jwt.ClaimKeyUserID].(float64)
	if !ok {
		return nil, errors.New("刷新令牌无效")
	}

	return &LoginResponse{
		ID:           int64(userID),
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

func (s *AuthLogic) GetProfile(ctx context.Context, accountID int64, isAdmin bool) (*ProfileResponse, error) {
	var account model.Account
	if err := s.db.WithContext(ctx).Preload("Roles", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).First(&account, accountID).Error; err != nil {
		return nil, err
	}

	var menus []model.Menu
	if isAdmin {
		if err := s.db.WithContext(ctx).Where("status = ?", true).Order("sort ASC").Find(&menus).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Preload("Roles.Menus", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", true)
		}).First(&account, accountID).Error; err != nil {
			return nil, err
		}
		seen := make(map[int64]bool)
		for _, role := range account.Roles {
			for _, menu := range role.Menus {
				if !seen[menu.ID] {
					seen[menu.ID] = true
					menus = append(menus, menu)
				}
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

func (s *AuthLogic) GetAccountByID(ctx context.Context, accountID int64) (*model.Account, error) {
	cacheKey := fmt.Sprintf("account:%d", accountID)
	var account model.Account
	if ok, _ := s.accountCache.GetInto(ctx, cacheKey, &account); ok {
		return &account, nil
	}
	logger.InfoCtx(ctx, "auth account cache miss", logger.Int64("account_id", accountID))

	if err := s.db.WithContext(ctx).Preload("Roles", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).Preload("Roles.Menus", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", true)
	}).First(&account, accountID).Error; err != nil {
		return nil, err
	}

	s.accountCache.Set(ctx, cacheKey, &account, 10*time.Second)
	return &account, nil
}
