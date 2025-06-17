/*
 * @module service/models/data_quality_engine_models
 * @description 数据质量引擎相关模型定义，包含质量检查、规则、问题等核心数据结构
 * @architecture 分层架构 - 数据模型层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 模型定义 -> 数据操作 -> 业务逻辑
 * @rules 确保数据质量模型的一致性和完整性
 * @dependencies gorm.io/gorm
 * @refs service/data_quality
 */

package models

import (
	"time"
)

// QualityCheckRequest 数据质量检查请求
type QualityCheckRequest struct {
	ObjectID    string                 `json:"object_id"`
	ObjectType  string                 `json:"object_type"` // interface, thematic_interface
	CheckTypes  []string               `json:"check_types"` // completeness, accuracy, consistency, etc.
	Config      map[string]interface{} `json:"config,omitempty"`
	SampleSize  int                    `json:"sample_size,omitempty"`
	ScheduledBy string                 `json:"scheduled_by"`
}

// QualityCheckResult 数据质量检查结果
type QualityCheckResult struct {
	CheckID         string                     `json:"check_id"`
	ObjectID        string                     `json:"object_id"`
	ObjectType      string                     `json:"object_type"`
	OverallScore    float64                    `json:"overall_score"`
	CheckResults    map[string]*CheckDimension `json:"check_results"`
	Issues          []QualityIssue             `json:"issues"`
	Recommendations []string                   `json:"recommendations"`
	Statistics      map[string]interface{}     `json:"statistics"`
	CheckTime       time.Time                  `json:"check_time"`
	Duration        time.Duration              `json:"duration"`
}

// CheckDimension 质量检查维度结果
type CheckDimension struct {
	Name        string                 `json:"name"`
	Score       float64                `json:"score"`
	Status      string                 `json:"status"` // pass, warning, fail
	Details     map[string]interface{} `json:"details"`
	IssueCount  int                    `json:"issue_count"`
	RecordCount int64                  `json:"record_count"`
}

// QualityIssue 数据质量问题
type QualityIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"` // high, medium, low
	Description string                 `json:"description"`
	Field       string                 `json:"field,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Count       int64                  `json:"count"`
	Percentage  float64                `json:"percentage"`
	Suggestion  string                 `json:"suggestion"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// QualityRuleEngine 质量规则引擎专用
type QualityRuleEngine struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // completeness, accuracy, consistency, etc.
	Category    string                 `json:"category"`
	Config      map[string]interface{} `json:"config"`
	Threshold   float64                `json:"threshold"`
	Weight      float64                `json:"weight"`
	IsEnabled   bool                   `json:"is_enabled"`
	Description string                 `json:"description"`
}
