package auth_provider

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/icreateapp-com/go-zLib/z"
)

// authProvider 认证提供者结构
type authProvider struct {
	guards     map[string]*GuardConfig // guard配置映射
	contexts   map[string]*AuthContext // 当前请求的认证上下文（按guard分组）
	ginContext *gin.Context            // 当前gin上下文
	jwtSecret  []byte                  // JWT密钥
	mutex      sync.RWMutex            // 读写锁
	once       sync.Once               // 确保JWT密钥只生成一次
}

// AuthProvider 全局认证提供者实例
var AuthProvider authProvider

// Init 初始化认证提供者，读取guards配置
func (a *authProvider) Init() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// 初始化guards映射
	a.guards = make(map[string]*GuardConfig)
	a.contexts = make(map[string]*AuthContext)

	// 从配置文件读取auth.guards配置
	authConfig := z.Config.GetStringMap("config.auth")
	if authConfig == nil {
		z.Error.Fatal("auth configuration not found")
	}

	guardsConfig, exists := authConfig["guards"]
	if !exists {
		z.Error.Fatal("guards configuration not found")
	}

	guardsMap, ok := guardsConfig.(map[string]interface{})
	if !ok {
		z.Error.Fatal("invalid guards configuration format")
	}

	// 解析每个guard配置
	for guardName, guardConfigInterface := range guardsMap {
		guardConfigMap, ok := guardConfigInterface.(map[string]interface{})
		if !ok {
			continue
		}

		guard := &GuardConfig{}

		// 解析配置字段
		if guardType, exists := guardConfigMap["type"]; exists {
			guard.Type = guardType.(string)
		}
		if token, exists := guardConfigMap["token"]; exists {
			guard.Token = token.(string)
		}
		if prefix, exists := guardConfigMap["prefix"]; exists {
			guard.Prefix = prefix.(string)
		}
		if cache, exists := guardConfigMap["cache"]; exists {
			guard.Cache = cache.(string)
		}
		if ssoEnabled, exists := guardConfigMap["sso_enabled"]; exists {
			guard.SSOEnabled = ssoEnabled.(bool)
		}

		// 解析匿名路由列表
		if anonymityInterface, exists := guardConfigMap["anonymity"]; exists {
			if anonymitySlice, ok := anonymityInterface.([]interface{}); ok {
				for _, item := range anonymitySlice {
					if path, ok := item.(string); ok {
						guard.Anonymity = append(guard.Anonymity, path)
					}
				}
			}
		}

		a.guards[guardName] = guard
	}
}

// extractToken 从token字符串中提取实际的JWT token，自动处理"Bearer "前缀
func (a *authProvider) extractToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}

// getTokenFromContext 从gin上下文中获取Authorization token
func (a *authProvider) getTokenFromContext(c *gin.Context) (string, error) {
	authHeader := c.Request.Header.Get("Authorization")
	if z.StringIsEmpty(authHeader) {
		return "", fmt.Errorf("authorization header is missing")
	}
	return a.extractToken(authHeader), nil
}

// getJWTSecret 获取JWT密钥，首次调用时从配置文件获取
func (a *authProvider) getJWTSecret() []byte {
	a.once.Do(func() {
		// 从配置文件获取密钥
		if configKey, err := z.Config.String("config.key"); err == nil && !z.StringIsEmpty(configKey) {
			a.jwtSecret = []byte(configKey)
		} else {
			// 配置文件中没有密钥，生成随机密钥
			secret := make([]byte, 32)
			if _, err := rand.Read(secret); err != nil {
				z.Error.Printf("Failed to generate JWT secret: %v", err)
				// 使用默认密钥作为后备方案
				a.jwtSecret = []byte("default-jwt-secret-for-go-zlib-auth-provider")
			} else {
				a.jwtSecret = secret
			}
		}
	})
	return a.jwtSecret
}

// isRedisCache 判断guard是否使用Redis缓存
func (a *authProvider) isRedisCache(guardName string) bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	guard, exists := a.guards[guardName]
	if !exists || guard.Cache == "redis" {
		return true
	}
	return false
}

// setCache 设置缓存
func (a *authProvider) setCache(guardName, key string, value interface{}, expiration time.Duration) error {
	if a.isRedisCache(guardName) {
		return z.RedisCache.Set(key, value, expiration)
	} else {
		z.MemCache.Set(key, value, expiration)
		return nil
	}
}

// getCache 获取缓存
func (a *authProvider) getCache(guardName, key string) (interface{}, bool) {
	if a.isRedisCache(guardName) {
		var result interface{}
		err := z.RedisCache.Get(key, &result)
		return result, err == nil
	} else {
		return z.MemCache.Get(key)
	}
}

// deleteCache 删除缓存
func (a *authProvider) deleteCache(guardName, key string) error {
	if a.isRedisCache(guardName) {
		return z.RedisCache.Delete(key)
	} else {
		z.MemCache.Delete(key)
		return nil
	}
}

// getCacheKey 生成缓存键
func (a *authProvider) getCacheKey(guardName, userID, device string) string {
	return fmt.Sprintf("auth_%s_%s_%s", guardName, userID, device)
}

// getUserDevicesKey 生成用户设备列表缓存键（用于SSO清理）
func (a *authProvider) getUserDevicesKey(guardName, userID string) string {
	return fmt.Sprintf("auth_devices_%s_%s", guardName, userID)
}

// getTokenHash 生成token哈希值（用于固定token模式）
func (a *authProvider) getTokenHash(token string) string {
	hash := md5.Sum([]byte(token))
	return fmt.Sprintf("%x", hash)
}

// setContext 设置当前请求的认证上下文
func (a *authProvider) setContext(c *gin.Context, guardName, userID, device string, data map[string]interface{}) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.ginContext = c
	a.contexts[guardName] = &AuthContext{
		GuardName: guardName,
		UserID:    userID,
		Device:    device,
		Data:      data,
	}
}

// clearUserAllDevices 清除用户在指定guard下的所有设备会话（用于SSO）
func (a *authProvider) clearUserAllDevices(guardName, userID string) error {
	devicesKey := a.getUserDevicesKey(guardName, userID)

	// 获取用户的设备列表
	if devices, exists := a.getCache(guardName, devicesKey); exists {
		if deviceList, ok := devices.([]interface{}); ok {
			// 清除每个设备的会话
			for _, deviceInterface := range deviceList {
				if device, ok := deviceInterface.(string); ok {
					cacheKey := a.getCacheKey(guardName, userID, device)
					a.deleteCache(guardName, cacheKey)
				}
			}
		}
	}

	// 清除设备列表
	a.deleteCache(guardName, devicesKey)
	return nil
}

// addUserDevice 将设备添加到用户的设备列表中
func (a *authProvider) addUserDevice(guardName, userID, device string) error {
	devicesKey := a.getUserDevicesKey(guardName, userID)

	var devices []string
	if existingDevices, exists := a.getCache(guardName, devicesKey); exists {
		if deviceList, ok := existingDevices.([]interface{}); ok {
			for _, deviceInterface := range deviceList {
				if d, ok := deviceInterface.(string); ok {
					devices = append(devices, d)
				}
			}
		}
	}

	// 检查设备是否已存在
	for _, d := range devices {
		if d == device {
			return nil // 设备已存在，无需添加
		}
	}

	// 添加新设备
	devices = append(devices, device)

	// 存储设备列表（设置较长的过期时间）
	return a.setCache(guardName, devicesKey, devices, 24*time.Hour)
}

// Login 用户登录，生成JWT token并存储到缓存
func (a *authProvider) Login(guard string, userID string, device string, duration time.Duration, data ...interface{}) (string, error) {
	// 验证参数
	if z.StringIsEmpty(guard) {
		return "", fmt.Errorf("guard name cannot be empty")
	}
	if z.StringIsEmpty(userID) {
		return "", fmt.Errorf("user ID cannot be empty")
	}
	if z.StringIsEmpty(device) {
		device = "default" // 如果设备为空，使用默认值
	}

	a.mutex.RLock()
	guardConfig, exists := a.guards[guard]
	a.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("guard '%s' not found", guard)
	}

	// 如果启用了SSO，清除该用户在当前guard下的所有其他设备会话
	if guardConfig.SSOEnabled {
		a.clearUserAllDevices(guard, userID)
	}

	// 创建JWT claims
	claims := MultiTenantClaims{
		UserID:    userID,
		GuardName: guard,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "go-zlib-auth",
			Subject:   userID,
		},
	}

	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.getJWTSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// 准备缓存数据
	sessionData := map[string]interface{}{
		"user_id":    userID,
		"guard_name": guard,
		"device":     device,
		"login_time": time.Now().Unix(),
		"expires_at": time.Now().Add(duration).Unix(),
	}

	// 如果传入了自定义数据，添加到会话中
	if len(data) > 0 && data[0] != nil {
		sessionData["data"] = data[0]
	}

	// 存储到缓存
	cacheKey := a.getCacheKey(guard, userID, device)
	if err := a.setCache(guard, cacheKey, sessionData, duration); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	// 将设备添加到用户设备列表（除非SSO启用，因为SSO时已清除所有设备）
	if !guardConfig.SSOEnabled {
		a.addUserDevice(guard, userID, device)
	} else {
		// SSO启用时，只添加当前设备
		a.addUserDevice(guard, userID, device)
	}

	return tokenString, nil
}

// Logout 登出指定设备，device为空时表示默认设备，userID为空时表示当前用户
func (a *authProvider) Logout(guard, device string, userID ...string) error {
	if z.StringIsEmpty(guard) {
		return fmt.Errorf("guard name cannot be empty")
	}

	var targetUserID string

	// 如果没有传递userID参数，则获取当前登录用户的ID
	if len(userID) == 0 || z.StringIsEmpty(userID[0]) {
		currentUserID, err := a.GetUserID(guard)
		if err != nil {
			return fmt.Errorf("failed to get current user ID: %w", err)
		}
		targetUserID = currentUserID

		// 如果device为空且是当前用户，获取当前设备
		if z.StringIsEmpty(device) {
			if context, exists := a.contexts[guard]; exists && context != nil {
				device = context.Device
			}
		}
	} else {
		targetUserID = userID[0]
	}

	if z.StringIsEmpty(targetUserID) {
		return fmt.Errorf("user ID cannot be empty")
	}
	if z.StringIsEmpty(device) {
		device = "default"
	}

	cacheKey := a.getCacheKey(guard, targetUserID, device)

	// 清除缓存中的登录信息
	if err := a.deleteCache(guard, cacheKey); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// 从设备列表中移除该设备
	a.removeUserDevice(guard, targetUserID, device)

	// 清除内存中的认证上下文（如果是当前设备）
	a.mutex.Lock()
	if context, exists := a.contexts[guard]; exists && context != nil &&
		context.UserID == targetUserID && context.Device == device {
		delete(a.contexts, guard)
	}
	a.mutex.Unlock()

	return nil
}

// LogoutAll 登出用户的所有设备，userID为空时表示当前用户
func (a *authProvider) LogoutAll(guard string, userID ...string) error {
	if z.StringIsEmpty(guard) {
		return fmt.Errorf("guard name cannot be empty")
	}

	var targetUserID string

	// 如果没有传递userID参数，则获取当前登录用户的ID
	if len(userID) == 0 || z.StringIsEmpty(userID[0]) {
		currentUserID, err := a.GetUserID(guard)
		if err != nil {
			return fmt.Errorf("failed to get current user ID: %w", err)
		}
		targetUserID = currentUserID
	} else {
		targetUserID = userID[0]
	}

	if z.StringIsEmpty(targetUserID) {
		return fmt.Errorf("user ID cannot be empty")
	}

	// 清除所有设备会话
	if err := a.clearUserAllDevices(guard, targetUserID); err != nil {
		return fmt.Errorf("failed to clear all devices: %w", err)
	}

	// 清除内存中的认证上下文
	a.mutex.Lock()
	delete(a.contexts, guard)
	a.mutex.Unlock()

	return nil
}

// removeUserDevice 从用户设备列表中移除指定设备
func (a *authProvider) removeUserDevice(guardName, userID, device string) error {
	devicesKey := a.getUserDevicesKey(guardName, userID)

	var devices []string
	if existingDevices, exists := a.getCache(guardName, devicesKey); exists {
		if deviceList, ok := existingDevices.([]interface{}); ok {
			for _, deviceInterface := range deviceList {
				if d, ok := deviceInterface.(string); ok && d != device {
					devices = append(devices, d)
				}
			}
		}
	}

	// 更新设备列表
	if len(devices) > 0 {
		return a.setCache(guardName, devicesKey, devices, 24*time.Hour)
	} else {
		// 如果没有设备了，删除设备列表
		return a.deleteCache(guardName, devicesKey)
	}
}

// GetUserID 获取当前登录用户的ID
func (a *authProvider) GetUserID(guard string) (string, error) {
	a.mutex.RLock()
	context, exists := a.contexts[guard]
	a.mutex.RUnlock()

	if !exists || context == nil {
		return "", fmt.Errorf("user not authenticated for guard '%s'", guard)
	}

	return context.UserID, nil
}

// GetData 获取当前登录用户的自定义数据
func (a *authProvider) GetData(guard string) (interface{}, error) {
	a.mutex.RLock()
	context, exists := a.contexts[guard]
	a.mutex.RUnlock()

	if !exists || context == nil {
		return nil, fmt.Errorf("user not authenticated for guard '%s'", guard)
	}

	if data, exists := context.Data["data"]; exists {
		return data, nil
	}

	return nil, nil
}

// parseJWTToken 解析JWT token并返回声明信息
func (a *authProvider) parseJWTToken(tokenString string) (*MultiTenantClaims, error) {
	if z.StringIsEmpty(tokenString) {
		return nil, ErrTokenInvalid
	}

	// 解析JWT token
	jwtToken, err := jwt.ParseWithClaims(tokenString, &MultiTenantClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenSignature
		}
		return a.getJWTSecret(), nil
	})

	if err != nil {
		// 让convertToFriendlyError处理具体的JWT错误
		return nil, err
	}

	if claims, ok := jwtToken.Claims.(*MultiTenantClaims); ok && jwtToken.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// GetUserID 泛型获取当前登录用户的ID
func GetUserID[T any](guard string) (T, error) {
	var zero T
	userIDStr, err := AuthProvider.GetUserID(guard)
	if err != nil {
		return zero, err
	}

	var result T
	if err := z.ToStruct(userIDStr, &result); err != nil {
		return zero, fmt.Errorf("failed to convert user ID: %w", err)
	}
	return result, nil
}

// GetData 泛型获取当前登录用户的自定义数据
func GetData[T any](guard string) (T, error) {
	var zero T
	data, err := AuthProvider.GetData(guard)
	if err != nil {
		return zero, err
	}

	if data == nil {
		return zero, nil
	}

	var result T
	if err := z.ToStruct(data, &result); err != nil {
		return zero, fmt.Errorf("failed to convert user data: %w", err)
	}
	return result, nil
}

// findMatchingGuard 查找匹配的guard
func (a *authProvider) findMatchingGuard(requestPath string) (string, *GuardConfig) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for guardName, guardConfig := range a.guards {
		// 检查路径前缀匹配
		if guardConfig.Prefix != "" && strings.HasPrefix(requestPath, guardConfig.Prefix) {
			return guardName, guardConfig
		}
	}
	return "", nil
}

// getTokenFromRequest 从请求中获取令牌，支持多种来源
func (a *authProvider) getTokenFromRequest(c *gin.Context) string {
	// 优先从 Authorization header 获取
	authHeader := c.Request.Header.Get("Authorization")
	if !z.StringIsEmpty(authHeader) {
		token := a.extractToken(authHeader)
		if !z.StringIsEmpty(token) {
			return token
		}
	}

	// 从 URL 参数 token 获取
	if tokenParam := c.Query("token"); !z.StringIsEmpty(tokenParam) {
		return tokenParam
	}

	return ""
}

// Authenticate 中间件专用认证方法
func (a *authProvider) Authenticate(c *gin.Context) (bool, string, error) {
	requestPath := c.Request.URL.Path

	// 获取匹配的guard
	guardName, guardConfig := a.findMatchingGuard(requestPath)
	if guardName == "" {
		return true, "", nil // 没有匹配的guard，放行
	}

	// 检查是否在匿名访问列表中
	for _, anonymousPath := range guardConfig.Anonymity {
		if strings.HasPrefix(requestPath, anonymousPath) {
			return true, "", nil // 匿名访问，放行
		}
	}

	// 从多种来源获取令牌
	token := a.getTokenFromRequest(c)
	if z.StringIsEmpty(token) {
		return false, "", ErrTokenMissing
	}

	// 根据guard类型进行认证
	var userID string
	var sessionData map[string]interface{}
	var err error

	switch guardConfig.Type {
	case "token":
		userID, sessionData, err = a.authenticateFixedToken(guardName, token, guardConfig)
	case "jwt":
		userID, sessionData, err = a.authenticateJWT(guardName, token)
	default:
		return false, "", ErrAuthTypeUnsupported
	}

	if err != nil {
		// 转换为友好的错误消息
		friendlyErr := convertToFriendlyError(err)
		return false, "", friendlyErr
	}

	// 从会话数据中获取设备信息
	device := "default"
	if sessionData != nil {
		if deviceValue, exists := sessionData["device"]; exists {
			if deviceStr, ok := deviceValue.(string); ok {
				device = deviceStr
			}
		}
	}

	// 设置认证上下文
	a.setContext(c, guardName, userID, device, sessionData)

	return true, guardName, nil
}

// authenticateFixedToken 固定token认证
func (a *authProvider) authenticateFixedToken(guardName, token string, guardConfig *GuardConfig) (string, map[string]interface{}, error) {
	// 比较固定token
	if token != guardConfig.Token {
		return "", nil, ErrTokenInvalid
	}

	// 生成token哈希作为用户ID
	tokenHash := a.getTokenHash(token)

	// 检查缓存中是否存在会话
	cacheKey := fmt.Sprintf("token_%s_%s", guardName, tokenHash)
	sessionData, exists := a.getCache(guardName, cacheKey)

	if !exists {
		// 创建新的会话数据
		sessionData = map[string]interface{}{
			"user_id":    tokenHash,
			"guard_name": guardName,
			"login_time": time.Now().Unix(),
			"token_type": "fixed",
		}

		// 存储到缓存（固定token永不过期，设置较长时间）
		a.setCache(guardName, cacheKey, sessionData, 24*365*time.Hour)
	}

	sessionMap, ok := sessionData.(map[string]interface{})
	if !ok {
		return "", nil, ErrSessionInvalid
	}

	return tokenHash, sessionMap, nil
}

// authenticateJWT JWT认证
func (a *authProvider) authenticateJWT(guardName, token string) (string, map[string]interface{}, error) {
	// 解析JWT token
	claims, err := a.parseJWTToken(token)
	if err != nil {
		return "", nil, err
	}

	// 验证guard名称匹配
	if claims.GuardName != guardName {
		return "", nil, ErrGuardMismatch
	}

	// JWT认证需要遍历用户的所有设备来查找有效会话
	// 首先获取用户的设备列表
	devicesKey := a.getUserDevicesKey(guardName, claims.UserID)
	devices, exists := a.getCache(guardName, devicesKey)

	if !exists {
		return "", nil, ErrSessionNotFound
	}

	// 遍历设备列表，查找有效的会话
	if deviceList, ok := devices.([]interface{}); ok {
		for _, deviceInterface := range deviceList {
			if device, ok := deviceInterface.(string); ok {
				cacheKey := a.getCacheKey(guardName, claims.UserID, device)
				sessionData, exists := a.getCache(guardName, cacheKey)

				if exists {
					sessionMap, ok := sessionData.(map[string]interface{})
					if ok {
						// 确保会话数据中包含设备信息
						sessionMap["device"] = device
						return claims.UserID, sessionMap, nil
					}
				}
			}
		}
	}

	return "", nil, ErrSessionNotFound
}

// GetCurrentDevice 获取当前设备标识
func (a *authProvider) GetCurrentDevice(guard string) (string, error) {
	a.mutex.RLock()
	context, exists := a.contexts[guard]
	a.mutex.RUnlock()

	if !exists || context == nil {
		return "", fmt.Errorf("user not authenticated for guard '%s'", guard)
	}

	return context.Device, nil
}

// GetUserDevices 获取用户的所有设备列表
func (a *authProvider) GetUserDevices(guard, userID string) ([]string, error) {
	if z.StringIsEmpty(guard) {
		return nil, fmt.Errorf("guard name cannot be empty")
	}
	if z.StringIsEmpty(userID) {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	devicesKey := a.getUserDevicesKey(guard, userID)
	devices, exists := a.getCache(guard, devicesKey)

	if !exists {
		return []string{}, nil // 返回空列表而不是错误
	}

	var deviceList []string
	if deviceSlice, ok := devices.([]interface{}); ok {
		for _, deviceInterface := range deviceSlice {
			if device, ok := deviceInterface.(string); ok {
				deviceList = append(deviceList, device)
			}
		}
	}

	return deviceList, nil
}

// IsDeviceOnline 检查指定设备是否在线（有有效会话）
func (a *authProvider) IsDeviceOnline(guard, userID, device string) (bool, error) {
	if z.StringIsEmpty(guard) {
		return false, fmt.Errorf("guard name cannot be empty")
	}
	if z.StringIsEmpty(userID) {
		return false, fmt.Errorf("user ID cannot be empty")
	}
	if z.StringIsEmpty(device) {
		return false, fmt.Errorf("device cannot be empty")
	}

	cacheKey := a.getCacheKey(guard, userID, device)
	_, exists := a.getCache(guard, cacheKey)

	return exists, nil
}

// GetDeviceInfo 获取设备的详细信息
func (a *authProvider) GetDeviceInfo(guard, userID, device string) (map[string]interface{}, error) {
	if z.StringIsEmpty(guard) {
		return nil, fmt.Errorf("guard name cannot be empty")
	}
	if z.StringIsEmpty(userID) {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if z.StringIsEmpty(device) {
		return nil, fmt.Errorf("device cannot be empty")
	}

	cacheKey := a.getCacheKey(guard, userID, device)
	sessionData, exists := a.getCache(guard, cacheKey)

	if !exists {
		return nil, fmt.Errorf("device session not found")
	}

	sessionMap, ok := sessionData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid session data")
	}

	return sessionMap, nil
}

// 全局函数，方便外部调用

// GetCurrentDevice 获取当前设备标识
func GetCurrentDevice(guard string) (string, error) {
	return AuthProvider.GetCurrentDevice(guard)
}

// GetUserDevices 获取用户的所有设备列表
func GetUserDevices(guard, userID string) ([]string, error) {
	return AuthProvider.GetUserDevices(guard, userID)
}

// Logout 全局登出函数
func Logout(guard, device string, userID ...string) error {
	return AuthProvider.Logout(guard, device, userID...)
}

// LogoutAll 登出用户的所有设备
func LogoutAll(guard string, userID ...string) error {
	return AuthProvider.LogoutAll(guard, userID...)
}

// IsDeviceOnline 检查指定设备是否在线
func IsDeviceOnline(guard, userID, device string) (bool, error) {
	return AuthProvider.IsDeviceOnline(guard, userID, device)
}

// GetDeviceInfo 获取设备的详细信息
func GetDeviceInfo(guard, userID, device string) (map[string]interface{}, error) {
	return AuthProvider.GetDeviceInfo(guard, userID, device)
}

// Login 全局登录函数
func Login(guard, userID, device string, duration time.Duration, data ...interface{}) (string, error) {
	return AuthProvider.Login(guard, userID, device, duration, data...)
}
