-- 数据库迁移脚本：支持同步任务多接口功能
-- 创建日期：2024-01-01
-- 描述：为同步任务添加多接口支持，创建中间表并迁移现有数据

-- 1. 创建同步任务接口关联表
CREATE TABLE IF NOT EXISTS sync_task_interfaces (
    id VARCHAR(36) PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    interface_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    processed_rows BIGINT DEFAULT 0,
    total_rows BIGINT DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    error_message TEXT,
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    config JSONB,
    result JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- 索引
    INDEX idx_sync_task_interfaces_task_id (task_id),
    INDEX idx_sync_task_interfaces_interface_id (interface_id),
    INDEX idx_sync_task_interfaces_status (status),
    
    -- 外键约束
    FOREIGN KEY (task_id) REFERENCES sync_tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (interface_id) REFERENCES data_interfaces(id) ON DELETE CASCADE,
    
    -- 唯一约束
    UNIQUE KEY uk_sync_task_interfaces_task_interface (task_id, interface_id)
);

-- 2. 迁移现有数据：将sync_tasks表中的interface_id迁移到新的关联表
INSERT INTO sync_task_interfaces (
    id,
    task_id,
    interface_id,
    status,
    progress,
    processed_rows,
    total_rows,
    error_count,
    error_message,
    start_time,
    end_time,
    created_at,
    updated_at
)
SELECT 
    UUID() as id,
    st.id as task_id,
    st.interface_id as interface_id,
    st.status as status,
    st.progress as progress,
    st.processed_rows as processed_rows,
    st.total_rows as total_rows,
    st.error_count as error_count,
    st.error_message as error_message,
    st.start_time as start_time,
    st.end_time as end_time,
    st.created_at as created_at,
    st.updated_at as updated_at
FROM sync_tasks st
WHERE st.interface_id IS NOT NULL AND st.interface_id != '';

-- 3. 备份原有的interface_id列（可选，用于回滚）
-- ALTER TABLE sync_tasks ADD COLUMN interface_id_backup VARCHAR(36);
-- UPDATE sync_tasks SET interface_id_backup = interface_id;

-- 4. 删除sync_tasks表中的interface_id列（注意：这是破坏性操作）
-- ALTER TABLE sync_tasks DROP COLUMN interface_id;

-- 5. 添加触发器以保持updated_at字段自动更新
DELIMITER //
CREATE TRIGGER sync_task_interfaces_updated_at
    BEFORE UPDATE ON sync_task_interfaces
    FOR EACH ROW
BEGIN
    SET NEW.updated_at = CURRENT_TIMESTAMP;
END//
DELIMITER ;

-- 6. 创建视图以便于查询（可选）
CREATE VIEW v_sync_task_with_interfaces AS
SELECT 
    st.*,
    GROUP_CONCAT(sti.interface_id) as interface_ids,
    COUNT(sti.interface_id) as interface_count,
    SUM(CASE WHEN sti.status = 'success' THEN 1 ELSE 0 END) as success_interface_count,
    SUM(CASE WHEN sti.status = 'failed' THEN 1 ELSE 0 END) as failed_interface_count,
    SUM(CASE WHEN sti.status = 'running' THEN 1 ELSE 0 END) as running_interface_count,
    SUM(sti.processed_rows) as total_processed_rows
FROM sync_tasks st
LEFT JOIN sync_task_interfaces sti ON st.id = sti.task_id
GROUP BY st.id;

-- 7. 插入测试数据（可选）
-- 这部分可以根据实际需要添加测试数据

-- 完成迁移
SELECT 'Multi-interface sync task migration completed successfully' as message;
