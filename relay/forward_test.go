package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/model"
)

func newForwardHandler() *RelayHandler {
	return &RelayHandler{
		relayCfg: config.RelayConfig{
			Timeout:   5,
			MaxBodyMB: 8,
			Upstream: config.UpstreamConfig{
				RetryEnabled: true,
				MaxRetries:   1,
				RetryDelayMs: 0,
			},
		},
		client:   &http.Client{},
		breakers: newBreakerManager(2, time.Minute, time.Minute),
	}
}

func candidate(baseURL string, providerID int64) upstreamCandidate {
	return upstreamCandidate{
		Model:    &model.ModelConfig{UpstreamModelName: "test-model"},
		Provider: &model.Provider{ID: providerID, Name: fmt.Sprintf("p%d", providerID), BaseURL: baseURL, APIKey: "sk-test"},
	}
}

func okUpstream(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"ok","usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
}

func runForward(t *testing.T, h *RelayHandler, cands []upstreamCandidate) (*usageData, bool, error, *upstreamCandidate) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{
		"model":    "alias",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	body := map[string]json.RawMessage{}
	_ = json.Unmarshal(raw, &body)
	return h.forward(context.Background(), rec, req, kindChat, cands, body, false)
}

func TestForwardFailover(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	live := okUpstream(t)
	defer dead.Close()
	defer live.Close()

	h := newForwardHandler()
	usage, delivered, err, winner := runForward(t, h, []upstreamCandidate{
		candidate(dead.URL, 1),
		candidate(live.URL, 2),
	})

	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if !delivered {
		t.Fatalf("expected delivered=true")
	}
	if winner == nil || winner.Provider.ID != 2 {
		t.Fatalf("expected winner provider 2, got %+v", winner)
	}
	if usage == nil || usage.TotalTokens != 15 {
		t.Fatalf("expected usage total 15, got %+v", usage)
	}
}

func TestForwardNonRetryableStopsFailover(t *testing.T) {
	var liveHits int32
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"bad request"}`)
	}))
	live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&liveHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer bad.Close()
	defer live.Close()

	h := newForwardHandler()
	_, delivered, err, _ := runForward(t, h, []upstreamCandidate{
		candidate(bad.URL, 1),
		candidate(live.URL, 2),
	})

	if delivered {
		t.Fatalf("expected not delivered for 4xx")
	}
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := atomic.LoadInt32(&liveHits); got != 0 {
		t.Fatalf("4xx must not failover to next candidate, live server hit %d times", got)
	}
}

func TestForwardInlineRetry(t *testing.T) {
	var hits int32
	flaky := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"ok","usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	}))
	defer flaky.Close()

	h := newForwardHandler()
	usage, delivered, err, winner := runForward(t, h, []upstreamCandidate{candidate(flaky.URL, 1)})

	if err != nil {
		t.Fatalf("forward after retry: %v", err)
	}
	if !delivered {
		t.Fatalf("expected delivered")
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
	if winner == nil || winner.Provider.ID != 1 {
		t.Fatalf("expected winner provider 1, got %+v", winner)
	}
	if usage == nil || usage.TotalTokens != 3 {
		t.Fatalf("expected usage total 3, got %+v", usage)
	}
}

func TestForwardCircuitOpenSkip(t *testing.T) {
	open := newCircuitBreaker(2, time.Minute, time.Minute)
	open.state = stateOpen
	open.openedAt = time.Now()

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	live := okUpstream(t)
	defer dead.Close()
	defer live.Close()

	h := newForwardHandler()
	h.breakers.mu.Lock()
	h.breakers.breakers[1] = open
	h.breakers.mu.Unlock()

	usage, delivered, err, winner := runForward(t, h, []upstreamCandidate{
		candidate(dead.URL, 1),
		candidate(live.URL, 2),
	})

	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if !delivered {
		t.Fatalf("expected delivered")
	}
	if winner == nil || winner.Provider.ID != 2 {
		t.Fatalf("circuit open provider must be skipped, winner = %+v", winner)
	}
	if usage == nil || usage.TotalTokens != 15 {
		t.Fatalf("expected usage total 15, got %+v", usage)
	}
}

func TestForwardAllFail(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer dead.Close()

	h := newForwardHandler()
	_, delivered, err, winner := runForward(t, h, []upstreamCandidate{
		candidate(dead.URL, 1),
		candidate(dead.URL, 2),
	})

	if delivered {
		t.Fatalf("expected not delivered when all candidates fail")
	}
	if err == nil {
		t.Fatalf("expected error")
	}
	if winner != nil {
		t.Fatalf("expected nil winner, got %+v", winner)
	}
}
