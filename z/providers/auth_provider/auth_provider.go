package auth_provider

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/mem_cache_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/redis_provider"
	"go.uber.org/fx"
)

const (
	defaultSessionDuration      = 24 * time.Hour
	defaultSessionTouchInterval = 5 * time.Minute
)

// Auth 鉴权 provider（fx 注入）
type Auth struct {
	cfg      *config_provider.Config
	log      *logger_provider.Logger
	redis    *redis_provider.Redis
	memCache *mem_cache_provider.MemCache

	guards map[string]*GuardConfig
	sorted []sortedGuard
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

// Init 初始化 guards
func (a *Auth) Init(cfg *config_provider.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	a.guards = make(map[string]*GuardConfig)
	a.sorted = nil

	guardMap := cfg.GetStringMap("auth.guards")
	var guardList []string
	if len(guardMap) > 0 {
		guardList = make([]string, 0, len(guardMap))
		for k := range guardMap {
			guardList = append(guardList, k)
		}
	}
	if len(guardList) == 0 {
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
		gc := &GuardConfig{
			Type:                 cfg.GetString("auth.guards." + g + ".type"),
			Token:                cfg.GetString("auth.guards." + g + ".token"),
			Prefix:               cfg.GetString("auth.guards." + g + ".prefix"),
			Cache:                cfg.GetString("auth.guards." + g + ".cache"),
			Duration:             cfg.GetInt("auth.guards." + g + ".duration"),
			TouchInterval:        cfg.GetInt("auth.guards." + g + ".touch_interval"),
			SingleSessionEnabled: cfg.GetBool("auth.guards." + g + ".single_session_enabled"),
			Anonymity:            cfg.GetStringSlice("auth.guards." + g + ".anonymity"),
		}
		a.guards[g] = gc
		if gc.Prefix != "" {
			a.sorted = append(a.sorted, sortedGuard{name: g, prefix: gc.Prefix})
		}
	}

	sort.Slice(a.sorted, func(i, j int) bool {
		return len(a.sorted[i].prefix) > len(a.sorted[j].prefix)
	})

	return nil
}

// extractToken 从token字符串中提取实际token，自动处理 Bearer 前缀
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

func (a *Auth) getSessionCacheKey(guardName, tokenHash string) string {
	return fmt.Sprintf("auth_session_%s_%s", guardName, tokenHash)
}

func (a *Auth) getUserSessionsKey(guardName, userID string) string {
	return fmt.Sprintf("auth_sessions_%s_%s", guardName, userID)
}

// getTokenHash 生成 token 哈希值
func (a *Auth) getTokenHash(token string) string {
	hash := md5.Sum([]byte(token))
	return fmt.Sprintf("%x", hash)
}

func (a *Auth) generateSessionToken() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(secret), nil
}

func (a *Auth) getGuardDuration(guardName string) time.Duration {
	guard, ok := a.guards[guardName]
	if !ok || guard == nil || guard.Duration <= 0 {
		return defaultSessionDuration
	}
	return time.Duration(guard.Duration) * time.Second
}

func (a *Auth) getGuardTouchInterval(guardName string) time.Duration {
	guard, ok := a.guards[guardName]
	if !ok || guard == nil || guard.TouchInterval <= 0 {
		return defaultSessionTouchInterval
	}
	return time.Duration(guard.TouchInterval) * time.Second
}

// GetGuardTouchInterval 返回 guard 的最小续期间隔。
func (a *Auth) GetGuardTouchInterval(guardName string) time.Duration {
	return a.getGuardTouchInterval(guardName)
}

func (a *Auth) getSession(guardName, tokenHash string) (*SessionData, bool, error) {
	key := a.getSessionCacheKey(guardName, tokenHash)
	if a.isRedisCache(guardName) {
		if a.redis == nil {
			return nil, false, fmt.Errorf("redis not enabled")
		}
		var session SessionData
		if err := a.redis.Get(key, &session); err != nil {
			return nil, false, nil
		}
		return &session, true, nil
	}
	if a.memCache == nil {
		return nil, false, fmt.Errorf("mem cache not enabled")
	}
	value, exists := a.memCache.Get(key)
	if !exists {
		return nil, false, nil
	}
	switch session := value.(type) {
	case *SessionData:
		return session, true, nil
	case SessionData:
		copy := session
		return &copy, true, nil
	default:
		return nil, false, fmt.Errorf("invalid session data")
	}
}

func (a *Auth) setSession(guardName string, session *SessionData, expiration time.Duration) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	return a.setCache(guardName, a.getSessionCacheKey(guardName, session.TokenHash), session, expiration)
}

func (a *Auth) deleteSession(guardName, tokenHash string) error {
	return a.deleteCache(guardName, a.getSessionCacheKey(guardName, tokenHash))
}

func (a *Auth) getUserSessionHashes(guardName, userID string) ([]string, error) {
	key := a.getUserSessionsKey(guardName, userID)
	if a.isRedisCache(guardName) {
		if a.redis == nil {
			return nil, fmt.Errorf("redis not enabled")
		}
		var hashes []string
		if err := a.redis.Get(key, &hashes); err != nil {
			return []string{}, nil
		}
		return hashes, nil
	}
	if a.memCache == nil {
		return nil, fmt.Errorf("mem cache not enabled")
	}
	value, exists := a.memCache.Get(key)
	if !exists {
		return []string{}, nil
	}
	switch hashes := value.(type) {
	case []string:
		return hashes, nil
	case []interface{}:
		result := make([]string, 0, len(hashes))
		for _, item := range hashes {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, nil
	default:
		return []string{}, nil
	}
}

func (a *Auth) setUserSessionHashes(guardName, userID string, hashes []string) error {
	key := a.getUserSessionsKey(guardName, userID)
	if len(hashes) == 0 {
		return a.deleteCache(guardName, key)
	}
	return a.setCache(guardName, key, hashes, a.getGuardDuration(guardName))
}

// touchUserSessionIndex 仅刷新用户会话索引的 TTL，避免续期时重复读写索引内容。
func (a *Auth) touchUserSessionIndex(guardName, userID string, duration time.Duration) error {
	key := a.getUserSessionsKey(guardName, userID)
	if a.isRedisCache(guardName) {
		if a.redis == nil {
			return fmt.Errorf("redis not enabled")
		}
		return a.redis.Expire(key, duration)
	}
	if a.memCache == nil {
		return fmt.Errorf("mem cache not enabled")
	}
	value, exists := a.memCache.Get(key)
	if !exists {
		return nil
	}
	a.memCache.Set(key, value, duration)
	return nil
}

func (a *Auth) addUserSessionHash(guardName, userID, tokenHash string) error {
	hashes, err := a.getUserSessionHashes(guardName, userID)
	if err != nil {
		return err
	}
	for _, hash := range hashes {
		if hash == tokenHash {
			return a.setUserSessionHashes(guardName, userID, hashes)
		}
	}
	hashes = append(hashes, tokenHash)
	return a.setUserSessionHashes(guardName, userID, hashes)
}

func (a *Auth) removeUserSessionHash(guardName, userID, tokenHash string) error {
	hashes, err := a.getUserSessionHashes(guardName, userID)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		if hash != tokenHash {
			filtered = append(filtered, hash)
		}
	}
	return a.setUserSessionHashes(guardName, userID, filtered)
}

func (a *Auth) clearUserAllSessions(guardName, userID string) error {
	hashes, err := a.getUserSessionHashes(guardName, userID)
	if err != nil {
		return err
	}
	for _, hash := range hashes {
		_ = a.deleteSession(guardName, hash)
	}
	return a.setUserSessionHashes(guardName, userID, nil)
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

	var authCtx *AuthContext
	var err error
	switch guardCfg.Type {
	case AuthTypeToken:
		authCtx, err = a.authenticateFixedToken(guardName, token, guardCfg)
	case AuthTypeSession:
		authCtx, err = a.authenticateSession(guardName, token)
	default:
		err = ErrAuthTypeUnsupported
	}
	if err != nil {
		return true, guardName, nil, convertToFriendlyError(err)
	}

	if guardCfg.Type == AuthTypeSession && authCtx != nil && authCtx.Session != nil {
		if err := a.touchSessionIfNeeded(guardName, authCtx.Session); err != nil {
			return true, guardName, nil, convertToFriendlyError(err)
		}
		authCtx.Data = authCtx.Session.Data
	}

	return true, guardName, authCtx, nil
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

func (a *Auth) authenticateFixedToken(guardName, token string, guardConfig *GuardConfig) (*AuthContext, error) {
	if token != guardConfig.Token {
		return nil, ErrTokenInvalid
	}

	tokenHash := a.getTokenHash(token)
	cacheKey := fmt.Sprintf("token_%s_%s", guardName, tokenHash)
	sessionData, exists := a.getCache(guardName, cacheKey)
	if !exists {
		sessionData = map[string]interface{}{
			"user_id":    tokenHash,
			"guard_name": guardName,
			"login_time": time.Now().Unix(),
			"token_type": "fixed",
		}
		_ = a.setCache(guardName, cacheKey, sessionData, 24*365*time.Hour)
	}

	return &AuthContext{
		GuardName: guardName,
		UserID:    tokenHash,
		Token:     token,
		Data:      sessionData,
	}, nil
}

func (a *Auth) authenticateSession(guardName, token string) (*AuthContext, error) {
	tokenHash := a.getTokenHash(token)
	session, exists, err := a.getSession(guardName, tokenHash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSessionNotFound
	}
	if session == nil || session.UserID == "" {
		return nil, ErrSessionInvalid
	}
	if session.GuardName != "" && session.GuardName != guardName {
		return nil, ErrTokenInvalid
	}

	return &AuthContext{
		GuardName: guardName,
		UserID:    session.UserID,
		Token:     token,
		Session:   session,
		Data:      session.Data,
	}, nil
}

func (a *Auth) touchSessionIfNeeded(guardName string, session *SessionData) error {
	if session == nil {
		return ErrSessionInvalid
	}

	duration := a.getGuardDuration(guardName)
	touchInterval := a.getGuardTouchInterval(guardName)
	now := time.Now()
	lastSeenAt := time.Unix(session.LastSeenAt, 0)

	if session.LastSeenAt > 0 && now.Sub(lastSeenAt) < touchInterval {
		return nil
	}

	session.LastSeenAt = now.Unix()
	session.ExpiresAt = now.Add(duration).Unix()
	if err := a.setSession(guardName, session, duration); err != nil {
		return err
	}

	if err := a.touchUserSessionIndex(guardName, session.UserID, duration); err != nil {
		return err
	}

	return nil
}

// TouchSession 按 token 触发 session 活跃续期。
// 主要用于 WebSocket 这类没有经过 HTTP 中间件的长连接场景。
func (a *Auth) TouchSession(guardName, token string) (*SessionData, error) {
	if strings.TrimSpace(guardName) == "" {
		return nil, fmt.Errorf("guard name cannot be empty")
	}

	token = a.extractToken(token)
	if strings.TrimSpace(token) == "" {
		return nil, ErrTokenMissing
	}

	guardCfg, ok := a.guards[guardName]
	if !ok {
		return nil, ErrGuardNotFound
	}

	switch guardCfg.Type {
	case AuthTypeToken:
		return nil, nil
	case AuthTypeSession:
		authCtx, err := a.authenticateSession(guardName, token)
		if err != nil {
			return nil, convertToFriendlyError(err)
		}
		if authCtx == nil || authCtx.Session == nil {
			return nil, ErrSessionInvalid
		}
		if err := a.touchSessionIfNeeded(guardName, authCtx.Session); err != nil {
			return nil, convertToFriendlyError(err)
		}
		return authCtx.Session, nil
	default:
		return nil, ErrAuthTypeUnsupported
	}
}

// Login 用户登录，生成 session token 并存储到缓存
func (a *Auth) Login(guard string, userID string, duration time.Duration, data ...interface{}) (string, error) {
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
	if guardConfig.Type != AuthTypeSession {
		return "", fmt.Errorf("guard '%s' does not support session login", guard)
	}

	if duration <= 0 {
		duration = a.getGuardDuration(guard)
	}

	if guardConfig.SingleSessionEnabled {
		if err := a.clearUserAllSessions(guard, userID); err != nil {
			return "", fmt.Errorf("failed to clear existing sessions: %w", err)
		}
	}

	token, err := a.generateSessionToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	session := &SessionData{
		TokenHash:  a.getTokenHash(token),
		UserID:     userID,
		GuardName:  guard,
		LoginTime:  now.Unix(),
		LastSeenAt: now.Unix(),
		ExpiresAt:  now.Add(duration).Unix(),
	}
	if len(data) > 0 && data[0] != nil {
		session.Data = data[0]
	}

	if err := a.setSession(guard, session, duration); err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}
	if err := a.addUserSessionHash(guard, userID, session.TokenHash); err != nil {
		_ = a.deleteSession(guard, session.TokenHash)
		return "", fmt.Errorf("failed to index session: %w", err)
	}

	return token, nil
}

// Logout 登出当前会话
func (a *Auth) Logout(guard, token string) error {
	if strings.TrimSpace(guard) == "" {
		return fmt.Errorf("guard name cannot be empty")
	}
	token = a.extractToken(token)
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token cannot be empty")
	}

	tokenHash := a.getTokenHash(token)
	session, exists, err := a.getSession(guard, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if !exists || session == nil {
		return nil
	}

	if err := a.deleteSession(guard, tokenHash); err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	if err := a.removeUserSessionHash(guard, session.UserID, tokenHash); err != nil {
		return fmt.Errorf("failed to clear session index: %w", err)
	}

	return nil
}

// LogoutAll 登出用户的所有会话
func (a *Auth) LogoutAll(guard, userID string) error {
	if strings.TrimSpace(guard) == "" {
		return fmt.Errorf("guard name cannot be empty")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if err := a.clearUserAllSessions(guard, userID); err != nil {
		return fmt.Errorf("failed to clear all sessions: %w", err)
	}
	return nil
}

// Authenticate 供 HTTP 中间件使用的鉴权入口
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
		path = c.Request.URL.Path
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

		_, _, authCtx, err := a.AuthenticateByGuard(guardName, token, "")
		if err != nil {
			continue
		}
		if authCtx == nil {
			continue
		}

		c.Set("auth.guard", guardName)
		c.Set("auth.user_id", authCtx.UserID)
		c.Set("auth.token", authCtx.Token)
		if authCtx.Session != nil {
			c.Set("auth.session", authCtx.Session)
		}
		if authCtx.Data != nil {
			c.Set("auth.data", authCtx.Data)
		}

		return true, guardName, nil
	}

	return false, "", ErrPermissionDenied
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
	value, ok := userID.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("user not authenticated")
	}
	return value, nil
}

// GetData 从 gin 上下文中获取当前登录用户的自定义数据
func (a *Auth) GetData(c *gin.Context) (interface{}, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	data, _ := c.Get("auth.data")
	return data, nil
}

// GetToken 从 gin 上下文中获取当前会话令牌
func (a *Auth) GetToken(c *gin.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("context is nil")
	}
	token, exists := c.Get("auth.token")
	if !exists {
		return "", fmt.Errorf("token not found in context")
	}
	value, ok := token.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("token not found in context")
	}
	return value, nil
}

// GetSession 从 gin 上下文中获取当前会话数据
func (a *Auth) GetSession(c *gin.Context) (*SessionData, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	session, exists := c.Get("auth.session")
	if !exists {
		return nil, fmt.Errorf("session not found in context")
	}
	value, ok := session.(*SessionData)
	if !ok || value == nil {
		return nil, fmt.Errorf("invalid session data")
	}
	return value, nil
}
