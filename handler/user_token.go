package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
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

func (h *TokenHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	token, err := h.svc.GetByID(ctx, id)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, token)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.TokenCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	token, err := h.svc.Create(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, token)
}

func (h *TokenHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.TokenUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	token, err := h.svc.Update(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, token)
}

func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	if err := h.svc.Delete(ctx, id); err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, nil)
}

func (h *TokenHandler) Reveal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	account := middleware.AccountFromContext(ctx)
	if account == nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeUnauthorized, "未登录")
		return
	}

	key, err := h.svc.RevealKey(ctx, id, account.ID)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, map[string]string{"key": key})
}
