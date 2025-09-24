/*
 * @module service/thematic_sync/types
 * @description 主题同步引擎的统一类型定义
 * @architecture 数据传输对象模式 - 统一定义所有同步引擎相关的类型
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 数据结构定义 -> 类型转换 -> 流程传递 -> 结果封装
 * @rules 确保数据结构的一致性和类型安全，遵循单一职责原则
 * @dependencies time, context, database/sql/driver
 * @refs sync_engine.go, data_fetcher.go, data_processor.go
 */

package thematic_sync

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ==================== 同步执行相关类型 ====================

// SyncExecutionPhase 同步执行阶段
type SyncExecutionPhase string

const (
	PhaseInitialize  SyncExecutionPhase = "initialize"  // 初始化
	PhaseDataFetch   SyncExecutionPhase = "data_fetch"  // 数据获取
	PhaseAggregation SyncExecutionPhase = "aggregation" // 数据汇聚
	PhaseGovernance  SyncExecutionPhase = "governance"  // 数据治理处理
	PhaseDataWrite   SyncExecutionPhase = "data_write"  // 数据写入
	PhaseLineage     SyncExecutionPhase = "lineage"     // 血缘记录
	PhaseComplete    SyncExecutionPhase = "complete"    // 完成
)

// SyncRequest 同步请求
type SyncRequest struct {
	TaskID            string                 `json:"task_id"`
	ExecutionType     string                 `json:"execution_type"` // manual, scheduled, retry
	SourceLibraries   []string               `json:"source_libraries"`
	SourceInterfaces  []string               `json:"source_interfaces"`
	TargetLibraryID   string                 `json:"target_library_id"`
	TargetInterfaceID string                 `json:"target_interface_id"`
	Config            map[string]interface{} `json:"config"`
	Context           context.Context        `json:"-"`
}

// SyncProgress 同步进度
type SyncProgress struct {
	ExecutionID    string             `json:"execution_id"`
	CurrentPhase   SyncExecutionPhase `json:"current_phase"`
	Progress       float64            `json:"progress"` // 0-100
	ProcessedCount int64              `json:"processed_count"`
	TotalCount     int64              `json:"total_count"`
	ErrorCount     int64              `json:"error_count"`
	Message        string             `json:"message"`
	StartTime      time.Time          `json:"start_time"`
	LastUpdateTime time.Time          `json:"last_update_time"`
}

// SyncResponse 同步响应
type SyncResponse struct {
	ExecutionID    string               `json:"execution_id"`
	Status         string               `json:"status"`
	Result         *SyncExecutionResult `json:"result,omitempty"`
	Error          string               `json:"error,omitempty"`
	Progress       *SyncProgress        `json:"progress,omitempty"`
	ProcessingTime time.Duration        `json:"processing_time"`
}

// SyncExecutionResult 同步执行结果
type SyncExecutionResult struct {
	SourceRecordCount    int64                `json:"source_record_count"`
	ProcessedRecordCount int64                `json:"processed_record_count"`
	InsertedRecordCount  int64                `json:"inserted_record_count"`
	UpdatedRecordCount   int64                `json:"updated_record_count"`
	ErrorRecordCount     int64                `json:"error_record_count"`
	QualityScore         float64              `json:"quality_score"`
	ProcessingSteps      []ProcessingStepInfo `json:"processing_steps"`
}

// ProcessingStepInfo 处理步骤信息
type ProcessingStepInfo struct {
	Phase       SyncExecutionPhase `json:"phase"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	Duration    time.Duration      `json:"duration"`
	RecordCount int64              `json:"record_count"`
	ErrorCount  int64              `json:"error_count"`
	Status      string             `json:"status"`
	Message     string             `json:"message"`
}

// ==================== 数据源配置相关类型 ====================

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

// SourceLibraryConfig 源库配置
type SourceLibraryConfig struct {
	LibraryID         string                 `json:"library_id"`
	InterfaceID       string                 `json:"interface_id"`
	SQLQuery          string                 `json:"sql_query,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	Filters           []FilterConfig         `json:"filters,omitempty"`
	Transforms        []TransformConfig      `json:"transforms,omitempty"`
	IncrementalConfig *IncrementalConfig     `json:"incremental_config,omitempty"`
}

// SourceInterfaceConfig 源接口配置
type SourceInterfaceConfig struct {
	InterfaceID       string             `json:"interface_id"`
	FieldMapping      []FieldMapping     `json:"field_mapping,omitempty"`
	FilterCondition   string             `json:"filter_condition,omitempty"`
	BatchSize         int                `json:"batch_size,omitempty"`
	Parameters        map[string]string  `json:"parameters,omitempty"`
	IncrementalConfig *IncrementalConfig `json:"incremental_config,omitempty"`
}

// FieldMapping 字段映射
type FieldMapping struct {
	SourceField  string      `json:"source_field"`
	TargetField  string      `json:"target_field"`
	Transform    string      `json:"transform,omitempty"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
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

// ==================== SQL数据源相关类型 ====================

// SQLDataSourceConfig SQL数据源配置
type SQLDataSourceConfig struct {
	LibraryID   string                 `json:"library_id"`
	InterfaceID string                 `json:"interface_id"`
	SQLQuery    string                 `json:"sql_query"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timeout     int                    `json:"timeout"`  // 查询超时时间（秒）
	MaxRows     int                    `json:"max_rows"` // 最大返回行数
}

// ==================== 数据治理相关类型 ====================

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
	OverallQualityScore   float64       `json:"overall_quality_score"`
	TotalProcessedRecords int64         `json:"total_processed_records"`
	TotalCleansingApplied int64         `json:"total_cleansing_applied"`
	TotalMaskingApplied   int64         `json:"total_masking_applied"`
	TotalValidationErrors int64         `json:"total_validation_errors"`
	ExecutionTime         time.Duration `json:"execution_time"`
	ComplianceStatus      string        `json:"compliance_status"`
}

// ==================== 执行选项相关类型 ====================

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

// ==================== JSONB转换支持 ====================

// Value 实现 driver.Valuer 接口
func (ic IncrementalConfig) Value() (driver.Value, error) {
	return json.Marshal(ic)
}

// Scan 实现 sql.Scanner 接口
func (ic *IncrementalConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into IncrementalConfig", value)
	}
	return json.Unmarshal(bytes, ic)
}
