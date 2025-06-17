/*
 * @module service/models/monitoring_service_models
 * @description 监控服务扩展相关模型定义，包含监控配置、指标快照、事件等核心数据结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 指标收集 -> 事件通知
 * @rules 确保监控服务模型的一致性和完整性
 * @dependencies time
 * @refs service/monitoring
 */

package models

import (
	"time"
)

// MonitoringServiceConfig 监控服务配置
type MonitoringServiceConfig struct {
	CollectionInterval   time.Duration          `json:"collection_interval"`    // 指标收集间隔
	HealthCheckInterval  time.Duration          `json:"health_check_interval"`  // 健康检查间隔
	AlertCheckInterval   time.Duration          `json:"alert_check_interval"`   // 告警检查间隔
	MetricsRetentionDays int                    `json:"metrics_retention_days"` // 指标保留天数
	EnabledMetrics       []string               `json:"enabled_metrics"`        // 启用的指标类型
	AlertRules           map[string]interface{} `json:"alert_rules"`            // 告警规则
	NotificationChannels []NotificationChannel  `json:"notification_channels"`  // 通知渠道
}

// MetricSnapshot 指标快照
type MetricSnapshot struct {
	MetricType   string                 `json:"metric_type"`
	Timestamp    time.Time              `json:"timestamp"`
	Value        interface{}            `json:"value"`
	Tags         map[string]string      `json:"tags"`
	Aggregations map[string]interface{} `json:"aggregations"`
	Trend        TrendInfo              `json:"trend"`
}

// TrendInfo 趋势信息
type TrendInfo struct {
	Direction    string  `json:"direction"`     // up, down, stable
	ChangeRate   float64 `json:"change_rate"`   // 变化率
	Confidence   float64 `json:"confidence"`    // 置信度
	PredictValue float64 `json:"predict_value"` // 预测值
}

// MonitoringEvent 监控事件
type MonitoringEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // metric_collected, alert_triggered, health_changed
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`   // 事件源
	Severity  string                 `json:"severity"` // info, warning, error, critical
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // email, webhook, sms
	Name     string                 `json:"name"`
	Config   map[string]interface{} `json:"config"`
	IsActive bool                   `json:"is_active"`
}

// 监控指标相关模型

// SystemMetricsSnapshot 系统指标快照 (避免与monitoring_models.go中的SystemMetrics冲突)
type SystemMetricsSnapshot struct {
	Timestamp           time.Time `json:"timestamp"`
	CPUUsage            float64   `json:"cpu_usage"`             // CPU使用率
	MemoryUsage         float64   `json:"memory_usage"`          // 内存使用率
	MemoryTotal         int64     `json:"memory_total"`          // 总内存
	MemoryUsed          int64     `json:"memory_used"`           // 已用内存
	DiskUsage           float64   `json:"disk_usage"`            // 磁盘使用率
	DiskTotal           int64     `json:"disk_total"`            // 总磁盘空间
	DiskUsed            int64     `json:"disk_used"`             // 已用磁盘空间
	NetworkInBytes      int64     `json:"network_in_bytes"`      // 网络入流量
	NetworkOutBytes     int64     `json:"network_out_bytes"`     // 网络出流量
	NetworkInPackets    int64     `json:"network_in_packets"`    // 网络入包数
	NetworkOutPackets   int64     `json:"network_out_packets"`   // 网络出包数
	LoadAverage1m       float64   `json:"load_average_1m"`       // 1分钟负载平均值
	LoadAverage5m       float64   `json:"load_average_5m"`       // 5分钟负载平均值
	LoadAverage15m      float64   `json:"load_average_15m"`      // 15分钟负载平均值
	ProcessCount        int       `json:"process_count"`         // 进程数
	ThreadCount         int       `json:"thread_count"`          // 线程数
	FileDescriptorCount int       `json:"file_descriptor_count"` // 文件描述符数量
	ConnectionCount     int       `json:"connection_count"`      // 连接数
}

// DataSourceMetrics 数据源指标
type DataSourceMetrics struct {
	DataSourceID      string    `json:"data_source_id"`
	Timestamp         time.Time `json:"timestamp"`
	ConnectionCount   int       `json:"connection_count"`   // 连接数
	ActiveConnections int       `json:"active_connections"` // 活跃连接数
	IdleConnections   int       `json:"idle_connections"`   // 空闲连接数
	QueueSize         int       `json:"queue_size"`         // 队列大小
	ResponseTime      float64   `json:"response_time"`      // 平均响应时间（毫秒）
	Throughput        float64   `json:"throughput"`         // 吞吐量（每秒操作数）
	ErrorRate         float64   `json:"error_rate"`         // 错误率
	SuccessRate       float64   `json:"success_rate"`       // 成功率
	LastSuccessTime   time.Time `json:"last_success_time"`  // 最后成功时间
	LastErrorTime     time.Time `json:"last_error_time"`    // 最后错误时间
	TotalRequests     int64     `json:"total_requests"`     // 总请求数
	SuccessRequests   int64     `json:"success_requests"`   // 成功请求数
	ErrorRequests     int64     `json:"error_requests"`     // 错误请求数
}

// SyncTaskMetrics 同步任务指标
type SyncTaskMetrics struct {
	TimeRange        string    `json:"time_range"`
	Timestamp        time.Time `json:"timestamp"`
	TotalTasks       int       `json:"total_tasks"`        // 总任务数
	RunningTasks     int       `json:"running_tasks"`      // 运行中任务数
	PendingTasks     int       `json:"pending_tasks"`      // 等待任务数
	CompletedTasks   int       `json:"completed_tasks"`    // 完成任务数
	FailedTasks      int       `json:"failed_tasks"`       // 失败任务数
	SuccessRate      float64   `json:"success_rate"`       // 成功率
	AvgExecutionTime float64   `json:"avg_execution_time"` // 平均执行时间（秒）
	TotalRecords     int64     `json:"total_records"`      // 总记录数
	ProcessedRecords int64     `json:"processed_records"`  // 已处理记录数
	ErrorRecords     int64     `json:"error_records"`      // 错误记录数
	ThroughputPerSec float64   `json:"throughput_per_sec"` // 每秒处理记录数
	DataVolumeBytes  int64     `json:"data_volume_bytes"`  // 数据量（字节）
	QueuedTasks      int       `json:"queued_tasks"`       // 排队任务数
	RetryTasks       int       `json:"retry_tasks"`        // 重试任务数
	TaskDistribution []string  `json:"task_distribution"`  // 任务类型分布
}

// HealthStatus 健康状态
type HealthStatus struct {
	Overall      string                      `json:"overall"` // overall, healthy, warning, critical
	Timestamp    time.Time                   `json:"timestamp"`
	Services     map[string]ServiceHealth    `json:"services"`     // 服务健康状态
	DataSources  map[string]DataSourceHealth `json:"data_sources"` // 数据源健康状态
	Dependencies map[string]DependencyHealth `json:"dependencies"` // 依赖服务健康状态
	Metrics      HealthMetrics               `json:"metrics"`      // 健康指标
	Issues       []HealthIssue               `json:"issues"`       // 健康问题
	LastChecked  time.Time                   `json:"last_checked"` // 最后检查时间
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`        // healthy, warning, critical, unknown
	ResponseTime float64   `json:"response_time"` // 响应时间（毫秒）
	Uptime       float64   `json:"uptime"`        // 运行时间（小时）
	ErrorRate    float64   `json:"error_rate"`    // 错误率
	LastChecked  time.Time `json:"last_checked"`  // 最后检查时间
	Message      string    `json:"message"`       // 状态描述
	Details      string    `json:"details"`       // 详细信息
}

// DataSourceHealth 数据源健康状态
type DataSourceHealth struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`            // healthy, warning, critical, unknown
	ConnectionOK    bool                   `json:"connection_ok"`     // 连接是否正常
	ResponseTime    float64                `json:"response_time"`     // 响应时间（毫秒）
	LastSyncTime    time.Time              `json:"last_sync_time"`    // 最后同步时间
	LastSuccessTime time.Time              `json:"last_success_time"` // 最后成功时间
	LastErrorTime   time.Time              `json:"last_error_time"`   // 最后错误时间
	ErrorMessage    string                 `json:"error_message"`     // 错误信息
	LastChecked     time.Time              `json:"last_checked"`      // 最后检查时间
	Metrics         map[string]interface{} `json:"metrics"`           // 相关指标
}

// DependencyHealth 依赖服务健康状态
type DependencyHealth struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`          // database, redis, kafka, http_service
	Status       string    `json:"status"`        // healthy, warning, critical, unknown
	Available    bool      `json:"available"`     // 是否可用
	ResponseTime float64   `json:"response_time"` // 响应时间（毫秒）
	LastChecked  time.Time `json:"last_checked"`  // 最后检查时间
	Message      string    `json:"message"`       // 状态描述
	Details      string    `json:"details"`       // 详细信息
}

// HealthMetrics 健康指标
type HealthMetrics struct {
	OverallScore      float64 `json:"overall_score"`      // 总体健康评分 (0-100)
	ServiceScore      float64 `json:"service_score"`      // 服务健康评分
	DataSourceScore   float64 `json:"data_source_score"`  // 数据源健康评分
	DependencyScore   float64 `json:"dependency_score"`   // 依赖服务健康评分
	PerformanceScore  float64 `json:"performance_score"`  // 性能评分
	AvailabilityScore float64 `json:"availability_score"` // 可用性评分
}

// HealthIssue 健康问题
type HealthIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // service, data_source, dependency, performance
	Severity    string    `json:"severity"`    // low, medium, high, critical
	Title       string    `json:"title"`       // 问题标题
	Description string    `json:"description"` // 问题描述
	Source      string    `json:"source"`      // 问题来源
	DetectedAt  time.Time `json:"detected_at"` // 检测时间
	Suggestion  string    `json:"suggestion"`  // 建议解决方案
}

// Alert 告警
type Alert struct {
	ID                string                 `json:"id"`
	RuleName          string                 `json:"rule_name"`          // 告警规则名称
	MetricName        string                 `json:"metric_name"`        // 指标名称
	Severity          string                 `json:"severity"`           // info, warning, error, critical
	Status            string                 `json:"status"`             // firing, resolved
	Value             float64                `json:"value"`              // 触发值
	Threshold         float64                `json:"threshold"`          // 阈值
	Condition         string                 `json:"condition"`          // 触发条件
	Description       string                 `json:"description"`        // 告警描述
	Labels            map[string]string      `json:"labels"`             // 标签
	Annotations       map[string]string      `json:"annotations"`        // 注解
	StartsAt          time.Time              `json:"starts_at"`          // 开始时间
	EndsAt            *time.Time             `json:"ends_at"`            // 结束时间
	LastEvaluated     time.Time              `json:"last_evaluated"`     // 最后评估时间
	EvaluationInfo    map[string]interface{} `json:"evaluation_info"`    // 评估信息
	NotificationsSent int                    `json:"notifications_sent"` // 已发送通知数
}
