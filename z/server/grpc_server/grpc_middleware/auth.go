package grpc_middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// AuthMiddleware 是一个 gRPC 服务器拦截器，用于验证请求的认证信息。
// 它从 gRPC metadata 中获取 Authorization 头，并根据配置的 auth 规则进行验证。
// 支持匿名访问路径配置和基于路径前缀的 token 验证。
func AuthMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 获取匿名访问路径列表
	if skips, err := z.Config.StringSlice("config.anonymity"); err == nil {
		// 检查是否为根路径
		if info.FullMethod == "/" {
			return handler(ctx, req)
		}

		// 检查是否匹配匿名访问路径
		for _, v := range skips {
			if strings.HasPrefix(info.FullMethod, v) {
				return handler(ctx, req)
			}
		}
	}

	// 从 metadata 中获取 token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	// 获取 Authorization 头
	var inputToken string
	if values := md.Get("authorization"); len(values) > 0 {
		inputToken = values[0]
	}

	// 检查 token 是否为空
	if z.StringIsEmpty(inputToken) {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// 如果 token 以 "Bearer " 开头，则去掉前缀
	if strings.HasPrefix(inputToken, "Bearer ") {
		inputToken = strings.TrimPrefix(inputToken, "Bearer ")
	}

	// 获取 auth 配置
	authConfig, err := z.Config.StringMap("config.auth")
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}

	// 标准化请求路径
	requestPath := strings.TrimPrefix(info.FullMethod, "/")

	// 遍历 auth 配置，找到匹配的路径前缀
	for pathPrefix, configToken := range authConfig {
		normalizedPrefix := strings.TrimPrefix(pathPrefix, "/")

		if strings.HasPrefix(requestPath, normalizedPrefix) {
			if inputToken == configToken {
				// 认证成功，继续处理请求
				return handler(ctx, req)
			} else {
				// 认证失败
				return nil, fmt.Errorf("unauthorized")
			}
		}
	}

	// 没有找到匹配的路径前缀，认证失败
	return nil, fmt.Errorf("access token error")
}
