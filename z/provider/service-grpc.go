package provider

import (
	"fmt"
	"net"

	. "github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
)

type serviceGrpcProvider struct {
	server *grpc.Server
}

var ServiceGrpcProvider serviceGrpcProvider

// Register 服务注册
// 该方法用于注册gRPC服务并启动gRPC服务器。
// 使用示例：
//
//	provider.ServiceGrpcProvider.Register(func(server *grpc.Server) {
//	    pb.RegisterYourServiceServer(server, &yourService{})
//	})
//
// 配置文件例子：
// config:
//
//	grpc:
//	  host: "0.0.0.0"
//	  port: 7000
func (s *serviceGrpcProvider) Register(services func(server *grpc.Server)) {

	// create grpc server
	s.server = grpc.NewServer()

	// get address config
	address := "0.0.0.0:7000"
	host, _ := Config.String("config.grpc.host")
	port, _ := Config.Int("config.grpc.port")

	if len(host) > 0 && port > 0 {
		address = fmt.Sprintf("%s:%d", host, port)
	}

	// register services
	services(s.server)

	// start server
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
