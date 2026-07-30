package handler

import (
	"net/http"
	"strconv"

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
	var req logic.TokenListRequest
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

func (h *TokenHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	token, err := h.svc.GetByID(id)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, token)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.TokenCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	token, err := h.svc.Create(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, token)
}

func (h *TokenHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.TokenUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	token, err := h.svc.Update(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, token)
}

func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, nil)
}

func (h *TokenHandler) Reveal(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	account := middleware.AccountFromContext(r.Context())
	if account == nil {
		httpx.WriteHTTPError(w, httpx.CodeUnauthorized, "未登录")
		return
	}

	key, err := h.svc.RevealKey(id, account.ID)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, map[string]string{"key": key})
}
