package z

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
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
	val := reflect.ValueOf(s).Elem()
	field := val.FieldByName(fieldName)
	if field.IsValid() {
		return field.Interface(), true
	}
	return nil, false
}
