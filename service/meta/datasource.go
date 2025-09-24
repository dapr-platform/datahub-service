package meta

import (
	"fmt"
	"regexp"
)

// DataSourceTypeDefinition 数据源类型完整定义
type DataSourceTypeDefinition struct {
	ID                string                     `json:"id"`
	Type              string                     `json:"type"` // database, messaging, api, file
	Name              string                     `json:"name"`
	Description       string                     `json:"description"`
	Category          string                     `json:"category"`
	Icon              string                     `json:"icon"`
	MetaConfig        []DataSourceConfigField    `json:"meta_config"`   // 连接配置字段
	ParamsConfig      []DataSourceConfigField    `json:"params_config"` // 参数配置字段
	ValidationRules   []DataSourceValidationRule `json:"validation_rules"`
	Examples          []DataSourceExample        `json:"examples"`
	SupportedFeatures []string                   `json:"supported_features"`
	Documentation     string                     `json:"documentation"`
	IsActive          bool                       `json:"is_active"`
}

// DataSourceConfigField 配置字段定义
type DataSourceConfigField struct {
	Name         string      `json:"name"`
	DisplayName  string      `json:"display_name"`
	Type         string      `json:"type"` // string, number, boolean, array, object, enum
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
	// 依赖关系配置
	Dependencies []DataSourceFieldDependency `json:"dependencies,omitempty"` // 字段依赖关系
}

// DataSourceFieldDependency 字段依赖关系
type DataSourceFieldDependency struct {
	Field     string      `json:"field"`     // 依赖的字段名
	Condition string      `json:"condition"` // 条件：equals, not_equals, in, not_in, greater_than, less_than, contains, not_contains
	Value     interface{} `json:"value"`     // 条件值
	Action    string      `json:"action"`    // 动作：show, hide, enable, disable, require, optional
}

// DataSourceValidationRule 验证规则定义
type DataSourceValidationRule struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // required, format, range, custom
	Field     string                 `json:"field"`
	Condition map[string]interface{} `json:"condition"`
	Message   string                 `json:"message"`
	Level     string                 `json:"level"` // error, warning
}

// DataSourceExample 示例配置
type DataSourceExample struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ConnectionConfig map[string]interface{} `json:"connection_config"`
	ParamsConfig     map[string]interface{} `json:"params_config,omitempty"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Score    int      `json:"score"` // 0-100
}

// ValidateConfig 验证配置
func (d *DataSourceTypeDefinition) ValidateConfig(connectionConfig, paramsConfig map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Score:    100,
	}

	// 验证连接配置
	d.validateFields(d.MetaConfig, connectionConfig, result)

	// 验证参数配置
	if paramsConfig != nil {
		d.validateFields(d.ParamsConfig, paramsConfig, result)
	}

	// 应用自定义验证规则
	d.applyValidationRules(connectionConfig, paramsConfig, result)

	// 计算最终分数
	d.calculateScore(result)

	return result
}

// validateFields 验证字段
func (d *DataSourceTypeDefinition) validateFields(fields []DataSourceConfigField, config map[string]interface{}, result *ValidationResult) {
	for _, field := range fields {
		value, exists := config[field.Name]

		// 检查必需字段
		if field.Required && (!exists || value == nil || value == "") {
			result.Errors = append(result.Errors, fmt.Sprintf("缺少必需字段: %s", field.DisplayName))
			result.IsValid = false
			continue
		}

		// 如果字段不存在或为空，跳过后续验证
		if !exists || value == nil {
			continue
		}

		// 类型验证
		if !d.validateFieldType(value, field.Type) {
			result.Errors = append(result.Errors, fmt.Sprintf("字段 %s 类型不正确，期望: %s", field.DisplayName, field.Type))
			result.IsValid = false
			continue
		}

		// 范围验证
		if field.Type == "number" {
			if numVal, ok := value.(float64); ok {
				if field.Min != 0 && numVal < field.Min {
					result.Errors = append(result.Errors, fmt.Sprintf("字段 %s 值过小，最小值: %.0f", field.DisplayName, field.Min))
					result.IsValid = false
				}
				if field.Max != 0 && numVal > field.Max {
					result.Errors = append(result.Errors, fmt.Sprintf("字段 %s 值过大，最大值: %.0f", field.DisplayName, field.Max))
					result.IsValid = false
				}
			}
		}

		// 选项验证
		if len(field.Options) > 0 && field.Type == "string" {
			if strVal, ok := value.(string); ok {
				isValid := false
				for _, option := range field.Options {
					if strVal == option {
						isValid = true
						break
					}
				}
				if !isValid {
					result.Errors = append(result.Errors, fmt.Sprintf("字段 %s 值不在允许的选项中: %v", field.DisplayName, field.Options))
					result.IsValid = false
				}
			}
		}

		// 正则验证
		if field.Pattern != "" && field.Type == "string" {
			if strVal, ok := value.(string); ok {
				matched, err := regexp.MatchString(field.Pattern, strVal)
				if err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("字段 %s 正则表达式验证失败: %v", field.DisplayName, err))
				} else if !matched {
					result.Errors = append(result.Errors, fmt.Sprintf("字段 %s 格式不正确", field.DisplayName))
					result.IsValid = false
				}
			}
		}
	}
}

// validateFieldType 验证字段类型
func (d *DataSourceTypeDefinition) validateFieldType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		if !ok {
			_, ok = value.(int)
		}
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true
	}
}

// applyValidationRules 应用自定义验证规则
func (d *DataSourceTypeDefinition) applyValidationRules(connectionConfig, paramsConfig map[string]interface{}, result *ValidationResult) {
	allConfig := make(map[string]interface{})

	// 合并配置
	for k, v := range connectionConfig {
		allConfig[k] = v
	}
	for k, v := range paramsConfig {
		allConfig[k] = v
	}

	// 应用验证规则
	for _, rule := range d.ValidationRules {
		if d.evaluateRule(rule, allConfig) {
			if rule.Level == "error" {
				result.Errors = append(result.Errors, rule.Message)
				result.IsValid = false
			} else {
				result.Warnings = append(result.Warnings, rule.Message)
			}
		}
	}
}

// evaluateRule 评估验证规则
func (d *DataSourceTypeDefinition) evaluateRule(rule DataSourceValidationRule, config map[string]interface{}) bool {
	// 简单的规则评估实现
	value, exists := config[rule.Field]
	if !exists {
		return false
	}

	// 根据规则类型进行不同的验证
	switch rule.Type {
	case "required":
		return value == nil || value == ""
	case "format":
		if pattern, ok := rule.Condition["pattern"].(string); ok {
			if strVal, ok := value.(string); ok {
				matched, _ := regexp.MatchString(pattern, strVal)
				return !matched
			}
		}
	case "range":
		if numVal, ok := value.(float64); ok {
			if min, ok := rule.Condition["min"].(float64); ok && numVal < min {
				return true
			}
			if max, ok := rule.Condition["max"].(float64); ok && numVal > max {
				return true
			}
		}
	}

	return false
}

// calculateScore 计算验证分数
func (d *DataSourceTypeDefinition) calculateScore(result *ValidationResult) {
	score := 100

	// 每个错误扣20分
	score -= len(result.Errors) * 20

	// 每个警告扣5分
	score -= len(result.Warnings) * 5

	// 确保评分不低于0
	if score < 0 {
		score = 0
	}

	// 如果有错误，最高分数不超过50
	if len(result.Errors) > 0 {
		result.IsValid = false
		if score > 50 {
			score = 50
		}
	}

	result.Score = score
}

const (
	DataSourceCategoryDatabase  = "database"
	DataSourceCategoryMessaging = "messaging"
	DataSourceCategoryAPI       = "api"
)

const (
	DataSourceTypeDBPostgreSQL      = "postgresql"
	DataSourceTypeApiHTTP           = "http"
	DataSourceTypeApiHTTPWithAuth   = "http_with_auth"
	DataSourceTypeMessagingMQTT     = "mqtt"
	DataSourceTypeMessagingHttpPost = "http_post"
)
const DataSourceFieldHost = "host"
const DataSourceFieldPort = "port"
const DataSourceFieldDatabase = "database"
const DataSourceFieldUsername = "username"
const DataSourceFieldPassword = "password"
const DataSourceFieldAuthType = "auth_type"
const DataSourceFieldApiKey = "api_key"
const DataSourceFieldApiSecret = "api_secret"
const DataSourceFieldApiKeyHeader = "api_key_header"
const DataSourceFieldScript = "script"
const DataSourceFieldScriptEnabled = "script_enabled"
const DataSourceFieldClientId = "client_id"
const DataSourceFieldClientSecret = "client_secret"
const DataSourceFieldGrantType = "grant_type"
const DataSourceFieldScope = "scope"
const DataSourceFieldSchema = "schema"
const DataSourceFieldSSLMode = "ssl_mode"
const DataSourceFieldMaxConnections = "max_connections"
const DataSourceFieldTimeout = "timeout"
const DataSourceFieldCharset = "charset"
const DataSourceFieldBaseUrl = "base_url"
const DataSourceFieldTopic = "topic"
const DataSourceFieldGroupId = "group_id"
const DataSourceFieldAutoOffsetReset = "auto_offset_reset"
const DataSourceFieldMaxPollRecords = "max_poll_records"
const DataSourceFieldBootstrapServers = "bootstrap_servers"
const DatasourceFieldCustomMap = "custom_map"

const (
	DataSourceAuthTypeBasic  = "basic"
	DataSourceAuthTypeBearer = "bearer"
	DataSourceAuthTypeAPIKey = "api_key"
	DataSourceAuthTypeOAuth2 = "oauth2"
	DataSourceAuthTypeCustom = "custom"
)

var DataSourceTypes = make(map[string]*DataSourceTypeDefinition)

func init() {
	initializeDefaultTypes()
}

// initializeDefaultTypes 初始化默认的数据源类型
func initializeDefaultTypes() {
	// PostgreSQL 数据源
	postgresql := &DataSourceTypeDefinition{
		ID:          DataSourceTypeDBPostgreSQL,
		Category:    DataSourceCategoryDatabase,
		Type:        DataSourceTypeDBPostgreSQL,
		Name:        "PostgreSQL",
		Description: "PostgreSQL关系型数据库",
		Icon:        "postgresql",
		MetaConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldHost,
				DisplayName:  "主机",
				Type:         "string",
				Required:     true,
				DefaultValue: "localhost",
				Description:  "数据库服务器地址",
				Pattern:      `^[a-zA-Z0-9.-]+$`,
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldPort,
				DisplayName:  "端口",
				Type:         "number",
				Required:     true,
				DefaultValue: float64(5432),
				Description:  "数据库端口号",
				Min:          1,
				Max:          65535,
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldDatabase,
				DisplayName:  "数据库",
				Type:         "string",
				Required:     true,
				DefaultValue: "postgres",
				Description:  "数据库名称",
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldUsername,
				DisplayName:  "用户名",
				Type:         "string",
				Required:     true,
				DefaultValue: "postgres",
				Description:  "数据库用户名",
				Group:        "认证配置",
			},
			{
				Name:        DataSourceFieldPassword,
				DisplayName: "密码",
				Type:        "string",
				Required:    true,
				Description: "数据库密码",
				Group:       "认证配置",
			},
			{
				Name:         DataSourceFieldSchema,
				DisplayName:  "Schema",
				Type:         "string",
				Required:     false,
				DefaultValue: "public",
				Description:  "数据库Schema",
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldSSLMode,
				DisplayName:  "SSL模式",
				Type:         "enum",
				Required:     false,
				DefaultValue: "disable",
				Description:  "SSL连接模式",
				Options:      []string{"disable", "require", "verify-ca", "verify-full"},
				Group:        "安全配置",
			},
		},
		ParamsConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldTimeout,
				DisplayName:  "超时时间(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(30),
				Description:  "连接超时时间",
				Min:          1,
				Max:          300,
				Group:        "性能配置",
			},
			{
				Name:         DataSourceFieldMaxConnections,
				DisplayName:  "最大连接数",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(100),
				Description:  "连接池最大连接数",
				Min:          1,
				Max:          1000,
				Group:        "性能配置",
			},
		},
		Examples: []DataSourceExample{
			{
				Name:        "本地开发环境",
				Description: "连接本地PostgreSQL数据库",
				ConnectionConfig: map[string]interface{}{
					DataSourceFieldHost:     "localhost",
					DataSourceFieldPort:     5432,
					DataSourceFieldDatabase: "dev_db",
					DataSourceFieldUsername: "dev_user",
					DataSourceFieldPassword: "dev_password",
					DataSourceFieldSchema:   "public",
					DataSourceFieldSSLMode:  "disable",
				},
			},
		},
		SupportedFeatures: []string{"batch_query", "real_time_sync", "transaction", "json_support"},
		Documentation:     "PostgreSQL是一个功能强大的开源对象关系型数据库系统",
		IsActive:          true,
	}

	// HTTP 数据源（无认证）
	httpNoAuth := &DataSourceTypeDefinition{
		ID:          DataSourceTypeApiHTTP,
		Category:    DataSourceCategoryAPI,
		Type:        DataSourceTypeApiHTTP,
		Name:        "HTTP(无认证)",
		Description: "HTTP REST API数据源",
		Icon:        "http",
		MetaConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldBaseUrl,
				DisplayName:  "基础URL",
				Type:         "string",
				Required:     true,
				DefaultValue: "http://localhost:8080",
				Description:  "API基础地址",
				Pattern:      `^https?://.*`,
				Group:        "连接配置",
			},
		},
		ParamsConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldTimeout,
				DisplayName:  "超时时间(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(30),
				Description:  "请求超时时间",
				Min:          1,
				Max:          300,
				Group:        "性能配置",
			},
		},
		Examples: []DataSourceExample{
			{
				Name:        "公开API",
				Description: "连接公开的REST API",
				ConnectionConfig: map[string]interface{}{
					DataSourceFieldBaseUrl: "https://api.example.com",
				},
			},
		},
		SupportedFeatures: []string{"rest_api", "json_data", "batch_processing"},
		Documentation:     "HTTP数据源支持从REST API获取数据",
		IsActive:          true,
	}
	// HTTP 数据源 带认证
	httpWithAuth := &DataSourceTypeDefinition{
		ID:          DataSourceTypeApiHTTPWithAuth,
		Category:    DataSourceCategoryAPI,
		Type:        DataSourceTypeApiHTTPWithAuth,
		Name:        "HTTP(带认证)",
		Description: "HTTP REST API数据源",
		IsActive:    true,
	}
	httpWithAuth.MetaConfig = []DataSourceConfigField{
		{
			Name:         DataSourceFieldBaseUrl,
			DisplayName:  "基础URL",
			Type:         "string",
			Required:     true,
			DefaultValue: "http://localhost:8080",
			Description:  "API基础地址",
			Pattern:      `^https?://.*`,
			Group:        "连接配置",
		},
		{
			Name:        DataSourceFieldUsername,
			DisplayName: "用户名",
			Type:        "string",
			Required:    false,
			Description: "用户名",
			Group:       "认证配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "in",
					Value:     []string{DataSourceAuthTypeBasic, DataSourceAuthTypeBearer, DataSourceAuthTypeOAuth2},
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     []string{DataSourceAuthTypeAPIKey, DataSourceAuthTypeCustom},
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldPassword,
			DisplayName: "密码",
			Type:        "string",
			Required:    false,
			Description: "密码",
			Group:       "认证配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "in",
					Value:     []string{DataSourceAuthTypeBasic, DataSourceAuthTypeBearer, DataSourceAuthTypeOAuth2},
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     []string{DataSourceAuthTypeAPIKey, DataSourceAuthTypeCustom},
					Action:    "hide",
				},
			},
		},
		{
			Name:         DataSourceFieldAuthType,
			DisplayName:  "认证类型",
			Type:         "enum",
			Required:     true,
			DefaultValue: DataSourceAuthTypeBasic,
			Description:  "认证类型",
			Options:      []string{DataSourceAuthTypeBasic, DataSourceAuthTypeBearer, DataSourceAuthTypeAPIKey, DataSourceAuthTypeOAuth2, DataSourceAuthTypeCustom},
			Group:        "认证配置",
		},
		{
			Name:         DatasourceFieldCustomMap,
			DisplayName:  "自定义映射",
			Type:         "map",
			Required:     false,
			DefaultValue: map[string]interface{}{},
			Description:  "自定义映射",
			Group:        "认证配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeCustom,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeCustom,
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldApiKey,
			DisplayName: "API Key",
			Type:        "string",
			Required:    false,
			Description: "API Key",
			Group:       "认证配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeAPIKey,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeAPIKey,
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldApiSecret,
			DisplayName: "API Secret",
			Type:        "string",
			Required:    false,
			Description: "API Secret",
			Group:       "认证配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeAPIKey,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeAPIKey,
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldApiKeyHeader,
			DisplayName: "API Key头名称",
			Type:        "string",
			Required:    false,
			Description: "API Key在HTTP头中的名称",
			Group:       "认证配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeAPIKey,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeAPIKey,
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldClientId,
			DisplayName: "客户端ID",
			Type:        "string",
			Required:    false,
			Description: "OAuth2客户端ID",
			Group:       "OAuth2配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldClientSecret,
			DisplayName: "客户端密钥",
			Type:        "string",
			Required:    false,
			Description: "OAuth2客户端密钥",
			Group:       "OAuth2配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "hide",
				},
			},
		},
		{
			Name:         DataSourceFieldGrantType,
			DisplayName:  "授权类型",
			Type:         "enum",
			Required:     false,
			DefaultValue: "client_credentials",
			Description:  "OAuth2授权类型",
			Options:      []string{"client_credentials", "authorization_code", "password", "refresh_token"},
			Group:        "OAuth2配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "hide",
				},
			},
		},
		{
			Name:        DataSourceFieldScope,
			DisplayName: "作用域",
			Type:        "string",
			Required:    false,
			Description: "OAuth2权限作用域，多个作用域用空格分隔",
			Placeholder: "read write",
			Group:       "OAuth2配置",
			Dependencies: []DataSourceFieldDependency{
				{
					Field:     DataSourceFieldAuthType,
					Condition: "equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "show",
				},
				{
					Field:     DataSourceFieldAuthType,
					Condition: "not_equals",
					Value:     DataSourceAuthTypeOAuth2,
					Action:    "hide",
				},
			},
		},
	}
	// MQTT 数据源
	mqtt := &DataSourceTypeDefinition{
		ID:          DataSourceTypeMessagingMQTT,
		Category:    DataSourceCategoryMessaging,
		Type:        DataSourceTypeMessagingMQTT,
		Name:        "MQTT",
		Description: "MQTT消息数据源（作为客户端订阅消息）",
		Icon:        "mqtt",
		MetaConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldHost,
				DisplayName:  "MQTT服务器地址",
				Type:         "string",
				Required:     true,
				DefaultValue: "localhost",
				Description:  "MQTT broker地址",
				Pattern:      `^[a-zA-Z0-9.-]+$`,
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldPort,
				DisplayName:  "端口",
				Type:         "number",
				Required:     true,
				DefaultValue: float64(1883),
				Description:  "MQTT端口号",
				Min:          1,
				Max:          65535,
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldClientId,
				DisplayName:  "客户端ID",
				Type:         "string",
				Required:     true,
				DefaultValue: "datahub-client",
				Description:  "MQTT客户端标识",
				Group:        "连接配置",
			},
			{
				Name:         DataSourceFieldUsername,
				DisplayName:  "用户名",
				Type:         "string",
				Required:     false,
				DefaultValue: "",
				Description:  "MQTT认证用户名（可选）",
				Group:        "认证配置",
			},
			{
				Name:        DataSourceFieldPassword,
				DisplayName: "密码",
				Type:        "string",
				Required:    false,
				Description: "MQTT认证密码（可选）",
				Group:       "认证配置",
			},
		},
		ParamsConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldTimeout,
				DisplayName:  "连接超时(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(30),
				Description:  "MQTT连接超时时间",
				Min:          1,
				Max:          300,
				Group:        "性能配置",
			},
		},
		Examples: []DataSourceExample{
			{
				Name:        "本地MQTT服务器",
				Description: "连接本地MQTT broker",
				ConnectionConfig: map[string]interface{}{
					DataSourceFieldHost:     "localhost",
					DataSourceFieldPort:     1883,
					DataSourceFieldClientId: "datahub-client-001",
					DataSourceFieldUsername: "",
					DataSourceFieldPassword: "",
				},
			},
		},
		SupportedFeatures: []string{"real_time_messaging", "topic_subscription", "qos_support"},
		Documentation:     "MQTT数据源支持实时消息订阅",
		IsActive:          true,
	}

	// Http Post 数据源
	httpPost := &DataSourceTypeDefinition{
		ID:          DataSourceTypeMessagingHttpPost,
		Category:    DataSourceCategoryMessaging,
		Type:        DataSourceTypeMessagingHttpPost,
		Name:        "HTTP POST接收器",
		Description: "HTTP POST数据源（作为服务器接收第三方POST数据）",
		Icon:        "http-post",
		MetaConfig: []DataSourceConfigField{
			{
				Name:         DataSourceFieldPort,
				DisplayName:  "监听端口",
				Type:         "number",
				Required:     true,
				DefaultValue: float64(8080),
				Description:  "HTTP服务器监听端口",
				Min:          1024,
				Max:          65535,
				Group:        "服务配置",
			},
			{
				Name:         "path_prefix",
				DisplayName:  "路径前缀",
				Type:         "string",
				Required:     false,
				DefaultValue: "/webhook",
				Description:  "接收POST请求的路径前缀",
				Pattern:      `^/.*`,
				Group:        "服务配置",
			},
			{
				Name:         "auth_required",
				DisplayName:  "是否需要认证",
				Type:         "boolean",
				Required:     false,
				DefaultValue: false,
				Description:  "是否需要认证才能接收数据",
				Group:        "认证配置",
			},
			{
				Name:         "auth_token",
				DisplayName:  "认证令牌",
				Type:         "string",
				Required:     false,
				DefaultValue: "",
				Description:  "用于验证POST请求的令牌",
				Group:        "认证配置",
				Dependencies: []DataSourceFieldDependency{
					{
						Field:     "auth_required",
						Condition: "equals",
						Value:     true,
						Action:    "show",
					},
					{
						Field:     "auth_required",
						Condition: "equals",
						Value:     false,
						Action:    "hide",
					},
				},
			},
		},
		ParamsConfig: []DataSourceConfigField{
			{
				Name:         "max_body_size",
				DisplayName:  "最大请求体大小(MB)",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(10),
				Description:  "允许的最大请求体大小",
				Min:          1,
				Max:          100,
				Group:        "性能配置",
			},
			{
				Name:         DataSourceFieldTimeout,
				DisplayName:  "请求超时(秒)",
				Type:         "number",
				Required:     false,
				DefaultValue: float64(30),
				Description:  "处理请求的超时时间",
				Min:          1,
				Max:          300,
				Group:        "性能配置",
			},
		},
		Examples: []DataSourceExample{
			{
				Name:        "Webhook接收器",
				Description: "接收第三方系统的webhook数据",
				ConnectionConfig: map[string]interface{}{
					DataSourceFieldPort: 8080,
					"path_prefix":       "/webhook",
					"auth_required":     true,
					"auth_token":        "your-secret-token",
				},
			},
		},
		SupportedFeatures: []string{"real_time_data", "webhook_receiver", "json_payload", "authentication"},
		Documentation:     "HTTP POST数据源作为服务器接收第三方系统推送的数据",
		IsActive:          true,
	}

	// 注册所有类型
	DataSourceTypes[postgresql.ID] = postgresql
	DataSourceTypes[httpNoAuth.ID] = httpNoAuth
	DataSourceTypes[httpWithAuth.ID] = httpWithAuth
	DataSourceTypes[mqtt.ID] = mqtt
	DataSourceTypes[httpPost.ID] = httpPost
}
