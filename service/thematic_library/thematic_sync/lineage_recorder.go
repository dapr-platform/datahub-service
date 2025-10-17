/*
 * @module service/thematic_sync/lineage_recorder
 * @description 血缘记录器，负责记录数据血缘关系和转换历史
 * @architecture 观察者模式 - 记录数据处理过程中的血缘信息
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 血缘分析 -> 关系建立 -> 记录存储 -> 历史跟踪
 * @rules 确保血缘记录的准确性和完整性
 * @dependencies gorm.io/gorm, fmt, time
 * @refs sync_types.go, models/thematic_sync.go
 */

package thematic_sync

import (
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LineageRecorder 血缘记录器
type LineageRecorder struct {
	db *gorm.DB
}

// NewLineageRecorder 创建血缘记录器
func NewLineageRecorder(db *gorm.DB) *LineageRecorder {
	return &LineageRecorder{db: db}
}

// RecordLineage 记录简单的血缘关系
func (lr *LineageRecorder) RecordLineage(sourceRecords []SourceRecordInfo, processedRecords []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult) error {
	// 获取目标主题接口的主键字段
	targetPrimaryKeys, err := lr.getThematicPrimaryKeyFields(request.TargetInterfaceID)
	if err != nil {
		slog.Debug("获取目标主键字段失败，不使用排序", "error", err)
		targetPrimaryKeys = []string{}
	}

	// 为每个处理后的记录创建血缘记录
	for i, processedRecord := range processedRecords {
		// 根据目标接口的主键字段提取记录ID
		targetRecordID := lr.extractPrimaryKeyByFields(processedRecord, targetPrimaryKeys)
		if targetRecordID == "" {
			targetRecordID = fmt.Sprintf("record_%d", i)
		}

		lineage := &models.ThematicDataLineage{
			ID:                    uuid.New().String(),
			ThematicInterfaceID:   request.TargetInterfaceID,
			ThematicRecordID:      targetRecordID,
			ProcessingRules:       models.JSONB{},
			TransformationDetails: models.JSONB{},
			QualityScore:          result.QualityScore,
			QualityIssues:         models.JSONB{},
			SourceDataTime:        time.Now(),
			ProcessedTime:         time.Now(),
			CreatedAt:             time.Now(),
		}

		// 设置源数据信息（使用第一个源记录作为代表）
		if len(sourceRecords) > 0 {
			source := sourceRecords[0]
			lineage.SourceLibraryID = source.LibraryID
			lineage.SourceInterfaceID = source.InterfaceID
			lineage.SourceRecordID = source.RecordID
		}

		if err := lr.db.Create(lineage).Error; err != nil {
			return fmt.Errorf("创建血缘记录失败: %w", err)
		}
	}

	return nil
}

// getThematicPrimaryKeyFields 获取主题接口的主键字段列表
func (lr *LineageRecorder) getThematicPrimaryKeyFields(thematicInterfaceID string) ([]string, error) {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := lr.db.First(&thematicInterface, "id = ?", thematicInterfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	return GetThematicPrimaryKeyFields(&thematicInterface), nil
}

// extractPrimaryKeyByFields 根据指定字段提取主键值
func (lr *LineageRecorder) extractPrimaryKeyByFields(record map[string]interface{}, primaryKeyFields []string) string {
	var keyParts []string

	for _, field := range primaryKeyFields {
		if value, exists := record[field]; exists && value != nil {
			keyParts = append(keyParts, fmt.Sprintf("%v", value))
		} else {
			// 如果任一主键字段缺失，返回空字符串
			return ""
		}
	}

	// 如果是复合主键，用下划线连接
	if len(keyParts) > 1 {
		return keyParts[0] // 简化实现，只取第一个
	} else if len(keyParts) == 1 {
		return keyParts[0]
	}

	return ""
}
