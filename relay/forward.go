package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"chihqiang/llm-gate/model"
)

type ForwardResult struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	IsStream   bool
}

type usageData struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatResponse struct {
	Usage *usageData `json:"usage"`
}

func Forward(w http.ResponseWriter, r *http.Request, provider *model.Provider, mc *model.ModelConfig, body []byte) {
	upstreamURL := strings.TrimRight(provider.BaseURL, "/") + "/v1/chat/completions"

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		raw["model"] = json.RawMessage(fmt.Sprintf(`"%s"`, mc.UpstreamModelName))
		body, _ = json.Marshal(raw)
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	if accept := r.Header.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	client := &http.Client{Timeout: 120 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	isStream := isStreamingResponse(resp)

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	if isStream {
		proxyStream(w, resp.Body)
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		w.Write(bodyBytes)
	}
}

func proxyStream(w http.ResponseWriter, upstreamBody io.ReadCloser) {
	buf := make([]byte, 4096)
	for {
		n, err := upstreamBody.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

func extractUsage(body []byte) *usageData {
	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return resp.Usage
}

func CalculateQuota(promptTokens, completionTokens int, mc *model.ModelConfig) int64 {
	totalTokens := promptTokens + completionTokens
	quota := float64(totalTokens) * mc.ModelRatio
	return int64(quota)
}

func isStreamingResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/event-stream")
}
