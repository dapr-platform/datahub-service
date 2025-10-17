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
	"datahub-service/api/middleware"
	"datahub-service/service"
	"datahub-service/service/governance"
	"datahub-service/service/sharing"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

// InitRoute 初始化所有API路由
func InitRoute(r *chi.Mux) {
	// 基础中间件
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// CORS配置
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 初始化PostgREST认证中间件并应用（必须在所有路由之前）
	postgrestAuth := middleware.NewPostgRESTAuthMiddleware()
	r.Use(postgrestAuth.Middleware)

	// 健康检查（无需认证，在白名单中）
	healthController := controllers.NewHealthController()
	r.Get("/health", healthController.Health)
	r.Get("/ready", healthController.Ready)

	// SSE事件订阅（需要认证）
	eventController := controllers.NewEventController()
	r.Get("/sse/{user_name}", eventController.HandleSSE)

	// 事件管理（需要认证）
	r.Route("/events", func(r chi.Router) {
		r.Post("/send", eventController.SendEvent)
		r.Post("/broadcast", eventController.BroadcastEvent)

		// 列表查询接口
		r.Get("/connections", eventController.GetSSEConnectionList)
		r.Get("/history", eventController.GetEventHistoryList)
	})

	// 表管理（需要认证）
	r.Route("/tables", func(r chi.Router) {
		tableController := controllers.NewTableController()
		r.Post("/manage-schema", tableController.ManageTableSchema)
	})

	// 元数据管理（需要认证）
	r.Route("/meta", func(r chi.Router) {
		metaController := controllers.NewMetaController()

		// 通用同步任务元数据（基础库和主题库共用）
		r.Get("/sync-tasks", metaController.GetSyncTaskMeta)

		// 基础库相关元数据
		r.Route("/basic-libraries", func(r chi.Router) {
			r.Get("/data-sources", metaController.GetDataSourceTypes)
			r.Get("/data-interface-configs", metaController.GetDataInterfaceConfigs)
		})

		// 主题库相关元数据
		r.Route("/thematic-libraries", func(r chi.Router) {
			r.Get("/categories", metaController.GetThematicLibraryCategories)
			r.Get("/domains", metaController.GetThematicLibraryDomains)
			r.Get("/access-levels", metaController.GetThematicLibraryAccessLevels)
			r.Get("/statuses", metaController.GetThematicLibraryStatuses)
			r.Get("/interface-types", metaController.GetThematicInterfaceTypes)
			r.Get("/interface-statuses", metaController.GetThematicInterfaceStatuses)
			r.Get("/all", metaController.GetThematicLibraryAllMetadata)
		})

		// 主题库同步相关元数据
		r.Get("/thematic-sync-tasks", metaController.GetThematicSyncTaskMeta)
		r.Get("/thematic-sync-configs", metaController.GetThematicSyncConfigDefinitions)

		// 数据治理相关元数据（统一接口）
		r.Get("/data-governance", metaController.GetDataGovernanceMetadata)
	})

	// 基础库管理（保留现有功能接口）
	r.Route("/basic-libraries", func(r chi.Router) {
		basicLibraryController := controllers.NewBasicLibraryController()

		// 列表查询接口
		r.Get("/", basicLibraryController.GetBasicLibraryList)
		r.Get("/datasources", basicLibraryController.GetDataSourceList)
		r.Get("/interfaces", basicLibraryController.GetDataInterfaceList)
		r.Get("/interfaces/{id}", basicLibraryController.GetDataInterface)

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
		r.Delete("/{id}", basicLibraryController.DeleteBasicLibrary)

		// 添加数据源
		r.Post("/add-datasource", basicLibraryController.AddDataSource)

		// 修改数据源
		r.Post("/update-datasource", basicLibraryController.UpdateDataSource)

		// 删除数据源
		r.Delete("/datasources/{id}", basicLibraryController.DeleteDataSource)

		// 添加数据接口
		r.Post("/add-interface", basicLibraryController.AddInterface)

		// 修改数据接口
		r.Post("/update-interface", basicLibraryController.UpdateInterface)

		// 删除数据接口
		r.Delete("/interfaces/{id}", basicLibraryController.DeleteInterface)

		// 更新接口字段配置
		r.Post("/update-interface-fields", basicLibraryController.UpdateInterfaceFields)

		// CSV导入接口
		r.Post("/import-csv", basicLibraryController.ImportCSV)

		// 表字段和索引管理接口
		r.Get("/interfaces/{id}/table-columns", basicLibraryController.GetInterfaceTableColumns)
		r.Get("/interfaces/{id}/table-indexes", basicLibraryController.GetInterfaceTableIndexes)
		r.Post("/interfaces/create-table-index", basicLibraryController.CreateInterfaceTableIndex)
		r.Post("/interfaces/drop-table-index", basicLibraryController.DropInterfaceTableIndex)

		// 数据源管理器相关接口
		r.Get("/datasource-manager-stats", basicLibraryController.GetDataSourceManagerStats)
		r.Get("/resident-datasources", basicLibraryController.GetResidentDataSources)
		r.Post("/restart-resident-datasource/{id}", basicLibraryController.RestartResidentDataSource)
		r.Post("/reload-datasource/{id}", basicLibraryController.ReloadDataSource)
		r.Post("/health-check-all", basicLibraryController.HealthCheckAllDataSources)
	})

	// 主题库管理
	r.Route("/thematic-libraries", func(r chi.Router) {
		thematicLibraryController := controllers.NewThematicLibraryController()

		// 列表查询接口
		r.Get("/", thematicLibraryController.GetThematicLibraryList)

		// 基础CRUD操作
		r.Post("/", thematicLibraryController.CreateThematicLibrary)
		r.Get("/{id}", thematicLibraryController.GetThematicLibrary)
		r.Put("/{id}", thematicLibraryController.UpdateThematicLibrary)
		r.Delete("/{id}", thematicLibraryController.DeleteThematicLibrary)

		// 发布操作
		r.Post("/{id}/publish", thematicLibraryController.PublishThematicLibrary)
	})

	// 主题接口管理
	r.Route("/thematic-interfaces", func(r chi.Router) {
		thematicLibraryController := controllers.NewThematicLibraryController()

		// 列表查询接口
		r.Get("/", thematicLibraryController.GetThematicInterfaceList)

		// 基础CRUD操作
		r.Post("/", thematicLibraryController.CreateThematicInterface)
		r.Get("/{id}", thematicLibraryController.GetThematicInterface)
		r.Put("/{id}", thematicLibraryController.UpdateThematicInterface)
		r.Delete("/{id}", thematicLibraryController.DeleteThematicInterface)

		// 更新主题接口字段配置
		r.Post("/update-fields", thematicLibraryController.UpdateThematicInterfaceFields)

		// 视图管理
		r.Post("/create-view", thematicLibraryController.CreateThematicInterfaceView)
		r.Post("/update-view", thematicLibraryController.UpdateThematicInterfaceView)
		r.Delete("/{id}/delete-view", thematicLibraryController.DeleteThematicInterfaceView)
		r.Get("/{id}/view-sql", thematicLibraryController.GetThematicInterfaceViewSQL)

		// 表字段和索引管理接口
		r.Get("/{id}/table-columns", thematicLibraryController.GetThematicInterfaceTableColumns)
		r.Get("/{id}/table-indexes", thematicLibraryController.GetThematicInterfaceTableIndexes)
		r.Post("/create-table-index", thematicLibraryController.CreateThematicInterfaceTableIndex)
		r.Post("/drop-table-index", thematicLibraryController.DropThematicInterfaceTableIndex)
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
			r.Post("/{id}/cancel", syncTaskController.CancelSyncTask) // 保留向后兼容，实际为暂停
			r.Post("/{id}/retry", syncTaskController.RetrySyncTask)
			r.Get("/{id}/status", syncTaskController.GetSyncTaskStatus)

			// 任务状态管理（新增）
			r.Post("/{id}/activate", syncTaskController.ActivateSyncTask) // 激活任务（draft/paused → active）
			r.Post("/{id}/pause", syncTaskController.PauseSyncTask)       // 暂停任务（active → paused）
			r.Post("/{id}/resume", syncTaskController.ResumeSyncTask)     // 恢复任务（paused → active）

			// 任务执行记录
			r.Get("/{id}/executions", syncTaskController.GetTaskExecutions)

			// 批量操作
			r.Post("/batch-delete", syncTaskController.BatchDeleteSyncTasks)

			// 统计信息
			r.Get("/statistics", syncTaskController.GetSyncTaskStatistics)

			// 执行记录管理
			r.Get("/executions", syncTaskController.GetSyncTaskExecutions)
			r.Get("/executions/{id}", syncTaskController.GetSyncTaskExecution)
		})
	})

	// 数据质量管理（统一入口）
	r.Route("/data-quality", func(r chi.Router) {
		dataQualityController := controllers.NewDataQualityController(governance.NewGovernanceService(service.DB))

		// 质量规则管理
		r.Route("/rules", func(r chi.Router) {
			r.Post("/", dataQualityController.CreateQualityRule)
			r.Get("/", dataQualityController.GetQualityRules)
			r.Get("/{id}", dataQualityController.GetQualityRuleByID)
			r.Put("/{id}", dataQualityController.UpdateQualityRule)
			r.Delete("/{id}", dataQualityController.DeleteQualityRule)
		})

		// 数据脱敏规则管理
		r.Route("/masking-rules", func(r chi.Router) {
			r.Post("/", dataQualityController.CreateMaskingRule)
			r.Get("/", dataQualityController.GetMaskingRules)
			r.Get("/{id}", dataQualityController.GetMaskingRuleByID)
			r.Put("/{id}", dataQualityController.UpdateMaskingRule)
			r.Delete("/{id}", dataQualityController.DeleteMaskingRule)
		})

		// 数据清洗规则管理
		r.Route("/cleansing-rules", func(r chi.Router) {
			r.Post("/", dataQualityController.CreateCleansingRule)
			r.Get("/", dataQualityController.GetCleansingRules)
			r.Get("/{id}", dataQualityController.GetCleansingRuleByID)
			r.Put("/{id}", dataQualityController.UpdateCleansingRule)
			r.Delete("/{id}", dataQualityController.DeleteCleansingRule)
		})

		// 数据质量检测任务管理
		r.Route("/tasks", func(r chi.Router) {
			r.Post("/", dataQualityController.CreateQualityTask)
			r.Get("/", dataQualityController.GetQualityTasks)
			r.Get("/{id}", dataQualityController.GetQualityTaskByID)
			r.Put("/{id}", dataQualityController.UpdateQualityTask)
			r.Delete("/{id}", dataQualityController.DeleteQualityTask)
			r.Post("/{id}/start", dataQualityController.StartQualityTask)
			r.Post("/{id}/stop", dataQualityController.StopQualityTask)
			r.Get("/{id}/executions", dataQualityController.GetQualityTaskExecutions)
		})

		// 数据血缘管理
		r.Route("/data-lineage", func(r chi.Router) {
			r.Post("/", dataQualityController.CreateDataLineage)
			r.Get("/", dataQualityController.GetDataLineage)
		})

		// 质量检查
		r.Post("/checks", dataQualityController.RunQualityCheck)

		// 质量报告
		r.Route("/reports", func(r chi.Router) {
			r.Get("/", dataQualityController.GetQualityReports)
			r.Get("/{id}", dataQualityController.GetQualityReportByID)
		})

		// 元数据管理
		r.Route("/metadata", func(r chi.Router) {
			r.Post("/", dataQualityController.CreateMetadata)
			r.Get("/", dataQualityController.GetMetadataList)
			r.Get("/{id}", dataQualityController.GetMetadataByID)
			r.Put("/{id}", dataQualityController.UpdateMetadata)
			r.Delete("/{id}", dataQualityController.DeleteMetadata)
		})

		// 系统日志管理
		r.Get("/system-logs", dataQualityController.GetSystemLogs)

		// 模板管理
		r.Route("/templates", func(r chi.Router) {
			r.Get("/quality-rules", dataQualityController.GetQualityRuleTemplates)
			r.Get("/masking-rules", dataQualityController.GetDataMaskingTemplates)
			r.Get("/cleansing-rules", dataQualityController.GetDataCleansingTemplates)
		})

		// 规则测试
		r.Route("/test", func(r chi.Router) {
			r.Post("/quality-rule", dataQualityController.TestQualityRule)
			r.Post("/masking-rule", dataQualityController.TestMaskingRule)
			r.Post("/cleansing-rule", dataQualityController.TestCleansingRule)
			r.Post("/batch-rules", dataQualityController.TestBatchRules)
			r.Post("/rule-preview", dataQualityController.TestRulePreview)
		})
	})

	// 数据共享服务
	r.Route("/sharing", func(r chi.Router) {
		sharingController := controllers.NewSharingController(sharing.NewSharingService(service.DB))

		// API应用管理
		r.Route("/api-applications", func(r chi.Router) {
			r.Post("/", sharingController.CreateApiApplication)
			r.Get("/", sharingController.GetApiApplications)
			r.Get("/{id}", sharingController.GetApiApplicationByID)
			r.Put("/{id}", sharingController.UpdateApiApplication)
			r.Delete("/{id}", sharingController.DeleteApiApplication)
		})

		// ApiKey管理（独立路由）
		r.Route("/api-keys", func(r chi.Router) {
			r.Post("/", sharingController.CreateApiKey)
			r.Get("/", sharingController.GetApiKeys)
			r.Get("/{id}", sharingController.GetApiKeyByID)
			r.Put("/{id}", sharingController.UpdateApiKey)
			r.Delete("/{id}", sharingController.DeleteApiKey)
			r.Put("/{id}/applications", sharingController.UpdateApiKeyApplications)
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

		// API使用日志管理
		r.Get("/api-usage-logs", sharingController.GetApiUsageLogs)

		// API接口管理
		r.Route("/api-interfaces", func(r chi.Router) {
			r.Post("/", sharingController.CreateApiInterface)
			r.Get("/", sharingController.GetApiInterfaces)
			r.Delete("/{id}", sharingController.DeleteApiInterface)
		})
	})

	// 数据访问代理API（只读查询）
	r.Route("/api/v1", func(r chi.Router) {
		dataProxyController := controllers.NewDataProxyController(sharing.NewSharingService(service.DB))

		// 数据代理接口，URL格式：/api/v1/share/{app_path}/{interface_path}
		r.Route("/share", func(r chi.Router) {
			// 通过API Key获取应用信息和接口列表，URL格式：/api/v1/share/
			r.Get("/", dataProxyController.GetApiApplicationByKey)
			// 获取应用信息和接口列表，URL格式：/api/v1/share/{app_path}
			r.Get("/{app_path}", dataProxyController.GetApplicationInfo)

			// 只支持GET和HEAD方法的代理请求
			r.Get("/{app_path}/{interface_path}", dataProxyController.ProxyDataAccess)
			r.Head("/{app_path}/{interface_path}", dataProxyController.ProxyDataAccess)
			r.Get("/{app_path}/{interface_path}/*", dataProxyController.ProxyDataAccess)
			r.Head("/{app_path}/{interface_path}/*", dataProxyController.ProxyDataAccess)
		})
	})

	// 监控管理（简化版 - 仅基于 VictoriaMetrics 和 Loki）
	r.Route("/monitoring", func(r chi.Router) {
		monitoringController := controllers.NewMonitoringController()

		// 通用查询接口
		r.Post("/query", monitoringController.ExecuteCustomQuery)
		r.Post("/query/metrics", monitoringController.QueryMetrics)
		r.Post("/query/logs", monitoringController.QueryLogs)
		r.Post("/query/validate", monitoringController.ValidateQuery)

		// 查询模板
		r.Get("/templates/metrics", monitoringController.GetMetricTemplates)
		r.Get("/templates/logs", monitoringController.GetLogTemplates)

		// 指标和日志描述
		r.Get("/metrics/descriptions", monitoringController.GetMetricDescriptions)
		r.Get("/logs/descriptions", monitoringController.GetLogTemplateDescriptions)

		// Loki 标签值查询
		r.Get("/loki/labels/{label}/values", monitoringController.GetLokiLabels)

		// 监控配置
		r.Get("/config", monitoringController.GetMonitoringConfig)
	})

	// 主题同步管理
	r.Route("/thematic-sync", func(r chi.Router) {
		thematicSyncController := controllers.NewThematicSyncController()

		// 同步任务管理
		r.Route("/tasks", func(r chi.Router) {
			// 基础CRUD操作
			r.Post("/", thematicSyncController.CreateSyncTask)
			r.Get("/", thematicSyncController.GetSyncTaskList)
			r.Get("/{id}", thematicSyncController.GetSyncTask)
			r.Put("/{id}", thematicSyncController.UpdateSyncTask)
			r.Delete("/{id}", thematicSyncController.DeleteSyncTask)

			// 任务控制操作
			r.Post("/{id}/execute", thematicSyncController.ExecuteSyncTask)
			r.Get("/{id}/status", thematicSyncController.GetSyncTaskStatus)

			// 任务执行记录
			r.Get("/{id}/executions", thematicSyncController.GetSyncTaskExecutions)

			// 统计信息
			r.Get("/statistics", thematicSyncController.GetSyncTaskStatistics)
		})

		// 执行记录管理
		r.Route("/executions", func(r chi.Router) {
			r.Get("/{id}", thematicSyncController.GetSyncExecution)
		})
	})

	// 数据查看路由
	dataViewController := controllers.NewDataViewController(service.DB)
	r.Route("/data-view", func(r chi.Router) {
		// 获取库的所有表
		r.Get("/{library_type}/{library_id}/tables", dataViewController.GetLibraryTables)

		// 获取表数据
		r.Get("/{library_type}/{library_id}/tables/{table_name}/data", dataViewController.GetTableData)

		// 获取表结构
		r.Get("/{library_type}/{library_id}/tables/{table_name}/structure", dataViewController.GetTableStructure)
	})

	// Dashboard统计数据（需要认证）
	r.Route("/dashboard", func(r chi.Router) {
		dashboardController := controllers.NewDashboardController()

		// 总览数据
		r.Get("/overview", dashboardController.GetDashboardOverview)

		// 各模块独立统计接口
		r.Get("/basic-library-stats", dashboardController.GetBasicLibraryStats)
		r.Get("/thematic-library-stats", dashboardController.GetThematicLibraryStats)
		r.Get("/sync-task-stats", dashboardController.GetSyncTaskStats)
		r.Get("/data-quality-stats", dashboardController.GetDataQualityStats)
		r.Get("/data-sharing-stats", dashboardController.GetDataSharingStats)
		r.Get("/system-activity-stats", dashboardController.GetSystemActivityStats)
	})

	// 认证中间件管理接口（需要管理员权限）
	r.Route("/admin/auth", func(r chi.Router) {
		// 需要管理员权限（全局中间件已经处理了基本认证）
		r.Use(middleware.RequireRole("admin"))

		// 获取缓存统计信息
		r.Get("/cache-stats", func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, map[string]interface{}{
				"status": 200,
				"msg":    "获取缓存统计成功",
				"data":   postgrestAuth.GetCacheStats(),
			})
		})

		// 清理过期缓存
		r.Post("/clear-expired-cache", func(w http.ResponseWriter, r *http.Request) {
			clearedCount := postgrestAuth.ClearExpiredCache()
			render.JSON(w, r, map[string]interface{}{
				"status": 200,
				"msg":    "清理过期缓存成功",
				"data": map[string]interface{}{
					"cleared_count": clearedCount,
				},
			})
		})
	})
}
