package z

import "reflect"

// ValidateRange 检查输入值是否在指定范围内，如果不在范围内或为空，则返回默认值
func ValidateRange[T int | float64](input *T, min, max, defaultValue T) T {
	if input == nil || reflect.ValueOf(input).IsNil() {
		return defaultValue
	}

	if *input >= min && *input <= max {
		return *input
	}

	return defaultValue
}
