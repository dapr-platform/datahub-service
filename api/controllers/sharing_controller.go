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
	Name          string  `json:"name" validate:"required"`
	AppSecret     string  `json:"app_secret" validate:"required"`
	Description   *string `json:"description"`
	ContactPerson string  `json:"contact_person" validate:"required"`
	ContactEmail  string  `json:"contact_email" validate:"required,email"`
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	app := &models.ApiApplication{
		Name:          req.Name,
		Description:   req.Description,
		ContactPerson: req.ContactPerson,
		ContactEmail:  req.ContactEmail,
	}

	if err := c.sharingService.CreateApiApplication(app, req.AppSecret); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "创建API应用失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusCreated,
		Msg:    "创建API应用成功",
		Data:   app,
	})
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
// @Success 200 {object} PaginatedResponse{data=[]models.ApiApplication} "获取成功"
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "获取API应用列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "获取API应用列表成功",
		Data:   apps,
		Total:  total,
		Page:   page,
		Size:   size,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "API应用不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "获取API应用成功",
		Data:   app,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	if err := c.sharingService.UpdateApiApplication(id, updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "更新API应用失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "更新API应用成功",
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "删除API应用失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "删除API应用成功",
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	if err := c.sharingService.CreateApiRateLimit(&limit); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "创建API限流规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusCreated,
		Msg:    "创建API限流规则成功",
		Data:   limit,
	})
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
// @Success 200 {object} PaginatedResponse{data=[]models.ApiRateLimit} "获取成功"
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "获取API限流规则列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "获取API限流规则列表成功",
		Data:   limits,
		Total:  total,
		Page:   page,
		Size:   size,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	if err := c.sharingService.UpdateApiRateLimit(id, updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "更新API限流规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "更新API限流规则成功",
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "删除API限流规则失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "删除API限流规则成功",
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	if err := c.sharingService.CreateDataSubscription(&subscription); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "创建数据订阅失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusCreated,
		Msg:    "创建数据订阅成功",
		Data:   subscription,
	})
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
// @Success 200 {object} PaginatedResponse{data=[]models.DataSubscription} "获取成功"
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "获取数据订阅列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "获取数据订阅列表成功",
		Data:   subscriptions,
		Total:  total,
		Page:   page,
		Size:   size,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "数据订阅不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "获取数据订阅成功",
		Data:   subscription,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	if err := c.sharingService.UpdateDataSubscription(id, updates); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "更新数据订阅失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "更新数据订阅成功",
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "删除数据订阅失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "删除数据订阅成功",
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	if err := c.sharingService.CreateDataAccessRequest(&request); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "创建数据使用申请失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusCreated,
		Msg:    "创建数据使用申请成功",
		Data:   request,
	})
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
// @Success 200 {object} PaginatedResponse{data=[]models.DataAccessRequest} "获取成功"
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "获取数据使用申请列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "获取数据使用申请列表成功",
		Data:   requests,
		Total:  total,
		Page:   page,
		Size:   size,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "数据使用申请不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "获取数据使用申请成功",
		Data:   request,
	})
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	// TODO: 从认证信息中获取审批人ID
	approverID := "system" // 临时使用系统ID

	if err := c.sharingService.ApproveDataAccessRequest(id, approverID, req.Approved, req.Comment); err != nil {
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "审批数据使用申请失败",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "审批数据使用申请成功",
	})
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
// @Success 200 {object} PaginatedResponse{data=[]models.ApiUsageLog} "获取成功"
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
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "获取API使用日志列表失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "获取API使用日志列表成功",
		Data:   logs,
		Total:  total,
		Page:   page,
		Size:   size,
	})
}
