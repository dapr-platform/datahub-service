/**
 * @module connection_pool
 * @description 连接池管理模块，负责管理数据库连接池、Redis连接池、HTTP客户端池、连接健康检查等功能
 * @architecture 连接池管理器模式，提供统一的连接管理接口
 * @documentReference 参考 ai_docs/basic_library_process_impl.md 第8.1节
 * @stateFlow 连接状态管理：创建 -> 活跃 -> 空闲 -> 超时/关闭
 * @rules
 *   - 连接池大小需要根据业务需求配置
 *   - 连接空闲超时需要自动回收
 *   - 连接健康检查需要定期执行
 *   - 连接异常需要及时清理和重建
 * @dependencies
 *   - database/sql: 数据库连接
 *   - redis: Redis客户端
 *   - net/http: HTTP客户端
 *   - sync: 并发控制
 * @refs
 *   - service/config/*: 配置管理
 *   - service/monitoring/*: 监控服务
 */

package utils

import (
	"log/slog"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	// 数据库连接池配置
	DBMaxOpenConns    int           `json:"db_max_open_conns"`
	DBMaxIdleConns    int           `json:"db_max_idle_conns"`
	DBConnMaxLifetime time.Duration `json:"db_conn_max_lifetime"`
	DBConnMaxIdleTime time.Duration `json:"db_conn_max_idle_time"`

	// Redis连接池配置
	RedisPoolSize     int           `json:"redis_pool_size"`
	RedisMinIdleConns int           `json:"redis_min_idle_conns"`
	RedisMaxConnAge   time.Duration `json:"redis_max_conn_age"`
	RedisIdleTimeout  time.Duration `json:"redis_idle_timeout"`

	// HTTP客户端池配置
	HTTPMaxIdleConns        int           `json:"http_max_idle_conns"`
	HTTPMaxIdleConnsPerHost int           `json:"http_max_idle_conns_per_host"`
	HTTPIdleConnTimeout     time.Duration `json:"http_idle_conn_timeout"`
	HTTPTimeout             time.Duration `json:"http_timeout"`

	// 健康检查配置
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
}

// DefaultConnectionPoolConfig 默认连接池配置
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		DBMaxOpenConns:          100,
		DBMaxIdleConns:          10,
		DBConnMaxLifetime:       time.Hour,
		DBConnMaxIdleTime:       time.Minute * 30,
		RedisPoolSize:           100,
		RedisMinIdleConns:       10,
		RedisMaxConnAge:         time.Hour,
		RedisIdleTimeout:        time.Minute * 30,
		HTTPMaxIdleConns:        100,
		HTTPMaxIdleConnsPerHost: 10,
		HTTPIdleConnTimeout:     time.Minute * 5,
		HTTPTimeout:             time.Second * 30,
		HealthCheckInterval:     time.Minute * 5,
		HealthCheckTimeout:      time.Second * 10,
	}
}

// ConnectionPool 连接池管理器
type ConnectionPool struct {
	config      *ConnectionPoolConfig
	dbPools     map[string]*sql.DB
	redisPools  map[string]*redis.Client
	httpClients map[string]*http.Client
	mutex       sync.RWMutex
	healthCheck *HealthChecker
	ctx         context.Context
	cancel      context.CancelFunc
}

// PoolStats 连接池统计信息
type PoolStats struct {
	Type        string    `json:"type"`         // 连接池类型
	Name        string    `json:"name"`         // 连接池名称
	MaxConns    int       `json:"max_conns"`    // 最大连接数
	ActiveConns int       `json:"active_conns"` // 活跃连接数
	IdleConns   int       `json:"idle_conns"`   // 空闲连接数
	WaitCount   int       `json:"wait_count"`   // 等待连接数
	IsHealthy   bool      `json:"is_healthy"`   // 健康状态
	LastCheck   time.Time `json:"last_check"`   // 最后检查时间
}

// NewConnectionPool 创建新的连接池管理器
func NewConnectionPool(config *ConnectionPoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		config:      config,
		dbPools:     make(map[string]*sql.DB),
		redisPools:  make(map[string]*redis.Client),
		httpClients: make(map[string]*http.Client),
		ctx:         ctx,
		cancel:      cancel,
	}

	pool.healthCheck = NewHealthChecker(pool, config.HealthCheckInterval, config.HealthCheckTimeout)

	return pool
}

// 数据库连接池管理

// GetDBPool 获取数据库连接池
func (cp *ConnectionPool) GetDBPool(name string) (*sql.DB, error) {
	cp.mutex.RLock()
	if pool, exists := cp.dbPools[name]; exists {
		cp.mutex.RUnlock()
		return pool, nil
	}
	cp.mutex.RUnlock()

	return nil, fmt.Errorf("数据库连接池 %s 不存在", name)
}

// AddDBPool 添加数据库连接池
func (cp *ConnectionPool) AddDBPool(name string, db *sql.DB) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 配置连接池参数
	db.SetMaxOpenConns(cp.config.DBMaxOpenConns)
	db.SetMaxIdleConns(cp.config.DBMaxIdleConns)
	db.SetConnMaxLifetime(cp.config.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(cp.config.DBConnMaxIdleTime)

	cp.dbPools[name] = db
	return nil
}

// RemoveDBPool 移除数据库连接池
func (cp *ConnectionPool) RemoveDBPool(name string) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if pool, exists := cp.dbPools[name]; exists {
		if err := pool.Close(); err != nil {
			return fmt.Errorf("关闭数据库连接池失败: %v", err)
		}
		delete(cp.dbPools, name)
	}

	return nil
}

// GetDBPoolStats 获取数据库连接池统计信息
func (cp *ConnectionPool) GetDBPoolStats(name string) (*PoolStats, error) {
	cp.mutex.RLock()
	pool, exists := cp.dbPools[name]
	cp.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("数据库连接池 %s 不存在", name)
	}

	stats := pool.Stats()
	return &PoolStats{
		Type:        "database",
		Name:        name,
		MaxConns:    stats.MaxOpenConnections,
		ActiveConns: stats.OpenConnections,
		IdleConns:   stats.Idle,
		WaitCount:   int(stats.WaitCount),
		IsHealthy:   cp.healthCheck.IsHealthy("db", name),
		LastCheck:   cp.healthCheck.GetLastCheck("db", name),
	}, nil
}

// Redis连接池管理

// GetRedisPool 获取Redis连接池
func (cp *ConnectionPool) GetRedisPool(name string) (*redis.Client, error) {
	cp.mutex.RLock()
	if pool, exists := cp.redisPools[name]; exists {
		cp.mutex.RUnlock()
		return pool, nil
	}
	cp.mutex.RUnlock()

	return nil, fmt.Errorf("Redis连接池 %s 不存在", name)
}

// AddRedisPool 添加Redis连接池
func (cp *ConnectionPool) AddRedisPool(name string, options *redis.Options) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 配置连接池参数
	if options.PoolSize == 0 {
		options.PoolSize = cp.config.RedisPoolSize
	}
	if options.MinIdleConns == 0 {
		options.MinIdleConns = cp.config.RedisMinIdleConns
	}
	if options.MaxConnAge == 0 {
		options.MaxConnAge = cp.config.RedisMaxConnAge
	}
	if options.IdleTimeout == 0 {
		options.IdleTimeout = cp.config.RedisIdleTimeout
	}

	client := redis.NewClient(options)
	cp.redisPools[name] = client

	return nil
}

// RemoveRedisPool 移除Redis连接池
func (cp *ConnectionPool) RemoveRedisPool(name string) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if pool, exists := cp.redisPools[name]; exists {
		if err := pool.Close(); err != nil {
			return fmt.Errorf("关闭Redis连接池失败: %v", err)
		}
		delete(cp.redisPools, name)
	}

	return nil
}

// GetRedisPoolStats 获取Redis连接池统计信息
func (cp *ConnectionPool) GetRedisPoolStats(name string) (*PoolStats, error) {
	cp.mutex.RLock()
	pool, exists := cp.redisPools[name]
	cp.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("Redis连接池 %s 不存在", name)
	}

	poolStats := pool.PoolStats()
	return &PoolStats{
		Type:        "redis",
		Name:        name,
		MaxConns:    int(poolStats.TotalConns),
		ActiveConns: int(poolStats.TotalConns - poolStats.IdleConns),
		IdleConns:   int(poolStats.IdleConns),
		WaitCount:   0, // Redis客户端没有提供等待计数
		IsHealthy:   cp.healthCheck.IsHealthy("redis", name),
		LastCheck:   cp.healthCheck.GetLastCheck("redis", name),
	}, nil
}

// HTTP客户端池管理

// GetHTTPClient 获取HTTP客户端
func (cp *ConnectionPool) GetHTTPClient(name string) (*http.Client, error) {
	cp.mutex.RLock()
	if client, exists := cp.httpClients[name]; exists {
		cp.mutex.RUnlock()
		return client, nil
	}
	cp.mutex.RUnlock()

	return nil, fmt.Errorf("HTTP客户端 %s 不存在", name)
}

// AddHTTPClient 添加HTTP客户端
func (cp *ConnectionPool) AddHTTPClient(name string, transport *http.Transport) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if transport == nil {
		transport = &http.Transport{}
	}

	// 配置连接池参数
	if transport.MaxIdleConns == 0 {
		transport.MaxIdleConns = cp.config.HTTPMaxIdleConns
	}
	if transport.MaxIdleConnsPerHost == 0 {
		transport.MaxIdleConnsPerHost = cp.config.HTTPMaxIdleConnsPerHost
	}
	if transport.IdleConnTimeout == 0 {
		transport.IdleConnTimeout = cp.config.HTTPIdleConnTimeout
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cp.config.HTTPTimeout,
	}

	cp.httpClients[name] = client
	return nil
}

// RemoveHTTPClient 移除HTTP客户端
func (cp *ConnectionPool) RemoveHTTPClient(name string) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if client, exists := cp.httpClients[name]; exists {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		delete(cp.httpClients, name)
	}

	return nil
}

// GetHTTPClientStats 获取HTTP客户端统计信息
func (cp *ConnectionPool) GetHTTPClientStats(name string) (*PoolStats, error) {
	cp.mutex.RLock()
	_, exists := cp.httpClients[name]
	cp.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("HTTP客户端 %s 不存在", name)
	}

	return &PoolStats{
		Type:        "http",
		Name:        name,
		MaxConns:    cp.config.HTTPMaxIdleConns,
		ActiveConns: 0, // HTTP客户端没有提供活跃连接统计
		IdleConns:   0, // HTTP客户端没有提供空闲连接统计
		WaitCount:   0, // HTTP客户端没有提供等待连接统计
		IsHealthy:   cp.healthCheck.IsHealthy("http", name),
		LastCheck:   cp.healthCheck.GetLastCheck("http", name),
	}, nil
}

// 通用管理接口

// GetAllStats 获取所有连接池统计信息
func (cp *ConnectionPool) GetAllStats() (map[string][]*PoolStats, error) {
	stats := make(map[string][]*PoolStats)

	// 数据库连接池统计
	cp.mutex.RLock()
	dbStats := make([]*PoolStats, 0, len(cp.dbPools))
	for name := range cp.dbPools {
		if stat, err := cp.GetDBPoolStats(name); err == nil {
			dbStats = append(dbStats, stat)
		}
	}
	stats["database"] = dbStats

	// Redis连接池统计
	redisStats := make([]*PoolStats, 0, len(cp.redisPools))
	for name := range cp.redisPools {
		if stat, err := cp.GetRedisPoolStats(name); err == nil {
			redisStats = append(redisStats, stat)
		}
	}
	stats["redis"] = redisStats

	// HTTP客户端统计
	httpStats := make([]*PoolStats, 0, len(cp.httpClients))
	for name := range cp.httpClients {
		if stat, err := cp.GetHTTPClientStats(name); err == nil {
			httpStats = append(httpStats, stat)
		}
	}
	stats["http"] = httpStats
	cp.mutex.RUnlock()

	return stats, nil
}

// Start 启动连接池管理器
func (cp *ConnectionPool) Start() error {
	// 启动健康检查
	go cp.healthCheck.Start(cp.ctx)
	return nil
}

// Stop 停止连接池管理器
func (cp *ConnectionPool) Stop() error {
	// 停止健康检查
	cp.cancel()

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// 关闭所有数据库连接池
	for name, pool := range cp.dbPools {
		if err := pool.Close(); err != nil {
			fmt.Printf("关闭数据库连接池 %s 失败: %v\n", name, err)
		}
	}

	// 关闭所有Redis连接池
	for name, pool := range cp.redisPools {
		if err := pool.Close(); err != nil {
			fmt.Printf("关闭Redis连接池 %s 失败: %v\n", name, err)
		}
	}

	// 关闭所有HTTP客户端连接
	for name, client := range cp.httpClients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		fmt.Printf("关闭HTTP客户端 %s\n", name)
	}

	return nil
}

// UpdateConfig 更新连接池配置
func (cp *ConnectionPool) UpdateConfig(config *ConnectionPoolConfig) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	cp.config = config

	// 更新现有数据库连接池配置
	for _, pool := range cp.dbPools {
		pool.SetMaxOpenConns(config.DBMaxOpenConns)
		pool.SetMaxIdleConns(config.DBMaxIdleConns)
		pool.SetConnMaxLifetime(config.DBConnMaxLifetime)
		pool.SetConnMaxIdleTime(config.DBConnMaxIdleTime)
	}

	// 更新健康检查配置
	cp.healthCheck.UpdateConfig(config.HealthCheckInterval, config.HealthCheckTimeout)

	return nil
}
