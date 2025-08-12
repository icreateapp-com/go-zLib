package trace_provider

import (
	"context"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	. "github.com/icreateapp-com/go-zLib/z"
)

// TraceProvider 链路追踪服务
type traceProvider struct {
	TracerProvider *tracesdk.TracerProvider
	Tracer         trace.Tracer
}

// TraceProvider 全局链路追踪提供者实例
var TraceProvider = &traceProvider{}

// Init 初始化链路追踪服务
// Init 初始化链路追踪服务
func (t *traceProvider) Init() {
	if !Config.GetBool("config.observe.trace.enable") {
		return
	}

	// 创建一个新的 OTLP gRPC exporter
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint(Config.GetString("config.observe.trace.jaeger.endpoint")), otlptracegrpc.WithDialOption(grpc.WithBlock()))
	if err != nil {
		Error.Fatalf("failed to create trace exporter: %v", err)
	}

	// 创建一个新的 TracerProvider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(Config.GetString("config.name")),
		)),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	t.TracerProvider = tp
	t.Tracer = tp.Tracer(Config.GetString("config.name"))
}

// Shutdown 优雅地关闭 TracerProvider
func (t *traceProvider) Shutdown() {
	if t.TracerProvider != nil {
		if err := t.TracerProvider.Shutdown(context.Background()); err != nil {
			Error.Printf("failed to shutdown trace provider: %v", err)
		}
	}
}

// Start 开启一个新的 span
// 可选的 string 类型参数作为 span name，如果没有提供，则自动从 runtime 获取
// 可选的 trace.SpanStartOption 类型参数作为 span 选项
func (t *traceProvider) Start(ctx context.Context, args ...interface{}) (context.Context, trace.Span) {
	if t.Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	var spanName string
	var opts []trace.SpanStartOption

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			spanName = v
		case trace.SpanStartOption:
			opts = append(opts, v)
		}
	}

	if spanName == "" {
		// 获取调用方的函数名
		pc, _, _, ok := runtime.Caller(1)
		if !ok {
			spanName = "unknown"
		} else {
			fn := runtime.FuncForPC(pc)
			if fn == nil {
				spanName = "unknown"
			} else {
				// 获取完整的函数名，例如：github.com/icreateapp-com/aiaop-server/internal/resource/api/http_api/controllers/resource_controllers.(*ArticleController).Create
				fullName := fn.Name()
				// 只保留包名和函数名
				lastSlash := strings.LastIndex(fullName, "/")
				if lastSlash > 0 {
					fullName = fullName[lastSlash+1:]
				}
				spanName = fullName
			}
		}
	}

	return t.Tracer.Start(ctx, spanName, opts...)
}

// Error 记录错误信息并返回错误
func (t *traceProvider) Error(span trace.Span, err error) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}
