package websocket_provider

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/event_bus_provider"
)

// WebSocketProvider WebSocket服务提供者
type webSocketProvider struct {
	manager *WebSocketManager  // WebSocket连接管理器
	handler WebSocketHandler   // 消息处理器
	enabled bool               // 是否启用
	mutex   sync.RWMutex       // 读写锁
	ctx     context.Context    // 上下文
	cancel  context.CancelFunc // 取消函数
}

// WebSocketProvider 全局WebSocket提供者实例
var WebSocketProvider webSocketProvider

// Init 初始化WebSocket提供者
func (p *webSocketProvider) Init() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 如果已经初始化，直接返回
	if p.enabled {
		return
	}

	// 创建上下文
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// 获取WebSocket管理器实例
	p.manager = GetManager()

	// 如果已经设置了处理器，将其传递给 manager
	if p.handler != nil {
		p.manager.SetHandler(p.handler)
	}

	// 发布初始化完成事件
	event_bus_provider.Emit(p.ctx, "websocket.provider.initialized", map[string]interface{}{
		"enabled": true,
	})

	p.enabled = true
}

// SetHandler 设置自定义消息处理器
func (p *webSocketProvider) SetHandler(handler WebSocketHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.handler = handler
	// 如果 manager 已经初始化，立即设置处理器
	if p.manager != nil && handler != nil {
		p.manager.SetHandler(handler)
	}
}

// GetManager 获取WebSocket管理器
func (p *webSocketProvider) GetManager() *WebSocketManager {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.manager
}

// Register 注册自定义处理器并返回http.Handler，用于gin路由绑定
func (p *webSocketProvider) Register(handler WebSocketHandler) http.Handler {
	p.SetHandler(handler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.enabled {
			http.Error(w, "WebSocket service not initialized", http.StatusServiceUnavailable)
			return
		}
		if p.handler == nil {
			http.Error(w, "WebSocket handler not set", http.StatusServiceUnavailable)
			return
		}
		// 创建临时的 gin.Context（这不是最佳实践，但为了兼容性）
		ginContext, _ := gin.CreateTestContext(w)
		ginContext.Request = r
		p.manager.HandleWebSocket(ginContext)
	})
}

// HandleWebSocket 提供gin路由处理函数
func (p *webSocketProvider) HandleWebSocket(c *gin.Context) {
	if !p.enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket service not initialized"})
		return
	}
	if p.handler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket handler not set"})
		return
	}
	p.manager.HandleWebSocket(c)
}

// Shutdown 关闭WebSocket提供者
func (p *webSocketProvider) Shutdown() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.enabled {
		return
	}

	z.Info.Println("正在关闭WebSocket Provider...")

	// 停止WebSocket管理器
	if p.manager != nil {
		p.manager.Stop()
	}

	// 取消上下文
	if p.cancel != nil {
		p.cancel()
	}

	// 发布关闭事件
	event_bus_provider.Emit(context.Background(), "websocket.provider.shutdown", map[string]interface{}{
		"enabled": false,
	})

	p.enabled = false
	z.Info.Println("WebSocket Provider 已关闭")
}
