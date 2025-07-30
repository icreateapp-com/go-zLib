# 事务处理 (db-transaction.md)

## 概述

数据库模块提供了完整的事务支持，确保数据操作的原子性、一致性、隔离性和持久性（ACID）。支持手动事务管理和自动事务回滚。

## 事务基础

### 1. 事务结构

```go
type db struct {
    *gorm.DB
}

// Transaction 方法用于执行事务
func (d *db) Transaction(fn func(tx *gorm.DB) error) error
```

### 2. 基本事务用法

```go
// 基本事务示例
err := db.DB.Transaction(func(tx *gorm.DB) error {
    // 在事务中执行操作
    user := User{Name: "张三", Email: "zhangsan@example.com"}
    if err := tx.Create(&user).Error; err != nil {
        return err // 自动回滚
    }
    
    profile := Profile{UserID: user.ID, Avatar: "avatar.jpg"}
    if err := tx.Create(&profile).Error; err != nil {
        return err // 自动回滚
    }
    
    return nil // 提交事务
})

if err != nil {
    log.Printf("事务执行失败: %v", err)
}
```

## 在构建器中使用事务

### 1. 创建操作事务

```go
func CreateUserWithProfile(userData User, profileData Profile) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 创建用户
        createBuilder := db.CreateBuilder[User]{TX: tx}
        user, err := createBuilder.Create(userData)
        if err != nil {
            return fmt.Errorf("创建用户失败: %w", err)
        }
        
        // 创建用户资料
        profileData.UserID = user.ID
        profileBuilder := db.CreateBuilder[Profile]{TX: tx}
        _, err = profileBuilder.Create(profileData)
        if err != nil {
            return fmt.Errorf("创建用户资料失败: %w", err)
        }
        
        return nil
    })
}

// 使用示例
user := User{
    Name:  "李四",
    Email: "lisi@example.com",
    Age:   25,
}

profile := Profile{
    Avatar:   "avatar.jpg",
    Bio:      "这是个人简介",
    Location: "北京",
}

err := CreateUserWithProfile(user, profile)
if err != nil {
    log.Printf("创建用户和资料失败: %v", err)
}
```

### 2. 更新操作事务

```go
func UpdateUserAndProfile(userID int64, userData User, profileData Profile) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 更新用户信息
        updateBuilder := db.UpdateBuilder[User]{TX: tx}
        success, err := updateBuilder.UpdateByID(userID, userData)
        if err != nil {
            return fmt.Errorf("更新用户失败: %w", err)
        }
        if !success {
            return errors.New("用户不存在")
        }
        
        // 更新用户资料
        profileBuilder := db.UpdateBuilder[Profile]{TX: tx}
        _, err = profileBuilder.Update(db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                    },
                },
            },
        }, profileData)
        if err != nil {
            return fmt.Errorf("更新用户资料失败: %w", err)
        }
        
        return nil
    })
}
```

### 3. 删除操作事务

```go
func DeleteUserAndRelatedData(userID int64) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 删除用户订单
        orderBuilder := db.DeleteBuilder[Order]{TX: tx}
        _, err := orderBuilder.Delete(db.Query{
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"user_id", userID},
                        {"status", []int{0, 4}, "in"}, // 只删除已取消或已完成的订单
                    },
                },
            },
        })
        if err != nil {
            return fmt.Errorf("删除用户订单失败: %w", err)
        }
        
        // 删除用户资料
        profileBuilder := db.DeleteBuilder[Profile]{TX: tx}
        _, err = profileBuilder.Delete(db.Query{
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

### 4. 查询操作事务

```go
func GetUserWithRelatedData(userID int64) (*UserWithDetails, error) {
    var result *UserWithDetails
    
    err := db.DB.Transaction(func(tx *gorm.DB) error {
        // 查询用户基本信息
        queryBuilder := db.QueryBuilder[User]{TX: tx}
        var user User
        err := queryBuilder.Find(userID, &user)
        if err != nil {
            return fmt.Errorf("查询用户失败: %w", err)
        }
        
        // 查询用户资料
        profileBuilder := db.QueryBuilder[Profile]{
            TX: tx,
            Query: db.Query{
                Search: []db.ConditionGroup{
                    {
                        Conditions: [][]interface{}{
                            {"user_id", userID},
                        },
                    },
                },
            },
        }
        var profile Profile
        err = profileBuilder.First(&profile)
        if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
            return fmt.Errorf("查询用户资料失败: %w", err)
        }
        
        // 查询用户订单
        orderBuilder := db.QueryBuilder[Order]{
            TX: tx,
            Query: db.Query{
                Search: []db.ConditionGroup{
                    {
                        Conditions: [][]interface{}{
                            {"user_id", userID},
                        },
                    },
                },
                OrderBy: []string{"created_at desc"},
                Limit:   10, // 最近10个订单
            },
        }
        var orders []Order
        err = orderBuilder.Get(&orders)
        if err != nil {
            return fmt.Errorf("查询用户订单失败: %w", err)
        }
        
        // 组装结果
        result = &UserWithDetails{
            User:    user,
            Profile: &profile,
            Orders:  orders,
        }
        
        return nil
    })
    
    return result, err
}

type UserWithDetails struct {
    User    User     `json:"user"`
    Profile *Profile `json:"profile"`
    Orders  []Order  `json:"orders"`
}
```

## 复杂事务场景

### 1. 订单处理事务

```go
func ProcessOrder(orderID int64, paymentData Payment) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 1. 查询订单信息
        orderBuilder := db.QueryBuilder[Order]{TX: tx}
        var order Order
        err := orderBuilder.Find(orderID, &order)
        if err != nil {
            return fmt.Errorf("查询订单失败: %w", err)
        }
        
        if order.Status != 1 { // 1: 待支付
            return errors.New("订单状态不正确")
        }
        
        // 2. 检查库存
        productBuilder := db.QueryBuilder[Product]{TX: tx}
        var product Product
        err = productBuilder.Find(order.ProductID, &product)
        if err != nil {
            return fmt.Errorf("查询商品失败: %w", err)
        }
        
        if product.Stock < order.Quantity {
            return errors.New("库存不足")
        }
        
        // 3. 扣减库存
        productUpdateBuilder := db.UpdateBuilder[Product]{TX: tx}
        _, err = productUpdateBuilder.UpdateByID(order.ProductID, Product{
            Stock: product.Stock - order.Quantity,
        })
        if err != nil {
            return fmt.Errorf("更新库存失败: %w", err)
        }
        
        // 4. 创建支付记录
        paymentData.OrderID = orderID
        paymentData.Amount = order.Amount
        paymentBuilder := db.CreateBuilder[Payment]{TX: tx}
        payment, err := paymentBuilder.Create(paymentData)
        if err != nil {
            return fmt.Errorf("创建支付记录失败: %w", err)
        }
        
        // 5. 更新订单状态
        orderUpdateBuilder := db.UpdateBuilder[Order]{TX: tx}
        _, err = orderUpdateBuilder.UpdateByID(orderID, Order{
            Status:    2, // 2: 已支付
            PaymentID: &payment.ID,
            PaidAt:    &db.WrapTime{Time: time.Now()},
        })
        if err != nil {
            return fmt.Errorf("更新订单状态失败: %w", err)
        }
        
        // 6. 记录操作日志
        logBuilder := db.CreateBuilder[OrderLog]{TX: tx}
        _, err = logBuilder.Create(OrderLog{
            OrderID: orderID,
            Action:  "payment_completed",
            Details: fmt.Sprintf("支付完成，支付ID: %d", payment.ID),
        })
        if err != nil {
            return fmt.Errorf("记录操作日志失败: %w", err)
        }
        
        return nil
    })
}
```

### 2. 转账事务

```go
func TransferMoney(fromUserID, toUserID int64, amount decimal.Decimal) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 1. 查询转出账户
        fromAccountBuilder := db.QueryBuilder[Account]{
            TX: tx,
            Query: db.Query{
                Search: []db.ConditionGroup{
                    {
                        Conditions: [][]interface{}{
                            {"user_id", fromUserID},
                        },
                    },
                },
            },
        }
        var fromAccount Account
        err := fromAccountBuilder.First(&fromAccount)
        if err != nil {
            return fmt.Errorf("查询转出账户失败: %w", err)
        }
        
        // 2. 检查余额
        if fromAccount.Balance.LessThan(amount) {
            return errors.New("余额不足")
        }
        
        // 3. 查询转入账户
        toAccountBuilder := db.QueryBuilder[Account]{
            TX: tx,
            Query: db.Query{
                Search: []db.ConditionGroup{
                    {
                        Conditions: [][]interface{}{
                            {"user_id", toUserID},
                        },
                    },
                },
            },
        }
        var toAccount Account
        err = toAccountBuilder.First(&toAccount)
        if err != nil {
            return fmt.Errorf("查询转入账户失败: %w", err)
        }
        
        // 4. 更新转出账户余额
        fromUpdateBuilder := db.UpdateBuilder[Account]{TX: tx}
        _, err = fromUpdateBuilder.UpdateByID(fromAccount.ID, Account{
            Balance: fromAccount.Balance.Sub(amount),
        })
        if err != nil {
            return fmt.Errorf("更新转出账户失败: %w", err)
        }
        
        // 5. 更新转入账户余额
        toUpdateBuilder := db.UpdateBuilder[Account]{TX: tx}
        _, err = toUpdateBuilder.UpdateByID(toAccount.ID, Account{
            Balance: toAccount.Balance.Add(amount),
        })
        if err != nil {
            return fmt.Errorf("更新转入账户失败: %w", err)
        }
        
        // 6. 创建转账记录
        transferBuilder := db.CreateBuilder[Transfer]{TX: tx}
        _, err = transferBuilder.Create(Transfer{
            FromUserID: fromUserID,
            ToUserID:   toUserID,
            Amount:     amount,
            Status:     1, // 1: 成功
        })
        if err != nil {
            return fmt.Errorf("创建转账记录失败: %w", err)
        }
        
        return nil
    })
}
```

### 3. 批量操作事务

```go
func BatchCreateUsers(users []User) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        createBuilder := db.CreateBuilder[User]{TX: tx}
        
        for i, user := range users {
            // 验证用户数据
            if user.Email == "" {
                return fmt.Errorf("第 %d 个用户邮箱不能为空", i+1)
            }
            
            // 检查邮箱是否已存在
            queryBuilder := db.QueryBuilder[User]{
                TX: tx,
                Query: db.Query{
                    Search: []db.ConditionGroup{
                        {
                            Conditions: [][]interface{}{
                                {"email", user.Email},
                            },
                        },
                    },
                },
            }
            exists, err := queryBuilder.Exists()
            if err != nil {
                return fmt.Errorf("检查邮箱存在性失败: %w", err)
            }
            if exists {
                return fmt.Errorf("邮箱 %s 已存在", user.Email)
            }
            
            // 创建用户
            _, err = createBuilder.Create(user)
            if err != nil {
                return fmt.Errorf("创建第 %d 个用户失败: %w", i+1, err)
            }
        }
        
        return nil
    })
}
```

## 事务隔离级别

### 1. 设置隔离级别

```go
func TransactionWithIsolationLevel() error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 设置事务隔离级别
        if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL READ COMMITTED").Error; err != nil {
            return fmt.Errorf("设置隔离级别失败: %w", err)
        }
        
        // 执行业务逻辑
        // ...
        
        return nil
    })
}
```

### 2. 处理并发冲突

```go
func UpdateWithOptimisticLock(userID int64, userData User) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 查询当前版本
        queryBuilder := db.QueryBuilder[User]{TX: tx}
        var currentUser User
        err := queryBuilder.Find(userID, &currentUser)
        if err != nil {
            return fmt.Errorf("查询用户失败: %w", err)
        }
        
        // 检查版本号（乐观锁）
        if userData.Version != currentUser.Version {
            return errors.New("数据已被其他用户修改，请刷新后重试")
        }
        
        // 更新数据并递增版本号
        userData.Version = currentUser.Version + 1
        updateBuilder := db.UpdateBuilder[User]{TX: tx}
        success, err := updateBuilder.UpdateByID(userID, userData)
        if err != nil {
            return fmt.Errorf("更新用户失败: %w", err)
        }
        if !success {
            return errors.New("更新失败")
        }
        
        return nil
    })
}
```

## 事务钩子和回调

### 1. 事务前后钩子

```go
func TransactionWithHooks() error {
    // 事务前钩子
    log.Printf("开始执行事务: %s", time.Now().Format("2006-01-02 15:04:05"))
    
    err := db.DB.Transaction(func(tx *gorm.DB) error {
        // 业务逻辑
        createBuilder := db.CreateBuilder[User]{TX: tx}
        user, err := createBuilder.Create(User{
            Name:  "测试用户",
            Email: "test@example.com",
        })
        if err != nil {
            return err
        }
        
        log.Printf("创建用户成功: ID=%d", user.ID)
        return nil
    })
    
    // 事务后钩子
    if err != nil {
        log.Printf("事务执行失败: %v", err)
    } else {
        log.Printf("事务执行成功: %s", time.Now().Format("2006-01-02 15:04:05"))
    }
    
    return err
}
```

### 2. 自定义事务包装器

```go
type TransactionManager struct {
    db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
    return &TransactionManager{db: db}
}

func (tm *TransactionManager) Execute(name string, fn func(tx *gorm.DB) error) error {
    start := time.Now()
    log.Printf("开始执行事务 [%s]", name)
    
    err := tm.db.Transaction(func(tx *gorm.DB) error {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("事务 [%s] 发生 panic: %v", name, r)
            }
        }()
        
        return fn(tx)
    })
    
    duration := time.Since(start)
    if err != nil {
        log.Printf("事务 [%s] 执行失败: %v (耗时: %v)", name, err, duration)
    } else {
        log.Printf("事务 [%s] 执行成功 (耗时: %v)", name, duration)
    }
    
    return err
}

// 使用示例
tm := NewTransactionManager(db.DB)
err := tm.Execute("创建用户和资料", func(tx *gorm.DB) error {
    // 事务逻辑
    return nil
})
```

## 分布式事务

### 1. 两阶段提交模拟

```go
type DistributedTransaction struct {
    operations []func(*gorm.DB) error
    rollbacks  []func(*gorm.DB) error
}

func NewDistributedTransaction() *DistributedTransaction {
    return &DistributedTransaction{
        operations: make([]func(*gorm.DB) error, 0),
        rollbacks:  make([]func(*gorm.DB) error, 0),
    }
}

func (dt *DistributedTransaction) AddOperation(op func(*gorm.DB) error, rollback func(*gorm.DB) error) {
    dt.operations = append(dt.operations, op)
    dt.rollbacks = append(dt.rollbacks, rollback)
}

func (dt *DistributedTransaction) Execute() error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 第一阶段：准备
        for i, op := range dt.operations {
            if err := op(tx); err != nil {
                // 回滚已执行的操作
                for j := i - 1; j >= 0; j-- {
                    if rollbackErr := dt.rollbacks[j](tx); rollbackErr != nil {
                        log.Printf("回滚操作 %d 失败: %v", j, rollbackErr)
                    }
                }
                return fmt.Errorf("操作 %d 失败: %w", i, err)
            }
        }
        
        // 第二阶段：提交
        return nil
    })
}
```

### 2. 补偿事务模式

```go
type CompensationTransaction struct {
    steps []TransactionStep
}

type TransactionStep struct {
    Name        string
    Execute     func(*gorm.DB) error
    Compensate  func(*gorm.DB) error
    executed    bool
}

func (ct *CompensationTransaction) AddStep(name string, execute, compensate func(*gorm.DB) error) {
    ct.steps = append(ct.steps, TransactionStep{
        Name:       name,
        Execute:    execute,
        Compensate: compensate,
    })
}

func (ct *CompensationTransaction) Execute() error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 执行所有步骤
        for i := range ct.steps {
            step := &ct.steps[i]
            log.Printf("执行步骤: %s", step.Name)
            
            if err := step.Execute(tx); err != nil {
                log.Printf("步骤 %s 执行失败: %v", step.Name, err)
                
                // 执行补偿操作
                for j := i - 1; j >= 0; j-- {
                    if ct.steps[j].executed {
                        log.Printf("执行补偿操作: %s", ct.steps[j].Name)
                        if compErr := ct.steps[j].Compensate(tx); compErr != nil {
                            log.Printf("补偿操作 %s 失败: %v", ct.steps[j].Name, compErr)
                        }
                    }
                }
                
                return fmt.Errorf("事务执行失败: %w", err)
            }
            
            step.executed = true
        }
        
        return nil
    })
}
```

## 事务性能优化

### 1. 减少事务持有时间

```go
// 不推荐：事务时间过长
func BadTransactionExample() error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 长时间的外部 API 调用
        response, err := callExternalAPI() // 可能耗时很长
        if err != nil {
            return err
        }
        
        // 数据库操作
        createBuilder := db.CreateBuilder[User]{TX: tx}
        _, err = createBuilder.Create(User{Name: response.Name})
        return err
    })
}

// 推荐：先完成外部调用，再执行事务
func GoodTransactionExample() error {
    // 先完成外部 API 调用
    response, err := callExternalAPI()
    if err != nil {
        return err
    }
    
    // 快速执行数据库事务
    return db.DB.Transaction(func(tx *gorm.DB) error {
        createBuilder := db.CreateBuilder[User]{TX: tx}
        _, err := createBuilder.Create(User{Name: response.Name})
        return err
    })
}
```

### 2. 批量操作优化

```go
func BatchOperationOptimized(users []User) error {
    // 分批处理，避免单个事务过大
    batchSize := 100
    
    for i := 0; i < len(users); i += batchSize {
        end := i + batchSize
        if end > len(users) {
            end = len(users)
        }
        
        batch := users[i:end]
        err := db.DB.Transaction(func(tx *gorm.DB) error {
            createBuilder := db.CreateBuilder[User]{TX: tx}
            
            for _, user := range batch {
                _, err := createBuilder.Create(user)
                if err != nil {
                    return err
                }
            }
            
            return nil
        })
        
        if err != nil {
            return fmt.Errorf("批次 %d-%d 处理失败: %w", i, end-1, err)
        }
    }
    
    return nil
}
```

## 错误处理和重试

### 1. 事务重试机制

```go
func TransactionWithRetry(maxRetries int, fn func(*gorm.DB) error) error {
    var lastErr error
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := db.DB.Transaction(fn)
        if err == nil {
            return nil // 成功
        }
        
        lastErr = err
        
        // 检查是否是可重试的错误
        if !isRetryableError(err) {
            return err // 不可重试的错误，直接返回
        }
        
        if attempt < maxRetries {
            // 指数退避
            backoff := time.Duration(1<<uint(attempt)) * time.Second
            log.Printf("事务执行失败，%v 后重试 (第 %d 次): %v", backoff, attempt+1, err)
            time.Sleep(backoff)
        }
    }
    
    return fmt.Errorf("事务重试 %d 次后仍然失败: %w", maxRetries, lastErr)
}

func isRetryableError(err error) bool {
    errStr := err.Error()
    // 检查是否是可重试的错误类型
    retryableErrors := []string{
        "deadlock",
        "lock wait timeout",
        "connection reset",
        "connection refused",
    }
    
    for _, retryableErr := range retryableErrors {
        if strings.Contains(strings.ToLower(errStr), retryableErr) {
            return true
        }
    }
    
    return false
}

// 使用示例
err := TransactionWithRetry(3, func(tx *gorm.DB) error {
    // 事务逻辑
    createBuilder := db.CreateBuilder[User]{TX: tx}
    _, err := createBuilder.Create(User{Name: "测试用户"})
    return err
})
```

### 2. 事务超时处理

```go
func TransactionWithTimeout(timeout time.Duration, fn func(*gorm.DB) error) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    done := make(chan error, 1)
    
    go func() {
        done <- db.DB.Transaction(fn)
    }()
    
    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        return fmt.Errorf("事务超时: %w", ctx.Err())
    }
}

// 使用示例
err := TransactionWithTimeout(30*time.Second, func(tx *gorm.DB) error {
    // 事务逻辑
    return nil
})
```

## 最佳实践

### 1. 事务边界设计

```go
// 推荐：事务边界清晰，只包含必要的数据库操作
func CreateOrderTransaction(orderData Order, items []OrderItem) error {
    return db.DB.Transaction(func(tx *gorm.DB) error {
        // 1. 创建订单
        orderBuilder := db.CreateBuilder[Order]{TX: tx}
        order, err := orderBuilder.Create(orderData)
        if err != nil {
            return err
        }
        
        // 2. 创建订单项
        itemBuilder := db.CreateBuilder[OrderItem]{TX: tx}
        for _, item := range items {
            item.OrderID = order.ID
            _, err := itemBuilder.Create(item)
            if err != nil {
                return err
            }
        }
        
        return nil
    })
}
```

### 2. 事务日志记录

```go
func TransactionWithLogging(name string, fn func(*gorm.DB) error) error {
    transactionID := generateTransactionID()
    start := time.Now()
    
    log.Printf("[%s] 事务开始: %s", transactionID, name)
    
    err := db.DB.Transaction(func(tx *gorm.DB) error {
        // 在事务上下文中记录操作
        log.Printf("[%s] 执行事务逻辑", transactionID)
        return fn(tx)
    })
    
    duration := time.Since(start)
    
    if err != nil {
        log.Printf("[%s] 事务失败: %s (耗时: %v, 错误: %v)", transactionID, name, duration, err)
    } else {
        log.Printf("[%s] 事务成功: %s (耗时: %v)", transactionID, name, duration)
    }
    
    return err
}

func generateTransactionID() string {
    return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}
```

### 3. 事务状态监控

```go
type TransactionMetrics struct {
    TotalTransactions    int64
    SuccessTransactions  int64
    FailedTransactions   int64
    AverageExecutionTime time.Duration
    mu                   sync.RWMutex
}

var metrics = &TransactionMetrics{}

func TransactionWithMetrics(name string, fn func(*gorm.DB) error) error {
    start := time.Now()
    
    metrics.mu.Lock()
    metrics.TotalTransactions++
    metrics.mu.Unlock()
    
    err := db.DB.Transaction(fn)
    
    duration := time.Since(start)
    
    metrics.mu.Lock()
    if err != nil {
        metrics.FailedTransactions++
    } else {
        metrics.SuccessTransactions++
    }
    
    // 更新平均执行时间
    totalTime := metrics.AverageExecutionTime * time.Duration(metrics.TotalTransactions-1)
    metrics.AverageExecutionTime = (totalTime + duration) / time.Duration(metrics.TotalTransactions)
    metrics.mu.Unlock()
    
    return err
}

func GetTransactionMetrics() TransactionMetrics {
    metrics.mu.RLock()
    defer metrics.mu.RUnlock()
    return *metrics
}
```

## 注意事项

1. **事务边界** - 保持事务边界尽可能小，只包含必要的数据库操作
2. **死锁避免** - 确保事务中的操作顺序一致，避免死锁
3. **超时设置** - 为长时间运行的事务设置合理的超时时间
4. **错误处理** - 正确处理事务中的错误，确保数据一致性
5. **性能考虑** - 避免在事务中执行耗时的外部调用
6. **隔离级别** - 根据业务需求选择合适的事务隔离级别
7. **重试机制** - 对于可重试的错误实现合理的重试策略
8. **监控日志** - 记录事务执行情况，便于问题排查和性能优化