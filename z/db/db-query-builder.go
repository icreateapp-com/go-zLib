package db

import (
	"errors"
	"fmt"
	"github.com/icreateapp-com/go-zLib/z"
	"gorm.io/gorm"
	"strings"
)

type ConditionGroup struct {
	Conditions [][]interface{} `json:"conditions"`
	Operator   string          `json:"operator"`
}

type PageInfo struct {
	Total       int                      `json:"total"`
	CurrentPage int                      `json:"current_page"`
	LastPage    int                      `json:"last_page"`
	Data        []map[string]interface{} `json:"data"`
}

type Query struct {
	Filter   []string         `json:"filter"`
	Search   []ConditionGroup `json:"search"`
	OrderBy  [][]string       `json:"orderby"`
	Limit    []int            `json:"limit"`
	Page     []int            `json:"page"`
	Include  []string         `json:"include"`
	Required []string         `json:"-"`
}

type QueryBuilder struct {
	Model interface{}
}

func (q QueryBuilder) Get(query Query) ([]map[string]interface{}, error) {
	// define result
	var ResultFields []map[string]interface{}

	// db
	db, err := q.ParseQuery(query, &ResultFields)
	if err != nil {
		return nil, err
	}

	// find rows
	if err := db.Find(&ResultFields).Error; err != nil {
		return nil, err
	}

	// process time fields
	for i, _ := range ResultFields {
		z.FormatTimeInMap(ResultFields[i])
	}

	return ResultFields, nil
}

func (q QueryBuilder) Page(query Query) (PageInfo, error) {
	pager := PageInfo{
		Total:       0,
		CurrentPage: 1,
		LastPage:    1,
		Data:        []map[string]interface{}{},
	}

	// set page info
	if len(query.Page) == 0 {
		query.Page = []int{1, 10}
	} else if len(query.Page) == 1 {
		query.Page = append(query.Page, 10)
	}

	// define result
	var ResultFields []map[string]interface{}

	// db
	db, err := q.ParseQuery(query, &ResultFields)
	if err != nil {
		return pager, err
	}

	// get total count
	count, err := QueryBuilder{Model: q.Model}.Count(query.Search)
	if err != nil {
		return pager, err
	}

	// get data
	if err := db.Find(&ResultFields).Error; err != nil {
		return pager, err
	}

	// process time fields
	for i, _ := range ResultFields {
		z.FormatTimeInMap(ResultFields[i])
	}

	// calculate page info
	pager.Total = count
	pager.CurrentPage = query.Page[0]
	pager.LastPage = count/query.Page[1] + 1
	pager.Data = ResultFields

	return pager, nil
}

func (q QueryBuilder) Find(query Query) (map[string]interface{}, error) {
	query.Limit = []int{0, 1}

	rows, err := q.Get(query)
	if err != nil {
		return nil, err
	}

	if len(rows) > 0 {
		return rows[0], nil
	} else {
		return nil, nil
	}
}

func (q QueryBuilder) FindById(id interface{}, query Query) (map[string]interface{}, error) {
	// set id search
	query.Search = append(query.Search, ConditionGroup{
		Conditions: [][]interface{}{{"id", id}},
	})

	return q.Find(query)
}

func (q QueryBuilder) Count(search []ConditionGroup) (int, error) {

	// define result
	var ResultFields []map[string]interface{}

	db, err := q.ParseQuery(Query{Search: search}, &ResultFields)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}

	return int(count), nil
}

func (q QueryBuilder) SUM(field string, search []ConditionGroup) (float64, error) {

	// define result
	var ResultFields []map[string]interface{}

	db, err := q.ParseQuery(Query{Search: search}, &ResultFields)
	if err != nil {
		return 0, err
	}

	var sum float64
	if err := db.Select(fmt.Sprintf("SUM(%s) as sum", DB.F(field))).Row().Scan(&sum); err != nil {
		return 0, err
	}

	return sum, nil
}

func (q QueryBuilder) Exists(search []ConditionGroup) (bool, error) {
	count, err := q.Count(search)
	if err != nil {
		return false, err
	}

	return count > 0, err
}

func (q QueryBuilder) ExistsById(id interface{}) (bool, error) {
	return q.Exists([]ConditionGroup{{Conditions: [][]interface{}{{"id", id}}}})
}

func (q QueryBuilder) ParseQuery(query Query, ResultFields *[]map[string]interface{}) (*gorm.DB, error) {
	// db
	db := DB.Model(q.Model)

	// parse filter fields
	db, rows, err := q.ParseFilter(db, query.Filter)
	if err != nil {
		return nil, err
	}
	ResultFields = &rows

	// parse where clause
	if db, err = q.ParseSearch(db, query.Search, query.Required); err != nil {
		return nil, err
	}

	// parse order by
	if db, err = q.ParseOrderBy(db, query.OrderBy); err != nil {
		return nil, err
	}

	// parse limit
	if db, err = q.ParseLimit(db, query.Limit); err != nil {
		return nil, err
	}

	// parse page
	if db, err = q.ParsePage(db, query.Page); err != nil {
		return nil, err
	}

	return db, nil
}

func (q QueryBuilder) ParseFilter(db *gorm.DB, filter []string) (*gorm.DB, []map[string]interface{}, error) {
	var rows []map[string]interface{}
	var selectFields []string

	if len(filter) == 0 {
		selectFields = []string{"*"}
	} else {
		for _, f := range filter {
			f = DB.F(f)
			selectFields = append(selectFields, f)
			for i, _ := range rows {
				rows[i][f] = nil
			}
		}
	}

	db = db.Select(strings.Join(selectFields, ", "))

	return db, rows, nil
}

func (q QueryBuilder) ParseSearch(db *gorm.DB, groups []ConditionGroup, required []string) (*gorm.DB, error) {
	// 检查 required 字段是否在 Search 中
	if len(required) > 0 {
		requiredFields := make(map[string]bool)
		for _, field := range required {
			requiredFields[field] = false
		}

		for _, group := range groups {
			for _, condition := range group.Conditions {
				if len(condition) < 2 {
					return nil, errors.New("invalid condition: each condition must have at least 2 elements")
				}

				field := condition[0].(string)
				if _, exists := requiredFields[field]; exists {
					requiredFields[field] = true
				}
			}
		}

		for field, found := range requiredFields {
			if !found {
				return nil, fmt.Errorf("required field '%s' is missing in search conditions", field)
			}
		}
	}

	var conditions []string
	var values []interface{}

	for _, group := range groups {
		var groupConditions []string

		for _, condition := range group.Conditions {
			if len(condition) < 2 {
				return nil, errors.New("invalid condition: each condition must have at least 2 elements")
			}

			field := condition[0].(string)
			value := condition[1]

			var operator string
			if len(condition) == 3 {
				operator = condition[2].(string)
			} else {
				operator = "="
			}

			// Validate operator
			validOperators := map[string]bool{
				"=":           true,
				"!=":          true,
				">":           true,
				"<":           true,
				">=":          true,
				"<=":          true,
				"like":        true,
				"left like":   true,
				"right like":  true,
				"not like":    true,
				"in":          true,
				"not in":      true,
				"between":     true,
				"not between": true,
			}
			if !validOperators[operator] {
				return nil, errors.New(fmt.Sprintf("invalid operator: '%s' is not a valid operator", operator))
			}

			// Handle like operators
			switch operator {
			case "like":
				value = "%" + value.(string) + "%"
				operator = "like"
			case "left like":
				value = "%" + value.(string)
				operator = "like"
			case "right like":
				value = value.(string) + "%"
				operator = "like"
			}

			field = DB.F(field)

			groupConditions = append(groupConditions, fmt.Sprintf("%s %s ?", field, operator))
			values = append(values, value)
		}

		if len(groupConditions) == 0 {
			continue
		}

		if group.Operator == "" {
			group.Operator = "AND"
		}

		// Combine conditions within the group
		groupClause := strings.Join(groupConditions, " "+group.Operator+" ")
		conditions = append(conditions, fmt.Sprintf("(%s)", groupClause))
	}

	if len(conditions) == 0 {
		return db, nil
	}

	// Combine all group conditions with AND
	whereClause := strings.Join(conditions, " AND ")

	db = db.Where(whereClause, values...)

	return db, nil
}

func (q QueryBuilder) ParseOrderBy(db *gorm.DB, order [][]string) (*gorm.DB, error) {
	if len(order) == 0 && z.HasField(q.Model, "CreatedAt") {
		order = [][]string{{"created_at", "asc"}}
	}
	for _, o := range order {
		if len(o) == 1 {
			// If only one element, use "asc" as the default direction
			o = append(o, "asc")
		} else if len(o) != 2 {
			return nil, errors.New("invalid order condition: each order condition must have exactly 1 or 2 elements")
		}

		field := DB.F(o[0])
		direction := o[1]

		// Validate direction
		validDirections := map[string]bool{"asc": true, "desc": true}
		if !validDirections[direction] {
			return nil, errors.New(fmt.Sprintf("invalid order direction: '%s' is not a valid direction", direction))
		}

		// Generate order clause
		orderClause := fmt.Sprintf("%s %s", field, direction)
		db = db.Order(orderClause)
	}

	return db, nil
}

func (q QueryBuilder) ParseLimit(db *gorm.DB, limit []int) (*gorm.DB, error) {
	if len(limit) == 2 {
		db = db.Offset(limit[0]).Limit(limit[1])
	} else if len(limit) == 1 {
		db = db.Limit(limit[0])
	} else if len(limit) > 2 {
		return nil, errors.New("limit must have at most 2 elements")
	}

	return db, nil
}

func (q QueryBuilder) ParsePage(db *gorm.DB, page []int) (*gorm.DB, error) {
	if len(page) != 2 {
		return db, nil
	}

	offset := (page[0] - 1) * page[1]
	limit := page[1]

	db = db.Offset(offset).Limit(limit)

	return db, nil
}
