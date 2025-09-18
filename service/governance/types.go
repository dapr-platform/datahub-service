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

// CreateQualityRuleRequest 创建质量规则请求
type CreateQualityRuleRequest struct {
	Name              string                 `json:"name" binding:"required" example:"完整性检查规则"`
	Type              string                 `json:"type" binding:"required" example:"completeness" enums:"completeness,accuracy,consistency,validity,uniqueness,timeliness,standardization"`
	Config            map[string]interface{} `json:"config" binding:"required" example:"{\"threshold\":0.95,\"fields\":[\"name\",\"email\"]}"`
	RelatedObjectID   string                 `json:"related_object_id" binding:"required" example:"uuid-123"`
	RelatedObjectType string                 `json:"related_object_type" binding:"required" example:"interface" enums:"interface,thematic_interface"`
	IsEnabled         bool                   `json:"is_enabled" example:"true"`
	Description       string                 `json:"description" example:"检查用户表的姓名和邮箱字段完整性"`
}

// UpdateQualityRuleRequest 更新质量规则请求
type UpdateQualityRuleRequest struct {
	Name        string                 `json:"name,omitempty" example:"更新后的规则名称"`
	Config      map[string]interface{} `json:"config,omitempty" example:"{\"threshold\":0.98}"`
	IsEnabled   *bool                  `json:"is_enabled,omitempty" example:"false"`
	Description string                 `json:"description,omitempty" example:"更新后的描述"`
}

// QualityRuleResponse 质量规则响应
type QualityRuleResponse struct {
	ID                string                 `json:"id" example:"uuid-123"`
	Name              string                 `json:"name" example:"完整性检查规则"`
	Type              string                 `json:"type" example:"completeness"`
	Config            map[string]interface{} `json:"config" `
	RelatedObjectID   string                 `json:"related_object_id" example:"uuid-456"`
	RelatedObjectType string                 `json:"related_object_type" example:"interface"`
	IsEnabled         bool                   `json:"is_enabled" example:"true"`
	CreatedAt         time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy         string                 `json:"created_by" example:"admin"`
	UpdatedAt         time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy         string                 `json:"updated_by" example:"admin"`
}

// QualityRuleListResponse 质量规则列表响应
type QualityRuleListResponse struct {
	List  []QualityRuleResponse `json:"list"`
	Total int64                 `json:"total" example:"100"`
	Page  int                   `json:"page" example:"1"`
	Size  int                   `json:"size" example:"10"`
}

// === 数据脱敏规则相关类型 ===

// CreateMaskingRuleRequest 创建脱敏规则请求
type CreateMaskingRuleRequest struct {
	Name          string                 `json:"name" binding:"required" example:"手机号脱敏规则"`
	DataSource    string                 `json:"data_source" binding:"required" example:"user_db"`
	DataTable     string                 `json:"data_table" binding:"required" example:"users"`
	FieldName     string                 `json:"field_name" binding:"required" example:"mobile"`
	FieldType     string                 `json:"field_type" binding:"required" example:"varchar"`
	MaskingType   string                 `json:"masking_type" binding:"required" example:"mask" enums:"mask,replace,encrypt,pseudonymize"`
	MaskingConfig map[string]interface{} `json:"masking_config" binding:"required" example:"{\"pattern\":\"***\",\"keep_start\":3,\"keep_end\":4}"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Description   string                 `json:"description" example:"对用户手机号进行掩码处理"`
}

// UpdateMaskingRuleRequest 更新脱敏规则请求
type UpdateMaskingRuleRequest struct {
	Name          string                 `json:"name,omitempty" example:"更新后的脱敏规则"`
	MaskingConfig map[string]interface{} `json:"masking_config,omitempty" example:"{\"pattern\":\"****\"}"`
	IsEnabled     *bool                  `json:"is_enabled,omitempty" example:"false"`
	Description   string                 `json:"description,omitempty" example:"更新后的描述"`
}

// MaskingRuleResponse 脱敏规则响应
type MaskingRuleResponse struct {
	ID            string                 `json:"id" example:"uuid-123"`
	Name          string                 `json:"name" example:"手机号脱敏规则"`
	DataSource    string                 `json:"data_source" example:"user_db"`
	DataTable     string                 `json:"data_table" example:"users"`
	FieldName     string                 `json:"field_name" example:"mobile"`
	FieldType     string                 `json:"field_type" example:"varchar"`
	MaskingType   string                 `json:"masking_type" example:"mask"`
	MaskingConfig map[string]interface{} `json:"masking_config"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
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

// CreateCleansingRuleRequest 创建清洗规则请求
type CreateCleansingRuleRequest struct {
	Name             string                 `json:"name" binding:"required" example:"邮箱格式标准化"`
	Description      string                 `json:"description" example:"统一邮箱格式为小写"`
	RuleType         string                 `json:"rule_type" binding:"required" example:"standardization" enums:"standardization,deduplication,validation,transformation,enrichment"`
	TargetTable      string                 `json:"target_table" binding:"required" example:"users"`
	TargetColumn     string                 `json:"target_column" example:"email"`
	TriggerCondition string                 `json:"trigger_condition" example:"email IS NOT NULL"`
	CleansingAction  map[string]interface{} `json:"cleansing_action" binding:"required" example:"{\"action\":\"lowercase\"}"`
	Priority         int                    `json:"priority" example:"50"`
	IsEnabled        bool                   `json:"is_enabled" example:"true"`
}

// UpdateCleansingRuleRequest 更新清洗规则请求
type UpdateCleansingRuleRequest struct {
	Name            string                 `json:"name,omitempty" example:"更新后的清洗规则"`
	Description     string                 `json:"description,omitempty" example:"更新后的描述"`
	CleansingAction map[string]interface{} `json:"cleansing_action,omitempty" example:"{\"action\":\"uppercase\"}"`
	Priority        *int                   `json:"priority,omitempty" example:"60"`
	IsEnabled       *bool                  `json:"is_enabled,omitempty" example:"false"`
}

// CleansingRuleResponse 清洗规则响应
type CleansingRuleResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	Name            string                 `json:"name" example:"邮箱格式标准化"`
	Description     string                 `json:"description" example:"统一邮箱格式为小写"`
	RuleType        string                 `json:"rule_type" example:"standardization"`
	TargetTable     string                 `json:"target_table" example:"users"`
	TargetColumn    string                 `json:"target_column" example:"email"`
	CleansingAction map[string]interface{} `json:"cleansing_action" example:"{\"action\":\"lowercase\"}"`
	Priority        int                    `json:"priority" example:"50"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	SuccessCount    int64                  `json:"success_count" example:"1000"`
	FailureCount    int64                  `json:"failure_count" example:"10"`
	LastExecuted    *time.Time             `json:"last_executed,omitempty" example:"2024-01-01T00:00:00Z"`
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
	IssueContext     map[string]interface{} `json:"issue_context" example:"{\"row_number\":123}"`
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
	ResolutionAction map[string]interface{} `json:"resolution_action" example:"{\"action\":\"manual_fix\",\"value\":\"user@example.com\"}"`
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
	Content     map[string]interface{} `json:"content,omitempty" example:"{\"updated_field\":\"value\"}"`
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
