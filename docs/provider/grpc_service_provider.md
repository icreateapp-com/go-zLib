# gRPC 服务提供者 (gRPC Service Provider)

gRPC 服务提供者为应用程序提供高性能的 gRPC 服务端和客户端功能，支持服务注册、发现、负载均衡和中间件扩展。

## 功能特性

- **服务端支持**: 快速创建 gRPC 服务端
- **客户端连接**: 自动管理 gRPC 客户端连接
- **服务发现**: 集成服务发现机制
- **负载均衡**: 支持多种负载均衡策略
- **中间件支持**: 支持拦截器和中间件
- **健康检查**: 内置健康检查机制
- **连接池**: 自动管理连接池
- **TLS 支持**: 支持安全连接

## 快速开始

### 1. 服务端设置

```go
package main

import (
    "context"
    "log"
    
    "github.com/icreateapp-com/go-zLib/z/provider/grpc_service_provider"
    "google.golang.org/grpc"
    pb "your-project/proto"
)

// 实现 gRPC 服务
type UserService struct {
    pb.UnimplementedUserServiceServer
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // 获取用户信息
    user := getUserByID(req.UserId)
    
    return &pb.GetUserResponse{
        User: &pb.User{
            Id:    user.ID,
            Name:  user.Name,
            Email: user.Email,
        },
    }, nil
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    // 创建用户
    userID := createUser(req.User)
    
    return &pb.CreateUserResponse{
        UserId: userID,
    }, nil
}

func main() {
    // 创建 gRPC 服务器
    server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
        Port:        8080,
        ServiceName: "user-service",
        Version:     "v1.0.0",
    })
    
    // 注册服务
    userService := &UserService{}
    pb.RegisterUserServiceServer(server.GetGRPCServer(), userService)
    
    // 启动服务器
    if err := server.Start(); err != nil {
        log.Fatalf("启动 gRPC 服务器失败: %v", err)
    }
}
```

### 2. 客户端使用

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/icreateapp-com/go-zLib/z/provider/grpc_service_provider"
    pb "your-project/proto"
)

func main() {
    // 创建 gRPC 客户端
    client, err := grpc_service_provider.NewClient(&grpc_service_provider.ClientConfig{
        ServiceName: "user-service",
        Address:     "localhost:8080",
        Timeout:     30 * time.Second,
    })
    if err != nil {
        log.Fatalf("创建 gRPC 客户端失败: %v", err)
    }
    defer client.Close()
    
    // 创建服务客户端
    userClient := pb.NewUserServiceClient(client.GetConnection())
    
    // 调用服务
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // 获取用户
    getUserResp, err := userClient.GetUser(ctx, &pb.GetUserRequest{
        UserId: "12345",
    })
    if err != nil {
        log.Fatalf("获取用户失败: %v", err)
    }
    
    log.Printf("用户信息: %+v", getUserResp.User)
    
    // 创建用户
    createUserResp, err := userClient.CreateUser(ctx, &pb.CreateUserRequest{
        User: &pb.User{
            Name:  "张三",
            Email: "zhangsan@example.com",
        },
    })
    if err != nil {
        log.Fatalf("创建用户失败: %v", err)
    }
    
    log.Printf("创建用户成功，ID: %s", createUserResp.UserId)
}
```

## API 参考

### 服务端 API

#### NewServer(config *ServerConfig) *Server

创建新的 gRPC 服务器。

```go
type ServerConfig struct {
    Port            int               // 监听端口
    ServiceName     string            // 服务名称
    Version         string            // 服务版本
    MaxRecvMsgSize  int               // 最大接收消息大小
    MaxSendMsgSize  int               // 最大发送消息大小
    EnableReflection bool             // 启用反射
    TLSConfig       *TLSConfig        // TLS 配置
    Interceptors    []grpc.UnaryServerInterceptor  // 拦截器
    StreamInterceptors []grpc.StreamServerInterceptor // 流拦截器
}

server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
    Port:        8080,
    ServiceName: "user-service",
    Version:     "v1.0.0",
    EnableReflection: true,
})
```

#### Server.Start() error

启动 gRPC 服务器。

```go
if err := server.Start(); err != nil {
    log.Fatalf("启动服务器失败: %v", err)
}
```

#### Server.Stop()

优雅停止 gRPC 服务器。

```go
server.Stop()
```

#### Server.GetGRPCServer() *grpc.Server

获取底层的 gRPC 服务器实例。

```go
grpcServer := server.GetGRPCServer()
pb.RegisterUserServiceServer(grpcServer, userService)
```

### 客户端 API

#### NewClient(config *ClientConfig) (*Client, error)

创建新的 gRPC 客户端。

```go
type ClientConfig struct {
    ServiceName     string            // 服务名称
    Address         string            // 服务地址
    Timeout         time.Duration     // 连接超时
    MaxRecvMsgSize  int               // 最大接收消息大小
    MaxSendMsgSize  int               // 最大发送消息大小
    TLSConfig       *TLSConfig        // TLS 配置
    Interceptors    []grpc.UnaryClientInterceptor  // 拦截器
    StreamInterceptors []grpc.StreamClientInterceptor // 流拦截器
    LoadBalancer    string            // 负载均衡策略
}

client, err := grpc_service_provider.NewClient(&grpc_service_provider.ClientConfig{
    ServiceName: "user-service",
    Address:     "localhost:8080",
    Timeout:     30 * time.Second,
})
```

#### Client.GetConnection() *grpc.ClientConn

获取 gRPC 连接。

```go
conn := client.GetConnection()
userClient := pb.NewUserServiceClient(conn)
```

#### Client.Close() error

关闭客户端连接。

```go
defer client.Close()
```

## 中间件和拦截器

### 1. 服务端拦截器

```go
// 日志拦截器
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    
    log.Printf("gRPC 请求开始: %s", info.FullMethod)
    
    resp, err := handler(ctx, req)
    
    duration := time.Since(start)
    if err != nil {
        log.Printf("gRPC 请求失败: %s, 耗时: %v, 错误: %v", info.FullMethod, duration, err)
    } else {
        log.Printf("gRPC 请求成功: %s, 耗时: %v", info.FullMethod, duration)
    }
    
    return resp, err
}

// 认证拦截器
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // 跳过健康检查
    if info.FullMethod == "/grpc.health.v1.Health/Check" {
        return handler(ctx, req)
    }
    
    // 从元数据中获取认证信息
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Errorf(codes.Unauthenticated, "缺少元数据")
    }
    
    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Errorf(codes.Unauthenticated, "缺少认证令牌")
    }
    
    token := tokens[0]
    if !validateToken(token) {
        return nil, status.Errorf(codes.Unauthenticated, "无效的认证令牌")
    }
    
    // 将用户信息添加到上下文
    userID := getUserIDFromToken(token)
    ctx = context.WithValue(ctx, "user_id", userID)
    
    return handler(ctx, req)
}

// 限流拦截器
func rateLimitInterceptor(limiter *rate.Limiter) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if !limiter.Allow() {
            return nil, status.Errorf(codes.ResourceExhausted, "请求频率过高")
        }
        
        return handler(ctx, req)
    }
}

// 使用拦截器
func createServerWithInterceptors() *grpc_service_provider.Server {
    limiter := rate.NewLimiter(100, 200) // 每秒100个请求，突发200个
    
    server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
        Port:        8080,
        ServiceName: "user-service",
        Interceptors: []grpc.UnaryServerInterceptor{
            loggingInterceptor,
            authInterceptor,
            rateLimitInterceptor(limiter),
        },
    })
    
    return server
}
```

### 2. 客户端拦截器

```go
// 客户端日志拦截器
func clientLoggingInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
    start := time.Now()
    
    log.Printf("gRPC 客户端请求: %s", method)
    
    err := invoker(ctx, method, req, reply, cc, opts...)
    
    duration := time.Since(start)
    if err != nil {
        log.Printf("gRPC 客户端请求失败: %s, 耗时: %v, 错误: %v", method, duration, err)
    } else {
        log.Printf("gRPC 客户端请求成功: %s, 耗时: %v", method, duration)
    }
    
    return err
}

// 客户端认证拦截器
func clientAuthInterceptor(token string) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        // 添加认证头
        ctx = metadata.AppendToOutgoingContext(ctx, "authorization", token)
        
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

// 客户端重试拦截器
func clientRetryInterceptor(maxRetries int) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        var err error
        
        for i := 0; i <= maxRetries; i++ {
            err = invoker(ctx, method, req, reply, cc, opts...)
            
            if err == nil {
                return nil
            }
            
            // 检查是否应该重试
            if !shouldRetry(err) {
                break
            }
            
            if i < maxRetries {
                // 指数退避
                backoff := time.Duration(i+1) * time.Second
                time.Sleep(backoff)
                log.Printf("gRPC 请求重试 %d/%d: %s", i+1, maxRetries, method)
            }
        }
        
        return err
    }
}

func shouldRetry(err error) bool {
    st, ok := status.FromError(err)
    if !ok {
        return false
    }
    
    // 只对特定错误码重试
    switch st.Code() {
    case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
        return true
    default:
        return false
    }
}

// 使用客户端拦截器
func createClientWithInterceptors() (*grpc_service_provider.Client, error) {
    client, err := grpc_service_provider.NewClient(&grpc_service_provider.ClientConfig{
        ServiceName: "user-service",
        Address:     "localhost:8080",
        Interceptors: []grpc.UnaryClientInterceptor{
            clientLoggingInterceptor,
            clientAuthInterceptor("your-auth-token"),
            clientRetryInterceptor(3),
        },
    })
    
    return client, err
}
```

## 服务发现集成

### 1. 服务注册

```go
func createServerWithServiceDiscovery() *grpc_service_provider.Server {
    server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
        Port:        8080,
        ServiceName: "user-service",
        Version:     "v1.0.0",
        ServiceDiscovery: &grpc_service_provider.ServiceDiscoveryConfig{
            Enable:   true,
            Registry: "consul", // 或 "etcd"
            Address:  "localhost:8500",
            TTL:      30 * time.Second,
            Metadata: map[string]string{
                "region": "us-west-1",
                "zone":   "us-west-1a",
            },
        },
    })
    
    return server
}
```

### 2. 服务发现客户端

```go
func createClientWithServiceDiscovery() (*grpc_service_provider.Client, error) {
    client, err := grpc_service_provider.NewClient(&grpc_service_provider.ClientConfig{
        ServiceName: "user-service",
        ServiceDiscovery: &grpc_service_provider.ServiceDiscoveryConfig{
            Enable:   true,
            Registry: "consul",
            Address:  "localhost:8500",
        },
        LoadBalancer: "round_robin", // 或 "random", "consistent_hash"
    })
    
    return client, err
}
```

## 健康检查

### 1. 服务端健康检查

```go
import (
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

func setupHealthCheck(server *grpc_service_provider.Server) {
    // 创建健康检查服务
    healthServer := health.NewServer()
    
    // 注册健康检查服务
    grpc_health_v1.RegisterHealthServer(server.GetGRPCServer(), healthServer)
    
    // 设置服务状态
    healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_SERVING)
    healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
    
    // 监控服务状态
    go monitorServiceHealth(healthServer)
}

func monitorServiceHealth(healthServer *health.Server) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        // 检查数据库连接
        if !checkDatabaseHealth() {
            healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
            continue
        }
        
        // 检查外部依赖
        if !checkExternalDependencies() {
            healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
            continue
        }
        
        // 所有检查通过
        healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_SERVING)
    }
}
```

### 2. 客户端健康检查

```go
func checkServiceHealth(client *grpc_service_provider.Client) error {
    healthClient := grpc_health_v1.NewHealthClient(client.GetConnection())
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
        Service: "user-service",
    })
    if err != nil {
        return fmt.Errorf("健康检查失败: %v", err)
    }
    
    if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
        return fmt.Errorf("服务不健康: %v", resp.Status)
    }
    
    return nil
}
```

## TLS 安全配置

### 1. 服务端 TLS

```go
func createSecureServer() *grpc_service_provider.Server {
    server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
        Port:        8443,
        ServiceName: "user-service",
        TLSConfig: &grpc_service_provider.TLSConfig{
            CertFile: "server.crt",
            KeyFile:  "server.key",
            CAFile:   "ca.crt",
            ClientAuth: true, // 启用客户端认证
        },
    })
    
    return server
}
```

### 2. 客户端 TLS

```go
func createSecureClient() (*grpc_service_provider.Client, error) {
    client, err := grpc_service_provider.NewClient(&grpc_service_provider.ClientConfig{
        ServiceName: "user-service",
        Address:     "localhost:8443",
        TLSConfig: &grpc_service_provider.TLSConfig{
            CertFile:   "client.crt",
            KeyFile:    "client.key",
            CAFile:     "ca.crt",
            ServerName: "user-service",
        },
    })
    
    return client, err
}
```

## 流式处理

### 1. 服务端流

```go
func (s *UserService) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
    users := getUserList(req.PageSize, req.PageToken)
    
    for _, user := range users {
        if err := stream.Send(&pb.ListUsersResponse{
            User: user,
        }); err != nil {
            return err
        }
        
        // 模拟处理延迟
        time.Sleep(100 * time.Millisecond)
    }
    
    return nil
}
```

### 2. 客户端流

```go
func (s *UserService) CreateUsers(stream pb.UserService_CreateUsersServer) error {
    var userCount int32
    
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            // 客户端完成发送
            return stream.SendAndClose(&pb.CreateUsersResponse{
                UserCount: userCount,
            })
        }
        if err != nil {
            return err
        }
        
        // 处理用户创建
        createUser(req.User)
        userCount++
    }
}
```

### 3. 双向流

```go
func (s *UserService) ChatWithUsers(stream pb.UserService_ChatWithUsersServer) error {
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }
        
        // 处理消息
        response := processMessage(req.Message)
        
        // 发送响应
        if err := stream.Send(&pb.ChatResponse{
            Message: response,
        }); err != nil {
            return err
        }
    }
}
```

## 错误处理

### 1. 标准错误处理

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // 参数验证
    if req.UserId == "" {
        return nil, status.Errorf(codes.InvalidArgument, "用户ID不能为空")
    }
    
    // 获取用户
    user, err := getUserByID(req.UserId)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return nil, status.Errorf(codes.NotFound, "用户不存在: %s", req.UserId)
        }
        
        log.Printf("获取用户失败: %v", err)
        return nil, status.Errorf(codes.Internal, "内部服务器错误")
    }
    
    return &pb.GetUserResponse{
        User: convertToProtoUser(user),
    }, nil
}
```

### 2. 错误详情

```go
import (
    "google.golang.org/genproto/googleapis/rpc/errdetails"
    "google.golang.org/grpc/status"
)

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    // 验证请求
    if violations := validateCreateUserRequest(req); len(violations) > 0 {
        st := status.New(codes.InvalidArgument, "请求参数无效")
        
        // 添加详细错误信息
        br := &errdetails.BadRequest{}
        for _, violation := range violations {
            br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
                Field:       violation.Field,
                Description: violation.Description,
            })
        }
        
        st, _ = st.WithDetails(br)
        return nil, st.Err()
    }
    
    // 创建用户逻辑...
    return &pb.CreateUserResponse{}, nil
}

type FieldViolation struct {
    Field       string
    Description string
}

func validateCreateUserRequest(req *pb.CreateUserRequest) []FieldViolation {
    var violations []FieldViolation
    
    if req.User.Name == "" {
        violations = append(violations, FieldViolation{
            Field:       "user.name",
            Description: "用户名不能为空",
        })
    }
    
    if req.User.Email == "" {
        violations = append(violations, FieldViolation{
            Field:       "user.email",
            Description: "邮箱不能为空",
        })
    } else if !isValidEmail(req.User.Email) {
        violations = append(violations, FieldViolation{
            Field:       "user.email",
            Description: "邮箱格式无效",
        })
    }
    
    return violations
}
```

## 性能优化

### 1. 连接池配置

```go
func createOptimizedClient() (*grpc_service_provider.Client, error) {
    client, err := grpc_service_provider.NewClient(&grpc_service_provider.ClientConfig{
        ServiceName: "user-service",
        Address:     "localhost:8080",
        PoolConfig: &grpc_service_provider.PoolConfig{
            MaxConnections:  10,              // 最大连接数
            MaxIdleTime:     30 * time.Minute, // 最大空闲时间
            KeepAliveTime:   30 * time.Second, // 保活时间
            KeepAliveTimeout: 5 * time.Second, // 保活超时
        },
    })
    
    return client, err
}
```

### 2. 消息压缩

```go
import "google.golang.org/grpc/encoding/gzip"

func createCompressedServer() *grpc_service_provider.Server {
    server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
        Port:        8080,
        ServiceName: "user-service",
        Compression: gzip.Name, // 启用 gzip 压缩
    })
    
    return server
}
```

### 3. 批量处理

```go
func (s *UserService) BatchGetUsers(ctx context.Context, req *pb.BatchGetUsersRequest) (*pb.BatchGetUsersResponse, error) {
    // 并发获取用户信息
    userChan := make(chan *pb.User, len(req.UserIds))
    errChan := make(chan error, len(req.UserIds))
    
    var wg sync.WaitGroup
    for _, userID := range req.UserIds {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            
            user, err := getUserByID(id)
            if err != nil {
                errChan <- err
                return
            }
            
            userChan <- convertToProtoUser(user)
        }(userID)
    }
    
    wg.Wait()
    close(userChan)
    close(errChan)
    
    // 收集结果
    var users []*pb.User
    for user := range userChan {
        users = append(users, user)
    }
    
    // 检查错误
    for err := range errChan {
        if err != nil {
            log.Printf("批量获取用户时发生错误: %v", err)
        }
    }
    
    return &pb.BatchGetUsersResponse{
        Users: users,
    }, nil
}
```

## 监控和指标

### 1. Prometheus 指标

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    grpcRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "grpc_requests_total",
            Help: "gRPC 请求总数",
        },
        []string{"method", "status"},
    )
    
    grpcRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "grpc_request_duration_seconds",
            Help: "gRPC 请求持续时间",
        },
        []string{"method"},
    )
)

func metricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    start := time.Now()
    
    resp, err := handler(ctx, req)
    
    duration := time.Since(start)
    method := info.FullMethod
    status := "success"
    if err != nil {
        status = "error"
    }
    
    grpcRequestsTotal.WithLabelValues(method, status).Inc()
    grpcRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
    
    return resp, err
}
```

### 2. 分布式追踪

```go
import (
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func createTracedServer() *grpc_service_provider.Server {
    server := grpc_service_provider.NewServer(&grpc_service_provider.ServerConfig{
        Port:        8080,
        ServiceName: "user-service",
        Interceptors: []grpc.UnaryServerInterceptor{
            otelgrpc.UnaryServerInterceptor(),
        },
        StreamInterceptors: []grpc.StreamServerInterceptor{
            otelgrpc.StreamServerInterceptor(),
        },
    })
    
    return server
}
```

## 最佳实践

### 1. 服务定义

```protobuf
syntax = "proto3";

package user.v1;

option go_package = "your-project/proto/user/v1";

// 用户服务
service UserService {
    // 获取用户信息
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    
    // 创建用户
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
    
    // 批量获取用户
    rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);
    
    // 列出用户（服务端流）
    rpc ListUsers(ListUsersRequest) returns (stream ListUsersResponse);
}

message User {
    string id = 1;
    string name = 2;
    string email = 3;
    int64 created_at = 4;
    int64 updated_at = 5;
}

message GetUserRequest {
    string user_id = 1;
}

message GetUserResponse {
    User user = 1;
}
```

### 2. 错误码定义

```go
const (
    // 用户相关错误码
    CodeUserNotFound     = "USER_NOT_FOUND"
    CodeUserExists       = "USER_EXISTS"
    CodeInvalidUserData  = "INVALID_USER_DATA"
    
    // 认证相关错误码
    CodeUnauthorized     = "UNAUTHORIZED"
    CodeForbidden        = "FORBIDDEN"
    CodeTokenExpired     = "TOKEN_EXPIRED"
    
    // 系统相关错误码
    CodeInternalError    = "INTERNAL_ERROR"
    CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

func NewUserNotFoundError(userID string) error {
    return status.Errorf(codes.NotFound, "用户不存在: %s", userID)
}

func NewInvalidUserDataError(field string, reason string) error {
    return status.Errorf(codes.InvalidArgument, "无效的用户数据 %s: %s", field, reason)
}
```

### 3. 配置管理

```go
type GRPCConfig struct {
    Server ServerConfig `yaml:"server"`
    Client ClientConfig `yaml:"client"`
}

type ServerConfig struct {
    Port            int           `yaml:"port"`
    MaxRecvMsgSize  int           `yaml:"max_recv_msg_size"`
    MaxSendMsgSize  int           `yaml:"max_send_msg_size"`
    ReadTimeout     time.Duration `yaml:"read_timeout"`
    WriteTimeout    time.Duration `yaml:"write_timeout"`
    EnableReflection bool         `yaml:"enable_reflection"`
}

type ClientConfig struct {
    Timeout         time.Duration `yaml:"timeout"`
    MaxRecvMsgSize  int           `yaml:"max_recv_msg_size"`
    MaxSendMsgSize  int           `yaml:"max_send_msg_size"`
    KeepAliveTime   time.Duration `yaml:"keep_alive_time"`
    KeepAliveTimeout time.Duration `yaml:"keep_alive_timeout"`
}

// 配置文件示例 (config.yaml)
/*
grpc:
  server:
    port: 8080
    max_recv_msg_size: 4194304  # 4MB
    max_send_msg_size: 4194304  # 4MB
    read_timeout: 30s
    write_timeout: 30s
    enable_reflection: true
  client:
    timeout: 30s
    max_recv_msg_size: 4194304  # 4MB
    max_send_msg_size: 4194304  # 4MB
    keep_alive_time: 30s
    keep_alive_timeout: 5s
*/
```

gRPC 服务提供者为应用程序提供了完整的 gRPC 服务端和客户端解决方案，支持高性能、高可用的微服务架构。