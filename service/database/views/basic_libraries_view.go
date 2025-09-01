/*
 * @module service/database/views/basic_libraries_view
 * @description 数据基础库相关视图定义，提供基础库、接口、数据源等实体的聚合查询视图
 * @architecture 数据库视图层 - 基于PostgreSQL视图实现数据聚合
 * @documentReference docs/database_design.md
 * @stateFlow 基础库数据生命周期视图管理
 * @rules 遵循PostgreSQL视图设计规范，使用json_agg聚合关联数据，确保数据完整性
 * @dependencies PostgreSQL JSONB支持, GORM模型定义
 * @refs service/models/basic_library.go, docs/api_design.md
 */

package views

var BasicLibraryViews = map[string]string{
	// 基础库信息视图 - 包含基础库的完整基本信息和关联数据
	"basic_libraries_info": `
		DROP VIEW IF EXISTS basic_libraries_info;
		CREATE VIEW basic_libraries_info AS
		SELECT 
			bl.id,
			bl.name_zh,
			bl.name_en,
			bl.description,
			bl.status,
			bl.created_at,
			bl.created_by,
			bl.updated_at,
			bl.updated_by,
			-- 接口信息JSON数组，来源：data_interfaces表
			-- 包含字段：id, name_zh, name_en, type, description, status, data_source_id, is_table_created, parse_config, created_at, created_by, updated_at, updated_by
			COALESCE(
				json_agg(
					DISTINCT jsonb_build_object(
						'id', di.id,
						'name_zh', di.name_zh,
						'name_en', di.name_en,
						'type', di.type,
						'description', di.description,
						'status', di.status,
						'data_source_id', di.data_source_id,
						'is_table_created', di.is_table_created,
						'parse_config', di.parse_config,
						'created_at', di.created_at,
						'created_by', di.created_by,
						'updated_at', di.updated_at,
						'updated_by', di.updated_by
					)
				) FILTER (WHERE di.id IS NOT NULL),
				'[]'::json
			) as interfaces,
			-- 数据源信息JSON数组，来源：data_sources表
			-- 包含字段：id, name, type, category, connection_config, params_config, created_at, created_by, updated_at, updated_by
			COALESCE(
				json_agg(
					DISTINCT jsonb_build_object(
						'id', ds.id,
						'name', ds.name,
						'type', ds.type,
						'category', ds.category,
						'connection_config', ds.connection_config,
						'params_config', ds.params_config,
						'created_at', ds.created_at,
						'created_by', ds.created_by,
						'updated_at', ds.updated_at,
						'updated_by', ds.updated_by
					)
				) FILTER (WHERE ds.id IS NOT NULL),
				'[]'::json
			) as data_sources,
			-- 统计信息
			COUNT(DISTINCT di.id) as interface_count,
			COUNT(DISTINCT ds.id) as data_source_count,
			COUNT(DISTINCT CASE WHEN di.status = 'active' THEN di.id END) as active_interface_count,
			COUNT(DISTINCT CASE WHEN di.is_table_created = true THEN di.id END) as table_created_interface_count,
			COUNT(DISTINCT CASE WHEN ds.category = 'messaging' THEN ds.id END) as messaging_source_count,
			COUNT(DISTINCT CASE WHEN ds.type = 'db' THEN ds.id END) as db_source_count,
			COUNT(DISTINCT CASE WHEN ds.type = 'http' THEN ds.id END) as http_source_count,
			COUNT(DISTINCT CASE WHEN ds.type = 'file' THEN ds.id END) as file_source_count,
			COUNT(DISTINCT CASE WHEN ds.type = 'stream' THEN ds.id END) as stream_source_count
		FROM basic_libraries bl
		LEFT JOIN data_interfaces di ON bl.id = di.library_id
		LEFT JOIN data_sources ds ON bl.id = ds.library_id
		GROUP BY bl.id, bl.name_zh, bl.name_en, bl.description, bl.status, 
				bl.created_at, bl.created_by, bl.updated_at, bl.updated_by;
		
		COMMENT ON VIEW basic_libraries_info IS '{
			"description": "基础库信息视图：聚合基础库基本信息、关联的数据接口和数据源信息",
			"fields": {
				"id": {"type": "string", "source": "basic_libraries.id", "description": "基础库ID"},
				"name_zh": {"type": "string", "source": "basic_libraries.name_zh", "description": "中文名称"},
				"name_en": {"type": "string", "source": "basic_libraries.name_en", "description": "英文名称"},
				"description": {"type": "string", "source": "basic_libraries.description", "description": "描述"},
				"status": {"type": "string", "source": "basic_libraries.status", "description": "状态"},
				"created_at": {"type": "Date", "source": "basic_libraries.created_at", "description": "创建时间"},
				"created_by": {"type": "string", "source": "basic_libraries.created_by", "description": "创建者"},
				"updated_at": {"type": "Date", "source": "basic_libraries.updated_at", "description": "更新时间"},
				"updated_by": {"type": "string", "source": "basic_libraries.updated_by", "description": "更新者"},
				"interfaces": {
					"type": "Array<Object>",
					"source": "data_interfaces",
					"description": "关联接口列表",
					"schema": {
						"id": {"type": "string", "description": "接口ID"},
						"name_zh": {"type": "string", "description": "接口中文名"},
						"name_en": {"type": "string", "description": "接口英文名"},
						"type": {"type": "string", "description": "接口类型"},
						"description": {"type": "string", "description": "接口描述"},
						"status": {"type": "string", "description": "接口状态"},
						"data_source_id": {"type": "string", "description": "数据源ID"},
						"is_table_created": {"type": "boolean", "description": "是否已创建表"},
						"parse_config": {"type": "Object", "description": "解析配置"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"data_sources": {
					"type": "Array<Object>",
					"source": "data_sources",
					"description": "关联数据源列表",
					"schema": {
						"id": {"type": "string", "description": "数据源ID"},
						"name": {"type": "string", "description": "数据源名称"},
						"type": {"type": "string", "description": "数据源类型"},
						"category": {"type": "string", "description": "数据源类别"},
						"connection_config": {"type": "Object", "description": "连接配置"},
						"params_config": {"type": "Object", "description": "参数配置"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"interface_count": {"type": "number", "description": "接口总数"},
				"data_source_count": {"type": "number", "description": "数据源总数"},
				"active_interface_count": {"type": "number", "description": "活跃接口数"},
				"table_created_interface_count": {"type": "number", "description": "已创建表的接口数"},
				"kafka_source_count": {"type": "number", "description": "Kafka数据源数"},
				"db_source_count": {"type": "number", "description": "数据库数据源数"},
				"http_source_count": {"type": "number", "description": "HTTP数据源数"},
				"file_source_count": {"type": "number", "description": "文件数据源数"},
				"stream_source_count": {"type": "number", "description": "流数据源数"}
			}
		}';
	`,

	// 数据接口详细信息视图 - 包含接口的所有字段和关联信息
	"data_interfaces_info": `
		DROP VIEW IF EXISTS data_interfaces_info;
		CREATE VIEW data_interfaces_info AS
		SELECT 
			di.id,
			di.library_id,
			di.name_zh,
			di.name_en,
			di.type,
			di.description,
			di.created_at,
			di.created_by,
			di.updated_at,
			di.updated_by,
			di.status,
			di.data_source_id,
			di.is_table_created,
			di.interface_config,
			di.parse_config,
			di.table_fields_config,
			-- 基础库信息对象，来源：basic_libraries表
			-- 包含字段：id, name_zh, name_en, description, status, created_at, created_by, updated_at, updated_by
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
			) as library,
			-- 数据源信息对象，来源：data_sources表
			-- 包含字段：id, name, type, category, connection_config, params_config, created_at, created_by, updated_at, updated_by
			CASE 
				WHEN ds.id IS NOT NULL THEN
					jsonb_build_object(
						'id', ds.id,
						'name', ds.name,
						'type', ds.type,
						'category', ds.category,
						'connection_config', ds.connection_config,
						'params_config', ds.params_config,
						'created_at', ds.created_at,
						'created_by', ds.created_by,
						'updated_at', ds.updated_at,
						'updated_by', ds.updated_by
					)
				ELSE NULL
			END as data_source,
			-- 清洗规则JSON数组，来源：cleansing_rules表（按order_num排序）
			-- 包含字段：id, type, config, order_num, is_enabled, created_at, created_by
			COALESCE(
				json_agg(
					jsonb_build_object(
						'id', cr.id,
						'type', cr.type,
						'config', cr.config,
						'order_num', cr.order_num,
						'is_enabled', cr.is_enabled,
						'created_at', cr.created_at,
						'created_by', cr.created_by
					) ORDER BY cr.order_num
				) FILTER (WHERE cr.id IS NOT NULL),
				'[]'::json
			) as cleansing_rules,
			-- 统计信息
			COUNT(DISTINCT cr.id) as cleansing_rule_count,
			COUNT(DISTINCT CASE WHEN cr.is_enabled = true THEN cr.id END) as active_rule_count
		FROM data_interfaces di
		LEFT JOIN basic_libraries bl ON di.library_id = bl.id
		LEFT JOIN data_sources ds ON di.data_source_id = ds.id
		LEFT JOIN cleansing_rules cr ON di.id = cr.interface_id
		GROUP BY di.id, di.library_id, di.name_zh, di.name_en, di.type, di.description, 
				di.created_at, di.created_by, di.updated_at, di.updated_by, di.status, di.data_source_id, di.is_table_created, di.parse_config, di.interface_config,
				bl.id, bl.name_zh, bl.name_en, bl.description, bl.status, bl.created_at, bl.created_by, bl.updated_at, bl.updated_by,
				ds.id, ds.type, ds.category, ds.connection_config, ds.params_config, ds.created_at, ds.created_by, ds.updated_at, ds.updated_by;
		
		COMMENT ON VIEW data_interfaces_info IS '{
			"description": "数据接口详细信息视图：聚合接口基本信息及其关联的基础库、数据源、字段和清洗规则",
			"fields": {
				"id": {"type": "string", "source": "data_interfaces.id", "description": "接口ID"},
				"library_id": {"type": "string", "source": "data_interfaces.library_id", "description": "基础库ID"},
				"name_zh": {"type": "string", "source": "data_interfaces.name_zh", "description": "接口中文名"},
				"name_en": {"type": "string", "source": "data_interfaces.name_en", "description": "接口英文名"},
				"type": {"type": "string", "source": "data_interfaces.type", "description": "接口类型"},
				"description": {"type": "string", "source": "data_interfaces.description", "description": "接口描述"},
				"created_at": {"type": "Date", "source": "data_interfaces.created_at", "description": "接口创建时间"},
				"created_by": {"type": "string", "source": "data_interfaces.created_by", "description": "接口创建者"},
				"updated_at": {"type": "Date", "source": "data_interfaces.updated_at", "description": "接口更新时间"},
				"updated_by": {"type": "string", "source": "data_interfaces.updated_by", "description": "接口更新者"},
				"status": {"type": "string", "source": "data_interfaces.status", "description": "接口状态"},
				"data_source_id": {"type": "string", "source": "data_interfaces.data_source_id", "description": "数据源ID"},
				"is_table_created": {"type": "boolean", "source": "data_interfaces.is_table_created", "description": "是否已创建表"},
				"parse_config": {"type": "Object", "source": "data_interfaces.parse_config", "description": "解析配置"},
				"interface_config": {"type": "Object", "source": "data_interfaces.interface_config", "description": "接口配置"},
				"table_fields_config": {"type": "Object", "source": "data_interfaces.table_fields_config", "description": "表字段配置"},
				"library": {
					"type": "Object",
					"source": "basic_libraries",
					"description": "关联基础库信息",
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
				"data_source": {
					"type": "Object | null",
					"source": "data_sources",
					"description": "关联数据源信息",
					"schema": {
						"id": {"type": "string", "description": "数据源ID"},
						"name": {"type": "string", "description": "数据源名称"},
						"type": {"type": "string", "description": "数据源类型"},
						"connection_config": {"type": "Object", "description": "连接配置"},
						"params_config": {"type": "Object", "description": "参数配置"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"},
						"updated_at": {"type": "Date", "description": "更新时间"},
						"updated_by": {"type": "string", "description": "更新者"}
					}
				},
				"cleansing_rules": {
					"type": "Array<Object>",
					"source": "cleansing_rules",
					"description": "数据清洗规则列表",
					"schema": {
						"id": {"type": "string", "description": "规则ID"},
						"type": {"type": "string", "description": "规则类型"},
						"config": {"type": "Object", "description": "规则配置"},
						"order_num": {"type": "number", "description": "排序号"},
						"is_enabled": {"type": "boolean", "description": "是否启用"},
						"created_at": {"type": "Date", "description": "创建时间"},
						"created_by": {"type": "string", "description": "创建者"}
					}
				},
				"cleansing_rule_count": {"type": "number", "description": "清洗规则总数"},
				"active_rule_count": {"type": "number", "description": "活跃规则数"}
			}
		}';
	`,

	// 数据源详细信息视图 - 包含数据源的所有字段和关联信息
	"data_sources_info": `
		DROP VIEW IF EXISTS data_sources_info;
		CREATE VIEW data_sources_info AS
		SELECT 
			ds.id,
			ds.library_id,
			ds.name,
			ds.type,
			ds.category,
			ds.connection_config,
			ds.params_config,
			ds.created_at,
			ds.created_by,
			ds.updated_at,
			ds.updated_by,
			-- 基础库信息对象，来源：basic_libraries表
			-- 包含字段：id, name_zh, name_en, description, status, created_at, created_by, updated_at, updated_by
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
			) as library,
			-- 关联接口JSON数组，来源：data_interfaces表
			-- 包含字段：id, name_zh, name_en, type, description, status, is_table_created, parse_config, created_at, created_by, updated_at, updated_by
			COALESCE(
				json_agg(
					DISTINCT jsonb_build_object(
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
				) FILTER (WHERE di.id IS NOT NULL),
				'[]'::json
			) as related_interfaces,
			-- 统计信息
			COUNT(DISTINCT di.id) as related_interface_count,
			COUNT(DISTINCT CASE WHEN di.status = 'active' THEN di.id END) as active_interface_count,
			COUNT(DISTINCT CASE WHEN di.is_table_created = true THEN di.id END) as table_created_interface_count
		FROM data_sources ds
		LEFT JOIN basic_libraries bl ON ds.library_id = bl.id
		LEFT JOIN data_interfaces di ON ds.id = di.data_source_id
		GROUP BY ds.id, ds.library_id, ds.name, ds.type, ds.category, ds.connection_config, ds.params_config,
				ds.created_at, ds.created_by, ds.updated_at, ds.updated_by,
				bl.id, bl.name_zh, bl.name_en, bl.description, bl.status, bl.created_at, bl.created_by, bl.updated_at, bl.updated_by;
		
		COMMENT ON VIEW data_sources_info IS '{
			"description": "数据源详细信息视图：聚合数据源基本信息及其关联的基础库和接口信息",
			"fields": {
				"id": {"type": "string", "source": "data_sources.id", "description": "数据源ID"},
				"library_id": {"type": "string", "source": "data_sources.library_id", "description": "基础库ID"},
				"name": {"type": "string", "source": "data_sources.name", "description": "数据源名称"},
				"type": {"type": "string", "source": "data_sources.type", "description": "数据源类型"},
				"category": {"type": "string", "source": "data_sources.category", "description": "数据源类别"},
				"connection_config": {"type": "Object", "source": "data_sources.connection_config", "description": "连接配置"},
				"params_config": {"type": "Object", "source": "data_sources.params_config", "description": "参数配置"},
				"created_at": {"type": "Date", "source": "data_sources.created_at", "description": "数据源创建时间"},
				"created_by": {"type": "string", "source": "data_sources.created_by", "description": "数据源创建者"},
				"updated_at": {"type": "Date", "source": "data_sources.updated_at", "description": "数据源更新时间"},
				"updated_by": {"type": "string", "source": "data_sources.updated_by", "description": "数据源更新者"},
				"library": {
					"type": "Object",
					"source": "basic_libraries",
					"description": "关联基础库信息",
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
				"related_interfaces": {
					"type": "Array<Object>",
					"source": "data_interfaces",
					"description": "关联接口列表",
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
				"related_interface_count": {"type": "number", "description": "关联接口数量"},
				"active_interface_count": {"type": "number", "description": "活跃接口数量"},
				"table_created_interface_count": {"type": "number", "description": "已创建表的接口数量"}
			}
		}';
	`,

}
