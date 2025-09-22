/*
 * @module service/governance/tests/rule_test_test
 * @description 数据治理规则测试功能的单元测试
 * @architecture 测试层
 * @documentReference ai_docs/data_governance_req.md
 * @stateFlow 测试数据准备 -> 规则执行测试 -> 结果验证
 * @rules 确保规则测试功能的正确性和稳定性
 * @dependencies testing, testify, gorm
 * @refs service/governance/governance_service.go, service/governance/types.go
 */

package tests

import (
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// RuleTestSuite 规则测试测试套件
type RuleTestSuite struct {
	suite.Suite
	db                *gorm.DB
	governanceService *governance.GovernanceService
	qualityRuleID     string
	maskingRuleID     string
	cleansingRuleID   string
}

// SetupSuite 设置测试套件
func (suite *RuleTestSuite) SetupSuite() {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)

	// 自动迁移
	err = db.AutoMigrate(
		&models.QualityRuleTemplate{},
		&models.DataMaskingTemplate{},
		&models.DataCleansingTemplate{},
		&models.Metadata{},
		&models.SystemLog{},
		&models.DataQualityReport{},
	)
	suite.Require().NoError(err)

	suite.db = db
	suite.governanceService = governance.NewGovernanceService(db)

	// 创建测试用的规则模板
	suite.setupTestRules()
}

// setupTestRules 设置测试规则
func (suite *RuleTestSuite) setupTestRules() {
	// 创建质量规则模板
	qualityRule := &models.QualityRuleTemplate{
		ID:          uuid.New().String(),
		Name:        "测试完整性检查",
		Type:        "completeness",
		Category:    "basic_quality",
		Description: "测试用的完整性检查规则",
		RuleLogic: models.JSONB(map[string]interface{}{
			"check_type": "not_null",
			"operator":   "is_not_null",
		}),
		Parameters: models.JSONB(map[string]interface{}{
			"threshold": 0.9,
		}),
		DefaultConfig: models.JSONB(map[string]interface{}{
			"strict_mode": true,
		}),
		IsBuiltIn: false,
		IsEnabled: true,
		Version:   "1.0",
		Tags: models.JSONB(map[string]interface{}{
			"category": "test",
		}),
		CreatedBy: "test",
		UpdatedBy: "test",
	}

	err := suite.governanceService.CreateQualityRule(qualityRule)
	suite.Require().NoError(err)
	suite.qualityRuleID = qualityRule.ID

	// 创建脱敏规则模板
	maskingRule := &models.DataMaskingTemplate{
		ID:              uuid.New().String(),
		Name:            "测试手机号脱敏",
		MaskingType:     "mask",
		Category:        "personal_info",
		Description:     "测试用的手机号脱敏规则",
		ApplicableTypes: pq.StringArray{"string"},
		MaskingLogic: models.JSONB(map[string]interface{}{
			"mask_type":   "middle",
			"mask_char":   "*",
			"keep_prefix": 3,
			"keep_suffix": 4,
		}),
		Parameters: models.JSONB(map[string]interface{}{
			"preserve_format": true,
		}),
		DefaultConfig: models.JSONB(map[string]interface{}{
			"strict_validation": true,
		}),
		SecurityLevel: "high",
		IsBuiltIn:     false,
		IsEnabled:     true,
		Version:       "1.0",
		Tags: models.JSONB(map[string]interface{}{
			"category": "test",
		}),
		CreatedBy: "test",
		UpdatedBy: "test",
	}

	err = suite.governanceService.CreateMaskingRule(maskingRule)
	suite.Require().NoError(err)
	suite.maskingRuleID = maskingRule.ID

	// 创建清洗规则模板
	cleansingRule := &models.DataCleansingTemplate{
		ID:          uuid.New().String(),
		Name:        "测试邮箱标准化",
		Description: "测试用的邮箱标准化规则",
		RuleType:    "standardization",
		Category:    "data_format",
		CleansingLogic: models.JSONB(map[string]interface{}{
			"operation": "lowercase",
			"trim":      true,
		}),
		Parameters: models.JSONB(map[string]interface{}{
			"validate_format": true,
		}),
		DefaultConfig: models.JSONB(map[string]interface{}{
			"strict_mode": false,
		}),
		ApplicableTypes: models.JSONB(map[string]interface{}{"types": []string{"string"}}),
		ComplexityLevel: "low",
		IsBuiltIn:       false,
		IsEnabled:       true,
		Version:         "1.0",
		Tags: models.JSONB(map[string]interface{}{
			"category": "test",
		}),
		CreatedBy: "test",
		UpdatedBy: "test",
	}

	err = suite.governanceService.CreateCleansingRule(cleansingRule)
	suite.Require().NoError(err)
	suite.cleansingRuleID = cleansingRule.ID
}

// TearDownSuite 清理测试套件
func (suite *RuleTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// TestQualityRuleTest 测试质量规则测试功能
func (suite *RuleTestSuite) TestQualityRuleTest() {
	req := &governance.TestQualityRuleRequest{
		RuleTemplateID: suite.qualityRuleID,
		TestData: map[string]interface{}{
			"name":  "张三",
			"email": "zhangsan@example.com",
			"phone": "13800138000",
			"age":   25,
		},
		TargetFields: []string{"name", "email"},
		RuntimeConfig: map[string]interface{}{
			"check_empty": true,
		},
		Threshold: map[string]interface{}{
			"completeness_threshold": 0.8,
		},
	}

	result, err := suite.governanceService.TestQualityRule(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 1, result.TotalRules)
	assert.NotEmpty(suite.T(), result.TestID)
	assert.Len(suite.T(), result.Results, 1)

	// 检查测试结果
	testResult := result.Results[0]
	assert.Equal(suite.T(), "quality", testResult.RuleType)
	assert.Equal(suite.T(), suite.qualityRuleID, testResult.RuleTemplateID)
	assert.Equal(suite.T(), "测试完整性检查", testResult.RuleName)
	assert.NotNil(suite.T(), testResult.ProcessedData)
	assert.NotNil(suite.T(), testResult.OriginalData)
}

// TestMaskingRuleTest 测试脱敏规则测试功能
func (suite *RuleTestSuite) TestMaskingRuleTest() {
	req := &governance.TestMaskingRuleRequest{
		TemplateID: suite.maskingRuleID,
		TestData: map[string]interface{}{
			"name":  "张三",
			"phone": "13800138000",
			"email": "zhangsan@example.com",
		},
		TargetFields: []string{"phone"},
		MaskingConfig: map[string]interface{}{
			"mask_pattern": "***",
		},
		PreserveFormat: true,
	}

	result, err := suite.governanceService.TestMaskingRule(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 1, result.TotalRules)
	assert.NotEmpty(suite.T(), result.TestID)
	assert.Len(suite.T(), result.Results, 1)

	// 检查测试结果
	testResult := result.Results[0]
	assert.Equal(suite.T(), "masking", testResult.RuleType)
	assert.Equal(suite.T(), suite.maskingRuleID, testResult.RuleTemplateID)
	assert.Equal(suite.T(), "测试手机号脱敏", testResult.RuleName)
	assert.NotNil(suite.T(), testResult.ProcessedData)
	assert.NotNil(suite.T(), testResult.OriginalData)
}

// TestCleansingRuleTest 测试清洗规则测试功能
func (suite *RuleTestSuite) TestCleansingRuleTest() {
	req := &governance.TestCleansingRuleRequest{
		TemplateID: suite.cleansingRuleID,
		TestData: map[string]interface{}{
			"name":  "张三",
			"email": "ZhangSan@EXAMPLE.COM",
			"phone": "13800138000",
		},
		TargetFields: []string{"email"},
		CleansingConfig: map[string]interface{}{
			"lowercase": true,
			"trim":      true,
		},
		TriggerCondition: "email != ''",
		BackupOriginal:   true,
	}

	result, err := suite.governanceService.TestCleansingRule(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 1, result.TotalRules)
	assert.NotEmpty(suite.T(), result.TestID)
	assert.Len(suite.T(), result.Results, 1)

	// 检查测试结果
	testResult := result.Results[0]
	assert.Equal(suite.T(), "cleansing", testResult.RuleType)
	assert.Equal(suite.T(), suite.cleansingRuleID, testResult.RuleTemplateID)
	assert.Equal(suite.T(), "测试邮箱标准化", testResult.RuleName)
	assert.NotNil(suite.T(), testResult.ProcessedData)
	assert.NotNil(suite.T(), testResult.OriginalData)
}

// TestBatchRulesTest 测试批量规则测试功能
func (suite *RuleTestSuite) TestBatchRulesTest() {
	req := &governance.TestBatchRulesRequest{
		TestData: map[string]interface{}{
			"name":  "张三",
			"email": "ZhangSan@EXAMPLE.COM",
			"phone": "13800138000",
			"age":   25,
		},
		QualityRules: []governance.TestQualityRuleItem{
			{
				RuleTemplateID: suite.qualityRuleID,
				TargetFields:   []string{"name", "email"},
				RuntimeConfig: map[string]interface{}{
					"check_empty": true,
				},
				Threshold: map[string]interface{}{
					"completeness_threshold": 0.8,
				},
			},
		},
		CleansingRules: []governance.TestCleansingRuleItem{
			{
				TemplateID:   suite.cleansingRuleID,
				TargetFields: []string{"email"},
				CleansingConfig: map[string]interface{}{
					"lowercase": true,
				},
				BackupOriginal: true,
			},
		},
		MaskingRules: []governance.TestMaskingRuleItem{
			{
				TemplateID:   suite.maskingRuleID,
				TargetFields: []string{"phone"},
				MaskingConfig: map[string]interface{}{
					"mask_pattern": "***",
				},
				PreserveFormat: true,
			},
		},
		ExecutionOrder: []string{"quality", "cleansing", "masking"},
	}

	result, err := suite.governanceService.TestBatchRules(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 3, result.TotalRules)
	assert.NotEmpty(suite.T(), result.TestID)
	assert.Len(suite.T(), result.Results, 3)

	// 检查汇总信息
	assert.Equal(suite.T(), 1, result.Summary.QualityChecks)
	assert.Equal(suite.T(), 1, result.Summary.MaskingRules)
	assert.Equal(suite.T(), 1, result.Summary.CleansingRules)
}

// TestRulePreview 测试规则预览功能
func (suite *RuleTestSuite) TestRulePreview() {
	// 测试质量规则预览
	qualityReq := &governance.TestRulePreviewRequest{
		RuleType:   "quality",
		TemplateID: suite.qualityRuleID,
		SampleData: map[string]interface{}{
			"name":  "张三",
			"email": "zhangsan@example.com",
		},
		TargetFields: []string{"name", "email"},
		Configuration: map[string]interface{}{
			"check_empty": true,
		},
	}

	qualityResult, err := suite.governanceService.TestRulePreview(qualityReq)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), qualityResult)
	assert.Equal(suite.T(), "quality", qualityResult.RuleType)
	assert.Equal(suite.T(), "测试完整性检查", qualityResult.RuleName)
	assert.NotNil(suite.T(), qualityResult.OriginalData)
	assert.NotNil(suite.T(), qualityResult.PreviewResult)
	assert.NotEmpty(suite.T(), qualityResult.ExpectedChanges)
	assert.True(suite.T(), qualityResult.ConfigValidation.IsValid)

	// 测试脱敏规则预览
	maskingReq := &governance.TestRulePreviewRequest{
		RuleType:   "masking",
		TemplateID: suite.maskingRuleID,
		SampleData: map[string]interface{}{
			"phone": "13800138000",
		},
		TargetFields: []string{"phone"},
		Configuration: map[string]interface{}{
			"mask_pattern": "***",
		},
	}

	maskingResult, err := suite.governanceService.TestRulePreview(maskingReq)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), maskingResult)
	assert.Equal(suite.T(), "masking", maskingResult.RuleType)
	assert.Equal(suite.T(), "测试手机号脱敏", maskingResult.RuleName)
	assert.Equal(suite.T(), "medium", maskingResult.EstimatedImpact.RiskLevel)

	// 测试清洗规则预览
	cleansingReq := &governance.TestRulePreviewRequest{
		RuleType:   "cleansing",
		TemplateID: suite.cleansingRuleID,
		SampleData: map[string]interface{}{
			"email": "ZhangSan@EXAMPLE.COM",
		},
		TargetFields: []string{"email"},
		Configuration: map[string]interface{}{
			"lowercase": true,
		},
	}

	cleansingResult, err := suite.governanceService.TestRulePreview(cleansingReq)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), cleansingResult)
	assert.Equal(suite.T(), "cleansing", cleansingResult.RuleType)
	assert.Equal(suite.T(), "测试邮箱标准化", cleansingResult.RuleName)
	assert.Equal(suite.T(), "low", cleansingResult.EstimatedImpact.RiskLevel)
}

// TestInvalidRuleID 测试无效规则ID的情况
func (suite *RuleTestSuite) TestInvalidRuleID() {
	req := &governance.TestQualityRuleRequest{
		RuleTemplateID: "invalid-rule-id",
		TestData: map[string]interface{}{
			"name": "张三",
		},
		TargetFields: []string{"name"},
	}

	result, err := suite.governanceService.TestQualityRule(req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "获取质量规则模板失败")
}

// TestEmptyTestData 测试空测试数据的情况
func (suite *RuleTestSuite) TestEmptyTestData() {
	req := &governance.TestQualityRuleRequest{
		RuleTemplateID: suite.qualityRuleID,
		TestData:       map[string]interface{}{},
		TargetFields:   []string{"name"},
	}

	result, err := suite.governanceService.TestQualityRule(req)

	// 应该能正常处理空数据，但可能会有质量问题
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 1, result.TotalRules)
}

// TestEmptyExecutionOrder 测试空执行顺序的批量测试
func (suite *RuleTestSuite) TestEmptyExecutionOrder() {
	req := &governance.TestBatchRulesRequest{
		TestData: map[string]interface{}{
			"name": "张三",
		},
		QualityRules: []governance.TestQualityRuleItem{
			{
				RuleTemplateID: suite.qualityRuleID,
				TargetFields:   []string{"name"},
			},
		},
		ExecutionOrder: []string{}, // 空执行顺序
	}

	result, err := suite.governanceService.TestBatchRules(req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 0, result.TotalRules) // 没有执行任何规则
}

// 运行测试套件
func TestRuleTestSuite(t *testing.T) {
	suite.Run(t, new(RuleTestSuite))
}

// TestRuleTestFunctions 单独的功能测试
func TestRuleTestFunctions(t *testing.T) {
	// 测试基本的结构体创建
	testData := map[string]interface{}{
		"name":  "测试用户",
		"email": "test@example.com",
		"phone": "13800138000",
	}

	// 测试TestQualityRuleRequest结构体
	qualityReq := governance.TestQualityRuleRequest{
		RuleTemplateID: "test-rule-id",
		TestData:       testData,
		TargetFields:   []string{"name", "email"},
		RuntimeConfig: map[string]interface{}{
			"check_empty": true,
		},
		Threshold: map[string]interface{}{
			"completeness_threshold": 0.8,
		},
	}

	assert.Equal(t, "test-rule-id", qualityReq.RuleTemplateID)
	assert.Equal(t, testData, qualityReq.TestData)
	assert.Equal(t, []string{"name", "email"}, qualityReq.TargetFields)

	// 测试TestRuleResponse结构体
	response := governance.TestRuleResponse{
		TestID:          "test-execution-id",
		TotalRules:      3,
		SuccessfulRules: 2,
		FailedRules:     1,
		OverallSuccess:  false,
		ExecutionTime:   150,
		Results: []governance.RuleTestResult{
			{
				RuleType:       "quality",
				RuleTemplateID: "rule-1",
				RuleName:       "完整性检查",
				Success:        true,
				ProcessedData:  testData,
				OriginalData:   testData,
				ExecutionTime:  50,
			},
		},
	}

	assert.Equal(t, "test-execution-id", response.TestID)
	assert.Equal(t, 3, response.TotalRules)
	assert.Equal(t, 2, response.SuccessfulRules)
	assert.Equal(t, 1, response.FailedRules)
	assert.False(t, response.OverallSuccess)
	assert.Len(t, response.Results, 1)
}
