/*
 * @module api/controllers/dashboard_controller
 * @description Dashboard统计数据控制器，提供系统总览和关键指标数据
 * @architecture MVC架构 - 控制器层
 * @documentReference dev_docs/requirements.md
 * @stateFlow HTTP请求处理流程
 * @rules 统一的错误处理和响应格式，数据聚合展示
 * @dependencies datahub-service/service, github.com/go-chi/render
 * @refs api/routes.go
 */

package controllers

import (
	"database/sql"
	"datahub-service/service"
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"datahub-service/service/sharing"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"gorm.io/gorm"
)

// DashboardController Dashboard控制器
type DashboardController struct {
	db                *gorm.DB
	governanceService *governance.GovernanceService
	sharingService    *sharing.SharingService
}

// NewDashboardController 创建Dashboard控制器实例
func NewDashboardController() *DashboardController {
	return &DashboardController{
		db:                service.DB,
		governanceService: governance.NewGovernanceService(service.DB),
		sharingService:    sharing.NewSharingService(service.DB),
	}
}

// DashboardOverviewResponse Dashboard总览响应
type DashboardOverviewResponse struct {
	// 基础库统计
	BasicLibraryStats BasicLibraryStats `json:"basic_library_stats"`

	// 主题库统计
	ThematicLibraryStats ThematicLibraryStats `json:"thematic_library_stats"`

	// 同步任务统计
	SyncTaskStats SyncTaskStats `json:"sync_task_stats"`

	// 数据质量统计
	DataQualityStats DataQualityStats `json:"data_quality_stats"`

	// 数据共享统计
	DataSharingStats DataSharingStats `json:"data_sharing_stats"`

	// 系统活动统计
	SystemActivityStats SystemActivityStats `json:"system_activity_stats"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// BasicLibraryStats 基础库统计
type BasicLibraryStats struct {
	TotalLibraries    int64                  `json:"total_libraries"`     // 总数
	ActiveLibraries   int64                  `json:"active_libraries"`    // 活跃数
	InactiveLibraries int64                  `json:"inactive_libraries"`  // 非活跃数
	TotalDataSources  int64                  `json:"total_data_sources"`  // 数据源总数
	ActiveDataSources int64                  `json:"active_data_sources"` // 活跃数据源
	TotalInterfaces   int64                  `json:"total_interfaces"`    // 接口总数
	ActiveInterfaces  int64                  `json:"active_interfaces"`   // 活跃接口
	CategoryBreakdown []CategoryCount        `json:"category_breakdown"`  // 分类统计
	RecentLibraries   []RecentLibraryInfo    `json:"recent_libraries"`    // 最近创建的库
	TopDataSources    []DataSourceUsageStats `json:"top_data_sources"`    // 数据源使用排行
}

// ThematicLibraryStats 主题库统计
type ThematicLibraryStats struct {
	TotalLibraries     int64                       `json:"total_libraries"`      // 总数
	PublishedLibraries int64                       `json:"published_libraries"`  // 已发布
	DraftLibraries     int64                       `json:"draft_libraries"`      // 草稿
	ArchivedLibraries  int64                       `json:"archived_libraries"`   // 已归档
	TotalInterfaces    int64                       `json:"total_interfaces"`     // 接口总数
	ViewInterfaces     int64                       `json:"view_interfaces"`      // 视图类型接口
	TableInterfaces    int64                       `json:"table_interfaces"`     // 表类型接口
	CategoryBreakdown  []CategoryCount             `json:"category_breakdown"`   // 分类统计
	DomainBreakdown    []DomainCount               `json:"domain_breakdown"`     // 领域统计
	RecentLibraries    []RecentThematicLibraryInfo `json:"recent_libraries"`     // 最近创建的主题库
	AccessLevelStats   []AccessLevelCount          `json:"access_level_stats"`   // 访问级别统计
	InterfaceTypeStats []InterfaceTypeCount        `json:"interface_type_stats"` // 接口类型统计
	TopAccessLibraries []LibraryAccessStats        `json:"top_access_libraries"` // 访问量排行
	TrendData          []LibraryTrendData          `json:"trend_data"`           // 趋势数据
}

// SyncTaskStats 同步任务统计
type SyncTaskStats struct {
	TotalTasks         int64                 `json:"total_tasks"`          // 总任务数
	RunningTasks       int64                 `json:"running_tasks"`        // 运行中
	PendingTasks       int64                 `json:"pending_tasks"`        // 等待中
	CompletedTasks     int64                 `json:"completed_tasks"`      // 已完成
	FailedTasks        int64                 `json:"failed_tasks"`         // 失败
	TodayExecutions    int64                 `json:"today_executions"`     // 今日执行次数
	TodaySuccessRate   float64               `json:"today_success_rate"`   // 今日成功率
	AvgExecutionTime   float64               `json:"avg_execution_time"`   // 平均执行时间(秒)
	TotalDataSynced    int64                 `json:"total_data_synced"`    // 总同步数据量
	RecentExecutions   []RecentExecutionInfo `json:"recent_executions"`    // 最近执行记录
	TaskTypeBreakdown  []TaskTypeCount       `json:"task_type_breakdown"`  // 任务类型分布
	TriggerTypeStats   []TriggerTypeCount    `json:"trigger_type_stats"`   // 触发类型统计
	ExecutionTrendData []ExecutionTrendData  `json:"execution_trend_data"` // 执行趋势数据
	FailureReasons     []FailureReasonStats  `json:"failure_reasons"`      // 失败原因统计
}

// DataQualityStats 数据质量统计
type DataQualityStats struct {
	TotalQualityRules    int64                 `json:"total_quality_rules"`    // 质量规则总数
	EnabledQualityRules  int64                 `json:"enabled_quality_rules"`  // 启用的规则
	TotalQualityTasks    int64                 `json:"total_quality_tasks"`    // 质量检测任务总数
	RunningQualityTasks  int64                 `json:"running_quality_tasks"`  // 运行中的任务
	TotalQualityReports  int64                 `json:"total_quality_reports"`  // 质量报告总数
	AvgQualityScore      float64               `json:"avg_quality_score"`      // 平均质量分数
	TotalMaskingRules    int64                 `json:"total_masking_rules"`    // 脱敏规则总数
	TotalCleansingRules  int64                 `json:"total_cleansing_rules"`  // 清洗规则总数
	QualityIssueCount    int64                 `json:"quality_issue_count"`    // 质量问题数量
	RecentQualityReports []RecentQualityReport `json:"recent_quality_reports"` // 最近质量报告
	RuleTypeBreakdown    []RuleTypeCount       `json:"rule_type_breakdown"`    // 规则类型分布
	QualityTrendData     []QualityTrendData    `json:"quality_trend_data"`     // 质量趋势
	IssueTypeStats       []IssueTypeCount      `json:"issue_type_stats"`       // 问题类型统计
	IssueSeverityStats   []IssueSeverityCount  `json:"issue_severity_stats"`   // 问题严重程度统计
}

// DataSharingStats 数据共享统计
type DataSharingStats struct {
	TotalApiApplications    int64                  `json:"total_api_applications"`    // API应用总数
	ActiveApiApplications   int64                  `json:"active_api_applications"`   // 活跃应用
	TotalApiKeys            int64                  `json:"total_api_keys"`            // API密钥总数
	ActiveApiKeys           int64                  `json:"active_api_keys"`           // 活跃密钥
	TotalApiInterfaces      int64                  `json:"total_api_interfaces"`      // API接口总数
	TotalApiCalls           int64                  `json:"total_api_calls"`           // API调用总数
	TodayApiCalls           int64                  `json:"today_api_calls"`           // 今日调用
	TotalDataSubscriptions  int64                  `json:"total_data_subscriptions"`  // 数据订阅总数
	ActiveDataSubscriptions int64                  `json:"active_data_subscriptions"` // 活跃订阅
	TotalAccessRequests     int64                  `json:"total_access_requests"`     // 访问申请总数
	PendingAccessRequests   int64                  `json:"pending_access_requests"`   // 待审批申请
	ApprovedAccessRequests  int64                  `json:"approved_access_requests"`  // 已批准申请
	TopApiApplications      []ApiApplicationStats  `json:"top_api_applications"`      // 热门应用
	ApiCallTrendData        []ApiCallTrendData     `json:"api_call_trend_data"`       // API调用趋势
	RecentApiUsageLogs      []RecentApiUsageLog    `json:"recent_api_usage_logs"`     // 最近API使用日志
	ResponseTimeStats       ResponseTimeStatistics `json:"response_time_stats"`       // 响应时间统计
	ErrorRateStats          ErrorRateStatistics    `json:"error_rate_stats"`          // 错误率统计
}

// SystemActivityStats 系统活动统计
type SystemActivityStats struct {
	TotalUsers             int64                   `json:"total_users"`              // 用户总数
	ActiveUsers            int64                   `json:"active_users"`             // 活跃用户
	TodayActiveUsers       int64                   `json:"today_active_users"`       // 今日活跃
	TotalOperations        int64                   `json:"total_operations"`         // 总操作数
	TodayOperations        int64                   `json:"today_operations"`         // 今日操作
	RecentSystemLogs       []RecentSystemLog       `json:"recent_system_logs"`       // 最近系统日志
	OperationTypeBreakdown []OperationTypeCount    `json:"operation_type_breakdown"` // 操作类型分布
	UserActivityTrendData  []UserActivityTrendData `json:"user_activity_trend_data"` // 用户活动趋势
	PeakUsageTime          *PeakUsageTimeInfo      `json:"peak_usage_time"`          // 高峰使用时段
}

// 辅助结构体定义
type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type DomainCount struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

type AccessLevelCount struct {
	AccessLevel string `json:"access_level"`
	Count       int64  `json:"count"`
}

type InterfaceTypeCount struct {
	InterfaceType string `json:"interface_type"`
	Count         int64  `json:"count"`
}

type RecentLibraryInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy *string   `json:"created_by"`
}

type RecentThematicLibraryInfo struct {
	ID        string    `json:"id"`
	NameZh    string    `json:"name_zh"`
	NameEn    string    `json:"name_en"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy *string   `json:"created_by"`
}

type DataSourceUsageStats struct {
	DataSourceID   string     `json:"data_source_id"`
	DataSourceName string     `json:"data_source_name"`
	UsageCount     int64      `json:"usage_count"`
	LastUsedAt     *time.Time `json:"last_used_at"`
}

type LibraryAccessStats struct {
	LibraryID   string `json:"library_id"`
	LibraryName string `json:"library_name"`
	AccessCount int64  `json:"access_count"`
}

type LibraryTrendData struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type RecentExecutionInfo struct {
	ExecutionID  string     `json:"execution_id"`
	TaskID       string     `json:"task_id"`
	TaskName     *string    `json:"task_name"`
	Status       string     `json:"status"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	RecordCount  *int64     `json:"record_count"`
	ErrorMessage *string    `json:"error_message"`
}

type TaskTypeCount struct {
	TaskType string `json:"task_type"`
	Count    int64  `json:"count"`
}

type TriggerTypeCount struct {
	TriggerType string `json:"trigger_type"`
	Count       int64  `json:"count"`
}

type ExecutionTrendData struct {
	Date         string `json:"date"`
	SuccessCount int64  `json:"success_count"`
	FailureCount int64  `json:"failure_count"`
}

type FailureReasonStats struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type RecentQualityReport struct {
	ReportID     string    `json:"report_id"`
	ReportName   string    `json:"report_name"`
	QualityScore float64   `json:"quality_score"`
	GeneratedAt  time.Time `json:"generated_at"`
	ObjectType   string    `json:"object_type"`
}

type RuleTypeCount struct {
	RuleType string `json:"rule_type"`
	Count    int64  `json:"count"`
}

type QualityTrendData struct {
	Date       string  `json:"date"`
	AvgScore   float64 `json:"avg_score"`
	IssueCount int64   `json:"issue_count"`
}

type IssueTypeCount struct {
	IssueType string `json:"issue_type"`
	Count     int64  `json:"count"`
}

type IssueSeverityCount struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

type ApiApplicationStats struct {
	ApplicationID   string     `json:"application_id"`
	ApplicationName string     `json:"application_name"`
	CallCount       int64      `json:"call_count"`
	LastCallTime    *time.Time `json:"last_call_time"`
}

type ApiCallTrendData struct {
	Date      string `json:"date"`
	CallCount int64  `json:"call_count"`
}

type RecentApiUsageLog struct {
	LogID        string    `json:"log_id"`
	ApiPath      string    `json:"api_path"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int       `json:"response_time"`
	CreatedAt    time.Time `json:"created_at"`
	RequestIP    string    `json:"request_ip"`
}

type ResponseTimeStatistics struct {
	AvgResponseTime float64 `json:"avg_response_time"` // 平均响应时间(ms)
	MinResponseTime int     `json:"min_response_time"` // 最小响应时间(ms)
	MaxResponseTime int     `json:"max_response_time"` // 最大响应时间(ms)
	P50ResponseTime float64 `json:"p50_response_time"` // P50响应时间(ms)
	P95ResponseTime float64 `json:"p95_response_time"` // P95响应时间(ms)
	P99ResponseTime float64 `json:"p99_response_time"` // P99响应时间(ms)
}

type ErrorRateStatistics struct {
	TotalRequests int64   `json:"total_requests"`
	ErrorRequests int64   `json:"error_requests"`
	ErrorRate     float64 `json:"error_rate"` // 错误率百分比
}

type RecentSystemLog struct {
	LogID           string    `json:"log_id"`
	OperationType   string    `json:"operation_type"`
	ObjectType      string    `json:"object_type"`
	OperatorName    *string   `json:"operator_name"`
	OperationTime   time.Time `json:"operation_time"`
	OperationResult string    `json:"operation_result"`
}

type OperationTypeCount struct {
	OperationType string `json:"operation_type"`
	Count         int64  `json:"count"`
}

type UserActivityTrendData struct {
	Date        string `json:"date"`
	ActiveUsers int64  `json:"active_users"`
	Operations  int64  `json:"operations"`
}

type PeakUsageTimeInfo struct {
	Hour          int     `json:"hour"`           // 小时 (0-23)
	AvgOperations float64 `json:"avg_operations"` // 该时段平均操作数
}

// GetDashboardOverview 获取Dashboard总览数据
// @Summary 获取Dashboard总览数据
// @Description 获取系统各模块的统计数据和关键指标
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=DashboardOverviewResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/overview [get]
func (c *DashboardController) GetDashboardOverview(w http.ResponseWriter, r *http.Request) {
	overview := DashboardOverviewResponse{
		BasicLibraryStats:    c.getBasicLibraryStats(),
		ThematicLibraryStats: c.getThematicLibraryStats(),
		SyncTaskStats:        c.getSyncTaskStats(),
		DataQualityStats:     c.getDataQualityStats(),
		DataSharingStats:     c.getDataSharingStats(),
		SystemActivityStats:  c.getSystemActivityStats(),
		UpdatedAt:            time.Now(),
	}

	render.JSON(w, r, SuccessResponse("获取Dashboard总览数据成功", overview))
}

// getBasicLibraryStats 获取基础库统计数据
func (c *DashboardController) getBasicLibraryStats() BasicLibraryStats {
	stats := BasicLibraryStats{
		CategoryBreakdown: []CategoryCount{},
		RecentLibraries:   []RecentLibraryInfo{},
		TopDataSources:    []DataSourceUsageStats{},
	}

	// 基础库总数和状态统计
	c.db.Model(&models.BasicLibrary{}).Count(&stats.TotalLibraries)
	c.db.Model(&models.BasicLibrary{}).Where("status = ?", "active").Count(&stats.ActiveLibraries)
	c.db.Model(&models.BasicLibrary{}).Where("status = ?", "inactive").Count(&stats.InactiveLibraries)

	// 数据源统计
	c.db.Model(&models.DataSource{}).Count(&stats.TotalDataSources)
	c.db.Model(&models.DataSource{}).Where("status = ?", "active").Count(&stats.ActiveDataSources)

	// 接口统计
	c.db.Model(&models.DataInterface{}).Count(&stats.TotalInterfaces)
	c.db.Model(&models.DataInterface{}).Where("status = ?", "active").Count(&stats.ActiveInterfaces)

	// 分类统计 (按数据源分类)
	c.db.Model(&models.DataSource{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Find(&stats.CategoryBreakdown)

	// 最近创建的基础库 (前5条)
	c.db.Model(&models.BasicLibrary{}).
		Select("id, name_zh as name, status, created_at, created_by").
		Order("created_at DESC").
		Limit(5).
		Find(&stats.RecentLibraries)

	// 数据源使用排行 (前10个，基于关联的接口数量)
	c.db.Table("t_data_source ds").
		Select("ds.id as data_source_id, ds.name as data_source_name, COUNT(di.id) as usage_count, MAX(di.updated_at) as last_used_at").
		Joins("LEFT JOIN t_data_interface di ON ds.id = di.data_source_id").
		Group("ds.id, ds.name").
		Order("usage_count DESC").
		Limit(10).
		Find(&stats.TopDataSources)

	return stats
}

// getThematicLibraryStats 获取主题库统计数据
func (c *DashboardController) getThematicLibraryStats() ThematicLibraryStats {
	stats := ThematicLibraryStats{
		CategoryBreakdown:  []CategoryCount{},
		DomainBreakdown:    []DomainCount{},
		RecentLibraries:    []RecentThematicLibraryInfo{},
		AccessLevelStats:   []AccessLevelCount{},
		InterfaceTypeStats: []InterfaceTypeCount{},
		TopAccessLibraries: []LibraryAccessStats{},
		TrendData:          []LibraryTrendData{},
	}

	// 主题库总数和状态统计
	c.db.Model(&models.ThematicLibrary{}).Count(&stats.TotalLibraries)
	c.db.Model(&models.ThematicLibrary{}).Where("status = ?", "published").Count(&stats.PublishedLibraries)
	c.db.Model(&models.ThematicLibrary{}).Where("status = ?", "draft").Count(&stats.DraftLibraries)
	c.db.Model(&models.ThematicLibrary{}).Where("status = ?", "archived").Count(&stats.ArchivedLibraries)

	// 主题接口统计
	c.db.Model(&models.ThematicInterface{}).Count(&stats.TotalInterfaces)
	c.db.Model(&models.ThematicInterface{}).Where("type = ?", "view").Count(&stats.ViewInterfaces)
	c.db.Model(&models.ThematicInterface{}).Where("type = ?", "table").Count(&stats.TableInterfaces)

	// 分类统计
	c.db.Model(&models.ThematicLibrary{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Find(&stats.CategoryBreakdown)

	// 领域统计
	c.db.Model(&models.ThematicLibrary{}).
		Select("domain, COUNT(*) as count").
		Group("domain").
		Find(&stats.DomainBreakdown)

	// 访问级别统计
	c.db.Model(&models.ThematicLibrary{}).
		Select("access_level, COUNT(*) as count").
		Group("access_level").
		Find(&stats.AccessLevelStats)

	// 接口类型统计
	c.db.Model(&models.ThematicInterface{}).
		Select("type as interface_type, COUNT(*) as count").
		Group("type").
		Find(&stats.InterfaceTypeStats)

	// 最近创建的主题库 (前5条)
	c.db.Model(&models.ThematicLibrary{}).
		Select("id, name_zh, name_en, category, status, created_at, created_by").
		Order("created_at DESC").
		Limit(5).
		Find(&stats.RecentLibraries)

	// 访问量排行 (基于API应用数量)
	c.db.Table("t_thematic_library tl").
		Select("tl.id as library_id, tl.name_zh as library_name, COUNT(aa.id) as access_count").
		Joins("LEFT JOIN t_api_application aa ON tl.id = aa.thematic_library_id").
		Group("tl.id, tl.name_zh").
		Order("access_count DESC").
		Limit(10).
		Find(&stats.TopAccessLibraries)

	// 趋势数据 (最近7天)
	c.db.Model(&models.ThematicLibrary{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&stats.TrendData)

	return stats
}

// getSyncTaskStats 获取同步任务统计数据
func (c *DashboardController) getSyncTaskStats() SyncTaskStats {
	stats := SyncTaskStats{
		RecentExecutions:   []RecentExecutionInfo{},
		TaskTypeBreakdown:  []TaskTypeCount{},
		TriggerTypeStats:   []TriggerTypeCount{},
		ExecutionTrendData: []ExecutionTrendData{},
		FailureReasons:     []FailureReasonStats{},
	}

	// 任务总数和状态统计
	c.db.Model(&models.SyncTask{}).Count(&stats.TotalTasks)
	c.db.Model(&models.SyncTask{}).Where("status = ?", "running").Count(&stats.RunningTasks)
	c.db.Model(&models.SyncTask{}).Where("status = ?", "pending").Count(&stats.PendingTasks)
	c.db.Model(&models.SyncTask{}).Where("status = ?", "completed").Count(&stats.CompletedTasks)
	c.db.Model(&models.SyncTask{}).Where("status = ?", "failed").Count(&stats.FailedTasks)

	// 今日执行统计
	today := time.Now().Format("2006-01-02")
	c.db.Model(&models.SyncTaskExecution{}).
		Where("DATE(start_time) = ?", today).
		Count(&stats.TodayExecutions)

	// 今日成功率
	var todaySuccess int64
	c.db.Model(&models.SyncTaskExecution{}).
		Where("DATE(start_time) = ? AND status = ?", today, "success").
		Count(&todaySuccess)
	if stats.TodayExecutions > 0 {
		stats.TodaySuccessRate = float64(todaySuccess) / float64(stats.TodayExecutions) * 100
	}

	// 平均执行时间
	var avgSeconds sql.NullFloat64
	c.db.Model(&models.SyncTaskExecution{}).
		Select("AVG(EXTRACT(EPOCH FROM (end_time - start_time)))").
		Where("status = ? AND end_time IS NOT NULL", "success").
		Scan(&avgSeconds)
	if avgSeconds.Valid {
		stats.AvgExecutionTime = avgSeconds.Float64
	}

	// 总同步数据量
	c.db.Model(&models.SyncTaskExecution{}).
		Select("COALESCE(SUM(record_count), 0)").
		Where("status = ?", "success").
		Scan(&stats.TotalDataSynced)

	// 最近执行记录 (前10条)
	c.db.Table("t_sync_task_execution ste").
		Select("ste.id as execution_id, ste.task_id, st.task_name, ste.status, ste.start_time, ste.end_time, ste.record_count, ste.error_message").
		Joins("LEFT JOIN t_sync_task st ON ste.task_id = st.id").
		Order("ste.start_time DESC").
		Limit(10).
		Find(&stats.RecentExecutions)

	// 任务类型分布
	c.db.Model(&models.SyncTask{}).
		Select("task_type, COUNT(*) as count").
		Group("task_type").
		Find(&stats.TaskTypeBreakdown)

	// 触发类型统计
	c.db.Model(&models.SyncTask{}).
		Select("trigger_type, COUNT(*) as count").
		Group("trigger_type").
		Find(&stats.TriggerTypeStats)

	// 执行趋势数据 (最近7天)
	c.db.Table("t_sync_task_execution").
		Select("DATE(start_time) as date, COUNT(CASE WHEN status = 'success' THEN 1 END) as success_count, COUNT(CASE WHEN status = 'failed' THEN 1 END) as failure_count").
		Where("start_time >= ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(start_time)").
		Order("date ASC").
		Find(&stats.ExecutionTrendData)

	// 失败原因统计 (前5个)
	c.db.Model(&models.SyncTaskExecution{}).
		Select("SUBSTRING(error_message FROM 1 FOR 50) as reason, COUNT(*) as count").
		Where("status = ? AND error_message IS NOT NULL", "failed").
		Group("SUBSTRING(error_message FROM 1 FOR 50)").
		Order("count DESC").
		Limit(5).
		Find(&stats.FailureReasons)

	return stats
}

// getDataQualityStats 获取数据质量统计数据
func (c *DashboardController) getDataQualityStats() DataQualityStats {
	stats := DataQualityStats{
		RecentQualityReports: []RecentQualityReport{},
		RuleTypeBreakdown:    []RuleTypeCount{},
		QualityTrendData:     []QualityTrendData{},
		IssueTypeStats:       []IssueTypeCount{},
		IssueSeverityStats:   []IssueSeverityCount{},
	}

	// 质量规则统计
	c.db.Model(&models.QualityRuleTemplate{}).Count(&stats.TotalQualityRules)
	c.db.Model(&models.QualityRuleTemplate{}).Where("is_enabled = ?", true).Count(&stats.EnabledQualityRules)

	// 质量任务统计
	c.db.Model(&models.QualityTask{}).Count(&stats.TotalQualityTasks)
	c.db.Model(&models.QualityTask{}).Where("status = ?", "running").Count(&stats.RunningQualityTasks)

	// 质量报告统计
	c.db.Model(&models.DataQualityReport{}).Count(&stats.TotalQualityReports)

	// 平均质量分数
	var avgScore sql.NullFloat64
	c.db.Model(&models.DataQualityReport{}).
		Select("AVG(quality_score)").
		Scan(&avgScore)
	if avgScore.Valid {
		stats.AvgQualityScore = avgScore.Float64
	}

	// 脱敏规则和清洗规则
	c.db.Model(&models.DataMaskingTemplate{}).Count(&stats.TotalMaskingRules)
	c.db.Model(&models.DataCleansingTemplate{}).Count(&stats.TotalCleansingRules)

	// 质量问题数量 (从报告中统计)
	c.db.Model(&models.DataQualityReport{}).
		Select("COALESCE(SUM(jsonb_array_length(issues)), 0)").
		Scan(&stats.QualityIssueCount)

	// 最近质量报告 (前5条)
	c.db.Model(&models.DataQualityReport{}).
		Select("id as report_id, report_name, quality_score, generated_at, related_object_type as object_type").
		Order("generated_at DESC").
		Limit(5).
		Find(&stats.RecentQualityReports)

	// 规则类型分布
	c.db.Model(&models.QualityRuleTemplate{}).
		Select("type as rule_type, COUNT(*) as count").
		Group("type").
		Find(&stats.RuleTypeBreakdown)

	// 质量趋势 (最近7天)
	c.db.Model(&models.DataQualityReport{}).
		Select("DATE(generated_at) as date, AVG(quality_score) as avg_score, COUNT(*) as issue_count").
		Where("generated_at >= ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(generated_at)").
		Order("date ASC").
		Find(&stats.QualityTrendData)

	return stats
}

// getDataSharingStats 获取数据共享统计数据
func (c *DashboardController) getDataSharingStats() DataSharingStats {
	stats := DataSharingStats{
		TopApiApplications: []ApiApplicationStats{},
		ApiCallTrendData:   []ApiCallTrendData{},
		RecentApiUsageLogs: []RecentApiUsageLog{},
	}

	// API应用统计
	c.db.Model(&models.ApiApplication{}).Count(&stats.TotalApiApplications)
	c.db.Model(&models.ApiApplication{}).Where("status = ?", "active").Count(&stats.ActiveApiApplications)

	// API密钥统计
	c.db.Model(&models.ApiKey{}).Count(&stats.TotalApiKeys)
	c.db.Model(&models.ApiKey{}).
		Where("status = ? AND (expires_at IS NULL OR expires_at > ?)", "active", time.Now()).
		Count(&stats.ActiveApiKeys)

	// API接口统计
	c.db.Model(&models.ApiInterface{}).Count(&stats.TotalApiInterfaces)

	// API调用统计
	c.db.Model(&models.ApiUsageLog{}).Count(&stats.TotalApiCalls)
	today := time.Now().Format("2006-01-02")
	c.db.Model(&models.ApiUsageLog{}).
		Where("DATE(created_at) = ?", today).
		Count(&stats.TodayApiCalls)

	// 数据订阅统计
	c.db.Model(&models.DataSubscription{}).Count(&stats.TotalDataSubscriptions)
	c.db.Model(&models.DataSubscription{}).Where("status = ?", "active").Count(&stats.ActiveDataSubscriptions)

	// 访问申请统计
	c.db.Model(&models.DataAccessRequest{}).Count(&stats.TotalAccessRequests)
	c.db.Model(&models.DataAccessRequest{}).Where("status = ?", "pending").Count(&stats.PendingAccessRequests)
	c.db.Model(&models.DataAccessRequest{}).Where("status = ?", "approved").Count(&stats.ApprovedAccessRequests)

	// 热门应用 (前10个，按调用次数)
	c.db.Table("t_api_usage_log aul").
		Select("aul.application_id, aa.name as application_name, COUNT(*) as call_count, MAX(aul.created_at) as last_call_time").
		Joins("LEFT JOIN t_api_application aa ON aul.application_id = aa.id").
		Where("aul.application_id IS NOT NULL").
		Group("aul.application_id, aa.name").
		Order("call_count DESC").
		Limit(10).
		Find(&stats.TopApiApplications)

	// API调用趋势 (最近7天)
	c.db.Model(&models.ApiUsageLog{}).
		Select("DATE(created_at) as date, COUNT(*) as call_count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&stats.ApiCallTrendData)

	// 最近API使用日志 (前10条)
	c.db.Model(&models.ApiUsageLog{}).
		Select("id as log_id, api_path, method, status_code, response_time, created_at, request_ip").
		Order("created_at DESC").
		Limit(10).
		Find(&stats.RecentApiUsageLogs)

	// 响应时间统计
	c.db.Model(&models.ApiUsageLog{}).
		Select("AVG(response_time) as avg_response_time, MIN(response_time) as min_response_time, MAX(response_time) as max_response_time").
		Where("status_code < 500").
		Scan(&stats.ResponseTimeStats)

	// 错误率统计
	c.db.Model(&models.ApiUsageLog{}).Count(&stats.ErrorRateStats.TotalRequests)
	c.db.Model(&models.ApiUsageLog{}).
		Where("status_code >= 400").
		Count(&stats.ErrorRateStats.ErrorRequests)
	if stats.ErrorRateStats.TotalRequests > 0 {
		stats.ErrorRateStats.ErrorRate = float64(stats.ErrorRateStats.ErrorRequests) / float64(stats.ErrorRateStats.TotalRequests) * 100
	}

	return stats
}

// getSystemActivityStats 获取系统活动统计数据
func (c *DashboardController) getSystemActivityStats() SystemActivityStats {
	stats := SystemActivityStats{
		RecentSystemLogs:       []RecentSystemLog{},
		OperationTypeBreakdown: []OperationTypeCount{},
		UserActivityTrendData:  []UserActivityTrendData{},
	}

	// 系统日志统计
	c.db.Model(&models.SystemLog{}).Count(&stats.TotalOperations)
	today := time.Now().Format("2006-01-02")
	c.db.Model(&models.SystemLog{}).
		Where("DATE(operation_time) = ?", today).
		Count(&stats.TodayOperations)

	// 活跃用户统计 (基于系统日志中的操作者)
	c.db.Model(&models.SystemLog{}).
		Distinct("operator_id").
		Where("operator_id IS NOT NULL").
		Count(&stats.TotalUsers)

	c.db.Model(&models.SystemLog{}).
		Select("COUNT(DISTINCT operator_id)").
		Where("operator_id IS NOT NULL AND operation_time >= ?", time.Now().AddDate(0, 0, -30)).
		Scan(&stats.ActiveUsers)

	c.db.Model(&models.SystemLog{}).
		Select("COUNT(DISTINCT operator_id)").
		Where("operator_id IS NOT NULL AND DATE(operation_time) = ?", today).
		Scan(&stats.TodayActiveUsers)

	// 最近系统日志 (前10条)
	c.db.Model(&models.SystemLog{}).
		Select("id as log_id, operation_type, object_type, operator_name, operation_time, operation_result").
		Order("operation_time DESC").
		Limit(10).
		Find(&stats.RecentSystemLogs)

	// 操作类型分布
	c.db.Model(&models.SystemLog{}).
		Select("operation_type, COUNT(*) as count").
		Group("operation_type").
		Order("count DESC").
		Find(&stats.OperationTypeBreakdown)

	// 用户活动趋势 (最近7天)
	c.db.Model(&models.SystemLog{}).
		Select("DATE(operation_time) as date, COUNT(DISTINCT operator_id) as active_users, COUNT(*) as operations").
		Where("operation_time >= ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(operation_time)").
		Order("date ASC").
		Find(&stats.UserActivityTrendData)

	// 高峰使用时段 (按小时统计)
	var peakHour struct {
		Hour          int
		AvgOperations float64
	}
	c.db.Model(&models.SystemLog{}).
		Select("EXTRACT(HOUR FROM operation_time) as hour, COUNT(*) / COUNT(DISTINCT DATE(operation_time)) as avg_operations").
		Group("EXTRACT(HOUR FROM operation_time)").
		Order("avg_operations DESC").
		Limit(1).
		Scan(&peakHour)
	if peakHour.Hour >= 0 {
		stats.PeakUsageTime = &PeakUsageTimeInfo{
			Hour:          peakHour.Hour,
			AvgOperations: peakHour.AvgOperations,
		}
	}

	return stats
}

// GetBasicLibraryStats 单独获取基础库统计
// @Summary 获取基础库统计数据
// @Description 获取基础库的详细统计信息
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=BasicLibraryStats} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/basic-library-stats [get]
func (c *DashboardController) GetBasicLibraryStats(w http.ResponseWriter, r *http.Request) {
	stats := c.getBasicLibraryStats()
	render.JSON(w, r, SuccessResponse("获取基础库统计数据成功", stats))
}

// GetThematicLibraryStats 单独获取主题库统计
// @Summary 获取主题库统计数据
// @Description 获取主题库的详细统计信息
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=ThematicLibraryStats} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/thematic-library-stats [get]
func (c *DashboardController) GetThematicLibraryStats(w http.ResponseWriter, r *http.Request) {
	stats := c.getThematicLibraryStats()
	render.JSON(w, r, SuccessResponse("获取主题库统计数据成功", stats))
}

// GetSyncTaskStats 单独获取同步任务统计
// @Summary 获取同步任务统计数据
// @Description 获取同步任务的详细统计信息
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=SyncTaskStats} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/sync-task-stats [get]
func (c *DashboardController) GetSyncTaskStats(w http.ResponseWriter, r *http.Request) {
	stats := c.getSyncTaskStats()
	render.JSON(w, r, SuccessResponse("获取同步任务统计数据成功", stats))
}

// GetDataQualityStats 单独获取数据质量统计
// @Summary 获取数据质量统计数据
// @Description 获取数据质量的详细统计信息
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=DataQualityStats} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/data-quality-stats [get]
func (c *DashboardController) GetDataQualityStats(w http.ResponseWriter, r *http.Request) {
	stats := c.getDataQualityStats()
	render.JSON(w, r, SuccessResponse("获取数据质量统计数据成功", stats))
}

// GetDataSharingStats 单独获取数据共享统计
// @Summary 获取数据共享统计数据
// @Description 获取数据共享的详细统计信息
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=DataSharingStats} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/data-sharing-stats [get]
func (c *DashboardController) GetDataSharingStats(w http.ResponseWriter, r *http.Request) {
	stats := c.getDataSharingStats()
	render.JSON(w, r, SuccessResponse("获取数据共享统计数据成功", stats))
}

// GetSystemActivityStats 单独获取系统活动统计
// @Summary 获取系统活动统计数据
// @Description 获取系统活动的详细统计信息
// @Tags Dashboard
// @Produce json
// @Success 200 {object} APIResponse{data=SystemActivityStats} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /dashboard/system-activity-stats [get]
func (c *DashboardController) GetSystemActivityStats(w http.ResponseWriter, r *http.Request) {
	stats := c.getSystemActivityStats()
	render.JSON(w, r, SuccessResponse("获取系统活动统计数据成功", stats))
}
