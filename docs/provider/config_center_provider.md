# 配置中心提供者 (Config Center Provider)

配置中心提供者为应用程序提供动态配置管理、配置同步和变更通知功能，支持配置的热更新和集中管理。

## 功能特性

- **动态配置管理**: 支持配置的实时更新和同步
- **配置版本控制**: 支持配置版本管理和回滚
- **变更通知**: 配置变更时自动通知应用程序
- **环境隔离**: 支持多环境配置管理
- **安全认证**: 支持Token认证和权限控制
- **缓存机制**: 本地缓存提高配置访问性能

## 快速开始

### 1. 配置文件

在 `config.yaml` 中添加配置中心配置：

```yaml
config:
  config_center:
    version: 1                                              # 配置版本
    address: "http://config-center.example.com"            # 配置中心地址
    env_id: "prod"                                          # 环境ID
    token: "your-config-center-token"                       # 认证令牌
    callback: "http://my-service.example.com/.well-known/config"  # 回调地址
    timeout: 30                                             # 请求超时时间(秒)
    retry_count: 3                                          # 重试次数
    cache_ttl: 300                                          # 缓存TTL(秒)
```

### 2. 基础使用

```go
package main

import (
    "log"
    
    "github.com/icreateapp-com/go-zLib/z"
    "github.com/icreateapp-com/go-zLib/z/provider/config_center_provider"
)

func main() {
    // 注册配置中心提供者
    config_center_provider.ConfigCenterProvider.Register()
    
    // 获取配置
    dbHost := z.Config.String("database.host")
    dbPort := z.Config.Int("database.port")
    
    log.Printf("数据库配置: %s:%d", dbHost, dbPort)
    
    // 监听配置变更
    setupConfigListener()
    
    // 启动应用
    startApplication()
}

func setupConfigListener() {
    // 监听配置变更事件
    event_bus_provider.On("config.changed", func(event event_bus_provider.Event) {
        log.Println("配置已更新，重新加载应用配置")
        
        // 重新加载数据库连接
        if err := reloadDatabase(); err != nil {
            log.Printf("数据库重载失败: %v", err)
        }
        
        // 重新加载缓存配置
        if err := reloadCache(); err != nil {
            log.Printf("缓存重载失败: %v", err)
        }
    })
}
```

## API 参考

### ConfigCenterProvider

#### Register()

注册配置中心提供者，启动配置同步和监听服务。

```go
config_center_provider.ConfigCenterProvider.Register()
```

#### Middleware()

返回Gin中间件，用于处理配置中心的回调请求。

```go
r := gin.Default()
r.Use(config_center_provider.ConfigCenterProvider.Middleware())
```

#### GetConfig(key string) interface{}

获取指定键的配置值。

```go
value := config_center_provider.ConfigCenterProvider.GetConfig("database.host")
```

#### SetConfig(key string, value interface{}) error

设置配置值（仅本地缓存）。

```go
err := config_center_provider.ConfigCenterProvider.SetConfig("feature.enabled", true)
```

#### RefreshConfig() error

手动刷新配置。

```go
err := config_center_provider.ConfigCenterProvider.RefreshConfig()
if err != nil {
    log.Printf("配置刷新失败: %v", err)
}
```

## 中间件集成

### Gin 中间件

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/provider/config_center_provider"
)

func main() {
    r := gin.Default()
    
    // 添加配置中心中间件
    r.Use(config_center_provider.ConfigCenterProvider.Middleware())
    
    // 配置中心回调端点会自动注册到 /.well-known/config
    
    r.GET("/api/status", func(c *gin.Context) {
        // 获取实时配置
        maintenanceMode := z.Config.Bool("maintenance.enabled")
        
        if maintenanceMode {
            c.JSON(503, gin.H{
                "status": "maintenance",
                "message": "系统维护中",
            })
            return
        }
        
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    r.Run(":8080")
}
```

## 事件集成

### 配置变更事件

```go
import "github.com/icreateapp-com/go-zLib/z/provider/event_bus_provider"

func setupConfigEvents() {
    // 监听配置变更
    event_bus_provider.On("config.changed", func(event event_bus_provider.Event) {
        changeInfo := event.Payload.(map[string]interface{})
        
        log.Printf("配置变更: %v", changeInfo)
        
        // 根据变更的配置项执行相应操作
        if changedKeys, ok := changeInfo["changed_keys"].([]string); ok {
            for _, key := range changedKeys {
                handleConfigChange(key)
            }
        }
    })
    
    // 监听配置同步失败事件
    event_bus_provider.On("config.sync.failed", func(event event_bus_provider.Event) {
        errorInfo := event.Payload.(map[string]interface{})
        log.Printf("配置同步失败: %v", errorInfo)
        
        // 发送告警通知
        sendAlert("配置同步失败", errorInfo)
    })
}

func handleConfigChange(key string) {
    switch key {
    case "database.host", "database.port":
        // 重新连接数据库
        reloadDatabase()
    case "cache.redis.host", "cache.redis.port":
        // 重新连接Redis
        reloadCache()
    case "log.level":
        // 更新日志级别
        updateLogLevel()
    }
}
```

## 配置管理

### 配置结构

```go
type AppConfig struct {
    Database struct {
        Host     string `yaml:"host"`
        Port     int    `yaml:"port"`
        Username string `yaml:"username"`
        Password string `yaml:"password"`
    } `yaml:"database"`
    
    Cache struct {
        Redis struct {
            Host     string `yaml:"host"`
            Port     int    `yaml:"port"`
            Password string `yaml:"password"`
            DB       int    `yaml:"db"`
        } `yaml:"redis"`
    } `yaml:"cache"`
    
    Features struct {
        EnableNewUI    bool `yaml:"enable_new_ui"`
        EnablePayment  bool `yaml:"enable_payment"`
        MaxUploadSize  int  `yaml:"max_upload_size"`
    } `yaml:"features"`
}

// 获取强类型配置
func getAppConfig() (*AppConfig, error) {
    var config AppConfig
    
    // 从配置中心获取配置并解析
    configData := z.Config.StringMap("app")
    
    // 将map转换为结构体
    if err := mapstructure.Decode(configData, &config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

### 配置验证

```go
func validateConfig() error {
    // 验证必需的配置项
    requiredConfigs := []string{
        "database.host",
        "database.port",
        "cache.redis.host",
    }
    
    for _, key := range requiredConfigs {
        if !z.Config.IsSet(key) {
            return fmt.Errorf("缺少必需的配置项: %s", key)
        }
    }
    
    // 验证配置值的有效性
    if port := z.Config.Int("database.port"); port <= 0 || port > 65535 {
        return fmt.Errorf("无效的数据库端口: %d", port)
    }
    
    return nil
}
```

## 安全考虑

### 敏感信息处理

```go
// 敏感配置加密存储
func getSecureConfig(key string) (string, error) {
    encryptedValue := z.Config.String(key)
    if encryptedValue == "" {
        return "", fmt.Errorf("配置项不存在: %s", key)
    }
    
    // 解密配置值
    decryptedValue, err := decrypt(encryptedValue)
    if err != nil {
        return "", fmt.Errorf("配置解密失败: %v", err)
    }
    
    return decryptedValue, nil
}

// 配置访问权限控制
func checkConfigAccess(key string) bool {
    // 检查当前服务是否有权限访问该配置
    allowedKeys := z.Config.StringSlice("config_center.allowed_keys")
    
    for _, allowedKey := range allowedKeys {
        if strings.HasPrefix(key, allowedKey) {
            return true
        }
    }
    
    return false
}
```

### Token 管理

```go
// 动态更新Token
func updateConfigCenterToken() {
    newToken := getTokenFromSecureStore()
    
    // 更新配置中心Token
    z.Config.Set("config_center.token", newToken)
    
    // 重新注册配置中心提供者
    config_center_provider.ConfigCenterProvider.Register()
}
```

## 错误处理

### 错误类型

```go
const (
    ErrConfigCenterUnreachable = "CONFIG_CENTER_UNREACHABLE"
    ErrInvalidToken           = "INVALID_TOKEN"
    ErrConfigNotFound         = "CONFIG_NOT_FOUND"
    ErrConfigSyncFailed       = "CONFIG_SYNC_FAILED"
)

func handleConfigError(err error) {
    switch {
    case strings.Contains(err.Error(), "connection refused"):
        log.Printf("配置中心不可达: %v", err)
        // 使用本地缓存配置
        useLocalCache()
        
    case strings.Contains(err.Error(), "unauthorized"):
        log.Printf("配置中心认证失败: %v", err)
        // 尝试刷新Token
        refreshToken()
        
    default:
        log.Printf("配置中心未知错误: %v", err)
    }
}
```

### 降级策略

```go
func setupFallbackStrategy() {
    // 配置中心不可用时的降级策略
    event_bus_provider.On("config.center.unavailable", func(event event_bus_provider.Event) {
        log.Println("配置中心不可用，启用降级模式")
        
        // 使用本地配置文件
        loadLocalConfig()
        
        // 禁用非关键功能
        disableNonCriticalFeatures()
        
        // 定期重试连接
        startRetryTimer()
    })
}

func loadLocalConfig() {
    // 加载本地备份配置
    localConfigPath := "config/fallback.yaml"
    if _, err := os.Stat(localConfigPath); err == nil {
        z.LoadConfig(localConfigPath)
        log.Println("已加载本地备份配置")
    }
}
```

## 性能优化

### 缓存策略

```go
// 配置缓存管理
type ConfigCache struct {
    cache    map[string]interface{}
    mutex    sync.RWMutex
    ttl      time.Duration
    lastSync time.Time
}

func (c *ConfigCache) Get(key string) (interface{}, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    // 检查缓存是否过期
    if time.Since(c.lastSync) > c.ttl {
        return nil, false
    }
    
    value, exists := c.cache[key]
    return value, exists
}

func (c *ConfigCache) Set(key string, value interface{}) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    if c.cache == nil {
        c.cache = make(map[string]interface{})
    }
    
    c.cache[key] = value
    c.lastSync = time.Now()
}
```

### 批量更新

```go
// 批量配置更新
func batchUpdateConfig(updates map[string]interface{}) error {
    // 收集配置变更
    var changedKeys []string
    
    for key, value := range updates {
        oldValue := z.Config.Get(key)
        if !reflect.DeepEqual(oldValue, value) {
            z.Config.Set(key, value)
            changedKeys = append(changedKeys, key)
        }
    }
    
    // 如果有变更，发布事件
    if len(changedKeys) > 0 {
        event_bus_provider.EmitAsync("config.changed", map[string]interface{}{
            "changed_keys": changedKeys,
            "timestamp":    time.Now(),
        })
    }
    
    return nil
}
```

## 监控和日志

### 配置访问监控

```go
func monitorConfigAccess() {
    // 监控配置访问频率
    event_bus_provider.On("config.accessed", func(event event_bus_provider.Event) {
        accessInfo := event.Payload.(map[string]interface{})
        
        // 记录访问日志
        log.Printf("配置访问: key=%s, source=%s", 
            accessInfo["key"], accessInfo["source"])
        
        // 更新访问统计
        updateAccessMetrics(accessInfo)
    })
}

func updateAccessMetrics(accessInfo map[string]interface{}) {
    // 更新Prometheus指标
    // configAccessCounter.WithLabelValues(
    //     accessInfo["key"].(string),
    //     accessInfo["source"].(string),
    // ).Inc()
}
```

### 健康检查

```go
func configCenterHealthCheck() bool {
    // 检查配置中心连接状态
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := config_center_provider.ConfigCenterProvider.Ping(ctx)
    return err == nil
}

// 注册健康检查端点
func registerHealthCheck(r *gin.Engine) {
    r.GET("/health/config-center", func(c *gin.Context) {
        if configCenterHealthCheck() {
            c.JSON(200, gin.H{
                "status": "healthy",
                "service": "config-center",
            })
        } else {
            c.JSON(503, gin.H{
                "status": "unhealthy",
                "service": "config-center",
            })
        }
    })
}
```

## 最佳实践

### 1. 配置分层管理

```go
// 配置优先级: 环境变量 > 配置中心 > 本地文件 > 默认值
func getConfigWithPriority(key string, defaultValue interface{}) interface{} {
    // 1. 检查环境变量
    if envValue := os.Getenv(strings.ToUpper(strings.ReplaceAll(key, ".", "_"))); envValue != "" {
        return envValue
    }
    
    // 2. 检查配置中心
    if z.Config.IsSet(key) {
        return z.Config.Get(key)
    }
    
    // 3. 返回默认值
    return defaultValue
}
```

### 2. 配置变更影响分析

```go
func analyzeConfigImpact(changedKeys []string) {
    impactMap := map[string][]string{
        "database": {"数据库连接", "事务处理", "数据迁移"},
        "cache":    {"缓存连接", "会话存储", "临时数据"},
        "auth":     {"用户认证", "权限验证", "Token生成"},
    }
    
    for _, key := range changedKeys {
        for prefix, impacts := range impactMap {
            if strings.HasPrefix(key, prefix) {
                log.Printf("配置变更影响分析: %s -> %v", key, impacts)
                
                // 发送影响分析事件
                event_bus_provider.EmitAsync("config.impact.analysis", map[string]interface{}{
                    "key":     key,
                    "impacts": impacts,
                })
            }
        }
    }
}
```

### 3. 配置回滚机制

```go
type ConfigSnapshot struct {
    Version   int                    `json:"version"`
    Timestamp time.Time              `json:"timestamp"`
    Config    map[string]interface{} `json:"config"`
}

func createConfigSnapshot() *ConfigSnapshot {
    return &ConfigSnapshot{
        Version:   getCurrentConfigVersion(),
        Timestamp: time.Now(),
        Config:    z.Config.AllSettings(),
    }
}

func rollbackConfig(snapshot *ConfigSnapshot) error {
    log.Printf("回滚配置到版本: %d", snapshot.Version)
    
    // 应用快照配置
    for key, value := range snapshot.Config {
        z.Config.Set(key, value)
    }
    
    // 发布配置回滚事件
    event_bus_provider.EmitAsync("config.rollback", map[string]interface{}{
        "version":   snapshot.Version,
        "timestamp": snapshot.Timestamp,
    })
    
    return nil
}
```

配置中心提供者为应用程序提供了强大的动态配置管理能力，通过合理使用可以实现配置的集中管理、实时更新和安全控制。