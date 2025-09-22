/*
 * @module api/controllers/data_quality_controller
 * @description 数据质量统一控制器，提供数据质量管理、元数据管理、数据脱敏等完整功能
 * @architecture 分层架构 - 控制器层
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow HTTP请求处理流程
 * @rules 统一的错误处理和响应格式，强类型API定义
 * @dependencies datahub-service/service/governance, github.com/go-chi/chi/v5
 * @refs service/models/governance.go, service/models/quality_models.go
 */

package controllers

import (
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// DataQualityController 数据质量统一控制器
type DataQualityController struct {
	governanceService *governance.GovernanceService
}

// NewDataQualityController 创建数据质量控制器实例
func NewDataQualityController(governanceService *governance.GovernanceService) *DataQualityController {
	return &DataQualityController{
		governanceService: governanceService,
	}
}

// === 数据质量规则管理 ===

// CreateQualityRule 创建数据质量规则
// @Summary 创建数据质量规则
// @Description 创建新的数据质量规则
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param rule body governance.CreateQualityRuleRequest true "数据质量规则信息"
// @Success 201 {object} APIResponse{data=governance.QualityRuleResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/rules [post]
func (c *DataQualityController) CreateQualityRule(w http.ResponseWriter, r *http.Request) {
	var req governance.CreateQualityRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	rule := &models.QualityRuleTemplate{
		Name:          req.Name,
		Type:          req.Type,
		Category:      req.Category,
		Description:   req.Description,
		RuleLogic:     req.RuleLogic,
		Parameters:    req.Parameters,
		DefaultConfig: req.DefaultConfig,
		IsEnabled:     req.IsEnabled,
		Tags:          req.Tags,
	}

	if err := c.governanceService.CreateQualityRule(rule); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据质量规则失败", err))
		return
	}

	response := &governance.QualityRuleResponse{
		ID:            rule.ID,
		Name:          rule.Name,
		Type:          rule.Type,
		Category:      rule.Category,
		Description:   rule.Description,
		RuleLogic:     rule.RuleLogic,
		Parameters:    rule.Parameters,
		DefaultConfig: rule.DefaultConfig,
		IsBuiltIn:     rule.IsBuiltIn,
		IsEnabled:     rule.IsEnabled,
		Version:       rule.Version,
		Tags:          rule.Tags,
		CreatedAt:     rule.CreatedAt,
		CreatedBy:     rule.CreatedBy,
		UpdatedAt:     rule.UpdatedAt,
		UpdatedBy:     rule.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("创建数据质量规则成功", response))
}

// GetQualityRules 获取数据质量规则列表
// @Summary 获取数据质量规则列表
// @Description 分页获取数据质量规则列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param type query string false "规则类型" Enums(completeness,accuracy,consistency,validity,uniqueness,timeliness,standardization)
// @Param object_type query string false "关联对象类型" Enums(interface,thematic_interface)
// @Success 200 {object} APIResponse{data=governance.QualityRuleListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/rules [get]
func (c *DataQualityController) GetQualityRules(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	ruleType := r.URL.Query().Get("type")
	objectType := r.URL.Query().Get("object_type")

	rules, total, err := c.governanceService.GetQualityRules(page, size, ruleType, objectType)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据质量规则列表失败", err))
		return
	}

	var ruleResponses []governance.QualityRuleResponse
	for _, rule := range rules {
		ruleResponses = append(ruleResponses, governance.QualityRuleResponse{
			ID:            rule.ID,
			Name:          rule.Name,
			Type:          rule.Type,
			Category:      rule.Category,
			Description:   rule.Description,
			RuleLogic:     rule.RuleLogic,
			Parameters:    rule.Parameters,
			DefaultConfig: rule.DefaultConfig,
			IsBuiltIn:     rule.IsBuiltIn,
			IsEnabled:     rule.IsEnabled,
			Version:       rule.Version,
			Tags:          rule.Tags,
			CreatedAt:     rule.CreatedAt,
			CreatedBy:     rule.CreatedBy,
			UpdatedAt:     rule.UpdatedAt,
			UpdatedBy:     rule.UpdatedBy,
		})
	}

	response := governance.QualityRuleListResponse{
		List:  ruleResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量规则列表成功", response))
}

// GetQualityRuleByID 根据ID获取数据质量规则
// @Summary 根据ID获取数据质量规则
// @Description 根据ID获取数据质量规则详情
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse{data=governance.QualityRuleResponse} "获取成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/rules/{id} [get]
func (c *DataQualityController) GetQualityRuleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rule, err := c.governanceService.GetQualityRuleByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据质量规则不存在", err))
		return
	}

	response := &governance.QualityRuleResponse{
		ID:            rule.ID,
		Name:          rule.Name,
		Type:          rule.Type,
		Category:      rule.Category,
		Description:   rule.Description,
		RuleLogic:     rule.RuleLogic,
		Parameters:    rule.Parameters,
		DefaultConfig: rule.DefaultConfig,
		IsBuiltIn:     rule.IsBuiltIn,
		IsEnabled:     rule.IsEnabled,
		Version:       rule.Version,
		Tags:          rule.Tags,
		CreatedAt:     rule.CreatedAt,
		CreatedBy:     rule.CreatedBy,
		UpdatedAt:     rule.UpdatedAt,
		UpdatedBy:     rule.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量规则成功", response))
}

// UpdateQualityRule 更新数据质量规则
// @Summary 更新数据质量规则
// @Description 更新数据质量规则信息
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param updates body governance.UpdateQualityRuleRequest true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/rules/{id} [put]
func (c *DataQualityController) UpdateQualityRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req governance.UpdateQualityRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.RuleLogic != nil {
		updates["rule_logic"] = req.RuleLogic
	}
	if req.Parameters != nil {
		updates["parameters"] = req.Parameters
	}
	if req.DefaultConfig != nil {
		updates["default_config"] = req.DefaultConfig
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := c.governanceService.UpdateQualityRule(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新数据质量规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新数据质量规则成功", nil))
}

// DeleteQualityRule 删除数据质量规则
// @Summary 删除数据质量规则
// @Description 删除指定的数据质量规则
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/rules/{id} [delete]
func (c *DataQualityController) DeleteQualityRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteQualityRule(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据质量规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据质量规则成功", nil))
}

// === 数据脱敏规则管理 ===

// CreateMaskingRule 创建数据脱敏规则
// @Summary 创建数据脱敏规则
// @Description 创建新的数据脱敏规则
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param rule body governance.CreateMaskingRuleRequest true "数据脱敏规则信息"
// @Success 201 {object} APIResponse{data=governance.MaskingRuleResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/masking-rules [post]
func (c *DataQualityController) CreateMaskingRule(w http.ResponseWriter, r *http.Request) {
	var req governance.CreateMaskingRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	rule := &models.DataMaskingTemplate{
		Name:          req.Name,
		MaskingType:   req.MaskingType,
		Category:      req.Category,
		SecurityLevel: req.SecurityLevel,
		Description:   req.Description,
		MaskingLogic:  req.MaskingLogic,
		Parameters:    req.Parameters,
		IsEnabled:     req.IsEnabled,
		Tags:          req.Tags,
	}

	if err := c.governanceService.CreateMaskingRule(rule); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据脱敏规则失败", err))
		return
	}

	response := &governance.MaskingRuleResponse{
		ID:            rule.ID,
		Name:          rule.Name,
		Category:      rule.Category,
		SecurityLevel: rule.SecurityLevel,
		MaskingType:   rule.MaskingType,
		Description:   rule.Description,
		MaskingLogic:  rule.MaskingLogic,
		Parameters:    rule.Parameters,
		IsBuiltIn:     rule.IsBuiltIn,
		Version:       rule.Version,
		Tags:          rule.Tags,
		IsEnabled:     rule.IsEnabled,
		CreatedAt:     rule.CreatedAt,
		CreatedBy:     rule.CreatedBy,
		UpdatedAt:     rule.UpdatedAt,
		UpdatedBy:     rule.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("创建数据脱敏规则成功", response))
}

// GetMaskingRules 获取数据脱敏规则列表
// @Summary 获取数据脱敏规则列表
// @Description 分页获取数据脱敏规则列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param data_source query string false "数据源"
// @Param masking_type query string false "脱敏类型" Enums(mask,replace,encrypt,pseudonymize)
// @Success 200 {object} APIResponse{data=governance.MaskingRuleListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/masking-rules [get]
func (c *DataQualityController) GetMaskingRules(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 10
	}

	dataSource := r.URL.Query().Get("data_source")
	maskingType := r.URL.Query().Get("masking_type")

	rules, total, err := c.governanceService.GetMaskingRules(page, pageSize, dataSource, maskingType)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据脱敏规则列表失败", err))
		return
	}

	var ruleResponses []governance.MaskingRuleResponse
	for _, rule := range rules {
		ruleResponses = append(ruleResponses, governance.MaskingRuleResponse{
			ID:            rule.ID,
			Name:          rule.Name,
			Category:      rule.Category,
			SecurityLevel: rule.SecurityLevel,
			MaskingType:   rule.MaskingType,
			Description:   rule.Description,
			MaskingLogic:  rule.MaskingLogic,
			Parameters:    rule.Parameters,
			IsBuiltIn:     rule.IsBuiltIn,
			Version:       rule.Version,
			Tags:          rule.Tags,
			IsEnabled:     rule.IsEnabled,
			CreatedAt:     rule.CreatedAt,
			CreatedBy:     rule.CreatedBy,
			UpdatedAt:     rule.UpdatedAt,
			UpdatedBy:     rule.UpdatedBy,
		})
	}

	response := governance.MaskingRuleListResponse{
		List:  ruleResponses,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}

	render.JSON(w, r, SuccessResponse("获取数据脱敏规则列表成功", response))
}

// GetMaskingRuleByID 根据ID获取数据脱敏规则
// @Summary 根据ID获取数据脱敏规则
// @Description 根据ID获取数据脱敏规则详情
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse{data=governance.MaskingRuleResponse} "获取成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/masking-rules/{id} [get]
func (c *DataQualityController) GetMaskingRuleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rule, err := c.governanceService.GetMaskingRuleByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据脱敏规则不存在", err))
		return
	}

	response := &governance.MaskingRuleResponse{
		ID:            rule.ID,
		Name:          rule.Name,
		Category:      rule.Category,
		SecurityLevel: rule.SecurityLevel,
		MaskingType:   rule.MaskingType,
		Description:   rule.Description,
		MaskingLogic:  rule.MaskingLogic,
		Parameters:    rule.Parameters,
		IsBuiltIn:     rule.IsBuiltIn,
		Version:       rule.Version,
		Tags:          rule.Tags,
		IsEnabled:     rule.IsEnabled,
		CreatedAt:     rule.CreatedAt,
		CreatedBy:     rule.CreatedBy,
		UpdatedAt:     rule.UpdatedAt,
		UpdatedBy:     rule.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("获取数据脱敏规则成功", response))
}

// UpdateMaskingRule 更新数据脱敏规则
// @Summary 更新数据脱敏规则
// @Description 更新数据脱敏规则信息
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param updates body governance.UpdateMaskingRuleRequest true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/masking-rules/{id} [put]
func (c *DataQualityController) UpdateMaskingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req governance.UpdateMaskingRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.MaskingLogic != nil {
		updates["masking_logic"] = req.MaskingLogic
	}
	if req.Parameters != nil {
		updates["parameters"] = req.Parameters
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := c.governanceService.UpdateMaskingRule(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新数据脱敏规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新数据脱敏规则成功", nil))
}

// DeleteMaskingRule 删除数据脱敏规则
// @Summary 删除数据脱敏规则
// @Description 删除指定的数据脱敏规则
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/masking-rules/{id} [delete]
func (c *DataQualityController) DeleteMaskingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteMaskingRule(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据脱敏规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据脱敏规则成功", nil))
}

// === 质量检查执行 ===

// RunQualityCheck 执行数据质量检查
// @Summary 执行数据质量检查
// @Description 对指定对象执行数据质量检查并生成报告
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param request body governance.RunQualityCheckRequest true "质量检查请求"
// @Success 200 {object} APIResponse{data=governance.QualityReportResponse} "检查成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/checks [post]
func (c *DataQualityController) RunQualityCheck(w http.ResponseWriter, r *http.Request) {
	var req governance.RunQualityCheckRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	report, err := c.governanceService.RunQualityCheck(req.ObjectID, req.ObjectType)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("执行数据质量检查失败", err))
		return
	}

	response := &governance.QualityReportResponse{
		ID:                report.ID,
		ReportName:        report.ReportName,
		RelatedObjectID:   report.RelatedObjectID,
		RelatedObjectType: report.RelatedObjectType,
		QualityScore:      report.QualityScore,
		QualityMetrics:    report.QualityMetrics,
		Issues:            report.Issues,
		Recommendations:   report.Recommendations,
		GeneratedAt:       report.GeneratedAt,
		GeneratedBy:       report.GeneratedBy,
	}

	render.JSON(w, r, SuccessResponse("执行数据质量检查成功", response))
}

// === 质量报告管理 ===

// GetQualityReports 获取数据质量报告列表
// @Summary 获取数据质量报告列表
// @Description 分页获取数据质量报告列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param object_type query string false "对象类型" Enums(interface,thematic_interface)
// @Success 200 {object} APIResponse{data=governance.QualityReportListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/reports [get]
func (c *DataQualityController) GetQualityReports(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 10
	}

	objectType := r.URL.Query().Get("object_type")

	reports, total, err := c.governanceService.GetQualityReports(page, pageSize, objectType)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据质量报告列表失败", err))
		return
	}

	var reportResponses []governance.QualityReportResponse
	for _, report := range reports {
		reportResponses = append(reportResponses, governance.QualityReportResponse{
			ID:                report.ID,
			ReportName:        report.ReportName,
			RelatedObjectID:   report.RelatedObjectID,
			RelatedObjectType: report.RelatedObjectType,
			QualityScore:      report.QualityScore,
			QualityMetrics:    report.QualityMetrics,
			Issues:            report.Issues,
			Recommendations:   report.Recommendations,
			GeneratedAt:       report.GeneratedAt,
			GeneratedBy:       report.GeneratedBy,
		})
	}

	response := governance.QualityReportListResponse{
		List:  reportResponses,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量报告列表成功", response))
}

// GetQualityReportByID 根据ID获取数据质量报告
// @Summary 根据ID获取数据质量报告
// @Description 根据ID获取数据质量报告详情
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "报告ID"
// @Success 200 {object} APIResponse{data=governance.QualityReportResponse} "获取成功"
// @Failure 404 {object} APIResponse "报告不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/reports/{id} [get]
func (c *DataQualityController) GetQualityReportByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	report, err := c.governanceService.GetQualityReportByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据质量报告不存在", err))
		return
	}

	response := &governance.QualityReportResponse{
		ID:                report.ID,
		ReportName:        report.ReportName,
		RelatedObjectID:   report.RelatedObjectID,
		RelatedObjectType: report.RelatedObjectType,
		QualityScore:      report.QualityScore,
		QualityMetrics:    report.QualityMetrics,
		Issues:            report.Issues,
		Recommendations:   report.Recommendations,
		GeneratedAt:       report.GeneratedAt,
		GeneratedBy:       report.GeneratedBy,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量报告成功", response))
}

// === 元数据管理 ===

// CreateMetadata 创建元数据
// @Summary 创建元数据
// @Description 创建新的元数据
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param metadata body governance.CreateMetadataRequest true "元数据信息"
// @Success 201 {object} APIResponse{data=governance.MetadataResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/metadata [post]
func (c *DataQualityController) CreateMetadata(w http.ResponseWriter, r *http.Request) {
	var req governance.CreateMetadataRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	metadata := &models.Metadata{
		Type:              req.Type,
		Name:              req.Name,
		Content:           req.Content,
		RelatedObjectID:   &req.RelatedObjectID,
		RelatedObjectType: &req.RelatedObjectType,
	}

	if err := c.governanceService.CreateMetadata(metadata); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建元数据失败", err))
		return
	}

	response := &governance.MetadataResponse{
		ID:                metadata.ID,
		Type:              metadata.Type,
		Name:              metadata.Name,
		Content:           metadata.Content,
		RelatedObjectID:   *metadata.RelatedObjectID,
		RelatedObjectType: *metadata.RelatedObjectType,
		CreatedAt:         metadata.CreatedAt,
		CreatedBy:         metadata.CreatedBy,
		UpdatedAt:         metadata.UpdatedAt,
		UpdatedBy:         metadata.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("创建元数据成功", response))
}

// GetMetadataList 获取元数据列表
// @Summary 获取元数据列表
// @Description 分页获取元数据列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param type query string false "元数据类型" Enums(technical,business,management)
// @Param name query string false "元数据名称"
// @Success 200 {object} APIResponse{data=governance.MetadataListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/metadata [get]
func (c *DataQualityController) GetMetadataList(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	metadataType := r.URL.Query().Get("type")
	name := r.URL.Query().Get("name")

	metadataList, total, err := c.governanceService.GetMetadataList(page, size, metadataType, name)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取元数据列表失败", err))
		return
	}

	var metadataResponses []governance.MetadataResponse
	for _, metadata := range metadataList {
		var relatedObjectID, relatedObjectType string
		if metadata.RelatedObjectID != nil {
			relatedObjectID = *metadata.RelatedObjectID
		}
		if metadata.RelatedObjectType != nil {
			relatedObjectType = *metadata.RelatedObjectType
		}

		metadataResponses = append(metadataResponses, governance.MetadataResponse{
			ID:                metadata.ID,
			Type:              metadata.Type,
			Name:              metadata.Name,
			Content:           metadata.Content,
			RelatedObjectID:   relatedObjectID,
			RelatedObjectType: relatedObjectType,
			CreatedAt:         metadata.CreatedAt,
			CreatedBy:         metadata.CreatedBy,
			UpdatedAt:         metadata.UpdatedAt,
			UpdatedBy:         metadata.UpdatedBy,
		})
	}

	response := governance.MetadataListResponse{
		List:  metadataResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取元数据列表成功", response))
}

// GetMetadataByID 根据ID获取元数据
// @Summary 根据ID获取元数据
// @Description 根据ID获取元数据详情
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "元数据ID"
// @Success 200 {object} APIResponse{data=governance.MetadataResponse} "获取成功"
// @Failure 404 {object} APIResponse "元数据不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/metadata/{id} [get]
func (c *DataQualityController) GetMetadataByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	metadata, err := c.governanceService.GetMetadataByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("元数据不存在", err))
		return
	}

	var relatedObjectID, relatedObjectType string
	if metadata.RelatedObjectID != nil {
		relatedObjectID = *metadata.RelatedObjectID
	}
	if metadata.RelatedObjectType != nil {
		relatedObjectType = *metadata.RelatedObjectType
	}

	response := &governance.MetadataResponse{
		ID:                metadata.ID,
		Type:              metadata.Type,
		Name:              metadata.Name,
		Content:           metadata.Content,
		RelatedObjectID:   relatedObjectID,
		RelatedObjectType: relatedObjectType,
		CreatedAt:         metadata.CreatedAt,
		CreatedBy:         metadata.CreatedBy,
		UpdatedAt:         metadata.UpdatedAt,
		UpdatedBy:         metadata.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("获取元数据成功", response))
}

// UpdateMetadata 更新元数据
// @Summary 更新元数据
// @Description 更新元数据信息
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "元数据ID"
// @Param updates body governance.UpdateMetadataRequest true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "元数据不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/metadata/{id} [put]
func (c *DataQualityController) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req governance.UpdateMetadataRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Content != nil {
		updates["content"] = req.Content
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := c.governanceService.UpdateMetadata(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新元数据失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新元数据成功", nil))
}

// DeleteMetadata 删除元数据
// @Summary 删除元数据
// @Description 删除指定的元数据
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "元数据ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "元数据不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/metadata/{id} [delete]
func (c *DataQualityController) DeleteMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteMetadata(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除元数据失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除元数据成功", nil))
}

// === 清洗规则管理 ===

// CreateCleansingRule 创建数据清洗规则
// @Summary 创建数据清洗规则
// @Description 创建新的数据清洗规则
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param rule body governance.CreateCleansingRuleRequest true "数据清洗规则信息"
// @Success 201 {object} APIResponse{data=governance.CleansingRuleResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/cleansing-rules [post]
func (c *DataQualityController) CreateCleansingRule(w http.ResponseWriter, r *http.Request) {
	var req governance.CreateCleansingRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	rule := &models.DataCleansingTemplate{
		Name:            req.Name,
		Description:     req.Description,
		RuleType:        req.RuleType,
		Category:        req.Category,
		CleansingLogic:  models.JSONB(req.CleansingLogic),
		Parameters:      models.JSONB(req.Parameters),
		DefaultConfig:   models.JSONB(req.DefaultConfig),
		ComplexityLevel: req.ComplexityLevel,
		IsEnabled:       req.IsEnabled,
		Tags:            models.JSONB(req.Tags),
	}

	if err := c.governanceService.CreateCleansingRule(rule); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据清洗规则失败", err))
		return
	}

	response := &governance.CleansingRuleResponse{
		ID:              rule.ID,
		Name:            rule.Name,
		Description:     rule.Description,
		RuleType:        rule.RuleType,
		Category:        rule.Category,
		CleansingLogic:  rule.CleansingLogic,
		Parameters:      rule.Parameters,
		DefaultConfig:   rule.DefaultConfig,
		ComplexityLevel: rule.ComplexityLevel,
		IsBuiltIn:       rule.IsBuiltIn,
		IsEnabled:       rule.IsEnabled,
		Version:         rule.Version,
		Tags:            rule.Tags,
		CreatedAt:       rule.CreatedAt,
		CreatedBy:       rule.CreatedBy,
		UpdatedAt:       rule.UpdatedAt,
		UpdatedBy:       rule.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("创建数据清洗规则成功", response))
}

// GetCleansingRules 获取数据清洗规则列表
// @Summary 获取数据清洗规则列表
// @Description 分页获取数据清洗规则列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param rule_type query string false "规则类型" Enums(standardization,deduplication,validation,transformation,enrichment)
// @Param target_table query string false "目标表"
// @Success 200 {object} APIResponse{data=governance.CleansingRuleListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/cleansing-rules [get]
func (c *DataQualityController) GetCleansingRules(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	ruleType := r.URL.Query().Get("rule_type")
	targetTable := r.URL.Query().Get("target_table")

	rules, total, err := c.governanceService.GetCleansingRules(page, size, ruleType, targetTable)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据清洗规则列表失败", err))
		return
	}

	var ruleResponses []governance.CleansingRuleResponse
	for _, rule := range rules {
		ruleResponses = append(ruleResponses, governance.CleansingRuleResponse{
			ID:              rule.ID,
			Name:            rule.Name,
			Description:     rule.Description,
			RuleType:        rule.RuleType,
			Category:        rule.Category,
			CleansingLogic:  rule.CleansingLogic,
			Parameters:      rule.Parameters,
			DefaultConfig:   rule.DefaultConfig,
			ComplexityLevel: rule.ComplexityLevel,
			IsBuiltIn:       rule.IsBuiltIn,
			IsEnabled:       rule.IsEnabled,
			Version:         rule.Version,
			Tags:            rule.Tags,
			CreatedAt:       rule.CreatedAt,
			CreatedBy:       rule.CreatedBy,
			UpdatedAt:       rule.UpdatedAt,
			UpdatedBy:       rule.UpdatedBy,
		})
	}

	response := governance.CleansingRuleListResponse{
		List:  ruleResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据清洗规则列表成功", response))
}

// GetCleansingRuleByID 根据ID获取数据清洗规则
// @Summary 根据ID获取数据清洗规则
// @Description 根据ID获取数据清洗规则详情
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse{data=governance.CleansingRuleResponse} "获取成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/cleansing-rules/{id} [get]
func (c *DataQualityController) GetCleansingRuleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rule, err := c.governanceService.GetCleansingRuleByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据清洗规则不存在", err))
		return
	}

	response := &governance.CleansingRuleResponse{
		ID:              rule.ID,
		Name:            rule.Name,
		Description:     rule.Description,
		RuleType:        rule.RuleType,
		Category:        rule.Category,
		CleansingLogic:  rule.CleansingLogic,
		Parameters:      rule.Parameters,
		DefaultConfig:   rule.DefaultConfig,
		ComplexityLevel: rule.ComplexityLevel,
		IsBuiltIn:       rule.IsBuiltIn,
		IsEnabled:       rule.IsEnabled,
		Version:         rule.Version,
		Tags:            rule.Tags,
		CreatedAt:       rule.CreatedAt,
		CreatedBy:       rule.CreatedBy,
		UpdatedAt:       rule.UpdatedAt,
		UpdatedBy:       rule.UpdatedBy,
	}

	render.JSON(w, r, SuccessResponse("获取数据清洗规则成功", response))
}

// UpdateCleansingRule 更新数据清洗规则
// @Summary 更新数据清洗规则
// @Description 更新数据清洗规则信息
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param updates body governance.UpdateCleansingRuleRequest true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/cleansing-rules/{id} [put]
func (c *DataQualityController) UpdateCleansingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req governance.UpdateCleansingRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.CleansingLogic != nil {
		updates["cleansing_logic"] = models.JSONB(req.CleansingLogic)
	}
	if req.Parameters != nil {
		updates["parameters"] = models.JSONB(req.Parameters)
	}
	if req.DefaultConfig != nil {
		updates["default_config"] = models.JSONB(req.DefaultConfig)
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	if err := c.governanceService.UpdateCleansingRule(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新数据清洗规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新数据清洗规则成功", nil))
}

// DeleteCleansingRule 删除数据清洗规则
// @Summary 删除数据清洗规则
// @Description 删除指定的数据清洗规则
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/cleansing-rules/{id} [delete]
func (c *DataQualityController) DeleteCleansingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteCleansingRule(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据清洗规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据清洗规则成功", nil))
}

// === 数据质量检测任务管理 ===

// CreateQualityTask 创建数据质量检测任务
// @Summary 创建数据质量检测任务
// @Description 创建新的数据质量检测任务
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param task body governance.CreateQualityTaskRequest true "质量检测任务信息"
// @Success 201 {object} APIResponse{data=governance.QualityTaskResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks [post]
func (c *DataQualityController) CreateQualityTask(w http.ResponseWriter, r *http.Request) {
	var req governance.CreateQualityTaskRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	task, err := c.governanceService.CreateQualityTask(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据质量检测任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建数据质量检测任务成功", task))
}

// GetQualityTasks 获取数据质量检测任务列表
// @Summary 获取数据质量检测任务列表
// @Description 分页获取数据质量检测任务列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "任务状态" Enums(pending,running,completed,failed,cancelled)
// @Param task_type query string false "任务类型" Enums(scheduled,manual,realtime)
// @Success 200 {object} APIResponse{data=governance.QualityTaskListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks [get]
func (c *DataQualityController) GetQualityTasks(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	status := r.URL.Query().Get("status")
	taskType := r.URL.Query().Get("task_type")

	tasks, total, err := c.governanceService.GetQualityTasks(page, size, status, taskType)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据质量检测任务列表失败", err))
		return
	}

	response := governance.QualityTaskListResponse{
		List:  tasks,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量检测任务列表成功", response))
}

// GetQualityTaskByID 根据ID获取数据质量检测任务
// @Summary 根据ID获取数据质量检测任务
// @Description 根据ID获取数据质量检测任务详情
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=governance.QualityTaskResponse} "获取成功"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks/{id} [get]
func (c *DataQualityController) GetQualityTaskByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	task, err := c.governanceService.GetQualityTaskByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据质量检测任务不存在", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取数据质量检测任务成功", task))
}

// StartQualityTask 启动数据质量检测任务
// @Summary 启动数据质量检测任务
// @Description 手动启动指定的数据质量检测任务
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=governance.QualityTaskExecutionResponse} "启动成功"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks/{id}/start [post]
func (c *DataQualityController) StartQualityTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	execution, err := c.governanceService.StartQualityTask(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("启动数据质量检测任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("启动数据质量检测任务成功", execution))
}

// StopQualityTask 停止数据质量检测任务
// @Summary 停止数据质量检测任务
// @Description 停止正在运行的数据质量检测任务
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "停止成功"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks/{id}/stop [post]
func (c *DataQualityController) StopQualityTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.StopQualityTask(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("停止数据质量检测任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("停止数据质量检测任务成功", nil))
}

// GetQualityTaskExecutions 获取数据质量检测任务执行记录
// @Summary 获取数据质量检测任务执行记录
// @Description 获取指定任务的执行记录列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} APIResponse{data=governance.QualityTaskExecutionListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks/{id}/executions [get]
func (c *DataQualityController) GetQualityTaskExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	executions, total, err := c.governanceService.GetQualityTaskExecutions(id, page, size)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据质量检测任务执行记录失败", err))
		return
	}

	response := governance.QualityTaskExecutionListResponse{
		List:  executions,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量检测任务执行记录成功", response))
}

// UpdateQualityTask 更新数据质量检测任务
// @Summary 更新数据质量检测任务
// @Description 更新数据质量检测任务信息
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param updates body governance.UpdateQualityTaskRequest true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks/{id} [put]
func (c *DataQualityController) UpdateQualityTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req governance.UpdateQualityTaskRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.governanceService.UpdateQualityTask(id, &req); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新数据质量检测任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新数据质量检测任务成功", nil))
}

// DeleteQualityTask 删除数据质量检测任务
// @Summary 删除数据质量检测任务
// @Description 删除指定的数据质量检测任务及其相关执行记录
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/tasks/{id} [delete]
func (c *DataQualityController) DeleteQualityTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteQualityTask(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据质量检测任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据质量检测任务成功", nil))
}

// === 数据血缘管理 ===

// CreateDataLineage 创建数据血缘关系
// @Summary 创建数据血缘关系
// @Description 创建新的数据血缘关系
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param lineage body governance.CreateDataLineageRequest true "数据血缘信息"
// @Success 201 {object} APIResponse{data=governance.DataLineageResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/data-lineage [post]
func (c *DataQualityController) CreateDataLineage(w http.ResponseWriter, r *http.Request) {
	var req governance.CreateDataLineageRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	lineage, err := c.governanceService.CreateDataLineage(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据血缘关系失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建数据血缘关系成功", lineage))
}

// GetDataLineage 获取数据血缘图
// @Summary 获取数据血缘图
// @Description 获取指定数据对象的血缘关系图
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param object_id path string true "数据对象ID"
// @Param object_type query string false "对象类型" Enums(table,interface,thematic_interface)
// @Param direction query string false "血缘方向" Enums(upstream,downstream,both) default(both)
// @Param depth query int false "血缘深度" default(3)
// @Success 200 {object} APIResponse{data=governance.DataLineageGraphResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/data-lineage/{object_id} [get]
func (c *DataQualityController) GetDataLineage(w http.ResponseWriter, r *http.Request) {
	objectID := chi.URLParam(r, "object_id")
	objectType := r.URL.Query().Get("object_type")
	direction := r.URL.Query().Get("direction")
	if direction == "" {
		direction = "both"
	}
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth <= 0 {
		depth = 3
	}

	lineageGraph, err := c.governanceService.GetDataLineage(objectID, objectType, direction, depth)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据血缘图失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取数据血缘图成功", lineageGraph))
}

// === 系统日志管理 ===

// GetSystemLogs 获取系统日志列表
// @Summary 获取系统日志列表
// @Description 分页获取系统日志列表
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param operation_type query string false "操作类型"
// @Param object_type query string false "对象类型"
// @Param start_time query string false "开始时间" format(date-time)
// @Param end_time query string false "结束时间" format(date-time)
// @Success 200 {object} APIResponse{data=governance.SystemLogListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/system-logs [get]
func (c *DataQualityController) GetSystemLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 10
	}

	operationType := r.URL.Query().Get("operation_type")
	objectType := r.URL.Query().Get("object_type")

	var startTime, endTime *time.Time
	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	logs, total, err := c.governanceService.GetSystemLogs(page, pageSize, operationType, objectType, startTime, endTime)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取系统日志列表失败", err))
		return
	}

	var logResponses []governance.SystemLogResponse
	for _, log := range logs {
		var operatorID, operatorName, operatorIP string
		if log.OperatorID != nil {
			operatorID = *log.OperatorID
		}
		if log.OperatorName != nil {
			operatorName = *log.OperatorName
		}
		if log.OperatorIP != nil {
			operatorIP = *log.OperatorIP
		}

		var objectID string
		if log.ObjectID != nil {
			objectID = *log.ObjectID
		}

		logResponses = append(logResponses, governance.SystemLogResponse{
			ID:               log.ID,
			OperationType:    log.OperationType,
			ObjectType:       log.ObjectType,
			ObjectID:         objectID,
			OperatorID:       operatorID,
			OperatorName:     operatorName,
			OperatorIP:       operatorIP,
			OperationContent: log.OperationContent,
			OperationTime:    log.OperationTime,
			OperationResult:  log.OperationResult,
		})
	}

	response := governance.SystemLogListResponse{
		List:  logResponses,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}

	render.JSON(w, r, SuccessResponse("获取系统日志列表成功", response))
}

// === 模板管理接口 ===

// GetQualityRuleTemplates 获取数据质量规则模板列表
// @Summary 获取数据质量规则模板列表
// @Description 分页获取数据质量规则模板列表，包括内置模板
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param rule_type query string false "规则类型" Enums(completeness,accuracy,consistency,validity,uniqueness,timeliness,standardization)
// @Param category query string false "分类" Enums(basic_quality,data_cleansing,data_validation)
// @Param is_built_in query bool false "是否为内置模板"
// @Success 200 {object} APIResponse{data=governance.QualityRuleTemplateListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/templates/quality-rules [get]
func (c *DataQualityController) GetQualityRuleTemplates(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	ruleType := r.URL.Query().Get("rule_type")
	category := r.URL.Query().Get("category")

	var isBuiltIn *bool
	if isBuiltInStr := r.URL.Query().Get("is_built_in"); isBuiltInStr != "" {
		val := isBuiltInStr == "true"
		isBuiltIn = &val
	}

	templateService := c.governanceService.GetTemplateService()
	templates, total, err := templateService.GetQualityRuleTemplates(page, size, ruleType, category, isBuiltIn)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据质量规则模板列表失败", err))
		return
	}

	var templateResponses []governance.QualityRuleTemplateResponse
	for _, template := range templates {
		templateResponses = append(templateResponses, governance.QualityRuleTemplateResponse{
			ID:            template.ID,
			Name:          template.Name,
			Type:          template.Type,
			Category:      template.Category,
			Description:   template.Description,
			RuleLogic:     template.RuleLogic,
			Parameters:    template.Parameters,
			DefaultConfig: template.DefaultConfig,
			IsBuiltIn:     template.IsBuiltIn,
			IsEnabled:     template.IsEnabled,
			Version:       template.Version,
			Tags:          template.Tags,
			CreatedAt:     template.CreatedAt,
			CreatedBy:     template.CreatedBy,
			UpdatedAt:     template.UpdatedAt,
			UpdatedBy:     template.UpdatedBy,
		})
	}

	response := governance.QualityRuleTemplateListResponse{
		List:  templateResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据质量规则模板列表成功", response))
}

// GetDataMaskingTemplates 获取数据脱敏模板列表
// @Summary 获取数据脱敏模板列表
// @Description 分页获取数据脱敏模板列表，包括内置模板
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param masking_type query string false "脱敏类型" Enums(mask,replace,encrypt,pseudonymize)
// @Param category query string false "分类" Enums(personal_info,financial,medical,business,custom)
// @Param security_level query string false "安全级别" Enums(low,medium,high,critical)
// @Param is_built_in query bool false "是否为内置模板"
// @Success 200 {object} APIResponse{data=governance.DataMaskingTemplateListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/templates/masking-rules [get]
func (c *DataQualityController) GetDataMaskingTemplates(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	maskingType := r.URL.Query().Get("masking_type")
	category := r.URL.Query().Get("category")
	securityLevel := r.URL.Query().Get("security_level")

	var isBuiltIn *bool
	if isBuiltInStr := r.URL.Query().Get("is_built_in"); isBuiltInStr != "" {
		val := isBuiltInStr == "true"
		isBuiltIn = &val
	}

	templateService := c.governanceService.GetTemplateService()
	templates, total, err := templateService.GetDataMaskingTemplates(page, size, maskingType, category, securityLevel, isBuiltIn)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据脱敏模板列表失败", err))
		return
	}

	var templateResponses []governance.DataMaskingTemplateResponse
	for _, template := range templates {
		templateResponses = append(templateResponses, governance.DataMaskingTemplateResponse{
			ID:              template.ID,
			Name:            template.Name,
			MaskingType:     template.MaskingType,
			Category:        template.Category,
			Description:     template.Description,
			ApplicableTypes: template.ApplicableTypes,
			MaskingLogic:    template.MaskingLogic,
			Parameters:      template.Parameters,
			DefaultConfig:   template.DefaultConfig,
			SecurityLevel:   template.SecurityLevel,
			IsBuiltIn:       template.IsBuiltIn,
			IsEnabled:       template.IsEnabled,
			Version:         template.Version,
			Tags:            template.Tags,
			CreatedAt:       template.CreatedAt,
			CreatedBy:       template.CreatedBy,
			UpdatedAt:       template.UpdatedAt,
			UpdatedBy:       template.UpdatedBy,
		})
	}

	response := governance.DataMaskingTemplateListResponse{
		List:  templateResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据脱敏模板列表成功", response))
}

// GetDataCleansingTemplates 获取数据清洗模板列表
// @Summary 获取数据清洗模板列表
// @Description 分页获取数据清洗模板列表，包括内置模板
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param rule_type query string false "规则类型" Enums(standardization,deduplication,validation,transformation,enrichment)
// @Param category query string false "分类" Enums(data_format,data_quality,data_integrity)
// @Param is_built_in query bool false "是否为内置模板"
// @Success 200 {object} APIResponse{data=governance.DataCleansingTemplateListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/templates/cleansing-rules [get]
func (c *DataQualityController) GetDataCleansingTemplates(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	ruleType := r.URL.Query().Get("rule_type")
	category := r.URL.Query().Get("category")

	var isBuiltIn *bool
	if isBuiltInStr := r.URL.Query().Get("is_built_in"); isBuiltInStr != "" {
		val := isBuiltInStr == "true"
		isBuiltIn = &val
	}

	templateService := c.governanceService.GetTemplateService()
	templates, total, err := templateService.GetDataCleansingTemplates(page, size, ruleType, category, isBuiltIn)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据清洗模板列表失败", err))
		return
	}

	var templateResponses []governance.DataCleansingTemplateResponse
	for _, template := range templates {
		templateResponses = append(templateResponses, governance.DataCleansingTemplateResponse{
			ID:              template.ID,
			Name:            template.Name,
			Description:     template.Description,
			RuleType:        template.RuleType,
			Category:        template.Category,
			CleansingLogic:  template.CleansingLogic,
			Parameters:      template.Parameters,
			DefaultConfig:   template.DefaultConfig,
			ApplicableTypes: template.ApplicableTypes,
			ComplexityLevel: template.ComplexityLevel,
			IsBuiltIn:       template.IsBuiltIn,
			IsEnabled:       template.IsEnabled,
			Version:         template.Version,
			Tags:            template.Tags,
			CreatedBy:       template.CreatedBy,
			UpdatedBy:       template.UpdatedBy,
			CreatedAt:       template.CreatedAt,
			UpdatedAt:       template.UpdatedAt,
		})
	}

	response := governance.DataCleansingTemplateListResponse{
		List:  templateResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据清洗模板列表成功", response))
}

// === 规则测试接口 ===

// TestQualityRule 测试数据质量规则
// @Summary 测试数据质量规则
// @Description 使用测试数据验证数据质量规则的执行效果
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param request body governance.TestQualityRuleRequest true "质量规则测试请求"
// @Success 200 {object} APIResponse{data=governance.TestRuleResponse} "测试成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/test/quality-rule [post]
func (c *DataQualityController) TestQualityRule(w http.ResponseWriter, r *http.Request) {
	var req governance.TestQualityRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.governanceService.TestQualityRule(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("测试质量规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("测试质量规则成功", result))
}

// TestMaskingRule 测试数据脱敏规则
// @Summary 测试数据脱敏规则
// @Description 使用测试数据验证数据脱敏规则的执行效果
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param request body governance.TestMaskingRuleRequest true "脱敏规则测试请求"
// @Success 200 {object} APIResponse{data=governance.TestRuleResponse} "测试成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/test/masking-rule [post]
func (c *DataQualityController) TestMaskingRule(w http.ResponseWriter, r *http.Request) {
	var req governance.TestMaskingRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.governanceService.TestMaskingRule(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("测试脱敏规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("测试脱敏规则成功", result))
}

// TestCleansingRule 测试数据清洗规则
// @Summary 测试数据清洗规则
// @Description 使用测试数据验证数据清洗规则的执行效果
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param request body governance.TestCleansingRuleRequest true "清洗规则测试请求"
// @Success 200 {object} APIResponse{data=governance.TestRuleResponse} "测试成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/test/cleansing-rule [post]
func (c *DataQualityController) TestCleansingRule(w http.ResponseWriter, r *http.Request) {
	var req governance.TestCleansingRuleRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.governanceService.TestCleansingRule(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("测试清洗规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("测试清洗规则成功", result))
}

// TestBatchRules 批量测试多个规则
// @Summary 批量测试多个规则
// @Description 批量测试质量规则、脱敏规则和清洗规则的组合执行效果
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param request body governance.TestBatchRulesRequest true "批量规则测试请求"
// @Success 200 {object} APIResponse{data=governance.TestRuleResponse} "测试成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/test/batch-rules [post]
func (c *DataQualityController) TestBatchRules(w http.ResponseWriter, r *http.Request) {
	var req governance.TestBatchRulesRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.governanceService.TestBatchRules(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("批量测试规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("批量测试规则成功", result))
}

// TestRulePreview 预览规则执行效果
// @Summary 预览规则执行效果
// @Description 预览规则执行效果，不实际执行规则，仅显示预期变化
// @Tags 数据质量
// @Accept json
// @Produce json
// @Param request body governance.TestRulePreviewRequest true "规则预览请求"
// @Success 200 {object} APIResponse{data=governance.TestRulePreviewResponse} "预览成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /data-quality/test/rule-preview [post]
func (c *DataQualityController) TestRulePreview(w http.ResponseWriter, r *http.Request) {
	var req governance.TestRulePreviewRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.governanceService.TestRulePreview(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("预览规则执行效果失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("预览规则执行效果成功", result))
}
