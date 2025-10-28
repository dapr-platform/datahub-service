#!/bin/bash

# 数据质量检测任务集成测试脚本

set -e

echo "=== 数据质量检测任务集成测试 ==="
echo ""

# 设置环境变量
export DB_HOST=${DB_HOST:-localhost}
export DB_PORT=${DB_PORT:-5432}
export DB_NAME=${DB_NAME:-postgres}
export DB_USER=${DB_USER:-supabase_admin}
export DB_PASSWORD=${DB_PASSWORD:-things2024}
export DB_SSLMODE=${DB_SSLMODE:-disable}
export DB_SCHEMA=${DB_SCHEMA:-public}

export REDIS_HOST=${REDIS_HOST:-localhost}
export REDIS_PORT=${REDIS_PORT:-6379}
export REDIS_PASSWORD=${REDIS_PASSWORD:-things2024}
export REDIS_DB=${REDIS_DB:-0}

export SCHEDULER_ENABLED=${SCHEDULER_ENABLED:-true}
export LISTEN_PORT=${LISTEN_PORT:-8080}
export BASE_CONTEXT=${BASE_CONTEXT:-/swagger/datahub-service}

cd ..

# 检查服务是否已在运行
if curl -s http://localhost:8080/swagger/datahub-service/data-quality/templates/quality-rules > /dev/null 2>&1; then
    echo "✓ 服务已在运行"
    SERVICE_RUNNING=true
else
    echo "× 服务未运行，启动服务..."
    SERVICE_RUNNING=false
    
    # 编译服务
    echo "编译服务..."
    go build -o datahub-service
    
    # 启动服务（后台）
    echo "启动服务..."
    ./datahub-service > /tmp/datahub-service.log 2>&1 &
    SERVICE_PID=$!
    echo "服务PID: $SERVICE_PID"
    
    # 等待服务启动
    echo "等待服务启动..."
    for i in {1..30}; do
        if curl -s http://localhost:8080/swagger/datahub-service/data-quality/templates/quality-rules > /dev/null 2>&1; then
            echo "✓ 服务启动成功"
            break
        fi
        if [ $i -eq 30 ]; then
            echo "✗ 服务启动超时"
            if [ -n "$SERVICE_PID" ]; then
                kill $SERVICE_PID 2>/dev/null || true
            fi
            exit 1
        fi
        sleep 1
    done
fi

# 运行集成测试
echo ""
echo "=== 运行集成测试 ==="
echo ""

go test -v ./service/governance/tests -run TestQualityTaskIntegration -timeout 60s

TEST_RESULT=$?

# 如果是我们启动的服务，测试结束后关闭
if [ "$SERVICE_RUNNING" = false ] && [ -n "$SERVICE_PID" ]; then
    echo ""
    echo "关闭测试服务..."
    kill $SERVICE_PID 2>/dev/null || true
    wait $SERVICE_PID 2>/dev/null || true
fi

echo ""
if [ $TEST_RESULT -eq 0 ]; then
    echo "=== ✓ 集成测试通过 ==="
else
    echo "=== ✗ 集成测试失败 ==="
fi

cd scripts
exit $TEST_RESULT

