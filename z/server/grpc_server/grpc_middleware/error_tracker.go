package grpc_middleware

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorTrackerMiddleware gRPC错误跟踪中间件
func ErrorTrackerMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 生成请求ID
		requestID := uuid.New().String()

		// 创建链路追踪上下文
		traceCtx, span := trace_provider.TraceProvider.Start(ctx, fmt.Sprintf("gRPC.%s", info.FullMethod))
		defer span.End()

		// 处理请求
		resp, err := handler(traceCtx, req)

		// 如果有错误，记录到跟踪器
		if err != nil {
			// 使用链路追踪记录错误
			trace_provider.TraceProvider.Error(traceCtx, span, err)
			// 同时使用日志记录错误
			Error.Println(fmt.Sprintf("gRPC Request %s Error: %v", requestID, err))
		}

		return resp, err
	}
}

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
				traceCtx, span := trace_provider.TraceProvider.Start(ctx, fmt.Sprintf("gRPC.Recovery.%s", info.FullMethod))
				defer span.End()
				trace_provider.TraceProvider.Error(traceCtx, span, panicErr)

				// 同时使用日志记录错误
				Error.Println(fmt.Sprintf("gRPC Panic in %s: %v", info.FullMethod, panicErr))

				// 返回gRPC内部服务器错误
				err = status.Errorf(codes.Internal, "Internal server error")
			}
		}()

		return handler(ctx, req)
	}
}
