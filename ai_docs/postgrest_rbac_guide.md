# PostgREST 角色权限管理系统 API 指南

## 系统概述

PostgREST RBAC 系统提供完整的用户、角色和权限管理功能，基于 JWT 认证机制，支持细粒度的权限控制。

## 数据模型

### 用户表 (users)

- `username`: 用户名（主键）
- `password_hash`: 密码哈希
- `email`: 邮箱
- `full_name`: 全名
- `display_name`: 中文显示名称
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `is_active`: 是否激活

### 角色表 (roles)

- `role_name`: 角色名（主键）
- `description`: 描述
- `display_name`: 中文显示名称
- `is_system_role`: 是否系统角色
- `created_at`: 创建时间
- `updated_at`: 更新时间

### 权限表 (permissions)

- `permission_name`: 权限名（主键）
- `description`: 描述
- `display_name`: 中文显示名称
- `resource_type`: 资源类型
- `action_type`: 操作类型
- `created_at`: 创建时间
- `updated_at`: 更新时间

## API 接口

### 认证相关

#### 获取访问令牌

```http
POST /rpc/get_token
Content-Type: application/json

{
  "username": "admin",
  "password": "password123"
}
```

**响应：**

```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "roles": ["admin"],
  "permissions": ["system.admin", "data.read", "data.write"],
  "username": "admin",
  "expires_at": "2024-01-01T12:00:00Z"
}
```

#### 刷新访问令牌

```http
POST /rpc/refresh_token
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应：**

```json
{
  "success": true,
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-01T12:15:00Z"
}
```

#### 验证令牌

```http
POST /rpc/verify_token
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应：**

```json
{
  "success": true,
  "valid": true,
  "username": "admin",
  "roles": ["admin"],
  "permissions": ["system.admin"],
  "expires_at": "2024-01-01T12:00:00Z"
}
```

#### 撤销刷新令牌

```http
POST /rpc/revoke_refresh_token
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应：**

```json
{
  "success": true,
  "message": "Refresh token已成功撤销",
  "username": "admin",
  "revoked_at": "2024-01-01T12:00:00Z"
}
```

### 用户管理

#### 创建用户

```http
POST /rpc/add_user
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "user_name": "newuser",
  "user_password": "password123",
  "target_schemas": "public,schema1",
  "email": "user@example.com",
  "full_name": "New User",
  "display_name": "新用户",
  "default_role": "user"
}
```

**响应：**

```json
{
  "success": true,
  "message": "用户创建成功",
  "username": "newuser",
  "target_schemas": "public,schema1",
  "default_role": "user"
}
```

#### 更新用户 Schema 权限

```http
POST /rpc/update_user_schemas
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "user_name": "username",
  "new_target_schemas": "public,schema1,schema2"
}
```

**响应：**

```json
{
  "success": true,
  "message": "用户schema权限更新成功",
  "username": "username",
  "new_schemas": "public,schema1,schema2"
}
```

#### 修改密码

```http
POST /rpc/change_password
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_name": "username",
  "new_password": "newpassword123"
}
```

**响应：** HTTP 204 No Content（成功）或错误信息

#### 删除用户

```http
POST /rpc/delete_user
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "user_name": "username",
  "force_delete": false
}
```

**响应：**

```json
{
  "success": true,
  "message": "用户已被禁用（软删除）",
  "deleted_from_table": false,
  "deleted_from_db": false,
  "disabled": true
}
```

#### 重新激活用户

```http
POST /rpc/reactivate_user
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "user_name": "username"
}
```

**响应：**

```json
{
  "success": true,
  "message": "用户已重新激活",
  "username": "username"
}
```

#### 列出用户

```http
POST /rpc/list_users
Content-Type: application/json
Authorization: Bearer <admin_token>

{}
```

**响应：**

```json
[
  {
    "username": "admin",
    "email": "admin@example.com",
    "full_name": "Administrator",
    "display_name": "管理员",
    "is_active": true,
    "created_at": "2024-01-01T10:00:00Z"
  }
]
```

#### 按角色获取用户

```http
POST /rpc/get_users_by_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "role_name": "admin"
}
```

**响应：**

```json
[
  {
    "username": "admin",
    "email": "admin@example.com",
    "full_name": "Administrator",
    "assigned_at": "2024-01-01T10:00:00Z"
  }
]
```

### 角色管理

#### 创建角色

```http
POST /rpc/create_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "role_name": "new_role",
  "description": "新角色描述",
  "display_name": "新角色",
  "is_system_role": false
}
```

**响应：**

```json
{
  "success": true,
  "message": "角色创建成功",
  "role_name": "new_role"
}
```

#### 更新角色

```http
POST /rpc/update_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "role_name": "existing_role",
  "new_description": "更新后的描述",
  "new_display_name": "更新后的角色名"
}
```

**响应：**

```json
{
  "success": true,
  "message": "角色更新成功",
  "role_name": "existing_role"
}
```

#### 删除角色

```http
POST /rpc/delete_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "role_name": "role_to_delete",
  "force_delete": false
}
```

**响应：**

```json
{
  "success": true,
  "message": "角色删除成功",
  "role_name": "role_to_delete"
}
```

#### 列出角色

```http
POST /rpc/list_roles
Content-Type: application/json
Authorization: Bearer <admin_token>

{}
```

**响应：**

```json
[
  {
    "role_name": "admin",
    "description": "管理员角色",
    "display_name": "管理员",
    "is_system_role": true,
    "created_at": "2024-01-01T10:00:00Z"
  }
]
```

#### 分配角色

```http
POST /rpc/assign_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "p_user_name": "username",
  "p_role_name": "role_name",
  "p_assigned_by": "admin",
  "p_expires_at": "2024-12-31T23:59:59Z"
}
```

**响应：**

```json
{
  "success": true,
  "message": "角色分配成功",
  "username": "username",
  "role_name": "role_name"
}
```

#### 撤销角色

```http
POST /rpc/revoke_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "user_name": "username",
  "role_name": "role_name"
}
```

**响应：**

```json
{
  "success": true,
  "message": "角色撤销成功",
  "username": "username",
  "role_name": "role_name"
}
```

### 权限管理

#### 创建权限

```http
POST /rpc/create_permission
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "permission_name": "resource.action",
  "description": "权限描述",
  "display_name": "权限显示名",
  "resource_type": "resource",
  "action_type": "action"
}
```

**响应：**

```json
{
  "success": true,
  "message": "权限创建成功",
  "permission_name": "resource.action"
}
```

#### 更新权限

```http
POST /rpc/update_permission
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "permission_name": "existing.permission",
  "new_description": "更新后的描述",
  "new_display_name": "更新后的显示名"
}
```

**响应：**

```json
{
  "success": true,
  "message": "权限更新成功",
  "permission_name": "existing.permission"
}
```

#### 删除权限

```http
POST /rpc/delete_permission
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "permission_name": "permission.to.delete",
  "force_delete": false
}
```

**响应：**

```json
{
  "success": true,
  "message": "权限删除成功",
  "permission_name": "permission.to.delete"
}
```

#### 列出权限

```http
POST /rpc/list_permissions
Content-Type: application/json
Authorization: Bearer <admin_token>

{}
```

**响应：**

```json
[
  {
    "permission_name": "system.admin",
    "description": "系统管理权限",
    "display_name": "系统管理",
    "resource_type": "system",
    "action_type": "admin",
    "created_at": "2024-01-01T10:00:00Z"
  }
]
```

#### 为角色授予权限

```http
POST /rpc/grant_permission_to_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "p_role_name": "role_name",
  "p_permission_name": "permission.name",
  "p_granted_by": "admin"
}
```

**响应：**

```json
{
  "success": true,
  "message": "权限授予成功",
  "role_name": "role_name",
  "permission_name": "permission.name"
}
```

#### 从角色撤销权限

```http
POST /rpc/revoke_permission_from_role
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "p_role_name": "role_name",
  "p_permission_name": "permission.name"
}
```

**响应：**

```json
{
  "success": true,
  "message": "权限撤销成功",
  "role_name": "role_name",
  "permission_name": "permission.name"
}
```

#### 获取角色权限

```http
POST /rpc/get_role_permissions
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "p_role_name": "role_name"
}
```

**响应：**

```json
[
  {
    "permission_name": "data.read",
    "description": "数据读取权限",
    "display_name": "数据读取",
    "resource_type": "data",
    "action_type": "read",
    "granted_at": "2024-01-01T10:00:00Z"
  }
]
```

#### 检查权限

```http
POST /rpc/check_permission
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_name": "username",
  "permission_name": "data.read"
}
```

**响应：** 布尔值 `true` 或 `false`

#### 获取用户权限

```http
POST /rpc/get_user_permissions
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_name": "username"
}
```

**响应：**

```json
[
  {
    "permission_name": "data.read",
    "description": "数据读取权限",
    "resource_type": "data",
    "action_type": "read",
    "role_name": "user"
  }
]
```

### Schema 管理

#### 添加 Schema

```http
POST /rpc/add_schema
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "schema_name": "new_schema"
}
```

**响应：**

```json
{
  "success": true,
  "message": "Schema创建成功",
  "schema_name": "new_schema"
}
```

#### 删除 Schema

```http
POST /rpc/del_schema
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "schema_name": "schema_to_delete",
  "cascade_delete": false
}
```

**响应：**

```json
{
  "success": true,
  "message": "Schema删除成功",
  "schema_name": "schema_to_delete",
  "cascade": false
}
```

### 工具函数

#### 检查当前用户是否为管理员

```http
POST /rpc/current_user_is_admin
Content-Type: application/json
Authorization: Bearer <token>

{}
```

**响应：** 布尔值 `true` 或 `false`

#### 获取当前用户名

```http
POST /rpc/current_username
Content-Type: application/json
Authorization: Bearer <token>

{}
```

**响应：** 字符串形式的用户名

## 权限要求

- **管理员权限**：需要 `system.admin` 权限

  - 用户管理（创建、删除、修改）
  - 角色管理（创建、删除、修改、分配）
  - 权限管理（创建、删除、修改、授予）
  - Schema 管理
  - 查看所有用户列表

- **用户权限**：普通用户权限

  - 修改自己的密码
  - 查看自己的权限
  - 检查自己的权限

- **匿名访问**：无需认证
  - 用户登录获取 Token

## 错误处理

所有 API 接口统一返回错误格式：

```json
{
  "success": false,
  "message": "错误描述信息"
}
```

常见错误：

- `401 Unauthorized`: 未认证或 Token 无效
- `403 Forbidden`: 权限不足
- `400 Bad Request`: 参数错误或业务逻辑错误

## 最佳实践

1. **Token 管理**

   - Access Token 有效期较短（默认 15 分钟）
   - Refresh Token 有效期较长（默认 7 天）
   - 定期刷新 Access Token
   - 及时撤销不需要的 Refresh Token

2. **权限设计**

   - 使用资源类型和操作类型的组合命名权限
   - 遵循最小权限原则
   - 定期审查和清理不需要的权限

3. **用户管理**
   - 优先使用软删除而非强制删除
   - 定期清理过期的角色分配
   - 合理设置用户的 Schema 权限范围
