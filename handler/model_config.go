package handler

import (
	"github.com/chihqiang/ten-go/core/cast"
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/ten-go/rest"
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

func (h *ModelHandler) AllList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	models, err := h.svc.AllList(ctx)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, models)
}

func (h *ModelHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	mc, err := h.svc.GetByID(ctx, id)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, mc)
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req logic.ModelCreateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	mc, err := h.svc.Create(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, mc)
}

func (h *ModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := cast.To[int64](rest.PathParam(r, "id"))
	if id <= 0 {
		rest.ErrorJSONCtx(ctx, w, rest.NewCodeError(http.StatusBadRequest, "无效的ID"))
		return
	}

	var req logic.ModelUpdateRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	req.ID = id

	mc, err := h.svc.Update(ctx, &req)
	if err != nil {
		rest.OkJSONCtx(ctx, w, rest.NewCodeError(rest.BusinessCodeError, err.Error()))
		return
	}

	rest.OkJSONCtx(ctx, w, mc)
}

func (h *ModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
