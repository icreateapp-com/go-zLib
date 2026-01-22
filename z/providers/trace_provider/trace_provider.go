package trace_provider

import (
	"context"
	"fmt"
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

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"go.uber.org/fx"
)

// Trace 链路追踪服务
type Trace struct {
	TracerProvider *tracesdk.TracerProvider
	Tracer         trace.Tracer
	serviceName    string
	enabled        bool
}

// TraceProviderModule 链路追踪模块
var TraceProviderModule = fx.Options(
	fx.Provide(NewTraceProvider),
)

// NewTraceProvider 创建链路追踪实例（fx Provider）
func NewTraceProvider(lc fx.Lifecycle, cfg *config_provider.Config, log *logger_provider.Logger) (*Trace, error) {
	tp := &Trace{}

	tp.enabled = cfg.GetBool("trace.enable", false)
	tp.serviceName = cfg.GetString("app.name", "")
	if tp.serviceName == "" {
		tp.serviceName = "service"
	}

	endpoint := cfg.GetString("trace.otlp.endpoint", "")
	insecure := cfg.GetBool("trace.otlp.insecure", true)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !tp.enabled {
				return nil
			}
			if endpoint == "" {
				return fmt.Errorf("trace enabled but trace.otlp.endpoint is empty")
			}

			tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithDialOption(grpc.WithBlock())}
			if insecure {
				opts = append(opts, otlptracegrpc.WithInsecure())
			}

			exporter, err := otlptracegrpc.New(tctx, opts...)
			if err != nil {
				return err
			}

			sdkTP := tracesdk.NewTracerProvider(
				tracesdk.WithBatcher(exporter),
				tracesdk.WithResource(resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String(tp.serviceName),
				)),
			)

			otel.SetTracerProvider(sdkTP)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

			tp.TracerProvider = sdkTP
			tp.Tracer = sdkTP.Tracer(tp.serviceName)

			log.Infow("provider[trace] enabled", "otlp_endpoint", endpoint)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if tp.TracerProvider == nil {
				return nil
			}
			return tp.TracerProvider.Shutdown(ctx)
		},
	})

	return tp, nil
}

// Shutdown 优雅地关闭 TracerProvider
func (t *Trace) Shutdown() {
	if t.TracerProvider != nil {
		_ = t.TracerProvider.Shutdown(context.Background())
	}
}

// Start 开启一个新的 span
// 可选的 string 类型参数作为 span name，如果没有提供，则自动从 runtime 获取
// 可选的 trace.SpanStartOption 类型参数作为 span 选项
func (t *Trace) Start(ctx context.Context, args ...interface{}) (context.Context, trace.Span) {
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
func (t *Trace) Error(span trace.Span, err error) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}
