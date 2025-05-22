package grpc_middleware

import (
	"context"
	"fmt"
	"net"

	"github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc"
)

// TrustedProxiesMiddleware 是一个 gRPC 服务器拦截器，用于限制只有指定 IP 列表中的客户端可以访问服务。
// 支持通配符 * 匹配，例如：192.168.*.*
// 本地 IP 地址（如 ::1、0.0.0.0、127.0.0.1 等）会被自动允许访问
func TrustedProxiesMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 获取白名单列表
	whitelist := z.Config.GetStringSlice("config.grpc.trusted_proxies")

	// 如果有值才去匹配
	if len(whitelist) > 0 {
		// 获取客户端 IP 地址
		peerAddr := ""
		if p, ok := peer.FromContext(ctx); ok {
			peerAddr = p.Addr.String()
		}

		// 解析客户端 IP 地址
		ip, _, err := net.SplitHostPort(peerAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client address: %w", err)
		}

		// 检查是否为本地 IP 地址，如果是则允许访问
		if z.IsLocalIP(ip) {
			// 本地 IP 地址直接允许访问
			goto HANDLE
		}

		// 检查客户端 IP 是否在白名单中
		matched := false
		for _, allowedIP := range whitelist {
			if z.MatchIP(ip, allowedIP) {
				matched = true
				break
			}
		}

		if !matched {
			return nil, fmt.Errorf("client IP %s is not in the whitelist", ip)
		}
	}

HANDLE:

	// 调用实际的处理函数
	resp, err := handler(ctx, req)
	return resp, err
}
