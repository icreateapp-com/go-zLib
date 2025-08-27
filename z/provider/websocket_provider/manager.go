package websocket_provider

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketManager WebSocket连接管理器
type WebSocketManager struct {
	mu            sync.RWMutex           // 读写锁保证线程安全
	connections   map[string]*Connection // sessionID -> Connection
	stats         *ConnectionStats       // 连接统计信息
	handler       WebSocketHandler       // 消息处理器
	upgrader      websocket.Upgrader     // WebSocket升级器
	cleanupTicker *time.Ticker           // 清理定时器
	stopChan      chan struct{}          // 停止信号
}

// 全局单例
var (
	globalManager *WebSocketManager
	managerOnce   sync.Once
)

// GetManager 获取全局WebSocket管理器单例
func GetManager() *WebSocketManager {
	managerOnce.Do(func() {
		globalManager = &WebSocketManager{
			connections: make(map[string]*Connection),
			stats: &ConnectionStats{
				ChannelConnections:  make(map[string]int),
				ConnectionsByClient: make(map[string][]string),
			},
			upgrader: websocket.Upgrader{
				ReadBufferSize:  ReadBufferSize,
				WriteBufferSize: WriteBufferSize,
				Subprotocols:    []string{"chat", "echo", "binary"},
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
				// 启用压缩扩展
				EnableCompression: true,
			},
			cleanupTicker: time.NewTicker(time.Minute),
			stopChan:      make(chan struct{}),
		}
		// 启动清理协程
		go globalManager.cleanupRoutine()
	})
	return globalManager
}

// SetHandler 设置消息处理器
func (m *WebSocketManager) SetHandler(handler WebSocketHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

// HandleWebSocket 处理WebSocket连接升级
func (m *WebSocketManager) HandleWebSocket(c *gin.Context) {
	w := c.Writer
	r := c.Request

	// 记录连接尝试信息
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	protocol := "ws"
	if r.TLS != nil {
		protocol = "wss"
	}
	log.Printf("WebSocket连接尝试: IP=%s, Protocol=%s, UserAgent=%s", clientIP, protocol, userAgent)

	// 升级HTTP连接为WebSocket
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: IP=%s, Protocol=%s, Error=%v", clientIP, protocol, err)
		// 检查常见的升级失败原因
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("WebSocket意外关闭错误: %v", err)
		}
		return
	}
	defer conn.Close()

	// 记录成功连接信息
	log.Printf("WebSocket连接成功: IP=%s, Protocol=%s, Subprotocol=%s", clientIP, protocol, conn.Subprotocol())

	// 生成会话ID
	sessionID := uuid.New().String()
	// clientIP 已在上面定义，这里不需要重新声明

	// 调用业务层连接处理
	var connInfo *ConnectionInfo
	if m.handler != nil {
		var err error
		connInfo, err = m.handler.OnConnected(c, sessionID, clientIP)
		if err != nil {
			// 业务层拒绝连接
			log.Printf("连接被拒绝: sessionID=%s, error=%v", sessionID, err)
			m.sendErrorAndClose(conn, "Connection rejected: "+err.Error())
			return
		}
	}

	// 设置默认连接信息
	channelID := "default"
	clientID := clientIP
	if connInfo != nil {
		if connInfo.ChannelID != "" {
			channelID = connInfo.ChannelID
		}
		if connInfo.ClientID != "" {
			clientID = connInfo.ClientID
		}
	}

	// 创建连接对象
	connection := &Connection{
		SessionID:    sessionID,
		Conn:         conn,
		Channel:      channelID,
		ClientID:     clientID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		SendChan:     make(chan []byte, SendBufferSize),
		Context:      c, // 存储gin.Context
	}

	// 添加到连接池
	m.addConnection(sessionID, connection)
	defer m.removeConnection(sessionID)

	// 启动连接处理协程
	go m.handleSend(connection)
	go m.handleHeartbeat(connection)

	// 处理消息接收
	m.handleReceive(connection)
}

// addConnection 添加连接
func (m *WebSocketManager) addConnection(sessionID string, conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections[sessionID] = conn
	m.updateStats()

	log.Printf("WebSocket连接建立: sessionID=%s, clientIP=%s", sessionID, conn.ClientID)
}

// removeConnection 移除连接
func (m *WebSocketManager) removeConnection(sessionID string) {
	m.mu.Lock()
	conn, exists := m.connections[sessionID]
	if exists {
		delete(m.connections, sessionID)
		close(conn.SendChan)
	}
	m.mu.Unlock()

	if exists {
		m.updateStats()
		if m.handler != nil {
			m.handler.OnClosed(conn.Context, sessionID)
		}
		log.Printf("WebSocket连接关闭: sessionID=%s", sessionID)
	}
}

// SendMessage 发送消息到指定客户端
func (m *WebSocketManager) SendMessage(clientID string, event string, content interface{}) error {
	// 根据 clientID 查找对应的会话
	var targetSessionID string
	m.mu.RLock()
	for sessionID, conn := range m.connections {
		if conn.ClientID == clientID {
			targetSessionID = sessionID
			break
		}
	}
	m.mu.RUnlock()

	if targetSessionID == "" {
		return fmt.Errorf("client not found: %s", clientID)
	}

	// 构建 Message 结构体
	message := &Message{
		SessionID: targetSessionID,
		MessageID: uuid.New().String(),
		Event:     event,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	m.mu.RLock()
	conn, exists := m.connections[targetSessionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", targetSessionID)
	}

	select {
	case conn.SendChan <- msgBytes:
		return nil
	default:
		return fmt.Errorf("send buffer full for session: %s", targetSessionID)
	}
}

// Broadcast 广播消息到指定频道
func (m *WebSocketManager) Broadcast(channelID string, event string, content interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, conn := range m.connections {
		if conn.Channel == channelID {
			// 为每个连接构建独立的 Message 结构体
			message := &Message{
				SessionID: conn.SessionID,
				MessageID: uuid.New().String(),
				Event:     event,
				Content:   content,
				Timestamp: time.Now().Unix(),
			}

			// 序列化消息
			msgBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("序列化消息失败: %v", err)
				continue
			}

			select {
			case conn.SendChan <- msgBytes:
				count++
			default:
				log.Printf("发送缓冲区满，跳过会话: %s", conn.SessionID)
			}
		}
	}

	log.Printf("广播消息到频道 %s，成功发送 %d 个连接", channelID, count)
	return nil
}

// GetStats 获取连接统计
func (m *WebSocketManager) GetStats() *ConnectionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 复制统计信息
	stats := &ConnectionStats{
		TotalConnections:    m.stats.TotalConnections,
		ActiveConnections:   len(m.connections),
		ChannelConnections:  make(map[string]int),
		ConnectionsByClient: make(map[string][]string),
	}

	// 统计各频道连接数
	for _, conn := range m.connections {
		stats.ChannelConnections[conn.Channel]++
		stats.ConnectionsByClient[conn.ClientID] = append(
			stats.ConnectionsByClient[conn.ClientID], conn.SessionID)
	}

	return stats
}

// handleReceive 处理消息接收
func (m *WebSocketManager) handleReceive(conn *Connection) {
	conn.Conn.SetReadLimit(MaxMessageSize)
	conn.Conn.SetReadDeadline(time.Now().Add(ConnectionTimeout))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(ConnectionTimeout))
		conn.LastActivity = time.Now()
		return nil
	})

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket读取错误: %v", err)
			}
			break
		}

		conn.LastActivity = time.Now()

		// 处理ping/pong消息
		if m.handlePingPong(conn, message) {
			continue
		}

		// 解析消息为 Message 结构体
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("消息解析错误: sessionID=%s, error=%v", conn.SessionID, err)
			continue
		}

		// 调用业务层消息处理
		if m.handler != nil {
			if err := m.handler.OnMessage(conn.Context, conn.SessionID, &msg); err != nil {
				log.Printf("消息处理错误: sessionID=%s, error=%v", conn.SessionID, err)
			}
		}
	}
}

// handleSend 处理消息发送
func (m *WebSocketManager) handleSend(conn *Connection) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-conn.SendChan:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("发送消息失败: %v", err)
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleHeartbeat 处理心跳检测
func (m *WebSocketManager) handleHeartbeat(conn *Connection) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(conn.LastActivity) > ConnectionTimeout {
				log.Printf("连接超时，关闭连接: sessionID=%s", conn.SessionID)
				conn.Conn.Close()
				return
			}
		case <-m.stopChan:
			return
		}
	}
}

// handlePingPong 处理ping/pong消息
func (m *WebSocketManager) handlePingPong(conn *Connection, message []byte) bool {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		return false
	}

	if msg.Event == "ping" {
		// 回复pong消息
		pongMsg := Message{
			SessionID: conn.SessionID,
			MessageID: fmt.Sprintf("pong_%d", time.Now().UnixNano()),
			Event:     "pong",
			Content:   msg.Content,
			Timestamp: time.Now().Unix(),
		}

		if msgBytes, err := json.Marshal(pongMsg); err == nil {
			select {
			case conn.SendChan <- msgBytes:
			default:
				log.Printf("发送pong消息失败，缓冲区满: sessionID=%s", conn.SessionID)
			}
		}
		return true
	}

	return false
}

// sendErrorAndClose 发送错误消息并关闭连接
func (m *WebSocketManager) sendErrorAndClose(conn *websocket.Conn, errorMsg string) {
	errorResponse := ErrorMessage{
		Code:    4000,
		Message: "Connection Error",
		Details: errorMsg,
	}

	if msgBytes, err := json.Marshal(errorResponse); err == nil {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		conn.WriteMessage(websocket.TextMessage, msgBytes)
	}

	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4000, errorMsg))
}

// updateStats 更新统计信息
func (m *WebSocketManager) updateStats() {
	m.stats.ActiveConnections = len(m.connections)
	m.stats.TotalConnections = len(m.connections) // 简化实现
}

// cleanupRoutine 清理协程
func (m *WebSocketManager) cleanupRoutine() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.cleanupTimeoutConnections()
		case <-m.stopChan:
			return
		}
	}
}

// cleanupTimeoutConnections 清理超时连接
func (m *WebSocketManager) cleanupTimeoutConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var timeoutSessions []string

	for sessionID, conn := range m.connections {
		if now.Sub(conn.LastActivity) > ConnectionTimeout {
			timeoutSessions = append(timeoutSessions, sessionID)
		}
	}

	// 关闭超时连接
	for _, sessionID := range timeoutSessions {
		if conn, exists := m.connections[sessionID]; exists {
			log.Printf("清理超时连接: sessionID=%s", sessionID)
			conn.Conn.Close()
		}
	}
}

// Stop 停止管理器
func (m *WebSocketManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	close(m.stopChan)
	m.cleanupTicker.Stop()

	// 关闭所有连接
	for _, conn := range m.connections {
		conn.Conn.Close()
	}

	log.Printf("WebSocket管理器已停止")
}

// getClientIP 获取客户端IP地址
func getClientIP(r *http.Request) string {
	// 尝试从 X-Forwarded-For 头获取
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// 尝试从 X-Real-IP 头获取
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// 从 RemoteAddr 获取
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}
