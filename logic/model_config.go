package logic

import (
	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type ModelLogic struct {
	db *gorm.DB
}

func NewModelLogic(db *gorm.DB) *ModelLogic {
	return &ModelLogic{db: db}
}

type ModelListRequest struct {
	Page       int    `form:"page" binding:"required,min=1"`
	Size       int    `form:"size" binding:"required,min=1,max=1000"`
	Name       string `form:"name"`
	ProviderID int64  `form:"provider_id"`
}

type ModelListResponse struct {
	Data  []model.ModelConfig `json:"data"`
	Total int64               `json:"total"`
}

func (s *ModelLogic) List(req *ModelListRequest) (*ModelListResponse, error) {
	var models []model.ModelConfig
	var total int64

	query := s.db.Model(&model.ModelConfig{})
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.ProviderID > 0 {
		query = query.Where("provider_id = ?", req.ProviderID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	return &ModelListResponse{Data: models, Total: total}, nil
}

func (s *ModelLogic) AllList() ([]model.ModelConfig, error) {
	var models []model.ModelConfig
	err := s.db.Where("status = ?", true).Find(&models).Error
	return models, err
}

func (s *ModelLogic) GetByID(id int64) (*model.ModelConfig, error) {
	var mc model.ModelConfig
	if err := s.db.First(&mc, id).Error; err != nil {
		return nil, err
	}
	return &mc, nil
}

func (s *ModelLogic) GetByName(name string) (*model.ModelConfig, error) {
	var mc model.ModelConfig
	if err := s.db.Where("name = ? AND status = ?", name, true).First(&mc).Error; err != nil {
		return nil, err
	}
	return &mc, nil
}

type ModelCreateRequest struct {
	Name              string  `json:"name" binding:"required"`
	ProviderID        int64   `json:"provider_id" binding:"required"`
	UpstreamModelName string  `json:"upstream_model_name" binding:"required"`
	ModelRatio        float64 `json:"model_ratio"`
	CompletionRatio   float64 `json:"completion_ratio"`
	Status            bool    `json:"status"`
	Remark            string  `json:"remark"`
}

func (s *ModelLogic) Create(req *ModelCreateRequest) (*model.ModelConfig, error) {
	mc := model.ModelConfig{
		Name:              req.Name,
		ProviderID:        req.ProviderID,
		UpstreamModelName: req.UpstreamModelName,
		ModelRatio:        req.ModelRatio,
		CompletionRatio:   req.CompletionRatio,
		Status:            req.Status,
		Remark:            req.Remark,
	}
	if mc.ModelRatio == 0 {
		mc.ModelRatio = 1.0
	}
	if mc.CompletionRatio == 0 {
		mc.CompletionRatio = 1.0
	}
	if err := s.db.Create(&mc).Error; err != nil {
		return nil, err
	}
	return &mc, nil
}

type ModelUpdateRequest struct {
	ID                int64   `json:"id" binding:"required"`
	Name              string  `json:"name" binding:"required"`
	ProviderID        int64   `json:"provider_id" binding:"required"`
	UpstreamModelName string  `json:"upstream_model_name" binding:"required"`
	ModelRatio        float64 `json:"model_ratio"`
	CompletionRatio   float64 `json:"completion_ratio"`
	Status            bool    `json:"status"`
	Remark            string  `json:"remark"`
}

func (s *ModelLogic) Update(req *ModelUpdateRequest) (*model.ModelConfig, error) {
	updates := map[string]interface{}{
		"name":                req.Name,
		"provider_id":         req.ProviderID,
		"upstream_model_name": req.UpstreamModelName,
		"model_ratio":         req.ModelRatio,
		"completion_ratio":    req.CompletionRatio,
		"status":              req.Status,
		"remark":              req.Remark,
	}
	if err := s.db.Model(&model.ModelConfig{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(req.ID)
}

func (s *ModelLogic) Delete(id int64) error {
	return s.db.Delete(&model.ModelConfig{}, id).Error
}
