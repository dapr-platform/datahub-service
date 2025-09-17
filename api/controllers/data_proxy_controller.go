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
	"bytes"
	"datahub-service/service/models"
	"datahub-service/service/sharing"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// DataProxyController 数据访问代理控制器
type DataProxyController struct {
	sharingService *sharing.SharingService
	postgrestURL   string
}

// NewDataProxyController 创建数据访问代理控制器实例
func NewDataProxyController(sharingService *sharing.SharingService) *DataProxyController {
	return &DataProxyController{
		sharingService: sharingService,
		postgrestURL:   "http://postgrest-service:3000", // TODO: 从配置文件读取
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

	// 9. 构造对PostgREST的请求
	targetURL := fmt.Sprintf("%s/%s", c.postgrestURL, tableName)

	// 复制请求体
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

	// 创建新的请求
	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "创建代理请求失败")
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "创建代理请求失败",
		})
		return
	}

	// 10. 复制查询参数
	proxyReq.URL.RawQuery = r.URL.RawQuery

	// 11. 复制大部分请求头，排除一些敏感头
	for key, values := range r.Header {
		if key == "Authorization" || key == "Host" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// 12. 添加关键的PostgREST头
	if r.Method == "GET" || r.Method == "HEAD" {
		proxyReq.Header.Set("Accept-Profile", schema)
	} else {
		proxyReq.Header.Set("Content-Profile", schema)
	}

	// 13. 发送请求到PostgREST
	client := &http.Client{Timeout: 30 * time.Second}
	proxyResp, err := client.Do(proxyReq)
	if err != nil {
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, http.StatusInternalServerError, time.Since(startTime), "代理请求失败: "+err.Error())
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "代理请求失败",
		})
		return
	}
	defer proxyResp.Body.Close()

	// 14. 复制响应头
	for key, values := range proxyResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 15. 设置响应状态码
	w.WriteHeader(proxyResp.StatusCode)

	// 16. 流式返回响应体
	responseSize, err := io.Copy(w, proxyResp.Body)
	if err != nil {
		// 日志记录错误，但不能再返回HTTP响应了
		c.logApiUsage(r, apiKey.ApiApplicationID, apiKey.ID, proxyResp.StatusCode, time.Since(startTime), "复制响应体失败: "+err.Error())
		return
	}

	// 17. 记录成功的API使用日志
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
