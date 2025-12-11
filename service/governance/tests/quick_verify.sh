#!/bin/bash

# 数据脱敏规则快速验证脚本
# 用途: 快速验证所有脱敏规则是否正常工作

set -e

echo "=========================================="
echo "数据脱敏规则快速验证"
echo "=========================================="
echo ""

# 进入项目目录
cd "$(dirname "$0")/../../.."

echo "📋 运行测试用例..."
echo ""

# 运行所有脱敏测试
echo "1. 测试身份证脱敏（自动识别15位和18位）..."
go test ./service/governance/tests -v -run TestMaskIDCard 2>&1 | grep -E "PASS|FAIL|RUN"
echo ""

echo "2. 测试银行卡号脱敏..."
go test ./service/governance/tests -v -run TestMaskBankCard 2>&1 | grep -E "PASS|FAIL|RUN"
echo ""

echo "3. 测试中文姓名脱敏..."
go test ./service/governance/tests -v -run TestMaskChineseName 2>&1 | grep -E "PASS|FAIL|RUN"
echo ""

echo "4. 测试邮箱脱敏..."
go test ./service/governance/tests -v -run TestMaskEmail 2>&1 | grep -E "PASS|FAIL|RUN"
echo ""

echo "5. 运行集成测试..."
go test ./service/governance/tests -v -run TestMaskingRulesWithRuleEngine 2>&1 | grep -E "PASS|FAIL|RUN"
echo ""

echo "=========================================="
echo "✅ 验证完成！"
echo "=========================================="
echo ""
echo "查看完整测试报告，请运行:"
echo "  go test ./service/governance/tests -v -run TestMask"
echo ""
echo "查看测试覆盖率，请运行:"
echo "  go test ./service/governance/tests -v -run TestMask -cover"
echo ""

