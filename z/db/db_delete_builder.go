package db

import (
	"context"

	"gorm.io/gorm"
)

// rawCondition 原生条件
type rawDeleteCondition struct {
	query string        // SQL 查询条件
	args  []interface{} // 查询参数
}

type DeleteBuilder[T IModel] struct {
	TX            *gorm.DB             // 事务支持
	Context       context.Context      // 上下文
	rawConditions []rawDeleteCondition // 原生条件
}

// WithContext 设置上下文
func (q *DeleteBuilder[T]) WithContext(ctx context.Context) *DeleteBuilder[T] {
	newBuilder := q.clone()
	newBuilder.Context = ctx
	return newBuilder
}

// Where 添加 WHERE 条件
func (q *DeleteBuilder[T]) Where(query string, args ...interface{}) *DeleteBuilder[T] {
	newBuilder := q.clone()
	newBuilder.rawConditions = append(newBuilder.rawConditions, rawDeleteCondition{
		query: query,
		args:  args,
	})
	return newBuilder
}

// clone 克隆 DeleteBuilder 实例
func (q *DeleteBuilder[T]) clone() *DeleteBuilder[T] {
	newBuilder := &DeleteBuilder[T]{
		TX:      q.TX,
		Context: q.Context,
	}

	// 深拷贝 rawConditions
	if len(q.rawConditions) > 0 {
		newBuilder.rawConditions = make([]rawDeleteCondition, len(q.rawConditions))
		copy(newBuilder.rawConditions, q.rawConditions)
	}

	return newBuilder
}

func (q DeleteBuilder[T]) Delete(query ...Query) (bool, error) {
	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		db = DB.Model(&zero)
	}

	// 应用上下文
	if q.Context != nil {
		db = db.WithContext(q.Context)
	}

	// 应用原生条件
	for _, condition := range q.rawConditions {
		db = db.Where(condition.query, condition.args...)
	}

	// 如果提供了查询参数，则应用查询条件
	if len(query) > 0 {
		db, err := ParseSearch(db, query[0].Search, query[0].Required)
		if err != nil {
			return false, WrapDBError(err) // 使用错误包装器
		}

		if err := db.Delete(&zero).Error; err != nil {
			return false, WrapDBError(err) // 使用错误包装器
		}
	} else {
		// 没有额外查询条件时直接删除
		if err := db.Delete(&zero).Error; err != nil {
			return false, WrapDBError(err) // 使用错误包装器
		}
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

	// 直接调用 Delete 方法，传入构建的查询条件
	return q.Delete(query)
}
