package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/retry"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func getBuf() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func putBuf(b *bytes.Buffer) {
	b.Reset()
	bufPool.Put(b)
}

const maxErrorRespBodySize = 1 << 20 // 1MB，限制上游错误响应体大小

type usageData struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatResponse struct {
	Usage *usageData `json:"usage"`
}

// forward 将请求转发到上游服务商，返回 usage 数据和可能的错误。
// 对于流式请求，会注入 stream_options 以获取 usage，同时实时转发 SSE 流。
// 对于非流式请求，读取完整响应后提取 usage 并转发。
// 无论上游返回成功或错误状态码，响应都会被写入 w。
// raw 为已解析的请求体，避免重复反序列化。
func (h *RelayHandler) forward(w http.ResponseWriter, r *http.Request, provider *model.Provider, mc *model.ModelConfig, raw map[string]json.RawMessage, isStream bool) (*usageData, error) {
	start := time.Now()
	upstreamURL := strings.TrimRight(provider.BaseURL, "/") + "/v1/chat/completions"

	reqBody := make(map[string]json.RawMessage, len(raw)+1)
	for k, v := range raw {
		reqBody[k] = v
	}
	reqBody["model"] = json.RawMessage(fmt.Sprintf(`"%s"`, mc.UpstreamModelName))
	if isStream {
		reqBody["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 流式请求不重试（SSE 无法安全重放）
	if isStream {
		return h.doForward(w, r, provider, upstreamURL, body, start)
	}

	// 非流式：配置了重试则包装重试逻辑
	if h.config.Upstream.RetryEnabled {
		var usage *usageData
		err := retry.DoWithConfig(r.Context(), func(ctx context.Context) error {
			var uErr error
			usage, uErr = h.doForward(w, r, provider, upstreamURL, body, start)
			return uErr
		},
			retry.WithMaxRetries(h.config.Upstream.MaxRetries),
			retry.WithDelay(time.Duration(h.config.Upstream.RetryDelayMs)*time.Millisecond),
			retry.WithJitter(),
			retry.WithRetryIf(func(err error) bool {
				return err != nil && !strings.Contains(err.Error(), "status 4")
			}),
		)
		if err != nil {
			return nil, err
		}
		return usage, nil
	}

	return h.doForward(w, r, provider, upstreamURL, body, start)
}

// doForward 执行单次上游请求
func (h *RelayHandler) doForward(w http.ResponseWriter, r *http.Request, provider *model.Provider, upstreamURL string, body []byte, start time.Time) (*usageData, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	if accept := r.Header.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		io.Copy(w, io.LimitReader(resp.Body, maxErrorRespBodySize))
		prefix := "status 5"
		if resp.StatusCode < 500 {
			prefix = "status 4"
		}
		return nil, fmt.Errorf("%s %d", prefix, resp.StatusCode)
	}

	if isStreamingResponse(resp) {
		return proxyStreamWithUsage(w, r.Context(), resp.Body), nil
	}

	buf := getBuf()
	if _, err := io.Copy(io.MultiWriter(w, buf), resp.Body); err != nil {
		logger.Error("relay: copy response body failed", logger.Err(err))
	}
	usage := extractUsage(buf.Bytes())
	putBuf(buf)

	logger.Info("relay upstream ok",
		logger.String("url", upstreamURL),
		logger.Duration("elapsed", time.Since(start)),
		logger.Int("status", resp.StatusCode))
	return usage, nil
}

// proxyStreamWithUsage 实时转发 SSE 流到客户端，同时解析流中的 usage 数据。
// OpenAI 流式响应在 stream_options.include_usage=true 时，
// 最后一个 chunk（choices 为空）会包含 usage 字段。
// 通过 ctx 检测客户端是否已断开连接，避免在上游已断开后继续无效写入。
func proxyStreamWithUsage(w http.ResponseWriter, ctx context.Context, upstreamBody io.ReadCloser) *usageData {
	var usage *usageData
	flusher, _ := w.(http.Flusher)

	scanner := bufio.NewScanner(upstreamBody)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		// 客户端已断开，停止转发
		if ctx.Err() != nil {
			break
		}

		line := scanner.Bytes()

		// 转发原始行 + 换行符给客户端
		if _, err := w.Write(line); err != nil {
			break
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			break
		}
		if flusher != nil {
			flusher.Flush()
		}

		// 解析 data 行，提取 usage
		// 用 bytes.Contains 预过滤：只有包含 "usage" 的行才做 JSON 解析
		// SSE 流中仅最后一个 chunk 含 usage，避免 N-1 次无意义的 JSON 解析
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

	return usage
}

func extractUsage(body []byte) *usageData {
	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return resp.Usage
}

// CalculateQuota 根据模型倍率计算额度消耗
// prompt tokens 按 ModelRatio 计费，completion tokens 按 ModelRatio * CompletionRatio 计费
func CalculateQuota(promptTokens, completionTokens int, mc *model.ModelConfig) int64 {
	promptCost := float64(promptTokens) * mc.ModelRatio
	completionCost := float64(completionTokens) * mc.ModelRatio * mc.CompletionRatio
	return int64(promptCost + completionCost)
}

func isStreamingResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/event-stream")
}
