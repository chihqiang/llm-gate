package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chihqiang/llm-gate/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrBalanceInsufficient = errors.New("insufficient balance")
	ErrOrderStateInvalid   = errors.New("invalid order state")
)

type BillingLogic struct {
	db *gorm.DB
}

func NewBillingLogic(db *gorm.DB) *BillingLogic {
	return &BillingLogic{db: db}
}

// --- 余额操作（供 relay 调用） ---

// DeductBalance 原子预扣账户余额，余额不足返回 ErrBalanceInsufficient。
func (s *BillingLogic) DeductBalance(ctx context.Context, accountID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	res := s.db.WithContext(ctx).Model(&model.Account{}).
		Where("id = ? AND balance_cents >= ?", accountID, cents).
		UpdateColumn("balance_cents", gorm.Expr("balance_cents - ?", cents))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrBalanceInsufficient
	}
	return nil
}

// RefundBalance 退还账户余额（多退少补 / 失败退款）。
func (s *BillingLogic) RefundBalance(ctx context.Context, accountID, cents int64) error {
	if cents <= 0 {
		return nil
	}
	return s.db.WithContext(ctx).Model(&model.Account{}).Where("id = ?", accountID).
		UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", cents)).Error
}

// GetBalance 读取账户余额。
func (s *BillingLogic) GetBalance(ctx context.Context, accountID int64) (int64, error) {
	var account model.Account
	if err := s.db.WithContext(ctx).Select("balance_cents").First(&account, accountID).Error; err != nil {
		return 0, err
	}
	return account.BalanceCents, nil
}

// AppendTransaction 记录余额流水并返回变动后余额。
func (s *BillingLogic) AppendTransaction(tx *gorm.DB, t model.Transaction) error {
	var account model.Account
	if err := tx.Select("balance_cents").First(&account, t.AccountID).Error; err != nil {
		return err
	}
	t.BalanceCents = account.BalanceCents
	return tx.Create(&t).Error
}

// AddSpent 累加 Token 已消费金额。
func AddSpent(ctx context.Context, db *gorm.DB, tokenID, cents int64) error {
	return db.WithContext(ctx).Model(&model.UserToken{}).Where("id = ?", tokenID).
		UpdateColumn("spent_cents", gorm.Expr("spent_cents + ?", cents)).Error
}

// --- 充值订单（管理端） ---

type RechargeOrderListRequest struct {
	Page      int    `form:"page" binding:"required,min=1"`
	Size      int    `form:"size" binding:"required,min=1,max=1000"`
	AccountID int64  `form:"account_id"`
	Status    string `form:"status"`
}

type RechargeOrderListResponse struct {
	Data  []model.RechargeOrder `json:"data"`
	Total int64                 `json:"total"`
}

func (s *BillingLogic) ListOrders(ctx context.Context, req *RechargeOrderListRequest) (*RechargeOrderListResponse, error) {
	var orders []model.RechargeOrder
	var total int64

	query := s.db.WithContext(ctx).Model(&model.RechargeOrder{})
	if req.AccountID > 0 {
		query = query.Where("account_id = ?", req.AccountID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("id DESC").Find(&orders).Error; err != nil {
		return nil, err
	}
	return &RechargeOrderListResponse{Data: orders, Total: total}, nil
}

type RechargeOrderCreateRequest struct {
	AccountID   int64  `json:"account_id" binding:"required"`
	AmountCents int64  `json:"amount_cents" binding:"required,min=1"`
	Remark      string `json:"remark"`
}

func (s *BillingLogic) CreateOrder(ctx context.Context, req *RechargeOrderCreateRequest, operatorID int64) (*model.RechargeOrder, error) {
	var account model.Account
	if err := s.db.WithContext(ctx).Select("id").First(&account, req.AccountID).Error; err != nil {
		return nil, errors.New("account not found")
	}
	order := model.RechargeOrder{
		AccountID:   req.AccountID,
		AmountCents: req.AmountCents,
		Status:      model.RechargeStatusPending,
		Remark:      req.Remark,
		CreatedBy:   operatorID,
	}
	if err := s.db.WithContext(ctx).Create(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// ConfirmOrder 确认充值订单入账：状态必须为 pending，同一事务内更新余额+写流水+更新订单，防止并发重复入账。
func (s *BillingLogic) ConfirmOrder(ctx context.Context, orderID, operatorID int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.RechargeOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			return errors.New("order not found")
		}
		if order.Status != model.RechargeStatusPending {
			return ErrOrderStateInvalid
		}

		if err := tx.Model(&model.Account{}).Where("id = ?", order.AccountID).
			UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", order.AmountCents)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.RechargeOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"status":  model.RechargeStatusPaid,
			"paid_by": operatorID,
			"paid_at": now,
		}).Error; err != nil {
			return err
		}
		return s.AppendTransaction(tx, model.Transaction{
			AccountID:   order.AccountID,
			Type:        model.TransactionRecharge,
			AmountCents: order.AmountCents,
			Remark:      fmt.Sprintf("充值入账，订单 #%d", orderID),
		})
	})
}

func (s *BillingLogic) CancelOrder(ctx context.Context, orderID, operatorID int64) error {
	res := s.db.WithContext(ctx).Model(&model.RechargeOrder{}).
		Where("id = ? AND status = ?", orderID, model.RechargeStatusPending).
		Updates(map[string]interface{}{"status": model.RechargeStatusCancelled})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrderStateInvalid
	}
	return nil
}

// AdjustBalance 人工调整余额（正数加，负数减）。
func (s *BillingLogic) AdjustBalance(ctx context.Context, accountID, cents int64, remark string, operatorID int64) error {
	if cents == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if cents > 0 {
			err = tx.Model(&model.Account{}).Where("id = ?", accountID).
				UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", cents)).Error
		} else {
			res := tx.Model(&model.Account{}).Where("id = ? AND balance_cents >= ?", accountID, -cents).
				UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", cents))
			err = res.Error
			if err == nil && res.RowsAffected == 0 {
				return ErrBalanceInsufficient
			}
		}
		if err != nil {
			return err
		}
		return s.AppendTransaction(tx, model.Transaction{
			AccountID:   accountID,
			Type:        model.TransactionAdjust,
			AmountCents: cents,
			Remark:      remark,
		})
	})
}

// --- 余额流水查询 ---

type TransactionListRequest struct {
	Page      int    `form:"page" binding:"required,min=1"`
	Size      int    `form:"size" binding:"required,min=1,max=1000"`
	AccountID int64  `form:"account_id"`
	Type      string `form:"type"`
}

type TransactionListResponse struct {
	Data  []model.Transaction `json:"data"`
	Total int64               `json:"total"`
}

func (s *BillingLogic) ListTransactions(ctx context.Context, req *TransactionListRequest) (*TransactionListResponse, error) {
	var txns []model.Transaction
	var total int64

	query := s.db.WithContext(ctx).Model(&model.Transaction{})
	if req.AccountID > 0 {
		query = query.Where("account_id = ?", req.AccountID)
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).Order("id DESC").Find(&txns).Error; err != nil {
		return nil, err
	}
	return &TransactionListResponse{Data: txns, Total: total}, nil
}
