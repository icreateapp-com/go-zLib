package auth_provider

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/mem_cache_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/redis_provider"
	"go.uber.org/fx"
)

// Auth 鉴权 provider（fx 注入）
type Auth struct {
	cfg      *config_provider.Config
	log      *logger_provider.Logger
	redis    *redis_provider.Redis
	memCache *mem_cache_provider.MemCache

	jwtSecret []byte
	guards    map[string]*GuardConfig
	sorted    []sortedGuard
}

// In Auth 的 fx 入参
type In struct {
	fx.In

	Cfg      *config_provider.Config
	Log      *logger_provider.Logger
	Redis    *redis_provider.Redis        `optional:"true"`
	MemCache *mem_cache_provider.MemCache `optional:"true"`
}

type sortedGuard struct {
	name   string
	prefix string
}

// NewAuthProvider 创建 Auth provider
func NewAuthProvider(lc fx.Lifecycle, in In) (*Auth, error) {
	a := &Auth{cfg: in.Cfg, log: in.Log, redis: in.Redis, memCache: in.MemCache}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return a.Init(in.Cfg)
		},
	})

	return a, nil
}

// AuthProviderModule fx module
var AuthProviderModule = fx.Options(
	fx.Provide(NewAuthProvider),
)

// Init 初始化 guards 与 jwt secret
func (a *Auth) Init(cfg *config_provider.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	a.guards = make(map[string]*GuardConfig)
	a.sorted = nil

	// 由于 Config 目前缺少 GetStringMapAny 之类的通用方法，这里按固定键路径读取。
	// guard 列表通过 auth.guards.<name>.* 访问
	// aiaop-server 的 config.yml 中 auth.guards 是 map，因此这里优先从 map key 枚举 guard。
	guardMap := cfg.GetStringMap("auth.guards")
	var guardList []string
	if len(guardMap) > 0 {
		guardList = make([]string, 0, len(guardMap))
		for k := range guardMap {
			guardList = append(guardList, k)
		}
	}
	if len(guardList) == 0 {
		// 兼容显式 list（若有的话）
		guardList = cfg.GetStringSlice("auth.guards_names")
	}
	if len(guardList) == 0 {
		guardList = cfg.GetStringSlice("auth.guards_list")
	}
	if len(guardList) == 0 {
		return fmt.Errorf("missing auth.guards")
	}
	sort.Strings(guardList)

	for _, guardName := range guardList {
		g := strings.TrimSpace(guardName)
		if g == "" {
			continue
		}
		gc := &GuardConfig{}
		gc.Type = cfg.GetString("auth.guards." + g + ".type")
		gc.Token = cfg.GetString("auth.guards." + g + ".token")
		gc.Prefix = cfg.GetString("auth.guards." + g + ".prefix")
		gc.Cache = cfg.GetString("auth.guards." + g + ".cache")
		gc.SingleDeviceEnabled = cfg.GetBool("auth.guards." + g + ".single_device_enabled")
		gc.Anonymity = cfg.GetStringSlice("auth.guards." + g + ".anonymity")
		a.guards[g] = gc
		if gc.Prefix != "" {
			a.sorted = append(a.sorted, sortedGuard{name: g, prefix: gc.Prefix})
		}
	}

	sort.Slice(a.sorted, func(i, j int) bool {
		// prefix 越长越优先
		return len(a.sorted[i].prefix) > len(a.sorted[j].prefix)
	})

	key := cfg.GetString("app.key")
	if key == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		a.jwtSecret = secret
	} else {
		a.jwtSecret = []byte(key)
	}

	return nil
}

// extractToken 从token字符串中提取实际的JWT token，自动处理"Bearer "前缀
func (a *Auth) extractToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}

func (a *Auth) isRedisCache(guardName string) bool {
	guard, ok := a.guards[guardName]
	if !ok {
		return a.redis != nil
	}
	if guard.Cache == "" || guard.Cache == CacheTypeRedis {
		return a.redis != nil
	}
	return false
}

func (a *Auth) setCache(guardName, key string, value interface{}, expiration time.Duration) error {
	if a.isRedisCache(guardName) {
		if a.redis == nil {
			return fmt.Errorf("redis not enabled")
		}
		return a.redis.Set(key, value, expiration)
	}
	if a.memCache == nil {
		return fmt.Errorf("mem cache not enabled")
	}
	a.memCache.Set(key, value, expiration)
	return nil
}

func (a *Auth) getCache(guardName, key string) (interface{}, bool) {
	if a.isRedisCache(guardName) {
		if a.redis == nil {
			return nil, false
		}
		var result interface{}
		err := a.redis.Get(key, &result)
		return result, err == nil
	}
	if a.memCache == nil {
		return nil, false
	}
	return a.memCache.Get(key)
}

func (a *Auth) deleteCache(guardName, key string) error {
	if a.isRedisCache(guardName) {
		if a.redis == nil {
			return fmt.Errorf("redis not enabled")
		}
		return a.redis.Delete(key)
	}
	if a.memCache == nil {
		return fmt.Errorf("mem cache not enabled")
	}
	a.memCache.Delete(key)
	return nil
}

// getCacheKey 生成缓存键
func (a *Auth) getCacheKey(guardName, userID, device string) string {
	return fmt.Sprintf("auth_%s_%s_%s", guardName, userID, device)
}

// getCacheKeyWithoutDevice 生成不带设备的缓存键
func (a *Auth) getCacheKeyWithoutDevice(guardName, userID string) string {
	return fmt.Sprintf("auth_%s_%s", guardName, userID)
}

// getUserDevicesKey 生成用户设备列表缓存键（用于SSO清理）
func (a *Auth) getUserDevicesKey(guardName, userID string) string {
	return fmt.Sprintf("auth_devices_%s_%s", guardName, userID)
}

// getTokenHash 生成token哈希值（用于固定token模式）
func (a *Auth) getTokenHash(token string) string {
	hash := md5.Sum([]byte(token))
	return fmt.Sprintf("%x", hash)
}

// AuthenticateRequest 根据 requestPath 选择 guard 并鉴权
func (a *Auth) AuthenticateRequest(requestPath string, tokenFromHeader string, tokenFromQuery string) (bool, string, *AuthContext, error) {
	guardName, guardCfg := a.matchGuard(requestPath)
	if guardName == "" || guardCfg == nil {
		return false, "", nil, nil
	}
	for _, p := range guardCfg.Anonymity {
		if strings.HasPrefix(requestPath, p) {
			return false, "", nil, nil
		}
	}

	token := a.extractToken(tokenFromHeader)
	if token == "" {
		token = strings.TrimSpace(tokenFromQuery)
	}
	if token == "" {
		return true, guardName, nil, ErrTokenMissing
	}

	ok, g, ctx, err := a.AuthenticateByGuard(guardName, tokenFromHeader, tokenFromQuery)
	return ok, g, ctx, err
}

// AuthenticateByGuard 按指定 guard 鉴权
func (a *Auth) AuthenticateByGuard(guardName string, tokenFromHeader string, tokenFromQuery string) (bool, string, *AuthContext, error) {
	guardCfg, ok := a.guards[guardName]
	if !ok {
		return true, guardName, nil, ErrGuardNotFound
	}

	token := a.extractToken(tokenFromHeader)
	if token == "" {
		token = strings.TrimSpace(tokenFromQuery)
	}
	if token == "" {
		return true, guardName, nil, ErrTokenMissing
	}

	var userID string
	var sessionData map[string]interface{}
	var err error
	switch guardCfg.Type {
	case AuthTypeToken:
		userID, sessionData, err = a.authenticateFixedToken(guardName, token, guardCfg)
	case AuthTypeJWT:
		userID, sessionData, err = a.authenticateJWT(guardName, token)
	default:
		err = ErrAuthTypeUnsupported
	}
	if err != nil {
		return true, guardName, nil, convertToFriendlyError(err)
	}

	device := "default"
	if sessionData != nil {
		if v, ok := sessionData["device"]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				device = strings.TrimSpace(s)
			}
		}
	}

	return true, guardName, &AuthContext{GuardName: guardName, UserID: userID, Device: device, Data: sessionData}, nil
}

func (a *Auth) matchGuard(path string) (string, *GuardConfig) {
	for _, sg := range a.sorted {
		if strings.HasPrefix(path, sg.prefix) {
			gc := a.guards[sg.name]
			return sg.name, gc
		}
	}
	return "", nil
}

// clearUserAllDevices 清除用户在指定guard下的所有设备会话（用于SSO）
func (a *Auth) clearUserAllDevices(guardName, userID string) error {
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
func (a *Auth) addUserDevice(guardName, userID, device string) error {
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
func (a *Auth) Login(guard string, userID string, duration time.Duration, data ...interface{}) (string, error) {
	// 验证参数
	if strings.TrimSpace(guard) == "" {
		return "", fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("user ID cannot be empty")
	}

	guardConfig, exists := a.guards[guard]
	if !exists {
		return "", fmt.Errorf("guard '%s' not found", guard)
	}

	// 如果启用了单设备登录，清除该用户在当前guard下的所有会话
	if guardConfig.SingleDeviceEnabled {
		cacheKey := a.getCacheKeyWithoutDevice(guard, userID)
		a.deleteCache(guard, cacheKey)
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
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// 准备缓存数据
	sessionData := map[string]interface{}{
		"user_id":    userID,
		"guard_name": guard,
		"login_time": time.Now().Unix(),
		"expires_at": time.Now().Add(duration).Unix(),
	}

	// 如果传入了自定义数据，添加到会话中
	if len(data) > 0 && data[0] != nil {
		sessionData["data"] = data[0]
	}

	// 存储到缓存（不使用 device）
	cacheKey := a.getCacheKeyWithoutDevice(guard, userID)
	if err := a.setCache(guard, cacheKey, sessionData, duration); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	return tokenString, nil
}

// Logout 登出
func (a *Auth) Logout(guard, userID string) error {
	if strings.TrimSpace(guard) == "" {
		return fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// 清除缓存中的登录信息
	cacheKey := a.getCacheKeyWithoutDevice(guard, userID)
	if err := a.deleteCache(guard, cacheKey); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// LogoutAll 登出用户的所有设备
func (a *Auth) LogoutAll(guard, userID string) error {
	if strings.TrimSpace(guard) == "" {
		return fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// 清除所有设备会话
	if err := a.clearUserAllDevices(guard, userID); err != nil {
		return fmt.Errorf("failed to clear all devices: %w", err)
	}

	return nil
}

// removeUserDevice 从用户设备列表中移除指定设备
func (a *Auth) removeUserDevice(guardName, userID, device string) error {
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

// GetUserID 从 gin 上下文中获取当前登录用户的ID
func (a *Auth) GetUserID(c *gin.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("context is nil")
	}
	userID, exists := c.Get("auth.user_id")
	if !exists {
		return "", fmt.Errorf("user not authenticated")
	}
	return userID.(string), nil
}

// GetData 从 gin 上下文中获取当前登录用户的自定义数据
func (a *Auth) GetData(c *gin.Context) (interface{}, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	data, _ := c.Get("auth.data")
	return data, nil
}

// GetCurrentDevice 从 gin 上下文中获取当前设备标识
func (a *Auth) GetCurrentDevice(c *gin.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("context is nil")
	}
	device, exists := c.Get("auth.device")
	if !exists {
		return "", fmt.Errorf("device not found in context")
	}
	return device.(string), nil
}

// parseJWTToken 解析JWT token并返回声明信息
func (a *Auth) parseJWTToken(tokenString string) (*MultiTenantClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrTokenInvalid
	}

	// 解析JWT token
	jwtToken, err := jwt.ParseWithClaims(tokenString, &MultiTenantClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenSignature
		}
		return a.jwtSecret, nil
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

// authenticateFixedToken 固定token认证
func (a *Auth) authenticateFixedToken(guardName, token string, guardConfig *GuardConfig) (string, map[string]interface{}, error) {
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
func (a *Auth) authenticateJWT(guardName, token string) (string, map[string]interface{}, error) {
	// 解析JWT token
	claims, err := a.parseJWTToken(token)
	if err != nil {
		return "", nil, err
	}

	// 验证guard名称匹配
	if claims.GuardName != guardName {
		return "", nil, ErrGuardMismatch
	}

	// 检查JWT是否已过期（关键！）
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return "", nil, ErrTokenExpired
	}

	// 直接检查JWT token对应的缓存（不使用 device）
	cacheKey := a.getCacheKeyWithoutDevice(guardName, claims.UserID)
	sessionData, exists := a.getCache(guardName, cacheKey)

	if !exists {
		return "", nil, ErrSessionNotFound
	}

	sessionMap, ok := sessionData.(map[string]interface{})
	if !ok {
		return "", nil, ErrSessionInvalid
	}

	return claims.UserID, sessionMap, nil
}

// Authenticate 供 HTTP 中间件使用的鉴权入口：
// - guard 从 gin.Context 的 "guard" 读取（支持逗号分隔多个 guard）
// - token 从 header Authorization 或 query token 获取
// - 成功后写入 gin.Context：auth.guard/auth.user_id/auth.device/auth.data
func (a *Auth) Authenticate(c *gin.Context) (bool, string, error) {
	if c == nil {
		return true, "", nil
	}

	guardRaw, _ := c.Get("guard")
	guards, _ := guardRaw.(string)
	guards = strings.TrimSpace(guards)
	if guards == "" {
		return true, "", nil
	}

	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	token := a.extractToken(authHeader)
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token == "" {
		return false, "", ErrTokenMissing
	}

	path := ""
	if c.Request != nil && c.Request.URL != nil {
		path = strings.TrimSpace(c.Request.URL.Path)
	}

	guardList := strings.Split(guards, ",")
	for _, g := range guardList {
		guardName := strings.TrimSpace(g)
		if guardName == "" {
			continue
		}
		guardCfg, ok := a.guards[guardName]
		if !ok {
			continue
		}
		for _, p := range guardCfg.Anonymity {
			if strings.HasPrefix(path, p) {
				return true, "", nil
			}
		}

		var userID string
		var sessionData map[string]interface{}
		var err error
		switch guardCfg.Type {
		case AuthTypeToken:
			userID, sessionData, err = a.authenticateFixedToken(guardName, token, guardCfg)
		case AuthTypeJWT:
			userID, sessionData, err = a.authenticateJWT(guardName, token)
		default:
			err = ErrAuthTypeUnsupported
		}
		if err != nil {
			continue
		}

		device := "default"
		if sessionData != nil {
			if v, ok := sessionData["device"]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					device = strings.TrimSpace(s)
				}
			}
		}

		// 仅写入当前请求上下文
		c.Set("auth.guard", guardName)
		c.Set("auth.user_id", userID)
		c.Set("auth.device", device)
		if sessionData != nil {
			if v, ok := sessionData["data"]; ok {
				c.Set("auth.data", v)
			}
		}

		return true, guardName, nil
	}

	return false, "", ErrPermissionDenied
}

// GetUserDevices 获取用户的所有设备列表
func (a *Auth) GetUserDevices(guard, userID string) ([]string, error) {
	if strings.TrimSpace(guard) == "" {
		return nil, fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
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
func (a *Auth) IsDeviceOnline(guard, userID, device string) (bool, error) {
	if strings.TrimSpace(guard) == "" {
		return false, fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
		return false, fmt.Errorf("user ID cannot be empty")
	}
	if strings.TrimSpace(device) == "" {
		return false, fmt.Errorf("device cannot be empty")
	}

	cacheKey := a.getCacheKey(guard, userID, device)
	_, exists := a.getCache(guard, cacheKey)

	return exists, nil
}

// GetDeviceInfo 获取设备的详细信息
func (a *Auth) GetDeviceInfo(guard, userID, device string) (map[string]interface{}, error) {
	if strings.TrimSpace(guard) == "" {
		return nil, fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if strings.TrimSpace(device) == "" {
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
