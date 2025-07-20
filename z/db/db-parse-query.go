package db

import (
	"gorm.io/gorm"
)

// QueryParser 查询解析器
type QueryParser[T IModel] struct {
	TX *gorm.DB
}

// ParseQuery 解析完整查询
func (p QueryParser[T]) ParseQuery(query Query, db *gorm.DB) (*gorm.DB, error) {
	var err error

	if db, err = p.ParseFilter(db, query.Filter); err != nil {
		return nil, err
	}

	if db, err = p.ParseSearch(db, query.Search, query.Required); err != nil {
		return nil, err
	}

	if db, err = p.ParseOrderBy(db, query.OrderBy); err != nil {
		return nil, err
	}

	if db, err = p.ParseLimit(db, query.Limit); err != nil {
		return nil, err
	}

	if db, err = p.ParsePage(db, query.Page); err != nil {
		return nil, err
	}

	return db, nil
}

// getDB 获取数据库连接（支持事务）
func (p QueryParser[T]) getDB() *gorm.DB {
	if p.TX != nil {
		return p.TX
	}
	return DB.DB
}
