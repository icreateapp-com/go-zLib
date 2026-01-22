package db_middlewares

import (
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/trace_provider"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type OtelGormIn struct {
	fx.In
	TP  *trace_provider.Trace `optional:"true"`
	Log *logger_provider.Logger
}

type OtelGormNamedOut struct {
	fx.Out

	Item NamedMiddleware `group:"db_named_middlewares"`
}

func NewOtelGormNamed(in OtelGormIn) OtelGormNamedOut {
	m := OtelGormMiddleware{TP: in.TP, Log: in.Log}
	return OtelGormNamedOut{
		Item: NamedMiddleware{
			Name: "otelgorm",
			New: func() Middleware {
				return func(db *gorm.DB) error { return m.Apply(db) }
			},
		},
	}
}

var OtelGormModule = fx.Options(
	fx.Provide(
		NewOtelGormNamed,
	),
)
