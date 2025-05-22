# 数据库操作模块

go-zLib 的数据库操作模块基于 GORM 库进行了封装，提供了更简单、更一致的数据库操作接口。本模块的主要目标是降低数据库访问的复杂度，提供类似构建器模式的 API 设计。

## 目录
- [数据库操作模块](#数据库操作模块)
  - [目录](#目录)
  - [初始化连接](#初始化连接)
    - [配置示例](#配置示例)
  - [模型定义](#模型定义)
  - [CRUD 操作](#crud-操作)
    - [创建记录](#创建记录)
      - [CreateBuilder 方法](#createbuilder-方法)
    - [查询记录](#查询记录)
      - [Query 结构说明](#query-结构说明)
      - [ConditionGroup 结构说明](#conditiongroup-结构说明)
      - [QueryBuilder 方法](#querybuilder-方法)
    - [更新记录](#更新记录)
      - [UpdateBuilder 方法](#updatebuilder-方法)
    - [删除记录](#删除记录)
      - [DeleteBuilder 方法](#deletebuilder-方法)
  - [事务处理](#事务处理)
  - [自定义时间格式](#自定义时间格式)
  - [JSON 字段处理](#json-字段处理)
  - [高级用法](#高级用法)
    - [字段转义](#字段转义)
    - [原始 SQL 查询](#原始-sql-查询)

## 初始化连接

数据库连接需要在应用启动时初始化。默认情况下，数据库配置从配置文件中读取。

```go
import "github.com/icreateapp-com/go-zLib/z/db"

// 初始化数据库连接
db := db.DB.New()
```

### 配置示例

在配置文件中需要添加以下配置：

```yaml
config:
  db:
    driver: "mysql"  # 数据库驱动类型
    host: "localhost"
    port: 3306
    username: "root"
    password: "password"
    database: "mydb"
    charset: "utf8mb4"
    loc: "Local"
  debug: true  # 是否开启调试模式，影响日志级别
```

## 模型定义

所有模型都应该嵌入 `db.Model` 结构体，它提供了基础的表字段和行为。

```go
import "github.com/icreateapp-com/go-zLib/z/db"

// 用户模型
type User struct {
    db.Model            // 嵌入基础模型
    Name     string     `json:"name" gorm:"column:name"`
    Email    string     `json:"email" gorm:"column:email;uniqueIndex"`
    Age      int        `json:"age" gorm:"column:age"`
    Active   bool       `json:"active" gorm:"column:active;default:true"`
}

// 表名
func (User) TableName() string {
    return "users"
}
```

`db.Model` 结构体包含以下字段：

```go
type Model struct {
    ID        uint       `gorm:"primaryKey" json:"id"`
    CreatedAt WrapTime   `json:"created_at"`
    UpdatedAt WrapTime   `json:"updated_at"`
    DeletedAt *time.Time `gorm:"index" json:"-"`
}
```

## CRUD 操作

### 创建记录

使用 `CreateBuilder` 创建新记录：

```go
// 初始化构建器
createBuilder := db.CreateBuilder{Model: &User{}}

// 创建单条记录
user := User{
    Name: "张三",
    Email: "zhangsan@example.com",
    Age: 25,
}
err := createBuilder.Create(&user)
if err != nil {
    // 处理错误
}
```

#### CreateBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| Create | data (interface{}) | error | 创建单条记录 |

### 查询记录

使用 `QueryBuilder` 进行各种查询操作：

```go
// 初始化查询构建器
queryBuilder := db.QueryBuilder{Model: &User{}}

// 定义查询条件
query := db.Query{
    Filter: []string{"id", "name", "email"}, // 要返回的字段
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{
                {"name", "like", "%张%"},
                {"age", ">", 18},
            },
            Logic: "AND", // 默认为 AND
        },
    },
    Sort: [][]string{
        {"created_at", "desc"},
    },
    Page: 1,
    PageSize: 10,
}

// 分页查询
result, err := queryBuilder.Page(query)
if err != nil {
    // 处理错误
}
```

#### Query 结构说明

| 字段 | 类型 | 说明 |
|------|------|------|
| Filter | []string | 要返回的字段列表 |
| Search | []ConditionGroup | 查询条件组 |
| Sort | [][]string | 排序字段，每项为 [字段名, 排序方向] |
| Page | int | 当前页码，从 1 开始 |
| PageSize | int | 每页记录数 |

#### ConditionGroup 结构说明

| 字段 | 类型 | 说明 |
|------|------|------|
| Conditions | [][]interface{} | 条件列表，每项为 [字段名, 操作符, 值] 或 [字段名, 值] |
| Logic | string | 条件间的逻辑关系，"AND" 或 "OR"，默认为 "AND" |

#### QueryBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| Page | query (Query) | (interface{}, error) | 分页查询 |
| Get | query (Query) | (interface{}, error) | 获取符合条件的所有记录 |
| First | query (Query) | (interface{}, error) | 获取符合条件的第一条记录 |
| FindById | id (interface{}), query (Query) | (interface{}, error) | 根据 ID 查询记录 |
| Count | query (Query) | (int64, error) | 统计符合条件的记录数 |

### 更新记录

使用 `UpdateBuilder` 更新记录：

```go
// 初始化更新构建器
updateBuilder := db.UpdateBuilder{Model: &User{}}

// 更新数据
updateData := map[string]interface{}{
    "name": "李四",
    "age": 30,
}

// 根据 ID 更新
success, err := updateBuilder.UpdateByID(1, updateData)
if err != nil {
    // 处理错误
}
```

#### UpdateBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| UpdateByID | id (interface{}), data (interface{}) | (bool, error) | 根据 ID 更新记录 |
| Update | query (Query), data (interface{}) | (bool, error) | 根据条件更新记录 |

### 删除记录

使用 `DeleteBuilder` 删除记录：

```go
// 初始化删除构建器
deleteBuilder := db.DeleteBuilder{Model: &User{}}

// 根据 ID 删除
success, err := deleteBuilder.DeleteByID(1)
if err != nil {
    // 处理错误
}

// 根据条件删除
query := db.Query{
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{
                {"age", "<", 18},
            },
        },
    },
}
success, err = deleteBuilder.Delete(query)
if err != nil {
    // 处理错误
}
```

#### DeleteBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| DeleteByID | id (interface{}) | (bool, error) | 根据 ID 删除记录 |
| Delete | query (Query) | (bool, error) | 根据条件删除记录 |

## 事务处理

使用 `Transaction` 方法进行事务操作：

```go
err := db.DB.Transaction(func(tx *gorm.DB) error {
    // 在事务中执行操作
    user := User{Name: "王五", Email: "wangwu@example.com"}
    if err := tx.Create(&user).Error; err != nil {
        // 返回任何错误都会回滚事务
        return err
    }
    
    // 更多数据库操作...
    
    // 返回 nil 提交事务
    return nil
})
```

## 自定义时间格式

通过 `WrapTime` 类型实现自定义的时间格式：

```go
// 在模型中使用
type Post struct {
    db.Model
    Title     string     `json:"title"`
    PublishAt db.WrapTime `json:"publish_at"`
}
```

`WrapTime` 类型会自动处理 JSON 序列化与反序列化，默认格式为 "2006-01-02 15:04:05"。

## JSON 字段处理

使用 `db.JSON` 类型处理 JSON 字段：

```go
type Settings struct {
    Theme string `json:"theme"`
    Notifications bool `json:"notifications"`
}

type UserProfile struct {
    db.Model
    UserID    uint      `json:"user_id"`
    Settings  db.JSON   `json:"settings" gorm:"type:json"`
}

// 使用
profile := UserProfile{
    UserID: 1,
    Settings: db.JSON{Data: Settings{Theme: "dark", Notifications: true}},
}
```

## 高级用法

### 字段转义

使用 `F` 方法进行字段名转义，避免 SQL 关键字冲突：

```go
fieldName := db.DB.F("name")
// 对于 MySQL，会返回 "`name`"
```

### 原始 SQL 查询

在需要执行原始 SQL 查询时，可以直接使用 GORM 的方法：

```go
var users []User
db.DB.Raw("SELECT * FROM users WHERE age > ?", 18).Scan(&users)
``` 