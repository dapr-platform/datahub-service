/*
 * @module service/interface_executor/error_handler
 * @description 统一错误处理和事务管理工具
 * @architecture 责任链模式 - 提供分层的错误处理和恢复机制
 * @documentReference design.md
 * @stateFlow 错误捕获 -> 错误分类 -> 恢复策略 -> 事务回滚 -> 日志记录
 * @rules 确保所有错误都有明确的处理策略，事务操作具有ACID特性
 * @dependencies gorm.io/gorm, context, time
 * @refs executor.go, data_sync_engine.go
 */

package interface_executor

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ErrorType 错误类型
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "validation"  // 数据验证错误
	ErrorTypeConnection    ErrorType = "connection"  // 连接错误
	ErrorTypeTimeout       ErrorType = "timeout"     // 超时错误
	ErrorTypeTransaction   ErrorType = "transaction" // 事务错误
	ErrorTypeDataSource    ErrorType = "datasource"  // 数据源错误
	ErrorTypeQuery         ErrorType = "query"       // 查询错误
	ErrorTypeSync          ErrorType = "sync"        // 同步错误
	ErrorTypeSystem        ErrorType = "system"      // 系统错误
	ErrorTypeBusinessLogic ErrorType = "business"    // 业务逻辑错误
)

// ErrorSeverity 错误严重级别
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"      // 低级别：警告级别
	SeverityMedium   ErrorSeverity = "medium"   // 中级别：需要注意
	SeverityHigh     ErrorSeverity = "high"     // 高级别：需要立即处理
	SeverityCritical ErrorSeverity = "critical" // 严重级别：系统级错误
)

// ErrorDetail 详细错误信息
type ErrorDetail struct {
	Type        ErrorType              `json:"type"`
	Severity    ErrorSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	Cause       error                  `json:"cause,omitempty"`
	Context     string                 `json:"context"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Recoverable bool                   `json:"recoverable"`
	RetryCount  int                    `json:"retry_count"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger *log.Logger
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		logger: log.Default(),
	}
}

// HandleError 处理错误
func (h *ErrorHandler) HandleError(ctx context.Context, err error, errorType ErrorType, context string) *ErrorDetail {
	if err == nil {
		return nil
	}

	detail := &ErrorDetail{
		Type:       errorType,
		Message:    err.Error(),
		Cause:      err,
		Context:    context,
		Timestamp:  time.Now(),
		RetryCount: 0,
		Metadata:   make(map[string]interface{}),
	}

	// 根据错误类型设置严重级别
	detail.Severity = h.determineSeverity(err, errorType)
	detail.Recoverable = h.isRecoverable(err, errorType)

	// 如果是高级别错误，记录堆栈跟踪
	if detail.Severity == SeverityHigh || detail.Severity == SeverityCritical {
		detail.StackTrace = h.getStackTrace()
	}

	// 记录错误日志
	h.logError(detail)

	return detail
}

// WrapWithTransaction 包装事务操作
func (h *ErrorHandler) WrapWithTransaction(db *gorm.DB, operation func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			h.logger.Printf("事务执行时发生panic，已回滚: %v", r)
			panic(r)
		}
	}()

	if err := operation(tx); err != nil {
		if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
			h.logger.Printf("事务回滚失败: %v", rollbackErr)
			return fmt.Errorf("操作失败且回滚失败: 操作错误=%w, 回滚错误=%v", err, rollbackErr)
		}
		return fmt.Errorf("事务操作失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// RetryWithBackoff 带退避的重试机制
func (h *ErrorHandler) RetryWithBackoff(ctx context.Context, operation func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return fmt.Errorf("重试被取消: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		if err := operation(); err != nil {
			lastErr = err
			h.logger.Printf("重试第 %d/%d 次失败: %v", attempt+1, maxRetries+1, err)
			continue
		}

		return nil
	}

	return fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries+1, lastErr)
}

// RecoverFromPanic 从panic中恢复
func (h *ErrorHandler) RecoverFromPanic(context string) *ErrorDetail {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic: %v", r)
		detail := &ErrorDetail{
			Type:        ErrorTypeSystem,
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("系统发生panic: %v", r),
			Cause:       err,
			Context:     context,
			StackTrace:  h.getStackTrace(),
			Timestamp:   time.Now(),
			Recoverable: false,
			RetryCount:  0,
			Metadata:    make(map[string]interface{}),
		}

		h.logError(detail)
		return detail
	}
	return nil
}

// determineSeverity 确定错误严重级别
func (h *ErrorHandler) determineSeverity(err error, errorType ErrorType) ErrorSeverity {
	errMsg := strings.ToLower(err.Error())

	// 系统级错误
	if errorType == ErrorTypeSystem {
		return SeverityCritical
	}

	// 事务错误通常是高级别
	if errorType == ErrorTypeTransaction {
		return SeverityHigh
	}

	// 连接和超时错误
	if errorType == ErrorTypeConnection || errorType == ErrorTypeTimeout {
		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "connection refused") {
			return SeverityHigh
		}
		return SeverityMedium
	}

	// 数据验证错误通常是低级别
	if errorType == ErrorTypeValidation {
		return SeverityLow
	}

	// 其他情况根据关键词判断
	if strings.Contains(errMsg, "fatal") || strings.Contains(errMsg, "critical") {
		return SeverityCritical
	}

	if strings.Contains(errMsg, "error") || strings.Contains(errMsg, "failed") {
		return SeverityMedium
	}

	return SeverityLow
}

// isRecoverable 判断错误是否可恢复
func (h *ErrorHandler) isRecoverable(err error, errorType ErrorType) bool {
	errMsg := strings.ToLower(err.Error())

	// 系统级错误通常不可恢复
	if errorType == ErrorTypeSystem {
		return false
	}

	// 数据验证错误不可恢复
	if errorType == ErrorTypeValidation {
		return false
	}

	// 连接错误通常可恢复
	if errorType == ErrorTypeConnection || errorType == ErrorTypeTimeout {
		return true
	}

	// 根据关键词判断
	unrecoverableKeywords := []string{"invalid", "not found", "permission denied", "unauthorized"}
	for _, keyword := range unrecoverableKeywords {
		if strings.Contains(errMsg, keyword) {
			return false
		}
	}

	recoverableKeywords := []string{"timeout", "connection", "temporary", "retry"}
	for _, keyword := range recoverableKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return true // 默认认为可恢复
}

// getStackTrace 获取堆栈跟踪
func (h *ErrorHandler) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// logError 记录错误日志
func (h *ErrorHandler) logError(detail *ErrorDetail) {
	logLevel := "INFO"
	switch detail.Severity {
	case SeverityLow:
		logLevel = "WARN"
	case SeverityMedium:
		logLevel = "ERROR"
	case SeverityHigh:
		logLevel = "ERROR"
	case SeverityCritical:
		logLevel = "FATAL"
	}

	h.logger.Printf("[%s] %s - %s: %s (Context: %s, Recoverable: %v)",
		logLevel,
		detail.Type,
		detail.Severity,
		detail.Message,
		detail.Context,
		detail.Recoverable,
	)

	if detail.StackTrace != "" {
		h.logger.Printf("Stack Trace:\n%s", detail.StackTrace)
	}
}

// CreateBusinessError 创建业务逻辑错误
func (h *ErrorHandler) CreateBusinessError(message string, context string) *ErrorDetail {
	return &ErrorDetail{
		Type:        ErrorTypeBusinessLogic,
		Severity:    SeverityMedium,
		Message:     message,
		Context:     context,
		Timestamp:   time.Now(),
		Recoverable: false,
		RetryCount:  0,
		Metadata:    make(map[string]interface{}),
	}
}

// CreateValidationError 创建数据验证错误
func (h *ErrorHandler) CreateValidationError(field string, value interface{}, rule string) *ErrorDetail {
	message := fmt.Sprintf("字段 '%s' 的值 '%v' 不符合验证规则 '%s'", field, value, rule)
	return &ErrorDetail{
		Type:        ErrorTypeValidation,
		Severity:    SeverityLow,
		Message:     message,
		Context:     "data_validation",
		Timestamp:   time.Now(),
		Recoverable: false,
		RetryCount:  0,
		Metadata: map[string]interface{}{
			"field": field,
			"value": value,
			"rule":  rule,
		},
	}
}
