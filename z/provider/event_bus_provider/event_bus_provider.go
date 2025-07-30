package event_bus_provider

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/icreateapp-com/go-zLib/z"
)

// Event 表示一个泛型事件
type Event[T any] struct {
	Name    string // 事件名称
	Payload T      // 事件载荷（泛型）
}

// Listener 是一个泛型处理函数
type Listener[T any] func(event Event[T])

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
}

// 全局事件总线实例（用于 any 类型）
var globalEventBus = &eventBusProvider[any]{
	listeners: make(map[string][]*listenerWrapper[any]),
	nextID:    1,
}

// NewEventBus 创建一个新的泛型事件总线实例
func NewEventBus[T any]() *eventBusProvider[T] {
	return &eventBusProvider[T]{
		listeners: make(map[string][]*listenerWrapper[T]),
		nextID:    1,
	}
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
func (bus *eventBusProvider[T]) Emit(eventName string, payload T) {
	// 验证事件名称
	if eventName == "" {
		return
	}

	bus.lock.RLock()
	defer bus.lock.RUnlock()

	wrappers := bus.listeners[eventName]
	event := Event[T]{Name: eventName, Payload: payload}

	for _, wrapper := range wrappers {
		// 添加 panic 恢复机制
		func(w *listenerWrapper[T], e Event[T]) {
			defer func() {
				if r := recover(); r != nil {
					z.Error.Printf("Panic in event listener: %v", r)
				}
			}()
			w.listener(e)
		}(wrapper, event)
	}
}

// EmitAsync 异步广播事件
func (bus *eventBusProvider[T]) EmitAsync(eventName string, payload T) {
	// 验证事件名称
	if eventName == "" {
		return
	}

	bus.lock.RLock()
	defer bus.lock.RUnlock()

	wrappers := bus.listeners[eventName]
	event := Event[T]{Name: eventName, Payload: payload}

	for _, wrapper := range wrappers {
		go func(w *listenerWrapper[T], e Event[T]) {
			// 添加 panic 恢复机制
			defer func() {
				if r := recover(); r != nil {
					z.Error.Printf("Panic in event listener: %v", r)
				}
			}()
			w.listener(e)
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

// ===== 便捷的全局方法（支持泛型） =====

// On 订阅事件（全局便捷方法，支持泛型）
func On[T any](eventName string, listener func(event Event[T])) uint64 {
	// 为每种类型创建独立的事件总线实例
	bus := getOrCreateTypedBus[T]()
	return bus.On(eventName, listener)
}

// Off 取消订阅事件（全局便捷方法）
func Off[T any](eventName string, listenerID uint64) bool {
	bus := getOrCreateTypedBus[T]()
	return bus.Off(eventName, listenerID)
}

// Emit 发布事件（全局便捷方法，支持泛型）
func Emit[T any](eventName string, payload T) {
	bus := getOrCreateTypedBus[T]()
	bus.Emit(eventName, payload)
}

// EmitAsync 异步发布事件（全局便捷方法，支持泛型）
func EmitAsync[T any](eventName string, payload T) {
	bus := getOrCreateTypedBus[T]()
	bus.EmitAsync(eventName, payload)
}

// GetListeners 获取指定事件的监听器数量（全局便捷方法）
func GetListeners[T any](eventName string) int {
	bus := getOrCreateTypedBus[T]()
	return bus.GetListeners(eventName)
}

// GetAllEvents 获取所有事件名称（全局便捷方法）
func GetAllEvents[T any]() []string {
	bus := getOrCreateTypedBus[T]()
	return bus.GetAllEvents()
}

// Clear 清空所有监听器（全局便捷方法）
func Clear[T any]() {
	bus := getOrCreateTypedBus[T]()
	bus.Clear()
}

// ClearEvent 清空指定事件的所有监听器（全局便捷方法）
func ClearEvent[T any](eventName string) {
	bus := getOrCreateTypedBus[T]()
	bus.ClearEvent(eventName)
}

// 全局事件总线管理器
var (
	globalBusMap  = make(map[string]interface{})
	globalBusLock sync.RWMutex
)

// getOrCreateTypedBus 获取或创建指定类型的事件总线
func getOrCreateTypedBus[T any]() *eventBusProvider[T] {
	// 使用类型名作为键
	typeName := getTypeName[T]()

	globalBusLock.RLock()
	if bus, exists := globalBusMap[typeName]; exists {
		globalBusLock.RUnlock()
		return bus.(*eventBusProvider[T])
	}
	globalBusLock.RUnlock()

	globalBusLock.Lock()
	defer globalBusLock.Unlock()

	// 双重检查
	if bus, exists := globalBusMap[typeName]; exists {
		return bus.(*eventBusProvider[T])
	}

	// 创建新的事件总线
	newBus := NewEventBus[T]()
	globalBusMap[typeName] = newBus
	return newBus
}

// getTypeName 获取类型名称
func getTypeName[T any]() string {
	var zero T
	return fmt.Sprintf("%T", zero)
}
