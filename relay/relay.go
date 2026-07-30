package relay

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/ratelimit"
	"github.com/chihqiang/infra-go/syncx"
	"gorm.io/gorm"
)

type settleJob struct {
	tokenID                int64
	accountID              int64
	modelName              string
	providerID             int64
	requestID              string
	preConsumeQuota        int64
	actualQuota            int64
	actualPromptTokens     int
	actualCompletionTokens int
}

type RelayHandler struct {
	db             *gorm.DB
	config         config.RelayConfig
	client         *http.Client
	authCache      cache.Cache
	providerCache  cache.Cache
	modelListCache cache.Cache
	usageBatch     *usageBatch
	settleCh       chan settleJob
	rateLimiters   *syncx.ConcurrentMap[string, *ratelimit.TokenBucket]
}

func NewRelayHandler(db *gorm.DB, cfg config.RelayConfig, authCache, providerCache, modelListCache cache.Cache) *RelayHandler {
	settleCh := make(chan settleJob, 1024)
	h := &RelayHandler{
		db:     db,
		config: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   50,
				IdleConnTimeout:       90 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				DisableKeepAlives:     false,
			},
		},
		authCache:      authCache,
		providerCache:  providerCache,
		modelListCache: modelListCache,
		settleCh:       settleCh,
		rateLimiters:   syncx.NewConcurrentMap[string, *ratelimit.TokenBucket](),
	}
	h.usageBatch = newUsageBatch(db)
	for range 4 {
		go h.settleWorker()
	}
	return h
}

func (h *RelayHandler) Stop() {
	close(h.settleCh)
	h.usageBatch.Stop()
}

func (h *RelayHandler) settleWorker() {
	for job := range h.settleCh {
		if job.actualQuota > 0 {
			delta := job.actualQuota - job.preConsumeQuota
			if delta > 0 {
				if err := DeductTokenQuota(h.db, job.tokenID, delta); err != nil {
					logger.Error("relay settle: deduct delta failed",
						logger.Err(err), logger.Int64("token_id", job.tokenID), logger.Int64("delta", delta))
				}
			} else if delta < 0 {
				if err := RefundTokenQuota(h.db, job.tokenID, -delta); err != nil {
					logger.Error("relay settle: refund delta failed",
						logger.Err(err), logger.Int64("token_id", job.tokenID), logger.Int64("delta", -delta))
				}
			}

			h.usageBatch.Append(model.UsageLog{
				AccountID:        job.accountID,
				TokenID:          job.tokenID,
				ModelName:        job.modelName,
				ProviderID:       job.providerID,
				PromptTokens:     job.actualPromptTokens,
				CompletionTokens: job.actualCompletionTokens,
				TotalTokens:      job.actualPromptTokens + job.actualCompletionTokens,
				QuotaCost:        job.actualQuota,
				RequestID:        job.requestID,
			})
		} else {
			if err := RefundTokenQuota(h.db, job.tokenID, job.preConsumeQuota); err != nil {
				logger.Error("relay settle: refund pre-consume failed",
					logger.Err(err), logger.Int64("token_id", job.tokenID))
			}
		}
	}
}

func (h *RelayHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	key := extractBearerToken(r)
	if key == "" {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeUnauthorized, "missing authorization header"))
		return
	}

	authResult, err := h.authenticateCached(key)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeUnauthorized, err.Error()))
		return
	}

	// 速率限制：per-token 令牌桶
	if h.config.RateLimit.Enabled {
		limiter, _ := h.rateLimiters.GetOrSet(key, ratelimit.NewTokenBucket(
			h.config.RateLimit.Rate,
			h.config.RateLimit.Burst,
		))
		if !limiter.Allow() {
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "rate limit exceeded"))
			return
		}
	}

	// 限制请求体大小，防止 OOM
	maxBodyBytes := int64(h.config.MaxBodyMB) << 20
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "failed to read request body"))
		return
	}
	if int64(len(body)) > maxBodyBytes {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "request body too large"))
		return
	}

	// 只做一次 JSON 反序列化，后续提取 model/stream 和转发都复用此 map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "invalid request body"))
		return
	}

	modelName := extractFieldString(raw, "model")
	if modelName == "" {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "model is required"))
		return
	}

	if err := validateRequestBody(raw); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	resolveResult, err := h.resolveCached(modelName)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	requestID := generateRequestID()

	// 预扣额度：先尝试配置值，失败则降级为 1
	// 如果缓存中 quota 已不足，跳过首轮大额尝试，直接尝试 1
	preConsumeQuota := h.config.PreConsumeQuota
	if authResult.Token.Quota < preConsumeQuota {
		preConsumeQuota = 1
	}
	if err := DeductTokenQuota(h.db, authResult.Token.ID, preConsumeQuota); err != nil {
		if preConsumeQuota <= 1 {
			errMsg := "insufficient quota"
			if !errors.Is(err, ErrInsufficientQuota) {
				errMsg = "quota service unavailable"
			}
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, errMsg))
			return
		}
		preConsumeQuota = 1
		if err := DeductTokenQuota(h.db, authResult.Token.ID, 1); err != nil {
			errMsg := "insufficient quota"
			if !errors.Is(err, ErrInsufficientQuota) {
				errMsg = "quota service unavailable"
			}
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, errMsg))
			return
		}
	}

	isStream := extractFieldBool(raw, "stream")

	logger.Info("relay request start",
		logger.String("request_id", requestID),
		logger.String("model", modelName),
		logger.Bool("stream", isStream),
		logger.Int64("token_id", authResult.Token.ID),
		logger.Int64("account_id", authResult.Account.ID))

	// forward 会将响应写入 w 并返回 usage 数据
	// 如果上游请求失败或返回非 200 状态码，返回 error（响应已写入 w）
	usage, forwardErr := h.forward(w, r, resolveResult.Provider, resolveResult.Model, raw, isStream)
	if forwardErr != nil {
		logger.Warn("relay forward failed",
			logger.String("request_id", requestID),
			logger.Err(forwardErr))
		refundWithRetry(h.db, authResult.Token.ID, preConsumeQuota)
		return
	}

	var actualPromptTokens, actualCompletionTokens int
	var actualQuota int64

	if usage != nil {
		logger.Info("relay request complete",
			logger.String("request_id", requestID),
			logger.Int("prompt_tokens", usage.PromptTokens),
			logger.Int("completion_tokens", usage.CompletionTokens))
		actualPromptTokens = usage.PromptTokens
		actualCompletionTokens = usage.CompletionTokens
		actualQuota = CalculateQuota(usage.PromptTokens, usage.CompletionTokens, resolveResult.Model)
	}

	// 额度结算异步化：通过有界 channel + 固定 worker 处理，避免无限制 goroutine 暴涨
	select {
	case h.settleCh <- settleJob{
		tokenID:                authResult.Token.ID,
		accountID:              authResult.Account.ID,
		modelName:              resolveResult.Model.Name,
		providerID:             resolveResult.Provider.ID,
		requestID:              requestID,
		preConsumeQuota:        preConsumeQuota,
		actualQuota:            actualQuota,
		actualPromptTokens:     actualPromptTokens,
		actualCompletionTokens: actualCompletionTokens,
	}:
	default:
		logger.Warn("relay settle channel full, falling back to sync refund",
			logger.String("request_id", requestID))
		refundWithRetry(h.db, authResult.Token.ID, preConsumeQuota)
	}
}

type modelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func (h *RelayHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	key := extractBearerToken(r)
	if key == "" {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeUnauthorized, "missing authorization header"))
		return
	}

	if _, err := h.authenticateCached(key); err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeUnauthorized, err.Error()))
		return
	}

	// 模型列表缓存：优先从缓存读取，miss 则查 DB 并回填
	const modelsKey = "models:all"
	var cachedItems []modelItem
	if ok, _ := h.modelListCache.GetInto(modelsKey, &cachedItems); ok {
		httpx.OkJSON(w, map[string]interface{}{"object": "list", "data": cachedItems})
		return
	}

	logger.Info("relay models list cache miss")
	var models []model.ModelConfig
	if err := h.db.Where("status = ?", true).Find(&models).Error; err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
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
	h.modelListCache.Set(modelsKey, data, 30*time.Second)

	httpx.OkJSON(w, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

func (h *RelayHandler) authenticateCached(key string) (*AuthResult, error) {
	cacheKey := "auth:" + key
	var cached AuthResult
	if ok, _ := h.authCache.GetInto(cacheKey, &cached); ok {
		return &cached, nil
	}

	var token model.UserToken
	if err := h.db.Where("key = ? AND status = ?", key, true).First(&token).Error; err != nil {
		return nil, fmt.Errorf("invalid or disabled token")
	}
	logger.Info("relay auth cache miss", logger.Int64("token_id", token.ID))

	// 检查是否过期（使用 UTC 避免时区偏差）
	if token.ExpiredAt != nil && token.ExpiredAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("token expired")
	}

	if token.Quota <= 0 {
		return nil, fmt.Errorf("insufficient quota")
	}

	var account model.Account
	if err := h.db.First(&account, token.AccountID).Error; err != nil {
		return nil, fmt.Errorf("account not found")
	}

	if !account.Status {
		return nil, fmt.Errorf("account disabled")
	}

	result := &AuthResult{Token: &token, Account: &account}
	h.authCache.Set(cacheKey, result, 10*time.Second)
	return result, nil
}

func (h *RelayHandler) resolveCached(modelName string) (*ResolveResult, error) {
	listKey := "model_list:" + modelName
	var modelList []model.ModelConfig
	if ok, _ := h.modelListCache.GetInto(listKey, &modelList); !ok {
		// 阴性缓存：不存在的模型名也缓存 30s，防止穿透到 DB
		var neg struct{}
		if ok, _ := h.modelListCache.GetInto("neg:"+modelName, &neg); ok {
			return nil, fmt.Errorf("model not found or disabled")
		}
		logger.Info("relay model cache miss", logger.String("model", modelName))
		if err := h.db.Where("name = ? AND status = ?", modelName, true).
			Order("id ASC").
			Find(&modelList).Error; err != nil || len(modelList) == 0 {
			logger.Warn("relay model resolve failed", logger.String("model", modelName))
			h.modelListCache.Set("neg:"+modelName, struct{}{}, 30*time.Second)
			return nil, fmt.Errorf("model not found or disabled")
		}
		h.modelListCache.Set(listKey, modelList, 30*time.Second)
	}

	var mc model.ModelConfig
	if len(modelList) == 1 {
		mc = modelList[0]
	} else {
		totalWeight := 0
		for _, m := range modelList {
			totalWeight += m.Weight
		}
		if totalWeight <= 0 {
			mc = modelList[0]
		} else {
			roll := rand.Int64N(int64(totalWeight)) + 1
			pos := 0
			for _, m := range modelList {
				pos += m.Weight
				if roll <= int64(pos) {
					mc = m
					break
				}
			}
		}
	}

	providerKey := "provider:" + strconv.FormatInt(mc.ProviderID, 10)
	var provider model.Provider
	v, cached := h.providerCache.Get(providerKey)
	if cached {
		if p, ok := v.(*model.Provider); ok {
			provider = *p
		} else {
			cached = false
		}
	}
	if !cached {
		logger.Info("relay provider cache miss", logger.Int64("provider_id", mc.ProviderID))
		if err := h.db.Where("id = ? AND status = ?", mc.ProviderID, true).First(&provider).Error; err != nil {
			return nil, fmt.Errorf("provider not found or disabled")
		}
		h.providerCache.Set(providerKey, &provider, 60*time.Second)
	}
	return &ResolveResult{Model: &mc, Provider: &provider}, nil
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// extractFieldString 从已解析的 JSON map 中提取字符串字段
func extractFieldString(raw map[string]json.RawMessage, key string) string {
	v, ok := raw[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		logger.Warn("relay: extract field string failed",
			logger.String("key", key), logger.Err(err))
	}
	return s
}

// extractFieldBool 从已解析的 JSON map 中提取布尔字段
func extractFieldBool(raw map[string]json.RawMessage, key string) bool {
	v, ok := raw[key]
	if !ok {
		return false
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		logger.Warn("relay: extract field bool failed",
			logger.String("key", key), logger.Err(err))
	}
	return b
}

func refundWithRetry(db *gorm.DB, tokenID int64, quota int64) {
	for retries := 3; retries > 0; retries-- {
		if err := RefundTokenQuota(db, tokenID, quota); err == nil {
			return
		}
		time.Sleep(time.Duration(50*retries) * time.Millisecond)
	}
	logger.Error("relay: refund failed after retries",
		logger.Int64("token_id", tokenID), logger.Int64("quota", quota))
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := cryptorand.Read(b); err != nil {
		return fmt.Sprintf("chat_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("chat_%s", hex.EncodeToString(b))
}

// validateRequestBody 校验请求体中常见字段的合法性。
// 仅校验已知字段，不限制未知字段透传。
func validateRequestBody(raw map[string]json.RawMessage) error {
	if v, ok := raw["max_tokens"]; ok {
		var n int
		if err := json.Unmarshal(v, &n); err == nil && n < 1 {
			return fmt.Errorf("max_tokens must be positive")
		}
	}
	if v, ok := raw["temperature"]; ok {
		var f float64
		if err := json.Unmarshal(v, &f); err == nil && (f < 0 || f > 2) {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
	}
	if v, ok := raw["top_p"]; ok {
		var f float64
		if err := json.Unmarshal(v, &f); err == nil && (f < 0 || f > 1) {
			return fmt.Errorf("top_p must be between 0 and 1")
		}
	}
	if v, ok := raw["n"]; ok {
		var n int
		if err := json.Unmarshal(v, &n); err == nil && n < 1 {
			return fmt.Errorf("n must be positive")
		}
	}
	return nil
}
