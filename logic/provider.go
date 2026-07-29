package logic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type ProviderLogic struct {
	db *gorm.DB
}

func NewProviderLogic(db *gorm.DB) *ProviderLogic {
	return &ProviderLogic{db: db}
}

type ProviderListRequest struct {
	Page int    `form:"page" binding:"required,min=1"`
	Size int    `form:"size" binding:"required,min=1,max=1000"`
	Name string `form:"name"`
}

type ProviderListResponse struct {
	Data  []model.Provider `json:"data"`
	Total int64            `json:"total"`
}

func (s *ProviderLogic) List(req *ProviderListRequest) (*ProviderListResponse, error) {
	var providers []model.Provider
	var total int64

	query := s.db.Model(&model.Provider{})
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("priority ASC, id ASC").Find(&providers).Error; err != nil {
		return nil, err
	}

	return &ProviderListResponse{Data: providers, Total: total}, nil
}

func (s *ProviderLogic) AllList() ([]model.Provider, error) {
	var providers []model.Provider
	err := s.db.Where("status = ?", true).Order("priority ASC, id ASC").Find(&providers).Error
	return providers, err
}

func (s *ProviderLogic) GetByID(id int64) (*model.Provider, error) {
	var provider model.Provider
	if err := s.db.First(&provider, id).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

type ProviderCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key" binding:"required"`
	Status   bool   `json:"status"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Remark   string `json:"remark"`
}

func (s *ProviderLogic) Create(req *ProviderCreateRequest) (*model.Provider, error) {
	provider := model.Provider{
		Name:     req.Name,
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Status:   req.Status,
		Priority: req.Priority,
		Weight:   req.Weight,
		Remark:   req.Remark,
	}
	if err := s.db.Create(&provider).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

type ProviderUpdateRequest struct {
	ID       int64  `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key"`
	Status   bool   `json:"status"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Remark   string `json:"remark"`
}

func (s *ProviderLogic) Update(req *ProviderUpdateRequest) (*model.Provider, error) {
	updates := map[string]interface{}{
		"name":     req.Name,
		"base_url": req.BaseURL,
		"status":   req.Status,
		"priority": req.Priority,
		"weight":   req.Weight,
		"remark":   req.Remark,
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}

	if err := s.db.Model(&model.Provider{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(req.ID)
}

func (s *ProviderLogic) Delete(id int64) error {
	return s.db.Delete(&model.Provider{}, id).Error
}

type UpstreamModel struct {
	ID     string `json:"id"`
	Exists bool   `json:"exists"`
}

type SyncModelsPreview struct {
	Total  int             `json:"total"`
	Models []UpstreamModel `json:"models"`
}

type SyncModelsResult struct {
	Total   int      `json:"total"`
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Models  []string `json:"models"`
}

type upstreamModelDTO struct {
	ID string `json:"id"`
}

func (s *ProviderLogic) fetchUpstreamModels(providerID int64) ([]upstreamModelDTO, error) {
	provider, err := s.GetByID(providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	req, err := http.NewRequest("GET", provider.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request upstream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body))
	}

	var upstream struct {
		Data []upstreamModelDTO `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&upstream); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return upstream.Data, nil
}

func (s *ProviderLogic) PreviewModels(providerID int64) (*SyncModelsPreview, error) {
	data, err := s.fetchUpstreamModels(providerID)
	if err != nil {
		return nil, err
	}

	result := &SyncModelsPreview{Models: make([]UpstreamModel, 0, len(data))}
	for _, m := range data {
		var count int64
		s.db.Model(&model.ModelConfig{}).
			Where("provider_id = ? AND upstream_model_name = ?", providerID, m.ID).
			Count(&count)

		result.Models = append(result.Models, UpstreamModel{
			ID:     m.ID,
			Exists: count > 0,
		})
		result.Total++
	}

	return result, nil
}

type SyncModelsRequest struct {
	Models []string `json:"models"`
}

func (s *ProviderLogic) SyncModels(providerID int64, models []string) (*SyncModelsResult, error) {
	provider, err := s.GetByID(providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	data, err := s.fetchUpstreamModels(providerID)
	if err != nil {
		return nil, err
	}

	selected := make(map[string]bool, len(models))
	for _, m := range models {
		selected[m] = true
	}

	result := &SyncModelsResult{Models: make([]string, 0, len(data))}
	for _, m := range data {
		if !selected[m.ID] {
			continue
		}
		result.Total++

		var count int64
		s.db.Model(&model.ModelConfig{}).
			Where("provider_id = ? AND upstream_model_name = ?", providerID, m.ID).
			Count(&count)
		if count > 0 {
			result.Skipped++
			continue
		}

		remark := fmt.Sprintf("自动同步自 %s", provider.Name)
		mc := model.ModelConfig{
			Name:              m.ID,
			ProviderID:        providerID,
			UpstreamModelName: m.ID,
			ModelRatio:        1.0,
			CompletionRatio:   1.0,
			Status:            true,
			Remark:            remark,
		}
		if err := s.db.Create(&mc).Error; err != nil {
			return nil, fmt.Errorf("create model %s: %w", m.ID, err)
		}
		result.Created++
	}

	return result, nil
}
