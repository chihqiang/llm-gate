package logic

import (
	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type RoleLogic struct {
	db *gorm.DB
}

func NewRoleLogic(db *gorm.DB) *RoleLogic {
	return &RoleLogic{db: db}
}

type RoleListRequest struct {
	Page int `form:"page" binding:"required,min=1"`
	Size int `form:"size" binding:"required,min=1,max=100"`
	ID   int `form:"id"`
}

type RoleListResponse struct {
	Data  []model.Role `json:"data"`
	Total int64        `json:"total"`
}

func (s *RoleLogic) List(req *RoleListRequest) (*RoleListResponse, error) {
	var roles []model.Role
	var total int64

	query := s.db.Model(&model.Role{})
	if req.ID > 0 {
		query = query.Where("id = ?", req.ID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("sort ASC").Find(&roles).Error; err != nil {
		return nil, err
	}

	return &RoleListResponse{Data: roles, Total: total}, nil
}

func (s *RoleLogic) AllList() ([]model.Role, error) {
	var roles []model.Role
	err := s.db.Order("sort ASC").Find(&roles).Error
	return roles, err
}

func (s *RoleLogic) GetByID(id int64) (*model.Role, error) {
	var role model.Role
	if err := s.db.Preload("Menus").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

type RoleCreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Sort   int    `json:"sort"`
	Status bool   `json:"status"`
	Remark string `json:"remark"`
}

func (s *RoleLogic) Create(req *RoleCreateRequest) (*model.Role, error) {
	role := model.Role{
		Name:   req.Name,
		Sort:   req.Sort,
		Status: req.Status,
		Remark: req.Remark,
	}
	if err := s.db.Create(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

type RoleUpdateRequest struct {
	ID     int64  `json:"id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Sort   int    `json:"sort"`
	Status bool   `json:"status"`
	Remark string `json:"remark"`
}

func (s *RoleLogic) Update(req *RoleUpdateRequest) (*model.Role, error) {
	role := model.Role{
		ID:     req.ID,
		Name:   req.Name,
		Sort:   req.Sort,
		Status: req.Status,
		Remark: req.Remark,
	}
	if err := s.db.Save(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleLogic) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var role model.Role
		if err := tx.First(&role, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&role).Association("Menus").Clear(); err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
}

type RoleMenuRequest struct {
	MenuIDs []int64 `json:"menu_ids" binding:"required"`
}

func (s *RoleLogic) AssociateMenus(roleID int64, menuIDs []int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var role model.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}

		if len(menuIDs) == 0 {
			return tx.Model(&role).Association("Menus").Clear()
		}

		var menus []model.Menu
		if err := tx.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
			return err
		}
		return tx.Model(&role).Association("Menus").Replace(menus)
	})
}
