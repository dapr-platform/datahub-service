-- 主题同步相关表的数据库迁移脚本
-- 版本: 1.0
-- 创建时间: 2025-01-13
-- 描述: 创建主题库数据同步功能所需的数据表

-- 主题同步任务表
CREATE TABLE IF NOT EXISTS thematic_sync_tasks (
    id VARCHAR(36) PRIMARY KEY,
    thematic_library_id VARCHAR(36) NOT NULL,
    thematic_interface_id VARCHAR(36) NOT NULL,
    task_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 源数据配置
    source_libraries JSONB,
    source_interfaces JSONB,
    
    -- 汇聚配置
    aggregation_config JSONB,
    key_matching_rules JSONB,
    field_mapping_rules JSONB,
    
    -- 处理配置
    cleansing_rules JSONB,
    privacy_rules JSONB,
    quality_rules JSONB,
    
    -- 调度配置
    trigger_type VARCHAR(20) NOT NULL DEFAULT 'manual',
    cron_expression VARCHAR(100),
    interval_seconds INTEGER,
    scheduled_time TIMESTAMP,
    next_run_time TIMESTAMP,
    
    -- 状态信息
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    last_sync_time TIMESTAMP,
    last_sync_status VARCHAR(20),
    last_sync_message TEXT,
    
    -- 统计信息
    total_sync_count BIGINT DEFAULT 0,
    successful_sync_count BIGINT DEFAULT 0,
    failed_sync_count BIGINT DEFAULT 0,
    
    -- 审计字段
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(100),
    
    -- 索引
    INDEX idx_thematic_library (thematic_library_id),
    INDEX idx_thematic_interface (thematic_interface_id),
    INDEX idx_status (status),
    INDEX idx_trigger_type (trigger_type),
    INDEX idx_next_run_time (next_run_time)
);

-- 主题同步执行记录表
CREATE TABLE IF NOT EXISTS thematic_sync_executions (
    id VARCHAR(36) PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    execution_type VARCHAR(20) NOT NULL DEFAULT 'manual',
    
    -- 执行状态
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    duration BIGINT DEFAULT 0,
    
    -- 数据统计
    source_record_count BIGINT DEFAULT 0,
    processed_record_count BIGINT DEFAULT 0,
    inserted_record_count BIGINT DEFAULT 0,
    updated_record_count BIGINT DEFAULT 0,
    deleted_record_count BIGINT DEFAULT 0,
    error_record_count BIGINT DEFAULT 0,
    
    -- 处理结果
    processing_result JSONB,
    error_details JSONB,
    quality_report JSONB,
    
    -- 审计字段
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    
    -- 外键约束
    FOREIGN KEY (task_id) REFERENCES thematic_sync_tasks(id) ON DELETE CASCADE,
    
    -- 索引
    INDEX idx_task_id (task_id),
    INDEX idx_status (status),
    INDEX idx_execution_type (execution_type),
    INDEX idx_start_time (start_time)
);

-- 主题数据血缘表
CREATE TABLE IF NOT EXISTS thematic_data_lineages (
    id VARCHAR(36) PRIMARY KEY,
    thematic_interface_id VARCHAR(36) NOT NULL,
    thematic_record_id VARCHAR(255) NOT NULL,
    
    -- 源数据信息
    source_library_id VARCHAR(36) NOT NULL,
    source_interface_id VARCHAR(36) NOT NULL,
    source_record_id VARCHAR(255) NOT NULL,
    source_record_hash VARCHAR(64),
    
    -- 处理信息
    processing_rules JSONB,
    transformation_details JSONB,
    
    -- 质量信息
    quality_score DECIMAL(5,2) DEFAULT 0,
    quality_issues JSONB,
    
    -- 时间信息
    source_data_time TIMESTAMP NOT NULL,
    processed_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- 索引
    INDEX idx_thematic_interface (thematic_interface_id),
    INDEX idx_thematic_record (thematic_record_id),
    INDEX idx_source_library (source_library_id),
    INDEX idx_source_interface (source_interface_id),
    INDEX idx_source_record (source_record_id),
    INDEX idx_processed_time (processed_time)
);

-- 主题同步任务表注释
ALTER TABLE thematic_sync_tasks COMMENT = '主题同步任务表，存储主题数据同步任务的配置和状态信息';

-- 主题同步执行记录表注释
ALTER TABLE thematic_sync_executions COMMENT = '主题同步执行记录表，存储每次同步执行的详细信息和结果';

-- 主题数据血缘表注释
ALTER TABLE thematic_data_lineages COMMENT = '主题数据血缘表，记录主题数据与源数据的血缘关系';

-- 创建触发器，自动更新 updated_at 字段
DELIMITER $$

CREATE TRIGGER IF NOT EXISTS update_thematic_sync_tasks_updated_at
    BEFORE UPDATE ON thematic_sync_tasks
    FOR EACH ROW
BEGIN
    SET NEW.updated_at = CURRENT_TIMESTAMP;
END$$

DELIMITER ;

-- 插入初始数据（如果需要）
-- INSERT INTO thematic_sync_tasks (id, task_name, description, trigger_type, status, created_by, updated_by)
-- VALUES ('default-task-001', '默认同步任务', '系统默认创建的同步任务', 'manual', 'draft', 'system', 'system');

-- 创建视图，方便查询同步任务的统计信息
CREATE VIEW IF NOT EXISTS v_thematic_sync_task_stats AS
SELECT 
    t.id,
    t.task_name,
    t.status,
    t.trigger_type,
    COUNT(e.id) as total_executions,
    COUNT(CASE WHEN e.status = 'success' THEN 1 END) as successful_executions,
    COUNT(CASE WHEN e.status = 'failed' THEN 1 END) as failed_executions,
    AVG(e.duration) as avg_duration,
    MAX(e.start_time) as last_execution_time,
    SUM(e.processed_record_count) as total_processed_records
FROM thematic_sync_tasks t
LEFT JOIN thematic_sync_executions e ON t.id = e.task_id
GROUP BY t.id, t.task_name, t.status, t.trigger_type;

-- 创建视图，方便查询数据血缘统计信息
CREATE VIEW IF NOT EXISTS v_thematic_lineage_stats AS
SELECT 
    source_library_id,
    source_interface_id,
    thematic_interface_id,
    COUNT(*) as lineage_count,
    AVG(quality_score) as avg_quality_score,
    MAX(processed_time) as latest_processed_time
FROM thematic_data_lineages
GROUP BY source_library_id, source_interface_id, thematic_interface_id;
