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
	"log/slog"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

		// 只有当配置了会话刷新间隔时才启动定时器
		if h.sessionRefreshInterval > 0 {
			h.startSessionRefresh()
		}
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
		// 如果baseURL以/结尾且Query以/开头，去掉一个/
		if strings.HasSuffix(url, "/") && strings.HasPrefix(request.Query, "/") {
			url = url[:len(url)-1] + request.Query
		} else if !strings.HasSuffix(url, "/") && !strings.HasPrefix(request.Query, "/") {
			// 如果baseURL不以/结尾且Query不以/开头，添加/
			url = url + "/" + request.Query
		} else {
			// 其他情况直接拼接
			url = url + request.Query
		}
	}

	// 添加查询参数
	if request.Params != nil && len(request.Params) > 0 {
		// 过滤出真正的查询参数（排除method、headers、body等元数据）
		queryParams := make([]string, 0)
		for key, value := range request.Params {
			// 跳过元数据字段
			if key == "method" || key == "headers" || key == "body" || key == "use_form_data" {
				continue
			}
			// 格式化查询参数值并进行URL编码
			var paramValue string
			if strValue, ok := value.(string); ok {
				paramValue = strValue
			} else {
				paramValue = fmt.Sprintf("%v", value)
			}
			// URL编码参数值
			encodedValue := h.urlEncodeQueryParam(paramValue)
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", key, encodedValue))
		}

		// 将查询参数拼接到URL
		if len(queryParams) > 0 {
			if strings.Contains(url, "?") {
				url += "&" + strings.Join(queryParams, "&")
			} else {
				url += "?" + strings.Join(queryParams, "&")
			}
		}
	}

	// 确定HTTP方法
	method := "GET"

	// 优先使用从查询构建器传递的方法
	if request.Params != nil {
		if methodParam, exists := request.Params["method"]; exists {
			if methodStr, ok := methodParam.(string); ok && methodStr != "" {
				method = strings.ToUpper(methodStr)
				slog.Debug("executeHTTPRequest - 使用查询构建器传递的HTTP方法", "value", method)
			}
		}
	}

	// 如果没有从参数中获取到方法，则根据Operation推断
	if method == "GET" && request.Operation != "" {
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
		slog.Debug("executeHTTPRequest - 根据Operation推断HTTP方法: %s (Operation: %s)\n", method, request.Operation)
	}

	// 准备请求体
	var reqBody io.Reader
	var contentType string = "application/json"

	// 检查是否有从查询构建器传递的body数据
	var bodyData interface{}
	var useFormData bool

	if request.Params != nil {
		if bodyParam, exists := request.Params["body"]; exists && bodyParam != nil {
			bodyData = bodyParam
			slog.Debug("executeHTTPRequest - 使用查询构建器传递的body数据", "data", bodyData)
		}
		if formDataParam, exists := request.Params["use_form_data"]; exists {
			useFormData, _ = formDataParam.(bool)
			slog.Debug("executeHTTPRequest - 使用表单数据模式", "value", useFormData)
		}
	}

	// 如果没有从参数中获取到body，使用request.Data
	if bodyData == nil {
		bodyData = request.Data
	}

	if bodyData != nil && (method == "POST" || method == "PUT") {
		if useFormData {
			// 使用表单数据格式
			if bodyStr, ok := bodyData.(string); ok {
				reqBody = strings.NewReader(bodyStr)
				contentType = "application/x-www-form-urlencoded"
				slog.Debug("executeHTTPRequest - 使用表单数据", "value", bodyStr)
			} else if bodyMap, ok := bodyData.(map[string]interface{}); ok {
				// 将map转换为表单数据
				values := make([]string, 0, len(bodyMap))
				for k, v := range bodyMap {
					values = append(values, fmt.Sprintf("%s=%s", k, fmt.Sprintf("%v", v)))
				}
				formBody := strings.Join(values, "&")
				reqBody = strings.NewReader(formBody)
				contentType = "application/x-www-form-urlencoded"
				slog.Debug("executeHTTPRequest - 转换map为表单数据", "value", formBody)
			}
		} else {
			// 使用JSON格式
			jsonData, err := json.Marshal(bodyData)
			if err != nil {
				response.Error = fmt.Sprintf("序列化请求数据失败: %v", err)
				response.Duration = time.Since(startTime)
				return response, err
			}
			reqBody = bytes.NewReader(jsonData)
			contentType = "application/json"
			slog.Debug("executeHTTPRequest - 使用JSON数据", "value", string(jsonData))
		}
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
		httpReq.Header.Set("Content-Type", contentType)
		slog.Debug("executeHTTPRequest - 设置Content-Type", "value", contentType)
	}

	// 设置从查询构建器传递的额外头部
	if request.Params != nil {
		if headersParam, exists := request.Params["headers"]; exists {
			if headers, ok := headersParam.(map[string]interface{}); ok {
				for key, value := range headers {
					if strValue, ok := value.(string); ok {
						httpReq.Header.Set(key, strValue)
						slog.Debug("executeHTTPRequest - 设置额外头部: %s = %s\n", key, strValue)
					}
				}
			}
		}
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
			slog.Error("停止脚本执行失败: %v\n", err.Error())
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
	// 提取基本认证信息
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

	// 提取OAuth2相关信息
	if clientId, ok := config[meta.DataSourceFieldClientId].(string); ok {
		h.credentials["client_id"] = clientId
	}
	if clientSecret, ok := config[meta.DataSourceFieldClientSecret].(string); ok {
		h.credentials["client_secret"] = clientSecret
	}
	if grantType, ok := config[meta.DataSourceFieldGrantType].(string); ok {
		h.credentials["grant_type"] = grantType
	}
	if scope, ok := config[meta.DataSourceFieldScope].(string); ok {
		h.credentials["scope"] = scope
	}

	// 如果是自定义认证类型，从custom_map中提取配置
	if h.authType == meta.DataSourceAuthTypeCustom {
		if customMapRaw, ok := config[meta.DatasourceFieldCustomMap]; ok {
			if customMap, ok := customMapRaw.(map[string]interface{}); ok {
				// 将custom_map的内容合并到credentials中
				for key, value := range customMap {
					h.credentials[key] = value
				}

				// 从custom_map中提取会话刷新间隔
				if refreshIntervalRaw, exists := customMap["session_refresh_interval"]; exists {
					if refreshInterval, ok := refreshIntervalRaw.(float64); ok && refreshInterval > 0 {
						h.sessionRefreshInterval = time.Duration(refreshInterval) * time.Second
					} else if refreshIntervalStr, ok := refreshIntervalRaw.(string); ok {
						if duration, err := time.ParseDuration(refreshIntervalStr); err == nil {
							h.sessionRefreshInterval = duration
						}
					}
				}
			}
		}
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
	case meta.DataSourceAuthTypeOAuth2:
		return h.setOAuth2Auth(req)
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

	if !ok1 || username == "" {
		return fmt.Errorf("Basic认证缺少用户名")
	}
	if !ok2 || password == "" {
		return fmt.Errorf("Basic认证缺少密码")
	}

	req.SetBasicAuth(username, password)

	// 添加常用的Basic认证相关头部
	req.Header.Set("Content-Type", "application/json")

	return nil
}

// setBearerAuth 设置Bearer认证
func (h *HTTPAuthDataSource) setBearerAuth(req *http.Request) error {
	// 优先使用token字段，如果没有则使用api_key字段
	var token string
	if tokenValue, ok := h.credentials["token"].(string); ok && tokenValue != "" {
		token = tokenValue
	} else if apiKeyValue, ok := h.credentials["api_key"].(string); ok && apiKeyValue != "" {
		token = apiKeyValue
	} else {
		return fmt.Errorf("Bearer认证缺少token或api_key")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 如果有额外的头部配置，也可以添加
	if extraHeaders, ok := h.credentials["headers"].(map[string]interface{}); ok {
		for key, value := range extraHeaders {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	return nil
}

// setAPIKeyAuth 设置API Key认证
func (h *HTTPAuthDataSource) setAPIKeyAuth(req *http.Request) error {
	apiKey, ok1 := h.credentials["api_key"].(string)
	if !ok1 || apiKey == "" {
		return fmt.Errorf("API Key认证缺少API Key")
	}

	// 获取API Key头部名称
	header, ok2 := h.credentials["api_key_header"].(string)
	if !ok2 || header == "" {
		header = "X-API-Key" // 默认header名称
	}

	req.Header.Set(header, apiKey)

	// 如果有API Secret，也添加到头部
	if apiSecret, ok := h.credentials["api_secret"].(string); ok && apiSecret != "" {
		secretHeader, ok := h.credentials["api_secret_header"].(string)
		if !ok || secretHeader == "" {
			secretHeader = "X-API-Secret" // 默认secret header名称
		}
		req.Header.Set(secretHeader, apiSecret)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 如果有额外的头部配置，也可以添加
	if extraHeaders, ok := h.credentials["headers"].(map[string]interface{}); ok {
		for key, value := range extraHeaders {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	return nil
}

// setOAuth2Auth 设置OAuth2认证
func (h *HTTPAuthDataSource) setOAuth2Auth(req *http.Request) error {
	// 检查是否已有访问令牌
	h.mu.RLock()
	accessToken, hasToken := h.sessionData["access_token"]
	tokenExpiry, hasExpiry := h.sessionData["token_expiry"]
	h.mu.RUnlock()

	// 检查令牌是否过期
	needsRefresh := !hasToken
	if hasExpiry {
		if expiryTime, ok := tokenExpiry.(time.Time); ok {
			if time.Now().After(expiryTime.Add(-30 * time.Second)) { // 提前30秒刷新
				needsRefresh = true
			}
		}
	}

	// 如果需要刷新令牌
	if needsRefresh {
		if err := h.refreshOAuth2Token(req.Context()); err != nil {
			return fmt.Errorf("OAuth2令牌刷新失败: %v", err)
		}

		// 重新获取令牌
		h.mu.RLock()
		accessToken = h.sessionData["access_token"]
		h.mu.RUnlock()
	}

	// 设置Authorization头部
	if token, ok := accessToken.(string); ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		return nil
	}

	return fmt.Errorf("OAuth2访问令牌不可用")
}

// refreshOAuth2Token 刷新OAuth2令牌
func (h *HTTPAuthDataSource) refreshOAuth2Token(ctx context.Context) error {
	clientId, ok1 := h.credentials["client_id"].(string)
	clientSecret, ok2 := h.credentials["client_secret"].(string)

	if !ok1 || clientId == "" {
		return fmt.Errorf("OAuth2认证缺少client_id")
	}
	if !ok2 || clientSecret == "" {
		return fmt.Errorf("OAuth2认证缺少client_secret")
	}

	// 获取授权类型，默认为client_credentials
	grantType, ok := h.credentials["grant_type"].(string)
	if !ok || grantType == "" {
		grantType = "client_credentials"
	}

	// 构建令牌请求
	tokenURL := h.baseURL
	if tokenEndpoint, ok := h.credentials["token_endpoint"].(string); ok && tokenEndpoint != "" {
		tokenURL = tokenEndpoint
	} else {
		// 尝试从baseURL构建token端点
		if strings.HasSuffix(h.baseURL, "/") {
			tokenURL = h.baseURL + "oauth/token"
		} else {
			tokenURL = h.baseURL + "/oauth/token"
		}
	}

	// 准备请求参数
	params := map[string]string{
		"grant_type":    grantType,
		"client_id":     clientId,
		"client_secret": clientSecret,
	}

	// 添加scope（如果有）
	if scope, ok := h.credentials["scope"].(string); ok && scope != "" {
		params["scope"] = scope
	}

	// 对于password授权类型，添加用户名和密码
	if grantType == "password" {
		if username, ok := h.credentials["username"].(string); ok && username != "" {
			params["username"] = username
		}
		if password, ok := h.credentials["password"].(string); ok && password != "" {
			params["password"] = password
		}
	}

	// 构建表单数据
	values := make([]string, 0, len(params))
	for k, v := range params {
		values = append(values, fmt.Sprintf("%s=%s", k, v))
	}
	formBody := strings.Join(values, "&")

	// 创建请求
	tokenReq, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(formBody))
	if err != nil {
		return fmt.Errorf("创建令牌请求失败: %v", err)
	}

	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")

	// 执行请求
	client := h.getHTTPClient(ctx)
	resp, err := client.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("令牌请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取令牌响应失败: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("令牌请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析令牌响应
	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("解析令牌响应失败: %v", err)
	}

	// 提取访问令牌
	accessToken, ok := tokenResponse["access_token"].(string)
	if !ok || accessToken == "" {
		return fmt.Errorf("响应中缺少access_token")
	}

	// 计算令牌过期时间
	var expiryTime time.Time
	if expiresIn, ok := tokenResponse["expires_in"].(float64); ok {
		expiryTime = time.Now().Add(time.Duration(expiresIn) * time.Second)
	} else {
		// 默认1小时过期
		expiryTime = time.Now().Add(time.Hour)
	}

	// 保存令牌信息
	h.mu.Lock()
	h.sessionData["access_token"] = accessToken
	h.sessionData["token_expiry"] = expiryTime
	h.sessionData["token_type"] = tokenResponse["token_type"]
	if refreshToken, ok := tokenResponse["refresh_token"].(string); ok {
		h.sessionData["refresh_token"] = refreshToken
	}
	h.sessionData["token_obtained_at"] = time.Now().Format(time.RFC3339)
	h.mu.Unlock()

	return nil
}

// setCustomAuth 设置自定义认证
func (h *HTTPAuthDataSource) setCustomAuth(req *http.Request) error {
	// 对于自定义认证，主要通过脚本实现
	// 但这里可以设置一些基础的头部信息

	// 设置默认Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 如果有预设的头部信息，添加到请求中
	if extraHeaders, ok := h.credentials["headers"].(map[string]interface{}); ok {
		for key, value := range extraHeaders {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// 检查是否有会话数据需要添加到头部
	h.mu.RLock()
	sessionId, hasSessionId := h.sessionData["sessionId"]
	authToken, hasAuthToken := h.sessionData["auth_token"]
	accessToken, hasAccessToken := h.sessionData["access_token"]
	h.mu.RUnlock()

	// 如果有sessionId，添加到头部或查询参数
	if hasSessionId {
		if sessionIdStr, ok := sessionId.(string); ok && sessionIdStr != "" {
			// 检查配置决定sessionId放在头部还是查询参数
			if sessionIdHeader, ok := h.credentials["session_id_header"].(string); ok && sessionIdHeader != "" {
				req.Header.Set(sessionIdHeader, sessionIdStr)
			} else {
				// 默认添加到查询参数
				query := req.URL.Query()
				query.Add("sessionId", sessionIdStr)
				req.URL.RawQuery = query.Encode()
			}
		}
	}

	// 如果有认证令牌，添加到Authorization头部
	if hasAuthToken {
		if tokenStr, ok := authToken.(string); ok && tokenStr != "" {
			req.Header.Set("Authorization", "Bearer "+tokenStr)
		}
	} else if hasAccessToken {
		if tokenStr, ok := accessToken.(string); ok && tokenStr != "" {
			req.Header.Set("Authorization", "Bearer "+tokenStr)
		}
	}

	// 从custom_map中获取其他认证相关配置
	if authHeader, ok := h.credentials["auth_header"].(string); ok && authHeader != "" {
		if authValue, ok := h.credentials["auth_value"].(string); ok && authValue != "" {
			req.Header.Set(authHeader, authValue)
		}
	}

	return nil
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
		slog.Error("数据源 %s 没有活跃会话，跳过刷新\n", h.GetID())
		return
	}

	slog.Info("开始刷新数据源 %s 的会话\n", h.GetID())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 准备刷新脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "refresh"

	// 执行刷新脚本
	if h.scriptExecutor != nil {
		result, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
		if err != nil {
			slog.Error("数据源 %s 会话刷新失败: %v\n", h.GetID(), err.Error())
		} else {
			// 更新会话数据
			if resultMap, ok := result.(map[string]interface{}); ok {
				h.mu.Lock()
				for k, v := range resultMap {
					h.sessionData[k] = v
				}
				h.sessionData["lastRefreshTime"] = time.Now().Format(time.RFC3339)
				h.mu.Unlock()
				slog.Info("数据源 %s 会话刷新成功\n", h.GetID())
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

// urlEncodeQueryParam 对查询参数值进行URL编码
func (h *HTTPAuthDataSource) urlEncodeQueryParam(value string) string {
	// 使用url.QueryEscape进行编码，它会将空格编码为%20，冒号编码为%3A等
	return url.QueryEscape(value)
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
