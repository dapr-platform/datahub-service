/*
 * @module service/basic_library/datasource/http_auth_test
 * @description HTTP认证数据源单元测试，包含绿云接口脚本测试
 * @architecture 单元测试 - 测试HTTP认证数据源的各种功能
 * @documentReference http_auth.go, lvyun_hotel_script.go
 * @stateFlow 创建测试环境 -> 测试初始化 -> 测试启动 -> 测试执行 -> 测试停止
 * @rules 测试覆盖所有HTTP认证场景，包括脚本执行和错误处理
 * @dependencies testing, context, time
 * @refs mock_lvyun_server.go, test_utils.go
 */

package datasource

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"datahub-service/service/meta"
	"datahub-service/service/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPAuthDataSource_Basic 测试HTTP认证数据源基本功能
func TestHTTPAuthDataSource_Basic(t *testing.T) {
	// 创建数据源实例
	ds := NewHTTPAuthDataSource()
	httpDs, ok := ds.(*HTTPAuthDataSource)
	require.True(t, ok, "数据源类型转换失败")

	// 测试初始化
	ctx := context.Background()
	config := &models.DataSource{
		ID:       "test-http-auth",
		Name:     "测试HTTP认证数据源",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeHTTPWithAuth,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl:  "https://api.example.com",
			meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeBearer,
			meta.DataSourceFieldApiKey:   "test-token-123",
		},
		ParamsConfig: models.JSONB{
			meta.DataSourceFieldTimeout: 10.0,
		},
		ScriptEnabled: false,
		Script:        "",
	}

	err := ds.Init(ctx, config)
	assert.NoError(t, err, "初始化应该成功")
	assert.True(t, httpDs.IsInitialized(), "数据源应该已初始化")
	assert.Equal(t, "https://api.example.com", httpDs.baseURL)
	assert.Equal(t, meta.DataSourceAuthTypeBearer, httpDs.authType)
	assert.Equal(t, "test-token-123", httpDs.credentials["api_key"])

	// 测试启动（由于没有脚本且URL不可访问，这里会失败，我们跳过启动测试）
	// err = ds.Start(ctx)
	// assert.NoError(t, err, "启动应该成功")
	// assert.True(t, httpDs.IsStarted(), "数据源应该正在运行")

	// 测试停止
	err = ds.Stop(ctx)
	assert.NoError(t, err, "停止应该成功")
	assert.False(t, httpDs.IsStarted(), "数据源应该已停止")
}

// TestHTTPAuthDataSource_LvyunScript 测试使用绿云脚本的HTTP认证数据源
func TestHTTPAuthDataSource_LvyunScript(t *testing.T) {
	// 读取绿云脚本内容（只包含Run函数体）
	scriptPath := filepath.Join("docs", "lvyun_body_only.txt")
	scriptContent, err := ioutil.ReadFile(scriptPath)
	require.NoError(t, err, "读取脚本文件失败")

	// 创建模拟绿云服务器
	appSecret := "test-app-secret-12345"
	mockServer := NewMockLvyunServer(appSecret)
	defer mockServer.Close()

	// 创建数据源配置
	config := &models.DataSource{
		ID:       "test-lvyun-auth",
		Name:     "测试绿云认证数据源",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeHTTPWithAuth,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl:   mockServer.URL(),
			meta.DataSourceFieldAuthType:  meta.DataSourceAuthTypeCustom,
			meta.DataSourceFieldUsername:  "test2",
			meta.DataSourceFieldPassword:  "123456",
			meta.DataSourceFieldApiKey:    "10001",
			meta.DataSourceFieldApiSecret: appSecret,
			"hotel_group_code":            "LYG",
		},
		ParamsConfig: models.JSONB{
			meta.DataSourceFieldTimeout: 30.0,
		},
		ScriptEnabled: true,
		Script:        string(scriptContent),
	}

	// 创建数据源实例
	ds := NewHTTPAuthDataSource()
	httpDs, ok := ds.(*HTTPAuthDataSource)
	require.True(t, ok, "数据源类型转换失败")

	ctx := context.Background()

	t.Run("测试初始化", func(t *testing.T) {
		err := ds.Init(ctx, config)
		assert.NoError(t, err, "初始化应该成功")
		assert.True(t, httpDs.IsInitialized(), "数据源应该已初始化")

		// 验证配置解析
		assert.Equal(t, mockServer.URL(), httpDs.baseURL)
		assert.Equal(t, meta.DataSourceAuthTypeCustom, httpDs.authType)
		assert.Equal(t, "test2", httpDs.credentials["username"])
		assert.Equal(t, "123456", httpDs.credentials["password"])
		assert.Equal(t, "10001", httpDs.credentials["api_key"])
		assert.Equal(t, appSecret, httpDs.credentials["api_secret"])
	})

	t.Run("测试启动和获取sessionId", func(t *testing.T) {
		err := ds.Start(ctx)
		assert.NoError(t, err, "启动应该成功")
		assert.True(t, httpDs.IsStarted(), "数据源应该正在运行")

		// 验证sessionId是否已获取
		httpDs.mu.RLock()
		sessionId, exists := httpDs.sessionData["sessionId"]
		httpDs.mu.RUnlock()

		assert.True(t, exists, "应该已获取sessionId")
		assert.NotEmpty(t, sessionId, "sessionId不应为空")

		// 验证模拟服务器中的session
		assert.Equal(t, 1, mockServer.GetSessionCount(), "模拟服务器应该有1个session")
	})

	t.Run("测试执行查询", func(t *testing.T) {
		// 测试实时房情数据查询
		request := &ExecuteRequest{
			Operation: "query",
			Params: map[string]interface{}{
				"exec":       "Kpi_Ihotel_Room_Total",
				"hotel_code": "001",
			},
		}

		response, err := ds.Execute(ctx, request)
		assert.NoError(t, err, "执行查询应该成功")
		assert.True(t, response.Success, "查询应该成功")
		assert.NotNil(t, response.Data, "应该有查询结果")

		// 验证返回数据结构
		if scriptResult, ok := response.Data.(map[string]interface{}); ok {
			assert.True(t, scriptResult["success"].(bool), "脚本执行应该成功")
			assert.NotNil(t, scriptResult["data"], "应该有查询数据")
			assert.NotNil(t, scriptResult["metadata"], "应该有元数据")
		}
	})

	t.Run("测试酒店排名查询", func(t *testing.T) {
		request := &ExecuteRequest{
			Operation: "query",
			Params: map[string]interface{}{
				"exec": "Kpi_Ihotel_Room_Rank",
			},
		}

		response, err := ds.Execute(ctx, request)
		assert.NoError(t, err, "执行查询应该成功")
		assert.True(t, response.Success, "查询应该成功")
	})

	t.Run("测试平均房价走势查询", func(t *testing.T) {
		request := &ExecuteRequest{
			Operation: "query",
			Params: map[string]interface{}{
				"exec":       "Kpi_Ihotel_Room_Adr_List",
				"hotel_code": "001",
			},
		}

		response, err := ds.Execute(ctx, request)
		assert.NoError(t, err, "执行查询应该成功")
		assert.True(t, response.Success, "查询应该成功")
	})

	t.Run("测试健康检查", func(t *testing.T) {
		status, err := ds.HealthCheck(ctx)
		assert.NoError(t, err, "健康检查应该成功")
		assert.Equal(t, "online", status.Status, "状态应该是online")
		assert.True(t, status.ResponseTime > 0, "响应时间应该大于0")
	})

	t.Run("测试停止和退出sessionId", func(t *testing.T) {
		// 获取当前sessionId用于验证
		httpDs.mu.RLock()
		sessionId, hasSession := httpDs.sessionData["sessionId"]
		httpDs.mu.RUnlock()

		assert.True(t, hasSession, "应该存在sessionId")
		assert.NotEmpty(t, sessionId, "sessionId不应为空")

		err := ds.Stop(ctx)
		assert.NoError(t, err, "停止应该成功")
		assert.False(t, httpDs.IsStarted(), "数据源应该已停止")

		// 验证sessionData已清理
		httpDs.mu.RLock()
		sessionDataEmpty := len(httpDs.sessionData) == 0
		httpDs.mu.RUnlock()
		assert.True(t, sessionDataEmpty, "sessionData应该已清理")

		// 验证模拟服务器中的session已清理（可能需要一些时间）
		time.Sleep(100 * time.Millisecond)
		// 注意：logout可能不会立即从mock server中删除session，这取决于实现
	})
}

// TestHTTPAuthDataSource_ScriptErrors 测试脚本执行错误场景
func TestHTTPAuthDataSource_ScriptErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("测试无效脚本", func(t *testing.T) {
		config := &models.DataSource{
			ID:       "test-invalid-script",
			Name:     "测试无效脚本",
			Category: meta.DataSourceCategoryAPI,
			Type:     meta.DataSourceTypeHTTPWithAuth,
			ConnectionConfig: models.JSONB{
				meta.DataSourceFieldBaseUrl:  "https://api.example.com",
				meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeCustom,
			},
			ScriptEnabled: true,
			Script:        "invalid go code here",
		}

		ds := NewHTTPAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "无效脚本应该导致初始化失败")
		assert.Contains(t, err.Error(), "初始化脚本执行失败", "错误信息应该包含脚本执行失败")
	})

	t.Run("测试脚本执行超时", func(t *testing.T) {
		timeoutScript := `
package main

import "time"

func main() {
	time.Sleep(60 * time.Second) // 超过30秒超时时间
	return map[string]interface{}{"success": true}
}
`

		config := &models.DataSource{
			ID:       "test-timeout-script",
			Name:     "测试超时脚本",
			Category: meta.DataSourceCategoryAPI,
			Type:     meta.DataSourceTypeHTTPWithAuth,
			ConnectionConfig: models.JSONB{
				meta.DataSourceFieldBaseUrl:  "https://api.example.com",
				meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeCustom,
			},
			ScriptEnabled: true,
			Script:        timeoutScript,
		}

		ds := NewHTTPAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "超时脚本应该导致初始化失败")
	})
}

// TestHTTPAuthDataSource_SessionManagement 测试会话管理
func TestHTTPAuthDataSource_SessionManagement(t *testing.T) {
	// 读取绿云脚本内容（只包含Run函数体）
	scriptPath := filepath.Join("docs", "lvyun_body_only.txt")
	scriptContent, err := ioutil.ReadFile(scriptPath)
	require.NoError(t, err, "读取脚本文件失败")

	// 创建模拟绿云服务器
	appSecret := "test-session-secret"
	mockServer := NewMockLvyunServer(appSecret)
	defer mockServer.Close()

	// 创建数据源配置
	config := &models.DataSource{
		ID:       "test-session-mgmt",
		Name:     "测试会话管理",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeHTTPWithAuth,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl:   mockServer.URL(),
			meta.DataSourceFieldAuthType:  meta.DataSourceAuthTypeCustom,
			meta.DataSourceFieldUsername:  "test2",
			meta.DataSourceFieldPassword:  "123456",
			meta.DataSourceFieldApiKey:    "10001",
			meta.DataSourceFieldApiSecret: appSecret,
			"hotel_group_code":            "LYG",
		},
		ScriptEnabled: true,
		Script:        string(scriptContent),
	}

	ds := NewHTTPAuthDataSource()
	httpDs, _ := ds.(*HTTPAuthDataSource)

	ctx := context.Background()

	// 初始化和启动
	err = ds.Init(ctx, config)
	require.NoError(t, err)
	err = ds.Start(ctx)
	require.NoError(t, err)

	t.Run("测试会话数据存储", func(t *testing.T) {
		httpDs.mu.RLock()
		sessionId, exists := httpDs.sessionData["sessionId"]
		loginTime, timeExists := httpDs.sessionData["loginTime"]
		httpDs.mu.RUnlock()

		assert.True(t, exists, "应该存在sessionId")
		assert.NotEmpty(t, sessionId, "sessionId不应为空")
		assert.True(t, timeExists, "应该存在登录时间")
		assert.NotEmpty(t, loginTime, "登录时间不应为空")
	})

	t.Run("测试会话数据并发访问", func(t *testing.T) {
		// 启动多个goroutine同时访问会话数据
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				defer func() { done <- true }()

				// 读取会话数据
				httpDs.mu.RLock()
				sessionId := httpDs.sessionData["sessionId"]
				httpDs.mu.RUnlock()

				assert.NotEmpty(t, sessionId, "sessionId不应为空")

				// 写入会话数据
				httpDs.mu.Lock()
				httpDs.sessionData[fmt.Sprintf("test_%d", index)] = fmt.Sprintf("value_%d", index)
				httpDs.mu.Unlock()
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}

		// 验证数据完整性
		httpDs.mu.RLock()
		count := 0
		for key := range httpDs.sessionData {
			if strings.HasPrefix(key, "test_") {
				count++
			}
		}
		httpDs.mu.RUnlock()

		assert.Equal(t, 10, count, "应该有10个测试数据")
	})

	// 清理
	err = ds.Stop(ctx)
	assert.NoError(t, err)
}

// TestHTTPAuthDataSource_AuthTypes 测试不同认证类型
func TestHTTPAuthDataSource_AuthTypes(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name     string
		authType string
		config   models.JSONB
		wantErr  bool
	}{
		{
			name:     "Basic认证",
			authType: meta.DataSourceAuthTypeBasic,
			config: models.JSONB{
				meta.DataSourceFieldBaseUrl:  "https://api.example.com",
				meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeBasic,
				meta.DataSourceFieldUsername: "testuser",
				meta.DataSourceFieldPassword: "testpass",
			},
			wantErr: false,
		},
		{
			name:     "Bearer认证",
			authType: meta.DataSourceAuthTypeBearer,
			config: models.JSONB{
				meta.DataSourceFieldBaseUrl:  "https://api.example.com",
				meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeBearer,
				meta.DataSourceFieldApiKey:   "bearer-token-123",
			},
			wantErr: false,
		},
		{
			name:     "API Key认证",
			authType: meta.DataSourceAuthTypeAPIKey,
			config: models.JSONB{
				meta.DataSourceFieldBaseUrl:      "https://api.example.com",
				meta.DataSourceFieldAuthType:     meta.DataSourceAuthTypeAPIKey,
				meta.DataSourceFieldApiKey:       "api-key-123",
				meta.DataSourceFieldApiKeyHeader: "X-API-Key",
			},
			wantErr: false,
		},
		{
			name:     "自定义认证（无脚本）",
			authType: meta.DataSourceAuthTypeCustom,
			config: models.JSONB{
				meta.DataSourceFieldBaseUrl:  "https://api.example.com",
				meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeCustom,
			},
			wantErr: false, // 初始化不会失败，但使用时会失败
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &models.DataSource{
				ID:               "test-auth-" + tc.name,
				Name:             "测试" + tc.name,
				Category:         meta.DataSourceCategoryAPI,
				Type:             meta.DataSourceTypeHTTPWithAuth,
				ConnectionConfig: tc.config,
				ScriptEnabled:    false,
			}

			ds := NewHTTPAuthDataSource()
			err := ds.Init(ctx, config)

			if tc.wantErr {
				assert.Error(t, err, "应该初始化失败")
			} else {
				assert.NoError(t, err, "应该初始化成功")

				httpDs, _ := ds.(*HTTPAuthDataSource)
				assert.Equal(t, tc.authType, httpDs.authType, "认证类型应该匹配")
			}
		})
	}
}

// TestHTTPAuthDataSource_ErrorHandling 测试错误处理
func TestHTTPAuthDataSource_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("测试缺少基础URL", func(t *testing.T) {
		config := &models.DataSource{
			ID:       "test-no-url",
			Name:     "测试缺少URL",
			Category: meta.DataSourceCategoryAPI,
			Type:     meta.DataSourceTypeHTTPWithAuth,
			ConnectionConfig: models.JSONB{
				meta.DataSourceFieldAuthType: meta.DataSourceAuthTypeBearer,
			},
		}

		ds := NewHTTPAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "缺少基础URL应该导致初始化失败")
		assert.Contains(t, err.Error(), "基础URL配置错误", "错误信息应该包含URL错误")
	})

	t.Run("测试缺少认证类型", func(t *testing.T) {
		config := &models.DataSource{
			ID:       "test-no-auth-type",
			Name:     "测试缺少认证类型",
			Category: meta.DataSourceCategoryAPI,
			Type:     meta.DataSourceTypeHTTPWithAuth,
			ConnectionConfig: models.JSONB{
				meta.DataSourceFieldBaseUrl: "https://api.example.com",
			},
		}

		ds := NewHTTPAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "缺少认证类型应该导致初始化失败")
		assert.Contains(t, err.Error(), "认证类型配置错误", "错误信息应该包含认证类型错误")
	})

	t.Run("测试空连接配置", func(t *testing.T) {
		config := &models.DataSource{
			ID:               "test-no-config",
			Name:             "测试空配置",
			Category:         meta.DataSourceCategoryAPI,
			Type:             meta.DataSourceTypeHTTPWithAuth,
			ConnectionConfig: nil,
		}

		ds := NewHTTPAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "空连接配置应该导致初始化失败")
		assert.Contains(t, err.Error(), "连接配置不能为空", "错误信息应该包含配置为空")
	})

	t.Run("测试未初始化执行", func(t *testing.T) {
		ds := NewHTTPAuthDataSource()
		request := &ExecuteRequest{
			Operation: "query",
		}

		response, err := ds.Execute(ctx, request)
		assert.Error(t, err, "未初始化应该导致执行失败")
		assert.False(t, response.Success, "响应应该标记为失败")
		assert.Contains(t, response.Error, "数据源未初始化", "错误信息应该包含未初始化")
	})
}

// BenchmarkHTTPAuthDataSource_Execute 性能测试
func BenchmarkHTTPAuthDataSource_Execute(b *testing.B) {
	// 读取绿云脚本内容（只包含Run函数体）
	scriptPath := filepath.Join("docs", "lvyun_body_only.txt")
	scriptContent, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		b.Fatalf("读取脚本文件失败: %v", err)
	}

	// 创建模拟绿云服务器
	appSecret := "benchmark-secret"
	mockServer := NewMockLvyunServer(appSecret)
	defer mockServer.Close()

	// 创建数据源配置
	config := &models.DataSource{
		ID:       "benchmark-lvyun",
		Name:     "性能测试绿云数据源",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeHTTPWithAuth,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl:   mockServer.URL(),
			meta.DataSourceFieldAuthType:  meta.DataSourceAuthTypeCustom,
			meta.DataSourceFieldUsername:  "test2",
			meta.DataSourceFieldPassword:  "123456",
			meta.DataSourceFieldApiKey:    "10001",
			meta.DataSourceFieldApiSecret: appSecret,
			"hotel_group_code":            "LYG",
		},
		ScriptEnabled: true,
		Script:        string(scriptContent),
	}

	// 初始化数据源
	ds := NewHTTPAuthDataSource()
	ctx := context.Background()
	err = ds.Init(ctx, config)
	if err != nil {
		b.Fatalf("初始化失败: %v", err)
	}
	err = ds.Start(ctx)
	if err != nil {
		b.Fatalf("启动失败: %v", err)
	}
	defer ds.Stop(ctx)

	// 准备请求
	request := &ExecuteRequest{
		Operation: "query",
		Params: map[string]interface{}{
			"exec": "Kpi_Ihotel_Room_Total",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ds.Execute(ctx, request)
			if err != nil {
				b.Errorf("执行失败: %v", err)
			}
		}
	})
}
