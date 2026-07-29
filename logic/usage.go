package logic

import (
	"time"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type UsageLogic struct {
	db *gorm.DB
}

func NewUsageLogic(db *gorm.DB) *UsageLogic {
	return &UsageLogic{db: db}
}

type UsageListRequest struct {
	Page      int    `form:"page" binding:"required,min=1"`
	Size      int    `form:"size" binding:"required,min=1,max=1000"`
	AccountID int64  `form:"account_id"`
	ModelName string `form:"model_name"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type UsageListResponse struct {
	Data  []model.UsageLog `json:"data"`
	Total int64            `json:"total"`
}

func (s *UsageLogic) List(req *UsageListRequest) (*UsageListResponse, error) {
	var logs []model.UsageLog
	var total int64

	query := s.db.Model(&model.UsageLog{})
	if req.AccountID > 0 {
		query = query.Where("account_id = ?", req.AccountID)
	}
	if req.ModelName != "" {
		query = query.Where("model_name = ?", req.ModelName)
	}
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate+" 23:59:59")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	return &UsageListResponse{Data: logs, Total: total}, nil
}

type UsageStat struct {
	ModelName      string `json:"model_name"`
	TotalTokens    int    `json:"total_tokens"`
	TotalQuotaCost int64  `json:"total_quota_cost"`
	RequestCount   int64  `json:"request_count"`
}

func (s *UsageLogic) GetStats(accountID int64, startDate, endDate string) ([]UsageStat, error) {
	query := s.db.Model(&model.UsageLog{}).
		Select("model_name, SUM(total_tokens) as total_tokens, SUM(quota_cost) as total_quota_cost, COUNT(*) as request_count").
		Group("model_name")

	if accountID > 0 {
		query = query.Where("account_id = ?", accountID)
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	stats := make([]UsageStat, 0)
	if err := query.Scan(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (s *UsageLogic) Record(accountID, tokenID, providerID int64, modelName string, promptTokens, completionTokens int, quotaCost int64, requestID string) error {
	log := model.UsageLog{
		AccountID:        accountID,
		TokenID:          tokenID,
		ModelName:        modelName,
		ProviderID:       providerID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		QuotaCost:        quotaCost,
		RequestID:        requestID,
	}
	return s.db.Create(&log).Error
}

func (s *UsageLogic) CleanupOldLogs(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	return s.db.Where("created_at < ?", cutoff).Delete(&model.UsageLog{}).Error
}
