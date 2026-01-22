package logger_provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志管理
type Logger struct {
	base *zap.Logger
	log  *zap.SugaredLogger
}

// Base 返回底层 zap.Logger。
func (l *Logger) Base() *zap.Logger {
	return l.base
}

// Sugar 返回 SugaredLogger。
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.log
}

// Debugw 输出 debug 级别结构化日志。
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.log.Debugw(msg, keysAndValues...)
}

// Infow 输出 info 级别结构化日志。
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.log.Infow(msg, keysAndValues...)
}

// Warnw 输出 warn 级别结构化日志。
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.log.Warnw(msg, keysAndValues...)
}

// Errorw 输出 error 级别结构化日志。
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.log.Errorw(msg, keysAndValues...)
}

// Debugf 输出 debug 级别格式化日志。
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.log.Debugf(template, args...)
}

// Infof 输出 info 级别格式化日志。
func (l *Logger) Infof(template string, args ...interface{}) {
	l.log.Infof(template, args...)
}

// Warnf 输出 warn 级别格式化日志。
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.log.Warnf(template, args...)
}

// Errorf 输出 error 级别格式化日志。
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.log.Errorf(template, args...)
}

// NewLoggerProvider 创建日志管理实例

func NewLoggerProvider(lc fx.Lifecycle, cfg *config_provider.Config) (*Logger, error) {
	debugMode := cfg.GetBool("app.debug", true)
	levelStr := cfg.GetString("logger.level", "info")
	logDir := cfg.GetString("logger.dir", "./storage/log")
	maxAge := cfg.GetInt("logger.max_age", 7)
	printLine := cfg.GetBool("logger.print_line", false)

	logDir = strings.TrimSpace(logDir)
	if logDir == "" {
		logDir = "./storage/log"
	}

	var lvl zapcore.Level
	if err := lvl.Set(levelStr); err != nil {
		lvl = zapcore.InfoLevel
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encCfg)
	fileEncoder := zapcore.NewJSONEncoder(encCfg)

	pattern := filepath.Join(logDir, "%Y%m%d.log")
	fileWriter, err := rotatelogs.New(
		pattern,
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(time.Duration(maxAge)*24*time.Hour),
	)
	if err != nil {
		return nil, err
	}
	fileWS := zapcore.AddSync(fileWriter)
	consoleWS := zapcore.AddSync(zapcore.Lock(os.Stdout))

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleWS, lvl),
		zapcore.NewCore(fileEncoder, fileWS, lvl),
	)

	opts := []zap.Option{}
	if debugMode && printLine {
		opts = append(opts, zap.AddCaller())
	}
	if debugMode {
		opts = append(opts, zap.Development())
	}
	base := zap.New(core, opts...)
	// 通过 AddCallerSkip(1) 跳过 logger_provider 的封装层，确保 caller 指向业务调用处
	sugar := base.WithOptions(zap.AddCallerSkip(1)).Sugar()

	l := &Logger{base: base, log: sugar}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			err := base.Sync()
			if err == nil {
				return nil
			}
			// zap 在某些环境下（例如输出被重定向/非 tty）对 stdout/stderr Sync 可能返回该错误。
			// 参考 zap 官方建议：忽略 stdout/stderr 的 sync 错误。
			if errors.Is(err, os.ErrInvalid) {
				return nil
			}
			if strings.Contains(err.Error(), "inappropriate ioctl") || strings.Contains(err.Error(), "bad file descriptor") {
				return nil
			}
			return err
		},
	})

	return l, nil
}

// LoggerProviderModule 日志管理模块
var LoggerProviderModule = fx.Options(
	fx.Provide(NewLoggerProvider),
)
