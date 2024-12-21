# `zLib` 包中 `Cron` 说明文档：

```markdown
# zLib 包定时任务说明文档

## `_cron` 结构体

`_cron` 结构体用于封装 `cron.Cron`，提供定时任务的管理功能。

## 全局变量

- `Cron`: 全局 `_cron` 对象，可以在包外部使用。

## 方法

### `Init`

初始化定时任务管理器。

```go
func (p *_cron) Init(opts ...cron.Option)
```

#### 参数

- `opts` (`...cron.Option`): 可选参数，用于配置 `cron.Cron` 实例。

### `Add`

增加定时任务。

```go
func (p *_cron) Add(spec string, cmd func()) (cron.EntryID, error)
```

#### 参数

- `spec` (`string`): Cron 表达式，用于指定任务的执行时间。
- `cmd` (`func()`): 任务函数，当定时任务触发时执行。

#### 返回值

- `cron.EntryID`: 任务 ID，用于后续操作（如删除任务）。
- `error`: 如果添加任务时发生错误，返回错误信息。

### `Remove`

删除定时任务。

```go
func (p *_cron) Remove(id cron.EntryID)
```

#### 参数

- `id` (`cron.EntryID`): 要删除的任务 ID。

### `Start`

后台运行定时任务。

```go
func (p *_cron) Start()
```

此方法会启动定时任务管理器，并在后台执行任务。

### `Run`

前台运行定时任务。

```go
func (p *_cron) Run()
```

此方法会启动定时任务管理器，并在前台阻塞执行任务。适用于希望程序在主线程中等待定时任务执行完毕的场景。

