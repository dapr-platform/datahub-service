package meta

// MonitorQueryRequest 监控查询请求
type MonitorQueryRequest struct {
	QueryType string `json:"query_type"` // metrics 或 logs
	Query     string `json:"query"`      // PromQL 或 LogQL 查询语句
	StartTime int64  `json:"start_time"` // Unix 时间戳（秒），可选
	EndTime   int64  `json:"end_time"`   // Unix 时间戳（秒），可选
	Step      int    `json:"step"`       // 步长（秒），仅用于区间查询
	Limit     int    `json:"limit"`      // 限制结果数量，主要用于日志查询
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // metrics 或 logs
	Query       string            `json:"query"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   int64             `json:"created_at"`
	UpdatedAt   int64             `json:"updated_at"`
}

// CommonMetricTemplates 常用指标模板
var CommonMetricTemplates = map[string]string{
	// 系统指标
	"cpu_usage":             "100 - (avg by (instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
	"memory_usage":          "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100",
	"disk_usage":            "(1 - (node_filesystem_avail_bytes{fstype!=\"tmpfs\"} / node_filesystem_size_bytes{fstype!=\"tmpfs\"})) * 100",
	"network_receive_rate":  "rate(node_network_receive_bytes_total[5m])",
	"network_transmit_rate": "rate(node_network_transmit_bytes_total[5m])",

	// 应用指标
	"http_request_rate":     "rate(http_requests_total[5m])",
	"http_request_duration": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
	"error_rate":            "rate(http_requests_total{status=~\"5..\"}[5m])",

	// 数据库指标
	"db_connections":      "pg_stat_database_numbackends",
	"db_query_duration":   "rate(pg_stat_statements_total_time[5m]) / rate(pg_stat_statements_calls[5m])",
	"db_transaction_rate": "rate(pg_stat_database_xact_commit[5m]) + rate(pg_stat_database_xact_rollback[5m])",
}

// CommonLogTemplates 常用日志查询模板
var CommonLogTemplates = map[string]string{
	// 基础日志查询
	"error_logs":   "{job=\"$job\"} |= \"error\" or \"ERROR\"",
	"warning_logs": "{job=\"$job\"} |= \"warning\" or \"WARNING\"",
	"all_app_logs": "{job=\"$job\", app=\"$app\"}",

	// 特定服务日志
	"nginx_error":   "{job=\"nginx\"} |= \"error\"",
	"app_exception": "{job=\"$job\"} |~ \"Exception|Error|Fatal\"",

	// 按日志级别统计
	"log_count_by_level": "sum by (level) (count_over_time({job=\"$job\"}[5m]))",
}
