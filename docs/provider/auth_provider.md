# 认证提供者 (Auth Provider)

`auth_provider` 提供统一的认证能力，当前支持两种底层模式：

- `session`: 服务端会话认证。登录后返回稳定的随机 token，会话数据存储在 Redis 或内存中，支持滑动续期。
- `token`: 固定令牌认证。适合内部服务、Webhook 或简单的系统间调用。

当前实现已经不再使用 JWT 登录态，也不需要 refresh token 机制。

## 功能特性

- 多 Guard 配置，支持不同路由前缀隔离
- 服务端 session 存储
- HTTP 中间件自动认证
- WebSocket 握手认证与消息活跃续期
- `touch_interval` 控制的滑动会话续期
- 可选单设备登录
- 统一错误类型与上下文访问

## 配置说明

### Guard 配置

```yaml
auth:
  guards:
    api:
      type: session                    # session | token
      token: ""                        # type=token 时使用
      prefix: /api
      anonymity:
        - /api/login
        - /api/health
      cache: redis                     # redis | memory
      duration: 259200                 # 会话空闲超时，单位秒
      touch_interval: 300              # 最小续期间隔，单位秒
      single_session_enabled: false    # true 时新登录会踢掉旧会话

    internal:
      type: token
      token: internal-service-token
      prefix: /internal
      anonymity: []
      cache: memory
```

### 字段含义

- `type`: 认证模式，支持 `session` 或 `token`
- `token`: 固定 token 模式使用的令牌
- `prefix`: 路由前缀，用于自动匹配 guard
- `anonymity`: 匿名路由列表，匹配到后跳过认证
- `cache`: 认证数据存储位置，支持 `redis` 和 `memory`
- `duration`: 会话空闲超时时间。超过该时长无活跃操作，会话失效
- `touch_interval`: 最小续期间隔。只有距离上次续期超过该值时，才会执行一次续期写入
- `single_session_enabled`: 是否只允许用户保留一个有效会话

## 工作方式

### Session 模式

1. 调用 `Login` 生成随机 token
2. token 对应的 session 数据写入 Redis 或内存
3. HTTP 请求通过认证中间件时自动校验 token
4. 若距离上次续期超过 `touch_interval`，自动延长 session TTL
5. WebSocket 握手时完成认证，收到消息时按 `touch_interval` 续期
6. 调用 `Logout` 删除当前 token 对应的 session

### Token 模式

1. 请求携带固定 token
2. 服务端直接比对配置中的 `token`
3. 不走登录态签发，也不做会话续期

## 在 fx 中注册

```go
package main

import (
    "github.com/icreateapp-com/go-zLib/z/providers/auth_provider"
    "go.uber.org/fx"
)

func main() {
    app := fx.New(
        auth_provider.AuthProviderModule,
    )

    app.Run()
}
```

## 登录与登出

### Login

签名：

```go
func (a *Auth) Login(guard string, userID string, duration time.Duration, data ...interface{}) (string, error)
```

说明：

- `guard`: guard 名称
- `userID`: 当前登录用户 ID
- `duration`: 本次登录的会话空闲超时时间。传 `<= 0` 时回退到配置中的 `duration`
- `data`: 可选的自定义用户数据，会存入 session

示例：

```go
token, err := auth.Login("api", "user123", 24*time.Hour, map[string]interface{}{
    "name": "张三",
    "role": "admin",
})
```

### Logout

签名：

```go
func (a *Auth) Logout(guard, token string) error
```

说明：

- 按当前 token 登出当前会话
- 支持自动剥离 `Bearer ` 前缀

示例：

```go
err := auth.Logout("api", c.GetHeader("Authorization"))
```

### LogoutAll

签名：

```go
func (a *Auth) LogoutAll(guard, userID string) error
```

示例：

```go
err := auth.LogoutAll("api", "user123")
```

## HTTP 认证

### 中间件入口

```go
func (a *Auth) Authenticate(c *gin.Context) (bool, string, error)
```

典型行为：

- 从 `Authorization` 头读取 token
- 若为空，再读取 `?token=...`
- 根据当前路由所属的 guard 进行校验
- 成功后将认证结果写入 gin context

写入的上下文键：

- `auth.guard`
- `auth.user_id`
- `auth.token`
- `auth.session`
- `auth.data`

### Gin 中使用

```go
func ProtectedHandler(auth *auth_provider.Auth) gin.HandlerFunc {
    return func(c *gin.Context) {
        ok, _, err := auth.Authenticate(c)
        if !ok || err != nil {
            c.JSON(401, gin.H{"error": err.Error()})
            c.Abort()
            return
        }

        userID, err := auth.GetUserID(c)
        if err != nil {
            c.JSON(401, gin.H{"error": err.Error()})
            c.Abort()
            return
        }

        c.JSON(200, gin.H{"user_id": userID})
    }
}
```

## WebSocket 认证

WebSocket 服务默认支持两类行为：

- 握手时读取 `Authorization` 头或 `?token=...`
- 握手认证成功后，消息到达时按 `touch_interval` 触发 session 续期

示例：

```text
ws://localhost:8080/ws?guard=api&token=your-session-token
```

注意：

- 如果连接长时间保持，但完全没有消息收发，不会自动续期
- 如果续期时发现 session 已失效，连接会被关闭

## 认证结果读取

### GetUserID

```go
userID, err := auth.GetUserID(c)
```

### GetData

```go
data, err := auth.GetData(c)
```

### GetToken

```go
token, err := auth.GetToken(c)
```

### GetSession

```go
session, err := auth.GetSession(c)
```

`SessionData` 结构：

```go
type SessionData struct {
    TokenHash  string
    UserID     string
    GuardName  string
    LoginTime  int64
    LastSeenAt int64
    ExpiresAt  int64
    Data       interface{}
}
```

## 令牌获取方式

### Authorization Header

推荐方式：

```bash
curl -H "Authorization: Bearer your-session-token" http://localhost:8080/api/profile
```

也支持不带 `Bearer` 前缀：

```bash
curl -H "Authorization: your-session-token" http://localhost:8080/api/profile
```

### URL 参数

适合 WebSocket 或受限客户端：

```bash
curl "http://localhost:8080/api/profile?token=your-session-token"
ws://localhost:8080/ws?guard=api&token=your-session-token
```

## 单设备登录

配置：

```yaml
auth:
  guards:
    api:
      type: session
      single_session_enabled: true
```

行为：

- 新会话登录时，会清理该用户旧的所有 session
- 旧 token 随即失效

## 错误处理

错误类型：

```go
type AuthError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

常见错误：

| 错误代码 | 错误消息 | 说明 |
|---------|---------|------|
| `TOKEN_MISSING` | `token required` | 缺少 token |
| `TOKEN_INVALID` | `invalid token` | token 无效 |
| `SESSION_EXPIRED` | `session expired` | session 已过期 |
| `SESSION_NOT_FOUND` | `session expired` | session 不存在或已过期 |
| `SESSION_INVALID` | `invalid session` | session 数据损坏或无效 |
| `GUARD_NOT_FOUND` | `guard not found` | guard 不存在 |
| `AUTH_TYPE_UNSUPPORTED` | `unsupported auth type` | 不支持的认证类型 |
| `PERMISSION_DENIED` | `access denied` | 无权限访问 |

统一返回示例：

```json
{
  "error": "SESSION_NOT_FOUND",
  "message": "session expired"
}
```

## 最佳实践

### 1. 用户登录态使用 session

```yaml
auth:
  guards:
    admin:
      type: session
      cache: redis
      duration: 259200
      touch_interval: 300
```

适合后台、控制台、普通用户登录态。

### 2. 内部服务调用使用固定 token

```yaml
auth:
  guards:
    internal:
      type: token
      token: internal-service-token
      cache: memory
```

适合内网服务、Webhook、管理接口。

### 3. `touch_interval` 不要太小

建议：

- 高频 API / WebSocket 场景：`300` 秒左右
- 安全要求更高时，可按业务缩短

过小会导致 Redis 写入频率过高，失去节流意义。

### 4. 前端不要依赖固定过期时间强退

session 模式是滑动续期：

- 只要近期有操作，会话就会继续延长
- 前端应以服务端返回的 401 为准，而不是只看首次登录返回的过期时间

## 故障排查

### 登录后很快掉线

检查：

- `duration` 是否配置过小
- 前端是否仍在按固定 `expires_at` 本地强退
- Redis TTL 是否正常写入

### 活跃用户仍被踢下线

检查：

- `touch_interval` 是否过大
- 请求是否真的经过 HTTP 中间件
- WebSocket 是否存在长连接但没有任何消息

### 单设备登录不生效

检查：

- `single_session_enabled` 是否为 `true`
- 是否多个 guard 混用导致登录态不在同一个 guard 下

### 固定 token 调用失败

检查：

- `type` 是否配置为 `token`
- 请求头中的 token 是否与配置值完全一致
