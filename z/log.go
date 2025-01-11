package z

import (
	"github.com/fatih/color"
	"io"
	sysLog "log"
	"os"
)

// 定义日志级别
const (
	DEBUG = 1 << iota
	INFO
	WARN
	ERROR
)

// 定义全局日志变量
var (
	Debug *sysLog.Logger
	Info  *sysLog.Logger
	Warn  *sysLog.Logger
	Error *sysLog.Logger
)

// 定义日志结构体
type _logger struct {
	WriteLogFile bool
	DebugMode    bool
}

// Log 定义全局日志实例
var Log _logger

// Init 初始化日志
func (logger *_logger) Init(writeLogFile bool, debugMode bool) {
	logger.WriteLogFile = writeLogFile
	logger.DebugMode = debugMode
	debugWriters := []io.Writer{&_customWriter{types: DEBUG}}
	infoWriters := []io.Writer{&_customWriter{types: INFO}}
	warnWriters := []io.Writer{&_customWriter{types: WARN}}
	errorWriters := []io.Writer{&_customWriter{types: ERROR}}

	// enable write log ?
	if writeLogFile {
		logFile, err := os.OpenFile(LogPath("debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			color.Red("open log file failed, err:", err)
			return
		}

		// debug mode, only write error log
		if debugMode {
			debugWriters = append(debugWriters, logFile)
			infoWriters = append(infoWriters, logFile)
			warnWriters = append(warnWriters, logFile)
		}
		errorWriters = append(errorWriters, logFile)
	}

	errorFormat := sysLog.Ldate | sysLog.Ltime

	if debugMode {
		errorFormat = sysLog.Lshortfile | sysLog.Ldate | sysLog.Ltime
	}

	Debug = sysLog.New(io.MultiWriter(debugWriters...), "[DEBUG] ", errorFormat)
	Info = sysLog.New(io.MultiWriter(infoWriters...), "[INFO]  ", errorFormat)
	Warn = sysLog.New(io.MultiWriter(warnWriters...), "[WARN]  ", errorFormat)
	Error = sysLog.New(io.MultiWriter(errorWriters...), "[FATAL] ", errorFormat)
}

// 定义自定义日志写入器
type _customWriter struct {
	types int
}

// Write 自定义日志写入方法
func (w _customWriter) Write(data []byte) (n int, err error) {
	if Log.DebugMode && DEBUG == w.types {
		_, _ = color.New(color.FgHiGreen).Print("[DEBUG] ")
		_, _ = color.New(color.FgWhite).Print(string(data[8:]))
	} else if INFO == w.types {
		_, _ = color.New(color.FgHiCyan).Print("[INFO]  ")
		_, _ = color.New(color.FgWhite).Print(string(data[8:]))
	} else if WARN == w.types {
		_, _ = color.New(color.FgHiYellow).Print("[WARN] ")
		_, _ = color.New(color.FgWhite).Print(string(data[8:]))
	} else if ERROR == w.types {
		_, _ = color.New(color.FgHiRed).Print("[ERROR] ")
		_, _ = color.New(color.FgWhite).Print(string(data[8:]))
	}

	return len(data), nil
}
