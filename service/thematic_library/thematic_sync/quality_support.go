/*
 * @module service/thematic_sync/quality_support
 * @description 数据质量检查的支持类，包括统计计算器、问题检测器和建议引擎
 * @architecture 工具类模式 - 提供质量检查的辅助功能和算法支持
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 数据收集 -> 统计分析 -> 问题识别 -> 建议生成 -> 报告输出
 * @rules 确保统计分析的准确性和建议的实用性
 * @dependencies time, sort, math
 * @refs quality_checker.go, quality_dimension_checkers.go
 */

package thematic_sync

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// StatisticsCalculator 统计计算器
type StatisticsCalculator struct{}

func NewStatisticsCalculator() *StatisticsCalculator {
	return &StatisticsCalculator{}
}

// Calculate 计算质量统计信息
func (sc *StatisticsCalculator) Calculate(records []map[string]interface{}, issues []QualityIssue) QualityStatistics {
	totalRecords := int64(len(records))
	if totalRecords == 0 {
		return QualityStatistics{}
	}

	// 计算基础统计
	validRecords := totalRecords
	invalidRecords := int64(0)
	nullValueCount := int64(0)
	totalFields := int64(0)
	completedFields := int64(0)

	// 收集所有字段名
	allFields := make(map[string]bool)
	for _, record := range records {
		for field := range record {
			allFields[field] = true
		}
	}

	// 计算字段完整性
	for _, record := range records {
		for field := range allFields {
			totalFields++
			if value, exists := record[field]; exists && !sc.isNullValue(value) {
				completedFields++
			} else {
				nullValueCount++
			}
		}
	}

	// 检查重复记录
	duplicateCount := sc.calculateDuplicates(records)

	// 统计问题类型
	issuesByType := make(map[string]int64)
	issuesBySeverity := make(map[string]int64)

	for _, issue := range issues {
		issuesByType[issue.Dimension.String()]++
		issuesBySeverity[issue.Severity]++
	}

	// 计算无效记录数（有问题的记录）
	recordsWithIssues := make(map[string]bool)
	for _, issue := range issues {
		if issue.RecordID != "" {
			recordsWithIssues[issue.RecordID] = true
		}
	}
	invalidRecords = int64(len(recordsWithIssues))
	validRecords = totalRecords - invalidRecords

	return QualityStatistics{
		TotalRecords:     totalRecords,
		ValidRecords:     validRecords,
		InvalidRecords:   invalidRecords,
		CompletedFields:  completedFields,
		TotalFields:      totalFields,
		DuplicateCount:   duplicateCount,
		NullValueCount:   nullValueCount,
		IssuesByType:     issuesByType,
		IssuesBySeverity: issuesBySeverity,
	}
}

func (sc *StatisticsCalculator) isNullValue(value interface{}) bool {
	if value == nil {
		return true
	}
	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	return str == "" || str == "null" || str == "NULL" || str == "nil"
}

func (sc *StatisticsCalculator) calculateDuplicates(records []map[string]interface{}) int64 {
	if len(records) <= 1 {
		return 0
	}

	// 简化的重复检测：基于所有字段的组合
	recordHashes := make(map[string]int)

	for _, record := range records {
		hash := sc.generateRecordHash(record)
		recordHashes[hash]++
	}

	duplicates := int64(0)
	for _, count := range recordHashes {
		if count > 1 {
			duplicates += int64(count - 1) // 重复的数量
		}
	}

	return duplicates
}

func (sc *StatisticsCalculator) generateRecordHash(record map[string]interface{}) string {
	// 按字段名排序生成一致的哈希
	var keys []string
	for k := range record {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", key, record[key]))
	}

	return strings.Join(parts, "|")
}

// IssueDetector 问题检测器
type IssueDetector struct{}

func NewIssueDetector() *IssueDetector {
	return &IssueDetector{}
}

// DetectIssues 检测质量问题
func (id *IssueDetector) DetectIssues(records []map[string]interface{}, rule QualityRule, result QualityRuleResult) []QualityIssue {
	var issues []QualityIssue

	if result.Passed {
		return issues // 规则通过，无问题
	}

	// 根据不同维度检测具体问题
	switch rule.Dimension {
	case CompletenessQuality:
		issues = id.detectCompletenessIssues(records, rule)
	case AccuracyQuality:
		issues = id.detectAccuracyIssues(records, rule)
	case ConsistencyQuality:
		issues = id.detectConsistencyIssues(records, rule)
	case ValidityQuality:
		issues = id.detectValidityIssues(records, rule)
	case UniquenessQuality:
		issues = id.detectUniquenessIssues(records, rule)
	case TimelinessQuality:
		issues = id.detectTimelinessIssues(records, rule)
	}

	return issues
}

func (id *IssueDetector) detectCompletenessIssues(records []map[string]interface{}, rule QualityRule) []QualityIssue {
	var issues []QualityIssue

	for i, record := range records {
		recordID := fmt.Sprintf("record_%d", i)

		for _, field := range rule.TargetFields {
			value, exists := record[field]
			if !exists || id.isEmptyValue(value) {
				issues = append(issues, QualityIssue{
					ID:         fmt.Sprintf("%s_%s_%d", rule.ID, field, i),
					RuleID:     rule.ID,
					Dimension:  rule.Dimension,
					Severity:   rule.Severity,
					Field:      field,
					Value:      value,
					RecordID:   recordID,
					Message:    fmt.Sprintf("字段 %s 值为空或缺失", field),
					Suggestion: fmt.Sprintf("为字段 %s 提供有效值", field),
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return issues
}

func (id *IssueDetector) detectAccuracyIssues(records []map[string]interface{}, rule QualityRule) []QualityIssue {
	var issues []QualityIssue
	checkType, _ := rule.Config["check_type"].(string)

	for i, record := range records {
		recordID := fmt.Sprintf("record_%d", i)

		for _, field := range rule.TargetFields {
			value, exists := record[field]
			if !exists {
				continue
			}

			if !id.isAccurateValue(value, checkType, rule.Config) {
				issues = append(issues, QualityIssue{
					ID:         fmt.Sprintf("%s_%s_%d", rule.ID, field, i),
					RuleID:     rule.ID,
					Dimension:  rule.Dimension,
					Severity:   rule.Severity,
					Field:      field,
					Value:      value,
					RecordID:   recordID,
					Message:    fmt.Sprintf("字段 %s 值 %v 不符合 %s 格式要求", field, value, checkType),
					Suggestion: id.getAccuracySuggestion(checkType),
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return issues
}

func (id *IssueDetector) detectConsistencyIssues(records []map[string]interface{}, rule QualityRule) []QualityIssue {
	var issues []QualityIssue

	// 检测格式不一致的问题
	for _, field := range rule.TargetFields {
		formats := make(map[string][]int) // format -> record indices

		for i, record := range records {
			if value, exists := record[field]; exists && value != nil {
				format := id.detectValueFormat(fmt.Sprintf("%v", value))
				formats[format] = append(formats[format], i)
			}
		}

		// 如果有多种格式，找出少数格式的记录
		if len(formats) > 1 {
			maxCount := 0
			majorFormat := ""
			for format, indices := range formats {
				if len(indices) > maxCount {
					maxCount = len(indices)
					majorFormat = format
				}
			}

			// 标记非主要格式的记录为问题
			for format, indices := range formats {
				if format != majorFormat {
					for _, i := range indices {
						issues = append(issues, QualityIssue{
							ID:         fmt.Sprintf("%s_%s_%d", rule.ID, field, i),
							RuleID:     rule.ID,
							Dimension:  rule.Dimension,
							Severity:   rule.Severity,
							Field:      field,
							Value:      records[i][field],
							RecordID:   fmt.Sprintf("record_%d", i),
							Message:    fmt.Sprintf("字段 %s 格式 %s 与主要格式 %s 不一致", field, format, majorFormat),
							Suggestion: fmt.Sprintf("将字段 %s 格式统一为 %s", field, majorFormat),
							DetectedAt: time.Now(),
						})
					}
				}
			}
		}
	}

	return issues
}

func (id *IssueDetector) detectValidityIssues(records []map[string]interface{}, rule QualityRule) []QualityIssue {
	var issues []QualityIssue
	validator := NewDataValidator()

	for i, record := range records {
		recordID := fmt.Sprintf("record_%d", i)

		for _, field := range rule.TargetFields {
			value, exists := record[field]
			if !exists {
				continue
			}

			if err := validator.ValidateField(value, rule.Config); err != nil {
				issues = append(issues, QualityIssue{
					ID:         fmt.Sprintf("%s_%s_%d", rule.ID, field, i),
					RuleID:     rule.ID,
					Dimension:  rule.Dimension,
					Severity:   rule.Severity,
					Field:      field,
					Value:      value,
					RecordID:   recordID,
					Message:    fmt.Sprintf("字段 %s 验证失败: %s", field, err.Error()),
					Suggestion: "请检查数据格式和内容是否符合要求",
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return issues
}

func (id *IssueDetector) detectUniquenessIssues(records []map[string]interface{}, rule QualityRule) []QualityIssue {
	var issues []QualityIssue

	if len(rule.TargetFields) == 1 {
		// 单字段唯一性
		field := rule.TargetFields[0]
		valueIndices := make(map[string][]int)

		for i, record := range records {
			if value, exists := record[field]; exists && value != nil {
				key := fmt.Sprintf("%v", value)
				valueIndices[key] = append(valueIndices[key], i)
			}
		}

		for value, indices := range valueIndices {
			if len(indices) > 1 {
				for _, i := range indices {
					issues = append(issues, QualityIssue{
						ID:         fmt.Sprintf("%s_%s_%d", rule.ID, field, i),
						RuleID:     rule.ID,
						Dimension:  rule.Dimension,
						Severity:   rule.Severity,
						Field:      field,
						Value:      value,
						RecordID:   fmt.Sprintf("record_%d", i),
						Message:    fmt.Sprintf("字段 %s 值 %s 存在重复", field, value),
						Suggestion: "确保字段值的唯一性",
						DetectedAt: time.Now(),
					})
				}
			}
		}
	}

	return issues
}

func (id *IssueDetector) detectTimelinessIssues(records []map[string]interface{}, rule QualityRule) []QualityIssue {
	var issues []QualityIssue
	maxAge, _ := rule.Config["max_age_hours"].(float64)
	if maxAge == 0 {
		maxAge = 24
	}
	cutoffTime := time.Now().Add(-time.Duration(maxAge) * time.Hour)

	for i, record := range records {
		recordID := fmt.Sprintf("record_%d", i)

		for _, field := range rule.TargetFields {
			value, exists := record[field]
			if !exists {
				continue
			}

			if !id.isTimelyValue(value, cutoffTime) {
				issues = append(issues, QualityIssue{
					ID:         fmt.Sprintf("%s_%s_%d", rule.ID, field, i),
					RuleID:     rule.ID,
					Dimension:  rule.Dimension,
					Severity:   rule.Severity,
					Field:      field,
					Value:      value,
					RecordID:   recordID,
					Message:    fmt.Sprintf("字段 %s 时间 %v 超出时效窗口 %.0f 小时", field, value, maxAge),
					Suggestion: "更新数据以确保时效性",
					DetectedAt: time.Now(),
				})
			}
		}
	}

	return issues
}

// 辅助方法
func (id *IssueDetector) isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	return str == "" || str == "null" || str == "NULL" || str == "nil"
}

func (id *IssueDetector) isAccurateValue(value interface{}, checkType string, config map[string]interface{}) bool {
	checker := NewAccuracyChecker()
	return checker.isAccurate(value, "", checkType, config)
}

func (id *IssueDetector) detectValueFormat(value string) string {
	checker := NewConsistencyChecker()
	return checker.detectValueFormat(value)
}

func (id *IssueDetector) isTimelyValue(value interface{}, cutoffTime time.Time) bool {
	checker := NewTimelinessChecker()
	return checker.isTimely(value, cutoffTime)
}

func (id *IssueDetector) getAccuracySuggestion(checkType string) string {
	suggestions := map[string]string{
		"email":   "请提供有效的邮箱地址格式，如：user@domain.com",
		"phone":   "请提供有效的手机号码格式，如：13812345678",
		"id_card": "请提供有效的18位身份证号码",
		"date":    "请提供有效的日期格式，如：2023-01-01",
		"number":  "请提供有效的数字格式",
		"range":   "请确保数值在指定范围内",
		"pattern": "请确保数据符合指定的模式",
	}

	if suggestion, exists := suggestions[checkType]; exists {
		return suggestion
	}
	return "请检查数据格式是否正确"
}

// RecommendationEngine 建议引擎
type RecommendationEngine struct{}

func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{}
}

// GenerateRecommendations 生成质量改善建议
func (re *RecommendationEngine) GenerateRecommendations(
	ruleResults []QualityRuleResult,
	issues []QualityIssue,
	statistics QualityStatistics) []QualityRecommendation {

	var recommendations []QualityRecommendation

	// 基于规则结果生成建议
	for _, result := range ruleResults {
		if !result.Passed {
			recs := re.generateRuleRecommendations(result)
			recommendations = append(recommendations, recs...)
		}
	}

	// 基于统计信息生成建议
	statRecs := re.generateStatisticsRecommendations(statistics)
	recommendations = append(recommendations, statRecs...)

	// 基于问题严重程度生成建议
	issueRecs := re.generateIssueRecommendations(issues)
	recommendations = append(recommendations, issueRecs...)

	// 去重和排序
	recommendations = re.deduplicateAndSort(recommendations)

	return recommendations
}

func (re *RecommendationEngine) generateRuleRecommendations(result QualityRuleResult) []QualityRecommendation {
	var recommendations []QualityRecommendation

	switch result.Dimension {
	case CompletenessQuality:
		if result.Score < 80 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "cleansing",
				Priority:    "high",
				Description: "数据完整性较低，建议进行数据补全",
				Action:      "添加数据验证规则，要求必填字段不能为空",
				Impact:      math.Min(100-result.Score, 20),
			})
		}
	case AccuracyQuality:
		if result.Score < 90 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "validation",
				Priority:    "high",
				Description: "数据准确性需要改善",
				Action:      "增强数据验证规则，添加格式检查",
				Impact:      math.Min(100-result.Score, 15),
			})
		}
	case ConsistencyQuality:
		if result.Score < 85 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "transformation",
				Priority:    "medium",
				Description: "数据一致性存在问题",
				Action:      "标准化数据格式，统一编码规则",
				Impact:      math.Min(100-result.Score, 18),
			})
		}
	case ValidityQuality:
		if result.Score < 95 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "validation",
				Priority:    "high",
				Description: "数据有效性需要提升",
				Action:      "加强业务规则验证，过滤无效数据",
				Impact:      math.Min(100-result.Score, 12),
			})
		}
	case UniquenessQuality:
		if result.Score < 98 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "cleansing",
				Priority:    "medium",
				Description: "存在重复数据",
				Action:      "实施去重策略，建立唯一性约束",
				Impact:      math.Min(100-result.Score, 10),
			})
		}
	case TimelinessQuality:
		if result.Score < 90 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "cleansing",
				Priority:    "low",
				Description: "数据时效性需要改善",
				Action:      "建立数据更新机制，定期刷新过期数据",
				Impact:      math.Min(100-result.Score, 8),
			})
		}
	}

	return recommendations
}

func (re *RecommendationEngine) generateStatisticsRecommendations(statistics QualityStatistics) []QualityRecommendation {
	var recommendations []QualityRecommendation

	// 基于空值比例
	if statistics.TotalFields > 0 {
		nullRate := float64(statistics.NullValueCount) / float64(statistics.TotalFields) * 100
		if nullRate > 20 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "cleansing",
				Priority:    "high",
				Description: fmt.Sprintf("空值比例过高 (%.1f%%)", nullRate),
				Action:      "实施数据补全策略，设置默认值或必填验证",
				Impact:      math.Min(nullRate/2, 15),
			})
		}
	}

	// 基于重复记录比例
	if statistics.TotalRecords > 0 {
		duplicateRate := float64(statistics.DuplicateCount) / float64(statistics.TotalRecords) * 100
		if duplicateRate > 5 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "cleansing",
				Priority:    "medium",
				Description: fmt.Sprintf("重复记录比例过高 (%.1f%%)", duplicateRate),
				Action:      "建立数据去重流程，设置唯一性检查",
				Impact:      math.Min(duplicateRate/3, 10),
			})
		}
	}

	// 基于无效记录比例
	if statistics.TotalRecords > 0 {
		invalidRate := float64(statistics.InvalidRecords) / float64(statistics.TotalRecords) * 100
		if invalidRate > 10 {
			recommendations = append(recommendations, QualityRecommendation{
				Type:        "validation",
				Priority:    "high",
				Description: fmt.Sprintf("无效记录比例过高 (%.1f%%)", invalidRate),
				Action:      "加强数据验证，建立数据质量监控",
				Impact:      math.Min(invalidRate/2, 20),
			})
		}
	}

	return recommendations
}

func (re *RecommendationEngine) generateIssueRecommendations(issues []QualityIssue) []QualityRecommendation {
	var recommendations []QualityRecommendation

	// 按严重程度统计问题
	severityCount := make(map[string]int)
	for _, issue := range issues {
		severityCount[issue.Severity]++
	}

	if severityCount["critical"] > 0 {
		recommendations = append(recommendations, QualityRecommendation{
			Type:        "validation",
			Priority:    "high",
			Description: fmt.Sprintf("发现 %d 个严重质量问题", severityCount["critical"]),
			Action:      "立即处理严重问题，建立紧急修复流程",
			Impact:      25,
		})
	}

	if severityCount["major"] > 10 {
		recommendations = append(recommendations, QualityRecommendation{
			Type:        "cleansing",
			Priority:    "medium",
			Description: fmt.Sprintf("发现 %d 个主要质量问题", severityCount["major"]),
			Action:      "制定批量处理计划，优先解决影响大的问题",
			Impact:      15,
		})
	}

	return recommendations
}

func (re *RecommendationEngine) deduplicateAndSort(recommendations []QualityRecommendation) []QualityRecommendation {
	// 简单去重：基于描述
	seen := make(map[string]bool)
	var unique []QualityRecommendation

	for _, rec := range recommendations {
		if !seen[rec.Description] {
			seen[rec.Description] = true
			unique = append(unique, rec)
		}
	}

	// 按影响程度排序
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].Priority != unique[j].Priority {
			priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
			return priorityOrder[unique[i].Priority] > priorityOrder[unique[j].Priority]
		}
		return unique[i].Impact > unique[j].Impact
	})

	return unique
}

// String 方法实现
func (qd QualityDimension) String() string {
	return string(qd)
}
