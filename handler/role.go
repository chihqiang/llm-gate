package handler

import (
	"net/http"
	"strconv"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/infra-go/httpx"
)

type RoleHandler struct {
	svc *logic.RoleLogic
}

func NewRoleHandler(svc *logic.RoleLogic) *RoleHandler {
	return &RoleHandler{svc: svc}
}

func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	var req logic.RoleListRequest
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

func (h *RoleHandler) AllList(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.AllList()
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, roles)
}

func (h *RoleHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	role, err := h.svc.GetByID(id)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, role)
}

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.RoleCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	role, err := h.svc.Create(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, role)
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.RoleUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	role, err := h.svc.Update(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, role)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *RoleHandler) AssociateMenus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.RoleMenuRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	if err := h.svc.AssociateMenus(id, req.MenuIDs); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, nil)
}
