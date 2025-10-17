/*
 * @module service/datasource/http_post
 * @description HTTP POST数据源实现，作为服务器接收第三方POST数据
 * @architecture 观察者模式 - 监听HTTP请求并处理数据
 * @documentReference ai_docs/datasource_req1.md
 * @stateFlow HTTP服务器生命周期：创建 -> 启动监听 -> 接收请求 -> 处理数据 -> 停止服务
 * @rules 支持认证、请求体大小限制、超时控制
 * @dependencies net/http, context, sync, encoding/json
 * @refs interface.go, base.go
 */

package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"datahub-service/service/models"
	"log/slog"
)

// HTTPPostDataSource HTTP POST数据源实现
type HTTPPostDataSource struct {
	*BaseDataSource
	suffix        string // URL后缀，用于识别数据源
	authRequired  bool
	authToken     string
	maxBodySize   int64
	timeout       time.Duration
	receivedData  []map[string]interface{}      // 存储接收到的数据
	mu            sync.RWMutex                  // 保护receivedData的并发访问
	dataChannel   chan map[string]interface{}   // 数据通道，用于实时数据传输
	subscribers   []chan map[string]interface{} // 订阅者列表
	subscribersMu sync.RWMutex                  // 保护subscribers的并发访问
}

// NewHTTPPostDataSource 创建HTTP POST数据源
func NewHTTPPostDataSource() DataSourceInterface {
	return &HTTPPostDataSource{
		BaseDataSource: NewBaseDataSource("http_post", true), // 常驻数据源
		receivedData:   make([]map[string]interface{}, 0),
		dataChannel:    make(chan map[string]interface{}, 1000), // 缓冲通道
		subscribers:    make([]chan map[string]interface{}, 0),
	}
}

// Init 初始化HTTP POST数据源
func (h *HTTPPostDataSource) Init(ctx context.Context, ds *models.DataSource) error {
	if err := h.BaseDataSource.Init(ctx, ds); err != nil {
		return err
	}

	// 解析连接配置
	config := ds.ConnectionConfig
	if config == nil {
		return fmt.Errorf("连接配置不能为空")
	}

	// 解析URL后缀
	if suffix, exists := config["suffix"]; exists {
		if suffixStr, ok := suffix.(string); ok {
			h.suffix = suffixStr
		} else {
			return fmt.Errorf("URL后缀格式错误")
		}
	} else {
		return fmt.Errorf("缺少URL后缀配置")
	}

	// 解析认证配置
	if authRequired, exists := config["auth_required"]; exists {
		if required, ok := authRequired.(bool); ok {
			h.authRequired = required
		}
	}

	if h.authRequired {
		if authToken, exists := config["auth_token"]; exists {
			if token, ok := authToken.(string); ok {
				h.authToken = token
			} else {
				return fmt.Errorf("认证令牌格式错误")
			}
		} else {
			return fmt.Errorf("启用认证时必须提供认证令牌")
		}
	}

	// 解析参数配置
	if ds.ParamsConfig != nil {
		h.parseParamsConfig(ds.ParamsConfig)
	}

	return nil
}

// parseParamsConfig 解析参数配置
func (h *HTTPPostDataSource) parseParamsConfig(params map[string]interface{}) {
	// 最大请求体大小
	if maxBodySize, exists := params["max_body_size"]; exists {
		switch v := maxBodySize.(type) {
		case float64:
			h.maxBodySize = int64(v) * 1024 * 1024 // MB转字节
		case int:
			h.maxBodySize = int64(v) * 1024 * 1024
		default:
			h.maxBodySize = 10 * 1024 * 1024 // 默认10MB
		}
	} else {
		h.maxBodySize = 10 * 1024 * 1024
	}

	// 超时时间
	if timeout, exists := params["timeout"]; exists {
		switch v := timeout.(type) {
		case float64:
			h.timeout = time.Duration(v) * time.Second
		case int:
			h.timeout = time.Duration(v) * time.Second
		default:
			h.timeout = 30 * time.Second
		}
	} else {
		h.timeout = 30 * time.Second
	}
}

// Start 启动HTTP POST数据源
func (h *HTTPPostDataSource) Start(ctx context.Context) error {
	if err := h.BaseDataSource.Start(ctx); err != nil {
		return err
	}

	// 启动数据处理协程
	go h.processData()

	// 注册到全局HTTP POST数据源管理器
	if err := RegisterHTTPPostDataSource(h.suffix, h); err != nil {
		return fmt.Errorf("注册HTTP POST数据源失败: %v", err)
	}

	slog.Info("HTTP POST数据源已启动，URL后缀: %s\n", h.suffix)
	return nil
}

// HandleWebhook 处理webhook请求（由控制器调用）
func (h *HTTPPostDataSource) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 只接受POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 检查认证
	if h.authRequired {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.Header.Get("X-Auth-Token")
		}
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token != h.authToken && "Bearer "+h.authToken != token {
			http.Error(w, "认证失败", http.StatusUnauthorized)
			return
		}
	}

	// 检查请求体大小
	if r.ContentLength > h.maxBodySize {
		http.Error(w, fmt.Sprintf("请求体过大，最大允许%dMB", h.maxBodySize/(1024*1024)), http.StatusRequestEntityTooLarge)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(io.LimitReader(r.Body, h.maxBodySize))
	if err != nil {
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}

	// 解析JSON数据
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// 如果不是JSON，保存为字符串
		data = map[string]interface{}{
			"raw_data":     string(body),
			"content_type": r.Header.Get("Content-Type"),
		}
	}

	// 添加元数据
	data["_metadata"] = map[string]interface{}{
		"received_at":    time.Now().Format(time.RFC3339),
		"remote_addr":    r.RemoteAddr,
		"user_agent":     r.Header.Get("User-Agent"),
		"content_length": r.ContentLength,
		"method":         r.Method,
		"url":            r.URL.String(),
		"headers":        r.Header,
	}

	// 发送到数据通道
	select {
	case h.dataChannel <- data:
		// 数据发送成功
	default:
		// 通道满了，记录警告但不阻塞
		slog.Error("HTTP POST数据源数据通道已满，丢弃数据\n")
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "数据接收成功",
		"timestamp": time.Now().Format(time.RFC3339),
		"data_size": len(body),
	})
}

// processData 处理接收到的数据
func (h *HTTPPostDataSource) processData() {
	for data := range h.dataChannel {
		// 存储数据
		h.mu.Lock()
		h.receivedData = append(h.receivedData, data)

		// 限制存储的数据量，只保留最近的1000条
		if len(h.receivedData) > 1000 {
			h.receivedData = h.receivedData[len(h.receivedData)-1000:]
		}
		h.mu.Unlock()

		// 通知所有订阅者
		h.notifySubscribers(data)
	}
}

// notifySubscribers 通知所有订阅者
func (h *HTTPPostDataSource) notifySubscribers(data map[string]interface{}) {
	h.subscribersMu.RLock()
	defer h.subscribersMu.RUnlock()

	for _, subscriber := range h.subscribers {
		select {
		case subscriber <- data:
			// 数据发送成功
		default:
			// 订阅者通道满了，跳过
		}
	}
}

// Execute 执行操作
func (h *HTTPPostDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	if !h.IsInitialized() {
		response.Error = "数据源未初始化"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("数据源未初始化")
	}

	// 如果启用了脚本，先尝试执行脚本
	if h.GetDataSource().ScriptEnabled && h.GetDataSource().Script != "" {
		scriptResult, err := h.executeScript(ctx, request)
		if err == nil && scriptResult != nil {
			response.Success = true
			response.Data = scriptResult
			response.Duration = time.Since(startTime)
			return response, nil
		}
	}

	switch request.Operation {
	case "query", "read":
		return h.handleQuery(ctx, request, startTime)
	case "subscribe":
		return h.handleSubscribe(ctx, request, startTime)
	case "status":
		return h.handleStatus(ctx, request, startTime)
	default:
		response.Error = fmt.Sprintf("不支持的操作: %s", request.Operation)
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("不支持的操作: %s", request.Operation)
	}
}

// handleQuery 处理查询操作
func (h *HTTPPostDataSource) handleQuery(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   true,
		Timestamp: startTime,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// 获取查询参数
	limit := 100 // 默认限制
	offset := 0  // 默认偏移

	if request.Params != nil {
		if l, exists := request.Params["limit"]; exists {
			if limitInt, ok := l.(int); ok {
				limit = limitInt
			}
		}
		if o, exists := request.Params["offset"]; exists {
			if offsetInt, ok := o.(int); ok {
				offset = offsetInt
			}
		}
	}

	// 计算数据范围
	total := len(h.receivedData)
	start := offset
	end := offset + limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var data []map[string]interface{}
	if start < end {
		data = h.receivedData[start:end]
	} else {
		data = make([]map[string]interface{}, 0)
	}

	response.Data = data
	response.RowCount = int64(len(data))
	response.Metadata = map[string]interface{}{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"suffix": h.suffix,
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// handleSubscribe 处理订阅操作
func (h *HTTPPostDataSource) handleSubscribe(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   true,
		Timestamp: startTime,
		Message:   "订阅功能需要通过实时接口实现",
	}

	// 创建订阅通道
	subscriber := make(chan map[string]interface{}, 100)

	h.subscribersMu.Lock()
	h.subscribers = append(h.subscribers, subscriber)
	h.subscribersMu.Unlock()

	response.Metadata = map[string]interface{}{
		"subscriber_count": len(h.subscribers),
		"message":          "已添加到订阅列表，请通过WebSocket或SSE接口获取实时数据",
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// handleStatus 处理状态查询
func (h *HTTPPostDataSource) handleStatus(ctx context.Context, request *ExecuteRequest, startTime time.Time) (*ExecuteResponse, error) {
	response := &ExecuteResponse{
		Success:   true,
		Timestamp: startTime,
	}

	h.mu.RLock()
	dataCount := len(h.receivedData)
	h.mu.RUnlock()

	h.subscribersMu.RLock()
	subscriberCount := len(h.subscribers)
	h.subscribersMu.RUnlock()

	response.Data = map[string]interface{}{
		"suffix":           h.suffix,
		"auth_required":    h.authRequired,
		"max_body_size":    h.maxBodySize,
		"timeout":          h.timeout.Seconds(),
		"data_count":       dataCount,
		"subscriber_count": subscriberCount,
		"is_running":       h.IsStarted(),
	}
	response.Duration = time.Since(startTime)

	return response, nil
}

// Stop 停止HTTP POST数据源
func (h *HTTPPostDataSource) Stop(ctx context.Context) error {
	if err := h.BaseDataSource.Stop(ctx); err != nil {
		return err
	}

	// 从全局管理器中注销
	UnregisterHTTPPostDataSource(h.suffix)

	// 关闭数据通道
	close(h.dataChannel)

	// 关闭所有订阅者通道
	h.subscribersMu.Lock()
	for _, subscriber := range h.subscribers {
		close(subscriber)
	}
	h.subscribers = make([]chan map[string]interface{}, 0)
	h.subscribersMu.Unlock()

	slog.Info("HTTP POST数据源已停止\n")
	return nil
}

// HealthCheck 健康检查
func (h *HTTPPostDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status, err := h.BaseDataSource.HealthCheck(ctx)
	if err != nil {
		return status, err
	}

	// 检查数据源状态
	if h.IsStarted() {
		status.Status = "online"
		status.Message = "HTTP POST数据源正在运行"

		h.mu.RLock()
		dataCount := len(h.receivedData)
		h.mu.RUnlock()

		status.Details["suffix"] = h.suffix
		status.Details["data_count"] = dataCount
		status.Details["auth_required"] = h.authRequired
	} else {
		status.Status = "offline"
		status.Message = "HTTP POST数据源未运行"
	}

	return status, nil
}

// GetReceivedData 获取接收到的数据（用于测试）
func (h *HTTPPostDataSource) GetReceivedData() []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 返回数据副本
	data := make([]map[string]interface{}, len(h.receivedData))
	copy(data, h.receivedData)
	return data
}

// ClearReceivedData 清空接收到的数据（用于测试）
func (h *HTTPPostDataSource) ClearReceivedData() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.receivedData = make([]map[string]interface{}, 0)
}

// GetSuffix 获取URL后缀（用于测试）
func (h *HTTPPostDataSource) GetSuffix() string {
	return h.suffix
}

// executeScript 执行脚本
func (h *HTTPPostDataSource) executeScript(ctx context.Context, request *ExecuteRequest) (interface{}, error) {
	if h.scriptExecutor == nil {
		return nil, fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := make(map[string]interface{})
	params["request"] = request
	params["dataSource"] = h.GetDataSource()
	params["connectionConfig"] = h.GetDataSource().ConnectionConfig
	params["paramsConfig"] = h.GetDataSource().ParamsConfig
	params["operation"] = request.Operation
	params["receivedData"] = h.GetReceivedData()

	return h.scriptExecutor.Execute(ctx, h.GetDataSource().Script, params)
}

// 全局HTTP POST数据源管理器
var (
	httpPostDataSources   = make(map[string]*HTTPPostDataSource)
	httpPostDataSourcesMu sync.RWMutex
)

// RegisterHTTPPostDataSource 注册HTTP POST数据源
func RegisterHTTPPostDataSource(suffix string, ds *HTTPPostDataSource) error {
	httpPostDataSourcesMu.Lock()
	defer httpPostDataSourcesMu.Unlock()

	if _, exists := httpPostDataSources[suffix]; exists {
		return fmt.Errorf("HTTP POST数据源后缀 %s 已存在", suffix)
	}

	httpPostDataSources[suffix] = ds
	return nil
}

// UnregisterHTTPPostDataSource 注销HTTP POST数据源
func UnregisterHTTPPostDataSource(suffix string) {
	httpPostDataSourcesMu.Lock()
	defer httpPostDataSourcesMu.Unlock()
	delete(httpPostDataSources, suffix)
}

// GetHTTPPostDataSource 根据后缀获取HTTP POST数据源
func GetHTTPPostDataSource(suffix string) (*HTTPPostDataSource, bool) {
	httpPostDataSourcesMu.RLock()
	defer httpPostDataSourcesMu.RUnlock()
	ds, exists := httpPostDataSources[suffix]
	return ds, exists
}

// ListHTTPPostDataSources 列出所有HTTP POST数据源
func ListHTTPPostDataSources() map[string]*HTTPPostDataSource {
	httpPostDataSourcesMu.RLock()
	defer httpPostDataSourcesMu.RUnlock()

	result := make(map[string]*HTTPPostDataSource)
	for suffix, ds := range httpPostDataSources {
		result[suffix] = ds
	}
	return result
}
