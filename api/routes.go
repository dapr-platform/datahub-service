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

	// 数据基础库管理
	r.Route("/basic-libraries", func(r chi.Router) {
		basicLibraryController := controllers.NewBasicLibraryController()

		r.Post("/", basicLibraryController.CreateBasicLibrary)
		r.Get("/", basicLibraryController.GetBasicLibraries)
		r.Get("/{id}", basicLibraryController.GetBasicLibrary)
		r.Put("/{id}", basicLibraryController.UpdateBasicLibrary)
		r.Delete("/{id}", basicLibraryController.DeleteBasicLibrary)
	})

	// 数据主题库管理
	r.Route("/thematic-libraries", func(r chi.Router) {
		thematicLibraryController := controllers.NewThematicLibraryController()

		r.Post("/", thematicLibraryController.CreateThematicLibrary)
		r.Get("/", thematicLibraryController.GetThematicLibraries)
		r.Get("/{id}", thematicLibraryController.GetThematicLibrary)
		r.Put("/{id}", thematicLibraryController.UpdateThematicLibrary)
		r.Delete("/{id}", thematicLibraryController.DeleteThematicLibrary)
		r.Post("/{id}/publish", thematicLibraryController.PublishThematicLibrary)
	})

	// 主题库接口管理
	r.Route("/thematic-interfaces", func(r chi.Router) {
		thematicLibraryController := controllers.NewThematicLibraryController()

		r.Post("/", thematicLibraryController.CreateThematicInterface)
		r.Get("/", thematicLibraryController.GetThematicInterfaces)
		r.Get("/{id}", thematicLibraryController.GetThematicInterface)
	})

	// 访问控制管理已移除，改为使用PostgREST RBAC
	// 权限管理通过PostgREST提供，详见ai_docs/postgrest_rbac_guide.md

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
}
