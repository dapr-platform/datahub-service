/*
 * @module service/thematic_sync_service
 * @description 主题同步服务，提供主题数据同步的业务逻辑和服务接口
 * @architecture 服务层 - 封装业务逻辑，提供统一的服务接口
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 任务管理 -> 同步执行 -> 状态跟踪 -> 结果处理
 * @rules 确保业务逻辑的完整性和一致性，支持事务性操作
 * @dependencies gorm.io/gorm, context, service/models
 * @refs service/thematic_sync/sync_engine.go, service/models/thematic_sync.go
 */

package thematic_library

import (
	"context"
	"datahub-service/service/models"
	"datahub-service/service/thematic_library/thematic_sync"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ThematicSyncService 主题同步服务
type ThematicSyncService struct {
	db         *gorm.DB
	syncEngine *thematic_sync.ThematicSyncEngine
}

// NewThematicSyncService 创建主题同步服务
func NewThematicSyncService(db *gorm.DB,
	basicLibraryService thematic_sync.BasicLibraryServiceInterface,
	thematicLibraryService thematic_sync.ThematicLibraryServiceInterface) *ThematicSyncService {

	syncEngine := thematic_sync.NewThematicSyncEngine(db, basicLibraryService, thematicLibraryService)

	return &ThematicSyncService{
		db:         db,
		syncEngine: syncEngine,
	}
}

// parseJSONToJSONB 将 JSON 字符串转换为 JSONB (兼容性保留)
func parseJSONToJSONB(jsonStr string) models.JSONB {
	if jsonStr == "" {
		return models.JSONB{}
	}

	var result models.JSONB
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// 如果解析失败，返回空 JSONB
		return models.JSONB{}
	}
	return result
}

// structToJSONB 将结构体转换为 JSONB
func structToJSONB(v interface{}) models.JSONB {
	if v == nil {
		return models.JSONB{}
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return models.JSONB{}
	}

	var result models.JSONB
	if err := json.Unmarshal(bytes, &result); err != nil {
		return models.JSONB{}
	}
	return result
}

// CreateSyncTask 创建同步任务
func (tss *ThematicSyncService) CreateSyncTask(ctx context.Context, req *CreateThematicSyncTaskRequest) (*models.ThematicSyncTask, error) {
	task := &models.ThematicSyncTask{
		ID:                  uuid.New().String(),
		ThematicLibraryID:   req.ThematicLibraryID,
		ThematicInterfaceID: req.ThematicInterfaceID,
		TaskName:            req.TaskName,
		Description:         req.Description,
		SourceLibraries:     structToJSONB(req.SourceLibraries),
		AggregationConfig:   structToJSONB(req.AggregationConfig),
		KeyMatchingRules:    structToJSONB(req.KeyMatchingRules),
		FieldMappingRules:   structToJSONB(req.FieldMappingRules),
		CleansingRules:      structToJSONB(req.CleansingRules),
		PrivacyRules:        structToJSONB(req.PrivacyRules),
		QualityRules:        structToJSONB(req.QualityRules),
		Status:              "draft",
		CreatedAt:           time.Now(),
		CreatedBy:           req.CreatedBy,
		UpdatedAt:           time.Now(),
		UpdatedBy:           req.CreatedBy,
	}

	// 处理调度配置
	if req.ScheduleConfig != nil {
		task.TriggerType = req.ScheduleConfig.Type
		task.CronExpression = req.ScheduleConfig.CronExpression
		task.IntervalSeconds = req.ScheduleConfig.IntervalSeconds
		task.ScheduledTime = req.ScheduleConfig.ScheduledTime
	}

	// 计算下次执行时间
	task.UpdateNextRunTime()

	if err := tss.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建同步任务失败: %w", err)
	}

	return task, nil
}

// UpdateSyncTask 更新同步任务
func (tss *ThematicSyncService) UpdateSyncTask(ctx context.Context, taskID string, req *UpdateThematicSyncTaskRequest) (*models.ThematicSyncTask, error) {
	var task models.ThematicSyncTask
	if err := tss.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取同步任务失败: %w", err)
	}

	// 更新字段
	if req.TaskName != "" {
		task.TaskName = req.TaskName
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Status != "" {
		task.Status = req.Status
	}

	// 更新调度配置
	if req.ScheduleConfig != nil {
		task.TriggerType = req.ScheduleConfig.Type
		task.CronExpression = req.ScheduleConfig.CronExpression
		task.IntervalSeconds = req.ScheduleConfig.IntervalSeconds
		task.ScheduledTime = req.ScheduleConfig.ScheduledTime
	}

	// 更新各种配置规则
	if req.AggregationConfig != nil {
		task.AggregationConfig = structToJSONB(req.AggregationConfig)
	}
	if req.KeyMatchingRules != nil {
		task.KeyMatchingRules = structToJSONB(req.KeyMatchingRules)
	}
	if req.FieldMappingRules != nil {
		task.FieldMappingRules = structToJSONB(req.FieldMappingRules)
	}
	if req.CleansingRules != nil {
		task.CleansingRules = structToJSONB(req.CleansingRules)
	}
	if req.PrivacyRules != nil {
		task.PrivacyRules = structToJSONB(req.PrivacyRules)
	}
	if req.QualityRules != nil {
		task.QualityRules = structToJSONB(req.QualityRules)
	}

	task.UpdatedAt = time.Now()
	task.UpdatedBy = req.UpdatedBy

	// 重新计算下次执行时间
	task.UpdateNextRunTime()

	if err := tss.db.Save(&task).Error; err != nil {
		return nil, fmt.Errorf("更新同步任务失败: %w", err)
	}

	return &task, nil
}

// GetSyncTask 获取同步任务
func (tss *ThematicSyncService) GetSyncTask(ctx context.Context, taskID string) (*models.ThematicSyncTask, error) {
	var task models.ThematicSyncTask
	if err := tss.db.Preload("ThematicLibrary").Preload("ThematicInterface").
		First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取同步任务失败: %w", err)
	}

	return &task, nil
}

// ListSyncTasks 获取同步任务列表
func (tss *ThematicSyncService) ListSyncTasks(ctx context.Context, req *ListSyncTasksRequest) (*ListSyncTasksResponse, error) {
	query := tss.db.Model(&models.ThematicSyncTask{})

	// 添加过滤条件
	if req.ThematicLibraryID != "" {
		query = query.Where("thematic_library_id = ?", req.ThematicLibraryID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.TriggerType != "" {
		query = query.Where("trigger_type = ?", req.TriggerType)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("获取任务总数失败: %w", err)
	}

	// 分页查询
	var tasks []models.ThematicSyncTask
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("ThematicLibrary").Preload("ThematicInterface").
		Order("created_at DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err)
	}

	return &ListSyncTasksResponse{
		Tasks:    tasks,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteSyncTask 删除同步任务
func (tss *ThematicSyncService) DeleteSyncTask(ctx context.Context, taskID string) error {
	// 检查任务是否存在
	var task models.ThematicSyncTask
	if err := tss.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("获取同步任务失败: %w", err)
	}

	// 检查任务状态，运行中的任务不能删除
	if task.Status == "running" {
		return fmt.Errorf("运行中的任务不能删除")
	}

	// 开启事务删除任务和相关记录
	return tss.db.Transaction(func(tx *gorm.DB) error {
		// 删除执行记录
		if err := tx.Where("task_id = ?", taskID).Delete(&models.ThematicSyncExecution{}).Error; err != nil {
			return fmt.Errorf("删除执行记录失败: %w", err)
		}

		// 删除任务
		if err := tx.Delete(&task).Error; err != nil {
			return fmt.Errorf("删除同步任务失败: %w", err)
		}

		return nil
	})
}

// ExecuteSyncTask 执行同步任务
func (tss *ThematicSyncService) ExecuteSyncTask(ctx context.Context, taskID string, req *ExecuteSyncTaskRequest) (*thematic_sync.SyncResponse, error) {
	// 获取任务信息
	task, err := tss.GetSyncTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 检查任务状态
	if !task.CanStart() {
		return nil, fmt.Errorf("任务状态不允许执行: %s", task.Status)
	}

	// 解析源库和接口列表
	var sourceLibraries []string
	if len(task.SourceLibraries) > 0 {
		// 简化处理：假设存储的是JSON格式的字符串数组
		// 实际项目中应该使用json.Unmarshal
		sourceLibraries = []string{} // 临时处理
	}

	var sourceInterfaces []string
	if len(task.SourceInterfaces) > 0 {
		// 简化处理：假设存储的是JSON格式的字符串数组
		// 实际项目中应该使用json.Unmarshal
		sourceInterfaces = []string{} // 临时处理
	}

	// 构建同步请求
	var configMap map[string]interface{}
	if req.Options != nil {
		// 将结构化配置转换为 map[string]interface{} 以兼容现有的同步引擎
		configBytes, _ := json.Marshal(req.Options)
		json.Unmarshal(configBytes, &configMap)
	}

	syncRequest := &thematic_sync.SyncRequest{
		TaskID:            taskID,
		ExecutionType:     req.ExecutionType,
		SourceLibraries:   sourceLibraries,
		SourceInterfaces:  sourceInterfaces,
		TargetLibraryID:   task.ThematicLibraryID,
		TargetInterfaceID: task.ThematicInterfaceID,
		Config:            configMap,
		Context:           ctx,
	}

	// 设置进度回调
	tss.syncEngine.SetProgressCallback(func(progress *thematic_sync.SyncProgress) {
		// 这里可以实现进度通知逻辑，比如通过WebSocket推送给前端
		// 或者存储到缓存中供查询
	})

	// 执行同步
	return tss.syncEngine.ExecuteSync(syncRequest)
}

// GetSyncExecution 获取同步执行记录
func (tss *ThematicSyncService) GetSyncExecution(ctx context.Context, executionID string) (*models.ThematicSyncExecution, error) {
	var execution models.ThematicSyncExecution
	if err := tss.db.Preload("Task").First(&execution, "id = ?", executionID).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}

	return &execution, nil
}

// ListSyncExecutions 获取同步执行记录列表
func (tss *ThematicSyncService) ListSyncExecutions(ctx context.Context, req *ListSyncExecutionsRequest) (*ListSyncExecutionsResponse, error) {
	query := tss.db.Model(&models.ThematicSyncExecution{})

	// 添加过滤条件
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

	// 分页查询
	var executions []models.ThematicSyncExecution
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("Task").
		Order("created_at DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录列表失败: %w", err)
	}

	return &ListSyncExecutionsResponse{
		Executions: executions,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// GetSyncTaskStatistics 获取同步任务统计信息
func (tss *ThematicSyncService) GetSyncTaskStatistics(ctx context.Context, taskID string) (*ThematicSyncTaskStatistics, error) {
	// 获取任务基本信息
	task, err := tss.GetSyncTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 获取执行统计
	var executionStats struct {
		TotalExecutions    int64   `json:"total_executions"`
		SuccessExecutions  int64   `json:"success_executions"`
		FailedExecutions   int64   `json:"failed_executions"`
		AverageProcessTime float64 `json:"average_process_time"`
		TotalProcessedRows int64   `json:"total_processed_rows"`
	}

	err = tss.db.Model(&models.ThematicSyncExecution{}).
		Select(`
			COUNT(*) as total_executions,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success_executions,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_executions,
			AVG(duration) as average_process_time,
			SUM(processed_record_count) as total_processed_rows
		`).
		Where("task_id = ?", taskID).
		Scan(&executionStats).Error

	if err != nil {
		return nil, fmt.Errorf("获取执行统计失败: %w", err)
	}

	// 获取最近执行记录
	var recentExecutions []models.ThematicSyncExecution
	err = tss.db.Where("task_id = ?", taskID).
		Order("created_at DESC").
		Limit(10).
		Find(&recentExecutions).Error

	if err != nil {
		return nil, fmt.Errorf("获取最近执行记录失败: %w", err)
	}

	statistics := &ThematicSyncTaskStatistics{
		Task:               task,
		TotalExecutions:    executionStats.TotalExecutions,
		SuccessExecutions:  executionStats.SuccessExecutions,
		FailedExecutions:   executionStats.FailedExecutions,
		SuccessRate:        0,
		AverageProcessTime: int64(executionStats.AverageProcessTime),
		TotalProcessedRows: executionStats.TotalProcessedRows,
		RecentExecutions:   recentExecutions,
	}

	// 计算成功率
	if statistics.TotalExecutions > 0 {
		statistics.SuccessRate = float64(statistics.SuccessExecutions) / float64(statistics.TotalExecutions) * 100
	}

	return statistics, nil
}

// StopSyncTask 停止同步任务（新增方法）
func (tss *ThematicSyncService) StopSyncTask(ctx context.Context, taskID string) error {
	// 获取任务信息
	task, err := tss.GetSyncTask(ctx, taskID)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.Status != "running" {
		return fmt.Errorf("任务状态不允许停止: %s", task.Status)
	}

	// 更新任务状态为停止
	updates := map[string]interface{}{
		"status":     "stopped",
		"updated_at": time.Now(),
	}

	if err := tss.db.Model(&models.ThematicSyncTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return fmt.Errorf("停止同步任务失败: %w", err)
	}

	return nil
}

// GetSyncTaskStatus 获取同步任务状态（新增方法）
func (tss *ThematicSyncService) GetSyncTaskStatus(ctx context.Context, taskID string) (map[string]interface{}, error) {
	task, err := tss.GetSyncTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 获取最近的执行记录
	var lastExecution models.ThematicSyncExecution
	err = tss.db.Where("task_id = ?", taskID).Order("created_at DESC").First(&lastExecution).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}

	status := map[string]interface{}{
		"task_id":          task.ID,
		"task_name":        task.TaskName,
		"status":           task.Status,
		"trigger_type":     task.TriggerType,
		"last_executed_at": task.LastSyncTime,
		"next_run_time":    task.NextRunTime,
		"created_at":       task.CreatedAt,
		"updated_at":       task.UpdatedAt,
	}

	if err != gorm.ErrRecordNotFound {
		status["last_execution"] = map[string]interface{}{
			"execution_id":   lastExecution.ID,
			"status":         lastExecution.Status,
			"started_at":     lastExecution.StartTime,
			"completed_at":   lastExecution.EndTime,
			"records_synced": lastExecution.ProcessedRecordCount,
			"error_message":  lastExecution.ErrorDetails,
		}
	}

	return status, nil
}

// GetSyncTaskExecutions 获取同步任务的执行记录列表（新增方法）
func (tss *ThematicSyncService) GetSyncTaskExecutions(ctx context.Context, taskID string, page, size int, status string) ([]interface{}, int64, error) {
	var executions []models.ThematicSyncExecution
	var total int64

	query := tss.db.Model(&models.ThematicSyncExecution{}).Where("task_id = ?", taskID)

	// 添加状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取执行记录总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&executions).Error; err != nil {
		return nil, 0, fmt.Errorf("获取执行记录列表失败: %w", err)
	}

	// 转换为interface{}数组
	result := make([]interface{}, len(executions))
	for i, exec := range executions {
		result[i] = exec
	}

	return result, total, nil
}
