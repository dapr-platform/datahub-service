/*
 * @module service/meta/monitoring
 * @description 监控系统元数据定义（简化版），仅包含基本的类型定义
 * @architecture 分层架构 - 元数据层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 常量定义 -> 业务逻辑使用
 * @rules 简化监控元数据，只保留必要的定义
 * @dependencies 无
 */

package meta

// 查询类型
type QueryType string

const (
	QueryTypeMetrics QueryType = "metrics" // 指标查询
	QueryTypeLogs    QueryType = "logs"    // 日志查询
)

// 时间范围
type TimeRange string

const (
	TimeRange1Hour  TimeRange = "1h"  // 1小时
	TimeRange6Hour  TimeRange = "6h"  // 6小时
	TimeRange24Hour TimeRange = "24h" // 24小时
	TimeRange7Day   TimeRange = "7d"  // 7天
	TimeRange30Day  TimeRange = "30d" // 30天
)

// 验证查询类型
func (q QueryType) IsValid() bool {
	switch q {
	case QueryTypeMetrics, QueryTypeLogs:
		return true
	default:
		return false
	}
}

// 解析时间范围为秒数
func (t TimeRange) ToSeconds() int64 {
	switch t {
	case TimeRange1Hour:
		return 3600
	case TimeRange6Hour:
		return 21600
	case TimeRange24Hour:
		return 86400
	case TimeRange7Day:
		return 604800
	case TimeRange30Day:
		return 2592000
	default:
		return 3600 // 默认1小时
	}
}
