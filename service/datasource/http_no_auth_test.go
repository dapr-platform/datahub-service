/*
 * @module service/basic_library/datasource/http_no_auth_test
 * @description HTTP无认证数据源单元测试
 * @architecture 单元测试 - 测试HTTP无认证数据源的各种功能
 * @documentReference http_no_auth.go
 * @stateFlow 创建测试环境 -> 测试初始化 -> 测试启动 -> 测试执行 -> 测试停止
 * @rules 测试覆盖所有HTTP无认证场景，包括错误处理和脚本执行
 * @dependencies testing, context, time, net/http/httptest
 * @refs test_utils.go
 */

package datasource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"datahub-service/service/meta"
	"datahub-service/service/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPNoAuthDataSource_Basic 测试HTTP无认证数据源基本功能
func TestHTTPNoAuthDataSource_Basic(t *testing.T) {
	// 创建数据源实例
	ds := NewHTTPNoAuthDataSource()
	httpDs, ok := ds.(*HTTPNoAuthDataSource)
	require.True(t, ok, "数据源类型转换失败")

	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "success",
			"method":  r.Method,
		})
	}))
	defer server.Close()

	// 测试初始化
	ctx := context.Background()
	config := &models.DataSource{
		ID:       "test-http-no-auth",
		Name:     "测试HTTP无认证数据源",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeApiHTTP,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl: server.URL,
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
	assert.Equal(t, server.URL, httpDs.baseURL)

	// 测试启动
	err = ds.Start(ctx)
	assert.NoError(t, err, "启动应该成功")
	assert.True(t, httpDs.IsStarted(), "数据源应该正在运行")

	// 测试停止
	err = ds.Stop(ctx)
	assert.NoError(t, err, "停止应该成功")
	assert.False(t, httpDs.IsStarted(), "数据源应该已停止")
}

// TestHTTPNoAuthDataSource_Execute 测试HTTP请求执行
func TestHTTPNoAuthDataSource_Execute(t *testing.T) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"query":        r.URL.Query(),
			"user_agent":   r.Header.Get("User-Agent"),
			"content_type": r.Header.Get("Content-Type"),
		}

		// 如果是POST或PUT请求，读取请求体
		if r.Method == "POST" || r.Method == "PUT" {
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				response["body"] = body
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建数据源配置
	config := &models.DataSource{
		ID:       "test-http-execute",
		Name:     "测试HTTP执行",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeApiHTTP,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl: server.URL,
		},
		ScriptEnabled: false,
	}

	// 创建数据源实例
	ds := NewHTTPNoAuthDataSource()
	ctx := context.Background()

	err := ds.Init(ctx, config)
	require.NoError(t, err)
	err = ds.Start(ctx)
	require.NoError(t, err)
	defer ds.Stop(ctx)

	tests := []struct {
		name      string
		request   *ExecuteRequest
		wantError bool
	}{
		{
			name: "GET请求",
			request: &ExecuteRequest{
				Operation: "get",
				Query:     "param1=value1&param2=value2",
			},
			wantError: false,
		},
		{
			name: "POST请求",
			request: &ExecuteRequest{
				Operation: "post",
				Data: map[string]interface{}{
					"name": "test",
					"type": "http",
				},
			},
			wantError: false,
		},
		{
			name: "PUT请求",
			request: &ExecuteRequest{
				Operation: "put",
				Data: map[string]interface{}{
					"id":   123,
					"name": "updated",
				},
			},
			wantError: false,
		},
		{
			name: "DELETE请求",
			request: &ExecuteRequest{
				Operation: "delete",
				Query:     "id=123",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := ds.Execute(ctx, tt.request)

			if tt.wantError {
				assert.Error(t, err, "应该返回错误")
			} else {
				assert.NoError(t, err, "不应该返回错误")
				assert.True(t, response.Success, "请求应该成功")
				assert.NotNil(t, response.Data, "应该有响应数据")

				// 验证响应数据
				if dataMap, ok := response.Data.(map[string]interface{}); ok {
					assert.Equal(t, "DataHub-Service/1.0", dataMap["user_agent"], "应该设置正确的User-Agent")

					expectedMethod := strings.ToUpper(tt.request.Operation)
					if expectedMethod == "QUERY" {
						expectedMethod = "GET"
					} else if expectedMethod == "INSERT" {
						expectedMethod = "POST"
					} else if expectedMethod == "UPDATE" {
						expectedMethod = "PUT"
					}
					assert.Equal(t, expectedMethod, dataMap["method"], "HTTP方法应该匹配")

					// 验证Content-Type
					if tt.request.Operation == "post" || tt.request.Operation == "put" {
						assert.Equal(t, "application/json", dataMap["content_type"], "POST/PUT请求应该设置JSON Content-Type")
					}
				}

				// 验证元数据
				assert.NotNil(t, response.Metadata, "应该有元数据")
				assert.Equal(t, 200, response.Metadata["status_code"], "状态码应该是200")
			}
		})
	}
}

// TestHTTPNoAuthDataSource_HealthCheck 测试健康检查
func TestHTTPNoAuthDataSource_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedStatus string
	}{
		{
			name: "健康的服务器",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedStatus: "online",
		},
		{
			name: "服务器错误",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedStatus: "error",
		},
		{
			name: "客户端错误（仍认为连接正常）",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedStatus: "online",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			config := &models.DataSource{
				ID:       "test-health-check",
				Name:     "测试健康检查",
				Category: meta.DataSourceCategoryAPI,
				Type:     meta.DataSourceTypeApiHTTP,
				ConnectionConfig: models.JSONB{
					meta.DataSourceFieldBaseUrl: server.URL,
				},
			}

			ds := NewHTTPNoAuthDataSource()
			ctx := context.Background()

			err := ds.Init(ctx, config)
			require.NoError(t, err)
			err = ds.Start(ctx)
			require.NoError(t, err)
			defer ds.Stop(ctx)

			status, err := ds.HealthCheck(ctx)
			assert.NoError(t, err, "健康检查不应该返回错误")
			assert.Equal(t, tt.expectedStatus, status.Status, "健康状态应该匹配")
			assert.True(t, status.ResponseTime > 0, "响应时间应该大于0")
		})
	}
}

// TestHTTPNoAuthDataSource_ErrorHandling 测试错误处理
func TestHTTPNoAuthDataSource_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("测试缺少基础URL", func(t *testing.T) {
		config := &models.DataSource{
			ID:               "test-no-url",
			Name:             "测试缺少URL",
			Category:         meta.DataSourceCategoryAPI,
			Type:             meta.DataSourceTypeHTTP,
			ConnectionConfig: models.JSONB{
				// 缺少baseURL
			},
		}

		ds := NewHTTPNoAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "缺少基础URL应该导致初始化失败")
		assert.Contains(t, err.Error(), "基础URL配置错误", "错误信息应该包含URL错误")
	})

	t.Run("测试空连接配置", func(t *testing.T) {
		config := &models.DataSource{
			ID:               "test-no-config",
			Name:             "测试空配置",
			Category:         meta.DataSourceCategoryAPI,
			Type:             meta.DataSourceTypeHTTP,
			ConnectionConfig: nil,
		}

		ds := NewHTTPNoAuthDataSource()
		err := ds.Init(ctx, config)
		assert.Error(t, err, "空连接配置应该导致初始化失败")
		assert.Contains(t, err.Error(), "连接配置不能为空", "错误信息应该包含配置为空")
	})

	t.Run("测试未初始化执行", func(t *testing.T) {
		ds := NewHTTPNoAuthDataSource()
		request := &ExecuteRequest{
			Operation: "get",
		}

		response, err := ds.Execute(ctx, request)
		assert.Error(t, err, "未初始化应该导致执行失败")
		assert.False(t, response.Success, "响应应该标记为失败")
		assert.Contains(t, response.Error, "数据源未初始化", "错误信息应该包含未初始化")
	})

	t.Run("测试连接失败", func(t *testing.T) {
		config := &models.DataSource{
			ID:       "test-connection-fail",
			Name:     "测试连接失败",
			Category: meta.DataSourceCategoryAPI,
			Type:     meta.DataSourceTypeApiHTTP,
			ConnectionConfig: models.JSONB{
				meta.DataSourceFieldBaseUrl: "http://localhost:99999", // 不存在的端口
			},
		}

		ds := NewHTTPNoAuthDataSource()
		err := ds.Init(ctx, config)
		assert.NoError(t, err, "初始化应该成功")

		err = ds.Start(ctx)
		assert.Error(t, err, "启动应该失败，因为无法连接")
		assert.Contains(t, err.Error(), "测试连接失败", "错误信息应该包含连接失败")
	})
}

// TestHTTPNoAuthDataSource_WithScript 测试脚本执行
func TestHTTPNoAuthDataSource_WithScript(t *testing.T) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "server response",
		})
	}))
	defer server.Close()

	// 简单的测试脚本
	testScript := `
// 根据操作类型返回不同结果
operationStr, ok := operation.(string)
if !ok {
    return nil, fmt.Errorf("operation参数类型错误")
}

switch operationStr {
case "init":
    return map[string]interface{}{
        "success": true,
        "message": "脚本初始化成功",
    }, nil
case "start":
    return map[string]interface{}{
        "success": true,
        "message": "脚本启动成功",
    }, nil
case "execute":
    return map[string]interface{}{
        "success": true,
        "message": "脚本执行成功",
        "data": "test data",
    }, nil
case "stop":
    return map[string]interface{}{
        "success": true,
        "message": "脚本停止成功",
    }, nil
default:
    return nil, fmt.Errorf("不支持的操作类型: %s", operationStr)
}
`

	config := &models.DataSource{
		ID:       "test-http-script",
		Name:     "测试HTTP脚本",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeApiHTTP,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl: server.URL,
		},
		ScriptEnabled: true,
		Script:        testScript,
	}

	ds := NewHTTPNoAuthDataSource()
	ctx := context.Background()

	t.Run("测试脚本初始化", func(t *testing.T) {
		err := ds.Init(ctx, config)
		assert.NoError(t, err, "脚本初始化应该成功")
	})

	t.Run("测试脚本启动", func(t *testing.T) {
		err := ds.Start(ctx)
		assert.NoError(t, err, "脚本启动应该成功")
	})

	t.Run("测试脚本执行", func(t *testing.T) {
		request := &ExecuteRequest{
			Operation: "get",
		}

		response, err := ds.Execute(ctx, request)
		assert.NoError(t, err, "脚本执行应该成功")
		assert.True(t, response.Success, "响应应该成功")

		// 验证脚本返回的数据
		if dataMap, ok := response.Data.(map[string]interface{}); ok {
			assert.Equal(t, true, dataMap["success"], "脚本应该返回成功")
			assert.Equal(t, "脚本执行成功", dataMap["message"], "应该有正确的消息")
		}
	})

	t.Run("测试脚本停止", func(t *testing.T) {
		err := ds.Stop(ctx)
		assert.NoError(t, err, "脚本停止应该成功")
	})
}

// TestHTTPNoAuthDataSource_Timeout 测试超时设置
func TestHTTPNoAuthDataSource_Timeout(t *testing.T) {
	// 创建一个慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 延迟2秒
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.DataSource{
		ID:       "test-timeout",
		Name:     "测试超时",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeApiHTTP,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl: server.URL,
		},
		ParamsConfig: models.JSONB{
			meta.DataSourceFieldTimeout: 1.0, // 1秒超时
		},
	}

	ds := NewHTTPNoAuthDataSource()
	ctx := context.Background()

	err := ds.Init(ctx, config)
	require.NoError(t, err)

	// 验证超时设置
	httpDs := ds.(*HTTPNoAuthDataSource)
	assert.Equal(t, 1*time.Second, httpDs.client.Timeout, "超时时间应该被正确设置")
}

// BenchmarkHTTPNoAuthDataSource_Execute 性能测试
func BenchmarkHTTPNoAuthDataSource_Execute(b *testing.B) {
	// 创建模拟HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "benchmark response",
		})
	}))
	defer server.Close()

	config := &models.DataSource{
		ID:       "benchmark-http",
		Name:     "性能测试HTTP",
		Category: meta.DataSourceCategoryAPI,
		Type:     meta.DataSourceTypeApiHTTP,
		ConnectionConfig: models.JSONB{
			meta.DataSourceFieldBaseUrl: server.URL,
		},
	}

	// 初始化数据源
	ds := NewHTTPNoAuthDataSource()
	ctx := context.Background()
	err := ds.Init(ctx, config)
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
		Operation: "get",
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
