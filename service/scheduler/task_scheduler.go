/*
 * @module service/scheduler/task_scheduler
 * @description 任务调度器，负责定时任务管理、任务队列管理、任务分发和执行
 * @architecture 分层架构 - 任务调度层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 调度配置 -> 任务生成 -> 任务分发 -> 任务执行 -> 状态更新
 * @rules 确保调度任务的可靠性和时效性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/sync_engine, service/basic_library
 */

package scheduler

import (
	"context"
	"datahub-service/service/models"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// TaskScheduler 任务调度器
type TaskScheduler struct {
	db           *gorm.DB
	taskQueue    chan *ScheduleTask
	workerPool   chan struct{}
	tasks        map[string]*ScheduleTask
	taskMutex    sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	maxWorkers   int
	taskExecutor *TaskExecutor
	retryManager *RetryManager
}

// ScheduleTask 调度任务
type ScheduleTask struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // sync, quality_check, backup
	DataSourceID string                 `json:"data_source_id"`
	InterfaceID  string                 `json:"interface_id,omitempty"`
	Config       map[string]interface{} `json:"config"`
	CronExpr     string                 `json:"cron_expr"`
	NextRunTime  time.Time              `json:"next_run_time"`
	LastRunTime  *time.Time             `json:"last_run_time,omitempty"`
	Status       string                 `json:"status"` // enabled, disabled, running, error
	CreatedBy    string                 `json:"created_by"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TaskExecution 任务执行记录
type TaskExecution struct {
	ID         string                 `json:"id"`
	TaskID     string                 `json:"task_id"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Status     string                 `json:"status"` // running, success, failed, cancelled
	Result     map[string]interface{} `json:"result,omitempty"`
	ErrorMsg   string                 `json:"error_msg,omitempty"`
	Duration   time.Duration          `json:"duration"`
	RetryCount int                    `json:"retry_count"`
	ExecutedBy string                 `json:"executed_by"`
}

// NewTaskScheduler 创建任务调度器实例
func NewTaskScheduler(db *gorm.DB, maxWorkers int) *TaskScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &TaskScheduler{
		db:           db,
		taskQueue:    make(chan *ScheduleTask, 1000),
		workerPool:   make(chan struct{}, maxWorkers),
		tasks:        make(map[string]*ScheduleTask),
		ctx:          ctx,
		cancel:       cancel,
		maxWorkers:   maxWorkers,
		taskExecutor: NewTaskExecutor(db),
		retryManager: NewRetryManager(db),
	}

	// 启动工作池
	for i := 0; i < maxWorkers; i++ {
		go scheduler.worker()
	}

	return scheduler
}

// AddScheduleTask 添加调度任务
func (s *TaskScheduler) AddScheduleTask(task *ScheduleTask) error {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	task.Status = "enabled"
	task.UpdatedAt = time.Now()

	// 保存到内存
	s.tasks[task.ID] = task

	// 保存到数据库
	scheduleConfig := &models.ScheduleConfig{
		ID:           task.ID,
		DataSourceID: &task.DataSourceID,
		ScheduleType: "cron",
		ScheduleConfig: map[string]interface{}{
			"cron_expr": task.CronExpr,
			"task_type": task.Type,
			"config":    task.Config,
		},
		IsEnabled:   true,
		NextRunTime: &task.NextRunTime,
		CreatedBy:   task.CreatedBy,
	}

	return s.db.Create(scheduleConfig).Error
}

// worker 工作协程
func (s *TaskScheduler) worker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case task := <-s.taskQueue:
			s.executeTask(task)
		}
	}
}

// executeTask 执行任务
func (s *TaskScheduler) executeTask(task *ScheduleTask) {
	// 获取工作者槽位
	s.workerPool <- struct{}{}
	defer func() { <-s.workerPool }()

	execution := &TaskExecution{
		ID:         fmt.Sprintf("%s_%d", task.ID, time.Now().UnixNano()),
		TaskID:     task.ID,
		StartTime:  time.Now(),
		Status:     "running",
		ExecutedBy: "scheduler",
	}

	// 更新任务状态
	s.updateTaskStatus(task.ID, "running", &execution.StartTime)

	// 执行任务
	result, err := s.taskExecutor.Execute(s.ctx, task)
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = execution.EndTime.Sub(execution.StartTime)

	if err != nil {
		execution.Status = "failed"
		execution.ErrorMsg = err.Error()
	} else {
		execution.Status = "success"
		execution.Result = result
	}

	// 更新任务状态
	s.updateTaskStatus(task.ID, "enabled", execution.EndTime)

	// 记录执行历史
	s.recordExecution(execution)
}

// updateTaskStatus 更新任务状态
func (s *TaskScheduler) updateTaskStatus(taskID, status string, lastRunTime *time.Time) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.Status = status
		if lastRunTime != nil {
			task.LastRunTime = lastRunTime
		}
		task.UpdatedAt = time.Now()
	}
}

// recordExecution 记录执行历史
func (s *TaskScheduler) recordExecution(execution *TaskExecution) {
	// TODO: 实现执行历史记录功能
}

// GetScheduledTasks 获取所有调度任务
func (s *TaskScheduler) GetScheduledTasks() map[string]*ScheduleTask {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	result := make(map[string]*ScheduleTask)
	for k, v := range s.tasks {
		result[k] = v
	}
	return result
}

// Stop 停止调度器
func (s *TaskScheduler) Stop() {
	s.cancel()
}
