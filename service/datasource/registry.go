/*
 * @module service/basic_library/datasource/registry
 * @description 数据源注册中心，负责数据源类型的注册和全局管理
 * @architecture 注册中心模式 + 单例模式 - 统一管理所有数据源类型
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 注册中心生命周期：初始化 -> 注册内置类型 -> 提供工厂服务 -> 管理实例
 * @rules 提供全局唯一的数据源工厂和管理器实例
 * @dependencies sync, log
 * @refs interface.go, base.go, manager.go, *_datasource.go
 */

package datasource

import (
	"fmt"
	"log"
	"sync"

	"datahub-service/service/meta"
)

// DataSourceRegistry 数据源注册中心
type DataSourceRegistry struct {
	mu      sync.RWMutex
	factory DataSourceFactory
	manager DataSourceManager
	logger  *log.Logger
}

// 全局注册中心实例
var (
	globalRegistry *DataSourceRegistry
	registryOnce   sync.Once
)

// GetGlobalRegistry 获取全局数据源注册中心实例
func GetGlobalRegistry() *DataSourceRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewDataSourceRegistry()
	})
	return globalRegistry
}

// NewDataSourceRegistry 创建数据源注册中心
func NewDataSourceRegistry() *DataSourceRegistry {
	factory := NewDefaultDataSourceFactory()
	manager := NewDefaultDataSourceManager(factory)

	registry := &DataSourceRegistry{
		factory: factory,
		manager: manager,
		logger:  log.Default(),
	}

	// 注册内置数据源类型
	registry.registerBuiltinTypes()

	return registry
}

// GetFactory 获取数据源工厂
func (r *DataSourceRegistry) GetFactory() DataSourceFactory {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.factory
}

// GetManager 获取数据源管理器
func (r *DataSourceRegistry) GetManager() DataSourceManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.manager
}

// RegisterType 注册数据源类型
func (r *DataSourceRegistry) RegisterType(dsType string, creator DataSourceCreator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.factory.RegisterType(dsType, creator); err != nil {
		return fmt.Errorf("注册数据源类型失败: %v", err)
	}

	r.logger.Printf("数据源类型 %s 注册成功", dsType)
	return nil
}

// GetSupportedTypes 获取支持的数据源类型
func (r *DataSourceRegistry) GetSupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.factory.GetSupportedTypes()
}

// CreateDataSource 创建数据源实例
func (r *DataSourceRegistry) CreateDataSource(dsType string) (DataSourceInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.factory.Create(dsType)
}

// GetStatistics 获取注册中心统计信息
func (r *DataSourceRegistry) GetStatistics() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["supported_types"] = r.factory.GetSupportedTypes()
	// stats["manager_stats"] = r.manager.GetStatistics() // 需要实现GetStatistics方法

	return stats
}

// registerBuiltinTypes 注册内置数据源类型
func (r *DataSourceRegistry) registerBuiltinTypes() {
	// 注册PostgreSQL数据源
	if err := r.factory.RegisterType(meta.DataSourceTypeDBPostgreSQL, NewPostgreSQLDataSource); err != nil {
		r.logger.Printf("注册PostgreSQL数据源失败: %v", err)
	}

	// 注册HTTP认证数据源
	if err := r.factory.RegisterType(meta.DataSourceTypeApiHTTPWithAuth, NewHTTPAuthDataSource); err != nil {
		r.logger.Printf("注册HTTP认证数据源失败: %v", err)
	}

	// 注册HTTP无认证数据源
	if err := r.factory.RegisterType(meta.DataSourceTypeApiHTTP, NewHTTPNoAuthDataSource); err != nil {
		r.logger.Printf("注册HTTP数据源失败: %v", err)
	}

	// 注册HTTP POST数据源
	if err := r.factory.RegisterType(meta.DataSourceTypeMessagingHttpPost, NewHTTPPostDataSource); err != nil {
		r.logger.Printf("注册HTTP POST数据源失败: %v", err)
	}

	// 注册MQTT数据源
	if err := r.factory.RegisterType(meta.DataSourceTypeMessagingMQTT, NewMQTTDataSource); err != nil {
		r.logger.Printf("注册MQTT数据源失败: %v", err)
	}

	r.logger.Printf("内置数据源类型注册完成，支持类型: %v", r.factory.GetSupportedTypes())
}

// DataSourceService 数据源服务，提供高级API
type DataSourceService struct {
	registry *DataSourceRegistry
}

// NewDataSourceService 创建数据源服务
func NewDataSourceService() *DataSourceService {
	return &DataSourceService{
		registry: GetGlobalRegistry(),
	}
}

// GetSupportedTypes 获取支持的数据源类型
func (s *DataSourceService) GetSupportedTypes() []string {
	return s.registry.GetSupportedTypes()
}

// ValidateDataSourceType 验证数据源类型是否支持
func (s *DataSourceService) ValidateDataSourceType(dsType string) error {
	supportedTypes := s.registry.GetSupportedTypes()
	for _, supportedType := range supportedTypes {
		if supportedType == dsType {
			return nil
		}
	}
	return fmt.Errorf("不支持的数据源类型: %s，支持的类型: %v", dsType, supportedTypes)
}

// GetDataSourceTypeDefinition 获取数据源类型定义
func (s *DataSourceService) GetDataSourceTypeDefinition(dsType string) (*meta.DataSourceTypeDefinition, error) {
	// 验证类型是否支持
	if err := s.ValidateDataSourceType(dsType); err != nil {
		return nil, err
	}

	// 从元数据中获取类型定义
	if definition, exists := meta.DataSourceTypes[dsType]; exists {
		return definition, nil
	}

	return nil, fmt.Errorf("数据源类型定义不存在: %s", dsType)
}

// ValidateDataSourceConfig 验证数据源配置
func (s *DataSourceService) ValidateDataSourceConfig(dsType string, connectionConfig, paramsConfig map[string]interface{}) (*meta.ValidationResult, error) {
	definition, err := s.GetDataSourceTypeDefinition(dsType)
	if err != nil {
		return nil, err
	}

	return definition.ValidateConfig(connectionConfig, paramsConfig), nil
}

// GetDataSourceExamples 获取数据源示例配置
func (s *DataSourceService) GetDataSourceExamples(dsType string) ([]meta.DataSourceExample, error) {
	definition, err := s.GetDataSourceTypeDefinition(dsType)
	if err != nil {
		return nil, err
	}

	return definition.Examples, nil
}

// GetRegistry 获取注册中心实例（便捷方法）
func GetRegistry() *DataSourceRegistry {
	return GetGlobalRegistry()
}

// GetFactory 获取全局数据源工厂（便捷方法）
func GetFactory() DataSourceFactory {
	return GetGlobalRegistry().GetFactory()
}

// GetManager 获取全局数据源管理器（便捷方法）
func GetManager() DataSourceManager {
	return GetGlobalRegistry().GetManager()
}

// GetService 获取数据源服务（便捷方法）
func GetService() *DataSourceService {
	return NewDataSourceService()
}

// RegisterDataSourceType 注册数据源类型（便捷方法）
func RegisterDataSourceType(dsType string, creator DataSourceCreator) error {
	return GetGlobalRegistry().RegisterType(dsType, creator)
}

// CreateDataSource 创建数据源实例（便捷方法）
func CreateDataSource(dsType string) (DataSourceInterface, error) {
	return GetGlobalRegistry().CreateDataSource(dsType)
}

// GetSupportedDataSourceTypes 获取支持的数据源类型（便捷方法）
func GetSupportedDataSourceTypes() []string {
	return GetGlobalRegistry().GetSupportedTypes()
}
