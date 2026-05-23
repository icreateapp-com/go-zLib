package db_provider

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm/logger"
)

// FilteredGormLogger 用于过滤请求取消产生的无意义 SQL 噪音日志。
//
// 仅压制 context.Canceled，保留真实慢查询与数据库错误日志，
// 避免把前端主动取消请求、连接切换等正常行为误判为数据库异常。
type FilteredGormLogger struct {
	inner logger.Interface
}

// NewFilteredGormLogger 创建带 context canceled 过滤能力的 GORM Logger。
func NewFilteredGormLogger(inner logger.Interface) logger.Interface {
	return &FilteredGormLogger{inner: inner}
}

// LogMode 保持原始日志级别配置行为。
func (l *FilteredGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &FilteredGormLogger{inner: l.inner.LogMode(level)}
}

// Info 透传普通信息日志。
func (l *FilteredGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.inner.Info(ctx, msg, data...)
}

// Warn 透传警告日志。
func (l *FilteredGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.inner.Warn(ctx, msg, data...)
}

// Error 透传错误日志。
func (l *FilteredGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.inner.Error(ctx, msg, data...)
}

// Trace 过滤因请求取消产生的 SQL 噪音，其他日志保持原样。
func (l *FilteredGormLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if errors.Is(err, context.Canceled) {
		return
	}
	l.inner.Trace(ctx, begin, fc, err)
}
