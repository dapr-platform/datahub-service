/*
 * @module service/models/quality_models
 * @description 数据质量扩展模型，包含质量检查记录、质量指标、清洗规则等模型
 * @architecture 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 质量检查执行 -> 质量评估 -> 数据清洗 -> 质量报告
 * @rules 确保数据质量评估的准确性和一致性，支持多维度质量评价体系
 * @dependencies gorm.io/gorm, time
 * @refs service/data_quality/, service/sync_engine/
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QualityCheckExecution 质量检查执行记录模型
type QualityCheckExecution struct {
	ID              string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	QualityRuleID   string     `gorm:"type:varchar(50);not null;index" json:"quality_rule_id"`
	SyncConfigID    string     `gorm:"type:varchar(50);index" json:"sync_config_id"`
	ExecutionID     string     `gorm:"type:varchar(50);index" json:"execution_id"`
	CheckType       string     `gorm:"type:varchar(30);not null" json:"check_type"` // batch, realtime, manual
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Duration        int64      `json:"duration"`                                // 检查时长，毫秒
	Status          string     `gorm:"type:varchar(20);not null" json:"status"` // running, passed, failed, warning
	TotalRecords    int64      `json:"total_records"`
	PassedRecords   int64      `json:"passed_records"`
	FailedRecords   int64      `json:"failed_records"`
	QualityScore    float64    `json:"quality_score"` // 质量评分 (0-1)
	ThresholdMet    bool       `json:"threshold_met"` // 是否达到阈值
	ErrorMessage    string     `gorm:"type:text" json:"error_message,omitempty"`
	CheckResults    JSONB      `gorm:"type:jsonb" json:"check_results"`   // 检查结果详情
	SampleData      JSONB      `gorm:"type:jsonb" json:"sample_data"`     // 样本数据
	Recommendations JSONB      `gorm:"type:jsonb" json:"recommendations"` // 改进建议
	FixedRecords    int64      `gorm:"default:0" json:"fixed_records"`    // 自动修复记录数
	FixedActions    JSONB      `gorm:"type:jsonb" json:"fixed_actions"`   // 修复动作记录
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityCheckExecution) TableName() string {
	return "quality_check_executions"
}

// BeforeCreate 创建前钩子
func (q *QualityCheckExecution) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityCheckExecution) BeforeUpdate(tx *gorm.DB) error {
	// 更新时不自动设置字段，保持原有逻辑
	return nil
}

// QualityMetricRecord 质量指标记录模型
type QualityMetricRecord struct {
	ID              string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	MetricName      string    `gorm:"type:varchar(100);not null" json:"metric_name"`
	MetricType      string    `gorm:"type:varchar(30);not null" json:"metric_type"` // completeness, accuracy, consistency, validity, uniqueness, timeliness
	TargetTable     string    `gorm:"type:varchar(100);not null;index" json:"target_table"`
	TargetColumn    string    `gorm:"type:varchar(100);index" json:"target_column"`
	MetricDate      time.Time `gorm:"type:date;not null;index" json:"metric_date"`
	MetricValue     float64   `json:"metric_value"`                  // 指标值
	BaselineValue   float64   `json:"baseline_value"`                // 基线值
	TargetValue     float64   `json:"target_value"`                  // 目标值
	Trend           string    `gorm:"type:varchar(20)" json:"trend"` // improving, stable, declining
	TotalCount      int64     `json:"total_count"`
	ValidCount      int64     `json:"valid_count"`
	InvalidCount    int64     `json:"invalid_count"`
	NullCount       int64     `json:"null_count"`
	DuplicateCount  int64     `json:"duplicate_count"`
	OutlierCount    int64     `json:"outlier_count"`
	MetricDetails   JSONB     `gorm:"type:jsonb" json:"metric_details"` // 指标详细信息
	CalculationTime time.Time `json:"calculation_time"`
	DataSource      string    `gorm:"type:varchar(100)" json:"data_source"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	DeletedAt       time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityMetricRecord) TableName() string {
	return "quality_metric_records"
}

// BeforeCreate 创建前钩子
func (q *QualityMetricRecord) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityMetricRecord) BeforeUpdate(tx *gorm.DB) error {
	// 更新时不自动设置字段，保持原有逻辑
	return nil
}

// DataCleansingTemplate 数据清洗规则模板模型
type DataCleansingTemplate struct {
	ID              string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	RuleType        string    `gorm:"type:varchar(30);not null" json:"rule_type"`                // standardization, deduplication, validation, transformation, enrichment
	Category        string    `gorm:"type:varchar(50);not null" json:"category"`                 // data_format/data_quality/data_integrity
	CleansingLogic  JSONB     `gorm:"type:jsonb;not null" json:"cleansing_logic"`                // 清洗逻辑模板
	Parameters      JSONB     `gorm:"type:jsonb" json:"parameters"`                              // 可配置参数定义
	DefaultConfig   JSONB     `gorm:"type:jsonb" json:"default_config"`                          // 默认配置
	ApplicableTypes JSONB     `gorm:"type:jsonb" json:"applicable_types"`                        // 适用的数据类型
	ComplexityLevel string    `gorm:"type:varchar(20);default:'medium'" json:"complexity_level"` // low/medium/high
	IsBuiltIn       bool      `gorm:"default:false" json:"is_built_in"`                          // 是否为内置模板
	IsEnabled       bool      `gorm:"default:true" json:"is_enabled"`
	Version         string    `gorm:"type:varchar(20);default:'1.0'" json:"version"`
	Tags            JSONB     `gorm:"type:jsonb" json:"tags"` // 标签，用于分类和搜索
	CreatedBy       string    `gorm:"type:varchar(50)" json:"created_by"`
	UpdatedBy       string    `gorm:"type:varchar(50)" json:"updated_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	DeletedAt       time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// DataCleansingConfig 数据清洗配置（运行时应用）
type DataCleansingConfig struct {
	TemplateID       string                 `json:"template_id"`
	TargetFields     []string               `json:"target_fields"`     // 目标字段列表
	TriggerCondition string                 `json:"trigger_condition"` // 触发条件
	CleansingConfig  map[string]interface{} `json:"cleansing_config"`  // 运行时清洗配置
	PreCondition     string                 `json:"pre_condition"`     // 前置条件
	PostCondition    string                 `json:"post_condition"`    // 后置条件
	BackupOriginal   bool                   `json:"backup_original"`   // 是否备份原始数据
	ValidationRules  map[string]interface{} `json:"validation_rules"`  // 验证规则
	ErrorHandling    map[string]interface{} `json:"error_handling"`    // 错误处理策略
	IsEnabled        bool                   `json:"is_enabled"`
}

// TableName 指定表名
func (DataCleansingTemplate) TableName() string {
	return "data_cleansing_templates"
}

// BeforeCreate 创建前钩子
func (d *DataCleansingTemplate) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedBy == "" {
		d.CreatedBy = "system"
	}
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (d *DataCleansingTemplate) BeforeUpdate(tx *gorm.DB) error {
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
	}
	return nil
}

// QualityDashboardReport 质量仪表板报告模型
type QualityDashboardReport struct {
	ID                  string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	ReportName          string     `gorm:"type:varchar(100);not null" json:"report_name"`
	ReportType          string     `gorm:"type:varchar(30);not null" json:"report_type"` // daily, weekly, monthly, ad_hoc
	ReportPeriod        string     `gorm:"type:varchar(50);not null" json:"report_period"`
	StartDate           time.Time  `json:"start_date"`
	EndDate             time.Time  `json:"end_date"`
	OverallQualityScore float64    `json:"overall_quality_score"`
	CompletenessScore   float64    `json:"completeness_score"`
	AccuracyScore       float64    `json:"accuracy_score"`
	ConsistencyScore    float64    `json:"consistency_score"`
	ValidityScore       float64    `json:"validity_score"`
	UniquenessScore     float64    `json:"uniqueness_score"`
	TimelinessScore     float64    `json:"timeliness_score"`
	TotalTablesChecked  int        `json:"total_tables_checked"`
	TotalRulesExecuted  int        `json:"total_rules_executed"`
	TotalRecordsChecked int64      `json:"total_records_checked"`
	TotalIssuesFound    int64      `json:"total_issues_found"`
	CriticalIssues      int        `json:"critical_issues"`
	HighIssues          int        `json:"high_issues"`
	MediumIssues        int        `json:"medium_issues"`
	LowIssues           int        `json:"low_issues"`
	ReportData          JSONB      `gorm:"type:jsonb" json:"report_data"`                  // 报告详细数据
	TrendAnalysis       JSONB      `gorm:"type:jsonb" json:"trend_analysis"`               // 趋势分析
	Recommendations     JSONB      `gorm:"type:jsonb" json:"recommendations"`              // 改进建议
	Status              string     `gorm:"type:varchar(20);default:'draft'" json:"status"` // draft, published, archived
	GeneratedBy         string     `gorm:"type:varchar(50)" json:"generated_by"`
	ReviewedBy          string     `gorm:"type:varchar(50)" json:"reviewed_by"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityDashboardReport) TableName() string {
	return "quality_dashboard_reports"
}

// BeforeCreate 创建前钩子
func (q *QualityDashboardReport) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.GeneratedBy == "" {
		q.GeneratedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityDashboardReport) BeforeUpdate(tx *gorm.DB) error {
	// 更新时不自动设置字段，保持原有逻辑
	return nil
}

// QualityIssueTracker 质量问题追踪模型
type QualityIssueTracker struct {
	ID               string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	QualityCheckID   string     `gorm:"type:varchar(50);not null;index" json:"quality_check_id"`
	QualityRuleID    string     `gorm:"type:varchar(50);not null;index" json:"quality_rule_id"`
	IssueType        string     `gorm:"type:varchar(50);not null" json:"issue_type"`
	Severity         string     `gorm:"type:varchar(20);not null" json:"severity"` // low, medium, high, critical
	TargetTable      string     `gorm:"type:varchar(100);not null" json:"target_table"`
	TargetColumn     string     `gorm:"type:varchar(100)" json:"target_column"`
	RecordIdentifier string     `gorm:"type:text" json:"record_identifier"` // 记录标识符
	IssueDescription string     `gorm:"type:text;not null" json:"issue_description"`
	ExpectedValue    string     `gorm:"type:text" json:"expected_value"`
	ActualValue      string     `gorm:"type:text" json:"actual_value"`
	IssueContext     JSONB      `gorm:"type:jsonb" json:"issue_context"` // 问题上下文
	DetectionTime    time.Time  `json:"detection_time"`
	Status           string     `gorm:"type:varchar(20);default:'open'" json:"status"` // open, investigating, resolved, ignored, false_positive
	AssignedTo       string     `gorm:"type:varchar(50)" json:"assigned_to"`
	ResolutionNote   string     `gorm:"type:text" json:"resolution_note"`
	ResolutionAction JSONB      `gorm:"type:jsonb" json:"resolution_action"` // 解决动作
	ResolvedBy       string     `gorm:"type:varchar(50)" json:"resolved_by"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	ReopenCount      int        `gorm:"default:0" json:"reopen_count"`
	Tags             JSONB      `gorm:"type:jsonb" json:"tags"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityIssueTracker) TableName() string {
	return "quality_issue_trackers"
}

// BeforeCreate 创建前钩子
func (q *QualityIssueTracker) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityIssueTracker) BeforeUpdate(tx *gorm.DB) error {
	// 更新时不自动设置字段，保持原有逻辑
	return nil
}

// ScheduleType 调度类型
type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"     // cron表达式
	ScheduleTypeInterval ScheduleType = "interval" // 间隔执行
	ScheduleTypeOnce     ScheduleType = "once"     // 一次性执行
	ScheduleTypeManual   ScheduleType = "manual"   // 手动执行
)

// ScheduleConfig 调度配置结构体
type ScheduleConfig struct {
	Type      string     `json:"type"`       // cron, interval, once, manual
	CronExpr  string     `json:"cron_expr"`  // cron表达式 (当type=cron时)
	Interval  int64      `json:"interval"`   // 间隔秒数 (当type=interval时)
	StartTime *time.Time `json:"start_time"` // 开始时间 (当type=once时)
}

// NotificationConfig 通知配置结构体
type NotificationConfig struct {
	Enabled         bool     `json:"enabled"`
	NotifyOnSuccess bool     `json:"notify_on_success"`
	NotifyOnFailure bool     `json:"notify_on_failure"`
	Recipients      []string `json:"recipients"`
	Channels        []string `json:"channels"` // email, webhook等
}

// QualityTask 质量检测任务模型
type QualityTask struct {
	ID              string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	Name            string     `gorm:"type:varchar(100);not null" json:"name"`
	Description     string     `gorm:"type:text" json:"description"`
	LibraryType     string     `gorm:"type:varchar(30);not null;index" json:"library_type"` // thematic(主题库), basic(基础库)
	LibraryID       string     `gorm:"type:varchar(50);not null;index" json:"library_id"`   // 库ID
	InterfaceID     string     `gorm:"type:varchar(50);not null;index" json:"interface_id"` // 接口ID
	TargetSchema    string     `gorm:"type:varchar(100)" json:"target_schema"`              // 目标schema
	TargetTable     string     `gorm:"type:varchar(100)" json:"target_table"`               // 目标表名
	ScheduleType    string     `gorm:"type:varchar(20);not null" json:"schedule_type"`      // cron, interval, once, manual
	CronExpression  string     `gorm:"type:varchar(100)" json:"cron_expression"`            // cron表达式
	IntervalSeconds int64      `gorm:"default:0" json:"interval_seconds"`                   // 间隔秒数
	ScheduledTime   *time.Time `json:"scheduled_time"`                                      // 计划执行时间(once类型)
	NotifyEnabled   bool       `gorm:"default:false" json:"notify_enabled"`
	NotifyOnSuccess bool       `gorm:"default:false" json:"notify_on_success"`
	NotifyOnFailure bool       `gorm:"default:true" json:"notify_on_failure"`
	Recipients      JSONB      `gorm:"type:jsonb" json:"recipients"`                     // 通知接收人列表
	NotifyChannels  JSONB      `gorm:"type:jsonb" json:"notify_channels"`                // 通知渠道
	Status          string     `gorm:"type:varchar(30);default:'pending'" json:"status"` // pending, running, completed, failed, completed_with_issues
	Priority        int        `gorm:"default:50" json:"priority"`                       // 优先级 (1-100)
	IsEnabled       bool       `gorm:"default:true" json:"is_enabled"`
	LastExecuted    *time.Time `json:"last_executed,omitempty"`
	NextExecution   *time.Time `json:"next_execution,omitempty"`
	ExecutionCount  int64      `gorm:"default:0" json:"execution_count"`
	SuccessCount    int64      `gorm:"default:0" json:"success_count"`
	FailureCount    int64      `gorm:"default:0" json:"failure_count"`
	CreatedBy       string     `gorm:"type:varchar(50)" json:"created_by"`
	UpdatedBy       string     `gorm:"type:varchar(50)" json:"updated_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityTask) TableName() string {
	return "quality_tasks"
}

// BeforeCreate 创建前钩子
func (q *QualityTask) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.CreatedBy == "" {
		q.CreatedBy = "system"
	}
	if q.UpdatedBy == "" {
		q.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityTask) BeforeUpdate(tx *gorm.DB) error {
	if q.UpdatedBy == "" {
		q.UpdatedBy = "system"
	}
	return nil
}

// QualityTaskExecution 质量检测任务执行记录模型
type QualityTaskExecution struct {
	ID                 string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	TaskID             string     `gorm:"type:varchar(50);not null;index" json:"task_id"`
	ExecutionType      string     `gorm:"type:varchar(30);not null" json:"execution_type"` // scheduled, manual, triggered
	StartTime          time.Time  `json:"start_time"`
	EndTime            *time.Time `json:"end_time,omitempty"`
	Duration           int64      `json:"duration"`                                // 执行时长，毫秒
	Status             string     `gorm:"type:varchar(30);not null" json:"status"` // running, completed, failed, cancelled, completed_with_issues
	TotalRulesExecuted int        `json:"total_rules_executed"`
	PassedRules        int        `json:"passed_rules"`
	FailedRules        int        `json:"failed_rules"`
	OverallScore       float64    `json:"overall_score"`                       // 总体质量评分 (0-1)
	IssueCount         int64      `json:"issue_count"`                         // 问题记录数量
	ExecutionResults   JSONB      `gorm:"type:jsonb" json:"execution_results"` // 执行结果详情
	ErrorMessage       string     `gorm:"type:text" json:"error_message,omitempty"`
	TriggerSource      string     `gorm:"type:varchar(50)" json:"trigger_source"` // 触发来源
	ExecutedBy         string     `gorm:"type:varchar(50)" json:"executed_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityTaskExecution) TableName() string {
	return "quality_task_executions"
}

// BeforeCreate 创建前钩子
func (q *QualityTaskExecution) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.ExecutedBy == "" {
		q.ExecutedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityTaskExecution) BeforeUpdate(tx *gorm.DB) error {
	// 更新时不自动设置字段，保持原有逻辑
	return nil
}

// DataLineage 数据血缘关系模型
type DataLineage struct {
	ID               string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	SourceObjectID   string    `gorm:"type:varchar(50);not null;index" json:"source_object_id"`
	SourceObjectType string    `gorm:"type:varchar(30);not null" json:"source_object_type"` // table, interface, thematic_interface
	TargetObjectID   string    `gorm:"type:varchar(50);not null;index" json:"target_object_id"`
	TargetObjectType string    `gorm:"type:varchar(30);not null" json:"target_object_type"`
	RelationType     string    `gorm:"type:varchar(30);not null" json:"relation_type"` // direct, derived, aggregated, transformed
	TransformRule    JSONB     `gorm:"type:jsonb" json:"transform_rule"`               // 转换规则
	ColumnMapping    JSONB     `gorm:"type:jsonb" json:"column_mapping"`               // 字段映射关系
	Confidence       float64   `gorm:"default:1.0" json:"confidence"`                  // 置信度 (0-1)
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	Description      string    `gorm:"type:text" json:"description"`
	CreatedBy        string    `gorm:"type:varchar(50)" json:"created_by"`
	UpdatedBy        string    `gorm:"type:varchar(50)" json:"updated_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	DeletedAt        time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (DataLineage) TableName() string {
	return "data_lineages"
}

// BeforeCreate 创建前钩子
func (d *DataLineage) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedBy == "" {
		d.CreatedBy = "system"
	}
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (d *DataLineage) BeforeUpdate(tx *gorm.DB) error {
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
	}
	return nil
}

// QualityTaskFieldRule 质量检测任务字段规则配置模型
type QualityTaskFieldRule struct {
	ID             string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	TaskID         string    `gorm:"type:varchar(50);not null;index" json:"task_id"`    // 关联任务ID
	FieldName      string    `gorm:"type:varchar(100);not null" json:"field_name"`      // 字段名称
	RuleTemplateID string    `gorm:"type:varchar(50);not null" json:"rule_template_id"` // 规则模板ID
	RuntimeConfig  JSONB     `gorm:"type:jsonb" json:"runtime_config"`                  // 运行时配置
	Threshold      JSONB     `gorm:"type:jsonb" json:"threshold"`                       // 阈值配置
	IsEnabled      bool      `gorm:"default:true" json:"is_enabled"`
	Priority       int       `gorm:"default:50" json:"priority"` // 规则执行优先级
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DeletedAt      time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityTaskFieldRule) TableName() string {
	return "quality_task_field_rules"
}

// BeforeCreate 创建前钩子
func (q *QualityTaskFieldRule) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityTaskFieldRule) BeforeUpdate(tx *gorm.DB) error {
	return nil
}

// QualityIssueRecord 质量问题数据记录模型
type QualityIssueRecord struct {
	ID               string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	ExecutionID      string    `gorm:"type:varchar(50);not null;index" json:"execution_id"` // 关联执行记录ID
	TaskID           string    `gorm:"type:varchar(50);not null;index" json:"task_id"`      // 关联任务ID
	FieldName        string    `gorm:"type:varchar(100);not null;index" json:"field_name"`  // 问题字段名
	RuleTemplateID   string    `gorm:"type:varchar(50);not null" json:"rule_template_id"`   // 触发的规则ID
	RecordIdentifier string    `gorm:"type:text;not null" json:"record_identifier"`         // 问题数据的主键/唯一标识
	IssueType        string    `gorm:"type:varchar(50);not null" json:"issue_type"`         // 问题类型
	IssueDescription string    `gorm:"type:text;not null" json:"issue_description"`         // 问题描述
	FieldValue       string    `gorm:"type:text" json:"field_value"`                        // 问题字段值
	ExpectedValue    string    `gorm:"type:text" json:"expected_value"`                     // 期望值
	Severity         string    `gorm:"type:varchar(20);not null;index" json:"severity"`     // 严重程度: low, medium, high, critical
	CreatedAt        time.Time `json:"created_at"`
	DeletedAt        time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (QualityIssueRecord) TableName() string {
	return "quality_issue_records"
}

// BeforeCreate 创建前钩子
func (q *QualityIssueRecord) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityIssueRecord) BeforeUpdate(tx *gorm.DB) error {
	return nil
}
