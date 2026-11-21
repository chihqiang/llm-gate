package handler

import (
	"github.com/chihqiang/ten-go/core/cast"
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/ten-go/rest"
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
	if err := rest.BindQuery(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	resp, err := h.svc.List(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, resp)
}

func (h *RoleHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roles, err := h.svc.AllList(ctx)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, roles)
}

func (h *RoleHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	role, err := h.svc.GetByID(ctx, id)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, role)
}

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.RoleCreateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	role, err := h.svc.Create(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, role)
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	var req logic.RoleUpdateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	req.ID = id

	role, err := h.svc.Update(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, role)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *RoleHandler) AssociateMenus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	var req logic.RoleMenuRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	if err := h.svc.AssociateMenus(ctx, id, req.MenuIDs); err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, nil)
}
