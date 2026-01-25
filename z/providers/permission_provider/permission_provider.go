package permission_provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/redis_provider"
	"go.uber.org/fx"
)

// Provider 权限 provider（fx 注入）
type Provider struct {
	cfg      *config_provider.Config
	log      *logger_provider.Logger
	redis    *redis_provider.Redis
	enforcer *casbin.Enforcer

	conf PermissionConfig

	mu       sync.RWMutex
	callback Callback
}

// In Provider 的 fx 入参
type In struct {
	fx.In

	Cfg   *config_provider.Config
	Log   *logger_provider.Logger
	Redis *redis_provider.Redis `optional:"true"`
}

// NewPermissionProvider 创建 Permission provider
func NewPermissionProvider(lc fx.Lifecycle, in In) (*Provider, error) {
	p := &Provider{cfg: in.Cfg, log: in.Log, redis: in.Redis}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return p.Init(in.Cfg)
		},
	})

	return p, nil
}

// PermissionProviderModule fx module
var PermissionProviderModule = fx.Options(
	fx.Provide(NewPermissionProvider),
)

// Init 初始化配置
func (p *Provider) Init(cfg *config_provider.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	ttlSeconds := cfg.GetInt("permission.ttl")
	if ttlSeconds <= 0 {
		// 默认 24h
		ttlSeconds = 24 * 60 * 60
	}
	p.conf = PermissionConfig{TTL: time.Duration(ttlSeconds) * time.Second}

	if p.redis == nil {
		return fmt.Errorf("redis not enabled")
	}

	// 创建 Casbin model
	modelText := `
[request_definition]
r = sub, obj, act, tenant

[policy_definition]
p = sub, obj, act, tenant

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.tenant) && r.obj == p.obj && r.act == p.act && r.tenant == r.tenant
`

	casbinModel, err := model.NewModelFromString(modelText)
	if err != nil {
		return fmt.Errorf("failed to create casbin model: %w", err)
	}

	// 创建基于 redis_provider 的 adapter
	adapter := NewRedisAdapter(p.redis)

	// 创建 enforcer
	enforcer, err := casbin.NewEnforcer(casbinModel, adapter)
	if err != nil {
		return fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	p.enforcer = enforcer

	p.log.Infow("provider[permission] enabled", "ttl_seconds", ttlSeconds)
	return nil
}

// RegisterCallback 注册权限回调
func (p *Provider) RegisterCallback(cb Callback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.callback = cb
	if p.log != nil {
		p.log.Debugw("provider[permission] callback registered")
	}
}

func (p *Provider) getCallback() Callback {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.callback
}

func (p *Provider) cacheKey(tenantType, userID string) string {
	return fmt.Sprintf("permission_%s_%s", tenantType, userID)
}

// GetUserPermissions 获取用户权限（优先 Redis，miss 时回源并回填）
func (p *Provider) GetUserPermissions(ctx context.Context, tenantType, tenantID, userID string) ([]Permission, error) {
	tenantType = strings.TrimSpace(tenantType)
	userID = strings.TrimSpace(userID)
	if tenantType == "" || userID == "" {
		return nil, fmt.Errorf("invalid tenantType/userID")
	}
	if p.redis == nil {
		return nil, fmt.Errorf("redis not enabled")
	}

	key := p.cacheKey(tenantType, userID)

	// 1) 尝试从 redis 取
	var cached []Permission
	if err := p.redis.Get(key, &cached); err == nil {
		return cached, nil
	}

	// 2) 回源
	cb := p.getCallback()
	if cb == nil {
		return nil, fmt.Errorf("permission callback not registered")
	}
	perms, err := cb.GetUserPermissions(ctx, tenantType, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// 3) 回填 redis
	if len(perms) > 0 {
		if err := p.redis.Set(key, perms, p.conf.TTL); err != nil {
			if p.log != nil {
				p.log.Warnw("permission cache set failed", "key", key, "error", err)
			}
		}
	}

	return perms, nil
}

// RefreshUserPermissions 刷新用户权限（供业务层主动调用）
func (p *Provider) RefreshUserPermissions(ctx context.Context, tenantType, tenantID, userID string, permissions []Permission) error {
	tenantType = strings.TrimSpace(tenantType)
	userID = strings.TrimSpace(userID)
	if tenantType == "" || userID == "" {
		return fmt.Errorf("invalid tenantType/userID")
	}
	if p.redis == nil {
		return fmt.Errorf("redis not enabled")
	}

	key := p.cacheKey(tenantType, userID)

	if len(permissions) == 0 {
		// 权限为空，删除缓存
		return p.redis.Delete(key)
	}

	return p.redis.Set(key, permissions, p.conf.TTL)
}

// ClearPermissions 清理权限缓存（供业务层主动调用）
func (p *Provider) ClearPermissions(ctx context.Context, tenantType, tenantID, userID string) error {
	tenantType = strings.TrimSpace(tenantType)
	userID = strings.TrimSpace(userID)
	if tenantType == "" || userID == "" {
		return fmt.Errorf("invalid tenantType/userID")
	}
	if p.redis == nil {
		return fmt.Errorf("redis not enabled")
	}

	key := p.cacheKey(tenantType, userID)
	return p.redis.Delete(key)
}

// PermissionMiddleware 权限中间件
// 权限字符串格式："resource1:action1,action2;resource2:action1,action2"
// 分号分隔多个资源，每个资源用冒号分隔资源和动作，动作用逗号分隔
func (p *Provider) PermissionMiddleware(permissions string) gin.HandlerFunc {
	// 解析权限白名单
	whitelist := p.parsePermissions(permissions)

	return func(c *gin.Context) {
		if p == nil {
			c.Next()
			return
		}
		if c == nil {
			return
		}

		// 跳过 OPTIONS
		if c.Request != nil && c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// 从 gin.Context 获取 auth 信息（由 auth_provider.AuthMiddleware 设置）
		// guard 兼容两种写法："auth.guard"（新）和 "guard"（旧）
		guardAny, _ := c.Get("auth.guard")
		if guardAny == nil {
			guardAny, _ = c.Get("guard")
		}
		userIDAny, _ := c.Get("auth.user_id")

		guard, _ := guardAny.(string)
		userID, _ := userIDAny.(string)

		guard = strings.TrimSpace(guard)
		userID = strings.TrimSpace(userID)

		if guard == "" || userID == "" {
			c.JSON(403, gin.H{
				"success": false,
				"message": "missing guard/user context",
				"code":    403,
			})
			c.Abort()
			return
		}

		// tenantType 直接由 guard 推导（与 auth_provider.Authenticate 写入的 auth.guard 对齐）
		tenantType := guard
		if tenantType == "" {
			c.JSON(403, gin.H{
				"success": false,
				"message": "missing tenant context",
				"code":    403,
			})
			c.Abort()
			return
		}

		// 获取用户权限
		perms, err := p.GetUserPermissions(c.Request.Context(), tenantType, "", userID)
		if err != nil {
			c.JSON(403, gin.H{
				"success": false,
				"message": err.Error(),
				"code":    403,
			})
			c.Abort()
			return
		}

		// 检查用户权限是否在白名单中
		allowed := p.checkPermissions(perms, whitelist)
		if !allowed {
			c.JSON(403, gin.H{
				"success": false,
				"message": "no permission to perform this operation",
				"code":    403,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Can 检查用户是否具有指定权限（PermissionMiddleware 别名）
func (p *Provider) Can(Permission string) gin.HandlerFunc {
	return p.PermissionMiddleware(Permission)
}

// parsePermissions 解析权限字符串
// 格式："resource1:action1,action2;resource2:action1,action2"
func (p *Provider) parsePermissions(permissions string) map[string][]string {
	whitelist := make(map[string][]string)
	if permissions == "" {
		return whitelist
	}

	groups := strings.Split(permissions, ";")
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}

		parts := strings.Split(group, ":")
		if len(parts) != 2 {
			continue
		}

		resource := strings.TrimSpace(parts[0])
		actionsStr := strings.TrimSpace(parts[1])
		if resource == "" || actionsStr == "" {
			continue
		}

		actions := strings.Split(actionsStr, ",")
		var actionList []string
		for _, action := range actions {
			action = strings.TrimSpace(action)
			if action != "" {
				actionList = append(actionList, action)
			}
		}

		if len(actionList) > 0 {
			whitelist[resource] = actionList
		}
	}

	return whitelist
}

// checkPermissions 检查用户权限是否在白名单中
func (p *Provider) checkPermissions(perms []Permission, whitelist map[string][]string) bool {
	if len(whitelist) == 0 {
		return true // 没有配置白名单，默认允许
	}

	// 构建用户权限的 map，便于快速查找
	userPerms := make(map[string]map[string]bool)
	for _, perm := range perms {
		if userPerms[perm.Resource] == nil {
			userPerms[perm.Resource] = make(map[string]bool)
		}
		userPerms[perm.Resource][perm.Action] = true
	}

	// 检查每个白名单资源
	for resource, actions := range whitelist {
		// 检查用户是否有该资源的任意一个动作
		for _, action := range actions {
			if userPerms[resource] != nil && userPerms[resource][action] {
				return true
			}
		}
	}

	return false
}

// CheckPermission 检查权限（直接方法，供业务层调用）
func (p *Provider) CheckPermission(ctx context.Context, tenantType, tenantID, userID, resource, action string) (bool, error) {
	tenantType = strings.TrimSpace(tenantType)
	userID = strings.TrimSpace(userID)
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)

	if tenantType == "" || userID == "" || resource == "" || action == "" {
		return false, fmt.Errorf("invalid parameters")
	}

	perms, err := p.GetUserPermissions(ctx, tenantType, tenantID, userID)
	if err != nil {
		return false, err
	}

	for _, perm := range perms {
		if perm.Resource == resource && perm.Action == action {
			return true, nil
		}
	}

	return false, nil
}

// AddRoleForUser 为用户添加角色（供业务层调用）
func (p *Provider) AddRoleForUser(tenantType, tenantID, userID, roleCode string) error {
	if p.enforcer == nil {
		return fmt.Errorf("enforcer not initialized")
	}

	tenant := p.buildTenantKey(tenantType, tenantID)
	subject := p.buildSubjectKey(tenantType, tenantID, userID)
	role := p.buildRoleKey(tenantType, tenantID, roleCode)

	_, err := p.enforcer.AddRoleForUser(subject, role, tenant)
	if err != nil {
		return fmt.Errorf("failed to add role for user: %w", err)
	}

	// 清理用户权限缓存
	ctx := context.Background()
	if err := p.ClearPermissions(ctx, tenantType, "", userID); err != nil {
		if p.log != nil {
			p.log.Warnw("failed to clear user permissions", "error", err)
		}
	}

	return nil
}

// DeleteRoleForUser 删除用户角色（供业务层调用）
func (p *Provider) DeleteRoleForUser(tenantType, tenantID, userID, roleCode string) error {
	if p.enforcer == nil {
		return fmt.Errorf("enforcer not initialized")
	}

	tenant := p.buildTenantKey(tenantType, tenantID)
	subject := p.buildSubjectKey(tenantType, tenantID, userID)
	role := p.buildRoleKey(tenantType, tenantID, roleCode)

	_, err := p.enforcer.DeleteRoleForUser(subject, role, tenant)
	if err != nil {
		return fmt.Errorf("failed to delete role for user: %w", err)
	}

	// 清理用户权限缓存
	ctx := context.Background()
	if err := p.ClearPermissions(ctx, tenantType, "", userID); err != nil {
		if p.log != nil {
			p.log.Warnw("failed to clear user permissions", "error", err)
		}
	}

	return nil
}

// AddPermissionForRole 为角色添加权限（供业务层调用）
func (p *Provider) AddPermissionForRole(tenantType, tenantID, roleCode, resource, action string) error {
	if p.enforcer == nil {
		return fmt.Errorf("enforcer not initialized")
	}

	tenant := p.buildTenantKey(tenantType, tenantID)
	role := p.buildRoleKey(tenantType, tenantID, roleCode)

	_, err := p.enforcer.AddPermissionForUser(role, resource, action, tenant)
	if err != nil {
		return fmt.Errorf("failed to add permission for role: %w", err)
	}

	// 清理该角色下所有用户的权限缓存
	// 这里简化处理，实际可能需要批量清理
	return nil
}

// DeletePermissionForRole 删除角色权限（供业务层调用）
func (p *Provider) DeletePermissionForRole(tenantType, tenantID, roleCode, resource, action string) error {
	if p.enforcer == nil {
		return fmt.Errorf("enforcer not initialized")
	}

	tenant := p.buildTenantKey(tenantType, tenantID)
	role := p.buildRoleKey(tenantType, tenantID, roleCode)

	_, err := p.enforcer.DeletePermissionForUser(role, resource, action, tenant)
	if err != nil {
		return fmt.Errorf("failed to delete permission for role: %w", err)
	}

	// 清理该角色下所有用户的权限缓存
	return nil
}

// GetImplicitPermissionsForUser 获取用户的所有权限（包括继承的）
func (p *Provider) GetImplicitPermissionsForUser(tenantType, tenantID, userID string) ([][]string, error) {
	if p.enforcer == nil {
		return nil, fmt.Errorf("enforcer not initialized")
	}

	tenant := p.buildTenantKey(tenantType, tenantID)
	subject := p.buildSubjectKey(tenantType, tenantID, userID)

	permissions, err := p.enforcer.GetImplicitPermissionsForUser(subject, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to get implicit permissions: %w", err)
	}

	return permissions, nil
}

// 辅助方法：构建 key
func (p *Provider) buildTenantKey(tenantType, tenantID string) string {
	return fmt.Sprintf("%s:%s", tenantType, tenantID)
}

func (p *Provider) buildSubjectKey(tenantType, tenantID, userID string) string {
	return fmt.Sprintf("%s:%s:%s", tenantType, tenantID, userID)
}

func (p *Provider) buildRoleKey(tenantType, tenantID, roleCode string) string {
	return fmt.Sprintf("%s:%s:%s", tenantType, tenantID, roleCode)
}
