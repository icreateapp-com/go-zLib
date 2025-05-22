<h1 align="center">go-zLib</h1>
<p align="center">
  <img alt="Go Version" src="https://img.shields.io/badge/Go-%3E%3D1.18-blue"/>
  <img alt="License" src="https://img.shields.io/badge/License-MIT-green"/>
</p>

<p align="center">一个全面的 Go 微服务开发工具库，简化您的开发流程。</p>

## 📑 概述

go-zLib 是专为 Go 微服务开发设计的实用工具库，整合了微服务开发所需的多种常用功能，包括数据库操作、HTTP 通信、服务发现、中间件、缓存等，旨在降低开发复杂度，提高开发效率。

## 🚀 特性

- **数据库操作**：基于 GORM 的 CRUD 构建器
- **HTTP 工具**：丰富的 HTTP 请求方法封装
- **服务注册与发现**：自动服务注册和健康检查
- **中间件**：认证、健康检查、查询转换
- **缓存**：内存缓存与 Redis 缓存
- **日志系统**：多级别的日志输出
- **配置管理**：灵活的配置加载和访问
- **工具函数**：提供大量实用工具函数

## 📦 安装

```bash
go get github.com/icreateapp-com/go-zLib
```

## ⚙️ 快速开始

### 基本导入

```go
import (
    . "github.com/icreateapp-com/go-zLib/z"
    "github.com/icreateapp-com/go-zLib/z/db"
    "github.com/icreateapp-com/go-zLib/z/grpc_middleware"
    "github.com/icreateapp-com/go-zLib/z/service"
    "github.com/icreateapp-com/go-zLib/z/provider"
)
```

### 配置数据库

```go
// 初始化数据库
db := db.DB.New()

// 定义模型
type User struct {
    db.Model
    Name     string `json:"name"`
    Email    string `json:"email"`
}

// 创建记录
createBuilder := db.CreateBuilder{Model: &User{}}
user := User{Name: "张三", Email: "zhangsan@example.com"}
err := createBuilder.Create(&user)
```

### 使用 HTTP 工具

```go
// 发起 GET 请求
resp, err := Get("https://api.example.com/users", nil)

// 发起 POST JSON 请求
data := map[string]interface{}{
    "name": "张三",
    "email": "zhangsan@example.com",
}
resp, err := PostJson("https://api.example.com/users", data, nil)
```

### 使用中间件

```go
// 在 Gin 框架中使用
r := gin.Default()
r.Use(middleware.AuthMiddleware())
r.GET("/health", middleware.HealthMiddleware())
```

### 服务注册

```go
// 注册服务到服务发现系统
provider.ServiceDiscoverProvider.Register()
```

## 📚 模块清单

go-zLib 包含以下主要模块：

| 模块 | 描述 |
|------|------|
| **z/db** | 数据库操作相关功能，包含 CRUD 构建器和模型定义 |
| **z/middleware** | Web 框架中间件，包括认证、健康检查等 |
| **z/provider** | 服务提供者，包括配置中心和服务发现 |
| **z/service** | 通用服务层，包括 CRUD 服务和性能探针 |
| **z** | 核心功能模块，包含各种工具函数 |

详细的模块文档请查看 [docs](./docs/) 目录。

## 📖 文档

每个模块的详细使用文档位于 [docs](./docs/) 目录下：

- [数据库操作](./docs/db.md)
- [中间件](./docs/middleware.md)
- [服务提供者](./docs/provider.md)
- [服务层](./docs/service.md)
- [HTTP工具](./docs/http.md)
- [缓存](./docs/cache.md)
- [日志](./docs/log.md)
- [配置管理](./docs/config.md)
- [工具函数](./docs/utils.md)

## 🤝 贡献

欢迎贡献代码或提出问题。请先查阅我们的 [贡献指南](./CONTRIBUTING.md)。

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](./LICENSE) 文件。
