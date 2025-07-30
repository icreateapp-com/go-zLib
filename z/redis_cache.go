package z

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisCache struct {
	client *redis.Client
}

var RedisCache redisCache

// Init 初始化 redis
func (r *redisCache) Init() {
	host, _ := Config.String("config.redis.host")
	port, _ := Config.Int("config.redis.port")
	password, _ := Config.String("config.redis.password")
	db, _ := Config.Int("config.redis.db")

	r.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	if _, err := r.client.Ping(ctx).Result(); err != nil {
		Error.Fatal("redis connect error: ", err.Error())
	}
}

// Get 获取 key 的值
func (r *redisCache) Get(key string, dest *interface{}) error {
	ctx := context.Background()
	res, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(res), dest)
}

// Set 设置 key 的值
func (r *redisCache) Set(key string, value interface{}, duration time.Duration) error {
	ctx := context.Background()

	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, jsonValue, duration).Err()
}

// Exists 判断 key 是否存在
func (r *redisCache) Exists(key string) bool {
	ctx := context.Background()

	if exists, err := r.client.Exists(ctx, key).Result(); 0 == exists || err != nil {
		return false
	}

	return true
}

// Delete 删除 key
func (r *redisCache) Delete(key string) error {
	ctx := context.Background()

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

// Expire 设置 key 的过期时间
func (r *redisCache) Expire(key string, duration time.Duration) error {
	ctx := context.Background()

	if err := r.client.Expire(ctx, key, duration).Err(); err != nil {
		return err
	}

	return nil
}

// TTL 获取 key 剩余的时间
func (r *redisCache) TTL(key string) (time.Duration, error) {
	ctx := context.Background()

	duration, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return duration, nil
}

// Keys 根据模式获取匹配的键列表
func (r *redisCache) Keys(pattern string) ([]string, error) {
	ctx := context.Background()

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	return keys, nil
}
