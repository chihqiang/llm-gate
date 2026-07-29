package logic

import (
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/hash"
	"gorm.io/gorm"
)

type AccountLogic struct {
	db *gorm.DB
}

func NewAccountLogic(db *gorm.DB) *AccountLogic {
	return &AccountLogic{db: db}
}

type AccountListRequest struct {
	Page int `form:"page" binding:"required,min=1"`
	Size int `form:"size" binding:"required,min=1,max=100"`
	ID   int `form:"id"`
}

type AccountListResponse struct {
	Data  []model.Account `json:"data"`
	Total int64           `json:"total"`
}

func (s *AccountLogic) List(req *AccountListRequest) (*AccountListResponse, error) {
	var accounts []model.Account
	var total int64

	query := s.db.Model(&model.Account{})
	if req.ID > 0 {
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

func (s *AccountLogic) GetByID(id int64) (*model.Account, error) {
	var account model.Account
	if err := s.db.Preload("Roles").First(&account, id).Error; err != nil {
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

func (s *AccountLogic) Create(req *AccountCreateRequest) (*model.Account, error) {
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

	err = s.db.Transaction(func(tx *gorm.DB) error {
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

	return s.GetByID(account.ID)
}

type AccountUpdateRequest struct {
	ID       int64     `json:"id" binding:"required"`
	Name     string    `json:"name" binding:"required"`
	Email    string    `json:"email" binding:"required,email"`
	Password string    `json:"password"`
	Status   bool      `json:"status"`
	Roles    []RoleRef `json:"roles"`
}

func (s *AccountLogic) Update(req *AccountUpdateRequest) (*model.Account, error) {
	var account model.Account
	if err := s.db.First(&account, req.ID).Error; err != nil {
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

	err := s.db.Transaction(func(tx *gorm.DB) error {
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

	return s.GetByID(account.ID)
}

func (s *AccountLogic) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("account_id = ?", id).Delete(&model.AccountRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Account{}, id).Error
	})
}
