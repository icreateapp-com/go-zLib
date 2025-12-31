# 前端 API 查询参数说明

本文档面向前端开发人员，说明如何通过 URL 查询参数向后端传递查询条件、排序与分页。

系统支持两种查询方案：

- **方案 A：便捷查询（URL 参数）**
  - 易读易写，适合简单列表筛选
  - 不支持复杂条件组（OR/嵌套分组）
- **方案 B：高级查询（`query` JSON）**（推荐）
  - 功能更强，支持条件组/更复杂的组合查询
  - 需要把 JSON 进行 URL 编码后放入 `query` 参数

安全约束：为避免危险操作，本接口文档不提供字段筛选（filter）与关联预加载（include）能力。

---

## 方案 A：便捷查询（URL 参数）

### 支持的参数

- `search`：搜索条件
- `orderby`：排序
- `page`：页码（从 1 开始）
- `limit`：每页数量（最大 100）

### 1) search

#### 格式

```
search=字段1:值1[:操作符1]|字段2:值2[:操作符2]|...
```

#### 规则

- 多个条件用 `|` 分隔
- 每个条件用 `:` 分隔
  - `字段:值`（默认操作符 `=`）
  - 或 `字段:值:操作符`

#### 支持的操作符（仅允许下划线风格）

| 操作符 | 说明 | 示例 |
|--------|------|------|
| = | 等于（默认） | status:1 |
| != 或 <> | 不等于 | status:0:!= |
| > | 大于 | age:18:> |
| >= | 大于等于 | age:18:>= |
| < | 小于 | price:100:< |
| <= | 小于等于 | price:100:<= |
| like | 模糊匹配 | name:john:like |
| not_like | 非模糊匹配 | city:paris:not_like |
| left_like | 左模糊匹配 | email:gmail:left_like |
| right_like | 右模糊匹配 | domain:com:right_like |
| in | 在指定范围内 | status:1,2,3:in |
| not_in | 不在指定范围内 | role:admin,user:not_in |
| is_null | 为空 | deleted_at:null:is_null |
| is_not_null | 不为空 | email:null:is_not_null |
| between | 区间 | created_at:2025-01-01,2025-01-31:between |
| not_between | 非区间 | created_at:2025-01-01,2025-01-31:not_between |

#### 示例

```
/users?search=name:john:like|status:1
/users?search=age:18:>=|city:北京
/users?search=city:paris:not_like
```

### 2) orderby

#### 格式

```
orderby=字段1:方向1|字段2:方向2|...
```

其中方向为：`asc` / `desc`

#### 示例

```
/users?orderby=created_at:desc|name:asc
```

### 3) page / limit

- `page`：页码，从 1 开始
- `limit`：每页条数，最大 100

#### 示例

```
/users?page=1&limit=10
```

---

## 方案 B：高级查询（`query` JSON）（推荐）

当查询需要“分组/OR/更复杂的组合条件”时，请使用 `query` 参数传递 JSON。

### 1) 基本用法

#### 格式

```
/users?query={JSON}
```

注意：`{JSON}` 必须进行 URL 编码（例如使用 `encodeURIComponent`）。

#### 示例（JavaScript）

```js
const query = {
  search: [
    {
      operator: "OR",
      conditions: [
        ["status", 1, "="],
        ["status", 2, "="],
      ],
    },
    {
      operator: "AND",
      conditions: [
        ["name", "john", "like"],
        ["age", 18, ">="],
      ],
    },
  ],
  orderby: [["created_at", "desc"]],
  page: 1,
  limit: 10,
}

const url = `/users?query=${encodeURIComponent(JSON.stringify(query))}`
```

### 2) JSON 结构

`query` JSON 支持以下字段：

- `search`: 条件组数组
- `orderby`: 排序数组
- `page`: 页码
- `limit`: 每页数量

#### `search`（条件组）

`search` 是一个数组，每个元素表示一个“条件组”。

- `operator`: 组内连接符，支持 `AND` / `OR`（大小写不敏感）
- `conditions`: 条件列表

每条 condition 的格式为：

```
[字段, 值, 操作符]
```

操作符请使用与“便捷查询”一致的集合（同样仅推荐下划线风格）。

#### `orderby`

格式：

```
orderby: [[字段, 方向], ...]
```

---

## 综合示例

```
/users?search=name:john:like|status:1&orderby=created_at:desc&page=2&limit=15
```

---

## 注意事项

1. 操作符请统一使用下划线风格（如 `is_null` / `not_like`），不要在 URL 里直接传空格。
2. `query` JSON 必须进行 URL 编码。
3. `limit` 最大为 100，超过会按 100 处理。