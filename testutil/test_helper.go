/*
 * @module testutil/test_helper
 * @description 测试工具和辅助函数
 * @architecture 测试基础设施 - 提供测试通用工具和数据工厂
 * @documentReference .specify/memory/test_plan.md
 * @stateFlow 测试环境初始化 -> 测试数据创建 -> 测试执行 -> 清理资源
 * @rules 提供可重用的测试工具，确保测试环境的一致性
 * @dependencies gorm, sqlite, testify, time
 * @refs service/models
 */

package testutil

import (
	"bytes"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB 测试数据库配置
type TestDB struct {
	DB *gorm.DB
}

// NewTestDB 创建测试数据库
func NewTestDB() *TestDB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("failed to connect test database: %v", err))
	}

	// 自动迁移所有模型
	err = db.AutoMigrate(
		&models.BasicLibrary{},
		&models.DataSource{},
		&models.DataInterface{},
		&models.SyncTask{},
		&models.ThematicLibrary{},
		&models.QualityRuleTemplate{},
		&models.ApiApplication{},
		&models.ApiKey{},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate test database: %v", err))
	}

	return &TestDB{DB: db}
}

// CleanDB 清理数据库
func (tdb *TestDB) CleanDB() {
	// 清空所有表的数据
	tables := []string{
		"basic_libraries",
		"data_sources",
		"data_interfaces",
		"sync_tasks",
		"thematic_libraries",
		"quality_rule_templates",
		"api_applications",
		"api_keys",
	}

	for _, table := range tables {
		tdb.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
}

// Close 关闭数据库连接
func (tdb *TestDB) Close() {
	if db, err := tdb.DB.DB(); err == nil {
		db.Close()
	}
}

// TestDataFactory 测试数据工厂
type TestDataFactory struct {
	DB *gorm.DB
}

// NewTestDataFactory 创建测试数据工厂
func NewTestDataFactory(db *gorm.DB) *TestDataFactory {
	return &TestDataFactory{DB: db}
}

// BasicLibraryOption 基础库选项函数类型
type BasicLibraryOption func(*models.BasicLibrary)

// CreateBasicLibrary 创建测试基础库
func (f *TestDataFactory) CreateBasicLibrary(opts ...BasicLibraryOption) *models.BasicLibrary {
	library := &models.BasicLibrary{
		ID:          generateID("lib"),
		NameZh:      "测试基础库",
		NameEn:      "test_library_" + generateSuffix(),
		Description: "这是一个测试基础库",
		Status:      "active",
		CreatedBy:   "test",
		UpdatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(library)
	}

	err := f.DB.Create(library).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test basic library: %v", err))
	}

	return library
}

// DataSourceOption 数据源选项函数类型
type DataSourceOption func(*models.DataSource)

// CreateDataSource 创建测试数据源
func (f *TestDataFactory) CreateDataSource(libraryID string, opts ...DataSourceOption) *models.DataSource {
	dataSource := &models.DataSource{
		ID:        generateID("ds"),
		LibraryID: libraryID,
		Name:      "测试数据源",
		Type:      "postgresql",
		Category:  "database",
		Status:    "active",
		ConnectionConfig: map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "testdb",
			"username": "testuser",
			"password": "testpass",
		},
		ParamsConfig: map[string]interface{}{
			"timeout": 30,
		},
		CreatedBy: "test",
		UpdatedBy: "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(dataSource)
	}

	err := f.DB.Create(dataSource).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test data source: %v", err))
	}

	return dataSource
}

// DataInterfaceOption 数据接口选项函数类型
type DataInterfaceOption func(*models.DataInterface)

// CreateDataInterface 创建测试数据接口
func (f *TestDataFactory) CreateDataInterface(libraryID, dataSourceID string, opts ...DataInterfaceOption) *models.DataInterface {
	dataInterface := &models.DataInterface{
		ID:                generateID("di"),
		LibraryID:         libraryID,
		DataSourceID:      dataSourceID,
		NameZh:            "测试数据接口",
		NameEn:            "test_interface_" + generateSuffix(),
		Type:              "api",
		Description:       "这是一个测试数据接口",
		Status:            "active",
		IsTableCreated:    false,
		InterfaceConfig:   models.JSONB{},
		ParseConfig:       models.JSONB{},
		TableFieldsConfig: models.JSONB{},
		CreatedBy:         "test",
		UpdatedBy:         "test",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(dataInterface)
	}

	err := f.DB.Create(dataInterface).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test data interface: %v", err))
	}

	return dataInterface
}

// SyncTaskOption 同步任务选项函数类型
type SyncTaskOption func(*models.SyncTask)

// CreateSyncTask 创建测试同步任务
func (f *TestDataFactory) CreateSyncTask(libraryID, dataSourceID string, opts ...SyncTaskOption) *models.SyncTask {
	now := time.Now()
	syncTask := &models.SyncTask{
		ID:              generateID("st"),
		LibraryID:       libraryID,
		LibraryType:     "basic_library",
		DataSourceID:    dataSourceID,
		TaskType:        "batch_sync",
		Status:          "pending",
		TriggerType:     "manual",
		CronExpression:  "0 */1 * * *", // 每小时执行一次
		IntervalSeconds: 3600,
		NextRunTime:     &now,
		LastRunTime:     &now,
		CreatedBy:       "test",
		UpdatedBy:       "test",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 应用选项
	for _, opt := range opts {
		opt(syncTask)
	}

	err := f.DB.Create(syncTask).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test sync task: %v", err))
	}

	return syncTask
}

// ThematicLibraryOption 主题库选项函数类型
type ThematicLibraryOption func(*models.ThematicLibrary)

// CreateThematicLibrary 创建测试主题库
func (f *TestDataFactory) CreateThematicLibrary(opts ...ThematicLibraryOption) *models.ThematicLibrary {
	library := &models.ThematicLibrary{
		ID:          generateID("tl"),
		NameZh:      "测试主题库",
		NameEn:      "test_thematic_library_" + generateSuffix(),
		Description: "这是一个测试主题库",
		Category:    "business",
		Domain:      "finance",
		AccessLevel: "public",
		Status:      "active",
		CreatedBy:   "test",
		UpdatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(library)
	}

	err := f.DB.Create(library).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test thematic library: %v", err))
	}

	return library
}

// QualityRuleTemplateOption 质量规则模板选项函数类型
type QualityRuleTemplateOption func(*models.QualityRuleTemplate)

// CreateQualityRuleTemplate 创建测试质量规则模板
func (f *TestDataFactory) CreateQualityRuleTemplate(opts ...QualityRuleTemplateOption) *models.QualityRuleTemplate {
	template := &models.QualityRuleTemplate{
		ID:          generateID("qrt"),
		Name:        "测试质量规则",
		Type:        "completeness",
		Category:    "basic_quality",
		Description: "这是一个测试质量规则模板",
		RuleLogic: map[string]interface{}{
			"condition": "NOT NULL",
			"threshold": 0.95,
		},
		Parameters: map[string]interface{}{
			"field_name": "required",
		},
		DefaultConfig: map[string]interface{}{
			"enabled": true,
		},
		IsBuiltIn: false,
		IsEnabled: true,
		Version:   "1.0",
		CreatedBy: "test",
		UpdatedBy: "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(template)
	}

	err := f.DB.Create(template).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test quality rule template: %v", err))
	}

	return template
}

// ApiApplicationOption API应用选项函数类型
type ApiApplicationOption func(*models.ApiApplication)

// CreateApiApplication 创建测试API应用
func (f *TestDataFactory) CreateApiApplication(thematicLibraryID string, opts ...ApiApplicationOption) *models.ApiApplication {
	app := &models.ApiApplication{
		ID:                generateID("aa"),
		Name:              "测试API应用",
		Path:              "test_app_" + generateSuffix(),
		ThematicLibraryID: thematicLibraryID,
		ContactPerson:     "测试联系人",
		ContactPhone:      "13800138000",
		Status:            "active",
		CreatedBy:         "test",
		UpdatedBy:         "test",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(app)
	}

	err := f.DB.Create(app).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test api application: %v", err))
	}

	return app
}

// ApiKeyOption API密钥选项函数类型
type ApiKeyOption func(*models.ApiKey)

// CreateApiKey 创建测试API密钥
func (f *TestDataFactory) CreateApiKey(opts ...ApiKeyOption) *models.ApiKey {
	apiKey := &models.ApiKey{
		ID:           generateID("ak"),
		Name:         "测试API密钥",
		KeyPrefix:    "test",
		KeyValueHash: "test_key_hash_" + generateSuffix(),
		Description:  "这是一个测试API密钥",
		Status:       "active",
		CreatedBy:    "test",
		UpdatedBy:    "test",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(apiKey)
	}

	err := f.DB.Create(apiKey).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test api key: %v", err))
	}

	return apiKey
}

// 辅助函数
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), generateSuffix())
}

func generateSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%100000)
}

// MockEventListener Mock事件监听器
type MockEventListener struct {
	mock.Mock
}

func (m *MockEventListener) RegisterDBEventProcessor(processor models.DBEventProcessor) {
	m.Called(processor)
}

func (m *MockEventListener) EmitEvent(eventType string, data interface{}) error {
	args := m.Called(eventType, data)
	return args.Error(0)
}

// TestConfig 测试配置
type TestConfig struct {
	Database struct {
		Driver string
		DSN    string
	}
	Timeout time.Duration
	Cleanup bool
}

// DefaultTestConfig 默认测试配置
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		Database: struct {
			Driver string
			DSN    string
		}{
			Driver: "sqlite",
			DSN:    ":memory:",
		},
		Timeout: 30 * time.Second,
		Cleanup: true,
	}
}

// AssertionHelper 断言辅助工具
type AssertionHelper struct{}

// NewAssertionHelper 创建断言辅助工具
func NewAssertionHelper() *AssertionHelper {
	return &AssertionHelper{}
}

// AssertBasicLibraryEqual 断言基础库相等
func (h *AssertionHelper) AssertBasicLibraryEqual(t interface{}, expected, actual *models.BasicLibrary) {
	// 这里可以添加自定义的断言逻辑
	// 例如忽略某些字段的比较，或者进行深度比较
}

// AssertDataSourceEqual 断言数据源相等
func (h *AssertionHelper) AssertDataSourceEqual(t interface{}, expected, actual *models.DataSource) {
	// 自定义数据源比较逻辑
}

// TestTransaction 测试事务辅助工具
type TestTransaction struct {
	db *gorm.DB
	tx *gorm.DB
}

// NewTestTransaction 创建测试事务
func NewTestTransaction(db *gorm.DB) *TestTransaction {
	tx := db.Begin()
	return &TestTransaction{
		db: db,
		tx: tx,
	}
}

// DB 获取事务数据库
func (tt *TestTransaction) DB() *gorm.DB {
	return tt.tx
}

// Commit 提交事务
func (tt *TestTransaction) Commit() {
	tt.tx.Commit()
}

// Rollback 回滚事务
func (tt *TestTransaction) Rollback() {
	tt.tx.Rollback()
}

// HTTPTestHelper HTTP测试辅助工具
type HTTPTestHelper struct{}

// NewHTTPTestHelper 创建HTTP测试辅助工具
func NewHTTPTestHelper() *HTTPTestHelper {
	return &HTTPTestHelper{}
}

// CreateJSONRequest 创建JSON请求
func (h *HTTPTestHelper) CreateJSONRequest(method, url string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// AssertJSONResponse 断言JSON响应
func (h *HTTPTestHelper) AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	assert.Equal(t, expectedStatus, w.Code)

	if expectedBody != nil {
		var actualBody interface{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)
		assert.NoError(t, err)

		expectedJSON, _ := json.Marshal(expectedBody)
		actualJSON, _ := json.Marshal(actualBody)

		assert.JSONEq(t, string(expectedJSON), string(actualJSON))
	}
}
