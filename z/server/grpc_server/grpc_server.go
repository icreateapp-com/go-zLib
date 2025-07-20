package grpc_server

import (
	"fmt"
	"net"
	"time"

	. "github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/server/grpc_server/grpc_middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GrpcServer 启动一个 gRPC 服务器
// 参数:
//   - registerServices: 用于注册 gRPC 服务的函数
//   - interceptors: 一个或多个 gRPC 拦截器函数
func GrpcServer(registerServices func(*grpc.Server), interceptors ...grpc.UnaryServerInterceptor) error {
	///////////////////////////////////////////////
	// 初始化系统
	///////////////////////////////////////////////

	// 设置时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST-8", 8*3600)
	}
	time.Local = loc

	// 加载配置
	if err := Config.LoadFile(BasePath(), "config.yml"); err != nil {
		Error.Fatalln(err.Error())
	}

	// 初始化日志系统
	debug, _ := Config.Bool("config.debug")
	Log.Init(true, debug)

	// 初始化内存缓存
	MemCache.Init(60*time.Minute, 10*time.Minute)

	///////////////////////////////////////////////
	// 启动 gRPC 服务器
	///////////////////////////////////////////////

	// 构建 gRPC 服务器地址
	host := Config.GetString("config.grpc.host")
	port := Config.GetInt("config.grpc.port")
	addr := fmt.Sprintf("%s:%d", host, port)

	// 创建监听器
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		Error.Fatalln(fmt.Sprintf("can not bind address %s", addr))
	}

	// 创建服务器选项
	var serverOptions []grpc.ServerOption

	// 添加默认的错误跟踪中间件
	defaultInterceptors := []grpc.UnaryServerInterceptor{
		grpc_middleware.RecoveryMiddleware,     // 恢复中间件
		grpc_middleware.ErrorTrackerMiddleware, // 错误跟踪中间件
	}

	// 合并用户提供的拦截器
	allInterceptors := append(defaultInterceptors, interceptors...)

	// 使用 grpc 自带的 ChainUnaryInterceptor 来链接多个拦截器
	serverOptions = append(serverOptions, grpc.ChainUnaryInterceptor(allInterceptors...))

	// 创建 gRPC 服务器实例
	grpcServer := grpc.NewServer(serverOptions...)

	// 注册 gRPC 服务
	registerServices(grpcServer)

	// 如果启用，则注册反射服务
	if Config.GetBool("config.grpc.enable_reflect") {
		reflection.Register(grpcServer)
	}

	Info.Printf("grpc server running at %s\n", addr)

	// 启动 gRPC 服务器并阻塞当前 goroutine
	return grpcServer.Serve(lis)
}
