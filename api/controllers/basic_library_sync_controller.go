/*
 * @module api/controllers/sync_controller
 * @description 数据同步控制器，提供数据同步任务的管理和监控功能
 * @architecture MVC架构 - 控制器层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 请求接收 -> 业务逻辑处理 -> 响应返回
 * @rules 确保同步任务的安全启停和状态管理，提供详细的同步监控信息
 * @dependencies net/http, strconv, time
 * @refs service/sync_engine/, service/scheduler/
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/basic_library"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"datahub-service/service/sync_engine"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// SyncController 数据同步控制器
type SyncController struct {
	scheduleService *basic_library.ScheduleService
	syncEngine      *sync_engine.SyncEngine
}

// NewSyncController 创建数据同步控制器实例
func NewSyncController() *SyncController {
	return &SyncController{
		scheduleService: service.GlobalBasicLibraryService.GetScheduleService(),
		syncEngine:      service.GlobalSyncEngine,
	}
}

// CreateSyncTask 创建同步任务
// @Summary 创建数据同步任务
// @Description 创建新的数据同步任务并提交到执行引擎
// @Description
// @Description **任务状态流转规则:**
// @Description - 创建任务 → pending (待执行)
// @Description - 开始执行 → running (执行中)
// @Description - 执行完成 → success (成功) 或 failed (失败)
// @Description - 任务取消 → cancelled (已取消)
// @Description - 失败重试 → failed → pending (重新待执行)
// @Description
// @Description **状态约束:**
// @Description - 只有 pending 状态的任务可以直接取消
// @Description - 只有 running 状态的任务可以停止
// @Description - 只有 failed 状态的任务可以重试
// @Description - 只有 pending 状态的任务可以更新配置
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param task body CreateSyncTaskRequest true "同步任务信息"
// @Success 200 {object} APIResponse{data=models.SyncTask}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks [post]
func (c *SyncController) CreateSyncTask(w http.ResponseWriter, r *http.Request) {
	var request CreateSyncTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "请求参数解析失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证必要参数
	if request.DataSourceID == "" {
		response := &APIResponse{
			Status: 1,
			Msg:    "数据源ID不能为空",
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if request.TaskType == "" {
		request.TaskType = meta.SyncTaskTypeFullSync // 默认全量同步
	}

	// 使用ScheduleService创建同步任务
	task, err := c.scheduleService.CreateSyncTask(request.DataSourceID, request.InterfaceID, request.TaskType, request.Parameters)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "创建同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 通知SyncEngine执行任务
	syncRequest := &models.SyncTaskRequest{
		DataSourceID: request.DataSourceID,
		InterfaceID:  request.InterfaceID,
		SyncType:     models.SyncType(request.TaskType),
		Config:       request.Parameters,
		ScheduledBy:  "manual", // 手动创建
	}

	if _, err := c.syncEngine.SubmitSyncTask(syncRequest); err != nil {
		// 记录错误但不阻塞响应，任务已经创建成功
		fmt.Printf("提交任务到执行引擎失败: %v\n", err)
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务创建成功",
		Data:   task,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSyncTasks 获取同步任务列表
// @Summary 获取同步任务列表
// @Description 分页获取同步任务列表，支持筛选
// @Description
// @Description **任务状态说明:**
// @Description - pending: 待执行
// @Description - running: 执行中
// @Description - success: 执行成功
// @Description - failed: 执行失败
// @Description - cancelled: 已取消
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param data_source_id query string false "数据源ID"
// @Param status query string false "任务状态(pending/running/success/failed/cancelled)"
// @Success 200 {object} APIResponse{data=GetSyncTasksResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks [get]
func (c *SyncController) GetSyncTasks(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}
	dataSourceID := r.URL.Query().Get("data_source_id")
	status := r.URL.Query().Get("status")

	// 使用ScheduleService获取任务列表
	tasks, total, err := c.scheduleService.GetSyncTasks(dataSourceID, status, page, size)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "获取同步任务列表失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 计算总页数
	totalPages := (total + int64(size) - 1) / int64(size)

	result := GetSyncTasksResponse{
		Tasks: tasks,
		Pagination: PaginationResponse{
			Page:       page,
			Size:       size,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "获取同步任务列表成功",
		Data:   result,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSyncTask 获取同步任务详情
// @Summary 获取同步任务详情
// @Description 获取指定任务的详细信息，包括实时进度信息（如果任务正在运行）
// @Description
// @Description **实时信息说明:**
// @Description - 对于 running 状态的任务，会显示实时进度、处理行数、错误计数等
// @Description - 对于其他状态的任务，显示最终结果信息
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=models.SyncTask}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id} [get]
func (c *SyncController) GetSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 使用ScheduleService获取任务详情
	task, err := c.scheduleService.GetSyncTask(taskID)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "获取同步任务详情失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 如果任务正在运行，合并实时进度信息
	if task.Status == meta.SyncTaskStatusRunning {
		if taskContext, err := c.syncEngine.GetTaskStatus(taskID); err == nil && taskContext.Progress != nil {
			task.Progress = taskContext.Progress.ProgressPercent
			task.ProcessedRows = taskContext.Progress.ProcessedRows
			task.TotalRows = taskContext.Progress.TotalRows
			task.ErrorCount = taskContext.Progress.ErrorCount
		}
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "获取同步任务详情成功",
		Data:   task,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateSyncTask 更新同步任务
// @Summary 更新数据同步任务
// @Description 更新指定的数据同步任务配置信息
// @Description
// @Description **状态约束:**
// @Description - 只有 pending 状态的任务可以更新配置
// @Description - running、success、failed、cancelled 状态的任务不允许更新
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param task body UpdateSyncTaskRequest true "任务更新信息"
// @Success 200 {object} APIResponse{data=models.SyncTask}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id} [put]
func (c *SyncController) UpdateSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var request UpdateSyncTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "请求参数解析失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 使用ScheduleService更新任务
	task, err := c.scheduleService.UpdateSyncTask(taskID, request.Config)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "任务不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "只有待执行状态的任务可以更新" {
			status = http.StatusBadRequest
		}

		response := &APIResponse{
			Status: 1,
			Msg:    "更新同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务更新成功",
		Data:   task,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CancelSyncTask 取消同步任务
// @Summary 取消数据同步任务
// @Description 取消指定的数据同步任务
// @Description
// @Description **状态约束:**
// @Description - 只有 pending 和 running 状态的任务可以取消
// @Description - success、failed、cancelled 状态的任务不允许取消
// @Description - 取消后任务状态变为 cancelled
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id}/cancel [post]
func (c *SyncController) CancelSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 使用ScheduleService取消任务
	err := c.scheduleService.CancelSyncTask(taskID)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "取消同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 如果任务正在执行，通知SyncEngine取消
	if err := c.syncEngine.CancelTask(taskID); err != nil {
		// 记录错误但不阻塞响应
		fmt.Printf("通知执行引擎取消任务失败: %v\n", err)
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务取消成功",
		Data:   nil,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RetryTask 重试同步任务
// @Summary 重试失败的同步任务
// @Description 重试指定的失败同步任务，会创建一个新的任务副本
// @Description
// @Description **状态约束:**
// @Description - 只有 failed 状态的任务可以重试
// @Description - 重试会创建新任务，状态为 pending
// @Description - 原任务状态保持 failed 不变
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=models.SyncTask}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id}/retry [post]
func (c *SyncController) RetryTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 使用ScheduleService重试任务
	newTask, err := c.scheduleService.RetryTask(taskID)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "重试同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 通知SyncEngine执行新的重试任务
	syncRequest := &models.SyncTaskRequest{
		DataSourceID: newTask.DataSourceID,
		InterfaceID:  "",
		SyncType:     models.SyncType(newTask.TaskType),
		Config:       newTask.Config,
		ScheduledBy:  "retry",
	}

	if newTask.InterfaceID != nil {
		syncRequest.InterfaceID = *newTask.InterfaceID
	}

	if _, err := c.syncEngine.SubmitSyncTask(syncRequest); err != nil {
		fmt.Printf("提交重试任务到执行引擎失败: %v\n", err)
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务重试成功",
		Data:   newTask,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DeleteSyncTask 删除同步任务
// @Summary 删除数据同步任务
// @Description 删除指定的数据同步任务，只允许删除已结束的任务
// @Description
// @Description **状态约束:**
// @Description - 只能删除 success、failed、cancelled 状态的任务
// @Description - pending 和 running 状态的任务不允许删除
// @Description - 删除操作不可逆，请谨慎操作
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id} [delete]
func (c *SyncController) DeleteSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 使用ScheduleService删除任务
	err := c.scheduleService.DeleteSyncTask(taskID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "任务不存在" {
			status = http.StatusNotFound
		} else if strings.Contains(err.Error(), "只能删除") {
			status = http.StatusBadRequest
		}

		response := &APIResponse{
			Status: 1,
			Msg:    "删除同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务删除成功",
		Data:   nil,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// BatchDeleteSyncTasks 批量删除同步任务
// @Summary 批量删除数据同步任务
// @Description 批量删除指定的数据同步任务，只允许删除已结束的任务
// @Description
// @Description **状态约束:**
// @Description - 只能删除 success、failed、cancelled 状态的任务
// @Description - 如果任何一个任务状态不允许删除，整个批量操作失败
// @Description - 删除操作不可逆，请谨慎操作
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param request body BatchDeleteSyncTasksRequest true "批量删除请求"
// @Success 200 {object} APIResponse{data=map[string]interface{}}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/batch-delete [post]
func (c *SyncController) BatchDeleteSyncTasks(w http.ResponseWriter, r *http.Request) {
	var request BatchDeleteSyncTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "请求参数解析失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(request.TaskIDs) == 0 {
		response := &APIResponse{
			Status: 1,
			Msg:    "任务ID列表不能为空",
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 使用ScheduleService批量删除任务
	deletedCount, err := c.scheduleService.BatchDeleteSyncTasks(request.TaskIDs)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "批量删除同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "批量删除同步任务成功",
		Data: map[string]interface{}{
			"deleted_count": deletedCount,
			"total_count":   len(request.TaskIDs),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CleanupCompletedTasks 清理已完成的历史任务
// @Summary 清理已完成的历史任务
// @Description 清理指定日期之前的已完成任务，用于维护数据库性能
// @Description
// @Description **清理规则:**
// @Description - 默认清理 success、failed、cancelled 状态的任务
// @Description - 可指定要清理的状态类型
// @Description - 只清理指定日期之前的任务
// @Description - 清理操作不可逆，请谨慎操作
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param request body CleanupCompletedTasksRequest true "清理请求"
// @Success 200 {object} APIResponse{data=map[string]interface{}}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/cleanup [post]
func (c *SyncController) CleanupCompletedTasks(w http.ResponseWriter, r *http.Request) {
	var request CleanupCompletedTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "请求参数解析失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 解析日期
	beforeDate, err := time.Parse(time.RFC3339, request.BeforeDate)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "日期格式错误，请使用RFC3339格式: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 使用ScheduleService清理任务
	deletedCount, err := c.scheduleService.CleanupCompletedTasks(beforeDate, request.Statuses)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "清理历史任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "历史任务清理成功",
		Data: map[string]interface{}{
			"deleted_count": deletedCount,
			"before_date":   request.BeforeDate,
			"statuses":      request.Statuses,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetTaskStatistics 获取任务统计信息
// @Summary 获取同步任务统计信息
// @Description 获取指定时间范围内的同步任务统计信息，包括成功率、执行时间等
// @Description
// @Description **统计信息包括:**
// @Description - 总任务数、成功任务数、失败任务数等
// @Description - 成功率百分比
// @Description - 平均执行时间
// @Description - 各状态任务分布
// @Description - 实时引擎统计信息
// @Tags 数据同步任务
// @Accept json
// @Produce json
// @Param data_source_id query string false "数据源ID"
// @Param start_time query string false "开始时间" format(datetime)
// @Param end_time query string false "结束时间" format(datetime)
// @Success 200 {object} APIResponse{data=map[string]interface{}}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/statistics [get]
func (c *SyncController) GetTaskStatistics(w http.ResponseWriter, r *http.Request) {
	dataSourceID := r.URL.Query().Get("data_source_id")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	// 默认查询最近7天的统计信息
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	if startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}
	if endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	// 使用ScheduleService获取统计信息
	statistics, err := c.scheduleService.GetTaskStatistics(dataSourceID, startTime, endTime)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "获取任务统计信息失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 合并SyncEngine的实时统计信息
	engineStats := c.syncEngine.GetStatistics()
	statistics["engine_statistics"] = engineStats

	response := &APIResponse{
		Status: 0,
		Msg:    "获取任务统计信息成功",
		Data:   statistics,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// 以下是任务控制相关接口

// StartSyncTask 启动同步任务
// @Summary 启动数据同步任务
// @Description 启动指定的数据同步任务，将任务提交到执行引擎
// @Description
// @Description **状态约束:**
// @Description - 只能启动 pending 状态的任务
// @Description - 启动后任务状态变为 running
// @Description - 其他状态的任务不允许启动
// @Tags 数据同步控制
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id}/start [post]
func (c *SyncController) StartSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 获取任务信息
	task, err := c.scheduleService.GetSyncTask(taskID)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "任务不存在: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查任务状态
	if task.Status != meta.SyncTaskStatusPending {
		response := &APIResponse{
			Status: 1,
			Msg:    "只能启动待执行状态的任务",
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 构建同步请求并提交到SyncEngine
	syncRequest := &models.SyncTaskRequest{
		DataSourceID: task.DataSourceID,
		InterfaceID:  "",
		SyncType:     models.SyncType(task.TaskType),
		Config:       task.Config,
		ScheduledBy:  "manual_start",
	}

	if task.InterfaceID != nil {
		syncRequest.InterfaceID = *task.InterfaceID
	}

	if _, err := c.syncEngine.SubmitSyncTask(syncRequest); err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "启动同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务启动成功",
		Data:   nil,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// StopSyncTask 停止同步任务
// @Summary 停止数据同步任务
// @Description 停止指定的正在运行的数据同步任务
// @Description
// @Description **状态约束:**
// @Description - 只能停止 running 状态的任务
// @Description - 停止后任务状态变为 cancelled
// @Description - 其他状态的任务不允许停止
// @Tags 数据同步控制
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id}/stop [post]
func (c *SyncController) StopSyncTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 通知SyncEngine停止任务
	if err := c.syncEngine.CancelTask(taskID); err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "停止同步任务失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "同步任务停止成功",
		Data:   nil,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSyncTaskStatus 获取同步状态
// @Summary 获取数据同步状态
// @Description 获取指定任务的当前同步状态和进度信息
// @Description
// @Description **返回信息包括:**
// @Description - 任务基本信息和当前状态
// @Description - 实时进度信息（进度百分比、处理行数等）
// @Description - 执行时间和处理速度
// @Description - 错误信息和统计数据
// @Tags 数据同步控制
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} APIResponse{data=SyncTaskStatusResponse}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/sync/tasks/{id}/status [get]
func (c *SyncController) GetSyncTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	// 从SyncEngine获取任务状态
	taskContext, err := c.syncEngine.GetTaskStatus(taskID)
	if err != nil {
		response := &APIResponse{
			Status: 1,
			Msg:    "获取同步状态失败: " + err.Error(),
			Data:   nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 转换为API友好的响应格式
	statusResponse := &SyncTaskStatusResponse{
		Task:      taskContext.Task,
		StartTime: taskContext.StartTime,
		Status:    string(taskContext.Status),
		Progress:  taskContext.Progress,
		Result:    taskContext.Result,
	}

	if taskContext.Error != nil {
		statusResponse.Error = taskContext.Error.Error()
	}

	if taskContext.Processor != nil {
		statusResponse.Processor = taskContext.Processor.GetProcessorType()
	}

	response := &APIResponse{
		Status: 0,
		Msg:    "获取同步状态成功",
		Data:   statusResponse,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// 请求和响应结构体定义

// CreateSyncTaskRequest 创建同步任务请求
type CreateSyncTaskRequest struct {
	DataSourceID string                 `json:"data_source_id" binding:"required"`
	InterfaceID  string                 `json:"interface_id,omitempty"`
	TaskType     string                 `json:"task_type" binding:"required"` // full_sync, incremental_sync, realtime_sync
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// UpdateSyncTaskRequest 更新同步任务请求
type UpdateSyncTaskRequest struct {
	Config map[string]interface{} `json:"config,omitempty"`
}

// GetSyncTasksResponse 获取任务列表响应
type GetSyncTasksResponse struct {
	Tasks      []models.SyncTask  `json:"tasks"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

// BatchDeleteSyncTasksRequest 批量删除同步任务请求
type BatchDeleteSyncTasksRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required"`
}

// CleanupCompletedTasksRequest 清理已完成任务请求
type CleanupCompletedTasksRequest struct {
	BeforeDate string   `json:"before_date" binding:"required"` // RFC3339格式的日期
	Statuses   []string `json:"statuses,omitempty"`             // 要清理的状态，默认["success", "failed", "cancelled"]
}

// SyncTaskStatusResponse 同步任务状态响应（用于API返回，避免context等字段）
type SyncTaskStatusResponse struct {
	Task      *models.SyncTask     `json:"task"`
	StartTime time.Time            `json:"start_time,omitempty"`
	Status    string               `json:"status"`
	Progress  *models.SyncProgress `json:"progress,omitempty"`
	Error     string               `json:"error,omitempty"`
	Result    *models.SyncResult   `json:"result,omitempty"`
	Processor string               `json:"processor,omitempty"`
}
