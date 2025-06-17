/*
 * @module service/data_quality/cleanser
 * @description 数据清洗器，负责空值处理、重复数据去除、格式标准化等
 * @architecture 分层架构 - 数据清洗层
 * @documentReference ai_docs/basic_library_process_impl.md
 * @stateFlow 数据读取 -> 清洗规则应用 -> 清洗后数据输出
 * @rules 确保数据清洗的准确性和一致性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/data_quality
 */

package data_quality

import (
	"gorm.io/gorm"
)

// Cleanser 数据清洗器
type Cleanser struct {
	db *gorm.DB
}

// NewCleanser 创建数据清洗器实例
func NewCleanser(db *gorm.DB) *Cleanser {
	return &Cleanser{
		db: db,
	}
}
