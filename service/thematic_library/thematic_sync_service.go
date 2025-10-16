/*
 * @module service/thematic_sync_service
 * @description 主题同步服务，提供主题数据同步的业务逻辑和服务接口，集成数据治理功能
 * @architecture 服务层 - 封装业务逻辑，提供统一的服务接口，集成数据治理
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 任务管理 -> 同步执行 -> 数据治理处理 -> 状态跟踪 -> 结果处理
 * @rules 确保业务逻辑的完整性和一致性，支持事务性操作，集成数据治理规则
 * @dependencies gorm.io/gorm, context, service/models, service/governance
 * @refs service/thematic_sync/sync_engine.go, service/models/thematic_sync.go, governance_integration.go
 */

package thematic_library

import (
	"context"
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"datahub-service/service/thematic_library/thematic_sync"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// GovernanceIntegrationAdapter 适配器，用于解决类型不匹配问题
type GovernanceIntegrationAdapter struct {
	service *GovernanceIntegrationService
}

// ApplyGovernanceRules 适配器方法，转换类型并调用实际服务
func (adapter *GovernanceIntegrationAdapter) ApplyGovernanceRules(
	ctx context.Context,
	records []map[string]interface{},
	task *models.ThematicSyncTask,
	config *thematic_sync.GovernanceExecutionConfig,
) ([]map[string]interface{}, *thematic_sync.GovernanceExecutionResult, error) {

	// 类型转换：从 thematic_sync 包的类型转换为 thematic_library 包的类型
	libraryConfig := &GovernanceExecutionConfig{
		EnableQualityCheck:   config.EnableQualityCheck,
		EnableCleansing:      config.EnableCleansing,
		EnableMasking:        config.EnableMasking,
		StopOnQualityFailure: config.StopOnQualityFailure,
		QualityThreshold:     config.QualityThreshold,
		BatchSize:            config.BatchSize,
		MaxRetries:           config.MaxRetries,
		TimeoutSeconds:       config.TimeoutSeconds,
		CustomConfig:         config.CustomConfig,
	}

	// 调用实际服务
	processedRecords, libraryResult, err := adapter.service.ApplyGovernanceRules(ctx, records, task, libraryConfig)
	if err != nil {
		return nil, nil, err
	}

	// 类型转换：从 thematic_library 包的类型转换为 thematic_sync 包的类型
	syncResult := &thematic_sync.GovernanceExecutionResult{
		OverallQualityScore:   libraryResult.OverallQualityScore,
		TotalProcessedRecords: libraryResult.TotalProcessedRecords,
		TotalCleansingApplied: libraryResult.TotalCleansingApplied,
		TotalMaskingApplied:   libraryResult.TotalMaskingApplied,
		TotalValidationErrors: libraryResult.TotalValidationErrors,
		ExecutionTime:         libraryResult.ExecutionTime,
		ComplianceStatus:      libraryResult.ComplianceStatus,
	}

	return processedRecords, syncResult, nil
}

// ThematicSyncService 主题同步服务
type ThematicSyncService struct {
	db                           *gorm.DB
	syncEngine                   *thematic_sync.ThematicSyncEngine
	governanceService            *governance.GovernanceService
	governanceIntegrationService *GovernanceIntegrationService
	// 调度器相关字段
	cron             *cron.Cron
	intervalTicker   *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
	schedulerStarted bool
	// 分布式锁
	distributedLock interface {
		TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
		Unlock(ctx context.Context, key string) error
	}
}

// NewThematicSyncService 创建主题同步服务 - 简化版本
func NewThematicSyncService(db *gorm.DB,
	governanceService *governance.GovernanceService) *ThematicSyncService {

	governanceIntegrationService := NewGovernanceIntegrationService(db, governanceService)
	// 创建适配器来解决类型不匹配问题
	governanceAdapter := &GovernanceIntegrationAdapter{service: governanceIntegrationService}
	syncEngine := thematic_sync.NewThematicSyncEngine(db, governanceAdapter)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建带秒级的cron调度器
	c := cron.New(cron.WithSeconds())

	return &ThematicSyncService{
		db:                           db,
		syncEngine:                   syncEngine,
		governanceService:            governanceService,
		governanceIntegrationService: governanceIntegrationService,
		cron:                         c,
		ctx:                          ctx,
		cancel:                       cancel,
		schedulerStarted:             false,
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

// structToJSONBGenericArray 将结构体转换为 JSONBGenericArray
func structToJSONBGenericArray(v interface{}) models.JSONBGenericArray {
	if v == nil {
		return models.JSONBGenericArray{}
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return models.JSONBGenericArray{}
	}

	var result models.JSONBGenericArray
	if err := json.Unmarshal(bytes, &result); err != nil {
		return models.JSONBGenericArray{}
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
		SourceLibraries:     structToJSONBGenericArray(req.SourceLibraries),
		SQLQueries:          structToJSONBGenericArray(req.SQLQueries), // SQL查询配置
		KeyMatchingRules:    structToJSONB(req.KeyMatchingRules),
		FieldMappingRules:   structToJSONB(req.FieldMappingRules),

		// 数据治理规则配置
		QualityRuleConfigs:   structToJSONBGenericArray(req.QualityRuleConfigs),
		CleansingRuleConfigs: structToJSONBGenericArray(req.CleansingRuleConfigs),
		MaskingRuleConfigs:   structToJSONBGenericArray(req.MaskingRuleConfigs),
		GovernanceConfig:     structToJSONB(req.GovernanceConfig),

		Status:    "draft",
		CreatedAt: time.Now(),
		CreatedBy: req.CreatedBy,
		UpdatedAt: time.Now(),
		UpdatedBy: req.CreatedBy,
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
	if req.KeyMatchingRules != nil {
		task.KeyMatchingRules = structToJSONB(req.KeyMatchingRules)
	}
	if req.FieldMappingRules != nil {
		task.FieldMappingRules = structToJSONB(req.FieldMappingRules)
	}

	// 更新数据治理规则配置
	if req.QualityRuleConfigs != nil {
		task.QualityRuleConfigs = structToJSONBGenericArray(req.QualityRuleConfigs)
	}
	if req.CleansingRuleConfigs != nil {
		task.CleansingRuleConfigs = structToJSONBGenericArray(req.CleansingRuleConfigs)
	}
	if req.MaskingRuleConfigs != nil {
		task.MaskingRuleConfigs = structToJSONBGenericArray(req.MaskingRuleConfigs)
	}
	if req.GovernanceConfig != nil {
		task.GovernanceConfig = structToJSONB(req.GovernanceConfig)
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
		// 从JSONBGenericArray中解析源库配置
		sourceLibrariesRaw := []interface{}(task.SourceLibraries)
		if len(sourceLibrariesRaw) > 0 {
			for _, configRaw := range sourceLibrariesRaw {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					libraryID := getStringFromMap(configMap, "library_id")

					// 处理嵌套的interfaces结构
					var interfaceConfigs []thematic_sync.SourceInterfaceConfig
					if interfacesRaw, exists := configMap["interfaces"]; exists {
						if interfacesSlice, ok := interfacesRaw.([]interface{}); ok {
							for _, interfaceRaw := range interfacesSlice {
								if interfaceMap, ok := interfaceRaw.(map[string]interface{}); ok {
									interfaceConfig := thematic_sync.SourceInterfaceConfig{
										InterfaceID: getStringFromMap(interfaceMap, "interface_id"),
									}

									// 解析增量配置
									if incrementalRaw, exists := interfaceMap["incremental_config"]; exists {
										if incrementalMap, ok := incrementalRaw.(map[string]interface{}); ok {
											incrementalConfig := &thematic_sync.IncrementalConfig{
												Enabled:            getBoolFromMap(incrementalMap, "enabled"),
												IncrementalField:   getStringFromMap(incrementalMap, "incremental_field"),
												FieldType:          getStringFromMap(incrementalMap, "field_type"),
												CompareOperator:    getStringFromMap(incrementalMap, "compare_operator"),
												LastSyncValue:      getStringFromMap(incrementalMap, "last_sync_value"),
												InitialValue:       getStringFromMap(incrementalMap, "initial_value"),
												MaxLookbackHours:   getIntFromMap(incrementalMap, "max_lookback_hours"),
												CheckDeletedField:  getStringFromMap(incrementalMap, "check_deleted_field"),
												DeletedValue:       getStringFromMap(incrementalMap, "deleted_value"),
												BatchSize:          getIntFromMap(incrementalMap, "batch_size"),
												SyncDeletedRecords: getBoolFromMap(incrementalMap, "sync_deleted_records"),
												TimestampFormat:    getStringFromMap(incrementalMap, "timestamp_format"),
												TimeZone:           getStringFromMap(incrementalMap, "timezone"),
											}

											// 设置默认值
											if incrementalConfig.CompareOperator == "" {
												incrementalConfig.CompareOperator = ">"
											}
											if incrementalConfig.BatchSize == 0 {
												incrementalConfig.BatchSize = 1000
											}
											if incrementalConfig.TimeZone == "" {
												incrementalConfig.TimeZone = "Asia/Shanghai"
											}

											interfaceConfig.IncrementalConfig = incrementalConfig
										}
									}

									interfaceConfigs = append(interfaceConfigs, interfaceConfig)
								}
							}
						}
					}

					// 如果没有找到嵌套结构，尝试直接获取interface_id（向前兼容）
					if len(interfaceConfigs) == 0 {
						if interfaceID := getStringFromMap(configMap, "interface_id"); interfaceID != "" {
							interfaceConfig := thematic_sync.SourceInterfaceConfig{
								InterfaceID: interfaceID,
							}
							interfaceConfigs = append(interfaceConfigs, interfaceConfig)
						}
					}

					// 为每个接口创建一个配置
					for _, interfaceConfig := range interfaceConfigs {
						config := thematic_sync.SourceLibraryConfig{
							LibraryID:         libraryID,
							InterfaceID:       interfaceConfig.InterfaceID,
							SQLQuery:          getStringFromMap(configMap, "sql_query"),
							IncrementalConfig: interfaceConfig.IncrementalConfig,
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

										if transformConfig, exists := transformMap["config"]; exists {
											if transformConfigMap, ok := transformConfig.(map[string]interface{}); ok {
												transform.Config = transformConfigMap
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
	}

	// 注意：已移除 SourceInterfaces 字段，所有源接口信息现在都在 SourceLibraries 中

	// 解析SQL查询配置（优先级更高）
	var sqlQueryConfigs []*thematic_sync.SQLQueryConfig
	if len(task.SQLQueries) > 0 {
		sqlConfigsRaw := []interface{}(task.SQLQueries)
		if len(sqlConfigsRaw) > 0 {
			for _, configRaw := range sqlConfigsRaw {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := &thematic_sync.SQLQueryConfig{
						SQLQuery: getStringFromMap(configMap, "sql_query"),
						Timeout:  30,    // 默认30秒
						MaxRows:  10000, // 默认1万行
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

					// 只添加有效的SQL查询
					if config.SQLQuery != "" {
						sqlQueryConfigs = append(sqlQueryConfigs, config)
					}
				}
			}
		}
	}

	// 构建同步请求配置
	configMap := make(map[string]interface{})

	// 优先添加SQL查询配置（SQL模式）
	if len(sqlQueryConfigs) > 0 {
		configMap["sql_queries"] = sqlQueryConfigs
		fmt.Printf("[DEBUG] 任务使用SQL查询模式，查询数量: %d\n", len(sqlQueryConfigs))
	} else if len(sourceLibraryConfigs) > 0 {
		// 使用接口模式
		configMap["source_libraries"] = sourceLibraryConfigs
		fmt.Printf("[DEBUG] 任务使用接口模式，源库数量: %d\n", len(sourceLibraryConfigs))
	} else {
		return nil, fmt.Errorf("任务必须配置数据源：请配置 SQLQueries(SQL模式) 或 SourceLibraries(接口模式)")
	}

	// 添加各种规则配置
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

	// 添加数据治理规则配置
	if len(task.QualityRuleConfigs) > 0 {
		var qualityRuleConfigs []models.QualityRuleConfig
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.QualityRuleConfigs)), &qualityRuleConfigs); err == nil {
			configMap["quality_rule_configs"] = qualityRuleConfigs
		}
	}

	if len(task.CleansingRuleConfigs) > 0 {
		var cleansingRuleConfigs []models.DataCleansingConfig
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.CleansingRuleConfigs)), &cleansingRuleConfigs); err == nil {
			configMap["cleansing_rule_configs"] = cleansingRuleConfigs
		}
	}

	if len(task.MaskingRuleConfigs) > 0 {
		var maskingRuleConfigs []models.DataMaskingConfig
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.MaskingRuleConfigs)), &maskingRuleConfigs); err == nil {
			configMap["masking_rule_configs"] = maskingRuleConfigs
		}
	}

	if len(task.GovernanceConfig) > 0 {
		var governanceConfig GovernanceExecutionConfig
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", task.GovernanceConfig)), &governanceConfig); err == nil {
			configMap["governance_config"] = governanceConfig
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

	// 构建源库和接口列表
	var sourceLibraries []string
	var finalSourceInterfaces []string

	for _, config := range sourceLibraryConfigs {
		sourceLibraries = append(sourceLibraries, config.LibraryID)
		finalSourceInterfaces = append(finalSourceInterfaces, config.InterfaceID)
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

// getBoolFromMap 从map中获取布尔值
func getBoolFromMap(m map[string]interface{}, key string) bool {
	if value, exists := m[key]; exists {
		if boolVal, ok := value.(bool); ok {
			return boolVal
		}
		// 尝试从字符串转换
		strVal := fmt.Sprintf("%v", value)
		return strVal == "true" || strVal == "1" || strVal == "yes"
	}
	return false
}

// getIntFromMap 从map中获取整数值
func getIntFromMap(m map[string]interface{}, key string) int {
	if value, exists := m[key]; exists {
		if intVal, ok := value.(int); ok {
			return intVal
		}
		if floatVal, ok := value.(float64); ok {
			return int(floatVal)
		}
		// 尝试从字符串转换
		strVal := fmt.Sprintf("%v", value)
		if intVal, err := fmt.Sscanf(strVal, "%d"); err == nil && intVal == 1 {
			var result int
			fmt.Sscanf(strVal, "%d", &result)
			return result
		}
	}
	return 0
}

// ======================== 调度器功能实现 ========================

// SetDistributedLock 设置分布式锁
func (tss *ThematicSyncService) SetDistributedLock(lock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}) {
	tss.distributedLock = lock
	if lock != nil {
		log.Println("主题同步任务服务已启用分布式锁")
	}
}

// StartScheduler 启动主题同步任务调度器
func (tss *ThematicSyncService) StartScheduler() error {
	if tss.schedulerStarted {
		return fmt.Errorf("调度器已经启动")
	}

	log.Println("启动主题库同步任务调度器")

	// 启动cron调度器
	tss.cron.Start()

	// 启动间隔任务检查器（每分钟检查一次）
	tss.intervalTicker = time.NewTicker(1 * time.Minute)
	go tss.runIntervalChecker()

	// 加载现有的调度任务
	if err := tss.loadScheduledTasks(); err != nil {
		log.Printf("加载主题调度任务失败: %v", err)
		return err
	}

	tss.schedulerStarted = true
	log.Println("主题库同步任务调度器启动完成")
	return nil
}

// StopScheduler 停止调度器
func (tss *ThematicSyncService) StopScheduler() {
	if !tss.schedulerStarted {
		return
	}

	log.Println("停止主题库同步任务调度器")

	tss.cancel()

	if tss.cron != nil {
		tss.cron.Stop()
	}

	if tss.intervalTicker != nil {
		tss.intervalTicker.Stop()
	}

	tss.schedulerStarted = false
	log.Println("主题库同步任务调度器已停止")
}

// loadScheduledTasks 加载调度任务
func (tss *ThematicSyncService) loadScheduledTasks() error {
	// 获取所有待执行的调度任务
	tasks, err := tss.getScheduledTasks(tss.ctx)
	if err != nil {
		return fmt.Errorf("获取调度任务失败: %w", err)
	}

	for _, task := range tasks {
		if err := tss.addTaskToScheduler(&task); err != nil {
			log.Printf("添加任务到调度器失败 [%s]: %v", task.ID, err)
		}
	}

	log.Printf("加载了 %d 个主题同步调度任务", len(tasks))
	return nil
}

// getScheduledTasks 获取需要执行的调度任务
func (tss *ThematicSyncService) getScheduledTasks(ctx context.Context) ([]models.ThematicSyncTask, error) {
	var tasks []models.ThematicSyncTask
	now := time.Now()

	// 查找状态为active且下次执行时间已到的主题任务
	err := tss.db.Where("status = ? AND next_run_time IS NOT NULL AND next_run_time <= ?",
		"active", now).Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("获取调度任务失败: %w", err)
	}

	return tasks, nil
}

// addTaskToScheduler 添加任务到调度器
func (tss *ThematicSyncService) addTaskToScheduler(task *models.ThematicSyncTask) error {
	switch task.TriggerType {
	case "cron":
		if task.CronExpression == "" {
			return fmt.Errorf("Cron任务缺少表达式")
		}

		_, err := tss.cron.AddFunc(task.CronExpression, func() {
			tss.executeScheduledTask(task.ID)
		})
		if err != nil {
			return fmt.Errorf("添加Cron任务失败: %w", err)
		}

		log.Printf("添加主题Cron任务: %s [%s]", task.ID, task.CronExpression)

	case "once":
		if task.ScheduledTime != nil && task.ScheduledTime.After(time.Now()) {
			go func() {
				timer := time.NewTimer(time.Until(*task.ScheduledTime))
				defer timer.Stop()

				select {
				case <-timer.C:
					tss.executeScheduledTask(task.ID)
				case <-tss.ctx.Done():
					return
				}
			}()

			log.Printf("添加主题单次任务: %s [%s]", task.ID, task.ScheduledTime.Format("2006-01-02 15:04:05"))
		}

	case "interval":
		// 间隔任务由intervalChecker处理
		log.Printf("添加主题间隔任务: %s [%d秒]", task.ID, task.IntervalSeconds)
	}

	return nil
}

// runIntervalChecker 运行间隔任务检查器
func (tss *ThematicSyncService) runIntervalChecker() {
	for {
		select {
		case <-tss.intervalTicker.C:
			tss.checkIntervalTasks()
		case <-tss.ctx.Done():
			return
		}
	}
}

// checkIntervalTasks 检查间隔任务
func (tss *ThematicSyncService) checkIntervalTasks() {
	tasks, err := tss.getScheduledTasks(tss.ctx)
	if err != nil {
		log.Printf("获取间隔任务失败: %v", err)
		return
	}

	for _, task := range tasks {
		if task.TriggerType == "interval" && task.ShouldExecuteNow() {
			go tss.executeScheduledTask(task.ID)
		}
	}
}

// executeScheduledTask 执行调度任务（带分布式锁）
func (tss *ThematicSyncService) executeScheduledTask(taskID string) {
	log.Printf("执行主题调度任务: %s", taskID)

	// 如果有分布式锁，使用锁保护执行
	if tss.distributedLock != nil {
		lockKey := fmt.Sprintf("thematic_library:%s", taskID)
		lockTTL := 30 * time.Minute // 主题同步可能耗时较长，设置30分钟

		// 尝试获取锁
		locked, err := tss.distributedLock.TryLock(tss.ctx, lockKey, lockTTL)
		if err != nil {
			log.Printf("获取分布式锁失败 [%s]: %v", taskID, err)
			return
		}

		if !locked {
			log.Printf("任务正在其他实例执行，跳过 [%s]", taskID)
			return
		}

		// 确保执行完毕后释放锁
		defer func() {
			if unlockErr := tss.distributedLock.Unlock(tss.ctx, lockKey); unlockErr != nil {
				log.Printf("释放分布式锁失败 [%s]: %v", taskID, unlockErr)
			}
		}()
	}

	// 获取任务详情
	task, err := tss.GetSyncTask(tss.ctx, taskID)
	if err != nil {
		log.Printf("获取任务失败 [%s]: %v", taskID, err)
		return
	}

	// 检查任务是否可以执行
	if !task.CanStart() {
		log.Printf("任务不能执行 [%s]: 状态=%s", taskID, task.Status)
		return
	}

	// 构建执行请求
	req := &ExecuteSyncTaskRequest{
		ExecutionType: "scheduled",
		Options:       nil,
	}

	// 执行同步任务
	_, err = tss.ExecuteSyncTask(tss.ctx, taskID, req)
	if err != nil {
		log.Printf("执行调度任务失败 [%s]: %v", taskID, err)
		return
	}

	// 更新下次执行时间
	task.UpdateNextRunTime()
	if err := tss.db.Model(&models.ThematicSyncTask{}).Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"next_run_time":  task.NextRunTime,
			"last_sync_time": time.Now(),
			"updated_at":     time.Now(),
		}).Error; err != nil {
		log.Printf("更新任务执行时间失败 [%s]: %v", taskID, err)
	}

	log.Printf("主题调度任务执行完成 [%s]", taskID)
}

// AddScheduledTask 添加调度任务
func (tss *ThematicSyncService) AddScheduledTask(task *models.ThematicSyncTask) error {
	return tss.addTaskToScheduler(task)
}

// RemoveScheduledTask 移除调度任务
func (tss *ThematicSyncService) RemoveScheduledTask(taskID string) error {
	// 由于cron库不支持按ID移除任务，这里我们重新加载所有任务
	tss.cron.Stop()
	tss.cron = cron.New(cron.WithSeconds())
	tss.cron.Start()

	return tss.loadScheduledTasks()
}

// ReloadScheduledTasks 重新加载调度任务
func (tss *ThematicSyncService) ReloadScheduledTasks() error {
	tss.cron.Stop()
	tss.cron = cron.New(cron.WithSeconds())
	tss.cron.Start()

	return tss.loadScheduledTasks()
}
