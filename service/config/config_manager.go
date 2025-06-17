/*
 * @module service/config/config_manager
 * @description 配置管理器，负责配置加载、配置验证、配置热更新和配置版本管理
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 配置加载 -> 配置验证 -> 配置应用 -> 变更监听
 * @rules 确保配置的一致性和安全性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/patch_basic_library_process.md
 */

package config

import (
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	db         *gorm.DB
	config     *ApplicationConfig
	configLock sync.RWMutex

	// 配置文件路径
	configFilePaths []string
	configFormat    string // json, yaml, env

	// 热更新
	watcherEnabled  bool
	configWatchers  map[string]*ConfigWatcher
	changeNotifiers []ConfigChangeNotifier

	// 配置版本管理
	configHistory   []*ConfigVersion
	currentVersion  string
	maxHistoryCount int

	// 环境变量覆盖
	envPrefix    string
	envOverrides map[string]interface{}

	// 缓存
	configCache  map[string]interface{}
	cacheEnabled bool
	cacheExpiry  time.Duration
}

// 使用models包中定义的类型
type ApplicationConfig = models.ApplicationConfig

// AppConfig 应用配置
type AppConfig struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Environment string `json:"environment" yaml:"environment"`
	Debug       bool   `json:"debug" yaml:"debug"`
	Timezone    string `json:"timezone" yaml:"timezone"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `json:"host" yaml:"host"`
	Port         int    `json:"port" yaml:"port"`
	Database     string `json:"database" yaml:"database"`
	Username     string `json:"username" yaml:"username"`
	Password     string `json:"password" yaml:"password"`
	MaxOpenConns int    `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns" yaml:"max_idle_conns"`
	SSLMode      string `json:"ssl_mode" yaml:"ssl_mode"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	CORS         CORSConfig    `json:"cors" yaml:"cors"`
	TLS          TLSConfig     `json:"tls" yaml:"tls"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level    string          `json:"level" yaml:"level"`
	Format   string          `json:"format" yaml:"format"`
	Output   []string        `json:"output" yaml:"output"`
	File     LogFileConfig   `json:"file" yaml:"file"`
	Rotation LogRotateConfig `json:"rotation" yaml:"rotation"`
}

// LogFileConfig 日志文件配置
type LogFileConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Path     string `json:"path" yaml:"path"`
	Filename string `json:"filename" yaml:"filename"`
}

// LogRotateConfig 日志轮转配置
type LogRotateConfig struct {
	Enabled    bool `json:"enabled" yaml:"enabled"`
	MaxSize    int  `json:"max_size" yaml:"max_size"` // MB
	MaxAge     int  `json:"max_age" yaml:"max_age"`   // 天
	MaxBackups int  `json:"max_backups" yaml:"max_backups"`
	Compress   bool `json:"compress" yaml:"compress"`
}

// DataSourcesConfig 数据源配置
type DataSourcesConfig struct {
	DefaultTimeout      time.Duration `json:"default_timeout" yaml:"default_timeout"`
	MaxRetries          int           `json:"max_retries" yaml:"max_retries"`
	RetryInterval       time.Duration `json:"retry_interval" yaml:"retry_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	ConnectionPoolSize  int           `json:"connection_pool_size" yaml:"connection_pool_size"`
}

// SyncEngineConfig 同步引擎配置
type SyncEngineConfig struct {
	MaxConcurrentTasks int                `json:"max_concurrent_tasks" yaml:"max_concurrent_tasks"`
	BatchSize          int                `json:"batch_size" yaml:"batch_size"`
	SyncInterval       time.Duration      `json:"sync_interval" yaml:"sync_interval"`
	RetryPolicy        RetryPolicyConfig  `json:"retry_policy" yaml:"retry_policy"`
	Transformers       TransformersConfig `json:"transformers" yaml:"transformers"`
}

// RetryPolicyConfig 重试策略配置
type RetryPolicyConfig struct {
	MaxRetries    int           `json:"max_retries" yaml:"max_retries"`
	BackoffFactor float64       `json:"backoff_factor" yaml:"backoff_factor"`
	MaxBackoff    time.Duration `json:"max_backoff" yaml:"max_backoff"`
}

// TransformersConfig 转换器配置
type TransformersConfig struct {
	Enabled          bool     `json:"enabled" yaml:"enabled"`
	DefaultTransform string   `json:"default_transform" yaml:"default_transform"`
	CustomTransforms []string `json:"custom_transforms" yaml:"custom_transforms"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled              bool          `json:"enabled" yaml:"enabled"`
	MetricsInterval      time.Duration `json:"metrics_interval" yaml:"metrics_interval"`
	HealthCheckInterval  time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	AlertsEnabled        bool          `json:"alerts_enabled" yaml:"alerts_enabled"`
	NotificationChannels []string      `json:"notification_channels" yaml:"notification_channels"`
}

// DataQualityConfig 数据质量配置
type DataQualityConfig struct {
	Enabled          bool     `json:"enabled" yaml:"enabled"`
	ValidationRules  []string `json:"validation_rules" yaml:"validation_rules"`
	QualityThreshold float64  `json:"quality_threshold" yaml:"quality_threshold"`
	AutoCleanEnabled bool     `json:"auto_clean_enabled" yaml:"auto_clean_enabled"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Enabled       bool          `json:"enabled" yaml:"enabled"`
	MaxWorkers    int           `json:"max_workers" yaml:"max_workers"`
	QueueSize     int           `json:"queue_size" yaml:"queue_size"`
	CheckInterval time.Duration `json:"check_interval" yaml:"check_interval"`
}

// FeatureFlags 功能标志配置
type FeatureFlags struct {
	RealtimeSync      bool `json:"realtime_sync" yaml:"realtime_sync"`
	DataCompression   bool `json:"data_compression" yaml:"data_compression"`
	AdvancedAnalytics bool `json:"advanced_analytics" yaml:"advanced_analytics"`
	APIRateLimit      bool `json:"api_rate_limit" yaml:"api_rate_limit"`
}

// ConfigVersion 配置版本
type ConfigVersion struct {
	Version     string             `json:"version"`
	Config      *ApplicationConfig `json:"config"`
	CreatedAt   time.Time          `json:"created_at"`
	CreatedBy   string             `json:"created_by"`
	Description string             `json:"description"`
	Changes     []ConfigChange     `json:"changes"`
}

// ConfigChange 配置变更
type ConfigChange struct {
	Path     string      `json:"path"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Action   string      `json:"action"` // add, update, delete
}

// ConfigWatcher 配置监听器
type ConfigWatcher struct {
	FilePath     string
	LastModified time.Time
	IsActive     bool
}

// ConfigChangeNotifier 配置变更通知器
type ConfigChangeNotifier interface {
	OnConfigChanged(oldConfig, newConfig *ApplicationConfig, changes []ConfigChange) error
}

// NewConfigManager 创建配置管理器实例
func NewConfigManager(db *gorm.DB) *ConfigManager {
	return &ConfigManager{
		db:              db,
		configFilePaths: []string{"config.yaml", "config.json"},
		configFormat:    "yaml",
		watcherEnabled:  true,
		configWatchers:  make(map[string]*ConfigWatcher),
		changeNotifiers: []ConfigChangeNotifier{},
		configHistory:   []*ConfigVersion{},
		maxHistoryCount: 10,
		envPrefix:       "DATAHUB_",
		envOverrides:    make(map[string]interface{}),
		configCache:     make(map[string]interface{}),
		cacheEnabled:    true,
		cacheExpiry:     5 * time.Minute,
	}
}

// LoadConfig 加载配置
func (c *ConfigManager) LoadConfig() error {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	// 1. 尝试从文件加载配置
	config, err := c.loadConfigFromFile()
	if err != nil {
		// 如果文件加载失败，尝试从数据库加载
		config, err = c.loadConfigFromDB()
		if err != nil {
			// 如果都失败，使用默认配置
			config = c.getDefaultConfig()
		}
	}

	// 2. 应用环境变量覆盖
	c.applyEnvironmentOverrides(config)

	// 3. 验证配置
	if err := c.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	// 4. 保存配置
	oldConfig := c.config
	c.config = config

	// 5. 如果启用监听，开始监听配置文件变化
	if c.watcherEnabled {
		c.startConfigWatcher()
	}

	// 6. 通知配置变更
	if oldConfig != nil {
		changes := c.calculateConfigChanges(oldConfig, config)
		c.notifyConfigChange(oldConfig, config, changes)
	}

	return nil
}

// GetConfig 获取完整配置
func (c *ConfigManager) GetConfig() *ApplicationConfig {
	c.configLock.RLock()
	defer c.configLock.RUnlock()
	return c.config
}

// GetConfigValue 获取指定路径的配置值
func (c *ConfigManager) GetConfigValue(path string) (interface{}, error) {
	c.configLock.RLock()
	defer c.configLock.RUnlock()

	// 先检查缓存
	if c.cacheEnabled {
		if value, exists := c.configCache[path]; exists {
			return value, nil
		}
	}

	value, err := c.getValueByPath(c.config, path)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	if c.cacheEnabled {
		c.configCache[path] = value
	}

	return value, nil
}

// SetConfigValue 设置配置值
func (c *ConfigManager) SetConfigValue(path string, value interface{}) error {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	oldValue, _ := c.getValueByPath(c.config, path)

	if err := c.setValueByPath(c.config, path, value); err != nil {
		return fmt.Errorf("设置配置值失败: %v", err)
	}

	// 清除缓存
	if c.cacheEnabled {
		delete(c.configCache, path)
	}

	// 验证配置
	if err := c.validateConfig(c.config); err != nil {
		// 如果验证失败，回滚变更
		c.setValueByPath(c.config, path, oldValue)
		return fmt.Errorf("配置验证失败，已回滚: %v", err)
	}

	// 记录变更
	change := ConfigChange{
		Path:     path,
		OldValue: oldValue,
		NewValue: value,
		Action:   "update",
	}

	// 通知变更
	c.notifyConfigChange(nil, c.config, []ConfigChange{change})

	return nil
}

// ReloadConfig 重新加载配置
func (c *ConfigManager) ReloadConfig() error {
	return c.LoadConfig()
}

// SaveConfigToDB 保存配置到数据库
func (c *ConfigManager) SaveConfigToDB() error {
	c.configLock.RLock()
	defer c.configLock.RUnlock()

	configJSON, err := json.Marshal(c.config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	configRecord := &models.SystemConfig{
		Key:         "application_config",
		Value:       string(configJSON),
		Environment: c.config.App.Environment,
		Version:     c.currentVersion,
		UpdatedAt:   time.Now(),
	}

	return c.db.Save(configRecord).Error
}

// AddChangeNotifier 添加配置变更通知器
func (c *ConfigManager) AddChangeNotifier(notifier ConfigChangeNotifier) {
	c.changeNotifiers = append(c.changeNotifiers, notifier)
}

// GetConfigHistory 获取配置历史
func (c *ConfigManager) GetConfigHistory() []*ConfigVersion {
	return c.configHistory
}

// RollbackToVersion 回滚到指定版本
func (c *ConfigManager) RollbackToVersion(version string) error {
	for _, v := range c.configHistory {
		if v.Version == version {
			oldConfig := c.config
			c.config = v.Config
			c.currentVersion = version

			// 验证配置
			if err := c.validateConfig(c.config); err != nil {
				c.config = oldConfig
				return fmt.Errorf("回滚后配置验证失败: %v", err)
			}

			// 通知变更
			changes := c.calculateConfigChanges(oldConfig, c.config)
			c.notifyConfigChange(oldConfig, c.config, changes)

			return nil
		}
	}

	return fmt.Errorf("版本 %s 不存在", version)
}

// 从文件加载配置
func (c *ConfigManager) loadConfigFromFile() (*ApplicationConfig, error) {
	var configData []byte
	var err error
	var configPath string

	// 尝试各个配置文件路径
	for _, path := range c.configFilePaths {
		if _, err := os.Stat(path); err == nil {
			configData, err = ioutil.ReadFile(path)
			if err == nil {
				configPath = path
				break
			}
		}
	}

	if configData == nil {
		return nil, fmt.Errorf("未找到可用的配置文件")
	}

	config := &ApplicationConfig{}

	// 根据文件扩展名决定解析方式
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(configData, config)
	case ".json":
		err = json.Unmarshal(configData, config)
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return config, nil
}

// 从数据库加载配置
func (c *ConfigManager) loadConfigFromDB() (*ApplicationConfig, error) {
	var configRecord models.SystemConfig
	err := c.db.Where("key = ?", "application_config").First(&configRecord).Error
	if err != nil {
		return nil, fmt.Errorf("从数据库加载配置失败: %v", err)
	}

	config := &ApplicationConfig{}
	err = json.Unmarshal([]byte(configRecord.Value), config)
	if err != nil {
		return nil, fmt.Errorf("反序列化配置失败: %v", err)
	}

	c.currentVersion = configRecord.Version

	return config, nil
}

// 获取默认配置
func (c *ConfigManager) getDefaultConfig() *ApplicationConfig {
	return &ApplicationConfig{
		App: AppConfig{
			Name:        "DataHub Service",
			Version:     "1.0.0",
			Environment: "development",
			Debug:       true,
			Timezone:    "UTC",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Database:     "datahub",
			Username:     "postgres",
			Password:     "",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
			SSLMode:      "disable",
		},
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			CORS: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: []string{"stdout"},
			File: LogFileConfig{
				Enabled:  false,
				Path:     "logs",
				Filename: "app.log",
			},
		},
		DataSources: DataSourcesConfig{
			DefaultTimeout:      30 * time.Second,
			MaxRetries:          3,
			RetryInterval:       5 * time.Second,
			HealthCheckInterval: 60 * time.Second,
			ConnectionPoolSize:  10,
		},
		SyncEngine: SyncEngineConfig{
			MaxConcurrentTasks: 10,
			BatchSize:          1000,
			SyncInterval:       5 * time.Minute,
			RetryPolicy: RetryPolicyConfig{
				MaxRetries:    3,
				BackoffFactor: 2.0,
				MaxBackoff:    300 * time.Second,
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:             true,
			MetricsInterval:     60 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			AlertsEnabled:       true,
		},
		DataQuality: DataQualityConfig{
			Enabled:          true,
			QualityThreshold: 0.8,
			AutoCleanEnabled: false,
		},
		Scheduler: SchedulerConfig{
			Enabled:       true,
			MaxWorkers:    5,
			QueueSize:     100,
			CheckInterval: 30 * time.Second,
		},
		Features: FeatureFlags{
			RealtimeSync:      true,
			DataCompression:   false,
			AdvancedAnalytics: false,
			APIRateLimit:      true,
		},
		Extensions: make(map[string]interface{}),
	}
}

// 应用环境变量覆盖
func (c *ConfigManager) applyEnvironmentOverrides(config *ApplicationConfig) {
	// 简化实现，实际应该遍历所有配置字段并检查对应的环境变量
	if dbHost := os.Getenv(c.envPrefix + "DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbPort := os.Getenv(c.envPrefix + "DB_PORT"); dbPort != "" {
		// 解析端口号...
	}
	// ... 其他环境变量覆盖
}

// 验证配置
func (c *ConfigManager) validateConfig(config *ApplicationConfig) error {
	if config.App.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}

	if config.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}

	if config.Database.Port <= 0 || config.Database.Port > 65535 {
		return fmt.Errorf("数据库端口无效")
	}

	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效")
	}

	// 更多验证逻辑...

	return nil
}

// 开始配置文件监听
func (c *ConfigManager) startConfigWatcher() {
	for _, path := range c.configFilePaths {
		if stat, err := os.Stat(path); err == nil {
			watcher := &ConfigWatcher{
				FilePath:     path,
				LastModified: stat.ModTime(),
				IsActive:     true,
			}
			c.configWatchers[path] = watcher

			go c.watchConfigFile(watcher)
		}
	}
}

// 监听配置文件变化
func (c *ConfigManager) watchConfigFile(watcher *ConfigWatcher) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for watcher.IsActive {
		select {
		case <-ticker.C:
			if stat, err := os.Stat(watcher.FilePath); err == nil {
				if stat.ModTime().After(watcher.LastModified) {
					watcher.LastModified = stat.ModTime()

					// 重新加载配置
					go func() {
						if err := c.ReloadConfig(); err != nil {
							fmt.Printf("重新加载配置失败: %v\n", err)
						}
					}()
				}
			}
		}
	}
}

// 获取指定路径的值
func (c *ConfigManager) getValueByPath(config *ApplicationConfig, path string) (interface{}, error) {
	// 简化实现，实际应该支持复杂的路径解析，如 "database.host"
	parts := strings.Split(path, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("路径格式无效")
	}

	switch parts[0] {
	case "app":
		return c.getAppConfigValue(config.App, parts[1])
	case "database":
		return c.getDatabaseConfigValue(config.Database, parts[1])
	case "server":
		return c.getServerConfigValue(config.Server, parts[1])
	default:
		return nil, fmt.Errorf("未知的配置节: %s", parts[0])
	}
}

// 设置指定路径的值
func (c *ConfigManager) setValueByPath(config *ApplicationConfig, path string, value interface{}) error {
	// 简化实现
	parts := strings.Split(path, ".")
	if len(parts) != 2 {
		return fmt.Errorf("路径格式无效")
	}

	switch parts[0] {
	case "app":
		return c.setAppConfigValue(&config.App, parts[1], value)
	case "database":
		return c.setDatabaseConfigValue(&config.Database, parts[1], value)
	case "server":
		return c.setServerConfigValue(&config.Server, parts[1], value)
	default:
		return fmt.Errorf("未知的配置节: %s", parts[0])
	}
}

// 获取应用配置值
func (c *ConfigManager) getAppConfigValue(config AppConfig, field string) (interface{}, error) {
	switch field {
	case "name":
		return config.Name, nil
	case "version":
		return config.Version, nil
	case "environment":
		return config.Environment, nil
	case "debug":
		return config.Debug, nil
	case "timezone":
		return config.Timezone, nil
	default:
		return nil, fmt.Errorf("未知的应用配置字段: %s", field)
	}
}

// 设置应用配置值
func (c *ConfigManager) setAppConfigValue(config *AppConfig, field string, value interface{}) error {
	switch field {
	case "name":
		if str, ok := value.(string); ok {
			config.Name = str
		} else {
			return fmt.Errorf("应用名称必须是字符串")
		}
	case "version":
		if str, ok := value.(string); ok {
			config.Version = str
		} else {
			return fmt.Errorf("版本必须是字符串")
		}
	case "environment":
		if str, ok := value.(string); ok {
			config.Environment = str
		} else {
			return fmt.Errorf("环境必须是字符串")
		}
	case "debug":
		if b, ok := value.(bool); ok {
			config.Debug = b
		} else {
			return fmt.Errorf("debug必须是布尔值")
		}
	case "timezone":
		if str, ok := value.(string); ok {
			config.Timezone = str
		} else {
			return fmt.Errorf("时区必须是字符串")
		}
	default:
		return fmt.Errorf("未知的应用配置字段: %s", field)
	}
	return nil
}

// 获取数据库配置值
func (c *ConfigManager) getDatabaseConfigValue(config DatabaseConfig, field string) (interface{}, error) {
	switch field {
	case "host":
		return config.Host, nil
	case "port":
		return config.Port, nil
	case "database":
		return config.Database, nil
	case "username":
		return config.Username, nil
	default:
		return nil, fmt.Errorf("未知的数据库配置字段: %s", field)
	}
}

// 设置数据库配置值
func (c *ConfigManager) setDatabaseConfigValue(config *DatabaseConfig, field string, value interface{}) error {
	switch field {
	case "host":
		if str, ok := value.(string); ok {
			config.Host = str
		} else {
			return fmt.Errorf("主机必须是字符串")
		}
	case "port":
		if i, ok := value.(int); ok {
			config.Port = i
		} else {
			return fmt.Errorf("端口必须是整数")
		}
	default:
		return fmt.Errorf("未知的数据库配置字段: %s", field)
	}
	return nil
}

// 获取服务器配置值
func (c *ConfigManager) getServerConfigValue(config ServerConfig, field string) (interface{}, error) {
	switch field {
	case "host":
		return config.Host, nil
	case "port":
		return config.Port, nil
	default:
		return nil, fmt.Errorf("未知的服务器配置字段: %s", field)
	}
}

// 设置服务器配置值
func (c *ConfigManager) setServerConfigValue(config *ServerConfig, field string, value interface{}) error {
	switch field {
	case "host":
		if str, ok := value.(string); ok {
			config.Host = str
		} else {
			return fmt.Errorf("主机必须是字符串")
		}
	case "port":
		if i, ok := value.(int); ok {
			config.Port = i
		} else {
			return fmt.Errorf("端口必须是整数")
		}
	default:
		return fmt.Errorf("未知的服务器配置字段: %s", field)
	}
	return nil
}

// 计算配置变更
func (c *ConfigManager) calculateConfigChanges(oldConfig, newConfig *ApplicationConfig) []ConfigChange {
	var changes []ConfigChange

	// 简化实现，实际应该深度比较所有配置字段
	if oldConfig.App.Name != newConfig.App.Name {
		changes = append(changes, ConfigChange{
			Path:     "app.name",
			OldValue: oldConfig.App.Name,
			NewValue: newConfig.App.Name,
			Action:   "update",
		})
	}

	// ... 比较其他字段

	return changes
}

// 通知配置变更
func (c *ConfigManager) notifyConfigChange(oldConfig, newConfig *ApplicationConfig, changes []ConfigChange) {
	for _, notifier := range c.changeNotifiers {
		go func(n ConfigChangeNotifier) {
			if err := n.OnConfigChanged(oldConfig, newConfig, changes); err != nil {
				fmt.Printf("通知配置变更失败: %v\n", err)
			}
		}(notifier)
	}
}
