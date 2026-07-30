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
	Page             int    `form:"page" binding:"required,min=1"`
	Size             int    `form:"size" binding:"required,min=1,max=1000"`
	AccountID        int64  `form:"account_id"`
	ModelName        string `form:"model_name"`
	StartDate        string `form:"start_date"`
	EndDate          string `form:"end_date"`
	CurrentAccountID int64  `form:"-"`
}

type UsageLogVO struct {
	model.UsageLog
	AccountName string `json:"account_name"`
	TokenName   string `json:"token_name"`
}

type UsageListResponse struct {
	Data  []UsageLogVO `json:"data"`
	Total int64        `json:"total"`
}

func (s *UsageLogic) List(req *UsageListRequest) (*UsageListResponse, error) {
	var logs []UsageLogVO
	var total int64

	query := s.db.Model(&model.UsageLog{}).
		Select("llm_usage_logs.*, COALESCE(sa.name, '') as account_name, COALESCE(lt.name, '') as token_name").
		Joins("LEFT JOIN sys_accounts sa ON sa.id = llm_usage_logs.account_id").
		Joins("LEFT JOIN llm_user_tokens lt ON lt.id = llm_usage_logs.token_id")
	if req.CurrentAccountID > 0 {
		query = query.Where("llm_usage_logs.account_id = ?", req.CurrentAccountID)
	} else if req.AccountID > 0 {
		query = query.Where("llm_usage_logs.account_id = ?", req.AccountID)
	}
	if req.ModelName != "" {
		query = query.Where("llm_usage_logs.model_name = ?", req.ModelName)
	}
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			query = query.Where("llm_usage_logs.created_at >= ?", t)
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			query = query.Where("llm_usage_logs.created_at < ?", t.Add(24*time.Hour))
		}
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("llm_usage_logs.id DESC").Find(&logs).Error; err != nil {
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
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("created_at < ?", t.Add(24*time.Hour))
		}
	}

	stats := make([]UsageStat, 0)
	if err := query.Scan(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}
