/*
 * @module service/basic_library/datasource/manager_test
 * @description 数据源管理器单元测试
 * @architecture 单元测试 - 测试数据源管理器的功能
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 测试流程：准备测试数据 -> 执行测试 -> 验证结果 -> 清理资源
 * @rules 覆盖管理器的所有功能，包括注册、获取、移除和生命周期管理
 * @dependencies testing, context, time
 * @refs manager.go, interface.go, test_utils.go
 */

package datasource

import (
	"context"
	"fmt"
	"testing"

	"datahub-service/service/models"
)

func TestDefaultDataSourceManager_Register(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (*DefaultDataSourceManager, *models.DataSource)
		expectError bool
	}{
		{
			name: "successful registration",
			setupFunc: func() (*DefaultDataSourceManager, *models.DataSource) {
				factory := NewDefaultDataSourceFactory()
				factory.RegisterType("mock", func() DataSourceInterface {
					return NewMockDataSource("mock", false)
				})
				manager := NewDefaultDataSourceManager(factory)
				ds := CreateTestDataSource(TestDataSourceConfig{
					ID:   "test-id",
					Type: "mock",
				})
				return manager, ds
			},
			expectError: false,
		},
		{
			name: "nil datasource",
			setupFunc: func() (*DefaultDataSourceManager, *models.DataSource) {
				factory := NewDefaultDataSourceFactory()
				manager := NewDefaultDataSourceManager(factory)
				return manager, nil
			},
			expectError: true,
		},
		{
			name: "empty ID",
			setupFunc: func() (*DefaultDataSourceManager, *models.DataSource) {
				factory := NewDefaultDataSourceFactory()
				manager := NewDefaultDataSourceManager(factory)
				ds := CreateTestDataSource(TestDataSourceConfig{
					ID:   "",
					Type: "mock",
				})
				return manager, ds
			},
			expectError: true,
		},
		{
			name: "empty type",
			setupFunc: func() (*DefaultDataSourceManager, *models.DataSource) {
				factory := NewDefaultDataSourceFactory()
				manager := NewDefaultDataSourceManager(factory)
				ds := CreateTestDataSource(TestDataSourceConfig{
					ID:   "test-id",
					Type: "",
				})
				return manager, ds
			},
			expectError: true,
		},
		{
			name: "unsupported type",
			setupFunc: func() (*DefaultDataSourceManager, *models.DataSource) {
				factory := NewDefaultDataSourceFactory()
				manager := NewDefaultDataSourceManager(factory)
				ds := CreateTestDataSource(TestDataSourceConfig{
					ID:   "test-id",
					Type: "unsupported",
				})
				return manager, ds
			},
			expectError: true,
		},
		{
			name: "duplicate registration",
			setupFunc: func() (*DefaultDataSourceManager, *models.DataSource) {
				factory := NewDefaultDataSourceFactory()
				factory.RegisterType("mock", func() DataSourceInterface {
					return NewMockDataSource("mock", false)
				})
				manager := NewDefaultDataSourceManager(factory)
				ds := CreateTestDataSource(TestDataSourceConfig{
					ID:   "test-id",
					Type: "mock",
				})
				// 先注册一次
				manager.Register(context.Background(), ds)
				return manager, ds
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, ds := tt.setupFunc()
			ctx := context.Background()

			err := manager.Register(ctx, ds)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && ds != nil {
				// 验证数据源已注册
				_, err := manager.Get(ds.ID)
				if err != nil {
					t.Errorf("failed to get registered datasource: %v", err)
				}
			}
		})
	}
}

func TestDefaultDataSourceManager_Get(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	// 注册一个数据源
	ds := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id",
		Type: "mock",
	})
	ctx := context.Background()
	manager.Register(ctx, ds)

	tests := []struct {
		name        string
		dsID        string
		expectError bool
	}{
		{
			name:        "get existing datasource",
			dsID:        "test-id",
			expectError: false,
		},
		{
			name:        "get non-existing datasource",
			dsID:        "non-existing",
			expectError: true,
		},
		{
			name:        "empty ID",
			dsID:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := manager.Get(tt.dsID)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if instance == nil {
					t.Errorf("expected non-nil instance")
				}
				if instance.GetID() != tt.dsID {
					t.Errorf("expected ID %s, got %s", tt.dsID, instance.GetID())
				}
			}
		})
	}
}

func TestDefaultDataSourceManager_Remove(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	// 注册一个数据源
	ds := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id",
		Type: "mock",
	})
	ctx := context.Background()
	manager.Register(ctx, ds)

	tests := []struct {
		name        string
		dsID        string
		expectError bool
	}{
		{
			name:        "remove existing datasource",
			dsID:        "test-id",
			expectError: false,
		},
		{
			name:        "remove non-existing datasource",
			dsID:        "non-existing",
			expectError: true,
		},
		{
			name:        "empty ID",
			dsID:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Remove(tt.dsID)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				// 验证数据源已被移除
				_, err := manager.Get(tt.dsID)
				if err == nil {
					t.Errorf("expected error when getting removed datasource")
				}
			}
		})
	}
}

func TestDefaultDataSourceManager_List(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	ctx := context.Background()

	// 初始应该为空
	list := manager.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	// 注册几个数据源
	ds1 := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id-1",
		Type: "mock",
	})
	ds2 := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id-2",
		Type: "mock",
	})

	manager.Register(ctx, ds1)
	manager.Register(ctx, ds2)

	list = manager.List()
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}

	// 验证数据源存在
	if _, exists := list["test-id-1"]; !exists {
		t.Errorf("test-id-1 not found in list")
	}
	if _, exists := list["test-id-2"]; !exists {
		t.Errorf("test-id-2 not found in list")
	}
}

func TestDefaultDataSourceManager_StartAll(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock-resident", func() DataSourceInterface {
		return NewMockDataSource("mock-resident", true)
	})
	factory.RegisterType("mock-non-resident", func() DataSourceInterface {
		return NewMockDataSource("mock-non-resident", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	ctx := context.Background()

	// 注册常驻和非常驻数据源
	residentDS := CreateTestDataSource(TestDataSourceConfig{
		ID:   "resident-id",
		Type: "mock-resident",
	})
	nonResidentDS := CreateTestDataSource(TestDataSourceConfig{
		ID:   "non-resident-id",
		Type: "mock-non-resident",
	})

	manager.Register(ctx, residentDS)
	manager.Register(ctx, nonResidentDS)

	// 启动所有数据源
	err := manager.StartAll(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 验证常驻数据源已启动
	residentInstance, _ := manager.Get("resident-id")
	if mockResident, ok := residentInstance.(*MockDataSource); ok {
		if !mockResident.WasStartCalled() {
			t.Errorf("resident datasource should be started")
		}
	}

	// 验证非常驻数据源未启动
	nonResidentInstance, _ := manager.Get("non-resident-id")
	if mockNonResident, ok := nonResidentInstance.(*MockDataSource); ok {
		if mockNonResident.WasStartCalled() {
			t.Errorf("non-resident datasource should not be started")
		}
	}
}

func TestDefaultDataSourceManager_StopAll(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", true)
	})
	manager := NewDefaultDataSourceManager(factory)

	ctx := context.Background()

	// 注册数据源
	ds := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id",
		Type: "mock",
	})
	manager.Register(ctx, ds)

	// 停止所有数据源
	err := manager.StopAll(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 验证数据源已停止
	instance, _ := manager.Get("test-id")
	if mockInstance, ok := instance.(*MockDataSource); ok {
		if !mockInstance.WasStopCalled() {
			t.Errorf("datasource should be stopped")
		}
	}
}

func TestDefaultDataSourceManager_HealthCheckAll(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	ctx := context.Background()

	// 注册数据源
	ds1 := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id-1",
		Type: "mock",
	})
	ds2 := CreateTestDataSource(TestDataSourceConfig{
		ID:   "test-id-2",
		Type: "mock",
	})

	manager.Register(ctx, ds1)
	manager.Register(ctx, ds2)

	// 执行健康检查
	results := manager.HealthCheckAll(ctx)

	if len(results) != 2 {
		t.Errorf("expected 2 health check results, got %d", len(results))
	}

	// 验证结果存在
	if _, exists := results["test-id-1"]; !exists {
		t.Errorf("health check result for test-id-1 not found")
	}
	if _, exists := results["test-id-2"]; !exists {
		t.Errorf("health check result for test-id-2 not found")
	}

	// 验证健康状态
	for id, status := range results {
		if status == nil {
			t.Errorf("health status for %s is nil", id)
		}
		if status.Status == "" {
			t.Errorf("health status for %s has empty status", id)
		}
	}
}

func TestDefaultDataSourceManager_ExecuteDataSource(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock-resident", func() DataSourceInterface {
		return NewMockDataSource("mock-resident", true)
	})
	factory.RegisterType("mock-non-resident", func() DataSourceInterface {
		return NewMockDataSource("mock-non-resident", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	ctx := context.Background()

	// 注册数据源
	residentDS := CreateTestDataSource(TestDataSourceConfig{
		ID:   "resident-id",
		Type: "mock-resident",
	})
	nonResidentDS := CreateTestDataSource(TestDataSourceConfig{
		ID:   "non-resident-id",
		Type: "mock-non-resident",
	})

	manager.Register(ctx, residentDS)
	manager.Register(ctx, nonResidentDS)

	request := CreateTestExecuteRequest("test", "test query", nil)

	tests := []struct {
		name        string
		dsID        string
		expectError bool
	}{
		{
			name:        "execute resident datasource",
			dsID:        "resident-id",
			expectError: false,
		},
		{
			name:        "execute non-resident datasource",
			dsID:        "non-resident-id",
			expectError: false,
		},
		{
			name:        "execute non-existing datasource",
			dsID:        "non-existing",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := manager.ExecuteDataSource(ctx, tt.dsID, request)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if response == nil {
					t.Errorf("expected non-nil response")
				}
				if !response.Success {
					t.Errorf("expected successful response")
				}
			}

			// 验证非常驻数据源的启动和停止行为
			if !tt.expectError && tt.dsID == "non-resident-id" {
				instance, _ := manager.Get(tt.dsID)
				if mockInstance, ok := instance.(*MockDataSource); ok {
					if !mockInstance.WasStartCalled() {
						t.Errorf("non-resident datasource should be started for execution")
					}
					// 注意：由于异步停止，这里可能无法立即验证Stop调用
				}
			}
		})
	}
}

func TestDefaultDataSourceManager_GetStatistics(t *testing.T) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock-resident", func() DataSourceInterface {
		return NewMockDataSource("mock-resident", true)
	})
	factory.RegisterType("mock-non-resident", func() DataSourceInterface {
		return NewMockDataSource("mock-non-resident", false)
	})
	manager := NewDefaultDataSourceManager(factory)

	ctx := context.Background()

	// 注册数据源
	residentDS := CreateTestDataSource(TestDataSourceConfig{
		ID:   "resident-id",
		Type: "mock-resident",
	})
	nonResidentDS := CreateTestDataSource(TestDataSourceConfig{
		ID:   "non-resident-id",
		Type: "mock-non-resident",
	})

	manager.Register(ctx, residentDS)
	manager.Register(ctx, nonResidentDS)

	stats := manager.GetStatistics()

	// 验证统计信息
	if stats["total_count"] != 2 {
		t.Errorf("expected total_count 2, got %v", stats["total_count"])
	}

	if stats["resident_count"] != 1 {
		t.Errorf("expected resident_count 1, got %v", stats["resident_count"])
	}

	typeDistribution, ok := stats["type_distribution"].(map[string]int)
	if !ok {
		t.Errorf("expected type_distribution to be map[string]int")
	} else {
		if typeDistribution["mock-resident"] != 1 {
			t.Errorf("expected mock-resident count 1, got %d", typeDistribution["mock-resident"])
		}
		if typeDistribution["mock-non-resident"] != 1 {
			t.Errorf("expected mock-non-resident count 1, got %d", typeDistribution["mock-non-resident"])
		}
	}

	supportedTypes, ok := stats["supported_types"].([]string)
	if !ok {
		t.Errorf("expected supported_types to be []string")
	} else {
		if len(supportedTypes) < 2 {
			t.Errorf("expected at least 2 supported types, got %d", len(supportedTypes))
		}
	}
}

// 基准测试
func BenchmarkDefaultDataSourceManager_Register(b *testing.B) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", false)
	})
	manager := NewDefaultDataSourceManager(factory)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ds := CreateTestDataSource(TestDataSourceConfig{
			ID:   fmt.Sprintf("test-id-%d", i),
			Type: "mock",
		})
		err := manager.Register(ctx, ds)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}

func BenchmarkDefaultDataSourceManager_Get(b *testing.B) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("mock", func() DataSourceInterface {
		return NewMockDataSource("mock", false)
	})
	manager := NewDefaultDataSourceManager(factory)
	ctx := context.Background()

	// 预先注册一个数据源
	ds := CreateTestDataSource(TestDataSourceConfig{
		ID:   "benchmark-id",
		Type: "mock",
	})
	manager.Register(ctx, ds)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.Get("benchmark-id")
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}
