# HTTP 工具模块

go-zLib 的 HTTP 工具模块提供了丰富的 HTTP 请求方法封装，简化外部 API 调用过程。此模块支持各种请求方法、数据格式、流式处理等功能。

## 目录
- [基本 HTTP 请求](#基本-http-请求)
  - [GET 请求](#get-请求)
  - [POST 请求](#post-请求)
  - [PUT 请求](#put-请求)
  - [DELETE 请求](#delete-请求)
- [高级功能](#高级功能)
  - [流式请求](#流式请求)
  - [文件下载](#文件下载)
  - [URL 工具](#url-工具)
  - [网络工具](#网络工具)

## 基本 HTTP 请求

### GET 请求

发送 HTTP GET 请求。

```go
import "github.com/icreateapp-com/go-zLib/z"

// 简单 GET 请求
response, err := z.Get("https://api.example.com/users", nil)
if err != nil {
    // 处理错误
}

// 带请求头的 GET 请求
headers := map[string]string{
    "Authorization": "Bearer token123",
    "Accept": "application/json",
}
response, err := z.Get("https://api.example.com/users", headers)
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 请求 URL |
| headers | map[string]string | 否 | 请求头信息 |

#### 返回值

- ([]byte, error): 响应内容字节数组和可能的错误

### POST 请求

发送 HTTP POST 请求，支持表单数据和 JSON 数据。

#### 表单数据 POST

```go
// 表单数据 POST 请求
data := map[string]interface{}{
    "username": "zhangsan",
    "password": "123456",
}
response, err := z.Post("https://api.example.com/login", data, nil, z.RequestContentTypeForm)
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 请求 URL |
| data | map[string]interface{} | 否 | 表单数据 |
| headers | map[string]string | 否 | 请求头信息 |
| contentType | RequestContentType | 是 | 请求内容类型 |

#### 返回值

- ([]byte, error): 响应内容字节数组和可能的错误

#### JSON 数据 POST

```go
// JSON 数据 POST 请求
data := map[string]interface{}{
    "username": "zhangsan",
    "password": "123456",
    "profile": map[string]interface{}{
        "age": 30,
        "email": "zhangsan@example.com",
    },
}
response, err := z.Post("https://api.example.com/login", data, nil, z.RequestContentTypeJSON)
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 请求 URL |
| data | map[string]interface{} | 否 | JSON 数据 |
| headers | map[string]string | 否 | 请求头信息 |
| contentType | RequestContentType | 是 | 请求内容类型 |

#### 返回值

- ([]byte, error): 响应内容字节数组和可能的错误

### PUT 请求

发送 HTTP PUT 请求。

```go
// PUT 请求
data := map[string]interface{}{
    "username": "lisi",
    "email": "lisi@example.com",
}
response, err := z.Put("https://api.example.com/users/1", data, nil, z.RequestContentTypeJSON)
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 请求 URL |
| data | map[string]interface{} | 否 | 请求数据 |
| headers | map[string]string | 否 | 请求头信息 |
| contentType | RequestContentType | 是 | 请求内容类型 |

#### 返回值

- ([]byte, error): 响应内容字节数组和可能的错误

### DELETE 请求

发送 HTTP DELETE 请求。

```go
// DELETE 请求
response, err := z.Delete("https://api.example.com/users/1", nil)
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 请求 URL |
| headers | map[string]string | 否 | 请求头信息 |

#### 返回值

- ([]byte, error): 响应内容字节数组和可能的错误

## 高级功能

### 流式请求

支持 Server-Sent Events (SSE) 流式响应处理，常用于 ChatGPT 等 API 的流式响应。

```go
// 定义处理流式响应的回调函数
handler := func(response string) error {
    // 处理每一块流式数据
    fmt.Println("收到数据:", response)
    return nil
}

// 发送流式 POST 请求
data := map[string]interface{}{
    "prompt": "你好，请生成一段文字",
    "stream": true,
}
headers := map[string]string{
    "Authorization": "Bearer token123",
}
err := z.PostSSEStream("https://api.openai.com/v1/completions", data, headers, handler)
if err != nil {
    // 处理错误
}
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 请求 URL |
| data | map[string]interface{} | 否 | 请求数据 |
| headers | map[string]string | 否 | 请求头信息 |
| streamHandler | func(response string) error | 是 | 流数据处理回调函数 |

#### 返回值

- error: 可能的错误

### 通用请求方法

`Request` 方法提供了更灵活的 HTTP 请求配置，支持自定义请求方法、参数类型等。

```go
// 发送通用请求
response, err := z.Request(z.RequestOptions{
    URL:     "https://api.example.com/users",
    Method:  "GET",
    Headers: map[string]string{"Authorization": "Bearer token123"},
    ContentType: z.RequestContentTypeJSON,
    Data:    map[string]interface{}{"page": 1, "size": 10},
})
if err != nil {
    // 处理错误
}

// 获取响应状态码
fmt.Println("状态码:", response.StatusCode)
// 获取响应内容
fmt.Println("响应内容:", string(response.Body))
// 获取响应头
fmt.Println("响应头:", response.Headers.Get("Content-Type"))
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| options | RequestOptions | 是 | 请求选项 |

#### 返回值

- ([]byte, error): 响应内容字节数组和可能的错误

### 文件下载

下载文件到本地。

```go
// 下载文件
err := z.Download("https://example.com/file.pdf", "/path/to/save/file.pdf")
if err != nil {
    // 处理错误
}
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 文件 URL |
| filePath | string | 是 | 保存文件的本地路径 |

#### 返回值

- error: 可能的错误

### URL 工具

#### 生成当前服务器的 URL 地址

```go
// 生成 URL
url := z.GetUrl("api/users")
// 返回: http://your-server.com/api/users
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| params | string | 是 | URL 路径 |

#### 返回值

- string: 完整的 URL 地址

#### 检查 URL 是否有效

```go
// 检查 URL 是否有效
valid := z.IsUrl("https://example.com")
// 返回: true
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| toTest | string | 是 | 要检查的 URL |

#### 返回值

- bool: URL 是否有效

#### 向 URL 添加查询参数

```go
// 向 URL 添加查询参数
newUrl, err := z.AppendQueryParamsToURL(
    "https://example.com/search",
    map[string]interface{}{
        "q": "golang",
        "page": 1,
    },
)
// 返回: https://example.com/search?q=golang&page=1
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| originalURL | string | 是 | 原始 URL |
| params | map[string]interface{} | 是 | 要添加的查询参数 |

#### 返回值

- (string, error): 添加查询参数后的 URL 和可能的错误

### 网络工具

#### 获取本地 IP 地址

```go
// 获取本地 IP 地址
ip, err := z.GetLocalIP()
if err != nil {
    // 处理错误
}
fmt.Println("本地 IP:", ip)
```

#### 参数说明

无

#### 返回值

- (string, error): 本地 IP 地址和可能的错误

#### 检查客户端 IP 是否匹配允许的 IP 模式

```go
// 检查客户端 IP 是否匹配允许的 IP 模式
match := z.MatchIP("192.168.1.100", "192.168.1.*")
// 返回: true
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| clientIP | string | 是 | 客户端 IP 地址 |
| allowedIP | string | 是 | 允许的 IP 模式 |

#### 返回值

- bool: IP 是否匹配

#### 检查给定的 IP 是否为本地 IP 地址

```go
// 检查给定的 IP 是否为本地 IP 地址
isLocal := z.IsLocalIP("127.0.0.1")
// 返回: true
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ip | string | 是 | 要检查的 IP 地址 |

#### 返回值

- bool: 是否为本地 IP 地址