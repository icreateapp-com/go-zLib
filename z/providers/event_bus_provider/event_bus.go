package event_bus_provider

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"go.uber.org/fx"
)

type EventBus = eventBusProvider[any]

// NewEventBusProvider 创建 EventBusProvider 实例（fx Provider）。
func NewEventBusProvider(lc fx.Lifecycle, log *logger_provider.Logger) *EventBus {
	bus := NewEventBus[any]().WithLogger(log)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if log != nil {
				log.Infow("provider[event_bus] enabled")
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			bus.Clear()
			if log != nil {
				log.Infow("provider[event_bus] stopped")
			}
			return nil
		},
	})

	return bus
}

// EventBusProviderModule 提供 EventBusProvider 的 fx 模块。
var EventBusProviderModule = fx.Options(
	fx.Provide(NewEventBusProvider),
)

// Event 表示一个泛型事件
type Event[T any] struct {
	Name    string          // 事件名称
	Payload T               // 事件载荷（泛型）
	Context context.Context // 上下文
}

// Listener 是一个泛型处理函数
type Listener[T any] func(ctx context.Context, event Event[T])

// listenerWrapper 泛型监听器包装器，包含ID和处理函数
type listenerWrapper[T any] struct {
	id       uint64      // 监听器唯一ID
	listener Listener[T] // 泛型处理函数
}

// eventBusProvider 泛型事件总线提供者
type eventBusProvider[T any] struct {
	listeners map[string][]*listenerWrapper[T] // 事件名称 -> 泛型监听器包装器列表
	lock      sync.RWMutex                     // 读写锁
	nextID    uint64                           // 下一个监听器ID
	log       *logger_provider.Logger          // 日志（可选）
}

// NewEventBus 创建一个新的泛型事件总线实例
func NewEventBus[T any]() *eventBusProvider[T] {
	return &eventBusProvider[T]{
		listeners: make(map[string][]*listenerWrapper[T]),
		nextID:    1,
	}
}

// WithLogger 为事件总线设置 logger（可选）。
func (bus *eventBusProvider[T]) WithLogger(log *logger_provider.Logger) *eventBusProvider[T] {
	bus.log = log
	return bus
}

// On 注册监听器，返回监听器ID用于取消订阅
func (bus *eventBusProvider[T]) On(eventName string, listener Listener[T]) uint64 {
	// 验证事件名称
	if eventName == "" {
		return 0
	}

	bus.lock.Lock()
	defer bus.lock.Unlock()

	// 生成唯一ID
	id := atomic.AddUint64(&bus.nextID, 1)

	// 创建监听器包装器
	wrapper := &listenerWrapper[T]{
		id:       id,
		listener: listener,
	}

	// 添加到监听器列表
	bus.listeners[eventName] = append(bus.listeners[eventName], wrapper)

	return id
}

// Emit 同步广播事件
func (bus *eventBusProvider[T]) Emit(ctx context.Context, eventName string, payload T) {
	// 验证事件名称
	if eventName == "" {
		return
	}

	bus.lock.RLock()
	defer bus.lock.RUnlock()

	wrappers := bus.listeners[eventName]
	event := Event[T]{Name: eventName, Payload: payload, Context: ctx}

	for _, wrapper := range wrappers {
		// 添加 panic 恢复机制
		func(w *listenerWrapper[T], e Event[T]) {
			defer func() {
				if r := recover(); r != nil {
					if bus.log != nil {
						bus.log.Errorw("panic in event listener", "event", eventName, "panic", fmt.Sprint(r))
					}
				}
			}()
			w.listener(ctx, e)
		}(wrapper, event)
	}
}

// EmitAsync 异步广播事件
func (bus *eventBusProvider[T]) EmitAsync(ctx context.Context, eventName string, payload T) {
	// 验证事件名称
	if eventName == "" {
		return
	}

	bus.lock.RLock()
	defer bus.lock.RUnlock()

	wrappers := bus.listeners[eventName]
	event := Event[T]{Name: eventName, Payload: payload, Context: ctx}

	for _, wrapper := range wrappers {
		go func(w *listenerWrapper[T], e Event[T]) {
			// 添加 panic 恢复机制
			defer func() {
				if r := recover(); r != nil {
					if bus.log != nil {
						bus.log.Errorw("panic in event listener", "event", eventName, "panic", fmt.Sprint(r))
					}
				}
			}()
			w.listener(ctx, e)
		}(wrapper, event)
	}
}

// Off 通过监听器ID取消订阅
func (bus *eventBusProvider[T]) Off(eventName string, listenerID uint64) bool {
	// 验证参数
	if eventName == "" || listenerID == 0 {
		return false
	}

	bus.lock.Lock()
	defer bus.lock.Unlock()

	wrappers, exists := bus.listeners[eventName]
	if !exists {
		return false
	}

	// 查找并移除指定ID的监听器
	for i, wrapper := range wrappers {
		if wrapper.id == listenerID {
			// 移除监听器
			bus.listeners[eventName] = append(wrappers[:i], wrappers[i+1:]...)

			// 如果该事件没有监听器了，删除该事件键以节省内存
			if len(bus.listeners[eventName]) == 0 {
				delete(bus.listeners, eventName)
			}

			return true
		}
	}

	return false
}

// GetListeners 获取指定事件的监听器数量
func (bus *eventBusProvider[T]) GetListeners(eventName string) int {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	if wrappers, exists := bus.listeners[eventName]; exists {
		return len(wrappers)
	}
	return 0
}

// GetAllEvents 获取所有事件名称
func (bus *eventBusProvider[T]) GetAllEvents() []string {
	bus.lock.RLock()
	defer bus.lock.RUnlock()

	events := make([]string, 0, len(bus.listeners))
	for eventName := range bus.listeners {
		events = append(events, eventName)
	}
	return events
}

// Clear 清空所有监听器
func (bus *eventBusProvider[T]) Clear() {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	bus.listeners = make(map[string][]*listenerWrapper[T])
}

// ClearEvent 清空指定事件的所有监听器
func (bus *eventBusProvider[T]) ClearEvent(eventName string) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	delete(bus.listeners, eventName)
}
