/*
 * @module service/models/config_models
 * @description 配置管理相关模型定义，包含应用配置、数据库配置、服务器配置等核心配置结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 配置加载 -> 配置验证 -> 配置应用
 * @rules 确保配置模型的一致性和完整性
 * @dependencies gopkg.in/yaml.v3
 * @refs service/config
 */

package models

import (
	"time"
)

// ApplicationConfig 应用配置
type ApplicationConfig struct {
	// 应用基础配置
	App      AppConfig      `json:"app" yaml:"app"`
	Database DatabaseConfig `json:"database" yaml:"database"`
	Server   ServerConfig   `json:"server" yaml:"server"`
	Logging  LoggingConfig  `json:"logging" yaml:"logging"`

	// 业务模块配置
	DataSources DataSourcesConfig      `json:"data_sources" yaml:"data_sources"`
	SyncEngine  SyncEngineConfig       `json:"sync_engine" yaml:"sync_engine"`
	Monitoring  ConfigMonitoringConfig `json:"monitoring" yaml:"monitoring"`
	DataQuality DataQualityConfig      `json:"data_quality" yaml:"data_quality"`
	Scheduler   SchedulerConfig        `json:"scheduler" yaml:"scheduler"`

	// 扩展配置
	Extensions map[string]interface{} `json:"extensions" yaml:"extensions"`
	Features   FeatureFlags           `json:"features" yaml:"features"`
}

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

// ConfigMonitoringConfig 监控配置
type ConfigMonitoringConfig struct {
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
	MaxWorkers    int           `json:"max_workers" yaml:"max_workers"`
	QueueSize     int           `json:"queue_size" yaml:"queue_size"`
	CheckInterval time.Duration `json:"check_interval" yaml:"check_interval"`
}

// FeatureFlags 功能特性开关
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

// ConfigWatcher 配置文件监控器
type ConfigWatcher struct {
	FilePath     string
	LastModified time.Time
	IsActive     bool
}

// ConfigChangeNotifier 配置变更通知器接口
type ConfigChangeNotifier interface {
	OnConfigChanged(oldConfig, newConfig *ApplicationConfig, changes []ConfigChange) error
}
