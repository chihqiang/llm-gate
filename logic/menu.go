package logic

import (
	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type MenuLogic struct {
	db *gorm.DB
}

func NewMenuLogic(db *gorm.DB) *MenuLogic {
	return &MenuLogic{db: db}
}

type MenuListRequest struct {
	Page int `form:"page" binding:"required,min=1"`
	Size int `form:"size" binding:"required,min=1,max=100"`
	ID   int `form:"id"`
}

type MenuListResponse struct {
	Data  []model.Menu `json:"data"`
	Total int64        `json:"total"`
}

func (s *MenuLogic) List(req *MenuListRequest) (*MenuListResponse, error) {
	var menus []model.Menu
	var total int64

	query := s.db.Model(&model.Menu{})
	if req.ID > 0 {
		query = query.Where("id = ?", req.ID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("sort ASC").Find(&menus).Error; err != nil {
		return nil, err
	}

	return &MenuListResponse{Data: menus, Total: total}, nil
}

func (s *MenuLogic) AllList() ([]model.Menu, error) {
	var menus []model.Menu
	err := s.db.Order("sort ASC").Find(&menus).Error
	return menus, err
}

func (s *MenuLogic) GetByID(id int64) (*model.Menu, error) {
	var menu model.Menu
	if err := s.db.First(&menu, id).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

type MenuCreateRequest struct {
	PID       int64  `json:"pid"`
	MenuType  int    `json:"menu_type" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	APIURL    string `json:"api_url"`
	APIMethod string `json:"api_method"`
	Visible   bool   `json:"visible"`
	Status    bool   `json:"status"`
	Remark    string `json:"remark"`
}

type MenuUpdateRequest struct {
	PID       int64  `json:"pid"`
	MenuType  int    `json:"menu_type" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	APIURL    string `json:"api_url"`
	APIMethod string `json:"api_method"`
	Visible   bool   `json:"visible"`
	Status    bool   `json:"status"`
	Remark    string `json:"remark"`
}

func (s *MenuLogic) Create(req *MenuCreateRequest) (*model.Menu, error) {
	menu := model.Menu{
		PID:       req.PID,
		MenuType:  req.MenuType,
		Name:      req.Name,
		Path:      req.Path,
		Component: req.Component,
		Icon:      req.Icon,
		Sort:      req.Sort,
		APIURL:    req.APIURL,
		APIMethod: req.APIMethod,
		Visible:   req.Visible,
		Status:    req.Status,
		Remark:    req.Remark,
	}
	if err := s.db.Create(&menu).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (s *MenuLogic) Update(id int64, req *MenuUpdateRequest) (*model.Menu, error) {
	var menu model.Menu
	if err := s.db.First(&menu, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&menu).Updates(map[string]interface{}{
		"pid":        req.PID,
		"menu_type":  req.MenuType,
		"name":       req.Name,
		"path":       req.Path,
		"component":  req.Component,
		"icon":       req.Icon,
		"sort":       req.Sort,
		"api_url":    req.APIURL,
		"api_method": req.APIMethod,
		"visible":    req.Visible,
		"status":     req.Status,
		"remark":     req.Remark,
	}).Error; err != nil {
		return nil, err
	}
	return &menu, nil
}

func (s *MenuLogic) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", id).Delete(&model.RoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Menu{}, id).Error
	})
}
