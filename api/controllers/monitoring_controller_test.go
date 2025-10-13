/*
 * @module api/controllers/monitoring_controller_test
 * @description 监控控制器单元测试（完整版）
 * @architecture 测试层
 * @documentReference .specify/memory/test_plan.md
 * @stateFlow 测试准备 -> 请求构建 -> 响应验证
 * @rules 确保监控API的正确性和完整性
 * @dependencies testing, net/http/httptest, stretchr/testify
 */

package controllers

import (
	"bytes"
	"context"
	"datahub-service/service/meta"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===================== 模板相关测试 =====================

// TestGetMetricTemplates 测试获取指标模板
func TestGetMetricTemplates(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/templates/metrics", nil)
	w := httptest.NewRecorder()

	controller.GetMetricTemplates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Status)
	assert.NotNil(t, response.Data)

	// 验证返回的数据结构
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "响应数据应该是map类型")
	assert.Greater(t, len(data), 0, "应该返回至少一个模板")

	// 验证关键模板存在
	assert.Contains(t, data, "cpu_usage_active")
	assert.Contains(t, data, "mem_used_percent")
	assert.Contains(t, data, "disk_used_percent")
}

// TestGetLogTemplates 测试获取日志模板
func TestGetLogTemplates(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/templates/logs", nil)
	w := httptest.NewRecorder()

	controller.GetLogTemplates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Status)
	assert.NotNil(t, response.Data)

	// 验证返回的数据结构
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Greater(t, len(data), 0)

	// 验证关键模板存在
	assert.Contains(t, data, "error_logs")
	assert.Contains(t, data, "warning_logs")
}

// TestGetMetricDescriptions 测试获取指标描述
func TestGetMetricDescriptions(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/metrics/descriptions", nil)
	w := httptest.NewRecorder()

	controller.GetMetricDescriptions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Status)
	assert.NotNil(t, response.Data)

	// 验证返回的数据结构
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Greater(t, len(data), 0)

	// 验证关键指标描述存在
	assert.Contains(t, data, "cpu_usage_active")
	assert.Contains(t, data, "mem_total")
}

// ===================== 配置相关测试 =====================

// TestGetMonitoringConfig 测试获取监控配置
func TestGetMonitoringConfig(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/config", nil)
	w := httptest.NewRecorder()

	controller.GetMonitoringConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Status)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "victoria_metrics")
	assert.Contains(t, data, "loki")
	assert.Contains(t, data, "description")
	assert.Contains(t, data, "version")
}

// ===================== 查询相关测试 =====================

// TestQueryMetrics_EmptyQuery 测试空查询
func TestQueryMetrics_EmptyQuery(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "metrics",
		Query:     "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/metrics", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryMetrics(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "查询语句不能为空")
}

// TestQueryMetrics_InvalidJSON 测试无效JSON
func TestQueryMetrics_InvalidJSON(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/metrics",
		bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryMetrics(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "解析请求失败")
}

// TestQueryMetrics_ValidQuery 测试有效查询（模拟）
func TestQueryMetrics_ValidQuery(t *testing.T) {
	// 注意：这个测试需要实际的 VictoriaMetrics 服务才能通过
	// 这里仅测试请求格式是否正确
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "metrics",
		Query:     "up",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/metrics", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryMetrics(w, req)

	// 如果服务不可用，会返回500错误
	// 如果服务可用，会返回200
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestQueryMetrics_RangeQuery 测试区间查询
func TestQueryMetrics_RangeQuery(t *testing.T) {
	controller := NewMonitoringController()

	now := int64(1640003600)
	reqBody := meta.MonitorQueryRequest{
		QueryType: "metrics",
		Query:     "cpu_usage_active",
		StartTime: now - 3600, // 1小时前
		EndTime:   now,
		Step:      15,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/metrics", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryMetrics(w, req)

	// 区间查询的请求格式应该是正确的
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestQueryLogs_EmptyQuery 测试空日志查询
func TestQueryLogs_EmptyQuery(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "logs",
		Query:     "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryLogs(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "查询语句不能为空")
}

// TestQueryLogs_WithLimit 测试带限制的日志查询
func TestQueryLogs_WithLimit(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "logs",
		Query:     "{job=\"test\"}",
		Limit:     100,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryLogs(w, req)

	// 请求格式应该正确
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestQueryLogs_RangeQuery 测试区间查询
func TestQueryLogs_RangeQuery(t *testing.T) {
	controller := NewMonitoringController()

	now := time.Now().Unix()
	reqBody := meta.MonitorQueryRequest{
		QueryType: "logs",
		Query:     "{app=\"flow-service\"}",
		StartTime: now - 3600, // 1小时前
		EndTime:   now,
		Limit:     1000,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryLogs(w, req)

	// 请求格式应该正确
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// ===================== 统一查询接口测试 =====================

// TestExecuteCustomQuery_InvalidType 测试无效查询类型
func TestExecuteCustomQuery_InvalidType(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "invalid",
		Query:     "test",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ExecuteCustomQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "不支持的查询类型")
}

// TestExecuteCustomQuery_EmptyType 测试空查询类型
func TestExecuteCustomQuery_EmptyType(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "",
		Query:     "test",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ExecuteCustomQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "查询类型不能为空")
}

// TestExecuteCustomQuery_MetricsType 测试指标类型查询
func TestExecuteCustomQuery_MetricsType(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "metrics",
		Query:     "up",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ExecuteCustomQuery(w, req)

	// 应该路由到 metrics 查询
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestExecuteCustomQuery_LogsType 测试日志类型查询
func TestExecuteCustomQuery_LogsType(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "logs",
		Query:     "{job=\"test\"}",
		Limit:     100,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ExecuteCustomQuery(w, req)

	// 应该路由到 logs 查询
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// ===================== 验证相关测试 =====================

// TestValidateQuery_EmptyQuery 测试空查询验证
func TestValidateQuery_EmptyQuery(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "metrics",
		Query:     "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ValidateQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
}

// TestValidateQuery_MetricsQuery 测试指标查询验证
func TestValidateQuery_MetricsQuery(t *testing.T) {
	controller := NewMonitoringController()

	reqBody := meta.MonitorQueryRequest{
		QueryType: "metrics",
		Query:     "up",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.ValidateQuery(w, req)

	// 验证应该返回200，无论查询是否有效（验证过程本身应该成功）
	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Status)

	// 检查返回的验证结果
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "valid")
	assert.Contains(t, data, "query")
	assert.Contains(t, data, "query_type")
}

// ===================== Loki 标签测试 =====================

// TestGetLokiLabels_EmptyLabel 测试空标签名
func TestGetLokiLabels_EmptyLabel(t *testing.T) {
	controller := NewMonitoringController()

	// 创建路由上下文
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("label", "")

	req := httptest.NewRequest(http.MethodGet, "/monitoring/loki/labels//values", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.GetLokiLabels(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "标签名称不能为空")
}

// TestGetLokiLabels_ValidLabel 测试有效标签
func TestGetLokiLabels_ValidLabel(t *testing.T) {
	controller := NewMonitoringController()

	// 创建路由上下文
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("label", "job")

	req := httptest.NewRequest(http.MethodGet, "/monitoring/loki/labels/job/values", nil)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	controller.GetLokiLabels(w, req)

	// 如果 Loki 服务不可用，会返回500；如果可用，返回200
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// ===================== 集成测试 =====================

// TestGetLogTemplateDescriptions 测试获取日志模板描述
func TestGetLogTemplateDescriptions(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/logs/descriptions", nil)
	w := httptest.NewRecorder()

	controller.GetLogTemplateDescriptions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Status)
	assert.NotNil(t, response.Data)

	// 验证返回的数据结构
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Greater(t, len(data), 0)

	// 验证关键描述存在
	assert.Contains(t, data, "all_app_logs")
	assert.Contains(t, data, "error_logs")
	assert.Contains(t, data, "http_requests")
}

// TestLogTemplates_RealWorld 测试真实世界的日志查询模板
func TestLogTemplates_RealWorld(t *testing.T) {
	controller := NewMonitoringController()

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "查询指定应用所有日志",
			template: "all_app_logs",
			expected: "{app=\"$app\"}",
		},
		{
			name:     "查询指定命名空间日志",
			template: "all_namespace_logs",
			expected: "{namespace=\"$namespace\"}",
		},
		{
			name:     "查询所有Pod日志",
			template: "all_pods_logs",
			expected: "{pod=~\".+\"}",
		},
		{
			name:     "查询所有应用日志",
			template: "all_apps_logs",
			expected: "{app=~\".+\"}",
		},
		{
			name:     "查询错误日志",
			template: "error_logs",
			expected: "{namespace=\"$namespace\"} |~ `(?i)error|exception|fatal|panic`",
		},
		{
			name:     "查询HTTP请求",
			template: "http_requests",
			expected: "{namespace=\"$namespace\"} |~ `\"(GET|POST|PUT|DELETE|PATCH)`",
		},
		{
			name:     "查询健康检查",
			template: "health_check_logs",
			expected: "{namespace=\"$namespace\"} |~ `/health`",
		},
		{
			name:     "多标签组合查询",
			template: "app_in_namespace",
			expected: "{namespace=\"$namespace\", app=\"$app\"}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 获取模板
			req := httptest.NewRequest(http.MethodGet, "/monitoring/templates/logs", nil)
			w := httptest.NewRecorder()
			controller.GetLogTemplates(w, req)

			var response APIResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)

			data, ok := response.Data.(map[string]interface{})
			require.True(t, ok)

			// 验证模板内容
			template, exists := data[tc.template]
			assert.True(t, exists, "模板 %s 应该存在", tc.template)
			assert.Equal(t, tc.expected, template, "模板内容应该匹配")
		})
	}
}

// TestLogQuery_InvalidRegex 测试无效的正则表达式查询
func TestLogQuery_InvalidRegex(t *testing.T) {
	controller := NewMonitoringController()

	// Loki 不支持 .* 这种可以匹配空字符串的表达式
	reqBody := meta.MonitorQueryRequest{
		QueryType: "logs",
		Query:     "{app=~\".*\"}", // 无效查询
		Limit:     1000,
		StartTime: 1760335392,
		EndTime:   1760338992,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryLogs(w, req)

	// 应该返回错误
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "HTTP请求失败: 状态码=400")
	t.Logf("错误信息: %s", response.Msg)
}

// TestLogQuery_WithTimeRange 测试带时间范围的日志查询（真实场景）
func TestLogQuery_WithTimeRange(t *testing.T) {
	controller := NewMonitoringController()

	// 使用修正后的查询（Loki 要求至少匹配一个字符）
	reqBody := meta.MonitorQueryRequest{
		QueryType: "logs",
		Query:     "{app=~\".+\"}", // Loki 要求至少匹配一个字符
		Limit:     1000,
		StartTime: 1760335392,
		EndTime:   1760338992,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/monitoring/query/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	controller.QueryLogs(w, req)

	// 应该成功或返回服务不可用错误
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)

	if w.Code == http.StatusOK {
		var response APIResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, 0, response.Status)
		t.Logf("查询成功，返回数据类型: %T", response.Data)
	} else {
		var response APIResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		t.Logf("查询失败: %s", response.Msg)
	}
}

// TestLogQuery_KubernetesLogs 测试 Kubernetes 日志查询
func TestLogQuery_KubernetesLogs(t *testing.T) {
	controller := NewMonitoringController()

	testCases := []struct {
		name    string
		query   string
		limit   int
		wantErr bool
	}{
		{
			name:    "查询flow-service日志",
			query:   "{app=\"flow-service\"}",
			limit:   10,
			wantErr: false,
		},
		{
			name:    "查询健康检查日志",
			query:   "{namespace=\"datahub\"} |~ `/health`",
			limit:   10,
			wantErr: false,
		},
		{
			name:    "查询HTTP错误",
			query:   "{namespace=\"datahub\"} |~ `\" - 5\\\\d{2}`",
			limit:   10,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := meta.MonitorQueryRequest{
				QueryType: "logs",
				Query:     tc.query,
				Limit:     tc.limit,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/monitoring/query/logs", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			controller.QueryLogs(w, req)

			// 如果 Loki 可用，应该返回200或500（查询失败）
			assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)

			if w.Code == http.StatusOK {
				var response APIResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, 0, response.Status)
				// 验证返回的数据结构
				if response.Data != nil {
					data, ok := response.Data.(map[string]interface{})
					if ok {
						// 应该包含 resultType 和 result 字段
						assert.Contains(t, data, "resultType")
					}
				}
			}
		})
	}
}

// TestMonitoringWorkflow 测试完整的监控工作流
func TestMonitoringWorkflow(t *testing.T) {
	controller := NewMonitoringController()

	t.Run("Step1_GetConfig", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monitoring/config", nil)
		w := httptest.NewRecorder()
		controller.GetMonitoringConfig(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Step2_GetMetricTemplates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monitoring/templates/metrics", nil)
		w := httptest.NewRecorder()
		controller.GetMetricTemplates(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Step3_GetLogTemplates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monitoring/templates/logs", nil)
		w := httptest.NewRecorder()
		controller.GetLogTemplates(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Step4_GetMetricDescriptions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monitoring/metrics/descriptions", nil)
		w := httptest.NewRecorder()
		controller.GetMetricDescriptions(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Step5_GetLogDescriptions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/monitoring/logs/descriptions", nil)
		w := httptest.NewRecorder()
		controller.GetLogTemplateDescriptions(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Step6_ValidateMetricQuery", func(t *testing.T) {
		reqBody := meta.MonitorQueryRequest{
			QueryType: "metrics",
			Query:     "cpu_usage_active",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/monitoring/query/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		controller.ValidateQuery(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Step7_ValidateLogQuery", func(t *testing.T) {
		reqBody := meta.MonitorQueryRequest{
			QueryType: "logs",
			Query:     "{app=\"flow-service\"}",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/monitoring/query/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		controller.ValidateQuery(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
