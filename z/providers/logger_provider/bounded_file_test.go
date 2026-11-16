package logger_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"go.uber.org/fx"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBoundedWriterRotationAndRetention(t *testing.T) {
	dir := t.TempDir()
	w, err := newBoundedFileWriter(dir, 1024, 2500, 7)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	w.now = func() time.Time { return now }
	for i := 0; i < 7; i++ {
		if _, err = w.Write([]byte(strings.Repeat("x", 600) + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var size int64
	for _, e := range entries {
		info, _ := e.Info()
		size += info.Size()
	}
	if size > 2500 {
		t.Fatalf("size %d exceeds quota", size)
	}
	if len(entries) < 2 {
		t.Fatalf("rotation failed: %d", len(entries))
	}
	if err = os.WriteFile(filepath.Join(dir, "other.log"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	now = now.AddDate(0, 0, 9)
	if _, err = w.Write([]byte("new\n")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(dir, "other.log")); err != nil {
		t.Fatal("non log file removed")
	}
	w.Close()
}
func TestBoundedWriterOversizedRecordIsSafeJSON(t *testing.T) {
	dir := t.TempDir()
	w, _ := newBoundedFileWriter(dir, 1024, 2048, 7)
	defer w.Close()
	if _, err := w.Write([]byte(`{"secret":"` + strings.Repeat("a", 2000) + `"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	b, _ := os.ReadFile(files[0])
	var record map[string]interface{}
	if json.Unmarshal(b, &record) != nil || record["truncated"] != true || strings.Contains(string(b), "secret") {
		t.Fatalf("unsafe oversized record: %s", b)
	}
}

func TestBoundedWriterRestartAndExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "20200101.log.1")
	if err := os.WriteFile(old, []byte("expired"), 0600); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(old, when, when); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(dir, "20200101.log.bad")
	if err := os.WriteFile(unrelated, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := newBoundedFileWriter(dir, 1024, 2048, 7)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("first\n")); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	w, err = newBoundedFileWriter(dir, 1024, 2048, 7)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err = w.Write([]byte("second\n")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("expired file not removed")
	}
	if _, err = os.Stat(unrelated); err != nil {
		t.Fatal("unrelated file removed")
	}
	p := filepath.Join(dir, time.Now().Format("20060102")+".log")
	data, err := os.ReadFile(p)
	if err != nil || string(data) != "first\nsecond\n" {
		t.Fatalf("restart lost records: %q %v", data, err)
	}
}
func TestLimitConsoleRecord(t *testing.T) {
	var out strings.Builder
	w := recordLimitWriter{Writer: &out}
	input := []byte(strings.Repeat("secret", maxLogRecord))
	if n, err := w.Write(input); err != nil || n != len(input) {
		t.Fatalf("write: %d %v", n, err)
	}
	if strings.Contains(out.String(), "secret") || !strings.Contains(out.String(), "omitted") {
		t.Fatal("oversized console record not safely omitted")
	}
}

func TestProviderEnablesCapacityLimits(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	content := fmt.Sprintf("debug: false\nlogger:\n  dir: %q\n  level: info\n  max_age: 7\n  max_file_size_mb: 1\n  max_total_size_mb: 2\n", dir)
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config_provider.NewConfigProvider(config_provider.ConfigOptions(configPath))
	if err != nil {
		t.Fatal(err)
	}
	lifecycle := &testLogLifecycle{}
	l, err := NewLoggerProvider(lifecycle, cfg)
	if err != nil {
		t.Fatal(err)
	}
	l.Infow("probe", "payload", strings.Repeat("secret", maxLogRecord))
	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	if len(files) != 1 {
		t.Fatalf("expected file, got %v", files)
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret") || !strings.Contains(string(b), `"truncated":true`) {
		t.Fatal("oversized file log not sanitized")
	}
	for _, hook := range lifecycle.hooks {
		if hook.OnStop != nil {
			if err := hook.OnStop(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

type testLogLifecycle struct{ hooks []fx.Hook }

func (l *testLogLifecycle) Append(h fx.Hook) { l.hooks = append(l.hooks, h) }
