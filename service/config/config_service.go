/*
 * @module service/config/config_service
 * @description 配置服务，提供业务层的配置管理功能
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 服务调用 -> 配置管理器 -> 数据库/环境变量/默认值
 * @rules 确保配置操作的业务逻辑正确性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/config/config_manager.go
 */

package config

import (
	"datahub-service/service/models"
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// ConfigService 配置服务
type ConfigService struct {
	db      *gorm.DB
	manager *ConfigManager
}

// NewConfigService 创建配置服务实例
func NewConfigService(db *gorm.DB) *ConfigService {
	return &ConfigService{
		db:      db,
		manager: NewConfigManager(db),
	}
}

// GetSystemConfig 获取系统配置
func (s *ConfigService) GetSystemConfig(key string) (string, error) {
	return s.manager.GetConfig(key)
}

// SetSystemConfig 设置系统配置
func (s *ConfigService) SetSystemConfig(key, value, description string) error {
	return s.manager.SetConfig(key, value, description)
}

// GetAllSystemConfigs 获取所有系统配置
func (s *ConfigService) GetAllSystemConfigs() ([]models.SystemConfigItem, error) {
	// 从数据库获取所有配置
	var configs []models.SystemConfig
	err := s.db.Where("environment = ?", "default").Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("查询配置失败: %w", err)
	}

	// 转换为配置项列表
	items := make([]models.SystemConfigItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, models.SystemConfigItem{
			Key:         config.Key,
			Value:       config.Value,
			Description: config.Description,
			ValueType:   "string", // 简化处理，都当字符串
		})
	}

	// 添加默认配置（如果数据库中不存在）
	existingKeys := make(map[string]bool)
	for _, item := range items {
		existingKeys[item.Key] = true
	}

	// 添加默认配置
	if !existingKeys[ConfigKeyBasicSyncLogRetentionDays] {
		items = append(items, models.SystemConfigItem{
			Key:         ConfigKeyBasicSyncLogRetentionDays,
			Value:       strconv.Itoa(DefaultBasicSyncLogRetentionDays),
			Description: "基础库同步任务执行日志保存天数",
			ValueType:   "int",
		})
	}

	if !existingKeys[ConfigKeyThematicSyncLogRetentionDays] {
		items = append(items, models.SystemConfigItem{
			Key:         ConfigKeyThematicSyncLogRetentionDays,
			Value:       strconv.Itoa(DefaultThematicSyncLogRetentionDays),
			Description: "主题库同步任务执行日志保存天数",
			ValueType:   "int",
		})
	}

	return items, nil
}

// GetBasicSyncLogRetentionDays 获取基础库同步日志保留天数
func (s *ConfigService) GetBasicSyncLogRetentionDays() (int, error) {
	valueStr, err := s.manager.GetConfig(ConfigKeyBasicSyncLogRetentionDays)
	if err != nil {
		return DefaultBasicSyncLogRetentionDays, nil // 返回默认值
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return DefaultBasicSyncLogRetentionDays, nil // 解析失败返回默认值
	}

	return value, nil
}

// GetThematicSyncLogRetentionDays 获取主题库同步日志保留天数
func (s *ConfigService) GetThematicSyncLogRetentionDays() (int, error) {
	valueStr, err := s.manager.GetConfig(ConfigKeyThematicSyncLogRetentionDays)
	if err != nil {
		return DefaultThematicSyncLogRetentionDays, nil // 返回默认值
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return DefaultThematicSyncLogRetentionDays, nil // 解析失败返回默认值
	}

	return value, nil
}

// ClearCache 清除配置缓存
func (s *ConfigService) ClearCache() {
	s.manager.ClearCache()
}


