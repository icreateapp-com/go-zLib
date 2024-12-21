# `zLib` 包中 `Validator` 说明文档：

```markdown
# zLib 包验证说明文档

## `valid` 结构体

`valid` 结构体用于处理数据验证和错误翻译。

## 全局变量

- `Validator`: 全局 `valid` 实例，可以在包外部使用。

## 方法

### `Init`

初始化翻译器。

```go
func (valid *valid) Init()
```

此方法会初始化一个翻译器，用于将验证错误翻译成指定的语言。目前支持英文（en）和中文（zh）。

### `T`

翻译错误。

```go
func (valid *valid) T(err error, req interface{}) string
```

#### 参数

- `err` (`error`): 验证过程中产生的错误。
- `req` (`interface{}`): 请求数据结构体，用于获取字段标签。

#### 返回值

- `string`: 翻译后的错误信息。

此方法会根据传入的错误和请求数据结构体，将错误信息翻译成指定的语言，并替换字段名为对应的标签。

## 使用示例

```go
type User struct {
	Name  string `json:"name" label:"用户名" validate:"required"`
	Email string `json:"email" label:"邮箱" validate:"required,email"`
}

user := User{
	Name:  "",
	Email: "invalid-email",
}

err := Validator.T(validator.New().Struct(user), user)
fmt.Println(err) // 输出：用户名不能为空，邮箱必须是一个有效的邮箱地址
```

