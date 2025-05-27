/*
 * @module service/governance_service
 * @description 数据治理服务，提供数据质量管理、元数据管理、数据脱敏等功能
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/requirements.md
 * @stateFlow 数据治理生命周期管理
 * @rules 确保数据质量、安全性和合规性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/model.md
 */

package service

import (
	"datahub-service/service/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GovernanceService 数据治理服务
type GovernanceService struct {
	db *gorm.DB
}

// NewGovernanceService 创建数据治理服务实例
func NewGovernanceService(db *gorm.DB) *GovernanceService {
	return &GovernanceService{db: db}
}

// === 数据质量规则管理 ===

// CreateQualityRule 创建数据质量规则
func (s *GovernanceService) CreateQualityRule(rule *models.QualityRule) error {
	// 验证规则类型
	validTypes := []string{"completeness", "standardization", "consistency", "accuracy", "uniqueness", "timeliness"}
	isValidType := false
	for _, validType := range validTypes {
		if rule.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的数据质量规则类型")
	}

	// 验证关联对象类型
	validObjectTypes := []string{"interface", "thematic_interface"}
	isValidObjectType := false
	for _, validObjectType := range validObjectTypes {
		if rule.RelatedObjectType == validObjectType {
			isValidObjectType = true
			break
		}
	}
	if !isValidObjectType {
		return errors.New("无效的关联对象类型")
	}

	return s.db.Create(rule).Error
}

// GetQualityRules 获取数据质量规则列表
func (s *GovernanceService) GetQualityRules(page, pageSize int, ruleType, objectType string) ([]models.QualityRule, int64, error) {
	var rules []models.QualityRule
	var total int64

	query := s.db.Model(&models.QualityRule{})

	if ruleType != "" {
		query = query.Where("type = ?", ruleType)
	}
	if objectType != "" {
		query = query.Where("related_object_type = ?", objectType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// GetQualityRuleByID 根据ID获取数据质量规则
func (s *GovernanceService) GetQualityRuleByID(id string) (*models.QualityRule, error) {
	var rule models.QualityRule
	if err := s.db.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateQualityRule 更新数据质量规则
func (s *GovernanceService) UpdateQualityRule(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.QualityRule{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteQualityRule 删除数据质量规则
func (s *GovernanceService) DeleteQualityRule(id string) error {
	return s.db.Delete(&models.QualityRule{}, "id = ?", id).Error
}

// === 元数据管理 ===

// CreateMetadata 创建元数据
func (s *GovernanceService) CreateMetadata(metadata *models.Metadata) error {
	// 验证元数据类型
	validTypes := []string{"technical", "business", "management"}
	isValidType := false
	for _, validType := range validTypes {
		if metadata.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的元数据类型")
	}

	return s.db.Create(metadata).Error
}

// GetMetadataList 获取元数据列表
func (s *GovernanceService) GetMetadataList(page, pageSize int, metadataType, name string) ([]models.Metadata, int64, error) {
	var metadataList []models.Metadata
	var total int64

	query := s.db.Model(&models.Metadata{})

	if metadataType != "" {
		query = query.Where("type = ?", metadataType)
	}
	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&metadataList).Error; err != nil {
		return nil, 0, err
	}

	return metadataList, total, nil
}

// GetMetadataByID 根据ID获取元数据
func (s *GovernanceService) GetMetadataByID(id string) (*models.Metadata, error) {
	var metadata models.Metadata
	if err := s.db.First(&metadata, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &metadata, nil
}

// UpdateMetadata 更新元数据
func (s *GovernanceService) UpdateMetadata(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.Metadata{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteMetadata 删除元数据
func (s *GovernanceService) DeleteMetadata(id string) error {
	return s.db.Delete(&models.Metadata{}, "id = ?", id).Error
}

// === 数据脱敏规则管理 ===

// CreateMaskingRule 创建数据脱敏规则
func (s *GovernanceService) CreateMaskingRule(rule *models.DataMaskingRule) error {
	// 验证脱敏类型
	validTypes := []string{"mask", "replace", "encrypt", "pseudonymize"}
	isValidType := false
	for _, validType := range validTypes {
		if rule.MaskingType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的数据脱敏类型")
	}

	return s.db.Create(rule).Error
}

// GetMaskingRules 获取数据脱敏规则列表
func (s *GovernanceService) GetMaskingRules(page, pageSize int, dataSource, maskingType string) ([]models.DataMaskingRule, int64, error) {
	var rules []models.DataMaskingRule
	var total int64

	query := s.db.Model(&models.DataMaskingRule{}).Preload("Creator")

	if dataSource != "" {
		query = query.Where("data_source = ?", dataSource)
	}
	if maskingType != "" {
		query = query.Where("masking_type = ?", maskingType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// GetMaskingRuleByID 根据ID获取数据脱敏规则
func (s *GovernanceService) GetMaskingRuleByID(id string) (*models.DataMaskingRule, error) {
	var rule models.DataMaskingRule
	if err := s.db.Preload("Creator").First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateMaskingRule 更新数据脱敏规则
func (s *GovernanceService) UpdateMaskingRule(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataMaskingRule{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteMaskingRule 删除数据脱敏规则
func (s *GovernanceService) DeleteMaskingRule(id string) error {
	return s.db.Delete(&models.DataMaskingRule{}, "id = ?", id).Error
}

// === 系统日志管理 ===

// CreateSystemLog 创建系统日志
func (s *GovernanceService) CreateSystemLog(log *models.SystemLog) error {
	return s.db.Create(log).Error
}

// GetSystemLogs 获取系统日志列表
func (s *GovernanceService) GetSystemLogs(page, pageSize int, operationType, objectType string, startTime, endTime *time.Time) ([]models.SystemLog, int64, error) {
	var logs []models.SystemLog
	var total int64

	query := s.db.Model(&models.SystemLog{}).Preload("Operator")

	if operationType != "" {
		query = query.Where("operation_type = ?", operationType)
	}
	if objectType != "" {
		query = query.Where("object_type = ?", objectType)
	}
	if startTime != nil {
		query = query.Where("operation_time >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("operation_time <= ?", endTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("operation_time DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// === 备份管理 ===

// CreateBackupConfig 创建备份配置
func (s *GovernanceService) CreateBackupConfig(config *models.BackupConfig) error {
	// 验证备份类型
	validTypes := []string{"full", "incremental"}
	isValidType := false
	for _, validType := range validTypes {
		if config.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的备份类型")
	}

	// 验证对象类型
	validObjectTypes := []string{"thematic_library", "basic_library"}
	isValidObjectType := false
	for _, validObjectType := range validObjectTypes {
		if config.ObjectType == validObjectType {
			isValidObjectType = true
			break
		}
	}
	if !isValidObjectType {
		return errors.New("无效的备份对象类型")
	}

	return s.db.Create(config).Error
}

// GetBackupConfigs 获取备份配置列表
func (s *GovernanceService) GetBackupConfigs(page, pageSize int, objectType string) ([]models.BackupConfig, int64, error) {
	var configs []models.BackupConfig
	var total int64

	query := s.db.Model(&models.BackupConfig{})

	if objectType != "" {
		query = query.Where("object_type = ?", objectType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// CreateBackupRecord 创建备份记录
func (s *GovernanceService) CreateBackupRecord(record *models.BackupRecord) error {
	return s.db.Create(record).Error
}

// GetBackupRecords 获取备份记录列表
func (s *GovernanceService) GetBackupRecords(page, pageSize int, configID, status string) ([]models.BackupRecord, int64, error) {
	var records []models.BackupRecord
	var total int64

	query := s.db.Model(&models.BackupRecord{}).Preload("BackupConfig")

	if configID != "" {
		query = query.Where("backup_config_id = ?", configID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("start_time DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// === 数据质量报告 ===

// CreateQualityReport 创建数据质量报告
func (s *GovernanceService) CreateQualityReport(report *models.DataQualityReport) error {
	return s.db.Create(report).Error
}

// GetQualityReports 获取数据质量报告列表
func (s *GovernanceService) GetQualityReports(page, pageSize int, objectType string) ([]models.DataQualityReport, int64, error) {
	var reports []models.DataQualityReport
	var total int64

	query := s.db.Model(&models.DataQualityReport{}).Preload("Generator")

	if objectType != "" {
		query = query.Where("related_object_type = ?", objectType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("generated_at DESC").Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

// GetQualityReportByID 根据ID获取数据质量报告
func (s *GovernanceService) GetQualityReportByID(id string) (*models.DataQualityReport, error) {
	var report models.DataQualityReport
	if err := s.db.Preload("Generator").First(&report, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

// === 数据质量检查 ===

// RunQualityCheck 执行数据质量检查
func (s *GovernanceService) RunQualityCheck(objectID, objectType string) (*models.DataQualityReport, error) {
	// 获取相关的质量规则
	var rules []models.QualityRule
	if err := s.db.Where("related_object_id = ? AND related_object_type = ? AND is_enabled = ?",
		objectID, objectType, true).Find(&rules).Error; err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, errors.New("未找到启用的数据质量规则")
	}

	// 执行质量检查逻辑（这里是示例实现）
	qualityScore := 85.0 // 示例分数
	qualityMetrics := map[string]interface{}{
		"completeness":    90.0,
		"standardization": 85.0,
		"consistency":     80.0,
		"accuracy":        88.0,
		"uniqueness":      92.0,
		"timeliness":      85.0,
	}

	issues := map[string]interface{}{
		"missing_values":    []string{"field1", "field2"},
		"format_errors":     []string{"field3"},
		"duplicate_records": 15,
	}

	recommendations := map[string]interface{}{
		"actions": []string{
			"清理缺失值",
			"标准化数据格式",
			"去除重复记录",
		},
	}

	// 创建质量报告
	report := &models.DataQualityReport{
		ReportName:        fmt.Sprintf("数据质量报告_%s_%s", objectType, time.Now().Format("20060102150405")),
		RelatedObjectID:   objectID,
		RelatedObjectType: objectType,
		QualityScore:      qualityScore,
		QualityMetrics:    qualityMetrics,
		Issues:            issues,
		Recommendations:   recommendations,
		GeneratedBy:       "system", // 系统生成
	}

	if err := s.CreateQualityReport(report); err != nil {
		return nil, err
	}

	return report, nil
}
