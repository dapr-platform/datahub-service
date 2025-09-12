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
	fmt.Printf("[DEBUG] DataProcessor.FetchDataFromSource - 开始从数据源获取数据\n")
	fmt.Printf("[DEBUG] FetchDataFromSource - 接口ID: %s\n", interfaceInfo.GetID())
	fmt.Printf("[DEBUG] FetchDataFromSource - 数据源ID: %s\n", interfaceInfo.GetDataSourceID())
	fmt.Printf("[DEBUG] FetchDataFromSource - 请求参数: %+v\n", parameters)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := dp.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		fmt.Printf("[ERROR] FetchDataFromSource - 获取数据源信息失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchDataFromSource - 数据源信息: %+v\n", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = dp.executor.datasourceManager.Get(dataSource.ID)
	if err != nil {
		fmt.Printf("[DEBUG] FetchDataFromSource - 数据源未注册，创建临时实例，错误: %v\n", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = dp.executor.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			fmt.Printf("[ERROR] FetchDataFromSource - 创建数据源实例失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			fmt.Printf("[ERROR] FetchDataFromSource - 初始化数据源失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}
		fmt.Printf("[DEBUG] FetchDataFromSource - 临时数据源实例创建并初始化成功\n")
	} else {
		fmt.Printf("[DEBUG] FetchDataFromSource - 使用已注册的数据源实例\n")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	fmt.Printf("[DEBUG] FetchDataFromSource - 接口配置信息:\n")
	fmt.Printf("[DEBUG] FetchDataFromSource - InterfaceConfig: %+v\n", interfaceInfo.GetInterfaceConfig())
	fmt.Printf("[DEBUG] FetchDataFromSource - ParseConfig: %+v\n", interfaceInfo.GetParseConfig())
	fmt.Printf("[DEBUG] FetchDataFromSource - TableFieldsConfig: %+v\n", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		fmt.Printf("[ERROR] FetchDataFromSource - 创建查询构建器失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchDataFromSource - 查询构建器创建成功\n")

	// 根据执行类型构建不同的请求
	var executeRequest *datasource.ExecuteRequest
	switch executeType {
	case "test", "preview":
		executeRequest, err = queryBuilder.BuildTestRequest(parameters)
	case "sync":
		executeRequest, err = queryBuilder.BuildSyncRequest("full", parameters)
	case "incremental_sync":
		executeRequest, err = queryBuilder.BuildSyncRequest("incremental", parameters)
	default:
		executeRequest, err = queryBuilder.BuildTestRequest(parameters)
	}

	if err != nil {
		fmt.Printf("[ERROR] FetchDataFromSource - 构建查询请求失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchDataFromSource - 执行请求: %+v\n", executeRequest)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		fmt.Printf("[ERROR] FetchDataFromSource - 执行接口查询失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchDataFromSource - 查询执行成功，响应: %+v\n", response)

	// 处理返回的数据
	data, dataTypes, warnings := dp.ProcessResponseData(response.Data)

	fmt.Printf("[DEBUG] FetchDataFromSource - 处理后的数据: %d 行\n", len(data))
	if len(data) > 0 {
		fmt.Printf("[DEBUG] FetchDataFromSource - 第一行数据示例: %+v\n", data[0])
	}
	fmt.Printf("[DEBUG] FetchDataFromSource - 数据类型: %+v\n", dataTypes)
	fmt.Printf("[DEBUG] FetchDataFromSource - 警告信息: %+v\n", warnings)

	return data, dataTypes, warnings, nil
}

// FetchBatchDataFromSource 从数据源获取批量数据（支持分页）
func (dp *DataProcessor) FetchBatchDataFromSource(ctx context.Context, interfaceInfo InterfaceInfo, parameters map[string]interface{}, pageParams map[string]interface{}) ([]map[string]interface{}, map[string]string, []string, error) {
	fmt.Printf("[DEBUG] DataProcessor.FetchBatchDataFromSource - 开始获取批量数据\n")
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 接口ID: %s\n", interfaceInfo.GetID())
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 数据源ID: %s\n", interfaceInfo.GetDataSourceID())
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 请求参数: %+v\n", parameters)
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 分页参数: %+v\n", pageParams)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := dp.executor.db.First(&dataSource, "id = ?", interfaceInfo.GetDataSourceID()).Error; err != nil {
		fmt.Printf("[ERROR] FetchBatchDataFromSource - 获取数据源信息失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 数据源信息: %+v\n", dataSource)

	// 获取或创建数据源实例
	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的实例
	dsInstance, err = dp.executor.datasourceManager.Get(dataSource.ID)
	if err != nil {
		fmt.Printf("[DEBUG] FetchBatchDataFromSource - 数据源未注册，创建临时实例，错误: %v\n", err)
		// 如果没有注册，创建临时实例
		dsInstance, err = dp.executor.datasourceManager.CreateInstance(dataSource.Type)
		if err != nil {
			fmt.Printf("[ERROR] FetchBatchDataFromSource - 创建数据源实例失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("创建数据源实例失败: %w", err)
		}

		// 初始化数据源
		if err := dsInstance.Init(ctx, &dataSource); err != nil {
			fmt.Printf("[ERROR] FetchBatchDataFromSource - 初始化数据源失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("初始化数据源失败: %w", err)
		}
		fmt.Printf("[DEBUG] FetchBatchDataFromSource - 临时数据源实例创建并初始化成功\n")
	} else {
		fmt.Printf("[DEBUG] FetchBatchDataFromSource - 使用已注册的数据源实例\n")
	}

	// 创建查询构建器
	// 将[]interface{}转换为JSONB
	tableFieldsConfig := make(models.JSONB)
	for i, v := range interfaceInfo.GetTableFieldsConfig() {
		tableFieldsConfig[fmt.Sprintf("%d", i)] = v
	}

	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 接口配置信息:\n")
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - InterfaceConfig: %+v\n", interfaceInfo.GetInterfaceConfig())
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - ParseConfig: %+v\n", interfaceInfo.GetParseConfig())
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - TableFieldsConfig: %+v\n", tableFieldsConfig)

	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, &models.DataInterface{
		ID:                interfaceInfo.GetID(),
		InterfaceConfig:   interfaceInfo.GetInterfaceConfig(),
		ParseConfig:       interfaceInfo.GetParseConfig(),
		TableFieldsConfig: tableFieldsConfig,
	})
	if err != nil {
		fmt.Printf("[ERROR] FetchBatchDataFromSource - 创建查询构建器失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 查询构建器创建成功\n")

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
			fmt.Printf("[ERROR] FetchBatchDataFromSource - 构建数据库分页同步请求失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("构建数据库分页同步请求失败: %w", err)
		}

	case meta.DataSourceCategoryAPI:
		// API类型：检查是否有分页配置
		interfaceConfig := interfaceInfo.GetInterfaceConfig()
		if paginationEnabled := cast.ToBool(interfaceConfig[meta.DataInterfaceConfigFieldPaginationEnabled]); paginationEnabled {
			// 使用带分页的API同步请求
			executeRequest, err = queryBuilder.BuildSyncRequestWithPagination("full", allParams, pageParams)
			if err != nil {
				fmt.Printf("[ERROR] FetchBatchDataFromSource - 构建API分页同步请求失败: %v\n", err)
				return nil, nil, nil, fmt.Errorf("构建API分页同步请求失败: %w", err)
			}
		} else {
			// 没有分页配置，使用普通同步请求
			executeRequest, err = queryBuilder.BuildSyncRequest("full", allParams)
			if err != nil {
				fmt.Printf("[ERROR] FetchBatchDataFromSource - 构建API同步请求失败: %v\n", err)
				return nil, nil, nil, fmt.Errorf("构建API同步请求失败: %w", err)
			}
		}

	default:
		// 其他类型：使用普通测试请求
		executeRequest, err = queryBuilder.BuildTestRequest(parameters)
		if err != nil {
			fmt.Printf("[ERROR] FetchBatchDataFromSource - 构建查询请求失败: %v\n", err)
			return nil, nil, nil, fmt.Errorf("构建查询请求失败: %w", err)
		}
	}

	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 执行请求: %+v\n", executeRequest)

	// 执行数据查询
	response, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		fmt.Printf("[ERROR] FetchBatchDataFromSource - 执行接口查询失败: %v\n", err)
		return nil, nil, nil, fmt.Errorf("执行接口查询失败: %w", err)
	}

	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 查询执行成功，响应: %+v\n", response)

	// 处理返回的数据
	data, dataTypes, warnings := dp.ProcessResponseData(response.Data)

	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 处理后的数据: %d 行\n", len(data))
	if len(data) > 0 {
		fmt.Printf("[DEBUG] FetchBatchDataFromSource - 第一行数据示例: %+v\n", data[0])
	}
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 数据类型: %+v\n", dataTypes)
	fmt.Printf("[DEBUG] FetchBatchDataFromSource - 警告信息: %+v\n", warnings)

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
