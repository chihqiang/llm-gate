package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
	"gorm.io/gorm"
)

type RelayHandler struct {
	db     *gorm.DB
	config config.RelayConfig
}

func NewRelayHandler(db *gorm.DB, cfg config.RelayConfig) *RelayHandler {
	return &RelayHandler{db: db, config: cfg}
}

func (h *RelayHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	authResult, err := Authenticate(r, h.db)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeUnauthorized, err.Error()))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "failed to read request body"))
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	resolveResult, err := ResolveModel(r, h.db)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	requestID := generateRequestID()

	preConsumeQuota := int64(1000)

	if err := DeductTokenQuota(h.db, authResult.Token.ID, preConsumeQuota); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "insufficient quota"))
		return
	}

	isStream := isStreamRequest(body)

	var recorder *responseRecorder
	if !isStream {
		recorder = &responseRecorder{ResponseWriter: w}
		Forward(recorder, r, resolveResult.Provider, resolveResult.Model, body)
	} else {
		Forward(w, r, resolveResult.Provider, resolveResult.Model, body)
	}

	var actualPromptTokens, actualCompletionTokens int
	var actualQuota int64

	if !isStream && recorder != nil && recorder.body != nil {
		usage := extractUsage([]byte(recorder.body.String()))
		if usage != nil {
			actualPromptTokens = usage.PromptTokens
			actualCompletionTokens = usage.CompletionTokens
			actualQuota = CalculateQuota(usage.PromptTokens, usage.CompletionTokens, resolveResult.Model)
		}
	}

	if actualQuota > 0 {
		delta := actualQuota - preConsumeQuota
		if delta > 0 {
			DeductTokenQuota(h.db, authResult.Token.ID, delta)
		} else if delta < 0 {
			RefundTokenQuota(h.db, authResult.Token.ID, -delta)
		}

		if err := RecordUsage(h.db, authResult.Account.ID, authResult.Token.ID, resolveResult.Provider.ID,
			resolveResult.Model.Name, actualPromptTokens, actualCompletionTokens, actualQuota, requestID); err != nil {
			logger.Error("failed to record usage", logger.Err(err))
		}
	} else {
		RefundTokenQuota(h.db, authResult.Token.ID, preConsumeQuota)
	}
}

func (h *RelayHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	_, err := Authenticate(r, h.db)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeUnauthorized, err.Error()))
		return
	}

	var models []model.ModelConfig
	if err := h.db.Where("status = ?", true).Find(&models).Error; err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	type modelItem struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	data := make([]modelItem, 0, len(models))
	for _, m := range models {
		data = append(data, modelItem{
			ID:      m.Name,
			Object:  "model",
			Created: m.CreatedAt.Unix(),
			OwnedBy: "llm-gate",
		})
	}

	httpx.OkJSON(w, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *strings.Builder
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.body == nil {
		r.body = &strings.Builder{}
	}
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func isStreamRequest(body []byte) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	return req.Stream
}

func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("chat_%s", hex.EncodeToString(b))
}
