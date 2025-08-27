# WebSocket Provider 使用文档

## 概述

WebSocket Provider 是基于 gin 框架的 WebSocket 服务提供者，提供了完整的 WebSocket 连接管理、消息处理和事件处理功能。

## 特性

- 基于 gin 框架，无需独立的 HTTP 服务器
- 支持自定义消息处理器
- 提供连接统计和健康检查接口
- 线程安全的连接管理
- 自动心跳检测和超时处理
- 支持频道广播功能

## 快速开始

### 1. 初始化服务

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/provider/websocket_provider"
)

func main() {
    // 初始化 WebSocket Provider
    websocket_provider.WebSocketProvider.Init()
    
    // 创建 gin 路由
    r := gin.Default()
    
    // 注册 WebSocket 路由
    r.GET("/ws", websocket_provider.WebSocketProvider.HandleWebSocket)
    
    r.Run(":8080")
}
```

### 2. 自定义消息处理器

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/provider/websocket_provider"
)

// 自定义消息处理器
type CustomHandler struct {
    manager *websocket_provider.WebSocketManager
}

func NewCustomHandler() *CustomHandler {
    return &CustomHandler{}
}

func (h *CustomHandler) SetManager(manager *websocket_provider.WebSocketManager) {
    h.manager = manager
}

func (h *CustomHandler) OnConnected(c *gin.Context, sessionID string, clientIP string) (*websocket_provider.ConnectionInfo, error) {
    log.Printf("新连接: sessionID=%s, clientIP=%s", sessionID, clientIP)
    
    // 根据业务逻辑设置频道和客户端ID
    channelID := "general" // 可以根据用户信息或请求参数设置
    clientID := fmt.Sprintf("client_%s", clientIP)
    
    // 发送欢迎消息
    err := h.manager.SendMessage(clientID, "welcome", map[string]interface{}{
        "message": "欢迎连接到服务器",
        "session": sessionID,
        "channel": channelID,
    })
    if err != nil {
        log.Printf("发送欢迎消息失败: %v", err)
    }
    
    return &websocket_provider.ConnectionInfo{
        ChannelID: channelID,
        ClientID:  clientID,
    }, nil
}

func (h *CustomHandler) OnClosed(c *gin.Context, sessionID string) {
    log.Printf("连接关闭: sessionID=%s", sessionID)
}

func (h *CustomHandler) OnMessage(c *gin.Context, sessionID string, message *websocket_provider.Message) error {
    log.Printf("收到消息: sessionID=%s, event=%s, content=%v", sessionID, message.Event, message.Content)
    
    // 根据消息事件类型处理
    switch message.Event {
    case "chat":
        // 处理聊天消息
        h.handleChatMessage(sessionID, message)
    case "join_room":
        // 处理加入房间
        h.handleJoinRoom(sessionID, message)
    default:
        // 回显消息
        if msgBytes, err := json.Marshal(message); err == nil {
            h.manager.SendMessage(sessionID, msgBytes)
        }
    }
    
    return nil
}

func (h *CustomHandler) handleChatMessage(sessionID string, message *websocket_provider.Message) {
    // 处理聊天消息
    if content, ok := message.Content.(map[string]interface{}); ok {
        if channel, ok := content["channel"].(string); ok {
            // 广播聊天消息到指定频道
            chatMsg := &websocket_provider.Message{
                SessionID: sessionID,
                MessageID: fmt.Sprintf("chat_%d", time.Now().UnixNano()),
                Event:     "chat",
                Content: map[string]interface{}{
                    "from":      sessionID,
                    "message":   content["message"],
                    "timestamp": time.Now().Unix(),
                },
                Timestamp: time.Now().Unix(),
            }
            
            h.manager.Broadcast(channel, "chat", chatMsg.Content)
        } else {
            // 发送个人回复消息
            response := &websocket_provider.Message{
                SessionID: sessionID,
                MessageID: fmt.Sprintf("response_%d", time.Now().UnixNano()),
                Event:     "chat_response",
                Content: map[string]interface{}{
                    "original": message.Content,
                    "reply":    "收到您的消息",
                },
                Timestamp: time.Now().Unix(),
            }
            
            h.manager.SendMessage(sessionID, "chat_response", response.Content)
        }
    }
}

func (h *CustomHandler) handleJoinRoom(sessionID string, message *websocket_provider.Message) {
    // 处理加入房间请求
    if content, ok := message.Content.(map[string]interface{}); ok {
        if roomID, ok := content["room_id"].(string); ok {
            // 这里可以添加房间管理逻辑
            log.Printf("用户 %s 加入房间 %s", sessionID, roomID)
            
            // 发送确认消息
            response := &websocket_provider.Message{
                SessionID: sessionID,
                MessageID: fmt.Sprintf("join_%d", time.Now().UnixNano()),
                Event:     "room_joined",
                Content: map[string]interface{}{
                    "room_id": roomID,
                    "status":  "success",
                },
                Timestamp: time.Now().Unix(),
            }
            
            h.manager.SendMessage(sessionID, "room_joined", response.Content)
        }
    }
}

func (h *CustomHandler) handlePing(sessionID string, message *websocket_provider.Message) {
    // 处理心跳消息
    pong := &websocket_provider.Message{
        SessionID: sessionID,
        MessageID: fmt.Sprintf("pong_%d", time.Now().UnixNano()),
        Event:     "pong",
        Content: map[string]interface{}{
            "timestamp": time.Now().Unix(),
        },
        Timestamp: time.Now().Unix(),
    }
    
    h.manager.SendMessage(sessionID, "pong", pong.Content)
}

func main() {
    // 初始化 WebSocket Provider
    websocket_provider.WebSocketProvider.Init()
    
    // 设置自定义处理器
    customHandler := NewCustomHandler()
    websocket_provider.WebSocketProvider.SetHandler(customHandler)
    
    // 其他初始化代码...
}
```

## API 接口

### WebSocket 连接
- **路径**: `/ws`
- **方法**: GET (WebSocket 升级)
- **描述**: WebSocket 连接端点

### 统计信息
- **路径**: `/ws/stats`
- **方法**: GET
- **描述**: 获取连接统计信息
- **响应**:
```json
{
  "success": true,
  "data": {
    "active_connections": 10,
    "total_connections": 100,
    "total_messages": 1000
  }
}
```

### 健康检查
- **路径**: `/ws/health`
- **方法**: GET
- **描述**: 服务健康检查
- **响应**:
```json
{
  "success": true,
  "status": "healthy",
  "connections": 10
}
```

## 消息格式

### 默认支持的消息类型

1. **Echo 消息**
```json
{
  "event": "echo",
  "content": {
    "message": "Hello World"
  }
}
```

2. **广播消息**
```json
{
  "event": "broadcast",
  "content": {
    "message": "广播内容",
    "channel": "room1"
  }
}
```

3. **心跳消息**
```json
{
  "event": "ping",
  "content": {}
}
```

## API 方法

### WebSocketProvider 方法

- `Init()` - 初始化WebSocket提供者
- `SetHandler(handler WebSocketHandler)` - 设置消息处理器
- `GetManager() *WebSocketManager` - 获取WebSocket管理器
- `HandleWebSocket(c *gin.Context)` - 处理WebSocket连接请求（Gin路由）
- `Register(handler WebSocketHandler) http.Handler` - 注册HTTP处理器（标准HTTP）
- `Shutdown()` - 关闭WebSocket提供者

### WebSocketManager 方法

- `HandleWebSocket(c *gin.Context)` - 处理WebSocket连接升级
- `SendMessage(clientID string, event string, content interface{}) error` - 发送消息到指定客户端
- `Broadcast(channel string, event string, content interface{})` - 广播消息到指定频道
- `GetStats() *ConnectionStats` - 获取连接统计信息
- `Stop()` - 停止管理器

### WebSocketHandler 接口

- `OnConnected(c *gin.Context, sessionID string, clientIP string) (*ConnectionInfo, error)` - 连接建立时调用
- `OnMessage(c *gin.Context, sessionID string, message *Message) error` - 收到消息时调用
- `OnClosed(c *gin.Context, sessionID string)` - 连接关闭时调用

## 数据结构

### ConnectionInfo 结构体
```go
type ConnectionInfo struct {
    ChannelID string `json:"channel_id"` // 频道ID
    ClientID  string `json:"client_id"`  // 客户端ID
}
```

### Message 结构体
```go
type Message struct {
    SessionID string      `json:"session_id"` // 会话ID
    MessageID string      `json:"message_id"` // 消息ID
    Event     string      `json:"event"`      // 事件类型
    Content   interface{} `json:"content"`    // 消息内容
    Timestamp int64       `json:"timestamp"`  // 时间戳
}
```

### Connection 结构体
```go
type Connection struct {
    Conn         *websocket.Conn // WebSocket连接
    SessionID    string          // 会话ID
    Channel      string          // 频道名称
    ClientID     string          // 客户端IP地址
    ConnectedAt  time.Time       // 连接建立时间
    LastActivity time.Time       // 最后活跃时间
    SendChan     chan []byte     // 发送消息通道
    Context      *gin.Context    // Gin上下文
}
```

### ConnectionStats 结构体
```go
type ConnectionStats struct {
    TotalConnections    int                 `json:"total_connections"`     // 总连接数
    ActiveConnections   int                 `json:"active_connections"`    // 当前活跃连接数
    ChannelConnections  map[string]int      `json:"channel_connections"`   // 各频道连接数
    ConnectionsByClient map[string][]string `json:"connections_by_client"` // 按客户端分组的连接
}
```

### 配置常量
```go
const (
    HeartbeatInterval = 30 * time.Second // 心跳间隔
    ConnectionTimeout = 5 * time.Minute  // 连接超时时间
    SendBufferSize    = 1000             // 发送消息缓冲区大小
    ReadBufferSize    = 1024             // 读取缓冲区大小
    WriteBufferSize   = 1024             // 写入缓冲区大小
    MaxMessageSize    = 512 * 1024       // 最大消息大小512KB
)
```

## 编程接口

### 发送消息
```go
// 发送消息到指定客户端
err := websocket_provider.WebSocketProvider.GetManager().SendMessage(clientID, "event_type", messageContent)

// 广播消息到指定频道
websocket_provider.WebSocketProvider.GetManager().Broadcast(channel, "event_type", messageContent)
```

### 获取统计信息
```go
stats := websocket_provider.WebSocketProvider.GetManager().GetStats()
fmt.Printf("总连接数: %d\n", stats.TotalConnections)
fmt.Printf("活跃连接数: %d\n", stats.ActiveConnections)
fmt.Printf("频道连接数: %+v\n", stats.ChannelConnections)
fmt.Printf("客户端连接: %+v\n", stats.ConnectionsByClient)
```

## 特性说明

### 主要功能
- **连接管理**: 自动管理 WebSocket 连接的生命周期
- **消息路由**: 基于事件类型的消息路由机制
- **频道支持**: 支持按频道分组管理连接
- **心跳检测**: 内置心跳机制保持连接活跃
- **统计监控**: 提供详细的连接统计信息
- **错误处理**: 完善的错误处理和异常恢复机制

### 技术特点
- **高性能**: 基于 Gorilla WebSocket 实现，支持高并发
- **易于集成**: 与 Gin 框架无缝集成
- **灵活扩展**: 通过接口设计支持自定义消息处理逻辑
- **线程安全**: 内置并发安全机制
- **资源管理**: 自动清理断开的连接和相关资源

### 适用场景
- 实时聊天应用
- 在线游戏
- 实时数据推送
- 协作编辑工具
- 监控面板
- 直播弹幕系统

## 事件系统

WebSocket Provider 会发布以下事件到事件总线：

- `websocket.provider.initialized`: 服务初始化完成
- `websocket.provider.shutdown`: 服务关闭

## 注意事项

1. 必须先调用 `Init()` 方法初始化服务
2. 自定义处理器需要实现 `WebSocketHandler` 接口
3. 消息处理应该是非阻塞的，避免影响其他连接
4. 建议在生产环境中实现适当的错误处理和日志记录
5. WebSocket 连接会自动进行心跳检测，超时连接会被自动清理