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

// === 新增数据治理元数据类型 ===

// DataGovernanceObjectType 数据治理对象类型定义
type DataGovernanceObjectType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DataGovernanceObjectTypes 数据治理对象类型元数据
var DataGovernanceObjectTypes = []DataGovernanceObjectType{
	{
		Code:        "interface",
		Name:        "数据接口",
		Description: "基础数据接口",
	},
	{
		Code:        "thematic_interface",
		Name:        "主题接口",
		Description: "主题数据接口",
	},
	{
		Code:        "table",
		Name:        "数据表",
		Description: "数据库表",
	},
}

// RuleTemplateCategory 规则模板分类定义
type RuleTemplateCategory struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RuleTemplateCategories 规则模板分类元数据
var RuleTemplateCategories = []RuleTemplateCategory{
	{
		Code:        "basic_quality",
		Name:        "基础质量",
		Description: "基础数据质量检查规则",
	},
	{
		Code:        "data_cleansing",
		Name:        "数据清洗",
		Description: "数据清洗和标准化规则",
	},
	{
		Code:        "data_validation",
		Name:        "数据校验",
		Description: "数据有效性校验规则",
	},
	{
		Code:        "data_security",
		Name:        "数据安全",
		Description: "数据脱敏和安全规则",
	},
	{
		Code:        "data_transformation",
		Name:        "数据转换",
		Description: "数据转换和处理规则",
	},
}

// MaskingTemplateCategory 脱敏模板分类定义
type MaskingTemplateCategory struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MaskingTemplateCategories 脱敏模板分类元数据
var MaskingTemplateCategories = []MaskingTemplateCategory{
	{
		Code:        "personal_info",
		Name:        "个人信息",
		Description: "姓名、身份证、手机号等个人信息脱敏",
	},
	{
		Code:        "financial",
		Name:        "金融信息",
		Description: "银行卡号、账户信息等金融数据脱敏",
	},
	{
		Code:        "medical",
		Name:        "医疗信息",
		Description: "病历、诊断信息等医疗数据脱敏",
	},
	{
		Code:        "business",
		Name:        "商业信息",
		Description: "商业机密、合同信息等商业数据脱敏",
	},
	{
		Code:        "custom",
		Name:        "自定义",
		Description: "用户自定义脱敏规则",
	},
}

// SecurityLevel 安全级别定义
type SecurityLevel struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

// SecurityLevels 安全级别元数据
var SecurityLevels = []SecurityLevel{
	{
		Code:        "low",
		Name:        "低",
		Description: "低安全级别，基础脱敏处理",
		Priority:    1,
	},
	{
		Code:        "medium",
		Name:        "中",
		Description: "中等安全级别，标准脱敏处理",
		Priority:    2,
	},
	{
		Code:        "high",
		Name:        "高",
		Description: "高安全级别，严格脱敏处理",
		Priority:    3,
	},
	{
		Code:        "critical",
		Name:        "严重",
		Description: "严重安全级别，最严格脱敏处理",
		Priority:    4,
	},
}

// ComplexityLevel 复杂度级别定义
type ComplexityLevel struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ComplexityLevels 复杂度级别元数据
var ComplexityLevels = []ComplexityLevel{
	{
		Code:        "low",
		Name:        "简单",
		Description: "简单规则，易于配置和理解",
	},
	{
		Code:        "medium",
		Name:        "中等",
		Description: "中等复杂度规则，需要一定配置经验",
	},
	{
		Code:        "high",
		Name:        "复杂",
		Description: "复杂规则，需要专业知识配置",
	},
}

// MetadataType 元数据类型定义
type MetadataType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MetadataTypes 元数据类型元数据
var MetadataTypes = []MetadataType{
	{
		Code:        "technical",
		Name:        "技术元数据",
		Description: "描述数据的技术特征，如字段类型、长度、约束等",
	},
	{
		Code:        "business",
		Name:        "业务元数据",
		Description: "描述数据的业务含义，如业务规则、计算逻辑等",
	},
	{
		Code:        "management",
		Name:        "管理元数据",
		Description: "描述数据的管理信息，如负责人、更新频率等",
	},
}

// TaskType 任务类型定义
type TaskType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TaskTypes 任务类型元数据
var TaskTypes = []TaskType{
	{
		Code:        "scheduled",
		Name:        "定时任务",
		Description: "按照设定的时间计划自动执行",
	},
	{
		Code:        "manual",
		Name:        "手动任务",
		Description: "需要人工手动触发执行",
	},
	{
		Code:        "realtime",
		Name:        "实时任务",
		Description: "数据变化时实时触发执行",
	},
}

// TaskStatus 任务状态定义
type TaskStatus struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// TaskStatuses 任务状态元数据
var TaskStatuses = []TaskStatus{
	{
		Code:        "pending",
		Name:        "待执行",
		Description: "任务已创建，等待执行",
		Color:       "#1890ff",
	},
	{
		Code:        "running",
		Name:        "执行中",
		Description: "任务正在执行",
		Color:       "#52c41a",
	},
	{
		Code:        "completed",
		Name:        "已完成",
		Description: "任务执行完成",
		Color:       "#722ed1",
	},
	{
		Code:        "failed",
		Name:        "执行失败",
		Description: "任务执行失败",
		Color:       "#f5222d",
	},
	{
		Code:        "cancelled",
		Name:        "已取消",
		Description: "任务被取消",
		Color:       "#8c8c8c",
	},
}

// TransformationRuleType 转换规则类型定义
type TransformationRuleType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// TransformationRuleTypes 转换规则类型元数据
var TransformationRuleTypes = []TransformationRuleType{
	{
		Code:        "format",
		Name:        "格式转换",
		Description: "改变数据格式，如日期格式、字符串格式等",
		Category:    "数据格式",
	},
	{
		Code:        "calculate",
		Name:        "计算转换",
		Description: "通过计算生成新的数据值",
		Category:    "数据计算",
	},
	{
		Code:        "aggregate",
		Name:        "聚合转换",
		Description: "对多条记录进行聚合计算",
		Category:    "数据聚合",
	},
	{
		Code:        "filter",
		Name:        "过滤转换",
		Description: "根据条件过滤数据",
		Category:    "数据过滤",
	},
	{
		Code:        "join",
		Name:        "关联转换",
		Description: "关联多个数据源",
		Category:    "数据关联",
	},
}

// ValidationRuleType 校验规则类型定义
type ValidationRuleType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// ValidationRuleTypes 校验规则类型元数据
var ValidationRuleTypes = []ValidationRuleType{
	{
		Code:        "format",
		Name:        "格式校验",
		Description: "校验数据格式是否符合要求",
		Category:    "格式校验",
	},
	{
		Code:        "range",
		Name:        "范围校验",
		Description: "校验数值是否在指定范围内",
		Category:    "数值校验",
	},
	{
		Code:        "enum",
		Name:        "枚举校验",
		Description: "校验值是否在枚举列表中",
		Category:    "枚举校验",
	},
	{
		Code:        "regex",
		Name:        "正则校验",
		Description: "使用正则表达式校验数据格式",
		Category:    "格式校验",
	},
	{
		Code:        "custom",
		Name:        "自定义校验",
		Description: "使用自定义逻辑校验数据",
		Category:    "自定义校验",
	},
	{
		Code:        "reference",
		Name:        "引用校验",
		Description: "校验数据是否在引用表中存在",
		Category:    "引用校验",
	},
}

// LineageRelationType 血缘关系类型定义
type LineageRelationType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// LineageRelationTypes 血缘关系类型元数据
var LineageRelationTypes = []LineageRelationType{
	{
		Code:        "direct",
		Name:        "直接关系",
		Description: "数据直接从源到目标",
	},
	{
		Code:        "derived",
		Name:        "派生关系",
		Description: "目标数据由源数据派生而来",
	},
	{
		Code:        "aggregated",
		Name:        "聚合关系",
		Description: "目标数据是源数据的聚合结果",
	},
	{
		Code:        "transformed",
		Name:        "转换关系",
		Description: "目标数据是源数据的转换结果",
	},
}

// 获取元数据的函数
func GetDataGovernanceObjectTypes() []DataGovernanceObjectType {
	return DataGovernanceObjectTypes
}

func GetRuleTemplateCategories() []RuleTemplateCategory {
	return RuleTemplateCategories
}

func GetMaskingTemplateCategories() []MaskingTemplateCategory {
	return MaskingTemplateCategories
}

func GetSecurityLevels() []SecurityLevel {
	return SecurityLevels
}

func GetComplexityLevels() []ComplexityLevel {
	return ComplexityLevels
}

func GetMetadataTypes() []MetadataType {
	return MetadataTypes
}

func GetTaskTypes() []TaskType {
	return TaskTypes
}

func GetTaskStatuses() []TaskStatus {
	return TaskStatuses
}

func GetTransformationRuleTypes() []TransformationRuleType {
	return TransformationRuleTypes
}

func GetValidationRuleTypes() []ValidationRuleType {
	return ValidationRuleTypes
}

func GetLineageRelationTypes() []LineageRelationType {
	return LineageRelationTypes
}
