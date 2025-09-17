/*
 * @module service/thematic_sync/quality_dimension_checkers
 * @description 数据质量维度检查器实现，提供各个质量维度的具体检查逻辑
 * @architecture 策略模式 - 每个维度独立实现检查策略
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 维度检查配置 -> 数据分析 -> 规则应用 -> 结果计算 -> 问题识别
 * @rules 确保各维度检查的准确性和一致性，支持可配置的检查参数
 * @dependencies time, regexp, strconv, strings
 * @refs quality_checker.go, data_validator.go
 */

package thematic_sync

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CompletenessChecker 完整性检查器
type CompletenessChecker struct{}

func NewCompletenessChecker() *CompletenessChecker {
	return &CompletenessChecker{}
}

func (cc *CompletenessChecker) Check(records []map[string]interface{}, rule QualityRule) QualityRuleResult {
	if len(records) == 0 {
		return QualityRuleResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Dimension: rule.Dimension,
			Score:     0,
			Passed:    false,
			Message:   "没有数据记录",
		}
	}

	totalFields := int64(0)
	completedFields := int64(0)

	for _, record := range records {
		if len(rule.TargetFields) > 0 {
			// 检查指定字段
			for _, field := range rule.TargetFields {
				totalFields++
				if value, exists := record[field]; exists && !cc.isEmptyValue(value) {
					completedFields++
				}
			}
		} else {
			// 检查所有字段
			for _, value := range record {
				totalFields++
				if !cc.isEmptyValue(value) {
					completedFields++
				}
			}
		}
	}

	var score float64
	if totalFields > 0 {
		score = float64(completedFields) / float64(totalFields) * 100
	}

	passed := score >= rule.Threshold

	return QualityRuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Dimension:    rule.Dimension,
		Score:        score,
		Passed:       passed,
		CheckedCount: totalFields,
		PassedCount:  completedFields,
		FailedCount:  totalFields - completedFields,
		Message:      fmt.Sprintf("完整性得分: %.2f%%, 阈值: %.2f%%", score, rule.Threshold),
	}
}

func (cc *CompletenessChecker) isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}

	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	return str == "" || str == "null" || str == "NULL" || str == "nil"
}

// AccuracyChecker 准确性检查器
type AccuracyChecker struct{}

func NewAccuracyChecker() *AccuracyChecker {
	return &AccuracyChecker{}
}

func (ac *AccuracyChecker) Check(records []map[string]interface{}, rule QualityRule) QualityRuleResult {
	if len(records) == 0 {
		return QualityRuleResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Dimension: rule.Dimension,
			Score:     0,
			Passed:    false,
			Message:   "没有数据记录",
		}
	}

	totalChecked := int64(0)
	accurateCount := int64(0)

	checkType, _ := rule.Config["check_type"].(string)

	for _, record := range records {
		for _, field := range rule.TargetFields {
			value, exists := record[field]
			if !exists {
				continue
			}

			totalChecked++
			if ac.isAccurate(value, field, checkType, rule.Config) {
				accurateCount++
			}
		}
	}

	var score float64
	if totalChecked > 0 {
		score = float64(accurateCount) / float64(totalChecked) * 100
	}

	passed := score >= rule.Threshold

	return QualityRuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Dimension:    rule.Dimension,
		Score:        score,
		Passed:       passed,
		CheckedCount: totalChecked,
		PassedCount:  accurateCount,
		FailedCount:  totalChecked - accurateCount,
		Message:      fmt.Sprintf("准确性得分: %.2f%%, 阈值: %.2f%%", score, rule.Threshold),
	}
}

func (ac *AccuracyChecker) isAccurate(value interface{}, field, checkType string, config map[string]interface{}) bool {
	str := fmt.Sprintf("%v", value)

	switch checkType {
	case "email":
		return ac.isValidEmail(str)
	case "phone":
		return ac.isValidPhone(str)
	case "id_card":
		return ac.isValidIdCard(str)
	case "date":
		return ac.isValidDate(str)
	case "number":
		return ac.isValidNumber(str)
	case "range":
		return ac.isInRange(str, config)
	case "pattern":
		pattern, _ := config["pattern"].(string)
		return ac.matchesPattern(str, pattern)
	default:
		return true // 未知类型默认准确
	}
}

func (ac *AccuracyChecker) isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func (ac *AccuracyChecker) isValidPhone(phone string) bool {
	// 中国手机号验证
	pattern := `^1[3-9]\d{9}$`
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	matched, _ := regexp.MatchString(pattern, cleaned)
	return matched
}

func (ac *AccuracyChecker) isValidIdCard(idCard string) bool {
	// 18位身份证号验证
	pattern := `^[1-9]\d{5}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$`
	matched, _ := regexp.MatchString(pattern, idCard)
	return matched
}

func (ac *AccuracyChecker) isValidDate(dateStr string) bool {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, dateStr); err == nil {
			return true
		}
	}
	return false
}

func (ac *AccuracyChecker) isValidNumber(numStr string) bool {
	_, err := strconv.ParseFloat(numStr, 64)
	return err == nil
}

func (ac *AccuracyChecker) isInRange(numStr string, config map[string]interface{}) bool {
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return false
	}

	if minVal, exists := config["min"]; exists {
		if min, err := strconv.ParseFloat(fmt.Sprintf("%v", minVal), 64); err == nil {
			if num < min {
				return false
			}
		}
	}

	if maxVal, exists := config["max"]; exists {
		if max, err := strconv.ParseFloat(fmt.Sprintf("%v", maxVal), 64); err == nil {
			if num > max {
				return false
			}
		}
	}

	return true
}

func (ac *AccuracyChecker) matchesPattern(str, pattern string) bool {
	if pattern == "" {
		return true
	}
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

// ConsistencyChecker 一致性检查器
type ConsistencyChecker struct{}

func NewConsistencyChecker() *ConsistencyChecker {
	return &ConsistencyChecker{}
}

func (cc *ConsistencyChecker) Check(records []map[string]interface{}, rule QualityRule) QualityRuleResult {
	if len(records) == 0 {
		return QualityRuleResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Dimension: rule.Dimension,
			Score:     0,
			Passed:    false,
			Message:   "没有数据记录",
		}
	}

	checkType, _ := rule.Config["check_type"].(string)
	var score float64

	switch checkType {
	case "format":
		score = cc.checkFormatConsistency(records, rule.TargetFields)
	case "type":
		score = cc.checkTypeConsistency(records, rule.TargetFields)
	case "domain":
		score = cc.checkDomainConsistency(records, rule.TargetFields, rule.Config)
	default:
		score = cc.checkGeneralConsistency(records, rule.TargetFields)
	}

	passed := score >= rule.Threshold
	totalChecked := int64(len(records) * len(rule.TargetFields))
	passedCount := int64(float64(totalChecked) * score / 100)

	return QualityRuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Dimension:    rule.Dimension,
		Score:        score,
		Passed:       passed,
		CheckedCount: totalChecked,
		PassedCount:  passedCount,
		FailedCount:  totalChecked - passedCount,
		Message:      fmt.Sprintf("一致性得分: %.2f%%, 阈值: %.2f%%", score, rule.Threshold),
	}
}

func (cc *ConsistencyChecker) checkFormatConsistency(records []map[string]interface{}, fields []string) float64 {
	if len(records) == 0 || len(fields) == 0 {
		return 0
	}

	totalScore := 0.0
	for _, field := range fields {
		formats := make(map[string]int)
		totalValues := 0

		for _, record := range records {
			if value, exists := record[field]; exists && value != nil {
				format := cc.detectValueFormat(fmt.Sprintf("%v", value))
				formats[format]++
				totalValues++
			}
		}

		if totalValues == 0 {
			continue
		}

		// 计算最常见格式的占比
		maxCount := 0
		for _, count := range formats {
			if count > maxCount {
				maxCount = count
			}
		}

		fieldScore := float64(maxCount) / float64(totalValues) * 100
		totalScore += fieldScore
	}

	return totalScore / float64(len(fields))
}

func (cc *ConsistencyChecker) checkTypeConsistency(records []map[string]interface{}, fields []string) float64 {
	if len(records) == 0 || len(fields) == 0 {
		return 0
	}

	totalScore := 0.0
	for _, field := range fields {
		types := make(map[string]int)
		totalValues := 0

		for _, record := range records {
			if value, exists := record[field]; exists && value != nil {
				valueType := cc.detectValueType(value)
				types[valueType]++
				totalValues++
			}
		}

		if totalValues == 0 {
			continue
		}

		// 计算最常见类型的占比
		maxCount := 0
		for _, count := range types {
			if count > maxCount {
				maxCount = count
			}
		}

		fieldScore := float64(maxCount) / float64(totalValues) * 100
		totalScore += fieldScore
	}

	return totalScore / float64(len(fields))
}

func (cc *ConsistencyChecker) checkDomainConsistency(records []map[string]interface{}, fields []string, config map[string]interface{}) float64 {
	allowedValues, exists := config["allowed_values"].([]interface{})
	if !exists {
		return 100 // 没有域值限制，认为一致
	}

	allowedSet := make(map[string]bool)
	for _, val := range allowedValues {
		allowedSet[fmt.Sprintf("%v", val)] = true
	}

	totalChecked := 0
	validCount := 0

	for _, record := range records {
		for _, field := range fields {
			if value, exists := record[field]; exists && value != nil {
				totalChecked++
				if allowedSet[fmt.Sprintf("%v", value)] {
					validCount++
				}
			}
		}
	}

	if totalChecked == 0 {
		return 100
	}

	return float64(validCount) / float64(totalChecked) * 100
}

func (cc *ConsistencyChecker) checkGeneralConsistency(records []map[string]interface{}, fields []string) float64 {
	// 综合检查格式和类型一致性
	formatScore := cc.checkFormatConsistency(records, fields)
	typeScore := cc.checkTypeConsistency(records, fields)
	return (formatScore + typeScore) / 2
}

func (cc *ConsistencyChecker) detectValueFormat(value string) string {
	patterns := map[string]string{
		"date_yyyy_mm_dd":     `^\d{4}-\d{2}-\d{2}$`,
		"date_yyyy_mm_dd_hms": `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`,
		"phone_11_digits":     `^1\d{10}$`,
		"email":               `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		"number_integer":      `^-?\d+$`,
		"number_decimal":      `^-?\d+\.\d+$`,
	}

	for format, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, value); matched {
			return format
		}
	}

	return "text"
}

func (cc *ConsistencyChecker) detectValueType(value interface{}) string {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return "integer"
	case uint, uint8, uint16, uint32, uint64:
		return "unsigned_integer"
	case float32, float64:
		return "float"
	case bool:
		return "boolean"
	case time.Time:
		return "datetime"
	case string:
		if _, err := strconv.Atoi(v); err == nil {
			return "string_integer"
		}
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return "string_float"
		}
		return "string"
	default:
		return "unknown"
	}
}

// ValidityChecker 有效性检查器
type ValidityChecker struct {
	validator *DataValidator
}

func NewValidityChecker() *ValidityChecker {
	return &ValidityChecker{
		validator: NewDataValidator(),
	}
}

func (vc *ValidityChecker) Check(records []map[string]interface{}, rule QualityRule) QualityRuleResult {
	if len(records) == 0 {
		return QualityRuleResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Dimension: rule.Dimension,
			Score:     0,
			Passed:    false,
			Message:   "没有数据记录",
		}
	}

	totalChecked := int64(0)
	validCount := int64(0)

	for _, record := range records {
		for _, field := range rule.TargetFields {
			if value, exists := record[field]; exists {
				totalChecked++
				if vc.isValid(value, field, rule.Config) {
					validCount++
				}
			}
		}
	}

	var score float64
	if totalChecked > 0 {
		score = float64(validCount) / float64(totalChecked) * 100
	}

	passed := score >= rule.Threshold

	return QualityRuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Dimension:    rule.Dimension,
		Score:        score,
		Passed:       passed,
		CheckedCount: totalChecked,
		PassedCount:  validCount,
		FailedCount:  totalChecked - validCount,
		Message:      fmt.Sprintf("有效性得分: %.2f%%, 阈值: %.2f%%", score, rule.Threshold),
	}
}

func (vc *ValidityChecker) isValid(value interface{}, field string, config map[string]interface{}) bool {
	// 使用数据验证器进行验证
	err := vc.validator.ValidateField(value, config)
	return err == nil
}

// UniquenessChecker 唯一性检查器
type UniquenessChecker struct{}

func NewUniquenessChecker() *UniquenessChecker {
	return &UniquenessChecker{}
}

func (uc *UniquenessChecker) Check(records []map[string]interface{}, rule QualityRule) QualityRuleResult {
	if len(records) == 0 {
		return QualityRuleResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Dimension: rule.Dimension,
			Score:     0,
			Passed:    false,
			Message:   "没有数据记录",
		}
	}

	totalRecords := int64(len(records))
	uniqueRecords := int64(0)
	duplicateCount := int64(0)

	if len(rule.TargetFields) == 1 {
		// 单字段唯一性检查
		uniqueRecords, duplicateCount = uc.checkSingleFieldUniqueness(records, rule.TargetFields[0])
	} else {
		// 多字段组合唯一性检查
		uniqueRecords, duplicateCount = uc.checkMultiFieldUniqueness(records, rule.TargetFields)
	}

	var score float64
	if totalRecords > 0 {
		score = float64(uniqueRecords) / float64(totalRecords) * 100
	}

	passed := score >= rule.Threshold

	return QualityRuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Dimension:    rule.Dimension,
		Score:        score,
		Passed:       passed,
		CheckedCount: totalRecords,
		PassedCount:  uniqueRecords,
		FailedCount:  duplicateCount,
		Message:      fmt.Sprintf("唯一性得分: %.2f%%, 重复记录: %d, 阈值: %.2f%%", score, duplicateCount, rule.Threshold),
	}
}

func (uc *UniquenessChecker) checkSingleFieldUniqueness(records []map[string]interface{}, field string) (int64, int64) {
	valueMap := make(map[string]int)
	totalCount := 0

	for _, record := range records {
		if value, exists := record[field]; exists && value != nil {
			key := fmt.Sprintf("%v", value)
			valueMap[key]++
			totalCount++
		}
	}

	uniqueCount := 0
	duplicateCount := 0

	for _, count := range valueMap {
		if count == 1 {
			uniqueCount++
		} else {
			duplicateCount += count - 1 // 重复的数量
		}
	}

	return int64(totalCount - duplicateCount), int64(duplicateCount)
}

func (uc *UniquenessChecker) checkMultiFieldUniqueness(records []map[string]interface{}, fields []string) (int64, int64) {
	combinationMap := make(map[string]int)
	totalCount := 0

	for _, record := range records {
		keyParts := make([]string, 0, len(fields))
		allExists := true

		for _, field := range fields {
			if value, exists := record[field]; exists && value != nil {
				keyParts = append(keyParts, fmt.Sprintf("%v", value))
			} else {
				allExists = false
				break
			}
		}

		if allExists {
			key := strings.Join(keyParts, "|")
			combinationMap[key]++
			totalCount++
		}
	}

	uniqueCount := 0
	duplicateCount := 0

	for _, count := range combinationMap {
		if count == 1 {
			uniqueCount++
		} else {
			duplicateCount += count - 1
		}
	}

	return int64(totalCount - duplicateCount), int64(duplicateCount)
}

// TimelinessChecker 时效性检查器
type TimelinessChecker struct{}

func NewTimelinessChecker() *TimelinessChecker {
	return &TimelinessChecker{}
}

func (tc *TimelinessChecker) Check(records []map[string]interface{}, rule QualityRule) QualityRuleResult {
	if len(records) == 0 {
		return QualityRuleResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Dimension: rule.Dimension,
			Score:     0,
			Passed:    false,
			Message:   "没有数据记录",
		}
	}

	totalChecked := int64(0)
	timelyCount := int64(0)

	maxAge, _ := rule.Config["max_age_hours"].(float64)
	if maxAge == 0 {
		maxAge = 24 // 默认24小时
	}

	cutoffTime := time.Now().Add(-time.Duration(maxAge) * time.Hour)

	for _, record := range records {
		for _, field := range rule.TargetFields {
			if value, exists := record[field]; exists && value != nil {
				totalChecked++
				if tc.isTimely(value, cutoffTime) {
					timelyCount++
				}
			}
		}
	}

	var score float64
	if totalChecked > 0 {
		score = float64(timelyCount) / float64(totalChecked) * 100
	}

	passed := score >= rule.Threshold

	return QualityRuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Dimension:    rule.Dimension,
		Score:        score,
		Passed:       passed,
		CheckedCount: totalChecked,
		PassedCount:  timelyCount,
		FailedCount:  totalChecked - timelyCount,
		Message:      fmt.Sprintf("时效性得分: %.2f%%, 阈值: %.2f%%, 时效窗口: %.0f小时", score, rule.Threshold, maxAge),
	}
}

func (tc *TimelinessChecker) isTimely(value interface{}, cutoffTime time.Time) bool {
	str := fmt.Sprintf("%v", value)

	// 尝试解析时间
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, str); err == nil {
			return t.After(cutoffTime)
		}
	}

	return false // 无法解析时间，认为不及时
}
