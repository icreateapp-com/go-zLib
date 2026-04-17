package servers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type processLock struct {
	file *os.File
}

// acquireProcessLock 获取单机进程锁。
// 进程退出后，内核会自动释放 flock，因此不会留下脏锁。
func acquireProcessLock(lockPath string) (*processLock, error) {
	lockPath = strings.TrimSpace(lockPath)
	if lockPath == "" {
		return nil, nil
	}

	lockPath = filepath.Clean(lockPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to prepare process lock dir: %w", err)
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open process lock file: %w", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("process lock is already held: %s", lockPath)
		}
		return nil, fmt.Errorf("failed to acquire process lock: %w", err)
	}

	if err := file.Truncate(0); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("failed to reset process lock file: %w", err)
	}
	if _, err := file.WriteString(fmt.Sprintf("%d\n", os.Getpid())); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("failed to write process lock pid: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("failed to rewind process lock file: %w", err)
	}

	return &processLock{file: file}, nil
}

// Close 释放进程锁。
func (lock *processLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}

	unlockErr := syscall.Flock(int(lock.file.Fd()), syscall.LOCK_UN)
	closeErr := lock.file.Close()
	lock.file = nil

	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
