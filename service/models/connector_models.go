/*
 * @module service/models/connector_models
 * @description 客户端连接器相关模型定义，包含Kafka、MQTT、Redis等连接器的配置和消息结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 连接器配置 -> 消息处理
 * @rules 确保连接器模型的一致性和完整性
 * @dependencies time
 * @refs client/connectors
 */

package models

import (
	"time"
)

// Kafka相关模型

// KafkaConfig Kafka配置信息
type KafkaConfig struct {
	Brokers           []string          `json:"brokers"`            // Kafka broker地址列表
	GroupID           string            `json:"group_id"`           // 消费者组ID
	Topics            []string          `json:"topics"`             // 订阅的主题列表
	SecurityConfig    *SecurityConfig   `json:"security_config"`    // 安全配置
	ProducerConfig    *ProducerConfig   `json:"producer_config"`    // 生产者配置
	ConsumerConfig    *ConsumerConfig   `json:"consumer_config"`    // 消费者配置
	ConnectionTimeout time.Duration     `json:"connection_timeout"` // 连接超时时间
	RetryAttempts     int               `json:"retry_attempts"`     // 重试次数
	CustomHeaders     map[string]string `json:"custom_headers"`     // 自定义消息头
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableSASL bool   `json:"enable_sasl"` // 是否启用SASL认证
	Username   string `json:"username"`    // SASL用户名
	Password   string `json:"password"`    // SASL密码
	EnableTLS  bool   `json:"enable_tls"`  // 是否启用TLS
	CertFile   string `json:"cert_file"`   // TLS证书文件
	KeyFile    string `json:"key_file"`    // TLS密钥文件
	CAFile     string `json:"ca_file"`     // CA证书文件
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	BatchSize    int           `json:"batch_size"`    // 批量大小
	BatchTimeout time.Duration `json:"batch_timeout"` // 批量超时时间
	RequiredAcks int           `json:"required_acks"` // 确认模式
	Compression  string        `json:"compression"`   // 压缩算法
	MaxRetries   int           `json:"max_retries"`   // 最大重试次数
	Async        bool          `json:"async"`         // 是否异步发送
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	MinBytes          int           `json:"min_bytes"`          // 最小字节数
	MaxBytes          int           `json:"max_bytes"`          // 最大字节数
	MaxWait           time.Duration `json:"max_wait"`           // 最大等待时间
	CommitInterval    time.Duration `json:"commit_interval"`    // 提交间隔
	StartOffset       int64         `json:"start_offset"`       // 起始偏移量
	AutoCommit        bool          `json:"auto_commit"`        // 是否自动提交
	SessionTimeout    time.Duration `json:"session_timeout"`    // 会话超时时间
	HeartbeatInterval time.Duration `json:"heartbeat_interval"` // 心跳间隔
}

// KafkaMessage Kafka消息结构体
type KafkaMessage struct {
	Topic     string            `json:"topic"`     // 主题
	Key       string            `json:"key"`       // 消息键
	Value     interface{}       `json:"value"`     // 消息值
	Headers   map[string]string `json:"headers"`   // 消息头
	Partition int               `json:"partition"` // 分区
	Offset    int64             `json:"offset"`    // 偏移量
	Timestamp time.Time         `json:"timestamp"` // 时间戳
}

// MessageHandler 消息处理函数类型
type MessageHandler func(*KafkaMessage) error

// MQTT相关模型

// MQTTConfig MQTT配置信息
type MQTTConfig struct {
	Broker               string            `json:"broker"`                 // MQTT broker地址
	ClientID             string            `json:"client_id"`              // 客户端ID
	Username             string            `json:"username"`               // 用户名
	Password             string            `json:"password"`               // 密码
	CleanSession         bool              `json:"clean_session"`          // 清理会话
	KeepAlive            time.Duration     `json:"keep_alive"`             // 保持连接时间
	Topics               []string          `json:"topics"`                 // 订阅主题列表
	QoS                  map[string]byte   `json:"qos"`                    // 主题对应的QoS级别
	WillConfig           *WillConfig       `json:"will_config"`            // 遗嘱配置
	TLSConfig            *TLSConfigMQTT    `json:"tls_config"`             // TLS配置
	CustomHeaders        map[string]string `json:"custom_headers"`         // 自定义头部
	AutoReconnect        bool              `json:"auto_reconnect"`         // 自动重连
	MaxReconnectInterval time.Duration     `json:"max_reconnect_interval"` // 最大重连间隔
}

// WillConfig 遗嘱配置
type WillConfig struct {
	Topic   string `json:"topic"`   // 遗嘱主题
	Payload string `json:"payload"` // 遗嘱消息
	QoS     byte   `json:"qos"`     // 遗嘱QoS
	Retain  bool   `json:"retain"`  // 遗嘱保留
}

// TLSConfigMQTT MQTT TLS配置
type TLSConfigMQTT struct {
	EnableTLS  bool   `json:"enable_tls"`  // 是否启用TLS
	CertFile   string `json:"cert_file"`   // 证书文件
	KeyFile    string `json:"key_file"`    // 密钥文件
	CAFile     string `json:"ca_file"`     // CA文件
	SkipVerify bool   `json:"skip_verify"` // 跳过证书验证
}

// MQTTMessage MQTT消息结构体
type MQTTMessage struct {
	Topic     string    `json:"topic"`      // 主题
	Payload   []byte    `json:"payload"`    // 消息载荷
	QoS       byte      `json:"qos"`        // 服务质量
	Retained  bool      `json:"retained"`   // 是否保留
	MessageID uint16    `json:"message_id"` // 消息ID
	Timestamp time.Time `json:"timestamp"`  // 时间戳
}

// MQTTMessageHandler MQTT消息处理函数类型
type MQTTMessageHandler func(*MQTTMessage) error

// Redis相关模型

// RedisConfig Redis配置信息
type RedisConfig struct {
	Address        string          `json:"address"`         // Redis地址
	Password       string          `json:"password"`        // 密码
	Database       int             `json:"database"`        // 数据库号
	PoolSize       int             `json:"pool_size"`       // 连接池大小
	MinIdleConns   int             `json:"min_idle_conns"`  // 最小空闲连接
	MaxConnAge     time.Duration   `json:"max_conn_age"`    // 最大连接时间
	PoolTimeout    time.Duration   `json:"pool_timeout"`    // 连接池超时
	IdleTimeout    time.Duration   `json:"idle_timeout"`    // 空闲超时
	IdleCheckFreq  time.Duration   `json:"idle_check_freq"` // 空闲检查频率
	DialTimeout    time.Duration   `json:"dial_timeout"`    // 拨号超时
	ReadTimeout    time.Duration   `json:"read_timeout"`    // 读取超时
	WriteTimeout   time.Duration   `json:"write_timeout"`   // 写入超时
	EnableTLS      bool            `json:"enable_tls"`      // 是否启用TLS
	TLSConfig      *TLSConfigRedis `json:"tls_config"`      // TLS配置
	ClusterMode    bool            `json:"cluster_mode"`    // 集群模式
	ClusterAddrs   []string        `json:"cluster_addrs"`   // 集群地址
	EnablePipeline bool            `json:"enable_pipeline"` // 启用管道
	PipelineSize   int             `json:"pipeline_size"`   // 管道大小
}

// TLSConfigRedis Redis TLS配置
type TLSConfigRedis struct {
	CertFile   string `json:"cert_file"`   // 证书文件
	KeyFile    string `json:"key_file"`    // 密钥文件
	CAFile     string `json:"ca_file"`     // CA文件
	SkipVerify bool   `json:"skip_verify"` // 跳过证书验证
}

// RedisMessage Redis消息结构体
type RedisMessage struct {
	Channel   string    `json:"channel"`   // 频道
	Pattern   string    `json:"pattern"`   // 模式（如果是模式订阅）
	Payload   string    `json:"payload"`   // 消息载荷
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// RedisMessageHandler Redis消息处理函数类型
type RedisMessageHandler func(*RedisMessage) error

// 通用连接器统计信息

// ConnectorStatistics 连接器统计信息
type ConnectorStatistics struct {
	ConnectorType     string                 `json:"connector_type"`     // 连接器类型
	ConnectionStatus  string                 `json:"connection_status"`  // 连接状态
	MessagesProduced  int64                  `json:"messages_produced"`  // 已生产消息数
	MessagesConsumed  int64                  `json:"messages_consumed"`  // 已消费消息数
	ErrorCount        int64                  `json:"error_count"`        // 错误计数
	LastError         string                 `json:"last_error"`         // 最后错误
	Uptime            time.Duration          `json:"uptime"`             // 运行时间
	LastActivity      time.Time              `json:"last_activity"`      // 最后活动时间
	Throughput        int64                  `json:"throughput"`         // 吞吐量(消息/秒)
	AdditionalMetrics map[string]interface{} `json:"additional_metrics"` // 额外指标
}

// MQTT连接器扩展模型

// MQTTConnectorExtended MQTT连接器扩展信息
type MQTTConnectorExtended struct {
	Config           *MQTTConfig                 `json:"config"`
	ConnectionInfo   *MQTTConnectionInfo         `json:"connection_info"`
	SubscriptionInfo map[string]MQTTSubscription `json:"subscription_info"`
	PublishStats     *MQTTPublishStats           `json:"publish_stats"`
	ConsumeStats     *MQTTConsumeStats           `json:"consume_stats"`
	LastMessages     []*MQTTMessage              `json:"last_messages"`
	ErrorHistory     []MQTTError                 `json:"error_history"`
}

// MQTTConnectionInfo MQTT连接信息
type MQTTConnectionInfo struct {
	IsConnected       bool          `json:"is_connected"`
	ConnectedAt       time.Time     `json:"connected_at"`
	LastPingAt        time.Time     `json:"last_ping_at"`
	LastPongAt        time.Time     `json:"last_pong_at"`
	ReconnectCount    int           `json:"reconnect_count"`
	NextReconnectAt   *time.Time    `json:"next_reconnect_at"`
	CurrentRetryDelay time.Duration `json:"current_retry_delay"`
	ServerVersion     string        `json:"server_version"`
	SessionPresent    bool          `json:"session_present"`
}

// MQTTSubscription MQTT订阅信息
type MQTTSubscription struct {
	Topic         string    `json:"topic"`
	QoS           byte      `json:"qos"`
	SubscribedAt  time.Time `json:"subscribed_at"`
	MessageCount  int64     `json:"message_count"`
	LastMessageAt time.Time `json:"last_message_at"`
	IsActive      bool      `json:"is_active"`
	HandlerType   string    `json:"handler_type"`
}

// MQTTPublishStats MQTT发布统计
type MQTTPublishStats struct {
	TotalPublished    int64     `json:"total_published"`
	SuccessfulPublish int64     `json:"successful_publish"`
	FailedPublish     int64     `json:"failed_publish"`
	QoS0Messages      int64     `json:"qos0_messages"`
	QoS1Messages      int64     `json:"qos1_messages"`
	QoS2Messages      int64     `json:"qos2_messages"`
	RetainedMessages  int64     `json:"retained_messages"`
	AverageLatency    float64   `json:"average_latency"`
	MaxLatency        float64   `json:"max_latency"`
	LastPublishAt     time.Time `json:"last_publish_at"`
}

// MQTTConsumeStats MQTT消费统计
type MQTTConsumeStats struct {
	TotalReceived      int64     `json:"total_received"`
	ProcessedMessages  int64     `json:"processed_messages"`
	FailedMessages     int64     `json:"failed_messages"`
	QoS0Received       int64     `json:"qos0_received"`
	QoS1Received       int64     `json:"qos1_received"`
	QoS2Received       int64     `json:"qos2_received"`
	RetainedReceived   int64     `json:"retained_received"`
	AverageProcessTime float64   `json:"average_process_time"`
	MaxProcessTime     float64   `json:"max_process_time"`
	LastReceiveAt      time.Time `json:"last_receive_at"`
}

// MQTTError MQTT错误信息
type MQTTError struct {
	Timestamp time.Time              `json:"timestamp"`
	ErrorType string                 `json:"error_type"` // connection, publish, subscribe, process
	ErrorCode string                 `json:"error_code"`
	Message   string                 `json:"message"`
	Topic     string                 `json:"topic"`    // 如果错误与特定主题相关
	Severity  string                 `json:"severity"` // low, medium, high, critical
	Context   map[string]interface{} `json:"context"`
}

// Redis连接器扩展模型

// RedisConnectorExtended Redis连接器扩展信息
type RedisConnectorExtended struct {
	Config           *RedisConfig                 `json:"config"`
	ConnectionInfo   *RedisConnectionInfo         `json:"connection_info"`
	PoolInfo         *RedisPoolInfo               `json:"pool_info"`
	CommandStats     *RedisCommandStats           `json:"command_stats"`
	PubSubStats      *RedisPubSubStats            `json:"pubsub_stats"`
	SubscriptionInfo map[string]RedisSubscription `json:"subscription_info"`
	CacheStats       *RedisCacheStats             `json:"cache_stats"`
	ErrorHistory     []RedisError                 `json:"error_history"`
}

// RedisConnectionInfo Redis连接信息
type RedisConnectionInfo struct {
	IsConnected      bool      `json:"is_connected"`
	ConnectedAt      time.Time `json:"connected_at"`
	LastPingAt       time.Time `json:"last_ping_at"`
	RedisVersion     string    `json:"redis_version"`
	ServerMode       string    `json:"server_mode"` // standalone, cluster, sentinel
	MemoryUsage      int64     `json:"memory_usage"`
	UptimeSeconds    int64     `json:"uptime_seconds"`
	ConnectedClients int       `json:"connected_clients"`
	BlockedClients   int       `json:"blocked_clients"`
	Role             string    `json:"role"` // master, slave
}

// RedisPoolInfo Redis连接池信息
type RedisPoolInfo struct {
	PoolSize      int   `json:"pool_size"`
	ActiveConns   int   `json:"active_conns"`
	IdleConns     int   `json:"idle_conns"`
	WaitingCount  int   `json:"waiting_count"`
	StaleConns    int   `json:"stale_conns"`
	TotalHits     int64 `json:"total_hits"`
	TotalMisses   int64 `json:"total_misses"`
	TotalTimeouts int64 `json:"total_timeouts"`
	CreatedConns  int64 `json:"created_conns"`
	ClosedConns   int64 `json:"closed_conns"`
}

// RedisCommandStats Redis命令统计
type RedisCommandStats struct {
	TotalCommands       int64              `json:"total_commands"`
	SuccessfulCommands  int64              `json:"successful_commands"`
	FailedCommands      int64              `json:"failed_commands"`
	AverageLatency      float64            `json:"average_latency"`
	MaxLatency          float64            `json:"max_latency"`
	CommandDistribution map[string]int64   `json:"command_distribution"`
	SlowCommands        []RedisSlowCommand `json:"slow_commands"`
	LastCommandAt       time.Time          `json:"last_command_at"`
}

// RedisSlowCommand Redis慢命令记录
type RedisSlowCommand struct {
	Command    string    `json:"command"`
	Args       []string  `json:"args"`
	Duration   float64   `json:"duration"`
	Timestamp  time.Time `json:"timestamp"`
	ClientAddr string    `json:"client_addr"`
	ClientName string    `json:"client_name"`
}

// RedisPubSubStats Redis发布订阅统计
type RedisPubSubStats struct {
	TotalPublished     int64     `json:"total_published"`
	TotalReceived      int64     `json:"total_received"`
	ActiveChannels     int       `json:"active_channels"`
	PatternChannels    int       `json:"pattern_channels"`
	SubscribedChannels []string  `json:"subscribed_channels"`
	PublishLatency     float64   `json:"publish_latency"`
	ReceiveLatency     float64   `json:"receive_latency"`
	LastPublishAt      time.Time `json:"last_publish_at"`
	LastReceiveAt      time.Time `json:"last_receive_at"`
}

// RedisSubscription Redis订阅信息
type RedisSubscription struct {
	Channel       string    `json:"channel"`
	IsPattern     bool      `json:"is_pattern"`
	SubscribedAt  time.Time `json:"subscribed_at"`
	MessageCount  int64     `json:"message_count"`
	LastMessageAt time.Time `json:"last_message_at"`
	IsActive      bool      `json:"is_active"`
	HandlerType   string    `json:"handler_type"`
}

// RedisCacheStats Redis缓存统计
type RedisCacheStats struct {
	HitCount           int64   `json:"hit_count"`
	MissCount          int64   `json:"miss_count"`
	HitRate            float64 `json:"hit_rate"`
	MissRate           float64 `json:"miss_rate"`
	EvictedKeys        int64   `json:"evicted_keys"`
	ExpiredKeys        int64   `json:"expired_keys"`
	TotalKeys          int64   `json:"total_keys"`
	MemoryUsage        int64   `json:"memory_usage"`
	MemoryPeak         int64   `json:"memory_peak"`
	FragmentationRatio float64 `json:"fragmentation_ratio"`
}

// RedisError Redis错误信息
type RedisError struct {
	Timestamp time.Time              `json:"timestamp"`
	ErrorType string                 `json:"error_type"` // connection, command, timeout, pool
	ErrorCode string                 `json:"error_code"`
	Message   string                 `json:"message"`
	Command   string                 `json:"command"`  // 如果错误与特定命令相关
	Key       string                 `json:"key"`      // 如果错误与特定键相关
	Severity  string                 `json:"severity"` // low, medium, high, critical
	Context   map[string]interface{} `json:"context"`
}

// 通用连接器健康检查模型

// ConnectorHealthCheck 连接器健康检查
type ConnectorHealthCheck struct {
	ConnectorID     string              `json:"connector_id"`
	ConnectorType   string              `json:"connector_type"`
	CheckTime       time.Time           `json:"check_time"`
	HealthStatus    string              `json:"health_status"` // healthy, warning, critical, unknown
	HealthScore     float64             `json:"health_score"`  // 0-100
	CheckResults    []HealthCheckResult `json:"check_results"`
	Recommendations []string            `json:"recommendations"`
	NextCheckTime   time.Time           `json:"next_check_time"`
}

// ConnectorHealthCheckResult 连接器健康检查结果 (避免与其他包冲突)
type ConnectorHealthCheckResult struct {
	CheckName    string                 `json:"check_name"`
	CheckType    string                 `json:"check_type"` // connectivity, performance, resource
	Status       string                 `json:"status"`     // pass, warn, fail
	Message      string                 `json:"message"`
	Value        interface{}            `json:"value"`
	Threshold    interface{}            `json:"threshold"`
	ResponseTime time.Duration          `json:"response_time"`
	Details      map[string]interface{} `json:"details"`
}
