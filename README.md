# 数据底座服务 (DataHub Service)

智慧园区数据底座后台服务，基于 Go 语言和 Dapr 微服务框架构建，提供数据采集、处理、存储、治理和共享功能。

## 项目特性

- 🏗️ **微服务架构**: 基于 Dapr 框架，支持云原生部署
- 📊 **数据管理**: 完整的数据基础库和主题库管理
- 🔐 **访问控制**: 基于 RBAC 的权限管理系统
- 🔄 **数据处理**: 支持实时和批量数据处理
- 📈 **数据治理**: 数据质量管理、元数据管理、数据脱敏
- 🌐 **数据共享**: API、订阅、同步等多种数据共享方式
- 📝 **API 文档**: 完整的 Swagger API 文档

## 技术栈

- **语言**: Go 1.23.1
- **框架**: Dapr, Chi Router
- **数据库**: PostgreSQL (通过 GORM)
- **文档**: Swagger/OpenAPI
- **监控**: Prometheus

## 快速开始

### 环境要求

- Go 1.23.1+
- PostgreSQL 12+
- Docker (可选)

### 安装依赖

```bash
go mod tidy
```

### 配置数据库

设置环境变量或使用默认配置：

```bash
export DATABASE_URL="host=localhost user=postgres password=postgres dbname=datahub port=5432 sslmode=disable TimeZone=Asia/Shanghai"
```

### 运行服务

```bash
go run main.go
```

服务将在端口 80 启动（可通过 LISTEN_PORT 环境变量修改）。

### 访问 API 文档

启动服务后，访问 `http://localhost/swagger/index.html` 查看完整的 API 文档。

## 项目结构

```
datahub-service/
├── api/                    # API层
│   ├── controllers/        # 控制器
│   └── routes.go          # 路由配置
├── service/               # 服务层
│   ├── models/            # 数据模型
│   ├── database/          # 数据库操作
│   ├── *_service.go       # 业务服务
│   └── init.go           # 服务初始化
├── docs/                  # API文档
├── dev_docs/             # 开发文档
│   ├── requirements.md    # 需求文档
│   ├── model.md          # 数据模型设计
│   └── dev_points.md     # 开发要点
├── main.go               # 程序入口
├── go.mod                # 依赖管理
└── README.md             # 项目说明
```

## 核心功能模块

### 1. 数据基础库管理

- 创建、查询、更新、删除数据基础库
- 数据接口管理
- 数据源配置
- 字段定义和清洗规则

### 2. 数据主题库管理

- 主题库创建和管理
- 数据流程图设计
- 复杂数据处理流程

### 3. 访问控制

- 用户管理
- 角色权限管理
- API 访问令牌
- 细粒度权限控制

### 4. 数据治理

- 数据质量监控
- 元数据管理
- 数据脱敏
- 审计日志

### 5. 数据共享服务

- RESTful API
- 数据订阅
- 库表同步
- 安全传输

## API 示例

### 创建数据基础库

```bash
curl -X POST http://localhost/api/v1/basic-libraries \
  -H "Content-Type: application/json" \
  -d '{
    "name_zh": "用户数据基础库",
    "name_en": "user_basic_library",
    "description": "存储用户基础信息的数据库"
  }'
```

### 查询数据基础库列表

```bash
curl "http://localhost/api/v1/basic-libraries?page=1&size=10&status=active"
```

## 开发规范

### 代码注释标准

每个代码文件必须包含开头注释，格式如下：

```go
/*
 * @module 模块名称
 * @description 模块功能描述
 * @architecture 架构模式
 * @documentReference 相关文档路径
 * @stateFlow 状态流转描述
 * @rules 业务规则和约束
 * @dependencies 依赖的模块或包
 * @refs 相关参考文档
 */
```

### 数据库设计

- 使用 UUID 作为主键
- 支持软删除（状态字段）
- 统一的时间戳字段
- JSONB 字段存储复杂配置

### API 设计

- RESTful 风格
- 统一的响应格式
- 完整的错误处理
- Swagger 文档注释

## 部署

### Docker 部署

```bash
# 构建镜像
docker build -t datahub-service .

# 运行容器
docker run -p 80:80 \
  -e DATABASE_URL="your_database_url" \
  datahub-service
```

### Kubernetes 部署

参考 `k8s/` 目录下的配置文件。

## 监控

服务提供 Prometheus 监控指标，访问 `/metrics` 端点获取监控数据。

## 贡献

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。

## 联系方式

如有问题或建议，请提交 Issue 或联系开发团队。
