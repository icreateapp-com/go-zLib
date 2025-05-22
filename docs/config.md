# 配置管理模块

go-zLib 的配置管理模块基于 Viper 库实现，提供了灵活的配置加载和访问功能，支持多种配置文件格式和动态配置更新。

## 目录
- [基本用法](#基本用法)
- [配置文件格式](#配置文件格式)
- [访问配置项](#访问配置项)
- [配置合并](#配置合并)
- [环境变量支持](#环境变量支持)
- [配置热加载](#配置热加载)

## 基本用法

配置管理模块会在应用启动时自动初始化，无需手动调用初始化函数。

```go
import "github.com/icreateapp-com/go-zLib/z"

func main() {
    // 读取字符串配置
    appName, err := z.Config.String("config.name")
    if err != nil {
        z.Error.Println("读取配置失败:", err)
    }
    
    // 读取数字配置
    port, err := z.Config.Int("config.port")
    if err != nil {
        z.Error.Println("读取配置失败:", err)
    }
    
    // 读取布尔值配置
    debug, err := z.Config.Bool("config.debug")
    if err != nil {
        z.Error.Println("读取配置失败:", err)
    }
    
    z.Info.Printf("应用名称: %s, 端口: %d, 调试模式: %v", appName, port, debug)
}
```

## 配置文件格式

配置管理模块支持多种配置文件格式，包括 YAML、JSON、TOML 等。默认情况下，模块会按照以下优先级顺序查找配置文件：

1. `./{env}.config.yaml`：根据 `GOENV` 环境变量加载环境特定配置
2. `./config.yaml`：默认配置文件

### 示例配置文件 (YAML 格式)

```yaml
# 基本配置项
config:
  name: "my-service"
  port: 8080
  debug: true
  
  # 数据库配置
  db:
    driver: "mysql"
    host: "localhost"
    port: 3306
    username: "root"
    password: "password"
    database: "mydb"
    
  # 日志配置
  log:
    level: "info"
    path: "./logs"
    
  # Redis 配置
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0

# 服务配置
name: "user-service"
version: "1.0.0"
```

## 访问配置项

配置管理模块提供了多种方法来访问不同类型的配置项：

### 获取字符串值

```go
// 获取字符串值
value, err := z.Config.String("config.name")
if err != nil {
    // 处理错误
}

// 带默认值的字符串获取
value := z.Config.GetString("config.name", "default-name")
```

### 获取数字值

```go
// 获取整数值
value, err := z.Config.Int("config.port")
if err != nil {
    // 处理错误
}

// 获取 int64 值
value, err := z.Config.Int64("config.maxSize")
if err != nil {
    // 处理错误
}

// 获取浮点值
value, err := z.Config.Float64("config.rate")
if err != nil {
    // 处理错误
}

// 带默认值
value := z.Config.GetInt("config.port", 8080)
value := z.Config.GetInt64("config.maxSize", 1024)
value := z.Config.GetFloat64("config.rate", 0.5)
```

### 获取布尔值

```go
// 获取布尔值
value, err := z.Config.Bool("config.debug")
if err != nil {
    // 处理错误
}

// 带默认值
value := z.Config.GetBool("config.debug", false)
```

### 获取时间值

```go
// 获取时间值
value, err := z.Config.Time("config.startTime")
if err != nil {
    // 处理错误
}

// 获取持续时间值
value, err := z.Config.Duration("config.timeout")
if err != nil {
    // 处理错误
}
```

### 获取数组和映射

```go
// 获取字符串数组
value, err := z.Config.StringSlice("config.tags")
if err != nil {
    // 处理错误
}

// 获取整数数组
value, err := z.Config.IntSlice("config.ports")
if err != nil {
    // 处理错误
}

// 获取字符串映射
value, err := z.Config.StringMap("config.headers")
if err != nil {
    // 处理错误
}

// 获取字符串到字符串映射的映射
value, err := z.Config.StringMapString("config.environement")
if err != nil {
    // 处理错误
}
```

### 路径访问规则

配置项路径使用点号 `.` 分隔层级，例如 `config.db.host` 表示：

```yaml
config:
  db:
    host: "localhost"
```

## 配置合并

配置管理模块支持多配置文件合并，这在不同环境下管理配置非常有用。例如，您可以有一个基本配置文件和特定环境的配置文件：

### 配置文件优先级

1. 环境特定配置文件 (如 `production.config.yaml`)
2. 默认配置文件 (`config.yaml`)

当配置项在多个文件中出现时，优先级高的配置会覆盖优先级低的配置。

### 设置 GOENV 环境变量

```bash
# Linux/macOS
export GOENV=production

# Windows
set GOENV=production
```

## 环境变量支持

配置管理模块也支持从环境变量加载配置，环境变量的优先级高于配置文件。

### 环境变量命名规则

环境变量名称需要按照以下规则转换：

1. 所有字母大写
2. 点号 `.` 替换为下划线 `_`

例如，配置路径 `config.db.host` 对应的环境变量名为 `CONFIG_DB_HOST`。

```bash
# 设置配置环境变量
export CONFIG_DB_HOST=db.example.com
export CONFIG_PORT=9000
```

## 配置热加载

配置管理模块支持配置热加载，可以在不重启应用的情况下更新配置。

```go
// 添加配置变更的回调函数
z.Config.OnConfigChange(func() {
    // 配置变更后的处理逻辑
    z.Info.Println("配置已更新")
    
    // 重新加载依赖配置的组件
    reloadComponents()
})
```

### 手动重新加载配置

在某些情况下，可能需要手动重新加载配置：

```go
// 重新加载配置
err := z.Config.ReloadConfig()
if err != nil {
    z.Error.Println("重新加载配置失败:", err)
}
```

## 方法说明

### 访问配置

#### String 方法

获取字符串配置值。

**参数**：
- key (string): 配置键路径

**返回值**：
- (string, error): 字符串值和可能的错误

```go
value, err := z.Config.String("config.name")
```

#### GetString 方法

获取字符串配置值，支持默认值。

**参数**：
- key (string): 配置键路径
- defaultValue (string): 默认值

**返回值**：
- string: 字符串值，如果配置不存在则返回默认值

```go
value := z.Config.GetString("config.name", "default-name")
```

#### Int/GetInt, Int64/GetInt64, Float64/GetFloat64 方法

获取数字类型配置值，使用方式类似 String/GetString。

#### Bool/GetBool 方法

获取布尔类型配置值，使用方式类似 String/GetString。

#### Time/GetTime, Duration/GetDuration 方法

获取时间类型配置值，使用方式类似 String/GetString。

#### StringSlice/GetStringSlice, IntSlice/GetIntSlice 方法

获取数组类型配置值，使用方式类似 String/GetString。

#### StringMap/GetStringMap, StringMapString/GetStringMapString 方法

获取映射类型配置值，使用方式类似 String/GetString。

### 配置管理

#### ReloadConfig 方法

重新加载配置文件。

**参数**：无

**返回值**：
- error: 重载过程中可能的错误

```go
err := z.Config.ReloadConfig()
```

#### OnConfigChange 方法

添加配置变更回调函数。

**参数**：
- callback (func()): 配置变更时调用的回调函数

**返回值**：无

```go
z.Config.OnConfigChange(func() {
    // 配置变更处理
})
``` 