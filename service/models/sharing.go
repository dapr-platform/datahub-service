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
	ID                string    `gorm:"type:uuid;primary_key" json:"id"`
	Name              string    `gorm:"not null;unique" json:"name"`
	Path              string    `gorm:"not null;unique" json:"path"` // 应用访问路径，例如 "user-center"
	ThematicLibraryID string    `gorm:"not null" json:"thematic_library_id"`
	Description       *string   `json:"description"`
	ContactPerson     string    `gorm:"not null" json:"contact_person"`
	ContactPhone      string    `gorm:"not null" json:"contact_phone"`
	Status            string    `gorm:"not null;default:'active'" json:"status"` // active/inactive
	CreatedAt         time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy         string    `gorm:"not null;default:'system';size:100" json:"created_by"`
	UpdatedAt         time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy         string    `gorm:"not null;default:'system';size:100" json:"updated_by"`

	// 关联关系
	ThematicLibrary ThematicLibrary `gorm:"foreignKey:ThematicLibraryID" json:"thematic_library,omitempty"`
	ApiKeys         []ApiKey        `gorm:"many2many:api_key_applications;" json:"api_keys,omitempty"`
	ApiInterfaces   []ApiInterface  `gorm:"foreignKey:ApiApplicationID" json:"api_interfaces,omitempty"`
}

// BeforeCreate 创建前钩子
func (a *ApiApplication) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedBy == "" {
		a.CreatedBy = "system"
	}
	if a.UpdatedBy == "" {
		a.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (a *ApiApplication) BeforeUpdate(tx *gorm.DB) error {
	if a.UpdatedBy == "" {
		a.UpdatedBy = "system"
	}
	return nil
}

// ApiKey API密钥模型 - 一个Key可以访问多个应用
type ApiKey struct {
	ID           string     `gorm:"type:uuid;primary_key" json:"id"`
	Name         string     `gorm:"not null" json:"name"`              // ApiKey名称
	KeyPrefix    string     `gorm:"not null;size:8" json:"key_prefix"` // Key的前缀，用于快速识别
	KeyValueHash string     `gorm:"not null;unique" json:"-"`          // 存储Hash后的Key值
	Description  string     `json:"description"`
	Status       string     `gorm:"not null;default:'active'" json:"status"` // active, inactive, revoked
	ExpiresAt    *time.Time `json:"expires_at"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	UsageCount   int64      `gorm:"default:0" json:"usage_count"`
	CreatedBy    string     `gorm:"size:100" json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	UpdatedBy    string     `gorm:"size:100" json:"updated_by"`

	// 多对多关系：一个ApiKey可以访问多个ApiApplication
	Applications []ApiApplication `gorm:"many2many:api_key_applications;" json:"applications,omitempty"`
}

// BeforeCreate 创建前钩子
func (ak *ApiKey) BeforeCreate(tx *gorm.DB) error {
	if ak.ID == "" {
		ak.ID = uuid.New().String()
	}
	if ak.CreatedBy == "" {
		ak.CreatedBy = "system"
	}
	if ak.UpdatedBy == "" {
		ak.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (ak *ApiKey) BeforeUpdate(tx *gorm.DB) error {
	if ak.UpdatedBy == "" {
		ak.UpdatedBy = "system"
	}
	return nil
}

// ApiKeyApplication ApiKey和ApiApplication的多对多关联表
type ApiKeyApplication struct {
	ApiKeyID         string    `gorm:"type:uuid;primary_key" json:"api_key_id"`
	ApiApplicationID string    `gorm:"type:uuid;primary_key" json:"api_application_id"`
	CreatedAt        time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy        string    `gorm:"not null;default:'system';size:100" json:"created_by"`

	// 关联关系
	ApiKey         ApiKey         `gorm:"foreignKey:ApiKeyID" json:"api_key,omitempty"`
	ApiApplication ApiApplication `gorm:"foreignKey:ApiApplicationID" json:"api_application,omitempty"`
}

// BeforeCreate 创建前钩子
func (aka *ApiKeyApplication) BeforeCreate(tx *gorm.DB) error {
	if aka.CreatedBy == "" {
		aka.CreatedBy = "system"
	}
	return nil
}

// ApiInterface API接口模型
type ApiInterface struct {
	ID                  string            `gorm:"type:uuid;primary_key" json:"id"`
	ApiApplicationID    string            `gorm:"not null;index" json:"api_application_id"`
	ThematicInterfaceID string            `gorm:"not null;index" json:"thematic_interface_id"`
	Path                string            `gorm:"not null;unique" json:"path"` // 对外暴露的路径，例如 "users"
	Description         string            `json:"description"`
	Status              string            `gorm:"not null;default:'active'" json:"status"` // active, inactive
	CreatedAt           time.Time         `json:"created_at"`
	CreatedBy           string            `gorm:"size:100" json:"created_by"`
	ApiApplication      ApiApplication    `gorm:"foreignKey:ApiApplicationID" json:"api_application,omitempty"`
	ThematicInterface   ThematicInterface `gorm:"foreignKey:ThematicInterfaceID" json:"thematic_interface,omitempty"`
}

// BeforeCreate 创建前钩子
func (ai *ApiInterface) BeforeCreate(tx *gorm.DB) error {
	if ai.ID == "" {
		ai.ID = uuid.New().String()
	}
	if ai.CreatedBy == "" {
		ai.CreatedBy = "system"
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
	CreatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy     string          `gorm:"not null;default:'system';size:100" json:"created_by"`
}

// BeforeCreate 创建前钩子
func (a *ApiRateLimit) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedBy == "" {
		a.CreatedBy = "system"
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
	CreatedBy          string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
	UpdatedAt          time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy          string                 `gorm:"not null;default:'system';size:100" json:"updated_by"`
}

// BeforeCreate 创建前钩子
func (d *DataSubscription) BeforeCreate(tx *gorm.DB) error {
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
func (d *DataSubscription) BeforeUpdate(tx *gorm.DB) error {
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
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
	CreatedBy        string     `gorm:"not null;default:'system';size:100" json:"created_by"`
	ApprovedAt       *time.Time `json:"approved_at"`
}

// BeforeCreate 创建前钩子
func (d *DataAccessRequest) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedBy == "" {
		d.CreatedBy = "system"
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
	CreatedBy     string          `gorm:"not null;default:'system';size:100" json:"created_by"`
}

// BeforeCreate 创建前钩子
func (a *ApiUsageLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedBy == "" {
		a.CreatedBy = "system"
	}
	return nil
}
