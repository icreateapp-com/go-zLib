package crud_service

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/icreateapp-com/go-zLib/z"
	"gorm.io/gorm"

	"github.com/icreateapp-com/go-zLib/z/db"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
	"github.com/icreateapp-com/go-zLib/z/service/base_service"
	"gorm.io/gorm/schema"
)

// ICrudService defines the interface for a generic CRUD service.
type ICrudService[TModel db.IModel, TCreateRequest any, TUpdateRequest any, TResponse any] interface {
	Get(query ...db.Query) ([]TResponse, error)
	Page(query db.Query) (*db.Pager, error)
	Find(id interface{}, query ...db.Query) (*TResponse, error)
	Create(req *TCreateRequest) (*TResponse, error)
	Update(id interface{}, req *TUpdateRequest) (*TResponse, error)
	Delete(query ...db.Query) (bool, error)
	DeleteByID(id interface{}, query ...db.Query) (bool, error)
}

// CrudService 是一个通用的 CRUD 服务，支持自定义请求和响应类型
type CrudService[TModel db.IModel, TCreateRequest any, TUpdateRequest any, TResponse any] struct {
	base_service.BaseService
	CreateOnly   []string                                                                                 // 创建时允许的字段
	CreateOmit   []string                                                                                 // 创建时忽略的字段
	UpdateOnly   []string                                                                                 // 更新时允许的字段
	UpdateOmit   []string                                                                                 // 更新时忽略的字段
	Unique       []string                                                                                 // 单字段唯一性(每个字段独立唯一)
	UniqueGroups [][]string                                                                               // 组合唯一性(每个数组内的字段组合唯一)
	BeforeCreate func(ctx context.Context, req *TCreateRequest, model *TModel) error                      // 创建前的钩子函数
	BeforeUpdate func(ctx context.Context, req *TUpdateRequest, model *TModel) error                      // 更新前的钩子函数
	BeforeDelete func(ctx context.Context, query db.Query) error                                          // 删除前的钩子函数
	AfterCreated func(ctx context.Context, req *TCreateRequest, model *TModel, response *TResponse) error // 创建后的钩子函数
	AfterUpdated func(ctx context.Context, req *TUpdateRequest, model *TModel, response *TResponse) error // 更新后的钩子函数
	AfterDeleted func(ctx context.Context, query db.Query, success bool) error                            // 删除后的钩子函数
	Context      context.Context
}

// Get 获取数据列表
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) Get(query ...db.Query) ([]TResponse, error) {
	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}
	var result []TModel
	if err := (&db.QueryBuilder[TModel]{Query: q}).Get(&result); err != nil {
		return nil, err
	}
	var response []TResponse
	if err := z.ToStruct(result, &response); err != nil {
		return nil, err
	}
	return response, nil
}

// Page 获取数据
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) Page(query db.Query) (*db.Pager, error) {
	var pager db.Pager
	if err := (&db.QueryBuilder[TModel]{Query: query}).Page(&pager, []TResponse{}); err != nil {
		return nil, err
	}
	return &pager, nil
}

// Find 查找数据
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) Find(id interface{}, query ...db.Query) (*TResponse, error) {
	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}
	var result TModel
	if err := (&db.QueryBuilder[TModel]{Query: q}).Find(id, &result); err != nil {
		return nil, err
	}
	var response TResponse
	if err := z.ToStruct(result, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Create 创建
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) Create(req *TCreateRequest) (*TResponse, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	var model TModel
	if err := z.ToStruct(req, &model); err != nil {
		return nil, trace_provider.TraceProvider.Error(span, err)
	}

	// 执行创建前的钩子函数
	if s.BeforeCreate != nil {
		if err := s.BeforeCreate(s.Context, req, &model); err != nil {
			return nil, trace_provider.TraceProvider.Error(span, err)
		}
	}

	// 唯一字段检查
	if err := s.checkUnique(&model); err != nil {
		return nil, err
	}
	result, err := db.CreateBuilder[TModel]{}.Create(model, func(tx *gorm.DB) *gorm.DB {
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
		return nil, trace_provider.TraceProvider.Error(span, err)
	}

	var response TResponse
	if err := z.ToStruct(result, &response); err != nil {
		return nil, trace_provider.TraceProvider.Error(span, err)
	}

	// 执行创建后的钩子函数
	if s.AfterCreated != nil {
		if err := s.AfterCreated(s.Context, req, &model, &response); err != nil {
			return nil, trace_provider.TraceProvider.Error(span, err)
		}
	}

	return &response, err
}

// Update 更新
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) Update(id interface{}, req *TUpdateRequest) (*TResponse, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	if req == nil {
		return nil, fmt.Errorf("request can not be nil")
	}

	var model TModel
	if err := z.ToStruct(req, &model); err != nil {
		return nil, trace_provider.TraceProvider.Error(span, err)
	}

	// 执行更新前的钩子函数
	if s.BeforeUpdate != nil {
		if err := s.BeforeUpdate(s.Context, req, &model); err != nil {
			return nil, trace_provider.TraceProvider.Error(span, err)
		}
	}

	// 唯一字段检查
	if err := s.checkUnique(&model, id); err != nil {
		return nil, err
	}
	_, err := db.UpdateBuilder[TModel]{}.UpdateByID(id, model, func(tx *gorm.DB) *gorm.DB {
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
		return nil, trace_provider.TraceProvider.Error(span, err)
	}

	// 获取更新后的用户信息
	updatedUser, err := s.Find(id)
	if err != nil {
		return nil, trace_provider.TraceProvider.Error(span, err)
	}

	// 执行更新后的钩子函数
	if s.AfterUpdated != nil {
		if err := s.AfterUpdated(s.Context, req, &model, updatedUser); err != nil {
			return nil, trace_provider.TraceProvider.Error(span, err)
		}
	}

	return updatedUser, nil
}

// Delete 根据查询条件删除数据
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) Delete(query ...db.Query) (bool, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}

	// 执行删除前的钩子函数
	if s.BeforeDelete != nil {
		if err := s.BeforeDelete(s.Context, q); err != nil {
			return false, trace_provider.TraceProvider.Error(span, err)
		}
	}

	res, err := db.DeleteBuilder[TModel]{}.Delete(q)

	// 记录错误
	if err != nil {
		trace_provider.TraceProvider.Error(span, err)
	}

	// 执行删除后的钩子函数
	if s.AfterDeleted != nil {
		if hookErr := s.AfterDeleted(s.Context, q, res); hookErr != nil {
			trace_provider.TraceProvider.Error(span, hookErr)
		}
	}

	return res, err
}

// DeleteByID 根据ID删除数据，支持额外的查询条件
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) DeleteByID(id interface{}, query ...db.Query) (bool, error) {
	// 开启一个新的子 span
	_, span := trace_provider.TraceProvider.Start(s.Context)
	defer span.End()

	q := db.Query{}
	if len(query) > 0 {
		q = query[0]
	}

	// 执行删除前的钩子函数
	if s.BeforeDelete != nil {
		if err := s.BeforeDelete(s.Context, q); err != nil {
			return false, trace_provider.TraceProvider.Error(span, err)
		}
	}

	var res bool
	var err error
	if len(query) > 0 {
		res, err = db.DeleteBuilder[TModel]{}.DeleteByID(id, query[0])
	} else {
		res, err = db.DeleteBuilder[TModel]{}.DeleteByID(id)
	}

	// 记录错误
	if err != nil {
		trace_provider.TraceProvider.Error(span, err)
	}

	// 执行删除后的钩子函数
	if s.AfterDeleted != nil {
		if hookErr := s.AfterDeleted(s.Context, q, res); hookErr != nil {
			trace_provider.TraceProvider.Error(span, hookErr)
		}
	}

	return res, err
}

// checkUnique 检查唯一字段（支持单字段和组合唯一性）
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) checkUnique(model *TModel, excludeID ...interface{}) error {
	// 如果没有设置唯一字段或者模型为空，则直接返回
	if (len(s.Unique) == 0 && len(s.UniqueGroups) == 0) || model == nil {
		return nil
	}

	// 使用 GORM 的 schema 解析来健壮地处理字段
	sch, err := schema.Parse(model, &sync.Map{}, db.DB.NamingStrategy)
	if err != nil {
		return fmt.Errorf("failed to parse model schema: %w", err)
	}

	modelValue := reflect.ValueOf(model).Elem()

	// 检查单字段唯一性
	for _, fieldName := range s.Unique {
		if err := s.checkSingleFieldUnique(sch, modelValue, fieldName, excludeID...); err != nil {
			return err
		}
	}

	// 检查组合唯一性
	for _, group := range s.UniqueGroups {
		if err := s.checkGroupUnique(sch, modelValue, group, excludeID...); err != nil {
			return err
		}
	}

	return nil
}

// checkSingleFieldUnique 检查单字段唯一性
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) checkSingleFieldUnique(sch *schema.Schema, modelValue reflect.Value, fieldName string, excludeID ...interface{}) error {
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
		return nil
	}

	// 获取字段的值，并检查是否为零值
	fieldValue, isZero := field.ValueOf(context.Background(), modelValue)
	if isZero {
		return nil
	}

	// 构建查询
	query := db.DB.Model(new(TModel)).Where(fmt.Sprintf("%s = ?", field.DBName), fieldValue)

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

	return nil
}

// checkGroupUnique 检查组合唯一性
func (s *CrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]) checkGroupUnique(sch *schema.Schema, modelValue reflect.Value, group []string, excludeID ...interface{}) error {
	// 构建组合唯一性查询
	query := db.DB.Model(new(TModel))
	var whereConditions []string
	var whereValues []interface{}
	var fieldNames []string

	for _, fieldName := range group {
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

		// 添加到查询条件
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", field.DBName))
		whereValues = append(whereValues, fieldValue)
		fieldNames = append(fieldNames, field.Name)
	}

	// 如果没有有效的字段值，则跳过检查
	if len(whereConditions) == 0 {
		return nil
	}

	// 逐个添加 WHERE 条件
	for i, condition := range whereConditions {
		query = query.Where(condition, whereValues[i])
	}

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
		return fmt.Errorf("combination of fields %s already exists", fmt.Sprintf("%v", fieldNames))
	}

	return nil
}
