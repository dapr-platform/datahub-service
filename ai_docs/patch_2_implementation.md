# Patch 2 实施说明：权限管理改为使用 PostgREST RBAC

## 修改概述

根据 `ai_docs/patch_2.md` 的要求，将本项目涉及到的权限相关内容改为使用 PostgREST RBAC，移除了原有的自定义权限管理系统。

## 具体修改内容

### 1. 删除的文件

- `service/models/access_control.go` - 访问控制模型（User, Role, Permission, AccessToken 等）
- `api/controllers/access_control_controller.go` - 访问控制控制器
- `service/access_control_service.go` - 访问控制服务

### 2. 修改的文件

#### `service/database/migrate.go`

- 移除了访问控制相关表的迁移代码
- 移除了默认权限和角色的初始化代码
- 添加了说明注释指向 PostgREST RBAC

#### `api/routes.go`

- 移除了所有访问控制相关的路由
- 移除了用户认证、用户管理、角色管理、权限管理、访问令牌管理的路由
- 添加了说明注释指向 `ai_docs/postgrest_rbac_guide.md`

#### `service/models/sharing.go`

- 移除了对 `User` 模型的外键关联
- 将用户关联改为字符串字段：
  - `Requester *User` → `RequesterName string`
  - `Approver *User` → `ApproverName *string`
  - `Creator *User` → `CreatorName string`
  - `User *User` → `UserName *string`

#### `service/models/governance.go`

- 移除了对 `User` 模型的外键关联
- 将用户关联改为字符串字段：
  - `Creator *User` → `CreatorName string`
  - `Operator *User` → `OperatorName *string`
  - `Generator *User` → `GeneratorName string`

### 3. 重新生成的文件

- `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` - 重新生成 Swagger 文档，移除了权限相关模型的引用

## PostgREST RBAC 集成

### 权限管理方式

现在权限管理完全通过 PostgREST 提供，具体实现方式请参考：

- `ai_docs/postgrest_rbac_guide.md` - 完整的 PostgREST RBAC 使用指南

### 主要特性

1. **用户管理**：通过 PostgREST 的 `postgrest.users` 表管理
2. **角色管理**：通过 PostgREST 的 `postgrest.roles` 表管理
3. **权限管理**：通过 PostgREST 的 `postgrest.permissions` 表管理
4. **JWT Token**：包含用户角色和权限信息
5. **Schema 权限**：支持多 schema 的权限控制

### API 访问方式

所有对 PostgREST 的 API 请求都必须包含以下 HTTP headers：

```
Accept-Profile: postgrest
Content-Profile: postgrest
```

### 示例用法

```bash
# 用户登录
curl -X POST "http://localhost:3000/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{"username": "admin", "password": "password"}'

# 创建用户
curl -X POST "http://localhost:3000/rpc/add_user" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "user_name": "john_doe",
    "user_password": "secure_password123",
    "target_schemas": "public,postgrest",
    "email": "john@example.com",
    "full_name": "John Doe",
    "default_role": "user"
  }'
```

## 影响和注意事项

### 数据库变更

- 原有的权限相关表（users, roles, permissions 等）不再由此服务管理
- 需要确保 PostgREST 的权限管理系统已正确部署和配置

### 应用程序变更

- 所有权限验证逻辑需要改为调用 PostgREST API
- 用户认证需要通过 PostgREST 的 JWT token 机制
- 前端应用需要更新权限管理相关的 API 调用

### 部署要求

- 需要部署 PostgREST 服务
- 需要配置 PostgreSQL 数据库的 `postgrest` schema
- 需要初始化 PostgREST RBAC 的基础数据

## 验证结果

- ✅ 项目编译成功
- ✅ Swagger 文档重新生成成功
- ✅ 所有权限相关的代码引用已清理
- ✅ 数据库迁移代码已更新
- ✅ 路由配置已更新

## 后续工作

1. 部署和配置 PostgREST 服务
2. 初始化 PostgREST RBAC 的基础数据
3. 更新前端应用的权限管理 API 调用
4. 测试新的权限管理系统
5. 更新相关文档和部署指南
