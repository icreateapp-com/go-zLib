package db_provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm/logger"
)

func TestFilteredLoggerPreservesParameterFilter(t *testing.T) {
	inner := logger.New(testPrinter{}, logger.Config{ParameterizedQueries: true})
	filtered := NewFilteredGormLogger(inner)
	filter, ok := filtered.(interface {
		ParamsFilter(context.Context, string, ...interface{}) (string, []interface{})
	})
	if !ok {
		t.Fatal("filtered logger does not implement ParamsFilter")
	}
	_, args := filter.ParamsFilter(context.Background(), "select ?", "secret")
	if len(args) != 0 {
		t.Fatal("SQL parameters were not filtered")
	}
}

type testPrinter struct{}

func (testPrinter) Printf(string, ...interface{}) {}

func TestGormTraceLevelAndFiltering(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	l := newGormLogger(zap.New(core), logger.Warn)
	ctx := context.Background()
	fc := func() (string, int64) { return "select ?", 1 }
	l.Trace(ctx, time.Now(), fc, nil)
	if logs.Len() != 0 {
		t.Fatal("normal query logged in production")
	}
	l.Trace(ctx, time.Now().Add(-6*time.Second), fc, nil)
	l.Trace(ctx, time.Now(), fc, errors.New("failure"))
	l.Trace(ctx, time.Now(), fc, context.Canceled)
	l.Trace(ctx, time.Now(), fc, logger.ErrRecordNotFound)
	if logs.Len() != 2 {
		t.Fatalf("got %d entries", logs.Len())
	}
	if logs.All()[0].Level != zapcore.WarnLevel || logs.All()[1].Level != zapcore.ErrorLevel {
		t.Fatal("incorrect SQL log levels")
	}
}

func TestSQLFailureDoesNotLeakErrorParameter(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	l := newGormLogger(zap.New(core), logger.Warn)
	l.Trace(context.Background(), time.Now(), func() (string, int64) { return "INSERT INTO probe VALUES (?)", 0 }, errors.New("duplicate entry secret-value"))
	if len(logs.All()) != 1 {
		t.Fatal("missing SQL failure")
	}
	if strings.Contains(fmt.Sprint(logs.All()[0].ContextMap()), "secret-value") {
		t.Fatal("SQL parameter leaked through error")
	}
}
