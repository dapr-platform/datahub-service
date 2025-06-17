# 数据库视图说明文档

本文档描述了数据底座服务中基础库相关的数据库视图，这些视图提供了丰富的查询接口，简化了复杂的多表关联查询。

## 视图列表

### 1. basic_library_info - 基础库信息视图

**功能**: 提供基础库的完整基本信息，包含统计数据

**包含字段**:

- 基础库基本信息：id, name_zh, name_en, description, status, created_at, created_by, updated_at, updated_by
- 统计信息：interface_count（接口总数）, data_source_count（数据源总数）
- 活跃统计：active_interface_count（活跃接口数）
- 数据源类型统计：kafka_source_count, db_source_count, http_source_count

**使用场景**: 基础库列表展示、概览统计

### 2. data_interface_info - 数据接口详细信息视图

**功能**: 提供数据接口的完整信息，包含关联的基础库和数据源信息

**包含字段**:

- 接口信息：interface_id, interface_name_zh/en, interface_type, interface_description, interface_status
- 关联基础库信息：library_id, library_name_zh/en, library_description, library_status
- 关联数据源信息：data_source_id, data_source_type, connection_config, params_config
- 统计信息：field_count（字段数）, cleansing_rule_count（清洗规则数）, primary_key_count（主键数）, active_rule_count（活跃规则数）

**使用场景**: 接口详情页面、接口管理、数据溯源

### 3. data_source_info - 数据源详细信息视图

**功能**: 提供数据源的完整信息，包含关联的基础库信息

**包含字段**:

- 数据源信息：data_source_id, data_source_type, connection_config, params_config
- 关联基础库信息：library_id, library_name_zh/en, library_description, library_status
- 统计信息：related_interface_count（关联接口数）

**使用场景**: 数据源管理、连接配置查看、影响分析

### 4. interface_field_info - 接口字段详细信息视图

**功能**: 提供接口字段的完整信息，包含接口和基础库上下文

**包含字段**:

- 字段信息：field_id, field_name_zh/en, data_type, is_primary_key, is_nullable, default_value, field_description, order_num
- 接口信息：interface_id, interface_name_zh/en, interface_type, interface_status
- 基础库信息：library_id, library_name_zh/en, library_status

**特点**: 按 interface_id 和 order_num 排序

**使用场景**: 字段管理、数据字典生成、接口文档

### 5. cleansing_rule_info - 数据清洗规则信息视图

**功能**: 提供数据清洗规则的完整信息，包含接口和基础库上下文

**包含字段**:

- 规则信息：rule_id, rule_type, rule_config, order_num, is_enabled
- 接口信息：interface_id, interface_name_zh/en, interface_type, interface_status
- 基础库信息：library_id, library_name_zh/en, library_status

**特点**: 按 interface_id 和 order_num 排序

**使用场景**: 数据清洗规则管理、数据质量监控

### 6. basic_library_summary - 基础库汇总统计视图

**功能**: 提供基础库的完整统计信息，用于仪表板和报表

**包含字段**:

- 基础库信息：library_id, library_name_zh/en, description, status
- 接口统计：total_interfaces, active_interfaces, realtime_interfaces, batch_interfaces
- 数据源统计：total_data_sources, kafka_sources, redis_sources, nats_sources, http_sources, db_sources, hostpath_sources
- 字段统计：total_fields, primary_key_fields, required_fields
- 清洗规则统计：total_cleansing_rules, active_cleansing_rules
- 时间信息：last_updated_at（最新更新时间）

**使用场景**: 仪表板、统计报表、资源分析

### 7. active_basic_libraries - 活跃基础库视图

**功能**: 只显示状态为活跃的基础库及其相关信息

**包含字段**:

- 基础库完整信息
- 简单统计：interface_count, data_source_count

**使用场景**: 生产环境监控、活跃资源列表

### 8. basic_library_health - 基础库健康状况视图

**功能**: 检查基础库的数据完整性和配置状况，提供健康评分

**包含字段**:

- 基础信息：library_id, library_name_zh/en, status
- 健康检查指标：
  - has_interfaces：是否有接口
  - has_data_sources：是否有数据源
  - has_fields：是否有字段定义
  - all_interfaces_have_sources：所有接口是否都有数据源
  - all_interfaces_have_primary_keys：所有接口是否都有主键
- 统计信息：interface_count, data_source_count, field_count, rule_count
- 健康评分：health_score（0-100 分）

**健康评分算法**:

- 基础分：20 分
- 有数据源：+20 分
- 有字段定义：+20 分
- 接口都有数据源：+20 分
- 接口都有主键：+20 分

**使用场景**: 数据质量监控、配置完整性检查、系统健康度评估

## 使用示例

### 查询基础库概览

```sql
SELECT * FROM basic_library_info;
```

### 查询特定基础库的详细统计

```sql
SELECT * FROM basic_library_summary WHERE library_id = 'your-library-id';
```

### 查询接口及其关联信息

```sql
SELECT * FROM data_interface_info WHERE library_id = 'your-library-id';
```

### 查询基础库健康状况

```sql
SELECT library_name_zh, health_score,
       has_interfaces, has_data_sources, has_fields
FROM basic_library_health
ORDER BY health_score DESC;
```

### 查询低健康分的基础库

```sql
SELECT library_name_zh, health_score,
       interface_count, data_source_count, field_count
FROM basic_library_health
WHERE health_score < 60;
```

### 查询活跃的基础库

```sql
SELECT * FROM active_basic_libraries;
```

## 维护说明

1. **自动创建**: 这些视图在系统启动时通过 `AutoMigrateView()` 函数自动创建
2. **性能优化**: 视图使用了适当的 JOIN 和 GROUP BY 来优化查询性能
3. **数据一致性**: 视图实时反映底层表的数据变化
4. **扩展性**: 可以根据业务需求添加新的统计字段或视图

## 注意事项

1. 视图中的统计数据是实时计算的，复杂查询可能影响性能
2. 对于大数据量场景，建议添加适当的索引
3. 视图的修改需要更新此文档
4. 删除视图前请确认没有其他系统依赖
