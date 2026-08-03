package logic

import (
	"context"
	"fmt"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/hash"
	"gorm.io/gorm"
)

type AccountLogic struct {
	db           *gorm.DB
	accountCache cache.Cache
}

func NewAccountLogic(db *gorm.DB, accountCache cache.Cache) *AccountLogic {
	return &AccountLogic{db: db, accountCache: accountCache}
}

type AccountListRequest struct {
	Page             int   `form:"page" binding:"required,min=1"`
	Size             int   `form:"size" binding:"required,min=1,max=1000"`
	ID               int   `form:"id"`
	CurrentAccountID int64 `form:"-"`
}

type AccountListResponse struct {
	Data  []model.Account `json:"data"`
	Total int64           `json:"total"`
}

func (s *AccountLogic) List(ctx context.Context, req *AccountListRequest) (*AccountListResponse, error) {
	var accounts []model.Account
	var total int64

	query := s.db.WithContext(ctx).Model(&model.Account{})
	if req.CurrentAccountID > 0 {
		query = query.Where("id = ?", req.CurrentAccountID)
	} else if req.ID > 0 {
		query = query.Where("id = ?", req.ID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Preload("Roles").Offset(offset).Limit(req.Size).Order("id ASC").Find(&accounts).Error; err != nil {
		return nil, err
	}

	return &AccountListResponse{Data: accounts, Total: total}, nil
}

func (s *AccountLogic) GetByID(ctx context.Context, id int64) (*model.Account, error) {
	var account model.Account
	if err := s.db.WithContext(ctx).Preload("Roles").First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

type AccountCreateRequest struct {
	Name     string    `json:"name" binding:"required"`
	Email    string    `json:"email" binding:"required,email"`
	Password string    `json:"password" binding:"required"`
	Status   bool      `json:"status"`
	Roles    []RoleRef `json:"roles"`
}

type RoleRef struct {
	ID int64 `json:"id"`
}

func (s *AccountLogic) Create(ctx context.Context, req *AccountCreateRequest) (*model.Account, error) {
	hashed, err := hash.BcryptHashDefault(req.Password)
	if err != nil {
		return nil, err
	}

	account := model.Account{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashed,
		Status:   req.Status,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}

		if len(req.Roles) > 0 {
			roleIDs := make([]int64, len(req.Roles))
			for i, r := range req.Roles {
				roleIDs[i] = r.ID
			}
			var roles []model.Role
			if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
				return err
			}
			return tx.Model(&account).Association("Roles").Replace(roles)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, account.ID)
}

type AccountUpdateRequest struct {
	ID       int64     `json:"id" binding:"required"`
	Name     string    `json:"name" binding:"required"`
	Email    string    `json:"email" binding:"required,email"`
	Password string    `json:"password"`
	Status   bool      `json:"status"`
	Roles    []RoleRef `json:"roles"`
}

func (s *AccountLogic) Update(ctx context.Context, req *AccountUpdateRequest) (*model.Account, error) {
	var account model.Account
	if err := s.db.WithContext(ctx).First(&account, req.ID).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":   req.Name,
		"email":  req.Email,
		"status": req.Status,
	}

	if req.Password != "" {
		hashed, err := hash.BcryptHashDefault(req.Password)
		if err != nil {
			return nil, err
		}
		updates["password"] = hashed
	}

	s.accountCache.Del(ctx, fmt.Sprintf("account:%d", req.ID))
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&account).Updates(updates).Error; err != nil {
			return err
		}

		if req.Roles != nil {
			roleIDs := make([]int64, len(req.Roles))
			for i, r := range req.Roles {
				roleIDs[i] = r.ID
			}
			var roles []model.Role
			if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
				return err
			}
			return tx.Model(&account).Association("Roles").Replace(roles)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, account.ID)
}

func (s *AccountLogic) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("account_id = ?", id).Delete(&model.AccountRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Account{}, id).Error
	})
}
