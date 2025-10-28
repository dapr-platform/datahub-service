/*
 * @module service/basic_library/sync_task_service
 * @description 基础库同步任务服务，专门处理基础库数据同步任务管理和调度
 * @architecture 分层架构 - 服务层，集成调度功能
 * @documentReference ai_docs/refactor_sync_task.md
 * @stateFlow 服务初始化 -> 任务CRUD操作 -> 任务执行管理 -> 调度器管理
 * @rules 专门支持基础库同步任务，统一使用interface_executor执行
 * @dependencies gorm.io/gorm, service/models, service/meta, service/interface_executor, github.com/robfig/cron/v3
 * @refs api/controllers/sync_task_controller.go, service/interface_executor
 */

package basic_library

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/distributed_lock"
	"datahub-service/service/interface_executor"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// BasicLibraryHandler 基础库处理器
type BasicLibraryHandler struct {
	db      *gorm.DB
	service *Service
}

// NewBasicLibraryHandler 创建基础库处理器
func NewBasicLibraryHandler(db *gorm.DB, service *Service) *BasicLibraryHandler {
	return &BasicLibraryHandler{
		db:      db,
		service: service,
	}
}

// ValidateLibrary 验证基础库是否存在
func (h *BasicLibraryHandler) ValidateLibrary(libraryID string) error {
	_, err := h.service.GetBasicLibrary(libraryID)
	if err != nil {
		return fmt.Errorf("基础库不存在: %w", err)
	}
	return nil
}

// ValidateDataSource 验证数据源是否属于基础库
func (h *BasicLibraryHandler) ValidateDataSource(libraryID, dataSourceID string) error {
	// 直接查询数据库验证数据源是否属于该基础库
	var dataSource models.DataSource
	err := h.db.Where("id = ? AND library_id = ?", dataSourceID, libraryID).First(&dataSource).Error
	if err != nil {
		return fmt.Errorf("数据源 %s 不属于基础库 %s", dataSourceID, libraryID)
	}
	return nil
}

// ValidateInterface 验证接口是否属于基础库
func (h *BasicLibraryHandler) ValidateInterface(libraryID, interfaceID string) error {
	// 直接查询数据库验证接口是否属于该基础库
	var dataInterface models.DataInterface
	err := h.db.Where("id = ? AND library_id = ?", interfaceID, libraryID).First(&dataInterface).Error
	if err != nil {
		return fmt.Errorf("接口 %s 不属于基础库 %s", interfaceID, libraryID)
	}
	return nil
}

// GetLibraryInfo 获取基础库信息
func (h *BasicLibraryHandler) GetLibraryInfo(libraryID string) (interface{}, error) {
	return h.service.GetBasicLibrary(libraryID)
}

// PrepareTaskConfig 准备基础库任务配置
func (h *BasicLibraryHandler) PrepareTaskConfig(libraryID string, config map[string]interface{}) (map[string]interface{}, error) {
	// 为基础库添加特定的配置项
	if config == nil {
		config = make(map[string]interface{})
	}

	config["library_type"] = meta.LibraryTypeBasic
	config["library_id"] = libraryID

	// 添加基础库特定的默认配置
	if _, exists := config["batch_size"]; !exists {
		config["batch_size"] = 1000
	}
	if _, exists := config["timeout"]; !exists {
		config["timeout"] = "30m"
	}

	return config, nil
}

// GetLibraryDataSources 获取基础库的数据源列表
func (h *BasicLibraryHandler) GetLibraryDataSources(libraryID string) ([]models.DataSource, error) {
	var dataSources []models.DataSource
	err := h.db.Where("library_id = ?", libraryID).Find(&dataSources).Error
	return dataSources, err
}

// GetLibraryInterfaces 获取基础库的接口列表
func (h *BasicLibraryHandler) GetLibraryInterfaces(libraryID string) ([]models.DataInterface, error) {
	var interfaces []models.DataInterface
	err := h.db.Where("library_id = ?", libraryID).Find(&interfaces).Error
	return interfaces, err
}

// SyncTaskService 基础库同步任务服务（集成调度功能）
type SyncTaskService struct {
	db                *gorm.DB
	handler           *BasicLibraryHandler
	interfaceExecutor *interface_executor.InterfaceExecutor
	datasourceManager datasource.DataSourceManager
	// 调度器相关字段
	cron             *cron.Cron
	intervalTicker   *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
	schedulerStarted bool
	// 分布式锁
	distributedLock distributed_lock.DistributedLock
}

// NewSyncTaskService 创建基础库同步任务服务
func NewSyncTaskService(db *gorm.DB, basicLibService *Service) *SyncTaskService {
	// 初始化数据源管理器
	datasourceManager := datasource.GetGlobalRegistry().GetManager()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建带时区的cron调度器
	c := cron.New(cron.WithSeconds())

	service := &SyncTaskService{
		db:                db,
		handler:           NewBasicLibraryHandler(db, basicLibService),
		interfaceExecutor: interface_executor.NewInterfaceExecutor(db, datasourceManager),
		datasourceManager: datasourceManager,
		cron:              c,
		ctx:               ctx,
		cancel:            cancel,
		schedulerStarted:  false,
	}

	return service
}

// CreateSyncTask 创建基础库同步任务
func (s *SyncTaskService) CreateSyncTask(ctx context.Context, req *CreateSyncTaskRequest) (*models.SyncTask, error) {
	// 验证库存在
	if err := s.handler.ValidateLibrary(req.LibraryID); err != nil {
		return nil, err
	}

	// 验证数据源
	if err := s.handler.ValidateDataSource(req.LibraryID, req.DataSourceID); err != nil {
		return nil, err
	}

	// 验证所有接口
	if len(req.InterfaceIDs) == 0 {
		return nil, fmt.Errorf("必须提供至少一个接口ID")
	}

	for _, interfaceID := range req.InterfaceIDs {
		if err := s.handler.ValidateInterface(req.LibraryID, interfaceID); err != nil {
			return nil, fmt.Errorf("验证接口 %s 失败: %w", interfaceID, err)
		}
	}

	// 准备任务配置
	config, err := s.handler.PrepareTaskConfig(req.LibraryID, req.Config)
	if err != nil {
		return nil, fmt.Errorf("准备任务配置失败: %w", err)
	}

	// 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建任务
	task := &models.SyncTask{
		LibraryType:     meta.LibraryTypeBasic, // 固定为基础库类型
		LibraryID:       req.LibraryID,
		DataSourceID:    req.DataSourceID,
		TaskType:        req.TaskType,
		TriggerType:     req.TriggerType,
		CronExpression:  req.CronExpression,
		IntervalSeconds: req.IntervalSeconds,
		ScheduledTime:   req.ScheduledTime,
		Status:          meta.SyncTaskStatusDraft,     // 默认状态为草稿
		ExecutionStatus: meta.SyncExecutionStatusIdle, // 默认执行状态为空闲
		Config:          config,
		CreatedBy:       req.CreatedBy,
	}

	// 根据触发类型设置下次执行时间
	if err := s.calculateNextRunTime(task); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("计算下次执行时间失败: %w", err)
	}

	if err := tx.Create(task).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建同步任务失败: %w", err)
	}

	// 创建接口配置映射
	interfaceConfigMap := make(map[string]map[string]interface{})
	for _, interfaceConfig := range req.InterfaceConfigs {
		interfaceConfigMap[interfaceConfig.InterfaceID] = interfaceConfig.Config
	}

	// 创建任务接口关联记录
	for _, interfaceID := range req.InterfaceIDs {
		taskInterface := &models.SyncTaskInterface{
			TaskID:      task.ID,
			InterfaceID: interfaceID,
			Status:      meta.SyncExecutionStatusIdle, // 初始状态为空闲
			Config:      interfaceConfigMap[interfaceID],
		}

		if err := tx.Create(taskInterface).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建任务接口关联失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 重新加载任务以包含关联数据
	if err := s.db.Preload("TaskInterfaces").Preload("DataSource").First(task, "id = ?", task.ID).Error; err != nil {
		return nil, fmt.Errorf("重新加载任务失败: %w", err)
	}

	// 注意：草稿状态的任务不会自动添加到调度器
	// 需要手动激活任务后才会加入调度
	slog.Info("任务已创建", "task_id", task.ID, "status", task.Status, "trigger_type", task.TriggerType)

	return task, nil
}

// GetSyncTaskByID 根据ID获取基础库同步任务
func (s *SyncTaskService) GetSyncTaskByID(ctx context.Context, taskID string) (*models.SyncTask, error) {
	var task models.SyncTask
	if err := s.db.Preload("DataSource").
		Preload("TaskInterfaces").
		Preload("TaskInterfaces.DataInterface").
		Preload("DataInterfaces").
		First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取同步任务失败: %w", err)
	}

	// 加载基础库信息
	if err := s.loadLibraryInfo(&task); err != nil {
		return nil, fmt.Errorf("加载库信息失败: %w", err)
	}

	return &task, nil
}

// loadLibraryInfo 加载基础库信息
func (s *SyncTaskService) loadLibraryInfo(task *models.SyncTask) error {
	libraryInfo, err := s.handler.GetLibraryInfo(task.LibraryID)
	if err != nil {
		return err
	}

	// 设置基础库信息
	if basicLib, ok := libraryInfo.(*models.BasicLibrary); ok {
		task.BasicLibrary = basicLib
	}

	return nil
}

// SyncTaskInterfaceConfig 接口级别的配置
type SyncTaskInterfaceConfig struct {
	InterfaceID string                 `json:"interface_id"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// CreateSyncTaskRequest 创建基础库同步任务请求
type CreateSyncTaskRequest struct {
	LibraryType      string                    `json:"library_type" binding:"required"`
	LibraryID        string                    `json:"library_id" binding:"required"`
	DataSourceID     string                    `json:"data_source_id" binding:"required"`
	InterfaceIDs     []string                  `json:"interface_ids" binding:"required,min=1"`
	InterfaceConfigs []SyncTaskInterfaceConfig `json:"interface_configs,omitempty"`
	TaskType         string                    `json:"task_type" binding:"required"`
	TriggerType      string                    `json:"trigger_type" binding:"required"`
	CronExpression   string                    `json:"cron_expression,omitempty"`
	IntervalSeconds  int                       `json:"interval_seconds,omitempty"`
	ScheduledTime    *time.Time                `json:"scheduled_time,omitempty"`
	Config           map[string]interface{}    `json:"config,omitempty"`
	CreatedBy        string                    `json:"created_by"`
}

// UpdateSyncTaskRequest 更新基础库同步任务请求
type UpdateSyncTaskRequest struct {
	Status           string                    `json:"status,omitempty"` // draft, active, paused
	TriggerType      string                    `json:"trigger_type,omitempty"`
	CronExpression   string                    `json:"cron_expression,omitempty"`
	IntervalSeconds  int                       `json:"interval_seconds,omitempty"`
	Config           map[string]interface{}    `json:"config,omitempty"`
	InterfaceIDs     []string                  `json:"interface_ids,omitempty"`
	InterfaceConfigs []SyncTaskInterfaceConfig `json:"interface_configs,omitempty"`
	UpdatedBy        string                    `json:"updated_by"`
	TaskType         string                    `json:"task_type,omitempty"`
	ScheduledTime    *time.Time                `json:"scheduled_time,omitempty"`
}

// GetSyncTaskListRequest 获取基础库同步任务列表请求
type GetSyncTaskListRequest struct {
	Page         int    `json:"page"`
	Size         int    `json:"size"`
	LibraryType  string `json:"library_type,omitempty"`
	LibraryID    string `json:"library_id,omitempty"`
	DataSourceID string `json:"data_source_id,omitempty"`
	Status       string `json:"status,omitempty"`
	TaskType     string `json:"task_type,omitempty"`
}

// SyncTaskListResponse 基础库同步任务列表响应
type SyncTaskListResponse struct {
	Tasks      []models.SyncTask `json:"tasks"`
	Pagination PaginationInfo    `json:"pagination"`
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

// SyncTaskStatusResponse 基础库同步任务状态响应
type SyncTaskStatusResponse struct {
	Task      *models.SyncTask     `json:"task"`
	StartTime *time.Time           `json:"start_time,omitempty"`
	Status    string               `json:"status"`
	Progress  *models.SyncProgress `json:"progress,omitempty"`
	Error     string               `json:"error,omitempty"`
	Result    *models.SyncResult   `json:"result,omitempty"`
	Processor string               `json:"processor,omitempty"`
}

// GetSyncTaskExecutionListRequest 获取基础库同步任务执行记录列表请求
type GetSyncTaskExecutionListRequest struct {
	Page          int    `json:"page"`
	Size          int    `json:"size"`
	TaskID        string `json:"task_id,omitempty"`
	Status        string `json:"status,omitempty"`
	ExecutionType string `json:"execution_type,omitempty"`
}

// SyncTaskExecutionListResponse 基础库同步任务执行记录列表响应
type SyncTaskExecutionListResponse struct {
	Executions []models.SyncTaskExecution `json:"executions"`
	Pagination PaginationInfo             `json:"pagination"`
}

// BatchDeleteResponse 批量删除响应
type BatchDeleteResponse struct {
	DeletedCount int      `json:"deleted_count"`
	FailedIDs    []string `json:"failed_ids,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// SyncTaskStatistics 基础库同步任务统计信息
type SyncTaskStatistics struct {
	TotalTasks     int64   `json:"total_tasks"`
	PendingTasks   int64   `json:"pending_tasks"`
	RunningTasks   int64   `json:"running_tasks"`
	SuccessTasks   int64   `json:"success_tasks"`
	FailedTasks    int64   `json:"failed_tasks"`
	CancelledTasks int64   `json:"cancelled_tasks"`
	SuccessRate    float64 `json:"success_rate"`
}

// GetSyncTaskList 获取基础库同步任务列表
func (s *SyncTaskService) GetSyncTaskList(ctx context.Context, req *GetSyncTaskListRequest) (*SyncTaskListResponse, error) {
	query := s.db.Model(&models.SyncTask{}).Where("library_type = ?", meta.LibraryTypeBasic)

	// 应用过滤条件
	if req.LibraryID != "" {
		query = query.Where("library_id = ?", req.LibraryID)
	}
	if req.DataSourceID != "" {
		query = query.Where("data_source_id = ?", req.DataSourceID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.TaskType != "" {
		query = query.Where("task_type = ?", req.TaskType)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("获取任务总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	var tasks []models.SyncTask
	if err := query.Preload("DataSource").
		Preload("TaskInterfaces").
		Preload("TaskInterfaces.DataInterface").
		Preload("DataInterfaces").
		Order("created_at DESC").
		Offset(offset).Limit(req.Size).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("查询任务列表失败: %w", err)
	}

	// 加载库信息
	for i := range tasks {
		if err := s.loadLibraryInfo(&tasks[i]); err != nil {
			// 记录错误但不阻塞
			slog.Error("加载库信息失败", "error", err)
		}
	}

	// 计算总页数
	totalPages := (total + int64(req.Size) - 1) / int64(req.Size)

	return &SyncTaskListResponse{
		Tasks: tasks,
		Pagination: PaginationInfo{
			Page:       req.Page,
			Size:       req.Size,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateSyncTask 更新基础库同步任务
func (s *SyncTaskService) UpdateSyncTask(ctx context.Context, taskID string, req *UpdateSyncTaskRequest) (*models.SyncTask, error) {
	// 获取任务
	var task models.SyncTask
	if err := s.db.Preload("TaskInterfaces").First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}

	// 记录原始状态
	oldStatus := task.Status

	// 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 准备更新数据
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	// 记录是否有状态变化
	var statusChanged bool
	var newStatus string

	if req.Status != "" && req.Status != oldStatus {
		// 验证状态转换是否合法
		if !meta.CanTransitionStatus(oldStatus, req.Status) {
			tx.Rollback()
			return nil, fmt.Errorf("不允许从 %s 状态转换到 %s 状态", oldStatus, req.Status)
		}
		updates["status"] = req.Status
		statusChanged = true
		newStatus = req.Status
	}

	if req.TriggerType != "" {
		updates["trigger_type"] = req.TriggerType
	}
	if req.CronExpression != "" {
		updates["cron_expression"] = req.CronExpression
	}
	if req.IntervalSeconds > 0 {
		updates["interval_seconds"] = req.IntervalSeconds
	}
	if req.Config != nil {
		updates["config"] = req.Config
	}
	if req.UpdatedBy != "" {
		updates["updated_by"] = req.UpdatedBy
	}
	if req.TaskType != "" {
		updates["task_type"] = req.TaskType
	}
	if req.ScheduledTime != nil {
		updates["scheduled_time"] = req.ScheduledTime
	}

	// 更新任务基本信息
	if err := tx.Model(&task).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新任务失败: %w", err)
	}

	// 如果修改了调度配置，重新计算下次执行时间
	scheduleChanged := req.TriggerType != "" || req.CronExpression != "" || req.IntervalSeconds > 0
	if scheduleChanged {
		// 重新加载任务以获取最新数据
		if err := tx.First(&task, "id = ?", taskID).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("重新加载任务失败: %w", err)
		}

		// 重新计算下次执行时间
		if err := s.calculateNextRunTime(&task); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("计算下次执行时间失败: %w", err)
		}

		// 更新下次执行时间
		if err := tx.Model(&task).Updates(map[string]interface{}{
			"next_run_time": task.NextRunTime,
		}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("更新下次执行时间失败: %w", err)
		}
	}

	// 更新接口配置（如果提供）
	if len(req.InterfaceIDs) > 0 {
		// 验证所有新接口
		for _, interfaceID := range req.InterfaceIDs {
			if err := s.handler.ValidateInterface(task.LibraryID, interfaceID); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("验证接口 %s 失败: %w", interfaceID, err)
			}
		}

		// 删除现有的接口关联
		if err := tx.Where("task_id = ?", taskID).Delete(&models.SyncTaskInterface{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("删除旧接口关联失败: %w", err)
		}

		// 创建接口配置映射
		interfaceConfigMap := make(map[string]map[string]interface{})
		for _, interfaceConfig := range req.InterfaceConfigs {
			interfaceConfigMap[interfaceConfig.InterfaceID] = interfaceConfig.Config
		}

		// 创建新的接口关联
		for _, interfaceID := range req.InterfaceIDs {
			taskInterface := &models.SyncTaskInterface{
				TaskID:      taskID,
				InterfaceID: interfaceID,
				Status:      meta.SyncExecutionStatusIdle, // 初始状态为空闲
				Config:      interfaceConfigMap[interfaceID],
			}

			if err := tx.Create(taskInterface).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("创建新接口关联失败: %w", err)
			}
		}
	} else if len(req.InterfaceConfigs) > 0 {
		// 仅更新接口级别配置，不改变接口列表
		for _, interfaceConfig := range req.InterfaceConfigs {
			if err := tx.Model(&models.SyncTaskInterface{}).
				Where("task_id = ? AND interface_id = ?", taskID, interfaceConfig.InterfaceID).
				Update("config", interfaceConfig.Config).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("更新接口配置失败: %w", err)
			}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 重新获取更新后的任务
	if err := s.db.Preload("DataSource").
		Preload("TaskInterfaces").
		Preload("TaskInterfaces.DataInterface").
		Preload("DataInterfaces").
		First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取更新后的任务失败: %w", err)
	}

	// 加载库信息
	if err := s.loadLibraryInfo(&task); err != nil {
		return nil, fmt.Errorf("加载库信息失败: %w", err)
	}

	// 处理状态变化（激活/暂停）
	if statusChanged {
		slog.Info("任务状态变化", "task_id", taskID, "old_status", oldStatus, "new_status", newStatus)

		switch newStatus {
		case meta.SyncTaskStatusActive:
			// 激活任务：添加到调度器
			if task.TriggerType == "cron" || task.TriggerType == "interval" || task.TriggerType == "once" {
				if err := s.AddScheduledTask(&task); err != nil {
					slog.Error("添加任务到调度器失败", "task_id", taskID, "error", err)
				} else {
					slog.Info("任务已激活并添加到调度器", "task_id", taskID)
				}
			}
		case meta.SyncTaskStatusPaused:
			// 暂停任务：从调度器移除
			if err := s.RemoveScheduledTask(taskID); err != nil {
				slog.Error("从调度器移除任务失败", "task_id", taskID, "error", err)
			} else {
				slog.Info("任务已暂停并从调度器移除", "task_id", taskID)
			}
		}
	} else if scheduleChanged {
		// 如果只是修改了调度配置（未改变状态），重新加载调度器
		slog.Info("任务调度配置已更新，准备重新加载调度器", "task_id", taskID)
		if err := s.ReloadScheduledTasks(); err != nil {
			slog.Error("重新加载调度器失败", "error", err)
		} else {
			slog.Info("调度配置已更新，调度器已重新加载", "task_id", taskID)
		}
	}

	return &task, nil
}

// DeleteSyncTask 删除基础库同步任务
func (s *SyncTaskService) DeleteSyncTask(ctx context.Context, taskID string) error {
	// 获取任务
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态是否允许删除
	if !contains(meta.GetDeletableTaskStatuses(), task.Status) {
		return fmt.Errorf("任务状态 %s 不允许删除", task.Status)
	}

	// 开启事务删除任务和相关记录
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除任务接口关联记录
		if err := tx.Where("task_id = ?", taskID).Delete(&models.SyncTaskInterface{}).Error; err != nil {
			return fmt.Errorf("删除任务接口关联记录失败: %w", err)
		}

		// 删除执行记录
		if err := tx.Where("task_id = ?", taskID).Delete(&models.SyncTaskExecution{}).Error; err != nil {
			return fmt.Errorf("删除执行记录失败: %w", err)
		}

		// 删除任务
		if err := tx.Delete(&task).Error; err != nil {
			return fmt.Errorf("删除任务失败: %w", err)
		}

		return nil
	})
}

// StartSyncTask 启动基础库同步任务
func (s *SyncTaskService) StartSyncTask(ctx context.Context, taskID string) error {
	slog.Debug("SyncTaskService.StartSyncTask - 开始启动任务", "value", taskID)

	// 获取任务详细信息
	var task models.SyncTask
	if err := s.db.Preload("TaskInterfaces").First(&task, "id = ?", taskID).Error; err != nil {
		slog.Error("SyncTaskService.StartSyncTask - 任务不存在", "value1", taskID, "value2", err)
		return fmt.Errorf("任务不存在: %w", err)
	}

	slog.Debug("SyncTaskService.StartSyncTask - 找到任务", "value1", task.ID, "value2", task.Status, "value3", task.TaskType)

	// 检查任务是否可以启动
	if !task.CanStart() {
		slog.Error("SyncTaskService.StartSyncTask - 任务状态不允许启动",
			"taskID", taskID,
			"status", task.Status,
			"executionStatus", task.ExecutionStatus)
		return fmt.Errorf("任务状态不允许启动: 任务ID=%s, 状态=%s, 执行状态=%s", taskID, task.Status, task.ExecutionStatus)
	}

	// 更新任务执行状态为运行中
	if err := s.db.Model(&task).Updates(map[string]interface{}{
		"execution_status": meta.SyncExecutionStatusRunning,
		"start_time":       time.Now(),
		"updated_at":       time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("更新任务执行状态失败: %w", err)
	}

	// 创建独立的context用于任务执行，避免HTTP请求context被取消影响任务执行
	taskCtx := context.Background()

	// 如果有指定接口，使用InterfaceExecutor执行
	if len(task.TaskInterfaces) > 0 {
		go s.executeTaskWithInterfaces(taskCtx, &task)
	} else {
		// 没有指定接口的情况，返回错误
		s.updateTaskExecutionStatus(task.ID, meta.SyncExecutionStatusFailed, "任务必须关联至少一个接口")
		return fmt.Errorf("任务必须关联至少一个接口")
	}

	return nil
}

// 注意：hasIncrementalConfig 和 getLastSyncTime 方法已被移除
// 现在统一使用 sync 执行类型，增量逻辑由 interface_executor 内部处理

// executeTaskWithInterfaces 使用InterfaceExecutor执行任务
func (s *SyncTaskService) executeTaskWithInterfaces(ctx context.Context, task *models.SyncTask) {
	slog.Debug("SyncTaskService.executeTaskWithInterfaces - 开始执行任务", "value", task.ID)

	// 创建执行记录
	execution, err := s.CreateSyncTaskExecution(ctx, task.ID, "interface_executor")
	if err != nil {
		slog.Error("创建执行记录失败", "error", err)
		s.updateTaskExecutionStatus(task.ID, meta.SyncExecutionStatusFailed, err.Error())
		return
	}

	var totalProcessed int64
	var hasError bool
	var errorMessages []string

	// 执行每个接口
	for _, taskInterface := range task.TaskInterfaces {
		slog.Debug("执行接口", "value", taskInterface.InterfaceID)

		// 使用统一的sync类型，内部根据接口的incremental_config自动判断全量/增量
		executeType := "sync" // 统一使用sync类型

		slog.Debug("接口使用统一的sync执行类型，将根据接口配置自动判断全量/增量同步", "interface_id", taskInterface.InterfaceID)

		// 准备执行请求
		executeRequest := &interface_executor.ExecuteRequest{
			InterfaceID:   taskInterface.InterfaceID,
			InterfaceType: "basic_library", // 固定为基础库
			ExecuteType:   executeType,     // 统一使用sync
			Parameters:    taskInterface.Config,
		}

		// 执行接口
		response, err := s.interfaceExecutor.Execute(ctx, executeRequest)
		if err != nil {
			hasError = true
			errorMsg := fmt.Sprintf("接口 %s 执行失败: %v", taskInterface.InterfaceID, err)
			errorMessages = append(errorMessages, errorMsg)
			slog.Error("Error occurred", "message", errorMsg)
			continue
		}

		if !response.Success {
			hasError = true
			errorMsg := fmt.Sprintf("接口 %s 执行失败: %s", taskInterface.InterfaceID, response.Error)
			errorMessages = append(errorMessages, errorMsg)
			slog.Error("Error occurred", "message", errorMsg)
			continue
		}

		totalProcessed += response.UpdatedRows
		slog.Debug("接口执行成功", "interface_id", taskInterface.InterfaceID, "updated_rows", response.UpdatedRows)
	}

	// 更新任务执行状态
	var finalExecutionStatus string
	var errorMessage string

	if hasError {
		if totalProcessed > 0 {
			finalExecutionStatus = meta.SyncExecutionStatusSuccess // 部分成功
			errorMessage = fmt.Sprintf("部分接口执行失败: %v", errorMessages)
		} else {
			finalExecutionStatus = meta.SyncExecutionStatusFailed
			errorMessage = fmt.Sprintf("所有接口执行失败: %v", errorMessages)
		}
	} else {
		finalExecutionStatus = meta.SyncExecutionStatusSuccess
	}

	// 更新任务
	updates := map[string]interface{}{
		"execution_status": finalExecutionStatus,
		"end_time":         time.Now(),
		"processed_rows":   totalProcessed,
		"progress":         100,
		"updated_at":       time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := s.db.Model(&models.SyncTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		slog.Error("更新任务执行状态失败", "error", err)
	} else {
		slog.Debug("任务执行状态更新成功", "status", finalExecutionStatus)
	}

	// 更新执行记录
	result := map[string]interface{}{
		"processed_rows":  totalProcessed,
		"interface_count": len(task.TaskInterfaces),
		"success_count":   len(task.TaskInterfaces) - len(errorMessages),
		"failed_count":    len(errorMessages),
	}

	if err := s.UpdateSyncTaskExecution(ctx, execution.ID, finalExecutionStatus, result, errorMessage); err != nil {
		slog.Error("更新执行记录失败", "error", err)
	} else {
		slog.Debug("执行记录更新成功", "status", finalExecutionStatus)
	}

	slog.Debug("任务执行完成", "task_id", task.ID, "execution_status", finalExecutionStatus, "processed_rows", totalProcessed)
}

// updateTaskExecutionStatus 更新任务执行状态的辅助方法
func (s *SyncTaskService) updateTaskExecutionStatus(taskID, executionStatus, errorMessage string) {
	updates := map[string]interface{}{
		"execution_status": executionStatus,
		"end_time":         time.Now(),
		"updated_at":       time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := s.db.Model(&models.SyncTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		slog.Error("更新任务执行状态失败", "error", err)
	} else {
		slog.Debug("任务执行状态更新成功", "executionStatus", executionStatus)
	}
}

// StopSyncTask 停止基础库同步任务
func (s *SyncTaskService) StopSyncTask(ctx context.Context, taskID string) error {
	// 获取任务
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务执行状态
	if task.ExecutionStatus != meta.SyncExecutionStatusRunning {
		return fmt.Errorf("只有运行中的任务可以停止，当前执行状态: %s", task.ExecutionStatus)
	}

	// 注意：这里需要调用同步引擎停止任务
	// 暂时更新执行状态为失败（被中断）
	updates := map[string]interface{}{
		"execution_status": meta.SyncExecutionStatusFailed,
		"end_time":         time.Now(),
		"error_message":    "任务被手动停止",
		"updated_at":       time.Now(),
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("停止任务失败: %w", err)
	}

	return nil
}

// CancelSyncTask 暂停基础库同步任务（将active状态改为paused）
func (s *SyncTaskService) CancelSyncTask(ctx context.Context, taskID string) error {
	// 获取任务
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态是否允许暂停
	if !task.CanCancel() {
		return fmt.Errorf("任务状态 %s 不允许暂停", task.Status)
	}

	// 将任务状态更新为暂停
	updates := map[string]interface{}{
		"status":     meta.SyncTaskStatusPaused,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("暂停任务失败: %w", err)
	}

	// 如果任务正在运行，需要调用同步引擎取消当前执行
	if task.ExecutionStatus == meta.SyncExecutionStatusRunning {
		// 注意：这里需要调用同步引擎取消任务
		slog.Info("停止运行中的任务", "task_id", taskID)
		// 更新执行状态
		s.updateTaskExecutionStatus(taskID, meta.SyncExecutionStatusFailed, "任务被暂停")
	}

	// 从调度器中移除任务
	if err := s.RemoveScheduledTask(taskID); err != nil {
		slog.Error("从调度器移除任务失败", "task_id", taskID, "error", err)
	}

	return nil
}

// RetrySyncTask 重试基础库同步任务
func (s *SyncTaskService) RetrySyncTask(ctx context.Context, taskID string) (*models.SyncTask, error) {
	// 获取原任务及其接口关联
	var originalTask models.SyncTask
	if err := s.db.Preload("TaskInterfaces").First(&originalTask, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态是否允许重试
	if !contains(meta.GetRetryableTaskStatuses(), originalTask.Status) {
		return nil, fmt.Errorf("任务状态 %s 不允许重试", originalTask.Status)
	}

	// 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建新任务（复制原任务配置，重置执行状态）
	newTask := &models.SyncTask{
		LibraryType:     originalTask.LibraryType,
		LibraryID:       originalTask.LibraryID,
		DataSourceID:    originalTask.DataSourceID,
		TaskType:        originalTask.TaskType,
		TriggerType:     originalTask.TriggerType,
		CronExpression:  originalTask.CronExpression,
		IntervalSeconds: originalTask.IntervalSeconds,
		Status:          originalTask.Status,          // 保持原状态
		ExecutionStatus: meta.SyncExecutionStatusIdle, // 重置执行状态为空闲
		Config:          originalTask.Config,
		CreatedBy:       originalTask.CreatedBy,
	}

	if err := tx.Create(newTask).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建重试任务失败: %w", err)
	}

	// 复制原任务的接口关联
	for _, originalTaskInterface := range originalTask.TaskInterfaces {
		newTaskInterface := &models.SyncTaskInterface{
			TaskID:      newTask.ID,
			InterfaceID: originalTaskInterface.InterfaceID,
			Status:      meta.SyncExecutionStatusIdle, // 初始状态为空闲
			Config:      originalTaskInterface.Config,
		}

		if err := tx.Create(newTaskInterface).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建重试任务接口关联失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 加载关联信息
	if err := s.db.Preload("DataSource").
		Preload("TaskInterfaces").
		Preload("TaskInterfaces.DataInterface").
		Preload("DataInterfaces").
		First(newTask, "id = ?", newTask.ID).Error; err != nil {
		return nil, fmt.Errorf("获取新任务失败: %w", err)
	}

	// 加载库信息
	if err := s.loadLibraryInfo(newTask); err != nil {
		return nil, fmt.Errorf("加载库信息失败: %w", err)
	}

	return newTask, nil
}

// GetSyncTaskStatus 获取基础库同步任务状态
func (s *SyncTaskService) GetSyncTaskStatus(ctx context.Context, taskID string) (*SyncTaskStatusResponse, error) {
	// 获取任务
	var task models.SyncTask
	if err := s.db.Preload("DataSource").Preload("DataInterfaces").First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}

	// 加载库信息
	if err := s.loadLibraryInfo(&task); err != nil {
		return nil, fmt.Errorf("加载库信息失败: %w", err)
	}

	response := &SyncTaskStatusResponse{
		Task:   &task,
		Status: task.Status,
	}

	// 如果任务正在运行，尝试从同步引擎获取实时状态
	if task.ExecutionStatus == meta.SyncExecutionStatusRunning {
		// 注意：这里需要从同步引擎获取实时状态
		// 暂时使用数据库中的信息
	}

	if task.StartTime != nil {
		response.StartTime = task.StartTime
	}

	if task.ErrorMessage != "" {
		response.Error = task.ErrorMessage
	}

	return response, nil
}

// BatchDeleteSyncTasks 批量删除基础库同步任务
func (s *SyncTaskService) BatchDeleteSyncTasks(ctx context.Context, taskIDs []string) (*BatchDeleteResponse, error) {
	response := &BatchDeleteResponse{
		FailedIDs: make([]string, 0),
		Errors:    make([]string, 0),
	}

	for _, taskID := range taskIDs {
		if err := s.DeleteSyncTask(ctx, taskID); err != nil {
			response.FailedIDs = append(response.FailedIDs, taskID)
			response.Errors = append(response.Errors, err.Error())
		} else {
			response.DeletedCount++
		}
	}

	return response, nil
}

// GetSyncTaskStatistics 获取基础库同步任务统计信息
func (s *SyncTaskService) GetSyncTaskStatistics(ctx context.Context, libraryType, libraryID, dataSourceID string) (*SyncTaskStatistics, error) {
	query := s.db.Model(&models.SyncTask{}).Where("library_type = ?", meta.LibraryTypeBasic)

	// 应用过滤条件
	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}
	if dataSourceID != "" {
		query = query.Where("data_source_id = ?", dataSourceID)
	}

	stats := &SyncTaskStatistics{}

	// 获取总任务数
	if err := query.Count(&stats.TotalTasks).Error; err != nil {
		return nil, fmt.Errorf("获取总任务数失败: %w", err)
	}

	// 获取各执行状态任务数
	s.db.Model(&models.SyncTask{}).Where("library_type = ? AND execution_status = ?", meta.LibraryTypeBasic, meta.SyncExecutionStatusIdle).Count(&stats.PendingTasks)
	s.db.Model(&models.SyncTask{}).Where("library_type = ? AND execution_status = ?", meta.LibraryTypeBasic, meta.SyncExecutionStatusRunning).Count(&stats.RunningTasks)
	s.db.Model(&models.SyncTask{}).Where("library_type = ? AND execution_status = ?", meta.LibraryTypeBasic, meta.SyncExecutionStatusSuccess).Count(&stats.SuccessTasks)
	s.db.Model(&models.SyncTask{}).Where("library_type = ? AND execution_status = ?", meta.LibraryTypeBasic, meta.SyncExecutionStatusFailed).Count(&stats.FailedTasks)
	s.db.Model(&models.SyncTask{}).Where("library_type = ? AND status = ?", meta.LibraryTypeBasic, meta.SyncTaskStatusPaused).Count(&stats.CancelledTasks)

	// 计算成功率
	if stats.TotalTasks > 0 {
		stats.SuccessRate = float64(stats.SuccessTasks) / float64(stats.TotalTasks) * 100
	}

	return stats, nil
}

// GetSyncTaskExecutionList 获取基础库同步任务执行记录列表
func (s *SyncTaskService) GetSyncTaskExecutionList(ctx context.Context, req *GetSyncTaskExecutionListRequest) (*SyncTaskExecutionListResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 || req.Size > 100 {
		req.Size = 10
	}

	query := s.db.Model(&models.SyncTaskExecution{}).Preload("Task")

	// 应用过滤条件
	if req.TaskID != "" {
		query = query.Where("task_id = ?", req.TaskID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.ExecutionType != "" {
		query = query.Where("execution_type = ?", req.ExecutionType)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录总数失败: %w", err)
	}

	// 获取分页数据
	var executions []models.SyncTaskExecution
	offset := (req.Page - 1) * req.Size
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.Size).Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录列表失败: %w", err)
	}

	totalPages := (total + int64(req.Size) - 1) / int64(req.Size)

	return &SyncTaskExecutionListResponse{
		Executions: executions,
		Pagination: PaginationInfo{
			Page:       req.Page,
			Size:       req.Size,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetSyncTaskExecutionByID 根据ID获取基础库同步任务执行记录
func (s *SyncTaskService) GetSyncTaskExecutionByID(ctx context.Context, executionID string) (*models.SyncTaskExecution, error) {
	var execution models.SyncTaskExecution
	if err := s.db.Preload("Task").Where("id = ?", executionID).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("执行记录不存在")
		}
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}

	return &execution, nil
}

// CreateSyncTaskExecution 创建基础库同步任务执行记录
func (s *SyncTaskService) CreateSyncTaskExecution(ctx context.Context, taskID, executionType string) (*models.SyncTaskExecution, error) {
	execution := &models.SyncTaskExecution{
		TaskID:        taskID,
		ExecutionType: executionType,
		Status:        meta.SyncExecutionStatusRunning,
		StartTime:     time.Now(),
	}

	if err := s.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	return execution, nil
}

// UpdateSyncTaskExecution 更新基础库同步任务执行记录
func (s *SyncTaskService) UpdateSyncTaskExecution(ctx context.Context, executionID string, status string, result map[string]interface{}, errorMessage string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status != meta.SyncExecutionStatusRunning {
		endTime := time.Now()
		updates["end_time"] = &endTime
	}

	if result != nil {
		updates["result"] = result
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := s.db.Model(&models.SyncTaskExecution{}).Where("id = ?", executionID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新执行记录失败: %w", err)
	}

	return nil
}

// calculateNextRunTime 计算下次执行时间
func (s *SyncTaskService) calculateNextRunTime(task *models.SyncTask) error {
	slog.Debug("计算下次执行时间", "task_id", task.ID, "trigger_type", task.TriggerType, "interval_seconds", task.IntervalSeconds, "cron_expression", task.CronExpression)

	switch task.TriggerType {
	case meta.SyncTaskTriggerManual:
		// 手动执行，不设置下次执行时间
		task.NextRunTime = nil
		slog.Debug("手动任务，不设置下次执行时间", "task_id", task.ID)

	case meta.SyncTaskTriggerOnce:
		// 单次执行，使用计划执行时间
		task.NextRunTime = task.ScheduledTime
		slog.Debug("单次任务", "task_id", task.ID, "scheduled_time", task.ScheduledTime)

	case meta.SyncTaskTriggerInterval:
		// 间隔执行，设置为当前时间加上间隔
		if task.IntervalSeconds > 0 {
			nextTime := time.Now().Add(time.Duration(task.IntervalSeconds) * time.Second)
			task.NextRunTime = &nextTime
			slog.Debug("间隔任务", "task_id", task.ID, "interval_seconds", task.IntervalSeconds, "next_run_time", nextTime.Format("2006-01-02 15:04:05"))
		} else {
			slog.Warn("间隔任务的间隔时间无效", "task_id", task.ID, "interval_seconds", task.IntervalSeconds)
			return fmt.Errorf("间隔任务的间隔时间必须大于0")
		}

	case meta.SyncTaskTriggerCron:
		// Cron表达式执行，使用cron库解析表达式计算下次执行时间
		if task.CronExpression == "" {
			slog.Warn("Cron任务缺少表达式", "task_id", task.ID)
			return fmt.Errorf("Cron任务缺少表达式")
		}

		// 解析Cron表达式 - 使用支持秒的解析器（6个字段：秒 分 时 日 月 周）
		parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedule, err := parser.Parse(task.CronExpression)
		if err != nil {
			slog.Error("解析Cron表达式失败", "task_id", task.ID, "cron_expression", task.CronExpression, "error", err)
			return fmt.Errorf("解析Cron表达式失败: %w", err)
		}

		nextTime := schedule.Next(time.Now())
		task.NextRunTime = &nextTime
		slog.Debug("Cron任务", "task_id", task.ID, "cron_expression", task.CronExpression, "next_run_time", nextTime.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// GetScheduledTasks 获取需要执行的调度任务
func (s *SyncTaskService) GetScheduledTasks(ctx context.Context) ([]models.SyncTask, error) {
	var tasks []models.SyncTask

	// 查找状态为active且配置了调度的基础库任务（cron, interval, once）
	err := s.db.Where("library_type = ? AND status = ? AND trigger_type IN (?, ?, ?)",
		meta.LibraryTypeBasic, meta.SyncTaskStatusActive, "cron", "interval", "once").
		Preload("TaskInterfaces").
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("获取调度任务失败: %w", err)
	}

	return tasks, nil
}

// getShouldExecuteNowTasks 获取应该立即执行的任务（用于interval检查）
func (s *SyncTaskService) getShouldExecuteNowTasks(ctx context.Context) ([]models.SyncTask, error) {
	var tasks []models.SyncTask
	now := time.Now()

	// 查找状态为active且下次执行时间已到的基础库任务
	err := s.db.Where("library_type = ? AND status = ? AND next_run_time IS NOT NULL AND next_run_time <= ?",
		meta.LibraryTypeBasic, meta.SyncTaskStatusActive, now).
		Preload("TaskInterfaces").
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("获取待执行任务失败: %w", err)
	}

	return tasks, nil
}

// UpdateTaskNextRunTime 更新任务的下次执行时间
func (s *SyncTaskService) UpdateTaskNextRunTime(ctx context.Context, taskID string) error {
	var task models.SyncTask
	if err := s.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	if err := s.calculateNextRunTime(&task); err != nil {
		return fmt.Errorf("计算下次执行时间失败: %w", err)
	}

	updates := map[string]interface{}{
		"next_run_time": task.NextRunTime,
		"last_run_time": time.Now(),
		"updated_at":    time.Now(),
	}

	if err := s.db.Model(&models.SyncTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新任务执行时间失败: %w", err)
	}

	return nil
}

// ActivateSyncTask 激活同步任务（将draft状态改为active并加入调度）
func (s *SyncTaskService) ActivateSyncTask(ctx context.Context, taskID string) error {
	// 获取任务
	var task models.SyncTask
	if err := s.db.Preload("TaskInterfaces").First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态
	if task.Status != meta.SyncTaskStatusDraft && task.Status != meta.SyncTaskStatusPaused {
		return fmt.Errorf("只有草稿或暂停状态的任务可以激活，当前状态: %s", task.Status)
	}

	// 更新任务状态为激活
	updates := map[string]interface{}{
		"status":     meta.SyncTaskStatusActive,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("激活任务失败: %w", err)
	}

	// 如果是调度任务，添加到调度器
	if task.TriggerType == "cron" || task.TriggerType == "interval" || task.TriggerType == "once" {
		task.Status = meta.SyncTaskStatusActive
		if err := s.AddScheduledTask(&task); err != nil {
			slog.Error("添加任务到调度器失败", "task_id", taskID, "error", err)
		} else {
			slog.Info("任务已激活并添加到调度器", "task_id", taskID, "trigger_type", task.TriggerType)
		}
	}

	return nil
}

// PauseSyncTask 暂停同步任务（将active状态改为paused并从调度器移除）
func (s *SyncTaskService) PauseSyncTask(ctx context.Context, taskID string) error {
	return s.CancelSyncTask(ctx, taskID) // CancelSyncTask 已经实现了暂停逻辑
}

// ResumeSyncTask 恢复同步任务（将paused状态改为active）
func (s *SyncTaskService) ResumeSyncTask(ctx context.Context, taskID string) error {
	return s.ActivateSyncTask(ctx, taskID) // 复用激活逻辑
}

// SetDistributedLock 设置分布式锁
func (s *SyncTaskService) SetDistributedLock(lock distributed_lock.DistributedLock) {
	s.distributedLock = lock
	if lock != nil {
		slog.Info("基础库同步任务服务已启用分布式锁")
	}
}

// StartScheduler 启动调度器
func (s *SyncTaskService) StartScheduler() error {
	if s.schedulerStarted {
		return fmt.Errorf("调度器已经启动")
	}

	slog.Info("启动基础库同步任务调度器")

	// 启动cron调度器
	s.cron.Start()

	// 启动间隔任务检查器（每分钟检查一次）
	s.intervalTicker = time.NewTicker(1 * time.Minute)
	go s.runIntervalChecker()

	// 加载现有的调度任务
	if err := s.loadScheduledTasks(); err != nil {
		slog.Error("加载调度任务失败", "error", err)
		return err
	}

	s.schedulerStarted = true
	slog.Info("基础库同步任务调度器启动完成")
	return nil
}

// StopScheduler 停止调度器
func (s *SyncTaskService) StopScheduler() {
	if !s.schedulerStarted {
		return
	}

	slog.Info("停止基础库同步任务调度器")

	s.cancel()

	if s.cron != nil {
		s.cron.Stop()
	}

	if s.intervalTicker != nil {
		s.intervalTicker.Stop()
	}

	s.schedulerStarted = false
	slog.Info("基础库同步任务调度器已停止")
}

// loadScheduledTasks 加载调度任务
func (s *SyncTaskService) loadScheduledTasks() error {
	slog.Info("开始加载调度任务")

	// 获取所有待执行的调度任务
	tasks, err := s.GetScheduledTasks(s.ctx)
	if err != nil {
		slog.Error("获取调度任务失败", "error", err)
		return fmt.Errorf("获取调度任务失败: %w", err)
	}

	slog.Info("找到调度任务", "count", len(tasks))

	successCount := 0
	failedCount := 0
	for _, task := range tasks {
		slog.Debug("加载任务", "task_id", task.ID, "trigger_type", task.TriggerType, "status", task.Status)

		if err := s.addTaskToScheduler(&task); err != nil {
			slog.Error("添加任务到调度器失败", "task_id", task.ID, "error", err)
			failedCount++
		} else {
			successCount++
		}
	}

	slog.Info("调度任务加载完成", "total", len(tasks), "success", successCount, "failed", failedCount)
	return nil
}

// addTaskToScheduler 添加任务到调度器
func (s *SyncTaskService) addTaskToScheduler(task *models.SyncTask) error {
	slog.Info("开始添加任务到调度器", "task_id", task.ID, "trigger_type", task.TriggerType, "cron_expression", task.CronExpression, "interval_seconds", task.IntervalSeconds)

	switch task.TriggerType {
	case "cron":
		if task.CronExpression == "" {
			return fmt.Errorf("Cron任务缺少表达式")
		}

		// 验证并添加Cron任务
		// cron.New(cron.WithSeconds()) 需要6个字段：秒 分 时 日 月 周
		taskID := task.ID // 捕获任务ID避免闭包问题
		_, err := s.cron.AddFunc(task.CronExpression, func() {
			s.executeScheduledTask(taskID)
		})
		if err != nil {
			slog.Error("添加Cron任务失败", "task_id", task.ID, "cron_expression", task.CronExpression, "error", err, "help", "Cron表达式需要6个字段（秒 分 时 日 月 周），例如：0 */5 * * * *（每5分钟）")
			return fmt.Errorf("添加Cron任务失败: %w", err)
		}

		slog.Info("添加Cron任务成功", "task_id", task.ID, "cron_expression", task.CronExpression)

	case "once":
		if task.ScheduledTime != nil && task.ScheduledTime.After(time.Now()) {
			// 捕获任务ID和执行时间，避免闭包问题
			taskID := task.ID
			scheduledTime := *task.ScheduledTime
			waitDuration := time.Until(scheduledTime)

			go func() {
				timer := time.NewTimer(waitDuration)
				defer timer.Stop()

				slog.Info("单次任务等待执行", "task_id", taskID, "scheduled_time", scheduledTime.Format("2006-01-02 15:04:05"), "wait_duration", waitDuration)

				select {
				case <-timer.C:
					slog.Info("单次任务时间到，开始执行", "task_id", taskID)
					s.executeScheduledTask(taskID)
				case <-s.ctx.Done():
					slog.Warn("单次任务被取消（调度器关闭）", "task_id", taskID)
					return
				}
			}()

			slog.Info("添加单次任务成功", "task_id", task.ID, "scheduled_time", task.ScheduledTime.Format("2006-01-02 15:04:05"), "wait_duration", waitDuration)
		} else {
			if task.ScheduledTime == nil {
				slog.Warn("单次任务缺少执行时间", "task_id", task.ID)
			} else {
				slog.Warn("单次任务的执行时间已过期", "task_id", task.ID, "scheduled_time", task.ScheduledTime.Format("2006-01-02 15:04:05"), "now", time.Now().Format("2006-01-02 15:04:05"))
			}
		}

	case "interval":
		// 间隔任务由intervalChecker处理
		if task.IntervalSeconds <= 0 {
			slog.Warn("间隔任务的间隔时间无效", "task_id", task.ID, "interval_seconds", task.IntervalSeconds)
			return fmt.Errorf("间隔任务的间隔时间必须大于0")
		}
		slog.Info("添加间隔任务成功", "task_id", task.ID, "interval_seconds", task.IntervalSeconds)
	}

	return nil
}

// runIntervalChecker 运行间隔任务检查器
func (s *SyncTaskService) runIntervalChecker() {
	for {
		select {
		case <-s.intervalTicker.C:
			s.checkIntervalTasks()
		case <-s.ctx.Done():
			return
		}
	}
}

// checkIntervalTasks 检查间隔任务
func (s *SyncTaskService) checkIntervalTasks() {
	slog.Debug("开始检查间隔任务", "timestamp", time.Now().Format("2006-01-02 15:04:05"))

	tasks, err := s.getShouldExecuteNowTasks(s.ctx)
	if err != nil {
		slog.Error("获取间隔任务失败", "error", err)
		return
	}

	slog.Debug("找到待检查的任务", "count", len(tasks))

	for _, task := range tasks {
		slog.Debug("检查任务", "task_id", task.ID, "trigger_type", task.TriggerType, "next_run_time", task.NextRunTime, "should_execute", task.ShouldExecuteNow())

		if task.TriggerType == "interval" && task.ShouldExecuteNow() {
			slog.Info("间隔任务达到执行时间，准备执行", "task_id", task.ID, "next_run_time", task.NextRunTime)
			go s.executeScheduledTask(task.ID)
		}
	}
}

// executeScheduledTask 执行调度任务（带分布式锁）
func (s *SyncTaskService) executeScheduledTask(taskID string) {
	slog.Info("执行调度任务", "task_id", taskID)

	// 如果有分布式锁，使用锁保护执行
	if s.distributedLock != nil {
		lockKey := fmt.Sprintf("basic_library:%s", taskID)
		lockTTL := 10 * time.Minute // 锁的过期时间

		// 尝试获取锁
		locked, err := s.distributedLock.TryLock(s.ctx, lockKey, lockTTL)
		if err != nil {
			slog.Error("获取分布式锁失败", "task_id", taskID, "error", err)
			return
		}

		if !locked {
			slog.Warn("任务正在其他实例执行，跳过", "task_id", taskID)
			return
		}

		// 确保执行完毕后释放锁
		defer func() {
			if unlockErr := s.distributedLock.Unlock(s.ctx, lockKey); unlockErr != nil {
				slog.Error("释放分布式锁失败", "task_id", taskID, "error", unlockErr)
			}
		}()
	}

	// 获取任务详情
	task, err := s.GetSyncTaskByID(s.ctx, taskID)
	if err != nil {
		slog.Error("获取任务失败", "task_id", taskID, "error", err)
		return
	}

	// 检查任务是否可以执行
	if !task.CanStart() {
		slog.Warn("任务不能执行", "task_id", taskID, "status", task.Status, "execution_status", task.ExecutionStatus)
		return
	}

	// 直接调用启动任务方法
	if err := s.StartSyncTask(s.ctx, taskID); err != nil {
		slog.Error("启动调度任务失败", "task_id", taskID, "error", err)
		return
	}

	// 更新下次执行时间
	if err := s.UpdateTaskNextRunTime(s.ctx, taskID); err != nil {
		slog.Error("更新下次执行时间失败", "task_id", taskID, "error", err)
	}

	slog.Info("调度任务已启动", "task_id", taskID)
}

// AddScheduledTask 添加调度任务
func (s *SyncTaskService) AddScheduledTask(task *models.SyncTask) error {
	return s.addTaskToScheduler(task)
}

// RemoveScheduledTask 移除调度任务
func (s *SyncTaskService) RemoveScheduledTask(taskID string) error {
	// 由于cron库不支持按ID移除任务，这里我们重新加载所有任务
	// 在生产环境中，可以考虑使用更高级的调度库
	s.cron.Stop()
	s.cron = cron.New(cron.WithSeconds())
	s.cron.Start()

	return s.loadScheduledTasks()
}

// ReloadScheduledTasks 重新加载调度任务
func (s *SyncTaskService) ReloadScheduledTasks() error {
	s.cron.Stop()
	s.cron = cron.New(cron.WithSeconds())
	s.cron.Start()

	return s.loadScheduledTasks()
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
