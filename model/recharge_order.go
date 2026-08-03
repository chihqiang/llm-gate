package model

import "time"

// RechargeOrderStatus 充值订单状态
const (
	RechargeStatusPending   = "pending"
	RechargeStatusPaid      = "paid"
	RechargeStatusCancelled = "cancelled"
)

// RechargeOrder 充值订单。真实货币计费：线下转账后由管理员确认入账。
type RechargeOrder struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	AccountID   int64     `json:"account_id" gorm:"not null;index;comment:账户ID"`
	AmountCents int64     `json:"amount_cents" gorm:"not null;comment:充值金额（分）"`
	Status      string    `json:"status" gorm:"size:16;not null;default:pending;index;comment:状态"`
	Remark      string    `json:"remark" gorm:"size:512;comment:备注"`
	CreatedBy   int64     `json:"created_by" gorm:"comment:创建人账户ID"`
	PaidBy      int64     `json:"paid_by" gorm:"comment:确认人账户ID"`
	PaidAt      *time.Time `json:"paid_at" gorm:"comment:确认时间"`
	CreatedAt   time.Time `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"comment:更新时间"`
}

func (RechargeOrder) TableName() string {
	return "llm_recharge_orders"
}
