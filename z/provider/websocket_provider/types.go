package websocket_provider

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ConnectionInfo 连接信息结构体
type ConnectionInfo struct {
	ChannelID string `json:"channel_id"` // 频道ID
	ClientID  string `json:"client_id"`  // 客户端ID
}

// WebSocketHandler WebSocket消息处理器接口
type WebSocketHandler interface {
	// OnConnected 连接建立时调用，返回连接信息和错误
	OnConnected(c *gin.Context, clientIP string) (*ConnectionInfo, error)

	// OnClosed 连接关闭时调用
	OnClosed(c *gin.Context, channelID, clientID string)

	// OnMessage 收到消息时调用
	OnMessage(c *gin.Context, channelID, clientID string, message *Message) error
}

// Message WebSocket消息结构体
type Message struct {
	MessageID string      `json:"message_id"` // 消息ID
	Event     string      `json:"event"`      // 事件类型
	Content   interface{} `json:"content"`    // 消息内容
	Timestamp int64       `json:"timestamp"`  // 时间戳
}

// Connection WebSocket连接信息
type Connection struct {
	Conn         *websocket.Conn // WebSocket连接
	Channel      string          // 频道名称
	ClientID     string          // 客户端ID
	ConnectedAt  time.Time       // 连接建立时间
	LastActivity time.Time       // 最后活跃时间
	SendChan     chan []byte     // 发送消息通道
	Context      *gin.Context    // Gin上下文
}

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	TotalConnections    int                 `json:"total_connections"`     // 总连接数
	ActiveConnections   int                 `json:"active_connections"`    // 当前活跃连接数
	ChannelConnections  map[string]int      `json:"channel_connections"`   // 各频道连接数
	ConnectionsByClient map[string][]string `json:"connections_by_client"` // 按客户端分组的连接
}

// HeartbeatMessage 心跳消息
type HeartbeatMessage struct {
	Type      string `json:"type"`      // ping 或 pong
	Timestamp int64  `json:"timestamp"` // 时间戳
}

// ErrorMessage 错误消息
type ErrorMessage struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
	Details string `json:"details"` // 详细信息
}

// 常量定义
const (
	// 心跳相关
	HeartbeatInterval = 30 * time.Second // 心跳间隔
	ConnectionTimeout = 5 * time.Minute  // 连接超时时间

	// 缓冲区大小
	SendBufferSize  = 1000       // 发送消息缓冲区大小
	ReadBufferSize  = 1024       // 读取缓冲区大小
	WriteBufferSize = 1024       // 写入缓冲区大小
	MaxMessageSize  = 512 * 1024 // 最大消息大小512KB
)
