package handler

import (
	"net/http"

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
	ctx := r.Context()
	var req logic.MenuListRequest
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

func (h *MenuHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	menus, err := h.svc.AllList(ctx)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, menus)
}

func (h *MenuHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	menu, err := h.svc.GetByID(ctx, id)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, menu)
}

func (h *MenuHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.MenuCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	menu, err := h.svc.Create(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, menu)
}

func (h *MenuHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.MenuUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	menu, err := h.svc.Update(ctx, id, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, menu)
}

func (h *MenuHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
