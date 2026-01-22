package http_server_middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/providers/rate_limiter_provider"
	"github.com/ulule/limiter/v3"
	limitergingw "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"go.uber.org/fx"
)

type rateLimiterStrategy struct {
	Name    string
	Rate    string
	KeyBy   string
	Prefix  string
	Methods []string
	Paths   []string
	Message string
	Headers bool
}

func rateLimiterPickStrategy(cfg *rate_limiter_provider.ProviderConfig, c *gin.Context) (rate_limiter_provider.Strategy, bool) {
	if cfg == nil {
		return rate_limiter_provider.Strategy{}, false
	}
	for _, st := range cfg.Strategies {
		if !rateLimiterMatchStrategy(c, st) {
			continue
		}
		return st, true
	}
	return rate_limiter_provider.Strategy{}, false
}

func rateLimiterMatchStrategy(c *gin.Context, st rate_limiter_provider.Strategy) bool {
	if len(st.Methods) > 0 {
		ok := false
		for _, m := range st.Methods {
			if strings.EqualFold(m, c.Request.Method) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if len(st.Paths) > 0 {
		p := c.FullPath()
		if p == "" {
			p = c.Request.URL.Path
		}
		ok := false
		for _, rp := range st.Paths {
			if rp == p {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	return true
}

// rateLimiterKeyGetter 构造 limiter key getter。
func rateLimiterKeyGetter(p *rate_limiter_provider.RateLimiter, st rate_limiter_provider.Strategy) limitergingw.KeyGetter {
	return func(c *gin.Context) string {
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		ip := c.ClientIP()
		guard := ""
		userID := ""
		if v, ok := c.Get("guard"); ok {
			if s, ok2 := v.(string); ok2 {
				guard = s
			}
		}
		if v, ok := c.Get("user_id"); ok {
			if s, ok2 := v.(string); ok2 {
				userID = s
			}
		}
		key := p.BuildKey(method, path, ip, guard, userID, st.KeyBy)
		if st.Prefix != "" {
			key = st.Prefix + ":" + key
		}
		if st.Name != "" {
			return st.Name + ":" + key
		}
		return key
	}
}

func rateLimiterRateFromStrategyOrDefault(st rate_limiter_provider.Strategy, def limiter.Rate) (limiter.Rate, error) {
	if strings.TrimSpace(st.Rate) == "" {
		return def, nil
	}
	return limiter.NewRateFromFormatted(st.Rate)
}

// RateLimiterMiddleware 创建限流 Gin 中间件。
func RateLimiterMiddleware(p *rate_limiter_provider.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if p == nil || !p.Enabled() {
			c.Next()
			return
		}

		cfg := p.GinConfig()
		st, ok := rateLimiterPickStrategy(cfg, c)
		if ok {
			rate, err := rateLimiterRateFromStrategyOrDefault(st, p.DefaultRate())
			if err == nil {
				inst := limiter.New(p.Store(), rate)
				mw := limitergingw.NewMiddleware(
					inst,
					limitergingw.WithKeyGetter(rateLimiterKeyGetter(p, st)),
					limitergingw.WithLimitReachedHandler(func(c *gin.Context) {
						message := "TOO_MANY_REQUESTS"
						if st.Message != "" {
							message = st.Message
						}
						c.AbortWithStatusJSON(http.StatusTooManyRequests, map[string]interface{}{
							"error":   "RATE_LIMITED",
							"message": message,
						})
					}),
				)
				mw(c)
				return
			}
			// err: fail open
			c.Next()
			return
		}

		// default
		mw := limitergingw.NewMiddleware(
			p.Limiter(),
			limitergingw.WithKeyGetter(func(c *gin.Context) string {
				method := c.Request.Method
				path := c.FullPath()
				if path == "" {
					path = c.Request.URL.Path
				}
				ip := c.ClientIP()
				guard := ""
				userID := ""
				if v, ok := c.Get("guard"); ok {
					if s, ok2 := v.(string); ok2 {
						guard = s
					}
				}
				if v, ok := c.Get("user_id"); ok {
					if s, ok2 := v.(string); ok2 {
						userID = s
					}
				}
				return "default:" + p.BuildKey(method, path, ip, guard, userID, "ip")
			}),
			limitergingw.WithLimitReachedHandler(func(c *gin.Context) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":   "RATE_LIMITED",
					"message": "TOO_MANY_REQUESTS",
				})
			}),
		)
		mw(c)
	}
}

var RateLimiterMiddlewareModule = fx.Options(
	fx.Provide(
		fx.Annotate(
			RateLimiterMiddleware,
			fx.ParamTags(``),
			fx.ResultTags(`group:"http_middlewares"`),
		),
	),
)
