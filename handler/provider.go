package handler

import (
	"net/http"
	"strconv"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/infra-go/httpx"
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

func (h *ProviderHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	providers, err := h.svc.AllList(ctx)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, providers)
}

func (h *ProviderHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	provider, err := h.svc.GetByID(ctx, id)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, provider)
}

func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.ProviderCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	provider, err := h.svc.Create(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, provider)
}

func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.ProviderUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	provider, err := h.svc.Update(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, provider)
}

func (h *ProviderHandler) PreviewSyncModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	result, err := h.svc.PreviewModels(ctx, id)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, result)
}

func (h *ProviderHandler) SyncModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.SyncModelsRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	result, err := h.svc.SyncModels(ctx, id, req.Models)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, result)
}

func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	if err := h.svc.Delete(ctx, id); err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, nil)
}
