/*
 * @module service/thematic_sync/conflict_resolver
 * @description 冲突解决器，处理数据汇聚过程中的字段冲突和数据合并
 * @architecture 策略模式 - 支持多种冲突解决策略
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 冲突检测 -> 策略选择 -> 冲突解决 -> 结果验证 -> 记录追踪
 * @rules 确保冲突解决的一致性和可追溯性，支持自定义解决规则
 * @dependencies time, fmt, reflect
 * @refs key_matcher.go, aggregation_engine.go
 */

package thematic_sync

import (
	"fmt"
	"reflect"
	"time"
)

// ConflictResolutionPolicy 冲突解决策略
type ConflictResolutionPolicy string

const (
	KeepSource  ConflictResolutionPolicy = "keep_source"  // 保留源数据
	KeepTarget  ConflictResolutionPolicy = "keep_target"  // 保留目标数据
	KeepLatest  ConflictResolutionPolicy = "keep_latest"  // 保留最新数据
	MergeFields ConflictResolutionPolicy = "merge_fields" // 字段级合并
	CustomRule  ConflictResolutionPolicy = "custom_rule"  // 自定义规则
)

// ConflictInfo 冲突信息
type ConflictInfo struct {
	FieldName    string      `json:"field_name"`
	SourceValue  interface{} `json:"source_value"`
	TargetValue  interface{} `json:"target_value"`
	ConflictType string      `json:"conflict_type"` // value_diff, type_diff, null_conflict
	Resolution   string      `json:"resolution"`    // 解决方案描述
}

// CustomResolutionRule 自定义解决规则
type CustomResolutionRule struct {
	FieldName string                 `json:"field_name"`
	Condition map[string]interface{} `json:"condition"`
	Action    string                 `json:"action"`    // keep_source, keep_target, merge, transform
	Transform string                 `json:"transform"` // 转换表达式
	Priority  int                    `json:"priority"`  // 优先级
}

// ConflictResolver 冲突解决器
type ConflictResolver struct {
	policies    map[string]ConflictResolutionPolicy
	customRules map[string][]CustomResolutionRule
}

// NewConflictResolver 创建冲突解决器
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{
		policies:    make(map[string]ConflictResolutionPolicy),
		customRules: make(map[string][]CustomResolutionRule),
	}
}

// SetFieldPolicy 设置字段级冲突解决策略
func (cr *ConflictResolver) SetFieldPolicy(fieldName string, policy ConflictResolutionPolicy) {
	cr.policies[fieldName] = policy
}

// AddCustomRule 添加自定义解决规则
func (cr *ConflictResolver) AddCustomRule(fieldName string, rule CustomResolutionRule) {
	if cr.customRules[fieldName] == nil {
		cr.customRules[fieldName] = make([]CustomResolutionRule, 0)
	}
	cr.customRules[fieldName] = append(cr.customRules[fieldName], rule)
}

// ResolveConflict 解决冲突
func (cr *ConflictResolver) ResolveConflict(sourceRecord, targetRecord map[string]interface{},
	policy ConflictResolutionPolicy) (map[string]interface{}, []ConflictInfo, error) {

	result := make(map[string]interface{})
	var conflicts []ConflictInfo

	// 合并所有字段
	allFields := cr.getAllFields(sourceRecord, targetRecord)

	for fieldName := range allFields {
		sourceValue := sourceRecord[fieldName]
		targetValue := targetRecord[fieldName]

		// 检查是否存在冲突
		if cr.hasConflict(sourceValue, targetValue) {
			conflict := ConflictInfo{
				FieldName:    fieldName,
				SourceValue:  sourceValue,
				TargetValue:  targetValue,
				ConflictType: cr.getConflictType(sourceValue, targetValue),
			}

			// 解决冲突
			resolvedValue, resolution, err := cr.resolveFieldConflict(
				fieldName, sourceValue, targetValue, policy)
			if err != nil {
				return nil, nil, fmt.Errorf("解决字段 %s 冲突失败: %w", fieldName, err)
			}

			result[fieldName] = resolvedValue
			conflict.Resolution = resolution
			conflicts = append(conflicts, conflict)
		} else {
			// 无冲突，选择非空值
			if sourceValue != nil {
				result[fieldName] = sourceValue
			} else {
				result[fieldName] = targetValue
			}
		}
	}

	return result, conflicts, nil
}

// resolveFieldConflict 解决字段冲突
func (cr *ConflictResolver) resolveFieldConflict(fieldName string, sourceValue, targetValue interface{},
	defaultPolicy ConflictResolutionPolicy) (interface{}, string, error) {

	// 优先使用字段级策略
	policy := defaultPolicy
	if fieldPolicy, exists := cr.policies[fieldName]; exists {
		policy = fieldPolicy
	}

	// 检查自定义规则
	if rules, exists := cr.customRules[fieldName]; exists {
		for _, rule := range rules {
			if cr.matchesCondition(sourceValue, targetValue, rule.Condition) {
				return cr.applyCustomRule(sourceValue, targetValue, rule)
			}
		}
	}

	// 应用默认策略
	switch policy {
	case KeepSource:
		return sourceValue, "保留源数据", nil
	case KeepTarget:
		return targetValue, "保留目标数据", nil
	case KeepLatest:
		return cr.keepLatest(sourceValue, targetValue)
	case MergeFields:
		return cr.mergeFields(sourceValue, targetValue)
	default:
		return sourceValue, "默认保留源数据", nil
	}
}

// keepLatest 保留最新数据
func (cr *ConflictResolver) keepLatest(sourceValue, targetValue interface{}) (interface{}, string, error) {
	// 尝试解析时间戳
	sourceTime := cr.parseTimestamp(sourceValue)
	targetTime := cr.parseTimestamp(targetValue)

	if sourceTime != nil && targetTime != nil {
		if sourceTime.After(*targetTime) {
			return sourceValue, "保留较新的源数据", nil
		}
		return targetValue, "保留较新的目标数据", nil
	}

	// 如果无法解析时间，默认保留源数据
	return sourceValue, "无法确定时间，保留源数据", nil
}

// mergeFields 合并字段
func (cr *ConflictResolver) mergeFields(sourceValue, targetValue interface{}) (interface{}, string, error) {
	// 如果是字符串，尝试合并
	sourceStr, sourceIsStr := sourceValue.(string)
	targetStr, targetIsStr := targetValue.(string)

	if sourceIsStr && targetIsStr {
		if sourceStr != "" && targetStr != "" && sourceStr != targetStr {
			merged := fmt.Sprintf("%s;%s", sourceStr, targetStr)
			return merged, "合并字符串值", nil
		}

		// 优先选择非空值
		if sourceStr != "" {
			return sourceStr, "选择非空源值", nil
		}
		return targetStr, "选择非空目标值", nil
	}

	// 如果是数值，尝试求平均
	sourceNum := cr.parseNumber(sourceValue)
	targetNum := cr.parseNumber(targetValue)

	if sourceNum != nil && targetNum != nil {
		average := (*sourceNum + *targetNum) / 2
		return average, "计算平均值", nil
	}

	// 默认保留源数据
	return sourceValue, "无法合并，保留源数据", nil
}

// applyCustomRule 应用自定义规则
func (cr *ConflictResolver) applyCustomRule(sourceValue, targetValue interface{},
	rule CustomResolutionRule) (interface{}, string, error) {

	switch rule.Action {
	case "keep_source":
		return sourceValue, fmt.Sprintf("自定义规则：%s", rule.Action), nil
	case "keep_target":
		return targetValue, fmt.Sprintf("自定义规则：%s", rule.Action), nil
	case "merge":
		return cr.mergeFields(sourceValue, targetValue)
	case "transform":
		return cr.applyTransform(sourceValue, targetValue, rule.Transform)
	default:
		return sourceValue, "未知自定义规则，保留源数据", nil
	}
}

// applyTransform 应用转换规则
func (cr *ConflictResolver) applyTransform(sourceValue, targetValue interface{},
	transform string) (interface{}, string, error) {

	// 这里可以实现复杂的转换逻辑
	// 例如：表达式解析、函数调用等
	switch transform {
	case "concat":
		return fmt.Sprintf("%v%v", sourceValue, targetValue), "连接值", nil
	case "max":
		sourceNum := cr.parseNumber(sourceValue)
		targetNum := cr.parseNumber(targetValue)
		if sourceNum != nil && targetNum != nil {
			if *sourceNum > *targetNum {
				return *sourceNum, "选择最大值", nil
			}
			return *targetNum, "选择最大值", nil
		}
		return sourceValue, "无法比较，保留源数据", nil
	case "min":
		sourceNum := cr.parseNumber(sourceValue)
		targetNum := cr.parseNumber(targetValue)
		if sourceNum != nil && targetNum != nil {
			if *sourceNum < *targetNum {
				return *sourceNum, "选择最小值", nil
			}
			return *targetNum, "选择最小值", nil
		}
		return sourceValue, "无法比较，保留源数据", nil
	default:
		return sourceValue, "未知转换规则，保留源数据", nil
	}
}

// hasConflict 检查是否存在冲突
func (cr *ConflictResolver) hasConflict(sourceValue, targetValue interface{}) bool {
	// 如果有一个为空，不算冲突
	if sourceValue == nil || targetValue == nil {
		return false
	}

	// 使用反射比较值
	return !reflect.DeepEqual(sourceValue, targetValue)
}

// getConflictType 获取冲突类型
func (cr *ConflictResolver) getConflictType(sourceValue, targetValue interface{}) string {
	if sourceValue == nil && targetValue != nil {
		return "null_conflict"
	}
	if sourceValue != nil && targetValue == nil {
		return "null_conflict"
	}
	if sourceValue == nil && targetValue == nil {
		return "both_null"
	}

	sourceType := reflect.TypeOf(sourceValue)
	targetType := reflect.TypeOf(targetValue)

	if sourceType != targetType {
		return "type_diff"
	}

	return "value_diff"
}

// getAllFields 获取所有字段
func (cr *ConflictResolver) getAllFields(sourceRecord, targetRecord map[string]interface{}) map[string]bool {
	allFields := make(map[string]bool)

	for field := range sourceRecord {
		allFields[field] = true
	}

	for field := range targetRecord {
		allFields[field] = true
	}

	return allFields
}

// matchesCondition 检查是否匹配条件
func (cr *ConflictResolver) matchesCondition(sourceValue, targetValue interface{},
	condition map[string]interface{}) bool {

	// 简化的条件匹配实现
	for key, expectedValue := range condition {
		switch key {
		case "source_not_null":
			if expectedValue.(bool) && sourceValue == nil {
				return false
			}
		case "target_not_null":
			if expectedValue.(bool) && targetValue == nil {
				return false
			}
		case "source_type":
			if sourceValue != nil && reflect.TypeOf(sourceValue).String() != expectedValue.(string) {
				return false
			}
		case "target_type":
			if targetValue != nil && reflect.TypeOf(targetValue).String() != expectedValue.(string) {
				return false
			}
		}
	}

	return true
}

// parseTimestamp 解析时间戳
func (cr *ConflictResolver) parseTimestamp(value interface{}) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return &t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return &t
		}
	case int64:
		t := time.Unix(v, 0)
		return &t
	}
	return nil
}

// parseNumber 解析数值
func (cr *ConflictResolver) parseNumber(value interface{}) *float64 {
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
		if f, err := parseFloat(v); err == nil {
			return &f
		}
	}
	return nil
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	// 简化的浮点数解析
	var result float64
	var err error

	if _, err = fmt.Sscanf(s, "%f", &result); err != nil {
		return 0, err
	}

	return result, nil
}
