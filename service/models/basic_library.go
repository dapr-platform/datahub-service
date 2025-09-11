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
	ID          string        `json:"id" gorm:"primaryKey;type:varchar(36)" example:"550e8400-e29b-41d4-a716-446655440000"`
	NameZh      string        `json:"name_zh" gorm:"not null;size:255" example:"用户数据基础库"`
	NameEn      string        `json:"name_en" gorm:"not null;unique;size:255" example:"user_basic_library"`
	Description string        `json:"description" gorm:"size:1000" example:"存储用户基础信息的数据库"`
	CreatedAt   time.Time     `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy   string        `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt   time.Time     `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy   string        `json:"updated_by" gorm:"not null;default:'system';size:100"`
	Status      string        `json:"status" gorm:"not null;default:'active';size:20" example:"active"`
	DataSources []*DataSource `json:"data_sources,omitempty" gorm:"foreignKey:LibraryID"`
	// 关联关系
	Interfaces []DataInterface `json:"interfaces,omitempty" gorm:"foreignKey:LibraryID"`
}

// DataInterface 数据接口模型
type DataInterface struct {
	ID                string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LibraryID         string    `json:"library_id" gorm:"not null;type:varchar(36);index"`
	NameZh            string    `json:"name_zh" gorm:"not null;size:255"`
	NameEn            string    `json:"name_en" gorm:"not null;size:255"`
	Type              string    `json:"type" gorm:"not null;size:20"` // realtime, batch
	Description       string    `json:"description" gorm:"size:1000"`
	CreatedAt         time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy         string    `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy         string    `json:"updated_by" gorm:"not null;default:'system';size:100"`
	Status            string    `json:"status" gorm:"not null;default:'active';size:20"`
	IsTableCreated    bool      `json:"is_table_created" gorm:"not null;default:false"`
	DataSourceID      string    `json:"data_source_id" gorm:"type:varchar(36)"`
	InterfaceConfig   JSONB     `json:"interface_config" gorm:"type:jsonb"`
	ParseConfig       JSONB     `json:"parse_config" gorm:"type:jsonb"`
	TableFieldsConfig JSONB     `json:"table_fields_config" gorm:"type:jsonb"`
	// 关联关系
	BasicLibrary BasicLibrary     `json:"basic_library,omitempty" gorm:"foreignKey:LibraryID"`
	DataSource   DataSource       `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`
	Fields       []InterfaceField `json:"fields,omitempty" gorm:"foreignKey:InterfaceID"`
	CleanRules   []CleansingRule  `json:"clean_rules,omitempty" gorm:"foreignKey:InterfaceID"`
}

// DataSource 数据源模型
type DataSource struct {
	ID               string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LibraryID        string    `json:"library_id" gorm:"not null;type:varchar(36);index"`
	Name             string    `json:"name" gorm:"not null;size:255;default:''"`
	Category         string    `json:"category" gorm:"not null;size:50"` // stream, http, db, file
	Type             string    `json:"type" gorm:"not null;size:50;default:''"`
	Status           string    `json:"status" gorm:"not null;default:'active';size:20"` // active, inactive
	ConnectionConfig JSONB     `json:"connection_config" gorm:"type:jsonb;not null"`
	ParamsConfig     JSONB     `json:"params_config" gorm:"type:jsonb"`
	Script           string    `json:"script" gorm:"type:text"`                      // 动态执行脚本，用于特殊认证处理
	ScriptEnabled    bool      `json:"script_enabled" gorm:"not null;default:false"` // 是否启用脚本执行
	CreatedAt        time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy        string    `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy        string    `json:"updated_by" gorm:"not null;default:'system';size:100"`
	// 关联关系
	BasicLibrary BasicLibrary `json:"basic_library,omitempty" gorm:"foreignKey:LibraryID"`
}

// InterfaceField 接口字段模型
type InterfaceField struct {
	ID               string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InterfaceID      string    `json:"interface_id" gorm:"not null;type:varchar(36);index"`
	NameZh           string    `json:"name_zh" gorm:"not null;size:255"`
	NameEn           string    `json:"name_en" gorm:"not null;size:255"`
	DataType         string    `json:"data_type" gorm:"not null;size:50"`
	IsPrimaryKey     bool      `json:"is_primary_key" gorm:"not null;default:false"`
	IsUnique         bool      `json:"is_unique" gorm:"not null;default:false"`
	IsNullable       bool      `json:"is_nullable" gorm:"not null;default:true"`
	DefaultValue     string    `json:"default_value" gorm:"size:255"`
	Description      string    `json:"description" gorm:"size:1000"`
	OrderNum         int       `json:"order_num" gorm:"not null"`
	CheckConstraint  string    `json:"check_constraint" gorm:"size:255"`
	IsIncrementField bool      `json:"is_increment_field" gorm:"not null;default:false"` // 是否为增量字段，增量更新时根据这个字段判断条件
	CreatedAt        time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy        string    `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy        string    `json:"updated_by" gorm:"not null;default:'system';size:100"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (if_ *InterfaceField) BeforeCreate(tx *gorm.DB) error {
	if if_.ID == "" {
		if_.ID = uuid.New().String()
	}
	if if_.CreatedBy == "" {
		if_.CreatedBy = "system"
	}
	return nil
}

// CleansingRule 数据清洗规则模型
type CleansingRule struct {
	ID          string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InterfaceID string                 `json:"interface_id" gorm:"not null;type:varchar(36);index"`
	Type        string                 `json:"type" gorm:"not null;size:50"`
	Config      map[string]interface{} `json:"config" gorm:"type:jsonb;not null"`
	OrderNum    int                    `json:"order_num" gorm:"not null"`
	IsEnabled   bool                   `json:"is_enabled" gorm:"not null;default:true"`
	CreatedAt   time.Time              `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy   string                 `json:"created_by" gorm:"not null;default:'system';size:100"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (bl *BasicLibrary) BeforeCreate(tx *gorm.DB) error {
	if bl.ID == "" {
		bl.ID = uuid.New().String()
	}
	if bl.CreatedBy == "" {
		bl.CreatedBy = "system"
	}
	if bl.UpdatedBy == "" {
		bl.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate GORM钩子，更新前设置更新者
func (bl *BasicLibrary) BeforeUpdate(tx *gorm.DB) error {
	if bl.UpdatedBy == "" {
		bl.UpdatedBy = "system"
	}
	return nil
}

func (di *DataInterface) BeforeCreate(tx *gorm.DB) error {
	if di.ID == "" {
		di.ID = uuid.New().String()
	}
	if di.CreatedBy == "" {
		di.CreatedBy = "system"
	}
	if di.UpdatedBy == "" {
		di.UpdatedBy = "system"
	}
	return nil
}

func (di *DataInterface) BeforeUpdate(tx *gorm.DB) error {
	if di.UpdatedBy == "" {
		di.UpdatedBy = "system"
	}
	return nil
}

func (ds *DataSource) BeforeCreate(tx *gorm.DB) error {
	if ds.ID == "" {
		ds.ID = uuid.New().String()
	}
	if ds.CreatedBy == "" {
		ds.CreatedBy = "system"
	}
	if ds.UpdatedBy == "" {
		ds.UpdatedBy = "system"
	}
	return nil
}

func (ds *DataSource) BeforeUpdate(tx *gorm.DB) error {
	if ds.UpdatedBy == "" {
		ds.UpdatedBy = "system"
	}
	return nil
}

func (cr *CleansingRule) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == "" {
		cr.ID = uuid.New().String()
	}
	if cr.CreatedBy == "" {
		cr.CreatedBy = "system"
	}
	return nil
}

// DataSourceStatus 数据源状态模型
type DataSourceStatus struct {
	ID              string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	DataSourceID    string                 `json:"data_source_id" gorm:"not null;type:varchar(36);unique;index"`
	Status          string                 `json:"status" gorm:"not null;size:20;default:'unknown'"` // online, offline, error, testing
	LastTestTime    *time.Time             `json:"last_test_time,omitempty"`
	LastSyncTime    *time.Time             `json:"last_sync_time,omitempty"`
	LastErrorTime   *time.Time             `json:"last_error_time,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty" gorm:"type:text"`
	ConnectionInfo  map[string]interface{} `json:"connection_info,omitempty" gorm:"type:jsonb"`  // 连接统计信息
	PerformanceInfo map[string]interface{} `json:"performance_info,omitempty" gorm:"type:jsonb"` // 性能统计信息
	SyncStatistics  map[string]interface{} `json:"sync_statistics,omitempty" gorm:"type:jsonb"`  // 同步统计信息
	HealthScore     int                    `json:"health_score" gorm:"default:0"`                // 健康评分 0-100
	UpdatedAt       time.Time              `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	DataSource DataSource `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`
}

// InterfaceStatus 接口状态模型
type InterfaceStatus struct {
	ID              string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	InterfaceID     string                 `json:"interface_id" gorm:"not null;type:varchar(36);unique;index"`
	Status          string                 `json:"status" gorm:"not null;size:20;default:'unknown'"` // active, inactive, error, testing
	LastTestTime    *time.Time             `json:"last_test_time,omitempty"`
	LastQueryTime   *time.Time             `json:"last_query_time,omitempty"`
	LastErrorTime   *time.Time             `json:"last_error_time,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty" gorm:"type:text"`
	QueryStatistics map[string]interface{} `json:"query_statistics,omitempty" gorm:"type:jsonb"` // 查询统计
	DataStatistics  map[string]interface{} `json:"data_statistics,omitempty" gorm:"type:jsonb"`  // 数据统计
	PerformanceInfo map[string]interface{} `json:"performance_info,omitempty" gorm:"type:jsonb"` // 性能信息
	QualityScore    int                    `json:"quality_score" gorm:"default:0"`               // 数据质量评分 0-100
	UpdatedAt       time.Time              `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	DataInterface DataInterface `json:"data_interface,omitempty" gorm:"foreignKey:InterfaceID"`
}

func (dss *DataSourceStatus) BeforeCreate(tx *gorm.DB) error {
	if dss.ID == "" {
		dss.ID = uuid.New().String()
	}
	return nil
}

func (is *InterfaceStatus) BeforeCreate(tx *gorm.DB) error {
	if is.ID == "" {
		is.ID = uuid.New().String()
	}
	return nil
}
