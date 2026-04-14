package auth_provider

import "strings"

// 认证类型常量
const (
	AuthTypeSession = "session" // 服务端会话认证类型
	AuthTypeToken   = "token"   // 固定Token认证类型
)

// 缓存类型常量
const (
	CacheTypeRedis  = "redis"  // Redis缓存
	CacheTypeMemory = "memory" // 内存缓存
)

// GuardConfig guard配置结构
type GuardConfig struct {
	Type                 string   `json:"type"`                   // session | token
	Token                string   `json:"token"`                  // 固定令牌
	Prefix               string   `json:"prefix"`                 // 路由前缀
	Anonymity            []string `json:"anonymity"`              // 匿名路由列表
	Cache                string   `json:"cache"`                  // memory | redis
	Duration             int      `json:"duration"`               // 会话空闲超时时间（秒）
	TouchInterval        int      `json:"touch_interval"`         // 最小续期间隔（秒）
	SingleSessionEnabled bool     `json:"single_session_enabled"` // 单会话登录开关（默认 false）
}

// AuthContext 认证上下文结构
type AuthContext struct {
	GuardName string       `json:"guard_name"` // 当前guard名称
	UserID    string       `json:"user_id"`    // 用户ID
	Token     string       `json:"token"`      // 当前会话令牌
	Session   *SessionData `json:"session"`    // 当前会话数据
	Data      interface{}  `json:"data"`       // 自定义数据
}

// SessionData 服务端会话数据
type SessionData struct {
	TokenHash  string      `json:"token_hash"`
	UserID     string      `json:"user_id"`
	GuardName  string      `json:"guard_name"`
	LoginTime  int64       `json:"login_time"`
	LastSeenAt int64       `json:"last_seen_at"`
	ExpiresAt  int64       `json:"expires_at"`
	Data       interface{} `json:"data,omitempty"`
}

// AuthError 认证错误类型
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// 预定义的认证错误
var (
	ErrTokenMissing        = &AuthError{Code: "TOKEN_MISSING", Message: "token required"}
	ErrTokenInvalid        = &AuthError{Code: "TOKEN_INVALID", Message: "invalid token"}
	ErrSessionExpired      = &AuthError{Code: "SESSION_EXPIRED", Message: "session expired"}
	ErrSessionNotFound     = &AuthError{Code: "SESSION_NOT_FOUND", Message: "session expired"}
	ErrSessionInvalid      = &AuthError{Code: "SESSION_INVALID", Message: "invalid session"}
	ErrGuardNotFound       = &AuthError{Code: "GUARD_NOT_FOUND", Message: "guard not found"}
	ErrAuthTypeUnsupported = &AuthError{Code: "AUTH_TYPE_UNSUPPORTED", Message: "unsupported auth type"}
	ErrPermissionDenied    = &AuthError{Code: "PERMISSION_DENIED", Message: "access denied"}
)

// convertToFriendlyError 将技术性错误转换为用户友好的错误
func convertToFriendlyError(err error) *AuthError {
	if err == nil {
		return nil
	}

	// 如果已经是AuthError，直接返回
	if authErr, ok := err.(*AuthError); ok {
		return authErr
	}

	errMsg := err.Error()

	if strings.Contains(errMsg, "authorization header is missing") {
		return ErrTokenMissing
	}

	if strings.Contains(errMsg, "invalid authorization token") ||
		strings.Contains(errMsg, "token cannot be empty") {
		return ErrTokenInvalid
	}

	if strings.Contains(errMsg, "session not found") ||
		strings.Contains(errMsg, "session not found or expired") {
		return ErrSessionNotFound
	}

	if strings.Contains(errMsg, "session expired") {
		return ErrSessionExpired
	}

	if strings.Contains(errMsg, "invalid session data") {
		return ErrSessionInvalid
	}

	if strings.Contains(errMsg, "unsupported authentication type") {
		return ErrAuthTypeUnsupported
	}

	// 默认返回通用的无效令牌错误
	return ErrTokenInvalid
}

// ConvertToFriendlyError 将错误转换为可返回给客户端的 AuthError
func ConvertToFriendlyError(err error) *AuthError {
	return convertToFriendlyError(err)
}
