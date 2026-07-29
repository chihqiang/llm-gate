package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/infra-go/httpx"
)

type DashboardHandler struct {
	svc *logic.DashboardLogic
}

func NewDashboardHandler(svc *logic.DashboardLogic) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats()
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}
	httpx.OkJSON(w, stats)
}
