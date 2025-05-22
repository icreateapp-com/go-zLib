# 缓存模块

go-zLib 的缓存模块提供了内存缓存和 Redis 缓存两种实现，用于提高数据读取效率，减少重复计算或数据库访问。

## 目录
- [内存缓存](#内存缓存)
- [Redis 缓存](#redis-缓存)

## 内存缓存

内存缓存基于 `github.com/patrickmn/go-cache` 库实现，提供了轻量级的进程内缓存功能。

### 使用方法

```go
import "github.com/icreateapp-com/go-zLib/z"

func main() {
    // 设置缓存，设置 1 分钟过期
    z.MemCache.Set("user:1", map[string]interface{}{
        "id": 1,
        "name": "张三",
    }, time.Minute)
    
    // 获取缓存
    value, found := z.MemCache.Get("user:1")
    if found {
        user := value.(map[string]interface{})
        fmt.Println("用户名:", user["name"])
    }
    
    // 删除缓存
    z.MemCache.Delete("user:1")
    
    // 获取不存在的缓存
    value, found = z.MemCache.Get("user:1")
    if !found {
        fmt.Println("缓存不存在")
    }
}
```

### 方法说明

#### Set 方法

设置缓存。

**参数**：
- key (string): 缓存键
- value (interface{}): 缓存值
- duration (time.Duration): 过期时间

**返回值**：无

```go
// 永不过期
z.MemCache.Set("permanent", "value", -1)

// 1 小时后过期
z.MemCache.Set("temporary", "value", time.Hour)
```

#### Get 方法

获取缓存。

**参数**：
- key (string): 缓存键

**返回值**：
- (interface{}, bool): 缓存值和是否找到

```go
value, found := z.MemCache.Get("key")
if found {
    // 使用缓存值
    fmt.Println(value)
} else {
    // 缓存不存在
}
```

#### Delete 方法

删除缓存。

**参数**：
- key (string): 缓存键

**返回值**：无

```go
z.MemCache.Delete("key")
```

#### Flush 方法

清空所有缓存。

**参数**：无

**返回值**：无

```go
z.MemCache.Flush()
```

### 注意事项

1. 内存缓存存储在当前进程内存中，不支持多进程共享
2. 当应用重启后，所有缓存将被清空
3. 缓存值可以是任意类型，但使用时需要进行类型断言
4. 内存缓存会定期自动清理过期项目，无需手动维护

## Redis 缓存

Redis 缓存基于 `github.com/redis/go-redis/v9` 库实现，提供了分布式缓存功能。

### 使用方法

```go
import "github.com/icreateapp-com/go-zLib/z"

func main() {
    // 设置缓存，设置 1 分钟过期
    err := z.RedisCache.Set("user:1", `{"id": 1, "name": "张三"}`, time.Minute)
    if err != nil {
        // 处理错误
    }
    
    // 获取缓存
    value, err := z.RedisCache.Get("user:1")
    if err == nil {
        fmt.Println("缓存值:", value)
    }
    
    // 删除缓存
    err = z.RedisCache.Delete("user:1")
    if err != nil {
        // 处理错误
    }
    
    // 检查键是否存在
    exists, err := z.RedisCache.Exists("user:1")
    if err == nil && !exists {
        fmt.Println("缓存不存在")
    }
}
```

### 配置

Redis 缓存需要在配置文件中设置 Redis 连接信息：

```yaml
config:
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
    prefix: "myapp:"  # 键前缀，可选
```

### 方法说明

#### Set 方法

设置缓存。

**参数**：
- key (string): 缓存键
- value (string): 缓存值
- duration (time.Duration): 过期时间

**返回值**：
- error: 可能的错误

```go
// 10 分钟后过期
err := z.RedisCache.Set("user:1", `{"id": 1, "name": "张三"}`, 10*time.Minute)

// 永不过期
err := z.RedisCache.Set("config:theme", "dark", -1)
```

#### Get 方法

获取缓存。

**参数**：
- key (string): 缓存键

**返回值**：
- (string, error): 缓存值和可能的错误

```go
value, err := z.RedisCache.Get("user:1")
if err != nil {
    if err == redis.Nil {
        // 缓存不存在
    } else {
        // 其他错误
    }
} else {
    // 使用缓存值
    fmt.Println(value)
}
```

#### Delete 方法

删除缓存。

**参数**：
- key (string): 缓存键

**返回值**：
- error: 可能的错误

```go
err := z.RedisCache.Delete("user:1")
```

#### Exists 方法

检查键是否存在。

**参数**：
- key (string): 缓存键

**返回值**：
- (bool, error): 是否存在和可能的错误

```go
exists, err := z.RedisCache.Exists("user:1")
if err == nil && exists {
    fmt.Println("缓存存在")
}
```

#### Expire 方法

更新键的过期时间。

**参数**：
- key (string): 缓存键
- duration (time.Duration): 新的过期时间

**返回值**：
- error: 可能的错误

```go
// 延长过期时间到 1 小时
err := z.RedisCache.Expire("user:1", time.Hour)
```

#### GetRaw 方法

获取 Redis 客户端实例，用于执行自定义命令。

**参数**：无

**返回值**：
- (*redis.Client): Redis 客户端实例

```go
client := z.RedisCache.GetRaw()

// 执行自定义命令
val, err := client.HGetAll(context.Background(), "user:profile:1").Result()
if err != nil {
    // 处理错误
}
```

### 注意事项

1. Redis 缓存可用于多进程共享数据
2. Redis 键会自动添加配置的前缀，以便于多应用共享同一 Redis 实例
3. 与内存缓存不同，Redis 缓存的值只支持字符串类型，通常使用 JSON 格式存储复杂数据
4. 操作 Redis 缓存时需要处理可能的错误 