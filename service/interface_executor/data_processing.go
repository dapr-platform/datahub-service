/*
 * @module service/interface_executor/data_processing
 * @description 数据处理相关方法，包括数据获取、响应处理、类型分析等
 * @architecture 工厂模式 - 提供统一的数据处理接口
 * @documentReference ai_docs/interface_executor.md
 * @stateFlow 数据获取 -> 数据解析 -> 类型推断 -> 数据清洗 -> 结果返回
 * @rules 确保数据处理的一致性和可靠性，支持多种数据源类型
 * @dependencies datahub-service/service/datasource, datahub-service/service/meta
 * @refs executor.go, execute_operations.go
 */

package interface_executor

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/spf13/cast"
)

// DataProcessor 数据处理器
type DataProcessor struct {
	executor *InterfaceExecutor
}

// NewDataProcessor 创建数据处理器
func NewDataProcessor(executor *InterfaceExecutor) *DataProcessor {
	return &DataProcessor{executor: executor}
}

// FetchDataFromSource 从数据源获取数据
func (dp *DataProcessor) FetchDataFromSource(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}) ([]map[string]interface{}, map[string]string, []string, error) {
	return dp.FetchDataFromSourceWithExecuteType(ctx, interfaceInfo, parameters, "test")
}

// FetchDataFromSourceWithExecuteType 从数据源获取数据（支持指定执行类型）
func (dp *DataProcessor) FetchDataFromSourceWithExecuteType(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}, executeType string) ([]map[string]interface{}, map[string]string, []string, error) {
	// 将executeType转换为syncStrategy
	syncStrategy := "full"
	if executeType == "incremental_sync" {
		syncStrategy = "incremental"
	}

	return dp.FetchDataFromSourceWithSyncStrategy(ctx, interfaceInfo, parameters, syncStrategy)
}

// FetchDataFromSourceWithSyncStrategy 从数据源获取数据（支持指定同步策略）
func (dp *DataProcessor) FetchDataFromSourceWithSyncStrategy(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}, syncStrategy string) ([]map[string]interface{}, map[string]string, []string, error) {
	slog.Debug("DataProcessor.FetchDataFromSourceWithSyncStrategy - 开始从数据源获取数据",
		"strategy", syncStrategy)
	slog.Debug("FetchDataFromSource - 接口ID", "interface_id", interfaceInfo.GetID())
	slog.Debug("FetchDataFromSource - 数据源ID", "datasource_id", interfaceInfo.GetDataSourceID())
	slog.Debug("FetchDataFromSource - 请求参数", "parameters", parameters)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := dp.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		slog.Error("FetchDataFromSource - 获取数据源信息失败", "error", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	slog.Debug("FetchDataFromSource - 数据源信息", "datasource", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = dp.executor.datasourceManager.Get(dataSource.ID)
	if err != nil {
		slog.Debug("FetchDataFromSource - 数据源未注册，创建临时实例", "error", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = dp.executor.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			slog.Error("FetchDataFromSource - 创建数据源实例失败", "error", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			slog.Error("FetchDataFromSource - 初始化数据源失败", "error", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}

		// 启动数据源（对于常驻数据源如数据库，需要建立连接池）
		if err := dsInstance.Start(ctx); err != nil {
			slog.Error("FetchDataFromSource - 启动数据源失败", "error", err)
			return nil, nil, nil, fmt.Errorf("启动数据源失败: %w", err)
		}

		slog.Debug("FetchDataFromSource - 临时数据源实例创建、初始化并启动成功")

		// 确保在函数返回前关闭临时数据源
		defer func() {
			if err := dsInstance.Stop(context.Background()); err != nil {
				slog.Warn("FetchDataFromSource - 关闭临时数据源失败", "error", err)
			} else {
				slog.Debug("FetchDataFromSource - 临时数据源已关闭")
			}
		}()
	} else {
		slog.Debug("FetchDataFromSource - 使用已注册的数据源实例")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	slog.Debug("FetchDataFromSource - 接口配置信息:")
	slog.Debug("FetchDataFromSource - InterfaceConfig", "data", interfaceInfo.GetInterfaceConfig())
	slog.Debug("FetchDataFromSource - ParseConfig", "data", interfaceInfo.GetParseConfig())
	slog.Debug("FetchDataFromSource - TableFieldsConfig", "data", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		slog.Error("FetchDataFromSource - 创建查询构建器失败", "error", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	slog.Debug("FetchDataFromSource - 查询构建器创建成功")

	// 根据同步策略构建不同的请求
	var executeRequest *datasource.ExecuteRequest
	switch syncStrategy {
	case "full":
		executeRequest, err = queryBuilder.BuildSyncRequest("full", parameters)
	case "incremental":
		// 如果有增量参数，使用增量请求构建器
		if incrementalField, exists := parameters["incremental_field"]; exists {
			incrementalParams := &datasource.IncrementalParams{
				LastSyncValue:  parameters["last_sync_value"],
				IncrementalKey: cast.ToString(incrementalField),
				ComparisonType: cast.ToString(parameters["comparison_type"]),
				BatchSize:      cast.ToInt(parameters["batch_size"]),
			}
			if incrementalParams.ComparisonType == "" {
				incrementalParams.ComparisonType = "gt"
			}
			if incrementalParams.BatchSize <= 0 {
				incrementalParams.BatchSize = 1000
			}

			executeRequest, err = queryBuilder.BuildIncrementalRequest("sync", incrementalParams)
		} else {
			executeRequest, err = queryBuilder.BuildSyncRequest("incremental", parameters)
		}
	default:
		executeRequest, err = queryBuilder.BuildTestRequest(parameters)
	}

	if err != nil {
		slog.Error("FetchDataFromSource - 构建查询请求失败", "error", err)
		return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
	}

	slog.Debug("FetchDataFromSource - 执行请求详情:")
	slog.Debug("FetchDataFromSource - Operation", "value", executeRequest.Operation)
	slog.Debug("FetchDataFromSource - Query", "value", executeRequest.Query)
	slog.Debug("FetchDataFromSource - Data", "data", executeRequest.Data)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		slog.Error("FetchDataFromSource - 执行接口查询失败", "error", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	slog.Debug("FetchDataFromSource - 查询执行成功，响应", "data", response)

	// 检查响应是否成功
	if !response.Success {
		errorMsg := response.Message
		if errorMsg == "" {
			errorMsg = "接口调用失败"
		}
		// 如果有错误详情，添加到错误消息中
		if response.Error != "" {
			errorMsg = fmt.Sprintf("%s: %s", errorMsg, response.Error)
		}
		slog.Error("FetchDataFromSource - 接口返回错误", "message", errorMsg)
		return nil, nil, nil, fmt.Errorf("接口调用失败: %s", errorMsg)
	}

	// 处理返回的数据
	data, dataTypes, warnings := dp.ProcessResponseData(response.Data)

	slog.Debug("FetchDataFromSource - 处理后的数据", "row_count", len(data))
	if len(data) > 0 {
		slog.Debug("FetchDataFromSource - 第一行数据示例", "data", data[0])
	}
	slog.Debug("FetchDataFromSource - 数据类型", "types", dataTypes)
	slog.Debug("FetchDataFromSource - 警告信息", "warnings", warnings)

	return data, dataTypes, warnings, nil
}

// FetchBatchDataFromSource 从数据源获取批量数据（支持分页）
func (dp *DataProcessor) FetchBatchDataFromSource(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}, pageParams map[string]interface{}) ([]map[string]interface{}, map[string]string, []string, error) {
	slog.Debug("DataProcessor.FetchBatchDataFromSource - 开始获取批量数据")
	slog.Debug("FetchBatchDataFromSource - 接口ID", "value", interfaceInfo.GetID())
	slog.Debug("FetchBatchDataFromSource - 数据源ID", "value", interfaceInfo.GetDataSourceID())
	slog.Debug("FetchBatchDataFromSource - 请求参数", "data", parameters)
	slog.Debug("FetchBatchDataFromSource - 分页参数", "data", pageParams)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := dp.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		slog.Error("FetchBatchDataFromSource - 获取数据源信息失败", "error", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSource - 数据源信息", "data", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = dp.executor.datasourceManager.Get(dataSource.ID)
	if err != nil {
		slog.Debug("FetchBatchDataFromSource - 数据源未注册，创建临时实例，错误", "value", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = dp.executor.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			slog.Error("FetchBatchDataFromSource - 创建数据源实例失败", "error", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			slog.Error("FetchBatchDataFromSource - 初始化数据源失败", "error", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}

		// 启动数据源（对于常驻数据源如数据库，需要建立连接池）
		if err := dsInstance.Start(ctx); err != nil {
			slog.Error("FetchBatchDataFromSource - 启动数据源失败", "error", err)
			return nil, nil, nil, fmt.Errorf("启动数据源失败: %w", err)
		}

		slog.Debug("FetchBatchDataFromSource - 临时数据源实例创建、初始化并启动成功")

		// 确保在函数返回前关闭临时数据源
		defer func() {
			if err := dsInstance.Stop(context.Background()); err != nil {
				slog.Warn("FetchBatchDataFromSource - 关闭临时数据源失败", "error", err)
			} else {
				slog.Debug("FetchBatchDataFromSource - 临时数据源已关闭")
			}
		}()
	} else {
		slog.Debug("FetchBatchDataFromSource - 使用已注册的数据源实例")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	slog.Debug("FetchBatchDataFromSource - 接口配置信息:")
	slog.Debug("FetchBatchDataFromSource - InterfaceConfig", "data", interfaceInfo.GetInterfaceConfig())
	slog.Debug("FetchBatchDataFromSource - ParseConfig", "data", interfaceInfo.GetParseConfig())
	slog.Debug("FetchBatchDataFromSource - TableFieldsConfig", "data", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		slog.Error("FetchBatchDataFromSource - 创建查询构建器失败", "error", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSource - 查询构建器创建成功")

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
			slog.Error("FetchBatchDataFromSource - 构建数据库分页同步请求失败", "error", err)
			return nil, nil, nil, fmt.Errorf("构建数据库分页同步请求失败: %w", err)
		}

	case meta.DataSourceCategoryAPI:
		// API类型：检查是否有分页配置
		interfaceConfig := interfaceInfo.GetInterfaceConfig()
		if paginationEnabled := cast.ToBool(interfaceConfig[meta.DataInterfaceConfigFieldPaginationEnabled]); paginationEnabled {
			// 使用带分页的API同步请求
			executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
			if err != nil {
				slog.Error("FetchBatchDataFromSource - 构建API分页同步请求失败", "error", err)
				return nil, nil, nil, fmt.Errorf("构建API分页同步请求失败: %w", err)
			}
		} else {
			// 没有分页配置，使用普通同步请求
			executeRequest, err = queryBuilder.BuildSyncRequest("full", allParams)
			if err != nil {
				slog.Error("FetchBatchDataFromSource - 构建API同步请求失败", "error", err)
				return nil, nil, nil, fmt.Errorf("构建API同步请求失败: %w", err)
			}
		}

	default:
		// 其他类型：使用普通测试请求
		executeRequest, err = queryBuilder.BuildTestRequest(parameters)
		if err != nil {
			slog.Error("FetchBatchDataFromSource - 构建查询请求失败", "error", err)
			return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
		}
	}

	slog.Debug("FetchBatchDataFromSource - 执行请求", "data", executeRequest)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		slog.Error("FetchBatchDataFromSource - 执行接口查询失败", "error", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSource - 查询执行成功，响应", "data", response)

	// 检查响应是否成功
	if !response.Success {
		errorMsg := response.Message
		if errorMsg == "" {
			errorMsg = "接口调用失败"
		}
		// 如果有错误详情，添加到错误消息中
		if response.Error != "" {
			errorMsg = fmt.Sprintf("%s: %s", errorMsg, response.Error)
		}
		slog.Error("FetchBatchDataFromSource - 接口返回错误", "message", errorMsg)
		return nil, nil, nil, fmt.Errorf("接口调用失败: %s", errorMsg)
	}

	// 处理返回的数据
	data, dataTypes, warnings := dp.ProcessResponseData(response.Data)

	slog.Debug("FetchBatchDataFromSource - 处理后的数据", "row_count", len(data))
	if len(data) > 0 {
		slog.Debug("FetchBatchDataFromSource - 第一行数据示例", "data", data[0])
	}
	slog.Debug("FetchBatchDataFromSource - 数据类型", "types", dataTypes)
	slog.Debug("FetchBatchDataFromSource - 警告信息", "warnings", warnings)

	return data, dataTypes, warnings, nil
}

// FetchBatchDataFromSourceWithStrategy 从数据源获取批量数据（支持指定同步策略）
func (dp *DataProcessor) FetchBatchDataFromSourceWithStrategy(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}, pageParams map[string]interface{}, syncStrategy string) ([]map[string]interface{}, map[string]string, []string, error) {
	slog.Debug("DataProcessor.FetchBatchDataFromSourceWithStrategy - 开始获取批量数据，策略", "value", syncStrategy)
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 接口ID", "value", interfaceInfo.GetID())
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 数据源ID", "value", interfaceInfo.GetDataSourceID())
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 请求参数", "data", parameters)
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 分页参数", "data", pageParams)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := dp.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		slog.Error("FetchBatchDataFromSourceWithStrategy - 获取数据源信息失败", "error", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSourceWithStrategy - 数据源信息", "data", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = dp.executor.datasourceManager.Get(dataSource.ID)
	if err != nil {
		slog.Debug("FetchBatchDataFromSourceWithStrategy - 数据源未注册，创建临时实例，错误", "value", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = dp.executor.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			slog.Error("FetchBatchDataFromSourceWithStrategy - 创建数据源实例失败", "error", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			slog.Error("FetchBatchDataFromSourceWithStrategy - 初始化数据源失败", "error", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}

		// 启动数据源（对于常驻数据源如数据库，需要建立连接池）
		if err := dsInstance.Start(ctx); err != nil {
			slog.Error("FetchBatchDataFromSourceWithStrategy - 启动数据源失败", "error", err)
			return nil, nil, nil, fmt.Errorf("启动数据源失败: %w", err)
		}

		slog.Debug("FetchBatchDataFromSourceWithStrategy - 临时数据源实例创建、初始化并启动成功")

		// 确保在函数返回前关闭临时数据源
		defer func() {
			if err := dsInstance.Stop(context.Background()); err != nil {
				slog.Warn("FetchBatchDataFromSourceWithStrategy - 关闭临时数据源失败", "error", err)
			} else {
				slog.Debug("FetchBatchDataFromSourceWithStrategy - 临时数据源已关闭")
			}
		}()
	} else {
		slog.Debug("FetchBatchDataFromSourceWithStrategy - 使用已注册的数据源实例")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	slog.Debug("FetchBatchDataFromSourceWithStrategy - 接口配置信息:")
	slog.Debug("FetchBatchDataFromSourceWithStrategy - InterfaceConfig", "data", interfaceInfo.GetInterfaceConfig())
	slog.Debug("FetchBatchDataFromSourceWithStrategy - ParseConfig", "data", interfaceInfo.GetParseConfig())
	slog.Debug("FetchBatchDataFromSourceWithStrategy - TableFieldsConfig", "data", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		slog.Error("FetchBatchDataFromSourceWithStrategy - 创建查询构建器失败", "error", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSourceWithStrategy - 查询构建器创建成功")

	// 合并分页参数到基础参数中
	allParams := make(map[string]interface{})
	for k, v := range parameters {
		allParams[k] = v
	}
	for k, v := range pageParams {
		allParams[k] = v
	}

	// 根据同步策略和数据源类型构建不同的请求
	var executeRequest *datasource.ExecuteRequest
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 根据策略构建请求", "sync_strategy", syncStrategy, "datasource_category", dataSource.Category)

	switch syncStrategy {
	case "incremental":
		slog.Debug("FetchBatchDataFromSourceWithStrategy - 增量同步策略")

		// 增量同步：检查是否有增量参数
		if incrementalField, exists := allParams["incremental_field"]; exists {
			slog.Debug("FetchBatchDataFromSourceWithStrategy - 找到增量字段", "incremental_field", incrementalField)

			incrementalParams := &datasource.IncrementalParams{
				LastSyncValue:  allParams["last_sync_value"],
				IncrementalKey: cast.ToString(incrementalField),
				ComparisonType: cast.ToString(allParams["comparison_type"]),
				BatchSize:      cast.ToInt(allParams["batch_size"]),
			}

			if incrementalParams.ComparisonType == "" {
				incrementalParams.ComparisonType = "gt"
			}
			if incrementalParams.BatchSize <= 0 {
				incrementalParams.BatchSize = cast.ToInt(pageParams["page_size"])
				if incrementalParams.BatchSize <= 0 {
					incrementalParams.BatchSize = 1000
				}
			}

			slog.Debug("FetchBatchDataFromSourceWithStrategy - 增量参数",
				"last_sync_value", incrementalParams.LastSyncValue,
				"incremental_key", incrementalParams.IncrementalKey,
				"comparison_type", incrementalParams.ComparisonType,
				"batch_size", incrementalParams.BatchSize)

			executeRequest, err = queryBuilder.BuildIncrementalRequest("sync", incrementalParams)
		} else {
			slog.Debug("FetchBatchDataFromSourceWithStrategy - 没有增量参数，退化为全量分页同步")
			// 没有增量参数，使用普通分页同步请求
			executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
		}

	case "full":
		slog.Debug("FetchBatchDataFromSourceWithStrategy - 全量同步策略")

		// 全量同步：根据数据源类型构建请求
		switch dataSource.Category {
		case meta.DataSourceCategoryDatabase:
			slog.Debug("FetchBatchDataFromSourceWithStrategy - 数据库全量同步，使用分页")
			// 数据库类型：使用带分页的同步请求
			executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)

		case meta.DataSourceCategoryAPI:
			// API类型：检查是否有分页配置
			interfaceConfig := interfaceInfo.GetInterfaceConfig()
			if paginationEnabled := cast.ToBool(interfaceConfig[meta.DataInterfaceConfigFieldPaginationEnabled]); paginationEnabled {
				slog.Debug("FetchBatchDataFromSourceWithStrategy - API全量同步，使用分页")
				// 使用带分页的API同步请求
				executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
			} else {
				slog.Debug("FetchBatchDataFromSourceWithStrategy - API全量同步，不使用分页")
				// 没有分页配置，使用普通同步请求
				executeRequest, err = queryBuilder.BuildSyncRequest("full", allParams)
			}

		default:
			slog.Debug("FetchBatchDataFromSourceWithStrategy - 其他类型，使用测试请求")
			// 其他类型：使用普通测试请求
			executeRequest, err = queryBuilder.BuildTestRequest(allParams)
		}

	default:
		slog.Debug("FetchBatchDataFromSourceWithStrategy - 默认策略，使用全量同步")
		// 默认使用全量同步
		executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
	}

	if err != nil {
		slog.Error("FetchBatchDataFromSourceWithStrategy - 构建查询请求失败", "error", err)
		return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSourceWithStrategy - 执行请求", "data", executeRequest)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		slog.Error("FetchBatchDataFromSourceWithStrategy - 执行接口查询失败", "error", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	slog.Debug("FetchBatchDataFromSourceWithStrategy - 查询执行成功，响应", "data", response)

	// 检查响应是否成功
	if !response.Success {
		errorMsg := response.Message
		if errorMsg == "" {
			errorMsg = "接口调用失败"
		}
		// 如果有错误详情，添加到错误消息中
		if response.Error != "" {
			errorMsg = fmt.Sprintf("%s: %s", errorMsg, response.Error)
		}
		slog.Error("FetchBatchDataFromSourceWithStrategy - 接口返回错误", "message", errorMsg)
		return nil, nil, nil, fmt.Errorf("接口调用失败: %s", errorMsg)
	}

	// 处理返回的数据
	data, dataTypes, warnings := dp.ProcessResponseData(response.Data)

	slog.Debug("FetchBatchDataFromSourceWithStrategy - 处理后的数据", "row_count", len(data))
	if len(data) > 0 {
		slog.Debug("FetchBatchDataFromSourceWithStrategy - 第一行数据示例", "data", data[0])
	}
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 数据类型", "types", dataTypes)
	slog.Debug("FetchBatchDataFromSourceWithStrategy - 警告信息", "warnings", warnings)

	return data, dataTypes, warnings, nil
}

// ProcessResponseData 处理响应数据
func (dp *DataProcessor) ProcessResponseData(result interface{}) ([]map[string]interface{}, map[string]string, []string) {
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
	dataTypes := dp.AnalyzeDataTypes(sampleData)

	// 生成警告
	if len(sampleData) > 1000 {
		warnings = append(warnings, "数据量较大，建议分页查询")
	}
	if len(sampleData) == 0 {
		warnings = append(warnings, "接口返回空数据")
	}

	return sampleData, dataTypes, warnings
}

// AnalyzeDataTypes 分析数据类型
func (dp *DataProcessor) AnalyzeDataTypes(data []map[string]interface{}) map[string]string {
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
				if dp.isDateTime(str) {
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

// InferDataTypes 推断数据类型
func (dp *DataProcessor) InferDataTypes(data []map[string]interface{}) map[string]string {
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

// isDateTime 检测字符串是否为日期时间格式
func (dp *DataProcessor) isDateTime(str string) bool {
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
