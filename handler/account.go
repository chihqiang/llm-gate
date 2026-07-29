package handler

import (
	"net/http"
	"strconv"

	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
)

type AccountHandler struct {
	svc *logic.AccountLogic
}

func NewAccountHandler(svc *logic.AccountLogic) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	var req logic.AccountListRequest
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

func (h *AccountHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	account, err := h.svc.GetByID(id)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, account)
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.AccountCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	account, err := h.svc.Create(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, account)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.AccountUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	account, err := h.svc.Update(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, account)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	account := middleware.AccountFromContext(r.Context())
	if account != nil && account.ID == id {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "不能删除自己的账号")
		return
	}

	if err := h.svc.Delete(id); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, nil)
}
