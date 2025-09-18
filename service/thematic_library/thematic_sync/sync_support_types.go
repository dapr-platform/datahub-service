/*
 * @module service/thematic_sync/sync_support_types
 * @description 同步引擎的支持类型和结构定义
 * @architecture 数据传输对象模式 - 定义同步过程中使用的各种数据结构
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 数据结构定义 -> 类型转换 -> 流程传递 -> 结果封装
 * @rules 确保数据结构的一致性和类型安全
 * @dependencies time
 * @refs sync_engine.go, aggregation_engine.go, cleansing_engine.go
 */

package thematic_sync

import (
	"time"
)

// SourceRecordInfo 源记录信息
type SourceRecordInfo struct {
	LibraryID   string                 `json:"library_id"`
	InterfaceID string                 `json:"interface_id"`
	RecordID    string                 `json:"record_id"`
	Record      map[string]interface{} `json:"record"`
	Quality     float64                `json:"quality"`
	LastUpdated time.Time              `json:"last_updated"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MatchingStrategy 匹配策略 - 已在key_matcher.go中定义

// KeyMatchingRule 主键匹配规则 - 已在key_matcher.go中定义

// ConflictResolutionPolicy 冲突解决策略 - 已在conflict_resolver.go中定义

// DeduplicationConfig 去重配置 - 已在aggregation_engine.go中定义

// AggregationResult 汇聚结果 - 已在aggregation_engine.go中定义

// ConflictRecord 冲突记录
type ConflictRecord struct {
	TargetRecordID string             `json:"target_record_id"`
	ConflictFields []ConflictField    `json:"conflict_fields"`
	SourceRecords  []SourceRecordInfo `json:"source_records"`
	Resolution     string             `json:"resolution"`
	ResolvedAt     time.Time          `json:"resolved_at"`
}

// ConflictField 冲突字段
type ConflictField struct {
	FieldName     string          `json:"field_name"`
	ConflictType  string          `json:"conflict_type"` // value, format, type
	Values        []ConflictValue `json:"values"`
	ResolvedValue interface{}     `json:"resolved_value"`
}

// ConflictValue 冲突值
type ConflictValue struct {
	Value       interface{}      `json:"value"`
	Source      SourceRecordInfo `json:"source"`
	Confidence  float64          `json:"confidence"`
	LastUpdated time.Time        `json:"last_updated"`
}

// CleansingRuleType 清洗规则类型 - 已在cleansing_engine.go中定义

// RuleCondition 规则条件 - 已在cleansing_engine.go中定义

// RuleAction 规则动作 - 已在cleansing_engine.go中定义

// SensitivityLevel 敏感级别 - 已在privacy_engine.go中定义

// MaskingStrategy 脱敏策略 - 已在privacy_engine.go中定义

// 辅助类定义已在各自的文件中实现

// 质量检查相关类型定义 - 已移除，后续单独处理

// SyncExecutionOptions 同步执行选项
type SyncExecutionOptions struct {
	BatchSize          int                    `json:"batch_size,omitempty"`
	MaxRetries         int                    `json:"max_retries,omitempty"`
	TimeoutSeconds     int                    `json:"timeout_seconds,omitempty"`
	SkipValidation     bool                   `json:"skip_validation,omitempty"`
	SkipCleansing      bool                   `json:"skip_cleansing,omitempty"`
	SkipPrivacy        bool                   `json:"skip_privacy,omitempty"`
	CustomConfig       map[string]interface{} `json:"custom_config,omitempty"`
	NotificationConfig *NotificationConfig    `json:"notification_config,omitempty"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled    bool     `json:"enabled"`
	Channels   []string `json:"channels"` // email, webhook, message
	Recipients []string `json:"recipients"`
	Template   string   `json:"template,omitempty"`
}

// SourceLibraryConfig 源库配置
type SourceLibraryConfig struct {
	LibraryID   string                 `json:"library_id"`
	InterfaceID string                 `json:"interface_id"`
	SQLQuery    string                 `json:"sql_query,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Filters     []FilterConfig         `json:"filters,omitempty"`
	Transforms  []TransformConfig      `json:"transforms,omitempty"`
}

// FilterConfig 过滤配置
type FilterConfig struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	LogicOp  string      `json:"logic_op,omitempty"`
}

// TransformConfig 转换配置
type TransformConfig struct {
	SourceField string                 `json:"source_field"`
	TargetField string                 `json:"target_field"`
	Transform   string                 `json:"transform"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// AggregationConfig 汇聚配置 - 已在aggregation_engine.go中定义

// FieldMappingRules 字段映射规则
type FieldMappingRules struct {
	Rules []FieldMapping `json:"rules"`
}

// KeyMatchingRules 主键匹配规则集
type KeyMatchingRules struct {
	Rules []KeyMatchingRule `json:"rules"`
}

// CleansingRules 清洗规则集
type CleansingRules struct {
	Rules []CleansingRule `json:"rules"`
}

// PrivacyRules 隐私规则集
type PrivacyRules struct {
	Rules []PrivacyRule `json:"rules"`
}

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	Type            string     `json:"type"` // manual, once, interval, cron
	CronExpression  string     `json:"cron_expression,omitempty"`
	IntervalSeconds int        `json:"interval_seconds,omitempty"`
	ScheduledTime   *time.Time `json:"scheduled_time,omitempty"`
	Timezone        string     `json:"timezone,omitempty"`
}

// ValidationError 验证错误 - 已在cleansing_engine.go中定义，此处为扩展版本
type ValidationErrorExt struct {
	Field       string      `json:"field"`
	Value       interface{} `json:"value"`
	RuleID      string      `json:"rule_id"`
	ErrorType   string      `json:"error_type"`
	Message     string      `json:"message"`
	Severity    string      `json:"severity"`
	RecordIndex int         `json:"record_index,omitempty"`
	Suggestion  string      `json:"suggestion,omitempty"`
}

// 请求和响应结构

// CreateThematicSyncTaskRequest 创建主题同步任务请求
type CreateThematicSyncTaskRequest struct {
	ThematicLibraryID   string                `json:"thematic_library_id"`
	ThematicInterfaceID string                `json:"thematic_interface_id"`
	TaskName            string                `json:"task_name"`
	Description         string                `json:"description"`
	SourceLibraries     []SourceLibraryConfig `json:"source_libraries"`
	AggregationConfig   *AggregationConfig    `json:"aggregation_config,omitempty"`
	KeyMatchingRules    *KeyMatchingRules     `json:"key_matching_rules,omitempty"`
	FieldMappingRules   *FieldMappingRules    `json:"field_mapping_rules,omitempty"`
	CleansingRules      *CleansingRules       `json:"cleansing_rules,omitempty"`
	PrivacyRules        *PrivacyRules         `json:"privacy_rules,omitempty"`
	ScheduleConfig      *ScheduleConfig       `json:"schedule_config"`
	CreatedBy           string                `json:"created_by"`
}

// UpdateThematicSyncTaskRequest 更新主题同步任务请求
type UpdateThematicSyncTaskRequest struct {
	TaskName          string             `json:"task_name,omitempty"`
	Description       string             `json:"description,omitempty"`
	Status            string             `json:"status,omitempty"`
	ScheduleConfig    *ScheduleConfig    `json:"schedule_config,omitempty"`
	AggregationConfig *AggregationConfig `json:"aggregation_config,omitempty"`
	KeyMatchingRules  *KeyMatchingRules  `json:"key_matching_rules,omitempty"`
	FieldMappingRules *FieldMappingRules `json:"field_mapping_rules,omitempty"`
	CleansingRules    *CleansingRules    `json:"cleansing_rules,omitempty"`
	PrivacyRules      *PrivacyRules      `json:"privacy_rules,omitempty"`
	UpdatedBy         string             `json:"updated_by"`
}

// ExecuteSyncTaskRequest 执行同步任务请求
type ExecuteSyncTaskRequest struct {
	ExecutionType string                `json:"execution_type"`
	Options       *SyncExecutionOptions `json:"options,omitempty"`
}

// ListSyncTasksRequest 同步任务列表请求
type ListSyncTasksRequest struct {
	Page              int    `json:"page"`
	PageSize          int    `json:"page_size"`
	Status            string `json:"status,omitempty"`
	TriggerType       string `json:"trigger_type,omitempty"`
	ThematicLibraryID string `json:"thematic_library_id,omitempty"`
}

// ListSyncTasksResponse 同步任务列表响应
type ListSyncTasksResponse struct {
	Tasks    []interface{} `json:"tasks"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// ListSyncExecutionsRequest 同步执行记录列表请求
type ListSyncExecutionsRequest struct {
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	TaskID        string `json:"task_id,omitempty"`
	Status        string `json:"status,omitempty"`
	ExecutionType string `json:"execution_type,omitempty"`
}

// ListSyncExecutionsResponse 同步执行记录列表响应
type ListSyncExecutionsResponse struct {
	Executions []interface{} `json:"executions"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
}

// ThematicSyncTaskStatistics 主题同步任务统计
type ThematicSyncTaskStatistics struct {
	Task               interface{}   `json:"task"`
	TotalExecutions    int64         `json:"total_executions"`
	SuccessExecutions  int64         `json:"success_executions"`
	FailedExecutions   int64         `json:"failed_executions"`
	SuccessRate        float64       `json:"success_rate"`
	AverageProcessTime int64         `json:"average_process_time"`
	TotalProcessedRows int64         `json:"total_processed_rows"`
	RecentExecutions   []interface{} `json:"recent_executions"`
}
