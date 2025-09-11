/*
 * @module service/basic_library/datasource/base_test
 * @description 基础数据源组件单元测试
 * @architecture 单元测试 - 测试基础数据源和工厂的功能
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 测试流程：准备测试数据 -> 执行测试 -> 验证结果 -> 清理资源
 * @rules 覆盖所有公共方法和错误场景，确保代码质量
 * @dependencies testing, context, time
 * @refs base.go, interface.go, test_utils.go
 */

package datasource

import (
	"context"
	"testing"

	"datahub-service/service/models"
)

func TestBaseDataSource_Init(t *testing.T) {
	tests := []struct {
		name        string
		dsType      string
		isResident  bool
		dataSource  *models.DataSource
		expectError bool
	}{
		{
			name:       "successful init",
			dsType:     "test",
			isResident: true,
			dataSource: CreateTestDataSource(TestDataSourceConfig{
				ID:   "test-id",
				Type: "test",
			}),
			expectError: false,
		},
		{
			name:        "init with nil datasource",
			dsType:      "test",
			isResident:  false,
			dataSource:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseDataSource(tt.dsType, tt.isResident)
			ctx := context.Background()

			err := base.Init(ctx, tt.dataSource)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if base.GetType() != tt.dsType {
					t.Errorf("expected type %s, got %s", tt.dsType, base.GetType())
				}
				if base.IsResident() != tt.isResident {
					t.Errorf("expected resident %v, got %v", tt.isResident, base.IsResident())
				}
				if tt.dataSource != nil && base.GetID() != tt.dataSource.ID {
					t.Errorf("expected ID %s, got %s", tt.dataSource.ID, base.GetID())
				}
			}
		})
	}
}

func TestBaseDataSource_Start(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*BaseDataSource)
		expectError bool
	}{
		{
			name: "successful start",
			setupFunc: func(base *BaseDataSource) {
				ds := CreateTestDataSource(TestDataSourceConfig{})
				base.Init(context.Background(), ds)
			},
			expectError: false,
		},
		{
			name: "start without init",
			setupFunc: func(base *BaseDataSource) {
				// 不调用Init
			},
			expectError: true,
		},
		{
			name: "start already started",
			setupFunc: func(base *BaseDataSource) {
				ds := CreateTestDataSource(TestDataSourceConfig{})
				base.Init(context.Background(), ds)
				base.Start(context.Background())
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseDataSource("test", false)
			tt.setupFunc(base)

			ctx := context.Background()
			err := base.Start(ctx)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBaseDataSource_Execute(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*BaseDataSource)
		request       *ExecuteRequest
		expectError   bool
		expectSuccess bool
	}{
		{
			name: "execute without script",
			setupFunc: func(base *BaseDataSource) {
				ds := CreateTestDataSource(TestDataSourceConfig{
					ScriptEnabled: false,
				})
				base.Init(context.Background(), ds)
			},
			request:       CreateTestExecuteRequest("test", "test query", nil),
			expectError:   true,
			expectSuccess: false,
		},
		{
			name: "execute not initialized",
			setupFunc: func(base *BaseDataSource) {
				// 不调用Init
			},
			request:       CreateTestExecuteRequest("test", "test query", nil),
			expectError:   true,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseDataSource("test", false)
			tt.setupFunc(base)

			ctx := context.Background()
			response, err := base.Execute(ctx, tt.request)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if response != nil && response.Success != tt.expectSuccess {
				t.Errorf("expected success %v, got %v", tt.expectSuccess, response.Success)
			}
		})
	}
}

func TestBaseDataSource_Stop(t *testing.T) {
	base := NewBaseDataSource("test", false)
	ds := CreateTestDataSource(TestDataSourceConfig{})

	ctx := context.Background()
	base.Init(ctx, ds)
	base.Start(ctx)

	// 测试停止
	err := base.Stop(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if base.IsStarted() {
		t.Errorf("expected datasource to be stopped")
	}

	// 测试重复停止
	err = base.Stop(ctx)
	if err != nil {
		t.Errorf("unexpected error on double stop: %v", err)
	}
}

func TestBaseDataSource_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*BaseDataSource)
		expectedStatus string
	}{
		{
			name: "healthy datasource",
			setupFunc: func(base *BaseDataSource) {
				ds := CreateTestDataSource(TestDataSourceConfig{})
				base.Init(context.Background(), ds)
				base.Start(context.Background())
			},
			expectedStatus: "online",
		},
		{
			name: "not initialized",
			setupFunc: func(base *BaseDataSource) {
				// 不调用Init
			},
			expectedStatus: "offline",
		},
		{
			name: "initialized but not started (resident)",
			setupFunc: func(base *BaseDataSource) {
				ds := CreateTestDataSource(TestDataSourceConfig{})
				base.Init(context.Background(), ds)
			},
			expectedStatus: "offline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isResident := tt.name == "initialized but not started (resident)"
			base := NewBaseDataSource("test", isResident)
			tt.setupFunc(base)

			ctx := context.Background()
			status, err := base.HealthCheck(ctx)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if status.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status.Status)
			}

			// 验证响应时间
			if status.ResponseTime <= 0 {
				t.Errorf("expected positive response time, got %v", status.ResponseTime)
			}

			// 验证详情
			if status.Details == nil {
				t.Errorf("expected details to be non-nil")
			}
		})
	}
}

func TestDefaultDataSourceFactory_Create(t *testing.T) {
	factory := NewDefaultDataSourceFactory()

	// 测试创建不存在的类型
	_, err := factory.Create("nonexistent")
	if err == nil {
		t.Errorf("expected error for nonexistent type")
	}

	// 注册测试类型
	err = factory.RegisterType("test", func() DataSourceInterface {
		return NewMockDataSource("test", false)
	})
	if err != nil {
		t.Errorf("unexpected error registering type: %v", err)
	}

	// 测试创建已注册的类型
	ds, err := factory.Create("test")
	if err != nil {
		t.Errorf("unexpected error creating datasource: %v", err)
	}

	if ds == nil {
		t.Errorf("expected non-nil datasource")
	}

	if ds.GetType() != "test" {
		t.Errorf("expected type test, got %s", ds.GetType())
	}
}

func TestDefaultDataSourceFactory_RegisterType(t *testing.T) {
	factory := NewDefaultDataSourceFactory()

	tests := []struct {
		name        string
		dsType      string
		creator     DataSourceCreator
		expectError bool
	}{
		{
			name:   "successful registration",
			dsType: "test",
			creator: func() DataSourceInterface {
				return NewMockDataSource("test", false)
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
			dsType:      "test",
			creator:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.RegisterType(tt.dsType, tt.creator)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				// 验证类型已注册
				supportedTypes := factory.GetSupportedTypes()
				found := false
				for _, t := range supportedTypes {
					if t == tt.dsType {
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

func TestDefaultDataSourceFactory_GetSupportedTypes(t *testing.T) {
	factory := NewDefaultDataSourceFactory()

	// 初始应该为空
	types := factory.GetSupportedTypes()
	if len(types) != 0 {
		t.Errorf("expected empty types list, got %v", types)
	}

	// 注册几个类型
	factory.RegisterType("type1", func() DataSourceInterface {
		return NewMockDataSource("type1", false)
	})
	factory.RegisterType("type2", func() DataSourceInterface {
		return NewMockDataSource("type2", true)
	})

	types = factory.GetSupportedTypes()
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d", len(types))
	}

	// 验证类型存在
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	if !typeMap["type1"] {
		t.Errorf("type1 not found in supported types")
	}
	if !typeMap["type2"] {
		t.Errorf("type2 not found in supported types")
	}
}

func TestYaegiScriptExecutor_Validate(t *testing.T) {
	executor := NewYaegiScriptExecutor()

	tests := []struct {
		name        string
		script      string
		expectError bool
	}{
		{
			name:        "valid script",
			script:      "1 + 1",
			expectError: false,
		},
		{
			name:        "invalid script",
			script:      "func invalid syntax {",
			expectError: true,
		},
		{
			name:        "empty script",
			script:      "",
			expectError: false, // 空脚本通常是有效的
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.script)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// 基准测试
func BenchmarkBaseDataSource_HealthCheck(b *testing.B) {
	base := NewBaseDataSource("test", false)
	ds := CreateTestDataSource(TestDataSourceConfig{})
	ctx := context.Background()

	base.Init(ctx, ds)
	base.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := base.HealthCheck(ctx)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}

func BenchmarkDefaultDataSourceFactory_Create(b *testing.B) {
	factory := NewDefaultDataSourceFactory()
	factory.RegisterType("test", func() DataSourceInterface {
		return NewMockDataSource("test", false)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.Create("test")
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}
