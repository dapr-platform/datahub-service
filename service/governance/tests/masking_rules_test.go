/*
 * @module service/governance/tests
 * @description 数据脱敏规则测试
 * @architecture 测试层
 */

package tests

import (
	"datahub-service/service/governance"
	"datahub-service/service/models"
	"testing"
)

// TestMaskIDCard 测试身份证脱敏（自动识别15位和18位）
func TestMaskIDCard(t *testing.T) {
	engine := governance.NewRuleEngine(nil)

	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "18位身份证-示例1",
			input:    "110101199001011234",
			expected: "110101********1234",
			wantErr:  false,
		},
		{
			name:     "18位身份证-示例2",
			input:    "330602199512155678",
			expected: "330602********5678",
			wantErr:  false,
		},
		{
			name:     "15位身份证-示例1",
			input:    "110101900101123",
			expected: "110101******123",
			wantErr:  false,
		},
		{
			name:     "15位身份证-示例2",
			input:    "330602951215456",
			expected: "330602******456",
			wantErr:  false,
		},
		{
			name:     "无效长度-错误测试",
			input:    "12345678901",
			expected: "12345678901",
			wantErr:  true,
		},
	}

	config := map[string]interface{}{
		"pattern":   "id_card",
		"mask_char": "*",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := engine.ExecuteMasking(tc.input, config)
			if tc.wantErr {
				if err == nil {
					t.Errorf("期望返回错误，但成功执行")
				}
				return
			}
			if err != nil {
				t.Errorf("脱敏失败: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("期望: %s, 实际: %s", tc.expected, result)
			}
		})
	}
}

// TestMaskBankCard 测试银行卡号脱敏
func TestMaskBankCard(t *testing.T) {
	engine := governance.NewRuleEngine(nil)

	testCases := []struct {
		name        string
		input       string
		groupFormat bool
		expected    string
	}{
		{
			name:        "16位银行卡号（不分组）",
			input:       "6222021234567890",
			groupFormat: false,
			expected:    "622202******7890",
		},
		{
			name:        "16位银行卡号（分组）",
			input:       "6222021234567890",
			groupFormat: true,
			expected:    "6222 02** **** 7890",
		},
		{
			name:        "19位银行卡号（分组）",
			input:       "6217001234567891234",
			groupFormat: true,
			expected:    "6217 00** **** ***1 234",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]interface{}{
				"pattern":      "bank_card",
				"mask_char":    "*",
				"group_format": tc.groupFormat,
			}
			result, err := engine.ExecuteMasking(tc.input, config)
			if err != nil {
				t.Errorf("脱敏失败: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("期望: %s, 实际: %s", tc.expected, result)
			}
		})
	}
}

// TestMaskChineseName 测试中文姓名脱敏
func TestMaskChineseName(t *testing.T) {
	engine := governance.NewRuleEngine(nil)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单字名",
			input:    "李",
			expected: "李",
		},
		{
			name:     "双字名",
			input:    "张三",
			expected: "张*",
		},
		{
			name:     "三字名",
			input:    "王明华",
			expected: "王*华",
		},
		{
			name:     "四字名",
			input:    "欧阳明芳",
			expected: "欧**芳",
		},
	}

	config := map[string]interface{}{
		"pattern":   "chinese_name",
		"mask_char": "*",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := engine.ExecuteMasking(tc.input, config)
			if err != nil {
				t.Errorf("脱敏失败: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("期望: %s, 实际: %s", tc.expected, result)
			}
		})
	}
}

// TestMaskEmail 测试邮箱脱敏
func TestMaskEmail(t *testing.T) {
	engine := governance.NewRuleEngine(nil)

	testCases := []struct {
		name              string
		input             string
		keepUsernameChars int
		expected          string
	}{
		{
			name:              "短用户名邮箱",
			input:             "zhang@163.com",
			keepUsernameChars: 2,
			expected:          "zh***@163.com",
		},
		{
			name:              "长用户名邮箱",
			input:             "wangming@gmail.com",
			keepUsernameChars: 2,
			expected:          "wa******@gmail.com",
		},
		{
			name:              "两字符用户名",
			input:             "li@qq.com",
			keepUsernameChars: 2,
			expected:          "li*@qq.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]interface{}{
				"pattern":             "email",
				"mask_char":           "*",
				"keep_username_chars": tc.keepUsernameChars,
			}
			result, err := engine.ExecuteMasking(tc.input, config)
			if err != nil {
				t.Errorf("脱敏失败: %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("期望: %s, 实际: %s", tc.expected, result)
			}
		})
	}
}

// TestMaskingRulesWithRuleEngine 测试使用RuleEngine应用脱敏规则
func TestMaskingRulesWithRuleEngine(t *testing.T) {
	engine := governance.NewRuleEngine(nil)

	// 测试数据
	data := map[string]interface{}{
		"id_card_18":   "110101199001011234",
		"id_card_15":   "110101900101123",
		"bank_card":    "6222021234567890",
		"name":         "张三",
		"email":        "zhangsan@163.com",
		"phone":        "13800138000",
	}

	// 脱敏配置
	configs := []models.DataMaskingConfig{
		{
			TemplateID:   "masking_template_002", // 身份证（统一处理18位和15位）
			TargetFields: []string{"id_card_18", "id_card_15"},
			MaskingConfig: map[string]interface{}{
				"pattern":   "id_card",
				"mask_char": "*",
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "masking_template_003", // 银行卡号
			TargetFields: []string{"bank_card"},
			MaskingConfig: map[string]interface{}{
				"pattern":      "bank_card",
				"mask_char":    "*",
				"group_format": true,
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "masking_template_004", // 姓名
			TargetFields: []string{"name"},
			MaskingConfig: map[string]interface{}{
				"pattern":   "chinese_name",
				"mask_char": "*",
			},
			IsEnabled: true,
		},
		{
			TemplateID:   "masking_template_005", // 邮箱
			TargetFields: []string{"email"},
			MaskingConfig: map[string]interface{}{
				"pattern":             "email",
				"mask_char":           "*",
				"keep_username_chars": 2,
			},
			IsEnabled: true,
		},
	}

	// 模拟模板数据
	templates := map[string]*models.DataMaskingTemplate{
		"masking_template_002": {
			MaskingType: "mask",
			MaskingLogic: map[string]interface{}{
				"pattern":   "id_card",
				"mask_char": "*",
			},
		},
		"masking_template_003": {
			MaskingType: "mask",
			MaskingLogic: map[string]interface{}{
				"pattern":      "bank_card",
				"mask_char":    "*",
				"group_format": true,
			},
		},
		"masking_template_004": {
			MaskingType: "mask",
			MaskingLogic: map[string]interface{}{
				"pattern":   "chinese_name",
				"mask_char": "*",
			},
		},
		"masking_template_005": {
			MaskingType: "mask",
			MaskingLogic: map[string]interface{}{
				"pattern":             "email",
				"mask_char":           "*",
				"keep_username_chars": 2,
			},
		},
	}

	// 执行脱敏
	result, err := engine.ApplyMaskingRulesWithTemplates(data, configs, templates)
	if err != nil {
		t.Fatalf("脱敏失败: %v", err)
	}

	// 验证结果
	expectedResults := map[string]string{
		"id_card_18": "110101********1234",
		"id_card_15": "110101******123",
		"bank_card":  "6222 02** **** 7890",
		"name":       "张*",
		"email":      "zh********@163.com",
	}

	for field, expectedValue := range expectedResults {
		actualValue := result.ProcessedData[field]
		if actualValue != expectedValue {
			t.Errorf("字段 %s: 期望 %s, 实际 %s", field, expectedValue, actualValue)
		}
	}

	// 检查修改记录
	if len(result.Modifications) == 0 {
		t.Error("应该有修改记录")
	}
}

