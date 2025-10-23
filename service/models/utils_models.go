/*
 * @module service/models/utils_models
 * @description 工具模块相关模型定义，包含连接池、健康检查、加密工具等核心数据结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 工具初始化 -> 功能提供
 * @rules 确保工具模块模型的一致性和安全性
 * @dependencies time, crypto/tls
 * @refs service/utils
 */

package models

import (
	"crypto/tls"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	// 基础配置
	Name               string        `json:"name"`                 // 连接池名称
	Type               string        `json:"type"`                 // 连接池类型: database, redis, http, tcp
	MaxConnections     int           `json:"max_connections"`      // 最大连接数
	MinConnections     int           `json:"min_connections"`      // 最小连接数
	InitialConnections int           `json:"initial_connections"`  // 初始连接数
	MaxIdleConnections int           `json:"max_idle_connections"` // 最大空闲连接数
	MaxActiveTime      time.Duration `json:"max_active_time"`      // 连接最大活跃时间
	MaxIdleTime        time.Duration `json:"max_idle_time"`        // 连接最大空闲时间
	ConnectionTimeout  time.Duration `json:"connection_timeout"`   // 连接超时时间
	ValidationTimeout  time.Duration `json:"validation_timeout"`   // 验证超时时间

	// 高级配置
	ValidateOnBorrow   bool          `json:"validate_on_borrow"`  // 借用时验证
	ValidateOnReturn   bool          `json:"validate_on_return"`  // 归还时验证
	ValidateWhileIdle  bool          `json:"validate_while_idle"` // 空闲时验证
	TestWhileIdle      bool          `json:"test_while_idle"`     // 空闲时测试
	ValidationQuery    string        `json:"validation_query"`    // 验证查询语句
	ValidationInterval time.Duration `json:"validation_interval"` // 验证间隔

	// 监控配置
	EnableMetrics       bool          `json:"enable_metrics"`        // 启用指标收集
	MetricsInterval     time.Duration `json:"metrics_interval"`      // 指标收集间隔
	EnableHealthCheck   bool          `json:"enable_health_check"`   // 启用健康检查
	HealthCheckInterval time.Duration `json:"health_check_interval"` // 健康检查间隔

	// 连接特定配置
	ConnectionConfig map[string]interface{} `json:"connection_config"` // 连接特定配置
}

// PoolStats 连接池统计信息
type PoolStats struct {
	PoolName  string    `json:"pool_name"` // 连接池名称
	Timestamp time.Time `json:"timestamp"` // 统计时间戳

	// 连接数量统计
	ActiveConnections int `json:"active_connections"` // 活跃连接数
	IdleConnections   int `json:"idle_connections"`   // 空闲连接数
	TotalConnections  int `json:"total_connections"`  // 总连接数
	PendingRequests   int `json:"pending_requests"`   // 等待连接的请求数

	// 生命周期统计
	ConnectionsCreated   int64 `json:"connections_created"`   // 创建的连接数
	ConnectionsDestroyed int64 `json:"connections_destroyed"` // 销毁的连接数
	ConnectionsReused    int64 `json:"connections_reused"`    // 重用的连接数
	ConnectionsBorrowed  int64 `json:"connections_borrowed"`  // 借用的连接数
	ConnectionsReturned  int64 `json:"connections_returned"`  // 归还的连接数

	// 性能统计
	AverageActiveTime float64 `json:"average_active_time"` // 平均活跃时间（毫秒）
	AverageIdleTime   float64 `json:"average_idle_time"`   // 平均空闲时间（毫秒）
	AverageWaitTime   float64 `json:"average_wait_time"`   // 平均等待时间（毫秒）
	MaxActiveTime     int64   `json:"max_active_time"`     // 最大活跃时间（毫秒）
	MaxIdleTime       int64   `json:"max_idle_time"`       // 最大空闲时间（毫秒）
	MaxWaitTime       int64   `json:"max_wait_time"`       // 最大等待时间（毫秒）

	// 错误统计
	ConnectionErrors int64 `json:"connection_errors"` // 连接错误数
	ValidationErrors int64 `json:"validation_errors"` // 验证错误数
	TimeoutErrors    int64 `json:"timeout_errors"`    // 超时错误数

	// 健康状态
	HealthStatus    string    `json:"health_status"`     // 健康状态: healthy, warning, critical
	HealthScore     float64   `json:"health_score"`      // 健康评分 (0-100)
	LastHealthCheck time.Time `json:"last_health_check"` // 最后健康检查时间

	// 配置信息
	MaxConnections int `json:"max_connections"` // 最大连接数配置
	MinConnections int `json:"min_connections"` // 最小连接数配置
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	ID                string                 `json:"id"`                 // 连接ID
	State             string                 `json:"state"`              // 连接状态: idle, active, validating, closed
	CreatedAt         time.Time              `json:"created_at"`         // 创建时间
	LastActiveAt      time.Time              `json:"last_active_at"`     // 最后活跃时间
	LastIdleAt        time.Time              `json:"last_idle_at"`       // 最后空闲时间
	UsageCount        int64                  `json:"usage_count"`        // 使用次数
	TotalActiveTime   time.Duration          `json:"total_active_time"`  // 总活跃时间
	TotalIdleTime     time.Duration          `json:"total_idle_time"`    // 总空闲时间
	LocalAddr         string                 `json:"local_addr"`         // 本地地址
	RemoteAddr        string                 `json:"remote_addr"`        // 远程地址
	Properties        map[string]interface{} `json:"properties"`         // 连接属性
	LastValidation    time.Time              `json:"last_validation"`    // 最后验证时间
	ValidationSuccess bool                   `json:"validation_success"` // 验证成功
	ErrorCount        int                    `json:"error_count"`        // 错误次数
	LastError         string                 `json:"last_error"`         // 最后错误信息
}

// HealthCheckerConfig 健康检查器配置
type HealthCheckerConfig struct {
	Name             string        `json:"name"`              // 健康检查器名称
	Type             string        `json:"type"`              // 检查类型: tcp, http, database, custom
	Enabled          bool          `json:"enabled"`           // 是否启用
	Interval         time.Duration `json:"interval"`          // 检查间隔
	Timeout          time.Duration `json:"timeout"`           // 超时时间
	RetryCount       int           `json:"retry_count"`       // 重试次数
	RetryInterval    time.Duration `json:"retry_interval"`    // 重试间隔
	FailureThreshold int           `json:"failure_threshold"` // 失败阈值
	SuccessThreshold int           `json:"success_threshold"` // 成功阈值

	// 检查目标配置
	Target          string            `json:"target"`           // 检查目标（URL、地址等）
	ExpectedStatus  []int             `json:"expected_status"`  // 期望的状态码
	ExpectedContent string            `json:"expected_content"` // 期望的内容
	Headers         map[string]string `json:"headers"`          // HTTP头部
	Method          string            `json:"method"`           // HTTP方法
	Body            string            `json:"body"`             // 请求体

	// TLS配置
	TLSConfig          *tls.Config `json:"-"`                    // TLS配置
	InsecureSkipVerify bool        `json:"insecure_skip_verify"` // 跳过TLS验证

	// 自定义配置
	CustomConfig map[string]interface{} `json:"custom_config"` // 自定义配置

	// 告警配置
	AlertOnFailure  bool     `json:"alert_on_failure"`  // 失败时告警
	AlertOnRecovery bool     `json:"alert_on_recovery"` // 恢复时告警
	AlertChannels   []string `json:"alert_channels"`    // 告警渠道
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	ID                   string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	HealthCheckID        string    `json:"health_check_id" gorm:"not null;type:varchar(36);index"`
	CheckerName          string    `json:"checker_name" gorm:"not null;size:255"`  // 检查器名称
	Target               string    `json:"target" gorm:"not null;size:500"`        // 检查目标
	Status               string    `json:"status" gorm:"not null;size:20"`         // 状态: healthy, unhealthy, unknown
	CheckedAt            time.Time `json:"checked_at" gorm:"not null"`             // 检查时间
	ResponseTime         int64     `json:"response_time" gorm:"default:0"`         // 响应时间
	StatusCode           int       `json:"status_code" gorm:"default:0"`           // 状态码
	Message              string    `json:"message" gorm:"type:text"`               // 状态消息
	Details              string    `json:"details" gorm:"type:text"`               // 详细信息
	Error                string    `json:"error" gorm:"type:text"`                 // 错误信息
	Metadata             JSONB     `json:"metadata" gorm:"type:jsonb"`             // 元数据
	ConsecutiveFailures  int       `json:"consecutive_failures" gorm:"default:0"`  // 连续失败次数
	ConsecutiveSuccesses int       `json:"consecutive_successes" gorm:"default:0"` // 连续成功次数
	LastSuccess          time.Time `json:"last_success"`                           // 最后成功时间
	LastFailure          time.Time `json:"last_failure"`                           // 最后失败时间
	CreatedAt            time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt            time.Time `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// BeforeCreate GORM钩子，创建前生成UUID
func (hcr *HealthCheckResult) BeforeCreate(tx *gorm.DB) error {
	if hcr.ID == "" {
		hcr.ID = uuid.New().String()
	}
	return nil
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	// 默认加密算法
	DefaultAlgorithm string `json:"default_algorithm"` // 默认算法: AES, RSA, etc.
	DefaultKeySize   int    `json:"default_key_size"`  // 默认密钥长度
	DefaultMode      string `json:"default_mode"`      // 默认模式: CBC, GCM, etc.
	DefaultPadding   string `json:"default_padding"`   // 默认填充: PKCS7, etc.

	// 密钥管理
	KeyStorePath        string        `json:"key_store_path"`        // 密钥存储路径
	KeyStorePassword    string        `json:"key_store_password"`    // 密钥存储密码
	KeyRotationInterval time.Duration `json:"key_rotation_interval"` // 密钥轮换间隔

	// 哈希配置
	DefaultHashAlgorithm string `json:"default_hash_algorithm"` // 默认哈希算法: SHA256, SHA512, etc.
	SaltLength           int    `json:"salt_length"`            // 盐值长度

	// 随机数配置
	RandomSource string `json:"random_source"` // 随机数源

	// 证书配置
	CertificatePath string `json:"certificate_path"` // 证书路径
	PrivateKeyPath  string `json:"private_key_path"` // 私钥路径
	CAPath          string `json:"ca_path"`          // CA证书路径

	// 安全策略
	MinKeySize          int      `json:"min_key_size"`         // 最小密钥长度
	AllowedAlgorithms   []string `json:"allowed_algorithms"`   // 允许的算法列表
	ForbiddenAlgorithms []string `json:"forbidden_algorithms"` // 禁止的算法列表
}

// EncryptionResult 加密结果
type EncryptionResult struct {
	Algorithm  string                 `json:"algorithm"`   // 使用的算法
	Mode       string                 `json:"mode"`        // 使用的模式
	KeyID      string                 `json:"key_id"`      // 密钥ID
	IV         []byte                 `json:"iv"`          // 初始化向量
	CipherText []byte                 `json:"cipher_text"` // 密文
	AuthTag    []byte                 `json:"auth_tag"`    // 认证标签（如果适用）
	Metadata   map[string]interface{} `json:"metadata"`    // 元数据
	Timestamp  time.Time              `json:"timestamp"`   // 加密时间
	Version    string                 `json:"version"`     // 版本信息
}

// DecryptionRequest 解密请求
type DecryptionRequest struct {
	Algorithm  string                 `json:"algorithm"`   // 算法
	Mode       string                 `json:"mode"`        // 模式
	KeyID      string                 `json:"key_id"`      // 密钥ID
	IV         []byte                 `json:"iv"`          // 初始化向量
	CipherText []byte                 `json:"cipher_text"` // 密文
	AuthTag    []byte                 `json:"auth_tag"`    // 认证标签
	Metadata   map[string]interface{} `json:"metadata"`    // 元数据
}

// HashResult 哈希结果
type HashResult struct {
	Algorithm  string    `json:"algorithm"`  // 哈希算法
	Hash       []byte    `json:"hash"`       // 哈希值
	Salt       []byte    `json:"salt"`       // 盐值
	Iterations int       `json:"iterations"` // 迭代次数（如PBKDF2）
	Timestamp  time.Time `json:"timestamp"`  // 计算时间
}

// DataConverterConfig 数据转换器配置
type DataConverterConfig struct {
	// 基础配置
	DefaultSourceEncoding string `json:"default_source_encoding"` // 默认源编码
	DefaultTargetEncoding string `json:"default_target_encoding"` // 默认目标编码
	DefaultDateFormat     string `json:"default_date_format"`     // 默认日期格式
	DefaultTimeZone       string `json:"default_time_zone"`       // 默认时区

	// 数值转换配置
	NumberPrecision    int    `json:"number_precision"`    // 数值精度
	DecimalSeparator   string `json:"decimal_separator"`   // 小数分隔符
	ThousandsSeparator string `json:"thousands_separator"` // 千位分隔符

	// 类型转换配置
	StrictTypeChecking  bool `json:"strict_type_checking"`  // 严格类型检查
	AutoTypeConversion  bool `json:"auto_type_conversion"`  // 自动类型转换
	TruncateLongStrings bool `json:"truncate_long_strings"` // 截断长字符串
	MaxStringLength     int  `json:"max_string_length"`     // 最大字符串长度

	// 错误处理配置
	IgnoreConversionErrors bool        `json:"ignore_conversion_errors"` // 忽略转换错误
	DefaultValueOnError    interface{} `json:"default_value_on_error"`   // 错误时的默认值
	LogConversionErrors    bool        `json:"log_conversion_errors"`    // 记录转换错误

	// 自定义转换规则
	CustomConverters map[string]interface{} `json:"custom_converters"` // 自定义转换器
	ConversionRules  map[string]interface{} `json:"conversion_rules"`  // 转换规则
}

// ConversionResult 转换结果
type ConversionResult struct {
	Success        bool                   `json:"success"`         // 是否成功
	SourceValue    interface{}            `json:"source_value"`    // 源值
	TargetValue    interface{}            `json:"target_value"`    // 目标值
	SourceType     string                 `json:"source_type"`     // 源类型
	TargetType     string                 `json:"target_type"`     // 目标类型
	ConversionPath []string               `json:"conversion_path"` // 转换路径
	Error          string                 `json:"error"`           // 错误信息
	Warnings       []string               `json:"warnings"`        // 警告信息
	Metadata       map[string]interface{} `json:"metadata"`        // 元数据
	ConversionTime time.Duration          `json:"conversion_time"` // 转换耗时
	Timestamp      time.Time              `json:"timestamp"`       // 转换时间
}
