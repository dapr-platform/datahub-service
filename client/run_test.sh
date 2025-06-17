#!/bin/bash

# PostgreSQL Meta客户端测试运行脚本

echo "========================================"
echo "PostgreSQL Meta客户端测试"
echo "========================================"
echo "测试服务器: http://localhost:3001"
echo "测试时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "========================================"

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: Go环境未安装"
    exit 1
fi

# 检查当前目录
if [ ! -f "pgmeta_test.go" ]; then
    echo "错误: 请在client目录下运行此脚本"
    exit 1
fi

# 检查服务器连接
echo "检查PostgreSQL Meta服务连接..."
if curl -s http://localhost:3001/schemas > /dev/null 2>&1; then
    echo "✅ 服务器连接正常"
else
    echo "❌ 无法连接到PostgreSQL Meta服务 (http://localhost:3001)"
    echo "请确保服务已启动"
    exit 1
fi

echo ""
echo "开始运行测试..."
echo "========================================"

# 运行全面测试套件
echo "📋 运行完整测试套件..."
go test -v -run TestPgMetaClient_FullSuite -timeout 5m

echo ""
echo "========================================"

# 单独运行数据类型兼容性测试
echo "🔍 重点测试数据类型兼容性..."
go test -v -run TestPgMetaClient_DataTypeCompatibility -timeout 2m

echo ""
echo "========================================"

# 运行简单的连接测试
echo "🚀 快速连接测试..."
go test -v -run TestPgMetaClient_Schemas/ListSchemas -timeout 30s

echo ""
echo "========================================"
echo "测试完成"
echo "========================================" 