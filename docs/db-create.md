# 创建操作 (db-create.md)

## 概述

`CreateBuilder` 提供了类型安全的数据创建功能，支持单条记录创建和批量创建操作。所有创建操作都会自动处理时间戳字段和主键生成。

## CreateBuilder 结构

```go
type CreateBuilder[T IModel] struct {
    TX *gorm.DB // 可选的事务对象
}
```

## 基本创建方法

### Create - 创建单条记录

```go
func (cb CreateBuilder[T]) Create(data T) (T, error)
```

**功能说明：**
- 创建单条记录到数据库
- 自动处理主键生成（自增ID或UUID）
- 自动填充时间戳字段（CreatedAt、UpdatedAt）
- 返回包含完整信息的记录（包括自动生成的ID）

**基本用法：**
```go
createBuilder := db.CreateBuilder[User]{}

// 创建用户
user, err := createBuilder.Create(User{
    Name:  "张三",
    Email: "zhangsan@example.com",
    Age:   25,
    Status: 1,
})

if err != nil {
    log.Printf("创建用户失败: %v", err)
    return
}

fmt.Printf("创建成功，用户ID: %d\n", user.ID)
fmt.Printf("创建时间: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
```

## 不同主键类型的创建

### 1. 自增主键 (AutoIncrement)

```go
type User struct {
    db.AutoIncrement // ID 字段会自动生成
    db.Timestamp
    Name  string `json:"name"`
    Email string `json:"email"`
}

// 创建时不需要指定 ID
user, err := createBuilder.Create(User{
    Name:  "李四",
    Email: "lisi@example.com",
})

// ID 会自动生成
fmt.Printf("生成的用户ID: %d\n", user.ID)
```

### 2. UUID 主键 (Uuid)

```go
type Session struct {
    db.Uuid // ID 字段会自动生成 UUID
    db.Timestamp
    UserID int64  `json:"user_id"`
    Token  string `json:"token"`
}

// 创建时不需要指定 ID
session, err := sessionBuilder.Create(Session{
    UserID: 123,
    Token:  "abc123token",
})

// UUID 会自动生成
fmt.Printf("生成的会话ID: %s\n", session.ID)
```

## 复杂数据类型创建

### 1. 包含 JSON 字段的创建

```go
type Product struct {
    db.AutoIncrement
    db.Timestamp
    Name     string       `json:"name"`
    Price    float64      `json:"price"`
    Metadata db.JsonField `json:"metadata" gorm:"type:json"`
}

// 创建包含 JSON 数据的产品
product, err := productBuilder.Create(Product{
    Name:  "iPhone 15",
    Price: 5999.00,
    Metadata: db.JsonField{
        Data: map[string]interface{}{
            "color":   "黑色",
            "storage": "128GB",
            "specs": map[string]interface{}{
                "screen": "6.1英寸",
                "chip":   "A17 Pro",
                "camera": "48MP主摄",
            },
            "features": []string{
                "Face ID",
                "无线充电",
                "防水防尘",
            },
        },
    },
})
```

### 2. 包含时间字段的创建

```go
type Article struct {
    db.AutoIncrement
    db.Timestamp
    Title       string     `json:"title"`
    Content     string     `json:"content"`
    PublishedAt *time.Time `json:"published_at"`
    Status      int        `json:"status"`
}

// 创建文章
now := time.Now()
article, err := articleBuilder.Create(Article{
    Title:       "Go语言最佳实践",
    Content:     "这是一篇关于Go语言的文章...",
    PublishedAt: &now,
    Status:      1,
})
```

## 批量创建

虽然 `CreateBuilder` 主要用于单条记录创建，但可以结合 GORM 原生方法进行批量创建：

```go
// 批量创建用户
users := []User{
    {Name: "用户1", Email: "user1@example.com", Age: 25},
    {Name: "用户2", Email: "user2@example.com", Age: 30},
    {Name: "用户3", Email: "user3@example.com", Age: 35},
}

// 使用 GORM 原生方法批量创建
err := db.DB.Create(&users).Error
if err != nil {
    log.Printf("批量创建失败: %v", err)
    return
}

// 批量创建后，每个用户的 ID 都会被填充
for _, user := range users {
    fmt.Printf("用户 %s 的ID: %d\n", user.Name, user.ID)
}
```

## 事务中的创建

在事务中使用创建构建器：

```go
err := db.DB.Transaction(func(tx *gorm.DB) error {
    // 在事务中创建构建器
    createBuilder := db.CreateBuilder[User]{TX: tx}
    
    // 创建用户
    user, err := createBuilder.Create(User{
        Name:  "事务用户",
        Email: "transaction@example.com",
    })
    if err != nil {
        return err // 自动回滚
    }
    
    // 创建用户资料
    profileBuilder := db.CreateBuilder[Profile]{TX: tx}
    _, err = profileBuilder.Create(Profile{
        UserID: user.ID,
        Bio:    "这是用户简介",
    })
    if err != nil {
        return err // 自动回滚
    }
    
    return nil // 提交事务
})

if err != nil {
    log.Printf("事务失败: %v", err)
}
```

## 创建前的数据验证

### 1. 模型级别的验证

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=0,max=150"`
}

// 创建前验证
func CreateUserWithValidation(userData User) (*User, error) {
    // 使用 validator 库进行验证
    validate := validator.New()
    if err := validate.Struct(userData); err != nil {
        return nil, fmt.Errorf("数据验证失败: %w", err)
    }
    
    // 验证通过后创建
    createBuilder := db.CreateBuilder[User]{}
    return createBuilder.Create(userData)
}
```

### 2. 业务逻辑验证

```go
func CreateUser(userData User) (*User, error) {
    // 检查邮箱是否已存在
    queryBuilder := db.QueryBuilder[User]{
        Query: db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"email", userData.Email},
                    },
                },
            },
        },
    }
    exists, err := queryBuilder.Exists()
    if err != nil {
        return nil, fmt.Errorf("检查邮箱失败: %w", err)
    }
    if exists {
        return nil, errors.New("邮箱已存在")
    }
    
    // 创建用户
    createBuilder := db.CreateBuilder[User]{}
    return createBuilder.Create(userData)
}
```

## GORM 钩子函数

可以在模型中定义钩子函数，在创建前后执行自定义逻辑：

### 1. BeforeCreate 钩子

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Salt     string `json:"salt"`
}

// 创建前钩子：加密密码
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.Password != "" {
        // 生成盐值
        u.Salt = generateSalt()
        
        // 加密密码
        hashedPassword, err := hashPassword(u.Password, u.Salt)
        if err != nil {
            return err
        }
        u.Password = hashedPassword
    }
    return nil
}

func generateSalt() string {
    // 生成随机盐值的逻辑
    return "random_salt"
}

func hashPassword(password, salt string) (string, error) {
    // 密码加密逻辑
    return "hashed_password", nil
}
```

### 2. AfterCreate 钩子

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name  string `json:"name"`
    Email string `json:"email"`
}

// 创建后钩子：发送欢迎邮件
func (u *User) AfterCreate(tx *gorm.DB) error {
    // 发送欢迎邮件的逻辑
    go sendWelcomeEmail(u.Email, u.Name)
    return nil
}

func sendWelcomeEmail(email, name string) {
    // 异步发送邮件的逻辑
    fmt.Printf("发送欢迎邮件给 %s (%s)\n", name, email)
}
```

## 错误处理

### 1. 常见错误类型

```go
user, err := createBuilder.Create(User{
    Name:  "张三",
    Email: "zhangsan@example.com",
})

if err != nil {
    // 检查是否是唯一约束错误
    if strings.Contains(err.Error(), "Duplicate entry") {
        fmt.Println("邮箱已存在")
        return
    }
    
    // 检查是否是非空约束错误
    if strings.Contains(err.Error(), "cannot be null") {
        fmt.Println("必填字段不能为空")
        return
    }
    
    // 其他错误
    log.Printf("创建失败: %v", err)
    return
}
```

### 2. 自定义错误处理

```go
func CreateUserSafely(userData User) (*User, error) {
    createBuilder := db.CreateBuilder[User]{}
    user, err := createBuilder.Create(userData)
    
    if err != nil {
        // 记录错误日志
        log.Printf("创建用户失败: %+v, 错误: %v", userData, err)
        
        // 返回用户友好的错误信息
        if strings.Contains(err.Error(), "Duplicate entry") {
            return nil, errors.New("邮箱或用户名已存在")
        }
        
        return nil, errors.New("创建用户失败，请稍后重试")
    }
    
    return user, nil
}
```

## JSON API 示例

### 1. HTTP 接口实现

```go
// 创建用户的 HTTP 接口
func CreateUserHandler(c *gin.Context) {
    var userData User
    
    // 绑定 JSON 数据
    if err := c.ShouldBindJSON(&userData); err != nil {
        c.JSON(400, gin.H{
            "error":   "Invalid JSON data",
            "details": err.Error(),
        })
        return
    }
    
    // 创建用户
    createBuilder := db.CreateBuilder[User]{}
    user, err := createBuilder.Create(userData)
    if err != nil {
        c.JSON(500, gin.H{
            "error":   "Failed to create user",
            "details": err.Error(),
        })
        return
    }
    
    // 返回创建的用户（隐藏敏感信息）
    c.JSON(201, gin.H{
        "message": "User created successfully",
        "data": gin.H{
            "id":         user.ID,
            "name":       user.Name,
            "email":      user.Email,
            "created_at": user.CreatedAt,
        },
    })
}
```

### 2. 请求示例

**创建用户请求：**
```json
POST /api/users
Content-Type: application/json

{
  "name": "张三",
  "email": "zhangsan@example.com",
  "age": 25,
  "status": 1
}
```

**成功响应：**
```json
{
  "message": "User created successfully",
  "data": {
    "id": 123,
    "name": "张三",
    "email": "zhangsan@example.com",
    "created_at": "2023-12-01 10:30:00"
  }
}
```

**错误响应：**
```json
{
  "error": "Failed to create user",
  "details": "Duplicate entry 'zhangsan@example.com' for key 'email'"
}
```

## 最佳实践

### 1. 数据验证

```go
// 在创建前进行完整的数据验证
func CreateUserWithValidation(userData User) (*User, error) {
    // 1. 基本字段验证
    if userData.Name == "" {
        return nil, errors.New("用户名不能为空")
    }
    if len(userData.Name) > 100 {
        return nil, errors.New("用户名长度不能超过100字符")
    }
    
    // 2. 邮箱格式验证
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !emailRegex.MatchString(userData.Email) {
        return nil, errors.New("邮箱格式不正确")
    }
    
    // 3. 业务逻辑验证
    if userData.Age < 0 || userData.Age > 150 {
        return nil, errors.New("年龄必须在0-150之间")
    }
    
    // 4. 创建用户
    createBuilder := db.CreateBuilder[User]{}
    return createBuilder.Create(userData)
}
```

### 2. 事务使用

```go
// 创建用户及其相关数据
func CreateUserWithProfile(userData User, profileData Profile) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 创建用户
        userBuilder := db.CreateBuilder[User]{TX: tx}
        user, err := userBuilder.Create(userData)
        if err != nil {
            return err
        }
        
        // 创建用户资料
        profileData.UserID = user.ID
        profileBuilder := db.CreateBuilder[Profile]{TX: tx}
        _, err = profileBuilder.Create(profileData)
        return err
    })
}
```

### 3. 错误日志记录

```go
func CreateUserWithLogging(userData User) (*User, error) {
    createBuilder := db.CreateBuilder[User]{}
    
    // 记录创建开始
    log.Printf("开始创建用户: %s (%s)", userData.Name, userData.Email)
    
    user, err := createBuilder.Create(userData)
    if err != nil {
        // 记录错误
        log.Printf("创建用户失败: %s (%s), 错误: %v", userData.Name, userData.Email, err)
        return nil, err
    }
    
    // 记录成功
    log.Printf("用户创建成功: ID=%d, Name=%s", user.ID, user.Name)
    return user, nil
}
```

## 注意事项

1. **主键生成** - 自增主键和 UUID 主键会自动生成，不需要手动指定
2. **时间戳字段** - `CreatedAt` 和 `UpdatedAt` 字段会自动设置
3. **事务支持** - 在事务中创建时需要传递 `TX` 参数
4. **数据验证** - 建议在创建前进行数据验证
5. **错误处理** - 正确处理各种创建错误，提供用户友好的错误信息
6. **钩子函数** - 合理使用 GORM 钩子函数处理创建前后的逻辑
7. **批量创建** - 大量数据创建时使用 GORM 原生的批量创建方法