package http_server_middlewares

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"go.uber.org/fx"
)

type corsConfig struct {
	enabled          bool
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	allowCredentials bool
	maxAge           int
}

func corsConfigFrom(cfg *config_provider.Config) corsConfig {
	conf := corsConfig{
		enabled:          cfg.GetBool("cors.enabled", false),
		allowOrigins:     cfg.GetStringSlice("cors.allow_origins"),
		allowMethods:     cfg.GetStringSlice("cors.allow_methods"),
		allowHeaders:     cfg.GetStringSlice("cors.allow_headers"),
		exposeHeaders:    cfg.GetStringSlice("cors.expose_headers"),
		allowCredentials: cfg.GetBool("cors.allow_credentials", false),
		maxAge:           cfg.GetInt("cors.max_age", 0),
	}
	if len(conf.allowMethods) == 0 {
		conf.allowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(conf.allowHeaders) == 0 {
		conf.allowHeaders = []string{"Authorization", "Content-Type", "X-Requested-With"}
	}
	return conf
}

func corsResolveOrigin(origin string, conf corsConfig) string {
	if origin == "" {
		return ""
	}
	if len(conf.allowOrigins) == 0 {
		if conf.allowCredentials {
			return origin
		}
		return "*"
	}
	for _, allowed := range conf.allowOrigins {
		if allowed == "*" {
			if conf.allowCredentials {
				return origin
			}
			return "*"
		}
		if corsMatchOrigin(allowed, origin) {
			return origin
		}
	}
	return ""
}

func corsMatchOrigin(allowed, origin string) bool {
	if strings.EqualFold(allowed, origin) {
		return true
	}
	if !strings.Contains(allowed, "*") {
		return false
	}
	parts := strings.Split(allowed, "*")
	if len(parts) == 0 {
		return false
	}
	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(strings.ToLower(origin[idx:]), strings.ToLower(part))
		if pos < 0 {
			return false
		}
		idx += pos + len(part)
		if i == 0 && !strings.HasPrefix(strings.ToLower(origin), strings.ToLower(part)) {
			return false
		}
	}
	last := parts[len(parts)-1]
	if last != "" && !strings.HasSuffix(strings.ToLower(origin), strings.ToLower(last)) {
		return false
	}
	return true
}

// CorsMiddleware HTTP CORS 中间件。
func CorsMiddleware(cfg *config_provider.Config) gin.HandlerFunc {
	conf := corsConfigFrom(cfg)
	return func(c *gin.Context) {
		if !conf.enabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		allowedOrigin := corsResolveOrigin(origin, conf)
		if allowedOrigin == "" {
			c.Next()
			return
		}

		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", allowedOrigin)
		if conf.allowCredentials {
			headers.Set("Access-Control-Allow-Credentials", "true")
		}
		if len(conf.exposeHeaders) > 0 {
			headers.Set("Access-Control-Expose-Headers", strings.Join(conf.exposeHeaders, ", "))
		}
		if allowedOrigin != "*" {
			headers.Add("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			headers.Set("Access-Control-Allow-Methods", strings.Join(conf.allowMethods, ", "))
			headers.Set("Access-Control-Allow-Headers", strings.Join(conf.allowHeaders, ", "))
			if conf.maxAge > 0 {
				headers.Set("Access-Control-Max-Age", strconv.Itoa(conf.maxAge))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

var CorsMiddlewareModule = fx.Options(
	fx.Provide(
		fx.Annotate(
			CorsMiddleware,
			fx.ParamTags(``),
			fx.ResultTags(`group:"http_middlewares"`),
		),
	),
)
