# 工具函数模块

go-zLib 的工具函数模块提供了一系列常用的辅助函数，涵盖字符串处理、切片操作、对象操作、路径处理、编码解码等多个方面，旨在简化日常开发中的常见任务。

## 目录
- [字符串工具](#字符串工具)
- [切片工具](#切片工具)
- [对象工具](#对象工具)
- [接口工具](#接口工具)
- [路径工具](#路径工具)
- [编码解码工具](#编码解码工具)
- [数学工具](#数学工具)
- [文件流工具](#文件流工具)

## 字符串工具

字符串工具提供了常用的字符串处理函数。

### 字符串判空

```go
// 判断字符串是否为空或空白
isEmpty := z.StringIsEmpty("")       // true
isEmpty := z.StringIsEmpty("  ")     // true
isEmpty := z.StringIsEmpty("hello")  // false
```

### 字符串转换

```go
// 字符串转整数
num, err := z.StringToInt("123")

// 字符串转浮点数
num, err := z.StringToFloat("123.45")

// 字符串转布尔值
b, err := z.StringToBool("true")

// 驼峰转蛇形
snake := z.CamelToSnake("HelloWorld")  // "hello_world"

// 蛇形转驼峰
camel := z.SnakeToCamel("hello_world")  // "HelloWorld"
```

### 字符串格式化

```go
// 格式化金额（保留两位小数）
formatted := z.FormatMoney(12345.678)  // "12,345.68"

// 字符串左填充
padded := z.StrPadLeft("123", "0", 5)  // "00123"

// 字符串右填充
padded := z.StrPadRight("123", "0", 5)  // "12300"

// 生成指定长度的随机字符串
random := z.RandomStr(8)  // 如 "a1b2c3d4"
```

### 字符串截取

```go
// 字符串截取
sub := z.SubStr("hello world", 0, 5)  // "hello"

// 截取指定长度，超出部分用省略号代替
ellipsis := z.SubStrWithEllipsis("hello world", 5)  // "hello..."
```

## 切片工具

切片工具提供了常用的切片操作函数。

### 切片判断

```go
// 判断元素是否在切片中
exists := z.SliceContains([]string{"a", "b", "c"}, "b")  // true
exists := z.SliceContains([]int{1, 2, 3}, 4)             // false
```

### 切片操作

```go
// 切片元素去重
unique := z.SliceUnique([]string{"a", "b", "a", "c"})  // ["a", "b", "c"]

// 移除切片中的元素
filtered := z.SliceRemove([]int{1, 2, 3, 4}, 2)  // [1, 3, 4]

// 切片转映射
m := z.SliceToMap([]string{"a", "b", "c"}, []string{"A", "B", "C"})
// 结果: map[string]string{"a": "A", "b": "B", "c": "C"}

// 获取切片交集
intersection := z.SliceIntersect([]int{1, 2, 3}, []int{2, 3, 4})  // [2, 3]

// 获取切片差集
diff := z.SliceDiff([]int{1, 2, 3}, []int{2, 3, 4})  // [1]
```

### 切片转换

```go
// 字符串切片转整数切片
intSlice, err := z.StringSliceToIntSlice([]string{"1", "2", "3"})

// 数字切片转字符串切片
strSlice := z.NumberSliceToStringSlice([]int{1, 2, 3})
```

## 对象工具

对象工具提供了结构体和映射的操作函数。

### 结构体操作

```go
// 结构体转映射
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
user := User{Name: "张三", Age: 30}
m, err := z.StructToMap(user)
// 结果: map[string]interface{}{"name": "张三", "age": 30}

// 映射转结构体
m := map[string]interface{}{"name": "李四", "age": 25}
var user User
err := z.MapToStruct(m, &user)
```

### 映射操作

```go
// 合并映射
m1 := map[string]interface{}{"a": 1, "b": 2}
m2 := map[string]interface{}{"b": 3, "c": 4}
merged := z.MergeMaps(m1, m2)
// 结果: map[string]interface{}{"a": 1, "b": 3, "c": 4}

// 获取映射键列表
keys := z.MapKeys(map[string]int{"a": 1, "b": 2})  // ["a", "b"]

// 获取映射值列表
values := z.MapValues(map[string]int{"a": 1, "b": 2})  // [1, 2]
```

### 深度拷贝

```go
// 深度拷贝对象
original := map[string]interface{}{
    "name": "张三",
    "contacts": map[string]string{
        "email": "zhangsan@example.com",
    },
}
copied := z.DeepCopy(original)
```

## 接口工具

接口工具提供了通用的接口转换和操作函数。

### 类型转换

```go
// 接口转字符串
str, err := z.InterfaceToString(123)  // "123"

// 接口转整数
num, err := z.InterfaceToInt("123")   // 123

// 接口转浮点数
f, err := z.InterfaceToFloat64("123.45")  // 123.45

// 接口转布尔值
b, err := z.InterfaceToBool("true")  // true
```

### 类型断言

```go
// 安全获取映射中的值
val, ok := z.SafeGet(map[string]interface{}{"name": "张三"}, "name")
if ok {
    name := val.(string)
}

// 安全获取映射中的字符串值
name, ok := z.SafeGetString(map[string]interface{}{"name": "张三"}, "name")

// 安全获取映射中的整数值
age, ok := z.SafeGetInt(map[string]interface{}{"age": 30}, "age")
```

## 路径工具

路径工具提供了文件和目录路径处理函数。

### 路径获取

```go
// 获取项目根目录
rootPath := z.BasePath()

// 获取存储目录
storagePath := z.StoragePath("uploads")

// 获取临时目录
tempPath := z.TempPath("cache")

// 获取日志目录
logPath := z.LogPath()

// 获取当前可执行文件路径
exePath := z.ExePath()
```

### 路径检查和创建

```go
// 检查路径是否存在
exists := z.PathExists("/path/to/file")

// 创建目录（如果不存在）
err := z.EnsureDirExists("/path/to/dir")
```

## 编码解码工具

编码解码工具提供了常用的编码和解码函数。

### 哈希计算

```go
// 计算 MD5 值
md5 := z.GetMd5("hello")

// 计算 SHA1 值
sha1 := z.GetSha1("hello")

// 计算 SHA256 值
sha256 := z.GetSha256("hello")
```

### Base64 编码解码

```go
// Base64 编码
encoded := z.EncodeBase64("hello")

// Base64 解码
decoded, err := z.DecodeBase64(encoded)
```

### URL 编码解码

```go
// URL 编码
encoded := z.UrlEncode("hello world")

// URL 解码
decoded, err := z.UrlDecode(encoded)
```

### JSON 操作

```go
// JSON 编码
jsonStr, err := z.JsonEncode(map[string]interface{}{"name": "张三", "age": 30})

// JSON 解码
var data map[string]interface{}
err := z.JsonDecode(jsonStr, &data)

// 从 JSON 中提取指定键值
name, err := z.JsonExtract(jsonStr, "name")
```

## 数学工具

数学工具提供了一些常用的数学操作函数。

### 数值范围限制

```go
// 限制数值在指定范围内
limited := z.Clamp(5, 1, 10)  // 5
limited := z.Clamp(0, 1, 10)  // 1
limited := z.Clamp(11, 1, 10) // 10
```

### 随机数生成

```go
// 生成指定范围内的随机整数
random := z.RandomInt(1, 100)

// 生成指定范围内的随机浮点数
random := z.RandomFloat(0, 1)
```

### 四舍五入

```go
// 四舍五入到指定小数位
rounded := z.Round(123.456, 2)  // 123.46
```

## 文件流工具

文件流工具提供了处理文件和数据流的功能。

### 创建和打开文件

```go
// 创建文件
file, err := z.CreateFile("/path/to/file.txt")
defer file.Close()

// 打开文件
file, err := z.OpenFile("/path/to/file.txt")
defer file.Close()
```

### 读写文件

```go
// 读取整个文件内容
content, err := z.ReadFile("/path/to/file.txt")

// 写入字符串到文件
err := z.WriteFile("/path/to/file.txt", "hello world")

// 追加字符串到文件
err := z.AppendFile("/path/to/file.txt", "new content")
```

### 数据流处理

```go
// 创建流处理对象
stream := z.NewStream()

// 添加处理函数
stream.Add(func(data interface{}) (interface{}, error) {
    str, ok := data.(string)
    if !ok {
        return nil, errors.New("数据类型不是字符串")
    }
    return strings.ToUpper(str), nil
})

// 处理数据
result, err := stream.Process("hello")  // "HELLO"
``` 