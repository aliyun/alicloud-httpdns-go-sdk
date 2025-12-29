#!/bin/bash

# 真实环境端到端测试脚本
# 使用规则文档中提供的测试账号信息

set -e

echo "🚀 开始真实环境端到端测试"

# 设置测试环境变量（请替换为您的真实账号信息）
export HTTPDNS_ACCOUNT_ID="${HTTPDNS_ACCOUNT_ID:-your-account-id}"
export HTTPDNS_SECRET_KEY="${HTTPDNS_SECRET_KEY:-your-secret-key}"

echo "📋 测试配置:"
echo "  账号ID: $HTTPDNS_ACCOUNT_ID"
echo "  密钥: ****"
echo "  可解析域名: www.aliyun.com, www.alibaba.com"
echo "  不可解析域名: www.baidu.com, www.qq.com"
echo ""

echo "🧪 运行真实环境测试..."
go test -v ./test -run "TestRealEndToEnd" -timeout 60s

echo ""
echo "✅ 真实环境端到端测试完成"