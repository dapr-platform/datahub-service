/*
 * @module service/governance/types
 * @description 数据治理相关的类型定义，包含请求响应模型和业务逻辑类型
 * @architecture 服务层 - 类型定义
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow 业务数据结构定义
 * @rules 确保业务逻辑的强类型定义，便于API接口使用
 * @dependencies time
 * @refs service/models/governance.go, service/models/quality_models.go
 */

package governance

import (
	"time"
)

// === 数据质量规则相关类型 ===

// CreateQualityRuleRequest 创建质量规则模板请求
type CreateQualityRuleRequest struct {
	Name          string                 `json:"name" binding:"required" example:"完整性检查模板"`
	Type          string                 `json:"type" binding:"required" example:"completeness" enums:"completeness,accuracy,consistency,validity,uniqueness,timeliness,standardization"`
	Category      string                 `json:"category" binding:"required" example:"basic_quality" enums:"basic_quality,data_cleansing,data_validation"`
	Description   string                 `json:"description" example:"检查数据完整性的通用模板"`
	RuleLogic     map[string]interface{} `json:"rule_logic" binding:"required" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig map[string]interface{} `json:"default_config" swaggertype:"object"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Tags          map[string]interface{} `json:"tags" swaggertype:"object"`
}

// UpdateQualityRuleRequest 更新质量规则模板请求
type UpdateQualityRuleRequest struct {
	Name          string                 `json:"name,omitempty" example:"更新后的模板名称"`
	Description   string                 `json:"description,omitempty" example:"更新后的描述"`
	RuleLogic     map[string]interface{} `json:"rule_logic,omitempty" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	IsEnabled     *bool                  `json:"is_enabled,omitempty" example:"false"`
	Tags          map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// QualityRuleResponse 质量规则模板响应
type QualityRuleResponse struct {
	ID            string                 `json:"id" example:"uuid-123"`
	Name          string                 `json:"name" example:"完整性检查模板"`
	Type          string                 `json:"type" example:"completeness"`
	Category      string                 `json:"category" example:"basic_quality"`
	Description   string                 `json:"description" example:"检查数据完整性的通用模板"`
	RuleLogic     map[string]interface{} `json:"rule_logic"`
	Parameters    map[string]interface{} `json:"parameters"`
	DefaultConfig map[string]interface{} `json:"default_config"`
	IsBuiltIn     bool                   `json:"is_built_in" example:"false"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Version       string                 `json:"version" example:"1.0"`
	Tags          map[string]interface{} `json:"tags"`
	CreatedAt     time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy     string                 `json:"created_by" example:"admin"`
	UpdatedAt     time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy     string                 `json:"updated_by" example:"admin"`
}

// QualityRuleListResponse 质量规则列表响应
type QualityRuleListResponse struct {
	List  []QualityRuleResponse `json:"list"`
	Total int64                 `json:"total" example:"100"`
	Page  int                   `json:"page" example:"1"`
	Size  int                   `json:"size" example:"10"`
}

// === 数据脱敏规则相关类型 ===

// CreateMaskingRuleRequest 创建脱敏规则模板请求
type CreateMaskingRuleRequest struct {
	Name          string                 `json:"name" binding:"required" example:"手机号脱敏模板"`
	MaskingType   string                 `json:"masking_type" binding:"required" example:"mask" enums:"mask,replace,encrypt,pseudonymize"`
	Category      string                 `json:"category" binding:"required" example:"personal_info" enums:"personal_info,financial,medical,business,custom"`
	SecurityLevel string                 `json:"security_level" binding:"required" example:"medium" enums:"low,medium,high,critical"`
	Description   string                 `json:"description" example:"手机号脱敏的通用模板"`
	MaskingLogic  map[string]interface{} `json:"masking_logic" binding:"required" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters" swaggertype:"object"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Tags          map[string]interface{} `json:"tags" swaggertype:"object"`
}

// UpdateMaskingRuleRequest 更新脱敏规则模板请求
type UpdateMaskingRuleRequest struct {
	Name         string                 `json:"name,omitempty" example:"更新后的脱敏模板"`
	Description  string                 `json:"description,omitempty" example:"更新后的描述"`
	MaskingLogic map[string]interface{} `json:"masking_logic,omitempty" swaggertype:"object"`
	Parameters   map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	IsEnabled    *bool                  `json:"is_enabled,omitempty" example:"false"`
	Tags         map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// MaskingRuleResponse 脱敏规则模板响应
type MaskingRuleResponse struct {
	ID            string                 `json:"id" example:"uuid-123"`
	Name          string                 `json:"name" example:"手机号脱敏模板"`
	MaskingType   string                 `json:"masking_type" example:"mask"`
	Category      string                 `json:"category" example:"personal_info"`
	SecurityLevel string                 `json:"security_level" example:"medium"`
	Description   string                 `json:"description" example:"手机号脱敏的通用模板"`
	MaskingLogic  map[string]interface{} `json:"masking_logic"`
	Parameters    map[string]interface{} `json:"parameters"`
	IsBuiltIn     bool                   `json:"is_built_in" example:"false"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Version       string                 `json:"version" example:"1.0"`
	Tags          map[string]interface{} `json:"tags"`
	CreatedAt     time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy     string                 `json:"created_by" example:"admin"`
	UpdatedAt     time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy     string                 `json:"updated_by" example:"admin"`
}

// MaskingRuleListResponse 脱敏规则列表响应
type MaskingRuleListResponse struct {
	List  []MaskingRuleResponse `json:"list"`
	Total int64                 `json:"total" example:"50"`
	Page  int                   `json:"page" example:"1"`
	Size  int                   `json:"size" example:"10"`
}

// === 质量检查相关类型 ===

// RunQualityCheckRequest 执行质量检查请求
type RunQualityCheckRequest struct {
	ObjectID   string `json:"object_id" binding:"required" example:"uuid-123"`
	ObjectType string `json:"object_type" binding:"required" example:"interface" enums:"interface,thematic_interface"`
}

// QualityCheckExecutionResponse 质量检查执行响应
type QualityCheckExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	QualityRuleID   string                 `json:"quality_rule_id" example:"uuid-456"`
	ExecutionID     string                 `json:"execution_id" example:"exec-789"`
	CheckType       string                 `json:"check_type" example:"manual"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:05:00Z"`
	Duration        int64                  `json:"duration" example:"300000"`
	Status          string                 `json:"status" example:"passed"`
	TotalRecords    int64                  `json:"total_records" example:"10000"`
	PassedRecords   int64                  `json:"passed_records" example:"9500"`
	FailedRecords   int64                  `json:"failed_records" example:"500"`
	QualityScore    float64                `json:"quality_score" example:"0.95"`
	ThresholdMet    bool                   `json:"threshold_met" example:"true"`
	CheckResults    map[string]interface{} `json:"check_results" `
	Recommendations map[string]interface{} `json:"recommendations" `
}

// QualityCheckExecutionListResponse 质量检查执行列表响应
type QualityCheckExecutionListResponse struct {
	List  []QualityCheckExecutionResponse `json:"list"`
	Total int64                           `json:"total" example:"200"`
	Page  int                             `json:"page" example:"1"`
	Size  int                             `json:"size" example:"10"`
}

// === 清洗规则相关类型 ===

// CreateCleansingRuleRequest 创建清洗规则模板请求
type CreateCleansingRuleRequest struct {
	Name            string                 `json:"name" binding:"required" example:"邮箱格式标准化模板"`
	Description     string                 `json:"description" example:"统一邮箱格式为小写的通用模板"`
	RuleType        string                 `json:"rule_type" binding:"required" example:"standardization" enums:"standardization,deduplication,validation,transformation,enrichment"`
	Category        string                 `json:"category" binding:"required" example:"data_format" enums:"data_format,data_quality,data_integrity"`
	CleansingLogic  map[string]interface{} `json:"cleansing_logic" binding:"required" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config" swaggertype:"object"`
	ComplexityLevel string                 `json:"complexity_level" example:"medium" enums:"low,medium,high"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Tags            map[string]interface{} `json:"tags" swaggertype:"object"`
}

// UpdateCleansingRuleRequest 更新清洗规则模板请求
type UpdateCleansingRuleRequest struct {
	Name           string                 `json:"name,omitempty" example:"更新后的清洗规则模板"`
	Description    string                 `json:"description,omitempty" example:"更新后的描述"`
	CleansingLogic map[string]interface{} `json:"cleansing_logic,omitempty" swaggertype:"object"`
	Parameters     map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig  map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	IsEnabled      *bool                  `json:"is_enabled,omitempty" example:"false"`
	Tags           map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// CleansingRuleResponse 清洗规则模板响应
type CleansingRuleResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	Name            string                 `json:"name" example:"邮箱格式标准化模板"`
	Description     string                 `json:"description" example:"统一邮箱格式为小写的通用模板"`
	RuleType        string                 `json:"rule_type" example:"standardization"`
	Category        string                 `json:"category" example:"data_format"`
	CleansingLogic  map[string]interface{} `json:"cleansing_logic"`
	Parameters      map[string]interface{} `json:"parameters"`
	DefaultConfig   map[string]interface{} `json:"default_config"`
	ComplexityLevel string                 `json:"complexity_level" example:"medium"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"false"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy       string                 `json:"created_by" example:"admin"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy       string                 `json:"updated_by" example:"admin"`
}

// CleansingRuleListResponse 清洗规则列表响应
type CleansingRuleListResponse struct {
	List  []CleansingRuleResponse `json:"list"`
	Total int64                   `json:"total" example:"30"`
	Page  int                     `json:"page" example:"1"`
	Size  int                     `json:"size" example:"10"`
}

// === 质量报告相关类型 ===

// GenerateQualityReportRequest 生成质量报告请求
type GenerateQualityReportRequest struct {
	ReportType string `json:"report_type" binding:"required" example:"daily" enums:"daily,weekly,monthly,ad_hoc"`
	StartDate  string `json:"start_date" binding:"required" example:"2024-01-01" format:"date"`
	EndDate    string `json:"end_date" binding:"required" example:"2024-01-31" format:"date"`
	ObjectType string `json:"object_type,omitempty" example:"interface" enums:"interface,thematic_interface"`
	ObjectID   string `json:"object_id,omitempty" example:"uuid-123"`
}

// QualityReportResponse 质量报告响应
type QualityReportResponse struct {
	ID                string                 `json:"id" example:"uuid-123"`
	ReportName        string                 `json:"report_name" example:"2024年1月数据质量月报"`
	RelatedObjectID   string                 `json:"related_object_id" example:"uuid-456"`
	RelatedObjectType string                 `json:"related_object_type" example:"interface"`
	QualityScore      float64                `json:"quality_score" example:"85.5"`
	QualityMetrics    map[string]interface{} `json:"quality_metrics" `
	Issues            map[string]interface{} `json:"issues" `
	Recommendations   map[string]interface{} `json:"recommendations" `
	GeneratedAt       time.Time              `json:"generated_at" example:"2024-01-01T00:00:00Z"`
	GeneratedBy       string                 `json:"generated_by" example:"system"`
}

// QualityReportListResponse 质量报告列表响应
type QualityReportListResponse struct {
	List  []QualityReportResponse `json:"list"`
	Total int64                   `json:"total" example:"20"`
	Page  int                     `json:"page" example:"1"`
	Size  int                     `json:"size" example:"10"`
}

// === 质量指标相关类型 ===

// QualityMetricResponse 质量指标响应
type QualityMetricResponse struct {
	ID              string    `json:"id" example:"uuid-123"`
	MetricName      string    `json:"metric_name" example:"用户表完整性"`
	MetricType      string    `json:"metric_type" example:"completeness"`
	TargetTable     string    `json:"target_table" example:"users"`
	TargetColumn    string    `json:"target_column" example:"email"`
	MetricDate      time.Time `json:"metric_date" example:"2024-01-01"`
	MetricValue     float64   `json:"metric_value" example:"0.95"`
	BaselineValue   float64   `json:"baseline_value" example:"0.90"`
	TargetValue     float64   `json:"target_value" example:"0.95"`
	Trend           string    `json:"trend" example:"improving"`
	TotalCount      int64     `json:"total_count" example:"10000"`
	ValidCount      int64     `json:"valid_count" example:"9500"`
	InvalidCount    int64     `json:"invalid_count" example:"500"`
	CalculationTime time.Time `json:"calculation_time" example:"2024-01-01T00:00:00Z"`
}

// QualityMetricListResponse 质量指标列表响应
type QualityMetricListResponse struct {
	List  []QualityMetricResponse `json:"list"`
	Total int64                   `json:"total" example:"100"`
	Page  int                     `json:"page" example:"1"`
	Size  int                     `json:"size" example:"10"`
}

// === 质量问题相关类型 ===

// QualityIssueResponse 质量问题响应
type QualityIssueResponse struct {
	ID               string                 `json:"id" example:"uuid-123"`
	QualityCheckID   string                 `json:"quality_check_id" example:"uuid-456"`
	QualityRuleID    string                 `json:"quality_rule_id" example:"uuid-789"`
	IssueType        string                 `json:"issue_type" example:"missing_value"`
	Severity         string                 `json:"severity" example:"medium"`
	TargetTable      string                 `json:"target_table" example:"users"`
	TargetColumn     string                 `json:"target_column" example:"email"`
	RecordIdentifier string                 `json:"record_identifier" example:"user_id=123"`
	IssueDescription string                 `json:"issue_description" example:"邮箱字段为空"`
	ExpectedValue    string                 `json:"expected_value" example:"有效邮箱地址"`
	ActualValue      string                 `json:"actual_value" example:"null"`
	IssueContext     map[string]interface{} `json:"issue_context" swaggertype:"object"`
	DetectionTime    time.Time              `json:"detection_time" example:"2024-01-01T00:00:00Z"`
	Status           string                 `json:"status" example:"open"`
	AssignedTo       string                 `json:"assigned_to" example:"admin"`
	ResolutionNote   string                 `json:"resolution_note" example:"已修复"`
	ResolvedBy       string                 `json:"resolved_by" example:"admin"`
	ResolvedAt       *time.Time             `json:"resolved_at,omitempty" example:"2024-01-01T01:00:00Z"`
}

// QualityIssueListResponse 质量问题列表响应
type QualityIssueListResponse struct {
	List  []QualityIssueResponse `json:"list"`
	Total int64                  `json:"total" example:"50"`
	Page  int                    `json:"page" example:"1"`
	Size  int                    `json:"size" example:"10"`
}

// ResolveQualityIssueRequest 解决质量问题请求
type ResolveQualityIssueRequest struct {
	ResolutionNote   string                 `json:"resolution_note" binding:"required" example:"已手动修复缺失的邮箱地址"`
	ResolutionAction map[string]interface{} `json:"resolution_action" swaggertype:"object"`
}

// === 系统日志相关类型 ===

// SystemLogResponse 系统日志响应
type SystemLogResponse struct {
	ID               string                 `json:"id" example:"uuid-123"`
	OperationType    string                 `json:"operation_type" example:"create"`
	ObjectType       string                 `json:"object_type" example:"quality_rule"`
	ObjectID         string                 `json:"object_id" example:"uuid-456"`
	OperatorID       string                 `json:"operator_id" example:"uuid-789"`
	OperatorName     string                 `json:"operator_name" example:"admin"`
	OperatorIP       string                 `json:"operator_ip" example:"192.168.1.1"`
	OperationContent map[string]interface{} `json:"operation_content" `
	OperationTime    time.Time              `json:"operation_time" example:"2024-01-01T00:00:00Z"`
	OperationResult  string                 `json:"operation_result" example:"success"`
}

// SystemLogListResponse 系统日志列表响应
type SystemLogListResponse struct {
	List  []SystemLogResponse `json:"list"`
	Total int64               `json:"total" example:"1000"`
	Page  int                 `json:"page" example:"1"`
	Size  int                 `json:"size" example:"10"`
}

// === 质量检测任务相关类型 ===

// CreateQualityTaskRequest 创建质量检测任务请求
type CreateQualityTaskRequest struct {
	Name               string                 `json:"name" binding:"required" example:"用户表质量检测任务"`
	Description        string                 `json:"description" example:"定期检测用户表数据质量"`
	TaskType           string                 `json:"task_type" binding:"required" example:"scheduled" enums:"scheduled,manual,realtime"`
	TargetObjectID     string                 `json:"target_object_id" binding:"required" example:"uuid-123"`
	TargetObjectType   string                 `json:"target_object_type" binding:"required" example:"interface" enums:"interface,thematic_interface,table"`
	QualityRuleIDs     []string               `json:"quality_rule_ids" example:"[\"uuid-456\",\"uuid-789\"]"`
	ScheduleConfig     map[string]interface{} `json:"schedule_config,omitempty" swaggertype:"object"`
	NotificationConfig map[string]interface{} `json:"notification_config,omitempty" swaggertype:"object"`
	Priority           int                    `json:"priority" example:"50"`
	IsEnabled          bool                   `json:"is_enabled" example:"true"`
}

// QualityTaskResponse 质量检测任务响应
type QualityTaskResponse struct {
	ID                 string                 `json:"id" example:"uuid-123"`
	Name               string                 `json:"name" example:"用户表质量检测任务"`
	Description        string                 `json:"description" example:"定期检测用户表数据质量"`
	TaskType           string                 `json:"task_type" example:"scheduled"`
	TargetObjectID     string                 `json:"target_object_id" example:"uuid-456"`
	TargetObjectType   string                 `json:"target_object_type" example:"interface"`
	QualityRuleIDs     []string               `json:"quality_rule_ids" example:"[\"uuid-789\"]"`
	ScheduleConfig     map[string]interface{} `json:"schedule_config" swaggertype:"object"`
	NotificationConfig map[string]interface{} `json:"notification_config" swaggertype:"object"`
	Status             string                 `json:"status" example:"pending"`
	Priority           int                    `json:"priority" example:"50"`
	IsEnabled          bool                   `json:"is_enabled" example:"true"`
	LastExecuted       *time.Time             `json:"last_executed,omitempty" example:"2024-01-01T00:00:00Z"`
	NextExecution      *time.Time             `json:"next_execution,omitempty" example:"2024-01-02T02:00:00Z"`
	ExecutionCount     int64                  `json:"execution_count" example:"10"`
	SuccessCount       int64                  `json:"success_count" example:"8"`
	FailureCount       int64                  `json:"failure_count" example:"2"`
	CreatedAt          time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy          string                 `json:"created_by" example:"admin"`
	UpdatedAt          time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy          string                 `json:"updated_by" example:"admin"`
}

// QualityTaskListResponse 质量检测任务列表响应
type QualityTaskListResponse struct {
	List  []QualityTaskResponse `json:"list"`
	Total int64                 `json:"total" example:"25"`
	Page  int                   `json:"page" example:"1"`
	Size  int                   `json:"size" example:"10"`
}

// QualityTaskExecutionResponse 质量检测任务执行响应
type QualityTaskExecutionResponse struct {
	ID                 string                 `json:"id" example:"uuid-123"`
	TaskID             string                 `json:"task_id" example:"uuid-456"`
	ExecutionType      string                 `json:"execution_type" example:"manual"`
	StartTime          time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime            *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:05:00Z"`
	Duration           int64                  `json:"duration" example:"300000"`
	Status             string                 `json:"status" example:"completed"`
	TotalRulesExecuted int                    `json:"total_rules_executed" example:"5"`
	PassedRules        int                    `json:"passed_rules" example:"4"`
	FailedRules        int                    `json:"failed_rules" example:"1"`
	OverallScore       float64                `json:"overall_score" example:"0.8"`
	ExecutionResults   map[string]interface{} `json:"execution_results" swaggertype:"object"`
	TriggerSource      string                 `json:"trigger_source" example:"manual"`
	ExecutedBy         string                 `json:"executed_by" example:"admin"`
}

// QualityTaskExecutionListResponse 质量检测任务执行列表响应
type QualityTaskExecutionListResponse struct {
	List  []QualityTaskExecutionResponse `json:"list"`
	Total int64                          `json:"total" example:"50"`
	Page  int                            `json:"page" example:"1"`
	Size  int                            `json:"size" example:"10"`
}

// === 数据血缘相关类型 ===

// CreateDataLineageRequest 创建数据血缘关系请求
type CreateDataLineageRequest struct {
	SourceObjectID   string                 `json:"source_object_id" binding:"required" example:"uuid-123"`
	SourceObjectType string                 `json:"source_object_type" binding:"required" example:"table" enums:"table,interface,thematic_interface"`
	TargetObjectID   string                 `json:"target_object_id" binding:"required" example:"uuid-456"`
	TargetObjectType string                 `json:"target_object_type" binding:"required" example:"interface"`
	RelationType     string                 `json:"relation_type" binding:"required" example:"direct" enums:"direct,derived,aggregated,transformed"`
	TransformRule    map[string]interface{} `json:"transform_rule,omitempty" swaggertype:"object"`
	ColumnMapping    map[string]interface{} `json:"column_mapping,omitempty" swaggertype:"object"`
	Confidence       float64                `json:"confidence" example:"0.95"`
	Description      string                 `json:"description" example:"用户表到用户接口的直接血缘关系"`
}

// DataLineageResponse 数据血缘关系响应
type DataLineageResponse struct {
	ID               string                 `json:"id" example:"uuid-123"`
	SourceObjectID   string                 `json:"source_object_id" example:"uuid-456"`
	SourceObjectType string                 `json:"source_object_type" example:"table"`
	TargetObjectID   string                 `json:"target_object_id" example:"uuid-789"`
	TargetObjectType string                 `json:"target_object_type" example:"interface"`
	RelationType     string                 `json:"relation_type" example:"direct"`
	TransformRule    map[string]interface{} `json:"transform_rule" swaggertype:"object"`
	ColumnMapping    map[string]interface{} `json:"column_mapping" swaggertype:"object"`
	Confidence       float64                `json:"confidence" example:"0.95"`
	IsActive         bool                   `json:"is_active" example:"true"`
	Description      string                 `json:"description" example:"用户表到用户接口的直接血缘关系"`
	CreatedAt        time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy        string                 `json:"created_by" example:"admin"`
	UpdatedAt        time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy        string                 `json:"updated_by" example:"admin"`
}

// DataLineageNode 数据血缘节点
type DataLineageNode struct {
	ID         string                 `json:"id" example:"uuid-123"`
	Type       string                 `json:"type" example:"table"`
	Name       string                 `json:"name" example:"users"`
	Properties map[string]interface{} `json:"properties" swaggertype:"object"`
}

// DataLineageEdge 数据血缘边
type DataLineageEdge struct {
	ID           string                 `json:"id" example:"uuid-456"`
	SourceID     string                 `json:"source_id" example:"uuid-123"`
	TargetID     string                 `json:"target_id" example:"uuid-789"`
	RelationType string                 `json:"relation_type" example:"direct"`
	Properties   map[string]interface{} `json:"properties" swaggertype:"object"`
}

// DataLineageGraphResponse 数据血缘图响应
type DataLineageGraphResponse struct {
	Nodes []DataLineageNode `json:"nodes"`
	Edges []DataLineageEdge `json:"edges"`
	Stats struct {
		TotalNodes int `json:"total_nodes" example:"10"`
		TotalEdges int `json:"total_edges" example:"8"`
		MaxDepth   int `json:"max_depth" example:"3"`
	} `json:"stats"`
}

// === 转换规则相关类型 ===

// CreateTransformationRuleRequest 创建转换规则请求
type CreateTransformationRuleRequest struct {
	Name             string                 `json:"name" binding:"required" example:"用户数据转换规则"`
	Description      string                 `json:"description" example:"将用户原始数据转换为标准格式"`
	RuleType         string                 `json:"rule_type" binding:"required" example:"format" enums:"format,calculate,aggregate,filter,join"`
	SourceObjectID   string                 `json:"source_object_id" binding:"required" example:"uuid-123"`
	SourceObjectType string                 `json:"source_object_type" binding:"required" example:"table"`
	TargetObjectID   string                 `json:"target_object_id" binding:"required" example:"uuid-456"`
	TargetObjectType string                 `json:"target_object_type" binding:"required" example:"interface"`
	TransformLogic   map[string]interface{} `json:"transform_logic" binding:"required" swaggertype:"object"`
	InputSchema      map[string]interface{} `json:"input_schema,omitempty" swaggertype:"object"`
	OutputSchema     map[string]interface{} `json:"output_schema,omitempty" swaggertype:"object"`
	ValidationRules  map[string]interface{} `json:"validation_rules,omitempty" swaggertype:"object"`
	ErrorHandling    map[string]interface{} `json:"error_handling,omitempty" swaggertype:"object"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
	ExecutionOrder   int                    `json:"execution_order" example:"1"`
}

// UpdateTransformationRuleRequest 更新转换规则请求
type UpdateTransformationRuleRequest struct {
	Name            string                 `json:"name,omitempty" example:"更新后的转换规则"`
	Description     string                 `json:"description,omitempty" example:"更新后的描述"`
	TransformLogic  map[string]interface{} `json:"transform_logic,omitempty" swaggertype:"object"`
	ValidationRules map[string]interface{} `json:"validation_rules,omitempty" swaggertype:"object"`
	ErrorHandling   map[string]interface{} `json:"error_handling,omitempty" swaggertype:"object"`
	IsEnabled       *bool                  `json:"is_enabled,omitempty" example:"false"`
	ExecutionOrder  *int                   `json:"execution_order,omitempty" example:"2"`
}

// TransformationRuleResponse 转换规则响应
type TransformationRuleResponse struct {
	ID               string                 `json:"id" example:"uuid-123"`
	Name             string                 `json:"name" example:"用户数据转换规则"`
	Description      string                 `json:"description" example:"将用户原始数据转换为标准格式"`
	RuleType         string                 `json:"rule_type" example:"format"`
	SourceObjectID   string                 `json:"source_object_id" example:"uuid-456"`
	SourceObjectType string                 `json:"source_object_type" example:"table"`
	TargetObjectID   string                 `json:"target_object_id" example:"uuid-789"`
	TargetObjectType string                 `json:"target_object_type" example:"interface"`
	TransformLogic   map[string]interface{} `json:"transform_logic" swaggertype:"object"`
	InputSchema      map[string]interface{} `json:"input_schema" swaggertype:"object"`
	OutputSchema     map[string]interface{} `json:"output_schema" swaggertype:"object"`
	ValidationRules  map[string]interface{} `json:"validation_rules" swaggertype:"object"`
	ErrorHandling    map[string]interface{} `json:"error_handling" swaggertype:"object"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
	ExecutionOrder   int                    `json:"execution_order" example:"1"`
	SuccessCount     int64                  `json:"success_count" example:"1000"`
	FailureCount     int64                  `json:"failure_count" example:"10"`
	LastExecuted     *time.Time             `json:"last_executed,omitempty" example:"2024-01-01T00:00:00Z"`
	CreatedAt        time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy        string                 `json:"created_by" example:"admin"`
	UpdatedAt        time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy        string                 `json:"updated_by" example:"admin"`
}

// TransformationRuleListResponse 转换规则列表响应
type TransformationRuleListResponse struct {
	List  []TransformationRuleResponse `json:"list"`
	Total int64                        `json:"total" example:"15"`
	Page  int                          `json:"page" example:"1"`
	Size  int                          `json:"size" example:"10"`
}

// TransformationExecutionResponse 转换执行响应
type TransformationExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	RuleID          string                 `json:"rule_id" example:"uuid-456"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:01:00Z"`
	Duration        int64                  `json:"duration" example:"60000"`
	Status          string                 `json:"status" example:"completed"`
	ProcessedCount  int64                  `json:"processed_count" example:"1000"`
	SuccessCount    int64                  `json:"success_count" example:"950"`
	FailureCount    int64                  `json:"failure_count" example:"50"`
	ExecutionResult map[string]interface{} `json:"execution_result" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// === 校验规则相关类型 ===

// CreateValidationRuleRequest 创建校验规则请求
type CreateValidationRuleRequest struct {
	Name             string                 `json:"name" binding:"required" example:"邮箱格式校验"`
	Description      string                 `json:"description" example:"验证邮箱字段格式是否正确"`
	RuleType         string                 `json:"rule_type" binding:"required" example:"regex" enums:"format,range,enum,regex,custom,reference"`
	TargetObjectID   string                 `json:"target_object_id" binding:"required" example:"uuid-123"`
	TargetObjectType string                 `json:"target_object_type" binding:"required" example:"interface"`
	TargetColumn     string                 `json:"target_column,omitempty" example:"email"`
	ValidationLogic  map[string]interface{} `json:"validation_logic" binding:"required" swaggertype:"object"`
	ErrorMessage     string                 `json:"error_message,omitempty" example:"邮箱格式不正确"`
	Severity         string                 `json:"severity" example:"medium" enums:"low,medium,high,critical"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
	StopOnFailure    bool                   `json:"stop_on_failure" example:"false"`
	Priority         int                    `json:"priority" example:"50"`
}

// UpdateValidationRuleRequest 更新校验规则请求
type UpdateValidationRuleRequest struct {
	Name            string                 `json:"name,omitempty" example:"更新后的校验规则"`
	Description     string                 `json:"description,omitempty" example:"更新后的描述"`
	ValidationLogic map[string]interface{} `json:"validation_logic,omitempty" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty" example:"更新后的错误消息"`
	Severity        string                 `json:"severity,omitempty" example:"high"`
	IsEnabled       *bool                  `json:"is_enabled,omitempty" example:"false"`
	StopOnFailure   *bool                  `json:"stop_on_failure,omitempty" example:"true"`
	Priority        *int                   `json:"priority,omitempty" example:"80"`
}

// ValidationRuleResponse 校验规则响应
type ValidationRuleResponse struct {
	ID               string                 `json:"id" example:"uuid-123"`
	Name             string                 `json:"name" example:"邮箱格式校验"`
	Description      string                 `json:"description" example:"验证邮箱字段格式是否正确"`
	RuleType         string                 `json:"rule_type" example:"regex"`
	TargetObjectID   string                 `json:"target_object_id" example:"uuid-456"`
	TargetObjectType string                 `json:"target_object_type" example:"interface"`
	TargetColumn     string                 `json:"target_column" example:"email"`
	ValidationLogic  map[string]interface{} `json:"validation_logic" swaggertype:"object"`
	ErrorMessage     string                 `json:"error_message" example:"邮箱格式不正确"`
	Severity         string                 `json:"severity" example:"medium"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
	StopOnFailure    bool                   `json:"stop_on_failure" example:"false"`
	Priority         int                    `json:"priority" example:"50"`
	SuccessCount     int64                  `json:"success_count" example:"5000"`
	FailureCount     int64                  `json:"failure_count" example:"100"`
	LastExecuted     *time.Time             `json:"last_executed,omitempty" example:"2024-01-01T00:00:00Z"`
	CreatedAt        time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy        string                 `json:"created_by" example:"admin"`
	UpdatedAt        time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy        string                 `json:"updated_by" example:"admin"`
}

// ValidationRuleListResponse 校验规则列表响应
type ValidationRuleListResponse struct {
	List  []ValidationRuleResponse `json:"list"`
	Total int64                    `json:"total" example:"40"`
	Page  int                      `json:"page" example:"1"`
	Size  int                      `json:"size" example:"10"`
}

// ValidationExecutionResponse 校验执行响应
type ValidationExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	RuleID          string                 `json:"rule_id" example:"uuid-456"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:00:30Z"`
	Duration        int64                  `json:"duration" example:"30000"`
	Status          string                 `json:"status" example:"completed"`
	TotalRecords    int64                  `json:"total_records" example:"10000"`
	ValidRecords    int64                  `json:"valid_records" example:"9500"`
	InvalidRecords  int64                  `json:"invalid_records" example:"500"`
	ValidationRate  float64                `json:"validation_rate" example:"0.95"`
	ExecutionResult map[string]interface{} `json:"execution_result" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// CleansingExecutionResponse 清洗执行响应
type CleansingExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	RuleID          string                 `json:"rule_id" example:"uuid-456"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:02:00Z"`
	Duration        int64                  `json:"duration" example:"120000"`
	Status          string                 `json:"status" example:"completed"`
	TotalRecords    int64                  `json:"total_records" example:"10000"`
	CleanedRecords  int64                  `json:"cleaned_records" example:"8000"`
	SkippedRecords  int64                  `json:"skipped_records" example:"2000"`
	CleansingRate   float64                `json:"cleansing_rate" example:"0.8"`
	ExecutionResult map[string]interface{} `json:"execution_result" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// === 质量规则模板相关类型 ===

// CreateQualityRuleTemplateRequest 创建质量规则模板请求
type CreateQualityRuleTemplateRequest struct {
	Name          string                 `json:"name" binding:"required" example:"字段完整性检查模板"`
	Type          string                 `json:"type" binding:"required" example:"completeness" enums:"completeness,accuracy,consistency,validity,uniqueness,timeliness,standardization"`
	Category      string                 `json:"category" binding:"required" example:"basic_quality" enums:"basic_quality,data_cleansing,data_validation"`
	Description   string                 `json:"description" example:"检查指定字段的完整性，支持配置阈值"`
	RuleLogic     map[string]interface{} `json:"rule_logic" binding:"required" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	IsBuiltIn     bool                   `json:"is_built_in" example:"false"`
	Version       string                 `json:"version" example:"1.0"`
	Tags          map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// UpdateQualityRuleTemplateRequest 更新质量规则模板请求
type UpdateQualityRuleTemplateRequest struct {
	Name          string                 `json:"name,omitempty" example:"更新后的模板名称"`
	Description   string                 `json:"description,omitempty" example:"更新后的描述"`
	RuleLogic     map[string]interface{} `json:"rule_logic,omitempty" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	IsEnabled     *bool                  `json:"is_enabled,omitempty" example:"true"`
	Tags          map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// QualityRuleTemplateResponse 质量规则模板响应
type QualityRuleTemplateResponse struct {
	ID            string                 `json:"id" example:"uuid-123"`
	Name          string                 `json:"name" example:"字段完整性检查模板"`
	Type          string                 `json:"type" example:"completeness"`
	Category      string                 `json:"category" example:"basic_quality"`
	Description   string                 `json:"description" example:"检查指定字段的完整性，支持配置阈值"`
	RuleLogic     map[string]interface{} `json:"rule_logic" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig map[string]interface{} `json:"default_config" swaggertype:"object"`
	IsBuiltIn     bool                   `json:"is_built_in" example:"false"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Version       string                 `json:"version" example:"1.0"`
	Tags          map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedAt     time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy     string                 `json:"created_by" example:"admin"`
	UpdatedAt     time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy     string                 `json:"updated_by" example:"admin"`
}

// QualityRuleTemplateListResponse 质量规则模板列表响应
type QualityRuleTemplateListResponse struct {
	List  []QualityRuleTemplateResponse `json:"list"`
	Total int64                         `json:"total" example:"50"`
	Page  int                           `json:"page" example:"1"`
	Size  int                           `json:"size" example:"10"`
}

// === 规则直接应用相关类型 ===

// ApplyDataGovernanceRulesRequest 应用数据治理规则请求
type ApplyDataGovernanceRulesRequest struct {
	Data           map[string]interface{}       `json:"data" binding:"required" swaggertype:"object"`
	QualityRules   []QualityRuleConfigRequest   `json:"quality_rules,omitempty"`
	MaskingRules   []DataMaskingConfigRequest   `json:"masking_rules,omitempty"`
	CleansingRules []DataCleansingConfigRequest `json:"cleansing_rules,omitempty"`
	Options        *GovernanceExecutionOptions  `json:"options,omitempty"`
}

// QualityRuleConfigRequest 质量规则配置请求
type QualityRuleConfigRequest struct {
	RuleTemplateID string                 `json:"rule_template_id" binding:"required" example:"uuid-123"`
	TargetFields   []string               `json:"target_fields" binding:"required" example:"[\"name\",\"email\"]"`
	RuntimeConfig  map[string]interface{} `json:"runtime_config,omitempty" swaggertype:"object"`
	Threshold      map[string]interface{} `json:"threshold,omitempty" swaggertype:"object"`
	IsEnabled      bool                   `json:"is_enabled" example:"true"`
}

// DataMaskingConfigRequest 数据脱敏配置请求
type DataMaskingConfigRequest struct {
	TemplateID       string                 `json:"template_id" binding:"required" example:"uuid-123"`
	TargetFields     []string               `json:"target_fields" binding:"required" example:"[\"phone\",\"id_card\"]"`
	MaskingConfig    map[string]interface{} `json:"masking_config" binding:"required" swaggertype:"object"`
	ApplyCondition   string                 `json:"apply_condition,omitempty" example:"user_type = 'normal'"`
	PreserveFormat   bool                   `json:"preserve_format" example:"true"`
	ReversibleConfig map[string]interface{} `json:"reversible_config,omitempty" swaggertype:"object"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
}

// DataCleansingConfigRequest 数据清洗配置请求
type DataCleansingConfigRequest struct {
	TemplateID       string                 `json:"template_id" binding:"required" example:"uuid-123"`
	TargetFields     []string               `json:"target_fields" binding:"required" example:"[\"email\",\"name\"]"`
	TriggerCondition string                 `json:"trigger_condition,omitempty" example:"email IS NOT NULL"`
	CleansingConfig  map[string]interface{} `json:"cleansing_config" binding:"required" swaggertype:"object"`
	PreCondition     string                 `json:"pre_condition,omitempty" example:"LENGTH(email) > 0"`
	PostCondition    string                 `json:"post_condition,omitempty" example:"email LIKE '%@%'"`
	BackupOriginal   bool                   `json:"backup_original" example:"true"`
	ValidationRules  map[string]interface{} `json:"validation_rules,omitempty" swaggertype:"object"`
	ErrorHandling    map[string]interface{} `json:"error_handling,omitempty" swaggertype:"object"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
}

// GovernanceExecutionOptions 治理执行选项
type GovernanceExecutionOptions struct {
	StopOnFirstError     bool    `json:"stop_on_first_error" example:"false"`
	SkipQualityCheck     bool    `json:"skip_quality_check" example:"false"`
	SkipMasking          bool    `json:"skip_masking" example:"false"`
	SkipCleansing        bool    `json:"skip_cleansing" example:"false"`
	QualityThreshold     float64 `json:"quality_threshold" example:"0.8"`
	EnableDetailedReport bool    `json:"enable_detailed_report" example:"true"`
}

// ApplyDataGovernanceRulesResponse 应用数据治理规则响应
type ApplyDataGovernanceRulesResponse struct {
	Success             bool                   `json:"success" example:"true"`
	ProcessedData       map[string]interface{} `json:"processed_data" swaggertype:"object"`
	QualityResult       *QualityCheckResult    `json:"quality_result,omitempty"`
	MaskingResult       *MaskingResult         `json:"masking_result,omitempty"`
	CleansingResult     *CleansingResult       `json:"cleansing_result,omitempty"`
	OverallQualityScore float64                `json:"overall_quality_score" example:"0.95"`
	TotalIssues         int                    `json:"total_issues" example:"2"`
	ExecutionTime       int64                  `json:"execution_time" example:"150"`
	RulesApplied        []string               `json:"rules_applied" example:"[\"completeness:name\",\"mask:phone\"]"`
	ErrorMessage        string                 `json:"error_message,omitempty"`
}

// QualityCheckResult 质量检查结果
type QualityCheckResult struct {
	QualityScore  float64  `json:"quality_score" example:"0.95"`
	Issues        []string `json:"issues" example:"[\"字段 email 为空\"]"`
	ChecksPassed  int      `json:"checks_passed" example:"8"`
	ChecksTotal   int      `json:"checks_total" example:"10"`
	RulesApplied  []string `json:"rules_applied" example:"[\"completeness:name\",\"accuracy:email\"]"`
	ExecutionTime int64    `json:"execution_time" example:"50"`
}

// MaskingResult 脱敏结果
type MaskingResult struct {
	FieldsProcessed []string               `json:"fields_processed" example:"[\"phone\",\"id_card\"]"`
	Modifications   map[string]interface{} `json:"modifications" swaggertype:"object"`
	RulesApplied    []string               `json:"rules_applied" example:"[\"mask:phone\",\"replace:id_card\"]"`
	ExecutionTime   int64                  `json:"execution_time" example:"30"`
}

// CleansingResult 清洗结果
type CleansingResult struct {
	FieldsCleaned     []string               `json:"fields_cleaned" example:"[\"email\",\"name\"]"`
	Modifications     map[string]interface{} `json:"modifications" swaggertype:"object"`
	RulesApplied      []string               `json:"rules_applied" example:"[\"standardization:email\"]"`
	ValidationsPassed int                    `json:"validations_passed" example:"5"`
	ValidationsFailed int                    `json:"validations_failed" example:"1"`
	ExecutionTime     int64                  `json:"execution_time" example:"70"`
}

// === 同步任务治理配置类型 ===

// SyncTaskGovernanceConfig 同步任务治理配置
type SyncTaskGovernanceConfig struct {
	ApplyQualityRules   bool                         `json:"apply_quality_rules" example:"true"`
	ApplyMaskingRules   bool                         `json:"apply_masking_rules" example:"true"`
	ApplyCleansingRules bool                         `json:"apply_cleansing_rules" example:"true"`
	QualityThreshold    float64                      `json:"quality_threshold" example:"0.8"`
	StopOnQualityFail   bool                         `json:"stop_on_quality_fail" example:"false"`
	QualityRules        []QualityRuleConfigRequest   `json:"quality_rules,omitempty"`
	MaskingRules        []DataMaskingConfigRequest   `json:"masking_rules,omitempty"`
	CleansingRules      []DataCleansingConfigRequest `json:"cleansing_rules,omitempty"`
}

// QualityCheckTaskConfig 质量检查任务配置
type QualityCheckTaskConfig struct {
	TaskName           string                     `json:"task_name" binding:"required" example:"用户数据质量检查"`
	TargetObjectID     string                     `json:"target_object_id" binding:"required" example:"uuid-123"`
	TargetObjectType   string                     `json:"target_object_type" binding:"required" example:"interface"`
	Schedule           string                     `json:"schedule" example:"0 2 * * *"`
	QualityRules       []QualityRuleConfigRequest `json:"quality_rules" binding:"required"`
	QualityThreshold   float64                    `json:"quality_threshold" example:"0.85"`
	NotificationConfig *NotificationConfig        `json:"notification_config,omitempty"`
	IsEnabled          bool                       `json:"is_enabled" example:"true"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled    bool     `json:"enabled" example:"true"`
	Threshold  float64  `json:"threshold" example:"0.85"`
	Recipients []string `json:"recipients" example:"[\"admin@example.com\"]"`
	WebhookURL string   `json:"webhook_url,omitempty" example:"https://hooks.slack.com/xxx"`
}

// BatchProcessingConfig 批量处理配置
type BatchProcessingConfig struct {
	BatchSize      int `json:"batch_size" example:"1000"`
	MaxConcurrency int `json:"max_concurrency" example:"4"`
	TimeoutSeconds int `json:"timeout_seconds" example:"300"`
}

// === 脱敏模板相关类型 ===

// CreateDataMaskingTemplateRequest 创建数据脱敏模板请求
type CreateDataMaskingTemplateRequest struct {
	Name            string                 `json:"name" binding:"required" example:"手机号掩码模板"`
	MaskingType     string                 `json:"masking_type" binding:"required" example:"mask" enums:"mask,replace,encrypt,pseudonymize"`
	Category        string                 `json:"category" binding:"required" example:"personal_info" enums:"personal_info,financial,medical,business,custom"`
	Description     string                 `json:"description" example:"对手机号进行掩码处理，保留前3位和后4位"`
	ApplicableTypes []string               `json:"applicable_types" example:"[\"varchar\",\"char\"]"`
	MaskingLogic    map[string]interface{} `json:"masking_logic" binding:"required" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	SecurityLevel   string                 `json:"security_level" example:"medium" enums:"low,medium,high,critical"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"false"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// UpdateDataMaskingTemplateRequest 更新数据脱敏模板请求
type UpdateDataMaskingTemplateRequest struct {
	Name            string                 `json:"name,omitempty" example:"更新后的模板名称"`
	Description     string                 `json:"description,omitempty" example:"更新后的描述"`
	ApplicableTypes []string               `json:"applicable_types,omitempty" example:"[\"text\"]"`
	MaskingLogic    map[string]interface{} `json:"masking_logic,omitempty" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	SecurityLevel   string                 `json:"security_level,omitempty" example:"high"`
	IsEnabled       *bool                  `json:"is_enabled,omitempty" example:"true"`
	Tags            map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// DataMaskingTemplateResponse 数据脱敏模板响应
type DataMaskingTemplateResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	Name            string                 `json:"name" example:"手机号掩码模板"`
	MaskingType     string                 `json:"masking_type" example:"mask"`
	Category        string                 `json:"category" example:"personal_info"`
	Description     string                 `json:"description" example:"对手机号进行掩码处理，保留前3位和后4位"`
	ApplicableTypes []string               `json:"applicable_types" example:"[\"varchar\",\"char\"]"`
	MaskingLogic    map[string]interface{} `json:"masking_logic" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config" swaggertype:"object"`
	SecurityLevel   string                 `json:"security_level" example:"medium"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"false"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy       string                 `json:"created_by" example:"admin"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy       string                 `json:"updated_by" example:"admin"`
}

// DataMaskingTemplateListResponse 数据脱敏模板列表响应
type DataMaskingTemplateListResponse struct {
	List  []DataMaskingTemplateResponse `json:"list"`
	Total int64                         `json:"total" example:"25"`
	Page  int                           `json:"page" example:"1"`
	Size  int                           `json:"size" example:"10"`
}

// === 脱敏应用相关类型 ===

// CreateDataMaskingApplicationRequest 创建数据脱敏应用请求
type CreateDataMaskingApplicationRequest struct {
	TemplateID       string                 `json:"template_id" binding:"required" example:"uuid-123"`
	Name             string                 `json:"name" binding:"required" example:"用户表手机号脱敏"`
	DataSource       string                 `json:"data_source" binding:"required" example:"user_db"`
	DataTable        string                 `json:"data_table" binding:"required" example:"users"`
	TargetFields     map[string]interface{} `json:"target_fields" binding:"required" swaggertype:"object"`
	MaskingConfig    map[string]interface{} `json:"masking_config" binding:"required" swaggertype:"object"`
	ApplyCondition   string                 `json:"apply_condition,omitempty" example:"user_type = 'normal'"`
	PreserveFormat   bool                   `json:"preserve_format" example:"true"`
	ReversibleConfig map[string]interface{} `json:"reversible_config,omitempty" swaggertype:"object"`
	Priority         int                    `json:"priority" example:"50"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
}

// UpdateDataMaskingApplicationRequest 更新数据脱敏应用请求
type UpdateDataMaskingApplicationRequest struct {
	Name             string                 `json:"name,omitempty" example:"更新后的应用名称"`
	TargetFields     map[string]interface{} `json:"target_fields,omitempty" swaggertype:"object"`
	MaskingConfig    map[string]interface{} `json:"masking_config,omitempty" swaggertype:"object"`
	ApplyCondition   string                 `json:"apply_condition,omitempty" example:"user_type IN ('normal', 'vip')"`
	PreserveFormat   *bool                  `json:"preserve_format,omitempty" example:"false"`
	ReversibleConfig map[string]interface{} `json:"reversible_config,omitempty" swaggertype:"object"`
	Priority         *int                   `json:"priority,omitempty" example:"60"`
	IsEnabled        *bool                  `json:"is_enabled,omitempty" example:"false"`
}

// DataMaskingApplicationResponse 数据脱敏应用响应
type DataMaskingApplicationResponse struct {
	ID               string                       `json:"id" example:"uuid-123"`
	TemplateID       string                       `json:"template_id" example:"uuid-456"`
	Template         *DataMaskingTemplateResponse `json:"template,omitempty"`
	Name             string                       `json:"name" example:"用户表手机号脱敏"`
	DataSource       string                       `json:"data_source" example:"user_db"`
	DataTable        string                       `json:"data_table" example:"users"`
	TargetFields     map[string]interface{}       `json:"target_fields" swaggertype:"object"`
	MaskingConfig    map[string]interface{}       `json:"masking_config" swaggertype:"object"`
	ApplyCondition   string                       `json:"apply_condition" example:"user_type = 'normal'"`
	PreserveFormat   bool                         `json:"preserve_format" example:"true"`
	ReversibleConfig map[string]interface{}       `json:"reversible_config" swaggertype:"object"`
	IsEnabled        bool                         `json:"is_enabled" example:"true"`
	Priority         int                          `json:"priority" example:"50"`
	LastApplied      *time.Time                   `json:"last_applied,omitempty" example:"2024-01-01T00:00:00Z"`
	ApplyCount       int64                        `json:"apply_count" example:"5000"`
	CreatedAt        time.Time                    `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy        string                       `json:"created_by" example:"admin"`
	UpdatedAt        time.Time                    `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy        string                       `json:"updated_by" example:"admin"`
}

// DataMaskingApplicationListResponse 数据脱敏应用列表响应
type DataMaskingApplicationListResponse struct {
	List  []DataMaskingApplicationResponse `json:"list"`
	Total int64                            `json:"total" example:"15"`
	Page  int                              `json:"page" example:"1"`
	Size  int                              `json:"size" example:"10"`
}

// MaskingApplicationExecutionResponse 脱敏应用执行响应
type MaskingApplicationExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	ApplicationID   string                 `json:"application_id" example:"uuid-456"`
	TemplateID      string                 `json:"template_id" example:"uuid-789"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:01:00Z"`
	Duration        int64                  `json:"duration" example:"60000"`
	Status          string                 `json:"status" example:"completed"`
	TotalRecords    int64                  `json:"total_records" example:"5000"`
	MaskedRecords   int64                  `json:"masked_records" example:"4800"`
	SkippedRecords  int64                  `json:"skipped_records" example:"200"`
	MaskingRate     float64                `json:"masking_rate" example:"0.96"`
	ExecutionResult map[string]interface{} `json:"execution_result" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// === 清洗模板相关类型 ===

// CreateDataCleansingTemplateRequest 创建数据清洗模板请求
type CreateDataCleansingTemplateRequest struct {
	Name            string                 `json:"name" binding:"required" example:"邮箱格式标准化模板"`
	Description     string                 `json:"description" example:"统一邮箱格式为小写并验证格式"`
	RuleType        string                 `json:"rule_type" binding:"required" example:"standardization" enums:"standardization,deduplication,validation,transformation,enrichment"`
	Category        string                 `json:"category" binding:"required" example:"data_format" enums:"data_format,data_quality,data_integrity"`
	CleansingLogic  map[string]interface{} `json:"cleansing_logic" binding:"required" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	ApplicableTypes map[string]interface{} `json:"applicable_types,omitempty" swaggertype:"object"`
	ComplexityLevel string                 `json:"complexity_level" example:"medium" enums:"low,medium,high"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"false"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// UpdateDataCleansingTemplateRequest 更新数据清洗模板请求
type UpdateDataCleansingTemplateRequest struct {
	Name            string                 `json:"name,omitempty" example:"更新后的模板名称"`
	Description     string                 `json:"description,omitempty" example:"更新后的描述"`
	CleansingLogic  map[string]interface{} `json:"cleansing_logic,omitempty" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters,omitempty" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config,omitempty" swaggertype:"object"`
	ApplicableTypes map[string]interface{} `json:"applicable_types,omitempty" swaggertype:"object"`
	ComplexityLevel string                 `json:"complexity_level,omitempty" example:"high"`
	IsEnabled       *bool                  `json:"is_enabled,omitempty" example:"true"`
	Tags            map[string]interface{} `json:"tags,omitempty" swaggertype:"object"`
}

// DataCleansingTemplateResponse 数据清洗模板响应
type DataCleansingTemplateResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	Name            string                 `json:"name" example:"邮箱格式标准化模板"`
	Description     string                 `json:"description" example:"统一邮箱格式为小写并验证格式"`
	RuleType        string                 `json:"rule_type" example:"standardization"`
	Category        string                 `json:"category" example:"data_format"`
	CleansingLogic  map[string]interface{} `json:"cleansing_logic" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config" swaggertype:"object"`
	ApplicableTypes map[string]interface{} `json:"applicable_types" swaggertype:"object"`
	ComplexityLevel string                 `json:"complexity_level" example:"medium"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"false"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedBy       string                 `json:"created_by" example:"admin"`
	UpdatedBy       string                 `json:"updated_by" example:"admin"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// DataCleansingTemplateListResponse 数据清洗模板列表响应
type DataCleansingTemplateListResponse struct {
	List  []DataCleansingTemplateResponse `json:"list"`
	Total int64                           `json:"total" example:"20"`
	Page  int                             `json:"page" example:"1"`
	Size  int                             `json:"size" example:"10"`
}

// === 清洗应用相关类型 ===

// CreateDataCleansingApplicationRequest 创建数据清洗应用请求
type CreateDataCleansingApplicationRequest struct {
	TemplateID       string                 `json:"template_id" binding:"required" example:"uuid-123"`
	Name             string                 `json:"name" binding:"required" example:"用户表邮箱清洗"`
	TargetTable      string                 `json:"target_table" binding:"required" example:"users"`
	TargetFields     map[string]interface{} `json:"target_fields" binding:"required" swaggertype:"object"`
	TriggerCondition string                 `json:"trigger_condition,omitempty" example:"email IS NOT NULL"`
	CleansingConfig  map[string]interface{} `json:"cleansing_config" binding:"required" swaggertype:"object"`
	Priority         int                    `json:"priority" example:"50"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
	PreCondition     string                 `json:"pre_condition,omitempty" example:"LENGTH(email) > 0"`
	PostCondition    string                 `json:"post_condition,omitempty" example:"email LIKE '%@%'"`
	BackupOriginal   bool                   `json:"backup_original" example:"true"`
	ValidationRules  map[string]interface{} `json:"validation_rules,omitempty" swaggertype:"object"`
	ErrorHandling    map[string]interface{} `json:"error_handling,omitempty" swaggertype:"object"`
	ExecutionOrder   int                    `json:"execution_order" example:"1"`
	Schedule         map[string]interface{} `json:"schedule,omitempty" swaggertype:"object"`
}

// UpdateDataCleansingApplicationRequest 更新数据清洗应用请求
type UpdateDataCleansingApplicationRequest struct {
	Name             string                 `json:"name,omitempty" example:"更新后的应用名称"`
	TargetFields     map[string]interface{} `json:"target_fields,omitempty" swaggertype:"object"`
	TriggerCondition string                 `json:"trigger_condition,omitempty" example:"email IS NOT NULL AND LENGTH(email) > 5"`
	CleansingConfig  map[string]interface{} `json:"cleansing_config,omitempty" swaggertype:"object"`
	Priority         *int                   `json:"priority,omitempty" example:"60"`
	IsEnabled        *bool                  `json:"is_enabled,omitempty" example:"false"`
	PreCondition     string                 `json:"pre_condition,omitempty" example:"email != ''"`
	PostCondition    string                 `json:"post_condition,omitempty" example:"email REGEXP '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$'"`
	BackupOriginal   *bool                  `json:"backup_original,omitempty" example:"false"`
	ValidationRules  map[string]interface{} `json:"validation_rules,omitempty" swaggertype:"object"`
	ErrorHandling    map[string]interface{} `json:"error_handling,omitempty" swaggertype:"object"`
	ExecutionOrder   *int                   `json:"execution_order,omitempty" example:"2"`
	Schedule         map[string]interface{} `json:"schedule,omitempty" swaggertype:"object"`
}

// DataCleansingApplicationResponse 数据清洗应用响应
type DataCleansingApplicationResponse struct {
	ID               string                         `json:"id" example:"uuid-123"`
	TemplateID       string                         `json:"template_id" example:"uuid-456"`
	Template         *DataCleansingTemplateResponse `json:"template,omitempty"`
	Name             string                         `json:"name" example:"用户表邮箱清洗"`
	TargetTable      string                         `json:"target_table" example:"users"`
	TargetFields     map[string]interface{}         `json:"target_fields" swaggertype:"object"`
	TriggerCondition string                         `json:"trigger_condition" example:"email IS NOT NULL"`
	CleansingConfig  map[string]interface{}         `json:"cleansing_config" swaggertype:"object"`
	Priority         int                            `json:"priority" example:"50"`
	IsEnabled        bool                           `json:"is_enabled" example:"true"`
	PreCondition     string                         `json:"pre_condition" example:"LENGTH(email) > 0"`
	PostCondition    string                         `json:"post_condition" example:"email LIKE '%@%'"`
	BackupOriginal   bool                           `json:"backup_original" example:"true"`
	ValidationRules  map[string]interface{}         `json:"validation_rules" swaggertype:"object"`
	ErrorHandling    map[string]interface{}         `json:"error_handling" swaggertype:"object"`
	ExecutionOrder   int                            `json:"execution_order" example:"1"`
	Schedule         map[string]interface{}         `json:"schedule" swaggertype:"object"`
	SuccessCount     int64                          `json:"success_count" example:"1000"`
	FailureCount     int64                          `json:"failure_count" example:"10"`
	LastExecuted     *time.Time                     `json:"last_executed,omitempty" example:"2024-01-01T00:00:00Z"`
	NextExecution    *time.Time                     `json:"next_execution,omitempty" example:"2024-01-02T02:00:00Z"`
	CreatedBy        string                         `json:"created_by" example:"admin"`
	UpdatedBy        string                         `json:"updated_by" example:"admin"`
	CreatedAt        time.Time                      `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt        time.Time                      `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// DataCleansingApplicationListResponse 数据清洗应用列表响应
type DataCleansingApplicationListResponse struct {
	List  []DataCleansingApplicationResponse `json:"list"`
	Total int64                              `json:"total" example:"12"`
	Page  int                                `json:"page" example:"1"`
	Size  int                                `json:"size" example:"10"`
}

// CleansingApplicationExecutionResponse 清洗应用执行响应
type CleansingApplicationExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	ApplicationID   string                 `json:"application_id" example:"uuid-456"`
	TemplateID      string                 `json:"template_id" example:"uuid-789"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:03:00Z"`
	Duration        int64                  `json:"duration" example:"180000"`
	Status          string                 `json:"status" example:"completed"`
	TotalRecords    int64                  `json:"total_records" example:"8000"`
	CleanedRecords  int64                  `json:"cleaned_records" example:"7500"`
	SkippedRecords  int64                  `json:"skipped_records" example:"500"`
	CleansingRate   float64                `json:"cleansing_rate" example:"0.9375"`
	ExecutionResult map[string]interface{} `json:"execution_result" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// === 元数据相关类型 ===

// CreateMetadataRequest 创建元数据请求
type CreateMetadataRequest struct {
	Type              string                 `json:"type" binding:"required" example:"technical" enums:"technical,business,management"`
	Name              string                 `json:"name" binding:"required" example:"用户表技术元数据"`
	Content           map[string]interface{} `json:"content" binding:"required" `
	RelatedObjectID   string                 `json:"related_object_id,omitempty" example:"uuid-123"`
	RelatedObjectType string                 `json:"related_object_type,omitempty" example:"interface"`
	Description       string                 `json:"description" example:"用户表的技术元数据信息"`
}

// UpdateMetadataRequest 更新元数据请求
type UpdateMetadataRequest struct {
	Name        string                 `json:"name,omitempty" example:"更新后的元数据名称"`
	Content     map[string]interface{} `json:"content,omitempty" swaggertype:"object"`
	Description string                 `json:"description,omitempty" example:"更新后的描述"`
}

// MetadataResponse 元数据响应
type MetadataResponse struct {
	ID                string                 `json:"id" example:"uuid-123"`
	Type              string                 `json:"type" example:"technical"`
	Name              string                 `json:"name" example:"用户表技术元数据"`
	Content           map[string]interface{} `json:"content" `
	RelatedObjectID   string                 `json:"related_object_id" example:"uuid-456"`
	RelatedObjectType string                 `json:"related_object_type" example:"interface"`
	CreatedAt         time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy         string                 `json:"created_by" example:"admin"`
	UpdatedAt         time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy         string                 `json:"updated_by" example:"admin"`
}

// MetadataListResponse 元数据列表响应
type MetadataListResponse struct {
	List  []MetadataResponse `json:"list"`
	Total int64              `json:"total" example:"80"`
	Page  int                `json:"page" example:"1"`
	Size  int                `json:"size" example:"10"`
}
