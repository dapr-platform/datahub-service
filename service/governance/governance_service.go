/*
 * @module service/governance/governance_service
 * @description 数据治理服务，提供数据质量、元数据、脱敏等治理功能
 * @architecture 分层架构 - 服务层
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow 数据治理生命周期管理
 * @rules 确保数据质量、安全性和合规性
 * @dependencies gorm.io/gorm, github.com/google/uuid
 * @refs service/models/governance.go
 */

package governance

import (
	"datahub-service/service/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GovernanceService 数据治理服务
type GovernanceService struct {
	db              *gorm.DB
	ruleEngine      *RuleEngine
	templateService *TemplateService
}

// NewGovernanceService 创建数据治理服务实例
func NewGovernanceService(db *gorm.DB) *GovernanceService {
	return &GovernanceService{
		db:              db,
		ruleEngine:      NewRuleEngine(db),
		templateService: NewTemplateService(db),
	}
}

// GetTemplateService 获取模板服务实例
func (s *GovernanceService) GetTemplateService() *TemplateService {
	return s.templateService
}

// === 数据质量规则管理 ===

// CreateQualityRule 创建数据质量规则
func (s *GovernanceService) CreateQualityRule(rule *models.QualityRuleTemplate) error {
	// 验证规则类型
	validTypes := []string{"completeness", "accuracy", "consistency", "validity", "uniqueness", "timeliness", "standardization"}
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

	// 验证分类
	validCategories := []string{"basic_quality", "data_cleansing", "data_validation"}
	isValidCategory := false
	for _, validCategory := range validCategories {
		if rule.Category == validCategory {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return errors.New("无效的数据质量规则分类")
	}

	return s.db.Create(rule).Error
}

// GetQualityRules 获取数据质量规则列表
func (s *GovernanceService) GetQualityRules(page, pageSize int, ruleType, objectType string) ([]models.QualityRuleTemplate, int64, error) {
	var rules []models.QualityRuleTemplate
	var total int64

	query := s.db.Model(&models.QualityRuleTemplate{})

	if ruleType != "" {
		query = query.Where("type = ?", ruleType)
	}
	if objectType != "" {
		// 这里可以根据对象类型进行过滤，暂时忽略
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
func (s *GovernanceService) GetQualityRuleByID(id string) (*models.QualityRuleTemplate, error) {
	var rule models.QualityRuleTemplate
	if err := s.db.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateQualityRule 更新数据质量规则
func (s *GovernanceService) UpdateQualityRule(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.QualityRuleTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteQualityRule 删除数据质量规则
func (s *GovernanceService) DeleteQualityRule(id string) error {
	return s.db.Delete(&models.QualityRuleTemplate{}, "id = ?", id).Error
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

// CreateMaskingRule 创建脱敏规则
func (s *GovernanceService) CreateMaskingRule(rule *models.DataMaskingTemplate) error {
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

// GetMaskingRules 获取脱敏规则列表
func (s *GovernanceService) GetMaskingRules(page, pageSize int, dataSource, maskingType string) ([]models.DataMaskingTemplate, int64, error) {
	var rules []models.DataMaskingTemplate
	var total int64

	query := s.db.Model(&models.DataMaskingTemplate{})

	if dataSource != "" {
		// 这里可以根据数据源进行过滤，暂时忽略
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

// GetMaskingRuleByID 根据ID获取脱敏规则
func (s *GovernanceService) GetMaskingRuleByID(id string) (*models.DataMaskingTemplate, error) {
	var rule models.DataMaskingTemplate
	if err := s.db.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateMaskingRule 更新脱敏规则
func (s *GovernanceService) UpdateMaskingRule(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataMaskingTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteMaskingRule 删除脱敏规则
func (s *GovernanceService) DeleteMaskingRule(id string) error {
	return s.db.Delete(&models.DataMaskingTemplate{}, "id = ?", id).Error
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

	query := s.db.Model(&models.SystemLog{})

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

// === 备份配置管理 ===

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

// === 数据质量报告管理 ===

// CreateQualityReport 创建质量报告
func (s *GovernanceService) CreateQualityReport(report *models.DataQualityReport) error {
	return s.db.Create(report).Error
}

// GetQualityReports 获取质量报告列表
func (s *GovernanceService) GetQualityReports(page, pageSize int, objectType string) ([]models.DataQualityReport, int64, error) {
	var reports []models.DataQualityReport
	var total int64

	query := s.db.Model(&models.DataQualityReport{})

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

// GetQualityReportByID 根据ID获取质量报告
func (s *GovernanceService) GetQualityReportByID(id string) (*models.DataQualityReport, error) {
	var report models.DataQualityReport
	if err := s.db.First(&report, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

// RunQualityCheck 执行数据质量检查
func (s *GovernanceService) RunQualityCheck(objectID, objectType string) (*models.DataQualityReport, error) {
	// 模拟质量检查过程
	report := &models.DataQualityReport{
		ReportName:        fmt.Sprintf("%s质量检查报告", objectType),
		RelatedObjectID:   objectID,
		RelatedObjectType: objectType,
		QualityScore:      85.5, // 模拟质量评分
		QualityMetrics: map[string]interface{}{
			"completeness": 90.0,
			"accuracy":     85.0,
			"consistency":  80.0,
			"validity":     88.0,
			"uniqueness":   95.0,
			"timeliness":   82.0,
		},
		Issues: map[string]interface{}{
			"missing_values": 150,
			"invalid_format": 45,
			"duplicates":     12,
		},
		Recommendations: map[string]interface{}{
			"actions": []string{
				"清理重复数据",
				"标准化数据格式",
				"完善数据验证规则",
			},
		},
		GeneratedAt: time.Now(),
		GeneratedBy: "system",
	}

	if err := s.CreateQualityReport(report); err != nil {
		return nil, err
	}

	return report, nil
}

// === 数据清洗规则管理 ===

// CreateCleansingRule 创建清洗规则
func (s *GovernanceService) CreateCleansingRule(rule *models.DataCleansingTemplate) error {
	// 验证清洗规则类型
	validTypes := []string{"standardization", "deduplication", "validation", "transformation", "enrichment"}
	isValidType := false
	for _, validType := range validTypes {
		if rule.RuleType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的数据清洗规则类型")
	}

	return s.db.Create(rule).Error
}

// GetCleansingRules 获取清洗规则列表
func (s *GovernanceService) GetCleansingRules(page, pageSize int, ruleType, targetTable string) ([]models.DataCleansingTemplate, int64, error) {
	var rules []models.DataCleansingTemplate
	var total int64

	query := s.db.Model(&models.DataCleansingTemplate{})

	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}
	if targetTable != "" {
		query = query.Where("target_table = ?", targetTable)
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

// GetCleansingRuleByID 根据ID获取清洗规则
func (s *GovernanceService) GetCleansingRuleByID(id string) (*models.DataCleansingTemplate, error) {
	var rule models.DataCleansingTemplate
	if err := s.db.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateCleansingRule 更新清洗规则
func (s *GovernanceService) UpdateCleansingRule(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataCleansingTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteCleansingRule 删除清洗规则
func (s *GovernanceService) DeleteCleansingRule(id string) error {
	return s.db.Delete(&models.DataCleansingTemplate{}, "id = ?", id).Error
}

// === 数据质量检测任务管理 ===

// CreateQualityTask 创建质量检测任务
func (s *GovernanceService) CreateQualityTask(req *CreateQualityTaskRequest) (*QualityTaskResponse, error) {
	task := &models.QualityTask{
		Name:               req.Name,
		Description:        req.Description,
		TaskType:           req.TaskType,
		TargetObjectID:     req.TargetObjectID,
		TargetObjectType:   req.TargetObjectType,
		QualityRuleIDs:     models.JSONBStringArray(req.QualityRuleIDs),
		ScheduleConfig:     models.JSONB(req.ScheduleConfig),
		NotificationConfig: models.JSONB(req.NotificationConfig),
		Status:             "pending",
		Priority:           req.Priority,
		IsEnabled:          req.IsEnabled,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	response := &QualityTaskResponse{
		ID:                 task.ID,
		Name:               task.Name,
		Description:        task.Description,
		TaskType:           task.TaskType,
		TargetObjectID:     task.TargetObjectID,
		TargetObjectType:   task.TargetObjectType,
		QualityRuleIDs:     req.QualityRuleIDs,
		ScheduleConfig:     req.ScheduleConfig,
		NotificationConfig: req.NotificationConfig,
		Status:             task.Status,
		Priority:           task.Priority,
		IsEnabled:          task.IsEnabled,
		LastExecuted:       task.LastExecuted,
		NextExecution:      task.NextExecution,
		ExecutionCount:     task.ExecutionCount,
		SuccessCount:       task.SuccessCount,
		FailureCount:       task.FailureCount,
		CreatedAt:          task.CreatedAt,
		CreatedBy:          task.CreatedBy,
		UpdatedAt:          task.UpdatedAt,
		UpdatedBy:          task.UpdatedBy,
	}

	return response, nil
}

// GetQualityTasks 获取质量检测任务列表
func (s *GovernanceService) GetQualityTasks(page, pageSize int, status, taskType string) ([]QualityTaskResponse, int64, error) {
	var tasks []models.QualityTask
	var total int64

	query := s.db.Model(&models.QualityTask{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	var responses []QualityTaskResponse
	for _, task := range tasks {
		qualityRuleIDs := []string(task.QualityRuleIDs)
		var scheduleConfig, notificationConfig map[string]interface{}

		if task.ScheduleConfig != nil {
			scheduleConfig = map[string]interface{}(task.ScheduleConfig)
		}
		if task.NotificationConfig != nil {
			notificationConfig = map[string]interface{}(task.NotificationConfig)
		}

		responses = append(responses, QualityTaskResponse{
			ID:                 task.ID,
			Name:               task.Name,
			Description:        task.Description,
			TaskType:           task.TaskType,
			TargetObjectID:     task.TargetObjectID,
			TargetObjectType:   task.TargetObjectType,
			QualityRuleIDs:     qualityRuleIDs,
			ScheduleConfig:     scheduleConfig,
			NotificationConfig: notificationConfig,
			Status:             task.Status,
			Priority:           task.Priority,
			IsEnabled:          task.IsEnabled,
			LastExecuted:       task.LastExecuted,
			NextExecution:      task.NextExecution,
			ExecutionCount:     task.ExecutionCount,
			SuccessCount:       task.SuccessCount,
			FailureCount:       task.FailureCount,
			CreatedAt:          task.CreatedAt,
			CreatedBy:          task.CreatedBy,
			UpdatedAt:          task.UpdatedAt,
			UpdatedBy:          task.UpdatedBy,
		})
	}

	return responses, total, nil
}

// GetQualityTaskByID 根据ID获取质量检测任务
func (s *GovernanceService) GetQualityTaskByID(id string) (*QualityTaskResponse, error) {
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}

	qualityRuleIDs := []string(task.QualityRuleIDs)
	var scheduleConfig, notificationConfig map[string]interface{}

	if task.ScheduleConfig != nil {
		scheduleConfig = map[string]interface{}(task.ScheduleConfig)
	}
	if task.NotificationConfig != nil {
		notificationConfig = map[string]interface{}(task.NotificationConfig)
	}

	response := &QualityTaskResponse{
		ID:                 task.ID,
		Name:               task.Name,
		Description:        task.Description,
		TaskType:           task.TaskType,
		TargetObjectID:     task.TargetObjectID,
		TargetObjectType:   task.TargetObjectType,
		QualityRuleIDs:     qualityRuleIDs,
		ScheduleConfig:     scheduleConfig,
		NotificationConfig: notificationConfig,
		Status:             task.Status,
		Priority:           task.Priority,
		IsEnabled:          task.IsEnabled,
		LastExecuted:       task.LastExecuted,
		NextExecution:      task.NextExecution,
		ExecutionCount:     task.ExecutionCount,
		SuccessCount:       task.SuccessCount,
		FailureCount:       task.FailureCount,
		CreatedAt:          task.CreatedAt,
		CreatedBy:          task.CreatedBy,
		UpdatedAt:          task.UpdatedAt,
		UpdatedBy:          task.UpdatedBy,
	}

	return response, nil
}

// StartQualityTask 启动质量检测任务
func (s *GovernanceService) StartQualityTask(id string) (*QualityTaskExecutionResponse, error) {
	// 检查任务是否存在
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}

	if !task.IsEnabled {
		return nil, errors.New("任务未启用")
	}

	if task.Status == "running" {
		return nil, errors.New("任务正在运行中")
	}

	// 创建执行记录
	execution := &models.QualityTaskExecution{
		TaskID:        id,
		ExecutionType: "manual",
		StartTime:     time.Now(),
		Status:        "running",
		ExecutedBy:    "system",
	}

	if err := s.db.Create(execution).Error; err != nil {
		return nil, err
	}

	// 更新任务状态
	s.db.Model(&models.QualityTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "running",
	})

	// 异步执行任务
	go s.executeQualityTask(execution)

	response := &QualityTaskExecutionResponse{
		ID:            execution.ID,
		TaskID:        execution.TaskID,
		ExecutionType: execution.ExecutionType,
		StartTime:     execution.StartTime,
		Status:        execution.Status,
		ExecutedBy:    execution.ExecutedBy,
	}

	return response, nil
}

// StopQualityTask 停止质量检测任务
func (s *GovernanceService) StopQualityTask(id string) error {
	// 更新任务状态为取消
	return s.db.Model(&models.QualityTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "cancelled",
	}).Error
}

// UpdateQualityTask 更新质量检测任务
func (s *GovernanceService) UpdateQualityTask(id string, req *UpdateQualityTaskRequest) error {
	// 检查任务是否存在
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return err
	}

	// 检查任务是否正在运行
	if task.Status == "running" {
		return errors.New("正在运行的任务不能修改")
	}

	// 构建更新数据
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.QualityRuleIDs != nil {
		updates["quality_rule_ids"] = models.JSONBStringArray(req.QualityRuleIDs)
	}
	if req.ScheduleConfig != nil {
		updates["schedule_config"] = models.JSONB(req.ScheduleConfig)
	}
	if req.NotificationConfig != nil {
		updates["notification_config"] = models.JSONB(req.NotificationConfig)
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}

	return s.db.Model(&models.QualityTask{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteQualityTask 删除质量检测任务
func (s *GovernanceService) DeleteQualityTask(id string) error {
	// 检查任务是否存在
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return err
	}

	// 检查任务是否正在运行
	if task.Status == "running" {
		return errors.New("正在运行的任务不能删除")
	}

	// 使用事务删除任务和相关的执行记录
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除执行记录
		if err := tx.Delete(&models.QualityTaskExecution{}, "task_id = ?", id).Error; err != nil {
			return err
		}

		// 删除任务
		if err := tx.Delete(&models.QualityTask{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetQualityTaskExecutions 获取质量检测任务执行记录
func (s *GovernanceService) GetQualityTaskExecutions(taskID string, page, pageSize int) ([]QualityTaskExecutionResponse, int64, error) {
	var executions []models.QualityTaskExecution
	var total int64

	query := s.db.Model(&models.QualityTaskExecution{}).Where("task_id = ?", taskID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("start_time DESC").Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	var responses []QualityTaskExecutionResponse
	for _, execution := range executions {
		var executionResults map[string]interface{}
		if execution.ExecutionResults != nil {
			executionResults = map[string]interface{}(execution.ExecutionResults)
		}

		responses = append(responses, QualityTaskExecutionResponse{
			ID:                 execution.ID,
			TaskID:             execution.TaskID,
			ExecutionType:      execution.ExecutionType,
			StartTime:          execution.StartTime,
			EndTime:            execution.EndTime,
			Duration:           execution.Duration,
			Status:             execution.Status,
			TotalRulesExecuted: execution.TotalRulesExecuted,
			PassedRules:        execution.PassedRules,
			FailedRules:        execution.FailedRules,
			OverallScore:       execution.OverallScore,
			ExecutionResults:   executionResults,
			ErrorMessage:       execution.ErrorMessage,
			TriggerSource:      execution.TriggerSource,
			ExecutedBy:         execution.ExecutedBy,
			CreatedAt:          execution.CreatedAt,
			UpdatedAt:          execution.UpdatedAt,
		})
	}

	return responses, total, nil
}

// executeQualityTask 异步执行质量检测任务
func (s *GovernanceService) executeQualityTask(execution *models.QualityTaskExecution) {
	// 模拟任务执行
	time.Sleep(5 * time.Second)

	// 更新执行结果
	endTime := time.Now()
	duration := endTime.Sub(execution.StartTime).Milliseconds()

	// 模拟执行结果
	totalRules := 5
	passedRules := 4
	failedRules := 1
	overallScore := 0.8

	executionResults := map[string]interface{}{
		"summary": "质量检测完成",
		"rules": []map[string]interface{}{
			{"rule_id": "rule_001", "status": "passed", "score": 0.9},
			{"rule_id": "rule_002", "status": "passed", "score": 0.85},
			{"rule_id": "rule_003", "status": "passed", "score": 0.75},
			{"rule_id": "rule_004", "status": "passed", "score": 0.95},
			{"rule_id": "rule_005", "status": "failed", "score": 0.45},
		},
	}

	updates := map[string]interface{}{
		"end_time":             &endTime,
		"duration":             duration,
		"status":               "completed",
		"total_rules_executed": totalRules,
		"passed_rules":         passedRules,
		"failed_rules":         failedRules,
		"overall_score":        overallScore,
		"execution_results":    models.JSONB(executionResults),
	}

	s.db.Model(&models.QualityTaskExecution{}).Where("id = ?", execution.ID).Updates(updates)

	// 更新任务状态和统计信息
	s.db.Model(&models.QualityTask{}).Where("id = ?", execution.TaskID).Updates(map[string]interface{}{
		"status":          "completed",
		"last_executed":   &endTime,
		"execution_count": gorm.Expr("execution_count + 1"),
		"success_count":   gorm.Expr("success_count + 1"),
	})
}

// === 数据血缘管理 ===

// CreateDataLineage 创建数据血缘关系
func (s *GovernanceService) CreateDataLineage(req *CreateDataLineageRequest) (*DataLineageResponse, error) {
	lineage := &models.DataLineage{
		SourceObjectID:   req.SourceObjectID,
		SourceObjectType: req.SourceObjectType,
		TargetObjectID:   req.TargetObjectID,
		TargetObjectType: req.TargetObjectType,
		RelationType:     req.RelationType,
		TransformRule:    models.JSONB(req.TransformRule),
		ColumnMapping:    models.JSONB(req.ColumnMapping),
		Confidence:       req.Confidence,
		IsActive:         req.IsActive,
		Description:      req.Description,
	}

	if err := s.db.Create(lineage).Error; err != nil {
		return nil, err
	}

	response := &DataLineageResponse{
		ID:               lineage.ID,
		SourceObjectID:   lineage.SourceObjectID,
		SourceObjectType: lineage.SourceObjectType,
		TargetObjectID:   lineage.TargetObjectID,
		TargetObjectType: lineage.TargetObjectType,
		RelationType:     lineage.RelationType,
		TransformRule:    req.TransformRule,
		ColumnMapping:    req.ColumnMapping,
		Confidence:       lineage.Confidence,
		IsActive:         lineage.IsActive,
		Description:      lineage.Description,
		CreatedAt:        lineage.CreatedAt,
		CreatedBy:        lineage.CreatedBy,
		UpdatedAt:        lineage.UpdatedAt,
		UpdatedBy:        lineage.UpdatedBy,
	}

	return response, nil
}

// GetDataLineage 获取数据血缘图
func (s *GovernanceService) GetDataLineage(objectID, objectType, direction string, depth int) (*DataLineageGraphResponse, error) {
	nodes := make(map[string]DataLineageNode)
	edges := make([]DataLineageEdge, 0)

	// 添加根节点
	nodes[objectID] = DataLineageNode{
		ID:         objectID,
		ObjectType: objectType,
		Name:       fmt.Sprintf("%s_%s", objectType, objectID),
		Level:      0,
	}

	// 递归构建血缘图
	if err := s.buildLineageGraph(objectID, objectType, direction, depth, 0, nodes, &edges); err != nil {
		return nil, err
	}

	// 转换为切片
	nodeSlice := make([]DataLineageNode, 0, len(nodes))
	for _, node := range nodes {
		nodeSlice = append(nodeSlice, node)
	}

	response := &DataLineageGraphResponse{
		Nodes: nodeSlice,
		Edges: edges,
		Stats: DataLineageStats{
			TotalNodes: len(nodeSlice),
			TotalEdges: len(edges),
			MaxDepth:   depth,
		},
	}

	return response, nil
}

// buildLineageGraph 递归构建血缘图
func (s *GovernanceService) buildLineageGraph(objectID, objectType, direction string, maxDepth, currentDepth int, nodes map[string]DataLineageNode, edges *[]DataLineageEdge) error {
	if currentDepth >= maxDepth {
		return nil
	}

	var lineages []models.DataLineage

	// 根据方向查询血缘关系
	query := s.db.Model(&models.DataLineage{}).Where("is_active = ?", true)

	switch direction {
	case "upstream":
		query = query.Where("target_object_id = ? AND target_object_type = ?", objectID, objectType)
	case "downstream":
		query = query.Where("source_object_id = ? AND source_object_type = ?", objectID, objectType)
	case "both":
		query = query.Where("(source_object_id = ? AND source_object_type = ?) OR (target_object_id = ? AND target_object_type = ?)",
			objectID, objectType, objectID, objectType)
	}

	if err := query.Find(&lineages).Error; err != nil {
		return err
	}

	for _, lineage := range lineages {
		var relatedObjectID, relatedObjectType string
		var isUpstream bool

		if lineage.SourceObjectID == objectID {
			// 下游关系
			relatedObjectID = lineage.TargetObjectID
			relatedObjectType = lineage.TargetObjectType
			isUpstream = false
		} else {
			// 上游关系
			relatedObjectID = lineage.SourceObjectID
			relatedObjectType = lineage.SourceObjectType
			isUpstream = true
		}

		// 添加节点
		if _, exists := nodes[relatedObjectID]; !exists {
			nodes[relatedObjectID] = DataLineageNode{
				ID:         relatedObjectID,
				ObjectType: relatedObjectType,
				Name:       fmt.Sprintf("%s_%s", relatedObjectType, relatedObjectID),
				Level:      currentDepth + 1,
			}

			// 递归处理
			if err := s.buildLineageGraph(relatedObjectID, relatedObjectType, direction, maxDepth, currentDepth+1, nodes, edges); err != nil {
				return err
			}
		}

		// 添加边
		var sourceID, targetID string
		if isUpstream {
			sourceID = relatedObjectID
			targetID = objectID
		} else {
			sourceID = objectID
			targetID = relatedObjectID
		}

		edge := DataLineageEdge{
			ID:           lineage.ID,
			SourceID:     sourceID,
			TargetID:     targetID,
			RelationType: lineage.RelationType,
			Confidence:   lineage.Confidence,
		}

		*edges = append(*edges, edge)
	}

	return nil
}

// === 规则测试方法 ===

// createTestSummary 创建测试汇总信息
func createTestSummary(qualityChecks, maskingRules, cleansingRules int, overallScore float64) struct {
	QualityChecks  int     `json:"quality_checks" example:"1"`
	MaskingRules   int     `json:"masking_rules" example:"1"`
	CleansingRules int     `json:"cleansing_rules" example:"1"`
	OverallScore   float64 `json:"overall_score,omitempty" example:"0.75"`
} {
	return struct {
		QualityChecks  int     `json:"quality_checks" example:"1"`
		MaskingRules   int     `json:"masking_rules" example:"1"`
		CleansingRules int     `json:"cleansing_rules" example:"1"`
		OverallScore   float64 `json:"overall_score,omitempty" example:"0.75"`
	}{
		QualityChecks:  qualityChecks,
		MaskingRules:   maskingRules,
		CleansingRules: cleansingRules,
		OverallScore:   overallScore,
	}
}

// TestQualityRule 测试数据质量规则
func (s *GovernanceService) TestQualityRule(req *TestQualityRuleRequest) (*TestRuleResponse, error) {
	startTime := time.Now()
	testID := uuid.New().String()

	// 获取规则模板
	template, err := s.GetQualityRuleByID(req.RuleTemplateID)
	if err != nil {
		return nil, fmt.Errorf("获取质量规则模板失败: %v", err)
	}

	// 构建质量规则配置
	config := models.QualityRuleConfig{
		RuleTemplateID: req.RuleTemplateID,
		TargetFields:   req.TargetFields,
		RuntimeConfig:  models.JSONB(req.RuntimeConfig),
		Threshold:      models.JSONB(req.Threshold),
		IsEnabled:      true,
	}

	// 执行质量规则
	result, err := s.ruleEngine.ApplyQualityRules(req.TestData, []models.QualityRuleConfig{config})
	if err != nil {
		return &TestRuleResponse{
			TestID:          testID,
			TotalRules:      1,
			SuccessfulRules: 0,
			FailedRules:     1,
			OverallSuccess:  false,
			ExecutionTime:   time.Since(startTime).Milliseconds(),
			Results: []RuleTestResult{{
				RuleType:       "quality",
				RuleTemplateID: req.RuleTemplateID,
				RuleName:       template.Name,
				Success:        false,
				OriginalData:   req.TestData,
				ExecutionTime:  time.Since(startTime).Milliseconds(),
				ErrorMessage:   err.Error(),
			}},
			Summary: createTestSummary(1, 0, 0, 0),
		}, nil
	}

	// 构建测试结果
	testResult := RuleTestResult{
		RuleType:       "quality",
		RuleTemplateID: req.RuleTemplateID,
		RuleName:       template.Name,
		Success:        result.Success,
		ProcessedData:  result.ProcessedData,
		OriginalData:   req.TestData,
		QualityScore:   &result.QualityScore,
		Issues:         result.Issues,
		Modifications:  result.Modifications,
		ExecutionTime:  result.ExecutionTime.Milliseconds(),
		ErrorMessage:   result.ErrorMessage,
	}

	successCount := 0
	if result.Success {
		successCount = 1
	}

	response := &TestRuleResponse{
		TestID:          testID,
		TotalRules:      1,
		SuccessfulRules: successCount,
		FailedRules:     1 - successCount,
		OverallSuccess:  result.Success,
		ExecutionTime:   time.Since(startTime).Milliseconds(),
		Results:         []RuleTestResult{testResult},
		Summary:         createTestSummary(1, 0, 0, result.QualityScore),
	}

	return response, nil
}

// TestMaskingRule 测试数据脱敏规则
func (s *GovernanceService) TestMaskingRule(req *TestMaskingRuleRequest) (*TestRuleResponse, error) {
	startTime := time.Now()
	testID := uuid.New().String()

	// 获取脱敏模板
	template, err := s.GetMaskingRuleByID(req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("获取脱敏规则模板失败: %v", err)
	}

	// 构建脱敏规则配置
	config := models.DataMaskingConfig{
		TemplateID:     req.TemplateID,
		TargetFields:   req.TargetFields,
		MaskingConfig:  models.JSONB(req.MaskingConfig),
		PreserveFormat: req.PreserveFormat,
		IsEnabled:      true,
	}

	// 执行脱敏规则
	result, err := s.ruleEngine.ApplyMaskingRules(req.TestData, []models.DataMaskingConfig{config})
	if err != nil {
		return &TestRuleResponse{
			TestID:          testID,
			TotalRules:      1,
			SuccessfulRules: 0,
			FailedRules:     1,
			OverallSuccess:  false,
			ExecutionTime:   time.Since(startTime).Milliseconds(),
			Results: []RuleTestResult{{
				RuleType:       "masking",
				RuleTemplateID: req.TemplateID,
				RuleName:       template.Name,
				Success:        false,
				OriginalData:   req.TestData,
				ExecutionTime:  time.Since(startTime).Milliseconds(),
				ErrorMessage:   err.Error(),
			}},
			Summary: createTestSummary(0, 1, 0, 0),
		}, nil
	}

	// 构建测试结果
	testResult := RuleTestResult{
		RuleType:       "masking",
		RuleTemplateID: req.TemplateID,
		RuleName:       template.Name,
		Success:        result.Success,
		ProcessedData:  result.ProcessedData,
		OriginalData:   req.TestData,
		Issues:         result.Issues,
		Modifications:  result.Modifications,
		ExecutionTime:  result.ExecutionTime.Milliseconds(),
		ErrorMessage:   result.ErrorMessage,
	}

	successCount := 0
	if result.Success {
		successCount = 1
	}

	response := &TestRuleResponse{
		TestID:          testID,
		TotalRules:      1,
		SuccessfulRules: successCount,
		FailedRules:     1 - successCount,
		OverallSuccess:  result.Success,
		ExecutionTime:   time.Since(startTime).Milliseconds(),
		Results:         []RuleTestResult{testResult},
		Summary:         createTestSummary(0, 1, 0, 0),
	}

	return response, nil
}

// TestCleansingRule 测试数据清洗规则
func (s *GovernanceService) TestCleansingRule(req *TestCleansingRuleRequest) (*TestRuleResponse, error) {
	startTime := time.Now()
	testID := uuid.New().String()

	// 获取清洗模板
	template, err := s.GetCleansingRuleByID(req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("获取清洗规则模板失败: %v", err)
	}

	// 构建清洗规则配置
	config := models.DataCleansingConfig{
		TemplateID:       req.TemplateID,
		TargetFields:     req.TargetFields,
		CleansingConfig:  req.CleansingConfig,
		TriggerCondition: req.TriggerCondition,
		BackupOriginal:   req.BackupOriginal,
		IsEnabled:        true,
	}

	// 执行清洗规则
	result, err := s.ruleEngine.ApplyCleansingRules(req.TestData, []models.DataCleansingConfig{config})
	if err != nil {
		return &TestRuleResponse{
			TestID:          testID,
			TotalRules:      1,
			SuccessfulRules: 0,
			FailedRules:     1,
			OverallSuccess:  false,
			ExecutionTime:   time.Since(startTime).Milliseconds(),
			Results: []RuleTestResult{{
				RuleType:       "cleansing",
				RuleTemplateID: req.TemplateID,
				RuleName:       template.Name,
				Success:        false,
				OriginalData:   req.TestData,
				ExecutionTime:  time.Since(startTime).Milliseconds(),
				ErrorMessage:   err.Error(),
			}},
			Summary: createTestSummary(0, 0, 1, 0),
		}, nil
	}

	// 构建测试结果
	testResult := RuleTestResult{
		RuleType:       "cleansing",
		RuleTemplateID: req.TemplateID,
		RuleName:       template.Name,
		Success:        result.Success,
		ProcessedData:  result.ProcessedData,
		OriginalData:   req.TestData,
		Issues:         result.Issues,
		Modifications:  result.Modifications,
		ExecutionTime:  result.ExecutionTime.Milliseconds(),
		ErrorMessage:   result.ErrorMessage,
	}

	successCount := 0
	if result.Success {
		successCount = 1
	}

	response := &TestRuleResponse{
		TestID:          testID,
		TotalRules:      1,
		SuccessfulRules: successCount,
		FailedRules:     1 - successCount,
		OverallSuccess:  result.Success,
		ExecutionTime:   time.Since(startTime).Milliseconds(),
		Results:         []RuleTestResult{testResult},
		Summary:         createTestSummary(0, 0, 1, 0),
	}

	return response, nil
}

// TestBatchRules 批量测试多个规则
func (s *GovernanceService) TestBatchRules(req *TestBatchRulesRequest) (*TestRuleResponse, error) {
	startTime := time.Now()
	testID := uuid.New().String()

	var results []RuleTestResult
	var totalRules, successfulRules, failedRules int
	var qualityChecks, maskingRules, cleansingRules int
	var overallScore float64
	var scoreCount int

	currentData := make(map[string]interface{})
	for k, v := range req.TestData {
		currentData[k] = v
	}

	// 根据执行顺序处理规则
	for _, ruleType := range req.ExecutionOrder {
		switch ruleType {
		case "quality":
			for _, qRule := range req.QualityRules {
				qualityChecks++
				totalRules++

				template, err := s.GetQualityRuleByID(qRule.RuleTemplateID)
				if err != nil {
					results = append(results, RuleTestResult{
						RuleType:       "quality",
						RuleTemplateID: qRule.RuleTemplateID,
						RuleName:       "未知规则",
						Success:        false,
						OriginalData:   currentData,
						ExecutionTime:  0,
						ErrorMessage:   fmt.Sprintf("获取规则模板失败: %v", err),
					})
					failedRules++
					continue
				}

				config := models.QualityRuleConfig{
					RuleTemplateID: qRule.RuleTemplateID,
					TargetFields:   qRule.TargetFields,
					RuntimeConfig:  models.JSONB(qRule.RuntimeConfig),
					Threshold:      models.JSONB(qRule.Threshold),
					IsEnabled:      true,
				}

				result, err := s.ruleEngine.ApplyQualityRules(currentData, []models.QualityRuleConfig{config})
				if err != nil {
					results = append(results, RuleTestResult{
						RuleType:       "quality",
						RuleTemplateID: qRule.RuleTemplateID,
						RuleName:       template.Name,
						Success:        false,
						OriginalData:   currentData,
						ExecutionTime:  0,
						ErrorMessage:   err.Error(),
					})
					failedRules++
					continue
				}

				results = append(results, RuleTestResult{
					RuleType:       "quality",
					RuleTemplateID: qRule.RuleTemplateID,
					RuleName:       template.Name,
					Success:        result.Success,
					ProcessedData:  result.ProcessedData,
					OriginalData:   currentData,
					QualityScore:   &result.QualityScore,
					Issues:         result.Issues,
					Modifications:  result.Modifications,
					ExecutionTime:  result.ExecutionTime.Milliseconds(),
					ErrorMessage:   result.ErrorMessage,
				})

				if result.Success {
					successfulRules++
					overallScore += result.QualityScore
					scoreCount++
				} else {
					failedRules++
				}
			}

		case "cleansing":
			for _, cRule := range req.CleansingRules {
				cleansingRules++
				totalRules++

				template, err := s.GetCleansingRuleByID(cRule.TemplateID)
				if err != nil {
					results = append(results, RuleTestResult{
						RuleType:       "cleansing",
						RuleTemplateID: cRule.TemplateID,
						RuleName:       "未知规则",
						Success:        false,
						OriginalData:   currentData,
						ExecutionTime:  0,
						ErrorMessage:   fmt.Sprintf("获取规则模板失败: %v", err),
					})
					failedRules++
					continue
				}

				config := models.DataCleansingConfig{
					TemplateID:       cRule.TemplateID,
					TargetFields:     cRule.TargetFields,
					CleansingConfig:  cRule.CleansingConfig,
					TriggerCondition: cRule.TriggerCondition,
					BackupOriginal:   cRule.BackupOriginal,
					IsEnabled:        true,
				}

				result, err := s.ruleEngine.ApplyCleansingRules(currentData, []models.DataCleansingConfig{config})
				if err != nil {
					results = append(results, RuleTestResult{
						RuleType:       "cleansing",
						RuleTemplateID: cRule.TemplateID,
						RuleName:       template.Name,
						Success:        false,
						OriginalData:   currentData,
						ExecutionTime:  0,
						ErrorMessage:   err.Error(),
					})
					failedRules++
					continue
				}

				results = append(results, RuleTestResult{
					RuleType:       "cleansing",
					RuleTemplateID: cRule.TemplateID,
					RuleName:       template.Name,
					Success:        result.Success,
					ProcessedData:  result.ProcessedData,
					OriginalData:   currentData,
					Issues:         result.Issues,
					Modifications:  result.Modifications,
					ExecutionTime:  result.ExecutionTime.Milliseconds(),
					ErrorMessage:   result.ErrorMessage,
				})

				if result.Success {
					successfulRules++
					// 更新当前数据为处理后的数据，供后续规则使用
					currentData = result.ProcessedData
				} else {
					failedRules++
				}
			}

		case "masking":
			for _, mRule := range req.MaskingRules {
				maskingRules++
				totalRules++

				template, err := s.GetMaskingRuleByID(mRule.TemplateID)
				if err != nil {
					results = append(results, RuleTestResult{
						RuleType:       "masking",
						RuleTemplateID: mRule.TemplateID,
						RuleName:       "未知规则",
						Success:        false,
						OriginalData:   currentData,
						ExecutionTime:  0,
						ErrorMessage:   fmt.Sprintf("获取规则模板失败: %v", err),
					})
					failedRules++
					continue
				}

				config := models.DataMaskingConfig{
					TemplateID:     mRule.TemplateID,
					TargetFields:   mRule.TargetFields,
					MaskingConfig:  models.JSONB(mRule.MaskingConfig),
					PreserveFormat: mRule.PreserveFormat,
					IsEnabled:      true,
				}

				result, err := s.ruleEngine.ApplyMaskingRules(currentData, []models.DataMaskingConfig{config})
				if err != nil {
					results = append(results, RuleTestResult{
						RuleType:       "masking",
						RuleTemplateID: mRule.TemplateID,
						RuleName:       template.Name,
						Success:        false,
						OriginalData:   currentData,
						ExecutionTime:  0,
						ErrorMessage:   err.Error(),
					})
					failedRules++
					continue
				}

				results = append(results, RuleTestResult{
					RuleType:       "masking",
					RuleTemplateID: mRule.TemplateID,
					RuleName:       template.Name,
					Success:        result.Success,
					ProcessedData:  result.ProcessedData,
					OriginalData:   currentData,
					Issues:         result.Issues,
					Modifications:  result.Modifications,
					ExecutionTime:  result.ExecutionTime.Milliseconds(),
					ErrorMessage:   result.ErrorMessage,
				})

				if result.Success {
					successfulRules++
					// 更新当前数据为处理后的数据，供后续规则使用
					currentData = result.ProcessedData
				} else {
					failedRules++
				}
			}
		}
	}

	// 计算平均分数
	if scoreCount > 0 {
		overallScore = overallScore / float64(scoreCount)
	}

	// 生成建议
	var recommendations []string
	if failedRules > 0 {
		recommendations = append(recommendations, fmt.Sprintf("有%d个规则执行失败，建议检查规则配置", failedRules))
	}
	if overallScore < 0.8 && scoreCount > 0 {
		recommendations = append(recommendations, "数据质量评分较低，建议优化数据质量规则")
	}

	response := &TestRuleResponse{
		TestID:          testID,
		TotalRules:      totalRules,
		SuccessfulRules: successfulRules,
		FailedRules:     failedRules,
		OverallSuccess:  failedRules == 0,
		ExecutionTime:   time.Since(startTime).Milliseconds(),
		Results:         results,
		Summary:         createTestSummary(qualityChecks, maskingRules, cleansingRules, overallScore),
		Recommendations: recommendations,
	}

	return response, nil
}

// TestRulePreview 预览规则执行效果（不实际执行）
func (s *GovernanceService) TestRulePreview(req *TestRulePreviewRequest) (*TestRulePreviewResponse, error) {
	var templateName string
	var expectedChanges []string
	configValidation := struct {
		IsValid bool     `json:"is_valid" example:"true"`
		Issues  []string `json:"issues,omitempty" example:"[\"阈值配置缺失\"]"`
	}{IsValid: true}

	estimatedImpact := struct {
		AffectedFields int     `json:"affected_fields" example:"2"`
		RiskLevel      string  `json:"risk_level" example:"low" enums:"low,medium,high"`
		Confidence     float64 `json:"confidence" example:"0.9"`
	}{
		AffectedFields: len(req.TargetFields),
		RiskLevel:      "low",
		Confidence:     0.9,
	}

	switch req.RuleType {
	case "quality":
		template, err := s.GetQualityRuleByID(req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("获取质量规则模板失败: %v", err)
		}
		templateName = template.Name

		// 分析预期变化
		for _, field := range req.TargetFields {
			switch template.Type {
			case "completeness":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被检查是否为空", field))
			case "accuracy":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被检查数据准确性", field))
			case "validity":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被检查格式有效性", field))
			default:
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将应用%s质量检查", field, template.Type))
			}
		}

		// 配置验证
		if req.Configuration == nil {
			configValidation.Issues = append(configValidation.Issues, "缺少运行时配置")
		}

	case "masking":
		template, err := s.GetMaskingRuleByID(req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("获取脱敏规则模板失败: %v", err)
		}
		templateName = template.Name
		estimatedImpact.RiskLevel = "medium"

		// 分析预期变化
		for _, field := range req.TargetFields {
			switch template.MaskingType {
			case "mask":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被部分掩码处理", field))
			case "replace":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被替换为固定值", field))
			case "encrypt":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被加密处理", field))
			case "pseudonymize":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被假名化处理", field))
			}
		}

	case "cleansing":
		template, err := s.GetCleansingRuleByID(req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("获取清洗规则模板失败: %v", err)
		}
		templateName = template.Name

		// 分析预期变化
		for _, field := range req.TargetFields {
			switch template.RuleType {
			case "standardization":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将被标准化处理", field))
			case "deduplication":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将进行去重处理", field))
			case "validation":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将进行数据验证", field))
			case "transformation":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将进行数据转换", field))
			case "enrichment":
				expectedChanges = append(expectedChanges, fmt.Sprintf("字段%s将进行数据丰富化", field))
			}
		}
	}

	// 创建预览结果（模拟处理后的数据）
	previewResult := make(map[string]interface{})
	for k, v := range req.SampleData {
		previewResult[k] = v
	}

	// 为目标字段添加预览标记
	for _, field := range req.TargetFields {
		if originalValue, exists := req.SampleData[field]; exists {
			switch req.RuleType {
			case "masking":
				if str, ok := originalValue.(string); ok && len(str) > 0 {
					previewResult[field] = fmt.Sprintf("[脱敏处理]%s", str[:1]+"***")
				}
			case "cleansing":
				if str, ok := originalValue.(string); ok {
					previewResult[field] = fmt.Sprintf("[清洗处理]%s", strings.ToLower(str))
				}
			case "quality":
				previewResult[field+"_质量评分"] = "[将计算质量评分]"
			}
		}
	}

	response := &TestRulePreviewResponse{
		RuleType:         req.RuleType,
		RuleName:         templateName,
		OriginalData:     req.SampleData,
		PreviewResult:    previewResult,
		ExpectedChanges:  expectedChanges,
		ConfigValidation: configValidation,
		EstimatedImpact:  estimatedImpact,
	}

	return response, nil
}
