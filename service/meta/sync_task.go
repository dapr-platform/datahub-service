package meta

import "time"

// 同步任务类型常量
const (
	// 新的任务类型（符合需求）
	SyncTaskTypeBatchSync    = "batch_sync"    // 批量同步（根据接口配置自动判断全量/增量）
	SyncTaskTypeRealtimeSync = "realtime_sync" // 实时同步

)

var SyncTaskTypes = []MetaField{
	{
		Name:         "batch_sync",
		DisplayName:  "批量同步",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "realtime_sync",
		DisplayName:  "实时同步",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
}

// 同步任务状态常量
const (
	SyncTaskStatusPending   = "pending"   // 待执行
	SyncTaskStatusRunning   = "running"   // 运行中
	SyncTaskStatusSuccess   = "success"   // 成功
	SyncTaskStatusFailed    = "failed"    // 失败
	SyncTaskStatusCancelled = "cancelled" // 已取消
)

// 同步任务执行时机常量
const (
	SyncTaskTriggerManual   = "manual"   // 手动执行
	SyncTaskTriggerOnce     = "once"     // 单次执行
	SyncTaskTriggerInterval = "interval" // 间隔执行
	SyncTaskTriggerCron     = "cron"     // Cron表达式执行
)

// 同步任务执行记录状态常量
const (
	SyncExecutionStatusRunning   = "running"   // 运行中
	SyncExecutionStatusSuccess   = "success"   // 成功
	SyncExecutionStatusFailed    = "failed"    // 失败
	SyncExecutionStatusCancelled = "cancelled" // 已取消
)

var SyncTaskStatuses = []MetaField{
	{
		Name:         "pending",
		DisplayName:  "待执行",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "running",
		DisplayName:  "执行中",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "success",
		DisplayName:  "成功",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "failed",
		DisplayName:  "失败",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "cancelled",
		DisplayName:  "取消",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
}

// 调度类型常量
const (
	SyncTaskScheduleTypeCron     = "cron"
	SyncTaskScheduleTypeInterval = "interval"
	SyncTaskScheduleTypeOnce     = "once"
	SyncTaskScheduleTypeManual   = "manual"
)

var SyncTaskScheduleTypes = []MetaField{
	{
		Name:         "cron",
		DisplayName:  "Cron",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "interval",
		DisplayName:  "Interval",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "once",
		DisplayName:  "Once",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "manual",
		DisplayName:  "Manual",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
}

// 事件类型常量
const (
	SyncEventTypeStart    = "start"
	SyncEventTypeProgress = "progress"
	SyncEventTypeComplete = "complete"
	SyncEventTypeError    = "error"
	SyncEventTypePause    = "pause"
	SyncEventTypeResume   = "resume"
	SyncEventTypeCancel   = "cancel"
)

var SyncEventTypes = []MetaField{
	{
		Name:         "start",
		DisplayName:  "开始",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "progress",
		DisplayName:  "进度",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "complete",
		DisplayName:  "完成",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "error",
		DisplayName:  "错误",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "pause",
		DisplayName:  "暂停",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "resume",
		DisplayName:  "恢复",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
	{
		Name:         "cancel",
		DisplayName:  "取消",
		Type:         "string",
		Required:     true,
		DefaultValue: "",
	},
}

// 复杂度级别常量
const (
	ComplexityLow    = "low"
	ComplexityMedium = "medium"
	ComplexityHigh   = "high"
)

// 调度配置字段常量
const (
	SyncTaskScheduleFieldRetryTimes        = "retry_times"
	SyncTaskScheduleFieldTimeoutSec        = "timeout_sec"
	SyncTaskScheduleFieldInterval          = "interval"
	SyncTaskScheduleFieldIntervalUnit      = "interval_unit"
	SyncTaskScheduleFieldStartTime         = "start_time"
	SyncTaskScheduleFieldCronExpression    = "cron_expression"
	SyncTaskScheduleFieldRetryIntervalSec  = "retry_interval_sec"
	SyncTaskScheduleFieldRetryIntervalUnit = "retry_interval_unit"
)

// 任务状态验证函数
func IsValidTaskStatus(status string) bool {
	validStatuses := map[string]bool{
		SyncTaskStatusPending:   true,
		SyncTaskStatusRunning:   true,
		SyncTaskStatusSuccess:   true,
		SyncTaskStatusFailed:    true,
		SyncTaskStatusCancelled: true,
	}
	return validStatuses[status]
}

// IsValidSyncTaskTrigger 验证同步任务执行时机是否有效
func IsValidSyncTaskTrigger(trigger string) bool {
	validTriggers := map[string]bool{
		SyncTaskTriggerManual:   true,
		SyncTaskTriggerOnce:     true,
		SyncTaskTriggerInterval: true,
		SyncTaskTriggerCron:     true,
	}
	return validTriggers[trigger]
}

// IsValidSyncExecutionStatus 验证同步执行记录状态是否有效
func IsValidSyncExecutionStatus(status string) bool {
	validStatuses := map[string]bool{
		SyncExecutionStatusRunning:   true,
		SyncExecutionStatusSuccess:   true,
		SyncExecutionStatusFailed:    true,
		SyncExecutionStatusCancelled: true,
	}
	return validStatuses[status]
}

// 同步类型验证函数
func IsValidSyncType(syncType string) bool {
	validTypes := map[string]bool{
		// 新类型
		SyncTaskTypeBatchSync:    true,
		SyncTaskTypeRealtimeSync: true,
	}
	return validTypes[syncType]
}

// 调度类型验证函数
func IsValidScheduleType(scheduleType string) bool {
	validTypes := map[string]bool{
		SyncTaskScheduleTypeCron:     true,
		SyncTaskScheduleTypeInterval: true,
		SyncTaskScheduleTypeOnce:     true,
		SyncTaskScheduleTypeManual:   true,
	}
	return validTypes[scheduleType]
}

// 获取可删除的任务状态
func GetDeletableTaskStatuses() []string {
	return []string{
		SyncTaskStatusSuccess,
		SyncTaskStatusFailed,
		SyncTaskStatusCancelled,
		SyncTaskStatusPending,
	}
}

// 获取可取消的任务状态
func GetCancellableTaskStatuses() []string {
	return []string{
		SyncTaskStatusPending,
		SyncTaskStatusRunning,
	}
}

// 获取可重试的任务状态
func GetRetryableTaskStatuses() []string {
	return []string{
		SyncTaskStatusFailed,
	}
}

// 获取可更新配置的任务状态
func GetUpdatableTaskStatuses() []string {
	return []string{
		SyncTaskStatusPending,
		SyncTaskStatusFailed,
	}
}

// 任务状态流转验证
func CanTransitionStatus(from, to string) bool {
	allowedTransitions := map[string][]string{
		SyncTaskStatusPending: {
			SyncTaskStatusRunning,
			SyncTaskStatusCancelled,
		},
		SyncTaskStatusRunning: {
			SyncTaskStatusSuccess,
			SyncTaskStatusFailed,
			SyncTaskStatusCancelled,
		},
		SyncTaskStatusFailed: {
			SyncTaskStatusPending, // 重试
		},
	}

	if allowed, exists := allowedTransitions[from]; exists {
		for _, status := range allowed {
			if status == to {
				return true
			}
		}
	}
	return false
}

type SyncTaskScheduleDefinition struct {
	ScheduleType         string                                 `json:"schedule_type"`
	ScheduleConfigFields map[string]SyncTaskScheduleConfigField `json:"schedule_config_fields"`
}

type SyncTaskScheduleConfigField struct {
	Name         string      `json:"name"`
	DisplayName  string      `json:"display_name"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Description  string      `json:"description"`
}

var SyncTaskScheduleDefinitions map[string]SyncTaskScheduleDefinition
var SyncTaskMetas map[string][]MetaField

func init() {
	initSyncTaskScheduleDefinitions()
	initSyncTaskMetas()
}

func initSyncTaskMetas() {
	SyncTaskMetas = make(map[string][]MetaField)
	SyncTaskMetas["sync_task_types"] = SyncTaskTypes
	SyncTaskMetas["sync_task_statuses"] = SyncTaskStatuses
	SyncTaskMetas["sync_task_schedule_types"] = SyncTaskScheduleTypes
	SyncTaskMetas["sync_event_types"] = SyncEventTypes
}

func initSyncTaskScheduleDefinitions() {
	SyncTaskScheduleDefinitions = make(map[string]SyncTaskScheduleDefinition)
	cronDefinition := SyncTaskScheduleDefinition{
		ScheduleType: SyncTaskScheduleTypeCron,
		ScheduleConfigFields: map[string]SyncTaskScheduleConfigField{
			SyncTaskScheduleFieldCronExpression: {
				Name:         "cron_expression",
				DisplayName:  "Cron Expression",
				Type:         "string",
				Required:     true,
				DefaultValue: "",
				Description:  "Cron Expression",
			},
			SyncTaskScheduleFieldRetryTimes: {
				Name:         "retry_times",
				DisplayName:  "重试次数",
				Type:         "number",
				Required:     false,
				DefaultValue: 3,
				Description:  "重试次数",
			},
			SyncTaskScheduleFieldTimeoutSec: {
				Name:         "timeout_sec",
				DisplayName:  "超时时间(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: 300,
				Description:  "超时时间(秒)",
			},
		},
	}
	SyncTaskScheduleDefinitions[SyncTaskScheduleTypeCron] = cronDefinition
	intervalDefinition := SyncTaskScheduleDefinition{
		ScheduleType: SyncTaskScheduleTypeInterval,
		ScheduleConfigFields: map[string]SyncTaskScheduleConfigField{
			SyncTaskScheduleFieldInterval: {
				Name:         "interval",
				DisplayName:  "间隔",
				Type:         "number",
				Required:     true,
				DefaultValue: 1,
				Description:  "间隔",
			},
			SyncTaskScheduleFieldIntervalUnit: {
				Name:         "interval_unit",
				DisplayName:  "间隔单位",
				Type:         "string",
				Required:     false,
				DefaultValue: "seconds",
				Description:  "间隔单位",
			},
			SyncTaskScheduleFieldRetryTimes: {
				Name:         "retry_times",
				DisplayName:  "重试次数",
				Type:         "number",
				Required:     false,
				DefaultValue: 3,
				Description:  "重试次数",
			},
			SyncTaskScheduleFieldTimeoutSec: {
				Name:         "timeout_sec",
				DisplayName:  "超时时间(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: 300,
				Description:  "超时时间(秒)",
			},
		},
	}
	SyncTaskScheduleDefinitions[SyncTaskScheduleTypeInterval] = intervalDefinition
	onceDefinition := SyncTaskScheduleDefinition{
		ScheduleType: SyncTaskScheduleTypeOnce,
		ScheduleConfigFields: map[string]SyncTaskScheduleConfigField{
			SyncTaskScheduleFieldStartTime: {
				Name:         "start_time",
				DisplayName:  "Start Time",
				Type:         "datetime",
				Required:     true,
				DefaultValue: time.Now(),
				Description:  "Start Time",
			},
			SyncTaskScheduleFieldRetryTimes: {
				Name:         "retry_times",
				DisplayName:  "重试次数",
				Type:         "number",
				Required:     false,
				DefaultValue: 3,
				Description:  "重试次数",
			},
			SyncTaskScheduleFieldTimeoutSec: {
				Name:         "timeout_sec",
				DisplayName:  "超时时间(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: 300,
				Description:  "超时时间(秒)",
			},
		},
	}
	SyncTaskScheduleDefinitions[SyncTaskScheduleTypeOnce] = onceDefinition
	manualDefinition := SyncTaskScheduleDefinition{
		ScheduleType: SyncTaskScheduleTypeManual,
		ScheduleConfigFields: map[string]SyncTaskScheduleConfigField{
			SyncTaskScheduleFieldRetryTimes: {
				Name:         "retry_times",
				DisplayName:  "重试次数",
				Type:         "number",
				Required:     false,
				DefaultValue: 3,
				Description:  "重试次数",
			},
			SyncTaskScheduleFieldTimeoutSec: {
				Name:         "timeout_sec",
				DisplayName:  "超时时间(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: 300,
				Description:  "超时时间(秒)",
			},
		},
	}
	SyncTaskScheduleDefinitions[SyncTaskScheduleTypeManual] = manualDefinition
}
