/*
 * @module service/models/scheduler_models
 * @description 调度器相关模型定义，包含调度任务、任务执行、重试管理等核心数据结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 任务调度 -> 任务执行 -> 结果处理
 * @rules 确保调度器模型的一致性和完整性
 * @dependencies time, context
 * @refs service/scheduler
 */

package models

import (
	"context"
	"time"
)

// ScheduleTask 调度任务模型
type ScheduleTask struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`               // 任务名称
	Type             string                 `json:"type"`               // 任务类型: sync, cleanup, maintenance, etc.
	Category         string                 `json:"category"`           // 任务分类: data_sync, system, business
	Priority         int                    `json:"priority"`           // 优先级: 1-10, 数字越大优先级越高
	CronExpression   string                 `json:"cron_expression"`    // Cron表达式
	IntervalSeconds  int64                  `json:"interval_seconds"`   // 间隔执行（秒）
	TimeoutSeconds   int64                  `json:"timeout_seconds"`    // 超时时间（秒）
	MaxRetries       int                    `json:"max_retries"`        // 最大重试次数
	RetryInterval    int64                  `json:"retry_interval"`     // 重试间隔（秒）
	Config           map[string]interface{} `json:"config"`             // 任务配置参数
	Enabled          bool                   `json:"enabled"`            // 是否启用
	NextRunTime      time.Time              `json:"next_run_time"`      // 下次执行时间
	LastRunTime      *time.Time             `json:"last_run_time"`      // 上次执行时间
	LastStatus       string                 `json:"last_status"`        // 上次执行状态
	LastDuration     int64                  `json:"last_duration"`      // 上次执行时长（毫秒）
	LastErrorMessage string                 `json:"last_error_message"` // 上次错误信息
	ExecutionCount   int64                  `json:"execution_count"`    // 总执行次数
	SuccessCount     int64                  `json:"success_count"`      // 成功次数
	FailureCount     int64                  `json:"failure_count"`      // 失败次数
	Description      string                 `json:"description"`        // 任务描述
	Owner            string                 `json:"owner"`              // 任务负责人
	Tags             []string               `json:"tags"`               // 标签
	Dependencies     []string               `json:"dependencies"`       // 依赖的其他任务ID
	Timezone         string                 `json:"timezone"`           // 时区
	CreatedAt        time.Time              `json:"created_at"`         // 创建时间
	UpdatedAt        time.Time              `json:"updated_at"`         // 更新时间
	CreatedBy        string                 `json:"created_by"`         // 创建者
	UpdatedBy        string                 `json:"updated_by"`         // 更新者
}

// TaskExecution 任务执行记录模型
type TaskExecution struct {
	ID             string                 `json:"id"`
	TaskID         string                 `json:"task_id"`         // 关联的任务ID
	TaskName       string                 `json:"task_name"`       // 任务名称
	ExecutionType  string                 `json:"execution_type"`  // 执行类型: scheduled, manual, retry
	Status         string                 `json:"status"`          // 执行状态: pending, running, success, failed, timeout, cancelled
	StartTime      time.Time              `json:"start_time"`      // 开始时间
	EndTime        *time.Time             `json:"end_time"`        // 结束时间
	Duration       int64                  `json:"duration"`        // 执行时长（毫秒）
	Progress       int                    `json:"progress"`        // 执行进度（0-100）
	ProcessedCount int64                  `json:"processed_count"` // 已处理数量
	TotalCount     int64                  `json:"total_count"`     // 总数量
	SuccessCount   int64                  `json:"success_count"`   // 成功数量
	ErrorCount     int64                  `json:"error_count"`     // 错误数量
	ErrorMessage   string                 `json:"error_message"`   // 错误信息
	ErrorDetail    string                 `json:"error_detail"`    // 错误详情
	StackTrace     string                 `json:"stack_trace"`     // 堆栈跟踪
	Result         map[string]interface{} `json:"result"`          // 执行结果
	Log            []string               `json:"log"`             // 执行日志
	Metrics        map[string]interface{} `json:"metrics"`         // 执行指标
	Context        map[string]interface{} `json:"context"`         // 执行上下文
	HostName       string                 `json:"host_name"`       // 执行主机
	ProcessID      int                    `json:"process_id"`      // 进程ID
	ThreadID       string                 `json:"thread_id"`       // 线程ID
	RetryCount     int                    `json:"retry_count"`     // 重试次数
	IsRetry        bool                   `json:"is_retry"`        // 是否为重试执行
	ParentExecID   string                 `json:"parent_exec_id"`  // 父执行ID（重试时指向原始执行）
	CreatedAt      time.Time              `json:"created_at"`      // 创建时间
	UpdatedAt      time.Time              `json:"updated_at"`      // 更新时间
}

// TaskDependency 任务依赖关系模型
type TaskDependency struct {
	ID                string    `json:"id"`
	TaskID            string    `json:"task_id"`             // 当前任务ID
	DependentTaskID   string    `json:"dependent_task_id"`   // 依赖的任务ID
	DependencyType    string    `json:"dependency_type"`     // 依赖类型: success, completion, time_based
	WaitForSuccess    bool      `json:"wait_for_success"`    // 是否等待成功
	WaitForCompletion bool      `json:"wait_for_completion"` // 是否等待完成
	TimeOffset        int64     `json:"time_offset"`         // 时间偏移（秒）
	IsEnabled         bool      `json:"is_enabled"`          // 是否启用
	Description       string    `json:"description"`         // 依赖描述
	CreatedAt         time.Time `json:"created_at"`          // 创建时间
	UpdatedAt         time.Time `json:"updated_at"`          // 更新时间
}

// TaskQueue 任务队列模型
type TaskQueue struct {
	ID          string     `json:"id"`
	QueueName   string     `json:"queue_name"`   // 队列名称
	TaskID      string     `json:"task_id"`      // 任务ID
	TaskName    string     `json:"task_name"`    // 任务名称
	Priority    int        `json:"priority"`     // 优先级
	Status      string     `json:"status"`       // 状态: waiting, processing, completed, failed
	ScheduledAt time.Time  `json:"scheduled_at"` // 计划执行时间
	StartedAt   *time.Time `json:"started_at"`   // 开始处理时间
	CompletedAt *time.Time `json:"completed_at"` // 完成时间
	WorkerID    string     `json:"worker_id"`    // 处理器ID
	RetryCount  int        `json:"retry_count"`  // 重试次数
	ErrorMsg    string     `json:"error_msg"`    // 错误信息
	CreatedAt   time.Time  `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time  `json:"updated_at"`   // 更新时间
}

// TaskWorker 任务工作器模型
type TaskWorker struct {
	ID             string                 `json:"id"`
	WorkerName     string                 `json:"worker_name"`      // 工作器名称
	WorkerType     string                 `json:"worker_type"`      // 工作器类型
	Status         string                 `json:"status"`           // 状态: idle, busy, stopped, error
	CurrentTaskID  string                 `json:"current_task_id"`  // 当前处理的任务ID
	ProcessedCount int64                  `json:"processed_count"`  // 已处理任务数
	SuccessCount   int64                  `json:"success_count"`    // 成功任务数
	ErrorCount     int64                  `json:"error_count"`      // 失败任务数
	StartTime      time.Time              `json:"start_time"`       // 启动时间
	LastActiveTime time.Time              `json:"last_active_time"` // 最后活跃时间
	HostName       string                 `json:"host_name"`        // 主机名
	ProcessID      int                    `json:"process_id"`       // 进程ID
	Config         map[string]interface{} `json:"config"`           // 工作器配置
	CreatedAt      time.Time              `json:"created_at"`       // 创建时间
	UpdatedAt      time.Time              `json:"updated_at"`       // 更新时间
}

// RetryPolicy 重试策略模型
type RetryPolicy struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`                 // 策略名称
	MaxRetries          int           `json:"max_retries"`          // 最大重试次数
	InitialInterval     time.Duration `json:"initial_interval"`     // 初始重试间隔
	MaxInterval         time.Duration `json:"max_interval"`         // 最大重试间隔
	Multiplier          float64       `json:"multiplier"`           // 退避乘数
	RandomizationFactor float64       `json:"randomization_factor"` // 随机化因子
	RetryableErrors     []string      `json:"retryable_errors"`     // 可重试的错误类型
	NonRetryableErrors  []string      `json:"non_retryable_errors"` // 不可重试的错误类型
	BackoffType         string        `json:"backoff_type"`         // 退避类型: fixed, exponential, linear
	JitterType          string        `json:"jitter_type"`          // 抖动类型: none, full, equal
	IsEnabled           bool          `json:"is_enabled"`           // 是否启用
	Description         string        `json:"description"`          // 策略描述
	CreatedAt           time.Time     `json:"created_at"`           // 创建时间
	UpdatedAt           time.Time     `json:"updated_at"`           // 更新时间
}

// TaskAlert 任务告警模型
type TaskAlert struct {
	ID                string                 `json:"id"`
	TaskID            string                 `json:"task_id"`            // 任务ID
	TaskName          string                 `json:"task_name"`          // 任务名称
	AlertType         string                 `json:"alert_type"`         // 告警类型: failure, timeout, delay, resource
	AlertLevel        string                 `json:"alert_level"`        // 告警级别: info, warning, error, critical
	AlertMessage      string                 `json:"alert_message"`      // 告警消息
	AlertDetail       string                 `json:"alert_detail"`       // 告警详情
	TriggerValue      interface{}            `json:"trigger_value"`      // 触发值
	ThresholdValue    interface{}            `json:"threshold_value"`    // 阈值
	AlertRule         map[string]interface{} `json:"alert_rule"`         // 告警规则
	Status            string                 `json:"status"`             // 状态: active, resolved, suppressed
	TriggerTime       time.Time              `json:"trigger_time"`       // 触发时间
	ResolveTime       *time.Time             `json:"resolve_time"`       // 解决时间
	Duration          int64                  `json:"duration"`           // 持续时间（秒）
	NotificationsSent int                    `json:"notifications_sent"` // 已发送通知数
	AckBy             string                 `json:"ack_by"`             // 确认人
	AckAt             *time.Time             `json:"ack_at"`             // 确认时间
	Note              string                 `json:"note"`               // 备注
	CreatedAt         time.Time              `json:"created_at"`         // 创建时间
	UpdatedAt         time.Time              `json:"updated_at"`         // 更新时间
}

// TaskStatistics 任务统计模型
type TaskStatistics struct {
	ID                string                 `json:"id"`
	TaskID            string                 `json:"task_id"`            // 任务ID
	StatDate          time.Time              `json:"stat_date"`          // 统计日期
	TotalExecutions   int64                  `json:"total_executions"`   // 总执行次数
	SuccessExecutions int64                  `json:"success_executions"` // 成功执行次数
	FailedExecutions  int64                  `json:"failed_executions"`  // 失败执行次数
	TimeoutExecutions int64                  `json:"timeout_executions"` // 超时执行次数
	AvgDuration       float64                `json:"avg_duration"`       // 平均执行时长（毫秒）
	MinDuration       int64                  `json:"min_duration"`       // 最短执行时长（毫秒）
	MaxDuration       int64                  `json:"max_duration"`       // 最长执行时长（毫秒）
	SuccessRate       float64                `json:"success_rate"`       // 成功率
	AvgRetryCount     float64                `json:"avg_retry_count"`    // 平均重试次数
	TotalProcessed    int64                  `json:"total_processed"`    // 总处理数量
	TotalErrors       int64                  `json:"total_errors"`       // 总错误数量
	ResourceUsage     map[string]interface{} `json:"resource_usage"`     // 资源使用情况
	CreatedAt         time.Time              `json:"created_at"`         // 创建时间
	UpdatedAt         time.Time              `json:"updated_at"`         // 更新时间
}

// TaskContext 任务执行上下文
type TaskContext struct {
	TaskID      string                 `json:"task_id"`
	ExecutionID string                 `json:"execution_id"`
	Context     context.Context        `json:"-"`
	Cancel      context.CancelFunc     `json:"-"`
	StartTime   time.Time              `json:"start_time"`
	TimeoutTime time.Time              `json:"timeout_time"`
	Config      map[string]interface{} `json:"config"`
	Progress    *TaskProgress          `json:"progress"`
	Logger      interface{}            `json:"-"` // 日志记录器
	Metrics     map[string]interface{} `json:"metrics"`
	Data        map[string]interface{} `json:"data"`
	Errors      []error                `json:"-"`
	IsRetry     bool                   `json:"is_retry"`
	RetryCount  int                    `json:"retry_count"`
	WorkerID    string                 `json:"worker_id"`
}

// TaskProgress 任务进度
type TaskProgress struct {
	ProcessedCount int64     `json:"processed_count"` // 已处理数量
	TotalCount     int64     `json:"total_count"`     // 总数量
	SuccessCount   int64     `json:"success_count"`   // 成功数量
	ErrorCount     int64     `json:"error_count"`     // 错误数量
	SkippedCount   int64     `json:"skipped_count"`   // 跳过数量
	Percentage     float64   `json:"percentage"`      // 完成百分比
	CurrentPhase   string    `json:"current_phase"`   // 当前阶段
	EstimatedTime  time.Time `json:"estimated_time"`  // 预计完成时间
	Speed          float64   `json:"speed"`           // 处理速度（单位/秒）
	UpdatedAt      time.Time `json:"updated_at"`      // 更新时间
}

// TaskHandlerFunc 任务处理函数类型
type TaskHandlerFunc func(ctx *TaskContext) error

// TaskSchedulerConfig 任务调度器配置
type TaskSchedulerConfig struct {
	MaxWorkers         int           `json:"max_workers"`          // 最大工作器数量
	QueueSize          int           `json:"queue_size"`           // 队列大小
	CheckInterval      time.Duration `json:"check_interval"`       // 检查间隔
	WorkerTimeout      time.Duration `json:"worker_timeout"`       // 工作器超时时间
	HeartbeatInterval  time.Duration `json:"heartbeat_interval"`   // 心跳间隔
	CleanupInterval    time.Duration `json:"cleanup_interval"`     // 清理间隔
	RetentionDays      int           `json:"retention_days"`       // 数据保留天数
	EnabledMetrics     bool          `json:"enabled_metrics"`      // 是否启用指标收集
	EnabledHealthCheck bool          `json:"enabled_health_check"` // 是否启用健康检查
	PanicRecovery      bool          `json:"panic_recovery"`       // 是否启用panic恢复
	LogLevel           string        `json:"log_level"`            // 日志级别
}
