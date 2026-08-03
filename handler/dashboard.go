package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
)

type DashboardHandler struct {
	svc *logic.DashboardLogic
}

func NewDashboardHandler(svc *logic.DashboardLogic) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	account := middleware.AccountFromContext(ctx)
	var accountID int64
	if account != nil && !middleware.IsAdmin(ctx) {
		accountID = account.ID
	}

	stats, err := h.svc.GetStats(ctx, accountID)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSONCtx(ctx, w, stats)
}
