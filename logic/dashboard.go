package logic

import (
	"time"

	"gorm.io/gorm"
)

type DashboardStats struct {
	TotalRequests  int64 `json:"total_requests"`
	TodayRequests  int64 `json:"today_requests"`
	TotalTokens    int64 `json:"total_tokens"`
	TodayTokens    int64 `json:"today_tokens"`
	TotalQuota     int64 `json:"total_quota"`
	TotalCostCents int64 `json:"total_cost_cents"`
	ActiveTokens   int64 `json:"active_tokens"`
	TotalProviders int64 `json:"total_providers"`
	TotalModels    int64 `json:"total_models"`
}

type DashboardLogic struct {
	db *gorm.DB
}

func NewDashboardLogic(db *gorm.DB) *DashboardLogic {
	return &DashboardLogic{db: db}
}

func (s *DashboardLogic) GetStats(accountID int64) (*DashboardStats, error) {
	stats := &DashboardStats{}
	todayStart := time.Now().Truncate(24 * time.Hour)
	now := time.Now()

	var args []interface{}
	usageAnd := ""
	tokenAnd := ""

	if accountID > 0 {
		usageAnd = " AND account_id = ?"
		tokenAnd = " AND account_id = ?"
		args = []interface{}{
			accountID,             // total_requests
			todayStart, accountID, // today_requests
			accountID,             // total_tokens
			todayStart, accountID, // today_tokens
			accountID,      // total_quota
			now, accountID, // active_tokens
		}
	} else {
		args = []interface{}{
			todayStart, // today_requests
			todayStart, // today_tokens
			now,        // active_tokens
		}
	}

	sql := `SELECT
		(SELECT COUNT(*) FROM llm_usage_logs WHERE 1=1` + usageAnd + `) AS total_requests,
		(SELECT COUNT(*) FROM llm_usage_logs WHERE created_at >= ?` + usageAnd + `) AS today_requests,
		(SELECT COALESCE(SUM(total_tokens), 0) FROM llm_usage_logs WHERE 1=1` + usageAnd + `) AS total_tokens,
		(SELECT COALESCE(SUM(total_tokens), 0) FROM llm_usage_logs WHERE created_at >= ?` + usageAnd + `) AS today_tokens,
		(SELECT COALESCE(SUM(quota_cost), 0) FROM llm_usage_logs WHERE 1=1` + usageAnd + `) AS total_quota,
		(SELECT COALESCE(SUM(cost_cents), 0) FROM llm_usage_logs WHERE 1=1` + usageAnd + `) AS total_cost_cents,
		(SELECT COUNT(*) FROM llm_user_tokens WHERE status = 1 AND (expired_at IS NULL OR expired_at > ?)` + tokenAnd + `) AS active_tokens,
		(SELECT COUNT(*) FROM llm_providers) AS total_providers,
		(SELECT COUNT(*) FROM llm_models) AS total_models`

	if err := s.db.Raw(sql, args...).Scan(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}
