# 服务提供者模块

go-zLib 的服务提供者模块主要用于服务注册、服务发现和配置中心集成，使微服务能够自动注册到服务中心，并获取动态配置。

## 目录
- [配置中心提供者](#配置中心提供者)
- [服务发现提供者](#服务发现提供者)
- [gRPC 服务](#grpc-服务)

## 配置中心提供者

配置中心提供者用于连接外部的配置中心服务，同步配置信息并注册服务。

### 使用方法

```go
import (
    "github.com/icreateapp-com/go-zLib/z/provider"
)

func init() {
    // 注册到配置中心
    provider.ConfigCenterProvider.Register()
}
```

### 配置

配置中心提供者需要在配置文件中设置以下参数：

```yaml
config:
  config_center:
    address: "http://config-center.example.com" # 配置中心地址
    apikey: "your-api-key"                      # 访问密钥
    sync: true                                  # 是否同步配置
    sync_interval: 60                           # 同步间隔(秒)
name: "your-service-name"                      # 服务名称
```

### 功能说明

配置中心提供者提供以下功能：

1. **服务注册**：将服务信息注册到配置中心
2. **配置同步**：定期从配置中心获取最新配置
3. **本地缓存**：配置信息会缓存在本地，避免频繁请求

### 参数与方法

#### Register 方法

注册服务并启动配置同步。

**参数**：无

**返回值**：无

#### GetConfig 方法

获取指定路径的配置值。

**参数**：
- key (string): 配置键路径

**返回值**：
- (interface{}, error): 配置值和可能的错误

```go
value, err := provider.ConfigCenterProvider.GetConfig("database.host")
if err != nil {
    // 处理错误
}
```

## 服务发现提供者

服务发现提供者用于服务注册与发现，支持服务健康检查和服务调用。

### 使用方法

```go
import (
    "github.com/icreateapp-com/go-zLib/z/provider"
)

func init() {
    // 注册服务
    provider.ServiceDiscoverProvider.Register()
}

func callService() {
    // 调用其他服务
    var response interface{}
    err := provider.ServiceDiscoverProvider.Call(
        "user-service",
        provider.ServiceRequestParam{
            Path:   "/api/users",
            Method: "GET",
            Query:  map[string]interface{}{"page": 1},
        },
        &response,
    )
    if err != nil {
        // 处理错误
    }
}
```

### 配置

服务发现提供者需要在配置文件中设置以下参数：

```yaml
config:
  service_discover:
    address: "http://service-discover.example.com" # 服务发现地址
    apikey: "your-api-key"                        # 访问密钥
    cache_ttl: 300                                # 缓存过期时间(秒)
    check: true                                   # 是否启用健康检查
    check_interval: 5                             # 健康检查间隔(秒)
    lost_timeout: 10                              # 服务丢失超时(秒)
name: "your-service-name"                         # 服务名称
port: 8080                                        # 服务端口
host: "auto"                                      # 服务主机(auto表示自动获取)
grpc:
  host: "auto"                                    # gRPC 主机
  port: 9090                                      # gRPC 端口
auth:                                             # 访问令牌
  token1: "value1"
  token2: "value2"
```

### 功能说明

服务发现提供者提供以下功能：

1. **服务注册**：自动将当前服务注册到服务发现中心
2. **健康检查**：定期发送健康检查请求，确保服务可用
3. **服务发现**：查询和调用其他已注册的服务
4. **负载均衡**：自动选择最佳的服务实例进行调用
5. **服务缓存**：缓存服务地址信息，提高性能

### 参数与方法

#### Register 方法

注册服务并启动健康检查。

**参数**：无

**返回值**：无

#### Call 方法

调用指定服务的 HTTP 接口。

**参数**：
- name (string): 服务名称
- request (ServiceRequestParam): 请求参数
- response (*interface{}): 响应结果指针

**返回值**：
- error: 可能的错误

```go
var response interface{}
err := provider.ServiceDiscoverProvider.Call(
    "user-service",
    provider.ServiceRequestParam{
        Path:    "/api/users",
        Method:  "GET",
        Query:   map[string]interface{}{"page": 1},
        Data:    nil,
        Headers: map[string]string{"X-Custom-Header": "value"},
    },
    &response,
)
```

#### Grpc 方法

调用指定服务的 gRPC 接口。

**参数**：
- name (string): 服务名称
- handler (func(*ServiceDiscoverServiceInfo, *grpc.ClientConn) error): 处理函数

**返回值**：
- error: 可能的错误

```go
err := provider.ServiceDiscoverProvider.Grpc(
    "user-service",
    func(service *provider.ServiceDiscoverServiceInfo, conn *grpc.ClientConn) error {
        client := pb.NewUserServiceClient(conn)
        // 使用客户端调用 gRPC 方法
        return nil
    },
)
```

#### GetAllServiceAddress 方法

获取指定服务的所有实例地址。

**参数**：
- name (string): 服务名称

**返回值**：
- (*[]ServiceDiscoverServiceInfo, error): 服务信息列表和可能的错误

```go
services, err := provider.ServiceDiscoverProvider.GetAllServiceAddress("user-service")
if err != nil {
    // 处理错误
}
```

#### GetBestServiceAddress 方法

获取指定服务的最佳实例地址（延迟最低）。

**参数**：
- name (string): 服务名称

**返回值**：
- (*ServiceDiscoverServiceInfo, error): 服务信息和可能的错误

```go
service, err := provider.ServiceDiscoverProvider.GetBestServiceAddress("user-service")
if err != nil {
    // 处理错误
}
```

### ServiceRequestParam 结构

请求参数结构，用于 Call 方法。

| 字段 | 类型 | 说明 |
|------|------|------|
| Path | string | 请求路径 |
| Method | string | 请求方法 (GET, POST, PUT, DELETE) |
| Query | map[string]interface{} | 查询参数 |
| Data | map[string]interface{} | 请求体数据 |
| Headers | map[string]string | 请求头 |

### ServiceDiscoverServiceInfo 结构

服务信息结构。

| 字段 | 类型 | 说明 |
|------|------|------|
| Name | string | 服务名称 |
| Host | string | 服务主机地址 |
| Port | int | 服务端口 |
| GrpcHost | string | gRPC 主机地址 |
| GrpcPort | int | gRPC 端口 |
| Latency | int | 响应延迟(毫秒) |
| AuthToken | map[string]string | 访问令牌 |

## gRPC 服务

go-zLib 中的 gRPC 服务提供者用于简化 gRPC 服务的注册和调用。

### 使用方法

```go
import (
    "github.com/icreateapp-com/go-zLib/z/provider"
    "google.golang.org/grpc"
)

// 注册 gRPC 服务
func RegisterGrpcServer() {
    // 创建 gRPC 服务器
    server := grpc.NewServer()
    
    // 注册服务
    pb.RegisterUserServiceServer(server, &UserService{})
    
    // 启动 gRPC 服务器
    provider.GrpcServiceProvider.Start(server)
}
```

### 配置

gRPC 服务提供者需要在配置文件中设置以下参数：

```yaml
config:
  grpc:
    host: "0.0.0.0"  # gRPC 服务监听主机
    port: 9090       # gRPC 服务监听端口
```

### 功能说明

gRPC 服务提供者提供以下功能：

1. **服务启动**：启动 gRPC 服务并监听指定端口
2. **服务注册**：自动将 gRPC 服务注册到服务发现中心
3. **连接管理**：管理与其他 gRPC 服务的连接

### 参数与方法

#### Start 方法

启动 gRPC 服务器。

**参数**：
- server (*grpc.Server): gRPC 服务器实例

**返回值**：
- error: 可能的错误

```go
err := provider.GrpcServiceProvider.Start(server)
if err != nil {
    // 处理错误
}
``` 