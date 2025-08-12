# 事件总线提供者 (Event Bus Provider)

事件总线提供者为应用程序提供事件发布订阅、异步处理和解耦通信功能，支持同步和异步事件处理。

## 功能特性

- **事件发布订阅**: 支持事件的发布和订阅机制
- **同步/异步处理**: 支持同步和异步事件处理
- **通配符订阅**: 支持通配符模式的事件订阅
- **事件过滤**: 支持基于条件的事件过滤
- **错误处理**: 内置错误处理和恢复机制
- **性能监控**: 提供事件处理性能监控

## 快速开始

### 1. 基础使用

```go
package main

import (
    "log"
    "time"
    
    "github.com/icreateapp-com/go-zLib/z/provider/event_bus_provider"
)

func main() {
    // 设置事件监听器
    setupEventListeners()
    
    // 发布同步事件
    event_bus_provider.Emit("user.login", map[string]interface{}{
        "user_id": "12345",
        "timestamp": time.Now(),
        "ip": "192.168.1.100",
    })
    
    // 发布异步事件
    event_bus_provider.EmitAsync("order.created", map[string]interface{}{
        "order_id": "order_12345",
        "user_id": "12345",
        "amount": 99.99,
    })
    
    // 保持程序运行
    select {}
}

func setupEventListeners() {
    // 监听用户登录事件
    event_bus_provider.On("user.login", func(event event_bus_provider.Event) {
        data := event.Payload.(map[string]interface{})
        log.Printf("用户登录: %s from %s", data["user_id"], data["ip"])
        
        // 记录登录日志
        recordLoginLog(data)
    })
    
    // 监听订单创建事件
    event_bus_provider.On("order.created", func(event event_bus_provider.Event) {
        data := event.Payload.(map[string]interface{})
        log.Printf("订单创建: %s, 金额: %v", data["order_id"], data["amount"])
        
        // 发送确认邮件
        sendOrderConfirmationEmail(data)
        
        // 更新库存
        updateInventory(data)
    })
}
```

### 2. 通配符订阅

```go
func setupWildcardListeners() {
    // 监听所有用户相关事件
    event_bus_provider.On("user.*", func(event event_bus_provider.Event) {
        log.Printf("用户事件: %s", event.Name)
        
        // 统一的用户事件处理
        handleUserEvent(event)
    })
    
    // 监听所有事件（调试用）
    event_bus_provider.On("*", func(event event_bus_provider.Event) {
        log.Printf("事件触发: %s, 数据: %v", event.Name, event.Payload)
    })
    
    // 监听特定模式的事件
    event_bus_provider.On("order.*.completed", func(event event_bus_provider.Event) {
        log.Printf("订单完成事件: %s", event.Name)
        handleOrderCompletion(event)
    })
}
```

## API 参考

### 事件发布

#### Emit(eventName string, payload interface{})

发布同步事件，所有监听器将按顺序执行。

```go
event_bus_provider.Emit("user.registered", map[string]interface{}{
    "user_id": "12345",
    "email": "user@example.com",
    "timestamp": time.Now(),
})
```

#### EmitAsync(eventName string, payload interface{})

发布异步事件，监听器将在独立的goroutine中执行。

```go
event_bus_provider.EmitAsync("email.send", map[string]interface{}{
    "to": "user@example.com",
    "subject": "欢迎注册",
    "template": "welcome",
})
```

#### EmitWithTimeout(eventName string, payload interface{}, timeout time.Duration) error

发布带超时的事件。

```go
err := event_bus_provider.EmitWithTimeout("payment.process", paymentData, 30*time.Second)
if err != nil {
    log.Printf("支付处理超时: %v", err)
}
```

### 事件订阅

#### On(eventName string, handler EventHandler)

订阅事件。

```go
event_bus_provider.On("user.login", func(event event_bus_provider.Event) {
    // 处理用户登录事件
    handleUserLogin(event.Payload)
})
```

#### Once(eventName string, handler EventHandler)

订阅事件，但只执行一次。

```go
event_bus_provider.Once("app.initialized", func(event event_bus_provider.Event) {
    log.Println("应用初始化完成")
    performOneTimeSetup()
})
```

#### Off(eventName string, handler EventHandler)

取消事件订阅。

```go
handler := func(event event_bus_provider.Event) {
    // 事件处理逻辑
}

// 订阅事件
event_bus_provider.On("test.event", handler)

// 取消订阅
event_bus_provider.Off("test.event", handler)
```

#### OffAll(eventName string)

取消指定事件的所有订阅。

```go
event_bus_provider.OffAll("user.login")
```

### 事件过滤

#### OnWithFilter(eventName string, filter EventFilter, handler EventHandler)

带过滤条件的事件订阅。

```go
// 只处理VIP用户的订单事件
event_bus_provider.OnWithFilter("order.created", 
    func(event event_bus_provider.Event) bool {
        data := event.Payload.(map[string]interface{})
        userType, ok := data["user_type"].(string)
        return ok && userType == "vip"
    },
    func(event event_bus_provider.Event) {
        handleVipOrder(event.Payload)
    })
```

## 事件结构

### Event

```go
type Event struct {
    Name      string      // 事件名称
    Payload   interface{} // 事件数据
    Timestamp time.Time   // 事件时间戳
    ID        string      // 事件唯一ID
    Source    string      // 事件来源
}
```

### EventHandler

```go
type EventHandler func(event Event)
```

### EventFilter

```go
type EventFilter func(event Event) bool
```

## 高级用法

### 1. 事件链

```go
func setupEventChain() {
    // 用户注册事件链
    event_bus_provider.On("user.registered", func(event event_bus_provider.Event) {
        data := event.Payload.(map[string]interface{})
        
        // 发送欢迎邮件事件
        event_bus_provider.EmitAsync("email.welcome", data)
        
        // 创建用户资料事件
        event_bus_provider.EmitAsync("profile.create", data)
        
        // 分配默认权限事件
        event_bus_provider.EmitAsync("permission.assign", map[string]interface{}{
            "user_id": data["user_id"],
            "role": "user",
        })
    })
    
    // 邮件发送事件
    event_bus_provider.On("email.welcome", func(event event_bus_provider.Event) {
        data := event.Payload.(map[string]interface{})
        
        // 发送邮件
        if err := sendWelcomeEmail(data); err != nil {
            // 发送邮件失败事件
            event_bus_provider.EmitAsync("email.failed", map[string]interface{}{
                "type": "welcome",
                "user_id": data["user_id"],
                "error": err.Error(),
            })
        } else {
            // 发送邮件成功事件
            event_bus_provider.EmitAsync("email.sent", map[string]interface{}{
                "type": "welcome",
                "user_id": data["user_id"],
            })
        }
    })
}
```

### 2. 事件聚合

```go
type EventAggregator struct {
    events []event_bus_provider.Event
    mutex  sync.Mutex
}

func (ea *EventAggregator) Collect(event event_bus_provider.Event) {
    ea.mutex.Lock()
    defer ea.mutex.Unlock()
    
    ea.events = append(ea.events, event)
}

func (ea *EventAggregator) Process() {
    ea.mutex.Lock()
    events := make([]event_bus_provider.Event, len(ea.events))
    copy(events, ea.events)
    ea.events = ea.events[:0] // 清空
    ea.mutex.Unlock()
    
    // 批量处理事件
    for _, event := range events {
        processAggregatedEvent(event)
    }
}

func setupEventAggregation() {
    aggregator := &EventAggregator{}
    
    // 收集所有用户行为事件
    event_bus_provider.On("user.*", aggregator.Collect)
    
    // 定期处理聚合事件
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            aggregator.Process()
        }
    }()
}
```

### 3. 事件重试机制

```go
func setupEventRetry() {
    event_bus_provider.On("payment.process", func(event event_bus_provider.Event) {
        data := event.Payload.(map[string]interface{})
        
        // 尝试处理支付
        if err := processPayment(data); err != nil {
            retryCount, _ := data["retry_count"].(int)
            
            if retryCount < 3 {
                // 增加重试次数
                data["retry_count"] = retryCount + 1
                data["last_error"] = err.Error()
                
                // 延迟重试
                time.AfterFunc(time.Duration(retryCount+1)*time.Minute, func() {
                    event_bus_provider.EmitAsync("payment.process", data)
                })
                
                log.Printf("支付处理失败，将在 %d 分钟后重试", retryCount+1)
            } else {
                // 重试次数用尽，发送失败事件
                event_bus_provider.EmitAsync("payment.failed", data)
                log.Printf("支付处理最终失败: %v", err)
            }
        } else {
            // 支付成功
            event_bus_provider.EmitAsync("payment.success", data)
        }
    })
}
```

## 错误处理

### 1. 全局错误处理

```go
func setupGlobalErrorHandler() {
    // 监听所有错误事件
    event_bus_provider.On("*.error", func(event event_bus_provider.Event) {
        errorData := event.Payload.(map[string]interface{})
        
        log.Printf("事件处理错误: %s, 错误: %v", 
            event.Name, errorData["error"])
        
        // 发送告警
        sendAlert(event.Name, errorData)
        
        // 记录错误统计
        recordErrorMetrics(event.Name, errorData)
    })
}

// 在事件处理器中使用错误处理
func safeEventHandler(eventName string, handler event_bus_provider.EventHandler) event_bus_provider.EventHandler {
    return func(event event_bus_provider.Event) {
        defer func() {
            if r := recover(); r != nil {
                // 发布错误事件
                event_bus_provider.EmitAsync(eventName+".error", map[string]interface{}{
                    "original_event": event,
                    "error": r,
                    "stack_trace": string(debug.Stack()),
                })
            }
        }()
        
        handler(event)
    }
}
```

### 2. 事件处理超时

```go
func timeoutEventHandler(handler event_bus_provider.EventHandler, timeout time.Duration) event_bus_provider.EventHandler {
    return func(event event_bus_provider.Event) {
        done := make(chan bool, 1)
        
        go func() {
            handler(event)
            done <- true
        }()
        
        select {
        case <-done:
            // 处理完成
        case <-time.After(timeout):
            // 处理超时
            log.Printf("事件处理超时: %s", event.Name)
            
            event_bus_provider.EmitAsync(event.Name+".timeout", map[string]interface{}{
                "original_event": event,
                "timeout": timeout,
            })
        }
    }
}

// 使用超时处理器
func setupTimeoutHandlers() {
    event_bus_provider.On("heavy.task", 
        timeoutEventHandler(func(event event_bus_provider.Event) {
            // 耗时任务处理
            performHeavyTask(event.Payload)
        }, 30*time.Second))
}
```

## 性能优化

### 1. 事件缓冲

```go
type EventBuffer struct {
    buffer   []event_bus_provider.Event
    mutex    sync.Mutex
    maxSize  int
    flushInterval time.Duration
}

func NewEventBuffer(maxSize int, flushInterval time.Duration) *EventBuffer {
    eb := &EventBuffer{
        buffer:        make([]event_bus_provider.Event, 0, maxSize),
        maxSize:       maxSize,
        flushInterval: flushInterval,
    }
    
    // 定期刷新缓冲区
    go eb.startFlushTimer()
    
    return eb
}

func (eb *EventBuffer) Add(event event_bus_provider.Event) {
    eb.mutex.Lock()
    defer eb.mutex.Unlock()
    
    eb.buffer = append(eb.buffer, event)
    
    // 如果缓冲区满了，立即刷新
    if len(eb.buffer) >= eb.maxSize {
        eb.flush()
    }
}

func (eb *EventBuffer) flush() {
    if len(eb.buffer) == 0 {
        return
    }
    
    events := make([]event_bus_provider.Event, len(eb.buffer))
    copy(events, eb.buffer)
    eb.buffer = eb.buffer[:0]
    
    // 批量处理事件
    go eb.processBatch(events)
}

func (eb *EventBuffer) startFlushTimer() {
    ticker := time.NewTicker(eb.flushInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        eb.mutex.Lock()
        eb.flush()
        eb.mutex.Unlock()
    }
}
```

### 2. 事件池

```go
var eventPool = sync.Pool{
    New: func() interface{} {
        return &event_bus_provider.Event{}
    },
}

func getEvent() *event_bus_provider.Event {
    return eventPool.Get().(*event_bus_provider.Event)
}

func putEvent(event *event_bus_provider.Event) {
    // 重置事件
    event.Name = ""
    event.Payload = nil
    event.Timestamp = time.Time{}
    event.ID = ""
    event.Source = ""
    
    eventPool.Put(event)
}

// 优化的事件发布
func EmitOptimized(eventName string, payload interface{}) {
    event := getEvent()
    event.Name = eventName
    event.Payload = payload
    event.Timestamp = time.Now()
    event.ID = generateEventID()
    
    // 处理事件
    processEvent(event)
    
    // 归还到池中
    putEvent(event)
}
```

## 监控和日志

### 1. 事件统计

```go
type EventMetrics struct {
    eventCounts   map[string]int64
    errorCounts   map[string]int64
    processingTime map[string]time.Duration
    mutex         sync.RWMutex
}

func NewEventMetrics() *EventMetrics {
    return &EventMetrics{
        eventCounts:    make(map[string]int64),
        errorCounts:    make(map[string]int64),
        processingTime: make(map[string]time.Duration),
    }
}

func (em *EventMetrics) RecordEvent(eventName string, duration time.Duration) {
    em.mutex.Lock()
    defer em.mutex.Unlock()
    
    em.eventCounts[eventName]++
    em.processingTime[eventName] += duration
}

func (em *EventMetrics) RecordError(eventName string) {
    em.mutex.Lock()
    defer em.mutex.Unlock()
    
    em.errorCounts[eventName]++
}

func (em *EventMetrics) GetStats() map[string]interface{} {
    em.mutex.RLock()
    defer em.mutex.RUnlock()
    
    stats := make(map[string]interface{})
    
    for eventName, count := range em.eventCounts {
        avgTime := em.processingTime[eventName] / time.Duration(count)
        errorCount := em.errorCounts[eventName]
        
        stats[eventName] = map[string]interface{}{
            "count":        count,
            "error_count":  errorCount,
            "avg_duration": avgTime,
            "error_rate":   float64(errorCount) / float64(count),
        }
    }
    
    return stats
}

// 全局指标实例
var globalMetrics = NewEventMetrics()

// 监控事件处理
func monitoredEventHandler(eventName string, handler event_bus_provider.EventHandler) event_bus_provider.EventHandler {
    return func(event event_bus_provider.Event) {
        start := time.Now()
        
        defer func() {
            duration := time.Since(start)
            globalMetrics.RecordEvent(eventName, duration)
            
            if r := recover(); r != nil {
                globalMetrics.RecordError(eventName)
                panic(r) // 重新抛出panic
            }
        }()
        
        handler(event)
    }
}
```

### 2. 事件追踪

```go
type EventTrace struct {
    TraceID   string                 `json:"trace_id"`
    Events    []EventTraceItem       `json:"events"`
    StartTime time.Time              `json:"start_time"`
    EndTime   time.Time              `json:"end_time"`
    Metadata  map[string]interface{} `json:"metadata"`
}

type EventTraceItem struct {
    EventName string      `json:"event_name"`
    Timestamp time.Time   `json:"timestamp"`
    Duration  time.Duration `json:"duration"`
    Success   bool        `json:"success"`
    Error     string      `json:"error,omitempty"`
}

func startEventTrace(traceID string) *EventTrace {
    return &EventTrace{
        TraceID:   traceID,
        Events:    make([]EventTraceItem, 0),
        StartTime: time.Now(),
        Metadata:  make(map[string]interface{}),
    }
}

func (et *EventTrace) AddEvent(eventName string, duration time.Duration, success bool, err error) {
    item := EventTraceItem{
        EventName: eventName,
        Timestamp: time.Now(),
        Duration:  duration,
        Success:   success,
    }
    
    if err != nil {
        item.Error = err.Error()
    }
    
    et.Events = append(et.Events, item)
}

func (et *EventTrace) Finish() {
    et.EndTime = time.Now()
    
    // 保存追踪信息
    saveEventTrace(et)
}
```

## 最佳实践

### 1. 事件命名规范

```go
// 推荐的事件命名规范
const (
    // 用户相关事件
    EventUserRegistered = "user.registered"
    EventUserLogin      = "user.login"
    EventUserLogout     = "user.logout"
    EventUserUpdated    = "user.updated"
    EventUserDeleted    = "user.deleted"
    
    // 订单相关事件
    EventOrderCreated   = "order.created"
    EventOrderPaid      = "order.paid"
    EventOrderShipped   = "order.shipped"
    EventOrderCompleted = "order.completed"
    EventOrderCancelled = "order.cancelled"
    
    // 系统相关事件
    EventSystemStarted  = "system.started"
    EventSystemStopped  = "system.stopped"
    EventSystemError    = "system.error"
)
```

### 2. 事件数据结构

```go
// 标准事件数据结构
type UserEvent struct {
    UserID    string    `json:"user_id"`
    Action    string    `json:"action"`
    Timestamp time.Time `json:"timestamp"`
    IP        string    `json:"ip,omitempty"`
    UserAgent string    `json:"user_agent,omitempty"`
}

type OrderEvent struct {
    OrderID   string    `json:"order_id"`
    UserID    string    `json:"user_id"`
    Amount    float64   `json:"amount"`
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
}

// 使用结构化数据
func emitUserEvent(action string, userID string, ip string) {
    event_bus_provider.EmitAsync(EventUserLogin, UserEvent{
        UserID:    userID,
        Action:    action,
        Timestamp: time.Now(),
        IP:        ip,
    })
}
```

### 3. 事件处理器组织

```go
// 按功能模块组织事件处理器
type UserEventHandlers struct{}

func (h *UserEventHandlers) Register() {
    event_bus_provider.On(EventUserRegistered, h.handleUserRegistered)
    event_bus_provider.On(EventUserLogin, h.handleUserLogin)
    event_bus_provider.On(EventUserLogout, h.handleUserLogout)
}

func (h *UserEventHandlers) handleUserRegistered(event event_bus_provider.Event) {
    // 处理用户注册事件
}

func (h *UserEventHandlers) handleUserLogin(event event_bus_provider.Event) {
    // 处理用户登录事件
}

func (h *UserEventHandlers) handleUserLogout(event event_bus_provider.Event) {
    // 处理用户登出事件
}

// 在应用启动时注册
func initEventHandlers() {
    userHandlers := &UserEventHandlers{}
    userHandlers.Register()
    
    orderHandlers := &OrderEventHandlers{}
    orderHandlers.Register()
}
```

事件总线提供者为应用程序提供了强大的事件驱动架构支持，通过合理使用可以实现松耦合的系统设计和高效的异步处理。