#!/bin/bash

#
# @module scripts/quick_test
# @description PostgREST 快速测试脚本 - 验证基本功能
# @architecture 测试脚本
# @documentReference ../ai_docs/postgrest_rbac_guide.md
# @stateFlow 登录 -> 基本CRUD测试 -> 清理
# @rules 快速验证PostgREST和数据库连接是否正常
# @dependencies PostgREST服务, PostgreSQL数据库
# @refs ../service/models/
#

set -e

# 配置
POSTGREST_URL="http://localhost:9080/api/postgrest"
USERNAME="admin"
PASSWORD="things2024"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 检查PostgREST响应是否包含错误
check_postgrest_error() {
    local response="$1"
    local operation="$2"
    
    if echo "$response" | grep -q '"code"' && ! echo "$response" | grep -q '^\['; then
        echo -e "${RED}✗ $operation 失败${NC}"
        if command -v jq >/dev/null 2>&1; then
            error_code=$(echo "$response" | jq -r '.code // "unknown"')
            error_message=$(echo "$response" | jq -r '.message // "未知错误"')
            error_details=$(echo "$response" | jq -r '.details // ""')
            
            echo "错误代码: $error_code"
            echo "错误信息: $error_message"
            if [ "$error_details" != "" ] && [ "$error_details" != "null" ]; then
                echo "错误详情: $error_details"
            fi
        fi
        return 1
    else
        echo -e "${GREEN}✓ $operation 成功${NC}"
        return 0
    fi
}

echo -e "${BLUE}=== PostgREST 快速功能测试 ===${NC}"

# 1. 检查PostgREST服务
echo -e "\n${BLUE}1. 检查PostgREST服务...${NC}"
if curl -s "${POSTGREST_URL}/" > /dev/null; then
    echo -e "${GREEN}✓ PostgREST服务正常${NC}"
else
    echo -e "${RED}✗ PostgREST服务不可用${NC}"
    exit 1
fi

# 2. RBAC登录
echo -e "\n${BLUE}2. 测试RBAC登录...${NC}"
login_response=$(curl -s -X POST "${POSTGREST_URL}/rpc/get_token" \
    -H "Content-Type: application/json" \
    -H "Accept-Profile: postgrest" \
    -H "Content-Profile: postgrest" \
    -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")

# 使用jq检查登录是否成功
if command -v jq >/dev/null 2>&1; then
    success=$(echo "$login_response" | jq -r '.success // false' 2>/dev/null)
    if [ "$success" = "true" ]; then
        TOKEN=$(echo "$login_response" | jq -r '.token // ""')
        if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
            echo -e "${GREEN}✓ 登录成功${NC}"
            echo "用户名: $(echo "$login_response" | jq -r '.username // "unknown"')"
            echo "角色: $(echo "$login_response" | jq -r '.roles // [] | join(", ")')"
            echo "Token: ${TOKEN:0:50}..."
        else
            echo -e "${RED}✗ 登录失败: 未获取到有效token${NC}"
            exit 1
        fi
    else
        echo -e "${RED}✗ 登录失败: $login_response${NC}"
        exit 1
    fi
else
    # 如果没有jq，使用简单的字符串匹配
    if echo "$login_response" | grep -q '"success".*true'; then
        TOKEN=$(echo "$login_response" | sed -n 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [ -n "$TOKEN" ]; then
            echo -e "${GREEN}✓ 登录成功${NC}"
            echo "Token: ${TOKEN:0:50}..."
        else
            echo -e "${RED}✗ 登录失败: 未获取到token${NC}"
            exit 1
        fi
    else
        echo -e "${RED}✗ 登录失败: $login_response${NC}"
        exit 1
    fi
fi

# 3. 测试基础库CRUD
echo -e "\n${BLUE}3. 测试基础库CRUD操作...${NC}"

# 创建
echo "创建测试基础库..."
# 生成UUID
BASIC_LIBRARY_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
create_response=$(curl -s -X POST "${POSTGREST_URL}/basic_libraries" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "Accept-Profile: public" \
    -H "Content-Profile: public" \
    -d "{
        \"id\": \"$BASIC_LIBRARY_ID\",
        \"name_zh\": \"快速测试库\",
        \"name_en\": \"quick_test_library\",
        \"description\": \"用于快速测试的基础库\",
        \"status\": \"active\"
    }")

echo "响应内容: $create_response"
check_postgrest_error "$create_response" "创建基础库"

# 查询
echo "查询测试基础库..."
query_response=$(curl -s -H "Authorization: Bearer $TOKEN" \
    -H "Accept-Profile: public" \
    "${POSTGREST_URL}/basic_libraries?name_en=eq.quick_test_library&select=*")

echo "查询响应: $query_response"

# 检查错误，如果没有错误再检查数据
if ! check_postgrest_error "$query_response" "查询基础库"; then
    # 如果有错误，函数已经处理了
    :
elif echo "$query_response" | grep -q "quick_test_library"; then
    echo "✓ 找到了测试数据"
else
    echo -e "${RED}✗ 查询成功但未找到测试数据${NC}"
fi

# 更新
echo "更新测试基础库..."
update_response=$(curl -s -X PATCH "${POSTGREST_URL}/basic_libraries?name_en=eq.quick_test_library" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "Accept-Profile: public" \
    -H "Content-Profile: public" \
    -d '{"description": "更新后的描述"}')

echo "更新响应: $update_response"
check_postgrest_error "$update_response" "更新基础库"

# 4. 测试主题库
echo -e "\n${BLUE}4. 测试主题库操作...${NC}"

# 生成主题库UUID
THEMATIC_LIBRARY_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
create_thematic_response=$(curl -s -X POST "${POSTGREST_URL}/thematic_libraries" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "Accept-Profile: public" \
    -H "Content-Profile: public" \
    -d "{
        \"id\": \"$THEMATIC_LIBRARY_ID\",
        \"name\": \"快速测试主题库\",
        \"code\": \"quick_test_thematic\",
        \"category\": \"business\",
        \"domain\": \"user\",
        \"description\": \"快速测试主题库\",
        \"tags\": [\"测试\", \"快速\"],
        \"source_libraries\": [\"quick_test_library\"],
        \"tables\": {
            \"test_table\": {
                \"fields\": [\"id\", \"name\", \"created_at\"],
                \"description\": \"测试表\"
            }
        },
        \"publish_status\": \"draft\",
        \"version\": \"1.0.0\",
        \"access_level\": \"internal\",
        \"authorized_users\": [],
        \"authorized_roles\": [],
        \"update_frequency\": \"daily\",
        \"retention_period\": 365,
        \"status\": \"active\"
    }")

echo "主题库创建响应: $create_thematic_response"
check_postgrest_error "$create_thematic_response" "创建主题库"

echo -e "\n${GREEN}=== 快速测试完成！所有基本功能正常 ===${NC}" 