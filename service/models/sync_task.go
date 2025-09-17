/*
 * @module service/models/sync_task
 * @description 通用同步任务模型，支持基础库和主题库的统一管理
 * @architecture DDD领域驱动设计 - 实体模型
 * @documentReference ai_docs/refactor_sync_task.md
 * @stateFlow 任务创建 -> 待执行 -> 执行中 -> 成功/失败/取消
 * @rules 支持多种库类型，确保数据完整性和一致性
 * @dependencies gorm.io/gorm, github.com/google/uuid, service/meta
 * @refs service/basic_library, service/thematic_library
 */

package models

import (
	"datahub-service/service/meta"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncTaskInterface 同步任务接口关联表
type SyncTaskInterface struct {
	ID            string     `json:"id" gorm:"primaryKey;type:varchar(36);uniqueIndex"`
	TaskID        string     `json:"task_id" gorm:"not null;type:varchar(36);index;uniqueIndex:uk_task_interface" example:"550e8400-e29b-41d4-a716-446655440000"`
	InterfaceID   string     `json:"interface_id" gorm:"not null;type:varchar(36);index;uniqueIndex:uk_task_interface" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status        string     `json:"status" gorm:"not null;size:20;default:'pending'" example:"pending"` // pending, running, success, failed, cancelled
	Progress      int        `json:"progress" gorm:"default:0" example:"0"`                              // 进度百分比 0-100
	ProcessedRows int64      `json:"processed_rows" gorm:"default:0" example:"0"`
	TotalRows     int64      `json:"total_rows" gorm:"default:0" example:"0"`
	ErrorCount    int        `json:"error_count" gorm:"default:0" example:"0"`
	ErrorMessage  string     `json:"error_message,omitempty" gorm:"type:text"`
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Config        JSONB      `json:"config,omitempty" gorm:"type:jsonb"` // 接口级别的配置
	Result        JSONB      `json:"result,omitempty" gorm:"type:jsonb"` // 接口级别的结果
	CreatedAt     time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	SyncTask      SyncTask      `json:"sync_task,omitempty" gorm:"foreignKey:TaskID"`
	DataInterface DataInterface `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (sti *SyncTaskInterface) BeforeCreate(tx *gorm.DB) error {
	if sti.ID == "" {
		sti.ID = uuid.New().String()
	}
	return nil
}

// IsCompleted 判断接口同步是否已完成
func (sti *SyncTaskInterface) IsCompleted() bool {
	completedStatuses := map[string]bool{
		"success":   true,
		"failed":    true,
		"cancelled": true,
	}
	return completedStatuses[sti.Status]
}

// IsRunning 判断接口同步是否正在运行
func (sti *SyncTaskInterface) IsRunning() bool {
	return sti.Status == "running"
}

// IsPending 判断接口同步是否待执行
func (sti *SyncTaskInterface) IsPending() bool {
	return sti.Status == "pending"
}

// GetDuration 获取接口同步执行时长
func (sti *SyncTaskInterface) GetDuration() *time.Duration {
	if sti.StartTime != nil && sti.EndTime != nil {
		duration := sti.EndTime.Sub(*sti.StartTime)
		return &duration
	}
	return nil
}

// GetProgressPercent 获取接口同步进度百分比的字符串表示
func (sti *SyncTaskInterface) GetProgressPercent() string {
	if sti.Progress < 0 {
		return "0%"
	}
	if sti.Progress > 100 {
		return "100%"
	}
	return fmt.Sprintf("%d%%", sti.Progress)
}

// SyncTask 通用同步任务模型
type SyncTask struct {
	ID           string `json:"id" gorm:"primaryKey;type:varchar(36)" example:"550e8400-e29b-41d4-a716-446655440000"`
	LibraryType  string `json:"library_type" gorm:"not null;size:20;index" example:"basic_library"`                               // basic_library, thematic_library
	LibraryID    string `json:"library_id" gorm:"not null;type:varchar(36);index" example:"550e8400-e29b-41d4-a716-446655440000"` // 基础库ID或主题库ID
	DataSourceID string `json:"data_source_id" gorm:"not null;type:varchar(36);index" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskType     string `json:"task_type" gorm:"not null;size:20" example:"full_sync"`              // full_sync, incremental_sync, realtime_sync
	Status       string `json:"status" gorm:"not null;size:20;default:'pending'" example:"pending"` // pending, running, success, failed, cancelled

	// 执行时机相关字段
	TriggerType     string     `json:"trigger_type" gorm:"not null;size:20;default:'manual'" example:"manual"` // manual, once, interval, cron
	CronExpression  string     `json:"cron_expression,omitempty" gorm:"size:100" example:"0 0 * * *"`          // Cron表达式
	IntervalSeconds int        `json:"interval_seconds,omitempty" gorm:"default:0" example:"3600"`             // 间隔秒数
	ScheduledTime   *time.Time `json:"scheduled_time,omitempty"`                                               // 计划执行时间
	NextRunTime     *time.Time `json:"next_run_time,omitempty"`                                                // 下次执行时间
	LastRunTime     *time.Time `json:"last_run_time,omitempty"`                                                // 上次执行时间

	// 执行状态相关字段
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Progress      int        `json:"progress" gorm:"default:0" example:"0"` // 进度百分比 0-100
	ProcessedRows int64      `json:"processed_rows" gorm:"default:0" example:"0"`
	TotalRows     int64      `json:"total_rows" gorm:"default:0" example:"0"`
	ErrorCount    int        `json:"error_count" gorm:"default:0" example:"0"`
	ErrorMessage  string     `json:"error_message,omitempty" gorm:"type:text"`

	// 配置和结果
	Config JSONB `json:"config,omitempty" gorm:"type:jsonb"` // 同步配置
	Result JSONB `json:"result,omitempty" gorm:"type:jsonb"` // 同步结果

	// 基础字段
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy string    `json:"created_by" gorm:"not null;default:'system';size:100" example:"system"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 动态关联 - 这些字段不存储在数据库中，在运行时根据LibraryType动态加载
	BasicLibrary    *BasicLibrary    `json:"basic_library,omitempty" gorm:"-"`
	ThematicLibrary *ThematicLibrary `json:"thematic_library,omitempty" gorm:"-"`
	DataSource      DataSource       `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`

	// 多接口关联
	TaskInterfaces []SyncTaskInterface `json:"task_interfaces,omitempty" gorm:"foreignKey:TaskID"`
	DataInterfaces []DataInterface     `json:"data_interfaces,omitempty" gorm:"many2many:sync_task_interfaces;joinForeignKey:task_id;joinReferences:interface_id"`

	// 执行记录关联
	Executions []SyncTaskExecution `json:"executions,omitempty" gorm:"foreignKey:TaskID"`
}

// BeforeCreate GORM钩子，创建前生成UUID并验证
func (st *SyncTask) BeforeCreate(tx *gorm.DB) error {
	if st.ID == "" {
		st.ID = uuid.New().String()
	}
	if st.CreatedBy == "" {
		st.CreatedBy = "system"
	}

	// 验证库类型
	if err := st.ValidateLibraryType(); err != nil {
		return err
	}

	return nil
}

// BeforeUpdate GORM钩子，更新前验证
func (st *SyncTask) BeforeUpdate(tx *gorm.DB) error {
	// 只在库类型字段不为空时验证（避免部分更新时的验证错误）
	if st.LibraryType != "" {
		if err := st.ValidateLibraryType(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateLibraryType 验证库类型
func (st *SyncTask) ValidateLibraryType() error {
	if !meta.IsValidLibraryType(st.LibraryType) {
		return errors.New("无效的库类型: " + st.LibraryType)
	}
	return nil
}

// IsBasicLibrary 判断是否为基础库任务
func (st *SyncTask) IsBasicLibrary() bool {
	return st.LibraryType == meta.LibraryTypeBasic
}

// IsThematicLibrary 判断是否为主题库任务
func (st *SyncTask) IsThematicLibrary() bool {
	return st.LibraryType == meta.LibraryTypeThematic
}

// GetLibraryDisplayName 获取库类型的显示名称
func (st *SyncTask) GetLibraryDisplayName() string {
	return meta.GetLibraryTypeDisplayName(st.LibraryType)
}

// CanDelete 判断任务是否可以删除
func (st *SyncTask) CanDelete() bool {
	deletableStatuses := map[string]bool{
		"success":   true,
		"failed":    true,
		"cancelled": true,
	}
	return deletableStatuses[st.Status]
}

// CanUpdate 判断任务是否可以更新
func (st *SyncTask) CanUpdate() bool {
	return st.Status == "pending"
}

// CanCancel 判断任务是否可以取消
func (st *SyncTask) CanCancel() bool {
	cancellableStatuses := map[string]bool{
		"pending": true,
		"running": true,
	}
	return cancellableStatuses[st.Status]
}

// CanRetry 判断任务是否可以重试
func (st *SyncTask) CanRetry() bool {
	return st.Status == "failed"
}

// GetDuration 获取任务执行时长
func (st *SyncTask) GetDuration() *time.Duration {
	if st.StartTime != nil && st.EndTime != nil {
		duration := st.EndTime.Sub(*st.StartTime)
		return &duration
	}
	return nil
}

// GetProgressPercent 获取进度百分比的字符串表示
func (st *SyncTask) GetProgressPercent() string {
	if st.Progress < 0 {
		return "0%"
	}
	if st.Progress > 100 {
		return "100%"
	}
	return fmt.Sprintf("%d%%", st.Progress)
}

// IsCompleted 判断任务是否已完成（成功或失败）
func (st *SyncTask) IsCompleted() bool {
	completedStatuses := map[string]bool{
		"success":   true,
		"failed":    true,
		"cancelled": true,
	}
	return completedStatuses[st.Status]
}

// IsRunning 判断任务是否正在运行
func (st *SyncTask) IsRunning() bool {
	return st.Status == "running"
}

// IsPending 判断任务是否待执行
func (st *SyncTask) IsPending() bool {
	return st.Status == "pending"
}

// IsScheduled 判断任务是否为调度任务
func (st *SyncTask) IsScheduled() bool {
	return st.TriggerType == "interval" || st.TriggerType == "cron"
}

// IsManual 判断任务是否为手动任务
func (st *SyncTask) IsManual() bool {
	return st.TriggerType == "manual"
}

// ShouldExecuteNow 判断任务是否应该立即执行
func (st *SyncTask) ShouldExecuteNow() bool {
	if st.IsRunning() || st.TriggerType == "manual" {
		return false
	}

	if st.NextRunTime == nil {
		return false
	}

	return time.Now().After(*st.NextRunTime)
}

// CanStart 判断任务是否可以启动
func (st *SyncTask) CanStart() bool {
	startableStatuses := map[string]bool{
		"pending":   true,
		"failed":    true,
		"cancelled": true,
	}
	return startableStatuses[st.Status]
}

// GetInterfaceCount 获取关联的接口数量
func (st *SyncTask) GetInterfaceCount() int {
	return len(st.TaskInterfaces)
}

// GetCompletedInterfaceCount 获取已完成的接口数量
func (st *SyncTask) GetCompletedInterfaceCount() int {
	count := 0
	for _, taskInterface := range st.TaskInterfaces {
		if taskInterface.IsCompleted() {
			count++
		}
	}
	return count
}

// GetRunningInterfaceCount 获取正在运行的接口数量
func (st *SyncTask) GetRunningInterfaceCount() int {
	count := 0
	for _, taskInterface := range st.TaskInterfaces {
		if taskInterface.IsRunning() {
			count++
		}
	}
	return count
}

// GetPendingInterfaceCount 获取待执行的接口数量
func (st *SyncTask) GetPendingInterfaceCount() int {
	count := 0
	for _, taskInterface := range st.TaskInterfaces {
		if taskInterface.IsPending() {
			count++
		}
	}
	return count
}

// GetOverallProgress 获取整体进度百分比
func (st *SyncTask) GetOverallProgress() int {
	if len(st.TaskInterfaces) == 0 {
		return 0
	}

	totalProgress := 0
	for _, taskInterface := range st.TaskInterfaces {
		totalProgress += taskInterface.Progress
	}
	return totalProgress / len(st.TaskInterfaces)
}

// IsAllInterfacesCompleted 判断是否所有接口都已完成
func (st *SyncTask) IsAllInterfacesCompleted() bool {
	if len(st.TaskInterfaces) == 0 {
		return false
	}

	for _, taskInterface := range st.TaskInterfaces {
		if !taskInterface.IsCompleted() {
			return false
		}
	}
	return true
}

// HasRunningInterfaces 判断是否有正在运行的接口
func (st *SyncTask) HasRunningInterfaces() bool {
	for _, taskInterface := range st.TaskInterfaces {
		if taskInterface.IsRunning() {
			return true
		}
	}
	return false
}

// GetInterfaceByID 根据接口ID获取任务接口
func (st *SyncTask) GetInterfaceByID(interfaceID string) *SyncTaskInterface {
	for i, taskInterface := range st.TaskInterfaces {
		if taskInterface.InterfaceID == interfaceID {
			return &st.TaskInterfaces[i]
		}
	}
	return nil
}

// SyncTaskExecution 同步任务执行记录模型
type SyncTaskExecution struct {
	ID            string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TaskID        string     `json:"task_id" gorm:"not null;type:varchar(36);index"`
	ExecutionType string     `json:"execution_type" gorm:"not null;size:20" example:"manual"`            // manual, scheduled, retry
	Status        string     `json:"status" gorm:"not null;size:20;default:'running'" example:"running"` // running, success, failed, cancelled
	StartTime     time.Time  `json:"start_time" gorm:"not null;default:CURRENT_TIMESTAMP"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Duration      int64      `json:"duration" gorm:"default:0"` // 执行时长，毫秒
	ProcessedRows int64      `json:"processed_rows" gorm:"default:0"`
	TotalRows     int64      `json:"total_rows" gorm:"default:0"`
	ErrorCount    int        `json:"error_count" gorm:"default:0"`
	ErrorMessage  string     `json:"error_message,omitempty" gorm:"type:text"`
	Progress      int        `json:"progress" gorm:"default:0"`          // 进度百分比 0-100
	Result        JSONB      `json:"result,omitempty" gorm:"type:jsonb"` // 执行结果详情
	CreatedAt     time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	Task SyncTask `json:"task,omitempty" gorm:"foreignKey:TaskID"`
}

// TableName 指定表名
func (SyncTaskExecution) TableName() string {
	return "sync_task_executions"
}

// BeforeCreate GORM钩子，创建前生成UUID
func (ste *SyncTaskExecution) BeforeCreate(tx *gorm.DB) error {
	if ste.ID == "" {
		ste.ID = uuid.New().String()
	}
	return nil
}

// IsCompleted 判断执行是否已完成
func (ste *SyncTaskExecution) IsCompleted() bool {
	completedStatuses := map[string]bool{
		"success":   true,
		"failed":    true,
		"cancelled": true,
	}
	return completedStatuses[ste.Status]
}

// IsRunning 判断执行是否正在运行
func (ste *SyncTaskExecution) IsRunning() bool {
	return ste.Status == "running"
}

// GetDurationSeconds 获取执行时长（秒）
func (ste *SyncTaskExecution) GetDurationSeconds() float64 {
	if ste.Duration > 0 {
		return float64(ste.Duration) / 1000.0
	}
	if ste.EndTime != nil {
		return ste.EndTime.Sub(ste.StartTime).Seconds()
	}
	return 0
}
