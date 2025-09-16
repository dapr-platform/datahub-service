/*
 * @module service/thematic_library/thematic_sync_config_types
 * @description 主题同步配置的结构化类型定义，替代map[string]interface{}和JSON字符串
 * @architecture 数据传输对象 - 强类型的配置结构定义
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow N/A
 * @rules 所有配置都使用强类型结构，避免map[string]interface{}
 * @dependencies time, database/sql/driver
 * @refs thematic_sync_types.go, models/thematic_sync.go
 */

package thematic_library

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ==================== 调度配置相关类型 ====================

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	Type            string           `json:"type" validate:"required,oneof=manual one_time interval cron"` // manual, one_time, interval, cron
	CronExpression  string           `json:"cron_expression,omitempty"`                                    // Cron表达式
	IntervalSeconds int              `json:"interval_seconds,omitempty"`                                   // 间隔秒数
	ScheduledTime   *time.Time       `json:"scheduled_time,omitempty"`                                     // 计划执行时间
	TimeZone        string           `json:"timezone,omitempty" default:"Asia/Shanghai"`                   // 时区
	MaxRetries      int              `json:"max_retries,omitempty" default:"3"`                            // 最大重试次数
	RetryInterval   int              `json:"retry_interval,omitempty" default:"300"`                       // 重试间隔(秒)
	Enabled         bool             `json:"enabled" default:"true"`                                       // 是否启用
	StartDate       *time.Time       `json:"start_date,omitempty"`                                         // 开始日期
	EndDate         *time.Time       `json:"end_date,omitempty"`                                           // 结束日期
	ExecutionWindow *ExecutionWindow `json:"execution_window,omitempty"`                                   // 执行时间窗口
}

// ExecutionWindow 执行时间窗口
type ExecutionWindow struct {
	StartTime string `json:"start_time" example:"09:00"` // 开始时间 HH:MM
	EndTime   string `json:"end_time" example:"18:00"`   // 结束时间 HH:MM
	Days      []int  `json:"days,omitempty"`             // 允许执行的星期几 (1-7)
	Holidays  bool   `json:"holidays" default:"false"`   // 是否在节假日执行
}

// ==================== 数据源配置相关类型 ====================

// SourceLibraryConfig 源库配置
type SourceLibraryConfig struct {
	LibraryID   string                  `json:"library_id" validate:"required"`
	Interfaces  []SourceInterfaceConfig `json:"interfaces" validate:"required,min=1"`
	FilterRules []DataFilterRule        `json:"filter_rules,omitempty"`
	Priority    int                     `json:"priority" default:"1"`     // 优先级，数字越小优先级越高
	Enabled     bool                    `json:"enabled" default:"true"`   // 是否启用
	SyncMode    string                  `json:"sync_mode" default:"full"` // full, incremental, realtime
}

// SourceInterfaceConfig 源接口配置
type SourceInterfaceConfig struct {
	InterfaceID     string            `json:"interface_id" validate:"required"`
	FieldMapping    []FieldMapping    `json:"field_mapping,omitempty"`
	FilterCondition string            `json:"filter_condition,omitempty"` // SQL WHERE 条件
	SortOrder       []SortField       `json:"sort_order,omitempty"`
	BatchSize       int               `json:"batch_size,omitempty" default:"1000"`
	Parameters      map[string]string `json:"parameters,omitempty"` // 接口参数
}

// FieldMapping 字段映射
type FieldMapping struct {
	SourceField  string      `json:"source_field" validate:"required"`
	TargetField  string      `json:"target_field" validate:"required"`
	Transform    string      `json:"transform,omitempty"` // 转换函数
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

// ==================== 汇聚配置相关类型 ====================

// AggregationConfig 汇聚配置
type AggregationConfig struct {
	Strategy        string                 `json:"strategy" validate:"required,oneof=merge union intersect"`
	ConflictPolicy  string                 `json:"conflict_policy" validate:"required,oneof=first last priority custom"`
	MergeRules      []MergeRule            `json:"merge_rules,omitempty"`
	GroupByFields   []string               `json:"group_by_fields,omitempty"`
	AggregateFields []AggregateField       `json:"aggregate_fields,omitempty"`
	CustomRules     []CustomAggregateRule  `json:"custom_rules,omitempty"`
	Validation      *AggregationValidation `json:"validation,omitempty"`
}

// MergeRule 合并规则
type MergeRule struct {
	Field      string `json:"field" validate:"required"`
	MergeType  string `json:"merge_type" validate:"required,oneof=first last max min sum avg concat"`
	Separator  string `json:"separator,omitempty" default:","` // 用于concat类型
	NullPolicy string `json:"null_policy" default:"ignore"`    // ignore, include, error
	UniqueOnly bool   `json:"unique_only" default:"false"`     // 是否去重
}

// AggregateField 聚合字段
type AggregateField struct {
	SourceField string `json:"source_field" validate:"required"`
	TargetField string `json:"target_field" validate:"required"`
	Function    string `json:"function" validate:"required,oneof=count sum avg max min"`
	Condition   string `json:"condition,omitempty"` // 聚合条件
}

// CustomAggregateRule 自定义聚合规则
type CustomAggregateRule struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description,omitempty"`
	Expression  string            `json:"expression" validate:"required"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Priority    int               `json:"priority" default:"1"`
}

// AggregationValidation 汇聚验证
type AggregationValidation struct {
	MinRecords        int      `json:"min_records,omitempty"`
	MaxRecords        int      `json:"max_records,omitempty"`
	RequiredFields    []string `json:"required_fields,omitempty"`
	UniqueFields      []string `json:"unique_fields,omitempty"`
	ValidateIntegrity bool     `json:"validate_integrity" default:"true"`
}

// ==================== 主键匹配规则相关类型 ====================

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
	Pattern    string `json:"pattern,omitempty"` // 正则表达式
	MinLength  int    `json:"min_length,omitempty"`
	MaxLength  int    `json:"max_length,omitempty"`
	AllowEmpty bool   `json:"allow_empty" default:"false"`
}

// FuzzyMatchConfig 模糊匹配配置
type FuzzyMatchConfig struct {
	Enabled       bool             `json:"enabled" default:"false"`
	Algorithm     string           `json:"algorithm" default:"levenshtein"` // levenshtein, jaro, soundex
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

// ==================== 字段映射规则相关类型 ====================

// FieldMappingRules 字段映射规则
type FieldMappingRules struct {
	Mappings       []FieldMappingRule    `json:"mappings" validate:"required,min=1"`
	DefaultPolicy  string                `json:"default_policy" default:"ignore"` // ignore, error, null
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

// ==================== 数据清洗规则相关类型 ====================

// CleansingRules 数据清洗规则
type CleansingRules struct {
	Rules      []CleansingRule      `json:"rules" validate:"required,min=1"`
	Order      []string             `json:"order,omitempty"`
	OnError    string               `json:"on_error" default:"skip"` // skip, abort, continue
	Parallel   bool                 `json:"parallel" default:"false"`
	Validation *CleansingValidation `json:"validation,omitempty"`
}

// CleansingRule 清洗规则
type CleansingRule struct {
	ID          string                 `json:"id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Type        string                 `json:"type" validate:"required,oneof=remove_duplicates trim standardize validate replace"`
	Fields      []string               `json:"fields,omitempty"`
	Condition   string                 `json:"condition,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Priority    int                    `json:"priority" default:"1"`
	Enabled     bool                   `json:"enabled" default:"true"`
	Description string                 `json:"description,omitempty"`
}

// CleansingValidation 清洗验证
type CleansingValidation struct {
	ValidateBeforeClean bool     `json:"validate_before_clean" default:"true"`
	ValidateAfterClean  bool     `json:"validate_after_clean" default:"true"`
	RequiredFields      []string `json:"required_fields,omitempty"`
	QualityThreshold    float64  `json:"quality_threshold" default:"0.8"`
}

// ==================== 隐私脱敏规则相关类型 ====================

// PrivacyRules 隐私脱敏规则
type PrivacyRules struct {
	Rules      []PrivacyRule      `json:"rules" validate:"required,min=1"`
	GlobalMode string             `json:"global_mode" default:"field"` // field, record, table
	AuditLog   bool               `json:"audit_log" default:"true"`
	Validation *PrivacyValidation `json:"validation,omitempty"`
}

// PrivacyRule 隐私规则
type PrivacyRule struct {
	ID          string                 `json:"id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Fields      []string               `json:"fields" validate:"required,min=1"`
	MaskingType string                 `json:"masking_type" validate:"required,oneof=mask hash encrypt tokenize anonymize"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Condition   string                 `json:"condition,omitempty"`
	Reversible  bool                   `json:"reversible" default:"false"`
	Enabled     bool                   `json:"enabled" default:"true"`
	Priority    int                    `json:"priority" default:"1"`
	Description string                 `json:"description,omitempty"`
}

// PrivacyValidation 隐私验证
type PrivacyValidation struct {
	ValidatePatterns  bool     `json:"validate_patterns" default:"true"`
	SensitiveFields   []string `json:"sensitive_fields,omitempty"`
	ComplianceRules   []string `json:"compliance_rules,omitempty"`
	AuditRequirements bool     `json:"audit_requirements" default:"true"`
}

// ==================== 数据质量规则相关类型 ====================

// QualityRules 数据质量规则
type QualityRules struct {
	Rules      []QualityRule      `json:"rules" validate:"required,min=1"`
	Threshold  float64            `json:"threshold" default:"0.8"`
	OnFailure  string             `json:"on_failure" default:"warn"` // warn, error, skip
	Metrics    []QualityMetric    `json:"metrics,omitempty"`
	Validation *QualityValidation `json:"validation,omitempty"`
}

// QualityRule 质量规则
type QualityRule struct {
	ID          string                 `json:"id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Type        string                 `json:"type" validate:"required,oneof=completeness accuracy consistency validity uniqueness timeliness"`
	Fields      []string               `json:"fields,omitempty"`
	Expression  string                 `json:"expression,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Weight      float64                `json:"weight" default:"1.0"`
	Threshold   float64                `json:"threshold" default:"0.8"`
	Enabled     bool                   `json:"enabled" default:"true"`
	Description string                 `json:"description,omitempty"`
}

// QualityMetric 质量指标
type QualityMetric struct {
	Name        string  `json:"name" validate:"required"`
	Type        string  `json:"type" validate:"required,oneof=count rate percentage score"`
	Target      float64 `json:"target"`
	Warning     float64 `json:"warning"`
	Critical    float64 `json:"critical"`
	Description string  `json:"description,omitempty"`
}

// QualityValidation 质量验证
type QualityValidation struct {
	EnablePreCheck   bool     `json:"enable_pre_check" default:"true"`
	EnablePostCheck  bool     `json:"enable_post_check" default:"true"`
	CriticalFields   []string `json:"critical_fields,omitempty"`
	BusinessRules    []string `json:"business_rules,omitempty"`
	StatisticalCheck bool     `json:"statistical_check" default:"true"`
}

// ==================== 执行配置相关类型 ====================

// ExecutionConfig 执行配置
type ExecutionConfig struct {
	Mode           string              `json:"mode" validate:"required,oneof=immediate scheduled test"`
	Parallel       bool                `json:"parallel" default:"false"`
	MaxWorkers     int                 `json:"max_workers" default:"1"`
	BatchSize      int                 `json:"batch_size" default:"1000"`
	Timeout        int                 `json:"timeout" default:"3600"` // 超时时间(秒)
	RetryPolicy    *RetryPolicy        `json:"retry_policy,omitempty"`
	Notification   *NotificationConfig `json:"notification,omitempty"`
	Monitoring     *MonitoringConfig   `json:"monitoring,omitempty"`
	CustomSettings map[string]string   `json:"custom_settings,omitempty"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries      int      `json:"max_retries" default:"3"`
	RetryInterval   int      `json:"retry_interval" default:"300"` // 重试间隔(秒)
	BackoffFactor   float64  `json:"backoff_factor" default:"2.0"` // 退避因子
	MaxInterval     int      `json:"max_interval" default:"3600"`  // 最大间隔(秒)
	RetryConditions []string `json:"retry_conditions,omitempty"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Enabled   bool                  `json:"enabled" default:"false"`
	OnSuccess []string              `json:"on_success,omitempty"`
	OnFailure []string              `json:"on_failure,omitempty"`
	OnWarning []string              `json:"on_warning,omitempty"`
	Channels  []NotificationChannel `json:"channels,omitempty"`
	Template  string                `json:"template,omitempty"`
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
	Type       string            `json:"type" validate:"required,oneof=email sms webhook"`
	Target     string            `json:"target" validate:"required"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Enabled    bool              `json:"enabled" default:"true"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled         bool     `json:"enabled" default:"true"`
	MetricsInterval int      `json:"metrics_interval" default:"60"` // 指标收集间隔(秒)
	LogLevel        string   `json:"log_level" default:"INFO"`
	TraceEnabled    bool     `json:"trace_enabled" default:"false"`
	AlertRules      []string `json:"alert_rules,omitempty"`
}

// ==================== JSONB 转换方法 ====================

// Value 实现 driver.Valuer 接口，用于将结构体转换为 JSONB
func (sc ScheduleConfig) Value() (driver.Value, error) {
	return json.Marshal(sc)
}

// Scan 实现 sql.Scanner 接口，用于将 JSONB 转换为结构体
func (sc *ScheduleConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into ScheduleConfig", value)
	}

	return json.Unmarshal(bytes, sc)
}

// 为其他主要配置类型实现 JSONB 转换方法
func (slc SourceLibraryConfig) Value() (driver.Value, error) {
	return json.Marshal(slc)
}

func (slc *SourceLibraryConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into SourceLibraryConfig", value)
	}
	return json.Unmarshal(bytes, slc)
}

func (ac AggregationConfig) Value() (driver.Value, error) {
	return json.Marshal(ac)
}

func (ac *AggregationConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into AggregationConfig", value)
	}
	return json.Unmarshal(bytes, ac)
}

func (kmr KeyMatchingRules) Value() (driver.Value, error) {
	return json.Marshal(kmr)
}

func (kmr *KeyMatchingRules) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into KeyMatchingRules", value)
	}
	return json.Unmarshal(bytes, kmr)
}

func (fmr FieldMappingRules) Value() (driver.Value, error) {
	return json.Marshal(fmr)
}

func (fmr *FieldMappingRules) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into FieldMappingRules", value)
	}
	return json.Unmarshal(bytes, fmr)
}

func (cr CleansingRules) Value() (driver.Value, error) {
	return json.Marshal(cr)
}

func (cr *CleansingRules) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into CleansingRules", value)
	}
	return json.Unmarshal(bytes, cr)
}

func (pr PrivacyRules) Value() (driver.Value, error) {
	return json.Marshal(pr)
}

func (pr *PrivacyRules) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into PrivacyRules", value)
	}
	return json.Unmarshal(bytes, pr)
}

func (qr QualityRules) Value() (driver.Value, error) {
	return json.Marshal(qr)
}

func (qr *QualityRules) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into QualityRules", value)
	}
	return json.Unmarshal(bytes, qr)
}

func (ec ExecutionConfig) Value() (driver.Value, error) {
	return json.Marshal(ec)
}

func (ec *ExecutionConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into ExecutionConfig", value)
	}
	return json.Unmarshal(bytes, ec)
}
