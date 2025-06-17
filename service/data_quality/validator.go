/*
 * @module service/data_quality/validator
 * @description 数据验证器，负责必填字段检查、数据类型验证、格式验证等
 * @architecture 分层架构 - 数据验证层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 数据读取 -> 验证规则应用 -> 验证结果生成
 * @rules 确保数据验证的准确性和完整性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/data_quality
 */

package data_quality

import (
	"gorm.io/gorm"
)

// Validator 数据验证器
type Validator struct {
	db *gorm.DB
}

// NewValidator 创建数据验证器实例
func NewValidator(db *gorm.DB) *Validator {
	return &Validator{
		db: db,
	}
}
