package logic

import (
	"errors"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

type RoleLogic struct {
	db          *gorm.DB
	adminRoleID int64
}

func NewRoleLogic(db *gorm.DB, adminRoleID int64) *RoleLogic {
	return &RoleLogic{db: db, adminRoleID: adminRoleID}
}

type RoleListRequest struct {
	Page int `form:"page" binding:"required,min=1"`
	Size int `form:"size" binding:"required,min=1,max=1000"`
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
	// 使用 map + Updates 而非 Save，避免零值字段（如 CreatedAt）被覆盖
	if err := s.db.Model(&model.Role{}).Where("id = ?", req.ID).Updates(map[string]interface{}{
		"name":   req.Name,
		"sort":   req.Sort,
		"status": req.Status,
		"remark": req.Remark,
	}).Error; err != nil {
		return nil, err
	}
	return &model.Role{
		ID:     req.ID,
		Name:   req.Name,
		Sort:   req.Sort,
		Status: req.Status,
		Remark: req.Remark,
	}, nil
}

func (s *RoleLogic) Delete(id int64) error {
	if s.adminRoleID > 0 && id == s.adminRoleID {
		logger.Warn("role delete forbidden: admin role", logger.Int64("admin_role_id", s.adminRoleID))
		return errors.New("不能删除超级管理员角色")
	}

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
	if s.adminRoleID > 0 && roleID == s.adminRoleID {
		logger.Warn("role associate menus forbidden: admin role", logger.Int64("admin_role_id", s.adminRoleID))
		return errors.New("不能修改超级管理员角色的菜单权限")
	}

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
