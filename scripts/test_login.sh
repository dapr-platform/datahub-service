#!/bin/bash

#
# @module scripts/test_login
# @description 简单的登录测试脚本 - 验证RBAC登录功能
# @architecture 测试脚本
# @documentReference ../ai_docs/postgrest_rbac_guide.md
# @stateFlow 登录测试 -> 显示用户信息
# @rules 验证登录功能和token获取
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
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}=== PostgREST 登录功能测试 ===${NC}"

# 检查PostgREST服务
echo -e "\n${BLUE}1. 检查PostgREST服务...${NC}"
if curl -s "${POSTGREST_URL}/" > /dev/null; then
    echo -e "${GREEN}✓ PostgREST服务正常${NC}"
else
    echo -e "${RED}✗ PostgREST服务不可用${NC}"
    exit 1
fi

# 检查jq工具
echo -e "\n${BLUE}2. 检查工具依赖...${NC}"
if command -v jq >/dev/null 2>&1; then
    echo -e "${GREEN}✓ jq工具可用${NC}"
else
    echo -e "${YELLOW}⚠ jq工具不可用，将使用基础解析${NC}"
fi

# 测试登录
echo -e "\n${BLUE}3. 测试RBAC登录...${NC}"
echo "请求URL: ${POSTGREST_URL}/rpc/get_token"
echo "用户名: ${USERNAME}"

login_response=$(curl -s -X POST "${POSTGREST_URL}/rpc/get_token" \
    -H "Content-Type: application/json" \
    -H "Accept-Profile: postgrest" \
    -H "Content-Profile: postgrest" \
    -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")

echo -e "\n${BLUE}原始响应:${NC}"
echo "$login_response"

# 使用jq解析响应
if command -v jq >/dev/null 2>&1; then
    echo -e "\n${BLUE}解析后的响应:${NC}"
    echo "$login_response" | jq '.'
    
    success=$(echo "$login_response" | jq -r '.success // false' 2>/dev/null)
    echo -e "\n${BLUE}登录状态检查:${NC}"
    echo "success字段值: $success"
    
    if [ "$success" = "true" ]; then
        TOKEN=$(echo "$login_response" | jq -r '.token // ""')
        USERNAME_RESP=$(echo "$login_response" | jq -r '.username // "unknown"')
        ROLES=$(echo "$login_response" | jq -r '.roles // [] | join(", ")')
        PERMISSIONS_COUNT=$(echo "$login_response" | jq -r '.permissions // [] | length')
        
        echo -e "\n${GREEN}✓ 登录成功！${NC}"
        echo "用户名: $USERNAME_RESP"
        echo "角色: $ROLES"
        echo "权限数量: $PERMISSIONS_COUNT"
        echo "Token长度: ${#TOKEN}"
        echo "Token前50字符: ${TOKEN:0:50}..."
        
        # 测试token是否有效
        echo -e "\n${BLUE}4. 测试token有效性...${NC}"
        test_response=$(curl -s -H "Authorization: Bearer $TOKEN" \
            -H "Accept-Profile: public" \
            "${POSTGREST_URL}/basic_libraries?limit=1")
        
        echo "API测试响应: $test_response"
        
        # 检查是否包含错误代码
        if echo "$test_response" | grep -q '"code"'; then
            echo -e "${RED}✗ Token无效或API访问失败${NC}"
            if command -v jq >/dev/null 2>&1; then
                error_code=$(echo "$test_response" | jq -r '.code // "unknown"')
                error_message=$(echo "$test_response" | jq -r '.message // "未知错误"')
                echo "错误代码: $error_code"
                echo "错误信息: $error_message"
            fi
        else
            echo -e "${GREEN}✓ Token有效，可以访问API${NC}"
            if command -v jq >/dev/null 2>&1; then
                record_count=$(echo "$test_response" | jq '. | length' 2>/dev/null || echo "0")
                echo "返回记录数: $record_count"
            fi
        fi
    else
        echo -e "\n${RED}✗ 登录失败${NC}"
        echo "错误信息: $(echo "$login_response" | jq -r '.message // "未知错误"')"
    fi
else
    echo -e "\n${YELLOW}使用基础解析方式:${NC}"
    if echo "$login_response" | grep -q '"success".*true'; then
        echo -e "${GREEN}✓ 登录成功（基础解析）${NC}"
        TOKEN=$(echo "$login_response" | sed -n 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        echo "Token: ${TOKEN:0:50}..."
    else
        echo -e "${RED}✗ 登录失败（基础解析）${NC}"
    fi
fi

echo -e "\n${GREEN}=== 登录测试完成 ===${NC}" 