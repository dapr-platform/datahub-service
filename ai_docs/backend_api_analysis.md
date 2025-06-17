# 基于 interfaces.md 的后台接口和模型需求分析

## 1. 需求分析总结

根据`interfaces.md`的描述，数据基础库管理需要完善以下功能：

### 1.1 数据源管理需求

- **批量数据源**：数据库、文件、HTTP 接口
- **实时数据源**：Kafka、MQTT
- **数据源测试**：连接测试和数据预览
- **调度配置**：批量数据源的调度管理

### 1.2 接口管理需求

- **接口与表映射**：一个接口对应一个数据库表
- **动态字段管理**：通过 PgMetaApi.ts 动态操作表结构
- **接口测试**：数据获取和性能测试
- **不同数据源配置**：根据数据源类型配置不同参数

## 2. 已实现的后台接口

### 2.1 数据源测试接口

```http
POST /basic-libraries/test-datasource
```

- 测试数据源连接
- 数据预览功能
- 返回测试结果和性能信息

### 2.2 接口调用测试接口

```http
POST /basic-libraries/test-interface
```

- 测试接口数据获取
- 性能测试
- 数据验证

### 2.3 表结构管理接口

```http
POST /basic-libraries/manage-schema
```

- 通过 PgMetaApi 动态创建、修改、删除表结构
- 支持字段定义管理

### 2.4 调度配置接口

```http
POST /basic-libraries/configure-schedule
```

- 配置批量数据源的调度任务
- 支持 cron、interval、manual 模式

### 2.5 状态查询接口

```http
GET /basic-libraries/datasource-status/{id}
GET /basic-libraries/interface-preview/{id}
```

- 数据源状态监控
- 接口数据预览

## 3. 还需要增加的后台接口

### 3.1 数据源配置验证接口

```http
POST /basic-libraries/validate-datasource-config
```

**功能**：验证数据源配置的正确性
**参数**：

- `data_source_type`: 数据源类型
- `connection_config`: 连接配置
- `params_config`: 参数配置

**用途**：在保存数据源前验证配置有效性

### 3.2 数据源模板接口

```http
GET /basic-libraries/datasource-templates/{type}
```

**功能**：获取不同数据源类型的配置模板
**支持类型**：

- `mysql`, `postgresql`, `oracle` (数据库类型)
- `kafka`, `mqtt` (实时流类型)
- `http`, `ftp`, `file` (批量接口类型)

### 3.3 接口字段推断接口

```http
POST /basic-libraries/infer-interface-fields
```

**功能**：根据数据源自动推断接口字段结构
**参数**：

- `data_source_id`: 数据源 ID
- `table_name`: 表名（数据库类型）
- `sample_data`: 样本数据（文件/HTTP 类型）

### 3.4 数据同步任务管理接口

```http
POST /basic-libraries/sync-tasks
GET /basic-libraries/sync-tasks
GET /basic-libraries/sync-tasks/{id}
PUT /basic-libraries/sync-tasks/{id}/cancel
POST /basic-libraries/sync-tasks/{id}/retry
```

**功能**：管理数据同步任务的创建、监控、取消、重试

### 3.5 实时数据源监控接口

```http
GET /basic-libraries/realtime-monitor/{data_source_id}
POST /basic-libraries/realtime-test/{data_source_id}
```

**功能**：

- 监控实时数据源的消息流量
- 测试 Kafka/MQTT 连接和数据接收

### 3.6 数据质量检查接口

```http
POST /basic-libraries/quality-check/{interface_id}
GET /basic-libraries/quality-reports/{interface_id}
```

**功能**：

- 对接口数据进行质量检查
- 生成数据质量报告

### 3.7 性能分析接口

```http
GET /basic-libraries/performance-analysis/{interface_id}
POST /basic-libraries/performance-benchmark
```

**功能**：

- 分析接口查询性能
- 进行性能基准测试

## 4. 需要增加的数据模型

### 4.1 已实现的模型

- `ScheduleConfig`: 调度配置
- `DataSourceStatus`: 数据源状态
- `InterfaceStatus`: 接口状态
- `SyncTask`: 数据同步任务

### 4.2 还需要的模型

#### 4.2.1 DataSourceTemplate (数据源模板)

```go
type DataSourceTemplate struct {
    ID           string                 `json:"id"`
    Type         string                 `json:"type"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    ConfigSchema map[string]interface{} `json:"config_schema"`
    ParamsSchema map[string]interface{} `json:"params_schema"`
    Examples     map[string]interface{} `json:"examples"`
}
```

#### 4.2.2 FieldInferenceRule (字段推断规则)

```go
type FieldInferenceRule struct {
    ID          string                 `json:"id"`
    DataType    string                 `json:"data_type"`
    Pattern     string                 `json:"pattern"`
    Rules       map[string]interface{} `json:"rules"`
    Priority    int                    `json:"priority"`
}
```

#### 4.2.3 DataQualityRule (数据质量规则)

```go
type DataQualityRule struct {
    ID          string                 `json:"id"`
    InterfaceID string                 `json:"interface_id"`
    FieldName   string                 `json:"field_name"`
    RuleType    string                 `json:"rule_type"`
    RuleConfig  map[string]interface{} `json:"rule_config"`
    Severity    string                 `json:"severity"`
    IsEnabled   bool                   `json:"is_enabled"`
}
```

#### 4.2.4 PerformanceMetric (性能指标)

```go
type PerformanceMetric struct {
    ID           string                 `json:"id"`
    InterfaceID  string                 `json:"interface_id"`
    MetricType   string                 `json:"metric_type"`
    MetricValue  float64                `json:"metric_value"`
    MetricUnit   string                 `json:"metric_unit"`
    TestTime     time.Time              `json:"test_time"`
    TestConfig   map[string]interface{} `json:"test_config"`
}
```

#### 4.2.5 RealtimeMonitor (实时监控)

```go
type RealtimeMonitor struct {
    ID           string                 `json:"id"`
    DataSourceID string                 `json:"data_source_id"`
    MessageCount int64                  `json:"message_count"`
    ErrorCount   int64                  `json:"error_count"`
    LastMessage  time.Time              `json:"last_message"`
    Throughput   float64                `json:"throughput"`
    Status       string                 `json:"status"`
    UpdatedAt    time.Time              `json:"updated_at"`
}
```

## 5. 技术实现要点

### 5.1 PgMetaApi 集成

- 需要集成 PostgreSQL Meta API 用于动态表结构管理
- 支持表的创建、修改、删除操作
- 字段类型映射和约束管理

### 5.2 调度系统集成

- 集成 Cron 调度引擎
- 支持任务状态监控和日志记录
- 错误重试和告警机制

### 5.3 实时数据处理

- Kafka Consumer 实现
- MQTT 订阅机制
- 消息格式解析和验证

### 5.4 数据质量检查

- 规则引擎实现
- 异常数据检测
- 质量报告生成

## 6. 优先级建议

### 高优先级

1. 数据源配置验证接口
2. 数据源模板接口
3. 接口字段推断接口
4. 数据同步任务管理接口

### 中优先级

1. 实时数据源监控接口
2. 数据质量检查接口
3. 性能分析接口

### 低优先级

1. 高级性能基准测试
2. 复杂数据质量规则
3. 实时告警系统
