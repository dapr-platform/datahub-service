/*
 * @module service/datasource/query_builder
 * @description 数据源查询构建器，根据数据源类型和接口配置构建执行请求
 * @architecture 构建器模式 - 根据元数据驱动生成查询请求
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 元数据解析 -> 配置验证 -> 请求构建 -> 参数注入
 * @rules 支持所有数据源类型的统一查询构建，遵循元数据定义
 * @dependencies datahub-service/service/models, datahub-service/service/meta
 * @refs interface.go, base.go
 */

package datasource

import (
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// QueryBuilder 查询构建器
type QueryBuilder struct {
	dataSource    *models.DataSource
	dataInterface *models.DataInterface
	sourceTypeDef *meta.DataSourceTypeDefinition
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	Total       int64 `json:"total"`
	HasNext     bool  `json:"has_next"`
}

// IncrementalParams 增量同步参数
type IncrementalParams struct {
	LastSyncTime   interface{} `json:"last_sync_time"`
	IncrementalKey string      `json:"incremental_key"`
	ComparisonType string      `json:"comparison_type"` // gt, gte, eq
	BatchSize      int         `json:"batch_size"`
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(dataSource *models.DataSource, dataInterface *models.DataInterface) (*QueryBuilder, error) {
	// 获取数据源类型定义
	sourceTypeDef, exists := meta.DataSourceTypes[dataSource.Type]
	if !exists {
		return nil, fmt.Errorf("不支持的数据源类型: %s", dataSource.Type)
	}

	return &QueryBuilder{
		dataSource:    dataSource,
		dataInterface: dataInterface,
		sourceTypeDef: sourceTypeDef,
	}, nil
}

// BuildTestRequest 构建测试请求
func (qb *QueryBuilder) BuildTestRequest(parameters map[string]interface{}) (*ExecuteRequest, error) {
	switch qb.sourceTypeDef.Category {
	case meta.DataSourceCategoryDatabase:
		return qb.buildDatabaseTestRequest(parameters)
	case meta.DataSourceCategoryAPI:
		return qb.buildAPITestRequest(parameters)
	case meta.DataSourceCategoryMessaging:
		return qb.buildMessagingTestRequest(parameters)
	default:
		return nil, fmt.Errorf("不支持的数据源类别: %s", qb.sourceTypeDef.Category)
	}
}

// BuildSyncRequest 构建同步请求
func (qb *QueryBuilder) BuildSyncRequest(syncType string, parameters map[string]interface{}) (*ExecuteRequest, error) {
	switch qb.sourceTypeDef.Category {
	case meta.DataSourceCategoryDatabase:
		return qb.buildDatabaseSyncRequest(syncType, parameters)
	case meta.DataSourceCategoryAPI:
		return qb.buildAPISyncRequest(syncType, parameters)
	case meta.DataSourceCategoryMessaging:
		return qb.buildMessagingSyncRequest(syncType, parameters)
	default:
		return nil, fmt.Errorf("不支持的数据源类别: %s", qb.sourceTypeDef.Category)
	}
}

// BuildSyncRequestWithPagination 构建带分页的同步请求
func (qb *QueryBuilder) BuildSyncRequestWithPagination(syncType string, parameters map[string]interface{}, pageParams map[string]interface{}) (*ExecuteRequest, error) {
	// 合并分页参数到基础参数中
	allParams := make(map[string]interface{})
	for k, v := range parameters {
		allParams[k] = v
	}
	for k, v := range pageParams {
		allParams[k] = v
	}

	switch qb.sourceTypeDef.Category {
	case meta.DataSourceCategoryDatabase:
		return qb.buildDatabaseSyncRequestWithPagination(syncType, allParams, pageParams)
	case meta.DataSourceCategoryAPI:
		return qb.buildAPISyncRequestWithPagination(syncType, allParams, pageParams)
	case meta.DataSourceCategoryMessaging:
		return qb.buildMessagingSyncRequest(syncType, allParams)
	default:
		return nil, fmt.Errorf("不支持的数据源类别: %s", qb.sourceTypeDef.Category)
	}
}

// BuildPaginatedRequest 构建分页请求
func (qb *QueryBuilder) BuildPaginatedRequest(syncType string, pageParams map[string]interface{}) (*ExecuteRequest, error) {
	// 使用现有的分页构建方法
	return qb.BuildSyncRequestWithPagination(syncType, make(map[string]interface{}), pageParams)
}

// BuildIncrementalRequest 构建增量查询请求
func (qb *QueryBuilder) BuildIncrementalRequest(syncType string, incrementalParams *IncrementalParams) (*ExecuteRequest, error) {
	parameters := make(map[string]interface{})

	// 添加增量参数
	if incrementalParams != nil {
		parameters["last_sync_time"] = incrementalParams.LastSyncTime
		parameters["incremental_key"] = incrementalParams.IncrementalKey
		parameters["comparison_type"] = incrementalParams.ComparisonType
		parameters["batch_size"] = incrementalParams.BatchSize
	}

	switch qb.sourceTypeDef.Category {
	case meta.DataSourceCategoryDatabase:
		return qb.buildDatabaseIncrementalRequest(syncType, parameters, incrementalParams)
	case meta.DataSourceCategoryAPI:
		return qb.buildAPIIncrementalRequest(syncType, parameters, incrementalParams)
	case meta.DataSourceCategoryMessaging:
		return qb.buildMessagingSyncRequest(syncType, parameters)
	default:
		return nil, fmt.Errorf("不支持的数据源类别: %s", qb.sourceTypeDef.Category)
	}
}

// GetNextPageParams 获取下一页参数
func (qb *QueryBuilder) GetNextPageParams(currentPage int, pageSize int) map[string]interface{} {
	return map[string]interface{}{
		"page":      currentPage + 1,
		"page_size": pageSize,
		"offset":    currentPage * pageSize,
		"limit":     pageSize,
	}
}

// ExtractPaginationInfo 从响应中提取分页信息
func (qb *QueryBuilder) ExtractPaginationInfo(response *ExecuteResponse) (*PaginationInfo, error) {
	if response == nil || response.Data == nil {
		return nil, fmt.Errorf("响应数据为空")
	}

	paginationInfo := &PaginationInfo{}

	// 尝试从响应数据中提取分页信息
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if metadata, exists := dataMap["metadata"]; exists {
			if metaMap, ok := metadata.(map[string]interface{}); ok {
				if page, exists := metaMap["page"]; exists {
					paginationInfo.CurrentPage = cast.ToInt(page)
				}
				if pageSize, exists := metaMap["page_size"]; exists {
					paginationInfo.PageSize = cast.ToInt(pageSize)
				}
				if total, exists := metaMap["total"]; exists {
					paginationInfo.Total = cast.ToInt64(total)
				}
				if hasNext, exists := metaMap["has_next"]; exists {
					paginationInfo.HasNext = cast.ToBool(hasNext)
				}
			}
		}
	}

	// 如果没有明确的has_next字段，根据数据量判断
	if paginationInfo.Total > 0 && paginationInfo.PageSize > 0 {
		totalPages := int((paginationInfo.Total + int64(paginationInfo.PageSize) - 1) / int64(paginationInfo.PageSize))
		paginationInfo.HasNext = paginationInfo.CurrentPage < totalPages
	}

	return paginationInfo, nil
}

// buildDatabaseTestRequest 构建数据库测试请求
func (qb *QueryBuilder) buildDatabaseTestRequest(parameters map[string]interface{}) (*ExecuteRequest, error) {
	var query string
	var operation string = "query"

	// 从接口配置中获取查询信息
	if qb.dataInterface != nil {
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 尝试获取自定义查询
		if q, exists := interfaceConfig["query"]; exists {
			if queryStr, ok := q.(string); ok {
				query = queryStr
			}
		}

		// 尝试获取表名
		if query == "" {
			if tableName, exists := interfaceConfig[meta.DataInterfaceConfigFieldTableName]; exists {
				if tableStr, ok := tableName.(string); ok {
					// 构建基本的SELECT查询
					query = fmt.Sprintf("SELECT * FROM %s LIMIT 5", tableStr)
				}
			}
		}
	}

	// 如果还没有查询，使用默认测试查询
	if query == "" {
		switch qb.dataSource.Type {
		case meta.DataSourceTypeDBPostgreSQL:
			query = "SELECT 1 as test_id, 'PostgreSQL测试数据' as test_name, CURRENT_TIMESTAMP as test_time"
		default:
			query = "SELECT 1 as test_id, '数据库测试数据' as test_name"
		}
	}

	request := &ExecuteRequest{
		Operation: operation,
		Query:     query,
		Params:    parameters,
		Timeout:   30 * time.Second,
	}

	return request, nil
}

// buildAPITestRequest 构建API测试请求
func (qb *QueryBuilder) buildAPITestRequest(parameters map[string]interface{}) (*ExecuteRequest, error) {
	return qb.buildAPIRequest(parameters, false)
}

// buildAPIRequest 构建API请求的通用方法
func (qb *QueryBuilder) buildAPIRequest(parameters map[string]interface{}, isSync bool) (*ExecuteRequest, error) {
	var method string = "GET"
	var headers map[string]interface{}
	var body interface{}
	var urlPattern string = "suffix"
	var urlSuffix string = "/"
	var queryParams map[string]interface{}
	var pathParams map[string]interface{}
	var dataPath string = "data"
	var paginationConfig map[string]interface{}

	// 从接口配置中获取API信息
	if qb.dataInterface != nil {
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 获取URL模式
		if pattern, exists := interfaceConfig[meta.DataInterfaceConfigFieldUrlPattern]; exists {
			urlPattern = cast.ToString(pattern)
		}

		// 获取URL后缀
		if suffix, exists := interfaceConfig[meta.DataInterfaceConfigFieldUrlSuffix]; exists {
			urlSuffix = cast.ToString(suffix)
		}

		// 获取请求方法
		if m, exists := interfaceConfig[meta.DataInterfaceConfigFieldMethod]; exists {
			method = cast.ToString(m)
		}

		// 获取请求头
		if h, exists := interfaceConfig[meta.DataInterfaceConfigFieldHeaders]; exists {
			if headerMap, ok := h.(map[string]interface{}); ok {
				headers = headerMap
			}
		}

		// 获取请求体
		if b, exists := interfaceConfig[meta.DataInterfaceConfigFieldBody]; exists {
			body = b
		}

		// 获取查询参数
		if qp, exists := interfaceConfig[meta.DataInterfaceConfigFieldQueryParams]; exists {
			if queryMap, ok := qp.(map[string]interface{}); ok {
				queryParams = queryMap
			}
		}

		// 获取路径参数
		if pp, exists := interfaceConfig[meta.DataInterfaceConfigFieldPathParams]; exists {
			if pathMap, ok := pp.(map[string]interface{}); ok {
				pathParams = pathMap
			}
		}

		// 获取数据路径
		if dp, exists := interfaceConfig[meta.DataInterfaceConfigFieldDataPath]; exists {
			dataPath = cast.ToString(dp)
		}

		// 获取分页配置
		paginationConfig = qb.GetPaginationConfig()
	}

	// 构建完整的URL和参数
	fullURL, finalQueryParams, err := qb.buildAPIURL(urlPattern, urlSuffix, queryParams, pathParams, parameters, paginationConfig, isSync)
	if err != nil {
		return nil, fmt.Errorf("构建API URL失败: %w", err)
	}

	// 准备请求数据
	requestData := map[string]interface{}{
		"method":      method,
		"headers":     headers,
		"data_path":   dataPath,
		"url_pattern": urlPattern,
	}

	if body != nil {
		requestData["body"] = body
	}

	if paginationConfig != nil {
		requestData["pagination"] = paginationConfig
	}

	// 添加响应解析配置
	if qb.dataInterface != nil {
		responseParserConfig := make(map[string]interface{})
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 复制所有响应解析相关的配置
		responseFields := []string{
			meta.DataInterfaceConfigFieldResponseType,
			meta.DataInterfaceConfigFieldResponseParser,
			meta.DataInterfaceConfigFieldSuccessCondition,
			meta.DataInterfaceConfigFieldStatusCodeSuccess,
			meta.DataInterfaceConfigFieldSuccessField,
			meta.DataInterfaceConfigFieldSuccessValue,
			meta.DataInterfaceConfigFieldErrorField,
			meta.DataInterfaceConfigFieldErrorMessageField,
			meta.DataInterfaceConfigFieldDataPath,
			meta.DataInterfaceConfigFieldTotalField,
			meta.DataInterfaceConfigFieldPageField,
			meta.DataInterfaceConfigFieldPageSizeField,
		}

		for _, field := range responseFields {
			if value, exists := interfaceConfig[field]; exists {
				responseParserConfig[field] = value
			}
		}

		requestData["response_parser"] = responseParserConfig
	}

	timeout := 30 * time.Second
	if isSync {
		timeout = 5 * time.Minute
	}

	request := &ExecuteRequest{
		Operation: "api_call",
		Query:     fullURL,
		Params:    finalQueryParams,
		Timeout:   timeout,
		Data:      requestData,
	}

	return request, nil
}

// buildAPIURL 构建API URL
func (qb *QueryBuilder) buildAPIURL(urlPattern, urlSuffix string, queryParams, pathParams, parameters map[string]interface{}, paginationConfig map[string]interface{}, isSync bool) (string, map[string]interface{}, error) {
	var finalURL string
	finalQueryParams := make(map[string]interface{})

	switch urlPattern {
	case "suffix":
		// 基础URL + URL后缀模式: http://ip:port/api/service + /device-info
		finalURL = urlSuffix
		if finalURL == "" {
			finalURL = "/"
		}

		// 合并查询参数
		for k, v := range queryParams {
			finalQueryParams[k] = qb.resolveParameterValue(v, parameters)
		}
		for k, v := range parameters {
			if !qb.isReservedParam(k) {
				finalQueryParams[k] = v
			}
		}

	case "query":
		// 基础URL + 查询参数模式: http://ip:port/api/service?type=device
		finalURL = "/"
		if urlSuffix != "" && urlSuffix != "/" {
			finalURL = urlSuffix
		}

		// 合并所有查询参数
		for k, v := range queryParams {
			finalQueryParams[k] = qb.resolveParameterValue(v, parameters)
		}
		for k, v := range parameters {
			if !qb.isReservedParam(k) {
				finalQueryParams[k] = v
			}
		}

	case "path":
		// 路径参数模式: http://ip:port/api/service/device/{id}
		finalURL = urlSuffix
		if finalURL == "" {
			finalURL = "/"
		}

		// 替换路径参数
		for key, value := range pathParams {
			placeholder := fmt.Sprintf("{%s}", key)
			resolvedValue := qb.resolveParameterValue(value, parameters)
			finalURL = strings.ReplaceAll(finalURL, placeholder, fmt.Sprintf("%v", resolvedValue))
		}

		// 查询参数
		for k, v := range queryParams {
			finalQueryParams[k] = qb.resolveParameterValue(v, parameters)
		}

	case "combined":
		// 组合模式: 支持URL后缀 + 路径参数 + 查询参数
		finalURL = urlSuffix
		if finalURL == "" {
			finalURL = "/"
		}

		// 替换路径参数
		for key, value := range pathParams {
			placeholder := fmt.Sprintf("{%s}", key)
			resolvedValue := qb.resolveParameterValue(value, parameters)
			finalURL = strings.ReplaceAll(finalURL, placeholder, fmt.Sprintf("%v", resolvedValue))
		}

		// 合并查询参数
		for k, v := range queryParams {
			finalQueryParams[k] = qb.resolveParameterValue(v, parameters)
		}
		for k, v := range parameters {
			if !qb.isReservedParam(k) {
				finalQueryParams[k] = v
			}
		}

	default:
		return "", nil, fmt.Errorf("不支持的URL模式: %s", urlPattern)
	}

	// 处理分页参数 - 只有在同步且分页配置启用时才添加
	if isSync && paginationConfig != nil {
		enabled := cast.ToBool(paginationConfig["enabled"])
		fmt.Printf("[DEBUG] QueryBuilder.buildAPIURL - 分页配置: enabled=%t, config=%+v\n", enabled, paginationConfig)

		if enabled {
			pageParam := cast.ToString(paginationConfig["page_param"])
			if pageParam != "" {
				if _, exists := finalQueryParams[pageParam]; !exists {
					pageStart := cast.ToInt(paginationConfig["page_start"])
					if pageStart <= 0 {
						pageStart = 1
					}
					finalQueryParams[pageParam] = pageStart
					fmt.Printf("[DEBUG] QueryBuilder.buildAPIURL - 添加分页参数: %s=%d\n", pageParam, pageStart)
				}
			}

			sizeParam := cast.ToString(paginationConfig["size_param"])
			if sizeParam != "" {
				if _, exists := finalQueryParams[sizeParam]; !exists {
					pageSize := cast.ToInt(paginationConfig["page_size"])
					if pageSize <= 0 {
						pageSize = 20
					}
					finalQueryParams[sizeParam] = pageSize
					fmt.Printf("[DEBUG] QueryBuilder.buildAPIURL - 添加页大小参数: %s=%d\n", sizeParam, pageSize)
				}
			}
		} else {
			fmt.Printf("[DEBUG] QueryBuilder.buildAPIURL - 分页配置未启用，跳过分页参数\n")
		}
	}

	return finalURL, finalQueryParams, nil
}

// resolveParameterValue 解析参数值，支持变量替换
func (qb *QueryBuilder) resolveParameterValue(value interface{}, parameters map[string]interface{}) interface{} {
	if strValue, ok := value.(string); ok {
		// 如果是字符串，检查是否是变量引用 ${variable_name}
		if strings.HasPrefix(strValue, "${") && strings.HasSuffix(strValue, "}") {
			varName := strValue[2 : len(strValue)-1]
			if paramValue, exists := parameters[varName]; exists {
				return paramValue
			}
		}
	}
	return value
}

// isReservedParam 检查是否是保留参数
func (qb *QueryBuilder) isReservedParam(key string) bool {
	reservedParams := []string{
		"sync_type", "batch_size", "last_sync_time", "time_field",
		"page", "size", "limit", "offset",
	}
	for _, reserved := range reservedParams {
		if key == reserved {
			return true
		}
	}
	return false
}

// buildMessagingTestRequest 构建消息测试请求
func (qb *QueryBuilder) buildMessagingTestRequest(parameters map[string]interface{}) (*ExecuteRequest, error) {
	// 消息数据源通常用于连接测试
	request := &ExecuteRequest{
		Operation: "connect_test",
		Query:     "",
		Params:    parameters,
		Timeout:   30 * time.Second,
	}

	// 添加消息配置信息
	if qb.dataInterface != nil {
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)
		request.Data = interfaceConfig
	}

	return request, nil
}

// buildDatabaseSyncRequest 构建数据库同步请求
func (qb *QueryBuilder) buildDatabaseSyncRequest(syncType string, parameters map[string]interface{}) (*ExecuteRequest, error) {
	var query string
	var operation string = "query"

	// 从接口配置中获取查询信息
	if qb.dataInterface != nil {
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 尝试获取自定义查询
		if q, exists := interfaceConfig["query"]; exists {
			if queryStr, ok := q.(string); ok {
				query = queryStr
			}
		}

		// 尝试获取表名并构建查询
		if query == "" {
			if tableName, exists := interfaceConfig[meta.DataInterfaceConfigFieldTableName]; exists {
				if tableStr, ok := tableName.(string); ok {
					switch syncType {
					case "full":
						query = fmt.Sprintf("SELECT * FROM %s", tableStr)
					case "incremental":
						// 增量同步需要时间字段
						timeField := "updated_at" // 默认时间字段
						if tf, exists := parameters["time_field"]; exists {
							if tfStr, ok := tf.(string); ok {
								timeField = tfStr
							}
						}
						if lastSyncTime, exists := parameters["last_sync_time"]; exists {
							query = fmt.Sprintf("SELECT * FROM %s WHERE %s > '%v'", tableStr, timeField, lastSyncTime)
						} else {
							query = fmt.Sprintf("SELECT * FROM %s", tableStr)
						}
					}
				}
			}
		}
	}

	if query == "" {
		return nil, fmt.Errorf("无法构建数据库同步查询")
	}

	// 合并参数
	allParams := make(map[string]interface{})
	allParams["sync_type"] = syncType
	for k, v := range parameters {
		allParams[k] = v
	}

	request := &ExecuteRequest{
		Operation: operation,
		Query:     query,
		Params:    allParams,
		Timeout:   5 * time.Minute, // 同步操作使用更长的超时时间
	}

	return request, nil
}

// buildDatabaseSyncRequestWithPagination 构建带分页的数据库同步请求
func (qb *QueryBuilder) buildDatabaseSyncRequestWithPagination(syncType string, parameters map[string]interface{}, pageParams map[string]interface{}) (*ExecuteRequest, error) {
	var query string
	var operation string = "query"

	// 从接口配置中获取查询信息
	if qb.dataInterface != nil {
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 尝试获取自定义查询
		if q, exists := interfaceConfig["query"]; exists {
			if queryStr, ok := q.(string); ok {
				query = queryStr
			}
		}

		// 尝试获取表名并构建查询
		if query == "" {
			if tableName, exists := interfaceConfig[meta.DataInterfaceConfigFieldTableName]; exists {
				if tableStr, ok := tableName.(string); ok {
					switch syncType {
					case "full":
						query = fmt.Sprintf("SELECT * FROM %s", tableStr)
					case "incremental":
						// 增量同步需要时间字段
						timeField := "updated_at" // 默认时间字段
						if tf, exists := parameters["time_field"]; exists {
							if tfStr, ok := tf.(string); ok {
								timeField = tfStr
							}
						}
						if lastSyncTime, exists := parameters["last_sync_time"]; exists {
							query = fmt.Sprintf("SELECT * FROM %s WHERE %s > '%v'", tableStr, timeField, lastSyncTime)
						} else {
							query = fmt.Sprintf("SELECT * FROM %s", tableStr)
						}
					}
				}
			}
		}
	}

	if query == "" {
		return nil, fmt.Errorf("无法构建数据库同步查询")
	}

	// 添加分页信息到查询中
	if page, exists := pageParams["page"]; exists {
		if pageSize, exists := pageParams["page_size"]; exists {
			pageInt := cast.ToInt(page)
			pageSizeInt := cast.ToInt(pageSize)
			if pageInt > 0 && pageSizeInt > 0 {
				offset := (pageInt - 1) * pageSizeInt
				query = fmt.Sprintf("%s LIMIT %d OFFSET %d", query, pageSizeInt, offset)
				fmt.Printf("[DEBUG] QueryBuilder.buildDatabaseSyncRequestWithPagination - 添加分页: LIMIT %d OFFSET %d\n", pageSizeInt, offset)
			}
		}
	}

	// 合并参数
	allParams := make(map[string]interface{})
	allParams["sync_type"] = syncType
	for k, v := range parameters {
		allParams[k] = v
	}

	request := &ExecuteRequest{
		Operation: operation,
		Query:     query,
		Params:    allParams,
		Timeout:   5 * time.Minute, // 同步操作使用更长的超时时间
	}

	return request, nil
}

// buildAPISyncRequest 构建API同步请求
func (qb *QueryBuilder) buildAPISyncRequest(syncType string, parameters map[string]interface{}) (*ExecuteRequest, error) {
	// 添加同步类型到参数中
	syncParams := make(map[string]interface{})
	for k, v := range parameters {
		syncParams[k] = v
	}
	syncParams["sync_type"] = syncType

	// 使用API请求构建器，标记为同步请求
	return qb.buildAPIRequest(syncParams, true)
}

// buildAPISyncRequestWithPagination 构建带分页的API同步请求
func (qb *QueryBuilder) buildAPISyncRequestWithPagination(syncType string, parameters map[string]interface{}, pageParams map[string]interface{}) (*ExecuteRequest, error) {
	// 添加同步类型到参数中
	syncParams := make(map[string]interface{})
	for k, v := range parameters {
		syncParams[k] = v
	}
	syncParams["sync_type"] = syncType

	fmt.Printf("[DEBUG] QueryBuilder.buildAPISyncRequestWithPagination - 构建API分页请求\n")
	fmt.Printf("[DEBUG] QueryBuilder.buildAPISyncRequestWithPagination - 分页参数: %+v\n", pageParams)

	// 使用API请求构建器，标记为同步请求，并传递分页参数
	return qb.buildAPIRequestWithPagination(syncParams, true, pageParams)
}

// buildMessagingSyncRequest 构建消息同步请求
func (qb *QueryBuilder) buildMessagingSyncRequest(syncType string, parameters map[string]interface{}) (*ExecuteRequest, error) {
	// 复用消息测试请求的构建逻辑
	request, err := qb.buildMessagingTestRequest(parameters)
	if err != nil {
		return nil, err
	}

	// 添加同步类型信息
	request.Params["sync_type"] = syncType
	request.Operation = "message_sync"
	request.Timeout = 10 * time.Minute

	return request, nil
}

// ValidateInterfaceConfig 验证接口配置
func (qb *QueryBuilder) ValidateInterfaceConfig() error {
	if qb.dataInterface == nil {
		return nil // 没有接口配置时不需要验证
	}

	// 获取接口配置定义
	interfaceConfigDef, exists := meta.DataInterfaceConfigDefinitions[qb.sourceTypeDef.Category]
	if !exists {
		return fmt.Errorf("找不到数据源类别 %s 的接口配置定义", qb.sourceTypeDef.Category)
	}

	interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

	// 验证必填字段
	for _, field := range interfaceConfigDef.MetaConfig {
		if field.Required {
			if _, exists := interfaceConfig[field.Name]; !exists {
				return fmt.Errorf("缺少必填字段: %s", field.DisplayName)
			}
		}
	}

	return nil
}

// buildAPIRequestWithPagination 构建带分页的API请求
func (qb *QueryBuilder) buildAPIRequestWithPagination(parameters map[string]interface{}, isSync bool, pageParams map[string]interface{}) (*ExecuteRequest, error) {
	var method string = "GET"
	var headers map[string]interface{}
	var body interface{}
	var urlPattern string = "suffix"
	var urlSuffix string = "/"
	var queryParams map[string]interface{}
	var pathParams map[string]interface{}
	var dataPath string = "data"
	var paginationConfig map[string]interface{}

	// 从接口配置中获取API信息
	if qb.dataInterface != nil {
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 获取URL模式
		if pattern, exists := interfaceConfig[meta.DataInterfaceConfigFieldUrlPattern]; exists {
			if patternStr, ok := pattern.(string); ok {
				urlPattern = patternStr
			}
		}

		// 获取URL后缀
		if suffix, exists := interfaceConfig[meta.DataInterfaceConfigFieldUrlSuffix]; exists {
			if suffixStr, ok := suffix.(string); ok {
				urlSuffix = suffixStr
			}
		}

		// 获取请求方法
		if m, exists := interfaceConfig[meta.DataInterfaceConfigFieldMethod]; exists {
			if methodStr, ok := m.(string); ok {
				method = methodStr
			}
		}

		// 获取请求头
		if h, exists := interfaceConfig[meta.DataInterfaceConfigFieldHeaders]; exists {
			if headerMap, ok := h.(map[string]interface{}); ok {
				headers = headerMap
			}
		}

		// 获取请求体
		if b, exists := interfaceConfig[meta.DataInterfaceConfigFieldBody]; exists {
			body = b
		}

		// 获取查询参数
		if qp, exists := interfaceConfig[meta.DataInterfaceConfigFieldQueryParams]; exists {
			if queryMap, ok := qp.(map[string]interface{}); ok {
				queryParams = make(map[string]interface{})
				for k, v := range queryMap {
					queryParams[k] = v
				}
			}
		}

		// 获取路径参数
		if pp, exists := interfaceConfig[meta.DataInterfaceConfigFieldPathParams]; exists {
			if pathMap, ok := pp.(map[string]interface{}); ok {
				pathParams = pathMap
			}
		}

		// 获取数据路径
		if dp, exists := interfaceConfig[meta.DataInterfaceConfigFieldDataPath]; exists {
			if dpStr, ok := dp.(string); ok && dpStr != "" {
				dataPath = dpStr
			}
		}

		// 获取分页配置
		paginationConfig = qb.GetPaginationConfig()
	}

	// 如果没有查询参数，初始化为空map
	if queryParams == nil {
		queryParams = make(map[string]interface{})
	}

	// 根据分页配置添加分页参数
	if paginationConfig != nil && isSync {
		enabled := cast.ToBool(paginationConfig["enabled"])
		fmt.Printf("[DEBUG] QueryBuilder.buildAPIRequestWithPagination - 检查分页配置: enabled=%t\n", enabled)

		if enabled {
			// 使用配置中的分页参数名
			pageParam := cast.ToString(paginationConfig["page_param"])
			if pageParam != "" {
				if page, exists := pageParams["page"]; exists {
					queryParams[pageParam] = page
					fmt.Printf("[DEBUG] QueryBuilder.buildAPIRequestWithPagination - 添加分页参数: %s=%v\n", pageParam, page)
				}
			}

			sizeParam := cast.ToString(paginationConfig["size_param"])
			if sizeParam != "" {
				if pageSize, exists := pageParams["page_size"]; exists {
					queryParams[sizeParam] = pageSize
					fmt.Printf("[DEBUG] QueryBuilder.buildAPIRequestWithPagination - 添加页大小参数: %s=%v\n", sizeParam, pageSize)
				}
			}
		} else {
			fmt.Printf("[DEBUG] QueryBuilder.buildAPIRequestWithPagination - 分页配置未启用，不添加分页参数\n")
		}
	} else if isSync {
		// 没有分页配置但是是同步请求时，使用默认参数名
		fmt.Printf("[DEBUG] QueryBuilder.buildAPIRequestWithPagination - 没有分页配置，使用默认参数名\n")
		if page, exists := pageParams["page"]; exists {
			queryParams["page"] = page
		}
		if pageSize, exists := pageParams["page_size"]; exists {
			queryParams["size"] = pageSize
		}
	}

	fmt.Printf("[DEBUG] QueryBuilder.buildAPIRequestWithPagination - 最终查询参数: %+v\n", queryParams)

	// 构建完整的URL和参数
	fullURL, finalQueryParams, err := qb.buildAPIURL(urlPattern, urlSuffix, queryParams, pathParams, parameters, paginationConfig, isSync)
	if err != nil {
		return nil, fmt.Errorf("构建API URL失败: %w", err)
	}

	// 准备请求数据
	requestData := map[string]interface{}{
		"method":      method,
		"headers":     headers,
		"data_path":   dataPath,
		"url_pattern": urlPattern,
	}

	if body != nil {
		requestData["body"] = body
	}

	if paginationConfig != nil {
		requestData["pagination"] = paginationConfig
	}

	// 添加响应解析配置
	if qb.dataInterface != nil {
		responseParserConfig := make(map[string]interface{})
		interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

		// 复制所有响应解析相关的配置
		responseFields := []string{
			meta.DataInterfaceConfigFieldResponseType,
			meta.DataInterfaceConfigFieldResponseParser,
			meta.DataInterfaceConfigFieldSuccessCondition,
			meta.DataInterfaceConfigFieldStatusCodeSuccess,
			meta.DataInterfaceConfigFieldSuccessField,
			meta.DataInterfaceConfigFieldSuccessValue,
			meta.DataInterfaceConfigFieldErrorField,
			meta.DataInterfaceConfigFieldErrorMessageField,
			meta.DataInterfaceConfigFieldDataPath,
			meta.DataInterfaceConfigFieldTotalField,
			meta.DataInterfaceConfigFieldPageField,
			meta.DataInterfaceConfigFieldPageSizeField,
		}

		for _, field := range responseFields {
			if value, exists := interfaceConfig[field]; exists {
				responseParserConfig[field] = value
			}
		}

		requestData["response_parser"] = responseParserConfig
	}

	timeout := 30 * time.Second
	if isSync {
		timeout = 5 * time.Minute
	}

	request := &ExecuteRequest{
		Operation: "api_call",
		Query:     fullURL,
		Params:    finalQueryParams,
		Timeout:   timeout,
		Data:      requestData,
	}

	return request, nil
}

// IsPaginationEnabled 检查是否启用了分页
func (qb *QueryBuilder) IsPaginationEnabled() bool {
	if qb.dataInterface == nil {
		return false
	}

	interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

	// 检查分页启用字段
	if enabled, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationEnabled]; exists {
		isEnabled := cast.ToBool(enabled)
		fmt.Printf("[DEBUG] QueryBuilder.IsPaginationEnabled - 分页配置: enabled=%t\n", isEnabled)
		return isEnabled
	}

	fmt.Printf("[DEBUG] QueryBuilder.IsPaginationEnabled - 不支持分页\n")
	return false
}

// GetPaginationConfig 获取分页配置
func (qb *QueryBuilder) GetPaginationConfig() map[string]interface{} {
	if qb.dataInterface == nil {
		return nil
	}

	interfaceConfig := map[string]interface{}(qb.dataInterface.InterfaceConfig)

	// 从独立字段构建分页配置
	paginationConfig := make(map[string]interface{})

	// 检查是否启用分页
	if enabled, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationEnabled]; exists {
		paginationConfig["enabled"] = cast.ToBool(enabled)
	} else {
		paginationConfig["enabled"] = false
	}

	// 获取页码参数名
	if pageParam, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationPageParam]; exists {
		paginationConfig["page_param"] = cast.ToString(pageParam)
	} else {
		paginationConfig["page_param"] = "page"
	}

	// 获取页大小参数名
	if sizeParam, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationSizeParam]; exists {
		paginationConfig["size_param"] = cast.ToString(sizeParam)
	} else {
		paginationConfig["size_param"] = "size"
	}

	// 获取起始页码
	if startValue, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationStartValue]; exists {
		paginationConfig["page_start"] = cast.ToInt(startValue)
	} else {
		paginationConfig["page_start"] = 1
	}

	// 获取默认页大小
	if defaultSize, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationDefaultSize]; exists {
		paginationConfig["page_size"] = cast.ToInt(defaultSize)
	} else {
		paginationConfig["page_size"] = 20
	}

	// 获取参数位置
	if paramLocation, exists := interfaceConfig[meta.DataInterfaceConfigFieldPaginationParamLocation]; exists {
		paginationConfig["param_location"] = cast.ToString(paramLocation)
	} else {
		paginationConfig["param_location"] = "query"
	}

	return paginationConfig
}

// BuildNextPageParams 构建下一页的参数
func (qb *QueryBuilder) BuildNextPageParams(currentPage int, pageSize int) map[string]interface{} {
	paginationConfig := qb.GetPaginationConfig()

	pageParams := make(map[string]interface{})

	// 使用配置中的参数名
	pageParam := cast.ToString(paginationConfig["page_param"])
	if pageParam == "" {
		pageParam = "page"
	}

	sizeParam := cast.ToString(paginationConfig["size_param"])
	if sizeParam == "" {
		sizeParam = "size"
	}

	pageParams[pageParam] = currentPage
	pageParams[sizeParam] = pageSize

	// 为了兼容性，也保留标准的参数名
	pageParams["page"] = currentPage
	pageParams["page_size"] = pageSize

	fmt.Printf("[DEBUG] QueryBuilder.BuildNextPageParams - 构建分页参数: %+v\n", pageParams)
	return pageParams
}

// buildDatabaseIncrementalRequest 构建数据库增量请求
func (qb *QueryBuilder) buildDatabaseIncrementalRequest(syncType string, parameters map[string]interface{}, incrementalParams *IncrementalParams) (*ExecuteRequest, error) {
	// 先构建基本的数据库同步请求
	baseRequest, err := qb.buildDatabaseSyncRequest(syncType, parameters)
	if err != nil {
		return nil, fmt.Errorf("构建基础数据库请求失败: %w", err)
	}

	// 修改查询以支持增量同步
	if incrementalParams != nil && incrementalParams.IncrementalKey != "" {
		originalQuery := baseRequest.Query
		comparisonOp := ">"

		switch incrementalParams.ComparisonType {
		case "gte":
			comparisonOp = ">="
		case "eq":
			comparisonOp = "="
		case "gt":
			comparisonOp = ">"
		}

		// 添加增量条件到WHERE子句
		if strings.Contains(strings.ToUpper(originalQuery), "WHERE") {
			baseRequest.Query = fmt.Sprintf("%s AND %s %s $last_sync_time",
				originalQuery, incrementalParams.IncrementalKey, comparisonOp)
		} else {
			baseRequest.Query = fmt.Sprintf("%s WHERE %s %s $last_sync_time",
				originalQuery, incrementalParams.IncrementalKey, comparisonOp)
		}

		// 添加排序以确保增量同步的一致性
		if !strings.Contains(strings.ToUpper(originalQuery), "ORDER BY") {
			baseRequest.Query = fmt.Sprintf("%s ORDER BY %s ASC", baseRequest.Query, incrementalParams.IncrementalKey)
		}

		// 添加批量限制
		if incrementalParams.BatchSize > 0 {
			baseRequest.Query = fmt.Sprintf("%s LIMIT %d", baseRequest.Query, incrementalParams.BatchSize)
		}

		// 添加增量参数到请求参数中
		if baseRequest.Params == nil {
			baseRequest.Params = make(map[string]interface{})
		}
		baseRequest.Params["last_sync_time"] = incrementalParams.LastSyncTime
	}

	// 标记为增量同步请求
	if baseRequest.Data == nil {
		baseRequest.Data = make(map[string]interface{})
	}
	if dataMap, ok := baseRequest.Data.(map[string]interface{}); ok {
		dataMap["sync_type"] = "incremental"
		dataMap["incremental_params"] = incrementalParams
	}

	return baseRequest, nil
}

// buildAPIIncrementalRequest 构建API增量请求
func (qb *QueryBuilder) buildAPIIncrementalRequest(syncType string, parameters map[string]interface{}, incrementalParams *IncrementalParams) (*ExecuteRequest, error) {
	// 先构建基本的API同步请求
	baseRequest, err := qb.buildAPISyncRequest(syncType, parameters)
	if err != nil {
		return nil, fmt.Errorf("构建基础API请求失败: %w", err)
	}

	// 添加增量参数到查询参数或请求体中
	if incrementalParams != nil {
		if baseRequest.Params == nil {
			baseRequest.Params = make(map[string]interface{})
		}

		// 添加增量同步相关参数
		if incrementalParams.LastSyncTime != nil {
			baseRequest.Params["last_sync_time"] = incrementalParams.LastSyncTime
			baseRequest.Params["since"] = incrementalParams.LastSyncTime
			baseRequest.Params["updated_after"] = incrementalParams.LastSyncTime
		}

		if incrementalParams.IncrementalKey != "" {
			baseRequest.Params["incremental_key"] = incrementalParams.IncrementalKey
		}

		if incrementalParams.BatchSize > 0 {
			baseRequest.Params["limit"] = incrementalParams.BatchSize
			baseRequest.Params["page_size"] = incrementalParams.BatchSize
		}

		// 添加排序参数确保增量同步的一致性
		if incrementalParams.IncrementalKey != "" {
			baseRequest.Params["sort"] = incrementalParams.IncrementalKey
			baseRequest.Params["order"] = "asc"
		}
	}

	// 标记为增量同步请求
	if baseRequest.Data == nil {
		baseRequest.Data = make(map[string]interface{})
	}
	if dataMap, ok := baseRequest.Data.(map[string]interface{}); ok {
		dataMap["sync_type"] = "incremental"
		dataMap["incremental_params"] = incrementalParams
	}

	return baseRequest, nil
}

// GetQueryBuilder 工厂方法，获取查询构建器实例
func GetQueryBuilder(dataSource *models.DataSource, dataInterface *models.DataInterface) (*QueryBuilder, error) {
	return NewQueryBuilder(dataSource, dataInterface)
}
