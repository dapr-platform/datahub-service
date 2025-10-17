/*
 * @module service/thematic_sync/data_fetcher
 * @description 数据获取器，负责从各种数据源获取数据
 * @architecture 工厂模式 - 支持多种数据源类型
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 数据源配置 -> 连接建立 -> 数据查询 -> 结果返回
 * @rules 确保数据获取的安全性和效率
 * @dependencies gorm.io/gorm, fmt, time
 * @refs sync_types.go, models/thematic_sync.go
 */

package thematic_sync

import (
	"context"
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// DataFetcher 数据获取器
type DataFetcher struct {
	db               *gorm.DB
	sqlQueryExecutor *SQLQueryExecutor
}

// NewDataFetcher 创建数据获取器
func NewDataFetcher(db *gorm.DB) *DataFetcher {
	return &DataFetcher{
		db:               db,
		sqlQueryExecutor: NewSQLQueryExecutor(db),
	}
}

// FetchSourceData 获取源数据 - 支持两种模式
func (df *DataFetcher) FetchSourceData(request *SyncRequest, result *SyncExecutionResult) ([]SourceRecordInfo, error) {
	var sourceRecords []SourceRecordInfo

	// 优先检查是否使用SQL查询模式
	sqlConfigs, hasSQLConfig := df.parseSQLQueryConfigs(request)
	if hasSQLConfig && len(sqlConfigs) > 0 {
		// SQL模式：直接执行SQL查询获取数据
		slog.Debug("使用SQL查询模式", "queryCount", len(sqlConfigs))
		return df.fetchDataFromSQLQueries(sqlConfigs, result)
	}

	// 接口模式：从基础库的数据接口获取数据
	slog.Debug("使用接口查询模式")
	sourceConfigs, err := df.parseSourceConfigs(request)
	if err != nil {
		return nil, fmt.Errorf("解析源库配置失败: %w", err)
	}

	// 调试：打印解析后的源库配置
	slog.Debug("解析源库配置完成", "count", len(sourceConfigs))
	for i, config := range sourceConfigs {
		slog.Debug("源库配置", "index", i, "libraryID", config.LibraryID, "interfaceID", config.InterfaceID)
		if config.IncrementalConfig != nil {
			slog.Debug("增量配置", "index", i, "enabled", config.IncrementalConfig.Enabled,
				"field", config.IncrementalConfig.IncrementalField,
				"fieldType", config.IncrementalConfig.FieldType,
				"lastSyncValue", config.IncrementalConfig.LastSyncValue,
				"initialValue", config.IncrementalConfig.InitialValue)
		} else {
			slog.Debug("无增量配置", "index", i)
		}
	}

	// 从请求配置中获取字段映射规则
	var fieldMappingRules interface{}
	if rules, exists := request.Config["field_mapping_rules"]; exists {
		fieldMappingRules = rules
		slog.Debug("获取到字段映射规则", "rules", rules)
	} else {
		slog.Debug("未找到字段映射规则配置")
	}

	// 为启用增量同步的配置从主题表查询最大值
	for i := range sourceConfigs {
		if sourceConfigs[i].IncrementalConfig != nil && sourceConfigs[i].IncrementalConfig.Enabled {
			// 从主题表查询增量字段的最大值（基础表字段名 -> 主题表字段名）
			maxValue, err := df.queryMaxIncrementalValueFromThematicTable(
				request.TargetInterfaceID,
				sourceConfigs[i].IncrementalConfig.IncrementalField,
				fieldMappingRules,
			)
			if err != nil {
				slog.Warn("从主题表查询增量最大值失败", "error", err, "interfaceID", request.TargetInterfaceID, "sourceField", sourceConfigs[i].IncrementalConfig.IncrementalField)
				// 继续使用配置中的值或初始值
			} else if maxValue != "" {
				// 使用从主题表查询到的最大值
				slog.Debug("从主题表查询到增量最大值", "sourceField", sourceConfigs[i].IncrementalConfig.IncrementalField, "maxValue", maxValue)
				sourceConfigs[i].IncrementalConfig.LastSyncValue = maxValue
			} else {
				slog.Debug("主题表为空，使用初始值", "sourceField", sourceConfigs[i].IncrementalConfig.IncrementalField, "initialValue", sourceConfigs[i].IncrementalConfig.InitialValue)
			}
		}
	}

	// 直接查询每个源接口的数据
	for _, config := range sourceConfigs {
		records, err := df.FetchDataFromInterfaceWithConfig(config.LibraryID, config.InterfaceID, config)
		if err != nil {
			return nil, fmt.Errorf("获取接口数据失败 [%s/%s]: %w",
				config.LibraryID, config.InterfaceID, err)
		}

		// 应用过滤器和转换
		if len(config.Filters) > 0 {
			records = df.applyFilters(records, config.Filters)
		}

		if len(config.Transforms) > 0 {
			records, err = df.applyTransforms(records, config.Transforms)
			if err != nil {
				return nil, fmt.Errorf("应用数据转换失败: %w", err)
			}
		}

		// 转换为源记录信息
		for j, record := range records {
			sourceRecord := SourceRecordInfo{
				LibraryID:   config.LibraryID,
				InterfaceID: config.InterfaceID,
				RecordID:    df.generateRecordID(config.LibraryID, config.InterfaceID, j, record),
				Record:      record,
				Quality:     df.calculateInitialQuality(record),
				LastUpdated: time.Now(),
				Metadata: map[string]interface{}{
					"data_source_type": "direct_query",
					"fetch_time":       time.Now(),
				},
			}
			sourceRecords = append(sourceRecords, sourceRecord)
		}
	}

	result.SourceRecordCount = int64(len(sourceRecords))
	return sourceRecords, nil
}

// FetchDataFromInterface 直接从接口查询数据 - 支持分批获取
func (df *DataFetcher) FetchDataFromInterface(libraryID, interfaceID string) ([]map[string]interface{}, error) {
	// 默认批次大小
	batchSize := 1000

	// 获取接口配置信息
	var dataInterface models.DataInterface
	if err := df.db.Preload("BasicLibrary").First(&dataInterface, "id = ?", interfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取接口信息失败: %w", err)
	}

	// 验证基础库信息
	if dataInterface.BasicLibrary.NameEn == "" {
		return nil, fmt.Errorf("基础库英文名为空")
	}
	if dataInterface.NameEn == "" {
		return nil, fmt.Errorf("基础接口英文名为空")
	}

	// 构建表名：基础库的name_en作为schema，基础接口的name_en作为表名
	schema := dataInterface.BasicLibrary.NameEn
	tableName := dataInterface.NameEn
	fullTableName := fmt.Sprintf("%s.%s", schema, tableName)

	return df.fetchDataInBatches(fullTableName, batchSize)
}

// FetchDataFromInterfaceWithConfig 从接口查询数据 - 支持配置参数和增量同步
func (df *DataFetcher) FetchDataFromInterfaceWithConfig(libraryID, interfaceID string, config SourceLibraryConfig) ([]map[string]interface{}, error) {
	// 从配置中获取批次大小
	batchSize := 1000
	if config.IncrementalConfig != nil && config.IncrementalConfig.BatchSize > 0 {
		batchSize = config.IncrementalConfig.BatchSize
	} else if len(config.Transforms) > 0 {
		// 从转换配置中查找batch_size参数
		for _, transform := range config.Transforms {
			if batchSizeVal, exists := transform.Config["batch_size"]; exists {
				if batchSizeInt, ok := batchSizeVal.(int); ok && batchSizeInt > 0 {
					batchSize = batchSizeInt
				}
			}
		}
	}

	// 获取接口配置信息
	var dataInterface models.DataInterface
	if err := df.db.Preload("BasicLibrary").First(&dataInterface, "id = ?", interfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取接口信息失败: %w", err)
	}

	// 验证基础库信息
	if dataInterface.BasicLibrary.NameEn == "" {
		return nil, fmt.Errorf("基础库英文名为空")
	}
	if dataInterface.NameEn == "" {
		return nil, fmt.Errorf("基础接口英文名为空")
	}

	// 构建表名：基础库的name_en作为schema，基础接口的name_en作为表名
	schema := dataInterface.BasicLibrary.NameEn
	tableName := dataInterface.NameEn
	fullTableName := fmt.Sprintf("%s.%s", schema, tableName)

	// 获取数据接口的主键字段
	primaryKeyFields := GetDataInterfacePrimaryKeyFields(&dataInterface)
	if len(primaryKeyFields) > 0 {
		slog.Debug("数据接口主键字段", "value", primaryKeyFields)
	} else {
		slog.Debug("数据接口没有配置主键字段，查询时不使用排序")
	}

	// 检查是否启用增量同步
	if config.IncrementalConfig != nil && config.IncrementalConfig.Enabled {
		slog.Debug("启用增量同步", "table", fullTableName, "incrementalField", config.IncrementalConfig.IncrementalField)
		return df.fetchIncrementalDataWithPrimaryKey(fullTableName, batchSize, config.IncrementalConfig, primaryKeyFields)
	}

	slog.Debug("未启用增量同步，执行全量查询", "table", fullTableName)
	return df.fetchDataInBatchesWithPrimaryKey(fullTableName, batchSize, primaryKeyFields)
}

// fetchDataInBatchesWithPrimaryKey 分批获取数据 - 支持动态主键
func (df *DataFetcher) fetchDataInBatchesWithPrimaryKey(fullTableName string, batchSize int, primaryKeyFields []string) ([]map[string]interface{}, error) {
	var allRecords []map[string]interface{}
	offset := 0

	// 构建排序子句
	var orderByClause string
	if len(primaryKeyFields) > 0 {
		orderByField := primaryKeyFields[0]
		orderByClause = fmt.Sprintf(" ORDER BY \"%s\"", orderByField)
		slog.Debug("开始分批获取数据", "table", fullTableName, "batchSize", batchSize, "orderByField", orderByField)
	} else {
		orderByClause = "" // 没有主键时不使用排序
		slog.Debug("开始分批获取数据", "table", fullTableName, "batchSize", batchSize, "orderBy", "无")
	}

	for {
		// 构建分页查询SQL
		sql := fmt.Sprintf("SELECT * FROM %s%s LIMIT %d OFFSET %d", fullTableName, orderByClause, batchSize, offset)

		rows, err := df.db.Raw(sql).Rows()
		if err != nil {
			return nil, fmt.Errorf("查询数据失败 (offset: %d): %w", offset, err)
		}

		// 获取列信息
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("获取列信息失败: %w", err)
		}

		// 扫描当前批次的数据
		batchRecords := make([]map[string]interface{}, 0, batchSize)
		for rows.Next() {
			values := make([]interface{}, len(columns))
			scanArgs := make([]interface{}, len(columns))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			if err := rows.Scan(scanArgs...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("扫描数据失败: %w", err)
			}

			record := make(map[string]interface{})
			for i, column := range columns {
				record[column] = df.convertDatabaseValue(values[i])
			}
			batchRecords = append(batchRecords, record)
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("遍历数据失败: %w", err)
		}

		// 如果当前批次没有数据，说明已经获取完毕
		if len(batchRecords) == 0 {
			break
		}

		slog.Debug("获取批次数据", "offset", offset, "recordCount", len(batchRecords))

		allRecords = append(allRecords, batchRecords...)

		// 如果当前批次的记录数少于批次大小，说明已经是最后一批
		if len(batchRecords) < batchSize {
			break
		}

		offset += batchSize
	}

	slog.Debug("总共获取记录数", "count", len(allRecords))
	return allRecords, nil
}

// fetchDataInBatches 分批获取数据
func (df *DataFetcher) fetchDataInBatches(fullTableName string, batchSize int) ([]map[string]interface{}, error) {
	var allRecords []map[string]interface{}
	offset := 0

	slog.Debug("开始分批获取数据", "table", fullTableName, "batchSize", batchSize)

	for {
		// 构建分页查询SQL - 注意：这个方法已废弃，应该使用fetchDataInBatchesWithPrimaryKey
		sql := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", fullTableName, batchSize, offset)

		rows, err := df.db.Raw(sql).Rows()
		if err != nil {
			return nil, fmt.Errorf("查询数据失败 (offset: %d): %w", offset, err)
		}

		// 获取列信息
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("获取列信息失败: %w", err)
		}

		// 扫描当前批次的数据
		batchRecords := make([]map[string]interface{}, 0, batchSize)
		for rows.Next() {
			values := make([]interface{}, len(columns))
			scanArgs := make([]interface{}, len(columns))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			if err := rows.Scan(scanArgs...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("扫描数据失败: %w", err)
			}

			record := make(map[string]interface{})
			for i, column := range columns {
				record[column] = df.convertDatabaseValue(values[i])
			}
			batchRecords = append(batchRecords, record)
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("遍历数据失败: %w", err)
		}

		// 如果当前批次没有数据，说明已经获取完毕
		if len(batchRecords) == 0 {
			break
		}

		slog.Debug("获取批次数据", "offset", offset, "recordCount", len(batchRecords))

		allRecords = append(allRecords, batchRecords...)

		// 如果当前批次的记录数少于批次大小，说明已经是最后一批
		if len(batchRecords) < batchSize {
			break
		}

		offset += batchSize
	}

	slog.Debug("总共获取记录数", "count", len(allRecords))
	return allRecords, nil
}

// convertDatabaseValue 转换数据库值
func (df *DataFetcher) convertDatabaseValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return v
	}
}

// parseSourceConfigs 解析源库配置
func (df *DataFetcher) parseSourceConfigs(request *SyncRequest) ([]SourceLibraryConfig, error) {
	var configs []SourceLibraryConfig

	// 从请求配置中解析源库配置
	if sourceConfigsRaw, exists := request.Config["source_libraries"]; exists {
		// 尝试直接转换
		if configSlice, ok := sourceConfigsRaw.([]SourceLibraryConfig); ok {
			return configSlice, nil
		}

		// 尝试从接口数组转换
		if configSlice, ok := sourceConfigsRaw.([]interface{}); ok {
			for _, configRaw := range configSlice {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := SourceLibraryConfig{
						LibraryID:   df.getStringFromMap(configMap, "library_id"),
						InterfaceID: df.getStringFromMap(configMap, "interface_id"),
						SQLQuery:    df.getStringFromMap(configMap, "sql_query"),
					}

					// 解析参数
					if params, exists := configMap["parameters"]; exists {
						if paramsMap, ok := params.(map[string]interface{}); ok {
							config.Parameters = paramsMap
						}
					}

					// 解析增量配置 - 关键修复点
					if incrementalRaw, exists := configMap["incremental_config"]; exists {
						if incrementalMap, ok := incrementalRaw.(map[string]interface{}); ok {
							incrementalConfig := &IncrementalConfig{
								Enabled:            df.getBoolFromMap(incrementalMap, "enabled"),
								IncrementalField:   df.getStringFromMap(incrementalMap, "incremental_field"),
								FieldType:          df.getStringFromMap(incrementalMap, "field_type"),
								CompareOperator:    df.getStringFromMap(incrementalMap, "compare_operator"),
								LastSyncValue:      df.getStringFromMap(incrementalMap, "last_sync_value"),
								InitialValue:       df.getStringFromMap(incrementalMap, "initial_value"),
								MaxLookbackHours:   df.getIntFromMap(incrementalMap, "max_lookback_hours"),
								CheckDeletedField:  df.getStringFromMap(incrementalMap, "check_deleted_field"),
								DeletedValue:       df.getStringFromMap(incrementalMap, "deleted_value"),
								BatchSize:          df.getIntFromMap(incrementalMap, "batch_size"),
								SyncDeletedRecords: df.getBoolFromMap(incrementalMap, "sync_deleted_records"),
								TimestampFormat:    df.getStringFromMap(incrementalMap, "timestamp_format"),
								TimeZone:           df.getStringFromMap(incrementalMap, "timezone"),
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

							config.IncrementalConfig = incrementalConfig
							slog.Debug("解析到增量配置", "libraryID", config.LibraryID, "interfaceID", config.InterfaceID,
								"enabled", incrementalConfig.Enabled, "field", incrementalConfig.IncrementalField)
						}
					}

					configs = append(configs, config)
				}
			}
			return configs, nil
		}
	}

	// 兜底：如果没有源库配置但有接口列表，构建简单配置
	if len(configs) == 0 && len(request.SourceInterfaces) > 0 {
		for i, interfaceID := range request.SourceInterfaces {
			libraryID := ""
			if i < len(request.SourceLibraries) {
				libraryID = request.SourceLibraries[i]
			}

			configs = append(configs, SourceLibraryConfig{
				LibraryID:   libraryID,
				InterfaceID: interfaceID,
			})
		}
	}

	return configs, nil
}

// applyFilters 应用过滤器
func (df *DataFetcher) applyFilters(records []map[string]interface{}, filters []FilterConfig) []map[string]interface{} {
	if len(filters) == 0 {
		return records
	}

	var filtered []map[string]interface{}

	for _, record := range records {
		if df.matchesFilters(record, filters) {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

// matchesFilters 检查记录是否匹配过滤条件
func (df *DataFetcher) matchesFilters(record map[string]interface{}, filters []FilterConfig) bool {
	for _, filter := range filters {
		if !df.matchesFilter(record, filter) {
			return false // 任意条件不匹配则过滤掉
		}
	}
	return true
}

// matchesFilter 检查单个过滤条件
func (df *DataFetcher) matchesFilter(record map[string]interface{}, filter FilterConfig) bool {
	value, exists := record[filter.Field]
	if !exists {
		return false
	}

	valueStr := fmt.Sprintf("%v", value)
	filterValueStr := fmt.Sprintf("%v", filter.Value)

	switch filter.Operator {
	case "eq", "=":
		return valueStr == filterValueStr
	case "ne", "!=":
		return valueStr != filterValueStr
	default:
		return true // 未知操作符默认通过
	}
}

// applyTransforms 应用数据转换
func (df *DataFetcher) applyTransforms(records []map[string]interface{}, transforms []TransformConfig) ([]map[string]interface{}, error) {
	if len(transforms) == 0 {
		return records, nil
	}

	transformer := NewDataTransformer()

	for i, record := range records {
		for _, transform := range transforms {
			if sourceValue, exists := record[transform.SourceField]; exists {
				// 执行转换
				transformedValue, err := transformer.Transform(sourceValue, transform.Transform)
				if err != nil {
					return nil, fmt.Errorf("记录 %d 字段 %s 转换失败: %w", i, transform.SourceField, err)
				}

				// 设置目标字段值
				record[transform.TargetField] = transformedValue
			}
		}
	}

	return records, nil
}

// generateRecordID 生成记录ID
func (df *DataFetcher) generateRecordID(libraryID, interfaceID string, index int, record map[string]interface{}) string {
	// 尝试使用记录中的主键字段
	keyFields := []string{"id", "uuid", "primary_key", "pk"}

	for _, keyField := range keyFields {
		if value, exists := record[keyField]; exists && value != nil {
			return fmt.Sprintf("%s_%s_%v", libraryID, interfaceID, value)
		}
	}

	// 使用索引生成ID
	return fmt.Sprintf("%s_%s_%d", libraryID, interfaceID, index)
}

// calculateInitialQuality 计算初始质量评分
func (df *DataFetcher) calculateInitialQuality(record map[string]interface{}) float64 {
	if len(record) == 0 {
		return 0.0
	}

	validFieldCount := 0
	for _, value := range record {
		if df.isValidFieldValue(value) {
			validFieldCount++
		}
	}

	return float64(validFieldCount) / float64(len(record))
}

// isValidFieldValue 检查字段值是否有效
func (df *DataFetcher) isValidFieldValue(value interface{}) bool {
	if value == nil {
		return false
	}

	str := fmt.Sprintf("%v", value)
	return str != "" && str != "null" && str != "NULL" && str != "nil"
}

// fetchIncrementalDataWithPrimaryKey 获取增量数据 - 支持动态主键
func (df *DataFetcher) fetchIncrementalDataWithPrimaryKey(fullTableName string, batchSize int, config *IncrementalConfig, primaryKeyFields []string) ([]map[string]interface{}, error) {
	// 构建排序字段，优先使用增量字段，否则使用第一个主键字段
	var orderByField string
	var orderByClause string

	if config.IncrementalField != "" {
		orderByField = config.IncrementalField
		orderByClause = fmt.Sprintf(" ORDER BY \"%s\"", orderByField)
	} else if len(primaryKeyFields) > 0 {
		orderByField = primaryKeyFields[0]
		orderByClause = fmt.Sprintf(" ORDER BY \"%s\"", orderByField)
	} else {
		orderByClause = "" // 没有排序字段时不使用ORDER BY
	}

	if orderByField != "" {
		slog.Debug("开始增量同步", "table", fullTableName, "incrementalField", config.IncrementalField, "orderByField", orderByField)
	} else {
		slog.Debug("开始增量同步", "table", fullTableName, "incrementalField", config.IncrementalField, "orderBy", "无")
	}

	// 构建增量查询条件
	whereCondition, err := df.buildIncrementalCondition(config)
	if err != nil {
		return nil, fmt.Errorf("构建增量查询条件失败: %w", err)
	}

	var allRecords []map[string]interface{}
	offset := 0

	for {
		// 构建增量查询SQL
		var sql string
		if whereCondition != "" {
			sql = fmt.Sprintf("SELECT * FROM %s WHERE %s%s LIMIT %d OFFSET %d",
				fullTableName, whereCondition, orderByClause, batchSize, offset)
		} else {
			// 如果没有增量条件（首次同步），使用普通查询
			sql = fmt.Sprintf("SELECT * FROM %s%s LIMIT %d OFFSET %d",
				fullTableName, orderByClause, batchSize, offset)
		}

		slog.Debug("增量查询SQL", "value", sql)

		rows, err := df.db.Raw(sql).Rows()
		if err != nil {
			return nil, fmt.Errorf("增量查询数据失败 (offset: %d): %w", offset, err)
		}

		// 获取列信息
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("获取列信息失败: %w", err)
		}

		// 扫描当前批次的数据
		batchRecords := make([]map[string]interface{}, 0, batchSize)
		for rows.Next() {
			values := make([]interface{}, len(columns))
			scanArgs := make([]interface{}, len(columns))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			if err := rows.Scan(scanArgs...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("扫描数据失败: %w", err)
			}

			record := make(map[string]interface{})
			for i, column := range columns {
				record[column] = df.convertDatabaseValue(values[i])
			}
			batchRecords = append(batchRecords, record)
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("遍历数据失败: %w", err)
		}

		// 如果当前批次没有数据，说明已经获取完毕
		if len(batchRecords) == 0 {
			break
		}

		slog.Debug("获取增量批次数据", "offset", offset, "recordCount", len(batchRecords))

		allRecords = append(allRecords, batchRecords...)

		// 如果当前批次的记录数少于批次大小，说明已经是最后一批
		if len(batchRecords) < batchSize {
			break
		}

		offset += batchSize
	}

	slog.Debug("增量同步总共获取记录数", "count", len(allRecords))
	return allRecords, nil
}

// fetchIncrementalData 获取增量数据
func (df *DataFetcher) fetchIncrementalData(fullTableName string, batchSize int, config *IncrementalConfig) ([]map[string]interface{}, error) {
	slog.Debug("开始增量同步", "table", fullTableName, "incrementalField", config.IncrementalField)

	// 构建增量查询条件
	whereCondition, err := df.buildIncrementalCondition(config)
	if err != nil {
		return nil, fmt.Errorf("构建增量查询条件失败: %w", err)
	}

	var allRecords []map[string]interface{}
	offset := 0

	for {
		// 构建增量查询SQL
		var sql string
		if whereCondition != "" {
			sql = fmt.Sprintf("SELECT * FROM %s WHERE %s ORDER BY %s LIMIT %d OFFSET %d",
				fullTableName, whereCondition, config.IncrementalField, batchSize, offset)
		} else {
			// 如果没有增量条件（首次同步），使用普通查询
			sql = fmt.Sprintf("SELECT * FROM %s ORDER BY %s LIMIT %d OFFSET %d",
				fullTableName, config.IncrementalField, batchSize, offset)
		}

		slog.Debug("增量查询SQL", "value", sql)

		rows, err := df.db.Raw(sql).Rows()
		if err != nil {
			return nil, fmt.Errorf("增量查询数据失败 (offset: %d): %w", offset, err)
		}

		// 获取列信息
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("获取列信息失败: %w", err)
		}

		// 扫描当前批次的数据
		batchRecords := make([]map[string]interface{}, 0, batchSize)
		for rows.Next() {
			values := make([]interface{}, len(columns))
			scanArgs := make([]interface{}, len(columns))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			if err := rows.Scan(scanArgs...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("扫描数据失败: %w", err)
			}

			record := make(map[string]interface{})
			for i, column := range columns {
				record[column] = df.convertDatabaseValue(values[i])
			}
			batchRecords = append(batchRecords, record)
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("遍历数据失败: %w", err)
		}

		// 如果当前批次没有数据，说明已经获取完毕
		if len(batchRecords) == 0 {
			break
		}

		slog.Debug("获取增量批次数据", "offset", offset, "recordCount", len(batchRecords))

		allRecords = append(allRecords, batchRecords...)

		// 如果当前批次的记录数少于批次大小，说明已经是最后一批
		if len(batchRecords) < batchSize {
			break
		}

		offset += batchSize
	}

	slog.Debug("增量同步总共获取记录数", "count", len(allRecords))
	return allRecords, nil
}

// buildIncrementalCondition 构建增量查询条件
func (df *DataFetcher) buildIncrementalCondition(config *IncrementalConfig) (string, error) {
	if config.LastSyncValue == "" {
		// 首次同步，使用初始值
		if config.InitialValue != "" {
			condition := fmt.Sprintf("%s %s '%s'", config.IncrementalField, config.CompareOperator, config.InitialValue)
			slog.Debug("首次增量同步使用初始值", "condition", condition)
			return condition, nil
		}
		// 没有初始值，返回空条件（获取所有数据）
		slog.Debug("首次增量同步无初始值，获取所有数据")
		return "", nil
	}

	// 构建基础增量条件
	condition := fmt.Sprintf("%s %s '%s'", config.IncrementalField, config.CompareOperator, config.LastSyncValue)
	slog.Debug("构建增量查询条件", "condition", condition, "lastSyncValue", config.LastSyncValue)

	// 如果配置了回溯时间，添加回溯条件
	if config.MaxLookbackHours > 0 && config.FieldType == "timestamp" {
		lookbackCondition := df.buildLookbackCondition(config)
		if lookbackCondition != "" {
			condition = fmt.Sprintf("(%s OR %s)", condition, lookbackCondition)
		}
	}

	// 如果配置了软删除检查，添加删除条件
	if config.CheckDeletedField != "" {
		if config.SyncDeletedRecords {
			// 包含已删除的记录
			if config.DeletedValue != "" {
				condition = fmt.Sprintf("(%s) AND (%s IS NULL OR %s = '%s')",
					condition, config.CheckDeletedField, config.CheckDeletedField, config.DeletedValue)
			}
		} else {
			// 排除已删除的记录
			if config.DeletedValue != "" {
				condition = fmt.Sprintf("(%s) AND (%s IS NULL OR %s != '%s')",
					condition, config.CheckDeletedField, config.CheckDeletedField, config.DeletedValue)
			} else {
				condition = fmt.Sprintf("(%s) AND %s IS NULL", condition, config.CheckDeletedField)
			}
		}
	}

	return condition, nil
}

// buildLookbackCondition 构建回溯条件
func (df *DataFetcher) buildLookbackCondition(config *IncrementalConfig) string {
	if config.FieldType != "timestamp" || config.MaxLookbackHours <= 0 {
		return ""
	}

	// 计算回溯时间点
	return fmt.Sprintf("%s >= NOW() - INTERVAL '%d hours'", config.IncrementalField, config.MaxLookbackHours)
}

// UpdateIncrementalValue 更新增量同步值
func (df *DataFetcher) UpdateIncrementalValue(config *IncrementalConfig, records []map[string]interface{}) string {
	if len(records) == 0 {
		return config.LastSyncValue
	}

	var maxValue string

	// 找到最大的增量字段值
	for _, record := range records {
		if value, exists := record[config.IncrementalField]; exists && value != nil {
			valueStr := fmt.Sprintf("%v", value)
			if maxValue == "" || df.compareIncrementalValues(valueStr, maxValue, config.FieldType) > 0 {
				maxValue = valueStr
			}
		}
	}

	if maxValue != "" {
		return maxValue
	}
	return config.LastSyncValue
}

// compareIncrementalValues 比较增量字段值
func (df *DataFetcher) compareIncrementalValues(a, b string, fieldType string) int {
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

// getStringFromMap 从map中获取字符串值
func (df *DataFetcher) getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// parseSQLQueryConfigs 解析SQL查询配置
func (df *DataFetcher) parseSQLQueryConfigs(request *SyncRequest) ([]*SQLQueryConfig, bool) {
	// 检查请求配置中是否有SQL查询配置
	sqlConfigsRaw, exists := request.Config["sql_queries"]
	if !exists {
		// 兼容旧的字段名 data_source_sql
		sqlConfigsRaw, exists = request.Config["data_source_sql"]
		if !exists {
			return nil, false
		}
	}

	var sqlConfigs []*SQLQueryConfig

	// 尝试直接转换
	if configSlice, ok := sqlConfigsRaw.([]*SQLQueryConfig); ok {
		return configSlice, true
	}

	// 尝试从接口数组转换
	if configSlice, ok := sqlConfigsRaw.([]interface{}); ok {
		for _, configRaw := range configSlice {
			if configMap, ok := configRaw.(map[string]interface{}); ok {
				config := &SQLQueryConfig{
					SQLQuery: df.getStringFromMap(configMap, "sql_query"),
					Timeout:  30,
					MaxRows:  10000,
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

				// 验证SQL查询不为空
				if config.SQLQuery != "" {
					sqlConfigs = append(sqlConfigs, config)
				}
			}
		}

		if len(sqlConfigs) > 0 {
			return sqlConfigs, true
		}
	}

	return nil, false
}

// fetchDataFromSQLQueries 从SQL查询获取数据
func (df *DataFetcher) fetchDataFromSQLQueries(sqlConfigs []*SQLQueryConfig, result *SyncExecutionResult) ([]SourceRecordInfo, error) {
	var sourceRecords []SourceRecordInfo

	for i, sqlConfig := range sqlConfigs {
		slog.Debug("执行SQL查询", "index", i+1, "total", len(sqlConfigs))

		// 使用SQL执行器执行查询
		records, err := df.sqlQueryExecutor.ExecuteQuery(context.TODO(), sqlConfig)
		if err != nil {
			return nil, fmt.Errorf("执行SQL查询失败 [查询%d]: %w", i+1, err)
		}

		// 转换为源记录信息
		for j, record := range records {
			sourceRecord := SourceRecordInfo{
				LibraryID:   "sql_query",                               // SQL查询模式使用固定的标识
				InterfaceID: fmt.Sprintf("sql_query_%d", i+1),          // 使用查询索引作为接口ID
				RecordID:    df.generateRecordIDForSQL(i+1, j, record), // 生成唯一记录ID
				Record:      record,
				Quality:     df.calculateInitialQuality(record),
				LastUpdated: time.Now(),
				Metadata: map[string]interface{}{
					"data_source_type": "sql_query",
					"query_index":      i + 1,
					"fetch_time":       time.Now(),
					"sql_query":        sqlConfig.SQLQuery,
				},
			}
			sourceRecords = append(sourceRecords, sourceRecord)
		}

		slog.Debug("SQL查询返回记录", "index", i+1, "recordCount", len(records))
	}

	result.SourceRecordCount = int64(len(sourceRecords))
	slog.Debug("SQL查询模式总记录数", "total", len(sourceRecords))

	return sourceRecords, nil
}

// generateRecordIDForSQL 为SQL查询结果生成记录ID
func (df *DataFetcher) generateRecordIDForSQL(queryIndex, recordIndex int, record map[string]interface{}) string {
	// 尝试使用记录中的主键字段
	keyFields := []string{"id", "uuid", "primary_key", "pk"}

	for _, keyField := range keyFields {
		if value, exists := record[keyField]; exists && value != nil {
			return fmt.Sprintf("sql_query_%d_%v", queryIndex, value)
		}
	}

	// 使用索引生成ID
	return fmt.Sprintf("sql_query_%d_record_%d", queryIndex, recordIndex)
}

// getBoolFromMap 从map中获取布尔值
func (df *DataFetcher) getBoolFromMap(m map[string]interface{}, key string) bool {
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
func (df *DataFetcher) getIntFromMap(m map[string]interface{}, key string) int {
	if value, exists := m[key]; exists {
		if intVal, ok := value.(int); ok {
			return intVal
		}
		if floatVal, ok := value.(float64); ok {
			return int(floatVal)
		}
		// 尝试从字符串转换
		strVal := fmt.Sprintf("%v", value)
		var result int
		if _, err := fmt.Sscanf(strVal, "%d", &result); err == nil {
			return result
		}
	}
	return 0
}

// queryMaxIncrementalValueFromThematicTable 从主题表查询增量字段的最大值
// sourceIncrementalField: 基础表中的增量字段名
// 返回: 主题表中对应字段的最大值
func (df *DataFetcher) queryMaxIncrementalValueFromThematicTable(thematicInterfaceID, sourceIncrementalField string, fieldMappingRules interface{}) (string, error) {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := df.db.Preload("ThematicLibrary").First(&thematicInterface, "id = ?", thematicInterfaceID).Error; err != nil {
		return "", fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证主题库信息
	if thematicInterface.ThematicLibrary.NameEn == "" {
		return "", fmt.Errorf("主题库英文名为空")
	}
	if thematicInterface.NameEn == "" {
		return "", fmt.Errorf("主题接口英文名为空")
	}

	// 构建主题表名：主题库的name_en作为schema，主题接口的name_en作为表名
	schema := thematicInterface.ThematicLibrary.NameEn
	tableName := thematicInterface.NameEn
	fullTableName := fmt.Sprintf("\"%s\".\"%s\"", schema, tableName)

	// 根据字段映射规则找到主题表中的对应字段名
	targetFieldName := df.findTargetFieldByMapping(sourceIncrementalField, fieldMappingRules)
	if targetFieldName == "" {
		// 如果没有找到映射，尝试直接使用源字段名
		targetFieldName = sourceIncrementalField
		slog.Debug("未找到字段映射，使用源字段名", "sourceField", sourceIncrementalField, "targetField", targetFieldName)
	} else {
		slog.Debug("找到字段映射", "sourceField", sourceIncrementalField, "targetField", targetFieldName)
	}

	// 检查目标字段是否存在于主题接口配置中
	fieldExists := false
	if len(thematicInterface.TableFieldsConfig) > 0 {
		for _, fieldConfig := range thematicInterface.TableFieldsConfig {
			if fieldConfig.(map[string]interface{})["name_en"] == targetFieldName {
				fieldExists = true
				break
			}
		}

	}

	if !fieldExists {
		slog.Warn("目标字段不存在于主题接口配置中", "targetField", targetFieldName, "interfaceID", thematicInterfaceID)
		// 仍然尝试查询，以防字段配置不完整但表中存在该字段
	}

	// 构建查询SQL，获取增量字段的最大值
	sql := fmt.Sprintf("SELECT MAX(\"%s\") as max_value FROM %s", targetFieldName, fullTableName)
	slog.Debug("查询主题表增量最大值", "sql", sql, "sourceField", sourceIncrementalField, "targetField", targetFieldName)

	// 执行查询，直接使用string类型接收
	var maxValue string
	row := df.db.Raw(sql).Row()
	if err := row.Scan(&maxValue); err != nil {
		// 如果扫描失败，可能是NULL值或其他错误
		if err.Error() == "sql: Scan error on column index 0, name \"max_value\": converting NULL to string is unsupported" {
			slog.Debug("主题表为空或字段值全为NULL", "table", fullTableName, "field", targetFieldName)
			return "", nil
		}
		return "", fmt.Errorf("查询增量最大值失败: %w", err)
	}

	// 处理空值
	if maxValue == "" {
		slog.Debug("主题表为空或字段值全为NULL", "table", fullTableName, "field", targetFieldName)
		return "", nil
	}

	slog.Debug("查询到增量最大值", "table", fullTableName, "sourceField", sourceIncrementalField, "targetField", targetFieldName, "maxValue", maxValue)
	return maxValue, nil
}

// findTargetFieldByMapping 根据字段映射规则查找目标字段名
func (df *DataFetcher) findTargetFieldByMapping(sourceField string, fieldMappingRules interface{}) string {
	if fieldMappingRules == nil {
		slog.Debug("字段映射规则为空", "sourceField", sourceField)
		return ""
	}

	slog.Debug("开始查找字段映射", "sourceField", sourceField, "rulesType", fmt.Sprintf("%T", fieldMappingRules))

	// models.JSONB 实际上是 map[string]interface{} 的别名
	// 需要先转换为 map[string]interface{}
	var rulesMap map[string]interface{}

	// 尝试直接转换为 map[string]interface{}
	if rm, ok := fieldMappingRules.(map[string]interface{}); ok {
		rulesMap = rm
	} else if jsonb, ok := fieldMappingRules.(models.JSONB); ok {
		// models.JSONB 类型
		rulesMap = map[string]interface{}(jsonb)
	} else {
		slog.Debug("字段映射规则类型不支持", "type", fmt.Sprintf("%T", fieldMappingRules))
		return ""
	}

	// 解析 FieldMappingRules 结构
	if mappings, exists := rulesMap["mappings"]; exists {
		slog.Debug("找到mappings配置", "mappingsType", fmt.Sprintf("%T", mappings))
		if mappingSlice, ok := mappings.([]interface{}); ok {
			slog.Debug("mappings是数组", "length", len(mappingSlice))
			for i, mapping := range mappingSlice {
				if mappingMap, ok := mapping.(map[string]interface{}); ok {
					src := df.getStringFromMap(mappingMap, "source_field")
					target := df.getStringFromMap(mappingMap, "target_field")
					slog.Debug("检查映射", "index", i, "sourceField", src, "targetField", target, "匹配", src == sourceField)
					if src == sourceField {
						slog.Debug("找到字段映射", "sourceField", sourceField, "targetField", target)
						return target
					}
				}
			}
		} else {
			slog.Debug("mappings不是数组类型", "type", fmt.Sprintf("%T", mappings))
		}
	} else {
		slog.Debug("mappings配置不存在")
	}

	slog.Debug("未找到字段映射", "sourceField", sourceField)
	return ""
}
