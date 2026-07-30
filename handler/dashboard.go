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
	account := middleware.AccountFromContext(r.Context())
	var accountID int64
	if account != nil && !middleware.IsAdmin(r.Context()) {
		accountID = account.ID
	}

	stats, err := h.svc.GetStats(accountID)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, stats)
}
