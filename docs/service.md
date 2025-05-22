# 服务层模块

go-zLib 的服务层模块提供了通用的业务逻辑服务封装，主要包括通用的 CRUD 服务和性能监控服务。

## 目录
- [CRUD 服务](#crud-服务)
- [性能探针服务](#性能探针服务)

## CRUD 服务

CRUD 服务封装了常见的创建、读取、更新和删除操作，简化业务逻辑层的数据操作。

### 使用方法

```go
import (
    "github.com/icreateapp-com/go-zLib/z/db"
    "github.com/icreateapp-com/go-zLib/z/service"
)

// 定义用户模型
type User struct {
    db.Model
    Name  string `json:"name"`
    Email string `json:"email"`
}

// 创建服务实例
func NewUserService() *service.CrudService {
    return &service.CrudService{Model: &User{}}
}

// 使用示例
func main() {
    userService := NewUserService()
    
    // 获取分页数据
    query := db.Query{
        Page:     1,
        PageSize: 10,
    }
    result, err := userService.Get(query)
    
    // 获取单条记录
    user, err := userService.Find(1, db.Query{})
    
    // 创建记录
    newUser := User{Name: "张三", Email: "zhangsan@example.com"}
    createdUser, err := userService.Create(&newUser)
    
    // 更新记录
    updateData := map[string]interface{}{
        "name": "李四",
    }
    success, err := userService.Update(1, updateData)
    
    // 删除记录
    success, err = userService.Delete(1)
}
```

### 结构说明

```go
type CrudService struct {
    Model interface{} // 数据模型
}
```

### 方法说明

#### Get 方法

获取分页数据。

**参数**：
- query (db.Query): 查询参数

**返回值**：
- (interface{}, error): 查询结果和可能的错误

```go
// 查询示例
query := db.Query{
    Filter: []string{"id", "name", "email"},
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{{"name", "like", "%张%"}},
        },
    },
    Sort: [][]string{{"created_at", "desc"}},
    Page: 1,
    PageSize: 10,
}
result, err := crudService.Get(query)
```

#### Find 方法

根据 ID 查询单条记录。

**参数**：
- id (interface{}): 记录 ID
- query (db.Query): 查询参数

**返回值**：
- (interface{}, error): 查询结果和可能的错误

```go
// 查询单条记录
user, err := crudService.Find(1, db.Query{
    Filter: []string{"id", "name", "email"},
})
```

#### Create 方法

创建新记录。

**参数**：
- req (interface{}): 要创建的数据

**返回值**：
- (interface{}, error): 创建的记录和可能的错误

```go
// 创建记录
newUser := User{Name: "张三", Email: "zhangsan@example.com"}
createdUser, err := crudService.Create(&newUser)
```

#### Update 方法

更新记录。

**参数**：
- id (interface{}): 记录 ID
- req (interface{}): 更新的数据

**返回值**：
- (bool, error): 更新是否成功和可能的错误

```go
// 更新记录
updateData := map[string]interface{}{
    "name": "李四",
    "email": "lisi@example.com",
}
success, err := crudService.Update(1, updateData)
```

#### Delete 方法

删除记录。

**参数**：
- id (interface{}): 记录 ID

**返回值**：
- (bool, error): 删除是否成功和可能的错误

```go
// 删除记录
success, err := crudService.Delete(1)
```

## 性能探针服务

性能探针服务用于记录函数执行时间和内存占用，帮助开发者分析性能瓶颈。

### 使用方法

```go
import (
    "github.com/icreateapp-com/go-zLib/z/service"
)

func main() {
    // 创建性能探针
    probe := service.PerformanceProbe{
        Name:    "UserService.List",
        LogType: service.ProbeLogTypeConsole, // 输出到控制台
    }
    
    // 开始计时
    probe.Start()
    
    // 执行业务逻辑
    // ...
    
    // 结束计时
    probe.End()
}
```

### 高级用法

#### 使用 defer 进行自动结束

```go
func GetUserList() {
    probe := service.PerformanceProbe{
        Name:    "UserService.List",
        LogType: service.ProbeLogTypeConsole,
    }
    probe.Start()
    defer probe.End()
    
    // 执行业务逻辑
    // ...
}
```

#### 带有回调的性能探针

```go
func GetUserList() {
    probe := service.PerformanceProbe{
        Name:    "UserService.List",
        LogType: service.ProbeLogTypeConsole,
        Callback: func(probe *service.PerformanceProbe) {
            // 性能数据处理逻辑
            fmt.Printf("执行时间: %s, 内存使用: %s\n", probe.Duration, probe.MemoryUsage)
            
            // 可以将性能数据保存到数据库或发送到监控系统
        },
    }
    probe.Start()
    defer probe.End()
    
    // 执行业务逻辑
    // ...
}
```

### 结构说明

```go
type PerformanceProbe struct {
    Name         string                    // 探针名称
    LogType      int                       // 日志类型
    StartTime    time.Time                 // 开始时间
    EndTime      time.Time                 // 结束时间
    Duration     time.Duration             // 执行时间
    MemoryUsage  string                    // 内存使用量
    Callback     func(*PerformanceProbe)   // 回调函数
}
```

### 常量说明

```go
const (
    ProbeLogTypeNone    = 0 // 不输出日志
    ProbeLogTypeConsole = 1 // 输出到控制台
    ProbeLogTypeFile    = 2 // 输出到文件
)
```

### 方法说明

#### Start 方法

开始性能计时和内存监控。

**参数**：无

**返回值**：无

#### End 方法

结束性能计时，计算执行时间和内存使用量，并根据 LogType 输出日志。

**参数**：无

**返回值**：无

### 监控指标

性能探针会记录以下指标：

1. **执行时间**：函数从开始到结束的时间差
2. **内存使用**：函数执行期间的内存使用量
3. **时间戳**：开始和结束的时间戳

### 输出格式

当设置 `LogType` 为 `ProbeLogTypeConsole` 时，输出格式如下：

```
[Performance] Name: UserService.List, Duration: 125.42ms, Memory: 1.25MB
```

当设置 `LogType` 为 `ProbeLogTypeFile` 时，会将上述信息写入日志文件。 