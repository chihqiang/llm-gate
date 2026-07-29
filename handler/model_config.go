package handler

import (
	"net/http"
	"strconv"

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
	var req logic.ModelListRequest
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

func (h *ModelHandler) AllList(w http.ResponseWriter, r *http.Request) {
	models, err := h.svc.AllList()
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, models)
}

func (h *ModelHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	mc, err := h.svc.GetByID(id)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, mc)
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.ModelCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	mc, err := h.svc.Create(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, mc)
}

func (h *ModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteHTTPError(w, httpx.CodeBadRequest, "无效的ID")
		return
	}

	var req logic.ModelUpdateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	req.ID = id

	mc, err := h.svc.Update(&req)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	httpx.OkJSON(w, mc)
}

func (h *ModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
