package db

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ParseOrderBy 解析排序
func ParseOrderBy(db *gorm.DB, orderBy [][]string) (*gorm.DB, error) {
	if len(orderBy) == 0 {
		return db, nil
	}

	for _, order := range orderBy {
		if len(order) == 1 {
			// 如果只有一个元素，使用 "asc" 作为默认方向
			order = append(order, "asc")
		} else if len(order) != 2 {
			return nil, errors.New("invalid order condition: each order condition must have exactly 1 or 2 elements")
		}

		field := order[0]
		if !isValidFieldName(field) {
			return nil, errors.New("invalid field name in order by: " + field)
		}

		direction := strings.ToLower(order[1])

		// 验证方向
		validDirections := map[string]bool{"asc": true, "desc": true}
		if !validDirections[direction] {
			return nil, errors.New("invalid order direction: '" + direction + "' is not a valid direction")
		}

		// 生成排序子句
		orderClause := fmt.Sprintf("%s %s", DB.F(field), strings.ToUpper(direction))
		db = db.Order(orderClause)
	}

	return db, nil
}

// ParseLimit 解析限制条数
func ParseLimit(db *gorm.DB, limit int) (*gorm.DB, error) {
	if limit > 0 {
		if limit > 100 {
			limit = 100
		}
		db = db.Limit(limit)
	}
	return db, nil
}

// ParsePage 解析分页
func ParsePage(db *gorm.DB, page int, limit int) (*gorm.DB, error) {
	if page > 0 {
		if limit <= 0 {
			limit = DefaultPageSize
		} else if limit > 100 {
			limit = 100
		}
		offset := (page - 1) * limit
		db = db.Offset(offset).Limit(limit)
	}
	return db, nil
}
