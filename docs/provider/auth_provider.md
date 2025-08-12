# 认证提供者 (Auth Provider)

认证提供者为应用程序提供完整的身份认证和授权管理功能，支持多种认证方式、多租户架构和灵活的权限控制。

## 功能特性

- **多认证方式**: 支持JWT和固定Token两种认证方式
- **多租户支持**: 支持多Guard配置，实现多租户隔离
- **灵活缓存**: 支持Redis和内存两种缓存方式
- **单点登录**: 可选的SSO功能，控制用户并发登录
- **路由保护**: 自动保护API路由，支持匿名访问配置
- **会话管理**: 完整的用户会话生命周期管理
- **友好错误**: 结构化的错误响应和用户友好的错误消息
- **泛型支持**: 类型安全的用户数据获取

## 快速开始

### 1. 配置文件

在 `config.yaml` 中添加认证配置：

```yaml
config:
  auth:
    guards:
      api:                                    # Guard名称
        type: "jwt"                          # 认证类型: jwt | token
        prefix: "/api"                       # 路由前缀
        anonymity:                           # 匿名路由列表
          - "/api/login"
          - "/api/register"
          - "/api/health"
        cache: "redis"                       # 缓存类型: redis | memory
        sso_enabled: true                    # 单点登录开关
      admin:
        type: "token"
        token: "admin-secret-token-2024"     # 固定Token
        prefix: "/admin"
        anonymity:
          - "/admin/login"
        cache: "memory"
        sso_enabled: false
```

### 2. 基础使用

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z"
    "github.com/icreateapp-com/go-zLib/z/provider/auth_provider"
)

func main() {
    // 注册认证提供者
    auth_provider.AuthProvider.Register()
    
    r := gin.Default()
    
    // 添加认证中间件
    r.Use(auth_provider.AuthProvider.HttpAuthProviderMiddleware())
    
    // 登录接口
    r.POST("/api/login", loginHandler)
    
    // 受保护的接口
    r.GET("/api/profile", profileHandler)
    r.POST("/api/logout", logoutHandler)
    
    r.Run(":8080")
}

// 用户登录
func loginHandler(c *gin.Context) {
    // 验证用户凭据
    userID := "user123"
    userData := map[string]interface{}{
        "name":  "张三",
        "email": "zhangsan@example.com",
        "role":  "user",
    }
    
    // 生成认证令牌
    token, err := auth_provider.AuthProvider.Login("api", userID, userData)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "login failed",
            "message": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "token": token,
        "user": userData,
    })
}

// 获取用户信息
func profileHandler(c *gin.Context) {
    // 获取当前用户ID
    userID, err := auth_provider.AuthProvider.GetUserID("api")
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "unauthorized",
            "message": err.Error(),
        })
        return
    }
    
    // 获取用户数据（泛型方式）
    type UserProfile struct {
        Name  string `json:"name"`
        Email string `json:"email"`
        Role  string `json:"role"`
    }
    
    profile, err := auth_provider.AuthProvider.GetData[UserProfile]("api")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "data_error",
            "message": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "user_id": userID,
        "profile": profile,
    })
}

// 用户登出
func logoutHandler(c *gin.Context) {
    // 登出当前用户
    err := auth_provider.AuthProvider.Logout("api", "")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "logout_failed",
            "message": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "logout successful",
    })
}
```

## API 参考

### AuthProvider

#### Register()

注册认证提供者，初始化所有Guard配置。

```go
auth_provider.AuthProvider.Register()
```

#### Login(guard, userID, device string, duration time.Duration, data ...interface{}) (string, error)

用户登录，生成认证令牌。

```go
// 登录到默认设备
token, err := auth_provider.AuthProvider.Login("api", "user123", "", time.Hour*24, map[string]interface{}{
    "name": "张三",
    "role": "admin",
})

// 登录到指定设备
token, err := auth_provider.AuthProvider.Login("api", "user123", "mobile-app", time.Hour*24, map[string]interface{}{
    "name": "张三",
    "role": "admin",
})
```

#### Logout

登出指定设备。

```go
func Logout(guard, device string, userID ...string) error
```

**参数：**
- `guard`: 守卫名称
- `device`: 设备标识，为空时表示默认设备
- `userID`: 用户ID（可选），为空时表示当前用户

**示例：**
```go
// 登出当前用户的默认设备
err := auth_provider.Logout("api", "")

// 登出当前用户的指定设备
err := auth_provider.Logout("api", "mobile-app")

// 登出指定用户的默认设备
err := auth_provider.Logout("api", "", "user123")

// 登出指定用户的指定设备
err := auth_provider.Logout("api", "mobile-app", "user123")
```

#### LogoutAll

登出用户的所有设备。

```go
func LogoutAll(guard string, userID ...string) error
```

**参数：**
- `guard`: 守卫名称
- `userID`: 用户ID（可选），为空时表示当前用户

**示例：**
```go
// 登出当前用户的所有设备
err := auth_provider.LogoutAll("api")

// 登出指定用户的所有设备
err := auth_provider.LogoutAll("api", "user123")
```

#### GetUserID(guard string) (string, error)

获取当前认证用户的ID。

```go
userID, err := auth_provider.AuthProvider.GetUserID("api")
```

#### GetData[T any](guard string) (T, error)

获取当前用户的数据（泛型方式）。

```go
type UserInfo struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

userInfo, err := auth_provider.AuthProvider.GetData[UserInfo]("api")
```

#### Authenticate(guard, token string) (*AuthContext, error)

验证认证令牌。

```go
ctx, err := auth_provider.AuthProvider.Authenticate("api", "jwt-token")
```

### 设备管理API

#### GetCurrentDevice(guard string) (string, error)

获取当前设备标识。

```go
device, err := auth_provider.AuthProvider.GetCurrentDevice("api")
```

#### GetUserDevices(guard, userID string) ([]string, error)

获取用户的所有设备列表。

```go
devices, err := auth_provider.AuthProvider.GetUserDevices("api", "user123")
```

#### IsDeviceOnline(guard, userID, device string) (bool, error)

检查指定设备是否在线（有有效会话）。

```go
online, err := auth_provider.AuthProvider.IsDeviceOnline("api", "user123", "mobile-app")
```

#### GetDeviceInfo(guard, userID, device string) (map[string]interface{}, error)

获取设备的详细信息。

```go
info, err := auth_provider.AuthProvider.GetDeviceInfo("api", "user123", "mobile-app")
```

#### HttpAuthProviderMiddleware() gin.HandlerFunc

返回Gin认证中间件。

```go
r.Use(auth_provider.AuthProvider.HttpAuthProviderMiddleware())
```

## 中间件集成

### Gin 中间件

认证中间件会自动处理所有HTTP请求的认证：

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/provider/auth_provider"
)

func main() {
    r := gin.Default()
    
    // 添加认证中间件
    r.Use(auth_provider.AuthProvider.HttpAuthProviderMiddleware())
    
    // 中间件会自动：
    // 1. 根据路由前缀匹配对应的Guard
    // 2. 检查是否为匿名路由
    // 3. 提取和验证认证令牌
    // 4. 设置认证上下文
    // 5. 返回友好的错误响应
    
    r.GET("/api/protected", func(c *gin.Context) {
        // 此时用户已通过认证
        userID, _ := auth_provider.AuthProvider.GetUserID("api")
        c.JSON(200, gin.H{"user_id": userID})
    })
    
    r.Run(":8080")
}
```

### 自定义认证检查

```go
func requireAuth(guard string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "token required"})
            c.Abort()
            return
        }
        
        // 移除 "Bearer " 前缀
        if len(token) > 7 && token[:7] == "Bearer " {
            token = token[7:]
        }
        
        ctx, err := auth_provider.AuthProvider.Authenticate(guard, token)
        if err != nil {
            c.JSON(401, gin.H{"error": err.Error()})
            c.Abort()
            return
        }
        
        // 设置用户上下文
        c.Set("auth_context", ctx)
        c.Next()
    }
}
```

## 认证方式

### JWT 认证

JWT认证支持动态令牌生成和验证：

```go
// 配置
guards:
  api:
    type: "jwt"
    prefix: "/api"
    cache: "redis"
    sso_enabled: true

// 使用
token, err := auth_provider.AuthProvider.Login("api", "user123", userData)
// 生成的token格式: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**JWT特性：**
- 自包含的用户信息
- 支持过期时间控制
- 无状态验证
- 支持分布式部署

### 固定Token认证

固定Token认证使用预设的令牌：

```go
// 配置
guards:
  admin:
    type: "token"
    token: "admin-secret-token-2024"
    prefix: "/admin"
    cache: "memory"

// 使用
token, err := auth_provider.AuthProvider.Login("admin", "admin001", adminData)
// 返回的token就是配置中的固定token
```

**固定Token特性：**
- 简单可靠
- 适合内部系统
- 易于管理和轮换
- 高性能验证

## 缓存策略

### Redis 缓存

适合分布式部署和高并发场景：

```go
guards:
  api:
    cache: "redis"
```

**特性：**
- 支持集群部署
- 数据持久化
- 高性能访问
- 支持TTL过期

### 内存缓存

适合单机部署和快速访问：

```go
guards:
  admin:
    cache: "memory"
```

**特性：**
- 极高性能
- 零网络延迟
- 简单部署
- 进程重启数据丢失

## 单点登录 (SSO)

启用SSO后，用户在一个设备登录会自动登出其他设备：

```go
guards:
  api:
    sso_enabled: true  # 启用SSO
```

**工作原理：**
1. 用户登录时检查是否已有活跃会话
2. 如果有，则清除之前的会话
3. 创建新的会话令牌
4. 旧令牌立即失效

## 错误处理

### 错误类型

认证系统提供结构化的错误响应：

```go
type AuthError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### 预定义错误

| 错误代码 | 错误消息 | 说明 |
|---------|---------|------|
| TOKEN_MISSING | token required | 缺少认证令牌 |
| TOKEN_INVALID | invalid token | 令牌无效 |
| TOKEN_EXPIRED | token expired | 令牌已过期 |
| TOKEN_MALFORMED | malformed token | 令牌格式错误 |
| TOKEN_SIGNATURE | invalid signature | 签名无效 |
| SESSION_NOT_FOUND | session expired | 会话已过期 |
| SESSION_INVALID | invalid session | 会话数据无效 |
| GUARD_NOT_FOUND | guard not found | Guard不存在 |
| GUARD_MISMATCH | token mismatch | 令牌不匹配 |
| AUTH_TYPE_UNSUPPORTED | unsupported auth type | 不支持的认证类型 |
| PERMISSION_DENIED | access denied | 权限不足 |

### 错误响应格式

```json
{
  "error": "TOKEN_EXPIRED",
  "message": "token expired"
}
```

## 高级用法

### 多租户架构

```go
// 为不同的客户端配置不同的Guard
guards:
  customer_a:
    type: "jwt"
    prefix: "/customer-a"
    cache: "redis"
  customer_b:
    type: "jwt"
    prefix: "/customer-b"
    cache: "redis"
  internal:
    type: "token"
    token: "internal-service-token"
    prefix: "/internal"
    cache: "memory"
```

### 权限控制

```go
func requireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, err := auth_provider.AuthProvider.GetUserID("api")
        if err != nil {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        
        // 获取用户数据
        type UserData struct {
            Role string `json:"role"`
        }
        
        userData, err := auth_provider.AuthProvider.GetData[UserData]("api")
        if err != nil || userData.Role != role {
            c.JSON(403, gin.H{"error": "access denied"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// 使用
r.GET("/admin/users", requireRole("admin"), adminUsersHandler)
```

### 令牌刷新

```go
func refreshTokenHandler(c *gin.Context) {
    // 获取当前用户信息
    userID, err := auth_provider.AuthProvider.GetUserID("api")
    if err != nil {
        c.JSON(401, gin.H{"error": "unauthorized"})
        return
    }
    
    userData, err := auth_provider.AuthProvider.GetData[map[string]interface{}]("api")
    if err != nil {
        c.JSON(500, gin.H{"error": "data error"})
        return
    }
    
    // 重新登录生成新令牌
    newToken, err := auth_provider.AuthProvider.Login("api", userID, userData)
    if err != nil {
        c.JSON(500, gin.H{"error": "refresh failed"})
        return
    }
    
    c.JSON(200, gin.H{"token": newToken})
}
```

### 批量登出

```go
func logoutAllUsersHandler(c *gin.Context) {
    // 管理员功能：登出所有用户
    userIDs := []string{"user1", "user2", "user3"}
    
    var errors []string
    for _, userID := range userIDs {
        if err := auth_provider.AuthProvider.Logout("api", "", userID); err != nil {
            errors = append(errors, fmt.Sprintf("用户 %s 登出失败: %v", userID, err))
        }
    }
    
    if len(errors) > 0 {
        c.JSON(500, gin.H{
            "message": "部分用户登出失败",
            "errors": errors,
        })
        return
    }
    
    c.JSON(200, gin.H{"message": "所有用户已登出"})
}
```

## 最佳实践

### 1. 安全配置

```yaml
# 生产环境建议
guards:
  api:
    type: "jwt"
    cache: "redis"           # 使用Redis缓存
    sso_enabled: true        # 启用SSO防止令牌滥用
    anonymity:               # 最小化匿名路由
      - "/api/login"
      - "/api/health"
```

### 2. 错误处理

```go
// 统一错误处理
func handleAuthError(c *gin.Context, err error) {
    if authErr, ok := err.(*auth_provider.AuthError); ok {
        c.JSON(401, gin.H{
            "error": authErr.Code,
            "message": authErr.Message,
        })
    } else {
        c.JSON(500, gin.H{
            "error": "INTERNAL_ERROR",
            "message": "internal server error",
        })
    }
}
```

### 3. 性能优化

```go
// 缓存用户数据减少数据库查询
func getUserProfile(userID string) (*UserProfile, error) {
    // 先从缓存获取
    if cached := getUserFromCache(userID); cached != nil {
        return cached, nil
    }
    
    // 从数据库获取
    profile, err := getUserFromDB(userID)
    if err != nil {
        return nil, err
    }
    
    // 缓存结果
    cacheUser(userID, profile)
    return profile, nil
}
```

### 4. 监控和日志

```go
// 认证事件记录
func logAuthEvent(event string, userID string, guard string) {
    log.Printf("[AUTH] %s - User: %s, Guard: %s", event, userID, guard)
}

// 在登录/登出时调用
logAuthEvent("LOGIN", userID, guard)
logAuthEvent("LOGOUT", userID, guard)
```

## 故障排除

### 常见问题

1. **令牌验证失败**
   - 检查JWT密钥配置
   - 确认令牌格式正确
   - 验证令牌是否过期

2. **缓存连接失败**
   - 检查Redis连接配置
   - 确认网络连通性
   - 验证认证信息

3. **路由匹配问题**
   - 检查prefix配置
   - 确认匿名路由设置
   - 验证中间件顺序

4. **SSO不生效**
   - 确认sso_enabled配置
   - 检查缓存是否正常工作
   - 验证用户ID唯一性

### 调试技巧

```go
// 启用调试模式
func debugAuth() {
    // 打印当前认证上下文
    userID, _ := auth_provider.AuthProvider.GetUserID("api")
    fmt.Printf("当前用户: %s\n", userID)
    
    // 打印用户数据
    data, _ := auth_provider.AuthProvider.GetData[map[string]interface{}]("api")
    fmt.Printf("用户数据: %+v\n", data)
}
```

## 设备认证使用示例

### 多设备登录管理

```go
package main

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/provider/auth_provider"
)

func main() {
    r := gin.Default()
    r.Use(auth_provider.AuthProvider.HttpAuthProviderMiddleware())
    
    // 登录接口 - 支持设备参数
    r.POST("/api/login", func(c *gin.Context) {
        var req struct {
            UserID   string `json:"user_id"`
            Device   string `json:"device"`   // 设备标识
            Password string `json:"password"`
        }
        
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        // 验证密码（省略实际验证逻辑）
        
        // 登录并指定设备
        token, err := auth_provider.Login("api", req.UserID, req.Device, time.Hour*24, map[string]interface{}{
            "name": "用户名",
            "role": "user",
        })
        
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{
            "token":  token,
            "device": req.Device,
        })
    })
    
    // 获取当前设备信息
    r.GET("/api/current-device", func(c *gin.Context) {
        device, err := auth_provider.GetCurrentDevice("api")
        if err != nil {
            c.JSON(401, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{"device": device})
    })
    
    // 获取用户所有设备
    r.GET("/api/user/:userID/devices", func(c *gin.Context) {
        userID := c.Param("userID")
        
        devices, err := auth_provider.GetUserDevices("api", userID)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{"devices": devices})
    })
    
    // 检查设备状态
    r.GET("/api/user/:userID/device/:device/status", func(c *gin.Context) {
        userID := c.Param("userID")
        device := c.Param("device")
        
        online, err := auth_provider.IsDeviceOnline("api", userID, device)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{
            "device": device,
            "online": online,
        })
    })
    
    // 登出指定设备
    r.POST("/api/logout", func(c *gin.Context) {
        var req struct {
            UserID string `json:"user_id"`
            Device string `json:"device"`
        }
        
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        err := auth_provider.Logout("api", req.Device, req.UserID)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{"message": "设备已登出"})
    })
    
    // 登出所有设备
    r.POST("/api/logout-all", func(c *gin.Context) {
        var req struct {
            UserID string `json:"user_id"`
        }
        
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        err := auth_provider.LogoutAll("api", req.UserID)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{"message": "所有设备已登出"})
    })
    
    r.Run(":8080")
}
```

### 设备标识建议

```go
// 设备类型标识
const (
    DeviceWebBrowser  = "web-browser"
    DeviceMobileApp   = "mobile-app"
    DeviceDesktopApp  = "desktop-app"
    DeviceTablet      = "tablet"
)

// 更具体的设备标识
func generateDeviceID(deviceType, platform, version string) string {
    return fmt.Sprintf("%s-%s-%s", deviceType, platform, version)
}

// 示例
deviceID := generateDeviceID("web", "chrome", "120")
// 结果: "web-chrome-120"
```

### 前端集成示例

```javascript
// 登录时指定设备
fetch('/api/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    user_id: 'user123',
    device: 'web-browser',  // 或 'mobile-app', 'desktop-app' 等
    password: 'password123'
  })
})

// 获取用户所有设备
fetch('/api/user/user123/devices')

// 检查设备状态
fetch('/api/user/user123/device/mobile-app/status')

// 登出指定设备
fetch('/api/logout', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    user_id: 'user123',
    device: 'mobile-app'
  })
})

// 登出所有设备
fetch('/api/logout-all', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    user_id: 'user123'
  })
})
```