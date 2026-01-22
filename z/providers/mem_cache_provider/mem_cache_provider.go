package mem_cache_provider

import (
	"time"

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"

	"github.com/patrickmn/go-cache"
	"go.uber.org/fx"
)

// MemCache 内存缓存
type MemCache struct {
	cache *cache.Cache
}

// NewMemCacheProvider 创建内存缓存实例
func NewMemCacheProvider(cfg *config_provider.Config) *MemCache {
	defaultExpiration := cfg.GetDuration("mem_cache.default_expiration", 60*time.Minute)
	cleanupInterval := cfg.GetDuration("mem_cache.cleanup_interval", 10*time.Minute)

	return &MemCache{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// MemCacheProviderModule 内存缓存模块
var MemCacheProviderModule = fx.Options(
	fx.Provide(NewMemCacheProvider),
)

// Set 设置缓存
func (p *MemCache) Set(k string, x interface{}, d time.Duration) {
	p.cache.Set(k, x, d)
}

// Get 获取缓存
func (p *MemCache) Get(k string) (interface{}, bool) {
	return p.cache.Get(k)
}

// Delete 删除缓存
func (p *MemCache) Delete(k string) {
	p.cache.Delete(k)
}
