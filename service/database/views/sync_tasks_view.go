package views

var SyncTasksViews = map[string]string{

	// 同步任务详细信息视图 - 包含任务的所有字段和关联信息，支持基础库和专题库
	"sync_tasks_info": `
		DROP VIEW IF EXISTS sync_tasks_info;
		CREATE VIEW sync_tasks_info AS
		SELECT 
			st.id,
			st.library_type,
			st.library_id,
			st.data_source_id,
			st.interface_id,
			st.task_type,
			st.status,
			st.start_time,
			st.end_time,
			st.progress,
			st.processed_rows,
			st.total_rows,
			st.error_count,
			st.error_message,
			st.config,
			st.result,
			st.created_at,
			st.created_by,
			st.updated_at,
			-- 计算执行时长（秒）
			CASE 
				WHEN st.start_time IS NOT NULL AND st.end_time IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (st.end_time - st.start_time))
				WHEN st.start_time IS NOT NULL AND st.end_time IS NULL AND st.status = 'running'
				THEN EXTRACT(EPOCH FROM (NOW() - st.start_time))
				ELSE NULL
			END as duration_seconds,
			-- 数据源信息对象，来源：data_sources表
			-- 包含字段：id, name, type, category, connection_config, params_config, library_id, created_at, created_by, updated_at, updated_by
			jsonb_build_object(
				'id', ds.id,
				'name', ds.name,
				'type', ds.type,
				'category', ds.category,
				'connection_config', ds.connection_config,
				'params_config', ds.params_config,
				'library_id', ds.library_id,
				'created_at', ds.created_at,
				'created_by', ds.created_by,
				'updated_at', ds.updated_at,
				'updated_by', ds.updated_by
			) as data_source,
			-- 基础库信息对象，来源：basic_libraries表（当library_type为basic_library时）
			-- 包含字段：id, name_zh, name_en, description, status, created_at, created_by, updated_at, updated_by
			CASE 
				WHEN st.library_type = 'basic_library' AND bl.id IS NOT NULL THEN
					jsonb_build_object(
						'id', bl.id,
						'name_zh', bl.name_zh,
						'name_en', bl.name_en,
						'description', bl.description,
						'status', bl.status,
						'created_at', bl.created_at,
						'created_by', bl.created_by,
						'updated_at', bl.updated_at,
						'updated_by', bl.updated_by
					)
				ELSE NULL
			END as basic_library,
			-- 专题库信息对象，来源：thematic_libraries表（当library_type为thematic_library时）
			-- 包含字段：id, name_zh, name_en, description, category, domain, publish_status, version, access_level, update_frequency, retention_period, status, created_at, created_by, updated_at, updated_by
			CASE 
				WHEN st.library_type = 'thematic_library' AND tl.id IS NOT NULL THEN
					jsonb_build_object(
						'id', tl.id,
						'name_zh', tl.name_zh,
						'name_en', tl.name_en,
						'description', tl.description,
						'category', tl.category,
						'domain', tl.domain,
						'publish_status', tl.publish_status,
						'version', tl.version,
						'access_level', tl.access_level,
						'update_frequency', tl.update_frequency,
						'retention_period', tl.retention_period,
						'status', tl.status,
						'created_at', tl.created_at,
						'created_by', tl.created_by,
						'updated_at', tl.updated_at,
						'updated_by', tl.updated_by
					)
				ELSE NULL
			END as thematic_library,
			-- 接口信息对象，来源：data_interfaces表（可选）
			-- 包含字段：id, name_zh, name_en, type, description, status, is_table_created, parse_config, interface_config, created_at, created_by, updated_at, updated_by
			CASE 
				WHEN di.id IS NOT NULL THEN
					jsonb_build_object(
						'id', di.id,
						'name_zh', di.name_zh,
						'name_en', di.name_en,
						'type', di.type,
						'description', di.description,
						'status', di.status,
						'is_table_created', di.is_table_created,
						'parse_config', di.parse_config,
						'interface_config', di.interface_config,
						'created_at', di.created_at,
						'created_by', di.created_by,
						'updated_at', di.updated_at,
						'updated_by', di.updated_by
					)
				ELSE NULL
			END as data_interface,
			-- 处理速率（行/秒）
			CASE 
				WHEN st.start_time IS NOT NULL AND st.end_time IS NOT NULL AND st.processed_rows > 0
				THEN st.processed_rows / GREATEST(EXTRACT(EPOCH FROM (st.end_time - st.start_time)), 1)
				ELSE 0
			END as processing_rate,
			-- 错误率
			CASE 
				WHEN st.processed_rows > 0 
				THEN (st.error_count::float / st.processed_rows::float) * 100
				ELSE 0
			END as error_rate,
			-- 完成率
			CASE 
				WHEN st.total_rows > 0 
				THEN (st.processed_rows::float / st.total_rows::float) * 100
				ELSE st.progress
			END as completion_rate
		FROM sync_tasks st
		INNER JOIN data_sources ds ON st.data_source_id = ds.id
		LEFT JOIN basic_libraries bl ON st.library_type = 'basic_library' AND st.library_id = bl.id
		LEFT JOIN thematic_libraries tl ON st.library_type = 'thematic_library' AND st.library_id = tl.id
		LEFT JOIN data_interfaces di ON st.interface_id = di.id;
		
		COMMENT ON VIEW sync_tasks_info IS '{
			"description": "同步任务详细信息视图：聚合同步任务基本信息及其关联的数据源、接口、基础库和专题库信息，支持多种库类型的统一管理",
			"fields": {
				"id": {"type": "string", "source": "sync_tasks.id", "description": "同步任务ID"},
				"library_type": {"type": "string", "source": "sync_tasks.library_type", "description": "库类型：basic_library（基础库）, thematic_library（专题库）"},
				"library_id": {"type": "string", "source": "sync_tasks.library_id", "description": "库ID：基础库ID或专题库ID"},
				"data_source_id": {"type": "string", "source": "sync_tasks.data_source_id", "description": "数据源ID"},
				"interface_id": {"type": "string | null", "source": "sync_tasks.interface_id", "description": "接口ID（可选）"},
				"task_type": {"type": "string", "source": "sync_tasks.task_type", "description": "任务类型：full_sync, incremental_sync, realtime_sync"},
				"status": {"type": "string", "source": "sync_tasks.status", "description": "任务状态：pending, running, success, failed, cancelled"},
				"start_time": {"type": "Date | null", "source": "sync_tasks.start_time", "description": "开始时间"},
				"end_time": {"type": "Date | null", "source": "sync_tasks.end_time", "description": "结束时间"},
				"progress": {"type": "number", "source": "sync_tasks.progress", "description": "进度百分比（0-100）"},
				"processed_rows": {"type": "number", "source": "sync_tasks.processed_rows", "description": "已处理行数"},
				"total_rows": {"type": "number", "source": "sync_tasks.total_rows", "description": "总行数"},
				"error_count": {"type": "number", "source": "sync_tasks.error_count", "description": "错误数量"},
				"error_message": {"type": "string", "source": "sync_tasks.error_message", "description": "错误信息"},
				"config": {"type": "Object", "source": "sync_tasks.config", "description": "同步配置"},
				"result": {"type": "Object", "source": "sync_tasks.result", "description": "同步结果"},
				"created_at": {"type": "Date", "source": "sync_tasks.created_at", "description": "创建时间"},
				"created_by": {"type": "string", "source": "sync_tasks.created_by", "description": "创建者"},
				"updated_at": {"type": "Date", "source": "sync_tasks.updated_at", "description": "更新时间"},
				"duration_seconds": {"type": "number | null", "description": "执行时长（秒）", "computed": true},
				"data_source": {
					"type": "Object",
					"source": "data_sources",
					"description": "关联数据源信息",
					"schema": {
						"id": {"type": "string", "description": "数据源ID"},
						"name": {"type": "string", "description": "数据源名称"},
						"type": {"type": "string", "description": "数据源类型"},
						"category": {"type": "string", "description": "数据源类别"},
						"connection_config": {"type": "Object", "description": "连接配置"},
						"params_config": {"type": "Object", "description": "参数配置"},
						"library_id": {"type": "string", "description": "基础库ID"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"basic_library": {
					"type": "Object | null",
					"source": "basic_libraries",
					"description": "关联基础库信息（当library_type为basic_library时）",
					"schema": {
						"id": {"type": "string", "description": "基础库ID"},
						"name_zh": {"type": "string", "description": "基础库中文名"},
						"name_en": {"type": "string", "description": "基础库英文名"},
						"description": {"type": "string", "description": "基础库描述"},
						"status": {"type": "string", "description": "基础库状态"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"thematic_library": {
					"type": "Object | null",
					"source": "thematic_libraries",
					"description": "关联专题库信息（当library_type为thematic_library时）",
					"schema": {
						"id": {"type": "string", "description": "专题库ID"},
						"name_zh": {"type": "string", "description": "专题库中文名"},
						"name_en": {"type": "string", "description": "专题库英文名"},
						"description": {"type": "string", "description": "专题库描述"},
						"category": {"type": "string", "description": "专题库类别：business, technical, analysis, report"},
						"domain": {"type": "string", "description": "专题库领域：user, order, product, finance, marketing"},
						"publish_status": {"type": "string", "description": "发布状态：draft, published, archived"},
						"version": {"type": "string", "description": "版本号"},
						"access_level": {"type": "string", "description": "访问级别：public, internal, private"},
						"update_frequency": {"type": "string", "description": "更新频率：realtime, hourly, daily, weekly, monthly"},
						"retention_period": {"type": "number", "description": "数据保留期（天）"},
						"status": {"type": "string", "description": "专题库状态"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"data_interface": {
					"type": "Object | null",
					"source": "data_interfaces",
					"description": "关联接口信息（可选）",
					"schema": {
						"id": {"type": "string", "description": "接口ID"},
						"name_zh": {"type": "string", "description": "接口中文名"},
						"name_en": {"type": "string", "description": "接口英文名"},
						"type": {"type": "string", "description": "接口类型"},
						"description": {"type": "string", "description": "接口描述"},
						"status": {"type": "string", "description": "接口状态"},
						"is_table_created": {"type": "boolean", "description": "是否已创建表"},
						"parse_config": {"type": "Object", "description": "解析配置"},
						"interface_config": {"type": "Object", "description": "接口配置"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"processing_rate": {"type": "number", "description": "处理速率（行/秒）", "computed": true},
				"error_rate": {"type": "number", "description": "错误率（百分比）", "computed": true},
				"completion_rate": {"type": "number", "description": "完成率（百分比）", "computed": true}
			}
		}';
	`,
}
