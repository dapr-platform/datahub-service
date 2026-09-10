/*
 * @module service/authbridge/sync_scheduler
 * @description 用户同步日调度（默认每天 02:00 增量）
 */
package authbridge

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	globalSyncOnce sync.Once
	globalSyncSvc  *UserSyncService
	globalCron     *cron.Cron
)

// GetGlobalUserSyncService 全局同步服务（供控制器复用）
func GetGlobalUserSyncService() *UserSyncService {
	return globalSyncSvc
}

// StartUserSyncScheduler 启动日增量调度；未启用则仅初始化服务实例供手动触发检测
func StartUserSyncScheduler() {
	globalSyncOnce.Do(func() {
		cfg := LoadConfig()
		globalSyncSvc = NewUserSyncService(cfg)
		if !cfg.UserSyncEnabled {
			slog.Info("用户同步未启用，跳过调度器")
			return
		}
		expr := cfg.UserSyncCron
		if expr == "" {
			expr = "0 0 2 * * *"
		}
		globalCron = cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))
		_, err := globalCron.AddFunc(expr, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			slog.Info("开始定时增量用户同步")
			if _, err := globalSyncSvc.Run(ctx, SyncModeIncremental); err != nil {
				slog.Error("定时用户同步失败", "error", err)
			}
		})
		if err != nil {
			slog.Error("注册用户同步 cron 失败", "cron", expr, "error", err)
			return
		}
		globalCron.Start()
		slog.Info("用户同步调度器已启动", "cron", expr)
	})
}
