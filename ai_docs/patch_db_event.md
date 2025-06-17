# 数据库事件触发器检查功能

## 需求描述

需要在 event_service 启动时，检查数据库中是否有对应的 trigger。
只需要 数据基础库和数据主题库这两个表的 trigger
正式运行时可以通过 dapr client 调用 postgre_meta 这个服务来获取 trigger 信息。 这个服务本地 debug 时，需要用 http client URL 为 http://localhost:3001. postgre_meta 的 api 接口信息为 pgmeta_openapi.json

## 实现方案

### 1. 触发器检查机制

- 在 `EventService` 启动时自动检查数据库触发器
- 检查的表：`basic_libraries`（数据基础库）和 `thematic_libraries`（数据主题库）
- 对应的触发器名称：`basic_libraries_notify` 和 `thematic_libraries_notify`

### 2. 环境适配

- **生产环境（Dapr）**：通过 Dapr 服务调用 `postgre-meta` 服务
  - URL: `http://localhost:{DAPR_HTTP_PORT}/v1.0/invoke/postgre-meta/method/triggers/`
  - 默认端口：3500
- **本地调试环境**：直接调用 HTTP 接口
  - URL: `http://localhost:3001/triggers/`

### 3. 环境检测

通过检查环境变量自动判断运行环境：

- `DAPR_HTTP_PORT` 或 `DAPR_GRPC_PORT` 存在时使用 Dapr 调用
- 否则使用直接 HTTP 调用

### 4. 触发器验证与自动创建

- 检查触发器是否存在
- 验证触发器是否启用（`ENABLED` 状态）
- **如果触发器不存在，自动创建触发器**
- 记录检查结果和创建过程

### 5. 错误处理

- 网络请求失败时记录错误日志
- 触发器缺失时自动创建
- 创建失败时记录详细错误信息

## 已完成功能

### 1. 触发器检查与创建

✅ **自动检查功能**：

- 服务启动时自动检查 `basic_libraries` 和 `thematic_libraries` 表的触发器
- 支持 Dapr 和本地 HTTP 两种调用方式
- 自动检测运行环境

✅ **自动创建功能**：

- 如果触发器不存在，自动执行 SQL 创建触发器
- 使用 `CREATE OR REPLACE TRIGGER` 确保幂等性
- 触发器调用 `notify_datahub_changes()` 函数

✅ **完整的错误处理**：

- 网络请求失败处理
- SQL 执行失败处理
- 详细的日志记录

### 2. 模型字段更新

✅ **所有模型添加审计字段**：

- `created_by`：记录创建者，默认值为 'system'
- `updated_by`：记录更新者，默认值为 'system'
- 字段长度限制为 100 字符

✅ **更新的模型文件**：

- `service/models/event.go`：事件相关模型
- `service/models/basic_library.go`：数据基础库模型
- `service/models/thematic_library.go`：数据主题库模型
- `service/models/governance.go`：数据治理模型
- `service/models/sharing.go`：数据共享模型

✅ **GORM 钩子函数**：

- `BeforeCreate`：创建前自动设置 `created_by` 和 `updated_by`
- `BeforeUpdate`：更新前自动设置 `updated_by`
- 支持手动指定创建者/更新者

### 3. 代码结构

```
service/
├── event_service.go           # 事件服务主文件
│   ├── checkDatabaseTriggers() # 检查数据库触发器
│   ├── checkTableTrigger()     # 检查单个表触发器
│   ├── createTableTrigger()    # 创建表触发器
│   ├── getTriggers()           # 获取触发器列表
│   ├── isDaprEnvironment()     # 检测运行环境
│   ├── getTriggersViaDapr()    # 通过 Dapr 获取触发器
│   └── getTriggersViaHTTP()    # 通过 HTTP 获取触发器
└── models/                    # 模型文件
    ├── event.go              # 事件模型（已更新）
    ├── basic_library.go      # 基础库模型（已更新）
    ├── thematic_library.go   # 主题库模型（已更新）
    ├── governance.go         # 治理模型（已更新）
    └── sharing.go            # 共享模型（已更新）
```

### 4. 触发器创建 SQL

```sql
CREATE OR REPLACE TRIGGER {table_name}_notify
AFTER INSERT OR UPDATE OR DELETE ON {table_name}
FOR EACH ROW
EXECUTE FUNCTION notify_datahub_changes();
```

### 5. 使用示例

#### 启动服务

```bash
go run main.go
```

#### 日志输出示例

```
2024-01-01 10:00:00 [INFO] 事件服务已启动
2024-01-01 10:00:05 [INFO] 开始检查数据库触发器
2024-01-01 10:00:05 [INFO] 通过 HTTP 获取到 5 个触发器
2024-01-01 10:00:05 [INFO] 表 basic_libraries 的触发器 basic_libraries_notify 已存在
2024-01-01 10:00:05 [WARN] 警告: 表 thematic_libraries 缺少触发器 thematic_libraries_notify，正在创建...
2024-01-01 10:00:05 [INFO] 成功创建表 thematic_libraries 的触发器 thematic_libraries_notify
```

## 技术特点

### 1. 自动化

- 无需手动创建触发器
- 服务启动时自动检查和创建
- 支持多环境自动适配

### 2. 可靠性

- 使用 `CREATE OR REPLACE` 确保幂等性
- 完整的错误处理和日志记录
- 支持网络异常重试

### 3. 可维护性

- 清晰的代码结构和注释
- 统一的审计字段管理
- 标准化的 GORM 钩子函数

### 4. 扩展性

- 易于添加新表的触发器检查
- 支持自定义触发器创建逻辑
- 模块化的设计便于功能扩展

## 配置说明

### 环境变量

- `DAPR_HTTP_PORT`：Dapr HTTP 端口（生产环境）
- `DAPR_GRPC_PORT`：Dapr GRPC 端口（生产环境）

### 依赖服务

- **生产环境**：`postgre-meta` Dapr 服务
- **本地环境**：`http://localhost:3001` PostgreSQL Meta API

## 注意事项

1. **数据库权限**：确保服务有创建触发器的权限
2. **函数依赖**：确保 `notify_datahub_changes()` 函数已存在
3. **网络连接**：确保能访问 PostgreSQL Meta API
4. **审计字段**：新增记录时会自动设置 `created_by` 和 `updated_by`
