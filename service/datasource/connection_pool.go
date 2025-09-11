/*
 * @module service/basic_library/datasource/connection_pool
 * @description HTTP连接池实现，提供连接复用和管理功能
 * @architecture 对象池模式 - 管理HTTP客户端连接的生命周期
 * @documentReference ai_docs/datasource_req1.md
 * @stateFlow 连接池生命周期：创建 -> 获取连接 -> 使用连接 -> 归还连接 -> 清理过期连接
 * @rules 支持连接复用、超时管理、最大连接数控制
 * @dependencies net/http, context, sync, time
 * @refs interface.go, http_auth.go
 */

package datasource

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HTTPConnectionPool HTTP连接池实现
type HTTPConnectionPool struct {
	mu             sync.RWMutex
	connections    chan *PooledConnection
	maxConnections int
	timeout        time.Duration
	idleTimeout    time.Duration
	cleanupTicker  *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
	stats          *PoolStats
}

// PooledConnection 池化的HTTP连接
type PooledConnection struct {
	Client     *http.Client
	CreatedAt  time.Time
	LastUsed   time.Time
	UsageCount int64
}

// PoolStats 连接池统计信息
type PoolStats struct {
	mu             sync.RWMutex
	TotalCreated   int64   `json:"total_created"`
	TotalReused    int64   `json:"total_reused"`
	ActiveCount    int     `json:"active_count"`
	IdleCount      int     `json:"idle_count"`
	MaxConnections int     `json:"max_connections"`
	HitRate        float64 `json:"hit_rate"`
}

// NewHTTPConnectionPool 创建HTTP连接池
func NewHTTPConnectionPool(maxConnections int, timeout, idleTimeout time.Duration) *HTTPConnectionPool {
	if maxConnections <= 0 {
		maxConnections = 10
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if idleTimeout <= 0 {
		idleTimeout = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &HTTPConnectionPool{
		connections:    make(chan *PooledConnection, maxConnections),
		maxConnections: maxConnections,
		timeout:        timeout,
		idleTimeout:    idleTimeout,
		ctx:            ctx,
		cancel:         cancel,
		stats: &PoolStats{
			MaxConnections: maxConnections,
		},
	}

	// 启动清理协程
	pool.startCleanup()

	return pool
}

// Get 获取连接
func (p *HTTPConnectionPool) Get(ctx context.Context) (interface{}, error) {
	select {
	case conn := <-p.connections:
		// 检查连接是否过期
		if time.Since(conn.LastUsed) > p.idleTimeout {
			// 连接过期，创建新连接
			return p.createConnection(), nil
		}

		// 更新使用统计
		conn.LastUsed = time.Now()
		conn.UsageCount++

		p.stats.mu.Lock()
		p.stats.TotalReused++
		p.stats.ActiveCount++
		p.stats.IdleCount--
		p.updateHitRate()
		p.stats.mu.Unlock()

		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// 没有可用连接，创建新连接
		conn := p.createConnection()

		p.stats.mu.Lock()
		p.stats.TotalCreated++
		p.stats.ActiveCount++
		p.updateHitRate()
		p.stats.mu.Unlock()

		return conn, nil
	}
}

// Put 归还连接
func (p *HTTPConnectionPool) Put(conn interface{}) error {
	pooledConn, ok := conn.(*PooledConnection)
	if !ok {
		return fmt.Errorf("连接类型错误")
	}

	pooledConn.LastUsed = time.Now()

	select {
	case p.connections <- pooledConn:
		p.stats.mu.Lock()
		p.stats.ActiveCount--
		p.stats.IdleCount++
		p.stats.mu.Unlock()
		return nil
	default:
		// 连接池已满，丢弃连接
		return nil
	}
}

// Close 关闭连接池
func (p *HTTPConnectionPool) Close() error {
	p.cancel()

	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	// 清空所有连接
	close(p.connections)
	for conn := range p.connections {
		// HTTP客户端没有显式关闭方法，依赖GC
		_ = conn
	}

	return nil
}

// Stats 获取连接池统计信息
func (p *HTTPConnectionPool) Stats() map[string]interface{} {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_created":   p.stats.TotalCreated,
		"total_reused":    p.stats.TotalReused,
		"active_count":    p.stats.ActiveCount,
		"idle_count":      p.stats.IdleCount,
		"max_connections": p.stats.MaxConnections,
		"hit_rate":        p.stats.HitRate,
	}
}

// createConnection 创建新连接
func (p *HTTPConnectionPool) createConnection() *PooledConnection {
	client := &http.Client{
		Timeout: p.timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &PooledConnection{
		Client:     client,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
		UsageCount: 0,
	}
}

// startCleanup 启动清理协程
func (p *HTTPConnectionPool) startCleanup() {
	p.cleanupTicker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.cleanupTicker.C:
				p.cleanup()
			}
		}
	}()
}

// cleanup 清理过期连接
func (p *HTTPConnectionPool) cleanup() {
	now := time.Now()

	// 从连接池中移除过期连接
	var validConnections []*PooledConnection

	// 收集所有当前连接
	for {
		select {
		case conn := <-p.connections:
			if now.Sub(conn.LastUsed) <= p.idleTimeout {
				validConnections = append(validConnections, conn)
			} else {
				// 过期连接，更新统计
				p.stats.mu.Lock()
				p.stats.IdleCount--
				p.stats.mu.Unlock()
			}
		default:
			// 没有更多连接
			goto done
		}
	}

done:
	// 将有效连接放回池中
	for _, conn := range validConnections {
		select {
		case p.connections <- conn:
		default:
			// 池已满，丢弃连接
			p.stats.mu.Lock()
			p.stats.IdleCount--
			p.stats.mu.Unlock()
		}
	}
}

// updateHitRate 更新命中率
func (p *HTTPConnectionPool) updateHitRate() {
	total := p.stats.TotalCreated + p.stats.TotalReused
	if total > 0 {
		p.stats.HitRate = float64(p.stats.TotalReused) / float64(total)
	}
}

// DefaultConnectionPool 默认连接池实现（用于不需要特殊配置的场景）
type DefaultConnectionPool struct {
	*HTTPConnectionPool
}

// NewDefaultConnectionPool 创建默认连接池
func NewDefaultConnectionPool() *DefaultConnectionPool {
	return &DefaultConnectionPool{
		HTTPConnectionPool: NewHTTPConnectionPool(20, 30*time.Second, 5*time.Minute),
	}
}
