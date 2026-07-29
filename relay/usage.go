package relay

import (
	"errors"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
)

func RecordUsage(db *gorm.DB, accountID, tokenID, providerID int64, modelName string, promptTokens, completionTokens int, quotaCost int64, requestID string) error {
	return db.Create(&model.UsageLog{
		AccountID:        accountID,
		TokenID:          tokenID,
		ModelName:        modelName,
		ProviderID:       providerID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		QuotaCost:        quotaCost,
		RequestID:        requestID,
	}).Error
}

func DeductTokenQuota(db *gorm.DB, tokenID int64, quota int64) error {
	result := db.Model(&model.UserToken{}).Where("id = ? AND quota >= ?", tokenID, quota).
		UpdateColumn("quota", gorm.Expr("quota - ?", quota))
	if result.RowsAffected == 0 {
		return errors.New("insufficient quota")
	}
	return result.Error
}

func RefundTokenQuota(db *gorm.DB, tokenID int64, quota int64) error {
	return db.Model(&model.UserToken{}).Where("id = ?", tokenID).
		UpdateColumn("quota", gorm.Expr("quota + ?", quota)).Error
}
