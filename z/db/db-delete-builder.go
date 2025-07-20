package db

import "gorm.io/gorm"

type DeleteBuilder[T IModel] struct {
	TX *gorm.DB
}

func (q DeleteBuilder[T]) Delete(query Query) (bool, error) {
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

	if err := db.Delete(&zero).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (q DeleteBuilder[T]) DeleteByID(id interface{}) (bool, error) {
	query := Query{
		Search: []ConditionGroup{
			{{"id", id}},
		},
	}
	return q.Delete(query)
}
