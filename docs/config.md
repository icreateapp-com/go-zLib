# `zLib` 包中 `Config` 说明文档：

```markdown
# zLib 包配置说明文档

## 全局变量

- `configs`: 存储所有配置文件。

## `config` 结构体

`config` 结构体用于管理配置文件。

## 全局变量

- `Config`: 全局配置对象，可以在包外部使用。

## 方法

### `LoadDir`

加载指定目录下的所有配置文件。

```go
func (c *config) LoadDir(dir string) error
```

#### 参数

- `dir` (`string`): 配置文件所在的目录。

#### 返回值

- `error`: 如果加载配置文件时发生错误，返回错误信息。

### `LoadFile`

加载指定文件。

```go
func (c *config) LoadFile(dir string, filename string) error
```

#### 参数

- `dir` (`string`): 配置文件所在的目录。
- `filename` (`string`): 配置文件的文件名。

#### 返回值

- `error`: 如果加载配置文件时发生错误，返回错误信息。

### `parseName`

解析配置文件名和配置项名。

```go
func (c *config) parseName(name string) (v *viper.Viper, valueName string, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `*viper.Viper`: 配置文件对应的 `viper.Viper` 对象。
- `string`: 配置项的名称。
- `error`: 如果解析名称时发生错误，返回错误信息。

### `String`

获取字符串类型的配置项。

```go
func (c *config) String(name string) (value string, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `string`: 配置项的字符串值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Bool`

获取布尔类型的配置项。

```go
func (c *config) Bool(name string) (value bool, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `bool`: 配置项的布尔值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Int`

获取整数类型的配置项。

```go
func (c *config) Int(name string) (value int, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `int`: 配置项的整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Int32`

获取32位整数类型的配置项。

```go
func (c *config) Int32(name string) (value int32, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `int32`: 配置项的32位整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Int64`

获取64位整数类型的配置项。

```go
func (c *config) Int64(name string) (value int64, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `int64`: 配置项的64位整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Uint`

获取无符号整数类型的配置项。

```go
func (c *config) Uint(name string) (value uint, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `uint`: 配置项的无符号整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Uint16`

获取16位无符号整数类型的配置项。

```go
func (c *config) Uint16(name string) (value uint16, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `uint16`: 配置项的16位无符号整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Uint32`

获取32位无符号整数类型的配置项。

```go
func (c *config) Uint32(name string) (value uint32, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `uint32`: 配置项的32位无符号整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Uint64`

获取64位无符号整数类型的配置项。

```go
func (c *config) Uint64(name string) (value uint64, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `uint64`: 配置项的64位无符号整数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Float64`

获取浮点数类型的配置项。

```go
func (c *config) Float64(name string) (value float64, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `float64`: 配置项的浮点数值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Time`

获取时间类型的配置项。

```go
func (c *config) Time(name string) (value time.Time, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `time.Time`: 配置项的时间值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `Duration`

获取时间间隔类型的配置项。

```go
func (c *config) Duration(name string) (value time.Duration, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `time.Duration`: 配置项的时间间隔值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `IntSlice`

获取整数切片类型的配置项。

```go
func (c *config) IntSlice(name string) (value []int, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `[]int`: 配置项的整数切片值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `StringSlice`

获取字符串切片类型的配置项。

```go
func (c *config) StringSlice(name string) (value []string, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `[]string`: 配置项的字符串切片值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `StringMap`

获取字符串映射类型的配置项。

```go
func (c *config) StringMap(name string) (value map[string]interface{}, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `map[string]interface{}`: 配置项的字符串映射值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `StringMapStringSlice`

获取字符串映射切片类型的配置项。

```go
func (c *config) StringMapStringSlice(name string) (value map[string][]string, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `map[string][]string`: 配置项的字符串映射切片值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

### `SizeInBytes`

获取字节大小的配置项。

```go
func (c *config) SizeInBytes(name string) (value uint, err error)
```

#### 参数

- `name` (`string`): 配置项的完整名称。

#### 返回值

- `uint`: 配置项的字节大小值。
- `error`: 如果获取配置项时发生错误，返回错误信息。

