package redis_provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// Redis 封装 go-redis 客户端并提供常用便捷方法。
type Redis struct {
	client *redis.Client
	log    *logger_provider.Logger
}

// NewRedisProvider 创建 redis 实例
func NewRedisProvider(lc fx.Lifecycle, cfg *config_provider.Config, log *logger_provider.Logger) (*Redis, error) {
	host := cfg.GetString("redis.host")
	port := cfg.GetInt("redis.port")
	password := cfg.GetString("redis.password")
	db := cfg.GetInt("redis.db")

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	r := &Redis{
		client: client,
		log:    log,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if _, err := client.Ping(ctx).Result(); err != nil {
				log.Errorw("redis connect error", "error", err)
				return err
			}
			log.Infow("provider[redis] enabled", "addr", fmt.Sprintf("%s:%d", host, port))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Infow("closing redis connection")
			err := client.Close()
			if err == nil {
				return nil
			}
			errMsg := strings.ToLower(strings.TrimSpace(err.Error()))
			if strings.Contains(errMsg, "bad connection") || strings.Contains(errMsg, "closed") {
				if log != nil {
					log.Debugw("redis connection close ignored", "error", err)
				}
				return nil
			}
			return err
		},
	})

	return r, nil
}

// RedisProviderModule redis 模块
var RedisProviderModule = fx.Options(
	fx.Provide(NewRedisProvider),
)

// Get 获取 key 的值
func (r *Redis) Get(key string, dest interface{}) error {
	ctx := context.Background()
	res, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(res), dest)
}

// Set 设置 key 的值
func (r *Redis) Set(key string, value interface{}, duration time.Duration) error {
	ctx := context.Background()

	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, jsonValue, duration).Err()
}

// Exists 判断 key 是否存在
func (r *Redis) Exists(key string) bool {
	ctx := context.Background()

	exists, err := r.client.Exists(ctx, key).Result()
	if 0 == exists || err != nil {
		return false
	}

	return true
}

// Delete 删除 key
func (r *Redis) Delete(key string) error {
	ctx := context.Background()
	return r.client.Del(ctx, key).Err()
}

// Expire 设置 key 的过期时间
func (r *Redis) Expire(key string, duration time.Duration) error {
	ctx := context.Background()
	return r.client.Expire(ctx, key, duration).Err()
}

// TTL 获取 key 剩余的时间
func (r *Redis) TTL(key string) (time.Duration, error) {
	ctx := context.Background()
	return r.client.TTL(ctx, key).Result()
}

// Keys 根据模式获取匹配的键列表
func (r *Redis) Keys(pattern string) ([]string, error) {
	ctx := context.Background()
	return r.client.Keys(ctx, pattern).Result()
}

// Client 获取 Redis 客户端实例
func (r *Redis) Client() *redis.Client {
	return r.client
}
