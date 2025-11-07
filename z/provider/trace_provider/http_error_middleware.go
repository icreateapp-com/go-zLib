package trace_provider

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HttpErrorRecoveryMiddleware HTTP错误恢复中间件 - 处理panic错误
func HttpErrorRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// 获取调用栈信息
		stack := debug.Stack()

		// 记录panic错误到跟踪器
		var err error
		if e, ok := recovered.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("%v", recovered)
		}

		// 获取请求信息
		requestID, _ := c.Get("request_id")
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()
		clientIP := c.ClientIP()

		// 构建详细的错误信息
		errorDetails := fmt.Sprintf(`=== HTTP PANIC RECOVERY ===
Request ID: %v
Method: %s
Path: %s
Client IP: %s
User Agent: %s
Error: %v

=== STACK TRACE ===
%s
=== END STACK TRACE ===`, requestID, method, path, clientIP, userAgent, err, string(stack))

		// 获取当前span（如果存在）
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			// 使用链路追踪记录错误
			TraceProvider.Error(span, err)
			// 添加详细的属性信息
			span.SetAttributes(
				attribute.String("error.type", "panic"),
				attribute.String("error.message", err.Error()),
				attribute.String("request.method", method),
				attribute.String("request.path", path),
				attribute.String("request.client_ip", clientIP),
			)
		} else {
			// 如果没有现有span，创建一个新的用于记录错误
			_, newSpan := TraceProvider.Start(c.Request.Context(), "HTTP.Recovery")
			defer newSpan.End()
			TraceProvider.Error(newSpan, err)
		}

		// 记录完整的错误信息到日志
		Error.Println(errorDetails)

		// 返回500错误
		Failure(c, "Internal Server Error", 500)
		c.Abort()
	})
}

// HttpErrorLoggingMiddleware HTTP错误日志中间件 - 记录请求处理过程中的错误
func HttpErrorLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求
		c.Next()

		// 请求处理完成后，检查是否有错误需要记录
		if len(c.Errors) > 0 {
			requestID, _ := c.Get("request_id")
			span := trace.SpanFromContext(c.Request.Context())
			method := c.Request.Method
			path := c.Request.URL.Path
			statusCode := c.Writer.Status()
			clientIP := c.ClientIP()

			for i, ginErr := range c.Errors {
				// 构建详细的错误信息
				errorDetails := fmt.Sprintf(`=== HTTP REQUEST ERROR #%d ===
Request ID: %v
Method: %s
Path: %s
Status Code: %d
Client IP: %s
Error Type: %s
Error Message: %v
Error Meta: %v
=== END ERROR #%d ===`, i+1, requestID, method, path, statusCode, clientIP, ginErr.Type, ginErr.Err, ginErr.Meta, i+1)

				// 使用链路追踪记录错误
				if span.IsRecording() {
					TraceProvider.Error(span, ginErr.Err)
					// 添加错误属性
					span.SetAttributes(
						attribute.String("error.type", fmt.Sprintf("%v", ginErr.Type)),
						attribute.String("error.message", ginErr.Err.Error()),
						attribute.String("request.method", method),
						attribute.String("request.path", path),
						attribute.Int("response.status_code", statusCode),
						attribute.String("request.client_ip", clientIP),
					)
				}

				// 记录详细的错误信息到日志
				Error.Println(errorDetails)
			}
		}
	}
}
