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

func (r *RedisCache) Get(key string) (any, bool) {
	data, err := r.client.Get(context.Background(), key)
	if err != nil {
		return nil, false
	}
	var v any
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return nil, false
	}
	return v, true
}

func (r *RedisCache) GetInto(key string, dest any) (bool, error) {
	data, err := r.client.Get(context.Background(), key)
	if err != nil {
		return false, nil
	}
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		logger.Error("redis cache: json unmarshal failed",
			logger.String("key", key), logger.Err(err))
		return false, err
	}
	return true, nil
}

func (r *RedisCache) Set(key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		logger.Error("redis cache: json marshal failed",
			logger.String("key", key), logger.Err(err))
		return
	}
	_ = r.client.Set(context.Background(), key, string(data), ttl)
}

func (r *RedisCache) Del(key string) {
	_, _ = r.client.Del(context.Background(), key)
}
