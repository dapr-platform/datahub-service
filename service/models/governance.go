/*
 * @module service/models/governance
 * @description 数据治理相关模型定义，包括数据质量、元数据、脱敏规则等
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/model.md
 * @stateFlow 数据治理生命周期管理
 * @rules 确保数据质量、安全性和合规性
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs ai_docs/requirements.md
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QualityRuleTemplate 数据质量规则模板模型（不绑定具体表字段）
type QualityRuleTemplate struct {
	ID            string                 `gorm:"type:uuid;primary_key" json:"id"`
	Name          string                 `gorm:"not null" json:"name"`
	Type          string                 `gorm:"not null" json:"type"`     // completeness/standardization/consistency/accuracy/uniqueness/timeliness
	Category      string                 `gorm:"not null" json:"category"` // basic_quality/data_cleansing/data_validation
	Description   string                 `gorm:"type:text" json:"description"`
	RuleLogic     map[string]interface{} `gorm:"type:jsonb;not null" json:"rule_logic"`     // 规则逻辑模板
	Parameters    map[string]interface{} `gorm:"type:jsonb" json:"parameters"`              // 可配置参数定义
	DefaultConfig map[string]interface{} `gorm:"type:jsonb" json:"default_config"`          // 默认配置
	IsBuiltIn     bool                   `gorm:"not null;default:false" json:"is_built_in"` // 是否为内置模板
	IsEnabled     bool                   `gorm:"not null;default:true" json:"is_enabled"`
	Version       string                 `gorm:"not null;default:'1.0'" json:"version"`
	Tags          map[string]interface{} `gorm:"type:jsonb" json:"tags"` // 标签，用于分类和搜索
	CreatedAt     time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy     string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
	UpdatedAt     time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy     string                 `gorm:"not null;default:'system';size:100" json:"updated_by"`
}

// QualityRuleConfig 数据质量规则配置（运行时应用）
type QualityRuleConfig struct {
	RuleTemplateID string                 `json:"rule_template_id"`
	TargetFields   []string               `json:"target_fields"`  // 目标字段列表
	RuntimeConfig  map[string]interface{} `json:"runtime_config"` // 运行时配置
	Threshold      map[string]interface{} `json:"threshold"`      // 阈值配置
	IsEnabled      bool                   `json:"is_enabled"`
}

// BeforeCreate 创建前钩子
func (q *QualityRuleTemplate) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.CreatedBy == "" {
		q.CreatedBy = "system"
	}
	if q.UpdatedBy == "" {
		q.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (q *QualityRuleTemplate) BeforeUpdate(tx *gorm.DB) error {
	if q.UpdatedBy == "" {
		q.UpdatedBy = "system"
	}
	return nil
}

// Metadata 元数据模型
type Metadata struct {
	ID                string                 `gorm:"type:uuid;primary_key" json:"id"`
	Type              string                 `gorm:"not null" json:"type"` // technical/business/management
	Name              string                 `gorm:"not null" json:"name"`
	Content           map[string]interface{} `gorm:"type:jsonb;not null" json:"content"`
	RelatedObjectID   *string                `json:"related_object_id"`
	RelatedObjectType *string                `json:"related_object_type"` // basic_library/data_interface/thematic_library等
	CreatedAt         time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy         string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
	UpdatedAt         time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy         string                 `gorm:"not null;default:'system';size:100" json:"updated_by"`
}

// BeforeCreate 创建前钩子
func (m *Metadata) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.CreatedBy == "" {
		m.CreatedBy = "system"
	}
	if m.UpdatedBy == "" {
		m.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (m *Metadata) BeforeUpdate(tx *gorm.DB) error {
	if m.UpdatedBy == "" {
		m.UpdatedBy = "system"
	}
	return nil
}

// DataMaskingTemplate 数据脱敏规则模板模型
type DataMaskingTemplate struct {
	ID              string                 `gorm:"type:uuid;primary_key" json:"id"`
	Name            string                 `gorm:"not null" json:"name"`
	MaskingType     string                 `gorm:"not null" json:"masking_type"` // mask/replace/encrypt/pseudonymize
	Category        string                 `gorm:"not null" json:"category"`     // personal_info/financial/medical/custom
	Description     string                 `gorm:"type:text" json:"description"`
	ApplicableTypes []string               `gorm:"type:text[]" json:"applicable_types"`             // 适用的数据类型
	MaskingLogic    map[string]interface{} `gorm:"type:jsonb;not null" json:"masking_logic"`        // 脱敏逻辑模板
	Parameters      map[string]interface{} `gorm:"type:jsonb" json:"parameters"`                    // 可配置参数定义
	DefaultConfig   map[string]interface{} `gorm:"type:jsonb" json:"default_config"`                // 默认配置
	SecurityLevel   string                 `gorm:"not null;default:'medium'" json:"security_level"` // low/medium/high/critical
	IsBuiltIn       bool                   `gorm:"not null;default:false" json:"is_built_in"`       // 是否为内置模板
	IsEnabled       bool                   `gorm:"not null;default:true" json:"is_enabled"`
	Version         string                 `gorm:"not null;default:'1.0'" json:"version"`
	Tags            map[string]interface{} `gorm:"type:jsonb" json:"tags"` // 标签，用于分类和搜索
	CreatedAt       time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy       string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
	UpdatedAt       time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy       string                 `gorm:"not null;default:'system';size:100" json:"updated_by"`
}

// DataMaskingConfig 数据脱敏配置（运行时应用）
type DataMaskingConfig struct {
	TemplateID       string                 `json:"template_id"`
	TargetFields     []string               `json:"target_fields"`     // 目标字段列表
	MaskingConfig    map[string]interface{} `json:"masking_config"`    // 运行时脱敏配置
	ApplyCondition   string                 `json:"apply_condition"`   // 应用条件
	PreserveFormat   bool                   `json:"preserve_format"`   // 是否保持格式
	ReversibleConfig map[string]interface{} `json:"reversible_config"` // 可逆配置（如果支持）
	IsEnabled        bool                   `json:"is_enabled"`
}

// BeforeCreate 创建前钩子
func (d *DataMaskingTemplate) BeforeCreate(tx *gorm.DB) error {
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
func (d *DataMaskingTemplate) BeforeUpdate(tx *gorm.DB) error {
	if d.UpdatedBy == "" {
		d.UpdatedBy = "system"
	}
	return nil
}

// SystemLog 系统日志模型
type SystemLog struct {
	ID               string                 `gorm:"type:uuid;primary_key" json:"id"`
	OperationType    string                 `gorm:"not null" json:"operation_type"` // create/update/delete/query等
	ObjectType       string                 `gorm:"not null" json:"object_type"`    // basic_library/thematic_library/interface/user等
	ObjectID         *string                `json:"object_id"`
	OperatorID       *string                `json:"operator_id"`
	OperatorName     *string                `json:"operator_name"`
	OperatorIP       *string                `json:"operator_ip"`
	OperationContent map[string]interface{} `gorm:"type:jsonb;not null" json:"operation_content"`
	OperationTime    time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"operation_time"`
	OperationResult  string                 `gorm:"not null" json:"operation_result"` // success/failure
	CreatedBy        string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
}

// BeforeCreate 创建前钩子
func (s *SystemLog) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.CreatedBy == "" {
		s.CreatedBy = "system"
	}
	return nil
}

// BackupConfig 备份配置模型
type BackupConfig struct {
	ID              string                 `gorm:"type:uuid;primary_key" json:"id"`
	Name            string                 `gorm:"not null" json:"name"`
	Type            string                 `gorm:"not null" json:"type"`        // full/incremental
	ObjectType      string                 `gorm:"not null" json:"object_type"` // thematic_library/basic_library
	ObjectID        string                 `gorm:"not null" json:"object_id"`
	Strategy        map[string]interface{} `gorm:"type:jsonb;not null" json:"strategy"`
	StorageLocation string                 `gorm:"not null" json:"storage_location"`
	IsEnabled       bool                   `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt       time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy       string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
	UpdatedAt       time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy       string                 `gorm:"not null;default:'system';size:100" json:"updated_by"`
}

// BeforeCreate 创建前钩子
func (b *BackupConfig) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	if b.CreatedBy == "" {
		b.CreatedBy = "system"
	}
	if b.UpdatedBy == "" {
		b.UpdatedBy = "system"
	}
	return nil
}

// BeforeUpdate 更新前钩子
func (b *BackupConfig) BeforeUpdate(tx *gorm.DB) error {
	if b.UpdatedBy == "" {
		b.UpdatedBy = "system"
	}
	return nil
}

// BackupRecord 备份记录模型
type BackupRecord struct {
	ID             string        `gorm:"type:uuid;primary_key" json:"id"`
	BackupConfigID string        `gorm:"not null" json:"backup_config_id"`
	BackupConfig   *BackupConfig `gorm:"foreignKey:BackupConfigID" json:"backup_config,omitempty"`
	StartTime      time.Time     `gorm:"not null" json:"start_time"`
	EndTime        *time.Time    `json:"end_time"`
	BackupSize     *int64        `json:"backup_size"`
	Status         string        `gorm:"not null" json:"status"` // in_progress/success/failure
	FilePath       *string       `json:"file_path"`
	ErrorMessage   *string       `json:"error_message"`
	CreatedBy      string        `gorm:"not null;default:'system';size:100" json:"created_by"`
}

// BeforeCreate 创建前钩子
func (b *BackupRecord) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	if b.CreatedBy == "" {
		b.CreatedBy = "system"
	}
	return nil
}

// DataQualityReport 数据质量报告模型
type DataQualityReport struct {
	ID                string                 `gorm:"type:uuid;primary_key" json:"id"`
	ReportName        string                 `gorm:"not null" json:"report_name"`
	RelatedObjectID   string                 `gorm:"not null" json:"related_object_id"`
	RelatedObjectType string                 `gorm:"not null" json:"related_object_type"`
	QualityScore      float64                `gorm:"not null" json:"quality_score"`
	QualityMetrics    map[string]interface{} `gorm:"type:jsonb;not null" json:"quality_metrics"`
	Issues            map[string]interface{} `gorm:"type:jsonb" json:"issues"`
	Recommendations   map[string]interface{} `gorm:"type:jsonb" json:"recommendations"`
	GeneratedAt       time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"generated_at"`
	GeneratedBy       string                 `gorm:"not null" json:"generated_by"`
	GeneratorName     string                 `json:"generator_name"`
	CreatedBy         string                 `gorm:"not null;default:'system';size:100" json:"created_by"`
}

// BeforeCreate 创建前钩子
func (d *DataQualityReport) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.CreatedBy == "" {
		d.CreatedBy = "system"
	}
	return nil
}
