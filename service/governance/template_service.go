/*
 * @module service/governance/template_service
 * @description 数据治理模板服务，提供规则模板和脱敏模板的管理功能
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow 模板管理生命周期
 * @rules 提供模板化的数据治理规则管理，支持模板创建、应用和执行
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/models/governance.go, service/models/quality_models.go
 */

package governance

import (
	"datahub-service/service/models"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// TemplateService 数据治理模板服务
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService 创建数据治理模板服务实例
func NewTemplateService(db *gorm.DB) *TemplateService {
	service := &TemplateService{db: db}
	// 初始化内置规则模板
	service.initializeBuiltinTemplates()
	return service
}

// === 数据质量规则模板管理 ===

// CreateQualityRuleTemplate 创建数据质量规则模板
func (s *TemplateService) CreateQualityRuleTemplate(template *models.QualityRuleTemplate) error {
	// 验证规则类型
	validTypes := []string{"completeness", "standardization", "consistency", "accuracy", "uniqueness", "timeliness"}
	isValidType := false
	for _, validType := range validTypes {
		if template.Type == validType {
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
		if template.Category == validCategory {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return errors.New("无效的规则模板分类")
	}

	return s.db.Create(template).Error
}

// GetQualityRuleTemplates 获取数据质量规则模板列表
func (s *TemplateService) GetQualityRuleTemplates(page, pageSize int, ruleType, category string, isBuiltIn *bool) ([]models.QualityRuleTemplate, int64, error) {
	var templates []models.QualityRuleTemplate
	var total int64

	query := s.db.Model(&models.QualityRuleTemplate{})

	if ruleType != "" {
		query = query.Where("type = ?", ruleType)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if isBuiltIn != nil {
		query = query.Where("is_built_in = ?", *isBuiltIn)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("is_built_in DESC, created_at DESC").Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// GetQualityRuleTemplateByID 根据ID获取数据质量规则模板
func (s *TemplateService) GetQualityRuleTemplateByID(id string) (*models.QualityRuleTemplate, error) {
	var template models.QualityRuleTemplate
	if err := s.db.First(&template, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateQualityRuleTemplate 更新数据质量规则模板
func (s *TemplateService) UpdateQualityRuleTemplate(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.QualityRuleTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteQualityRuleTemplate 删除数据质量规则模板
func (s *TemplateService) DeleteQualityRuleTemplate(id string) error {
	// 模板删除检查（直接应用模式下不需要检查应用实例）
	// 在直接应用模式下，模板只是定义，不会被持久化的应用实例引用

	return s.db.Delete(&models.QualityRuleTemplate{}, "id = ?", id).Error
}

// === 数据质量规则直接应用 ===
// 在重构后，数据质量规则直接应用到数据记录，不需要持久化的应用实例
// 相关功能已转移到 RuleEngine 中实现

// === 数据脱敏模板管理 ===

// CreateDataMaskingTemplate 创建数据脱敏模板
func (s *TemplateService) CreateDataMaskingTemplate(template *models.DataMaskingTemplate) error {
	// 验证脱敏类型
	validTypes := []string{"mask", "replace", "encrypt", "pseudonymize"}
	isValidType := false
	for _, validType := range validTypes {
		if template.MaskingType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的数据脱敏类型")
	}

	// 验证分类
	validCategories := []string{"personal_info", "financial", "medical", "business", "custom"}
	isValidCategory := false
	for _, validCategory := range validCategories {
		if template.Category == validCategory {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return errors.New("无效的脱敏模板分类")
	}

	// 验证安全级别
	validSecurityLevels := []string{"low", "medium", "high", "critical"}
	isValidSecurityLevel := false
	for _, validLevel := range validSecurityLevels {
		if template.SecurityLevel == validLevel {
			isValidSecurityLevel = true
			break
		}
	}
	if !isValidSecurityLevel {
		return errors.New("无效的安全级别")
	}

	return s.db.Create(template).Error
}

// GetDataMaskingTemplates 获取数据脱敏模板列表
func (s *TemplateService) GetDataMaskingTemplates(page, pageSize int, maskingType, category, securityLevel string, isBuiltIn *bool) ([]models.DataMaskingTemplate, int64, error) {
	var templates []models.DataMaskingTemplate
	var total int64

	query := s.db.Model(&models.DataMaskingTemplate{})

	if maskingType != "" {
		query = query.Where("masking_type = ?", maskingType)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if securityLevel != "" {
		query = query.Where("security_level = ?", securityLevel)
	}
	if isBuiltIn != nil {
		query = query.Where("is_built_in = ?", *isBuiltIn)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("is_built_in DESC, created_at DESC").Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// GetDataMaskingTemplateByID 根据ID获取数据脱敏模板
func (s *TemplateService) GetDataMaskingTemplateByID(id string) (*models.DataMaskingTemplate, error) {
	var template models.DataMaskingTemplate
	if err := s.db.First(&template, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateDataMaskingTemplate 更新数据脱敏模板
func (s *TemplateService) UpdateDataMaskingTemplate(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataMaskingTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDataMaskingTemplate 删除数据脱敏模板
func (s *TemplateService) DeleteDataMaskingTemplate(id string) error {
	// 模板删除检查（直接应用模式下不需要检查应用实例）
	// 在直接应用模式下，模板只是定义，不会被持久化的应用实例引用

	return s.db.Delete(&models.DataMaskingTemplate{}, "id = ?", id).Error
}

// === 数据脱敏直接应用 ===
// 在重构后，数据脱敏规则直接应用到数据记录，不需要持久化的应用实例
// 相关功能已转移到 RuleEngine 中实现

// === 数据清洗模板管理 ===

// CreateDataCleansingTemplate 创建数据清洗模板
func (s *TemplateService) CreateDataCleansingTemplate(template *models.DataCleansingTemplate) error {
	// 验证清洗规则类型
	validTypes := []string{"standardization", "deduplication", "validation", "transformation", "enrichment"}
	isValidType := false
	for _, validType := range validTypes {
		if template.RuleType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的数据清洗规则类型")
	}

	// 验证分类
	validCategories := []string{"data_format", "data_quality", "data_integrity"}
	isValidCategory := false
	for _, validCategory := range validCategories {
		if template.Category == validCategory {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return errors.New("无效的清洗模板分类")
	}

	return s.db.Create(template).Error
}

// GetDataCleansingTemplates 获取数据清洗模板列表
func (s *TemplateService) GetDataCleansingTemplates(page, pageSize int, ruleType, category string, isBuiltIn *bool) ([]models.DataCleansingTemplate, int64, error) {
	var templates []models.DataCleansingTemplate
	var total int64

	query := s.db.Model(&models.DataCleansingTemplate{})

	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if isBuiltIn != nil {
		query = query.Where("is_built_in = ?", *isBuiltIn)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("is_built_in DESC, created_at DESC").Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// GetDataCleansingTemplateByID 根据ID获取数据清洗模板
func (s *TemplateService) GetDataCleansingTemplateByID(id string) (*models.DataCleansingTemplate, error) {
	var template models.DataCleansingTemplate
	if err := s.db.First(&template, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateDataCleansingTemplate 更新数据清洗模板
func (s *TemplateService) UpdateDataCleansingTemplate(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataCleansingTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDataCleansingTemplate 删除数据清洗模板
func (s *TemplateService) DeleteDataCleansingTemplate(id string) error {
	// 模板删除检查（直接应用模式下不需要检查应用实例）
	// 在直接应用模式下，模板只是定义，不会被持久化的应用实例引用

	return s.db.Delete(&models.DataCleansingTemplate{}, "id = ?", id).Error
}

// initializeBuiltinTemplates 初始化内置规则模板
func (s *TemplateService) initializeBuiltinTemplates() {
	// 初始化数据质量规则模板
	s.initQualityRuleTemplates()
	// 初始化数据脱敏规则模板
	s.initMaskingRuleTemplates()
	// 初始化数据清洗规则模板
	s.initCleansingRuleTemplates()
}

// initQualityRuleTemplates 初始化数据质量规则模板
func (s *TemplateService) initQualityRuleTemplates() {
	qualityTemplates := []models.QualityRuleTemplate{
		// 完整性检查模板
		{
			ID:          "completeness_template_001",
			Name:        "字段完整性检查模板",
			Type:        "completeness",
			Category:    "basic_quality",
			Description: "检查指定字段是否为空、null或空字符串",
			RuleLogic: map[string]interface{}{
				"check_null":            true,
				"check_empty_string":    true,
				"check_whitespace_only": true,
			},
			Parameters: map[string]interface{}{
				"allow_zero": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "是否允许数值0",
				},
			},
			DefaultConfig: map[string]interface{}{
				"allow_zero": true,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "completeness",
			},
		},
		// 准确性检查模板
		{
			ID:          "accuracy_template_001",
			Name:        "邮箱格式准确性检查",
			Type:        "accuracy",
			Category:    "basic_quality",
			Description: "验证邮箱地址格式的准确性",
			RuleLogic: map[string]interface{}{
				"validation_type": "email",
				"regex_pattern":   "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$",
			},
			Parameters: map[string]interface{}{
				"strict_mode": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "是否启用严格模式验证",
				},
			},
			DefaultConfig: map[string]interface{}{
				"strict_mode": false,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "accuracy",
			},
		},
		// 一致性检查模板
		{
			ID:          "consistency_template_001",
			Name:        "跨字段一致性检查",
			Type:        "consistency",
			Category:    "basic_quality",
			Description: "检查相关字段之间的数据一致性",
			RuleLogic: map[string]interface{}{
				"check_type":   "cross_field",
				"relationship": "dependent",
			},
			Parameters: map[string]interface{}{
				"reference_field": map[string]interface{}{
					"type":        "string",
					"description": "参考字段名",
				},
			},
			DefaultConfig: map[string]interface{}{
				"tolerance": 0.05,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "consistency",
			},
		},
		// 有效性检查模板
		{
			ID:          "validity_template_001",
			Name:        "手机号有效性检查",
			Type:        "validity",
			Category:    "basic_quality",
			Description: "验证手机号格式的有效性",
			RuleLogic: map[string]interface{}{
				"validation_type": "phone",
				"regex_pattern":   "^1[3-9]\\d{9}$",
				"country_code":    "CN",
			},
			Parameters: map[string]interface{}{
				"allow_international": map[string]interface{}{
					"type":        "boolean",
					"default":     false,
					"description": "是否允许国际号码",
				},
			},
			DefaultConfig: map[string]interface{}{
				"allow_international": false,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "validity",
			},
		},
		// 唯一性检查模板
		{
			ID:          "uniqueness_template_001",
			Name:        "字段唯一性检查",
			Type:        "uniqueness",
			Category:    "basic_quality",
			Description: "检查字段值在数据集中的唯一性",
			RuleLogic: map[string]interface{}{
				"check_scope":    "dataset",
				"case_sensitive": true,
			},
			Parameters: map[string]interface{}{
				"ignore_null": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "是否忽略空值",
				},
			},
			DefaultConfig: map[string]interface{}{
				"ignore_null": true,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "uniqueness",
			},
		},
		// 及时性检查模板
		{
			ID:          "timeliness_template_001",
			Name:        "数据及时性检查",
			Type:        "timeliness",
			Category:    "basic_quality",
			Description: "检查数据的时效性",
			RuleLogic: map[string]interface{}{
				"check_type":   "age",
				"time_unit":    "days",
				"max_age_days": 30,
			},
			Parameters: map[string]interface{}{
				"time_field": map[string]interface{}{
					"type":        "string",
					"description": "时间字段名",
				},
				"max_age_days": map[string]interface{}{
					"type":        "number",
					"default":     30,
					"description": "最大天数",
				},
			},
			DefaultConfig: map[string]interface{}{
				"max_age_days": 30,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "timeliness",
			},
		},
		// 标准化检查模板
		{
			ID:          "standardization_template_001",
			Name:        "数据格式标准化检查",
			Type:        "standardization",
			Category:    "data_cleansing",
			Description: "检查数据格式是否符合标准化要求",
			RuleLogic: map[string]interface{}{
				"format_type":     "string",
				"standard_format": "lowercase",
				"trim_spaces":     true,
			},
			Parameters: map[string]interface{}{
				"standard_format": map[string]interface{}{
					"type":        "string",
					"description": "标准格式",
					"enum":        []string{"uppercase", "lowercase", "title_case"},
					"default":     "lowercase",
				},
			},
			DefaultConfig: map[string]interface{}{
				"standard_format": "lowercase",
				"trim_spaces":     true,
			},
			IsBuiltIn: true,
			IsEnabled: true,
			Version:   "1.0",
			Tags: map[string]interface{}{
				"category": "quality",
				"type":     "standardization",
			},
		},
	}

	// 批量插入或更新质量规则模板
	for _, template := range qualityTemplates {
		var existingTemplate models.QualityRuleTemplate
		result := s.db.Where("name = ? AND is_built_in = ?", template.Name, true).First(&existingTemplate)

		if result.Error != nil {
			// 模板不存在，创建新模板（让BeforeCreate钩子生成UUID）
			template.ID = "" // 清空ID，让BeforeCreate钩子生成UUID
			if err := s.db.Create(&template).Error; err != nil {
				fmt.Printf("创建内置质量规则模板失败: %s, 错误: %v\n", template.Name, err)
			}
		} else {
			// 模板已存在，更新非用户修改的字段
			if existingTemplate.IsBuiltIn {
				updates := map[string]interface{}{
					"description":    template.Description,
					"rule_logic":     template.RuleLogic,
					"parameters":     template.Parameters,
					"default_config": template.DefaultConfig,
					"version":        template.Version,
					"tags":           template.Tags,
				}
				if err := s.db.Model(&existingTemplate).Updates(updates).Error; err != nil {
					fmt.Printf("更新内置质量规则模板失败: %s, 错误: %v\n", template.Name, err)
				}
			}
		}
	}
}

// initMaskingRuleTemplates 初始化数据脱敏规则模板
func (s *TemplateService) initMaskingRuleTemplates() {
	maskingTemplates := []models.DataMaskingTemplate{
		// 手机号掩码脱敏模板
		{
			ID:              "masking_template_001",
			Name:            "手机号掩码脱敏",
			MaskingType:     "mask",
			Category:        "personal_info",
			Description:     "对手机号进行掩码处理，保留前3位和后4位",
			ApplicableTypes: pq.StringArray{"varchar", "char", "text"},
			MaskingLogic: map[string]interface{}{
				"keep_start": 3,
				"keep_end":   4,
				"mask_char":  "*",
			},
			Parameters: map[string]interface{}{
				"mask_char": map[string]interface{}{
					"type":        "string",
					"default":     "*",
					"description": "掩码字符",
				},
				"keep_start": map[string]interface{}{
					"type":        "number",
					"default":     3,
					"description": "保留开头字符数",
				},
				"keep_end": map[string]interface{}{
					"type":        "number",
					"default":     4,
					"description": "保留结尾字符数",
				},
			},
			DefaultConfig: map[string]interface{}{
				"preserve_format": true,
			},
			SecurityLevel: "medium",
			IsBuiltIn:     true,
			IsEnabled:     true,
			Version:       "1.0",
			Tags: map[string]interface{}{
				"category": "masking",
				"type":     "phone",
			},
		},
		// 身份证号替换脱敏模板
		{
			ID:              "masking_template_002",
			Name:            "身份证号替换脱敏",
			MaskingType:     "replace",
			Category:        "personal_info",
			Description:     "将身份证号完全替换为固定字符串",
			ApplicableTypes: pq.StringArray{"varchar", "char"},
			MaskingLogic: map[string]interface{}{
				"replacement": "***************",
			},
			Parameters: map[string]interface{}{
				"replacement": map[string]interface{}{
					"type":        "string",
					"default":     "***************",
					"description": "替换值",
				},
			},
			DefaultConfig: map[string]interface{}{
				"replacement": "***************",
			},
			SecurityLevel: "high",
			IsBuiltIn:     true,
			IsEnabled:     true,
			Version:       "1.0",
			Tags: map[string]interface{}{
				"category": "masking",
				"type":     "id_card",
			},
		},
		// 银行卡号加密脱敏模板
		{
			ID:              "masking_template_003",
			Name:            "银行卡号加密脱敏",
			MaskingType:     "encrypt",
			Category:        "financial",
			Description:     "对银行卡号进行加密处理",
			ApplicableTypes: pq.StringArray{"varchar", "char"},
			MaskingLogic: map[string]interface{}{
				"encryption_algorithm": "AES256",
				"key_rotation":         true,
			},
			Parameters: map[string]interface{}{
				"algorithm": map[string]interface{}{
					"type":        "string",
					"default":     "AES256",
					"description": "加密算法",
				},
			},
			DefaultConfig: map[string]interface{}{
				"algorithm": "AES256",
			},
			SecurityLevel: "critical",
			IsBuiltIn:     true,
			IsEnabled:     true,
			Version:       "1.0",
			Tags: map[string]interface{}{
				"category": "masking",
				"type":     "bank_card",
			},
		},
		// 姓名假名化脱敏模板
		{
			ID:              "masking_template_004",
			Name:            "姓名假名化脱敏",
			MaskingType:     "pseudonymize",
			Category:        "personal_info",
			Description:     "将真实姓名替换为假名",
			ApplicableTypes: pq.StringArray{"varchar", "char", "text"},
			MaskingLogic: map[string]interface{}{
				"pseudonym_type": "consistent_hash",
				"name_pool":      []string{"用户A", "用户B", "用户C", "用户D", "用户E"},
			},
			Parameters: map[string]interface{}{
				"name_pool_size": map[string]interface{}{
					"type":        "integer",
					"default":     100,
					"description": "假名池大小",
				},
			},
			DefaultConfig: map[string]interface{}{
				"consistent_mapping": true,
			},
			SecurityLevel: "medium",
			IsBuiltIn:     true,
			IsEnabled:     true,
			Version:       "1.0",
			Tags: map[string]interface{}{
				"category": "masking",
				"type":     "name",
			},
		},
	}

	// 批量插入或更新脱敏规则模板
	for _, template := range maskingTemplates {
		var existingTemplate models.DataMaskingTemplate
		result := s.db.Where("name = ? AND is_built_in = ?", template.Name, true).First(&existingTemplate)

		if result.Error != nil {
			// 模板不存在，创建新模板（让BeforeCreate钩子生成UUID）
			template.ID = "" // 清空ID，让BeforeCreate钩子生成UUID
			if err := s.db.Create(&template).Error; err != nil {
				fmt.Printf("创建内置脱敏规则模板失败: %s, 错误: %v\n", template.Name, err)
			}
		} else {
			// 模板已存在，更新非用户修改的字段
			if existingTemplate.IsBuiltIn {
				updates := map[string]interface{}{
					"description":      template.Description,
					"applicable_types": template.ApplicableTypes,
					"masking_logic":    template.MaskingLogic,
					"parameters":       template.Parameters,
					"default_config":   template.DefaultConfig,
					"version":          template.Version,
					"tags":             template.Tags,
				}
				if err := s.db.Model(&existingTemplate).Updates(updates).Error; err != nil {
					fmt.Printf("更新内置脱敏规则模板失败: %s, 错误: %v\n", template.Name, err)
				}
			}
		}
	}
}

// initCleansingRuleTemplates 初始化数据清洗规则模板
func (s *TemplateService) initCleansingRuleTemplates() {
	cleansingTemplates := []models.DataCleansingTemplate{
		// 邮箱标准化清洗模板
		{
			ID:          "cleansing_template_001",
			Name:        "邮箱标准化清洗",
			RuleType:    "standardization",
			Category:    "data_format",
			Description: "将邮箱地址标准化为小写格式",
			CleansingLogic: models.JSONB(map[string]interface{}{
				"operations": []map[string]interface{}{
					{"type": "trim", "target": "whitespace"},
					{"type": "case_convert", "target": "lower"},
					{"type": "format_validate", "pattern": "email"},
				},
			}),
			Parameters: models.JSONB(map[string]interface{}{
				"case": map[string]interface{}{
					"type":        "string",
					"default":     "lower",
					"description": "大小写转换",
				},
			}),
			DefaultConfig: models.JSONB(map[string]interface{}{
				"case":        "lower",
				"trim_spaces": true,
			}),
			ApplicableTypes: models.JSONB(map[string]interface{}{
				"types": []string{"email", "varchar", "text"},
			}),
			ComplexityLevel: "low",
			IsBuiltIn:       true,
			IsEnabled:       true,
			Version:         "1.0",
			Tags: models.JSONB(map[string]interface{}{
				"category": "cleansing",
				"type":     "email",
			}),
		},
		// 重复字符去除模板
		{
			ID:          "cleansing_template_002",
			Name:        "重复字符去除",
			RuleType:    "deduplication",
			Category:    "data_quality",
			Description: "去除字符串中的重复字符",
			CleansingLogic: models.JSONB(map[string]interface{}{
				"dedup_type":     "consecutive_chars",
				"preserve_order": true,
			}),
			Parameters: models.JSONB(map[string]interface{}{
				"preserve_order": map[string]interface{}{
					"type":        "boolean",
					"default":     true,
					"description": "保持字符顺序",
				},
			}),
			DefaultConfig: models.JSONB(map[string]interface{}{
				"preserve_order": true,
			}),
			ApplicableTypes: models.JSONB(map[string]interface{}{
				"types": []string{"varchar", "char", "text"},
			}),
			ComplexityLevel: "medium",
			IsBuiltIn:       true,
			IsEnabled:       true,
			Version:         "1.0",
			Tags: models.JSONB(map[string]interface{}{
				"category": "cleansing",
				"type":     "deduplication",
			}),
		},
		// 数据格式验证修正模板
		{
			ID:          "cleansing_template_003",
			Name:        "数据格式验证修正",
			RuleType:    "validation",
			Category:    "data_integrity",
			Description: "验证数据格式并进行修正",
			CleansingLogic: models.JSONB(map[string]interface{}{
				"validation_rules": []map[string]interface{}{
					{"field_type": "phone", "pattern": "^1[3-9]\\d{9}$"},
					{"field_type": "email", "pattern": "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"},
				},
				"correction_strategy": "default_value",
			}),
			Parameters: models.JSONB(map[string]interface{}{
				"correction_strategy": map[string]interface{}{
					"type":        "string",
					"default":     "default_value",
					"description": "修正策略",
				},
			}),
			DefaultConfig: models.JSONB(map[string]interface{}{
				"correction_strategy": "default_value",
			}),
			ApplicableTypes: models.JSONB(map[string]interface{}{
				"types": []string{"phone", "email", "varchar"},
			}),
			ComplexityLevel: "high",
			IsBuiltIn:       true,
			IsEnabled:       true,
			Version:         "1.0",
			Tags: models.JSONB(map[string]interface{}{
				"category": "cleansing",
				"type":     "validation",
			}),
		},
		// 日期格式转换模板
		{
			ID:          "cleansing_template_004",
			Name:        "日期格式转换",
			RuleType:    "transformation",
			Category:    "data_format",
			Description: "将日期格式从一种格式转换为另一种格式",
			CleansingLogic: models.JSONB(map[string]interface{}{
				"transform_type": "date_format",
				"source_formats": []string{"yyyy-MM-dd", "yyyy/MM/dd", "dd-MM-yyyy"},
				"target_format":  "yyyy-MM-dd",
			}),
			Parameters: models.JSONB(map[string]interface{}{
				"target_format": map[string]interface{}{
					"type":        "string",
					"default":     "yyyy-MM-dd",
					"description": "目标日期格式",
				},
			}),
			DefaultConfig: models.JSONB(map[string]interface{}{
				"target_format": "yyyy-MM-dd",
			}),
			ApplicableTypes: models.JSONB(map[string]interface{}{
				"types": []string{"date", "datetime", "varchar"},
			}),
			ComplexityLevel: "medium",
			IsBuiltIn:       true,
			IsEnabled:       true,
			Version:         "1.0",
			Tags: models.JSONB(map[string]interface{}{
				"category": "cleansing",
				"type":     "date",
			}),
		},
		// 数据丰富模板
		{
			ID:          "cleansing_template_005",
			Name:        "地址信息丰富",
			RuleType:    "enrichment",
			Category:    "data_enhancement",
			Description: "根据邮编或地址信息丰富地理数据",
			CleansingLogic: models.JSONB(map[string]interface{}{
				"enrichment_type": "address",
				"data_source":     "geo_api",
				"fields_to_add":   []string{"province", "city", "district"},
			}),
			Parameters: models.JSONB(map[string]interface{}{
				"data_source": map[string]interface{}{
					"type":        "string",
					"default":     "geo_api",
					"description": "数据源",
				},
			}),
			DefaultConfig: models.JSONB(map[string]interface{}{
				"data_source": "geo_api",
			}),
			ApplicableTypes: models.JSONB(map[string]interface{}{
				"types": []string{"address", "postcode", "varchar"},
			}),
			ComplexityLevel: "high",
			IsBuiltIn:       true,
			IsEnabled:       true,
			Version:         "1.0",
			Tags: models.JSONB(map[string]interface{}{
				"category": "cleansing",
				"type":     "enrichment",
			}),
		},
	}

	// 批量插入或更新清洗规则模板
	for _, template := range cleansingTemplates {
		var existingTemplate models.DataCleansingTemplate
		result := s.db.Where("name = ? AND is_built_in = ?", template.Name, true).First(&existingTemplate)

		if result.Error != nil {
			// 模板不存在，创建新模板（让BeforeCreate钩子生成UUID）
			template.ID = "" // 清空ID，让BeforeCreate钩子生成UUID
			if err := s.db.Create(&template).Error; err != nil {
				fmt.Printf("创建内置清洗规则模板失败: %s, 错误: %v\n", template.Name, err)
			}
		} else {
			// 模板已存在，更新非用户修改的字段
			if existingTemplate.IsBuiltIn {
				updates := map[string]interface{}{
					"description":      template.Description,
					"cleansing_logic":  template.CleansingLogic,
					"parameters":       template.Parameters,
					"default_config":   template.DefaultConfig,
					"applicable_types": template.ApplicableTypes,
					"version":          template.Version,
					"tags":             template.Tags,
				}
				if err := s.db.Model(&existingTemplate).Updates(updates).Error; err != nil {
					fmt.Printf("更新内置清洗规则模板失败: %s, 错误: %v\n", template.Name, err)
				}
			}
		}
	}
}

// === 数据清洗直接应用 ===
// 在重构后，数据清洗规则直接应用到数据记录，不需要持久化的应用实例
// 相关功能已转移到 RuleEngine 中实现
