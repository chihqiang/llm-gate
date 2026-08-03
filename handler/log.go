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
	ctx := r.Context()
	var req logic.LogListRequest
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
