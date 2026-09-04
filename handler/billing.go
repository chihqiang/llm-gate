package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
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
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	resp, err := h.svc.ListOrders(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, resp)
}

func (h *BillingHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RechargeOrderCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	account := middleware.AccountFromContext(ctx)
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	order, err := h.svc.CreateOrder(ctx, &req, operatorID)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, order)
}

func (h *BillingHandler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}
	account := middleware.AccountFromContext(ctx)
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	if err := h.svc.ConfirmOrder(ctx, id, operatorID); err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, nil)
}

func (h *BillingHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}
	account := middleware.AccountFromContext(ctx)
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	if err := h.svc.CancelOrder(ctx, id, operatorID); err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, nil)
}

func (h *BillingHandler) AdjustBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		AccountID   int64  `json:"account_id" binding:"required"`
		AmountCents int64  `json:"amount_cents" binding:"required"`
		Remark      string `json:"remark"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.svc.AdjustBalance(ctx, req.AccountID, req.AmountCents, req.Remark, 0); err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, nil)
}

func (h *BillingHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.TransactionListRequest
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	resp, err := h.svc.ListTransactions(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, resp)
}
