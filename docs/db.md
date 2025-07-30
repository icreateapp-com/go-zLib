# 数据库模块 (DB) 概要说明

## 概述

数据库模块是一个基于 GORM 的高级数据库操作封装，提供了类型安全的 CRUD 操作、灵活的查询构建、事务支持、分页查询等功能。该模块采用泛型设计，支持复杂的查询条件构建，并内置了 SQL 注入防护机制。

## 模块架构

### 核心组件

- **QueryBuilder** - 查询构建器，支持复杂查询、分页、统计等操作
- **CreateBuilder** - 创建构建器，用于数据创建操作
- **UpdateBuilder** - 更新构建器，用于数据更新操作
- **DeleteBuilder** - 删除构建器，用于数据删除操作
- **QueryParser** - 查询解析器，负责将查询条件解析为 GORM 查询

### 文件结构

```
z/db/
├── db.go                    # 数据库连接和初始化
├── model.go                 # 模型基类和接口定义
├── mysql.go                 # MySQL 驱动配置
├── timestamp.go             # 时间戳字段处理
├── json.go                  # JSON 字段处理
├── db-query-builder.go      # 查询构建器
├── db-create-builder.go     # 创建构建器
├── db-update-builder.go     # 更新构建器
├── db-delete-builder.go     # 删除构建器
├── db-query-helper.go       # 查询条件辅助器
├── db-parse-query.go        # 查询解析器主入口
├── db-parse-filter.go       # 字段过滤解析器
├── db-parse-search.go       # 搜索条件解析器
└── db-parse-order.go        # 排序和分页解析器
```

## 快速开始

### 1. 初始化数据库连接

```go
package main

import (
    "github.com/icreateapp-com/go-zLib/z/db"
)

func main() {
    // 初始化数据库连接
    db.DB.Init()
}
```

### 2. 定义模型

```go
// 用户模型
type User struct {
    db.AutoIncrement // 自增主键
    db.Timestamp     // 时间戳字段
    Name     string `json:"name" gorm:"size:100;not null"`
    Email    string `json:"email" gorm:"size:255;uniqueIndex"`
    Age      int    `json:"age"`
    Status   int    `json:"status" gorm:"default:1"`
}

func (User) TableName() string {
    return "users"
}
```

### 3. 基本操作示例

```go
// 创建用户
createBuilder := db.CreateBuilder[User]{}
user, err := createBuilder.Create(User{
    Name:  "张三",
    Email: "zhangsan@example.com",
    Age:   25,
})

// 查询用户
queryBuilder := db.QueryBuilder[User]{
    Query: db.Query{
        Filter: []string{"id", "name", "email"}, // 只查询指定字段
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"age", 25, ">="},
                    {"status", 1},
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

// 查询多条记录
var users []User
err := queryBuilder.Get(&users)

// 分页查询
queryBuilder.Query = db.Query{
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{
                {"status", 1},
            },
        },
    },
    Page: []int{1, 20}, // 第1页，每页20条
}
var pageUsers []User
pager := &db.Pager{
    Data: &pageUsers, // 设置数据接收容器
}
err = queryBuilder.Page(pager)
```

## 核心数据结构

### Query 查询参数

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

### ConditionGroup 条件组

```go
type ConditionGroup struct {
    Conditions [][]interface{} `json:"conditions"` // 条件列表
    Operator   string          `json:"operator"`   // 组内操作符 "and" 或 "or"
}
```

### 条件格式

每个条件是一个 `[]interface{}` 数组：
- `[字段名, 值]` - 等值查询（默认 "=" 操作符）
- `[字段名, 值, 操作符]` - 指定操作符的查询

支持的操作符：
- 比较：`=`, `!=`, `<>`, `>`, `>=`, `<`, `<=`
- 模糊：`like`, `not like`, `left like`, `right like`
- 范围：`in`, `not in`, `between`, `not between`
- 空值：`is null`, `is not null`

## 详细文档

- [模型定义 (db-model.md)](./db-model.md) - 数据模型的定义和使用
- [查询操作 (db-query.md)](./db-query.md) - 数据查询的详细说明
- [创建操作 (db-create.md)](./db-create.md) - 数据创建的详细说明
- [更新操作 (db-update.md)](./db-update.md) - 数据更新的详细说明
- [删除操作 (db-delete.md)](./db-delete.md) - 数据删除的详细说明
- [事务处理 (db-transaction.md)](./db-transaction.md) - 事务管理的详细说明
- [配置和连接](./db-config.md) - 数据库配置和连接管理
- [安全特性](./db-security.md) - SQL 注入防护和安全机制
- [性能优化](./db-performance.md) - 性能优化技巧和最佳实践

## JSON API 格式

### 查询请求示例

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
  "page": [1, 20]
}
```

### 分页响应示例

```json
{
  "data": [
    {
      "id": 1,
      "name": "张三",
      "email": "zhangsan@example.com",
      "age": 25,
      "status": 1,
      "created_at": "2023-12-01 10:30:00",
      "updated_at": "2023-12-01 10:30:00"
    }
  ],
  "pager": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "last_page": 5
  }
}
```

## 安全特性

- **SQL 注入防护** - 所有字段名和操作符都经过验证
- **参数化查询** - 所有值都通过参数化查询传递
- **类型安全** - 使用 Go 泛型确保类型安全
- **输入验证** - 严格的字段名和操作符验证

## 最佳实践

1. **模型设计** - 使用内置的模型组件（AutoIncrement、Timestamp 等）
2. **错误处理** - 正确处理 `gorm.ErrRecordNotFound` 等错误
3. **事务使用** - 在需要原子性操作时使用事务
4. **性能优化** - 使用字段过滤、索引查询、关联预加载等
5. **安全考虑** - 验证输入数据，使用参数化查询

## 注意事项

- 所有模型必须实现 `IModel` 接口
- 字段名只能包含字母、数字、下划线和点号
- 分页从第 1 页开始计算
- 事务中的操作需要传递 `TX` 参数
- 建议在生产环境中关闭调试模式以提高性能
