/*
 * @module service/interface_executor/execute_operations
 * @description 具体的接口执行操作方法，包括预览、测试、同步等操作
 * @architecture 策略模式 - 根据执行类型选择不同的执行策略
 * @documentReference ai_docs/interface_executor.md
 * @stateFlow 执行类型判断 -> 策略选择 -> 具体执行 -> 结果返回
 * @rules 每种执行类型都有明确的执行逻辑和返回格式
 * @dependencies datahub-service/service/datasource, datahub-service/service/meta
 * @refs executor.go, data_processing.go
 */

package interface_executor

import (
	"context"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cast"
)

// ExecuteOperations 执行操作处理器
type ExecuteOperations struct {
	executor *InterfaceExecutor
}

// NewExecuteOperations 创建执行操作处理器
func NewExecuteOperations(executor *InterfaceExecutor) *ExecuteOperations {
	return &ExecuteOperations{executor: executor}
}

// ExecutePreview 执行预览操作
func (ops *ExecuteOperations) ExecutePreview(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 预览操作：调用一次接口，获取数据并返回
	slog.Debug("ExecuteOperations.ExecutePreview - 开始预览接口", "value", interfaceInfo.GetID())

	// 执行数据获取
	dataProcessor := NewDataProcessor(ops.executor)
	data, dataTypes, warnings, err := dataProcessor.FetchDataFromSourceWithExecuteType(ctx, interfaceInfo, request.Parameters, request.ExecuteType)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "预览数据获取失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 处理数据限制
	limit := request.Limit
	if limit == 0 {
		limit = cast.ToInt(request.Parameters["limit"])
	}
	if limit <= 0 || limit > 1000 {
		limit = 10
	}

	// 限制返回的数据量
	limitedData := ops.limitDataRows(data, limit)

	return &ExecuteResponse{
		Success:     true,
		Message:     "数据预览成功",
		Duration:    time.Since(startTime).Milliseconds(),
		ExecuteType: request.ExecuteType,
		Data:        limitedData,
		RowCount:    len(limitedData),
		ColumnCount: len(dataTypes),
		DataTypes:   dataTypes,
		Warnings:    warnings,
		Metadata: map[string]interface{}{
			"interface_id":    interfaceInfo.GetID(),
			"interface_name":  interfaceInfo.GetName(),
			"schema_name":     interfaceInfo.GetSchemaName(),
			"table_name":      interfaceInfo.GetTableName(),
			"requested_limit": limit,
		},
	}, nil
}

// ExecuteTest 执行测试操作
func (ops *ExecuteOperations) ExecuteTest(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 测试操作：实际执行一次接口同步，更新表数据
	slog.Debug("ExecuteOperations.ExecuteTest - 开始测试接口", "value", interfaceInfo.GetID())

	// 执行数据获取
	dataProcessor := NewDataProcessor(ops.executor)
	data, dataTypes, warnings, err := dataProcessor.FetchDataFromSourceWithExecuteType(ctx, interfaceInfo, request.Parameters, request.ExecuteType)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "测试数据获取失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 更新表数据
	var updatedRows int64
	var tableUpdated bool
	if interfaceInfo.IsTableCreated() {
		fieldMapper := NewFieldMapper()
		updatedRows, err = fieldMapper.UpdateTableData(ctx, ops.executor.db, interfaceInfo, data)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("更新表数据失败: %v", err))
		} else {
			tableUpdated = true
		}
	} else {
		warnings = append(warnings, "接口表尚未创建，跳过数据更新")
	}

	return &ExecuteResponse{
		Success:      true,
		Message:      "接口测试成功",
		Duration:     time.Since(startTime).Milliseconds(),
		ExecuteType:  request.ExecuteType,
		Data:         data,
		RowCount:     len(data),
		ColumnCount:  len(dataTypes),
		DataTypes:    dataTypes,
		TableUpdated: tableUpdated,
		UpdatedRows:  updatedRows,
		Warnings:     warnings,
		Metadata: map[string]interface{}{
			"interface_id":   interfaceInfo.GetID(),
			"interface_name": interfaceInfo.GetName(),
			"schema_name":    interfaceInfo.GetSchemaName(),
			"table_name":     interfaceInfo.GetTableName(),
		},
	}, nil
}

// ExecuteSync 执行同步操作（统一处理全量和增量同步）
func (ops *ExecuteOperations) ExecuteSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	slog.Debug("ExecuteOperations.ExecuteSync - 开始同步接口", "value", interfaceInfo.GetID())

	// 检查表是否创建
	if !interfaceInfo.IsTableCreated() {
		return &ExecuteResponse{
			Success:     false,
			Message:     "接口表尚未创建，无法执行同步",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       "table not created",
		}, fmt.Errorf("接口表尚未创建")
	}

	interfaceConfig := interfaceInfo.GetInterfaceConfig()
	slog.Debug("ExecuteSync - 接口配置", "data", interfaceConfig)

	// 1. 检查是否启用增量同步
	syncStrategy := "full" // 默认全量同步
	var lastSyncValue interface{}
	var incrementalKey string

	slog.Debug("ExecuteSync - 开始检查增量配置")
	if incrementalConfig, exists := interfaceConfig["incremental_config"]; exists {
		slog.Debug("ExecuteSync - 找到增量配置", "incremental_config", incrementalConfig)

		if configMap, ok := incrementalConfig.(map[string]interface{}); ok {
			enabled := cast.ToBool(configMap["enabled"])
			slog.Debug("ExecuteSync - 增量配置启用状态", "enabled", enabled)

			if enabled {
				syncStrategy = "incremental"
				slog.Debug("ExecuteSync - 设置同步策略为增量", "sync_strategy", syncStrategy)

				// 获取增量字段名（源字段）- 兼容 increment_field 和 incremental_field 两种字段名
				sourceFieldName := cast.ToString(configMap["incremental_field"])
				if sourceFieldName == "" {
					sourceFieldName = cast.ToString(configMap["increment_field"])
				}
				slog.Debug("ExecuteSync - 增量字段名", "incremental_field", sourceFieldName)

				if sourceFieldName == "" {
					slog.Error("ExecuteSync - 增量配置缺少字段名")
					return &ExecuteResponse{
						Success:     false,
						Message:     "增量配置缺少字段名",
						Duration:    time.Since(startTime).Milliseconds(),
						ExecuteType: request.ExecuteType,
						Error:       "incremental_field or increment_field is required in incremental_config",
					}, fmt.Errorf("增量配置缺少字段名")
				}

				// 获取本系统表中对应字段的最新值
				slog.Debug("ExecuteSync - 开始获取最后同步值", "source_field", sourceFieldName)
				mappedFieldName, lastValue, err := ops.getLastSyncValue(interfaceInfo, sourceFieldName, configMap)
				if err != nil {
					slog.Warn("ExecuteSync - 获取最后同步值失败，将使用全量同步", "error", err)
					syncStrategy = "full"
				} else {
					lastSyncValue = lastValue
					incrementalKey = sourceFieldName
					slog.Debug("ExecuteSync - 增量同步参数",
						"source_field", sourceFieldName,
						"mapped_field", mappedFieldName,
						"last_sync_value", lastValue,
						"incremental_key", incrementalKey)
				}
			} else {
				slog.Debug("ExecuteSync - 增量配置未启用，使用全量同步")
			}
		} else {
			slog.Error("ExecuteSync - 增量配置类型转换失败", "type", fmt.Sprintf("%T", incrementalConfig))
		}
	} else {
		slog.Debug("ExecuteSync - 接口配置中没有增量配置，使用全量同步")
	}

	// 2. 检查是否需要批量同步
	limitConfig, hasLimitConfig := interfaceConfig[meta.DataInterfaceConfigFieldLimitConfig]
	if hasLimitConfig {
		if limitMap, ok := limitConfig.(map[string]interface{}); ok {
			enabled := cast.ToBool(limitMap["enabled"])
			slog.Debug("ExecuteSync - 批量配置启用状态", "enabled", enabled)

			if enabled {
				// 使用批量同步（支持增量）
				return ops.ExecuteBatchSyncWithStrategy(ctx, interfaceInfo, request, startTime, limitMap, syncStrategy, lastSyncValue, incrementalKey)
			}
		}
	}

	// 3. 执行单次数据同步
	return ops.ExecuteSingleSync(ctx, interfaceInfo, request, startTime, syncStrategy, lastSyncValue, incrementalKey)
}

// ExecuteBatchSync 执行批量同步操作
func (ops *ExecuteOperations) ExecuteBatchSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time, limitConfig map[string]interface{}) (*ExecuteResponse, error) {
	slog.Debug("ExecuteBatchSync - 开始批量同步，配置", "data", limitConfig)

	// 获取批量配置参数
	defaultLimit := cast.ToInt(limitConfig["default_limit"])
	maxLimit := cast.ToInt(limitConfig["max_limit"])

	if defaultLimit <= 0 {
		defaultLimit = 1000 // 默认每批1000条
	}
	if maxLimit <= 0 {
		maxLimit = 10000 // 默认最大10000条
	}

	// 确保批量大小不超过最大限制
	batchSize := defaultLimit
	if batchSize > maxLimit {
		batchSize = maxLimit
	}

	slog.Debug("ExecuteBatchSync - 批量大小", "batch_size", batchSize, "max_limit", int(maxLimit))

	// 获取数据源信息
	var dataSource models.DataSource
	if err := ops.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		slog.Error("ExecuteBatchSync - 获取数据源信息失败", "error", err)
		return &ExecuteResponse{
			Success:     false,
			Message:     "获取数据源信息失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 检查数据源类型，数据库和API接口都可能需要批量处理
	slog.Debug("ExecuteBatchSync - 数据源类型: %s, 分类: %s\n", dataSource.Type, dataSource.Category)

	// 检查是否支持批量处理
	supportsBatch := dataSource.Category == meta.DataSourceCategoryDatabase || dataSource.Category == meta.DataSourceCategoryAPI
	if !supportsBatch {
		slog.Debug("ExecuteBatchSync - 数据源类型不支持批量同步，使用单次同步")
		// 对于不支持批量的数据源，回退到单次同步
		return ops.ExecuteSync(ctx, interfaceInfo, request, startTime)
	}

	// 开始事务，确保批量同步的原子性
	tx := ops.executor.db.Begin()
	if tx.Error != nil {
		slog.Error("ExecuteBatchSync - 开始事务失败", "error", tx.Error)
		return &ExecuteResponse{
			Success:     false,
			Message:     "开始事务失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       tx.Error.Error(),
		}, tx.Error
	}

	// 确保在函数结束时处理事务
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			slog.Error("ExecuteBatchSync - 发生panic，事务已回滚", "error", r)
		}
	}()

	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())
	slog.Debug("ExecuteBatchSync - 开始事务批量同步，目标表", "value", fullTableName)

	// 批量数据同步
	dataProcessor := NewDataProcessor(ops.executor)
	fieldMapper := NewFieldMapper()

	var totalRows int64 = 0
	var allDataTypes map[string]string
	var allWarnings []string
	var allBatchData []map[string]interface{} // 收集所有批次的数据
	currentPage := 1
	hasMoreData := true

	// 根据数据源类型确定分页参数
	var pageParamName, sizeParamName string
	var startPage int

	if dataSource.Category == meta.DataSourceCategoryAPI {
		// API类型：从接口配置获取分页参数名
		interfaceConfig := interfaceInfo.GetInterfaceConfig()
		pageParamName = cast.ToString(interfaceConfig[meta.DataInterfaceConfigFieldPaginationPageParam])
		sizeParamName = cast.ToString(interfaceConfig[meta.DataInterfaceConfigFieldPaginationSizeParam])
		startPage = cast.ToInt(interfaceConfig[meta.DataInterfaceConfigFieldPaginationStartValue])

		if pageParamName == "" {
			pageParamName = "page"
		}
		if sizeParamName == "" {
			sizeParamName = "size"
		}
		if startPage <= 0 {
			startPage = 1
		}

		currentPage = startPage
		slog.Debug("ExecuteBatchSync - API分页配置", "page_param", pageParamName, "size_param", sizeParamName, "start_page", startPage)
	} else {
		// 数据库类型：使用标准分页参数
		pageParamName = "page"
		sizeParamName = "page_size"
		startPage = 1
		currentPage = startPage
		slog.Debug("ExecuteBatchSync - 数据库分页配置", "page_param", pageParamName, "size_param", sizeParamName, "start_page", startPage)
	}

	for hasMoreData {
		slog.Debug("ExecuteBatchSync - 处理批次", "page", currentPage, "batch_size", batchSize)

		// 构建分页参数
		pageParams := map[string]interface{}{
			pageParamName: currentPage,
			sizeParamName: batchSize,
		}

		// 获取批量数据
		batchData, dataTypes, warnings, err := dataProcessor.FetchBatchDataFromSource(ctx, interfaceInfo, request.Parameters, pageParams)
		if err != nil {
			slog.Error("ExecuteBatchSync - 获取批数据失败", "page", currentPage, "error", err)
			// 回滚事务
			tx.Rollback()
			return &ExecuteResponse{
				Success:     false,
				Message:     fmt.Sprintf("获取第 %d 批数据失败", currentPage),
				Duration:    time.Since(startTime).Milliseconds(),
				ExecuteType: request.ExecuteType,
				Error:       err.Error(),
			}, err
		}

		// 记录数据类型（使用第一批的数据类型）
		if allDataTypes == nil {
			allDataTypes = dataTypes
		}

		// 合并警告
		allWarnings = append(allWarnings, warnings...)

		// 判断是否还有更多数据
		if len(batchData) == 0 {
			slog.Debug("ExecuteBatchSync - 批次没有数据，结束数据收集", "batch", currentPage)
			hasMoreData = false
			break
		}

		// 将批次数据添加到总数据中
		allBatchData = append(allBatchData, batchData...)
		slog.Debug("ExecuteBatchSync - 批次收集数据", "batch", currentPage, "batch_count", len(batchData), "total", len(allBatchData))

		// 判断是否有更多数据的逻辑
		if len(batchData) < batchSize {
			slog.Debug("ExecuteBatchSync - 批次数据不足，停止收集", "batch", currentPage, "batch_size", batchSize)
			hasMoreData = false
		}

		currentPage++

		// 防止无限循环
		if currentPage > 1000 {
			slog.Warn("ExecuteBatchSync - 达到最大批次限制(1000)，停止数据收集")
			allWarnings = append(allWarnings, "达到最大批次限制，可能还有更多数据未同步")
			break
		}
	}

	slog.Debug("ExecuteBatchSync - 数据收集完成", "total_batches", currentPage-1, "total_rows", len(allBatchData))

	// 如果没有数据，提交事务并返回
	if len(allBatchData) == 0 {
		tx.Commit()
		return &ExecuteResponse{
			Success:      true,
			Message:      "批量同步完成，但没有新数据",
			Duration:     time.Since(startTime).Milliseconds(),
			ExecuteType:  request.ExecuteType,
			RowCount:     0,
			ColumnCount:  len(allDataTypes),
			DataTypes:    allDataTypes,
			TableUpdated: false,
			UpdatedRows:  0,
			Warnings:     allWarnings,
			Metadata: map[string]interface{}{
				"interface_id":   interfaceInfo.GetID(),
				"interface_name": interfaceInfo.GetName(),
				"schema_name":    interfaceInfo.GetSchemaName(),
				"table_name":     interfaceInfo.GetTableName(),
				"batch_count":    currentPage - 1,
				"batch_size":     batchSize,
			},
		}, nil
	}

	// 在事务中执行数据库操作
	slog.Debug("ExecuteBatchSync - 开始在事务中执行数据库操作")

	// 1. 清空目标表（在事务中执行）
	slog.Debug("ExecuteBatchSync - 清空表", "value", fullTableName)
	if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", fullTableName)).Error; err != nil {
		slog.Error("ExecuteBatchSync - 清空表失败", "error", err)
		tx.Rollback()
		return &ExecuteResponse{
			Success:     false,
			Message:     "清空表数据失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 2. 批量插入所有数据（在事务中执行）
	slog.Debug("ExecuteBatchSync - 开始批量插入数据", "row_count", len(allBatchData))
	insertedRows, err := fieldMapper.InsertBatchDataWithTx(ctx, tx, interfaceInfo, allBatchData)
	if err != nil {
		slog.Error("ExecuteBatchSync - 批量插入数据失败", "error", err)
		tx.Rollback()
		return &ExecuteResponse{
			Success:     false,
			Message:     "批量插入数据失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 3. 提交事务
	if err := tx.Commit().Error; err != nil {
		slog.Error("ExecuteBatchSync - 提交事务失败", "error", err)
		return &ExecuteResponse{
			Success:     false,
			Message:     "提交事务失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	totalRows = insertedRows
	slog.Debug("ExecuteBatchSync - 事务提交成功，总共插入", "count", totalRows) // 行

	return &ExecuteResponse{
		Success:      true,
		Message:      fmt.Sprintf("批量数据同步成功，处理 %d 批", currentPage-1),
		Duration:     time.Since(startTime).Milliseconds(),
		ExecuteType:  request.ExecuteType,
		RowCount:     int(totalRows),
		ColumnCount:  len(allDataTypes),
		DataTypes:    allDataTypes,
		TableUpdated: true,
		UpdatedRows:  totalRows,
		Warnings:     allWarnings,
		Metadata: map[string]interface{}{
			"interface_id":   interfaceInfo.GetID(),
			"interface_name": interfaceInfo.GetName(),
			"schema_name":    interfaceInfo.GetSchemaName(),
			"table_name":     interfaceInfo.GetTableName(),
			"batch_count":    currentPage - 1,
			"batch_size":     batchSize,
			"total_rows":     totalRows,
			"transaction":    "committed",
		},
	}, nil
}

// getLastSyncValue 获取本系统表中增量字段的最新值
func (ops *ExecuteOperations) getLastSyncValue(interfaceInfo InterfaceInfo, sourceFieldName string, incrementalConfig map[string]interface{}) (string, interface{}, error) {
	// 1. 获取字段映射关系
	mappedFieldName := sourceFieldName // 默认使用源字段名

	// 从解析配置中获取字段映射
	parseConfig := interfaceInfo.GetParseConfig()
	if parseConfig != nil {
		if fieldMapping, exists := parseConfig["field_mapping"]; exists {
			if mappingMap, ok := fieldMapping.(map[string]interface{}); ok {
				if mappedName, exists := mappingMap[sourceFieldName]; exists {
					mappedFieldName = cast.ToString(mappedName)
				}
			}
		}
	}

	// 2. 构建查询SQL
	schemaName := interfaceInfo.GetSchemaName()
	tableName := interfaceInfo.GetTableName()
	fullTableName := fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)

	// 检查表是否存在数据
	var count int64
	if err := ops.executor.db.Table(fullTableName).Count(&count).Error; err != nil {
		return mappedFieldName, nil, fmt.Errorf("查询表数据失败: %w", err)
	}

	if count == 0 {
		slog.Debug("getLastSyncValue - 表为空，返回nil作为初始值", "table", fullTableName)
		return mappedFieldName, nil, nil
	}

	// 3. 查询最新值
	var lastValue interface{}
	sql := fmt.Sprintf(`SELECT MAX("%s") FROM %s`, mappedFieldName, fullTableName)

	row := ops.executor.db.Raw(sql).Row()
	if err := row.Scan(&lastValue); err != nil {
		return mappedFieldName, nil, fmt.Errorf("查询最新值失败: %w", err)
	}

	slog.Debug("getLastSyncValue - 查询结果", "sql", sql, "result", lastValue)
	return mappedFieldName, lastValue, nil
}

// ExecuteSingleSync 执行单次数据同步
func (ops *ExecuteOperations) ExecuteSingleSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time, syncStrategy string, lastSyncValue interface{}, incrementalKey string) (*ExecuteResponse, error) {
	slog.Debug("ExecuteSingleSync - 开始单次同步，策略", "sync_strategy", syncStrategy, "last_sync_value", lastSyncValue, "incremental_key", incrementalKey)

	// 准备增量参数
	syncParams := make(map[string]interface{})
	for k, v := range request.Parameters {
		syncParams[k] = v
	}

	if syncStrategy == "incremental" && lastSyncValue != nil {
		// 添加增量查询参数
		syncParams["incremental_field"] = incrementalKey
		syncParams["last_sync_value"] = lastSyncValue
		syncParams["comparison_type"] = "gt" // 大于比较
	}

	// 执行数据获取
	dataProcessor := NewDataProcessor(ops.executor)
	data, dataTypes, warnings, err := dataProcessor.FetchDataFromSourceWithSyncStrategy(ctx, interfaceInfo, syncParams, syncStrategy)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "同步数据获取失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 如果是增量同步且没有新数据，直接返回
	if syncStrategy == "incremental" && len(data) == 0 {
		return &ExecuteResponse{
			Success:      true,
			Message:      "增量同步完成，无新数据",
			Duration:     time.Since(startTime).Milliseconds(),
			ExecuteType:  request.ExecuteType,
			Data:         data,
			RowCount:     0,
			ColumnCount:  len(dataTypes),
			DataTypes:    dataTypes,
			TableUpdated: false,
			UpdatedRows:  0,
			Warnings:     warnings,
			Metadata: map[string]interface{}{
				"interface_id":    interfaceInfo.GetID(),
				"interface_name":  interfaceInfo.GetName(),
				"schema_name":     interfaceInfo.GetSchemaName(),
				"table_name":      interfaceInfo.GetTableName(),
				"sync_strategy":   syncStrategy,
				"last_sync_value": lastSyncValue,
				"incremental_key": incrementalKey,
			},
		}, nil
	}

	// 更新表数据
	fieldMapper := NewFieldMapper()
	var updatedRows int64

	if syncStrategy == "full" {
		// 全量同步：先清空表，再插入新数据
		updatedRows, err = fieldMapper.ReplaceTableData(ctx, ops.executor.db, interfaceInfo, data)
	} else {
		// 增量同步：使用真正的UPSERT操作（插入或更新，不删除现有数据）
		updatedRows, err = fieldMapper.UpsertTableData(ctx, ops.executor.db, interfaceInfo, data)
	}

	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "同步数据更新失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	return &ExecuteResponse{
		Success:      true,
		Message:      fmt.Sprintf("%s同步成功", map[string]string{"full": "全量", "incremental": "增量"}[syncStrategy]),
		Duration:     time.Since(startTime).Milliseconds(),
		ExecuteType:  request.ExecuteType,
		Data:         data,
		RowCount:     len(data),
		ColumnCount:  len(dataTypes),
		DataTypes:    dataTypes,
		TableUpdated: true,
		UpdatedRows:  updatedRows,
		Warnings:     warnings,
		Metadata: map[string]interface{}{
			"interface_id":    interfaceInfo.GetID(),
			"interface_name":  interfaceInfo.GetName(),
			"schema_name":     interfaceInfo.GetSchemaName(),
			"table_name":      interfaceInfo.GetTableName(),
			"sync_strategy":   syncStrategy,
			"last_sync_value": lastSyncValue,
			"incremental_key": incrementalKey,
		},
	}, nil
}

// ExecuteBatchSyncWithStrategy 执行批量同步（支持增量策略）
func (ops *ExecuteOperations) ExecuteBatchSyncWithStrategy(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time, limitConfig map[string]interface{}, syncStrategy string, lastSyncValue interface{}, incrementalKey string) (*ExecuteResponse, error) {
	slog.Debug("ExecuteBatchSyncWithStrategy - 开始批量同步",
		"sync_strategy", syncStrategy,
		"last_sync_value", lastSyncValue,
		"incremental_key", incrementalKey,
		"limit_config", limitConfig)

	// 获取批量配置参数
	defaultLimit := cast.ToInt(limitConfig["default_limit"])
	maxLimit := cast.ToInt(limitConfig["max_limit"])

	if defaultLimit <= 0 {
		defaultLimit = 1000
	}
	if maxLimit <= 0 {
		maxLimit = 10000
	}

	batchSize := defaultLimit
	if batchSize > maxLimit {
		batchSize = maxLimit
	}

	slog.Debug("ExecuteBatchSyncWithStrategy - 批量大小", "batch_size", batchSize, "default_limit", defaultLimit, "max_limit", maxLimit)

	// 准备同步参数
	syncParams := make(map[string]interface{})
	for k, v := range request.Parameters {
		syncParams[k] = v
	}

	slog.Debug("ExecuteBatchSyncWithStrategy - 原始请求参数", "parameters", request.Parameters)

	if syncStrategy == "incremental" && lastSyncValue != nil {
		syncParams["incremental_field"] = incrementalKey
		syncParams["last_sync_value"] = lastSyncValue
		syncParams["comparison_type"] = "gt"
		slog.Debug("ExecuteBatchSyncWithStrategy - 添加增量参数",
			"incremental_field", incrementalKey,
			"last_sync_value", lastSyncValue,
			"comparison_type", "gt")
	}

	slog.Debug("ExecuteBatchSyncWithStrategy - 最终同步参数", "sync_params", syncParams)

	// 开始事务
	tx := ops.executor.db.Begin()
	if tx.Error != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "开始事务失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       tx.Error.Error(),
		}, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			slog.Error("ExecuteBatchSyncWithStrategy - 发生panic，事务已回滚", "error", r)
		}
	}()

	// 如果是全量同步，先清空表
	if syncStrategy == "full" {
		fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())
		if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", fullTableName)).Error; err != nil {
			tx.Rollback()
			return &ExecuteResponse{
				Success:     false,
				Message:     "清空表数据失败",
				Duration:    time.Since(startTime).Milliseconds(),
				ExecuteType: request.ExecuteType,
				Error:       err.Error(),
			}, err
		}
	}

	// 批量获取并处理数据
	dataProcessor := NewDataProcessor(ops.executor)
	fieldMapper := NewFieldMapper()

	var totalRows int64
	var allDataTypes map[string]string
	var allWarnings []string
	currentPage := 1
	hasMoreData := true

	for hasMoreData {
		pageParams := map[string]interface{}{
			"page":      currentPage,
			"page_size": batchSize,
		}

		batchData, dataTypes, warnings, err := dataProcessor.FetchBatchDataFromSourceWithStrategy(ctx, interfaceInfo, syncParams, pageParams, syncStrategy)
		if err != nil {
			tx.Rollback()
			return &ExecuteResponse{
				Success:     false,
				Message:     fmt.Sprintf("获取第 %d 批数据失败", currentPage),
				Duration:    time.Since(startTime).Milliseconds(),
				ExecuteType: request.ExecuteType,
				Error:       err.Error(),
			}, err
		}

		if allDataTypes == nil {
			allDataTypes = dataTypes
		}
		allWarnings = append(allWarnings, warnings...)

		if len(batchData) == 0 {
			hasMoreData = false
			break
		}

		// 批量处理数据
		var batchRows int64
		if syncStrategy == "full" {
			batchRows, err = fieldMapper.InsertBatchDataWithTx(ctx, tx, interfaceInfo, batchData)
		} else {
			batchRows, err = fieldMapper.UpsertBatchDataWithTx(ctx, tx, interfaceInfo, batchData)
		}

		if err != nil {
			tx.Rollback()
			return &ExecuteResponse{
				Success:     false,
				Message:     fmt.Sprintf("处理第 %d 批数据失败", currentPage),
				Duration:    time.Since(startTime).Milliseconds(),
				ExecuteType: request.ExecuteType,
				Error:       err.Error(),
			}, err
		}

		totalRows += batchRows

		if len(batchData) < batchSize {
			hasMoreData = false
		}

		currentPage++

		if currentPage > 1000 {
			allWarnings = append(allWarnings, "达到最大批次限制，可能还有更多数据未同步")
			break
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "提交事务失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	return &ExecuteResponse{
		Success:      true,
		Message:      fmt.Sprintf("批量%s同步成功，处理 %d 批", map[string]string{"full": "全量", "incremental": "增量"}[syncStrategy], currentPage-1),
		Duration:     time.Since(startTime).Milliseconds(),
		ExecuteType:  request.ExecuteType,
		RowCount:     int(totalRows),
		ColumnCount:  len(allDataTypes),
		DataTypes:    allDataTypes,
		TableUpdated: true,
		UpdatedRows:  totalRows,
		Warnings:     allWarnings,
		Metadata: map[string]interface{}{
			"interface_id":    interfaceInfo.GetID(),
			"interface_name":  interfaceInfo.GetName(),
			"schema_name":     interfaceInfo.GetSchemaName(),
			"table_name":      interfaceInfo.GetTableName(),
			"sync_strategy":   syncStrategy,
			"last_sync_value": lastSyncValue,
			"incremental_key": incrementalKey,
			"batch_count":     currentPage - 1,
			"batch_size":      batchSize,
			"total_rows":      totalRows,
		},
	}, nil
}

// limitDataRows 限制数据行数
func (ops *ExecuteOperations) limitDataRows(data []map[string]interface{}, limit int) []map[string]interface{} {
	if len(data) <= limit {
		return data
	}
	return data[:limit]
}
