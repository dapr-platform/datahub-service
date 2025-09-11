/*
 * @module service/interface_executor/executor
 * @description 通用接口执行器，支持基础库和主题库的数据接口执行
 * @architecture 分层架构 - 通用服务层
 * @documentReference ai_docs/interface_executor.md
 * @stateFlow 接口执行流程：获取接口信息 -> 获取数据源 -> 执行查询 -> 解析数据 -> 更新表 -> 返回结果
 * @rules 统一的接口执行逻辑，支持多种接口类型
 * @dependencies datahub-service/service/datasource, datahub-service/service/models
 * @refs service/basic_library, service/thematic_library
 */

package interface_executor

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// InterfaceExecutor 通用接口执行器
type InterfaceExecutor struct {
	db                *gorm.DB
	datasourceManager datasource.DataSourceManager
	dataSyncEngine    *DataSyncEngine
	errorHandler      *ErrorHandler
}

// NewInterfaceExecutor 创建接口执行器实例
func NewInterfaceExecutor(db *gorm.DB, datasourceManager datasource.DataSourceManager) *InterfaceExecutor {
	return &InterfaceExecutor{
		db:                db,
		datasourceManager: datasourceManager,
		dataSyncEngine:    NewDataSyncEngine(db),
		errorHandler:      NewErrorHandler(),
	}
}

// ExecuteRequest 接口执行请求
type ExecuteRequest struct {
	InterfaceID    string                 `json:"interface_id"`
	InterfaceType  string                 `json:"interface_type"`          // basic_library, thematic_library
	ExecuteType    string                 `json:"execute_type"`            // preview, test, sync, incremental_sync
	SyncStrategy   string                 `json:"sync_strategy,omitempty"` // full, incremental, realtime
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	Options        map[string]interface{} `json:"options,omitempty"`
	Limit          int                    `json:"limit,omitempty"`           // 用于预览时限制数据量
	LastSyncTime   interface{}            `json:"last_sync_time,omitempty"`  // 增量同步的最后同步时间
	IncrementalKey string                 `json:"incremental_key,omitempty"` // 增量同步的关键字段
	BatchSize      int                    `json:"batch_size,omitempty"`      // 批处理大小
}

// ExecuteResponse 接口执行响应
type ExecuteResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Duration     int64                  `json:"duration"` // 执行耗时（毫秒）
	ExecuteType  string                 `json:"execute_type"`
	Data         interface{}            `json:"data,omitempty"`
	RowCount     int                    `json:"row_count,omitempty"`
	ColumnCount  int                    `json:"column_count,omitempty"`
	DataTypes    map[string]string      `json:"data_types,omitempty"`
	TableUpdated bool                   `json:"table_updated,omitempty"` // 是否更新了表数据
	UpdatedRows  int64                  `json:"updated_rows,omitempty"`  // 更新的行数
	Error        string                 `json:"error,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Execute 执行接口操作
func (e *InterfaceExecutor) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()

	// 根据接口类型获取接口信息
	var interfaceInfo InterfaceInfo
	var err error

	switch request.InterfaceType {
	case "basic_library":
		interfaceInfo, err = e.getBasicLibraryInterface(request.InterfaceID)
	case "thematic_library":
		interfaceInfo, err = e.getThematicLibraryInterface(request.InterfaceID)
	default:
		return &ExecuteResponse{
			Success:     false,
			Message:     "不支持的接口类型",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       fmt.Sprintf("unsupported interface type: %s", request.InterfaceType),
		}, fmt.Errorf("不支持的接口类型: %s", request.InterfaceType)
	}

	if err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "获取接口信息失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 根据执行类型进行不同的处理
	switch request.ExecuteType {
	case "preview":
		return e.executePreview(ctx, interfaceInfo, request, startTime)
	case "test":
		return e.executeTest(ctx, interfaceInfo, request, startTime)
	case "sync":
		return e.executeSync(ctx, interfaceInfo, request, startTime)
	case "incremental_sync":
		return e.executeIncrementalSync(ctx, interfaceInfo, request, startTime)
	default:
		return &ExecuteResponse{
			Success:     false,
			Message:     "不支持的执行类型",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       fmt.Sprintf("unsupported execute type: %s", request.ExecuteType),
		}, fmt.Errorf("不支持的执行类型: %s", request.ExecuteType)
	}
}

// InterfaceInfo 接口信息接口
type InterfaceInfo interface {
	GetID() string
	GetName() string
	GetType() string
	GetDataSourceID() string
	GetSchemaName() string
	GetTableName() string
	GetInterfaceConfig() map[string]interface{}
	GetParseConfig() map[string]interface{}
	GetTableFieldsConfig() []interface{}
	IsTableCreated() bool
}

// BasicLibraryInterfaceInfo 基础库接口信息
type BasicLibraryInterfaceInfo struct {
	*models.DataInterface
}

func (b *BasicLibraryInterfaceInfo) GetID() string           { return b.ID }
func (b *BasicLibraryInterfaceInfo) GetName() string         { return b.NameZh }
func (b *BasicLibraryInterfaceInfo) GetType() string         { return b.Type }
func (b *BasicLibraryInterfaceInfo) GetDataSourceID() string { return b.DataSourceID }
func (b *BasicLibraryInterfaceInfo) GetSchemaName() string   { return b.BasicLibrary.NameEn }
func (b *BasicLibraryInterfaceInfo) GetTableName() string    { return b.NameEn }
func (b *BasicLibraryInterfaceInfo) GetInterfaceConfig() map[string]interface{} {
	return b.InterfaceConfig
}
func (b *BasicLibraryInterfaceInfo) GetParseConfig() map[string]interface{} { return b.ParseConfig }
func (b *BasicLibraryInterfaceInfo) GetTableFieldsConfig() []interface{} {
	if b.TableFieldsConfig == nil {
		return []interface{}{}
	}
	// 将JSONB转换为[]interface{}
	result := make([]interface{}, 0)
	for _, v := range b.TableFieldsConfig {
		result = append(result, v)
	}
	return result
}
func (b *BasicLibraryInterfaceInfo) IsTableCreated() bool { return b.DataInterface.IsTableCreated }

// ThematicLibraryInterfaceInfo 主题库接口信息
type ThematicLibraryInterfaceInfo struct {
	*models.ThematicInterface
}

func (t *ThematicLibraryInterfaceInfo) GetID() string           { return t.ID }
func (t *ThematicLibraryInterfaceInfo) GetName() string         { return t.NameZh }
func (t *ThematicLibraryInterfaceInfo) GetType() string         { return t.Type }
func (t *ThematicLibraryInterfaceInfo) GetDataSourceID() string { return t.DataSourceID }
func (t *ThematicLibraryInterfaceInfo) GetSchemaName() string   { return t.ThematicLibrary.NameEn }
func (t *ThematicLibraryInterfaceInfo) GetTableName() string    { return t.NameEn }
func (t *ThematicLibraryInterfaceInfo) GetInterfaceConfig() map[string]interface{} {
	return t.InterfaceConfig
}
func (t *ThematicLibraryInterfaceInfo) GetParseConfig() map[string]interface{} { return t.ParseConfig }
func (t *ThematicLibraryInterfaceInfo) GetTableFieldsConfig() []interface{} {
	if t.TableFieldsConfig == nil {
		return []interface{}{}
	}
	// 将JSONB转换为[]interface{}
	result := make([]interface{}, 0)
	for _, v := range t.TableFieldsConfig {
		result = append(result, v)
	}
	return result
}
func (t *ThematicLibraryInterfaceInfo) IsTableCreated() bool {
	return t.ThematicInterface.IsTableCreated
}

// getBasicLibraryInterface 获取基础库接口信息
func (e *InterfaceExecutor) getBasicLibraryInterface(interfaceID string) (InterfaceInfo, error) {
	var dataInterface models.DataInterface
	err := e.db.Preload("BasicLibrary").
		Preload("DataSource").Preload("Fields").Preload("CleanRules").
		First(&dataInterface, "id = ?", interfaceID).Error
	if err != nil {
		return nil, err
	}
	return &BasicLibraryInterfaceInfo{&dataInterface}, nil
}

// getThematicLibraryInterface 获取主题库接口信息
func (e *InterfaceExecutor) getThematicLibraryInterface(interfaceID string) (InterfaceInfo, error) {
	var thematicInterface models.ThematicInterface
	err := e.db.Preload("ThematicLibrary").
		Preload("DataSource").
		First(&thematicInterface, "id = ?", interfaceID).Error
	if err != nil {
		return nil, err
	}
	return &ThematicLibraryInterfaceInfo{&thematicInterface}, nil
}

// executePreview 执行预览操作
func (e *InterfaceExecutor) executePreview(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 预览操作：调用一次接口，获取数据并返回
	fmt.Printf("[DEBUG] InterfaceExecutor.executePreview - 开始预览接口: %s\n", interfaceInfo.GetID())

	// 执行数据获取
	data, dataTypes, warnings, err := e.fetchDataFromSource(ctx, interfaceInfo, request.Parameters)
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
	if limit <= 0 || limit > 1000 {
		limit = 10
	}

	// 限制返回的数据量
	limitedData := e.limitDataRows(data, limit)

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

// executeTest 执行测试操作
func (e *InterfaceExecutor) executeTest(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 测试操作：实际执行一次接口同步，更新表数据
	fmt.Printf("[DEBUG] InterfaceExecutor.executeTest - 开始测试接口: %s\n", interfaceInfo.GetID())

	// 执行数据获取
	data, dataTypes, warnings, err := e.fetchDataFromSource(ctx, interfaceInfo, request.Parameters)
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
		updatedRows, err = e.updateTableData(ctx, interfaceInfo, data)
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

// executeSync 执行同步操作
func (e *InterfaceExecutor) executeSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 同步操作：完整的数据同步流程
	fmt.Printf("[DEBUG] InterfaceExecutor.executeSync - 开始同步接口: %s\n", interfaceInfo.GetID())

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

	fmt.Printf("[DEBUG] executeSync - 接口配置: %+v\n", interfaceConfig)
	fmt.Printf("[DEBUG] executeSync - 限制配置: hasLimitConfig=%t, config=%+v\n", hasLimitConfig, limitConfig)

	if hasLimitConfig {
		if limitMap, ok := limitConfig.(map[string]interface{}); ok {
			enabled := cast.ToBool(limitMap["enabled"])
			fmt.Printf("[DEBUG] executeSync - 限制配置启用状态: %t\n", enabled)

			if enabled {
				// 使用批量同步
				return e.executeBatchSync(ctx, interfaceInfo, request, startTime, limitMap)
			}
		}
	}

	// 执行单次数据获取（传统方式）
	data, dataTypes, warnings, err := e.fetchDataFromSource(ctx, interfaceInfo, request.Parameters)
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
	updatedRows, err := e.updateTableData(ctx, interfaceInfo, data)
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

// executeBatchSync 执行批量同步操作
func (e *InterfaceExecutor) executeBatchSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time, limitConfig map[string]interface{}) (*ExecuteResponse, error) {
	fmt.Printf("[DEBUG] executeBatchSync - 开始批量同步，配置: %+v\n", limitConfig)

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

	fmt.Printf("[DEBUG] executeBatchSync - 批量大小: %d, 最大限制: %d\n", batchSize, int(maxLimit))

	// 获取数据源信息
	var dataSource models.DataSource
	if err := e.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		fmt.Printf("[ERROR] executeBatchSync - 获取数据源信息失败: %v\n", err)
		return &ExecuteResponse{
			Success:     false,
			Message:     "获取数据源信息失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 检查数据源类型，数据库和API接口都可能需要批量处理
	fmt.Printf("[DEBUG] executeBatchSync - 数据源类型: %s, 分类: %s\n", dataSource.Type, dataSource.Category)

	// 检查是否支持批量处理
	supportsBatch := dataSource.Category == meta.DataSourceCategoryDatabase || dataSource.Category == meta.DataSourceCategoryAPI
	if !supportsBatch {
		fmt.Printf("[DEBUG] executeBatchSync - 数据源类型不支持批量同步，使用单次同步\n")
		// 对于不支持批量的数据源，回退到单次同步
		data, dataTypes, warnings, err := e.fetchDataFromSource(ctx, interfaceInfo, request.Parameters)
		if err != nil {
			return &ExecuteResponse{
				Success:     false,
				Message:     "同步数据获取失败",
				Duration:    time.Since(startTime).Milliseconds(),
				ExecuteType: request.ExecuteType,
				Error:       err.Error(),
			}, err
		}

		updatedRows, err := e.updateTableData(ctx, interfaceInfo, data)
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
			Message:      "数据同步成功（单次）",
			Duration:     time.Since(startTime).Milliseconds(),
			ExecuteType:  request.ExecuteType,
			Data:         data,
			RowCount:     len(data),
			ColumnCount:  len(dataTypes),
			DataTypes:    dataTypes,
			TableUpdated: true,
			UpdatedRows:  updatedRows,
			Warnings:     warnings,
		}, nil
	}

	// 清空目标表
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())
	fmt.Printf("[DEBUG] executeBatchSync - 清空表: %s\n", fullTableName)

	if err := e.db.Exec(fmt.Sprintf("DELETE FROM %s", fullTableName)).Error; err != nil {
		fmt.Printf("[ERROR] executeBatchSync - 清空表失败: %v\n", err)
		return &ExecuteResponse{
			Success:     false,
			Message:     "清空表数据失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 批量数据同步
	var totalRows int64 = 0
	var allDataTypes map[string]string
	var allWarnings []string
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
		fmt.Printf("[DEBUG] executeBatchSync - API分页配置: pageParam=%s, sizeParam=%s, startPage=%d\n", pageParamName, sizeParamName, startPage)
	} else {
		// 数据库类型：使用标准分页参数
		pageParamName = "page"
		sizeParamName = "page_size"
		startPage = 1
		currentPage = startPage
		fmt.Printf("[DEBUG] executeBatchSync - 数据库分页配置: pageParam=%s, sizeParam=%s, startPage=%d\n", pageParamName, sizeParamName, startPage)
	}

	for hasMoreData {
		fmt.Printf("[DEBUG] executeBatchSync - 处理第 %d 批，批量大小: %d\n", currentPage, batchSize)

		// 构建分页参数
		pageParams := map[string]interface{}{
			pageParamName: currentPage,
			sizeParamName: batchSize,
		}

		// 获取批量数据
		batchData, dataTypes, warnings, err := e.fetchBatchDataFromSource(ctx, interfaceInfo, request.Parameters, pageParams)
		if err != nil {
			fmt.Printf("[ERROR] executeBatchSync - 获取第 %d 批数据失败: %v\n", currentPage, err)
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
			fmt.Printf("[DEBUG] executeBatchSync - 第 %d 批没有数据，结束同步\n", currentPage)
			hasMoreData = false
			break
		}

		// 对于API接口，可能需要检查响应中的分页信息来判断是否有更多数据
		if dataSource.Category == meta.DataSourceCategoryAPI {
			// API接口：检查是否有更多数据的逻辑
			// 1. 如果返回的数据量小于请求的批量大小，说明没有更多数据
			if len(batchData) < batchSize {
				fmt.Printf("[DEBUG] executeBatchSync - API第 %d 批数据量(%d)小于批量大小(%d)，这是最后一批\n", currentPage, len(batchData), batchSize)
				hasMoreData = false
			}

			// 2. 可以通过检查API配置中的分页字段来判断是否有更多数据
			// 这需要在响应处理中实现，这里先使用简单的数据量判断
			interfaceConfig := interfaceInfo.GetInterfaceConfig()

			// 检查是否配置了总数字段或页码字段来进行更精确的判断
			totalField := cast.ToString(interfaceConfig[meta.DataInterfaceConfigFieldTotalField])
			pageField := cast.ToString(interfaceConfig[meta.DataInterfaceConfigFieldPageField])

			if totalField != "" || pageField != "" {
				fmt.Printf("[DEBUG] executeBatchSync - API配置了分页字段(total: %s, page: %s)，可以实现更精确的分页判断\n", totalField, pageField)
				// TODO: 在未来版本中，可以通过解析响应中的这些字段来更精确地判断是否有更多数据
			}
		} else {
			// 数据库接口：数据量小于批量大小说明没有更多数据
			if len(batchData) < batchSize {
				fmt.Printf("[DEBUG] executeBatchSync - 数据库第 %d 批数据量(%d)小于批量大小(%d)，这是最后一批\n", currentPage, len(batchData), batchSize)
				hasMoreData = false
			}
		}

		// 插入批量数据
		batchRows, err := e.insertBatchData(ctx, interfaceInfo, batchData)
		if err != nil {
			fmt.Printf("[ERROR] executeBatchSync - 插入第 %d 批数据失败: %v\n", currentPage, err)
			return &ExecuteResponse{
				Success:     false,
				Message:     fmt.Sprintf("插入第 %d 批数据失败", currentPage),
				Duration:    time.Since(startTime).Milliseconds(),
				ExecuteType: request.ExecuteType,
				Error:       err.Error(),
			}, err
		}

		totalRows += batchRows
		fmt.Printf("[DEBUG] executeBatchSync - 第 %d 批成功插入 %d 行，累计 %d 行\n", currentPage, batchRows, totalRows)

		currentPage++

		// 防止无限循环
		if currentPage > 1000 {
			fmt.Printf("[WARN] executeBatchSync - 达到最大批次限制(1000)，停止同步\n")
			allWarnings = append(allWarnings, "达到最大批次限制，可能还有更多数据未同步")
			break
		}
	}

	fmt.Printf("[DEBUG] executeBatchSync - 批量同步完成，总共处理 %d 批，插入 %d 行\n", currentPage-1, totalRows)

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
		},
	}, nil
}

// fetchDataFromSource 从数据源获取数据
func (e *InterfaceExecutor) fetchDataFromSource(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}) ([]map[string]interface{}, map[string]string, []string, error) {
	fmt.Printf("[DEBUG] fetchDataFromSource - 开始从数据源获取数据\n")
	fmt.Printf("[DEBUG] fetchDataFromSource - 接口ID: %s\n", interfaceInfo.GetID())
	fmt.Printf("[DEBUG] fetchDataFromSource - 数据源ID: %s\n", interfaceInfo.GetDataSourceID())
	fmt.Printf("[DEBUG] fetchDataFromSource - 请求参数: %+v\n", parameters)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := e.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		fmt.Printf("[ERROR] fetchDataFromSource - 获取数据源信息失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchDataFromSource - 数据源信息: %+v\n", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = e.datasourceManager.Get(dataSource.ID)
	if err != nil {
		fmt.Printf("[DEBUG] fetchDataFromSource - 数据源未注册，创建临时实例，错误: %v\n", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = e.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			fmt.Printf("[ERROR] fetchDataFromSource - 创建数据源实例失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			fmt.Printf("[ERROR] fetchDataFromSource - 初始化数据源失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}
		fmt.Printf("[DEBUG] fetchDataFromSource - 临时数据源实例创建并初始化成功\n")
	} else {
		fmt.Printf("[DEBUG] fetchDataFromSource - 使用已注册的数据源实例\n")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	fmt.Printf("[DEBUG] fetchDataFromSource - 接口配置信息:\n")
	fmt.Printf("[DEBUG] fetchDataFromSource - InterfaceConfig: %+v\n", interfaceInfo.GetInterfaceConfig())
	fmt.Printf("[DEBUG] fetchDataFromSource - ParseConfig: %+v\n", interfaceInfo.GetParseConfig())
	fmt.Printf("[DEBUG] fetchDataFromSource - TableFieldsConfig: %+v\n", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		fmt.Printf("[ERROR] fetchDataFromSource - 创建查询构建器失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchDataFromSource - 查询构建器创建成功\n")

	// 构建执行请求
	executeRequest, err := queryBuilder.BuildTestRequest(parameters)
	if err != nil {
		fmt.Printf("[ERROR] fetchDataFromSource - 构建查询请求失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchDataFromSource - 执行请求: %+v\n", executeRequest)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		fmt.Printf("[ERROR] fetchDataFromSource - 执行接口查询失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchDataFromSource - 查询执行成功，响应: %+v\n", response)

	// 处理返回的数据
	data, dataTypes, warnings := e.processResponseData(response.Data)

	fmt.Printf("[DEBUG] fetchDataFromSource - 处理后的数据: %d 行\n", len(data))
	if len(data) > 0 {
		fmt.Printf("[DEBUG] fetchDataFromSource - 第一行数据示例: %+v\n", data[0])
	}
	fmt.Printf("[DEBUG] fetchDataFromSource - 数据类型: %+v\n", dataTypes)
	fmt.Printf("[DEBUG] fetchDataFromSource - 警告信息: %+v\n", warnings)

	return data, dataTypes, warnings, nil
}

// fetchBatchDataFromSource 从数据源获取批量数据（支持分页）
func (e *InterfaceExecutor) fetchBatchDataFromSource(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}, pageParams map[string]interface{}) ([]map[string]interface{}, map[string]string, []string, error) {
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 开始获取批量数据\n")
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 接口ID: %s\n", interfaceInfo.GetID())
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 数据源ID: %s\n", interfaceInfo.GetDataSourceID())
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 请求参数: %+v\n", parameters)
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 分页参数: %+v\n", pageParams)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := e.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		fmt.Printf("[ERROR] fetchBatchDataFromSource - 获取数据源信息失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 数据源信息: %+v\n", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = e.datasourceManager.Get(dataSource.ID)
	if err != nil {
		fmt.Printf("[DEBUG] fetchBatchDataFromSource - 数据源未注册，创建临时实例，错误: %v\n", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = e.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			fmt.Printf("[ERROR] fetchBatchDataFromSource - 创建数据源实例失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			fmt.Printf("[ERROR] fetchBatchDataFromSource - 初始化数据源失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}
		fmt.Printf("[DEBUG] fetchBatchDataFromSource - 临时数据源实例创建并初始化成功\n")
	} else {
		fmt.Printf("[DEBUG] fetchBatchDataFromSource - 使用已注册的数据源实例\n")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 接口配置信息:\n")
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - InterfaceConfig: %+v\n", interfaceInfo.GetInterfaceConfig())
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - ParseConfig: %+v\n", interfaceInfo.GetParseConfig())
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - TableFieldsConfig: %+v\n", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		fmt.Printf("[ERROR] fetchBatchDataFromSource - 创建查询构建器失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 查询构建器创建成功\n")

	// 根据数据源类型构建不同的请求
	var executeRequest *datasource.ExecuteRequest

	// 合并分页参数到基础参数中
	allParams := make(map[string]interface{})
	for k, v := range parameters {
		allParams[k] = v
	}
	for k, v := range pageParams {
		allParams[k] = v
	}

	switch dataSource.Category {
	case meta.DataSourceCategoryDatabase:
		// 数据库类型：使用带分页的同步请求
		executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
		if err != nil {
			fmt.Printf("[ERROR] fetchBatchDataFromSource - 构建数据库分页同步请求失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("构建数据库分页同步请求失败: %w", err)
		}

	case meta.DataSourceCategoryAPI:
		// API类型：检查是否有分页配置
		interfaceConfig := interfaceInfo.GetInterfaceConfig()
		if paginationEnabled := cast.ToBool(interfaceConfig[meta.DataInterfaceConfigFieldPaginationEnabled]); paginationEnabled {
			// 使用带分页的API同步请求
			executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
			if err != nil {
				fmt.Printf("[ERROR] fetchBatchDataFromSource - 构建API分页同步请求失败: %v\n", err)
				return nil, nil, nil, fmt.Errorf("构建API分页同步请求失败: %w", err)
			}
		} else {
			// 没有分页配置，使用普通同步请求
			executeRequest, err = queryBuilder.BuildSyncRequest("full", allParams)
			if err != nil {
				fmt.Printf("[ERROR] fetchBatchDataFromSource - 构建API同步请求失败: %v\n", err)
				return nil, nil, nil, fmt.Errorf("构建API同步请求失败: %w", err)
			}
		}

	default:
		// 其他类型：使用普通测试请求
		executeRequest, err = queryBuilder.BuildTestRequest(parameters)
		if err != nil {
			fmt.Printf("[ERROR] fetchBatchDataFromSource - 构建查询请求失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
		}
	}

	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 执行请求: %+v\n", executeRequest)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		fmt.Printf("[ERROR] fetchBatchDataFromSource - 执行接口查询失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 查询执行成功，响应: %+v\n", response)

	// 处理返回的数据
	data, dataTypes, warnings := e.processResponseData(response.Data)

	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 处理后的数据: %d 行\n", len(data))
	if len(data) > 0 {
		fmt.Printf("[DEBUG] fetchBatchDataFromSource - 第一行数据示例: %+v\n", data[0])
	}
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 数据类型: %+v\n", dataTypes)
	fmt.Printf("[DEBUG] fetchBatchDataFromSource - 警告信息: %+v\n", warnings)

	return data, dataTypes, warnings, nil
}

// insertBatchData 插入批量数据
func (e *InterfaceExecutor) insertBatchData(ctx context.Context, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	fmt.Printf("[DEBUG] insertBatchData - 开始插入批量数据到表: %s\n", fullTableName)
	fmt.Printf("[DEBUG] insertBatchData - 数据行数: %d\n", len(data))

	// 开启事务
	tx := e.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR] insertBatchData - 事务回滚，原因: %v\n", r)
			tx.Rollback()
		}
	}()

	// 插入数据
	var insertedRows int64
	for i, row := range data {
		fmt.Printf("[DEBUG] insertBatchData - 处理第 %d 行数据: %+v\n", i+1, row)

		// 应用parseConfig中的fieldMapping
		parseConfig := interfaceInfo.GetParseConfig()
		mappedRow := e.applyFieldMapping(row, parseConfig)
		fmt.Printf("[DEBUG] insertBatchData - 字段映射后的数据: %+v\n", mappedRow)

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := e.processValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 修复SQL格式错误 - 使用strings.Join而不是fmt.Sprintf
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		fmt.Printf("[DEBUG] insertBatchData - 插入SQL: %s\n", insertSQL)
		fmt.Printf("[DEBUG] insertBatchData - 插入参数: %+v\n", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] insertBatchData - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] insertBatchData - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] insertBatchData - 失败的参数: %+v\n", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("[ERROR] insertBatchData - 提交事务失败: %v\n", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("[DEBUG] insertBatchData - 成功插入 %d 行数据\n", insertedRows)
	return insertedRows, nil
}

// processResponseData 处理响应数据
func (e *InterfaceExecutor) processResponseData(result interface{}) ([]map[string]interface{}, map[string]string, []string) {
	var sampleData []map[string]interface{}
	var warnings []string

	// 根据返回结果的类型进行处理
	switch data := result.(type) {
	case []map[string]interface{}:
		sampleData = data
	case []interface{}:
		// 转换为 []map[string]interface{}
		for _, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				sampleData = append(sampleData, itemMap)
			}
		}
	case map[string]interface{}:
		// 单条记录
		sampleData = []map[string]interface{}{data}
	default:
		// 尝试解析JSON
		if jsonBytes, ok := result.([]byte); ok {
			var jsonData interface{}
			if err := json.Unmarshal(jsonBytes, &jsonData); err == nil {
				if jsonArray, ok := jsonData.([]interface{}); ok {
					for _, item := range jsonArray {
						if itemMap, ok := item.(map[string]interface{}); ok {
							sampleData = append(sampleData, itemMap)
						}
					}
				}
			}
		}
	}

	// 分析数据类型
	dataTypes := e.analyzeDataTypes(sampleData)

	// 生成警告
	if len(sampleData) > 1000 {
		warnings = append(warnings, "数据量较大，建议分页查询")
	}
	if len(sampleData) == 0 {
		warnings = append(warnings, "接口返回空数据")
	}

	return sampleData, dataTypes, warnings
}

// analyzeDataTypes 分析数据类型
func (e *InterfaceExecutor) analyzeDataTypes(data []map[string]interface{}) map[string]string {
	dataTypes := make(map[string]string)

	if len(data) == 0 {
		return dataTypes
	}

	// 使用第一行数据分析字段类型
	firstRow := data[0]
	for key, value := range firstRow {
		if value == nil {
			dataTypes[key] = "null"
			continue
		}

		switch reflect.TypeOf(value).Kind() {
		case reflect.String:
			dataTypes[key] = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			dataTypes[key] = "integer"
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dataTypes[key] = "integer"
		case reflect.Float32, reflect.Float64:
			dataTypes[key] = "float"
		case reflect.Bool:
			dataTypes[key] = "boolean"
		default:
			// 尝试检测是否为日期时间字符串
			if str, ok := value.(string); ok {
				if e.isDateTime(str) {
					dataTypes[key] = "datetime"
				} else {
					dataTypes[key] = "string"
				}
			} else {
				dataTypes[key] = "object"
			}
		}
	}

	return dataTypes
}

// isDateTime 检测字符串是否为日期时间格式
func (e *InterfaceExecutor) isDateTime(str string) bool {
	// 常见的日期时间格式
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, str); err == nil {
			return true
		}
	}
	return false
}

// limitDataRows 限制数据行数
func (e *InterfaceExecutor) limitDataRows(data []map[string]interface{}, limit int) []map[string]interface{} {
	if len(data) <= limit {
		return data
	}
	return data[:limit]
}

// updateTableData 更新表数据
func (e *InterfaceExecutor) updateTableData(ctx context.Context, interfaceInfo InterfaceInfo, data []map[string]interface{}) (int64, error) {
	// 构造表名
	fullTableName := fmt.Sprintf(`"%s"."%s"`, interfaceInfo.GetSchemaName(), interfaceInfo.GetTableName())

	fmt.Printf("[DEBUG] updateTableData - 开始更新表数据\n")
	fmt.Printf("[DEBUG] updateTableData - 表名: %s\n", fullTableName)
	fmt.Printf("[DEBUG] updateTableData - 数据行数: %d\n", len(data))

	// 打印parseConfig信息
	parseConfig := interfaceInfo.GetParseConfig()
	fmt.Printf("[DEBUG] updateTableData - parseConfig: %+v\n", parseConfig)

	if len(data) > 0 {
		fmt.Printf("[DEBUG] updateTableData - 第一行数据示例: %+v\n", data[0])
	}

	// 这里应该根据接口类型（实时/批量）和配置来决定更新策略
	// 简化实现：清空表后插入新数据

	// 开启事务
	tx := e.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ERROR] updateTableData - 事务回滚，原因: %v\n", r)
			tx.Rollback()
		}
	}()

	// 清空现有数据
	deleteSQL := fmt.Sprintf("DELETE FROM %s", fullTableName)
	fmt.Printf("[DEBUG] updateTableData - 清空表SQL: %s\n", deleteSQL)

	if err := tx.Exec(deleteSQL).Error; err != nil {
		fmt.Printf("[ERROR] updateTableData - 清空表数据失败: %v\n", err)
		tx.Rollback()
		return 0, fmt.Errorf("清空表数据失败: %w", err)
	}

	// 插入新数据
	var insertedRows int64
	for i, row := range data {
		fmt.Printf("[DEBUG] updateTableData - 处理第 %d 行数据: %+v\n", i+1, row)

		// 应用parseConfig中的fieldMapping
		mappedRow := e.applyFieldMapping(row, parseConfig)
		fmt.Printf("[DEBUG] updateTableData - 字段映射后的数据: %+v\n", mappedRow)

		// 构建插入SQL
		columns := make([]string, 0, len(mappedRow))
		placeholders := make([]string, 0, len(mappedRow))
		values := make([]interface{}, 0, len(mappedRow))

		for col, val := range mappedRow {
			columns = append(columns, fmt.Sprintf(`"%s"`, col))
			placeholders = append(placeholders, "?")
			// 处理数据类型转换，特别是时间字段
			processedVal := e.processValueForDatabase(col, val)
			values = append(values, processedVal)
		}

		// 修复SQL格式错误 - 使用strings.Join而不是fmt.Sprintf
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			fullTableName,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		fmt.Printf("[DEBUG] updateTableData - 插入SQL: %s\n", insertSQL)
		fmt.Printf("[DEBUG] updateTableData - 插入参数: %+v\n", values)

		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			fmt.Printf("[ERROR] updateTableData - 插入数据失败: %v\n", err)
			fmt.Printf("[ERROR] updateTableData - 失败的SQL: %s\n", insertSQL)
			fmt.Printf("[ERROR] updateTableData - 失败的参数: %+v\n", values)
			tx.Rollback()
			return 0, fmt.Errorf("插入数据失败: %w", err)
		}
		insertedRows++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("[ERROR] updateTableData - 提交事务失败: %v\n", err)
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("[DEBUG] updateTableData - 成功插入 %d 行数据\n", insertedRows)
	return insertedRows, nil
}

// applyFieldMapping 应用字段映射配置
func (e *InterfaceExecutor) applyFieldMapping(row map[string]interface{}, parseConfig map[string]interface{}) map[string]interface{} {
	fmt.Printf("[DEBUG] applyFieldMapping - 原始数据: %+v\n", row)
	fmt.Printf("[DEBUG] applyFieldMapping - parseConfig: %+v\n", parseConfig)

	// 如果没有parseConfig，直接返回原始数据
	if parseConfig == nil {
		fmt.Printf("[DEBUG] applyFieldMapping - parseConfig为空，返回原始数据\n")
		return row
	}

	// 获取fieldMapping配置
	fieldMappingInterface, exists := parseConfig["fieldMapping"]
	if !exists {
		fmt.Printf("[DEBUG] applyFieldMapping - 没有fieldMapping配置，返回原始数据\n")
		return row
	}

	fmt.Printf("[DEBUG] applyFieldMapping - fieldMapping原始配置: %+v\n", fieldMappingInterface)

	// 支持两种格式：数组格式（新）和对象格式（旧）
	var fieldMappingArray []interface{}
	var fieldMappingMap map[string]interface{}
	var isArrayFormat bool

	// 尝试解析为数组格式（新格式）
	if mappingArray, ok := fieldMappingInterface.([]interface{}); ok {
		fieldMappingArray = mappingArray
		isArrayFormat = true
		fmt.Printf("[DEBUG] applyFieldMapping - 使用数组格式fieldMapping，条目数: %d\n", len(fieldMappingArray))
	} else if mappingMap, ok := fieldMappingInterface.(map[string]interface{}); ok {
		// 兼容旧的对象格式
		fieldMappingMap = mappingMap
		isArrayFormat = false
		fmt.Printf("[DEBUG] applyFieldMapping - 使用对象格式fieldMapping（兼容模式）\n")
	} else {
		fmt.Printf("[DEBUG] applyFieldMapping - fieldMapping格式不支持，返回原始数据\n")
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
					fmt.Printf("[DEBUG] applyFieldMapping - 映射规则: %s -> %s\n", source, target)
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
			fmt.Printf("[DEBUG] applyFieldMapping - 字段映射: %s -> %s, 值: %v\n", sourceField, targetField, value)
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
			fmt.Printf("[DEBUG] applyFieldMapping - 字段映射（兼容模式）: %s -> %s, 值: %v\n", sourceField, targetField, value)
		}
	}

	fmt.Printf("[DEBUG] applyFieldMapping - 映射后数据: %+v\n", mappedRow)
	return mappedRow
}

// processValueForDatabase 处理数据库值，特别是时间字段格式转换
func (e *InterfaceExecutor) processValueForDatabase(columnName string, value interface{}) interface{} {
	if value == nil {
		return value
	}

	fmt.Printf("[DEBUG] processValueForDatabase - 处理字段: %s, 原始值: %+v, 类型: %T\n", columnName, value, value)

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
			fmt.Printf("[DEBUG] processValueForDatabase - 时间字段转换: %s -> %s\n", v.String(), formatted)
			return formatted
		case string:
			// 尝试解析字符串时间并重新格式化
			if parsedTime, err := time.Parse(time.RFC3339, v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				fmt.Printf("[DEBUG] processValueForDatabase - 字符串时间转换(RFC3339): %s -> %s\n", v, formatted)
				return formatted
			}
			if parsedTime, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				fmt.Printf("[DEBUG] processValueForDatabase - 字符串时间转换(标准): %s -> %s\n", v, formatted)
				return formatted
			}
			if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				fmt.Printf("[DEBUG] processValueForDatabase - 字符串时间转换(ISO): %s -> %s\n", v, formatted)
				return formatted
			}
			if parsedTime, err := time.Parse("2006-01-02T15:04:05.000Z", v); err == nil {
				formatted := parsedTime.Format("2006-01-02 15:04:05.000")
				fmt.Printf("[DEBUG] processValueForDatabase - 字符串时间转换(ISO毫秒): %s -> %s\n", v, formatted)
				return formatted
			}
			// 如果无法解析，返回原值
			fmt.Printf("[DEBUG] processValueForDatabase - 无法解析时间字符串，返回原值: %s\n", v)
			return v
		default:
			// 尝试转换为字符串再解析
			if timeStr := cast.ToString(v); timeStr != "" {
				if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
					formatted := parsedTime.Format("2006-01-02 15:04:05.000")
					fmt.Printf("[DEBUG] processValueForDatabase - 其他类型时间转换: %v -> %s\n", v, formatted)
					return formatted
				}
			}
			fmt.Printf("[DEBUG] processValueForDatabase - 无法处理的时间类型，返回原值: %v\n", v)
			return v
		}
	}

	// 非时间字段，直接返回原值
	fmt.Printf("[DEBUG] processValueForDatabase - 非时间字段，返回原值: %v\n", value)
	return value
}

// executeIncrementalSync 执行增量同步
func (e *InterfaceExecutor) executeIncrementalSync(ctx context.Context, interfaceInfo InterfaceInfo, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	// 获取数据源
	dataSourceID := interfaceInfo.GetDataSourceID()
	dataSource, err := e.datasourceManager.Get(dataSourceID)
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
	err = e.db.First(&dataSourceModel, "id = ?", dataSourceID).Error
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
	data, dataTypes, warnings := e.processResponseData(response.Data)

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
	syncResult, err := e.dataSyncEngine.ExecuteSync(ctx, DataIncrementalSync, data, target)
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

// upsertDataToTable 执行数据的UPSERT操作（增量同步）
func (e *InterfaceExecutor) upsertDataToTable(data []map[string]interface{}, schemaName, tableName string) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	fullTableName := fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)
	var insertedRows int64 = 0

	// 使用错误处理器包装事务操作
	err := e.errorHandler.WrapWithTransaction(e.db, func(tx *gorm.DB) error {
		// 分批处理数据，避免单个事务过大
		batchSize := 1000
		for i := 0; i < len(data); i += batchSize {
			end := i + batchSize
			if end > len(data) {
				end = len(data)
			}
			batch := data[i:end]

			// 处理单个批次
			if err := e.processBatch(tx, batch, fullTableName); err != nil {
				return fmt.Errorf("处理批次 %d-%d 失败: %w", i, end, err)
			}
			insertedRows += int64(len(batch))
		}
		return nil
	})

	if err != nil {
		errorDetail := e.errorHandler.HandleError(
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
func (e *InterfaceExecutor) processBatch(tx *gorm.DB, batch []map[string]interface{}, fullTableName string) error {
	for _, row := range batch {
		if err := e.processRow(tx, row, fullTableName); err != nil {
			return fmt.Errorf("处理行数据失败: %w", err)
		}
	}
	return nil
}

// processRow 处理单行数据
func (e *InterfaceExecutor) processRow(tx *gorm.DB, row map[string]interface{}, fullTableName string) error {
	if len(row) == 0 {
		return nil
	}

	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	values := make([]interface{}, 0, len(row))

	for col, val := range row {
		// 数据验证
		if col == "" {
			errorDetail := e.errorHandler.CreateValidationError("column_name", col, "non_empty")
			return fmt.Errorf("列名验证失败: %s", errorDetail.Message)
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
		// 处理数据库错误
		errorDetail := e.errorHandler.HandleError(
			context.Background(),
			err,
			ErrorTypeQuery,
			fmt.Sprintf("execute insert SQL: %s", insertSQL),
		)
		return fmt.Errorf("执行插入SQL失败: %s", errorDetail.Message)
	}

	return nil
}

// validateRequest 验证执行请求
func (e *InterfaceExecutor) validateRequest(request *ExecuteRequest) error {
	if request.InterfaceID == "" {
		return fmt.Errorf("接口ID不能为空")
	}

	if request.InterfaceType == "" {
		return fmt.Errorf("接口类型不能为空")
	}

	// 验证接口类型
	validInterfaceTypes := []string{"basic_library", "thematic_library"}
	validType := false
	for _, vt := range validInterfaceTypes {
		if request.InterfaceType == vt {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("无效的接口类型: %s", request.InterfaceType)
	}

	if request.ExecuteType == "" {
		return fmt.Errorf("执行类型不能为空")
	}

	// 验证执行类型
	validExecuteTypes := []string{"preview", "test", "sync", "incremental_sync"}
	validExecute := false
	for _, ve := range validExecuteTypes {
		if request.ExecuteType == ve {
			validExecute = true
			break
		}
	}
	if !validExecute {
		return fmt.Errorf("无效的执行类型: %s", request.ExecuteType)
	}

	// 增量同步特殊验证
	if request.ExecuteType == "incremental_sync" {
		if request.LastSyncTime == nil {
			return fmt.Errorf("增量同步需要提供LastSyncTime参数")
		}
		if request.IncrementalKey == "" {
			return fmt.Errorf("增量同步需要提供IncrementalKey参数")
		}
	}

	return nil
}

// inferDataTypes 推断数据类型
func (e *InterfaceExecutor) inferDataTypes(data []map[string]interface{}) map[string]string {
	if len(data) == 0 {
		return make(map[string]string)
	}

	dataTypes := make(map[string]string)

	// 遍历所有记录来推断数据类型
	for _, record := range data {
		for column, value := range record {
			if value == nil {
				continue // 跳过nil值
			}

			// 如果已经有类型定义，跳过
			if _, exists := dataTypes[column]; exists {
				continue
			}

			// 根据值的类型推断数据库类型
			switch v := value.(type) {
			case int, int8, int16, int32, int64:
				dataTypes[column] = "INTEGER"
			case uint, uint8, uint16, uint32, uint64:
				dataTypes[column] = "INTEGER"
			case float32, float64:
				dataTypes[column] = "REAL"
			case bool:
				dataTypes[column] = "BOOLEAN"
			case time.Time:
				dataTypes[column] = "DATETIME"
			case string:
				dataTypes[column] = "TEXT"
			default:
				// 未知类型，默认为TEXT
				dataTypes[column] = "TEXT"
				_ = v // 避免未使用变量警告
			}
		}
	}

	// 对于仍然没有类型的列（所有值都是nil），设为TEXT
	if len(data) > 0 {
		for column := range data[0] {
			if _, exists := dataTypes[column]; !exists {
				dataTypes[column] = "TEXT"
			}
		}
	}

	return dataTypes
}
