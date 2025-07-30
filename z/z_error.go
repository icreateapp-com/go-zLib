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

// StackFrame 调用栈帧信息
type StackFrame struct {
	File     string // 文件路径
	Line     int    // 行号
	Function string // 函数名
	Package  string // 包名
}

// TraceData 追踪数据
type TraceData struct {
	No         int          // 编号
	Type       string       // 类型：ERROR, PANIC
	Time       time.Time    // 时间
	Message    string       // 错误消息
	StackTrace []StackFrame // 完整调用栈
	ErrorFile  string       // 错误发生的文件
	ErrorLine  int          // 错误发生的行号
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
	maxStackDepth int // 最大调用栈深度
}

// Tracker 追踪器
var Tracker = &tracker{
	traces:        make(map[string][]TraceData),
	requestErrors: make(map[string][]string),
	maxStackDepth: 32, // 默认最大调用栈深度
}

// captureStackTrace 捕获完整的调用栈信息
func (t *tracker) captureStackTrace(skip int) []StackFrame {
	var frames []StackFrame

	// 获取调用栈，跳过指定的层数
	pcs := make([]uintptr, t.maxStackDepth)
	n := runtime.Callers(skip, pcs)

	if n == 0 {
		return frames
	}

	// 获取调用栈帧信息
	callersFrames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := callersFrames.Next()

		// 过滤掉runtime相关的调用
		if !strings.Contains(frame.Function, "runtime.") {
			// 获取包名和函数名
			funcName := frame.Function
			packageName := ""
			if idx := strings.LastIndex(funcName, "/"); idx != -1 {
				if idx2 := strings.Index(funcName[idx:], "."); idx2 != -1 {
					packageName = funcName[:idx+idx2]
					funcName = funcName[idx+idx2+1:]
				}
			} else if idx := strings.LastIndex(funcName, "."); idx != -1 {
				packageName = funcName[:idx]
				funcName = funcName[idx+1:]
			}

			frames = append(frames, StackFrame{
				File:     t.getRelativePath(frame.File),
				Line:     frame.Line,
				Function: funcName,
				Package:  packageName,
			})
		}

		if !more {
			break
		}
	}

	return frames
}

// Error 错误包装器 - 记录错误调用链路
func (t *tracker) Error(err error) error {
	if err == nil {
		return nil
	}

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

	// 捕获完整的调用栈（跳过当前函数）
	stackTrace := t.captureStackTrace(2)

	// 确定错误发生的位置（调用Error函数的位置）
	var errorFile string
	var errorLine int
	if len(stackTrace) > 0 {
		errorFile = stackTrace[0].File
		errorLine = stackTrace[0].Line
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// 创建新的追踪数据
	traceData := TraceData{
		No:         len(existingTraces),
		Type:       "ERROR",
		Time:       time.Now(),
		Message:    err.Error(),
		StackTrace: stackTrace,
		ErrorFile:  errorFile,
		ErrorLine:  errorLine,
	}

	// 添加到追踪链
	newTraces := append(existingTraces, traceData)
	t.traces[traceID] = newTraces

	// 标记当前请求有错误（如果能获取到请求ID）
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
	if err == nil {
		return nil
	}

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

	// 捕获完整的调用栈（跳过当前函数）
	stackTrace := t.captureStackTrace(2)

	// 确定错误发生的位置（调用Errorf函数的位置）
	var errorFile string
	var errorLine int
	if len(stackTrace) > 0 {
		errorFile = stackTrace[0].File
		errorLine = stackTrace[0].Line
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// 创建新的追踪数据
	traceData := TraceData{
		No:         len(existingTraces),
		Type:       "ERROR",
		Time:       time.Now(),
		Message:    err.Error(),
		StackTrace: stackTrace,
		ErrorFile:  errorFile,
		ErrorLine:  errorLine,
	}

	// 添加到追踪链
	newTraces := append(existingTraces, traceData)
	t.traces[traceID] = newTraces

	// 标记当前请求有错误（如果能获取到请求ID）
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
	builder.WriteString(fmt.Sprintf("%s:%d: %s\n", lastTrace.ErrorFile, lastTrace.ErrorLine, lastTrace.Message))

	// 添加堆栈跟踪
	builder.WriteString("[stacktrace]\n")

	// 输出完整的调用栈信息
	if len(lastTrace.StackTrace) > 0 {
		for i, frame := range lastTrace.StackTrace {
			builder.WriteString(fmt.Sprintf("#%d %s:%d in %s\n",
				i, frame.File, frame.Line, frame.Function))
		}
	} else {
		// 如果没有调用栈信息，显示基本信息
		builder.WriteString(fmt.Sprintf("#0 %s:%d\n", lastTrace.ErrorFile, lastTrace.ErrorLine))
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
		// 生成新的 traceID
		traceID := t.generateTraceID()

		// 捕获完整的调用栈（跳过当前函数和recover相关的帧）
		stackTrace := t.captureStackTrace(3)

		// 确定panic发生的位置
		var errorFile string
		var errorLine int
		if len(stackTrace) > 0 {
			// 找到第一个非runtime相关的调用栈帧
			for _, frame := range stackTrace {
				if !strings.Contains(frame.File, "runtime/") &&
					!strings.Contains(frame.File, "z_error.go") {
					errorFile = frame.File
					errorLine = frame.Line
					break
				}
			}
			// 如果没找到合适的帧，使用第一个
			if errorFile == "" && len(stackTrace) > 0 {
				errorFile = stackTrace[0].File
				errorLine = stackTrace[0].Line
			}
		}

		t.mutex.Lock()
		defer t.mutex.Unlock()

		// 创建panic追踪数据
		traceData := TraceData{
			No:         0,
			Type:       "PANIC",
			Time:       time.Now(),
			Message:    fmt.Sprintf("%v", r),
			StackTrace: stackTrace,
			ErrorFile:  errorFile,
			ErrorLine:  errorLine,
		}

		// 存储追踪数据
		t.traces[traceID] = []TraceData{traceData}

		// 标记当前请求有错误（如果能获取到请求ID）
		requestID := t.currentReqID
		if requestID != "" {
			if _, exists := t.requestErrors[requestID]; !exists {
				t.requestErrors[requestID] = []string{}
			}
			t.requestErrors[requestID] = append(t.requestErrors[requestID], traceID)
		}

		// 记录错误日志
		t.LogError(&TrackedError{
			OriginalError: fmt.Errorf("%v", r),
			TraceID:       traceID,
			Traces:        []TraceData{traceData},
		})
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

	// 重新编号所有追踪数据
	for i := range allTraces {
		allTraces[i].No = i
	}

	// 生成合并后的日志消息
	logMessage := fmt.Sprintf(t.formatTraceLog(allTraces))
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
