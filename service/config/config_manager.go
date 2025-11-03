/*
 * @module service/config/config_manager
 * @description 配置管理器，基于数据库的配置管理，支持优先级：数据库 > 环境变量 > 默认值
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 配置读取 -> 优先级判断 -> 返回配置值
 * @rules 确保配置的一致性和安全性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/patch_basic_library_process.md
 */

package config

import (
	"datahub-service/service/models"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// 配置键常量
const (
	ConfigKeyBasicSyncLogRetentionDays    = "basic_sync_log_retention_days"
	ConfigKeyThematicSyncLogRetentionDays = "thematic_sync_log_retention_days"

	// 默认值
	DefaultBasicSyncLogRetentionDays    = 7
	DefaultThematicSyncLogRetentionDays = 7

	// 环境变量前缀
	EnvPrefix = "DATAHUB_"
)

// ConfigManager 配置管理器（简化版）
type ConfigManager struct {
	db         *gorm.DB
	configLock sync.RWMutex
	// 配置缓存，避免频繁查询数据库
	configCache map[string]string
}

// 默认配置映射
var defaultConfigs = map[string]string{
	ConfigKeyBasicSyncLogRetentionDays:    strconv.Itoa(DefaultBasicSyncLogRetentionDays),
	ConfigKeyThematicSyncLogRetentionDays: strconv.Itoa(DefaultThematicSyncLogRetentionDays),
}

// NewConfigManager 创建配置管理器实例
func NewConfigManager(db *gorm.DB) *ConfigManager {
	return &ConfigManager{
		db:          db,
		configCache: make(map[string]string),
	}
}

// GetConfig 获取配置值（优先级：数据库 > 环境变量 > 默认值）
func (c *ConfigManager) GetConfig(key string) (string, error) {
	c.configLock.RLock()
	// 先检查缓存
	if cached, exists := c.configCache[key]; exists {
		c.configLock.RUnlock()
		return cached, nil
	}
	c.configLock.RUnlock()

	// 1. 尝试从数据库读取
	var config models.SystemConfig
	err := c.db.Where("key = ? AND environment = ?", key, "default").First(&config).Error
	if err == nil {
		// 更新缓存
		c.configLock.Lock()
		c.configCache[key] = config.Value
		c.configLock.Unlock()
		return config.Value, nil
	}

	// 2. 尝试从环境变量读取
	envKey := EnvPrefix + convertToEnvKey(key)
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue, nil
	}

	// 3. 返回默认值
	if defaultValue, exists := defaultConfigs[key]; exists {
		return defaultValue, nil
	}

	return "", fmt.Errorf("配置项不存在: %s", key)
}

// SetConfig 设置配置到数据库
func (c *ConfigManager) SetConfig(key, value, description string) error {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	// 清除缓存
	delete(c.configCache, key)

	// 检查配置是否存在
	var config models.SystemConfig
	err := c.db.Where("key = ? AND environment = ?", key, "default").First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = models.SystemConfig{
			Key:         key,
			Value:       value,
			Environment: "default",
			Description: description,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		return c.db.Create(&config).Error
	} else if err != nil {
		return fmt.Errorf("查询配置失败: %w", err)
	}

	// 更新已存在的配置
	config.Value = value
	if description != "" {
		config.Description = description
	}
	config.UpdatedAt = time.Now()

	return c.db.Save(&config).Error
}

// GetAllConfigs 获取所有配置
func (c *ConfigManager) GetAllConfigs() (map[string]string, error) {
	var configs []models.SystemConfig
	err := c.db.Where("environment = ?", "default").Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("查询配置失败: %w", err)
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.Key] = config.Value
	}

	// 添加默认配置（如果数据库中不存在）
	for key, value := range defaultConfigs {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}

	return result, nil
}

// GetConfigWithType 按类型获取配置
func (c *ConfigManager) GetConfigWithType(key string, valueType string) (interface{}, error) {
	valueStr, err := c.GetConfig(key)
	if err != nil {
		return nil, err
	}

	switch valueType {
	case "int":
		return strconv.Atoi(valueStr)
	case "bool":
		return strconv.ParseBool(valueStr)
	case "float":
		return strconv.ParseFloat(valueStr, 64)
	case "string":
		return valueStr, nil
	default:
		return valueStr, nil
	}
}

// ClearCache 清除配置缓存
func (c *ConfigManager) ClearCache() {
	c.configLock.Lock()
	defer c.configLock.Unlock()
	c.configCache = make(map[string]string)
}

// convertToEnvKey 将配置键转换为环境变量键（小写下划线转大写）
func convertToEnvKey(key string) string {
	return strings.ToUpper(key)
}
