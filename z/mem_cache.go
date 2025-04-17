package z

import (
	"github.com/patrickmn/go-cache"
	"time"
)

type _memCache struct {
	cache             *cache.Cache
	DefaultExpiration time.Duration
	initialized       bool
}

// MemCache 内存缓存
var MemCache _memCache

// Init 初始化缓存（默认有效期，清理缓存间隔时间）
func (p *_memCache) Init(defaultExpiration, cleanupInterval time.Duration) {
	if p.initialized {
		return
	}
	p.DefaultExpiration = defaultExpiration
	p.cache = cache.New(defaultExpiration, cleanupInterval)
	p.initialized = true
}

// Set 创建缓存
func (p *_memCache) Set(k string, x interface{}, d time.Duration) {
	p.cache.Set(k, x, d)
}

// Get 获取缓存
func (p *_memCache) Get(k string) (interface{}, bool) {
	return p.cache.Get(k)
}

// Delete 删除缓存
func (p *_memCache) Delete(k string) {
	p.cache.Delete(k)
}
