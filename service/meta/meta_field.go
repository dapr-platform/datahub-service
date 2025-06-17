package meta

type MetaField struct {
	Name string `json:"name"`
	DisplayName string `json:"display_name"`
	Type string `json:"type"`
	Required bool `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Description string `json:"description"`
}