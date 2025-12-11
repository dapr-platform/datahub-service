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
	"context"
	"datahub-service/client"
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"datahub-service/service/rate_limiter"
	"datahub-service/service/sharing"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	sharingService    *sharing.SharingService
	governanceService *governance.GovernanceService
	rateLimiter       *rate_limiter.RedisRateLimiter
	postgrestURL      string
	clientCache       map[string]*client.PostgRESTClient // API Key ID -> PostgREST Client
	cacheMutex        sync.RWMutex                       // 缓存读写锁
	clientTimeout     time.Duration
	refreshInterval   time.Duration
	maxRetries        int
}

// NewDataProxyController 创建数据访问代理控制器实例
func NewDataProxyController(sharingService *sharing.SharingService, governanceService *governance.GovernanceService) *DataProxyController {
	postgrestURL := os.Getenv("POSTGREST_URL")
	if postgrestURL == "" {
		postgrestURL = "http://postgrest:3000"
	}

	// 初始化限流器
	rateLimiter, err := rate_limiter.NewRedisRateLimiter()
	if err != nil {
		slog.Error("初始化Redis限流器失败，限流功能将不可用", "error", err)
	}

	return &DataProxyController{
		sharingService:    sharingService,
		governanceService: governanceService,
		rateLimiter:       rateLimiter,
		postgrestURL:      postgrestURL,
		clientCache:       make(map[string]*client.PostgRESTClient),
		clientTimeout:     30 * time.Second,
		refreshInterval:   55 * time.Minute, // 55分钟刷新一次Token
		maxRetries:        3,
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
		c.logApiUsage(r, "", apiKey.ID, http.StatusNotFound, time.Since(startTime), "接口不存在或已禁用")
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "接口不存在或已禁用",
		})
		return
	}

	// 5. 验证API Key是否可以访问该接口的应用
	hasAccess, err := c.verifyApiKeyAccess(apiKey.ID, apiInterface.ApiApplicationID)
	if err != nil || !hasAccess {
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusUnauthorized, time.Since(startTime), "API Key无权访问该应用接口")
		render.JSON(w, r, APIResponse{
			Status: http.StatusUnauthorized,
			Msg:    "API Key无权访问该应用接口",
		})
		return
	}

	// 5.5. 检查限流（全局 -> 密钥 -> 应用）
	if c.rateLimiter != nil {
		rateLimitResult, err := c.checkRateLimit(r.Context(), apiKey.ID, apiInterface.ApiApplicationID)
		if err != nil {
			slog.Error("限流检查失败", "error", err)
			// 限流检查失败不影响正常流程，记录日志即可
		} else if !rateLimitResult.Allowed {
			c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusTooManyRequests, time.Since(startTime), rateLimitResult.Message)

			// 设置限流相关响应头
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitResult.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rateLimitResult.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", rateLimitResult.ResetAt))
			w.Header().Set("X-RateLimit-Type", rateLimitResult.RateLimitType)

			render.JSON(w, r, APIResponse{
				Status: http.StatusTooManyRequests,
				Msg:    rateLimitResult.Message,
			})
			return
		}
	}

	// 6. 获取主题库schema和主题接口信息（table_name）
	schema := apiInterface.ApiApplication.ThematicLibrary.NameEn
	if schema == "" {
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "主题库英文名为空")
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "主题库配置错误",
		})
		return
	}

	tableName := apiInterface.ThematicInterface.NameEn
	if tableName == "" {
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "主题接口英文名为空")
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
			c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "读取请求体失败")
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
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "获取PostgREST客户端失败: "+err.Error())
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
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "代理请求失败: "+err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "代理请求失败",
		})
		return
	}
	defer proxyResp.Body.Close()

	// 13. 读取响应体以便应用脱敏规则
	responseBody, err := io.ReadAll(proxyResp.Body)
	if err != nil {
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "读取响应体失败: "+err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "读取响应体失败",
		})
		return
	}

	// 14. 应用数据脱敏规则（如果配置了）
	var finalResponseBody []byte
	if len(apiInterface.MaskingRules) > 0 && proxyResp.StatusCode == http.StatusOK {
		// 解析脱敏规则
		var maskingConfigs []models.DataMaskingConfig
		for _, ruleValue := range apiInterface.MaskingRules {
			if ruleData, ok := ruleValue.(map[string]interface{}); ok {
				var rule models.DataMaskingConfig

				// 解析规则配置
				if templateID, ok := ruleData["template_id"].(string); ok {
					rule.TemplateID = templateID
				}
				if targetFields, ok := ruleData["target_fields"].([]interface{}); ok {
					for _, field := range targetFields {
						if fieldStr, ok := field.(string); ok {
							rule.TargetFields = append(rule.TargetFields, fieldStr)
						}
					}
				}
				if maskingConfig, ok := ruleData["masking_config"].(map[string]interface{}); ok {
					rule.MaskingConfig = maskingConfig
				}
				if applyCondition, ok := ruleData["apply_condition"].(string); ok {
					rule.ApplyCondition = applyCondition
				}
				if preserveFormat, ok := ruleData["preserve_format"].(bool); ok {
					rule.PreserveFormat = preserveFormat
				}
				if isEnabled, ok := ruleData["is_enabled"].(bool); ok {
					rule.IsEnabled = isEnabled
				} else {
					rule.IsEnabled = true
				}

				maskingConfigs = append(maskingConfigs, rule)
			}
		}

		// 应用脱敏处理
		maskedData, maskErr := c.applyMaskingToResponseData(responseBody, maskingConfigs)
		if maskErr != nil {
			// 脱敏失败记录日志但不中断请求，返回原始数据
			slog.Error("应用脱敏规则失败", "error", maskErr, "interface_id", apiInterface.ID)
			finalResponseBody = responseBody
		} else {
			finalResponseBody = maskedData
		}
	} else {
		finalResponseBody = responseBody
	}

	// 15. 复制响应头
	for key, values := range proxyResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 更新Content-Length头
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(finalResponseBody)))

	// 16. 设置响应状态码
	w.WriteHeader(proxyResp.StatusCode)

	// 17. 返回响应体
	responseSize, err := w.Write(finalResponseBody)
	if err != nil {
		// 日志记录错误，但不能再返回HTTP响应了
		c.logApiUsage(r, apiInterface.ApiApplicationID, apiKey.ID, proxyResp.StatusCode, time.Since(startTime), "写入响应体失败: "+err.Error())
		return
	}

	// 18. 记录成功的API使用日志
	c.logApiUsageWithSize(r, apiInterface.ApiApplicationID, apiKey.ID, proxyResp.StatusCode, time.Since(startTime), "", int64(len(bodyBytes)), int64(responseSize))
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
			slog.Error("记录API使用日志失败", "error", err)
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

// checkRateLimit 检查限流
func (c *DataProxyController) checkRateLimit(ctx context.Context, apiKeyID, applicationID string) (*rate_limiter.RateLimitResult, error) {
	// 获取适用的限流规则
	rateLimits, err := c.sharingService.GetApplicableRateLimits(apiKeyID, applicationID)
	if err != nil {
		return nil, fmt.Errorf("获取限流规则失败: %w", err)
	}

	// 如果没有限流规则，允许通过
	if len(rateLimits) == 0 {
		return &rate_limiter.RateLimitResult{
			Allowed:       true,
			Limit:         -1,
			Remaining:     -1,
			RateLimitType: "none",
			Message:       "无限流规则",
		}, nil
	}

	// 转换为限流器规则格式
	var rules []rate_limiter.RateLimitRule
	for _, limit := range rateLimits {
		rule := rate_limiter.RateLimitRule{
			Type:        limit.RateLimitType,
			TimeWindow:  limit.TimeWindow,
			MaxRequests: limit.MaxRequests,
		}
		if limit.TargetID != nil {
			rule.TargetID = *limit.TargetID
		}
		rules = append(rules, rule)
	}

	// 检查限流
	return c.rateLimiter.CheckRateLimit(ctx, rules)
}

// Close 关闭控制器并释放资源
func (c *DataProxyController) Close() error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// 关闭所有缓存的客户端
	for apiKeyID, client := range c.clientCache {
		if err := client.Close(); err != nil {
			slog.Error("关闭PostgREST客户端失败", "api_key_id", apiKeyID, "error", err)
		}
	}

	// 清空缓存
	c.clientCache = make(map[string]*client.PostgRESTClient)

	// 关闭限流器
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Close(); err != nil {
			slog.Error("关闭限流器失败", "error", err)
		}
	}

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
			slog.Warn("清理高错误率的PostgREST客户端", "api_key_id", apiKeyID, "error_count", errorCount)
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

// verifyApiKeyAccess 验证API Key是否可以访问指定的应用
func (c *DataProxyController) verifyApiKeyAccess(apiKeyID, appID string) (bool, error) {
	// 通过API Key获取可访问的应用列表
	apps, err := c.sharingService.GetApiApplicationsByApiKey(apiKeyID)
	if err != nil {
		return false, err
	}

	// 检查目标应用是否在可访问列表中
	for _, app := range apps {
		if app.ID == appID {
			return true, nil
		}
	}

	return false, nil
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

	// 5. 根据API Key和应用路径获取API应用信息及其接口信息
	appInfo, err := c.sharingService.GetApiApplicationByApiKeyAndPath(apiKey.ID, appPath)
	if err != nil {
		c.logApiUsage(r, "", apiKey.ID, http.StatusNotFound, time.Since(startTime), "应用不存在、已禁用或API Key无权访问")
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "应用不存在、已禁用或API Key无权访问",
		})
		return
	}

	// 6. 记录成功的API使用日志
	c.logApiUsage(r, appInfo.ID, apiKey.ID, http.StatusOK, time.Since(startTime), "")

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

	// 4. 根据API Key获取该Key可访问的所有API应用信息及其接口信息
	appInfos, err := c.sharingService.GetApiApplicationsByApiKey(apiKey.ID)
	if err != nil || len(appInfos) == 0 {
		c.logApiUsage(r, "", apiKey.ID, http.StatusNotFound, time.Since(startTime), "该API Key无可访问的应用")
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "该API Key无可访问的应用",
		})
		return
	}

	// 5. 记录成功的API使用日志（记录第一个应用的ID作为示例）
	c.logApiUsage(r, appInfos[0].ID, apiKey.ID, http.StatusOK, time.Since(startTime), "")

	// 6. 转换为简化的响应结构并返回所有应用信息
	var simplifiedInfos []SimplifiedApplicationInfo
	for _, appInfo := range appInfos {
		simplifiedInfo := convertToSimplifiedApplicationInfo(&appInfo)
		simplifiedInfos = append(simplifiedInfos, *simplifiedInfo)
	}
	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "获取应用信息成功",
		Data:   simplifiedInfos,
	})
}

// === 数据脱敏处理方法 ===

// applyMaskingToResponseData 对响应数据应用脱敏规则
func (c *DataProxyController) applyMaskingToResponseData(responseBody []byte, maskingConfigs []models.DataMaskingConfig) ([]byte, error) {
	// 解析JSON响应
	var data interface{}
	if err := json.Unmarshal(responseBody, &data); err != nil {
		// 如果不是JSON格式，直接返回原数据
		return responseBody, nil
	}

	// 处理不同的响应格式
	var maskedData interface{}
	var err error

	switch v := data.(type) {
	case map[string]interface{}:
		// 单条记录
		maskedData, err = c.maskSingleRecord(v, maskingConfigs)
	case []interface{}:
		// 多条记录
		maskedData, err = c.maskMultipleRecords(v, maskingConfigs)
	default:
		// 其他格式，不处理
		return responseBody, nil
	}

	if err != nil {
		return nil, err
	}

	// 将脱敏后的数据转回JSON
	maskedBytes, err := json.Marshal(maskedData)
	if err != nil {
		return nil, fmt.Errorf("序列化脱敏数据失败: %w", err)
	}

	return maskedBytes, nil
}

// maskSingleRecord 对单条记录应用脱敏
func (c *DataProxyController) maskSingleRecord(record map[string]interface{}, maskingConfigs []models.DataMaskingConfig) (map[string]interface{}, error) {
	if c.governanceService == nil {
		return record, nil
	}

	// 使用 GovernanceService 应用脱敏规则
	result, err := c.governanceService.ApplyMaskingRules(record, maskingConfigs)
	if err != nil {
		return record, err
	}

	return result.ProcessedData, nil
}

// maskMultipleRecords 对多条记录应用脱敏
func (c *DataProxyController) maskMultipleRecords(records []interface{}, maskingConfigs []models.DataMaskingConfig) ([]interface{}, error) {
	maskedRecords := make([]interface{}, len(records))

	for i, item := range records {
		if record, ok := item.(map[string]interface{}); ok {
			masked, err := c.maskSingleRecord(record, maskingConfigs)
			if err != nil {
				// 记录错误但继续处理其他记录
				slog.Error("脱敏记录失败", "index", i, "error", err)
				maskedRecords[i] = record
			} else {
				maskedRecords[i] = masked
			}
		} else {
			maskedRecords[i] = item
		}
	}

	return maskedRecords, nil
}
