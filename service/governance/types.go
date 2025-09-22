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
	Name          string                 `json:"name,omitempty" example:"更新后的质量规则模板"`
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

// QualityRuleListResponse 质量规则模板列表响应
type QualityRuleListResponse struct {
	List  []QualityRuleResponse `json:"list"`
	Total int64                 `json:"total" example:"25"`
	Page  int                   `json:"page" example:"1"`
	Size  int                   `json:"size" example:"10"`
}

// === 数据脱敏规则相关类型 ===

// CreateMaskingRuleRequest 创建脱敏规则模板请求
type CreateMaskingRuleRequest struct {
	Name          string                 `json:"name" binding:"required" example:"手机号脱敏模板"`
	MaskingType   string                 `json:"masking_type" binding:"required" example:"mask" enums:"mask,replace,encrypt,pseudonymize"`
	Category      string                 `json:"category" binding:"required" example:"personal_info" enums:"personal_info,financial,medical,business,custom"`
	SecurityLevel string                 `json:"security_level" example:"high" enums:"low,medium,high,critical"`
	Description   string                 `json:"description" example:"对手机号进行脱敏处理的通用模板"`
	MaskingLogic  map[string]interface{} `json:"masking_logic" binding:"required" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters" swaggertype:"object"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Tags          map[string]interface{} `json:"tags" swaggertype:"object"`
}

// UpdateMaskingRuleRequest 更新脱敏规则模板请求
type UpdateMaskingRuleRequest struct {
	Name         string                 `json:"name,omitempty" example:"更新后的脱敏规则模板"`
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
	Category      string                 `json:"category" example:"personal_info"`
	SecurityLevel string                 `json:"security_level" example:"high"`
	MaskingType   string                 `json:"masking_type" example:"mask"`
	Description   string                 `json:"description" example:"对手机号进行脱敏处理的通用模板"`
	MaskingLogic  map[string]interface{} `json:"masking_logic" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters" swaggertype:"object"`
	IsBuiltIn     bool                   `json:"is_built_in" example:"false"`
	Version       string                 `json:"version" example:"1.0"`
	Tags          map[string]interface{} `json:"tags" swaggertype:"object"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	CreatedAt     time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy     string                 `json:"created_by" example:"admin"`
	UpdatedAt     time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy     string                 `json:"updated_by" example:"admin"`
}

// MaskingRuleListResponse 脱敏规则模板列表响应
type MaskingRuleListResponse struct {
	List  []MaskingRuleResponse `json:"list"`
	Total int64                 `json:"total" example:"12"`
	Page  int                   `json:"page" example:"1"`
	Size  int                   `json:"size" example:"10"`
}

// === 质量检查相关类型 ===

// RunQualityCheckRequest 执行质量检查请求
type RunQualityCheckRequest struct {
	ObjectID   string `json:"object_id" binding:"required" example:"uuid-123"`
	ObjectType string `json:"object_type" binding:"required" example:"interface" enums:"interface,thematic_interface"`
}

// QualityReportResponse 质量报告响应
type QualityReportResponse struct {
	ID                string                 `json:"id" example:"uuid-123"`
	ReportName        string                 `json:"report_name" example:"接口质量检查报告"`
	RelatedObjectID   string                 `json:"related_object_id" example:"uuid-456"`
	RelatedObjectType string                 `json:"related_object_type" example:"interface"`
	QualityScore      float64                `json:"quality_score" example:"85.5"`
	QualityMetrics    map[string]interface{} `json:"quality_metrics" swaggertype:"object"`
	Issues            map[string]interface{} `json:"issues" swaggertype:"object"`
	Recommendations   map[string]interface{} `json:"recommendations" swaggertype:"object"`
	GeneratedAt       time.Time              `json:"generated_at" example:"2024-01-01T00:00:00Z"`
	GeneratedBy       string                 `json:"generated_by" example:"system"`
}

// QualityReportListResponse 质量报告列表响应
type QualityReportListResponse struct {
	List  []QualityReportResponse `json:"list"`
	Total int64                   `json:"total" example:"8"`
	Page  int                     `json:"page" example:"1"`
	Size  int                     `json:"size" example:"10"`
}

// === 元数据相关类型 ===

// CreateMetadataRequest 创建元数据请求
type CreateMetadataRequest struct {
	Type              string                 `json:"type" binding:"required" example:"technical" enums:"technical,business,management"`
	Name              string                 `json:"name" binding:"required" example:"用户表元数据"`
	Content           map[string]interface{} `json:"content" binding:"required" swaggertype:"object"`
	RelatedObjectID   string                 `json:"related_object_id" example:"uuid-123"`
	RelatedObjectType string                 `json:"related_object_type" example:"interface"`
}

// UpdateMetadataRequest 更新元数据请求
type UpdateMetadataRequest struct {
	Name        string                 `json:"name,omitempty" example:"更新后的元数据"`
	Content     map[string]interface{} `json:"content,omitempty" swaggertype:"object"`
	Description string                 `json:"description,omitempty" example:"更新后的描述"`
}

// MetadataResponse 元数据响应
type MetadataResponse struct {
	ID                string                 `json:"id" example:"uuid-123"`
	Type              string                 `json:"type" example:"technical"`
	Name              string                 `json:"name" example:"用户表元数据"`
	Content           map[string]interface{} `json:"content" swaggertype:"object"`
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
	Total int64              `json:"total" example:"30"`
	Page  int                `json:"page" example:"1"`
	Size  int                `json:"size" example:"10"`
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
	CleansingLogic  map[string]interface{} `json:"cleansing_logic" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config" swaggertype:"object"`
	ComplexityLevel string                 `json:"complexity_level" example:"medium"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"false"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy       string                 `json:"created_by" example:"admin"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy       string                 `json:"updated_by" example:"admin"`
}

// CleansingRuleListResponse 清洗规则模板列表响应
type CleansingRuleListResponse struct {
	List  []CleansingRuleResponse `json:"list"`
	Total int64                   `json:"total" example:"18"`
	Page  int                     `json:"page" example:"1"`
	Size  int                     `json:"size" example:"10"`
}

// CleansingExecutionResponse 清洗执行响应
type CleansingExecutionResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	RuleID          string                 `json:"rule_id" example:"uuid-456"`
	StartTime       time.Time              `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         *time.Time             `json:"end_time,omitempty" example:"2024-01-01T00:01:00Z"`
	Duration        int64                  `json:"duration" example:"60000"`
	Status          string                 `json:"status" example:"completed"`
	ProcessedCount  int64                  `json:"processed_count" example:"5000"`
	CleanedCount    int64                  `json:"cleaned_count" example:"4800"`
	ErrorCount      int64                  `json:"error_count" example:"200"`
	ExecutionResult map[string]interface{} `json:"execution_result" swaggertype:"object"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// === 质量检测任务相关类型 ===

// CreateQualityTaskRequest 创建质量检测任务请求
type CreateQualityTaskRequest struct {
	Name               string                 `json:"name" binding:"required" example:"用户表质量检测任务"`
	Description        string                 `json:"description" example:"定期检测用户表数据质量"`
	TaskType           string                 `json:"task_type" binding:"required" example:"scheduled" enums:"scheduled,manual,realtime"`
	TargetObjectID     string                 `json:"target_object_id" binding:"required" example:"uuid-123"`
	TargetObjectType   string                 `json:"target_object_type" binding:"required" example:"interface" enums:"interface,thematic_interface,table"`
	QualityRuleIDs     []string               `json:"quality_rule_ids" example:"[\"uuid-456\"]"`
	ScheduleConfig     map[string]interface{} `json:"schedule_config" swaggertype:"object"`
	NotificationConfig map[string]interface{} `json:"notification_config" swaggertype:"object"`
	Priority           int                    `json:"priority" example:"50"`
	IsEnabled          bool                   `json:"is_enabled" example:"true"`
}

// UpdateQualityTaskRequest 更新质量检测任务请求
type UpdateQualityTaskRequest struct {
	Name               string                 `json:"name,omitempty" example:"更新后的质量检测任务"`
	Description        string                 `json:"description,omitempty" example:"更新后的描述"`
	QualityRuleIDs     []string               `json:"quality_rule_ids,omitempty" example:"[\"uuid-789\"]"`
	ScheduleConfig     map[string]interface{} `json:"schedule_config,omitempty" swaggertype:"object"`
	NotificationConfig map[string]interface{} `json:"notification_config,omitempty" swaggertype:"object"`
	Priority           *int                   `json:"priority,omitempty" example:"80"`
	IsEnabled          *bool                  `json:"is_enabled,omitempty" example:"false"`
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
	NextExecution      *time.Time             `json:"next_execution,omitempty" example:"2024-01-02T00:00:00Z"`
	ExecutionCount     int64                  `json:"execution_count" example:"5"`
	SuccessCount       int64                  `json:"success_count" example:"4"`
	FailureCount       int64                  `json:"failure_count" example:"1"`
	CreatedAt          time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy          string                 `json:"created_by" example:"admin"`
	UpdatedAt          time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy          string                 `json:"updated_by" example:"admin"`
}

// QualityTaskListResponse 质量检测任务列表响应
type QualityTaskListResponse struct {
	List  []QualityTaskResponse `json:"list"`
	Total int64                 `json:"total" example:"20"`
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
	ErrorMessage       string                 `json:"error_message,omitempty"`
	TriggerSource      string                 `json:"trigger_source" example:"manual"`
	ExecutedBy         string                 `json:"executed_by" example:"admin"`
	CreatedAt          time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt          time.Time              `json:"updated_at" example:"2024-01-01T00:05:00Z"`
}

// QualityTaskExecutionListResponse 质量检测任务执行记录列表响应
type QualityTaskExecutionListResponse struct {
	List  []QualityTaskExecutionResponse `json:"list"`
	Total int64                          `json:"total" example:"50"`
	Page  int                            `json:"page" example:"1"`
	Size  int                            `json:"size" example:"10"`
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
	OperationContent map[string]interface{} `json:"operation_content" swaggertype:"object"`
	OperationTime    time.Time              `json:"operation_time" example:"2024-01-01T00:00:00Z"`
	OperationResult  string                 `json:"operation_result" example:"success"`
}

// SystemLogListResponse 系统日志列表响应
type SystemLogListResponse struct {
	List  []SystemLogResponse `json:"list"`
	Total int64               `json:"total" example:"100"`
	Page  int                 `json:"page" example:"1"`
	Size  int                 `json:"size" example:"10"`
}

// === 数据血缘相关类型 ===

// CreateDataLineageRequest 创建数据血缘请求
type CreateDataLineageRequest struct {
	SourceObjectID   string                 `json:"source_object_id" binding:"required" example:"uuid-123"`
	SourceObjectType string                 `json:"source_object_type" binding:"required" example:"table"`
	TargetObjectID   string                 `json:"target_object_id" binding:"required" example:"uuid-456"`
	TargetObjectType string                 `json:"target_object_type" binding:"required" example:"interface"`
	RelationType     string                 `json:"relation_type" binding:"required" example:"direct" enums:"direct,derived,aggregated,transformed"`
	TransformRule    map[string]interface{} `json:"transform_rule,omitempty" swaggertype:"object"`
	ColumnMapping    map[string]interface{} `json:"column_mapping,omitempty" swaggertype:"object"`
	Confidence       float64                `json:"confidence" example:"1.0"`
	IsActive         bool                   `json:"is_active" example:"true"`
	Description      string                 `json:"description,omitempty" example:"用户表到用户接口的直接映射"`
}

// DataLineageResponse 数据血缘响应
type DataLineageResponse struct {
	ID               string                 `json:"id" example:"uuid-123"`
	SourceObjectID   string                 `json:"source_object_id" example:"uuid-456"`
	SourceObjectType string                 `json:"source_object_type" example:"table"`
	TargetObjectID   string                 `json:"target_object_id" example:"uuid-789"`
	TargetObjectType string                 `json:"target_object_type" example:"interface"`
	RelationType     string                 `json:"relation_type" example:"direct"`
	TransformRule    map[string]interface{} `json:"transform_rule" swaggertype:"object"`
	ColumnMapping    map[string]interface{} `json:"column_mapping" swaggertype:"object"`
	Confidence       float64                `json:"confidence" example:"1.0"`
	IsActive         bool                   `json:"is_active" example:"true"`
	Description      string                 `json:"description" example:"用户表到用户接口的直接映射"`
	CreatedAt        time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy        string                 `json:"created_by" example:"admin"`
	UpdatedAt        time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy        string                 `json:"updated_by" example:"admin"`
}

// DataLineageNode 血缘图节点
type DataLineageNode struct {
	ID         string `json:"id" example:"uuid-123"`
	ObjectType string `json:"object_type" example:"table"`
	Name       string `json:"name" example:"users_table"`
	Level      int    `json:"level" example:"0"`
}

// DataLineageEdge 血缘图边
type DataLineageEdge struct {
	ID           string  `json:"id" example:"uuid-123"`
	SourceID     string  `json:"source_id" example:"uuid-456"`
	TargetID     string  `json:"target_id" example:"uuid-789"`
	RelationType string  `json:"relation_type" example:"direct"`
	Confidence   float64 `json:"confidence" example:"1.0"`
}

// DataLineageStats 血缘图统计信息
type DataLineageStats struct {
	TotalNodes int `json:"total_nodes" example:"10"`
	TotalEdges int `json:"total_edges" example:"8"`
	MaxDepth   int `json:"max_depth" example:"3"`
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

// === 模板管理相关类型 ===

// QualityRuleTemplateResponse 质量规则模板响应（用于模板管理接口）
type QualityRuleTemplateResponse struct {
	ID            string                 `json:"id" example:"uuid-123"`
	Name          string                 `json:"name" example:"完整性检查模板"`
	Type          string                 `json:"type" example:"completeness"`
	Category      string                 `json:"category" example:"basic_quality"`
	Description   string                 `json:"description" example:"检查数据完整性的通用模板"`
	RuleLogic     map[string]interface{} `json:"rule_logic" swaggertype:"object"`
	Parameters    map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig map[string]interface{} `json:"default_config" swaggertype:"object"`
	IsBuiltIn     bool                   `json:"is_built_in" example:"true"`
	IsEnabled     bool                   `json:"is_enabled" example:"true"`
	Version       string                 `json:"version" example:"1.0"`
	Tags          map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedAt     time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy     string                 `json:"created_by" example:"system"`
	UpdatedAt     time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy     string                 `json:"updated_by" example:"system"`
}

// QualityRuleTemplateListResponse 质量规则模板列表响应
type QualityRuleTemplateListResponse struct {
	List  []QualityRuleTemplateResponse `json:"list"`
	Total int64                         `json:"total" example:"50"`
	Page  int                           `json:"page" example:"1"`
	Size  int                           `json:"size" example:"10"`
}

// DataMaskingTemplateResponse 数据脱敏模板响应
type DataMaskingTemplateResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	Name            string                 `json:"name" example:"手机号脱敏模板"`
	MaskingType     string                 `json:"masking_type" example:"mask"`
	Category        string                 `json:"category" example:"personal_info"`
	Description     string                 `json:"description" example:"对手机号进行脱敏处理"`
	ApplicableTypes []string               `json:"applicable_types" example:"[\"string\"]"`
	MaskingLogic    map[string]interface{} `json:"masking_logic" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config" swaggertype:"object"`
	SecurityLevel   string                 `json:"security_level" example:"high"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"true"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	CreatedBy       string                 `json:"created_by" example:"system"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	UpdatedBy       string                 `json:"updated_by" example:"system"`
}

// DataMaskingTemplateListResponse 数据脱敏模板列表响应
type DataMaskingTemplateListResponse struct {
	List  []DataMaskingTemplateResponse `json:"list"`
	Total int64                         `json:"total" example:"30"`
	Page  int                           `json:"page" example:"1"`
	Size  int                           `json:"size" example:"10"`
}

// DataCleansingTemplateResponse 数据清洗模板响应
type DataCleansingTemplateResponse struct {
	ID              string                 `json:"id" example:"uuid-123"`
	Name            string                 `json:"name" example:"邮箱格式标准化模板"`
	Description     string                 `json:"description" example:"统一邮箱格式为小写"`
	RuleType        string                 `json:"rule_type" example:"standardization"`
	Category        string                 `json:"category" example:"data_format"`
	CleansingLogic  map[string]interface{} `json:"cleansing_logic" swaggertype:"object"`
	Parameters      map[string]interface{} `json:"parameters" swaggertype:"object"`
	DefaultConfig   map[string]interface{} `json:"default_config" swaggertype:"object"`
	ApplicableTypes map[string]interface{} `json:"applicable_types" swaggertype:"object"`
	ComplexityLevel string                 `json:"complexity_level" example:"medium"`
	IsBuiltIn       bool                   `json:"is_built_in" example:"true"`
	IsEnabled       bool                   `json:"is_enabled" example:"true"`
	Version         string                 `json:"version" example:"1.0"`
	Tags            map[string]interface{} `json:"tags" swaggertype:"object"`
	CreatedBy       string                 `json:"created_by" example:"system"`
	UpdatedBy       string                 `json:"updated_by" example:"system"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// DataCleansingTemplateListResponse 数据清洗模板列表响应
type DataCleansingTemplateListResponse struct {
	List  []DataCleansingTemplateResponse `json:"list"`
	Total int64                           `json:"total" example:"25"`
	Page  int                             `json:"page" example:"1"`
	Size  int                             `json:"size" example:"10"`
}

// === 规则测试相关类型定义 ===

// TestQualityRuleRequest 测试数据质量规则请求
type TestQualityRuleRequest struct {
	RuleTemplateID string                 `json:"rule_template_id" binding:"required" example:"uuid-123"`
	TestData       map[string]interface{} `json:"test_data" binding:"required" swaggertype:"object"`
	TargetFields   []string               `json:"target_fields" binding:"required" example:"[\"name\",\"email\"]"`
	RuntimeConfig  map[string]interface{} `json:"runtime_config" swaggertype:"object"`
	Threshold      map[string]interface{} `json:"threshold" swaggertype:"object"`
}

// TestMaskingRuleRequest 测试数据脱敏规则请求
type TestMaskingRuleRequest struct {
	TemplateID     string                 `json:"template_id" binding:"required" example:"uuid-123"`
	TestData       map[string]interface{} `json:"test_data" binding:"required" swaggertype:"object"`
	TargetFields   []string               `json:"target_fields" binding:"required" example:"[\"phone\",\"id_card\"]"`
	MaskingConfig  map[string]interface{} `json:"masking_config" swaggertype:"object"`
	PreserveFormat bool                   `json:"preserve_format" example:"true"`
}

// TestCleansingRuleRequest 测试数据清洗规则请求
type TestCleansingRuleRequest struct {
	TemplateID       string                 `json:"template_id" binding:"required" example:"uuid-123"`
	TestData         map[string]interface{} `json:"test_data" binding:"required" swaggertype:"object"`
	TargetFields     []string               `json:"target_fields" binding:"required" example:"[\"email\",\"address\"]"`
	CleansingConfig  map[string]interface{} `json:"cleansing_config" swaggertype:"object"`
	TriggerCondition string                 `json:"trigger_condition,omitempty" example:"email != ''"`
	BackupOriginal   bool                   `json:"backup_original" example:"true"`
}

// TestBatchRulesRequest 批量测试规则请求
type TestBatchRulesRequest struct {
	TestData       map[string]interface{}  `json:"test_data" binding:"required" swaggertype:"object"`
	QualityRules   []TestQualityRuleItem   `json:"quality_rules,omitempty"`
	MaskingRules   []TestMaskingRuleItem   `json:"masking_rules,omitempty"`
	CleansingRules []TestCleansingRuleItem `json:"cleansing_rules,omitempty"`
	ExecutionOrder []string                `json:"execution_order" example:"[\"quality\",\"cleansing\",\"masking\"]"`
}

// TestQualityRuleItem 质量规则测试项
type TestQualityRuleItem struct {
	RuleTemplateID string                 `json:"rule_template_id" example:"uuid-123"`
	TargetFields   []string               `json:"target_fields" example:"[\"name\"]"`
	RuntimeConfig  map[string]interface{} `json:"runtime_config" swaggertype:"object"`
	Threshold      map[string]interface{} `json:"threshold" swaggertype:"object"`
}

// TestMaskingRuleItem 脱敏规则测试项
type TestMaskingRuleItem struct {
	TemplateID     string                 `json:"template_id" example:"uuid-123"`
	TargetFields   []string               `json:"target_fields" example:"[\"phone\"]"`
	MaskingConfig  map[string]interface{} `json:"masking_config" swaggertype:"object"`
	PreserveFormat bool                   `json:"preserve_format" example:"true"`
}

// TestCleansingRuleItem 清洗规则测试项
type TestCleansingRuleItem struct {
	TemplateID       string                 `json:"template_id" example:"uuid-123"`
	TargetFields     []string               `json:"target_fields" example:"[\"email\"]"`
	CleansingConfig  map[string]interface{} `json:"cleansing_config" swaggertype:"object"`
	TriggerCondition string                 `json:"trigger_condition,omitempty"`
	BackupOriginal   bool                   `json:"backup_original" example:"true"`
}

// RuleTestResult 规则测试结果
type RuleTestResult struct {
	RuleType       string                 `json:"rule_type" example:"quality"`
	RuleTemplateID string                 `json:"rule_template_id" example:"uuid-123"`
	RuleName       string                 `json:"rule_name" example:"完整性检查"`
	Success        bool                   `json:"success" example:"true"`
	ProcessedData  map[string]interface{} `json:"processed_data" swaggertype:"object"`
	OriginalData   map[string]interface{} `json:"original_data" swaggertype:"object"`
	QualityScore   *float64               `json:"quality_score,omitempty" example:"0.85"`
	Issues         []string               `json:"issues,omitempty" example:"[\"字段name为空\"]"`
	Modifications  map[string]interface{} `json:"modifications,omitempty" swaggertype:"object"`
	ExecutionTime  int64                  `json:"execution_time" example:"50"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	FieldResults   map[string]interface{} `json:"field_results,omitempty" swaggertype:"object"`
}

// TestRuleResponse 测试规则响应
type TestRuleResponse struct {
	TestID          string           `json:"test_id" example:"uuid-test-123"`
	TotalRules      int              `json:"total_rules" example:"3"`
	SuccessfulRules int              `json:"successful_rules" example:"2"`
	FailedRules     int              `json:"failed_rules" example:"1"`
	OverallSuccess  bool             `json:"overall_success" example:"false"`
	ExecutionTime   int64            `json:"execution_time" example:"150"`
	Results         []RuleTestResult `json:"results"`
	Summary         struct {
		QualityChecks  int     `json:"quality_checks" example:"1"`
		MaskingRules   int     `json:"masking_rules" example:"1"`
		CleansingRules int     `json:"cleansing_rules" example:"1"`
		OverallScore   float64 `json:"overall_score,omitempty" example:"0.75"`
	} `json:"summary"`
	Recommendations []string `json:"recommendations,omitempty" example:"[\"建议调整完整性检查阈值\"]"`
}

// TestRulePreviewRequest 规则预览测试请求（不执行，仅预览效果）
type TestRulePreviewRequest struct {
	RuleType      string                 `json:"rule_type" binding:"required" example:"quality" enums:"quality,masking,cleansing"`
	TemplateID    string                 `json:"template_id" binding:"required" example:"uuid-123"`
	SampleData    map[string]interface{} `json:"sample_data" binding:"required" swaggertype:"object"`
	TargetFields  []string               `json:"target_fields" binding:"required" example:"[\"name\"]"`
	Configuration map[string]interface{} `json:"configuration" swaggertype:"object"`
}

// TestRulePreviewResponse 规则预览测试响应
type TestRulePreviewResponse struct {
	RuleType         string                 `json:"rule_type" example:"quality"`
	RuleName         string                 `json:"rule_name" example:"完整性检查"`
	OriginalData     map[string]interface{} `json:"original_data" swaggertype:"object"`
	PreviewResult    map[string]interface{} `json:"preview_result" swaggertype:"object"`
	ExpectedChanges  []string               `json:"expected_changes" example:"[\"字段name将被检查是否为空\"]"`
	ConfigValidation struct {
		IsValid bool     `json:"is_valid" example:"true"`
		Issues  []string `json:"issues,omitempty" example:"[\"阈值配置缺失\"]"`
	} `json:"config_validation"`
	EstimatedImpact struct {
		AffectedFields int     `json:"affected_fields" example:"2"`
		RiskLevel      string  `json:"risk_level" example:"low" enums:"low,medium,high"`
		Confidence     float64 `json:"confidence" example:"0.9"`
	} `json:"estimated_impact"`
}
