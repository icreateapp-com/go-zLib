package auth_provider

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/icreateapp-com/go-zLib/z"
	. "github.com/icreateapp-com/go-zLib/z"
)

// AuthClaims JWT声明结构
type AuthClaims struct {
	UserID     string `json:"user_id"`
	DeviceCode string `json:"device_code"` // 设备代码，用于多设备登录管理
	jwt.RegisteredClaims
}

// authProvider 认证提供者结构
type authProvider struct {
	jwtSecret []byte    // JWT密钥
	once      sync.Once // 确保JWT密钥只生成一次
}

// AuthProvider 全局认证提供者实例
var AuthProvider authProvider

// extractToken 从 token 字符串中提取实际的 JWT token，自动处理 "Bearer " 前缀
func (a *authProvider) extractToken(token string) string {
	// 去除前后空格
	token = strings.TrimSpace(token)

	// 检查是否包含 "Bearer " 前缀（不区分大小写）
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		// 去除 "Bearer " 前缀（7个字符）
		return strings.TrimSpace(token[7:])
	}

	return token
}

// getTokenFromContext 从 gin 上下文中获取 Authorization token
func (a *authProvider) getTokenFromContext(c interface{}) (string, error) {
	// 尝试将接口转换为 gin.Context
	if ginCtx, ok := c.(*gin.Context); ok {
		// 从请求头获取 Authorization token
		authHeader := ginCtx.Request.Header.Get("Authorization")
		if StringIsEmpty(authHeader) {
			return "", fmt.Errorf("authorization header is missing")
		}

		// 提取实际的 JWT token
		return a.extractToken(authHeader), nil
	}

	return "", fmt.Errorf("invalid context type, expected *gin.Context")
}

// getJWTSecret 获取JWT密钥，首次调用时从配置文件获取
func (a *authProvider) getJWTSecret() []byte {
	a.once.Do(func() {
		// 从配置文件获取密钥
		if configKey, err := Config.String("config.key"); err == nil && !StringIsEmpty(configKey) {
			a.jwtSecret = []byte(configKey)
		} else {
			// 配置文件中没有密钥，生成随机密钥
			secret := make([]byte, 32)
			if _, err := rand.Read(secret); err != nil {
				Error.Printf("Failed to generate JWT secret: %v", err)
				// 使用默认密钥作为后备方案
				a.jwtSecret = []byte("default-jwt-secret-for-go-zlib-auth-provider")
			} else {
				a.jwtSecret = secret
			}
		}
	})
	return a.jwtSecret
}

// getCachePrefix 从配置文件获取缓存键前缀
func (a *authProvider) getCachePrefix() string {
	prefix, err := Config.String("config.auth.cache_auth_prefix")
	if err != nil || StringIsEmpty(prefix) {
		return "AUTH_TOKEN_" // 默认前缀
	}
	return prefix
}

// getProjectName 从配置文件获取缓项目名称
func (a *authProvider) getProjectName() string {
	name, err := Config.String("config.name")
	if err != nil || StringIsEmpty(name) {
		return "__MY_APP__"
	}
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// Login 用户登录，生成JWT token并存储到Redis，支持多设备登录管理
// userID: 用户ID
// deviceCode: 设备代码，用于区分不同设备
// expiration: token过期时间
// userData: 用户数据，可选参数，传入时会自动存储用户信息
func (a *authProvider) Login(userID string, deviceCode string, expiration time.Duration, userData ...interface{}) (string, error) {
	// 验证参数
	if StringIsEmpty(userID) {
		return "", fmt.Errorf("user ID cannot be empty")
	}
	if StringIsEmpty(deviceCode) {
		return "", fmt.Errorf("device code cannot be empty")
	}

	// 清除该设备之前的登录会话（同一设备只能登录一次）
	if err := a.LogoutDevice(userID, deviceCode); err != nil {
		// 记录错误但不阻止登录流程
		Error.Printf("Failed to clear previous device session: %v", err)
	}

	// 创建JWT claims
	claims := AuthClaims{
		UserID:     userID,
		DeviceCode: deviceCode,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    a.getProjectName(),
			Subject:   userID,
		},
	}

	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.getJWTSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// 存储到Redis - 使用token作为主键
	tokenCacheKey := fmt.Sprintf("%s_%s_%s", a.getProjectName(), a.getCachePrefix(), tokenString)
	sessionData := map[string]interface{}{
		"user_id":     userID,
		"device_code": deviceCode,
		"login_time":  time.Now().Unix(),
		"expires_at":  time.Now().Add(expiration).Unix(),
	}

	if err := RedisCache.Set(tokenCacheKey, sessionData, expiration); err != nil {
		return "", fmt.Errorf("failed to store session in redis: %w", err)
	}

	// 存储设备到用户的映射关系 - 用于设备管理
	deviceCacheKey := fmt.Sprintf("%s_%s_DEVICE_%s_%s", a.getProjectName(), a.getCachePrefix(), userID, deviceCode)
	deviceData := map[string]interface{}{
		"token":      tokenString,
		"login_time": time.Now().Unix(),
		"expires_at": time.Now().Add(expiration).Unix(),
	}

	if err := RedisCache.Set(deviceCacheKey, deviceData, expiration); err != nil {
		// 设备映射存储失败，清理已存储的token会话
		RedisCache.Delete(tokenCacheKey)
		return "", fmt.Errorf("failed to store device mapping in redis: %w", err)
	}

	// 如果传入了用户数据，则存储用户信息
	if len(userData) > 0 && userData[0] != nil {
		userCacheKey := fmt.Sprintf("%s_%s_USER_%s", a.getProjectName(), a.getCachePrefix(), userID)
		if err := RedisCache.Set(userCacheKey, userData[0], expiration); err != nil {
			// 用户信息存储失败，记录错误但不影响登录流程
			Error.Printf("Failed to store user info during login: %v", err)
		}
	}

	return tokenString, nil
}

// Logout 用户登出，支持多种登出模式
// deviceCode 参数：
// - 不传参数：登出当前token对应的设备
// - 传入具体设备代码：登出指定设备
// - 传入 "*"：登出所有设备
func (a *authProvider) Logout(c *gin.Context, deviceCode ...string) error {
	// 获取用户ID
	userID, isValid := a.Verify(c)
	if !isValid {
		return fmt.Errorf("invalid token")
	}

	// 根据参数决定登出模式
	if len(deviceCode) > 0 && !StringIsEmpty(deviceCode[0]) {
		if deviceCode[0] == "*" {
			// 登出所有设备时清除用户信息
			a.clearUserInfo(userID)
			return a.LogoutAllDevices(userID)
		} else {
			// 登出指定设备
			return a.LogoutDevice(userID, deviceCode[0])
		}
	}

	// 登出当前设备时清除用户信息
	a.clearUserInfo(userID)

	// 登出当前设备
	actualToken, err := a.getTokenFromContext(c)
	if err != nil {
		return fmt.Errorf("failed to get token from context: %w", err)
	}

	// 构造缓存键
	tokenCacheKey := fmt.Sprintf("%s_%s_%s", a.getProjectName(), a.getCachePrefix(), actualToken)

	// 获取会话数据以获取设备信息
	var sessionData interface{}
	if err := RedisCache.Get(tokenCacheKey, &sessionData); err != nil {
		return fmt.Errorf("session not found or expired: %w", err)
	}

	// 解析会话数据
	if sessionMap, ok := sessionData.(map[string]interface{}); ok {
		if deviceCodeInterface, exists := sessionMap["device_code"]; exists {
			if currentDeviceCode, ok := deviceCodeInterface.(string); ok {
				// 删除设备映射
				deviceCacheKey := fmt.Sprintf("%s_%s_DEVICE_%s_%s", a.getProjectName(), a.getCachePrefix(), userID, currentDeviceCode)
				RedisCache.Delete(deviceCacheKey)
			}
		}
	}

	// 删除token会话
	return RedisCache.Delete(tokenCacheKey)
}

// Refresh 刷新token过期时间
func (a *authProvider) Refresh(c *gin.Context, newExpiration time.Duration) error {
	// 从 gin 上下文获取 token
	actualToken, err := a.getTokenFromContext(c)
	if err != nil {
		return fmt.Errorf("failed to get token from context: %w", err)
	}

	// 验证token格式
	if StringIsEmpty(actualToken) {
		return fmt.Errorf("token cannot be empty")
	}

	// 构造缓存键
	cacheKey := fmt.Sprintf("%s_%s_%s", a.getProjectName(), a.getCachePrefix(), actualToken)

	// 检查会话是否存在
	var sessionData interface{}
	if err := RedisCache.Get(cacheKey, &sessionData); err != nil {
		return fmt.Errorf("session not found or expired: %w", err)
	}

	// 更新过期时间
	if err := RedisCache.Expire(cacheKey, newExpiration); err != nil {
		return fmt.Errorf("failed to refresh session expiration: %w", err)
	}

	return nil
}

// Verify 验证token是否有效并返回用户ID
func (a *authProvider) Verify(c *gin.Context) (string, bool) {
	userID, err := a.GetUserID(c)
	if err != nil {
		return "", false
	}
	return userID, true
}

// parseJWTToken 解析JWT token并返回声明信息
func (a *authProvider) parseJWTToken(actualToken string) (*AuthClaims, error) {
	if StringIsEmpty(actualToken) {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// 解析JWT token
	jwtToken, err := jwt.ParseWithClaims(actualToken, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.getJWTSecret(), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if claims, ok := jwtToken.Claims.(*AuthClaims); ok && jwtToken.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GetUserID 从token中获取用户ID并验证会话状态
// 当用户未登录、会话过期或会话不存在时，返回错误
func (a *authProvider) GetUserID(c *gin.Context) (string, error) {
	// 从 gin 上下文获取 token
	actualToken, err := a.getTokenFromContext(c)
	if err != nil {
		return "", fmt.Errorf("user not logged in: %w", err)
	}

	// 构造缓存键
	cacheKey := fmt.Sprintf("%s_%s_%s", a.getProjectName(), a.getCachePrefix(), actualToken)

	// 检查Redis中是否存在会话
	var sessionData interface{}
	if err := RedisCache.Get(cacheKey, &sessionData); err != nil {
		return "", fmt.Errorf("user session expired or not found: %w", err)
	}

	// 解析JWT token以获取用户信息
	claims, err := a.parseJWTToken(actualToken)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return claims.UserID, nil
}

// GetDeviceCode 从token中获取设备代码
func (a *authProvider) GetDeviceCode(c *gin.Context) (string, error) {
	actualToken, err := a.getTokenFromContext(c)
	if err != nil {
		return "", fmt.Errorf("failed to get token from context: %w", err)
	}

	claims, err := a.parseJWTToken(actualToken)
	if err != nil {
		return "", err
	}
	return claims.DeviceCode, nil
}

// clearUserInfo 清除用户信息（内部方法）
func (a *authProvider) clearUserInfo(userID string) {
	if StringIsEmpty(userID) {
		return
	}

	userCacheKey := fmt.Sprintf("%s_%s_USER_%s", a.getProjectName(), a.getCachePrefix(), userID)
	RedisCache.Delete(userCacheKey)
}

// GetUser 获取用户信息，支持泛型用户模型
func GetUser[T any](c *gin.Context) (*T, error) {
	// 获取用户ID以确保用户已登录
	userID, err := AuthProvider.GetUserID(c)
	if err != nil {
		return nil, fmt.Errorf("user not logged in: %w", err)
	}

	// 构造用户信息缓存键
	userCacheKey := fmt.Sprintf("%s_%s_USER_%s", AuthProvider.getProjectName(), AuthProvider.getCachePrefix(), userID)

	// 从Redis获取用户信息
	var userData interface{}
	if err := RedisCache.Get(userCacheKey, &userData); err != nil {
		return nil, fmt.Errorf("user info not found: %w", err)
	}

	// 类型转换
	var user T
	z.ToStruct(userData, &user)

	return &user, nil
}

// LogoutDevice 登出指定用户的指定设备
func (a *authProvider) LogoutDevice(userID, deviceCode string) error {
	if StringIsEmpty(userID) {
		return fmt.Errorf("user ID cannot be empty")
	}
	if StringIsEmpty(deviceCode) {
		return fmt.Errorf("device code cannot be empty")
	}

	// 构造设备缓存键
	deviceCacheKey := fmt.Sprintf("%s_%s_DEVICE_%s_%s", a.getProjectName(), a.getCachePrefix(), userID, deviceCode)

	// 获取设备信息
	var deviceData interface{}
	if err := RedisCache.Get(deviceCacheKey, &deviceData); err != nil {
		// 设备不存在或已过期，不需要处理
		return nil
	}

	// 解析设备数据
	if deviceMap, ok := deviceData.(map[string]interface{}); ok {
		if tokenInterface, exists := deviceMap["token"]; exists {
			if token, ok := tokenInterface.(string); ok {
				// 删除token会话
				tokenCacheKey := fmt.Sprintf("%s_%s_%s", a.getProjectName(), a.getCachePrefix(), token)
				RedisCache.Delete(tokenCacheKey)
			}
		}
	}

	// 删除设备映射
	return RedisCache.Delete(deviceCacheKey)
}

// LogoutAllDevices 登出用户的所有设备
func (a *authProvider) LogoutAllDevices(userID string) error {
	if StringIsEmpty(userID) {
		return fmt.Errorf("user ID cannot be empty")
	}

	// 清除用户信息
	a.clearUserInfo(userID)

	// 构造设备键模式
	deviceKeyPattern := fmt.Sprintf("%s_%s_DEVICE_%s_*", a.getProjectName(), a.getCachePrefix(), userID)

	// 获取所有匹配的设备键
	deviceKeys, err := RedisCache.Keys(deviceKeyPattern)
	if err != nil {
		return fmt.Errorf("failed to get device keys: %w", err)
	}

	// 逐个删除设备会话
	for _, deviceKey := range deviceKeys {
		// 获取设备信息
		var deviceData interface{}
		if err := RedisCache.Get(deviceKey, &deviceData); err != nil {
			continue // 跳过已过期或不存在的设备
		}

		// 解析设备数据并删除对应的token会话
		if deviceMap, ok := deviceData.(map[string]interface{}); ok {
			if tokenInterface, exists := deviceMap["token"]; exists {
				if token, ok := tokenInterface.(string); ok {
					tokenCacheKey := fmt.Sprintf("%s_%s_%s", a.getProjectName(), a.getCachePrefix(), token)
					RedisCache.Delete(tokenCacheKey)
				}
			}
		}

		// 删除设备映射
		RedisCache.Delete(deviceKey)
	}

	return nil
}

// GetUserDevices 获取用户的所有在线设备
func (a *authProvider) GetUserDevices(userID string) ([]map[string]interface{}, error) {
	if StringIsEmpty(userID) {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// 构造设备键模式
	deviceKeyPattern := fmt.Sprintf("%s_%s_DEVICE_%s_*", a.getProjectName(), a.getCachePrefix(), userID)

	// 获取所有匹配的设备键
	deviceKeys, err := RedisCache.Keys(deviceKeyPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get device keys: %w", err)
	}

	var devices []map[string]interface{}

	// 遍历设备键，获取设备信息
	for _, deviceKey := range deviceKeys {
		var deviceData interface{}
		if err := RedisCache.Get(deviceKey, &deviceData); err != nil {
			continue // 跳过已过期或不存在的设备
		}

		if deviceMap, ok := deviceData.(map[string]interface{}); ok {
			// 从设备键中提取设备代码
			keyParts := strings.Split(deviceKey, "_")
			if len(keyParts) > 0 {
				deviceCode := keyParts[len(keyParts)-1]
				deviceMap["device_code"] = deviceCode
			}
			devices = append(devices, deviceMap)
		}
	}

	return devices, nil
}
