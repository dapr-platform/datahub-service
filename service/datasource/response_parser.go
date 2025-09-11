/*
 * @module service/datasource/response_parser
 * @description HTTP响应解析器，根据配置解析API响应结果
 * @architecture 策略模式 - 根据不同的解析策略处理响应
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 响应接收 -> 成功判断 -> 数据提取 -> 错误处理 -> 分页信息提取
 * @rules 支持多种响应格式和成功判断条件，提供灵活的数据提取能力
 * @dependencies datahub-service/service/meta
 * @refs http_no_auth.go, query_builder.go
 */

package datasource

import (
	"datahub-service/service/meta"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ResponseParser HTTP响应解析器
type ResponseParser struct {
	config map[string]interface{}
}

// ParsedResponse 解析后的响应
type ParsedResponse struct {
	Success      bool                   `json:"success"`
	Data         interface{}            `json:"data"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Total        int64                  `json:"total,omitempty"`
	Page         int                    `json:"page,omitempty"`
	PageSize     int                    `json:"page_size,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewResponseParser 创建响应解析器
func NewResponseParser(interfaceConfig map[string]interface{}) *ResponseParser {
	return &ResponseParser{
		config: interfaceConfig,
	}
}

// Parse 解析HTTP响应
func (p *ResponseParser) Parse(statusCode int, responseBody []byte, responseHeaders map[string][]string) (*ParsedResponse, error) {
	result := &ParsedResponse{
		Success:  false,
		Metadata: make(map[string]interface{}),
	}

	// 记录原始响应信息
	result.Metadata["status_code"] = statusCode
	result.Metadata["response_size"] = len(responseBody)

	// 解析响应体
	var responseData interface{}
	responseType := p.getConfigString(meta.DataInterfaceConfigFieldResponseType, "json")

	switch responseType {
	case "json":
		if len(responseBody) > 0 {
			if err := json.Unmarshal(responseBody, &responseData); err != nil {
				return nil, fmt.Errorf("JSON解析失败: %w", err)
			}
		}
	case "text", "html":
		responseData = string(responseBody)
	default:
		responseData = responseBody
	}

	// 判断请求是否成功
	success, err := p.isSuccess(statusCode, responseData)
	if err != nil {
		return nil, fmt.Errorf("成功判断失败: %w", err)
	}
	result.Success = success

	if success {
		// 提取数据
		data, err := p.extractData(responseData)
		if err != nil {
			return nil, fmt.Errorf("数据提取失败: %w", err)
		}
		result.Data = data

		// 提取分页信息
		p.extractPaginationInfo(responseData, result)
	} else {
		// 提取错误信息
		p.extractErrorInfo(responseData, result)
	}

	return result, nil
}

// isSuccess 判断请求是否成功
func (p *ResponseParser) isSuccess(statusCode int, responseData interface{}) (bool, error) {
	successCondition := p.getConfigString(meta.DataInterfaceConfigFieldSuccessCondition, "status_code")

	switch successCondition {
	case "status_code":
		return p.checkStatusCodeSuccess(statusCode), nil

	case "field_value":
		return p.checkFieldValueSuccess(responseData), nil

	case "both":
		statusSuccess := p.checkStatusCodeSuccess(statusCode)
		fieldSuccess := p.checkFieldValueSuccess(responseData)
		return statusSuccess && fieldSuccess, nil

	case "custom":
		// 自定义成功判断逻辑，可以扩展
		return p.checkCustomSuccess(statusCode, responseData), nil

	default:
		return p.checkStatusCodeSuccess(statusCode), nil
	}
}

// checkStatusCodeSuccess 检查HTTP状态码是否表示成功
func (p *ResponseParser) checkStatusCodeSuccess(statusCode int) bool {
	successRange := p.getConfigString(meta.DataInterfaceConfigFieldStatusCodeSuccess, "200-299")

	// 解析状态码范围
	if strings.Contains(successRange, "-") {
		// 范围格式：200-299
		parts := strings.Split(successRange, "-")
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				return statusCode >= start && statusCode <= end
			}
		}
	} else if strings.Contains(successRange, ",") {
		// 列表格式：200,201,202
		codes := strings.Split(successRange, ",")
		for _, code := range codes {
			if c, err := strconv.Atoi(strings.TrimSpace(code)); err == nil {
				if statusCode == c {
					return true
				}
			}
		}
		return false
	} else {
		// 单个状态码
		if c, err := strconv.Atoi(successRange); err == nil {
			return statusCode == c
		}
	}

	// 默认检查2xx状态码
	return statusCode >= 200 && statusCode < 300
}

// checkFieldValueSuccess 检查响应字段值是否表示成功
func (p *ResponseParser) checkFieldValueSuccess(responseData interface{}) bool {
	successField := p.getConfigString(meta.DataInterfaceConfigFieldSuccessField, "status")
	successValue := p.getConfigString(meta.DataInterfaceConfigFieldSuccessValue, "0")

	if successField == "" {
		return true // 如果没有配置成功字段，则认为成功
	}

	// 从响应中提取字段值
	fieldValue := p.extractFieldByPath(responseData, successField)
	if fieldValue == nil {
		return false
	}

	// 检查字段值是否匹配成功值
	return p.matchSuccessValue(fieldValue, successValue)
}

// checkCustomSuccess 自定义成功判断逻辑
func (p *ResponseParser) checkCustomSuccess(statusCode int, responseData interface{}) bool {
	// 这里可以实现更复杂的自定义成功判断逻辑
	// 目前先使用状态码和字段值的组合判断
	return p.checkStatusCodeSuccess(statusCode) && p.checkFieldValueSuccess(responseData)
}

// extractData 提取响应数据
func (p *ResponseParser) extractData(responseData interface{}) (interface{}, error) {
	dataPath := p.getConfigString(meta.DataInterfaceConfigFieldDataPath, "data")

	if dataPath == "" || dataPath == "." {
		return responseData, nil
	}

	// 根据路径提取数据
	data := p.extractFieldByPath(responseData, dataPath)
	if data == nil {
		// 如果指定路径没有数据，返回原始响应
		return responseData, nil
	}

	return data, nil
}

// extractPaginationInfo 提取分页信息
func (p *ResponseParser) extractPaginationInfo(responseData interface{}, result *ParsedResponse) {
	// 提取总数
	totalField := p.getConfigString(meta.DataInterfaceConfigFieldTotalField, "total")
	if totalField != "" {
		if totalValue := p.extractFieldByPath(responseData, totalField); totalValue != nil {
			if total, ok := p.convertToInt64(totalValue); ok {
				result.Total = total
			}
		}
	}

	// 提取页码
	pageField := p.getConfigString(meta.DataInterfaceConfigFieldPageField, "page")
	if pageField != "" {
		if pageValue := p.extractFieldByPath(responseData, pageField); pageValue != nil {
			if page, ok := p.convertToInt(pageValue); ok {
				result.Page = page
			}
		}
	}

	// 提取页大小
	pageSizeField := p.getConfigString(meta.DataInterfaceConfigFieldPageSizeField, "size")
	if pageSizeField != "" {
		if sizeValue := p.extractFieldByPath(responseData, pageSizeField); sizeValue != nil {
			if size, ok := p.convertToInt(sizeValue); ok {
				result.PageSize = size
			}
		}
	}
}

// extractErrorInfo 提取错误信息
func (p *ResponseParser) extractErrorInfo(responseData interface{}, result *ParsedResponse) {
	// 提取错误代码
	errorField := p.getConfigString(meta.DataInterfaceConfigFieldErrorField, "error")
	if errorField != "" {
		if errorValue := p.extractFieldByPath(responseData, errorField); errorValue != nil {
			result.ErrorCode = fmt.Sprintf("%v", errorValue)
		}
	}

	// 提取错误消息
	errorMessageField := p.getConfigString(meta.DataInterfaceConfigFieldErrorMessageField, "message")
	if errorMessageField != "" {
		if messageValue := p.extractFieldByPath(responseData, errorMessageField); messageValue != nil {
			result.ErrorMessage = fmt.Sprintf("%v", messageValue)
		}
	}

	// 如果没有提取到错误消息，使用默认消息
	if result.ErrorMessage == "" {
		result.ErrorMessage = "请求失败"
	}
}

// extractFieldByPath 根据路径提取字段值
func (p *ResponseParser) extractFieldByPath(data interface{}, path string) interface{} {
	if path == "" || path == "." {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			if value, exists := v[part]; exists {
				current = value
			} else {
				return nil
			}
		case []interface{}:
			// 如果当前是数组，尝试提取数组中每个元素的指定字段
			var results []interface{}
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if value, exists := itemMap[part]; exists {
						results = append(results, value)
					}
				}
			}
			if len(results) > 0 {
				current = results
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// matchSuccessValue 匹配成功值
func (p *ResponseParser) matchSuccessValue(fieldValue interface{}, successValue string) bool {
	fieldStr := fmt.Sprintf("%v", fieldValue)

	// 支持多个成功值，用逗号分隔
	if strings.Contains(successValue, ",") {
		values := strings.Split(successValue, ",")
		for _, value := range values {
			if strings.TrimSpace(value) == fieldStr {
				return true
			}
		}
		return false
	}

	return fieldStr == successValue
}

// getConfigString 获取配置字符串值
func (p *ResponseParser) getConfigString(key, defaultValue string) string {
	if value, exists := p.config[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

// convertToInt64 转换为int64
func (p *ResponseParser) convertToInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

// convertToInt 转换为int
func (p *ResponseParser) convertToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}
