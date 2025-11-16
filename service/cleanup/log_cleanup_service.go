/*
 * @module service/cleanup/log_cleanup_service
 * @description 日志清理服务，负责定期清理过期的同步任务执行日志
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 定时触发 -> 读取配置 -> 执行清理 -> 记录结果
 * @rules 确保日志清理不影响系统正常运行
 * @dependencies datahub-service/service/config, gorm.io/gorm, github.com/robfig/cron/v3
 * @refs service/config
 */

package cleanup

import (
	"context"
	"datahub-service/service/config"
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// LogCleanupService 日志清理服务
type LogCleanupService struct {
	db            *gorm.DB
	configService *config.ConfigService
	cron          *cron.Cron
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewLogCleanupService 创建日志清理服务实例
func NewLogCleanupService(db *gorm.DB, configService *config.ConfigService) *LogCleanupService {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &LogCleanupService{
		db:            db,
		configService: configService,
		cron:          cron.New(cron.WithSeconds()),
		ctx:           ctx,
		cancel:        cancel,
		started:       false,
	}
}

// CleanupExpiredLogs 清理所有过期日志
func (s *LogCleanupService) CleanupExpiredLogs(ctx context.Context) error {
	slog.Info("开始清理过期日志")
	startTime := time.Now()

	// 1. 清理基础库同步日志
	basicRetentionDays, err := s.configService.GetBasicSyncLogRetentionDays()
	if err != nil {
		slog.Error("获取基础库日志保留天数失败", "error", err)
		basicRetentionDays = config.DefaultBasicSyncLogRetentionDays
	}

	basicDeleted, err := s.CleanupBasicSyncLogs(ctx, basicRetentionDays)
	if err != nil {
		slog.Error("清理基础库同步日志失败", "error", err)
	} else {
		slog.Info("清理基础库同步日志完成", "deleted_count", basicDeleted, "retention_days", basicRetentionDays)
	}

	// 2. 清理主题库同步日志
	thematicRetentionDays, err := s.configService.GetThematicSyncLogRetentionDays()
	if err != nil {
		slog.Error("获取主题库日志保留天数失败", "error", err)
		thematicRetentionDays = config.DefaultThematicSyncLogRetentionDays
	}

	thematicDeleted, err := s.CleanupThematicSyncLogs(ctx, thematicRetentionDays)
	if err != nil {
		slog.Error("清理主题库同步日志失败", "error", err)
	} else {
		slog.Info("清理主题库同步日志完成", "deleted_count", thematicDeleted, "retention_days", thematicRetentionDays)
	}

	duration := time.Since(startTime)
	slog.Info("日志清理完成", 
		"basic_deleted", basicDeleted, 
		"thematic_deleted", thematicDeleted,
		"total_deleted", basicDeleted+thematicDeleted,
		"duration_ms", duration.Milliseconds())

	return nil
}

// CleanupBasicSyncLogs 清理基础库同步日志
func (s *LogCleanupService) CleanupBasicSyncLogs(ctx context.Context, retentionDays int) (int64, error) {
	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	slog.Debug("清理基础库同步日志", "cutoff_date", cutoffDate.Format("2006-01-02 15:04:05"), "retention_days", retentionDays)

	// 执行删除操作
	result := s.db.Where("created_at < ?", cutoffDate).Delete(&models.SyncTaskExecution{})
	
	if result.Error != nil {
		return 0, fmt.Errorf("删除基础库同步日志失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// CleanupThematicSyncLogs 清理主题库同步日志
func (s *LogCleanupService) CleanupThematicSyncLogs(ctx context.Context, retentionDays int) (int64, error) {
	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	slog.Debug("清理主题库同步日志", "cutoff_date", cutoffDate.Format("2006-01-02 15:04:05"), "retention_days", retentionDays)

	// 执行删除操作
	result := s.db.Where("created_at < ?", cutoffDate).Delete(&models.ThematicSyncExecution{})
	
	if result.Error != nil {
		return 0, fmt.Errorf("删除主题库同步日志失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// StartScheduledCleanup 启动定时清理任务
func (s *LogCleanupService) StartScheduledCleanup() error {
	if s.started {
		return fmt.Errorf("日志清理调度器已经启动")
	}

	slog.Info("启动日志清理调度器")

	// 每天凌晨2点执行清理任务
	// Cron表达式：秒 分 时 日 月 周
	// 0 0 2 * * * 表示每天凌晨2点
	_, err := s.cron.AddFunc("0 0 2 * * *", func() {
		slog.Info("开始执行定时日志清理任务")
		
		if err := s.CleanupExpiredLogs(s.ctx); err != nil {
			slog.Error("定时日志清理任务失败", "error", err)
		}
	})

	if err != nil {
		return fmt.Errorf("添加定时任务失败: %w", err)
	}

	// 启动调度器
	s.cron.Start()
	s.started = true

	slog.Info("日志清理调度器启动成功，将于每天凌晨2点执行清理任务")
	
	// 可选：启动时立即执行一次清理
	go func() {
		slog.Info("执行首次日志清理")
		if err := s.CleanupExpiredLogs(s.ctx); err != nil {
			slog.Error("首次日志清理失败", "error", err)
		}
	}()

	return nil
}

// StopScheduledCleanup 停止定时清理任务
func (s *LogCleanupService) StopScheduledCleanup() {
	if !s.started {
		return
	}

	slog.Info("停止日志清理调度器")
	
	s.cancel()
	
	if s.cron != nil {
		s.cron.Stop()
	}
	
	s.started = false
	
	slog.Info("日志清理调度器已停止")
}














