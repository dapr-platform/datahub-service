/*
 * @module api/controllers/monitoring_controller_test
 * @description 监控控制器单元测试
 * @architecture 测试层
 * @documentReference .specify/memory/test_plan.md
 * @stateFlow 测试准备 -> 请求构建 -> 响应验证
 * @rules 确保监控API的正确性
 * @dependencies testing, net/http/httptest
 */

package controllers

import (
	"bytes"
	"datahub-service/service/meta"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetMetricTemplates 测试获取指标模板
func TestGetMetricTemplates(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/templates/metrics", nil)
	w := httptest.NewRecorder()

	controller.GetMetricTemplates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Status)
	assert.NotNil(t, response.Data)
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
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Status)
	assert.NotNil(t, response.Data)
}

// TestGetMonitoringConfig 测试获取监控配置
func TestGetMonitoringConfig(t *testing.T) {
	controller := NewMonitoringController()

	req := httptest.NewRequest(http.MethodGet, "/monitoring/config", nil)
	w := httptest.NewRecorder()

	controller.GetMonitoringConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Status)

	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, data, "victoria_metrics")
	assert.Contains(t, data, "loki")
}

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
	assert.NoError(t, err)
	assert.Equal(t, -1, response.Status)
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
	assert.NoError(t, err)
	assert.Equal(t, -1, response.Status)
}

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
	assert.NoError(t, err)
	assert.Equal(t, -1, response.Status)
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
	assert.NoError(t, err)
	assert.Equal(t, -1, response.Status)
	assert.Contains(t, response.Msg, "查询类型不能为空")
}
