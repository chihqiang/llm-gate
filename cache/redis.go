package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/redisx"
)

// RedisCache 基于 Redis 的缓存实现，用于多节点部署。
// Get/Set 使用 JSON 编码，避免 gob 的 schema 耦合问题。
type RedisCache struct {
	client *redisx.Client
}

func NewRedis(client *redisx.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, bool) {
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, false
	}
	var v any
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return nil, false
	}
	return v, true
}

func (r *RedisCache) GetInto(ctx context.Context, key string, dest any) (bool, error) {
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return false, nil
	}
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		logger.ErrorCtx(ctx, "redis cache: json unmarshal failed",
			logger.String("key", key), logger.Err(err))
		return false, err
	}
	return true, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		logger.ErrorCtx(ctx, "redis cache: json marshal failed",
			logger.String("key", key), logger.Err(err))
		return
	}
	_ = r.client.Set(ctx, key, string(data), ttl)
}

func (r *RedisCache) Del(ctx context.Context, key string) {
	_, _ = r.client.Del(ctx, key)
}

// FlushByPrefix 使用 SCAN 游标删除所有以 prefix 开头的键。
func (r *RedisCache) FlushByPrefix(ctx context.Context, prefix string) {
	cursor := uint64(0)
	for {
		keys, next, err := r.client.Scan(ctx, cursor, prefix+"*", 100)
		if err != nil {
			logger.ErrorCtx(ctx, "redis cache: scan failed", logger.String("prefix", prefix), logger.Err(err))
			return
		}
		if len(keys) > 0 {
			_, _ = r.client.Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
