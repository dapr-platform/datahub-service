/*
 * @module service/thematic_sync/key_matcher
 * @description 主键匹配器，负责识别和匹配来自不同源的记录主键
 * @architecture 策略模式 - 支持多种匹配策略的主键识别
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 匹配规则加载 -> 记录预处理 -> 策略匹配 -> 结果验证 -> 冲突解决
 * @rules 确保主键匹配的准确性和一致性，支持模糊匹配和规则匹配
 * @dependencies crypto/md5, strings, strconv
 * @refs conflict_resolver.go, aggregation_engine.go
 */

package thematic_sync

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// KeyMatchingStrategy 主键匹配策略
type KeyMatchingStrategy string

const (
	ExactMatch     KeyMatchingStrategy = "exact"      // 精确匹配
	FuzzyMatch     KeyMatchingStrategy = "fuzzy"      // 模糊匹配
	RuleBasedMatch KeyMatchingStrategy = "rule_based" // 基于规则匹配
	MLMatch        KeyMatchingStrategy = "ml_based"   // 机器学习匹配
)

// KeyMatchingRule 主键匹配规则
type KeyMatchingRule struct {
	Strategy       KeyMatchingStrategy `json:"strategy"`
	MatchFields    []string            `json:"match_fields"`    // 用于匹配的字段
	WeightConfig   map[string]float64  `json:"weight_config"`   // 字段权重配置
	ThresholdScore float64             `json:"threshold_score"` // 匹配阈值
	ConflictPolicy string              `json:"conflict_policy"` // 冲突处理策略
}

// MatchResult 匹配结果
type MatchResult struct {
	SourceRecordID string                 `json:"source_record_id"`
	TargetRecordID string                 `json:"target_record_id"`
	MatchScore     float64                `json:"match_score"`
	MatchedFields  []string               `json:"matched_fields"`
	IsExactMatch   bool                   `json:"is_exact_match"`
	ConflictFields []string               `json:"conflict_fields"`
	SourceRecord   map[string]interface{} `json:"source_record"`
	TargetRecord   map[string]interface{} `json:"target_record"`
}

// KeyMatcher 主键匹配器
type KeyMatcher struct {
	rules        []KeyMatchingRule
	fuzzyMatcher *FuzzyMatcher
	ruleMatcher  *RuleMatcher
}

// NewKeyMatcher 创建主键匹配器
func NewKeyMatcher(rules []KeyMatchingRule) *KeyMatcher {
	return &KeyMatcher{
		rules:        rules,
		fuzzyMatcher: NewFuzzyMatcher(),
		ruleMatcher:  NewRuleMatcher(),
	}
}

// MatchRecords 匹配源记录和目标记录
func (km *KeyMatcher) MatchRecords(sourceRecords []map[string]interface{},
	targetRecords []map[string]interface{}) ([]MatchResult, error) {

	var results []MatchResult

	for _, sourceRecord := range sourceRecords {
		sourceID := km.generateRecordID(sourceRecord)

		for _, targetRecord := range targetRecords {
			targetID := km.generateRecordID(targetRecord)

			// 应用每个匹配规则
			for _, rule := range km.rules {
				matchScore, matchedFields, conflictFields := km.applyMatchingRule(
					sourceRecord, targetRecord, rule)

				if matchScore >= rule.ThresholdScore {
					result := MatchResult{
						SourceRecordID: sourceID,
						TargetRecordID: targetID,
						MatchScore:     matchScore,
						MatchedFields:  matchedFields,
						IsExactMatch:   matchScore == 1.0,
						ConflictFields: conflictFields,
						SourceRecord:   sourceRecord,
						TargetRecord:   targetRecord,
					}
					results = append(results, result)
					break // 找到匹配就跳出规则循环
				}
			}
		}
	}

	return results, nil
}

// applyMatchingRule 应用匹配规则
func (km *KeyMatcher) applyMatchingRule(sourceRecord, targetRecord map[string]interface{},
	rule KeyMatchingRule) (float64, []string, []string) {

	switch rule.Strategy {
	case ExactMatch:
		return km.exactMatch(sourceRecord, targetRecord, rule)
	case FuzzyMatch:
		return km.fuzzyMatch(sourceRecord, targetRecord, rule)
	case RuleBasedMatch:
		return km.ruleBasedMatch(sourceRecord, targetRecord, rule)
	default:
		return 0.0, nil, nil
	}
}

// exactMatch 精确匹配
func (km *KeyMatcher) exactMatch(sourceRecord, targetRecord map[string]interface{},
	rule KeyMatchingRule) (float64, []string, []string) {

	var matchedFields []string
	var conflictFields []string
	var totalWeight float64
	var matchedWeight float64

	for _, field := range rule.MatchFields {
		weight := rule.WeightConfig[field]
		if weight == 0 {
			weight = 1.0 // 默认权重
		}
		totalWeight += weight

		sourceValue := km.normalizeValue(sourceRecord[field])
		targetValue := km.normalizeValue(targetRecord[field])

		if sourceValue == targetValue {
			matchedFields = append(matchedFields, field)
			matchedWeight += weight
		} else {
			conflictFields = append(conflictFields, field)
		}
	}

	var score float64
	if totalWeight > 0 {
		score = matchedWeight / totalWeight
	}

	return score, matchedFields, conflictFields
}

// fuzzyMatch 模糊匹配
func (km *KeyMatcher) fuzzyMatch(sourceRecord, targetRecord map[string]interface{},
	rule KeyMatchingRule) (float64, []string, []string) {

	var matchedFields []string
	var conflictFields []string
	var totalWeight float64
	var matchedWeight float64

	for _, field := range rule.MatchFields {
		weight := rule.WeightConfig[field]
		if weight == 0 {
			weight = 1.0
		}
		totalWeight += weight

		sourceValue := km.normalizeValue(sourceRecord[field])
		targetValue := km.normalizeValue(targetRecord[field])

		similarity := km.fuzzyMatcher.CalculateSimilarity(sourceValue, targetValue)

		if similarity >= 0.8 { // 模糊匹配阈值
			matchedFields = append(matchedFields, field)
			matchedWeight += weight * similarity
		} else {
			conflictFields = append(conflictFields, field)
		}
	}

	var score float64
	if totalWeight > 0 {
		score = matchedWeight / totalWeight
	}

	return score, matchedFields, conflictFields
}

// ruleBasedMatch 基于规则匹配
func (km *KeyMatcher) ruleBasedMatch(sourceRecord, targetRecord map[string]interface{},
	rule KeyMatchingRule) (float64, []string, []string) {

	return km.ruleMatcher.Match(sourceRecord, targetRecord, rule)
}

// normalizeValue 标准化值
func (km *KeyMatcher) normalizeValue(value interface{}) string {
	if value == nil {
		return ""
	}

	str := fmt.Sprintf("%v", value)
	// 去除空格，转换为小写
	return strings.ToLower(strings.TrimSpace(str))
}

// generateRecordID 生成记录ID
func (km *KeyMatcher) generateRecordID(record map[string]interface{}) string {
	// 使用记录的哈希值作为ID
	hash := md5.New()
	for key, value := range record {
		hash.Write([]byte(fmt.Sprintf("%s:%v", key, value)))
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// FuzzyMatcher 模糊匹配器
type FuzzyMatcher struct{}

// NewFuzzyMatcher 创建模糊匹配器
func NewFuzzyMatcher() *FuzzyMatcher {
	return &FuzzyMatcher{}
}

// CalculateSimilarity 计算相似度
func (fm *FuzzyMatcher) CalculateSimilarity(str1, str2 string) float64 {
	if str1 == str2 {
		return 1.0
	}

	if str1 == "" || str2 == "" {
		return 0.0
	}

	// 使用编辑距离计算相似度
	distance := fm.levenshteinDistance(str1, str2)
	maxLen := len(str1)
	if len(str2) > maxLen {
		maxLen = len(str2)
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance 计算编辑距离
func (fm *FuzzyMatcher) levenshteinDistance(str1, str2 string) int {
	len1, len2 := len(str1), len(str2)
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if str1[i-1] != str2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // 删除
				matrix[i][j-1]+1,      // 插入
				matrix[i-1][j-1]+cost, // 替换
			)
		}
	}

	return matrix[len1][len2]
}

// RuleMatcher 规则匹配器
type RuleMatcher struct{}

// NewRuleMatcher 创建规则匹配器
func NewRuleMatcher() *RuleMatcher {
	return &RuleMatcher{}
}

// Match 规则匹配
func (rm *RuleMatcher) Match(sourceRecord, targetRecord map[string]interface{},
	rule KeyMatchingRule) (float64, []string, []string) {

	// 这里可以实现复杂的规则匹配逻辑
	// 例如：电话号码格式化匹配、身份证号码匹配等

	var matchedFields []string
	var conflictFields []string
	var totalWeight float64
	var matchedWeight float64

	for _, field := range rule.MatchFields {
		weight := rule.WeightConfig[field]
		if weight == 0 {
			weight = 1.0
		}
		totalWeight += weight

		if rm.applyFieldRule(sourceRecord[field], targetRecord[field], field) {
			matchedFields = append(matchedFields, field)
			matchedWeight += weight
		} else {
			conflictFields = append(conflictFields, field)
		}
	}

	var score float64
	if totalWeight > 0 {
		score = matchedWeight / totalWeight
	}

	return score, matchedFields, conflictFields
}

// applyFieldRule 应用字段规则
func (rm *RuleMatcher) applyFieldRule(sourceValue, targetValue interface{}, fieldName string) bool {
	switch fieldName {
	case "phone", "mobile":
		return rm.matchPhoneNumbers(sourceValue, targetValue)
	case "email":
		return rm.matchEmails(sourceValue, targetValue)
	case "id_card", "identity_card":
		return rm.matchIDCards(sourceValue, targetValue)
	default:
		return rm.defaultMatch(sourceValue, targetValue)
	}
}

// matchPhoneNumbers 匹配电话号码
func (rm *RuleMatcher) matchPhoneNumbers(source, target interface{}) bool {
	sourcePhone := rm.normalizePhone(fmt.Sprintf("%v", source))
	targetPhone := rm.normalizePhone(fmt.Sprintf("%v", target))
	return sourcePhone == targetPhone
}

// normalizePhone 标准化电话号码
func (rm *RuleMatcher) normalizePhone(phone string) string {
	// 移除所有非数字字符
	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}

	result := digits.String()
	// 如果是11位手机号且以1开头，保持原样
	if len(result) == 11 && result[0] == '1' {
		return result
	}
	// 如果是带区号的固话，去掉区号
	if len(result) > 11 {
		return result[len(result)-8:] // 取后8位
	}
	return result
}

// matchEmails 匹配邮箱
func (rm *RuleMatcher) matchEmails(source, target interface{}) bool {
	sourceEmail := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", source)))
	targetEmail := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", target)))
	return sourceEmail == targetEmail
}

// matchIDCards 匹配身份证号
func (rm *RuleMatcher) matchIDCards(source, target interface{}) bool {
	sourceID := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", source)))
	targetID := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", target)))
	return sourceID == targetID
}

// defaultMatch 默认匹配
func (rm *RuleMatcher) defaultMatch(source, target interface{}) bool {
	return fmt.Sprintf("%v", source) == fmt.Sprintf("%v", target)
}

// min 返回最小值
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}
