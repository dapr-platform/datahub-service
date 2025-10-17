/*
 * @module client/postgrest_client
 * @description PostgREST HTTP客户端，提供Token管理和自动刷新功能
 * @architecture 适配器模式 - 封装PostgREST认证和HTTP请求
 * @documentReference ai_docs/postgrest_rbac_guide.md
 * @stateFlow Token获取 -> Token使用 -> Token刷新 -> 重试机制
 * @rules 自动Token管理、错误重试、连接池复用
 * @dependencies net/http, encoding/json, sync, time
 * @refs api/controllers/data_proxy_controller.go
 */

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
	"log/slog"
)

// PostgRESTClient PostgREST HTTP客户端
type PostgRESTClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	schema     string
	// Token管理
	accessToken  string
	refreshToken string
	tokenExpiry  time.Time
	tokenMutex   sync.RWMutex

	// 自动刷新
	refreshTicker *time.Ticker
	stopRefresh   chan bool
	ctx           context.Context
	cancel        context.CancelFunc

	// 统计信息
	stats *ClientStats
}

// ClientStats 客户端统计信息
type ClientStats struct {
	RequestCount    int64     `json:"request_count"`     // 请求总数
	SuccessCount    int64     `json:"success_count"`     // 成功请求数
	ErrorCount      int64     `json:"error_count"`       // 错误请求数
	TokenRefreshed  int       `json:"token_refreshed"`   // Token刷新次数
	LastTokenTime   time.Time `json:"last_token_time"`   // 最后获取Token时间
	LastRequestTime time.Time `json:"last_request_time"` // 最后请求时间
	mutex           sync.RWMutex
}

// TokenResponse Token响应结构
type TokenResponse struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	AccessToken      string   `json:"access_token"`
	RefreshToken     string   `json:"refresh_token"`
	AccessExpiresIn  int      `json:"access_expires_in"`
	RefreshExpiresIn int      `json:"refresh_expires_in"`
	Username         string   `json:"username"`
	Roles            []string `json:"roles"`
	Permissions      []string `json:"permissions"`
	IsSuperuser      bool     `json:"is_superuser"`
	UserInfo         struct {
		Username    string    `json:"username"`
		Email       string    `json:"email"`
		FullName    string    `json:"full_name"`
		DisplayName string    `json:"display_name"`
		IsActive    bool      `json:"is_active"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"user_info"`
}

// RefreshTokenResponse 刷新Token响应结构
type RefreshTokenResponse struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	AccessToken      string   `json:"access_token"`
	RefreshToken     string   `json:"refresh_token,omitempty"` // 可选，仅在轮换时返回
	AccessExpiresIn  int      `json:"access_expires_in"`
	RefreshExpiresIn int      `json:"refresh_expires_in,omitempty"` // 可选，仅在轮换时返回
	Username         string   `json:"username"`
	Roles            []string `json:"roles"`
	Permissions      []string `json:"permissions"`
	IsSuperuser      bool     `json:"is_superuser"`
	UserInfo         struct {
		Username    string    `json:"username"`
		Email       string    `json:"email"`
		FullName    string    `json:"full_name"`
		DisplayName string    `json:"display_name"`
		IsActive    bool      `json:"is_active"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"user_info"`
}

// PostgRESTConfig PostgREST客户端配置
type PostgRESTConfig struct {
	BaseURL         string        `json:"base_url"`         // PostgREST服务地址
	Username        string        `json:"username"`         // 数据库用户名
	Password        string        `json:"password"`         // 数据库密码
	Timeout         time.Duration `json:"timeout"`          // HTTP超时时间
	RefreshInterval time.Duration `json:"refresh_interval"` // Token刷新间隔
	MaxRetries      int           `json:"max_retries"`      // 最大重试次数
	Schema          string        `json:"schema"`           // 数据库模式
}

// NewPostgRESTClient 创建新的PostgREST客户端
func NewPostgRESTClient(config *PostgRESTConfig) *PostgRESTClient {
	ctx, cancel := context.WithCancel(context.Background())

	client := &PostgRESTClient{
		baseURL:  config.BaseURL,
		username: config.Username,
		password: config.Password,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		ctx:    ctx,
		cancel: cancel,
		stats:  &ClientStats{},
		schema: config.Schema,
	}

	// 设置默认刷新间隔（Token过期前5分钟刷新）
	refreshInterval := config.RefreshInterval
	if refreshInterval == 0 {
		refreshInterval = 55 * time.Minute // 默认55分钟刷新一次
	}

	// 启动Token自动刷新
	client.startTokenRefresh(refreshInterval)

	return client
}

// Connect 建立连接并获取初始Token
func (c *PostgRESTClient) Connect() error {
	return c.getInitialToken()
}

// getInitialToken 获取初始Token
func (c *PostgRESTClient) getInitialToken() error {
	tokenReq := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	reqBody, err := json.Marshal(tokenReq)
	if err != nil {
		return fmt.Errorf("序列化Token请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/rpc/get_token", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("创建Token请求失败: %v", err)
	}

	// 设置必要的请求头
	req.Header.Set("Accept-Profile", "postgrest")
	req.Header.Set("Content-Profile", "postgrest")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Token请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Token请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("解析Token响应失败: %v", err)
	}

	if !tokenResp.Success {
		slog.Error("Token获取失败: 服务器返回success=false, 响应: %v", tokenResp)
		return fmt.Errorf("Token获取失败: 服务器返回success=false")
	}

	// 更新Token信息
	c.tokenMutex.Lock()
	c.accessToken = tokenResp.AccessToken
	c.refreshToken = tokenResp.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.AccessExpiresIn) * time.Second)
	c.tokenMutex.Unlock()

	// 更新统计信息
	c.stats.mutex.Lock()
	c.stats.LastTokenTime = time.Now()
	c.stats.mutex.Unlock()

	return nil
}

// refreshTokenIfNeeded 如果需要则刷新Token
func (c *PostgRESTClient) refreshTokenIfNeeded() error {
	c.tokenMutex.RLock()
	needRefresh := time.Now().Add(5 * time.Minute).After(c.tokenExpiry) // 提前5分钟刷新
	currentRefreshToken := c.refreshToken
	c.tokenMutex.RUnlock()

	if !needRefresh || currentRefreshToken == "" {
		return nil
	}

	return c.refreshAccessToken()
}

// refreshAccessToken 刷新访问Token
func (c *PostgRESTClient) refreshAccessToken() error {
	c.tokenMutex.RLock()
	currentRefreshToken := c.refreshToken
	c.tokenMutex.RUnlock()

	if currentRefreshToken == "" {
		return fmt.Errorf("没有可用的刷新Token")
	}

	refreshReq := map[string]interface{}{
		"refresh_token":        currentRefreshToken,
		"rotate_refresh_token": true,
	}

	reqBody, err := json.Marshal(refreshReq)
	if err != nil {
		return fmt.Errorf("序列化刷新Token请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/rpc/refresh_token", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("创建刷新Token请求失败: %v", err)
	}

	// 设置必要的请求头
	req.Header.Set("Accept-Profile", "postgrest")
	req.Header.Set("Content-Profile", "postgrest")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("刷新Token请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("刷新Token失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var refreshResp RefreshTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return fmt.Errorf("解析刷新Token响应失败: %v", err)
	}

	if !refreshResp.Success {
		return fmt.Errorf("刷新Token失败: 服务器返回success=false")
	}

	// 更新Token信息
	c.tokenMutex.Lock()
	c.accessToken = refreshResp.AccessToken
	// 只有在轮换时才更新refresh token
	if refreshResp.RefreshToken != "" {
		c.refreshToken = refreshResp.RefreshToken
	}
	c.tokenExpiry = time.Now().Add(time.Duration(refreshResp.AccessExpiresIn) * time.Second)
	c.tokenMutex.Unlock()

	// 更新统计信息
	c.stats.mutex.Lock()
	c.stats.TokenRefreshed++
	c.stats.LastTokenTime = time.Now()
	c.stats.mutex.Unlock()

	return nil
}

// startTokenRefresh 启动Token自动刷新
func (c *PostgRESTClient) startTokenRefresh(interval time.Duration) {
	c.refreshTicker = time.NewTicker(interval)
	c.stopRefresh = make(chan bool)

	go func() {
		for {
			select {
			case <-c.refreshTicker.C:
				if err := c.refreshTokenIfNeeded(); err != nil {
					// 如果刷新失败，尝试重新获取Token
					if err := c.getInitialToken(); err != nil {
						// Token获取失败，记录错误但继续尝试
						c.stats.mutex.Lock()
						c.stats.ErrorCount++
						c.stats.mutex.Unlock()
					}
				}
			case <-c.stopRefresh:
				return
			case <-c.ctx.Done():
				return
			}
		}
	}()
}

// MakeRequest 发起HTTP请求（带Token认证）
func (c *PostgRESTClient) MakeRequest(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	// 确保Token有效
	if err := c.refreshTokenIfNeeded(); err != nil {
		return nil, fmt.Errorf("Token刷新失败: %v", err)
	}

	// 构建完整URL
	fullURL := c.baseURL + path

	// 创建请求
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, fullURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, fullURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置认证头
	c.tokenMutex.RLock()
	accessToken := c.accessToken
	c.tokenMutex.RUnlock()

	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	// 设置PostgREST必要的头
	req.Header.Set("Accept-Profile", c.schema)
	if method != "GET" && method != "HEAD" {
		req.Header.Set("Content-Profile", c.schema)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置Accept头，确保返回JSON格式
	req.Header.Set("Accept", "application/json")

	// 设置自定义头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 更新统计信息
	c.stats.mutex.Lock()
	c.stats.RequestCount++
	c.stats.LastRequestTime = time.Now()
	c.stats.mutex.Unlock()

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.stats.mutex.Lock()
		c.stats.ErrorCount++
		c.stats.mutex.Unlock()
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}

	// 更新成功统计
	if resp.StatusCode < 400 {
		c.stats.mutex.Lock()
		c.stats.SuccessCount++
		c.stats.mutex.Unlock()
	} else {
		c.stats.mutex.Lock()
		c.stats.ErrorCount++
		c.stats.mutex.Unlock()
	}

	return resp, nil
}

// ProxyRequest 代理请求到PostgREST（用于数据查询）
func (c *PostgRESTClient) ProxyRequest(method, tableName, queryParams string, body []byte, additionalHeaders map[string]string) (*http.Response, error) {
	path := "/" + tableName
	if queryParams != "" {
		path += "?" + queryParams
	}

	return c.MakeRequest(method, path, body, additionalHeaders)
}

// CheckSchemaAccess 检查当前用户对指定schema的访问权限
func (c *PostgRESTClient) CheckSchemaAccess(schemaName string) error {
	// 构建权限检查查询
	query := fmt.Sprintf("select=schema_name&schema_name=eq.%s", schemaName)

	resp, err := c.MakeRequest("GET", "/information_schema.schemata?"+query, nil, nil)
	if err != nil {
		return fmt.Errorf("检查schema权限失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return fmt.Errorf("没有访问schema '%s' 的权限", schemaName)
	} else if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("权限检查失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAccessToken 获取当前访问Token
func (c *PostgRESTClient) GetAccessToken() string {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.accessToken
}

// IsTokenValid 检查Token是否有效
func (c *PostgRESTClient) IsTokenValid() bool {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.accessToken != "" && time.Now().Before(c.tokenExpiry)
}

// GetTokenExpiry 获取Token过期时间
func (c *PostgRESTClient) GetTokenExpiry() time.Time {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.tokenExpiry
}

// GetStatistics 获取客户端统计信息
func (c *PostgRESTClient) GetStatistics() map[string]interface{} {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	c.tokenMutex.RLock()
	tokenValid := c.accessToken != "" && time.Now().Before(c.tokenExpiry)
	tokenExpiry := c.tokenExpiry
	c.tokenMutex.RUnlock()

	return map[string]interface{}{
		"base_url":          c.baseURL,
		"username":          c.username,
		"request_count":     c.stats.RequestCount,
		"success_count":     c.stats.SuccessCount,
		"error_count":       c.stats.ErrorCount,
		"token_refreshed":   c.stats.TokenRefreshed,
		"last_token_time":   c.stats.LastTokenTime,
		"last_request_time": c.stats.LastRequestTime,
		"token_valid":       tokenValid,
		"token_expiry":      tokenExpiry,
	}
}

// Close 关闭客户端
func (c *PostgRESTClient) Close() error {
	// 停止Token自动刷新
	if c.refreshTicker != nil {
		c.refreshTicker.Stop()
	}
	if c.stopRefresh != nil {
		close(c.stopRefresh)
	}

	// 撤销刷新Token
	if c.refreshToken != "" {
		c.revokeRefreshToken()
	}

	c.cancel()
	return nil
}

// revokeRefreshToken 撤销刷新Token
func (c *PostgRESTClient) revokeRefreshToken() error {
	c.tokenMutex.RLock()
	currentRefreshToken := c.refreshToken
	currentAccessToken := c.accessToken
	c.tokenMutex.RUnlock()

	if currentRefreshToken == "" {
		return nil
	}

	revokeReq := map[string]string{
		"refresh_token": currentRefreshToken,
	}

	reqBody, err := json.Marshal(revokeReq)
	if err != nil {
		return fmt.Errorf("序列化撤销Token请求失败: %v", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/rpc/revoke_refresh_token", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("创建撤销Token请求失败: %v", err)
	}

	// 设置必要的请求头
	req.Header.Set("Accept-Profile", "postgrest")
	req.Header.Set("Content-Profile", "postgrest")
	req.Header.Set("Content-Type", "application/json")
	if currentAccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+currentAccessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("撤销Token请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 清空Token
	c.tokenMutex.Lock()
	c.accessToken = ""
	c.refreshToken = ""
	c.tokenExpiry = time.Time{}
	c.tokenMutex.Unlock()

	return nil
}
