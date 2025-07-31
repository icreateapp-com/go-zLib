package db

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ParseInclude 解析预加载关联数据
func ParseInclude(db *gorm.DB, includes []string) (*gorm.DB, error) {
	if len(includes) == 0 {
		return db, nil
	}

	for _, include := range includes {
		// 验证预加载字段名安全性
		if !isValidIncludeName(include) {
			return nil, errors.New("invalid include name: " + include)
		}

		// 支持嵌套预加载，如 "User.Profile"
		db = db.Preload(include)
	}

	return db, nil
}

// isValidIncludeName 验证预加载字段名是否安全
func isValidIncludeName(name string) bool {
	if name == "" {
		return false
	}

	// 分割嵌套关联（如 "User.Profile"）
	parts := strings.Split(name, ".")
	for _, part := range parts {
		if !isValidFieldName(part) {
			return false
		}
	}

	return true
}
