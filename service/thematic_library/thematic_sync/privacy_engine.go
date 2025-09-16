/*
 * @module service/thematic_sync/privacy_engine
 * @description 数据脱敏引擎，负责敏感数据识别和隐私保护处理
 * @architecture 策略模式 - 支持多种脱敏策略和合规框架
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 敏感数据识别 -> 脱敏策略选择 -> 脱敏处理 -> 合规检查 -> 结果输出
 * @rules 确保数据脱敏的完整性和合规性，支持可配置的隐私策略
 * @dependencies crypto/rand, crypto/sha256, encoding/hex, regexp
 * @refs compliance_checker.go, data_anonymizer.go
 */

package thematic_sync

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MaskingStrategy 脱敏策略
type MaskingStrategy string

const (
	FullMasking      MaskingStrategy = "full"         // 完全脱敏
	PartialMasking   MaskingStrategy = "partial"      // 部分脱敏
	Tokenization     MaskingStrategy = "tokenize"     // 令牌化
	Encryption       MaskingStrategy = "encrypt"      // 加密
	Anonymization    MaskingStrategy = "anonymize"    // 匿名化
	Pseudonymization MaskingStrategy = "pseudonymize" // 假名化
)

// SensitivityLevel 敏感级别
type SensitivityLevel string

const (
	Public       SensitivityLevel = "public"       // 公开
	Internal     SensitivityLevel = "internal"     // 内部
	Restricted   SensitivityLevel = "restricted"   // 受限
	Confidential SensitivityLevel = "confidential" // 机密
)

// PrivacyRule 隐私规则
type PrivacyRule struct {
	ID               string                 `json:"id"`
	FieldPattern     string                 `json:"field_pattern"`     // 字段匹配模式
	DataType         string                 `json:"data_type"`         // 数据类型
	SensitivityLevel SensitivityLevel       `json:"sensitivity_level"` // 敏感级别
	MaskingStrategy  MaskingStrategy        `json:"masking_strategy"`
	MaskingConfig    map[string]interface{} `json:"masking_config"`
	IsEnabled        bool                   `json:"is_enabled"`
}

// MaskingResult 脱敏结果
type MaskingResult struct {
	OriginalRecord map[string]interface{} `json:"original_record"`
	MaskedRecord   map[string]interface{} `json:"masked_record"`
	AppliedRules   []string               `json:"applied_rules"`
	MaskingLog     []MaskingLogEntry      `json:"masking_log"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// MaskingLogEntry 脱敏日志条目
type MaskingLogEntry struct {
	Field          string          `json:"field"`
	Strategy       MaskingStrategy `json:"strategy"`
	OriginalValue  string          `json:"original_value,omitempty"` // 仅用于调试
	MaskedValue    string          `json:"masked_value"`
	RuleID         string          `json:"rule_id"`
	ProcessingTime time.Time       `json:"processing_time"`
}

// PrivacyEngine 数据脱敏引擎
type PrivacyEngine struct {
	masker            *DataMasker
	encryptor         *DataEncryptor
	tokenizer         *DataTokenizer
	anonymizer        *DataAnonymizer
	complianceChecker *ComplianceChecker
}

// NewPrivacyEngine 创建数据脱敏引擎
func NewPrivacyEngine() *PrivacyEngine {
	return &PrivacyEngine{
		masker:            NewDataMasker(),
		encryptor:         NewDataEncryptor(),
		tokenizer:         NewDataTokenizer(),
		anonymizer:        NewDataAnonymizer(),
		complianceChecker: NewComplianceChecker(),
	}
}

// MaskRecords 脱敏记录
func (pe *PrivacyEngine) MaskRecords(records []map[string]interface{},
	rules []PrivacyRule) ([]MaskingResult, error) {

	var results []MaskingResult

	for _, record := range records {
		result, err := pe.maskRecord(record, rules)
		if err != nil {
			return nil, fmt.Errorf("脱敏记录失败: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// maskRecord 脱敏单条记录
func (pe *PrivacyEngine) maskRecord(record map[string]interface{},
	rules []PrivacyRule) (MaskingResult, error) {

	startTime := time.Now()
	originalRecord := pe.copyRecord(record)
	maskedRecord := pe.copyRecord(record)
	var appliedRules []string
	var maskingLog []MaskingLogEntry

	// 应用脱敏规则
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}

		// 查找匹配的字段
		matchedFields := pe.findMatchingFields(maskedRecord, rule.FieldPattern)

		for _, field := range matchedFields {
			originalValue := maskedRecord[field]
			if originalValue == nil {
				continue
			}

			// 应用脱敏策略
			maskedValue, err := pe.applyMaskingStrategy(originalValue, rule)
			if err != nil {
				return MaskingResult{}, fmt.Errorf("应用脱敏策略失败: %w", err)
			}

			maskedRecord[field] = maskedValue

			// 记录脱敏日志
			logEntry := MaskingLogEntry{
				Field:          field,
				Strategy:       rule.MaskingStrategy,
				MaskedValue:    fmt.Sprintf("%v", maskedValue),
				RuleID:         rule.ID,
				ProcessingTime: time.Now(),
			}
			maskingLog = append(maskingLog, logEntry)
		}

		if len(matchedFields) > 0 {
			appliedRules = append(appliedRules, rule.ID)
		}
	}

	result := MaskingResult{
		OriginalRecord: originalRecord,
		MaskedRecord:   maskedRecord,
		AppliedRules:   appliedRules,
		MaskingLog:     maskingLog,
		ProcessingTime: time.Since(startTime),
	}

	return result, nil
}

// findMatchingFields 查找匹配的字段
func (pe *PrivacyEngine) findMatchingFields(record map[string]interface{}, pattern string) []string {
	var matchedFields []string

	// 编译正则表达式
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// 如果正则表达式无效，尝试精确匹配
		if _, exists := record[pattern]; exists {
			matchedFields = append(matchedFields, pattern)
		}
		return matchedFields
	}

	// 使用正则表达式匹配字段名
	for field := range record {
		if regex.MatchString(field) {
			matchedFields = append(matchedFields, field)
		}
	}

	return matchedFields
}

// applyMaskingStrategy 应用脱敏策略
func (pe *PrivacyEngine) applyMaskingStrategy(value interface{}, rule PrivacyRule) (interface{}, error) {
	switch rule.MaskingStrategy {
	case FullMasking:
		return pe.masker.FullMask(value, rule.MaskingConfig)
	case PartialMasking:
		return pe.masker.PartialMask(value, rule.MaskingConfig)
	case Tokenization:
		return pe.tokenizer.Tokenize(value, rule.MaskingConfig)
	case Encryption:
		return pe.encryptor.Encrypt(value, rule.MaskingConfig)
	case Anonymization:
		return pe.anonymizer.Anonymize(value, rule.MaskingConfig)
	case Pseudonymization:
		return pe.anonymizer.Pseudonymize(value, rule.MaskingConfig)
	default:
		return value, fmt.Errorf("未知脱敏策略: %s", rule.MaskingStrategy)
	}
}

// copyRecord 复制记录
func (pe *PrivacyEngine) copyRecord(record map[string]interface{}) map[string]interface{} {
	copied := make(map[string]interface{})
	for key, value := range record {
		copied[key] = value
	}
	return copied
}

// DataMasker 数据脱敏器
type DataMasker struct{}

// NewDataMasker 创建数据脱敏器
func NewDataMasker() *DataMasker {
	return &DataMasker{}
}

// FullMask 完全脱敏
func (dm *DataMasker) FullMask(value interface{}, config map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return "", nil
	}

	// 获取掩码字符
	maskChar := "*"
	if char, ok := config["mask_char"].(string); ok && char != "" {
		maskChar = char
	}

	return strings.Repeat(maskChar, len(str)), nil
}

// PartialMask 部分脱敏
func (dm *DataMasker) PartialMask(value interface{}, config map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return "", nil
	}

	// 获取配置参数
	prefixLen := 2
	suffixLen := 2
	maskChar := "*"

	if len, ok := config["prefix_length"].(int); ok {
		prefixLen = len
	}
	if len, ok := config["suffix_length"].(int); ok {
		suffixLen = len
	}
	if char, ok := config["mask_char"].(string); ok && char != "" {
		maskChar = char
	}

	strLen := len(str)

	// 如果字符串太短，直接完全脱敏
	if strLen <= prefixLen+suffixLen {
		return strings.Repeat(maskChar, strLen), nil
	}

	prefix := str[:prefixLen]
	suffix := str[strLen-suffixLen:]
	middleLen := strLen - prefixLen - suffixLen

	return prefix + strings.Repeat(maskChar, middleLen) + suffix, nil
}

// DataEncryptor 数据加密器
type DataEncryptor struct{}

// NewDataEncryptor 创建数据加密器
func NewDataEncryptor() *DataEncryptor {
	return &DataEncryptor{}
}

// Encrypt 加密数据
func (de *DataEncryptor) Encrypt(value interface{}, config map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return "", nil
	}

	// 使用SHA256哈希（简化实现）
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:]), nil
}

// DataTokenizer 数据令牌化器
type DataTokenizer struct {
	tokenMap map[string]string // 原值到令牌的映射
}

// NewDataTokenizer 创建数据令牌化器
func NewDataTokenizer() *DataTokenizer {
	return &DataTokenizer{
		tokenMap: make(map[string]string),
	}
}

// Tokenize 令牌化数据
func (dt *DataTokenizer) Tokenize(value interface{}, config map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return "", nil
	}

	// 检查是否已有令牌
	if token, exists := dt.tokenMap[str]; exists {
		return token, nil
	}

	// 生成新令牌
	token, err := dt.generateToken()
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	dt.tokenMap[str] = token
	return token, nil
}

// generateToken 生成令牌
func (dt *DataTokenizer) generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// DataAnonymizer 数据匿名化器
type DataAnonymizer struct{}

// NewDataAnonymizer 创建数据匿名化器
func NewDataAnonymizer() *DataAnonymizer {
	return &DataAnonymizer{}
}

// Anonymize 匿名化数据
func (da *DataAnonymizer) Anonymize(value interface{}, config map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return "", nil
	}

	// 根据数据类型进行匿名化
	dataType := "string"
	if t, ok := config["data_type"].(string); ok {
		dataType = t
	}

	switch dataType {
	case "name":
		return da.anonymizeName(str), nil
	case "email":
		return da.anonymizeEmail(str), nil
	case "phone":
		return da.anonymizePhone(str), nil
	case "address":
		return da.anonymizeAddress(str), nil
	default:
		return da.anonymizeGeneral(str), nil
	}
}

// Pseudonymize 假名化数据
func (da *DataAnonymizer) Pseudonymize(value interface{}, config map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" {
		return "", nil
	}

	// 生成一致的假名（基于哈希）
	hash := sha256.Sum256([]byte(str))
	hashStr := hex.EncodeToString(hash[:])

	// 根据数据类型生成相应格式的假名
	dataType := "string"
	if t, ok := config["data_type"].(string); ok {
		dataType = t
	}

	switch dataType {
	case "name":
		return fmt.Sprintf("User_%s", hashStr[:8]), nil
	case "email":
		return fmt.Sprintf("user_%s@example.com", hashStr[:8]), nil
	case "phone":
		return fmt.Sprintf("138%s", hashStr[:8]), nil
	default:
		return fmt.Sprintf("PSEUDO_%s", hashStr[:8]), nil
	}
}

// anonymizeName 匿名化姓名
func (da *DataAnonymizer) anonymizeName(name string) string {
	return "匿名用户"
}

// anonymizeEmail 匿名化邮箱
func (da *DataAnonymizer) anonymizeEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "anonymous@example.com"
	}
	return "anonymous@" + parts[1]
}

// anonymizePhone 匿名化电话
func (da *DataAnonymizer) anonymizePhone(phone string) string {
	if len(phone) >= 4 {
		return phone[:3] + "****" + phone[len(phone)-4:]
	}
	return "****"
}

// anonymizeAddress 匿名化地址
func (da *DataAnonymizer) anonymizeAddress(address string) string {
	return "匿名地址"
}

// anonymizeGeneral 通用匿名化
func (da *DataAnonymizer) anonymizeGeneral(str string) string {
	return "***"
}

// ComplianceFramework 合规框架
type ComplianceFramework string

const (
	GDPR  ComplianceFramework = "gdpr"  // 欧盟GDPR
	CCPA  ComplianceFramework = "ccpa"  // 加州CCPA
	PIPL  ComplianceFramework = "pipl"  // 中国个保法
	HIPAA ComplianceFramework = "hipaa" // 美国HIPAA
)

// ComplianceReport 合规报告
type ComplianceReport struct {
	Framework       ComplianceFramework   `json:"framework"`
	IsCompliant     bool                  `json:"is_compliant"`
	Violations      []ComplianceViolation `json:"violations"`
	Recommendations []string              `json:"recommendations"`
	CheckTime       time.Time             `json:"check_time"`
}

// ComplianceViolation 合规违规
type ComplianceViolation struct {
	Field       string `json:"field"`
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// ComplianceChecker 合规检查器
type ComplianceChecker struct {
	frameworks []ComplianceFramework
}

// NewComplianceChecker 创建合规检查器
func NewComplianceChecker() *ComplianceChecker {
	return &ComplianceChecker{
		frameworks: []ComplianceFramework{GDPR, PIPL},
	}
}

// CheckCompliance 检查合规性
func (cc *ComplianceChecker) CheckCompliance(records []map[string]interface{}) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Framework:       GDPR, // 默认使用GDPR
		IsCompliant:     true,
		Violations:      make([]ComplianceViolation, 0),
		Recommendations: make([]string, 0),
		CheckTime:       time.Now(),
	}

	// 检查每条记录
	for _, record := range records {
		violations := cc.checkRecord(record)
		report.Violations = append(report.Violations, violations...)
	}

	// 如果有违规，标记为不合规
	if len(report.Violations) > 0 {
		report.IsCompliant = false
		report.Recommendations = append(report.Recommendations,
			"建议对敏感字段应用适当的脱敏策略")
	}

	return report, nil
}

// checkRecord 检查单条记录
func (cc *ComplianceChecker) checkRecord(record map[string]interface{}) []ComplianceViolation {
	var violations []ComplianceViolation

	// 检查常见敏感字段
	sensitiveFields := []string{
		"id_card", "identity_card", "passport", "phone", "mobile",
		"email", "address", "name", "birthday", "salary",
	}

	for _, field := range sensitiveFields {
		if value, exists := record[field]; exists && value != nil {
			// 简单检查是否已脱敏
			str := fmt.Sprintf("%v", value)
			if !cc.isMasked(str) {
				violation := ComplianceViolation{
					Field:       field,
					Rule:        "sensitive_data_protection",
					Severity:    "high",
					Description: fmt.Sprintf("敏感字段 %s 未进行脱敏处理", field),
				}
				violations = append(violations, violation)
			}
		}
	}

	return violations
}

// isMasked 检查是否已脱敏
func (cc *ComplianceChecker) isMasked(value string) bool {
	// 简单检查是否包含掩码字符
	return strings.Contains(value, "*") ||
		strings.Contains(value, "PSEUDO_") ||
		strings.Contains(value, "anonymous") ||
		strings.Contains(value, "匿名")
}
