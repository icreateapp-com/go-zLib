package http_server_middlewares

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"go.uber.org/fx"
)

func RecoveryMiddleware(log *logger_provider.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		if log != nil {
			log.Errorw("panic recovered", "recovered", recovered, "stack", string(debug.Stack()))
		}
		z.Failure(c, "Internal Server Error", 500)
		c.Abort()
	})
}

func HealthMiddleware(cfg *config_provider.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/.well-known/alive" {
			z.Success(c, map[string]interface{}{
				"status":    "UP",
				"timestamp": time.Now().Unix(),
			})
			c.Abort()
		}
		if c.Request.URL.Path == "/.well-known/health" {
			host := cfg.GetString("http.host")
			port := cfg.GetInt("http.port")
			name := cfg.GetString("app.name")
			z.Success(c, map[string]interface{}{
				"status":    "UP",
				"timestamp": time.Now().Unix(),
				"name":      name,
				"host":      fmt.Sprintf("%s:%d", host, port),
			})
			c.Abort()
		}

		c.Next()
	}
}

var HealthMiddlewareModule = fx.Options(
	fx.Provide(
		fx.Annotate(
			HealthMiddleware,
			fx.ParamTags(``),
			fx.ResultTags(`group:"http_middlewares"`),
		),
	),
)
