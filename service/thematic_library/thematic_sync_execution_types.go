/*
 * @module service/thematic_library/thematic_sync_execution_types
 * @description 主题同步执行相关的类型定义
 * @architecture 数据传输对象 - 执行配置和选项的强类型定义
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow N/A
 * @rules 所有执行配置都使用强类型结构，避免map[string]interface{}
 * @dependencies time
 * @refs thematic_sync_types.go, thematic_sync_config_types.go
 */

package thematic_library

import (
	"time"
)

// SyncExecutionOptions 同步执行选项
type SyncExecutionOptions struct {
	Mode             string              `json:"mode" validate:"required,oneof=full incremental realtime test"`
	DryRun           bool                `json:"dry_run" default:"false"`
	BatchSize        int                 `json:"batch_size" default:"1000"`
	MaxRecords       int                 `json:"max_records,omitempty"`
	Timeout          int                 `json:"timeout" default:"3600"` // 超时时间(秒)
	ForceRefresh     bool                `json:"force_refresh" default:"false"`
	SkipValidation   bool                `json:"skip_validation" default:"false"`
	IgnoreErrors     bool                `json:"ignore_errors" default:"false"`
	ParallelWorkers  int                 `json:"parallel_workers" default:"1"`
	DataRange        *ExecutionDataRange `json:"data_range,omitempty"`
	FilterConditions []ExecutionFilter   `json:"filter_conditions,omitempty"`
	OutputSettings   *ExecutionOutput    `json:"output_settings,omitempty"`
	DebugMode        bool                `json:"debug_mode" default:"false"`
	CustomParameters map[string]string   `json:"custom_parameters,omitempty"`
}

// ExecutionDataRange 执行数据范围
type ExecutionDataRange struct {
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	DateField    string     `json:"date_field,omitempty"`
	PartitionKey string     `json:"partition_key,omitempty"`
	Conditions   []string   `json:"conditions,omitempty"`
}

// ExecutionFilter 执行过滤条件
type ExecutionFilter struct {
	Field    string      `json:"field" validate:"required"`
	Operator string      `json:"operator" validate:"required,oneof=eq ne gt lt ge le in nin like"`
	Value    interface{} `json:"value" validate:"required"`
	LogicOp  string      `json:"logic_op,omitempty" validate:"oneof=AND OR" default:"AND"`
}

// ExecutionOutput 执行输出设置
type ExecutionOutput struct {
	LogLevel       string   `json:"log_level" default:"INFO"`
	DetailedLog    bool     `json:"detailed_log" default:"false"`
	ProgressReport bool     `json:"progress_report" default:"true"`
	ReportInterval int      `json:"report_interval" default:"1000"` // 每处理多少条记录报告一次
	SaveMetrics    bool     `json:"save_metrics" default:"true"`
	ExportResults  bool     `json:"export_results" default:"false"`
	ExportFormat   string   `json:"export_format" default:"json"` // json, csv, excel
	ExportPath     string   `json:"export_path,omitempty"`
	IncludeFields  []string `json:"include_fields,omitempty"`
	ExcludeFields  []string `json:"exclude_fields,omitempty"`
}

// SyncExecutionStatus 同步执行状态
type SyncExecutionStatus struct {
	TaskID           string               `json:"task_id"`
	ExecutionID      string               `json:"execution_id"`
	Status           string               `json:"status"`
	Progress         *ExecutionProgress   `json:"progress,omitempty"`
	Statistics       *ExecutionStatistics `json:"statistics,omitempty"`
	CurrentStage     string               `json:"current_stage,omitempty"`
	StartTime        *time.Time           `json:"start_time,omitempty"`
	EndTime          *time.Time           `json:"end_time,omitempty"`
	EstimatedEndTime *time.Time           `json:"estimated_end_time,omitempty"`
	ErrorMessage     string               `json:"error_message,omitempty"`
	Warnings         []ExecutionWarning   `json:"warnings,omitempty"`
	CanCancel        bool                 `json:"can_cancel"`
	CanRetry         bool                 `json:"can_retry"`
	NextRetryTime    *time.Time           `json:"next_retry_time,omitempty"`
}

// ExecutionProgress 执行进度
type ExecutionProgress struct {
	TotalRecords     int64   `json:"total_records"`
	ProcessedRecords int64   `json:"processed_records"`
	SuccessRecords   int64   `json:"success_records"`
	ErrorRecords     int64   `json:"error_records"`
	SkippedRecords   int64   `json:"skipped_records"`
	Percentage       float64 `json:"percentage"`
	CurrentBatch     int     `json:"current_batch"`
	TotalBatches     int     `json:"total_batches"`
	RecordsPerSecond float64 `json:"records_per_second"`
	ElapsedTime      int64   `json:"elapsed_time"`   // 已用时间(秒)
	RemainingTime    int64   `json:"remaining_time"` // 剩余时间(秒)
}

// ExecutionStatistics 执行统计
type ExecutionStatistics struct {
	SourceRecords    map[string]int64      `json:"source_records"` // 按源库统计
	TargetRecords    int64                 `json:"target_records"`
	InsertedRecords  int64                 `json:"inserted_records"`
	UpdatedRecords   int64                 `json:"updated_records"`
	DeletedRecords   int64                 `json:"deleted_records"`
	DuplicateRecords int64                 `json:"duplicate_records"`
	ValidationErrors int64                 `json:"validation_errors"`
	TransformErrors  int64                 `json:"transform_errors"`
	DataQualityScore float64               `json:"data_quality_score"`
	ProcessingTime   int64                 `json:"processing_time"`  // 处理时间(毫秒)
	NetworkTime      int64                 `json:"network_time"`     // 网络时间(毫秒)
	DatabaseTime     int64                 `json:"database_time"`    // 数据库时间(毫秒)
	MemoryUsage      int64                 `json:"memory_usage"`     // 内存使用(字节)
	DiskUsage        int64                 `json:"disk_usage"`       // 磁盘使用(字节)
	StageStatistics  map[string]StageStats `json:"stage_statistics"` // 按阶段统计
}

// StageStats 阶段统计
type StageStats struct {
	StageName    string    `json:"stage_name"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     int64     `json:"duration"` // 持续时间(毫秒)
	RecordsIn    int64     `json:"records_in"`
	RecordsOut   int64     `json:"records_out"`
	ErrorCount   int64     `json:"error_count"`
	WarningCount int64     `json:"warning_count"`
	MemoryPeak   int64     `json:"memory_peak"` // 内存峰值(字节)
	Status       string    `json:"status"`
}

// ExecutionWarning 执行警告
type ExecutionWarning struct {
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Field       string    `json:"field,omitempty"`
	RecordID    string    `json:"record_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity" default:"warning"` // info, warning, error
	Recoverable bool      `json:"recoverable"`
	Suggestion  string    `json:"suggestion,omitempty"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExecutionID   string                  `json:"execution_id"`
	TaskID        string                  `json:"task_id"`
	Status        string                  `json:"status"`
	StartTime     time.Time               `json:"start_time"`
	EndTime       time.Time               `json:"end_time"`
	Duration      int64                   `json:"duration"` // 持续时间(毫秒)
	Statistics    *ExecutionStatistics    `json:"statistics"`
	DataLineage   *ExecutionDataLineage   `json:"data_lineage,omitempty"`
	QualityReport *ExecutionQualityReport `json:"quality_report,omitempty"`
	ErrorSummary  *ExecutionErrorSummary  `json:"error_summary,omitempty"`
	Warnings      []ExecutionWarning      `json:"warnings,omitempty"`
	Artifacts     []ExecutionArtifact     `json:"artifacts,omitempty"`
	Metadata      map[string]string       `json:"metadata,omitempty"`
}

// ExecutionDataLineage 执行数据血缘
type ExecutionDataLineage struct {
	SourceTables    []TableInfo  `json:"source_tables"`
	TargetTables    []TableInfo  `json:"target_tables"`
	Transformations []Transform  `json:"transformations"`
	Dependencies    []Dependency `json:"dependencies"`
}

// TableInfo 表信息
type TableInfo struct {
	LibraryID   string            `json:"library_id"`
	InterfaceID string            `json:"interface_id"`
	TableName   string            `json:"table_name"`
	RecordCount int64             `json:"record_count"`
	Fields      []string          `json:"fields"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Transform 转换信息
type Transform struct {
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	InputFields  []string          `json:"input_fields"`
	OutputFields []string          `json:"output_fields"`
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// Dependency 依赖关系
type Dependency struct {
	SourceTable string `json:"source_table"`
	TargetTable string `json:"target_table"`
	Type        string `json:"type"`     // direct, derived, aggregated
	Strength    string `json:"strength"` // strong, weak, conditional
}

// ExecutionQualityReport 执行质量报告
type ExecutionQualityReport struct {
	OverallScore    float64                 `json:"overall_score"`
	DimensionScores map[string]float64      `json:"dimension_scores"`
	RuleResults     []QualityRuleResult     `json:"rule_results"`
	Recommendations []QualityRecommendation `json:"recommendations"`
	Trends          []QualityTrend          `json:"trends,omitempty"`
}

// QualityRuleResult 质量规则结果
type QualityRuleResult struct {
	RuleID          string  `json:"rule_id"`
	RuleName        string  `json:"rule_name"`
	Status          string  `json:"status"` // passed, failed, warning
	Score           float64 `json:"score"`
	Message         string  `json:"message,omitempty"`
	Details         string  `json:"details,omitempty"`
	AffectedRecords int64   `json:"affected_records"`
}

// QualityRecommendation 质量建议
type QualityRecommendation struct {
	Type        string `json:"type"`     // improvement, fix, optimization
	Priority    string `json:"priority"` // high, medium, low
	Description string `json:"description"`
	Action      string `json:"action"`
	Impact      string `json:"impact"`
}

// QualityTrend 质量趋势
type QualityTrend struct {
	Dimension string    `json:"dimension"`
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	Change    float64   `json:"change"` // 相对于上次的变化
	Trend     string    `json:"trend"`  // improving, declining, stable
}

// ExecutionErrorSummary 执行错误汇总
type ExecutionErrorSummary struct {
	TotalErrors       int64            `json:"total_errors"`
	ErrorsByType      map[string]int64 `json:"errors_by_type"`
	ErrorsByStage     map[string]int64 `json:"errors_by_stage"`
	CriticalErrors    []ExecutionError `json:"critical_errors"`
	RecoverableErrors []ExecutionError `json:"recoverable_errors"`
	ErrorPatterns     []ErrorPattern   `json:"error_patterns,omitempty"`
}

// ExecutionError 执行错误
type ExecutionError struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Stage       string    `json:"stage"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	RecordID    string    `json:"record_id,omitempty"`
	Field       string    `json:"field,omitempty"`
	Value       string    `json:"value,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"` // critical, error, warning
	Recoverable bool      `json:"recoverable"`
	Resolution  string    `json:"resolution,omitempty"`
}

// ErrorPattern 错误模式
type ErrorPattern struct {
	Pattern     string  `json:"pattern"`
	Count       int64   `json:"count"`
	Percentage  float64 `json:"percentage"`
	Description string  `json:"description,omitempty"`
	Suggestion  string  `json:"suggestion,omitempty"`
}

// ExecutionArtifact 执行产出物
type ExecutionArtifact struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // log, report, data, config
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Size        int64             `json:"size"`
	Format      string            `json:"format"`
	CreatedTime time.Time         `json:"created_time"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
