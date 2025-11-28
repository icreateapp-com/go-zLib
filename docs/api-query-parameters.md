# 前端API查询参数说明

本文档详细说明了前端如何通过URL查询参数与后端API进行交互，包括搜索条件、排序规则、分页控制等。

## 概述

前端可以通过URL查询参数与后端进行交互，系统支持多种参数类型，包括：
- `search`：搜索条件参数
- `orderby`：排序参数
- `limit`：限制返回记录数
- `page`：分页参数

## 1. search 参数

### 基本格式
```
search=字段1:值1[:操作符1],字段2:值2[:操作符2],...
```

### 参数说明
- **字段**：数据库表中的字段名
- **值**：要搜索的值
- **操作符**（可选）：指定比较操作，默认为 "="

### 支持的操作符

| 操作符 | 说明 | 示例 | 生成的SQL |
|--------|------|------|-----------|
| = | 等于（默认） | name:john | name = 'john' |
| != 或 <> | 不等于 | status:0:!= | status != 0 |
| > | 大于 | age:18:> | age > 18 |
| >= | 大于等于 | score:60:>= | score >= 60 |
| < | 小于 | price:100:< | price < 100 |
| <= | 小于等于 | quantity:10:<= | quantity <= 10 |
| like | 模糊匹配 | name:john:like | name LIKE '%john%' |
| not like | 非模糊匹配 | city:paris:not like | city NOT LIKE '%paris%' |
| left like | 左模糊匹配 | email:gmail:left like | email LIKE '%gmail' |
| right like | 右模糊匹配 | domain:com:right like | domain LIKE 'com%' |
| in | 在指定范围内 | status:1,2,3:in | status IN (1,2,3) |
| not in | 不在指定范围内 | role:admin,user:not in | role NOT IN ('admin','user') |
| is null | 为空 | deleted_at:null:is null | deleted_at IS NULL |
| is not null | 不为空 | email:null:is not null | email IS NOT NULL |

### 示例
```
# 搜索名字包含 john 且状态为 1 的用户
/search?search=name:john:like,status:1

# 搜索年龄大于等于18且城市为北京或上海的用户
/search?search=age:18:>=,city:北京,city:上海

# 搜索邮箱不为空的用户
/search?search=email:null:is not null
```

## 2. orderby 参数

### 基本格式
```
orderby=字段1:排序方向1,字段2:排序方向2,...
```

### 参数说明
- **字段**：数据库表中的字段名
- **排序方向**：`asc`（升序）或 `desc`（降序）

### 示例
```
# 按创建时间倒序排列，再按名字正序排列
/search?orderby=created_at:desc,name:asc

# 按ID倒序排列
/search?orderby=id:desc
```

## 3. limit 参数

### 基本格式
```
limit=数字
```

### 参数说明
- **数字**：限制返回的记录数量，最大值为100

### 示例
```
# 限制返回10条记录
/search?limit=10

# 限制返回50条记录
/search?limit=50
```

## 4. page 参数

### 基本格式
```
page=页码
```

### 参数说明
- **页码**：从1开始的页码数
- 默认每页显示数量为20条记录，可通过limit参数调整
- 最大每页显示100条记录

### 示例
```
# 获取第1页数据（默认每页20条）
/search?page=1

# 获取第3页数据，每页10条
/search?page=3&limit=10
```

## 综合示例

```
# 搜索名字包含 john，状态为 1 的用户，
# 按创建时间倒序排列，
# 获取第2页，每页15条记录
/users?search=name:john:like,status:1&orderby=created_at:desc&page=2&limit=15
```

## 注意事项

1. 所有参数都是可选的，可以根据需要组合使用
2. 多个条件之间使用英文逗号`,`分隔
3. 字段名和值中如果包含特殊字符，需要进行URL编码
4. limit参数最大值为100，超过将被截断
5. page参数从1开始计数
6. orderby参数默认排序方向为升序(asc)