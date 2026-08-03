package handler

import (
	"net/http"
	"strconv"

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
	var req logic.RechargeOrderListRequest
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	resp, err := h.svc.ListOrders(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, resp)
}

func (h *BillingHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req logic.RechargeOrderCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	account := middleware.AccountFromContext(r.Context())
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	order, err := h.svc.CreateOrder(&req, operatorID)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, order)
}

func (h *BillingHandler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}
	account := middleware.AccountFromContext(r.Context())
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	if err := h.svc.ConfirmOrder(id, operatorID); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, nil)
}

func (h *BillingHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}
	account := middleware.AccountFromContext(r.Context())
	var operatorID int64
	if account != nil {
		operatorID = account.ID
	}
	if err := h.svc.CancelOrder(id, operatorID); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, nil)
}

func (h *BillingHandler) AdjustBalance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID int64  `json:"account_id" binding:"required"`
		AmountCents int64 `json:"amount_cents" binding:"required"`
		Remark    string `json:"remark"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.svc.AdjustBalance(req.AccountID, req.AmountCents, req.Remark, 0); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, nil)
}

func (h *BillingHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	var req logic.TransactionListRequest
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	resp, err := h.svc.ListTransactions(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, resp)
}
