/*
 * @module service/models/thematic_sync
 * @description 主题库数据同步相关模型定义，包括同步任务、执行记录、数据血缘等核心实体
 * @architecture DDD领域驱动设计 - 实体模型
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 同步任务创建 -> 调度执行 -> 数据处理 -> 结果记录 -> 血缘追踪
 * @rules 遵循数据库设计规范，支持复杂的数据同步和血缘追踪
 * @dependencies gorm.io/gorm, time
 * @refs service/models/thematic_library.go, service/models/basic_library.go
 */

package models

import (
	"time"
)

// ThematicSyncTask 主题同步任务模型
type ThematicSyncTask struct {
	ID                  string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ThematicLibraryID   string `json:"thematic_library_id" gorm:"not null;type:varchar(36);index"`
	ThematicInterfaceID string `json:"thematic_interface_id" gorm:"not null;type:varchar(36);index"`
	TaskName            string `json:"task_name" gorm:"not null;size:255"`
	Description         string `json:"description" gorm:"size:1000"`

	// 源数据配置
	SourceLibraries  JSONB `json:"source_libraries" gorm:"type:jsonb"`  // 源基础库列表
	SourceInterfaces JSONB `json:"source_interfaces" gorm:"type:jsonb"` // 源接口列表

	// SQL数据源配置 - 如果配置了SQL，则优先使用SQL获取数据
	DataSourceSQL  JSONB `json:"data_source_sql" gorm:"type:jsonb"`  // SQL数据源配置
	SQLQueryConfig JSONB `json:"sql_query_config" gorm:"type:jsonb"` // SQL查询配置（参数、超时等）

	// 汇聚配置
	AggregationConfig JSONB `json:"aggregation_config" gorm:"type:jsonb"`  // 汇聚配置
	KeyMatchingRules  JSONB `json:"key_matching_rules" gorm:"type:jsonb"`  // 主键匹配规则
	FieldMappingRules JSONB `json:"field_mapping_rules" gorm:"type:jsonb"` // 字段映射规则

	// 处理配置
	CleansingRules JSONB `json:"cleansing_rules" gorm:"type:jsonb"` // 清洗规则
	PrivacyRules   JSONB `json:"privacy_rules" gorm:"type:jsonb"`   // 脱敏规则

	// 调度配置
	TriggerType     string     `json:"trigger_type" gorm:"not null;size:20"` // manual, once, interval, cron
	CronExpression  string     `json:"cron_expression,omitempty" gorm:"size:100"`
	IntervalSeconds int        `json:"interval_seconds,omitempty"`
	ScheduledTime   *time.Time `json:"scheduled_time,omitempty"`
	NextRunTime     *time.Time `json:"next_run_time,omitempty"`

	// 状态信息
	Status          string     `json:"status" gorm:"not null;default:'draft'"` // draft, active, paused, completed, failed
	LastSyncTime    *time.Time `json:"last_sync_time,omitempty"`
	LastSyncStatus  string     `json:"last_sync_status,omitempty"`
	LastSyncMessage string     `json:"last_sync_message,omitempty"`

	// 统计信息
	TotalSyncCount      int64 `json:"total_sync_count" gorm:"default:0"`
	SuccessfulSyncCount int64 `json:"successful_sync_count" gorm:"default:0"`
	FailedSyncCount     int64 `json:"failed_sync_count" gorm:"default:0"`

	// 审计字段
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by" gorm:"size:100"`

	// 关联关系
	ThematicLibrary   *ThematicLibrary        `json:"thematic_library,omitempty" gorm:"foreignKey:ThematicLibraryID"`
	ThematicInterface *ThematicInterface      `json:"thematic_interface,omitempty" gorm:"foreignKey:ThematicInterfaceID"`
	SyncExecutions    []ThematicSyncExecution `json:"sync_executions,omitempty" gorm:"foreignKey:TaskID"`
}

// TableName 指定表名
func (ThematicSyncTask) TableName() string {
	return "thematic_sync_tasks"
}

// ThematicSyncExecution 主题同步执行记录
type ThematicSyncExecution struct {
	ID            string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TaskID        string `json:"task_id" gorm:"not null;type:varchar(36);index"`
	ExecutionType string `json:"execution_type" gorm:"not null;size:20"` // manual, scheduled, retry

	// 执行状态
	Status    string     `json:"status" gorm:"not null;default:'pending'"` // pending, running, success, failed, cancelled
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Duration  int64      `json:"duration" gorm:"default:0"` // 执行时长（秒）

	// 数据统计
	SourceRecordCount    int64 `json:"source_record_count" gorm:"default:0"`    // 源记录数
	ProcessedRecordCount int64 `json:"processed_record_count" gorm:"default:0"` // 已处理记录数
	InsertedRecordCount  int64 `json:"inserted_record_count" gorm:"default:0"`  // 新增记录数
	UpdatedRecordCount   int64 `json:"updated_record_count" gorm:"default:0"`   // 更新记录数
	DeletedRecordCount   int64 `json:"deleted_record_count" gorm:"default:0"`   // 删除记录数
	ErrorRecordCount     int64 `json:"error_record_count" gorm:"default:0"`     // 错误记录数

	// 处理结果
	ProcessingResult JSONB `json:"processing_result" gorm:"type:jsonb"` // 处理结果详情
	ErrorDetails     JSONB `json:"error_details" gorm:"type:jsonb"`     // 错误详情
	QualityReport    JSONB `json:"quality_report" gorm:"type:jsonb"`    // 质量报告

	// 审计字段
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by" gorm:"size:100"`

	// 关联关系
	Task *ThematicSyncTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
}

// TableName 指定表名
func (ThematicSyncExecution) TableName() string {
	return "thematic_sync_executions"
}

// ThematicDataLineage 主题数据血缘模型
type ThematicDataLineage struct {
	ID                  string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ThematicInterfaceID string `json:"thematic_interface_id" gorm:"not null;type:varchar(36);index"`
	ThematicRecordID    string `json:"thematic_record_id" gorm:"not null;size:255;index"` // 主题记录ID

	// 源数据信息
	SourceLibraryID   string `json:"source_library_id" gorm:"not null;type:varchar(36)"`
	SourceInterfaceID string `json:"source_interface_id" gorm:"not null;type:varchar(36)"`
	SourceRecordID    string `json:"source_record_id" gorm:"not null;size:255"`
	SourceRecordHash  string `json:"source_record_hash" gorm:"size:64"` // 源记录哈希值

	// 处理信息
	ProcessingRules       JSONB `json:"processing_rules" gorm:"type:jsonb"`       // 应用的处理规则
	TransformationDetails JSONB `json:"transformation_details" gorm:"type:jsonb"` // 转换详情

	// 质量信息
	QualityScore  float64 `json:"quality_score" gorm:"default:0"`   // 质量评分
	QualityIssues JSONB   `json:"quality_issues" gorm:"type:jsonb"` // 质量问题

	// 时间信息
	SourceDataTime time.Time `json:"source_data_time"` // 源数据时间
	ProcessedTime  time.Time `json:"processed_time"`   // 处理时间
	CreatedAt      time.Time `json:"created_at"`

	// 关联关系
	ThematicInterface *ThematicInterface `json:"thematic_interface,omitempty" gorm:"foreignKey:ThematicInterfaceID"`
	SourceLibrary     *BasicLibrary      `json:"source_library,omitempty" gorm:"foreignKey:SourceLibraryID"`
	SourceInterface   *DataInterface     `json:"source_interface,omitempty" gorm:"foreignKey:SourceInterfaceID"`
}

// TableName 指定表名
func (ThematicDataLineage) TableName() string {
	return "thematic_data_lineages"
}

// ShouldExecuteNow 判断任务是否应该立即执行
func (t *ThematicSyncTask) ShouldExecuteNow() bool {
	if t.Status != "active" {
		return false
	}

	if t.NextRunTime == nil {
		return false
	}

	return time.Now().After(*t.NextRunTime)
}

// CanStart 判断任务是否可以开始执行
func (t *ThematicSyncTask) CanStart() bool {
	return t.Status == "active" || t.Status == "draft"
}

// UpdateNextRunTime 更新下次执行时间
func (t *ThematicSyncTask) UpdateNextRunTime() {
	now := time.Now()

	switch t.TriggerType {
	case "once":
		// 一次性任务执行后不再执行
		t.NextRunTime = nil
	case "interval":
		// 按间隔执行
		if t.IntervalSeconds > 0 {
			nextTime := now.Add(time.Duration(t.IntervalSeconds) * time.Second)
			t.NextRunTime = &nextTime
		}
	case "cron":
		// Cron表达式执行（这里简化处理，实际需要使用cron库）
		if t.CronExpression != "" {
			// 实际实现中需要解析cron表达式计算下次执行时间
			nextTime := now.Add(1 * time.Hour) // 临时实现
			t.NextRunTime = &nextTime
		}
	}
}

// GetDuration 获取执行时长
func (e *ThematicSyncExecution) GetDuration() time.Duration {
	if e.StartTime != nil && e.EndTime != nil {
		return e.EndTime.Sub(*e.StartTime)
	}
	return 0
}

// IsCompleted 判断执行是否完成
func (e *ThematicSyncExecution) IsCompleted() bool {
	return e.Status == "success" || e.Status == "failed" || e.Status == "cancelled"
}

// IsSuccess 判断执行是否成功
func (e *ThematicSyncExecution) IsSuccess() bool {
	return e.Status == "success"
}

// GetProcessingRate 获取处理成功率
func (e *ThematicSyncExecution) GetProcessingRate() float64 {
	if e.SourceRecordCount == 0 {
		return 0
	}
	successCount := e.InsertedRecordCount + e.UpdatedRecordCount
	return float64(successCount) / float64(e.SourceRecordCount) * 100
}
