# 软删除中的唯一字段处理

## 问题描述

当使用软删除时，`deleted_at` 字段会被设置为删除时间，但记录仍然存在于数据库中。这会导致唯一约束的问题：

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Email     string         `gorm:"unique" json:"email"`  // 唯一邮箱
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`       // 软删除字段
}

// 问题：用户 A 被软删除后，无法再创建相同邮箱的新用户
// 因为数据库中仍然存在 email='test@example.com' 的记录
```

## 解决方案

### 1. 复合唯一索引（推荐）

创建包含 `deleted_at` 字段的复合唯一索引：

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    db.SoftDelete
    Email string `json:"email" gorm:"index:idx_email_deleted,unique"`
    Phone string `json:"phone" gorm:"index:idx_phone_deleted,unique"`
}

// 在数据库迁移中创建复合唯一索引
func (User) TableName() string {
    return "users"
}

// 迁移时会自动创建：
// UNIQUE INDEX idx_email_deleted ON users(email, deleted_at)
// UNIQUE INDEX idx_phone_deleted ON users(phone, deleted_at)
```

**原理：**
- 活跃用户：`(email='test@example.com', deleted_at=NULL)` - 唯一
- 已删除用户：`(email='test@example.com', deleted_at='2024-01-01 10:00:00')` - 不冲突
- 新用户：`(email='test@example.com', deleted_at=NULL)` - 可以创建

### 2. 删除时修改唯一字段值

在删除时将唯一字段修改为包含时间戳的值：

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    db.SoftDelete
    Email string `json:"email" gorm:"unique"`
    Phone string `json:"phone" gorm:"unique"`
}

// 自定义删除钩子
func (u *User) BeforeDelete(tx *gorm.DB) error {
    // 在软删除前修改唯一字段
    timestamp := time.Now().Unix()
    u.Email = fmt.Sprintf("%s_deleted_%d", u.Email, timestamp)
    u.Phone = fmt.Sprintf("%s_deleted_%d", u.Phone, timestamp)
    
    // 更新记录
    return tx.Model(u).Select("email", "phone").Updates(u).Error
}
```

### 3. 使用部分索引（MySQL 8.0+）

对于支持部分索引的数据库：

```sql
-- 只对未删除的记录创建唯一索引
CREATE UNIQUE INDEX idx_users_email_active 
ON users(email) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_users_phone_active 
ON users(phone) 
WHERE deleted_at IS NULL;
```

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    db.SoftDelete
    Email string `json:"email"`
    Phone string `json:"phone"`
}

// 在模型中定义部分索引（需要手动创建）
func (User) CreateIndexes(db *gorm.DB) error {
    // 创建部分唯一索引
    return db.Exec(`
        CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active 
        ON users(email) 
        WHERE deleted_at IS NULL
    `).Error
}
```

## 最佳实践

### 1. 推荐的模型定义

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    db.SoftDelete
    
    // 使用复合唯一索引
    Email    string `json:"email" gorm:"index:idx_email_deleted,unique"`
    Username string `json:"username" gorm:"index:idx_username_deleted,unique"`
    Phone    string `json:"phone" gorm:"index:idx_phone_deleted,unique"`
    
    // 其他字段
    Name   string `json:"name"`
    Status int    `json:"status"`
}

func (User) TableName() string {
    return "users"
}
```

### 2. 数据库迁移

```go
func MigrateUser() error {
    // 自动迁移会创建复合唯一索引
    err := db.DB.AutoMigrate(&User{})
    if err != nil {
        return err
    }
    
    // 手动创建额外的索引（如果需要）
    return db.DB.Exec(`
        CREATE INDEX IF NOT EXISTS idx_users_status_active 
        ON users(status) 
        WHERE deleted_at IS NULL
    `).Error
}
```

### 3. 业务逻辑处理

```go
// 创建用户前检查邮箱是否已存在（包括软删除的）
func CreateUserSafely(userData User) (*User, error) {
    // 检查是否存在相同邮箱的用户（包括已删除的）
    var existingUser User
    err := db.DB.Unscoped().Where("email = ?", userData.Email).First(&existingUser).Error
    
    if err == nil {
        if existingUser.DeletedAt.Valid {
            // 存在已删除的用户，可以选择恢复或提示用户
            return nil, errors.New("该邮箱曾经注册过，请联系管理员恢复账户")
        } else {
            // 存在活跃用户
            return nil, errors.New("该邮箱已被注册")
        }
    } else if !errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, err
    }
    
    // 创建新用户
    createBuilder := db.CreateBuilder[User]{}
    user, err := createBuilder.Create(userData)
    return &user, err
}

// 恢复软删除的用户
func RestoreUser(email string) error {
    result := db.DB.Unscoped().Model(&User{}).
        Where("email = ? AND deleted_at IS NOT NULL", email).
        Update("deleted_at", nil)
    
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return errors.New("未找到已删除的用户")
    }
    
    return nil
}
```

### 4. 查询已删除的记录

```go
// 查询所有用户（包括已删除的）
func GetAllUsersIncludeDeleted() ([]User, error) {
    var users []User
    err := db.DB.Unscoped().Find(&users).Error
    return users, err
}

// 只查询已删除的用户
func GetDeletedUsers() ([]User, error) {
    var users []User
    err := db.DB.Unscoped().Where("deleted_at IS NOT NULL").Find(&users).Error
    return users, err
}
```

## 注意事项

1. **复合唯一索引是最推荐的方案**，兼容性好，实现简单
2. **删除时修改字段值**适用于不需要恢复原始数据的场景
3. **部分索引**性能最好，但数据库支持有限
4. **考虑数据恢复需求**，选择合适的方案
5. **定期清理**长期软删除的数据，避免数据库膨胀

## 性能优化

```go
// 为软删除字段创建索引
type User struct {
    db.AutoIncrement
    db.Timestamp
    Email     string         `json:"email" gorm:"index:idx_email_deleted,unique"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index:idx_deleted_at"`  // 为删除时间创建索引
}

// 定期清理软删除数据
func CleanupSoftDeletedUsers(daysOld int) error {
    cutoffDate := time.Now().AddDate(0, 0, -daysOld)
    
    // 硬删除超过指定天数的软删除记录
    result := db.DB.Unscoped().
        Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffDate).
        Delete(&User{})
    
    log.Printf("清理了 %d 条软删除记录", result.RowsAffected)
    return result.Error
}
```