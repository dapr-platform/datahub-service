## Development Guidelines

### Framework and Language

> 项目基于 Go 1.23.1 + Dapr 微服务框架构建，重点关注最佳实践和标准化。

**Framework Considerations:**

- **Dapr Integration**: 充分利用 Dapr 的服务调用、状态管理、发布订阅等功能，避免重复造轮子
- **Version Compatibility**: 确保所有依赖项与 Go 1.23.1 和 Dapr v1.11.0 兼容
- **Performance Patterns**: 遵循 Go 并发模式，合理使用 goroutine 和 channel
- **Upgrade Strategy**: 为未来的框架升级制定最小影响的策略
- **Important Notes for Framework**:
  - 使用 GORM 作为 ORM 框架，统一数据库操作
  - Chi Router 提供轻量级 HTTP 路由，支持中间件链
  - Swagger 自动生成 API 文档，确保文档与代码同步
  - Prometheus 集成用于监控和指标收集

**Language Best Practices:**

- **Type Safety**: 使用 Go 的强类型系统防止运行时错误，合理使用 interface{}
- **Error Handling**: 遵循 Go 的错误处理模式，返回 error 而不是 panic
- **Modern Features**: 利用 Go 1.23.1 的新特性，如泛型、工作区模式等
- **Consistency**: 在整个代码库中应用一致的命名和代码模式
- **Documentation**: 为 Go 特定的实现和解决方案编写文档注释

### Code Abstraction and Reusability

> 开发过程中优先考虑代码抽象和可重用性，确保模块化和组件化功能。在重新发明轮子之前尝试搜索现有解决方案。

**Modular Design Principles:**

- **Single Responsibility**: 每个模块只负责一个功能领域（如 basic_library、sync_engine、data_quality）
- **High Cohesion, Low Coupling**: 相关功能集中在同一模块，减少模块间依赖
- **Stable Interfaces**: 对外暴露稳定的接口，内部实现可以变化

**Reusable Component Library:**

```
service/
- utils/                    // 通用工具包
    - connection_pool.go    // 数据库连接池复用
    - crypto_utils.go       // 加密、解密、脱敏工具
    - data_converter.go     // 数据类型转换工具
    - health_checker.go     // 健康检查通用组件
- models/                   // 数据模型层
    - jsonb.go              // PostgreSQL JSONB类型支持
    - utils_models.go       // 通用数据结构定义
- meta/                     // 元数据和常量定义
    - sync_task.go          // 同步任务相关常量和验证
    - meta_field.go         // 通用字段元数据结构
client/connectors/          // 外部系统连接器
    - kafka_connector.go    // Kafka消息队列连接器
    - mqtt_connector.go     // MQTT物联网协议连接器
    - redis_connector.go    // Redis缓存连接器
```

### Coding Standards and Tools

**Code Formatting Tools:**

- [gofmt]() // Go 代码标准格式化工具
- [golint]() // Go 代码风格检查工具
- [go vet]() // Go 代码静态分析工具
- [swag]() // Swagger 文档生成工具

**Naming and Structure Conventions:**

- **Package Naming**: 使用简短、小写的包名，避免下划线
- **Function Naming**: 公开函数使用 PascalCase，私有函数使用 camelCase
- **Constant Naming**: 使用描述性的常量名，遵循 Go 约定
- **Directory Structure**: 按功能模块组织，遵循标准 Go 项目布局

### Frontend-Backend Collaboration Standards

**API Design and Documentation:**

- **RESTful Design**: 遵循 REST 设计原则
  - 使用 HTTP 方法表示操作：GET（查询）、POST（创建）、PUT（更新）、DELETE（删除）
  - 资源命名使用复数形式：/basic-libraries、/sync/tasks
  - 状态码规范：200（成功）、201（创建）、400（客户端错误）、500（服务器错误）
- **API Documentation**: Swagger 文档及时更新
  - 使用注释标签自动生成文档：@Summary、@Description、@Param、@Success、@Failure
  - 提供完整的请求/响应示例
  - 详细的错误码说明
- **Unified Error Handling**: 统一错误处理规范
  - 使用标准的 APIResponse 结构：Status（状态码）、Msg（消息）、Data（数据）
  - 错误消息国际化支持
  - 详细的错误日志记录

**Data Flow:**

- **Service Layer Management**: 清晰的服务层状态管理
  - 使用依赖注入模式管理服务实例
  - 服务间通过接口交互，降低耦合
  - 统一的事务管理和错误处理
- **Data Validation**: 前后端数据验证
  - 后端进行完整的数据验证和清洗
  - 使用 GORM 的数据验证标签
  - 自定义验证器处理复杂业务规则
- **Async Operation Handling**: 标准化异步操作处理
  - 使用一致的任务状态管理：pending、running、success、failed、cancelled
  - 实时进度反馈和状态通知
  - 优雅的超时和重试机制

### Performance and Security

**Performance Optimization Focus:**

- **Database Optimization**: 数据库性能优化
  - 合理使用数据库索引和查询优化
  - 连接池管理避免连接泄漏
  - 大数据量处理时使用分页和批处理
- **Concurrency Optimization**: 并发性能优化
  - 合理使用 goroutine 处理并发任务
  - 使用 sync.Pool 复用对象减少 GC 压力
  - 避免 goroutine 泄漏和死锁
- **Caching Strategy**: 适当的缓存使用
  - Redis 缓存热点数据
  - 内存缓存频繁访问的元数据
  - 缓存失效策略和一致性保证

**Security Measures:**

- **Input Validation**: 输入验证和过滤
  - SQL 注入防护，使用参数化查询
  - XSS 攻击防护，输入数据转义
  - 数据长度和格式验证
- **Sensitive Data Protection**: 敏感信息保护
  - 数据库连接字符串等敏感配置使用环境变量
  - 用户密码使用安全哈希算法
  - 数据脱敏功能保护隐私数据
- **Access Control**: 访问控制机制
  - 基于角色的访问控制（RBAC）
  - API 访问令牌验证
  - 操作审计日志记录

### Data Management and Sync Engine

**Data Synchronization Patterns:**

- **Task Management**: 同步任务管理
  - 状态流转：pending → running → success/failed/cancelled
  - 重试机制：失败任务自动重试，支持退避策略
  - 并发控制：控制同时执行的任务数量
- **Processor Types**: 处理器类型
  - BatchProcessor：批量数据处理，适合大数据量同步
  - RealtimeProcessor：实时数据处理，低延迟同步
  - IncrementalSync：增量数据同步，节省资源
  - DataTransformer：数据转换和清洗
- **Library Support**: 库类型支持
  - 基础库（BasicLibrary）：结构化数据管理
  - 主题库（ThematicLibrary）：复杂数据流程管理
  - 统一的同步任务模型支持多种库类型

**Data Quality and Governance:**

- **Quality Engine**: 数据质量引擎
  - 数据验证规则配置和执行
  - 质量监控和告警机制
  - 数据清洗和标准化处理
- **Metadata Management**: 元数据管理
  - 数据源元数据自动发现
  - 数据血缘关系追踪
  - 数据字典和文档管理
