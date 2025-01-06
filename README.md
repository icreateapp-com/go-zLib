<h1 align="center">
  zLib
</h1>
<h3 align="center">
  Tool libraries based on Golang.
</h3>

## 📦 包说明

| 文档                             | 说明         |
|--------------------------------|------------|
| [Cache](docs/cache.md)         | 缓存工具包      |
| [Config](docs/config.md)       | 配置工具包      |
| [Cron](docs/cron.md)           | 定时任务工具包    |
| [Grpc](docs/grpc.md)           | Grpc 工具包   |
| [Log](docs/log.md)             | 日志工具包      |
| [Response](docs/response.md)   | HTTP 响应工具包 |
| [Validator](docs/validator.md) | HTTP 验证工具包 |

## 🚀 函数说明

| 函数名称             | 说明                    | 参数                                                                         | 返回值                                                   |
|------------------|-----------------------|----------------------------------------------------------------------------|-------------------------------------------------------|
| BasePath         | 返回项目目录的绝对路径。          | 无                                                                          | `string`：项目目录的绝对路径。<br>`error`：可能出现的错误。               |
| StorePath        | 返回存储目录的绝对路径。          | `paths ...string`：可变参数，表示存储目录的子路径。                                         | `string`：存储目录的绝对路径。                                   |
| TmpPath          | 返回临时目录的绝对路径。          | `paths ...string`：可变参数，表示临时目录的子路径。                                         | `string`：临时目录的绝对路径。                                   |
| CachePath        | 返回缓存目录的绝对路径。          | `paths ...string`：可变参数，表示缓存目录的子路径。                                         | `string`：缓存目录的绝对路径。                                   |
| LogPath          | 返回日志目录的绝对路径。          | `paths ...string`：可变参数，表示日志目录的子路径。                                         | `string`：日志目录的绝对路径。                                   |
| IsExists         | 判断文件或目录是否存在。          | `path string`：要检查的文件或目录的路径。                                                | `bool`：文件或目录是否存在。<br>`error`：可能出现的错误。                 |
| GetSha1          | 对字符串进行 SHA1 编码。       | `str string`：要编码的字符串。                                                      | `string`：SHA1 编码后的字符串。                                |
| GetMd5           | 对字符串进行 MD5 编码。        | `str string`：要编码的字符串。                                                      | `string`：MD5 编码后的字符串。                                 |
| EncodeJson       | 将对象编码为 JSON 字符串。      | `v any`：要编码的对象。                                                            | `string`：JSON 编码后的字符串。<br>`error`：可能出现的错误。            |
| DecodeJson       | 将 JSON 字符串解码为对象。      | `str string`：要解码的 JSON 字符串。                                                | `map[string]interface{}`：解码后的对象。<br>`error`：可能出现的错误。  |
| DecodeJsonValue  | 将 JSON 字符串解码并返回其中一个值。 | `str string`：要解码的 JSON 字符串。<br>`key string`：要获取的值的键。                       | `string`：解码后的值。<br>`error`：可能出现的错误。                   |
| EncodeBase64     | 对字符串进行 Base64 编码。     | `str string`：要编码的字符串。                                                      | `string`：Base64 编码后的字符串。                              |
| DecodeBase64     | 对字符串进行 Base64 解码。     | `str string`：要解码的 Base64 字符串。                                              | `string`：解码后的字符串。<br>`error`：可能出现的错误。                 |
| IsMd5            | 判断字符串是否是 MD5 格式。      | `s string`：要检查的字符串。                                                        | `bool`：字符串是否是 MD5 格式。                                 |
| IsCron           | 判断字符串是否是 Cron 格式。     | `cronStr string`：要检查的字符串。                                                  | `bool`：字符串是否是 Cron 格式。                                |
| IsBase64Image    | 判断字符串是否是 base64 图片。   | `toTest string`：要检查的字符串。                                                   | `bool`：字符串是否是 base64 图片。                              |
| GetUrl           | 生成当前服务器的 URL 地址。      | `params string`：URL 的参数部分。                                                 | `string`：生成的 URL 地址。                                  |
| Post             | 发起 POST 请求。           | `url string`：请求的 URL 地址。<br>`data map[string]interface{}`：请求的数据，以键值对的形式传递。 | `map[string]interface{}`：请求的响应数据。<br>`error`：可能出现的错误。 |
| Get              | 发起 GET 请求。            | `url string`：请求的 URL 地址。                                                   | `map[string]interface{}`：请求的响应数据。<br>`error`：可能出现的错误。 |
| IsUrl            | 判断是否是有效的 URL。         | `toTest string`：要测试的字符串。                                                   | `bool`：字符串是否是有效的 URL。                                 |
| Download         | 下载文件。                 | `url string`：文件的 URL 地址。<br>`filePath string`：下载文件的保存路径。                   | `error`：可能出现的错误。                                      |
| GetSortedMapKeys | 返回排序后的 MAP 所有键。       | `elements map[string]interface{}`：要获取键的 MAP。                               | `[]string`：排序后的键列表。                                   |
| GetMapKeys       | 返回 MAP 的所有键。          | `elements map[string]interface{}`：要获取键的 MAP。                               | `[]string`：键列表。                                       |
| ToValues         | 转换表单项。                | `data map[string]interface{}`：要转换的表单数据。                                    | `url.Values`：转换后的表单数据。                                |
| StringIsEmpty    | 判断字符串是否为空。            | `str string`：要检查的字符串。                                                      | `bool`：字符串是否为空。                                       |
| StringToNum      | 将字符串转换为数字。            | `str string`：要转换的字符串。                                                      | `uint`：转换后的数字。<br>`error`：可能出现的错误。                    |
| ToString         | 将任意类型转换为字符串。          | `v interface{}`：要转换的值。                                                     | `string`：转换后的字符串。                                     |

## 前端查询
```json

{
    "filter": [
        "id",
        "name"
    ],
    "search": [
        {
            "operator": "and",
            "conditions": [
                [
                    "id",
                    "423223bc-044c-4be8-87d1-fa52dcc31183"
                ],
                [
                    "name",
                    "bobby",
                    "like"
                ],
                [
                    "created_at",
                    "2024-12-31 00:00:00",
                    ">"
                ]
            ]
        }
    ],
    "orderby": [
        [
            "created_at",
            "desc"
        ]
    ],
    "limit": [
        1,
        10
    ],
    "page": [
        1,
        30
    ]
}

```