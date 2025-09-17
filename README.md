<h1 align="center">go-zLib</h1>
<p align="center">
  <img alt="Go Version" src="https://img.shields.io/badge/Go-%3E%3D1.23-blue"/>
  <img alt="License" src="https://img.shields.io/badge/License-MIT-green"/>
  <img alt="Build Status" src="https://img.shields.io/badge/Build-Passing-brightgreen"/>
</p>

<p align="center">🚀 现代化的 Go 微服务全栈开发框架，让微服务开发变得简单高效</p>

## 📑 概述

go-zLib 是一个功能完整的 Go 微服务开发框架，采用现代化的设计理念和最佳实践。框架整合了微服务开发的全生命周期所需功能，从数据库操作到服务治理，从性能监控到分布式追踪，为开发者提供开箱即用的企业级解决方案。

## ✨ 核心特性

### 🗄️ 数据库层
- **类型安全的 CRUD 操作**：基于 Go 泛型的查询构建器
- **智能查询解析**：支持复杂条件构建和 SQL 注入防护
- **事务管理**：声明式事务装饰器，简化事务操作
- **关联查询**：支持预加载和关联数据查询
- **分页查询**：内置高效分页和统计功能

### 🌐 网络通信
- **HTTP 客户端**：支持多种内容类型和连接池复用
- **gRPC 服务器**：完整的 gRPC 服务支持和中间件
- **WebSocket**：实时双向通信支持
- **SSE 流式数据**：服务端推送事件支持

### 🔧 服务治理
- **服务发现**：支持 Consul、Etcd、Nacos 等注册中心
- **负载均衡**：多种负载均衡策略
- **健康检查**：自动服务健康监测
- **配置中心**：动态配置管理和热更新

### 📊 可观测性
- **分布式追踪**：集成 OpenTelemetry 链路追踪
- **指标监控**：Prometheus 指标收集
- **结构化日志**：多级别日志输出和格式化
- **性能监控**：内置性能探针和分析

### 🛡️ 安全与中间件
- **认证授权**：JWT 和 API Token 多种认证方式
- **请求限流**：防止服务过载
- **参数验证**：自动请求参数校验
- **CORS 支持**：跨域请求处理

### 💾 缓存系统
- **多级缓存**：内存缓存 + Redis 分布式缓存
- **缓存策略**：支持 TTL、LRU 等多种策略
- **缓存穿透防护**：防止缓存击穿和雪崩

## 📦 安装

```bash
go get github.com/icreateapp-com/go-zLib
```

## 🏗️ 架构设计

go-zLib 采用分层架构设计，各模块职责清晰，易于扩展：

```
┌─────────────────────────────────────────────────────────────┐
│                    应用层 (Application)                      │
├─────────────────────────────────────────────────────────────┤
│                   服务层 (Service Layer)                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ CRUD Service│  │Base Service │  │Custom Service│         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│                  控制器层 (Controller Layer)                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │Base Controller│ │CRUD Controller│ │HTTP Server │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│                   中间件层 (Middleware)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │    Auth     │  │   Logging   │  │  Validation │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│                   数据访问层 (Data Access)                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │Query Builder│  │Create Builder│ │Update Builder│         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│                   基础设施层 (Infrastructure)                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Database  │  │    Cache    │  │   Message   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## ⚙️ 快速开始

### 1. 基本项目结构

```
your-project/
├── main.go                 # 应用入口
├── config/
│   └── config.yaml        # 配置文件
├── models/                # 数据模型
├── services/              # 业务逻辑层
├── controllers/           # 控制器层
└── providers/             # 服务提供者
```

### 2. 初始化应用

```go
package main

import (
    . "github.com/icreateapp-com/go-zLib/z"
    "github.com/icreateapp-com/go-zLib/z/db"
    "github.com/icreateapp-com/go-zLib/z/provider/service_discover_provider"
)

func main() {
    // 初始化数据库
    db.DB.Init()
    
    // 注册服务发现
    service_discover_provider.ServiceDiscoverProvider.Register()
    
    // 启动 HTTP 服务器
    MustServeHttp(func(router *gin.Engine) {
        // 注册路由
        router.GET("/health", func(c *gin.Context) {
            Success(c, "服务运行正常")
        })
    })
}
```

### 3. 数据库操作示例

```go
// 定义用户模型
type User struct {
    db.AutoIncrement  // 自增主键
    db.Timestamp      // 时间戳字段
    Name     string   `json:"name" gorm:"size:100;not null"`
    Email    string   `json:"email" gorm:"size:255;uniqueIndex"`
    Age      int      `json:"age"`
    Status   int      `json:"status" gorm:"default:1"`
}

func (User) TableName() string {
    return "users"
}

// CRUD 操作
func userOperations() {
    // 创建用户
    createBuilder := db.CreateBuilder[User]{}
    user := User{Name: "张三", Email: "zhangsan@example.com", Age: 25}
    err := createBuilder.Create(&user)
    
    // 查询用户
    queryBuilder := db.QueryBuilder[User]{
        Query: db.Query{
            Filter: []string{"id", "name", "email"},
            Search: []db.ConditionGroup{
                {
                    Conditions: [][]interface{}{
                        {"status", 1},
                        {"age", 18, ">="},
                    },
                },
            },
            Page: []int{1, 10}, // 第1页，每页10条
        },
    }
    
    var users []User
    pager := &db.Pager{Data: &users}
    err = queryBuilder.Page(pager)
    
    // 更新用户
    updateBuilder := db.UpdateBuilder[User]{}
    err = updateBuilder.UpdateByID(user.ID, map[string]interface{}{
        "name": "李四",
        "age":  26,
    })
    
    // 删除用户
    deleteBuilder := db.DeleteBuilder[User]{}
    err = deleteBuilder.DeleteByID(user.ID)
}
```

### 4. HTTP 服务和中间件

```go
func setupHttpServer() {
    MustServeHttp(func(router *gin.Engine) {
        // 全局中间件
        router.Use(middleware.AuthMiddleware())
        router.Use(middleware.CorsMiddleware())
        
        // API 路由组
        api := router.Group("/api/v1")
        {
            // 用户相关路由
            users := api.Group("/users")
            {
                users.GET("", getUserList)
                users.POST("", createUser)
                users.GET("/:id", getUserByID)
                users.PUT("/:id", updateUser)
                users.DELETE("/:id", deleteUser)
            }
            
            // 健康检查
            api.GET("/health", func(c *gin.Context) {
                Success(c, map[string]interface{}{
                    "status": "healthy",
                    "timestamp": time.Now(),
                })
            })
        }
    })
}
```

## 📚 模块架构

go-zLib 采用模块化设计，各模块相互独立又紧密协作：

| 模块分类 | 模块名称 | 主要功能 | 文档链接 |
|---------|---------|---------|---------|
| **核心基础** | z | 工具函数、类型转换、字符串处理 | [详细文档](./docs/utils.md) |
| **数据访问** | z/db | 数据库连接、CRUD构建器、事务管理 | [详细文档](./docs/db.md) |
| | | - 查询构建器 | [查询操作](./docs/db-query.md) |
| | | - 创建构建器 | [创建操作](./docs/db-create.md) |
| | | - 更新构建器 | [更新操作](./docs/db-update.md) |
| | | - 删除构建器 | [删除操作](./docs/db-delete.md) |
| | | - 模型定义 | [模型设计](./docs/db-model.md) |
| | | - 事务处理 | [事务管理](./docs/db-transaction.md) |
| | | - 软删除 | [软删除机制](./docs/db-soft-delete-unique.md) |
| **网络通信** | z/server | HTTP/gRPC 服务器 | [HTTP服务](./docs/http.md) |
| | | - gRPC 服务器 | [gRPC服务](./docs/grpc_server.md) |
| **服务治理** | z/provider | 服务提供者集合 | - |
| | | - 服务发现 | [服务发现](./docs/provider/service_discovery_provider.md) |
| | | - 配置中心 | [配置中心](./docs/provider/config_center_provider.md) |
| | | - 认证服务 | [认证提供者](./docs/provider/auth_provider.md) |
| | | - 事件总线 | [事件总线](./docs/provider/event_bus_provider.md) |
| | | - WebSocket | [WebSocket](./docs/provider/websocket_provider.md) |
| | | - gRPC 服务 | [gRPC服务提供者](./docs/provider/grpc_service_provider.md) |
| **业务逻辑** | z/service | 通用服务层 | [详细文档](./docs/service.md) |
| **控制器** | z/controller | 控制器基类和CRUD控制器 | - |
| **中间件** | z/middleware | 认证、日志、验证等中间件 | [详细文档](./docs/middleware.md) |
| **缓存系统** | z/cache | 内存缓存和Redis缓存 | [详细文档](./docs/cache.md) |
| **日志系统** | z/log | 结构化日志和多级别输出 | [详细文档](./docs/log.md) |
| **配置管理** | z/config | 配置文件加载和管理 | [详细文档](./docs/config.md) |

## 📖 文档

### 核心模块文档

#### 数据库模块 (z/db)
- [数据库基础](./docs/db.md) - 数据库连接、配置和基础操作
- [查询构建器](./docs/db-query.md) - 复杂查询、条件构建、分页查询
- [创建构建器](./docs/db-create.md) - 数据创建、批量插入操作
- [更新构建器](./docs/db-update.md) - 数据更新、批量更新操作
- [删除构建器](./docs/db-delete.md) - 数据删除、软删除操作
- [模型定义](./docs/db-model.md) - 模型结构、关联关系定义
- [事务管理](./docs/db-transaction.md) - 事务操作、回滚机制
- [软删除机制](./docs/db-soft-delete-unique.md) - 软删除实现和唯一性处理

#### 网络通信模块
- [HTTP 服务](./docs/http.md) - HTTP 客户端、请求处理
- [gRPC 服务器](./docs/grpc_server.md) - gRPC 服务端实现

#### 服务提供者模块 (z/provider)
- [服务发现](./docs/provider/service_discovery_provider.md) - 服务注册、发现、负载均衡
- [配置中心](./docs/provider/config_center_provider.md) - 配置管理、动态更新
- [认证提供者](./docs/provider/auth_provider.md) - 身份认证、权限管理
- [事件总线](./docs/provider/event_bus_provider.md) - 事件发布、订阅机制
- [WebSocket 提供者](./docs/provider/websocket_provider.md) - WebSocket 连接管理
- [gRPC 服务提供者](./docs/provider/grpc_service_provider.md) - gRPC 服务管理

#### 业务支撑模块
- [服务层](./docs/service.md) - CRUD 服务、业务逻辑封装
- [中间件](./docs/middleware.md) - 认证、日志、CORS 等中间件
- [缓存系统](./docs/cache.md) - 内存缓存、Redis 缓存
- [日志系统](./docs/log.md) - 结构化日志、日志级别管理
- [配置管理](./docs/config.md) - 配置文件加载、环境变量管理

### 快速导航

| 使用场景 | 推荐文档 |
|---------|----------|
| 🚀 **快速上手** | [快速开始](#-快速开始) → [数据库基础](./docs/db.md) |
| 🗄️ **数据库操作** | [查询构建器](./docs/db-query.md) → [模型定义](./docs/db-model.md) |
| 🌐 **Web 开发** | [HTTP 服务](./docs/http.md) → [中间件](./docs/middleware.md) |
| 🔧 **微服务架构** | [服务发现](./docs/provider/service_discovery_provider.md) → [配置中心](./docs/provider/config_center_provider.md) |
| 📊 **性能优化** | [缓存系统](./docs/cache.md) → [日志系统](./docs/log.md) |
| 🔐 **安全认证** | [认证提供者](./docs/provider/auth_provider.md) → [中间件](./docs/middleware.md) |

## 🤝 贡献

欢迎贡献代码或提出问题。请先查阅我们的 [贡献指南](./CONTRIBUTING.md)。

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](./LICENSE) 文件。
