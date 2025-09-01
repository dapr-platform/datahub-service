#!/bin/bash

#
# @module scripts/cleanup_test_data
# @description 清理PostgREST测试数据脚本
# @architecture 清理脚本
# @documentReference ../ai_docs/postgrest_rbac_guide.md
# @stateFlow 登录 -> 清理所有测试数据
# @rules 清理所有测试相关的数据，避免重复数据冲突
# @dependencies PostgREST服务, PostgreSQL数据库
# @refs ../service/models/
#

set -e

# 配置
POSTGREST_URL="http://localhost:3000"
USERNAME="admin"
PASSWORD="things2024"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== 清理PostgREST测试数据 ===${NC}"

# 登录获取token
echo "登录获取token..."
login_response=$(curl -s -X POST "${POSTGREST_URL}/rpc/get_token" \
    -H "Content-Type: application/json" \
    -H "Accept-Profile: postgrest" \
    -H "Content-Profile: postgrest" \
    -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")

if command -v jq >/dev/null 2>&1; then
    ACCESS_TOKEN=$(echo "$login_response" | jq -r '.access_token // ""')
    TOKEN="$ACCESS_TOKEN"  # 使用access token进行后续API调用
else
    ACCESS_TOKEN=$(echo "$login_response" | sed -n 's/.*"access_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
    TOKEN="$ACCESS_TOKEN"  # 使用access token进行后续API调用
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo -e "${RED}登录失败，无法获取token${NC}"
    exit 1
fi

echo -e "${GREEN}登录成功${NC}"

# 清理函数
cleanup_table() {
    local table=$1
    local condition=$2
    local description=$3
    
    echo "清理 $description..."
    response=$(curl -s -X DELETE "${POSTGREST_URL}/${table}?${condition}" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Accept-Profile: public" \
        -H "Content-Profile: public")
    
    if echo "$response" | grep -q '"code"' && ! echo "$response" | grep -q '^\['; then
        echo -e "${RED}清理 $description 失败: $response${NC}"
    else
        echo -e "${GREEN}清理 $description 成功${NC}"
    fi
}

# 按依赖关系倒序清理
cleanup_table "data_interfaces" "name_en=eq.user_info_interface" "测试数据接口"
cleanup_table "thematic_interfaces" "name_en=eq.user_analysis_interface" "测试主题接口"
cleanup_table "quality_rules" "name=eq.用户邮箱完整性检查" "测试质量规则"
cleanup_table "metadata" "name=eq.用户表技术元数据" "测试元数据"
cleanup_table "data_masking_rules" "name=eq.用户邮箱脱敏规则" "测试脱敏规则"
cleanup_table "api_applications" "name=eq.测试应用" "测试API应用"
cleanup_table "data_subscriptions" "subscriber_id=eq.test_user_123" "测试数据订阅"
cleanup_table "data_sync_tasks" "name=eq.用户数据同步任务" "测试同步任务"
cleanup_table "system_logs" "operator_id=eq.admin" "测试系统日志"
cleanup_table "thematic_libraries" "code=eq.test_user_thematic" "测试主题库"
cleanup_table "basic_libraries" "name_en=eq.test_user_basic_library" "测试基础库"

# 清理快速测试数据
cleanup_table "thematic_libraries" "code=eq.quick_test_thematic" "快速测试主题库"
cleanup_table "basic_libraries" "name_en=eq.quick_test_library" "快速测试基础库"

echo -e "${GREEN}=== 测试数据清理完成 ===${NC}" 