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
		DataSourceSQL:       structToJSONB(req.DataSourceSQL), // 新增SQL数据源配置
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

	// 解析源库配置
	var sourceLibraryConfigs []thematic_sync.SourceLibraryConfig
	if len(task.SourceLibraries) > 0 {
		// 从JSONB中解析源库配置
		var sourceLibrariesRaw []interface{}
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.SourceLibraries)), &sourceLibrariesRaw); err == nil {
			for _, configRaw := range sourceLibrariesRaw {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := thematic_sync.SourceLibraryConfig{
						LibraryID:   getStringFromMap(configMap, "library_id"),
						InterfaceID: getStringFromMap(configMap, "interface_id"),
						SQLQuery:    getStringFromMap(configMap, "sql_query"),
					}

					// 解析参数
					if params, exists := configMap["parameters"]; exists {
						if paramsMap, ok := params.(map[string]interface{}); ok {
							config.Parameters = paramsMap
						}
					}

					// 解析过滤器
					if filters, exists := configMap["filters"]; exists {
						if filtersSlice, ok := filters.([]interface{}); ok {
							for _, filterRaw := range filtersSlice {
								if filterMap, ok := filterRaw.(map[string]interface{}); ok {
									filter := thematic_sync.FilterConfig{
										Field:    getStringFromMap(filterMap, "field"),
										Operator: getStringFromMap(filterMap, "operator"),
										Value:    filterMap["value"],
										LogicOp:  getStringFromMap(filterMap, "logic_op"),
									}
									config.Filters = append(config.Filters, filter)
								}
							}
						}
					}

					// 解析转换配置
					if transforms, exists := configMap["transforms"]; exists {
						if transformsSlice, ok := transforms.([]interface{}); ok {
							for _, transformRaw := range transformsSlice {
								if transformMap, ok := transformRaw.(map[string]interface{}); ok {
									transform := thematic_sync.TransformConfig{
										SourceField: getStringFromMap(transformMap, "source_field"),
										TargetField: getStringFromMap(transformMap, "target_field"),
										Transform:   getStringFromMap(transformMap, "transform"),
									}

									if config, exists := transformMap["config"]; exists {
										if configMap, ok := config.(map[string]interface{}); ok {
											transform.Config = configMap
										}
									}

									config.Transforms = append(config.Transforms, transform)
								}
							}
						}
					}

					sourceLibraryConfigs = append(sourceLibraryConfigs, config)
				}
			}
		}
	}

	// 兜底：使用接口列表（如果存在）
	var sourceInterfaces []string
	if len(task.SourceInterfaces) > 0 {
		// 从JSONB中解析接口列表
		var interfacesRaw []interface{}
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.SourceInterfaces)), &interfacesRaw); err == nil {
			for _, interfaceRaw := range interfacesRaw {
				sourceInterfaces = append(sourceInterfaces, fmt.Sprintf("%v", interfaceRaw))
			}
		}
	}

	// 解析SQL数据源配置（优先级更高）
	var sqlDataSourceConfigs []thematic_sync.SQLDataSourceConfig
	if len(task.DataSourceSQL) > 0 {
		var sqlConfigsRaw []interface{}
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.DataSourceSQL)), &sqlConfigsRaw); err == nil {
			for _, configRaw := range sqlConfigsRaw {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := thematic_sync.SQLDataSourceConfig{
						LibraryID:   getStringFromMap(configMap, "library_id"),
						InterfaceID: getStringFromMap(configMap, "interface_id"),
						SQLQuery:    getStringFromMap(configMap, "sql_query"),
						Timeout:     30,    // 默认30秒
						MaxRows:     10000, // 默认1万行
					}

					// 解析参数
					if params, exists := configMap["parameters"]; exists {
						if paramsMap, ok := params.(map[string]interface{}); ok {
							config.Parameters = paramsMap
						}
					}

					// 解析超时时间
					if timeout, exists := configMap["timeout"]; exists {
						if timeoutFloat, ok := timeout.(float64); ok {
							config.Timeout = int(timeoutFloat)
						} else if timeoutInt, ok := timeout.(int); ok {
							config.Timeout = timeoutInt
						}
					}

					// 解析最大行数
					if maxRows, exists := configMap["max_rows"]; exists {
						if maxRowsFloat, ok := maxRows.(float64); ok {
							config.MaxRows = int(maxRowsFloat)
						} else if maxRowsInt, ok := maxRows.(int); ok {
							config.MaxRows = maxRowsInt
						}
					}

					sqlDataSourceConfigs = append(sqlDataSourceConfigs, config)
				}
			}
		}
	}

	// 构建同步请求配置
	configMap := make(map[string]interface{})

	// 优先添加SQL数据源配置
	if len(sqlDataSourceConfigs) > 0 {
		configMap["data_source_sql"] = sqlDataSourceConfigs
	} else if len(sourceLibraryConfigs) > 0 {
		// 兜底：添加传统源库配置
		configMap["source_libraries"] = sourceLibraryConfigs
	}

	// 添加各种规则配置
	if len(task.AggregationConfig) > 0 {
		var aggConfig AggregationConfig
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.AggregationConfig)), &aggConfig); err == nil {
			configMap["aggregation_config"] = aggConfig
		}
	}

	if len(task.KeyMatchingRules) > 0 {
		var keyRules KeyMatchingRules
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.KeyMatchingRules)), &keyRules); err == nil {
			configMap["key_matching_rules"] = keyRules
		}
	}

	if len(task.FieldMappingRules) > 0 {
		var fieldRules FieldMappingRules
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.FieldMappingRules)), &fieldRules); err == nil {
			configMap["field_mapping_rules"] = fieldRules
		}
	}

	if len(task.CleansingRules) > 0 {
		var cleansingRules CleansingRules
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.CleansingRules)), &cleansingRules); err == nil {
			configMap["cleansing_rules"] = cleansingRules
		}
	}

	if len(task.PrivacyRules) > 0 {
		var privacyRules PrivacyRules
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.PrivacyRules)), &privacyRules); err == nil {
			configMap["privacy_rules"] = privacyRules
		}
	}

	if len(task.QualityRules) > 0 {
		var qualityRules QualityRules
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.QualityRules)), &qualityRules); err == nil {
			configMap["quality_rules"] = qualityRules
		}
	}

	// 添加执行选项
	if req.Options != nil {
		optionsBytes, _ := json.Marshal(req.Options)
		var optionsMap map[string]interface{}
		if err := json.Unmarshal(optionsBytes, &optionsMap); err == nil {
			for key, value := range optionsMap {
				configMap[key] = value
			}
		}
	}

	// 构建源库和接口列表（兜底处理）
	var sourceLibraries []string
	var finalSourceInterfaces []string

	if len(sourceLibraryConfigs) > 0 {
		for _, config := range sourceLibraryConfigs {
			sourceLibraries = append(sourceLibraries, config.LibraryID)
			finalSourceInterfaces = append(finalSourceInterfaces, config.InterfaceID)
		}
	} else if len(sourceInterfaces) > 0 {
		finalSourceInterfaces = sourceInterfaces
		// 如果没有明确的库ID，使用空字符串
		for range sourceInterfaces {
			sourceLibraries = append(sourceLibraries, "")
		}
	}

	syncRequest := &thematic_sync.SyncRequest{
		TaskID:            taskID,
		ExecutionType:     req.ExecutionType,
		SourceLibraries:   sourceLibraries,
		SourceInterfaces:  finalSourceInterfaces,
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

// getStringFromMap 从map中获取字符串值
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}
