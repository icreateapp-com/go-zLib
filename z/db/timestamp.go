package db

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/icreateapp-com/go-zLib/z"
)

// WrapTime 包装了 time.Time 类型，用于自定义时间格式的序列化和反序列化。
type WrapTime struct {
	time.Time
}

// MarshalJSON 实现了 json.Marshaler 接口，将时间格式化为 "2006-01-02 15:04:05" 的字符串形式。
// 这个方法确保在 JSON 序列化时使用统一的时间格式。
func (t WrapTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))), nil
}

// UnmarshalJSON 自定义 JSON 反序列化格式
func (t *WrapTime) UnmarshalJSON(data []byte) error {
	str := string(data)

	// 检查是否为空值或无效值
	if len(str) < 2 || str == `""` || str == `null` {
		t.Time = time.Time{} // 设置为零值
		return nil
	}

	str = str[1 : len(str)-1] // 去除引号

	// 检查去除引号后是否为空
	if str == "" {
		t.Time = time.Time{} // 设置为零值
		return nil
	}

	// 明确指定北京时区进行解析，确保时间一致性
	timezone := z.Config.GetString("config.timezone")
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// 如果加载时区失败，使用本地时区
		loc = time.Local
	}

	// 尝试多种时间格式进行解析
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.999999999Z",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		var tt time.Time
		tt, err = time.ParseInLocation(format, str, loc)
		if err == nil {
			t.Time = tt
			return nil
		}
	}

	// 如果所有格式都解析失败，设置为零值而不是返回错误
	// 这样可以避免因为数据库中的无效时间值导致整个接口失败
	t.Time = time.Time{}
	return nil
}

// Value 实现了 driver.Valuer 接口，用于将时间值插入到数据库中。
// 如果时间是零值（即未设置），则返回 nil；否则返回实际的时间值。
func (t WrapTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

// Scan 实现了 sql.Scanner 接口，用于从数据库中读取时间值并将其转换为 WrapTime 类型。
// 如果传入的值不是 time.Time 类型，则返回错误。
func (t *WrapTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = WrapTime{Time: value}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", v)
}
