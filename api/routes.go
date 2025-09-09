/*
 * @module api/routes
 * @description API路由配置模块，负责初始化和配置所有HTTP路由
 * @architecture RESTful API架构
 * @documentReference dev_docs/backend_requirements.md
 * @stateFlow 无状态HTTP请求处理
 * @rules 遵循RESTful API设计规范，统一错误处理和响应格式
 * @dependencies github.com/go-chi/chi/v5, github.com/go-chi/cors, github.com/go-chi/render
 * @refs dev_docs/model.md
 */

package api

import (
	"datahub-service/api/controllers"
	"datahub-service/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

// InitRoute 初始化所有API路由
func InitRoute(r *chi.Mux) {
	// 基础中间件
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// CORS配置
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 健康检查
	healthController := controllers.NewHealthController()
	r.Get("/health", healthController.Health)
	r.Get("/ready", healthController.Ready)

	// SSE事件订阅
	eventController := controllers.NewEventController()
	r.Get("/sse/{user_name}", eventController.HandleSSE)

	// 事件管理
	r.Route("/events", func(r chi.Router) {
		r.Post("/send", eventController.SendEvent)
		r.Post("/broadcast", eventController.BroadcastEvent)
	})

	// 表管理
	r.Route("/tables", func(r chi.Router) {
		tableController := controllers.NewTableController()
		r.Post("/manage-schema", tableController.ManageTableSchema)
	})

	// 元数据管理
	r.Route("/meta", func(r chi.Router) {
		metaController := controllers.NewMetaController()
		r.Get("/basic-libraries/data-sources", metaController.GetDataSourceTypes)
		r.Get("/basic-libraries/data-interface-configs", metaController.GetDataInterfaceConfigs)
		r.Get("/basic-libraries/sync-task-types", metaController.GetSyncTaskTypes)
		r.Get("/basic-libraries/sync-task-meta", metaController.GetSyncTaskRelated)
		r.Get("/thematic-libraries/categories", metaController.GetThematicLibraryCategories)
		r.Get("/thematic-libraries/domains", metaController.GetThematicLibraryDomains)
		r.Get("/thematic-libraries/access-levels", metaController.GetThematicLibraryAccessLevels)
	})

	// 基础库管理（保留现有功能接口）
	r.Route("/basic-libraries", func(r chi.Router) {
		basicLibraryController := controllers.NewBasicLibraryController()

		// 数据源测试
		r.Post("/test-datasource", basicLibraryController.TestDataSource)

		// 接口调用测试
		r.Post("/test-interface", basicLibraryController.TestInterface)

		// 数据源状态查询
		r.Get("/datasource-status/{id}", basicLibraryController.GetDataSourceStatus)

		// 接口数据预览
		r.Get("/interface-preview/{id}", basicLibraryController.PreviewInterfaceData)

		// 添加数据基础库,需要创建schema
		r.Post("/add-basic-library", basicLibraryController.AddBasicLibrary)

		// 修改数据基础库
		r.Post("/update-basic-library", basicLibraryController.UpdateBasicLibrary)

		// 删除数据基础库，需要删除schema
		r.Post("/delete-basic-library", basicLibraryController.DeleteBasicLibrary)

		// 添加数据源
		r.Post("/add-datasource", basicLibraryController.AddDataSource)

		// 删除数据源
		r.Post("/delete-datasource", basicLibraryController.DeleteDataSource)

		// 添加数据接口
		r.Post("/add-interface", basicLibraryController.AddInterface)

		// 删除数据接口
		r.Post("/delete-interface", basicLibraryController.DeleteInterface)
	})

	// 主题库管理
	r.Route("/thematic-libraries", func(r chi.Router) {
		thematicLibraryController := controllers.NewThematicLibraryController()

		// 基础CRUD操作
		r.Post("/", thematicLibraryController.CreateThematicLibrary)
		r.Get("/{id}", thematicLibraryController.GetThematicLibrary)
		r.Put("/{id}", thematicLibraryController.UpdateThematicLibrary)
		r.Delete("/{id}", thematicLibraryController.DeleteThematicLibrary)
	})

	// 通用同步任务管理（统一接口）
	r.Route("/sync", func(r chi.Router) {
		// 使用全局服务初始化控制器
		syncTaskController := controllers.NewSyncTaskController()

		r.Route("/tasks", func(r chi.Router) {
			// 基础CRUD操作
			r.Post("/", syncTaskController.CreateSyncTask)
			r.Get("/", syncTaskController.GetSyncTaskList)
			r.Get("/{id}", syncTaskController.GetSyncTask)
			r.Put("/{id}", syncTaskController.UpdateSyncTask)
			r.Delete("/{id}", syncTaskController.DeleteSyncTask)

			// 任务控制操作
			r.Post("/{id}/start", syncTaskController.StartSyncTask)
			r.Post("/{id}/stop", syncTaskController.StopSyncTask)
			r.Post("/{id}/cancel", syncTaskController.CancelSyncTask)
			r.Post("/{id}/retry", syncTaskController.RetrySyncTask)
			r.Get("/{id}/status", syncTaskController.GetSyncTaskStatus)

			// 批量操作
			r.Post("/batch-delete", syncTaskController.BatchDeleteSyncTasks)

			// 统计信息
			r.Get("/statistics", syncTaskController.GetSyncTaskStatistics)
		})
	})

	// 数据质量管理
	r.Route("/quality", func(r chi.Router) {
		qualityController := controllers.NewQualityController()

		// 质量规则管理
		r.Route("/rules", func(r chi.Router) {
			r.Post("/", qualityController.CreateQualityRule)
			r.Get("/", qualityController.GetQualityRules)
			r.Get("/{id}", qualityController.GetQualityRule)
			r.Put("/{id}", qualityController.UpdateQualityRule)
			r.Delete("/{id}", qualityController.DeleteQualityRule)
		})

		// 质量检查
		r.Route("/checks", func(r chi.Router) {
			r.Post("/", qualityController.ExecuteQualityCheck)
			r.Get("/", qualityController.GetQualityChecks)
			r.Get("/{id}", qualityController.GetQualityCheck)
		})

		// 清洗规则管理
		r.Route("/cleansing", func(r chi.Router) {
			r.Post("/", qualityController.CreateCleansingRule)
			r.Get("/", qualityController.GetCleansingRules)
			r.Post("/{id}/execute", qualityController.ExecuteCleansingRule)
		})

		// 质量报告
		r.Route("/reports", func(r chi.Router) {
			r.Get("/", qualityController.GetQualityReports)
			r.Get("/{id}", qualityController.GetQualityReport)
			r.Post("/generate", qualityController.GenerateQualityReport)
		})

		// 问题追踪
		r.Route("/issues", func(r chi.Router) {
			r.Get("/", qualityController.GetQualityIssues)
			r.Post("/{id}/resolve", qualityController.ResolveQualityIssue)
		})

		// 质量指标
		r.Get("/metrics", qualityController.GetQualityMetrics)
	})

	// 数据治理
	r.Route("/governance", func(r chi.Router) {
		governanceController := controllers.NewGovernanceController(service.NewGovernanceService(service.DB))

		// 数据质量规则管理
		r.Route("/quality-rules", func(r chi.Router) {
			r.Post("/", governanceController.CreateQualityRule)
			r.Get("/", governanceController.GetQualityRules)
			r.Get("/{id}", governanceController.GetQualityRuleByID)
			r.Put("/{id}", governanceController.UpdateQualityRule)
			r.Delete("/{id}", governanceController.DeleteQualityRule)
		})

		// 元数据管理
		r.Route("/metadata", func(r chi.Router) {
			r.Post("/", governanceController.CreateMetadata)
			r.Get("/", governanceController.GetMetadataList)
			r.Get("/{id}", governanceController.GetMetadataByID)
			r.Put("/{id}", governanceController.UpdateMetadata)
			r.Delete("/{id}", governanceController.DeleteMetadata)
		})

		// 数据脱敏规则管理
		r.Route("/masking-rules", func(r chi.Router) {
			r.Post("/", governanceController.CreateMaskingRule)
			r.Get("/", governanceController.GetMaskingRules)
			r.Get("/{id}", governanceController.GetMaskingRuleByID)
			r.Put("/{id}", governanceController.UpdateMaskingRule)
			r.Delete("/{id}", governanceController.DeleteMaskingRule)
		})

		// 系统日志管理
		r.Get("/system-logs", governanceController.GetSystemLogs)

		// 数据质量报告
		r.Route("/quality-reports", func(r chi.Router) {
			r.Get("/", governanceController.GetQualityReports)
			r.Get("/{id}", governanceController.GetQualityReportByID)
		})

		// 数据质量检查
		r.Post("/quality-check", governanceController.RunQualityCheck)
	})

	// 数据共享服务
	r.Route("/sharing", func(r chi.Router) {
		sharingController := controllers.NewSharingController(service.NewSharingService(service.DB))

		// API应用管理
		r.Route("/api-applications", func(r chi.Router) {
			r.Post("/", sharingController.CreateApiApplication)
			r.Get("/", sharingController.GetApiApplications)
			r.Get("/{id}", sharingController.GetApiApplicationByID)
			r.Put("/{id}", sharingController.UpdateApiApplication)
			r.Delete("/{id}", sharingController.DeleteApiApplication)
		})

		// API限流管理
		r.Route("/api-rate-limits", func(r chi.Router) {
			r.Post("/", sharingController.CreateApiRateLimit)
			r.Get("/", sharingController.GetApiRateLimits)
			r.Put("/{id}", sharingController.UpdateApiRateLimit)
			r.Delete("/{id}", sharingController.DeleteApiRateLimit)
		})

		// 数据订阅管理
		r.Route("/data-subscriptions", func(r chi.Router) {
			r.Post("/", sharingController.CreateDataSubscription)
			r.Get("/", sharingController.GetDataSubscriptions)
			r.Get("/{id}", sharingController.GetDataSubscriptionByID)
			r.Put("/{id}", sharingController.UpdateDataSubscription)
			r.Delete("/{id}", sharingController.DeleteDataSubscription)
		})

		// 数据使用申请管理
		r.Route("/data-access-requests", func(r chi.Router) {
			r.Post("/", sharingController.CreateDataAccessRequest)
			r.Get("/", sharingController.GetDataAccessRequests)
			r.Get("/{id}", sharingController.GetDataAccessRequestByID)
			r.Post("/{id}/approve", sharingController.ApproveDataAccessRequest)
		})

		// 数据同步任务管理
		r.Route("/data-sync-tasks", func(r chi.Router) {
			r.Post("/", sharingController.CreateDataSyncTask)
			r.Get("/", sharingController.GetDataSyncTasks)
			r.Get("/{id}", sharingController.GetDataSyncTaskByID)
			r.Put("/{id}", sharingController.UpdateDataSyncTask)
			r.Delete("/{id}", sharingController.DeleteDataSyncTask)
		})

		// 数据同步日志管理
		r.Get("/data-sync-logs", sharingController.GetDataSyncLogs)

		// API使用日志管理
		r.Get("/api-usage-logs", sharingController.GetApiUsageLogs)
	})

	// 监控管理
	r.Route("/monitoring", func(r chi.Router) {
		monitoringController := controllers.NewMonitoringController()

		// 监控指标
		r.Route("/metrics", func(r chi.Router) {
			r.Get("/system", monitoringController.GetSystemMetrics)
			r.Get("/performance", monitoringController.GetPerformanceMetrics)
		})

		// 告警管理
		r.Route("/alerts", func(r chi.Router) {
			r.Get("/", monitoringController.GetAlerts)
			r.Get("/{id}", monitoringController.GetAlert)
			r.Post("/{id}/acknowledge", monitoringController.AcknowledgeAlert)
			r.Post("/{id}/resolve", monitoringController.ResolveAlert)
		})

		// 告警规则
		r.Route("/alert-rules", func(r chi.Router) {
			r.Post("/", monitoringController.CreateAlertRule)
			r.Get("/", monitoringController.GetAlertRules)
			r.Put("/{id}", monitoringController.UpdateAlertRule)
			r.Delete("/{id}", monitoringController.DeleteAlertRule)
		})

		// 健康检查
		r.Route("/health", func(r chi.Router) {
			r.Get("/", monitoringController.GetHealthStatus)
			r.Get("/checks", monitoringController.GetHealthChecks)
		})

		// 服务状态
		r.Get("/services", monitoringController.GetServiceStatus)

		// 监控仪表板
		r.Get("/dashboard", monitoringController.GetMonitoringDashboard)

		// 性能报告
		r.Get("/performance-report", monitoringController.GeneratePerformanceReport)
	})
}
