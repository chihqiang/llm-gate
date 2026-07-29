package logic

import (
	"time"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type DashboardStats struct {
	TotalRequests int64 `json:"total_requests"`
	TodayRequests int64 `json:"today_requests"`
	TotalTokens   int64 `json:"total_tokens"`
	TodayTokens   int64 `json:"today_tokens"`
	TotalQuota    int64 `json:"total_quota"`
	ActiveTokens  int64 `json:"active_tokens"`
	TotalProviders int64 `json:"total_providers"`
	TotalModels   int64 `json:"total_models"`
}

type DashboardLogic struct {
	db *gorm.DB
}

func NewDashboardLogic(db *gorm.DB) *DashboardLogic {
	return &DashboardLogic{db: db}
}

func (s *DashboardLogic) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	todayStart := time.Now().Truncate(24 * time.Hour)

	var totalRequests int64
	s.db.Model(&model.UsageLog{}).Count(&totalRequests)
	stats.TotalRequests = totalRequests

	var todayRequests int64
	s.db.Model(&model.UsageLog{}).Where("created_at >= ?", todayStart).Count(&todayRequests)
	stats.TodayRequests = todayRequests

	var totalTokens int64
	s.db.Model(&model.UsageLog{}).Select("COALESCE(SUM(total_tokens), 0)").Scan(&totalTokens)
	stats.TotalTokens = totalTokens

	var todayTokens int64
	s.db.Model(&model.UsageLog{}).Select("COALESCE(SUM(total_tokens), 0)").Where("created_at >= ?", todayStart).Scan(&todayTokens)
	stats.TodayTokens = todayTokens

	var totalQuota int64
	s.db.Model(&model.UsageLog{}).Select("COALESCE(SUM(quota_cost), 0)").Scan(&totalQuota)
	stats.TotalQuota = totalQuota

	var activeTokens int64
	s.db.Model(&model.UserToken{}).Where("status = ? AND (expired_at IS NULL OR expired_at > ?)", true, time.Now()).Count(&activeTokens)
	stats.ActiveTokens = activeTokens

	var totalProviders int64
	s.db.Model(&model.Provider{}).Count(&totalProviders)
	stats.TotalProviders = totalProviders

	var totalModels int64
	s.db.Model(&model.ModelConfig{}).Count(&totalModels)
	stats.TotalModels = totalModels

	return stats, nil
}
