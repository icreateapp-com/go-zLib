package db

import (
	"errors"

	"gorm.io/gorm"
)

// ParseFilter 解析字段过滤
func ParseFilter(db *gorm.DB, filter []string) (*gorm.DB, error) {
	if len(filter) == 0 {
		return db, nil
	}

	var selectFields []string
	for _, f := range filter {
		// 防止SQL注入
		if !isValidFieldName(f) {
			return nil, errors.New("invalid field name: " + f)
		}
		f = DB.F(f)
		selectFields = append(selectFields, f)
	}

	db = db.Select(selectFields)
	return db, nil
}

// isValidFieldName 验证字段名是否安全（防止SQL注入）
func isValidFieldName(field string) bool {
	// 只允许字母、数字、下划线和点号
	for _, char := range field {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '.') {
			return false
		}
	}
	return len(field) > 0
}
