/*
 * @module service/basic_library/status_service
 * @description 状态管理服务，管理数据源和接口的状态监控和健康评估
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 状态监控 -> 健康评估 -> 报警通知 -> 状态更新
 * @rules 确保状态信息的实时性和准确性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/interfaces.md
 */

package basic_library

import (
	"datahub-service/service/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StatusService 状态管理服务
type StatusService struct {
	db *gorm.DB
}

// NewStatusService 创建状态管理服务实例
func NewStatusService(db *gorm.DB) *StatusService {
	return &StatusService{
		db: db,
	}
}

// GetDataSourceStatus 获取数据源状态
func (s *StatusService) GetDataSourceStatus(dataSourceID string) (*models.DataSourceStatus, error) {
	var status models.DataSourceStatus
	err := s.db.Preload("DataSource").Where("data_source_id = ?", dataSourceID).First(&status).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果状态不存在，创建默认状态
			return s.createDefaultDataSourceStatus(dataSourceID)
		}
		return nil, err
	}
	return &status, nil
}

// GetDataSourceStatuses 获取数据源状态列表
func (s *StatusService) GetDataSourceStatuses(libraryID string, status string, page, pageSize int) ([]models.DataSourceStatus, int64, error) {
	var statuses []models.DataSourceStatus
	var total int64

	query := s.db.Model(&models.DataSourceStatus{}).Preload("DataSource")

	// 如果指定了基础库ID，需要关联查询
	if libraryID != "" {
		query = query.Joins("JOIN data_sources ON data_source_statuses.data_source_id = data_sources.id").
			Where("data_sources.library_id = ?", libraryID)
	}

	if status != "" {
		query = query.Where("data_source_statuses.status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("data_source_statuses.updated_at DESC").Offset(offset).Limit(pageSize).Find(&statuses).Error

	return statuses, total, err
}

// UpdateDataSourceStatus 更新数据源状态
func (s *StatusService) UpdateDataSourceStatus(dataSourceID, status string, errorMessage string, stats map[string]interface{}) error {
	var statusRecord models.DataSourceStatus
	err := s.db.Where("data_source_id = ?", dataSourceID).First(&statusRecord).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 创建新状态记录
		statusRecord = models.DataSourceStatus{
			DataSourceID: dataSourceID,
			Status:       status,
			ErrorMessage: errorMessage,
			UpdatedAt:    now,
		}

		if status == "online" {
			statusRecord.LastTestTime = &now
		} else if status == "error" || status == "offline" {
			statusRecord.LastErrorTime = &now
		}

		if stats != nil {
			if connInfo, exists := stats["connection_info"]; exists {
				statusRecord.ConnectionInfo = connInfo.(map[string]interface{})
			}
			if perfInfo, exists := stats["performance_info"]; exists {
				statusRecord.PerformanceInfo = perfInfo.(map[string]interface{})
			}
			if syncStats, exists := stats["sync_statistics"]; exists {
				statusRecord.SyncStatistics = syncStats.(map[string]interface{})
			}
		}

		// 计算健康评分
		statusRecord.HealthScore = s.calculateDataSourceHealthScore(&statusRecord)

		return s.db.Create(&statusRecord).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    now,
	}

	if status == "online" {
		updates["last_test_time"] = &now
	} else if status == "error" || status == "offline" {
		updates["last_error_time"] = &now
	}

	if stats != nil {
		if connInfo, exists := stats["connection_info"]; exists {
			updates["connection_info"] = connInfo
		}
		if perfInfo, exists := stats["performance_info"]; exists {
			updates["performance_info"] = perfInfo
		}
		if syncStats, exists := stats["sync_statistics"]; exists {
			updates["sync_statistics"] = syncStats
		}
	}

	// 重新计算健康评分
	if err := s.db.Model(&statusRecord).Updates(updates).Error; err != nil {
		return err
	}

	// 获取更新后的记录来计算健康评分
	if err := s.db.Where("data_source_id = ?", dataSourceID).First(&statusRecord).Error; err != nil {
		return err
	}

	healthScore := s.calculateDataSourceHealthScore(&statusRecord)
	return s.db.Model(&statusRecord).Update("health_score", healthScore).Error
}

// createDefaultDataSourceStatus 创建默认数据源状态
func (s *StatusService) createDefaultDataSourceStatus(dataSourceID string) (*models.DataSourceStatus, error) {
	status := models.DataSourceStatus{
		DataSourceID: dataSourceID,
		Status:       "unknown",
		HealthScore:  0,
		UpdatedAt:    time.Now(),
	}

	if err := s.db.Create(&status).Error; err != nil {
		return nil, err
	}

	return &status, nil
}

// calculateDataSourceHealthScore 计算数据源健康评分
func (s *StatusService) calculateDataSourceHealthScore(status *models.DataSourceStatus) int {
	score := 0

	// 基础状态评分
	switch status.Status {
	case "online":
		score += 40
	case "testing":
		score += 20
	case "offline":
		score += 10
	case "error":
		score += 0
	}

	// 连接稳定性评分
	if status.LastTestTime != nil {
		timeSinceLastTest := time.Since(*status.LastTestTime)
		if timeSinceLastTest < 5*time.Minute {
			score += 20
		} else if timeSinceLastTest < 30*time.Minute {
			score += 15
		} else if timeSinceLastTest < 2*time.Hour {
			score += 10
		} else {
			score += 5
		}
	}

	// 错误率评分
	if status.LastErrorTime != nil {
		timeSinceLastError := time.Since(*status.LastErrorTime)
		if timeSinceLastError > 24*time.Hour {
			score += 20
		} else if timeSinceLastError > 12*time.Hour {
			score += 15
		} else if timeSinceLastError > 6*time.Hour {
			score += 10
		} else {
			score += 5
		}
	} else {
		score += 20 // 没有错误记录
	}

	// 性能评分
	if status.PerformanceInfo != nil {
		if avgResponseTime, exists := status.PerformanceInfo["avg_response_time"]; exists {
			if rt, ok := avgResponseTime.(float64); ok {
				if rt < 100 {
					score += 20
				} else if rt < 500 {
					score += 15
				} else if rt < 1000 {
					score += 10
				} else {
					score += 5
				}
			}
		}
	}

	// 确保评分在0-100范围内
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// GetInterfaceStatus 获取接口状态
func (s *StatusService) GetInterfaceStatus(interfaceID string) (*models.InterfaceStatus, error) {
	var status models.InterfaceStatus
	err := s.db.Preload("DataInterface").Where("interface_id = ?", interfaceID).First(&status).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果状态不存在，创建默认状态
			return s.createDefaultInterfaceStatus(interfaceID)
		}
		return nil, err
	}
	return &status, nil
}

// GetInterfaceStatuses 获取接口状态列表
func (s *StatusService) GetInterfaceStatuses(libraryID string, status string, page, pageSize int) ([]models.InterfaceStatus, int64, error) {
	var statuses []models.InterfaceStatus
	var total int64

	query := s.db.Model(&models.InterfaceStatus{}).Preload("DataInterface")

	// 如果指定了基础库ID，需要关联查询
	if libraryID != "" {
		query = query.Joins("JOIN data_interfaces ON interface_statuses.interface_id = data_interfaces.id").
			Where("data_interfaces.library_id = ?", libraryID)
	}

	if status != "" {
		query = query.Where("interface_statuses.status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("interface_statuses.updated_at DESC").Offset(offset).Limit(pageSize).Find(&statuses).Error

	return statuses, total, err
}

// UpdateInterfaceStatus 更新接口状态
func (s *StatusService) UpdateInterfaceStatus(interfaceID, status string, errorMessage string, stats map[string]interface{}) error {
	var statusRecord models.InterfaceStatus
	err := s.db.Where("interface_id = ?", interfaceID).First(&statusRecord).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 创建新状态记录
		statusRecord = models.InterfaceStatus{
			InterfaceID:  interfaceID,
			Status:       status,
			ErrorMessage: errorMessage,
			UpdatedAt:    now,
		}

		if status == "active" {
			statusRecord.LastTestTime = &now
		} else if status == "error" || status == "inactive" {
			statusRecord.LastErrorTime = &now
		}

		if stats != nil {
			if queryStats, exists := stats["query_statistics"]; exists {
				statusRecord.QueryStatistics = queryStats.(map[string]interface{})
			}
			if dataStats, exists := stats["data_statistics"]; exists {
				statusRecord.DataStatistics = dataStats.(map[string]interface{})
			}
			if perfInfo, exists := stats["performance_info"]; exists {
				statusRecord.PerformanceInfo = perfInfo.(map[string]interface{})
			}
		}

		// 计算质量评分
		statusRecord.QualityScore = s.calculateInterfaceQualityScore(&statusRecord)

		return s.db.Create(&statusRecord).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMessage,
		"updated_at":    now,
	}

	if status == "active" {
		updates["last_test_time"] = &now
		updates["last_query_time"] = &now
	} else if status == "error" || status == "inactive" {
		updates["last_error_time"] = &now
	}

	if stats != nil {
		if queryStats, exists := stats["query_statistics"]; exists {
			updates["query_statistics"] = queryStats
		}
		if dataStats, exists := stats["data_statistics"]; exists {
			updates["data_statistics"] = dataStats
		}
		if perfInfo, exists := stats["performance_info"]; exists {
			updates["performance_info"] = perfInfo
		}
	}

	// 重新计算质量评分
	if err := s.db.Model(&statusRecord).Updates(updates).Error; err != nil {
		return err
	}

	// 获取更新后的记录来计算质量评分
	if err := s.db.Where("interface_id = ?", interfaceID).First(&statusRecord).Error; err != nil {
		return err
	}

	qualityScore := s.calculateInterfaceQualityScore(&statusRecord)
	return s.db.Model(&statusRecord).Update("quality_score", qualityScore).Error
}

// createDefaultInterfaceStatus 创建默认接口状态
func (s *StatusService) createDefaultInterfaceStatus(interfaceID string) (*models.InterfaceStatus, error) {
	status := models.InterfaceStatus{
		InterfaceID:  interfaceID,
		Status:       "unknown",
		QualityScore: 0,
		UpdatedAt:    time.Now(),
	}

	if err := s.db.Create(&status).Error; err != nil {
		return nil, err
	}

	return &status, nil
}

// calculateInterfaceQualityScore 计算接口质量评分
func (s *StatusService) calculateInterfaceQualityScore(status *models.InterfaceStatus) int {
	score := 0

	// 基础状态评分
	switch status.Status {
	case "active":
		score += 30
	case "testing":
		score += 15
	case "inactive":
		score += 5
	case "error":
		score += 0
	}

	// 查询稳定性评分
	if status.LastTestTime != nil {
		timeSinceLastTest := time.Since(*status.LastTestTime)
		if timeSinceLastTest < 5*time.Minute {
			score += 15
		} else if timeSinceLastTest < 30*time.Minute {
			score += 10
		} else if timeSinceLastTest < 2*time.Hour {
			score += 5
		}
	}

	// 错误率评分
	if status.LastErrorTime != nil {
		timeSinceLastError := time.Since(*status.LastErrorTime)
		if timeSinceLastError > 24*time.Hour {
			score += 15
		} else if timeSinceLastError > 12*time.Hour {
			score += 10
		} else if timeSinceLastError > 6*time.Hour {
			score += 5
		}
	} else {
		score += 15 // 没有错误记录
	}

	// 数据质量评分
	if status.DataStatistics != nil {
		if completeness, exists := status.DataStatistics["completeness"]; exists {
			if comp, ok := completeness.(float64); ok {
				score += int(comp * 20) // 完整性评分最高20分
			}
		}

		if accuracy, exists := status.DataStatistics["accuracy"]; exists {
			if acc, ok := accuracy.(float64); ok {
				score += int(acc * 15) // 准确性评分最高15分
			}
		}
	}

	// 性能评分
	if status.PerformanceInfo != nil {
		if avgResponseTime, exists := status.PerformanceInfo["avg_response_time"]; exists {
			if rt, ok := avgResponseTime.(float64); ok {
				if rt < 100 {
					score += 5
				} else if rt < 500 {
					score += 3
				} else if rt < 1000 {
					score += 1
				}
			}
		}
	}

	// 确保评分在0-100范围内
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// GetSystemStatus 获取系统总体状态
func (s *StatusService) GetSystemStatus(libraryID string) (map[string]interface{}, error) {
	// 获取数据源状态统计
	dataSourceStats, err := s.getDataSourceStatusStats(libraryID)
	if err != nil {
		return nil, fmt.Errorf("获取数据源状态统计失败: %v", err)
	}

	// 获取接口状态统计
	interfaceStats, err := s.getInterfaceStatusStats(libraryID)
	if err != nil {
		return nil, fmt.Errorf("获取接口状态统计失败: %v", err)
	}

	// 计算系统健康评分
	systemHealth := s.calculateSystemHealthScore(dataSourceStats, interfaceStats)

	return map[string]interface{}{
		"data_source_status": dataSourceStats,
		"interface_status":   interfaceStats,
		"system_health":      systemHealth,
		"timestamp":          time.Now(),
	}, nil
}

// getDataSourceStatusStats 获取数据源状态统计
func (s *StatusService) getDataSourceStatusStats(libraryID string) (map[string]interface{}, error) {
	query := s.db.Model(&models.DataSourceStatus{})

	if libraryID != "" {
		query = query.Joins("JOIN data_sources ON data_source_statuses.data_source_id = data_sources.id").
			Where("data_sources.library_id = ?", libraryID)
	}

	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&statusStats).Error; err != nil {
		return nil, err
	}

	// 总数
	var total int64
	query.Count(&total)

	// 平均健康评分
	var avgHealthScore float64
	query.Select("AVG(health_score)").Scan(&avgHealthScore)

	return map[string]interface{}{
		"total":            total,
		"status_breakdown": statusStats,
		"avg_health_score": avgHealthScore,
	}, nil
}

// getInterfaceStatusStats 获取接口状态统计
func (s *StatusService) getInterfaceStatusStats(libraryID string) (map[string]interface{}, error) {
	query := s.db.Model(&models.InterfaceStatus{})

	if libraryID != "" {
		query = query.Joins("JOIN data_interfaces ON interface_statuses.interface_id = data_interfaces.id").
			Where("data_interfaces.library_id = ?", libraryID)
	}

	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&statusStats).Error; err != nil {
		return nil, err
	}

	// 总数
	var total int64
	query.Count(&total)

	// 平均质量评分
	var avgQualityScore float64
	query.Select("AVG(quality_score)").Scan(&avgQualityScore)

	return map[string]interface{}{
		"total":             total,
		"status_breakdown":  statusStats,
		"avg_quality_score": avgQualityScore,
	}, nil
}

// calculateSystemHealthScore 计算系统健康评分
func (s *StatusService) calculateSystemHealthScore(dataSourceStats, interfaceStats map[string]interface{}) int {
	// 数据源健康评分权重50%
	dsHealthScore := 0.0
	if avgScore, exists := dataSourceStats["avg_health_score"]; exists {
		if score, ok := avgScore.(float64); ok {
			dsHealthScore = score * 0.5
		}
	}

	// 接口质量评分权重50%
	ifQualityScore := 0.0
	if avgScore, exists := interfaceStats["avg_quality_score"]; exists {
		if score, ok := avgScore.(float64); ok {
			ifQualityScore = score * 0.5
		}
	}

	totalScore := int(dsHealthScore + ifQualityScore)
	if totalScore > 100 {
		totalScore = 100
	}
	if totalScore < 0 {
		totalScore = 0
	}

	return totalScore
}
