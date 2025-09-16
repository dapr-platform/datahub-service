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
	"datahub-service/service/basic_library/basic_sync"
	"datahub-service/service/database"
	"datahub-service/service/event"
	"datahub-service/service/governance"
	"datahub-service/service/scheduler"
	"datahub-service/service/sharing"
	"datahub-service/service/thematic_library"
	"fmt"
	"log"
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
	GlobalSyncEngine             *basic_sync.SyncEngine
	GlobalSchemaService          *database.SchemaService
	GlobalSyncTaskService        *basic_library.SyncTaskService
	GlobalSchedulerService       *scheduler.SchedulerService
	GlobalGovernanceService      *governance.GovernanceService
	GlobalSharingService         *sharing.SharingService
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

	log.Println("数据库连接成功")
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
	log.Println("开始运行数据库迁移...")

	if err := database.AutoMigrate(DB); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库表结构迁移完成")

	if err := database.InitializeData(DB); err != nil {
		log.Fatalf("基础数据初始化失败: %v", err)
	}
	log.Println("基础数据初始化完成")

	if err := database.AutoMigrateView(DB); err != nil {
		log.Fatalf("视图迁移失败: %v", err)
	}
	log.Println("视图迁移完成")

	log.Println("所有数据库迁移任务完成")
}

// initServices 初始化服务
func initServices() {
	// 初始化事件服务
	GlobalEventService = event.NewEventService(DB)
	// 将事件服务作为参数传递给BasicLibraryService
	GlobalBasicLibraryService = basic_library.NewService(DB, GlobalEventService)
	GlobalThematicLibraryService = thematic_library.NewService(DB)
	GlobalSchemaService = database.NewSchemaService(DB)
	GlobalSyncTaskService = basic_library.NewSyncTaskService(DB, GlobalBasicLibraryService, nil)
	GlobalSyncEngine = basic_sync.NewSyncEngine(DB, 10, GlobalSyncTaskService)
	// 更新SyncTaskService中的SyncEngine引用
	GlobalSyncTaskService.SetSyncEngine(GlobalSyncEngine)
	// 初始化主题同步服务
	// GlobalThematicSyncService = NewThematicSyncService(DB, GlobalBasicLibraryService, GlobalThematicLibraryService)
	// 初始化调度器服务
	GlobalSchedulerService = scheduler.NewSchedulerService(DB, GlobalSyncTaskService, GlobalSyncEngine)
	GlobalGovernanceService = governance.NewGovernanceService(DB)
	GlobalSharingService = sharing.NewSharingService(DB)

	// 初始化数据源
	initializeDataSources()

	// 启动调度器
	if err := GlobalSchedulerService.Start(); err != nil {
		log.Printf("启动调度器服务失败: %v", err)
	}
	log.Println("服务初始化完成")
}

// initializeDataSources 初始化数据源
func initializeDataSources() {
	log.Println("开始初始化数据源...")

	// 获取数据源初始化服务
	datasourceInitService := GlobalBasicLibraryService.GetDatasourceInitService()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 初始化并启动所有数据源（合并操作）
	result, err := datasourceInitService.InitializeAndStartAllDataSources(ctx)
	if err != nil {
		log.Printf("数据源初始化失败: %v", err)
		return
	}

	// 输出初始化结果
	log.Printf("数据源初始化结果: 总计=%d, 成功=%d, 失败=%d, 跳过=%d, 耗时=%dms",
		result.TotalCount, result.SuccessCount, result.FailedCount, result.SkippedCount, result.Duration)

	if result.FailedCount > 0 {
		log.Printf("失败的数据源: %v", result.FailedSources)
	}

	log.Println("数据源初始化和启动流程完成")
}
