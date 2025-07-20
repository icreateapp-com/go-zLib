package db

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// 默认分页配置
const (
	DefaultPage     = 1
	DefaultPageSize = 10
)

// QueryBuilder 查询构建器
type QueryBuilder[T IModel] struct {
	TX *gorm.DB
}

// Query 查询参数
type Query struct {
	Filter   []string         `json:"filter"`
	Search   []ConditionGroup `json:"search"`
	OrderBy  []string         `json:"order_by"`
	Limit    []int            `json:"limit"`
	Page     []int            `json:"page"`
	Required []string         `json:"required"`
}

// ConditionGroup 条件组
type ConditionGroup [][]interface{}

// Pager 分页信息
type Pager struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	LastPage int `json:"last_page"`
}

// PaginatedResult 分页结果
type PaginatedResult[T IModel] struct {
	Data  []T   `json:"data"`
	Pager Pager `json:"pager"`
}

// getDB 获取数据库连接（支持事务）
func (q QueryBuilder[T]) getDB() *gorm.DB {
	if q.TX != nil {
		return q.TX
	}
	return DB.DB
}

// Get 查询多条记录
func (q QueryBuilder[T]) Get(query Query) ([]T, error) {
	var results []T

	parser := QueryParser[T]{TX: q.TX}
	db := q.getDB().Model(new(T))

	parsedDB, err := parser.ParseQuery(query, db)
	if err != nil {
		return nil, err
	}

	if err := parsedDB.Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// Page 查询多条记录（带分页）
func (q QueryBuilder[T]) Page(query Query) (*PaginatedResult[T], error) {
	var results []T

	// 设置默认分页
	if len(query.Page) == 0 {
		query.Page = []int{DefaultPage, DefaultPageSize}
	}

	parser := QueryParser[T]{TX: q.TX}
	db := q.getDB().Model(new(T))

	// 先获取总数（不包含分页）
	countDB, err := parser.ParseQuery(Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}, db)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		return nil, err
	}

	// 再获取分页数据
	dataDB, err := parser.ParseQuery(query, q.getDB().Model(new(T)))
	if err != nil {
		return nil, err
	}

	if err := dataDB.Find(&results).Error; err != nil {
		return nil, err
	}

	// 计算分页信息
	page := query.Page[0]
	pageSize := query.Page[1]
	lastPage := int((total + int64(pageSize) - 1) / int64(pageSize)) // 修复分页计算
	if lastPage == 0 {
		lastPage = 1
	}

	pager := Pager{
		Page:     page,
		PageSize: pageSize,
		Total:    int(total),
		LastPage: lastPage,
	}

	return &PaginatedResult[T]{
		Data:  results,
		Pager: pager,
	}, nil
}

// First 查询单条记录
func (q QueryBuilder[T]) First(query Query) (*T, error) {
	var result T

	parser := QueryParser[T]{TX: q.TX}
	db := q.getDB().Model(new(T))

	parsedDB, err := parser.ParseQuery(query, db)
	if err != nil {
		return nil, err
	}

	if err := parsedDB.First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}

	return &result, nil
}

// Find 使用主键查找记录
func (q QueryBuilder[T]) Find(id interface{}, query Query) (*T, error) {
	var result T

	// 将 ID 条件添加到查询中
	if query.Search == nil {
		query.Search = []ConditionGroup{}
	}

	// 添加 ID 查询条件
	idCondition := ConditionGroup{{"id", id, "="}}
	query.Search = append(query.Search, idCondition)

	parser := QueryParser[T]{TX: q.TX}
	db := q.getDB().Model(new(T))

	parsedDB, err := parser.ParseQuery(query, db)
	if err != nil {
		return nil, err
	}

	if err := parsedDB.First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}

	return &result, nil
}

// Count 统计记录数量
func (q QueryBuilder[T]) Count(query Query) (int64, error) {
	parser := QueryParser[T]{TX: q.TX}
	db := q.getDB().Model(new(T))

	// 只解析搜索条件，不需要排序、分页等
	countQuery := Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}

	parsedDB, err := parser.ParseQuery(countQuery, db)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := parsedDB.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// Sum 计算字段总和
func (q QueryBuilder[T]) Sum(field string, query Query) (float64, error) {
	parser := QueryParser[T]{TX: q.TX}

	// 验证字段名安全性
	if !parser.isValidFieldName(field) {
		return 0, errors.New("invalid field name: " + field)
	}

	db := q.getDB().Model(new(T))

	// 只解析搜索条件
	sumQuery := Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}

	parsedDB, err := parser.ParseQuery(sumQuery, db)
	if err != nil {
		return 0, err
	}

	var sum float64
	if err := parsedDB.Select(fmt.Sprintf("COALESCE(SUM(%s), 0) as sum", DB.F(field))).Row().Scan(&sum); err != nil {
		return 0, err
	}

	return sum, nil
}

// Exists 检查记录是否存在
func (q QueryBuilder[T]) Exists(query Query) (bool, error) {
	count, err := q.Count(query)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ExistsById 通过主键检查记录是否存在
func (q QueryBuilder[T]) ExistsById(id interface{}) (bool, error) {
	query := Query{
		Search: []ConditionGroup{
			{{"id", id}},
		},
	}
	return q.Exists(query)
}
