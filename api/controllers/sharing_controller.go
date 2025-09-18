/*
 * @module api/controllers/sharing_controller
 * @description 数据共享服务控制器，提供API应用管理、数据订阅、数据使用申请等API接口
 * @architecture 分层架构 - 控制器层
 * @documentReference ai_docs/requirements.md
 * @stateFlow HTTP请求处理流程
 * @rules 统一的错误处理和响应格式
 * @dependencies datahub-service/service, github.com/go-chi/chi/v5
 * @refs ai_docs/model.md
 */

package controllers

import (
	"datahub-service/service/models"
	"datahub-service/service/sharing"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// SharingController 数据共享服务控制器
type SharingController struct {
	sharingService *sharing.SharingService
}

// NewSharingController 创建数据共享服务控制器实例
func NewSharingController(sharingService *sharing.SharingService) *SharingController {
	return &SharingController{
		sharingService: sharingService,
	}
}

// CreateApiApplicationRequest 创建API应用请求结构
type CreateApiApplicationRequest struct {
	Name              string  `json:"name" validate:"required"`
	Path              string  `json:"path" validate:"required"`
	ThematicLibraryID string  `json:"thematic_library_id" validate:"required"`
	Description       *string `json:"description"`
	ContactPerson     string  `json:"contact_person" validate:"required"`
	ContactPhone      string  `json:"contact_phone" validate:"required"`
}

// ApiApplicationListResponse API应用列表响应结构
type ApiApplicationListResponse struct {
	List  []models.ApiApplication `json:"list"`
	Total int64                   `json:"total"`
	Page  int                     `json:"page"`
	Size  int                     `json:"size"`
}

// ApiRateLimitListResponse API限流规则列表响应结构
type ApiRateLimitListResponse struct {
	List  []models.ApiRateLimit `json:"list"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// DataSubscriptionListResponse 数据订阅列表响应结构
type DataSubscriptionListResponse struct {
	List  []models.DataSubscription `json:"list"`
	Total int64                     `json:"total"`
	Page  int                       `json:"page"`
	Size  int                       `json:"size"`
}

// DataAccessRequestListResponse 数据使用申请列表响应结构
type DataAccessRequestListResponse struct {
	List  []models.DataAccessRequest `json:"list"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

// ApiUsageLogListResponse API使用日志列表响应结构
type ApiUsageLogListResponse struct {
	List  []models.ApiUsageLog `json:"list"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
}

// === API应用管理 ===

// CreateApiApplication 创建API应用
// @Summary 创建API应用
// @Description 创建新的API接入应用
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param app body CreateApiApplicationRequest true "API应用信息"
// @Success 201 {object} APIResponse{data=models.ApiApplication} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-applications [post]
func (c *SharingController) CreateApiApplication(w http.ResponseWriter, r *http.Request) {
	var req CreateApiApplicationRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	app := &models.ApiApplication{
		Name:              req.Name,
		Path:              req.Path,
		ThematicLibraryID: req.ThematicLibraryID,
		Description:       req.Description,
		ContactPerson:     req.ContactPerson,
		ContactPhone:      req.ContactPhone,
	}

	if err := c.sharingService.CreateApiApplication(app); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建API应用失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse(
		"创建API应用成功",
		app,
	))
}

// GetApiApplications 获取API应用列表
// @Summary 获取API应用列表
// @Description 分页获取API应用列表
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "应用状态"
// @Success 200 {object} APIResponse{data=ApiApplicationListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-applications [get]
func (c *SharingController) GetApiApplications(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	status := r.URL.Query().Get("status")

	apps, total, err := c.sharingService.GetApiApplications(page, size, status)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取API应用列表失败", err))
		return
	}

	// 构建响应
	response := ApiApplicationListResponse{
		List:  apps,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取API应用列表成功", response))
}

// GetApiApplicationByID 根据ID获取API应用
// @Summary 根据ID获取API应用
// @Description 根据ID获取API应用详情
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "应用ID"
// @Success 200 {object} APIResponse{data=models.ApiApplication} "获取成功"
// @Failure 404 {object} APIResponse "应用不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-applications/{id} [get]
func (c *SharingController) GetApiApplicationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	app, err := c.sharingService.GetApiApplicationByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("API应用不存在", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取API应用成功", app))
}

// UpdateApiApplication 更新API应用
// @Summary 更新API应用
// @Description 更新API应用信息
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "应用ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "应用不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-applications/{id} [put]
func (c *SharingController) UpdateApiApplication(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.UpdateApiApplication(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新API应用失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新API应用成功", nil))
}

// DeleteApiApplication 删除API应用
// @Summary 删除API应用
// @Description 删除指定的API应用
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "应用ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "应用不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-applications/{id} [delete]
func (c *SharingController) DeleteApiApplication(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.sharingService.DeleteApiApplication(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除API应用失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除API应用成功", nil))
}

// === API限流管理 ===

// CreateApiRateLimit 创建API限流规则
// @Summary 创建API限流规则
// @Description 创建新的API限流规则
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param limit body models.ApiRateLimit true "API限流规则信息"
// @Success 201 {object} APIResponse{data=models.ApiRateLimit} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-rate-limits [post]
func (c *SharingController) CreateApiRateLimit(w http.ResponseWriter, r *http.Request) {
	var limit models.ApiRateLimit
	if err := render.DecodeJSON(r.Body, &limit); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.CreateApiRateLimit(&limit); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建API限流规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建API限流规则成功", limit))
}

// GetApiRateLimits 获取API限流规则列表
// @Summary 获取API限流规则列表
// @Description 分页获取API限流规则列表
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param application_id query string false "应用ID"
// @Success 200 {object} APIResponse{data=ApiRateLimitListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-rate-limits [get]
func (c *SharingController) GetApiRateLimits(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	applicationID := r.URL.Query().Get("application_id")

	limits, total, err := c.sharingService.GetApiRateLimits(page, size, applicationID)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取API限流规则列表失败", err))
		return
	}

	// 构建响应
	response := ApiRateLimitListResponse{
		List:  limits,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取API限流规则列表成功", response))
}

// UpdateApiRateLimit 更新API限流规则
// @Summary 更新API限流规则
// @Description 更新API限流规则信息
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-rate-limits/{id} [put]
func (c *SharingController) UpdateApiRateLimit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.UpdateApiRateLimit(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新API限流规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新API限流规则成功", nil))
}

// DeleteApiRateLimit 删除API限流规则
// @Summary 删除API限流规则
// @Description 删除指定的API限流规则
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "规则不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-rate-limits/{id} [delete]
func (c *SharingController) DeleteApiRateLimit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.sharingService.DeleteApiRateLimit(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除API限流规则失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除API限流规则成功", nil))
}

// === 数据订阅管理 ===

// CreateDataSubscription 创建数据订阅
// @Summary 创建数据订阅
// @Description 创建新的数据订阅
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param subscription body models.DataSubscription true "数据订阅信息"
// @Success 201 {object} APIResponse{data=models.DataSubscription} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-subscriptions [post]
func (c *SharingController) CreateDataSubscription(w http.ResponseWriter, r *http.Request) {
	var subscription models.DataSubscription
	if err := render.DecodeJSON(r.Body, &subscription); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.CreateDataSubscription(&subscription); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据订阅失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建数据订阅成功", subscription))
}

// GetDataSubscriptions 获取数据订阅列表
// @Summary 获取数据订阅列表
// @Description 分页获取数据订阅列表
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param subscriber_id query string false "订阅者ID"
// @Param resource_type query string false "资源类型"
// @Param status query string false "订阅状态"
// @Success 200 {object} APIResponse{data=DataSubscriptionListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-subscriptions [get]
func (c *SharingController) GetDataSubscriptions(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	subscriberID := r.URL.Query().Get("subscriber_id")
	resourceType := r.URL.Query().Get("resource_type")
	status := r.URL.Query().Get("status")

	subscriptions, total, err := c.sharingService.GetDataSubscriptions(page, size, subscriberID, resourceType, status)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据订阅列表失败", err))
		return
	}

	// 构建响应
	response := DataSubscriptionListResponse{
		List:  subscriptions,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据订阅列表成功", response))
}

// GetDataSubscriptionByID 根据ID获取数据订阅
// @Summary 根据ID获取数据订阅
// @Description 根据ID获取数据订阅详情
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "订阅ID"
// @Success 200 {object} APIResponse{data=models.DataSubscription} "获取成功"
// @Failure 404 {object} APIResponse "订阅不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-subscriptions/{id} [get]
func (c *SharingController) GetDataSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	subscription, err := c.sharingService.GetDataSubscriptionByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据订阅不存在", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取数据订阅成功", subscription))
}

// UpdateDataSubscription 更新数据订阅
// @Summary 更新数据订阅
// @Description 更新数据订阅信息
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "订阅ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "订阅不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-subscriptions/{id} [put]
func (c *SharingController) UpdateDataSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.UpdateDataSubscription(id, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新数据订阅失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新数据订阅成功", nil))
}

// DeleteDataSubscription 删除数据订阅
// @Summary 删除数据订阅
// @Description 删除指定的数据订阅
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "订阅ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 404 {object} APIResponse "订阅不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-subscriptions/{id} [delete]
func (c *SharingController) DeleteDataSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.sharingService.DeleteDataSubscription(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据订阅失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据订阅成功", nil))
}

// === 数据使用申请管理 ===

// CreateDataAccessRequest 创建数据使用申请
// @Summary 创建数据使用申请
// @Description 创建新的数据使用申请
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param request body models.DataAccessRequest true "数据使用申请信息"
// @Success 201 {object} APIResponse{data=models.DataAccessRequest} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-access-requests [post]
func (c *SharingController) CreateDataAccessRequest(w http.ResponseWriter, r *http.Request) {
	var request models.DataAccessRequest
	if err := render.DecodeJSON(r.Body, &request); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.CreateDataAccessRequest(&request); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建数据使用申请失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建数据使用申请成功", request))
}

// GetDataAccessRequests 获取数据使用申请列表
// @Summary 获取数据使用申请列表
// @Description 分页获取数据使用申请列表
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param requester_id query string false "申请人ID"
// @Param resource_type query string false "资源类型"
// @Param status query string false "申请状态"
// @Success 200 {object} APIResponse{data=DataAccessRequestListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-access-requests [get]
func (c *SharingController) GetDataAccessRequests(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	requesterID := r.URL.Query().Get("requester_id")
	resourceType := r.URL.Query().Get("resource_type")
	status := r.URL.Query().Get("status")

	requests, total, err := c.sharingService.GetDataAccessRequests(page, size, requesterID, resourceType, status)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据使用申请列表失败", err))
		return
	}

	// 构建响应
	response := DataAccessRequestListResponse{
		List:  requests,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据使用申请列表成功", response))
}

// GetDataAccessRequestByID 根据ID获取数据使用申请
// @Summary 根据ID获取数据使用申请
// @Description 根据ID获取数据使用申请详情
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "申请ID"
// @Success 200 {object} APIResponse{data=models.DataAccessRequest} "获取成功"
// @Failure 404 {object} APIResponse "申请不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-access-requests/{id} [get]
func (c *SharingController) GetDataAccessRequestByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	request, err := c.sharingService.GetDataAccessRequestByID(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据使用申请不存在", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取数据使用申请成功", request))
}

// ApproveDataAccessRequestRequest 审批数据使用申请请求结构
type ApproveDataAccessRequestRequest struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment"`
}

// ApproveDataAccessRequest 审批数据使用申请
// @Summary 审批数据使用申请
// @Description 审批数据使用申请
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "申请ID"
// @Param approval body ApproveDataAccessRequestRequest true "审批信息"
// @Success 200 {object} APIResponse "审批成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "申请不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/data-access-requests/{id}/approve [post]
func (c *SharingController) ApproveDataAccessRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req ApproveDataAccessRequestRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	// TODO: 从认证信息中获取审批人ID
	approverID := "system" // 临时使用系统ID

	if err := c.sharingService.ApproveDataAccessRequest(id, approverID, req.Approved, req.Comment); err != nil {
		render.JSON(w, r, InternalErrorResponse("审批数据使用申请失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("审批数据使用申请成功", nil))
}

// === API使用日志管理 ===

// GetApiUsageLogs 获取API使用日志列表
// @Summary 获取API使用日志列表
// @Description 分页获取API使用日志列表
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param application_id query string false "应用ID"
// @Param user_id query string false "用户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} APIResponse{data=ApiUsageLogListResponse} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-usage-logs [get]
func (c *SharingController) GetApiUsageLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	applicationID := r.URL.Query().Get("application_id")
	userID := r.URL.Query().Get("user_id")

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

	logs, total, err := c.sharingService.GetApiUsageLogs(page, size, applicationID, userID, startTime, endTime)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取API使用日志列表失败", err))
		return
	}

	// 构建响应
	response := ApiUsageLogListResponse{
		List:  logs,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取API使用日志列表成功", response))
}

// === ApiKey管理 ===

// CreateApiKeyRequest 创建ApiKey请求结构
type CreateApiKeyRequest struct {
	Name           string     `json:"name" validate:"required"`
	Description    string     `json:"description"`
	ApplicationIDs []string   `json:"application_ids" validate:"required,min=1"` // 关联的应用ID列表
	ExpiresAt      *time.Time `json:"expires_at"`
}

// CreateApiKeyResponse 创建ApiKey响应结构
type CreateApiKeyResponse struct {
	ApiKey   models.ApiKey `json:"api_key"`
	KeyValue string        `json:"key_value"` // 完整的Key值，仅返回一次
}

// UpdateApiKeyApplicationsRequest 更新ApiKey关联应用请求结构
type UpdateApiKeyApplicationsRequest struct {
	ApplicationIDs []string `json:"application_ids" validate:"required,min=1"` // 关联的应用ID列表
}

// CreateApiKey 创建一个新的ApiKey并关联到指定的应用
// @Summary 生成API密钥
// @Description 创建一个新的ApiKey并关联到指定的应用，返回完整的Key值（仅此一次）
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param key body CreateApiKeyRequest true "ApiKey信息"
// @Success 201 {object} APIResponse{data=CreateApiKeyResponse} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-keys [post]
func (c *SharingController) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	var req CreateApiKeyRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	apiKey, keyValue, err := c.sharingService.CreateApiKey(req.Name, req.Description, req.ApplicationIDs, req.ExpiresAt)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("生成API密钥失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("生成API密钥成功", CreateApiKeyResponse{
		ApiKey:   *apiKey,
		KeyValue: keyValue,
	}))
}

// GetApiKeys 获取API密钥列表
// @Summary 获取API密钥列表
// @Description 获取API密钥列表（不包含Key本身），可选择按应用过滤
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param app_id query string false "应用ID，用于过滤特定应用的ApiKey"
// @Success 200 {object} APIResponse{data=[]models.ApiKey} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-keys [get]
func (c *SharingController) GetApiKeys(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")

	keys, err := c.sharingService.GetApiKeys(appID)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取API密钥列表失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取API密钥列表成功", keys))
}

// GetApiKeyByID 根据ID获取API密钥详情
// @Summary 获取API密钥详情
// @Description 根据ID获取API密钥详情（不包含Key本身）
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Success 200 {object} APIResponse{data=models.ApiKey} "获取成功"
// @Failure 404 {object} APIResponse "密钥不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-keys/{id} [get]
func (c *SharingController) GetApiKeyByID(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "id")

	key, err := c.sharingService.GetApiKeyByID(keyID)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("API密钥不存在", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取API密钥成功", key))
}

// UpdateApiKey 更新ApiKey信息
// @Summary 更新API密钥
// @Description 更新ApiKey信息（如名称、描述、状态）
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-keys/{id} [put]
func (c *SharingController) UpdateApiKey(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := render.DecodeJSON(r.Body, &updates); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.UpdateApiKey(keyID, updates); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新API密钥失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新API密钥成功", nil))
}

// UpdateApiKeyApplications 更新ApiKey关联的应用
// @Summary 更新API密钥关联应用
// @Description 更新ApiKey关联的应用列表
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Param applications body UpdateApiKeyApplicationsRequest true "关联应用信息"
// @Success 200 {object} APIResponse "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-keys/{id}/applications [put]
func (c *SharingController) UpdateApiKeyApplications(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "id")

	var req UpdateApiKeyApplicationsRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.sharingService.UpdateApiKeyApplications(keyID, req.ApplicationIDs); err != nil {
		render.JSON(w, r, InternalErrorResponse("更新API密钥关联应用失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新API密钥关联应用成功", nil))
}

// DeleteApiKey 吊销（删除）一个ApiKey
// @Summary 删除API密钥
// @Description 吊销（删除）一个ApiKey
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-keys/{id} [delete]
func (c *SharingController) DeleteApiKey(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "id")

	if err := c.sharingService.DeleteApiKey(keyID); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除API密钥失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除API密钥成功", nil))
}

// === ApiInterface管理 ===

// CreateApiInterfaceRequest 创建ApiInterface请求结构
type CreateApiInterfaceRequest struct {
	ApiApplicationID    string `json:"api_application_id" validate:"required"`
	ThematicInterfaceID string `json:"thematic_interface_id" validate:"required"`
	Path                string `json:"path" validate:"required"`
	Description         string `json:"description"`
}

// CreateApiInterface 创建一个共享接口
// @Summary 创建共享接口
// @Description 创建一个共享接口，请求体包含 api_application_id, thematic_interface_id, path
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param interface_param body CreateApiInterfaceRequest true "接口信息"
// @Success 201 {object} APIResponse{data=models.ApiInterface} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-interfaces [post]
func (c *SharingController) CreateApiInterface(w http.ResponseWriter, r *http.Request) {
	var req CreateApiInterfaceRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	apiInterface := &models.ApiInterface{
		ApiApplicationID:    req.ApiApplicationID,
		ThematicInterfaceID: req.ThematicInterfaceID,
		Path:                req.Path,
		Description:         req.Description,
	}

	if err := c.sharingService.CreateApiInterface(apiInterface); err != nil {
		render.JSON(w, r, InternalErrorResponse("创建共享接口失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建共享接口成功", apiInterface))
}

// GetApiInterfaces 查询共享接口列表
// @Summary 获取共享接口列表
// @Description 查询共享接口列表，可按 api_application_id 过滤
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param api_application_id query string false "应用ID"
// @Success 200 {object} APIResponse{data=[]models.ApiInterface} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-interfaces [get]
func (c *SharingController) GetApiInterfaces(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("api_application_id")

	interfaces, err := c.sharingService.GetApiInterfaces(appID)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取共享接口列表失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取共享接口列表成功", interfaces))
}

// DeleteApiInterface 删除一个共享接口
// @Summary 删除共享接口
// @Description 删除一个共享接口
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param id path string true "接口ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sharing/api-interfaces/{id} [delete]
func (c *SharingController) DeleteApiInterface(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := c.sharingService.DeleteApiInterface(id); err != nil {
		render.JSON(w, r, InternalErrorResponse("删除共享接口失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除共享接口成功", nil))
}
