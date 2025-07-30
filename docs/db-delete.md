# 删除操作 (db-delete.md)

## 概述

`DeleteBuilder` 提供了类型安全的数据删除功能，支持根据条件删除和根据主键删除两种方式。支持软删除和硬删除两种模式。

## DeleteBuilder 结构

```go
type DeleteBuilder[T IModel] struct {
    TX *gorm.DB // 可选的事务对象
}
```

## 基本删除方法

### 1. Delete - 根据条件删除

```go
func (db DeleteBuilder[T]) Delete(query Query) (bool, error)
```

**基本用法：**
```go
deleteBuilder := db.DeleteBuilder[User]{}

// 删除符合条件的用户
success, err := deleteBuilder.Delete(db.Query{
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{
                {"status", 0},                                    // 状态为禁用
                {"last_login_at", time.Now().AddDate(0, 0, -365), "<"}, // 一年未登录
            },
        },
    },
})

if err != nil {
    log.Printf("删除失败: %v", err)
    return
}

if success {
    fmt.Println("删除成功")
} else {
    fmt.Println("没有记录被删除")
}
```

### 2. DeleteByID - 根据主键删除

```go
func (db DeleteBuilder[T]) DeleteByID(id interface{}) (bool, error)
```

**基本用法：**
```go
// 根据 ID 删除用户
success, err := deleteBuilder.DeleteByID(123)

if err != nil {
    log.Printf("删除失败: %v", err)
    return
}

if success {
    fmt.Println("用户删除成功")
} else {
    fmt.Println("用户不存在或删除失败")
}
```

## 不同主键类型的删除

### 1. 自增主键删除

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name   string `json:"name"`
    Email  string `json:"email"`
    Status int    `json:"status"`
}

// 使用整型 ID 删除
success, err := deleteBuilder.DeleteByID(123)
```

### 2. UUID 主键删除

```go
type Session struct {
    db.Uuid
    db.Timestamp
    UserID int64  `json:"user_id"`
    Token  string `json:"token"`
    Status int    `json:"status"`
}

// 使用 UUID 删除
sessionBuilder := db.DeleteBuilder[Session]{}
success, err := sessionBuilder.DeleteByID("550e8400-e29b-41d4-a716-446655440000")
```

## 软删除 vs 硬删除

### 1. 软删除（推荐）

使用 `gorm.DeletedAt` 字段实现软删除：

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name      string         `json:"name"`
    Email     string         `json:"email"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 软删除字段
}

// 软删除用户（推荐方式）
success, err := deleteBuilder.DeleteByID(123)
// 实际上是将 deleted_at 字段设置为当前时间
// 查询时会自动过滤掉已软删除的记录
```

**软删除的优点：**
- 数据可恢复
- 保持数据完整性
- 便于审计和追踪
- 避免外键约束问题

### 2. 硬删除

对于不包含 `DeletedAt` 字段的模型，执行硬删除：

```go
type TempData struct {
    db.AutoIncrement
    db.Timestamp
    Data string `json:"data"`
    // 没有 DeletedAt 字段，执行硬删除
}

// 硬删除临时数据
tempBuilder := db.DeleteBuilder[TempData]{}
success, err := tempBuilder.DeleteByID(456)
// 直接从数据库中删除记录
```

### 3. 强制硬删除

即使模型有软删除字段，也可以强制执行硬删除：

```go
// 使用 GORM 原生方法强制硬删除
err := db.DB.Unscoped().Delete(&User{}, 123).Error
```

## 批量删除操作

### 1. 条件批量删除

```go
// 删除所有非活跃用户
success, err := deleteBuilder.Delete(db.Query{
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{
                {"status", 0},                                    // 状态为非活跃
                {"last_login_at", time.Now().AddDate(0, -6, 0), "<"}, // 6个月未登录
            },
        },
    },
})

fmt.Printf("删除了 %d 个非活跃用户\n", success)
```

### 2. 按 ID 列表批量删除

```go
// 批量删除指定 ID 的用户
userIDs := []int64{123, 456, 789}
success, err := deleteBuilder.Delete(db.Query{
    Search: []db.ConditionGroup{
        {
            Conditions: [][]interface{}{
                {"id", userIDs, "in"},
            },
        },
    },
})

fmt.Printf("删除了 %d 个用户\n", success)
```

## 事务中的删除

在事务中使用删除构建器：

```go
err := db.DB.Transaction(func(tx *gorm.DB) error {
    deleteBuilder := db.DeleteBuilder[User]{TX: tx}
    
    // 删除用户
    success, err := deleteBuilder.DeleteByID(123)
    if err != nil {
        return err
    }
    if !success {
        return errors.New("用户不存在")
    }
    
    // 删除用户相关数据
    profileBuilder := db.DeleteBuilder[Profile]{TX: tx}
    _, err = profileBuilder.Delete(db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"user_id", 123},
                },
            },
        },
    })
    if err != nil {
        return err
    }
    
    return nil
})

if err != nil {
    log.Printf("事务删除失败: %v", err)
}
```

## 删除前的安全检查

### 1. 检查记录是否存在

```go
func DeleteUserSafely(userID int64) error {
    // 检查用户是否存在
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
    
    // 执行删除
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.DeleteByID(userID)
    if err != nil {
        return fmt.Errorf("删除用户失败: %w", err)
    }
    if !success {
        return errors.New("删除失败")
    }
    
    return nil
}
```

### 2. 检查关联数据

```go
func DeleteUserWithDependencyCheck(userID int64) error {
    // 检查用户是否有未完成的订单
    orderBuilder := db.QueryBuilder[Order]{
        Query: db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                        {"status", []int{1, 2, 3}, "in"}, // 未完成状态
                    },
                },
            },
        },
    }
    hasOrders, err := orderBuilder.Exists()
    if err != nil {
        return fmt.Errorf("检查用户订单失败: %w", err)
    }
    if hasOrders {
        return errors.New("用户有未完成的订单，无法删除")
    }
    
    // 检查用户是否有未处理的投诉
    complaintBuilder := db.QueryBuilder[Complaint]{
        Query: db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                        {"status", 0}, // 未处理状态
                    },
                },
            },
        },
    }
    hasComplaints, err := complaintBuilder.Exists()
    if err != nil {
        return fmt.Errorf("检查用户投诉失败: %w", err)
    }
    if hasComplaints {
        return errors.New("用户有未处理的投诉，无法删除")
    }
    
    // 执行删除
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.DeleteByID(userID)
    if err != nil {
        return fmt.Errorf("删除用户失败: %w", err)
    }
    if !success {
        return errors.New("用户不存在")
    }
    
    return nil
}
```

## GORM 钩子函数

可以在模型中定义钩子函数，在删除前后执行自定义逻辑：

### 1. BeforeDelete 钩子

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name      string         `json:"name"`
    Email     string         `json:"email"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// 删除前钩子：备份重要数据
func (u *User) BeforeDelete(tx *gorm.DB) error {
    // 备份用户数据到归档表
    archive := UserArchive{
        OriginalID: u.ID,
        Name:       u.Name,
        Email:      u.Email,
        DeletedAt:  time.Now(),
    }
    
    if err := tx.Create(&archive).Error; err != nil {
        return fmt.Errorf("备份用户数据失败: %w", err)
    }
    
    return nil
}
```

### 2. AfterDelete 钩子

```go
// 删除后钩子：清理相关数据
func (u *User) AfterDelete(tx *gorm.DB) error {
    // 异步清理用户相关文件
    go cleanupUserFiles(u.ID)
    
    // 记录删除日志
    log.Printf("用户已删除: ID=%d, Name=%s, Time=%s", u.ID, u.Name, time.Now().Format("2006-01-02 15:04:05"))
    
    return nil
}

func cleanupUserFiles(userID int64) {
    // 清理用户上传的文件
    log.Printf("清理用户 %d 的文件", userID)
}
```

## 软删除数据恢复

### 1. 恢复单条记录

```go
func RestoreUser(userID int64) error {
    // 使用 GORM 原生方法恢复软删除的记录
    result := db.DB.Unscoped().Model(&User{}).Where("id = ?", userID).Update("deleted_at", nil)
    if result.Error != nil {
        return fmt.Errorf("恢复用户失败: %w", result.Error)
    }
    if result.RowsAffected == 0 {
        return errors.New("用户不存在或未被删除")
    }
    
    log.Printf("用户 %d 已恢复", userID)
    return nil
}
```

### 2. 批量恢复记录

```go
func RestoreUsersByEmail(emailPattern string) error {
    // 批量恢复匹配邮箱模式的用户
    result := db.DB.Unscoped().Model(&User{}).
        Where("email LIKE ? AND deleted_at IS NOT NULL", emailPattern).
        Update("deleted_at", nil)
    
    if result.Error != nil {
        return fmt.Errorf("批量恢复用户失败: %w", result.Error)
    }
    
    log.Printf("恢复了 %d 个用户", result.RowsAffected)
    return nil
}
```

## 查询已删除的记录

```go
func GetDeletedUsers() ([]User, error) {
    var users []User
    
    // 查询已软删除的用户
    err := db.DB.Unscoped().Where("deleted_at IS NOT NULL").Find(&users).Error
    if err != nil {
        return nil, fmt.Errorf("查询已删除用户失败: %w", err)
    }
    
    return users, nil
}

func GetDeletedUserByID(userID int64) (*User, error) {
    var user User
    
    // 查询指定 ID 的已删除用户
    err := db.DB.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", userID).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.New("用户不存在或未被删除")
        }
        return nil, fmt.Errorf("查询已删除用户失败: %w", err)
    }
    
    return &user, nil
}
```

## 错误处理

### 1. 常见错误处理

```go
success, err := deleteBuilder.DeleteByID(123)

if err != nil {
    // 检查是否是外键约束错误
    if strings.Contains(err.Error(), "foreign key constraint") {
        fmt.Println("无法删除，存在关联数据")
        return
    }
    
    // 其他错误
    log.Printf("删除失败: %v", err)
    return
}

if !success {
    fmt.Println("记录不存在")
}
```

### 2. 自定义错误处理

```go
func DeleteUserSafely(userID int64) error {
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.DeleteByID(userID)
    
    if err != nil {
        // 记录错误日志
        log.Printf("删除用户失败: ID=%d, 错误=%v", userID, err)
        
        // 返回用户友好的错误信息
        if strings.Contains(err.Error(), "foreign key constraint") {
            return errors.New("无法删除用户，存在关联数据")
        }
        
        return errors.New("删除失败，请稍后重试")
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
// 删除用户的 HTTP 接口
func DeleteUserHandler(c *gin.Context) {
    // 获取用户 ID
    userIDStr := c.Param("id")
    userID, err := strconv.ParseInt(userIDStr, 10, 64)
    if err != nil {
        c.JSON(400, gin.H{
            "error": "Invalid user ID",
        })
        return
    }
    
    // 执行删除
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.DeleteByID(userID)
    if err != nil {
        c.JSON(500, gin.H{
            "error":   "Failed to delete user",
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
    
    c.JSON(200, gin.H{
        "message": "User deleted successfully",
    })
}

// 批量删除用户的 HTTP 接口
func BatchDeleteUsersHandler(c *gin.Context) {
    var request struct {
        UserIDs []int64 `json:"user_ids" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{
            "error":   "Invalid JSON data",
            "details": err.Error(),
        })
        return
    }
    
    // 批量删除
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.Delete(db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"id", request.UserIDs, "in"},
                },
            },
        },
    })
    
    if err != nil {
        c.JSON(500, gin.H{
            "error":   "Failed to delete users",
            "details": err.Error(),
        })
        return
    }
    
    c.JSON(200, gin.H{
        "message": fmt.Sprintf("Successfully deleted %d users", success),
        "count":   success,
    })
}
```

### 2. 请求示例

**删除单个用户：**
```http
DELETE /api/users/123
```

**成功响应：**
```json
{
  "message": "User deleted successfully"
}
```

**批量删除用户：**
```json
POST /api/users/batch-delete
Content-Type: application/json

{
  "user_ids": [123, 456, 789]
}
```

**批量删除响应：**
```json
{
  "message": "Successfully deleted 3 users",
  "count": 3
}
```

## 定时清理任务

### 1. 清理过期数据

```go
func CleanupExpiredSessions() error {
    sessionBuilder := db.DeleteBuilder[Session]{}
    
    // 删除过期的会话
    success, err := sessionBuilder.Delete(db.Query{
        Search: []db.ConditionGroup{
            {
                Conditions: [][]interface{}{
                    {"expires_at", time.Now(), "<"}, // 已过期
                },
            },
        },
    })
    
    if err != nil {
        return fmt.Errorf("清理过期会话失败: %w", err)
    }
    
    log.Printf("清理了 %d 个过期会话", success)
    return nil
}
```

### 2. 清理软删除数据

```go
func CleanupSoftDeletedUsers() error {
    // 硬删除30天前软删除的用户
    thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
    
    result := db.DB.Unscoped().
        Where("deleted_at IS NOT NULL AND deleted_at < ?", thirtyDaysAgo).
        Delete(&User{})
    
    if result.Error != nil {
        return fmt.Errorf("清理软删除用户失败: %w", result.Error)
    }
    
    log.Printf("清理了 %d 个软删除用户", result.RowsAffected)
    return nil
}
```

## 最佳实践

### 1. 安全删除检查

```go
func SafeDeleteUser(userID int64) error {
    // 1. 检查用户是否存在
    queryBuilder := db.QueryBuilder[User]{
        Query: db.Query{
            Filter: []string{"id", "name", "email"},
        },
    }
    var user User
    err := queryBuilder.Find(userID, &user)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return errors.New("用户不存在")
        }
        return fmt.Errorf("查询用户失败: %w", err)
    }
    
    // 2. 检查是否为管理员用户
    if user.Role == "admin" {
        return errors.New("无法删除管理员用户")
    }
    
    // 3. 检查关联数据
    orderBuilder := db.QueryBuilder[Order]{
        Query: db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                        {"status", []int{1, 2, 3}, "in"}, // 活跃订单
                    },
                },
            },
        },
    }
    hasActiveOrders, err := orderBuilder.Exists()
    if err != nil {
        return fmt.Errorf("检查用户订单失败: %w", err)
    }
    if hasActiveOrders {
        return errors.New("用户有活跃订单，无法删除")
    }
    
    // 4. 执行删除
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.DeleteByID(userID)
    if err != nil {
        return fmt.Errorf("删除用户失败: %w", err)
    }
    if !success {
        return errors.New("删除失败")
    }
    
    return nil
}
```

### 2. 事务删除

```go
func DeleteUserAndRelatedData(userID int64) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 删除用户资料
        profileBuilder := db.DeleteBuilder[Profile]{TX: tx}
        _, err := profileBuilder.Delete(db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                    },
                },
            },
        })
        if err != nil {
            return fmt.Errorf("删除用户资料失败: %w", err)
        }
        
        // 删除用户设置
        settingsBuilder := db.DeleteBuilder[UserSettings]{TX: tx}
        _, err = settingsBuilder.Delete(db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                    },
                },
            },
        })
        if err != nil {
            return fmt.Errorf("删除用户设置失败: %w", err)
        }
        
        // 删除用户
        userBuilder := db.DeleteBuilder[User]{TX: tx}
        success, err := userBuilder.DeleteByID(userID)
        if err != nil {
            return fmt.Errorf("删除用户失败: %w", err)
        }
        if !success {
            return errors.New("用户不存在")
        }
        
        return nil
    })
}
```

### 3. 操作日志记录

```go
func DeleteUserWithLogging(userID int64, operatorID int64) error {
    // 记录删除开始
    log.Printf("开始删除用户: ID=%d, 操作者=%d", userID, operatorID)
    
    deleteBuilder := db.DeleteBuilder[User]{}
    success, err := deleteBuilder.DeleteByID(userID)
    
    if err != nil {
        // 记录错误
        log.Printf("删除用户失败: ID=%d, 操作者=%d, 错误=%v", userID, operatorID, err)
        return err
    }
    
    if !success {
        // 记录未找到
        log.Printf("用户不存在: ID=%d, 操作者=%d", userID, operatorID)
        return errors.New("用户不存在")
    }
    
    // 记录成功
    log.Printf("用户删除成功: ID=%d, 操作者=%d", userID, operatorID)
    
    // 记录操作日志到数据库
    auditBuilder := db.CreateBuilder[AuditLog]{}
    _, err = auditBuilder.Create(AuditLog{
        Action:     "delete_user",
        TargetID:   userID,
        OperatorID: operatorID,
        Details:    fmt.Sprintf("删除用户 ID: %d", userID),
    })
    if err != nil {
        log.Printf("记录审计日志失败: %v", err)
    }
    
    return nil
}
```

## 注意事项

1. **软删除优先** - 对于重要数据建议使用软删除而非硬删除
2. **关联检查** - 删除前检查是否存在关联数据，避免数据不一致
3. **事务使用** - 删除相关联的多个表时使用事务确保数据一致性
4. **权限检查** - 删除前检查操作权限，防止误删重要数据
5. **备份策略** - 重要数据删除前进行备份
6. **操作日志** - 记录删除操作的详细日志，便于审计和恢复
7. **定时清理** - 定期清理过期的软删除数据，避免数据库膨胀