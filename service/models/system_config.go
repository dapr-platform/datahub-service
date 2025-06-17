/*
 * @module service/models/system_config
 * @description 系统配置模型，用于存储应用程序配置信息
 * @architecture 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 配置存储 -> 配置读取 -> 配置更新
 * @rules 确保配置数据的安全性和一致性
 * @dependencies gorm.io/gorm
 * @refs ai_docs/patch_basic_library_process.md
 */

package models

import (
	"time"
)

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          string    `gorm:"type:varchar(50);primaryKey" json:"id"`
	Key         string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_config_key_env" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Environment string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_config_key_env" json:"environment"`
	Version     string    `gorm:"type:varchar(20)" json:"version"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}
