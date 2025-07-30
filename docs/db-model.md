# 模型定义 (db-model.md)

## 概述

数据库模块提供了一套完整的模型基类和接口，用于定义数据库表结构。所有模型都必须实现 `IModel` 接口，并可以组合使用内置的模型组件。

## 基础接口

### IModel 接口

所有模型必须实现 `IModel` 接口：

```go
type IModel interface {
    TableName() string
}
```

**示例：**
```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    Name  string `json:"name" gorm:"size:100;not null"`
    Email string `json:"email" gorm:"size:255;uniqueIndex"`
}

func (User) TableName() string {
    return "users"
}
```

## 内置模型组件

### 1. 自增主键 (AutoIncrement)

提供自增整型主键：

```go
type AutoIncrement struct {
    ID int64 `json:"id" gorm:"unique;primaryKey;autoIncrement"`
}
```

**使用示例：**
```go
type Product struct {
    db.AutoIncrement // 包含 ID 字段
    Name  string `json:"name"`
    Price float64 `json:"price"`
}
```

### 2. UUID 主键 (Uuid)

提供 UUID 字符串主键，自动生成：

```go
type Uuid struct {
    ID string `gorm:"unique;primaryKey" json:"id" form:"id"`
}

// 创建前自动生成 UUID
func (m *Uuid) BeforeCreate(tx *gorm.DB) (err error) {
    m.ID = googleUuid.New().String()
    return
}
```

**使用示例：**
```go
type Session struct {
    db.Uuid // 包含 UUID ID 字段
    UserID int64  `json:"user_id"`
    Token  string `json:"token"`
}
```

### 3. 时间戳 (Timestamp)

提供创建时间和更新时间字段：

```go
type Timestamp struct {
    CreatedAt WrapTime `gorm:"type:datetime;autoCreateTime;->;<-:create" json:"created_at"`
    UpdatedAt WrapTime `gorm:"type:datetime;autoUpdateTime;->;<-:update" json:"updated_at"`
}
```

**字段说明：**
- `CreatedAt`: 创建时间，只在记录创建时自动设置，后续更新不会改变
- `UpdatedAt`: 更新时间，在记录创建和更新时都会自动设置为当前时间
- `->`: 只读字段，防止手动修改
- `<-:create`: CreatedAt 只在创建时写入
- `<-:update`: UpdatedAt 只在更新时写入

**使用示例：**
```go
type Article struct {
    db.AutoIncrement
    db.Timestamp // 包含 CreatedAt 和 UpdatedAt 字段
    Title   string `json:"title"`
    Content string `json:"content"`
}
```

## 特殊字段类型

### 1. WrapTime 时间类型

自定义时间类型，支持特定格式的序列化：

```go
type WrapTime struct {
    time.Time
}

// 实现 json.Marshaler 接口
func (t WrapTime) MarshalJSON() ([]byte, error) {
    formatted := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
    return []byte(formatted), nil
}

// 实现 driver.Valuer 接口
func (t WrapTime) Value() (driver.Value, error) {
    return t.Time, nil
}

// 实现 sql.Scanner 接口
func (t *WrapTime) Scan(value interface{}) error {
    if value == nil {
        *t = WrapTime{Time: time.Time{}}
        return nil
    }
    
    switch v := value.(type) {
    case time.Time:
        *t = WrapTime{Time: v}
        return nil
    case string:
        parsedTime, err := time.Parse("2006-01-02 15:04:05", v)
        if err != nil {
            return err
        }
        *t = WrapTime{Time: parsedTime}
        return nil
    default:
        return fmt.Errorf("cannot scan %T into WrapTime", value)
    }
}
```

**JSON 格式：**
```json
{
  "created_at": "2023-12-01 10:30:00",
  "updated_at": "2023-12-01 10:30:00"
}
```

### 2. JsonField JSON 字段

用于存储 JSON 数据的字段类型：

```go
type JsonField struct {
    Data interface{} `json:"data"`
}

// 实现 driver.Valuer 接口
func (j JsonField) Value() (driver.Value, error) {
    if j.Data == nil {
        return nil, nil
    }
    return json.Marshal(j.Data)
}

// 实现 sql.Scanner 接口
func (j *JsonField) Scan(value interface{}) error {
    if value == nil {
        j.Data = nil
        return nil
    }
    
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("cannot scan %T into JsonField", value)
    }
    
    return json.Unmarshal(bytes, &j.Data)
}
```

**使用示例：**
```go
type Product struct {
    db.AutoIncrement
    db.Timestamp
    Name     string       `json:"name"`
    Metadata db.JsonField `json:"metadata" gorm:"type:json"`
}

// 创建产品
product := Product{
    Name: "iPhone 15",
    Metadata: db.JsonField{
        Data: map[string]interface{}{
            "color":   "黑色",
            "storage": "128GB",
            "specs": map[string]interface{}{
                "screen": "6.1英寸",
                "chip":   "A17 Pro",
            },
        },
    },
}
```

**JSON 格式：**
```json
{
  "id": 1,
  "name": "iPhone 15",
  "metadata": {
    "data": {
      "color": "黑色",
      "storage": "128GB",
      "specs": {
        "screen": "6.1英寸",
        "chip": "A17 Pro"
      }
    }
  },
  "created_at": "2023-12-01 10:30:00",
  "updated_at": "2023-12-01 10:30:00"
}
```

## 完整模型示例

### 用户模型

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    
    // 基本信息
    Name     string `json:"name" gorm:"size:100;not null;comment:用户名"`
    Email    string `json:"email" gorm:"size:255;uniqueIndex;comment:邮箱"`
    Phone    string `json:"phone" gorm:"size:20;index;comment:手机号"`
    Avatar   string `json:"avatar" gorm:"size:500;comment:头像URL"`
    
    // 状态信息
    Status   int    `json:"status" gorm:"default:1;comment:状态:1=正常,0=禁用"`
    VipLevel int    `json:"vip_level" gorm:"default:0;comment:VIP等级"`
    
    // 扩展信息
    Profile  db.JsonField `json:"profile" gorm:"type:json;comment:用户资料"`
    Settings db.JsonField `json:"settings" gorm:"type:json;comment:用户设置"`
    
    // 软删除
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (User) TableName() string {
    return "users"
}
```

### 文章模型

```go
type Article struct {
    db.AutoIncrement
    db.Timestamp
    
    // 基本信息
    Title       string `json:"title" gorm:"size:255;not null;comment:标题"`
    Slug        string `json:"slug" gorm:"size:255;uniqueIndex;comment:URL别名"`
    Summary     string `json:"summary" gorm:"size:500;comment:摘要"`
    Content     string `json:"content" gorm:"type:longtext;comment:内容"`
    CoverImage  string `json:"cover_image" gorm:"size:500;comment:封面图"`
    
    // 关联信息
    AuthorID    int64 `json:"author_id" gorm:"not null;index;comment:作者ID"`
    CategoryID  int64 `json:"category_id" gorm:"index;comment:分类ID"`
    
    // 状态信息
    Status      int   `json:"status" gorm:"default:1;comment:状态:1=发布,0=草稿"`
    ViewCount   int   `json:"view_count" gorm:"default:0;comment:浏览次数"`
    LikeCount   int   `json:"like_count" gorm:"default:0;comment:点赞次数"`
    
    // 扩展信息
    Tags        db.JsonField `json:"tags" gorm:"type:json;comment:标签"`
    Metadata    db.JsonField `json:"metadata" gorm:"type:json;comment:元数据"`
    
    // 时间信息
    PublishedAt *time.Time `json:"published_at" gorm:"comment:发布时间"`
    
    // 软删除
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Article) TableName() string {
    return "articles"
}
```

### 订单模型

```go
type Order struct {
    db.Uuid // 使用 UUID 主键
    db.Timestamp
    
    // 基本信息
    OrderNo     string  `json:"order_no" gorm:"size:32;uniqueIndex;not null;comment:订单号"`
    UserID      int64   `json:"user_id" gorm:"not null;index;comment:用户ID"`
    TotalAmount float64 `json:"total_amount" gorm:"type:decimal(10,2);not null;comment:总金额"`
    
    // 状态信息
    Status      int     `json:"status" gorm:"default:1;comment:状态:1=待付款,2=已付款,3=已发货,4=已完成,5=已取消"`
    PayStatus   int     `json:"pay_status" gorm:"default:0;comment:支付状态:0=未支付,1=已支付"`
    
    // 地址信息
    ShippingAddress db.JsonField `json:"shipping_address" gorm:"type:json;comment:收货地址"`
    
    // 商品信息
    Items       db.JsonField `json:"items" gorm:"type:json;comment:订单商品"`
    
    // 时间信息
    PaidAt      *time.Time `json:"paid_at" gorm:"comment:支付时间"`
    ShippedAt   *time.Time `json:"shipped_at" gorm:"comment:发货时间"`
    CompletedAt *time.Time `json:"completed_at" gorm:"comment:完成时间"`
}

func (Order) TableName() string {
    return "orders"
}
```

## 模型设计最佳实践

### 1. 字段命名规范

```go
// 推荐：使用驼峰命名
type User struct {
    FirstName string `json:"first_name" gorm:"column:first_name"`
    LastName  string `json:"last_name" gorm:"column:last_name"`
}

// 避免：使用下划线命名
type User struct {
    first_name string // 不推荐
    last_name  string // 不推荐
}
```

### 2. 索引设计

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    
    Email    string `gorm:"uniqueIndex"`           // 唯一索引
    Phone    string `gorm:"index"`                 // 普通索引
    Status   int    `gorm:"index:idx_status_type"` // 复合索引
    Type     int    `gorm:"index:idx_status_type"` // 复合索引
    DeletedAt gorm.DeletedAt `gorm:"index"`       // 软删除索引
}
```

### 3. 字段验证

```go
type User struct {
    db.AutoIncrement
    db.Timestamp
    
    Name  string `json:"name" gorm:"size:100;not null" validate:"required,min=2,max=100"`
    Email string `json:"email" gorm:"size:255;uniqueIndex" validate:"required,email"`
    Age   int    `json:"age" gorm:"check:age >= 0 AND age <= 150" validate:"min=0,max=150"`
}
```

### 4. 关联关系

```go
// 一对一关系
type User struct {
    db.AutoIncrement
    Profile Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    db.AutoIncrement
    UserID int64 `gorm:"uniqueIndex"`
    Bio    string
}

// 一对多关系
type User struct {
    db.AutoIncrement
    Orders []Order `gorm:"foreignKey:UserID"`
}

type Order struct {
    db.AutoIncrement
    UserID int64
    Amount float64
}

// 多对多关系
type User struct {
    db.AutoIncrement
    Roles []Role `gorm:"many2many:user_roles;"`
}

type Role struct {
    db.AutoIncrement
    Name string
}
```

## 注意事项

1. **表名规范** - 使用复数形式，如 `users`、`articles`
2. **字段类型** - 根据数据长度选择合适的字段类型和长度
3. **索引优化** - 为经常查询的字段添加索引
4. **软删除** - 对于重要数据使用软删除而非物理删除
5. **JSON 字段** - 适度使用 JSON 字段存储非结构化数据
6. **时间字段** - 统一使用 `WrapTime` 类型确保格式一致
7. **注释说明** - 为字段添加清晰的注释说明