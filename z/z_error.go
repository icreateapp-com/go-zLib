package z

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// TraceData 追踪数据
type TraceData struct {
	No       int       // 编号
	Type     string    // 类型：ERROR
	Time     time.Time // 时间：2025/07/20 00:21:48
	File     string    // 文件：项目相对路径：/src/main.go
	Line     int       // 行号
	Message  string    // 错误消息
	Function string    // 函数名
}

// TrackedError 包装的错误类型
type TrackedError struct {
	OriginalError error
	TraceID       string
	Traces        []TraceData
}

func (e *TrackedError) Error() string {
	return e.OriginalError.Error()
}

// tracker 追踪器
type tracker struct {
	traces        map[string][]TraceData // 使用 traceID 作为 key
	requestErrors map[string][]string    // 请求级别的错误标记，key为请求ID，value为traceID列表
	currentReqID  string                 // 当前请求ID（简化实现）
	mutex         sync.RWMutex
	counter       int
}

// Tracker 追踪器
var Tracker = &tracker{
	traces:        make(map[string][]TraceData),
	requestErrors: make(map[string][]string),
}

// Error 错误包装器 - 记录错误调用链路
func (t *tracker) Error(err error) error {
	if err == nil {
		return nil
	}

	// 获取调用者信息
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return err
	}

	// 获取函数名
	funcName := runtime.FuncForPC(pc).Name()
	if idx := strings.LastIndex(funcName, "."); idx != -1 {
		funcName = funcName[idx+1:]
	}

	// 获取相对路径
	relativeFile := t.getRelativePath(file)

	var traceID string
	var existingTraces []TraceData

	// 检查是否已经是被跟踪的错误
	if trackedErr, ok := err.(*TrackedError); ok {
		traceID = trackedErr.TraceID
		existingTraces = trackedErr.Traces
	} else {
		// 新的错误，生成新的 traceID
		traceID = t.generateTraceID()
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// 创建新的追踪数据
	traceData := TraceData{
		No:       len(existingTraces),
		Type:     "ERROR",
		Time:     time.Now(),
		File:     relativeFile,
		Line:     line,
		Message:  err.Error(),
		Function: funcName,
	}

	// 添加到追踪链
	newTraces := append(existingTraces, traceData)
	t.traces[traceID] = newTraces

	// 标记当前请求有错误（如果能获取到请求ID）
	// 注意：这里直接访问currentReqID，因为已经持有了锁
	requestID := t.currentReqID
	if requestID != "" {
		if _, exists := t.requestErrors[requestID]; !exists {
			t.requestErrors[requestID] = []string{}
		}
		// 避免重复添加相同的traceID
		found := false
		for _, existingTraceID := range t.requestErrors[requestID] {
			if existingTraceID == traceID {
				found = true
				break
			}
		}
		if !found {
			t.requestErrors[requestID] = append(t.requestErrors[requestID], traceID)
		}
	}

	// 返回包装的错误
	return &TrackedError{
		OriginalError: err,
		TraceID:       traceID,
		Traces:        newTraces,
	}
}

// Errorf 格式化错误包装器 - 支持 fmt.Errorf 的格式化形式
func (t *tracker) Errorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	return t.Error(err)
}

// GetData 获取所有追踪数据
func (t *tracker) GetData() []TraceData {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var allTraces []TraceData
	for _, traces := range t.traces {
		allTraces = append(allTraces, traces...)
	}
	return allTraces
}

// GetTracesByID 根据 traceID 获取特定的追踪数据
func (t *tracker) GetTracesByID(traceID string) []TraceData {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if traces, exists := t.traces[traceID]; exists {
		return traces
	}
	return nil
}

// Clear 清空所有追踪数据
func (t *tracker) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.traces = make(map[string][]TraceData)
}

// LogError 将错误追踪数据写入日志
func (t *tracker) LogError(err error) {
	if !t.ensureLogger() {
		fmt.Fprintf(os.Stderr, "Logger not available, error: %v\n", err)
		return
	}

	if trackedErr, ok := err.(*TrackedError); ok {
		logMessage := t.formatTraceLog(trackedErr.Traces)
		Error.Println(logMessage)
	} else {
		Error.Println(err.Error())
	}
}

// LogAllTraces 将所有追踪数据写入日志
func (t *tracker) LogAllTraces() {
	if !t.ensureLogger() {
		fmt.Fprintf(os.Stderr, "Logger not available\n")
		return
	}

	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// 遍历所有追踪数据
	for traceID, traces := range t.traces {
		if len(traces) > 0 {
			logMessage := fmt.Sprintf("TraceID: %s\n%s", traceID, t.formatTraceLog(traces))
			Error.Println(logMessage)
		}
	}
}

// formatTraceLog 格式化追踪日志
func (t *tracker) formatTraceLog(traces []TraceData) string {
	if len(traces) == 0 {
		return ""
	}

	var builder strings.Builder

	// 最后一个错误作为主错误信息
	lastTrace := traces[len(traces)-1]
	builder.WriteString(fmt.Sprintf("%s:%d: %s\n", lastTrace.File, lastTrace.Line, lastTrace.Message))

	// 添加堆栈跟踪
	builder.WriteString("[stacktrace]\n")

	// 倒序输出调用链（从最深层开始）
	for i := len(traces) - 1; i >= 0; i-- {
		trace := traces[i]
		builder.WriteString(fmt.Sprintf("#%d %s:%d %s\n",
			len(traces)-1-i, trace.File, trace.Line, trace.Function))
	}

	builder.WriteString("{main}")

	return builder.String()
}

// generateTraceID 生成追踪ID
func (t *tracker) generateTraceID() string {
	t.counter++
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), t.counter)
}

// getRelativePath 获取相对路径
func (t *tracker) getRelativePath(fullPath string) string {
	// 安全地尝试获取项目根路径
	defer func() {
		if r := recover(); r != nil {
			// 如果BasePath()出错，直接返回文件名
		}
	}()

	basePath := BasePath()
	if relPath, err := filepath.Rel(basePath, fullPath); err == nil {
		return "/" + strings.ReplaceAll(relPath, "\\", "/")
	}

	// 如果无法获取相对路径，返回文件名
	return "/" + filepath.Base(fullPath)
}

// ensureLogDirectory 确保日志目录存在
func (t *tracker) ensureLogDirectory() {
	defer func() {
		if r := recover(); r != nil {
			// 如果创建目录失败，忽略错误
		}
	}()

	logDir := LogPath()
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}
}

// ensureLogger 确保日志系统已初始化
func (t *tracker) ensureLogger() bool {
	if Error == nil {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", r)
			}
		}()
		t.ensureLogDirectory()
		Log.Init(true, true)
	}
	return Error != nil
}

// RecoverAndLog 恢复panic并记录错误
func (t *tracker) RecoverAndLog() {
	if r := recover(); r != nil {
		var err error
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("%v", r)
		}

		// 记录panic错误
		trackedErr := t.Error(err)
		t.LogError(trackedErr)

		// 重新抛出panic
		panic(r)
	}
}

// SetRequestID 设置当前请求ID
func (t *tracker) SetRequestID(requestID string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.currentReqID = requestID
}

// HasRequestErrors 检查指定请求是否有错误
func (t *tracker) HasRequestErrors(requestID string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	traceIDs, exists := t.requestErrors[requestID]
	return exists && len(traceIDs) > 0
}

// LogRequestErrors 记录指定请求的所有错误
func (t *tracker) LogRequestErrors(requestID string) {
	t.mutex.RLock()
	traceIDs, exists := t.requestErrors[requestID]
	if !exists || len(traceIDs) == 0 {
		t.mutex.RUnlock()
		return
	}

	// 收集所有错误的追踪数据并合并
	var allTraces []TraceData
	for _, traceID := range traceIDs {
		if traces, exists := t.traces[traceID]; exists {
			allTraces = append(allTraces, traces...)
		}
	}
	t.mutex.RUnlock()

	if len(allTraces) == 0 {
		return
	}

	if !t.ensureLogger() {
		fmt.Fprintf(os.Stderr, "Logger not available for request %s\n", requestID)
		return
	}

	// 合并所有错误为一个完整的调用链
	// 使用第一个traceID作为主要ID
	mainTraceID := traceIDs[0]

	// 重新编号所有追踪数据
	for i := range allTraces {
		allTraces[i].No = i
	}

	// 生成合并后的日志消息
	logMessage := fmt.Sprintf("Request: %s, TraceID: %s\n%s",
		requestID, mainTraceID, t.formatTraceLog(allTraces))
	Error.Println(logMessage)
}

// ClearRequestErrors 清理指定请求的错误记录
func (t *tracker) ClearRequestErrors(requestID string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if traceIDs, exists := t.requestErrors[requestID]; exists {
		// 清理对应的traces
		for _, traceID := range traceIDs {
			delete(t.traces, traceID)
		}
		// 清理请求错误记录
		delete(t.requestErrors, requestID)
	}
}
