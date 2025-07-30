# 服务发现提供者 (Service Discovery Provider)

服务发现提供者为应用程序提供自动化的服务注册和发现功能，支持多种注册中心，实现微服务架构中的服务治理。

## 功能特性

- **多注册中心支持**: 支持 Consul、Etcd、Nacos 等主流注册中心
- **自动服务注册**: 应用启动时自动注册服务信息
- **健康检查**: 定期检查服务健康状态
- **服务发现**: 动态发现和获取可用服务实例
- **负载均衡**: 支持多种负载均衡策略
- **故障转移**: 自动剔除不健康的服务实例
- **配置热更新**: 支持服务配置的动态更新
- **元数据管理**: 支持服务元数据的存储和查询

## 快速开始

### 1. 基本配置

```yaml
# config.yaml
service_discovery:
  registry: "consul"  # 注册中心类型: consul, etcd, nacos
  address: "localhost:8500"
  service:
    name: "user-service"
    version: "v1.0.0"
    port: 8080
    health_check:
      interval: "30s"
      timeout: "10s"
      path: "/health"
  metadata:
    region: "us-west-1"
    zone: "us-west-1a"
    environment: "production"
```

### 2. 服务注册

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"
    
    "github.com/icreateapp-com/go-zLib/z/provider/service_discovery_provider"
)

func main() {
    // 创建服务发现客户端
    client, err := service_discovery_provider.NewClient(&service_discovery_provider.Config{
        Registry: "consul",
        Address:  "localhost:8500",
        Service: &service_discovery_provider.ServiceConfig{
            Name:    "user-service",
            Version: "v1.0.0",
            Port:    8080,
            Tags:    []string{"api", "user", "v1"},
            HealthCheck: &service_discovery_provider.HealthCheckConfig{
                Interval: 30 * time.Second,
                Timeout:  10 * time.Second,
                HTTP:     "http://localhost:8080/health",
            },
        },
        Metadata: map[string]string{
            "region":      "us-west-1",
            "zone":        "us-west-1a",
            "environment": "production",
        },
    })
    if err != nil {
        log.Fatalf("创建服务发现客户端失败: %v", err)
    }
    defer client.Close()
    
    // 注册服务
    if err := client.Register(); err != nil {
        log.Fatalf("注册服务失败: %v", err)
    }
    
    log.Println("服务注册成功")
    
    // 启动 HTTP 服务器
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/users", usersHandler)
    
    log.Println("启动 HTTP 服务器在端口 8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("启动 HTTP 服务器失败: %v", err)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    // 健康检查逻辑
    if checkDatabaseConnection() && checkExternalDependencies() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Service Unavailable"))
    }
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
    // 用户服务逻辑
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"users": []}`))
}

func checkDatabaseConnection() bool {
    // 检查数据库连接
    return true
}

func checkExternalDependencies() bool {
    // 检查外部依赖
    return true
}
```

### 3. 服务发现

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/icreateapp-com/go-zLib/z/provider/service_discovery_provider"
)

func main() {
    // 创建服务发现客户端
    client, err := service_discovery_provider.NewClient(&service_discovery_provider.Config{
        Registry: "consul",
        Address:  "localhost:8500",
    })
    if err != nil {
        log.Fatalf("创建服务发现客户端失败: %v", err)
    }
    defer client.Close()
    
    // 发现服务
    services, err := client.Discover("user-service")
    if err != nil {
        log.Fatalf("发现服务失败: %v", err)
    }
    
    if len(services) == 0 {
        log.Fatalf("未找到可用的用户服务实例")
    }
    
    // 选择服务实例（负载均衡）
    service := selectService(services)
    
    // 调用服务
    url := fmt.Sprintf("http://%s:%d/users", service.Address, service.Port)
    resp, err := http.Get(url)
    if err != nil {
        log.Fatalf("调用服务失败: %v", err)
    }
    defer resp.Body.Close()
    
    log.Printf("调用服务成功，状态码: %d", resp.StatusCode)
}

func selectService(services []*service_discovery_provider.ServiceInstance) *service_discovery_provider.ServiceInstance {
    // 简单的轮询负载均衡
    if len(services) == 0 {
        return nil
    }
    
    // 这里可以实现更复杂的负载均衡策略
    return services[0]
}
```

## API 参考

### 客户端 API

#### NewClient(config *Config) (*Client, error)

创建新的服务发现客户端。

```go
type Config struct {
    Registry    string                 // 注册中心类型
    Address     string                 // 注册中心地址
    Username    string                 // 认证用户名
    Password    string                 // 认证密码
    Timeout     time.Duration          // 连接超时
    Service     *ServiceConfig         // 服务配置
    Metadata    map[string]string      // 元数据
}

type ServiceConfig struct {
    Name        string                 // 服务名称
    Version     string                 // 服务版本
    Address     string                 // 服务地址
    Port        int                    // 服务端口
    Tags        []string               // 服务标签
    HealthCheck *HealthCheckConfig     // 健康检查配置
}

type HealthCheckConfig struct {
    Interval    time.Duration          // 检查间隔
    Timeout     time.Duration          // 检查超时
    HTTP        string                 // HTTP 健康检查 URL
    TCP         string                 // TCP 健康检查地址
    Script      string                 // 脚本健康检查
    TTL         time.Duration          // TTL 健康检查
}

client, err := service_discovery_provider.NewClient(&service_discovery_provider.Config{
    Registry: "consul",
    Address:  "localhost:8500",
    Service: &service_discovery_provider.ServiceConfig{
        Name:    "user-service",
        Version: "v1.0.0",
        Port:    8080,
    },
})
```

#### Client.Register() error

注册服务到注册中心。

```go
if err := client.Register(); err != nil {
    log.Fatalf("注册服务失败: %v", err)
}
```

#### Client.Deregister() error

从注册中心注销服务。

```go
if err := client.Deregister(); err != nil {
    log.Printf("注销服务失败: %v", err)
}
```

#### Client.Discover(serviceName string) ([]*ServiceInstance, error)

发现指定名称的服务实例。

```go
type ServiceInstance struct {
    ID          string                 // 实例ID
    Name        string                 // 服务名称
    Version     string                 // 服务版本
    Address     string                 // 服务地址
    Port        int                    // 服务端口
    Tags        []string               // 服务标签
    Metadata    map[string]string      // 元数据
    Health      string                 // 健康状态
}

services, err := client.Discover("user-service")
if err != nil {
    log.Fatalf("发现服务失败: %v", err)
}

for _, service := range services {
    log.Printf("发现服务: %s:%d", service.Address, service.Port)
}
```

#### Client.Watch(serviceName string, callback func([]*ServiceInstance)) error

监听服务变化。

```go
err := client.Watch("user-service", func(services []*ServiceInstance) {
    log.Printf("服务列表更新，当前实例数: %d", len(services))
    for _, service := range services {
        log.Printf("  - %s:%d (健康状态: %s)", service.Address, service.Port, service.Health)
    }
})
if err != nil {
    log.Fatalf("监听服务失败: %v", err)
}
```

#### Client.UpdateMetadata(metadata map[string]string) error

更新服务元数据。

```go
newMetadata := map[string]string{
    "version":     "v1.1.0",
    "environment": "production",
    "region":      "us-east-1",
}

if err := client.UpdateMetadata(newMetadata); err != nil {
    log.Printf("更新元数据失败: %v", err)
}
```

#### Client.Close() error

关闭客户端连接。

```go
defer client.Close()
```

## 负载均衡策略

### 1. 轮询负载均衡

```go
type RoundRobinBalancer struct {
    services []*service_discovery_provider.ServiceInstance
    current  int
    mutex    sync.Mutex
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
    return &RoundRobinBalancer{}
}

func (b *RoundRobinBalancer) UpdateServices(services []*service_discovery_provider.ServiceInstance) {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    
    // 过滤健康的服务实例
    var healthyServices []*service_discovery_provider.ServiceInstance
    for _, service := range services {
        if service.Health == "passing" {
            healthyServices = append(healthyServices, service)
        }
    }
    
    b.services = healthyServices
    b.current = 0
}

func (b *RoundRobinBalancer) Select() *service_discovery_provider.ServiceInstance {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    
    if len(b.services) == 0 {
        return nil
    }
    
    service := b.services[b.current]
    b.current = (b.current + 1) % len(b.services)
    
    return service
}
```

### 2. 随机负载均衡

```go
type RandomBalancer struct {
    services []*service_discovery_provider.ServiceInstance
    mutex    sync.RWMutex
    rand     *rand.Rand
}

func NewRandomBalancer() *RandomBalancer {
    return &RandomBalancer{
        rand: rand.New(rand.NewSource(time.Now().UnixNano())),
    }
}

func (b *RandomBalancer) UpdateServices(services []*service_discovery_provider.ServiceInstance) {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    
    var healthyServices []*service_discovery_provider.ServiceInstance
    for _, service := range services {
        if service.Health == "passing" {
            healthyServices = append(healthyServices, service)
        }
    }
    
    b.services = healthyServices
}

func (b *RandomBalancer) Select() *service_discovery_provider.ServiceInstance {
    b.mutex.RLock()
    defer b.mutex.RUnlock()
    
    if len(b.services) == 0 {
        return nil
    }
    
    index := b.rand.Intn(len(b.services))
    return b.services[index]
}
```

### 3. 加权轮询负载均衡

```go
type WeightedRoundRobinBalancer struct {
    services []*WeightedService
    mutex    sync.Mutex
}

type WeightedService struct {
    Service       *service_discovery_provider.ServiceInstance
    Weight        int
    CurrentWeight int
}

func NewWeightedRoundRobinBalancer() *WeightedRoundRobinBalancer {
    return &WeightedRoundRobinBalancer{}
}

func (b *WeightedRoundRobinBalancer) UpdateServices(services []*service_discovery_provider.ServiceInstance) {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    
    var weightedServices []*WeightedService
    for _, service := range services {
        if service.Health == "passing" {
            weight := getServiceWeight(service) // 从元数据或配置中获取权重
            weightedServices = append(weightedServices, &WeightedService{
                Service:       service,
                Weight:        weight,
                CurrentWeight: 0,
            })
        }
    }
    
    b.services = weightedServices
}

func (b *WeightedRoundRobinBalancer) Select() *service_discovery_provider.ServiceInstance {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    
    if len(b.services) == 0 {
        return nil
    }
    
    var totalWeight int
    var selected *WeightedService
    
    for _, service := range b.services {
        service.CurrentWeight += service.Weight
        totalWeight += service.Weight
        
        if selected == nil || service.CurrentWeight > selected.CurrentWeight {
            selected = service
        }
    }
    
    if selected != nil {
        selected.CurrentWeight -= totalWeight
        return selected.Service
    }
    
    return nil
}

func getServiceWeight(service *service_discovery_provider.ServiceInstance) int {
    if weightStr, exists := service.Metadata["weight"]; exists {
        if weight, err := strconv.Atoi(weightStr); err == nil && weight > 0 {
            return weight
        }
    }
    return 1 // 默认权重
}
```

## 健康检查

### 1. HTTP 健康检查

```go
func setupHTTPHealthCheck() *service_discovery_provider.HealthCheckConfig {
    return &service_discovery_provider.HealthCheckConfig{
        Interval: 30 * time.Second,
        Timeout:  10 * time.Second,
        HTTP:     "http://localhost:8080/health",
    }
}

// 健康检查端点实现
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    // 检查数据库连接
    if !checkDatabase() {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte(`{"status": "unhealthy", "reason": "database connection failed"}`))
        return
    }
    
    // 检查外部依赖
    if !checkExternalServices() {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte(`{"status": "unhealthy", "reason": "external service unavailable"}`))
        return
    }
    
    // 检查内存使用率
    if getMemoryUsage() > 0.9 {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte(`{"status": "unhealthy", "reason": "high memory usage"}`))
        return
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "healthy"}`))
}

func checkDatabase() bool {
    // 实现数据库连接检查
    return true
}

func checkExternalServices() bool {
    // 实现外部服务检查
    return true
}

func getMemoryUsage() float64 {
    // 实现内存使用率检查
    return 0.5
}
```

### 2. TCP 健康检查

```go
func setupTCPHealthCheck() *service_discovery_provider.HealthCheckConfig {
    return &service_discovery_provider.HealthCheckConfig{
        Interval: 30 * time.Second,
        Timeout:  5 * time.Second,
        TCP:      "localhost:8080",
    }
}
```

### 3. TTL 健康检查

```go
func setupTTLHealthCheck(client *service_discovery_provider.Client) *service_discovery_provider.HealthCheckConfig {
    config := &service_discovery_provider.HealthCheckConfig{
        TTL: 60 * time.Second,
    }
    
    // 启动 TTL 更新 goroutine
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            if isServiceHealthy() {
                client.UpdateTTL("pass", "服务运行正常")
            } else {
                client.UpdateTTL("fail", "服务异常")
            }
        }
    }()
    
    return config
}

func isServiceHealthy() bool {
    // 实现服务健康检查逻辑
    return checkDatabase() && checkExternalServices()
}
```

## 服务网格集成

### 1. Consul Connect 集成

```go
func setupConsulConnect() *service_discovery_provider.Config {
    return &service_discovery_provider.Config{
        Registry: "consul",
        Address:  "localhost:8500",
        Service: &service_discovery_provider.ServiceConfig{
            Name:    "user-service",
            Version: "v1.0.0",
            Port:    8080,
            Connect: &service_discovery_provider.ConnectConfig{
                SidecarService: &service_discovery_provider.SidecarConfig{
                    Port: 21000,
                    Proxy: &service_discovery_provider.ProxyConfig{
                        DestinationServiceName: "user-service",
                        DestinationServiceID:   "user-service-1",
                        LocalServiceAddress:    "127.0.0.1",
                        LocalServicePort:       8080,
                        Upstreams: []*service_discovery_provider.Upstream{
                            {
                                DestinationType: "service",
                                DestinationName: "database-service",
                                LocalBindPort:   5432,
                            },
                        },
                    },
                },
            },
        },
    }
}
```

### 2. Istio 集成

```go
func setupIstioIntegration() *service_discovery_provider.Config {
    return &service_discovery_provider.Config{
        Registry: "consul",
        Address:  "localhost:8500",
        Service: &service_discovery_provider.ServiceConfig{
            Name:    "user-service",
            Version: "v1.0.0",
            Port:    8080,
            Metadata: map[string]string{
                "istio.io/rev":     "default",
                "sidecar.istio.io/inject": "true",
                "version":          "v1",
            },
        },
    }
}
```

## 配置管理

### 1. 动态配置更新

```go
func setupConfigWatch(client *service_discovery_provider.Client) {
    // 监听配置变化
    client.WatchConfig("user-service/config", func(config map[string]interface{}) {
        log.Printf("配置更新: %+v", config)
        
        // 更新应用配置
        updateApplicationConfig(config)
        
        // 更新服务元数据
        metadata := map[string]string{
            "config_version": fmt.Sprintf("%d", time.Now().Unix()),
        }
        client.UpdateMetadata(metadata)
    })
}

func updateApplicationConfig(config map[string]interface{}) {
    // 实现配置更新逻辑
    if dbConfig, exists := config["database"]; exists {
        updateDatabaseConfig(dbConfig.(map[string]interface{}))
    }
    
    if cacheConfig, exists := config["cache"]; exists {
        updateCacheConfig(cacheConfig.(map[string]interface{}))
    }
}

func updateDatabaseConfig(config map[string]interface{}) {
    // 更新数据库配置
    log.Printf("更新数据库配置: %+v", config)
}

func updateCacheConfig(config map[string]interface{}) {
    // 更新缓存配置
    log.Printf("更新缓存配置: %+v", config)
}
```

### 2. 配置模板

```yaml
# consul-config.yaml
service_discovery:
  registry: "consul"
  address: "${CONSUL_ADDRESS:localhost:8500}"
  username: "${CONSUL_USERNAME:}"
  password: "${CONSUL_PASSWORD:}"
  timeout: "${CONSUL_TIMEOUT:30s}"
  
  service:
    name: "${SERVICE_NAME:user-service}"
    version: "${SERVICE_VERSION:v1.0.0}"
    address: "${SERVICE_ADDRESS:}"
    port: ${SERVICE_PORT:8080}
    tags:
      - "${ENVIRONMENT:development}"
      - "api"
      - "user"
    
    health_check:
      interval: "${HEALTH_CHECK_INTERVAL:30s}"
      timeout: "${HEALTH_CHECK_TIMEOUT:10s}"
      http: "http://${SERVICE_ADDRESS:localhost}:${SERVICE_PORT:8080}/health"
  
  metadata:
    region: "${AWS_REGION:us-west-1}"
    zone: "${AWS_AZ:us-west-1a}"
    environment: "${ENVIRONMENT:development}"
    version: "${SERVICE_VERSION:v1.0.0}"
```

## 监控和指标

### 1. Prometheus 指标

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    serviceDiscoveryOperations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "service_discovery_operations_total",
            Help: "服务发现操作总数",
        },
        []string{"operation", "status"},
    )
    
    serviceInstances = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "service_discovery_instances",
            Help: "发现的服务实例数量",
        },
        []string{"service_name"},
    )
    
    healthCheckDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "service_discovery_health_check_duration_seconds",
            Help: "健康检查持续时间",
        },
        []string{"service_name"},
    )
)

func recordServiceDiscoveryMetrics(operation, status string) {
    serviceDiscoveryOperations.WithLabelValues(operation, status).Inc()
}

func updateServiceInstancesMetric(serviceName string, count int) {
    serviceInstances.WithLabelValues(serviceName).Set(float64(count))
}

func recordHealthCheckDuration(serviceName string, duration time.Duration) {
    healthCheckDuration.WithLabelValues(serviceName).Observe(duration.Seconds())
}
```

### 2. 分布式追踪

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func discoverServiceWithTracing(ctx context.Context, client *service_discovery_provider.Client, serviceName string) ([]*service_discovery_provider.ServiceInstance, error) {
    tracer := otel.Tracer("service-discovery")
    
    ctx, span := tracer.Start(ctx, "service.discover",
        trace.WithAttributes(
            attribute.String("service.name", serviceName),
        ),
    )
    defer span.End()
    
    services, err := client.Discover(serviceName)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetAttributes(
        attribute.Int("service.instances.count", len(services)),
    )
    
    return services, nil
}
```

## 错误处理和重试

### 1. 重试机制

```go
type RetryConfig struct {
    MaxRetries  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func discoverServiceWithRetry(client *service_discovery_provider.Client, serviceName string, config *RetryConfig) ([]*service_discovery_provider.ServiceInstance, error) {
    var lastErr error
    delay := config.InitialDelay
    
    for i := 0; i <= config.MaxRetries; i++ {
        services, err := client.Discover(serviceName)
        if err == nil {
            return services, nil
        }
        
        lastErr = err
        
        if i < config.MaxRetries {
            log.Printf("服务发现失败，%v 后重试 (%d/%d): %v", delay, i+1, config.MaxRetries, err)
            time.Sleep(delay)
            
            // 指数退避
            delay = time.Duration(float64(delay) * config.Multiplier)
            if delay > config.MaxDelay {
                delay = config.MaxDelay
            }
        }
    }
    
    return nil, fmt.Errorf("服务发现失败，已重试 %d 次: %v", config.MaxRetries, lastErr)
}
```

### 2. 断路器模式

```go
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    failures     int
    lastFailTime time.Time
    state        string // "closed", "open", "half-open"
    mutex        sync.RWMutex
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        maxFailures:  maxFailures,
        resetTimeout: resetTimeout,
        state:        "closed",
    }
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    // 检查是否可以从 open 状态转换到 half-open 状态
    if cb.state == "open" && time.Since(cb.lastFailTime) > cb.resetTimeout {
        cb.state = "half-open"
        cb.failures = 0
    }
    
    // 如果断路器是开启状态，直接返回错误
    if cb.state == "open" {
        return fmt.Errorf("断路器开启，服务不可用")
    }
    
    // 执行函数
    err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        // 如果失败次数超过阈值，开启断路器
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        
        return err
    }
    
    // 成功执行，重置状态
    cb.failures = 0
    cb.state = "closed"
    
    return nil
}

// 使用断路器进行服务发现
func discoverServiceWithCircuitBreaker(client *service_discovery_provider.Client, serviceName string, cb *CircuitBreaker) ([]*service_discovery_provider.ServiceInstance, error) {
    var services []*service_discovery_provider.ServiceInstance
    var err error
    
    cbErr := cb.Call(func() error {
        services, err = client.Discover(serviceName)
        return err
    })
    
    if cbErr != nil {
        return nil, cbErr
    }
    
    return services, err
}
```

## 最佳实践

### 1. 服务命名规范

```go
// 服务命名规范
const (
    ServiceNamePattern = "%s-%s-%s" // {domain}-{service}-{version}
)

func generateServiceName(domain, service, version string) string {
    return fmt.Sprintf(ServiceNamePattern, domain, service, version)
}

// 示例
func main() {
    serviceName := generateServiceName("user", "api", "v1")
    // 结果: user-api-v1
    
    config := &service_discovery_provider.Config{
        Service: &service_discovery_provider.ServiceConfig{
            Name:    serviceName,
            Version: "v1.0.0",
            Tags:    []string{"api", "user", "v1"},
        },
    }
}
```

### 2. 元数据管理

```go
func generateServiceMetadata() map[string]string {
    hostname, _ := os.Hostname()
    
    return map[string]string{
        "hostname":     hostname,
        "pid":          fmt.Sprintf("%d", os.Getpid()),
        "start_time":   time.Now().Format(time.RFC3339),
        "go_version":   runtime.Version(),
        "git_commit":   getGitCommit(),
        "build_time":   getBuildTime(),
        "environment":  getEnvironment(),
        "region":       getRegion(),
        "zone":         getZone(),
    }
}

func getGitCommit() string {
    // 从构建时注入的变量获取
    return "abc123"
}

func getBuildTime() string {
    // 从构建时注入的变量获取
    return time.Now().Format(time.RFC3339)
}

func getEnvironment() string {
    return os.Getenv("ENVIRONMENT")
}

func getRegion() string {
    return os.Getenv("AWS_REGION")
}

func getZone() string {
    return os.Getenv("AWS_AZ")
}
```

### 3. 优雅关闭

```go
func setupGracefulShutdown(client *service_discovery_provider.Client) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        log.Println("收到关闭信号，开始优雅关闭...")
        
        // 从注册中心注销服务
        if err := client.Deregister(); err != nil {
            log.Printf("注销服务失败: %v", err)
        } else {
            log.Println("服务注销成功")
        }
        
        // 等待一段时间让负载均衡器更新
        time.Sleep(5 * time.Second)
        
        // 关闭客户端
        client.Close()
        
        log.Println("优雅关闭完成")
        os.Exit(0)
    }()
}
```

服务发现提供者为微服务架构提供了完整的服务治理解决方案，支持多种注册中心和高级功能，确保服务的高可用性和可扩展性。