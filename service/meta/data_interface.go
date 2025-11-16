package meta

type DataInterfaceConfigDefinition struct {
	ID          string                     `json:"id"`
	Type        string                     `json:"type"` // database, messaging, api, file
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Icon        string                     `json:"icon"`
	MetaConfig  []DataInterfaceConfigField `json:"meta_config"` // 连接配置字段
}

type DataInterfaceConfigField struct {
	Name         string                                 `json:"name"`
	DisplayName  string                                 `json:"display_name"`
	Type         string                                 `json:"type"` // string,text, number, boolean, array, map, enum,map_variable
	Required     bool                                   `json:"required"`
	DefaultValue interface{}                            `json:"default_value,omitempty"`
	Description  string                                 `json:"description"`
	Options      []string                               `json:"options,omitempty"`      // 用于select类型
	Min          float64                                `json:"min,omitempty"`          // 用于number类型
	Max          float64                                `json:"max,omitempty"`          // 用于number类型
	Pattern      string                                 `json:"pattern,omitempty"`      // 用于string类型的正则验证
	Placeholder  string                                 `json:"placeholder,omitempty"`  // 前端显示的占位符
	HelpText     string                                 `json:"help_text,omitempty"`    // 帮助文本
	Group        string                                 `json:"group,omitempty"`        // 字段分组
	MapVariable  map[string]DataInterfaceConfigVariable `json:"map_variable,omitempty"` // 变量
	// 依赖关系配置
	Dependencies []FieldDependency `json:"dependencies,omitempty"` // 字段依赖关系
}
type DataInterfaceConfigVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"`     // string,number,timestamp,db_field
	Value       string `json:"value"`    //固定值或者 current_time 表示当前时间,current_time+1d 表示当前时间加1天
	Format      string `json:"format"`   // timestamp 类型时，格式为 "2006-01-02T15:04:05Z"
	DbTable     string `json:"db_table"` // db_field 类型时，表名
	DbField     string `json:"db_field"` // db_field 类型时，字段名
	Description string `json:"description"`
}

// FieldDependency 字段依赖关系
type FieldDependency struct {
	Field     string      `json:"field"`     // 依赖的字段名
	Condition string      `json:"condition"` // 条件：equals, not_equals, in, not_in, greater_than, less_than, contains, not_contains
	Value     interface{} `json:"value"`     // 条件值
	Action    string      `json:"action"`    // 动作：show, hide, enable, disable, require, optional
}

var DataInterfaceConfigDefinitions = make(map[string]DataInterfaceConfigDefinition)

const DataInterfaceConfigFieldTableName = "table_name"
const DataInterfaceConfigFieldUrlSuffix = "url_suffix"
const DataInterfaceConfigFieldMethod = "method"
const DataInterfaceConfigFieldHeaders = "headers"
const DataInterfaceConfigFieldBody = "body"
const DataInterfaceConfigFieldPathParams = "path_params"
const DataInterfaceConfigFieldFormData = "form_data"
const DataInterfaceConfigFieldResponseType = "response_type"
const DataInterfaceConfigFieldUseFormData = "use_form_data"
const DataInterfaceConfigFieldUrlPattern = "url_pattern"
const DataInterfaceConfigFieldDataPath = "data_path"
const DataInterfaceConfigFieldPaginationEnabled = "pagination_enabled"
const DataInterfaceConfigFieldPaginationPageParam = "pagination_page_param"
const DataInterfaceConfigFieldPaginationSizeParam = "pagination_size_param"
const DataInterfaceConfigFieldPaginationStartValue = "pagination_start_value"
const DataInterfaceConfigFieldPaginationDefaultSize = "pagination_default_size"
const DataInterfaceConfigFieldPaginationParamLocation = "pagination_param_location"

// 数据库接口字段常量
const DataInterfaceConfigFieldQueryType = "query_type"
const DataInterfaceConfigFieldCustomSQL = "custom_sql"
const DataInterfaceConfigFieldQueryParams = "query_params"
const DataInterfaceConfigFieldWhereConditions = "where_conditions"
const DataInterfaceConfigFieldOrderBy = "order_by"
const DataInterfaceConfigFieldLimitConfig = "limit_config"
const DataInterfaceConfigFieldIncrementalConfig = "incremental_config"
const DataInterfaceConfigFieldConnectionConfig = "connection_config"

// 增量更新字段常量
const DataInterfaceConfigFieldIncrementalFieldName = "incremental_field_name"
const DataInterfaceConfigFieldIncrementalFieldType = "incremental_field_type"
const DataInterfaceConfigFieldIncrementalFieldFormat = "incremental_field_format"
const DataInterfaceConfigFieldIncrementalParamName = "incremental_param_name"

// API接口字段常量
const DataInterfaceConfigFieldSyncMode = "sync_mode"
const DataInterfaceConfigFieldContentType = "content_type"

// 消息接口字段常量
const DataInterfaceConfigFieldMessagingType = "messaging_type"
const DataInterfaceConfigFieldQOS = "qos"
const DataInterfaceConfigFieldRetain = "retain"
const DataInterfaceConfigFieldAuthValidation = "auth_validation"
const DataInterfaceConfigFieldPayloadValidation = "payload_validation"
const DataInterfaceConfigFieldMessageFormat = "message_format"
const DataInterfaceConfigFieldBatchConfig = "batch_config"
const DataInterfaceConfigFieldErrorHandling = "error_handling"

// HTTP响应解析相关字段
const DataInterfaceConfigFieldResponseParser = "response_parser"
const DataInterfaceConfigFieldSuccessField = "success_field"
const DataInterfaceConfigFieldSuccessValue = "success_value"
const DataInterfaceConfigFieldSuccessCondition = "success_condition"
const DataInterfaceConfigFieldErrorField = "error_field"
const DataInterfaceConfigFieldErrorMessageField = "error_message_field"
const DataInterfaceConfigFieldTotalField = "total_field"
const DataInterfaceConfigFieldPageField = "page_field"
const DataInterfaceConfigFieldPageSizeField = "page_size_field"
const DataInterfaceConfigFieldStatusCodeSuccess = "status_code_success"

func init() {
	initializeDefaultDataInterfaceConfigs()
}
func initializeDefaultDataInterfaceConfigs() {
	databaseInterfaceConfig := DataInterfaceConfigDefinition{
		ID:          DataSourceCategoryDatabase,
		Type:        DataSourceCategoryDatabase,
		Name:        "数据库",
		Description: "数据库接口",
		Icon:        "database",
		MetaConfig: []DataInterfaceConfigField{
			{
				Name:         DataSourceFieldSchema,
				DisplayName:  "Schema",
				Type:         "string",
				Required:     false,
				DefaultValue: "public",
				Description:  "数据库Schema名称",
				Placeholder:  "public",
				Group:        "数据库配置",
			},
			{
				Name:         DataInterfaceConfigFieldTableName,
				DisplayName:  "表名",
				Type:         "string",
				Required:     true,
				DefaultValue: "",
				Description:  "数据库表名",
				Placeholder:  "user_info",
				Group:        "数据库配置",
			},
			{
				Name:         DataInterfaceConfigFieldQueryType,
				DisplayName:  "查询类型",
				Type:         "enum",
				Required:     false,
				DefaultValue: "select",
				Description:  "数据库查询类型",
				Options: []string{
					"select",    // 查询数据
					"custom",    // 自定义SQL
					"procedure", // 存储过程
					"function",  // 函数调用
				},
				Group: "查询配置",
			},
			{
				Name:         DataInterfaceConfigFieldCustomSQL,
				DisplayName:  "自定义SQL",
				Type:         "text",
				Required:     false,
				DefaultValue: "",
				Description:  "自定义SQL查询语句，支持参数占位符 :param",
				Placeholder:  "SELECT * FROM users WHERE created_at > :since_time",
				Group:        "查询配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldQueryType,
						Condition: "in",
						Value:     []string{"custom", "procedure", "function"},
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldQueryType,
						Condition: "equals",
						Value:     "select",
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldQueryParams,
				DisplayName:  "查询参数",
				Type:         "map_variable",
				Required:     false,
				DefaultValue: map[string]DataInterfaceConfigVariable{},
				Description:  "查询参数定义，支持动态参数",
				Group:        "查询配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldQueryType,
						Condition: "in",
						Value:     []string{"custom", "procedure", "function"},
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldQueryType,
						Condition: "equals",
						Value:     "select",
						Action:    "hide",
					},
				},
			},
			{
				Name:        DataInterfaceConfigFieldWhereConditions,
				DisplayName: "WHERE条件",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":    false,
					"conditions": []map[string]interface{}{},
				},
				Description: "WHERE条件配置，用于SELECT查询",
				Group:       "查询配置",
			},
			{
				Name:         DataInterfaceConfigFieldOrderBy,
				DisplayName:  "排序字段",
				Type:         "string",
				Required:     false,
				DefaultValue: "",
				Description:  "排序字段和方向，如 created_at DESC, id ASC",
				Placeholder:  "created_at DESC",
				Group:        "查询配置",
			},
			{
				Name:        DataInterfaceConfigFieldLimitConfig,
				DisplayName: "限制配置",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":       false,
					"default_limit": 1000,
					"max_limit":     10000,
				},
				Description: "查询结果数量限制配置",
				Group:       "查询配置",
			},
			{
				Name:        DataInterfaceConfigFieldIncrementalConfig,
				DisplayName: "增量查询配置",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":         false,
					"increment_field": "updated_at",
					"field_type":      "timestamp", // timestamp, number
					"initial_value":   "",
				},
				Description: "增量查询配置，用于数据同步",
				Group:       "增量配置",
			},
			{
				Name:        DataInterfaceConfigFieldConnectionConfig,
				DisplayName: "连接配置",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"read_timeout":  30, // 秒
					"query_timeout": 60, // 秒
					"max_idle_conn": 10,
					"max_open_conn": 100,
				},
				Description: "数据库连接配置",
				Group:       "连接配置",
			},
		},
	}

	apiInterfaceConfig := DataInterfaceConfigDefinition{
		ID:          DataSourceCategoryAPI,
		Type:        DataSourceCategoryAPI,
		Name:        "API",
		Description: "API接口",
		Icon:        "api",
		MetaConfig: []DataInterfaceConfigField{
			{
				Name:         DataInterfaceConfigFieldUrlPattern,
				DisplayName:  "URL模式",
				Type:         "enum",
				Required:     true,
				DefaultValue: "suffix",
				Description:  "URL构建模式",
				Options: []string{
					"suffix",   // 基础URL + URL后缀
					"query",    // 基础URL + 查询参数
					"path",     // 基础URL + 路径参数
					"combined", // 组合模式
				},
			},
			{
				Name:         DataInterfaceConfigFieldUrlSuffix,
				DisplayName:  "URL后缀",
				Type:         "string",
				Required:     false,
				DefaultValue: "",
				Description:  "API URL后缀，如 /device-info",
				Placeholder:  "/device-info",
			},
			{
				Name:         DataInterfaceConfigFieldMethod,
				DisplayName:  "请求方法",
				Type:         "enum",
				Required:     true,
				DefaultValue: "GET",
				Description:  "API请求方法",
				Options: []string{
					"GET",
					"POST",
					"PUT",
					"DELETE",
				},
			},
			{
				Name:         DataInterfaceConfigFieldHeaders,
				DisplayName:  "请求头",
				Type:         "map",
				Required:     false,
				DefaultValue: map[string]string{},
				Description:  "请求头(key-value)",
			},
			{
				Name:         DataInterfaceConfigFieldBody,
				DisplayName:  "请求体",
				Type:         "text",
				Required:     false,
				DefaultValue: "{}",
				Description:  "请求体(json格式)",
			},
			{
				Name:         DataInterfaceConfigFieldQueryParams,
				DisplayName:  "查询参数",
				Type:         "map_variable",
				Required:     false,
				DefaultValue: map[string]DataInterfaceConfigVariable{},
				Description:  "查询参数(key-value)，如 type=device",
			},
			{
				Name:         DataInterfaceConfigFieldPathParams,
				DisplayName:  "路径参数",
				Type:         "map_variable",
				Required:     false,
				DefaultValue: map[string]DataInterfaceConfigVariable{},
				Description:  "路径参数，如 {id}, {type}",
			},
			{
				Name:         DataInterfaceConfigFieldDataPath,
				DisplayName:  "数据路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "data",
				Description:  "响应中数据的路径，如 data.items",
				Placeholder:  "data.items",
			},
			{
				Name:         DataInterfaceConfigFieldPaginationEnabled,
				DisplayName:  "分页启用",
				Type:         "boolean",
				Required:     false,
				DefaultValue: false,
				Description:  "是否启用分页",
				Group:        "分页详细配置",
			},
			// 分页详细配置字段 - 依赖于分页是否启用
			{
				Name:         DataInterfaceConfigFieldPaginationPageParam,
				DisplayName:  "页码参数名",
				Type:         "string",
				Required:     false,
				DefaultValue: "page",
				Description:  "分页请求中页码参数的名称",
				Placeholder:  "page",
				Group:        "分页详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldPaginationSizeParam,
				DisplayName:  "页大小参数名",
				Type:         "string",
				Required:     false,
				DefaultValue: "size",
				Description:  "分页请求中页大小参数的名称",
				Placeholder:  "size",
				Group:        "分页详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldPaginationStartValue,
				DisplayName:  "起始页码",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(1),
				Description:  "分页的起始页码（通常为0或1）",
				Min:          0,
				Max:          10,
				Group:        "分页详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldPaginationDefaultSize,
				DisplayName:  "默认页大小",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(20),
				Description:  "默认每页返回的记录数",
				Min:          1,
				Max:          1000,
				Group:        "分页详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldPaginationParamLocation,
				DisplayName:  "参数位置",
				Type:         "enum",
				Required:     false,
				DefaultValue: "query",
				Description:  "分页参数在请求中的位置",
				Options:      []string{"query", "body"},
				Group:        "分页详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldPaginationEnabled,
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldUseFormData,
				DisplayName:  "使用表单数据",
				Type:         "boolean",
				Required:     false,
				DefaultValue: false,
				Description:  "使用表单数据而非JSON",
			},
			{
				Name:         DataInterfaceConfigFieldResponseType,
				DisplayName:  "响应类型",
				Type:         "enum",
				Required:     true,
				DefaultValue: "json",
				Description:  "API响应类型",
				Options: []string{
					"json",
					"xml",
					"html",
					"text",
				},
			},
			{
				Name:        DataInterfaceConfigFieldResponseParser,
				DisplayName: "响应解析配置",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":            true,
					"success_detection":  "status_code", // status_code, field_value, custom
					"data_extraction":    "simple",      // simple, nested, custom
					"error_handling":     "standard",    // standard, custom
					"pagination_support": false,
				},
				Description: "响应解析配置",
				Group:       "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldSuccessCondition,
				DisplayName:  "成功判断条件",
				Type:         "enum",
				Required:     false,
				DefaultValue: "status_code",
				Description:  "如何判断请求成功",
				Options: []string{
					"status_code", // 仅根据HTTP状态码判断
					"field_value", // 根据响应字段值判断
					"both",        // 同时检查状态码和字段值
					"custom",      // 自定义判断逻辑
				},
				Group: "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldStatusCodeSuccess,
				DisplayName:  "成功状态码范围",
				Type:         "string",
				Required:     false,
				DefaultValue: "200-299",
				Description:  "HTTP成功状态码范围，如 200-299 或 200,201,202",
				Placeholder:  "200-299",
				Group:        "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldSuccessField,
				DisplayName:  "成功字段路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "status",
				Description:  "响应中表示成功的字段路径，如 status 或 result.code",
				Placeholder:  "status",
				Group:        "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldSuccessValue,
				DisplayName:  "成功值",
				Type:         "string",
				Required:     false,
				DefaultValue: "0",
				Description:  "表示成功的字段值，支持多个值用逗号分隔，如 0,200,success",
				Placeholder:  "0,success",
				Group:        "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldErrorField,
				DisplayName:  "错误字段路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "error",
				Description:  "响应中表示错误的字段路径",
				Placeholder:  "error",
				Group:        "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldErrorMessageField,
				DisplayName:  "错误消息字段路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "message",
				Description:  "响应中错误消息的字段路径",
				Placeholder:  "message",
				Group:        "响应解析",
			},
			{
				Name:         DataInterfaceConfigFieldTotalField,
				DisplayName:  "总数字段路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "total",
				Description:  "响应中表示总记录数的字段路径，用于分页",
				Placeholder:  "total",
				Group:        "分页解析",
			},
			{
				Name:         DataInterfaceConfigFieldPageField,
				DisplayName:  "页码字段路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "page",
				Description:  "响应中表示当前页码的字段路径",
				Placeholder:  "page",
				Group:        "分页解析",
			},
			{
				Name:         DataInterfaceConfigFieldPageSizeField,
				DisplayName:  "页大小字段路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "size",
				Description:  "响应中表示每页大小的字段路径",
				Placeholder:  "size",
				Group:        "分页解析",
			},
			{
				Name:        "incremental_config",
				DisplayName: "增量更新配置",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":         false,
					"increment_field": "updated_at",
					"field_type":      "timestamp", // timestamp, number, string
					"field_format":    "2006-01-02T15:04:05Z",
					"param_location":  "query", // query, body
					"param_name":      "since",
					"initial_value":   "",
					"auto_increment":  true,
				},
				Description: "增量更新配置：enabled-是否启用增量更新，increment_field-增量字段名，field_type-字段类型(timestamp/number/string)，field_format-时间格式(仅timestamp类型)，param_location-参数位置(query/body)，param_name-请求参数名，initial_value-初始值，auto_increment-自动递增",
				Group:       "增量更新",
			},
			// 增量更新详细配置字段 - 依赖于增量更新是否启用
			{
				Name:         DataInterfaceConfigFieldIncrementalFieldName,
				DisplayName:  "增量字段名",
				Type:         "string",
				Required:     false,
				DefaultValue: "updated_at",
				Description:  "用于增量更新的字段名称",
				Placeholder:  "updated_at",
				Group:        "增量更新详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldIncrementalFieldType,
				DisplayName:  "字段类型",
				Type:         "enum",
				Required:     false,
				DefaultValue: "timestamp",
				Description:  "增量字段的数据类型",
				Options:      []string{"timestamp", "number", "string"},
				Group:        "增量更新详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldIncrementalFieldFormat,
				DisplayName:  "时间格式",
				Type:         "string",
				Required:     false,
				DefaultValue: "2006-01-02T15:04:05Z",
				Description:  "时间字段的格式（仅当字段类型为timestamp时有效）",
				Placeholder:  "2006-01-02T15:04:05Z",
				Group:        "增量更新详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
					{
						Field:     DataInterfaceConfigFieldIncrementalFieldType,
						Condition: "equals",
						Value:     "timestamp",
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldIncrementalFieldType,
						Condition: "not_equals",
						Value:     "timestamp",
						Action:    "hide",
					},
					{
						Field:     DataInterfaceConfigFieldIncrementalFieldType,
						Condition: "==",
						Value:     "timestamp",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldIncrementalParamName,
				DisplayName:  "请求参数名",
				Type:         "string",
				Required:     false,
				DefaultValue: "since",
				Description:  "增量更新请求参数的名称",
				Placeholder:  "since",
				Group:        "增量更新详细配置",
				Dependencies: []FieldDependency{
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     "incremental_config.enabled",
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldSyncMode,
				DisplayName:  "同步模式",
				Type:         "enum",
				Required:     false,
				DefaultValue: "full",
				Description:  "数据同步模式",
				Options: []string{
					"full",        // 全量更新
					"incremental", // 增量更新
					"mixed",       // 混合模式，支持全量和增量
				},
				Group: "同步配置",
			},
			{
				Name:         DataInterfaceConfigFieldContentType,
				DisplayName:  "请求内容类型",
				Type:         "enum",
				Required:     false,
				DefaultValue: "application/json",
				Description:  "HTTP请求的Content-Type",
				Options: []string{
					"application/json",
					"application/x-www-form-urlencoded",
					"multipart/form-data",
					"text/plain",
					"application/xml",
				},
				Group: "请求配置",
			},
		},
	}
	// 实时消息数据源接口配置
	messagingInterfaceConfig := DataInterfaceConfigDefinition{
		ID:          DataSourceCategoryMessaging,
		Type:        DataSourceCategoryMessaging,
		Name:        "实时消息",
		Description: "实时消息接口",
		Icon:        "messaging",
		MetaConfig: []DataInterfaceConfigField{
			{
				Name:         DataInterfaceConfigFieldMessagingType,
				DisplayName:  "消息类型",
				Type:         "enum",
				Required:     true,
				DefaultValue: "mqtt",
				Description:  "实时消息数据源类型",
				Options: []string{
					"mqtt",      // MQTT订阅
					"http_post", // HTTP POST接收
				},
			},
			// MQTT相关配置
			{
				Name:         DataSourceFieldTopic,
				DisplayName:  "MQTT主题",
				Type:         "string",
				Required:     false,
				DefaultValue: "data/+",
				Description:  "订阅的MQTT主题，支持通配符，如 data/+, sensor/# 等",
				Placeholder:  "data/+",
				Group:        "MQTT配置",
				Dependencies: []FieldDependency{
					{
						Field:     DataInterfaceConfigFieldMessagingType,
						Condition: "equals",
						Value:     "mqtt",
						Action:    "show",
					},
					{
						Field:     DataInterfaceConfigFieldMessagingType,
						Condition: "not_equals",
						Value:     "mqtt",
						Action:    "hide",
					},
				},
			},
			{
				Name:         DataInterfaceConfigFieldQOS,
				DisplayName:  "服务质量等级",
				Type:         "enum",
				Required:     false,
				DefaultValue: "1",
				Description:  "MQTT消息服务质量等级",
				Options:      []string{"0", "1", "2"},
				Group:        "MQTT配置",
			},
			{
				Name:         DataInterfaceConfigFieldRetain,
				DisplayName:  "保留消息",
				Type:         "boolean",
				Required:     false,
				DefaultValue: false,
				Description:  "是否处理保留消息",
				Group:        "MQTT配置",
			},
			// HTTP POST相关配置
			{
				Name:         DataInterfaceConfigFieldUrlSuffix,
				DisplayName:  "URL后缀",
				Type:         "string",
				Required:     false,
				DefaultValue: "/data",
				Description:  "接收POST数据的URL后缀，如 /data, /webhook/device 等",
				Placeholder:  "/data",
				Group:        "HTTP POST配置",
			},
			{
				Name:        DataInterfaceConfigFieldAuthValidation,
				DisplayName: "认证验证",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":    false,
					"auth_type":  "token", // token, header, query
					"auth_field": "token",
					"auth_value": "",
				},
				Description: "认证验证配置",
				Group:       "HTTP POST配置",
			},
			{
				Name:        DataInterfaceConfigFieldPayloadValidation,
				DisplayName: "载荷验证",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":                           false,
					DataInterfaceConfigFieldContentType: "application/json",
					"max_size":                          1048576, // 1MB
					"required_fields":                   []string{},
				},
				Description: "载荷验证配置",
				Group:       "HTTP POST配置",
			},
			// 通用消息处理配置
			{
				Name:         DataInterfaceConfigFieldMessageFormat,
				DisplayName:  "消息格式",
				Type:         "enum",
				Required:     false,
				DefaultValue: "json",
				Description:  "消息数据格式",
				Options: []string{
					"json",
					"xml",
					"plain_text",
					"binary",
				},
				Group: "消息处理",
			},
			{
				Name:         DataInterfaceConfigFieldDataPath,
				DisplayName:  "数据路径",
				Type:         "string",
				Required:     false,
				DefaultValue: "",
				Description:  "消息中数据的路径，如 payload.data, 空表示整个消息",
				Placeholder:  "payload.data",
				Group:        "消息处理",
			},
			{
				Name:        "batch_config",
				DisplayName: "批处理配置",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"enabled":           false,
					"batch_size":        100,
					"batch_timeout":     30, // 秒
					"flush_on_shutdown": true,
				},
				Description: "批处理配置：enabled-启用批处理，batch_size-批大小，batch_timeout-批超时时间(秒)，flush_on_shutdown-关闭时刷新",
				Group:       "消息处理",
			},
			{
				Name:        "error_handling",
				DisplayName: "错误处理",
				Type:        "map",
				Required:    false,
				DefaultValue: map[string]interface{}{
					"retry_enabled":       true,
					"max_retries":         3,
					"retry_interval":      5, // 秒
					"dead_letter_enabled": false,
					"dead_letter_topic":   "errors",
				},
				Description: "错误处理配置",
				Group:       "消息处理",
			},
		},
	}

	DataInterfaceConfigDefinitions[DataSourceCategoryDatabase] = databaseInterfaceConfig
	DataInterfaceConfigDefinitions[DataSourceCategoryAPI] = apiInterfaceConfig
	DataInterfaceConfigDefinitions[DataSourceCategoryMessaging] = messagingInterfaceConfig
}
