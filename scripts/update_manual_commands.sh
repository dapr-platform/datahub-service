#!/bin/bash

#
# @module scripts/update_manual_commands
# @description 批量更新手动测试命令文档中的curl命令，添加schema标头
# @architecture 工具脚本
# @documentReference ../scripts/manual_test_commands.md
# @stateFlow 读取文档 -> 替换curl命令 -> 保存文档
# @rules 为所有PostgREST API调用添加Accept-Profile和Content-Profile标头
# @dependencies sed, grep
# @refs ../scripts/manual_test_commands.md
#

set -e

TARGET_FILE="scripts/manual_test_commands.md"
BACKUP_FILE="scripts/manual_test_commands.md.backup"

echo "正在更新手动测试命令文档..."

# 创建备份
cp "$TARGET_FILE" "$BACKUP_FILE"
echo "已创建备份文件: $BACKUP_FILE"

# 更新所有GET请求，添加Accept-Profile
sed -i.tmp 's|curl -H "Authorization: Bearer \$TOKEN"|curl -H "Authorization: Bearer $TOKEN" \\\
  -H "Accept-Profile: public"|g' "$TARGET_FILE"

# 更新所有POST请求，添加Accept-Profile和Content-Profile
sed -i.tmp 's|-H "Content-Type: application/json"|-H "Content-Type: application/json" \\\
  -H "Accept-Profile: public" \\\
  -H "Content-Profile: public"|g' "$TARGET_FILE"

# 更新PATCH请求
sed -i.tmp 's|curl -X PATCH|curl -X PATCH|g' "$TARGET_FILE"

# 更新DELETE请求
sed -i.tmp 's|curl -X DELETE|curl -X DELETE|g' "$TARGET_FILE"

# 清理临时文件
rm -f "$TARGET_FILE.tmp"

echo "手动测试命令文档更新完成！"
echo "如需恢复，请使用: cp $BACKUP_FILE $TARGET_FILE" 