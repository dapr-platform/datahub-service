/*
 * @module service/governance/quality_scheduler
 * @description 数据质量检测任务调度器，负责任务的定时调度和执行
 * @architecture 分层架构 - 服务层
 * @documentReference ai_docs/data_governance_task_req.md
 * @stateFlow 启动调度器 -> 加载任务 -> 定时检查 -> 触发执行
 * @rules 支持cron、interval、once、manual四种调度类型，支持分布式锁
 * @dependencies github.com/robfig/cron/v3, service/distributed_lock
 * @refs quality_task_service.go, sync_task_service.go
 */

package governance

import (
	"context"
	"datahub-service/service/distributed_lock"
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// QualityScheduler 质量检测任务调度器
type QualityScheduler struct {
	service          *GovernanceService
	cron             *cron.Cron
	intervalTicker   *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
	schedulerStarted bool
	distributedLock  distributed_lock.DistributedLock
}

// NewQualityScheduler 创建质量检测任务调度器
func NewQualityScheduler(service *GovernanceService) *QualityScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	c := cron.New(cron.WithSeconds())

	return &QualityScheduler{
		service:          service,
		cron:             c,
		ctx:              ctx,
		cancel:           cancel,
		schedulerStarted: false,
	}
}

// SetDistributedLock 设置分布式锁
func (qs *QualityScheduler) SetDistributedLock(lock distributed_lock.DistributedLock) {
	qs.distributedLock = lock
	if lock != nil {
		slog.Info("质量检测任务调度器已启用分布式锁")
	}
}

// StartScheduler 启动调度器
func (qs *QualityScheduler) StartScheduler() error {
	if qs.schedulerStarted {
		return fmt.Errorf("调度器已经启动")
	}

	slog.Info("启动数据质量检测任务调度器")

	// 启动cron调度器
	qs.cron.Start()

	// 启动间隔任务检查器（每分钟检查一次）
	qs.intervalTicker = time.NewTicker(1 * time.Minute)
	go qs.runIntervalChecker()

	// 加载现有的调度任务
	if err := qs.loadScheduledTasks(); err != nil {
		slog.Error("加载质量检测调度任务失败", "error", err)
		return err
	}

	qs.schedulerStarted = true
	slog.Info("数据质量检测任务调度器启动完成")
	return nil
}

// StopScheduler 停止调度器
func (qs *QualityScheduler) StopScheduler() {
	if !qs.schedulerStarted {
		return
	}

	slog.Info("停止数据质量检测任务调度器")

	qs.cancel()

	if qs.cron != nil {
		qs.cron.Stop()
	}

	if qs.intervalTicker != nil {
		qs.intervalTicker.Stop()
	}

	qs.schedulerStarted = false
	slog.Info("数据质量检测任务调度器已停止")
}

// loadScheduledTasks 加载调度任务
func (qs *QualityScheduler) loadScheduledTasks() error {
	slog.Info("开始加载质量检测调度任务")

	// 获取所有启用且配置了调度的任务
	var tasks []models.QualityTask
	err := qs.service.db.Where("is_enabled = ? AND schedule_type IN (?, ?, ?)",
		true, "cron", "interval", "once").
		Find(&tasks).Error
	if err != nil {
		slog.Error("获取质量检测调度任务失败", "error", err)
		return fmt.Errorf("获取调度任务失败: %w", err)
	}

	slog.Info("找到质量检测调度任务", "count", len(tasks))

	successCount := 0
	failedCount := 0
	for _, task := range tasks {
		slog.Debug("加载任务", "task_id", task.ID, "schedule_type", task.ScheduleType, "name", task.Name)

		if err := qs.addTaskToScheduler(&task); err != nil {
			slog.Error("添加任务到调度器失败", "task_id", task.ID, "error", err)
			failedCount++
		} else {
			successCount++
		}
	}

	slog.Info("质量检测调度任务加载完成", "total", len(tasks), "success", successCount, "failed", failedCount)
	return nil
}

// addTaskToScheduler 添加任务到调度器
func (qs *QualityScheduler) addTaskToScheduler(task *models.QualityTask) error {
	slog.Info("开始添加质量检测任务到调度器",
		"task_id", task.ID,
		"schedule_type", task.ScheduleType,
		"cron_expression", task.CronExpression,
		"interval_seconds", task.IntervalSeconds)

	switch task.ScheduleType {
	case "cron":
		if task.CronExpression == "" {
			return fmt.Errorf("Cron任务缺少表达式")
		}

		taskID := task.ID
		_, err := qs.cron.AddFunc(task.CronExpression, func() {
			qs.executeScheduledTask(taskID)
		})
		if err != nil {
			slog.Error("添加Cron任务失败",
				"task_id", task.ID,
				"cron_expression", task.CronExpression,
				"error", err,
				"help", "Cron表达式需要6个字段（秒 分 时 日 月 周），例如：0 */5 * * * *（每5分钟）")
			return fmt.Errorf("添加Cron任务失败: %w", err)
		}

		slog.Info("添加Cron任务成功", "task_id", task.ID, "cron_expression", task.CronExpression)

	case "once":
		if task.ScheduledTime != nil && task.ScheduledTime.After(time.Now()) {
			taskID := task.ID
			scheduledTime := *task.ScheduledTime
			waitDuration := time.Until(scheduledTime)

			go func() {
				timer := time.NewTimer(waitDuration)
				defer timer.Stop()

				slog.Info("单次任务等待执行",
					"task_id", taskID,
					"scheduled_time", scheduledTime.Format("2006-01-02 15:04:05"),
					"wait_duration", waitDuration)

				select {
				case <-timer.C:
					slog.Info("单次任务时间到，开始执行", "task_id", taskID)
					qs.executeScheduledTask(taskID)
				case <-qs.ctx.Done():
					slog.Warn("单次任务被取消（调度器关闭）", "task_id", taskID)
					return
				}
			}()

			slog.Info("添加单次任务成功",
				"task_id", task.ID,
				"scheduled_time", task.ScheduledTime.Format("2006-01-02 15:04:05"),
				"wait_duration", waitDuration)
		} else {
			if task.ScheduledTime == nil {
				slog.Warn("单次任务缺少执行时间", "task_id", task.ID)
			} else {
				slog.Warn("单次任务的执行时间已过期",
					"task_id", task.ID,
					"scheduled_time", task.ScheduledTime.Format("2006-01-02 15:04:05"),
					"now", time.Now().Format("2006-01-02 15:04:05"))
			}
		}

	case "interval":
		if task.IntervalSeconds <= 0 {
			slog.Warn("间隔任务的间隔时间无效", "task_id", task.ID, "interval_seconds", task.IntervalSeconds)
			return fmt.Errorf("间隔任务的间隔时间必须大于0")
		}
		slog.Info("添加间隔任务成功", "task_id", task.ID, "interval_seconds", task.IntervalSeconds)
	}

	return nil
}

// runIntervalChecker 运行间隔任务检查器
func (qs *QualityScheduler) runIntervalChecker() {
	for {
		select {
		case <-qs.intervalTicker.C:
			qs.checkIntervalTasks()
		case <-qs.ctx.Done():
			return
		}
	}
}

// checkIntervalTasks 检查间隔任务
func (qs *QualityScheduler) checkIntervalTasks() {
	slog.Debug("开始检查质量检测间隔任务", "timestamp", time.Now().Format("2006-01-02 15:04:05"))

	var tasks []models.QualityTask
	now := time.Now()

	// 查找应该执行的间隔任务
	err := qs.service.db.Where("is_enabled = ? AND schedule_type = ? AND (next_execution IS NULL OR next_execution <= ?)",
		true, "interval", now).
		Find(&tasks).Error
	if err != nil {
		slog.Error("获取质量检测间隔任务失败", "error", err)
		return
	}

	slog.Debug("找到待执行的质量检测任务", "count", len(tasks))

	for _, task := range tasks {
		slog.Info("间隔任务达到执行时间，准备执行",
			"task_id", task.ID,
			"name", task.Name,
			"next_execution", task.NextExecution)
		go qs.executeScheduledTask(task.ID)
	}
}

// executeScheduledTask 执行调度任务（带分布式锁）
func (qs *QualityScheduler) executeScheduledTask(taskID string) {
	slog.Info("执行质量检测调度任务", "task_id", taskID)

	// 如果有分布式锁，使用锁保护执行
	if qs.distributedLock != nil {
		lockKey := fmt.Sprintf("quality_task:%s", taskID)
		lockTTL := 30 * time.Minute // 锁的过期时间

		// 尝试获取锁
		locked, err := qs.distributedLock.TryLock(qs.ctx, lockKey, lockTTL)
		if err != nil {
			slog.Error("获取分布式锁失败", "task_id", taskID, "error", err)
			return
		}

		if !locked {
			slog.Warn("任务正在其他实例执行，跳过", "task_id", taskID)
			return
		}

		// 确保执行完毕后释放锁
		defer func() {
			if unlockErr := qs.distributedLock.Unlock(qs.ctx, lockKey); unlockErr != nil {
				slog.Error("释放分布式锁失败", "task_id", taskID, "error", unlockErr)
			}
		}()
	}

	// 获取任务详情
	var task models.QualityTask
	if err := qs.service.db.First(&task, "id = ?", taskID).Error; err != nil {
		slog.Error("获取质量检测任务失败", "task_id", taskID, "error", err)
		return
	}

	// 检查任务状态
	if task.Status == "running" {
		slog.Warn("任务正在运行中，跳过本次执行", "task_id", taskID)
		return
	}

	if !task.IsEnabled {
		slog.Warn("任务已禁用，跳过执行", "task_id", taskID)
		return
	}

	// 启动任务
	_, err := qs.service.StartQualityTask(taskID)
	if err != nil {
		slog.Error("启动质量检测调度任务失败", "task_id", taskID, "error", err)
		return
	}

	// 更新下次执行时间
	if err := qs.updateTaskNextExecution(taskID); err != nil {
		slog.Error("更新下次执行时间失败", "task_id", taskID, "error", err)
	}

	slog.Info("质量检测调度任务已启动", "task_id", taskID)
}

// updateTaskNextExecution 更新任务的下次执行时间
func (qs *QualityScheduler) updateTaskNextExecution(taskID string) error {
	var task models.QualityTask
	if err := qs.service.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 构建调度配置
	scheduleConfig := ScheduleConfigRequest{
		Type:      task.ScheduleType,
		CronExpr:  task.CronExpression,
		Interval:  task.IntervalSeconds,
		StartTime: task.ScheduledTime,
	}

	// 计算下次执行时间
	now := time.Now()
	nextExec, err := qs.service.CalculateNextExecution(scheduleConfig, &now)
	if err != nil {
		return fmt.Errorf("计算下次执行时间失败: %w", err)
	}

	// 更新数据库
	updates := map[string]interface{}{
		"next_execution": nextExec,
		"last_executed":  time.Now(),
		"updated_at":     time.Now(),
	}

	if err := qs.service.db.Model(&models.QualityTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新任务执行时间失败: %w", err)
	}

	return nil
}

// AddScheduledTask 添加调度任务
func (qs *QualityScheduler) AddScheduledTask(task *models.QualityTask) error {
	return qs.addTaskToScheduler(task)
}

// RemoveScheduledTask 移除调度任务
func (qs *QualityScheduler) RemoveScheduledTask(taskID string) error {
	// 由于cron库不支持按ID移除任务，这里我们重新加载所有任务
	qs.cron.Stop()
	qs.cron = cron.New(cron.WithSeconds())
	qs.cron.Start()

	return qs.loadScheduledTasks()
}

// ReloadScheduledTasks 重新加载调度任务
func (qs *QualityScheduler) ReloadScheduledTasks() error {
	qs.cron.Stop()
	qs.cron = cron.New(cron.WithSeconds())
	qs.cron.Start()

	return qs.loadScheduledTasks()
}


