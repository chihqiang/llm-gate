package relay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"chihqiang/llm-gate/config"

	"github.com/chihqiang/infra-go/logger"
)

// notifier 通过 webhook 发送告警，同类事件带冷却期防轰炸。
type notifier struct {
	cfg      config.AlertConfig
	client   *http.Client
	mu       sync.Mutex
	lastSent map[string]time.Time
}

func newNotifier(cfg config.AlertConfig) *notifier {
	if !cfg.Enabled {
		return nil
	}
	return &notifier{
		cfg:      cfg,
		client:   &http.Client{Timeout: 5 * time.Second},
		lastSent: make(map[string]time.Time),
	}
}

// Send 发送告警。key 用于冷却期去重（同类事件只发一次）。
func (n *notifier) Send(ctx context.Context, key, title, message string) {
	if n == nil || !n.cfg.Enabled || n.cfg.WebhookURL == "" {
		return
	}

	n.mu.Lock()
	cooldown := n.cfg.Cooldown
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	if last, ok := n.lastSent[key]; ok && time.Since(last) < cooldown {
		n.mu.Unlock()
		return
	}
	n.lastSent[key] = time.Now()
	n.mu.Unlock()

	payload := map[string]string{
		"title":   title,
		"message": message,
		"time":    time.Now().Format(time.RFC3339),
		"source":  "llm-gate",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		logger.ErrorCtx(ctx, "alert: create request failed", logger.Err(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// 告警体签名：X-LLM-Gate-Signature: sha256=hex(hmac_sha256(secret, body))
	if n.cfg.SignSecret != "" {
		mac := hmac.New(sha256.New, []byte(n.cfg.SignSecret))
		mac.Write(body)
		req.Header.Set("X-LLM-Gate-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	if resp, err := n.client.Do(req); err != nil {
		logger.ErrorCtx(ctx, "alert: webhook send failed", logger.Err(err))
	} else {
		resp.Body.Close()
	}
}
