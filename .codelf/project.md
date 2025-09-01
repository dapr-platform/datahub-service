## 数据底座服务 (DataHub Service)

> 智慧园区数据底座后台服务，基于 Go 语言和 Dapr 微服务框架构建，提供数据采集、处理、存储、治理和共享功能。

> 为智慧园区提供统一的数据管理和服务平台，支持多种数据源接入、数据治理、数据共享等核心功能。

> 开发中 - 核心功能模块基本完成，SyncTask 重构项目已完成，实现了基础库和专题库的统一同步任务管理架构。已完成完整的 CRUD 接口、任务控制接口、批量操作和统计功能，删除了向后兼容接口，提供了完整的 Swagger 文档支持。采用全局服务初始化模式，简化了控制器创建逻辑，提高了代码一致性和维护性。

> 开发团队：数据平台团队

> Go 1.23.1 + Dapr 微服务框架 + PostgreSQL + GORM + Chi Router + Swagger

## Dependencies

- **Core Framework & Runtime**

  - github.com/dapr/go-sdk (v1.11.0): Dapr 微服务开发 SDK，提供服务调用、状态管理、发布订阅等功能
  - go (1.23.1): Go 语言运行时

- **HTTP Framework & Middleware**

  - github.com/go-chi/chi/v5 (v5.1.0): 轻量级 HTTP 路由框架，支持中间件和子路由
  - github.com/go-chi/cors (v1.2.1): CORS 跨域支持中间件
  - github.com/go-chi/render (v1.0.3): HTTP 响应渲染工具

- **Database & ORM**

  - gorm.io/gorm (v1.25.12): Go 语言 ORM 框架，提供数据库操作抽象
  - gorm.io/driver/postgres (v1.5.9): PostgreSQL 数据库驱动
  - github.com/lib/pq (v1.10.9): Pure Go PostgreSQL 驱动

- **Message Queue & Cache**

  - github.com/segmentio/kafka-go (v0.4.48): Kafka 消息队列客户端
  - github.com/eclipse/paho.mqtt.golang (v1.4.3): MQTT 消息协议客户端
  - github.com/go-redis/redis/v8 (v8.11.5): Redis 缓存客户端

- **Monitoring & Documentation**

  - github.com/prometheus/client_golang (v1.18.0): Prometheus 监控指标客户端
  - github.com/swaggo/swag (v1.16.4): Swagger API 文档生成工具
  - github.com/swaggo/http-swagger (v1.3.4): HTTP Swagger 文档服务

- **Utilities**
  - github.com/google/uuid (v1.6.0): UUID 生成工具
  - golang.org/x/crypto (v0.38.0): 加密相关工具包
  - golang.org/x/text (v0.25.0): 文本处理工具包

## Development Environment

> 开发环境包含以下工具和依赖：

**运行环境要求：**

- Go 1.23.1+
- PostgreSQL 12+
- Docker (可选)

**开发工具：**

- Swagger 文档：访问 `/swagger/index.html` 查看 API 文档
- 健康检查：访问 `/health` 检查服务状态
- 监控指标：访问 `/metrics` 获取 Prometheus 指标

**环境配置：**

```bash
export DATABASE_URL="host=localhost user=postgres password=postgres dbname=datahub port=5432 sslmode=disable TimeZone=Asia/Shanghai"
export LISTEN_PORT=80  # 可选，默认80端口
```

**构建和运行：**

```bash
# 安装依赖
go mod tidy

# 生成Swagger文档
./swagger.sh

# 运行服务
go run main.go

# 测试
cd scripts && ./quick_test.sh
```

**可用脚本 (scripts/)：**

- `start.sh`: 启动服务脚本
- `quick_test.sh`: 快速测试脚本
- `test_*.sh`: 各种功能测试脚本
- `cleanup_test_data.sh`: 清理测试数据

## Structure

> 项目采用分层架构设计，按功能模块组织代码，确保清晰的职责分离和良好的可维护性。

```
root
- .cursor                          // Cursor编辑器配置目录
    - rules/                       // 编码规则和AI指导文件
        - core.mdc                 // CursorRIPER框架核心配置
        - state.mdc                // 项目状态管理
        - customization.mdc        // 自定义配置
- ai_docs/                         // AI生成的技术文档目录
    - requirements.md              // 项目需求文档
    - backend_api_analysis.md      // 后端API分析文档
    - basic_library_process_impl.md // 基础库处理流程实现文档
    - refactor_sync_task.md        // 同步任务重构计划（刚创建，包含详细重构方案）
    - model.md                     // 数据模型设计文档
    - interfaces.md                // 接口设计文档
    - sse_event_guide.md          // 服务器发送事件指南
- memory-bank/                     // CursorRIPER框架的项目记忆库
    - projectbrief.md             // 项目简介和核心需求
    - systemPatterns.md           // 系统架构模式
    - techContext.md              // 技术上下文
    - activeContext.md            // 当前开发上下文
    - progress.md                 // 开发进度记录
- api/                            // API层 - 处理HTTP请求和响应
    - controllers/                // 控制器目录 - 实现各业务模块的API端点
        - basic_library_controller.go        // 基础库管理API（CRUD操作、数据源管理、接口配置）
        - basic_library_sync_controller.go   // 基础库同步任务API（创建任务、监控进度、管理状态）
        - sync_task_controller.go            // 统一同步任务API（支持基础库和专题库的通用同步接口）
        - thematic_library_controller.go     // 主题库管理API（复杂数据流程、主题库配置）
        - event_controller.go               // 事件管理API（SSE事件推送、事件订阅）
        - governance_controller.go          // 数据治理API（质量监控、元数据管理）
        - meta_controller.go               // 元数据API（字段定义、常量管理）
        - monitoring_controller.go         // 监控API（健康检查、性能指标）
        - quality_controller.go           // 数据质量API（质量规则、清洗配置）
        - sharing_controller.go           // 数据共享API（API访问、订阅服务）
        - table_controller.go             // 数据表管理API（表结构、表操作）
        - health_controller.go            // 健康检查API
        - response.go                     // 统一响应格式定义
    - routes.go                   // 路由配置 - 定义所有API路径和中间件
- service/                        // 服务层 - 核心业务逻辑实现
    - basic_library/              // 基础库服务模块
        - service.go              // 基础库核心服务（库的CRUD、生命周期管理）
        - datasource_service.go   // 数据源服务（连接配置、状态监控、测试连接）
        - interface_service.go    // 接口服务（接口定义、字段配置、表创建）
        - schedule_service.go     // 调度服务（任务调度、状态管理、重试机制）
        - status_service.go       // 状态服务（数据源状态、接口状态监控）
        - validation_service.go   // 验证服务（数据验证、规则检查）
    - thematic_library/           // 主题库服务模块
        - service.go              // 主题库核心服务（主题库管理、复杂流程处理）
    - sync_task_service.go        // 统一同步任务服务（支持基础库和专题库的统一任务管理）
    - sync_engine/                // 同步引擎模块 - 核心数据同步处理
        - sync_engine.go          // 同步引擎主控制器（任务分发、状态管理、并发控制）
        - batch_processor.go      // 批量处理器（大数据量批量同步、性能优化）
        - realtime_processor.go   // 实时处理器（实时数据流处理、低延迟同步）
        - data_transformer.go     // 数据转换器（数据格式转换、字段映射、类型转换）
        - incremental_sync.go     // 增量同步处理器（增量数据识别、差异计算）
    - data_quality/               // 数据质量模块
        - quality_engine.go       // 质量引擎（质量评估、规则执行）
        - quality_monitor.go      // 质量监控（实时监控、告警机制）
        - validator.go            // 数据验证器（字段验证、约束检查）
        - cleanser.go             // 数据清洗器（数据清洗、格式标准化）
    - monitoring/                 // 监控模块
        - monitor_service.go      // 监控服务主控制器（系统监控、性能采集）
        - health_checker.go       // 健康检查（服务健康状态、依赖检查）
        - metrics_collector.go    // 指标收集器（业务指标、性能指标）
        - alert_manager.go        // 告警管理（告警规则、通知机制）
        - notification.go         // 通知服务（多渠道通知、消息格式化）
    - scheduler/                  // 调度器模块
        - task_scheduler.go       // 任务调度器（定时任务、调度策略）
        - task_executor.go        // 任务执行器（任务执行、状态更新）
        - retry_manager.go        // 重试管理器（失败重试、退避策略）
    - database/                   // 数据库模块
        - migrate.go              // 数据库迁移（表结构升级、数据迁移）
        - schema_service.go       // 模式服务（动态表创建、结构管理）
        - sync_migrate.go         // 同步迁移（数据同步时的表结构处理）
        - auto_migrate_view.go    // 自动视图迁移（数据库视图管理）
        - utils.go                // 数据库工具函数
        - views/                  // 数据库视图定义
            - basic_libraries_view.go    // 基础库视图（复杂查询、聚合数据）
            - thematic_libraries_view.go // 主题库视图
            - sync_tasks_view.go          // 同步任务视图（支持基础库和专题库的统一查询）
            - views.md                    // 视图设计规范和说明文档
    - models/                     // 数据模型层 - 定义所有实体模型和数据结构
        - basic_library.go        // 基础库相关模型（BasicLibrary、DataSource、DataInterface等）
        - thematic_library.go     // 主题库相关模型（ThematicLibrary、DataFlow等）
        - sync_task.go            // 通用同步任务模型（统一支持基础库和专题库）
        - sync_engine_models.go   // 同步引擎模型（任务上下文、进度、结果等）
        - sync_models.go          // 同步相关模型（同步配置、状态定义）
        - quality_models.go       // 数据质量模型（质量规则、评估结果）
        - monitoring_models.go    // 监控相关模型（指标定义、告警配置）
        - governance.go           // 数据治理模型（治理策略、合规规则）
        - event.go                // 事件模型（事件定义、消息格式）
        - sharing.go              // 数据共享模型（共享配置、访问控制）
        - table.go                // 数据表模型（表结构、字段定义）
        - jsonb.go                // JSONB类型支持（PostgreSQL JSON字段）
        - config_models.go        // 配置相关模型
        - utils_models.go         // 工具模型（通用数据结构）
    - meta/                       // 元数据模块 - 系统元数据和常量定义
        - sync_task.go            // 同步任务元数据（任务类型、状态常量、验证函数）
        - library_types.go        // 库类型元数据（基础库和专题库的类型定义、验证函数）
        - data_interface.go       // 数据接口元数据（接口类型、字段类型）
        - datasource.go           // 数据源元数据（数据源类型、连接参数）
        - meta_field.go           // 元数据字段定义（通用字段结构）
        - thematic_library.go     // 主题库元数据
    - utils/                      // 工具模块
        - connection_pool.go      // 连接池管理（数据库连接复用）
        - crypto_utils.go         // 加密工具（数据脱敏、密码加密）
        - data_converter.go       // 数据转换工具（类型转换、格式化）
        - health_checker.go       // 健康检查工具
    - config/                     // 配置模块
        - config_manager.go       // 配置管理器（环境配置、动态配置）
    - event_service.go            // 事件服务（SSE事件推送、事件管理）
    - governance_service.go       // 数据治理服务（治理策略执行）
    - sharing_service.go          // 数据共享服务（API服务、订阅管理）
    - init.go                     // 服务初始化（依赖注入、服务启动、全局服务管理）
- client/                         // 客户端连接器模块 - 外部系统集成
    - connectors/                 // 连接器实现
        - kafka_connector.go      // Kafka消息队列连接器（消息发送、消费）
        - mqtt_connector.go       // MQTT物联网协议连接器（设备数据接入）
        - redis_connector.go      // Redis缓存连接器（缓存操作、会话管理）
    - pgmeta.go                   // PostgreSQL元数据客户端（表结构查询、元数据获取）
    - *_test.go                   // 客户端测试文件
- scripts/                        // 脚本目录 - 开发和运维脚本
    - start.sh                    // 服务启动脚本
    - quick_test.sh               // 快速测试脚本（API基本功能测试）
    - test_*.sh                   // 各种专项测试脚本（登录、SSE、PostgREST等）
    - cleanup_test_data.sh        // 测试数据清理脚本
    - manual_test_commands.md     // 手工测试命令集合
    - sse_test.html              // SSE功能测试页面
- docs/                          // 文档目录
    - docs.go                    // Swagger文档生成配置
    - swagger.json/swagger.yaml  // 生成的API文档
- main.go                        // 程序入口（服务启动、路由初始化、中间件配置）
- go.mod/go.sum                  // Go模块依赖管理
- Dockerfile                     // Docker容器化配置
- swagger.sh                     // Swagger文档生成脚本
- README.md                      // 项目说明文档
- *_SUMMARY.md                   // 项目开发总结文档
```
