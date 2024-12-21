# `zLib` 包中 HTTP 响应说明文档：

```markdown
# zLib 包 HTTP 响应说明文档

## 函数列表

- `Json`: 返回 JSON 格式的数据。
- `Success`: 返回成功信息。
- `Failure`: 返回失败信息。

## `Json` 函数

### 功能

返回 JSON 格式的数据。

### 原型

```go
func Json(c *gin.Context, obj any)
```

### 参数

- `c` (`*gin.Context`): Gin 框架的上下文对象，用于处理 HTTP 请求和响应。
- `obj` (`any`): 要返回的数据对象。

### 示例

```go
Json(c, gin.H{"key": "value"})
```

## `Success` 函数

### 功能

返回成功信息。

### 原型

```go
func Success(c *gin.Context, message any, code ...int)
```

### 参数

- `c` (`*gin.Context`): Gin 框架的上下文对象，用于处理 HTTP 请求和响应。
- `message` (`any`): 成功信息。
- `code` (`...int`): 可选参数，返回的状态码。如果不提供，默认为 10000。

### 示例

```go
Success(c, "操作成功")
```

## `Failure` 函数

### 功能

返回失败信息。

### 原型

```go
func Failure(c *gin.Context, message any, code ...int)
```

### 参数

- `c` (`*gin.Context`): Gin 框架的上下文对象，用于处理 HTTP 请求和响应。
- `message` (`any`): 失败信息。
- `code` (`...int`): 可选参数，返回的状态码。如果不提供，默认为 20000。

### 示例

```go
Failure(c, "操作失败")
```

