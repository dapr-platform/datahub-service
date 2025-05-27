/*
 * @module service/models/basic_library
 * @description 数据基础库相关模型定义，包括基础库、接口、数据源等核心实体
 * @architecture DDD领域驱动设计 - 实体模型
 * @documentReference dev_docs/model.md
 * @stateFlow 数据基础库生命周期管理
 * @rules 遵循数据库设计规范，确保数据完整性和一致性
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs dev_docs/requirements.md
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BasicLibrary 数据基础库模型
type BasicLibrary struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)" example:"550e8400-e29b-41d4-a716-446655440000"`
	NameZh      string    `json:"name_zh" gorm:"not null;size:255" example:"用户数据基础库"`
	NameEn      string    `json:"name_en" gorm:"not null;unique;size:255" example:"user_basic_library"`
	Description string    `json:"description" gorm:"size:1000" example:"存储用户基础信息的数据库"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	Status      string    `json:"status" gorm:"not null;default:'active';size:20" example:"active"`

	// 关联关系
	Interfaces []DataInterface `json:"interfaces,omitempty" gorm:"foreignKey:LibraryID"`
}

// DataInterface 数据接口模型
type DataInterface struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LibraryID   string    `json:"library_id" gorm:"not null;type:varchar(36);index"`
	NameZh      string    `json:"name_zh" gorm:"not null;size:255"`
	NameEn      string    `json:"name_en" gorm:"not null;size:255"`
	Type        string    `json:"type" gorm:"not null;size:20"` // realtime, batch
	Description string    `json:"description" gorm:"size:1000"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	Status      string    `json:"status" gorm:"not null;default:'active';size:20"`

	// 关联关系
	BasicLibrary BasicLibrary     `json:"basic_library,omitempty" gorm:"foreignKey:LibraryID"`
	DataSource   *DataSource      `json:"data_source,omitempty" gorm:"foreignKey:InterfaceID"`
	Fields       []InterfaceField `json:"fields,omitempty" gorm:"foreignKey:InterfaceID"`
	CleanRules   []CleansingRule  `json:"clean_rules,omitempty" gorm:"foreignKey:InterfaceID"`
}

// DataSource 数据源模型
type DataSource struct {
	ID               string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InterfaceID      string                 `json:"interface_id" gorm:"not null;type:varchar(36);index"`
	Type             string                 `json:"type" gorm:"not null;size:50"` // kafka, redis, nats, http, db, hostpath
	ConnectionConfig map[string]interface{} `json:"connection_config" gorm:"type:jsonb;not null"`
	ParamsConfig     map[string]interface{} `json:"params_config" gorm:"type:jsonb"`
	CreatedAt        time.Time              `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time              `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	DataInterface DataInterface `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
}

// InterfaceField 接口字段模型
type InterfaceField struct {
	ID           string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InterfaceID  string `json:"interface_id" gorm:"not null;type:varchar(36);index"`
	NameZh       string `json:"name_zh" gorm:"not null;size:255"`
	NameEn       string `json:"name_en" gorm:"not null;size:255"`
	DataType     string `json:"data_type" gorm:"not null;size:50"`
	IsPrimaryKey bool   `json:"is_primary_key" gorm:"not null;default:false"`
	IsNullable   bool   `json:"is_nullable" gorm:"not null;default:true"`
	DefaultValue string `json:"default_value" gorm:"size:255"`
	Description  string `json:"description" gorm:"size:1000"`
	OrderNum     int    `json:"order_num" gorm:"not null"`

	// 关联关系
	DataInterface DataInterface `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
}

// CleansingRule 数据清洗规则模型
type CleansingRule struct {
	ID          string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InterfaceID string                 `json:"interface_id" gorm:"not null;type:varchar(36);index"`
	Type        string                 `json:"type" gorm:"not null;size:50"`
	Config      map[string]interface{} `json:"config" gorm:"type:jsonb;not null"`
	OrderNum    int                    `json:"order_num" gorm:"not null"`
	IsEnabled   bool                   `json:"is_enabled" gorm:"not null;default:true"`

	// 关联关系
	DataInterface DataInterface `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (bl *BasicLibrary) BeforeCreate(tx *gorm.DB) error {
	if bl.ID == "" {
		bl.ID = uuid.New().String()
	}
	return nil
}

func (di *DataInterface) BeforeCreate(tx *gorm.DB) error {
	if di.ID == "" {
		di.ID = uuid.New().String()
	}
	return nil
}

func (ds *DataSource) BeforeCreate(tx *gorm.DB) error {
	if ds.ID == "" {
		ds.ID = uuid.New().String()
	}
	return nil
}

func (if_ *InterfaceField) BeforeCreate(tx *gorm.DB) error {
	if if_.ID == "" {
		if_.ID = uuid.New().String()
	}
	return nil
}

func (cr *CleansingRule) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == "" {
		cr.ID = uuid.New().String()
	}
	return nil
}
