package trace_provider

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/icreateapp-com/go-zLib/z"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"time"
)

// HttpTraceMiddleware 链路追踪中间件 - 专门处理链路追踪功能
func HttpTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 生成请求ID并存储到上下文中
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// 从请求头中提取 trace context
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 创建一个新的 span
		spanName := c.Request.Method + " " + c.FullPath()
		var span trace.Span
		ctx, span = TraceProvider.Tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// 将 trace context 注入到 gin.Context 中
		c.Request = c.Request.WithContext(ctx)

		// 添加基础属性
		span.SetAttributes(
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPURL(c.Request.URL.String()),
			semconv.UserAgentOriginal(c.Request.UserAgent()),
			attribute.String("request.id", requestID),
			attribute.String("request.client_ip", c.ClientIP()),
			attribute.String("request.remote_addr", c.Request.RemoteAddr),
			attribute.Int64("request.content_length", c.Request.ContentLength),
		)

		// 处理请求
		c.Next()

		// 计算请求处理时间
		duration := time.Since(startTime)

		// 记录响应状态码和其他响应信息
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		span.SetAttributes(
			semconv.HTTPStatusCode(statusCode),
			attribute.Int64("response.size", int64(responseSize)),
			attribute.Float64("request.duration_ms", float64(duration.Nanoseconds())/1e6),
		)

		// 检查是否有错误并记录详细信息
		if len(c.Errors) > 0 {
			// 记录错误到日志
			errorSummary := fmt.Sprintf(`=== HTTP TRACE ERROR SUMMARY ===
Request ID: %s
Method: %s
Path: %s
Status Code: %d
Duration: %v
Client IP: %s
Errors Count: %d`, requestID, c.Request.Method, c.FullPath(), statusCode, duration, c.ClientIP(), len(c.Errors))

			Info.Println(errorSummary)

			// 为每个错误添加属性
			for i, ginErr := range c.Errors {
				span.SetAttributes(
					attribute.String(fmt.Sprintf("error.%d.type", i), fmt.Sprintf("%v", ginErr.Type)),
					attribute.String(fmt.Sprintf("error.%d.message", i), ginErr.Err.Error()),
				)
			}
		}

		// 设置span状态
		spanCode, msg := SpanStatusFromHTTP(statusCode)
		span.SetStatus(spanCode, msg)

		// 记录请求完成信息（仅在调试模式下）
		if statusCode >= 400 {
			requestSummary := fmt.Sprintf(`=== HTTP REQUEST COMPLETED ===
Request ID: %s
Method: %s
Path: %s
Status Code: %d
Duration: %v
Response Size: %d bytes
Client IP: %s
User Agent: %s
=== END REQUEST ===`, requestID, c.Request.Method, c.FullPath(), statusCode, duration, responseSize, c.ClientIP(), c.Request.UserAgent())

			Info.Println(requestSummary)
		}
	}
}

// SpanStatusFromHTTP 根据HTTP状态码返回span状态码和消息
func SpanStatusFromHTTP(httpStatusCode int) (codes.Code, string) {
	switch {
	case httpStatusCode >= 100 && httpStatusCode < 400:
		return codes.Unset, ""
	case httpStatusCode >= 400 && httpStatusCode < 500:
		return codes.Error, fmt.Sprintf("Client Error: HTTP %d", httpStatusCode)
	case httpStatusCode >= 500:
		return codes.Error, fmt.Sprintf("Server Error: HTTP %d", httpStatusCode)
	default:
		return codes.Error, fmt.Sprintf("Unknown HTTP Status: %d", httpStatusCode)
	}
}
