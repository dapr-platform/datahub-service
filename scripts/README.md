# PostgREST 接口测试脚本使用指南

## 概述

本目录包含用于测试 PostgREST 接口的脚本，支持对数据底座服务所有模型表的增删改查操作。

## 脚本说明

### 1. `test_postgrest.sh` - 完整测试脚本

**功能**：

- 完整的 RBAC 登录流程
- 所有模型表的 CRUD 操作测试
- 数据关联关系验证
- 自动清理测试数据

**测试覆盖的表**：

- `basic_libraries` - 数据基础库
- `data_interfaces` - 数据接口
- `thematic_libraries` - 数据主题库
- `thematic_interfaces` - 主题接口
- `quality_rules` - 数据质量规则
- `metadata` - 元数据
- `data_masking_rules` - 数据脱敏规则
- `api_applications` - API 应用
- `data_subscriptions` - 数据订阅
- `data_sync_tasks` - 数据同步任务
- `system_logs` - 系统日志

### 2. `quick_test.sh` - 快速测试脚本

**功能**：

- 快速验证 PostgREST 服务状态
- 基本的 RBAC 登录测试
- 核心表的基础 CRUD 操作
- 自动清理测试数据

### 3. `test_login.sh` - 双 Token 登录功能测试脚本

**功能**：

- 专门测试双 Token RBAC 登录功能
- 详细显示登录响应信息（access_token 和 refresh_token）
- 验证 access token 有效性
- 测试 refresh token 功能
- 测试 token 撤销功能
- 完整的双 Token 机制验证流程
- 支持有无 jq 工具的环境

### 4. `cleanup_test_data.sh` - 清理脚本

**功能**：

- 清理所有测试数据
- 按依赖关系正确删除测试数据

## 使用前准备

### 1. 环境要求

```bash
# 安装必要工具
sudo apt-get install curl jq  # Ubuntu/Debian
# 或
brew install curl jq          # macOS
```

### 2. 服务启动

确保以下服务正在运行：

```bash
# PostgreSQL 数据库
sudo systemctl start postgresql

# PostgREST 服务 (默认端口 3000)
postgrest config.conf
```

### 3. 数据库准备

确保数据库中已经：

- 创建了所有必要的表结构
- 初始化了 RBAC 权限系统
- 创建了 admin 用户（用户名：admin，密码：things2024）

### 4. PostgREST Schema 配置

所有 API 调用都需要指定正确的数据库 schema：

- **Accept-Profile: public** - 指定读取数据的 schema
- **Content-Profile: public** - 指定写入数据的 schema（POST/PATCH/PUT 请求）

这些标头确保 PostgREST 在正确的数据库 schema 中操作。

### 5. 错误处理机制

测试脚本现在包含完整的 PostgREST 错误检测和处理：

- **自动错误检测**：检查响应中是否包含 `code` 字段
- **详细错误信息**：显示错误代码、消息、详情和提示
- **友好的错误提示**：根据 [PostgREST 错误响应格式](https://postgrest.postgresql.ac.cn/en/v12/references/api/schemas.html) 解析错误

**PostgREST 错误响应示例**：

```json
{
  "code": "23502",
  "details": "Failing row contains (null, ...)",
  "hint": null,
  "message": "null value in column \"id\" violates not-null constraint"
}
```

## 使用方法

### 双 Token 登录功能测试

```bash
# 专门测试双Token登录功能，包含完整的token管理流程
./scripts/test_login.sh
```

**测试内容包括**：

- 双 Token 登录获取 access_token 和 refresh_token
- Access Token 有效性验证
- Refresh Token 刷新机制测试
- Token 撤销功能验证
- 撤销后 Token 无效性验证

### 快速测试

```bash
# 运行快速测试，验证基本功能
./scripts/quick_test.sh
```

### 完整测试

```bash
# 运行完整测试，覆盖所有功能
./scripts/test_postgrest.sh
```

### 清理测试数据

```bash
# 清理所有测试数据
./scripts/cleanup_test_data.sh
```

### 自定义配置

可以通过环境变量自定义配置：

```bash
# 自定义 PostgREST 地址
export POSTGREST_URL="http://your-server:3000"

# 自定义登录凭据
export USERNAME="your_admin"
export PASSWORD="your_password"

# 运行测试
./scripts/test_postgrest.sh
```

## 测试流程

### 1. 服务检查

- 验证 PostgREST 服务是否可访问
- 检查必要的依赖工具

### 2. 双 Token RBAC 登录

```bash
# 双Token登录请求示例 - 返回access_token和refresh_token
curl -X POST "http://localhost:3000/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{"username": "admin", "password": "things2024"}'

# 响应示例:
# {
#   "success": true,
#   "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
#   "refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
#   "username": "admin",
#   "roles": ["admin"],
#   "permissions": [...]
# }

# 刷新access token
curl -X POST "http://localhost:3000/rpc/refresh_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{"refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc..."}'

# 撤销refresh token
curl -X POST "http://localhost:3000/rpc/revoke_refresh_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{"refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc..."}'
```

### 3. 数据操作测试

#### 创建数据基础库

```bash
curl -X POST "http://localhost:3000/basic_libraries" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: public" \
  -H "Content-Profile: public" \
  -d '{
    "name_zh": "测试用户基础库",
    "name_en": "test_user_basic_library",
    "description": "用于测试的用户基础数据库"
  }'
```

#### 查询数据基础库

```bash
curl -H "Authorization: Bearer $TOKEN" \
  -H "Accept-Profile: public" \
  "http://localhost:3000/basic_libraries?select=*&limit=10"
```

#### 更新数据基础库

```bash
curl -X PATCH "http://localhost:3000/basic_libraries?name_en=eq.test_user_basic_library" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: public" \
  -H "Content-Profile: public" \
  -d '{"description": "更新后的描述"}'
```

#### 删除数据基础库

```bash
curl -X DELETE "http://localhost:3000/basic_libraries?name_en=eq.test_user_basic_library" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept-Profile: public"
```

## 常见问题

### 1. 连接失败

**问题**：`PostgREST服务不可用`

**解决方案**：

- 检查 PostgREST 服务是否启动：`ps aux | grep postgrest`
- 检查端口是否被占用：`netstat -tlnp | grep 3000`
- 检查防火墙设置

### 2. 登录失败

**问题**：`登录失败: {"code":"42883","details":null,"hint":null,"message":"function postgrest.get_token(...) does not exist"}`

**解决方案**：

- 确保已正确执行 RBAC 初始化脚本
- 检查 PostgREST 配置中的 `db-schemas` 设置
- 验证数据库中是否存在 `postgrest` schema

### 3. 数据创建失败

**问题**：`{"code":"23502","details":"Failing row contains (null, ...)","message":"null value in column \"id\" violates not-null constraint"}`

**解决方案**：

- 检查数据模型是否正确设置了主键自动生成
- 确保 GORM 的 `BeforeCreate` 钩子正常工作
- 验证数据库表结构是否与模型定义一致

### 4. 权限错误

**问题**：`permission denied for schema postgrest`

**解决方案**：

- 检查 `authenticator` 角色的权限设置
- 重新执行权限授予脚本：

```sql
GRANT USAGE ON SCHEMA postgrest TO authenticator;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA postgrest TO authenticator;
```

### 5. 表不存在

**问题**：`relation "public.basic_libraries" does not exist`

**解决方案**：

- 确保已运行数据库迁移
- 检查表是否在正确的 schema 中
- 验证 PostgREST 配置的 schema 设置

## 输出示例

### 成功输出

```
=== PostgREST 快速功能测试 ===

1. 检查PostgREST服务...
✓ PostgREST服务正常

2. 测试RBAC登录...
✓ 登录成功

3. 测试基础库CRUD操作...
创建测试基础库...
✓ 创建成功
查询测试基础库...
✓ 查询成功
更新测试基础库...
✓ 更新成功

4. 测试主题库操作...
✓ 主题库创建成功

5. 清理测试数据...
✓ 清理完成

=== 快速测试完成！所有基本功能正常 ===
```

### 错误输出

```
=== PostgREST 快速功能测试 ===

1. 检查PostgREST服务...
✗ PostgREST服务不可用
```

## 扩展使用

### 1. 添加新的测试用例

在脚本中添加新的测试函数：

```bash
test_new_feature() {
    log_info "=== 测试新功能 ==="

    local test_data='{
        "field1": "value1",
        "field2": "value2"
    }'

    api_call "POST" "/new_endpoint" "$test_data" "创建新资源"
}
```

### 2. 批量数据测试

创建批量测试数据：

```bash
for i in {1..10}; do
    api_call "POST" "/basic_libraries" "{
        \"name_zh\": \"测试库$i\",
        \"name_en\": \"test_library_$i\",
        \"description\": \"批量测试库$i\"
    }" "创建测试库$i"
done
```

### 3. 性能测试

添加响应时间测量：

```bash
start_time=$(date +%s%N)
api_call "GET" "/basic_libraries?limit=1000" "" "性能测试"
end_time=$(date +%s%N)
duration=$((($end_time - $start_time) / 1000000))
echo "响应时间: ${duration}ms"
```

## 注意事项

1. **数据安全**：测试脚本会创建和删除数据，请在测试环境中运行
2. **权限要求**：需要使用具有管理员权限的账户
3. **网络环境**：确保网络连接稳定，避免超时
4. **并发限制**：避免同时运行多个测试脚本
5. **日志记录**：测试过程中的所有操作都会被记录到系统日志中
