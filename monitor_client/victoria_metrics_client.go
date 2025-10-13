package monitor_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var VictoriaMetricsUrl = "http://mh1:38428"
var client = &http.Client{
	Timeout: 30 * time.Second,
}

func init() {
	if envUrl := os.Getenv("VICTORIA_METRICS_URL"); envUrl != "" {
		VictoriaMetricsUrl = envUrl
	}
}

// SetVictoriaMetricsUrl 设置 VictoriaMetrics 的 URL（用于测试）
func SetVictoriaMetricsUrl(url string) {
	VictoriaMetricsUrl = url
}

// GetVictoriaMetricsUrl 获取当前 VictoriaMetrics 的 URL
func GetVictoriaMetricsUrl() string {
	return VictoriaMetricsUrl
}

// Query 执行即时查询
func Query(ctx context.Context, query string, queryTime time.Time) (result *QueryResult, err error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	if queryTime.IsZero() {
		queryTime = time.Now()
	}

	values := url.Values{}
	values.Add("query", query)
	values.Add("time", formatTime(queryTime))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, VictoriaMetricsUrl+"/api/v1/query", nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = values.Encode()

	resp, err := client.Do(req)
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
func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix()), 'f', -1, 64)
}

// QueryRange 执行区间查询
func QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (result *QueryResult, err error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	if start.IsZero() || end.IsZero() {
		return nil, errors.New("start and end time cannot be zero")
	}

	if start.After(end) {
		return nil, errors.New("start time must be before end time")
	}

	if step <= 0 {
		step = 15 * time.Second // 默认步长15秒
	}

	u, err := url.Parse(VictoriaMetricsUrl + "/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("解析URL失败: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", formatTime(start))
	q.Set("end", formatTime(end))
	q.Set("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(q.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var metricsResp QueryResultResp
	if err = json.Unmarshal(body, &metricsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if metricsResp.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", metricsResp.Status)
	}

	return &metricsResp.Data, nil
}
