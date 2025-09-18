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

// === 数据治理相关元数据接口 ===

// DataGovernanceMetadataResponse 数据治理完整元数据响应结构
type DataGovernanceMetadataResponse struct {
	// 质量规则相关
	QualityRuleTypes     []meta.QualityRuleType    `json:"quality_rule_types"`
	QualityCheckStatuses []meta.QualityCheckStatus `json:"quality_check_statuses"`

	// 脱敏和清洗相关
	MaskingTypes       []meta.DataMaskingType   `json:"masking_types"`
	CleansingRuleTypes []meta.CleansingRuleType `json:"cleansing_rule_types"`

	// 转换和校验相关
	TransformationRuleTypes []meta.TransformationRuleType `json:"transformation_rule_types"`
	ValidationRuleTypes     []meta.ValidationRuleType     `json:"validation_rule_types"`

	// 质量指标和报告相关
	QualityMetricTypes []meta.QualityMetricType `json:"quality_metric_types"`
	QualityReportTypes []meta.QualityReportType `json:"quality_report_types"`

	// 问题管理相关
	QualityIssueSeverities []meta.QualityIssueSeverity `json:"quality_issue_severities"`
	QualityIssueStatuses   []meta.QualityIssueStatus   `json:"quality_issue_statuses"`

	// 对象和任务相关
	ObjectTypes   []meta.DataGovernanceObjectType `json:"object_types"`
	MetadataTypes []meta.MetadataType             `json:"metadata_types"`
	TaskTypes     []meta.TaskType                 `json:"task_types"`
	TaskStatuses  []meta.TaskStatus               `json:"task_statuses"`

	// 血缘关系相关
	LineageRelationTypes []meta.LineageRelationType `json:"lineage_relation_types"`
}

// @Summary 获取数据治理完整元数据
// @Description 获取数据治理相关的所有元数据，包括质量规则类型、检查状态、脱敏类型、转换规则类型、校验规则类型、任务类型、血缘关系类型等
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=DataGovernanceMetadataResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /meta/data-governance [get]
func (c *MetaController) GetDataGovernanceMetadata(w http.ResponseWriter, r *http.Request) {
	response := DataGovernanceMetadataResponse{
		// 质量规则相关
		QualityRuleTypes:     meta.GetQualityRuleTypes(),
		QualityCheckStatuses: meta.GetQualityCheckStatuses(),

		// 脱敏和清洗相关
		MaskingTypes:       meta.GetDataMaskingTypes(),
		CleansingRuleTypes: meta.GetCleansingRuleTypes(),

		// 转换和校验相关
		TransformationRuleTypes: meta.GetTransformationRuleTypes(),
		ValidationRuleTypes:     meta.GetValidationRuleTypes(),

		// 质量指标和报告相关
		QualityMetricTypes: meta.GetQualityMetricTypes(),
		QualityReportTypes: meta.GetQualityReportTypes(),

		// 问题管理相关
		QualityIssueSeverities: meta.GetQualityIssueSeverities(),
		QualityIssueStatuses:   meta.GetQualityIssueStatuses(),

		// 对象和任务相关
		ObjectTypes:   meta.GetDataGovernanceObjectTypes(),
		MetadataTypes: meta.GetMetadataTypes(),
		TaskTypes:     meta.GetTaskTypes(),
		TaskStatuses:  meta.GetTaskStatuses(),

		// 血缘关系相关
		LineageRelationTypes: meta.GetLineageRelationTypes(),
	}

	render.JSON(w, r, SuccessResponse("获取数据治理完整元数据成功", response))
}
