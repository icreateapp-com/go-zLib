package z

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RemoveAtIndex 删除切片指定位置的元素
func RemoveAtIndex(slice []interface{}, index int) []interface{} {
	if index < 0 || index >= len(slice) {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}

// SortMapSlice 对 map 切片按照指定字段排序
func SortMapSlice(data []map[string]interface{}, field string, ascending bool) {
	sort.Slice(data, func(i, j int) bool {
		// 获取字段值
		valI, okI := data[i][field]
		valJ, okJ := data[j][field]

		if !okI || !okJ {
			// 如果字段不存在，默认认为相等
			return false
		}

		// 根据字段类型进行比较
		switch valI.(type) {
		case time.Time:
			timestampI := valI.(time.Time)
			timestampJ := valJ.(time.Time)
			if ascending {
				return timestampI.Before(timestampJ)
			}
			return timestampI.After(timestampJ)
		case int:
			intI := valI.(int)
			intJ := valJ.(int)
			if ascending {
				return intI < intJ
			}
			return intI > intJ
		case float64:
			floatI := valI.(float64)
			floatJ := valJ.(float64)
			if ascending {
				return floatI < floatJ
			}
			return floatI > floatJ
		case string:
			strI := valI.(string)
			strJ := valJ.(string)
			if ascending {
				return strings.Compare(strI, strJ) < 0
			}
			return strings.Compare(strI, strJ) > 0
		default:
			// 默认情况下，认为相等
			return false
		}
	})
}

// InSlice 检查某个元素是否存在于切片中
func InSlice[T comparable](val T, list []T) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

// IsMapString 检查给定的值是否为 map[string]interface{}
func IsMapString(value interface{}) bool {
	_, ok := value.(map[string]interface{})
	return ok
}

// AddFieldToMap 向 map 中添加新的键值对
func AddFieldToMap(value *interface{}, key string, valueToAdd interface{}) error {
	// 尝试将 value 断言为 map[string]interface{}
	m, ok := (*value).(map[string]interface{})
	if !ok {
		return fmt.Errorf("value is not a map[string]interface{}")
	}

	// 向 map 中添加新的键值对
	m[key] = valueToAdd
	return nil
}

// HasFieldInMap 检查给定的 map 中是否存在指定的字段
func HasFieldInMap(value interface{}, field string) bool {
	// 尝试将 value 断言为 map[string]interface{}
	m, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	// 检查指定的字段是否存在
	_, exists := m[field]
	return exists
}

// SliceUnique 删除切片中的重复元素
func SliceUnique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	var result []T
	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// JoinStringSlice 将字符串切片连接为逗号分隔的引用字符串
func JoinStringSlice(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	quoted := make([]string, len(strs))
	for i, str := range strs {
		quoted[i] = fmt.Sprintf(`"%s"`, str)
	}

	// 用逗号连接所有引用的字符串
	result := quoted[0]
	for i := 1; i < len(quoted); i++ {
		result += ", " + quoted[i]
	}
	return result
}
