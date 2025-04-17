package z

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type StreamSender struct {
	Context *gin.Context
	flusher http.Flusher
}

// NewStreamSender 初始化流式服务，支持 event-stream 模式
func NewStreamSender(c *gin.Context) *StreamSender {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Printf("Error: stream error - not support flusher")
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	return &StreamSender{
		Context: c,
		flusher: flusher,
	}
}

// writeData 写入数据到响应流
func (e *StreamSender) writeData(data []byte) error {
	if _, err := e.Context.Writer.Write(data); err != nil {
		log.Printf("Error: stream write error - %v", err)
		return err
	}
	return nil
}

// SendMessage 发送普通消息
func (e *StreamSender) SendMessage(message string) {
	if e.flusher == nil {
		log.Printf("Error: stream error - flusher not initialized")
		return
	}

	// 优化写入逻辑，减少多次调用
	data := []byte("data: " + message + "\n\n")
	if err := e.writeData(data); err != nil {
		return
	}

	e.flusher.Flush()
}

// SendError 发送错误消息
func (e *StreamSender) SendError(errMsg string) {
	if e.flusher == nil {
		log.Printf("Error: stream error - flusher not initialized")
		return
	}

	// 优化写入逻辑，减少多次调用
	data := []byte("event: error\ndata: " + errMsg + "\n\n")
	if err := e.writeData(data); err != nil {
		return
	}

	e.flusher.Flush()
}

// SendDone 结束流式响应
func (e *StreamSender) SendDone() {
	if e.flusher == nil {
		log.Printf("Error: stream error - flusher not initialized")
		return
	}

	// 确保结束时的空行符合标准
	if err := e.writeData([]byte("\n\n")); err != nil {
		return
	}

	e.flusher.Flush()
}
