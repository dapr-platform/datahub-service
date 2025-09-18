package controllers

import (
	"datahub-service/service/meta"
	"net/http"

	"github.com/go-chi/render"
)

type MetaController struct {
}

func NewMetaController() *MetaController {
	return &MetaController{}
}

// @Summary 获取所有数据源类型元数据
// @Description 获取所有数据源类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]meta.DataSourceTypeDefinition}
// @Failure 500 {object} APIResponse
// @Router /meta/basic-libraries/data-sources [get]
func (c *MetaController) GetDataSourceTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据源类型元数据成功", meta.DataSourceTypes))
}

// @Summary 获取所有数据接口配置元数据
// @Description 获取所有数据接口配置元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]meta.DataInterfaceConfigDefinition}
// @Failure 500 {object} APIResponse
// @Router /meta/basic-libraries/data-interface-configs [get]
func (c *MetaController) GetDataInterfaceConfigs(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据接口配置元数据成功", meta.DataInterfaceConfigDefinitions))
}

// @Summary 获取所有同步任务元数据
// @Description 获取所有同步任务相关元数据，包括任务类型、状态、调度类型等
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}}
// @Failure 500 {object} APIResponse
// @Router /meta/sync-tasks [get]
func (c *MetaController) GetSyncTaskMeta(w http.ResponseWriter, r *http.Request) {
	syncTaskMeta := map[string]interface{}{
		"task_types":       meta.SyncTaskMetas["sync_task_types"],
		"task_statuses":    meta.SyncTaskMetas["sync_task_statuses"],
		"schedule_types":   meta.SyncTaskMetas["sync_task_schedule_types"],
		"event_types":      meta.SyncTaskMetas["sync_event_types"],
		"execute_types":    meta.SyncTaskMetas["sync_execute_types"],
		"sync_strategies":  meta.SyncTaskMetas["sync_strategies"],
		"schedule_configs": meta.SyncTaskScheduleDefinitions,
	}
	render.JSON(w, r, SuccessResponse("获取同步任务元数据成功", syncTaskMeta))
}

// @Summary 获取所有数据主题库分类元数据
// @Description 获取所有数据主题库分类元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryCategory}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/categories [get]
func (c *MetaController) GetThematicLibraryCategories(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据主题库分类元数据成功", meta.ThematicLibraryCategories))
}

// @Summary 获取所有数据主题库域元数据
// @Description 获取所有数据主题库域元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryDomain}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/domains [get]
func (c *MetaController) GetThematicLibraryDomains(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据主题库域元数据成功", meta.ThematicLibraryDomains))
}

// @Summary 获取所有数据主题库访问级别元数据
// @Description 获取所有数据主题库访问级别元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryAccessLevel}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/access-levels [get]
func (c *MetaController) GetThematicLibraryAccessLevels(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据主题库访问级别元数据成功", meta.ThematicLibraryAccessLevels))
}

// @Summary 获取所有主题库同步任务元数据
// @Description 获取所有主题库同步任务相关元数据，包括任务状态、触发类型、执行状态等
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-sync-tasks [get]
func (c *MetaController) GetThematicSyncTaskMeta(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取主题库同步任务元数据成功", meta.ThematicSyncMetas))
}

// @Summary 获取主题库同步配置定义
// @Description 获取主题库同步各种配置的字段定义，用于前端动态生成配置表单
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]meta.ThematicSyncConfigDefinition}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-sync-configs [get]
func (c *MetaController) GetThematicSyncConfigDefinitions(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取主题库同步配置定义成功", meta.ThematicSyncConfigDefinitions))
}

// @Summary 获取主题库状态元数据
// @Description 获取主题库状态元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryStatus}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/statuses [get]
func (c *MetaController) GetThematicLibraryStatuses(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取主题库状态元数据成功", meta.GetThematicLibraryStatuses()))
}

// @Summary 获取主题接口类型元数据
// @Description 获取主题接口类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicInterfaceType}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/interface-types [get]
func (c *MetaController) GetThematicInterfaceTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取主题接口类型元数据成功", meta.GetThematicInterfaceTypes()))
}

// @Summary 获取主题接口状态元数据
// @Description 获取主题接口状态元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicInterfaceStatus}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/interface-statuses [get]
func (c *MetaController) GetThematicInterfaceStatuses(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取主题接口状态元数据成功", meta.GetThematicInterfaceStatuses()))
}

// ThematicLibraryMetadataResponse 主题库完整元数据响应结构
type ThematicLibraryMetadataResponse struct {
	Categories        []meta.ThematicLibraryCategory    `json:"categories"`
	Domains           []meta.ThematicLibraryDomain      `json:"domains"`
	AccessLevels      []meta.ThematicLibraryAccessLevel `json:"access_levels"`
	LibraryStatuses   []meta.ThematicLibraryStatus      `json:"library_statuses"`
	InterfaceTypes    []meta.ThematicInterfaceType      `json:"interface_types"`
	InterfaceStatuses []meta.ThematicInterfaceStatus    `json:"interface_statuses"`
}

// @Summary 获取主题库完整元数据
// @Description 获取主题库相关的所有元数据，包括分类、数据域、访问级别、状态等
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=ThematicLibraryMetadataResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /meta/thematic-libraries/all [get]
func (c *MetaController) GetThematicLibraryAllMetadata(w http.ResponseWriter, r *http.Request) {
	response := ThematicLibraryMetadataResponse{
		Categories:        meta.GetThematicLibraryCategories(),
		Domains:           meta.GetThematicLibraryDomains(),
		AccessLevels:      meta.GetThematicLibraryAccessLevels(),
		LibraryStatuses:   meta.GetThematicLibraryStatuses(),
		InterfaceTypes:    meta.GetThematicInterfaceTypes(),
		InterfaceStatuses: meta.GetThematicInterfaceStatuses(),
	}

	render.JSON(w, r, SuccessResponse("获取主题库完整元数据成功", response))
}

// === 数据质量相关元数据接口 ===

// @Summary 获取质量规则类型元数据
// @Description 获取所有质量规则类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.QualityRuleType}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/rule-types [get]
func (c *MetaController) GetQualityRuleTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取质量规则类型元数据成功", meta.GetQualityRuleTypes()))
}

// @Summary 获取质量检查状态元数据
// @Description 获取所有质量检查状态元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.QualityCheckStatus}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/check-statuses [get]
func (c *MetaController) GetQualityCheckStatuses(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取质量检查状态元数据成功", meta.GetQualityCheckStatuses()))
}

// @Summary 获取数据脱敏类型元数据
// @Description 获取所有数据脱敏类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.DataMaskingType}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/masking-types [get]
func (c *MetaController) GetDataMaskingTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据脱敏类型元数据成功", meta.GetDataMaskingTypes()))
}

// @Summary 获取清洗规则类型元数据
// @Description 获取所有清洗规则类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.CleansingRuleType}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/cleansing-rule-types [get]
func (c *MetaController) GetCleansingRuleTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取清洗规则类型元数据成功", meta.GetCleansingRuleTypes()))
}

// @Summary 获取质量指标类型元数据
// @Description 获取所有质量指标类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.QualityMetricType}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/metric-types [get]
func (c *MetaController) GetQualityMetricTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取质量指标类型元数据成功", meta.GetQualityMetricTypes()))
}

// @Summary 获取质量报告类型元数据
// @Description 获取所有质量报告类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.QualityReportType}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/report-types [get]
func (c *MetaController) GetQualityReportTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取质量报告类型元数据成功", meta.GetQualityReportTypes()))
}

// @Summary 获取质量问题严重程度元数据
// @Description 获取所有质量问题严重程度元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.QualityIssueSeverity}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/issue-severities [get]
func (c *MetaController) GetQualityIssueSeverities(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取质量问题严重程度元数据成功", meta.GetQualityIssueSeverities()))
}

// @Summary 获取质量问题状态元数据
// @Description 获取所有质量问题状态元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.QualityIssueStatus}
// @Failure 500 {object} APIResponse
// @Router /meta/data-quality/issue-statuses [get]
func (c *MetaController) GetQualityIssueStatuses(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取质量问题状态元数据成功", meta.GetQualityIssueStatuses()))
}

// DataQualityMetadataResponse 数据质量完整元数据响应结构
type DataQualityMetadataResponse struct {
	RuleTypes       []meta.QualityRuleType      `json:"rule_types"`
	CheckStatuses   []meta.QualityCheckStatus   `json:"check_statuses"`
	MaskingTypes    []meta.DataMaskingType      `json:"masking_types"`
	CleansingTypes  []meta.CleansingRuleType    `json:"cleansing_types"`
	MetricTypes     []meta.QualityMetricType    `json:"metric_types"`
	ReportTypes     []meta.QualityReportType    `json:"report_types"`
	IssueSeverities []meta.QualityIssueSeverity `json:"issue_severities"`
	IssueStatuses   []meta.QualityIssueStatus   `json:"issue_statuses"`
}

// @Summary 获取数据质量完整元数据
// @Description 获取数据质量相关的所有元数据，包括规则类型、检查状态、脱敏类型等
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=DataQualityMetadataResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /meta/data-quality/all [get]
func (c *MetaController) GetDataQualityAllMetadata(w http.ResponseWriter, r *http.Request) {
	response := DataQualityMetadataResponse{
		RuleTypes:       meta.GetQualityRuleTypes(),
		CheckStatuses:   meta.GetQualityCheckStatuses(),
		MaskingTypes:    meta.GetDataMaskingTypes(),
		CleansingTypes:  meta.GetCleansingRuleTypes(),
		MetricTypes:     meta.GetQualityMetricTypes(),
		ReportTypes:     meta.GetQualityReportTypes(),
		IssueSeverities: meta.GetQualityIssueSeverities(),
		IssueStatuses:   meta.GetQualityIssueStatuses(),
	}

	render.JSON(w, r, SuccessResponse("获取数据质量完整元数据成功", response))
}
