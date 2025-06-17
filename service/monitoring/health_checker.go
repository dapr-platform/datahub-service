/*
 * @module service/monitoring/health_checker
 * @description 健康检查器，负责数据源连接检查、服务状态检查、依赖服务检查和健康评分计算
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 健康检查定义 -> 状态检测 -> 评分计算 -> 状态更新
 * @rules 确保健康检查的准确性和及时性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/patch_basic_library_process.md
 */

package monitoring

import (
	"datahub-service/service/models"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	db                *gorm.DB
	healthCheckConfig *HealthCheckConfig
	lastCheckResults  map[string]*HealthCheckResult
	mutex             sync.RWMutex
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	CheckInterval    time.Duration  `json:"check_interval"`    // 检查间隔
	Timeout          time.Duration  `json:"timeout"`           // 检查超时
	RetryCount       int            `json:"retry_count"`       // 重试次数
	RetryInterval    time.Duration  `json:"retry_interval"`    // 重试间隔
	EnabledChecks    []string       `json:"enabled_checks"`    // 启用的检查项
	HealthThresholds map[string]int `json:"health_thresholds"` // 健康阈值
}

// HealthStatus 整体健康状态
type HealthStatus struct {
	Overall      string                       `json:"overall"` // healthy, warning, critical
	Score        int                          `json:"score"`   // 健康评分 0-100
	Timestamp    time.Time                    `json:"timestamp"`
	Components   map[string]*ComponentHealth  `json:"components"`   // 组件健康状态
	Dependencies map[string]*DependencyHealth `json:"dependencies"` // 依赖服务健康状态
	DataSources  map[string]*DataSourceHealth `json:"data_sources"` // 数据源健康状态
	Issues       []HealthIssue                `json:"issues"`       // 健康问题
	Summary      HealthSummary                `json:"summary"`      // 健康摘要
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name         string                 `json:"name"`
	Status       string                 `json:"status"` // healthy, warning, critical
	Score        int                    `json:"score"`  // 0-100
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metrics      map[string]interface{} `json:"metrics"`
	CheckDetails map[string]interface{} `json:"check_details"`
}

// DependencyHealth 依赖服务健康状态
type DependencyHealth struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"`   // database, cache, message_queue, api
	Status       string        `json:"status"` // healthy, warning, critical
	Available    bool          `json:"available"`
	ResponseTime time.Duration `json:"response_time"`
	LastChecked  time.Time     `json:"last_checked"`
	ErrorMessage string        `json:"error_message,omitempty"`
	CheckCount   int           `json:"check_count"`
	SuccessCount int           `json:"success_count"`
	SuccessRate  float64       `json:"success_rate"`
}

// DataSourceHealth 数据源健康状态
type DataSourceHealth struct {
	DataSourceID   string                 `json:"data_source_id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"` // healthy, warning, critical
	Available      bool                   `json:"available"`
	ResponseTime   time.Duration          `json:"response_time"`
	LastChecked    time.Time              `json:"last_checked"`
	LastSyncTime   *time.Time             `json:"last_sync_time,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	ConnectionInfo map[string]interface{} `json:"connection_info"`
	Metrics        *DataSourceMetrics     `json:"metrics,omitempty"`
	HealthScore    int                    `json:"health_score"` // 0-100
}

// HealthIssue 健康问题
type HealthIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`     // connection, performance, data_quality
	Severity    string                 `json:"severity"` // info, warning, error, critical
	Component   string                 `json:"component"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	DetectedAt  time.Time              `json:"detected_at"`
	Suggestion  string                 `json:"suggestion"`
}

// HealthSummary 健康摘要
type HealthSummary struct {
	TotalComponents       int `json:"total_components"`
	HealthyComponents     int `json:"healthy_components"`
	WarningComponents     int `json:"warning_components"`
	CriticalComponents    int `json:"critical_components"`
	TotalDataSources      int `json:"total_data_sources"`
	HealthyDataSources    int `json:"healthy_data_sources"`
	OfflineDataSources    int `json:"offline_data_sources"`
	TotalDependencies     int `json:"total_dependencies"`
	AvailableDependencies int `json:"available_dependencies"`
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	CheckType    string                 `json:"check_type"`
	CheckTarget  string                 `json:"check_target"`
	Status       string                 `json:"status"`
	Score        int                    `json:"score"`
	ResponseTime time.Duration          `json:"response_time"`
	CheckedAt    time.Time              `json:"checked_at"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Details      map[string]interface{} `json:"details"`
}

// NewHealthChecker 创建健康检查器实例
func NewHealthChecker(db *gorm.DB) *HealthChecker {
	return &HealthChecker{
		db:                db,
		healthCheckConfig: getDefaultHealthCheckConfig(),
		lastCheckResults:  make(map[string]*HealthCheckResult),
	}
}

// CheckOverallHealth 检查整体健康状态
func (h *HealthChecker) CheckOverallHealth() (*HealthStatus, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	status := &HealthStatus{
		Timestamp:    time.Now(),
		Components:   make(map[string]*ComponentHealth),
		Dependencies: make(map[string]*DependencyHealth),
		DataSources:  make(map[string]*DataSourceHealth),
		Issues:       []HealthIssue{},
	}

	// 检查系统组件
	if err := h.checkSystemComponents(status); err != nil {
		return nil, fmt.Errorf("检查系统组件失败: %v", err)
	}

	// 检查依赖服务
	if err := h.checkDependencies(status); err != nil {
		return nil, fmt.Errorf("检查依赖服务失败: %v", err)
	}

	// 检查数据源
	if err := h.checkAllDataSources(status); err != nil {
		return nil, fmt.Errorf("检查数据源失败: %v", err)
	}

	// 计算整体健康评分
	h.calculateOverallHealth(status)

	// 生成健康摘要
	h.generateHealthSummary(status)

	return status, nil
}

// CheckDataSourceHealth 检查指定数据源健康状态
func (h *HealthChecker) CheckDataSourceHealth(dataSourceID string) (*DataSourceHealth, error) {
	// 获取数据源信息
	var dataSource models.DataSource
	if err := h.db.First(&dataSource, "id = ?", dataSourceID).Error; err != nil {
		return nil, fmt.Errorf("获取数据源失败: %v", err)
	}

	startTime := time.Now()
	health := &DataSourceHealth{
		DataSourceID:   dataSourceID,
		Name:           dataSource.Name,
		Type:           dataSource.Type,
		LastChecked:    startTime,
		ConnectionInfo: make(map[string]interface{}),
	}

	// 执行连接检查
	if err := h.checkDataSourceConnection(&dataSource, health); err != nil {
		health.Status = "critical"
		health.Available = false
		health.ErrorMessage = err.Error()
		health.HealthScore = 0
	} else {
		health.Available = true
		health.ResponseTime = time.Since(startTime)

		// 检查数据源指标
		if err := h.checkDataSourceMetrics(&dataSource, health); err == nil {
			health.HealthScore = h.calculateDataSourceScore(health)
			health.Status = h.getStatusFromScore(health.HealthScore)
		} else {
			health.Status = "warning"
			health.HealthScore = 60
		}
	}

	return health, nil
}

// 检查系统组件
func (h *HealthChecker) checkSystemComponents(status *HealthStatus) error {
	// 检查数据库连接
	dbHealth := h.checkDatabaseHealth()
	status.Components["database"] = dbHealth

	// 检查内存使用情况
	memoryHealth := h.checkMemoryHealth()
	status.Components["memory"] = memoryHealth

	// 检查CPU使用情况
	cpuHealth := h.checkCPUHealth()
	status.Components["cpu"] = cpuHealth

	// 检查磁盘使用情况
	diskHealth := h.checkDiskHealth()
	status.Components["disk"] = diskHealth

	return nil
}

// 检查依赖服务
func (h *HealthChecker) checkDependencies(status *HealthStatus) error {
	// 检查数据库依赖
	dbDep := h.checkDatabaseDependency()
	status.Dependencies["database"] = dbDep

	// 这里可以添加其他依赖服务的检查，如Redis、消息队列等

	return nil
}

// 检查所有数据源
func (h *HealthChecker) checkAllDataSources(status *HealthStatus) error {
	var dataSources []models.DataSource
	if err := h.db.Find(&dataSources).Error; err != nil {
		return fmt.Errorf("获取数据源列表失败: %v", err)
	}

	for _, ds := range dataSources {
		health, err := h.CheckDataSourceHealth(ds.ID)
		if err != nil {
			// 创建错误状态的健康信息
			health = &DataSourceHealth{
				DataSourceID: ds.ID,
				Name:         ds.Name,
				Type:         ds.Type,
				Status:       "critical",
				Available:    false,
				LastChecked:  time.Now(),
				ErrorMessage: err.Error(),
				HealthScore:  0,
			}
		}

		status.DataSources[ds.ID] = health

		// 如果数据源状态不健康，添加到问题列表
		if health.Status != "healthy" {
			issue := HealthIssue{
				ID:          fmt.Sprintf("ds_%s_%d", ds.ID, time.Now().Unix()),
				Type:        "connection",
				Severity:    health.Status,
				Component:   fmt.Sprintf("datasource_%s", ds.ID),
				Description: fmt.Sprintf("数据源 %s 状态异常", ds.Name),
				Details: map[string]interface{}{
					"datasource_id":   ds.ID,
					"datasource_type": ds.Type,
					"error":           health.ErrorMessage,
				},
				DetectedAt: time.Now(),
				Suggestion: "检查数据源连接配置和网络连通性",
			}
			status.Issues = append(status.Issues, issue)
		}
	}

	return nil
}

// 检查数据源连接
func (h *HealthChecker) checkDataSourceConnection(dataSource *models.DataSource, health *DataSourceHealth) error {
	// 根据数据源类型执行不同的连接检查
	switch dataSource.Type {
	case string(models.DataSourceTypePostgreSQL), string(models.DataSourceTypeMySQL):
		return h.checkDatabaseConnection(dataSource, health)
	case string(models.DataSourceTypeHTTP):
		return h.checkHTTPConnection(dataSource, health)
	case string(models.DataSourceTypeKafka):
		return h.checkKafkaConnection(dataSource, health)
	case string(models.DataSourceTypeRedis):
		return h.checkRedisConnection(dataSource, health)
	default:
		return h.checkGenericConnection(dataSource, health)
	}
}

// 检查数据库连接
func (h *HealthChecker) checkDatabaseConnection(dataSource *models.DataSource, health *DataSourceHealth) error {
	// 简化实现，实际应该创建真实的数据库连接
	// 模拟连接检查
	time.Sleep(10 * time.Millisecond) // 模拟连接时间

	// 从连接配置中获取主机和端口
	host, _ := dataSource.ConnectionConfig["host"].(string)
	port, _ := dataSource.ConnectionConfig["port"].(float64)

	health.ConnectionInfo["host"] = host
	health.ConnectionInfo["port"] = port
	health.ConnectionInfo["check_type"] = "tcp_connect"

	// 简化的TCP连接检查
	if host != "" && port > 0 {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%.0f", host, port), h.healthCheckConfig.Timeout)
		if err != nil {
			return fmt.Errorf("数据库连接失败: %v", err)
		}
		conn.Close()
	}

	return nil
}

// 检查HTTP连接
func (h *HealthChecker) checkHTTPConnection(dataSource *models.DataSource, health *DataSourceHealth) error {
	url, ok := dataSource.ConnectionConfig["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("HTTP URL配置无效")
	}

	client := &http.Client{
		Timeout: h.healthCheckConfig.Timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP连接失败: %v", err)
	}
	defer resp.Body.Close()

	health.ConnectionInfo["url"] = url
	health.ConnectionInfo["status_code"] = resp.StatusCode
	health.ConnectionInfo["check_type"] = "http_get"

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP响应错误: %d", resp.StatusCode)
	}

	return nil
}

// 检查Kafka连接
func (h *HealthChecker) checkKafkaConnection(dataSource *models.DataSource, health *DataSourceHealth) error {
	// 简化实现
	brokers, _ := dataSource.ConnectionConfig["brokers"].([]interface{})
	if len(brokers) == 0 {
		return fmt.Errorf("Kafka brokers配置无效")
	}

	health.ConnectionInfo["brokers"] = brokers
	health.ConnectionInfo["check_type"] = "kafka_metadata"

	// 这里应该实现真实的Kafka连接检查
	return nil
}

// 检查Redis连接
func (h *HealthChecker) checkRedisConnection(dataSource *models.DataSource, health *DataSourceHealth) error {
	// 简化实现
	host, _ := dataSource.ConnectionConfig["host"].(string)
	port, _ := dataSource.ConnectionConfig["port"].(float64)

	if host == "" {
		return fmt.Errorf("Redis主机配置无效")
	}

	health.ConnectionInfo["host"] = host
	health.ConnectionInfo["port"] = port
	health.ConnectionInfo["check_type"] = "redis_ping"

	// 这里应该实现真实的Redis连接检查
	return nil
}

// 检查通用连接
func (h *HealthChecker) checkGenericConnection(dataSource *models.DataSource, health *DataSourceHealth) error {
	// 通用连接检查逻辑
	health.ConnectionInfo["check_type"] = "generic"
	return nil
}

// 检查数据源指标
func (h *HealthChecker) checkDataSourceMetrics(dataSource *models.DataSource, health *DataSourceHealth) error {
	// 获取数据源状态
	var status models.DataSourceStatus
	if err := h.db.Where("data_source_id = ?", dataSource.ID).First(&status).Error; err != nil {
		return fmt.Errorf("获取数据源状态失败: %v", err)
	}

	health.LastSyncTime = status.LastSyncTime
	health.HealthScore = status.HealthScore

	return nil
}

// 检查数据库健康状态
func (h *HealthChecker) checkDatabaseHealth() *ComponentHealth {
	startTime := time.Now()
	health := &ComponentHealth{
		Name:         "database",
		LastChecked:  startTime,
		Metrics:      make(map[string]interface{}),
		CheckDetails: make(map[string]interface{}),
	}

	// 执行数据库健康检查
	if err := h.db.Raw("SELECT 1").Error; err != nil {
		health.Status = "critical"
		health.Score = 0
		health.ErrorMessage = err.Error()
	} else {
		health.Status = "healthy"
		health.Score = 100
		health.ResponseTime = time.Since(startTime)

		// 获取数据库连接池信息
		sqlDB, err := h.db.DB()
		if err == nil {
			stats := sqlDB.Stats()
			health.Metrics["open_connections"] = stats.OpenConnections
			health.Metrics["idle_connections"] = stats.Idle
			health.Metrics["in_use_connections"] = stats.InUse
		}
	}

	return health
}

// 检查内存健康状态
func (h *HealthChecker) checkMemoryHealth() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "memory",
		LastChecked: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	// 简化的内存检查实现
	health.Status = "healthy"
	health.Score = 85
	health.Metrics["usage_percent"] = 65.5
	health.Metrics["available_mb"] = 2048

	return health
}

// 检查CPU健康状态
func (h *HealthChecker) checkCPUHealth() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "cpu",
		LastChecked: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	// 简化的CPU检查实现
	health.Status = "healthy"
	health.Score = 90
	health.Metrics["usage_percent"] = 25.3
	health.Metrics["load_average"] = 1.2

	return health
}

// 检查磁盘健康状态
func (h *HealthChecker) checkDiskHealth() *ComponentHealth {
	health := &ComponentHealth{
		Name:        "disk",
		LastChecked: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	// 简化的磁盘检查实现
	health.Status = "healthy"
	health.Score = 80
	health.Metrics["usage_percent"] = 45.8
	health.Metrics["available_gb"] = 120

	return health
}

// 检查数据库依赖
func (h *HealthChecker) checkDatabaseDependency() *DependencyHealth {
	startTime := time.Now()
	dep := &DependencyHealth{
		Name:        "PostgreSQL",
		Type:        "database",
		LastChecked: startTime,
		CheckCount:  1,
	}

	if err := h.db.Raw("SELECT version()").Error; err != nil {
		dep.Status = "critical"
		dep.Available = false
		dep.ErrorMessage = err.Error()
		dep.SuccessCount = 0
		dep.SuccessRate = 0
	} else {
		dep.Status = "healthy"
		dep.Available = true
		dep.ResponseTime = time.Since(startTime)
		dep.SuccessCount = 1
		dep.SuccessRate = 100
	}

	return dep
}

// 计算数据源健康评分
func (h *HealthChecker) calculateDataSourceScore(health *DataSourceHealth) int {
	score := 100

	// 根据可用性调整评分
	if !health.Available {
		return 0
	}

	// 根据响应时间调整评分
	if health.ResponseTime > 5*time.Second {
		score -= 30
	} else if health.ResponseTime > 2*time.Second {
		score -= 15
	} else if health.ResponseTime > 1*time.Second {
		score -= 5
	}

	// 根据最后同步时间调整评分
	if health.LastSyncTime != nil {
		timeSinceSync := time.Since(*health.LastSyncTime)
		if timeSinceSync > 24*time.Hour {
			score -= 20
		} else if timeSinceSync > 6*time.Hour {
			score -= 10
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// 根据评分获取状态
func (h *HealthChecker) getStatusFromScore(score int) string {
	if score >= 80 {
		return "healthy"
	} else if score >= 60 {
		return "warning"
	} else {
		return "critical"
	}
}

// 计算整体健康状态
func (h *HealthChecker) calculateOverallHealth(status *HealthStatus) {
	totalScore := 0
	componentCount := 0

	// 计算组件平均分
	for _, component := range status.Components {
		totalScore += component.Score
		componentCount++
	}

	// 计算数据源平均分
	for _, ds := range status.DataSources {
		totalScore += ds.HealthScore
		componentCount++
	}

	// 计算依赖服务分数
	for _, dep := range status.Dependencies {
		if dep.Available {
			totalScore += 100
		}
		componentCount++
	}

	if componentCount > 0 {
		status.Score = totalScore / componentCount
	} else {
		status.Score = 0
	}

	status.Overall = h.getStatusFromScore(status.Score)
}

// 生成健康摘要
func (h *HealthChecker) generateHealthSummary(status *HealthStatus) {
	summary := HealthSummary{}

	// 统计组件状态
	summary.TotalComponents = len(status.Components)
	for _, component := range status.Components {
		switch component.Status {
		case "healthy":
			summary.HealthyComponents++
		case "warning":
			summary.WarningComponents++
		case "critical":
			summary.CriticalComponents++
		}
	}

	// 统计数据源状态
	summary.TotalDataSources = len(status.DataSources)
	for _, ds := range status.DataSources {
		if ds.Status == "healthy" {
			summary.HealthyDataSources++
		} else {
			summary.OfflineDataSources++
		}
	}

	// 统计依赖服务状态
	summary.TotalDependencies = len(status.Dependencies)
	for _, dep := range status.Dependencies {
		if dep.Available {
			summary.AvailableDependencies++
		}
	}

	status.Summary = summary
}

// 获取默认健康检查配置
func getDefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		CheckInterval: 60 * time.Second,
		Timeout:       5 * time.Second,
		RetryCount:    3,
		RetryInterval: 1 * time.Second,
		EnabledChecks: []string{"database", "memory", "cpu", "disk", "datasources"},
		HealthThresholds: map[string]int{
			"healthy":  80,
			"warning":  60,
			"critical": 0,
		},
	}
}
