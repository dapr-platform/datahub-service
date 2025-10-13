/*
 * @module service/governance/rule_engine
 * @description 数据治理规则引擎，提供规则直接应用到数据记录的功能
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow 规则加载 -> 数据处理 -> 结果返回
 * @rules 确保数据治理规则的正确应用和执行
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/data_governance_example.md
 */

package governance

import (
	"datahub-service/service/models"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// RuleEngine 规则引擎
type RuleEngine struct {
	db *gorm.DB
}

// NewRuleEngine 创建规则引擎实例
func NewRuleEngine(db *gorm.DB) *RuleEngine {
	return &RuleEngine{db: db}
}

// RuleExecutionResult 规则执行结果
type RuleExecutionResult struct {
	Success       bool                   `json:"success"`
	ProcessedData map[string]interface{} `json:"processed_data"`
	QualityScore  float64                `json:"quality_score,omitempty"`
	Issues        []string               `json:"issues,omitempty"`
	Modifications map[string]interface{} `json:"modifications,omitempty"`
	ExecutionTime time.Duration          `json:"execution_time"`
	RulesApplied  []string               `json:"rules_applied"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

// ApplyQualityRules 应用数据质量规则
func (re *RuleEngine) ApplyQualityRules(data map[string]interface{}, configs []models.QualityRuleConfig) (*RuleExecutionResult, error) {
	startTime := time.Now()
	result := &RuleExecutionResult{
		Success:       true,
		ProcessedData: make(map[string]interface{}),
		Issues:        []string{},
		RulesApplied:  []string{},
		QualityScore:  1.0,
	}

	// 复制原始数据
	for k, v := range data {
		result.ProcessedData[k] = v
	}

	totalChecks := 0
	passedChecks := 0

	for _, config := range configs {
		if !config.IsEnabled {
			continue
		}

		// 获取规则模板
		template, err := re.getQualityRuleTemplate(config.RuleTemplateID)
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("规则模板 %s 不存在: %v", config.RuleTemplateID, err))
			continue
		}

		// 应用规则到指定字段
		for _, fieldName := range config.TargetFields {
			totalChecks++
			fieldValue, exists := result.ProcessedData[fieldName]
			if !exists {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 不存在", fieldName))
				continue
			}

			// 根据规则类型执行检查
			passed, issue := re.executeQualityRule(template, fieldName, fieldValue, config.RuntimeConfig, config.Threshold)
			if passed {
				passedChecks++
			} else {
				result.Issues = append(result.Issues, issue)
			}

			result.RulesApplied = append(result.RulesApplied, fmt.Sprintf("%s:%s", template.Type, fieldName))
		}
	}

	// 计算质量分数
	if totalChecks > 0 {
		result.QualityScore = float64(passedChecks) / float64(totalChecks)
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// ApplyMaskingRules 应用数据脱敏规则
func (re *RuleEngine) ApplyMaskingRules(data map[string]interface{}, configs []models.DataMaskingConfig) (*RuleExecutionResult, error) {
	startTime := time.Now()
	result := &RuleExecutionResult{
		Success:       true,
		ProcessedData: make(map[string]interface{}),
		Issues:        []string{},
		RulesApplied:  []string{},
		Modifications: make(map[string]interface{}),
	}

	// 复制原始数据
	for k, v := range data {
		result.ProcessedData[k] = v
	}

	for _, config := range configs {
		if !config.IsEnabled {
			continue
		}

		// 获取脱敏模板
		template, err := re.getMaskingTemplate(config.TemplateID)
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("脱敏模板 %s 不存在: %v", config.TemplateID, err))
			continue
		}

		// 检查应用条件
		if config.ApplyCondition != "" {
			shouldApply, err := re.evaluateCondition(config.ApplyCondition, result.ProcessedData)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("条件评估失败: %v", err))
				continue
			}
			if !shouldApply {
				continue
			}
		}

		// 应用脱敏规则到指定字段
		for _, fieldName := range config.TargetFields {
			fieldValue, exists := result.ProcessedData[fieldName]
			if !exists {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 不存在", fieldName))
				continue
			}

			originalValue := fieldValue
			maskedValue, err := re.executeMaskingRule(template, fieldValue, config.MaskingConfig, config.PreserveFormat)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 脱敏失败: %v", fieldName, err))
				continue
			}

			result.ProcessedData[fieldName] = maskedValue
			result.Modifications[fieldName] = map[string]interface{}{
				"original": originalValue,
				"masked":   maskedValue,
				"rule":     template.MaskingType,
			}
			result.RulesApplied = append(result.RulesApplied, fmt.Sprintf("%s:%s", template.MaskingType, fieldName))
		}
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// ApplyCleansingRules 应用数据清洗规则
func (re *RuleEngine) ApplyCleansingRules(data map[string]interface{}, configs []models.DataCleansingConfig) (*RuleExecutionResult, error) {
	startTime := time.Now()
	result := &RuleExecutionResult{
		Success:       true,
		ProcessedData: make(map[string]interface{}),
		Issues:        []string{},
		RulesApplied:  []string{},
		Modifications: make(map[string]interface{}),
	}

	// 复制原始数据
	for k, v := range data {
		result.ProcessedData[k] = v
	}

	for _, config := range configs {
		if !config.IsEnabled {
			continue
		}

		// 获取清洗模板
		template, err := re.getCleansingTemplate(config.TemplateID)
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("清洗模板 %s 不存在: %v", config.TemplateID, err))
			continue
		}

		// 检查前置条件
		if config.PreCondition != "" {
			shouldProcess, err := re.evaluateCondition(config.PreCondition, result.ProcessedData)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("前置条件评估失败: %v", err))
				continue
			}
			if !shouldProcess {
				continue
			}
		}

		// 应用清洗规则到指定字段
		for _, fieldName := range config.TargetFields {
			fieldValue, exists := result.ProcessedData[fieldName]
			if !exists {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 不存在", fieldName))
				continue
			}

			originalValue := fieldValue
			cleanedValue, err := re.executeCleansingRule(template, fieldValue, config.CleansingConfig)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 清洗失败: %v", fieldName, err))
				continue
			}

			result.ProcessedData[fieldName] = cleanedValue
			result.Modifications[fieldName] = map[string]interface{}{
				"original": originalValue,
				"cleaned":  cleanedValue,
				"rule":     template.RuleType,
			}
			result.RulesApplied = append(result.RulesApplied, fmt.Sprintf("%s:%s", template.RuleType, fieldName))
		}

		// 检查后置条件
		if config.PostCondition != "" {
			conditionMet, err := re.evaluateCondition(config.PostCondition, result.ProcessedData)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("后置条件评估失败: %v", err))
			} else if !conditionMet {
				result.Issues = append(result.Issues, "后置条件未满足")
			}
		}
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// 获取质量规则模板
func (re *RuleEngine) getQualityRuleTemplate(templateID string) (*models.QualityRuleTemplate, error) {
	var template models.QualityRuleTemplate
	if err := re.db.First(&template, "id = ? AND is_enabled = ?", templateID, true).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// 获取脱敏模板
func (re *RuleEngine) getMaskingTemplate(templateID string) (*models.DataMaskingTemplate, error) {
	var template models.DataMaskingTemplate
	if err := re.db.First(&template, "id = ? AND is_enabled = ?", templateID, true).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// 获取清洗模板
func (re *RuleEngine) getCleansingTemplate(templateID string) (*models.DataCleansingTemplate, error) {
	var template models.DataCleansingTemplate
	if err := re.db.First(&template, "id = ? AND is_enabled = ?", templateID, true).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// 执行质量规则检查
func (re *RuleEngine) executeQualityRule(template *models.QualityRuleTemplate, fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	// 合并模板的RuleLogic和运行时配置
	mergedConfig := make(map[string]interface{})

	// 首先添加模板的RuleLogic
	for k, v := range template.RuleLogic {
		mergedConfig[k] = v
	}

	// 然后添加运行时配置（覆盖模板配置）
	for k, v := range runtimeConfig {
		mergedConfig[k] = v
	}

	switch template.Type {
	case "completeness":
		return re.checkCompletenessWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	case "accuracy":
		return re.checkAccuracyWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	case "consistency":
		return re.checkConsistencyWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	case "validity":
		return re.checkValidityWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	case "uniqueness":
		return re.checkUniquenessWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	case "timeliness":
		return re.checkTimelinessWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	case "standardization":
		return re.checkStandardizationWithConfig(fieldName, fieldValue, mergedConfig, threshold)
	default:
		return false, fmt.Sprintf("未知的质量规则类型: %s", template.Type)
	}
}

// 执行脱敏规则
func (re *RuleEngine) executeMaskingRule(template *models.DataMaskingTemplate, fieldValue interface{}, maskingConfig map[string]interface{}, preserveFormat bool) (interface{}, error) {
	if fieldValue == nil {
		return nil, nil
	}

	// 合并模板的MaskingLogic和运行时配置
	mergedConfig := make(map[string]interface{})

	// 首先添加模板的MaskingLogic
	for k, v := range template.MaskingLogic {
		mergedConfig[k] = v
	}

	// 然后添加运行时配置（覆盖模板配置）
	for k, v := range maskingConfig {
		mergedConfig[k] = v
	}

	strValue := fmt.Sprintf("%v", fieldValue)
	switch template.MaskingType {
	case "mask":
		return re.maskValue(strValue, mergedConfig, preserveFormat)
	case "replace":
		return re.replaceValue(strValue, mergedConfig)
	case "encrypt":
		return re.encryptValue(strValue, mergedConfig)
	case "pseudonymize":
		return re.pseudonymizeValue(strValue, mergedConfig)
	default:
		return fieldValue, fmt.Errorf("未知的脱敏类型: %s", template.MaskingType)
	}
}

// 执行清洗规则
func (re *RuleEngine) executeCleansingRule(template *models.DataCleansingTemplate, fieldValue interface{}, cleansingConfig map[string]interface{}) (interface{}, error) {
	// 合并模板的CleansingLogic和运行时配置
	mergedConfig := make(map[string]interface{})

	// 首先添加模板的CleansingLogic
	for k, v := range template.CleansingLogic {
		mergedConfig[k] = v
	}

	// 然后添加运行时配置（覆盖模板配置）
	for k, v := range cleansingConfig {
		mergedConfig[k] = v
	}

	switch template.RuleType {
	case "standardization":
		return re.standardizeValue(fieldValue, mergedConfig)
	case "deduplication":
		return re.deduplicateValue(fieldValue, mergedConfig)
	case "validation":
		return re.validateValue(fieldValue, mergedConfig)
	case "transformation":
		return re.transformValue(fieldValue, mergedConfig)
	case "enrichment":
		return re.enrichValue(fieldValue, mergedConfig)
	default:
		return fieldValue, fmt.Errorf("未知的清洗规则类型: %s", template.RuleType)
	}
}

// 完整性检查
func (re *RuleEngine) checkCompleteness(fieldName string, fieldValue interface{}, threshold map[string]interface{}) (bool, string) {
	if fieldValue == nil {
		return false, fmt.Sprintf("字段 %s 为空", fieldName)
	}

	strValue := strings.TrimSpace(fmt.Sprintf("%v", fieldValue))
	if strValue == "" {
		return false, fmt.Sprintf("字段 %s 为空字符串", fieldName)
	}

	return true, ""
}

// 准确性检查
func (re *RuleEngine) checkAccuracy(fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	if fieldValue == nil {
		return false, fmt.Sprintf("字段 %s 为空", fieldName)
	}

	// 检查数据格式准确性
	if expectedFormat, ok := runtimeConfig["expected_format"].(string); ok {
		strValue := fmt.Sprintf("%v", fieldValue)
		matched, _ := regexp.MatchString(expectedFormat, strValue)
		if !matched {
			return false, fmt.Sprintf("字段 %s 格式不正确，期望格式: %s", fieldName, expectedFormat)
		}
	}

	return true, ""
}

// 一致性检查
func (re *RuleEngine) checkConsistency(fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	// 简化实现，实际应用中需要跨表或跨字段检查
	return true, ""
}

// 有效性检查
func (re *RuleEngine) checkValidity(fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	if fieldValue == nil {
		return false, fmt.Sprintf("字段 %s 为空", fieldName)
	}

	strValue := fmt.Sprintf("%v", fieldValue)

	// 检查值范围
	if validValues, ok := runtimeConfig["valid_values"].([]interface{}); ok {
		for _, validValue := range validValues {
			if strValue == fmt.Sprintf("%v", validValue) {
				return true, ""
			}
		}
		return false, fmt.Sprintf("字段 %s 值 %s 不在有效值范围内", fieldName, strValue)
	}

	// 检查正则表达式
	if pattern, ok := runtimeConfig["pattern"].(string); ok {
		matched, _ := regexp.MatchString(pattern, strValue)
		if !matched {
			return false, fmt.Sprintf("字段 %s 值 %s 不匹配模式 %s", fieldName, strValue, pattern)
		}
	}

	return true, ""
}

// 唯一性检查
func (re *RuleEngine) checkUniqueness(fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	// 简化实现，实际应用中需要检查数据库中的唯一性
	return true, ""
}

// 及时性检查
func (re *RuleEngine) checkTimeliness(fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	if fieldValue == nil {
		return false, fmt.Sprintf("字段 %s 为空", fieldName)
	}

	// 检查时间字段的新鲜度
	if timeValue, ok := fieldValue.(time.Time); ok {
		maxAge, _ := runtimeConfig["max_age_hours"].(float64)
		if maxAge > 0 {
			age := time.Since(timeValue).Hours()
			if age > maxAge {
				return false, fmt.Sprintf("字段 %s 数据过期，已超过 %f 小时", fieldName, maxAge)
			}
		}
	}

	return true, ""
}

// 标准化检查
func (re *RuleEngine) checkStandardization(fieldName string, fieldValue interface{}, runtimeConfig, threshold map[string]interface{}) (bool, string) {
	if fieldValue == nil {
		return false, fmt.Sprintf("字段 %s 为空", fieldName)
	}

	strValue := fmt.Sprintf("%v", fieldValue)

	// 检查标准格式
	if standardFormat, ok := runtimeConfig["standard_format"].(string); ok {
		matched, _ := regexp.MatchString(standardFormat, strValue)
		if !matched {
			return false, fmt.Sprintf("字段 %s 值 %s 不符合标准格式 %s", fieldName, strValue, standardFormat)
		}
	}

	return true, ""
}

// 掩码处理
func (re *RuleEngine) maskValue(value string, config map[string]interface{}, preserveFormat bool) (string, error) {
	if value == "" {
		return value, nil
	}

	maskChar := "*"
	if mc, ok := config["mask_char"].(string); ok {
		maskChar = mc
	}

	keepStart := 0
	if ks, ok := config["keep_start"].(float64); ok {
		keepStart = int(ks)
	} else if ks, ok := config["keep_start"].(int); ok {
		keepStart = ks
	}

	keepEnd := 0
	if ke, ok := config["keep_end"].(float64); ok {
		keepEnd = int(ke)
	} else if ke, ok := config["keep_end"].(int); ok {
		keepEnd = ke
	}

	// 如果保留的字符数大于等于总长度，则全部脱敏
	if keepStart+keepEnd >= len(value) {
		return strings.Repeat(maskChar, len(value)), nil
	}

	// 构建脱敏后的字符串
	var result strings.Builder

	// 保留开头的字符
	if keepStart > 0 {
		result.WriteString(value[:keepStart])
	}

	// 中间部分用掩码字符替换
	middleLength := len(value) - keepStart - keepEnd
	if middleLength > 0 {
		result.WriteString(strings.Repeat(maskChar, middleLength))
	}

	// 保留结尾的字符
	if keepEnd > 0 {
		result.WriteString(value[len(value)-keepEnd:])
	}

	return result.String(), nil
}

// 替换处理
func (re *RuleEngine) replaceValue(value string, config map[string]interface{}) (string, error) {
	replacement := "***"
	if r, ok := config["replacement"].(string); ok {
		replacement = r
	}
	return replacement, nil
}

// 加密处理
func (re *RuleEngine) encryptValue(value string, config map[string]interface{}) (string, error) {
	// 简化实现，实际应用中应使用真正的加密算法
	return fmt.Sprintf("ENC_%x", []byte(value)), nil
}

// 假名化处理
func (re *RuleEngine) pseudonymizeValue(value string, config map[string]interface{}) (string, error) {
	// 检查是否有特定的伪名化模式配置
	if pattern, ok := config["pattern"].(string); ok {
		switch pattern {
		case "chinese_name":
			// 生成中文假名
			hash := 0
			for _, c := range value {
				hash = hash*31 + int(c)
			}
			return fmt.Sprintf("用户%d", hash%1000), nil
		case "user_id":
			hash := 0
			for _, c := range value {
				hash = hash*31 + int(c)
			}
			return fmt.Sprintf("USER_%d", hash%10000), nil
		}
	}

	// 默认实现：生成中文用户名
	hash := 0
	for _, c := range value {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("用户%d", hash%1000), nil
}

// 标准化处理
func (re *RuleEngine) standardizeValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	strValue := fmt.Sprintf("%v", value)

	// 去除空格
	if trimSpaces, ok := config["trim_spaces"].(bool); ok && trimSpaces {
		strValue = strings.TrimSpace(strValue)
	}

	// 特殊的标准化类型
	if standardizationType, ok := config["standardization_type"].(string); ok {
		switch standardizationType {
		case "email_lowercase":
			strValue = strings.ToLower(strValue)
		case "phone_format":
			// 移除非数字字符
			re := regexp.MustCompile(`[^0-9]`)
			strValue = re.ReplaceAllString(strValue, "")
		}
	}

	// 转换大小写
	if caseType, ok := config["case"].(string); ok {
		switch caseType {
		case "upper":
			strValue = strings.ToUpper(strValue)
		case "lower":
			strValue = strings.ToLower(strValue)
		case "title":
			strValue = strings.Title(strValue)
		}
	}

	// 格式化数字
	if formatType, ok := config["format_type"].(string); ok && formatType == "number" {
		if precision, ok := config["precision"].(float64); ok {
			if numValue, err := strconv.ParseFloat(strValue, 64); err == nil {
				strValue = fmt.Sprintf("%."+fmt.Sprintf("%.0f", precision)+"f", numValue)
			}
		}
	}

	return strValue, nil
}

// 去重处理
func (re *RuleEngine) deduplicateValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	// 对于单个值，去重主要是去除重复字符
	strValue := fmt.Sprintf("%v", value)

	if removeDuplicateChars, ok := config["remove_duplicate_chars"].(bool); ok && removeDuplicateChars {
		seen := make(map[rune]bool)
		var result strings.Builder
		for _, char := range strValue {
			if !seen[char] {
				seen[char] = true
				result.WriteRune(char)
			}
		}
		return result.String(), nil
	}

	return value, nil
}

// 验证处理
func (re *RuleEngine) validateValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	strValue := fmt.Sprintf("%v", value)

	// 验证并修正格式
	if pattern, ok := config["validation_pattern"].(string); ok {
		matched, _ := regexp.MatchString(pattern, strValue)
		if !matched {
			if defaultValue, ok := config["default_value"]; ok {
				return defaultValue, nil
			}
			return nil, fmt.Errorf("值 %s 验证失败", strValue)
		}
	}

	return value, nil
}

// 转换处理
func (re *RuleEngine) transformValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	if transformType, ok := config["transform_type"].(string); ok {
		switch transformType {
		case "date_format":
			if fromFormat, ok := config["from_format"].(string); ok {
				if toFormat, ok := config["to_format"].(string); ok {
					if t, err := time.Parse(fromFormat, fmt.Sprintf("%v", value)); err == nil {
						return t.Format(toFormat), nil
					}
				}
			}
		case "number_format":
			if numValue, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64); err == nil {
				if multiplier, ok := config["multiplier"].(float64); ok {
					numValue *= multiplier
				}
				return numValue, nil
			}
		case "format_conversion":
			strValue := fmt.Sprintf("%v", value)
			if targetFormat, ok := config["target_format"].(string); ok {
				switch targetFormat {
				case "uppercase":
					return strings.ToUpper(strValue), nil
				case "lowercase":
					return strings.ToLower(strValue), nil
				case "title_case":
					return strings.Title(strValue), nil
				}
			}
		}
	}

	return value, nil
}

// 丰富处理
func (re *RuleEngine) enrichValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	// 处理空值填充
	if value == nil || value == "" {
		if defaultValue, ok := config["default_value"]; ok {
			return defaultValue, nil
		}
	}

	// 如果不是空值，继续其他处理
	strValue := fmt.Sprintf("%v", value)

	// 添加前缀
	if prefix, ok := config["prefix"].(string); ok {
		strValue = prefix + strValue
	}

	// 添加后缀
	if suffix, ok := config["suffix"].(string); ok {
		strValue = strValue + suffix
	}

	return strValue, nil
}

// 评估条件表达式
func (re *RuleEngine) evaluateCondition(condition string, data map[string]interface{}) (bool, error) {
	// 简化实现，实际应用中需要更完整的表达式解析器
	// 这里只处理基本的字段存在性和值比较

	if strings.Contains(condition, "IS NOT NULL") {
		fieldName := strings.TrimSpace(strings.Replace(condition, "IS NOT NULL", "", 1))
		_, exists := data[fieldName]
		return exists, nil
	}

	if strings.Contains(condition, "IS NULL") {
		fieldName := strings.TrimSpace(strings.Replace(condition, "IS NULL", "", 1))
		_, exists := data[fieldName]
		return !exists, nil
	}

	if strings.Contains(condition, "=") {
		parts := strings.Split(condition, "=")
		if len(parts) == 2 {
			fieldName := strings.TrimSpace(parts[0])
			expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			actualValue := fmt.Sprintf("%v", data[fieldName])
			return actualValue == expectedValue, nil
		}
	}

	// 默认返回true
	return true, nil
}

// ApplyQualityRulesWithTemplates 应用数据质量规则（直接传入模板）
func (re *RuleEngine) ApplyQualityRulesWithTemplates(data map[string]interface{}, configs []models.QualityRuleConfig, templates map[string]*models.QualityRuleTemplate) (*RuleExecutionResult, error) {
	startTime := time.Now()
	result := &RuleExecutionResult{
		Success:       true,
		ProcessedData: make(map[string]interface{}),
		Issues:        []string{},
		RulesApplied:  []string{},
		QualityScore:  1.0,
	}

	// 复制原始数据
	for k, v := range data {
		result.ProcessedData[k] = v
	}

	totalChecks := 0
	passedChecks := 0

	for _, config := range configs {
		if !config.IsEnabled {
			continue
		}

		// 从传入的模板中获取规则模板
		template, exists := templates[config.RuleTemplateID]
		if !exists {
			result.Issues = append(result.Issues, fmt.Sprintf("规则模板 %s 不存在", config.RuleTemplateID))
			continue
		}

		// 应用规则到指定字段
		for _, fieldName := range config.TargetFields {
			totalChecks++
			fieldValue, exists := result.ProcessedData[fieldName]
			if !exists {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 不存在", fieldName))
				continue
			}

			// 根据规则类型执行检查
			passed, issue := re.executeQualityRule(template, fieldName, fieldValue, config.RuntimeConfig, config.Threshold)
			if passed {
				passedChecks++
			} else {
				result.Issues = append(result.Issues, issue)
			}

			result.RulesApplied = append(result.RulesApplied, fmt.Sprintf("%s:%s", template.Type, fieldName))
		}
	}

	// 计算质量分数
	if totalChecks > 0 {
		result.QualityScore = float64(passedChecks) / float64(totalChecks)
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// ApplyMaskingRulesWithTemplates 应用数据脱敏规则（直接传入模板）
func (re *RuleEngine) ApplyMaskingRulesWithTemplates(data map[string]interface{}, configs []models.DataMaskingConfig, templates map[string]*models.DataMaskingTemplate) (*RuleExecutionResult, error) {
	startTime := time.Now()
	result := &RuleExecutionResult{
		Success:       true,
		ProcessedData: make(map[string]interface{}),
		Issues:        []string{},
		RulesApplied:  []string{},
		Modifications: make(map[string]interface{}),
	}

	// 复制原始数据
	for k, v := range data {
		result.ProcessedData[k] = v
	}

	for _, config := range configs {
		if !config.IsEnabled {
			continue
		}

		// 从传入的模板中获取脱敏模板
		template, exists := templates[config.TemplateID]
		if !exists {
			result.Issues = append(result.Issues, fmt.Sprintf("脱敏模板 %s 不存在", config.TemplateID))
			continue
		}

		// 应用脱敏规则到指定字段
		for _, fieldName := range config.TargetFields {
			fieldValue, exists := result.ProcessedData[fieldName]
			if !exists {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 不存在", fieldName))
				continue
			}

			// 执行脱敏处理
			maskedValue, err := re.executeMaskingRule(template, fieldValue, config.MaskingConfig, config.PreserveFormat)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 脱敏失败: %v", fieldName, err))
				continue
			}

			// 记录修改
			result.Modifications[fieldName] = map[string]interface{}{
				"original": fieldValue,
				"masked":   maskedValue,
			}

			// 更新处理后的数据
			result.ProcessedData[fieldName] = maskedValue
			result.RulesApplied = append(result.RulesApplied, fmt.Sprintf("%s:%s", template.MaskingType, fieldName))
		}
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// ApplyCleansingRulesWithTemplates 应用数据清洗规则（直接传入模板）
func (re *RuleEngine) ApplyCleansingRulesWithTemplates(data map[string]interface{}, configs []models.DataCleansingConfig, templates map[string]*models.DataCleansingTemplate) (*RuleExecutionResult, error) {
	startTime := time.Now()
	result := &RuleExecutionResult{
		Success:       true,
		ProcessedData: make(map[string]interface{}),
		Issues:        []string{},
		RulesApplied:  []string{},
		Modifications: make(map[string]interface{}),
	}

	// 复制原始数据
	for k, v := range data {
		result.ProcessedData[k] = v
	}

	for _, config := range configs {
		if !config.IsEnabled {
			continue
		}

		// 从传入的模板中获取清洗模板
		template, exists := templates[config.TemplateID]
		if !exists {
			result.Issues = append(result.Issues, fmt.Sprintf("清洗模板 %s 不存在", config.TemplateID))
			continue
		}

		// 检查触发条件
		if config.TriggerCondition != "" {
			triggered, err := re.evaluateCondition(config.TriggerCondition, result.ProcessedData)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("触发条件评估失败: %v", err))
				continue
			}
			if !triggered {
				continue
			}
		}

		// 应用清洗规则到指定字段
		for _, fieldName := range config.TargetFields {
			fieldValue, exists := result.ProcessedData[fieldName]
			if !exists {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 不存在", fieldName))
				continue
			}

			// 执行清洗处理
			cleanedValue, err := re.executeCleansingRule(template, fieldValue, config.CleansingConfig)
			if err != nil {
				result.Issues = append(result.Issues, fmt.Sprintf("字段 %s 清洗失败: %v", fieldName, err))
				continue
			}

			// 记录修改
			result.Modifications[fieldName] = map[string]interface{}{
				"original": fieldValue,
				"cleaned":  cleanedValue,
			}

			// 更新处理后的数据
			result.ProcessedData[fieldName] = cleanedValue
			result.RulesApplied = append(result.RulesApplied, fmt.Sprintf("%s:%s", template.RuleType, fieldName))
		}
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// 带配置的质量检查方法

// checkCompletenessWithConfig 检查完整性（带配置）
func (re *RuleEngine) checkCompletenessWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 检查是否启用null检查 (默认启用)
	checkNull := true
	if val, exists := config["check_null"]; exists {
		if b, ok := val.(bool); ok {
			checkNull = b
		}
	}

	// 检查字段是否为空或nil
	if checkNull && fieldValue == nil {
		return false, fmt.Sprintf("字段 %s 完整性检查失败: 字段为空值", fieldName)
	}

	// 检查是否启用空字符串检查 (默认启用)
	checkEmptyString := true
	if val, exists := config["check_empty_string"]; exists {
		if b, ok := val.(bool); ok {
			checkEmptyString = b
		}
	}

	// 检查是否启用空白字符检查 (默认启用)
	checkWhitespaceOnly := true
	if val, exists := config["check_whitespace_only"]; exists {
		if b, ok := val.(bool); ok {
			checkWhitespaceOnly = b
		}
	}

	// 检查字符串是否为空
	if str, ok := fieldValue.(string); ok {
		if checkEmptyString && str == "" {
			return false, fmt.Sprintf("字段 %s 完整性检查失败: 字段为空字符串", fieldName)
		}

		if checkWhitespaceOnly && strings.TrimSpace(str) == "" {
			return false, fmt.Sprintf("字段 %s 完整性检查失败: 字段仅包含空白字符", fieldName)
		}

		// 检查最小长度要求
		if minLength, exists := config["min_length"]; exists {
			if minLen, ok := minLength.(float64); ok {
				if len(str) < int(minLen) {
					return false, fmt.Sprintf("字段 %s 完整性检查失败: 长度不足，要求至少 %d 个字符", fieldName, int(minLen))
				}
			}
		}
	}

	// 检查是否允许数值0
	allowZero := true
	if val, exists := config["allow_zero"]; exists {
		if b, ok := val.(bool); ok {
			allowZero = b
		}
	}

	if !allowZero {
		// 检查数值类型
		if num, ok := fieldValue.(float64); ok && num == 0 {
			return false, fmt.Sprintf("字段 %s 完整性检查失败: 不允许数值0", fieldName)
		}
		if num, ok := fieldValue.(int); ok && num == 0 {
			return false, fmt.Sprintf("字段 %s 完整性检查失败: 不允许数值0", fieldName)
		}
	}

	return true, ""
}

// checkAccuracyWithConfig 检查准确性（带配置）
func (re *RuleEngine) checkAccuracyWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 检查数值范围
	if numValue, ok := fieldValue.(float64); ok {
		if minVal, exists := config["min_value"]; exists {
			if min, ok := minVal.(float64); ok && numValue < min {
				return false, fmt.Sprintf("字段 %s 准确性检查失败: 值 %f 小于最小值 %f", fieldName, numValue, min)
			}
		}
		if maxVal, exists := config["max_value"]; exists {
			if max, ok := maxVal.(float64); ok && numValue > max {
				return false, fmt.Sprintf("字段 %s 准确性检查失败: 值 %f 大于最大值 %f", fieldName, numValue, max)
			}
		}
	}

	// 检查正则表达式匹配
	if pattern, exists := config["regex_pattern"]; exists {
		if patternStr, ok := pattern.(string); ok {
			if regex, err := regexp.Compile(patternStr); err == nil {
				if str, ok := fieldValue.(string); ok {
					if !regex.MatchString(str) {
						return false, fmt.Sprintf("字段 %s 准确性检查失败: 邮箱格式不正确", fieldName)
					}
				}
			}
		}
	}

	// 检查验证类型
	if validationType, exists := config["validation_type"]; exists {
		if validationTypeStr, ok := validationType.(string); ok {
			switch validationTypeStr {
			case "email":
				if str, ok := fieldValue.(string); ok {
					emailRegex := `^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$`
					// 处理转义字符
					emailRegex = strings.ReplaceAll(emailRegex, "\\\\", "\\")
					if matched, _ := regexp.MatchString(emailRegex, str); !matched {
						return false, fmt.Sprintf("字段 %s 准确性检查失败: 邮箱格式不正确", fieldName)
					}
				}
			}
		}
	}

	return true, ""
}

// checkConsistencyWithConfig 检查一致性（带配置）
func (re *RuleEngine) checkConsistencyWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 检查与其他字段的一致性
	if relatedField, exists := config["related_field"]; exists {
		if relatedFieldName, ok := relatedField.(string); ok {
			// 这里需要从数据记录中获取相关字段的值进行比较
			// 在实际应用中，这个方法需要访问完整的数据记录
			return true, fmt.Sprintf("需要检查字段 %s 与 %s 的一致性", fieldName, relatedFieldName)
		}
	}

	// 检查值的格式一致性
	if format, exists := config["format"]; exists {
		if formatStr, ok := format.(string); ok {
			switch formatStr {
			case "email":
				if str, ok := fieldValue.(string); ok {
					emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
					if matched, _ := regexp.MatchString(emailRegex, str); !matched {
						return false, fmt.Sprintf("字段 %s 值 '%s' 不是有效的邮箱格式", fieldName, str)
					}
				}
			case "phone":
				if str, ok := fieldValue.(string); ok {
					phoneRegex := `^1[3-9]\d{9}$`
					if matched, _ := regexp.MatchString(phoneRegex, str); !matched {
						return false, fmt.Sprintf("字段 %s 值 '%s' 不是有效的手机号格式", fieldName, str)
					}
				}
			}
		}
	}

	return true, ""
}

// checkValidityWithConfig 检查有效性（带配置）
func (re *RuleEngine) checkValidityWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 检查值是否在允许的列表中
	if allowedValues, exists := config["allowed_values"]; exists {
		if allowedList, ok := allowedValues.([]interface{}); ok {
			valueFound := false
			for _, allowed := range allowedList {
				if fieldValue == allowed {
					valueFound = true
					break
				}
			}
			if !valueFound {
				return false, fmt.Sprintf("字段 %s 值 '%v' 不在允许的值列表中", fieldName, fieldValue)
			}
		}
	}

	// 检查日期有效性
	if dateFormat, exists := config["date_format"]; exists {
		if formatStr, ok := dateFormat.(string); ok {
			if str, ok := fieldValue.(string); ok {
				if _, err := time.Parse(formatStr, str); err != nil {
					return false, fmt.Sprintf("字段 %s 值 '%s' 不符合日期格式 '%s'", fieldName, str, formatStr)
				}
			}
		}
	}

	return true, ""
}

// checkUniquenessWithConfig 检查唯一性（带配置）
func (re *RuleEngine) checkUniquenessWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 唯一性检查需要访问数据库或缓存
	// 这里返回一个占位符结果
	return true, fmt.Sprintf("字段 %s 唯一性检查需要数据库支持", fieldName)
}

// checkTimelinessWithConfig 检查时效性（带配置）
func (re *RuleEngine) checkTimelinessWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 检查数据是否在有效时间范围内
	if maxAge, exists := config["max_age_days"]; exists {
		if maxAgeDays, ok := maxAge.(float64); ok {
			if dateStr, ok := fieldValue.(string); ok {
				// 尝试解析日期
				formats := []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339}
				var parsedTime time.Time
				var err error

				for _, format := range formats {
					parsedTime, err = time.Parse(format, dateStr)
					if err == nil {
						break
					}
				}

				if err != nil {
					return false, fmt.Sprintf("字段 %s 值 '%s' 无法解析为日期", fieldName, dateStr)
				}

				daysDiff := time.Since(parsedTime).Hours() / 24
				if daysDiff > maxAgeDays {
					return false, fmt.Sprintf("字段 %s 数据过期，距今 %.1f 天，超过最大允许 %.0f 天", fieldName, daysDiff, maxAgeDays)
				}
			}
		}
	}

	return true, ""
}

// checkStandardizationWithConfig 检查标准化（带配置）
func (re *RuleEngine) checkStandardizationWithConfig(fieldName string, fieldValue interface{}, config map[string]interface{}, threshold map[string]interface{}) (bool, string) {
	// 检查数据是否符合标准化格式
	if standardFormat, exists := config["standard_format"]; exists {
		if formatStr, ok := standardFormat.(string); ok {
			if str, ok := fieldValue.(string); ok {
				switch formatStr {
				case "uppercase":
					if str != strings.ToUpper(str) {
						return false, fmt.Sprintf("字段 %s 值 '%s' 应该是大写格式", fieldName, str)
					}
				case "lowercase":
					if str != strings.ToLower(str) {
						return false, fmt.Sprintf("字段 %s 值 '%s' 应该是小写格式", fieldName, str)
					}
				case "title_case":
					if str != strings.Title(str) {
						return false, fmt.Sprintf("字段 %s 值 '%s' 应该是标题格式", fieldName, str)
					}
				}
			}
		}
	}

	return true, ""
}
