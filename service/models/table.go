package models

// TableSchemaRequest 表结构管理请求
type TableSchemaRequest struct {
	InterfaceID string       `json:"interface_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Operation   string       `json:"operation" validate:"required" example:"create_table"` // create_table, alter_table, drop_table
	SchemaName  string       `json:"schema_name" validate:"required" example:"user_basic_library"`
	TableName   string       `json:"table_name" validate:"required" example:"users"`
	Fields      []TableField `json:"fields,omitempty"`
}

// TableField 表字段模型
type TableField struct {
	NameZh          string `json:"name_zh" gorm:"not null;size:255"`
	NameEn          string `json:"name_en" gorm:"not null;size:255"`
	DataType        string `json:"data_type" gorm:"not null;size:50"`
	IsPrimaryKey    bool   `json:"is_primary_key" gorm:"not null;default:false"`
	IsUnique        bool   `json:"is_unique" gorm:"not null;default:false"`
	IsNullable      bool   `json:"is_nullable" gorm:"not null;default:true"`
	DefaultValue    string `json:"default_value" gorm:"size:255"`
	Description     string `json:"description" gorm:"size:1000"`
	OrderNum        int    `json:"order_num" gorm:"not null"`
	CheckConstraint string `json:"check_constraint" gorm:"size:255"`
}
