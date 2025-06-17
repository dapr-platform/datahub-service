/*
 * @module service/database/views/thematic_libraries_view
 * @description 数据主题库相关视图定义，提供主题库、接口、字段、流程图等实体的聚合查询视图
 * @architecture 数据库视图层 - 基于PostgreSQL视图实现数据聚合
 * @documentReference docs/database_design.md
 * @stateFlow 主题库数据生命周期视图管理
 * @rules 遵循PostgreSQL视图设计规范，使用json_agg聚合关联数据，确保数据完整性
 * @dependencies PostgreSQL JSONB支持, GORM模型定义
 * @refs service/models/thematic_library.go, docs/api_design.md
 */

package views

var ThematicLibraryViews = map[string]string{
	// 主题库信息视图 - 包含主题库的完整基本信息和关联数据
	"thematic_libraries_info": `
		DROP VIEW IF EXISTS thematic_libraries_info;
		CREATE VIEW thematic_libraries_info AS
		SELECT 
			tl.id,
			tl.name_zh,
			tl.name_en,
			tl.category,
			tl.domain,
			tl.description,
			tl.tags,
			tl.source_libraries,
			tl.publish_status,
			tl.version,
			tl.access_level,
			tl.authorized_users,
			tl.authorized_roles,
			tl.update_frequency,
			tl.retention_period,
			tl.created_at,
			tl.created_by,
			tl.updated_at,
			tl.updated_by,
			tl.status,
			-- 主题接口信息JSON数组，来源：thematic_interfaces表
			-- 包含字段：id, name_zh, name_en, type, description, status, is_table_created, interface_config, parse_config, created_at, created_by, updated_at, updated_by
			COALESCE(
				json_agg(
					jsonb_build_object(
						'id', ti.id,
						'name_zh', ti.name_zh,
						'name_en', ti.name_en,
						'type', ti.type,
						'description', ti.description,
						'status', ti.status,
						'is_table_created', ti.is_table_created,
						'interface_config', ti.interface_config,
						'parse_config', ti.parse_config,
						'created_at', ti.created_at,
						'created_by', ti.created_by,
						'updated_at', ti.updated_at,
						'updated_by', ti.updated_by
					)
				) FILTER (WHERE ti.id IS NOT NULL),
				'[]'::json
			) as thematic_interfaces,
			-- 数据流程图信息JSON数组，来源：data_flow_graphs表
			-- 包含字段：id, thematic_interface_id, name, description, definition, status, created_at, created_by, updated_at, updated_by
			COALESCE(
				json_agg(
					jsonb_build_object(
						'id', dfg.id,
						'thematic_interface_id', dfg.thematic_interface_id,
						'name', dfg.name,
						'description', dfg.description,
						'definition', dfg.definition,
						'status', dfg.status,
						'created_at', dfg.created_at,
						'created_by', dfg.created_by,
						'updated_at', dfg.updated_at,
						'updated_by', dfg.updated_by
					)
				) FILTER (WHERE dfg.id IS NOT NULL),
				'[]'::json
			) as data_flow_graphs,
			-- 流程图节点信息JSON数组，来源：flow_nodes表
			-- 包含字段：id, flow_graph_id, type, config, position_x, position_y, name, created_at, created_by
			COALESCE(
				json_agg(
					jsonb_build_object(
						'id', fn.id,
						'flow_graph_id', fn.flow_graph_id,
						'type', fn.type,
						'config', fn.config,
						'position_x', fn.position_x,
						'position_y', fn.position_y,
						'name', fn.name,
						'created_at', fn.created_at,
						'created_by', fn.created_by
					)
				) FILTER (WHERE fn.id IS NOT NULL),
				'[]'::json
			) as flow_nodes,
			-- 统计信息
			COUNT(DISTINCT ti.id) as interface_count,
			COUNT(DISTINCT dfg.id) as flow_graph_count,
			COUNT(DISTINCT fn.id) as flow_node_count,
			COUNT(DISTINCT CASE WHEN ti.status = 'active' THEN ti.id END) as active_interface_count,
			COUNT(DISTINCT CASE WHEN ti.is_table_created = true THEN ti.id END) as table_created_interface_count,
			COUNT(DISTINCT CASE WHEN dfg.status = 'active' THEN dfg.id END) as active_flow_graph_count
		FROM thematic_libraries tl
		LEFT JOIN thematic_interfaces ti ON tl.id = ti.library_id
		LEFT JOIN data_flow_graphs dfg ON ti.id = dfg.thematic_interface_id
		LEFT JOIN flow_nodes fn ON dfg.id = fn.flow_graph_id
		GROUP BY tl.id, tl.name_zh, tl.name_en, tl.category, tl.domain, tl.description, 
				tl.tags, tl.source_libraries, tl.publish_status, tl.version, 
				tl.access_level, tl.authorized_users, tl.authorized_roles, tl.update_frequency, 
				tl.retention_period, tl.created_at, tl.created_by, tl.updated_at, tl.updated_by, tl.status;
		
		COMMENT ON VIEW thematic_libraries_info IS '{
			"description": "主题库信息视图：聚合主题库基本信息、关联的主题接口、字段、数据流程图和节点信息",
			"fields": {
				"id": {"type": "string", "source": "thematic_libraries.id", "description": "主题库ID"},
				"name_zh": {"type": "string", "source": "thematic_libraries.name_zh", "description": "主题库中文名称"},
				"name_en": {"type": "string", "source": "thematic_libraries.name_en", "description": "主题库英文名称"},
				"category": {"type": "string", "source": "thematic_libraries.category", "description": "主题库类别：business, technical, analysis, report"},
				"domain": {"type": "string", "source": "thematic_libraries.domain", "description": "业务域：user, order, product, finance, marketing"},
				"description": {"type": "string", "source": "thematic_libraries.description", "description": "主题库描述"},
				"tags": {"type": "Array<string>", "source": "thematic_libraries.tags", "description": "标签列表"},
				"source_libraries": {"type": "Array<string>", "source": "thematic_libraries.source_libraries", "description": "源基础库列表"},
				"publish_status": {"type": "string", "source": "thematic_libraries.publish_status", "description": "发布状态：draft, published, archived"},
				"version": {"type": "string", "source": "thematic_libraries.version", "description": "版本号"},
				"access_level": {"type": "string", "source": "thematic_libraries.access_level", "description": "访问级别：public, internal, private"},
				"authorized_users": {"type": "Array<string>", "source": "thematic_libraries.authorized_users", "description": "授权用户列表"},
				"authorized_roles": {"type": "Array<string>", "source": "thematic_libraries.authorized_roles", "description": "授权角色列表"},
				"update_frequency": {"type": "string", "source": "thematic_libraries.update_frequency", "description": "更新频率：realtime, hourly, daily, weekly, monthly"},
				"retention_period": {"type": "number", "source": "thematic_libraries.retention_period", "description": "数据保留期（天）"},
				"created_at": {"type": "Date", "source": "thematic_libraries.created_at", "description": "创建时间"},
				"created_by": {"type": "string", "source": "thematic_libraries.created_by", "description": "创建者"},
				"updated_at": {"type": "Date", "source": "thematic_libraries.updated_at", "description": "更新时间"},
				"updated_by": {"type": "string", "source": "thematic_libraries.updated_by", "description": "更新者"},
				"status": {"type": "string", "source": "thematic_libraries.status", "description": "状态"},
				"thematic_interfaces": {
					"type": "Array<Object>",
					"source": "thematic_interfaces",
					"description": "关联主题接口列表",
					"schema": {
						"id": {"type": "string", "source": "thematic_interfaces.id", "description": "接口ID"},
						"name_zh": {"type": "string", "source": "thematic_interfaces.name_zh", "description": "接口中文名"},
						"name_en": {"type": "string", "source": "thematic_interfaces.name_en", "description": "接口英文名"},
						"type": {"type": "string", "source": "thematic_interfaces.type", "description": "接口类型：realtime, batch"},
						"description": {"type": "string", "source": "thematic_interfaces.description", "description": "接口描述"},
						"status": {"type": "string", "source": "thematic_interfaces.status", "description": "接口状态"},
						"is_table_created": {"type": "boolean", "source": "thematic_interfaces.is_table_created", "description": "是否已创建表"},
						"interface_config": {"type": "Object", "source": "thematic_interfaces.interface_config", "description": "接口配置"},
						"parse_config": {"type": "Object", "source": "thematic_interfaces.parse_config", "description": "解析配置"},
						"created_at": {"type": "Date", "source": "thematic_interfaces.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "thematic_interfaces.created_by", "description": "创建者"},
						"updated_at": {"type": "Date", "source": "thematic_interfaces.updated_at", "description": "更新时间"},
						"updated_by": {"type": "string", "source": "thematic_interfaces.updated_by", "description": "更新者"}
					}
				},
				"data_flow_graphs": {
					"type": "Array<Object>",
					"source": "data_flow_graphs",
					"description": "数据流程图列表",
					"schema": {
						"id": {"type": "string", "source": "data_flow_graphs.id", "description": "流程图ID"},
						"thematic_interface_id": {"type": "string", "source": "data_flow_graphs.thematic_interface_id", "description": "主题接口ID"},
						"name": {"type": "string", "source": "data_flow_graphs.name", "description": "流程图名称"},
						"description": {"type": "string", "source": "data_flow_graphs.description", "description": "流程图描述"},
						"definition": {"type": "Object", "source": "data_flow_graphs.definition", "description": "流程图定义"},
						"status": {"type": "string", "source": "data_flow_graphs.status", "description": "流程图状态：draft, active, inactive"},
						"created_at": {"type": "Date", "source": "data_flow_graphs.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "data_flow_graphs.created_by", "description": "创建者"},
						"updated_at": {"type": "Date", "source": "data_flow_graphs.updated_at", "description": "更新时间"},
						"updated_by": {"type": "string", "source": "data_flow_graphs.updated_by", "description": "更新者"}
					}
				},
				"flow_nodes": {
					"type": "Array<Object>",
					"source": "flow_nodes",
					"description": "流程图节点列表",
					"schema": {
						"id": {"type": "string", "source": "flow_nodes.id", "description": "节点ID"},
						"flow_graph_id": {"type": "string", "source": "flow_nodes.flow_graph_id", "description": "流程图ID"},
						"type": {"type": "string", "source": "flow_nodes.type", "description": "节点类型：datasource, api, file, filter, transform, aggregate, output"},
						"config": {"type": "Object", "source": "flow_nodes.config", "description": "节点配置"},
						"position_x": {"type": "number", "source": "flow_nodes.position_x", "description": "X坐标"},
						"position_y": {"type": "number", "source": "flow_nodes.position_y", "description": "Y坐标"},
						"name": {"type": "string", "source": "flow_nodes.name", "description": "节点名称"},
						"created_at": {"type": "Date", "source": "flow_nodes.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "flow_nodes.created_by", "description": "创建者"}
					}
				},
				"interface_count": {"type": "number", "description": "接口总数", "computed": true},
				"field_count": {"type": "number", "description": "字段总数", "computed": true},
				"flow_graph_count": {"type": "number", "description": "流程图总数", "computed": true},
				"flow_node_count": {"type": "number", "description": "流程图节点总数", "computed": true},
				"active_interface_count": {"type": "number", "description": "活跃接口数", "computed": true},
				"table_created_interface_count": {"type": "number", "description": "已创建表的接口数", "computed": true},
				"active_flow_graph_count": {"type": "number", "description": "活跃流程图数", "computed": true},
				"primary_key_field_count": {"type": "number", "description": "主键字段数", "computed": true},
				"unique_field_count": {"type": "number", "description": "唯一字段数", "computed": true},
				"not_null_field_count": {"type": "number", "description": "非空字段数", "computed": true}
			}
		}';
	`,

	// 主题接口详细信息视图 - 包含接口的所有字段和关联信息
	"thematic_interfaces_info": `
		DROP VIEW IF EXISTS thematic_interfaces_info;
		CREATE VIEW thematic_interfaces_info AS
		SELECT 
			ti.id,
			ti.library_id,
			ti.name_zh,
			ti.name_en,
			ti.type,
			ti.description,
			ti.created_at,
			ti.created_by,
			ti.updated_at,
			ti.updated_by,
			ti.status,
			ti.is_table_created,
			ti.interface_config,
			ti.parse_config,
			ti.table_fields_config,
			-- 主题库信息对象，来源：thematic_libraries表
			-- 包含字段：id, name_zh, name_en, category, domain, description, tags, source_libraries, publish_status, version, access_level, authorized_users, authorized_roles, update_frequency, retention_period, created_at, created_by, updated_at, updated_by, status
			jsonb_build_object(
				'id', tl.id,
				'name_zh', tl.name_zh,
				'name_en', tl.name_en,
				'category', tl.category,
				'domain', tl.domain,
				'description', tl.description,
				'tags', tl.tags,
				'source_libraries', tl.source_libraries,
				'publish_status', tl.publish_status,
				'version', tl.version,
				'access_level', tl.access_level,
				'authorized_users', tl.authorized_users,
				'authorized_roles', tl.authorized_roles,
				'update_frequency', tl.update_frequency,
				'retention_period', tl.retention_period,
				'created_at', tl.created_at,
				'created_by', tl.created_by,
				'updated_at', tl.updated_at,
				'updated_by', tl.updated_by,
				'status', tl.status
			) as thematic_library,
			-- 数据流程图信息JSON数组，来源：data_flow_graphs表
			-- 包含字段：id, thematic_interface_id, name, description, definition, created_at, created_by, updated_at, updated_by, status
			COALESCE(
				json_agg(
					jsonb_build_object(
						'id', dfg.id,
						'thematic_interface_id', dfg.thematic_interface_id,
						'name', dfg.name,
						'description', dfg.description,
						'definition', dfg.definition,
						'created_at', dfg.created_at,
						'created_by', dfg.created_by,
						'updated_at', dfg.updated_at,
						'updated_by', dfg.updated_by,
						'status', dfg.status
					)
				) FILTER (WHERE dfg.id IS NOT NULL),
				'[]'::json
			) as data_flow_graphs,
			-- 流程图节点信息JSON数组，来源：flow_nodes表
			-- 包含字段：id, flow_graph_id, type, config, position_x, position_y, name, created_at, created_by
			COALESCE(
				json_agg(
					jsonb_build_object(
						'id', fn.id,
						'flow_graph_id', fn.flow_graph_id,
						'type', fn.type,
						'config', fn.config,
						'position_x', fn.position_x,
						'position_y', fn.position_y,
						'name', fn.name,
						'created_at', fn.created_at,
						'created_by', fn.created_by
					)
				) FILTER (WHERE fn.id IS NOT NULL),
				'[]'::json
			) as flow_nodes,
			-- 统计信息
			COUNT(DISTINCT dfg.id) as flow_graph_count,
			COUNT(DISTINCT fn.id) as flow_node_count,
			COUNT(DISTINCT CASE WHEN dfg.status = 'active' THEN dfg.id END) as active_flow_graph_count,
			COUNT(DISTINCT CASE WHEN dfg.status = 'draft' THEN dfg.id END) as draft_flow_graph_count,
			COUNT(DISTINCT CASE WHEN dfg.status = 'inactive' THEN dfg.id END) as inactive_flow_graph_count
		FROM thematic_interfaces ti
		LEFT JOIN thematic_libraries tl ON ti.library_id = tl.id
		LEFT JOIN data_flow_graphs dfg ON ti.id = dfg.thematic_interface_id
		LEFT JOIN flow_nodes fn ON dfg.id = fn.flow_graph_id
		GROUP BY ti.id, ti.library_id, ti.name_zh, ti.name_en, ti.type, ti.description, 
				ti.created_at, ti.created_by, ti.updated_at, ti.updated_by, ti.status, 
				ti.is_table_created, ti.interface_config, ti.parse_config, ti.table_fields_config,
				tl.id, tl.name_zh, tl.name_en, tl.category, tl.domain, tl.description, 
				tl.tags, tl.source_libraries, tl.publish_status, tl.version, 
				tl.access_level, tl.authorized_users, tl.authorized_roles, tl.update_frequency, 
				tl.retention_period, tl.created_at, tl.created_by, tl.updated_at, tl.updated_by, tl.status;
		
		COMMENT ON VIEW thematic_interfaces_info IS '{
			"description": "主题接口详细信息视图：聚合主题接口基本信息及其关联的主题库、字段、数据流程图和节点信息",
			"fields": {
				"id": {"type": "string", "source": "thematic_interfaces.id", "description": "主题接口ID"},
				"library_id": {"type": "string", "source": "thematic_interfaces.library_id", "description": "主题库ID"},
				"name_zh": {"type": "string", "source": "thematic_interfaces.name_zh", "description": "接口中文名"},
				"name_en": {"type": "string", "source": "thematic_interfaces.name_en", "description": "接口英文名"},
				"type": {"type": "string", "source": "thematic_interfaces.type", "description": "接口类型：realtime, batch"},
				"description": {"type": "string", "source": "thematic_interfaces.description", "description": "接口描述"},
				"created_at": {"type": "Date", "source": "thematic_interfaces.created_at", "description": "接口创建时间"},
				"created_by": {"type": "string", "source": "thematic_interfaces.created_by", "description": "接口创建者"},
				"updated_at": {"type": "Date", "source": "thematic_interfaces.updated_at", "description": "接口更新时间"},
				"updated_by": {"type": "string", "source": "thematic_interfaces.updated_by", "description": "接口更新者"},
				"status": {"type": "string", "source": "thematic_interfaces.status", "description": "接口状态"},
				"is_table_created": {"type": "boolean", "source": "thematic_interfaces.is_table_created", "description": "是否已创建表"},
				"interface_config": {"type": "Object", "source": "thematic_interfaces.interface_config", "description": "接口配置"},
				"parse_config": {"type": "Object", "source": "thematic_interfaces.parse_config", "description": "解析配置"},
				"thematic_library": {
					"type": "Object",
					"source": "thematic_libraries",
					"description": "关联主题库信息",
					"schema": {
						"id": {"type": "string", "source": "thematic_libraries.id", "description": "主题库ID"},
						"name_zh": {"type": "string", "source": "thematic_libraries.name_zh", "description": "主题库中文名"},
						"name_en": {"type": "string", "source": "thematic_libraries.name_en", "description": "主题库英文名"},
						"category": {"type": "string", "source": "thematic_libraries.category", "description": "主题库类别：business, technical, analysis, report"},
						"domain": {"type": "string", "source": "thematic_libraries.domain", "description": "业务域：user, order, product, finance, marketing"},
						"description": {"type": "string", "source": "thematic_libraries.description", "description": "主题库描述"},
						"tags": {"type": "Array<string>", "source": "thematic_libraries.tags", "description": "标签列表"},
						"source_libraries": {"type": "Array<string>", "source": "thematic_libraries.source_libraries", "description": "源基础库列表"},
						"publish_status": {"type": "string", "source": "thematic_libraries.publish_status", "description": "发布状态：draft, published, archived"},
						"version": {"type": "string", "source": "thematic_libraries.version", "description": "版本号"},
						"access_level": {"type": "string", "source": "thematic_libraries.access_level", "description": "访问级别：public, internal, private"},
						"authorized_users": {"type": "Array<string>", "source": "thematic_libraries.authorized_users", "description": "授权用户列表"},
						"authorized_roles": {"type": "Array<string>", "source": "thematic_libraries.authorized_roles", "description": "授权角色列表"},
						"update_frequency": {"type": "string", "source": "thematic_libraries.update_frequency", "description": "更新频率：realtime, hourly, daily, weekly, monthly"},
						"retention_period": {"type": "number", "source": "thematic_libraries.retention_period", "description": "数据保留期（天）"},
						"created_at": {"type": "Date", "source": "thematic_libraries.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "thematic_libraries.created_by", "description": "创建者"},
						"updated_at": {"type": "Date", "source": "thematic_libraries.updated_at", "description": "更新时间"},
						"updated_by": {"type": "string", "source": "thematic_libraries.updated_by", "description": "更新者"},
						"status": {"type": "string", "source": "thematic_libraries.status", "description": "状态"}
					}
				},
				"table_fields_config": {
					"type": "Object",
					"source": "thematic_interfaces.table_fields_config",
					"description": "表字段配置",
					"schema": {
						"id": {"type": "string", "source": "thematic_interfaces.table_fields_config.id", "description": "字段ID"},
						"interface_id": {"type": "string", "source": "thematic_interface_fields.interface_id", "description": "接口ID"},
						"name_zh": {"type": "string", "source": "thematic_interface_fields.name_zh", "description": "字段中文名"},
						"name_en": {"type": "string", "source": "thematic_interface_fields.name_en", "description": "字段英文名"},
						"data_type": {"type": "string", "source": "thematic_interface_fields.data_type", "description": "数据类型"},
						"is_primary_key": {"type": "boolean", "source": "thematic_interface_fields.is_primary_key", "description": "是否主键"},
						"is_unique": {"type": "boolean", "source": "thematic_interface_fields.is_unique", "description": "是否唯一"},
						"is_nullable": {"type": "boolean", "source": "thematic_interface_fields.is_nullable", "description": "是否可空"},
						"default_value": {"type": "string", "source": "thematic_interface_fields.default_value", "description": "默认值"},
						"description": {"type": "string", "source": "thematic_interface_fields.description", "description": "字段描述"},
						"order_num": {"type": "number", "source": "thematic_interface_fields.order_num", "description": "排序号"},
						"check_constraint": {"type": "string", "source": "thematic_interface_fields.check_constraint", "description": "检查约束"},
						"created_at": {"type": "Date", "source": "thematic_interface_fields.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "thematic_interface_fields.created_by", "description": "创建者"},
						"updated_at": {"type": "Date", "source": "thematic_interface_fields.updated_at", "description": "更新时间"},
						"updated_by": {"type": "string", "source": "thematic_interface_fields.updated_by", "description": "更新者"}
					}
				},
				"data_flow_graphs": {
					"type": "Array<Object>",
					"source": "data_flow_graphs",
					"description": "数据流程图列表",
					"schema": {
						"id": {"type": "string", "source": "data_flow_graphs.id", "description": "流程图ID"},
						"thematic_interface_id": {"type": "string", "source": "data_flow_graphs.thematic_interface_id", "description": "主题接口ID"},
						"name": {"type": "string", "source": "data_flow_graphs.name", "description": "流程图名称"},
						"description": {"type": "string", "source": "data_flow_graphs.description", "description": "流程图描述"},
						"definition": {"type": "Object", "source": "data_flow_graphs.definition", "description": "流程图定义"},
						"created_at": {"type": "Date", "source": "data_flow_graphs.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "data_flow_graphs.created_by", "description": "创建者"},
						"updated_at": {"type": "Date", "source": "data_flow_graphs.updated_at", "description": "更新时间"},
						"updated_by": {"type": "string", "source": "data_flow_graphs.updated_by", "description": "更新者"},
						"status": {"type": "string", "source": "data_flow_graphs.status", "description": "流程图状态：draft, active, inactive"}
					}
				},
				"flow_nodes": {
					"type": "Array<Object>",
					"source": "flow_nodes",
					"description": "流程图节点列表",
					"schema": {
						"id": {"type": "string", "source": "flow_nodes.id", "description": "节点ID"},
						"flow_graph_id": {"type": "string", "source": "flow_nodes.flow_graph_id", "description": "流程图ID"},
						"type": {"type": "string", "source": "flow_nodes.type", "description": "节点类型：datasource, api, file, filter, transform, aggregate, output"},
						"config": {"type": "Object", "source": "flow_nodes.config", "description": "节点配置"},
						"position_x": {"type": "number", "source": "flow_nodes.position_x", "description": "X坐标"},
						"position_y": {"type": "number", "source": "flow_nodes.position_y", "description": "Y坐标"},
						"name": {"type": "string", "source": "flow_nodes.name", "description": "节点名称"},
						"created_at": {"type": "Date", "source": "flow_nodes.created_at", "description": "创建时间"},
						"created_by": {"type": "string", "source": "flow_nodes.created_by", "description": "创建者"}
					}
				},
				"field_count": {"type": "number", "description": "字段总数", "computed": true},
				"flow_graph_count": {"type": "number", "description": "流程图总数", "computed": true},
				"flow_node_count": {"type": "number", "description": "流程图节点总数", "computed": true},
				"primary_key_count": {"type": "number", "description": "主键字段数", "computed": true},
				"unique_field_count": {"type": "number", "description": "唯一字段数", "computed": true},
				"not_null_field_count": {"type": "number", "description": "非空字段数", "computed": true},
				"active_flow_graph_count": {"type": "number", "description": "活跃流程图数", "computed": true},
				"draft_flow_graph_count": {"type": "number", "description": "草稿流程图数", "computed": true},
				"inactive_flow_graph_count": {"type": "number", "description": "非活跃流程图数", "computed": true}
			}
		}';
	`,
}
