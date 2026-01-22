package servers

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"
)

func AppRun(options []fx.Option) error {
	ctx := context.Background()

	options = append(options, fx.NopLogger)

	// 创建 fx.App
	app := fx.New(options...)

	// 启动应用
	if err := app.Start(ctx); err != nil {
		return err
	}

	// 等待信号
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// 停止应用
	stopErr := app.Stop(ctx)
	if stopErr != nil {
		return stopErr
	}

	return nil
}
