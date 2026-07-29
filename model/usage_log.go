package model

import "time"

type UsageLog struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	AccountID        int64     `json:"account_id" gorm:"not null;index;comment:账户ID"`
	TokenID          int64     `json:"token_id" gorm:"index;comment:令牌ID"`
	ModelName        string    `json:"model_name" gorm:"size:128;comment:模型名称"`
	ProviderID       int64     `json:"provider_id" gorm:"comment:服务商ID"`
	PromptTokens     int       `json:"prompt_tokens" gorm:"comment:提示令牌数"`
	CompletionTokens int       `json:"completion_tokens" gorm:"comment:补全令牌数"`
	TotalTokens      int       `json:"total_tokens" gorm:"comment:总令牌数"`
	QuotaCost        int64     `json:"quota_cost" gorm:"comment:额度消耗"`
	RequestID        string    `json:"request_id" gorm:"size:64;index;comment:请求ID"`
	CreatedAt        time.Time `json:"created_at" gorm:"comment:创建时间"`
}

func (UsageLog) TableName() string {
	return "llm_usage_logs"
}
