/*
 * @module service/monitoring/alert_manager
 * @description 告警管理器，负责告警规则管理、告警触发检测、告警通知发送和告警升级机制
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 规则定义 -> 指标监控 -> 触发检测 -> 通知发送 -> 状态跟踪
 * @rules 确保告警的及时性和准确性，避免告警风暴
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/patch_basic_library_process.md
 */

package monitoring

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// AlertManager 告警管理器
type AlertManager struct {
	db           *gorm.DB
	alertRules   map[string]*AlertRule
	activeAlerts map[string]*Alert
	alertHistory []*Alert
	mutex        sync.RWMutex

	// 告警配置
	alertConfig *AlertConfig

	// 通知渠道
	notificationChannels map[string]NotificationSender

	// 告警抑制
	suppressionRules map[string]*SuppressionRule
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	MetricType  string            `json:"metric_type"` // system, datasource, sync_task
	Condition   AlertCondition    `json:"condition"`
	Severity    string            `json:"severity"` // info, warning, error, critical
	IsEnabled   bool              `json:"is_enabled"`
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"` // 持续时间
	Actions     []AlertAction     `json:"actions"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// AlertCondition 告警条件
type AlertCondition struct {
	Operator string      `json:"operator"` // gt, lt, eq, ne, gte, lte
	Value    interface{} `json:"value"`
	Field    string      `json:"field"`    // 指标字段
	Function string      `json:"function"` // avg, max, min, sum, count
}

// AlertAction 告警动作
type AlertAction struct {
	Type     string                 `json:"type"`   // notification, webhook, script
	Target   string                 `json:"target"` // 目标地址或脚本路径
	Config   map[string]interface{} `json:"config"`
	IsActive bool                   `json:"is_active"`
}

// Alert 告警实例
type Alert struct {
	ID          string `json:"id"`
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Severity    string `json:"severity"`
	Status      string `json:"status"` // firing, resolved, silenced
	Message     string `json:"message"`
	Description string `json:"description"`

	// 告警触发信息
	TriggeredAt time.Time  `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	LastSent    *time.Time `json:"last_sent,omitempty"`
	SendCount   int        `json:"send_count"`

	// 关联的指标值
	MetricValue interface{} `json:"metric_value"`
	Threshold   float64     `json:"threshold"`

	// 标签和注释
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`

	// 源信息
	Source     string `json:"source"`      // 告警源
	ObjectID   string `json:"object_id"`   // 关联对象ID
	ObjectType string `json:"object_type"` // 关联对象类型
}

// AlertConfig 告警配置
type AlertConfig struct {
	RepeatInterval  time.Duration `json:"repeat_interval"`  // 重复通知间隔
	ResolveTimeout  time.Duration `json:"resolve_timeout"`  // 自动解决超时
	EscalationTime  time.Duration `json:"escalation_time"`  // 升级时间
	MaxSendCount    int           `json:"max_send_count"`   // 最大发送次数
	GroupInterval   time.Duration `json:"group_interval"`   // 分组间隔
	SilenceDuration time.Duration `json:"silence_duration"` // 静默时长
}

// SuppressionRule 告警抑制规则
type SuppressionRule struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Matchers  map[string]string `json:"matchers"` // 匹配条件
	IsActive  bool              `json:"is_active"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	CreatedBy string            `json:"created_by"`
}

// NewAlertManager 创建告警管理器实例
func NewAlertManager(db *gorm.DB) *AlertManager {
	return &AlertManager{
		db:                   db,
		alertRules:           make(map[string]*AlertRule),
		activeAlerts:         make(map[string]*Alert),
		alertHistory:         []*Alert{},
		notificationChannels: make(map[string]NotificationSender),
		suppressionRules:     make(map[string]*SuppressionRule),
		alertConfig:          getDefaultAlertConfig(),
	}
}

// AddAlertRule 添加告警规则
func (a *AlertManager) AddAlertRule(rule *AlertRule) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err := a.validateAlertRule(rule); err != nil {
		return fmt.Errorf("告警规则验证失败: %v", err)
	}

	rule.ID = generateAlertRuleID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	a.alertRules[rule.ID] = rule

	// 保存到数据库
	if err := a.saveAlertRule(rule); err != nil {
		return fmt.Errorf("保存告警规则失败: %v", err)
	}

	return nil
}

// UpdateAlertRule 更新告警规则
func (a *AlertManager) UpdateAlertRule(ruleID string, updates *AlertRule) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	rule, exists := a.alertRules[ruleID]
	if !exists {
		return fmt.Errorf("告警规则 %s 不存在", ruleID)
	}

	if err := a.validateAlertRule(updates); err != nil {
		return fmt.Errorf("告警规则验证失败: %v", err)
	}

	// 更新字段
	rule.Name = updates.Name
	rule.Description = updates.Description
	rule.Condition = updates.Condition
	rule.Severity = updates.Severity
	rule.IsEnabled = updates.IsEnabled
	rule.Threshold = updates.Threshold
	rule.Duration = updates.Duration
	rule.Actions = updates.Actions
	rule.UpdatedAt = time.Now()

	// 保存到数据库
	if err := a.saveAlertRule(rule); err != nil {
		return fmt.Errorf("更新告警规则失败: %v", err)
	}

	return nil
}

// DeleteAlertRule 删除告警规则
func (a *AlertManager) DeleteAlertRule(ruleID string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.alertRules[ruleID]; !exists {
		return fmt.Errorf("告警规则 %s 不存在", ruleID)
	}

	delete(a.alertRules, ruleID)

	// 从数据库删除
	if err := a.deleteAlertRule(ruleID); err != nil {
		return fmt.Errorf("删除告警规则失败: %v", err)
	}

	return nil
}

// GetAlertRules 获取所有告警规则
func (a *AlertManager) GetAlertRules() map[string]*AlertRule {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	rules := make(map[string]*AlertRule)
	for id, rule := range a.alertRules {
		rules[id] = rule
	}
	return rules
}

// CheckAlertRules 检查告警规则
func (a *AlertManager) CheckAlertRules(metricsCache map[string]*MetricSnapshot) ([]*Alert, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	var triggeredAlerts []*Alert

	for _, rule := range a.alertRules {
		if !rule.IsEnabled {
			continue
		}

		// 获取对应的指标数据
		metricSnapshot, exists := metricsCache[rule.MetricType]
		if !exists {
			continue
		}

		// 检查告警条件
		if a.evaluateAlertCondition(rule, metricSnapshot) {
			alert := a.createAlert(rule, metricSnapshot)

			// 检查是否需要触发新告警或更新现有告警
			if a.shouldTriggerAlert(alert) {
				triggeredAlerts = append(triggeredAlerts, alert)
				a.activeAlerts[alert.ID] = alert
				a.alertHistory = append(a.alertHistory, alert)

				// 发送告警通知
				go a.sendAlertNotification(alert)
			}
		} else {
			// 检查是否需要解决现有告警
			a.resolveAlertsForRule(rule.ID)
		}
	}

	return triggeredAlerts, nil
}

// GetActiveAlerts 获取活跃告警
func (a *AlertManager) GetActiveAlerts() ([]*Alert, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var alerts []*Alert
	for _, alert := range a.activeAlerts {
		if alert.Status == "firing" {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// GetAlertHistory 获取告警历史
func (a *AlertManager) GetAlertHistory(limit int) ([]*Alert, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	history := make([]*Alert, 0)
	start := len(a.alertHistory) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(a.alertHistory); i++ {
		history = append(history, a.alertHistory[i])
	}

	return history, nil
}

// SilenceAlert 静默告警
func (a *AlertManager) SilenceAlert(alertID string, duration time.Duration, reason string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	alert, exists := a.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("告警 %s 不存在", alertID)
	}

	alert.Status = "silenced"
	alert.Annotations["silence_reason"] = reason
	alert.Annotations["silence_until"] = time.Now().Add(duration).Format(time.RFC3339)

	return nil
}

// ResolveAlert 解决告警
func (a *AlertManager) ResolveAlert(alertID string, reason string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	alert, exists := a.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("告警 %s 不存在", alertID)
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedAt = &now
	alert.Annotations["resolve_reason"] = reason

	delete(a.activeAlerts, alertID)

	return nil
}

// 验证告警规则
func (a *AlertManager) validateAlertRule(rule *AlertRule) error {
	if rule.Name == "" {
		return fmt.Errorf("告警规则名称不能为空")
	}

	if rule.MetricType == "" {
		return fmt.Errorf("指标类型不能为空")
	}

	if rule.Condition.Operator == "" {
		return fmt.Errorf("条件操作符不能为空")
	}

	validOperators := []string{"gt", "lt", "eq", "ne", "gte", "lte"}
	validOperator := false
	for _, op := range validOperators {
		if rule.Condition.Operator == op {
			validOperator = true
			break
		}
	}
	if !validOperator {
		return fmt.Errorf("无效的条件操作符: %s", rule.Condition.Operator)
	}

	validSeverities := []string{"info", "warning", "error", "critical"}
	validSeverity := false
	for _, sev := range validSeverities {
		if rule.Severity == sev {
			validSeverity = true
			break
		}
	}
	if !validSeverity {
		return fmt.Errorf("无效的严重性级别: %s", rule.Severity)
	}

	return nil
}

// 评估告警条件
func (a *AlertManager) evaluateAlertCondition(rule *AlertRule, snapshot *MetricSnapshot) bool {
	// 从指标快照中提取值
	value := a.extractMetricValue(snapshot, rule.Condition.Field)
	if value == nil {
		return false
	}

	// 根据条件操作符进行比较
	switch rule.Condition.Operator {
	case "gt":
		return a.compareValues(value, rule.Threshold, ">")
	case "lt":
		return a.compareValues(value, rule.Threshold, "<")
	case "eq":
		return a.compareValues(value, rule.Threshold, "==")
	case "ne":
		return a.compareValues(value, rule.Threshold, "!=")
	case "gte":
		return a.compareValues(value, rule.Threshold, ">=")
	case "lte":
		return a.compareValues(value, rule.Threshold, "<=")
	default:
		return false
	}
}

// 从指标快照中提取值
func (a *AlertManager) extractMetricValue(snapshot *MetricSnapshot, field string) interface{} {
	if field == "" {
		return snapshot.Value
	}

	// 支持嵌套字段访问，如 "system.cpu_usage"
	fields := strings.Split(field, ".")
	current := snapshot.Value

	for _, f := range fields {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[f]
		case *SystemMetrics:
			current = a.getSystemMetricField(v, f)
		case *DataSourceMetrics:
			current = a.getDataSourceMetricField(v, f)
		case *SyncTaskMetrics:
			current = a.getSyncTaskMetricField(v, f)
		default:
			return nil
		}
	}

	return current
}

// 获取系统指标字段值
func (a *AlertManager) getSystemMetricField(metrics *SystemMetrics, field string) interface{} {
	switch field {
	case "cpu_usage":
		return metrics.CPUUsage
	case "memory_usage":
		return metrics.MemoryUsage
	case "disk_usage":
		return metrics.DiskUsage
	case "goroutine_count":
		return metrics.GoroutineCount
	case "qps":
		return metrics.QPS
	case "response_time":
		return metrics.ResponseTime
	default:
		return nil
	}
}

// 获取数据源指标字段值
func (a *AlertManager) getDataSourceMetricField(metrics *DataSourceMetrics, field string) interface{} {
	switch field {
	case "success_rate":
		return metrics.SuccessRate
	case "error_rate":
		return metrics.ErrorRate
	case "avg_response_time":
		return metrics.AvgResponseTime
	case "throughput":
		return metrics.Throughput
	case "quality_score":
		return metrics.QualityScore
	default:
		return nil
	}
}

// 获取同步任务指标字段值
func (a *AlertManager) getSyncTaskMetricField(metrics *SyncTaskMetrics, field string) interface{} {
	switch field {
	case "success_rate":
		return metrics.SuccessRate
	case "avg_execution_time":
		return metrics.AvgExecutionTime
	case "throughput":
		return metrics.Throughput
	case "failed_tasks":
		return metrics.FailedTasks
	default:
		return nil
	}
}

// 比较值
func (a *AlertManager) compareValues(value interface{}, threshold float64, operator string) bool {
	var numValue float64

	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	default:
		return false
	}

	switch operator {
	case ">":
		return numValue > threshold
	case "<":
		return numValue < threshold
	case "==":
		return numValue == threshold
	case "!=":
		return numValue != threshold
	case ">=":
		return numValue >= threshold
	case "<=":
		return numValue <= threshold
	default:
		return false
	}
}

// 创建告警
func (a *AlertManager) createAlert(rule *AlertRule, snapshot *MetricSnapshot) *Alert {
	alert := &Alert{
		ID:          generateAlertID(),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		Status:      "firing",
		Message:     fmt.Sprintf("告警规则 %s 被触发", rule.Name),
		Description: rule.Description,
		TriggeredAt: time.Now(),
		MetricValue: a.extractMetricValue(snapshot, rule.Condition.Field),
		Threshold:   rule.Threshold,
		Labels:      rule.Labels,
		Annotations: make(map[string]interface{}),
		Source:      "alert_manager",
	}

	// 复制规则注释
	for k, v := range rule.Annotations {
		alert.Annotations[k] = v
	}

	return alert
}

// 检查是否应该触发告警
func (a *AlertManager) shouldTriggerAlert(alert *Alert) bool {
	// 检查是否存在相同的活跃告警
	for _, existingAlert := range a.activeAlerts {
		if existingAlert.RuleID == alert.RuleID &&
			existingAlert.ObjectID == alert.ObjectID &&
			existingAlert.Status == "firing" {
			return false // 已存在相同告警
		}
	}

	// 检查抑制规则
	if a.isAlertSuppressed(alert) {
		return false
	}

	return true
}

// 检查告警是否被抑制
func (a *AlertManager) isAlertSuppressed(alert *Alert) bool {
	for _, rule := range a.suppressionRules {
		if !rule.IsActive {
			continue
		}

		if rule.ExpiresAt != nil && time.Now().After(*rule.ExpiresAt) {
			rule.IsActive = false
			continue
		}

		// 检查匹配条件
		matches := true
		for key, pattern := range rule.Matchers {
			if labelValue, exists := alert.Labels[key]; !exists || labelValue != pattern {
				matches = false
				break
			}
		}

		if matches {
			return true
		}
	}

	return false
}

// 解决规则相关的告警
func (a *AlertManager) resolveAlertsForRule(ruleID string) {
	now := time.Now()
	for id, alert := range a.activeAlerts {
		if alert.RuleID == ruleID && alert.Status == "firing" {
			alert.Status = "resolved"
			alert.ResolvedAt = &now
			alert.Annotations["auto_resolved"] = true
			delete(a.activeAlerts, id)
		}
	}
}

// 发送告警通知
func (a *AlertManager) sendAlertNotification(alert *Alert) {
	rule := a.alertRules[alert.RuleID]
	if rule == nil {
		return
	}

	for _, action := range rule.Actions {
		if !action.IsActive {
			continue
		}

		switch action.Type {
		case "notification":
			a.sendNotification(alert, action)
		case "webhook":
			a.sendWebhook(alert, action)
		case "script":
			a.executeScript(alert, action)
		}
	}

	alert.SendCount++
	now := time.Now()
	alert.LastSent = &now
}

// 发送通知
func (a *AlertManager) sendNotification(alert *Alert, action AlertAction) {
	// 简化实现
	fmt.Printf("发送告警通知: %s - %s\n", alert.Severity, alert.Message)
}

// 发送Webhook
func (a *AlertManager) sendWebhook(alert *Alert, action AlertAction) {
	// 简化实现
	fmt.Printf("发送Webhook告警: %s\n", action.Target)
}

// 执行脚本
func (a *AlertManager) executeScript(alert *Alert, action AlertAction) {
	// 简化实现
	fmt.Printf("执行告警脚本: %s\n", action.Target)
}

// 保存告警规则到数据库
func (a *AlertManager) saveAlertRule(rule *AlertRule) error {
	// 简化实现，实际应保存到数据库
	return nil
}

// 从数据库删除告警规则
func (a *AlertManager) deleteAlertRule(ruleID string) error {
	// 简化实现，实际应从数据库删除
	return nil
}

// 获取默认告警配置
func getDefaultAlertConfig() *AlertConfig {
	return &AlertConfig{
		RepeatInterval:  15 * time.Minute,
		ResolveTimeout:  5 * time.Minute,
		EscalationTime:  30 * time.Minute,
		MaxSendCount:    5,
		GroupInterval:   30 * time.Second,
		SilenceDuration: 1 * time.Hour,
	}
}

// 生成告警规则ID
func generateAlertRuleID() string {
	return fmt.Sprintf("rule_%d", time.Now().UnixNano())
}

// 生成告警ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
