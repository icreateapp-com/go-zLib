package crud_service

import (
	"context"
	"fmt"
	"github.com/icreateapp-com/go-zLib/z/db"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
	"github.com/icreateapp-com/go-zLib/z/service/base_service"
	"go.opentelemetry.io/otel/codes"
	"reflect"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type ICrudService[T db.IModel] interface {
	Get(query ...db.Query) ([]T, error)
	Page(query ...db.Query) (*db.Pager, error)
	Find(id interface{}, query ...db.Query) (*T, error)
	Create(model *T) (*T, error)
	Update(id interface{}, model *T) (bool, error)
	Delete(query ...db.Query) (bool, error)
	DeleteByID(id interface{}, query ...db.Query) (bool, error)
}

type CrudService[T db.IModel] struct {
	base_service.BaseService
	CreateOnly []string // 创建时允许的字段
	CreateOmit []string // 创建时忽略的字段
	UpdateOnly []string // 更新时允许的字段
	UpdateOmit []string // 更新时忽略的字段
	Unique     []string // 唯一字段(更新时要忽略更新数据的ID)
	Context    context.Context
}

// Get 获取数据列表
func (s *CrudService[T]) Get(query ...db.Query) ([]T, error) {
	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}
	var result []T
	err := (&db.QueryBuilder[T]{Query: q}).Get(&result)
	return result, err
}

// Page 获取数据
func (s *CrudService[T]) Page(query ...db.Query) (*db.Pager, error) {
	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}
	var pager db.Pager
	err := (&db.QueryBuilder[T]{Query: q}).Page(&pager)
	return &pager, err
}

// Find 查找数据
func (s *CrudService[T]) Find(id interface{}, query ...db.Query) (*T, error) {
	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}
	var result T
	err := (&db.QueryBuilder[T]{Query: q}).Find(id, &result)
	return &result, err
}

// Create 创建
func (s *CrudService[T]) Create(model *T) (*T, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	// 唯一字段检查
	if err := s.checkUnique(model); err != nil {
		return nil, err
	}
	res, err := db.CreateBuilder[T]{}.Create(*model, func(tx *gorm.DB) *gorm.DB {
		if len(s.CreateOnly) > 0 {
			return tx.Select(s.CreateOnly)
		}
		if len(s.CreateOmit) > 0 {
			return tx.Omit(s.CreateOmit...)
		}
		return tx
	})

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return &res, err
}

// Update 更新
func (s *CrudService[T]) Update(id interface{}, model *T) (bool, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	if model == nil {
		return false, fmt.Errorf("model can not be nil")
	}
	// 唯一字段检查
	if err := s.checkUnique(model, id); err != nil {
		return false, err
	}
	res, err := db.UpdateBuilder[T]{}.UpdateByID(id, *model, func(tx *gorm.DB) *gorm.DB {
		if len(s.UpdateOnly) > 0 {
			return tx.Select(s.UpdateOnly)
		}
		if len(s.UpdateOmit) > 0 {
			return tx.Omit(s.UpdateOmit...)
		}
		return tx
	})

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return res, err
}

// Delete 根据查询条件删除数据
func (s *CrudService[T]) Delete(query ...db.Query) (bool, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}
	res, err := db.DeleteBuilder[T]{}.Delete(q)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return res, err
}

// DeleteByID 根据ID删除数据，支持额外的查询条件
func (s *CrudService[T]) DeleteByID(id interface{}, query ...db.Query) (bool, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	if len(query) > 0 {
		return db.DeleteBuilder[T]{}.DeleteByID(id, query[0])
	}
	res, err := db.DeleteBuilder[T]{}.DeleteByID(id)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return res, err
}

// checkUnique 检查唯一字段
func (s *CrudService[T]) checkUnique(model *T, excludeID ...interface{}) error {
	// 如果没有设置唯一字段或者模型为空，则直接返回
	if len(s.Unique) == 0 || model == nil {
		return nil
	}

	// 使用 GORM 的 schema 解析来健壮地处理字段
	sch, err := schema.Parse(model, &sync.Map{}, db.DB.NamingStrategy)
	if err != nil {
		return fmt.Errorf("failed to parse model schema: %w", err)
	}

	modelValue := reflect.ValueOf(model).Elem()

	for _, fieldName := range s.Unique {
		// 优先按结构体字段名查找
		field := sch.LookUpField(fieldName)
		if field == nil {
			// 然后按数据库列名查找
			if f, ok := sch.FieldsByDBName[fieldName]; ok {
				field = f
			}
		}
		// 如果都找不到，说明配置有误，跳过此字段
		if field == nil {
			continue
		}

		// 获取字段的值，并检查是否为零值
		fieldValue, isZero := field.ValueOf(context.Background(), modelValue)
		if isZero {
			continue
		}

		// 构建查询
		query := db.DB.Model(new(T)).Where(fmt.Sprintf("%s = ?", field.DBName), fieldValue)

		// 如果是更新操作，则排除当前 ID
		if len(excludeID) > 0 && excludeID[0] != nil {
			// 动态获取主键的数据库列名
			if len(sch.PrimaryFields) > 0 {
				pkColumnName := sch.PrimaryFields[0].DBName
				query = query.Where(fmt.Sprintf("%s != ?", pkColumnName), excludeID[0])
			} else {
				// Fallback for models without explicit primary key tag, assuming 'id'
				query = query.Where("id != ?", excludeID[0])
			}
		}

		// 执行查询
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return err
		}

		// 如果存在重复记录，则返回错误
		if count > 0 {
			return fmt.Errorf("field '%s' with value '%v' already exists", field.Name, fieldValue)
		}
	}

	return nil
}
