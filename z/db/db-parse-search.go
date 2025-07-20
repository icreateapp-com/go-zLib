package db

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// ParseSearch 解析搜索条件
func (p QueryParser[T]) ParseSearch(db *gorm.DB, search []ConditionGroup, required []string) (*gorm.DB, error) {
	if len(search) == 0 && len(required) == 0 {
		return db, nil
	}

	// 处理必需条件
	for _, req := range required {
		if !p.isValidFieldName(req) {
			return nil, errors.New("invalid required field name: " + req)
		}
		db = db.Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", DB.F(req), DB.F(req)))
	}

	// 处理搜索条件组
	for _, group := range search {
		if len(group) == 0 {
			continue
		}

		var conditions []string
		var args []interface{}

		for _, condition := range group {
			if len(condition) < 2 {
				return nil, errors.New("invalid condition: must have at least field and value")
			}

			// 安全的类型断言
			field, ok := condition[0].(string)
			if !ok {
				return nil, errors.New("invalid condition: field must be string")
			}

			if !p.isValidFieldName(field) {
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
			if !p.isValidOperator(operator) {
				return nil, fmt.Errorf("invalid operator: '%s' is not a valid operator", operator)
			}

			conditionSQL, conditionArgs, err := p.buildCondition(field, value, operator)
			if err != nil {
				return nil, err
			}

			conditions = append(conditions, conditionSQL)
			args = append(args, conditionArgs...)
		}

		if len(conditions) > 0 {
			// 组内条件用 OR 连接
			groupCondition := "(" + strings.Join(conditions, " OR ") + ")"
			db = db.Where(groupCondition, args...)
		}
	}

	return db, nil
}

// isValidOperator 验证操作符是否有效
func (p QueryParser[T]) isValidOperator(operator string) bool {
	validOperators := map[string]bool{
		"=":           true,
		"!=":          true,
		"<>":          true,
		">":           true,
		">=":          true,
		"<":           true,
		"<=":          true,
		"LIKE":        true,
		"NOT LIKE":    true,
		"IN":          true,
		"NOT IN":      true,
		"IS NULL":     true,
		"IS NOT NULL": true,
		"BETWEEN":     true,
		"NOT BETWEEN": true,
	}
	return validOperators[strings.ToUpper(operator)]
}

// buildCondition 构建单个条件
func (p QueryParser[T]) buildCondition(field string, value interface{}, operator string) (string, []interface{}, error) {
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
