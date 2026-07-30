package cache

import (
	"encoding/json"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// MemoryCache 基于 go-cache 的内存缓存实现，用于单节点部署。
type MemoryCache struct {
	c *gocache.Cache
}

// NewMemory 创建内存缓存。defaultTTL 和 cleanupInterval 仅用于 go-cache
// 内部清理，实际 Set 时 ttl 由调用方指定。
func NewMemory() *MemoryCache {
	return &MemoryCache{
		c: gocache.New(5*time.Minute, 1*time.Minute),
	}
}

func (m *MemoryCache) Get(key string) (any, bool) {
	return m.c.Get(key)
}

func (m *MemoryCache) GetInto(key string, dest any) (bool, error) {
	v, ok := m.c.Get(key)
	if !ok {
		return false, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (m *MemoryCache) Set(key string, value any, ttl time.Duration) {
	m.c.Set(key, value, ttl)
}

func (m *MemoryCache) Del(key string) {
	m.c.Delete(key)
}
