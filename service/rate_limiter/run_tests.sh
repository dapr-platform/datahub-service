#!/bin/bash

# Redis限流器测试运行脚本
# 自动启动Redis容器并运行所有测试

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Redis限流器测试套件${NC}"
echo -e "${GREEN}========================================${NC}"

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}错误: Docker未运行，请先启动Docker${NC}"
    exit 1
fi

# 清理可能存在的旧容器
echo -e "${YELLOW}1. 清理旧的Redis容器...${NC}"
docker rm -f redis-rate-limiter-test > /dev/null 2>&1 || true

# 启动Redis容器
echo -e "${YELLOW}2. 启动Redis测试容器...${NC}"
docker run -d \
    --name redis-rate-limiter-test \
    -p 6379:6379 \
    redis:latest > /dev/null

# 等待Redis启动
echo -e "${YELLOW}3. 等待Redis就绪...${NC}"
sleep 2

# 验证Redis连接
MAX_RETRIES=10
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if docker exec redis-rate-limiter-test redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}   Redis已就绪!${NC}"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT+1))
    echo -e "${YELLOW}   等待Redis启动... ($RETRY_COUNT/$MAX_RETRIES)${NC}"
    sleep 1
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}错误: Redis启动超时${NC}"
    docker rm -f redis-rate-limiter-test
    exit 1
fi

# 设置环境变量（与start.sh保持一致）
export REDIS_HOST=${REDIS_HOST:-localhost}
export REDIS_PORT=${REDIS_PORT:-6379}
export REDIS_PASSWORD=${REDIS_PASSWORD:-things2024}
export REDIS_DB=${REDIS_DB:-0}

# 运行功能测试
echo -e "${YELLOW}4. 运行功能测试...${NC}"
echo -e "${GREEN}========================================${NC}"
if go test -v ./service/rate_limiter/ -run "^Test" -timeout 30s; then
    echo -e "${GREEN}✓ 功能测试通过${NC}"
    FUNC_TEST_PASSED=1
else
    echo -e "${RED}✗ 功能测试失败${NC}"
    FUNC_TEST_PASSED=0
fi

# 运行性能测试
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}5. 运行性能基准测试...${NC}"
echo -e "${GREEN}========================================${NC}"
if go test -bench=. ./service/rate_limiter/ -benchmem -benchtime=3s 2>&1 | tee benchmark.txt; then
    echo -e "${GREEN}✓ 性能测试完成${NC}"
    BENCH_TEST_PASSED=1
else
    echo -e "${RED}✗ 性能测试失败${NC}"
    BENCH_TEST_PASSED=0
fi

# 生成覆盖率报告
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}6. 生成测试覆盖率报告...${NC}"
if go test ./service/rate_limiter/ -coverprofile=coverage.out > /dev/null 2>&1; then
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo -e "${GREEN}   测试覆盖率: ${COVERAGE}${NC}"
    
    # 生成HTML报告
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}   HTML报告已生成: coverage.html${NC}"
    COVERAGE_PASSED=1
else
    echo -e "${RED}✗ 覆盖率报告生成失败${NC}"
    COVERAGE_PASSED=0
fi

# 分析性能结果
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}7. 性能测试结果摘要...${NC}"
if [ -f benchmark.txt ]; then
    echo ""
    echo "单规则检查性能:"
    grep "BenchmarkCheckRateLimit_SingleRule" benchmark.txt | tail -1
    echo ""
    echo "多规则检查性能:"
    grep "BenchmarkCheckRateLimit_MultipleRules" benchmark.txt | tail -1
    echo ""
    echo "并发访问性能:"
    grep "BenchmarkConcurrentAccess" benchmark.txt | tail -1
    echo ""
fi

# 清理Redis容器
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}8. 清理测试环境...${NC}"
docker stop redis-rate-limiter-test > /dev/null
docker rm redis-rate-limiter-test > /dev/null
echo -e "${GREEN}   测试容器已清理${NC}"

# 输出测试总结
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}测试总结${NC}"
echo -e "${GREEN}========================================${NC}"

if [ $FUNC_TEST_PASSED -eq 1 ]; then
    echo -e "${GREEN}✓ 功能测试: 通过${NC}"
else
    echo -e "${RED}✗ 功能测试: 失败${NC}"
fi

if [ $BENCH_TEST_PASSED -eq 1 ]; then
    echo -e "${GREEN}✓ 性能测试: 完成${NC}"
else
    echo -e "${RED}✗ 性能测试: 失败${NC}"
fi

if [ $COVERAGE_PASSED -eq 1 ]; then
    echo -e "${GREEN}✓ 覆盖率报告: ${COVERAGE}${NC}"
else
    echo -e "${RED}✗ 覆盖率报告: 失败${NC}"
fi

echo -e "${GREEN}========================================${NC}"

# 返回总体结果
if [ $FUNC_TEST_PASSED -eq 1 ] && [ $BENCH_TEST_PASSED -eq 1 ]; then
    echo -e "${GREEN}🎉 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}❌ 部分测试失败${NC}"
    exit 1
fi

