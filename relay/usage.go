package relay

import (
	"errors"
	"math"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

var ErrInsufficientQuota = errors.New("insufficient quota")

// DeductBalance 原子预扣账户余额，余额不足返回 ErrInsufficientQuota。
func DeductBalance(db *gorm.DB, accountID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	result := db.Model(&model.Account{}).Where("id = ? AND balance_cents >= ?", accountID, cents).
		UpdateColumn("balance_cents", gorm.Expr("balance_cents - ?", cents))
	if result.Error != nil {
		logger.Error("relay: deduct balance db error",
			logger.Err(result.Error), logger.Int64("account_id", accountID), logger.Int64("cents", cents))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInsufficientQuota
	}
	return nil
}

// RefundBalance 退还账户余额。
func RefundBalance(db *gorm.DB, accountID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	err := db.Model(&model.Account{}).Where("id = ?", accountID).
		UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", cents)).Error
	if err != nil {
		logger.Error("relay: refund balance db error",
			logger.Err(err), logger.Int64("account_id", accountID), logger.Int64("cents", cents))
	}
	return err
}

// GetAccountBalance 读取账户当前余额。
func GetAccountBalance(db *gorm.DB, accountID int64) (int64, error) {
	var account model.Account
	if err := db.Select("balance_cents").First(&account, accountID).Error; err != nil {
		return 0, err
	}
	return account.BalanceCents, nil
}

// AddTokenSpent 累加 Token 累计消费。
func AddTokenSpent(db *gorm.DB, tokenID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	return db.Model(&model.UserToken{}).Where("id = ?", tokenID).
		UpdateColumn("spent_cents", gorm.Expr("spent_cents + ?", cents)).Error
}

// CalculateCostCents 根据模型倍率计算单次请求费用（分）。
// prompt 按 model_ratio 计费，completion 按 model_ratio * completion_ratio 计费，
// 以 billing.base_price_cents_per_1k 为每千 token 基础单价。
func CalculateCostCents(promptTokens, completionTokens int, mc *model.ModelConfig, basePriceCentsPer1K int64) int64 {
	if basePriceCentsPer1K <= 0 {
		basePriceCentsPer1K = 1
	}
	promptCost := float64(promptTokens) / 1000 * mc.ModelRatio
	completionCost := float64(completionTokens) / 1000 * mc.ModelRatio * mc.CompletionRatio
	return int64(math.Round((promptCost + completionCost) * float64(basePriceCentsPer1K)))
}
