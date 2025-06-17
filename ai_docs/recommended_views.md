# 数据底座建议增加的视图设计

## 概述

基于数据底座的功能需求和业务场景，建议增加以下视图来提升系统的可观测性、可管理性和易用性。

## 1. 数据质量监控视图

### 1.1 数据完整性统计视图 (v_data_completeness_stats)

```sql
-- 数据完整性统计视图
CREATE VIEW v_data_completeness_stats AS
SELECT
    bl.library_name,
    bl.library_type,
    COUNT(*) as total_records,
    COUNT(CASE WHEN bl.status = 'complete' THEN 1 END) as complete_records,
    COUNT(CASE WHEN bl.status = 'incomplete' THEN 1 END) as incomplete_records,
    ROUND(
        COUNT(CASE WHEN bl.status = 'complete' THEN 1 END) * 100.0 / COUNT(*),
        2
    ) as completeness_rate,
    bl.created_at::date as check_date
FROM basic_libraries bl
GROUP BY bl.library_name, bl.library_type, bl.created_at::date
ORDER BY check_date DESC, completeness_rate ASC;
```

**用途**：监控各基础库的数据完整性情况，支持数据质量管理中的完整性检查需求。

### 1.2 数据准确性监控视图 (v_data_accuracy_monitor)

```sql
-- 数据准确性监控视图
CREATE VIEW v_data_accuracy_monitor AS
SELECT
    library_name,
    data_source,
    COUNT(*) as total_validations,
    COUNT(CASE WHEN validation_status = 'passed' THEN 1 END) as passed_validations,
    COUNT(CASE WHEN validation_status = 'failed' THEN 1 END) as failed_validations,
    ROUND(
        COUNT(CASE WHEN validation_status = 'passed' THEN 1 END) * 100.0 / COUNT(*),
        2
    ) as accuracy_rate,
    MAX(last_validation_time) as last_check_time
FROM basic_libraries bl
WHERE validation_status IS NOT NULL
GROUP BY library_name, data_source
ORDER BY accuracy_rate ASC;
```

**用途**：监控数据准确性验证结果，及时发现数据质量问题。

### 1.3 数据时效性分析视图 (v_data_timeliness_analysis)

```sql
-- 数据时效性分析视图
CREATE VIEW v_data_timeliness_analysis AS
SELECT
    library_name,
    data_source,
    COUNT(*) as total_records,
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))/3600) as avg_update_delay_hours,
    COUNT(CASE WHEN updated_at - created_at > INTERVAL '24 hours' THEN 1 END) as delayed_records,
    ROUND(
        COUNT(CASE WHEN updated_at - created_at <= INTERVAL '24 hours' THEN 1 END) * 100.0 / COUNT(*),
        2
    ) as timeliness_rate,
    MAX(updated_at) as last_update_time
FROM basic_libraries
WHERE updated_at IS NOT NULL
GROUP BY library_name, data_source
ORDER BY timeliness_rate ASC;
```

**用途**：分析数据更新的时效性，确保数据按要求及时更新。

## 2. 数据资产管理视图

### 2.1 数据资产地图视图 (v_data_asset_map)

```sql
-- 数据资产地图视图
CREATE VIEW v_data_asset_map AS
SELECT
    bl.library_name,
    bl.library_type,
    bl.data_source,
    bl.description,
    tl.theme_name,
    tl.theme_category,
    COUNT(DISTINCT bl.id) as asset_count,
    SUM(bl.data_size_mb) as total_size_mb,
    bl.created_at,
    bl.updated_at,
    CASE
        WHEN bl.updated_at > NOW() - INTERVAL '7 days' THEN 'hot'
        WHEN bl.updated_at > NOW() - INTERVAL '30 days' THEN 'warm'
        ELSE 'cold'
    END as data_temperature
FROM basic_libraries bl
LEFT JOIN thematic_libraries tl ON bl.theme_id = tl.id
GROUP BY bl.library_name, bl.library_type, bl.data_source, bl.description,
         tl.theme_name, tl.theme_category, bl.created_at, bl.updated_at
ORDER BY bl.updated_at DESC;
```

**用途**：提供数据资产的全景视图，支持数据发现和数据目录管理。

### 2.2 数据使用情况统计视图 (v_data_usage_stats)

```sql
-- 数据使用情况统计视图
CREATE VIEW v_data_usage_stats AS
SELECT
    bl.library_name,
    bl.library_type,
    COUNT(DISTINCT s.user_id) as unique_users,
    COUNT(s.id) as total_access_count,
    COUNT(CASE WHEN s.access_type = 'api' THEN 1 END) as api_access_count,
    COUNT(CASE WHEN s.access_type = 'subscription' THEN 1 END) as subscription_count,
    COUNT(CASE WHEN s.access_type = 'sync' THEN 1 END) as sync_count,
    MAX(s.access_time) as last_access_time,
    AVG(s.response_time_ms) as avg_response_time_ms
FROM basic_libraries bl
LEFT JOIN sharing_logs s ON bl.id = s.library_id
WHERE s.access_time > NOW() - INTERVAL '30 days'
GROUP BY bl.library_name, bl.library_type
ORDER BY total_access_count DESC;
```

**用途**：统计数据使用情况，支持数据热度分析和资源优化。

## 3. 数据访问监控视图

### 3.1 API 调用统计视图 (v_api_call_stats)

```sql
-- API调用统计视图
CREATE VIEW v_api_call_stats AS
SELECT
    DATE(s.access_time) as access_date,
    s.api_endpoint,
    COUNT(*) as total_calls,
    COUNT(CASE WHEN s.status_code = 200 THEN 1 END) as successful_calls,
    COUNT(CASE WHEN s.status_code >= 400 THEN 1 END) as failed_calls,
    ROUND(
        COUNT(CASE WHEN s.status_code = 200 THEN 1 END) * 100.0 / COUNT(*),
        2
    ) as success_rate,
    AVG(s.response_time_ms) as avg_response_time,
    MAX(s.response_time_ms) as max_response_time
FROM sharing_logs s
WHERE s.access_type = 'api'
  AND s.access_time > NOW() - INTERVAL '30 days'
GROUP BY DATE(s.access_time), s.api_endpoint
ORDER BY access_date DESC, total_calls DESC;
```

**用途**：监控 API 调用性能和成功率，支持系统性能优化。

### 3.2 用户行为分析视图 (v_user_behavior_analysis)

```sql
-- 用户行为分析视图
CREATE VIEW v_user_behavior_analysis AS
SELECT
    s.user_id,
    u.username,
    u.role,
    COUNT(DISTINCT s.library_id) as accessed_libraries,
    COUNT(s.id) as total_operations,
    COUNT(CASE WHEN s.access_type = 'api' THEN 1 END) as api_calls,
    COUNT(CASE WHEN s.access_type = 'subscription' THEN 1 END) as subscriptions,
    MIN(s.access_time) as first_access,
    MAX(s.access_time) as last_access,
    EXTRACT(EPOCH FROM (MAX(s.access_time) - MIN(s.access_time)))/86400 as active_days
FROM sharing_logs s
LEFT JOIN users u ON s.user_id = u.id
WHERE s.access_time > NOW() - INTERVAL '90 days'
GROUP BY s.user_id, u.username, u.role
ORDER BY total_operations DESC;
```

**用途**：分析用户使用行为，支持用户管理和权限优化。

## 4. 数据治理视图

### 4.1 权限分配概览视图 (v_permission_overview)

```sql
-- 权限分配概览视图
CREATE VIEW v_permission_overview AS
SELECT
    r.role_name,
    COUNT(DISTINCT ur.user_id) as user_count,
    COUNT(DISTINCT rp.permission_id) as permission_count,
    STRING_AGG(DISTINCT p.permission_name, ', ') as permissions,
    COUNT(DISTINCT g.library_id) as governed_libraries
FROM roles r
LEFT JOIN user_roles ur ON r.id = ur.role_id
LEFT JOIN role_permissions rp ON r.id = rp.role_id
LEFT JOIN permissions p ON rp.permission_id = p.id
LEFT JOIN governance_policies g ON r.id = g.role_id
GROUP BY r.role_name
ORDER BY user_count DESC;
```

**用途**：提供权限分配的全局视图，支持权限管理和审计。

### 4.2 数据脱敏状态视图 (v_data_masking_status)

```sql
-- 数据脱敏状态视图
CREATE VIEW v_data_masking_status AS
SELECT
    bl.library_name,
    bl.library_type,
    gp.masking_level,
    gp.masking_rules,
    COUNT(*) as record_count,
    COUNT(CASE WHEN gp.masking_status = 'masked' THEN 1 END) as masked_records,
    COUNT(CASE WHEN gp.masking_status = 'unmasked' THEN 1 END) as unmasked_records,
    ROUND(
        COUNT(CASE WHEN gp.masking_status = 'masked' THEN 1 END) * 100.0 / COUNT(*),
        2
    ) as masking_coverage,
    MAX(gp.updated_at) as last_masking_update
FROM basic_libraries bl
LEFT JOIN governance_policies gp ON bl.id = gp.library_id
WHERE gp.policy_type = 'masking'
GROUP BY bl.library_name, bl.library_type, gp.masking_level, gp.masking_rules
ORDER BY masking_coverage ASC;
```

**用途**：监控数据脱敏状态，确保敏感数据得到适当保护。

## 5. 运营监控视图

### 5.1 系统性能监控视图 (v_system_performance_monitor)

```sql
-- 系统性能监控视图
CREATE VIEW v_system_performance_monitor AS
SELECT
    DATE(s.access_time) as monitor_date,
    COUNT(*) as total_requests,
    AVG(s.response_time_ms) as avg_response_time,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY s.response_time_ms) as p95_response_time,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY s.response_time_ms) as p99_response_time,
    COUNT(CASE WHEN s.response_time_ms > 200 THEN 1 END) as slow_requests,
    ROUND(
        COUNT(CASE WHEN s.status_code = 200 THEN 1 END) * 100.0 / COUNT(*),
        2
    ) as success_rate,
    COUNT(DISTINCT s.user_id) as active_users
FROM sharing_logs s
WHERE s.access_time > NOW() - INTERVAL '30 days'
GROUP BY DATE(s.access_time)
ORDER BY monitor_date DESC;
```

**用途**：监控系统性能指标，确保满足性能要求（99%的 API 调用在 200ms 内响应）。

### 5.2 存储空间使用视图 (v_storage_usage_monitor)

```sql
-- 存储空间使用视图
CREATE VIEW v_storage_usage_monitor AS
SELECT
    library_type,
    theme_name,
    COUNT(*) as library_count,
    SUM(data_size_mb) as total_size_mb,
    AVG(data_size_mb) as avg_size_mb,
    MAX(data_size_mb) as max_size_mb,
    SUM(data_size_mb) / 1024.0 as total_size_gb,
    ROUND(
        SUM(data_size_mb) * 100.0 / SUM(SUM(data_size_mb)) OVER (),
        2
    ) as storage_percentage
FROM basic_libraries bl
LEFT JOIN thematic_libraries tl ON bl.theme_id = tl.id
GROUP BY library_type, theme_name
ORDER BY total_size_mb DESC;
```

**用途**：监控存储空间使用情况，支持容量规划和资源管理。

## 6. 数据血缘关系视图

### 6.1 数据血缘关系视图 (v_data_lineage)

```sql
-- 数据血缘关系视图
CREATE VIEW v_data_lineage AS
SELECT
    source_bl.library_name as source_library,
    source_bl.library_type as source_type,
    target_bl.library_name as target_library,
    target_bl.library_type as target_type,
    dl.transformation_type,
    dl.transformation_rules,
    dl.created_at as lineage_created,
    dl.updated_at as lineage_updated
FROM data_lineage dl
JOIN basic_libraries source_bl ON dl.source_library_id = source_bl.id
JOIN basic_libraries target_bl ON dl.target_library_id = target_bl.id
ORDER BY dl.updated_at DESC;
```

**用途**：追踪数据流转关系，支持数据血缘分析和影响分析。

## 实施建议

### 1. 优先级排序

1. **高优先级**：数据质量监控视图、系统性能监控视图
2. **中优先级**：数据资产管理视图、API 调用统计视图
3. **低优先级**：数据血缘关系视图、用户行为分析视图

### 2. 实施步骤

1. 首先创建基础的监控视图，确保系统可观测性
2. 然后创建业务相关的管理视图，提升管理效率
3. 最后创建高级分析视图，支持深度分析需求

### 3. 性能考虑

- 为视图中的关键字段创建索引
- 考虑使用物化视图来提升查询性能
- 定期更新统计信息以保证查询优化器的准确性

### 4. 权限控制

- 为不同角色的用户分配不同的视图访问权限
- 敏感数据视图需要额外的权限控制
- 审计所有视图的访问记录

## 总结

这些视图的设计基于数据底座的功能需求，涵盖了数据质量管理、数据治理、运营监控等核心场景。通过这些视图，可以大大提升数据底座的可观测性、可管理性和易用性，更好地支持智慧园区的数据驱动决策。
