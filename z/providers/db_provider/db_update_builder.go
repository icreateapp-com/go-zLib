package db_provider

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
	DB            *DB                  // 数据库连接（DI 注入）
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
		DB:            q.DB,
		TX:            q.TX,
		Query:         q.Query,
		Context:       q.Context,
		rawConditions: q.rawConditions,
	}

	// 深拷贝 rawConditions
	if len(q.rawConditions) > 0 {
		newBuilder.rawConditions = make([]rawUpdateCondition, len(q.rawConditions))
		copy(newBuilder.rawConditions, q.rawConditions)
	}

	return newBuilder
}

func (q *UpdateBuilder[T]) Update(query Query, values T, customFunc ...func(*gorm.DB) *gorm.DB) (bool, error) {
	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		if q.DB == nil {
			return false, WrapDBError(errors.New("db is nil"))
		}
		db = q.DB.Model(&zero)
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

func (q *UpdateBuilder[T]) UpdateByID(id interface{}, values T, customFunc ...func(*gorm.DB) *gorm.DB) (bool, error) {
	if id == nil || id == "" {
		return false, errors.New("id cannot be empty")
	}

	queryBuilder := QueryBuilder[T]{
		DB:      q.DB,
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
			if q.DB == nil {
				return false, WrapDBError(errors.New("db is nil"))
			}
			db = q.DB.Model(&zero)
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
		DB:            q.DB,
		TX:            q.TX,
		Query:         q.Query,
		Context:       q.Context,
		rawConditions: q.rawConditions,
	}

	return newBuilder.Update(query, values)
}

// UpdateBatch 批量更新多条记录，每条记录有不同的值
func (q *UpdateBuilder[T]) UpdateBatch(values []T) (int64, error) {
	if len(values) == 0 {
		return 0, errors.New("values cannot be empty")
	}

	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		if q.DB == nil {
			return 0, WrapDBError(errors.New("db is nil"))
		}
		db = q.DB.Model(&zero)
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
			return 0, WrapDBError(err)
		}
	}

	// 执行批量更新，GORM 会生成一条 SQL 语句使用 CASE WHEN
	result := db.Updates(&values)
	if result.Error != nil {
		return 0, WrapDBError(result.Error)
	}

	return result.RowsAffected, nil
}
