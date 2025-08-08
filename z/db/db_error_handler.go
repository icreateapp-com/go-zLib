package db

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// DBError 数据库错误类型
type DBError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e DBError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// 错误代码常量
const (
	ErrCodeNotFound         = "RECORD_NOT_FOUND"
	ErrCodeDuplicate        = "DUPLICATE_ENTRY"
	ErrCodeForeignKey       = "FOREIGN_KEY_CONSTRAINT"
	ErrCodeInvalidData      = "INVALID_DATA"
	ErrCodeDatabaseError    = "DATABASE_ERROR"
	ErrCodeConstraintFailed = "CONSTRAINT_FAILED"
)

// WrapDBError 包装数据库错误为用户友好的错误消息
func WrapDBError(err error) error {
	if err == nil {
		return nil
	}

	// 处理 GORM 特定错误
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DBError{
			Code:    ErrCodeNotFound,
			Message: "Record not found",
		}
	}

	// 处理 MySQL 错误
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return handleMySQLError(mysqlErr)
	}

	// 处理字符串形式的 MySQL 错误（有时 GORM 会将错误转换为字符串）
	errStr := err.Error()
	if strings.Contains(errStr, "Error 1452") {
		return handleForeignKeyError(errStr)
	}
	if strings.Contains(errStr, "Error 1062") {
		return handleDuplicateError(errStr)
	}
	if strings.Contains(errStr, "Error 1048") {
		return handleNullConstraintError(errStr)
	}
	if strings.Contains(errStr, "Error 1406") {
		return handleDataTooLongError(errStr)
	}

	// 默认返回通用数据库错误
	return DBError{
		Code:    ErrCodeDatabaseError,
		Message: errStr,
	}
}

// handleMySQLError 处理 MySQL 特定错误
func handleMySQLError(mysqlErr *mysql.MySQLError) error {
	switch mysqlErr.Number {
	case 1452: // Foreign key constraint fails
		return handleForeignKeyError(mysqlErr.Message)
	case 1062: // Duplicate entry
		return handleDuplicateError(mysqlErr.Message)
	case 1048: // Column cannot be null
		return handleNullConstraintError(mysqlErr.Message)
	case 1406: // Data too long for column
		return handleDataTooLongError(mysqlErr.Message)
	case 1451: // Cannot delete or update a parent row
		return DBError{
			Code:    ErrCodeConstraintFailed,
			Message: "Cannot delete record because it is referenced by other records",
		}
	default:
		return DBError{
			Code:    ErrCodeDatabaseError,
			Message: "Database operation failed",
		}
	}
}

// handleForeignKeyError 处理外键约束错误
func handleForeignKeyError(errMsg string) error {
	// 正则表达式匹配外键字段名
	// 示例: FOREIGN KEY (`partner_id`) REFERENCES
	re := regexp.MustCompile(`FOREIGN KEY \(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	matches := re.FindStringSubmatch(errMsg)

	if len(matches) > 1 {
		fieldName := matches[1]
		return DBError{
			Code:    ErrCodeForeignKey,
			Message: fmt.Sprintf("Referenced %s does not exist", fieldName),
			Field:   fieldName,
		}
	}

	return DBError{
		Code:    ErrCodeForeignKey,
		Message: "Foreign key constraint violation",
	}
}

// handleDuplicateError 处理重复键错误
func handleDuplicateError(errMsg string) error {
	// 正则表达式匹配重复的键值和字段
	// 示例: Duplicate entry 'value' for key 'field_name'
	re := regexp.MustCompile(`Duplicate entry '([^']+)' for key '([^']+)'`)
	matches := re.FindStringSubmatch(errMsg)

	if len(matches) > 2 {
		value := matches[1]
		keyName := matches[2]

		// 处理主键重复
		if strings.Contains(keyName, "PRIMARY") {
			return DBError{
				Code:    ErrCodeDuplicate,
				Message: "Record with this ID already exists",
				Field:   "id",
			}
		}

		// 处理唯一键重复
		return DBError{
			Code:    ErrCodeDuplicate,
			Message: fmt.Sprintf("Value '%s' already exists", value),
			Field:   extractFieldFromKey(keyName),
		}
	}

	return DBError{
		Code:    ErrCodeDuplicate,
		Message: "Duplicate entry detected",
	}
}

// handleNullConstraintError 处理非空约束错误
func handleNullConstraintError(errMsg string) error {
	// 正则表达式匹配字段名
	// 示例: Column 'field_name' cannot be null
	re := regexp.MustCompile(`Column '([^']+)' cannot be null`)
	matches := re.FindStringSubmatch(errMsg)

	if len(matches) > 1 {
		fieldName := matches[1]
		return DBError{
			Code:    ErrCodeInvalidData,
			Message: fmt.Sprintf("Field %s is required", fieldName),
			Field:   fieldName,
		}
	}

	return DBError{
		Code:    ErrCodeInvalidData,
		Message: "Required field is missing",
	}
}

// handleDataTooLongError 处理数据过长错误
func handleDataTooLongError(errMsg string) error {
	// 正则表达式匹配字段名
	// 示例: Data too long for column 'field_name'
	re := regexp.MustCompile(`Data too long for column '([^']+)'`)
	matches := re.FindStringSubmatch(errMsg)

	if len(matches) > 1 {
		fieldName := matches[1]
		return DBError{
			Code:    ErrCodeInvalidData,
			Message: fmt.Sprintf("Data too long for field %s", fieldName),
			Field:   fieldName,
		}
	}

	return DBError{
		Code:    ErrCodeInvalidData,
		Message: "Data exceeds maximum length",
	}
}

// extractFieldFromKey 从键名中提取字段名
func extractFieldFromKey(keyName string) string {
	// 移除常见的前缀和后缀
	keyName = strings.TrimPrefix(keyName, "uk_")
	keyName = strings.TrimPrefix(keyName, "idx_")
	keyName = strings.TrimPrefix(keyName, "uniq_")

	// 如果包含表名，尝试提取字段名
	parts := strings.Split(keyName, "_")
	if len(parts) > 1 {
		// 通常最后一部分是字段名
		return parts[len(parts)-1]
	}

	return keyName
}
