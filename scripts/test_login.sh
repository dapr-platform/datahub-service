#!/bin/bash

#
# @module scripts/test_login
# @description 双token机制登录测试脚本 - 验证RBAC登录功能和token刷新
# @architecture 测试脚本
# @documentReference ../ai_docs/postgrest_rbac_guide.md
# @stateFlow 登录测试 -> 解析双token -> 测试token刷新 -> 测试token撤销 -> 显示用户信息
# @rules 验证双token登录功能和refresh token机制
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

echo -e "${BLUE}=== PostgREST 双Token机制登录功能测试 ===${NC}"

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

# 测试登录获取双token
echo -e "\n${BLUE}3. 测试双Token登录...${NC}"
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
        ACCESS_TOKEN=$(echo "$login_response" | jq -r '.access_token // ""')
        REFRESH_TOKEN=$(echo "$login_response" | jq -r '.refresh_token // ""')
        USERNAME_RESP=$(echo "$login_response" | jq -r '.username // "unknown"')
        ROLES=$(echo "$login_response" | jq -r '.roles // [] | join(", ")')
        PERMISSIONS_COUNT=$(echo "$login_response" | jq -r '.permissions // [] | length')
        
        echo -e "\n${GREEN}✓ 双Token登录成功！${NC}"
        echo "用户名: $USERNAME_RESP"
        echo "角色: $ROLES"
        echo "权限数量: $PERMISSIONS_COUNT"
        echo "Access Token长度: ${#ACCESS_TOKEN}"
        echo "Access Token前50字符: ${ACCESS_TOKEN:0:50}..."
        echo "Refresh Token长度: ${#REFRESH_TOKEN}"
        echo "Refresh Token前50字符: ${REFRESH_TOKEN:0:50}..."
        
        # 测试access token是否有效
        echo -e "\n${BLUE}4. 测试Access Token有效性...${NC}"
        test_response=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
            -H "Accept-Profile: public" \
            "${POSTGREST_URL}/basic_libraries?limit=1")
        
        echo "API测试响应: $test_response"
        
        # 检查是否包含错误代码
        if echo "$test_response" | grep -q '"code"'; then
            echo -e "${RED}✗ Access Token无效或API访问失败${NC}"
            if command -v jq >/dev/null 2>&1; then
                error_code=$(echo "$test_response" | jq -r '.code // "unknown"')
                error_message=$(echo "$test_response" | jq -r '.message // "未知错误"')
                echo "错误代码: $error_code"
                echo "错误信息: $error_message"
            fi
        else
            echo -e "${GREEN}✓ Access Token有效，可以访问API${NC}"
            if command -v jq >/dev/null 2>&1; then
                record_count=$(echo "$test_response" | jq '. | length' 2>/dev/null || echo "0")
                echo "返回记录数: $record_count"
            fi
        fi
        
        # 测试refresh token功能
        echo -e "\n${BLUE}5. 测试Refresh Token功能...${NC}"
        refresh_response=$(curl -s -X POST "${POSTGREST_URL}/rpc/refresh_token" \
            -H "Content-Type: application/json" \
            -H "Accept-Profile: postgrest" \
            -H "Content-Profile: postgrest" \
            -d "{\"refresh_token\": \"${REFRESH_TOKEN}\"}")
        
        echo "Refresh Token响应: $refresh_response"
        
        refresh_success=$(echo "$refresh_response" | jq -r '.success // false' 2>/dev/null)
        if [ "$refresh_success" = "true" ]; then
            NEW_ACCESS_TOKEN=$(echo "$refresh_response" | jq -r '.access_token // ""')
            NEW_REFRESH_TOKEN=$(echo "$refresh_response" | jq -r '.refresh_token // ""')
            
            echo -e "${GREEN}✓ Token刷新成功！${NC}"
            echo "新Access Token长度: ${#NEW_ACCESS_TOKEN}"
            echo "新Access Token前50字符: ${NEW_ACCESS_TOKEN:0:50}..."
            
            if [ -n "$NEW_REFRESH_TOKEN" ]; then
                echo "新Refresh Token长度: ${#NEW_REFRESH_TOKEN}"
                echo "新Refresh Token前50字符: ${NEW_REFRESH_TOKEN:0:50}..."
                REFRESH_TOKEN="$NEW_REFRESH_TOKEN"  # 更新refresh token用于后续测试
            else
                echo "未返回新的Refresh Token（使用原Token）"
            fi
            
            # 使用新access token测试API
            echo -e "\n${BLUE}6. 测试新Access Token有效性...${NC}"
            new_test_response=$(curl -s -H "Authorization: Bearer $NEW_ACCESS_TOKEN" \
                -H "Accept-Profile: public" \
                "${POSTGREST_URL}/basic_libraries?limit=1")
            
            if echo "$new_test_response" | grep -q '"code"'; then
                echo -e "${RED}✗ 新Access Token无效${NC}"
            else
                echo -e "${GREEN}✓ 新Access Token有效${NC}"
            fi
        else
            echo -e "${RED}✗ Token刷新失败${NC}"
            refresh_error=$(echo "$refresh_response" | jq -r '.message // "未知错误"')
            echo "错误信息: $refresh_error"
        fi
        
        # 测试撤销refresh token
        echo -e "\n${BLUE}7. 测试Refresh Token撤销...${NC}"
        revoke_response=$(curl -s -X POST "${POSTGREST_URL}/rpc/revoke_refresh_token" \
            -H "Content-Type: application/json" \
            -H "Accept-Profile: postgrest" \
            -H "Content-Profile: postgrest" \
            -d "{\"refresh_token\": \"${REFRESH_TOKEN}\"}")
        
        echo "撤销Token响应: $revoke_response"
        
        revoke_success=$(echo "$revoke_response" | jq -r '.success // false' 2>/dev/null)
        if [ "$revoke_success" = "true" ]; then
            echo -e "${GREEN}✓ Refresh Token撤销成功！${NC}"
            
            # 验证撤销后的token无法再使用
            echo -e "\n${BLUE}8. 验证撤销后Token无效...${NC}"
            verify_response=$(curl -s -X POST "${POSTGREST_URL}/rpc/refresh_token" \
                -H "Content-Type: application/json" \
                -H "Accept-Profile: postgrest" \
                -H "Content-Profile: postgrest" \
                -d "{\"refresh_token\": \"${REFRESH_TOKEN}\"}")
            
            verify_success=$(echo "$verify_response" | jq -r '.success // false' 2>/dev/null)
            if [ "$verify_success" = "false" ]; then
                echo -e "${GREEN}✓ 撤销验证成功，Token已无效${NC}"
            else
                echo -e "${RED}✗ 撤销验证失败，Token仍可使用${NC}"
            fi
        else
            echo -e "${RED}✗ Refresh Token撤销失败${NC}"
            revoke_error=$(echo "$revoke_response" | jq -r '.message // "未知错误"')
            echo "错误信息: $revoke_error"
        fi
        
    else
        echo -e "\n${RED}✗ 登录失败${NC}"
        echo "错误信息: $(echo "$login_response" | jq -r '.message // "未知错误"')"
    fi
else
    echo -e "\n${YELLOW}使用基础解析方式:${NC}"
    if echo "$login_response" | grep -q '"success".*true'; then
        echo -e "${GREEN}✓ 登录成功（基础解析）${NC}"
        ACCESS_TOKEN=$(echo "$login_response" | sed -n 's/.*"access_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        REFRESH_TOKEN=$(echo "$login_response" | sed -n 's/.*"refresh_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        echo "Access Token: ${ACCESS_TOKEN:0:50}..."
        echo "Refresh Token: ${REFRESH_TOKEN:0:50}..."
    else
        echo -e "${RED}✗ 登录失败（基础解析）${NC}"
    fi
fi

echo -e "\n${GREEN}=== 双Token机制测试完成 ===${NC}" 