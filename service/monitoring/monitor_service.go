/*
 * @module service/monitoring/monitor_service
 * @description 监控服务，负责系统性能监控、同步任务监控、数据源健康监控和指标收集聚合
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 指标收集 -> 数据聚合 -> 状态评估 -> 告警检测
 * @rules 确保监控数据的准确性和实时性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/patch_basic_library_process.md
 */

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// MonitorService 监控服务
type MonitorService struct {
	db               *gorm.DB
	metricsCollector *MetricsCollector
	healthChecker    *HealthChecker
	alertManager     *AlertManager

	// 监控配置
	monitoringConfig *MonitoringConfig

	// 运行状态
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc
	monitorMutex sync.RWMutex

	// 指标缓存
	metricsCache map[string]*MetricSnapshot
	cacheMutex   sync.RWMutex

	// 事件通知器
	eventNotifier func(event *MonitoringEvent)
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
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

// NewMonitorService 创建监控服务实例
func NewMonitorService(db *gorm.DB) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &MonitorService{
		db:               db,
		ctx:              ctx,
		cancel:           cancel,
		isRunning:        false,
		metricsCache:     make(map[string]*MetricSnapshot),
		monitoringConfig: getDefaultMonitoringConfig(),
	}

	// 初始化子组件
	service.metricsCollector = NewMetricsCollector(db)
	service.healthChecker = NewHealthChecker(db)
	service.alertManager = NewAlertManager(db)

	return service
}

// Start 启动监控服务
func (m *MonitorService) Start() error {
	m.monitorMutex.Lock()
	defer m.monitorMutex.Unlock()

	if m.isRunning {
		return fmt.Errorf("监控服务已在运行中")
	}

	m.isRunning = true

	// 启动各个监控协程
	go m.metricsCollectionLoop()
	go m.healthCheckLoop()
	go m.alertCheckLoop()
	go m.cacheCleanupLoop()

	m.notifyEvent(&MonitoringEvent{
		ID:        generateEventID(),
		Type:      "service_started",
		Timestamp: time.Now(),
		Source:    "monitor_service",
		Severity:  "info",
		Message:   "监控服务已启动",
	})

	return nil
}

// Stop 停止监控服务
func (m *MonitorService) Stop() error {
	m.monitorMutex.Lock()
	defer m.monitorMutex.Unlock()

	if !m.isRunning {
		return fmt.Errorf("监控服务未运行")
	}

	m.cancel()
	m.isRunning = false

	m.notifyEvent(&MonitoringEvent{
		ID:        generateEventID(),
		Type:      "service_stopped",
		Timestamp: time.Now(),
		Source:    "monitor_service",
		Severity:  "info",
		Message:   "监控服务已停止",
	})

	return nil
}

// GetSystemMetrics 获取系统指标
func (m *MonitorService) GetSystemMetrics() (*SystemMetrics, error) {
	return m.metricsCollector.CollectSystemMetrics()
}

// GetDataSourceMetrics 获取数据源指标
func (m *MonitorService) GetDataSourceMetrics(dataSourceID string) (*DataSourceMetrics, error) {
	return m.metricsCollector.CollectDataSourceMetrics(dataSourceID)
}

// GetSyncTaskMetrics 获取同步任务指标
func (m *MonitorService) GetSyncTaskMetrics(timeRange string) (*SyncTaskMetrics, error) {
	return m.metricsCollector.CollectSyncTaskMetrics(timeRange)
}

// GetHealthStatus 获取健康状态
func (m *MonitorService) GetHealthStatus() (*HealthStatus, error) {
	return m.healthChecker.CheckOverallHealth()
}

// GetDataSourceHealth 获取数据源健康状态
func (m *MonitorService) GetDataSourceHealth(dataSourceID string) (*DataSourceHealth, error) {
	return m.healthChecker.CheckDataSourceHealth(dataSourceID)
}

// GetActiveAlerts 获取活跃告警
func (m *MonitorService) GetActiveAlerts() ([]*Alert, error) {
	return m.alertManager.GetActiveAlerts()
}

// GetMetricSnapshot 获取指标快照
func (m *MonitorService) GetMetricSnapshot(metricType string) (*MetricSnapshot, error) {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	snapshot, exists := m.metricsCache[metricType]
	if !exists {
		return nil, fmt.Errorf("指标类型 %s 不存在", metricType)
	}

	return snapshot, nil
}

// UpdateMonitoringConfig 更新监控配置
func (m *MonitorService) UpdateMonitoringConfig(config *MonitoringConfig) error {
	m.monitorMutex.Lock()
	defer m.monitorMutex.Unlock()

	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	m.monitoringConfig = config

	// 保存配置到数据库
	if err := m.saveMonitoringConfig(config); err != nil {
		return fmt.Errorf("保存配置失败: %v", err)
	}

	m.notifyEvent(&MonitoringEvent{
		ID:        generateEventID(),
		Type:      "config_updated",
		Timestamp: time.Now(),
		Source:    "monitor_service",
		Severity:  "info",
		Message:   "监控配置已更新",
	})

	return nil
}

// SetEventNotifier 设置事件通知器
func (m *MonitorService) SetEventNotifier(notifier func(event *MonitoringEvent)) {
	m.eventNotifier = notifier
}

// 指标收集循环
func (m *MonitorService) metricsCollectionLoop() {
	ticker := time.NewTicker(m.monitoringConfig.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectAllMetrics()
		}
	}
}

// 健康检查循环
func (m *MonitorService) healthCheckLoop() {
	ticker := time.NewTicker(m.monitoringConfig.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthChecks()
		}
	}
}

// 告警检查循环
func (m *MonitorService) alertCheckLoop() {
	ticker := time.NewTicker(m.monitoringConfig.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAlerts()
		}
	}
}

// 缓存清理循环
func (m *MonitorService) cacheCleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupCache()
		}
	}
}

// 收集所有指标
func (m *MonitorService) collectAllMetrics() {
	// 收集系统指标
	if m.isMetricEnabled("system") {
		if systemMetrics, err := m.metricsCollector.CollectSystemMetrics(); err == nil {
			m.updateMetricCache("system", systemMetrics)
		}
	}

	// 收集数据源指标
	if m.isMetricEnabled("datasource") {
		if dataSourceMetrics, err := m.metricsCollector.CollectAllDataSourceMetrics(); err == nil {
			m.updateMetricCache("datasource", dataSourceMetrics)
		}
	}

	// 收集同步任务指标
	if m.isMetricEnabled("sync_task") {
		if syncMetrics, err := m.metricsCollector.CollectSyncTaskMetrics("1h"); err == nil {
			m.updateMetricCache("sync_task", syncMetrics)
		}
	}
}

// 执行健康检查
func (m *MonitorService) performHealthChecks() {
	// 检查系统整体健康状态
	if health, err := m.healthChecker.CheckOverallHealth(); err == nil {
		m.updateMetricCache("health", health)

		// 如果健康状态发生变化，发送事件
		if health.Score < 80 {
			m.notifyEvent(&MonitoringEvent{
				ID:        generateEventID(),
				Type:      "health_warning",
				Timestamp: time.Now(),
				Source:    "health_checker",
				Severity:  "warning",
				Message:   fmt.Sprintf("系统健康评分较低: %d", health.Score),
				Data:      map[string]interface{}{"health": health},
			})
		}
	}
}

// 检查告警
func (m *MonitorService) checkAlerts() {
	alerts, err := m.alertManager.CheckAlertRules(m.metricsCache)
	if err != nil {
		return
	}

	for _, alert := range alerts {
		m.notifyEvent(&MonitoringEvent{
			ID:        generateEventID(),
			Type:      "alert_triggered",
			Timestamp: time.Now(),
			Source:    "alert_manager",
			Severity:  alert.Severity,
			Message:   alert.Message,
			Data:      map[string]interface{}{"alert": alert},
		})
	}
}

// 更新指标缓存
func (m *MonitorService) updateMetricCache(metricType string, value interface{}) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	snapshot := &MetricSnapshot{
		MetricType: metricType,
		Timestamp:  time.Now(),
		Value:      value,
		Tags:       make(map[string]string),
	}

	// 计算趋势信息
	if prevSnapshot, exists := m.metricsCache[metricType]; exists {
		snapshot.Trend = m.calculateTrend(prevSnapshot, snapshot)
	}

	m.metricsCache[metricType] = snapshot
}

// 计算趋势信息
func (m *MonitorService) calculateTrend(prev, current *MetricSnapshot) TrendInfo {
	// 简化的趋势计算逻辑
	trend := TrendInfo{
		Direction:  "stable",
		ChangeRate: 0,
		Confidence: 0.8,
	}

	// 这里可以实现更复杂的趋势分析算法
	return trend
}

// 清理缓存
func (m *MonitorService) cleanupCache() {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	cutoffTime := time.Now().Add(-24 * time.Hour)
	for key, snapshot := range m.metricsCache {
		if snapshot.Timestamp.Before(cutoffTime) {
			delete(m.metricsCache, key)
		}
	}
}

// 检查指标是否启用
func (m *MonitorService) isMetricEnabled(metricType string) bool {
	for _, enabled := range m.monitoringConfig.EnabledMetrics {
		if enabled == metricType {
			return true
		}
	}
	return false
}

// 验证配置
func (m *MonitorService) validateConfig(config *MonitoringConfig) error {
	if config.CollectionInterval < time.Second {
		return fmt.Errorf("收集间隔不能小于1秒")
	}
	if config.HealthCheckInterval < time.Second {
		return fmt.Errorf("健康检查间隔不能小于1秒")
	}
	if config.AlertCheckInterval < time.Second {
		return fmt.Errorf("告警检查间隔不能小于1秒")
	}
	if config.MetricsRetentionDays < 1 {
		return fmt.Errorf("指标保留天数不能小于1天")
	}
	return nil
}

// 保存监控配置
func (m *MonitorService) saveMonitoringConfig(config *MonitoringConfig) error {
	_, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// 这里可以保存到数据库或配置文件
	// 暂时使用内存存储
	return nil
}

// 发送事件通知
func (m *MonitorService) notifyEvent(event *MonitoringEvent) {
	if m.eventNotifier != nil {
		go m.eventNotifier(event)
	}
}

// 获取默认监控配置
func getDefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		CollectionInterval:   30 * time.Second,
		HealthCheckInterval:  60 * time.Second,
		AlertCheckInterval:   30 * time.Second,
		MetricsRetentionDays: 7,
		EnabledMetrics:       []string{"system", "datasource", "sync_task"},
		AlertRules:           make(map[string]interface{}),
		NotificationChannels: []NotificationChannel{},
	}
}

// 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}
