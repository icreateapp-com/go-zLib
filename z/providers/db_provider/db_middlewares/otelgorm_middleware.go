package db_middlewares

import (
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/trace_provider"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/gorm"
)

type OtelGormMiddleware struct {
	TP  *trace_provider.Trace
	Log *logger_provider.Logger
}

func (m OtelGormMiddleware) Apply(db *gorm.DB) error {
	if m.TP != nil && m.TP.TracerProvider != nil {
		if err := db.Use(otelgorm.NewPlugin(otelgorm.WithTracerProvider(m.TP.TracerProvider))); err != nil {
			if m.Log != nil {
				m.Log.Errorw("use otelgorm error", "error", err)
			}
			return err
		}
		return nil
	}
	if m.Log != nil {
		m.Log.Warnw("trace provider is nil, skip otelgorm plugin")
	}
	return nil
}
