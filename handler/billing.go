package handler

import (
	"github.com/chihqiang/ten-go/core/cast"
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/ten-go/rest"
)

type BillingHandler struct {
	svc *logic.BillingLogic
}

func NewBillingHandler(svc *logic.BillingLogic) *BillingHandler {
	return &BillingHandler{svc: svc}
}

func (h *BillingHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RechargeOrderListRequest
	if err := rest.BindQuery(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	resp, err := h.svc.ListOrders(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}
	rest.OkJSONCtx(ctx, w, resp)
}

func (h *BillingHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RechargeOrderCreateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	account := middleware.AccountFromContext(ctx)
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	order, err := h.svc.CreateOrder(ctx, &req, operatorID)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}
	rest.OkJSONCtx(ctx, w, order)
}

func (h *BillingHandler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}
	account := middleware.AccountFromContext(ctx)
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	if err := h.svc.ConfirmOrder(ctx, id, operatorID); err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}
	rest.OkJSONCtx(ctx, w, nil)
}

func (h *BillingHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}
	account := middleware.AccountFromContext(ctx)
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	if err := h.svc.CancelOrder(ctx, id, operatorID); err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}
	rest.OkJSONCtx(ctx, w, nil)
}

func (h *BillingHandler) AdjustBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		AccountID   int64  `json:"account_id" binding:"required"`
		AmountCents int64  `json:"amount_cents" binding:"required"`
		Remark      string `json:"remark"`
	}
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	if err := h.svc.AdjustBalance(ctx, req.AccountID, req.AmountCents, req.Remark, 0); err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}
	rest.OkJSONCtx(ctx, w, nil)
}

func (h *BillingHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.TransactionListRequest
	if err := rest.BindQuery(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	resp, err := h.svc.ListTransactions(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}
	rest.OkJSONCtx(ctx, w, resp)
}
