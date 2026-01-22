package rate_limiter_provider

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/redis_provider"
	"github.com/ulule/limiter/v3"
	limiterredis "github.com/ulule/limiter/v3/drivers/store/redis"
	"go.uber.org/fx"
)

// RateLimiter 提供基于 Redis 的分布式限流能力。
type RateLimiter struct {
	cfg   *config_provider.Config
	log   *logger_provider.Logger
	redis *redis_provider.Redis

	defaultRate limiter.Rate
	store       limiter.Store
	limiter     *limiter.Limiter

	prefix       string
	clientIPHdr  string
	ipv6MaskBits int

	enabled bool
}

// In 表示 RateLimiter 的 fx 入参。
type In struct {
	fx.In

	Cfg   *config_provider.Config
	Log   *logger_provider.Logger
	Redis *redis_provider.Redis `optional:"true"`
}

// NewRateLimiterProvider 创建 RateLimiter 实例。
func NewRateLimiterProvider(in In) (*RateLimiter, error) {
	p := &RateLimiter{cfg: in.Cfg, log: in.Log, redis: in.Redis}

	enabled := in.Cfg.GetBool("rate_limiter.enabled", false)
	p.enabled = enabled
	if !enabled {
		if p.log != nil {
			p.log.Infow("provider[rate_limiter] disabled")
		}
		return p, nil
	}

	if p.redis == nil {
		return nil, errors.New("rate_limiter enabled but redis provider is nil")
	}

	prefix := strings.TrimSpace(in.Cfg.GetString("rate_limiter.redis.prefix", "limiter"))
	p.prefix = prefix

	clientIPHdr := strings.TrimSpace(in.Cfg.GetString("rate_limiter.client_ip_header", ""))
	p.clientIPHdr = clientIPHdr

	ipv6MaskBits := in.Cfg.GetInt("rate_limiter.ipv6_mask_bits", 0)
	p.ipv6MaskBits = ipv6MaskBits

	rateStr := strings.TrimSpace(in.Cfg.GetString("rate_limiter.default_rate", "60-M"))
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limiter.default_rate: %w", err)
	}
	p.defaultRate = rate

	store, err := limiterredis.NewStoreWithOptions(p.redis.Client(), limiter.StoreOptions{Prefix: p.prefix})
	if err != nil {
		return nil, err
	}
	p.store = store

	var opts []limiter.Option
	if p.clientIPHdr != "" {
		opts = append(opts, limiter.WithClientIPHeader(p.clientIPHdr))
	}
	if p.ipv6MaskBits > 0 {
		opts = append(opts, limiter.WithIPv6Mask(net.CIDRMask(p.ipv6MaskBits, 128)))
	}

	p.limiter = limiter.New(p.store, p.defaultRate, opts...)

	if p.log != nil {
		p.log.Infow("provider[rate_limiter] enabled", "default_rate", rateStr, "prefix", p.prefix)
	}

	return p, nil
}

// ProviderConfig 表示给 HTTP 中间件层使用的限流配置快照。
type ProviderConfig struct {
	Strategies []Strategy
}

// Strategy 表示一个命名限流策略。
type Strategy struct {
	Name    string
	Rate    string
	KeyBy   string
	Prefix  string
	Methods []string
	Paths   []string
	Message string
	Headers bool
}

// Enabled 返回限流是否启用。
func (p *RateLimiter) Enabled() bool { return p.enabled }

// Store 返回底层 limiter store。
func (p *RateLimiter) Store() limiter.Store { return p.store }

// Limiter 返回默认 limiter 实例。
func (p *RateLimiter) Limiter() *limiter.Limiter { return p.limiter }

// DefaultRate 返回默认限流速率。
func (p *RateLimiter) DefaultRate() limiter.Rate { return p.defaultRate }

// GinConfig 返回给 Gin middleware 使用的策略列表。
func (p *RateLimiter) GinConfig() *ProviderConfig {
	if p.cfg == nil || !p.enabled {
		return nil
	}

	strategies := p.cfg.GetStringMap("rate_limiter.strategies")
	if strategies == nil {
		return &ProviderConfig{Strategies: nil}
	}

	items := make([]Strategy, 0, len(strategies))
	for name, vv := range strategies {
		m, ok := vv.(map[string]interface{})
		if !ok {
			continue
		}
		rateStr, _ := m["rate"].(string)
		keyBy, _ := m["key_by"].(string)
		prefix, _ := m["prefix"].(string)
		message, _ := m["message"].(string)
		headers, _ := m["headers"].(bool)

		var methods []string
		if vMethods, ok := m["methods"].([]interface{}); ok {
			for _, it := range vMethods {
				if s, ok := it.(string); ok {
					methods = append(methods, strings.ToUpper(strings.TrimSpace(s)))
				}
			}
		}
		var paths []string
		if vPaths, ok := m["paths"].([]interface{}); ok {
			for _, it := range vPaths {
				if s, ok := it.(string); ok {
					paths = append(paths, strings.TrimSpace(s))
				}
			}
		}

		items = append(items, Strategy{
			Name:    name,
			Rate:    strings.TrimSpace(rateStr),
			KeyBy:   strings.TrimSpace(keyBy),
			Prefix:  strings.TrimSpace(prefix),
			Methods: methods,
			Paths:   paths,
			Message: strings.TrimSpace(message),
			Headers: headers,
		})
	}

	return &ProviderConfig{Strategies: items}
}

// BuildKey 根据策略计算限流 key。
func (p *RateLimiter) BuildKey(method string, path string, clientIP string, guard string, userID string, keyBy string) string {
	switch strings.TrimSpace(strings.ToLower(keyBy)) {
	case "ip":
		if clientIP == "" {
			return "unknown"
		}
		return clientIP
	case "path":
		return path
	case "method_path":
		return method + ":" + path
	case "user_id":
		if userID == "" {
			return "anonymous"
		}
		return userID
	case "guard":
		if guard == "" {
			return "unknown"
		}
		return guard
	case "guard_user_id":
		g := guard
		u := userID
		if g == "" {
			g = "unknown"
		}
		if u == "" {
			u = "anonymous"
		}
		return g + ":" + u
	default:
		if clientIP == "" {
			return "unknown"
		}
		return clientIP
	}
}

// RateLimiterProviderModule 提供 RateLimiter 的 fx 模块。
var RateLimiterProviderModule = fx.Options(
	fx.Provide(NewRateLimiterProvider),
)
