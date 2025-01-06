package db

import (
	"database/sql/driver"
	"fmt"
	"time"
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
