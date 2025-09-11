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
	"fmt"
	"time"

	"gorm.io/gorm"
)

// InterfaceExecutor 通用接口执行器
type InterfaceExecutor struct {
	db                *gorm.DB
	datasourceManager datasource.DataSourceManager
	dataSyncEngine    *DataSyncEngine
	errorHandler      *ErrorHandler
	infoProvider      *InterfaceInfoProvider
	executeOps        *ExecuteOperations
}

// NewInterfaceExecutor 创建接口执行器实例
func NewInterfaceExecutor(db *gorm.DB, datasourceManager datasource.DataSourceManager) *InterfaceExecutor {
	executor := &InterfaceExecutor{
		db:                db,
		datasourceManager: datasourceManager,
		dataSyncEngine:    NewDataSyncEngine(db),
		errorHandler:      NewErrorHandler(),
		infoProvider:      NewInterfaceInfoProvider(db),
	}

	// 创建执行操作处理器（需要引用executor）
	executor.executeOps = NewExecuteOperations(executor)

	return executor
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

	// 验证请求参数
	if err := e.validateRequest(request); err != nil {
		return &ExecuteResponse{
			Success:     false,
			Message:     "请求参数验证失败",
			Duration:    time.Since(startTime).Milliseconds(),
			ExecuteType: request.ExecuteType,
			Error:       err.Error(),
		}, err
	}

	// 根据接口类型获取接口信息
	var interfaceInfo InterfaceInfo
	var err error

	switch request.InterfaceType {
	case "basic_library":
		interfaceInfo, err = e.infoProvider.GetBasicLibraryInterface(request.InterfaceID)
	case "thematic_library":
		interfaceInfo, err = e.infoProvider.GetThematicLibraryInterface(request.InterfaceID)
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
		return e.executeOps.ExecutePreview(ctx, interfaceInfo, request, startTime)
	case "test":
		return e.executeOps.ExecuteTest(ctx, interfaceInfo, request, startTime)
	case "sync":
		return e.executeOps.ExecuteSync(ctx, interfaceInfo, request, startTime)
	case "incremental_sync":
		return e.executeOps.ExecuteIncrementalSync(ctx, interfaceInfo, request, startTime)
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

// GetDataSyncEngine 获取数据同步引擎（供其他组件使用）
func (e *InterfaceExecutor) GetDataSyncEngine() *DataSyncEngine {
	return e.dataSyncEngine
}

// GetErrorHandler 获取错误处理器（供其他组件使用）
func (e *InterfaceExecutor) GetErrorHandler() *ErrorHandler {
	return e.errorHandler
}

// GetDB 获取数据库连接（供其他组件使用）
func (e *InterfaceExecutor) GetDB() *gorm.DB {
	return e.db
}

// GetDatasourceManager 获取数据源管理器（供其他组件使用）
func (e *InterfaceExecutor) GetDatasourceManager() datasource.DataSourceManager {
	return e.datasourceManager
}

// 为了兼容现有测试，保留一些方法的包装器
// 这些方法将委托给相应的组件

// processBatch 处理单个数据批次 (兼容性方法)
func (e *InterfaceExecutor) processBatch(tx *gorm.DB, batch []map[string]interface{}, fullTableName string) error {
	fieldMapper := NewFieldMapper()
	return fieldMapper.processBatch(tx, batch, fullTableName)
}

// processRow 处理单行数据 (兼容性方法)
func (e *InterfaceExecutor) processRow(tx *gorm.DB, row map[string]interface{}, fullTableName string) error {
	fieldMapper := NewFieldMapper()
	return fieldMapper.processRow(tx, row, fullTableName)
}

// upsertDataToTable 执行数据的UPSERT操作 (兼容性方法)
func (e *InterfaceExecutor) upsertDataToTable(data []map[string]interface{}, schemaName, tableName string) (int64, error) {
	fieldMapper := NewFieldMapper()
	return fieldMapper.UpsertDataToTable(e.db, e.errorHandler, data, schemaName, tableName)
}

// inferDataTypes 推断数据类型 (兼容性方法)
func (e *InterfaceExecutor) inferDataTypes(data []map[string]interface{}) map[string]string {
	dataProcessor := NewDataProcessor(e)
	return dataProcessor.InferDataTypes(data)
}
