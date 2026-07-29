package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"

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

	resp, err := h.svc.List(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, resp)
}
