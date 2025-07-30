package db

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// ParseSearch 解析搜索条件
func ParseSearch(db *gorm.DB, search []ConditionGroup, required []string) (*gorm.DB, error) {
	if len(search) == 0 && len(required) == 0 {
		return db, nil
	}

	// 处理必需条件
	for _, req := range required {
		if !isValidFieldName(req) {
			return nil, errors.New("invalid required field name: " + req)
		}
		db = db.Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", DB.F(req), DB.F(req)))
	}

	// 检查 required 字段是否在 Search 中
	if len(required) > 0 {
		requiredFields := make(map[string]bool)
		for _, field := range required {
			requiredFields[field] = false
		}

		for _, group := range search {
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

	// 处理搜索条件组
	for _, group := range search {
		if len(group.Conditions) == 0 {
			continue
		}

		var groupConditions []string

		for _, condition := range group.Conditions {
			if len(condition) < 2 {
				return nil, errors.New("invalid condition: each condition must have at least 2 elements")
			}

			// 安全的类型断言
			field, ok := condition[0].(string)
			if !ok {
				return nil, errors.New("invalid condition: field must be string")
			}

			if !isValidFieldName(field) {
				return nil, errors.New("invalid field name: " + field)
			}

			value := condition[1]
			operator := "="
			if len(condition) > 2 {
				if op, ok := condition[2].(string); ok {
					operator = op
				}
			}

			// 验证操作符
			if !isValidOperator(operator) {
				return nil, fmt.Errorf("invalid operator: '%s' is not a valid operator", operator)
			}

			// 处理特殊的 like 操作符
			switch strings.ToLower(operator) {
			case "like":
				if str, ok := value.(string); ok && !strings.Contains(str, "%") {
					value = "%" + str + "%"
				}
			case "left like":
				if str, ok := value.(string); ok {
					value = "%" + str
					operator = "like"
				}
			case "right like":
				if str, ok := value.(string); ok {
					value = str + "%"
					operator = "like"
				}
			}

			field = DB.F(field)
			groupConditions = append(groupConditions, fmt.Sprintf("%s %s ?", field, operator))
			values = append(values, value)
		}

		if len(groupConditions) == 0 {
			continue
		}

		// 设置默认操作符
		if group.Operator == "" {
			group.Operator = "AND"
		}

		// 组内条件用指定的操作符连接
		groupClause := strings.Join(groupConditions, " "+strings.ToUpper(group.Operator)+" ")
		conditions = append(conditions, fmt.Sprintf("(%s)", groupClause))
	}

	if len(conditions) > 0 {
		// 组间条件用 AND 连接
		whereClause := strings.Join(conditions, " AND ")
		db = db.Where(whereClause, values...)
	}

	return db, nil
}

// isValidOperator 验证操作符是否有效
func isValidOperator(operator string) bool {
	validOperators := map[string]bool{
		"=":           true,
		"!=":          true,
		"<>":          true,
		">":           true,
		">=":          true,
		"<":           true,
		"<=":          true,
		"like":        true,
		"left like":   true,
		"right like":  true,
		"not like":    true,
		"LIKE":        true,
		"NOT LIKE":    true,
		"IN":          true,
		"NOT IN":      true,
		"in":          true,
		"not in":      true,
		"IS NULL":     true,
		"IS NOT NULL": true,
		"BETWEEN":     true,
		"NOT BETWEEN": true,
		"between":     true,
		"not between": true,
	}
	return validOperators[operator]
}

// buildCondition 构建单个条件
func buildCondition(field string, value interface{}, operator string) (string, []interface{}, error) {
	field = DB.F(field)
	operator = strings.ToUpper(operator)

	switch operator {
	case "IS NULL", "IS NOT NULL":
		return fmt.Sprintf("%s %s", field, operator), nil, nil

	case "IN", "NOT IN":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return "", nil, errors.New("IN/NOT IN operator requires slice value")
		}
		return fmt.Sprintf("%s %s (?)", field, operator), []interface{}{value}, nil

	case "BETWEEN", "NOT BETWEEN":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return "", nil, errors.New("BETWEEN/NOT BETWEEN operator requires slice value with 2 elements")
		}
		v := reflect.ValueOf(value)
		if v.Len() != 2 {
			return "", nil, errors.New("BETWEEN/NOT BETWEEN operator requires slice value with 2 elements")
		}
		return fmt.Sprintf("%s %s ? AND ?", field, operator), []interface{}{v.Index(0).Interface(), v.Index(1).Interface()}, nil

	case "LIKE", "NOT LIKE":
		// 自动添加通配符
		if str, ok := value.(string); ok && !strings.Contains(str, "%") {
			value = "%" + str + "%"
		}
		return fmt.Sprintf("%s %s ?", field, operator), []interface{}{value}, nil

	default:
		return fmt.Sprintf("%s %s ?", field, operator), []interface{}{value}, nil
	}
}
