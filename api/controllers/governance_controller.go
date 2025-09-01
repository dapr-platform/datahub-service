/*
 * @module api/controllers/governance_controller
 * @description 数据治理控制器，提供数据质量管理、元数据管理、数据脱敏等API接口
 * @architecture 分层架构 - 控制器层
 * @documentReference ai_docs/requirements.md
 * @stateFlow HTTP请求处理流程
 * @rules 统一的错误处理和响应格式
 * @dependencies datahub-service/service, github.com/go-chi/chi/v5
 * @refs ai_docs/model.md
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/models"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// GovernanceController 数据治理控制器
type GovernanceController struct {
	governanceService *service.GovernanceService
}

// NewGovernanceController 创建数据治理控制器实例
func NewGovernanceController(governanceService *service.GovernanceService) *GovernanceController {
	return &GovernanceController{
		governanceService: governanceService,
	}
}

// === 数据质量规则管理 ===

// CreateQualityRule 创建数据质量规则
// @Summary 创建数据质量规则
// @Description 创建新的数据质量规则
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param rule body models.QualityRule true "数据质量规则信息"
// @Success 201 {object} APIResponse{data=models.QualityRule} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-rules [post]
func (c *GovernanceController) CreateQualityRule(w http.ResponseWriter, r *http.Request) {
	var rule models.QualityRule
	if err := render.DecodeJSON(r.Body, &rule); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	if err := c.governanceService.CreateQualityRule(&rule); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "创建数据质量规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusCreated,
		Msg: "创建数据质量规则成功",
		Data:    rule,
	})
}

// GetQualityRules 获取数据质量规则列表
// @Summary 获取数据质量规则列表
// @Description 分页获取数据质量规则列表
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param type query string false "规则类型"
// @Param object_type query string false "关联对象类型"
// @Success 200 {object} PaginatedResponse{data=[]models.QualityRule} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-rules [get]
func (c *GovernanceController) GetQualityRules(w http.ResponseWriter, r *http.Request) {
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
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "获取数据质量规则列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status:    http.StatusOK,
		Msg: "获取数据质量规则列表成功",
		Data:    rules,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// GetQualityRuleByID 根据ID获取数据质量规则
// @Summary 根据ID获取数据质量规则
// @Description 根据ID获取数据质量规则详情
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse{data=models.QualityRule} "获取成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-rules/{id} [get]
func (c *GovernanceController) GetQualityRuleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rule, err := c.governanceService.GetQualityRuleByID(id)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusNotFound,
			Msg: "数据质量规则不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "获取数据质量规则成功",
		Data:    rule,
	})
}

// UpdateQualityRule 更新数据质量规则
// @Summary 更新数据质量规则
// @Description 更新数据质量规则信息
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-rules/{id} [put]
func (c *GovernanceController) UpdateQualityRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	if err := c.governanceService.UpdateQualityRule(id, updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "更新数据质量规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "更新数据质量规则成功",
	})
}

// DeleteQualityRule 删除数据质量规则
// @Summary 删除数据质量规则
// @Description 删除指定的数据质量规则
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-rules/{id} [delete]
func (c *GovernanceController) DeleteQualityRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteQualityRule(id); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "删除数据质量规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "删除数据质量规则成功",
	})
}

// === 元数据管理 ===

// CreateMetadata 创建元数据
// @Summary 创建元数据
// @Description 创建新的元数据
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param metadata body models.Metadata true "元数据信息"
// @Success 201 {object} APIResponse{data=models.Metadata} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/metadata [post]
func (c *GovernanceController) CreateMetadata(w http.ResponseWriter, r *http.Request) {
	var metadata models.Metadata
	if err := render.DecodeJSON(r.Body, &metadata); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	if err := c.governanceService.CreateMetadata(&metadata); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "创建元数据失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusCreated,
		Msg: "创建元数据成功",
		Data:    metadata,
	})
}

// GetMetadataList 获取元数据列表
// @Summary 获取元数据列表
// @Description 分页获取元数据列表
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param type query string false "元数据类型"
// @Param name query string false "元数据名称"
// @Success 200 {object} PaginatedResponse{data=[]models.Metadata} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/metadata [get]
func (c *GovernanceController) GetMetadataList(w http.ResponseWriter, r *http.Request) {
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
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "获取元数据列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status:    http.StatusOK,
		Msg: "获取元数据列表成功",
		Data:    metadataList,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// GetMetadataByID 根据ID获取元数据
// @Summary 根据ID获取元数据
// @Description 根据ID获取元数据详情
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "元数据ID"
// @Success 200 {object} APIResponse{data=models.Metadata} "获取成功"
// @Failure 404 {object} APIResponse "元数据不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/metadata/{id} [get]
func (c *GovernanceController) GetMetadataByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	metadata, err := c.governanceService.GetMetadataByID(id)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusNotFound,
			Msg: "元数据不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "获取元数据成功",
		Data:    metadata,
	})
}

// UpdateMetadata 更新元数据
// @Summary 更新元数据
// @Description 更新元数据信息
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "元数据ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "元数据不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/metadata/{id} [put]
func (c *GovernanceController) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	if err := c.governanceService.UpdateMetadata(id, updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "更新元数据失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "更新元数据成功",
	})
}

// DeleteMetadata 删除元数据
// @Summary 删除元数据
// @Description 删除指定的元数据
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "元数据ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "元数据不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/metadata/{id} [delete]
func (c *GovernanceController) DeleteMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteMetadata(id); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "删除元数据失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "删除元数据成功",
	})
}

// === 数据脱敏规则管理 ===

// CreateMaskingRule 创建数据脱敏规则
// @Summary 创建数据脱敏规则
// @Description 创建新的数据脱敏规则
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param rule body models.DataMaskingRule true "数据脱敏规则信息"
// @Success 201 {object} APIResponse{data=models.DataMaskingRule} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/masking-rules [post]
func (c *GovernanceController) CreateMaskingRule(w http.ResponseWriter, r *http.Request) {
	var rule models.DataMaskingRule
	if err := render.DecodeJSON(r.Body, &rule); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	if err := c.governanceService.CreateMaskingRule(&rule); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "创建数据脱敏规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusCreated,
		Msg: "创建数据脱敏规则成功",
		Data:    rule,
	})
}

// GetMaskingRules 获取数据脱敏规则列表
// @Summary 获取数据脱敏规则列表
// @Description 分页获取数据脱敏规则列表
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param data_source query string false "数据源"
// @Param masking_type query string false "脱敏类型"
// @Success 200 {object} PaginatedResponse{data=[]models.DataMaskingRule} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/masking-rules [get]
func (c *GovernanceController) GetMaskingRules(w http.ResponseWriter, r *http.Request) {
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
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "获取数据脱敏规则列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status:    http.StatusOK,
		Msg: "获取数据脱敏规则列表成功",
		Data:    rules,
		Total:   total,
		Page:    page,
		Size:    pageSize,
	})
}

// GetMaskingRuleByID 根据ID获取数据脱敏规则
// @Summary 根据ID获取数据脱敏规则
// @Description 根据ID获取数据脱敏规则详情
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse{data=models.DataMaskingRule} "获取成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/masking-rules/{id} [get]
func (c *GovernanceController) GetMaskingRuleByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rule, err := c.governanceService.GetMaskingRuleByID(id)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusNotFound,
			Msg: "数据脱敏规则不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "获取数据脱敏规则成功",
		Data:    rule,
	})
}

// UpdateMaskingRule 更新数据脱敏规则
// @Summary 更新数据脱敏规则
// @Description 更新数据脱敏规则信息
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/masking-rules/{id} [put]
func (c *GovernanceController) UpdateMaskingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	if err := c.governanceService.UpdateMaskingRule(id, updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "更新数据脱敏规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "更新数据脱敏规则成功",
	})
}

// DeleteMaskingRule 删除数据脱敏规则
// @Summary 删除数据脱敏规则
// @Description 删除指定的数据脱敏规则
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/masking-rules/{id} [delete]
func (c *GovernanceController) DeleteMaskingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.governanceService.DeleteMaskingRule(id); err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "删除数据脱敏规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "删除数据脱敏规则成功",
	})
}

// === 系统日志管理 ===

// GetSystemLogs 获取系统日志列表
// @Summary 获取系统日志列表
// @Description 分页获取系统日志列表
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param operation_type query string false "操作类型"
// @Param object_type query string false "对象类型"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} PaginatedResponse{data=[]models.SystemLog} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/system-logs [get]
func (c *GovernanceController) GetSystemLogs(w http.ResponseWriter, r *http.Request) {
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
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "获取系统日志列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status:    http.StatusOK,
		Msg: "获取系统日志列表成功",
		Data:    logs,
		Total:   total,
		Page:    page,
		Size:    pageSize,
	})
}

// === 数据质量报告 ===

// GetQualityReports 获取数据质量报告列表
// @Summary 获取数据质量报告列表
// @Description 分页获取数据质量报告列表
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param object_type query string false "对象类型"
// @Success 200 {object} PaginatedResponse{data=[]models.DataQualityReport} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-reports [get]
func (c *GovernanceController) GetQualityReports(w http.ResponseWriter, r *http.Request) {
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
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "获取数据质量报告列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status:    http.StatusOK,
		Msg: "获取数据质量报告列表成功",
		Data:    reports,
		Total:   total,
		Page:    page,
		Size:    pageSize,
	})
}

// GetQualityReportByID 根据ID获取数据质量报告
// @Summary 根据ID获取数据质量报告
// @Description 根据ID获取数据质量报告详情
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param id path string true "报告ID"
// @Success 200 {object} APIResponse{data=models.DataQualityReport} "获取成功"
// @Failure 404 {object} APIResponse "报告不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-reports/{id} [get]
func (c *GovernanceController) GetQualityReportByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	report, err := c.governanceService.GetQualityReportByID(id)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusNotFound,
			Msg: "数据质量报告不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "获取数据质量报告成功",
		Data:    report,
	})
}

// RunQualityCheck 执行数据质量检查
// @Summary 执行数据质量检查
// @Description 对指定对象执行数据质量检查并生成报告
// @Tags 数据治理
// @Accept json
// @Produce json
// @Param object_id query string true "对象ID"
// @Param object_type query string true "对象类型"
// @Success 200 {object} APIResponse{data=models.DataQualityReport} "检查成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /governance/quality-check [post]
func (c *GovernanceController) RunQualityCheck(w http.ResponseWriter, r *http.Request) {
	objectID := r.URL.Query().Get("object_id")
	objectType := r.URL.Query().Get("object_type")

	if objectID == "" || objectType == "" {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "缺少必要参数",
		})
		return
	}

	report, err := c.governanceService.RunQualityCheck(objectID, objectType)
	if err != nil {
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "执行数据质量检查失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "执行数据质量检查成功",
		Data:    report,
	})
}
