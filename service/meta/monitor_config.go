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

// CommonMetricTemplates 常用指标模板（基于实际采集的指标）
var CommonMetricTemplates = map[string]string{
	// CPU 指标
	"cpu_usage_active":  "cpu_usage_active",
	"cpu_usage_idle":    "cpu_usage_idle",
	"cpu_usage_iowait":  "cpu_usage_iowait",
	"cpu_usage_system":  "cpu_usage_system",
	"cpu_usage_user":    "cpu_usage_user",
	"cpu_usage_percent": "100 - cpu_usage_idle",

	// 内存指标
	"mem_total":             "mem_total",
	"mem_used":              "mem_used",
	"mem_free":              "mem_free",
	"mem_available":         "mem_available",
	"mem_used_percent":      "mem_used_percent",
	"mem_available_percent": "mem_available_percent",
	"mem_cached":            "mem_cached",
	"mem_buffered":          "mem_buffered",

	// 磁盘指标
	"disk_total":        "disk_total",
	"disk_used":         "disk_used",
	"disk_free":         "disk_free",
	"disk_used_percent": "disk_used_percent",

	// 磁盘 IO 指标
	"diskio_reads":       "rate(diskio_reads[5m])",
	"diskio_writes":      "rate(diskio_writes[5m])",
	"diskio_read_bytes":  "rate(diskio_read_bytes[5m])",
	"diskio_write_bytes": "rate(diskio_write_bytes[5m])",
	"diskio_read_time":   "rate(diskio_read_time[5m])",
	"diskio_write_time":  "rate(diskio_write_time[5m])",

	// 网络指标
	"net_bytes_recv":   "rate(net_bytes_recv[5m])",
	"net_bytes_sent":   "rate(net_bytes_sent[5m])",
	"net_packets_recv": "rate(net_packets_recv[5m])",
	"net_packets_sent": "rate(net_packets_sent[5m])",
	"net_drop_in":      "rate(net_drop_in[5m])",
	"net_drop_out":     "rate(net_drop_out[5m])",
	"net_err_in":       "rate(net_err_in[5m])",
	"net_err_out":      "rate(net_err_out[5m])",

	// 系统负载
	"system_load1":  "system_load1",
	"system_load5":  "system_load5",
	"system_load15": "system_load15",
	"system_n_cpus": "system_n_cpus",
	"system_uptime": "system_uptime",

	// 进程指标
	"processes_total":         "processes_total",
	"processes_running":       "processes_running",
	"processes_sleeping":      "processes_sleeping",
	"processes_blocked":       "processes_blocked",
	"processes_zombies":       "processes_zombies",
	"processes_total_threads": "processes_total_threads",

	// 网络连接状态
	"netstat_tcp_established": "netstat_tcp_established",
	"netstat_tcp_listen":      "netstat_tcp_listen",
	"netstat_tcp_time_wait":   "netstat_tcp_time_wait",
	"netstat_tcp_close_wait":  "netstat_tcp_close_wait",
	"netstat_tcp_syn_sent":    "netstat_tcp_syn_sent",
	"netstat_tcp_syn_recv":    "netstat_tcp_syn_recv",

	// Docker 容器指标（如果使用）
	"docker_container_cpu_usage_percent": "docker_container_cpu_usage_percent",
	"docker_container_mem_usage_percent": "docker_container_mem_usage_percent",
	"docker_n_containers_running":        "docker_n_containers_running",
	"docker_n_containers_stopped":        "docker_n_containers_stopped",
}

// CommonLogTemplates 常用日志查询模板（使用变量占位符，支持动态替换）
// 说明：模板中的变量格式为 $variable_name，使用时需要替换为实际值
// 例如：{app="$app"} -> {app="flow-service"}
var CommonLogTemplates = map[string]string{
	// 基础查询 - 按标签筛选
	"all_app_logs":       "{app=\"$app\"}",             // 指定应用的所有日志
	"all_namespace_logs": "{namespace=\"$namespace\"}", // 指定命名空间的所有日志
	"all_pod_logs":       "{pod=\"$pod\"}",             // 指定 Pod 的所有日志
	"all_container_logs": "{container=\"$container\"}", // 指定容器的所有日志
	"all_cluster_logs":   "{cluster=\"$cluster\"}",     // 指定集群的所有日志
	"all_node_logs":      "{node=\"$node\"}",           // 指定节点的所有日志
	"all_pods_logs":      "{pod=~\".+\"}",              // 所有 Pod 的日志
	"all_apps_logs":      "{app=~\".+\"}",              // 所有应用的日志

	// 错误日志查询
	"error_logs":       "{namespace=\"$namespace\"} |~ `(?i)error|exception|fatal|panic`", // 指定命名空间的错误日志
	"error_logs_all":   "{namespace=~\".+\"} |~ `(?i)error|exception|fatal|panic`",        // 所有命名空间的错误日志
	"app_error_logs":   "{app=\"$app\"} |~ `(?i)error|exception|fatal|panic`",             // 指定应用的错误日志
	"pod_error_logs":   "{pod=~\".+\"} |~ `(?i)error|exception|fatal|panic`",              // 所有 Pod 的错误日志
	"warning_logs":     "{namespace=\"$namespace\"} |~ `(?i)warning|warn`",                // 指定命名空间的警告日志
	"warning_logs_all": "{namespace=~\".+\"} |~ `(?i)warning|warn`",                       // 所有命名空间的警告日志
	"app_warning_logs": "{app=\"$app\"} |~ `(?i)warning|warn`",                            // 指定应用的警告日志

	// HTTP 请求日志
	"http_requests":      "{namespace=\"$namespace\"} |~ `\"(GET|POST|PUT|DELETE|PATCH)`", // HTTP 请求日志
	"http_requests_all":  "{namespace=~\".+\"} |~ `\"(GET|POST|PUT|DELETE|PATCH)`",        // 所有 HTTP 请求日志
	"http_errors":        "{namespace=\"$namespace\"} |~ `\" - (4|5)\\d{2}`",              // HTTP 错误日志
	"http_5xx_errors":    "{namespace=\"$namespace\"} |~ `\" - 5\\d{2}`",                  // 5xx 错误日志
	"http_4xx_errors":    "{namespace=\"$namespace\"} |~ `\" - 4\\d{2}`",                  // 4xx 错误日志
	"http_slow_requests": "{namespace=\"$namespace\"} |~ `in \\d+\\.\\d+s`",               // 慢请求日志
	"http_get_requests":  "{namespace=\"$namespace\"} |~ `\"GET`",                         // GET 请求日志
	"http_post_requests": "{namespace=\"$namespace\"} |~ `\"POST`",                        // POST 请求日志

	// 健康检查日志
	"health_check_logs":     "{namespace=\"$namespace\"} |~ `/health`",             // 健康检查日志
	"health_check_errors":   "{namespace=\"$namespace\"} |~ `/health.*[^2]\\d{2}`", // 健康检查错误
	"health_check_logs_all": "{namespace=~\".+\"} |~ `/health`",                    // 所有健康检查日志

	// 按时间和来源过滤
	"recent_logs":     "{namespace=\"$namespace\"} | __timestamp__ > 1h", // 最近1小时日志
	"stdout_logs":     "{stream=\"stdout\"}",                             // 标准输出日志
	"stderr_logs":     "{stream=\"stderr\"}",                             // 标准错误日志
	"stdout_logs_app": "{app=\"$app\", stream=\"stdout\"}",               // 指定应用的标准输出
	"stderr_logs_app": "{app=\"$app\", stream=\"stderr\"}",               // 指定应用的标准错误

	// 聚合统计查询
	"log_count_by_app":       "sum by (app) (count_over_time({namespace=\"$namespace\"}[5m]))",                    // 按应用统计日志数
	"log_count_by_pod":       "sum by (pod) (count_over_time({namespace=\"$namespace\"}[5m]))",                    // 按 Pod 统计日志数
	"log_count_by_container": "sum by (container) (count_over_time({namespace=\"$namespace\"}[5m]))",              // 按容器统计日志数
	"log_count_by_node":      "sum by (node) (count_over_time({namespace=\"$namespace\"}[5m]))",                   // 按节点统计日志数
	"error_rate_by_app":      "sum by (app) (rate({namespace=\"$namespace\"} |~ `(?i)error` [5m]))",               // 按应用统计错误率
	"request_rate_by_app":    "sum by (app) (rate({namespace=\"$namespace\"} |~ `\"(GET|POST|PUT|DELETE)` [5m]))", // 按应用统计请求速率

	// 高级查询 - 使用 LogQL 解析器
	"parsed_json_logs":    "{namespace=\"$namespace\"} | json",                                                        // 解析 JSON 日志
	"filtered_level_logs": "{namespace=\"$namespace\"} | json | level=`error`",                                        // 按日志级别过滤
	"request_duration":    "{namespace=\"$namespace\"} | regexp `in (?P<duration>\\d+\\.\\d+)(µ|m)?s` | duration > 1", // 提取请求耗时

	// 多标签组合查询
	"app_in_namespace": "{namespace=\"$namespace\", app=\"$app\"}", // 命名空间+应用
	"pod_in_namespace": "{namespace=\"$namespace\", pod=\"$pod\"}", // 命名空间+Pod
	"container_in_pod": "{pod=\"$pod\", container=\"$container\"}", // Pod+容器
}

// LogTemplateDescriptions 日志查询模板描述
var LogTemplateDescriptions = map[string]string{
	// 基础查询
	"all_app_logs":       "查询指定应用的所有日志（需要替换 $app）",
	"all_namespace_logs": "查询指定命名空间的所有日志（需要替换 $namespace）",
	"all_pod_logs":       "查询指定 Pod 的所有日志（需要替换 $pod）",
	"all_container_logs": "查询指定容器的所有日志（需要替换 $container）",
	"all_cluster_logs":   "查询指定集群的所有日志（需要替换 $cluster）",
	"all_node_logs":      "查询指定节点的所有日志（需要替换 $node）",
	"all_pods_logs":      "查询所有 Pod 的日志",
	"all_apps_logs":      "查询所有应用的日志",

	// 错误和警告日志
	"error_logs":       "查询指定命名空间的错误日志（需要替换 $namespace）",
	"error_logs_all":   "查询所有命名空间的错误日志",
	"app_error_logs":   "查询指定应用的错误日志（需要替换 $app）",
	"pod_error_logs":   "查询所有 Pod 的错误日志",
	"warning_logs":     "查询指定命名空间的警告日志（需要替换 $namespace）",
	"warning_logs_all": "查询所有命名空间的警告日志",
	"app_warning_logs": "查询指定应用的警告日志（需要替换 $app）",

	// HTTP 请求日志
	"http_requests":      "查询指定命名空间的 HTTP 请求日志（需要替换 $namespace）",
	"http_requests_all":  "查询所有命名空间的 HTTP 请求日志",
	"http_errors":        "查询指定命名空间的 HTTP 错误（4xx、5xx）（需要替换 $namespace）",
	"http_5xx_errors":    "查询指定命名空间的服务端错误（5xx）（需要替换 $namespace）",
	"http_4xx_errors":    "查询指定命名空间的客户端错误（4xx）（需要替换 $namespace）",
	"http_slow_requests": "查询指定命名空间的慢请求日志（需要替换 $namespace）",
	"http_get_requests":  "查询指定命名空间的 GET 请求日志（需要替换 $namespace）",
	"http_post_requests": "查询指定命名空间的 POST 请求日志（需要替换 $namespace）",

	// 健康检查日志
	"health_check_logs":     "查询指定命名空间的健康检查日志（需要替换 $namespace）",
	"health_check_errors":   "查询指定命名空间的健康检查错误（需要替换 $namespace）",
	"health_check_logs_all": "查询所有命名空间的健康检查日志",

	// 按来源过滤
	"recent_logs":     "查询指定命名空间最近1小时的日志（需要替换 $namespace）",
	"stdout_logs":     "查询标准输出日志",
	"stderr_logs":     "查询标准错误日志",
	"stdout_logs_app": "查询指定应用的标准输出（需要替换 $app）",
	"stderr_logs_app": "查询指定应用的标准错误（需要替换 $app）",

	// 统计查询
	"log_count_by_app":       "按应用统计指定命名空间的日志数量（需要替换 $namespace）",
	"log_count_by_pod":       "按 Pod 统计指定命名空间的日志数量（需要替换 $namespace）",
	"log_count_by_container": "按容器统计指定命名空间的日志数量（需要替换 $namespace）",
	"log_count_by_node":      "按节点统计指定命名空间的日志数量（需要替换 $namespace）",
	"error_rate_by_app":      "按应用统计指定命名空间的错误率（需要替换 $namespace）",
	"request_rate_by_app":    "按应用统计指定命名空间的请求速率（需要替换 $namespace）",

	// 高级查询
	"parsed_json_logs":    "解析指定命名空间的 JSON 格式日志（需要替换 $namespace）",
	"filtered_level_logs": "按日志级别过滤指定命名空间的日志（需要替换 $namespace）",
	"request_duration":    "提取指定命名空间的请求耗时信息（需要替换 $namespace）",

	// 多标签组合
	"app_in_namespace": "查询指定命名空间中指定应用的日志（需要替换 $namespace 和 $app）",
	"pod_in_namespace": "查询指定命名空间中指定 Pod 的日志（需要替换 $namespace 和 $pod）",
	"container_in_pod": "查询指定 Pod 中指定容器的日志（需要替换 $pod 和 $container）",
}

// MetricDescriptions 指标描述信息（从 metric.json 导入）
var MetricDescriptions = map[string]string{
	"cpu_usage_active":        "CPU使用率（单位：%）",
	"cpu_usage_idle":          "CPU空闲率（单位：%）",
	"cpu_usage_iowait":        "CPU等待I/O的时间占比（单位：%）",
	"cpu_usage_system":        "CPU内核态时间占比（单位：%）",
	"cpu_usage_user":          "CPU用户态时间占比（单位：%）",
	"mem_total":               "内存总数",
	"mem_used":                "已用内存数",
	"mem_free":                "空闲内存数",
	"mem_available":           "应用程序可用内存数",
	"mem_used_percent":        "已用内存数百分比(0~100)",
	"mem_available_percent":   "内存剩余百分比(0~100)",
	"disk_total":              "硬盘分区总量（单位：byte）",
	"disk_used":               "硬盘分区使用量（单位：byte）",
	"disk_free":               "硬盘分区剩余量（单位：byte）",
	"disk_used_percent":       "硬盘分区使用率（单位：%）",
	"net_bytes_recv":          "网卡收包总数(bytes)",
	"net_bytes_sent":          "网卡发包总数(bytes)",
	"system_load1":            "1分钟平均load值",
	"system_load5":            "5分钟平均load值",
	"system_load15":           "15分钟平均load值",
	"processes_total":         "总进程数",
	"processes_running":       "运行中的进程数('R')",
	"processes_zombies":       "僵尸态进程数('Z')",
	"netstat_tcp_established": "ESTABLISHED状态的网络链接数",
	"netstat_tcp_time_wait":   "TIME_WAIT状态的网络链接数",
}
