package z

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Equal 判断两个值是否相等
func Equal(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// GreaterThanLength 判断a的长度是否大于b
func GreaterThanLength(a, b interface{}) bool {
	lengthA := GetLength(a)
	lengthB, ok := b.(float64)
	if !ok {
		return false
	}
	return lengthA > int(lengthB)
}

// GreaterThanOrEqualLength 判断a的长度是否大于等于b
func GreaterThanOrEqualLength(a, b interface{}) bool {
	lengthA := GetLength(a)
	lengthB, ok := b.(float64)
	if !ok {
		return false
	}
	return lengthA >= int(lengthB)
}

// LessThanLength 判断a的长度是否小于b
func LessThanLength(a, b interface{}) bool {
	lengthA := GetLength(a)
	lengthB, ok := b.(float64)
	if !ok {
		return false
	}
	return lengthA < int(lengthB)
}

// LessThanOrEqualLength 判断a的长度是否小于等于b
func LessThanOrEqualLength(a, b interface{}) bool {
	lengthA := GetLength(a)
	lengthB, ok := b.(float64)
	if !ok {
		return false
	}
	return lengthA <= int(lengthB)
}

// Contains 判断a是否包含b
func Contains(a, b interface{}) bool {
	switch a := a.(type) {
	case string:
		bStr, ok := b.(string)
		if !ok {
			return false
		}
		return strings.Contains(a, bStr)
	case []interface{}:
		for _, item := range a {
			if Equal(item, b) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// IsEmpty 判断a是否为空
func IsEmpty(a interface{}) bool {
	switch a := a.(type) {
	case string:
		return a == ""
	case []interface{}:
		return len(a) == 0
	case map[string]interface{}:
		return len(a) == 0
	default:
		return false
	}
}

// GreaterThan 判断a是否大于b
func GreaterThan(a, b interface{}) bool {
	return CompareNumeric(a, b, func(x, y float64) bool { return x > y })
}

// GreaterThanOrEqual 判断a是否大于等于b
func GreaterThanOrEqual(a, b interface{}) bool {
	return CompareNumeric(a, b, func(x, y float64) bool { return x >= y })
}

// LessThan 判断a是否小于b
func LessThan(a, b interface{}) bool {
	return CompareNumeric(a, b, func(x, y float64) bool { return x < y })
}

// LessThanOrEqual 判断a是否小于等于b
func LessThanOrEqual(a, b interface{}) bool {
	return CompareNumeric(a, b, func(x, y float64) bool { return x <= y })
}

// IsTrue 判断a是否为真
func IsTrue(a interface{}) bool {
	b, ok := a.(bool)
	if !ok {
		return false
	}
	return b
}

// IsFalse 判断a是否为假
func IsFalse(a interface{}) bool {
	b, ok := a.(bool)
	if !ok {
		return false
	}
	return !b
}

// CompareNumeric 比较两个数字
func CompareNumeric(a, b interface{}, compareFunc func(float64, float64) bool) bool {
	numA, okA := a.(float64)
	numB, okB := b.(float64)
	if !okA || !okB {
		return false
	}
	return compareFunc(numA, numB)
}

// GetLength 获取变量的长度
func GetLength(a interface{}) int {
	switch a := a.(type) {
	case string:
		return len(a)
	case []interface{}:
		return len(a)
	default:
		return 0
	}
}

// ExtractField 提取嵌套字段或索引值
func ExtractField(message interface{}, convertPath string) (interface{}, error) {
	parts := SplitPath(convertPath)

	for _, part := range parts {
		if val, ok := message.(map[string]interface{}); ok {
			var exists bool
			message, exists = val[part]
			if !exists {
				return nil, fmt.Errorf("field %s does not exist", part)
			}
		} else if val, ok := message.([]interface{}); ok {
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(val) {
				return nil, fmt.Errorf("invalid index %s", part)
			}
			message = val[index]
		} else {
			return nil, fmt.Errorf("cannot navigate into non-map or non-slice type")
		}
	}

	return message, nil
}

// SplitPath 将转换路径分割成部分，考虑到方括号中的索引
func SplitPath(path string) []string {
	var parts []string
	current := ""
	isInBracket := false
	bracketContent := ""

	for i := 0; i < len(path); i++ {
		char := path[i]
		switch {
		case char == '.' && !isInBracket:
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case char == '[':
			isInBracket = true
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case char == ']':
			isInBracket = false
			if bracketContent != "" {
				parts = append(parts, bracketContent)
				bracketContent = ""
			}
		default:
			if isInBracket {
				bracketContent += string(char)
			} else {
				current += string(char)
			}
		}
	}

	if current != "" {
		parts = append(parts, current)
	}
	if bracketContent != "" {
		parts = append(parts, bracketContent)
	}

	return parts
}

// ToInterface 将结构体转换为 map
func ToInterface(data interface{}, result interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, result)
}

// IsString 判断a是否为字符串类型
func IsString(a interface{}) bool {
	_, ok := a.(string)
	return ok
}

// IsScalar 判断输入的 interface{} 是否是标量类型（如 string, int, float64, bool）
func IsScalar(a interface{}) bool {
	switch a.(type) {
	case string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool:
		return true
	default:
		return false
	}
}

// ToMap 将 interface{} 转换为 map[string]interface{}
func ToMap(obj interface{}, fields ...string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	val := reflect.ValueOf(obj)

	if !val.IsValid() {
		return nil, errors.New("invalid value")
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, errors.New("nil pointer passed")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	fieldSet := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldSet[f] = struct{}{}
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		if _, ok := fieldSet[field.Name]; ok {
			if field.PkgPath != "" {
				return nil, fmt.Errorf("cannot access unexported field: %s", field.Name)
			}
			result[field.Name] = val.Field(i).Interface()
		}
	}

	return result, nil
}

// ToJsonString 将 interface{} 转换为 json.RawMessage
func ToJsonString(s interface{}) []byte {
	marshal, err := json.Marshal(s)
	if err != nil {
		return nil
	}

	return marshal
}

// GetValueInMap 获取 map 中的值
func GetValueInMap[T any](values map[string]interface{}, key string) (*T, error) {
	value, exists := values[key]
	if !exists {
		return nil, fmt.Errorf("missing %s in job payload", key)
	}

	// 检查value是否已经是指定类型T
	if v, ok := value.(T); ok {
		return &v, nil
	}

	// 如果不是，则尝试将其作为JSON进行解析
	var r T
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", value)), &r); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", key, err)
	}

	return &r, nil
}
