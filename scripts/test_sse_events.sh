#!/bin/bash

# SSE事件功能测试脚本
# 测试SSE连接、事件发送和数据库监听功能

set -e

# 配置
BASE_URL="http://localhost:8080"
TEST_USER="admin"
LOG_FILE="sse_test.log"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
}

# 检查服务是否运行
check_service() {
    log "检查服务状态..."
    if curl -s "$BASE_URL/health" > /dev/null; then
        success "服务运行正常"
    else
        error "服务未运行，请先启动datahub-service"
        exit 1
    fi
}

# 测试SSE连接
test_sse_connection() {
    log "测试SSE连接..."
    
    # 启动SSE连接（后台运行）
    timeout 10s curl -s -N "$BASE_URL/sse/$TEST_USER" > sse_output.log 2>&1 &
    SSE_PID=$!
    
    sleep 2
    
    if ps -p $SSE_PID > /dev/null; then
        success "SSE连接建立成功"
        
        # 检查连接响应
        if grep -q "connected" sse_output.log; then
            success "收到连接确认消息"
        else
            warning "未收到连接确认消息"
        fi
    else
        error "SSE连接失败"
        return 1
    fi
    
    # 清理
    kill $SSE_PID 2>/dev/null || true
    wait $SSE_PID 2>/dev/null || true
}

# 测试发送事件给指定用户
test_send_event() {
    log "测试发送事件给指定用户..."
    
    # 启动SSE连接监听
    timeout 15s curl -s -N "$BASE_URL/sse/$TEST_USER" > sse_receive.log 2>&1 &
    SSE_PID=$!
    
    sleep 2
    
    # 发送测试事件
    EVENT_DATA='{
        "user_name": "'$TEST_USER'",
        "event_type": "system_notification",
        "data": {
            "title": "测试通知",
            "message": "这是一个测试消息",
            "priority": "high",
            "timestamp": "'$(date -Iseconds)'"
        }
    }'
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/events/send" \
        -H "Content-Type: application/json" \
        -d "$EVENT_DATA")
    
    if echo "$RESPONSE" | grep -q '"status":0'; then
        success "事件发送成功"
        
        # 等待接收事件
        sleep 3
        
        # 检查是否收到事件
        if grep -q "system_notification" sse_receive.log; then
            success "SSE客户端收到事件"
        else
            warning "SSE客户端未收到事件"
            cat sse_receive.log
        fi
    else
        error "事件发送失败: $RESPONSE"
    fi
    
    # 清理
    kill $SSE_PID 2>/dev/null || true
    wait $SSE_PID 2>/dev/null || true
}

# 测试广播事件
test_broadcast_event() {
    log "测试广播事件..."
    
    # 启动多个SSE连接
    timeout 15s curl -s -N "$BASE_URL/sse/user1" > sse_user1.log 2>&1 &
    SSE_PID1=$!
    
    timeout 15s curl -s -N "$BASE_URL/sse/user2" > sse_user2.log 2>&1 &
    SSE_PID2=$!
    
    sleep 2
    
    # 发送广播事件
    BROADCAST_DATA='{
        "event_type": "system_announcement",
        "data": {
            "title": "系统公告",
            "message": "系统将在30分钟后进行维护",
            "type": "maintenance",
            "timestamp": "'$(date -Iseconds)'"
        }
    }'
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/events/broadcast" \
        -H "Content-Type: application/json" \
        -d "$BROADCAST_DATA")
    
    if echo "$RESPONSE" | grep -q '"status":0'; then
        success "广播事件发送成功"
        
        # 等待接收事件
        sleep 3
        
        # 检查两个用户是否都收到事件
        USER1_RECEIVED=false
        USER2_RECEIVED=false
        
        if grep -q "system_announcement" sse_user1.log; then
            USER1_RECEIVED=true
        fi
        
        if grep -q "system_announcement" sse_user2.log; then
            USER2_RECEIVED=true
        fi
        
        if $USER1_RECEIVED && $USER2_RECEIVED; then
            success "所有用户都收到广播事件"
        elif $USER1_RECEIVED || $USER2_RECEIVED; then
            warning "部分用户收到广播事件"
        else
            warning "没有用户收到广播事件"
        fi
    else
        error "广播事件发送失败: $RESPONSE"
    fi
    
    # 清理
    kill $SSE_PID1 $SSE_PID2 2>/dev/null || true
    wait $SSE_PID1 $SSE_PID2 2>/dev/null || true
}

# 测试数据库事件监听器管理
test_db_listener_management() {
    log "测试数据库事件监听器管理..."
    
    # 创建监听器
    LISTENER_DATA='{
        "name": "基础库变更监听",
        "table_name": "basic_libraries",
        "event_types": ["INSERT", "UPDATE", "DELETE"],
        "condition": {"status": "active"},
        "target_users": ["'$TEST_USER'"]
    }'
    
    CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/events/db-listeners" \
        -H "Content-Type: application/json" \
        -d "$LISTENER_DATA")
    
    if echo "$CREATE_RESPONSE" | grep -q '"status":0'; then
        success "数据库监听器创建成功"
        
        # 提取监听器ID
        LISTENER_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        
        if [ -n "$LISTENER_ID" ]; then
            log "监听器ID: $LISTENER_ID"
            
            # 获取监听器列表
            LIST_RESPONSE=$(curl -s "$BASE_URL/events/db-listeners")
            if echo "$LIST_RESPONSE" | grep -q "$LISTENER_ID"; then
                success "监听器列表获取成功"
            else
                warning "监听器列表中未找到新创建的监听器"
            fi
            
            # 更新监听器
            UPDATE_DATA='{"name": "基础库变更监听(已更新)"}'
            UPDATE_RESPONSE=$(curl -s -X PUT "$BASE_URL/events/db-listeners/$LISTENER_ID" \
                -H "Content-Type: application/json" \
                -d "$UPDATE_DATA")
            
            if echo "$UPDATE_RESPONSE" | grep -q '"status":0'; then
                success "监听器更新成功"
            else
                warning "监听器更新失败"
            fi
            
            # 删除监听器
            DELETE_RESPONSE=$(curl -s -X DELETE "$BASE_URL/events/db-listeners/$LISTENER_ID")
            if echo "$DELETE_RESPONSE" | grep -q '"status":0'; then
                success "监听器删除成功"
            else
                warning "监听器删除失败"
            fi
        else
            warning "无法提取监听器ID"
        fi
    else
        error "数据库监听器创建失败: $CREATE_RESPONSE"
    fi
}

# 测试数据库变更触发事件
test_db_change_events() {
    log "测试数据库变更触发事件..."
    
    # 首先创建一个监听器
    LISTENER_DATA='{
        "name": "测试监听器",
        "table_name": "basic_libraries",
        "event_types": ["INSERT", "UPDATE", "DELETE"],
        "target_users": ["'$TEST_USER'"]
    }'
    
    CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/events/db-listeners" \
        -H "Content-Type: application/json" \
        -d "$LISTENER_DATA")
    
    if echo "$CREATE_RESPONSE" | grep -q '"status":0'; then
        LISTENER_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        
        # 启动SSE连接监听数据库事件
        timeout 20s curl -s -N "$BASE_URL/sse/$TEST_USER" > sse_db_events.log 2>&1 &
        SSE_PID=$!
        
        sleep 2
        
        # 创建一个基础库记录（这应该触发数据库事件）
        LIBRARY_DATA='{
            "name": "测试基础库",
            "description": "用于测试数据库事件的基础库",
            "data_source": "test_source",
            "status": "active"
        }'
        
        # 注意：这里需要调用基础库创建API
        CREATE_LIB_RESPONSE=$(curl -s -X POST "$BASE_URL/basic-libraries" \
            -H "Content-Type: application/json" \
            -d "$LIBRARY_DATA")
        
        if echo "$CREATE_LIB_RESPONSE" | grep -q '"status":0'; then
            success "基础库创建成功，应该触发数据库事件"
            
            # 等待事件传播
            sleep 5
            
            # 检查是否收到数据库变更事件
            if grep -q "data_change" sse_db_events.log; then
                success "收到数据库变更事件"
            else
                warning "未收到数据库变更事件"
                log "SSE输出内容:"
                cat sse_db_events.log
            fi
            
            # 提取创建的库ID并删除
            LIB_ID=$(echo "$CREATE_LIB_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
            if [ -n "$LIB_ID" ]; then
                curl -s -X DELETE "$BASE_URL/basic-libraries/$LIB_ID" > /dev/null
            fi
        else
            warning "基础库创建失败，无法测试数据库事件"
        fi
        
        # 清理监听器
        curl -s -X DELETE "$BASE_URL/events/db-listeners/$LISTENER_ID" > /dev/null
        
        # 清理SSE连接
        kill $SSE_PID 2>/dev/null || true
        wait $SSE_PID 2>/dev/null || true
    else
        error "无法创建测试监听器"
    fi
}

# 清理函数
cleanup() {
    log "清理测试文件..."
    rm -f sse_output.log sse_receive.log sse_user1.log sse_user2.log sse_db_events.log
    success "清理完成"
}

# 主测试流程
main() {
    log "开始SSE事件功能测试"
    log "测试用户: $TEST_USER"
    log "服务地址: $BASE_URL"
    
    # 清理之前的日志
    > "$LOG_FILE"
    
    # 执行测试
    check_service
    test_sse_connection
    test_send_event
    test_broadcast_event
    test_db_listener_management
    test_db_change_events
    
    # 清理
    cleanup
    
    success "所有测试完成！"
    log "详细日志请查看: $LOG_FILE"
}

# 捕获退出信号进行清理
trap cleanup EXIT

# 运行测试
main "$@" 