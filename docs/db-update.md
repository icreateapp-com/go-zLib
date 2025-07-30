# 更新操作 (db-update.md)

## 概述

`UpdateBuilder` 提供了类型安全的数据更新功能，支持根据条件更新和根据主键更新两种方式。所有更新操作都会自动处理 `UpdatedAt` 时间戳字段。

## UpdateBuilder 结构

```go
type UpdateBuilder[T IModel] struct {
    TX *gorm.DB // 可选的事务对象
}
```

## 基本更新方法

### 1. Update - 根据条件更新

```go
func (ub UpdateBuilder[T]) Update(query Query, data T) (bool, error)
```

**基本用法：**
```go
updateBuilder := db.UpdateBuilder[User]{}

// 更新符合条件的用户
success, err := updateBuilder.Update(
    db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"age", 30, "<"},
                    {"status", 1},
                },
            },
        },
    },
    User{
        Status: 2, // 更新状态为2
    },
)

if err != nil {
    log.Printf("更新失败: %v", err)
    return
}

if success {
    fmt.Println("更新成功")
} else {
    fmt.Println("没有记录被更新")
}
```

### 2. UpdateByID - 根据主键更新

```go
func (ub UpdateBuilder[T]) UpdateByID(id interface{}, data T) (bool, error)
```

**基本用法：**
```go
// 根据 ID 更新用户
success, err := updateBuilder.UpdateByID(123, User{
    Name: "更新后的名字",
    Age:  35,
})

if err != nil {
    log.Printf("更新失败: %v", err)
    return
}

if success {
    fmt.Println("用户更新成功")
} else {
    fmt.Println("用户不存在或更新失败")
}
```

## 不同主键类型的更新

### 1. 自增主键更新

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name   string `json:"name"`
    Email  string `json:"email"`
    Status int    `json:"status"`
}

// 使用整型 ID 更新
success, err := updateBuilder.UpdateByID(123, User{
    Name:   "新名字",
    Status: 1,
})
```

### 2. UUID 主键更新

```go
type Session struct {
    db.Uuid
    db.Timestamp
    UserID int64  `json:"user_id"`
    Token  string `json:"token"`
    Status int    `json:"status"`
}

// 使用 UUID 更新
sessionBuilder := db.UpdateBuilder[Session]{}
success, err := sessionBuilder.UpdateByID("550e8400-e29b-41d4-a716-446655440000", Session{
    Status: 0, // 设置为过期状态
})
```

## 复杂更新操作

### 1. 部分字段更新

```go
// 只更新指定字段，其他字段保持不变
success, err := updateBuilder.UpdateByID(123, User{
    Name: "新名字", // 只更新名字字段
    // 其他字段不会被更新
})
```

### 2. JSON 字段更新

```go
type Product struct {
    db.AutoIncrement
    db.Timestamp
    Name     string       `json:"name"`
    Metadata db.JsonField `json:"metadata" gorm:"type:json"`
}

// 更新包含 JSON 数据的产品
productBuilder := db.UpdateBuilder[Product]{}
success, err := productBuilder.UpdateByID(456, Product{
    Metadata: db.JsonField{
        Data: map[string]interface{}{
            "color":   "红色",
            "storage": "256GB",
            "specs": map[string]interface{}{
                "screen": "6.7英寸",
                "chip":   "A17 Pro",
            },
            "updated_features": []string{
                "新增功能1",
                "新增功能2",
            },
        },
    },
})
```

### 3. 条件更新示例

```go
// 批量更新用户状态
success, err := updateBuilder.Update(
    db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"last_login_at", time.Now().AddDate(0, 0, -90), "<"}, // 90天未登录
                    {"status", 1}, // 当前状态为活跃
                },
            },
        },
    },
    User{
        Status: 0, // 设置为非活跃状态
    },
)

fmt.Printf("更新了 %d 个用户的状态\n", success)
```

## 使用 GORM 表达式更新

对于需要基于当前值进行计算的更新，可以使用 GORM 表达式：

```go
// 增加用户积分
success, err := updateBuilder.UpdateByID(123, User{
    Points: gorm.Expr("points + ?", 100), // 积分增加100
})

// 减少库存
success, err := productBuilder.Update(
    db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"id", 456},
                    {"stock", 0, ">"}, // 确保库存大于0
                },
            },
        },
    },
    Product{
        Stock: gorm.Expr("stock - ?", 1), // 库存减1
    },
)
```

## 事务中的更新

在事务中使用更新构建器：

```go
err := db.DB.Transaction(func(tx *gorm.DB) error {
    updateBuilder := db.UpdateBuilder[User]{TX: tx}
    
    // 更新用户信息
    success, err := updateBuilder.UpdateByID(123, User{
        Name:   "新名字",
        Status: 1,
    })
    if err != nil {
        return err
    }
    if !success {
        return errors.New("用户不存在")
    }
    
    // 其他事务操作...
    return nil
})

if err != nil {
    log.Printf("事务失败: %v", err)
}
```

## 更新前的数据验证

### 1. 检查记录是否存在

```go
func UpdateUserSafely(userID int64, updateData User) error {
    // 检查记录是否存在
    queryBuilder := db.QueryBuilder[User]{
        Query: db.Query{
            Filter: []string{"id"},
        },
    }
    exists, err := queryBuilder.ExistsById(userID)
    if err != nil {
        return fmt.Errorf("检查用户存在性失败: %w", err)
    }
    if !exists {
        return errors.New("用户不存在")
    }
    
    // 执行更新
    updateBuilder := db.UpdateBuilder[User]{}
    success, err := updateBuilder.UpdateByID(userID, updateData)
    if err != nil {
        return fmt.Errorf("更新用户失败: %w", err)
    }
    if !success {
        return errors.New("更新失败")
    }
    
    return nil
}
```

### 2. 数据验证

```go
func UpdateUserWithValidation(userID int64, updateData User) error {
    // 验证更新数据
    if updateData.Name != "" && len(updateData.Name) > 100 {
        return errors.New("用户名长度不能超过100字符")
    }
    
    if updateData.Email != "" {
        emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
        if !emailRegex.MatchString(updateData.Email) {
            return errors.New("邮箱格式不正确")
        }
        
        // 检查邮箱是否已被其他用户使用
        queryBuilder := db.QueryBuilder[User]{
            Query: db.Query{
                Search: []db.ConditionGroup{
                    {
                        Conditions: [][]interface{}{
                            {"email", updateData.Email},
                            {"id", userID, "!="},
                        },
                    },
                },
            },
        }
        exists, err := queryBuilder.Exists()
        if err != nil {
            return fmt.Errorf("检查邮箱重复失败: %w", err)
        }
        if exists {
            return errors.New("邮箱已被其他用户使用")
        }
    }
    
    // 执行更新
    updateBuilder := db.UpdateBuilder[User]{}
    success, err := updateBuilder.UpdateByID(userID, updateData)
    if err != nil {
        return fmt.Errorf("更新用户失败: %w", err)
    }
    if !success {
        return errors.New("用户不存在或更新失败")
    }
    
    return nil
}
```

## GORM 钩子函数

可以在模型中定义钩子函数，在更新前后执行自定义逻辑：

### 1. BeforeUpdate 钩子

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Version  int    `json:"version"` // 版本号，用于乐观锁
}

// 更新前钩子：版本号递增
func (u *User) BeforeUpdate(tx *gorm.DB) error {
    u.Version++
    
    // 如果密码有变化，重新加密
    if u.Password != "" {
        hashedPassword, err := hashPassword(u.Password)
        if err != nil {
            return err
        }
        u.Password = hashedPassword
    }
    
    return nil
}
```

### 2. AfterUpdate 钩子

```go
// 更新后钩子：记录操作日志
func (u *User) AfterUpdate(tx *gorm.DB) error {
    // 记录更新日志
    go logUserUpdate(u.ID, u.Name)
    return nil
}

func logUserUpdate(userID int64, userName string) {
    log.Printf("用户更新: ID=%d, Name=%s, Time=%s", userID, userName, time.Now().Format("2006-01-02 15:04:05"))
}
```

## 乐观锁更新

使用版本号实现乐观锁：

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name    string `json:"name"`
    Email   string `json:"email"`
    Version int    `json:"version" gorm:"default:1"`
}

func UpdateUserWithOptimisticLock(userID int64, updateData User, expectedVersion int) error {
    updateBuilder := db.UpdateBuilder[User]{}
    
    // 带版本号的条件更新
    success, err := updateBuilder.Update(
        db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"id", userID},
                        {"version", expectedVersion}, // 检查版本号
                    },
                },
            },
        },
        User{
            Name:    updateData.Name,
            Email:   updateData.Email,
            Version: expectedVersion + 1, // 版本号递增
        },
    )
    
    if err != nil {
        return fmt.Errorf("更新失败: %w", err)
    }
    
    if !success {
        return errors.New("数据已被其他用户修改，请刷新后重试")
    }
    
    return nil
}
```

## 错误处理

### 1. 常见错误处理

```go
success, err := updateBuilder.UpdateByID(123, User{
    Name:  "新名字",
    Email: "newemail@example.com",
})

if err != nil {
    // 检查是否是唯一约束错误
    if strings.Contains(err.Error(), "Duplicate entry") {
        fmt.Println("邮箱已存在")
        return
    }
    
    // 检查是否是外键约束错误
    if strings.Contains(err.Error(), "foreign key constraint") {
        fmt.Println("关联数据不存在")
        return
    }
    
    // 其他错误
    log.Printf("更新失败: %v", err)
    return
}

if !success {
    fmt.Println("记录不存在或没有变化")
}
```

### 2. 自定义错误处理

```go
func UpdateUserSafely(userID int64, updateData User) error {
    updateBuilder := db.UpdateBuilder[User]{}
    success, err := updateBuilder.UpdateByID(userID, updateData)
    
    if err != nil {
        // 记录错误日志
        log.Printf("更新用户失败: ID=%d, 数据=%+v, 错误=%v", userID, updateData, err)
        
        // 返回用户友好的错误信息
        if strings.Contains(err.Error(), "Duplicate entry") {
            return errors.New("邮箱或用户名已存在")
        }
        
        return errors.New("更新失败，请稍后重试")
    }
    
    if !success {
        return errors.New("用户不存在")
    }
    
    return nil
}
```

## JSON API 示例

### 1. HTTP 接口实现

```go
// 更新用户的 HTTP 接口
func UpdateUserHandler(c *gin.Context) {
    // 获取用户 ID
    userIDStr := c.Param("id")
    userID, err := strconv.ParseInt(userIDStr, 10, 64)
    if err != nil {
        c.JSON(400, gin.H{
            "error": "Invalid user ID",
        })
        return
    }
    
    // 绑定更新数据
    var updateData User
    if err := c.ShouldBindJSON(&updateData); err != nil {
        c.JSON(400, gin.H{
            "error":   "Invalid JSON data",
            "details": err.Error(),
        })
        return
    }
    
    // 执行更新
    updateBuilder := db.UpdateBuilder[User]{}
    success, err := updateBuilder.UpdateByID(userID, updateData)
    if err != nil {
        c.JSON(500, gin.H{
            "error":   "Failed to update user",
            "details": err.Error(),
        })
        return
    }
    
    if !success {
        c.JSON(404, gin.H{
            "error": "User not found",
        })
        return
    }
    
    // 返回更新后的用户信息
    queryBuilder := db.QueryBuilder[User]{
        Query: db.Query{
            Filter: []string{"id", "name", "email", "updated_at"},
        },
    }
    var user User
    err = queryBuilder.Find(userID, &user)
    if err != nil {
        c.JSON(500, gin.H{
            "error": "Failed to fetch updated user",
        })
        return
    }
    
    c.JSON(200, gin.H{
        "message": "User updated successfully",
        "data":    user,
    })
}
```

### 2. 请求示例

**更新用户请求：**
```json
PUT /api/users/123
Content-Type: application/json

{
  "name": "更新后的名字",
  "email": "newemail@example.com",
  "age": 30
}
```

**成功响应：**
```json
{
  "message": "User updated successfully",
  "data": {
    "id": 123,
    "name": "更新后的名字",
    "email": "newemail@example.com",
    "age": 30,
    "created_at": "2023-12-01 10:30:00",
    "updated_at": "2023-12-01 15:45:00"
  }
}
```

**错误响应：**
```json
{
  "error": "User not found"
}
```

## 批量更新示例

### 1. 批量状态更新

```go
// 批量激活用户
func ActivateUsers(userIDs []int64) error {
    updateBuilder := db.UpdateBuilder[User]{}
    
    success, err := updateBuilder.Update(
        db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"id", userIDs, "in"},
                        {"status", 0}, // 当前为非活跃状态
                    },
                },
            },
        },
        User{
            Status: 1, // 设置为活跃状态
        },
    )
    
    if err != nil {
        return fmt.Errorf("批量激活用户失败: %w", err)
    }
    
    log.Printf("成功激活 %d 个用户", success)
    return nil
}
```

### 2. 条件批量更新

```go
// 更新过期会话状态
func ExpireOldSessions() error {
    sessionBuilder := db.UpdateBuilder[Session]{}
    
    success, err := sessionBuilder.Update(
        db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"expires_at", time.Now(), "<"}, // 已过期
                        {"status", 1}, // 当前为活跃状态
                    },
                },
            },
        },
        Session{
            Status: 0, // 设置为过期状态
        },
    )
    
    if err != nil {
        return fmt.Errorf("更新过期会话失败: %w", err)
    }
    
    log.Printf("成功更新 %d 个过期会话", success)
    return nil
}
```

## 最佳实践

### 1. 数据验证

```go
// 更新前进行完整的数据验证
func UpdateUserWithValidation(userID int64, updateData User) error {
    // 1. 基本字段验证
    if updateData.Name != "" && len(updateData.Name) > 100 {
        return errors.New("用户名长度不能超过100字符")
    }
    
    // 2. 邮箱格式验证
    if updateData.Email != "" {
        emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
        if !emailRegex.MatchString(updateData.Email) {
            return errors.New("邮箱格式不正确")
        }
    }
    
    // 3. 业务逻辑验证
    if updateData.Age != 0 && (updateData.Age < 0 || updateData.Age > 150) {
        return errors.New("年龄必须在0-150之间")
    }
    
    // 4. 执行更新
    updateBuilder := db.UpdateBuilder[User]{}
    success, err := updateBuilder.UpdateByID(userID, updateData)
    if err != nil {
        return err
    }
    if !success {
        return errors.New("用户不存在")
    }
    
    return nil
}
```

### 2. 事务使用

```go
// 更新用户及其相关数据
func UpdateUserAndProfile(userID int64, userData User, profileData Profile) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 更新用户
        userBuilder := db.UpdateBuilder[User]{TX: tx}
        success, err := userBuilder.UpdateByID(userID, userData)
        if err != nil {
            return err
        }
        if !success {
            return errors.New("用户不存在")
        }
        
        // 更新用户资料
        profileBuilder := db.UpdateBuilder[Profile]{TX: tx}
        success, err = profileBuilder.Update(
            db.Query{
                Search: []db.ConditionGroup{
                    {
                        Conditions: [][]interface{}{
                            {"user_id", userID},
                        },
                    },
                },
            },
            profileData,
        )
        if err != nil {
            return err
        }
        
        return nil
    })
}
```

### 3. 操作日志记录

```go
func UpdateUserWithLogging(userID int64, updateData User) error {
    // 记录更新开始
    log.Printf("开始更新用户: ID=%d, 数据=%+v", userID, updateData)
    
    updateBuilder := db.UpdateBuilder[User]{}
    success, err := updateBuilder.UpdateByID(userID, updateData)
    
    if err != nil {
        // 记录错误
        log.Printf("更新用户失败: ID=%d, 错误=%v", userID, err)
        return err
    }
    
    if !success {
        // 记录未找到
        log.Printf("用户不存在: ID=%d", userID)
        return errors.New("用户不存在")
    }
    
    // 记录成功
    log.Printf("用户更新成功: ID=%d", userID)
    return nil
}
```

## 注意事项

1. **时间戳字段** - `UpdatedAt` 字段会自动更新为当前时间
2. **零值处理** - Go 的零值（如 0、""、false）不会被更新，除非使用指针或 `gorm.Expr`
3. **事务支持** - 在事务中更新时需要传递 `TX` 参数
4. **返回值** - `Update` 方法返回布尔值表示是否有记录被更新
5. **乐观锁** - 对于并发更新场景，建议使用版本号实现乐观锁
6. **数据验证** - 更新前进行必要的数据验证和业务逻辑检查
7. **错误处理** - 正确处理各种更新错误，提供用户友好的错误信息