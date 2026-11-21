package model

import "time"

type UsageLog struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	AccountID        int64     `json:"account_id" gorm:"not null;index:idx_usage_account_created,priority:1;comment:账户ID"`
	TokenID          int64     `json:"token_id" gorm:"index;comment:令牌ID"`
	ModelName        string    `json:"model_name" gorm:"size:128;index:idx_usage_model_created,priority:1;comment:模型名称"`
	ProviderID       int64     `json:"provider_id" gorm:"comment:服务商ID"`
	PromptTokens     int       `json:"prompt_tokens" gorm:"comment:提示令牌数"`
	CompletionTokens int       `json:"completion_tokens" gorm:"comment:补全令牌数"`
	TotalTokens      int       `json:"total_tokens" gorm:"comment:总令牌数"`
	QuotaCost        int64     `json:"quota_cost" gorm:"comment:额度消耗（兼容旧字段）"`
	CostCents        int64     `json:"cost_cents" gorm:"comment:实际扣费（分）"`
	Estimated        bool      `json:"estimated" gorm:"default:false;comment:是否估算计费（流式未返回usage）"`
	RequestID        string    `json:"request_id" gorm:"size:64;index;comment:请求ID"`
	CreatedAt        time.Time `json:"created_at" gorm:"index:idx_usage_account_created,priority:2;index:idx_usage_model_created,priority:2;comment:创建时间"`
}

func (UsageLog) TableName() string {
	return "llm_usage_logs"
}
