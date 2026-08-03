package relay

import (
	"sync"
	"time"
)

// breakerState 熔断器状态
type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

// circuitBreaker 滑动窗口熔断器。
// 连续失败达到阈值后进入 Open 状态，Open 期间请求直接拒绝；
// 冷却结束后进入 HalfOpen，放行探针请求验证上游恢复，成功则关闭，失败则重新打开。
type circuitBreaker struct {
	mu sync.Mutex

	threshold int
	window    time.Duration
	cooldown  time.Duration

	state      breakerState
	failures   []time.Time // 窗口内的失败时间
	openedAt   time.Time
	allowProbe bool
}

func newCircuitBreaker(threshold int, window, cooldown time.Duration) *circuitBreaker {
	return &circuitBreaker{
		threshold: threshold,
		window:    window,
		cooldown:  cooldown,
	}
}

// Allow 判断是否允许请求通过。
func (b *circuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	switch b.state {
	case stateClosed:
		return true
	case stateOpen:
		if now.Sub(b.openedAt) >= b.cooldown {
			b.state = stateHalfOpen
			b.allowProbe = true
			return true
		}
		return false
	default: // halfOpen
		if b.allowProbe {
			b.allowProbe = false
			return true
		}
		return false
	}
}

// Success 记录成功，重置失败计数。
func (b *circuitBreaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == stateHalfOpen {
		b.state = stateClosed
	}
	b.failures = b.failures[:0]
}

// Failure 记录一次失败，超过阈值则打开熔断器。
func (b *circuitBreaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == stateHalfOpen {
		b.state = stateOpen
		b.openedAt = time.Now()
		b.failures = b.failures[:0]
		return
	}

	now := time.Now()
	// 清理窗口外的失败记录
	cutoff := now.Add(-b.window)
	kept := b.failures[:0]
	for _, t := range b.failures {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	b.failures = kept

	if len(b.failures) >= b.threshold && b.threshold > 0 {
		b.state = stateOpen
		b.openedAt = now
		b.failures = b.failures[:0]
	}
}

// IsOpen 是否处于熔断打开状态（用于告警）。
func (b *circuitBreaker) IsOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state == stateOpen
}

// breakerManager 管理所有服务商的熔断器。
type breakerManager struct {
	mu       sync.Mutex
	breakers map[int64]*circuitBreaker
	cfg      struct {
		threshold int
		window    time.Duration
		cooldown  time.Duration
	}
}

func newBreakerManager(threshold int, window, cooldown time.Duration) *breakerManager {
	return &breakerManager{
		breakers: make(map[int64]*circuitBreaker),
		cfg: struct {
			threshold int
			window    time.Duration
			cooldown  time.Duration
		}{threshold: threshold, window: window, cooldown: cooldown},
	}
}

func (m *breakerManager) get(providerID int64) *circuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.breakers[providerID]
	if !ok {
		b = newCircuitBreaker(m.cfg.threshold, m.cfg.window, m.cfg.cooldown)
		m.breakers[providerID] = b
	}
	return b
}

func (m *breakerManager) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.breakers = make(map[int64]*circuitBreaker)
}
