package servers

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
)

const (
	defaultAppStartTimeout = 15 * time.Second
	defaultAppStopTimeout  = 15 * time.Second
)

// AppRuntimeOption 定义应用运行时选项。
type AppRuntimeOption func(*appRuntimeOptions)

type appRuntimeOptions struct {
	singleInstanceLockPath string
}

// WithSingleInstanceLock 启用单机单实例锁，避免同一台机器重复启动同一服务。
func WithSingleInstanceLock(lockPath string) AppRuntimeOption {
	return func(opts *appRuntimeOptions) {
		opts.singleInstanceLockPath = lockPath
	}
}

func AppRun(options []fx.Option, runtimeOptions ...AppRuntimeOption) error {
	appRuntimeOpts := appRuntimeOptions{}
	for _, option := range runtimeOptions {
		if option != nil {
			option(&appRuntimeOpts)
		}
	}

	lockHandle, err := acquireProcessLock(appRuntimeOpts.singleInstanceLockPath)
	if err != nil {
		return err
	}
	defer func() {
		if lockHandle != nil {
			_ = lockHandle.Close()
		}
	}()

	options = append(options, fx.NopLogger)

	// 创建 fx.App
	app := fx.New(options...)

	// 启动应用
	startCtx, startCancel := context.WithTimeout(context.Background(), defaultAppStartTimeout)
	defer startCancel()
	if err := app.Start(startCtx); err != nil {
		return err
	}

	// 等待信号
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// 停止应用
	stopCtx, stopCancel := context.WithTimeout(context.Background(), defaultAppStopTimeout)
	defer stopCancel()
	stopErr := app.Stop(stopCtx)
	if stopErr != nil {
		if errors.Is(stopErr, context.DeadlineExceeded) || errors.Is(stopErr, context.Canceled) {
			return nil
		}
		return stopErr
	}

	return nil
}
