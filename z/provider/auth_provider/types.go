package auth_provider

import (
	"strings"
	
	"github.com/golang-jwt/jwt/v5"
)

// 认证类型常量
const (
	AuthTypeJWT   = "jwt"   // JWT认证类型
	AuthTypeToken = "token" // 固定Token认证类型
)

// 缓存类型常量
const (
	CacheTypeRedis  = "redis"  // Redis缓存
	CacheTypeMemory = "memory" // 内存缓存
)

// GuardConfig guard配置结构
type GuardConfig struct {
	Type       string   `json:"type"`        // jwt | token
	Token      string   `json:"token"`       // 固定令牌
	Prefix     string   `json:"prefix"`      // 路由前缀
	Anonymity  []string `json:"anonymity"`   // 匿名路由列表
	Cache      string   `json:"cache"`       // memory | redis
	SSOEnabled bool     `json:"sso_enabled"` // 单点登录开关
}

// AuthContext 认证上下文结构
type AuthContext struct {
	GuardName string                 `json:"guard_name"` // 当前guard名称
	UserID    string                 `json:"user_id"`    // 用户ID
	Device    string                 `json:"device"`     // 设备标识
	Data      map[string]interface{} `json:"data"`       // 自定义数据
}

// MultiTenantClaims 多租户JWT声明结构
type MultiTenantClaims struct {
	UserID    string `json:"user_id"`
	GuardName string `json:"guard_name"` // guard名称
	jwt.RegisteredClaims
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
	ErrTokenMissing       = &AuthError{Code: "TOKEN_MISSING", Message: "token required"}
	ErrTokenInvalid       = &AuthError{Code: "TOKEN_INVALID", Message: "invalid token"}
	ErrTokenExpired       = &AuthError{Code: "TOKEN_EXPIRED", Message: "token expired"}
	ErrTokenMalformed     = &AuthError{Code: "TOKEN_MALFORMED", Message: "malformed token"}
	ErrTokenSignature     = &AuthError{Code: "TOKEN_SIGNATURE", Message: "invalid signature"}
	ErrSessionNotFound    = &AuthError{Code: "SESSION_NOT_FOUND", Message: "session expired"}
	ErrSessionInvalid     = &AuthError{Code: "SESSION_INVALID", Message: "invalid session"}
	ErrGuardNotFound      = &AuthError{Code: "GUARD_NOT_FOUND", Message: "guard not found"}
	ErrGuardMismatch      = &AuthError{Code: "GUARD_MISMATCH", Message: "token mismatch"}
	ErrAuthTypeUnsupported = &AuthError{Code: "AUTH_TYPE_UNSUPPORTED", Message: "unsupported auth type"}
	ErrPermissionDenied   = &AuthError{Code: "PERMISSION_DENIED", Message: "access denied"}
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
	
	// JWT相关错误
	if strings.Contains(errMsg, "token signature is invalid") || 
	   strings.Contains(errMsg, "signature is invalid") {
		return ErrTokenSignature
	}
	
	if strings.Contains(errMsg, "token is expired") ||
	   strings.Contains(errMsg, "token has expired") {
		return ErrTokenExpired
	}
	
	if strings.Contains(errMsg, "token is malformed") ||
	   strings.Contains(errMsg, "token contains an invalid number of segments") {
		return ErrTokenMalformed
	}
	
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
	
	if strings.Contains(errMsg, "invalid session data") {
		return ErrSessionInvalid
	}
	
	if strings.Contains(errMsg, "token guard mismatch") {
		return ErrGuardMismatch
	}
	
	if strings.Contains(errMsg, "unsupported authentication type") {
		return ErrAuthTypeUnsupported
	}
	
	// 默认返回通用的无效令牌错误
	return ErrTokenInvalid
}