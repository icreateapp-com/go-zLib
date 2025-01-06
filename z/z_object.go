package z

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
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
		if !Contains(fields, field.Name) {
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
		if !Contains(fields, field.Name) {
			newModelValue.FieldByName(field.Name).Set(modelValue.Field(i))
		}
	}

	return newModelValue.Interface()
}

// Contains 判断字符串是否在切片中
func Contains(slice []string, str string) bool {
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
