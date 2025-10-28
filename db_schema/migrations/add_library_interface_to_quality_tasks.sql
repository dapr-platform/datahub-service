-- 为 quality_tasks 表添加库和接口关联字段
-- 执行日期: 2024-01-XX
-- 描述: 添加 library_type, library_id, interface_id 字段，以明确质量检测任务与库和接口的关联关系

-- 1. 添加新字段
ALTER TABLE quality_tasks 
ADD COLUMN library_type VARCHAR(30) NOT NULL DEFAULT 'thematic',
ADD COLUMN library_id VARCHAR(50) NOT NULL DEFAULT '',
ADD COLUMN interface_id VARCHAR(50) NOT NULL DEFAULT '';

-- 2. 为新字段添加注释
COMMENT ON COLUMN quality_tasks.library_type IS '库类型: thematic(主题库), basic(基础库)';
COMMENT ON COLUMN quality_tasks.library_id IS '库ID，关联到具体的库记录';
COMMENT ON COLUMN quality_tasks.interface_id IS '接口ID，关联到具体的接口记录';

-- 3. 创建单字段索引
CREATE INDEX idx_quality_tasks_library_type ON quality_tasks(library_type);
CREATE INDEX idx_quality_tasks_library_id ON quality_tasks(library_id);
CREATE INDEX idx_quality_tasks_interface_id ON quality_tasks(interface_id);

-- 4. 创建复合索引以支持组合查询
CREATE INDEX idx_quality_tasks_lib_interface ON quality_tasks(library_type, library_id, interface_id);

-- 5. 数据迁移（根据实际情况调整）
-- 注意：以下脚本需要根据实际数据情况进行调整
-- 如果有现有数据，需要根据 target_schema 和 target_table 来推导 library_type, library_id, interface_id

-- 示例：更新现有记录（需要根据实际业务逻辑修改）
-- UPDATE quality_tasks qt
-- SET 
--   library_type = CASE 
--     WHEN qt.target_schema LIKE 'thematic_%' THEN 'thematic'
--     ELSE 'basic'
--   END,
--   library_id = (SELECT id FROM libraries WHERE schema_name = qt.target_schema LIMIT 1),
--   interface_id = (SELECT id FROM interfaces WHERE table_name = qt.target_table AND library_id = qt.library_id LIMIT 1)
-- WHERE qt.library_id = '' OR qt.interface_id = '';

-- 6. 验证数据
-- SELECT 
--   id, 
--   name, 
--   library_type, 
--   library_id, 
--   interface_id, 
--   target_schema, 
--   target_table 
-- FROM quality_tasks 
-- LIMIT 10;

-- 7. 可选：如果需要外键约束（根据实际情况决定是否添加）
-- ALTER TABLE quality_tasks 
--   ADD CONSTRAINT fk_quality_tasks_library 
--   FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE;

-- ALTER TABLE quality_tasks 
--   ADD CONSTRAINT fk_quality_tasks_interface 
--   FOREIGN KEY (interface_id) REFERENCES interfaces(id) ON DELETE CASCADE;

-- 8. 移除默认值（在数据迁移完成后）
-- ALTER TABLE quality_tasks ALTER COLUMN library_type DROP DEFAULT;
-- ALTER TABLE quality_tasks ALTER COLUMN library_id DROP DEFAULT;
-- ALTER TABLE quality_tasks ALTER COLUMN interface_id DROP DEFAULT;

