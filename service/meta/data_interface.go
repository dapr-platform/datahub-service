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
	Name         string      `json:"name"`
	DisplayName  string      `json:"display_name"`
	Type         string      `json:"type"` // string,text, number, boolean, array, map, enum,map_variable
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description"`
	Options      []string    `json:"options,omitempty"`     // 用于select类型
	Min          float64     `json:"min,omitempty"`         // 用于number类型
	Max          float64     `json:"max,omitempty"`         // 用于number类型
	Pattern      string      `json:"pattern,omitempty"`     // 用于string类型的正则验证
	Placeholder  string      `json:"placeholder,omitempty"` // 前端显示的占位符
	HelpText     string      `json:"help_text,omitempty"`   // 帮助文本
	Group        string      `json:"group,omitempty"`       // 字段分组
	MapVariable  map[string]DataInterfaceConfigVariable `json:"map_variable,omitempty"` // 变量
}
type DataInterfaceConfigVariable struct {
	Name          string `json:"name"`
	Type          string `json:"type"` // number,timestamp,db_field
	InitValue     string `json:"init_value"`
	AutoIncrement bool   `json:"auto_increment"`
	IncrementStep string `json:"increment_step"` //number 类型时，步长, timestamp 类型时，格式为 "1s", "1m", "1h", "1d"
	DbTable       string `json:"db_table"`       // db_field 类型时，表名
	DbField       string `json:"db_field"`       // db_field 类型时，字段名
	Description   string `json:"description"`
}

var DataInterfaceConfigDefinitions = make(map[string]DataInterfaceConfigDefinition)

const DataInterfaceConfigFieldTableName = "table_name"
const DataInterfaceConfigFieldUrlSuffix = "url_suffix"
const DataInterfaceConfigFieldMethod = "method"
const DataInterfaceConfigFieldHeaders = "headers"
const DataInterfaceConfigFieldBody = "body"
const DataInterfaceConfigFieldQueryParams = "query_params"
const DataInterfaceConfigFieldPathParams = "path_params"
const DataInterfaceConfigFieldFormData = "form_data"
const DataInterfaceConfigFieldResponseType = "response_type"
const DataInterfaceConfigFieldUseFormData = "use_form_data"

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
				Name:         DataInterfaceConfigFieldTableName,
				DisplayName:  "表名",
				Type:         "string",
				Required:     true,
				DefaultValue: "",
				Description:  "数据库表名",
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
				Name:         DataInterfaceConfigFieldUrlSuffix,
				DisplayName:  "URL后缀",
				Type:         "string",
				Required:     true,
				DefaultValue: "",
				Description:  "API URL后缀",
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
				Description:  "查询参数(key-value)",
			},
			{
				Name:         DataInterfaceConfigFieldPathParams,
				DisplayName:  "路径参数",
				Type:         "map_variable",
				Required:     false,
				DefaultValue: map[string]DataInterfaceConfigVariable{},
				Description:  "路径参数",
			},
			{
				Name:         DataInterfaceConfigFieldUseFormData,
				DisplayName:  "使用表单数据",
				Type:         "boolean",
				Required:     false,
				DefaultValue: false,
				Description:  "使用表单数据",
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
		},
	}
	DataInterfaceConfigDefinitions[DataSourceCategoryDatabase] = databaseInterfaceConfig
	DataInterfaceConfigDefinitions[DataSourceCategoryAPI] = apiInterfaceConfig
}
