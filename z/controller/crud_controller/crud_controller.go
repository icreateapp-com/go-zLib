package crud_controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/controller/base_controller"
	"github.com/icreateapp-com/go-zLib/z/db"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
	"github.com/icreateapp-com/go-zLib/z/service/crud_service"
)

// ICrudController CRUD控制器接口
type ICrudController[T db.IModel] interface {
	Get(c *gin.Context)
	Find(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

// ServiceFactory 服务工厂函数类型
type ServiceFactory[T db.IModel, S crud_service.ICrudService[T]] func(ctx context.Context) S

// CrudController 通用CRUD控制器，支持自定义请求和响应结构体
type CrudController[T db.IModel, S crud_service.ICrudService[T], R any, O any] struct {
	base_controller.BaseController
	serviceFactory   ServiceFactory[T, S]                        // 服务工厂函数
	createValuesFunc func(c *gin.Context) map[string]interface{} // 创建时覆盖值的回调函数
	updateValuesFunc func(c *gin.Context) map[string]interface{} // 更新时覆盖值的回调函数
	Context          *gin.Context
}

// SetCreateValues 设置创建时的覆盖值回调函数
func (ctrl *CrudController[T, S, R, O]) SetCreateValues(fn func(c *gin.Context) map[string]interface{}) {
	ctrl.createValuesFunc = fn
}

// SetUpdateValues 设置更新时的覆盖值回调函数
func (ctrl *CrudController[T, S, R, O]) SetUpdateValues(fn func(c *gin.Context) map[string]interface{}) {
	ctrl.updateValuesFunc = fn
}

// New 创建新的CRUD控制器
func New[T db.IModel, S crud_service.ICrudService[T], R any, O any](factory ServiceFactory[T, S]) *CrudController[T, S, R, O] {
	return &CrudController[T, S, R, O]{
		serviceFactory: factory,
	}
}

// Get 获取数据列表
func (ctrl *CrudController[T, S, R, O]) Get(c *gin.Context) {
	ctrl.Context = c
	query := ctrl.GetQuery(c)

	service := ctrl.serviceFactory(c.Request.Context())
	if res, err := service.Page(query); err != nil {
		z.Failure(c, err)
	} else {
		// 处理分页数据转换
		if res != nil && res.Data != nil {
			// 转换分页数据中的 Data 字段
			convertedData := ctrl.convertModelToResponse(res.Data)
			// 创建新的分页对象，保持分页信息不变，只转换数据部分
			response := &db.Pager{
				CurrentPage: res.CurrentPage,
				Total:       res.Total,
				LastPage:    res.LastPage,
				Data:        convertedData,
			}
			z.Success(c, response)
		} else {
			z.Success(c, res)
		}
	}
}

// Find 获取单个数据
func (ctrl *CrudController[T, S, R, O]) Find(c *gin.Context) {
	ctrl.Context = c
	id := c.Param("id")
	query := ctrl.GetQuery(c)

	service := ctrl.serviceFactory(c.Request.Context())
	if res, err := service.Find(id, query); err != nil {
		z.Failure(c, err)
	} else {
		// 转换响应数据
		response := ctrl.convertModelToResponse(res)
		z.Success(c, response)
	}
}

// Create 创建数据
func (ctrl *CrudController[T, S, R, O]) Create(c *gin.Context) {
	ctrl.Context = c
	ctx, span := trace_provider.TraceProvider.Start(c.Request.Context())
	defer span.End()

	// 获取创建时的覆盖值
	var overrideValues map[string]interface{}
	if ctrl.createValuesFunc != nil {
		overrideValues = ctrl.createValuesFunc(c)
	}

	// 绑定并转换数据
	model, err := ctrl.bindAndConvert(c, overrideValues)
	if err != nil {
		trace_provider.TraceProvider.Error(ctx, span, err)
		z.Failure(c, err)
		return
	}

	// 创建数据
	service := ctrl.serviceFactory(c.Request.Context())
	res, err := service.Create(&model)
	if err != nil {
		trace_provider.TraceProvider.Error(ctx, span, err)
		z.Failure(c, err)
	} else {
		// 转换响应数据
		response := ctrl.convertModelToResponse(res)
		z.Success(c, response)
	}
}

// Update 更新数据
func (ctrl *CrudController[T, S, R, O]) Update(c *gin.Context) {
	ctrl.Context = c
	id := c.Param("id")

	// 获取更新时的覆盖值
	var overrideValues map[string]interface{}
	if ctrl.updateValuesFunc != nil {
		overrideValues = ctrl.updateValuesFunc(c)
	}

	// 绑定并转换数据
	model, err := ctrl.bindAndConvert(c, overrideValues)
	if err != nil {
		z.Failure(c, err)
		return
	}

	// 更新数据
	service := ctrl.serviceFactory(c.Request.Context())
	if _, err := service.Update(id, &model); err != nil {
		z.Failure(c, err)
	} else {
		z.Success(c)
	}
}

// Delete 删除数据
func (ctrl *CrudController[T, S, R, O]) Delete(c *gin.Context) {
	ctrl.Context = c
	id := c.Param("id")

	service := ctrl.serviceFactory(c.Request.Context())
	if _, err := service.DeleteByID(id); err != nil {
		z.Failure(c, err)
	} else {
		z.Success(c)
	}
}

// bindAndConvert 绑定请求数据并转换为模型
func (ctrl *CrudController[T, S, R, O]) bindAndConvert(c *gin.Context, overrideValues map[string]interface{}) (T, error) {
	var model T
	var zeroR R

	// 判断R是否为nil或与T类型一致，决定是否需要转换
	rType := reflect.TypeOf(zeroR)
	if rType == nil {
		// R为nil，直接绑定到T类型
		if err := c.ShouldBindJSON(&model); err != nil {
			return model, fmt.Errorf(z.Validator.T(err, model))
		}
	} else {
		// 检查R和T是否为同一类型
		tType := reflect.TypeOf(model)

		// 处理指针类型比较
		rType = ctrl.getActualType(rType)
		tType = ctrl.getActualType(tType)

		if rType == tType {
			// R和T是同一类型，直接绑定到T类型，跳过转换
			if err := c.ShouldBindJSON(&model); err != nil {
				return model, fmt.Errorf(z.Validator.T(err, model))
			}
		} else {
			// R和T不是同一类型，需要转换
			var request R
			if err := c.ShouldBindJSON(&request); err != nil {
				return model, fmt.Errorf(z.Validator.T(err, request))
			}

			// 转换R到T
			var err error
			model, err = ctrl.convertRequestToModel(request)
			if err != nil {
				return model, err
			}
		}
	}

	// 应用覆盖值（无论是否转换都要执行）
	ctrl.applyOverrideValues(&model, overrideValues)

	return model, nil
}

// getActualType 获取类型的实际类型（处理指针类型）
func (ctrl *CrudController[T, S, R, O]) getActualType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// 字段映射缓存
var fieldMappingCache = sync.Map{}

// fieldMapping 存储字段映射信息
type fieldMapping struct {
	sourceIndex  int  // 源字段索引
	targetIndex  int  // 目标字段索引
	needsConvert bool // 是否需要类型转换
}

// getFieldMappings 获取字段映射关系（带缓存）
func (ctrl *CrudController[T, S, R, O]) getFieldMappings() []fieldMapping {
	var zeroT T
	var zeroO O

	sourceType := reflect.TypeOf(zeroT)
	targetType := reflect.TypeOf(zeroO)

	// 生成缓存键
	cacheKey := fmt.Sprintf("%s->%s", sourceType.String(), targetType.String())

	// 尝试从缓存获取
	if cached, ok := fieldMappingCache.Load(cacheKey); ok {
		return cached.([]fieldMapping)
	}

	// 构建字段映射
	var mappings []fieldMapping

	// 处理指针类型
	sourceType = ctrl.getActualType(sourceType)
	targetType = ctrl.getActualType(targetType)

	// 构建源类型的字段索引映射（包括嵌入字段）
	sourceFieldMap := ctrl.buildFieldIndexMap(sourceType)

	// 为目标类型的每个字段查找源字段
	for i := 0; i < targetType.NumField(); i++ {
		targetField := targetType.Field(i)

		// 在源类型中查找同名字段（包括嵌入字段）
		if sourceIndex, found := sourceFieldMap[targetField.Name]; found {
			// 获取源字段信息
			sourceField := ctrl.getFieldByIndex(sourceType, sourceIndex)

			mapping := fieldMapping{
				sourceIndex:  sourceIndex[len(sourceIndex)-1], // 使用最后一级的索引
				targetIndex:  i,
				needsConvert: !sourceField.Type.AssignableTo(targetField.Type),
			}
			mappings = append(mappings, mapping)
		}
	}

	// 存入缓存
	fieldMappingCache.Store(cacheKey, mappings)
	return mappings
}

// buildFieldIndexMap 构建字段名到索引路径的映射（支持嵌入字段）
func (ctrl *CrudController[T, S, R, O]) buildFieldIndexMap(t reflect.Type) map[string][]int {
	fieldMap := make(map[string][]int)
	ctrl.buildFieldIndexMapRecursive(t, []int{}, fieldMap)
	return fieldMap
}

// buildFieldIndexMapRecursive 递归构建字段索引映射
func (ctrl *CrudController[T, S, R, O]) buildFieldIndexMapRecursive(t reflect.Type, indexPath []int, fieldMap map[string][]int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		currentPath := append(indexPath, i)

		// 如果是嵌入字段，递归处理
		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				ctrl.buildFieldIndexMapRecursive(fieldType, currentPath, fieldMap)
			}
		} else {
			// 非嵌入字段，直接添加到映射中
			fieldMap[field.Name] = currentPath
		}
	}
}

// getFieldByIndex 根据索引路径获取字段信息
func (ctrl *CrudController[T, S, R, O]) getFieldByIndex(t reflect.Type, indexPath []int) reflect.StructField {
	currentType := t
	var field reflect.StructField

	for _, index := range indexPath {
		field = currentType.Field(index)
		if field.Anonymous {
			currentType = field.Type
			if currentType.Kind() == reflect.Ptr {
				currentType = currentType.Elem()
			}
		}
	}

	return field
}

// convertModelToResponse 将模型转换为响应数据
func (ctrl *CrudController[T, S, R, O]) convertModelToResponse(data interface{}) interface{} {
	var zeroO O
	var zeroT T

	// 检查O是否为nil类型
	if reflect.TypeOf(zeroO) == nil {
		// 直接返回原数据
		return data
	}

	// 检查O和T是否为同一类型
	oType := reflect.TypeOf(zeroO)
	tType := reflect.TypeOf(zeroT)

	// 处理指针类型比较
	oType = ctrl.getActualType(oType)
	tType = ctrl.getActualType(tType)

	if oType == tType {
		// O和T是同一类型，直接返回原数据，跳过转换
		return data
	}

	// O和T不是同一类型，需要转换为O类型
	return ctrl.convertToResponseType(data)
}

// convertToResponseType 转换数据到响应类型
func (ctrl *CrudController[T, S, R, O]) convertToResponseType(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	dataValue := reflect.ValueOf(data)

	// 如果data是指针，获取其指向的值
	if dataValue.Kind() == reflect.Ptr {
		if dataValue.IsNil() {
			return nil
		}
		dataValue = dataValue.Elem()
	}

	dataType := dataValue.Type()

	// 处理切片类型 - 优化批量处理
	if dataType.Kind() == reflect.Slice {
		length := dataValue.Len()
		if length == 0 {
			return []O{}
		}

		// 预分配切片容量
		results := make([]O, 0, length)

		// 获取字段映射（只需要获取一次）
		mappings := ctrl.getFieldMappings()

		// 批量转换
		for i := 0; i < length; i++ {
			item := dataValue.Index(i).Interface()
			converted := ctrl.convertSingleItemOptimized(item, mappings)
			results = append(results, converted)
		}
		return results
	}

	// 处理单个项目
	mappings := ctrl.getFieldMappings()
	return ctrl.convertSingleItemOptimized(data, mappings)
}

// convertSingleItemOptimized 优化的单项转换
func (ctrl *CrudController[T, S, R, O]) convertSingleItemOptimized(item interface{}, mappings []fieldMapping) O {
	var response O

	if item == nil {
		return response
	}

	itemValue := reflect.ValueOf(item)

	// 如果item是指针，获取其指向的值
	if itemValue.Kind() == reflect.Ptr {
		if itemValue.IsNil() {
			return response
		}
		itemValue = itemValue.Elem()
	}

	responseValue := reflect.ValueOf(&response).Elem()

	// 构建源类型的字段索引映射（包括嵌入字段）
	sourceFieldMap := ctrl.buildFieldIndexMap(itemValue.Type())

	// 使用预计算的字段映射进行转换
	for _, mapping := range mappings {
		targetField := responseValue.Field(mapping.targetIndex)
		targetFieldName := responseValue.Type().Field(mapping.targetIndex).Name

		// 通过字段名查找源字段的完整路径
		if indexPath, found := sourceFieldMap[targetFieldName]; found {
			sourceField := ctrl.getFieldValueByIndexPath(itemValue, indexPath)

			if sourceField.IsValid() && targetField.CanSet() {
				if !mapping.needsConvert {
					// 直接赋值
					targetField.Set(sourceField)
				} else {
					// 需要类型转换的情况（可以根据需要扩展）
					if sourceField.Type().ConvertibleTo(targetField.Type()) {
						targetField.Set(sourceField.Convert(targetField.Type()))
					}
				}
			}
		}
	}

	return response
}

// getFieldValueByIndexPath 根据索引路径获取字段值
func (ctrl *CrudController[T, S, R, O]) getFieldValueByIndexPath(value reflect.Value, indexPath []int) reflect.Value {
	currentValue := value

	for _, index := range indexPath {
		if currentValue.Kind() == reflect.Ptr {
			if currentValue.IsNil() {
				return reflect.Value{}
			}
			currentValue = currentValue.Elem()
		}

		if index >= currentValue.NumField() {
			return reflect.Value{}
		}

		currentValue = currentValue.Field(index)
	}

	return currentValue
}

// 字段名映射缓存
var fieldNameMappingCache = sync.Map{}

// getFieldNameMapping 获取字段名映射关系（结构体字段名 -> 数据库字段名）
func (ctrl *CrudController[T, S, R, O]) getFieldNameMapping() map[string]string {
	var zeroT T
	modelType := reflect.TypeOf(zeroT)

	// 处理指针类型
	modelType = ctrl.getActualType(modelType)

	// 生成缓存键
	cacheKey := modelType.String()

	// 尝试从缓存获取
	if cached, ok := fieldNameMappingCache.Load(cacheKey); ok {
		return cached.(map[string]string)
	}

	// 构建字段名映射
	mapping := make(map[string]string)
	ctrl.buildFieldNameMappingRecursive(modelType, mapping)

	// 存入缓存
	fieldNameMappingCache.Store(cacheKey, mapping)
	return mapping
}

// buildFieldNameMappingRecursive 递归构建字段名映射（支持嵌入字段）
func (ctrl *CrudController[T, S, R, O]) buildFieldNameMappingRecursive(t reflect.Type, mapping map[string]string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 如果是嵌入字段，递归处理
		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				ctrl.buildFieldNameMappingRecursive(fieldType, mapping)
			}
			continue
		}

		structFieldName := field.Name

		// 获取数据库字段名（从 json 或 gorm 标签）
		dbFieldName := structFieldName

		// 优先使用 gorm 标签中的 column 名称
		if gormTag := field.Tag.Get("gorm"); gormTag != "" {
			if columnStart := strings.Index(gormTag, "column:"); columnStart != -1 {
				columnStart += 7 // len("column:")
				columnEnd := strings.Index(gormTag[columnStart:], ";")
				if columnEnd == -1 {
					columnEnd = len(gormTag[columnStart:])
				}
				dbFieldName = gormTag[columnStart : columnStart+columnEnd]
			}
		}

		// 如果没有 gorm 标签，使用 json 标签
		if dbFieldName == structFieldName {
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				// 去除 json 标签中的选项（如 omitempty）
				if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
					dbFieldName = jsonTag[:commaIndex]
				} else {
					dbFieldName = jsonTag
				}
			}
		}

		// 如果还是没有，使用蛇形命名转换
		if dbFieldName == structFieldName {
			dbFieldName = ctrl.toSnakeCase(structFieldName)
		}

		// 建立双向映射：结构体字段名 -> 数据库字段名，数据库字段名 -> 结构体字段名
		mapping[structFieldName] = dbFieldName
		if dbFieldName != structFieldName {
			mapping[dbFieldName] = structFieldName
		}
	}
}

// toSnakeCase 将驼峰命名转换为蛇形命名
func (ctrl *CrudController[T, S, R, O]) toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// applyOverrideValues 应用覆盖值到模型（支持结构体字段名和数据库字段名）
func (ctrl *CrudController[T, S, R, O]) applyOverrideValues(model *T, overrideValues map[string]interface{}) {
	if overrideValues == nil {
		return
	}

	modelValue := reflect.ValueOf(model).Elem()
	fieldMapping := ctrl.getFieldNameMapping()

	for fieldName, value := range overrideValues {
		var targetFieldName string

		// 首先尝试直接使用字段名（可能是结构体字段名）
		if modelField := modelValue.FieldByName(fieldName); modelField.IsValid() && modelField.CanSet() {
			targetFieldName = fieldName
		} else {
			// 如果直接查找失败，尝试通过映射查找
			if mappedName, exists := fieldMapping[fieldName]; exists {
				if modelField := modelValue.FieldByName(mappedName); modelField.IsValid() && modelField.CanSet() {
					targetFieldName = mappedName
				}
			}
		}

		// 如果找到了目标字段，进行赋值
		if targetFieldName != "" {
			modelField := modelValue.FieldByName(targetFieldName)
			valueReflect := reflect.ValueOf(value)

			if valueReflect.Type().AssignableTo(modelField.Type()) {
				modelField.Set(valueReflect)
			} else if valueReflect.Type().ConvertibleTo(modelField.Type()) {
				// 尝试类型转换
				modelField.Set(valueReflect.Convert(modelField.Type()))
			}
		}
	}
}

// convertRequestToModel 将请求结构体转换为模型
func (ctrl *CrudController[T, S, R, O]) convertRequestToModel(request R) (T, error) {
	var model T
	modelValue := reflect.ValueOf(&model).Elem()
	requestValue := reflect.ValueOf(request)

	// 如果request是指针，获取其指向的值
	if requestValue.Kind() == reflect.Ptr {
		requestValue = requestValue.Elem()
	}

	requestType := requestValue.Type()

	// 遍历request的字段，复制到model中
	for i := 0; i < requestValue.NumField(); i++ {
		requestField := requestValue.Field(i)
		requestFieldName := requestType.Field(i).Name

		// 在model中查找同名字段
		if modelField := modelValue.FieldByName(requestFieldName); modelField.IsValid() && modelField.CanSet() {
			// 处理指针类型字段
			if requestField.Kind() == reflect.Ptr {
				if !requestField.IsNil() {
					// 指针不为空，取值并设置
					requestFieldValue := requestField.Elem()
					if requestFieldValue.Type().AssignableTo(modelField.Type()) {
						modelField.Set(requestFieldValue)
					}
				}
				// 指针为空时不设置，保持模型字段的零值
			} else {
				// 非指针类型，直接检查类型兼容性并设置
				if requestField.Type().AssignableTo(modelField.Type()) {
					modelField.Set(requestField)
				}
			}
		}
	}

	return model, nil
}
