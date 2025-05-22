# 日志模块

go-zLib 的日志模块提供了多级别的日志输出功能，支持控制台和文件日志。该模块基于 Go 标准库的 `log` 包进行封装，提供了更丰富的功能和更简单的接口。

## 目录
- [基本用法](#基本用法)
- [日志级别](#日志级别)
- [日志配置](#日志配置)
- [高级用法](#高级用法)

## 基本用法

日志模块在导入 `z` 包后即可使用，无需额外初始化。

```go
import "github.com/icreateapp-com/go-zLib/z"

func main() {
    // 输出各级别日志
    z.Debug.Println("这是调试信息")
    z.Info.Println("这是普通信息")
    z.Warn.Println("这是警告信息")
    z.Error.Println("这是错误信息")
    
    // 格式化输出
    z.Info.Printf("用户 %s 登录成功，ID: %d", "张三", 123)
    
    // 输出致命错误并终止程序
    z.Error.Fatal("发生致命错误，程序终止")
}
```

## 日志级别

日志模块定义了以下日志级别：

| 日志对象 | 级别 | 颜色 | 说明 |
|------|------|------|------|
| z.Debug | 调试 | 蓝色 | 详细的调试信息，通常仅在开发环境使用 |
| z.Info | 信息 | 绿色 | 普通信息，记录程序的正常运行状态 |
| z.Warn | 警告 | 黄色 | 警告信息，表示可能出现的问题 |
| z.Error | 错误 | 红色 | 错误信息，表示程序遇到的错误 |

## 日志配置

日志模块会根据配置文件中的设置进行初始化。可配置的选项包括日志级别、日志路径、文件日志等。

### 配置示例

```yaml
config:
  log:
    level: "debug"  # 日志级别：debug, info, warn, error
    path: "./logs"  # 日志文件存储路径
    file: true      # 是否输出到文件
    console: true   # 是否输出到控制台
  debug: true       # 是否为调试模式
```

### 配置说明

| 配置项 | 类型 | 默认值 | 说明 |
|------|------|------|------|
| config.log.level | string | "info" | 日志级别，低于此级别的日志将不会记录 |
| config.log.path | string | "./logs" | 日志文件存储路径 |
| config.log.file | bool | true | 是否将日志输出到文件 |
| config.log.console | bool | true | 是否将日志输出到控制台 |
| config.debug | bool | false | 是否为调试模式，为 true 时会记录更详细的日志 |

### 日志文件

当启用文件日志时，日志文件将按日期进行分割，文件命名格式为 `{level}-{date}.log`，例如：

- `debug-2023-07-01.log`
- `info-2023-07-01.log`
- `warn-2023-07-01.log`
- `error-2023-07-01.log`

## 方法说明

每个日志级别对象都提供以下方法：

### Println 方法

输出一行日志。

**参数**：
- v (...interface{}): 日志内容

**返回值**：无

```go
z.Info.Println("用户登录成功")
```

### Printf 方法

格式化输出日志。

**参数**：
- format (string): 格式化字符串
- v (...interface{}): 格式化参数

**返回值**：无

```go
z.Info.Printf("用户 %s (ID: %d) 登录成功", "张三", 123)
```

### Fatal 方法

输出致命错误日志并终止程序（调用 `os.Exit(1)`）。

**参数**：
- v (...interface{}): 日志内容

**返回值**：无

```go
z.Error.Fatal("数据库连接失败，程序无法继续运行")
```

### Fatalf 方法

格式化输出致命错误日志并终止程序。

**参数**：
- format (string): 格式化字符串
- v (...interface{}): 格式化参数

**返回值**：无

```go
z.Error.Fatalf("数据库连接失败: %s", err.Error())
```

## 高级用法

### 动态控制日志输出

在某些情况下，可能需要动态控制日志输出，例如在特定条件下才输出详细日志：

```go
func processRequest(req *Request) {
    // 为重要用户或特定请求输出详细日志
    if req.IsVIP || req.IsSpecial {
        z.Debug.Printf("处理请求详情: %+v", req)
    }
    
    // 正常处理流程...
}
```

### 日志格式

日志输出格式包含时间戳、日志级别和内容，例如：

```
[2023-07-01 12:34:56] [INFO] 这是一条信息日志
```

控制台输出时会根据日志级别使用不同颜色，使日志更易于阅读。

### 性能考虑

日志输出会对程序性能产生一定影响，尤其是大量的调试日志。在生产环境中，建议将日志级别设置为 `info` 或 `warn`，以减少不必要的输出。

```go
// 在性能敏感的代码中，可以先检查日志级别
if z.IsDebugEnabled() {
    // 构建详细日志，这部分可能较耗性能
    detailedLog := buildDetailedLog()
    z.Debug.Println(detailedLog)
}
```

### 配合性能探针使用

日志模块可以配合性能探针一起使用，记录关键操作的执行时间：

```go
func handleRequest() {
    // 创建性能探针
    probe := service.PerformanceProbe{
        Name:    "handleRequest",
        LogType: service.ProbeLogTypeConsole,
    }
    probe.Start()
    defer probe.End()
    
    // 记录重要操作
    z.Info.Println("开始处理请求")
    
    // 处理请求...
    
    z.Info.Println("请求处理完成")
}
```

### 自定义日志输出

如果需要更多自定义的日志功能，可以获取原始的 log.Logger 对象：

```go
// 获取原始 logger
infoLogger := z.GetRawLogger("info")
if infoLogger != nil {
    // 使用原始 logger 的更多功能
}
``` 