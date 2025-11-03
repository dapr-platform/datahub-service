/*
 * @module service/init
 * @description 服务初始化模块，负责数据库连接、配置加载等初始化工作
 * @architecture 分层架构 - 服务层
 * @documentReference dev_docs/backend_requirements.md
 * @stateFlow 应用启动时执行初始化流程
 * @rules 确保所有依赖服务正常启动后才提供API服务
 * @dependencies gorm.io/gorm, gorm.io/driver/postgres
 * @refs dev_docs/model.md
 */

package service

import (
	"context"
	"datahub-service/service/basic_library"
	"datahub-service/service/cleanup"
	"datahub-service/service/config"
	"datahub-service/service/database"
	"datahub-service/service/datasource"
	"datahub-service/service/distributed_lock"
	"datahub-service/service/event"
	"datahub-service/service/governance"
	"datahub-service/service/sharing"
	"datahub-service/service/thematic_library"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB                           *gorm.DB
	GlobalEventService           *event.EventService
	GlobalBasicLibraryService    *basic_library.Service
	GlobalThematicLibraryService *thematic_library.Service
	GlobalThematicSyncService    *thematic_library.ThematicSyncService
	GlobalSchemaService          *database.SchemaService
	GlobalSyncTaskService        *basic_library.SyncTaskService // 现在包含调度功能
	GlobalGovernanceService      *governance.GovernanceService
	GlobalSharingService         *sharing.SharingService
	GlobalDistributedLock        *distributed_lock.RedisLock // Redis分布式锁
	GlobalConfigService          *config.ConfigService       // 配置服务
	GlobalLogCleanupService      *cleanup.LogCleanupService  // 日志清理服务
)

func init() {
	initDatabase()
	runMigrations()
	initServices()
}

// initDatabase 初始化数据库连接
func initDatabase() {
	var dsn string

	// 优先使用DATABASE_URL环境变量
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		dsn = databaseURL
	} else {
		// 使用分离的环境变量构建连接字符串
		host := getEnvWithDefault("DB_HOST", "localhost")
		port := getEnvWithDefault("DB_PORT", "5432")
		user := getEnvWithDefault("DB_USER", "postgres")
		password := getEnvWithDefault("DB_PASSWORD", "things2024")
		dbname := getEnvWithDefault("DB_NAME", "postgres")
		sslmode := getEnvWithDefault("DB_SSLMODE", "disable")
		schema := getEnvWithDefault("DB_SCHEMA", "public")

		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=%s TimeZone=Asia/Shanghai",
			host, port, user, password, dbname, sslmode, schema)
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	slog.Info("数据库连接成功")
}

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// runMigrations 运行数据库迁移
func runMigrations() {
	slog.Info("开始运行数据库迁移...")

	if err := database.AutoMigrate(DB); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	slog.Info("数据库表结构迁移完成")

	if err := database.InitializeData(DB); err != nil {
		log.Fatalf("基础数据初始化失败: %v", err)
	}
	slog.Info("基础数据初始化完成")

	if err := database.AutoMigrateView(DB); err != nil {
		log.Fatalf("视图迁移失败: %v", err)
	}
	slog.Info("视图迁移完成")

	slog.Info("所有数据库迁移任务完成")
}

// initServices 初始化服务
func initServices() {
	// 初始化配置服务（优先初始化，其他服务可能需要）
	GlobalConfigService = config.NewConfigService(DB)

	// 初始化事件服务
	GlobalEventService = event.NewEventService(DB)
	// 将事件服务作为参数传递给BasicLibraryService
	GlobalBasicLibraryService = basic_library.NewService(DB, GlobalEventService)
	GlobalThematicLibraryService = thematic_library.NewService(DB)
	GlobalSchemaService = database.NewSchemaService(DB)
	// 初始化同步任务服务（现在集成了调度功能）
	GlobalSyncTaskService = basic_library.NewSyncTaskService(DB, GlobalBasicLibraryService)
	// 初始化数据治理服务
	GlobalGovernanceService = governance.NewGovernanceService(DB)
	// 初始化主题同步服务
	GlobalThematicSyncService = thematic_library.NewThematicSyncService(DB, GlobalGovernanceService)
	GlobalSharingService = sharing.NewSharingService(DB)

	// 初始化全局实时处理器
	initRealtimeProcessor()

	// 初始化Redis分布式锁
	if schedulerEnabled := getEnvWithDefault("SCHEDULER_ENABLED", "true"); schedulerEnabled == "true" {
		lock, err := distributed_lock.NewRedisLock()
		if err != nil {
			slog.Warn("警告: Redis分布式锁初始化失败，调度器将在单实例模式运行", "error", err)
			GlobalDistributedLock = nil
		} else {
			GlobalDistributedLock = lock
			slog.Info("Redis分布式锁初始化成功")

			// 将分布式锁注入到服务中
			GlobalSyncTaskService.SetDistributedLock(lock)
			GlobalThematicSyncService.SetDistributedLock(lock)
		}
	}

	// 初始化数据源
	initializeDataSources()

	// 重置运行中的任务状态（程序重启会中断正在执行的任务）
	resetRunningTasksOnStartup()

	// 启动基础库调度器
	if err := GlobalSyncTaskService.StartScheduler(); err != nil {
		slog.Error("启动基础库同步任务调度器失败", "error", err)
	}

	// 启动主题库调度器
	if err := GlobalThematicSyncService.StartScheduler(); err != nil {
		slog.Error("启动主题库同步任务调度器失败", "error", err)
	}

	// 启动质量检测调度器
	qualityScheduler := GlobalGovernanceService.GetQualityScheduler()
	if qualityScheduler != nil {
		// 设置分布式锁
		if GlobalDistributedLock != nil {
			qualityScheduler.SetDistributedLock(GlobalDistributedLock)
		}

		// 启动调度器
		if err := qualityScheduler.StartScheduler(); err != nil {
			slog.Error("启动数据质量检测调度器失败", "error", err)
		} else {
			slog.Info("数据质量检测调度器启动成功")
		}
	}

	// 初始化并启动日志清理服务
	GlobalLogCleanupService = cleanup.NewLogCleanupService(DB, GlobalConfigService)
	if err := GlobalLogCleanupService.StartScheduledCleanup(); err != nil {
		slog.Error("启动日志清理调度器失败", "error", err)
	} else {
		slog.Info("日志清理调度器启动成功")
	}

	slog.Info("服务初始化完成")
}

// initializeDataSources 初始化数据源
func initializeDataSources() {
	slog.Info("开始初始化数据源...")

	// 获取数据源初始化服务
	datasourceInitService := GlobalBasicLibraryService.GetDatasourceInitService()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 初始化并启动所有数据源（合并操作）
	result, err := datasourceInitService.InitializeAndStartAllDataSources(ctx)
	if err != nil {
		slog.Error("数据源初始化失败", "error", err)
		return
	}

	// 输出初始化结果
	slog.Info("数据源初始化结果",
		"total_count", result.TotalCount,
		"success_count", result.SuccessCount,
		"failed_count", result.FailedCount,
		"skipped_count", result.SkippedCount,
		"duration_ms", result.Duration)

	if result.FailedCount > 0 {
		slog.Warn("存在失败的数据源", "failed_sources", result.FailedSources)
	}

	slog.Info("数据源初始化和启动流程完成")

	// 初始化实时接口绑定
	initializeRealtimeInterfaceBindings(ctx, datasourceInitService)
}

// initRealtimeProcessor 初始化全局实时处理器
func initRealtimeProcessor() {
	slog.Info("开始初始化全局实时处理器...")

	// 创建适配器
	dataWriter := basic_library.NewRealtimeDataWriter(DB)
	interfaceLoader := basic_library.NewRealtimeInterfaceLoader(DB)

	// 初始化全局实时处理器
	datasource.InitGlobalRealtimeProcessor(DB, dataWriter, interfaceLoader)

	slog.Info("全局实时处理器初始化完成")
}

// initializeRealtimeInterfaceBindings 初始化实时接口绑定
func initializeRealtimeInterfaceBindings(ctx context.Context, datasourceInitService *basic_library.DatasourceInitService) {
	slog.Info("开始初始化实时接口绑定...")

	// TODO: 从数据库加载所有实时接口绑定关系并注册到处理器
	// 这里暂时跳过，因为需要在interface_service中实现相关方法

	slog.Info("实时接口绑定初始化完成")
}

// resetRunningTasksOnStartup 在程序启动时重置所有运行中的任务状态
// 因为程序重启会中断正在执行的任务，需要将其标记为失败
func resetRunningTasksOnStartup() {
	slog.Info("开始重置运行中的任务状态...")

	// 重置基础库运行中的任务
	if GlobalSyncTaskService != nil {
		if err := GlobalSyncTaskService.ResetRunningTasksOnStartup(); err != nil {
			slog.Error("重置基础库运行中的任务失败", "error", err)
		}
	}

	// 重置主题库运行中的任务
	if GlobalThematicSyncService != nil {
		if err := GlobalThematicSyncService.ResetRunningTasksOnStartup(); err != nil {
			slog.Error("重置主题库运行中的任务失败", "error", err)
		}
	}

	slog.Info("运行中的任务状态重置完成")
}
