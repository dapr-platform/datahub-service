/*
 * @module api/controllers/thematic_sync_controller
 * @description 主题同步控制器，处理主题数据同步的HTTP接口
 * @architecture MVC架构 - 控制器层
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow HTTP请求处理流程，主题数据同步流程
 * @rules 统一的错误处理和响应格式，参数验证
 * @dependencies datahub-service/service, github.com/go-chi/render
 * @refs ai_docs/thematic_sync_design.md, service/thematic_sync_service.go
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/thematic_library"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ThematicSyncController 主题同步控制器
type ThematicSyncController struct {
	thematicSyncService *thematic_library.ThematicSyncService
}

// NewThematicSyncController 创建主题同步控制器实例 - 简化版本
func NewThematicSyncController() *ThematicSyncController {
	// 直接创建服务，无需复杂的适配器
	thematicSyncService := thematic_library.NewThematicSyncService(
		service.DB,
		service.GlobalGovernanceService,
	)
	return &ThematicSyncController{
		thematicSyncService: thematicSyncService,
	}
}

// SQLDataSourceConfig SQL数据源配置
type SQLDataSourceConfig struct {
	LibraryID   string                 `json:"library_id" example:"lib_001"`
	InterfaceID string                 `json:"interface_id" example:"interface_001"`
	SQLQuery    string                 `json:"sql_query" example:"SELECT * FROM users WHERE status = {{status}}"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" example:"{\"status\":\"active\"}"`
	Timeout     int                    `json:"timeout,omitempty" example:"30"`
	MaxRows     int                    `json:"max_rows,omitempty" example:"10000"`
}

// ExecuteSyncTaskRequest 执行同步任务请求结构
type ExecuteSyncTaskRequest struct {
	ExecutionType string                                 `json:"execution_type,omitempty" example:"manual"` // manual, auto
	Options       *thematic_library.SyncExecutionOptions `json:"options,omitempty"`                         // 执行选项
	ExecutedBy    string                                 `json:"executed_by" validate:"required" example:"admin"`
}

// SyncTaskListResponse 同步任务列表响应结构
type SyncTaskListResponse struct {
	List  []interface{} `json:"list"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

// SyncExecutionListResponse 同步执行记录列表响应结构
type SyncExecutionListResponse struct {
	List  []interface{} `json:"list"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

// @Summary 创建同步任务
// @Description 创建主题数据同步任务
// @Tags 主题同步
// @Accept json
// @Produce json
// @Param request body thematic_library.CreateThematicSyncTaskRequest true "创建同步任务请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks [post]
func (c *ThematicSyncController) CreateSyncTask(w http.ResponseWriter, r *http.Request) {
	var req thematic_library.CreateThematicSyncTaskRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	// 参数验证
	if req.TaskName == "" {
		render.JSON(w, r, BadRequestResponse("任务名称不能为空", nil))
		return
	}
	if req.ThematicLibraryID == "" {
		render.JSON(w, r, BadRequestResponse("主题库ID不能为空", nil))
		return
	}
	if req.ThematicInterfaceID == "" {
		render.JSON(w, r, BadRequestResponse("主题接口ID不能为空", nil))
		return
	}

	// 验证数据源配置 - 必须配置其中一种
	if len(req.DataSourceSQL) == 0 && len(req.SourceLibraries) == 0 {
		render.JSON(w, r, BadRequestResponse("必须配置SQL数据源或基础库源配置", nil))
		return
	}

	// 验证SQL数据源配置
	for i, sqlConfig := range req.DataSourceSQL {
		if sqlConfig.SQLQuery == "" {
			render.JSON(w, r, BadRequestResponse(fmt.Sprintf("第%d个SQL配置的查询语句不能为空", i+1), nil))
			return
		}
		if sqlConfig.LibraryID == "" {
			render.JSON(w, r, BadRequestResponse(fmt.Sprintf("第%d个SQL配置的库ID不能为空", i+1), nil))
			return
		}
		if sqlConfig.InterfaceID == "" {
			render.JSON(w, r, BadRequestResponse(fmt.Sprintf("第%d个SQL配置的接口ID不能为空", i+1), nil))
			return
		}
	}

	if req.ScheduleConfig == nil {
		render.JSON(w, r, BadRequestResponse("调度配置不能为空", nil))
		return
	}

	// 设置默认值
	if req.CreatedBy == "" {
		req.CreatedBy = "system"
	}

	// 转换SQL数据源配置到服务层类型
	var serviceDataSourceSQL []thematic_library.SQLDataSourceConfig
	for _, sqlConfig := range req.DataSourceSQL {
		serviceDataSourceSQL = append(serviceDataSourceSQL, thematic_library.SQLDataSourceConfig{
			LibraryID:   sqlConfig.LibraryID,
			InterfaceID: sqlConfig.InterfaceID,
			SQLQuery:    sqlConfig.SQLQuery,
			Parameters:  sqlConfig.Parameters,
			Timeout:     sqlConfig.Timeout,
			MaxRows:     sqlConfig.MaxRows,
		})
	}

	// 直接使用请求结构，只需要更新SQL数据源配置
	serviceReq := &req
	if len(serviceDataSourceSQL) > 0 {
		serviceReq.DataSourceSQL = serviceDataSourceSQL
	}

	task, err := c.thematicSyncService.CreateSyncTask(r.Context(), serviceReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("创建同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建同步任务成功", task))
}

// @Summary 获取同步任务列表
// @Description 分页获取主题同步任务列表，支持多种过滤条件
// @Tags 主题同步
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param name query string false "任务名称搜索"
// @Param status query string false "状态过滤" Enums(active,inactive,paused)
// @Param sync_mode query string false "同步模式过滤" Enums(manual,one_time,timed,cron)
// @Param thematic_library_id query string false "主题库ID过滤"
// @Success 200 {object} APIResponse{data=SyncTaskListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /thematic-sync/tasks [get]
func (c *ThematicSyncController) GetSyncTaskList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	status := r.URL.Query().Get("status")
	syncMode := r.URL.Query().Get("sync_mode")
	thematicLibraryID := r.URL.Query().Get("thematic_library_id")

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法 - 使用实际的服务接口
	listReq := &thematic_library.ListSyncTasksRequest{
		Page:              page,
		PageSize:          size,
		Status:            status,
		TriggerType:       syncMode,
		ThematicLibraryID: thematicLibraryID,
	}

	listResp, err := c.thematicSyncService.ListSyncTasks(r.Context(), listReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步任务列表失败", err))
		return
	}

	// 构建响应 - 转换为interface{}数组
	taskList := make([]interface{}, len(listResp.Tasks))
	for i, task := range listResp.Tasks {
		taskList[i] = task
	}

	response := SyncTaskListResponse{
		List:  taskList,
		Total: listResp.Total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取同步任务列表成功", response))
}

// @Summary 获取同步任务详情
// @Description 获取指定ID的同步任务详细信息
// @Tags 主题同步
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/{id} [get]
func (c *ThematicSyncController) GetSyncTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	task, err := c.thematicSyncService.GetSyncTask(r.Context(), id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步任务详情失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务详情成功", task))
}

// @Summary 更新同步任务
// @Description 更新指定ID的同步任务信息
// @Tags 主题同步
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body thematic_library.UpdateThematicSyncTaskRequest true "更新同步任务请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/{id} [put]
func (c *ThematicSyncController) UpdateSyncTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	var req thematic_library.UpdateThematicSyncTaskRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	// 设置默认值
	if req.UpdatedBy == "" {
		req.UpdatedBy = "system"
	}

	// 直接使用服务层请求结构
	serviceReq := &req

	task, err := c.thematicSyncService.UpdateSyncTask(r.Context(), id, serviceReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("更新同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新同步任务成功", task))
}

// @Summary 删除同步任务
// @Description 删除指定ID的同步任务
// @Tags 主题同步
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/{id} [delete]
func (c *ThematicSyncController) DeleteSyncTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	err := c.thematicSyncService.DeleteSyncTask(r.Context(), id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("删除同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除同步任务成功", nil))
}

// @Summary 执行同步任务
// @Description 立即执行指定的同步任务
// @Tags 主题同步
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body ExecuteSyncTaskRequest true "执行同步任务请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/{id}/execute [post]
func (c *ThematicSyncController) ExecuteSyncTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	var req ExecuteSyncTaskRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	// 设置默认值
	if req.ExecutedBy == "" {
		req.ExecutedBy = "system"
	}
	if req.ExecutionType == "" {
		req.ExecutionType = "manual"
	}

	// 转换为服务层请求结构
	execReq := &thematic_library.ExecuteSyncTaskRequest{
		ExecutionType: req.ExecutionType,
		Options:       req.Options,
	}

	syncResult, err := c.thematicSyncService.ExecuteSyncTask(r.Context(), id, execReq)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("执行同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("执行同步任务成功", syncResult))
}

// @Summary 停止同步任务
// @Description 停止正在执行的同步任务
// @Tags 主题同步
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/{id}/stop [post]
func (c *ThematicSyncController) StopSyncTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	err := c.thematicSyncService.StopSyncTask(r.Context(), id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("停止同步任务失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("停止同步任务成功", nil))
}

// @Summary 获取同步任务状态
// @Description 获取指定同步任务的当前状态信息
// @Tags 主题同步
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/{id}/status [get]
func (c *ThematicSyncController) GetSyncTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	status, err := c.thematicSyncService.GetSyncTaskStatus(r.Context(), id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步任务状态失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务状态成功", status))
}

// @Summary 获取同步执行记录列表
// @Description 分页获取指定任务的同步执行记录列表
// @Tags 主题同步
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param status query string false "执行状态过滤" Enums(running,completed,failed,cancelled)
// @Success 200 {object} APIResponse{data=SyncExecutionListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /thematic-sync/tasks/{id}/executions [get]
func (c *ThematicSyncController) GetSyncTaskExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("任务ID参数不能为空", nil))
		return
	}

	// 解析查询参数
	page := 1
	size := 10
	status := r.URL.Query().Get("status")

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	executions, total, err := c.thematicSyncService.GetSyncTaskExecutions(r.Context(), id, page, size, status)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步执行记录失败", err))
		return
	}

	// 构建响应
	response := SyncExecutionListResponse{
		List:  executions,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取同步执行记录成功", response))
}

// @Summary 获取同步执行记录详情
// @Description 获取指定执行记录的详细信息
// @Tags 主题同步
// @Produce json
// @Param id path string true "执行记录ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/executions/{id} [get]
func (c *ThematicSyncController) GetSyncExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("执行记录ID参数不能为空", nil))
		return
	}

	execution, err := c.thematicSyncService.GetSyncExecution(r.Context(), id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步执行记录详情失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步执行记录详情成功", execution))
}

// @Summary 获取同步任务统计信息
// @Description 获取主题同步任务的统计信息，包括总数、状态分布等
// @Tags 主题同步
// @Produce json
// @Param thematic_library_id query string false "主题库ID过滤"
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-sync/tasks/statistics [get]
func (c *ThematicSyncController) GetSyncTaskStatistics(w http.ResponseWriter, r *http.Request) {
	thematicLibraryID := r.URL.Query().Get("thematic_library_id")

	stats, err := c.thematicSyncService.GetSyncTaskStatistics(r.Context(), thematicLibraryID)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取同步任务统计信息失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取同步任务统计信息成功", stats))
}
