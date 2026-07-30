// Package cache 提供统一的缓存抽象层，支持内存（go-cache）和 Redis 两种后端。
// 多节点部署时使用 Redis 保证缓存一致性；单节点时使用内存缓存，零外部依赖。
package cache

import "time"

// Cache 缓存接口，屏蔽底层实现。
type Cache interface {
	// Get 获取缓存值，key 不存在时 ok=false。
	// 注意：返回值是 interface{}，对于 JSON 后端（Redis）需要配合 GetInto 使用
	// 或由调用方自行处理类型断言。
	Get(key string) (value any, ok bool)
	// GetInto 将缓存值反序列化到 dest 中。dest 必须为指针类型。
	// 比 Get 更安全：自动处理 JSON 到具体类型的反序列化。
	GetInto(key string, dest any) (ok bool, err error)
	// Set 写入缓存，ttl 指定过期时间。
	Set(key string, value any, ttl time.Duration)
	// Del 删除缓存键。
	Del(key string)
}
