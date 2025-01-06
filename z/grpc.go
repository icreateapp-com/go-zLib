package z

import (
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
)

// 定义一个结构体，用于存储grpc实例
type _grpc struct {
	grpcIns *grpc.Server
}

// Grpc 定义一个全局变量，用于存储grpc实例
var Grpc _grpc

// Init 初始化grpc实例
func (p *_grpc) Init(opts ...cron.Option) {
	p.grpcIns = grpc.NewServer()
}

// Register 注册grpc服务
func (p *_grpc) Register() {

}
