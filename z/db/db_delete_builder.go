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

	db, err := ParseSearch(db, query.Search, query.Required)
	if err != nil {
		return false, WrapDBError(err) // 使用错误包装器
	}

	if err := db.Delete(&zero).Error; err != nil {
		return false, WrapDBError(err) // 使用错误包装器
	}

	return true, nil
}

func (q DeleteBuilder[T]) DeleteByID(id interface{}, additionalQuery ...Query) (bool, error) {
	// 构建基础的ID查询条件
	query := Query{
		Search: []ConditionGroup{
			{
				Conditions: [][]interface{}{{"id", id}},
				Operator:   "AND",
			},
		},
	}

	// 如果有额外的查询条件，合并到现有查询中
	if len(additionalQuery) > 0 {
		additional := additionalQuery[0]

		// 合并搜索条件
		if len(additional.Search) > 0 {
			query.Search = append(query.Search, additional.Search...)
		}

		// 合并其他查询参数
		if len(additional.Filter) > 0 {
			query.Filter = additional.Filter
		}
		if len(additional.Required) > 0 {
			query.Required = additional.Required
		}
	}

	return q.Delete(query)
}
