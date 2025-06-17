package scheduler

import (
	"gorm.io/gorm"
)

// RetryManager 重试管理器
type RetryManager struct {
	db *gorm.DB
}

// NewRetryManager 创建重试管理器实例
func NewRetryManager(db *gorm.DB) *RetryManager {
	return &RetryManager{
		db: db,
	}
}

// ShouldRetry 判断是否需要重试
func (r *RetryManager) ShouldRetry(taskID string, err error) bool {
	// TODO: 实现重试逻辑
	return false
}

// GetRetryCount 获取重试次数
func (r *RetryManager) GetRetryCount(taskID string) int {
	// TODO: 实现获取重试次数逻辑
	return 0
}

// ScheduleRetry 安排重试
func (r *RetryManager) ScheduleRetry(taskID string, task *ScheduleTask) {
	// TODO: 实现重试安排逻辑
}

// ClearRetryCount 清除重试次数
func (r *RetryManager) ClearRetryCount(taskID string) {
	// TODO: 实现清除重试次数逻辑
}
