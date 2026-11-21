package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/ten-go/rest"
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
	if err := rest.BindQuery(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	account := middleware.AccountFromContext(ctx)
	if account != nil && !middleware.IsAdmin(ctx) {
		req.CurrentAccountID = account.ID
	}

	resp, err := h.svc.List(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, resp)
}
