package z

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// TrackedError 包装的错误结构体，包含调用栈信息
type TrackedError struct {
	OriginalError error
	Message       string
	Description   string
	File          string
	Line          int
	Function      string
	Timestamp     time.Time
	StackTrace    []StackFrame
}

// StackFrame 调用栈帧信息
type StackFrame struct {
	File     string
	Line     int
	Function string
}

// Error 实现 error 接口
func (te *TrackedError) Error() string {
	if te.OriginalError != nil {
		return te.OriginalError.Error()
	}
	return te.Message
}

// String 返回详细的错误信息
func (te *TrackedError) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s\n", te.Timestamp.Format("2006-01-02 15:04:05"), te.Error()))

	if te.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", te.Description))
	}

	sb.WriteString(fmt.Sprintf("Location: %s:%d in %s\n", te.File, te.Line, te.Function))

	if len(te.StackTrace) > 0 {
		sb.WriteString("Stack Trace:\n")
		for i, frame := range te.StackTrace {
			if i >= 10 { // 限制栈深度
				break
			}
			sb.WriteString(fmt.Sprintf("  %d. %s:%d in %s\n", i+1, frame.File, frame.Line, frame.Function))
		}
	}

	return sb.String()
}

// Tracker 错误跟踪器
type Tracker struct {
	DebugMode   bool
	MaxStack    int  // 最大栈深度
	SkipFrames  int  // 跳过的栈帧数
	WriteToFile bool // 是否写入文件
}

// NewTracker 创建新的错误跟踪器
func NewTracker() *Tracker {
	// 从配置中读取 debug 模式
	debugMode, _ := Config.Bool("config.debug")

	return &Tracker{
		DebugMode:   debugMode,
		MaxStack:    10,
		SkipFrames:  2, // 跳过 runtime.Callers 和当前函数
		WriteToFile: true,
	}
}

// DefaultTracker 默认的全局跟踪器实例
var DefaultTracker = NewTracker()

// getCallerInfo 获取调用者信息
func (t *Tracker) getCallerInfo(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0, "unknown"
	}

	func_name := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		func_name = fn.Name()
		// 简化函数名，只保留包名和函数名
		if lastSlash := strings.LastIndex(func_name, "/"); lastSlash >= 0 {
			func_name = func_name[lastSlash+1:]
		}
	}

	// 简化文件路径，只保留相对路径
	if lastSlash := strings.LastIndex(file, "/"); lastSlash >= 0 {
		file = file[lastSlash+1:]
	}

	return file, line, func_name
}

// getStackTrace 获取调用栈
func (t *Tracker) getStackTrace() []StackFrame {
	var frames []StackFrame

	pcs := make([]uintptr, t.MaxStack)
	n := runtime.Callers(t.SkipFrames+1, pcs)

	for i := 0; i < n; i++ {
		pc := pcs[i]
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		file, line := fn.FileLine(pc)
		funcName := fn.Name()

		// 简化路径和函数名
		if lastSlash := strings.LastIndex(file, "/"); lastSlash >= 0 {
			file = file[lastSlash+1:]
		}
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
			funcName = funcName[lastSlash+1:]
		}

		frames = append(frames, StackFrame{
			File:     file,
			Line:     line,
			Function: funcName,
		})
	}

	return frames
}

// Error 记录错误信息
func (t *Tracker) Error(err error, desc ...string) error {
	if err == nil {
		return nil
	}

	description := ""
	if len(desc) > 0 {
		description = desc[0]
	}

	file, line, function := t.getCallerInfo(2)

	trackedErr := &TrackedError{
		OriginalError: err,
		Message:       err.Error(),
		Description:   description,
		File:          file,
		Line:          line,
		Function:      function,
		Timestamp:     time.Now(),
		StackTrace:    t.getStackTrace(),
	}

	// 记录到日志
	logMsg := trackedErr.String()
	if Error != nil {
		Error.Print(logMsg)
	} else {
		fmt.Print(logMsg)
	}

	return trackedErr
}

// Warn 记录警告信息
func (t *Tracker) Warn(msg string, desc ...string) {
	description := ""
	if len(desc) > 0 {
		description = desc[0]
	}

	file, line, function := t.getCallerInfo(2)

	logMsg := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	if description != "" {
		logMsg += fmt.Sprintf("Description: %s\n", description)
	}
	logMsg += fmt.Sprintf("Location: %s:%d in %s\n", file, line, function)

	if t.DebugMode {
		logMsg += "Stack Trace:\n"
		frames := t.getStackTrace()
		for i, frame := range frames {
			if i >= 5 { // 警告信息栈深度较浅
				break
			}
			logMsg += fmt.Sprintf("  %d. %s:%d in %s\n", i+1, frame.File, frame.Line, frame.Function)
		}
	}

	if Warn != nil {
		Warn.Print(logMsg)
	} else {
		fmt.Print(logMsg)
	}
}

// Debug 记录调试信息
func (t *Tracker) Debug(msg string, desc ...string) {
	if !t.DebugMode {
		return
	}

	description := ""
	if len(desc) > 0 {
		description = desc[0]
	}

	file, line, function := t.getCallerInfo(2)

	logMsg := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	if description != "" {
		logMsg += fmt.Sprintf("Description: %s\n", description)
	}
	logMsg += fmt.Sprintf("Location: %s:%d in %s\n", file, line, function)

	if Debug != nil {
		Debug.Print(logMsg)
	} else {
		fmt.Print(logMsg)
	}
}

// Info 记录信息
func (t *Tracker) Info(msg string, desc ...string) {
	description := ""
	if len(desc) > 0 {
		description = desc[0]
	}

	file, line, function := t.getCallerInfo(2)

	logMsg := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	if description != "" {
		logMsg += fmt.Sprintf("Description: %s\n", description)
	}
	logMsg += fmt.Sprintf("Location: %s:%d in %s\n", file, line, function)

	if Info != nil {
		Info.Print(logMsg)
	} else {
		fmt.Print(logMsg)
	}
}

// 全局便捷函数

// TrackError 全局错误跟踪函数
func TrackError(err error, desc ...string) error {
	return DefaultTracker.Error(err, desc...)
}

// TrackWarn 全局警告跟踪函数
func TrackWarn(msg string, desc ...string) {
	DefaultTracker.Warn(msg, desc...)
}

// TrackDebug 全局调试跟踪函数
func TrackDebug(msg string, desc ...string) {
	DefaultTracker.Debug(msg, desc...)
}

// TrackInfo 全局信息跟踪函数
func TrackInfo(msg string, desc ...string) {
	DefaultTracker.Info(msg, desc...)
}

// SetTrackerDebug 设置全局跟踪器的调试模式
func SetTrackerDebug(debug bool) {
	DefaultTracker.DebugMode = debug
}

// GetTrackedErrorDetails 获取跟踪错误的详细信息
func GetTrackedErrorDetails(err error) *TrackedError {
	if trackedErr, ok := err.(*TrackedError); ok {
		return trackedErr
	}
	return nil
}

// IsTrackedError 判断是否为跟踪错误
func IsTrackedError(err error) bool {
	_, ok := err.(*TrackedError)
	return ok
}
