package db

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ParseOrderBy 解析排序
func (p QueryParser[T]) ParseOrderBy(db *gorm.DB, orderBy []string) (*gorm.DB, error) {
	if len(orderBy) == 0 {
		return db, nil
	}

	var orderClauses []string
	for _, order := range orderBy {
		parts := strings.Fields(order)
		if len(parts) == 0 {
			continue
		}

		field := parts[0]
		if !p.isValidFieldName(field) {
			return nil, errors.New("invalid field name in order by: " + field)
		}

		direction := "ASC"
		if len(parts) > 1 {
			dir := strings.ToUpper(parts[1])
			if dir == "DESC" || dir == "ASC" {
				direction = dir
			}
		}

		orderClauses = append(orderClauses, DB.F(field)+" "+direction)
	}

	if len(orderClauses) > 0 {
		db = db.Order(strings.Join(orderClauses, ", "))
	}

	return db, nil
}

// ParseLimit 解析限制条数
func (p QueryParser[T]) ParseLimit(db *gorm.DB, limit []int) (*gorm.DB, error) {
	if len(limit) > 0 && limit[0] > 0 {
		db = db.Limit(limit[0])
	}
	return db, nil
}

// ParsePage 解析分页
func (p QueryParser[T]) ParsePage(db *gorm.DB, page []int) (*gorm.DB, error) {
	if len(page) >= 2 && page[0] > 0 && page[1] > 0 {
		offset := (page[0] - 1) * page[1]
		db = db.Offset(offset).Limit(page[1])
	}
	return db, nil
}
