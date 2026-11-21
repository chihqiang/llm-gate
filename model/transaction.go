package model

import "time"

// TransactionType 余额流水类型
const (
	TransactionConsume  = "consume"  // 消费（请求扣款，负数）
	TransactionRefund   = "refund"   // 退款/多退少补（正数）
	TransactionRecharge = "recharge" // 充值入账（正数）
	TransactionAdjust   = "adjust"   // 人工调整（正/负）
)

// Transaction 账户余额流水，记录每一笔余额变动及其变动后余额。
type Transaction struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	AccountID    int64     `json:"account_id" gorm:"not null;index:idx_transaction_account_created,priority:1;comment:账户ID"`
	Type         string    `json:"type" gorm:"size:16;not null;index;comment:流水类型"`
	AmountCents  int64     `json:"amount_cents" gorm:"not null;comment:变动金额（分，可正可负）"`
	BalanceCents int64     `json:"balance_cents" gorm:"not null;comment:变动后余额（分）"`
	TokenID      int64     `json:"token_id" gorm:"index;comment:关联TokenID"`
	RequestID    string    `json:"request_id" gorm:"size:64;index;comment:关联请求ID"`
	Remark       string    `json:"remark" gorm:"size:512;comment:备注"`
	CreatedAt    time.Time `json:"created_at" gorm:"index:idx_transaction_account_created,priority:2;comment:创建时间"`
}

func (Transaction) TableName() string {
	return "llm_transactions"
}
