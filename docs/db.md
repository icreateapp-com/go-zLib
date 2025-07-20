# 数据库操作模块

go-zLib 的数据库操作模块基于 GORM 库进行了封装，提供了更简单、更一致的数据库操作接口。本模块采用泛型设计，确保类型安全，消除了 `interface{}` 的使用，提供直接返回指定模型类型的 API。

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
  - [查询解析器](#查询解析器)
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

所有模型都应该嵌入 `db.Model` 结构体，它提供了基础的表字段和行为，并自动实现了 `db.IModel` 接口。

```go
import "github.com/icreateapp-com/go-zLib/z/db"

// 用户模型
type User struct {
    db.Model            // 嵌入基础模型，自动实现 IModel 接口
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

`db.IModel` 接口定义了模型必须实现的方法：

```go
type IModel interface {
    TableName() string
    BeforeCreate(tx *gorm.DB) error
}
```

## CRUD 操作

### 创建记录

使用 `CreateBuilder` 创建新记录，支持泛型类型安全：

```go
// 创建单条记录
user := User{
    Name: "张三",
    Email: "zhangsan@example.com",
    Age: 25,
}

// 使用泛型 CreateBuilder，直接返回 User 类型
createdUser, err := db.CreateBuilder[User]{}.Create(user)
if err != nil {
    // 处理错误
}
// createdUser 是 User 类型，无需类型转换
fmt.Printf("创建的用户ID: %d\n", createdUser.ID)

// 在事务中创建
tx := db.DB.Begin()
createdUser, err = db.CreateBuilder[User]{TX: tx}.Create(user)
if err != nil {
    tx.Rollback()
    return err
}
tx.Commit()
```

#### CreateBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| Create | data (T) | (T, error) | 创建单条记录，返回创建后的模型 |

### 查询记录

使用 `QueryBuilder` 进行各种查询操作，支持泛型类型安全：

```go
// 定义查询条件
query := db.Query{
    Filter: []string{"id", "name", "email"}, // 要返回的字段
    Search: []db.ConditionGroup{
        {
            {"name", "%张%", "like"},
            {"age", 18, ">"},
        },
    },
    OrderBy: []string{"created_at", "desc"},
    Page: []int{1, 10}, // [页码, 每页数量]
}

// 分页查询，直接返回 PaginatedResult[User] 类型
result, err := db.QueryBuilder[User]{}.Page(query)
if err != nil {
    // 处理错误
}
// result.Data 是 []User 类型，无需类型转换
for _, user := range result.Data {
    fmt.Printf("用户: %s, 邮箱: %s\n", user.Name, user.Email)
}
fmt.Printf("总数: %d, 当前页: %d\n", result.Pager.Total, result.Pager.Page)

// 根据ID查询，支持额外查询条件，直接返回 *User 类型
user, err := db.QueryBuilder[User]{}.Find(1, db.Query{
    Filter: []string{"id", "name", "email"}, // 只返回指定字段
})
if err != nil {
    // 处理错误
}
// user 是 *User 类型，无需类型转换
fmt.Printf("用户名: %s\n", user.Name)

// Find 方法支持复杂查询条件
user, err = db.QueryBuilder[User]{}.Find(1, db.Query{
    Filter: []string{"id", "name", "email"},
    Search: []db.ConditionGroup{
        {{"status", "active", "="}}, // 额外条件：状态必须为 active
    },
    Required: []string{"email"}, // 邮箱不能为空
})
// 这将查询 ID=1 且 status='active' 且 email 不为空的用户

// 查询第一条记录，直接返回 *User 类型
user, err = db.QueryBuilder[User]{}.First(query)
if err != nil {
    // 处理错误
}

// 获取所有记录，直接返回 []User 类型
users, err := db.QueryBuilder[User]{}.Get(query)
if err != nil {
    // 处理错误
}
// users 是 []User 类型，无需类型转换

// 在事务中查询
tx := db.DB.Begin()
user, err = db.QueryBuilder[User]{TX: tx}.Find(1, db.Query{})
```

#### Query 结构说明

| 字段 | 类型 | 说明 |
|------|------|------|
| Filter | []string | 要返回的字段列表 |
| Search | []ConditionGroup | 查询条件组 |
| OrderBy | []string | 排序字段，格式为 [字段名, 排序方向] |
| Limit | []int | 限制记录数，格式为 [数量] 或 [偏移量, 数量] |
| Page | []int | 分页参数，格式为 [页码, 每页数量] |
| Required | []string | 必需字段列表（非空字段） |

#### ConditionGroup 结构说明

`ConditionGroup` 是一个二维数组 `[][]interface{}`，每个内层数组代表一个查询条件：

| 格式 | 说明 | 示例 |
|------|------|------|
| [字段名, 值] | 等值查询，默认操作符为 "=" | `{"name", "张三"}` |
| [字段名, 值, 操作符] | 指定操作符的查询 | `{"age", 18, ">"}` |

**支持的操作符：**
- 比较操作符：`=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`
- 模糊查询：`LIKE`, `NOT LIKE`
- 范围查询：`IN`, `NOT IN`, `BETWEEN`, `NOT BETWEEN`
- 空值查询：`IS NULL`, `IS NOT NULL`

**条件组逻辑：**
- 同一个 `ConditionGroup` 内的条件使用 `OR` 连接
- 不同 `ConditionGroup` 之间使用 `AND` 连接

```go
// 示例：查询名字包含"张"或年龄大于18的用户
query := db.Query{
    Search: []db.ConditionGroup{
        {
            {"name", "%张%", "like"},
            {"age", 18, ">"},
        },
    },
}
// 生成的SQL: WHERE (name LIKE '%张%' OR age > 18)

// 示例：查询名字包含"张"且年龄大于18的用户
query := db.Query{
    Search: []db.ConditionGroup{
        {{"name", "%张%", "like"}},
        {{"age", 18, ">"}},
    },
}
// 生成的SQL: WHERE (name LIKE '%张%') AND (age > 18)
```

#### QueryBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| Page | query (Query) | (*PaginatedResult[T], error) | 分页查询，返回分页信息和数据列表 |
| Get | query (Query) | ([]T, error) | 获取符合条件的所有记录 |
| First | query (Query) | (*T, error) | 获取符合条件的第一条记录 |
| Find | id (interface{}), query (Query) | (*T, error) | 根据 ID 查询记录，支持额外查询条件 |
| Count | query (Query) | (int64, error) | 统计符合条件的记录数 |
| Sum | field (string), query (Query) | (float64, error) | 对指定字段求和 |
| Exists | query (Query) | (bool, error) | 检查是否存在符合条件的记录 |
| ExistsById | id (interface{}) | (bool, error) | 检查指定ID的记录是否存在 |

### 更新记录

使用 `UpdateBuilder` 更新记录，支持泛型类型安全：

```go
// 更新数据
updateData := User{
    Name: "李四",
    Age: 30,
}

// 根据 ID 更新，直接传入模型类型
success, err := db.UpdateBuilder[User]{}.UpdateByID(1, updateData)
if err != nil {
    // 处理错误
}
if success {
    fmt.Println("更新成功")
}

// 根据条件更新
query := db.Query{
    Search: []db.ConditionGroup{
        {{"age", 25, "<"}},
    },
}
success, err = db.UpdateBuilder[User]{}.Update(query, updateData)
if err != nil {
    // 处理错误
}

// 在事务中更新
tx := db.DB.Begin()
success, err = db.UpdateBuilder[User]{TX: tx}.UpdateByID(1, updateData)
if err != nil {
    tx.Rollback()
    return err
}
tx.Commit()
```

#### UpdateBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| UpdateByID | id (interface{}), data (T) | (bool, error) | 根据 ID 更新记录 |
| Update | query (Query), data (T) | (bool, error) | 根据条件更新记录 |

### 删除记录

使用 `DeleteBuilder` 删除记录，支持泛型类型安全：

```go
// 根据 ID 删除
success, err := db.DeleteBuilder[User]{}.DeleteByID(1)
if err != nil {
    // 处理错误
}
if success {
    fmt.Println("删除成功")
}

// 根据条件删除
query := db.Query{
    Search: []db.ConditionGroup{
        {{"age", 18, "<"}},
    },
}
success, err = db.DeleteBuilder[User]{}.Delete(query)
if err != nil {
    // 处理错误
}

// 在事务中删除
tx := db.DB.Begin()
success, err = db.DeleteBuilder[User]{TX: tx}.DeleteByID(1)
if err != nil {
    tx.Rollback()
    return err
}
tx.Commit()
```

#### DeleteBuilder 方法

| 方法 | 参数 | 返回值 | 说明 |
|------|------|------|------|
| DeleteByID | id (interface{}) | (bool, error) | 根据 ID 删除记录 |
| Delete | query (Query) | (bool, error) | 根据条件删除记录 |

## 事务处理

### 使用 Transaction 方法

使用 `db.Transaction` 方法可以自动管理事务生命周期，无需手动处理 Begin/Commit/Rollback：

```go
// 自动管理事务生命周期
err := db.DB.Transaction(func(tx *gorm.DB) error {
    // 创建用户
    user := User{Name: "王五", Email: "wangwu@example.com"}
    createdUser, err := db.CreateBuilder[User]{TX: tx}.Create(user)
    if err != nil {
        return err // 自动回滚
    }
    
    // 更新用户
    updateData := User{Age: 30}
    success, err := db.UpdateBuilder[User]{TX: tx}.UpdateByID(createdUser.ID, updateData)
    if err != nil {
        return err // 自动回滚
    }
    
    // 查询验证
    query := db.Query{Filter: []string{"id", "name", "age"}}
    updatedUser, err := db.QueryBuilder[User]{TX: tx}.Find(createdUser.ID, query)
    if err != nil {
        return err // 自动回滚
    }
    
    fmt.Printf("事务完成，用户: %+v\n", updatedUser)
    return nil // 自动提交
})

if err != nil {
    log.Printf("事务失败: %v", err)
}
```

### 手动管理事务

如果需要更细粒度的控制，可以手动管理事务：

```go
// 手动管理事务
tx := db.DB.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

// 使用事务版本的 Builder - 设置 TX 字段
user := User{Name: "王五", Email: "wangwu@example.com"}
createdUser, err := db.CreateBuilder[User]{TX: tx}.Create(user)
if err != nil {
    tx.Rollback()
    return err
}

// 更新操作
updateData := User{Age: 30}
success, err := db.UpdateBuilder[User]{TX: tx}.UpdateByID(createdUser.ID, updateData)
if err != nil {
    tx.Rollback()
    return err
}

// 查询操作
updatedUser, err := db.QueryBuilder[User]{TX: tx}.Find(createdUser.ID, db.Query{})
if err != nil {
    tx.Rollback()
    return err
}

// 提交事务
tx.Commit()
```

## 查询解析器

为了提高代码的可维护性和模块化程度，查询解析功能已被拆分为独立的解析器模块。`QueryParser` 负责将 `Query` 结构体解析为 GORM 查询条件。

### 模块化设计

查询解析器采用模块化设计，分为以下几个独立文件：

- **db-parse-query.go**: 主解析器，协调各个子解析器
- **db-parse-filter.go**: 字段过滤解析器
- **db-parse-search.go**: 搜索条件解析器  
- **db-parse-order.go**: 排序、限制和分页解析器

### QueryParser 结构

```go
type QueryParser[T IModel] struct {
    TX *gorm.DB // 可选的事务连接
}
```

### 安全特性

查询解析器内置了多项安全特性：

1. **SQL 注入防护**: 所有字段名都经过验证，只允许字母、数字、下划线和点号
2. **操作符验证**: 只允许预定义的安全操作符
3. **类型安全**: 使用泛型确保类型安全，避免运行时类型错误
4. **参数化查询**: 所有值都通过参数化查询传递，避免 SQL 注入

### 字段名验证

```go
// 有效的字段名示例
validFields := []string{
    "name",           // 简单字段
    "user_id",        // 下划线分隔
    "profile.email",  // 关联字段
    "created_at",     // 时间字段
}

// 无效的字段名（会被拒绝）
invalidFields := []string{
    "name; DROP TABLE users;", // SQL 注入尝试
    "name OR 1=1",             // 恶意条件
    "name--",                  // SQL 注释
}
```

### 操作符支持

支持的安全操作符包括：

```go
// 比较操作符
"=", "!=", "<>", ">", ">=", "<", "<="

// 模糊查询
"LIKE", "NOT LIKE"

// 范围查询  
"IN", "NOT IN", "BETWEEN", "NOT BETWEEN"

// 空值查询
"IS NULL", "IS NOT NULL"
```

### 错误处理

查询解析器提供详细的错误信息：

```go
// 字段名验证错误
"invalid field name: malicious_field"

// 操作符验证错误  
"invalid operator: 'UNION' is not a valid operator"

// 条件格式错误
"invalid condition: must have at least field and value"

// 类型断言错误
"invalid condition: field must be string"
```

### 性能优化

1. **COUNT 查询优化**: COUNT 查询不包含排序和分页，提高性能
2. **字段过滤**: 只查询需要的字段，减少数据传输
3. **分页计算优化**: 修复了分页计算错误，确保正确的页数计算
4. **COALESCE 优化**: SUM 查询使用 `COALESCE(SUM(field), 0)` 确保无结果时返回 0

### 使用示例

```go
// 复杂查询示例
query := db.Query{
    Filter: []string{"id", "name", "email", "created_at"},
    Search: []db.ConditionGroup{
        // 第一组条件：名字包含"张"或"李"
        {
            {"name", "%张%", "like"},
            {"name", "%李%", "like"},
        },
        // 第二组条件：年龄在18-65之间
        {
            {"age", []int{18, 65}, "between"},
        },
        // 第三组条件：状态为活跃
        {
            {"active", true},
        },
    },
    Required: []string{"email"}, // 邮箱不能为空
    OrderBy: []string{"created_at", "desc"},
    Page: []int{1, 20},
}

// 执行查询
result, err := db.QueryBuilder[User]{}.GetWithPager(query)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

// 生成的 SQL 类似于：
// SELECT `id`, `name`, `email`, `created_at` FROM `users` 
// WHERE (`email` IS NOT NULL AND `email` != '') 
// AND (`name` LIKE '%张%' OR `name` LIKE '%李%') 
// AND (`age` BETWEEN 18 AND 65) 
// AND (`active` = true)
// ORDER BY `created_at` DESC 
// LIMIT 20 OFFSET 0
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
