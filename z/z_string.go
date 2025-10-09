package z

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
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

// ToSnakeCase 驼峰转蛇形
func ToSnakeCase(str string) string {
	var snakeCase string
	for i, char := range str {
		if i > 0 && unicode.IsUpper(char) {
			snakeCase += "_" + strings.ToLower(string(char))
		} else {
			snakeCase += strings.ToLower(string(char))
		}
	}
	return snakeCase
}

// GetJsonByString 从字符串中提取JSON对象
func GetJsonByString(input string) (interface{}, error) {
	// Define a regular expression that matches JSON objects, arrays, strings, numbers, booleans, and null.
	re := regexp.MustCompile(`(?s)\{.*?\}|\[.*?\]|"(?:\\.|[^"\\])*"|[-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?|true|false|null`)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no valid JSON found")
	}

	var result interface{}
	jsonValue := matches[0]

	// Try to unmarshal as JSON object or array.
	if err := json.Unmarshal([]byte(jsonValue), &result); err == nil {
		return result, nil
	}

	// Try to parse as a number.
	if num, err := strconv.ParseFloat(jsonValue, 64); err == nil {
		return num, nil
	}

	// Check for true, false, null.
	switch jsonValue {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}

	// If it's a JSON string, remove quotes and return the string value.
	if len(jsonValue) > 0 && jsonValue[0] == '"' && jsonValue[len(jsonValue)-1] == '"' {
		result, _ = strconv.Unquote(jsonValue)
		return result, nil
	}

	return nil, fmt.Errorf("no valid JSON found")
}

// TernaryString 三元运算符
func TernaryString(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}

// StringToSlice 将输入字符串按指定分割因素分割成切片
func StringToSlice(input string, separators ...string) []string {
	for _, sep := range separators[1:] {
		input = strings.ReplaceAll(input, sep, separators[0])
	}

	splitSlice := strings.Split(input, separators[0])

	for i := range splitSlice {
		splitSlice[i] = strings.TrimSpace(splitSlice[i])
	}

	var result []string
	for _, element := range splitSlice {
		if element != "" {
			result = append(result, element)
		}
	}

	return result
}

// ToInt64 获取Int64
func ToInt64(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}

// CountWords 获取字符串长度
func CountWords(text string) int {
	return utf8.RuneCountInString(text)
}

// Substring 按字符截取字符串，支持可选后缀参数
func Substring(s string, start int, length int, suffix ...string) string {
	if start < 0 || length <= 0 {
		return ""
	}

	runes := []rune(s)
	total := len(runes)

	if start >= total {
		return ""
	}

	end := start + length
	if end > total {
		end = total
	}

	result := string(runes[start:end])
	// 如果还有更多字符并且提供了后缀，则添加后缀
	if len(runes) > end && len(suffix) > 0 && suffix[0] != "" {
		result += suffix[0]
	}

	return result
}
