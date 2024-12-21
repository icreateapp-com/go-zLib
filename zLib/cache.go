package zLib

import (
	"github.com/patrickmn/go-cache"
	"time"
)

// _cache 结构体，用于存储缓存对象和默认有效期
type _cache struct {
	cache             *cache.Cache
	DefaultExpiration time.Duration
}

// Cache 全局缓存对象
var Cache _cache

// Init 初始化缓存（默认有效期，清理缓存间隔时间）
func (p *_cache) Init(defaultExpiration, cleanupInterval time.Duration) {
	p.DefaultExpiration = defaultExpiration
	p.cache = cache.New(defaultExpiration, cleanupInterval)

}

// Set 创建缓存
func (p *_cache) Set(k string, x interface{}, d time.Duration) {
	p.cache.Set(k, x, d)
}

// Get 获取缓存
func (p *_cache) Get(k string) (interface{}, bool) {
	return p.cache.Get(k)
}

// Delete 删除缓存
func (p *_cache) Delete(k string) {
	p.cache.Delete(k)
}
