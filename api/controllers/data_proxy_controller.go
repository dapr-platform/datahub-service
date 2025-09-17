/*
 * @module api/controllers/data_proxy_controller
 * @description 数据访问代理控制器，提供统一的数据共享网关功能
 * @architecture 分层架构 - 控制器层
 * @documentReference ai_docs/api_req.md
 * @stateFlow HTTP请求代理流程
 * @rules 实现鉴权、日志、限流和路由功能
 * @dependencies datahub-service/service/sharing, net/http, io
 * @refs ai_docs/requirements.md
 */

package controllers

import (
	"datahub-service/client"
	"datahub-service/service/models"
	"datahub-service/service/sharing"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// SimplifiedApplicationInfo 简化的应用信息响应结构
type SimplifiedApplicationInfo struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Path        string                    `json:"path"`
	Description *string                   `json:"description,omitempty"`
	Interfaces  []SimplifiedInterfaceInfo `json:"interfaces"`
}

// SimplifiedInterfaceInfo 简化的接口信息响应结构
type SimplifiedInterfaceInfo struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Path        string                `json:"path"`
	Type        string                `json:"type"`
	Description string                `json:"description,omitempty"`
	Fields      []SimplifiedFieldInfo `json:"fields"`
}

// SimplifiedFieldInfo 简化的字段信息响应结构
type SimplifiedFieldInfo struct {
	NameZh      string `json:"name_zh"`
	NameEn      string `json:"name_en"`
	DataType    string `json:"data_type"`
	Description string `json:"description,omitempty"`
	IsNullable  bool   `json:"is_nullable"`
	OrderNum    int    `json:"order_num"`
}

// DataProxyController 数据访问代理控制器
type DataProxyController struct {
	sharingService  *sharing.SharingService
	postgrestURL    string
	clientCache     map[string]*client.PostgRESTClient // API Key ID -> PostgREST Client
	cacheMutex      sync.RWMutex                       // 缓存读写锁
	clientTimeout   time.Duration
	refreshInterval time.Duration
	maxRetries      int
}

// NewDataProxyController 创建数据访问代理控制器实例
func NewDataProxyController(sharingService *sharing.SharingService) *DataProxyController {
	postgrestURL := os.Getenv("POSTGREST_URL")
	if postgrestURL == "" {
		postgrestURL = "http://postgrest:3000"
	}

	return &DataProxyController{
		sharingService:  sharingService,
		postgrestURL:    postgrestURL,
		clientCache:     make(map[string]*client.PostgRESTClient),
		clientTimeout:   30 * time.Second,
		refreshInterval: 55 * time.Minute, // 55分钟刷新一次Token
		maxRetries:      3,
	}
}

// getOrCreatePostgRESTClient 获取或创建指定API Key的PostgREST客户端
func (c *DataProxyController) getOrCreatePostgRESTClient(apiKeyID string, schema string) (*client.PostgRESTClient, error) {
	// 先尝试读取缓存
	c.cacheMutex.RLock()
	if client, exists := c.clientCache[apiKeyID]; exists {
		c.cacheMutex.RUnlock()
		return client, nil
	}
	c.cacheMutex.RUnlock()

	// 获取写锁，再次检查缓存（双重检查锁定模式）
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	if client, exists := c.clientCache[apiKeyID]; exists {
		return client, nil
	}

	// 创建新的PostgREST客户端配置
	config := &client.PostgRESTConfig{
		BaseURL:         c.postgrestURL,
		Username:        apiKeyID, // 使用API Key ID作为用户名
		Password:        apiKeyID, // 使用API Key ID作为密码
		Timeout:         c.clientTimeout,
		RefreshInterval: c.refreshInterval,
		MaxRetries:      c.maxRetries,
		Schema:          schema,
	}

	// 创建PostgREST客户端
	postgrestClient := client.NewPostgRESTClient(config)

	// 初始化连接并获取Token
	if err := postgrestClient.Connect(); err != nil {
		return nil, fmt.Errorf("PostgREST客户端初始化失败: %v", err)
	}

	// 缓存客户端
	c.clientCache[apiKeyID] = postgrestClient

	return postgrestClient, nil
}

// removePostgRESTClient 从缓存中移除指定的PostgREST客户端
func (c *DataProxyController) removePostgRESTClient(apiKeyID string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	if client, exists := c.clientCache[apiKeyID]; exists {
		// 关闭客户端连接
		client.Close()
		// 从缓存中删除
		delete(c.clientCache, apiKeyID)
	}
}

// ProxyDataAccess 数据访问代理处理器
// @Summary 数据访问代理（只读查询）
// @Description 代理对PostgREST的查询请求，实现统一的鉴权、日志、限流和路由功能，仅支持数据查询操作
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param app_path path string true "应用路径"
// @Param interface_path path string true "接口路径"
// @Param Authorization header string true "Bearer Token格式的API Key"
// @Success 200 {object} interface{} "查询成功"
// @Failure 401 {object} APIResponse "未授权"
// @Failure 404 {object} APIResponse "资源不存在"
// @Failure 429 {object} APIResponse "请求过于频繁"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /api/v1/share/{app_path}/{interface_path} [get]
// @Router /api/v1/share/{app_path}/{interface_path} [head]
func (c *DataProxyController) ProxyDataAccess(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 0. 验证HTTP方法，只允许GET和HEAD
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		c.logApiUsage(r, "", "", http.StatusMethodNotAllowed, time.Since(startTime), "不支持的HTTP方法: "+r.Method)
		w.Header().Set("Allow", "GET, HEAD")
		render.JSON(w, r, APIResponse{
			Status: http.StatusMethodNotAllowed,
			Msg:    "仅支持GET和HEAD方法进行数据查询",
		})
		return
	}

	// 1. 从URL中解析参数
	appPath := chi.URLParam(r, "app_path")             // 应用路径
	interfacePath := chi.URLParam(r, "interface_path") // 接口路径

	// 2. 鉴权中间件：从Authorization头中提取API Key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), "缺少Authorization头")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "缺少Authorization头",
		})
		return
	}

	// 解析Bearer Token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), "无效的Authorization格式")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "无效的Authorization格式，请使用Bearer Token",
		})
		return
	}

	apiKeyValue := strings.TrimPrefix(authHeader, "Bearer ")

	// 3. 验证API Key
	apiKey, err := c.sharingService.VerifyApiKey(apiKeyValue)
	if err != nil {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "API Key验证失败: " + err.Error(),
		})
		return
	}

	// 4. 根据应用路径和接口路径查询ApiInterface
	apiInterface, err := c.sharingService.GetApiInterfaceByAppPathAndInterfacePath(appPath, interfacePath)
	if err != nil {
		c.logApiUsage(r, apiKey.ApiApplicationID, "", http.StatusNotFound, time.Since(startTime), "接口不存在或已禁用")
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "接口不存在或已禁用",
		})
		return
	}

	// 5. 验证API Key是否属于该接口的应用
	if apiKey.ApiApplicationID != apiInterface.ApiApplicationID {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusUnauthorized, time.Since(startTime), "API Key与接口应用不匹配")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "API Key与接口应用不匹配",
		})
		return
	}

	// 6. 获取主题库schema和主题接口信息（table_name）
	schema := apiInterface.ApiApplication.ThematicLibrary.NameEn
	if schema == "" {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "主题库英文名为空")
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "主题库配置错误",
		})
		return
	}

	tableName := apiInterface.ThematicInterface.NameEn
	if tableName == "" {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "主题接口英文名为空")
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "主题接口配置错误",
		})
		return
	}

	// 9. 读取请求体
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "读取请求体失败")
			render.JSON(w, r, APIResponse{
				Status: http.StatusInternalServerError,
				Msg:    "读取请求体失败",
			})
			return
		}
	}

	// 10. 准备额外的请求头
	additionalHeaders := make(map[string]string)

	// 复制大部分请求头，排除一些敏感头
	for key, values := range r.Header {
		if key == "Authorization" || key == "Host" ||
			key == "Accept-Profile" || key == "Content-Profile" {
			continue
		}
		if len(values) > 0 {
			additionalHeaders[key] = values[0]
		}
	}

	// 设置schema头
	if r.Method == "GET" || r.Method == "HEAD" {
		additionalHeaders["Accept-Profile"] = schema
	} else {
		additionalHeaders["Content-Profile"] = schema
	}

	// 11. 获取或创建专用的PostgREST客户端
	postgrestClient, err := c.getOrCreatePostgRESTClient(apiKey.ID, schema)
	if err != nil {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "获取PostgREST客户端失败: "+err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "服务初始化失败",
		})
		return
	}

	// 12. 使用PostgREST客户端发送请求
	proxyResp, err := postgrestClient.ProxyRequest(r.Method, tableName, r.URL.RawQuery, bodyBytes, additionalHeaders)
	if err != nil {
		// 如果是认证错误，可能需要重新创建客户端
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
			c.removePostgRESTClient(apiKey.ID)
		}
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "代理请求失败: "+err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "代理请求失败",
		})
		return
	}
	defer proxyResp.Body.Close()

	// 13. 复制响应头
	for key, values := range proxyResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 14. 设置响应状态码
	w.WriteHeader(proxyResp.StatusCode)

	// 15. 流式返回响应体
	responseSize, err := io.Copy(w, proxyResp.Body)
	if err != nil {
		// 日志记录错误，但不能再返回HTTP响应了
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, proxyResp.StatusCode, time.Since(startTime), "复制响应体失败: "+err.Error())
		return
	}

	// 16. 记录成功的API使用日志
	c.logApiUsageWithSize(r, apiKey.ApiApplicationID, apiKey.ID, proxyResp.StatusCode, time.Since(startTime), "", int64(len(bodyBytes)), responseSize)
}

// logApiUsage 记录API使用日志
func (c *DataProxyController) logApiUsage(r *http.Request, appID, keyID string, statusCode int, duration time.Duration, errorMsg string) {
	c.logApiUsageWithSize(r, appID, keyID, statusCode, duration, errorMsg, 0, 0)
}

// logApiUsageWithSize 记录带大小信息的API使用日志
func (c *DataProxyController) logApiUsageWithSize(r *http.Request, appID, keyID string, statusCode int, duration time.Duration, errorMsg string, requestSize, responseSize int64) {
	log := &models.ApiUsageLog{
		ApiPath:      r.URL.Path,
		Method:       r.Method,
		RequestIP:    getClientIP(r),
		UserAgent:    getStringPointer(r.UserAgent()),
		StatusCode:   statusCode,
		ResponseTime: int(duration.Milliseconds()),
		RequestSize:  requestSize,
		ResponseSize: responseSize,
	}

	if appID != "" {
		log.ApplicationID = &appID
	}
	if keyID != "" {
		log.UserID = &keyID // 这里用UserID字段存储KeyID
	}
	if errorMsg != "" {
		log.ErrorMessage = &errorMsg
	}

	// 异步记录日志，不影响响应性能
	go func() {
		if err := c.sharingService.CreateApiUsageLog(log); err != nil {
			// 日志记录失败，可以考虑写入本地日志文件
			fmt.Printf("记录API使用日志失败: %v\n", err)
		}
	}()
}

// getClientIP 获取客户端IP地址
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	// 检查X-Real-IP头
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// 使用RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// getStringPointer 获取字符串指针
func getStringPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Close 关闭控制器并释放资源
func (c *DataProxyController) Close() error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// 关闭所有缓存的客户端
	for apiKeyID, client := range c.clientCache {
		if err := client.Close(); err != nil {
			fmt.Printf("关闭API Key %s 的PostgREST客户端失败: %v\n", apiKeyID, err)
		}
	}

	// 清空缓存
	c.clientCache = make(map[string]*client.PostgRESTClient)

	return nil
}

// GetPostgRESTClientStats 获取PostgREST客户端统计信息
func (c *DataProxyController) GetPostgRESTClientStats() map[string]interface{} {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_clients"] = len(c.clientCache)

	clientStats := make(map[string]interface{})
	for apiKeyID, client := range c.clientCache {
		clientStats[apiKeyID] = client.GetStatistics()
	}
	stats["clients"] = clientStats

	return stats
}

// ClearExpiredClients 清理过期或无效的客户端
func (c *DataProxyController) ClearExpiredClients() int {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	clearedCount := 0
	for apiKeyID, client := range c.clientCache {
		// 检查客户端状态，如果连接失败或过期，则移除
		stats := client.GetStatistics()
		if errorCount, ok := stats["error_count"].(int); ok && errorCount > 10 {
			// 如果错误次数过多，移除客户端
			client.Close()
			delete(c.clientCache, apiKeyID)
			clearedCount++
			fmt.Printf("清理高错误率的PostgREST客户端: %s\n", apiKeyID)
		}
	}

	return clearedCount
}

// GetCachedClientCount 获取缓存的客户端数量
func (c *DataProxyController) GetCachedClientCount() int {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	return len(c.clientCache)
}

// convertToSimplifiedApplicationInfo 将完整的应用信息转换为简化的响应结构
func convertToSimplifiedApplicationInfo(app *models.ApiApplication) *SimplifiedApplicationInfo {
	simplified := &SimplifiedApplicationInfo{
		ID:          app.ID,
		Name:        app.Name,
		Path:        app.Path,
		Description: app.Description,
		Interfaces:  make([]SimplifiedInterfaceInfo, 0, len(app.ApiInterfaces)),
	}

	for _, apiInterface := range app.ApiInterfaces {
		interfaceInfo := SimplifiedInterfaceInfo{
			ID:          apiInterface.ID,
			Name:        apiInterface.ThematicInterface.NameZh,
			Path:        apiInterface.Path,
			Type:        apiInterface.ThematicInterface.Type,
			Description: apiInterface.Description,
			Fields:      make([]SimplifiedFieldInfo, 0),
		}

		// 解析字段配置
		if apiInterface.ThematicInterface.TableFieldsConfig != nil {
			for _, fieldData := range apiInterface.ThematicInterface.TableFieldsConfig {
				if fieldMap, ok := fieldData.(map[string]interface{}); ok {
					field := SimplifiedFieldInfo{}

					if nameZh, ok := fieldMap["name_zh"].(string); ok {
						field.NameZh = nameZh
					}
					if nameEn, ok := fieldMap["name_en"].(string); ok {
						field.NameEn = nameEn
					}
					if dataType, ok := fieldMap["data_type"].(string); ok {
						field.DataType = dataType
					}
					if description, ok := fieldMap["description"].(string); ok {
						field.Description = description
					}
					if isNullable, ok := fieldMap["is_nullable"].(bool); ok {
						field.IsNullable = isNullable
					}
					if orderNum, ok := fieldMap["order_num"].(float64); ok {
						field.OrderNum = int(orderNum)
					}

					interfaceInfo.Fields = append(interfaceInfo.Fields, field)
				}
			}

			// 按照 order_num 排序字段
			sort.Slice(interfaceInfo.Fields, func(i, j int) bool {
				return interfaceInfo.Fields[i].OrderNum < interfaceInfo.Fields[j].OrderNum
			})
		}

		simplified.Interfaces = append(simplified.Interfaces, interfaceInfo)
	}

	return simplified
}

// GetApplicationInfo 获取应用信息和相关接口信息
// @Summary 获取API应用信息和接口列表
// @Description 根据应用路径获取API应用的详细信息以及该应用下的所有接口信息，包括主题接口的字段定义
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param app_path path string true "应用路径"
// @Param Authorization header string true "Bearer Token格式的API Key"
// @Success 200 {object} interface{} "获取成功"
// @Failure 401 {object} APIResponse "未授权"
// @Failure 404 {object} APIResponse "应用不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /api/v1/{app_path} [get]
func (c *DataProxyController) GetApplicationInfo(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 1. 验证HTTP方法，只允许GET
	if r.Method != http.MethodGet {
		c.logApiUsage(r, "", "", http.StatusMethodNotAllowed, time.Since(startTime), "不支持的HTTP方法: "+r.Method)
		w.Header().Set("Allow", "GET")
		render.JSON(w, r, APIResponse{
			Status: http.StatusMethodNotAllowed,
			Msg:    "仅支持GET方法获取应用信息",
		})
		return
	}

	// 2. 从URL中解析应用路径参数
	appPath := chi.URLParam(r, "app_path")
	if appPath == "" {
		c.logApiUsage(r, "", "", http.StatusBadRequest, time.Since(startTime), "应用路径参数为空")
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "应用路径参数不能为空",
		})
		return
	}

	// 3. 鉴权中间件：从Authorization头中提取API Key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), "缺少Authorization头")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "缺少Authorization头",
		})
		return
	}

	// 解析Bearer Token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), "无效的Authorization格式")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "无效的Authorization格式，请使用Bearer Token",
		})
		return
	}

	apiKeyValue := strings.TrimPrefix(authHeader, "Bearer ")

	// 4. 验证API Key
	apiKey, err := c.sharingService.VerifyApiKey(apiKeyValue)
	if err != nil {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "API Key验证失败: " + err.Error(),
		})
		return
	}

	// 5. 根据应用路径获取API应用信息及其接口信息
	appInfo, err := c.sharingService.GetApiApplicationByPath(appPath)
	if err != nil {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusNotFound, time.Since(startTime), "应用不存在或已禁用")
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "应用不存在或已禁用",
		})
		return
	}

	// 6. 验证API Key是否属于该应用
	if apiKey.ApiApplicationID != appInfo.ID {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusUnauthorized, time.Since(startTime), "API Key与应用不匹配")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "API Key与应用不匹配",
		})
		return
	}

	// 7. 记录成功的API使用日志
	c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusOK, time.Since(startTime), "")

	// 8. 转换为简化的响应结构并返回
	simplifiedInfo := convertToSimplifiedApplicationInfo(appInfo)
	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "获取应用信息成功",
		Data:   simplifiedInfo,
	})
}

// GetApiApplicationByKey 通过API Key获取应用信息和相关接口信息
// @Summary 通过API Key获取API应用信息和接口列表
// @Description 根据API Key获取该Key所属的API应用详细信息以及该应用下的所有接口信息，包括主题接口的字段定义
// @Tags 数据共享服务
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token格式的API Key"
// @Success 200 {object} interface{} "获取成功"
// @Failure 401 {object} APIResponse "未授权"
// @Failure 404 {object} APIResponse "应用不存在"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /api/v1/share/ [get]
func (c *DataProxyController) GetApiApplicationByKey(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 1. 验证HTTP方法，只允许GET
	if r.Method != http.MethodGet {
		c.logApiUsage(r, "", "", http.StatusMethodNotAllowed, time.Since(startTime), "不支持的HTTP方法: "+r.Method)
		w.Header().Set("Allow", "GET")
		render.JSON(w, r, APIResponse{
			Status: http.StatusMethodNotAllowed,
			Msg:    "仅支持GET方法获取应用信息",
		})
		return
	}

	// 2. 鉴权中间件：从Authorization头中提取API Key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), "缺少Authorization头")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "缺少Authorization头",
		})
		return
	}

	// 解析Bearer Token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), "无效的Authorization格式")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "无效的Authorization格式，请使用Bearer Token",
		})
		return
	}

	apiKeyValue := strings.TrimPrefix(authHeader, "Bearer ")

	// 3. 验证API Key
	apiKey, err := c.sharingService.VerifyApiKey(apiKeyValue)
	if err != nil {
		c.logApiUsage(r, "", "", http.StatusUnauthorized, time.Since(startTime), err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "API Key验证失败: " + err.Error(),
		})
		return
	}

	// 4. 根据API Key获取API应用信息及其接口信息
	appInfo, err := c.sharingService.GetApiApplicationByApiKey(apiKey.ID)
	if err != nil {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusNotFound, time.Since(startTime), "应用不存在或已禁用")
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "应用不存在或已禁用",
		})
		return
	}

	// 5. 记录成功的API使用日志
	c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusOK, time.Since(startTime), "")

	// 6. 转换为简化的响应结构并返回
	simplifiedInfo := convertToSimplifiedApplicationInfo(appInfo)
	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "获取应用信息成功",
		Data:   simplifiedInfo,
	})
}
