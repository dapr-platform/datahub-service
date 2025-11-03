/*
 * @module service/models/config_models
 * @description 配置管理相关模型定义，简化版本仅保留必要的配置结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 配置加载 -> 配置验证 -> 配置应用
 * @rules 确保配置模型的一致性和完整性
 * @dependencies gorm.io/gorm
 * @refs service/config
 */

package models

import (
	"time"
)

// SystemConfigItem 系统配置项（用于API返回）
type SystemConfigItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	ValueType   string `json:"value_type"` // string, int, bool, json
}

// ConfigVersion 配置版本（保留用于未来扩展）
type ConfigVersion struct {
	Version     string         `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
	CreatedBy   string         `json:"created_by"`
	Description string         `json:"description"`
	Changes     []ConfigChange `json:"changes"`
}

// ConfigChange 配置变更（保留用于未来扩展）
type ConfigChange struct {
	Path     string      `json:"path"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Action   string      `json:"action"` // add, update, delete
}

// ConfigWatcher 配置文件监控器（保留用于未来扩展）
type ConfigWatcher struct {
	FilePath     string
	LastModified time.Time
	IsActive     bool
}
