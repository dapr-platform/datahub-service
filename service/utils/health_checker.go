/**
 * @module health_checker
 * @description 健康检查器模块，负责对连接池进行定期健康检查
 * @architecture 健康检查器模式，定期检查连接健康状态
 * @documentReference 参考 ai_docs/basic_library_process_impl.md 第8.1节
 * @stateFlow 健康检查：启动 -> 检查 -> 记录状态 -> 等待间隔 -> 检查...
 * @rules
 *   - 健康检查间隔需要可配置
 *   - 检查超时需要设置合理值
 *   - 检查失败需要记录并重试
 *   - 健康状态需要持久化记录
 * @dependencies
 *   - context: 上下文控制
 *   - time: 时间处理
 *   - sync: 并发控制
 * @refs
 *   - service/utils/connection_pool.go: 连接池管理
 */

package utils

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus 健康状态
type HealthStatus struct {
	IsHealthy   bool      `json:"is_healthy"`
	LastCheck   time.Time `json:"last_check"`
	LastError   string    `json:"last_error,omitempty"`
	CheckCount  int       `json:"check_count"`
	FailCount   int       `json:"fail_count"`
	SuccessRate float64   `json:"success_rate"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	pool          interface{} // 连接池实例
	checkInterval time.Duration
	checkTimeout  time.Duration
	healthStatus  map[string]map[string]*HealthStatus // type -> name -> status
	mutex         sync.RWMutex
	stopChan      chan struct{}
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(pool interface{}, interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		pool:          pool,
		checkInterval: interval,
		checkTimeout:  timeout,
		healthStatus:  make(map[string]map[string]*HealthStatus),
		stopChan:      make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopChan:
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

// performHealthCheck 执行健康检查
func (hc *HealthChecker) performHealthCheck() {
	if cp, ok := hc.pool.(*ConnectionPool); ok {
		hc.checkDatabasePools(cp)
		hc.checkRedisPools(cp)
		hc.checkHTTPClients(cp)
	}
}

// checkDatabasePools 检查数据库连接池
func (hc *HealthChecker) checkDatabasePools(cp *ConnectionPool) {
	cp.mutex.RLock()
	pools := make(map[string]*sql.DB)
	for name, pool := range cp.dbPools {
		pools[name] = pool
	}
	cp.mutex.RUnlock()

	for name, pool := range pools {
		hc.checkDatabasePool(name, pool)
	}
}

// checkDatabasePool 检查单个数据库连接池
func (hc *HealthChecker) checkDatabasePool(name string, pool *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.checkTimeout)
	defer cancel()

	isHealthy := true
	var errorMsg string

	// 执行简单的ping检查
	if err := pool.PingContext(ctx); err != nil {
		isHealthy = false
		errorMsg = fmt.Sprintf("数据库ping失败: %v", err)
	}

	hc.updateHealthStatus("db", name, isHealthy, errorMsg)
}

// checkRedisPools 检查Redis连接池（这里简化处理，实际需要Redis客户端）
func (hc *HealthChecker) checkRedisPools(cp *ConnectionPool) {
	cp.mutex.RLock()
	pools := make(map[string]interface{})
	for name := range cp.redisPools {
		pools[name] = cp.redisPools[name]
	}
	cp.mutex.RUnlock()

	for name := range pools {
		hc.checkRedisPool(name)
	}
}

// checkRedisPool 检查单个Redis连接池
func (hc *HealthChecker) checkRedisPool(name string) {
	// 这里简化处理，假设Redis连接是健康的
	// 实际实现中需要使用Redis客户端的Ping方法
	hc.updateHealthStatus("redis", name, true, "")
}

// checkHTTPClients 检查HTTP客户端
func (hc *HealthChecker) checkHTTPClients(cp *ConnectionPool) {
	cp.mutex.RLock()
	clients := make(map[string]*http.Client)
	for name, client := range cp.httpClients {
		clients[name] = client
	}
	cp.mutex.RUnlock()

	for name, client := range clients {
		hc.checkHTTPClient(name, client)
	}
}

// checkHTTPClient 检查单个HTTP客户端
func (hc *HealthChecker) checkHTTPClient(name string, client *http.Client) {
	// HTTP客户端本身没有连接状态，这里简化处理
	// 实际实现中可以尝试发送一个测试请求
	hc.updateHealthStatus("http", name, true, "")
}

// updateHealthStatus 更新健康状态
func (hc *HealthChecker) updateHealthStatus(poolType, name string, isHealthy bool, errorMsg string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if hc.healthStatus[poolType] == nil {
		hc.healthStatus[poolType] = make(map[string]*HealthStatus)
	}

	status := hc.healthStatus[poolType][name]
	if status == nil {
		status = &HealthStatus{}
		hc.healthStatus[poolType][name] = status
	}

	status.IsHealthy = isHealthy
	status.LastCheck = time.Now()
	status.CheckCount++

	if !isHealthy {
		status.FailCount++
		status.LastError = errorMsg
	}

	// 计算成功率
	if status.CheckCount > 0 {
		status.SuccessRate = float64(status.CheckCount-status.FailCount) / float64(status.CheckCount)
	}
}

// IsHealthy 检查指定连接池是否健康
func (hc *HealthChecker) IsHealthy(poolType, name string) bool {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	if typeStatus, exists := hc.healthStatus[poolType]; exists {
		if status, exists := typeStatus[name]; exists {
			return status.IsHealthy
		}
	}

	return true // 默认为健康状态
}

// GetLastCheck 获取最后检查时间
func (hc *HealthChecker) GetLastCheck(poolType, name string) time.Time {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	if typeStatus, exists := hc.healthStatus[poolType]; exists {
		if status, exists := typeStatus[name]; exists {
			return status.LastCheck
		}
	}

	return time.Time{}
}

// GetHealthStatus 获取健康状态
func (hc *HealthChecker) GetHealthStatus(poolType, name string) *HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	if typeStatus, exists := hc.healthStatus[poolType]; exists {
		if status, exists := typeStatus[name]; exists {
			// 返回副本避免并发修改
			statusCopy := *status
			return &statusCopy
		}
	}

	return nil
}

// GetAllHealthStatus 获取所有健康状态
func (hc *HealthChecker) GetAllHealthStatus() map[string]map[string]*HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	// 返回深度副本
	result := make(map[string]map[string]*HealthStatus)
	for poolType, typeStatus := range hc.healthStatus {
		result[poolType] = make(map[string]*HealthStatus)
		for name, status := range typeStatus {
			statusCopy := *status
			result[poolType][name] = &statusCopy
		}
	}

	return result
}

// UpdateConfig 更新配置
func (hc *HealthChecker) UpdateConfig(interval, timeout time.Duration) {
	hc.checkInterval = interval
	hc.checkTimeout = timeout
}
