/*
 * @module service/thematic_monitor_service
 * @description 主题同步监控服务，提供性能监控、质量监控和告警功能
 * @architecture 服务层 - 封装监控和告警相关的业务逻辑
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 指标收集 -> 阈值检查 -> 告警触发 -> 通知发送
 * @rules 确保监控数据的准确性和告警的及时性
 * @dependencies gorm.io/gorm, context, time
 * @refs service/models/thematic_sync.go
 */

package thematic_library

import (
	"context"
	"datahub-service/service/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ThematicMonitorService 主题同步监控服务
type ThematicMonitorService struct {
	db *gorm.DB
}

// NewThematicMonitorService 创建主题同步监控服务
func NewThematicMonitorService(db *gorm.DB) *ThematicMonitorService {
	return &ThematicMonitorService{
		db: db,
	}
}

// GetPerformanceMetrics 获取性能指标
func (tms *ThematicMonitorService) GetPerformanceMetrics(ctx context.Context, req *PerformanceMetricsRequest) (*PerformanceMetricsResponse, error) {
	query := tms.db.Model(&models.ThematicSyncExecution{})

	// 添加时间范围过滤
	if req.StartTime != nil {
		query = query.Where("start_time >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("start_time <= ?", *req.EndTime)
	}
	if req.TaskID != "" {
		query = query.Where("task_id = ?", req.TaskID)
	}

	// 查询执行记录
	var executions []models.ThematicSyncExecution
	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	// 计算性能指标
	response := &PerformanceMetricsResponse{
		TimeRange: TimeRange{
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		},
		Metrics: make([]PerformanceMetric, 0),
	}

	if len(executions) == 0 {
		return response, nil
	}

	// 计算汇总指标
	var totalDuration int64
	var totalRecords int64
	var successCount int64
	var failedCount int64

	for _, exec := range executions {
		totalDuration += exec.Duration
		totalRecords += exec.ProcessedRecordCount

		if exec.Status == "success" {
			successCount++
		} else if exec.Status == "failed" {
			failedCount++
		}
	}

	// 平均处理时间
	avgDuration := float64(0)
	if len(executions) > 0 {
		avgDuration = float64(totalDuration) / float64(len(executions))
	}

	// 吞吐量（记录数/秒）
	throughput := float64(0)
	if totalDuration > 0 {
		throughput = float64(totalRecords) / float64(totalDuration)
	}

	// 成功率
	successRate := float64(0)
	if len(executions) > 0 {
		successRate = float64(successCount) / float64(len(executions)) * 100
	}

	// 构建指标
	metrics := []PerformanceMetric{
		{
			MetricName:    "avg_duration",
			MetricValue:   avgDuration,
			Unit:          "seconds",
			Description:   "平均执行时间",
			CollectedTime: time.Now(),
		},
		{
			MetricName:    "throughput",
			MetricValue:   throughput,
			Unit:          "records/second",
			Description:   "数据处理吞吐量",
			CollectedTime: time.Now(),
		},
		{
			MetricName:    "success_rate",
			MetricValue:   successRate,
			Unit:          "percent",
			Description:   "执行成功率",
			CollectedTime: time.Now(),
		},
		{
			MetricName:    "total_executions",
			MetricValue:   float64(len(executions)),
			Unit:          "count",
			Description:   "总执行次数",
			CollectedTime: time.Now(),
		},
	}

	response.Metrics = metrics
	return response, nil
}

// GetQualityMetrics 获取质量指标
func (tms *ThematicMonitorService) GetQualityMetrics(ctx context.Context, req *QualityMetricsRequest) (*QualityMetricsResponse, error) {
	query := tms.db.Model(&models.ThematicSyncExecution{})

	// 添加过滤条件
	if req.StartTime != nil {
		query = query.Where("start_time >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("start_time <= ?", *req.EndTime)
	}
	if req.TaskID != "" {
		query = query.Where("task_id = ?", req.TaskID)
	}

	// 查询执行记录
	var executions []models.ThematicSyncExecution
	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	response := &QualityMetricsResponse{
		TimeRange: TimeRange{
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		},
		Metrics: make([]MonitorQualityMetric, 0),
	}

	if len(executions) == 0 {
		return response, nil
	}

	// 计算质量指标
	var totalSourceRecords int64
	var totalProcessedRecords int64
	var totalErrorRecords int64

	for _, exec := range executions {
		totalSourceRecords += exec.SourceRecordCount
		totalProcessedRecords += exec.ProcessedRecordCount
		totalErrorRecords += exec.ErrorRecordCount
	}

	// 完整性评分
	completenessScore := float64(0)
	if totalSourceRecords > 0 {
		completenessScore = float64(totalProcessedRecords) / float64(totalSourceRecords) * 100
	}

	// 准确性评分
	accuracyScore := float64(0)
	if totalProcessedRecords > 0 {
		accuracyScore = float64(totalProcessedRecords-totalErrorRecords) / float64(totalProcessedRecords) * 100
	}

	// 错误率
	errorRate := float64(0)
	if totalProcessedRecords > 0 {
		errorRate = float64(totalErrorRecords) / float64(totalProcessedRecords) * 100
	}

	metrics := []MonitorQualityMetric{
		{
			MetricName:    "completeness_score",
			MetricValue:   completenessScore,
			Unit:          "percent",
			Description:   "数据完整性评分",
			Threshold:     95.0,
			IsHealthy:     completenessScore >= 95.0,
			CollectedTime: time.Now(),
		},
		{
			MetricName:    "accuracy_score",
			MetricValue:   accuracyScore,
			Unit:          "percent",
			Description:   "数据准确性评分",
			Threshold:     90.0,
			IsHealthy:     accuracyScore >= 90.0,
			CollectedTime: time.Now(),
		},
		{
			MetricName:    "error_rate",
			MetricValue:   errorRate,
			Unit:          "percent",
			Description:   "数据错误率",
			Threshold:     5.0,
			IsHealthy:     errorRate <= 5.0,
			CollectedTime: time.Now(),
		},
	}

	response.Metrics = metrics
	return response, nil
}

// CheckAlerts 检查告警
func (tms *ThematicMonitorService) CheckAlerts(ctx context.Context) ([]AlertInfo, error) {
	var alerts []AlertInfo

	// 检查性能告警
	perfAlerts, err := tms.checkPerformanceAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查性能告警失败: %w", err)
	}
	alerts = append(alerts, perfAlerts...)

	// 检查质量告警
	qualityAlerts, err := tms.checkQualityAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查质量告警失败: %w", err)
	}
	alerts = append(alerts, qualityAlerts...)

	// 检查错误告警
	errorAlerts, err := tms.checkErrorAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查错误告警失败: %w", err)
	}
	alerts = append(alerts, errorAlerts...)

	return alerts, nil
}

// checkPerformanceAlerts 检查性能告警
func (tms *ThematicMonitorService) checkPerformanceAlerts(ctx context.Context) ([]AlertInfo, error) {
	var alerts []AlertInfo

	// 查询最近的执行记录
	var executions []models.ThematicSyncExecution
	if err := tms.db.Where("start_time >= ?", time.Now().Add(-1*time.Hour)).
		Find(&executions).Error; err != nil {
		return nil, err
	}

	// 检查执行时间过长
	for _, exec := range executions {
		if exec.Duration > 3600 { // 超过1小时
			alert := AlertInfo{
				AlertID:     fmt.Sprintf("perf_duration_%s", exec.ID),
				AlertType:   "performance",
				Severity:    "medium",
				Title:       "执行时间过长",
				Description: fmt.Sprintf("任务 %s 执行时间 %d 秒超过阈值", exec.TaskID, exec.Duration),
				TaskID:      exec.TaskID,
				MetricName:  "duration",
				MetricValue: float64(exec.Duration),
				Threshold:   3600.0,
				TriggeredAt: time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// checkQualityAlerts 检查质量告警
func (tms *ThematicMonitorService) checkQualityAlerts(ctx context.Context) ([]AlertInfo, error) {
	var alerts []AlertInfo

	// 查询最近的执行记录
	var executions []models.ThematicSyncExecution
	if err := tms.db.Where("start_time >= ?", time.Now().Add(-1*time.Hour)).
		Find(&executions).Error; err != nil {
		return nil, err
	}

	// 检查错误率
	for _, exec := range executions {
		if exec.ProcessedRecordCount > 0 {
			errorRate := float64(exec.ErrorRecordCount) / float64(exec.ProcessedRecordCount) * 100
			if errorRate > 5.0 { // 错误率超过5%
				alert := AlertInfo{
					AlertID:     fmt.Sprintf("quality_error_rate_%s", exec.ID),
					AlertType:   "quality",
					Severity:    "high",
					Title:       "数据质量异常",
					Description: fmt.Sprintf("任务 %s 错误率 %.2f%% 超过阈值", exec.TaskID, errorRate),
					TaskID:      exec.TaskID,
					MetricName:  "error_rate",
					MetricValue: errorRate,
					Threshold:   5.0,
					TriggeredAt: time.Now(),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts, nil
}

// checkErrorAlerts 检查错误告警
func (tms *ThematicMonitorService) checkErrorAlerts(ctx context.Context) ([]AlertInfo, error) {
	var alerts []AlertInfo

	// 查询失败的执行记录
	var failedExecutions []models.ThematicSyncExecution
	if err := tms.db.Where("status = ? AND start_time >= ?", "failed", time.Now().Add(-1*time.Hour)).
		Find(&failedExecutions).Error; err != nil {
		return nil, err
	}

	// 为每个失败的执行创建告警
	for _, exec := range failedExecutions {
		alert := AlertInfo{
			AlertID:     fmt.Sprintf("error_execution_%s", exec.ID),
			AlertType:   "error",
			Severity:    "high",
			Title:       "同步任务执行失败",
			Description: fmt.Sprintf("任务 %s 执行失败", exec.TaskID),
			TaskID:      exec.TaskID,
			ExecutionID: exec.ID,
			TriggeredAt: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetSystemHealth 获取系统健康状况
func (tms *ThematicMonitorService) GetSystemHealth(ctx context.Context) (*SystemHealthResponse, error) {
	// 查询最近24小时的执行记录
	var executions []models.ThematicSyncExecution
	if err := tms.db.Where("start_time >= ?", time.Now().Add(-24*time.Hour)).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	// 查询活跃任务数
	var activeTasks int64
	if err := tms.db.Model(&models.ThematicSyncTask{}).
		Where("status = ?", "active").
		Count(&activeTasks).Error; err != nil {
		return nil, fmt.Errorf("查询活跃任务数失败: %w", err)
	}

	// 计算健康指标
	totalExecutions := len(executions)
	successfulExecutions := 0
	failedExecutions := 0

	for _, exec := range executions {
		if exec.Status == "success" {
			successfulExecutions++
		} else if exec.Status == "failed" {
			failedExecutions++
		}
	}

	// 计算健康评分
	healthScore := float64(100)
	if totalExecutions > 0 {
		successRate := float64(successfulExecutions) / float64(totalExecutions)
		healthScore = successRate * 100
	}

	// 确定健康状态
	healthStatus := "healthy"
	if healthScore < 50 {
		healthStatus = "critical"
	} else if healthScore < 80 {
		healthStatus = "warning"
	}

	response := &SystemHealthResponse{
		OverallHealth: HealthStatus{
			Status:      healthStatus,
			Score:       healthScore,
			Description: fmt.Sprintf("系统健康评分 %.1f%%", healthScore),
			CheckTime:   time.Now(),
		},
		Components: []ComponentHealth{
			{
				ComponentName: "sync_execution",
				Status:        healthStatus,
				Score:         healthScore,
				Metrics: map[string]float64{
					"total_executions":      float64(totalExecutions),
					"successful_executions": float64(successfulExecutions),
					"failed_executions":     float64(failedExecutions),
				},
				LastCheckTime: time.Now(),
			},
			{
				ComponentName: "active_tasks",
				Status:        "healthy",
				Score:         100.0,
				Metrics: map[string]float64{
					"active_task_count": float64(activeTasks),
				},
				LastCheckTime: time.Now(),
			},
		},
		CheckTime: time.Now(),
	}

	return response, nil
}

// 请求和响应结构体

// PerformanceMetricsRequest 性能指标请求
type PerformanceMetricsRequest struct {
	TaskID    string     `json:"task_id,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// PerformanceMetricsResponse 性能指标响应
type PerformanceMetricsResponse struct {
	TimeRange TimeRange           `json:"time_range"`
	Metrics   []PerformanceMetric `json:"metrics"`
}

// PerformanceMetric 性能指标
type PerformanceMetric struct {
	MetricName    string    `json:"metric_name"`
	MetricValue   float64   `json:"metric_value"`
	Unit          string    `json:"unit"`
	Description   string    `json:"description"`
	CollectedTime time.Time `json:"collected_time"`
}

// QualityMetricsRequest 质量指标请求
type QualityMetricsRequest struct {
	TaskID    string     `json:"task_id,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// QualityMetricsResponse 质量指标响应
type QualityMetricsResponse struct {
	TimeRange TimeRange              `json:"time_range"`
	Metrics   []MonitorQualityMetric `json:"metrics"`
}

// QualityMetric 质量指标
type MonitorQualityMetric struct {
	MetricName    string    `json:"metric_name"`
	MetricValue   float64   `json:"metric_value"`
	Unit          string    `json:"unit"`
	Description   string    `json:"description"`
	Threshold     float64   `json:"threshold"`
	IsHealthy     bool      `json:"is_healthy"`
	CollectedTime time.Time `json:"collected_time"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

// AlertInfo 告警信息
type AlertInfo struct {
	AlertID     string    `json:"alert_id"`
	AlertType   string    `json:"alert_type"` // performance, quality, error
	Severity    string    `json:"severity"`   // low, medium, high, critical
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TaskID      string    `json:"task_id,omitempty"`
	ExecutionID string    `json:"execution_id,omitempty"`
	MetricName  string    `json:"metric_name,omitempty"`
	MetricValue float64   `json:"metric_value,omitempty"`
	Threshold   float64   `json:"threshold,omitempty"`
	TriggeredAt time.Time `json:"triggered_at"`
}

// SystemHealthResponse 系统健康响应
type SystemHealthResponse struct {
	OverallHealth HealthStatus      `json:"overall_health"`
	Components    []ComponentHealth `json:"components"`
	CheckTime     time.Time         `json:"check_time"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status      string    `json:"status"` // healthy, warning, critical
	Score       float64   `json:"score"`  // 0-100
	Description string    `json:"description"`
	CheckTime   time.Time `json:"check_time"`
}

// ComponentHealth 组件健康状况
type ComponentHealth struct {
	ComponentName string             `json:"component_name"`
	Status        string             `json:"status"`
	Score         float64            `json:"score"`
	Metrics       map[string]float64 `json:"metrics"`
	LastCheckTime time.Time          `json:"last_check_time"`
}
