package trace_provider

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type traceIDContextKey struct{}

// WithTraceID 将 trace_id 写入 context，便于业务日志与 HTTP header 对齐。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return ctx
	}

	return context.WithValue(ctx, traceIDContextKey{}, traceID)
}

// GetTraceID 优先从 context value 读取 trace_id，兜底从当前 span 中提取。
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if traceID, ok := ctx.Value(traceIDContextKey{}).(string); ok {
		traceID = strings.TrimSpace(traceID)
		if traceID != "" {
			return traceID
		}
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}

	traceID := spanCtx.TraceID().String()
	if traceID == "" || traceID == "00000000000000000000000000000000" {
		return ""
	}

	return traceID
}
