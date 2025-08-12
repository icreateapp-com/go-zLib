package trace_provider

import (
	"context"
	"fmt"

	. "github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryMiddleware gRPC恢复中间件 - 捕获panic并返回gRPC错误
func RecoveryMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				var panicErr error
				if e, ok := r.(error); ok {
					panicErr = e
				} else {
					panicErr = fmt.Errorf("panic: %v", r)
				}

				// 使用链路追踪记录panic错误
				_, span := TraceProvider.Start(ctx, fmt.Sprintf("gRPC.Recovery.%s", info.FullMethod))
				defer span.End()
				TraceProvider.Error(span, panicErr)

				// 同时使用日志记录错误
				Error.Println(fmt.Sprintf("gRPC Panic in %s: %v", info.FullMethod, panicErr))

				// 返回gRPC内部服务器错误
				err = status.Errorf(codes.Internal, "Internal server error")
			}
		}()

		return handler(ctx, req)
	}
}