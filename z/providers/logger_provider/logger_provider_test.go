package logger_provider

import (
	"bytes"
	"errors"
	"syscall"
	"testing"
)

type syncFailWriter struct {
	bytes.Buffer
	syncCalled bool
}

func (w *syncFailWriter) Sync() error {
	w.syncCalled = true
	return syscall.EINVAL
}

func TestNoSyncWriteSyncerWritesWithoutSyncingUnderlyingWriter(t *testing.T) {
	underlying := &syncFailWriter{}
	writer := noSyncWriteSyncer{Writer: underlying}

	if _, err := writer.Write([]byte("log line")); err != nil {
		t.Fatalf("write console log: %v", err)
	}
	if got := underlying.String(); got != "log line" {
		t.Fatalf("unexpected console output: %q", got)
	}
	if err := writer.Sync(); err != nil {
		t.Fatalf("sync should be a no-op, got: %v", err)
	}
	if underlying.syncCalled {
		t.Fatal("underlying stdout sync must not be called")
	}
}

func TestSyncFailWriterDocumentsContainerStdoutFailure(t *testing.T) {
	underlying := &syncFailWriter{}

	if err := underlying.Sync(); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("expected EINVAL, got: %v", err)
	}
}
