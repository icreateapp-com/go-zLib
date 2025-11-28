# 查询操作 (db-query.md)

## 概述

`QueryBuilder` 是数据库模块的核心查询构建器，提供了完整的查询功能，包括条件查询、分页查询、统计查询、排序等。支持复杂的查询条件构建和类型安全的操作。

## QueryBuilder 结构

```go
type QueryBuilder[T IModel] struct {
    TX    *gorm.DB // 可选的事务对象
    Query Query    // 查询参数，在初始化时指定
}
```

## 初始化 QueryBuilder

```go
// 基本初始化
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Filter: []string{"id", "name", "email"},
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 1},
                },
            },
        },
    },
}

// 在事务中初始化
queryBuilder := db.QueryBuilder[User]{
    TX: tx, // 事务对象
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"age", 18, ">="},
                },
            },
        },
    },
}
```

## Query 查询参数

```go
type Query struct {
    Filter   []string         `json:"filter"`   // 字段过滤列表
    Search   []ConditionGroup `json:"search"`   // 搜索条件组
    OrderBy  [][]string       `json:"orderby"`  // 排序条件
    Limit    []int            `json:"limit"`    // 限制条数
    Page     []int            `json:"page"`     // 分页参数 [页码, 每页数量]
    Required []string         `json:"required"` // 必需字段（非空）
    Include  []string         `json:"include"`  // 关联预加载
}
```

## 基本查询方法

### 1. Get - 查询多条记录

```go
func (qb QueryBuilder[T]) Get(dest interface{}) error
```

**功能说明：**
- 查询多条记录并将结果写入 `dest` 参数
- `dest` 必须是指向切片的指针，如 `*[]User`
- 使用 QueryBuilder 初始化时指定的查询条件

**基本用法：**
```go
// 初始化查询构建器
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Filter: []string{"id", "name", "email"}, // 只查询指定字段
        Search: []db.ConditionGroup{
            {
                Conditions: [][] interface{}{
                    {"status", 1},
                    {"age", 18, ">="},
                },
            },
        },
        OrderBy: [][]string{
            {"created_at", "desc"},
            {"name", "asc"},
        },
        Limit: []int{10},
    },
}

// 执行查询
var users []User
err := queryBuilder.Get(&users)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

fmt.Printf("查询到 %d 个用户\n", len(users))
for _, user := range users {
    fmt.Printf("用户: %s (%s)\n", user.Name, user.Email)
}
```

**JSON 请求格式：**
```json
{
  "filter": ["id", "name", "email"],
  "search": [
    {
      "operator": "and",
      "conditions": [
        ["status", 1],
        ["age", 18, ">="]
      ]
    }
  ],
  "orderby": [
    ["created_at", "desc"],
    ["name", "asc"]
  ],
  "limit": [10]
}
```

### 2. Page - 分页查询

```go
func (qb QueryBuilder[T]) Page(pager *Pager) error
```

**功能说明：**
- 执行分页查询并将结果写入 `pager.Data` 参数
- `pager.Data` 必须是指向切片的指针，如 `*[]User`
- 分页信息直接更新到 `pager` 对象中

**基本用法：**
```go
// 初始化分页查询构建器
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 1},
                },
            },
        },
        OrderBy: [][]string{
            {"created_at", "desc"},
        },
        Page: []int{1, 20}, // 第1页，每页20条
    },
}

// 执行分页查询
var users []User
pager := &db.Pager{
    Data: &users, // 设置数据接收容器
}
err := queryBuilder.Page(pager)
if err != nil {
    log.Printf("分页查询失败: %v", err)
    return
}

// 处理分页结果
fmt.Printf("当前页: %d\n", pager.Page)
fmt.Printf("每页大小: %d\n", pager.PageSize)
fmt.Printf("总记录数: %d\n", pager.Total)
fmt.Printf("当前页数据量: %d\n", len(users))

for _, user := range users {
    fmt.Printf("用户: %s\n", user.Name)
}
```

**自定义结构体示例：**
```go
// 可以使用自定义结构体接收数据
var userInfos []struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

pager := &db.Pager{
    Data: &userInfos, // 使用自定义结构体切片
}
err := queryBuilder.Page(pager)
```

**动态类型切换示例：**
```go
// 可以在运行时动态改变 Data 的类型
var simpleUsers []struct {
    Name string `json:"name"`
}

// 重用同一个 pager 对象，但改变 Data 类型
pager.Data = &simpleUsers
err := queryBuilder.Page(pager)
```

**分页信息结构：**
```go
type Pager struct {
    Page     int         `json:"page"`      // 当前页
    PageSize int         `json:"page_size"` // 每页大小
    Total    int         `json:"total"`     // 总记录数
    LastPage int         `json:"last_page"` // 最后一页
    Data     interface{} `json:"data"`      // 分页数据，可以是任何类型的切片
}
```

**JSON 响应格式：**
```json
{
  "data": [
    {
      "id": 1,
      "name": "张三",
      "email": "zhangsan@example.com",
      "status": 1,
      "created_at": "2023-12-01 10:30:00"
    }
  ],
  "pager": {
    "page": 1,
    "page_size": 20,
    "total": 100
  }
}
```

### 3. First - 查询单条记录

```go
func (qb QueryBuilder[T]) First(dest interface{}) error
```

**功能说明：**
- 查询第一条符合条件的记录并写入 `dest` 参数
- `dest` 必须是指向结构体的指针，如 `*User`
- 如果没有找到记录，返回 `gorm.ErrRecordNotFound` 错误

**基本用法：**
```go
// 初始化查询构建器
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"email", "zhangsan@example.com"},
                },
            },
        },
    },
}

// 查询第一条记录
var user User
err := queryBuilder.First(&user)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        fmt.Println("用户不存在")
    } else {
        log.Printf("查询失败: %v", err)
    }
    return
}

fmt.Printf("找到用户: %s\n", user.Name)
```

### 4. Find - 根据主键查询

```go
func (qb QueryBuilder[T]) Find(id interface{}, dest interface{}) error
```

**功能说明：**
- 根据主键查询记录并写入 `dest` 参数
- `dest` 必须是指向结构体的指针，如 `*User`
- 可以结合 QueryBuilder 中的其他查询条件（如字段过滤）

**基本用法：**
```go
// 初始化查询构建器（可指定字段过滤）
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Filter: []string{"id", "name", "email"}, // 只查询指定字段
    },
}

// 根据 ID 查询
var user User
err := queryBuilder.Find(123, &user)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        fmt.Println("用户不存在")
    } else {
        log.Printf("查询失败: %v", err)
    }
    return
}

fmt.Printf("用户: %s (%s)\n", user.Name, user.Email)
```

**UUID 主键示例：**
```go
queryBuilder := db.QueryBuilder[Session]{
    Query: db.Query{
        Filter: []string{"id", "user_id", "expires_at"},
    },
}
var session Session
err := queryBuilder.Find("550e8400-e29b-41d4-a716-446655440000", &session)
```
```

## 统计查询方法

### 1. Count - 统计记录数量

```go
func (qb QueryBuilder[T]) Count() (int64, error)
```

**功能说明：**
- 统计符合条件的记录数量
- 使用 QueryBuilder 初始化时指定的查询条件

**基本用法：**
```go
// 统计所有用户
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{},
}
count, err := queryBuilder.Count()

// 统计活跃用户
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 1},
                    {"last_login_at", time.Now().AddDate(0, 0, -30), ">="},
                },
            },
        },
    },
}
count, err := queryBuilder.Count()

fmt.Printf("活跃用户数量: %d\n", count)
```

### 2. Sum - 计算字段总和

```go
func (qb QueryBuilder[T]) Sum(field string) (float64, error)
```

**功能说明：**
- 计算指定字段的总和
- 使用 QueryBuilder 初始化时指定的查询条件

**基本用法：**
```go
// 计算所有用户年龄总和
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{},
}
totalAge, err := queryBuilder.Sum("age")

// 计算活跃用户年龄总和
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 1},
                },
            },
        },
    },
}
totalAge, err := queryBuilder.Sum("age")

fmt.Printf("活跃用户年龄总和: %.2f\n", totalAge)
```

### 3. Exists 和 ExistsById - 检查记录是否存在

```go
func (qb QueryBuilder[T]) Exists() (bool, error)
func (qb QueryBuilder[T]) ExistsById(id interface{}) (bool, error)
```

**功能说明：**
- `Exists()`: 检查是否存在符合 QueryBuilder 条件的记录
- `ExistsById()`: 检查指定主键的记录是否存在

**基本用法：**
```go
// 检查邮箱是否已存在
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"email", "test@example.com"},
                },
            },
        },
    },
}
exists, err := queryBuilder.Exists()

if exists {
    fmt.Println("邮箱已存在")
}

// 检查用户 ID 是否存在
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{},
}
exists, err := queryBuilder.ExistsById(123)
```

## 查询条件详解

### 1. Filter - 字段过滤

`Filter` 字段用于指定要查询的字段列表，类似于 SQL 的 `SELECT` 子句：

```go
// 只查询指定字段
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Filter: []string{"id", "name", "email"},
    },
}
var users []User
err := queryBuilder.Get(&users)
```

**生成的 SQL：**
```sql
SELECT `id`, `name`, `email` FROM `users`
```

### 2. Search - 搜索条件

`Search` 字段包含条件组数组，每个条件组内的条件默认使用 `AND` 连接，条件组之间使用 `AND` 连接：

```go
type ConditionGroup struct {
    Conditions [][]interface{} `json:"conditions"` // 条件列表
    Operator   string          `json:"operator"`   // 组内操作符 "and" 或 "or"
}
```

**条件格式：**
- `[字段名, 值]` - 等值查询（默认 "=" 操作符）
- `[字段名, 值, 操作符]` - 指定操作符的查询

**支持的操作符：**
- 比较：`=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`
- 模糊：`like`, `not like`, `left like`, `right like`
- 范围：`in`, `not in`, `between`, `not between`
- 空值：`is null`, `is not null`

**示例：**
```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                // AND 条件组（默认）
                Conditions: [][]interface{}{
                    {"status", 1},                    // status = 1
                    {"age", 18, ">="},               // age >= 18
                    {"name", "%张%", "like"},         // name LIKE '%张%'
                    {"email", "@gmail.com", "right like"}, // email LIKE '%@gmail.com'
                },
            },
            {
                // OR 条件组
                Operator: "or",
                Conditions: [][]interface{}{
                    {"vip_level", 0, ">"},           // vip_level > 0
                    {"total_orders", 10, ">="},     // total_orders >= 10
                },
            },
        },
    },
}
var users []User
err := queryBuilder.Get(&users)
```

**生成的 SQL：**
```sql
SELECT * FROM `users` 
WHERE (
    `status` = 1 
    AND `age` >= 18 
    AND `name` LIKE '%张%' 
    AND `email` LIKE '%@gmail.com'
) AND (
    `vip_level` > 0 
    OR `total_orders` >= 10
)
```

### 3. OrderBy - 排序条件

`OrderBy` 字段用于指定排序规则：

```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        OrderBy: [][]string{
            {"status", "desc"},      // 按状态降序
            {"created_at", "desc"},  // 按创建时间降序
            {"name", "asc"},         // 按姓名升序
        },
    },
}
var users []User
err := queryBuilder.Get(&users)
```

**生成的 SQL：**
```sql
SELECT * FROM `users` 
ORDER BY `status` DESC, `created_at` DESC, `name` ASC
```

### 4. Limit - 限制条数

`Limit` 字段用于限制返回的记录数量：

```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Limit: []int{10}, // 限制返回10条记录
    },
}
var users []User
err := queryBuilder.Get(&users)
```

### 5. Page - 分页参数

`Page` 字段用于分页查询，格式为 `[页码, 每页数量]`：

```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Page: []int{2, 15}, // 第2页，每页15条
    },
}
var users []User
pager := &db.Pager{
    Data: &users, // 设置数据接收容器
}
err := queryBuilder.Page(pager)
```

### 6. Required - 必需字段

`Required` 字段用于指定必须非空的字段：

```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Required: []string{"email", "phone"}, // email 和 phone 必须非空
    },
}
var users []User
err := queryBuilder.Get(&users)
```

**生成的 SQL：**
```sql
SELECT * FROM `users` 
WHERE `email` IS NOT NULL AND `phone` IS NOT NULL
```

### 7. Include - 关联预加载

`Include` 字段用于预加载关联数据，避免 N+1 查询问题：

```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Include: []string{"Profile", "Orders", "Orders.Items"},
    },
}
var users []User
err := queryBuilder.Get(&users)
```

## 复杂查询示例

### 1. 用户活跃度查询

```go
// 查询最近30天活跃的VIP用户
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Filter: []string{"id", "name", "email", "vip_level", "last_login_at"},
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 1},                                    // 状态正常
                    {"vip_level", 0, ">"},                           // VIP用户
                    {"last_login_at", time.Now().AddDate(0, 0, -30), ">="}, // 最近30天登录
                },
            },
        },
        OrderBy: [][]string{
            {"vip_level", "desc"},
            {"last_login_at", "desc"},
        },
        Limit: []int{50},
    },
}

var activeVipUsers []User
err := queryBuilder.Get(&activeVipUsers)
```

### 2. 文章搜索查询

```go
// 搜索已发布的文章
articleBuilder := db.QueryBuilder[Article]{
    Query: db.Query{
        Filter: []string{"id", "title", "summary", "author_id", "view_count", "published_at"},
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 1},                    // 已发布
                    {"published_at", nil, "is not null"}, // 有发布时间
                },
            },
            {
                Operator: "or",
                Conditions: [][]interface{}{
                    {"title", "%Go语言%", "like"},    // 标题包含关键词
                    {"summary", "%Go语言%", "like"},  // 摘要包含关键词
                },
            },
        },
        OrderBy: [][]string{
            {"published_at", "desc"},
            {"view_count", "desc"},
        },
        Include: []string{"Author", "Category"},
        Page: []int{1, 20},
    },
}

var articles []Article
pager := &db.Pager{
    Data: &articles, // 设置数据接收容器
}
err := articleBuilder.Page(pager)
```

### 3. 订单统计查询

```go
// 统计本月已完成订单的总金额
thisMonth := time.Now().Format("2006-01")
orderBuilder := db.QueryBuilder[Order]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"status", 3},                           // 已完成状态
                    {"created_at", thisMonth+"%", "like"},   // 本月创建
                },
            },
        },
    },
}

totalAmount, err := orderBuilder.Sum("total_amount")
fmt.Printf("本月已完成订单总金额: %.2f\n", totalAmount)

// 统计本月订单数量
countBuilder := db.QueryBuilder[Order]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"created_at", thisMonth+"%", "like"}, // 本月创建
                },
            },
        },
    },
}

orderCount, err := countBuilder.Count()
fmt.Printf("本月订单数量: %d\n", orderCount)
```

## 事务中的查询

在事务中使用查询构建器：

```go
err := db.DB.Transaction(func(tx *gorm.DB) error {
    // 在事务中创建查询构建器
    queryBuilder := db.QueryBuilder[User]{
        TX: tx,
        Query: db.Query{
            Filter: []string{"id", "name", "email"},
        },
    }
    
    // 查询用户
    var user User
    err := queryBuilder.Find(123, &user)
    if err != nil {
        return err
    }
    
    // 其他事务操作...
    return nil
})
    return nil
})
```

## 错误处理

```go
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"email", "test@example.com"},
                },
            },
        },
    },
}

var user User
err := queryBuilder.First(&user)

if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        // 记录不存在
        fmt.Println("用户不存在")
    } else {
        // 其他错误
        log.Printf("查询失败: %v", err)
    }
    return
}

// 处理查询结果
fmt.Printf("用户: %s\n", user.Name)
```

## 性能优化建议

1. **使用字段过滤** - 只查询需要的字段
2. **添加适当索引** - 为经常查询的字段添加索引
3. **使用关联预加载** - 避免 N+1 查询问题
4. **限制结果数量** - 使用 `Limit` 限制返回记录数
5. **优化查询条件** - 将选择性高的条件放在前面
6. **使用分页查询** - 避免一次性加载大量数据

## 注意事项

1. **字段名验证** - 所有字段名都会经过验证，只允许字母、数字、下划线和点号
2. **SQL 注入防护** - 所有值都通过参数化查询传递，确保安全性
3. **分页从1开始** - 分页页码从第1页开始计算
4. **操作符大小写** - 操作符不区分大小写
5. **空值处理** - 正确处理 `nil` 值和空字符串的区别