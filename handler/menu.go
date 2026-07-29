package handler

import (
	"net/http"
	"strconv"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/infra-go/httpx"
)

type MenuHandler struct {
	svc *logic.MenuLogic
}

func NewMenuHandler(svc *logic.MenuLogic) *MenuHandler {
	return &MenuHandler{svc: svc}
}

func (h *MenuHandler) List(w http.ResponseWriter, r *http.Request) {
	var req logic.MenuListRequest
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

func (h *MenuHandler) AllList(w http.ResponseWriter, r *http.Request) {
	menus, err := h.svc.AllList()
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, menus)
}

func (h *MenuHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	menu, err := h.svc.GetByID(id)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, menu)
}

func (h *MenuHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.MenuCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	menu, err := h.svc.Create(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, menu)
}

func (h *MenuHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.MenuUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	menu, err := h.svc.Update(id, &req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, menu)
}

func (h *MenuHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
