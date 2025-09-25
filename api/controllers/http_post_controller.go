/*
 * @module api/controllers/http_post_controller
 * @description HTTP POST数据源控制器，处理统一的webhook接收
 * @architecture RESTful API控制器
 * @documentReference ai_docs/datasource_req1.md
 * @stateFlow 无状态HTTP请求处理
 * @rules 根据URL后缀路由到对应的数据源处理器
 * @dependencies github.com/go-chi/chi/v5, encoding/json, net/http
 * @refs service/datasource/http_post.go
 */

package controllers

import (
	"encoding/json"
	"net/http"

	"datahub-service/service/datasource"

	"github.com/go-chi/chi/v5"
)

// HTTPPostController HTTP POST数据源控制器
type HTTPPostController struct {
}

// NewHTTPPostController 创建HTTP POST控制器
func NewHTTPPostController() *HTTPPostController {
	return &HTTPPostController{}
}

// HandleWebhook 处理webhook请求
func (c *HTTPPostController) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 获取URL后缀参数
	suffix := chi.URLParam(r, "suffix")
	if suffix == "" {
		c.sendErrorResponse(w, http.StatusBadRequest, "缺少URL后缀参数")
		return
	}

	// 查找对应的数据源
	ds, exists := datasource.GetHTTPPostDataSource(suffix)
	if !exists {
		c.sendErrorResponse(w, http.StatusNotFound, "未找到对应的HTTP POST数据源: "+suffix)
		return
	}

	// 检查数据源是否已启动
	if !ds.IsStarted() {
		c.sendErrorResponse(w, http.StatusServiceUnavailable, "HTTP POST数据源未启动: "+suffix)
		return
	}

	// 委托给数据源处理
	ds.HandleWebhook(w, r)
}

// GetDataSourceList 获取HTTP POST数据源列表
func (c *HTTPPostController) GetDataSourceList(w http.ResponseWriter, r *http.Request) {
	dataSources := datasource.ListHTTPPostDataSources()

	result := make([]map[string]interface{}, 0, len(dataSources))
	for suffix, ds := range dataSources {
		info := map[string]interface{}{
			"suffix":      suffix,
			"id":          ds.GetID(),
			"type":        ds.GetType(),
			"is_started":  ds.IsStarted(),
			"is_resident": ds.IsResident(),
			"data_count":  len(ds.GetReceivedData()),
		}
		result = append(result, info)
	}

	c.sendSuccessResponse(w, map[string]interface{}{
		"success":     true,
		"message":     "获取HTTP POST数据源列表成功",
		"data":        result,
		"total_count": len(result),
	})
}

// GetDataSourceStatus 获取特定HTTP POST数据源状态
func (c *HTTPPostController) GetDataSourceStatus(w http.ResponseWriter, r *http.Request) {
	suffix := chi.URLParam(r, "suffix")
	if suffix == "" {
		c.sendErrorResponse(w, http.StatusBadRequest, "缺少URL后缀参数")
		return
	}

	ds, exists := datasource.GetHTTPPostDataSource(suffix)
	if !exists {
		c.sendErrorResponse(w, http.StatusNotFound, "未找到对应的HTTP POST数据源: "+suffix)
		return
	}

	// 执行健康检查
	healthStatus, err := ds.HealthCheck(r.Context())
	if err != nil {
		c.sendErrorResponse(w, http.StatusInternalServerError, "健康检查失败: "+err.Error())
		return
	}

	c.sendSuccessResponse(w, map[string]interface{}{
		"success": true,
		"message": "获取数据源状态成功",
		"data": map[string]interface{}{
			"suffix":        suffix,
			"id":            ds.GetID(),
			"type":          ds.GetType(),
			"health_status": healthStatus,
			"data_count":    len(ds.GetReceivedData()),
		},
	})
}

// GetReceivedData 获取接收到的数据
func (c *HTTPPostController) GetReceivedData(w http.ResponseWriter, r *http.Request) {
	suffix := chi.URLParam(r, "suffix")
	if suffix == "" {
		c.sendErrorResponse(w, http.StatusBadRequest, "缺少URL后缀参数")
		return
	}

	ds, exists := datasource.GetHTTPPostDataSource(suffix)
	if !exists {
		c.sendErrorResponse(w, http.StatusNotFound, "未找到对应的HTTP POST数据源: "+suffix)
		return
	}

	// 获取查询参数
	query := r.URL.Query()
	limit := 100 // 默认限制
	offset := 0  // 默认偏移

	if l := query.Get("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := query.Get("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// 获取数据
	allData := ds.GetReceivedData()
	total := len(allData)

	// 计算分页
	start := offset
	end := offset + limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var data []map[string]interface{}
	if start < end {
		data = allData[start:end]
	} else {
		data = make([]map[string]interface{}, 0)
	}

	c.sendSuccessResponse(w, map[string]interface{}{
		"success": true,
		"message": "获取接收数据成功",
		"data":    data,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
			"count":  len(data),
		},
	})
}

// ClearReceivedData 清空接收到的数据
func (c *HTTPPostController) ClearReceivedData(w http.ResponseWriter, r *http.Request) {
	suffix := chi.URLParam(r, "suffix")
	if suffix == "" {
		c.sendErrorResponse(w, http.StatusBadRequest, "缺少URL后缀参数")
		return
	}

	ds, exists := datasource.GetHTTPPostDataSource(suffix)
	if !exists {
		c.sendErrorResponse(w, http.StatusNotFound, "未找到对应的HTTP POST数据源: "+suffix)
		return
	}

	// 清空数据
	ds.ClearReceivedData()

	c.sendSuccessResponse(w, map[string]interface{}{
		"success": true,
		"message": "清空接收数据成功",
	})
}

// sendSuccessResponse 发送成功响应
func (c *HTTPPostController) sendSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// sendErrorResponse 发送错误响应
func (c *HTTPPostController) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
		"code":    statusCode,
	})
}

// parseIntParam 解析整数参数
func parseIntParam(param string) (int, error) {
	var result int
	if err := json.Unmarshal([]byte(param), &result); err != nil {
		return 0, err
	}
	return result, nil
}
