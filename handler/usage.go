package handler

import (
	"net/http"
	"strconv"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
)

type UsageHandler struct {
	svc *logic.UsageLogic
}

func NewUsageHandler(svc *logic.UsageLogic) *UsageHandler {
	return &UsageHandler{svc: svc}
}

func (h *UsageHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.UsageListRequest
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}

	account := middleware.AccountFromContext(ctx)
	if account != nil && !middleware.IsAdmin(ctx) {
		req.CurrentAccountID = account.ID
	}

	resp, err := h.svc.List(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, resp)
}

func (h *UsageHandler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountIDStr := r.URL.Query().Get("account_id")
	var accountID int64
	if accountIDStr != "" {
		accountID, _ = strconv.ParseInt(accountIDStr, 10, 64)
	}

	account := middleware.AccountFromContext(ctx)
	if account != nil && !middleware.IsAdmin(ctx) {
		accountID = account.ID
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	stats, err := h.svc.GetStats(ctx, accountID, startDate, endDate)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, stats)
}
