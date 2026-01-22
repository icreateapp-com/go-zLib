package helpers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"

	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/providers/auth_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/db_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/trace_provider"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// Validator 数据验证器
type Validator struct {
	trans ut.Translator
}

// Init 初始化验证器
func (v *Validator) Init() error {
	zhTrans := zh.New()
	enTrans := en.New()
	uni := ut.New(enTrans, zhTrans)

	trans, _ := uni.GetTranslator("en")
	if trans == nil {
		return errors.New("validator translator is nil")
	}

	engine := binding.Validator.Engine()
	validate, ok := engine.(*validator.Validate)
	if !ok || validate == nil {
		return errors.New("binding validator engine is not *validator.Validate")
	}

	if err := entranslations.RegisterDefaultTranslations(validate, trans); err != nil {
		return err
	}

	v.trans = trans
	return nil
}

// T 翻译验证错误消息
func (v *Validator) T(err error, req interface{}) string {
	if io.EOF == err {
		return "request body is empty"
	}

	labels := map[string]string{}

	t := reflect.TypeOf(req)
	if t == nil {
		return err.Error()
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return err.Error()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		label := field.Tag.Get("label")
		if len(label) == 0 {
			label = field.Tag.Get("json")
			if idx := strings.Index(label, ","); idx >= 0 {
				label = label[:idx]
			}
		}
		labels[field.Name] = label
	}

	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		if len(errs) > 0 {
			msg := errs[0].Error()
			if v != nil && v.trans != nil {
				msg = errs[0].Translate(v.trans)
			}

			field := errs[0].Field()
			label := labels[field]
			if len(label) < 1 {
				label = strings.ToLower(field)
			}
			return strings.Replace(msg, field, label, -1)
		}
	}

	return err.Error()
}

type BaseController struct {
	Trace     *trace_provider.Trace
	Log       *logger_provider.Logger
	Auth      *auth_provider.Auth
	Validator *Validator
}

type TraceIn struct {
	fx.In

	Trace     *trace_provider.Trace   `optional:"true"`
	Log       *logger_provider.Logger `optional:"true"`
	Auth      *auth_provider.Auth     `optional:"true"`
	Validator *Validator              `optional:"true"`
}

func NewBaseController(in TraceIn) *BaseController {
	return &BaseController{Trace: in.Trace, Log: in.Log, Auth: in.Auth, Validator: in.Validator}
}

var BaseControllerModule = fx.Options(
	fx.Provide(NewBaseController),
)

// Handler 返回请求处理
func (b *BaseController) Handler(c *gin.Context, spanName string, handler func(ctx context.Context) (interface{}, error)) {
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if b != nil && b.Trace != nil {
		ctx, span = b.Trace.Start(ctx, spanName)
		defer span.End()
	}

	result, err := handler(ctx)
	if err != nil {
		z.Failure(c, err)
		return
	}

	z.Success(c, result)
}

func (b *BaseController) GetUserID(c *gin.Context) (string, error) {
	if c == nil {
		return "", fmt.Errorf("context is nil")
	}
	if v, ok := c.Get("auth.user_id"); ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s), nil
		}
	}
	return "", fmt.Errorf("user not found")
}

func (b *BaseController) GetUserData(c *gin.Context) (interface{}, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if v, ok := c.Get("auth.data"); ok {
		return v, nil
	}
	return nil, nil
}

// StreamHandler 统一流式处理（SSE）：
// - 从 gin.Context 提取 request context
// - 可选创建 trace span
// - 初始化 StreamSender 并交给业务回调写入
// - 发生错误时使用 z.Failure 输出
func (b *BaseController) StreamHandler(c *gin.Context, spanName string, handler func(ctx context.Context, s *z.StreamSender) error) {
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if b != nil && b.Trace != nil {
		ctx, span = b.Trace.Start(ctx, spanName)
		defer span.End()
	}

	s := z.NewStreamSender(c)
	defer func() { s.Done() }()

	if err := handler(ctx, s); err != nil {
		z.Failure(c, err)
		return
	}
}

// GetQuery 从 gin.Context 中获取查询参数
func (b *BaseController) GetQuery(c *gin.Context) db_provider.Query {
	// 标准方案（优先）：如果上游（middleware）已解析并写入 context，则直接使用
	if value, exists := c.Get("query"); exists {
		if q, ok := value.(db_provider.Query); ok {
			return q
		}
	}

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

	// 默认返回空查询
	return db_provider.Query{}
}

// getQueryFromURL 从 URL 参数中构建查询
func (b *BaseController) getQueryFromURL(c *gin.Context) db_provider.Query {
	query := &db_provider.Query{}

	// 手动解析查询字符串
	queryParams, _ := url.ParseQuery(c.Request.URL.RawQuery)

	// 解析 filter
	if filters, ok := queryParams["filter"]; ok && len(filters) > 0 {
		query.AddRequired(strings.Split(filters[0], ",")...)
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
			query.AddSearchGroup("AND", conditions...)
		}
	}

	// 解析 orderby
	if orderByStrs, ok := queryParams["orderby"]; ok && len(orderByStrs) > 0 {
		orderByStr := orderByStrs[0]
		for _, part := range strings.Split(orderByStr, "|") {
			parts := strings.Split(part, ":")
			if len(parts) == 2 {
				query.AddOrderBy(parts[0], parts[1])
			}
		}
	}

	// 解析 limit
	if limitStrs, ok := queryParams["limit"]; ok && len(limitStrs) > 0 {
		limitStr := limitStrs[0]
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.SetLimit(limit)
		}
	}

	// 解析 page
	if pageStrs, ok := queryParams["page"]; ok && len(pageStrs) > 0 {
		pageStr := pageStrs[0]
		if page, err := strconv.Atoi(pageStr); err == nil {
			query.SetPage(page)
		}
	}

	return *query
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

// T 翻译校验错误
func (b *BaseController) T(err error, req interface{}) string {
	if b == nil || b.Validator == nil {
		if err == nil {
			return ""
		}
		return err.Error()
	}
	return b.Validator.T(err, req)
}
