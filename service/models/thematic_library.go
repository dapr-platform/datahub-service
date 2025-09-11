/*
 * @module service/models/thematic_library
 * @description 数据主题库相关模型定义，包括主题库、主题接口、流程图等
 * @architecture DDD领域驱动设计 - 实体模型
 * @documentReference dev_docs/model.md
 * @stateFlow 数据主题库生命周期管理
 * @rules 遵循数据库设计规范，支持复杂的数据处理流程
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs dev_docs/requirements.md
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ThematicLibrary 数据主题库模型
type ThematicLibrary struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	NameZh          string     `json:"name_zh" gorm:"not null;size:255"`
	NameEn          string     `json:"name_en" gorm:"not null;unique;size:255"`
	Category        string     `json:"category" gorm:"not null;size:50"` // business, technical, analysis, report
	Domain          string     `json:"domain" gorm:"not null;size:50"`   // user, order, product, finance, marketing
	Description     string     `json:"description" gorm:"size:1000"`
	Tags            JSONBArray `json:"tags" gorm:"type:jsonb"`
	SourceLibraries JSONBArray `json:"source_libraries" gorm:"type:jsonb"`
	PublishStatus   string     `json:"publish_status" gorm:"not null;default:'draft';size:20"` // draft, published, archived
	Version         string     `json:"version" gorm:"not null;default:'1.0.0';size:20"`
	AccessLevel     string     `json:"access_level" gorm:"not null;default:'internal';size:20"` // public, internal, private
	AuthorizedUsers JSONBArray `json:"authorized_users" gorm:"type:jsonb"`
	AuthorizedRoles JSONBArray `json:"authorized_roles" gorm:"type:jsonb"`
	UpdateFrequency string     `json:"update_frequency" gorm:"not null;default:'daily';size:20"` // realtime, hourly, daily, weekly, monthly
	RetentionPeriod int        `json:"retention_period" gorm:"not null;default:365"`
	CreatedAt       time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy       string     `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy       string     `json:"updated_by" gorm:"not null;default:'system';size:100"`
	Status          string     `json:"status" gorm:"not null;default:'active';size:20"`
}

type ThematicInterface struct {
	ID                string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	LibraryID         string    `json:"library_id" gorm:"not null;type:varchar(36);index"`
	NameZh            string    `json:"name_zh" gorm:"not null;size:255"`
	NameEn            string    `json:"name_en" gorm:"not null;size:255"`
	Type              string    `json:"type" gorm:"not null;size:20"` // realtime, batch
	Description       string    `json:"description" gorm:"size:1000"`
	DataSourceID      string    `json:"data_source_id" gorm:"type:varchar(36)"`
	CreatedAt         time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy         string    `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy         string    `json:"updated_by" gorm:"not null;default:'system';size:100"`
	Status            string    `json:"status" gorm:"not null;default:'active';size:20"`
	IsTableCreated    bool      `json:"is_table_created" gorm:"not null;default:false"`
	InterfaceConfig   JSONB     `json:"interface_config" gorm:"type:jsonb"`
	ParseConfig       JSONB     `json:"parse_config" gorm:"type:jsonb"`
	TableFieldsConfig JSONB     `json:"table_fields_config" gorm:"type:jsonb"`
	// 关联关系
	ThematicLibrary ThematicLibrary `json:"thematic_library,omitempty" gorm:"foreignKey:LibraryID"`
	DataSource      DataSource      `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`
}

// DataFlowGraph 数据流程图模型
type DataFlowGraph struct {
	ID                  string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ThematicInterfaceID string                 `json:"thematic_interface_id" gorm:"not null;type:varchar(36);index"`
	Name                string                 `json:"name" gorm:"not null;size:255"`
	Description         string                 `json:"description" gorm:"size:1000"`
	Definition          map[string]interface{} `json:"definition" gorm:"type:jsonb;not null"`
	CreatedAt           time.Time              `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy           string                 `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedAt           time.Time              `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy           string                 `json:"updated_by" gorm:"not null;default:'system';size:100"`
	Status              string                 `json:"status" gorm:"not null;default:'active';size:20"` // draft, active, inactive

	// 关联关系
	Nodes []FlowNode `json:"nodes,omitempty" gorm:"foreignKey:FlowGraphID"`
}

// FlowNode 流程图节点模型
type FlowNode struct {
	ID          string                 `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FlowGraphID string                 `json:"flow_graph_id" gorm:"not null;type:varchar(36);index"`
	Type        string                 `json:"type" gorm:"not null;size:50"` // datasource, api, file, filter, transform, aggregate, output
	Config      map[string]interface{} `json:"config" gorm:"type:jsonb;not null"`
	PositionX   int                    `json:"position_x" gorm:"not null"`
	PositionY   int                    `json:"position_y" gorm:"not null"`
	Name        string                 `json:"name" gorm:"not null;size:255"`
	CreatedAt   time.Time              `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy   string                 `json:"created_by" gorm:"not null;default:'system';size:100"`

	// 关联关系
	DataFlowGraph DataFlowGraph `json:"data_flow_graph,omitempty" gorm:"foreignKey:FlowGraphID"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (tl *ThematicLibrary) BeforeCreate(tx *gorm.DB) error {
	if tl.ID == "" {
		tl.ID = uuid.New().String()
	}
	if tl.CreatedBy == "" {
		tl.CreatedBy = "system"
	}
	if tl.UpdatedBy == "" {
		tl.UpdatedBy = "system"
	}
	return nil
}

func (tl *ThematicLibrary) BeforeUpdate(tx *gorm.DB) error {
	if tl.UpdatedBy == "" {
		tl.UpdatedBy = "system"
	}
	return nil
}

func (ti *ThematicInterface) BeforeCreate(tx *gorm.DB) error {
	if ti.ID == "" {
		ti.ID = uuid.New().String()
	}
	if ti.CreatedBy == "" {
		ti.CreatedBy = "system"
	}
	if ti.UpdatedBy == "" {
		ti.UpdatedBy = "system"
	}
	return nil
}

func (ti *ThematicInterface) BeforeUpdate(tx *gorm.DB) error {
	if ti.UpdatedBy == "" {
		ti.UpdatedBy = "system"
	}
	return nil
}

func (dfg *DataFlowGraph) BeforeCreate(tx *gorm.DB) error {
	if dfg.ID == "" {
		dfg.ID = uuid.New().String()
	}
	if dfg.CreatedBy == "" {
		dfg.CreatedBy = "system"
	}
	if dfg.UpdatedBy == "" {
		dfg.UpdatedBy = "system"
	}
	return nil
}

func (dfg *DataFlowGraph) BeforeUpdate(tx *gorm.DB) error {
	if dfg.UpdatedBy == "" {
		dfg.UpdatedBy = "system"
	}
	return nil
}

func (fn *FlowNode) BeforeCreate(tx *gorm.DB) error {
	if fn.ID == "" {
		fn.ID = uuid.New().String()
	}
	if fn.CreatedBy == "" {
		fn.CreatedBy = "system"
	}
	return nil
}
