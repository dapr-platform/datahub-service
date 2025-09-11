/*
 * @module service/sync_task_service
 * @description 通用同步任务服务，支持基础库和主题库的统一同步任务管理
 * @architecture 分层架构 - 服务层
 * @documentReference ai_docs/refactor_sync_task.md
 * @stateFlow 服务初始化 -> 库类型处理器注册 -> 任务CRUD操作 -> 任务执行管理
 * @rules 通过策略模式支持不同库类型的特定业务逻辑，保持接口统一
 * @dependencies gorm.io/gorm, service/models, service/meta, service/basic_library, service/thematic_library
 * @refs api/controllers/sync_task_controller.go, service/sync_engine
 */

package service

import (
	"context"
	"datahub-service/service/basic_library"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"datahub-service/service/sync_engine"
	"datahub-service/service/thematic_library"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LibraryHandler 库类型处理器接口
type LibraryHandler interface {
	// ValidateLibrary 验证库是否存在
	ValidateLibrary(libraryID string) error

	// ValidateDataSource 验证数据源是否属于该库
	ValidateDataSource(libraryID, dataSourceID string) error

	// ValidateInterface 验证接口是否属于该库
	ValidateInterface(libraryID, interfaceID string) error

	// GetLibraryInfo 获取库信息
	GetLibraryInfo(libraryID string) (interface{}, error)

	// PrepareTaskConfig 准备任务配置
	PrepareTaskConfig(libraryID string, config map[string]interface{}) (map[string]interface{}, error)

	// GetLibraryDataSources 获取库的数据源列表
	GetLibraryDataSources(libraryID string) ([]models.DataSource, error)

	// GetLibraryInterfaces 获取库的接口列表
	GetLibraryInterfaces(libraryID string) ([]models.DataInterface, error)
}

// BasicLibraryHandler 基础库处理器
type BasicLibraryHandler struct {
	db      *gorm.DB
	service *basic_library.Service
}

// NewBasicLibraryHandler 创建基础库处理器
func NewBasicLibraryHandler(db *gorm.DB, service *basic_library.Service) *BasicLibraryHandler {
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

// ThematicLibraryHandler 主题库处理器
type ThematicLibraryHandler struct {
	db      *gorm.DB
	service *thematic_library.Service
}

// NewThematicLibraryHandler 创建主题库处理器
func NewThematicLibraryHandler(db *gorm.DB, service *thematic_library.Service) *ThematicLibraryHandler {
	return &ThematicLibraryHandler{
		db:      db,
		service: service,
	}
}

// ValidateLibrary 验证主题库是否存在
func (h *ThematicLibraryHandler) ValidateLibrary(libraryID string) error {
	// 暂时通过查询主题库表来验证
	var thematicLibrary models.ThematicLibrary
	err := h.db.Where("id = ?", libraryID).First(&thematicLibrary).Error
	if err != nil {
		return fmt.Errorf("主题库不存在: %w", err)
	}
	return nil
}

// ValidateDataSource 验证数据源是否属于主题库
func (h *ThematicLibraryHandler) ValidateDataSource(libraryID, dataSourceID string) error {
	// 主题库的数据源验证逻辑（待实现）
	// 目前返回成功，实际项目中需要根据主题库的数据源关系进行验证
	return nil
}

// ValidateInterface 验证接口是否属于主题库
func (h *ThematicLibraryHandler) ValidateInterface(libraryID, interfaceID string) error {
	// 主题库的接口验证逻辑（待实现）
	// 目前返回成功，实际项目中需要根据主题库的接口关系进行验证
	return nil
}

// GetLibraryInfo 获取主题库信息
func (h *ThematicLibraryHandler) GetLibraryInfo(libraryID string) (interface{}, error) {
	var thematicLibrary models.ThematicLibrary
	err := h.db.Where("id = ?", libraryID).First(&thematicLibrary).Error
	if err != nil {
		return nil, err
	}
	return &thematicLibrary, nil
}

// PrepareTaskConfig 准备主题库任务配置
func (h *ThematicLibraryHandler) PrepareTaskConfig(libraryID string, config map[string]interface{}) (map[string]interface{}, error) {
	// 为主题库添加特定的配置项
	if config == nil {
		config = make(map[string]interface{})
	}

	config["library_type"] = meta.LibraryTypeThematic
	config["library_id"] = libraryID

	// 添加主题库特定的默认配置
	if _, exists := config["batch_size"]; !exists {
		config["batch_size"] = 2000 // 主题库可能需要更大的批处理
	}
	if _, exists := config["timeout"]; !exists {
		config["timeout"] = "60m" // 主题库可能需要更长的超时时间
	}

	return config, nil
}

// GetLibraryDataSources 获取主题库的数据源列表
func (h *ThematicLibraryHandler) GetLibraryDataSources(libraryID string) ([]models.DataSource, error) {
	// 主题库的数据源获取逻辑（待实现）
	// 目前返回空列表，实际项目中需要根据主题库的数据源关系进行查询
	return []models.DataSource{}, nil
}

// GetLibraryInterfaces 获取主题库的接口列表
func (h *ThematicLibraryHandler) GetLibraryInterfaces(libraryID string) ([]models.DataInterface, error) {
	// 主题库的接口获取逻辑（待实现）
	// 目前返回空列表，实际项目中需要根据主题库的接口关系进行查询
	return []models.DataInterface{}, nil
}

// SyncTaskService 通用同步任务服务
type SyncTaskService struct {
	db         *gorm.DB
	handlers   map[string]LibraryHandler
	syncEngine *sync_engine.SyncEngine
}

// NewSyncTaskService 创建同步任务服务
func NewSyncTaskService(db *gorm.DB, basicLibService *basic_library.Service, thematicLibService *thematic_library.Service, syncEngine *sync_engine.SyncEngine) *SyncTaskService {
	service := &SyncTaskService{
		db:         db,
		handlers:   make(map[string]LibraryHandler),
		syncEngine: syncEngine,
	}

	// 注册库类型处理器
	service.handlers[meta.LibraryTypeBasic] = NewBasicLibraryHandler(db, basicLibService)
	service.handlers[meta.LibraryTypeThematic] = NewThematicLibraryHandler(db, thematicLibService)

	return service
}

// getHandler 获取库类型处理器
func (s *SyncTaskService) getHandler(libraryType string) (LibraryHandler, error) {
	handler, exists := s.handlers[libraryType]
	if !exists {
		return nil, fmt.Errorf("不支持的库类型: %s", libraryType)
	}
	return handler, nil
}

// CreateSyncTask 创建同步任务
func (s *SyncTaskService) CreateSyncTask(ctx context.Context, req *CreateSyncTaskRequest) (*models.SyncTask, error) {
	// 验证库类型
	if !meta.IsValidLibraryType(req.LibraryType) {
		return nil, fmt.Errorf("无效的库类型: %s", req.LibraryType)
	}

	// 获取处理器
	handler, err := s.getHandler(req.LibraryType)
	if err != nil {
		return nil, err
	}

	// 验证库存在
	if err := handler.ValidateLibrary(req.LibraryID); err != nil {
		return nil, err
	}

	// 验证数据源
	if err := handler.ValidateDataSource(req.LibraryID, req.DataSourceID); err != nil {
		return nil, err
	}

	// 验证所有接口
	if len(req.InterfaceIDs) == 0 {
		return nil, fmt.Errorf("必须提供至少一个接口ID")
	}

	for _, interfaceID := range req.InterfaceIDs {
		if err := handler.ValidateInterface(req.LibraryID, interfaceID); err != nil {
			return nil, fmt.Errorf("验证接口 %s 失败: %w", interfaceID, err)
		}
	}

	// 准备任务配置
	config, err := handler.PrepareTaskConfig(req.LibraryID, req.Config)
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
		LibraryType:     req.LibraryType,
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

// GetSyncTaskByID 根据ID获取同步任务
func (s *SyncTaskService) GetSyncTaskByID(ctx context.Context, taskID string) (*models.SyncTask, error) {
	var task models.SyncTask
	if err := s.db.Preload("DataSource").
		Preload("TaskInterfaces").
		Preload("TaskInterfaces.DataInterface").
		Preload("DataInterfaces").
		First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取同步任务失败: %w", err)
	}

	// 加载库信息
	if err := s.loadLibraryInfo(&task); err != nil {
		return nil, fmt.Errorf("加载库信息失败: %w", err)
	}

	return &task, nil
}

// loadLibraryInfo 加载库信息
func (s *SyncTaskService) loadLibraryInfo(task *models.SyncTask) error {
	handler, err := s.getHandler(task.LibraryType)
	if err != nil {
		return err
	}

	libraryInfo, err := handler.GetLibraryInfo(task.LibraryID)
	if err != nil {
		return err
	}

	// 根据库类型设置对应的库信息
	switch task.LibraryType {
	case meta.LibraryTypeBasic:
		if basicLib, ok := libraryInfo.(*models.BasicLibrary); ok {
			task.BasicLibrary = basicLib
		}
	case meta.LibraryTypeThematic:
		if thematicLib, ok := libraryInfo.(*models.ThematicLibrary); ok {
			task.ThematicLibrary = thematicLib
		}
	}

	return nil
}

// SyncTaskInterfaceConfig 接口级别的配置
type SyncTaskInterfaceConfig struct {
	InterfaceID string                 `json:"interface_id"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// CreateSyncTaskRequest 创建同步任务请求
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

// UpdateSyncTaskRequest 更新同步任务请求
type UpdateSyncTaskRequest struct {
	TriggerType      string                    `json:"trigger_type,omitempty"`
	CronExpression   string                    `json:"cron_expression,omitempty"`
	IntervalSeconds  int                       `json:"interval_seconds,omitempty"`
	Config           map[string]interface{}    `json:"config,omitempty"`
	InterfaceIDs     []string                  `json:"interface_ids,omitempty"`
	InterfaceConfigs []SyncTaskInterfaceConfig `json:"interface_configs,omitempty"`
	UpdatedBy        string                    `json:"updated_by"`
}

// GetSyncTaskListRequest 获取同步任务列表请求
type GetSyncTaskListRequest struct {
	Page         int    `json:"page"`
	Size         int    `json:"size"`
	LibraryType  string `json:"library_type,omitempty"`
	LibraryID    string `json:"library_id,omitempty"`
	DataSourceID string `json:"data_source_id,omitempty"`
	Status       string `json:"status,omitempty"`
	TaskType     string `json:"task_type,omitempty"`
}

// SyncTaskListResponse 同步任务列表响应
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

// SyncTaskStatusResponse 同步任务状态响应
type SyncTaskStatusResponse struct {
	Task      *models.SyncTask     `json:"task"`
	StartTime *time.Time           `json:"start_time,omitempty"`
	Status    string               `json:"status"`
	Progress  *models.SyncProgress `json:"progress,omitempty"`
	Error     string               `json:"error,omitempty"`
	Result    *models.SyncResult   `json:"result,omitempty"`
	Processor string               `json:"processor,omitempty"`
}

// GetSyncTaskExecutionListRequest 获取同步任务执行记录列表请求
type GetSyncTaskExecutionListRequest struct {
	Page          int    `json:"page"`
	Size          int    `json:"size"`
	TaskID        string `json:"task_id,omitempty"`
	Status        string `json:"status,omitempty"`
	ExecutionType string `json:"execution_type,omitempty"`
}

// SyncTaskExecutionListResponse 同步任务执行记录列表响应
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

// SyncTaskStatistics 同步任务统计信息
type SyncTaskStatistics struct {
	TotalTasks     int64   `json:"total_tasks"`
	PendingTasks   int64   `json:"pending_tasks"`
	RunningTasks   int64   `json:"running_tasks"`
	SuccessTasks   int64   `json:"success_tasks"`
	FailedTasks    int64   `json:"failed_tasks"`
	CancelledTasks int64   `json:"cancelled_tasks"`
	SuccessRate    float64 `json:"success_rate"`
}

// GetSyncTaskList 获取同步任务列表
func (s *SyncTaskService) GetSyncTaskList(ctx context.Context, req *GetSyncTaskListRequest) (*SyncTaskListResponse, error) {
	query := s.db.Model(&models.SyncTask{})

	// 应用过滤条件
	if req.LibraryType != "" {
		query = query.Where("library_type = ?", req.LibraryType)
	}
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

// UpdateSyncTask 更新同步任务
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
		// 获取处理器以验证接口
		handler, err := s.getHandler(task.LibraryType)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		// 验证所有新接口
		for _, interfaceID := range req.InterfaceIDs {
			if err := handler.ValidateInterface(task.LibraryID, interfaceID); err != nil {
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

// DeleteSyncTask 删除同步任务
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

// StartSyncTask 启动同步任务
func (s *SyncTaskService) StartSyncTask(ctx context.Context, taskID string) error {
	fmt.Printf("[DEBUG] SyncTaskService.StartSyncTask - 开始启动任务: %s\n", taskID)

	// 获取任务
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		fmt.Printf("[ERROR] SyncTaskService.StartSyncTask - 任务不存在: %s, 错误: %v\n", taskID, err)
		return fmt.Errorf("任务不存在: %w", err)
	}

	fmt.Printf("[DEBUG] SyncTaskService.StartSyncTask - 找到任务: %s, 当前状态: %s, 类型: %s\n", task.ID, task.Status, task.TaskType)

	// 检查任务状态
	if task.Status != meta.SyncTaskStatusPending && task.Status != meta.SyncTaskStatusFailed {
		fmt.Printf("[ERROR] SyncTaskService.StartSyncTask - 任务状态不允许启动: %s, 当前状态: %s\n", taskID, task.Status)
		return fmt.Errorf("只有待执行状态或失败状态的任务可以启动，当前状态: %s", task.Status)
	}

	// 创建同步引擎请求
	syncRequest := &sync_engine.SyncTaskRequest{
		TaskID:       task.ID, // 传递任务ID，用于手动执行
		LibraryType:  task.LibraryType,
		LibraryID:    task.LibraryID,
		DataSourceID: task.DataSourceID,
		SyncType:     sync_engine.SyncType(task.TaskType),
		Config:       map[string]interface{}(task.Config),
		Priority:     1, // 默认优先级
		ScheduledBy:  "manual",
	}

	// 加载任务接口关联
	if err := s.db.Preload("TaskInterfaces").First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("加载任务接口信息失败: %w", err)
	}

	// 设置接口ID列表
	var interfaceIDs []string
	for _, taskInterface := range task.TaskInterfaces {
		interfaceIDs = append(interfaceIDs, taskInterface.InterfaceID)
	}
	syncRequest.InterfaceIDs = interfaceIDs


	fmt.Printf("[DEBUG] SyncTaskService.StartSyncTask - 准备提交任务到同步引擎: %+v\n", syncRequest)

	// 提交任务到同步引擎
	syncTask, err := s.syncEngine.SubmitSyncTask(syncRequest)
	if err != nil {
		fmt.Printf("[ERROR] SyncTaskService.StartSyncTask - 提交任务到引擎失败: %v\n", err)
		return fmt.Errorf("提交同步任务到引擎失败: %w", err)
	}

	fmt.Printf("[DEBUG] SyncTaskService.StartSyncTask - 任务已提交到同步引擎，返回任务ID: %s\n", syncTask.ID)

	return nil
}

// StopSyncTask 停止同步任务
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

// CancelSyncTask 取消同步任务
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

// RetrySyncTask 重试同步任务
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

// GetSyncTaskStatus 获取同步任务状态
func (s *SyncTaskService) GetSyncTaskStatus(ctx context.Context, taskID string) (*SyncTaskStatusResponse, error) {
	// 获取任务
	var task models.SyncTask
	if err := s.db.Preload("DataSource").Preload("DataInterface").First(&task, "id = ?", taskID).Error; err != nil {
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

// BatchDeleteSyncTasks 批量删除同步任务
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

// GetSyncTaskStatistics 获取同步任务统计信息
func (s *SyncTaskService) GetSyncTaskStatistics(ctx context.Context, libraryType, libraryID, dataSourceID string) (*SyncTaskStatistics, error) {
	query := s.db.Model(&models.SyncTask{})

	// 应用过滤条件
	if libraryType != "" {
		query = query.Where("library_type = ?", libraryType)
	}
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

// GetSyncTaskExecutionList 获取同步任务执行记录列表
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

// GetSyncTaskExecutionByID 根据ID获取同步任务执行记录
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

// CreateSyncTaskExecution 创建同步任务执行记录
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

// UpdateSyncTaskExecution 更新同步任务执行记录
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

	// 查找状态为pending且下次执行时间已到的任务
	err := s.db.Where("status = ? AND next_run_time IS NOT NULL AND next_run_time <= ?",
		meta.SyncTaskStatusPending, now).Find(&tasks).Error
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

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
