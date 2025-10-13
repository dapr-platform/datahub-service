#!/bin/bash

# 监控客户端测试脚本
# 用于测试 VictoriaMetrics 和 Loki 客户端的基本功能

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始测试监控客户端...${NC}\n"

# 配置
VICTORIA_METRICS_URL=${VICTORIA_METRICS_URL:-http://mh1:38428}
LOKI_URL=${LOKI_URL:-http://mh1:3100}

echo "VictoriaMetrics URL: $VICTORIA_METRICS_URL"
echo "Loki URL: $LOKI_URL"
echo ""

# 测试 VictoriaMetrics 连接
echo -e "${YELLOW}1. 测试 VictoriaMetrics 连接...${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" "$VICTORIA_METRICS_URL/api/v1/query?query=up")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ VictoriaMetrics 连接成功${NC}"
else
    echo -e "${RED}✗ VictoriaMetrics 连接失败 (HTTP $response)${NC}"
fi
echo ""

# 测试 VictoriaMetrics 即时查询
echo -e "${YELLOW}2. 测试 VictoriaMetrics 即时查询...${NC}"
result=$(curl -s "$VICTORIA_METRICS_URL/api/v1/query?query=up")
if echo "$result" | grep -q '"status":"success"'; then
    echo -e "${GREEN}✓ 即时查询成功${NC}"
    echo "查询结果示例:"
    echo "$result" | jq -r '.data.result[0] // "无结果"' 2>/dev/null || echo "$result"
else
    echo -e "${RED}✗ 即时查询失败${NC}"
    echo "$result"
fi
echo ""

# 测试 VictoriaMetrics 区间查询
echo -e "${YELLOW}3. 测试 VictoriaMetrics 区间查询...${NC}"
end_time=$(date +%s)
start_time=$((end_time - 3600))
result=$(curl -s -X POST "$VICTORIA_METRICS_URL/api/v1/query_range" \
    -d "query=up" \
    -d "start=$start_time" \
    -d "end=$end_time" \
    -d "step=60")
if echo "$result" | grep -q '"status":"success"'; then
    echo -e "${GREEN}✓ 区间查询成功${NC}"
    data_points=$(echo "$result" | jq -r '.data.result[0].values | length' 2>/dev/null || echo "0")
    echo "数据点数量: $data_points"
else
    echo -e "${RED}✗ 区间查询失败${NC}"
    echo "$result"
fi
echo ""

# 测试 Loki 连接
echo -e "${YELLOW}4. 测试 Loki 连接...${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" "$LOKI_URL/loki/api/v1/query?query={job=\"test\"}")
if [ "$response" = "200" ] || [ "$response" = "400" ]; then
    echo -e "${GREEN}✓ Loki 连接成功${NC}"
else
    echo -e "${RED}✗ Loki 连接失败 (HTTP $response)${NC}"
fi
echo ""

# 测试 Loki 标签查询
echo -e "${YELLOW}5. 测试 Loki 标签查询...${NC}"
result=$(curl -s "$LOKI_URL/loki/api/v1/label/job/values")
if echo "$result" | grep -q '"status":"success"'; then
    echo -e "${GREEN}✓ 标签查询成功${NC}"
    labels=$(echo "$result" | jq -r '.data | join(", ")' 2>/dev/null || echo "解析失败")
    echo "可用的 job 标签: $labels"
else
    echo -e "${RED}✗ 标签查询失败${NC}"
    echo "$result"
fi
echo ""

# 运行 Go 单元测试
echo -e "${YELLOW}6. 运行 Go 单元测试...${NC}"
if command -v go &> /dev/null; then
    cd "$(dirname "$0")"
    echo "运行 victoria_metrics_client 测试..."
    go test -v -run TestQuery victoria_metrics_client_test.go victoria_metrics_client.go entity.go 2>&1 | grep -E "PASS|FAIL|RUN|ok|SKIP" || true
    echo ""
    echo "运行 loki_client 测试..."
    go test -v -run TestLokiQuery loki_client_test.go loki_client.go entity.go 2>&1 | grep -E "PASS|FAIL|RUN|ok|SKIP" || true
else
    echo -e "${YELLOW}⚠ Go 未安装，跳过单元测试${NC}"
fi
echo ""

# 测试总结
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}测试完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "下一步:"
echo "1. 如果连接失败，请检查环境变量:"
echo "   export VICTORIA_METRICS_URL=http://your-host:port"
echo "   export LOKI_URL=http://your-host:port"
echo ""
echo "2. 运行完整的单元测试:"
echo "   go test -v ./..."
echo ""
echo "3. 查看 API 文档:"
echo "   cat README.md"
echo ""

