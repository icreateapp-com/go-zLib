package db

import (
	"errors"

	"gorm.io/gorm"
)

type UpdateBuilder[T IModel] struct {
	TX *gorm.DB
}

func (q UpdateBuilder[T]) Update(query Query, values T, customFunc ...func(*gorm.DB) *gorm.DB) (bool, error) {
	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		db = DB.Model(&zero)
	}

	// 应用自定义函数（如 Select、Omit 等）
	if len(customFunc) > 0 && customFunc[0] != nil {
		db = customFunc[0](db)
	}

	db, err := ParseSearch(db, query.Search, query.Required)
	if err != nil {
		return false, err
	}

	if err := db.Updates(&values).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (q UpdateBuilder[T]) UpdateByID(id interface{}, values T, customFunc ...func(*gorm.DB) *gorm.DB) (bool, error) {
	queryBuilder := QueryBuilder[T]{TX: q.TX}
	exists, _ := queryBuilder.ExistsById(id)
	if !exists {
		return false, errors.New("row not found")
	}

	// 如果提供了自定义函数，使用直接更新方式
	if len(customFunc) > 0 && customFunc[0] != nil {
		var zero T
		var db *gorm.DB
		if q.TX != nil {
			db = q.TX.Model(&zero)
		} else {
			db = DB.Model(&zero)
		}

		// 应用自定义函数（如 Select、Omit 等）
		db = customFunc[0](db)

		// 添加 ID 条件并执行更新
		if err := db.Where("id = ?", id).Updates(&values).Error; err != nil {
			return false, err
		}

		return true, nil
	}

	// 默认行为：使用原有的查询方式
	query := Query{
		Search: []ConditionGroup{
			{
				Conditions: [][]interface{}{{"id", id}},
			},
		},
	}
	return q.Update(query, values)
}
