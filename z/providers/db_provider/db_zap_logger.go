package db_provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// gormZapLogger preserves SQL severity and never interpolates SQL parameters.
type gormZapLogger struct {
	log   *zap.Logger
	level logger.LogLevel
}

func newGormLogger(log *zap.Logger, level logger.LogLevel) logger.Interface {
	return &gormZapLogger{log, level}
}
func (l *gormZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &gormZapLogger{l.log, level}
}
func (l *gormZapLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Info {
		l.log.Info(fmt.Sprintf(msg, args...))
	}
}
func (l *gormZapLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Warn {
		l.log.Warn(fmt.Sprintf(msg, args...))
	}
}
func (l *gormZapLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Error {
		l.log.Error(fmt.Sprintf(msg, args...))
	}
}
func (l *gormZapLogger) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}
func (l *gormZapLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent || errors.Is(err, context.Canceled) || errors.Is(err, logger.ErrRecordNotFound) {
		return
	}
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= logger.Error:
		sql, rows := fc()
		fields := []zap.Field{zap.String("sql", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed), zap.String("error_type", fmt.Sprintf("%T", err))}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			fields = append(fields, zap.Uint16("mysql_code", mysqlErr.Number))
		}
		l.log.Error("sql execution failed", fields...)
	case elapsed > 5*time.Second && l.level >= logger.Warn:
		sql, rows := fc()
		l.log.Warn("slow sql", zap.String("sql", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed))
	case l.level >= logger.Info:
		sql, rows := fc()
		l.log.Info("sql executed", zap.String("sql", sql), zap.Int64("rows", rows), zap.Duration("elapsed", elapsed))
	}
}
