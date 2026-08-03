package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/model"
	"chihqiang/llm-gate/security"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/trace"
	"gorm.io/gorm"
)

// upstreamClient 用于请求上游服务商 API，设置超时防止挂起
var upstreamClient = &http.Client{
	Timeout: 30 * time.Second,
}

type ProviderLogic struct {
	db            *gorm.DB
	providerCache cache.Cache
	cipher        *security.Cipher
}

func NewProviderLogic(db *gorm.DB, providerCache cache.Cache, cipher *security.Cipher) *ProviderLogic {
	return &ProviderLogic{db: db, providerCache: providerCache, cipher: cipher}
}

// decryptKey 解密服务商 API 密钥，解密失败返回错误。
func (s *ProviderLogic) decryptKey(ctx context.Context, encrypted string) (string, error) {
	key, err := s.cipher.Decrypt(encrypted)
	if err != nil {
		logger.ErrorCtx(ctx, "provider: decrypt api key failed", logger.Err(err))
		return "", err
	}
	return key, nil
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

func (s *ProviderLogic) List(ctx context.Context, req *ProviderListRequest) (*ProviderListResponse, error) {
	var providers []model.Provider
	var total int64

	query := s.db.WithContext(ctx).Model(&model.Provider{})
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

func (s *ProviderLogic) AllList(ctx context.Context) ([]model.Provider, error) {
	var providers []model.Provider
	err := s.db.WithContext(ctx).Where("status = ?", true).Order("priority ASC, id ASC").Find(&providers).Error
	return providers, err
}

func (s *ProviderLogic) GetByID(ctx context.Context, id int64) (*model.Provider, error) {
	var provider model.Provider
	if err := s.db.WithContext(ctx).First(&provider, id).Error; err != nil {
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

func (s *ProviderLogic) Create(ctx context.Context, req *ProviderCreateRequest) (*model.Provider, error) {
	encrypted, err := s.cipher.Encrypt(req.APIKey)
	if err != nil {
		return nil, err
	}
	provider := model.Provider{
		Name:     req.Name,
		BaseURL:  req.BaseURL,
		APIKey:   encrypted,
		Status:   req.Status,
		Priority: req.Priority,
		Weight:   req.Weight,
		Remark:   req.Remark,
	}
	if err := s.db.WithContext(ctx).Create(&provider).Error; err != nil {
		return nil, err
	}
	s.invalidateRoutingCaches(ctx)
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

func (s *ProviderLogic) Update(ctx context.Context, req *ProviderUpdateRequest) (*model.Provider, error) {
	updates := map[string]interface{}{
		"name":     req.Name,
		"base_url": req.BaseURL,
		"status":   req.Status,
		"priority": req.Priority,
		"weight":   req.Weight,
		"remark":   req.Remark,
	}
	if req.APIKey != "" {
		encrypted, err := s.cipher.Encrypt(req.APIKey)
		if err != nil {
			return nil, err
		}
		updates["api_key"] = encrypted
	}

	if err := s.db.WithContext(ctx).Model(&model.Provider{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	s.providerCache.Del(ctx, fmt.Sprintf("provider:%d", req.ID))
	s.invalidateRoutingCaches(ctx)
	return s.GetByID(ctx, req.ID)
}

func (s *ProviderLogic) Delete(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&model.Provider{}, id).Error; err != nil {
		return err
	}
	s.providerCache.Del(ctx, fmt.Sprintf("provider:%d", id))
	s.invalidateRoutingCaches(ctx)
	return nil
}

// invalidateRoutingCaches 服务商变更影响模型路由，失效所有路由相关缓存。
func (s *ProviderLogic) invalidateRoutingCaches(ctx context.Context) {
	s.providerCache.FlushByPrefix(ctx, "provider:")
	s.providerCache.FlushByPrefix(ctx, "model_list:")
	s.providerCache.FlushByPrefix(ctx, "neg:")
	s.providerCache.FlushByPrefix(ctx, "models:")
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

func (s *ProviderLogic) fetchUpstreamModels(ctx context.Context, providerID int64) ([]upstreamModelDTO, error) {
	provider, err := s.GetByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	key, err := s.decryptKey(ctx, provider.APIKey)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", strings.TrimRight(provider.BaseURL, "/")+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	ctx, span := trace.StartSpan(ctx, "sync models",
		trace.WithAttributes(
			trace.AttrInt64("provider_id", providerID),
			trace.AttrString("provider", provider.Name),
			trace.AttrString("upstream_url", provider.BaseURL),
		),
	)
	defer span.End()
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	// 透传 W3C 链路上下文，便于上游接收端关联
	trace.InjectHeader(ctx, req.Header)

	resp, err := upstreamClient.Do(req)
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

// getExistingModelMap 批量查询已存在的模型，返回 upstream_model_name -> bool 的映射
func (s *ProviderLogic) getExistingModelMap(ctx context.Context, providerID int64, upstreamNames []string) (map[string]bool, error) {
	if len(upstreamNames) == 0 {
		return make(map[string]bool), nil
	}

	var existing []model.ModelConfig
	if err := s.db.WithContext(ctx).Where("provider_id = ? AND upstream_model_name IN ?", providerID, upstreamNames).
		Select("upstream_model_name").Find(&existing).Error; err != nil {
		return nil, err
	}

	existingMap := make(map[string]bool, len(existing))
	for _, m := range existing {
		existingMap[m.UpstreamModelName] = true
	}
	return existingMap, nil
}

func (s *ProviderLogic) PreviewModels(ctx context.Context, providerID int64) (*SyncModelsPreview, error) {
	data, err := s.fetchUpstreamModels(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// 批量查询已存在的模型，避免 N+1
	upstreamNames := make([]string, len(data))
	for i, m := range data {
		upstreamNames[i] = m.ID
	}
	existingMap, err := s.getExistingModelMap(ctx, providerID, upstreamNames)
	if err != nil {
		return nil, err
	}

	result := &SyncModelsPreview{Models: make([]UpstreamModel, 0, len(data))}
	for _, m := range data {
		result.Models = append(result.Models, UpstreamModel{
			ID:     m.ID,
			Exists: existingMap[m.ID],
		})
		result.Total++
	}

	return result, nil
}

type SyncModelsRequest struct {
	Models []string `json:"models"`
}

func (s *ProviderLogic) SyncModels(ctx context.Context, providerID int64, models []string) (*SyncModelsResult, error) {
	provider, err := s.GetByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	data, err := s.fetchUpstreamModels(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// 批量查询已存在的模型，避免 N+1
	upstreamNames := make([]string, len(data))
	for i, m := range data {
		upstreamNames[i] = m.ID
	}
	existingMap, err := s.getExistingModelMap(ctx, providerID, upstreamNames)
	if err != nil {
		return nil, err
	}

	logger.InfoCtx(ctx, "provider sync start", logger.Int64("provider_id", providerID), logger.Int("candidates", len(data)))
	selected := make(map[string]bool, len(models))
	for _, m := range models {
		selected[m] = true
	}

	result := &SyncModelsResult{Models: make([]string, 0, len(data))}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, m := range data {
			if !selected[m.ID] {
				continue
			}
			result.Total++

			if existingMap[m.ID] {
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
			if err := tx.Create(&mc).Error; err != nil {
				return fmt.Errorf("create model %s: %w", m.ID, err)
			}
			result.Created++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
