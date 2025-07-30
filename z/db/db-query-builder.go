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
type QueryBuilder[T any] struct {
	TX    *gorm.DB    // 事务支持
	Query Query       // 查询参数
	Model interface{} // 显式设置查询模型
}

// SetModel 设置查询模型
func (q *QueryBuilder[T]) SetModel(model interface{}) *QueryBuilder[T] {
	q.Model = model
	return q
}

// Query 查询参数
type Query struct {
	Filter   []string         `json:"filter"`
	Search   []ConditionGroup `json:"search"`
	OrderBy  [][]string       `json:"orderby"`
	Limit    int              `json:"limit"`
	Page     int              `json:"page"`
	Required []string         `json:"required"`
	Include  []string         `json:"include"`
}

// ConditionGroup 条件组
type ConditionGroup struct {
	Conditions [][]interface{} `json:"conditions"`
	Operator   string          `json:"operator"`
}

// Pager 分页信息
type Pager struct {
	CurrentPage int `json:"current_page"` // 当前页码
	Total       int `json:"total"`        // 总记录数
	LastPage    int `json:"last_page"`    // 最后一页
	Data        any `json:"data"`         // 分页数据
}

// getDB 获取数据库连接（支持事务）
func (q *QueryBuilder[T]) getDB() *gorm.DB {
	if q.TX != nil {
		return q.TX
	}
	return DB.DB
}

// getDBWithModel 获取带模型的数据库连接
func (q *QueryBuilder[T]) getDBWithModel() *gorm.DB {
	db := q.getDB()
	model := q.Model
	if model == nil {
		model = new(T)
	}
	return db.Model(model)
}

// Get 查询多条记录
func (q *QueryBuilder[T]) Get(dest interface{}) error {
	query := q.Query

	db := q.getDBWithModel()

	parsedDB, err := ParseQuery(query, db)
	if err != nil {
		return err
	}

	return parsedDB.Find(dest).Error
}

// Page 查询多条记录（带分页）
func (q *QueryBuilder[T]) Page(pager *Pager) error {
	query := q.Query

	// 如果未设置分页，则使用默认值
	if query.Page <= 0 {
		query.Page = DefaultPage
	}
	if query.Limit <= 0 {
		query.Limit = DefaultPageSize
	}

	db := q.getDBWithModel()

	// 先获取总数（不包含分页和关联数据）
	countDB, err := ParseQuery(Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}, db)
	if err != nil {
		return err
	}

	var total int64
	if err := countDB.Count(&total).Error; err != nil {
		return err
	}

	// 再获取分页数据
	dataDB, err := ParseQuery(query, q.getDBWithModel())
	if err != nil {
		return err
	}

	// 创建一个用于接收数据的切片
	var data []T
	if err := dataDB.Find(&data).Error; err != nil {
		return err
	}
	pager.Data = data

	// 计算分页信息
	limit := query.Limit
	if limit <= 0 {
		limit = DefaultPageSize
	}

	lastPage := int((total + int64(limit) - 1) / int64(limit))
	if lastPage == 0 {
		lastPage = 1
	}

	// 设置分页信息
	pager.CurrentPage = query.Page
	pager.Total = int(total)
	pager.LastPage = lastPage

	return nil
}

// First 查询单条记录
func (q *QueryBuilder[T]) First(dest interface{}) error {
	query := q.Query

	db := q.getDBWithModel()

	parsedDB, err := ParseQuery(query, db)
	if err != nil {
		return err
	}

	return parsedDB.First(dest).Error
}

// Find 使用主键查找记录
func (q *QueryBuilder[T]) Find(id interface{}, dest interface{}) error {
	// 复制一份新的 query
	newQuery := q.Query
	newQuery.Search = append([]ConditionGroup{}, q.Query.Search...)

	// 将 ID 条件添加到查询中
	if newQuery.Search == nil {
		newQuery.Search = []ConditionGroup{}
	}
	// 添加 ID 查询条件
	idCondition := ConditionGroup{
		Conditions: [][]interface{}{{"id", id, "="}},
	}
	newQuery.Search = append(newQuery.Search, idCondition)

	return (&QueryBuilder[T]{TX: q.TX, Query: newQuery, Model: q.Model}).First(dest)
}

// Count 统计记录数量
func (q *QueryBuilder[T]) Count() (int64, error) {
	query := q.Query
	db := q.getDBWithModel()

	// 只解析搜索条件，不需要排序、分页等
	countQuery := Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}

	parsedDB, err := ParseQuery(countQuery, db)
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
func (q *QueryBuilder[T]) Sum(field string) (float64, error) {
	query := q.Query

	// 验证字段名安全性
	if !isValidFieldName(field) {
		return 0, errors.New("invalid field name: " + field)
	}

	db := q.getDBWithModel()

	// 只解析搜索条件
	sumQuery := Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}

	parsedDB, err := ParseQuery(sumQuery, db)
	if err != nil {
		return 0, err
	}

	var sum float64
	if err := parsedDB.Select(fmt.Sprintf("COALESCE(SUM(%s), 0) as sum", DB.F(field))).Row().Scan(&sum); err != nil {
		return 0, err
	}

	return sum, nil
}

// Avg 计算字段平均值
func (q *QueryBuilder[T]) Avg(field string) (float64, error) {
	query := q.Query

	// 验证字段名安全性
	if !isValidFieldName(field) {
		return 0, errors.New("invalid field name: " + field)
	}

	db := q.getDBWithModel()

	// 只解析搜索条件
	avgQuery := Query{
		Filter:   query.Filter,
		Search:   query.Search,
		Required: query.Required,
	}

	parsedDB, err := ParseQuery(avgQuery, db)
	if err != nil {
		return 0, err
	}

	var avg float64
	if err := parsedDB.Select(fmt.Sprintf("COALESCE(AVG(%s), 0) as avg", DB.F(field))).Row().Scan(&avg); err != nil {
		return 0, err
	}

	return avg, nil
}

// Exists 检查记录是否存在
func (q *QueryBuilder[T]) Exists() (bool, error) {
	count, err := q.Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ExistsById 通过主键检查记录是否存在
func (q *QueryBuilder[T]) ExistsById(id interface{}) (bool, error) {
	query := Query{
		Search: []ConditionGroup{
			{
				Conditions: [][]interface{}{{"id", id}},
			},
		},
	}
	return (&QueryBuilder[T]{TX: q.TX, Query: query, Model: q.Model}).Exists()
}
