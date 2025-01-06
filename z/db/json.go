package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JsonField 结构体用于存储 JSON 数据
type JsonField struct {
	Data interface{} `json:"data"`
}

// Value 方法用于将 JsonField 序列化为 JSON 字符串
func (j JsonField) Value() (driver.Value, error) {
	if j.Data == nil {
		return nil, nil
	}
	fmt.Println(j.Data)
	return json.Marshal(j.Data)
}

// Scan 方法用于将 JSON 字符串反序列化为 JsonField
func (j *JsonField) Scan(value interface{}) error {
	if value == nil {
		j.Data = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	if len(b) == 0 {
		j.Data = nil
		return nil
	}

	err := json.Unmarshal(b, &j.Data)

	fmt.Println(j.Data)

	return err
}
