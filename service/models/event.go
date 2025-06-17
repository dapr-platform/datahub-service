/*
 * @module service/models/event
 * @description 事件管理相关模型定义，包括SSE事件、数据库事件监听等
 * @architecture 事件驱动架构 - 数据模型层
 * @documentReference ai_docs/patch_db_event.md
 * @stateFlow 事件生产 -> 事件分发 -> 事件消费
 * @rules 确保事件的可靠传递和处理
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs ai_docs/requirements.md
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SSEEvent SSE事件模型
type SSEEvent struct {
	ID        string                 `gorm:"type:uuid;primary_key" json:"id"`
	EventType string                 `gorm:"not null" json:"event_type"` // data_change, system_notification, user_message等
	UserName  string                 `gorm:"not null;index" json:"user_name"`
	Data      map[string]interface{} `gorm:"type:jsonb;not null" json:"data"`
	CreatedAt time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy string                 `gorm:"not null;default:'system'" json:"created_by"`
	Sent      bool                   `gorm:"not null;default:false" json:"sent"`
	SentAt    *time.Time             `json:"sent_at"`
	SentBy    string                 `gorm:"not null;default:'system'" json:"sent_by"`
	Read      bool                   `gorm:"not null;default:false" json:"read"`
	ReadAt    *time.Time             `json:"read_at"`
	ReadBy    string                 `gorm:"not null;default:'system'" json:"read_by"`
}

// BeforeCreate 创建前钩子
func (s *SSEEvent) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.CreatedBy == "" {
		s.CreatedBy = "system"
	}
	return nil
}

type EventListener interface {
	RegisterDBEventProcessor(processor DBEventProcessor) error
}

// DBEventListener 数据库事件监听配置模型
type DBEventListener struct {
	ID          string                 `gorm:"type:uuid;primary_key" json:"id"`
	Name        string                 `gorm:"not null;unique" json:"name"`
	TableName   string                 `gorm:"not null" json:"table_name"`
	EventTypes  []string               `gorm:"type:jsonb;not null" json:"event_types"` // INSERT, UPDATE, DELETE
	Condition   map[string]interface{} `gorm:"type:jsonb" json:"condition"`            // 触发条件
	TargetUsers []string               `gorm:"type:jsonb" json:"target_users"`         // 目标用户列表，空表示广播
	IsEnabled   bool                   `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt   time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy   string                 `gorm:"not null;default:'system'" json:"created_by"`
	UpdatedAt   time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy   string                 `gorm:"not null;default:'system'" json:"updated_by"`
}
type DBEventProcessor interface {
	ProcessDBChangeEvent(changeData map[string]interface{}) error
	TableName() string
}

// BeforeCreate 创建前钩子
func (d *DBEventListener) BeforeCreate(tx *gorm.DB) error {
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
func (d *DBEventListener) BeforeUpdate(tx *gorm.DB) error {
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
	}
	return nil
}

// DBChangeEvent 数据库变更事件模型
type DBChangeEvent struct {
	ID           string                 `gorm:"type:uuid;primary_key" json:"id"`
	ListenerID   string                 `gorm:"not null;index" json:"listener_id"`
	Listener     *DBEventListener       `gorm:"foreignKey:ListenerID" json:"listener,omitempty"`
	TableName    string                 `gorm:"not null" json:"table_name"`
	EventType    string                 `gorm:"not null" json:"event_type"` // INSERT, UPDATE, DELETE
	RecordID     string                 `json:"record_id"`
	OldData      map[string]interface{} `gorm:"type:jsonb" json:"old_data"`
	NewData      map[string]interface{} `gorm:"type:jsonb" json:"new_data"`
	ChangedAt    time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"changed_at"`
	CreatedAt    time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy    string                 `gorm:"not null;default:'system'" json:"created_by"`
	Processed    bool                   `gorm:"not null;default:false" json:"processed"`
	ProcessedAt  *time.Time             `json:"processed_at"`
	ProcessedBy  string                 `json:"processed_by"`
	ErrorMessage *string                `json:"error_message"`
}

// BeforeCreate 创建前钩子
func (d *DBChangeEvent) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedBy == "" {
		d.CreatedBy = "system"
	}
	return nil
}

// SSEConnection SSE连接管理模型
type SSEConnection struct {
	ID           string    `gorm:"type:uuid;primary_key" json:"id"`
	UserName     string    `gorm:"not null;index" json:"user_name"`
	ConnectionID string    `gorm:"not null;unique" json:"connection_id"`
	ClientIP     string    `gorm:"not null" json:"client_ip"`
	UserAgent    string    `json:"user_agent"`
	ConnectedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"connected_at"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy    string    `gorm:"not null;default:'system'" json:"created_by"`
	LastPingAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"last_ping_at"`
	IsActive     bool      `gorm:"not null;default:true" json:"is_active"`
}

// BeforeCreate 创建前钩子
func (s *SSEConnection) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.CreatedBy == "" {
		s.CreatedBy = "system"
	}
	return nil
}
