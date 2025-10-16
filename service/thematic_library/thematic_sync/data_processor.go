/*
 * @module service/thematic_sync/data_processor
 * @description 数据处理器，负责数据合并、治理和质量处理
 * @architecture 管道模式 - 通过多个处理阶段完成数据处理
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 数据合并 -> 数据治理 -> 质量检查 -> 处理结果
 * @rules 确保数据处理的完整性和一致性
 * @dependencies gorm.io/gorm, fmt, time, context
 * @refs sync_types.go, models/thematic_sync.go
 */

package thematic_sync

import (
	"datahub-service/service/models"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DataProcessor 数据处理器
type DataProcessor struct {
	db                    *gorm.DB
	governanceIntegration GovernanceIntegrationServiceInterface
	fieldMapper           *FieldMapper
}

// NewDataProcessor 创建数据处理器
func NewDataProcessor(db *gorm.DB, governanceIntegration GovernanceIntegrationServiceInterface) *DataProcessor {
	return &DataProcessor{
		db:                    db,
		governanceIntegration: governanceIntegration,
		fieldMapper:           NewFieldMapper(db),
	}
}

// ProcessData 处理数据（合并+治理）
func (dp *DataProcessor) ProcessData(sourceRecords []SourceRecordInfo, request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, *GovernanceExecutionResult, error) {
	// 1. 数据合并
	mergedRecords, err := dp.MergeData(sourceRecords, request, result)
	if err != nil {
		return nil, nil, fmt.Errorf("数据合并失败: %w", err)
	}

	// 2. 字段映射处理
	mappedRecords, err := dp.applyFieldMapping(mergedRecords, request)
	if err != nil {
		return nil, nil, fmt.Errorf("字段映射失败: %w", err)
	}

	// 3. 数据治理处理
	governedRecords, governanceResult, err := dp.performGovernanceProcessing(mappedRecords, request, result)
	if err != nil {
		return nil, governanceResult, err
	}

	// 4. 更新增量同步值（如果启用了增量同步）
	err = dp.updateIncrementalValues(sourceRecords, request)
	if err != nil {
		fmt.Printf("[WARNING] 更新增量同步值失败: %v\n", err)
	}

	return governedRecords, governanceResult, nil
}

// applyFieldMapping 应用字段映射
func (dp *DataProcessor) applyFieldMapping(mergedRecords []map[string]interface{}, request *SyncRequest) ([]map[string]interface{}, error) {
	// 从请求配置中获取字段映射规则
	var fieldMappingRules interface{}
	if rules, exists := request.Config["field_mapping_rules"]; exists {
		fieldMappingRules = rules
	}

	// 应用字段映射
	mappedRecords, err := dp.fieldMapper.ApplyFieldMapping(
		mergedRecords,
		request.TargetInterfaceID,
		fieldMappingRules,
	)
	if err != nil {
		return nil, fmt.Errorf("应用字段映射失败: %w", err)
	}

	// 记录处理步骤信息（暂时注释，后续可以添加到result中）
	// stepInfo := ProcessingStepInfo{
	// 	Phase:       PhaseAggregation, // 使用聚合阶段表示字段映射
	// 	StartTime:   time.Now().Add(-time.Millisecond * 100), // 简化时间处理
	// 	EndTime:     time.Now(),
	// 	Duration:    time.Millisecond * 100,
	// 	RecordCount: int64(len(mappedRecords)),
	// 	ErrorCount:  int64(len(mergedRecords) - len(mappedRecords)),
	// 	Status:      "success",
	// 	Message:     fmt.Sprintf("字段映射完成，处理记录数: %d", len(mappedRecords)),
	// }

	fmt.Printf("[DEBUG] 字段映射完成，原记录数: %d，映射后记录数: %d\n",
		len(mergedRecords), len(mappedRecords))

	return mappedRecords, nil
}

// MergeData 执行简单的数据合并（基于主键ID）
func (dp *DataProcessor) MergeData(sourceRecords []SourceRecordInfo, request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, error) {
	// 调试：打印源记录数量
	slog.Debug("源记录数量", "count", len(sourceRecords))

	// 获取目标主题接口的主键字段
	targetPrimaryKeys, err := dp.getThematicPrimaryKeyFields(request.TargetInterfaceID)
	if err != nil {
		fmt.Printf("[DEBUG] 获取目标主键字段失败: %v, 不使用排序\n", err)
		targetPrimaryKeys = []string{}
	}
	if len(targetPrimaryKeys) > 0 {
		slog.Debug("目标主键字段", "value", targetPrimaryKeys)
	} else {
		slog.Debug("目标接口没有配置主键字段")
	}

	// 使用map按ID合并数据
	recordMap := make(map[string]map[string]interface{})

	for _, sourceRecord := range sourceRecords {
		// 根据目标接口的主键字段提取记录ID
		id := dp.extractPrimaryKeyByFields(sourceRecord.Record, targetPrimaryKeys)
		if id == "" {
			// 如果没有主键，使用记录的哈希值作为ID
			id = dp.generateRecordHash(sourceRecord.Record)
			slog.Debug("记录缺少主键字段，使用哈希ID", "value", id)
		}

		// 如果已存在相同ID的记录，合并字段
		if existingRecord, exists := recordMap[id]; exists {
			// 合并字段，新数据覆盖旧数据
			for key, value := range sourceRecord.Record {
				existingRecord[key] = value
			}
		} else {
			// 复制记录数据
			recordData := make(map[string]interface{})
			for key, value := range sourceRecord.Record {
				recordData[key] = value
			}
			recordMap[id] = recordData
		}
	}

	// 将map转换为切片
	mergedRecords := make([]map[string]interface{}, 0, len(recordMap))
	for _, record := range recordMap {
		mergedRecords = append(mergedRecords, record)
	}

	// 调试：打印合并结果
	slog.Debug("合并结果记录数量", "count", len(mergedRecords))
	for i, record := range mergedRecords {
		slog.Debug("合并记录", "index", i, "record", record)
		if i >= 2 { // 只打印前3条记录，避免日志太长
			break
		}
	}

	// 更新处理记录数
	result.ProcessedRecordCount = int64(len(mergedRecords))

	return mergedRecords, nil
}

// performGovernanceProcessing 执行数据治理处理
func (dp *DataProcessor) performGovernanceProcessing(records []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, *GovernanceExecutionResult, error) {
	// 获取任务信息以获取数据治理配置
	task, err := dp.getTaskInfo(request.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取任务信息失败: %w", err)
	}

	// 从请求配置中获取数据治理配置
	var governanceConfig *GovernanceExecutionConfig
	if configInterface, exists := request.Config["governance_config"]; exists {
		if config, ok := configInterface.(GovernanceExecutionConfig); ok {
			governanceConfig = &config
		} else {
			// 如果类型不匹配，使用默认配置
			governanceConfig = &GovernanceExecutionConfig{
				EnableQualityCheck:   true,
				EnableCleansing:      true,
				EnableMasking:        false,
				StopOnQualityFailure: false,
				QualityThreshold:     0.8,
				BatchSize:            1000,
				MaxRetries:           3,
				TimeoutSeconds:       300,
			}
		}
	} else {
		// 使用默认配置
		governanceConfig = &GovernanceExecutionConfig{
			EnableQualityCheck:   true,
			EnableCleansing:      true,
			EnableMasking:        false,
			StopOnQualityFailure: false,
			QualityThreshold:     0.8,
			BatchSize:            1000,
			MaxRetries:           3,
			TimeoutSeconds:       300,
		}
	}

	// 使用数据治理集成服务处理数据
	processedRecords, governanceResult, err := dp.governanceIntegration.ApplyGovernanceRules(
		request.Context,
		records,
		task,
		governanceConfig,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("数据治理处理失败: %w", err)
	}

	// 更新结果统计
	result.QualityScore = governanceResult.OverallQualityScore

	// 添加处理步骤信息
	stepInfo := ProcessingStepInfo{
		Phase:       PhaseGovernance,
		StartTime:   time.Now().Add(-governanceResult.ExecutionTime),
		EndTime:     time.Now(),
		Duration:    governanceResult.ExecutionTime,
		RecordCount: governanceResult.TotalProcessedRecords,
		ErrorCount:  governanceResult.TotalValidationErrors,
		Status:      governanceResult.ComplianceStatus,
		Message:     fmt.Sprintf("数据治理处理完成，质量评分: %.2f", governanceResult.OverallQualityScore),
	}
	result.ProcessingSteps = append(result.ProcessingSteps, stepInfo)

	return processedRecords, governanceResult, nil
}

// getTaskInfo 获取任务信息
func (dp *DataProcessor) getTaskInfo(taskID string) (*models.ThematicSyncTask, error) {
	var task models.ThematicSyncTask
	if err := dp.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取任务信息失败: %w", err)
	}
	return &task, nil
}

// getThematicPrimaryKeyFields 获取主题接口的主键字段列表
func (dp *DataProcessor) getThematicPrimaryKeyFields(thematicInterfaceID string) ([]string, error) {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := dp.db.First(&thematicInterface, "id = ?", thematicInterfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	return GetThematicPrimaryKeyFields(&thematicInterface), nil
}

// extractPrimaryKeyByFields 根据指定字段提取主键值
func (dp *DataProcessor) extractPrimaryKeyByFields(record map[string]interface{}, primaryKeyFields []string) string {
	var keyParts []string

	for _, field := range primaryKeyFields {
		if value, exists := record[field]; exists && value != nil {
			keyParts = append(keyParts, fmt.Sprintf("%v", value))
		} else {
			// 如果任一主键字段缺失，返回空字符串
			return ""
		}
	}

	// 如果是复合主键，用下划线连接
	if len(keyParts) > 1 {
		return strings.Join(keyParts, "_")
	} else if len(keyParts) == 1 {
		return keyParts[0]
	}

	return ""
}

// generateRecordHash 生成记录的哈希值作为ID
func (dp *DataProcessor) generateRecordHash(record map[string]interface{}) string {
	// 将记录转换为字符串并生成哈希
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}

	// 排序确保一致性
	sort.Strings(keys)

	var builder strings.Builder
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(":")
		builder.WriteString(fmt.Sprintf("%v", record[k]))
		builder.WriteString(";")
	}

	// 使用简单的哈希算法
	h := fnv.New32a()
	h.Write([]byte(builder.String()))
	return fmt.Sprintf("hash_%x", h.Sum32())
}

// updateIncrementalValues 更新增量同步值
func (dp *DataProcessor) updateIncrementalValues(sourceRecords []SourceRecordInfo, request *SyncRequest) error {
	// 从请求配置中解析源库配置
	sourceConfigsRaw, exists := request.Config["source_libraries"]
	if !exists {
		return nil // 没有源库配置，跳过更新
	}

	sourceConfigs, ok := sourceConfigsRaw.([]SourceLibraryConfig)
	if !ok {
		// 尝试从接口数组转换
		if configSlice, ok := sourceConfigsRaw.([]interface{}); ok {
			for _, configRaw := range configSlice {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					// 检查是否有增量配置
					if incrementalRaw, exists := configMap["incremental_config"]; exists {
						if incrementalMap, ok := incrementalRaw.(map[string]interface{}); ok {
							// 解析增量配置
							enabled, _ := incrementalMap["enabled"].(bool)
							if !enabled {
								continue
							}

							libraryID := fmt.Sprintf("%v", configMap["library_id"])
							interfaceID := fmt.Sprintf("%v", configMap["interface_id"])
							incrementalField := fmt.Sprintf("%v", incrementalMap["incremental_field"])
							fieldType := fmt.Sprintf("%v", incrementalMap["field_type"])

							// 找到对应的源记录
							var interfaceRecords []map[string]interface{}
							for _, sourceRecord := range sourceRecords {
								if sourceRecord.LibraryID == libraryID && sourceRecord.InterfaceID == interfaceID {
									interfaceRecords = append(interfaceRecords, sourceRecord.Record)
								}
							}

							if len(interfaceRecords) > 0 {
								// 计算最新的增量值
								maxValue := dp.findMaxIncrementalValue(interfaceRecords, incrementalField, fieldType)
								if maxValue != "" {
									// 更新增量配置中的LastSyncValue
									incrementalMap["last_sync_value"] = maxValue
									fmt.Printf("[DEBUG] 更新增量同步值 - 库: %s, 接口: %s, 字段: %s, 值: %s\n",
										libraryID, interfaceID, incrementalField, maxValue)
								}
							}
						}
					}
				}
			}
		}
	} else {
		// 直接处理SourceLibraryConfig数组
		for i, config := range sourceConfigs {
			if config.IncrementalConfig != nil && config.IncrementalConfig.Enabled {
				// 找到对应的源记录
				var interfaceRecords []map[string]interface{}
				for _, sourceRecord := range sourceRecords {
					if sourceRecord.LibraryID == config.LibraryID && sourceRecord.InterfaceID == config.InterfaceID {
						interfaceRecords = append(interfaceRecords, sourceRecord.Record)
					}
				}

				if len(interfaceRecords) > 0 {
					// 计算最新的增量值
					maxValue := dp.findMaxIncrementalValue(interfaceRecords,
						config.IncrementalConfig.IncrementalField,
						config.IncrementalConfig.FieldType)
					if maxValue != "" {
						// 更新增量配置中的LastSyncValue
						sourceConfigs[i].IncrementalConfig.LastSyncValue = maxValue
						fmt.Printf("[DEBUG] 更新增量同步值 - 库: %s, 接口: %s, 字段: %s, 值: %s\n",
							config.LibraryID, config.InterfaceID,
							config.IncrementalConfig.IncrementalField, maxValue)
					}
				}
			}
		}
		// 更新请求配置
		request.Config["source_libraries"] = sourceConfigs
	}

	return nil
}

// findMaxIncrementalValue 找到记录中增量字段的最大值
func (dp *DataProcessor) findMaxIncrementalValue(records []map[string]interface{}, incrementalField, fieldType string) string {
	var maxValue string

	for _, record := range records {
		if value, exists := record[incrementalField]; exists && value != nil {
			valueStr := fmt.Sprintf("%v", value)
			if maxValue == "" || dp.compareIncrementalValues(valueStr, maxValue, fieldType) > 0 {
				maxValue = valueStr
			}
		}
	}

	return maxValue
}

// compareIncrementalValues 比较增量字段值
func (dp *DataProcessor) compareIncrementalValues(a, b string, fieldType string) int {
	switch fieldType {
	case "number":
		// 简化的数字比较
		if a > b {
			return 1
		} else if a < b {
			return -1
		}
		return 0
	case "timestamp":
		// 简化的时间戳比较
		if a > b {
			return 1
		} else if a < b {
			return -1
		}
		return 0
	default:
		// 字符串比较
		if a > b {
			return 1
		} else if a < b {
			return -1
		}
		return 0
	}
}
