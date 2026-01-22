package db_provider

// Query 查询参数
type Query struct {
	Search   []ConditionGroup `json:"search"`
	OrderBy  [][]string       `json:"orderby"`
	Limit    int              `json:"limit"`
	Page     int              `json:"page"`
	Required []string         `json:"required"`
}

// ConditionGroup 条件组
type ConditionGroup struct {
	Conditions [][]interface{} `json:"conditions"`
	Operator   string          `json:"operator"`
}

// AddSearch 添加搜索条件
// field: 字段名
// value: 字段值
// operator: 操作符，默认为 "="
// 支持的操作符: =, !=, >, <, >=, <=, like, not_like, in, not_in, between, not_between, is_null, is_not_null
func (q *Query) AddSearch(field string, value interface{}, operator ...string) *Query {
	if q.Search == nil {
		q.Search = []ConditionGroup{}
	}

	op := "="
	if len(operator) > 0 && operator[0] != "" {
		op = operator[0]
	}

	// 创建新的条件组
	condition := ConditionGroup{
		Conditions: [][]interface{}{{field, value, op}},
		Operator:   "AND",
	}

	q.Search = append(q.Search, condition)
	return q
}

// AddSearchGroup 添加条件组（支持 OR 等复杂逻辑）
// operator: 组内操作符，默认为 "AND"
func (q *Query) AddSearchGroup(operator string, conditions ...[]interface{}) *Query {
	if q.Search == nil {
		q.Search = []ConditionGroup{}
	}

	if operator == "" {
		operator = "AND"
	}

	group := ConditionGroup{
		Conditions: conditions,
		Operator:   operator,
	}

	q.Search = append(q.Search, group)
	return q
}

// AddOrderBy 添加排序
// field: 字段名
// direction: 排序方向，"asc" 或 "desc"，默认为 "asc"
func (q *Query) AddOrderBy(field string, direction ...string) *Query {
	if q.OrderBy == nil {
		q.OrderBy = [][]string{}
	}

	dir := "asc"
	if len(direction) > 0 && direction[0] != "" {
		dir = direction[0]
	}

	q.OrderBy = append(q.OrderBy, []string{field, dir})
	return q
}

// AddOrderByAsc 添加升序排序
func (q *Query) AddOrderByAsc(field string) *Query {
	return q.AddOrderBy(field, "asc")
}

// AddOrderByDesc 添加降序排序
func (q *Query) AddOrderByDesc(field string) *Query {
	return q.AddOrderBy(field, "desc")
}

// SetPage 设置页码
func (q *Query) SetPage(page int) *Query {
	q.Page = page
	return q
}

// SetLimit 设置每页数量
func (q *Query) SetLimit(limit int) *Query {
	q.Limit = limit
	return q
}

// SetPagination 设置分页（页码和每页数量）
func (q *Query) SetPagination(page, limit int) *Query {
	q.Page = page
	q.Limit = limit
	return q
}

// AddRequired 添加必填字段
func (q *Query) AddRequired(fields ...string) *Query {
	if q.Required == nil {
		q.Required = []string{}
	}

	q.Required = append(q.Required, fields...)
	return q
}

// Clone 克隆 Query 实例
func (q *Query) Clone() Query {
	clone := Query{
		Limit:    q.Limit,
		Page:     q.Page,
		Required: make([]string, len(q.Required)),
	}

	// 深拷贝 Required
	copy(clone.Required, q.Required)

	// 深拷贝 Search
	if len(q.Search) > 0 {
		clone.Search = make([]ConditionGroup, len(q.Search))
		for i, group := range q.Search {
			clone.Search[i] = ConditionGroup{
				Conditions: make([][]interface{}, len(group.Conditions)),
				Operator:   group.Operator,
			}
			for j, condition := range group.Conditions {
				clone.Search[i].Conditions[j] = make([]interface{}, len(condition))
				copy(clone.Search[i].Conditions[j], condition)
			}
		}
	}

	// 深拷贝 OrderBy
	if len(q.OrderBy) > 0 {
		clone.OrderBy = make([][]string, len(q.OrderBy))
		for i, order := range q.OrderBy {
			clone.OrderBy[i] = make([]string, len(order))
			copy(clone.OrderBy[i], order)
		}
	}

	return clone
}
