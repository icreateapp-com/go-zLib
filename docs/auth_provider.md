# 认证提供者 (Auth Provider)

auth_provider 是 go-zLib 提供的JWT认证解决方案，支持用户登录、登出、token验证和刷新等功能。它基于JWT标准实现，并使用Redis进行会话管理。

## 功能特性

- **JWT认证**: 基于JWT标准的token生成和验证
- **多设备登录管理**: 支持同一用户在不同设备上登录，同一设备只能有一个有效会话
- **设备级会话控制**: 支持登出指定设备或所有设备
- **Redis会话管理**: 使用Redis存储会话信息，支持分布式部署
- **路由认证控制**: 支持匿名路径和受限路径的精细化访问控制
- **配置驱动**: JWT密钥和缓存前缀可通过配置文件管理
- **线程安全**: 使用sync.Once确保JWT密钥安全生成
- **事件发布**: 集成事件总线，支持认证事件通知
- **灵活的认证策略**: 支持缓存认证和特定令牌认证

## 快速开始

### 1. 配置文件

在 `config.yaml` 中添加认证相关配置：

```yaml
auth:
  jwt_secret: "your-jwt-secret-key"           # JWT密钥
  cache_auth_prefix: "AUTH_TOKEN_"            # 缓存前缀，默认为"AUTH_TOKEN_"
  use_cache_auth: true                        # 是否使用缓存认证，默认true
  anonymity:                                  # 匿名访问路径（无需认证）
    - "/.well-known"
    - "/api/partner/v1/auth"
    - "/public"
    - "/health"
  restricted:                                 # 受限访问路径（需要特定令牌认证）
    "/api": "f3c19dfa6334395596384fd4a97b640f"
    "/admin": "admin-secret-token"
```

### 2. 基础使用

```go
package main

import (
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/icreateapp-com/go-zLib/z/provider/auth_provider"
    "github.com/icreateapp-com/go-zLib/z/server/http_server/http_middleware"
)

func main() {
    r := gin.Default()
    
    // 添加认证中间件
    r.Use(http_middleware.AuthMiddleware())
    
    // 登录接口（匿名访问）
    r.POST("/api/auth/login", loginHandler)
    
    // 需要认证的接口
    r.GET("/api/users/profile", getUserProfile)
    r.POST("/api/auth/logout", logoutHandler)
    r.POST("/api/auth/refresh", refreshHandler)
    
    r.Run(":8080")
}

func loginHandler(c *gin.Context) {
    // 验证用户凭据（省略具体实现）
    userID := "user123"
    
    // 获取设备代码（可以从请求头、参数或生成）
    deviceCode := c.GetHeader("X-Device-Code")
    if deviceCode == "" {
        deviceCode = c.Query("device_code")
    }
    if deviceCode == "" {
        // 如果没有提供设备代码，可以生成一个默认的
        deviceCode = "web_browser" // 或者根据 User-Agent 生成
    }
    
    // 生成token，有效期24小时，支持多设备登录
    token, err := auth_provider.AuthProvider.Login(userID, deviceCode, time.Hour*24)
    if err != nil {
        c.JSON(500, gin.H{"error": "登录失败"})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "token": token,
        "device_code": deviceCode,
        "expires_in": 86400, // 24小时
    })
}

func getUserProfile(c *gin.Context) {
    // 从上下文获取用户ID（由认证中间件设置）
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "未认证"})
        return
    }
    
    c.JSON(200, gin.H{
        "user_id": userID,
        "name": "张三",
        "email": "zhangsan@example.com",
    })
}

func logoutHandler(c *gin.Context) {
    // 检查是否指定了设备代码
    deviceCode := c.Query("device_code")
    
    var err error
    if deviceCode == "*" {
        // 登出所有设备
        err = auth_provider.AuthProvider.Logout(c, "*")
    } else if deviceCode != "" {
        // 登出指定设备
        err = auth_provider.AuthProvider.Logout(c, deviceCode)
    } else {
        // 登出当前设备
        err = auth_provider.AuthProvider.Logout(c)
    }
    
    if err != nil {
        c.JSON(500, gin.H{"error": "登出失败"})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "登出成功"})
}

func logoutAllHandler(c *gin.Context) {
    // 登出所有设备
    err := auth_provider.AuthProvider.Logout(c, "*")
    if err != nil {
        c.JSON(500, gin.H{"error": "登出失败"})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "已登出所有设备"})
}

func refreshHandler(c *gin.Context) {
    // 直接传入 Gin 上下文，延长7天
    err := auth_provider.AuthProvider.Refresh(c, time.Hour*24*7)
    if err != nil {
        c.JSON(500, gin.H{"error": "刷新失败"})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "message": "刷新成功"})
}
```

## Bearer Token 处理机制

auth_provider 提供了智能的 Bearer token 处理机制，能够自动识别和处理不同格式的 token 输入。

### 自动格式处理

所有接收 token 参数的方法都支持以下格式：

1. **纯 JWT token**:
   ```
   eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsImV4cCI6MTY5...
   ```

2. **Bearer 前缀格式**:
   ```
   Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsImV4cCI6MTY5...
   ```

3. **带空格的格式**:
   ```
   "  Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsImV4cCI6MTY5...  "
   ```

### 处理规则

- **大小写不敏感**: "Bearer"、"bearer"、"BEARER" 都会被正确识别
- **自动去除空格**: 前后的空格会被自动去除
- **智能提取**: 自动提取 "Bearer " 后面的实际 JWT token
- **向后兼容**: 纯 JWT token 格式仍然完全支持

### 使用示例

```go
// 直接从 Gin 上下文验证 token
userID, isValid := auth_provider.AuthProvider.Verify(c)
if isValid {
    fmt.Printf("token有效，用户ID: %s", userID)
} else {
    fmt.Println("token无效或已过期")
}
```

### HTTP 中间件集成

```go
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 直接传入 Gin 上下文验证 token
        userID, isValid := auth_provider.AuthProvider.Verify(c)
        if !isValid {
            c.JSON(401, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }
        
        c.Set("user_id", userID)
        c.Next()
    }
}
```

### Token 生成

`Login` 方法生成的 token 是纯 JWT 格式，不包含 "Bearer " 前缀：

```go
token, err := auth_provider.AuthProvider.Login("user123", time.Hour*2)
// token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

// 在 HTTP 响应中使用时，可以根据需要添加 Bearer 前缀
c.JSON(200, gin.H{
    "token": token,                    // 纯 token
    "authorization": "Bearer " + token, // 带 Bearer 前缀
})
```

## 多设备登录管理

auth_provider 提供了完整的多设备登录管理功能，支持同一用户在不同设备上登录，同时确保同一设备只能有一个有效会话。

### 核心概念

- **用户ID**: 唯一标识用户身份
- **设备代码**: 唯一标识用户设备（如：mobile_app、web_browser、desktop_app等）
- **设备唯一性**: 同一设备只能有一个有效登录会话
- **多设备支持**: 同一用户可以在不同设备上同时登录

### 设备代码规范

设备代码应该能够唯一标识设备类型或具体设备，建议使用以下格式：

```go
// 按设备类型
"mobile_app"     // 移动应用
"web_browser"    // 网页浏览器
"desktop_app"    // 桌面应用
"api_client"     // API客户端

// 按具体设备（更精确）
"mobile_ios_12345"      // iOS设备 + 设备ID
"web_chrome_67890"      // Chrome浏览器 + 会话ID
"android_app_54321"     // Android应用 + 设备ID
```

### 登录流程

```go
// 1. 用户登录时提供设备代码
func loginHandler(c *gin.Context) {
    userID := "user123"
    deviceCode := c.GetHeader("X-Device-Code") // 从请求头获取
    
    if deviceCode == "" {
        deviceCode = generateDeviceCode(c) // 生成设备代码
    }
    
    // 登录时会自动清除该设备之前的会话
    token, err := auth_provider.AuthProvider.Login(userID, deviceCode, time.Hour*24)
    if err != nil {
        c.JSON(500, gin.H{"error": "登录失败"})
        return
    }
    
    c.JSON(200, gin.H{
        "token": token,
        "device_code": deviceCode,
    })
}

func generateDeviceCode(c *gin.Context) string {
    userAgent := c.GetHeader("User-Agent")
    if strings.Contains(userAgent, "Mobile") {
        return "mobile_app"
    } else if strings.Contains(userAgent, "Chrome") {
        return "web_chrome"
    }
    return "web_browser"
}
```

### 登出管理

#### 1. 登出当前设备

```go
func logoutCurrentDevice(c *gin.Context) {
    err := auth_provider.AuthProvider.Logout(c)
    if err != nil {
        c.JSON(500, gin.H{"error": "登出失败"})
        return
    }
    c.JSON(200, gin.H{"message": "当前设备登出成功"})
}
```

#### 2. 登出指定设备

```go
func logoutSpecificDevice(c *gin.Context) {
    deviceCode := c.Param("device_code") // 从路径参数获取
    
    err := auth_provider.AuthProvider.Logout(c, deviceCode)
    if err != nil {
        c.JSON(500, gin.H{"error": "登出失败"})
        return
    }
    c.JSON(200, gin.H{"message": "指定设备登出成功"})
}
```

#### 3. 登出所有设备

```go
func logoutAllDevices(c *gin.Context) {
    err := auth_provider.AuthProvider.Logout(c, "*")
    if err != nil {
        c.JSON(500, gin.H{"error": "登出失败"})
        return
    }
    c.JSON(200, gin.H{"message": "所有设备登出成功"})
}
```

### 设备管理

#### 获取用户所有在线设备

```go
func getUserDevices(c *gin.Context) {
    userID, err := auth_provider.AuthProvider.GetUserID(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "未认证"})
        return
    }
    
    devices, err := auth_provider.AuthProvider.GetUserDevices(userID)
    if err != nil {
        c.JSON(500, gin.H{"error": "获取设备列表失败"})
        return
    }
    
    c.JSON(200, gin.H{
        "devices": devices,
        "count": len(devices),
    })
}
```

#### 获取当前设备代码

```go
func getCurrentDevice(c *gin.Context) {
    deviceCode, err := auth_provider.AuthProvider.GetDeviceCode(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "未认证"})
        return
    }
    
    c.JSON(200, gin.H{"device_code": deviceCode})
}
```

### 管理员功能

#### 强制登出用户所有设备

```go
func adminLogoutUser(c *gin.Context) {
    targetUserID := c.Param("user_id")
    
    err := auth_provider.AuthProvider.LogoutAllDevices(targetUserID)
    if err != nil {
        c.JSON(500, gin.H{"error": "操作失败"})
        return
    }
    
    c.JSON(200, gin.H{"message": "用户所有设备已登出"})
}
```

#### 强制登出指定用户的指定设备

```go
func adminLogoutUserDevice(c *gin.Context) {
    targetUserID := c.Param("user_id")
    deviceCode := c.Param("device_code")
    
    err := auth_provider.AuthProvider.LogoutDevice(targetUserID, deviceCode)
    if err != nil {
        c.JSON(500, gin.H{"error": "操作失败"})
        return
    }
    
    c.JSON(200, gin.H{"message": "指定设备已登出"})
}
```

### 路由配置示例

```go
func setupAuthRoutes(r *gin.Engine) {
    auth := r.Group("/api/auth")
    {
        auth.POST("/login", loginHandler)
        auth.POST("/logout", logoutCurrentDevice)
        auth.POST("/logout/:device_code", logoutSpecificDevice)
        auth.POST("/logout-all", logoutAllDevices)
        auth.GET("/devices", getUserDevices)
        auth.GET("/device", getCurrentDevice)
    }
    
    admin := r.Group("/api/admin")
    admin.Use(adminAuthMiddleware()) // 管理员认证中间件
    {
        admin.POST("/users/:user_id/logout", adminLogoutUser)
        admin.POST("/users/:user_id/logout/:device_code", adminLogoutUserDevice)
    }
}
```

### 最佳实践

1. **设备代码生成**: 根据客户端类型和特征生成有意义的设备代码
2. **设备识别**: 结合 User-Agent、IP地址等信息提高设备识别准确性
3. **安全考虑**: 敏感操作（如登出所有设备）应该要求额外验证
4. **用户体验**: 提供设备管理界面，让用户查看和管理登录设备
5. **监控告警**: 监控异常登录行为，如短时间内多设备登录

## API 参考

### AuthProvider 实例

全局认证提供者实例，提供所有认证相关功能。

```go
var AuthProvider authProvider
```

### Login 方法

用户登录，生成JWT token并存储到Redis，支持多设备登录管理。

```go
func (a *authProvider) Login(userID string, deviceCode string, expiration time.Duration) (string, error)
```

**参数:**
- `userID`: 用户ID
- `deviceCode`: 设备代码，用于区分不同设备
- `expiration`: token过期时间

**返回:**
- `string`: 生成的JWT token（不包含 "Bearer " 前缀）
- `error`: 错误信息

**特性:**
- 同一用户可以在不同设备上登录
- 同一设备重复登录会清除之前的会话
- 自动管理设备与用户的映射关系

**示例:**
```go
token, err := auth_provider.AuthProvider.Login("user123", "mobile_app", time.Hour*24)
if err != nil {
    log.Printf("登录失败: %v", err)
    return
}
fmt.Printf("登录成功，token: %s", token)
```

### Logout 方法

用户登出，支持设备级登出管理。

```go
func (a *authProvider) Logout(c *gin.Context, deviceCode ...string) error
```

**参数:**
- `c`: Gin 上下文，自动从 Authorization 头获取 token
- `deviceCode`: 可选的设备代码，指定要登出的设备

**返回:**
- `error`: 错误信息

**行为:**
- 如果不提供 `deviceCode`，登出当前token对应的设备
- 如果提供 `deviceCode`，登出指定设备（需要当前用户有权限）
- 自动清理设备映射关系

**示例:**
```go
// 登出当前设备
err := auth_provider.AuthProvider.Logout(c)
if err != nil {
    log.Printf("登出失败: %v", err)
    return
}

// 登出指定设备
err := auth_provider.AuthProvider.Logout(c, "mobile_app")
if err != nil {
    log.Printf("登出指定设备失败: %v", err)
    return
}
fmt.Println("登出成功")
```

### Verify 方法

验证token是否有效。

```go
func (a *authProvider) Verify(c *gin.Context) (string, bool)
```

**参数:**
- `c`: Gin 上下文，自动从 Authorization 头获取 token

**返回:**
- `string`: 用户ID
- `bool`: 是否有效

**示例:**
```go
userID, isValid := auth_provider.AuthProvider.Verify(c)
if isValid {
    fmt.Printf("token有效，用户ID: %s", userID)
} else {
    fmt.Println("token无效或已过期")
}
```

### Refresh 方法

刷新token过期时间。

```go
func (a *authProvider) Refresh(c *gin.Context, newExpiration time.Duration) error
```

**参数:**
- `c`: Gin 上下文，自动从 Authorization 头获取 token
- `newExpiration`: 新的过期时间

**返回:**
- `error`: 错误信息

**示例:**
```go
err := auth_provider.AuthProvider.Refresh(c, time.Hour*24*7)
if err != nil {
    log.Printf("刷新失败: %v", err)
    return
}
fmt.Println("刷新成功")
```

### GetUserID 方法

从token中获取用户ID（不验证Redis会话）。

```go
func (a *authProvider) GetUserID(c *gin.Context) (string, error)
```

**参数:**
- `c`: Gin 上下文，自动从 Authorization 头获取 token

**返回:**
- `string`: 用户ID
- `error`: 错误信息

**示例:**
```go
userID, err := auth_provider.AuthProvider.GetUserID(c)
if err != nil {
    log.Printf("获取用户ID失败: %v", err)
    return
}
fmt.Printf("用户ID: %s", userID)
```

### GetTokenClaims 方法

从token中获取完整的声明信息（不验证Redis会话）。

```go
func (a *authProvider) GetTokenClaims(c *gin.Context) (*AuthClaims, error)
```

**参数:**
- `c`: Gin 上下文，自动从 Authorization 头获取 token

**返回:**
- `*AuthClaims`: 完整的JWT声明信息
- `error`: 错误信息

**示例:**
```go
claims, err := auth_provider.AuthProvider.GetTokenClaims(c)
if err != nil {
    log.Printf("获取token声明失败: %v", err)
    return
}
fmt.Printf("用户ID: %s, 设备代码: %s", claims.UserID, claims.DeviceCode)
```



### LogoutDevice 方法

登出指定用户的指定设备。

```go
func (a *authProvider) LogoutDevice(userID, deviceCode string) error
```

**参数:**
- `userID`: 用户ID
- `deviceCode`: 设备代码

**返回:**
- `error`: 错误信息

**示例:**
```go
err := auth_provider.AuthProvider.LogoutDevice("user123", "mobile_app")
if err != nil {
    log.Printf("登出设备失败: %v", err)
    return
}
fmt.Println("设备登出成功")
```

### LogoutAllDevices 方法

登出指定用户的所有设备。

```go
func (a *authProvider) LogoutAllDevices(userID string) error
```

**参数:**
- `userID`: 用户ID

**返回:**
- `error`: 错误信息

**示例:**
```go
err := auth_provider.AuthProvider.LogoutAllDevices("user123")
if err != nil {
    log.Printf("登出所有设备失败: %v", err)
    return
}
fmt.Println("已登出用户所有设备")
```

### GetDeviceCode 方法

从当前上下文获取设备代码。

```go
func (a *authProvider) GetDeviceCode(c *gin.Context) (string, error)
```

**参数:**
- `c`: Gin 上下文，自动从 Authorization 头获取 token

**返回:**
- `string`: 设备代码
- `error`: 错误信息

**示例:**
```go
deviceCode, err := auth_provider.AuthProvider.GetDeviceCode(c)
if err != nil {
    log.Printf("获取设备代码失败: %v", err)
    return
}
fmt.Printf("设备代码: %s", deviceCode)
```

### GetUserDevices 方法

获取指定用户的所有设备列表。

```go
func (a *authProvider) GetUserDevices(userID string) ([]string, error)
```

**参数:**
- `userID`: 用户ID

**返回:**
- `[]string`: 设备代码列表
- `error`: 错误信息

**示例:**
```go
devices, err := auth_provider.AuthProvider.GetUserDevices("user123")
if err != nil {
    log.Printf("获取设备列表失败: %v", err)
    return
}
fmt.Printf("用户设备: %v", devices)
```

### IsAnonymousPath 方法

检查路径是否允许匿名访问。

```go
func (a *authProvider) IsAnonymousPath(path string) bool
```

**参数:**
- `path`: 请求路径

**返回:**
- `bool`: 是否允许匿名访问

**示例:**
```go
isAnonymous := auth_provider.AuthProvider.IsAnonymousPath("/public/info")
if isAnonymous {
    fmt.Println("路径允许匿名访问")
}
```

### IsRestrictedPath 方法

检查路径是否需要特定令牌认证。

```go
func (a *authProvider) IsRestrictedPath(path string) (bool, string)
```

**参数:**
- `path`: 请求路径

**返回:**
- `bool`: 是否为受限路径
- `string`: 所需的认证令牌

**示例:**
```go
isRestricted, token := auth_provider.AuthProvider.IsRestrictedPath("/api/users")
if isRestricted {
    fmt.Printf("路径需要认证，所需令牌: %s", token)
}
```

### IsUseCacheAuth 方法

检查是否使用缓存认证方案。

```go
func (a *authProvider) IsUseCacheAuth() bool
```

**返回:**
- `bool`: 是否使用缓存认证

**示例:**
```go
useCacheAuth := auth_provider.AuthProvider.IsUseCacheAuth()
if useCacheAuth {
    fmt.Println("使用缓存认证方案")
}
```

## 路由认证逻辑

auth_provider 提供了灵活的路由认证控制机制，支持三种访问模式：

### 1. 匿名访问 (Anonymity)

配置在 `auth.anonymity` 中的路径前缀允许匿名访问，无需任何认证。

```yaml
auth:
  anonymity:
    - "/.well-known"          # OAuth2/OpenID Connect 发现端点
    - "/api/partner/v1/auth"  # 合作伙伴认证接口
    - "/public"               # 公共资源
    - "/health"               # 健康检查
```

**使用示例:**
```go
// 检查路径是否允许匿名访问
if auth_provider.AuthProvider.IsAnonymousPath("/public/logo.png") {
    // 直接处理请求，无需认证
    handleRequest(c)
    return
}
```

### 2. 受限访问 (Restricted)

配置在 `auth.restricted` 中的路径前缀需要特定的认证令牌。

```yaml
auth:
  restricted:
    "/api": "f3c19dfa6334395596384fd4a97b640f"    # API接口需要特定令牌
    "/admin": "admin-secret-token"                # 管理后台需要管理员令牌
```

**使用示例:**
```go
// 检查路径是否为受限路径
isRestricted, requiredToken := auth_provider.AuthProvider.IsRestrictedPath("/api/users")
if isRestricted {
    // 验证请求中的令牌是否匹配
    providedToken := c.GetHeader("X-API-Token")
    if providedToken != requiredToken {
        c.JSON(403, gin.H{"error": "Invalid API token"})
        return
    }
}
```

### 3. 标准JWT认证

既不在匿名列表也不在受限列表中的路径，使用标准的JWT认证流程。

**认证流程:**
1. 从请求头获取 `Authorization: Bearer <token>`
2. 验证JWT token的有效性
3. 如果启用缓存认证，检查Redis中的会话
4. 将用户ID存储到请求上下文中

### 认证优先级

路径认证按以下优先级进行判断：

1. **匿名路径** - 如果匹配匿名路径前缀，直接允许访问
2. **受限路径** - 如果匹配受限路径前缀，验证特定令牌
3. **JWT认证** - 其他路径使用标准JWT认证

```go
func authenticationFlow(path string) {
    // 1. 检查匿名路径
    if auth_provider.AuthProvider.IsAnonymousPath(path) {
        return // 允许访问
    }
    
    // 2. 检查受限路径
    if isRestricted, token := auth_provider.AuthProvider.IsRestrictedPath(path); isRestricted {
        // 验证特定令牌
        validateSpecificToken(token)
        return
    }
    
    // 3. 标准JWT认证
    validateJWTToken()
}
```

### 缓存认证控制

通过 `use_cache_auth` 配置可以控制是否使用Redis缓存认证：

```yaml
auth:
  use_cache_auth: true  # 启用缓存认证（默认）
```

- **启用缓存认证**: 验证JWT token + 检查Redis会话
- **禁用缓存认证**: 仅验证JWT token，不检查Redis会话

**使用场景:**
- 启用缓存认证：适用于需要强制登出功能的场景
- 禁用缓存认证：适用于无状态的API服务，提高性能

### 实际应用示例

```go
func setupRoutes() {
    r := gin.Default()
    
    // 使用认证中间件
    r.Use(authMiddleware())
    
    // 匿名访问路径
    r.GET("/.well-known/openid_configuration", openIDConfig)  // 自动允许
    r.POST("/api/partner/v1/auth", partnerAuth)               // 自动允许
    
    // 受限访问路径
    r.GET("/api/users", getUsers)        // 需要API令牌
    r.GET("/admin/dashboard", dashboard) // 需要管理员令牌
    
    // 标准JWT认证路径
    r.GET("/profile", getUserProfile)    // 需要JWT认证
    r.POST("/logout", logout)            // 需要JWT认证
}

func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path
        
        // 1. 检查匿名路径
        if auth_provider.AuthProvider.IsAnonymousPath(path) {
            c.Next()
            return
        }
        
        // 2. 检查受限路径
        if isRestricted, requiredToken := auth_provider.AuthProvider.IsRestrictedPath(path); isRestricted {
            providedToken := c.GetHeader("X-API-Token")
            if providedToken != requiredToken {
                c.JSON(403, gin.H{"error": "Invalid API token"})
                c.Abort()
                return
            }
            c.Next()
            return
        }
        
        // 3. 标准JWT认证
        token := extractBearerToken(c)
        userID, isValid := auth_provider.AuthProvider.Verify(token)
        if !isValid {
            c.JSON(401, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }
        
        c.Set("user_id", userID)
        c.Next()
    }
}
```

## 中间件集成

### AuthMiddleware

认证中间件，自动验证请求中的JWT token。

```go
import "github.com/icreateapp-com/go-zLib/z/server/http_server/http_middleware"

r := gin.Default()
r.Use(http_middleware.AuthMiddleware())
```

**功能:**
- 自动从请求头获取Authorization token
- 支持Bearer token格式
- 验证token有效性
- 将用户ID存储到Gin上下文中
- 支持匿名访问路径配置
- 发布认证事件

**匿名路径配置:**
```yaml
auth:
  anonymity:
    - "/.well-known"
    - "/api/partner/v1/auth"
    - "/public"
    - "/health"
  restricted:
    "/api": "f3c19dfa6334395596384fd4a97b640f"
    "/admin": "admin-secret-token"
```

## 事件集成

auth_provider 集成了事件总线，会发布以下事件：

### app.auth.verify

认证验证事件，在每次token验证时触发。

```go
import "github.com/icreateapp-com/go-zLib/z/provider/event_bus_provider"

// 监听认证事件
event_bus_provider.On("app.auth.verify", func(event event_bus_provider.Event) {
    isValid := event.Payload.(bool)
    if isValid {
        fmt.Println("用户认证成功")
    } else {
        fmt.Println("用户认证失败")
    }
})
```

## 配置说明

### 必需配置

```yaml
auth:
  jwt_secret: "your-jwt-secret-key"  # JWT签名密钥
```

### 可选配置

```yaml
auth:
  cache_auth_prefix: "AUTH_TOKEN_"   # Redis缓存键前缀，默认"AUTH_TOKEN_"
  use_cache_auth: true               # 是否使用缓存认证，默认true
  anonymity:                         # 匿名访问路径列表（无需认证）
    - "/.well-known"
    - "/api/partner/v1/auth"
    - "/public"
    - "/health"
  restricted:                        # 受限访问路径（需要特定令牌认证）
    "/api": "f3c19dfa6334395596384fd4a97b640f"
    "/admin": "admin-secret-token"
```

### 配置说明

- **jwt_secret**: JWT签名密钥，用于生成和验证JWT token
- **cache_auth_prefix**: Redis缓存键前缀，用于区分不同应用的认证缓存
- **use_cache_auth**: 是否启用缓存认证方案，关闭后仅验证JWT不检查Redis会话
- **anonymity**: 匿名访问路径列表，这些路径无需任何认证即可访问
- **restricted**: 受限访问路径映射，key为路径前缀，value为所需的特定认证令牌

## 安全考虑

### 1. JWT密钥管理

- JWT密钥应该足够复杂，建议使用32位以上的随机字符串
- 生产环境中应通过环境变量或安全的配置管理系统提供密钥
- 定期轮换JWT密钥以提高安全性

```yaml
# 生产环境配置示例
auth:
  jwt_secret: "${JWT_SECRET_KEY}"  # 从环境变量获取
```

### 2. Token过期策略

- 设置合理的token过期时间，平衡安全性和用户体验
- 实现token刷新机制，避免用户频繁重新登录
- 考虑实现滑动过期时间

```go
// 短期token + 刷新机制
shortToken, _ := auth_provider.AuthProvider.Login(userID, time.Hour*2)    // 2小时
longToken, _ := auth_provider.AuthProvider.Login(userID, time.Hour*24*7)  // 7天
```

### 3. 会话管理

- Redis会话存储提供了集中式的会话管理
- 支持强制登出（删除Redis中的会话）
- 可以实现单点登录（SSO）控制

## 错误处理

### 常见错误类型

```go
// token为空
userID, isValid := auth_provider.AuthProvider.Verify("")
// isValid = false

// token格式错误
userID, isValid := auth_provider.AuthProvider.Verify("invalid-token")
// isValid = false

// token已过期或会话不存在
userID, isValid := auth_provider.AuthProvider.Verify("expired-token")
// isValid = false
```

### 错误处理最佳实践

```go
func handleAuthError(c *gin.Context, err error) {
    switch {
    case strings.Contains(err.Error(), "token cannot be empty"):
        c.JSON(400, gin.H{"error": "缺少认证token", "code": 40001})
    case strings.Contains(err.Error(), "session not found"):
        c.JSON(401, gin.H{"error": "会话已过期", "code": 40101})
    case strings.Contains(err.Error(), "invalid token"):
        c.JSON(401, gin.H{"error": "无效的token", "code": 40102})
    default:
        c.JSON(500, gin.H{"error": "认证服务异常", "code": 50001})
    }
}
```

## 性能优化

### 1. Redis连接池

确保Redis连接池配置合理：

```yaml
redis:
  host: "localhost"
  port: 6379
  pool_size: 10
  min_idle_conns: 5
```

### 2. 缓存策略

- 合理设置缓存前缀，避免键冲突
- 使用适当的过期时间，避免内存浪费
- 考虑使用Redis集群提高可用性

### 3. JWT优化

- JWT payload保持精简，只包含必要信息
- 避免在JWT中存储敏感信息
- 考虑使用压缩算法减少token大小

## 监控和日志

### 1. 认证指标

```go
// 监控认证成功率
event_bus_provider.On("app.auth.verify", func(event event_bus_provider.Event) {
    isValid := event.Payload.(bool)
    if isValid {
        metrics.IncrementCounter("auth_success")
    } else {
        metrics.IncrementCounter("auth_failure")
    }
})
```

### 2. 日志记录

auth_provider 会自动记录关键操作：

- 用户登录成功/失败
- token验证结果
- 会话刷新操作
- 错误信息

## 最佳实践

### 1. 登录流程

```go
func loginFlow(username, password string) (string, error) {
    // 1. 验证用户凭据
    user, err := validateCredentials(username, password)
    if err != nil {
        return "", err
    }
    
    // 2. 生成token
    token, err := auth_provider.AuthProvider.Login(user.ID, time.Hour*24)
    if err != nil {
        return "", err
    }
    
    // 3. 记录登录日志
    logUserLogin(user.ID, time.Now())
    
    return token, nil
}
```

### 2. 登出流程

```go
func logoutFlow(token string) error {
    // 1. 获取用户信息
    userID, err := auth_provider.AuthProvider.GetUserID(token)
    if err != nil {
        return err
    }
    
    // 2. 执行登出
    err = auth_provider.AuthProvider.Logout(token)
    if err != nil {
        return err
    }
    
    // 3. 记录登出日志
    logUserLogout(userID, time.Now())
    
    return nil
}
```

### 3. 权限控制

```go
func checkPermission(c *gin.Context, requiredRole string) bool {
    userID, exists := c.Get("user_id")
    if !exists {
        return false
    }
    
    // 从数据库获取用户角色
    userRole, err := getUserRole(userID.(string))
    if err != nil {
        return false
    }
    
    return userRole == requiredRole || userRole == "admin"
}

// 使用示例
r.GET("/api/admin/users", func(c *gin.Context) {
    if !checkPermission(c, "admin") {
        c.JSON(403, gin.H{"error": "权限不足"})
        return
    }
    
    // 管理员操作...
})
```

## 总结

auth_provider 提供了完整的JWT认证解决方案，具有以下优势：

- **简单易用**: 提供简洁的API接口
- **功能完整**: 支持登录、登出、验证、刷新等完整流程
- **高性能**: 基于Redis的会话管理，支持分布式部署
- **安全可靠**: 遵循JWT标准，支持配置化的安全策略
- **易于集成**: 提供中间件和事件集成，方便与现有系统集成

通过合理配置和使用auth_provider，可以快速构建安全可靠的认证系统。