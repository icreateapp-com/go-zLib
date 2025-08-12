package grpc_service_provider

import (
	"fmt"
	"net"
	"sync"

	. "github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
)

// grpcServiceProvider gRPC服务提供者
type grpcServiceProvider struct {
	server *grpc.Server
	mutex  sync.RWMutex // 读写锁保护server字段
}

// GrpcServiceProvider 全局gRPC服务提供者实例
var GrpcServiceProvider = &grpcServiceProvider{}

// Register 注册gRPC服务并启动服务器
func (s *grpcServiceProvider) Register(services func(server *grpc.Server)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 创建gRPC服务器
	s.server = grpc.NewServer()

	// 获取地址配置
	address := "0.0.0.0:7000"
	host, _ := Config.String("config.grpc.host")
	port, _ := Config.Int("config.grpc.port")

	if len(host) > 0 && port > 0 {
		address = fmt.Sprintf("%s:%d", host, port)
	}

	// 注册服务
	services(s.server)

	// 启动服务器
	go func() {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			Error.Printf("failed to listen: %v", err)
			return
		}

		Info.Printf("[GRPC] server listening at %v", address)

		if err := s.server.Serve(listener); err != nil {
			Error.Printf("failed to serve: %v", err)
			return
		}
	}()
}
