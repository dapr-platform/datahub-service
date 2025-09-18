/*
 * @module service/thematic_sync_types
 * @description 主题同步服务相关的请求和响应类型定义
 * @architecture 数据传输对象 - 定义API请求和响应结构
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow N/A
 * @rules 确保类型定义的一致性和可扩展性
 * @dependencies time, models包
 * @refs thematic_sync_service.go, models/thematic_sync.go
 */

package thematic_library

import (
	"datahub-service/service/models"
	"time"
)

// SQLDataSourceConfig SQL数据源配置
type SQLDataSourceConfig struct {
	LibraryID   string                 `json:"library_id"`
	InterfaceID string                 `json:"interface_id"`
	SQLQuery    string                 `json:"sql_query"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	MaxRows     int                    `json:"max_rows,omitempty"`
}

// CreateThematicSyncTaskRequest 创建主题同步任务请求
type CreateThematicSyncTaskRequest struct {
	ThematicLibraryID   string `json:"thematic_library_id" binding:"required"`
	ThematicInterfaceID string `json:"thematic_interface_id" binding:"required"`
	TaskName            string `json:"task_name" binding:"required"`
	Description         string `json:"description"`

	// 数据源配置 - 两种方式二选一，SQL数据源优先级更高
	SourceLibraries []SourceLibraryConfig `json:"source_libraries,omitempty"`
	DataSourceSQL   []SQLDataSourceConfig `json:"data_source_sql,omitempty"`

	AggregationConfig *AggregationConfig `json:"aggregation_config,omitempty"`
	KeyMatchingRules  *KeyMatchingRules  `json:"key_matching_rules,omitempty"`
	FieldMappingRules *FieldMappingRules `json:"field_mapping_rules,omitempty"`

	// 数据治理规则配置 - 使用数据治理中定义的规则ID
	QualityRuleIDs    []string                   `json:"quality_rule_ids,omitempty"`    // 质量规则ID列表
	CleansingRuleIDs  []string                   `json:"cleansing_rule_ids,omitempty"`  // 清洗规则ID列表
	MaskingRuleIDs    []string                   `json:"masking_rule_ids,omitempty"`    // 脱敏规则ID列表
	TransformRuleIDs  []string                   `json:"transform_rule_ids,omitempty"`  // 转换规则ID列表
	ValidationRuleIDs []string                   `json:"validation_rule_ids,omitempty"` // 校验规则ID列表
	GovernanceConfig  *GovernanceExecutionConfig `json:"governance_config,omitempty"`   // 数据治理执行配置

	ScheduleConfig *ScheduleConfig `json:"schedule_config" binding:"required"`
	CreatedBy      string          `json:"created_by" binding:"required"`
}

// UpdateThematicSyncTaskRequest 更新主题同步任务请求
type UpdateThematicSyncTaskRequest struct {
	TaskName          string             `json:"task_name"`
	Description       string             `json:"description"`
	Status            string             `json:"status"`
	ScheduleConfig    *ScheduleConfig    `json:"schedule_config,omitempty"`
	AggregationConfig *AggregationConfig `json:"aggregation_config,omitempty"`
	KeyMatchingRules  *KeyMatchingRules  `json:"key_matching_rules,omitempty"`
	FieldMappingRules *FieldMappingRules `json:"field_mapping_rules,omitempty"`

	// 数据治理规则配置 - 使用数据治理中定义的规则ID
	QualityRuleIDs    []string                   `json:"quality_rule_ids,omitempty"`    // 质量规则ID列表
	CleansingRuleIDs  []string                   `json:"cleansing_rule_ids,omitempty"`  // 清洗规则ID列表
	MaskingRuleIDs    []string                   `json:"masking_rule_ids,omitempty"`    // 脱敏规则ID列表
	TransformRuleIDs  []string                   `json:"transform_rule_ids,omitempty"`  // 转换规则ID列表
	ValidationRuleIDs []string                   `json:"validation_rule_ids,omitempty"` // 校验规则ID列表
	GovernanceConfig  *GovernanceExecutionConfig `json:"governance_config,omitempty"`   // 数据治理执行配置

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
	ExecutionType string                `json:"execution_type,default=manual"`
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

// GovernanceExecutionConfig 数据治理执行配置
type GovernanceExecutionConfig struct {
	EnableQualityCheck      bool                   `json:"enable_quality_check"`       // 启用质量检查
	EnableCleansing         bool                   `json:"enable_cleansing"`           // 启用数据清洗
	EnableMasking           bool                   `json:"enable_masking"`             // 启用数据脱敏
	EnableTransformation    bool                   `json:"enable_transformation"`      // 启用数据转换
	EnableValidation        bool                   `json:"enable_validation"`          // 启用数据校验
	StopOnQualityFailure    bool                   `json:"stop_on_quality_failure"`    // 质量检查失败时停止
	StopOnValidationFailure bool                   `json:"stop_on_validation_failure"` // 校验失败时停止
	QualityThreshold        float64                `json:"quality_threshold"`          // 质量阈值
	BatchSize               int                    `json:"batch_size"`                 // 批处理大小
	MaxRetries              int                    `json:"max_retries"`                // 最大重试次数
	TimeoutSeconds          int                    `json:"timeout_seconds"`            // 超时时间（秒）
	CustomConfig            map[string]interface{} `json:"custom_config,omitempty"`    // 自定义配置
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
