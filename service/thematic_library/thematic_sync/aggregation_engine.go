/*
 * @module service/thematic_sync/aggregation_engine
 * @description 数据汇聚引擎，负责多源数据汇聚、去重和合并处理
 * @architecture 策略模式 - 支持多种汇聚策略和数据合并方式
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 源数据获取 -> 主键匹配 -> 数据合并 -> 冲突解决 -> 血缘记录
 * @rules 确保数据汇聚的完整性和一致性，支持可配置的汇聚规则
 * @dependencies gorm.io/gorm, time, fmt
 * @refs key_matcher.go, conflict_resolver.go, data_lineage.go
 */

package thematic_sync

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AggregationStrategy 汇聚策略
type AggregationStrategy string

const (
	MergeStrategy   AggregationStrategy = "merge"   // 合并策略
	ReplaceStrategy AggregationStrategy = "replace" // 替换策略
	AppendStrategy  AggregationStrategy = "append"  // 追加策略
	UnionStrategy   AggregationStrategy = "union"   // 联合策略
)

// AggregationConfig 汇聚配置
type AggregationConfig struct {
	Strategy            AggregationStrategy      `json:"strategy"`
	KeyMatchingRules    []KeyMatchingRule        `json:"key_matching_rules"`
	FieldMappings       []FieldMapping           `json:"field_mappings"`
	ConflictPolicy      ConflictResolutionPolicy `json:"conflict_policy"`
	DeduplicationConfig DeduplicationConfig      `json:"deduplication_config"`
}

// FieldMapping 字段映射
type FieldMapping struct {
	SourceField     string           `json:"source_field"`
	TargetField     string           `json:"target_field"`
	Transform       string           `json:"transform,omitempty"`
	DefaultValue    interface{}      `json:"default_value,omitempty"`
	IsRequired      bool             `json:"is_required"`
	ValidationRules []ValidationRule `json:"validation_rules,omitempty"`
}

// ValidationRule 验证规则
type ValidationRule struct {
	Type      string      `json:"type"` // required, type, range, pattern
	Parameter interface{} `json:"parameter"`
	Message   string      `json:"message"`
}

// DeduplicationConfig 去重配置
type DeduplicationConfig struct {
	Enabled       bool     `json:"enabled"`
	KeyFields     []string `json:"key_fields"`     // 去重关键字段
	Strategy      string   `json:"strategy"`       // first, last, best_quality
	QualityFields []string `json:"quality_fields"` // 质量评估字段
}

// AggregationResult 汇聚结果
type AggregationResult struct {
	SourceRecords     []SourceRecordInfo       `json:"source_records"`
	AggregatedRecords []map[string]interface{} `json:"aggregated_records"`
	MatchResults      []MatchResult            `json:"match_results"`
	ConflictResults   []ConflictResolution     `json:"conflict_results"`
	LineageRecords    []LineageRecord          `json:"lineage_records"`
	Statistics        AggregationStatistics    `json:"statistics"`
	ProcessingTime    time.Duration            `json:"processing_time"`
}

// SourceRecordInfo 源记录信息
type SourceRecordInfo struct {
	LibraryID   string                 `json:"library_id"`
	InterfaceID string                 `json:"interface_id"`
	RecordID    string                 `json:"record_id"`
	Record      map[string]interface{} `json:"record"`
	Quality     float64                `json:"quality"`
	LastUpdated time.Time              `json:"last_updated"`
}

// ConflictResolution 冲突解决
type ConflictResolution struct {
	RecordID      string         `json:"record_id"`
	Conflicts     []ConflictInfo `json:"conflicts"`
	Resolution    string         `json:"resolution"`
	ResolvedValue interface{}    `json:"resolved_value"`
}

// LineageRecord 血缘记录
type LineageRecord struct {
	TargetRecordID    string               `json:"target_record_id"`
	SourceRecords     []SourceRecordInfo   `json:"source_records"`
	TransformationLog []TransformationStep `json:"transformation_log"`
	CreatedAt         time.Time            `json:"created_at"`
}

// TransformationStep 转换步骤
type TransformationStep struct {
	Step        string      `json:"step"`
	Input       interface{} `json:"input"`
	Output      interface{} `json:"output"`
	Rule        string      `json:"rule"`
	ProcessTime time.Time   `json:"process_time"`
}

// AggregationStatistics 汇聚统计
type AggregationStatistics struct {
	TotalSourceRecords  int64         `json:"total_source_records"`
	TotalTargetRecords  int64         `json:"total_target_records"`
	MatchedRecords      int64         `json:"matched_records"`
	ConflictedRecords   int64         `json:"conflicted_records"`
	DeduplicatedRecords int64         `json:"deduplicated_records"`
	ProcessingDuration  time.Duration `json:"processing_duration"`
	AverageQualityScore float64       `json:"average_quality_score"`
}

// AggregationEngine 数据汇聚引擎
type AggregationEngine struct {
	db               *gorm.DB
	keyMatcher       *KeyMatcher
	conflictResolver *ConflictResolver
	fieldMapper      *FieldMapper
	dataConverter    *DataConverter
	lineageTracker   *LineageTracker
}

// NewAggregationEngine 创建数据汇聚引擎
func NewAggregationEngine(db *gorm.DB) *AggregationEngine {
	return &AggregationEngine{
		db:               db,
		keyMatcher:       NewKeyMatcher(nil),
		conflictResolver: NewConflictResolver(),
		fieldMapper:      NewFieldMapper(),
		dataConverter:    NewDataConverter(),
		lineageTracker:   NewLineageTracker(),
	}
}

// AggregateData 汇聚数据
func (ae *AggregationEngine) AggregateData(sourceRecords []SourceRecordInfo,
	config AggregationConfig) (*AggregationResult, error) {

	startTime := time.Now()

	// 初始化结果
	result := &AggregationResult{
		SourceRecords:     sourceRecords,
		AggregatedRecords: make([]map[string]interface{}, 0),
		MatchResults:      make([]MatchResult, 0),
		ConflictResults:   make([]ConflictResolution, 0),
		LineageRecords:    make([]LineageRecord, 0),
		Statistics: AggregationStatistics{
			TotalSourceRecords: int64(len(sourceRecords)),
		},
	}

	// 1. 应用字段映射
	mappedRecords, err := ae.applyFieldMappings(sourceRecords, config.FieldMappings)
	if err != nil {
		return nil, fmt.Errorf("字段映射失败: %w", err)
	}

	// 2. 执行主键匹配
	ae.keyMatcher = NewKeyMatcher(config.KeyMatchingRules)
	matchResults, err := ae.performKeyMatching(mappedRecords)
	if err != nil {
		return nil, fmt.Errorf("主键匹配失败: %w", err)
	}
	result.MatchResults = matchResults
	result.Statistics.MatchedRecords = int64(len(matchResults))

	// 3. 执行数据汇聚
	aggregatedRecords, conflictResults, err := ae.performAggregation(
		mappedRecords, matchResults, config)
	if err != nil {
		return nil, fmt.Errorf("数据汇聚失败: %w", err)
	}
	result.AggregatedRecords = aggregatedRecords
	result.ConflictResults = conflictResults
	result.Statistics.ConflictedRecords = int64(len(conflictResults))

	// 4. 执行去重处理
	if config.DeduplicationConfig.Enabled {
		deduplicatedRecords, deduplicatedCount, err := ae.performDeduplication(
			aggregatedRecords, config.DeduplicationConfig)
		if err != nil {
			return nil, fmt.Errorf("去重处理失败: %w", err)
		}
		result.AggregatedRecords = deduplicatedRecords
		result.Statistics.DeduplicatedRecords = deduplicatedCount
	}

	// 5. 创建血缘记录
	lineageRecords := ae.createLineageRecords(result.AggregatedRecords, sourceRecords)
	result.LineageRecords = lineageRecords

	// 6. 计算统计信息
	result.Statistics.TotalTargetRecords = int64(len(result.AggregatedRecords))
	result.Statistics.ProcessingDuration = time.Since(startTime)
	result.Statistics.AverageQualityScore = ae.calculateAverageQuality(result.AggregatedRecords)
	result.ProcessingTime = time.Since(startTime)

	return result, nil
}

// applyFieldMappings 应用字段映射
func (ae *AggregationEngine) applyFieldMappings(sourceRecords []SourceRecordInfo,
	mappings []FieldMapping) ([]SourceRecordInfo, error) {

	var mappedRecords []SourceRecordInfo

	for _, sourceRecord := range sourceRecords {
		mappedRecord := SourceRecordInfo{
			LibraryID:   sourceRecord.LibraryID,
			InterfaceID: sourceRecord.InterfaceID,
			RecordID:    sourceRecord.RecordID,
			Record:      make(map[string]interface{}),
			Quality:     sourceRecord.Quality,
			LastUpdated: sourceRecord.LastUpdated,
		}

		// 应用字段映射
		for _, mapping := range mappings {
			sourceValue := sourceRecord.Record[mapping.SourceField]

			// 检查必填字段
			if mapping.IsRequired && sourceValue == nil {
				if mapping.DefaultValue != nil {
					sourceValue = mapping.DefaultValue
				} else {
					return nil, fmt.Errorf("必填字段 %s 缺失", mapping.SourceField)
				}
			}

			// 应用转换
			if mapping.Transform != "" && sourceValue != nil {
				transformedValue, err := ae.dataConverter.Transform(sourceValue, mapping.Transform)
				if err != nil {
					return nil, fmt.Errorf("字段转换失败 %s: %w", mapping.SourceField, err)
				}
				sourceValue = transformedValue
			}

			// 验证字段
			if err := ae.validateField(sourceValue, mapping.ValidationRules); err != nil {
				return nil, fmt.Errorf("字段验证失败 %s: %w", mapping.SourceField, err)
			}

			mappedRecord.Record[mapping.TargetField] = sourceValue
		}

		mappedRecords = append(mappedRecords, mappedRecord)
	}

	return mappedRecords, nil
}

// performKeyMatching 执行主键匹配
func (ae *AggregationEngine) performKeyMatching(records []SourceRecordInfo) ([]MatchResult, error) {
	if len(records) < 2 {
		return []MatchResult{}, nil
	}

	var allRecords []map[string]interface{}
	for _, record := range records {
		allRecords = append(allRecords, record.Record)
	}

	// 使用第一个记录作为源，其余作为目标进行匹配
	sourceRecords := []map[string]interface{}{allRecords[0]}
	targetRecords := allRecords[1:]

	return ae.keyMatcher.MatchRecords(sourceRecords, targetRecords)
}

// performAggregation 执行数据汇聚
func (ae *AggregationEngine) performAggregation(records []SourceRecordInfo,
	matchResults []MatchResult, config AggregationConfig) ([]map[string]interface{}, []ConflictResolution, error) {

	var aggregatedRecords []map[string]interface{}
	var conflictResults []ConflictResolution
	var err error

	switch config.Strategy {
	case MergeStrategy:
		aggregatedRecords, conflictResults, err = ae.performMergeAggregation(records, matchResults, config.ConflictPolicy)
		return aggregatedRecords, conflictResults, err
	case ReplaceStrategy:
		aggregatedRecords, conflictResults, err = ae.performReplaceAggregation(records, matchResults)
		return aggregatedRecords, conflictResults, err
	case AppendStrategy:
		aggregatedRecords, conflictResults, err = ae.performAppendAggregation(records)
		return aggregatedRecords, conflictResults, err
	case UnionStrategy:
		aggregatedRecords, conflictResults, err = ae.performUnionAggregation(records, matchResults)
		return aggregatedRecords, conflictResults, err
	default:
		return nil, nil, fmt.Errorf("未知汇聚策略: %s", config.Strategy)
	}
}

// performMergeAggregation 执行合并汇聚
func (ae *AggregationEngine) performMergeAggregation(records []SourceRecordInfo,
	matchResults []MatchResult, conflictPolicy ConflictResolutionPolicy) ([]map[string]interface{}, []ConflictResolution, error) {

	var aggregatedRecords []map[string]interface{}
	var conflictResults []ConflictResolution

	// 如果没有匹配结果，直接返回所有记录
	if len(matchResults) == 0 {
		for _, record := range records {
			aggregatedRecords = append(aggregatedRecords, record.Record)
		}
		return aggregatedRecords, conflictResults, nil
	}

	// 处理匹配的记录
	for _, matchResult := range matchResults {
		mergedRecord, conflicts, err := ae.conflictResolver.ResolveConflict(
			matchResult.SourceRecord, matchResult.TargetRecord, conflictPolicy)
		if err != nil {
			return nil, nil, fmt.Errorf("冲突解决失败: %w", err)
		}

		aggregatedRecords = append(aggregatedRecords, mergedRecord)

		if len(conflicts) > 0 {
			conflictResult := ConflictResolution{
				RecordID:   matchResult.SourceRecordID,
				Conflicts:  conflicts,
				Resolution: "merged",
			}
			conflictResults = append(conflictResults, conflictResult)
		}
	}

	return aggregatedRecords, conflictResults, nil
}

// performReplaceAggregation 执行替换汇聚
func (ae *AggregationEngine) performReplaceAggregation(records []SourceRecordInfo,
	matchResults []MatchResult) ([]map[string]interface{}, []ConflictResolution, error) {

	var aggregatedRecords []map[string]interface{}

	// 简单替换策略：使用最新的记录
	recordMap := make(map[string]SourceRecordInfo)

	for _, record := range records {
		key := ae.generateRecordKey(record.Record)
		if existing, exists := recordMap[key]; !exists || record.LastUpdated.After(existing.LastUpdated) {
			recordMap[key] = record
		}
	}

	for _, record := range recordMap {
		aggregatedRecords = append(aggregatedRecords, record.Record)
	}

	return aggregatedRecords, []ConflictResolution{}, nil
}

// performAppendAggregation 执行追加汇聚
func (ae *AggregationEngine) performAppendAggregation(records []SourceRecordInfo) ([]map[string]interface{}, []ConflictResolution, error) {
	var aggregatedRecords []map[string]interface{}

	for _, record := range records {
		aggregatedRecords = append(aggregatedRecords, record.Record)
	}

	return aggregatedRecords, []ConflictResolution{}, nil
}

// performUnionAggregation 执行联合汇聚
func (ae *AggregationEngine) performUnionAggregation(records []SourceRecordInfo,
	matchResults []MatchResult) ([]map[string]interface{}, []ConflictResolution, error) {

	var aggregatedRecords []map[string]interface{}
	recordSet := make(map[string]bool)

	for _, record := range records {
		key := ae.generateRecordKey(record.Record)
		if !recordSet[key] {
			aggregatedRecords = append(aggregatedRecords, record.Record)
			recordSet[key] = true
		}
	}

	return aggregatedRecords, []ConflictResolution{}, nil
}

// performDeduplication 执行去重处理
func (ae *AggregationEngine) performDeduplication(records []map[string]interface{},
	config DeduplicationConfig) ([]map[string]interface{}, int64, error) {

	if !config.Enabled || len(config.KeyFields) == 0 {
		return records, 0, nil
	}

	recordMap := make(map[string]map[string]interface{})
	var deduplicatedCount int64

	for _, record := range records {
		key := ae.generateDeduplicationKey(record, config.KeyFields)

		if existing, exists := recordMap[key]; exists {
			deduplicatedCount++
			// 根据策略选择保留哪个记录
			switch config.Strategy {
			case "last":
				recordMap[key] = record // 保留最后一个
			case "best_quality":
				if ae.calculateRecordQuality(record, config.QualityFields) >
					ae.calculateRecordQuality(existing, config.QualityFields) {
					recordMap[key] = record
				}
			case "first":
				// 保留第一个，不做任何操作
			default:
				recordMap[key] = record
			}
		} else {
			recordMap[key] = record
		}
	}

	var deduplicatedRecords []map[string]interface{}
	for _, record := range recordMap {
		deduplicatedRecords = append(deduplicatedRecords, record)
	}

	return deduplicatedRecords, deduplicatedCount, nil
}

// createLineageRecords 创建血缘记录
func (ae *AggregationEngine) createLineageRecords(aggregatedRecords []map[string]interface{},
	sourceRecords []SourceRecordInfo) []LineageRecord {

	var lineageRecords []LineageRecord

	for i, aggregatedRecord := range aggregatedRecords {
		lineageRecord := LineageRecord{
			TargetRecordID:    fmt.Sprintf("target_%d", i),
			SourceRecords:     sourceRecords,
			TransformationLog: []TransformationStep{},
			CreatedAt:         time.Now(),
		}

		// 添加转换步骤记录
		step := TransformationStep{
			Step:        "aggregation",
			Input:       sourceRecords,
			Output:      aggregatedRecord,
			Rule:        "merge_strategy",
			ProcessTime: time.Now(),
		}
		lineageRecord.TransformationLog = append(lineageRecord.TransformationLog, step)

		lineageRecords = append(lineageRecords, lineageRecord)
	}

	return lineageRecords
}

// validateField 验证字段
func (ae *AggregationEngine) validateField(value interface{}, rules []ValidationRule) error {
	for _, rule := range rules {
		if err := ae.applyValidationRule(value, rule); err != nil {
			return err
		}
	}
	return nil
}

// applyValidationRule 应用验证规则
func (ae *AggregationEngine) applyValidationRule(value interface{}, rule ValidationRule) error {
	switch rule.Type {
	case "required":
		if value == nil {
			return fmt.Errorf(rule.Message)
		}
	case "type":
		expectedType := rule.Parameter.(string)
		if !ae.isValidType(value, expectedType) {
			return fmt.Errorf(rule.Message)
		}
		// 可以添加更多验证规则
	}
	return nil
}

// isValidType 检查类型是否有效
func (ae *AggregationEngine) isValidType(value interface{}, expectedType string) bool {
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

// generateRecordKey 生成记录键
func (ae *AggregationEngine) generateRecordKey(record map[string]interface{}) string {
	// 简化实现：使用所有字段值的组合
	key := ""
	for field, value := range record {
		key += fmt.Sprintf("%s:%v;", field, value)
	}
	return key
}

// generateDeduplicationKey 生成去重键
func (ae *AggregationEngine) generateDeduplicationKey(record map[string]interface{}, keyFields []string) string {
	key := ""
	for _, field := range keyFields {
		if value, exists := record[field]; exists {
			key += fmt.Sprintf("%s:%v;", field, value)
		}
	}
	return key
}

// calculateRecordQuality 计算记录质量
func (ae *AggregationEngine) calculateRecordQuality(record map[string]interface{}, qualityFields []string) float64 {
	if len(qualityFields) == 0 {
		return 1.0
	}

	score := 0.0
	for _, field := range qualityFields {
		if value, exists := record[field]; exists && value != nil {
			score += 1.0
		}
	}

	return score / float64(len(qualityFields))
}

// calculateAverageQuality 计算平均质量
func (ae *AggregationEngine) calculateAverageQuality(records []map[string]interface{}) float64 {
	if len(records) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, record := range records {
		// 简化的质量计算：非空字段比例
		nonNullCount := 0
		for _, value := range record {
			if value != nil {
				nonNullCount++
			}
		}
		if len(record) > 0 {
			totalScore += float64(nonNullCount) / float64(len(record))
		}
	}

	return totalScore / float64(len(records))
}

// FieldMapper 字段映射器（简化实现）
type FieldMapper struct{}

// NewFieldMapper 创建字段映射器
func NewFieldMapper() *FieldMapper {
	return &FieldMapper{}
}

// DataConverter 数据转换器（简化实现）
type DataConverter struct{}

// NewDataConverter 创建数据转换器
func NewDataConverter() *DataConverter {
	return &DataConverter{}
}

// Transform 转换数据
func (dc *DataConverter) Transform(value interface{}, transform string) (interface{}, error) {
	// 简化的转换实现
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
	}
	return value, nil
}

// LineageTracker 血缘追踪器（简化实现）
type LineageTracker struct{}

// NewLineageTracker 创建血缘追踪器
func NewLineageTracker() *LineageTracker {
	return &LineageTracker{}
}
