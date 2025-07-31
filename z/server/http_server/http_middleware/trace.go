package http_middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TraceMiddleware 链路追踪中间件
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中提取 trace context
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 创建一个新的 span
		tracer := trace_provider.TraceProvider.Tracer
		spanName := c.Request.Method + " " + c.FullPath()
		var span trace.Span
		ctx, span = tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// 将 trace context 注入到 gin.Context 中
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// 记录响应状态码
		code, msg := SpanStatusFromHTTP(c.Writer.Status())
		span.SetStatus(code, msg)
		span.SetAttributes(HTTPServerAttributesFromHTTPStatusCode(c.Writer.Status())...)
	}
}

// SpanStatusFromHTTP returns a span status code and message for an HTTP status code.
func SpanStatusFromHTTP(httpStatusCode int) (codes.Code, string) {
	if httpStatusCode >= 100 && httpStatusCode < 400 {
		return codes.Unset, ""
	}
	return codes.Error, ""
}

// HTTPServerAttributesFromHTTPStatusCode returns the conventional OTel attributes for an HTTP status code.
func HTTPServerAttributesFromHTTPStatusCode(httpStatusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{semconv.HTTPStatusCode(httpStatusCode)}
}
