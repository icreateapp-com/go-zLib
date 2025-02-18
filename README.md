<h1 align="center">
  zLib
</h1>
<h3 align="center">
  基于 Golang 的工具库。
</h3>

## 📦 模块说明

### 数据库DB

#### 功能描述：
- **初始化**：提供数据库连接初始化功能，支持 MySQL 数据库驱动，并根据配置文件动态加载数据库连接信息。
- **事务管理**：提供事务处理方法，确保数据操作的原子性。
- **字段转义**：根据不同的数据库类型（如 MySQL），对 SQL 字段进行适当的转义处理。
- **自定义时间格式**：通过 `WrapTime` 类型实现自定义的时间格式化和反序列化。
- **CRUD 操作**：提供创建、读取、更新、删除等基本操作的构建器模式，简化数据库操作。

#### 主要文件：
- `db.go`：数据库初始化和基础方法。
- `timestamp.go`：自定义时间格式。
- `db-create-builder.go`：创建操作构建器。
- `db-delete-builder.go`：删除操作构建器。
- `db-query-builder.go`：查询操作构建器。
- `db-update-builder.go`：更新操作构建器。
- `json.go`：JSON 字段处理。
- `model.go`：模型基类和时间戳处理。
- `mysql.go`：MySQL 驱动实现。

### 中间件

#### 功能描述：
- **认证中间件**：用于验证请求中的授权令牌，确保只有合法用户可以访问受保护的 API 路径。
- **健康检查中间件**：提供 `/health` 和 `/alive` 接口，用于监控服务的健康状态。
- **查询转换中间件**：将前端传递的查询字符串转换为 JSON 对象，方便后续处理。

#### 主要文件：
- `auth.go`：认证中间件。
- `health.go`：健康检查中间件。
- `query.go`：查询转换中间件。

### 提供者

#### 功能描述：
- **配置中心提供者**：与配置中心集成，动态同步配置信息并注册服务。
- **服务发现提供者**：实现服务注册和发现功能，支持自动健康检查和重新注册。

#### 主要文件：
- `config-center.go`：配置中心提供者。
- `service-discover.go`：服务发现提供者。

### 服务

#### 功能描述：
- **CRUD 服务**：封装了常见的 CRUD 操作，简化业务逻辑层的数据操作。
- **性能探针**：记录函数执行时间和内存占用，帮助开发者分析性能瓶颈。

#### 主要文件：
- `crud.go`：CRUD 服务。
- `performance_probe.go`：性能探针。

### 功能类

#### 功能描述：
- **HTTP 请求**：封装 HTTP 请求方法，简化外部 API 调用。
- **日志系统**：提供多级别的日志输出功能，支持控制台和文件日志。
- **缓存**：实现内存缓存和 Redis 缓存，提高数据读取效率。
- **定时任务**：基于 Cron 表达式的定时任务调度。
- **响应处理**：统一的 HTTP 响应格式，简化 API 返回值处理。
- **异常处理**：自定义异常类型，便于错误捕获和处理。
- **字符串处理**：提供常用的字符串操作函数。
- **切片操作**：提供切片的常用操作函数。
- **对象操作**：提供结构体和映射的操作函数。
- **接口操作**：提供通用的接口转换和操作函数。

#### 主要文件：
- `z_http.go`：HTTP 请求。
- `log.go`：日志系统。
- `mem_cache.go`：内存缓存。
- `redis_cache.go`：Redis 缓存。
- `cron.go`：定时任务。
- `response.go`：响应处理。
- `exception.go`：异常处理。
- `z_string.go`：字符串处理。
- `z_slice.go`：切片操作。
- `z_object.go`：对象操作。
- `z_interface.go`：接口操作。

### 函数

#### 功能描述：
- **路径操作**：提供项目目录、存储目录、临时目录、缓存目录和日志目录的绝对路径获取。
- **编码解码**：提供 SHA1、MD5、Base64 等编码解码功能。
- **JSON 操作**：提供 JSON 编码、解码和提取指定键值的功能。
- **URL 操作**：生成当前服务器的 URL 地址，发起 HTTP 请求，判断 URL 有效性，下载文件等。
- **其他辅助函数**：提供各种辅助函数，如判断字符串是否为空、字符串转数字、驼峰转蛇形等。

#### 主要文件：
- `z_base.go`：路径操作。
- `z_coding.go`：编码解码。
- `z_json.go`：JSON 操作。
- `z_url.go`：URL 操作。
- `z_other.go`：其他辅助函数。

---

## 🚀 使用示例

### 数据库操作

```go
// 初始化数据库
db := db.New()
// 创建记录
createBuilder := db.CreateBuilder{Model: &User{}}
err := createBuilder.Create(&user)
// 查询记录
query := db.Query{
    Filter: []string{"id", "name"},
    Search: []ConditionGroup{{Conditions: [][]interface{}{{"id", "1"}}}},
}
result, err := QueryBuilder{Model: &User{}}.Get(query)
```


### 中间件使用

```go
router.Use(middleware.AuthMiddleware())
router.GET("/health", middleware.HealthMiddleware())
```


### 提供者使用

```go
provider.ConfigCenterProvider.Register()
provider.ServiceDiscoverProvider.Register()
```


### 服务使用

```go
crudService := service.CrudService{Model: &User{}}
result, err := crudService.Get(query)
```


### 功能类使用

```go
// 日志输出
z.Info.Println("This is an info message")
// 发起 HTTP 请求
res, err := z.Post("https://example.com/api", map[string]interface{}{"key": "value"}, nil)
// 缓存操作
z.MemCache.Set("key", "value", time.Minute)
// 定时任务
z.Cron.Add("@daily", func() { fmt.Println("Daily task") })
```


### 函数使用

```go
// 获取项目路径
projectPath := z.BasePath()
// 判断字符串是否为空
isEmpty := z.StringIsEmpty("")
// 编码解码
md5Str := z.GetMd5("test")
base64Str := z.EncodeBase64("test")
decodedStr, _ := z.DecodeBase64(base64Str)
```


---

