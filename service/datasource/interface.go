/*
 * @module service/basic_library/datasource/interface
 * @description 数据源统一接口定义，提供Init, Start, Execute, Stop等标准方法
 * @architecture 接口隔离原则 - 定义数据源操作的标准接口
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 数据源生命周期：Init -> Start -> Execute -> Stop
 * @rules 所有数据源实现必须遵循统一接口，支持常驻和动态脚本执行
 * @dependencies github.com/traefik/yaegi, context
 * @refs service/models/basic_library.go
 */

package datasource

import (
	"context"
	"time"

	"datahub-service/service/models"
)

// DataSourceInterface 数据源统一接口
type DataSourceInterface interface {
	// Init 初始化数据源，设置连接参数和配置
	Init(ctx context.Context, ds *models.DataSource) error

	// Start 启动数据源，建立连接，准备接收请求
	Start(ctx context.Context) error

	// Execute 执行数据操作，根据请求参数执行相应操作
	Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error)

	// Stop 停止数据源，关闭连接，清理资源
	Stop(ctx context.Context) error

	// HealthCheck 健康检查，返回数据源当前状态
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// GetType 获取数据源类型
	GetType() string

	// GetID 获取数据源ID
	GetID() string

	// IsResident 是否为常驻数据源（需要保持连接）
	IsResident() bool

	// IsInitialized 检查是否已初始化
	IsInitialized() bool

	// IsStarted 检查是否已启动
	IsStarted() bool
}

// ExecuteRequest 执行请求参数
type ExecuteRequest struct {
	Operation string                 `json:"operation"` // query, insert, update, delete, sync
	Query     string                 `json:"query,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timeout   time.Duration          `json:"timeout,omitempty"`
}

// ExecuteResponse 执行响应结果
type ExecuteResponse struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data,omitempty"`
	RowCount  int64                  `json:"row_count,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status       string                 `json:"status"` // online, offline, error, testing
	Message      string                 `json:"message,omitempty"`
	LastCheck    time.Time              `json:"last_check"`
	ResponseTime time.Duration          `json:"response_time"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	MaxConnections int           `json:"max_connections"`
	MinConnections int           `json:"min_connections"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
	MaxLifetime    time.Duration `json:"max_lifetime"`
	HealthCheck    time.Duration `json:"health_check_interval"`
}

// DataSourceStatus 数据源状态
type DataSourceStatus struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"`
	Name              string                 `json:"name"`
	IsResident        bool                   `json:"is_resident"`
	IsInitialized     bool                   `json:"is_initialized"`
	IsStarted         bool                   `json:"is_started"`
	LastHealthCheck   time.Time              `json:"last_health_check"`
	HealthStatus      string                 `json:"health_status"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	StartedAt         time.Time              `json:"started_at,omitempty"`
	LastUsed          time.Time              `json:"last_used,omitempty"`
	UsageCount        int64                  `json:"usage_count"`
	ReconnectAttempts int                    `json:"reconnect_attempts"`
	MaxReconnects     int                    `json:"max_reconnects"`
	AutoRestart       bool                   `json:"auto_restart"`
	ConnectionPool    *ConnectionPoolConfig  `json:"connection_pool,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// DataSourceFactory 数据源工厂接口
type DataSourceFactory interface {
	// Create 创建数据源实例
	Create(dsType string) (DataSourceInterface, error)

	// GetSupportedTypes 获取支持的数据源类型列表
	GetSupportedTypes() []string

	// RegisterType 注册新的数据源类型
	RegisterType(dsType string, creator DataSourceCreator) error
}

// DataSourceCreator 数据源创建器函数类型
type DataSourceCreator func() DataSourceInterface

// DataSourceManager 数据源管理器接口
type DataSourceManager interface {
	// Register 注册数据源实例
	Register(ctx context.Context, ds *models.DataSource) error

	// Get 获取数据源实例
	Get(dsID string) (DataSourceInterface, error)

	// Remove 移除数据源实例
	Remove(dsID string) error

	// CreateInstance 创建数据源实例（不注册到管理器中，用于测试）
	CreateInstance(dsType string) (DataSourceInterface, error)

	// CreateTestInstance 创建测试数据源实例（非常驻模式，用于连接测试）
	CreateTestInstance(dsType string) (DataSourceInterface, error)

	// List 列出所有注册的数据源
	List() map[string]DataSourceInterface

	// StartAll 启动所有常驻数据源
	StartAll(ctx context.Context) error

	// StopAll 停止所有数据源
	StopAll(ctx context.Context) error

	// HealthCheckAll 对所有数据源进行健康检查
	HealthCheckAll(ctx context.Context) map[string]*HealthStatus

	// ExecuteDataSource 执行数据源操作（便捷方法）
	ExecuteDataSource(ctx context.Context, dsID string, request *ExecuteRequest) (*ExecuteResponse, error)

	// GetStatistics 获取管理器统计信息
	GetStatistics() map[string]interface{}

	// GetDataSourceStatus 获取数据源状态
	GetDataSourceStatus(dsID string) (*DataSourceStatus, error)

	// GetAllDataSourceStatus 获取所有数据源状态
	GetAllDataSourceStatus() map[string]*DataSourceStatus

	// RestartResidentDataSource 重启常驻数据源
	RestartResidentDataSource(ctx context.Context, dsID string) error

	// GetResidentDataSources 获取所有常驻数据源
	GetResidentDataSources() map[string]*DataSourceStatus

	// Shutdown 关闭管理器
	Shutdown() error
}

// ScriptExecutor 脚本执行器接口
type ScriptExecutor interface {
	// Execute 执行脚本
	Execute(ctx context.Context, script string, params map[string]interface{}) (interface{}, error)

	// Validate 验证脚本语法
	Validate(script string) error
}

// ConnectionPool 连接池接口
type ConnectionPool interface {
	// Get 获取连接
	Get(ctx context.Context) (interface{}, error)

	// Put 归还连接
	Put(conn interface{}) error

	// Close 关闭连接池
	Close() error

	// Stats 获取连接池统计信息
	Stats() map[string]interface{}
}
