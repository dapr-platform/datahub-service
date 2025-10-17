/*
 * @module service/interface_executor/field_mapping
 * @description 字段映射和数据转换逻辑，处理数据库插入和字段格式转换
 * @architecture 转换器模式 - 提供灵活的字段映射和数据转换功能
 * @documentReference ai_docs/interface_executor.md
 * @stateFlow 字段映射配置解析 -> 数据转换 -> 类型处理 -> 数据库插入
 * @rules 确保字段映射的准确性和数据转换的完整性
 * @dependencies gorm.io/gorm, github.com/spf13/cast
 * @refs executor.go, execute_operations.go
 */

package interface_executor

import (
	"log/slog"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// FieldMapper 字段映射器
type FieldMapper struct {
	// 字段类型映射缓存，提高性能
	fieldTypeCache map[string]map[string]string // interfaceID -> fieldName -> dataType
}

// NewFieldMapper 创建字段映射器
func NewFieldMapper() *FieldMapper {
	return &FieldMapper{
		fieldTypeCache: make(map[string]map[string]string),
	}
}

// FieldTypeInfo 字段类型信息
type FieldTypeInfo struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

// buildFieldTypeMapping 构建字段类型映射
func (fm *FieldMapper) buildFieldTypeMapping(interfaceInfo InterfaceInfo) map[string]string {
	interfaceID := interfaceInfo.GetID()

	// 检查缓存
	if cached, exists := fm.fieldTypeCache[interfaceID]; exists {
		return cached
	}

	fieldTypeMap := make(map[string]string)

	// 方法1：从TableFieldsConfig获取字段类型
	tableFieldsConfig := interfaceInfo.GetTableFieldsConfig()
	if len(tableFieldsConfig) > 0 {
		for _, fieldConfigInterface := range tableFieldsConfig {
			if fieldConfig, ok := fieldConfigInterface.(map[string]interface{}); ok {
				// 尝试从fields数组中获取字段信息
				if fieldsData, exists := fieldConfig["fields"]; exists {
					if fieldsArray, ok := fieldsData.([]interface{}); ok {
						for _, fieldData := range fieldsArray {
							if fieldMap, ok := fieldData.(map[string]interface{}); ok {
								if fieldName, ok := fieldMap["field_name"].(string); ok {
									if fieldType, ok := fieldMap["field_type"].(string); ok {
										fieldTypeMap[fieldName] = fieldType
									}
								}
							}
						}
					}
				}

				// 直接从配置对象获取字段信息
				if fieldName, ok := fieldConfig["field_name"].(string); ok {
					if fieldType, ok := fieldConfig["field_type"].(string); ok {
						fieldTypeMap[fieldName] = fieldType
					}
				}
			}
		}
	}

	// 方法2：从InterfaceInfo的字段信息获取（如果方法1没有获取到）
	// 这需要InterfaceInfo提供字段信息的访问方法
	// 由于当前InterfaceInfo接口没有直接提供Fields访问方法，我们先跳过这部分

	// 缓存结果
	fm.fieldTypeCache[interfaceID] = fieldTypeMap

	return fieldTypeMap
}

// getFieldDataType 获取字段的数据类型
func (fm *FieldMapper) getFieldDataType(fieldName string, interfaceInfo InterfaceInfo) string {
	fieldTypeMap := fm.buildFieldTypeMapping(interfaceInfo)
	if dataType, exists := fieldTypeMap[fieldName]; exists {
		return strings.ToLower(dataType)
	}

	// 如果没有找到配置，尝试通过字段名推断（作为后备方案）
	fieldNameLower := strings.ToLower(fieldName)
	if strings.Contains(fieldNameLower, "time") ||
		strings.Contains(fieldNameLower, "date") ||
		strings.Contains(fieldNameLower, "created_at") ||
		strings.Contains(fieldNameLower, "updated_at") {
		return "timestamp"
	}

	if strings.Contains(fieldNameLower, "id") && strings.HasSuffix(fieldNameLower, "id") {
		return "varchar"
	}

	// 默认返回字符串类型
	return "varchar"
}

// UpdateTableData 更新表数据
func (fm *FieldMapper) UpdateTableData(ctx context.Context, db *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	// 构造表名
	schemaName := interfaceInfo.GetSchemaName()
	tableName := interfaceInfo.GetTableName()
	var fullTableName string
	if schemaName != "" {
		fullTableName = fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)
	} else {
		fullTableName = fmt.Sprintf(`"%s"`, tableName)
	}

	slog.Debug("FieldMapper.UpdateTableData - 开始更新表数据")
	slog.Debug("UpdateTableData - 表名", "value", fullTableName)
	slog.Debug("UpdateTableData - 数据行数", "count", len(data))

	// 打印parseConfig信息
	parseConfig := interfaceInfo.GetParseConfig()
	slog.Debug("UpdateTableData - parseConfig", "data", parseConfig)

	if len(data) > 0 {
		slog.Debug("UpdateTableData - 第一行数据示例", "data", data[0])
	}

	// 这里应该根据接口类型（实时/批量）和配置来决定更新策略
	// 简化实现：清空表后插入新数据

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("UpdateTableData - 事务回滚，原因", "error", r)
			tx.Rollback()
		}
	}()

	// 清空现有数据
	deleteSQL := fmt.Sprintf("DELETE FROM %s", fullTableName)
	slog.Debug("UpdateTableData - 清空表SQL", "value", deleteSQL)

	if err := tx.Exec(deleteSQL).Error; err != nil {
		slog.Error("UpdateTableData - 清空表数据失败", "error", err)
		tx.Rollback()
		return 0, fmt.Errorf("清空表数据失败: %w", err)
	}

	// 插入新数据
	var insertedRows int64
	for i, row := range data {
		// 只对第一行数据输出详细调试信息
		if i == 0 {
			slog.Debug("UpdateTableData - 处理第 %d 行数据", "data", i+1, row)
		} else if i%100 == 0 {
			slog.Debug("UpdateTableData - 已处理", "count", i+1) // 行数据...
		}

		// 应用parseConfig中的fieldMapping
		mappedRow := fm.ApplyFieldMapping(row, parseConfig, i == 0)
		if i == 0 {
			slog.Debug("UpdateTableData - 字段映射后的数据", "data", mappedRow)
		}

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，基于字段配置
			processedVal := fm.ProcessValueForDatabase(col, val, interfaceInfo, i == 0)
			values = append(values, processedVal)
		}

		// 修复SQL格式错误 - 使用strings.Join而不是fmt.Sprintf
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if i == 0 {
			slog.Debug("UpdateTableData - 插入SQL", "value", insertSQL)
			slog.Debug("UpdateTableData - 插入参数", "data", values)
		}

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			slog.Error("UpdateTableData - 插入数据失败", "error", err)
			slog.Error("UpdateTableData - 失败的SQL", "message", insertSQL)
			slog.Error("UpdateTableData - 失败的参数", "data", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		slog.Error("UpdateTableData - 提交事务失败", "error", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	slog.Debug("UpdateTableData - 成功插入", "count", insertedRows) // 行数据
	return insertedRows, nil
}

// InsertBatchData 插入批量数据
func (fm *FieldMapper) InsertBatchData(ctx context.Context, db *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	slog.Debug("FieldMapper.InsertBatchData - 开始插入批量数据到表", "value", fullTableName)
	slog.Debug("InsertBatchData - 数据行数", "count", len(data))

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("InsertBatchData - 事务回滚，原因", "error", r)
			tx.Rollback()
		}
	}()

	// 插入数据
	var insertedRows int64
	for i, row := range data {
		slog.Debug("InsertBatchData - 处理第 %d 行数据", "data", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig)
		slog.Debug("InsertBatchData - 字段映射后的数据", "data", mappedRow)

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，基于字段配置
			processedVal := fm.ProcessValueForDatabase(col, val, interfaceInfo)
			values = append(values, processedVal)
		}

		// 修复SQL格式错误 - 使用strings.Join而不是fmt.Sprintf
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		slog.Debug("InsertBatchData - 插入SQL", "value", insertSQL)
		slog.Debug("InsertBatchData - 插入参数", "data", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			slog.Error("InsertBatchData - 插入数据失败", "error", err)
			slog.Error("InsertBatchData - 失败的SQL", "message", insertSQL)
			slog.Error("InsertBatchData - 失败的参数", "data", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		slog.Error("InsertBatchData - 提交事务失败", "error", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	slog.Debug("InsertBatchData - 成功插入", "count", insertedRows) // 行数据
	return insertedRows, nil
}

// InsertBatchDataWithTx 使用提供的事务插入批量数据（不创建新事务）
func (fm *FieldMapper) InsertBatchDataWithTx(ctx context.Context, tx *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	slog.Debug("FieldMapper.InsertBatchDataWithTx - 开始插入批量数据到表", "value", fullTableName)
	slog.Debug("InsertBatchDataWithTx - 数据行数", "count", len(data))

	// 插入数据（使用提供的事务）
	var insertedRows int64
	for i, row := range data {
		slog.Debug("InsertBatchDataWithTx - 处理第 %d 行数据", "data", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig)
		slog.Debug("InsertBatchDataWithTx - 字段映射后的数据", "data", mappedRow)

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，基于字段配置
			processedVal := fm.ProcessValueForDatabase(col, val, interfaceInfo)
			values = append(values, processedVal)
		}

		// 构建插入SQL
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		slog.Debug("InsertBatchDataWithTx - 插入SQL", "value", insertSQL)
		slog.Debug("InsertBatchDataWithTx - 插入参数", "data", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			slog.Error("InsertBatchDataWithTx - 插入数据失败", "error", err)
			slog.Error("InsertBatchDataWithTx - 失败的SQL", "message", insertSQL)
			slog.Error("InsertBatchDataWithTx - 失败的参数", "data", values)
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	slog.Debug("InsertBatchDataWithTx - 成功插入", "count", insertedRows) // 行数据
	return insertedRows, nil
}

// ApplyFieldMapping 应用字段映射配置
func (fm *FieldMapper) ApplyFieldMapping(row map[string]interface{}, parseConfig map[string]interface{}, debugLog ...bool) map[string]interface{} {
	debug := len(debugLog) > 0 && debugLog[0]
	if debug {
		slog.Debug("FieldMapper.ApplyFieldMapping - 原始数据", "data", row)
		slog.Debug("ApplyFieldMapping - parseConfig", "data", parseConfig)
	}

	// 如果没有parseConfig，直接返回原始数据
	if parseConfig == nil {
		if debug {
			slog.Debug("ApplyFieldMapping - parseConfig为空，返回原始数据")
		}
		return row
	}

	// 获取fieldMapping配置
	fieldMappingInterface, exists := parseConfig["fieldMapping"]
	if !exists {
		if debug {
			slog.Debug("ApplyFieldMapping - 没有fieldMapping配置，返回原始数据")
		}
		return row
	}

	if debug {
		slog.Debug("ApplyFieldMapping - fieldMapping原始配置", "data", fieldMappingInterface)
	}

	// 支持两种格式：数组格式（新）和对象格式（旧）
	var fieldMappingArray []interface{}
	var fieldMappingMap map[string]interface{}
	var isArrayFormat bool

	// 尝试解析为数组格式（新格式）
	if mappingArray, ok := fieldMappingInterface.([]interface{}); ok {
		fieldMappingArray = mappingArray
		isArrayFormat = true
		if debug {
			slog.Debug("ApplyFieldMapping - 使用数组格式fieldMapping，条目数", "count", len(fieldMappingArray))
		}
	} else if mappingMap, ok := fieldMappingInterface.(map[string]interface{}); ok {
		// 兼容旧的对象格式
		fieldMappingMap = mappingMap
		isArrayFormat = false
		if debug {
			slog.Debug("ApplyFieldMapping - 使用对象格式fieldMapping（兼容模式）")
		}
	} else {
		if debug {
			slog.Debug("ApplyFieldMapping - fieldMapping格式不支持，返回原始数据")
		}
		return row
	}

	// 应用字段映射
	mappedRow := make(map[string]interface{})

	if isArrayFormat {
		// 处理新的数组格式：[{"source": "age", "target": "age"}, ...]
		// 构建源字段到目标字段的映射表
		sourceToTargetMap := make(map[string]string)
		for _, mappingItem := range fieldMappingArray {
			if mappingObj, ok := mappingItem.(map[string]interface{}); ok {
				source := cast.ToString(mappingObj["source"])
				target := cast.ToString(mappingObj["target"])
				if source != "" && target != "" {
					sourceToTargetMap[source] = target
					if debug {
						slog.Debug("ApplyFieldMapping - 映射规则: %s -> %s\n", source, target)
					}
				}
			}
		}

		// 遍历原始数据的每个字段，应用映射
		for sourceField, value := range row {
			var targetField string

			// 查找映射目标字段
			if target, exists := sourceToTargetMap[sourceField]; exists {
				targetField = target
			} else {
				// 如果没找到映射，使用原字段名
				targetField = sourceField
			}

			mappedRow[targetField] = value
			if debug {
				slog.Debug("ApplyFieldMapping - 字段映射: %s -> %s, 值: %v\n", sourceField, targetField, value)
			}
		}

	} else {
		// 处理旧的对象格式：{"age": "age", "email": "email", ...}（兼容模式）
		for sourceField, value := range row {
			var targetField string

			// 在fieldMapping中查找源字段对应的目标字段
			for target, source := range fieldMappingMap {
				if sourceStr, ok := source.(string); ok && sourceStr == sourceField {
					targetField = target
					break
				}
			}

			// 如果没找到映射，使用原字段名
			if targetField == "" {
				targetField = sourceField
			}

			mappedRow[targetField] = value
			if debug {
				slog.Debug("ApplyFieldMapping - 字段映射（兼容模式）: %s -> %s, 值: %v\n", sourceField, targetField, value)
			}
		}
	}

	if debug {
		slog.Debug("ApplyFieldMapping - 映射后数据", "data", mappedRow)
	}
	return mappedRow
}

// ProcessValueForDatabase 基于字段配置处理数据库值，支持多种数据类型转换
func (fm *FieldMapper) ProcessValueForDatabase(columnName string, value interface{}, interfaceInfo InterfaceInfo, debugLog ...bool) interface{} {
	if value == nil {
		return value
	}

	debug := len(debugLog) > 0 && debugLog[0]
	if debug {
		slog.Debug("FieldMapper.ProcessValueForDatabase - 处理字段: %s, 原始值: %+v, 类型: %T\n", columnName, value, value)
	}

	// 获取字段的数据类型
	dataType := fm.getFieldDataType(columnName, interfaceInfo)
	if debug {
		slog.Debug("ProcessValueForDatabase - 字段 %s 的数据类型", "value", columnName, dataType)
	}

	// 根据数据类型进行转换
	return fm.convertValueByDataType(value, dataType, columnName, debug)
}

// convertValueByDataType 根据数据类型转换值
func (fm *FieldMapper) convertValueByDataType(value interface{}, dataType, columnName string, debug bool) interface{} {
	switch strings.ToLower(dataType) {
	case "timestamp", "datetime", "timestamptz":
		return fm.convertToTimestamp(value, debug)
	case "date":
		return fm.convertToDate(value, debug)
	case "time":
		return fm.convertToTime(value, debug)
	case "int", "integer", "int4":
		return fm.convertToInteger(value, debug)
	case "bigint", "int8":
		return fm.convertToBigInt(value, debug)
	case "decimal", "numeric", "float", "double", "float8":
		return fm.convertToFloat(value, debug)
	case "boolean", "bool":
		return fm.convertToBoolean(value, debug)
	case "varchar", "text", "char", "string":
		return fm.convertToString(value, debug)
	case "json", "jsonb":
		return fm.convertToJSON(value, debug)
	default:
		// 未知类型，使用字符串转换
		if debug {
			slog.Debug("convertValueByDataType - 未知数据类型 %s，使用字符串转换\n", dataType)
		}
		return fm.convertToString(value, debug)
	}
}

// convertToTimestamp 转换为时间戳格式
func (fm *FieldMapper) convertToTimestamp(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case time.Time:
		formatted := v.Format("2006-01-02 15:04:05.000")
		if debug {
			slog.Debug("convertToTimestamp - time.Time转换: %s -> %s\n", v.String(), formatted)
		}
		return formatted
	case string:
		// 尝试多种时间格式解析
		timeFormats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05.000",
			"2006-01-02",
		}

		for _, format := range timeFormats {
			if parsedTime, err := time.Parse(format, v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				if debug {
					slog.Debug("convertToTimestamp - 字符串时间转换(%s): %s -> %s\n", format, v, formatted)
				}
				return formatted
			}
		}

		if debug {
			slog.Debug("convertToTimestamp - 无法解析时间字符串，返回原值", "value", v)
		}
		return v
	default:
		// 尝试转换为字符串再解析
		if timeStr := cast.ToString(v); timeStr != "" {
			return fm.convertToTimestamp(timeStr, debug)
		}
		if debug {
			slog.Debug("convertToTimestamp - 无法处理的时间类型，返回原值", "value", v)
		}
		return v
	}
}

// convertToDate 转换为日期格式
func (fm *FieldMapper) convertToDate(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case time.Time:
		formatted := v.Format("2006-01-02")
		if debug {
			slog.Debug("convertToDate - time.Time转换: %s -> %s\n", v.String(), formatted)
		}
		return formatted
	case string:
		// 尝试解析日期
		dateFormats := []string{
			"2006-01-02",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			time.RFC3339,
		}

		for _, format := range dateFormats {
			if parsedTime, err := time.Parse(format, v); err == nil {
				formatted := parsedTime.Format("2006-01-02")
				if debug {
					slog.Debug("convertToDate - 字符串日期转换(%s): %s -> %s\n", format, v, formatted)
				}
				return formatted
			}
		}

		if debug {
			slog.Debug("convertToDate - 无法解析日期字符串，返回原值", "value", v)
		}
		return v
	default:
		if timeStr := cast.ToString(v); timeStr != "" {
			return fm.convertToDate(timeStr, debug)
		}
		return v
	}
}

// convertToTime 转换为时间格式
func (fm *FieldMapper) convertToTime(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case time.Time:
		formatted := v.Format("15:04:05")
		if debug {
			slog.Debug("convertToTime - time.Time转换: %s -> %s\n", v.String(), formatted)
		}
		return formatted
	case string:
		// 尝试解析时间
		if _, err := time.Parse("15:04:05", v); err == nil {
			return v
		}
		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", v); err == nil {
			formatted := parsedTime.Format("15:04:05")
			if debug {
				slog.Debug("convertToTime - 字符串时间转换: %s -> %s\n", v, formatted)
			}
			return formatted
		}
		return v
	default:
		return cast.ToString(v)
	}
}

// convertToInteger 转换为整数
func (fm *FieldMapper) convertToInteger(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case int, int32, int64:
		return v
	case float32, float64:
		intVal := int(cast.ToFloat64(v))
		if debug {
			slog.Debug("convertToInteger - 浮点数转整数: %v -> %d\n", v, intVal)
		}
		return intVal
	case string:
		if intVal, err := strconv.Atoi(v); err == nil {
			if debug {
				slog.Debug("convertToInteger - 字符串转整数: %s -> %d\n", v, intVal)
			}
			return intVal
		}
		// 尝试先转换为浮点数再转整数
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			intVal := int(floatVal)
			if debug {
				slog.Debug("convertToInteger - 字符串(浮点)转整数: %s -> %d\n", v, intVal)
			}
			return intVal
		}
		if debug {
			slog.Debug("convertToInteger - 无法转换字符串为整数，返回原值", "value", v)
		}
		return v
	default:
		intVal := cast.ToInt(v)
		if debug {
			slog.Debug("convertToInteger - 其他类型转整数: %v -> %d\n", v, intVal)
		}
		return intVal
	}
}

// convertToBigInt 转换为大整数
func (fm *FieldMapper) convertToBigInt(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case int64:
		return v
	case int, int32:
		return int64(cast.ToInt64(v))
	case float32, float64:
		bigIntVal := int64(cast.ToFloat64(v))
		if debug {
			slog.Debug("convertToBigInt - 浮点数转大整数: %v -> %d\n", v, bigIntVal)
		}
		return bigIntVal
	case string:
		if bigIntVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			if debug {
				slog.Debug("convertToBigInt - 字符串转大整数: %s -> %d\n", v, bigIntVal)
			}
			return bigIntVal
		}
		return v
	default:
		bigIntVal := cast.ToInt64(v)
		if debug {
			slog.Debug("convertToBigInt - 其他类型转大整数: %v -> %d\n", v, bigIntVal)
		}
		return bigIntVal
	}
}

// convertToFloat 转换为浮点数
func (fm *FieldMapper) convertToFloat(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case float32, float64:
		return v
	case int, int32, int64:
		floatVal := cast.ToFloat64(v)
		if debug {
			slog.Debug("convertToFloat - 整数转浮点数: %v -> %f\n", v, floatVal)
		}
		return floatVal
	case string:
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			if debug {
				slog.Debug("convertToFloat - 字符串转浮点数: %s -> %f\n", v, floatVal)
			}
			return floatVal
		}
		return v
	default:
		floatVal := cast.ToFloat64(v)
		if debug {
			slog.Debug("convertToFloat - 其他类型转浮点数: %v -> %f\n", v, floatVal)
		}
		return floatVal
	}
}

// convertToBoolean 转换为布尔值
func (fm *FieldMapper) convertToBoolean(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		lowerV := strings.ToLower(v)
		switch lowerV {
		case "true", "1", "yes", "y", "on":
			if debug {
				slog.Debug("convertToBoolean - 字符串转布尔值: %s -> true\n", v)
			}
			return true
		case "false", "0", "no", "n", "off":
			if debug {
				slog.Debug("convertToBoolean - 字符串转布尔值: %s -> false\n", v)
			}
			return false
		default:
			return v
		}
	case int, int32, int64:
		boolVal := cast.ToInt64(v) != 0
		if debug {
			slog.Debug("convertToBoolean - 整数转布尔值: %v -> %t\n", v, boolVal)
		}
		return boolVal
	default:
		boolVal := cast.ToBool(v)
		if debug {
			slog.Debug("convertToBoolean - 其他类型转布尔值: %v -> %t\n", v, boolVal)
		}
		return boolVal
	}
}

// convertToString 转换为字符串
func (fm *FieldMapper) convertToString(value interface{}, debug bool) interface{} {
	strVal := cast.ToString(value)
	if debug && fmt.Sprintf("%v", value) != strVal {
		slog.Debug("convertToString - 类型转字符串: %v -> %s\n", value, strVal)
	}
	return strVal
}

// convertToJSON 转换为JSON格式
func (fm *FieldMapper) convertToJSON(value interface{}, debug bool) interface{} {
	switch v := value.(type) {
	case string:
		// 如果已经是字符串，检查是否是有效的JSON
		return v
	case map[string]interface{}, []interface{}:
		// 如果是map或slice，直接返回
		return v
	default:
		// 其他类型转换为字符串
		return cast.ToString(v)
	}
}

// UpsertDataToTable 执行数据的UPSERT操作（增量同步）
func (fm *FieldMapper) UpsertDataToTable(db *gorm.DB, errorHandler *ErrorHandler, data []map[string]interface{}, schemaName, tableName string) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	var fullTableName string
	if schemaName != "" {
		fullTableName = fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)
	} else {
		fullTableName = fmt.Sprintf(`"%s"`, tableName)
	}
	var insertedRows int64 = 0

	// 使用错误处理器包装事务操作
	err := errorHandler.WrapWithTransaction(db, func(tx *gorm.DB) error {
		// 分批处理数据，避免单个事务过大
		batchSize := 1000
		for i := 0; i < len(data); i += batchSize {
			end := i + batchSize
			if end > len(data) {
				end = len(data)
			}
			batch := data[i:end]

			// 处理单个批次
			if err := fm.processBatch(tx, batch, fullTableName); err != nil {
				return fmt.Errorf("处理批次 %d-%d 失败: %w", i, end, err)
			}
			insertedRows += int64(len(batch))
		}
		return nil
	})

	if err != nil {
		errorDetail := errorHandler.HandleError(
			context.Background(),
			err,
			ErrorTypeTransaction,
			fmt.Sprintf("upsert data to table %s", fullTableName),
		)
		return 0, fmt.Errorf("UPSERT操作失败: %s", errorDetail.Message)
	}

	return insertedRows, nil
}

// processBatch 处理单个数据批次
func (fm *FieldMapper) processBatch(tx *gorm.DB, batch []map[string]interface{}, fullTableName string) error {
	for _, row := range batch {
		if err := fm.processRow(tx, row, fullTableName); err != nil {
			return fmt.Errorf("处理行数据失败: %w", err)
		}
	}
	return nil
}

// processRow 处理单行数据
func (fm *FieldMapper) processRow(tx *gorm.DB, row map[string]interface{}, fullTableName string) error {
	if len(row) == 0 {
		return nil
	}

	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	values := make([]interface{}, 0, len(row))

	for col, val := range row {
		// 数据验证
		if col == "" {
			return fmt.Errorf("列名不能为空")
		}

		columns = append(columns, fmt.Sprintf(`"%s"`, col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	if len(columns) == 0 {
		return nil
	}

	// 构建插入SQL（这里简化为INSERT，实际应该实现UPSERT逻辑）
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		fullTableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	if err := tx.Exec(insertSQL, values...).Error; err != nil {
		return fmt.Errorf("执行插入SQL失败: %w", err)
	}

	return nil
}

// ReplaceTableData 替换表数据（全量同步）
func (fm *FieldMapper) ReplaceTableData(ctx context.Context, db *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	// 构造表名
	schemaName := interfaceInfo.GetSchemaName()
	tableName := interfaceInfo.GetTableName()
	var fullTableName string
	if schemaName != "" {
		fullTableName = fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)
	} else {
		fullTableName = fmt.Sprintf(`"%s"`, tableName)
	}

	slog.Debug("FieldMapper.ReplaceTableData - 开始替换表数据")
	slog.Debug("ReplaceTableData - 表名", "value", fullTableName)
	slog.Debug("ReplaceTableData - 数据行数", "count", len(data))

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("ReplaceTableData - 事务回滚，原因", "error", r)
			tx.Rollback()
		}
	}()

	// 清空现有数据
	deleteSQL := fmt.Sprintf("DELETE FROM %s", fullTableName)
	slog.Debug("ReplaceTableData - 清空表SQL", "value", deleteSQL)

	if err := tx.Exec(deleteSQL).Error; err != nil {
		slog.Error("ReplaceTableData - 清空表数据失败", "error", err)
		tx.Rollback()
		return 0, fmt.Errorf("清空表数据失败: %w", err)
	}

	// 插入新数据
	var insertedRows int64
	for i, row := range data {
		// 只对第一行数据输出详细调试信息
		if i == 0 {
			slog.Debug("ReplaceTableData - 处理第 %d 行数据", "data", i+1, row)
		} else if i%100 == 0 {
			slog.Debug("ReplaceTableData - 已处理", "count", i+1) // 行数据...
		}

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig, i == 0)
		if i == 0 {
			slog.Debug("ReplaceTableData - 字段映射后的数据", "data", mappedRow)
		}

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，基于字段配置
			processedVal := fm.ProcessValueForDatabase(col, val, interfaceInfo)
			values = append(values, processedVal)
		}

		// 构建插入SQL
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if i == 0 {
			slog.Debug("ReplaceTableData - 插入SQL", "value", insertSQL)
			slog.Debug("ReplaceTableData - 插入参数", "data", values)
		}

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			slog.Error("ReplaceTableData - 插入数据失败", "error", err)
			slog.Error("ReplaceTableData - 失败的SQL", "message", insertSQL)
			slog.Error("ReplaceTableData - 失败的参数", "data", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		slog.Error("ReplaceTableData - 提交事务失败", "error", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	slog.Debug("ReplaceTableData - 成功插入", "count", insertedRows) // 行数据
	return insertedRows, nil
}

// UpsertBatchDataWithTx 使用提供的事务进行批量UPSERT操作（增量同步）
func (fm *FieldMapper) UpsertBatchDataWithTx(ctx context.Context, tx *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	slog.Debug("FieldMapper.UpsertBatchDataWithTx - 开始UPSERT批量数据到表", "value", fullTableName)
	slog.Debug("UpsertBatchDataWithTx - 数据行数", "count", len(data))

	// 处理数据（使用提供的事务）
	var processedRows int64
	for i, row := range data {
		slog.Debug("UpsertBatchDataWithTx - 处理第 %d 行数据", "data", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig)
		slog.Debug("UpsertBatchDataWithTx - 字段映射后的数据", "data", mappedRow)

		// 构建UPSERT SQL（这里简化为INSERT，实际应该实现UPSERT逻辑）
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，基于字段配置
			processedVal := fm.ProcessValueForDatabase(col, val, interfaceInfo)
			values = append(values, processedVal)
		}

		// 构建INSERT SQL（简化版，实际应该是UPSERT）
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		slog.Debug("UpsertBatchDataWithTx - 插入SQL", "value", insertSQL)
		slog.Debug("UpsertBatchDataWithTx - 插入参数", "data", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			slog.Error("UpsertBatchDataWithTx - 插入数据失败", "error", err)
			slog.Error("UpsertBatchDataWithTx - 失败的SQL", "message", insertSQL)
			slog.Error("UpsertBatchDataWithTx - 失败的参数", "data", values)
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		processedRows++
	}

	slog.Debug("UpsertBatchDataWithTx - 成功处理", "count", processedRows) // 行数据
	return processedRows, nil
}
