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
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DataFetcher 数据获取器
type DataFetcher struct {
	db *gorm.DB
}

// NewDataFetcher 创建数据获取器
func NewDataFetcher(db *gorm.DB) *DataFetcher {
	return &DataFetcher{db: db}
}

// FetchSourceData 获取源数据
func (df *DataFetcher) FetchSourceData(request *SyncRequest, result *SyncExecutionResult) ([]SourceRecordInfo, error) {
	var sourceRecords []SourceRecordInfo

	// 从配置中获取源库配置
	sourceConfigs, err := df.parseSourceConfigs(request)
	if err != nil {
		return nil, fmt.Errorf("解析源库配置失败: %w", err)
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

	// 检查是否启用增量同步
	if config.IncrementalConfig != nil && config.IncrementalConfig.Enabled {
		return df.fetchIncrementalData(fullTableName, batchSize, config.IncrementalConfig)
	}

	return df.fetchDataInBatches(fullTableName, batchSize)
}

// fetchDataInBatches 分批获取数据
func (df *DataFetcher) fetchDataInBatches(fullTableName string, batchSize int) ([]map[string]interface{}, error) {
	var allRecords []map[string]interface{}
	offset := 0

	fmt.Printf("[DEBUG] 开始分批获取数据，表: %s, 批次大小: %d\n", fullTableName, batchSize)

	// 获取表的主键字段用于排序
	orderByClause := "ORDER BY id" // 默认排序

	// 尝试从表名解析出接口ID来获取主键字段
	if primaryKeys := df.getPrimaryKeysFromTableName(fullTableName); len(primaryKeys) > 0 {
		// 构建ORDER BY子句，使用双引号包围字段名
		var quotedKeys []string
		for _, key := range primaryKeys {
			quotedKeys = append(quotedKeys, fmt.Sprintf("\"%s\"", key))
		}
		orderByClause = fmt.Sprintf("ORDER BY %s", strings.Join(quotedKeys, ", "))
	}

	for {
		// 构建分页查询SQL
		sql := fmt.Sprintf("SELECT * FROM %s %s LIMIT %d OFFSET %d", fullTableName, orderByClause, batchSize, offset)

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

		fmt.Printf("[DEBUG] 获取批次数据 offset: %d, 记录数: %d\n", offset, len(batchRecords))

		allRecords = append(allRecords, batchRecords...)

		// 如果当前批次的记录数少于批次大小，说明已经是最后一批
		if len(batchRecords) < batchSize {
			break
		}

		offset += batchSize
	}

	fmt.Printf("[DEBUG] 总共获取记录数: %d\n", len(allRecords))
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

// fetchIncrementalData 获取增量数据
func (df *DataFetcher) fetchIncrementalData(fullTableName string, batchSize int, config *IncrementalConfig) ([]map[string]interface{}, error) {
	fmt.Printf("[DEBUG] 开始增量同步，表: %s, 增量字段: %s\n", fullTableName, config.IncrementalField)

	// 构建增量查询条件
	whereCondition, err := df.buildIncrementalCondition(config)
	if err != nil {
		return nil, fmt.Errorf("构建增量查询条件失败: %w", err)
	}

	var allRecords []map[string]interface{}
	offset := 0

	// 获取表的主键字段用于排序
	orderByClause := fmt.Sprintf("ORDER BY \"%s\"", config.IncrementalField) // 使用增量字段排序

	// 尝试获取主键字段，如果增量字段不是主键，则添加主键作为辅助排序
	if primaryKeys := df.getPrimaryKeysFromTableName(fullTableName); len(primaryKeys) > 0 {
		// 检查增量字段是否已经是主键之一
		isIncrementalFieldPrimaryKey := false
		for _, pk := range primaryKeys {
			if pk == config.IncrementalField {
				isIncrementalFieldPrimaryKey = true
				break
			}
		}

		// 如果增量字段不是主键，则添加主键作为辅助排序确保结果一致性
		if !isIncrementalFieldPrimaryKey {
			var quotedKeys []string
			for _, key := range primaryKeys {
				quotedKeys = append(quotedKeys, fmt.Sprintf("\"%s\"", key))
			}
			orderByClause = fmt.Sprintf("ORDER BY \"%s\", %s", config.IncrementalField, strings.Join(quotedKeys, ", "))
		}
	}

	for {
		// 构建增量查询SQL
		var sql string
		if whereCondition != "" {
			sql = fmt.Sprintf("SELECT * FROM %s WHERE %s %s LIMIT %d OFFSET %d",
				fullTableName, whereCondition, orderByClause, batchSize, offset)
		} else {
			// 如果没有增量条件（首次同步），使用普通查询
			sql = fmt.Sprintf("SELECT * FROM %s %s LIMIT %d OFFSET %d",
				fullTableName, orderByClause, batchSize, offset)
		}

		fmt.Printf("[DEBUG] 增量查询SQL: %s\n", sql)

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

		fmt.Printf("[DEBUG] 获取增量批次数据 offset: %d, 记录数: %d\n", offset, len(batchRecords))

		allRecords = append(allRecords, batchRecords...)

		// 如果当前批次的记录数少于批次大小，说明已经是最后一批
		if len(batchRecords) < batchSize {
			break
		}

		offset += batchSize
	}

	fmt.Printf("[DEBUG] 增量同步总共获取记录数: %d\n", len(allRecords))
	return allRecords, nil
}

// buildIncrementalCondition 构建增量查询条件
func (df *DataFetcher) buildIncrementalCondition(config *IncrementalConfig) (string, error) {
	if config.LastSyncValue == "" {
		// 首次同步，使用初始值
		if config.InitialValue != "" {
			return fmt.Sprintf("%s %s '%s'", config.IncrementalField, config.CompareOperator, config.InitialValue), nil
		}
		// 没有初始值，返回空条件（获取所有数据）
		return "", nil
	}

	// 构建基础增量条件
	condition := fmt.Sprintf("%s %s '%s'", config.IncrementalField, config.CompareOperator, config.LastSyncValue)

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

// getPrimaryKeysFromTableName 从表名获取主键字段
func (df *DataFetcher) getPrimaryKeysFromTableName(fullTableName string) []string {
	// 从完整表名中解析schema和table名
	parts := strings.Split(fullTableName, ".")
	if len(parts) != 2 {
		return []string{"id"} // 默认主键
	}

	schemaName := parts[0]
	tableName := parts[1]

	// 通过schema名和table名查找对应的主题接口
	var thematicInterface models.ThematicInterface
	if err := df.db.Joins("JOIN thematic_libraries ON thematic_interfaces.library_id = thematic_libraries.id").
		Where("thematic_libraries.name_en = ? AND thematic_interfaces.name_en = ?", schemaName, tableName).
		First(&thematicInterface).Error; err != nil {
		fmt.Printf("[DEBUG] 无法找到主题接口，使用默认主键: %v\n", err)
		return []string{"id"} // 默认主键
	}

	// 解析主键字段配置
	return df.getThematicPrimaryKeyFields(thematicInterface.ID)
}

// getThematicPrimaryKeyFields 获取主题接口的主键字段列表
func (df *DataFetcher) getThematicPrimaryKeyFields(thematicInterfaceID string) []string {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := df.db.First(&thematicInterface, "id = ?", thematicInterfaceID).Error; err != nil {
		fmt.Printf("[DEBUG] 获取主题接口信息失败: %v, 使用默认主键\n", err)
		return []string{"id"}
	}

	var primaryKeys []string

	// 从TableFieldsConfig中解析主键字段
	if len(thematicInterface.TableFieldsConfig) > 0 {
		var tableFields []models.TableField
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", thematicInterface.TableFieldsConfig)), &tableFields); err == nil {
			for _, field := range tableFields {
				if field.IsPrimaryKey {
					primaryKeys = append(primaryKeys, field.NameEn)
				}
			}
		}
	}

	// 如果没有主键，使用默认的id字段
	if len(primaryKeys) == 0 {
		primaryKeys = []string{"id"}
	}

	return primaryKeys
}
