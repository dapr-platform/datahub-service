/*
 * @module service/basic_library/validation_service
 * @description 数据源配置验证服务，基于统一的数据源类型注册表进行验证
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 配置验证 -> 类型注册表查询 -> 统一验证规则 -> 结果返回
 * @rules 确保数据源配置的正确性和安全性，基于统一配置避免重复验证逻辑
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/interfaces.md, service/models/datasource_types.go
 */

package basic_library

import (
	"datahub-service/service/meta"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ValidationService 数据源配置验证服务
type ValidationService struct {
	db *gorm.DB
}

// NewValidationService 创建数据源配置验证服务实例
func NewValidationService(db *gorm.DB) *ValidationService {
	return &ValidationService{
		db: db,
	}
}

// ConfigValidationResult 配置验证结果（兼容原接口）
type ConfigValidationResult struct {
	IsValid     bool                   `json:"is_valid"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	Score       int                    `json:"score"` // 配置质量评分 0-100
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ValidateDataSourceConfig 验证数据源配置
func (s *ValidationService) ValidateDataSourceConfig(dataSourceType string, connectionConfig, paramsConfig map[string]interface{}) (*ConfigValidationResult, error) {
	// 从注册表获取类型定义
	definition, exists := meta.DataSourceTypes[dataSourceType]
	if !exists {
		return &ConfigValidationResult{
			IsValid: false,
			Errors:  []string{fmt.Sprintf("不支持的数据源类型: %s", dataSourceType)},
			Score:   0,
		}, nil
	}

	// 使用统一的验证逻辑
	validationResult := definition.ValidateConfig(connectionConfig, paramsConfig)

	// 转换为兼容的结果格式
	result := &ConfigValidationResult{
		IsValid:  validationResult.IsValid,
		Errors:   validationResult.Errors,
		Warnings: validationResult.Warnings,
		Score:    validationResult.Score,
	}

	// 添加建议信息
	result.Suggestions = s.generateSuggestions(definition, connectionConfig, paramsConfig)

	// 添加详细信息
	result.Details = s.generateDetails(definition, result)

	return result, nil
}

// generateSuggestions 生成配置建议
func (s *ValidationService) generateSuggestions(definition *meta.DataSourceTypeDefinition, connectionConfig, paramsConfig map[string]interface{}) []string {
	suggestions := make([]string, 0)

	// 根据数据源类型添加通用建议
	switch definition.Category {
	case meta.DataSourceCategoryDatabase:
		suggestions = append(suggestions, "建议启用SSL连接确保数据传输安全")
		suggestions = append(suggestions, "建议定期检查数据库连接状态")
		suggestions = append(suggestions, "确保数据库用户权限配置正确")
	case meta.DataSourceCategoryMessaging:
		suggestions = append(suggestions, "建议配置适当的消息确认机制")
		suggestions = append(suggestions, "建议监控消息队列健康状态")
	case meta.DataSourceCategoryAPI:
		suggestions = append(suggestions, "建议使用HTTPS确保数据传输安全")
		suggestions = append(suggestions, "建议配置适当的超时时间")
		suggestions = append(suggestions, "确保API授权信息正确")
	}

	// 根据具体配置添加针对性建议
	if password, exists := connectionConfig["password"]; exists {
		if pwd, ok := password.(string); ok && len(pwd) < 8 {
			suggestions = append(suggestions, "建议使用至少8位的强密码")
		}
	}

	if timeout, exists := connectionConfig["timeout"]; exists {
		if timeoutVal, ok := timeout.(float64); ok && timeoutVal > 60 {
			suggestions = append(suggestions, "建议将超时时间设置在合理范围内（通常不超过60秒）")
		}
	}

	return suggestions
}

// generateDetails 生成详细信息
func (s *ValidationService) generateDetails(definition *meta.DataSourceTypeDefinition, result *ConfigValidationResult) map[string]interface{} {
	details := map[string]interface{}{
		"data_source_type":   definition.ID,
		"category":           definition.Category,
		"supported_features": definition.SupportedFeatures,
		"validation_summary": map[string]interface{}{
			"total_errors":   len(result.Errors),
			"total_warnings": len(result.Warnings),
			"quality_score":  result.Score,
		},
	}

	// 根据验证结果添加分类统计
	if len(result.Errors) > 0 {
		details["error_categories"] = s.categorizeMessages(result.Errors)
	}

	if len(result.Warnings) > 0 {
		details["warning_categories"] = s.categorizeMessages(result.Warnings)
	}

	return details
}

// categorizeMessages 对消息进行分类
func (s *ValidationService) categorizeMessages(messages []string) map[string][]string {
	categories := make(map[string][]string)

	for _, message := range messages {
		category := "general"

		if strings.Contains(message, "缺少必需字段") {
			category = "required_fields"
		} else if strings.Contains(message, "类型不正确") {
			category = "type_validation"
		} else if strings.Contains(message, "格式") {
			category = "format_validation"
		} else if strings.Contains(message, "范围") || strings.Contains(message, "值过大") || strings.Contains(message, "值过小") {
			category = "range_validation"
		} else if strings.Contains(message, "密码") {
			category = "security"
		}

		categories[category] = append(categories[category], message)
	}

	return categories
}

// ValidateConnectionOnly 仅验证连接配置
func (s *ValidationService) ValidateConnectionOnly(dataSourceType string, connectionConfig map[string]interface{}) (*ConfigValidationResult, error) {
	return s.ValidateDataSourceConfig(dataSourceType, connectionConfig, nil)
}

// validateFieldType 验证字段类型
func (s *ValidationService) validateFieldType(value interface{}, expectedType string) bool {
	if value == nil {
		return true // nil值的类型验证由required字段处理
	}

	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		if !ok {
			_, ok = value.(int)
		}
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true
	}
}

// GetValidationRules 获取数据源类型的验证规则
func (s *ValidationService) GetValidationRules(dataSourceType string) (map[string]interface{}, error) {
	definition, exists := meta.DataSourceTypes[dataSourceType]
	if !exists {
		return nil, fmt.Errorf("不支持的数据源类型: %s", dataSourceType)
	}

	rules := map[string]interface{}{
		"connection_fields": s.convertFieldsToValidationRules(definition.MetaConfig),
		"params_fields":     s.convertFieldsToValidationRules(definition.ParamsConfig),
		"custom_rules":      definition.ValidationRules,
	}

	return rules, nil
}

// convertFieldsToValidationRules 将字段定义转换为验证规则
func (s *ValidationService) convertFieldsToValidationRules(fields []meta.DataSourceConfigField) []map[string]interface{} {
	rules := make([]map[string]interface{}, 0, len(fields))

	for _, field := range fields {
		rule := map[string]interface{}{
			"name":     field.Name,
			"type":     field.Type,
			"required": field.Required,
		}

		if field.Min != 0 {
			rule["min"] = field.Min
		}

		if field.Max != 0 {
			rule["max"] = field.Max
		}

		if len(field.Options) > 0 {
			rule["options"] = field.Options
		}

		if field.Pattern != "" {
			rule["pattern"] = field.Pattern
		}

		rules = append(rules, rule)
	}

	return rules
}

// FieldValidationResult 字段验证结果
type FieldValidationResult struct {
	IsValid bool   `json:"is_valid"`
	Field   string `json:"field"`
	Type    string `json:"type"`
	Error   string `json:"error,omitempty"`
}

// GetSupportedDataSourceTypes 获取支持验证的数据源类型
func (s *ValidationService) GetSupportedDataSourceTypes() []string {
	types := make([]string, 0)

	for typeID := range meta.DataSourceTypes {
		types = append(types, typeID)
	}

	return types
}

// BatchValidateConfigs 批量验证配置
func (s *ValidationService) BatchValidateConfigs(configs []map[string]interface{}) (map[string]*ConfigValidationResult, error) {
	results := make(map[string]*ConfigValidationResult)

	for i, config := range configs {
		key := fmt.Sprintf("config_%d", i)

		dataSourceType, exists := config["type"]
		if !exists {
			results[key] = &ConfigValidationResult{
				IsValid: false,
				Errors:  []string{"缺少数据源类型字段"},
				Score:   0,
			}
			continue
		}

		typeStr, ok := dataSourceType.(string)
		if !ok {
			results[key] = &ConfigValidationResult{
				IsValid: false,
				Errors:  []string{"数据源类型必须是字符串"},
				Score:   0,
			}
			continue
		}

		connectionConfig, _ := config["connection_config"].(map[string]interface{})
		paramsConfig, _ := config["params_config"].(map[string]interface{})

		result, err := s.ValidateDataSourceConfig(typeStr, connectionConfig, paramsConfig)
		if err != nil {
			results[key] = &ConfigValidationResult{
				IsValid: false,
				Errors:  []string{err.Error()},
				Score:   0,
			}
			continue
		}

		results[key] = result
	}

	return results, nil
}
