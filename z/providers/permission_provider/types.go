package permission_provider

import "time"

// Permission 权限结构
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Role 角色结构
type Role struct {
	RoleCode string `json:"role_code"`
}

// PermissionConfig 配置结构
type PermissionConfig struct {
	TTL time.Duration
}
