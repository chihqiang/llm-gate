package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
)

type AuthHandler struct {
	svc *logic.AuthLogic
}

func NewAuthHandler(svc *logic.AuthLogic) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req logic.LoginRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	resp, err := h.svc.Login(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	account := middleware.AccountFromContext(r.Context())
	if account == nil {
		httpx.WriteHTTPError(w, httpx.CodeUnauthorized, "未登录")
		return
	}

	profile, err := h.svc.GetProfile(account.ID)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, profile)
}
