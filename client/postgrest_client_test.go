/*
 * @module client/postgrest_client_test
 * @description PostgREST HTTP客户端测试
 * @architecture 测试架构 - Token管理和HTTP请求测试
 * @documentReference client/postgrest_client.go
 * @stateFlow 创建客户端 -> 测试连接 -> 测试Token -> 测试请求 -> 清理
 * @rules 使用环境变量配置，测试Token自动刷新
 * @dependencies testing, time, os
 * @refs ai_docs/postgrest_rbac_guide.md
 */

package client

import (
	"os"
	"testing"
	"time"
)

// TestPostgRESTClient_BasicFunctionality 测试PostgREST客户端基本功能
func TestPostgRESTClient_BasicFunctionality(t *testing.T) {
	// 检查环境变量
	postgrestURL := os.Getenv("POSTGREST_URL")
	if postgrestURL == "" {
		postgrestURL = "http://localhost:3000"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "things2024"
	}

	t.Logf("测试配置: URL=%s, User=%s", postgrestURL, dbUser)

	// 创建客户端配置
	config := &PostgRESTConfig{
		BaseURL:         postgrestURL,
		Username:        dbUser,
		Password:        dbPassword,
		Timeout:         10 * time.Second,
		RefreshInterval: 1 * time.Minute, // 测试时使用较短的刷新间隔
		MaxRetries:      3,
	}

	// 创建客户端
	client := NewPostgRESTClient(config)
	defer client.Close()

	t.Logf("✅ PostgREST客户端创建成功")

	// 测试连接和Token获取
	err := client.Connect()
	if err != nil {
		t.Logf("⚠️ PostgREST连接失败（可能是服务未启动）: %v", err)
		t.Skip("跳过PostgREST测试，服务可能未启动")
		return
	}

	t.Logf("✅ PostgREST连接成功")

	// 验证Token
	if !client.IsTokenValid() {
		t.Errorf("❌ Token无效")
		return
	}

	accessToken := client.GetAccessToken()
	if accessToken == "" {
		t.Errorf("❌ 访问Token为空")
		return
	}

	t.Logf("✅ Token获取成功，长度: %d", len(accessToken))

	// 获取Token过期时间
	expiry := client.GetTokenExpiry()
	if expiry.Before(time.Now()) {
		t.Errorf("❌ Token已过期: %v", expiry)
		return
	}

	t.Logf("✅ Token有效期至: %v", expiry.Format("2006-01-02 15:04:05"))

	// 测试统计信息
	stats := client.GetStatistics()
	t.Logf("📊 客户端统计信息: %+v", stats)

	// 测试简单的HTTP请求（获取schemas）
	resp, err := client.MakeRequest("GET", "/", nil, nil)
	if err != nil {
		t.Logf("⚠️ HTTP请求失败: %v", err)
	} else {
		defer resp.Body.Close()
		t.Logf("✅ HTTP请求成功，状态码: %d", resp.StatusCode)
	}
}

// TestPostgRESTClient_TokenRefresh 测试Token刷新功能
func TestPostgRESTClient_TokenRefresh(t *testing.T) {
	// 检查环境变量
	postgrestURL := os.Getenv("POSTGREST_URL")
	if postgrestURL == "" {
		postgrestURL = "http://localhost:3000"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "things2024"
	}

	// 创建客户端配置（短刷新间隔用于测试）
	config := &PostgRESTConfig{
		BaseURL:         postgrestURL,
		Username:        dbUser,
		Password:        dbPassword,
		Timeout:         10 * time.Second,
		RefreshInterval: 5 * time.Second, // 5秒刷新一次
		MaxRetries:      3,
	}

	// 创建客户端
	client := NewPostgRESTClient(config)
	defer client.Close()

	// 连接并获取初始Token
	err := client.Connect()
	if err != nil {
		t.Logf("⚠️ PostgREST连接失败（可能是服务未启动）: %v", err)
		t.Skip("跳过PostgREST刷新测试，服务可能未启动")
		return
	}

	// 记录初始统计
	initialStats := client.GetStatistics()
	t.Logf("📊 初始统计: Token刷新次数=%v", initialStats["token_refreshed"])

	// 等待一段时间让自动刷新触发
	t.Logf("⏰ 等待10秒观察Token自动刷新...")
	time.Sleep(10 * time.Second)

	// 检查刷新后的统计
	finalStats := client.GetStatistics()
	t.Logf("📊 最终统计: Token刷新次数=%v", finalStats["token_refreshed"])

	// 验证Token仍然有效
	if !client.IsTokenValid() {
		t.Errorf("❌ Token在刷新后无效")
	} else {
		t.Logf("✅ Token刷新后仍然有效")
	}
}

// TestPostgRESTClient_ProxyRequest 测试代理请求功能
func TestPostgRESTClient_ProxyRequest(t *testing.T) {
	// 检查环境变量
	postgrestURL := os.Getenv("POSTGREST_URL")
	if postgrestURL == "" {
		postgrestURL = "http://localhost:3000"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "things2024"
	}

	// 创建客户端配置
	config := &PostgRESTConfig{
		BaseURL:         postgrestURL,
		Username:        dbUser,
		Password:        dbPassword,
		Timeout:         10 * time.Second,
		RefreshInterval: 55 * time.Minute,
		MaxRetries:      3,
	}

	// 创建客户端
	client := NewPostgRESTClient(config)
	defer client.Close()

	// 连接
	err := client.Connect()
	if err != nil {
		t.Logf("⚠️ PostgREST连接失败（可能是服务未启动）: %v", err)
		t.Skip("跳过PostgREST代理测求测试，服务可能未启动")
		return
	}

	// 测试代理请求 - 尝试访问一个可能存在的表
	testCases := []struct {
		method      string
		tableName   string
		queryParams string
		description string
	}{
		{"GET", "", "", "根路径请求"},
		{"HEAD", "", "", "根路径HEAD请求"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resp, err := client.ProxyRequest(tc.method, tc.tableName, tc.queryParams, nil, nil)
			if err != nil {
				t.Logf("⚠️ %s 请求失败: %v", tc.description, err)
			} else {
				defer resp.Body.Close()
				t.Logf("✅ %s 请求成功，状态码: %d", tc.description, resp.StatusCode)
			}
		})
	}

	// 获取最终统计信息
	stats := client.GetStatistics()
	t.Logf("📊 代理请求测试完成，统计信息: 总请求=%v, 成功=%v, 错误=%v",
		stats["request_count"], stats["success_count"], stats["error_count"])
}

// TestPostgRESTClient_Configuration 测试不同配置选项
func TestPostgRESTClient_Configuration(t *testing.T) {
	// 测试默认配置
	config1 := &PostgRESTConfig{
		BaseURL:  "http://localhost:3000",
		Username: "test",
		Password: "test",
	}

	client1 := NewPostgRESTClient(config1)
	defer client1.Close()

	stats1 := client1.GetStatistics()
	t.Logf("✅ 默认配置客户端创建成功: %+v", stats1)

	// 测试自定义配置
	config2 := &PostgRESTConfig{
		BaseURL:         "http://localhost:3001",
		Username:        "custom",
		Password:        "custom",
		Timeout:         5 * time.Second,
		RefreshInterval: 30 * time.Minute,
		MaxRetries:      5,
	}

	client2 := NewPostgRESTClient(config2)
	defer client2.Close()

	stats2 := client2.GetStatistics()
	t.Logf("✅ 自定义配置客户端创建成功: %+v", stats2)
}
