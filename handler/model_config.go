package handler

import (
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/infra-go/httpx"
)

type ModelHandler struct {
	svc *logic.ModelLogic
}

func NewModelHandler(svc *logic.ModelLogic) *ModelHandler {
	return &ModelHandler{svc: svc}
}

func (h *ModelHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.ModelListRequest
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

func (h *ModelHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	models, err := h.svc.AllList(ctx)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, models)
}

func (h *ModelHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	mc, err := h.svc.GetByID(ctx, id)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, mc)
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.ModelCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	mc, err := h.svc.Create(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, mc)
}

func (h *ModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := httpx.PathValue[int64](r, "id")
	if id <= 0 {
		httpx.WriteHTTPErrorCtx(ctx, w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.ModelUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	mc, err := h.svc.Update(ctx, &req)
	if err != nil {
		httpx.OkJSONCtx(ctx, w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSONCtx(ctx, w, mc)
}

func (h *ModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
