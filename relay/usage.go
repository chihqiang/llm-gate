package relay

import (
	"errors"
	"math"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

var ErrInsufficientQuota = errors.New("insufficient quota")
var ErrQuotaExhausted = errors.New("key quota exhausted")

// DeductBalance 原子预扣账户余额，余额不足返回 ErrInsufficientQuota。
func DeductBalance(db *gorm.DB, accountID, cents int64) error {
	return DeductBalanceTx(db, accountID, cents)
}

func DeductBalanceTx(tx *gorm.DB, accountID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	result := tx.Model(&model.Account{}).Where("id = ? AND balance_cents >= ?", accountID, cents).
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
	return RefundBalanceTx(db, accountID, cents)
}

func RefundBalanceTx(tx *gorm.DB, accountID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	err := tx.Model(&model.Account{}).Where("id = ?", accountID).
		UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", cents)).Error
	if err != nil {
		logger.Error("relay: refund balance db error",
			logger.Err(err), logger.Int64("account_id", accountID), logger.Int64("cents", cents))
	}
	return err
}

// GetAccountBalance 读取账户当前余额。
func GetAccountBalance(db *gorm.DB, accountID int64) (int64, error) {
	return GetAccountBalanceTx(db, accountID)
}

func GetAccountBalanceTx(tx *gorm.DB, accountID int64) (int64, error) {
	var account model.Account
	if err := tx.Select("balance_cents").First(&account, accountID).Error; err != nil {
		return 0, err
	}
	return account.BalanceCents, nil
}

// AddTokenSpent 累加 Token 累计消费。
func AddTokenSpent(db *gorm.DB, tokenID, cents int64) error {
	return AddTokenSpentTx(db, tokenID, cents)
}

func AddTokenSpentTx(tx *gorm.DB, tokenID, cents int64) error {
	return AdjustTokenSpentTx(tx, tokenID, cents)
}

// AdjustTokenSpentTx 调整 Token 累计消费，delta 可为负（用于多退少补）。
func AdjustTokenSpentTx(tx *gorm.DB, tokenID, delta int64) error {
	if delta == 0 {
		return nil
	}
	return tx.Model(&model.UserToken{}).Where("id = ?", tokenID).
		UpdateColumn("spent_cents", gorm.Expr("spent_cents + ?", delta)).Error
}

// ReserveTokenQuota 请求开始时原子预占 Token 预算额度。
// quota<=0 表示不限额；限额时保证 spent_cents + cents <= quota，超额返回 ErrQuotaExhausted。
// 预占后 spent_cents 为"已占+已用"，结算时按 delta = actual - preConsume 调整。
func ReserveTokenQuota(tx *gorm.DB, tokenID, cents, quota int64) error {
	if cents <= 0 {
		return nil
	}
	if quota > 0 {
		res := tx.Model(&model.UserToken{}).
			Where("id = ? AND spent_cents + ? <= quota", tokenID, cents).
			UpdateColumn("spent_cents", gorm.Expr("spent_cents + ?", cents))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrQuotaExhausted
		}
		return nil
	}
	return tx.Model(&model.UserToken{}).Where("id = ?", tokenID).
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
