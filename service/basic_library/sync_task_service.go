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
	"datahub-service/service/interface_executor"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
	"log"
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
		Status:          meta.SyncTaskStatusPending,
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
			Status:      meta.SyncTaskStatusPending,
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
	TriggerType      string                    `json:"trigger_type,omitempty"`
	CronExpression   string                    `json:"cron_expression,omitempty"`
	IntervalSeconds  int                       `json:"interval_seconds,omitempty"`
	Config           map[string]interface{}    `json:"config,omitempty"`
	InterfaceIDs     []string                  `json:"interface_ids,omitempty"`
	InterfaceConfigs []SyncTaskInterfaceConfig `json:"interface_configs,omitempty"`
	UpdatedBy        string                    `json:"updated_by"`
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
			fmt.Printf("加载库信息失败: %v\n", err)
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

	// 检查任务状态是否允许更新
	if !contains(meta.GetUpdatableTaskStatuses(), task.Status) {
		return nil, fmt.Errorf("任务状态 %s 不允许更新", task.Status)
	}

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

	// 更新任务基本信息
	if err := tx.Model(&task).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新任务失败: %w", err)
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
				Status:      meta.SyncTaskStatusPending,
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

	// 删除任务
	if err := s.db.Delete(&task).Error; err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}

	return nil
}

// StartSyncTask 启动基础库同步任务
func (s *SyncTaskService) StartSyncTask(ctx context.Context, taskID string) error {
	fmt.Printf("[DEBUG] SyncTaskService.StartSyncTask - 开始启动任务: %s\n", taskID)

	// 获取任务详细信息
	var task models.SyncTask
	if err := s.db.Preload("TaskInterfaces").First(&task, "id = ?", taskID).Error; err != nil {
		fmt.Printf("[ERROR] SyncTaskService.StartSyncTask - 任务不存在: %s, 错误: %v\n", taskID, err)
		return fmt.Errorf("任务不存在: %w", err)
	}

	fmt.Printf("[DEBUG] SyncTaskService.StartSyncTask - 找到任务: %s, 当前状态: %s, 类型: %s\n", task.ID, task.Status, task.TaskType)

	// 检查任务状态
	if task.Status != meta.SyncTaskStatusPending && task.Status != meta.SyncTaskStatusFailed && task.Status != meta.SyncTaskStatusCancelled {
		fmt.Printf("[ERROR] SyncTaskService.StartSyncTask - 任务状态不允许启动: %s, 当前状态: %s\n", taskID, task.Status)
		return fmt.Errorf("只有待执行状态或失败状态的任务可以启动，当前状态: %s", task.Status)
	}

	// 更新任务状态为运行中
	if err := s.db.Model(&task).Updates(map[string]interface{}{
		"status":     meta.SyncTaskStatusRunning,
		"start_time": time.Now(),
		"updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 创建独立的context用于任务执行，避免HTTP请求context被取消影响任务执行
	taskCtx := context.Background()

	// 如果有指定接口，使用InterfaceExecutor执行
	if len(task.TaskInterfaces) > 0 {
		go s.executeTaskWithInterfaces(taskCtx, &task)
	} else {
		// 没有指定接口的情况，返回错误
		s.updateTaskStatus(task.ID, meta.SyncTaskStatusFailed, "任务必须关联至少一个接口")
		return fmt.Errorf("任务必须关联至少一个接口")
	}

	return nil
}

// 注意：hasIncrementalConfig 和 getLastSyncTime 方法已被移除
// 现在统一使用 sync 执行类型，增量逻辑由 interface_executor 内部处理

// executeTaskWithInterfaces 使用InterfaceExecutor执行任务
func (s *SyncTaskService) executeTaskWithInterfaces(ctx context.Context, task *models.SyncTask) {
	fmt.Printf("[DEBUG] SyncTaskService.executeTaskWithInterfaces - 开始执行任务: %s\n", task.ID)

	// 创建执行记录
	execution, err := s.CreateSyncTaskExecution(ctx, task.ID, "interface_executor")
	if err != nil {
		fmt.Printf("[ERROR] 创建执行记录失败: %v\n", err)
		s.updateTaskStatus(task.ID, meta.SyncTaskStatusFailed, err.Error())
		return
	}

	var totalProcessed int64
	var hasError bool
	var errorMessages []string

	// 执行每个接口
	for _, taskInterface := range task.TaskInterfaces {
		fmt.Printf("[DEBUG] 执行接口: %s\n", taskInterface.InterfaceID)

		// 使用统一的sync类型，内部根据接口的incremental_config自动判断全量/增量
		executeType := "sync" // 统一使用sync类型

		fmt.Printf("[DEBUG] 接口 %s 使用统一的sync执行类型，将根据接口配置自动判断全量/增量同步\n", taskInterface.InterfaceID)

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
			fmt.Printf("[ERROR] %s\n", errorMsg)
			continue
		}

		if !response.Success {
			hasError = true
			errorMsg := fmt.Sprintf("接口 %s 执行失败: %s", taskInterface.InterfaceID, response.Error)
			errorMessages = append(errorMessages, errorMsg)
			fmt.Printf("[ERROR] %s\n", errorMsg)
			continue
		}

		totalProcessed += response.UpdatedRows
		fmt.Printf("[DEBUG] 接口 %s 执行成功，更新行数: %d\n", taskInterface.InterfaceID, response.UpdatedRows)
	}

	// 更新任务状态
	var finalStatus string
	var errorMessage string

	if hasError {
		if totalProcessed > 0 {
			finalStatus = meta.SyncTaskStatusSuccess // 部分成功
			errorMessage = fmt.Sprintf("部分接口执行失败: %v", errorMessages)
		} else {
			finalStatus = meta.SyncTaskStatusFailed
			errorMessage = fmt.Sprintf("所有接口执行失败: %v", errorMessages)
		}
	} else {
		finalStatus = meta.SyncTaskStatusSuccess
	}

	// 更新任务
	updates := map[string]interface{}{
		"status":         finalStatus,
		"end_time":       time.Now(),
		"processed_rows": totalProcessed,
		"progress":       100,
		"updated_at":     time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := s.db.Model(&models.SyncTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		fmt.Printf("[ERROR] 更新任务状态失败: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] 任务状态更新成功: %s -> %s\n", task.ID, finalStatus)
	}

	// 更新执行记录
	result := map[string]interface{}{
		"processed_rows":  totalProcessed,
		"interface_count": len(task.TaskInterfaces),
		"success_count":   len(task.TaskInterfaces) - len(errorMessages),
		"failed_count":    len(errorMessages),
	}

	if err := s.UpdateSyncTaskExecution(ctx, execution.ID, finalStatus, result, errorMessage); err != nil {
		fmt.Printf("[ERROR] 更新执行记录失败: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] 执行记录更新成功: %s -> %s\n", execution.ID, finalStatus)
	}

	fmt.Printf("[DEBUG] 任务 %s 执行完成，状态: %s，处理行数: %d\n", task.ID, finalStatus, totalProcessed)
}

// updateTaskStatus 更新任务状态的辅助方法
func (s *SyncTaskService) updateTaskStatus(taskID, status, errorMessage string) {
	updates := map[string]interface{}{
		"status":     status,
		"end_time":   time.Now(),
		"updated_at": time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := s.db.Model(&models.SyncTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		fmt.Printf("[ERROR] 更新任务状态失败: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] 任务状态更新成功: %s -> %s\n", taskID, status)
	}
}

// StopSyncTask 停止基础库同步任务
func (s *SyncTaskService) StopSyncTask(ctx context.Context, taskID string) error {
	// 获取任务
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态
	if task.Status != meta.SyncTaskStatusRunning {
		return fmt.Errorf("只有运行中的任务可以停止，当前状态: %s", task.Status)
	}

	// 注意：这里需要调用同步引擎停止任务
	// 暂时更新状态为已取消
	updates := map[string]interface{}{
		"status":     meta.SyncTaskStatusCancelled,
		"end_time":   time.Now(),
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("停止任务失败: %w", err)
	}

	return nil
}

// CancelSyncTask 取消基础库同步任务
func (s *SyncTaskService) CancelSyncTask(ctx context.Context, taskID string) error {
	// 获取任务
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查任务状态是否允许取消
	if !contains(meta.GetCancellableTaskStatuses(), task.Status) {
		return fmt.Errorf("任务状态 %s 不允许取消", task.Status)
	}

	// 更新任务状态
	updates := map[string]interface{}{
		"status":     meta.SyncTaskStatusCancelled,
		"updated_at": time.Now(),
	}

	if task.Status == meta.SyncTaskStatusPending {
		updates["end_time"] = time.Now()
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}

	// 如果任务正在运行，需要调用同步引擎取消
	if task.Status == meta.SyncTaskStatusRunning {
		// 注意：这里需要调用同步引擎取消任务
		fmt.Printf("取消运行中的任务: %s\n", taskID)
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

	// 创建新任务
	newTask := &models.SyncTask{
		LibraryType:  originalTask.LibraryType,
		LibraryID:    originalTask.LibraryID,
		DataSourceID: originalTask.DataSourceID,
		TaskType:     originalTask.TaskType,
		Status:       meta.SyncTaskStatusPending,
		Config:       originalTask.Config,
		CreatedBy:    originalTask.CreatedBy,
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
			Status:      meta.SyncTaskStatusPending,
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
	if task.Status == meta.SyncTaskStatusRunning {
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

	// 获取各状态任务数
	query.Where("status = ?", meta.SyncTaskStatusPending).Count(&stats.PendingTasks)
	query.Where("status = ?", meta.SyncTaskStatusRunning).Count(&stats.RunningTasks)
	query.Where("status = ?", meta.SyncTaskStatusSuccess).Count(&stats.SuccessTasks)
	query.Where("status = ?", meta.SyncTaskStatusFailed).Count(&stats.FailedTasks)
	query.Where("status = ?", meta.SyncTaskStatusCancelled).Count(&stats.CancelledTasks)

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
	switch task.TriggerType {
	case meta.SyncTaskTriggerManual:
		// 手动执行，不设置下次执行时间
		task.NextRunTime = nil
	case meta.SyncTaskTriggerOnce:
		// 单次执行，使用计划执行时间
		task.NextRunTime = task.ScheduledTime
	case meta.SyncTaskTriggerInterval:
		// 间隔执行，设置为当前时间加上间隔
		if task.IntervalSeconds > 0 {
			nextTime := time.Now().Add(time.Duration(task.IntervalSeconds) * time.Second)
			task.NextRunTime = &nextTime
		}
	case meta.SyncTaskTriggerCron:
		// Cron表达式执行，需要解析Cron表达式计算下次执行时间
		// 这里可以使用第三方库如 github.com/robfig/cron/v3 来解析
		// 暂时简化处理，设置为1小时后
		nextTime := time.Now().Add(time.Hour)
		task.NextRunTime = &nextTime
	}

	return nil
}

// GetScheduledTasks 获取需要执行的调度任务
func (s *SyncTaskService) GetScheduledTasks(ctx context.Context) ([]models.SyncTask, error) {
	var tasks []models.SyncTask
	now := time.Now()

	// 查找状态为pending且下次执行时间已到的基础库任务
	err := s.db.Where("library_type = ? AND status = ? AND next_run_time IS NOT NULL AND next_run_time <= ?",
		meta.LibraryTypeBasic, meta.SyncTaskStatusPending, now).Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("获取调度任务失败: %w", err)
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

// StartScheduler 启动调度器
func (s *SyncTaskService) StartScheduler() error {
	if s.schedulerStarted {
		return fmt.Errorf("调度器已经启动")
	}

	log.Println("启动基础库同步任务调度器")

	// 启动cron调度器
	s.cron.Start()

	// 启动间隔任务检查器（每分钟检查一次）
	s.intervalTicker = time.NewTicker(1 * time.Minute)
	go s.runIntervalChecker()

	// 加载现有的调度任务
	if err := s.loadScheduledTasks(); err != nil {
		log.Printf("加载调度任务失败: %v", err)
		return err
	}

	s.schedulerStarted = true
	log.Println("基础库同步任务调度器启动完成")
	return nil
}

// StopScheduler 停止调度器
func (s *SyncTaskService) StopScheduler() {
	if !s.schedulerStarted {
		return
	}

	log.Println("停止基础库同步任务调度器")

	s.cancel()

	if s.cron != nil {
		s.cron.Stop()
	}

	if s.intervalTicker != nil {
		s.intervalTicker.Stop()
	}

	s.schedulerStarted = false
	log.Println("基础库同步任务调度器已停止")
}

// loadScheduledTasks 加载调度任务
func (s *SyncTaskService) loadScheduledTasks() error {
	// 获取所有待执行的调度任务
	tasks, err := s.GetScheduledTasks(s.ctx)
	if err != nil {
		return fmt.Errorf("获取调度任务失败: %w", err)
	}

	for _, task := range tasks {
		if err := s.addTaskToScheduler(&task); err != nil {
			log.Printf("添加任务到调度器失败 [%s]: %v", task.ID, err)
		}
	}

	log.Printf("加载了 %d 个调度任务", len(tasks))
	return nil
}

// addTaskToScheduler 添加任务到调度器
func (s *SyncTaskService) addTaskToScheduler(task *models.SyncTask) error {
	switch task.TriggerType {
	case "cron":
		if task.CronExpression == "" {
			return fmt.Errorf("Cron任务缺少表达式")
		}

		_, err := s.cron.AddFunc(task.CronExpression, func() {
			s.executeScheduledTask(task.ID)
		})
		if err != nil {
			return fmt.Errorf("添加Cron任务失败: %w", err)
		}

		log.Printf("添加Cron任务: %s [%s]", task.ID, task.CronExpression)

	case "once":
		if task.ScheduledTime != nil && task.ScheduledTime.After(time.Now()) {
			go func() {
				timer := time.NewTimer(time.Until(*task.ScheduledTime))
				defer timer.Stop()

				select {
				case <-timer.C:
					s.executeScheduledTask(task.ID)
				case <-s.ctx.Done():
					return
				}
			}()

			log.Printf("添加单次任务: %s [%s]", task.ID, task.ScheduledTime.Format("2006-01-02 15:04:05"))
		}

	case "interval":
		// 间隔任务由intervalChecker处理
		log.Printf("添加间隔任务: %s [%d秒]", task.ID, task.IntervalSeconds)
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
	tasks, err := s.GetScheduledTasks(s.ctx)
	if err != nil {
		log.Printf("获取间隔任务失败: %v", err)
		return
	}

	for _, task := range tasks {
		if task.TriggerType == "interval" && task.ShouldExecuteNow() {
			go s.executeScheduledTask(task.ID)
		}
	}
}

// executeScheduledTask 执行调度任务
func (s *SyncTaskService) executeScheduledTask(taskID string) {
	log.Printf("执行调度任务: %s", taskID)

	// 获取任务详情
	task, err := s.GetSyncTaskByID(s.ctx, taskID)
	if err != nil {
		log.Printf("获取任务失败 [%s]: %v", taskID, err)
		return
	}

	// 检查任务是否可以执行
	if !task.CanStart() {
		log.Printf("任务不能执行 [%s]: 状态=%s", taskID, task.Status)
		return
	}

	// 直接调用启动任务方法
	if err := s.StartSyncTask(s.ctx, taskID); err != nil {
		log.Printf("启动调度任务失败 [%s]: %v", taskID, err)
		return
	}

	log.Printf("调度任务已启动 [%s]", taskID)
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
