/*
 * @module service/models/sharing
 * @description 数据共享服务相关模型定义，包括API管理、数据订阅、数据使用申请等
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/model.md
 * @stateFlow 数据共享服务生命周期管理
 * @rules 确保数据安全共享和访问控制
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs ai_docs/requirements.md
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApiApplication API接入应用模型
type ApiApplication struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	Name          string    `gorm:"not null;unique" json:"name"`
	AppKey        string    `gorm:"not null;unique" json:"app_key"`
	AppSecretHash string    `gorm:"not null" json:"app_secret_hash"`
	Description   *string   `json:"description"`
	ContactPerson string    `gorm:"not null" json:"contact_person"`
	ContactEmail  string    `gorm:"not null" json:"contact_email"`
	Status        string    `gorm:"not null;default:'active'" json:"status"` // active/inactive
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// BeforeCreate 创建前钩子
func (a *ApiApplication) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// ApiRateLimit API调用限制模型
type ApiRateLimit struct {
	ID            string          `gorm:"type:uuid;primary_key" json:"id"`
	ApplicationID string          `gorm:"not null" json:"application_id"`
	Application   *ApiApplication `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	ApiPath       string          `gorm:"not null" json:"api_path"`
	TimeWindow    int             `gorm:"not null" json:"time_window"`  // 时间窗口，单位秒
	MaxRequests   int             `gorm:"not null" json:"max_requests"` // 最大请求数
	IsEnabled     bool            `gorm:"not null;default:true" json:"is_enabled"`
}

// BeforeCreate 创建前钩子
func (a *ApiRateLimit) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// DataSubscription 数据订阅模型
type DataSubscription struct {
	ID                 string                 `gorm:"type:uuid;primary_key" json:"id"`
	SubscriberID       string                 `gorm:"not null" json:"subscriber_id"`
	SubscriberType     string                 `gorm:"not null" json:"subscriber_type"` // user/application
	ResourceID         string                 `gorm:"not null" json:"resource_id"`
	ResourceType       string                 `gorm:"not null" json:"resource_type"`       // thematic_interface/basic_interface
	NotificationMethod string                 `gorm:"not null" json:"notification_method"` // webhook/message_queue/email
	NotificationConfig map[string]interface{} `gorm:"type:jsonb;not null" json:"notification_config"`
	FilterCondition    map[string]interface{} `gorm:"type:jsonb" json:"filter_condition"`
	Status             string                 `gorm:"not null;default:'active'" json:"status"` // active/paused/terminated
	CreatedAt          time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// BeforeCreate 创建前钩子
func (d *DataSubscription) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// DataAccessRequest 数据使用申请模型
type DataAccessRequest struct {
	ID               string     `gorm:"type:uuid;primary_key" json:"id"`
	RequesterID      string     `gorm:"not null" json:"requester_id"`
	RequesterName    string     `json:"requester_name"`
	ResourceID       string     `gorm:"not null" json:"resource_id"`
	ResourceType     string     `gorm:"not null" json:"resource_type"` // thematic_library/basic_library/interface
	RequestReason    string     `gorm:"not null" json:"request_reason"`
	AccessPermission string     `gorm:"not null" json:"access_permission"` // read/write
	ValidUntil       *time.Time `json:"valid_until"`
	Status           string     `gorm:"not null;default:'pending'" json:"status"` // pending/approved/rejected/expired
	ApprovalComment  *string    `json:"approval_comment"`
	ApproverID       *string    `json:"approver_id"`
	ApproverName     *string    `json:"approver_name"`
	RequestedAt      time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"requested_at"`
	ApprovedAt       *time.Time `json:"approved_at"`
}

// BeforeCreate 创建前钩子
func (d *DataAccessRequest) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// DataSyncTask 数据同步任务模型
type DataSyncTask struct {
	ID             string                 `gorm:"type:uuid;primary_key" json:"id"`
	Name           string                 `gorm:"not null" json:"name"`
	SourceType     string                 `gorm:"not null" json:"source_type"` // database/api/file
	SourceConfig   map[string]interface{} `gorm:"type:jsonb;not null" json:"source_config"`
	TargetType     string                 `gorm:"not null" json:"target_type"` // database/api/file
	TargetConfig   map[string]interface{} `gorm:"type:jsonb;not null" json:"target_config"`
	SyncStrategy   string                 `gorm:"not null" json:"sync_strategy"` // full/incremental
	ScheduleConfig map[string]interface{} `gorm:"type:jsonb" json:"schedule_config"`
	TransformRules map[string]interface{} `gorm:"type:jsonb" json:"transform_rules"`
	Status         string                 `gorm:"not null;default:'active'" json:"status"` // active/inactive/error
	LastSyncTime   *time.Time             `json:"last_sync_time"`
	NextSyncTime   *time.Time             `json:"next_sync_time"`
	CreatedBy      string                 `gorm:"not null" json:"created_by"`
	CreatorName    string                 `json:"creator_name"`
	CreatedAt      time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// BeforeCreate 创建前钩子
func (d *DataSyncTask) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// DataSyncLog 数据同步日志模型
type DataSyncLog struct {
	ID           string                 `gorm:"type:uuid;primary_key" json:"id"`
	TaskID       string                 `gorm:"not null" json:"task_id"`
	Task         *DataSyncTask          `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	StartTime    time.Time              `gorm:"not null" json:"start_time"`
	EndTime      *time.Time             `json:"end_time"`
	Status       string                 `gorm:"not null" json:"status"` // running/success/failure
	RecordsTotal int64                  `gorm:"default:0" json:"records_total"`
	RecordsSync  int64                  `gorm:"default:0" json:"records_sync"`
	RecordsError int64                  `gorm:"default:0" json:"records_error"`
	ErrorMessage *string                `json:"error_message"`
	Details      map[string]interface{} `gorm:"type:jsonb" json:"details"`
}

// BeforeCreate 创建前钩子
func (d *DataSyncLog) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// ApiUsageLog API使用日志模型
type ApiUsageLog struct {
	ID            string          `gorm:"type:uuid;primary_key" json:"id"`
	ApplicationID *string         `json:"application_id"`
	Application   *ApiApplication `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	UserID        *string         `json:"user_id"`
	UserName      *string         `json:"user_name"`
	ApiPath       string          `gorm:"not null" json:"api_path"`
	Method        string          `gorm:"not null" json:"method"`
	RequestIP     string          `gorm:"not null" json:"request_ip"`
	UserAgent     *string         `json:"user_agent"`
	RequestTime   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"request_time"`
	ResponseTime  int             `gorm:"not null" json:"response_time"` // 响应时间，毫秒
	StatusCode    int             `gorm:"not null" json:"status_code"`
	RequestSize   int64           `gorm:"default:0" json:"request_size"`
	ResponseSize  int64           `gorm:"default:0" json:"response_size"`
	ErrorMessage  *string         `json:"error_message"`
}

// BeforeCreate 创建前钩子
func (a *ApiUsageLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
