/**
 * @module SchedulerService
 * @description 同步任务调度器服务，负责定时执行同步任务
 * @architecture 基于Go协程和定时器的调度器模式
 * @documentReference ../ai_docs/sync_task_req.md
 * @stateFlow N/A
 * @rules 支持Cron表达式和间隔调度，自动处理任务状态更新
 * @dependencies gorm, sync_engine, cron库
 * @refs ../models/sync_task.go, ./sync_engine/sync_engine.go
 */

package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"datahub-service/service/basic_library"
	"datahub-service/service/basic_library/basic_sync"
	"datahub-service/service/models"
)

// SchedulerService 调度器服务
type SchedulerService struct {
	db              *gorm.DB
	syncTaskService *basic_library.SyncTaskService
	syncEngine      *basic_sync.SyncEngine
	cron            *cron.Cron
	intervalTicker  *time.Ticker
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewSchedulerService 创建调度器服务
func NewSchedulerService(db *gorm.DB, syncTaskService *basic_library.SyncTaskService, syncEngine *basic_sync.SyncEngine) *SchedulerService {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建带时区的cron调度器
	c := cron.New(cron.WithSeconds())

	scheduler := &SchedulerService{
		db:              db,
		syncTaskService: syncTaskService,
		syncEngine:      syncEngine,
		cron:            c,
		ctx:             ctx,
		cancel:          cancel,
	}

	return scheduler
}

// Start 启动调度器
func (s *SchedulerService) Start() error {
	log.Println("启动同步任务调度器")

	// 启动cron调度器
	s.cron.Start()

	// 启动间隔任务检查器（每分钟检查一次）
	s.intervalTicker = time.NewTicker(1 * time.Minute)
	go s.runIntervalChecker()

	// 加载现有的调度任务
	if err := s.loadScheduledTasks(); err != nil {
		log.Printf("加载调度任务失败: %v", err)
		return err
	}

	log.Println("同步任务调度器启动完成")
	return nil
}

// Stop 停止调度器
func (s *SchedulerService) Stop() {
	log.Println("停止同步任务调度器")

	s.cancel()

	if s.cron != nil {
		s.cron.Stop()
	}

	if s.intervalTicker != nil {
		s.intervalTicker.Stop()
	}

	log.Println("同步任务调度器已停止")
}

// loadScheduledTasks 加载调度任务
func (s *SchedulerService) loadScheduledTasks() error {
	// 获取所有待执行的调度任务
	tasks, err := s.syncTaskService.GetScheduledTasks(s.ctx)
	if err != nil {
		return fmt.Errorf("获取调度任务失败: %w", err)
	}

	for _, task := range tasks {
		if err := s.addTaskToScheduler(&task); err != nil {
			log.Printf("添加任务到调度器失败 [%s]: %v", task.ID, err)
		}
	}

	log.Printf("加载了 %d 个调度任务", len(tasks))
	return nil
}

// addTaskToScheduler 添加任务到调度器
func (s *SchedulerService) addTaskToScheduler(task *models.SyncTask) error {
	switch task.TriggerType {
	case "cron":
		if task.CronExpression == "" {
			return fmt.Errorf("Cron任务缺少表达式")
		}

		_, err := s.cron.AddFunc(task.CronExpression, func() {
			s.executeScheduledTask(task.ID)
		})
		if err != nil {
			return fmt.Errorf("添加Cron任务失败: %w", err)
		}

		log.Printf("添加Cron任务: %s [%s]", task.ID, task.CronExpression)

	case "once":
		if task.ScheduledTime != nil && task.ScheduledTime.After(time.Now()) {
			go func() {
				timer := time.NewTimer(time.Until(*task.ScheduledTime))
				defer timer.Stop()

				select {
				case <-timer.C:
					s.executeScheduledTask(task.ID)
				case <-s.ctx.Done():
					return
				}
			}()

			log.Printf("添加单次任务: %s [%s]", task.ID, task.ScheduledTime.Format("2006-01-02 15:04:05"))
		}

	case "interval":
		// 间隔任务由intervalChecker处理
		log.Printf("添加间隔任务: %s [%d秒]", task.ID, task.IntervalSeconds)
	}

	return nil
}

// runIntervalChecker 运行间隔任务检查器
func (s *SchedulerService) runIntervalChecker() {
	for {
		select {
		case <-s.intervalTicker.C:
			s.checkIntervalTasks()
		case <-s.ctx.Done():
			return
		}
	}
}

// checkIntervalTasks 检查间隔任务
func (s *SchedulerService) checkIntervalTasks() {
	tasks, err := s.syncTaskService.GetScheduledTasks(s.ctx)
	if err != nil {
		log.Printf("获取间隔任务失败: %v", err)
		return
	}

	for _, task := range tasks {
		if task.TriggerType == "interval" && task.ShouldExecuteNow() {
			go s.executeScheduledTask(task.ID)
		}
	}
}

// executeScheduledTask 执行调度任务
func (s *SchedulerService) executeScheduledTask(taskID string) {
	log.Printf("执行调度任务: %s", taskID)

	// 获取任务详情
	task, err := s.syncTaskService.GetSyncTaskByID(s.ctx, taskID)
	if err != nil {
		log.Printf("获取任务失败 [%s]: %v", taskID, err)
		return
	}

	// 检查任务是否可以执行
	if !task.CanStart() {
		log.Printf("任务不能执行 [%s]: 状态=%s", taskID, task.Status)
		return
	}

	// 获取任务关联的接口信息
	var taskInterfaces []models.SyncTaskInterface
	if err := s.db.Where("task_id = ?", task.ID).Find(&taskInterfaces).Error; err != nil {
		log.Printf("获取任务接口关联失败 [%s]: %v", taskID, err)
		return
	}

	var interfaceIDs []string
	for _, taskInterface := range taskInterfaces {
		interfaceIDs = append(interfaceIDs, taskInterface.InterfaceID)
	}

	request := &models.SyncTaskRequest{
		TaskID:       taskID,
		LibraryType:  task.LibraryType,
		LibraryID:    task.LibraryID,
		DataSourceID: task.DataSourceID,
		InterfaceIDs: interfaceIDs,
		SyncType:     models.SyncType(task.TaskType),
		Priority:     1,
		ScheduledBy:  "scheduler",
		IsScheduled:  true,
	}

	// 提交到同步引擎执行
	_, err = s.syncEngine.SubmitSyncTask(request)
	if err != nil {
		log.Printf("提交调度任务失败 [%s]: %v", taskID, err)
		return
	}

	log.Printf("调度任务已提交 [%s]", taskID)
}

// AddScheduledTask 添加调度任务
func (s *SchedulerService) AddScheduledTask(task *models.SyncTask) error {
	return s.addTaskToScheduler(task)
}

// RemoveScheduledTask 移除调度任务
func (s *SchedulerService) RemoveScheduledTask(taskID string) error {
	// 由于cron库不支持按ID移除任务，这里我们重新加载所有任务
	// 在生产环境中，可以考虑使用更高级的调度库
	s.cron.Stop()
	s.cron = cron.New(cron.WithSeconds())
	s.cron.Start()

	return s.loadScheduledTasks()
}

// ReloadScheduledTasks 重新加载调度任务
func (s *SchedulerService) ReloadScheduledTasks() error {
	s.cron.Stop()
	s.cron = cron.New(cron.WithSeconds())
	s.cron.Start()

	return s.loadScheduledTasks()
}
