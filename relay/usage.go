package relay

import (
	"errors"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

var ErrInsufficientQuota = errors.New("insufficient quota")

func DeductTokenQuota(db *gorm.DB, tokenID int64, quota int64) error {
	result := db.Model(&model.UserToken{}).Where("id = ? AND quota >= ?", tokenID, quota).
		UpdateColumn("quota", gorm.Expr("quota - ?", quota))
	if result.Error != nil {
		logger.Error("deduct token quota db error",
			logger.Err(result.Error), logger.Int64("token_id", tokenID), logger.Int64("quota", quota))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInsufficientQuota
	}
	return nil
}

func RefundTokenQuota(db *gorm.DB, tokenID int64, quota int64) error {
	err := db.Model(&model.UserToken{}).Where("id = ?", tokenID).
		UpdateColumn("quota", gorm.Expr("quota + ?", quota)).Error
	if err != nil {
		logger.Error("refund token quota db error",
			logger.Err(err), logger.Int64("token_id", tokenID), logger.Int64("quota", quota))
	}
	return err
}
