package base_controller

import (
	"context"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z/db"
)

type BaseController struct {
}

// Handler 返回请求处理
func (b *BaseController) Handler(c *gin.Context, spanName string, handler func(ctx context.Context) (interface{}, error)) {
	ctx, span := trace_provider.TraceProvider.Start(c.Request.Context(), spanName)
	defer span.End()

	result, err := handler(ctx)
	if err != nil {
		z.Failure(c, err)
		return
	}

	z.Success(c, result)
}

// GetQuery 从 gin.Context 中获取查询参数
func (b *BaseController) GetQuery(c *gin.Context) db.Query {
	// 优先处理便捷方案，检查是否存在便捷参数
	hasConvenienceParams := c.Query("filter") != "" ||
		c.Query("search") != "" ||
		c.Query("orderby") != "" ||
		c.Query("limit") != "" ||
		c.Query("page") != "" ||
		c.Query("include") != ""

	if hasConvenienceParams {
		return b.getQueryFromURL(c)
	}

	// 标准方案: 检查 'query' 参数
	if queryStr := c.Query("query"); queryStr != "" {
		var query db.Query
		if value, exists := c.Get("query"); exists {
			if q, ok := value.(db.Query); ok {
				query = q
			}
		}
		return query
	}

	// 默认返回空查询
	return db.Query{}
}

// getQueryFromURL 从 URL 参数中构建查询
func (b *BaseController) getQueryFromURL(c *gin.Context) db.Query {
	query := db.Query{}

	// 手动解析查询字符串
	queryParams, _ := url.ParseQuery(c.Request.URL.RawQuery)

	// 解析 filter
	if filters, ok := queryParams["filter"]; ok && len(filters) > 0 {
		query.Filter = strings.Split(filters[0], ",")
	}

	// 解析 search
	if searchStrs, ok := queryParams["search"]; ok && len(searchStrs) > 0 {
		searchStr := searchStrs[0]
		var conditions [][]interface{}
		for _, part := range strings.Split(searchStr, "|") {
			parts := strings.Split(part, ":")
			if len(parts) >= 2 {
				var value interface{}
				field := parts[0]
				value = parts[1]
				operator := "="
				if len(parts) > 2 {
					operator = parts[2]
					if strings.ToUpper(operator) == "IN" {
						value = strings.Split(z.ToString(value), ",")
					}
				}
				conditions = append(conditions, []interface{}{field, value, operator})
			}
		}
		if len(conditions) > 0 {
			query.Search = []db.ConditionGroup{{
				Conditions: conditions,
				Operator:   "AND",
			}}
		}
	}

	// 解析 orderby
	if orderByStrs, ok := queryParams["orderby"]; ok && len(orderByStrs) > 0 {
		orderByStr := orderByStrs[0]
		var orderBy [][]string
		for _, part := range strings.Split(orderByStr, "|") {
			parts := strings.Split(part, ":")
			if len(parts) == 2 {
				orderBy = append(orderBy, []string{parts[0], parts[1]})
			}
		}
		query.OrderBy = orderBy
	}

	// 解析 limit
	if limitStrs, ok := queryParams["limit"]; ok && len(limitStrs) > 0 {
		limitStr := limitStrs[0]
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = limit
		}
	}

	// 解析 page
	if pageStrs, ok := queryParams["page"]; ok && len(pageStrs) > 0 {
		pageStr := pageStrs[0]
		if page, err := strconv.Atoi(pageStr); err == nil {
			query.Page = page
		}
	}

	// 解析 include
	if includeStrs, ok := queryParams["include"]; ok && len(includeStrs) > 0 {
		includeStr := includeStrs[0]
		query.Include = strings.Split(includeStr, "|")
	}

	return query
}

// GetParamString 获取字符串类型的路径参数
func (b *BaseController) GetParamString(c *gin.Context, name string) string {
	return c.Param(name)
}

// GetParamInt 获取 int 类型的路径参数，转换失败返回 0
func (b *BaseController) GetParamInt(c *gin.Context, name string) int {
	paramValue := c.Param(name)
	if val, err := strconv.Atoi(paramValue); err == nil {
		return val
	}
	return 0
}

// GetParamInt64 获取 int64 类型的路径参数，转换失败返回 0
func (b *BaseController) GetParamInt64(c *gin.Context, name string) int64 {
	paramValue := c.Param(name)
	if val, err := strconv.ParseInt(paramValue, 10, 64); err == nil {
		return val
	}
	return 0
}

// GetParamInt32 获取 int32 类型的路径参数，转换失败返回 0
func (b *BaseController) GetParamInt32(c *gin.Context, name string) int32 {
	paramValue := c.Param(name)
	if val, err := strconv.ParseInt(paramValue, 10, 32); err == nil {
		return int32(val)
	}
	return 0
}

// GetParamUint 获取 uint 类型的路径参数，转换失败返回 0
func (b *BaseController) GetParamUint(c *gin.Context, name string) uint {
	paramValue := c.Param(name)
	if val, err := strconv.ParseUint(paramValue, 10, 0); err == nil {
		return uint(val)
	}
	return 0
}

// GetParamUint64 获取 uint64 类型的路径参数，转换失败返回 0
func (b *BaseController) GetParamUint64(c *gin.Context, name string) uint64 {
	paramValue := c.Param(name)
	if val, err := strconv.ParseUint(paramValue, 10, 64); err == nil {
		return val
	}
	return 0
}

// GetParamFloat64 获取 float64 类型的路径参数，转换失败返回 0.0
func (b *BaseController) GetParamFloat64(c *gin.Context, name string) float64 {
	paramValue := c.Param(name)
	if val, err := strconv.ParseFloat(paramValue, 64); err == nil {
		return val
	}
	return 0.0
}

// GetParamFloat32 获取 float32 类型的路径参数，转换失败返回 0.0
func (b *BaseController) GetParamFloat32(c *gin.Context, name string) float32 {
	paramValue := c.Param(name)
	if val, err := strconv.ParseFloat(paramValue, 32); err == nil {
		return float32(val)
	}
	return 0.0
}

// GetParamBool 获取 bool 类型的路径参数，转换失败返回 false
func (b *BaseController) GetParamBool(c *gin.Context, name string) bool {
	paramValue := c.Param(name)
	if val, err := strconv.ParseBool(paramValue); err == nil {
		return val
	}
	return false
}

// GetString 获取字符串类型的查询参数
func (b *BaseController) GetString(c *gin.Context, name string) string {
	return c.Query(name)
}

// GetInt 获取 int 类型的查询参数，转换失败返回 0
func (b *BaseController) GetInt(c *gin.Context, name string) int {
	queryValue := c.Query(name)
	if val, err := strconv.Atoi(queryValue); err == nil {
		return val
	}
	return 0
}

// GetInt64 获取 int64 类型的查询参数，转换失败返回 0
func (b *BaseController) GetInt64(c *gin.Context, name string) int64 {
	queryValue := c.Query(name)
	if val, err := strconv.ParseInt(queryValue, 10, 64); err == nil {
		return val
	}
	return 0
}

// GetInt32 获取 int32 类型的查询参数，转换失败返回 0
func (b *BaseController) GetInt32(c *gin.Context, name string) int32 {
	queryValue := c.Query(name)
	if val, err := strconv.ParseInt(queryValue, 10, 32); err == nil {
		return int32(val)
	}
	return 0
}

// GetUint 获取 uint 类型的查询参数，转换失败返回 0
func (b *BaseController) GetUint(c *gin.Context, name string) uint {
	queryValue := c.Query(name)
	if val, err := strconv.ParseUint(queryValue, 10, 0); err == nil {
		return uint(val)
	}
	return 0
}

// GetUint64 获取 uint64 类型的查询参数，转换失败返回 0
func (b *BaseController) GetUint64(c *gin.Context, name string) uint64 {
	queryValue := c.Query(name)
	if val, err := strconv.ParseUint(queryValue, 10, 64); err == nil {
		return val
	}
	return 0
}

// GetFloat64 获取 float64 类型的查询参数，转换失败返回 0.0
func (b *BaseController) GetFloat64(c *gin.Context, name string) float64 {
	queryValue := c.Query(name)
	if val, err := strconv.ParseFloat(queryValue, 64); err == nil {
		return val
	}
	return 0.0
}

// GetFloat32 获取 float32 类型的查询参数，转换失败返回 0.0
func (b *BaseController) GetFloat32(c *gin.Context, name string) float32 {
	queryValue := c.Query(name)
	if val, err := strconv.ParseFloat(queryValue, 32); err == nil {
		return float32(val)
	}
	return 0.0
}

// GetBool 获取 bool 类型的查询参数，转换失败返回 false
func (b *BaseController) GetBool(c *gin.Context, name string) bool {
	queryValue := c.Query(name)
	if val, err := strconv.ParseBool(queryValue); err == nil {
		return val
	}
	return false
}
