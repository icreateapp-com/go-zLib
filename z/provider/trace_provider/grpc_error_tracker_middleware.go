package trace_provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
)

// ErrorTrackerMiddleware gRPC错误跟踪中间件
func ErrorTrackerMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 生成请求ID
		requestID := uuid.New().String()

		// 创建链路追踪上下文
		traceCtx, span := TraceProvider.Start(ctx, fmt.Sprintf("gRPC.%s", info.FullMethod))
		defer span.End()

		// 处理请求
		resp, err := handler(traceCtx, req)

		// 如果有错误，记录到跟踪器
		if err != nil {
			// 使用链路追踪记录错误
			TraceProvider.Error(span, err)
			// 同时使用日志记录错误
			Error.Println(fmt.Sprintf("gRPC Request %s Error: %v", requestID, err))
		}

		return resp, err
	}
}