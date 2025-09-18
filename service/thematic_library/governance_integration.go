/*
 * @module service/thematic_library/governance_integration
 * @description 数据治理集成服务，负责在主题同步过程中调用数据治理模块的规则进行数据处理
 * @architecture 适配器模式 - 适配数据治理服务接口
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 规则加载 -> 数据处理 -> 结果返回
 * @rules 统一调用数据治理规则，确保数据处理的合规性和一致性
 * @dependencies gorm.io/gorm, context, datahub-service/service/governance, datahub-service/service/models
 * @refs thematic_sync_service.go
 */

package thematic_library

import (
	"context"
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GovernanceIntegrationService 数据治理集成服务
type GovernanceIntegrationService struct {
	db                *gorm.DB
	governanceService *governance.GovernanceService
	ruleEngine        *governance.RuleEngine
}

// NewGovernanceIntegrationService 创建数据治理集成服务实例
func NewGovernanceIntegrationService(db *gorm.DB, governanceService *governance.GovernanceService) *GovernanceIntegrationService {
	return &GovernanceIntegrationService{
		db:                db,
		governanceService: governanceService,
		ruleEngine:        governance.NewRuleEngine(db),
	}
}

// ApplyGovernanceRules 应用数据治理规则
func (gis *GovernanceIntegrationService) ApplyGovernanceRules(
	ctx context.Context,
	records []map[string]interface{},
	task *models.ThematicSyncTask,
	config *GovernanceExecutionConfig,
) (*GovernanceExecutionResult, error) {

	startTime := time.Now()
	result := &GovernanceExecutionResult{
		QualityCheckResults:   []QualityCheckResult{},
		CleansingResults:      []CleansingResult{},
		MaskingResults:        []MaskingResult{},
		TransformationResults: []TransformationResult{},
		ValidationResults:     []ValidationResult{},
		OverallQualityScore:   100.0, // 初始满分
		TotalProcessedRecords: int64(len(records)),
		TotalCleansingApplied: 0,
		TotalMaskingApplied:   0,
		TotalValidationErrors: 0,
		ComplianceStatus:      "compliant",
		Issues:                []GovernanceIssue{},
	}

	// 从任务中提取规则配置
	var qualityConfigs []models.QualityRuleConfig
	var maskingConfigs []models.DataMaskingConfig
	var cleansingConfigs []models.DataCleansingConfig

	// 从任务的治理配置中获取规则配置
	if task.GovernanceConfig != nil {
		governanceConfigMap := map[string]interface{}(task.GovernanceConfig)
		if governanceConfigMap != nil {
			// 解析质量规则配置
			if qualityRulesInterface, exists := governanceConfigMap["quality_rules"]; exists {
				if qualityRulesBytes, err := json.Marshal(qualityRulesInterface); err == nil {
					json.Unmarshal(qualityRulesBytes, &qualityConfigs)
				}
			}

			// 解析脱敏规则配置
			if maskingRulesInterface, exists := governanceConfigMap["masking_rules"]; exists {
				if maskingRulesBytes, err := json.Marshal(maskingRulesInterface); err == nil {
					json.Unmarshal(maskingRulesBytes, &maskingConfigs)
				}
			}

			// 解析清洗规则配置
			if cleansingRulesInterface, exists := governanceConfigMap["cleansing_rules"]; exists {
				if cleansingRulesBytes, err := json.Marshal(cleansingRulesInterface); err == nil {
					json.Unmarshal(cleansingRulesBytes, &cleansingConfigs)
				}
			}
		}
	}

	// 处理每条记录
	processedRecords := make([]map[string]interface{}, 0, len(records))
	totalQualityScore := 0.0
	totalValidationErrors := 0
	totalCleansingApplied := 0
	totalMaskingApplied := 0

	for _, record := range records {
		processedRecord := make(map[string]interface{})
		// 复制原始数据
		for k, v := range record {
			processedRecord[k] = v
		}

		// 1. 应用质量检查规则 (如果启用)
		if config.EnableQualityCheck && len(qualityConfigs) > 0 {
			qualityResult, err := gis.ruleEngine.ApplyQualityRules(processedRecord, qualityConfigs)
			if err != nil {
				result.Issues = append(result.Issues, GovernanceIssue{
					Type:        "quality_check_error",
					Description: fmt.Sprintf("质量检查失败: %v", err),
					Record:      fmt.Sprintf("%v", record["id"]),
					Severity:    "error",
				})
			} else {
				totalQualityScore += qualityResult.QualityScore
				processedRecord = qualityResult.ProcessedData
			}
		}

		// 2. 应用清洗规则 (如果启用)
		if config.EnableCleansing && len(cleansingConfigs) > 0 {
			cleansingResult, err := gis.ruleEngine.ApplyCleansingRules(processedRecord, cleansingConfigs)
			if err != nil {
				result.Issues = append(result.Issues, GovernanceIssue{
					Type:        "cleansing_error",
					Description: fmt.Sprintf("清洗处理失败: %v", err),
					Record:      fmt.Sprintf("%v", record["id"]),
					Severity:    "warning",
				})
			} else {
				processedRecord = cleansingResult.ProcessedData
				if len(cleansingResult.RulesApplied) > 0 {
					totalCleansingApplied++
				}
			}
		}

		// 3. 应用脱敏规则 (如果启用)
		if config.EnableMasking && len(maskingConfigs) > 0 {
			maskingResult, err := gis.ruleEngine.ApplyMaskingRules(processedRecord, maskingConfigs)
			if err != nil {
				result.Issues = append(result.Issues, GovernanceIssue{
					Type:        "masking_error",
					Description: fmt.Sprintf("脱敏处理失败: %v", err),
					Record:      fmt.Sprintf("%v", record["id"]),
					Severity:    "error",
				})
			} else {
				processedRecord = maskingResult.ProcessedData
				if len(maskingResult.RulesApplied) > 0 {
					totalMaskingApplied++
				}
			}
		}

		processedRecords = append(processedRecords, processedRecord)
	}

	// 计算平均质量分数
	if len(records) > 0 {
		result.OverallQualityScore = totalQualityScore / float64(len(records)) * 100
	}

	result.TotalCleansingApplied = int64(totalCleansingApplied)
	result.TotalMaskingApplied = int64(totalMaskingApplied)
	result.TotalValidationErrors = int64(totalValidationErrors)

	// 设置合规状态
	if len(result.Issues) == 0 {
		result.ComplianceStatus = "compliant"
	} else {
		result.ComplianceStatus = "non_compliant"
	}

	result.ExecutionTime = time.Since(startTime)

	return result, nil
}
