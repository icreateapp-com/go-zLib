package db_middlewares

import (
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type CachesIn struct {
	fx.In
	Cfg *config_provider.Config
}

type CachesNamedOut struct {
	fx.Out
	Item NamedMiddleware `group:"db_named_middlewares"`
}

func NewCachesNamed(in CachesIn) CachesNamedOut {
	easer := in.Cfg.GetBool("db.caches.easer", true)
	return CachesNamedOut{
		Item: NamedMiddleware{
			Name: "caches",
			New: func() Middleware {
				m := CachesMiddleware{Easer: easer}
				return func(db *gorm.DB) error { return m.Apply(db) }
			},
		},
	}
}

var CachesModule = fx.Options(
	fx.Provide(
		NewCachesNamed,
	),
)
