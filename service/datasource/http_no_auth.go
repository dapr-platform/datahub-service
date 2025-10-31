/*
 * @module service/basic_library/datasource/http_no_auth
 * @description HTTP无认证数据源实现，用于访问不需要认证的HTTP接口
 * @architecture 简单HTTP客户端模式 - 直接发送HTTP请求，无需认证处理
 * @documentReference ai_docs/datasource_req.md, service/meta/datasource.go
 * @stateFlow HTTP连接生命周期：初始化配置 -> 建立连接 -> 执行请求 -> 关闭连接
 * @rules 支持GET、POST、PUT、DELETE等HTTP方法，不处理任何认证逻辑
 * @dependencies net/http, encoding/json, time
 * @refs interface.go, base.go, http_auth.go
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
	"time"

	"datahub-service/service/meta"
	"datahub-service/service/models"
)

// HTTPNoAuthDataSource HTTP无认证数据源实现
type HTTPNoAuthDataSource struct {
	*BaseDataSource
	client  *http.Client
	baseURL string
}

// NewHTTPNoAuthDataSource 创建HTTP无认证数据源
func NewHTTPNoAuthDataSource() DataSourceInterface {
	base := NewBaseDataSource(meta.DataSourceTypeApiHTTP, false) // HTTP数据源通常不是常驻的
	return &HTTPNoAuthDataSource{
		BaseDataSource: base,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Init 初始化HTTP无认证数据源
func (h *HTTPNoAuthDataSource) Init(ctx context.Context, ds *models.DataSource) error {
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

// Start 启动HTTP无认证数据源
func (h *HTTPNoAuthDataSource) Start(ctx context.Context) error {
	if err := h.BaseDataSource.Start(ctx); err != nil {
		return err
	}

	// 如果启用了脚本执行，调用启动脚本
	ds := h.GetDataSource()
	if ds.ScriptEnabled && ds.Script != "" {
		if err := h.executeStartScript(ctx); err != nil {
			return fmt.Errorf("启动脚本执行失败: %v", err)
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
func (h *HTTPNoAuthDataSource) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	slog.Debug("HTTPNoAuthDataSource.Execute - 开始执行HTTP请求")
	slog.Debug("HTTPNoAuthDataSource.Execute - 请求操作", "value", request.Operation)
	slog.Debug("HTTPNoAuthDataSource.Execute - 请求查询", "value", request.Query)
	slog.Debug("HTTPNoAuthDataSource.Execute - 请求参数", "data", request.Params)
	slog.Debug("HTTPNoAuthDataSource.Execute - 请求数据", "data", request.Data)

	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
	}

	// 检查数据源状态
	if !h.IsInitialized() {
		slog.Error("HTTPNoAuthDataSource.Execute - 数据源未初始化")
		response.Error = "数据源未初始化"
		response.Duration = time.Since(startTime)
		return response, fmt.Errorf("数据源未初始化")
	}

	slog.Debug("HTTPNoAuthDataSource.Execute - 数据源已初始化，基础URL", "value", h.baseURL)

	// 如果启用了脚本执行，优先使用脚本
	ds := h.GetDataSource()
	if ds.ScriptEnabled && ds.Script != "" {
		slog.Debug("HTTPNoAuthDataSource.Execute - 使用脚本执行")
		scriptResult, err := h.executeScript(ctx, request)
		if err != nil {
			slog.Error("HTTPNoAuthDataSource.Execute - 脚本执行失败", "error", err)
			response.Error = fmt.Sprintf("脚本执行失败: %v", err)
			response.Duration = time.Since(startTime)
			return response, err
		}

		response.Success = true
		response.Data = scriptResult
		response.Duration = time.Since(startTime)
		return response, nil
	}

	slog.Debug("HTTPNoAuthDataSource.Execute - 使用默认HTTP请求处理")
	// 默认HTTP请求处理
	return h.executeHTTPRequest(ctx, request)
}

// executeHTTPRequest 执行HTTP请求
func (h *HTTPNoAuthDataSource) executeHTTPRequest(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	startTime := time.Now()
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 开始执行HTTP请求")

	response := &ExecuteResponse{
		Success:   false,
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// 从请求数据中获取配置信息
	var method string = "GET"
	var headers map[string]interface{}
	var body interface{}
	var dataPath string = "data"
	var urlPattern string = "suffix"

	if request.Data != nil {
		slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 解析请求数据配置")
		if requestData, ok := request.Data.(map[string]interface{}); ok {
			if m, exists := requestData["method"]; exists {
				if methodStr, ok := m.(string); ok {
					method = methodStr
				}
			}
			if h, exists := requestData["headers"]; exists {
				if headerMap, ok := h.(map[string]interface{}); ok {
					headers = headerMap
				}
			}
			if b, exists := requestData["body"]; exists {
				body = b
			}
			if dp, exists := requestData["data_path"]; exists {
				if dpStr, ok := dp.(string); ok {
					dataPath = dpStr
				}
			}
			if up, exists := requestData["url_pattern"]; exists {
				if upStr, ok := up.(string); ok {
					urlPattern = upStr
				}
			}
		}
	}

	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 请求配置: method=%s, dataPath=%s, urlPattern=%s\n",
		method, dataPath, urlPattern)
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 请求头", "data", headers)

	// 构建完整的请求URL
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 构建完整URL")
	fullURL, err := h.buildFullURL(request.Query, request.Params, urlPattern)
	if err != nil {
		slog.Error("HTTPNoAuthDataSource.executeHTTPRequest - 构建URL失败", "error", err)
		response.Error = fmt.Sprintf("构建URL失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 完整URL", "value", fullURL)

	// 准备请求体
	var reqBody io.Reader
	if body != nil && (method == "POST" || method == "PUT") {
		slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 准备请求体")
		jsonData, err := json.Marshal(body)
		if err != nil {
			slog.Error("HTTPNoAuthDataSource.executeHTTPRequest - 序列化请求数据失败", "error", err)
			response.Error = fmt.Sprintf("序列化请求数据失败: %v", err)
			response.Duration = time.Since(startTime)
			return response, err
		}
		reqBody = bytes.NewReader(jsonData)
		slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 请求体", "value", string(jsonData))
	}

	// 创建HTTP请求
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 创建HTTP请求: %s %s\n", method, fullURL)
	httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		slog.Error("HTTPNoAuthDataSource.executeHTTPRequest - 创建HTTP请求失败", "error", err)
		response.Error = fmt.Sprintf("创建HTTP请求失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	// 设置请求头
	if reqBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("User-Agent", "DataHub-Service/1.0")

	// 设置自定义请求头
	for key, value := range headers {
		httpReq.Header.Set(key, fmt.Sprintf("%v", value))
	}

	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 请求头设置完成", "data", httpReq.Header)

	// 执行请求
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 发送HTTP请求")
	httpResp, err := h.client.Do(httpReq)
	if err != nil {
		slog.Error("HTTPNoAuthDataSource.executeHTTPRequest - HTTP请求失败", "error", err)
		response.Error = fmt.Sprintf("HTTP请求失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}
	defer httpResp.Body.Close()

	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 收到响应，状态码", "count", httpResp.StatusCode)

	// 读取响应体
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 读取响应体")
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		slog.Error("HTTPNoAuthDataSource.executeHTTPRequest - 读取响应体失败", "error", err)
		response.Error = fmt.Sprintf("读取响应体失败: %v", err)
		response.Duration = time.Since(startTime)
		return response, err
	}

	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 响应体长度", "count", len(respBody))
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 响应体内容", "value", string(respBody))

	// 设置响应信息
	response.Duration = time.Since(startTime)
	response.Metadata["status_code"] = httpResp.StatusCode
	response.Metadata["headers"] = httpResp.Header
	response.Metadata["url"] = fullURL
	response.Metadata["method"] = method
	response.Metadata["data_path"] = dataPath

	// 使用响应解析器处理响应
	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 开始处理响应")
	if responseParserConfig, exists := request.Data.(map[string]interface{})["response_parser"]; exists {
		slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 使用响应解析器处理")
		if parserConfig, ok := responseParserConfig.(map[string]interface{}); ok {
			slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 解析器配置", "data", parserConfig)
			parser := NewResponseParser(parserConfig)
			parsedResponse, err := parser.Parse(httpResp.StatusCode, respBody, httpResp.Header)
			if err != nil {
				slog.Error("HTTPNoAuthDataSource.executeHTTPRequest - 响应解析失败", "error", err)
				response.Error = fmt.Sprintf("响应解析失败: %v", err)
				response.Data = string(respBody)
			} else {
				slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 响应解析成功: success=%t\n", parsedResponse.Success)
				slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 解析后数据类型: %T\n", parsedResponse.Data)

				response.Success = parsedResponse.Success
				response.Data = parsedResponse.Data

				// 添加解析后的元数据
				if parsedResponse.ErrorMessage != "" {
					slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 错误消息", "value", parsedResponse.ErrorMessage)
					response.Error = parsedResponse.ErrorMessage
				}

				// 将分页信息添加到元数据中
				if parsedResponse.Total > 0 {
					response.Metadata["total"] = parsedResponse.Total
				}
				if parsedResponse.Page > 0 {
					response.Metadata["page"] = parsedResponse.Page
				}
				if parsedResponse.PageSize > 0 {
					response.Metadata["page_size"] = parsedResponse.PageSize
				}
				if parsedResponse.ErrorCode != "" {
					response.Metadata["error_code"] = parsedResponse.ErrorCode
				}

				// 合并解析器的元数据
				for k, v := range parsedResponse.Metadata {
					response.Metadata[k] = v
				}

				slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 最终响应元数据", "data", response.Metadata)
			}
		} else {
			slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 解析器配置类型错误，使用回退处理")
			// 回退到简单的状态码判断
			h.handleResponseFallback(httpResp.StatusCode, respBody, dataPath, response)
		}
	} else {
		slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 没有响应解析器配置，使用回退处理")
		// 回退到简单的状态码判断
		h.handleResponseFallback(httpResp.StatusCode, respBody, dataPath, response)
	}

	slog.Debug("HTTPNoAuthDataSource.executeHTTPRequest - 响应处理完成: success=%t, error=%s\n",
		response.Success, response.Error)

	return response, nil
}

// buildFullURL 构建完整的URL
func (h *HTTPNoAuthDataSource) buildFullURL(urlPath string, params map[string]interface{}, urlPattern string) (string, error) {
	baseURL := strings.TrimRight(h.baseURL, "/")

	// 构建路径部分
	var fullURL string
	if urlPath != "" && urlPath != "/" {
		if strings.HasPrefix(urlPath, "/") {
			fullURL = baseURL + urlPath
		} else {
			fullURL = baseURL + "/" + urlPath
		}
	} else {
		fullURL = baseURL
	}

	// 添加查询参数
	if len(params) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", fmt.Errorf("解析URL失败: %w", err)
		}

		query := u.Query()
		for key, value := range params {
			// 跳过元数据字段
			if key == "method" || key == "headers" || key == "body" || key == "use_form_data" {
				continue
			}
			query.Set(key, fmt.Sprintf("%v", value))
		}
		u.RawQuery = query.Encode()
		fullURL = u.String()
	}

	return fullURL, nil
}

// extractDataByPath 根据路径提取数据
func (h *HTTPNoAuthDataSource) extractDataByPath(data interface{}, path string) interface{} {
	if path == "" || path == "." {
		return data
	}

	// 分割路径
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			if value, exists := v[part]; exists {
				current = value
			} else {
				// 如果路径不存在，返回原始数据
				return data
			}
		case []interface{}:
			// 如果当前是数组，尝试提取数组中每个元素的指定字段
			var results []interface{}
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if value, exists := itemMap[part]; exists {
						results = append(results, value)
					}
				}
			}
			if len(results) > 0 {
				current = results
			} else {
				return data
			}
		default:
			// 无法继续解析路径，返回原始数据
			return data
		}
	}

	return current
}

// handleResponseFallback 回退处理响应（当没有响应解析配置时使用）
func (h *HTTPNoAuthDataSource) handleResponseFallback(statusCode int, respBody []byte, dataPath string, response *ExecuteResponse) {
	slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - 使用回退处理, 状态码", "count", statusCode)
	slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - 数据路径", "value", dataPath)

	// 检查HTTP状态码
	if statusCode >= 200 && statusCode < 300 {
		slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - 状态码成功，解析响应体")
		response.Success = true

		// 尝试解析JSON响应
		var jsonData interface{}
		if err := json.Unmarshal(respBody, &jsonData); err == nil {
			slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - JSON解析成功，数据类型: %T\n", jsonData)
			// 根据数据路径提取数据
			extractedData := h.extractDataByPath(jsonData, dataPath)
			slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - 数据提取完成，提取后类型: %T\n", extractedData)
			response.Data = extractedData
		} else {
			slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - JSON解析失败，使用原始字符串", "value", err)
			response.Data = string(respBody)
		}
	} else {
		slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - 状态码失败", "count", statusCode)
		response.Error = fmt.Sprintf("HTTP请求失败，状态码: %d, 响应: %s", statusCode, string(respBody))
		response.Data = string(respBody)
	}

	slog.Debug("HTTPNoAuthDataSource.handleResponseFallback - 回退处理完成: success=%t\n", response.Success)
}

// Stop 停止HTTP无认证数据源
func (h *HTTPNoAuthDataSource) Stop(ctx context.Context) error {
	// 如果启用了脚本执行，调用停止脚本
	ds := h.GetDataSource()
	if ds != nil && ds.ScriptEnabled && ds.Script != "" {
		if err := h.executeStopScript(ctx); err != nil {
			// 记录错误但不阻止停止流程
			slog.Error("停止脚本执行失败: %v\n", err.Error())
		}
	}

	return h.BaseDataSource.Stop(ctx)
}

// HealthCheck HTTP数据源健康检查
func (h *HTTPNoAuthDataSource) HealthCheck(ctx context.Context) (*HealthStatus, error) {
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

// testConnection 测试连接
func (h *HTTPNoAuthDataSource) testConnection(ctx context.Context) error {
	// 创建简单的HEAD请求测试连接
	req, err := http.NewRequestWithContext(ctx, "HEAD", h.baseURL, nil)
	if err != nil {
		// 如果HEAD请求创建失败，尝试GET请求
		req, err = http.NewRequestWithContext(ctx, "GET", h.baseURL, nil)
		if err != nil {
			return fmt.Errorf("创建测试请求失败: %v", err)
		}
	}

	// 设置User-Agent
	req.Header.Set("User-Agent", "DataHub-Service/1.0")

	// 执行请求
	resp, err := h.client.Do(req)
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
func (h *HTTPNoAuthDataSource) executeScript(ctx context.Context, request *ExecuteRequest) (interface{}, error) {
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
func (h *HTTPNoAuthDataSource) executeInitScript(ctx context.Context) error {
	ds := h.GetDataSource()
	if h.scriptExecutor == nil {
		return fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "init"

	_, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
	return err
}

// executeStartScript 执行启动脚本
func (h *HTTPNoAuthDataSource) executeStartScript(ctx context.Context) error {
	ds := h.GetDataSource()
	if h.scriptExecutor == nil {
		return fmt.Errorf("脚本执行器未初始化")
	}

	// 准备脚本参数
	params := h.prepareScriptParams(ctx)
	params["operation"] = "start"

	_, err := h.scriptExecutor.Execute(ctx, ds.Script, params)
	return err
}

// executeStopScript 执行停止脚本
func (h *HTTPNoAuthDataSource) executeStopScript(ctx context.Context) error {
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
func (h *HTTPNoAuthDataSource) prepareScriptParams(ctx context.Context) map[string]interface{} {
	ds := h.GetDataSource()

	params := make(map[string]interface{})
	params["dataSource"] = ds
	params["baseURL"] = h.baseURL
	params["httpClient"] = h.client
	params["context"] = ctx

	// 添加辅助函数
	params["httpPost"] = h.createHTTPPostHelper()
	params["httpGet"] = h.createHTTPGetHelper()

	return params
}

// createHTTPPostHelper 创建HTTP POST辅助函数
func (h *HTTPNoAuthDataSource) createHTTPPostHelper() func(string, map[string]interface{}) (map[string]interface{}, error) {
	return func(url string, data map[string]interface{}) (map[string]interface{}, error) {
		// 将数据转换为JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "DataHub-Service/1.0")

		resp, err := h.client.Do(req)
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
func (h *HTTPNoAuthDataSource) createHTTPGetHelper() func(string) (map[string]interface{}, error) {
	return func(url string) (map[string]interface{}, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "DataHub-Service/1.0")

		resp, err := h.client.Do(req)
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
