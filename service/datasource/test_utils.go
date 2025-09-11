/*
 * @module service/basic_library/datasource/test_utils
 * @description 数据源测试工具，提供测试辅助函数和Mock实现
 * @architecture 测试辅助模式 - 提供通用的测试工具和Mock对象
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 测试工具生命周期：创建Mock -> 设置期望 -> 执行测试 -> 验证结果
 * @rules 仅用于测试环境，提供数据源的Mock实现和测试辅助函数
 * @dependencies testing, context, time
 * @refs interface.go, base.go
 */

package datasource

import (
	"context"
	"fmt"
	"sync"
	"time"

	"datahub-service/service/models"
)

// MockDataSource Mock数据源实现，用于测试
type MockDataSource struct {
	*BaseDataSource
	initCalled        bool
	startCalled       bool
	stopCalled        bool
	executeCalled     bool
	healthCheckCalled bool

	initError        error
	startError       error
	stopError        error
	executeError     error
	healthCheckError error

	executeResponse *ExecuteResponse
	healthStatus    *HealthStatus

	mu sync.RWMutex
}

// NewMockDataSource 创建Mock数据源
func NewMockDataSource(dsType string, isResident bool) *MockDataSource {
	base := NewBaseDataSource(dsType, isResident)
	return &MockDataSource{
		BaseDataSource: base,
		executeResponse: &ExecuteResponse{
			Success:   true,
			Data:      "mock data",
			RowCount:  1,
			Message:   "mock success",
			Timestamp: time.Now(),
		},
		healthStatus: &HealthStatus{
			Status:       "online",
			Message:      "mock healthy",
			LastCheck:    time.Now(),
			ResponseTime: 10 * time.Millisecond,
			Details:      make(map[string]interface{}),
		},
	}
}

// Init Mock初始化
func (m *MockDataSource) Init(ctx context.Context, ds *models.DataSource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initCalled = true
	if m.initError != nil {
		return m.initError
	}

	return m.BaseDataSource.Init(ctx, ds)
}

// Start Mock启动
func (m *MockDataSource) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startCalled = true
	if m.startError != nil {
		return m.startError
	}

	return m.BaseDataSource.Start(ctx)
}

// Execute Mock执行
func (m *MockDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executeCalled = true
	if m.executeError != nil {
		return nil, m.executeError
	}

	// 复制响应以避免并发问题
	response := &ExecuteResponse{
		Success:   m.executeResponse.Success,
		Data:      m.executeResponse.Data,
		RowCount:  m.executeResponse.RowCount,
		Message:   m.executeResponse.Message,
		Error:     m.executeResponse.Error,
		Metadata:  make(map[string]interface{}),
		Duration:  10 * time.Millisecond,
		Timestamp: time.Now(),
	}

	// 复制元数据
	for k, v := range m.executeResponse.Metadata {
		response.Metadata[k] = v
	}

	return response, nil
}

// Stop Mock停止
func (m *MockDataSource) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopCalled = true
	if m.stopError != nil {
		return m.stopError
	}

	return m.BaseDataSource.Stop(ctx)
}

// HealthCheck Mock健康检查
func (m *MockDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.healthCheckCalled = true
	if m.healthCheckError != nil {
		return nil, m.healthCheckError
	}

	// 复制状态以避免并发问题
	status := &HealthStatus{
		Status:       m.healthStatus.Status,
		Message:      m.healthStatus.Message,
		LastCheck:    time.Now(),
		ResponseTime: m.healthStatus.ResponseTime,
		Details:      make(map[string]interface{}),
	}

	// 复制详情
	for k, v := range m.healthStatus.Details {
		status.Details[k] = v
	}

	return status, nil
}

// 设置Mock行为的方法

// SetInitError 设置初始化错误
func (m *MockDataSource) SetInitError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initError = err
}

// SetStartError 设置启动错误
func (m *MockDataSource) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startError = err
}

// SetStopError 设置停止错误
func (m *MockDataSource) SetStopError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopError = err
}

// SetExecuteError 设置执行错误
func (m *MockDataSource) SetExecuteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeError = err
}

// SetHealthCheckError 设置健康检查错误
func (m *MockDataSource) SetHealthCheckError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCheckError = err
}

// SetExecuteResponse 设置执行响应
func (m *MockDataSource) SetExecuteResponse(response *ExecuteResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeResponse = response
}

// SetHealthStatus 设置健康状态
func (m *MockDataSource) SetHealthStatus(status *HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthStatus = status
}

// 验证Mock调用的方法

// WasInitCalled 检查是否调用了Init
func (m *MockDataSource) WasInitCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initCalled
}

// WasStartCalled 检查是否调用了Start
func (m *MockDataSource) WasStartCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.startCalled
}

// WasStopCalled 检查是否调用了Stop
func (m *MockDataSource) WasStopCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopCalled
}

// WasExecuteCalled 检查是否调用了Execute
func (m *MockDataSource) WasExecuteCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.executeCalled
}

// WasHealthCheckCalled 检查是否调用了HealthCheck
func (m *MockDataSource) WasHealthCheckCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthCheckCalled
}

// ResetCalls 重置所有调用标记
func (m *MockDataSource) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initCalled = false
	m.startCalled = false
	m.stopCalled = false
	m.executeCalled = false
	m.healthCheckCalled = false
}

// TestDataSourceConfig 测试数据源配置
type TestDataSourceConfig struct {
	ID               string
	Name             string
	Category         string
	Type             string
	ConnectionConfig map[string]interface{}
	ParamsConfig     map[string]interface{}
	Script           string
	ScriptEnabled    bool
}

// CreateTestDataSource 创建测试数据源模型
func CreateTestDataSource(config TestDataSourceConfig) *models.DataSource {
	ds := &models.DataSource{
		ID:               config.ID,
		LibraryID:        "test-library-id",
		Name:             config.Name,
		Category:         config.Category,
		Type:             config.Type,
		ConnectionConfig: config.ConnectionConfig,
		ParamsConfig:     config.ParamsConfig,
		Script:           config.Script,
		ScriptEnabled:    config.ScriptEnabled,
		CreatedAt:        time.Now(),
		CreatedBy:        "test",
		UpdatedAt:        time.Now(),
		UpdatedBy:        "test",
	}

	// 设置默认值
	if ds.ID == "" {
		ds.ID = "test-datasource-id"
	}
	if ds.Name == "" {
		ds.Name = "Test DataSource"
	}
	if ds.Category == "" {
		ds.Category = "test"
	}
	if ds.Type == "" {
		ds.Type = "mock"
	}
	if ds.ConnectionConfig == nil {
		ds.ConnectionConfig = make(map[string]interface{})
	}
	if ds.ParamsConfig == nil {
		ds.ParamsConfig = make(map[string]interface{})
	}

	return ds
}

// CreateTestExecuteRequest 创建测试执行请求
func CreateTestExecuteRequest(operation string, query string, data interface{}) *ExecuteRequest {
	return &ExecuteRequest{
		Operation: operation,
		Query:     query,
		Data:      data,
		Params:    make(map[string]interface{}),
		Timeout:   30 * time.Second,
	}
}

// AssertExecuteResponse 断言执行响应
func AssertExecuteResponse(t interface{}, response *ExecuteResponse, expectedSuccess bool, expectedError string) {
	// 这里使用interface{}类型以避免在非测试包中导入testing包
	// 实际使用时应该传入*testing.T

	if response == nil {
		panic("response is nil")
	}

	if response.Success != expectedSuccess {
		panic(fmt.Sprintf("expected success %v, got %v", expectedSuccess, response.Success))
	}

	if expectedError != "" && response.Error != expectedError {
		panic(fmt.Sprintf("expected error '%s', got '%s'", expectedError, response.Error))
	}
}

// AssertHealthStatus 断言健康状态
func AssertHealthStatus(t interface{}, status *HealthStatus, expectedStatus string, expectedMessage string) {
	if status == nil {
		panic("status is nil")
	}

	if status.Status != expectedStatus {
		panic(fmt.Sprintf("expected status '%s', got '%s'", expectedStatus, status.Status))
	}

	if expectedMessage != "" && status.Message != expectedMessage {
		panic(fmt.Sprintf("expected message '%s', got '%s'", expectedMessage, status.Message))
	}
}

// WaitForCondition 等待条件满足
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待条件超时")
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}
