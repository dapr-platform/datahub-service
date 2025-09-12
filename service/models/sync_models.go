/*
 * @module service/models/sync_models
 * @description 数据同步相关模型，包含同步配置、执行记录、增量状态、错误日志等模型
 * @architecture 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 同步配置 -> 任务执行 -> 状态记录 -> 结果分析
 * @rules 确保同步状态数据的完整性和一致性，支持同步任务的全生命周期管理
 * @dependencies gorm.io/gorm, time
 * @refs service/sync_engine/, service/scheduler/
 */

package models

import (
	"time"
)

// SyncConfig 同步配置模型
type SyncConfig struct {
	ID                string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	Name              string    `gorm:"type:varchar(100);not null" json:"name"`
	Description       string    `gorm:"type:text" json:"description"`
	DataSourceID      string    `gorm:"type:varchar(50);not null" json:"datasource_id"`
	InterfaceID       string    `gorm:"type:varchar(50);not null" json:"interface_id"`
	SyncType          string    `gorm:"type:varchar(20);not null" json:"sync_type"` // realtime, batch
	ScheduleConfig    JSONB     `gorm:"type:jsonb" json:"schedule_config"`          // 调度配置
	TransformRules    JSONB     `gorm:"type:jsonb" json:"transform_rules"`          // 转换规则
	FilterRules       JSONB     `gorm:"type:jsonb" json:"filter_rules"`             // 过滤规则
	TargetTable       string    `gorm:"type:varchar(100);not null" json:"target_table"`
	IncrementalConfig JSONB     `gorm:"type:jsonb" json:"incremental_config"`            // 增量配置
	Status            string    `gorm:"type:varchar(20);default:'active'" json:"status"` // active, inactive, paused
	MaxRetries        int       `gorm:"default:3" json:"max_retries"`
	TimeoutSeconds    int       `gorm:"default:3600" json:"timeout_seconds"`
	CreatedBy         string    `gorm:"type:varchar(50)" json:"created_by"`
	UpdatedBy         string    `gorm:"type:varchar(50)" json:"updated_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (SyncConfig) TableName() string {
	return "sync_configs"
}

// IncrementalState 增量同步状态模型
type IncrementalState struct {
	ID              string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	SyncConfigID    string    `gorm:"type:varchar(50);not null;index" json:"sync_config_id"`
	IncrementalType string    `gorm:"type:varchar(20);not null" json:"incremental_type"` // timestamp, id_range, log_based
	LastSyncValue   string    `gorm:"type:text" json:"last_sync_value"`                  // 最后同步的值
	LastSyncTime    time.Time `json:"last_sync_time"`
	CheckpointData  JSONB     `gorm:"type:jsonb" json:"checkpoint_data"`               // 检查点数据
	WatermarkHigh   string    `gorm:"type:text" json:"watermark_high"`                 // 高水位线
	WatermarkLow    string    `gorm:"type:text" json:"watermark_low"`                  // 低水位线
	SyncOffset      int64     `gorm:"default:0" json:"sync_offset"`                    // 同步偏移量
	BatchSize       int       `gorm:"default:1000" json:"batch_size"`                  // 批次大小
	Status          string    `gorm:"type:varchar(20);default:'active'" json:"status"` // active, paused, reset
	LastExecutionID string    `gorm:"type:varchar(50)" json:"last_execution_id"`
	FailureCount    int       `gorm:"default:0" json:"failure_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	DeletedAt       time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (IncrementalState) TableName() string {
	return "incremental_states"
}

// SyncErrorLog 同步错误日志模型
type SyncErrorLog struct {
	ID             string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	SyncConfigID   string     `gorm:"type:varchar(50);not null;index" json:"sync_config_id"`
	ExecutionID    string     `gorm:"type:varchar(50);index" json:"execution_id"`
	ErrorType      string     `gorm:"type:varchar(50);not null" json:"error_type"` // connection, data, validation, system, business
	ErrorCode      string     `gorm:"type:varchar(50)" json:"error_code"`
	ErrorMessage   string     `gorm:"type:text;not null" json:"error_message"`
	ErrorDetail    JSONB      `gorm:"type:jsonb" json:"error_detail"` // 错误详细信息
	DataContext    JSONB      `gorm:"type:jsonb" json:"data_context"` // 出错时的数据上下文
	StackTrace     string     `gorm:"type:text" json:"stack_trace"`
	Severity       string     `gorm:"type:varchar(20);default:'error'" json:"severity"` // info, warning, error, critical
	Status         string     `gorm:"type:varchar(20);default:'new'" json:"status"`     // new, investigating, resolved, ignored
	ResolutionNote string     `gorm:"type:text" json:"resolution_note"`
	AssignedTo     string     `gorm:"type:varchar(50)" json:"assigned_to"`
	ResolvedBy     string     `gorm:"type:varchar(50)" json:"resolved_by"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (SyncErrorLog) TableName() string {
	return "sync_error_logs"
}

// SyncSchedule 同步调度任务模型
type SyncSchedule struct {
	ID                string     `gorm:"type:varchar(50);primaryKey" json:"id"`
	SyncConfigID      string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"sync_config_id"`
	ScheduleType      string     `gorm:"type:varchar(20);not null" json:"schedule_type"` // cron, interval, once, manual
	CronExpression    string     `gorm:"type:varchar(100)" json:"cron_expression"`
	IntervalSeconds   int        `json:"interval_seconds"`
	StartTime         *time.Time `json:"start_time,omitempty"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	TimeZone          string     `gorm:"type:varchar(50);default:'UTC'" json:"timezone"`
	IsEnabled         bool       `gorm:"default:true" json:"is_enabled"`
	MaxConcurrency    int        `gorm:"default:1" json:"max_concurrency"`
	TimeWindow        JSONB      `gorm:"type:jsonb" json:"time_window"`  // 执行时间窗口
	RetryPolicy       JSONB      `gorm:"type:jsonb" json:"retry_policy"` // 重试策略
	AlertPolicy       JSONB      `gorm:"type:jsonb" json:"alert_policy"` // 告警策略
	LastExecutionID   string     `gorm:"type:varchar(50)" json:"last_execution_id"`
	LastExecutionTime *time.Time `json:"last_execution_time,omitempty"`
	NextExecutionTime *time.Time `json:"next_execution_time,omitempty"`
	ExecutionCount    int64      `gorm:"default:0" json:"execution_count"`
	SuccessCount      int64      `gorm:"default:0" json:"success_count"`
	FailureCount      int64      `gorm:"default:0" json:"failure_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         time.Time  `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (SyncSchedule) TableName() string {
	return "sync_schedules"
}

// SyncStatistics 同步统计模型
type SyncStatistics struct {
	ID              string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	SyncConfigID    string    `gorm:"type:varchar(50);not null;index" json:"sync_config_id"`
	StatDate        time.Time `gorm:"type:date;not null;index" json:"stat_date"`
	ExecutionCount  int64     `gorm:"default:0" json:"execution_count"`
	SuccessCount    int64     `gorm:"default:0" json:"success_count"`
	FailureCount    int64     `gorm:"default:0" json:"failure_count"`
	TotalRecords    int64     `gorm:"default:0" json:"total_records"`
	TotalDataVolume int64     `gorm:"default:0" json:"total_data_volume"`
	AvgDuration     float64   `gorm:"default:0" json:"avg_duration"`
	MaxDuration     int64     `gorm:"default:0" json:"max_duration"`
	MinDuration     int64     `gorm:"default:0" json:"min_duration"`
	SuccessRate     float64   `gorm:"default:0" json:"success_rate"`
	ThroughputRate  float64   `gorm:"default:0" json:"throughput_rate"` // 吞吐率：记录数/秒
	ErrorRate       float64   `gorm:"default:0" json:"error_rate"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	DeletedAt       time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (SyncStatistics) TableName() string {
	return "sync_statistics"
}
