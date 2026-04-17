package http_server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/trace_provider"
	"github.com/icreateapp-com/go-zLib/z/servers/http_server/http_server_middlewares"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type HttpMiddlewaresIn struct {
	fx.In

	Items []gin.HandlerFunc `group:"http_middlewares"`
}

type TraceProviderIn struct {
	fx.In

	TraceProvider *trace_provider.Trace `optional:"true"`
}

type RoutesIn struct {
	fx.In

	Engine *gin.Engine
	Routes []RouteRegister `group:"routes"`
}

type RouteRegister func(r *gin.Engine)

func NewHttpServer(in HttpMiddlewaresIn, tpIn TraceProviderIn, cfg *config_provider.Config, log *logger_provider.Logger) (*gin.Engine, error) {
	// set mode
	if !cfg.GetBool("app.debug", true) {
		gin.SetMode(gin.ReleaseMode)
	}

	// instance engine
	r := gin.New()

	// default middlewares
	r.Use(gin.Logger())
	if tpIn.TraceProvider != nil {
		r.Use(http_server_middlewares.TraceChainMiddleware(tpIn.TraceProvider, log))
		r.Use(http_server_middlewares.RecoveryMiddleware(log))
	} else {
		r.Use(gin.Recovery())
	}

	// injected middlewares
	r.Use(in.Items...)

	// static directory
	staticDir := cfg.GetString("http.static_dir")
	staticDir = strings.TrimSpace(staticDir)
	if staticDir != "" {
		cleaned := filepath.Clean(staticDir)
		if cleaned == "/" {
			return nil, errors.New("invalid http.static_dir: cannot be '/' ")
		}
		if filepath.IsAbs(cleaned) {
			return nil, fmt.Errorf("invalid http.static_dir: must be a relative directory, got %q", staticDir)
		}
		info, err := os.Stat(cleaned)
		if err != nil {
			return nil, fmt.Errorf("invalid http.static_dir: %q not found: %w", staticDir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("invalid http.static_dir: %q is not a directory", staticDir)
		}
		r.Use(static.Serve("/", static.LocalFile(cleaned, false)))
	}

	return r, nil
}

func RegisterRoutes(in RoutesIn) {
	for _, register := range in.Routes {
		register(in.Engine)
	}
}

func RegisterHTTPServer(lc fx.Lifecycle, r *gin.Engine, cfg *config_provider.Config, log *logger_provider.Logger) {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.GetString("http.host"), cfg.GetInt("http.port")),
		Handler: r,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Infow("start http server", "addr", srv.Addr)
			go srv.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Infow("stopping http server", "addr", srv.Addr)
			if err := srv.Shutdown(ctx); err != nil {
				log.Errorw("http server stop failed", "addr", srv.Addr, "error", err)
				return err
			}
			log.Infow("http server stopped", "addr", srv.Addr)
			return nil
		},
	})
}

var HttpServerModule = fx.Options(
	fx.Provide(NewHttpServer),
	fx.Invoke(RegisterHTTPServer),
	fx.Invoke(RegisterRoutes),
)
