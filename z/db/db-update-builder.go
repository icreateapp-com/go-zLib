package db

import (
	"errors"

	"gorm.io/gorm"
)

type UpdateBuilder[T IModel] struct {
	TX *gorm.DB
}

func (q UpdateBuilder[T]) Update(query Query, values T) (bool, error) {
	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		db = DB.Model(&zero)
	}

	// 使用解析器解析查询条件
	parser := QueryParser[T]{TX: q.TX}
	db, err := parser.ParseSearch(db, query.Search, query.Required)
	if err != nil {
		return false, err
	}

	if err := db.Updates(&values).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (q UpdateBuilder[T]) UpdateByID(id interface{}, values T) (bool, error) {
	queryBuilder := QueryBuilder[T]{TX: q.TX}
	exists, _ := queryBuilder.ExistsById(id)
	if !exists {
		return false, errors.New("row not found")
	}

	query := Query{
		Search: []ConditionGroup{
			{{"id", id}},
		},
	}
	return q.Update(query, values)
}
