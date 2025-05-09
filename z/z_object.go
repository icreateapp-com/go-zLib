package z

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetSortedMapKeys 返回排序后的 MAP 所有键
func GetSortedMapKeys(elements map[string]interface{}) []string {
	keys := make([]string, 0, len(elements))

	for k := range elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// GetMapKeys 返回 MAP 的所有键
func GetMapKeys(elements map[string]interface{}) []string {
	keys := make([]string, 0, len(elements))

	for k, _ := range elements {
		keys = append(keys, k)
	}

	return keys
}

// ToValues 转换表单项
func ToValues(data map[string]interface{}) url.Values {
	values := url.Values{}

	for k, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			nestedValues := ToValues(val)
			for nk, nv := range nestedValues {
				values.Add(fmt.Sprintf("%s[%s]", k, nk), nv[0])
			}
		default:
			values.Add(k, fmt.Sprintf("%v", v))
		}
	}

	return values
}

// RemoveFields 移除结构体字段
func RemoveFields(model interface{}, fields ...string) interface{} {
	modelType := reflect.TypeOf(model).Elem()
	modelValue := reflect.ValueOf(model).Elem()

	var newModelFields []reflect.StructField
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if !InStringSlice(fields, field.Name) {
			newModelFields = append(newModelFields, reflect.StructField{
				Name: field.Name,
				Type: field.Type,
				Tag:  field.Tag,
			})
		}
	}

	newModelType := reflect.StructOf(newModelFields)
	newModelValue := reflect.New(newModelType).Elem()

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if !InStringSlice(fields, field.Name) {
			newModelValue.FieldByName(field.Name).Set(modelValue.Field(i))
		}
	}

	return newModelValue.Interface()
}

// InStringSlice 判断字符串是否在切片中
func InStringSlice(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

// HasField 判断结构体是否有指定字段
func HasField(obj interface{}, name string) bool {
	_, has := reflect.TypeOf(obj).FieldByName(name)
	return has
}

// CreateNewStruct 创建一个新的结构体，只包含指定的字段
func CreateNewStruct(original interface{}, fields []string) (interface{}, error) {
	origVal := reflect.ValueOf(original)
	origType := origVal.Type()

	if origVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got %s", origVal.Kind())
	}

	newStructType := reflect.StructOf(BuildFieldTypes(origType, fields))
	newStructValue := reflect.New(newStructType).Elem()

	for _, field := range fields {
		if origVal.FieldByName(field).IsValid() {
			newStructValue.FieldByName(field).Set(origVal.FieldByName(field))
		}
	}

	return newStructValue.Interface(), nil
}

// BuildFieldTypes 构建字段类型
func BuildFieldTypes(origType reflect.Type, fields []string) []reflect.StructField {
	var structFields []reflect.StructField
	for _, field := range fields {
		origField, _ := origType.FieldByName(field)
		structField := reflect.StructField{
			Name: field,
			Type: origField.Type,
			Tag:  origField.Tag,
		}
		structFields = append(structFields, structField)
	}
	return structFields
}

// FormatTimeInMap 格式化对象时间
func FormatTimeInMap(m map[string]interface{}) {
	for k, v := range m {
		if t, ok := v.(time.Time); ok {
			m[k] = t.Format("2006-01-02 15:04:05")
		} else if reflect.ValueOf(v).Kind() == reflect.Map {
			subMap, ok := v.(map[string]interface{})
			if ok {
				FormatTimeInMap(subMap)
			}
		}
	}
}

// ToStruct 函数将 interface{} 转换为指定的 struct 类型
func ToStruct(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonData, &target); err != nil {
		return err
	}

	return nil
}

// ToFloat64 将接口转换为 float64
func ToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		num, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return num, true
	default:
		return 0, false
	}
}

// ToInt 将接口转换为 int
func ToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		num, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return num, true
	default:
		return 0, false
	}
}

// ToBool 将接口转换为 bool
func ToBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, false
		}
		return b, true
	default:
		return false, false
	}
}

// GetStructField 使用反射获取结构体字段的值
func GetStructField(s interface{}, fieldName string) (interface{}, bool) {
	if s == nil {
		return nil, false
	}
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil, false
	}
	elem := val.Elem()
	field := elem.FieldByName(fieldName)
	if field.IsValid() {
		return field.Interface(), true
	}
	return nil, false
}

// GetValidDataByStruct 验证并修正输入数据
func GetValidDataByStruct(nodeDataMap map[string]interface{}, nodeDataStruct interface{}) (map[string]interface{}, error) {
	validate := validator.New()

	stValue := reflect.ValueOf(nodeDataStruct)
	if stValue.Kind() == reflect.Ptr {
		if stValue.IsNil() {
			return nil, fmt.Errorf("nodeDataStruct pointer is nil")
		}
		stValue = stValue.Elem()
	}

	stType := stValue.Type()
	if stType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("nodeDataStruct must be a struct or pointer to struct")
	}

	result := make(map[string]interface{})

	for i := 0; i < stType.NumField(); i++ {
		field := stType.Field(i)

		// 跳过未导出字段
		if field.PkgPath != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" || jsonTag == "" {
			continue
		}
		jsonName := strings.Split(jsonTag, ",")[0]

		value, exists := nodeDataMap[jsonName]
		if exists {
			result[jsonName] = value
			continue
		}

		// 使用默认值
		defaultValue := field.Tag.Get("default")
		if defaultValue != "" {
			if converted, ok := ConvertDefaultValue(defaultValue, field.Type.Kind()); ok {
				result[jsonName] = converted
			}
		}
	}

	// 将 map 转成结构体以进行验证
	newInstance := reflect.New(stType).Interface()
	if err := ToStruct(result, newInstance); err != nil {
		return nil, fmt.Errorf("convert to struct failed: %w", err)
	}

	if err := validate.Struct(newInstance); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return result, nil
}

// ConvertDefaultValue 尝试将字符串默认值转换为指定类型
func ConvertDefaultValue(str string, kind reflect.Kind) (interface{}, bool) {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			return val, true
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(str, 10, 64); err == nil {
			return val, true
		}
	case reflect.Float32, reflect.Float64:
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			return val, true
		}
	case reflect.Bool:
		if val, err := strconv.ParseBool(str); err == nil {
			return val, true
		}
	case reflect.String:
		return str, true
	}
	return nil, false
}
