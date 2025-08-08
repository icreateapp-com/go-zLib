package http_middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
)

// ErrorTrackerMiddleware HTTP错误跟踪中间件
func ErrorTrackerMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// 记录panic错误到跟踪器
		var err error
		if e, ok := recovered.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("%v", recovered)
		}

		// 使用链路追踪记录错误
		ctx, span := trace_provider.TraceProvider.Start(c.Request.Context(), "HTTP.Recovery")
		defer span.End()
		trace_provider.TraceProvider.Error(ctx, span, err)

		// 同时使用日志记录错误
		Error.Println(fmt.Sprintf("HTTP Recovery Error: %v", err))

		// 返回500错误
		Failure(c, "Internal Server Error", 500)
		c.Abort()
	})
}

// ErrorLogMiddleware 错误日志中间件 - 处理请求级别的错误记录
func ErrorLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求ID并存储到上下文中
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// 创建链路追踪上下文
		ctx, span := trace_provider.TraceProvider.Start(c.Request.Context(), "HTTP.Request")
		defer span.End()

		// 将追踪上下文设置到请求中
		c.Request = c.Request.WithContext(ctx)

		// 处理请求
		c.Next()

		// 请求处理完成后，检查是否有错误需要记录
		defer func() {
			// 检查Gin框架的错误
			if len(c.Errors) > 0 {
				for _, ginErr := range c.Errors {
					// 使用链路追踪记录错误
					trace_provider.TraceProvider.Error(ctx, span, ginErr.Err)
					// 同时使用日志记录错误
					Error.Println(fmt.Sprintf("Request %s Error: %v", requestID, ginErr.Err))
				}
			}
		}()
	}
}
