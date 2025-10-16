/*
 * @module api/controllers/sync_task_controller
 * @description 基础库同步任务控制器，专门处理基础库数据同步任务管理
 * @architecture 分层架构 - 控制器层
 * @documentReference ai_docs/refactor_sync_task.md
 * @stateFlow HTTP请求 -> 参数验证 -> 服务调用 -> 响应返回
 * @rules 专门支持基础库同步任务，不支持主题库同步
 * @dependencies service/basic_library/sync_task_service, service/models, service/meta
 * @refs api/routes.go
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/basic_library"
	"datahub-service/service/meta"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// SyncTaskController 基础库同步任务控制器
type SyncTaskController struct {
	syncTaskService *basic_library.SyncTaskService
}

// NewSyncTaskController 创建基础库同步任务控制器
func NewSyncTaskController() *SyncTaskController {
	return &SyncTaskController{
		syncTaskService: service.GlobalSyncTaskService,
	}
}

// SyncTaskInterfaceConfig 接口级别的配置
type SyncTaskInterfaceConfig struct {
	InterfaceID string                 `json:"interface_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Config      map[string]interface{} `json:"config,omitempty"` // 接口级别的特殊配置
}

// SyncTaskCreateRequest 创建基础库同步任务请求
type SyncTaskCreateRequest struct {
	LibraryID        string                    `json:"library_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	DataSourceID     string                    `json:"data_source_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	InterfaceIDs     []string                  `json:"interface_ids" binding:"required,min=1" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
	InterfaceConfigs []SyncTaskInterfaceConfig `json:"interface_configs,omitempty"` // 接口级别的配置，可选
	TaskType         string                    `json:"task_type" binding:"required" example:"batch_sync"`
	TriggerType      string                    `json:"trigger_type" binding:"required" example:"manual"`
	CronExpression   string                    `json:"cron_expression,omitempty" example:"0 0 * * *"`
	IntervalSeconds  int                       `json:"interval_seconds,omitempty" example:"3600"`
	ScheduledTime    *string                   `json:"scheduled_time,omitempty" example:"2024-01-01T00:00:00Z"`
	Config           map[string]interface{}    `json:"config,omitempty"` // 任务级别的全局配置
	CreatedBy        string                    `json:"created_by" example:"admin"`

	// 向后兼容字段（已废弃，但保留以支持旧版本API）
	InterfaceID *string `json:"interface_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// SyncTaskUpdateRequest 更新同步任务请求
type SyncTaskUpdateRequest struct {
	Status           string                    `json:"status,omitempty" example:"active"` // draft, active, paused
	TriggerType      string                    `json:"trigger_type,omitempty" example:"manual"`
	CronExpression   string                    `json:"cron_expression,omitempty" example:"0 0 * * *"`
	IntervalSeconds  int                       `json:"interval_seconds,omitempty" example:"3600"`
	ScheduledTime    *string                   `json:"scheduled_time,omitempty" example:"2024-01-01T00:00:00Z"`
	Config           map[string]interface{}    `json:"config,omitempty"`            // 任务级别的全局配置
	InterfaceIDs     []string                  `json:"interface_ids,omitempty"`     // 更新接口列表
	InterfaceConfigs []SyncTaskInterfaceConfig `json:"interface_configs,omitempty"` // 更新接口级别的配置
	UpdatedBy        string                    `json:"updated_by" example:"admin"`
	TaskType         string                    `json:"task_type,omitempty" example:"batch_sync"`
}

// SyncTaskListRequest 基础库同步任务列表请求
type SyncTaskListRequest struct {
	Page         int    `json:"page" example:"1"`
	Size         int    `json:"size" example:"10"`
	LibraryID    string `json:"library_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	DataSourceID string `json:"data_source_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status       string `json:"status,omitempty" example:"pending"`
	TaskType     string `json:"task_type,omitempty" example:"batch_sync"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required"`
}

// SyncTaskExecutionListRequest 同步任务执行记录列表请求
type SyncTaskExecutionListRequest struct {
	Page          int    `json:"page" example:"1"`
	Size          int    `json:"size" example:"10"`
	TaskID        string `json:"task_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status        string `json:"status,omitempty" example:"success"`
	ExecutionType string `json:"execution_type,omitempty" example:"manual"`
}

// CreateSyncTask 创建基础库同步任务
// @Summary 创建基础库同步任务
// @Description 创建新的基础库数据同步任务，专门处理基础库数据同步
// @Description
// @Description **支持的任务类型:**
// @Description - batch_sync: 批量同步（根据接口配置自动判断全量/增量）
// @Description - realtime_sync: 实时同步
// @Description
// @Description **任务生命周期状态:**
// @Description - draft: 草稿，正在编辑，不参与调度
// @Description - active: 激活，已配置完成，参与调度
// @Description - paused: 暂停，临时停止调度，但保留配置
// @Description
// @Description **任务执行状态:**
// @Description - idle: 空闲，等待下次执行
// @Description - running: 执行中，正在执行同步
// @Description - success: 成功，最近一次执行成功
// @Description - failed: 失败，最近一次执行失败
// @Description
// @Description **状态流转:**
// @Description 生命周期: draft → active ↔ paused
// @Description 执行状态: idle → running → success/failed → idle
// @Description
// @Description **注意:**
// @Description - 新创建的任务默认为 draft 状态，需要手动激活后才会参与调度
// @Description - 此接口仅支持基础库同步任务，不支持主题库同步
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param task body SyncTaskCreateRequest true "基础库同步任务创建信息"
// @Success 200 {object} APIResponse{data=models.SyncTask} "创建成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks [post]
func (c *SyncTaskController) CreateSyncTask(w http.ResponseWriter, r *http.Request) {
	var req SyncTaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数解析失败", err))
		return
	}

	// 验证任务类型
	if !meta.IsValidSyncType(req.TaskType) {
		render.JSON(w, r, BadRequestResponse("无效的任务类型", nil))
		return
	}

	// 验证执行时机类型
	if !meta.IsValidSyncTaskTrigger(req.TriggerType) {
		render.JSON(w, r, BadRequestResponse("无效的执行时机类型", nil))
		return
	}

	// 向后兼容处理：如果使用了旧的InterfaceID字段，转换为新格式
	var interfaceIDs []string
	var interfaceConfigs []basic_library.SyncTaskInterfaceConfig

	if req.InterfaceID != nil && *req.InterfaceID != "" && len(req.InterfaceIDs) == 0 {
		// 使用旧格式
		interfaceIDs = []string{*req.InterfaceID}
	} else if len(req.InterfaceIDs) > 0 {
		// 使用新格式
		interfaceIDs = req.InterfaceIDs
	} else {
		render.JSON(w, r, BadRequestResponse("必须提供至少一个接口ID", nil))
		return
	}

	// 转换接口配置
	if len(req.InterfaceConfigs) > 0 {
		for _, config := range req.InterfaceConfigs {
			interfaceConfigs = append(interfaceConfigs, basic_library.SyncTaskInterfaceConfig{
				InterfaceID: config.InterfaceID,
				Config:      config.Config,
			})
		}
	}

	// 解析计划执行时间
	var scheduledTime *time.Time
	if req.ScheduledTime != nil && *req.ScheduledTime != "" {
		if parsedTime, err := time.Parse(time.RFC3339, *req.ScheduledTime); err != nil {
			render.JSON(w, r, BadRequestResponse("无效的计划执行时间格式", err))
			return
		} else {
			scheduledTime = &parsedTime
		}
	}

	// 创建服务请求（基础库同步任务）
	serviceReq := &basic_library.CreateSyncTaskRequest{
		LibraryType:      meta.LibraryTypeBasic, // 固定为基础库类型
		LibraryID:        req.LibraryID,
		DataSourceID:     req.DataSourceID,
		InterfaceIDs:     interfaceIDs,
		InterfaceConfigs: interfaceConfigs,
		TaskType:         req.TaskType,
		TriggerType:      req.TriggerType,
		CronExpression:   req.CronExpression,
		IntervalSeconds:  req.IntervalSeconds,
		ScheduledTime:    scheduledTime,
		Config:           req.Config,
		CreatedBy:        req.CreatedBy,
	}

	task, err := c.syncTaskService.CreateSyncTask(r.Context(), serviceReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("创建同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建同步任务成功", task))
}

// GetSyncTaskList 获取基础库同步任务列表
// @Summary 获取基础库同步任务列表
// @Description 分页获取基础库同步任务列表，支持多种过滤条件
// @Description
// @Description **查询参数说明:**
// @Description - page: 页码，默认1
// @Description - size: 每页大小，默认10，最大100
// @Description - library_id: 基础库ID过滤
// @Description - data_source_id: 数据源ID过滤
// @Description - status: 任务状态过滤
// @Description - task_type: 任务类型过滤
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param library_id query string false "基础库ID"
// @Param data_source_id query string false "数据源ID"
// @Param status query string false "任务状态"
// @Param task_type query string false "任务类型"
// @Success 200 {object} APIResponse{data=basic_library.SyncTaskListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks [get]
func (c *SyncTaskController) GetSyncTaskList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	req := SyncTaskListRequest{
		Page:         1,
		Size:         10,
		LibraryID:    r.URL.Query().Get("library_id"),
		DataSourceID: r.URL.Query().Get("data_source_id"),
		Status:       r.URL.Query().Get("status"),
		TaskType:     r.URL.Query().Get("task_type"),
	}

	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && page > 0 {
		req.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && size > 0 && size <= 100 {
		req.Size = size
	}

	// 创建服务请求（基础库同步任务）
	serviceReq := &basic_library.GetSyncTaskListRequest{
		Page:         req.Page,
		Size:         req.Size,
		LibraryType:  meta.LibraryTypeBasic, // 固定为基础库类型
		LibraryID:    req.LibraryID,
		DataSourceID: req.DataSourceID,
		Status:       req.Status,
		TaskType:     req.TaskType,
	}

	response, err := c.syncTaskService.GetSyncTaskList(r.Context(), serviceReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步任务列表失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务列表成功", response))
}

// GetSyncTask 获取同步任务详情
// @Summary 获取同步任务详情
// @Description 根据任务ID获取同步任务的详细信息，包括关联的库信息、数据源信息等
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=models.SyncTask} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id} [get]
func (c *SyncTaskController) GetSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, BadRequestResponse("任务ID不能为空", nil))
		return
	}

	task, err := c.syncTaskService.GetSyncTaskByID(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, NotFoundResponse("获取同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务成功", task))
}

// UpdateSyncTask 更新同步任务
// @Summary 更新同步任务
// @Description 更新同步任务的配置信息和状态
// @Description
// @Description **可更新状态:**
// @Description - draft 和 paused 状态的任务可以更新任何配置
// @Description - active 状态且非运行中的任务也可以更新
// @Description - running 状态的任务不允许更新
// @Description
// @Description **状态变更:**
// @Description - 可以通过 status 字段变更任务状态（draft/active/paused）
// @Description - 状态变更会自动触发调度器的添加/移除操作
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param task body SyncTaskUpdateRequest true "更新信息"
// @Success 200 {object} APIResponse{data=models.SyncTask} "更新成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许更新"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id} [put]
func (c *SyncTaskController) UpdateSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	var req SyncTaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数解析失败", err))
		return
	}

	// 转换接口配置
	var interfaceConfigs []basic_library.SyncTaskInterfaceConfig
	if len(req.InterfaceConfigs) > 0 {
		for _, config := range req.InterfaceConfigs {
			interfaceConfigs = append(interfaceConfigs, basic_library.SyncTaskInterfaceConfig{
				InterfaceID: config.InterfaceID,
				Config:      config.Config,
			})
		}
	}

	// 创建更新请求
	updateReq := &basic_library.UpdateSyncTaskRequest{
		Status:           req.Status,
		TriggerType:      req.TriggerType,
		CronExpression:   req.CronExpression,
		IntervalSeconds:  req.IntervalSeconds,
		Config:           req.Config,
		InterfaceIDs:     req.InterfaceIDs,
		InterfaceConfigs: interfaceConfigs,
		UpdatedBy:        req.UpdatedBy,
		TaskType:         req.TaskType,
	}

	task, err := c.syncTaskService.UpdateSyncTask(r.Context(), taskID, updateReq)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "更新同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新同步任务成功", task))
}

// DeleteSyncTask 删除同步任务
// @Summary 删除同步任务
// @Description 删除指定的同步任务，只能删除已完成、失败或已取消的任务
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "删除成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许删除"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id} [delete]
func (c *SyncTaskController) DeleteSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.DeleteSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "删除同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除同步任务成功", nil))
}

// StartSyncTask 启动同步任务
// @Summary 启动同步任务
// @Description 启动指定的同步任务，将任务提交给同步引擎执行
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "启动成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许启动"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/start [post]
func (c *SyncTaskController) StartSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.StartSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "启动同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("启动同步任务成功", nil))
}

// StopSyncTask 停止同步任务
// @Summary 停止同步任务
// @Description 停止正在执行的同步任务
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "停止成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许停止"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/stop [post]
func (c *SyncTaskController) StopSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.StopSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "停止同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("停止同步任务成功", nil))
}

// CancelSyncTask 暂停同步任务（保留向后兼容）
// @Summary 暂停同步任务
// @Description 暂停指定的同步任务，将 active 状态改为 paused，并从调度器中移除
// @Description
// @Description **注意:**
// @Description - 此接口已重命名为 PauseSyncTask，但保留 cancel 路由以保持向后兼容
// @Description - 建议使用 /sync/tasks/{id}/pause 接口
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "暂停成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许暂停"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/cancel [post]
func (c *SyncTaskController) CancelSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.CancelSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "暂停同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("暂停同步任务成功", nil))
}

// RetrySyncTask 重试同步任务
// @Summary 重试同步任务
// @Description 重试失败的同步任务，会创建一个新的任务实例
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=models.SyncTask} "重试成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许重试"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/retry [post]
func (c *SyncTaskController) RetrySyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	newTask, err := c.syncTaskService.RetrySyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "重试同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("重试同步任务成功", newTask))
}

// GetSyncTaskStatus 获取同步任务状态
// @Summary 获取同步任务状态
// @Description 获取同步任务的实时执行状态和进度信息
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=basic_library.SyncTaskStatusResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/status [get]
func (c *SyncTaskController) GetSyncTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	status, err := c.syncTaskService.GetSyncTaskStatus(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "获取同步任务状态失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务状态成功", status))
}

// BatchDeleteSyncTasks 批量删除同步任务
// @Summary 批量删除同步任务
// @Description 批量删除多个同步任务，只能删除已完成、失败或已取消的任务
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param tasks body BatchDeleteRequest true "批量删除请求"
// @Success 200 {object} APIResponse{data=basic_library.BatchDeleteResponse} "删除成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/batch-delete [post]
func (c *SyncTaskController) BatchDeleteSyncTasks(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数解析失败", err))
		return
	}

	if len(req.TaskIDs) == 0 {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID列表不能为空", nil))
		return
	}

	response, err := c.syncTaskService.BatchDeleteSyncTasks(r.Context(), req.TaskIDs)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "批量删除同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("批量删除同步任务成功", response))
}

// GetSyncTaskStatistics 获取同步任务统计信息
// @Summary 获取同步任务统计信息
// @Description 获取同步任务的统计数据，包括各状态任务数量、成功率等
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param library_id query string false "基础库ID过滤"
// @Param data_source_id query string false "数据源ID过滤"
// @Success 200 {object} APIResponse{data=basic_library.SyncTaskStatistics} "获取成功"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/statistics [get]
func (c *SyncTaskController) GetSyncTaskStatistics(w http.ResponseWriter, r *http.Request) {
	libraryID := r.URL.Query().Get("library_id")
	dataSourceID := r.URL.Query().Get("data_source_id")

	// 固定为基础库类型
	statistics, err := c.syncTaskService.GetSyncTaskStatistics(r.Context(), meta.LibraryTypeBasic, libraryID, dataSourceID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "获取同步任务统计信息失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务统计信息成功", statistics))
}

// GetSyncTaskExecutions 获取同步任务执行记录列表
// @Summary 获取同步任务执行记录列表
// @Description 分页获取同步任务执行记录列表，支持多种过滤条件
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param task_id query string false "任务ID"
// @Param status query string false "执行状态"
// @Param execution_type query string false "执行类型"
// @Success 200 {object} APIResponse{data=basic_library.SyncTaskExecutionListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/executions [get]
func (c *SyncTaskController) GetSyncTaskExecutions(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	req := SyncTaskExecutionListRequest{
		Page:          1,
		Size:          10,
		TaskID:        r.URL.Query().Get("task_id"),
		Status:        r.URL.Query().Get("status"),
		ExecutionType: r.URL.Query().Get("execution_type"),
	}

	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && page > 0 {
		req.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && size > 0 && size <= 100 {
		req.Size = size
	}

	// 创建服务请求
	serviceReq := &basic_library.GetSyncTaskExecutionListRequest{
		Page:          req.Page,
		Size:          req.Size,
		TaskID:        req.TaskID,
		Status:        req.Status,
		ExecutionType: req.ExecutionType,
	}

	response, err := c.syncTaskService.GetSyncTaskExecutionList(r.Context(), serviceReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步任务执行记录列表失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务执行记录列表成功", response))
}

// GetSyncTaskExecution 获取同步任务执行记录详情
// @Summary 获取同步任务执行记录详情
// @Description 根据执行记录ID获取同步任务执行记录的详细信息
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "执行记录ID"
// @Success 200 {object} APIResponse{data=models.SyncTaskExecution} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "执行记录不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/executions/{id} [get]
func (c *SyncTaskController) GetSyncTaskExecution(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "id")
	if executionID == "" {
		render.JSON(w, r, BadRequestResponse("执行记录ID不能为空", nil))
		return
	}

	execution, err := c.syncTaskService.GetSyncTaskExecutionByID(r.Context(), executionID)
	if err != nil {
		render.JSON(w, r, NotFoundResponse("获取同步任务执行记录失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务执行记录成功", execution))
}

// GetTaskExecutions 获取指定任务的执行记录
// @Summary 获取指定任务的执行记录
// @Description 获取指定同步任务的所有执行记录
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Success 200 {object} APIResponse{data=basic_library.SyncTaskExecutionListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/executions [get]
func (c *SyncTaskController) GetTaskExecutions(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, BadRequestResponse("任务ID不能为空", nil))
		return
	}

	// 解析查询参数
	page := 1
	size := 10
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 创建服务请求
	serviceReq := &basic_library.GetSyncTaskExecutionListRequest{
		Page:   page,
		Size:   size,
		TaskID: taskID,
	}

	response, err := c.syncTaskService.GetSyncTaskExecutionList(r.Context(), serviceReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取任务执行记录列表失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取任务执行记录列表成功", response))
}

// ActivateSyncTask 激活同步任务
// @Summary 激活同步任务
// @Description 激活同步任务，将 draft 或 paused 状态改为 active，并添加到调度器
// @Description
// @Description **状态转换:**
// @Description - draft → active: 首次激活任务
// @Description - paused → active: 恢复已暂停的任务
// @Description
// @Description **注意:**
// @Description - 只有 draft 或 paused 状态的任务可以激活
// @Description - 激活后，如果配置了调度（cron/interval/once），任务会自动加入调度器
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "激活成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许激活"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/activate [post]
func (c *SyncTaskController) ActivateSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.ActivateSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "激活同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("激活同步任务成功", nil))
}

// PauseSyncTask 暂停同步任务
// @Summary 暂停同步任务
// @Description 暂停同步任务，将 active 状态改为 paused，并从调度器中移除
// @Description
// @Description **状态转换:**
// @Description - active → paused: 暂停任务调度
// @Description
// @Description **注意:**
// @Description - 只有 active 状态的任务可以暂停
// @Description - 暂停后，任务会从调度器中移除，不再自动执行
// @Description - 如果任务正在运行，会先停止执行，然后更新为 paused 状态
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "暂停成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许暂停"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/pause [post]
func (c *SyncTaskController) PauseSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.PauseSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "暂停同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("暂停同步任务成功", nil))
}

// ResumeSyncTask 恢复同步任务
// @Summary 恢复同步任务
// @Description 恢复已暂停的同步任务，将 paused 状态改为 active，并重新添加到调度器
// @Description
// @Description **状态转换:**
// @Description - paused → active: 恢复任务调度
// @Description
// @Description **注意:**
// @Description - 只有 paused 状态的任务可以恢复
// @Description - 恢复后，任务会重新加入调度器
// @Description - 此接口等同于 ActivateSyncTask，但语义更明确
// @Tags 基础库同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse "恢复成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 404 {object} APIResponse "任务不存在"
// @Failure 409 {object} APIResponse "任务状态不允许恢复"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /sync/tasks/{id}/resume [post]
func (c *SyncTaskController) ResumeSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "任务ID不能为空", nil))
		return
	}

	err := c.syncTaskService.ResumeSyncTask(r.Context(), taskID)
	if err != nil {
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "恢复同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("恢复同步任务成功", nil))
}
