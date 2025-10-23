/*
 * @module service/datasource/realtime_processor_test
 * @description 实时数据处理器单元测试
 */

package datasource

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockInterfaceInfo Mock接口信息
type MockInterfaceInfo struct {
	ID          string
	SchemaName  string
	TableName   string
	ParseConfig map[string]interface{}
}

func (m *MockInterfaceInfo) GetID() string                          { return m.ID }
func (m *MockInterfaceInfo) GetSchemaName() string                  { return m.SchemaName }
func (m *MockInterfaceInfo) GetTableName() string                   { return m.TableName }
func (m *MockInterfaceInfo) GetParseConfig() map[string]interface{} { return m.ParseConfig }

// MockDataWriter Mock数据写入器
type MockDataWriter struct {
	WriteCount int64
	WriteError error
	WriteCalls []WriteCall
}

type WriteCall struct {
	InterfaceID string
	DataCount   int
}

func (m *MockDataWriter) WriteData(ctx context.Context, interfaceID string, data []map[string]interface{}) (int64, error) {
	m.WriteCalls = append(m.WriteCalls, WriteCall{
		InterfaceID: interfaceID,
		DataCount:   len(data),
	})

	if m.WriteError != nil {
		return 0, m.WriteError
	}
	m.WriteCount += int64(len(data))
	return int64(len(data)), nil
}

// MockInterfaceLoader Mock接口加载器
type MockInterfaceLoader struct {
	Interfaces map[string]InterfaceInfo
	LoadError  error
}

func (m *MockInterfaceLoader) LoadInterface(ctx context.Context, interfaceID string) (InterfaceInfo, error) {
	if m.LoadError != nil {
		return nil, m.LoadError
	}
	if info, exists := m.Interfaces[interfaceID]; exists {
		return info, nil
	}
	return nil, fmt.Errorf("interface not found: %s", interfaceID)
}

// TestNewDefaultRealtimeDataProcessor 测试创建处理器
func TestNewDefaultRealtimeDataProcessor(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.dataSourceInterfaces)
	assert.NotNil(t, processor.interfaceCache)
	assert.NotNil(t, processor.dataBatches)
	assert.Equal(t, 100, processor.batchSize)
	assert.Equal(t, 100*time.Millisecond, processor.batchTimeout)
}

// TestDefaultRealtimeDataProcessor_SetDB 测试设置数据库
func TestDefaultRealtimeDataProcessor_SetDB(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	processor.SetDB(db)

	assert.NotNil(t, processor.db)
}

// TestDefaultRealtimeDataProcessor_SetDependencies 测试设置依赖
func TestDefaultRealtimeDataProcessor_SetDependencies(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	writer := &MockDataWriter{}
	loader := &MockInterfaceLoader{Interfaces: make(map[string]InterfaceInfo)}

	processor.SetDataWriter(writer)
	processor.SetInterfaceLoader(loader)

	assert.NotNil(t, processor.dataWriter)
	assert.NotNil(t, processor.interfaceLoader)
}

// TestDefaultRealtimeDataProcessor_RegisterInterface 测试注册接口
func TestDefaultRealtimeDataProcessor_RegisterInterface(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:         "test-interface-id",
		SchemaName: "test_schema",
		TableName:  "test_table",
		ParseConfig: map[string]interface{}{
			"fieldMapping": []interface{}{
				map[string]interface{}{"source": "name", "target": "name"},
			},
		},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	processor.SetInterfaceLoader(loader)

	// 注册接口
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 验证注册结果
	processor.mu.RLock()
	interfaces := processor.dataSourceInterfaces["test-datasource-id"]
	_, cached := processor.interfaceCache["test-interface-id"]
	processor.mu.RUnlock()

	assert.Len(t, interfaces, 1)
	assert.Equal(t, "test-interface-id", interfaces[0])
	assert.True(t, cached)
}

// TestDefaultRealtimeDataProcessor_RegisterInterface_Duplicate 测试重复注册
func TestDefaultRealtimeDataProcessor_RegisterInterface_Duplicate(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:          "test-interface-id",
		SchemaName:  "test_schema",
		TableName:   "test_table",
		ParseConfig: map[string]interface{}{},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	processor.SetInterfaceLoader(loader)

	ctx := context.Background()

	// 首次注册
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 重复注册
	err = processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 验证仍然只有一个
	processor.mu.RLock()
	interfaces := processor.dataSourceInterfaces["test-datasource-id"]
	processor.mu.RUnlock()

	assert.Len(t, interfaces, 1)
}

// TestDefaultRealtimeDataProcessor_UnregisterInterface 测试注销接口
func TestDefaultRealtimeDataProcessor_UnregisterInterface(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:          "test-interface-id",
		SchemaName:  "test_schema",
		TableName:   "test_table",
		ParseConfig: map[string]interface{}{},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	processor.SetInterfaceLoader(loader)

	// 先注册
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 注销接口
	err = processor.UnregisterInterface("test-interface-id")
	assert.NoError(t, err)

	// 验证注销结果
	processor.mu.RLock()
	interfaces := processor.dataSourceInterfaces["test-datasource-id"]
	_, cached := processor.interfaceCache["test-interface-id"]
	processor.mu.RUnlock()

	assert.Len(t, interfaces, 0)
	assert.False(t, cached)
}

// TestDefaultRealtimeDataProcessor_ProcessRealtimeData 测试处理实时数据
func TestDefaultRealtimeDataProcessor_ProcessRealtimeData(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:          "test-interface-id",
		SchemaName:  "test_schema",
		TableName:   "test_table",
		ParseConfig: map[string]interface{}{},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	writer := &MockDataWriter{}

	processor.SetInterfaceLoader(loader)
	processor.SetDataWriter(writer)

	// 注册接口
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 处理数据
	testData := map[string]interface{}{
		"id":   "user-1",
		"name": "张三",
		"age":  25,
	}

	err = processor.ProcessRealtimeData(ctx, "test-datasource-id", testData)
	assert.NoError(t, err)

	// 验证数据已添加到批次
	processor.batchMu.RLock()
	batch := processor.dataBatches["test-interface-id"]
	processor.batchMu.RUnlock()

	assert.Len(t, batch, 1)
	assert.Equal(t, "user-1", batch[0]["id"])
}

// TestDefaultRealtimeDataProcessor_BatchWrite 测试批量写入
func TestDefaultRealtimeDataProcessor_BatchWrite(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	processor.batchSize = 3 // 设置较小的批次大小
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:          "test-interface-id",
		SchemaName:  "test_schema",
		TableName:   "test_table",
		ParseConfig: map[string]interface{}{},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	writer := &MockDataWriter{}

	processor.SetInterfaceLoader(loader)
	processor.SetDataWriter(writer)

	// 注册接口
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 处理多条数据触发批量写入
	for i := 0; i < 5; i++ {
		testData := map[string]interface{}{
			"id":   fmt.Sprintf("user-%d", i),
			"name": fmt.Sprintf("用户%d", i),
			"age":  20 + i,
		}
		err = processor.ProcessRealtimeData(ctx, "test-datasource-id", testData)
		assert.NoError(t, err)
	}

	// 等待异步刷新完成
	time.Sleep(200 * time.Millisecond)

	// 验证写入调用
	assert.GreaterOrEqual(t, len(writer.WriteCalls), 1)
	assert.GreaterOrEqual(t, writer.WriteCount, int64(3))
}

// TestDefaultRealtimeDataProcessor_FieldMapping 测试字段映射
func TestDefaultRealtimeDataProcessor_FieldMapping(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:         "test-interface-id",
		SchemaName: "test_schema",
		TableName:  "test_table",
		ParseConfig: map[string]interface{}{
			"fieldMapping": []interface{}{
				map[string]interface{}{"source": "user_name", "target": "name"},
				map[string]interface{}{"source": "user_age", "target": "age"},
			},
		},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	writer := &MockDataWriter{}

	processor.SetInterfaceLoader(loader)
	processor.SetDataWriter(writer)

	// 注册接口
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 处理使用源字段名的数据
	testData := map[string]interface{}{
		"id":        "user-1",
		"user_name": "李四",
		"user_age":  30,
	}

	err = processor.ProcessRealtimeData(ctx, "test-datasource-id", testData)
	assert.NoError(t, err)

	// 验证字段已被映射
	processor.batchMu.RLock()
	batch := processor.dataBatches["test-interface-id"]
	processor.batchMu.RUnlock()

	assert.Len(t, batch, 1)
	assert.Equal(t, "李四", batch[0]["name"])
	assert.Equal(t, 30, batch[0]["age"])
}

// TestDefaultRealtimeDataProcessor_MultipleInterfaces 测试多接口处理
func TestDefaultRealtimeDataProcessor_MultipleInterfaces(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface1 := &MockInterfaceInfo{
		ID:          "test-interface-1",
		SchemaName:  "test_schema",
		TableName:   "test_table_1",
		ParseConfig: map[string]interface{}{},
	}

	mockInterface2 := &MockInterfaceInfo{
		ID:          "test-interface-2",
		SchemaName:  "test_schema",
		TableName:   "test_table_2",
		ParseConfig: map[string]interface{}{},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-1": mockInterface1,
			"test-interface-2": mockInterface2,
		},
	}
	writer := &MockDataWriter{}

	processor.SetInterfaceLoader(loader)
	processor.SetDataWriter(writer)

	// 注册两个接口
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-1", "test-datasource-id")
	assert.NoError(t, err)
	err = processor.RegisterInterface(ctx, "test-interface-2", "test-datasource-id")
	assert.NoError(t, err)

	// 处理数据
	testData := map[string]interface{}{
		"id":   "user-1",
		"name": "王五",
		"age":  35,
	}

	err = processor.ProcessRealtimeData(ctx, "test-datasource-id", testData)
	assert.NoError(t, err)

	// 验证两个接口都收到了数据
	processor.batchMu.RLock()
	batch1 := processor.dataBatches["test-interface-1"]
	batch2 := processor.dataBatches["test-interface-2"]
	processor.batchMu.RUnlock()

	assert.Len(t, batch1, 1)
	assert.Len(t, batch2, 1)
}

// TestDefaultRealtimeDataProcessor_GetProcessorStats 测试获取统计信息
func TestDefaultRealtimeDataProcessor_GetProcessorStats(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	mockInterface := &MockInterfaceInfo{
		ID:          "test-interface-id",
		SchemaName:  "test_schema",
		TableName:   "test_table",
		ParseConfig: map[string]interface{}{},
	}

	loader := &MockInterfaceLoader{
		Interfaces: map[string]InterfaceInfo{
			"test-interface-id": mockInterface,
		},
	}
	writer := &MockDataWriter{}

	processor.SetInterfaceLoader(loader)
	processor.SetDataWriter(writer)

	// 注册接口
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.NoError(t, err)

	// 处理数据
	testData := map[string]interface{}{
		"id":   "user-1",
		"name": "赵六",
		"age":  40,
	}
	err = processor.ProcessRealtimeData(ctx, "test-datasource-id", testData)
	assert.NoError(t, err)

	// 获取统计信息
	stats := processor.GetProcessorStats()

	assert.Equal(t, int64(1), stats["total_processed"])
	assert.Equal(t, 1, stats["interface_count"])
	assert.Equal(t, 1, stats["datasource_count"])
	assert.Equal(t, 100, stats["batch_size"])
}

// TestDefaultRealtimeDataProcessor_LoadError 测试加载错误处理
func TestDefaultRealtimeDataProcessor_LoadError(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	loader := &MockInterfaceLoader{
		LoadError: errors.New("load error"),
	}
	processor.SetInterfaceLoader(loader)

	// 注册接口应该失败
	ctx := context.Background()
	err := processor.RegisterInterface(ctx, "test-interface-id", "test-datasource-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "加载接口信息失败")
}

// TestDefaultRealtimeDataProcessor_ProcessWithoutInterfaces 测试没有接口时处理数据
func TestDefaultRealtimeDataProcessor_ProcessWithoutInterfaces(t *testing.T) {
	processor := NewDefaultRealtimeDataProcessor()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	processor.SetDB(db)

	ctx := context.Background()
	testData := map[string]interface{}{
		"id":   "user-1",
		"name": "张三",
	}

	// 处理数据不应该出错
	err := processor.ProcessRealtimeData(ctx, "nonexistent-datasource", testData)
	assert.NoError(t, err)

	// 验证统计
	stats := processor.GetProcessorStats()
	assert.Equal(t, int64(0), stats["total_processed"]) // 没有接口，不处理
}
