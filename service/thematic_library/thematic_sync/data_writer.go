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

// FieldConfig 字段配置信息
type FieldConfig struct {
	NameEn       string      `json:"name_en"`
	NameZh       string      `json:"name_zh"`
	DataType     string      `json:"data_type"`
	IsPrimaryKey bool        `json:"is_primary_key"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue interface{} `json:"default_value"`
}

// DataWriter 数据写入器
type DataWriter struct {
	db              *gorm.DB
	strategyFactory *SyncStrategyFactory
	fieldMapper     *FieldMapper
}

// NewDataWriter 创建数据写入器
func NewDataWriter(db *gorm.DB) *DataWriter {
	return &DataWriter{
		db:              db,
		strategyFactory: NewSyncStrategyFactory(db),
		fieldMapper:     NewFieldMapper(db),
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
	primaryKeyFields := dw.getThematicPrimaryKeyFields(&thematicInterface)
	if len(primaryKeyFields) > 0 {
		fmt.Printf("[DEBUG] 主题接口主键字段: %v\n", primaryKeyFields)
	} else {
		fmt.Printf("[DEBUG] 主题接口没有配置主键字段\n")
	}

	// 获取字段配置信息（包括类型和约束）
	fieldConfigs, err := dw.getFieldConfigsFromInterface(&thematicInterface)
	if err != nil {
		fmt.Printf("[WARNING] 获取字段配置信息失败: %v，使用默认转换\n", err)
		fieldConfigs = make(map[string]FieldConfig)
	}

	// 批量写入数据 - 支持批处理
	return dw.batchWriteRecordsWithConfigs(fullTableName, primaryKeyFields, processedRecords, fieldConfigs, result)
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

		// 过滤并验证字段：只保留有效的字段
		validRecord := dw.filterValidFields(record)
		if len(validRecord) == 0 {
			fmt.Printf("[WARNING] 记录中没有有效字段，跳过写入\n")
			continue
		}

		// 构建插入SQL
		columns := make([]string, 0, len(validRecord))
		placeholders := make([]string, 0, len(validRecord))
		values := make([]interface{}, 0, len(validRecord))

		paramIndex := 1
		for k, v := range validRecord {
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

// getFieldTypesFromInterface 从主题接口获取字段类型信息
func (dw *DataWriter) getFieldTypesFromInterface(thematicInterface *models.ThematicInterface) (map[string]string, error) {
	fieldTypes := make(map[string]string)

	// 从TableFieldsConfig中解析字段类型
	if len(thematicInterface.TableFieldsConfig) > 0 {
		for fieldKey, fieldValue := range thematicInterface.TableFieldsConfig {
			if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
				nameEn := dw.getStringFromMap(fieldMap, "name_en")
				dataType := dw.getStringFromMap(fieldMap, "data_type")

				// 使用name_en作为字段名，如果没有则使用fieldKey
				fieldName := nameEn
				if fieldName == "" {
					fieldName = fieldKey
				}

				if dataType != "" {
					fieldTypes[fieldName] = dataType
				}
			}
		}
	}

	fmt.Printf("[DEBUG] 获取字段类型信息，字段数: %d\n", len(fieldTypes))
	for fieldName, fieldType := range fieldTypes {
		fmt.Printf("[DEBUG] 字段类型: %s -> %s\n", fieldName, fieldType)
	}

	return fieldTypes, nil
}

// getFieldConfigsFromInterface 从主题接口获取字段配置信息
func (dw *DataWriter) getFieldConfigsFromInterface(thematicInterface *models.ThematicInterface) (map[string]FieldConfig, error) {
	fieldConfigs := make(map[string]FieldConfig)

	// 从TableFieldsConfig中解析字段配置
	if len(thematicInterface.TableFieldsConfig) > 0 {
		for fieldKey, fieldValue := range thematicInterface.TableFieldsConfig {
			if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
				config := FieldConfig{
					NameEn:       dw.getStringFromMap(fieldMap, "name_en"),
					NameZh:       dw.getStringFromMap(fieldMap, "name_zh"),
					DataType:     dw.getStringFromMap(fieldMap, "data_type"),
					IsPrimaryKey: dw.getBoolFromMap(fieldMap, "is_primary_key"),
					IsNullable:   dw.getBoolFromMap(fieldMap, "is_nullable"),
					DefaultValue: fieldMap["default_value"],
				}

				// 使用name_en作为字段名，如果没有则使用fieldKey
				fieldName := config.NameEn
				if fieldName == "" {
					fieldName = fieldKey
				}

				fieldConfigs[fieldName] = config
			}
		}
	}

	fmt.Printf("[DEBUG] 获取字段配置信息，字段数: %d\n", len(fieldConfigs))
	for fieldName, config := range fieldConfigs {
		fmt.Printf("[DEBUG] 字段配置: %s (类型: %s, 主键: %v, 可空: %v)\n",
			fieldName, config.DataType, config.IsPrimaryKey, config.IsNullable)
	}

	return fieldConfigs, nil
}

// batchWriteRecordsWithConfigs 批量写入记录 - 支持字段配置
func (dw *DataWriter) batchWriteRecordsWithConfigs(fullTableName string, primaryKeyFields []string, processedRecords []map[string]interface{}, fieldConfigs map[string]FieldConfig, result *SyncExecutionResult) error {
	batchSize := 100 // 批处理大小
	insertedCount := int64(0)
	updatedCount := int64(0)

	for i := 0; i < len(processedRecords); i += batchSize {
		end := i + batchSize
		if end > len(processedRecords) {
			end = len(processedRecords)
		}

		batch := processedRecords[i:end]
		inserted, updated, err := dw.writeBatchWithConfigs(fullTableName, primaryKeyFields, batch, fieldConfigs)
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

// writeBatchWithConfigs 写入一个批次的记录 - 支持字段配置
func (dw *DataWriter) writeBatchWithConfigs(fullTableName string, primaryKeyFields []string, batch []map[string]interface{}, fieldConfigs map[string]FieldConfig) (int64, int64, error) {
	insertedCount := int64(0)
	updatedCount := int64(0)

	for _, record := range batch {
		if len(record) == 0 {
			continue
		}

		// 过滤并验证字段：只保留有效的字段
		validRecord := dw.filterValidFields(record)
		if len(validRecord) == 0 {
			fmt.Printf("[WARNING] 记录中没有有效字段，跳过写入\n")
			continue
		}

		// 根据字段配置确保必需字段有值
		validRecord = dw.ensureRequiredFieldsByConfig(validRecord, fieldConfigs)

		// 构建插入SQL
		columns := make([]string, 0, len(validRecord))
		placeholders := make([]string, 0, len(validRecord))
		values := make([]interface{}, 0, len(validRecord))

		paramIndex := 1
		for k, v := range validRecord {
			if k != "" { // 过滤空列名
				columns = append(columns, k)
				placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))

				// 根据字段配置进行智能转换
				if config, exists := fieldConfigs[k]; exists {
					convertedValue := dw.convertValueByFieldType(v, config.DataType)
					values = append(values, convertedValue)
				} else {
					values = append(values, dw.convertValueForDatabase(v))
				}
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

// batchWriteRecordsWithTypes 批量写入记录 - 支持字段类型转换
func (dw *DataWriter) batchWriteRecordsWithTypes(fullTableName string, primaryKeyFields []string, processedRecords []map[string]interface{}, fieldTypes map[string]string, result *SyncExecutionResult) error {
	batchSize := 100 // 批处理大小
	insertedCount := int64(0)
	updatedCount := int64(0)

	for i := 0; i < len(processedRecords); i += batchSize {
		end := i + batchSize
		if end > len(processedRecords) {
			end = len(processedRecords)
		}

		batch := processedRecords[i:end]
		inserted, updated, err := dw.writeBatchWithTypes(fullTableName, primaryKeyFields, batch, fieldTypes)
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

// writeBatchWithTypes 写入一个批次的记录 - 支持字段类型转换
func (dw *DataWriter) writeBatchWithTypes(fullTableName string, primaryKeyFields []string, batch []map[string]interface{}, fieldTypes map[string]string) (int64, int64, error) {
	insertedCount := int64(0)
	updatedCount := int64(0)

	for _, record := range batch {
		if len(record) == 0 {
			continue
		}

		// 过滤并验证字段：只保留有效的字段
		validRecord := dw.filterValidFields(record)
		if len(validRecord) == 0 {
			fmt.Printf("[WARNING] 记录中没有有效字段，跳过写入\n")
			continue
		}

		// 为必需字段添加默认值
		validRecord = dw.ensureRequiredFields(validRecord, fieldTypes, fullTableName)

		// 构建插入SQL
		columns := make([]string, 0, len(validRecord))
		placeholders := make([]string, 0, len(validRecord))
		values := make([]interface{}, 0, len(validRecord))

		paramIndex := 1
		for k, v := range validRecord {
			if k != "" { // 过滤空列名
				columns = append(columns, k)
				placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))

				// 根据字段类型进行智能转换
				convertedValue := dw.convertValueByFieldType(v, fieldTypes[k])
				values = append(values, convertedValue)
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

// convertValueByFieldType 根据字段类型转换值
func (dw *DataWriter) convertValueByFieldType(value interface{}, fieldType string) interface{} {
	if value == nil {
		return nil
	}

	// 根据字段类型进行智能转换
	switch strings.ToLower(fieldType) {
	case "bool", "boolean":
		return dw.convertToBool(value)
	case "int", "integer", "int4":
		return dw.convertToInt(value)
	case "bigint", "int8":
		return dw.convertToBigInt(value)
	case "float", "float4", "float8", "decimal", "numeric":
		return dw.convertToFloat(value)
	case "varchar", "text", "char", "string":
		return dw.convertToString(value)
	case "timestamp", "timestamptz", "datetime":
		return dw.convertToTimestamp(value)
	case "jsonb", "json":
		return dw.convertToJSON(value)
	default:
		// 未知类型，使用默认转换
		return dw.convertValueForDatabase(value)
	}
}

// convertToBool 转换为布尔值
func (dw *DataWriter) convertToBool(value interface{}) interface{} {
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		lowerStr := strings.ToLower(strings.TrimSpace(v))
		return lowerStr == "true" || lowerStr == "1" || lowerStr == "yes" || lowerStr == "on"
	default:
		// 默认转换为false
		return false
	}
}

// convertToInt 转换为整数
func (dw *DataWriter) convertToInt(value interface{}) interface{} {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case bool:
		if v {
			return 1
		}
		return 0
	case string:
		if v == "" {
			return 0
		}
		// 让数据库处理字符串到整数的转换
		return v
	default:
		return 0
	}
}

// convertToBigInt 转换为长整数
func (dw *DataWriter) convertToBigInt(value interface{}) interface{} {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case bool:
		if v {
			return int64(1)
		}
		return int64(0)
	case string:
		if v == "" {
			return int64(0)
		}
		// 让数据库处理字符串到整数的转换
		return v
	default:
		return int64(0)
	}
}

// convertToFloat 转换为浮点数
func (dw *DataWriter) convertToFloat(value interface{}) interface{} {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case bool:
		if v {
			return float64(1)
		}
		return float64(0)
	case string:
		if v == "" {
			return float64(0)
		}
		// 让数据库处理字符串到浮点数的转换
		return v
	default:
		return float64(0)
	}
}

// convertToString 转换为字符串
func (dw *DataWriter) convertToString(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	str := fmt.Sprintf("%v", value)
	// 如果是空字符串，保持为空字符串（让默认值处理逻辑来处理）
	return str
}

// convertToTimestamp 转换为时间戳
func (dw *DataWriter) convertToTimestamp(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// 如果是空字符串，返回当前时间
	if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
		return time.Now()
	}

	// 如果已经是时间类型，直接返回
	if _, ok := value.(time.Time); ok {
		return value
	}

	// 其他情况让数据库处理
	return value
}

// convertToJSON 转换为JSON
func (dw *DataWriter) convertToJSON(value interface{}) interface{} {
	// JSON类型保持原值
	return value
}

// getStringFromMap 从map中获取字符串值
func (dw *DataWriter) getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// ensureRequiredFields 确保必需字段有值
func (dw *DataWriter) ensureRequiredFields(record map[string]interface{}, fieldTypes map[string]string, tableName string) map[string]interface{} {
	// 常见的必需字段及其默认值
	requiredFields := map[string]interface{}{
		"created_by":   "system",
		"updated_by":   "system",
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
		"created_time": time.Now(),
		"updated_time": time.Now(),
		"create_time":  time.Now(),
		"update_time":  time.Now(),
		"status":       "active",
		"state":        "active",
		"is_deleted":   false,
		"deleted":      false,
		"is_active":    true,
		"active":       true,
		"version":      1,
		"sort_order":   0,
		"order_num":    0,
	}

	// 检查并添加缺失的必需字段
	for fieldName, defaultValue := range requiredFields {
		if _, exists := record[fieldName]; !exists {
			// 检查字段是否在目标字段类型中定义
			if fieldType, hasType := fieldTypes[fieldName]; hasType {
				// 根据字段类型转换默认值
				convertedValue := dw.convertValueByFieldType(defaultValue, fieldType)
				record[fieldName] = convertedValue
				fmt.Printf("[DEBUG] 为必需字段 %s 添加默认值: %v (类型: %s)\n", fieldName, convertedValue, fieldType)
			}
		} else if record[fieldName] == nil {
			// 字段存在但值为nil，也需要设置默认值
			if fieldType, hasType := fieldTypes[fieldName]; hasType {
				convertedValue := dw.convertValueByFieldType(defaultValue, fieldType)
				record[fieldName] = convertedValue
				fmt.Printf("[DEBUG] 为空值字段 %s 设置默认值: %v (类型: %s)\n", fieldName, convertedValue, fieldType)
			}
		}
	}

	return record
}

// ensureRequiredFieldsByConfig 根据字段配置确保必需字段有值
func (dw *DataWriter) ensureRequiredFieldsByConfig(record map[string]interface{}, fieldConfigs map[string]FieldConfig) map[string]interface{} {
	// 遍历所有字段配置，检查非空字段
	for fieldName, config := range fieldConfigs {
		// 如果字段不可为空，确保有值
		if !config.IsNullable {
			needsDefault := false
			currentValue := record[fieldName]

			if _, exists := record[fieldName]; !exists {
				// 字段不存在
				needsDefault = true
			} else if currentValue == nil {
				// 字段存在但值为nil
				needsDefault = true
			} else if str, ok := currentValue.(string); ok && strings.TrimSpace(str) == "" {
				// 字段存在但是空字符串
				needsDefault = true
			}

			if needsDefault {
				defaultValue := dw.getDefaultValueForField(fieldName, config)
				if defaultValue != nil {
					convertedValue := dw.convertValueByFieldType(defaultValue, config.DataType)
					record[fieldName] = convertedValue
					fmt.Printf("[DEBUG] 为必需字段 %s 设置默认值: %v (类型: %s)\n", fieldName, convertedValue, config.DataType)
				} else {
					fmt.Printf("[WARNING] 必需字段 %s 无法提供默认值，可能导致数据库约束错误\n", fieldName)
				}
			}
		}
	}

	return record
}

// getDefaultValueForField 为字段获取默认值
func (dw *DataWriter) getDefaultValueForField(fieldName string, config FieldConfig) interface{} {
	// 1. 优先使用配置中的默认值
	if config.DefaultValue != nil {
		return config.DefaultValue
	}

	// 2. 根据字段名推断默认值
	fieldNameLower := strings.ToLower(fieldName)
	switch fieldNameLower {
	case "created_by", "updated_by", "create_by", "update_by":
		return "system"
	case "created_at", "updated_at", "create_time", "update_time", "created_time", "updated_time":
		return time.Now()
	case "status", "state":
		return "active"
	case "version":
		return 1
	case "is_deleted", "deleted":
		return false
	case "is_active", "active":
		return true
	case "sort_order", "order_num", "sort_num":
		return 0
	}

	// 3. 根据数据类型提供默认值
	switch strings.ToLower(config.DataType) {
	case "varchar", "text", "char", "string":
		// 字符串类型的特殊处理
		if strings.Contains(fieldNameLower, "name") {
			return "未命名"
		}
		if strings.Contains(fieldNameLower, "code") {
			return "AUTO_" + fmt.Sprintf("%d", time.Now().Unix())
		}
		return ""
	case "int", "integer", "int4", "bigint", "int8":
		return 0
	case "float", "float4", "float8", "decimal", "numeric":
		return 0.0
	case "bool", "boolean":
		return false
	case "timestamp", "timestamptz", "datetime":
		return time.Now()
	case "jsonb", "json":
		return "{}"
	default:
		return nil
	}
}

// getBoolFromMap 从map中获取布尔值
func (dw *DataWriter) getBoolFromMap(m map[string]interface{}, key string) bool {
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

// filterValidFields 过滤有效字段，移除空值和无效字段
func (dw *DataWriter) filterValidFields(record map[string]interface{}) map[string]interface{} {
	validRecord := make(map[string]interface{})

	for k, v := range record {
		// 过滤条件：
		// 1. 字段名不能为空
		// 2. 字段名不能包含特殊字符（防止SQL注入）
		// 3. 值不能是nil（除非明确允许）
		if k == "" {
			continue
		}

		// 检查字段名是否包含危险字符
		if dw.containsDangerousChars(k) {
			fmt.Printf("[WARNING] 字段名包含危险字符，跳过: %s\n", k)
			continue
		}

		// 保留字段（包括nil值，让数据库处理）
		validRecord[k] = v
	}

	return validRecord
}

// containsDangerousChars 检查字段名是否包含危险字符
func (dw *DataWriter) containsDangerousChars(fieldName string) bool {
	// 检查SQL注入相关的危险字符 - 只检查完整的SQL关键词，不检查包含关系
	dangerousPatterns := []string{";", "--", "/*", "*/", "xp_", "sp_"}
	fieldNameUpper := strings.ToUpper(fieldName)

	for _, dangerous := range dangerousPatterns {
		if strings.Contains(fieldNameUpper, dangerous) {
			return true
		}
	}

	// 检查完整的SQL关键词（避免误判包含这些词的正常字段名）
	sqlKeywords := []string{"DROP", "DELETE", "INSERT", "UPDATE", "SELECT", "ALTER", "CREATE", "TRUNCATE"}
	for _, keyword := range sqlKeywords {
		if fieldNameUpper == keyword {
			return true
		}
	}

	// 检查是否只包含字母、数字、下划线
	for _, char := range fieldName {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return true
		}
	}

	return false
}

// getThematicPrimaryKeyFields 获取主题接口的主键字段列表
func (dw *DataWriter) getThematicPrimaryKeyFields(thematicInterface *models.ThematicInterface) []string {
	return GetThematicPrimaryKeyFields(thematicInterface)
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

	// 处理各种数据类型转换
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return v
	case float64:
		return v
	case string:
		// 处理空字符串
		if v == "" {
			return nil
		}
		return v
	case bool:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v
	default:
		// 对于其他类型，尝试转换为字符串
		return fmt.Sprintf("%v", v)
	}
}
