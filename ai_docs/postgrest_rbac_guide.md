# PostgREST 角色权限管理系统完整指南

## 概述

基于 PostgREST 的角色权限管理系统（RBAC - Role-Based Access Control）为少量用户的系统提供了企业级的权限控制功能。该系统完全基于 PostgreSQL 和 PostgREST，无需额外的权限管理服务。

## 重要修复说明

### 加密函数访问问题修复

**问题**：在 PostgREST 环境中，当函数在 `postgrest` schema 中执行时，默认的 `search_path` 只包含 `postgrest` schema，无法访问 `extensions` schema 中的加密函数（如 `crypt` 和 `gen_salt`）。

**解决方案**：在所有需要使用加密函数的函数开头添加：

```sql
perform set_config('search_path', 'postgrest,extensions,public', true);
```

**影响的函数**：

- `get_token`：用户登录验证
- `add_user`：创建新用户
- `change_password`：修改密码
- `verify_token`：验证 JWT token

这个修复确保了所有加密相关的操作能够正常工作，解决了 "function crypt(text, text) does not exist" 错误。

## PostgREST Schema 配置

### 重要说明

所有对 PostgREST 的 API 请求都必须包含以下 HTTP headers 来指定要访问的数据库 schema：

- `Accept-Profile: postgrest` - 指定响应数据来源的 schema
- `Content-Profile: postgrest` - 指定请求数据目标的 schema

这些 headers 告诉 PostgREST 要访问 `postgrest` schema 中的函数和表，这是我们权限管理系统所在的 schema。

### 示例

```bash
curl -X POST "http://localhost:3000/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{"username": "admin", "password": "things2024"}'
```

## 系统架构

### 核心组件

1. **用户管理**：用户基本信息和认证
2. **角色管理**：定义不同的用户角色
3. **权限管理**：细粒度的权限定义
4. **角色权限关联**：角色与权限的映射关系
5. **用户角色关联**：用户与角色的分配关系
6. **JWT Token**：包含用户角色和权限信息

### 数据库表结构

```sql
-- 用户表
postgrest.users (
  username text primary key,
  password_hash text not null,
  email text,
  full_name text,
  created_at timestamp with time zone,
  updated_at timestamp with time zone,
  is_active boolean
)

-- 角色表
postgrest.roles (
  role_name text primary key,
  description text,
  is_system_role boolean,
  created_at timestamp with time zone,
  updated_at timestamp with time zone
)

-- 权限表
postgrest.permissions (
  permission_name text primary key,
  description text,
  resource_type text,
  action_type text,
  created_at timestamp with time zone
)

-- 角色权限关联表
postgrest.role_permissions (
  role_name text,
  permission_name text,
  granted_at timestamp with time zone,
  granted_by text,
  primary key (role_name, permission_name)
)

-- 用户角色关联表
postgrest.user_roles (
  username text,
  role_name text,
  assigned_at timestamp with time zone,
  assigned_by text,
  expires_at timestamp with time zone,
  is_active boolean,
  primary key (username, role_name)
)
```

## 默认角色和权限

### 系统预定义角色

1. **admin**：系统管理员，拥有所有权限
2. **user**：普通用户，基本读写权限
3. **readonly**：只读用户，仅查看权限
4. **guest**：访客用户，最小权限

### 权限分类

#### 用户管理权限

- `users.select`：查看用户信息
- `users.insert`：创建用户
- `users.update`：更新用户信息
- `users.delete`：删除用户

#### 系统管理权限

- `roles.manage`：管理角色
- `permissions.manage`：管理权限
- `system.admin`：系统管理

#### 数据操作权限

- `data.read`：数据读取
- `data.write`：数据写入
- `data.delete`：数据删除

## API 使用指南

### 1. 用户管理

#### 创建用户（带角色分配）

```bash
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
    "display_name": "约翰·多伊",
    "default_role": "user"
  }'
```

**参数说明：**

- `target_schemas`: 支持多个 schema，用逗号分隔，如 "public,postgrest,custom_schema"
- 系统会为用户在每个指定的 schema 中授予完整的权限（表、序列、函数的增删改查权限）
- 如果某个 schema 不存在，系统会跳过并记录警告，不会影响其他 schema 的权限授予

响应：

```json
{
  "success": true,
  "message": "用户创建成功",
  "username": "john_doe",
  "target_schemas": "public,postgrest",
  "default_role": "user"
}
```

#### 更新用户 Schema 权限

```bash
curl -X POST "http://localhost:3000/rpc/update_user_schemas" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "user_name": "john_doe",
    "new_target_schemas": "public,postgrest,custom_schema"
  }'
```

**功能说明：**

- 重新设置已存在用户的 schema 权限
- 会先撤销用户在所有 schema 中的现有权限，然后授予新 schema 的权限
- 支持多个 schema，用逗号分隔
- 只有 admin 用户可以执行此操作

响应：

```json
{
  "success": true,
  "message": "用户schema权限更新成功",
  "username": "john_doe",
  "new_target_schemas": "public,postgrest,custom_schema"
}
```

#### 用户登录（获取包含角色权限的 Token）

##### 普通用户登录

```bash
curl -X POST "http://localhost:3000/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "username": "john_doe",
    "password": "secure_password123"
  }'
```

响应：

```json
{
  "success": true,
  "message": "登录成功",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "username": "john_doe",
  "roles": ["user"],
  "permissions": ["users.select", "data.read", "data.write"],
  "is_superuser": false
}
```

##### PostgreSQL 超级用户登录

系统支持 `postgres` 超级用户直接登录，无需在 `postgrest.users` 表中创建记录。这特别适合后台服务使用固定的数据库用户配置：

```bash
curl -X POST "http://localhost:3000/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "username": "postgres",
    "password": "your_postgres_password"
  }'
```

响应：

```json
{
  "success": true,
  "message": "postgres超级用户登录成功",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "username": "postgres",
  "roles": ["admin", "postgres_superuser"],
  "permissions": [
    "system.admin",
    "users.select",
    "users.insert",
    "users.update",
    "users.delete",
    "roles.manage",
    "permissions.manage",
    "data.read",
    "data.write",
    "data.delete",
    "schema.create",
    "schema.delete",
    "database.admin"
  ],
  "is_superuser": true
}
```

**特殊说明：**

- `postgres` 用户的密码验证直接通过数据库进行，不依赖 `postgrest.users` 表
- `postgres` 用户自动获得所有管理员权限，包括特殊的数据库管理权限
- Token 中包含 `is_superuser: true` 标记，便于客户端识别
- 适合后台服务使用，避免因普通用户密码变更导致的配置问题

### 2. 角色管理

#### 创建角色

```bash
curl -X POST "http://localhost:3000/rpc/create_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "role_name": "project_manager",
    "description": "项目管理员角色",
    "display_name": "项目管理员",
    "is_system_role": false
  }'
```

响应：

```json
{
  "success": true,
  "message": "角色创建成功",
  "role_name": "project_manager",
  "description": "项目管理员角色",
  "display_name": "项目管理员",
  "is_system_role": false
}
```

#### 更新角色

```bash
curl -X POST "http://localhost:3000/rpc/update_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "role_name": "project_manager",
    "new_description": "更新后的项目管理员角色描述",
    "new_display_name": "高级项目管理员"
  }'
```

#### 删除角色

```bash
curl -X POST "http://localhost:3000/rpc/delete_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "role_name": "project_manager",
    "force_delete": false
  }'
```

**参数说明：**

- `force_delete`: 是否强制删除。如果为 false，系统角色和正在使用的角色不能删除

#### 列出所有角色

```bash
curl -X POST "http://localhost:3000/rpc/list_roles" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{}'
```

响应：

```json
[
  {
    "role_name": "admin",
    "description": "系统管理员，拥有所有权限",
    "is_system_role": true,
    "created_at": "2024-01-01T10:00:00+00:00",
    "updated_at": "2024-01-01T10:00:00+00:00",
    "user_count": 1,
    "permission_count": 10
  },
  {
    "role_name": "project_manager",
    "description": "项目管理员角色",
    "is_system_role": false,
    "created_at": "2024-01-01T11:00:00+00:00",
    "updated_at": "2024-01-01T11:00:00+00:00",
    "user_count": 0,
    "permission_count": 0
  }
]
```

#### 分配角色给用户

```bash
curl -X POST "http://localhost:3000/rpc/assign_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "username": "john_doe",
    "role_name": "admin",
    "assigned_by": "system_admin",
    "expires_at": "2024-12-31T23:59:59Z"
  }'
```

#### 撤销用户角色

```bash
curl -X POST "http://localhost:3000/rpc/revoke_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "username": "john_doe",
    "role_name": "user"
  }'
```

### 3. 权限管理

#### 创建权限

```bash
curl -X POST "http://localhost:3000/rpc/create_permission" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "permission_name": "projects.manage",
    "description": "管理项目权限",
    "display_name": "项目管理",
    "resource_type": "project",
    "action_type": "manage"
  }'
```

响应：

```json
{
  "success": true,
  "message": "权限创建成功",
  "permission_name": "projects.manage",
  "description": "管理项目权限",
  "display_name": "项目管理",
  "resource_type": "project",
  "action_type": "manage"
}
```

#### 更新权限

```bash
curl -X POST "http://localhost:3000/rpc/update_permission" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "permission_name": "projects.manage",
    "new_description": "更新后的项目管理权限描述",
    "new_display_name": "高级项目管理"
  }'
```

#### 删除权限

```bash
curl -X POST "http://localhost:3000/rpc/delete_permission" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "permission_name": "projects.manage",
    "force_delete": false
  }'
```

**参数说明：**

- `force_delete`: 是否强制删除。如果为 false，正在被角色使用的权限不能删除

#### 列出所有权限

```bash
curl -X POST "http://localhost:3000/rpc/list_permissions" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{}'
```

响应：

```json
[
  {
    "permission_name": "users.select",
    "description": "查看用户信息",
    "resource_type": "table",
    "action_type": "select",
    "created_at": "2024-01-01T10:00:00+00:00",
    "role_count": 3
  },
  {
    "permission_name": "projects.manage",
    "description": "管理项目权限",
    "resource_type": "project",
    "action_type": "manage",
    "created_at": "2024-01-01T11:00:00+00:00",
    "role_count": 0
  }
]
```

#### 为角色分配权限

```bash
curl -X POST "http://localhost:3000/rpc/grant_permission_to_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "p_role_name": "project_manager",
    "p_permission_name": "projects.manage",
    "p_granted_by": "admin"
  }'
```

#### 从角色撤销权限

```bash
curl -X POST "http://localhost:3000/rpc/revoke_permission_from_role" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "p_role_name": "project_manager",
    "p_permission_name": "projects.manage"
  }'
```

#### 获取角色的所有权限

```bash
curl -X POST "http://localhost:3000/rpc/get_role_permissions" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "p_role_name": "project_manager"
  }'
```

响应：

```json
[
  {
    "permission_name": "projects.manage",
    "description": "管理项目权限",
    "resource_type": "project",
    "action_type": "manage",
    "granted_at": "2024-01-01T12:00:00+00:00",
    "granted_by": "admin"
  }
]
```

### 4. 权限检查

#### 检查用户是否有特定权限

```bash
curl -X POST "http://localhost:3000/rpc/check_permission" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "username": "john_doe",
    "permission_name": "users.delete"
  }'
```

#### 获取用户的所有权限

```bash
curl -X POST "http://localhost:3000/rpc/get_user_permissions" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d '{
    "username": "john_doe"
  }'
```

响应：

```json
[
  {
    "permission_name": "users.select",
    "description": "查看用户信息",
    "resource_type": "table",
    "action_type": "select",
    "role_name": "user"
  },
  {
    "permission_name": "data.read",
    "description": "数据读取",
    "resource_type": "data",
    "action_type": "select",
    "role_name": "user"
  }
]
```

### 5. 用户列表（包含角色信息）

```bash
curl -X POST "http://localhost:3000/rpc/list_users" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest"
```

响应：

```json
[
  {
    "username": "john_doe",
    "email": "john@example.com",
    "full_name": "John Doe",
    "is_active": true,
    "created_at": "2024-01-01T10:00:00+00:00",
    "updated_at": "2024-01-01T10:00:00+00:00",
    "has_db_user": true,
    "roles": ["user", "admin"]
  }
]
```

## 实现行级安全策略（RLS）

### 1. 启用 RLS

```sql
-- 为业务表启用行级安全
ALTER TABLE your_business_table ENABLE ROW LEVEL SECURITY;
```

### 2. 创建安全策略

```sql
-- 示例：用户只能访问自己的数据
CREATE POLICY user_data_policy ON your_business_table
  FOR ALL TO authenticator
  USING (
    -- 检查JWT token中的用户名
    current_setting('request.jwt.claims', true)::json->>'role' = owner_username
  );

-- 示例：管理员可以访问所有数据
CREATE POLICY admin_all_access ON your_business_table
  FOR ALL TO authenticator
  USING (
    -- 检查用户是否有admin角色
    postgrest.check_permission(
      current_setting('request.jwt.claims', true)::json->>'role',
      'system.admin'
    )
  );

-- 示例：基于权限的访问控制
CREATE POLICY permission_based_access ON your_business_table
  FOR SELECT TO authenticator
  USING (
    postgrest.check_permission(
      current_setting('request.jwt.claims', true)::json->>'role',
      'data.read'
    )
  );
```

### 3. 动态权限检查函数

```sql
-- 创建权限检查函数，用于RLS策略
CREATE OR REPLACE FUNCTION check_user_permission(required_permission text)
RETURNS boolean AS $$
DECLARE
  current_user text;
  user_permissions text[];
BEGIN
  -- 从JWT token获取当前用户
  current_user := current_setting('request.jwt.claims', true)::json->>'role';

  -- 从JWT token获取用户权限
  user_permissions := ARRAY(
    SELECT json_array_elements_text(
      current_setting('request.jwt.claims', true)::json->'permissions'
    )
  );

  -- 检查是否有所需权限
  RETURN required_permission = ANY(user_permissions);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
```

## Go 语言集成示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type RBACClient struct {
    BaseURL string
    Token   string
}

type User struct {
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    FullName  string    `json:"full_name"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    HasDBUser bool      `json:"has_db_user"`
    Roles     []string  `json:"roles"`
}

type Permission struct {
    PermissionName string `json:"permission_name"`
    Description    string `json:"description"`
    ResourceType   string `json:"resource_type"`
    ActionType     string `json:"action_type"`
    RoleName       string `json:"role_name"`
}

func NewRBACClient(baseURL string) *RBACClient {
    return &RBACClient{BaseURL: baseURL}
}

func (c *RBACClient) CreateUser(username, password, targetSchemas, email, fullName, displayName, role string) error {
    data := map[string]interface{}{
        "user_name":       username,
        "user_password":   password,
        "target_schemas":  targetSchemas,
        "email":           email,
        "full_name":       fullName,
        "display_name":    displayName,
        "default_role":    role,
    }

    return c.callRPC("add_user", data, nil)
}

func (c *RBACClient) UpdateUserSchemas(username, newTargetSchemas string) error {
    data := map[string]interface{}{
        "user_name":          username,
        "new_target_schemas": newTargetSchemas,
    }

    return c.callRPC("update_user_schemas", data, nil)
}

func (c *RBACClient) AddSchema(schemaName string) error {
    data := map[string]interface{}{
        "schema_name": schemaName,
    }

    return c.callRPC("add_schema", data, nil)
}

func (c *RBACClient) DelSchema(schemaName string, cascadeDelete bool) error {
    data := map[string]interface{}{
        "schema_name":    schemaName,
        "cascade_delete": cascadeDelete,
    }

    return c.callRPC("del_schema", data, nil)
}

func (c *RBACClient) Login(username, password string) error {
    data := map[string]string{
        "username": username,
        "password": password,
    }

    var response map[string]interface{}
    err := c.callRPC("get_token", data, &response)
    if err != nil {
        return err
    }

    if success, ok := response["success"].(bool); ok && success {
        if token, ok := response["token"].(string); ok {
            c.Token = token
        }
    }

    return nil
}

// PostgresLogin 专门用于 postgres 超级用户登录
func (c *RBACClient) PostgresLogin(password string) error {
    return c.Login("postgres", password)
}

// IsPostgresSuperuser 检查当前登录用户是否是 postgres 超级用户
func (c *RBACClient) IsPostgresSuperuser() (bool, error) {
    if c.Token == "" {
        return false, fmt.Errorf("未登录")
    }

    // 验证当前 token
    var response map[string]interface{}
    err := c.callRPC("verify_token", map[string]string{"token": c.Token}, &response)
    if err != nil {
        return false, err
    }

    if success, ok := response["success"].(bool); ok && success {
        if username, ok := response["username"].(string); ok && username == "postgres" {
            return true, nil
        }
    }

    return false, nil
}

func (c *RBACClient) AssignRole(username, roleName, assignedBy string, expiresAt *time.Time) error {
    data := map[string]interface{}{
        "username":    username,
        "role_name":   roleName,
        "assigned_by": assignedBy,
    }

    if expiresAt != nil {
        data["expires_at"] = expiresAt.Format(time.RFC3339)
    }

    return c.callRPC("assign_role", data, nil)
}

func (c *RBACClient) CheckPermission(username, permission string) (bool, error) {
    data := map[string]string{
        "username":        username,
        "permission_name": permission,
    }

    var result bool
    err := c.callRPC("check_permission", data, &result)
    return result, err
}

func (c *RBACClient) GetUserPermissions(username string) ([]Permission, error) {
    data := map[string]string{"username": username}

    var permissions []Permission
    err := c.callRPC("get_user_permissions", data, &permissions)
    return permissions, err
}

func (c *RBACClient) ListUsers() ([]User, error) {
    var users []User
    err := c.callRPC("list_users", map[string]interface{}{}, &users)
    return users, err
}

// 角色管理方法
func (c *RBACClient) CreateRole(roleName, description, displayName string, isSystemRole bool) error {
    data := map[string]interface{}{
        "role_name":      roleName,
        "description":    description,
        "display_name":   displayName,
        "is_system_role": isSystemRole,
    }
    return c.callRPC("create_role", data, nil)
}

func (c *RBACClient) UpdateRole(roleName, newDescription, newDisplayName string) error {
    data := map[string]interface{}{
        "role_name":         roleName,
        "new_description":   newDescription,
        "new_display_name":  newDisplayName,
    }
    return c.callRPC("update_role", data, nil)
}

func (c *RBACClient) DeleteRole(roleName string, forceDelete bool) error {
    data := map[string]interface{}{
        "role_name":    roleName,
        "force_delete": forceDelete,
    }
    return c.callRPC("delete_role", data, nil)
}

func (c *RBACClient) ListRoles() ([]map[string]interface{}, error) {
    var roles []map[string]interface{}
    err := c.callRPC("list_roles", map[string]interface{}{}, &roles)
    return roles, err
}

// 权限管理方法
func (c *RBACClient) CreatePermission(permissionName, description, displayName, resourceType, actionType string) error {
    data := map[string]interface{}{
        "permission_name": permissionName,
        "description":     description,
        "display_name":    displayName,
        "resource_type":   resourceType,
        "action_type":     actionType,
    }
    return c.callRPC("create_permission", data, nil)
}

func (c *RBACClient) UpdatePermission(permissionName, newDescription, newDisplayName, newResourceType, newActionType string) error {
    data := map[string]interface{}{
        "permission_name":     permissionName,
        "new_description":     newDescription,
        "new_display_name":    newDisplayName,
        "new_resource_type":   newResourceType,
        "new_action_type":     newActionType,
    }
    return c.callRPC("update_permission", data, nil)
}

func (c *RBACClient) DeletePermission(permissionName string, forceDelete bool) error {
    data := map[string]interface{}{
        "permission_name": permissionName,
        "force_delete":    forceDelete,
    }
    return c.callRPC("delete_permission", data, nil)
}

func (c *RBACClient) ListPermissions() ([]map[string]interface{}, error) {
    var permissions []map[string]interface{}
    err := c.callRPC("list_permissions", map[string]interface{}{}, &permissions)
    return permissions, err
}

// 角色权限关联管理方法
func (c *RBACClient) GrantPermissionToRole(roleName, permissionName, grantedBy string) error {
    data := map[string]interface{}{
        "p_role_name":       roleName,
        "p_permission_name": permissionName,
        "p_granted_by":      grantedBy,
    }
    return c.callRPC("grant_permission_to_role", data, nil)
}

func (c *RBACClient) RevokePermissionFromRole(roleName, permissionName string) error {
    data := map[string]interface{}{
        "p_role_name":       roleName,
        "p_permission_name": permissionName,
    }
    return c.callRPC("revoke_permission_from_role", data, nil)
}

func (c *RBACClient) GetRolePermissions(roleName string) ([]map[string]interface{}, error) {
    data := map[string]string{"p_role_name": roleName}
    var permissions []map[string]interface{}
    err := c.callRPC("get_role_permissions", data, &permissions)
    return permissions, err
}

func (c *RBACClient) callRPC(function string, data interface{}, result interface{}) error {
    jsonData, _ := json.Marshal(data)

    req, err := http.NewRequest("POST", c.BaseURL+"/rpc/"+function, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept-Profile", "postgrest")
    req.Header.Set("Content-Profile", "postgrest")
    if c.Token != "" {
        req.Header.Set("Authorization", "Bearer "+c.Token)
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if result != nil {
        return json.NewDecoder(resp.Body).Decode(result)
    }

    return nil
}

// 使用示例
func main() {
    client := NewRBACClient("http://localhost:3000")

    // 方式1：使用 postgres 超级用户登录（推荐用于后台服务）
    err := client.PostgresLogin("your_postgres_password")
    if err != nil {
        fmt.Printf("Error logging in as postgres: %v\n", err)

        // 方式2：创建并使用管理员用户登录
        err = client.CreateUser("admin", "admin123", "public,postgrest", "admin@example.com", "Administrator", "系统管理员", "admin")
        if err != nil {
            fmt.Printf("Error creating admin: %v\n", err)
        }

        err = client.Login("admin", "admin123")
        if err != nil {
            fmt.Printf("Error logging in: %v\n", err)
            return
        }
    }

    // 检查是否是超级用户
    isSuperuser, err := client.IsPostgresSuperuser()
    if err != nil {
        fmt.Printf("Error checking superuser status: %v\n", err)
    } else {
        fmt.Printf("Is postgres superuser: %v\n", isSuperuser)
    }

    // 创建普通用户
    err = client.CreateUser("user1", "user123", "public", "user1@example.com", "User One", "用户一", "user")
    if err != nil {
        fmt.Printf("Error creating user: %v\n", err)
    }

    // 创建测试schema
    err = client.AddSchema("test_schema")
    if err != nil {
        fmt.Printf("Error creating schema: %v\n", err)
    }

    // 更新用户schema权限
    err = client.UpdateUserSchemas("user1", "public,test_schema")
    if err != nil {
        fmt.Printf("Error updating user schemas: %v\n", err)
    }

    // 分配额外角色
    err = client.AssignRole("user1", "readonly", "admin", nil)
    if err != nil {
        fmt.Printf("Error assigning role: %v\n", err)
    }

    // 检查权限
    hasPermission, err := client.CheckPermission("user1", "data.read")
    if err != nil {
        fmt.Printf("Error checking permission: %v\n", err)
    } else {
        fmt.Printf("User1 has data.read permission: %v\n", hasPermission)
    }

    // 获取用户权限列表
    permissions, err := client.GetUserPermissions("user1")
    if err != nil {
        fmt.Printf("Error getting permissions: %v\n", err)
    } else {
        fmt.Printf("User1 permissions: %+v\n", permissions)
    }

    // 列出所有用户
    users, err := client.ListUsers()
    if err != nil {
        fmt.Printf("Error listing users: %v\n", err)
    } else {
        fmt.Printf("All users: %+v\n", users)
    }

    // 创建自定义角色
    err = client.CreateRole("project_manager", "项目管理员角色", "项目管理员", false)
    if err != nil {
        fmt.Printf("Error creating role: %v\n", err)
    }

    // 创建自定义权限
    err = client.CreatePermission("projects.manage", "管理项目权限", "项目管理", "project", "manage")
    if err != nil {
        fmt.Printf("Error creating permission: %v\n", err)
    }

    // 为角色分配权限
    err = client.GrantPermissionToRole("project_manager", "projects.manage", "admin")
    if err != nil {
        fmt.Printf("Error granting permission to role: %v\n", err)
    }

    // 为用户分配新角色
    err = client.AssignRole("user1", "project_manager", "admin", nil)
    if err != nil {
        fmt.Printf("Error assigning role: %v\n", err)
    }

    // 列出所有角色
    roles, err := client.ListRoles()
    if err != nil {
        fmt.Printf("Error listing roles: %v\n", err)
    } else {
        fmt.Printf("All roles: %+v\n", roles)
    }

    // 列出所有权限
    permissions, err := client.ListPermissions()
    if err != nil {
        fmt.Printf("Error listing permissions: %v\n", err)
    } else {
        fmt.Printf("All permissions: %+v\n", permissions)
    }

    // 获取角色的权限
    rolePermissions, err := client.GetRolePermissions("project_manager")
    if err != nil {
        fmt.Printf("Error getting role permissions: %v\n", err)
    } else {
        fmt.Printf("Project manager permissions: %+v\n", rolePermissions)
    }

    // 清理测试数据
    err = client.RevokePermissionFromRole("project_manager", "projects.manage")
    if err != nil {
        fmt.Printf("Error revoking permission: %v\n", err)
    }

    err = client.DeletePermission("projects.manage", true)
    if err != nil {
        fmt.Printf("Error deleting permission: %v\n", err)
    }

    err = client.DeleteRole("project_manager", true)
    if err != nil {
        fmt.Printf("Error deleting role: %v\n", err)
    }

    // 清理测试schema
    err = client.DelSchema("test_schema", true)
    if err != nil {
        fmt.Printf("Error deleting schema: %v\n", err)
    }
}
```

## PostgreSQL 超级用户支持

### 功能概述

为了满足后台服务的需求，系统特别支持 `postgres` 超级用户直接通过 `get_token` 函数获取 JWT token，无需在 `postgrest.users` 表中创建用户记录。这解决了以下问题：

1. **配置稳定性**：后台服务可以使用固定的 `postgres` 用户配置，不会因为其他用户密码变更而影响服务
2. **权限完整性**：`postgres` 超级用户自动获得所有管理权限，包括数据库级别的特殊权限
3. **部署简化**：无需预先创建管理员用户，直接使用数据库超级用户即可

### 实现原理

当 `get_token` 函数检测到用户名为 `postgres` 时，会：

1. **密码验证**：直接通过 `pg_authid` 系统表验证密码，支持 MD5 和 crypt 加密方式
2. **角色分配**：自动分配 `admin` 和 `postgres_superuser` 角色
3. **权限授予**：授予所有系统权限，包括：
   - 用户管理权限（增删改查）
   - 角色和权限管理权限
   - 数据操作权限（读写删除）
   - Schema 管理权限
   - 数据库管理权限
4. **Token 生成**：生成包含超级用户标识的 JWT token

### 安全考虑

1. **密码保护**：`postgres` 用户密码应该设置为强密码，并妥善保管
2. **访问控制**：建议只在后台服务中使用，避免在前端应用中暴露
3. **审计日志**：所有 `postgres` 用户的操作都会被记录
4. **网络安全**：确保 PostgREST 服务的网络访问受到适当保护

### 使用场景

#### 1. 后台服务配置

```yaml
# docker-compose.yml 或配置文件
services:
  backend-service:
    environment:
      - DB_USER=postgres
      - DB_PASSWORD=your_secure_password
      - POSTGREST_URL=http://postgrest:3000
```

#### 2. 应用启动时的初始化

```go
func initializeService() error {
    client := NewRBACClient(os.Getenv("POSTGREST_URL"))

    // 使用 postgres 超级用户登录
    err := client.PostgresLogin(os.Getenv("DB_PASSWORD"))
    if err != nil {
        return fmt.Errorf("failed to login as postgres: %v", err)
    }

    // 验证超级用户状态
    isSuperuser, err := client.IsPostgresSuperuser()
    if err != nil || !isSuperuser {
        return fmt.Errorf("postgres superuser verification failed")
    }

    log.Println("Successfully initialized with postgres superuser privileges")
    return nil
}
```

#### 3. 系统维护脚本

```bash
#!/bin/bash
# 系统维护脚本示例

POSTGRES_PASSWORD="your_secure_password"
POSTGREST_URL="http://localhost:3000"

# 获取 postgres 超级用户 token
TOKEN_RESPONSE=$(curl -s -X POST "${POSTGREST_URL}/rpc/get_token" \
  -H "Content-Type: application/json" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest" \
  -d "{\"username\": \"postgres\", \"password\": \"${POSTGRES_PASSWORD}\"}")

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.token')

# 使用 token 执行管理操作
curl -X POST "${POSTGREST_URL}/rpc/list_users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept-Profile: postgrest" \
  -H "Content-Profile: postgrest"
```

### 与普通用户的区别

| 特性       | 普通用户                             | postgres 超级用户                        |
| ---------- | ------------------------------------ | ---------------------------------------- |
| 用户记录   | 需要在 `postgrest.users` 表中        | 不需要用户记录                           |
| 密码验证   | 通过 `postgrest.users.password_hash` | 通过 `pg_authid.rolpassword`             |
| 角色分配   | 通过 `postgrest.user_roles` 表       | 自动分配 `admin` 和 `postgres_superuser` |
| 权限获取   | 通过角色权限关联表                   | 自动获得所有权限                         |
| Token 标识 | `is_superuser: false`                | `is_superuser: true`                     |
| 适用场景   | 业务用户、前端应用                   | 后台服务、系统维护                       |

## 最佳实践

### 1. 角色设计原则

- **最小权限原则**：用户只获得完成工作所需的最小权限
- **职责分离**：不同职能的用户分配不同角色
- **角色层次化**：设计清晰的角色层次结构

### 2. 权限管理

- **细粒度权限**：根据业务需求定义具体的权限
- **权限组合**：通过角色组合实现复杂的权限需求
- **定期审查**：定期检查和更新权限分配

### 3. 安全考虑

- **JWT 安全**：使用强密钥，设置合理的过期时间
- **密码策略**：强制使用强密码
- **审计日志**：记录重要的权限变更操作
- **会话管理**：实现安全的会话管理机制

### 4. 性能优化

- **权限缓存**：在 JWT token 中包含权限信息，减少数据库查询
- **索引优化**：为权限查询相关的字段创建索引
- **批量操作**：支持批量的角色和权限分配

## 扩展功能

### 1. 动态权限

```sql
-- 创建动态权限表
CREATE TABLE postgrest.dynamic_permissions (
  id serial primary key,
  username text,
  resource_id text,
  permission_type text,
  granted_at timestamp with time zone default now(),
  expires_at timestamp with time zone
);
```

### 2. 权限继承

```sql
-- 创建角色层次表
CREATE TABLE postgrest.role_hierarchy (
  parent_role text,
  child_role text,
  primary key (parent_role, child_role)
);
```

### 3. 条件权限

```sql
-- 基于时间的权限控制
CREATE OR REPLACE FUNCTION check_time_based_permission(
  username text,
  permission_name text,
  check_time timestamp with time zone default now()
) RETURNS boolean AS $$
-- 实现基于时间的权限检查逻辑
$$ LANGUAGE plpgsql;
```

## 故障排除

### 常见问题

#### 1. Permission denied for schema postgrest

**错误信息**：

```json
{
  "code": "42501",
  "details": null,
  "hint": null,
  "message": "permission denied for schema postgrest"
}
```

**原因**：PostgREST 的 `authenticator` 角色没有足够的权限访问 `postgrest` schema 中的函数和表。

**解决方案**：
确保在数据库初始化脚本中包含以下权限设置：

```sql
-- 授予 schema 使用权限
grant usage on schema postgrest to authenticator;
grant create on schema postgrest to authenticator;

-- 授予表权限
grant select, insert, update, delete on all tables in schema postgrest to authenticator;

-- 授予序列权限
grant usage, select on all sequences in schema postgrest to authenticator;

-- 授予函数执行权限
grant execute on all functions in schema postgrest to authenticator;

-- 设置默认权限（对未来创建的对象）
ALTER DEFAULT PRIVILEGES IN SCHEMA postgrest GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticator;
ALTER DEFAULT PRIVILEGES IN SCHEMA postgrest GRANT USAGE, SELECT ON SEQUENCES TO authenticator;
ALTER DEFAULT PRIVILEGES IN SCHEMA postgrest GRANT EXECUTE ON FUNCTIONS TO authenticator;
```

#### 2. 函数不存在错误

**错误信息**：

```json
{
  "code": "42883",
  "details": null,
  "hint": null,
  "message": "function postgrest.get_token(username => text, password => text) does not exist"
}
```

**解决方案**：

1. 确认已正确执行 `postgrest.sql` 初始化脚本
2. 检查 PostgREST 配置中的 `db-schemas` 设置是否包含 `postgrest`
3. 重启 PostgREST 服务

#### 3. JWT 密钥配置问题

**问题**：Token 验证失败或无法生成 Token

**解决方案**：
在 PostgREST 配置中设置 JWT 密钥：

```bash
# 环境变量方式
export PGRST_JWT_SECRET="your-secret-key"

# 或在配置文件中
jwt-secret = "your-secret-key"
```

在数据库中也可以设置：

```sql
-- 设置应用级别的 JWT 密钥
ALTER DATABASE your_database SET app.settings.jwt_secret TO 'your-secret-key';
```

### 调试技巧

#### 1. 检查权限

```sql
-- 检查 authenticator 角色的权限
SELECT
    schemaname,
    tablename,
    privilege_type
FROM information_schema.table_privileges
WHERE grantee = 'authenticator'
    AND schemaname = 'postgrest';

-- 检查函数权限
SELECT
    routine_schema,
    routine_name,
    privilege_type
FROM information_schema.routine_privileges
WHERE grantee = 'authenticator'
    AND routine_schema = 'postgrest';
```

#### 2. 测试数据库连接

```bash
# 直接连接数据库测试
psql -h localhost -U authenticator -d your_database -c "SELECT current_user;"
```

#### 3. 查看 PostgREST 日志

启动 PostgREST 时启用详细日志：

```bash
postgrest config.conf --log-level debug
```

## 总结

PostgREST 完全可以实现一个功能完整、安全可靠的角色权限管理系统，特别适合少量用户的系统。该方案具有以下优势：

1. **完全基于 PostgreSQL**：利用数据库的强大功能
2. **标准化 API**：遵循 RESTful 设计原则
3. **高性能**：直接数据库操作，减少中间层
4. **易于维护**：SQL 函数易于理解和维护
5. **灵活扩展**：可根据业务需求灵活扩展
6. **安全可靠**：基于成熟的数据库安全机制

对于少量用户的系统，这个方案提供了企业级的权限管理功能，同时保持了简单性和可维护性。
