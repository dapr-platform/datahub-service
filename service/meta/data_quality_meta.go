/*
 * @module service/meta/data_quality_meta
 * @description 数据质量相关元数据定义，包括规则类型、检查状态、脱敏类型等常量
 * @architecture 元数据层
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow 静态元数据定义
 * @rules 提供标准化的数据质量元数据定义，确保系统一致性
 * @dependencies 无
 * @refs service/models/governance.go, service/models/quality_models.go
 */

package meta

// QualityRuleType 数据质量规则类型定义
type QualityRuleType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// QualityRuleTypes 数据质量规则类型元数据
var QualityRuleTypes = []QualityRuleType{
	{
		Code:        "completeness",
		Name:        "完整性",
		Description: "检查数据是否完整，字段是否为空或缺失",
		Category:    "基础质量",
	},
	{
		Code:        "accuracy",
		Name:        "准确性",
		Description: "检查数据是否正确反映真实世界或验证点",
		Category:    "基础质量",
	},
	{
		Code:        "consistency",
		Name:        "一致性",
		Description: "检查跨源表/接口之间的数据是否一致",
		Category:    "基础质量",
	},
	{
		Code:        "validity",
		Name:        "有效性",
		Description: "检查数据格式、类型、范围是否符合规范",
		Category:    "基础质量",
	},
	{
		Code:        "uniqueness",
		Name:        "唯一性",
		Description: "检查主键或自然键是否重复",
		Category:    "基础质量",
	},
	{
		Code:        "timeliness",
		Name:        "及时性",
		Description: "检查数据刷新/更新频率，是否延迟超期",
		Category:    "基础质量",
	},
	{
		Code:        "standardization",
		Name:        "标准化",
		Description: "检查数据格式是否符合标准化要求",
		Category:    "清洗规则",
	},
}

// QualityCheckStatus 质量检查状态定义
type QualityCheckStatus struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// QualityCheckStatuses 质量检查状态元数据
var QualityCheckStatuses = []QualityCheckStatus{
	{
		Code:        "running",
		Name:        "执行中",
		Description: "质量检查正在执行",
		Color:       "#1890ff",
	},
	{
		Code:        "passed",
		Name:        "通过",
		Description: "质量检查通过",
		Color:       "#52c41a",
	},
	{
		Code:        "failed",
		Name:        "失败",
		Description: "质量检查失败",
		Color:       "#f5222d",
	},
	{
		Code:        "warning",
		Name:        "警告",
		Description: "质量检查有警告",
		Color:       "#fa8c16",
	},
}

// DataMaskingType 数据脱敏类型定义
type DataMaskingType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// DataMaskingTypes 数据脱敏类型元数据
var DataMaskingTypes = []DataMaskingType{
	{
		Code:        "mask",
		Name:        "掩码",
		Description: "部分字符用*替换",
		Example:     "138****8888",
	},
	{
		Code:        "replace",
		Name:        "替换",
		Description: "用固定值替换敏感数据",
		Example:     "***替换***",
	},
	{
		Code:        "encrypt",
		Name:        "加密",
		Description: "使用加密算法加密敏感数据",
		Example:     "AES加密后的值",
	},
	{
		Code:        "pseudonymize",
		Name:        "假名化",
		Description: "用假名替换真实标识符",
		Example:     "用户A、用户B",
	},
}

// CleansingRuleType 清洗规则类型定义
type CleansingRuleType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// CleansingRuleTypes 清洗规则类型元数据
var CleansingRuleTypes = []CleansingRuleType{
	{
		Code:        "standardization",
		Name:        "标准化",
		Description: "规范字符串格式、统一大小写、日期格式等",
		Category:    "格式规范",
	},
	{
		Code:        "deduplication",
		Name:        "去重",
		Description: "去除重复记录",
		Category:    "数据清理",
	},
	{
		Code:        "validation",
		Name:        "验证",
		Description: "验证数据有效性",
		Category:    "数据验证",
	},
	{
		Code:        "transformation",
		Name:        "转换",
		Description: "数据类型或格式转换",
		Category:    "数据转换",
	},
	{
		Code:        "enrichment",
		Name:        "丰富",
		Description: "补充缺失数据或增强数据",
		Category:    "数据增强",
	},
}

// QualityMetricType 质量指标类型定义
type QualityMetricType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

// QualityMetricTypes 质量指标类型元数据
var QualityMetricTypes = []QualityMetricType{
	{
		Code:        "completeness",
		Name:        "完整性指标",
		Description: "非空值占总记录数的比例",
		Unit:        "百分比",
	},
	{
		Code:        "accuracy",
		Name:        "准确性指标",
		Description: "准确值占总记录数的比例",
		Unit:        "百分比",
	},
	{
		Code:        "consistency",
		Name:        "一致性指标",
		Description: "一致值占总记录数的比例",
		Unit:        "百分比",
	},
	{
		Code:        "validity",
		Name:        "有效性指标",
		Description: "有效值占总记录数的比例",
		Unit:        "百分比",
	},
	{
		Code:        "uniqueness",
		Name:        "唯一性指标",
		Description: "唯一值占总记录数的比例",
		Unit:        "百分比",
	},
	{
		Code:        "timeliness",
		Name:        "及时性指标",
		Description: "及时更新的记录占总记录数的比例",
		Unit:        "百分比",
	},
}

// QualityReportType 质量报告类型定义
type QualityReportType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// QualityReportTypes 质量报告类型元数据
var QualityReportTypes = []QualityReportType{
	{
		Code:        "daily",
		Name:        "日报",
		Description: "每日质量报告",
	},
	{
		Code:        "weekly",
		Name:        "周报",
		Description: "每周质量报告",
	},
	{
		Code:        "monthly",
		Name:        "月报",
		Description: "每月质量报告",
	},
	{
		Code:        "ad_hoc",
		Name:        "临时报告",
		Description: "按需生成的质量报告",
	},
}

// QualityIssueSeverity 质量问题严重程度定义
type QualityIssueSeverity struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Priority    int    `json:"priority"`
}

// QualityIssueSeverities 质量问题严重程度元数据
var QualityIssueSeverities = []QualityIssueSeverity{
	{
		Code:        "low",
		Name:        "低",
		Description: "轻微质量问题，不影响业务",
		Color:       "#52c41a",
		Priority:    1,
	},
	{
		Code:        "medium",
		Name:        "中",
		Description: "中等质量问题，可能影响业务",
		Color:       "#fa8c16",
		Priority:    2,
	},
	{
		Code:        "high",
		Name:        "高",
		Description: "严重质量问题，影响业务正常运行",
		Color:       "#f5222d",
		Priority:    3,
	},
	{
		Code:        "critical",
		Name:        "严重",
		Description: "严重质量问题，严重影响业务",
		Color:       "#a61d24",
		Priority:    4,
	},
}

// QualityIssueStatus 质量问题状态定义
type QualityIssueStatus struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// QualityIssueStatuses 质量问题状态元数据
var QualityIssueStatuses = []QualityIssueStatus{
	{
		Code:        "open",
		Name:        "开放",
		Description: "问题已发现，待处理",
		Color:       "#f5222d",
	},
	{
		Code:        "investigating",
		Name:        "调查中",
		Description: "问题正在调查分析",
		Color:       "#1890ff",
	},
	{
		Code:        "resolved",
		Name:        "已解决",
		Description: "问题已解决",
		Color:       "#52c41a",
	},
	{
		Code:        "ignored",
		Name:        "已忽略",
		Description: "问题被标记为忽略",
		Color:       "#8c8c8c",
	},
	{
		Code:        "false_positive",
		Name:        "误报",
		Description: "问题被标记为误报",
		Color:       "#722ed1",
	},
}

// GetQualityRuleTypes 获取质量规则类型列表
func GetQualityRuleTypes() []QualityRuleType {
	return QualityRuleTypes
}

// GetQualityCheckStatuses 获取质量检查状态列表
func GetQualityCheckStatuses() []QualityCheckStatus {
	return QualityCheckStatuses
}

// GetDataMaskingTypes 获取数据脱敏类型列表
func GetDataMaskingTypes() []DataMaskingType {
	return DataMaskingTypes
}

// GetCleansingRuleTypes 获取清洗规则类型列表
func GetCleansingRuleTypes() []CleansingRuleType {
	return CleansingRuleTypes
}

// GetQualityMetricTypes 获取质量指标类型列表
func GetQualityMetricTypes() []QualityMetricType {
	return QualityMetricTypes
}

// GetQualityReportTypes 获取质量报告类型列表
func GetQualityReportTypes() []QualityReportType {
	return QualityReportTypes
}

// GetQualityIssueSeverities 获取质量问题严重程度列表
func GetQualityIssueSeverities() []QualityIssueSeverity {
	return QualityIssueSeverities
}

// GetQualityIssueStatuses 获取质量问题状态列表
func GetQualityIssueStatuses() []QualityIssueStatus {
	return QualityIssueStatuses
}
