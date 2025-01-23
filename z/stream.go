package z

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type streamSender struct {
	context *gin.Context
	flusher http.Flusher
}

var StreamSender streamSender

func (e *streamSender) New(c *gin.Context) {
	e.context = c
}

func (e *streamSender) IsInstance() bool {
	return e.context != nil
}

// Header 输出流式头部
func (e *streamSender) Header(mode string) {
	if flusher, ok := e.context.Writer.(http.Flusher); !ok {
		Error.Println("StreamHeader error: not support flusher")
	} else {
		e.flusher = flusher
	}

	if strings.HasPrefix(mode, "text/") {
		mode = strings.TrimPrefix(mode, "text/")
	}
	e.context.Writer.Header().Set("Content-Type", fmt.Sprintf("text/%s", mode))
	e.context.Writer.Header().Set("Cache-Control", "no-cache")
	e.context.Writer.Header().Set("Connection", "keep-alive")
	e.context.Writer.Header().Set("X-Accel-Buffering", "no")
}

// Send 输出流式数据
func (e *streamSender) Send(event string, message string) {
	data := fmt.Sprintf("event: %s\ndata: %s\n\n", event, message)
	if _, err := e.context.Writer.WriteString(data); err != nil {
		Error.Println("StreamSend error:", err)
	}
	e.flusher.Flush()
}

// Message 输出流式消息
func (e *streamSender) Message(message string) {
	e.Send("message", message)
}

// StreamSendError 输出流式错误信息
func (e *streamSender) Error(message string) {
	e.Send("error", message)
	e.Done("")
}

// Done 输出流式完成信息
func (e *streamSender) Done(message string) {
	e.Send("done", message)
}
