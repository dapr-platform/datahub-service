/*
 * @module service/thematic_sync/cleansing_engine
 * @description 数据清洗引擎，负责数据验证、标准化、转换和质量检查
 * @architecture 管道模式 - 通过规则链进行数据处理
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 规则加载 -> 数据验证 -> 标准化处理 -> 质量检查 -> 结果输出
 * @rules 确保数据清洗的完整性和一致性，支持可配置的清洗规则
 * @dependencies fmt, strings, regexp, strconv
 * @refs quality_checker.go, data_transformer.go
 */

package thematic_sync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CleansingRuleType 清洗规则类型
type CleansingRuleType string

const (
	DataValidation     CleansingRuleType = "validation"     // 数据验证
	DataNormalization  CleansingRuleType = "normalization"  // 数据标准化
	DataTransformation CleansingRuleType = "transformation" // 数据转换
	DataEnrichment     CleansingRuleType = "enrichment"     // 数据丰富
)

// RuleCondition 规则条件
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, contains, regex, is_null, not_null
	Value    interface{} `json:"value"`
}

// RuleAction 规则动作
type RuleAction struct {
	Type      string                 `json:"type"` // set, remove, transform, validate
	Field     string                 `json:"field"`
	Value     interface{}            `json:"value"`
	Transform string                 `json:"transform"` // 转换函数名
	Config    map[string]interface{} `json:"config"`    // 配置参数
}

// CleansingRule 清洗规则
type CleansingRule struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         CleansingRuleType `json:"type"`
	TargetFields []string          `json:"target_fields"`
	Conditions   []RuleCondition   `json:"conditions"`
	Actions      []RuleAction      `json:"actions"`
	Priority     int               `json:"priority"`
	IsEnabled    bool              `json:"is_enabled"`
}

// CleansingResult 清洗结果
type CleansingResult struct {
	OriginalRecord   map[string]interface{} `json:"original_record"`
	CleanedRecord    map[string]interface{} `json:"cleaned_record"`
	AppliedRules     []string               `json:"applied_rules"`
	ValidationErrors []ValidationError      `json:"validation_errors"`
	QualityScore     float64                `json:"quality_score"`
	ProcessingTime   time.Duration          `json:"processing_time"`
}

// ValidationError 验证错误
type ValidationError struct {
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	RuleID    string      `json:"rule_id"`
	ErrorType string      `json:"error_type"`
	Message   string      `json:"message"`
	Severity  string      `json:"severity"` // error, warning, info
}

// CleansingEngine 数据清洗引擎
type CleansingEngine struct {
	ruleEngine     *RuleEngine
	validator      *DataValidator
	transformer    *DataTransformer
	qualityChecker *QualityChecker
}

// NewCleansingEngine 创建数据清洗引擎
func NewCleansingEngine() *CleansingEngine {
	return &CleansingEngine{
		ruleEngine:     NewRuleEngine(),
		validator:      NewDataValidator(),
		transformer:    NewDataTransformer(),
		qualityChecker: NewQualityChecker(),
	}
}

// CleanseRecords 清洗记录
func (ce *CleansingEngine) CleanseRecords(records []map[string]interface{},
	rules []CleansingRule) ([]CleansingResult, error) {

	var results []CleansingResult

	for _, record := range records {
		result, err := ce.cleanseRecord(record, rules)
		if err != nil {
			return nil, fmt.Errorf("清洗记录失败: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// cleanseRecord 清洗单条记录
func (ce *CleansingEngine) cleanseRecord(record map[string]interface{},
	rules []CleansingRule) (CleansingResult, error) {

	startTime := time.Now()
	originalRecord := ce.copyRecord(record)
	cleanedRecord := ce.copyRecord(record)
	var appliedRules []string
	var validationErrors []ValidationError

	// 按优先级排序规则
	sortedRules := ce.sortRulesByPriority(rules)

	// 应用清洗规则
	for _, rule := range sortedRules {
		if !rule.IsEnabled {
			continue
		}

		// 检查规则条件
		if ce.matchesConditions(cleanedRecord, rule.Conditions) {
			// 应用规则动作
			errors := ce.applyRuleActions(cleanedRecord, rule.Actions, rule.ID)
			validationErrors = append(validationErrors, errors...)
			appliedRules = append(appliedRules, rule.ID)
		}
	}

	// 计算质量评分
	qualityScore := ce.qualityChecker.CalculateRecordQuality(cleanedRecord, validationErrors)

	result := CleansingResult{
		OriginalRecord:   originalRecord,
		CleanedRecord:    cleanedRecord,
		AppliedRules:     appliedRules,
		ValidationErrors: validationErrors,
		QualityScore:     qualityScore,
		ProcessingTime:   time.Since(startTime),
	}

	return result, nil
}

// matchesConditions 检查是否匹配条件
func (ce *CleansingEngine) matchesConditions(record map[string]interface{},
	conditions []RuleCondition) bool {

	if len(conditions) == 0 {
		return true // 无条件则匹配
	}

	for _, condition := range conditions {
		if !ce.matchesCondition(record, condition) {
			return false // 任意条件不匹配则失败
		}
	}

	return true
}

// matchesCondition 检查单个条件
func (ce *CleansingEngine) matchesCondition(record map[string]interface{},
	condition RuleCondition) bool {

	fieldValue := record[condition.Field]

	switch condition.Operator {
	case "eq":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", condition.Value)
	case "ne":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", condition.Value)
	case "is_null":
		return fieldValue == nil
	case "not_null":
		return fieldValue != nil
	case "contains":
		if fieldValue == nil {
			return false
		}
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", condition.Value))
	case "regex":
		if fieldValue == nil {
			return false
		}
		matched, err := regexp.MatchString(fmt.Sprintf("%v", condition.Value), fmt.Sprintf("%v", fieldValue))
		return err == nil && matched
	case "gt":
		return ce.compareNumbers(fieldValue, condition.Value, ">")
	case "lt":
		return ce.compareNumbers(fieldValue, condition.Value, "<")
	case "gte":
		return ce.compareNumbers(fieldValue, condition.Value, ">=")
	case "lte":
		return ce.compareNumbers(fieldValue, condition.Value, "<=")
	default:
		return false
	}
}

// applyRuleActions 应用规则动作
func (ce *CleansingEngine) applyRuleActions(record map[string]interface{},
	actions []RuleAction, ruleID string) []ValidationError {

	var errors []ValidationError

	for _, action := range actions {
		switch action.Type {
		case "set":
			record[action.Field] = action.Value
		case "remove":
			delete(record, action.Field)
		case "transform":
			transformedValue, err := ce.transformer.Transform(record[action.Field], action.Transform, action.Config)
			if err != nil {
				errors = append(errors, ValidationError{
					Field:     action.Field,
					Value:     record[action.Field],
					RuleID:    ruleID,
					ErrorType: "transform_error",
					Message:   err.Error(),
					Severity:  "warning",
				})
			} else {
				record[action.Field] = transformedValue
			}
		case "validate":
			if err := ce.validator.ValidateField(record[action.Field], action.Config); err != nil {
				errors = append(errors, ValidationError{
					Field:     action.Field,
					Value:     record[action.Field],
					RuleID:    ruleID,
					ErrorType: "validation_error",
					Message:   err.Error(),
					Severity:  "error",
				})
			}
		}
	}

	return errors
}

// sortRulesByPriority 按优先级排序规则
func (ce *CleansingEngine) sortRulesByPriority(rules []CleansingRule) []CleansingRule {
	sorted := make([]CleansingRule, len(rules))
	copy(sorted, rules)

	// 简单的冒泡排序，按优先级降序
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Priority < sorted[j+1].Priority {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// compareNumbers 比较数值
func (ce *CleansingEngine) compareNumbers(value1, value2 interface{}, operator string) bool {
	num1 := ce.parseNumber(value1)
	num2 := ce.parseNumber(value2)

	if num1 == nil || num2 == nil {
		return false
	}

	switch operator {
	case ">":
		return *num1 > *num2
	case "<":
		return *num1 < *num2
	case ">=":
		return *num1 >= *num2
	case "<=":
		return *num1 <= *num2
	default:
		return false
	}
}

// parseNumber 解析数值
func (ce *CleansingEngine) parseNumber(value interface{}) *float64 {
	switch v := value.(type) {
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case float64:
		return &v
	case float32:
		f := float64(v)
		return &f
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}

// copyRecord 复制记录
func (ce *CleansingEngine) copyRecord(record map[string]interface{}) map[string]interface{} {
	copied := make(map[string]interface{})
	for key, value := range record {
		copied[key] = value
	}
	return copied
}

// RuleEngine 规则引擎
type RuleEngine struct{}

// NewRuleEngine 创建规则引擎
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{}
}

// DataValidator 数据验证器
type DataValidator struct{}

// NewDataValidator 创建数据验证器
func NewDataValidator() *DataValidator {
	return &DataValidator{}
}

// ValidateField 验证字段
func (dv *DataValidator) ValidateField(value interface{}, config map[string]interface{}) error {
	if value == nil {
		if required, ok := config["required"].(bool); ok && required {
			return fmt.Errorf("字段不能为空")
		}
		return nil
	}

	// 验证数据类型
	if expectedType, ok := config["type"].(string); ok {
		if !dv.validateType(value, expectedType) {
			return fmt.Errorf("数据类型不匹配，期望 %s", expectedType)
		}
	}

	// 验证长度
	if maxLength, ok := config["max_length"].(int); ok {
		if str, isStr := value.(string); isStr && len(str) > maxLength {
			return fmt.Errorf("字符串长度超过最大限制 %d", maxLength)
		}
	}

	// 验证正则表达式
	if pattern, ok := config["pattern"].(string); ok {
		if str, isStr := value.(string); isStr {
			matched, err := regexp.MatchString(pattern, str)
			if err != nil || !matched {
				return fmt.Errorf("字符串格式不匹配模式 %s", pattern)
			}
		}
	}

	return nil
}

// validateType 验证类型
func (dv *DataValidator) validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "int":
		_, ok := value.(int)
		return ok
	case "float":
		_, ok := value.(float64)
		return ok
	case "bool":
		_, ok := value.(bool)
		return ok
	default:
		return true
	}
}

// DataTransformer 数据转换器
type DataTransformer struct{}

// NewDataTransformer 创建数据转换器
func NewDataTransformer() *DataTransformer {
	return &DataTransformer{}
}

// Transform 转换数据
func (dt *DataTransformer) Transform(value interface{}, transform string,
	config map[string]interface{}) (interface{}, error) {

	if value == nil {
		return nil, nil
	}

	switch transform {
	case "trim":
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str), nil
		}
	case "upper":
		if str, ok := value.(string); ok {
			return strings.ToUpper(str), nil
		}
	case "lower":
		if str, ok := value.(string); ok {
			return strings.ToLower(str), nil
		}
	case "normalize_phone":
		if str, ok := value.(string); ok {
			return dt.normalizePhone(str), nil
		}
	case "normalize_email":
		if str, ok := value.(string); ok {
			return dt.normalizeEmail(str), nil
		}
	case "format_date":
		return dt.formatDate(value, config)
	case "round":
		return dt.roundNumber(value, config)
	default:
		return value, fmt.Errorf("未知转换函数: %s", transform)
	}

	return value, nil
}

// normalizePhone 标准化电话号码
func (dt *DataTransformer) normalizePhone(phone string) string {
	// 移除所有非数字字符
	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}

// normalizeEmail 标准化邮箱
func (dt *DataTransformer) normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// formatDate 格式化日期
func (dt *DataTransformer) formatDate(value interface{}, config map[string]interface{}) (interface{}, error) {
	format, ok := config["format"].(string)
	if !ok {
		format = "2006-01-02"
	}

	switch v := value.(type) {
	case time.Time:
		return v.Format(format), nil
	case string:
		// 尝试解析多种日期格式
		layouts := []string{
			"2006-01-02",
			"2006/01/02",
			"2006-01-02 15:04:05",
			"2006/01/02 15:04:05",
			time.RFC3339,
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, v); err == nil {
				return t.Format(format), nil
			}
		}
		return v, fmt.Errorf("无法解析日期格式: %s", v)
	default:
		return value, nil
	}
}

// roundNumber 四舍五入数值
func (dt *DataTransformer) roundNumber(value interface{}, config map[string]interface{}) (interface{}, error) {
	precision := 2
	if p, ok := config["precision"].(int); ok {
		precision = p
	}

	switch v := value.(type) {
	case float64:
		multiplier := 1.0
		for i := 0; i < precision; i++ {
			multiplier *= 10
		}
		return float64(int(v*multiplier+0.5)) / multiplier, nil
	case float32:
		multiplier := float32(1.0)
		for i := 0; i < precision; i++ {
			multiplier *= 10
		}
		return float32(int(v*multiplier+0.5)) / multiplier, nil
	default:
		return value, nil
	}
}

// QualityChecker 质量检查器
type QualityChecker struct{}

// NewQualityChecker 创建质量检查器
func NewQualityChecker() *QualityChecker {
	return &QualityChecker{}
}

// CalculateRecordQuality 计算记录质量评分
func (qc *QualityChecker) CalculateRecordQuality(record map[string]interface{},
	validationErrors []ValidationError) float64 {

	if len(record) == 0 {
		return 0.0
	}

	// 计算完整性得分
	nonNullFields := 0
	for _, value := range record {
		if value != nil && fmt.Sprintf("%v", value) != "" {
			nonNullFields++
		}
	}
	completenessScore := float64(nonNullFields) / float64(len(record))

	// 计算准确性得分（基于验证错误）
	errorCount := 0
	for _, err := range validationErrors {
		if err.Severity == "error" {
			errorCount += 2 // 错误权重更高
		} else if err.Severity == "warning" {
			errorCount += 1
		}
	}

	accuracyScore := 1.0
	if errorCount > 0 {
		accuracyScore = 1.0 / (1.0 + float64(errorCount)*0.1)
	}

	// 综合评分
	overallScore := (completenessScore*0.6 + accuracyScore*0.4) * 100

	if overallScore > 100 {
		overallScore = 100
	}
	if overallScore < 0 {
		overallScore = 0
	}

	return overallScore
}
