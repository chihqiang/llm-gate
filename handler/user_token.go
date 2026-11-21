package handler

import (
	"github.com/chihqiang/ten-go/core/cast"
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/ten-go/rest"
)

type TokenHandler struct {
	svc *logic.TokenLogic
}

func NewTokenHandler(svc *logic.TokenLogic) *TokenHandler {
	return &TokenHandler{svc: svc}
}

func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.TokenListRequest
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

func (h *TokenHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	token, err := h.svc.GetByID(ctx, id)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, token)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.TokenCreateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	token, err := h.svc.Create(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, token)
}

func (h *TokenHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	var req logic.TokenUpdateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	req.ID = id

	token, err := h.svc.Update(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, token)
}

func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	if err := h.svc.Delete(ctx, id); err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, nil)
}

func (h *TokenHandler) Reveal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	account := middleware.AccountFromContext(ctx)
	if account == nil {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusUnauthorized, "未登录"))
		return
	}

	key, err := h.svc.RevealKey(ctx, id, account.ID)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, map[string]string{"key": key})
}
