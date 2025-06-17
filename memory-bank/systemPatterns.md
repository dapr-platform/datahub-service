# 系统架构和技术决策

## 整体架构

### 分层架构设计

```
┌─────────────────────────────────────────┐
│              API Layer                  │
│         (RESTful Controllers)           │
├─────────────────────────────────────────┤
│            Service Layer                │
│         (Business Logic)                │
├─────────────────────────────────────────┤
│             Model Layer                 │
│         (Data Models)                   │
├─────────────────────────────────────────┤
│           Database Layer                │
│      (PostgreSQL + GORM)               │
└─────────────────────────────────────────┘
```

### 微服务架构集成

- **Dapr 框架**: 提供微服务基础设施
- **服务发现**: 通过 Dapr 进行服务注册和发现
- **配置管理**: 支持外部配置注入
- **状态管理**: 利用 Dapr 的状态存储能力

## 核心技术决策

### 1. 编程语言和框架

**Go 语言选择理由**:

- 高性能并发处理
- 简洁的语法和强类型系统
- 优秀的微服务生态
- 容器化部署友好

**关键依赖**:

- `github.com/go-chi/chi/v5`: HTTP 路由框架
- `gorm.io/gorm`: ORM 框架
- `github.com/swaggo/swag`: API 文档生成
- `github.com/google/uuid`: UUID 生成

### 2. 数据库设计

**PostgreSQL 选择理由**:

- 强大的 JSONB 支持
- 优秀的事务处理能力
- 丰富的数据类型支持
- 与 PostgREST 完美集成

**设计模式**:

- UUID 主键设计
- 软删除模式
- JSONB 存储复杂配置
- 外键关联关系

### 3. 权限管理架构

**PostgREST RBAC 选择理由**:

- 基于 PostgreSQL 的原生权限系统
- JWT Token 认证机制
- 细粒度权限控制
- 减少自定义权限代码复杂度

**架构模式**:

```
Client → PostgREST API → PostgreSQL RBAC
                ↓
        DataHub Service API
```

### 4. API 设计模式

**RESTful 设计原则**:

- 资源导向的 URL 设计
- 标准 HTTP 方法使用
- 统一的响应格式
- 完整的错误处理

**响应格式标准化**:

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

## 数据模型设计模式

### 1. 基础模型模式

所有模型都包含标准字段:

- `ID`: UUID 主键
- `CreatedAt`: 创建时间
- `UpdatedAt`: 更新时间
- 软删除支持

### 2. 关联关系模式

- **一对多**: 使用外键关联
- **多对多**: 使用中间表
- **JSONB 配置**: 存储复杂配置信息

### 3. 业务模型分组

- **基础库模块**: BasicLibrary, DataInterface, DataSource
- **主题库模块**: ThematicLibrary, ThematicInterface, DataFlowGraph
- **治理模块**: QualityRule, Metadata, DataMaskingRule
- **共享模块**: ApiApplication, DataSubscription, DataSyncTask

## 服务层设计模式

### 1. 依赖注入模式

```go
type Service struct {
    db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
    return &Service{db: db}
}
```

### 2. 事务处理模式

- 使用 GORM 事务机制
- 自动回滚错误操作
- 支持嵌套事务

### 3. 错误处理模式

- 统一错误类型定义
- 分层错误处理
- 详细错误日志记录

## 部署架构

### 1. 容器化设计

```dockerfile
# 多阶段构建
FROM golang:1.21-alpine AS builder
# 构建阶段...

FROM alpine:latest
# 运行阶段...
```

### 2. 配置管理

- 环境变量配置
- Docker Compose 本地开发
- Kubernetes 生产部署支持

### 3. 监控和日志

- 结构化日志输出
- 健康检查端点
- 性能监控指标

## 安全架构

### 1. 认证授权

- PostgREST JWT Token
- 基于角色的访问控制
- API 密钥管理

### 2. 数据安全

- 数据脱敏规则
- 敏感信息加密
- 访问日志审计

### 3. 网络安全

- CORS 配置
- API 限流控制
- 请求验证

## 扩展性设计

### 1. 水平扩展

- 无状态服务设计
- 数据库连接池
- 负载均衡支持

### 2. 功能扩展

- 模块化代码结构
- 插件化架构支持
- 标准化接口定义

### 3. 数据扩展

- 多数据源支持
- 数据同步机制
- 分布式存储支持
