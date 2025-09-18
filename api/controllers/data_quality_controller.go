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

	rule := &models.QualityRule{
		Name:              req.Name,
		Type:              req.Type,
		Config:            req.Config,
		RelatedObjectID:   req.RelatedObjectID,
		RelatedObjectType: req.RelatedObjectType,
		IsEnabled:         req.IsEnabled,
	}

	if err := c.governanceService.CreateQualityRule(rule); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据质量规则失败", err))
		return
	}

	response := &governance.QualityRuleResponse{
		ID:                rule.ID,
		Name:              rule.Name,
		Type:              rule.Type,
		Config:            rule.Config,
		RelatedObjectID:   rule.RelatedObjectID,
		RelatedObjectType: rule.RelatedObjectType,
		IsEnabled:         rule.IsEnabled,
		CreatedAt:         rule.CreatedAt,
		CreatedBy:         rule.CreatedBy,
		UpdatedAt:         rule.UpdatedAt,
		UpdatedBy:         rule.UpdatedBy,
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
			ID:                rule.ID,
			Name:              rule.Name,
			Type:              rule.Type,
			Config:            rule.Config,
			RelatedObjectID:   rule.RelatedObjectID,
			RelatedObjectType: rule.RelatedObjectType,
			IsEnabled:         rule.IsEnabled,
			CreatedAt:         rule.CreatedAt,
			CreatedBy:         rule.CreatedBy,
			UpdatedAt:         rule.UpdatedAt,
			UpdatedBy:         rule.UpdatedBy,
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
		ID:                rule.ID,
		Name:              rule.Name,
		Type:              rule.Type,
		Config:            rule.Config,
		RelatedObjectID:   rule.RelatedObjectID,
		RelatedObjectType: rule.RelatedObjectType,
		IsEnabled:         rule.IsEnabled,
		CreatedAt:         rule.CreatedAt,
		CreatedBy:         rule.CreatedBy,
		UpdatedAt:         rule.UpdatedAt,
		UpdatedBy:         rule.UpdatedBy,
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
	if req.Config != nil {
		updates["config"] = req.Config
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

	rule := &models.DataMaskingRule{
		Name:          req.Name,
		DataSource:    req.DataSource,
		DataTable:     req.DataTable,
		FieldName:     req.FieldName,
		FieldType:     req.FieldType,
		MaskingType:   req.MaskingType,
		MaskingConfig: req.MaskingConfig,
		IsEnabled:     req.IsEnabled,
	}

	if err := c.governanceService.CreateMaskingRule(rule); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据脱敏规则失败", err))
		return
	}

	response := &governance.MaskingRuleResponse{
		ID:            rule.ID,
		Name:          rule.Name,
		DataSource:    rule.DataSource,
		DataTable:     rule.DataTable,
		FieldName:     rule.FieldName,
		FieldType:     rule.FieldType,
		MaskingType:   rule.MaskingType,
		MaskingConfig: rule.MaskingConfig,
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
			DataSource:    rule.DataSource,
			DataTable:     rule.DataTable,
			FieldName:     rule.FieldName,
			FieldType:     rule.FieldType,
			MaskingType:   rule.MaskingType,
			MaskingConfig: rule.MaskingConfig,
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
		DataSource:    rule.DataSource,
		DataTable:     rule.DataTable,
		FieldName:     rule.FieldName,
		FieldType:     rule.FieldType,
		MaskingType:   rule.MaskingType,
		MaskingConfig: rule.MaskingConfig,
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
	if req.MaskingConfig != nil {
		updates["masking_config"] = req.MaskingConfig
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
