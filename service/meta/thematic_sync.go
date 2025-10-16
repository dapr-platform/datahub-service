/*
 * @module service/meta/thematic_sync
 * @description 主题库数据同步相关的元数据定义，为前端提供同步任务的配置选项和状态信息
 * @architecture 元数据驱动架构 - 提供前端配置界面所需的元数据
 * @documentReference ai_docs/thematic_sync_design.md
 * @rules 遵循系统元数据标准，提供完整的前端配置支持
 * @dependencies 无外部依赖
 * @refs service/models/thematic_sync.go, service/meta/sync_task.go
 */

package meta

// 主题库同步任务状态常量
// 注意：任务状态只有三种，completed/failed 等状态属于执行记录(ThematicSyncExecution)
const (
	ThematicSyncTaskStatusDraft  = "draft"  // 草稿 - 正在编辑，尚未激活
	ThematicSyncTaskStatusActive = "active" // 激活 - 可以被调度执行
	ThematicSyncTaskStatusPaused = "paused" // 暂停 - 不会被调度执行
)

// 主题库同步任务触发类型常量
const (
	ThematicSyncTriggerManual   = "manual"   // 手动触发
	ThematicSyncTriggerOnce     = "once"     // 单次执行
	ThematicSyncTriggerInterval = "interval" // 间隔执行
	ThematicSyncTriggerCron     = "cron"     // Cron表达式
)

// 主题库同步执行状态常量
const (
	ThematicSyncExecutionStatusPending   = "pending"   // 待执行
	ThematicSyncExecutionStatusRunning   = "running"   // 运行中
	ThematicSyncExecutionStatusSuccess   = "success"   // 成功
	ThematicSyncExecutionStatusFailed    = "failed"    // 失败
	ThematicSyncExecutionStatusCancelled = "cancelled" // 取消
)

// 主题库同步执行类型常量
const (
	ThematicSyncExecutionTypeManual    = "manual"    // 手动执行
	ThematicSyncExecutionTypeScheduled = "scheduled" // 计划执行
	ThematicSyncExecutionTypeRetry     = "retry"     // 重试执行
)

// ThematicSyncTaskStatus 主题库同步任务状态元数据
var ThematicSyncTaskStatuses = []MetaField{
	{
		Name:        "draft",
		DisplayName: "草稿",
		Type:        "string",
		Description: "任务处于草稿状态，正在编辑，尚未激活",
	},
	{
		Name:        "active",
		DisplayName: "激活",
		Type:        "string",
		Description: "任务已激活，可以被调度执行或手动执行",
	},
	{
		Name:        "paused",
		DisplayName: "暂停",
		Type:        "string",
		Description: "任务已暂停，不会被调度执行，但可以手动执行",
	},
}

// ThematicSyncTriggerTypes 主题库同步触发类型元数据
var ThematicSyncTriggerTypes = []MetaField{
	{
		Name:        "manual",
		DisplayName: "手动触发",
		Type:        "string",
		Description: "手动点击执行按钮触发同步",
	},
	{
		Name:        "once",
		DisplayName: "单次执行",
		Type:        "string",
		Description: "在指定时间执行一次同步",
	},
	{
		Name:        "interval",
		DisplayName: "间隔执行",
		Type:        "string",
		Description: "按固定时间间隔重复执行同步",
	},
	{
		Name:        "cron",
		DisplayName: "Cron表达式",
		Type:        "string",
		Description: "使用Cron表达式定义复杂的调度规则",
	},
}

// ThematicSyncExecutionStatuses 主题库同步执行状态元数据
var ThematicSyncExecutionStatuses = []MetaField{
	{
		Name:        "pending",
		DisplayName: "待执行",
		Type:        "string",
		Description: "同步任务已提交，等待执行",
	},
	{
		Name:        "running",
		DisplayName: "运行中",
		Type:        "string",
		Description: "同步任务正在执行",
	},
	{
		Name:        "success",
		DisplayName: "成功",
		Type:        "string",
		Description: "同步任务执行成功",
	},
	{
		Name:        "failed",
		DisplayName: "失败",
		Type:        "string",
		Description: "同步任务执行失败",
	},
	{
		Name:        "cancelled",
		DisplayName: "取消",
		Type:        "string",
		Description: "同步任务被取消执行",
	},
}

// ThematicSyncExecutionTypes 主题库同步执行类型元数据
var ThematicSyncExecutionTypes = []MetaField{
	{
		Name:        "manual",
		DisplayName: "手动执行",
		Type:        "string",
		Description: "用户手动触发的执行",
	},
	{
		Name:        "scheduled",
		DisplayName: "计划执行",
		Type:        "string",
		Description: "系统按计划自动执行",
	},
	{
		Name:        "retry",
		DisplayName: "重试执行",
		Type:        "string",
		Description: "失败后重新执行",
	},
}

// ThematicSyncAggregationStrategies 主题库同步汇聚策略元数据
var ThematicSyncAggregationStrategies = []MetaField{
	{
		Name:        "merge",
		DisplayName: "合并汇聚",
		Type:        "string",
		Description: "将多个源的数据合并到一条记录",
	},
	{
		Name:        "union",
		DisplayName: "联合汇聚",
		Type:        "string",
		Description: "将多个源的数据联合保存为多条记录",
	},
	{
		Name:        "priority",
		DisplayName: "优先级汇聚",
		Type:        "string",
		Description: "根据优先级选择最优数据源",
	},
	{
		Name:        "latest",
		DisplayName: "最新优先",
		Type:        "string",
		Description: "选择时间戳最新的数据",
	},
}

// ThematicSyncKeyMatchingTypes 主题库同步主键匹配类型元数据
var ThematicSyncKeyMatchingTypes = []MetaField{
	{
		Name:        "exact",
		DisplayName: "精确匹配",
		Type:        "string",
		Description: "主键必须完全相同",
	},
	{
		Name:        "fuzzy",
		DisplayName: "模糊匹配",
		Type:        "string",
		Description: "使用相似度算法匹配主键",
	},
	{
		Name:        "composite",
		DisplayName: "复合匹配",
		Type:        "string",
		Description: "使用多个字段组合匹配",
	},
	{
		Name:        "hash",
		DisplayName: "哈希匹配",
		Type:        "string",
		Description: "使用字段哈希值匹配",
	},
}

// ThematicSyncCleansingRuleTypes 主题库同步清洗规则类型元数据
var ThematicSyncCleansingRuleTypes = []MetaField{
	{
		Name:        "null_handling",
		DisplayName: "空值处理",
		Type:        "string",
		Description: "处理空值和缺失数据",
	},
	{
		Name:        "format_standardization",
		DisplayName: "格式标准化",
		Type:        "string",
		Description: "统一数据格式",
	},
	{
		Name:        "duplicate_removal",
		DisplayName: "去重处理",
		Type:        "string",
		Description: "删除重复记录",
	},
	{
		Name:        "data_validation",
		DisplayName: "数据校验",
		Type:        "string",
		Description: "验证数据的有效性",
	},
	{
		Name:        "outlier_detection",
		DisplayName: "异常值检测",
		Type:        "string",
		Description: "识别和处理异常数据",
	},
}

// ThematicSyncPrivacyRuleTypes 主题库同步脱敏规则类型元数据
var ThematicSyncPrivacyRuleTypes = []MetaField{
	{
		Name:        "masking",
		DisplayName: "掩码脱敏",
		Type:        "string",
		Description: "使用*等字符遮盖敏感信息",
	},
	{
		Name:        "encryption",
		DisplayName: "加密脱敏",
		Type:        "string",
		Description: "对敏感数据进行加密处理",
	},
	{
		Name:        "hashing",
		DisplayName: "哈希脱敏",
		Type:        "string",
		Description: "使用哈希算法处理敏感数据",
	},
	{
		Name:        "generalization",
		DisplayName: "泛化脱敏",
		Type:        "string",
		Description: "将具体值替换为范围或分类",
	},
	{
		Name:        "suppression",
		DisplayName: "抑制脱敏",
		Type:        "string",
		Description: "完全删除敏感字段",
	},
}

// ThematicSyncQualityRuleTypes 主题库同步质量规则类型元数据
var ThematicSyncQualityRuleTypes = []MetaField{
	{
		Name:        "completeness",
		DisplayName: "完整性检查",
		Type:        "string",
		Description: "检查数据的完整性",
	},
	{
		Name:        "accuracy",
		DisplayName: "准确性检查",
		Type:        "string",
		Description: "验证数据的准确性",
	},
	{
		Name:        "consistency",
		DisplayName: "一致性检查",
		Type:        "string",
		Description: "检查数据的一致性",
	},
	{
		Name:        "uniqueness",
		DisplayName: "唯一性检查",
		Type:        "string",
		Description: "验证数据的唯一性",
	},
	{
		Name:        "timeliness",
		DisplayName: "时效性检查",
		Type:        "string",
		Description: "检查数据的时效性",
	},
}

// ThematicSyncMetas 主题库同步元数据集合
var ThematicSyncMetas = map[string]interface{}{
	"task_statuses":          ThematicSyncTaskStatuses,
	"trigger_types":          ThematicSyncTriggerTypes,
	"execution_statuses":     ThematicSyncExecutionStatuses,
	"execution_types":        ThematicSyncExecutionTypes,
	"aggregation_strategies": ThematicSyncAggregationStrategies,
	"key_matching_types":     ThematicSyncKeyMatchingTypes,
	"cleansing_rule_types":   ThematicSyncCleansingRuleTypes,
	"privacy_rule_types":     ThematicSyncPrivacyRuleTypes,
	"quality_rule_types":     ThematicSyncQualityRuleTypes,
}

// ThematicSyncConfigDefinition 主题库同步配置定义
type ThematicSyncConfigDefinition struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Category    string                    `json:"category"` // aggregation, key_matching, cleansing, privacy, quality
	Fields      []ThematicSyncConfigField `json:"fields"`
}

// ThematicSyncConfigField 主题库同步配置字段
type ThematicSyncConfigField struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"display_name"`
	Type         string            `json:"type"` // string, number, boolean, array, object, enum
	Required     bool              `json:"required"`
	DefaultValue interface{}       `json:"default_value,omitempty"`
	Description  string            `json:"description"`
	Options      []string          `json:"options,omitempty"`      // 用于enum类型
	Min          *float64          `json:"min,omitempty"`          // 用于number类型
	Max          *float64          `json:"max,omitempty"`          // 用于number类型
	Pattern      string            `json:"pattern,omitempty"`      // 用于string类型的正则验证
	Placeholder  string            `json:"placeholder,omitempty"`  // 前端显示的占位符
	HelpText     string            `json:"help_text,omitempty"`    // 帮助文本
	Group        string            `json:"group,omitempty"`        // 字段分组
	Dependencies []FieldDependency `json:"dependencies,omitempty"` // 字段依赖关系
}

// ThematicSyncConfigDefinitions 主题库同步配置定义集合
var ThematicSyncConfigDefinitions = map[string]ThematicSyncConfigDefinition{
	"aggregation_config": {
		ID:          "aggregation_config",
		Name:        "汇聚配置",
		Description: "配置多个数据源的汇聚策略和规则",
		Category:    "aggregation",
		Fields: []ThematicSyncConfigField{
			{
				Name:        "strategy",
				DisplayName: "汇聚策略",
				Type:        "enum",
				Required:    true,
				Options:     []string{"merge", "union", "priority", "latest"},
				Description: "选择数据汇聚的策略",
				HelpText:    "merge: 合并多个源的数据; union: 联合保存; priority: 按优先级选择; latest: 选择最新数据",
			},
			{
				Name:         "conflict_resolution",
				DisplayName:  "冲突解决方案",
				Type:         "enum",
				Required:     true,
				Options:      []string{"source_priority", "timestamp", "manual", "ignore"},
				DefaultValue: "source_priority",
				Description:  "当多个数据源存在冲突时的解决方案",
			},
			{
				Name:        "priority_rules",
				DisplayName: "优先级规则",
				Type:        "array",
				Required:    false,
				Description: "定义数据源的优先级顺序",
				Dependencies: []FieldDependency{
					{Field: "strategy", Condition: "equals", Value: "priority", Action: "require"},
				},
			},
		},
	},
	"key_matching_config": {
		ID:          "key_matching_config",
		Name:        "主键匹配配置",
		Description: "配置不同数据源之间的主键匹配规则",
		Category:    "key_matching",
		Fields: []ThematicSyncConfigField{
			{
				Name:        "matching_type",
				DisplayName: "匹配类型",
				Type:        "enum",
				Required:    true,
				Options:     []string{"exact", "fuzzy", "composite", "hash"},
				Description: "选择主键匹配的算法类型",
			},
			{
				Name:         "similarity_threshold",
				DisplayName:  "相似度阈值",
				Type:         "number",
				Required:     false,
				Min:          &[]float64{0.0}[0],
				Max:          &[]float64{1.0}[0],
				DefaultValue: 0.8,
				Description:  "模糊匹配时的相似度阈值",
				Dependencies: []FieldDependency{
					{Field: "matching_type", Condition: "equals", Value: "fuzzy", Action: "require"},
				},
			},
			{
				Name:        "composite_fields",
				DisplayName: "复合字段",
				Type:        "array",
				Required:    false,
				Description: "复合匹配时使用的字段列表",
				Dependencies: []FieldDependency{
					{Field: "matching_type", Condition: "equals", Value: "composite", Action: "require"},
				},
			},
		},
	},
	"cleansing_config": {
		ID:          "cleansing_config",
		Name:        "清洗配置",
		Description: "配置数据清洗和预处理规则",
		Category:    "cleansing",
		Fields: []ThematicSyncConfigField{
			{
				Name:        "enabled_rules",
				DisplayName: "启用的规则",
				Type:        "array",
				Required:    true,
				Description: "选择要启用的清洗规则类型",
			},
			{
				Name:         "null_handling_strategy",
				DisplayName:  "空值处理策略",
				Type:         "enum",
				Required:     false,
				Options:      []string{"skip", "default_value", "interpolation", "remove"},
				DefaultValue: "skip",
				Description:  "处理空值的策略",
			},
			{
				Name:        "validation_rules",
				DisplayName: "校验规则",
				Type:        "object",
				Required:    false,
				Description: "自定义的数据校验规则",
			},
		},
	},
	"privacy_config": {
		ID:          "privacy_config",
		Name:        "脱敏配置",
		Description: "配置敏感数据的脱敏处理规则",
		Category:    "privacy",
		Fields: []ThematicSyncConfigField{
			{
				Name:        "sensitive_fields",
				DisplayName: "敏感字段",
				Type:        "array",
				Required:    true,
				Description: "标识需要脱敏处理的字段",
			},
			{
				Name:        "masking_rules",
				DisplayName: "脱敏规则",
				Type:        "object",
				Required:    true,
				Description: "定义各字段的脱敏处理方式",
			},
			{
				Name:         "preserve_format",
				DisplayName:  "保持格式",
				Type:         "boolean",
				Required:     false,
				DefaultValue: true,
				Description:  "脱敏后是否保持原始数据格式",
			},
		},
	},
	"quality_config": {
		ID:          "quality_config",
		Name:        "质量配置",
		Description: "配置数据质量检查和评估规则",
		Category:    "quality",
		Fields: []ThematicSyncConfigField{
			{
				Name:        "quality_dimensions",
				DisplayName: "质量维度",
				Type:        "array",
				Required:    true,
				Description: "选择要检查的数据质量维度",
			},
			{
				Name:         "quality_threshold",
				DisplayName:  "质量阈值",
				Type:         "number",
				Required:     false,
				Min:          &[]float64{0.0}[0],
				Max:          &[]float64{100.0}[0],
				DefaultValue: 80.0,
				Description:  "数据质量的最低要求分数",
			},
			{
				Name:         "auto_fix",
				DisplayName:  "自动修复",
				Type:         "boolean",
				Required:     false,
				DefaultValue: false,
				Description:  "是否自动修复检测到的质量问题",
			},
		},
	},
}

// IsValidThematicSyncTaskStatus 验证主题库同步任务状态是否有效
func IsValidThematicSyncTaskStatus(status string) bool {
	validStatuses := []string{
		ThematicSyncTaskStatusDraft,
		ThematicSyncTaskStatusActive,
		ThematicSyncTaskStatusPaused,
	}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// IsValidThematicSyncTriggerType 验证主题库同步触发类型是否有效
func IsValidThematicSyncTriggerType(triggerType string) bool {
	validTypes := []string{
		ThematicSyncTriggerManual,
		ThematicSyncTriggerOnce,
		ThematicSyncTriggerInterval,
		ThematicSyncTriggerCron,
	}
	for _, validType := range validTypes {
		if triggerType == validType {
			return true
		}
	}
	return false
}

// IsValidThematicSyncExecutionStatus 验证主题库同步执行状态是否有效
func IsValidThematicSyncExecutionStatus(status string) bool {
	validStatuses := []string{
		ThematicSyncExecutionStatusPending,
		ThematicSyncExecutionStatusRunning,
		ThematicSyncExecutionStatusSuccess,
		ThematicSyncExecutionStatusFailed,
		ThematicSyncExecutionStatusCancelled,
	}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// GetThematicSyncTaskStatusDisplayName 获取主题库同步任务状态的显示名称
func GetThematicSyncTaskStatusDisplayName(status string) string {
	for _, field := range ThematicSyncTaskStatuses {
		if field.Name == status {
			return field.DisplayName
		}
	}
	return status
}

// GetThematicSyncTriggerTypeDisplayName 获取主题库同步触发类型的显示名称
func GetThematicSyncTriggerTypeDisplayName(triggerType string) string {
	for _, field := range ThematicSyncTriggerTypes {
		if field.Name == triggerType {
			return field.DisplayName
		}
	}
	return triggerType
}

// GetThematicSyncExecutionStatusDisplayName 获取主题库同步执行状态的显示名称
func GetThematicSyncExecutionStatusDisplayName(status string) string {
	for _, field := range ThematicSyncExecutionStatuses {
		if field.Name == status {
			return field.DisplayName
		}
	}
	return status
}
