/*
 * @module api/middleware/postgrest_auth
 * @description PostgREST Token鉴权中间件，验证JWT Token的有效性
 * @architecture 中间件模式 - HTTP请求拦截和验证
 * @documentReference deploy/local_dev/datahub/docker-compose/db/postgrest.sql
 * @stateFlow Token提取 -> Token验证 -> 上下文注入 -> 下一个处理器
 * @rules 统一鉴权、安全验证、错误处理
 * @dependencies net/http, encoding/json, strings, context
 * @refs client/postgrest_client.go, api/routes.go
 */

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/render"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	// TokenKey Token在上下文中的键
	TokenKey ContextKey = "token"
	// UserInfoKey 用户信息在上下文中的键
	UserInfoKey ContextKey = "user_info"
)

// TokenVerificationResponse Token验证响应结构
type TokenVerificationResponse struct {
	Success     bool       `json:"success"`
	Valid       bool       `json:"valid"`
	Message     string     `json:"message"`
	Username    string     `json:"username"`
	Roles       []string   `json:"roles"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// UserInfo 用户信息结构
type UserInfo struct {
	Username    string    `json:"username"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// PostgRESTAuthMiddleware PostgREST认证中间件
type PostgRESTAuthMiddleware struct {
	postgrestURL string
	httpClient   *http.Client
	// Token验证结果缓存（可选优化）
	cache      map[string]*cacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	// 白名单路径（不需要鉴权）
	whitelistPaths []string
}

// cacheEntry 缓存条目
type cacheEntry struct {
	userInfo  *UserInfo
	expiresAt time.Time
}

// NewPostgRESTAuthMiddleware 创建PostgREST认证中间件实例
func NewPostgRESTAuthMiddleware() *PostgRESTAuthMiddleware {
	postgrestURL := os.Getenv("POSTGREST_URL")
	if postgrestURL == "" {
		postgrestURL = "http://postgrest:3000"
	}

	return &PostgRESTAuthMiddleware{
		postgrestURL: postgrestURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 5 * time.Minute, // 缓存5分钟
		whitelistPaths: []string{
			"/health",       // 健康检查
			"/ready",        // 就绪检查
			"/swagger",      // Swagger文档
			"/api/v1/share", // 数据访问代理API（有自己的鉴权机制）
		},
	}
}

// AddWhitelistPath 添加白名单路径
func (m *PostgRESTAuthMiddleware) AddWhitelistPath(path string) {
	m.whitelistPaths = append(m.whitelistPaths, path)
}

// IsWhitelistPath 检查路径是否在白名单中
func (m *PostgRESTAuthMiddleware) IsWhitelistPath(path string) bool {
	for _, whitelistPath := range m.whitelistPaths {
		// 支持前缀匹配
		if strings.HasPrefix(path, whitelistPath) {
			return true
		}
	}
	return false
}

// Middleware 认证中间件处理函数
func (m *PostgRESTAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否在白名单中
		if m.IsWhitelistPath(r.URL.Path) {
			// 白名单路径，跳过鉴权
			next.ServeHTTP(w, r)
			return
		}

		// 从Authorization头中提取Token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.respondUnauthorized(w, r, "缺少Authorization头")
			return
		}

		// 验证Bearer格式
		if !strings.HasPrefix(authHeader, "Bearer ") {
			m.respondUnauthorized(w, r, "无效的Authorization格式，需要Bearer Token")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			m.respondUnauthorized(w, r, "Token为空")
			return
		}

		// 先检查缓存
		if userInfo := m.getFromCache(token); userInfo != nil {
			// 缓存命中，直接使用
			ctx := context.WithValue(r.Context(), TokenKey, token)
			ctx = context.WithValue(ctx, UserInfoKey, userInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 验证Token
		userInfo, err := m.verifyToken(token)
		if err != nil {
			m.respondUnauthorized(w, r, fmt.Sprintf("Token验证失败: %v", err))
			return
		}

		// 保存到缓存
		m.saveToCache(token, userInfo)

		// 将Token和用户信息注入到上下文中
		ctx := context.WithValue(r.Context(), TokenKey, token)
		ctx = context.WithValue(ctx, UserInfoKey, userInfo)

		// 调用下一个处理器
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// verifyToken 调用PostgREST验证Token
func (m *PostgRESTAuthMiddleware) verifyToken(token string) (*UserInfo, error) {
	// 构建验证请求
	verifyReq := map[string]string{
		"token": token,
	}

	reqBody, err := json.Marshal(verifyReq)
	if err != nil {
		return nil, fmt.Errorf("序列化验证请求失败: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", m.postgrestURL+"/rpc/verify_token", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建验证请求失败: %v", err)
	}

	// 设置必要的请求头
	req.Header.Set("Accept-Profile", "postgrest")
	req.Header.Set("Content-Profile", "postgrest")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("验证请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取验证响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("验证请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var verifyResp TokenVerificationResponse
	if err := json.Unmarshal(respBody, &verifyResp); err != nil {
		return nil, fmt.Errorf("解析验证响应失败: %v, 响应: %s", err, string(respBody))
	}

	// 检查验证结果
	if !verifyResp.Success || !verifyResp.Valid {
		return nil, fmt.Errorf("Token无效: %s", verifyResp.Message)
	}

	// 构建用户信息
	userInfo := &UserInfo{
		Username:    verifyResp.Username,
		Roles:       verifyResp.Roles,
		Permissions: verifyResp.Permissions,
	}

	// 设置过期时间
	if verifyResp.ExpiresAt != nil {
		userInfo.ExpiresAt = *verifyResp.ExpiresAt
	} else {
		// 如果没有过期时间，默认1小时后过期
		userInfo.ExpiresAt = time.Now().Add(1 * time.Hour)
	}

	return userInfo, nil
}

// getFromCache 从缓存中获取用户信息
func (m *PostgRESTAuthMiddleware) getFromCache(token string) *UserInfo {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	entry, exists := m.cache[token]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Now().After(entry.expiresAt) {
		// 异步删除过期缓存
		go m.removeFromCache(token)
		return nil
	}

	return entry.userInfo
}

// saveToCache 保存用户信息到缓存
func (m *PostgRESTAuthMiddleware) saveToCache(token string, userInfo *UserInfo) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	// 计算缓存过期时间（取Token过期时间和缓存TTL的较小值）
	cacheExpiry := time.Now().Add(m.cacheTTL)
	if userInfo.ExpiresAt.Before(cacheExpiry) {
		cacheExpiry = userInfo.ExpiresAt
	}

	m.cache[token] = &cacheEntry{
		userInfo:  userInfo,
		expiresAt: cacheExpiry,
	}
}

// removeFromCache 从缓存中删除Token
func (m *PostgRESTAuthMiddleware) removeFromCache(token string) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	delete(m.cache, token)
}

// ClearExpiredCache 清理过期缓存（可以定期调用）
func (m *PostgRESTAuthMiddleware) ClearExpiredCache() int {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	now := time.Now()
	clearedCount := 0

	for token, entry := range m.cache {
		if now.After(entry.expiresAt) {
			delete(m.cache, token)
			clearedCount++
		}
	}

	return clearedCount
}

// GetCacheStats 获取缓存统计信息
func (m *PostgRESTAuthMiddleware) GetCacheStats() map[string]interface{} {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	stats := map[string]interface{}{
		"total_cached": len(m.cache),
		"cache_ttl":    m.cacheTTL.String(),
	}

	// 统计过期和有效的缓存项
	now := time.Now()
	validCount := 0
	expiredCount := 0

	for _, entry := range m.cache {
		if now.After(entry.expiresAt) {
			expiredCount++
		} else {
			validCount++
		}
	}

	stats["valid_cached"] = validCount
	stats["expired_cached"] = expiredCount

	return stats
}

// respondUnauthorized 返回401未授权响应
func (m *PostgRESTAuthMiddleware) respondUnauthorized(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	render.JSON(w, r, map[string]interface{}{
		"status":  http.StatusUnauthorized,
		"message": message,
		"error":   "Unauthorized",
	})
}

// GetUserInfoFromContext 从上下文中获取用户信息
func GetUserInfoFromContext(ctx context.Context) (*UserInfo, bool) {
	userInfo, ok := ctx.Value(UserInfoKey).(*UserInfo)
	return userInfo, ok
}

// GetTokenFromContext 从上下文中获取Token
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(TokenKey).(string)
	return token, ok
}

// RequirePermission 创建一个需要特定权限的中间件
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userInfo, ok := GetUserInfoFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				render.JSON(w, r, map[string]interface{}{
					"status":  http.StatusUnauthorized,
					"message": "未找到用户信息",
					"error":   "Unauthorized",
				})
				return
			}

			// 检查用户是否具有所需权限
			hasPermission := false
			for _, perm := range userInfo.Permissions {
				if perm == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, map[string]interface{}{
					"status":  http.StatusForbidden,
					"message": fmt.Sprintf("缺少所需权限: %s", permission),
					"error":   "Forbidden",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole 创建一个需要特定角色的中间件
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userInfo, ok := GetUserInfoFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				render.JSON(w, r, map[string]interface{}{
					"status":  http.StatusUnauthorized,
					"message": "未找到用户信息",
					"error":   "Unauthorized",
				})
				return
			}

			// 检查用户是否具有所需角色
			hasRole := false
			for _, r := range userInfo.Roles {
				if r == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				render.JSON(w, r, map[string]interface{}{
					"status":  http.StatusForbidden,
					"message": fmt.Sprintf("缺少所需角色: %s", role),
					"error":   "Forbidden",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
