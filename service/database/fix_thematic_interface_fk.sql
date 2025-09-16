-- 修复主题接口表的外键约束问题
-- 删除不必要的 data_source_id 字段和相关外键约束

-- 检查并删除外键约束
DO $$
BEGIN
    -- 删除 fk_thematic_interfaces_data_source 外键约束（如果存在）
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_thematic_interfaces_data_source' 
        AND table_name = 'thematic_interfaces'
    ) THEN
        ALTER TABLE thematic_interfaces DROP CONSTRAINT fk_thematic_interfaces_data_source;
        RAISE NOTICE '已删除外键约束: fk_thematic_interfaces_data_source';
    END IF;
    
    -- 删除 data_source_id 字段（如果存在）
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'thematic_interfaces' 
        AND column_name = 'data_source_id'
    ) THEN
        ALTER TABLE thematic_interfaces DROP COLUMN data_source_id;
        RAISE NOTICE '已删除字段: data_source_id';
    END IF;
    
    -- 检查并删除其他可能存在的重复外键约束
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_thematic_libraries_interfaces' 
        AND table_name = 'thematic_interfaces'
    ) THEN
        ALTER TABLE thematic_interfaces DROP CONSTRAINT fk_thematic_libraries_interfaces;
        RAISE NOTICE '已删除重复的外键约束: fk_thematic_libraries_interfaces';
    END IF;
    
END $$;

-- 确保正确的外键约束存在
DO $$
BEGIN
    -- 检查并创建正确的主题库外键约束
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_thematic_interfaces_thematic_library' 
        AND table_name = 'thematic_interfaces'
    ) THEN
        ALTER TABLE thematic_interfaces 
        ADD CONSTRAINT fk_thematic_interfaces_thematic_library 
        FOREIGN KEY (library_id) REFERENCES thematic_libraries(id) ON DELETE CASCADE;
        RAISE NOTICE '已创建外键约束: fk_thematic_interfaces_thematic_library';
    END IF;
END $$;
