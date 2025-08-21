package job_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/event_bus_provider"
)

// JobStatus 任务状态枚举
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"   // 等待中
	JobStatusRunning   JobStatus = "running"   // 执行中
	JobStatusCompleted JobStatus = "completed" // 已完成
	JobStatusFailed    JobStatus = "failed"    // 失败
	JobStatusRetrying  JobStatus = "retrying"  // 重试中
)

// Job 任务结构体
type Job struct {
	ID          string                 `json:"id"`           // 任务唯一ID
	Name        string                 `json:"name"`         // 任务名称
	Status      JobStatus              `json:"status"`       // 任务状态
	Payload     map[string]interface{} `json:"payload"`      // 任务载荷数据
	Handler     JobHandler             `json:"-"`            // 任务处理函数（不序列化）
	CreatedAt   time.Time              `json:"created_at"`   // 创建时间
	StartedAt   *time.Time             `json:"started_at"`   // 开始执行时间
	CompletedAt *time.Time             `json:"completed_at"` // 完成时间
	RetryCount  int                    `json:"retry_count"`  // 重试次数
	MaxRetries  int                    `json:"max_retries"`  // 最大重试次数
	Timeout     time.Duration          `json:"timeout"`      // 超时时间
	Error       string                 `json:"error"`        // 错误信息
}

// JobHandler 任务处理函数类型
type JobHandler func(ctx context.Context, job *Job) error

// JobEvent 任务事件结构体
type JobEvent struct {
	JobID  string    `json:"job_id"`
	Status JobStatus `json:"status"`
	Error  string    `json:"error,omitempty"`
}

// jobProvider 任务提供者结构体
type jobProvider struct {
	jobs        map[string]*Job       // 任务存储映射
	handlers    map[string]JobHandler // 任务处理器映射
	workerCount int                   // 工作者数量
	maxRetries  int                   // 最大重试次数
	timeout     time.Duration         // 默认超时时间
	redisPrefix string                // Redis键前缀
	mutex       sync.RWMutex          // 读写锁
	workerPool  chan struct{}         // 工作者池
	ctx         context.Context       // 上下文
	cancel      context.CancelFunc    // 取消函数
}

// JobProvider 全局任务提供者实例
var JobProvider jobProvider

// Init 初始化任务提供者
func (p *jobProvider) Init() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 从配置文件加载配置
	workerCount := z.Config.GetInt("config.job.worker_count", 4)
	if workerCount <= 0 {
		z.Error.Fatal(fmt.Sprintf("worker count must be positive, got %d", workerCount))
	}

	maxRetries := z.Config.GetInt("config.job.max_retries", 3)
	if maxRetries < 0 {
		z.Error.Fatal(fmt.Sprintf("max retries must be non-negative, got %d", maxRetries))
	}

	timeout := z.Config.GetInt("config.job.timeout", 30)
	if timeout <= 0 {
		z.Error.Fatal(fmt.Sprintf("timeout must be positive, got %d", timeout))
	}

	redisPrefix := z.Config.GetString("config.job.redis_prefix", "job:")
	if redisPrefix == "" {
		z.Error.Fatal("redis prefix cannot be empty")
	}

	p.workerCount = workerCount
	p.maxRetries = maxRetries
	p.timeout = time.Duration(timeout) * time.Second
	p.redisPrefix = redisPrefix

	// 初始化内部结构
	p.jobs = make(map[string]*Job)
	p.handlers = make(map[string]JobHandler)
	p.workerPool = make(chan struct{}, p.workerCount)

	// 创建上下文
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// 启动工作者池
	for i := 0; i < p.workerCount; i++ {
		go p.worker()
	}
}

// RegisterHandler 注册任务处理器
func (p *jobProvider) RegisterHandler(name string, handler JobHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.handlers[name] = handler
}

// AddJob 添加任务到队列
// args[0]: payload (可选) - 任务参数
// args[1]: delay (可选) - 延迟执行时间，默认立即执行
func (p *jobProvider) AddJob(ctx context.Context, name string, args ...interface{}) (string, error) {
	// 检查处理器是否存在
	p.mutex.RLock()
	handler, exists := p.handlers[name]
	p.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("handler not found for job: %s", name)
	}

	// 解析可选参数
	var payload map[string]interface{}
	var delay time.Duration

	if len(args) > 0 {
		if p, ok := args[0].(map[string]interface{}); ok {
			payload = p
		}
	}

	if len(args) > 1 {
		if d, ok := args[1].(time.Duration); ok {
			delay = d
		}
	}

	// 创建任务
	job := &Job{
		ID:         uuid.New().String(),
		Name:       name,
		Status:     JobStatusPending,
		Payload:    payload,
		Handler:    handler,
		CreatedAt:  time.Now(),
		RetryCount: 0,
		MaxRetries: p.maxRetries,
		Timeout:    p.timeout,
	}

	// 存储任务
	p.mutex.Lock()
	p.jobs[job.ID] = job
	p.mutex.Unlock()

	// 处理延迟执行
	if delay > 0 {
		// 延迟执行
		go func() {
			time.Sleep(delay)
			if err := p.pushToQueue(job); err != nil {
				z.Error.Printf("Failed to push delayed job to queue: %v", err)
				return
			}
			z.Info.Printf("Delayed job pushed to queue: %s (%s)", job.ID, job.Name)
		}()
	} else {
		// 立即执行
		if err := p.pushToQueue(job); err != nil {
			return "", fmt.Errorf("failed to push job to queue: %w", err)
		}
	}

	// 广播任务创建事件
	p.emitJobEvent(job.ID, JobStatusPending, "")

	z.Info.Printf("Job added: %s (%s), delay: %v", job.ID, job.Name, delay)
	return job.ID, nil
}

// GetJob 获取任务状态
func (p *jobProvider) GetJob(jobID string) (*Job, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	job, exists := p.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// GetJobs 获取所有任务状态
func (p *jobProvider) GetJobs() map[string]*Job {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 创建副本以避免并发问题
	jobs := make(map[string]*Job)
	for id, job := range p.jobs {
		jobs[id] = job
	}

	return jobs
}

// CountJobs 获取等待中任务数量
func (p *jobProvider) CountJobs(status JobStatus) int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	count := 0
	for _, job := range p.jobs {
		if job.Status == status {
			count++
		}
	}

	return count
}

// pushToQueue 将任务推送到Redis队列
func (p *jobProvider) pushToQueue(job *Job) error {
	// 序列化任务数据（不包含Handler）
	jobData := map[string]interface{}{
		"id":          job.ID,
		"name":        job.Name,
		"payload":     job.Payload,
		"created_at":  job.CreatedAt,
		"retry_count": job.RetryCount,
		"max_retries": job.MaxRetries,
		"timeout":     job.Timeout,
	}

	jobJSON, err := json.Marshal(jobData)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// 推送到Redis列表
	queueKey := p.redisPrefix + "queue"
	return z.RedisCache.Client().LPush(context.Background(), queueKey, jobJSON).Err()
}

// worker 工作者协程
func (p *jobProvider) worker() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// 从Redis队列获取任务
			if err := p.processNextJob(); err != nil {
				// 如果没有任务，稍等片刻
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNextJob 处理下一个任务
func (p *jobProvider) processNextJob() error {
	queueKey := p.redisPrefix + "queue"

	// 从Redis队列弹出任务
	result, err := z.RedisCache.Client().BRPop(context.Background(), 1*time.Second, queueKey).Result()
	if err != nil {
		return err // 可能是超时或没有任务
	}

	if len(result) < 2 {
		return fmt.Errorf("invalid queue result")
	}

	// 解析任务数据
	var jobData map[string]interface{}
	if err := json.Unmarshal([]byte(result[1]), &jobData); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	jobID, ok := jobData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid job ID")
	}

	// 获取完整的任务信息
	p.mutex.RLock()
	job, exists := p.jobs[jobID]
	p.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("job not found in memory: %s", jobID)
	}

	// 执行任务
	p.executeJob(job)
	return nil
}

// executeJob 执行任务
func (p *jobProvider) executeJob(job *Job) {
	// 更新任务状态为运行中
	now := time.Now()
	job.Status = JobStatusRunning
	job.StartedAt = &now

	// 广播任务开始事件
	p.emitJobEvent(job.ID, JobStatusRunning, "")

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
	defer cancel()

	// 执行任务处理函数
	err := job.Handler(ctx, job)

	// 更新完成时间
	completedAt := time.Now()
	job.CompletedAt = &completedAt

	if err != nil {
		// 任务执行失败
		job.Error = err.Error()
		job.RetryCount++

		if job.RetryCount < job.MaxRetries {
			// 需要重试
			job.Status = JobStatusRetrying
			p.emitJobEvent(job.ID, JobStatusRetrying, err.Error())

			// 重新推送到队列
			if pushErr := p.pushToQueue(job); pushErr != nil {
				z.Error.Printf("Failed to retry job %s: %v", job.ID, pushErr)
				job.Status = JobStatusFailed
				p.emitJobEvent(job.ID, JobStatusFailed, err.Error())
			}
		} else {
			// 重试次数已达上限，标记为失败
			job.Status = JobStatusFailed
			p.emitJobEvent(job.ID, JobStatusFailed, err.Error())
		}

		z.Error.Printf("Job failed: %s (%s) - %v", job.ID, job.Name, err)
	} else {
		// 任务执行成功
		job.Status = JobStatusCompleted
		job.Error = ""
		p.emitJobEvent(job.ID, JobStatusCompleted, "")

		z.Info.Printf("Job completed: %s (%s)", job.ID, job.Name)
	}
}

// emitJobEvent 广播任务事件
func (p *jobProvider) emitJobEvent(jobID string, status JobStatus, errorMsg string) {
	event := JobEvent{
		JobID:  jobID,
		Status: status,
		Error:  errorMsg,
	}

	// 使用事件总线广播事件
	event_bus_provider.EmitAsync(context.Background(), "job.status.changed", event)
}

// Shutdown 关闭任务提供者
func (p *jobProvider) Shutdown() {
	if p.cancel != nil {
		p.cancel()
	}
	z.Info.Println("JobProvider shutdown")
}
