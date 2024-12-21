package zLib

import (
	"fmt"
	"strconv"
)

// StringIsEmpty 判断字符串是否为空
func StringIsEmpty(str string) bool {
	return len(str) == 0
}

// StringToNum 字符串转数字
func StringToNum(str string) (uint, error) {
	num, err := strconv.ParseUint(str, 10, strconv.IntSize)
	if err != nil {
		return 0, err
	}

	return uint(num), nil
}

// ToString 转换为字符串
func ToString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
