# PostgREST 手动测试命令集合

## 配置变量

```bash
# 设置基本配置
export POSTGREST_URL="http://localhost:3000"
export USERNAME="admin"
export PASSWORD="things2024"
```

## 1. 登录获取 Token

```bash
# 登录获取token
TOKEN=$(curl -s -X POST "${POSTGREST_URL}/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}" | \
  jq -r '.token')

echo "Token: $TOKEN"
```

## 2. 数据基础库

### 创建基础库

```bash
curl -X POST "${POSTGREST_URL}/basic_libraries" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: public" \
  -H "Content-Profile: public" \
  -d '{
    "name_zh": "测试用户基础库",
    "name_en": "test_user_basic_library",
    "description": "用于测试的用户基础数据库"
  }' | jq '.'
```

### 查询基础库列表

```bash
curl -H "Authorization: Bearer $TOKEN" \
  -H "Accept-Profile: public" \
  "${POSTGREST_URL}/basic_libraries?select=*&limit=10" | jq '.'
```

### 根据英文名称查询

```bash
curl -H "Authorization: Bearer $TOKEN" \
  -H "Accept-Profile: public" \
  "${POSTGREST_URL}/basic_libraries?name_en=eq.test_user_basic_library&select=*" | jq '.'
```

### 更新基础库

```bash
curl -X PATCH "${POSTGREST_URL}/basic_libraries?name_en=eq.test_user_basic_library" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: public" \
  -H "Content-Profile: public" \
  -d '{"description": "更新后的用户基础数据库描述"}' | jq '.'
```

### 删除基础库

```bash
curl -X DELETE "${POSTGREST_URL}/basic_libraries?name_en=eq.test_user_basic_library" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept-Profile: public"
```

## 3. 数据接口测试

### 获取基础库 ID（用于创建接口）

```bash
LIBRARY_ID=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/basic_libraries?name_en=eq.test_user_basic_library&select=id" | \
  jq -r '.[0].id')

echo "Library ID: $LIBRARY_ID"
```

### 创建数据接口

```bash
curl -X POST "${POSTGREST_URL}/data_interfaces" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"library_id\": \"$LIBRARY_ID\",
    \"name_zh\": \"用户信息接口\",
    \"name_en\": \"user_info_interface\",
    \"type\": \"realtime\",
    \"description\": \"实时用户信息数据接口\"
  }" | jq '.'
```

### 查询数据接口

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/data_interfaces?library_id=eq.$LIBRARY_ID&select=*" | jq '.'
```

### 更新数据接口

```bash
curl -X PATCH "${POSTGREST_URL}/data_interfaces?name_en=eq.user_info_interface" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"description": "更新后的用户信息数据接口"}' | jq '.'
```

## 4. 数据主题库测试

### 创建主题库

```bash
curl -X POST "${POSTGREST_URL}/thematic_libraries" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试用户主题库",
    "code": "test_user_thematic",
    "category": "business",
    "domain": "user",
    "description": "用于测试的用户主题数据库",
    "tags": ["用户", "测试"],
    "source_libraries": ["test_user_basic_library"],
    "tables": {"users": {"fields": ["id", "name", "email"]}},
    "access_level": "internal",
    "update_frequency": "daily"
  }' | jq '.'
```

### 查询主题库列表

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/thematic_libraries?select=*&limit=10" | jq '.'
```

### 根据分类查询

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/thematic_libraries?category=eq.business&select=*" | jq '.'
```

### 更新主题库

```bash
curl -X PATCH "${POSTGREST_URL}/thematic_libraries?code=eq.test_user_thematic" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "更新后的用户主题数据库描述",
    "version": "1.1.0"
  }' | jq '.'
```

## 5. 主题接口测试

### 获取主题库 ID

```bash
THEMATIC_ID=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/thematic_libraries?code=eq.test_user_thematic&select=id" | \
  jq -r '.[0].id')

echo "Thematic Library ID: $THEMATIC_ID"
```

### 创建主题接口

```bash
curl -X POST "${POSTGREST_URL}/thematic_interfaces" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"library_id\": \"$THEMATIC_ID\",
    \"name_zh\": \"用户分析接口\",
    \"name_en\": \"user_analysis_interface\",
    \"type\": \"http\",
    \"config\": {\"endpoint\": \"/api/user/analysis\", \"method\": \"GET\"},
    \"description\": \"用户数据分析接口\"
  }" | jq '.'
```

### 查询主题接口

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/thematic_interfaces?library_id=eq.$THEMATIC_ID&select=*" | jq '.'
```

## 6. 数据质量规则测试

### 创建质量规则

```bash
curl -X POST "${POSTGREST_URL}/quality_rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "用户邮箱完整性检查",
    "type": "completeness",
    "config": {
      "field": "email",
      "required": true,
      "check_format": true
    },
    "related_object_id": "test-object-id",
    "related_object_type": "interface",
    "is_enabled": true
  }' | jq '.'
```

### 查询质量规则

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/quality_rules?select=*&limit=10" | jq '.'
```

### 根据类型查询

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/quality_rules?type=eq.completeness&select=*" | jq '.'
```

## 7. 元数据测试

### 创建元数据

```bash
curl -X POST "${POSTGREST_URL}/metadata" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "technical",
    "name": "用户表技术元数据",
    "content": {
      "table_name": "users",
      "columns": [
        {"name": "id", "type": "uuid", "nullable": false},
        {"name": "name", "type": "varchar(255)", "nullable": false},
        {"name": "email", "type": "varchar(255)", "nullable": true}
      ],
      "indexes": ["id", "email"],
      "constraints": ["PRIMARY KEY (id)"]
    },
    "related_object_type": "basic_library"
  }' | jq '.'
```

### 查询元数据

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/metadata?select=*&limit=10" | jq '.'
```

## 8. 数据脱敏规则测试

### 创建脱敏规则

```bash
curl -X POST "${POSTGREST_URL}/data_masking_rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "用户邮箱脱敏规则",
    "data_source": "users_table",
    "data_table": "users",
    "field_name": "email",
    "field_type": "varchar",
    "masking_type": "mask",
    "masking_config": {
      "pattern": "***@***.com",
      "preserve_domain": false
    },
    "creator_id": "admin",
    "creator_name": "系统管理员"
  }' | jq '.'
```

### 查询脱敏规则

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/data_masking_rules?select=*&limit=10" | jq '.'
```

## 9. API 应用管理测试

### 创建 API 应用

```bash
curl -X POST "${POSTGREST_URL}/api_applications" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试应用",
    "app_key": "test_app_key_123",
    "app_secret_hash": "$2a$10$example_hash",
    "description": "用于测试的API应用",
    "contact_person": "测试人员",
    "contact_email": "test@example.com"
  }' | jq '.'
```

### 查询 API 应用

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/api_applications?select=*&limit=10" | jq '.'
```

## 10. 数据订阅测试

### 创建数据订阅

```bash
curl -X POST "${POSTGREST_URL}/data_subscriptions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriber_id": "test_user_123",
    "subscriber_type": "user",
    "resource_id": "test_resource_456",
    "resource_type": "thematic_interface",
    "notification_method": "webhook",
    "notification_config": {
      "url": "https://example.com/webhook",
      "headers": {"Authorization": "Bearer token"}
    },
    "filter_condition": {
      "status": "active"
    }
  }' | jq '.'
```

### 查询数据订阅

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/data_subscriptions?select=*&limit=10" | jq '.'
```

## 11. 系统日志测试

### 创建系统日志

```bash
curl -X POST "${POSTGREST_URL}/system_logs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "operation_type": "create",
    "object_type": "basic_library",
    "object_id": "test-library-id",
    "operator_id": "admin",
    "operator_name": "系统管理员",
    "operator_ip": "192.168.1.100",
    "operation_content": {
      "action": "创建数据基础库",
      "details": "创建了测试用户基础库"
    },
    "operation_result": "success"
  }' | jq '.'
```

### 查询系统日志

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "${POSTGREST_URL}/system_logs?select=*&limit=10&order=operation_time.desc" | jq '.'
```

## 12. 清理测试数据

```bash
# 删除测试数据（按依赖关系倒序删除）
curl -X DELETE "${POSTGREST_URL}/data_interfaces?name_en=eq.user_info_interface" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/thematic_interfaces?name_en=eq.user_analysis_interface" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/basic_libraries?name_en=eq.test_user_basic_library" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/thematic_libraries?code=eq.test_user_thematic" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/quality_rules?name=eq.用户邮箱完整性检查" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/metadata?name=eq.用户表技术元数据" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/data_masking_rules?name=eq.用户邮箱脱敏规则" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/api_applications?name=eq.测试应用" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "${POSTGREST_URL}/data_subscriptions?subscriber_id=eq.test_user_123" \
  -H "Authorization: Bearer $TOKEN"
```

## 使用说明

1. **设置环境变量**：首先运行配置变量部分的命令
2. **获取 Token**：运行登录命令获取访问令牌
3. **执行测试**：按需复制粘贴相应的测试命令
4. **查看结果**：所有命令都使用 `jq` 格式化 JSON 输出
5. **清理数据**：测试完成后运行清理命令

## 注意事项

- 确保 PostgREST 服务在 localhost:3000 运行
- 确保已安装 `curl` 和 `jq` 工具
- 确保数据库中已创建 admin 用户
- 测试命令会创建真实数据，请在测试环境中运行
