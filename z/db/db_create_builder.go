package db

import (
	"context"
	"gorm.io/gorm"
)

// rawCreateCondition 原生条件
type rawCreateCondition struct {
	query string        // SQL 查询条件
	args  []interface{} // 查询参数
}

type CreateBuilder[T IModel] struct {
	TX            *gorm.DB               // 事务支持
	Context       context.Context        // 上下文
	rawConditions []rawCreateCondition   // 原生条件
}

// WithContext 设置上下文
func (q *CreateBuilder[T]) WithContext(ctx context.Context) *CreateBuilder[T] {
	newBuilder := q.clone()
	newBuilder.Context = ctx
	return newBuilder
}

// Where 添加 WHERE 条件
func (q *CreateBuilder[T]) Where(query string, args ...interface{}) *CreateBuilder[T] {
	newBuilder := q.clone()
	newBuilder.rawConditions = append(newBuilder.rawConditions, rawCreateCondition{
		query: query,
		args:  args,
	})
	return newBuilder
}

// clone 克隆 CreateBuilder 实例
func (q *CreateBuilder[T]) clone() *CreateBuilder[T] {
	newBuilder := &CreateBuilder[T]{
		TX:      q.TX,
		Context: q.Context,
	}
	
	// 深拷贝 rawConditions
	if len(q.rawConditions) > 0 {
		newBuilder.rawConditions = make([]rawCreateCondition, len(q.rawConditions))
		copy(newBuilder.rawConditions, q.rawConditions)
	}
	
	return newBuilder
}

func (q CreateBuilder[T]) Create(values T, customFunc ...func(*gorm.DB) *gorm.DB) (T, error) {
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

	// 应用自定义函数（如 Select、Omit、OnConflict 等）
	for _, fn := range customFunc {
		if fn != nil {
			db = fn(db)
		}
	}

	// 创建一个副本用于数据库操作，确保原始数据不被修改
	result := values
	if err := db.Create(&result).Error; err != nil {
		return zero, WrapDBError(err) // 使用错误包装器
	}

	// 返回包含自动生成字段（如 ID）的结果
	return result, nil
}
