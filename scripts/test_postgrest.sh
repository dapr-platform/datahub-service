#!/bin/bash

#
# @module scripts/test_postgrest
# @description PostgREST 接口完整测试脚本 - 包括RBAC登录和所有模型表的CRUD操作
# @architecture 测试脚本
# @documentReference ../ai_docs/postgrest_rbac_guide.md
# @stateFlow 登录获取token -> 测试各表CRUD操作
# @rules 
# - 使用admin用户登录获取管理员权限
# - 测试所有模型表的增删改查操作
# - 验证数据完整性和关联关系
# @dependencies PostgREST服务, PostgreSQL数据库
# @refs ../service/models/
#

set -e

# 配置
POSTGREST_URL="http://localhost:3000"
USERNAME="admin"
PASSWORD="things2024"
TOKEN=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# UUID生成函数
generate_uuid() {
    if command -v uuidgen >/dev/null 2>&1; then
        uuidgen | tr '[:upper:]' '[:lower:]'
    else
        # 如果没有uuidgen，使用简单的随机字符串
        cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$(date +%s)-$(shuf -i 1000-9999 -n 1)-$(shuf -i 1000-9999 -n 1)-$(shuf -i 1000-9999 -n 1)-$(shuf -i 100000000000-999999999999 -n 1)"
    fi
}

# 检查PostgREST服务是否可用
check_postgrest() {
    log_info "检查PostgREST服务状态..."
    if curl -s "${POSTGREST_URL}/" > /dev/null; then
        log_success "PostgREST服务正常运行"
    else
        log_error "PostgREST服务不可用，请检查服务是否启动"
        exit 1
    fi
}

# 双Token RBAC登录获取token
login() {
    log_info "使用admin用户进行双Token登录..."
    
    response=$(curl -s -X POST "${POSTGREST_URL}/rpc/get_token" \
        -H "Content-Type: application/json" \
        -H "Accept-Profile: postgrest" \
        -H "Content-Profile: postgrest" \
        -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")
    
    # 使用jq检查登录是否成功
    if command -v jq >/dev/null 2>&1; then
        success=$(echo "$response" | jq -r '.success // false' 2>/dev/null)
        if [ "$success" = "true" ]; then
            ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token // ""')
            REFRESH_TOKEN=$(echo "$response" | jq -r '.refresh_token // ""')
            if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]; then
                TOKEN="$ACCESS_TOKEN"  # 使用access token进行API调用
                log_success "双Token登录成功，获取到access_token和refresh_token"
                echo "用户名: $(echo "$response" | jq -r '.username // "unknown"')"
                echo "角色: $(echo "$response" | jq -r '.roles // [] | join(", ")')"
                echo "权限数量: $(echo "$response" | jq -r '.permissions // [] | length')"
                echo "Access Token: ${ACCESS_TOKEN:0:50}..."
                echo "Refresh Token: ${REFRESH_TOKEN:0:50}..."
            else
                log_error "登录失败: 未获取到有效access_token"
                exit 1
            fi
        else
            log_error "登录失败: $response"
            exit 1
        fi
    else
        # 如果没有jq，使用简单的字符串匹配
        if echo "$response" | grep -q '"success".*true'; then
            ACCESS_TOKEN=$(echo "$response" | sed -n 's/.*"access_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
            REFRESH_TOKEN=$(echo "$response" | sed -n 's/.*"refresh_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
            if [ -n "$ACCESS_TOKEN" ]; then
                TOKEN="$ACCESS_TOKEN"  # 使用access token进行API调用
                log_success "双Token登录成功，获取到access_token和refresh_token"
                echo "Access Token: ${ACCESS_TOKEN:0:50}..."
                echo "Refresh Token: ${REFRESH_TOKEN:0:50}..."
            else
                log_error "登录失败: 未获取到access_token"
                exit 1
            fi
        else
            log_error "登录失败: $response"
            exit 1
        fi
    fi
}

# 通用API调用函数
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    log_info "$description"
    
    local curl_cmd="curl -s -X $method \"${POSTGREST_URL}$endpoint\""
    curl_cmd="$curl_cmd -H \"Authorization: Bearer $TOKEN\""
    curl_cmd="$curl_cmd -H \"Content-Type: application/json\""
    curl_cmd="$curl_cmd -H \"Accept: application/json\""
    curl_cmd="$curl_cmd -H \"Accept-Profile: public\""
    
    # 对于有数据的请求（POST, PATCH, PUT, DELETE），添加Content-Profile
    if [ "$method" = "POST" ] || [ "$method" = "PATCH" ] || [ "$method" = "PUT" ] || [ "$method" = "DELETE" ]; then
        curl_cmd="$curl_cmd -H \"Content-Profile: public\""
    fi
    
    if [ "$data" != "" ]; then
        curl_cmd="$curl_cmd -d '$data'"
    fi
    
    response=$(eval $curl_cmd)
    
    # 检查PostgREST错误响应 - 只有当响应是对象且包含code字段时才是错误
    if echo "$response" | grep -q '"code"' && ! echo "$response" | grep -q '^\['; then
        log_error "$description - 失败"
        echo "Response: $response"
        
        # 使用jq解析错误信息
        if command -v jq >/dev/null 2>&1; then
            error_code=$(echo "$response" | jq -r '.code // "unknown"' 2>/dev/null)
            error_message=$(echo "$response" | jq -r '.message // "未知错误"' 2>/dev/null)
            error_details=$(echo "$response" | jq -r '.details // ""' 2>/dev/null)
            error_hint=$(echo "$response" | jq -r '.hint // ""' 2>/dev/null)
            
            echo "错误代码: $error_code"
            echo "错误信息: $error_message"
            if [ "$error_details" != "" ] && [ "$error_details" != "null" ]; then
                echo "错误详情: $error_details"
            fi
            if [ "$error_hint" != "" ] && [ "$error_hint" != "null" ]; then
                echo "提示: $error_hint"
            fi
        fi
        echo ""
        return 1
    else
        log_success "$description - 成功"
        # 使用jq美化JSON输出，如果失败则直接输出原始响应
        if command -v jq >/dev/null 2>&1 && echo "$response" | jq empty 2>/dev/null; then
            echo "Response:"
            echo "$response" | jq '.' 2>/dev/null || echo "$response"
        else
            echo "Response: $response"
        fi
        echo ""
        return 0
    fi
}

# 测试基础库管理
test_basic_libraries() {
    log_info "=== 测试数据基础库管理 ==="
    
    # 创建基础库
    local basic_library_id=$(generate_uuid)
    local basic_library_data="{
        \"id\": \"$basic_library_id\",
        \"name_zh\": \"测试用户基础库\",
        \"name_en\": \"test_user_basic_library\",
        \"description\": \"用于测试的用户基础数据库\",
        \"status\": \"active\"
    }"
    
    api_call "POST" "/basic_libraries" "$basic_library_data" "创建数据基础库"
    
    # 获取基础库列表
    api_call "GET" "/basic_libraries?select=*&limit=10" "" "获取数据基础库列表"
    
    # 获取特定基础库（假设ID存在）
    api_call "GET" "/basic_libraries?name_en=eq.test_user_basic_library&select=*" "" "根据英文名称查询基础库"
    
    # 更新基础库
    local update_data='{
        "description": "更新后的用户基础数据库描述"
    }'
    
    api_call "PATCH" "/basic_libraries?name_en=eq.test_user_basic_library" "$update_data" "更新数据基础库"
}

# 测试数据接口管理
test_data_interfaces() {
    log_info "=== 测试数据接口管理 ==="
    
    # 首先获取基础库ID
    local library_response=$(curl -s -H "Authorization: Bearer $TOKEN" \
        -H "Accept-Profile: public" \
        "${POSTGREST_URL}/basic_libraries?name_en=eq.test_user_basic_library&select=id")
    local library_id=$(echo "$library_response" | jq -r '.[0].id' 2>/dev/null)
    
    if [ "$library_id" != "null" ] && [ "$library_id" != "" ]; then
        # 创建数据接口
        local interface_id=$(generate_uuid)
        local interface_data="{
            \"id\": \"$interface_id\",
            \"library_id\": \"$library_id\",
            \"name_zh\": \"用户信息接口\",
            \"name_en\": \"user_info_interface\",
            \"type\": \"realtime\",
            \"description\": \"实时用户信息数据接口\",
            \"status\": \"active\"
        }"
        
        api_call "POST" "/data_interfaces" "$interface_data" "创建数据接口"
        
        # 获取数据接口列表
        api_call "GET" "/data_interfaces?library_id=eq.$library_id&select=*" "" "获取数据接口列表"
        
        # 更新数据接口
        local update_interface_data='{
            "description": "更新后的用户信息数据接口"
        }'
        
        api_call "PATCH" "/data_interfaces?name_en=eq.user_info_interface" "$update_interface_data" "更新数据接口"
    else
        log_warning "未找到测试基础库，跳过数据接口测试"
    fi
}

# 测试主题库管理
test_thematic_libraries() {
    log_info "=== 测试数据主题库管理 ==="
    
    # 创建主题库
    local thematic_library_id=$(generate_uuid)
    local thematic_library_data="{
        \"id\": \"$thematic_library_id\",
        \"name\": \"测试用户主题库\",
        \"code\": \"test_user_thematic\",
        \"category\": \"business\",
        \"domain\": \"user\",
        \"description\": \"用于测试的用户主题数据库\",
        \"tags\": [\"用户\", \"测试\"],
        \"source_libraries\": [\"test_user_basic_library\"],
        \"tables\": {
            \"users\": {
                \"fields\": [\"id\", \"name\", \"email\"],
                \"description\": \"用户信息表\"
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
    }"
    
    api_call "POST" "/thematic_libraries" "$thematic_library_data" "创建数据主题库"
    
    # 获取主题库列表
    api_call "GET" "/thematic_libraries?select=*&limit=10" "" "获取数据主题库列表"
    
    # 根据分类查询
    api_call "GET" "/thematic_libraries?category=eq.business&select=*" "" "根据分类查询主题库"
    
    # 更新主题库
    local update_thematic_data='{
        "description": "更新后的用户主题数据库描述",
        "version": "1.1.0"
    }'
    
    api_call "PATCH" "/thematic_libraries?code=eq.test_user_thematic" "$update_thematic_data" "更新数据主题库"
}

# 测试主题接口管理
test_thematic_interfaces() {
    log_info "=== 测试主题接口管理 ==="
    
    # 获取主题库ID
    local thematic_response=$(curl -s -H "Authorization: Bearer $TOKEN" \
        -H "Accept-Profile: public" \
        "${POSTGREST_URL}/thematic_libraries?code=eq.test_user_thematic&select=id")
    local thematic_id=$(echo "$thematic_response" | jq -r '.[0].id' 2>/dev/null)
    
    if [ "$thematic_id" != "null" ] && [ "$thematic_id" != "" ]; then
        # 创建主题接口
        local thematic_interface_id=$(generate_uuid)
        local thematic_interface_data="{
            \"id\": \"$thematic_interface_id\",
            \"library_id\": \"$thematic_id\",
            \"name_zh\": \"用户分析接口\",
            \"name_en\": \"user_analysis_interface\",
            \"type\": \"http\",
            \"config\": {\"endpoint\": \"/api/user/analysis\", \"method\": \"GET\"},
            \"description\": \"用户数据分析接口\",
            \"status\": \"active\"
        }"
        
        api_call "POST" "/thematic_interfaces" "$thematic_interface_data" "创建主题接口"
        
        # 获取主题接口列表
        api_call "GET" "/thematic_interfaces?library_id=eq.$thematic_id&select=*" "" "获取主题接口列表"
    else
        log_warning "未找到测试主题库，跳过主题接口测试"
    fi
}

# 测试数据质量规则
test_quality_rules() {
    log_info "=== 测试数据质量规则管理 ==="
    
    # 创建数据质量规则
    local quality_rule_id=$(generate_uuid)
    local quality_rule_data="{
        \"id\": \"$quality_rule_id\",
        \"name\": \"用户邮箱完整性检查\",
        \"type\": \"completeness\",
        \"config\": {
            \"field\": \"email\",
            \"required\": true,
            \"check_format\": true,
            \"pattern\": \"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\\\.[a-zA-Z]{2,}$\"
        },
        \"related_object_id\": \"test-object-id\",
        \"related_object_type\": \"interface\",
        \"is_enabled\": true
    }"
    
    api_call "POST" "/quality_rules" "$quality_rule_data" "创建数据质量规则"
    
    # 获取质量规则列表
    api_call "GET" "/quality_rules?select=*&limit=10" "" "获取数据质量规则列表"
    
    # 根据类型查询
    api_call "GET" "/quality_rules?type=eq.completeness&select=*" "" "根据类型查询质量规则"
}

# 测试元数据管理
test_metadata() {
    log_info "=== 测试元数据管理 ==="
    
    # 创建元数据
    local metadata_id=$(generate_uuid)
    local metadata_data="{
        \"id\": \"$metadata_id\",
        \"type\": \"technical\",
        \"name\": \"用户表技术元数据\",
        \"content\": {
            \"table_name\": \"users\",
            \"columns\": [
                {\"name\": \"id\", \"type\": \"uuid\", \"nullable\": false, \"primary_key\": true},
                {\"name\": \"name\", \"type\": \"varchar(255)\", \"nullable\": false},
                {\"name\": \"email\", \"type\": \"varchar(255)\", \"nullable\": true},
                {\"name\": \"created_at\", \"type\": \"timestamp\", \"nullable\": false, \"default\": \"CURRENT_TIMESTAMP\"},
                {\"name\": \"updated_at\", \"type\": \"timestamp\", \"nullable\": false, \"default\": \"CURRENT_TIMESTAMP\"}
            ],
            \"indexes\": [
                {\"name\": \"idx_users_id\", \"columns\": [\"id\"], \"unique\": true},
                {\"name\": \"idx_users_email\", \"columns\": [\"email\"], \"unique\": false}
            ],
            \"constraints\": [
                {\"type\": \"PRIMARY KEY\", \"columns\": [\"id\"]},
                {\"type\": \"UNIQUE\", \"columns\": [\"email\"]}
            ]
        },
        \"related_object_id\": \"test-library-id\",
        \"related_object_type\": \"basic_library\"
    }"
    
    api_call "POST" "/metadata" "$metadata_data" "创建元数据"
    
    # 获取元数据列表
    api_call "GET" "/metadata?select=*&limit=10" "" "获取元数据列表"
    
    # 根据类型查询
    api_call "GET" "/metadata?type=eq.technical&select=*" "" "根据类型查询元数据"
}

# 测试数据脱敏规则
test_masking_rules() {
    log_info "=== 测试数据脱敏规则管理 ==="
    
    # 创建脱敏规则
    local masking_rule_id=$(generate_uuid)
    local masking_rule_data="{
        \"id\": \"$masking_rule_id\",
        \"name\": \"用户邮箱脱敏规则\",
        \"data_source\": \"users_table\",
        \"data_table\": \"users\",
        \"field_name\": \"email\",
        \"field_type\": \"varchar\",
        \"masking_type\": \"mask\",
        \"masking_config\": {
            \"pattern\": \"***@***.com\",
            \"preserve_domain\": false,
            \"mask_char\": \"*\",
            \"preserve_length\": true
        },
        \"is_enabled\": true,
        \"creator_id\": \"admin\",
        \"creator_name\": \"系统管理员\"
    }"
    
    api_call "POST" "/data_masking_rules" "$masking_rule_data" "创建数据脱敏规则"
    
    # 获取脱敏规则列表
    api_call "GET" "/data_masking_rules?select=*&limit=10" "" "获取数据脱敏规则列表"
}

# 测试API应用管理
test_api_applications() {
    log_info "=== 测试API应用管理 ==="
    
    # 创建API应用
    local api_app_id=$(generate_uuid)
    local api_app_data="{
        \"id\": \"$api_app_id\",
        \"name\": \"测试应用\",
        \"app_key\": \"test_app_key_123\",
        \"app_secret_hash\": \"\$2a\$10\$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy\",
        \"description\": \"用于测试的API应用\",
        \"contact_person\": \"测试人员\",
        \"contact_email\": \"test@example.com\",
        \"status\": \"active\"
    }"
    
    api_call "POST" "/api_applications" "$api_app_data" "创建API应用"
    
    # 获取API应用列表
    api_call "GET" "/api_applications?select=*&limit=10" "" "获取API应用列表"
}

# 测试数据订阅
test_data_subscriptions() {
    log_info "=== 测试数据订阅管理 ==="
    
    # 创建数据订阅
    local subscription_id=$(generate_uuid)
    local subscription_data="{
        \"id\": \"$subscription_id\",
        \"subscriber_id\": \"test_user_123\",
        \"subscriber_type\": \"user\",
        \"resource_id\": \"test_resource_456\",
        \"resource_type\": \"thematic_interface\",
        \"notification_method\": \"webhook\",
        \"notification_config\": {
            \"url\": \"https://example.com/webhook\",
            \"headers\": {\"Authorization\": \"Bearer token\"},
            \"timeout\": 30,
            \"retry_count\": 3
        },
        \"filter_condition\": {
            \"status\": \"active\",
            \"data_type\": \"user_data\"
        },
        \"status\": \"active\"
    }"
    
    api_call "POST" "/data_subscriptions" "$subscription_data" "创建数据订阅"
    
    # 获取数据订阅列表
    api_call "GET" "/data_subscriptions?select=*&limit=10" "" "获取数据订阅列表"
}

# 测试数据同步任务
test_sync_tasks() {
    log_info "=== 测试数据同步任务管理 ==="
    
    # 创建同步任务
    local sync_task_id=$(generate_uuid)
    local sync_task_data="{
        \"id\": \"$sync_task_id\",
        \"name\": \"用户数据同步任务\",
        \"source_type\": \"database\",
        \"source_config\": {
            \"host\": \"source.db.com\",
            \"port\": 5432,
            \"database\": \"source_db\",
            \"username\": \"source_user\",
            \"table\": \"users\",
            \"connection_timeout\": 30
        },
        \"target_type\": \"database\",
        \"target_config\": {
            \"host\": \"target.db.com\",
            \"port\": 5432,
            \"database\": \"target_db\",
            \"username\": \"target_user\",
            \"table\": \"users_copy\",
            \"connection_timeout\": 30
        },
        \"sync_strategy\": \"incremental\",
        \"schedule_config\": {
            \"cron\": \"0 2 * * *\",
            \"timezone\": \"Asia/Shanghai\",
            \"enabled\": true
        },
        \"transform_rules\": {
            \"field_mapping\": {
                \"id\": \"user_id\",
                \"name\": \"user_name\",
                \"email\": \"user_email\"
            },
            \"filters\": {
                \"status\": \"active\"
            }
        },
        \"status\": \"active\",
        \"created_by\": \"admin\",
        \"creator_name\": \"系统管理员\"
    }"
    
    api_call "POST" "/data_sync_tasks" "$sync_task_data" "创建数据同步任务"
    
    # 获取同步任务列表
    api_call "GET" "/data_sync_tasks?select=*&limit=10" "" "获取数据同步任务列表"
}

# 测试系统日志
test_system_logs() {
    log_info "=== 测试系统日志管理 ==="
    
    # 创建系统日志
    local log_id=$(generate_uuid)
    local log_data="{
        \"id\": \"$log_id\",
        \"operation_type\": \"create\",
        \"object_type\": \"basic_library\",
        \"object_id\": \"test-library-id\",
        \"operator_id\": \"admin\",
        \"operator_name\": \"系统管理员\",
        \"operator_ip\": \"192.168.1.100\",
        \"operation_content\": {
            \"action\": \"创建数据基础库\",
            \"details\": \"创建了测试用户基础库\",
            \"before_data\": null,
            \"after_data\": {
                \"name_zh\": \"测试用户基础库\",
                \"name_en\": \"test_user_basic_library\",
                \"status\": \"active\"
            },
            \"request_id\": \"req_123456789\",
            \"user_agent\": \"PostgREST Test Script\"
        },
        \"operation_result\": \"success\"
    }"
    
    api_call "POST" "/system_logs" "$log_data" "创建系统日志"
    
    # 获取系统日志列表
    api_call "GET" "/system_logs?select=*&limit=10&order=operation_time.desc" "" "获取系统日志列表"
}

# 清理测试数据
cleanup_test_data() {
    log_info "=== 清理测试数据 ==="
    
    # 删除测试数据（按依赖关系倒序删除）
    api_call "DELETE" "/data_interfaces?name_en=eq.user_info_interface" "" "删除测试数据接口"
    api_call "DELETE" "/thematic_interfaces?name_en=eq.user_analysis_interface" "" "删除测试主题接口"
    api_call "DELETE" "/basic_libraries?name_en=eq.test_user_basic_library" "" "删除测试基础库"
    api_call "DELETE" "/thematic_libraries?code=eq.test_user_thematic" "" "删除测试主题库"
    api_call "DELETE" "/quality_rules?name=eq.用户邮箱完整性检查" "" "删除测试质量规则"
    api_call "DELETE" "/metadata?name=eq.用户表技术元数据" "" "删除测试元数据"
    api_call "DELETE" "/data_masking_rules?name=eq.用户邮箱脱敏规则" "" "删除测试脱敏规则"
    api_call "DELETE" "/api_applications?name=eq.测试应用" "" "删除测试API应用"
    api_call "DELETE" "/data_subscriptions?subscriber_id=eq.test_user_123" "" "删除测试数据订阅"
    api_call "DELETE" "/data_sync_tasks?name=eq.用户数据同步任务" "" "删除测试同步任务"
    
    log_success "测试数据清理完成"
}

# 主函数
main() {
    echo "========================================"
    echo "PostgREST 接口完整测试脚本"
    echo "========================================"
    echo ""
    
    # 检查依赖
    if ! command -v curl &> /dev/null; then
        log_error "curl 命令未找到，请安装 curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warning "jq 命令未找到，JSON输出可能不够美观"
    fi
    
    # 执行测试流程
    check_postgrest
    login
    
    # 执行各模块测试
    test_basic_libraries
    test_data_interfaces
    test_thematic_libraries
    test_thematic_interfaces
    test_quality_rules
    test_metadata
    test_masking_rules
    test_api_applications
    test_data_subscriptions
    test_sync_tasks
    test_system_logs
    
    # 询问是否清理测试数据
    echo ""
    read -p "是否清理测试数据？(y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_test_data
    else
        log_info "保留测试数据，可手动清理"
    fi
    
    echo ""
    log_success "PostgREST 接口测试完成！"
}

# 执行主函数
main "$@"
