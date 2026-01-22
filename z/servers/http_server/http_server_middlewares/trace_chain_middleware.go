package http_server_middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/trace_provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func TraceChainMiddleware(tp *trace_provider.Trace, log *logger_provider.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tp == nil {
			c.Next()
			return
		}

		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := c.Request.Method + " " + c.FullPath()
		ctx, span := tp.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		traceID := span.SpanContext().TraceID().String()
		if traceID == "00000000000000000000000000000000" {
			traceID = uuid.New().String()
		}
		c.Set("trace_id", traceID)
		c.Writer.Header().Set("X-Trace-Id", traceID)
		c.Request.Header.Set("X-Trace-Id", traceID)

		c.Request = c.Request.WithContext(ctx)

		span.SetAttributes(
			semconv.HTTPMethod(c.Request.Method),
			semconv.HTTPURL(c.Request.URL.String()),
			semconv.UserAgentOriginal(c.Request.UserAgent()),
			attribute.String("trace.id", traceID),
		)

		c.Next()

		c.Writer.Header().Set("X-Trace-Id", traceID)

		statusCode := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))
		spanCode, msg := SpanStatusFromHTTP(statusCode)
		span.SetStatus(spanCode, msg)

		if len(c.Errors) == 0 {
			return
		}

		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		for i, ginErr := range c.Errors {
			errorDetails := fmt.Sprintf(`=== HTTP REQUEST ERROR #%d ===
Trace ID: %v
Method: %s
Path: %s
Status Code: %d
Client IP: %s
Error Type: %v
Error Message: %v
Error Meta: %v
=== END ERROR #%d ===`, i+1, traceID, method, path, statusCode, clientIP, ginErr.Type, ginErr.Err, ginErr.Meta, i+1)

			if span.IsRecording() {
				tp.Error(span, ginErr.Err)
				span.SetAttributes(
					attribute.String("error.type", fmt.Sprintf("%v", ginErr.Type)),
					attribute.String("error.message", ginErr.Err.Error()),
					attribute.String("trace.id", traceID),
					attribute.String("request.method", method),
					attribute.String("request.path", path),
					attribute.Int("response.status_code", statusCode),
					attribute.String("request.client_ip", clientIP),
				)
			}

			log.Errorw("request error", "details", errorDetails)
		}
	}
}

func SpanStatusFromHTTP(httpStatusCode int) (codes.Code, string) {
	if httpStatusCode >= 100 && httpStatusCode < 400 {
		return codes.Unset, ""
	}
	return codes.Error, ""
}

var TraceChainMiddlewareModule = fx.Options(
	fx.Provide(
		fx.Annotate(
			TraceChainMiddleware,
			fx.ParamTags(``),
			fx.ResultTags(`group:"http_middlewares"`),
		),
	),
)
