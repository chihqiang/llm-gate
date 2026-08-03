package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/trace"
	"go.opentelemetry.io/otel/codes"
)

const (
	kindChat       = "chat"
	kindEmbeddings = "embeddings"

	maxErrorRespBodySize = 1 << 20 // 1MB，限制错误响应体大小
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

type usageData struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatResponse struct {
	Usage *usageData `json:"usage"`
}

// upstreamResult 单次上游请求结果。
type upstreamResult struct {
	status   int
	header   http.Header
	body     []byte        // 非流式响应体（已缓冲）
	stream   io.ReadCloser // 流式响应体
	isStream bool
}

// upstreamError 携带错误响应信息的上游错误。
// status==0 表示网络层错误（可重试）；>=500 表示上游 5xx（可重试）；4xx 不可重试。
type upstreamError struct {
	status int
	header http.Header
	body   []byte
	msg    string
}

func (e *upstreamError) Error() string { return e.msg }

func (e *upstreamError) retryable() bool {
	return e.status == 0 || e.status >= 500
}

// committedError 表示响应头已写入客户端后的错误（流式中断）。
// 此时无法重试或切换候选，只能记录熔断统计与计费。
type committedError struct {
	err error
}

func (e *committedError) Error() string { return "response committed: " + e.err.Error() }

// forward 按候选顺序尝试上游，支持熔断跳过与自动降级。
// 仅在所有候选都失败或遇到不可重试错误时，才向客户端写入错误响应。
// 返回 usage（可能为 nil）、是否成功投递响应、错误，以及实际生效的候选（失败时可能为 nil）。
func (h *RelayHandler) forward(ctx context.Context, w http.ResponseWriter, r *http.Request, kind string,
	candidates []upstreamCandidate, raw map[string]json.RawMessage, isStream bool) (*usageData, bool, error, *upstreamCandidate) {

	var lastErr error
	for i := range candidates {
		cand := &candidates[i]
		breaker := h.breakers.get(cand.Provider.ID)
		if !breaker.Allow() {
			lastErr = &upstreamError{status: http.StatusServiceUnavailable,
				msg: fmt.Sprintf("provider %s circuit open", cand.Provider.Name)}
			logger.WarnCtx(ctx, "relay: circuit open, skip provider",
				logger.Int64("provider_id", cand.Provider.ID))
			h.notifier.Send(ctx, fmt.Sprintf("circuit_open_%d", cand.Provider.ID), "上游熔断打开",
				fmt.Sprintf("服务商 %s 连续失败被熔断，请求自动跳过", cand.Provider.Name))
			continue
		}

		reqBody := buildRequestBody(raw, cand.Model.UpstreamModelName, isStream)
		usage, delivered, err := h.attempt(ctx, w, r, kind, *cand, reqBody, isStream)
		if err != nil {
			breaker.Failure()
			logger.WarnCtx(ctx, "relay: upstream attempt failed",
				logger.Err(err), logger.Int64("provider_id", cand.Provider.ID))
			h.notifier.Send(ctx, fmt.Sprintf("provider_fail_%d", cand.Provider.ID), "上游故障",
				fmt.Sprintf("服务商 %s 请求失败：%v", cand.Provider.Name, err))
			lastErr = err
			// 已提交响应（流式中断）：不能再次写入错误响应，仅停止降级尝试。
			// 该候选已实际生效，返回给上层用于结算/用量记录。
			var committed *committedError
			if errors.As(err, &committed) {
				return usage, delivered, err, cand
			}
			var ue *upstreamError
			// 重试器会用 ErrMaxRetries 包装底层错误，需 errors.As 解包
			ok := errors.As(err, &ue)
			// 不可重试错误（4xx）或已无更多候选：写入最终错误
			if !ok || !ue.retryable() || i == len(candidates)-1 {
				writeFinalError(w, err)
				return nil, false, err, nil
			}
			continue
		}
		breaker.Success()
		logger.InfoCtx(ctx, "relay upstream ok",
			logger.String("provider", cand.Provider.Name))
		return usage, delivered, nil, cand
	}

	writeFinalError(w, lastErr)
	return nil, false, lastErr, nil
}

// attempt 执行单次上游请求（含非流式重试）。
// 流式请求使用独立于客户端连接的超时上下文：客户端断连后仍继续读取上游以获取 usage，确保计费准确。
func (h *RelayHandler) attempt(ctx context.Context, w http.ResponseWriter, r *http.Request, kind string,
	cand upstreamCandidate, body []byte, isStream bool) (*usageData, bool, error) {

	ctx, span := trace.StartSpan(ctx, "upstream "+endpointPath(kind),
		trace.WithAttributes(
			trace.AttrString("provider", cand.Provider.Name),
			trace.AttrInt64("provider_id", cand.Provider.ID),
			trace.AttrString("upstream_model", cand.Model.UpstreamModelName),
		),
	)
	defer span.End()

	// 流式：解耦客户端断连，但保留链路上下文；非流式：客户端取消则中止
	if isStream {
		detached := context.WithoutCancel(ctx)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(detached, time.Duration(h.relayCfg.Timeout)*time.Second)
		defer cancel()
	}

	run := func() (*upstreamResult, error) {
		return h.doForward(ctx, kind, cand, body)
	}

	var last *upstreamResult
	var err error
	if isStream {
		last, err = run()
	} else if h.relayCfg.Upstream.RetryEnabled {
		// 内联重试：retry 包会把底层错误用 %s 序列化，导致 errors.As 无法解包 upstreamError，
		// 进而阻断多服务商降级，故在此内联实现并保留原始错误类型。
		maxRetries := h.relayCfg.Upstream.MaxRetries
		if maxRetries < 0 {
			maxRetries = 0
		}
		delay := time.Duration(h.relayCfg.Upstream.RetryDelayMs) * time.Millisecond
		for attempt := 0; ; attempt++ {
			last, err = run()
			var ue *upstreamError
			if err == nil || !errors.As(err, &ue) || !ue.retryable() || attempt >= maxRetries {
				break
			}
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-time.After(delay + time.Duration(rand.Int64N(int64(delay)+1))):
			}
		}
	} else {
		last, err = run()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, false, err
	}

	writeHeaders(w, last.header, last.status)

	if last.isStream {
		usage, delivered, serr := proxyStreamWithUsage(w, ctx, last.stream)
		if serr != nil {
			// 响应头已写入，客户端可能已收到部分内容：标记为已提交错误，仅影响熔断统计与结算
			span.SetStatus(codes.Error, serr.Error())
			span.RecordError(serr)
			return usage, delivered, &committedError{err: serr}
		}
		return usage, delivered, nil
	}

	if _, werr := w.Write(last.body); werr != nil {
		logger.ErrorCtx(ctx, "relay: write response body failed", logger.Err(werr))
	}
	return extractUsage(last.body), true, nil
}

// doForward 执行一次上游 HTTP 请求并缓冲结果。
// 非 200 响应不写入客户端，而是封装为 upstreamError 返回（携带错误体）。
func (h *RelayHandler) doForward(ctx context.Context, kind string, cand upstreamCandidate, body []byte) (*upstreamResult, error) {
	url := strings.TrimRight(cand.Provider.BaseURL, "/") + endpointPath(kind)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &upstreamError{msg: fmt.Sprintf("create request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cand.Provider.APIKey)
	// 注入 W3C trace context，使上游（若接入）与当前请求共享同一链路
	trace.InjectHeader(ctx, req.Header)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, &upstreamError{msg: fmt.Sprintf("upstream request failed: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		// 非 200：读取错误体（有界）后关闭连接，不写入客户端
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorRespBodySize))
		resp.Body.Close()
		prefix := "status 5"
		if resp.StatusCode < 500 {
			prefix = "status 4"
		}
		return &upstreamResult{status: resp.StatusCode, header: resp.Header.Clone(), body: body},
			&upstreamError{status: resp.StatusCode, header: resp.Header.Clone(), body: body,
				msg: fmt.Sprintf("%s %d", prefix, resp.StatusCode)}
	}

	if isStreamingResponse(resp) {
		// 流式：返回流，由调用方消费（不在此关闭）
		return &upstreamResult{status: resp.StatusCode, header: resp.Header.Clone(), stream: resp.Body, isStream: true}, nil
	}

	// 非流式：缓冲完整响应（有界，防止上游返回超大内容）
	maxBody := int64(h.relayCfg.MaxBodyMB) << 20
	bodyBuf, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	resp.Body.Close()
	if readErr != nil {
		return nil, &upstreamError{msg: fmt.Sprintf("read upstream body: %v", readErr)}
	}
	if int64(len(bodyBuf)) > maxBody {
		return nil, &upstreamError{status: http.StatusBadGateway, msg: "upstream response too large"}
	}
	return &upstreamResult{status: resp.StatusCode, header: resp.Header.Clone(), body: bodyBuf}, nil
}

// proxyStreamWithUsage 实时转发 SSE 流，同时解析 usage。
// 客户端断开后停止写入但继续读取上游（排水），以捕获末尾的 usage 数据，确保计费准确。
// 返回 usage、是否成功投递了至少一行数据、上游读取错误（若流中断）。
func proxyStreamWithUsage(w http.ResponseWriter, ctx context.Context, upstreamBody io.ReadCloser) (*usageData, bool, error) {
	defer upstreamBody.Close()

	var usage *usageData
	delivered := false
	clientGone := false
	sawDone := false
	flusher, _ := w.(http.Flusher)

	scanner := bufio.NewScanner(upstreamBody)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		// 客户端未断开时实时转发原始行 + 换行符
		if !clientGone {
			if _, err := w.Write(line); err != nil {
				clientGone = true
			} else if _, err := w.Write([]byte("\n")); err != nil {
				clientGone = true
			} else {
				if flusher != nil {
					flusher.Flush()
				}
				delivered = true
			}
		}

		// 客户端断连后不再写，但继续解析 usage（排水计费）
		data := line
		if bytes.HasPrefix(data, []byte("data: ")) {
			data = data[6:]
		} else if bytes.HasPrefix(data, []byte("data:")) {
			data = data[5:]
		} else {
			continue
		}

		if bytes.Equal(bytes.TrimSpace(data), []byte("[DONE]")) {
			sawDone = true
			continue
		}
		if !bytes.Contains(data, []byte("usage")) {
			continue
		}

		var chunk chatResponse
		if json.Unmarshal(data, &chunk) == nil && chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	// 上游正常结束但未发 DONE 时补发，保证流式协议完整（客户端未断开且无错误）
	if scanner.Err() == nil && !sawDone && !clientGone && delivered {
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 上游读取失败（非 EOF）视为流中断，用于熔断统计
	if err := scanner.Err(); err != nil {
		logger.ErrorCtx(ctx, "relay stream: scanner error, usage may be incomplete", logger.Err(err))
		return usage, delivered, err
	}

	return usage, delivered, nil
}

func extractUsage(body []byte) *usageData {
	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return resp.Usage
}

// buildRequestBody 构造转发给上游的请求体：替换 model 为上游模型名，流式时注入 stream_options。
func buildRequestBody(raw map[string]json.RawMessage, upstreamModel string, isStream bool) []byte {
	reqBody := make(map[string]json.RawMessage, len(raw)+1)
	for k, v := range raw {
		reqBody[k] = v
	}
	reqBody["model"] = json.RawMessage(fmt.Sprintf(`"%s"`, upstreamModel))
	if isStream {
		reqBody["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	}
	b, _ := json.Marshal(reqBody)
	return b
}

func endpointPath(kind string) string {
	if kind == kindEmbeddings {
		return "/v1/embeddings"
	}
	return "/v1/chat/completions"
}

// writeHeaders 复制上游响应头并写入状态码。
// 剔除 hop-by-hop 头与 Content-Length：Content-Length 由 Go 在响应结束时自动计算/分块。
func writeHeaders(w http.ResponseWriter, header http.Header, status int) {
	for k, v := range header {
		if isHopByHopHeader(k) || k == "Content-Length" {
			continue
		}
		w.Header()[k] = v
	}
	w.WriteHeader(status)
}

var hopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Proxy-Connection":    true,
	"TE":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

func isHopByHopHeader(name string) bool {
	return hopByHopHeaders[name]
}

// writeFinalError 向客户端写入最终错误响应。
// 优先透传上游错误体；网络错误等回退为 502 JSON。
func writeFinalError(w http.ResponseWriter, err error) {
	if err == nil {
		err = errors.New("all upstreams unavailable")
	}
	var ue *upstreamError
	if errors.As(err, &ue) {
		status := http.StatusBadGateway
		if ue.status > 0 {
			status = ue.status
		}
		for k, v := range ue.header {
			w.Header()[k] = v
		}
		if len(ue.body) > 0 {
			w.WriteHeader(status)
			_, _ = w.Write(ue.body)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":{"message":"` + strings.ReplaceAll(err.Error(), `"`, `\"`) + `"}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	_, _ = w.Write([]byte(`{"error":{"message":"upstream unavailable"}}`))
}

func isStreamingResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/event-stream")
}

// writeOpenAIError 按 OpenAI 兼容格式写入错误，携带正确的 HTTP 状态码。
func writeOpenAIError(w http.ResponseWriter, status int, errType, message string) {
	body, _ := json.Marshal(map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    errType,
		},
	})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
