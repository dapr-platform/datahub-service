/*
 * @module service/models/sync_engine_models
 * @description 数据同步引擎相关模型定义
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 数据操作 -> 业务逻辑
 * @rules 确保数据模型的一致性和完整性
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs service/sync_engine
 */

package models

import (
	"context"
	"datahub-service/service/meta"
	"time"
)

// TaskStatus 任务状态枚举 - 使用meta中的定义
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = meta.SyncTaskStatusPending
	TaskStatusRunning   TaskStatus = meta.SyncTaskStatusRunning
	TaskStatusSuccess   TaskStatus = meta.SyncTaskStatusSuccess
	TaskStatusFailed    TaskStatus = meta.SyncTaskStatusFailed
	TaskStatusCancelled TaskStatus = meta.SyncTaskStatusCancelled
	TaskStatusPaused    TaskStatus = meta.SyncTaskStatusPaused
)

// SyncType 同步类型枚举 - 使用meta中的定义
type SyncType string

const (
	SyncTypeFull        SyncType = meta.SyncTaskTypeFullSync
	SyncTypeIncremental SyncType = meta.SyncTaskTypeIncrementalSync
	SyncTypeRealtime    SyncType = meta.SyncTaskTypeRealtimeSync
)

// SyncTaskContext 同步任务上下文
type SyncTaskContext struct {
	Task      *SyncTask
	Context   context.Context
	Cancel    context.CancelFunc
	StartTime time.Time
	Processor SyncProcessor
	Status    TaskStatus
	Progress  *SyncProgress
	Error     error
	Result    *SyncResult
}

// SyncProcessor 同步处理器接口
type SyncProcessor interface {
	Process(ctx context.Context, task *SyncTask, progress *SyncProgress) (*SyncResult, error)
	GetProcessorType() string
	Validate(task *SyncTask) error
	EstimateProgress(task *SyncTask) (*ProgressEstimate, error)
}

// SyncProgress 同步进度
type SyncProgress struct {
	ProcessedRows   int64     `json:"processed_rows"`
	TotalRows       int64     `json:"total_rows"`
	ErrorCount      int       `json:"error_count"`
	ProgressPercent int       `json:"progress_percent"`
	CurrentPhase    string    `json:"current_phase"`
	EstimatedTime   time.Time `json:"estimated_time"`
	Speed           int64     `json:"speed"` // 每秒处理行数
	UpdatedAt       time.Time `json:"updated_at"`
}

// SyncResult 同步结果
type SyncResult struct {
	TaskID        string                 `json:"task_id"`
	Status        TaskStatus             `json:"status"`
	ProcessedRows int64                  `json:"processed_rows"`
	SuccessRows   int64                  `json:"success_rows"`
	ErrorRows     int64                  `json:"error_rows"`
	Duration      time.Duration          `json:"-"`           // 不用于API序列化
	DurationMs    int64                  `json:"duration_ms"` // 毫秒数，便于JSON序列化
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Statistics    map[string]interface{} `json:"statistics,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SyncEvent 同步事件
type SyncEvent struct {
	TaskID    string                 `json:"task_id"`
	EventType string                 `json:"event_type"` // start, progress, complete, error
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// SyncTaskRequest 同步任务请求
type SyncTaskRequest struct {
	DataSourceID string                 `json:"data_source_id"`
	InterfaceID  string                 `json:"interface_id,omitempty"`
	SyncType     SyncType               `json:"sync_type"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Priority     int                    `json:"priority"`
	ScheduledBy  string                 `json:"scheduled_by"`
	Callback     func(*SyncResult)      `json:"-"`
}

// ProgressEstimate 进度预估
type ProgressEstimate struct {
	EstimatedRows     int64                  `json:"estimated_rows"`
	EstimatedTime     time.Duration          `json:"estimated_time"`
	Complexity        string                 `json:"complexity"` // low, medium, high
	RequiredResources map[string]interface{} `json:"required_resources"`
}
