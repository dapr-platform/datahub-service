/*
 * @module service/basic_library/datasource/http_auth
 * @description HTTP认证数据源实现，支持多种认证方式和动态脚本执行
 * @architecture 策略模式 - 支持多种HTTP认证策略
 * @documentReference ai_docs/datasource_req.md, service/meta/datasource.go
 * @stateFlow HTTP连接生命周期：初始化认证 -> 建立连接 -> 执行请求 -> 关闭连接
 * @rules 支持Basic、Bearer、API Key等认证方式，可通过脚本自定义认证逻辑
 * @dependencies net/http, encoding/json, time
 * @refs interface.go, base.go
 */

package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"datahub-service/service/meta"
	"datahub-service/service/models"
)

// HTTPAuthDataSource HTTP认证数据源实现
type HTTPAuthDataSource struct {
	*BaseDataSource
	client      *http.Client
	baseURL     string
	authType    string
	credentials map[string]interface{}
	sessionData map[string]interface{} // 存储会话数据，如sessionId等
	mu          sync.RWMutex           // 保护sessionData的并发访问

	// 会话管理相关
	sessionRefreshTicker   *time.Ticker
	sessionCtx             context.Context
	sessionCancel          context.CancelFunc
	sessionRefreshInterval time.Duration
	isSessionActive        bool

	// 连接池支持
	connectionPool    ConnectionPool
	useConnectionPool bool
}

// NewHTTPAuthDataSource 创建HTTP认证数据源
func NewHTTPAuthDataSource() DataSourceInterface {
	// HTTP认证数据源支持常驻模式，特别是需要保持会话的接口（如绿云接口）
	base := NewBaseDataSource(meta.DataSourceTypeApiHTTPWithAuth, true)
	return &HTTPAuthDataSource{
		BaseDataSource: base,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		credentials:            make(map[string]interface{}),
		sessionData:            make(map[string]interface{}),
		sessionRefreshInterval: 11 * time.Hour, // 绿云接口需要每12小时刷新一次，这里设置11小时
		isSessionActive:        false,
		connectionPool:         NewDefaultConnectionPool(),
		useConnectionPool:      true, // 默认启用连接池
	}
}

// Init 初始化HTTP认证数据源
func (h *HTTPAuthDataSource) Init(ctx context.Context, ds *models.DataSource) error {
	if err := h.BaseDataSource.Init(ctx, ds); err != nil {
		return err
	}

	// 解析连接配置
	config := ds.ConnectionConfig
	if config == nil {
		return fmt.Errorf("连接配置不能为空")
	}

	// 获取基础URL
	if baseURL, ok := config[meta.DataSourceFieldBaseUrl].(string); ok {
		h.baseURL = baseURL
	} else {
		return fmt.Errorf("基础URL配置错误")
	}

	// 获取认证类型
	if authType, ok := config[meta.DataSourceFieldAuthType].(string); ok {
		h.authType = authType
	} else {
		return fmt.Errorf("认证类型配置错误")
	}

	// 提取认证凭据
	h.extractCredentials(config)

	// 设置超时时间
	if params := ds.ParamsConfig; params != nil {
		if timeout, ok := params[meta.DataSourceFieldTimeout].(float64); ok {
			h.client.Timeout = time.Duration(timeout) * time.Second
		}
	}

	// 如果启用了脚本执行，调用初始化脚本
	if ds.ScriptEnabled && ds.Script != "" {
		if err := h.executeInitScript(ctx); err != nil {
			return fmt.Errorf("初始化脚本执行失败: %v", err)
		}
	}

	return nil
}

// Start 启动HTTP认证数据源
func (h *HTTPAuthDataSource) Start(ctx context.Context) error {
	if err := h.BaseDataSource.Start(ctx); err != nil {
		return err
	}

	// 启动会话管理
	h.mu.Lock()
	if !h.isSessionActive {
		h.sessionCtx, h.sessionCancel = context.WithCancel(context.Background())
		h.isSessionActive = true
	}
	h.mu.Unlock()

	// 如果启用了脚本执行，调用启动脚本（如获取sessionId）
	ds := h.GetDataSource()
	if ds.ScriptEnabled && ds.Script != "" {
		if err := h.executeStartScript(ctx); err != nil {
			return fmt.Errorf("启动脚本执行失败: %v", err)
		}

		// 启动会话刷新定时器
		h.startSessionRefresh()
	} else {
		// HTTP数据源启动时可以进行连接测试
		if err := h.testConnection(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Execute 执行HTTP请求
func (h *HTTPAuthDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	// 检查数据源状态
	if !h.IsInitialized() {
		response.Error = "数据源未初始化"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("数据源未初始化")
	}

	// 如果启用了脚本执行，优先使用脚本
	ds := h.GetDataSource()
	if ds.ScriptEnabled && ds.Script != "" {
		scriptResult, err := h.executeScript(ctx, request)
		if err != nil {
			response.Error = fmt.Sprintf("脚本执行失败: %v", err)
			response.Duration = time.Since(startTime)
			return response, err
		}

		response.Success = true
		response.Data = scriptResult
		response.Duration = time.Since(startTime)
		return response, nil
	}

	// 默认HTTP请求处理
	return h.executeHTTPRequest(ctx, request)
}

// getHTTPClient 获取HTTP客户端，优先使用连接池
func (h *HTTPAuthDataSource) getHTTPClient(ctx context.Context) *http.Client {
	if h.useConnectionPool && h.connectionPool != nil {
		client, err := h.connectionPool.Get(ctx)
		if err == nil && client != nil {
			if httpClient, ok := client.(*http.Client); ok {
				return httpClient
			}
		}
	}
	return h.client
}

// executeHTTPRequest 执行HTTP请求
func (h *HTTPAuthDataSource) executeHTTPRequest(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// 构建请求URL
	url := h.baseURL
	if request.Query != "" {
		if strings.Contains(url, "?") {
			url += "&" + request.Query
		} else {
			url += "?" + request.Query
		}
	}

	// 确定HTTP方法
	method := "GET"
	if request.Operation != "" {
		switch strings.ToLower(request.Operation) {
		case "query", "get":
			method = "GET"
		case "insert", "post":
			method = "POST"
		case "update", "put":
			method = "PUT"
		case "delete":
			method = "DELETE"
		default:
			method = "GET"
		}
	}

	// 准备请求体
	var reqBody io.Reader
	if request.Data != nil && (method == "POST" || method == "PUT") {
		jsonData, err := json.Marshal(request.Data)
		if err != nil {
			response.Error = fmt.Sprintf("序列化请求数据失败: %v", err)
			response.Duration = time.Since(startTime)
			return response, err
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		response.Error = fmt.Sprintf("创建HTTP请求失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	// 设置认证头
	if err := h.setAuthHeaders(httpReq); err != nil {
		response.Error = fmt.Sprintf("设置认证头失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	// 设置Content-Type
	if reqBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 执行请求
	client := h.getHTTPClient(ctx)
	httpResp, err := client.Do(httpReq)
	if err != nil {
		response.Error = fmt.Sprintf("HTTP请求失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}
	defer httpResp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		response.Error = fmt.Sprintf("读取响应体失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	// 设置响应信息
	response.Duration = time.Since(startTime)
	response.Metadata["status_code"] = httpResp.StatusCode
	response.Metadata["headers"] = httpResp.Header
	response.Metadata["url"] = url
	response.Metadata["method"] = method

	// 检查HTTP状态码
	if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
		response.Success = true

		// 尝试解析JSON响应
		var jsonData interface{}
		if err := json.Unmarshal(respBody, &jsonData); err == nil {
			response.Data = jsonData
		} else {
			response.Data = string(respBody)
		}
	} else {
		response.Error = fmt.Sprintf("HTTP请求失败，状态码: %d, 响应: %s", httpResp.StatusCode, string(respBody))
		response.Data = string(respBody)
	}

	return response, nil
}

// Stop 停止HTTP认证数据源
func (h *HTTPAuthDataSource) Stop(ctx context.Context) error {
	// 停止会话刷新
	h.stopSessionRefresh()

	// 如果启用了脚本执行，调用停止脚本（如退出sessionId）
	ds := h.GetDataSource()
	if ds != nil && ds.ScriptEnabled && ds.Script != "" {
		if err := h.executeStopScript(ctx); err != nil {
			// 记录错误但不阻止停止流程
			fmt.Printf("停止脚本执行失败: %v\n", err)
		}
	}

	// 清理会话数据
	h.mu.Lock()
	h.sessionData = make(map[string]interface{})
	if h.sessionCancel != nil {
		h.sessionCancel()
	}
	h.isSessionActive = false
	h.mu.Unlock()

	// 关闭连接池
	if h.connectionPool != nil {
		h.connectionPool.Close()
	}

	return h.BaseDataSource.Stop(ctx)
}

// HealthCheck HTTP数据源健康检查
func (h *HTTPAuthDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	baseStatus, err := h.BaseDataSource.HealthCheck(ctx)
	if err != nil {
		return baseStatus, err
	}

	// 如果基础检查失败，直接返回
	if baseStatus.Status != "online" {
		return baseStatus, nil
	}

	// 执行HTTP连接测试
	startTime := time.Now()
	if err := h.testConnection(ctx); err != nil {
		baseStatus.Status = "error"
		baseStatus.Message = fmt.Sprintf("HTTP连接测试失败: %v", err)
	}
	baseStatus.ResponseTime = time.Since(startTime)

	return baseStatus, nil
}

// extractCredentials 提取认证凭据
func (h *HTTPAuthDataSource) extractCredentials(config map[string]interface{}) {
	// 提取用户名和密码
	if username, ok := config[meta.DataSourceFieldUsername].(string); ok {
		h.credentials["username"] = username
	}
	if password, ok := config[meta.DataSourceFieldPassword].(string); ok {
		h.credentials["password"] = password
	}

	// 提取API Key相关信息
	if apiKey, ok := config[meta.DataSourceFieldApiKey].(string); ok {
		h.credentials["api_key"] = apiKey
	}
	if apiSecret, ok := config[meta.DataSourceFieldApiSecret].(string); ok {
		h.credentials["api_secret"] = apiSecret
	}
	if apiKeyHeader, ok := config[meta.DataSourceFieldApiKeyHeader].(string); ok {
		h.credentials["api_key_header"] = apiKeyHeader
	}
}

// setAuthHeaders 设置认证头
func (h *HTTPAuthDataSource) setAuthHeaders(req *http.Request) error {
	switch h.authType {
	case meta.DataSourceAuthTypeBasic:
		return h.setBasicAuth(req)
	case meta.DataSourceAuthTypeBearer:
		return h.setBearerAuth(req)
	case meta.DataSourceAuthTypeAPIKey:
		return h.setAPIKeyAuth(req)
	case meta.DataSourceAuthTypeCustom:
		return h.setCustomAuth(req)
	default:
		return fmt.Errorf("不支持的认证类型: %s", h.authType)
	}
}

// setBasicAuth 设置Basic认证
func (h *HTTPAuthDataSource) setBasicAuth(req *http.Request) error {
	username, ok1 := h.credentials["username"].(string)
	password, ok2 := h.credentials["password"].(string)

	if !ok1 || !ok2 {
		return fmt.Errorf("Basic认证缺少用户名或密码")
	}

	req.SetBasicAuth(username, password)
	return nil
}

// setBearerAuth 设置Bearer认证
func (h *HTTPAuthDataSource) setBearerAuth(req *http.Request) error {
	token, ok := h.credentials["api_key"].(string)
	if !ok || token == "" {
		return fmt.Errorf("Bearer认证缺少token")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// setAPIKeyAuth 设置API Key认证
func (h *HTTPAuthDataSource) setAPIKeyAuth(req *http.Request) error {
	apiKey, ok1 := h.credentials["api_key"].(string)
	header, ok2 := h.credentials["api_key_header"].(string)

	if !ok1 || apiKey == "" {
		return fmt.Errorf("API Key认证缺少API Key")
	}

	if !ok2 || header == "" {
		header = "X-API-Key" // 默认header名称
	}

	req.Header.Set(header, apiKey)
	return nil
}

// setCustomAuth 设置自定义认证
func (h *HTTPAuthDataSource) setCustomAuth(req *http.Request) error {
	// 自定义认证逻辑，可以通过脚本实现
	return fmt.Errorf("自定义认证需要通过脚本实现")
}

// testConnection 测试连接
func (h *HTTPAuthDataSource) testConnection(ctx context.Context) error {
	// 对于自定义认证类型，如果有脚本，则跳过基本连接测试
	// 因为脚本已经在启动时验证了连接
	if h.authType == meta.DataSourceAuthTypeCustom {
		ds := h.GetDataSource()
		if ds.ScriptEnabled && ds.Script != "" {
			// 检查是否有sessionId，如果有则认为连接正常
			h.mu.RLock()
			sessionId, hasSession := h.sessionData["sessionId"]
			h.mu.RUnlock()

			if hasSession && sessionId != nil {
				return nil // 连接正常
			}
			return fmt.Errorf("脚本数据源未建立有效会话")
		}
	}

	// 创建简单的GET请求测试连接
	req, err := http.NewRequestWithContext(ctx, "GET", h.baseURL, nil)
	if err != nil {
		return fmt.Errorf("创建测试请求失败: %v", err)
	}

	// 设置认证头
	if err := h.setAuthHeaders(req); err != nil {
		return fmt.Errorf("设置认证头失败: %v", err)
	}

	// 执行请求
	client := h.getHTTPClient(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("测试连接失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码（2xx或4xx都认为连接正常，5xx认为服务器错误）
	if resp.StatusCode >= 500 {
		return fmt.Errorf("服务器错误，状态码: %d", resp.StatusCode)
	}

	return nil
}

// executeScript 执行自定义脚本
func (h *HTTPAuthDataSource) executeScript(ctx context.Context, request *ExecuteRequest) (interface{}, error) {
	ds := h.GetDataSource()
	if h.scriptExecutor == nil {
		return nil, fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := h.prepareScriptParams(ctx)
	params["request"] = request
	params["operation"] = "execute"

	return h.scriptExecutor.Execute(ctx, ds.Script, params)
}

// executeInitScript 执行初始化脚本
func (h *HTTPAuthDataSource) executeInitScript(ctx context.Context) error {
	ds := h.GetDataSource()
	if h.scriptExecutor == nil {
		return fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "init"

	result, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
	if err != nil {
		return err
	}

	// 处理初始化结果，可能包含配置更新
	if resultMap, ok := result.(map[string]interface{}); ok {
		if config, exists := resultMap["config"]; exists {
			if configMap, ok := config.(map[string]interface{}); ok {
				h.mu.Lock()
				for k, v := range configMap {
					h.sessionData[k] = v
				}
				h.mu.Unlock()
			}
		}
	}

	return nil
}

// executeStartScript 执行启动脚本
func (h *HTTPAuthDataSource) executeStartScript(ctx context.Context) error {
	ds := h.GetDataSource()
	if h.scriptExecutor == nil {
		return fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "start"

	result, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
	if err != nil {
		return err
	}

	// 处理启动结果，通常包含sessionId等会话信息
	if resultMap, ok := result.(map[string]interface{}); ok {
		h.mu.Lock()
		for k, v := range resultMap {
			h.sessionData[k] = v
		}
		h.mu.Unlock()
	}

	return nil
}

// executeStopScript 执行停止脚本
func (h *HTTPAuthDataSource) executeStopScript(ctx context.Context) error {
	ds := h.GetDataSource()
	if h.scriptExecutor == nil {
		return fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "stop"

	_, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
	return err
}

// prepareScriptParams 准备脚本参数
func (h *HTTPAuthDataSource) prepareScriptParams(ctx context.Context) map[string]interface{} {
	ds := h.GetDataSource()

	// 获取会话数据的副本
	h.mu.RLock()
	sessionDataCopy := make(map[string]interface{})
	for k, v := range h.sessionData {
		sessionDataCopy[k] = v
	}
	h.mu.RUnlock()

	params := make(map[string]interface{})
	params["dataSource"] = ds
	params["baseURL"] = h.baseURL
	params["authType"] = h.authType
	params["credentials"] = h.credentials
	params["sessionData"] = sessionDataCopy
	params["httpClient"] = h.client
	params["context"] = ctx

	// 添加辅助函数
	params["updateSessionData"] = func(key string, value interface{}) {
		h.mu.Lock()
		h.sessionData[key] = value
		h.mu.Unlock()
	}

	params["getSessionData"] = func(key string) interface{} {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.sessionData[key]
	}

	params["httpPost"] = h.createHTTPPostHelper(ctx)
	params["httpGet"] = h.createHTTPGetHelper(ctx)

	return params
}

// createHTTPPostHelper 创建HTTP POST辅助函数
func (h *HTTPAuthDataSource) createHTTPPostHelper(ctx context.Context) func(string, map[string]interface{}) (map[string]interface{}, error) {
	return func(url string, data map[string]interface{}) (map[string]interface{}, error) {
		// 将数据转换为表单格式
		formData := make(map[string]string)
		for k, v := range data {
			formData[k] = fmt.Sprintf("%v", v)
		}

		// 构建表单数据
		values := make([]string, 0, len(formData))
		for k, v := range formData {
			values = append(values, fmt.Sprintf("%s=%s", k, v))
		}
		formBody := strings.Join(values, "&")

		req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := h.getHTTPClient(ctx)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return map[string]interface{}{
				"status_code": resp.StatusCode,
				"body":        string(body),
			}, nil
		}

		result["status_code"] = resp.StatusCode
		return result, nil
	}
}

// createHTTPGetHelper 创建HTTP GET辅助函数
func (h *HTTPAuthDataSource) createHTTPGetHelper(ctx context.Context) func(string) (map[string]interface{}, error) {
	return func(url string) (map[string]interface{}, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		client := h.getHTTPClient(ctx)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return map[string]interface{}{
				"status_code": resp.StatusCode,
				"body":        string(body),
			}, nil
		}

		result["status_code"] = resp.StatusCode
		return result, nil
	}
}

// startSessionRefresh 启动会话刷新定时器
func (h *HTTPAuthDataSource) startSessionRefresh() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessionRefreshTicker != nil {
		h.sessionRefreshTicker.Stop()
	}

	h.sessionRefreshTicker = time.NewTicker(h.sessionRefreshInterval)

	go func() {
		for {
			select {
			case <-h.sessionCtx.Done():
				return
			case <-h.sessionRefreshTicker.C:
				h.refreshSession()
			}
		}
	}()
}

// stopSessionRefresh 停止会话刷新定时器
func (h *HTTPAuthDataSource) stopSessionRefresh() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sessionRefreshTicker != nil {
		h.sessionRefreshTicker.Stop()
		h.sessionRefreshTicker = nil
	}
}

// refreshSession 刷新会话
func (h *HTTPAuthDataSource) refreshSession() {
	ds := h.GetDataSource()
	if ds == nil || !ds.ScriptEnabled || ds.Script == "" {
		return
	}

	// 检查是否有sessionId需要刷新
	h.mu.RLock()
	sessionId, hasSession := h.sessionData["sessionId"]
	h.mu.RUnlock()

	if !hasSession || sessionId == nil {
		fmt.Printf("数据源 %s 没有活跃会话，跳过刷新\n", h.GetID())
		return
	}

	fmt.Printf("开始刷新数据源 %s 的会话\n", h.GetID())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 准备刷新脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "refresh"

	// 执行刷新脚本
	if h.scriptExecutor != nil {
		result, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
		if err != nil {
			fmt.Printf("数据源 %s 会话刷新失败: %v\n", h.GetID(), err)
		} else {
			// 更新会话数据
			if resultMap, ok := result.(map[string]interface{}); ok {
				h.mu.Lock()
				for k, v := range resultMap {
					h.sessionData[k] = v
				}
				h.sessionData["lastRefreshTime"] = time.Now().Format(time.RFC3339)
				h.mu.Unlock()
				fmt.Printf("数据源 %s 会话刷新成功\n", h.GetID())
			}
		}
	}
}

// GetSessionData 获取会话数据（供外部调用）
func (h *HTTPAuthDataSource) GetSessionData(key string) interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessionData[key]
}

// UpdateSessionData 更新会话数据（供外部调用）
func (h *HTTPAuthDataSource) UpdateSessionData(key string, value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessionData[key] = value
}

// IsSessionActive 检查会话是否活跃
func (h *HTTPAuthDataSource) IsSessionActive() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessionId, hasSession := h.sessionData["sessionId"]
	return h.isSessionActive && hasSession && sessionId != nil
}
