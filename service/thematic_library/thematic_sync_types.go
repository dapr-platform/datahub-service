/*
 * @module service/thematic_sync_types
 * @description 主题同步服务相关的请求和响应类型定义
 * @architecture 数据传输对象 - 定义API请求和响应结构
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow N/A
 * @rules 确保类型定义的一致性和可扩展性
 * @dependencies time, models包
 * @refs thematic_sync_service.go, models/thematic_sync.go
 */

package thematic_library

import (
	"datahub-service/service/models"
)

// CreateThematicSyncTaskRequest 创建主题同步任务请求
type CreateThematicSyncTaskRequest struct {
	ThematicLibraryID   string                `json:"thematic_library_id" binding:"required"`
	ThematicInterfaceID string                `json:"thematic_interface_id" binding:"required"`
	TaskName            string                `json:"task_name" binding:"required"`
	Description         string                `json:"description"`
	SourceLibraries     []SourceLibraryConfig `json:"source_libraries" binding:"required,min=1"`
	AggregationConfig   *AggregationConfig    `json:"aggregation_config,omitempty"`
	KeyMatchingRules    *KeyMatchingRules     `json:"key_matching_rules,omitempty"`
	FieldMappingRules   *FieldMappingRules    `json:"field_mapping_rules,omitempty"`
	CleansingRules      *CleansingRules       `json:"cleansing_rules,omitempty"`
	PrivacyRules        *PrivacyRules         `json:"privacy_rules,omitempty"`
	QualityRules        *QualityRules         `json:"quality_rules,omitempty"`
	ScheduleConfig      *ScheduleConfig       `json:"schedule_config" binding:"required"`
	CreatedBy           string                `json:"created_by" binding:"required"`
}

// UpdateThematicSyncTaskRequest 更新主题同步任务请求
type UpdateThematicSyncTaskRequest struct {
	TaskName          string             `json:"task_name"`
	Description       string             `json:"description"`
	Status            string             `json:"status"`
	ScheduleConfig    *ScheduleConfig    `json:"schedule_config,omitempty"`
	AggregationConfig *AggregationConfig `json:"aggregation_config,omitempty"`
	KeyMatchingRules  *KeyMatchingRules  `json:"key_matching_rules,omitempty"`
	FieldMappingRules *FieldMappingRules `json:"field_mapping_rules,omitempty"`
	CleansingRules    *CleansingRules    `json:"cleansing_rules,omitempty"`
	PrivacyRules      *PrivacyRules      `json:"privacy_rules,omitempty"`
	QualityRules      *QualityRules      `json:"quality_rules,omitempty"`
	UpdatedBy         string             `json:"updated_by" binding:"required"`
}

// ListSyncTasksRequest 获取同步任务列表请求
type ListSyncTasksRequest struct {
	ThematicLibraryID string `form:"thematic_library_id"`
	Status            string `form:"status"`
	TriggerType       string `form:"trigger_type"`
	Page              int    `form:"page,default=1"`
	PageSize          int    `form:"page_size,default=10"`
}

// ListSyncTasksResponse 获取同步任务列表响应
type ListSyncTasksResponse struct {
	Tasks    []models.ThematicSyncTask `json:"tasks"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

// ExecuteSyncTaskRequest 执行同步任务请求
type ExecuteSyncTaskRequest struct {
	ExecutionType string                `json:"execution_type,default=manual"`
	Options       *SyncExecutionOptions `json:"options,omitempty"`
}

// ListSyncExecutionsRequest 获取执行记录列表请求
type ListSyncExecutionsRequest struct {
	TaskID        string `form:"task_id"`
	Status        string `form:"status"`
	ExecutionType string `form:"execution_type"`
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"page_size,default=10"`
}

// ListSyncExecutionsResponse 获取执行记录列表响应
type ListSyncExecutionsResponse struct {
	Executions []models.ThematicSyncExecution `json:"executions"`
	Total      int64                          `json:"total"`
	Page       int                            `json:"page"`
	PageSize   int                            `json:"page_size"`
}

// ThematicSyncTaskStatistics 主题同步任务统计信息
type ThematicSyncTaskStatistics struct {
	Task               *models.ThematicSyncTask       `json:"task"`
	TotalExecutions    int64                          `json:"total_executions"`
	SuccessExecutions  int64                          `json:"success_executions"`
	FailedExecutions   int64                          `json:"failed_executions"`
	SuccessRate        float64                        `json:"success_rate"`
	AverageProcessTime int64                          `json:"average_process_time"` // 处理时长（秒）
	TotalProcessedRows int64                          `json:"total_processed_rows"`
	RecentExecutions   []models.ThematicSyncExecution `json:"recent_executions"`
}
