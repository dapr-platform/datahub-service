/*
 * @module service/governance/tests/builtin_rules_test
 * @description 内置数据治理规则测试，不依赖数据库
 * @architecture 测试层
 * @documentReference ai_docs/data_governance_comprehensive_examples.md
 * @stateFlow 测试数据输入 -> 内置规则应用 -> 结果验证
 * @rules 测试所有内置规则模板的功能，确保规则引擎正常工作
 * @dependencies testing, datahub-service/service/governance, datahub-service/service/models
 * @refs rule_engine.go, template_service.go
 */

package tests

import (
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 创建内置质量规则模板
func createBuiltinQualityTemplates() map[string]*models.QualityRuleTemplate {
	templates := make(map[string]*models.QualityRuleTemplate)

	// 完整性检查模板
	templates["completeness_template_001"] = &models.QualityRuleTemplate{
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
	}

	// 准确性检查模板（邮箱）
	templates["accuracy_template_001"] = &models.QualityRuleTemplate{
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
	}

	// 一致性检查模板
	templates["consistency_template_001"] = &models.QualityRuleTemplate{
		ID:          "consistency_template_001",
		Name:        "字段一致性检查模板",
		Type:        "consistency",
		Category:    "basic_quality",
		Description: "检查字段值在不同记录间的一致性",
		RuleLogic: map[string]interface{}{
			"check_type": "cross_field",
			"reference":  "master_data",
		},
		Parameters: map[string]interface{}{
			"tolerance": map[string]interface{}{
				"type":        "number",
				"default":     0.1,
				"description": "允许的不一致容忍度",
			},
		},
		DefaultConfig: map[string]interface{}{
			"tolerance": 0.1,
		},
		IsBuiltIn: true,
		IsEnabled: true,
		Version:   "1.0",
	}

	// 有效性检查模板
	templates["validity_template_001"] = &models.QualityRuleTemplate{
		ID:          "validity_template_001",
		Name:        "数据有效性检查模板",
		Type:        "validity",
		Category:    "basic_quality",
		Description: "检查数据是否符合有效性规范",
		RuleLogic: map[string]interface{}{
			"check_type": "format",
			"format":     "phone",
		},
		Parameters: map[string]interface{}{
			"country_code": map[string]interface{}{
				"type":        "string",
				"default":     "CN",
				"description": "国家代码",
			},
		},
		DefaultConfig: map[string]interface{}{
			"country_code": "CN",
		},
		IsBuiltIn: true,
		IsEnabled: true,
		Version:   "1.0",
	}

	// 唯一性检查模板
	templates["uniqueness_template_001"] = &models.QualityRuleTemplate{
		ID:          "uniqueness_template_001",
		Name:        "字段唯一性检查模板",
		Type:        "uniqueness",
		Category:    "basic_quality",
		Description: "检查字段值的唯一性",
		RuleLogic: map[string]interface{}{
			"check_type": "unique",
			"scope":      "global",
		},
		Parameters: map[string]interface{}{
			"ignore_case": map[string]interface{}{
				"type":        "boolean",
				"default":     false,
				"description": "是否忽略大小写",
			},
		},
		DefaultConfig: map[string]interface{}{
			"ignore_case": false,
		},
		IsBuiltIn: true,
		IsEnabled: true,
		Version:   "1.0",
	}

	// 及时性检查模板
	templates["timeliness_template_001"] = &models.QualityRuleTemplate{
		ID:          "timeliness_template_001",
		Name:        "数据及时性检查模板",
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
	}

	// 标准化检查模板
	templates["standardization_template_001"] = &models.QualityRuleTemplate{
		ID:          "standardization_template_001",
		Name:        "数据标准化检查模板",
		Type:        "standardization",
		Category:    "data_cleansing",
		Description: "检查数据是否符合标准化格式",
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
	}

	return templates
}

// 创建内置脱敏规则模板
func createBuiltinMaskingTemplates() map[string]*models.DataMaskingTemplate {
	templates := make(map[string]*models.DataMaskingTemplate)

	// 手机号掩码模板
	templates["masking_template_001"] = &models.DataMaskingTemplate{
		ID:          "masking_template_001",
		Name:        "手机号掩码模板",
		MaskingType: "mask",
		Category:    "personal_info",
		Description: "对手机号进行掩码处理，保留前3位和后4位",
		MaskingLogic: map[string]interface{}{
			"keep_start": 3,
			"keep_end":   4,
			"mask_char":  "*",
		},
		Parameters: map[string]interface{}{
			"keep_start": map[string]interface{}{
				"type":        "number",
				"default":     3,
				"description": "保留开头字符数",
			},
		},
		DefaultConfig: map[string]interface{}{
			"keep_start": 3,
			"keep_end":   4,
			"mask_char":  "*",
		},
		SecurityLevel: "medium",
		IsBuiltIn:     true,
		IsEnabled:     true,
		Version:       "1.0",
	}

	// 邮箱替换模板
	templates["masking_template_002"] = &models.DataMaskingTemplate{
		ID:          "masking_template_002",
		Name:        "邮箱替换模板",
		MaskingType: "replace",
		Category:    "personal_info",
		Description: "用固定值替换邮箱地址",
		MaskingLogic: map[string]interface{}{
			"replacement": "***@***.com",
		},
		Parameters: map[string]interface{}{
			"replacement": map[string]interface{}{
				"type":        "string",
				"default":     "***@***.com",
				"description": "替换值",
			},
		},
		DefaultConfig: map[string]interface{}{
			"replacement": "***@***.com",
		},
		SecurityLevel: "high",
		IsBuiltIn:     true,
		IsEnabled:     true,
		Version:       "1.0",
	}

	// 身份证加密模板
	templates["masking_template_003"] = &models.DataMaskingTemplate{
		ID:          "masking_template_003",
		Name:        "身份证加密模板",
		MaskingType: "encrypt",
		Category:    "personal_info",
		Description: "对身份证号进行加密处理",
		MaskingLogic: map[string]interface{}{
			"algorithm": "AES256",
			"key":       "default_key_placeholder",
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
	}

	// 姓名假名化模板
	templates["masking_template_004"] = &models.DataMaskingTemplate{
		ID:          "masking_template_004",
		Name:        "姓名假名化模板",
		MaskingType: "pseudonymize",
		Category:    "personal_info",
		Description: "用假名替换真实姓名",
		MaskingLogic: map[string]interface{}{
			"prefix":      "用户",
			"use_hash":    true,
			"hash_length": 6,
		},
		Parameters: map[string]interface{}{
			"prefix": map[string]interface{}{
				"type":        "string",
				"default":     "用户",
				"description": "假名前缀",
			},
		},
		DefaultConfig: map[string]interface{}{
			"prefix":      "用户",
			"use_hash":    true,
			"hash_length": 6,
		},
		SecurityLevel: "medium",
		IsBuiltIn:     true,
		IsEnabled:     true,
		Version:       "1.0",
	}

	return templates
}

// 创建内置清洗规则模板
func createBuiltinCleansingTemplates() map[string]*models.DataCleansingTemplate {
	templates := make(map[string]*models.DataCleansingTemplate)

	// 邮箱标准化模板
	templates["cleansing_template_001"] = &models.DataCleansingTemplate{
		ID:          "cleansing_template_001",
		Name:        "邮箱格式标准化模板",
		Description: "统一邮箱格式为小写并验证格式",
		RuleType:    "standardization",
		Category:    "data_format",
		CleansingLogic: map[string]interface{}{
			"standardization_type": "email_lowercase",
			"trim_spaces":          true,
			"validate_format":      true,
		},
		Parameters: map[string]interface{}{
			"validate_format": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "是否验证邮箱格式",
			},
		},
		DefaultConfig: map[string]interface{}{
			"validate_format": true,
		},
		ComplexityLevel: "low",
		IsBuiltIn:       true,
		IsEnabled:       true,
		Version:         "1.0",
	}

	// 去重模板
	templates["cleansing_template_002"] = &models.DataCleansingTemplate{
		ID:          "cleansing_template_002",
		Name:        "数据去重模板",
		Description: "去除重复的数据记录",
		RuleType:    "deduplication",
		Category:    "data_quality",
		CleansingLogic: map[string]interface{}{
			"dedup_strategy": "keep_first",
			"ignore_case":    true,
		},
		Parameters: map[string]interface{}{
			"dedup_strategy": map[string]interface{}{
				"type":        "string",
				"default":     "keep_first",
				"description": "去重策略",
			},
		},
		DefaultConfig: map[string]interface{}{
			"dedup_strategy": "keep_first",
		},
		ComplexityLevel: "medium",
		IsBuiltIn:       true,
		IsEnabled:       true,
		Version:         "1.0",
	}

	// 数据验证模板
	templates["cleansing_template_003"] = &models.DataCleansingTemplate{
		ID:          "cleansing_template_003",
		Name:        "数据验证模板",
		Description: "验证数据的有效性并修复",
		RuleType:    "validation",
		Category:    "data_integrity",
		CleansingLogic: map[string]interface{}{
			"validation_type": "format",
			"auto_fix":        true,
		},
		Parameters: map[string]interface{}{
			"auto_fix": map[string]interface{}{
				"type":        "boolean",
				"default":     true,
				"description": "是否自动修复",
			},
		},
		DefaultConfig: map[string]interface{}{
			"auto_fix": true,
		},
		ComplexityLevel: "high",
		IsBuiltIn:       true,
		IsEnabled:       true,
		Version:         "1.0",
	}

	// 数据转换模板
	templates["cleansing_template_004"] = &models.DataCleansingTemplate{
		ID:          "cleansing_template_004",
		Name:        "数据转换模板",
		Description: "转换数据格式和类型",
		RuleType:    "transformation",
		Category:    "data_format",
		CleansingLogic: map[string]interface{}{
			"transform_type": "format_conversion",
			"target_format":  "standard",
		},
		Parameters: map[string]interface{}{
			"target_format": map[string]interface{}{
				"type":        "string",
				"default":     "standard",
				"description": "目标格式",
			},
		},
		DefaultConfig: map[string]interface{}{
			"target_format": "standard",
		},
		ComplexityLevel: "medium",
		IsBuiltIn:       true,
		IsEnabled:       true,
		Version:         "1.0",
	}

	// 数据丰富模板
	templates["cleansing_template_005"] = &models.DataCleansingTemplate{
		ID:          "cleansing_template_005",
		Name:        "数据丰富模板",
		Description: "补充缺失数据或增强数据",
		RuleType:    "enrichment",
		Category:    "data_quality",
		CleansingLogic: map[string]interface{}{
			"enrichment_type": "default_value",
			"default_value":   "N/A",
		},
		Parameters: map[string]interface{}{
			"default_value": map[string]interface{}{
				"type":        "string",
				"default":     "N/A",
				"description": "默认值",
			},
		},
		DefaultConfig: map[string]interface{}{
			"default_value": "N/A",
		},
		ComplexityLevel: "low",
		IsBuiltIn:       true,
		IsEnabled:       true,
		Version:         "1.0",
	}

	return templates
}

// 创建简单的测试数据库
func setupBuiltinTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	return db
}

// 测试完整性规则
func TestBuiltinCompletenessRule(t *testing.T) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	templates := createBuiltinQualityTemplates()

	// 测试数据
	testData := map[string]interface{}{
		"name":  "张三",
		"email": "zhangsan@example.com",
		"phone": "",
		"age":   0,
	}

	// 规则配置
	configs := []models.QualityRuleConfig{
		{
			RuleTemplateID: "completeness_template_001",
			TargetFields:   []string{"name", "email", "phone", "age"},
			RuntimeConfig:  map[string]interface{}{},
			Threshold:      map[string]interface{}{},
			IsEnabled:      true,
		},
	}

	// 应用规则
	result, err := re.ApplyQualityRulesWithTemplates(testData, configs, templates)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, len(result.RulesApplied))

	// 应该有一个问题：phone字段为空
	assert.Contains(t, result.Issues, "字段 phone 完整性检查失败: 字段为空字符串")

	// 质量分数应该是 3/4 = 0.75
	assert.Equal(t, 0.75, result.QualityScore)
}

// 测试准确性规则（邮箱格式）
func TestBuiltinAccuracyRule(t *testing.T) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	templates := createBuiltinQualityTemplates()

	// 测试数据
	testData := map[string]interface{}{
		"email1": "valid@example.com",
		"email2": "invalid-email",
		"email3": "another@test.org",
	}

	// 规则配置
	configs := []models.QualityRuleConfig{
		{
			RuleTemplateID: "accuracy_template_001",
			TargetFields:   []string{"email1", "email2", "email3"},
			RuntimeConfig:  map[string]interface{}{},
			Threshold:      map[string]interface{}{},
			IsEnabled:      true,
		},
	}

	// 应用规则
	result, err := re.ApplyQualityRulesWithTemplates(testData, configs, templates)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result.RulesApplied))

	// 应该有一个问题：email2格式不正确
	assert.Contains(t, result.Issues, "字段 email2 准确性检查失败: 邮箱格式不正确")

	// 质量分数应该是 2/3 ≈ 0.67
	assert.InDelta(t, 0.67, result.QualityScore, 0.01)
}

// 测试脱敏规则
func TestBuiltinMaskingRules(t *testing.T) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	templates := createBuiltinMaskingTemplates()

	// 测试数据
	testData := map[string]interface{}{
		"phone": "13800138000",
		"email": "user@example.com",
		"name":  "张三",
	}

	// 脱敏配置
	configs := []models.DataMaskingConfig{
		{
			TemplateID:   "masking_template_001",
			TargetFields: []string{"phone"},
			MaskingConfig: map[string]interface{}{
				"keep_start": 3,
				"keep_end":   4,
				"mask_char":  "*",
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "masking_template_002",
			TargetFields: []string{"email"},
			MaskingConfig: map[string]interface{}{
				"replacement": "***@***.com",
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "masking_template_004",
			TargetFields: []string{"name"},
			MaskingConfig: map[string]interface{}{
				"prefix":      "用户",
				"use_hash":    true,
				"hash_length": 6,
			},
			IsEnabled: true,
		},
	}

	// 应用脱敏规则
	result, err := re.ApplyMaskingRulesWithTemplates(testData, configs, templates)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result.RulesApplied))

	// 验证脱敏结果
	assert.Equal(t, "138****8000", result.ProcessedData["phone"])
	assert.Equal(t, "***@***.com", result.ProcessedData["email"])
	assert.Contains(t, result.ProcessedData["name"].(string), "用户")

	// 验证修改记录
	assert.Contains(t, result.Modifications, "phone")
	assert.Contains(t, result.Modifications, "email")
	assert.Contains(t, result.Modifications, "name")
}

// 测试清洗规则
func TestBuiltinCleansingRules(t *testing.T) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	templates := createBuiltinCleansingTemplates()

	// 测试数据
	testData := map[string]interface{}{
		"email":    "  TEST@EXAMPLE.COM  ",
		"name":     "张三",
		"status":   nil,
		"category": "PRODUCT",
	}

	// 清洗配置
	configs := []models.DataCleansingConfig{
		{
			TemplateID:   "cleansing_template_001",
			TargetFields: []string{"email"},
			CleansingConfig: map[string]interface{}{
				"standardization_type": "email_lowercase",
				"trim_spaces":          true,
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "cleansing_template_005",
			TargetFields: []string{"status"},
			CleansingConfig: map[string]interface{}{
				"default_value": "未知",
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "cleansing_template_004",
			TargetFields: []string{"category"},
			CleansingConfig: map[string]interface{}{
				"transform_type": "format_conversion",
				"target_format":  "lowercase",
			},
			IsEnabled: true,
		},
	}

	// 应用清洗规则
	result, err := re.ApplyCleansingRulesWithTemplates(testData, configs, templates)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result.RulesApplied))

	// 验证清洗结果
	assert.Equal(t, "test@example.com", result.ProcessedData["email"])
	assert.Equal(t, "未知", result.ProcessedData["status"])
	assert.Equal(t, "product", result.ProcessedData["category"])

	// 验证修改记录
	assert.Contains(t, result.Modifications, "email")
	assert.Contains(t, result.Modifications, "status")
	assert.Contains(t, result.Modifications, "category")
}

// 测试综合场景
func TestBuiltinComprehensiveScenario(t *testing.T) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	qualityTemplates := createBuiltinQualityTemplates()
	maskingTemplates := createBuiltinMaskingTemplates()
	cleansingTemplates := createBuiltinCleansingTemplates()

	// 测试数据
	testData := map[string]interface{}{
		"name":   "  张三  ",
		"email":  "  ZHANGSAN@EXAMPLE.COM  ",
		"phone":  "13800138000",
		"age":    25,
		"status": nil,
	}

	// 1. 首先应用清洗规则
	cleansingConfigs := []models.DataCleansingConfig{
		{
			TemplateID:   "cleansing_template_001",
			TargetFields: []string{"email"},
			CleansingConfig: map[string]interface{}{
				"standardization_type": "email_lowercase",
				"trim_spaces":          true,
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "cleansing_template_005",
			TargetFields: []string{"status"},
			CleansingConfig: map[string]interface{}{
				"default_value": "active",
			},
			IsEnabled: true,
		},
	}

	cleansingResult, err := re.ApplyCleansingRulesWithTemplates(testData, cleansingConfigs, cleansingTemplates)
	assert.NoError(t, err)
	assert.Equal(t, "zhangsan@example.com", cleansingResult.ProcessedData["email"])
	assert.Equal(t, "active", cleansingResult.ProcessedData["status"])

	// 2. 然后应用质量检查规则
	qualityConfigs := []models.QualityRuleConfig{
		{
			RuleTemplateID: "completeness_template_001",
			TargetFields:   []string{"name", "email", "phone", "age", "status"},
			RuntimeConfig:  map[string]interface{}{},
			Threshold:      map[string]interface{}{},
			IsEnabled:      true,
		},
		{
			RuleTemplateID: "accuracy_template_001",
			TargetFields:   []string{"email"},
			RuntimeConfig:  map[string]interface{}{},
			Threshold:      map[string]interface{}{},
			IsEnabled:      true,
		},
	}

	qualityResult, err := re.ApplyQualityRulesWithTemplates(cleansingResult.ProcessedData, qualityConfigs, qualityTemplates)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, qualityResult.QualityScore) // 所有字段都应该通过质量检查

	// 3. 最后应用脱敏规则
	maskingConfigs := []models.DataMaskingConfig{
		{
			TemplateID:   "masking_template_001",
			TargetFields: []string{"phone"},
			MaskingConfig: map[string]interface{}{
				"keep_start": 3,
				"keep_end":   4,
				"mask_char":  "*",
			},
			IsEnabled: true,
		},
	}

	maskingResult, err := re.ApplyMaskingRulesWithTemplates(qualityResult.ProcessedData, maskingConfigs, maskingTemplates)
	assert.NoError(t, err)
	assert.Equal(t, "138****8000", maskingResult.ProcessedData["phone"])

	// 验证最终结果
	assert.Equal(t, "zhangsan@example.com", maskingResult.ProcessedData["email"])
	assert.Equal(t, "active", maskingResult.ProcessedData["status"])
	assert.Equal(t, "138****8000", maskingResult.ProcessedData["phone"])
}

// 测试所有内置质量规则类型
func TestAllBuiltinQualityRuleTypes(t *testing.T) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	templates := createBuiltinQualityTemplates()

	testCases := []struct {
		name        string
		templateID  string
		data        map[string]interface{}
		field       string
		expectPass  bool
		expectIssue string
	}{
		{
			name:       "完整性检查-通过",
			templateID: "completeness_template_001",
			data:       map[string]interface{}{"name": "张三"},
			field:      "name",
			expectPass: true,
		},
		{
			name:        "完整性检查-失败",
			templateID:  "completeness_template_001",
			data:        map[string]interface{}{"name": ""},
			field:       "name",
			expectPass:  false,
			expectIssue: "完整性检查失败",
		},
		{
			name:       "准确性检查-通过",
			templateID: "accuracy_template_001",
			data:       map[string]interface{}{"email": "test@example.com"},
			field:      "email",
			expectPass: true,
		},
		{
			name:        "准确性检查-失败",
			templateID:  "accuracy_template_001",
			data:        map[string]interface{}{"email": "invalid-email"},
			field:       "email",
			expectPass:  false,
			expectIssue: "准确性检查失败",
		},
		{
			name:       "一致性检查-通过",
			templateID: "consistency_template_001",
			data:       map[string]interface{}{"status": "active"},
			field:      "status",
			expectPass: true,
		},
		{
			name:       "有效性检查-通过",
			templateID: "validity_template_001",
			data:       map[string]interface{}{"phone": "13800138000"},
			field:      "phone",
			expectPass: true,
		},
		{
			name:       "唯一性检查-通过",
			templateID: "uniqueness_template_001",
			data:       map[string]interface{}{"id": "unique123"},
			field:      "id",
			expectPass: true,
		},
		{
			name:       "及时性检查-通过",
			templateID: "timeliness_template_001",
			data:       map[string]interface{}{"updated_at": "2024-01-01"},
			field:      "updated_at",
			expectPass: true,
		},
		{
			name:       "标准化检查-通过",
			templateID: "standardization_template_001",
			data:       map[string]interface{}{"code": "abc123"},
			field:      "code",
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := []models.QualityRuleConfig{
				{
					RuleTemplateID: tc.templateID,
					TargetFields:   []string{tc.field},
					RuntimeConfig:  map[string]interface{}{},
					Threshold:      map[string]interface{}{},
					IsEnabled:      true,
				},
			}

			result, err := re.ApplyQualityRulesWithTemplates(tc.data, config, templates)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tc.expectPass {
				assert.Equal(t, 1.0, result.QualityScore, "质量分数应该为1.0")
				assert.Empty(t, result.Issues, "不应该有问题")
			} else {
				assert.Less(t, result.QualityScore, 1.0, "质量分数应该小于1.0")
				assert.NotEmpty(t, result.Issues, "应该有问题")
				if tc.expectIssue != "" {
					found := false
					for _, issue := range result.Issues {
						if assert.Contains(t, issue, tc.expectIssue) {
							found = true
							break
						}
					}
					assert.True(t, found, "应该包含期望的问题描述")
				}
			}
		})
	}
}

// 性能测试
func BenchmarkBuiltinRuleApplication(b *testing.B) {
	re := governance.NewRuleEngine(setupBuiltinTestDB())
	qualityTemplates := createBuiltinQualityTemplates()
	maskingTemplates := createBuiltinMaskingTemplates()
	cleansingTemplates := createBuiltinCleansingTemplates()

	testData := map[string]interface{}{
		"name":   "张三",
		"email":  "zhangsan@example.com",
		"phone":  "13800138000",
		"age":    25,
		"status": "active",
	}

	qualityConfigs := []models.QualityRuleConfig{
		{
			RuleTemplateID: "completeness_template_001",
			TargetFields:   []string{"name", "email", "phone", "age", "status"},
			RuntimeConfig:  map[string]interface{}{},
			Threshold:      map[string]interface{}{},
			IsEnabled:      true,
		},
	}

	maskingConfigs := []models.DataMaskingConfig{
		{
			TemplateID:    "masking_template_001",
			TargetFields:  []string{"phone"},
			MaskingConfig: map[string]interface{}{},
			IsEnabled:     true,
		},
	}

	cleansingConfigs := []models.DataCleansingConfig{
		{
			TemplateID:      "cleansing_template_001",
			TargetFields:    []string{"email"},
			CleansingConfig: map[string]interface{}{},
			IsEnabled:       true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 质量检查
		_, _ = re.ApplyQualityRulesWithTemplates(testData, qualityConfigs, qualityTemplates)
		// 脱敏处理
		_, _ = re.ApplyMaskingRulesWithTemplates(testData, maskingConfigs, maskingTemplates)
		// 清洗处理
		_, _ = re.ApplyCleansingRulesWithTemplates(testData, cleansingConfigs, cleansingTemplates)
	}
}
