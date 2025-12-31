package db

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func normalizeOperator(operator string) string {
	op := strings.TrimSpace(operator)
	if op == "" {
		return "="
	}

	// URL 传参通常不便携带空格，允许用下划线表达
	// 例如：not_like / left_like / is_not_null / not_in / not_between
	op = strings.ReplaceAll(op, "_", " ")
	// 合并连续空格
	op = strings.Join(strings.Fields(op), " ")

	// 统一为小写，后续再按需要输出大写（比如 IN/IS NULL）
	opLower := strings.ToLower(op)
	return opLower
}

// ParseSearch 解析搜索条件
func ParseSearch(db *gorm.DB, search []ConditionGroup, required []string) (*gorm.DB, error) {
	if len(search) == 0 && len(required) == 0 {
		return db, nil
	}

	// 允许空字符串参与查询条件的字段集合（例如 id = '' 需要生效）
	allowEmptyStringFields := map[string]bool{
		"id": true,
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
			operator = normalizeOperator(operator)

			// 如果操作符不是 IS NULL 或 IS NOT NULL，且值为 nil 或空字符串，则跳过该条件
			if operator != "is null" && operator != "is not null" {
				if value == nil {
					continue
				}
				if s, ok := value.(string); ok && s == "" {
					// 部分字段（例如主键 id）空字符串是允许的，需要生成明确条件，避免条件缺失导致误查询
					if !allowEmptyStringFields[field] {
						continue
					}
				}
			}

			// 验证操作符
			if !isValidOperator(operator) {
				return nil, fmt.Errorf("invalid operator: '%s' is not a valid operator", operator)
			}

			// 处理特殊的 like 操作符
			switch operator {
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

			// 输出到 SQL 时，对部分操作符做标准化大写
			switch operator {
			case "in", "not in", "is null", "is not null", "between", "not between":
				operator = strings.ToUpper(operator)
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
		"in":          true,
		"not in":      true,
		"is null":     true,
		"is not null": true,
		"between":     true,
		"not between": true,
	}
	return validOperators[operator]
}
