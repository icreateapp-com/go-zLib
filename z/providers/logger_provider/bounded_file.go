package logger_provider

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

var datedLogName = regexp.MustCompile(`^\d{8}\.log(?:\.\d+)?$`)

const maxLogRecord = 64 * 1024

type boundedFileWriter struct {
	mu                sync.Mutex
	dir               string
	maxFile, maxTotal int64
	maxAge            time.Duration
	now               func() time.Time
	file              *os.File
	name, date        string
	size, total       int64
}

func newBoundedFileWriter(dir string, maxFile, maxTotal int64, days int) (*boundedFileWriter, error) {
	if maxFile < 256 || maxTotal < maxFile || days <= 0 {
		return nil, fmt.Errorf("invalid bounded logging configuration")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &boundedFileWriter{dir: dir, maxFile: maxFile, maxTotal: maxTotal, maxAge: time.Duration(days) * 24 * time.Hour, now: time.Now}, nil
}
func (w *boundedFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	limit := int64(maxLogRecord)
	if w.maxFile < limit {
		limit = w.maxFile
	}
	if int64(n) > limit {
		p, _ = json.Marshal(map[string]interface{}{"level": "WARN", "msg": "oversized log record omitted", "truncated": true, "original_bytes": n})
		p = append(p, '\n')
	}
	today := w.now().Format("20060102")
	if w.file == nil || w.date != today || w.size+int64(len(p)) > w.maxFile {
		if err := w.rotate(today, int64(len(p))); err != nil {
			return 0, err
		}
	}
	if w.total+int64(len(p)) > w.maxTotal {
		if err := w.prune(int64(len(p))); err != nil {
			return 0, err
		}
	}
	written, err := w.file.Write(p)
	w.size += int64(written)
	w.total += int64(written)
	if err != nil {
		return 0, err
	}
	return n, nil
}
func (w *boundedFileWriter) rotate(date string, incoming int64) error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	for i := 0; ; i++ {
		name := date + ".log"
		if i > 0 {
			name = fmt.Sprintf("%s.%d", name, i)
		}
		path := filepath.Join(w.dir, name)
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && (!info.Mode().IsRegular() || info.Size()+incoming > w.maxFile) {
			continue
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		info, err = f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		w.file = f
		w.name = name
		w.date = date
		w.size = info.Size()
		break
	}
	if err := w.prune(0); err != nil {
		_ = w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

type logFileInfo struct {
	name string
	size int64
	mod  time.Time
}

func (w *boundedFileWriter) prune(incoming int64) error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}
	files := make([]logFileInfo, 0, len(entries))
	var total int64
	for _, e := range entries {
		if !datedLogName.MatchString(e.Name()) || e.Type()&os.ModeSymlink != 0 {
			continue
		}
		i, err := e.Info()
		if err != nil || !i.Mode().IsRegular() {
			continue
		}
		total += i.Size()
		if e.Name() != w.name {
			files = append(files, logFileInfo{e.Name(), i.Size(), i.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].mod.Equal(files[j].mod) {
			return files[i].name < files[j].name
		}
		return files[i].mod.Before(files[j].mod)
	})
	cutoff := w.now().Add(-w.maxAge)
	for _, f := range files {
		if total+incoming <= w.maxTotal && !f.mod.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(w.dir, f.name)); err != nil {
			return err
		}
		total -= f.size
	}
	w.total = total
	if total+incoming > w.maxTotal {
		return fmt.Errorf("active log exceeds total size limit")
	}
	return nil
}
func (w *boundedFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// recordLimitWriter prevents unbounded console records without echoing payloads.
type recordLimitWriter struct{ io.Writer }

func (w recordLimitWriter) Write(p []byte) (int, error) {
	if len(p) <= maxLogRecord {
		return w.Writer.Write(p)
	}
	_, err := fmt.Fprintf(w.Writer, "WARN oversized log record omitted (original_bytes=%d)\n", len(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
