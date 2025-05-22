# gRPC 服务器

本文档描述了 go-zLib 库中 gRPC 服务器的实现。

## 概述

go-zLib 中的 gRPC 服务器实现允许您：

1. 创建纯 gRPC 服务器并通过 gRPC 协议暴露您的服务
2. 通过简单的配置设置服务器参数
3. 添加自定义中间件以增强 gRPC 服务器功能
4. 支持同步和异步启动服务器

## 使用方法

### 基本用法

要使用 gRPC 服务器，您需要：

1. 使用 protobuf 定义您的服务
2. 使用 protoc 或 buf 生成代码
3. 实现您的 gRPC 服务方法
4. 使用 `ServeGrpc` 或 `MustServeGrpc` 函数启动服务器

以下是一个基本示例：

```go
package main

import (
	"github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
)

func main() {
	// 使用默认配置或根据需要自定义
	config := z.DefaultGrpcConfig()
	
	// 启动 gRPC 服务器
	z.MustServeGrpc(
		// 注册 gRPC 服务
		func(server *grpc.Server) {
			// 在此注册您的服务实现
			// 示例: pb.RegisterYourServiceServer(server, &YourServiceImpl{})
		},
		// 使用配置
		config,
		// 可选：添加中间件
		grpc.UnaryInterceptor(YourInterceptor),
	)
}
```

### 配置选项

gRPC 服务器可以使用 `GrpcServerConfig` 结构进行配置：

```go
type GrpcServerConfig struct {
	Host          string // gRPC 服务器主机地址
	Port          int    // gRPC 服务器端口
	EnableReflect bool   // 是否启用 gRPC 反射服务
}
```

`DefaultGrpcConfig()` 函数提供了合理的默认值：

```go
func DefaultGrpcConfig() GrpcServerConfig {
	return GrpcServerConfig{
		Host:          "localhost",
		Port:          50051,
		EnableReflect: true,
	}
}
```

### 添加中间件

您可以向 gRPC 服务器添加中间件（拦截器）：

```go
// 创建一个简单的拦截器
func LogInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 请求前的处理
	log.Printf("请求: %s", info.FullMethod)
	
	// 调用处理程序
	resp, err := handler(ctx, req)
	
	// 请求后的处理
	log.Printf("响应: %v", resp)
	
	return resp, err
}

// 使用拦截器
z.ServeGrpc(
	registerFunc,
	config,
	grpc.UnaryInterceptor(LogInterceptor),
)
```

### 异步启动服务器

如果您需要在同一个应用程序中运行其他服务或需要控制服务器的生命周期，可以使用 `ServeGrpcAsync` 函数：

```go
// 异步启动 gRPC 服务器
stopServer, errChan := z.ServeGrpcAsync(
	func(server *grpc.Server) {
		// 注册服务
		pb.RegisterYourServiceServer(server, &YourServiceImpl{})
	},
	z.DefaultGrpcConfig(),
)

// 监听错误
go func() {
	if err := <-errChan; err != nil {
		log.Printf("gRPC 服务器错误: %v", err)
	}
}()

// 在适当的时候停止服务器
// 例如，在接收到终止信号时
signals := make(chan os.Signal, 1)
signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
<-signals
stopServer()
```

## Protobuf 服务定义

要定义 gRPC 服务，您需要创建 protobuf 文件：

```protobuf
syntax = "proto3";

package example;

option go_package = "github.com/icreateapp-com/go-zLib/example";

service YourService {
  rpc YourMethod(YourRequest) returns (YourResponse);
}

message YourRequest {
  string input = 1;
}

message YourResponse {
  string output = 1;
}
```

## 代码生成

您需要从 protobuf 定义生成 Go 代码。推荐使用 [buf](https://buf.build/)，但您也可以直接使用 protoc：

```bash
# 使用 protoc
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  your_service.proto

# 或使用 buf
buf generate
``` 