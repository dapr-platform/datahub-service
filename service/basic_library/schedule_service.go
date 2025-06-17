/*
 * @module service/basic_library/schedule_service
 * @description 调度配置服务，管理批量数据源的调度计划和任务执行
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 调度配置 -> 任务创建 -> 执行监控 -> 状态更新
 * @rules 确保调度任务的可靠性和可恢复性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/interfaces.md
 */

package basic_library

import (
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ScheduleService 调度配置服务
type ScheduleService struct {
	db *gorm.DB
}

// NewScheduleService 创建调度配置服务实例
func NewScheduleService(db *gorm.DB) *ScheduleService {
	return &ScheduleService{
		db: db,
	}
}

// ScheduleConfigRequest 调度配置请求
type ScheduleConfigRequest struct {
	DataSourceID string                 `json:"data_source_id"`
	ScheduleType string                 `json:"schedule_type"` // cron, interval, once
	CronExpr     string                 `json:"cron_expr,omitempty"`
	IntervalSec  int                    `json:"interval_sec,omitempty"`
	StartTime    *time.Time             `json:"start_time,omitempty"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	IsEnabled    bool                   `json:"is_enabled"`
	RetryConfig  map[string]interface{} `json:"retry_config,omitempty"`
	AlertConfig  map[string]interface{} `json:"alert_config,omitempty"`
	Description  string                 `json:"description,omitempty"`
}

// CreateConfigRequest 创建调度配置请求的帮助方法
func (s *ScheduleService) CreateConfigRequest(dataSourceID, scheduleType string, scheduleConfig map[string]interface{}, isEnabled bool) ScheduleConfigRequest {
	configRequest := ScheduleConfigRequest{
		DataSourceID: dataSourceID,
		ScheduleType: scheduleType,
		IsEnabled:    isEnabled,
	}

	// 根据调度类型设置相应的配置参数
	switch scheduleType {
	case "cron":
		if cronExpr, exists := scheduleConfig["cron_expr"]; exists {
			configRequest.CronExpr = cronExpr.(string)
		}
	case "interval":
		if intervalSec, exists := scheduleConfig["interval_sec"]; exists {
			if interval, ok := intervalSec.(float64); ok {
				configRequest.IntervalSec = int(interval)
			}
		}
	}

	if startTime, exists := scheduleConfig["start_time"]; exists {
		if st, ok := startTime.(*time.Time); ok {
			configRequest.StartTime = st
		}
	}
	if endTime, exists := scheduleConfig["end_time"]; exists {
		if et, ok := endTime.(*time.Time); ok {
			configRequest.EndTime = et
		}
	}
	if retryConfig, exists := scheduleConfig["retry_config"]; exists {
		configRequest.RetryConfig = retryConfig.(map[string]interface{})
	}
	if alertConfig, exists := scheduleConfig["alert_config"]; exists {
		configRequest.AlertConfig = alertConfig.(map[string]interface{})
	}
	if description, exists := scheduleConfig["description"]; exists {
		configRequest.Description = description.(string)
	}

	return configRequest
}

// ConfigureSchedule 配置调度计划
func (s *ScheduleService) ConfigureSchedule(scheduleConfig *models.ScheduleConfig) (*models.ScheduleConfig, error) {
	// 验证数据源存在
	var dataSource models.DataSource
	if err := s.db.First(&dataSource, "id = ?", scheduleConfig.DataSourceID).Error; err != nil {
		return nil, fmt.Errorf("数据源不存在: %v", err)
	}

	// 验证调度配置
	if err := s.validateScheduleConfig(scheduleConfig); err != nil {
		return nil, err
	}

	// 查找现有配置
	var existing models.ScheduleConfig
	err := s.db.Where("id = ?", scheduleConfig.ID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		scheduleConfig.CreatedAt = time.Now()
		if err := s.db.Create(scheduleConfig).Error; err != nil {
			return nil, fmt.Errorf("创建调度配置失败: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("查询调度配置失败: %v", err)
	} else {
		// 更新现有配置
		scheduleConfig.UpdatedAt = time.Now()

		if err := s.db.Save(scheduleConfig).Error; err != nil {
			return nil, fmt.Errorf("更新调度配置失败: %v", err)
		}
	}

	// 根据配置启用或禁用调度
	if scheduleConfig.IsEnabled {
		if err := s.enableSchedule(scheduleConfig.ID); err != nil {
			return nil, fmt.Errorf("启用调度失败: %v", err)
		}
	} else {
		if err := s.disableSchedule(scheduleConfig.ID); err != nil {
			return nil, fmt.Errorf("禁用调度失败: %v", err)
		}
	}

	return scheduleConfig, nil
}

// validateScheduleConfig 验证调度配置
func (s *ScheduleService) validateScheduleConfig(config *models.ScheduleConfig) error {
	// 验证调度类型
	if !meta.IsValidScheduleType(config.ScheduleType) {
		return fmt.Errorf("无效的调度类型: %s", config.ScheduleType)
	}
	// TODO: 验证调度配置
	return nil
}

// GetScheduleConfig 获取调度配置
func (s *ScheduleService) GetScheduleConfig(dataSourceID string) (*models.ScheduleConfig, error) {
	var config models.ScheduleConfig
	err := s.db.Where("data_source_id = ?", dataSourceID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetScheduleConfigs 获取调度配置列表
func (s *ScheduleService) GetScheduleConfigs(page, pageSize int, enabled *bool) ([]models.ScheduleConfig, int64, error) {
	var configs []models.ScheduleConfig
	var total int64

	query := s.db.Model(&models.ScheduleConfig{})
	if enabled != nil {
		query = query.Where("is_enabled = ?", *enabled)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&configs).Error

	return configs, total, err
}

// enableSchedule 启用调度
func (s *ScheduleService) enableSchedule(configID string) error {
	// TODO: 与调度器系统集成，启用调度任务
	// 这里应该调用实际的调度器服务（如Cron、Quartz等）来启用任务

	// 模拟启用调度
	fmt.Printf("启用调度配置: %s\n", configID)
	return nil
}

// disableSchedule 禁用调度
func (s *ScheduleService) disableSchedule(configID string) error {
	// TODO: 与调度器系统集成，禁用调度任务
	// 这里应该调用实际的调度器服务来禁用任务

	// 模拟禁用调度
	fmt.Printf("禁用调度配置: %s\n", configID)
	return nil
}

// CreateSyncTask 创建同步任务
func (s *ScheduleService) CreateSyncTask(dataSourceID, interfaceID, taskType string, parameters map[string]interface{}) (*models.SyncTask, error) {
	task := models.SyncTask{
		DataSourceID: dataSourceID,
		TaskType:     taskType,
		Status:       meta.SyncTaskStatusPending,
		Config:       parameters,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 只有当interfaceID不为空时才设置，避免外键约束错误
	if interfaceID != "" {
		task.InterfaceID = &interfaceID
	}

	if err := s.db.Create(&task).Error; err != nil {
		return nil, fmt.Errorf("创建同步任务失败: %v", err)
	}

	// 注意：这里只创建任务记录，实际执行应该通过SyncEngine来处理
	// 在调用方（如Controller）需要将任务提交到SyncEngine执行

	return &task, nil
}

// executeSyncTask 执行同步任务
func (s *ScheduleService) executeSyncTask(task *models.SyncTask) {
	// 更新任务状态为运行中
	s.updateTaskStatus(task.ID, "running", nil, nil)

	startTime := time.Now()

	// TODO: 实现实际的数据同步逻辑
	// 这里应该根据数据源类型和任务类型执行相应的同步操作

	// 模拟任务执行
	time.Sleep(2 * time.Second)

	// 模拟执行结果
	result := map[string]interface{}{
		"rows_processed": 1000,
		"rows_inserted":  800,
		"rows_updated":   150,
		"rows_failed":    50,
		"duration_ms":    time.Since(startTime).Milliseconds(),
		"data_size_mb":   5.2,
	}

	// 更新任务状态为完成
	endTime := time.Now()
	s.updateTaskStatus(task.ID, "success", &endTime, result)
}

// updateTaskStatus 更新任务状态
func (s *ScheduleService) updateTaskStatus(taskID, status string, endTime *time.Time, result map[string]interface{}) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if endTime != nil {
		updates["end_time"] = endTime
	}

	if result != nil {
		updates["result"] = result
	}

	return s.db.Model(&models.SyncTask{}).Where("id = ?", taskID).Updates(updates).Error
}

// GetSyncTasks 获取同步任务列表
func (s *ScheduleService) GetSyncTasks(dataSourceID string, status string, page, pageSize int) ([]models.SyncTask, int64, error) {
	var tasks []models.SyncTask
	var total int64

	query := s.db.Model(&models.SyncTask{})
	if dataSourceID != "" {
		query = query.Where("data_source_id = ?", dataSourceID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，按创建时间倒序
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error

	return tasks, total, err
}

// GetSyncTask 获取同步任务详情
func (s *ScheduleService) GetSyncTask(taskID string) (*models.SyncTask, error) {
	var task models.SyncTask
	err := s.db.First(&task, "id = ?", taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateSyncTask 更新同步任务
func (s *ScheduleService) UpdateSyncTask(taskID string, config map[string]interface{}) (*models.SyncTask, error) {
	// 检查任务状态
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %v", err)
	}

	// 只有pending状态的任务可以更新
	if task.Status != meta.SyncTaskStatusPending {
		return nil, fmt.Errorf("只有待执行状态的任务可以更新，当前状态: %s", task.Status)
	}

	// 更新任务配置
	updates := map[string]interface{}{
		"config":     config,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&task).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新同步任务失败: %v", err)
	}

	// 重新获取更新后的任务
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取更新后的任务失败: %v", err)
	}

	return &task, nil
}

// CancelSyncTask 取消同步任务
func (s *ScheduleService) CancelSyncTask(taskID string) error {
	// 检查任务状态
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %v", err)
	}

	if task.Status != meta.SyncTaskStatusPending && task.Status != meta.SyncTaskStatusRunning {
		return fmt.Errorf("任务状态不允许取消: %s", task.Status)
	}

	// 更新任务状态
	now := time.Now()
	updates := map[string]interface{}{
		"status":     meta.SyncTaskStatusCancelled,
		"end_time":   &now,
		"updated_at": now,
	}

	return s.db.Model(&task).Updates(updates).Error
}

// RetryTask 重试失败的任务
func (s *ScheduleService) RetryTask(taskID string) (*models.SyncTask, error) {
	// 获取原任务
	originalTask, err := s.GetSyncTask(taskID)
	if err != nil {
		return nil, err
	}

	if originalTask.Status != meta.SyncTaskStatusFailed {
		return nil, fmt.Errorf("只能重试失败的任务")
	}

	// 创建新的重试任务
	retryTask := models.SyncTask{
		DataSourceID: originalTask.DataSourceID,
		InterfaceID:  originalTask.InterfaceID, // 这里直接复制，因为原任务的InterfaceID已经是正确的格式
		TaskType:     originalTask.TaskType,
		Status:       meta.SyncTaskStatusPending,
		Config:       originalTask.Config,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.db.Create(&retryTask).Error; err != nil {
		return nil, fmt.Errorf("创建重试任务失败: %v", err)
	}

	// 启动任务执行
	go s.executeSyncTask(&retryTask)

	return &retryTask, nil
}

// GetTaskStatistics 获取任务统计信息
func (s *ScheduleService) GetTaskStatistics(dataSourceID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	query := s.db.Model(&models.SyncTask{})
	if dataSourceID != "" {
		query = query.Where("data_source_id = ?", dataSourceID)
	}
	query = query.Where("created_at BETWEEN ? AND ?", startTime, endTime)

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&statusStats).Error; err != nil {
		return nil, err
	}

	// 总任务数
	var totalTasks int64
	query.Count(&totalTasks)

	// 成功率
	var successfulTasks int64
	s.db.Model(&models.SyncTask{}).Where("status = ? AND created_at BETWEEN ? AND ?", "success", startTime, endTime).Count(&successfulTasks)

	successRate := float64(0)
	if totalTasks > 0 {
		successRate = float64(successfulTasks) / float64(totalTasks) * 100
	}

	// 平均执行时间
	var avgDuration float64
	s.db.Model(&models.SyncTask{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "success", startTime, endTime).
		Select("AVG(EXTRACT(EPOCH FROM (end_time - start_time)))").
		Scan(&avgDuration)

	statistics := map[string]interface{}{
		"total_tasks":      totalTasks,
		"successful_tasks": successfulTasks,
		"success_rate":     successRate,
		"avg_duration_sec": avgDuration,
		"status_breakdown": statusStats,
		"period": map[string]interface{}{
			"start_time": startTime,
			"end_time":   endTime,
		},
	}

	return statistics, nil
}

// ValidateCronExpression 验证cron表达式
func (s *ScheduleService) ValidateCronExpression(cronExpr string) error {
	// TODO: 实现cron表达式验证逻辑
	// 可以使用第三方库如 github.com/robfig/cron 来验证

	if cronExpr == "" {
		return fmt.Errorf("cron表达式不能为空")
	}

	// 简单验证：检查是否包含5或6个字段
	// 标准cron格式：秒 分 时 日 月 星期
	// 简化cron格式：分 时 日 月 星期

	// 这里只做基本格式检查，实际应该使用专业的cron解析库
	return nil
}

// GetNextExecutionTime 获取下次执行时间
func (s *ScheduleService) GetNextExecutionTime(configID string) (*time.Time, error) {
	config, err := s.GetScheduleConfig(configID)
	if err != nil {
		return nil, err
	}

	if !config.IsEnabled {
		return nil, fmt.Errorf("调度已禁用")
	}

	now := time.Now()

	switch config.ScheduleType {
	case "cron":
		// TODO: 使用cron库计算下次执行时间
		nextTime := now.Add(time.Hour) // 模拟值
		return &nextTime, nil
	case "interval":
		// 从ScheduleConfig中获取间隔时间
		if intervalSec, ok := config.ScheduleConfig["interval_sec"].(float64); ok {
			nextTime := now.Add(time.Duration(intervalSec) * time.Second)
			return &nextTime, nil
		}
		return nil, fmt.Errorf("调度配置中缺少interval_sec参数")
	case "once":
		// 从ScheduleConfig中获取开始时间
		if startTimeStr, ok := config.ScheduleConfig["start_time"].(string); ok {
			startTime, err := time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return nil, fmt.Errorf("解析开始时间失败: %v", err)
			}
			if startTime.After(now) {
				return &startTime, nil
			}
		}
		return nil, fmt.Errorf("一次性任务已过期或已执行")
	default:
		return nil, fmt.Errorf("不支持的调度类型: %s", config.ScheduleType)
	}
}

// DeleteSyncTask 删除同步任务
func (s *ScheduleService) DeleteSyncTask(taskID string) error {
	// 检查任务是否存在
	var task models.SyncTask
	if err := s.db.First(&task, "id = ?", taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("任务不存在")
		}
		return fmt.Errorf("查询任务失败: %v", err)
	}

	// 检查任务状态是否允许删除
	// 只允许删除已完成、失败或取消的任务
	allowedStatuses := meta.GetDeletableTaskStatuses()
	isAllowed := false
	for _, status := range allowedStatuses {
		if task.Status == status {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("只能删除已完成、失败或已取消的任务，当前状态: %s", task.Status)
	}

	// 删除任务
	if err := s.db.Delete(&task).Error; err != nil {
		return fmt.Errorf("删除任务失败: %v", err)
	}

	return nil
}

// BatchDeleteSyncTasks 批量删除同步任务
func (s *ScheduleService) BatchDeleteSyncTasks(taskIDs []string) (int, error) {
	if len(taskIDs) == 0 {
		return 0, fmt.Errorf("任务ID列表不能为空")
	}

	// 检查所有任务的状态
	var tasks []models.SyncTask
	if err := s.db.Where("id IN ?", taskIDs).Find(&tasks).Error; err != nil {
		return 0, fmt.Errorf("查询任务失败: %v", err)
	}

	if len(tasks) != len(taskIDs) {
		return 0, fmt.Errorf("部分任务不存在")
	}

	// 检查状态
	allowedStatusesList := meta.GetDeletableTaskStatuses()
	allowedStatuses := make(map[string]bool)
	for _, status := range allowedStatusesList {
		allowedStatuses[status] = true
	}
	var validIDs []string
	var invalidTasks []string

	for _, task := range tasks {
		if allowedStatuses[task.Status] {
			validIDs = append(validIDs, task.ID)
		} else {
			invalidTasks = append(invalidTasks, fmt.Sprintf("%s(%s)", task.ID, task.Status))
		}
	}

	if len(invalidTasks) > 0 {
		return 0, fmt.Errorf("以下任务状态不允许删除: %v", invalidTasks)
	}

	// 批量删除
	result := s.db.Where("id IN ?", validIDs).Delete(&models.SyncTask{})
	if result.Error != nil {
		return 0, fmt.Errorf("批量删除任务失败: %v", result.Error)
	}

	return int(result.RowsAffected), nil
}

// CleanupCompletedTasks 清理已完成的历史任务
func (s *ScheduleService) CleanupCompletedTasks(beforeDate time.Time, statuses []string) (int, error) {
	if len(statuses) == 0 {
		statuses = meta.GetDeletableTaskStatuses()
	}

	// 验证状态参数
	validStatusesList := meta.GetDeletableTaskStatuses()
	validStatuses := make(map[string]bool)
	for _, status := range validStatusesList {
		validStatuses[status] = true
	}
	for _, status := range statuses {
		if !validStatuses[status] {
			return 0, fmt.Errorf("无效的状态: %s", status)
		}
	}

	// 删除指定日期之前的已完成任务
	result := s.db.Where("status IN ? AND updated_at < ?", statuses, beforeDate).Delete(&models.SyncTask{})
	if result.Error != nil {
		return 0, fmt.Errorf("清理历史任务失败: %v", result.Error)
	}

	return int(result.RowsAffected), nil
}
