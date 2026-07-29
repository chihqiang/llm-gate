package logic

import (
	"time"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

type DashboardStats struct {
	TotalAccounts int64  `json:"total_accounts"`
	TodayVisits   int64  `json:"today_visits"`
	ActiveAccount int64  `json:"active_accounts"`
	SystemStatus  string `json:"system_status"`
}

type DashboardLogic struct {
	db *gorm.DB
}

func NewDashboardLogic(db *gorm.DB) *DashboardLogic {
	return &DashboardLogic{db: db}
}

func (s *DashboardLogic) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{SystemStatus: "正常"}

	var totalAccounts int64
	if err := s.db.Model(&model.Account{}).Count(&totalAccounts).Error; err != nil {
		return nil, err
	}
	stats.TotalAccounts = totalAccounts

	todayStart := time.Now().Truncate(24 * time.Hour)
	var todayVisits int64
	if err := s.db.Model(&model.Log{}).Where("created_at >= ?", todayStart).Count(&todayVisits).Error; err != nil {
		return nil, err
	}
	stats.TodayVisits = todayVisits

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var activeAccount int64
	if err := s.db.Model(&model.Log{}).
		Where("created_at >= ?", sevenDaysAgo).
		Distinct("account_id").
		Count(&activeAccount).Error; err != nil {
		return nil, err
	}
	stats.ActiveAccount = activeAccount

	return stats, nil
}
