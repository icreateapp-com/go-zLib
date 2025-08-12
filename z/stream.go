package z

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StreamSender struct {
	Context *gin.Context
	flusher http.Flusher
}

// NewStreamSender SetHeaders 设置响应头
func NewStreamSender(ctx *gin.Context) *StreamSender {
	f, ok := ctx.Writer.(http.Flusher)
	if !ok {
		Error.Println("stream error: not support flusher")
	}

	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("X-Accel-Buffering", "no")
	ctx.Writer.Header().Del("Content-Length")
	ctx.Writer.WriteHeader(http.StatusOK)

	return &StreamSender{
		Context: ctx,
		flusher: f,
	}
}

// writeData 写入数据到响应流
func (e *StreamSender) writeData(data []byte) error {
	if e.Context == nil || e.Context.Writer == nil {
		Error.Println("stream error: context or writer is nil")
		return errors.New("context or writer is nil")
	}
	if cn, ok := e.Context.Writer.(http.CloseNotifier); ok {
		select {
		case <-cn.CloseNotify():
			Error.Println("stream error: client closed connection")
			return errors.New("client closed connection")
		default:
		}
	}
	if _, err := e.Context.Writer.Write(data); err != nil {
		Error.Printf("stream error: write failed: %v", err)
		return err
	}
	return nil
}

// SendMessage 发送普通消息
func (e *StreamSender) SendMessage(message string) {
	if e.Context == nil || e.Context.Writer == nil {
		Error.Println("stream error: context or writer is nil in SendMessage")
		return
	}
	data := []byte("event: message\ndata: " + message + "\n\n")
	if err := e.writeData(data); err != nil {
		Error.Printf("stream error: SendMessage failed: %v", err)
		return
	}
	e.flusher.Flush()
}

// SendError 发送错误消息
func (e *StreamSender) SendError(errMsg string) {
	if e.Context == nil || e.Context.Writer == nil {
		Error.Println("stream error: context or writer is nil in SendError")
		return
	}
	data := []byte("event: error\ndata: " + errMsg + "\n\n")
	if err := e.writeData(data); err != nil {
		Error.Printf("stream error: SendError failed: %v", err)
		return
	}
	e.flusher.Flush()
}

// Done 结束流式响应
func (e *StreamSender) Done() {
	if e.flusher == nil {
		Error.Println("stream error: flusher not initialized")
		return
	}

	if _, err := e.Context.Writer.Write([]byte("\n\n")); err != nil {
		Error.Printf("stream error: %v", err)
		return
	}

	e.flusher.Flush()
	e.Context = nil
	e.flusher = nil
}
