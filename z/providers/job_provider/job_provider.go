package job_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/event_bus_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/redis_provider"
	"go.uber.org/fx"
)

// JobStatus 任务状态枚举
// 与旧版 z/provider/job_provider 保持一致
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
)

// Job 任务结构体（尽量与旧版字段保持一致）
// 注意：asynq 的 payload 必须可序列化，因此这里把 Payload 约束为 json.RawMessage
// 旧版的 interface{} 在分布式队列下不可直接传递。
type Job struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Status      JobStatus       `json:"status"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	Timeout     time.Duration   `json:"timeout"`
	Error       string          `json:"error"`
}

// JobHandler 任务处理函数类型（保持旧版签名）
// ctx 来自 asynq worker context
// job 为我们维护的 job 元数据（包含 payload 等）
type JobHandler func(ctx context.Context, job *Job) error

// JobEvent 任务事件结构体（与旧版一致）
type JobEvent struct {
	JobID  string    `json:"job_id"`
	Status JobStatus `json:"status"`
	Error  string    `json:"error,omitempty"`
}

// JobHandlerRegister 由业务模块提供，用于注册任务处理器（fx group）
type JobHandlerRegister struct {
	Name    string
	Handler JobHandler
}

// JobClient 用于在业务逻辑中 enqueue 任务（分布式场景：web 节点只需要 JobClient）
type JobClient struct {
	client     *asynq.Client
	log        *logger_provider.Logger
	bus        *event_bus_provider.EventBus
	queue      string
	maxRetries int
	timeout    time.Duration
}

type AddJobOptions struct {
	Delay     time.Duration
	ProcessAt *time.Time

	MaxRetry *int
	Timeout  *time.Duration

	Queue    *string
	Priority *int

	UniqueTTL *time.Duration
	TaskID    *string
	Retention *time.Duration
}

// JobWorker 用于运行 worker 并执行任务（分布式场景：worker 节点只需要 JobWorker + 业务模块提供的 handlers）
type JobWorker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	log    *logger_provider.Logger
	bus    *event_bus_provider.EventBus
}

type ClientIn struct {
	fx.In
	Cfg   *config_provider.Config
	Log   *logger_provider.Logger
	Redis *redis_provider.Redis        `optional:"true"`
	Bus   *event_bus_provider.EventBus `optional:"true"`
}

func NewJobClient(in ClientIn) (*JobClient, error) {
	var client *asynq.Client
	if in.Redis != nil {
		client = asynq.NewClientFromRedisClient(in.Redis.Client())
	} else {
		// 兼容：允许 job.yml 单独配置 redis
		redisHost := strings.TrimSpace(in.Cfg.GetString("job.redis.host"))
		redisPort := in.Cfg.GetInt("job.redis.port", 6379)
		redisPassword := in.Cfg.GetString("job.redis.password")
		redisDB := in.Cfg.GetInt("job.redis.db", 0)
		redisAddr := strings.TrimSpace(in.Cfg.GetString("job.redis.addr", ""))
		if redisHost != "" {
			redisAddr = fmt.Sprintf("%s:%d", redisHost, redisPort)
		}
		if redisAddr == "" {
			redisAddr = "127.0.0.1:6379"
		}
		redisOpt := asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: redisDB}
		client = asynq.NewClient(redisOpt)
	}

	queue := in.Cfg.GetString("job.queue", "default")
	maxRetries := in.Cfg.GetInt("job.max_retries", 3)
	timeoutSeconds := in.Cfg.GetInt("job.timeout", 3600)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 3600
	}
	timeout := time.Duration(timeoutSeconds) * time.Second

	return &JobClient{client: client, log: in.Log, bus: in.Bus, queue: queue, maxRetries: maxRetries, timeout: timeout}, nil
}

type WorkerIn struct {
	fx.In
	LC       fx.Lifecycle
	Cfg      *config_provider.Config
	Log      *logger_provider.Logger
	Redis    *redis_provider.Redis        `optional:"true"`
	Bus      *event_bus_provider.EventBus `optional:"true"`
	Handlers []JobHandlerRegister         `group:"job_handlers"`
}

func NewJobWorker(in WorkerIn) (*JobWorker, error) {
	queue := in.Cfg.GetString("job.queue", "default")
	concurrency := in.Cfg.GetInt("job.concurrency", 10)
	if concurrency <= 0 {
		concurrency = 10
	}

	redisDesc := ""
	var server *asynq.Server
	serverCfg := asynq.Config{
		Concurrency: concurrency,
		Queues: map[string]int{
			queue: 1,
		},
	}
	if in.Redis != nil {
		redisOpt := in.Redis.Client().Options()
		if redisOpt != nil {
			redisDesc = redisOpt.Addr
		}
		server = asynq.NewServerFromRedisClient(in.Redis.Client(), serverCfg)
	} else {
		// 兼容：允许 job.yml 单独配置 redis
		redisHost := strings.TrimSpace(in.Cfg.GetString("job.redis.host"))
		redisPort := in.Cfg.GetInt("job.redis.port", 6379)
		redisPassword := in.Cfg.GetString("job.redis.password")
		redisDB := in.Cfg.GetInt("job.redis.db", 0)
		redisAddr := strings.TrimSpace(in.Cfg.GetString("job.redis.addr", ""))
		if redisHost != "" {
			redisAddr = fmt.Sprintf("%s:%d", redisHost, redisPort)
		}
		if redisAddr == "" {
			redisAddr = "127.0.0.1:6379"
		}
		redisDesc = redisAddr
		redisOpt := asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: redisDB}
		server = asynq.NewServer(redisOpt, serverCfg)
	}

	mux := asynq.NewServeMux()
	registered := 0
	w := &JobWorker{server: server, mux: mux, log: in.Log, bus: in.Bus}
	for _, r := range in.Handlers {
		name := strings.TrimSpace(r.Name)
		if name == "" || r.Handler == nil {
			continue
		}
		registered++
		mux.HandleFunc(name, func(h JobHandler) func(context.Context, *asynq.Task) error {
			return func(ctx context.Context, task *asynq.Task) error {
				var job Job
				if err := json.Unmarshal(task.Payload(), &job); err != nil {
					return err
				}
				w.emitJobEvent(job.ID, JobStatusRunning, "")
				now := time.Now()
				job.StartedAt = &now
				err := h(ctx, &job)
				completedAt := time.Now()
				job.CompletedAt = &completedAt
				if err != nil {
					w.emitJobEvent(job.ID, JobStatusFailed, err.Error())
					if in.Log != nil {
						in.Log.Infow("job executed", "id", job.ID, "name", job.Name, "status", "failed", "error", err.Error())
					}
					return err
				}
				w.emitJobEvent(job.ID, JobStatusCompleted, "")
				if in.Log != nil {
					in.Log.Infow("job executed", "id", job.ID, "name", job.Name, "status", "completed")
				}
				return nil
			}
		}(r.Handler))
	}

	in.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// server.Run 会阻塞，因此异步启动
			go func() {
				if err := w.server.Run(w.mux); err != nil {
					if w.log != nil {
						w.log.Errorw("asynq server stopped", "error", err)
					}
				}
			}()
			if w.log != nil {
				w.log.Infow("provider[job_worker] enabled", "redis", redisDesc, "queue", queue, "concurrency", concurrency, "handlers", registered)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			w.server.Stop()
			w.server.Shutdown()
			return nil
		},
	})

	return w, nil
}

// AddJob 添加任务并入队（破坏性改造：使用 payload + options，更符合 asynq 习惯）
func (c *JobClient) AddJob(ctx context.Context, name string, payload any, opt *AddJobOptions) (*asynq.TaskInfo, error) {
	if opt == nil {
		opt = &AddJobOptions{}
	}
	if opt.Delay > 0 && opt.ProcessAt != nil {
		return nil, fmt.Errorf("job: Delay and ProcessAt cannot be set at the same time")
	}

	queue := c.queue
	if opt.Queue != nil && strings.TrimSpace(*opt.Queue) != "" {
		queue = strings.TrimSpace(*opt.Queue)
	}

	maxRetry := c.maxRetries
	if opt.MaxRetry != nil {
		maxRetry = *opt.MaxRetry
	}

	timeout := c.timeout
	if opt.Timeout != nil {
		timeout = *opt.Timeout
	}

	jobID := uuid.New().String()
	job := &Job{
		ID:         jobID,
		Name:       name,
		Status:     JobStatusPending,
		CreatedAt:  time.Now(),
		RetryCount: 0,
		MaxRetries: maxRetry,
		Timeout:    timeout,
	}

	// payload 序列化
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		job.Payload = b
	}

	jobBytes, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(name, jobBytes)
	opts := []asynq.Option{
		asynq.Queue(queue),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(timeout),
	}
	if opt.Delay > 0 {
		opts = append(opts, asynq.ProcessIn(opt.Delay))
	}
	if opt.ProcessAt != nil {
		opts = append(opts, asynq.ProcessAt(*opt.ProcessAt))
	}
	if opt.UniqueTTL != nil && *opt.UniqueTTL > 0 {
		opts = append(opts, asynq.Unique(*opt.UniqueTTL))
	}
	if opt.TaskID != nil && strings.TrimSpace(*opt.TaskID) != "" {
		opts = append(opts, asynq.TaskID(strings.TrimSpace(*opt.TaskID)))
	}
	if opt.Retention != nil && *opt.Retention > 0 {
		opts = append(opts, asynq.Retention(*opt.Retention))
	}

	info, err := c.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		// asynq 在相同 TaskID 已存在时会返回冲突错误；对业务层来说这是幂等场景，自动忽略并返回成功
		if strings.Contains(err.Error(), "task ID conflicts with another task") {
			if c.log != nil {
				c.log.Infow("job already exists, skip enqueue", "name", name, "id", jobID)
			}
			return nil, nil
		}
		return nil, err
	}

	c.emitJobEvent(jobID, JobStatusPending, "")
	if c.log != nil {
		c.log.Infow("job enqueued", "id", jobID, "name", name, "queue", info.Queue, "next_process_at", info.NextProcessAt)
	}

	return info, nil
}

func (c *JobClient) emitJobEvent(jobID string, status JobStatus, errorMsg string) {
	if c == nil || c.bus == nil {
		return
	}
	event := JobEvent{JobID: jobID, Status: status, Error: errorMsg}
	// 复用旧事件名
	c.bus.EmitAsync(context.Background(), "job.status.changed", event)
}

func (w *JobWorker) emitJobEvent(jobID string, status JobStatus, errorMsg string) {
	if w == nil || w.bus == nil {
		return
	}
	event := JobEvent{JobID: jobID, Status: status, Error: errorMsg}
	// 复用旧事件名
	w.bus.EmitAsync(context.Background(), "job.status.changed", event)
}

type HandlerOut struct {
	fx.Out
	Handler JobHandlerRegister `group:"job_handlers"`
}

func Register(name string, handler JobHandler) HandlerOut {
	return HandlerOut{Handler: JobHandlerRegister{Name: name, Handler: handler}}
}

var JobProviderModule = fx.Options(
	fx.Provide(NewJobClient),
	fx.Invoke(func(_ *JobClient) {}),
	fx.Provide(NewJobWorker),
	fx.Invoke(func(_ *JobWorker) {}),
)
