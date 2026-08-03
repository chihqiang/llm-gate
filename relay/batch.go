package relay

import (
	"context"
	"strings"
	"sync"
	"time"

	"chihqiang/llm-gate/model"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/trace"
	"gorm.io/gorm"
)

const (
	maxBatchSize  = 32
	maxRetryLogs  = 512 // 最多保留 512 条待重试日志，防止 DB 故障时内存无限增长
	flushInterval = 3 * time.Second
)

type usageBatch struct {
	mu       sync.Mutex
	buf      []model.UsageLog
	retryBuf []model.UsageLog
	db       *gorm.DB
	flushC   chan struct{}
	stopCh   chan struct{}
}

func newUsageBatch(db *gorm.DB) *usageBatch {
	b := &usageBatch{
		buf:      make([]model.UsageLog, 0, maxBatchSize),
		retryBuf: make([]model.UsageLog, 0, maxBatchSize),
		db:       db,
		flushC:   make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
	}
	go b.loop(flushInterval)
	return b
}

func (b *usageBatch) Append(log model.UsageLog) {
	b.mu.Lock()
	if len(b.retryBuf) > 0 {
		b.retryBuf = append(b.retryBuf, log)
		if len(b.retryBuf) >= maxRetryLogs {
			drop := len(b.retryBuf) - maxRetryLogs
			logger.Warn("usage batch retry buf full, dropping oldest",
				logger.Int("count", drop))
			b.retryBuf = b.retryBuf[drop:]
		}
	} else {
		b.buf = append(b.buf, log)
		if len(b.buf) >= maxBatchSize {
			select {
			case b.flushC <- struct{}{}:
			default:
			}
		}
	}
	b.mu.Unlock()
}

func (b *usageBatch) flush() {
	b.mu.Lock()
	batch := b.retryBuf
	b.retryBuf = nil
	if len(batch) == 0 {
		batch = b.buf
		b.buf = make([]model.UsageLog, 0, maxBatchSize)
	}
	b.mu.Unlock()

	if len(batch) == 0 {
		return
	}

	// 后台批量写入没有请求 ctx，从 Background 起根 span，便于在 trace 中定位异步落库问题
	ctx, span := trace.StartSpan(context.Background(), "usage batch flush",
		trace.WithAttributes(
			trace.AttrInt("count", len(batch)),
		),
	)
	defer span.End()

	if err := b.db.CreateInBatches(batch, maxBatchSize).Error; err != nil {
		logger.ErrorCtx(ctx, "flush usage logs failed, queuing for retry", logger.Err(err),
			logger.Int("count", len(batch)),
			logger.String("request_ids", requestIDsOf(batch)))
		b.mu.Lock()
		b.retryBuf = append(b.retryBuf, batch...)
		if len(b.retryBuf) > maxRetryLogs {
			drop := len(b.retryBuf) - maxRetryLogs
			b.retryBuf = b.retryBuf[drop:]
		}
		b.mu.Unlock()
		return
	}
	logger.InfoCtx(ctx, "usage batch flush ok", logger.Int("count", len(batch)))
}

// requestIDsOf 汇总一批用量日志的 request_id，用于失败日志关联具体请求。
func requestIDsOf(batch []model.UsageLog) string {
	seen := make(map[string]struct{}, len(batch))
	var ids []string
	for _, l := range batch {
		if l.RequestID == "" {
			continue
		}
		if _, ok := seen[l.RequestID]; ok {
			continue
		}
		seen[l.RequestID] = struct{}{}
		ids = append(ids, l.RequestID)
	}
	if len(ids) > 8 {
		ids = ids[:8]
	}
	return strings.Join(ids, ",")
}

func (b *usageBatch) loop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-b.flushC:
			b.flush()
		case <-b.stopCh:
			b.flush()
			return
		}
	}
}

func (b *usageBatch) Stop() {
	close(b.stopCh)
}
