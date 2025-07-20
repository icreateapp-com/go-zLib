# 错误跟踪器 (Error Tracker)

go-zLib 提供了一个强大的错误跟踪系统，能够自动记录错误调用链、支持请求级别的错误管理，并提供详细的堆栈跟踪信息。

## 概述

错误跟踪器的核心功能：
- **自动错误链跟踪**：记录错误的完整调用链路
- **请求级别错误管理**：将同一请求中的多个错误关联起来
- **智能错误合并**：自动合并同一请求的错误为完整调用链
- **中间件自动处理**：HTTP/gRPC 中间件自动处理错误记录
- **详细堆栈跟踪**：提供文件、行号、函数名等详细信息

## 核心数据结构

### TraceData
```go
type TraceData struct {
    No       int       // 编号
    Type     string    // 类型：ERROR
    Time     time.Time // 时间
    File     string    // 文件：项目相对路径
    Line     int       // 行号
    Message  string    // 错误消息
    Function string    // 函数名
}
```

### TrackedError
```go
type TrackedError struct {
    OriginalError error
    TraceID       string
    Traces        []TraceData
}
```

## 基本使用

### 1. 错误包装和跟踪

```go
import "github.com/icreateapp-com/go-zLib/z"

func businessLogic() error {
    // 原始错误
    err := errors.New("数据库连接失败")
    
    // 使用 Tracker.Error 包装错误，自动记录调用信息
    return z.Tracker.Error(err)
}

func controller() error {
    if err := businessLogic(); err != nil {
        // 继续包装错误，形成调用链
        return z.Tracker.Error(fmt.Errorf("业务处理失败: %w", err))
    }
    return nil
}
```

### 2. 格式化错误创建

```go
func validateUser(userID string, username string) error {
    if userID == "" {
        // 使用 Tracker.Errorf 直接创建格式化错误
        return z.Tracker.Errorf("用户ID不能为空，用户名: %s", username)
    }
    
    if len(userID) < 3 {
        // 支持多个参数的格式化
        return z.Tracker.Errorf("用户ID长度不足，当前长度: %d，最小长度: %d", len(userID), 3)
    }
    
    return nil
}

func processUser(userID string) error {
    if err := validateUser(userID, "testuser"); err != nil {
        // 使用 Errorf 包装其他错误
        return z.Tracker.Errorf("用户处理失败，ID: %s, 错误: %w", userID, err)
    }
    return nil
}
```

### 3. 手动记录错误到日志

```go
func handleError() {
    if err := controller(); err != nil {
        // 记录单个错误到日志
        z.Tracker.LogError(err)
    }
}
```

### 4. 记录所有错误

```go
func cleanup() {
    // 记录所有追踪的错误到日志
    z.Tracker.LogAllTraces()
    
    // 清空所有追踪数据
    z.Tracker.Clear()
}
```

## 请求级别错误管理

### 设置请求ID

```go
func handleRequest() {
    // 设置当前请求ID
    requestID := "req-12345"
    z.Tracker.SetRequestID(requestID)
    
    // 业务逻辑中的错误会自动关联到这个请求ID
    if err := businessLogic(); err != nil {
        // 错误会自动标记到当前请求
        z.Tracker.Error(err)
    }
    
    // 检查请求是否有错误
    if z.Tracker.HasRequestErrors(requestID) {
        // 记录请求的所有错误（合并为一个完整调用链）
        z.Tracker.LogRequestErrors(requestID)
    }
    
    // 清理请求错误记录
    z.Tracker.ClearRequestErrors(requestID)
    z.Tracker.SetRequestID("")
}
```

## HTTP 中间件集成

### 基本设置

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/server/http_server/http_middleware"
)

func setupRouter() *gin.Engine {
    r := gin.New()
    
    // 添加错误跟踪中间件
    r.Use(http_middleware.ErrorLogMiddleware())      // 请求级别错误管理
    r.Use(http_middleware.ErrorTrackerMiddleware())  // Panic 恢复和错误记录
    
    return r
}
```

### 在处理器中使用

```go
func paymentHandler(c *gin.Context) {
    // 获取请求ID（由中间件自动设置）
    requestID, _ := c.Get("request_id")
    
    // 业务逻辑中的错误会自动关联到请求
    if err := validatePayment(); err != nil {
        // 错误会自动记录到当前请求
        z.Tracker.Error(err)
        z.Failure(c, "验证失败", 40001)
        return
    }
    
    if err := processPayment(); err != nil {
        z.Tracker.Error(err)
        z.Failure(c, "处理失败", 50001)
        return
    }
    
    z.Success(c, "支付成功", nil)
    // 中间件会自动处理错误记录和清理
}
```

## gRPC 中间件集成

```go
import (
    "github.com/icreateapp-com/go-zLib/z/server/grpc_server/grpc_middleware"
    "google.golang.org/grpc"
)

func setupGRPCServer() *grpc.Server {
    s := grpc.NewServer(
        grpc.UnaryInterceptor(grpc_middleware.ErrorTrackerMiddleware),
    )
    return s
}
```

## API 参考

### 核心方法

#### `Tracker.Error(err error) error`
包装错误并记录调用信息，返回 TrackedError。

#### `Tracker.Errorf(format string, args ...interface{}) error`
格式化错误包装器，支持 fmt.Errorf 的格式化形式，内部调用 Error 方法进行错误跟踪。

#### `Tracker.LogError(err error)`
将单个错误记录到日志文件。

#### `Tracker.LogAllTraces()`
将所有追踪的错误记录到日志文件。

#### `Tracker.Clear()`
清空所有追踪数据。

### 请求级别方法

#### `Tracker.SetRequestID(requestID string)`
设置当前请求ID，后续的错误会关联到此请求。

#### `Tracker.HasRequestErrors(requestID string) bool`
检查指定请求是否有错误。

#### `Tracker.LogRequestErrors(requestID string)`
记录指定请求的所有错误，自动合并为完整调用链。

#### `Tracker.ClearRequestErrors(requestID string)`
清理指定请求的错误记录。

### 数据获取方法

#### `Tracker.GetData() []TraceData`
获取所有追踪数据。

#### `Tracker.GetTracesByID(traceID string) []TraceData`
根据 traceID 获取特定的追踪数据。

#### `Tracker.RecoverAndLog()`
恢复 panic 并记录错误（通常在 defer 中使用）。

## 日志格式

### 单个错误日志格式
```
/src/controller.go:25: 业务处理失败: 数据库连接失败
[stacktrace]
#0 /src/controller.go:25 controller
#1 /src/service.go:15 businessLogic
{main}
```

### 请求级别错误日志格式
```
Request: req-12345, TraceID: trace_1752979034577341000_1
/src/controller.go:25: 业务处理失败: 数据库连接失败
[stacktrace]
#0 /src/controller.go:25 controller
#1 /src/service.go:15 businessLogic
#2 /src/validator.go:10 validatePayment
{main}
```

## 最佳实践

### 1. 业务逻辑中的错误处理
```go
func validateUser(userID string) error {
    if userID == "" {
        // 使用 Errorf 直接创建格式化错误
        return z.Tracker.Errorf("用户ID不能为空")
    }
    return nil
}

func getUserBalance(userID string) (float64, error) {
    if err := validateUser(userID); err != nil {
        // 使用 Errorf 包装错误，形成调用链
        return 0, z.Tracker.Errorf("获取余额失败，用户ID: %s, 错误: %w", userID, err)
    }
    // ... 业务逻辑
    return 100.0, nil
}

// 也可以使用传统的 Error 方法
func validateUserTraditional(userID string) error {
    if userID == "" {
        err := errors.New("用户ID不能为空")
        return z.Tracker.Error(err)  // 自动记录调用信息
    }
    return nil
}
```

### 2. 中间件自动处理
推荐使用中间件自动处理错误记录，无需手动调用日志方法：

```go
// HTTP 中间件会自动：
// 1. 生成请求ID
// 2. 设置到错误跟踪器
// 3. 处理请求结束后检查错误
// 4. 自动记录和清理错误

r.Use(http_middleware.ErrorLogMiddleware())
r.Use(http_middleware.ErrorTrackerMiddleware())
```

### 3. 错误链的构建
```go
func processOrder(orderID string, userID string) error {
    // 第一层错误 - 使用 Errorf 创建格式化错误
    if err := validateOrder(orderID); err != nil {
        return z.Tracker.Errorf("订单验证失败，订单ID: %s, 错误: %w", orderID, err)
    }
    
    // 第二层错误 - 继续使用 Errorf 包装
    if err := calculatePrice(orderID); err != nil {
        return z.Tracker.Errorf("价格计算失败，订单ID: %s, 用户ID: %s, 错误: %w", orderID, userID, err)
    }
    
    // 第三层错误 - 混合使用 Error 和 Errorf
    if err := saveOrder(orderID); err != nil {
        // 可以根据需要选择使用 Error 或 Errorf
        return z.Tracker.Errorf("订单保存失败，订单ID: %s, 错误: %w", orderID, err)
    }
    
    return nil
}

// 传统方式（仍然支持）
func processOrderTraditional(orderID string) error {
    if err := validateOrder(orderID); err != nil {
        return z.Tracker.Error(err)
    }
    
    if err := calculatePrice(orderID); err != nil {
        return z.Tracker.Error(fmt.Errorf("价格计算失败: %w", err))
    }
    
    if err := saveOrder(orderID); err != nil {
        return z.Tracker.Error(fmt.Errorf("订单保存失败: %w", err))
    }
    
    return nil
}
```

## 注意事项

1. **线程安全**：错误跟踪器是线程安全的，可以在并发环境中使用。

2. **内存管理**：定期调用 `Clear()` 或 `ClearRequestErrors()` 清理数据，避免内存泄漏。

3. **性能考虑**：错误跟踪会增加少量性能开销，主要用于开发和调试环境。

4. **日志文件**：错误会记录到 `storage/log/debug.log` 文件中。

5. **请求ID**：在使用请求级别功能时，确保正确设置和清理请求ID。

## 配置

错误跟踪器依赖于 go-zLib 的日志系统，确保正确初始化：

```go
import "github.com/icreateapp-com/go-zLib/z"

func init() {
    // 日志系统会自动初始化
    // 错误跟踪器会自动使用配置的日志路径
}
```