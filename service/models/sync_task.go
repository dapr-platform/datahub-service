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

// SyncTask 通用同步任务模型
type SyncTask struct {
	ID            string                 `json:"id" gorm:"primaryKey;type:varchar(36)" example:"550e8400-e29b-41d4-a716-446655440000"`
	LibraryType   string                 `json:"library_type" gorm:"not null;size:20;index" example:"basic_library"`                               // basic_library, thematic_library
	LibraryID     string                 `json:"library_id" gorm:"not null;type:varchar(36);index" example:"550e8400-e29b-41d4-a716-446655440000"` // 基础库ID或主题库ID
	DataSourceID  string                 `json:"data_source_id" gorm:"not null;type:varchar(36);index" example:"550e8400-e29b-41d4-a716-446655440000"`
	InterfaceID   *string                `json:"interface_id,omitempty" gorm:"type:varchar(36);index" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskType      string                 `json:"task_type" gorm:"not null;size:20" example:"full_sync"`              // full_sync, incremental_sync, realtime_sync
	Status        string                 `json:"status" gorm:"not null;size:20;default:'pending'" example:"pending"` // pending, running, success, failed, cancelled
	StartTime     *time.Time             `json:"start_time,omitempty"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Progress      int                    `json:"progress" gorm:"default:0" example:"0"` // 进度百分比 0-100
	ProcessedRows int64                  `json:"processed_rows" gorm:"default:0" example:"0"`
	TotalRows     int64                  `json:"total_rows" gorm:"default:0" example:"0"`
	ErrorCount    int                    `json:"error_count" gorm:"default:0" example:"0"`
	ErrorMessage  string                 `json:"error_message,omitempty" gorm:"type:text"`
	Config        map[string]interface{} `json:"config,omitempty" gorm:"type:jsonb"` // 同步配置
	Result        map[string]interface{} `json:"result,omitempty" gorm:"type:jsonb"` // 同步结果
	CreatedAt     time.Time              `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy     string                 `json:"created_by" gorm:"not null;default:'system';size:100" example:"system"`
	UpdatedAt     time.Time              `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 动态关联 - 这些字段不存储在数据库中，在运行时根据LibraryType动态加载
	BasicLibrary    *BasicLibrary    `json:"basic_library,omitempty" gorm:"-"`
	ThematicLibrary *ThematicLibrary `json:"thematic_library,omitempty" gorm:"-"`
	DataSource      DataSource       `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`
	DataInterface   *DataInterface   `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
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
	// 验证库类型
	if err := st.ValidateLibraryType(); err != nil {
		return err
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
