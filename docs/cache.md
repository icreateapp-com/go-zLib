# `zLib` 包中 `Cache` 说明文档：

```markdown
# zLib 包缓存说明文档

## `_cache` 结构体

`_cache` 结构体用于存储缓存对象和默认有效期。

### 字段

- `cache`: `*cache.Cache` - 用于存储实际的缓存对象。
- `DefaultExpiration`: `time.Duration` - 缓存的默认有效期。

## 全局变量

- `Cache`: `_cache` - 全局缓存对象，可以在包外部使用。

## 方法

### `Init`

初始化缓存（默认有效期，清理缓存间隔时间）。

```go
func (p *_cache) Init(defaultExpiration, cleanupInterval time.Duration)
```

#### 参数

- `defaultExpiration` (`time.Duration`): 缓存的默认有效期。
- `cleanupInterval` (`time.Duration`): 清理缓存的间隔时间。

### `Set`

创建缓存。

```go
func (p *_cache) Set(k string, x interface{}, d time.Duration)
```

#### 参数

- `k` (`string`): 缓存的键。
- `x` (`interface{}`): 缓存的值。
- `d` (`time.Duration`): 缓存的持续时间。如果为 0，则使用默认有效期。

### `Get`

获取缓存。

```go
func (p *_cache) Get(k string) (interface{}, bool)
```

#### 参数

- `k` (`string`): 缓存的键。

#### 返回值

- `interface{}`: 缓存的值。
- `bool`: 如果键存在返回 `true`，否则返回 `false`。

### `Delete`

删除缓存。

```go
func (p *_cache) Delete(k string)
```

#### 参数

- `k` (`string`): 要删除的缓存键。

