package grpc_middleware

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorTrackerMiddleware gRPC错误跟踪中间件
func ErrorTrackerMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 生成请求ID并设置到错误跟踪器
	requestID := uuid.New().String()
	Tracker.SetRequestID(requestID)

	// 使用defer确保请求结束时处理错误记录
	defer func() {
		// 如果当前请求有错误，记录到日志
		if Tracker.HasRequestErrors(requestID) {
			Tracker.LogRequestErrors(requestID)
		}

		// 清理请求错误记录
		Tracker.ClearRequestErrors(requestID)

		// 清理当前请求ID
		Tracker.SetRequestID("")
	}()

	// 使用defer捕获panic
	defer func() {
		if r := recover(); r != nil {
			var err error
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("panic: %v", r)
			}

			// 记录panic错误到跟踪器
			Tracker.Error(err)

			// 重新抛出panic
			panic(r)
		}
	}()

	// 调用实际的处理器
	resp, err := handler(ctx, req)

	// 如果有错误，记录到跟踪器
	if err != nil {
		Tracker.Error(err)
	}

	return resp, err
}

// RecoveryMiddleware gRPC恢复中间件
func RecoveryMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = fmt.Errorf("panic: %v", r)
			}

			// 记录panic错误到跟踪器
			Tracker.Error(panicErr)

			// 返回gRPC错误而不是重新抛出panic
			err = status.Errorf(codes.Internal, "Internal server error")
		}
	}()

	return handler(ctx, req)
}
