/*
 * @module service/interface_executor/executor_test
 * @description InterfaceExecutor的单元测试，使用实际接口配置进行测试
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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func (m *MockDataSource) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDataSource) IsResident() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDataSource) IsInitialized() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDataSource) IsStarted() bool {
	args := m.Called()
	return args.Bool(0)
}

// TestInterfaceConfig 测试接口配置结构体
type TestInterfaceConfig struct {
	ID                string                 `json:"id"`
	LibraryID         string                 `json:"library_id"`
	NameZh            string                 `json:"name_zh"`
	NameEn            string                 `json:"name_en"`
	Type              string                 `json:"type"`
	Description       string                 `json:"description"`
	CreatedAt         string                 `json:"created_at"`
	CreatedBy         string                 `json:"created_by"`
	UpdatedAt         string                 `json:"updated_at"`
	UpdatedBy         string                 `json:"updated_by"`
	Status            string                 `json:"status"`
	IsTableCreated    bool                   `json:"is_table_created"`
	DataSourceID      string                 `json:"data_source_id"`
	InterfaceConfig   map[string]interface{} `json:"interface_config"`
	ParseConfig       map[string]interface{} `json:"parse_config"`
	TableFieldsConfig interface{}            `json:"table_fields_config"`
	BasicLibrary      struct {
		ID          string `json:"id"`
		NameZh      string `json:"name_zh"`
		NameEn      string `json:"name_en"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
		CreatedBy   string `json:"created_by"`
		UpdatedAt   string `json:"updated_at"`
		UpdatedBy   string `json:"updated_by"`
		Status      string `json:"status"`
	} `json:"basic_library"`
	DataSource struct {
		ID               string                 `json:"id"`
		LibraryID        string                 `json:"library_id"`
		Name             string                 `json:"name"`
		Category         string                 `json:"category"`
		Type             string                 `json:"type"`
		Status           string                 `json:"status"`
		ConnectionConfig map[string]interface{} `json:"connection_config"`
		ParamsConfig     map[string]interface{} `json:"params_config"`
		Script           string                 `json:"script"`
		ScriptEnabled    bool                   `json:"script_enabled"`
		CreatedAt        string                 `json:"created_at"`
		CreatedBy        string                 `json:"created_by"`
		UpdatedAt        string                 `json:"updated_at"`
		UpdatedBy        string                 `json:"updated_by"`
		BasicLibrary     struct {
			ID          string `json:"id"`
			NameZh      string `json:"name_zh"`
			NameEn      string `json:"name_en"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
			CreatedBy   string `json:"created_by"`
			UpdatedAt   string `json:"updated_at"`
			UpdatedBy   string `json:"updated_by"`
			Status      string `json:"status"`
		} `json:"basic_library"`
	} `json:"data_source"`
}

// MockInterfaceInfo 模拟接口信息
type MockInterfaceInfo struct {
	mock.Mock
	testConfig *TestInterfaceConfig
}

// loadTestInterfaceConfig 加载测试接口配置
func loadTestInterfaceConfig() (*TestInterfaceConfig, error) {
	// 获取当前文件所在目录
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 构建测试配置文件路径
	testConfigPath := filepath.Join(currentDir, "tests", "testinterface.json")

	// 读取配置文件
	data, err := os.ReadFile(testConfigPath)
	if err != nil {
		return nil, fmt.Errorf("读取测试配置文件失败: %w", err)
	}

	// 解析JSON配置
	var config TestInterfaceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析测试配置文件失败: %w", err)
	}

	return &config, nil
}

func (m *MockInterfaceInfo) GetID() string {
	if m.testConfig != nil {
		return m.testConfig.ID
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetName() string {
	if m.testConfig != nil {
		return m.testConfig.NameZh
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetType() string {
	if m.testConfig != nil {
		return m.testConfig.Type
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetDataSourceID() string {
	if m.testConfig != nil {
		return m.testConfig.DataSourceID
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetSchemaName() string {
	if m.testConfig != nil {
		// 测试中返回空字符串，因为SQLite不支持schema语法
		return ""
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetTableName() string {
	if m.testConfig != nil {
		// 测试中使用下划线格式的表名，与SetupSuite中创建的表名一致
		return fmt.Sprintf("%s_%s", m.testConfig.BasicLibrary.NameEn, m.testConfig.NameEn)
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockInterfaceInfo) GetInterfaceConfig() map[string]interface{} {
	if m.testConfig != nil {
		return m.testConfig.InterfaceConfig
	}
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockInterfaceInfo) GetParseConfig() map[string]interface{} {
	if m.testConfig != nil {
		return m.testConfig.ParseConfig
	}
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockInterfaceInfo) GetTableFieldsConfig() []interface{} {
	if m.testConfig != nil {
		if m.testConfig.TableFieldsConfig == nil {
			return []interface{}{}
		}
		if arr, ok := m.testConfig.TableFieldsConfig.([]interface{}); ok {
			return arr
		}
		return []interface{}{m.testConfig.TableFieldsConfig}
	}
	args := m.Called()
	return args.Get(0).([]interface{})
}

func (m *MockInterfaceInfo) IsTableCreated() bool {
	if m.testConfig != nil {
		return true // 测试中总是返回true，表示表已创建
	}
	args := m.Called()
	return args.Bool(0)
}

// InterfaceExecutorTestSuite 接口执行器测试套件
type InterfaceExecutorTestSuite struct {
	suite.Suite
	db             *gorm.DB
	mockDSManager  *MockDataSourceManager
	mockDataSource *MockDataSource
	mockInterface  *MockInterfaceInfo
	executor       *InterfaceExecutor
	testConfig     *TestInterfaceConfig
}

// SetupSuite 设置测试套件
func (suite *InterfaceExecutorTestSuite) SetupSuite() {
	// 加载测试接口配置
	testConfig, err := loadTestInterfaceConfig()
	suite.Require().NoError(err)
	suite.testConfig = testConfig

	// 设置内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// 创建测试需要的表
	// 1. 创建通用测试表
	err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER
		)
	`).Error
	suite.Require().NoError(err)

	// 2. 根据测试配置创建接口表
	schemaName := testConfig.BasicLibrary.NameEn
	tableName := testConfig.NameEn

	err = db.Exec(fmt.Sprintf(`
		CREATE TABLE "%s_%s" (
			id INTEGER PRIMARY KEY,
			hotel_id TEXT,
			hotel_name TEXT,
			room_count INTEGER,
			occupancy_rate REAL,
			created_at TEXT,
			updated_at TEXT
		)
	`, schemaName, tableName)).Error
	suite.Require().NoError(err)

	// 3. 创建数据源表（用于测试）
	err = db.Exec(`
		CREATE TABLE data_sources (
			id TEXT PRIMARY KEY,
			library_id TEXT,
			name TEXT,
			category TEXT,
			type TEXT,
			status TEXT,
			connection_config TEXT,
			params_config TEXT,
			script TEXT,
			script_enabled INTEGER,
			created_at TEXT,
			created_by TEXT,
			updated_at TEXT,
			updated_by TEXT
		)
	`).Error
	suite.Require().NoError(err)

	// 4. 插入测试数据源数据，使用GORM的Create方法
	createdAt, _ := time.Parse("2006-01-02T15:04:05Z", testConfig.DataSource.CreatedAt)
	updatedAt, _ := time.Parse("2006-01-02T15:04:05Z", testConfig.DataSource.UpdatedAt)

	dataSource := models.DataSource{
		ID:               testConfig.DataSource.ID,
		LibraryID:        testConfig.DataSource.LibraryID,
		Name:             testConfig.DataSource.Name,
		Category:         testConfig.DataSource.Category,
		Type:             testConfig.DataSource.Type,
		Status:           testConfig.DataSource.Status,
		ConnectionConfig: testConfig.DataSource.ConnectionConfig,
		ParamsConfig:     testConfig.DataSource.ParamsConfig,
		Script:           testConfig.DataSource.Script,
		ScriptEnabled:    testConfig.DataSource.ScriptEnabled,
		CreatedAt:        createdAt,
		CreatedBy:        testConfig.DataSource.CreatedBy,
		UpdatedAt:        updatedAt,
		UpdatedBy:        testConfig.DataSource.UpdatedBy,
	}
	err = db.Create(&dataSource).Error
	suite.Require().NoError(err)

	// 创建mock对象
	suite.mockDSManager = new(MockDataSourceManager)
	suite.mockDataSource = new(MockDataSource)
	suite.mockInterface = &MockInterfaceInfo{testConfig: testConfig}

	// 创建执行器
	suite.executor = NewInterfaceExecutor(db, suite.mockDSManager)
}

// TearDownSuite 清理测试套件
func (suite *InterfaceExecutorTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Exec("DROP TABLE IF EXISTS test_table")
		suite.db.Exec("DROP TABLE IF EXISTS data_sources")
		if suite.testConfig != nil {
			schemaName := suite.testConfig.BasicLibrary.NameEn
			tableName := suite.testConfig.NameEn
			suite.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS \"%s_%s\"", schemaName, tableName))
		}
	}
}

// SetupTest 设置每个测试
func (suite *InterfaceExecutorTestSuite) SetupTest() {
	// 清空测试表
	suite.db.Exec("DELETE FROM test_table")

	if suite.testConfig != nil {
		schemaName := suite.testConfig.BasicLibrary.NameEn
		tableName := suite.testConfig.NameEn
		suite.db.Exec(fmt.Sprintf("DELETE FROM \"%s_%s\"", schemaName, tableName))
	}

	// 重置mock对象
	suite.mockDSManager.ExpectedCalls = nil
	suite.mockDataSource.ExpectedCalls = nil
	if suite.mockInterface != nil {
		suite.mockInterface.ExpectedCalls = nil
	}
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

// TestExecuteWithRealInterfaceConfig 使用真实接口配置测试执行
func (suite *InterfaceExecutorTestSuite) TestExecuteWithRealInterfaceConfig() {
	// 设置mock数据源返回测试数据
	mockResponseData := []map[string]interface{}{
		{
			"hotel_id":       "hotel_001",
			"hotel_name":     "测试酒店1",
			"room_count":     100,
			"occupancy_rate": 0.85,
			"created_at":     "2025-09-24T12:18:34.713+08:00",
			"updated_at":     "2025-09-24T12:19:22.238833+08:00",
		},
		{
			"hotel_id":       "hotel_002",
			"hotel_name":     "测试酒店2",
			"room_count":     150,
			"occupancy_rate": 0.92,
			"created_at":     "2025-09-24T12:18:34.713+08:00",
			"updated_at":     "2025-09-24T12:19:22.238833+08:00",
		},
	}

	// 设置mock数据源
	suite.mockDataSource.On("GetID").Return(suite.testConfig.DataSourceID)
	suite.mockDataSource.On("Execute", mock.Anything, mock.Anything).Return(
		&datasource.ExecuteResponse{
			Success: true,
			Message: "查询成功",
			Data:    mockResponseData,
		}, nil)

	// 设置mock数据源管理器
	suite.mockDSManager.On("Get", suite.testConfig.DataSourceID).Return(suite.mockDataSource, nil)

	// 测试预览操作
	suite.T().Run("预览操作", func(t *testing.T) {
		request := &ExecuteRequest{
			InterfaceID:   suite.testConfig.ID,
			InterfaceType: "basic_library",
			ExecuteType:   "preview",
			Parameters: map[string]interface{}{
				"hotelGroupCode": "GCBZ",
				"exec":           "Kpi_Ihotel_Master_in",
			},
			Limit: 10,
		}

		// 模拟接口信息提供者返回测试配置
		suite.executor.infoProvider = &MockInterfaceInfoProvider{
			testConfig: suite.testConfig,
		}

		response, err := suite.executor.Execute(context.Background(), request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Success)
		assert.Equal(t, "preview", response.ExecuteType)
		assert.Equal(t, 2, response.RowCount)
		assert.NotNil(t, response.Data)

		// 验证元数据
		assert.NotNil(t, response.Metadata)
		assert.Equal(t, suite.testConfig.ID, response.Metadata["interface_id"])
		assert.Equal(t, suite.testConfig.NameZh, response.Metadata["interface_name"])
	})

	// 测试同步操作
	suite.T().Run("同步操作", func(t *testing.T) {
		request := &ExecuteRequest{
			InterfaceID:   suite.testConfig.ID,
			InterfaceType: "basic_library",
			ExecuteType:   "sync",
			Parameters: map[string]interface{}{
				"hotelGroupCode": "GCBZ",
				"exec":           "Kpi_Ihotel_Master_in",
			},
			SyncStrategy: "full",
		}

		// 模拟接口信息提供者返回测试配置
		suite.executor.infoProvider = &MockInterfaceInfoProvider{
			testConfig: suite.testConfig,
		}

		response, err := suite.executor.Execute(context.Background(), request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Success)
		assert.Equal(t, "sync", response.ExecuteType)
		assert.Equal(t, 2, response.RowCount)
		assert.NotNil(t, response.Data)

		// 验证元数据
		assert.NotNil(t, response.Metadata)
		assert.Equal(t, suite.testConfig.ID, response.Metadata["interface_id"])
		assert.Equal(t, suite.testConfig.NameZh, response.Metadata["interface_name"])
	})
}

// TestInterfaceConfigValidation 测试接口配置验证
func (suite *InterfaceExecutorTestSuite) TestInterfaceConfigValidation() {
	suite.T().Run("验证接口配置结构", func(t *testing.T) {
		config := suite.testConfig

		// 验证基本信息
		assert.NotEmpty(t, config.ID)
		assert.NotEmpty(t, config.NameZh)
		assert.NotEmpty(t, config.NameEn)
		assert.Equal(t, "api", config.Type)

		// 验证接口配置
		assert.NotNil(t, config.InterfaceConfig)
		assert.Equal(t, "POST", config.InterfaceConfig["method"])
		assert.Equal(t, "application/json", config.InterfaceConfig["content_type"])
		assert.Equal(t, true, config.InterfaceConfig["use_form_data"])

		// 验证数据源配置
		assert.NotEmpty(t, config.DataSource.ID)
		assert.Equal(t, "api", config.DataSource.Category)
		assert.Equal(t, "http_with_auth", config.DataSource.Type)
		assert.NotNil(t, config.DataSource.ConnectionConfig)

		// 验证基础库信息
		assert.NotEmpty(t, config.BasicLibrary.NameEn)
		assert.Equal(t, "basic_things", config.BasicLibrary.NameEn)
	})
}

// TestDataProcessingWithRealConfig 测试使用真实配置的数据处理
func (suite *InterfaceExecutorTestSuite) TestDataProcessingWithRealConfig() {
	suite.T().Run("数据处理流程", func(t *testing.T) {
		// 创建数据处理器
		dataProcessor := NewDataProcessor(suite.executor)

		// 模拟API响应数据
		testData := []map[string]interface{}{
			{
				"hotelId":       "H001",
				"hotelName":     "绿云测试酒店",
				"totalRooms":    200,
				"occupiedRooms": 170,
				"occupancyRate": 0.85,
				"lastUpdated":   "2025-09-24T12:19:22.238833+08:00",
			},
		}

		// 分析数据类型
		dataTypes := dataProcessor.AnalyzeDataTypes(testData)

		assert.NotNil(t, dataTypes)
		assert.Contains(t, dataTypes, "hotelId")
		assert.Contains(t, dataTypes, "hotelName")
		assert.Contains(t, dataTypes, "totalRooms")
		assert.Contains(t, dataTypes, "occupiedRooms")
		assert.Contains(t, dataTypes, "occupancyRate")
		assert.Contains(t, dataTypes, "lastUpdated")

		// 验证数据类型推断
		assert.Equal(t, "string", dataTypes["hotelId"])
		assert.Equal(t, "string", dataTypes["hotelName"])
		assert.Equal(t, "integer", dataTypes["totalRooms"])
		assert.Equal(t, "integer", dataTypes["occupiedRooms"])
		assert.Equal(t, "float", dataTypes["occupancyRate"])
		// 注意：AnalyzeDataTypes使用简单的字段名检测，lastUpdated不包含time/date关键词
		// 所以会被识别为string，这是正常的
		assert.Equal(t, "string", dataTypes["lastUpdated"])
	})
}

// TestFieldMappingWithRealConfig 测试字段映射
func (suite *InterfaceExecutorTestSuite) TestFieldMappingWithRealConfig() {
	suite.T().Run("字段映射处理", func(t *testing.T) {
		fieldMapper := NewFieldMapper()

		// 模拟原始数据
		originalData := map[string]interface{}{
			"hotelId":       "H001",
			"hotelName":     "测试酒店",
			"totalRooms":    100,
			"occupiedRooms": 85,
			"updateTime":    "2025-09-24T12:19:22.238833+08:00",
		}

		// 模拟字段映射配置
		parseConfig := map[string]interface{}{
			"fieldMapping": []interface{}{
				map[string]interface{}{"source": "hotelId", "target": "hotel_id"},
				map[string]interface{}{"source": "hotelName", "target": "hotel_name"},
				map[string]interface{}{"source": "totalRooms", "target": "total_rooms"},
				map[string]interface{}{"source": "occupiedRooms", "target": "occupied_rooms"},
				map[string]interface{}{"source": "updateTime", "target": "updated_at"},
			},
		}

		// 应用字段映射
		mappedData := fieldMapper.ApplyFieldMapping(originalData, parseConfig, true)

		// 验证映射结果
		assert.Contains(t, mappedData, "hotel_id")
		assert.Contains(t, mappedData, "hotel_name")
		assert.Contains(t, mappedData, "total_rooms")
		assert.Contains(t, mappedData, "occupied_rooms")
		assert.Contains(t, mappedData, "updated_at")

		assert.Equal(t, "H001", mappedData["hotel_id"])
		assert.Equal(t, "测试酒店", mappedData["hotel_name"])
		assert.Equal(t, 100, mappedData["total_rooms"])
		assert.Equal(t, 85, mappedData["occupied_rooms"])
		assert.Equal(t, "2025-09-24T12:19:22.238833+08:00", mappedData["updated_at"])
	})
}

// MockInterfaceInfoProvider 模拟接口信息提供者
type MockInterfaceInfoProvider struct {
	testConfig *TestInterfaceConfig
}

// 确保MockInterfaceInfoProvider实现InterfaceInfoProviderInterface接口
var _ InterfaceInfoProviderInterface = (*MockInterfaceInfoProvider)(nil)

func (m *MockInterfaceInfoProvider) GetBasicLibraryInterface(interfaceID string) (InterfaceInfo, error) {
	if m.testConfig != nil && m.testConfig.ID == interfaceID {
		return &MockInterfaceInfo{testConfig: m.testConfig}, nil
	}
	return nil, fmt.Errorf("接口未找到: %s", interfaceID)
}

func (m *MockInterfaceInfoProvider) GetThematicLibraryInterface(interfaceID string) (InterfaceInfo, error) {
	return nil, fmt.Errorf("主题库接口未实现")
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
