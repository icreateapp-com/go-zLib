package permission_provider

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/icreateapp-com/go-zLib/z/providers/redis_provider"
)

// RedisAdapter Casbin Redis Adapter（基于 redis_provider）
type RedisAdapter struct {
	redis *redis_provider.Redis
}

// NewRedisAdapter 创建 Redis adapter
func NewRedisAdapter(redis *redis_provider.Redis) *RedisAdapter {
	return &RedisAdapter{redis: redis}
}

// LoadPolicy 从 Redis 加载策略
func (a *RedisAdapter) LoadPolicy(model model.Model) error {
	// 加载策略规则
	policyKeys, err := a.redis.Keys("casbin_policy_*")
	if err != nil {
		return fmt.Errorf("failed to get policy keys: %w", err)
	}

	for _, key := range policyKeys {
		var policy []string
		if err := a.redis.Get(key, &policy); err != nil {
			continue
		}

		if len(policy) >= 4 {
			sec := "p"
			pType := "p"
			if len(policy) > 4 {
				sec = policy[0]
				pType = policy[1]
				policy = policy[2:]
			}

			line := strings.Join(policy, ", ")
			if !model.AddDef(sec, pType, line) {
				return fmt.Errorf("failed to add policy")
			}
		}
	}

	// 加载角色规则
	roleKeys, err := a.redis.Keys("casbin_role_*")
	if err != nil {
		return fmt.Errorf("failed to get role keys: %w", err)
	}

	for _, key := range roleKeys {
		var role []string
		if err := a.redis.Get(key, &role); err != nil {
			continue
		}

		if len(role) >= 3 {
			line := strings.Join(role, ", ")
			if !model.AddDef("g", "g", line) {
				return fmt.Errorf("failed to add role")
			}
		}
	}

	return nil
}

// SavePolicy 保存策略到 Redis
func (a *RedisAdapter) SavePolicy(model model.Model) error {
	// 先删除所有现有策略
	a.clearAllPolicies()

	// 保存策略规则
	if rules, err := model.GetPolicy("p", "p"); err == nil && len(rules) > 0 {
		for _, rule := range rules {
			key := fmt.Sprintf("casbin_policy_%s", strings.Join(rule, "_"))
			if err := a.redis.Set(key, rule, 0); err != nil {
				return fmt.Errorf("failed to save policy: %w", err)
			}
		}
	}

	// 保存角色规则
	if rules, err := model.GetPolicy("g", "g"); err == nil && len(rules) > 0 {
		for _, rule := range rules {
			key := fmt.Sprintf("casbin_role_%s", strings.Join(rule, "_"))
			if err := a.redis.Set(key, rule, 0); err != nil {
				return fmt.Errorf("failed to save role: %w", err)
			}
		}
	}

	return nil
}

// clearAllPolicies 清理所有策略
func (a *RedisAdapter) clearAllPolicies() {
	policyKeys, _ := a.redis.Keys("casbin_policy_*")
	for _, key := range policyKeys {
		a.redis.Delete(key)
	}

	roleKeys, _ := a.redis.Keys("casbin_role_*")
	for _, key := range roleKeys {
		a.redis.Delete(key)
	}
}

// AddPolicy 添加策略
func (a *RedisAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	if sec == "p" {
		key := fmt.Sprintf("casbin_policy_%s", strings.Join(rule, "_"))
		return a.redis.Set(key, rule, 0)
	} else if sec == "g" {
		key := fmt.Sprintf("casbin_role_%s", strings.Join(rule, "_"))
		return a.redis.Set(key, rule, 0)
	}

	return fmt.Errorf("unsupported section: %s", sec)
}

// RemovePolicy 删除策略
func (a *RedisAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	if sec == "p" {
		key := fmt.Sprintf("casbin_policy_%s", strings.Join(rule, "_"))
		return a.redis.Delete(key)
	} else if sec == "g" {
		key := fmt.Sprintf("casbin_role_%s", strings.Join(rule, "_"))
		return a.redis.Delete(key)
	}

	return fmt.Errorf("unsupported section: %s", sec)
}

// RemoveFilteredPolicy 批量删除策略
func (a *RedisAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	// 获取所有策略
	var keys []string

	if sec == "p" {
		keys, _ = a.redis.Keys("casbin_policy_*")
	} else if sec == "g" {
		keys, _ = a.redis.Keys("casbin_role_*")
	}

	for _, key := range keys {
		var rule []string
		if err := a.redis.Get(key, &rule); err != nil {
			continue
		}

		// 检查是否匹配过滤条件
		matched := true
		for i, value := range fieldValues {
			if i >= len(rule) {
				matched = false
				break
			}
			if rule[fieldIndex+i] != value {
				matched = false
				break
			}
		}

		if matched {
			a.redis.Delete(key)
		}
	}

	return nil
}

// IsFiltered 返回 true 表示支持过滤
func (a *RedisAdapter) IsFiltered() bool {
	return true
}

// AddPolicies 批量添加策略
func (a *RedisAdapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	for _, rule := range rules {
		if err := a.AddPolicy(sec, ptype, rule); err != nil {
			return err
		}
	}
	return nil
}

// RemovePolicies 批量删除策略
func (a *RedisAdapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
	for _, rule := range rules {
		if err := a.RemovePolicy(sec, ptype, rule); err != nil {
			return err
		}
	}
	return nil
}

// Ensure ensure adapter是初始化
func (a *RedisAdapter) Ensure() error {
	return nil
}

// Close 关闭 adapter
func (a *RedisAdapter) Close() error {
	return nil
}

// BatchTest 批量测试策略
func (a *RedisAdapter) BatchTest(sec string, ptype string, rules [][]string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// UpdatePolicy 更新策略
func (a *RedisAdapter) UpdatePolicy(sec string, ptype string, oldRule, newRule []string) error {
	if err := a.RemovePolicy(sec, ptype, oldRule); err != nil {
		return err
	}
	return a.AddPolicy(sec, ptype, newRule)
}

// UpdatePolicies 批量更新策略
func (a *RedisAdapter) UpdatePolicies(sec string, ptype string, oldRules, newRules [][]string) error {
	for i, oldRule := range oldRules {
		if i < len(newRules) {
			if err := a.UpdatePolicy(sec, ptype, oldRule, newRules[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetFilteredPolicy 获取过滤后的策略
func (a *RedisAdapter) GetFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) ([][]string, error) {
	var keys []string
	var result [][]string

	if sec == "p" {
		keys, _ = a.redis.Keys("casbin_policy_*")
	} else if sec == "g" {
		keys, _ = a.redis.Keys("casbin_role_*")
	}

	for _, key := range keys {
		var rule []string
		if err := a.redis.Get(key, &rule); err != nil {
			continue
		}

		// 检查是否匹配过滤条件
		matched := true
		for i, value := range fieldValues {
			if i >= len(rule) {
				matched = false
				break
			}
			if rule[fieldIndex+i] != value {
				matched = false
				break
			}
		}

		if matched {
			result = append(result, rule)
		}
	}

	return result, nil
}

// LoadFilteredPolicy 加载过滤后的策略
func (a *RedisAdapter) LoadFilteredPolicy(model model.Model) error {
	// 加载策略规则
	policyKeys, err := a.redis.Keys("casbin_policy_*")
	if err != nil {
		return fmt.Errorf("failed to get policy keys: %w", err)
	}

	for _, key := range policyKeys {
		var policy []string
		if err := a.redis.Get(key, &policy); err != nil {
			continue
		}

		if len(policy) >= 4 {
			sec := "p"
			pType := "p"
			if len(policy) > 4 {
				sec = policy[0]
				pType = policy[1]
				policy = policy[2:]
			}

			line := strings.Join(policy, ", ")
			if !model.AddDef(sec, pType, line) {
				return fmt.Errorf("failed to add policy")
			}
		}
	}

	// 加载角色规则
	roleKeys, err := a.redis.Keys("casbin_role_*")
	if err != nil {
		return fmt.Errorf("failed to get role keys: %w", err)
	}

	for _, key := range roleKeys {
		var role []string
		if err := a.redis.Get(key, &role); err != nil {
			continue
		}

		if len(role) >= 3 {
			line := strings.Join(role, ", ")
			if !model.AddDef("g", "g", line) {
				return fmt.Errorf("failed to add role")
			}
		}
	}

	return nil
}

var _ persist.Adapter = &RedisAdapter{}
