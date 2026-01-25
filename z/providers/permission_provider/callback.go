package permission_provider

import "context"

// Callback 权限数据回调
// 由业务层在启动时注册，用于 Redis 缓存 miss 时回源获取数据并回填缓存。
type Callback interface {
	// GetUserPermissions 获取用户权限列表
	GetUserPermissions(ctx context.Context, tenantType, tenantID, userID string) ([]Permission, error)
}
