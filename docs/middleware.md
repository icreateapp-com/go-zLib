# 中间件模块

go-zLib 中间件模块提供了一系列可以与 Gin 框架配合使用的中间件，用于处理常见的 Web 应用场景，如认证、健康检查和查询参数处理。

## 目录
- [认证中间件](#认证中间件)
- [健康检查中间件](#健康检查中间件)
- [查询转换中间件](#查询转换中间件)

## 认证中间件

认证中间件用于验证请求中的授权令牌，确保只有合法用户可以访问受保护的 API 路径。

### 使用方法

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/grpc_middleware"
)

func main() {
    router := gin.Default()
    
    // 应用认证中间件
    router.Use(middleware.AuthMiddleware())
    
    // 路由定义
    router.GET("/api/users", getUsers)
    
    router.Run(":8080")
}
```

### 配置

认证中间件需要在配置文件中定义认证规则：

```yaml
config:
  anonymity:
    - "/public"  # 不需要认证的路径前缀
    - "/api/v1/auth"
  auth:
    "/api": "api-token-value"  # 路径前缀与对应的访问令牌
    "/admin": "admin-token-value"
```

### 工作原理

1. 中间件首先检查请求路径是否匹配 `config.anonymity` 中的前缀，如果匹配则跳过认证
2. 然后从请求头的 `Authorization` 字段提取访问令牌
3. 根据请求路径匹配 `config.auth` 中的配置项
4. 比对令牌是否与配置中的值匹配，如果匹配则允许请求继续，否则返回 401 未授权错误

### 参数

认证中间件没有直接的参数，但通过配置文件进行配置：

| 配置项 | 类型 | 说明 |
|------|------|------|
| config.anonymity | []string | 不需要认证的路径前缀列表 |
| config.auth | map[string]string | 路径前缀与对应令牌的映射 |

### 返回值

当认证失败时，中间件会返回：

```json
{
  "success": false,
  "message": "Access token error",
  "code": 20000
}
```

## 健康检查中间件

健康检查中间件提供了检查服务健康状态的端点，通常用于监控系统和负载均衡器。

### 使用方法

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/grpc_middleware"
)

func main() {
    router := gin.Default()
    
    // 添加健康检查端点
    router.GET("/health", middleware.HealthMiddleware())
    router.GET("/alive", middleware.HealthMiddleware())
    
    router.Run(":8080")
}
```

### 工作原理

健康检查中间件会返回服务的状态信息，包括：

- 服务名称
- 服务版本
- 运行时间
- 当前时间
- 内存使用情况

### 返回值

健康检查端点返回以下 JSON 格式的响应：

```json
{
  "success": true,
  "message": {
    "name": "your-service-name",
    "version": "1.0.0",
    "uptime": "1h 2m 3s",
    "time": "2023-01-01 12:00:00",
    "memory": {
      "allocated": "10 MB",
      "total": "20 MB",
      "system": "30 MB"
    }
  },
  "code": 10000
}
```

## 查询转换中间件

查询转换中间件用于将前端传递的查询字符串转换为 JSON 对象，便于后续处理。

### 使用方法

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/grpc_middleware"
)

func main() {
    router := gin.Default()
    
    // 应用查询转换中间件
    router.Use(middleware.QueryMiddleware())
    
    // 路由定义
    router.GET("/api/users", getUsers)
    
    router.Run(":8080")
}

func getUsers(c *gin.Context) {
    // 获取转换后的查询参数
    query, _ := c.Get("query")
    // ...
}
```

### 请求示例

当前端发送如下请求：

```
GET /api/users?filter[]=id&filter[]=name&search[][name]=John&page=1&page_size=10
```

中间件会将查询参数转换为：

```json
{
  "filter": ["id", "name"],
  "search": [{"name": "John"}],
  "page": 1,
  "page_size": 10
}
```

### 工作原理

1. 中间件解析请求 URL 中的查询参数
2. 将键名包含 `[]` 的参数识别为数组
3. 将键名包含 `[key]` 的参数识别为对象属性
4. 将转换后的 JSON 对象存储在 Gin 上下文中，键名为 `query`

### 支持的参数类型

- 字符串：`key=value`
- 数组：`key[]=value1&key[]=value2`
- 对象：`key[property]=value`
- 对象数组：`key[][property]=value`
- 数字：自动转换数字字符串为数字类型
- 布尔值：自动转换 "true"/"false" 为布尔类型

### 获取转换后的查询

在路由处理函数中，可以通过 Gin 上下文获取转换后的查询参数：

```go
func handler(c *gin.Context) {
    query, exists := c.Get("query")
    if !exists {
        // 处理错误
        return
    }
    
    // 使用查询参数
    queryMap := query.(map[string]interface{})
    // ...
}
``` 