/*
 * @module api/controllers/event_controller
 * @description 事件管理控制器，提供SSE连接和数据库事件监听管理API
 * @architecture RESTful API架构 - 控制器层
 * @documentReference ai_docs/patch_db_event.md
 * @stateFlow HTTP请求 -> 业务逻辑处理 -> 响应返回
 * @rules 遵循RESTful API设计规范，统一错误处理和响应格式
 * @dependencies datahub-service/service, github.com/go-chi/chi/v5, github.com/go-chi/render
 * @refs ai_docs/requirements.md
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// EventController 事件管理控制器
type EventController struct {
	eventService *service.EventService
}

// NewEventController 创建事件控制器实例
func NewEventController() *EventController {
	return &EventController{
		eventService: service.GlobalEventService,
	}
}

// === SSE连接处理 ===

// HandleSSE 处理SSE连接
// @Summary 建立SSE连接
// @Description 前端页面通过此接口建立SSE连接，接收实时事件推送
// @Tags 事件管理
// @Param user_name path string true "用户名"
// @Success 200 {string} string "SSE事件流"
// @Router /sse/{user_name} [get]
func (c *EventController) HandleSSE(w http.ResponseWriter, r *http.Request) {
	userName := chi.URLParam(r, "user_name")
	if userName == "" {
		http.Error(w, "用户名不能为空", http.StatusBadRequest)
		return
	}

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// 生成连接ID
	connectionID := uuid.New().String()
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	// 添加SSE连接
	client := c.eventService.AddSSEConnection(userName, connectionID, clientIP)
	defer c.eventService.RemoveSSEConnection(userName, connectionID)

	// 发送连接成功事件
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"connection_id\":\"%s\",\"timestamp\":\"%s\"}\n\n",
		connectionID, time.Now().Format(time.RFC3339))

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 处理事件推送
	for {
		select {
		case event := <-client.Channel:
			// 发送事件数据

			fmt.Fprintf(w, "data: %s\n\n", toJSON(event))

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-client.Done:
			return

		case <-r.Context().Done():
			return
		}
	}
}

// SendEvent 发送事件给指定用户
// @Summary 发送事件
// @Description 向指定用户发送SSE事件
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body SendEventRequest true "发送事件请求"
// @Success 200 {object} APIResponse
// @Router /events/send [post]
func (c *EventController) SendEvent(w http.ResponseWriter, r *http.Request) {
	var req SendEventRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "请求参数解析失败", err))
		return
	}

	// 验证请求参数
	if req.UserName == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "用户名不能为空", nil))
		return
	}
	if req.EventType == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "事件类型不能为空", nil))
		return
	}

	// 创建SSE事件
	event := &models.SSEEvent{
		EventType: req.EventType,
		UserName:  req.UserName,
		Data:      req.Data,
		CreatedAt: time.Now(),
	}

	// 发送事件
	if err := c.eventService.SendEventToUser(req.UserName, event); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "发送事件失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("事件发送成功", map[string]interface{}{
		"event_id": event.ID,
	}))
}

// BroadcastEvent 广播事件
// @Summary 广播事件
// @Description 向所有连接的用户广播SSE事件
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body BroadcastEventRequest true "广播事件请求"
// @Success 200 {object} APIResponse
// @Router /events/broadcast [post]
func (c *EventController) BroadcastEvent(w http.ResponseWriter, r *http.Request) {
	var req BroadcastEventRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "请求参数解析失败", err))
		return
	}

	// 验证请求参数
	if req.EventType == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "事件类型不能为空", nil))
		return
	}

	// 创建SSE事件
	event := &models.SSEEvent{
		EventType: req.EventType,
		Data:      req.Data,
		CreatedAt: time.Now(),
	}

	// 广播事件
	if err := c.eventService.BroadcastEvent(event); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "广播事件失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("事件广播成功", map[string]interface{}{
		"event_id": event.ID,
	}))
}

// === 请求和响应结构体 ===

// SendEventRequest 发送事件请求
type SendEventRequest struct {
	UserName  string                 `json:"user_name" example:"admin"`
	EventType string                 `json:"event_type" example:"system_notification"`
	Data      map[string]interface{} `json:"data"`
}

// BroadcastEventRequest 广播事件请求
type BroadcastEventRequest struct {
	EventType string                 `json:"event_type" example:"system_notification"`
	Data      map[string]interface{} `json:"data"`
}

// GetSSEConnectionList 获取SSE连接列表
// @Summary 获取SSE连接列表
// @Description 分页获取SSE连接列表，支持多种过滤条件
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param user_name query string false "用户名过滤"
// @Param is_active query bool false "连接状态过滤"
// @Param client_ip query string false "客户端IP过滤"
// @Success 200 {object} APIResponse{data=SSEConnectionListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /events/connections [get]
func (c *EventController) GetSSEConnectionList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	userName := r.URL.Query().Get("user_name")
	clientIP := r.URL.Query().Get("client_ip")
	isActiveStr := r.URL.Query().Get("is_active")

	var isActive *bool
	if isActiveStr != "" {
		if isActiveStr == "true" {
			val := true
			isActive = &val
		} else if isActiveStr == "false" {
			val := false
			isActive = &val
		}
	}

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	connections, total, err := c.eventService.GetSSEConnectionList(page, size, userName, clientIP, isActive)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "获取SSE连接列表失败", err))
		return
	}

	// 构建响应
	response := SSEConnectionListResponse{
		List:  connections,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.Render(w, r, SuccessResponse("获取SSE连接列表成功", response))
}

// GetEventHistoryList 获取事件历史列表
// @Summary 获取事件历史列表
// @Description 分页获取事件历史列表，支持多种过滤条件
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param user_name query string false "用户名过滤"
// @Param event_type query string false "事件类型过滤"
// @Param sent query bool false "发送状态过滤"
// @Param read query bool false "读取状态过滤"
// @Success 200 {object} APIResponse{data=EventHistoryListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /events/history [get]
func (c *EventController) GetEventHistoryList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	userName := r.URL.Query().Get("user_name")
	eventType := r.URL.Query().Get("event_type")
	sentStr := r.URL.Query().Get("sent")
	readStr := r.URL.Query().Get("read")

	var sent *bool
	if sentStr != "" {
		if sentStr == "true" {
			val := true
			sent = &val
		} else if sentStr == "false" {
			val := false
			sent = &val
		}
	}

	var read *bool
	if readStr != "" {
		if readStr == "true" {
			val := true
			read = &val
		} else if readStr == "false" {
			val := false
			read = &val
		}
	}

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	events, total, err := c.eventService.GetEventHistoryList(page, size, userName, eventType, sent, read)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "获取事件历史列表失败", err))
		return
	}

	// 构建响应
	response := EventHistoryListResponse{
		List:  events,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.Render(w, r, SuccessResponse("获取事件历史列表成功", response))
}

// === 请求和响应结构体 ===

// SSEConnectionListResponse SSE连接列表响应结构
type SSEConnectionListResponse struct {
	List  []models.SSEConnection `json:"list"`
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Size  int                    `json:"size"`
}

// EventHistoryListResponse 事件历史列表响应结构
type EventHistoryListResponse struct {
	List  []models.SSEEvent `json:"list"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}

// toJSON 将对象转换为JSON字符串
func toJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
