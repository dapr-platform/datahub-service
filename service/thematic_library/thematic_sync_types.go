/*
 * @module service/thematic_library/thematic_sync_types
 * @description 主题同步API层类型定义 - 请求和响应结构
 * @architecture 数据传输对象 - 定义API请求和响应结构
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow N/A
 * @rules 确保类型定义的一致性和可扩展性，只包含API层相关类型
 * @dependencies time, models包
 * @refs thematic_sync_service.go, models/thematic_sync.go
 */

package thematic_library

import (
	"datahub-service/service/models"
	"time"
)

// ==================== 任务管理相关请求/响应 ====================

// CreateThematicSyncTaskRequest 创建主题同步任务请求
type CreateThematicSyncTaskRequest struct {
	ThematicLibraryID   string `json:"thematic_library_id" binding:"required"`
	ThematicInterfaceID string `json:"thematic_interface_id" binding:"required"`
	TaskName            string `json:"task_name" binding:"required"`
	Description         string `json:"description"`

	// 数据源配置 - 两种模式二选一
	// 模式1: 接口模式 - 从基础库的数据接口获取数据
	SourceLibraries []SourceLibraryConfig `json:"source_libraries,omitempty"`

	// 模式2: SQL模式 - 直接执行SQL查询获取数据（优先级更高）
	SQLQueries []SQLQueryConfig `json:"sql_queries,omitempty"`

	// 规则配置
	KeyMatchingRules  *KeyMatchingRules  `json:"key_matching_rules,omitempty"`
	FieldMappingRules *FieldMappingRules `json:"field_mapping_rules,omitempty"`

	// 数据治理规则配置
	QualityRuleConfigs   []models.QualityRuleConfig   `json:"quality_rule_configs,omitempty"`
	CleansingRuleConfigs []models.DataCleansingConfig `json:"cleansing_rule_configs,omitempty"`
	MaskingRuleConfigs   []models.DataMaskingConfig   `json:"masking_rule_configs,omitempty"`
	GovernanceConfig     *GovernanceExecutionConfig   `json:"governance_config,omitempty"`

	ScheduleConfig *ScheduleConfig `json:"schedule_config" binding:"required"`
	CreatedBy      string          `json:"created_by" binding:"required"`
}

// UpdateThematicSyncTaskRequest 更新主题同步任务请求
type UpdateThematicSyncTaskRequest struct {
	TaskName          string             `json:"task_name"`
	Description       string             `json:"description"`
	Status            string             `json:"status"`
	ScheduleConfig    *ScheduleConfig    `json:"schedule_config,omitempty"`
	KeyMatchingRules  *KeyMatchingRules  `json:"key_matching_rules,omitempty"`
	FieldMappingRules *FieldMappingRules `json:"field_mapping_rules,omitempty"`

	// 数据治理规则配置
	QualityRuleConfigs   []models.QualityRuleConfig   `json:"quality_rule_configs,omitempty"`
	CleansingRuleConfigs []models.DataCleansingConfig `json:"cleansing_rule_configs,omitempty"`
	MaskingRuleConfigs   []models.DataMaskingConfig   `json:"masking_rule_configs,omitempty"`
	GovernanceConfig     *GovernanceExecutionConfig   `json:"governance_config,omitempty"`

	UpdatedBy string `json:"updated_by" binding:"required"`
}

// ListSyncTasksRequest 获取同步任务列表请求
type ListSyncTasksRequest struct {
	ThematicLibraryID string `form:"thematic_library_id"`
	Status            string `form:"status"`
	TriggerType       string `form:"trigger_type"`
	Page              int    `form:"page,default=1"`
	PageSize          int    `form:"page_size,default=10"`
}

// ListSyncTasksResponse 获取同步任务列表响应
type ListSyncTasksResponse struct {
	Tasks    []models.ThematicSyncTask `json:"tasks"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

// ExecuteSyncTaskRequest 执行同步任务请求
type ExecuteSyncTaskRequest struct {
	ExecutionType string                `json:"execution_type"`
	Options       *SyncExecutionOptions `json:"options,omitempty"`
}

// ListSyncExecutionsRequest 获取执行记录列表请求
type ListSyncExecutionsRequest struct {
	TaskID        string `form:"task_id"`
	Status        string `form:"status"`
	ExecutionType string `form:"execution_type"`
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"page_size,default=10"`
}

// ListSyncExecutionsResponse 获取执行记录列表响应
type ListSyncExecutionsResponse struct {
	Executions []models.ThematicSyncExecution `json:"executions"`
	Total      int64                          `json:"total"`
	Page       int                            `json:"page"`
	PageSize   int                            `json:"page_size"`
}

// ThematicSyncTaskStatistics 主题同步任务统计信息
type ThematicSyncTaskStatistics struct {
	Task               *models.ThematicSyncTask       `json:"task"`
	TotalExecutions    int64                          `json:"total_executions"`
	SuccessExecutions  int64                          `json:"success_executions"`
	FailedExecutions   int64                          `json:"failed_executions"`
	SuccessRate        float64                        `json:"success_rate"`
	AverageProcessTime int64                          `json:"average_process_time"` // 处理时长（秒）
	TotalProcessedRows int64                          `json:"total_processed_rows"`
	RecentExecutions   []models.ThematicSyncExecution `json:"recent_executions"`
}

// ==================== 配置相关类型 ====================

// SourceLibraryConfig 源库配置
type SourceLibraryConfig struct {
	LibraryID   string                  `json:"library_id" validate:"required"`
	Interfaces  []SourceInterfaceConfig `json:"interfaces" validate:"required,min=1"`
	FilterRules []DataFilterRule        `json:"filter_rules,omitempty"`
	Priority    int                     `json:"priority" default:"1"`
	Enabled     bool                    `json:"enabled" default:"true"`
	SyncMode    string                  `json:"sync_mode" default:"full"` // full, incremental, realtime
}

// SourceInterfaceConfig 源接口配置
type SourceInterfaceConfig struct {
	InterfaceID       string             `json:"interface_id" validate:"required"`
	FieldMapping      []FieldMapping     `json:"field_mapping,omitempty"`
	FilterCondition   string             `json:"filter_condition,omitempty"`
	SortOrder         []SortField        `json:"sort_order,omitempty"`
	BatchSize         int                `json:"batch_size,omitempty" default:"1000"`
	Parameters        map[string]string  `json:"parameters,omitempty"`
	IncrementalConfig *IncrementalConfig `json:"incremental_config,omitempty"`
}

// FieldMapping 字段映射
type FieldMapping struct {
	SourceField  string      `json:"source_field" validate:"required"`
	TargetField  string      `json:"target_field" validate:"required"`
	Transform    string      `json:"transform,omitempty"`
	Required     bool        `json:"required" default:"false"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// SortField 排序字段
type SortField struct {
	Field string `json:"field" validate:"required"`
	Order string `json:"order" validate:"oneof=ASC DESC" default:"ASC"`
}

// DataFilterRule 数据过滤规则
type DataFilterRule struct {
	Field    string      `json:"field" validate:"required"`
	Operator string      `json:"operator" validate:"required,oneof=eq ne gt lt ge le in nin like"`
	Value    interface{} `json:"value" validate:"required"`
	LogicOp  string      `json:"logic_op,omitempty" validate:"oneof=AND OR" default:"AND"`
}

// IncrementalConfig 增量同步配置
type IncrementalConfig struct {
	Enabled            bool   `json:"enabled"`                          // 是否启用增量同步
	IncrementalField   string `json:"incremental_field"`                // 增量字段名称
	FieldType          string `json:"field_type"`                       // 字段类型：timestamp, number, string
	CompareOperator    string `json:"compare_operator" default:">"`     // 比较操作符
	LastSyncValue      string `json:"last_sync_value,omitempty"`        // 上次同步的值
	InitialValue       string `json:"initial_value,omitempty"`          // 初始值
	MaxLookbackHours   int    `json:"max_lookback_hours,omitempty"`     // 最大回溯小时数
	CheckDeletedField  string `json:"check_deleted_field,omitempty"`    // 软删除字段名称
	DeletedValue       string `json:"deleted_value,omitempty"`          // 删除标记值
	BatchSize          int    `json:"batch_size" default:"1000"`        // 增量同步批次大小
	SyncDeletedRecords bool   `json:"sync_deleted_records"`             // 是否同步已删除的记录
	TimestampFormat    string `json:"timestamp_format,omitempty"`       // 时间戳格式
	TimeZone           string `json:"timezone" default:"Asia/Shanghai"` // 时区
}

// SQLQueryConfig SQL查询配置 - 简化版本,直接执行SQL语句
// 用于执行自定义SQL查询,如统计查询、复杂关联查询等
type SQLQueryConfig struct {
	SQLQuery   string                 `json:"sql_query" validate:"required"`      // SQL查询语句
	Parameters map[string]interface{} `json:"parameters,omitempty"`               // 查询参数(支持参数化查询)
	Timeout    int                    `json:"timeout,omitempty" default:"30"`     // 查询超时时间（秒）
	MaxRows    int                    `json:"max_rows,omitempty" default:"10000"` // 最大返回行数
}

// 向后兼容的类型别名
type SQLDataSourceConfig = SQLQueryConfig

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	Type            string           `json:"type" validate:"required,oneof=manual one_time interval cron"`
	CronExpression  string           `json:"cron_expression,omitempty"`
	IntervalSeconds int              `json:"interval_seconds,omitempty"`
	ScheduledTime   *time.Time       `json:"scheduled_time,omitempty"`
	TimeZone        string           `json:"timezone,omitempty" default:"Asia/Shanghai"`
	MaxRetries      int              `json:"max_retries,omitempty" default:"3"`
	RetryInterval   int              `json:"retry_interval,omitempty" default:"300"`
	Enabled         bool             `json:"enabled" default:"true"`
	StartDate       *time.Time       `json:"start_date,omitempty"`
	EndDate         *time.Time       `json:"end_date,omitempty"`
	ExecutionWindow *ExecutionWindow `json:"execution_window,omitempty"`
}

// ExecutionWindow 执行时间窗口
type ExecutionWindow struct {
	StartTime string `json:"start_time" example:"09:00"`
	EndTime   string `json:"end_time" example:"18:00"`
	Days      []int  `json:"days,omitempty"`
	Holidays  bool   `json:"holidays" default:"false"`
}

// SyncExecutionOptions 同步执行选项
type SyncExecutionOptions struct {
	Mode             string              `json:"mode" validate:"required,oneof=full incremental realtime test"`
	DryRun           bool                `json:"dry_run" default:"false"`
	BatchSize        int                 `json:"batch_size" default:"1000"`
	MaxRecords       int                 `json:"max_records,omitempty"`
	Timeout          int                 `json:"timeout" default:"3600"`
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
	ReportInterval int      `json:"report_interval" default:"1000"`
	SaveMetrics    bool     `json:"save_metrics" default:"true"`
	ExportResults  bool     `json:"export_results" default:"false"`
	ExportFormat   string   `json:"export_format" default:"json"`
	ExportPath     string   `json:"export_path,omitempty"`
	IncludeFields  []string `json:"include_fields,omitempty"`
	ExcludeFields  []string `json:"exclude_fields,omitempty"`
}

// ==================== 规则配置相关类型 ====================

// KeyMatchingRules 主键匹配规则
type KeyMatchingRules struct {
	PrimaryKeys    []PrimaryKeyRule  `json:"primary_keys" validate:"required,min=1"`
	FuzzyMatching  *FuzzyMatchConfig `json:"fuzzy_matching,omitempty"`
	ConflictPolicy string            `json:"conflict_policy" validate:"required,oneof=first last merge error"`
	MatchThreshold float64           `json:"match_threshold,omitempty" default:"0.8"`
	CaseSensitive  bool              `json:"case_sensitive" default:"true"`
}

// PrimaryKeyRule 主键规则
type PrimaryKeyRule struct {
	Fields     []string       `json:"fields" validate:"required,min=1"`
	Weight     float64        `json:"weight" default:"1.0"`
	Transform  []KeyTransform `json:"transform,omitempty"`
	Validation *KeyValidation `json:"validation,omitempty"`
}

// KeyTransform 主键转换
type KeyTransform struct {
	Type       string            `json:"type" validate:"required,oneof=trim upper lower hash normalize"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// KeyValidation 主键验证
type KeyValidation struct {
	Required   bool   `json:"required" default:"true"`
	Pattern    string `json:"pattern,omitempty"`
	MinLength  int    `json:"min_length,omitempty"`
	MaxLength  int    `json:"max_length,omitempty"`
	AllowEmpty bool   `json:"allow_empty" default:"false"`
}

// FuzzyMatchConfig 模糊匹配配置
type FuzzyMatchConfig struct {
	Enabled       bool             `json:"enabled" default:"false"`
	Algorithm     string           `json:"algorithm" default:"levenshtein"`
	Threshold     float64          `json:"threshold" default:"0.8"`
	Fields        []string         `json:"fields,omitempty"`
	CaseIgnore    bool             `json:"case_ignore" default:"true"`
	Preprocessing []PreprocessRule `json:"preprocessing,omitempty"`
}

// PreprocessRule 预处理规则
type PreprocessRule struct {
	Type       string `json:"type" validate:"required,oneof=trim normalize remove_special"`
	Parameters string `json:"parameters,omitempty"`
}

// FieldMappingRules 字段映射规则
type FieldMappingRules struct {
	Mappings       []FieldMappingRule    `json:"mappings" validate:"required,min=1"`
	DefaultPolicy  string                `json:"default_policy" default:"ignore"`
	CaseSensitive  bool                  `json:"case_sensitive" default:"true"`
	TypeConversion *TypeConversionConfig `json:"type_conversion,omitempty"`
	Validation     *MappingValidation    `json:"validation,omitempty"`
}

// FieldMappingRule 字段映射规则
type FieldMappingRule struct {
	SourceField  string               `json:"source_field" validate:"required"`
	TargetField  string               `json:"target_field" validate:"required"`
	DataType     string               `json:"data_type,omitempty"`
	Transform    *FieldTransform      `json:"transform,omitempty"`
	Validation   *FieldValidationRule `json:"validation,omitempty"`
	DefaultValue interface{}          `json:"default_value,omitempty"`
	Required     bool                 `json:"required" default:"false"`
	Nullable     bool                 `json:"nullable" default:"true"`
}

// FieldTransform 字段转换
type FieldTransform struct {
	Type        string            `json:"type" validate:"required"`
	Expression  string            `json:"expression,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Conditional []ConditionalRule `json:"conditional,omitempty"`
}

// ConditionalRule 条件规则
type ConditionalRule struct {
	Condition string      `json:"condition" validate:"required"`
	Value     interface{} `json:"value"`
	Transform string      `json:"transform,omitempty"`
}

// FieldValidationRule 字段验证规则
type FieldValidationRule struct {
	Pattern       string        `json:"pattern,omitempty"`
	MinLength     int           `json:"min_length,omitempty"`
	MaxLength     int           `json:"max_length,omitempty"`
	MinValue      interface{}   `json:"min_value,omitempty"`
	MaxValue      interface{}   `json:"max_value,omitempty"`
	AllowedValues []interface{} `json:"allowed_values,omitempty"`
	CustomRules   []string      `json:"custom_rules,omitempty"`
}

// TypeConversionConfig 类型转换配置
type TypeConversionConfig struct {
	AutoConvert  bool             `json:"auto_convert" default:"true"`
	StrictMode   bool             `json:"strict_mode" default:"false"`
	DateFormat   string           `json:"date_format" default:"2006-01-02 15:04:05"`
	NumberFormat string           `json:"number_format,omitempty"`
	BooleanMap   map[string]bool  `json:"boolean_map,omitempty"`
	CustomRules  []ConversionRule `json:"custom_rules,omitempty"`
}

// ConversionRule 转换规则
type ConversionRule struct {
	FromType string `json:"from_type" validate:"required"`
	ToType   string `json:"to_type" validate:"required"`
	Rule     string `json:"rule" validate:"required"`
}

// MappingValidation 映射验证
type MappingValidation struct {
	CheckDuplicates bool     `json:"check_duplicates" default:"true"`
	RequiredFields  []string `json:"required_fields,omitempty"`
	ValidateTypes   bool     `json:"validate_types" default:"true"`
	AllowUnmapped   bool     `json:"allow_unmapped" default:"true"`
}

// GovernanceExecutionConfig 数据治理执行配置
type GovernanceExecutionConfig struct {
	EnableQualityCheck   bool                   `json:"enable_quality_check"`
	EnableCleansing      bool                   `json:"enable_cleansing"`
	EnableMasking        bool                   `json:"enable_masking"`
	StopOnQualityFailure bool                   `json:"stop_on_quality_failure"`
	QualityThreshold     float64                `json:"quality_threshold"`
	BatchSize            int                    `json:"batch_size"`
	MaxRetries           int                    `json:"max_retries"`
	TimeoutSeconds       int                    `json:"timeout_seconds"`
	CustomConfig         map[string]interface{} `json:"custom_config,omitempty"`
}

// GovernanceExecutionResult 数据治理执行结果
type GovernanceExecutionResult struct {
	QualityCheckResults   []QualityCheckResult   `json:"quality_check_results"`
	CleansingResults      []CleansingResult      `json:"cleansing_results"`
	MaskingResults        []MaskingResult        `json:"masking_results"`
	TransformationResults []TransformationResult `json:"transformation_results"`
	ValidationResults     []ValidationResult     `json:"validation_results"`
	OverallQualityScore   float64                `json:"overall_quality_score"`
	TotalProcessedRecords int64                  `json:"total_processed_records"`
	TotalCleansingApplied int64                  `json:"total_cleansing_applied"`
	TotalMaskingApplied   int64                  `json:"total_masking_applied"`
	TotalValidationErrors int64                  `json:"total_validation_errors"`
	ExecutionTime         time.Duration          `json:"execution_time"`
	ComplianceStatus      string                 `json:"compliance_status"`
	Issues                []GovernanceIssue      `json:"issues,omitempty"`
}

// QualityCheckResult 质量检查结果
type QualityCheckResult struct {
	RuleID          string    `json:"rule_id"`
	RuleName        string    `json:"rule_name"`
	RuleType        string    `json:"rule_type"`
	Status          string    `json:"status"` // passed, failed, warning
	Score           float64   `json:"score"`  // 0-1
	TotalRecords    int64     `json:"total_records"`
	PassedRecords   int64     `json:"passed_records"`
	FailedRecords   int64     `json:"failed_records"`
	ThresholdMet    bool      `json:"threshold_met"`
	ExecutionTime   time.Time `json:"execution_time"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Recommendations []string  `json:"recommendations,omitempty"`
}

// CleansingResult 清洗结果
type CleansingResult struct {
	RuleID           string    `json:"rule_id"`
	RuleName         string    `json:"rule_name"`
	RuleType         string    `json:"rule_type"`
	Status           string    `json:"status"` // completed, failed, skipped
	ProcessedRecords int64     `json:"processed_records"`
	CleanedRecords   int64     `json:"cleaned_records"`
	SkippedRecords   int64     `json:"skipped_records"`
	ErrorRecords     int64     `json:"error_records"`
	ExecutionTime    time.Time `json:"execution_time"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	CleansingActions []string  `json:"cleansing_actions,omitempty"`
}

// MaskingResult 脱敏结果
type MaskingResult struct {
	RuleID          string    `json:"rule_id"`
	RuleName        string    `json:"rule_name"`
	MaskingType     string    `json:"masking_type"` // mask, replace, encrypt, pseudonymize
	Status          string    `json:"status"`       // completed, failed, skipped
	ProcessedFields int64     `json:"processed_fields"`
	MaskedFields    int64     `json:"masked_fields"`
	SkippedFields   int64     `json:"skipped_fields"`
	ErrorFields     int64     `json:"error_fields"`
	ExecutionTime   time.Time `json:"execution_time"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	ComplianceLevel string    `json:"compliance_level,omitempty"`
}

// TransformationResult 转换结果
type TransformationResult struct {
	RuleID             string    `json:"rule_id"`
	RuleName           string    `json:"rule_name"`
	TransformationType string    `json:"transformation_type"`
	Status             string    `json:"status"` // completed, failed, skipped
	ProcessedRecords   int64     `json:"processed_records"`
	TransformedRecords int64     `json:"transformed_records"`
	SkippedRecords     int64     `json:"skipped_records"`
	ErrorRecords       int64     `json:"error_records"`
	ExecutionTime      time.Time `json:"execution_time"`
	ErrorMessage       string    `json:"error_message,omitempty"`
}

// ValidationResult 校验结果
type ValidationResult struct {
	RuleID           string    `json:"rule_id"`
	RuleName         string    `json:"rule_name"`
	ValidationType   string    `json:"validation_type"`
	Status           string    `json:"status"` // passed, failed, warning
	TotalRecords     int64     `json:"total_records"`
	ValidRecords     int64     `json:"valid_records"`
	InvalidRecords   int64     `json:"invalid_records"`
	ValidationRate   float64   `json:"validation_rate"` // 0-1
	ExecutionTime    time.Time `json:"execution_time"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	ValidationErrors []string  `json:"validation_errors,omitempty"`
}

// GovernanceIssue 数据治理问题
type GovernanceIssue struct {
	Type        string                 `json:"type"`     // quality, compliance, security
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Description string                 `json:"description"`
	Field       string                 `json:"field,omitempty"`
	Record      string                 `json:"record,omitempty"`
	RuleID      string                 `json:"rule_id,omitempty"`
	Suggestion  string                 `json:"suggestion,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}
