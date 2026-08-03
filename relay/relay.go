package relay

import (
	"context"
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
	"chihqiang/llm-gate/security"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/ratelimit"
	"github.com/chihqiang/infra-go/syncx"
	"gorm.io/gorm"
)

// settleJob 请求结算任务，由有界 channel + 固定 worker 异步处理。
type settleJob struct {
	accountID            int64
	tokenID              int64
	modelName            string
	providerID           int64
	requestID            string
	preConsumeCents      int64
	actualCents          int64
	estimated            bool
	actualPromptTokens   int
	actualCompletionTokens int
}

type RelayHandler struct {
	db             *gorm.DB
	relayCfg       config.RelayConfig
	billingCfg     config.BillingConfig
	cipher         *security.Cipher
	client         *http.Client
	authCache      cache.Cache
	providerCache  cache.Cache
	modelListCache cache.Cache
	usageBatch     *usageBatch
	settleCh       chan settleJob
	rateLimiters   *syncx.ConcurrentMap[string, *ratelimit.TokenBucket]
	accountLimiters *syncx.ConcurrentMap[int64, *ratelimit.TokenBucket]
	globalLimiter  *ratelimit.TokenBucket
	breakers       *breakerManager
	notifier       *notifier
	probeStop      chan struct{}
}

func NewRelayHandler(db *gorm.DB, cfg config.Config, cipher *security.Cipher, authCache, providerCache, modelListCache cache.Cache) *RelayHandler {
	settleCh := make(chan settleJob, 1024)
	h := &RelayHandler{
		db:     db,
		relayCfg: cfg.Relay,
		billingCfg: cfg.Billing,
		cipher: cipher,
		client: &http.Client{
			Timeout: time.Duration(cfg.Relay.Timeout) * time.Second,
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
		accountLimiters: syncx.NewConcurrentMap[int64, *ratelimit.TokenBucket](),
		breakers:       newBreakerManager(cfg.Relay.Failover.FailureThreshold, cfg.Relay.Failover.Window, cfg.Relay.Failover.Cooldown),
		notifier:       newNotifier(cfg.Alert),
		probeStop:      make(chan struct{}),
	}
	if cfg.Relay.RateLimit.GlobalRate > 0 {
		h.globalLimiter = ratelimit.NewTokenBucket(cfg.Relay.RateLimit.GlobalRate, cfg.Relay.RateLimit.GlobalBurst)
	}
	h.usageBatch = newUsageBatch(db)
	for range 4 {
		go h.settleWorker()
	}
	if cfg.Relay.Failover.Enabled && cfg.Relay.Failover.HealthProbeEnabled {
		go h.probeLoop()
	}
	return h
}

func (h *RelayHandler) Stop() {
	close(h.settleCh)
	close(h.probeStop)
	h.usageBatch.Stop()
}

func (h *RelayHandler) settleWorker() {
	for job := range h.settleCh {
		if job.actualCents > 0 {
			delta := job.actualCents - job.preConsumeCents
			var err error
			if delta > 0 {
				err = DeductBalance(h.db, job.accountID, delta)
			} else if delta < 0 {
				err = RefundBalance(h.db, job.accountID, -delta)
			}
			if err != nil {
				logger.Error("relay settle: balance settle failed, refund pre-consume",
					logger.Err(err),
					logger.Int64("account_id", job.accountID),
					logger.Int64("pre_consume", job.preConsumeCents))
				// 结算失败时退回首轮预扣，避免用户被多扣
				_ = RefundBalance(h.db, job.accountID, job.preConsumeCents)
				continue
			}

			if err := AddTokenSpent(h.db, job.tokenID, job.actualCents); err != nil {
				logger.Error("relay settle: add token spent failed",
					logger.Err(err), logger.Int64("token_id", job.tokenID))
			}
			h.appendBalanceTxn(job, -job.actualCents, model.TransactionConsume,
				fmt.Sprintf("消费：模型 %s", job.modelName))

			h.usageBatch.Append(model.UsageLog{
				AccountID:        job.accountID,
				TokenID:          job.tokenID,
				ModelName:        job.modelName,
				ProviderID:       job.providerID,
				PromptTokens:     job.actualPromptTokens,
				CompletionTokens: job.actualCompletionTokens,
				TotalTokens:      job.actualPromptTokens + job.actualCompletionTokens,
				CostCents:        job.actualCents,
				Estimated:        job.estimated,
				RequestID:        job.requestID,
			})
		} else {
			// 无实际费用：退还预扣
			if err := RefundBalance(h.db, job.accountID, job.preConsumeCents); err != nil {
				logger.Error("relay settle: refund pre-consume failed",
					logger.Err(err), logger.Int64("account_id", job.accountID))
				continue
			}
			h.appendBalanceTxn(job, job.preConsumeCents, model.TransactionRefund,
				fmt.Sprintf("退款：模型 %s", job.modelName))
		}
	}
}

// appendBalanceTxn 写入余额流水（余额快照读取当前值）。
func (h *RelayHandler) appendBalanceTxn(job settleJob, amountCents int64, txnType, remark string) {
	balance, err := GetAccountBalance(h.db, job.accountID)
	if err != nil {
		return
	}
	if err := h.db.Create(&model.Transaction{
		AccountID:    job.accountID,
		Type:         txnType,
		AmountCents:  amountCents,
		BalanceCents: balance,
		TokenID:      job.tokenID,
		RequestID:    job.requestID,
		Remark:       remark,
	}).Error; err != nil {
		logger.Error("relay settle: append transaction failed", logger.Err(err))
	}
}

// ChatCompletions 对话补全（流式/非流式）。
func (h *RelayHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	h.relay(w, r, kindChat, false)
}

// Embeddings 向量化接口（非流式）。
func (h *RelayHandler) Embeddings(w http.ResponseWriter, r *http.Request) {
	h.relay(w, r, kindEmbeddings, true)
}

// relay 通用转发流程：认证 → 限流 → 解析模型 → 预扣余额 → 转发 → 异步结算。
func (h *RelayHandler) relay(w http.ResponseWriter, r *http.Request, kind string, forceNonStream bool) {
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

	// 限流：token 级 → 账户级 → 全局
	rl := h.relayCfg.RateLimit
	if rl.Enabled {
		limiter, _ := h.rateLimiters.GetOrSet(key, ratelimit.NewTokenBucket(rl.Rate, rl.Burst))
		if !limiter.Allow() {
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "rate limit exceeded"))
			return
		}
	}
	if rl.AccountRate > 0 {
		limiter, _ := h.accountLimiters.GetOrSet(authResult.Account.ID,
			ratelimit.NewTokenBucket(rl.AccountRate, rl.AccountBurst))
		if !limiter.Allow() {
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "account rate limit exceeded"))
			return
		}
	}
	if h.globalLimiter != nil && !h.globalLimiter.Allow() {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "global rate limit exceeded"))
		return
	}

	// 限制请求体大小，防止 OOM
	maxBodyBytes := int64(h.relayCfg.MaxBodyMB) << 20
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "failed to read request body"))
		return
	}
	if int64(len(body)) > maxBodyBytes {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, "request body too large"))
		return
	}

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

	resolveResult, err := h.resolveCached(modelName, authResult.Token)
	if err != nil {
		httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
		return
	}

	requestID := h.requestID(r)

	isStream := !forceNonStream && extractFieldBool(raw, "stream")

	// 预扣余额：先尝试配置值，余额不足则降级为 1
	preConsumeCents := h.relayCfg.PreConsumeCents
	if err := DeductBalance(h.db, authResult.Account.ID, preConsumeCents); err != nil {
		preConsumeCents = 1
		if err := DeductBalance(h.db, authResult.Account.ID, 1); err != nil {
			errMsg := "insufficient balance"
			if !errors.Is(err, ErrInsufficientQuota) {
				errMsg = "billing service unavailable"
			}
			h.notifyBalanceLow(authResult.Account)
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, errMsg))
			return
		}
	}

	logger.Info("relay request start",
		logger.String("request_id", requestID),
		logger.String("model", modelName),
		logger.String("kind", kind),
		logger.Bool("stream", isStream),
		logger.Int64("token_id", authResult.Token.ID),
		logger.Int64("account_id", authResult.Account.ID))

	usage, delivered, forwardErr := h.forward(w, r, kind, resolveResult.Candidates, raw, isStream, requestID)
	if forwardErr != nil {
		logger.Warn("relay forward failed",
			logger.String("request_id", requestID),
			logger.Err(forwardErr))
		refundWithRetry(h.db, authResult.Account.ID, preConsumeCents)
		h.appendBalanceTxn(settleJob{
			accountID:       authResult.Account.ID,
			tokenID:         authResult.Token.ID,
			requestID:       requestID,
			preConsumeCents: preConsumeCents,
		}, preConsumeCents, model.TransactionRefund, "退款：上游失败")
		return
	}

	var actualPromptTokens, actualCompletionTokens int
	var actualCents int64
	estimated := false

	if usage != nil {
		actualPromptTokens = usage.PromptTokens
		actualCompletionTokens = usage.CompletionTokens
		actualCents = CalculateCostCents(usage.PromptTokens, usage.CompletionTokens, resolveResult.Model, h.billingCfg.BasePriceCentsPer1K)
		logger.Info("relay request complete",
			logger.String("request_id", requestID),
			logger.Int("prompt_tokens", usage.PromptTokens),
			logger.Int("completion_tokens", usage.CompletionTokens),
			logger.Int64("cost_cents", actualCents))
	} else if delivered {
		// 成功响应但未拿到 usage（流式未返回 / 上游不支持），按兜底费用计费，防止免费滥用
		estimated = true
		actualCents = h.relayCfg.StreamFallbackCents
		if actualCents <= 0 {
			actualCents = h.relayCfg.PreConsumeCents
		}
		logger.Warn("relay usage fallback charged",
			logger.String("request_id", requestID),
			logger.Int64("cost_cents", actualCents))
	}

	select {
	case h.settleCh <- settleJob{
		accountID:            authResult.Account.ID,
		tokenID:              authResult.Token.ID,
		modelName:            resolveResult.Model.Name,
		providerID:           resolveResult.Provider.ID,
		requestID:            requestID,
		preConsumeCents:      preConsumeCents,
		actualCents:          actualCents,
		estimated:            estimated,
		actualPromptTokens:   actualPromptTokens,
		actualCompletionTokens: actualCompletionTokens,
	}:
	default:
		logger.Warn("relay settle channel full, falling back to sync refund",
			logger.String("request_id", requestID))
		refundWithRetry(h.db, authResult.Account.ID, preConsumeCents)
	}
}

func (h *RelayHandler) ListModels(w http.ResponseWriter, r *http.Request) {
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

	allowedIDs := parseTokenModelIDs(authResult.Token.ModelIDs)

	// 有模型白名单时按 ID 过滤，需从 DB 取最新配置，不走全量缓存
	if len(allowedIDs) > 0 {
		allowed := make(map[int64]bool, len(allowedIDs))
		for _, id := range allowedIDs {
			allowed[id] = true
		}
		var models []model.ModelConfig
		if err := h.db.Where("status = ?", true).Find(&models).Error; err != nil {
			httpx.OkJSON(w, httpx.NewCodeError(httpx.CodeDefaultError, err.Error()))
			return
		}
		data := make([]modelItem, 0, len(models))
		for _, m := range models {
			if allowed[m.ID] {
				data = append(data, modelItem{ID: m.Name, Object: "model", Created: m.CreatedAt.Unix(), OwnedBy: "llm-gate"})
			}
		}
		httpx.OkJSON(w, map[string]interface{}{"object": "list", "data": data})
		return
	}

	// 无白名单：走模型列表缓存
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

	keyHash := security.SHA256Hex(key)
	var token model.UserToken
	if err := h.db.Where("key_hash = ? AND status = ?", keyHash, true).First(&token).Error; err != nil {
		return nil, fmt.Errorf("invalid or disabled token")
	}
	logger.Info("relay auth cache miss", logger.Int64("token_id", token.ID))

	if token.ExpiredAt != nil && token.ExpiredAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("token expired")
	}

	// 预算：quota>0 表示该 Key 的累计消费上限，0 表示不限
	if token.Quota > 0 && token.SpentCents >= token.Quota {
		return nil, fmt.Errorf("key budget exhausted")
	}

	var account model.Account
	if err := h.db.First(&account, token.AccountID).Error; err != nil {
		return nil, fmt.Errorf("account not found")
	}
	if !account.Status {
		return nil, fmt.Errorf("account disabled")
	}
	if account.BalanceCents <= 0 {
		return nil, fmt.Errorf("insufficient balance")
	}

	result := &AuthResult{Token: &token, Account: &account}
	h.authCache.Set(cacheKey, result, 10*time.Second)
	return result, nil
}

// resolveCached 解析模型并构建候选服务商列表。
// 应用 Token 模型白名单过滤，按权重随机选主候选，其余按权重降序作为降级候选。
func (h *RelayHandler) resolveCached(modelName string, token *model.UserToken) (*ResolveResult, error) {
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

	// 模型白名单过滤
	allowedIDs := parseTokenModelIDs(token.ModelIDs)
	if len(allowedIDs) > 0 {
		allowed := make(map[int64]bool, len(allowedIDs))
		for _, id := range allowedIDs {
			allowed[id] = true
		}
		filtered := modelList[:0]
		for _, m := range modelList {
			if allowed[m.ID] {
				filtered = append(filtered, m)
			}
		}
		modelList = filtered
		if len(modelList) == 0 {
			return nil, fmt.Errorf("model not allowed for this key")
		}
	}

	// 构建候选并排序：权重随机选主候选，其余按权重降序
	candidates, err := h.buildCandidates(modelList)
	if err != nil {
		return nil, err
	}

	return &ResolveResult{
		Model:      candidates[0].Model,
		Provider:   candidates[0].Provider,
		Candidates: candidates,
	}, nil
}

// buildCandidates 为模型列表构建候选：加权随机选主候选，其余按权重降序排列。
func (h *RelayHandler) buildCandidates(modelList []model.ModelConfig) ([]upstreamCandidate, error) {
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

	// 剩余候选按权重降序、ID 升序排列
	rest := make([]model.ModelConfig, 0, len(modelList)-1)
	for _, m := range modelList {
		if m.ID == mc.ID {
			continue
		}
		rest = append(rest, m)
	}
	sortModelsByWeight(rest)

	order := append([]model.ModelConfig{mc}, rest...)
	candidates := make([]upstreamCandidate, 0, len(order))
	for _, m := range order {
		provider, err := h.loadProvider(m.ProviderID)
		if err != nil {
			continue
		}
		candidates = append(candidates, upstreamCandidate{Model: &m, Provider: provider})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("provider not found or disabled")
	}
	return candidates, nil
}

func sortModelsByWeight(models []model.ModelConfig) {
	for i := 1; i < len(models); i++ {
		for j := i; j > 0 && (models[j].Weight > models[j-1].Weight ||
			(models[j].Weight == models[j-1].Weight && models[j].ID < models[j-1].ID)); j-- {
			models[j], models[j-1] = models[j-1], models[j]
		}
	}
}

// loadProvider 从缓存或 DB 加载服务商，解密 API 密钥。
func (h *RelayHandler) loadProvider(providerID int64) (*model.Provider, error) {
	providerKey := "provider:" + strconv.FormatInt(providerID, 10)
	var provider model.Provider
	if v, cached := h.providerCache.Get(providerKey); cached {
		if p, ok := v.(*model.Provider); ok {
			provider = *p
			return &provider, nil
		}
	}
	logger.Info("relay provider cache miss", logger.Int64("provider_id", providerID))
	if err := h.db.Where("id = ? AND status = ?", providerID, true).First(&provider).Error; err != nil {
		return nil, fmt.Errorf("provider not found or disabled")
	}
	key, err := h.cipher.Decrypt(provider.APIKey)
	if err != nil {
		return nil, fmt.Errorf("provider key decrypt failed")
	}
	provider.APIKey = key
	h.providerCache.Set(providerKey, &provider, 60*time.Second)
	return &provider, nil
}

// requestID 优先沿用客户端 X-Request-ID，否则生成。
func (h *RelayHandler) requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return generateRequestID()
}

func (h *RelayHandler) notifyBalanceLow(account *model.Account) {
	h.notifier.Send(fmt.Sprintf("balance_low_%d", account.ID), "账户余额不足",
		fmt.Sprintf("账户 %s (ID=%d) 余额不足，请求被拒绝", account.Name, account.ID))
}

func (h *RelayHandler) probeLoop() {
	interval := h.relayCfg.Failover.HealthProbeInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.probeAll()
		case <-h.probeStop:
			return
		}
	}
}

func (h *RelayHandler) probeAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var providers []model.Provider
	if err := h.db.Select("id", "base_url", "api_key", "status").Where("status = ?", true).Find(&providers).Error; err != nil {
		return
	}
	for _, p := range providers {
		if h.breakers.get(p.ID).IsOpen() {
			continue
		}
		key, err := h.cipher.Decrypt(p.APIKey)
		if err != nil {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			strings.TrimRight(p.BaseURL, "/")+"/v1/models", nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := h.client.Do(req)
		if err != nil {
			h.breakers.get(p.ID).Failure()
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			h.breakers.get(p.ID).Failure()
			continue
		}
		h.breakers.get(p.ID).Success()
	}
}

type modelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
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

func parseTokenModelIDs(raw string) []int64 {
	if raw == "" {
		return nil
	}
	var ids []int64
	_ = json.Unmarshal([]byte(raw), &ids)
	return ids
}

func refundWithRetry(db *gorm.DB, accountID, cents int64) {
	for retries := 3; retries > 0; retries-- {
		if err := RefundBalance(db, accountID, cents); err == nil {
			return
		}
		time.Sleep(time.Duration(50*retries) * time.Millisecond)
	}
	logger.Error("relay: refund failed after retries",
		logger.Int64("account_id", accountID), logger.Int64("cents", cents))
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := cryptorand.Read(b); err != nil {
		return fmt.Sprintf("chat_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("chat_%s", hex.EncodeToString(b))
}

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
