/*
 * @module client/pgmeta
 * @description PostgreSQL元数据管理客户端，提供数据库对象的CRUD操作
 * @architecture 客户端架构 - REST API客户端
 * @documentReference client/docs.json
 * @stateFlow 认证 -> API调用 -> 响应处理
 * @rules 确保API调用的安全性和错误处理的完整性
 * @dependencies net/http, encoding/json, fmt, time
 * @refs postgres-meta API 文档
 */

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PgMetaClient PostgreSQL元数据客户端
type PgMetaClient struct {
	BaseURL    string
	HTTPClient *http.Client
	PgHeader   string // pg header value for database connection
}

// NewPgMetaClient 创建新的PgMeta客户端
func NewPgMetaClient(baseURL, pgHeader string) *PgMetaClient {
	return &PgMetaClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		PgHeader:   pgHeader,
	}
}

// ==================== 数据模型定义 ====================

// Schema 数据库模式
type Schema struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

// CreateSchemaRequest 创建模式请求
type CreateSchemaRequest struct {
	Name  string `json:"name"`
	Owner string `json:"owner,omitempty"`
}

// UpdateSchemaRequest 更新模式请求
type UpdateSchemaRequest struct {
	Name  string `json:"name,omitempty"`
	Owner string `json:"owner,omitempty"`
}

// Table 数据表
type Table struct {
	ID               int            `json:"id"`
	Schema           string         `json:"schema"`
	Name             string         `json:"name"`
	RLSEnabled       bool           `json:"rls_enabled"`
	RLSForced        bool           `json:"rls_forced"`
	ReplicaIdentity  string         `json:"replica_identity"`
	Bytes            int            `json:"bytes"`
	Size             string         `json:"size"`
	LiveRowsEstimate int            `json:"live_rows_estimate"`
	DeadRowsEstimate int            `json:"dead_rows_estimate"`
	Comment          *string        `json:"comment"`
	Columns          []Column       `json:"columns,omitempty"`
	PrimaryKeys      []PrimaryKey   `json:"primary_keys"`
	Relationships    []Relationship `json:"relationships"`
}

// CreateTableRequest 创建表请求
type CreateTableRequest struct {
	Name    string `json:"name"`
	Schema  string `json:"schema,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// UpdateTableRequest 更新表请求
type UpdateTableRequest struct {
	Name                 string       `json:"name,omitempty"`
	Schema               string       `json:"schema,omitempty"`
	RLSEnabled           *bool        `json:"rls_enabled,omitempty"`
	RLSForced            *bool        `json:"rls_forced,omitempty"`
	ReplicaIdentity      string       `json:"replica_identity,omitempty"`
	ReplicaIdentityIndex string       `json:"replica_identity_index,omitempty"`
	PrimaryKeys          []PrimaryKey `json:"primary_keys,omitempty"`
	Comment              string       `json:"comment,omitempty"`
}

// Column 数据列
type Column struct {
	TableID            int         `json:"table_id"`
	Schema             string      `json:"schema"`
	Table              string      `json:"table"`
	ID                 string      `json:"id"`
	OrdinalPosition    int         `json:"ordinal_position"`
	Name               string      `json:"name"`
	DefaultValue       interface{} `json:"default_value"`
	DataType           string      `json:"data_type"`
	Format             string      `json:"format"`
	IsIdentity         bool        `json:"is_identity"`
	IdentityGeneration *string     `json:"identity_generation"`
	IsGenerated        bool        `json:"is_generated"`
	IsNullable         bool        `json:"is_nullable"`
	IsUpdatable        bool        `json:"is_updatable"`
	IsUnique           bool        `json:"is_unique"`
	Enums              []string    `json:"enums"`
	Check              *string     `json:"check"`
	Comment            *string     `json:"comment"`
}

// CreateColumnRequest 创建列请求
type CreateColumnRequest struct {
	TableID            int         `json:"table_id"`
	Name               string      `json:"name"`
	Type               string      `json:"type"`
	DefaultValue       interface{} `json:"default_value,omitempty"`
	DefaultValueFormat string      `json:"default_value_format,omitempty"`
	IsIdentity         *bool       `json:"is_identity,omitempty"`
	IdentityGeneration string      `json:"identity_generation,omitempty"`
	IsNullable         *bool       `json:"is_nullable,omitempty"`
	IsPrimaryKey       *bool       `json:"is_primary_key,omitempty"`
	IsUnique           *bool       `json:"is_unique,omitempty"`
	Comment            string      `json:"comment,omitempty"`
	Check              string      `json:"check,omitempty"`
}

// UpdateColumnRequest 更新列请求
type UpdateColumnRequest struct {
	Name               string      `json:"name,omitempty"`
	Type               string      `json:"type,omitempty"`
	DropDefault        *bool       `json:"drop_default,omitempty"`
	DefaultValue       interface{} `json:"default_value,omitempty"`
	DefaultValueFormat string      `json:"default_value_format,omitempty"`
	IsIdentity         *bool       `json:"is_identity,omitempty"`
	IdentityGeneration string      `json:"identity_generation,omitempty"`
	IsNullable         *bool       `json:"is_nullable,omitempty"`
	IsUnique           *bool       `json:"is_unique,omitempty"`
	Comment            string      `json:"comment,omitempty"`
	Check              *string     `json:"check,omitempty"`
}

// PrimaryKey 主键
type PrimaryKey struct {
	Schema    string `json:"schema"`
	TableName string `json:"table_name"`
	Name      string `json:"name"`
	TableID   int    `json:"table_id"`
}

// Relationship 关系
type Relationship struct {
	ID                int    `json:"id"`
	ConstraintName    string `json:"constraint_name"`
	SourceSchema      string `json:"source_schema"`
	SourceTableName   string `json:"source_table_name"`
	SourceColumnName  string `json:"source_column_name"`
	TargetTableSchema string `json:"target_table_schema"`
	TargetTableName   string `json:"target_table_name"`
	TargetColumnName  string `json:"target_column_name"`
}

// Role 数据库角色
type Role struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	IsSuperuser       bool                   `json:"is_superuser"`
	CanCreateDB       bool                   `json:"can_create_db"`
	CanCreateRole     bool                   `json:"can_create_role"`
	InheritRole       bool                   `json:"inherit_role"`
	CanLogin          bool                   `json:"can_login"`
	IsReplicationRole bool                   `json:"is_replication_role"`
	CanBypassRLS      bool                   `json:"can_bypass_rls"`
	ActiveConnections int                    `json:"active_connections"`
	ConnectionLimit   int                    `json:"connection_limit"`
	Password          string                 `json:"password"`
	ValidUntil        *string                `json:"valid_until"`
	Config            map[string]interface{} `json:"config"`
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name              string            `json:"name"`
	Password          string            `json:"password,omitempty"`
	InheritRole       *bool             `json:"inherit_role,omitempty"`
	CanLogin          *bool             `json:"can_login,omitempty"`
	IsSuperuser       *bool             `json:"is_superuser,omitempty"`
	CanCreateDB       *bool             `json:"can_create_db,omitempty"`
	CanCreateRole     *bool             `json:"can_create_role,omitempty"`
	IsReplicationRole *bool             `json:"is_replication_role,omitempty"`
	CanBypassRLS      *bool             `json:"can_bypass_rls,omitempty"`
	ConnectionLimit   *int              `json:"connection_limit,omitempty"`
	MemberOf          []string          `json:"member_of,omitempty"`
	Members           []string          `json:"members,omitempty"`
	Admins            []string          `json:"admins,omitempty"`
	ValidUntil        string            `json:"valid_until,omitempty"`
	Config            map[string]string `json:"config,omitempty"`
}

// View 视图
type View struct {
	ID          int      `json:"id"`
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	IsUpdatable bool     `json:"is_updatable"`
	Comment     *string  `json:"comment"`
	Columns     []Column `json:"columns,omitempty"`
}

// MaterializedView 物化视图
type MaterializedView struct {
	ID          int      `json:"id"`
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	IsPopulated bool     `json:"is_populated"`
	Comment     *string  `json:"comment"`
	Columns     []Column `json:"columns,omitempty"`
}

// ForeignTable 外部表
type ForeignTable struct {
	ID      int      `json:"id"`
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Comment *string  `json:"comment"`
	Columns []Column `json:"columns,omitempty"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	Query string `json:"query"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Rows    []map[string]interface{} `json:"rows,omitempty"`
	Columns []string                 `json:"columns,omitempty"`
	Error   string                   `json:"error,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// ==================== 通用方法 ====================

// makeRequest 发送HTTP请求
func (c *PgMetaClient) makeRequest(method, path string, body interface{}) ([]byte, error) {
	fullURL := c.BaseURL + path
	log.Printf("[DEBUG] PgMetaClient.makeRequest - 开始请求: %s %s", method, fullURL)
	log.Printf("[DEBUG] PgMetaClient.makeRequest - pg header: %s", c.PgHeader)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			log.Printf("[ERROR] PgMetaClient.makeRequest - 序列化请求体失败: %v", err)
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		log.Printf("[DEBUG] PgMetaClient.makeRequest - 请求体: %s", string(jsonBody))
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		log.Printf("[ERROR] PgMetaClient.makeRequest - 创建请求失败: %v", err)
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("pg", c.PgHeader)

	log.Printf("[DEBUG] PgMetaClient.makeRequest - 发送HTTP请求...")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] PgMetaClient.makeRequest - 发送请求失败: %v", err)
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] PgMetaClient.makeRequest - 响应状态码: %d", resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] PgMetaClient.makeRequest - 读取响应失败: %v", err)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("[DEBUG] PgMetaClient.makeRequest - 响应体长度: %d bytes", len(respBody))
	if len(respBody) > 0 && len(respBody) < 1000 {
		log.Printf("[DEBUG] PgMetaClient.makeRequest - 响应体内容: %s", string(respBody))
	}

	if resp.StatusCode >= 400 {
		log.Printf("[ERROR] PgMetaClient.makeRequest - HTTP错误状态: %d", resp.StatusCode)
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			log.Printf("[ERROR] PgMetaClient.makeRequest - 解析到API错误: %s", errResp.Error)
			return nil, fmt.Errorf("API错误 [%d]: %s", resp.StatusCode, errResp.Error)
		}
		log.Printf("[ERROR] PgMetaClient.makeRequest - 原始错误响应: %s", string(respBody))
		return nil, fmt.Errorf("HTTP错误 [%d]: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[DEBUG] PgMetaClient.makeRequest - 请求成功完成")
	return respBody, nil
}

// buildQueryParams 构建查询参数
func buildQueryParams(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range params {
		if value != nil {
			switch v := value.(type) {
			case string:
				if v != "" {
					values.Add(key, v)
				}
			case int:
				values.Add(key, strconv.Itoa(v))
			case bool:
				values.Add(key, strconv.FormatBool(v))
			}
		}
	}

	if len(values) > 0 {
		return "?" + values.Encode()
	}
	return ""
}

// ==================== Schema 相关方法 ====================

// ListSchemas 获取所有模式
func (c *PgMetaClient) ListSchemas(includeSystemSchemas *bool, limit, offset *int) ([]Schema, error) {
	params := map[string]interface{}{
		"include_system_schemas": includeSystemSchemas,
		"limit":                  limit,
		"offset":                 offset,
	}

	respBody, err := c.makeRequest("GET", "/schemas/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var schemas []Schema
	if err := json.Unmarshal(respBody, &schemas); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return schemas, nil
}

// GetSchema 根据ID获取模式
func (c *PgMetaClient) GetSchema(id int) (*Schema, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/schemas/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := json.Unmarshal(respBody, &schema); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &schema, nil
}

// CreateSchema 创建新模式
func (c *PgMetaClient) CreateSchema(req CreateSchemaRequest) (*Schema, error) {
	respBody, err := c.makeRequest("POST", "/schemas/", req)
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := json.Unmarshal(respBody, &schema); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &schema, nil
}

// UpdateSchema 更新模式
func (c *PgMetaClient) UpdateSchema(id int, req UpdateSchemaRequest) (*Schema, error) {
	respBody, err := c.makeRequest("PATCH", fmt.Sprintf("/schemas/%d", id), req)
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := json.Unmarshal(respBody, &schema); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &schema, nil
}

// DeleteSchema 删除模式
func (c *PgMetaClient) DeleteSchema(id int, cascade *bool) (*Schema, error) {
	params := map[string]interface{}{
		"cascade": cascade,
	}

	respBody, err := c.makeRequest("DELETE", fmt.Sprintf("/schemas/%d", id)+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := json.Unmarshal(respBody, &schema); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &schema, nil
}

// ==================== Table 相关方法 ====================

// ListTables 获取所有表
func (c *PgMetaClient) ListTables(includeSystemSchemas *bool, includedSchemas, excludedSchemas string, limit, offset *int, includeColumns *bool) ([]Table, error) {
	params := map[string]interface{}{
		"include_system_schemas": includeSystemSchemas,
		"included_schemas":       includedSchemas,
		"excluded_schemas":       excludedSchemas,
		"limit":                  limit,
		"offset":                 offset,
		"include_columns":        includeColumns,
	}

	respBody, err := c.makeRequest("GET", "/tables/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var tables []Table
	if err := json.Unmarshal(respBody, &tables); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return tables, nil
}

// GetTable 根据ID获取表
func (c *PgMetaClient) GetTable(id int) (*Table, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/tables/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(respBody, &table); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &table, nil
}

// CreateTable 创建新表
func (c *PgMetaClient) CreateTable(req CreateTableRequest) (*Table, error) {
	respBody, err := c.makeRequest("POST", "/tables/", req)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(respBody, &table); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &table, nil
}

// UpdateTable 更新表
func (c *PgMetaClient) UpdateTable(id int, req UpdateTableRequest) (*Table, error) {
	respBody, err := c.makeRequest("PATCH", fmt.Sprintf("/tables/%d", id), req)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(respBody, &table); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &table, nil
}

// DeleteTable 删除表
func (c *PgMetaClient) DeleteTable(id int, cascade *bool) (*Table, error) {
	params := map[string]interface{}{
		"cascade": cascade,
	}

	respBody, err := c.makeRequest("DELETE", fmt.Sprintf("/tables/%d", id)+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(respBody, &table); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &table, nil
}

// ==================== Column 相关方法 ====================

// ListColumns 获取所有列
func (c *PgMetaClient) ListColumns(includeSystemSchemas *bool, includedSchemas, excludedSchemas string, limit, offset *int) ([]Column, error) {
	params := map[string]interface{}{
		"include_system_schemas": includeSystemSchemas,
		"included_schemas":       includedSchemas,
		"excluded_schemas":       excludedSchemas,
		"limit":                  limit,
		"offset":                 offset,
	}

	respBody, err := c.makeRequest("GET", "/columns/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var columns []Column
	if err := json.Unmarshal(respBody, &columns); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return columns, nil
}

// GetColumn 根据表ID和序号获取列
func (c *PgMetaClient) GetColumn(tableID, ordinalPosition int, includeSystemSchemas *bool, limit, offset *int) (*Column, error) {
	params := map[string]interface{}{
		"include_system_schemas": includeSystemSchemas,
		"limit":                  limit,
		"offset":                 offset,
	}

	respBody, err := c.makeRequest("GET", fmt.Sprintf("/columns/%d%d", tableID, ordinalPosition)+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var column Column
	if err := json.Unmarshal(respBody, &column); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &column, nil
}

// CreateColumn 创建新列
func (c *PgMetaClient) CreateColumn(req CreateColumnRequest) (*Column, error) {
	respBody, err := c.makeRequest("POST", "/columns/", req)
	if err != nil {
		return nil, err
	}

	var column Column
	if err := json.Unmarshal(respBody, &column); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &column, nil
}

// UpdateColumn 更新列
func (c *PgMetaClient) UpdateColumn(id string, req UpdateColumnRequest) (*Column, error) {
	respBody, err := c.makeRequest("PATCH", fmt.Sprintf("/columns/%s", id), req)
	if err != nil {
		return nil, err
	}

	var column Column
	if err := json.Unmarshal(respBody, &column); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &column, nil
}

// DeleteColumn 删除列
func (c *PgMetaClient) DeleteColumn(id string, cascade *string) (*Column, error) {
	params := map[string]interface{}{
		"cascade": cascade,
	}

	respBody, err := c.makeRequest("DELETE", fmt.Sprintf("/columns/%s", id)+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var column Column
	if err := json.Unmarshal(respBody, &column); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &column, nil
}

// ==================== Role 相关方法 ====================

// ListRoles 获取所有角色
func (c *PgMetaClient) ListRoles(includeSystemSchemas, limit, offset *string) ([]Role, error) {
	params := map[string]interface{}{
		"include_system_schemas": includeSystemSchemas,
		"limit":                  limit,
		"offset":                 offset,
	}

	respBody, err := c.makeRequest("GET", "/roles/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var roles []Role
	if err := json.Unmarshal(respBody, &roles); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return roles, nil
}

// GetRole 根据ID获取角色
func (c *PgMetaClient) GetRole(id string) (*Role, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/roles/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var role Role
	if err := json.Unmarshal(respBody, &role); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &role, nil
}

// CreateRole 创建新角色
func (c *PgMetaClient) CreateRole(req CreateRoleRequest) (*Role, error) {
	respBody, err := c.makeRequest("POST", "/roles/", req)
	if err != nil {
		return nil, err
	}

	var role Role
	if err := json.Unmarshal(respBody, &role); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &role, nil
}

// DeleteRole 删除角色
func (c *PgMetaClient) DeleteRole(id string, cascade *string) (*Role, error) {
	params := map[string]interface{}{
		"cascade": cascade,
	}

	respBody, err := c.makeRequest("DELETE", fmt.Sprintf("/roles/%s", id)+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var role Role
	if err := json.Unmarshal(respBody, &role); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &role, nil
}

// ==================== View 相关方法 ====================

// ListViews 获取所有视图
func (c *PgMetaClient) ListViews(includeSystemSchemas *bool, includedSchemas, excludedSchemas string, limit, offset *int, includeColumns *bool) ([]View, error) {
	params := map[string]interface{}{
		"include_system_schemas": includeSystemSchemas,
		"included_schemas":       includedSchemas,
		"excluded_schemas":       excludedSchemas,
		"limit":                  limit,
		"offset":                 offset,
		"include_columns":        includeColumns,
	}

	respBody, err := c.makeRequest("GET", "/views/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var views []View
	if err := json.Unmarshal(respBody, &views); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return views, nil
}

// GetView 根据ID获取视图
func (c *PgMetaClient) GetView(id int) (*View, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/views/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var view View
	if err := json.Unmarshal(respBody, &view); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &view, nil
}

// ==================== MaterializedView 相关方法 ====================

// ListMaterializedViews 获取所有物化视图
func (c *PgMetaClient) ListMaterializedViews(includedSchemas, excludedSchemas string, limit, offset *int, includeColumns *bool) ([]MaterializedView, error) {
	params := map[string]interface{}{
		"included_schemas": includedSchemas,
		"excluded_schemas": excludedSchemas,
		"limit":            limit,
		"offset":           offset,
		"include_columns":  includeColumns,
	}

	respBody, err := c.makeRequest("GET", "/materialized-views/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var views []MaterializedView
	if err := json.Unmarshal(respBody, &views); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return views, nil
}

// GetMaterializedView 根据ID获取物化视图
func (c *PgMetaClient) GetMaterializedView(id int) (*MaterializedView, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/materialized-views/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var view MaterializedView
	if err := json.Unmarshal(respBody, &view); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &view, nil
}

// ==================== ForeignTable 相关方法 ====================

// ListForeignTables 获取所有外部表
func (c *PgMetaClient) ListForeignTables(limit, offset *int, includeColumns *bool) ([]ForeignTable, error) {
	params := map[string]interface{}{
		"limit":           limit,
		"offset":          offset,
		"include_columns": includeColumns,
	}

	respBody, err := c.makeRequest("GET", "/foreign-tables/"+buildQueryParams(params), nil)
	if err != nil {
		return nil, err
	}

	var tables []ForeignTable
	if err := json.Unmarshal(respBody, &tables); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return tables, nil
}

// GetForeignTable 根据ID获取外部表
func (c *PgMetaClient) GetForeignTable(id int) (*ForeignTable, error) {
	respBody, err := c.makeRequest("GET", fmt.Sprintf("/foreign-tables/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var table ForeignTable
	if err := json.Unmarshal(respBody, &table); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &table, nil
}

// ==================== Query 相关方法 ====================

// ExecuteQuery 执行SQL查询
func (c *PgMetaClient) ExecuteQuery(query string) (*QueryResponse, error) {
	req := QueryRequest{Query: query}

	respBody, err := c.makeRequest("POST", "/query/", req)
	if err != nil {
		return nil, err
	}

	var response QueryResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
}

// FormatQuery 格式化SQL查询
func (c *PgMetaClient) FormatQuery(query string) (string, error) {
	req := QueryRequest{Query: query}

	respBody, err := c.makeRequest("POST", "/query/format", req)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if formattedQuery, ok := response["formatted_query"].(string); ok {
		return formattedQuery, nil
	}

	return "", fmt.Errorf("格式化查询响应格式错误")
}

// ParseQuery 解析SQL查询
func (c *PgMetaClient) ParseQuery(query string) (map[string]interface{}, error) {
	req := QueryRequest{Query: query}

	respBody, err := c.makeRequest("POST", "/query/parse", req)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return response, nil
}

// DeparseQuery 反解析SQL查询
func (c *PgMetaClient) DeparseQuery(ast map[string]interface{}) (string, error) {
	respBody, err := c.makeRequest("POST", "/query/deparse", ast)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if query, ok := response["query"].(string); ok {
		return query, nil
	}

	return "", fmt.Errorf("反解析查询响应格式错误")
}
