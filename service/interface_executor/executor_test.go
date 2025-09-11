/*
 * @module service/interface_executor/executor_test
 * @description InterfaceExecutor的单元测试
 * @architecture 测试驱动开发 - 确保接口执行器的各种操作正常工作
 * @documentReference design.md
 * @stateFlow 测试准备 -> Mock构造 -> 执行测试 -> 结果验证 -> 清理资源
 * @rules 测试用例需要覆盖预览、测试、同步等各种执行类型
 * @dependencies testing, testify, gorm, sqlite, mock
 * @refs executor.go
 */

package interface_executor

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/models"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockDataSourceManager 模拟数据源管理器
type MockDataSourceManager struct {
	mock.Mock
}

func (m *MockDataSourceManager) Register(ctx context.Context, ds *models.DataSource) error {
	args := m.Called(ctx, ds)
	return args.Error(0)
}

func (m *MockDataSourceManager) Get(dsID string) (datasource.DataSourceInterface, error) {
	args := m.Called(dsID)
	return args.Get(0).(datasource.DataSourceInterface), args.Error(1)
}

func (m *MockDataSourceManager) GetAll() map[string]datasource.DataSourceInterface {
	args := m.Called()
	return args.Get(0).(map[string]datasource.DataSourceInterface)
}

func (m *MockDataSourceManager) Remove(dsID string) error {
	args := m.Called(dsID)
	return args.Error(0)
}

func (m *MockDataSourceManager) StartAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDataSourceManager) StopAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDataSourceManager) GetStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockDataSourceManager) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDataSourceManager) CreateInstance(dsType string) (datasource.DataSourceInterface, error) {
	args := m.Called(dsType)
	return args.Get(0).(datasource.DataSourceInterface), args.Error(1)
}

func (m *MockDataSourceManager) ExecuteDataSource(ctx context.Context, dsID string, request *datasource.ExecuteRequest) (*datasource.ExecuteResponse, error) {
	args := m.Called(ctx, dsID, request)
	return args.Get(0).(*datasource.ExecuteResponse), args.Error(1)
}

func (m *MockDataSourceManager) GetStatistics() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockDataSourceManager) HealthCheckAll(ctx context.Context) map[string]*datasource.HealthStatus {
	args := m.Called(ctx)
	return args.Get(0).(map[string]*datasource.HealthStatus)
}

func (m *MockDataSourceManager) List() map[string]datasource.DataSourceInterface {
	args := m.Called()
	return args.Get(0).(map[string]datasource.DataSourceInterface)
}

func (m *MockDataSourceManager) CreateTestInstance(dsType string) (datasource.DataSourceInterface, error) {
	args := m.Called(dsType)
	return args.Get(0).(datasource.DataSourceInterface), args.Error(1)
}

func (m *MockDataSourceManager) GetAllDataSourceStatus() map[string]*datasource.DataSourceStatus {
	args := m.Called()
	return args.Get(0).(map[string]*datasource.DataSourceStatus)
}

func (m *MockDataSourceManager) GetDataSourceStatus(dsID string) (*datasource.DataSourceStatus, error) {
	args := m.Called(dsID)
	return args.Get(0).(*datasource.DataSourceStatus), args.Error(1)
}

func (m *MockDataSourceManager) GetResidentDataSources() map[string]*datasource.DataSourceStatus {
	args := m.Called()
	return args.Get(0).(map[string]*datasource.DataSourceStatus)
}

func (m *MockDataSourceManager) RestartResidentDataSource(ctx context.Context, dsID string) error {
	args := m.Called(ctx, dsID)
	return args.Error(0)
}

// MockDataSource 模拟数据源
type MockDataSource struct {
	mock.Mock
}

func (m *MockDataSource) Init(ctx context.Context, config *models.DataSource) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockDataSource) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDataSource) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDataSource) Execute(ctx context.Context, request *datasource.ExecuteRequest) (*datasource.ExecuteResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*datasource.ExecuteResponse), args.Error(1)
}

func (m *MockDataSource) HealthCheck(ctx context.Context) (*datasource.HealthStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*datasource.HealthStatus), args.Error(1)
}

func (m *MockDataSource) GetType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDataSource) GetConfig() *models.DataSource {
	args := m.Called()
	return args.Get(0).(*models.DataSource)
}

func (m *MockDataSource) IsResident() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockInterfaceInfo 模拟接口信息
type MockInterfaceInfo struct {
	mock.Mock
}

func (m *MockInterfaceInfo) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetDataSourceID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetSchemaName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetTableName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetParameters() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockInterfaceInfo) ValidateRequest(request *ExecuteRequest) error {
	args := m.Called(request)
	return args.Error(0)
}

// InterfaceExecutorTestSuite 接口执行器测试套件
type InterfaceExecutorTestSuite struct {
	suite.Suite
	db             *gorm.DB
	mockDSManager  *MockDataSourceManager
	mockDataSource *MockDataSource
	mockInterface  *MockInterfaceInfo
	executor       *InterfaceExecutor
}

// SetupSuite 设置测试套件
func (suite *InterfaceExecutorTestSuite) SetupSuite() {
	// 设置内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// 创建测试表
	err = db.Exec(`
		CREATE TABLE test_schema.test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER
		)
	`).Error
	if err != nil {
		// 如果schema不存在，直接创建表
		err = db.Exec(`
			CREATE TABLE test_table (
				id INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				value INTEGER
			)
		`).Error
		suite.Require().NoError(err)
	}

	// 创建mock对象
	suite.mockDSManager = new(MockDataSourceManager)
	suite.mockDataSource = new(MockDataSource)
	suite.mockInterface = new(MockInterfaceInfo)

	// 创建执行器
	suite.executor = NewInterfaceExecutor(db, suite.mockDSManager)
}

// TearDownSuite 清理测试套件
func (suite *InterfaceExecutorTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Exec("DROP TABLE IF EXISTS test_table")
	}
}

// SetupTest 设置每个测试
func (suite *InterfaceExecutorTestSuite) SetupTest() {
	// 清空测试表
	suite.db.Exec("DELETE FROM test_table")

	// 重置mock对象
	suite.mockDSManager.ExpectedCalls = nil
	suite.mockDataSource.ExpectedCalls = nil
	suite.mockInterface.ExpectedCalls = nil
}

// TestNewInterfaceExecutor 测试创建接口执行器
func (suite *InterfaceExecutorTestSuite) TestNewInterfaceExecutor() {
	executor := NewInterfaceExecutor(suite.db, suite.mockDSManager)

	assert.NotNil(suite.T(), executor)
	assert.NotNil(suite.T(), executor.db)
	assert.NotNil(suite.T(), executor.datasourceManager)
	assert.NotNil(suite.T(), executor.dataSyncEngine)
	assert.NotNil(suite.T(), executor.errorHandler)
}

// TestValidateRequest 测试请求验证
func (suite *InterfaceExecutorTestSuite) TestValidateRequest() {
	testCases := []struct {
		name        string
		request     *ExecuteRequest
		expectError bool
	}{
		{
			name: "有效的预览请求",
			request: &ExecuteRequest{
				InterfaceID:   "test-interface",
				InterfaceType: "basic_library",
				ExecuteType:   "preview",
				Limit:         10,
			},
			expectError: false,
		},
		{
			name: "空接口ID",
			request: &ExecuteRequest{
				InterfaceID:   "",
				InterfaceType: "basic_library",
				ExecuteType:   "preview",
			},
			expectError: true,
		},
		{
			name: "无效的接口类型",
			request: &ExecuteRequest{
				InterfaceID:   "test-interface",
				InterfaceType: "invalid_type",
				ExecuteType:   "preview",
			},
			expectError: true,
		},
		{
			name: "无效的执行类型",
			request: &ExecuteRequest{
				InterfaceID:   "test-interface",
				InterfaceType: "basic_library",
				ExecuteType:   "invalid_execute",
			},
			expectError: true,
		},
		{
			name: "增量同步缺少必要参数",
			request: &ExecuteRequest{
				InterfaceID:   "test-interface",
				InterfaceType: "basic_library",
				ExecuteType:   "incremental_sync",
				// 缺少 LastSyncTime 和 IncrementalKey
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.executor.validateRequest(tc.request)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProcessBatch 测试批次处理
func (suite *InterfaceExecutorTestSuite) TestProcessBatch() {
	// 准备测试数据
	testBatch := []map[string]interface{}{
		{"id": 1, "name": "test1", "value": 100},
		{"id": 2, "name": "test2", "value": 200},
	}

	tx := suite.db.Begin()
	defer tx.Rollback()

	err := suite.executor.processBatch(tx, testBatch, "test_table")
	assert.NoError(suite.T(), err)
}

// TestProcessRow 测试行处理
func (suite *InterfaceExecutorTestSuite) TestProcessRow() {
	testCases := []struct {
		name        string
		row         map[string]interface{}
		expectError bool
	}{
		{
			name:        "有效的行数据",
			row:         map[string]interface{}{"id": 1, "name": "test", "value": 100},
			expectError: false,
		},
		{
			name:        "空行数据",
			row:         map[string]interface{}{},
			expectError: false, // 空行应该被忽略，不报错
		},
		{
			name:        "包含空列名的行",
			row:         map[string]interface{}{"": "value", "name": "test"},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			tx := suite.db.Begin()
			defer tx.Rollback()

			err := suite.executor.processRow(tx, tc.row, "test_table")
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpsertDataToTable 测试数据UPSERT操作
func (suite *InterfaceExecutorTestSuite) TestUpsertDataToTable() {
	// 准备测试数据
	testData := []map[string]interface{}{
		{"id": 1, "name": "test1", "value": 100},
		{"id": 2, "name": "test2", "value": 200},
		{"id": 3, "name": "test3", "value": 300},
	}

	// 执行UPSERT操作
	insertedRows, err := suite.executor.upsertDataToTable(testData, "", "test_table")

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), insertedRows)

	// 验证数据库中的数据
	var count int64
	err = suite.db.Table("test_table").Count(&count).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), count)
}

// TestUpsertDataToTableWithEmptyData 测试空数据UPSERT
func (suite *InterfaceExecutorTestSuite) TestUpsertDataToTableWithEmptyData() {
	var testData []map[string]interface{}

	insertedRows, err := suite.executor.upsertDataToTable(testData, "", "test_table")

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), insertedRows)
}

// TestUpsertDataToTableWithLargeData 测试大数据量UPSERT
func (suite *InterfaceExecutorTestSuite) TestUpsertDataToTableWithLargeData() {
	// 准备大量数据（超过批次大小）
	testData := make([]map[string]interface{}, 2500)
	for i := 0; i < 2500; i++ {
		testData[i] = map[string]interface{}{
			"id":    i + 1,
			"name":  fmt.Sprintf("test%d", i+1),
			"value": (i + 1) * 10,
		}
	}

	// 执行UPSERT操作
	insertedRows, err := suite.executor.upsertDataToTable(testData, "", "test_table")

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2500), insertedRows)

	// 验证数据库中的数据
	var count int64
	err = suite.db.Table("test_table").Count(&count).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2500), count)
}

// TestInferDataTypes 测试数据类型推断
func (suite *InterfaceExecutorTestSuite) TestInferDataTypes() {
	testData := []map[string]interface{}{
		{
			"id":         1,
			"name":       "test",
			"price":      99.99,
			"is_active":  true,
			"created_at": time.Now(),
		},
		{
			"id":         2,
			"name":       "test2",
			"price":      199.99,
			"is_active":  false,
			"created_at": time.Now(),
		},
	}

	dataTypes := suite.executor.inferDataTypes(testData)

	assert.NotNil(suite.T(), dataTypes)
	assert.Contains(suite.T(), dataTypes, "id")
	assert.Contains(suite.T(), dataTypes, "name")
	assert.Contains(suite.T(), dataTypes, "price")
	assert.Contains(suite.T(), dataTypes, "is_active")
	assert.Contains(suite.T(), dataTypes, "created_at")

	// 验证数据类型推断
	assert.Equal(suite.T(), "INTEGER", dataTypes["id"])
	assert.Equal(suite.T(), "TEXT", dataTypes["name"])
	assert.Equal(suite.T(), "REAL", dataTypes["price"])
	assert.Equal(suite.T(), "BOOLEAN", dataTypes["is_active"])
	assert.Equal(suite.T(), "DATETIME", dataTypes["created_at"])
}

// TestInferDataTypesWithEmptyData 测试空数据的类型推断
func (suite *InterfaceExecutorTestSuite) TestInferDataTypesWithEmptyData() {
	var testData []map[string]interface{}

	dataTypes := suite.executor.inferDataTypes(testData)

	assert.NotNil(suite.T(), dataTypes)
	assert.Empty(suite.T(), dataTypes)
}

// TestInferDataTypesWithNilValues 测试包含nil值的数据类型推断
func (suite *InterfaceExecutorTestSuite) TestInferDataTypesWithNilValues() {
	testData := []map[string]interface{}{
		{
			"id":       1,
			"name":     nil,
			"optional": nil,
		},
		{
			"id":       2,
			"name":     "test",
			"optional": "value",
		},
	}

	dataTypes := suite.executor.inferDataTypes(testData)

	assert.NotNil(suite.T(), dataTypes)
	assert.Equal(suite.T(), "INTEGER", dataTypes["id"])
	assert.Equal(suite.T(), "TEXT", dataTypes["name"])
	assert.Equal(suite.T(), "TEXT", dataTypes["optional"])
}

// 运行测试套件
func TestInterfaceExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(InterfaceExecutorTestSuite))
}

// 基准测试
func BenchmarkUpsertDataToTable(b *testing.B) {
	// 设置测试环境
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.Exec("CREATE TABLE bench_table (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)")

	mockManager := new(MockDataSourceManager)
	executor := NewInterfaceExecutor(db, mockManager)

	// 准备测试数据
	testData := []map[string]interface{}{
		{"id": 1, "name": "bench1", "value": 100},
		{"id": 2, "name": "bench2", "value": 200},
		{"id": 3, "name": "bench3", "value": 300},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 清空表
		db.Exec("DELETE FROM bench_table")

		// 执行UPSERT
		_, err := executor.upsertDataToTable(testData, "", "bench_table")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInferDataTypes(b *testing.B) {
	mockManager := new(MockDataSourceManager)
	executor := NewInterfaceExecutor(nil, mockManager)

	testData := []map[string]interface{}{
		{"id": 1, "name": "test1", "value": 100, "price": 99.99, "active": true},
		{"id": 2, "name": "test2", "value": 200, "price": 199.99, "active": false},
		{"id": 3, "name": "test3", "value": 300, "price": 299.99, "active": true},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		executor.inferDataTypes(testData)
	}
}
