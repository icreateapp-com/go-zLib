package zLib

import (
	"fmt"
	"net/url"
	"sort"
)

// GetSortedMapKeys 返回排序后的 MAP 所有键
func GetSortedMapKeys(elements map[string]interface{}) []string {
	keys := make([]string, 0, len(elements))

	for k := range elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// GetMapKeys 返回 MAP 的所有键
func GetMapKeys(elements map[string]interface{}) []string {
	keys := make([]string, 0, len(elements))

	for k, _ := range elements {
		keys = append(keys, k)
	}

	return keys
}

// ToValues 转换表单项
func ToValues(data map[string]interface{}) url.Values {
	values := url.Values{}

	for k, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			nestedValues := ToValues(val)
			for nk, nv := range nestedValues {
				values.Add(fmt.Sprintf("%s[%s]", k, nk), nv[0])
			}
		default:
			values.Add(k, fmt.Sprintf("%v", v))
		}
	}

	return values
}
