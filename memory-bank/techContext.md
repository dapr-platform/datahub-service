# 技术栈和开发环境

## 核心技术栈

### 后端技术

**编程语言**: Go 1.21+

- 高性能并发处理
- 强类型系统
- 丰富的标准库
- 优秀的微服务生态

**Web 框架**: Chi Router v5

- 轻量级 HTTP 路由器
- 中间件支持
- 高性能路由匹配
- RESTful API 友好

**ORM 框架**: GORM v1.25+

- 功能丰富的 Go ORM
- 自动迁移支持
- 关联关系处理
- 事务管理

**数据库**: PostgreSQL 13+

- 强大的 JSONB 支持
- 优秀的事务处理
- 丰富的数据类型
- PostgREST 集成

### 微服务架构

**Dapr 框架**: v1.12+

- 微服务运行时
- 服务发现和调用
- 状态管理
- 配置管理
- 发布订阅

**权限管理**: PostgREST RBAC

- 基于 PostgreSQL 的权限系统
- JWT Token 认证
- 细粒度权限控制
- RESTful API 自动生成

### 开发工具

**API 文档**: Swagger/OpenAPI 3.0

- 自动生成 API 文档
- 交互式 API 测试
- 代码注释驱动

**容器化**: Docker + Docker Compose

- 多阶段构建
- 本地开发环境
- 生产部署支持

## 依赖管理

### Go Modules (go.mod)

```go
module datahub-service

go 1.21

require (
    github.com/go-chi/chi/v5 v5.0.10
    github.com/go-chi/cors v1.2.1
    github.com/go-chi/render v1.0.3
    github.com/google/uuid v1.4.0
    github.com/swaggo/http-swagger v1.3.4
    github.com/swaggo/swag v1.16.2
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)
```

### 关键依赖说明

**HTTP 处理**:

- `chi/v5`: 核心路由框架
- `chi/cors`: CORS 中间件
- `chi/render`: 响应渲染

**数据库**:

- `gorm`: ORM 框架
- `driver/postgres`: PostgreSQL 驱动

**工具库**:

- `google/uuid`: UUID 生成
- `swaggo/swag`: Swagger 文档生成

## 开发环境配置

### 本地开发环境

**必需软件**:

- Go 1.21+
- PostgreSQL 13+
- Docker & Docker Compose
- Git

**IDE 推荐**:

- VS Code + Go 扩展
- GoLand
- Vim/Neovim + Go 插件

### 环境变量配置

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=datahub

# 服务配置
PORT=8080
GIN_MODE=debug

# PostgREST配置
POSTGREST_URL=http://localhost:3000
POSTGREST_JWT_SECRET=your-jwt-secret
```

### Docker 开发环境

**docker-compose.yml**:

```yaml
version: "3.8"
services:
  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: datahub
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  postgrest:
    image: postgrest/postgrest:latest
    environment:
      PGRST_DB_URI: postgres://postgres:password@postgres:5432/datahub
      PGRST_DB_SCHEMAS: postgrest
      PGRST_DB_ANON_ROLE: anonymous
      PGRST_JWT_SECRET: your-jwt-secret
    ports:
      - "3000:3000"
    depends_on:
      - postgres

  datahub-service:
    build: .
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: password
      DB_NAME: datahub
      PORT: 8080
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - postgrest
```

## 构建和部署

### 本地构建

```bash
# 安装依赖
go mod download

# 生成Swagger文档
swag init

# 构建应用
go build -o datahub-service

# 运行应用
./datahub-service
```

### Docker 构建

```bash
# 构建镜像
docker build -t datahub-service .

# 运行容器
docker run -p 8080:8080 datahub-service
```

### 开发工作流

1. **代码开发**

   ```bash
   # 启动开发环境
   docker-compose up -d postgres postgrest

   # 运行应用
   go run main.go
   ```

2. **API 文档更新**

   ```bash
   # 更新Swagger注释后重新生成
   swag init
   ```

3. **数据库迁移**
   ```bash
   # 应用启动时自动执行迁移
   # 或手动执行迁移逻辑
   ```

## 测试环境

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./service/...

# 生成测试覆盖率报告
go test -cover ./...
```

### API 测试

**Swagger UI**: http://localhost:8080/swagger/index.html

- 交互式 API 测试
- 请求/响应示例
- 参数验证

**Postman 集合**:

- 导入 Swagger JSON
- 自动化 API 测试
- 环境变量管理

### 集成测试

```bash
# 启动完整环境
docker-compose up -d

# 运行集成测试
go test -tags=integration ./tests/...
```

## 监控和调试

### 日志配置

- 结构化日志输出
- 不同级别日志分离
- 请求追踪 ID
- 错误堆栈信息

### 健康检查

**端点**:

- `GET /health`: 基础健康检查
- `GET /ready`: 就绪状态检查

### 性能监控

- HTTP 请求指标
- 数据库连接池状态
- 内存和 CPU 使用率
- 响应时间统计

## 代码质量

### 代码规范

- Go 官方代码风格
- gofmt 格式化
- golint 静态检查
- 统一的错误处理

### 文档规范

- 完整的函数注释
- Swagger API 文档
- README 使用说明
- 架构设计文档

### 版本控制

- Git 工作流
- 语义化版本号
- 变更日志维护
- 代码审查流程
