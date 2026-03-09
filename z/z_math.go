package z

import (
	"golang.org/x/exp/constraints"
	"reflect"
)

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

// Clamp 限制值在 [min, max] 区间内
func Clamp[T constraints.Integer | constraints.Float](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Min 返回两个可比较值中的较小值
func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max 返回两个可比较值中的较大值
func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
