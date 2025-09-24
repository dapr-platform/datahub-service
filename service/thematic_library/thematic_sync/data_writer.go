/*
 * @module service/thematic_sync/data_writer
 * @description 数据写入器，负责将处理后的数据写入目标数据库
 * @architecture 适配器模式 - 适配不同的数据库写入策略
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 数据准备 -> SQL构建 -> 批量写入 -> 结果统计
 * @rules 确保数据写入的一致性和完整性，支持事务操作
 * @dependencies gorm.io/gorm, fmt, strings, time
 * @refs sync_types.go, models/thematic_sync.go
 */

package thematic_sync

import (
	"datahub-service/service/models"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DataWriter 数据写入器
type DataWriter struct {
	db              *gorm.DB
	strategyFactory *SyncStrategyFactory
}

// NewDataWriter 创建数据写入器
func NewDataWriter(db *gorm.DB) *DataWriter {
	return &DataWriter{
		db:              db,
		strategyFactory: NewSyncStrategyFactory(db),
	}
}

// WriteData 写入数据 - 支持不同的同步策略
func (dw *DataWriter) WriteData(processedRecords []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error {
	// 从请求配置中获取同步模式，默认为全量同步
	syncMode := "full"
	if modeInterface, exists := request.Config["sync_mode"]; exists {
		if mode, ok := modeInterface.(string); ok && mode != "" {
			syncMode = mode
		}
	}

	// 创建同步策略
	strategy := dw.strategyFactory.CreateStrategy(syncMode)

	// 使用策略处理删除同步（仅全量同步需要）
	if syncMode == "full" {
		if err := strategy.ProcessSync(processedRecords, request, result); err != nil {
			return fmt.Errorf("同步策略处理失败: %w", err)
		}
	}

	// 继续执行原有的插入/更新逻辑
	return dw.writeDataToTable(processedRecords, request, result, governanceResult)
}

// writeDataToTable 写入数据到表 - 原有的写入逻辑
func (dw *DataWriter) writeDataToTable(processedRecords []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := dw.db.Preload("ThematicLibrary").First(&thematicInterface, "id = ?", request.TargetInterfaceID).Error; err != nil {
		return fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证主题库信息
	if thematicInterface.ThematicLibrary.NameEn == "" {
		return fmt.Errorf("主题库英文名为空")
	}
	if thematicInterface.NameEn == "" {
		return fmt.Errorf("主题接口英文名为空")
	}

	if len(processedRecords) == 0 {
		return nil // 没有数据需要写入
	}

	// 构建表名：主题库的name_en作为schema，主题接口的name_en作为表名
	schema := thematicInterface.ThematicLibrary.NameEn
	tableName := thematicInterface.NameEn
	fullTableName := fmt.Sprintf("%s.%s", schema, tableName)

	// 获取主题接口的主键字段
	primaryKeyFields, err := dw.getThematicPrimaryKeyFields(request.TargetInterfaceID)
	if err != nil {
		fmt.Printf("[DEBUG] 获取主题接口主键字段失败: %v, 使用默认主键\n", err)
		primaryKeyFields = []string{"id"}
	}
	fmt.Printf("[DEBUG] 主题接口主键字段: %v\n", primaryKeyFields)

	// 批量写入数据 - 支持批处理
	return dw.batchWriteRecords(fullTableName, primaryKeyFields, processedRecords, result)
}

// batchWriteRecords 批量写入记录
func (dw *DataWriter) batchWriteRecords(fullTableName string, primaryKeyFields []string, processedRecords []map[string]interface{}, result *SyncExecutionResult) error {
	batchSize := 100 // 批处理大小
	insertedCount := int64(0)
	updatedCount := int64(0)

	for i := 0; i < len(processedRecords); i += batchSize {
		end := i + batchSize
		if end > len(processedRecords) {
			end = len(processedRecords)
		}

		batch := processedRecords[i:end]
		inserted, updated, err := dw.writeBatch(fullTableName, primaryKeyFields, batch)
		if err != nil {
			return fmt.Errorf("批量写入失败 (batch %d-%d): %w", i, end-1, err)
		}

		insertedCount += inserted
		updatedCount += updated
	}

	result.ProcessedRecordCount = int64(len(processedRecords))
	result.InsertedRecordCount = insertedCount
	result.UpdatedRecordCount = updatedCount

	return nil
}

// writeBatch 写入一个批次的记录
func (dw *DataWriter) writeBatch(fullTableName string, primaryKeyFields []string, batch []map[string]interface{}) (int64, int64, error) {
	insertedCount := int64(0)
	updatedCount := int64(0)

	for _, record := range batch {
		if len(record) == 0 {
			continue
		}

		// 确保为NOT NULL字段提供默认值
		record = dw.ensureRequiredFields(record)

		// 构建插入SQL
		columns := make([]string, 0, len(record))
		placeholders := make([]string, 0, len(record))
		values := make([]interface{}, 0, len(record))

		paramIndex := 1
		for k, v := range record {
			if k != "" { // 过滤空列名
				columns = append(columns, k)
				placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
				values = append(values, dw.convertValueForDatabase(v))
				paramIndex++
			}
		}

		if len(columns) == 0 {
			continue
		}

		updateClause := dw.generateUpdateClauseWithPrimaryKeys(columns, primaryKeyFields)
		conflictColumns := dw.buildConflictColumns(primaryKeyFields)
		var sql string
		if updateClause != "" {
			sql = fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
				fullTableName,
				strings.Join(columns, "\", \""),
				strings.Join(placeholders, ", "),
				conflictColumns,
				updateClause)
		} else {
			// 如果没有可更新的列，使用DO NOTHING
			sql = fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s) ON CONFLICT (%s) DO NOTHING",
				fullTableName,
				strings.Join(columns, "\", \""),
				strings.Join(placeholders, ", "),
				conflictColumns)
		}

		// 执行SQL
		result := dw.db.Exec(sql, values...)
		if result.Error != nil {
			return insertedCount, updatedCount, fmt.Errorf("写入数据到表 %s 失败: %w", fullTableName, result.Error)
		}

		// 统计插入和更新数量（简化处理，实际应该根据SQL执行结果判断）
		if result.RowsAffected > 0 {
			insertedCount++
		}
	}

	return insertedCount, updatedCount, nil
}

// getThematicPrimaryKeyFields 获取主题接口的主键字段列表
func (dw *DataWriter) getThematicPrimaryKeyFields(thematicInterfaceID string) ([]string, error) {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := dw.db.First(&thematicInterface, "id = ?", thematicInterfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	var primaryKeys []string

	// 从TableFieldsConfig中解析主键字段
	if len(thematicInterface.TableFieldsConfig) > 0 {
		// 这里需要实现JSON解析逻辑
		// 简化实现，直接返回默认主键
		primaryKeys = []string{"id"}
	}

	// 如果没有主键，使用默认的id字段
	if len(primaryKeys) == 0 {
		primaryKeys = []string{"id"}
	}

	return primaryKeys, nil
}

// generateUpdateClauseWithPrimaryKeys 生成UPDATE子句，跳过指定的主键字段
func (dw *DataWriter) generateUpdateClauseWithPrimaryKeys(columns []string, primaryKeyFields []string) string {
	// 创建主键字段映射，用于快速查找
	primaryKeyMap := make(map[string]bool)
	for _, pk := range primaryKeyFields {
		primaryKeyMap[pk] = true
	}

	var updateParts []string
	for _, column := range columns {
		if !primaryKeyMap[column] { // 跳过主键字段
			updateParts = append(updateParts, fmt.Sprintf("\"%s\" = EXCLUDED.\"%s\"", column, column))
		}
	}
	return strings.Join(updateParts, ", ")
}

// buildConflictColumns 构建ON CONFLICT子句中的列名部分
func (dw *DataWriter) buildConflictColumns(primaryKeyFields []string) string {
	var quotedFields []string
	for _, field := range primaryKeyFields {
		quotedFields = append(quotedFields, fmt.Sprintf("\"%s\"", field))
	}
	return strings.Join(quotedFields, ", ")
}

// convertValueForDatabase 转换值用于数据库写入
func (dw *DataWriter) convertValueForDatabase(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// 处理布尔值转换
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return v
	case float64:
		return v
	case string:
		return v
	case bool:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ensureRequiredFields 确保为NOT NULL字段提供默认值
func (dw *DataWriter) ensureRequiredFields(record map[string]interface{}) map[string]interface{} {
	// 创建记录副本
	result := make(map[string]interface{})
	for k, v := range record {
		result[k] = v
	}

	// 确保必需字段有值
	if result["created_by"] == nil || result["created_by"] == "" {
		result["created_by"] = "system"
	}
	if result["updated_by"] == nil || result["updated_by"] == "" {
		result["updated_by"] = "system"
	}
	if result["created_time"] == nil {
		result["created_time"] = time.Now()
	}
	if result["updated_time"] == nil {
		result["updated_time"] = time.Now()
	}
	if result["group_id"] == nil {
		result["group_id"] = ""
	}
	if result["parent_id"] == nil {
		result["parent_id"] = ""
	}
	if result["product_id"] == nil {
		result["product_id"] = ""
	}
	if result["protocol_config"] == nil {
		result["protocol_config"] = ""
	}

	// 处理布尔类型字段的转换
	if result["enabled"] != nil {
		if v, ok := result["enabled"].(int); ok {
			result["enabled"] = v != 0
		} else if v, ok := result["enabled"].(int64); ok {
			result["enabled"] = v != 0
		}
	} else {
		result["enabled"] = false
	}

	if result["status"] != nil {
		if v, ok := result["status"].(int); ok {
			result["status"] = v != 0
		} else if v, ok := result["status"].(int64); ok {
			result["status"] = v != 0
		}
	} else {
		result["status"] = false
	}

	if result["type"] != nil {
		if v, ok := result["type"].(int); ok {
			result["type"] = v != 0
		} else if v, ok := result["type"].(int64); ok {
			result["type"] = v != 0
		}
	} else {
		result["type"] = false
	}

	return result
}
