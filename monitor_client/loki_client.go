package monitor_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cast"
)

var LokiUrl = "http://mh1:3100"
var lokiClient = &http.Client{
	Timeout: 30 * time.Second,
}

func init() {
	if envUrl := os.Getenv("LOKI_URL"); envUrl != "" {
		LokiUrl = envUrl
	}
}

// SetLokiUrl 设置 Loki 的 URL（用于测试）
func SetLokiUrl(url string) {
	LokiUrl = url
}

// GetLokiUrl 获取当前 Loki 的 URL
func GetLokiUrl() string {
	return LokiUrl
}

// LokiQuery 执行 Loki 即时查询
func LokiQuery(ctx context.Context, query string, limit int) (result *QueryResult, err error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	if limit <= 0 {
		limit = 100 // 默认限制100条
	}

	values := url.Values{}
	values.Add("query", query)
	values.Add("limit", cast.ToString(limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LokiUrl+"/loki/api/v1/query", nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = values.Encode()

	resp, err := lokiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	var metricsResp QueryResultResp
	if err = json.NewDecoder(resp.Body).Decode(&metricsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if metricsResp.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", metricsResp.Status)
	}

	return &metricsResp.Data, nil
}

// LokiStreamQuery 执行 Loki 流查询（区间查询）
func LokiStreamQuery(ctx context.Context, query string, limit int, preHours int) (result *LokiQueryResult, err error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	if limit <= 0 {
		limit = 1000 // 默认限制1000条
	}

	if preHours <= 0 {
		preHours = 1 // 默认查询1小时
	}

	values := url.Values{}
	values.Add("query", query)
	values.Add("limit", cast.ToString(limit))
	end := time.Now().UnixNano()
	start := time.Now().Add(time.Duration(-1*preHours) * time.Hour).UnixNano()
	values.Add("start", cast.ToString(start))
	values.Add("end", cast.ToString(end))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LokiUrl+"/loki/api/v1/query_range", nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = values.Encode()

	resp, err := lokiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 先读取响应体用于调试
	var lokiResp LokiQueryResultResp
	if err = json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", lokiResp.Status)
	}

	return &lokiResp.Data, nil
}

// LokiRangeQuery 执行 Loki 区间查询（支持指定时间范围）
func LokiRangeQuery(ctx context.Context, query string, limit int, start, end time.Time) (result *LokiQueryResult, err error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	if limit <= 0 {
		limit = 1000 // 默认限制1000条
	}

	values := url.Values{}
	values.Add("query", query)
	values.Add("limit", cast.ToString(limit))
	values.Add("start", cast.ToString(start.UnixNano()))
	values.Add("end", cast.ToString(end.UnixNano()))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LokiUrl+"/loki/api/v1/query_range", nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = values.Encode()

	resp, err := lokiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 读取错误消息
		bodyBytes := make([]byte, 500)
		n, _ := resp.Body.Read(bodyBytes)
		return nil, fmt.Errorf("HTTP请求失败: 状态码=%d, 响应=%s", resp.StatusCode, string(bodyBytes[:n]))
	}

	var lokiResp LokiQueryResultResp
	if err = json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", lokiResp.Status)
	}

	return &lokiResp.Data, nil
}

// LokiLabelValues 获取指定标签的所有值
func LokiLabelValues(ctx context.Context, label string) (result []string, err error) {
	if label == "" {
		return nil, errors.New("label cannot be empty")
	}

	urlSuffix := "/loki/api/v1/label/" + label + "/values"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LokiUrl+urlSuffix, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := lokiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	var lokiResp LokiLabelValueResp
	if err = json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", lokiResp.Status)
	}

	return lokiResp.Data, nil
}
