/*
 * @module service/basic_library/datasource/manager
 * @description 数据源管理器实现，负责数据源的注册、管理和生命周期控制
 * @architecture 单例模式 + 工厂模式 - 统一管理所有数据源实例，支持常驻数据源自动管理
 * @documentReference ai_docs/datasource_req.md, ai_docs/datasource_req1.md
 * @stateFlow 管理器生命周期：初始化 -> 注册数据源 -> 启动常驻源 -> 监控健康 -> 自动重连 -> 停止清理
 * @rules 常驻数据源自动启动并保持连接，实时流数据源持续监听消息更新
 * @dependencies context, sync, log, time
 * @refs interface.go, base.go, service/models/basic_library.go
 */

package datasource

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"datahub-service/service/models"
)

// DefaultDataSourceManager 默认数据源管理器实现
type DefaultDataSourceManager struct {
	mu              sync.RWMutex
	dataSources     map[string]DataSourceInterface
	dataSourceStats map[string]*DataSourceStatus
	factory         DataSourceFactory
	logger          *log.Logger

	// 监控和管理相关
	ctx               context.Context
	cancel            context.CancelFunc
	healthCheckTicker *time.Ticker
	reconnectTicker   *time.Ticker
	isRunning         bool

	// 配置选项
	healthCheckInterval  time.Duration
	reconnectInterval    time.Duration
	maxReconnectAttempts int
}

// NewDefaultDataSourceManager 创建默认数据源管理器
func NewDefaultDataSourceManager(factory DataSourceFactory) *DefaultDataSourceManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &DefaultDataSourceManager{
		dataSources:          make(map[string]DataSourceInterface),
		dataSourceStats:      make(map[string]*DataSourceStatus),
		factory:              factory,
		logger:               log.Default(),
		ctx:                  ctx,
		cancel:               cancel,
		healthCheckInterval:  30 * time.Second, // 30秒健康检查
		reconnectInterval:    5 * time.Minute,  // 5分钟重连检查
		maxReconnectAttempts: 3,
		isRunning:            false,
	}

	// 启动后台监控
	go manager.startBackgroundMonitoring()

	return manager
}

// Register 注册数据源实例
func (m *DefaultDataSourceManager) Register(ctx context.Context, ds *models.DataSource) error {
	if ds == nil {
		return fmt.Errorf("数据源配置不能为空")
	}

	if ds.ID == "" {
		return fmt.Errorf("数据源ID不能为空")
	}

	if ds.Type == "" {
		return fmt.Errorf("数据源类型不能为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if _, exists := m.dataSources[ds.ID]; exists {
		return fmt.Errorf("数据源 %s 已存在", ds.ID)
	}

	// 创建数据源实例
	instance, err := m.factory.Create(ds.Type)
	if err != nil {
		return fmt.Errorf("创建数据源实例失败: %v", err)
	}

	// 初始化数据源
	if err := instance.Init(ctx, ds); err != nil {
		return fmt.Errorf("初始化数据源失败: %v", err)
	}

	// 创建数据源状态记录
	status := &DataSourceStatus{
		ID:                ds.ID,
		Type:              ds.Type,
		Name:              ds.Name,
		IsResident:        instance.IsResident(),
		IsInitialized:     true,
		IsStarted:         false,
		UsageCount:        0,
		ReconnectAttempts: 0,
		MaxReconnects:     m.maxReconnectAttempts,
		AutoRestart:       instance.IsResident(), // 常驻数据源默认自动重启
		Metadata:          make(map[string]interface{}),
	}

	// 如果是常驻数据源，立即启动并保持连接
	if instance.IsResident() {
		m.logger.Printf("检测到常驻数据源 %s (%s)，准备启动...", ds.ID, ds.Type)

		// 检查是否已经启动
		if instance.IsStarted() {
			m.logger.Printf("常驻数据源 %s (%s) 已经启动，跳过启动步骤", ds.ID, ds.Type)
			status.IsStarted = true
			status.StartedAt = time.Now()
			status.HealthStatus = "online"
		} else {
			m.logger.Printf("开始启动常驻数据源 %s (%s)...", ds.ID, ds.Type)
			if err := instance.Start(ctx); err != nil {
				status.ErrorMessage = fmt.Sprintf("启动失败: %v", err)
				status.HealthStatus = "error"
				m.logger.Printf("常驻数据源 %s (%s) 启动失败: %v", ds.ID, ds.Type, err)
				// 注册失败的数据源也要保存，以便后续重试
			} else {
				status.IsStarted = true
				status.StartedAt = time.Now()
				status.HealthStatus = "online"
				m.logger.Printf("常驻数据源 %s (%s) 启动成功", ds.ID, ds.Type)
			}
		}
		status.LastHealthCheck = time.Now()
	} else {
		m.logger.Printf("数据源 %s (%s) 为非常驻数据源，设置为ready状态", ds.ID, ds.Type)
		status.HealthStatus = "ready"
	}

	// 注册到管理器
	m.dataSources[ds.ID] = instance
	m.dataSourceStats[ds.ID] = status
	m.logger.Printf("数据源 %s (%s) 注册成功", ds.ID, ds.Type)

	return nil
}

// Get 获取数据源实例
func (m *DefaultDataSourceManager) Get(dsID string) (DataSourceInterface, error) {
	if dsID == "" {
		return nil, fmt.Errorf("数据源ID不能为空")
	}

	m.mu.RLock()
	instance, exists := m.dataSources[dsID]
	status, statusExists := m.dataSourceStats[dsID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("数据源 %s 不存在", dsID)
	}

	// 更新使用统计
	if statusExists {
		m.mu.Lock()
		status.LastUsed = time.Now()
		status.UsageCount++
		m.mu.Unlock()
	}

	return instance, nil
}

// CreateInstance 创建数据源实例（不注册到管理器中，用于测试）
func (m *DefaultDataSourceManager) CreateInstance(dsType string) (DataSourceInterface, error) {
	if dsType == "" {
		return nil, fmt.Errorf("数据源类型不能为空")
	}

	// 直接使用工厂创建实例，不注册到管理器中
	instance, err := m.factory.Create(dsType)
	if err != nil {
		return nil, fmt.Errorf("创建数据源实例失败: %v", err)
	}

	return instance, nil
}

// CreateTestInstance 创建测试数据源实例（非常驻模式，用于连接测试）
func (m *DefaultDataSourceManager) CreateTestInstance(dsType string) (DataSourceInterface, error) {
	if dsType == "" {
		return nil, fmt.Errorf("数据源类型不能为空")
	}

	// 使用工厂创建实例
	instance, err := m.factory.Create(dsType)
	if err != nil {
		return nil, fmt.Errorf("创建数据源实例失败: %v", err)
	}

	// 对于所有具体实现，都设置为非常驻模式
	switch v := instance.(type) {
	case *PostgreSQLDataSource:
		v.BaseDataSource.SetResident(false)
	case *HTTPAuthDataSource:
		v.BaseDataSource.SetResident(false)
	case *HTTPNoAuthDataSource:
		v.BaseDataSource.SetResident(false)
	default:
		// 如果是其他类型的BaseDataSource子类，尝试直接设置
		if baseDS, ok := instance.(*BaseDataSource); ok {
			baseDS.SetResident(false)
		}
	}

	return instance, nil
}

// Remove 移除数据源实例
func (m *DefaultDataSourceManager) Remove(dsID string) error {
	if dsID == "" {
		return fmt.Errorf("数据源ID不能为空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.dataSources[dsID]
	if !exists {
		return fmt.Errorf("数据源 %s 不存在", dsID)
	}

	// 停止数据源
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := instance.Stop(ctx); err != nil {
		m.logger.Printf("停止数据源 %s 时发生错误: %v", dsID, err)
	}

	// 从管理器中移除
	delete(m.dataSources, dsID)
	delete(m.dataSourceStats, dsID)
	m.logger.Printf("数据源 %s 已移除", dsID)

	return nil
}

// List 列出所有注册的数据源
func (m *DefaultDataSourceManager) List() map[string]DataSourceInterface {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建副本避免并发问题
	result := make(map[string]DataSourceInterface, len(m.dataSources))
	for id, instance := range m.dataSources {
		result[id] = instance
	}

	return result
}

// StartAll 启动所有常驻数据源
func (m *DefaultDataSourceManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	dataSources := make(map[string]DataSourceInterface, len(m.dataSources))
	dataSourceStats := make(map[string]*DataSourceStatus, len(m.dataSourceStats))
	for id, instance := range m.dataSources {
		dataSources[id] = instance
		if stat, exists := m.dataSourceStats[id]; exists {
			dataSourceStats[id] = stat
		}
	}
	m.mu.RUnlock()

	m.logger.Printf("开始启动所有常驻数据源，共找到 %d 个数据源", len(dataSources))

	var errors []string
	residentCount := 0
	alreadyStartedCount := 0

	for id, instance := range dataSources {
		if instance.IsResident() {
			residentCount++
			m.logger.Printf("处理常驻数据源 %s，类型: %s", id, instance.GetType())

			// 检查状态
			if stat, exists := dataSourceStats[id]; exists {
				m.logger.Printf("数据源 %s 当前状态: IsStarted=%v, HealthStatus=%s", id, stat.IsStarted, stat.HealthStatus)
			}

			// 检查是否已经启动
			if instance.IsStarted() {
				m.logger.Printf("常驻数据源 %s 已经启动，跳过启动", id)
				alreadyStartedCount++
				continue
			}

			m.logger.Printf("开始启动常驻数据源 %s...", id)
			if err := instance.Start(ctx); err != nil {
				errMsg := fmt.Sprintf("启动常驻数据源 %s 失败: %v", id, err)
				errors = append(errors, errMsg)
				m.logger.Print(errMsg)

				// 更新状态
				m.mu.Lock()
				if stat, exists := m.dataSourceStats[id]; exists {
					stat.ErrorMessage = err.Error()
					stat.HealthStatus = "error"
				}
				m.mu.Unlock()
			} else {
				m.logger.Printf("常驻数据源 %s 启动成功", id)

				// 更新状态
				m.mu.Lock()
				if stat, exists := m.dataSourceStats[id]; exists {
					stat.IsStarted = true
					stat.StartedAt = time.Now()
					stat.HealthStatus = "online"
					stat.ErrorMessage = ""
				}
				m.mu.Unlock()
			}
		} else {
			m.logger.Printf("数据源 %s 为非常驻数据源，跳过启动", id)
		}
	}

	m.logger.Printf("常驻数据源启动统计: 总数=%d, 常驻数据源=%d, 已启动=%d, 错误=%d",
		len(dataSources), residentCount, alreadyStartedCount, len(errors))

	if len(errors) > 0 {
		return fmt.Errorf("启动部分数据源失败: %v", errors)
	}

	return nil
}

// StopAll 停止所有数据源
func (m *DefaultDataSourceManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	dataSources := make(map[string]DataSourceInterface, len(m.dataSources))
	for id, instance := range m.dataSources {
		dataSources[id] = instance
	}
	m.mu.RUnlock()

	var errors []string
	for id, instance := range dataSources {
		if err := instance.Stop(ctx); err != nil {
			errMsg := fmt.Sprintf("停止数据源 %s 失败: %v", id, err)
			errors = append(errors, errMsg)
			m.logger.Print(errMsg)
		} else {
			m.logger.Printf("数据源 %s 停止成功", id)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("停止部分数据源失败: %v", errors)
	}

	return nil
}

// HealthCheckAll 对所有数据源进行健康检查
func (m *DefaultDataSourceManager) HealthCheckAll(ctx context.Context) map[string]*HealthStatus {
	m.mu.RLock()
	dataSources := make(map[string]DataSourceInterface, len(m.dataSources))
	for id, instance := range m.dataSources {
		dataSources[id] = instance
	}
	m.mu.RUnlock()

	results := make(map[string]*HealthStatus, len(dataSources))

	// 使用并发进行健康检查
	var wg sync.WaitGroup
	var resultMu sync.Mutex

	for id, instance := range dataSources {
		wg.Add(1)
		go func(dsID string, ds DataSourceInterface) {
			defer wg.Done()

			status, err := ds.HealthCheck(ctx)
			if err != nil {
				status = &HealthStatus{
					Status:    "error",
					Message:   fmt.Sprintf("健康检查失败: %v", err),
					LastCheck: time.Now(),
				}
			}

			resultMu.Lock()
			results[dsID] = status
			resultMu.Unlock()
		}(id, instance)
	}

	wg.Wait()
	return results
}

// ExecuteDataSource 执行数据源操作（便捷方法）
func (m *DefaultDataSourceManager) ExecuteDataSource(ctx context.Context, dsID string, request *ExecuteRequest) (*ExecuteResponse, error) {
	instance, err := m.Get(dsID)
	if err != nil {
		return nil, err
	}

	// 对于非常驻数据源，需要先启动
	if !instance.IsResident() {
		if err := instance.Start(ctx); err != nil {
			return nil, fmt.Errorf("启动数据源失败: %v", err)
		}
		// 执行完成后停止
		defer func() {
			if stopErr := instance.Stop(ctx); stopErr != nil {
				m.logger.Printf("停止数据源 %s 时发生错误: %v", dsID, stopErr)
			}
		}()
	}

	return instance.Execute(ctx, request)
}

// GetStatistics 获取管理器统计信息
func (m *DefaultDataSourceManager) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_count"] = len(m.dataSources)

	typeCount := make(map[string]int)
	residentCount := 0
	onlineCount := 0

	for _, instance := range m.dataSources {
		// 统计类型
		dsType := instance.GetType()
		typeCount[dsType]++

		// 统计常驻数据源
		if instance.IsResident() {
			residentCount++
		}

		// 简单的在线检查（不执行完整健康检查）
		if baseDS, ok := instance.(*BaseDataSource); ok {
			if baseDS.IsInitialized() && (baseDS.IsStarted() || !baseDS.IsResident()) {
				onlineCount++
			}
		}
	}

	stats["type_distribution"] = typeCount
	stats["resident_count"] = residentCount
	stats["online_count"] = onlineCount
	stats["supported_types"] = m.factory.GetSupportedTypes()

	return stats
}

// startBackgroundMonitoring 启动后台监控
func (m *DefaultDataSourceManager) startBackgroundMonitoring() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.healthCheckTicker = time.NewTicker(m.healthCheckInterval)
	m.reconnectTicker = time.NewTicker(m.reconnectInterval)
	m.mu.Unlock()

	m.logger.Printf("数据源管理器后台监控已启动")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Printf("数据源管理器后台监控停止")
			return
		case <-m.healthCheckTicker.C:
			m.performHealthCheck()
		case <-m.reconnectTicker.C:
			m.performReconnection()
		}
	}
}

// performHealthCheck 执行健康检查
func (m *DefaultDataSourceManager) performHealthCheck() {
	m.mu.RLock()
	dataSources := make(map[string]DataSourceInterface)
	for id, ds := range m.dataSources {
		dataSources[id] = ds
	}
	m.mu.RUnlock()

	for id, instance := range dataSources {
		go func(dsID string, ds DataSourceInterface) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			status, err := ds.HealthCheck(ctx)

			m.mu.Lock()
			if dsStatus, exists := m.dataSourceStats[dsID]; exists {
				dsStatus.LastHealthCheck = time.Now()
				if err != nil {
					dsStatus.HealthStatus = "error"
					dsStatus.ErrorMessage = err.Error()
				} else {
					dsStatus.HealthStatus = status.Status
					dsStatus.ErrorMessage = ""
				}
			}
			m.mu.Unlock()

			if err != nil {
				m.logger.Printf("数据源 %s 健康检查失败: %v", dsID, err)
			}
		}(id, instance)
	}
}

// performReconnection 执行重连检查
func (m *DefaultDataSourceManager) performReconnection() {
	m.mu.RLock()
	needReconnect := make([]string, 0)
	for id, status := range m.dataSourceStats {
		if status.IsResident && !status.IsStarted && status.HealthStatus == "error" {
			needReconnect = append(needReconnect, id)
		}
	}
	m.mu.RUnlock()

	for _, dsID := range needReconnect {
		go m.attemptReconnect(dsID)
	}
}

// attemptReconnect 尝试重连数据源
func (m *DefaultDataSourceManager) attemptReconnect(dsID string) {
	m.mu.RLock()
	instance, exists := m.dataSources[dsID]
	status, statusExists := m.dataSourceStats[dsID]
	m.mu.RUnlock()

	if !exists || !statusExists {
		return
	}

	// 检查是否达到最大重连次数
	if status.ReconnectAttempts >= status.MaxReconnects {
		m.logger.Printf("数据源 %s 已达到最大重连次数 %d，停止重连", dsID, status.MaxReconnects)
		return
	}

	m.logger.Printf("尝试重连数据源 %s (第 %d/%d 次)", dsID, status.ReconnectAttempts+1, status.MaxReconnects)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 增加重连尝试次数
	m.mu.Lock()
	status.ReconnectAttempts++
	m.mu.Unlock()

	// 尝试停止然后重新启动
	if status.IsStarted {
		if err := instance.Stop(ctx); err != nil {
			m.logger.Printf("停止数据源 %s 失败: %v", dsID, err)
		}
	}

	// 尝试启动
	if err := instance.Start(ctx); err != nil {
		m.mu.Lock()
		status.ErrorMessage = fmt.Sprintf("重连失败 (第%d次): %v", status.ReconnectAttempts, err)
		status.HealthStatus = "error"
		m.mu.Unlock()
		m.logger.Printf("数据源 %s 重连失败 (第%d次): %v", dsID, status.ReconnectAttempts, err)
	} else {
		m.mu.Lock()
		status.IsStarted = true
		status.StartedAt = time.Now()
		status.HealthStatus = "online"
		status.ErrorMessage = ""
		m.mu.Unlock()
		m.logger.Printf("数据源 %s 重连成功", dsID)
	}
}

// GetDataSourceStatus 获取数据源状态
func (m *DefaultDataSourceManager) GetDataSourceStatus(dsID string) (*DataSourceStatus, error) {
	if dsID == "" {
		return nil, fmt.Errorf("数据源ID不能为空")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.dataSourceStats[dsID]
	if !exists {
		return nil, fmt.Errorf("数据源 %s 不存在", dsID)
	}

	// 返回状态的副本
	statusCopy := *status
	return &statusCopy, nil
}

// GetAllDataSourceStatus 获取所有数据源状态
func (m *DefaultDataSourceManager) GetAllDataSourceStatus() map[string]*DataSourceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*DataSourceStatus, len(m.dataSourceStats))
	for id, status := range m.dataSourceStats {
		statusCopy := *status
		result[id] = &statusCopy
	}

	return result
}

// Shutdown 关闭管理器
func (m *DefaultDataSourceManager) Shutdown() error {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return nil
	}

	m.cancel()
	if m.healthCheckTicker != nil {
		m.healthCheckTicker.Stop()
	}
	if m.reconnectTicker != nil {
		m.reconnectTicker.Stop()
	}
	m.isRunning = false
	m.mu.Unlock()

	// 停止所有数据源
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return m.StopAll(ctx)
}

// RestartResidentDataSource 重启指定的常驻数据源
func (m *DefaultDataSourceManager) RestartResidentDataSource(ctx context.Context, dsID string) error {
	m.mu.RLock()
	instance, exists := m.dataSources[dsID]
	status, statusExists := m.dataSourceStats[dsID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("数据源 %s 不存在", dsID)
	}

	if !statusExists {
		return fmt.Errorf("数据源 %s 状态不存在", dsID)
	}

	if !status.IsResident {
		return fmt.Errorf("数据源 %s 不是常驻数据源", dsID)
	}

	m.logger.Printf("重启常驻数据源 %s", dsID)

	// 停止数据源
	if status.IsStarted {
		if err := instance.Stop(ctx); err != nil {
			return fmt.Errorf("停止数据源失败: %w", err)
		}
	}

	// 重置重连计数
	m.mu.Lock()
	status.ReconnectAttempts = 0
	status.ErrorMessage = ""
	m.mu.Unlock()

	// 启动数据源
	if err := instance.Start(ctx); err != nil {
		m.mu.Lock()
		status.ErrorMessage = fmt.Sprintf("重启失败: %v", err)
		status.HealthStatus = "error"
		status.IsStarted = false
		m.mu.Unlock()
		return fmt.Errorf("启动数据源失败: %w", err)
	}

	m.mu.Lock()
	status.IsStarted = true
	status.StartedAt = time.Now()
	status.HealthStatus = "online"
	status.LastHealthCheck = time.Now()
	m.mu.Unlock()

	m.logger.Printf("常驻数据源 %s 重启成功", dsID)
	return nil
}

// GetResidentDataSources 获取所有常驻数据源
func (m *DefaultDataSourceManager) GetResidentDataSources() map[string]*DataSourceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	residentSources := make(map[string]*DataSourceStatus)
	for id, status := range m.dataSourceStats {
		if status.IsResident {
			// 创建副本以避免并发修改
			statusCopy := *status
			residentSources[id] = &statusCopy
		}
	}

	return residentSources
}

// SetAutoRestart 设置数据源的自动重启功能
func (m *DefaultDataSourceManager) SetAutoRestart(dsID string, autoRestart bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.dataSourceStats[dsID]
	if !exists {
		return fmt.Errorf("数据源 %s 不存在", dsID)
	}

	status.AutoRestart = autoRestart
	m.logger.Printf("数据源 %s 自动重启设置为 %v", dsID, autoRestart)
	return nil
}

// ResetReconnectAttempts 重置数据源的重连尝试次数
func (m *DefaultDataSourceManager) ResetReconnectAttempts(dsID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.dataSourceStats[dsID]
	if !exists {
		return fmt.Errorf("数据源 %s 不存在", dsID)
	}

	status.ReconnectAttempts = 0
	status.ErrorMessage = ""
	m.logger.Printf("数据源 %s 重连尝试次数已重置", dsID)
	return nil
}

// UpdateConnectionPool 更新数据源的连接池配置
func (m *DefaultDataSourceManager) UpdateConnectionPool(dsID string, poolConfig *ConnectionPoolConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.dataSourceStats[dsID]
	if !exists {
		return fmt.Errorf("数据源 %s 不存在", dsID)
	}

	status.ConnectionPool = poolConfig
	m.logger.Printf("数据源 %s 连接池配置已更新", dsID)
	return nil
}

// GetDataSourceMetrics 获取数据源的性能指标
func (m *DefaultDataSourceManager) GetDataSourceMetrics(dsID string) (map[string]interface{}, error) {
	m.mu.RLock()
	status, exists := m.dataSourceStats[dsID]
	instance, instanceExists := m.dataSources[dsID]
	m.mu.RUnlock()

	if !exists || !instanceExists {
		return nil, fmt.Errorf("数据源 %s 不存在", dsID)
	}

	metrics := map[string]interface{}{
		"id":                 status.ID,
		"type":               status.Type,
		"name":               status.Name,
		"is_resident":        status.IsResident,
		"is_started":         status.IsStarted,
		"usage_count":        status.UsageCount,
		"reconnect_attempts": status.ReconnectAttempts,
		"max_reconnects":     status.MaxReconnects,
		"auto_restart":       status.AutoRestart,
		"health_status":      status.HealthStatus,
		"last_health_check":  status.LastHealthCheck,
		"started_at":         status.StartedAt,
		"last_used":          status.LastUsed,
	}

	if status.ConnectionPool != nil {
		metrics["connection_pool"] = map[string]interface{}{
			"max_connections": status.ConnectionPool.MaxConnections,
			"min_connections": status.ConnectionPool.MinConnections,
			"idle_timeout":    status.ConnectionPool.IdleTimeout,
			"max_lifetime":    status.ConnectionPool.MaxLifetime,
		}
	}

	// 如果数据源支持额外的指标，添加它们
	if metricsProvider, ok := instance.(interface {
		GetMetrics() map[string]interface{}
	}); ok {
		additionalMetrics := metricsProvider.GetMetrics()
		for k, v := range additionalMetrics {
			metrics[k] = v
		}
	}

	return metrics, nil
}
