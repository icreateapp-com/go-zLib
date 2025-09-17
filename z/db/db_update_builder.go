package db

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// rawUpdateCondition 原生条件
type rawUpdateCondition struct {
	query string        // SQL 查询条件
	args  []interface{} // 查询参数
}

type UpdateBuilder[T IModel] struct {
	TX            *gorm.DB             // 事务支持
	Query         Query                // 查询参数
	Context       context.Context      // 上下文
	rawConditions []rawUpdateCondition // 原生条件
}

// WithContext 设置上下文
func (q *UpdateBuilder[T]) WithContext(ctx context.Context) *UpdateBuilder[T] {
	newBuilder := q.clone()
	newBuilder.Context = ctx
	return newBuilder
}

// Where 添加 WHERE 条件
func (q *UpdateBuilder[T]) Where(query string, args ...interface{}) *UpdateBuilder[T] {
	newBuilder := q.clone()
	newBuilder.rawConditions = append(newBuilder.rawConditions, rawUpdateCondition{
		query: query,
		args:  args,
	})
	return newBuilder
}

// clone 克隆 UpdateBuilder 实例
func (q *UpdateBuilder[T]) clone() *UpdateBuilder[T] {
	newBuilder := &UpdateBuilder[T]{
		TX:      q.TX,
		Query:   q.Query,
		Context: q.Context,
	}

	// 深拷贝 rawConditions
	if len(q.rawConditions) > 0 {
		newBuilder.rawConditions = make([]rawUpdateCondition, len(q.rawConditions))
		copy(newBuilder.rawConditions, q.rawConditions)
	}

	return newBuilder
}

func (q UpdateBuilder[T]) Update(query Query, values T, customFunc ...func(*gorm.DB) *gorm.DB) (bool, error) {
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

	// 先应用初始化时的 Query 参数
	if len(q.Query.Search) > 0 || len(q.Query.Required) > 0 {
		var err error
		db, err = ParseSearch(db, q.Query.Search, q.Query.Required)
		if err != nil {
			return false, WrapDBError(err)
		}
	}

	// 应用自定义函数（如 Select、Omit 等）
	if len(customFunc) > 0 && customFunc[0] != nil {
		db = customFunc[0](db)
	}

	// 再应用传入的查询参数
	db, err := ParseSearch(db, query.Search, query.Required)
	if err != nil {
		return false, WrapDBError(err)
	}

	if err := db.Updates(&values).Error; err != nil {
		return false, WrapDBError(err)
	}

	return true, nil
}

func (q UpdateBuilder[T]) UpdateByID(id interface{}, values T, customFunc ...func(*gorm.DB) *gorm.DB) (bool, error) {
	queryBuilder := QueryBuilder[T]{
		TX:      q.TX,
		Context: q.Context,
	}
	exists, _ := queryBuilder.ExistsById(id)
	if !exists {
		return false, WrapDBError(errors.New("row not found"))
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

		// 应用上下文
		if q.Context != nil {
			db = db.WithContext(q.Context)
		}

		// 应用原生条件
		for _, condition := range q.rawConditions {
			db = db.Where(condition.query, condition.args...)
		}

		// 应用自定义函数（如 Select、Omit 等）
		db = customFunc[0](db)

		// 添加 ID 条件并执行更新
		if err := db.Where("id = ?", id).Updates(&values).Error; err != nil {
			return false, WrapDBError(err)
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

	// 创建新的 UpdateBuilder，保持所有字段
	newBuilder := UpdateBuilder[T]{
		TX:            q.TX,
		Query:         q.Query,
		Context:       q.Context,
		rawConditions: q.rawConditions,
	}

	return newBuilder.Update(query, values)
}
