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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// FieldMapper 字段映射器
type FieldMapper struct{}

// NewFieldMapper 创建字段映射器
func NewFieldMapper() *FieldMapper {
	return &FieldMapper{}
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

	fmt.Printf("[DEBUG] FieldMapper.UpdateTableData - 开始更新表数据\n")
	fmt.Printf("[DEBUG] UpdateTableData - 表名: %s\n", fullTableName)
	fmt.Printf("[DEBUG] UpdateTableData - 数据行数: %d\n", len(data))

	// 打印parseConfig信息
	parseConfig := interfaceInfo.GetParseConfig()
	fmt.Printf("[DEBUG] UpdateTableData - parseConfig: %+v\n", parseConfig)

	if len(data) > 0 {
		fmt.Printf("[DEBUG] UpdateTableData - 第一行数据示例: %+v\n", data[0])
	}

	// 这里应该根据接口类型（实时/批量）和配置来决定更新策略
	// 简化实现：清空表后插入新数据

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR] UpdateTableData - 事务回滚，原因: %v\n", r)
			tx.Rollback()
		}
	}()

	// 清空现有数据
	deleteSQL := fmt.Sprintf("DELETE FROM %s", fullTableName)
	fmt.Printf("[DEBUG] UpdateTableData - 清空表SQL: %s\n", deleteSQL)

	if err := tx.Exec(deleteSQL).Error; err != nil {
		fmt.Printf("[ERROR] UpdateTableData - 清空表数据失败: %v\n", err)
		tx.Rollback()
		return 0, fmt.Errorf("清空表数据失败: %w", err)
	}

	// 插入新数据
	var insertedRows int64
	for i, row := range data {
		// 只对第一行数据输出详细调试信息
		if i == 0 {
			fmt.Printf("[DEBUG] UpdateTableData - 处理第 %d 行数据: %+v\n", i+1, row)
		} else if i%100 == 0 {
			fmt.Printf("[DEBUG] UpdateTableData - 已处理 %d 行数据...\n", i+1)
		}

		// 应用parseConfig中的fieldMapping
		mappedRow := fm.ApplyFieldMapping(row, parseConfig, i == 0)
		if i == 0 {
			fmt.Printf("[DEBUG] UpdateTableData - 字段映射后的数据: %+v\n", mappedRow)
		}

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := fm.ProcessValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 修复SQL格式错误 - 使用strings.Join而不是fmt.Sprintf
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if i == 0 {
			fmt.Printf("[DEBUG] UpdateTableData - 插入SQL: %s\n", insertSQL)
			fmt.Printf("[DEBUG] UpdateTableData - 插入参数: %+v\n", values)
		}

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] UpdateTableData - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] UpdateTableData - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] UpdateTableData - 失败的参数: %+v\n", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("[ERROR] UpdateTableData - 提交事务失败: %v\n", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("[DEBUG] UpdateTableData - 成功插入 %d 行数据\n", insertedRows)
	return insertedRows, nil
}

// InsertBatchData 插入批量数据
func (fm *FieldMapper) InsertBatchData(ctx context.Context, db *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	fmt.Printf("[DEBUG] FieldMapper.InsertBatchData - 开始插入批量数据到表: %s\n", fullTableName)
	fmt.Printf("[DEBUG] InsertBatchData - 数据行数: %d\n", len(data))

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR] InsertBatchData - 事务回滚，原因: %v\n", r)
			tx.Rollback()
		}
	}()

	// 插入数据
	var insertedRows int64
	for i, row := range data {
		fmt.Printf("[DEBUG] InsertBatchData - 处理第 %d 行数据: %+v\n", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig)
		fmt.Printf("[DEBUG] InsertBatchData - 字段映射后的数据: %+v\n", mappedRow)

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := fm.ProcessValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 修复SQL格式错误 - 使用strings.Join而不是fmt.Sprintf
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		fmt.Printf("[DEBUG] InsertBatchData - 插入SQL: %s\n", insertSQL)
		fmt.Printf("[DEBUG] InsertBatchData - 插入参数: %+v\n", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] InsertBatchData - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] InsertBatchData - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] InsertBatchData - 失败的参数: %+v\n", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("[ERROR] InsertBatchData - 提交事务失败: %v\n", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("[DEBUG] InsertBatchData - 成功插入 %d 行数据\n", insertedRows)
	return insertedRows, nil
}

// InsertBatchDataWithTx 使用提供的事务插入批量数据（不创建新事务）
func (fm *FieldMapper) InsertBatchDataWithTx(ctx context.Context, tx *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	fmt.Printf("[DEBUG] FieldMapper.InsertBatchDataWithTx - 开始插入批量数据到表: %s\n", fullTableName)
	fmt.Printf("[DEBUG] InsertBatchDataWithTx - 数据行数: %d\n", len(data))

	// 插入数据（使用提供的事务）
	var insertedRows int64
	for i, row := range data {
		fmt.Printf("[DEBUG] InsertBatchDataWithTx - 处理第 %d 行数据: %+v\n", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig)
		fmt.Printf("[DEBUG] InsertBatchDataWithTx - 字段映射后的数据: %+v\n", mappedRow)

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := fm.ProcessValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 构建插入SQL
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		fmt.Printf("[DEBUG] InsertBatchDataWithTx - 插入SQL: %s\n", insertSQL)
		fmt.Printf("[DEBUG] InsertBatchDataWithTx - 插入参数: %+v\n", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] InsertBatchDataWithTx - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] InsertBatchDataWithTx - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] InsertBatchDataWithTx - 失败的参数: %+v\n", values)
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	fmt.Printf("[DEBUG] InsertBatchDataWithTx - 成功插入 %d 行数据\n", insertedRows)
	return insertedRows, nil
}

// ApplyFieldMapping 应用字段映射配置
func (fm *FieldMapper) ApplyFieldMapping(row map[string]interface{}, parseConfig map[string]interface{}, debugLog ...bool) map[string]interface{} {
	debug := len(debugLog) > 0 && debugLog[0]
	if debug {
		fmt.Printf("[DEBUG] FieldMapper.ApplyFieldMapping - 原始数据: %+v\n", row)
		fmt.Printf("[DEBUG] ApplyFieldMapping - parseConfig: %+v\n", parseConfig)
	}

	// 如果没有parseConfig，直接返回原始数据
	if parseConfig == nil {
		if debug {
			fmt.Printf("[DEBUG] ApplyFieldMapping - parseConfig为空，返回原始数据\n")
		}
		return row
	}

	// 获取fieldMapping配置
	fieldMappingInterface, exists := parseConfig["fieldMapping"]
	if !exists {
		if debug {
			fmt.Printf("[DEBUG] ApplyFieldMapping - 没有fieldMapping配置，返回原始数据\n")
		}
		return row
	}

	if debug {
		fmt.Printf("[DEBUG] ApplyFieldMapping - fieldMapping原始配置: %+v\n", fieldMappingInterface)
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
			fmt.Printf("[DEBUG] ApplyFieldMapping - 使用数组格式fieldMapping，条目数: %d\n", len(fieldMappingArray))
		}
	} else if mappingMap, ok := fieldMappingInterface.(map[string]interface{}); ok {
		// 兼容旧的对象格式
		fieldMappingMap = mappingMap
		isArrayFormat = false
		if debug {
			fmt.Printf("[DEBUG] ApplyFieldMapping - 使用对象格式fieldMapping（兼容模式）\n")
		}
	} else {
		if debug {
			fmt.Printf("[DEBUG] ApplyFieldMapping - fieldMapping格式不支持，返回原始数据\n")
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
						fmt.Printf("[DEBUG] ApplyFieldMapping - 映射规则: %s -> %s\n", source, target)
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
				fmt.Printf("[DEBUG] ApplyFieldMapping - 字段映射: %s -> %s, 值: %v\n", sourceField, targetField, value)
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
				fmt.Printf("[DEBUG] ApplyFieldMapping - 字段映射（兼容模式）: %s -> %s, 值: %v\n", sourceField, targetField, value)
			}
		}
	}

	if debug {
		fmt.Printf("[DEBUG] ApplyFieldMapping - 映射后数据: %+v\n", mappedRow)
	}
	return mappedRow
}

// ProcessValueForDatabase 处理数据库值，特别是时间字段格式转换
func (fm *FieldMapper) ProcessValueForDatabase(columnName string, value interface{}, debugLog ...bool) interface{} {
	if value == nil {
		return value
	}

	debug := len(debugLog) > 0 && debugLog[0]
	if debug {
		fmt.Printf("[DEBUG] FieldMapper.ProcessValueForDatabase - 处理字段: %s, 原始值: %+v, 类型: %T\n", columnName, value, value)
	}

	// 检查是否是时间相关字段
	isTimeField := strings.Contains(strings.ToLower(columnName), "time") ||
		strings.Contains(strings.ToLower(columnName), "date") ||
		strings.Contains(strings.ToLower(columnName), "created_at") ||
		strings.Contains(strings.ToLower(columnName), "updated_at")

	if isTimeField {
		// 处理时间类型
		switch v := value.(type) {
		case time.Time:
			// 转换为PostgreSQL兼容的字符串格式
			formatted := v.Format("2006-01-02 15:04:05.000")
			if debug {
				fmt.Printf("[DEBUG] ProcessValueForDatabase - 时间字段转换: %s -> %s\n", v.String(), formatted)
			}
			return formatted
		case string:
			// 尝试解析字符串时间并重新格式化
			if parsedTime, err := time.Parse(time.RFC3339, v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				if debug {
					fmt.Printf("[DEBUG] ProcessValueForDatabase - 字符串时间转换(RFC3339): %s -> %s\n", v, formatted)
				}
				return formatted
			}
			if parsedTime, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				if debug {
					fmt.Printf("[DEBUG] ProcessValueForDatabase - 字符串时间转换(标准): %s -> %s\n", v, formatted)
				}
				return formatted
			}
			if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				if debug {
					fmt.Printf("[DEBUG] ProcessValueForDatabase - 字符串时间转换(ISO): %s -> %s\n", v, formatted)
				}
				return formatted
			}
			if parsedTime, err := time.Parse("2006-01-02T15:04:05.000Z", v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				if debug {
					fmt.Printf("[DEBUG] ProcessValueForDatabase - 字符串时间转换(ISO毫秒): %s -> %s\n", v, formatted)
				}
				return formatted
			}
			// 如果无法解析，返回原值
			fmt.Printf("[DEBUG] ProcessValueForDatabase - 无法解析时间字符串，返回原值: %s\n", v)
			return v
		default:
			// 尝试转换为字符串再解析
			if timeStr := cast.ToString(v); timeStr != "" {
				if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
					formatted := parsedTime.Format("2006-01-02 15:04:05.000")
					fmt.Printf("[DEBUG] ProcessValueForDatabase - 其他类型时间转换: %v -> %s\n", v, formatted)
					return formatted
				}
			}
			fmt.Printf("[DEBUG] ProcessValueForDatabase - 无法处理的时间类型，返回原值: %v\n", v)
			return v
		}
	}

	// 非时间字段，直接返回原值
	fmt.Printf("[DEBUG] ProcessValueForDatabase - 非时间字段，返回原值: %v\n", value)
	return value
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

	fmt.Printf("[DEBUG] FieldMapper.ReplaceTableData - 开始替换表数据\n")
	fmt.Printf("[DEBUG] ReplaceTableData - 表名: %s\n", fullTableName)
	fmt.Printf("[DEBUG] ReplaceTableData - 数据行数: %d\n", len(data))

	// 开启事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR] ReplaceTableData - 事务回滚，原因: %v\n", r)
			tx.Rollback()
		}
	}()

	// 清空现有数据
	deleteSQL := fmt.Sprintf("DELETE FROM %s", fullTableName)
	fmt.Printf("[DEBUG] ReplaceTableData - 清空表SQL: %s\n", deleteSQL)

	if err := tx.Exec(deleteSQL).Error; err != nil {
		fmt.Printf("[ERROR] ReplaceTableData - 清空表数据失败: %v\n", err)
		tx.Rollback()
		return 0, fmt.Errorf("清空表数据失败: %w", err)
	}

	// 插入新数据
	var insertedRows int64
	for i, row := range data {
		// 只对第一行数据输出详细调试信息
		if i == 0 {
			fmt.Printf("[DEBUG] ReplaceTableData - 处理第 %d 行数据: %+v\n", i+1, row)
		} else if i%100 == 0 {
			fmt.Printf("[DEBUG] ReplaceTableData - 已处理 %d 行数据...\n", i+1)
		}

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig, i == 0)
		if i == 0 {
			fmt.Printf("[DEBUG] ReplaceTableData - 字段映射后的数据: %+v\n", mappedRow)
		}

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := fm.ProcessValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 构建插入SQL
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if i == 0 {
			fmt.Printf("[DEBUG] ReplaceTableData - 插入SQL: %s\n", insertSQL)
			fmt.Printf("[DEBUG] ReplaceTableData - 插入参数: %+v\n", values)
		}

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] ReplaceTableData - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] ReplaceTableData - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] ReplaceTableData - 失败的参数: %+v\n", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("[ERROR] ReplaceTableData - 提交事务失败: %v\n", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("[DEBUG] ReplaceTableData - 成功插入 %d 行数据\n", insertedRows)
	return insertedRows, nil
}

// UpsertBatchDataWithTx 使用提供的事务进行批量UPSERT操作（增量同步）
func (fm *FieldMapper) UpsertBatchDataWithTx(ctx context.Context, tx *gorm.DB, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	fmt.Printf("[DEBUG] FieldMapper.UpsertBatchDataWithTx - 开始UPSERT批量数据到表: %s\n", fullTableName)
	fmt.Printf("[DEBUG] UpsertBatchDataWithTx - 数据行数: %d\n", len(data))

	// 处理数据（使用提供的事务）
	var processedRows int64
	for i, row := range data {
		fmt.Printf("[DEBUG] UpsertBatchDataWithTx - 处理第 %d 行数据: %+v\n", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := fm.ApplyFieldMapping(row, parseConfig)
		fmt.Printf("[DEBUG] UpsertBatchDataWithTx - 字段映射后的数据: %+v\n", mappedRow)

		// 构建UPSERT SQL（这里简化为INSERT，实际应该实现UPSERT逻辑）
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := fm.ProcessValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 构建INSERT SQL（简化版，实际应该是UPSERT）
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		fmt.Printf("[DEBUG] UpsertBatchDataWithTx - 插入SQL: %s\n", insertSQL)
		fmt.Printf("[DEBUG] UpsertBatchDataWithTx - 插入参数: %+v\n", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] UpsertBatchDataWithTx - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] UpsertBatchDataWithTx - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] UpsertBatchDataWithTx - 失败的参数: %+v\n", values)
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		processedRows++
	}

	fmt.Printf("[DEBUG] UpsertBatchDataWithTx - 成功处理 %d 行数据\n", processedRows)
	return processedRows, nil
}
