/*
 * @module service/models/monitoring_models
 * @description 监控相关模型定义，包含告警规则、监控指标、健康检查等
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 监控配置 -> 指标收集 -> 告警检测 -> 通知发送
 * @rules 确保模型字段的完整性和业务规则约束
 * @dependencies gorm.io/gorm, time, github.com/google/uuid
 * @refs ai_docs/patch_basic_library_process.md
 */

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MonitoringMetric 监控指标模型
type MonitoringMetric struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	MetricName string    `json:"metric_name" gorm:"not null;size:255;index"` // 指标名称
	MetricType string    `json:"metric_type" gorm:"not null;size:50"`        // 指标类型：counter, gauge, histogram, summary
	Category   string    `json:"category" gorm:"not null;size:50"`           // 指标类别：system, business, application
	Source     string    `json:"source" gorm:"not null;size:100"`            // 指标来源
	Value      float64   `json:"value" gorm:"not null"`                      // 指标值
	Unit       string    `json:"unit" gorm:"size:20"`                        // 单位
	Labels     JSONB     `json:"labels,omitempty" gorm:"type:jsonb"`         // 标签
	Timestamp  time.Time `json:"timestamp" gorm:"not null;index"`            // 时间戳
	TTL        int       `json:"ttl" gorm:"default:86400"`                   // 生存时间（秒）
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// AlertRule 告警规则模型
type AlertRule struct {
	ID                   string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name                 string     `json:"name" gorm:"not null;size:255"`                 // 规则名称
	Description          string     `json:"description" gorm:"size:1000"`                  // 规则描述
	MetricName           string     `json:"metric_name" gorm:"not null;size:255;index"`    // 监控指标名称
	Condition            JSONB      `json:"condition" gorm:"type:jsonb;not null"`          // 告警条件
	Severity             string     `json:"severity" gorm:"not null;size:20"`              // 严重级别：info, warning, error, critical
	Threshold            float64    `json:"threshold" gorm:"not null"`                     // 阈值
	Operator             string     `json:"operator" gorm:"not null;size:10"`              // 操作符：>, <, >=, <=, ==, !=
	EvaluationWindow     int        `json:"evaluation_window" gorm:"not null;default:300"` // 评估窗口（秒）
	NotificationChannels JSONB      `json:"notification_channels" gorm:"type:jsonb"`       // 通知渠道
	IsEnabled            bool       `json:"is_enabled" gorm:"not null;default:true"`       // 是否启用
	LastTriggered        *time.Time `json:"last_triggered,omitempty"`                      // 最后触发时间
	TriggerCount         int64      `json:"trigger_count" gorm:"default:0"`                // 触发次数
	CreatedAt            time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt            time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy            string     `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedBy            string     `json:"updated_by" gorm:"not null;default:'system';size:100"`

	// 关联关系
	AlertInstances []AlertInstance `json:"alert_instances,omitempty" gorm:"foreignKey:RuleID"`
}

// AlertInstance 告警实例模型
type AlertInstance struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RuleID      string     `json:"rule_id" gorm:"not null;type:varchar(36);index"` // 告警规则ID
	Status      string     `json:"status" gorm:"not null;size:20"`                 // 状态：firing, resolved, suppressed
	StartsAt    time.Time  `json:"starts_at" gorm:"not null"`                      // 告警开始时间
	EndsAt      *time.Time `json:"ends_at,omitempty"`                              // 告警结束时间
	Value       float64    `json:"value" gorm:"not null"`                          // 触发值
	Labels      JSONB      `json:"labels,omitempty" gorm:"type:jsonb"`             // 标签
	Annotations JSONB      `json:"annotations,omitempty" gorm:"type:jsonb"`        // 注解
	Fingerprint string     `json:"fingerprint" gorm:"not null;size:64;unique"`     // 指纹（用于去重）
	CreatedAt   time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	AlertRule     AlertRule           `json:"alert_rule,omitempty" gorm:"foreignKey:RuleID"`
	Notifications []AlertNotification `json:"notifications,omitempty" gorm:"foreignKey:AlertInstanceID"`
}

// AlertNotification 告警通知模型
type AlertNotification struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	AlertInstanceID string     `json:"alert_instance_id" gorm:"not null;type:varchar(36);index"` // 告警实例ID
	Channel         string     `json:"channel" gorm:"not null;size:50"`                          // 通知渠道：email, sms, webhook, slack, dingtalk
	Recipient       string     `json:"recipient" gorm:"not null;size:255"`                       // 接收者
	Subject         string     `json:"subject" gorm:"size:500"`                                  // 主题
	Content         string     `json:"content" gorm:"type:text"`                                 // 内容
	Status          string     `json:"status" gorm:"not null;size:20;default:'pending'"`         // 状态：pending, sent, failed, retrying
	RetryCount      int        `json:"retry_count" gorm:"default:0"`                             // 重试次数
	SentAt          *time.Time `json:"sent_at,omitempty"`                                        // 发送时间
	ErrorMessage    string     `json:"error_message,omitempty" gorm:"type:text"`                 // 错误信息
	CreatedAt       time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联关系
	AlertInstance AlertInstance `json:"alert_instance,omitempty" gorm:"foreignKey:AlertInstanceID"`
}

// HealthCheck 健康检查模型
type HealthCheck struct {
	ID            string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name          string     `json:"name" gorm:"not null;size:255"`                // 检查名称
	Type          string     `json:"type" gorm:"not null;size:50"`                 // 检查类型：http, tcp, database, service
	Target        string     `json:"target" gorm:"not null;size:500"`              // 检查目标
	Config        JSONB      `json:"config" gorm:"type:jsonb"`                     // 检查配置
	Interval      int        `json:"interval" gorm:"not null;default:60"`          // 检查间隔（秒）
	Timeout       int        `json:"timeout" gorm:"not null;default:30"`           // 超时时间（秒）
	IsEnabled     bool       `json:"is_enabled" gorm:"not null;default:true"`      // 是否启用
	LastCheckTime *time.Time `json:"last_check_time,omitempty"`                    // 最后检查时间
	LastStatus    string     `json:"last_status" gorm:"size:20;default:'unknown'"` // 最后状态：healthy, unhealthy, unknown
	FailureCount  int        `json:"failure_count" gorm:"default:0"`               // 连续失败次数
	CreatedAt     time.Time  `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	CreatedBy     string     `json:"created_by" gorm:"not null;default:'system';size:100"`
	UpdatedBy     string     `json:"updated_by" gorm:"not null;default:'system';size:100"`

	// 关联关系
	HealthCheckResults []HealthCheckResult `json:"health_check_results,omitempty" gorm:"foreignKey:HealthCheckID"`
}

// SystemMetrics 系统指标模型
type SystemMetrics struct {
	ID                  string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	HostName            string    `json:"host_name" gorm:"not null;size:255;index"` // 主机名
	CPUUsage            float64   `json:"cpu_usage" gorm:"default:0"`               // CPU使用率
	MemoryUsage         float64   `json:"memory_usage" gorm:"default:0"`            // 内存使用率
	DiskUsage           float64   `json:"disk_usage" gorm:"default:0"`              // 磁盘使用率
	NetworkIn           int64     `json:"network_in" gorm:"default:0"`              // 网络入流量（字节）
	NetworkOut          int64     `json:"network_out" gorm:"default:0"`             // 网络出流量（字节）
	LoadAverage         float64   `json:"load_average" gorm:"default:0"`            // 负载平均值
	ProcessCount        int       `json:"process_count" gorm:"default:0"`           // 进程数
	ConnectionCount     int       `json:"connection_count" gorm:"default:0"`        // 连接数
	DatabaseConnections int       `json:"database_connections" gorm:"default:0"`    // 数据库连接数
	ActiveSessions      int       `json:"active_sessions" gorm:"default:0"`         // 活跃会话数
	QueueLength         int       `json:"queue_length" gorm:"default:0"`            // 队列长度
	ErrorRate           float64   `json:"error_rate" gorm:"default:0"`              // 错误率
	ResponseTime        float64   `json:"response_time" gorm:"default:0"`           // 平均响应时间
	Timestamp           time.Time `json:"timestamp" gorm:"not null;index"`          // 时间戳
	CreatedAt           time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// PerformanceSnapshot 性能快照模型
type PerformanceSnapshot struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SnapshotType    string    `json:"snapshot_type" gorm:"not null;size:50"`       // 快照类型：hourly, daily, weekly
	StartTime       time.Time `json:"start_time" gorm:"not null;index"`            // 开始时间
	EndTime         time.Time `json:"end_time" gorm:"not null"`                    // 结束时间
	MetricsSummary  JSONB     `json:"metrics_summary" gorm:"type:jsonb;not null"`  // 指标摘要
	AlertsSummary   JSONB     `json:"alerts_summary" gorm:"type:jsonb"`            // 告警摘要
	HealthSummary   JSONB     `json:"health_summary" gorm:"type:jsonb"`            // 健康状况摘要
	TrendData       JSONB     `json:"trend_data,omitempty" gorm:"type:jsonb"`      // 趋势数据
	TopIssues       JSONB     `json:"top_issues,omitempty" gorm:"type:jsonb"`      // 主要问题
	Recommendations JSONB     `json:"recommendations,omitempty" gorm:"type:jsonb"` // 建议
	CreatedAt       time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// BeforeCreate 钩子
func (mm *MonitoringMetric) BeforeCreate(tx *gorm.DB) error {
	if mm.ID == "" {
		mm.ID = uuid.New().String()
	}
	return nil
}

func (ar *AlertRule) BeforeCreate(tx *gorm.DB) error {
	if ar.ID == "" {
		ar.ID = uuid.New().String()
	}
	return nil
}

func (ai *AlertInstance) BeforeCreate(tx *gorm.DB) error {
	if ai.ID == "" {
		ai.ID = uuid.New().String()
	}
	return nil
}

func (an *AlertNotification) BeforeCreate(tx *gorm.DB) error {
	if an.ID == "" {
		an.ID = uuid.New().String()
	}
	return nil
}

func (hc *HealthCheck) BeforeCreate(tx *gorm.DB) error {
	if hc.ID == "" {
		hc.ID = uuid.New().String()
	}
	return nil
}



func (sm *SystemMetrics) BeforeCreate(tx *gorm.DB) error {
	if sm.ID == "" {
		sm.ID = uuid.New().String()
	}
	return nil
}

func (ps *PerformanceSnapshot) BeforeCreate(tx *gorm.DB) error {
	if ps.ID == "" {
		ps.ID = uuid.New().String()
	}
	return nil
}

// 业务方法

// GetActiveAlertRules 获取活跃的告警规则
func GetActiveAlertRules(db *gorm.DB) ([]AlertRule, error) {
	var rules []AlertRule
	err := db.Where("is_enabled = ?", true).Find(&rules).Error
	return rules, err
}

// GetFiringAlerts 获取正在触发的告警
func GetFiringAlerts(db *gorm.DB) ([]AlertInstance, error) {
	var alerts []AlertInstance
	err := db.Where("status = ?", "firing").Find(&alerts).Error
	return alerts, err
}

// GetHealthChecksByStatus 根据状态获取健康检查
func GetHealthChecksByStatus(db *gorm.DB, status string) ([]HealthCheck, error) {
	var checks []HealthCheck
	err := db.Where("last_status = ? AND is_enabled = ?", status, true).Find(&checks).Error
	return checks, err
}

// GetLatestMetrics 获取最新的指标数据
func GetLatestMetrics(db *gorm.DB, metricName string, limit int) ([]MonitoringMetric, error) {
	var metrics []MonitoringMetric
	err := db.Where("metric_name = ?", metricName).
		Order("timestamp DESC").
		Limit(limit).
		Find(&metrics).Error
	return metrics, err
}

// UpdateAlertRuleTrigger 更新告警规则触发信息
func (ar *AlertRule) UpdateAlertRuleTrigger(db *gorm.DB) error {
	now := time.Now()
	return db.Model(ar).Updates(map[string]interface{}{
		"last_triggered": &now,
		"trigger_count":  gorm.Expr("trigger_count + 1"),
		"updated_at":     now,
	}).Error
}

// UpdateHealthCheckResult 更新健康检查结果
func (hc *HealthCheck) UpdateHealthCheckResult(db *gorm.DB, status string, failureCount int) error {
	now := time.Now()
	return db.Model(hc).Updates(map[string]interface{}{
		"last_check_time": &now,
		"last_status":     status,
		"failure_count":   failureCount,
		"updated_at":      now,
	}).Error
}

// TableName 指定表名
func (MonitoringMetric) TableName() string {
	return "monitoring_metrics"
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

func (AlertInstance) TableName() string {
	return "alert_instances"
}

func (AlertNotification) TableName() string {
	return "alert_notifications"
}

func (HealthCheck) TableName() string {
	return "health_checks"
}

func (HealthCheckResult) TableName() string {
	return "health_check_results"
}

func (SystemMetrics) TableName() string {
	return "system_metrics"
}

func (PerformanceSnapshot) TableName() string {
	return "performance_snapshots"
}
