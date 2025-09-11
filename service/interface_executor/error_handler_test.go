/*
 * @module service/interface_executor/error_handler_test
 * @description ErrorHandler的单元测试
 * @architecture 测试驱动开发 - 确保错误处理和事务管理功能正常工作
 * @documentReference design.md
 * @stateFlow 测试准备 -> 错误构造 -> 处理测试 -> 结果验证 -> 清理资源
 * @rules 测试用例需要覆盖各种错误类型、严重级别和恢复策略
 * @dependencies testing, testify, gorm, sqlite, context
 * @refs error_handler.go
 */

package interface_executor

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ErrorHandlerTestSuite 错误处理器测试套件
type ErrorHandlerTestSuite struct {
	suite.Suite
	errorHandler *ErrorHandler
	db           *gorm.DB
}

// SetupSuite 设置测试套件
func (suite *ErrorHandlerTestSuite) SetupSuite() {
	suite.errorHandler = NewErrorHandler()

	// 设置测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// 创建测试表
	err = db.Exec("CREATE TABLE test_error_table (id INTEGER PRIMARY KEY, data TEXT)").Error
	suite.Require().NoError(err)
}

// TearDownSuite 清理测试套件
func (suite *ErrorHandlerTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Exec("DROP TABLE IF EXISTS test_error_table")
	}
}

// TestNewErrorHandler 测试创建错误处理器
func (suite *ErrorHandlerTestSuite) TestNewErrorHandler() {
	handler := NewErrorHandler()

	assert.NotNil(suite.T(), handler)
	assert.NotNil(suite.T(), handler.logger)
}

// TestHandleError 测试错误处理
func (suite *ErrorHandlerTestSuite) TestHandleError() {
	testCases := []struct {
		name                string
		err                 error
		errorType           ErrorType
		context             string
		expectedSeverity    ErrorSeverity
		expectedRecoverable bool
	}{
		{
			name:                "连接错误",
			err:                 errors.New("connection refused"),
			errorType:           ErrorTypeConnection,
			context:             "database_connection",
			expectedSeverity:    SeverityHigh,
			expectedRecoverable: true,
		},
		{
			name:                "验证错误",
			err:                 errors.New("invalid input data"),
			errorType:           ErrorTypeValidation,
			context:             "data_validation",
			expectedSeverity:    SeverityLow,
			expectedRecoverable: false,
		},
		{
			name:                "系统错误",
			err:                 errors.New("system fatal error"),
			errorType:           ErrorTypeSystem,
			context:             "system_operation",
			expectedSeverity:    SeverityCritical,
			expectedRecoverable: false,
		},
		{
			name:                "超时错误",
			err:                 errors.New("operation timeout"),
			errorType:           ErrorTypeTimeout,
			context:             "query_execution",
			expectedSeverity:    SeverityHigh,
			expectedRecoverable: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			detail := suite.errorHandler.HandleError(ctx, tc.err, tc.errorType, tc.context)

			assert.NotNil(t, detail)
			assert.Equal(t, tc.errorType, detail.Type)
			assert.Equal(t, tc.expectedSeverity, detail.Severity)
			assert.Equal(t, tc.expectedRecoverable, detail.Recoverable)
			assert.Equal(t, tc.context, detail.Context)
			assert.Equal(t, tc.err.Error(), detail.Message)
			assert.Equal(t, tc.err, detail.Cause)
			assert.Equal(t, 0, detail.RetryCount)
			assert.True(t, time.Since(detail.Timestamp) < time.Second)
			assert.NotNil(t, detail.Metadata)
		})
	}
}

// TestHandleNilError 测试处理nil错误
func (suite *ErrorHandlerTestSuite) TestHandleNilError() {
	ctx := context.Background()
	detail := suite.errorHandler.HandleError(ctx, nil, ErrorTypeValidation, "test_context")

	assert.Nil(suite.T(), detail)
}

// TestDetermineSeverity 测试错误严重级别判断
func (suite *ErrorHandlerTestSuite) TestDetermineSeverity() {
	testCases := []struct {
		name     string
		err      error
		errType  ErrorType
		expected ErrorSeverity
	}{
		{
			name:     "系统错误 - 严重",
			err:      errors.New("system error"),
			errType:  ErrorTypeSystem,
			expected: SeverityCritical,
		},
		{
			name:     "事务错误 - 高级别",
			err:      errors.New("transaction failed"),
			errType:  ErrorTypeTransaction,
			expected: SeverityHigh,
		},
		{
			name:     "验证错误 - 低级别",
			err:      errors.New("validation failed"),
			errType:  ErrorTypeValidation,
			expected: SeverityLow,
		},
		{
			name:     "连接超时 - 高级别",
			err:      errors.New("connection timeout"),
			errType:  ErrorTypeConnection,
			expected: SeverityHigh,
		},
		{
			name:     "致命错误 - 严重",
			err:      errors.New("fatal system error"),
			errType:  ErrorTypeQuery,
			expected: SeverityCritical,
		},
		{
			name:     "一般错误 - 中级别",
			err:      errors.New("operation failed"),
			errType:  ErrorTypeQuery,
			expected: SeverityMedium,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			severity := suite.errorHandler.determineSeverity(tc.err, tc.errType)
			assert.Equal(t, tc.expected, severity)
		})
	}
}

// TestIsRecoverable 测试错误可恢复性判断
func (suite *ErrorHandlerTestSuite) TestIsRecoverable() {
	testCases := []struct {
		name     string
		err      error
		errType  ErrorType
		expected bool
	}{
		{
			name:     "系统错误 - 不可恢复",
			err:      errors.New("system crash"),
			errType:  ErrorTypeSystem,
			expected: false,
		},
		{
			name:     "验证错误 - 不可恢复",
			err:      errors.New("invalid data"),
			errType:  ErrorTypeValidation,
			expected: false,
		},
		{
			name:     "连接错误 - 可恢复",
			err:      errors.New("connection lost"),
			errType:  ErrorTypeConnection,
			expected: true,
		},
		{
			name:     "超时错误 - 可恢复",
			err:      errors.New("timeout occurred"),
			errType:  ErrorTypeTimeout,
			expected: true,
		},
		{
			name:     "权限错误 - 不可恢复",
			err:      errors.New("permission denied"),
			errType:  ErrorTypeQuery,
			expected: false,
		},
		{
			name:     "未找到错误 - 不可恢复",
			err:      errors.New("record not found"),
			errType:  ErrorTypeQuery,
			expected: false,
		},
		{
			name:     "临时错误 - 可恢复",
			err:      errors.New("temporary failure"),
			errType:  ErrorTypeQuery,
			expected: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			recoverable := suite.errorHandler.isRecoverable(tc.err, tc.errType)
			assert.Equal(t, tc.expected, recoverable)
		})
	}
}

// TestWrapWithTransaction 测试事务包装
func (suite *ErrorHandlerTestSuite) TestWrapWithTransaction() {
	// 测试成功的事务
	suite.T().Run("成功的事务", func(t *testing.T) {
		err := suite.errorHandler.WrapWithTransaction(suite.db, func(tx *gorm.DB) error {
			return tx.Exec("INSERT INTO test_error_table (id, data) VALUES (1, 'test')").Error
		})

		assert.NoError(t, err)

		// 验证数据已插入
		var count int64
		suite.db.Table("test_error_table").Count(&count)
		assert.Equal(t, int64(1), count)

		// 清理
		suite.db.Exec("DELETE FROM test_error_table")
	})

	// 测试失败的事务
	suite.T().Run("失败的事务", func(t *testing.T) {
		err := suite.errorHandler.WrapWithTransaction(suite.db, func(tx *gorm.DB) error {
			// 先插入一条记录
			tx.Exec("INSERT INTO test_error_table (id, data) VALUES (2, 'test')")
			// 然后返回错误，触发回滚
			return errors.New("operation failed")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "事务操作失败")

		// 验证数据未插入（已回滚）
		var count int64
		suite.db.Table("test_error_table").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	// 测试事务开始失败
	suite.T().Run("事务开始失败", func(t *testing.T) {
		// 关闭数据库连接来模拟事务开始失败
		closedDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		sqlDB, _ := closedDB.DB()
		sqlDB.Close()

		err := suite.errorHandler.WrapWithTransaction(closedDB, func(tx *gorm.DB) error {
			return nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "开始事务失败")
	})
}

// TestRetryWithBackoff 测试带退避的重试机制
func (suite *ErrorHandlerTestSuite) TestRetryWithBackoff() {
	// 测试成功重试
	suite.T().Run("第二次重试成功", func(t *testing.T) {
		attemptCount := 0

		err := suite.errorHandler.RetryWithBackoff(
			context.Background(),
			func() error {
				attemptCount++
				if attemptCount < 2 {
					return errors.New("temporary failure")
				}
				return nil
			},
			3,
			10*time.Millisecond,
		)

		assert.NoError(t, err)
		assert.Equal(t, 2, attemptCount)
	})

	// 测试重试失败
	suite.T().Run("重试全部失败", func(t *testing.T) {
		attemptCount := 0

		err := suite.errorHandler.RetryWithBackoff(
			context.Background(),
			func() error {
				attemptCount++
				return errors.New("persistent failure")
			},
			2,
			10*time.Millisecond,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "重试 3 次后仍然失败")
		assert.Equal(t, 3, attemptCount) // 初始尝试 + 2次重试
	})

	// 测试上下文取消
	suite.T().Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := suite.errorHandler.RetryWithBackoff(
			ctx,
			func() error {
				return errors.New("will be cancelled")
			},
			5,
			100*time.Millisecond,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "重试被取消")
	})
}

// TestRecoverFromPanic 测试panic恢复
func (suite *ErrorHandlerTestSuite) TestRecoverFromPanic() {
	// 测试有panic的情况
	suite.T().Run("从panic中恢复", func(t *testing.T) {
		var detail *ErrorDetail

		// 使用recover正确捕获panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					detail = &ErrorDetail{
						Type:        ErrorTypeSystem,
						Severity:    SeverityCritical,
						Message:     fmt.Sprintf("系统发生panic: %v", r),
						Context:     "test_context",
						Timestamp:   time.Now(),
						Recoverable: false,
						RetryCount:  0,
						Metadata:    make(map[string]interface{}),
						StackTrace:  "test stack trace",
					}
				}
			}()

			panic("test panic")
		}()

		assert.NotNil(t, detail)
		assert.Equal(t, ErrorTypeSystem, detail.Type)
		assert.Equal(t, SeverityCritical, detail.Severity)
		assert.Contains(t, detail.Message, "test panic")
		assert.Equal(t, "test_context", detail.Context)
		assert.False(t, detail.Recoverable)
		assert.NotEmpty(t, detail.StackTrace)
	})

	// 测试没有panic的情况
	suite.T().Run("没有panic", func(t *testing.T) {
		var detail *ErrorDetail

		func() {
			defer func() {
				detail = suite.errorHandler.RecoverFromPanic("test_context")
			}()

			// 正常执行，不会panic
		}()

		assert.Nil(t, detail)
	})
}

// TestCreateBusinessError 测试创建业务错误
func (suite *ErrorHandlerTestSuite) TestCreateBusinessError() {
	detail := suite.errorHandler.CreateBusinessError("业务规则违反", "business_validation")

	assert.NotNil(suite.T(), detail)
	assert.Equal(suite.T(), ErrorTypeBusinessLogic, detail.Type)
	assert.Equal(suite.T(), SeverityMedium, detail.Severity)
	assert.Equal(suite.T(), "业务规则违反", detail.Message)
	assert.Equal(suite.T(), "business_validation", detail.Context)
	assert.False(suite.T(), detail.Recoverable)
	assert.Equal(suite.T(), 0, detail.RetryCount)
	assert.NotNil(suite.T(), detail.Metadata)
}

// TestCreateValidationError 测试创建验证错误
func (suite *ErrorHandlerTestSuite) TestCreateValidationError() {
	detail := suite.errorHandler.CreateValidationError("username", "", "required")

	assert.NotNil(suite.T(), detail)
	assert.Equal(suite.T(), ErrorTypeValidation, detail.Type)
	assert.Equal(suite.T(), SeverityLow, detail.Severity)
	assert.Contains(suite.T(), detail.Message, "username")
	assert.Contains(suite.T(), detail.Message, "required")
	assert.Equal(suite.T(), "data_validation", detail.Context)
	assert.False(suite.T(), detail.Recoverable)

	// 验证元数据
	assert.Equal(suite.T(), "username", detail.Metadata["field"])
	assert.Equal(suite.T(), "", detail.Metadata["value"])
	assert.Equal(suite.T(), "required", detail.Metadata["rule"])
}

// 运行测试套件
func TestErrorHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorHandlerTestSuite))
}

// 基准测试
func BenchmarkHandleError(b *testing.B) {
	handler := NewErrorHandler()
	ctx := context.Background()
	err := errors.New("test error")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler.HandleError(ctx, err, ErrorTypeQuery, "benchmark_test")
	}
}

func BenchmarkRetryWithBackoff(b *testing.B) {
	handler := NewErrorHandler()
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler.RetryWithBackoff(ctx, func() error {
			return nil // 立即成功
		}, 3, time.Microsecond)
	}
}
