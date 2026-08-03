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
	ctx := r.Context()
	var req logic.LoginRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	resp, err := h.svc.Login(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RefreshRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	resp, err := h.svc.Refresh(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	account := middleware.AccountFromContext(ctx)
	if account == nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeUnauthorized, "未登录")
		return
	}

	profile, err := h.svc.GetProfile(ctx, account.ID, middleware.IsAdmin(ctx))
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, profile)
}
