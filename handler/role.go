package handler

import (
	"net/http"

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
	ctx := r.Context()
	var req logic.RoleListRequest
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}

	resp, err := h.svc.List(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, resp)
}

func (h *RoleHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roles, err := h.svc.AllList(ctx)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, roles)
}

func (h *RoleHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	role, err := h.svc.GetByID(ctx, id)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, role)
}

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RoleCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	role, err := h.svc.Create(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, role)
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.RoleUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	role, err := h.svc.Update(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, role)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *RoleHandler) AssociateMenus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.RoleMenuRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	if err := h.svc.AssociateMenus(ctx, id, req.MenuIDs); err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, nil)
}
