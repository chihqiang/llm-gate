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
	"github.com/chihqiang/infra-go/trace"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// settleJob 请求结算任务，由有界 channel + 固定 worker 异步处理。
// ctx 为脱离请求生命周期、仅保留链路上下文的 context，供结算日志携带 trace_id/span_id。
type settleJob struct {
	ctx                    context.Context
	accountID              int64
	tokenID                int64
	modelName              string
	providerID             int64
	requestID              string
	preConsumeCents        int64
	reserveQuotaCents      int64
	actualCents            int64
	estimated              bool
	actualPromptTokens     int
	actualCompletionTokens int
}

type RelayHandler struct {
	db              *gorm.DB
	relayCfg        config.RelayConfig
	billingCfg      config.BillingConfig
	cipher          *security.Cipher
	client          *http.Client
	authCache       cache.Cache
	providerCache   cache.Cache
	modelListCache  cache.Cache
	usageBatch      *usageBatch
	settleCh        chan settleJob
	rateLimiters    *syncx.ConcurrentMap[string, *ratelimit.TokenBucket]
	accountLimiters *syncx.ConcurrentMap[int64, *ratelimit.TokenBucket]
	globalLimiter   *ratelimit.TokenBucket
	breakers        *breakerManager
	notifier        *notifier
	probeStop       chan struct{}
}

func NewRelayHandler(db *gorm.DB, cfg config.Config, cipher *security.Cipher, authCache, providerCache, modelListCache cache.Cache) *RelayHandler {
	settleCh := make(chan settleJob, 1024)
	h := &RelayHandler{
		db:         db,
		relayCfg:   cfg.Relay,
		billingCfg: cfg.Billing,
		cipher:     cipher,
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
		authCache:       authCache,
		providerCache:   providerCache,
		modelListCache:  modelListCache,
		settleCh:        settleCh,
		rateLimiters:    syncx.NewConcurrentMap[string, *ratelimit.TokenBucket](),
		accountLimiters: syncx.NewConcurrentMap[int64, *ratelimit.TokenBucket](),
		breakers:        newBreakerManager(cfg.Relay.Failover.FailureThreshold, cfg.Relay.Failover.Window, cfg.Relay.Failover.Cooldown),
		notifier:        newNotifier(cfg.Alert),
		probeStop:       make(chan struct{}),
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
		h.processJob(job)
	}
}

// processJob 执行单个结算任务：多退少补 + 写流水 + 累计 Token 消费 + 用量日志。
// 资金变动与流水写入在同一事务中，保证账目一致。
func (h *RelayHandler) processJob(job settleJob) {
	_, span := trace.StartSpan(job.ctx, "relay settle",
		trace.WithAttributes(
			trace.AttrString("request_id", job.requestID),
			trace.AttrInt64("account_id", job.accountID),
			trace.AttrInt64("token_id", job.tokenID),
			trace.AttrString("model", job.modelName),
			trace.AttrInt64("pre_consume_cents", job.preConsumeCents),
			trace.AttrInt64("actual_cents", job.actualCents),
			trace.AttrBool("estimated", job.estimated),
		),
	)
	defer span.End()

	err := h.db.WithContext(job.ctx).Transaction(func(tx *gorm.DB) error {
		if job.actualCents <= 0 {
			// 无实际费用：退还预扣并释放预占额度
			if err := RefundBalanceTx(tx, job.accountID, job.preConsumeCents); err != nil {
				return err
			}
			if err := AdjustTokenSpentTx(tx, job.tokenID, -job.reserveQuotaCents); err != nil {
				return err
			}
			return h.appendBalanceTxnTx(tx, job, job.preConsumeCents, model.TransactionRefund,
				fmt.Sprintf("退款：模型 %s", job.modelName))
		}

		delta := job.actualCents - job.preConsumeCents
		if delta > 0 {
			if err := DeductBalanceTx(tx, job.accountID, delta); err != nil {
				return err
			}
		} else if delta < 0 {
			if err := RefundBalanceTx(tx, job.accountID, -delta); err != nil {
				return err
			}
		}
		// 结算时把预占额度调整为实际消费（spent_cents 已含预占 reserveQuotaCents）
		if err := AdjustTokenSpentTx(tx, job.tokenID, job.actualCents-job.reserveQuotaCents); err != nil {
			return err
		}
		return h.appendBalanceTxnTx(tx, job, -job.actualCents, model.TransactionConsume,
			fmt.Sprintf("消费：模型 %s", job.modelName))
	})

	if err != nil {
		logger.ErrorCtx(job.ctx, "relay settle: settle failed, keep pre-consume for manual review",
			logger.Err(err),
			logger.Int64("account_id", job.accountID),
			logger.Int64("token_id", job.tokenID),
			logger.Int64("pre_consume", job.preConsumeCents),
			logger.Int64("actual_cents", job.actualCents))
		// 结算失败不退还预扣：请求已消费上游资源，退还会造成免费滥用。
		// 保留 preConsume 扣款，释放预占额度，等待人工对账。
		_ = AdjustTokenSpentTx(h.db.WithContext(job.ctx), job.tokenID, -job.reserveQuotaCents)
		h.notifier.Send(job.ctx, fmt.Sprintf("settle_fail_%d", job.accountID), "账单结算失败",
			fmt.Sprintf("账户 %d 请求 %s 结算失败，已按预扣 %d 分计费，请人工对账",
				job.accountID, job.requestID, job.preConsumeCents))
		return
	}

	// 用量日志走异步批量写入，不计入资金事务
	if job.actualCents > 0 {
		h.usageBatch.Append(model.UsageLog{
			AccountID:        job.accountID,
			TokenID:          job.tokenID,
			ModelName:        job.modelName,
			ProviderID:       job.providerID,
			PromptTokens:     job.actualPromptTokens,
			CompletionTokens: job.actualCompletionTokens,
			TotalTokens:      job.actualPromptTokens + job.actualCompletionTokens,
			QuotaCost:        job.actualCents,
			CostCents:        job.actualCents,
			Estimated:        job.estimated,
			RequestID:        job.requestID,
		})
	}
}

// appendBalanceTxnTx 在给定事务内写入余额流水，余额快照读取当前值。
func (h *RelayHandler) appendBalanceTxnTx(tx *gorm.DB, job settleJob, amountCents int64, txnType, remark string) error {
	balance, err := GetAccountBalanceTx(tx, job.accountID)
	if err != nil {
		return err
	}
	return tx.Create(&model.Transaction{
		AccountID:    job.accountID,
		Type:         txnType,
		AmountCents:  amountCents,
		BalanceCents: balance,
		TokenID:      job.tokenID,
		RequestID:    job.requestID,
		Remark:       remark,
	}).Error
}

// appendBalanceTxn 独立写入余额流水（非事务场景使用，如预扣失败退款）。
func (h *RelayHandler) appendBalanceTxn(job settleJob, amountCents int64, txnType, remark string) {
	if err := h.appendBalanceTxnTx(h.db.WithContext(job.ctx), job, amountCents, txnType, remark); err != nil {
		logger.ErrorCtx(job.ctx, "relay settle: append transaction failed", logger.Err(err))
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
	ctx := r.Context()
	// 优先沿用全局中间件注入的 request_id（与响应头 X-Request-Id 一致）
	requestID := httpx.RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = h.requestID(r)
	}
	ctx, span := trace.StartSpan(ctx, "relay "+kind,
		trace.WithAttributes(
			trace.AttrString("request_id", requestID),
			trace.AttrString("kind", kind),
		),
	)
	defer span.End()

	key := extractBearerToken(r)
	if key == "" {
		writeOpenAIError(w, http.StatusUnauthorized, "authentication_error", "missing authorization header")
		return
	}

	authResult, err := h.authenticateCached(ctx, key)
	if err != nil {
		writeOpenAIError(w, http.StatusUnauthorized, "authentication_error", err.Error())
		return
	}

	// 限流：token 级 → 账户级 → 全局
	rl := h.relayCfg.RateLimit
	if rl.Enabled {
		limiter, _ := h.rateLimiters.GetOrSet(key, ratelimit.NewTokenBucket(rl.Rate, rl.Burst))
		if !limiter.Allow() {
			writeOpenAIError(w, http.StatusTooManyRequests, "rate_limit_error", "rate limit exceeded")
			return
		}
	}
	if rl.AccountRate > 0 {
		limiter, _ := h.accountLimiters.GetOrSet(authResult.Account.ID,
			ratelimit.NewTokenBucket(rl.AccountRate, rl.AccountBurst))
		if !limiter.Allow() {
			writeOpenAIError(w, http.StatusTooManyRequests, "rate_limit_error", "account rate limit exceeded")
			return
		}
	}
	if h.globalLimiter != nil && !h.globalLimiter.Allow() {
		writeOpenAIError(w, http.StatusTooManyRequests, "rate_limit_error", "global rate limit exceeded")
		return
	}

	// 限制请求体大小，防止 OOM
	maxBodyBytes := int64(h.relayCfg.MaxBodyMB) << 20
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}
	if int64(len(body)) > maxBodyBytes {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "request body too large")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "invalid request body")
		return
	}

	modelName := extractFieldString(ctx, raw, "model")
	if modelName == "" {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	if err := validateRequestBody(raw); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	resolveResult, err := h.resolveCached(ctx, modelName, authResult.Token)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	span.SetAttributes(trace.AttrString("model", modelName))

	isStream := !forceNonStream && extractFieldBool(ctx, raw, "stream")

	// 预扣余额：余额必须足以支付预扣金额，不足直接拒绝（不允许降级预扣，避免账单漏洞）
	preConsumeCents := h.relayCfg.PreConsumeCents
	if err := DeductBalance(ctx, h.db, authResult.Account.ID, preConsumeCents); err != nil {
		errMsg := "insufficient balance"
		errType := "insufficient_quota"
		status := http.StatusPaymentRequired
		if !errors.Is(err, ErrInsufficientQuota) {
			errMsg = "billing service unavailable"
			errType = "api_error"
			status = http.StatusInternalServerError
		}
		h.notifyBalanceLow(ctx, authResult.Account)
		writeOpenAIError(w, status, errType, errMsg)
		return
	}

	// 原子预占 Token 预算额度（最多预扣金额，保证并发下不超预算；预算耗尽时退回余额预扣并拒绝）
	reserveQuotaCents := preConsumeCents
	if authResult.Token.Quota > 0 && reserveQuotaCents > authResult.Token.Quota {
		reserveQuotaCents = authResult.Token.Quota
	}
	if err := ReserveTokenQuota(h.db.WithContext(ctx), authResult.Token.ID, reserveQuotaCents, authResult.Token.Quota); err != nil {
		_ = RefundBalance(ctx, h.db, authResult.Account.ID, preConsumeCents)
		if errors.Is(err, ErrQuotaExhausted) {
			writeOpenAIError(w, http.StatusPaymentRequired, "insufficient_quota", "key budget exhausted")
		} else {
			logger.ErrorCtx(ctx, "relay: reserve token quota failed", logger.Err(err),
				logger.Int64("token_id", authResult.Token.ID))
			writeOpenAIError(w, http.StatusInternalServerError, "api_error", "billing service unavailable")
		}
		return
	}

	logger.InfoCtx(ctx, "relay request start",
		logger.String("model", modelName),
		logger.String("kind", kind),
		logger.Bool("stream", isStream),
		logger.Int64("token_id", authResult.Token.ID),
		logger.Int64("account_id", authResult.Account.ID))

	usage, delivered, forwardErr, winner := h.forward(ctx, w, r, kind, resolveResult.Candidates, raw, isStream)
	if forwardErr != nil {
		span.SetStatus(codes.Error, forwardErr.Error())
		span.RecordError(forwardErr)
		logger.WarnCtx(ctx, "relay forward failed",
			logger.Err(forwardErr))
		// 已提交响应（流式中断）：上游可能已产出内容，按 usage/投递情况结算，不退款
		if _, committed := forwardErr.(*committedError); committed {
			h.finalizeAndSettle(ctx, authResult, resolveResult, requestID, preConsumeCents, reserveQuotaCents, usage, delivered, winner)
			return
		}
		refundWithRetry(ctx, h.db, authResult.Account.ID, preConsumeCents)
		_ = AdjustTokenSpentTx(h.db.WithContext(ctx), authResult.Token.ID, -reserveQuotaCents)
		h.appendBalanceTxn(settleJob{
			ctx:             context.WithoutCancel(ctx),
			accountID:       authResult.Account.ID,
			tokenID:         authResult.Token.ID,
			requestID:       requestID,
			preConsumeCents: preConsumeCents,
		}, preConsumeCents, model.TransactionRefund, "退款：上游失败")
		return
	}

	h.finalizeAndSettle(ctx, authResult, resolveResult, requestID, preConsumeCents, reserveQuotaCents, usage, delivered, winner)
}

// finalizeAndSettle 计算实际费用并入队异步结算。
// 计费模型与用量记录均以实际生效候选（winner）为准，降级时可能与主候选不同。
func (h *RelayHandler) finalizeAndSettle(ctx context.Context, authResult *AuthResult, resolveResult *ResolveResult, requestID string,
	preConsumeCents, reserveQuotaCents int64, usage *usageData, delivered bool, winner *upstreamCandidate) {

	model := resolveResult.Model
	if winner != nil && winner.Model != nil {
		model = winner.Model
	}

	var actualPromptTokens, actualCompletionTokens int
	var actualCents int64
	estimated := false

	if usage != nil {
		actualPromptTokens = usage.PromptTokens
		actualCompletionTokens = usage.CompletionTokens
		actualCents = CalculateCostCents(usage.PromptTokens, usage.CompletionTokens, model, h.billingCfg.BasePriceCentsPer1K)
		logger.InfoCtx(ctx, "relay request complete",
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
		logger.WarnCtx(ctx, "relay usage fallback charged",
			logger.Int64("cost_cents", actualCents))
	}

	// 结算在请求结束后由 worker 异步执行，请求 ctx 已被服务端取消。
	// WithoutCancel 保留真实 recording span（父链路）与 request_id 等值，
	// 同时脱离取消信号避免事务中止；SpanContextWithContext 包装会退化为 noop tracer 导致子 span 丢失。
	jobCtx := context.WithoutCancel(ctx)

	job := settleJob{
		ctx:                    jobCtx,
		accountID:              authResult.Account.ID,
		tokenID:                authResult.Token.ID,
		modelName:              model.Name,
		providerID:             model.ProviderID,
		requestID:              requestID,
		preConsumeCents:        preConsumeCents,
		reserveQuotaCents:      reserveQuotaCents,
		actualCents:            actualCents,
		estimated:              estimated,
		actualPromptTokens:     actualPromptTokens,
		actualCompletionTokens: actualCompletionTokens,
	}

	select {
	case h.settleCh <- job:
	default:
		// 队列满：同步结算，避免账单丢失（请求已在客户端写入完毕，短暂阻塞可接受）
		logger.WarnCtx(ctx, "relay settle channel full, settling synchronously")
		h.processJob(job)
	}
}

func (h *RelayHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := extractBearerToken(r)
	if key == "" {
		writeOpenAIError(w, http.StatusUnauthorized, "authentication_error", "missing authorization header")
		return
	}

	authResult, err := h.authenticateCached(ctx, key)
	if err != nil {
		writeOpenAIError(w, http.StatusUnauthorized, "authentication_error", err.Error())
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
		if err := h.db.WithContext(ctx).Where("status = ?", true).Find(&models).Error; err != nil {
			writeOpenAIError(w, http.StatusInternalServerError, "api_error", err.Error())
			return
		}
		data := make([]modelItem, 0, len(models))
		for _, m := range models {
			if allowed[m.ID] {
				data = append(data, modelItem{ID: m.Name, Object: "model", Created: m.CreatedAt.Unix(), OwnedBy: "llm-gate"})
			}
		}
		httpx.OkJSONCtx(ctx, w, map[string]interface{}{"object": "list", "data": data})
		return
	}

	// 无白名单：走模型列表缓存
	const modelsKey = "models:all"
	var cachedItems []modelItem
	if ok, _ := h.modelListCache.GetInto(ctx, modelsKey, &cachedItems); ok {
		httpx.OkJSONCtx(ctx, w, map[string]interface{}{"object": "list", "data": cachedItems})
		return
	}

	logger.InfoCtx(ctx, "relay models list cache miss")
	var models []model.ModelConfig
	if err := h.db.WithContext(ctx).Where("status = ?", true).Find(&models).Error; err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "api_error", err.Error())
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
	h.modelListCache.Set(ctx, modelsKey, data, 30*time.Second)

	httpx.OkJSONCtx(ctx, w, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

func (h *RelayHandler) authenticateCached(ctx context.Context, key string) (*AuthResult, error) {
	cacheKey := "auth:" + key
	var cached AuthResult
	if ok, _ := h.authCache.GetInto(ctx, cacheKey, &cached); ok {
		return &cached, nil
	}

	keyHash := security.SHA256Hex(key)
	var token model.UserToken
	if err := h.db.WithContext(ctx).Where("key_hash = ? AND status = ?", keyHash, true).First(&token).Error; err != nil {
		return nil, fmt.Errorf("invalid or disabled token")
	}
	logger.InfoCtx(ctx, "relay auth cache miss", logger.Int64("token_id", token.ID))

	if token.ExpiredAt != nil && token.ExpiredAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("token expired")
	}

	// 预算：quota>0 表示该 Key 的累计消费上限，0 表示不限。
	// SpentCents 随每次结算变化，且认证结果被缓存，因此预算超限必须读实时值，避免缓存导致的超额放行。
	if token.Quota > 0 {
		if err := h.db.WithContext(ctx).Select("spent_cents").First(&token, token.ID).Error; err != nil {
			return nil, fmt.Errorf("token not found")
		}
		if token.SpentCents >= token.Quota {
			return nil, fmt.Errorf("key budget exhausted")
		}
	}

	var account model.Account
	if err := h.db.WithContext(ctx).First(&account, token.AccountID).Error; err != nil {
		return nil, fmt.Errorf("account not found")
	}
	if !account.Status {
		return nil, fmt.Errorf("account disabled")
	}
	// 认证结果被缓存，余额 <= 0 时读实时值，避免充值后仍被误拒（真实扣款由 DeductBalance 原子保障）
	if account.BalanceCents <= 0 {
		if err := h.db.WithContext(ctx).Select("balance_cents").First(&account, account.ID).Error; err != nil {
			return nil, fmt.Errorf("account not found")
		}
		if account.BalanceCents <= 0 {
			return nil, fmt.Errorf("insufficient balance")
		}
	}

	result := &AuthResult{Token: &token, Account: &account}
	h.authCache.Set(ctx, cacheKey, result, 10*time.Second)
	return result, nil
}

// resolveCached 解析模型并构建候选服务商列表。
// 应用 Token 模型白名单过滤，按权重随机选主候选，其余按权重降序作为降级候选。
func (h *RelayHandler) resolveCached(ctx context.Context, modelName string, token *model.UserToken) (*ResolveResult, error) {
	listKey := "model_list:" + modelName
	var modelList []model.ModelConfig
	if ok, _ := h.modelListCache.GetInto(ctx, listKey, &modelList); !ok {
		// 阴性缓存：不存在的模型名也缓存 30s，防止穿透到 DB
		var neg struct{}
		if ok, _ := h.modelListCache.GetInto(ctx, "neg:"+modelName, &neg); ok {
			return nil, fmt.Errorf("model not found or disabled")
		}
		logger.InfoCtx(ctx, "relay model cache miss", logger.String("model", modelName))
		if err := h.db.WithContext(ctx).Where("name = ? AND status = ?", modelName, true).
			Order("id ASC").
			Find(&modelList).Error; err != nil || len(modelList) == 0 {
			logger.WarnCtx(ctx, "relay model resolve failed", logger.String("model", modelName))
			h.modelListCache.Set(ctx, "neg:"+modelName, struct{}{}, 30*time.Second)
			return nil, fmt.Errorf("model not found or disabled")
		}
		h.modelListCache.Set(ctx, listKey, modelList, 30*time.Second)
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
	candidates, err := h.buildCandidates(ctx, modelList)
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
func (h *RelayHandler) buildCandidates(ctx context.Context, modelList []model.ModelConfig) ([]upstreamCandidate, error) {
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
		provider, err := h.loadProvider(ctx, m.ProviderID)
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
func (h *RelayHandler) loadProvider(ctx context.Context, providerID int64) (*model.Provider, error) {
	providerKey := "provider:" + strconv.FormatInt(providerID, 10)
	var provider model.Provider
	if v, cached := h.providerCache.Get(ctx, providerKey); cached {
		if p, ok := v.(*model.Provider); ok {
			provider = *p
			return &provider, nil
		}
	}
	logger.InfoCtx(ctx, "relay provider cache miss", logger.Int64("provider_id", providerID))
	if err := h.db.WithContext(ctx).Where("id = ? AND status = ?", providerID, true).First(&provider).Error; err != nil {
		return nil, fmt.Errorf("provider not found or disabled")
	}
	key, err := h.cipher.Decrypt(provider.APIKey)
	if err != nil {
		return nil, fmt.Errorf("provider key decrypt failed")
	}
	provider.APIKey = key
	h.providerCache.Set(ctx, providerKey, &provider, 60*time.Second)
	return &provider, nil
}

// requestID 优先沿用客户端 X-Request-ID，否则生成。
func (h *RelayHandler) requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return generateRequestID()
}

func (h *RelayHandler) notifyBalanceLow(ctx context.Context, account *model.Account) {
	h.notifier.Send(ctx, fmt.Sprintf("balance_low_%d", account.ID), "账户余额不足",
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

func extractFieldString(ctx context.Context, raw map[string]json.RawMessage, key string) string {
	v, ok := raw[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		logger.WarnCtx(ctx, "relay: extract field string failed",
			logger.String("key", key), logger.Err(err))
	}
	return s
}

func extractFieldBool(ctx context.Context, raw map[string]json.RawMessage, key string) bool {
	v, ok := raw[key]
	if !ok {
		return false
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		logger.WarnCtx(ctx, "relay: extract field bool failed",
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

func refundWithRetry(ctx context.Context, db *gorm.DB, accountID, cents int64) {
	for retries := 3; retries > 0; retries-- {
		if err := RefundBalance(ctx, db, accountID, cents); err == nil {
			return
		}
		time.Sleep(time.Duration(50*retries) * time.Millisecond)
	}
	logger.ErrorCtx(ctx, "relay: refund failed after retries",
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
