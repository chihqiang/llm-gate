package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/retry"
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

// forward 按候选顺序尝试上游，支持熔断跳过与自动降级。
// 仅在所有候选都失败或遇到不可重试错误时，才向客户端写入错误响应。
// 返回 usage（可能为 nil）、是否成功投递响应、错误。
func (h *RelayHandler) forward(w http.ResponseWriter, r *http.Request, kind string,
	candidates []upstreamCandidate, raw map[string]json.RawMessage, isStream bool, requestID string) (*usageData, bool, error) {

	var lastErr error
	for i, cand := range candidates {
		breaker := h.breakers.get(cand.Provider.ID)
		if !breaker.Allow() {
			lastErr = &upstreamError{status: http.StatusServiceUnavailable,
				msg: fmt.Sprintf("provider %s circuit open", cand.Provider.Name)}
			logger.Warn("relay: circuit open, skip provider",
				logger.Int64("provider_id", cand.Provider.ID), logger.String("request_id", requestID))
			h.notifier.Send(fmt.Sprintf("circuit_open_%d", cand.Provider.ID), "上游熔断打开",
				fmt.Sprintf("服务商 %s 连续失败被熔断，请求自动跳过", cand.Provider.Name))
			continue
		}

		reqBody := buildRequestBody(raw, cand.Model.UpstreamModelName, isStream)
		usage, delivered, err := h.attempt(w, r, kind, cand, reqBody, isStream)
		if err != nil {
			breaker.Failure()
			logger.Warn("relay: upstream attempt failed",
				logger.Err(err), logger.Int64("provider_id", cand.Provider.ID), logger.String("request_id", requestID))
			h.notifier.Send(fmt.Sprintf("provider_fail_%d", cand.Provider.ID), "上游故障",
				fmt.Sprintf("服务商 %s 请求失败：%v", cand.Provider.Name, err))
			lastErr = err
			ue, ok := err.(*upstreamError)
			// 不可重试错误（4xx）或已无更多候选：写入最终错误
			if !ok || !ue.retryable() || i == len(candidates)-1 {
				writeFinalError(w, err)
				return nil, false, err
			}
			continue
		}
		breaker.Success()
		logger.Info("relay upstream ok",
			logger.String("provider", cand.Provider.Name),
			logger.String("request_id", requestID))
		return usage, delivered, nil
	}

	writeFinalError(w, lastErr)
	return nil, false, lastErr
}

// attempt 执行单次上游请求（含非流式重试）。
// 流式请求使用独立于客户端连接的超时上下文：客户端断连后仍继续读取上游以获取 usage，确保计费准确。
func (h *RelayHandler) attempt(w http.ResponseWriter, r *http.Request, kind string,
	cand upstreamCandidate, body []byte, isStream bool) (*usageData, bool, error) {

	// 流式：解耦客户端断连；非流式：客户端取消则中止
	var ctx context.Context
	var cancel context.CancelFunc
	if isStream {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(h.relayCfg.Timeout)*time.Second)
		defer cancel()
	} else {
		ctx = r.Context()
	}

	run := func() (*upstreamResult, error) {
		return h.doForward(ctx, kind, cand, body)
	}

	var last *upstreamResult
	var err error
	if isStream {
		last, err = run()
	} else if h.relayCfg.Upstream.RetryEnabled {
		err = retry.DoWithConfig(ctx, func(ctx context.Context) error {
			var uErr error
			last, uErr = h.doForward(ctx, kind, cand, body)
			return uErr
		},
			retry.WithMaxRetries(h.relayCfg.Upstream.MaxRetries),
			retry.WithDelay(time.Duration(h.relayCfg.Upstream.RetryDelayMs)*time.Millisecond),
			retry.WithJitter(),
			retry.WithRetryIf(func(err error) bool {
				ue, ok := err.(*upstreamError)
				return ok && ue.retryable()
			}),
		)
	} else {
		last, err = run()
	}
	if err != nil {
		return nil, false, err
	}

	writeHeaders(w, last.header, last.status)

	if last.isStream {
		usage, delivered := proxyStreamWithUsage(w, r.Context(), last.stream)
		return usage, delivered, nil
	}

	if _, werr := w.Write(last.body); werr != nil {
		logger.Error("relay: write response body failed", logger.Err(werr))
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
// 返回 usage 和是否成功投递了至少一行数据。
func proxyStreamWithUsage(w http.ResponseWriter, ctx context.Context, upstreamBody io.ReadCloser) (*usageData, bool) {
	defer upstreamBody.Close()

	var usage *usageData
	delivered := false
	clientGone := false
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

	if err := scanner.Err(); err != nil {
		logger.Error("relay stream: scanner error, usage may be incomplete", logger.Err(err))
	}

	return usage, delivered
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
func writeHeaders(w http.ResponseWriter, header http.Header, status int) {
	for k, v := range header {
		w.Header()[k] = v
	}
	w.WriteHeader(status)
}

// writeFinalError 向客户端写入最终错误响应。
// 优先透传上游错误体；网络错误等回退为 502 JSON。
func writeFinalError(w http.ResponseWriter, err error) {
	if err == nil {
		err = errors.New("all upstreams unavailable")
	}
	if ue, ok := err.(*upstreamError); ok {
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
