/*
 * @module api/controllers/monitoring_controller
 * @description 监控控制器（简化版），基于 VictoriaMetrics 和 Loki 提供监控查询功能
 * @architecture MVC架构 - 控制器层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 请求接收 -> 查询执行 -> 响应返回
 * @rules 简化监控功能，直接使用 VictoriaMetrics 和 Loki 客户端
 * @dependencies datahub-service/monitor_client, net/http
 */

package controllers

import (
	"datahub-service/monitor_client"
	"datahub-service/service/meta"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// MonitoringController 监控控制器（简化版）
type MonitoringController struct{}

// NewMonitoringController 创建监控控制器实例
func NewMonitoringController() *MonitoringController {
	return &MonitoringController{}
}

// QueryMetrics 通用指标查询接口
// @Summary 通用指标查询
// @Description 执行自定义的 PromQL 查询，支持即时查询和区间查询
// @Tags 监控查询
// @Accept json
// @Produce json
// @Param query body meta.MonitorQueryRequest true "查询请求"
// @Success 200 {object} APIResponse{data=object}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /monitoring/query/metrics [post]
func (c *MonitoringController) QueryMetrics(w http.ResponseWriter, r *http.Request) {
	var req meta.MonitorQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, "解析请求失败", err)
		return
	}

	if req.Query == "" {
		c.writeErrorResponse(w, http.StatusBadRequest, "查询语句不能为空", nil)
		return
	}

	ctx := r.Context()

	// 判断是即时查询还是区间查询
	if req.StartTime > 0 && req.EndTime > 0 {
		// 区间查询
		start := time.Unix(req.StartTime, 0)
		end := time.Unix(req.EndTime, 0)
		step := time.Duration(req.Step) * time.Second
		if step == 0 {
			step = 15 * time.Second
		}

		result, err := monitor_client.QueryRange(ctx, req.Query, start, end, step)
		if err != nil {
			c.writeErrorResponse(w, http.StatusInternalServerError, "查询指标失败", err)
			return
		}

		c.writeSuccessResponse(w, "查询指标成功", result)
	} else {
		// 即时查询
		queryTime := time.Now()
		if req.EndTime > 0 {
			queryTime = time.Unix(req.EndTime, 0)
		}

		result, err := monitor_client.Query(ctx, req.Query, queryTime)
		if err != nil {
			c.writeErrorResponse(w, http.StatusInternalServerError, "查询指标失败", err)
			return
		}

		c.writeSuccessResponse(w, "查询指标成功", result)
	}
}

// QueryLogs 通用日志查询接口
// @Summary 通用日志查询
// @Description 执行自定义的 LogQL 查询
// @Tags 监控查询
// @Accept json
// @Produce json
// @Param query body meta.MonitorQueryRequest true "查询请求"
// @Success 200 {object} APIResponse{data=object}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /monitoring/query/logs [post]
func (c *MonitoringController) QueryLogs(w http.ResponseWriter, r *http.Request) {
	var req meta.MonitorQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, "解析请求失败", err)
		return
	}

	if req.Query == "" {
		c.writeErrorResponse(w, http.StatusBadRequest, "查询语句不能为空", nil)
		return
	}

	ctx := r.Context()
	limit := req.Limit
	if limit <= 0 {
		limit = 1000
	}

	// 判断是即时查询还是区间查询
	if req.StartTime > 0 && req.EndTime > 0 {
		// 区间查询 - 使用指定的时间范围
		start := time.Unix(req.StartTime, 0)
		end := time.Unix(req.EndTime, 0)

		result, err := monitor_client.LokiRangeQuery(ctx, req.Query, limit, start, end)
		if err != nil {
			c.writeErrorResponse(w, http.StatusInternalServerError, "查询日志失败", err)
			return
		}

		c.writeSuccessResponse(w, "查询日志成功", result)
	} else {
		// 即时查询
		result, err := monitor_client.LokiQuery(ctx, req.Query, limit)
		if err != nil {
			c.writeErrorResponse(w, http.StatusInternalServerError, "查询日志失败", err)
			return
		}

		c.writeSuccessResponse(w, "查询日志成功", result)
	}
}

// GetMetricTemplates 获取指标查询模板
// @Summary 获取指标查询模板
// @Description 获取常用的指标查询模板列表
// @Tags 监控模板
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]string}
// @Failure 500 {object} APIResponse
// @Router /monitoring/templates/metrics [get]
func (c *MonitoringController) GetMetricTemplates(w http.ResponseWriter, r *http.Request) {
	c.writeSuccessResponse(w, "获取指标模板成功", meta.CommonMetricTemplates)
}

// GetLogTemplates 获取日志查询模板
// @Summary 获取日志查询模板
// @Description 获取常用的日志查询模板列表
// @Tags 监控模板
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]string}
// @Failure 500 {object} APIResponse
// @Router /monitoring/templates/logs [get]
func (c *MonitoringController) GetLogTemplates(w http.ResponseWriter, r *http.Request) {
	c.writeSuccessResponse(w, "获取日志模板成功", meta.CommonLogTemplates)
}

// GetMetricDescriptions 获取指标描述信息
// @Summary 获取指标描述
// @Description 获取所有指标的中文描述信息
// @Tags 监控模板
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]string}
// @Router /monitoring/metrics/descriptions [get]
func (c *MonitoringController) GetMetricDescriptions(w http.ResponseWriter, r *http.Request) {
	c.writeSuccessResponse(w, "获取指标描述成功", meta.MetricDescriptions)
}

// GetLogTemplateDescriptions 获取日志模板描述信息
// @Summary 获取日志模板描述
// @Description 获取所有日志查询模板的中文描述信息
// @Tags 监控模板
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]string}
// @Router /monitoring/logs/descriptions [get]
func (c *MonitoringController) GetLogTemplateDescriptions(w http.ResponseWriter, r *http.Request) {
	c.writeSuccessResponse(w, "获取日志模板描述成功", meta.LogTemplateDescriptions)
}

// GetLokiLabels 获取 Loki 标签值
// @Summary 获取日志标签值
// @Description 获取指定标签的所有可能值
// @Tags 监控查询
// @Accept json
// @Produce json
// @Param label path string true "标签名称"
// @Success 200 {object} APIResponse{data=[]string}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /monitoring/loki/labels/{label}/values [get]
func (c *MonitoringController) GetLokiLabels(w http.ResponseWriter, r *http.Request) {
	label := chi.URLParam(r, "label")
	if label == "" {
		c.writeErrorResponse(w, http.StatusBadRequest, "标签名称不能为空", nil)
		return
	}

	ctx := r.Context()
	values, err := monitor_client.LokiLabelValues(ctx, label)
	if err != nil {
		c.writeErrorResponse(w, http.StatusInternalServerError, "获取标签值失败", err)
		return
	}

	c.writeSuccessResponse(w, "获取标签值成功", values)
}

// ExecuteCustomQuery 执行自定义查询（统一入口）
// @Summary 执行自定义查询
// @Description 根据查询类型自动选择 Metrics 或 Logs 查询
// @Tags 监控查询
// @Accept json
// @Produce json
// @Param query body meta.MonitorQueryRequest true "查询请求"
// @Success 200 {object} APIResponse{data=object}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /monitoring/query [post]
func (c *MonitoringController) ExecuteCustomQuery(w http.ResponseWriter, r *http.Request) {
	var req meta.MonitorQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, "解析请求失败", err)
		return
	}

	if req.QueryType == "" {
		c.writeErrorResponse(w, http.StatusBadRequest, "查询类型不能为空", nil)
		return
	}

	if req.Query == "" {
		c.writeErrorResponse(w, http.StatusBadRequest, "查询语句不能为空", nil)
		return
	}

	ctx := r.Context()

	// 根据查询类型执行对应的查询
	switch req.QueryType {
	case "metrics":
		// 判断是即时查询还是区间查询
		if req.StartTime > 0 && req.EndTime > 0 {
			start := time.Unix(req.StartTime, 0)
			end := time.Unix(req.EndTime, 0)
			step := time.Duration(req.Step) * time.Second
			if step == 0 {
				step = 15 * time.Second
			}
			result, err := monitor_client.QueryRange(ctx, req.Query, start, end, step)
			if err != nil {
				c.writeErrorResponse(w, http.StatusInternalServerError, "查询指标失败", err)
				return
			}
			c.writeSuccessResponse(w, "查询指标成功", result)
		} else {
			queryTime := time.Now()
			if req.EndTime > 0 {
				queryTime = time.Unix(req.EndTime, 0)
			}
			result, err := monitor_client.Query(ctx, req.Query, queryTime)
			if err != nil {
				c.writeErrorResponse(w, http.StatusInternalServerError, "查询指标失败", err)
				return
			}
			c.writeSuccessResponse(w, "查询指标成功", result)
		}
	case "logs":
		limit := req.Limit
		if limit <= 0 {
			limit = 1000
		}
		if req.StartTime > 0 && req.EndTime > 0 {
			// 区间查询 - 使用指定的时间范围
			start := time.Unix(req.StartTime, 0)
			end := time.Unix(req.EndTime, 0)
			result, err := monitor_client.LokiRangeQuery(ctx, req.Query, limit, start, end)
			if err != nil {
				c.writeErrorResponse(w, http.StatusInternalServerError, "查询日志失败", err)
				return
			}
			c.writeSuccessResponse(w, "查询日志成功", result)
		} else {
			// 即时查询
			result, err := monitor_client.LokiQuery(ctx, req.Query, limit)
			if err != nil {
				c.writeErrorResponse(w, http.StatusInternalServerError, "查询日志失败", err)
				return
			}
			c.writeSuccessResponse(w, "查询日志成功", result)
		}
	default:
		c.writeErrorResponse(w, http.StatusBadRequest, "不支持的查询类型: "+req.QueryType, nil)
	}
}

// GetMonitoringConfig 获取监控配置
// @Summary 获取监控配置
// @Description 获取VictoriaMetrics和Loki的连接配置
// @Tags 监控配置
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=object}
// @Router /monitoring/config [get]
func (c *MonitoringController) GetMonitoringConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"victoria_metrics": map[string]string{
			"url":    monitor_client.GetVictoriaMetricsUrl(),
			"status": "connected",
		},
		"loki": map[string]string{
			"url":    monitor_client.GetLokiUrl(),
			"status": "connected",
		},
		"description": "基于 VictoriaMetrics 和 Loki 的监控系统",
		"version":     "1.0.0",
	}

	c.writeSuccessResponse(w, "获取监控配置成功", config)
}

// ValidateQuery 验证查询语法
// @Summary 验证查询语法
// @Description 验证 PromQL 或 LogQL 查询语法是否正确
// @Tags 监控查询
// @Accept json
// @Produce json
// @Param query body meta.MonitorQueryRequest true "查询请求"
// @Success 200 {object} APIResponse{data=object}
// @Failure 400 {object} APIResponse
// @Router /monitoring/query/validate [post]
func (c *MonitoringController) ValidateQuery(w http.ResponseWriter, r *http.Request) {
	var req meta.MonitorQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, "解析请求失败", err)
		return
	}

	if req.Query == "" {
		c.writeErrorResponse(w, http.StatusBadRequest, "查询语句不能为空", nil)
		return
	}

	ctx := r.Context()
	validation := map[string]interface{}{
		"valid":      true,
		"query":      req.Query,
		"query_type": req.QueryType,
	}

	// 简单的验证：尝试执行查询
	if req.QueryType == "metrics" {
		_, err := monitor_client.Query(ctx, req.Query, time.Now())
		if err != nil {
			validation["valid"] = false
			validation["error"] = err.Error()
		}
	} else if req.QueryType == "logs" {
		_, err := monitor_client.LokiQuery(ctx, req.Query, 1)
		if err != nil {
			validation["valid"] = false
			validation["error"] = err.Error()
		}
	}

	c.writeSuccessResponse(w, "查询验证完成", validation)
}

// 辅助方法

// writeSuccessResponse 写入成功响应
func (c *MonitoringController) writeSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	response := &APIResponse{
		Status: 0,
		Msg:    message,
		Data:   data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse 写入错误响应
func (c *MonitoringController) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	response := &APIResponse{
		Status: -1,
		Msg:    errorMsg,
		Data:   nil,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
