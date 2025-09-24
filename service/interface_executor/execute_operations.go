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
	"datahub-service/service/datasource"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
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
	fmt.Printf("[DEBUG] ExecuteOperations.ExecutePreview - 开始预览接口: %s\n", interfaceInfo.GetID())

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
	fmt.Printf("[DEBUG] ExecuteOperations.ExecuteTest - 开始测试接口: %s\n", interfaceInfo.GetID())

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

// ExecuteSync 执行同步操作
func (ops *ExecuteOperations) ExecuteSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 同步操作：完整的数据同步流程
	fmt.Printf("[DEBUG] ExecuteOperations.ExecuteSync - 开始同步接口: %s\n", interfaceInfo.GetID())

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

	// 检查是否需要批量同步
	interfaceConfig := interfaceInfo.GetInterfaceConfig()
	limitConfig, hasLimitConfig := interfaceConfig[meta.DataInterfaceConfigFieldLimitConfig]

	fmt.Printf("[DEBUG] ExecuteSync - 接口配置: %+v\n", interfaceConfig)
	fmt.Printf("[DEBUG] ExecuteSync - 限制配置: hasLimitConfig=%t, config=%+v\n", hasLimitConfig, limitConfig)

	if hasLimitConfig {
		if limitMap, ok := limitConfig.(map[string]interface{}); ok {
			enabled := cast.ToBool(limitMap["enabled"])
			fmt.Printf("[DEBUG] ExecuteSync - 限制配置启用状态: %t\n", enabled)

			if enabled {
				// 使用批量同步
				return ops.ExecuteBatchSync(ctx, interfaceInfo, request, startTime, limitMap)
			}
		}
	}

	// 执行单次数据获取（传统方式）
	dataProcessor := NewDataProcessor(ops.executor)
	data, dataTypes, warnings, err := dataProcessor.FetchDataFromSourceWithExecuteType(ctx, interfaceInfo, request.Parameters, request.ExecuteType)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "同步数据获取失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 更新表数据
	fieldMapper := NewFieldMapper()
	updatedRows, err := fieldMapper.UpdateTableData(ctx, ops.executor.db, interfaceInfo, data)
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
		Message:      "数据同步成功",
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
			"interface_id":   interfaceInfo.GetID(),
			"interface_name": interfaceInfo.GetName(),
			"schema_name":    interfaceInfo.GetSchemaName(),
			"table_name":     interfaceInfo.GetTableName(),
		},
	}, nil
}

// ExecuteBatchSync 执行批量同步操作
func (ops *ExecuteOperations) ExecuteBatchSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time, limitConfig map[string]interface{}) (*ExecuteResponse, error) {
	fmt.Printf("[DEBUG] ExecuteBatchSync - 开始批量同步，配置: %+v\n", limitConfig)

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

	fmt.Printf("[DEBUG] ExecuteBatchSync - 批量大小: %d, 最大限制: %d\n", batchSize, int(maxLimit))

	// 获取数据源信息
	var dataSource models.DataSource
	if err := ops.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		fmt.Printf("[ERROR] ExecuteBatchSync - 获取数据源信息失败: %v\n", err)
		return &ExecuteResponse{
			Success:     false,
			Message:     "获取数据源信息失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 检查数据源类型，数据库和API接口都可能需要批量处理
	fmt.Printf("[DEBUG] ExecuteBatchSync - 数据源类型: %s, 分类: %s\n", dataSource.Type, dataSource.Category)

	// 检查是否支持批量处理
	supportsBatch := dataSource.Category == meta.DataSourceCategoryDatabase || dataSource.Category == meta.DataSourceCategoryAPI
	if !supportsBatch {
		fmt.Printf("[DEBUG] ExecuteBatchSync - 数据源类型不支持批量同步，使用单次同步\n")
		// 对于不支持批量的数据源，回退到单次同步
		return ops.ExecuteSync(ctx, interfaceInfo, request, startTime)
	}

	// 开始事务，确保批量同步的原子性
	tx := ops.executor.db.Begin()
	if tx.Error != nil {
		fmt.Printf("[ERROR] ExecuteBatchSync - 开始事务失败: %v\n", tx.Error)
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
			fmt.Printf("[ERROR] ExecuteBatchSync - 发生panic，事务已回滚: %v\n", r)
		}
	}()

	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())
	fmt.Printf("[DEBUG] ExecuteBatchSync - 开始事务批量同步，目标表: %s\n", fullTableName)

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
		fmt.Printf("[DEBUG] ExecuteBatchSync - API分页配置: pageParam=%s, sizeParam=%s, startPage=%d\n", pageParamName, sizeParamName, startPage)
	} else {
		// 数据库类型：使用标准分页参数
		pageParamName = "page"
		sizeParamName = "page_size"
		startPage = 1
		currentPage = startPage
		fmt.Printf("[DEBUG] ExecuteBatchSync - 数据库分页配置: pageParam=%s, sizeParam=%s, startPage=%d\n", pageParamName, sizeParamName, startPage)
	}

	for hasMoreData {
		fmt.Printf("[DEBUG] ExecuteBatchSync - 处理第 %d 批，批量大小: %d\n", currentPage, batchSize)

		// 构建分页参数
		pageParams := map[string]interface{}{
			pageParamName: currentPage,
			sizeParamName: batchSize,
		}

		// 获取批量数据
		batchData, dataTypes, warnings, err := dataProcessor.FetchBatchDataFromSource(ctx, interfaceInfo, request.Parameters, pageParams)
		if err != nil {
			fmt.Printf("[ERROR] ExecuteBatchSync - 获取第 %d 批数据失败: %v\n", currentPage, err)
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
			fmt.Printf("[DEBUG] ExecuteBatchSync - 第 %d 批没有数据，结束数据收集\n", currentPage)
			hasMoreData = false
			break
		}

		// 将批次数据添加到总数据中
		allBatchData = append(allBatchData, batchData...)
		fmt.Printf("[DEBUG] ExecuteBatchSync - 第 %d 批收集了 %d 行数据，总计 %d 行\n", currentPage, len(batchData), len(allBatchData))

		// 判断是否有更多数据的逻辑
		if len(batchData) < batchSize {
			fmt.Printf("[DEBUG] ExecuteBatchSync - 第 %d 批数据量(%d)小于批量大小(%d)，这是最后一批\n", currentPage, len(batchData), batchSize)
			hasMoreData = false
		}

		currentPage++

		// 防止无限循环
		if currentPage > 1000 {
			fmt.Printf("[WARN] ExecuteBatchSync - 达到最大批次限制(1000)，停止数据收集\n")
			allWarnings = append(allWarnings, "达到最大批次限制，可能还有更多数据未同步")
			break
		}
	}

	fmt.Printf("[DEBUG] ExecuteBatchSync - 数据收集完成，总共收集 %d 批，%d 行数据\n", currentPage-1, len(allBatchData))

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
	fmt.Printf("[DEBUG] ExecuteBatchSync - 开始在事务中执行数据库操作\n")

	// 1. 清空目标表（在事务中执行）
	fmt.Printf("[DEBUG] ExecuteBatchSync - 清空表: %s\n", fullTableName)
	if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", fullTableName)).Error; err != nil {
		fmt.Printf("[ERROR] ExecuteBatchSync - 清空表失败: %v\n", err)
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
	fmt.Printf("[DEBUG] ExecuteBatchSync - 开始批量插入 %d 行数据\n", len(allBatchData))
	insertedRows, err := fieldMapper.InsertBatchDataWithTx(ctx, tx, interfaceInfo, allBatchData)
	if err != nil {
		fmt.Printf("[ERROR] ExecuteBatchSync - 批量插入数据失败: %v\n", err)
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
		fmt.Printf("[ERROR] ExecuteBatchSync - 提交事务失败: %v\n", err)
		return &ExecuteResponse{
			Success:     false,
			Message:     "提交事务失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	totalRows = insertedRows
	fmt.Printf("[DEBUG] ExecuteBatchSync - 事务提交成功，总共插入 %d 行\n", totalRows)

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

// ExecuteIncrementalSync 执行增量同步
func (ops *ExecuteOperations) ExecuteIncrementalSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 获取数据源
	dataSourceID := interfaceInfo.GetDataSourceID()
	dataSource, err := ops.executor.datasourceManager.Get(dataSourceID)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "获取数据源失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 获取实际的数据源模型
	var dataSourceModel *models.DataSource
	err = ops.executor.db.First(&dataSourceModel, "id = ?", dataSourceID).Error
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "获取数据源模型失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 构建增量查询请求
	queryBuilder, err := datasource.NewQueryBuilder(dataSourceModel, &models.DataInterface{
		InterfaceConfig: interfaceInfo.GetInterfaceConfig(),
	})
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "创建查询构建器失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 构建增量参数
	incrementalParams := &datasource.IncrementalParams{
		LastSyncTime:   request.LastSyncTime,
		IncrementalKey: request.IncrementalKey,
		ComparisonType: "gt", // 默认使用大于比较
		BatchSize:      request.BatchSize,
	}

	if incrementalParams.BatchSize <= 0 {
		incrementalParams.BatchSize = 1000 // 默认批量大小
	}

	// 构建增量查询请求
	executeRequest, err := queryBuilder.BuildIncrementalRequest("sync", incrementalParams)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "构建增量查询请求失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 执行查询
	response, err := dataSource.Execute(ctx, executeRequest)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "执行增量查询失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 处理返回的数据
	dataProcessor := NewDataProcessor(ops.executor)
	data, dataTypes, warnings := dataProcessor.ProcessResponseData(response.Data)

	// 如果没有数据，返回成功但无更新
	if len(data) == 0 {
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
				"interface_name":  interfaceInfo.GetName(),
				"schema_name":     interfaceInfo.GetSchemaName(),
				"table_name":      interfaceInfo.GetTableName(),
				"sync_strategy":   "incremental",
				"last_sync_time":  request.LastSyncTime,
				"incremental_key": request.IncrementalKey,
			},
		}, nil
	}

	// 使用DataSyncEngine进行增量同步
	target := TableTarget{
		TableName:   interfaceInfo.GetTableName(),
		Schema:      interfaceInfo.GetSchemaName(),
		PrimaryKeys: []string{"id"}, // 默认主键，实际应该从接口配置中获取
		Columns:     make([]string, 0, len(dataTypes)),
	}

	// 从数据类型中提取列名
	for column := range dataTypes {
		target.Columns = append(target.Columns, column)
	}

	// 执行增量同步
	syncResult, err := ops.executor.dataSyncEngine.ExecuteSync(ctx, DataIncrementalSync, data, target)
	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "增量同步执行失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
			Warnings:    warnings,
		}, err
	}

	// TODO: 更新接口的最后同步时间
	// 这里需要根据接口类型更新相应的表

	return &ExecuteResponse{
		Success:      true,
		Message:      "增量同步完成",
		Duration:     time.Since(startTime).Milliseconds(),
		ExecuteType:  request.ExecuteType,
		Data:         data,
		RowCount:     len(data),
		ColumnCount:  len(dataTypes),
		DataTypes:    dataTypes,
		TableUpdated: syncResult.InsertedCount > 0 || syncResult.UpdatedCount > 0,
		UpdatedRows:  syncResult.InsertedCount + syncResult.UpdatedCount,
		Warnings:     warnings,
		Metadata: map[string]interface{}{
			"interface_name":   interfaceInfo.GetName(),
			"schema_name":      interfaceInfo.GetSchemaName(),
			"table_name":       interfaceInfo.GetTableName(),
			"sync_strategy":    "incremental",
			"last_sync_time":   request.LastSyncTime,
			"incremental_key":  request.IncrementalKey,
			"inserted_count":   syncResult.InsertedCount,
			"updated_count":    syncResult.UpdatedCount,
			"error_count":      syncResult.ErrorCount,
			"sync_duration_ms": syncResult.Duration.Milliseconds(),
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
