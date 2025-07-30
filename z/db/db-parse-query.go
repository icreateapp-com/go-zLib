package db

import (
	"gorm.io/gorm"
)

// ParseQuery 解析完整查询
func ParseQuery(query Query, db *gorm.DB) (*gorm.DB, error) {
	var err error

	if db, err = ParseFilter(db, query.Filter); err != nil {
		return nil, err
	}

	if db, err = ParseSearch(db, query.Search, query.Required); err != nil {
		return nil, err
	}

	if db, err = ParseInclude(db, query.Include); err != nil {
		return nil, err
	}

	if db, err = ParseOrderBy(db, query.OrderBy); err != nil {
		return nil, err
	}

	if query.Page > 0 {
		if db, err = ParsePage(db, query.Page, query.Limit); err != nil {
			return nil, err
		}
	} else if query.Limit > 0 {
		if db, err = ParseLimit(db, query.Limit); err != nil {
			return nil, err
		}
	}

	return db, nil
}
