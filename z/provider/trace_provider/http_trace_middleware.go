package trace_provider

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// HttpTraceMiddleware 链路追踪中间件 - 专门处理链路追踪功能
func HttpTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		)

		// 处理请求
		c.Next()

		// 记录响应状态码
		statusCode := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		// 设置span状态
		spanCode, msg := SpanStatusFromHTTP(statusCode)
		span.SetStatus(spanCode, msg)
	}
}

// SpanStatusFromHTTP 根据HTTP状态码返回span状态码和消息
func SpanStatusFromHTTP(httpStatusCode int) (codes.Code, string) {
	if httpStatusCode >= 100 && httpStatusCode < 400 {
		return codes.Unset, ""
	}
	return codes.Error, ""
}
