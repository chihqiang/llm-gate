package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
)

type LogHandler struct {
	svc *logic.LogLogic
}

func NewLogHandler(svc *logic.LogLogic) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) List(w http.ResponseWriter, r *http.Request) {
	var req logic.LogListRequest
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}

	account := middleware.AccountFromContext(r.Context())
	if account != nil && !middleware.IsAdmin(r.Context()) {
		req.CurrentAccountID = account.ID
	}

	resp, err := h.svc.List(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, resp)
}
