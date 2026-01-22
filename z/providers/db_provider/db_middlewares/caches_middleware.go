package db_middlewares

import (
	"sync"

	"github.com/go-gorm/caches"
	"gorm.io/gorm"
)

type CachesMiddleware struct {
	Easer bool
}

type inMemoryCacher struct {
	store *sync.Map
}

func (c *inMemoryCacher) init() {
	if c.store == nil {
		c.store = &sync.Map{}
	}
}

func (c *inMemoryCacher) Get(key string) interface{} {
	c.init()
	if v, ok := c.store.Load(key); ok {
		return v
	}
	return nil
}

func (c *inMemoryCacher) Store(key string, val interface{}) error {
	c.init()
	c.store.Store(key, val)
	return nil
}

func (m CachesMiddleware) Apply(db *gorm.DB) error {
	plugin := &caches.Caches{Conf: &caches.Config{
		Easer:  m.Easer,
		Cacher: &inMemoryCacher{},
	}}
	return db.Use(plugin)
}
