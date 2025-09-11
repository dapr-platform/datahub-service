/*
 * @module service/sync_engine/sync_engine
 * @description 数据同步核心引擎，提供统一的数据同步入口和任务管理
 * @architecture 分层架构 - 核心服务层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 同步任务创建 -> 任务分发 -> 处理器执行 -> 状态更新 -> 结果通知
 * @rules 确保数据同步的可靠性、一致性和可恢复性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/basic_library, service/scheduler
 */

package sync_engine

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/models"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncEngine 数据同步核心引擎
type SyncEngine struct {
	db                 *gorm.DB
	batchProcessor     *BatchProcessor
	realtimeProcessor  *RealtimeProcessor
	dataTransformer    *DataTransformer
	incrementalSync    *IncrementalSync
	datasourceManager  datasource.DataSourceManager
	runningTasks       map[string]*SyncTaskContext
	taskMutex          sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	eventNotifier      func(event *SyncEvent)
	maxConcurrentTasks int
	taskQueue          chan *SyncTaskRequest
	workerPool         chan struct{}
	syncTaskService    SyncTaskServiceInterface // 添加服务接口
}

// SyncTaskServiceInterface 定义同步任务服务接口
type SyncTaskServiceInterface interface {
	CreateSyncTaskExecution(ctx context.Context, taskID, executionType string) (*models.SyncTaskExecution, error)
	UpdateSyncTaskExecution(ctx context.Context, executionID string, status string, result map[string]interface{}, errorMessage string) error
	UpdateTaskNextRunTime(ctx context.Context, taskID string) error
}

// 使用models包中定义的类型
type SyncTaskContext = models.SyncTaskContext
type TaskStatus = models.TaskStatus
type SyncType = models.SyncType
type SyncProcessor = models.SyncProcessor
type SyncProgress = models.SyncProgress
type SyncResult = models.SyncResult
type SyncEvent = models.SyncEvent
type SyncTaskRequest = models.SyncTaskRequest
type ProgressEstimate = models.ProgressEstimate

// 重新导入常量
const (
	TaskStatusPending   = models.TaskStatusPending
	TaskStatusRunning   = models.TaskStatusRunning
	TaskStatusSuccess   = models.TaskStatusSuccess
	TaskStatusFailed    = models.TaskStatusFailed
	TaskStatusCancelled = models.TaskStatusCancelled
)

const (
	SyncTypeFull        = models.SyncTypeFull
	SyncTypeIncremental = models.SyncTypeIncremental
	SyncTypeRealtime    = models.SyncTypeRealtime
)

// NewSyncEngine 创建同步引擎实例
func NewSyncEngine(db *gorm.DB, maxConcurrentTasks int, syncTaskService SyncTaskServiceInterface) *SyncEngine {
	ctx, cancel := context.WithCancel(context.Background())

	// 使用全局数据源注册表，确保内置类型已注册
	registry := datasource.GetGlobalRegistry()
	datasourceManager := registry.GetManager()

	engine := &SyncEngine{
		db:                 db,
		datasourceManager:  datasourceManager,
		runningTasks:       make(map[string]*SyncTaskContext),
		ctx:                ctx,
		cancel:             cancel,
		maxConcurrentTasks: maxConcurrentTasks,
		taskQueue:          make(chan *SyncTaskRequest, 1000),
		workerPool:         make(chan struct{}, maxConcurrentTasks),
		syncTaskService:    syncTaskService,
	}

	// 初始化各个处理器，传入数据源管理器
	engine.batchProcessor = NewBatchProcessor(db, datasourceManager)
	engine.realtimeProcessor = NewRealtimeProcessor(db, datasourceManager)
	engine.dataTransformer = NewDataTransformer(db)
	engine.incrementalSync = NewIncrementalSync(db, datasourceManager)

	// 启动任务处理器
	go engine.processTaskQueue()

	return engine
}

// SubmitSyncTask 提交同步任务
func (e *SyncEngine) SubmitSyncTask(request *SyncTaskRequest) (*models.SyncTask, error) {
	var task *models.SyncTask

	// 检查是否为手动执行的已存在任务
	if request.TaskID != "" && request.ScheduledBy == "manual" {
		// 手动执行：查找已存在的任务并更新其状态
		existingTask := &models.SyncTask{}
		if err := e.db.First(existingTask, "id = ?", request.TaskID).Error; err != nil {
			return nil, fmt.Errorf("查找任务失败: %w", err)
		}

		fmt.Printf("[DEBUG] 手动执行已存在任务: %s, 当前状态: %s\n", existingTask.ID, existingTask.Status)
		task = existingTask

		// 更新任务为运行状态
		updates := map[string]interface{}{
			"status":        string(TaskStatusRunning),
			"start_time":    time.Now(),
			"updated_at":    time.Now(),
			"error_message": "", // 清空之前的错误信息
		}
		if err := e.db.Model(task).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新任务状态失败: %w", err)
		}

		fmt.Printf("[DEBUG] 任务状态已更新为运行中: %s\n", task.ID)
	} else {
		// 定时任务或新任务：创建新的任务记录
		task = &models.SyncTask{
			ID:           uuid.New().String(),
			LibraryType:  request.LibraryType,
			LibraryID:    request.LibraryID,
			DataSourceID: request.DataSourceID,
			TaskType:     string(request.SyncType),
			Status:       string(TaskStatusPending),
			Config:       request.Config,
			CreatedBy:    request.ScheduledBy,
		}

		// 保存到数据库
		if err := e.db.Create(task).Error; err != nil {
			return nil, fmt.Errorf("创建同步任务失败: %w", err)
		}

		fmt.Printf("[DEBUG] 创建新任务: %s, 调度方式: %s\n", task.ID, request.ScheduledBy)
	}

	// 设置任务ID到请求中，以便后续处理
	request.TaskID = task.ID

	// 加入任务队列
	select {
	case e.taskQueue <- request:
		fmt.Printf("[DEBUG] 任务已加入执行队列: %s\n", task.ID)
		return task, nil
	default:
		// 队列满了，更新任务状态为失败
		e.updateTaskStatus(task.ID, TaskStatusFailed, "任务队列已满")
		return nil, errors.New("任务队列已满，请稍后重试")
	}
}

// processTaskQueue 处理任务队列
func (e *SyncEngine) processTaskQueue() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case request := <-e.taskQueue:
			// 获取工作者槽位
			e.workerPool <- struct{}{}

			// 启动协程处理任务
			go func(req *SyncTaskRequest) {
				defer func() { <-e.workerPool }()
				e.executeTask(req)
			}(request)
		}
	}
}

// executeTask 执行同步任务
func (e *SyncEngine) executeTask(request *SyncTaskRequest) {
	fmt.Printf("[DEBUG] SyncEngine.executeTask - 开始执行任务: %s\n", request.TaskID)

	// 获取任务信息
	var task models.SyncTask
	var err error

	if request.TaskID != "" {
		// 直接通过任务ID查找
		err = e.db.First(&task, "id = ?", request.TaskID).Error
		fmt.Printf("[DEBUG] SyncEngine.executeTask - 通过任务ID查找任务: %s\n", request.TaskID)
	} else {
		// 通过其他条件查找（用于定时任务）
		query := e.db.Where("library_type = ? AND library_id = ? AND data_source_id = ? AND task_type = ?",
			request.LibraryType, request.LibraryID, request.DataSourceID, request.SyncType).
			Order("created_at DESC")
		err = query.First(&task).Error
		fmt.Printf("[DEBUG] SyncEngine.executeTask - 通过条件查找任务: %s/%s/%s/%s\n",
			request.LibraryType, request.LibraryID, request.DataSourceID, request.SyncType)
	}

	if err != nil {
		fmt.Printf("[ERROR] SyncEngine.executeTask - 查找任务失败: %v\n", err)
		e.notifyEvent(&SyncEvent{
			EventType: "error",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error":          "任务不存在",
				"task_id":        request.TaskID,
				"library_type":   request.LibraryType,
				"library_id":     request.LibraryID,
				"data_source_id": request.DataSourceID,
				"task_type":      request.SyncType,
			},
		})
		return
	}

	fmt.Printf("[DEBUG] SyncEngine.executeTask - 找到任务: %s, 状态: %s, 类型: %s\n",
		task.ID, task.Status, task.TaskType)

	// 创建执行记录
	executionType := "manual"
	if request.IsScheduled {
		executionType = "scheduled"
	}

	execution, err := e.syncTaskService.CreateSyncTaskExecution(e.ctx, task.ID, executionType)
	if err != nil {
		fmt.Printf("[ERROR] SyncEngine.executeTask - 创建执行记录失败: %v\n", err)
		return
	}

	fmt.Printf("[DEBUG] SyncEngine.executeTask - 创建执行记录: %s\n", execution.ID)

	// 创建任务上下文
	taskCtx, cancel := context.WithCancel(e.ctx)
	defer cancel()

	taskContext := &SyncTaskContext{
		Task:      &task,
		Execution: execution,
		Context:   taskCtx,
		Cancel:    cancel,
		StartTime: time.Now(),
		Status:    TaskStatusRunning,
		Progress:  &SyncProgress{},
	}

	// 注册运行中的任务
	e.taskMutex.Lock()
	e.runningTasks[task.ID] = taskContext
	e.taskMutex.Unlock()

	// 清理任务上下文
	defer func() {
		e.taskMutex.Lock()
		delete(e.runningTasks, task.ID)
		e.taskMutex.Unlock()
	}()

	// 根据同步类型选择处理器
	fmt.Printf("[DEBUG] SyncEngine.executeTask - 选择处理器，同步类型: %s\n", request.SyncType)
	processor, err := e.selectProcessor(request.SyncType)
	if err != nil {
		fmt.Printf("[ERROR] SyncEngine.executeTask - 选择处理器失败: %v\n", err)
		e.handleTaskError(&task, execution, err)
		return
	}

	taskContext.Processor = processor
	fmt.Printf("[DEBUG] SyncEngine.executeTask - 处理器已选择: %s\n", processor.GetProcessorType())

	// 更新任务状态为运行中
	fmt.Printf("[DEBUG] SyncEngine.executeTask - 更新任务状态为运行中: %s\n", task.ID)
	e.updateTaskStatus(task.ID, TaskStatusRunning, "")

	// 发送任务开始事件
	e.notifyEvent(&SyncEvent{
		TaskID:    task.ID,
		EventType: "start",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task_type":      task.TaskType,
			"library_type":   task.LibraryType,
			"library_id":     task.LibraryID,
			"data_source_id": task.DataSourceID,
		},
	})

	fmt.Printf("[DEBUG] SyncEngine.executeTask - 开始执行同步处理: %s\n", task.ID)
	// 执行同步处理
	result, err := processor.Process(taskCtx, &task, taskContext.Progress)
	if err != nil {
		fmt.Printf("[ERROR] SyncEngine.executeTask - 同步处理失败: %v\n", err)
		e.handleTaskError(&task, execution, err)
		return
	}

	// 处理成功
	fmt.Printf("[DEBUG] SyncEngine.executeTask - 同步处理成功: %s, 处理行数: %d\n", task.ID, result.ProcessedRows)
	taskContext.Result = result
	e.handleTaskSuccess(&task, execution, result)

	// 更新任务的下次运行时间（针对调度任务）
	if request.IsScheduled {
		err = e.syncTaskService.UpdateTaskNextRunTime(e.ctx, task.ID)
		if err != nil {
			fmt.Printf("[ERROR] SyncEngine.executeTask - 更新下次运行时间失败: %v\n", err)
		}
	}

	// 执行回调
	if request.Callback != nil {
		request.Callback(result)
	}
}

// selectProcessor 选择同步处理器
func (e *SyncEngine) selectProcessor(syncType SyncType) (SyncProcessor, error) {
	switch syncType {
	case SyncTypeFull:
		return e.batchProcessor, nil
	case SyncTypeRealtime:
		return e.realtimeProcessor, nil
	case SyncTypeIncremental:
		return e.incrementalSync, nil
	default:
		return nil, fmt.Errorf("不支持的同步类型: %s", syncType)
	}
}

// GetTaskStatus 获取任务状态
func (e *SyncEngine) GetTaskStatus(taskID string) (*SyncTaskContext, error) {
	e.taskMutex.RLock()
	defer e.taskMutex.RUnlock()

	if context, exists := e.runningTasks[taskID]; exists {
		return context, nil
	}

	// 从数据库查询已完成的任务
	var task models.SyncTask
	if err := e.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, err
	}

	return &SyncTaskContext{
		Task:   &task,
		Status: TaskStatus(task.Status),
	}, nil
}

// CancelTask 取消任务
func (e *SyncEngine) CancelTask(taskID string) error {
	e.taskMutex.Lock()
	defer e.taskMutex.Unlock()

	if context, exists := e.runningTasks[taskID]; exists {
		context.Cancel()
		context.Status = TaskStatusCancelled
		e.updateTaskStatus(taskID, TaskStatusCancelled, "任务被用户取消")
		return nil
	}

	return errors.New("任务不存在或已完成")
}

// GetRunningTasks 获取运行中的任务列表
func (e *SyncEngine) GetRunningTasks() map[string]*SyncTaskContext {
	e.taskMutex.RLock()
	defer e.taskMutex.RUnlock()

	result := make(map[string]*SyncTaskContext)
	for k, v := range e.runningTasks {
		result[k] = v
	}
	return result
}

// SetEventNotifier 设置事件通知器
func (e *SyncEngine) SetEventNotifier(notifier func(event *SyncEvent)) {
	e.eventNotifier = notifier
}

// updateTaskStatus 更新任务状态
func (e *SyncEngine) updateTaskStatus(taskID string, status TaskStatus, errorMessage string) {
	updates := map[string]interface{}{
		"status":     string(status),
		"updated_at": time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if status == TaskStatusSuccess || status == TaskStatusFailed || status == TaskStatusCancelled {
		updates["end_time"] = time.Now()
	}

	e.db.Model(&models.SyncTask{}).Where("id = ?", taskID).Updates(updates)
}

// handleTaskError 处理任务错误
func (e *SyncEngine) handleTaskError(task *models.SyncTask, execution *models.SyncTaskExecution, err error) {
	e.updateTaskStatus(task.ID, TaskStatusFailed, err.Error())

	// 更新执行记录
	if execution != nil {
		resultMap := map[string]interface{}{
			"error": err.Error(),
		}
		updateErr := e.syncTaskService.UpdateSyncTaskExecution(e.ctx, execution.ID, "failed", resultMap, err.Error())
		if updateErr != nil {
			fmt.Printf("[ERROR] SyncEngine.handleTaskError - 更新执行记录失败: %v\n", updateErr)
		}
	}

	e.notifyEvent(&SyncEvent{
		TaskID:    task.ID,
		EventType: "error",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"error": err.Error(),
		},
	})
}

// handleTaskSuccess 处理任务成功
func (e *SyncEngine) handleTaskSuccess(task *models.SyncTask, execution *models.SyncTaskExecution, result *SyncResult) {
	// 更新任务结果
	updates := map[string]interface{}{
		"status":         string(TaskStatusSuccess),
		"end_time":       time.Now(),
		"processed_rows": result.ProcessedRows,
		"result":         result.Statistics,
		"updated_at":     time.Now(),
		"last_run_time":  time.Now(),
	}

	e.db.Model(&models.SyncTask{}).Where("id = ?", task.ID).Updates(updates)

	// 更新执行记录
	if execution != nil {
		resultMap := map[string]interface{}{
			"processed_rows": result.ProcessedRows,
			"success_rows":   result.SuccessRows,
			"error_rows":     result.ErrorRows,
			"duration":       result.Duration.String(),
			"statistics":     result.Statistics,
		}
		updateErr := e.syncTaskService.UpdateSyncTaskExecution(e.ctx, execution.ID, "success", resultMap, "")
		if updateErr != nil {
			fmt.Printf("[ERROR] SyncEngine.handleTaskSuccess - 更新执行记录失败: %v\n", updateErr)
		}
	}

	e.notifyEvent(&SyncEvent{
		TaskID:    task.ID,
		EventType: "complete",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"processed_rows": result.ProcessedRows,
			"duration":       result.Duration.String(),
		},
	})
}

// notifyEvent 发送事件通知
func (e *SyncEngine) notifyEvent(event *SyncEvent) {
	if e.eventNotifier != nil {
		go e.eventNotifier(event)
	}
}

// Stop 停止同步引擎
func (e *SyncEngine) Stop() {
	e.cancel()

	// 等待所有任务完成或超时
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// 强制取消所有任务
			e.taskMutex.Lock()
			for _, taskCtx := range e.runningTasks {
				taskCtx.Cancel()
			}
			e.taskMutex.Unlock()
			return
		case <-ticker.C:
			e.taskMutex.RLock()
			count := len(e.runningTasks)
			e.taskMutex.RUnlock()

			if count == 0 {
				return
			}
		}
	}
}

// GetStatistics 获取同步统计信息
func (e *SyncEngine) GetStatistics() map[string]interface{} {
	e.taskMutex.RLock()
	runningCount := len(e.runningTasks)
	e.taskMutex.RUnlock()

	var stats struct {
		TotalTasks   int64 `json:"total_tasks"`
		SuccessTasks int64 `json:"success_tasks"`
		FailedTasks  int64 `json:"failed_tasks"`
		PendingTasks int64 `json:"pending_tasks"`
	}

	e.db.Model(&models.SyncTask{}).Count(&stats.TotalTasks)
	e.db.Model(&models.SyncTask{}).Where("status = ?", string(TaskStatusSuccess)).Count(&stats.SuccessTasks)
	e.db.Model(&models.SyncTask{}).Where("status = ?", string(TaskStatusFailed)).Count(&stats.FailedTasks)
	e.db.Model(&models.SyncTask{}).Where("status = ?", string(TaskStatusPending)).Count(&stats.PendingTasks)

	return map[string]interface{}{
		"running_tasks":  runningCount,
		"total_tasks":    stats.TotalTasks,
		"success_tasks":  stats.SuccessTasks,
		"failed_tasks":   stats.FailedTasks,
		"pending_tasks":  stats.PendingTasks,
		"queue_length":   len(e.taskQueue),
		"max_concurrent": e.maxConcurrentTasks,
	}
}
