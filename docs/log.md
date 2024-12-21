# `zLib` 包中 `Log` 说明文档：

```markdown
# zLib 包日志说明文档

## `_logger` 结构体

`_logger` 结构体用于管理日志设置，包括是否写入日志文件和是否启用调试模式。

## 全局变量

- `Log`: 全局 `_logger` 实例，可以在包外部使用。

## 方法

### `Init`

初始化日志设置。

```go
func (logger *_logger) Init(writeLogFile bool, debugMode bool)
```

#### 参数

- `writeLogFile` (`bool`): 是否将日志写入文件。
- `debugMode` (`bool`): 是否启用调试模式。

此方法会根据传入的参数设置日志的输出方式和格式。

## 日志级别

- `DEBUG`: 调试级别，用于输出调试信息。
- `INFO`: 信息级别，用于输出一般信息。
- `ERROR`: 错误级别，用于输出错误信息。

## 日志实例

- `Debug`: 调试级别日志实例。
- `Info`: 信息级别日志实例。
- `Error`: 错误级别日志实例。

## 自定义日志写入器

### `_customWriter` 结构体

`_customWriter` 结构体用于自定义日志的输出格式。

### `Write`

自定义日志写入方法。

```go
func (w _customWriter) Write(data []byte) (n int, err error)
```

#### 参数

- `data` (`[]byte`): 要写入的日志数据。

#### 返回值

- `n` (`int`): 写入的字节数。
- `err` (`error`): 如果写入过程中发生错误，返回错误信息。

此方法会根据日志级别和调试模式设置日志的前缀颜色和内容。
