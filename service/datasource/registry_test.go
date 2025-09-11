/*
 * @module service/basic_library/datasource/registry_test
 * @description 数据源注册中心单元测试
 * @architecture 单元测试 - 测试注册中心和服务的功能
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 测试流程：准备测试数据 -> 执行测试 -> 验证结果 -> 清理资源
 * @rules 覆盖注册中心的所有功能，包括类型注册、服务创建和配置验证
 * @dependencies testing, context
 * @refs registry.go, interface.go, test_utils.go
 */

package datasource

import (
	"testing"

	"datahub-service/service/meta"
)

func TestDataSourceRegistry_RegisterType(t *testing.T) {
	registry := NewDataSourceRegistry()

	tests := []struct {
		name        string
		dsType      string
		creator     DataSourceCreator
		expectError bool
	}{
		{
			name:   "successful registration",
			dsType: "test-type",
			creator: func() DataSourceInterface {
				return NewMockDataSource("test-type", false)
			},
			expectError: false,
		},
		{
			name:        "empty type",
			dsType:      "",
			creator:     func() DataSourceInterface { return nil },
			expectError: true,
		},
		{
			name:        "nil creator",
			dsType:      "test-type",
			creator:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.RegisterType(tt.dsType, tt.creator)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				// 验证类型已注册
				supportedTypes := registry.GetSupportedTypes()
				found := false
				for _, supportedType := range supportedTypes {
					if supportedType == tt.dsType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("type %s not found in supported types", tt.dsType)
				}
			}
		})
	}
}

func TestDataSourceRegistry_CreateDataSource(t *testing.T) {
	registry := NewDataSourceRegistry()

	// 注册测试类型
	registry.RegisterType("test-type", func() DataSourceInterface {
		return NewMockDataSource("test-type", false)
	})

	tests := []struct {
		name        string
		dsType      string
		expectError bool
	}{
		{
			name:        "create registered type",
			dsType:      "test-type",
			expectError: false,
		},
		{
			name:        "create unregistered type",
			dsType:      "unregistered",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, err := registry.CreateDataSource(tt.dsType)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if ds == nil {
					t.Errorf("expected non-nil datasource")
				}
				if ds.GetType() != tt.dsType {
					t.Errorf("expected type %s, got %s", tt.dsType, ds.GetType())
				}
			}
		})
	}
}

func TestDataSourceRegistry_GetSupportedTypes(t *testing.T) {
	registry := NewDataSourceRegistry()

	// 获取内置类型数量
	initialTypes := registry.GetSupportedTypes()
	initialCount := len(initialTypes)

	// 应该包含内置类型
	expectedBuiltinTypes := []string{
		meta.DataSourceTypeDBPostgreSQL,
		meta.DataSourceTypeApiHTTPWithAuth,
		meta.DataSourceTypeApiHTTP,
	}

	for _, expectedType := range expectedBuiltinTypes {
		found := false
		for _, actualType := range initialTypes {
			if actualType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected builtin type %s not found", expectedType)
		}
	}

	// 注册新类型
	registry.RegisterType("custom-type", func() DataSourceInterface {
		return NewMockDataSource("custom-type", false)
	})

	newTypes := registry.GetSupportedTypes()
	if len(newTypes) != initialCount+1 {
		t.Errorf("expected %d types after registration, got %d", initialCount+1, len(newTypes))
	}

	// 验证新类型存在
	found := false
	for _, t := range newTypes {
		if t == "custom-type" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("custom-type not found in supported types")
	}
}

func TestDataSourceRegistry_GetStatistics(t *testing.T) {
	registry := NewDataSourceRegistry()

	stats := registry.GetStatistics()

	// 验证统计信息结构
	if stats == nil {
		t.Errorf("expected non-nil statistics")
	}

	supportedTypes, ok := stats["supported_types"].([]string)
	if !ok {
		t.Errorf("expected supported_types to be []string")
	}

	if len(supportedTypes) == 0 {
		t.Errorf("expected at least some supported types")
	}

	managerStats, ok := stats["manager_stats"].(map[string]interface{})
	if !ok {
		t.Errorf("expected manager_stats to be map[string]interface{}")
	}

	if managerStats == nil {
		t.Errorf("expected non-nil manager stats")
	}
}

func TestGetGlobalRegistry(t *testing.T) {
	// 获取全局注册中心
	registry1 := GetGlobalRegistry()
	registry2 := GetGlobalRegistry()

	// 应该是同一个实例（单例）
	if registry1 != registry2 {
		t.Errorf("expected same registry instance, got different instances")
	}

	// 验证注册中心功能
	if registry1 == nil {
		t.Errorf("expected non-nil registry")
	}

	supportedTypes := registry1.GetSupportedTypes()
	if len(supportedTypes) == 0 {
		t.Errorf("expected at least some supported types")
	}
}

func TestDataSourceService_ValidateDataSourceType(t *testing.T) {
	service := NewDataSourceService()

	tests := []struct {
		name        string
		dsType      string
		expectError bool
	}{
		{
			name:        "valid builtin type",
			dsType:      meta.DataSourceTypeDBPostgreSQL,
			expectError: false,
		},
		{
			name:        "invalid type",
			dsType:      "invalid-type",
			expectError: true,
		},
		{
			name:        "empty type",
			dsType:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateDataSourceType(tt.dsType)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDataSourceService_GetDataSourceTypeDefinition(t *testing.T) {
	service := NewDataSourceService()

	tests := []struct {
		name        string
		dsType      string
		expectError bool
	}{
		{
			name:        "get postgresql definition",
			dsType:      meta.DataSourceTypeDBPostgreSQL,
			expectError: false,
		},
		{
			name:        "get invalid definition",
			dsType:      "invalid-type",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition, err := service.GetDataSourceTypeDefinition(tt.dsType)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if definition == nil {
					t.Errorf("expected non-nil definition")
				}
				if definition.Type != tt.dsType {
					t.Errorf("expected type %s, got %s", tt.dsType, definition.Type)
				}
				if definition.Name == "" {
					t.Errorf("expected non-empty name")
				}
				if len(definition.MetaConfig) == 0 {
					t.Errorf("expected non-empty meta config")
				}
			}
		})
	}
}

func TestDataSourceService_ValidateDataSourceConfig(t *testing.T) {
	service := NewDataSourceService()

	tests := []struct {
		name             string
		dsType           string
		connectionConfig map[string]interface{}
		paramsConfig     map[string]interface{}
		expectError      bool
		expectValid      bool
	}{
		{
			name:   "valid postgresql config",
			dsType: meta.DataSourceTypeDBPostgreSQL,
			connectionConfig: map[string]interface{}{
				"host":     "localhost",
				"port":     5432.0,
				"database": "testdb",
				"username": "testuser",
				"password": "testpass",
			},
			paramsConfig: map[string]interface{}{
				"timeout":         30.0,
				"max_connections": 100.0,
			},
			expectError: false,
			expectValid: true,
		},
		{
			name:   "invalid postgresql config - missing host",
			dsType: meta.DataSourceTypeDBPostgreSQL,
			connectionConfig: map[string]interface{}{
				"port":     5432.0,
				"database": "testdb",
				"username": "testuser",
				"password": "testpass",
			},
			expectError: false,
			expectValid: false,
		},
		{
			name:             "invalid type",
			dsType:           "invalid-type",
			connectionConfig: map[string]interface{}{},
			expectError:      true,
			expectValid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateDataSourceConfig(tt.dsType, tt.connectionConfig, tt.paramsConfig)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if result == nil {
					t.Errorf("expected non-nil validation result")
				}
				if result.IsValid != tt.expectValid {
					t.Errorf("expected valid %v, got %v", tt.expectValid, result.IsValid)
				}
				if result.Score < 0 || result.Score > 100 {
					t.Errorf("expected score between 0-100, got %d", result.Score)
				}
			}
		})
	}
}

func TestDataSourceService_GetDataSourceExamples(t *testing.T) {
	service := NewDataSourceService()

	tests := []struct {
		name        string
		dsType      string
		expectError bool
	}{
		{
			name:        "get postgresql examples",
			dsType:      meta.DataSourceTypeDBPostgreSQL,
			expectError: false,
		},
		{
			name:        "get invalid type examples",
			dsType:      "invalid-type",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			examples, err := service.GetDataSourceExamples(tt.dsType)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if examples == nil {
					t.Errorf("expected non-nil examples")
				}
				// 大多数内置类型都应该有示例
				if len(examples) == 0 && tt.dsType != meta.DataSourceTypeHTTPWithAuth {
					t.Errorf("expected at least one example for type %s", tt.dsType)
				}
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// 测试便捷函数
	registry := GetRegistry()
	if registry == nil {
		t.Errorf("expected non-nil registry")
	}

	factory := GetFactory()
	if factory == nil {
		t.Errorf("expected non-nil factory")
	}

	manager := GetManager()
	if manager == nil {
		t.Errorf("expected non-nil manager")
	}

	service := GetService()
	if service == nil {
		t.Errorf("expected non-nil service")
	}

	// 测试类型注册便捷函数
	err := RegisterDataSourceType("test-convenience", func() DataSourceInterface {
		return NewMockDataSource("test-convenience", false)
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 测试创建数据源便捷函数
	ds, err := CreateDataSource("test-convenience")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ds == nil {
		t.Errorf("expected non-nil datasource")
	}

	// 测试获取支持类型便捷函数
	types := GetSupportedDataSourceTypes()
	if len(types) == 0 {
		t.Errorf("expected at least some supported types")
	}

	// 验证新注册的类型存在
	found := false
	for _, t := range types {
		if t == "test-convenience" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("test-convenience type not found in supported types")
	}
}

// 基准测试
func BenchmarkDataSourceRegistry_CreateDataSource(b *testing.B) {
	registry := NewDataSourceRegistry()
	registry.RegisterType("benchmark-type", func() DataSourceInterface {
		return NewMockDataSource("benchmark-type", false)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.CreateDataSource("benchmark-type")
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}

func BenchmarkDataSourceService_ValidateDataSourceConfig(b *testing.B) {
	service := NewDataSourceService()
	connectionConfig := map[string]interface{}{
		"host":     "localhost",
		"port":     5432.0,
		"database": "testdb",
		"username": "testuser",
		"password": "testpass",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ValidateDataSourceConfig(meta.DataSourceTypeDBPostgreSQL, connectionConfig, nil)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}
