#!/bin/bash

# 快速测试脚本 - 使用现有Redis
# 适用于本地开发和快速验证

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Redis限流器快速测试${NC}"
echo -e "${GREEN}========================================${NC}"

# 设置环境变量（与start.sh保持一致）
export REDIS_HOST=${REDIS_HOST:-localhost}
export REDIS_PORT=${REDIS_PORT:-6379}
export REDIS_PASSWORD=${REDIS_PASSWORD:-things2024}
export REDIS_DB=${REDIS_DB:-0}

echo -e "${YELLOW}Redis配置:${NC}"
echo -e "  Host: $REDIS_HOST"
echo -e "  Port: $REDIS_PORT"
echo -e "  DB: $REDIS_DB"
echo ""

# 运行测试
cd /Users/liu/Work/go-project/dapr-platform/datahub-service

echo -e "${YELLOW}1. 运行功能测试...${NC}"
echo -e "${GREEN}========================================${NC}"
if go test -v ./service/rate_limiter/ -run "^Test" -timeout 60s; then
    echo -e "${GREEN}✓ 功能测试通过${NC}"
else
    echo -e "${RED}✗ 功能测试失败${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}2. 运行性能测试...${NC}"
echo -e "${GREEN}========================================${NC}"
go test -bench=. ./service/rate_limiter/ -benchmem -benchtime=3s

echo ""
echo -e "${YELLOW}3. 生成覆盖率报告...${NC}"
go test ./service/rate_limiter/ -coverprofile=service/rate_limiter/coverage.out > /dev/null 2>&1
COVERAGE=$(go tool cover -func=service/rate_limiter/coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}  测试覆盖率: ${COVERAGE}${NC}"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}🎉 所有测试完成！${NC}"
echo -e "${GREEN}========================================${NC}"

