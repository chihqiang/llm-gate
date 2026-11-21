package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/ten-go/rest"
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
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	resp, err := h.svc.Login(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RefreshRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	resp, err := h.svc.Refresh(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	account := middleware.AccountFromContext(ctx)
	if account == nil {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusUnauthorized, "未登录"))
		return
	}

	profile, err := h.svc.GetProfile(ctx, account.ID, middleware.IsAdmin(ctx))
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, profile)
}
