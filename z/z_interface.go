package z

import (
	"fmt"
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
