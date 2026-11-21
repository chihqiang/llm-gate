package handler

import (
	"github.com/chihqiang/ten-go/core/cast"
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/ten-go/rest"
)

type ProviderHandler struct {
	svc *logic.ProviderLogic
}

func NewProviderHandler(svc *logic.ProviderLogic) *ProviderHandler {
	return &ProviderHandler{svc: svc}
}

func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.ProviderListRequest
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

func (h *ProviderHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	providers, err := h.svc.AllList(ctx)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, providers)
}

func (h *ProviderHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	provider, err := h.svc.GetByID(ctx, id)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, provider)
}

func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.ProviderCreateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	provider, err := h.svc.Create(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, provider)
}

func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	var req logic.ProviderUpdateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	req.ID = id

	provider, err := h.svc.Update(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, provider)
}

func (h *ProviderHandler) PreviewSyncModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	result, err := h.svc.PreviewModels(ctx, id)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, result)
}

func (h *ProviderHandler) SyncModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	var req logic.SyncModelsRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	result, err := h.svc.SyncModels(ctx, id, req.Models)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, result)
}

func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
