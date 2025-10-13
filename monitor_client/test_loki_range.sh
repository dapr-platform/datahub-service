#!/bin/bash

# 测试 Loki 区间查询
LOKI_URL="http://mh1:3100"
QUERY='{app=~".*"}'
START="1760335392000000000"  # 纳秒时间戳
END="1760338992000000000"    # 纳秒时间戳
LIMIT=10

echo "测试 Loki 区间查询"
echo "URL: ${LOKI_URL}/loki/api/v1/query_range"
echo "Query: ${QUERY}"
echo "Start: ${START}"
echo "End: ${END}"
echo "Limit: ${LIMIT}"
echo ""

# 使用 GET 请求
curl -v -G "${LOKI_URL}/loki/api/v1/query_range" \
  --data-urlencode "query=${QUERY}" \
  --data-urlencode "start=${START}" \
  --data-urlencode "end=${END}" \
  --data-urlencode "limit=${LIMIT}" \
  2>&1 | head -100

echo ""
echo "---"
echo ""

# 测试即时查询
echo "测试 Loki 即时查询"
curl -v -G "${LOKI_URL}/loki/api/v1/query" \
  --data-urlencode "query=${QUERY}" \
  --data-urlencode "limit=${LIMIT}" \
  2>&1 | head -100

