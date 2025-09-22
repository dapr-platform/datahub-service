/*
 * @module service/basic_library/schema_service
 * @description 表结构管理服务，通过PgMetaClient客户端动态操作数据库表结构
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 表结构操作请求 -> PgMetaClient调用 -> 结果验证 -> 状态更新
 * @rules 确保表结构操作的安全性和一致性
 * @dependencies datahub-service/service/models, datahub-service/client, gorm.io/gorm
 * @refs ai_docs/interfaces.md
 */

package database

import (
	"datahub-service/client"
	"datahub-service/service/models"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SchemaService 表结构管理服务
type SchemaService struct {
	db       *gorm.DB
	pgClient *client.PgMetaClient
}

// NewSchemaService 创建表结构管理服务实例
func NewSchemaService(db *gorm.DB) *SchemaService {
	baseURL := "http://localhost:3001" // 默认postgres-meta服务地址

	// 检查是否在Dapr环境中
	if daprPort := os.Getenv("DAPR_HTTP_PORT"); daprPort != "" {
		baseURL = fmt.Sprintf("http://localhost:%s/v1.0/invoke/postgres-meta/method", daprPort)
		log.Printf("[DEBUG] SchemaService - 使用Dapr环境，baseURL: %s", baseURL)
	} else {
		log.Printf("[DEBUG] SchemaService - 使用默认环境，baseURL: %s", baseURL)
	}

	// 从环境变量获取数据库连接信息用于构建pg header
	pgHeader := "default"
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "postgres"
		}
		dbUser := os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "postgres"
		}
		dbPassword := os.Getenv("DB_PASSWORD")
		if dbPassword == "" {
			dbPassword = "postgres"
		}

		// 构建PostgreSQL连接字符串用作pg header
		pgHeader = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
		log.Printf("[DEBUG] SchemaService - 构建pg header: postgresql://%s:***@%s:%s/%s", dbUser, dbHost, dbPort, dbName)
	} else {
		log.Printf("[DEBUG] SchemaService - 使用默认pg header: %s", pgHeader)
	}

	// 创建PgMeta客户端
	pgClient := client.NewPgMetaClient(baseURL, pgHeader)
	log.Printf("[DEBUG] SchemaService - PgMeta客户端创建成功")

	return &SchemaService{
		db:       db,
		pgClient: pgClient,
	}
}

// TableDefinition 表定义结构（保持兼容性）
type TableDefinition struct {
	Name        string                 `json:"name"`
	Schema      string                 `json:"schema"`
	Comment     string                 `json:"comment,omitempty"`
	Columns     []ColumnDefinition     `json:"columns"`
	Constraints []ConstraintDefinition `json:"constraints,omitempty"`
}

// ColumnDefinition 列定义结构（保持兼容性）
type ColumnDefinition struct {
	Name         string      `json:"name"`
	DataType     string      `json:"data_type"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Comment      string      `json:"comment,omitempty"`
	IsPrimaryKey bool        `json:"is_primary_key,omitempty"`
	IsUnique     bool        `json:"is_unique,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty"`
}

// ConstraintDefinition 约束定义结构（保持兼容性）
type ConstraintDefinition struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // primary_key, foreign_key, unique, check
	Columns []string `json:"columns"`
	Check   string   `json:"check,omitempty"`
}

// ManageTableSchema 管理表结构
func (s *SchemaService) ManageTableSchema(interfaceID, operation, schemaName, tableName string, fields []models.TableField) error {

	switch operation {
	case "create_table":
		return s.createTable(schemaName, tableName, fields)
	case "alter_table":
		return s.alterTable(schemaName, tableName, fields)
	case "drop_table":
		return s.dropTable(schemaName, tableName)
	default:
		return fmt.Errorf("不支持的操作类型: %s", operation)
	}

}

// ManageViewSchema 管理视图结构
func (s *SchemaService) ManageViewSchema(interfaceID, operation, schemaName, viewName, viewSQL string) error {
	switch operation {
	case "create_view":
		return s.createView(schemaName, viewName, viewSQL)
	case "update_view":
		return s.updateView(schemaName, viewName, viewSQL)
	case "drop_view":
		return s.dropView(schemaName, viewName)
	default:
		return fmt.Errorf("不支持的视图操作类型: %s", operation)
	}
}

// createTable 创建表
func (s *SchemaService) createTable(schemaName, tableName string, fields []models.TableField) error {
	// 首先创建表
	createReq := client.CreateTableRequest{
		Name:    tableName,
		Schema:  schemaName,
		Comment: fmt.Sprintf("数据接口表: %s", tableName),
	}

	table, err := s.pgClient.CreateTable(createReq)
	if err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	// 然后添加列
	for _, field := range fields {
		dataType := s.mapDataType(field.DataType)
		columnReq := client.CreateColumnRequest{
			TableID:      table.ID,
			Name:         field.NameEn,
			Type:         s.buildColumnTypeWithLength(dataType, field),
			IsNullable:   &[]bool{field.IsNullable}[0],
			IsPrimaryKey: &[]bool{field.IsPrimaryKey}[0],
			Comment:      fmt.Sprintf("%s - %s", field.NameZh, field.Description),
		}

		// 设置默认值，需要验证和转换
		if defaultValue := s.processDefaultValue(field, dataType); defaultValue != nil {
			columnReq.DefaultValue = defaultValue
		}

		// 设置唯一性
		if field.IsUnique {
			columnReq.IsUnique = &[]bool{field.IsUnique}[0]
		}

		_, err := s.pgClient.CreateColumn(columnReq)
		if err != nil {
			return fmt.Errorf("创建列 %s 失败: %v", columnReq.Name, err)
		}
	}

	return nil
}

// alterTable 修改表结构
func (s *SchemaService) alterTable(schemaName, tableName string, fields []models.TableField) error {
	// 获取当前表信息
	tables, err := s.pgClient.ListTables(nil, schemaName, "", nil, nil, &[]bool{true}[0])
	if err != nil {
		return fmt.Errorf("获取表列表失败: %v", err)
	}

	var currentTable *client.Table
	for _, table := range tables {
		if table.Name == tableName && table.Schema == schemaName {
			currentTable = &table
			break
		}
	}

	if currentTable == nil {
		return fmt.Errorf("表 %s.%s 不存在", schemaName, tableName)
	}

	// 比较差异并生成修改操作
	alterOperations := s.generateAlterOperations(currentTable, fields)

	// 执行修改操作
	for _, operation := range alterOperations {
		if err := s.executeAlterOperation(currentTable.ID, operation); err != nil {
			return fmt.Errorf("执行表结构修改失败: %v", err)
		}
	}

	return nil
}

// dropTable 删除表
func (s *SchemaService) dropTable(schemaName, tableName string) error {
	// 获取表信息
	tables, err := s.pgClient.ListTables(nil, schemaName, "", nil, nil, nil)
	if err != nil {
		return fmt.Errorf("获取表列表失败: %v", err)
	}

	var tableID int
	for _, table := range tables {
		if table.Name == tableName && table.Schema == schemaName {
			tableID = table.ID
			break
		}
	}

	if tableID == 0 {

		//return fmt.Errorf("表 %s.%s 不存在", schemaName, tableName)
		return nil
	}

	_, err = s.pgClient.DeleteTable(tableID, nil)
	return err
}

// buildColumnDefinitions 构建列定义（保持兼容性）
func (s *SchemaService) buildColumnDefinitions(fields []map[string]interface{}) []ColumnDefinition {
	columns := make([]ColumnDefinition, 0, len(fields))

	for _, field := range fields {
		column := ColumnDefinition{
			Name:         field["name_en"].(string),
			DataType:     s.mapDataType(field["data_type"].(string)),
			IsNullable:   field["is_nullable"].(bool),
			Comment:      fmt.Sprintf("%s - %s", field["name_zh"].(string), field["description"].(string)),
			IsPrimaryKey: field["is_primary_key"].(bool),
		}

		// 设置默认值
		if defaultValue, exists := field["default_value"]; exists && defaultValue != nil {
			column.DefaultValue = defaultValue
		}

		// 设置字符串长度限制
		if column.DataType == "varchar" || column.DataType == "character varying" {
			maxLength := 255 // 默认长度
			column.MaxLength = &maxLength
		}

		columns = append(columns, column)
	}

	return columns
}

// buildConstraints 构建约束（保持兼容性）
func (s *SchemaService) buildConstraints(fields []map[string]interface{}) []ConstraintDefinition {
	constraints := make([]ConstraintDefinition, 0)

	// 收集主键字段
	primaryKeyColumns := make([]string, 0)
	for _, field := range fields {
		if field["is_primary_key"].(bool) {
			primaryKeyColumns = append(primaryKeyColumns, field["name_en"].(string))
		}
	}

	// 添加主键约束
	if len(primaryKeyColumns) > 0 {
		constraint := ConstraintDefinition{
			Name:    "pk_" + fields[0]["interface_id"].(string)[:8], // 使用接口ID的前8位作为约束名
			Type:    "primary_key",
			Columns: primaryKeyColumns,
		}
		constraints = append(constraints, constraint)
	}

	return constraints
}

// mapDataType 映射数据类型到PostgreSQL Meta API支持的类型
func (s *SchemaService) mapDataType(dataType string) string {
	typeMap := map[string]string{
		"integer":   "integer",
		"int":       "integer",
		"bigint":    "bigint",
		"smallint":  "smallint",
		"string":    "varchar",
		"varchar":   "varchar",
		"text":      "text",
		"boolean":   "boolean",
		"bool":      "boolean",
		"datetime":  "timestamp",
		"timestamp": "timestamp",
		"date":      "date",
		"time":      "time",
		"decimal":   "numeric",
		"numeric":   "numeric",
		"float":     "real",
		"real":      "real",
		"double":    "float8", // 使用float8而不是double precision
		"json":      "json",
		"jsonb":     "jsonb",
		"uuid":      "uuid",
		"inet":      "inet",
		"cidr":      "cidr",
		"macaddr":   "macaddr",
		"bytea":     "bytea",
		"money":     "money",
		"interval":  "interval",
		"point":     "point",
		"line":      "line",
		"box":       "box",
		"circle":    "circle",
	}

	if pgType, exists := typeMap[dataType]; exists {
		return pgType
	}

	return "varchar" // 默认类型
}

// buildColumnTypeWithLength 构建PostgreSQL Meta API兼容的列类型
func (s *SchemaService) buildColumnTypeWithLength(dataType string, field models.TableField) string {
	// PostgreSQL Meta API不支持带参数的类型定义
	// 基于测试结果，我们需要使用基础类型名
	switch dataType {
	case "varchar", "character varying":
		// PostgreSQL Meta API只支持无参数的varchar
		return "varchar"
	case "decimal", "numeric":
		// PostgreSQL Meta API只支持无参数的numeric
		return "numeric"
	case "double precision":
		// 有空格的类型名不支持，使用float8
		return "float8"
	case "timestamp":
		// 只支持简单的timestamp，不支持with/without time zone
		return "timestamp"
	case "time":
		// 只支持简单的time，不支持with/without time zone
		return "time"
	default:
		return dataType
	}
}

// processDefaultValue 处理默认值，确保类型兼容性
func (s *SchemaService) processDefaultValue(field models.TableField, dataType string) interface{} {
	defaultValueInterface := field.DefaultValue

	// 如果默认值为nil，直接返回nil
	if defaultValueInterface == "" {
		return nil
	}

	// 转换为字符串进行处理
	defaultValue := defaultValueInterface

	// 如果是空字符串，根据数据类型决定是否设置默认值
	if defaultValue == "" {
		switch dataType {
		case "timestamp", "date", "time":
			// 时间类型的空字符串不设置默认值
			return nil
		case "boolean":
			// 布尔类型的空字符串不设置默认值
			return nil
		case "integer", "bigint", "real", "double precision", "decimal":
			// 数值类型的空字符串不设置默认值
			return nil
		case "uuid":
			// UUID类型的空字符串不设置默认值
			return nil
		case "inet":
			// 网络地址类型的空字符串不设置默认值
			return nil
		default:
			// 字符串类型可以设置空字符串作为默认值
			return "''"
		}
	}

	// 根据数据类型格式化默认值
	switch dataType {
	case "varchar", "text":
		// 字符串类型需要加引号
		return fmt.Sprintf("'%s'", defaultValue)
	case "boolean":
		// 布尔类型转换
		switch strings.ToLower(defaultValue) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		default:
			// 无效的布尔值，不设置默认值
			return nil
		}
	case "timestamp":
		// 时间戳类型验证
		if defaultValue == "now()" || defaultValue == "CURRENT_TIMESTAMP" {
			return defaultValue
		}
		// 尝试解析时间格式
		if _, err := time.Parse("2006-01-02 15:04:05", defaultValue); err == nil {
			return fmt.Sprintf("'%s'", defaultValue)
		}
		if _, err := time.Parse("2006-01-02T15:04:05Z07:00", defaultValue); err == nil {
			return fmt.Sprintf("'%s'", defaultValue)
		}
		// 无效的时间格式，不设置默认值
		return nil
	case "date":
		// 日期类型验证
		if _, err := time.Parse("2006-01-02", defaultValue); err == nil {
			return fmt.Sprintf("'%s'", defaultValue)
		}
		// 无效的日期格式，不设置默认值
		return nil
	case "uuid":
		// UUID类型的默认值处理
		// 根据测试结果，PostgreSQL Meta API不接受函数作为默认值字符串
		// 对于gen_random_uuid()这样的函数，我们不设置默认值
		if defaultValue == "uuid_generate_v4()" || defaultValue == "gen_random_uuid()" {
			// 这些函数值会导致错误，暂时不设置默认值
			return nil
		}
		// 验证UUID格式 (简单验证)
		if len(defaultValue) == 36 && strings.Count(defaultValue, "-") == 4 {
			return fmt.Sprintf("'%s'", defaultValue)
		}
		// 无效的UUID格式，不设置默认值
		return nil
	default:
		// 其他类型直接返回
		return defaultValue
	}
}

// getTableStructure 获取表结构（保持兼容性）
func (s *SchemaService) getTableStructure(schemaName, tableName string) (*TableDefinition, error) {
	// 使用PgMetaClient获取表信息
	tables, err := s.pgClient.ListTables(nil, schemaName, "", nil, nil, &[]bool{true}[0])
	if err != nil {
		return nil, fmt.Errorf("获取表列表失败: %v", err)
	}

	var table *client.Table
	for _, t := range tables {
		if t.Name == tableName && t.Schema == schemaName {
			table = &t
			break
		}
	}

	if table == nil {
		return nil, fmt.Errorf("表 %s.%s 不存在", schemaName, tableName)
	}

	// 转换为本地TableDefinition格式
	tableDef := &TableDefinition{
		Name:    table.Name,
		Schema:  table.Schema,
		Columns: make([]ColumnDefinition, 0, len(table.Columns)),
	}

	if table.Comment != nil {
		tableDef.Comment = *table.Comment
	}

	// 转换列信息
	for _, col := range table.Columns {
		column := ColumnDefinition{
			Name:       col.Name,
			DataType:   col.DataType,
			IsNullable: col.IsNullable,
		}

		if col.DefaultValue != nil {
			column.DefaultValue = col.DefaultValue
		}

		if col.Comment != nil {
			column.Comment = *col.Comment
		}

		column.IsUnique = col.IsUnique

		tableDef.Columns = append(tableDef.Columns, column)
	}

	return tableDef, nil
}

// AlterOperation 表结构修改操作
type AlterOperation struct {
	Type   string      `json:"type"`   // add_column, drop_column, modify_column
	Column interface{} `json:"column"` // 列定义或列名
}

// generateAlterOperations 生成表结构修改操作
func (s *SchemaService) generateAlterOperations(currentTable *client.Table, newFields []models.TableField) []AlterOperation {
	operations := make([]AlterOperation, 0)

	// 构建新字段映射
	newFieldsMap := make(map[string]models.TableField)
	for _, field := range newFields {
		newFieldsMap[field.NameEn] = field
	}

	// 构建当前字段映射
	currentFieldsMap := make(map[string]client.Column)
	for _, column := range currentTable.Columns {
		currentFieldsMap[column.Name] = column
	}

	// 构建主键列映射
	primaryKeyColumns := make(map[string]bool)
	for _, pk := range currentTable.PrimaryKeys {
		primaryKeyColumns[pk.Name] = true
	}

	// 检查需要添加的字段
	for fieldName, field := range newFieldsMap {
		if _, exists := currentFieldsMap[fieldName]; !exists {
			// 需要添加的字段
			dataType := s.mapDataType(field.DataType)
			columnReq := client.CreateColumnRequest{
				TableID:      currentTable.ID,
				Name:         field.NameEn,
				Type:         s.buildColumnTypeWithLength(dataType, field),
				IsNullable:   &[]bool{field.IsNullable}[0],
				IsPrimaryKey: &[]bool{field.IsPrimaryKey}[0],
				Comment:      fmt.Sprintf("%s - %s", field.NameZh, field.Description),
			}

			// 处理默认值
			if defaultValue := s.processDefaultValue(field, dataType); defaultValue != nil {
				columnReq.DefaultValue = defaultValue
			}

			operations = append(operations, AlterOperation{
				Type:   "add_column",
				Column: columnReq,
			})
		}
	}

	// 检查需要删除的字段
	for fieldName, column := range currentFieldsMap {
		if _, exists := newFieldsMap[fieldName]; !exists {
			// 需要删除的字段
			operations = append(operations, AlterOperation{
				Type:   "drop_column",
				Column: column.ID,
			})
		}
	}

	// 检查需要修改的字段
	for fieldName, field := range newFieldsMap {
		if currentColumn, exists := currentFieldsMap[fieldName]; exists {
			newDataType := s.mapDataType(field.DataType)
			newTypeWithLength := s.buildColumnTypeWithLength(newDataType, field)

			// 检查是否需要修改
			needsUpdate := false
			updateReq := client.UpdateColumnRequest{
				Name:    field.NameEn,
				Comment: fmt.Sprintf("%s - %s", field.NameZh, field.Description),
			}

			// 检查数据类型是否需要修改
			if currentColumn.DataType != newTypeWithLength {
				updateReq.Type = newTypeWithLength
				needsUpdate = true
			}

			// 检查可空性是否需要修改（但跳过主键列）
			if !primaryKeyColumns[currentColumn.Name] && currentColumn.IsNullable != field.IsNullable {
				updateReq.IsNullable = &[]bool{field.IsNullable}[0]
				needsUpdate = true
			}

			// 处理默认值
			if defaultValue := s.processDefaultValue(field, newDataType); defaultValue != nil {
				updateReq.DefaultValue = defaultValue
				needsUpdate = true
			}

			// 只有在需要更新时才添加操作
			if needsUpdate {
				operations = append(operations, AlterOperation{
					Type: "modify_column",
					Column: map[string]interface{}{
						"id":      currentColumn.ID,
						"request": updateReq,
					},
				})
			}
		}
	}

	return operations
}

// executeAlterOperation 执行表结构修改操作
func (s *SchemaService) executeAlterOperation(tableID int, operation AlterOperation) error {
	log.Printf("[DEBUG] SchemaService.executeAlterOperation - 执行操作类型: %s", operation.Type)

	switch operation.Type {
	case "add_column":
		columnReq := operation.Column.(client.CreateColumnRequest)
		log.Printf("[DEBUG] SchemaService.executeAlterOperation - 添加列: %s, 类型: %s", columnReq.Name, columnReq.Type)
		_, err := s.pgClient.CreateColumn(columnReq)
		if err != nil {
			log.Printf("[ERROR] SchemaService.executeAlterOperation - 添加列失败: %v", err)
		}
		return err
	case "drop_column":
		columnID := operation.Column.(string)
		log.Printf("[DEBUG] SchemaService.executeAlterOperation - 删除列ID: %s", columnID)
		_, err := s.pgClient.DeleteColumn(columnID, nil)
		if err != nil {
			log.Printf("[ERROR] SchemaService.executeAlterOperation - 删除列失败: %v", err)
		}
		return err
	case "modify_column":
		data := operation.Column.(map[string]interface{})
		columnID := data["id"].(string)
		updateReq := data["request"].(client.UpdateColumnRequest)
		log.Printf("[DEBUG] SchemaService.executeAlterOperation - 修改列ID: %s, 名称: %s", columnID, updateReq.Name)

		// 检查是否有需要更新的字段
		hasUpdates := false
		if updateReq.Type != "" {
			log.Printf("[DEBUG] SchemaService.executeAlterOperation - 更新数据类型: %s", updateReq.Type)
			hasUpdates = true
		}
		if updateReq.IsNullable != nil {
			log.Printf("[DEBUG] SchemaService.executeAlterOperation - 更新可空性: %v", *updateReq.IsNullable)
			hasUpdates = true
		}
		if updateReq.Comment != "" {
			log.Printf("[DEBUG] SchemaService.executeAlterOperation - 更新注释: %s", updateReq.Comment)
			hasUpdates = true
		}

		if !hasUpdates {
			log.Printf("[DEBUG] SchemaService.executeAlterOperation - 没有需要更新的字段，跳过")
			return nil
		}

		_, err := s.pgClient.UpdateColumn(columnID, updateReq)
		if err != nil {
			log.Printf("[ERROR] SchemaService.executeAlterOperation - 修改列失败: %v", err)
			// 如果是主键相关的错误，提供更友好的错误信息
			if strings.Contains(err.Error(), "primary key") {
				return fmt.Errorf("无法修改主键列的约束属性，主键列必须保持非空: %v", err)
			}
		}
		return err
	default:
		return fmt.Errorf("不支持的修改操作类型: %s", operation.Type)
	}
}

// GetTableInfo 获取表信息
func (s *SchemaService) GetTableInfo(schemaName, tableName string) (*TableDefinition, error) {
	return s.getTableStructure(schemaName, tableName)
}

// ListTables 列出schema中的所有表
func (s *SchemaService) ListTables(schemaName string) ([]string, error) {
	log.Printf("[DEBUG] SchemaService.ListTables - 开始获取表列表，schemaName: %s", schemaName)

	// 调用PgMetaClient获取表列表
	tables, err := s.pgClient.ListTables(nil, schemaName, "", nil, nil, nil)
	if err != nil {
		log.Printf("[ERROR] SchemaService.ListTables - PgMetaClient调用失败: %v", err)
		return nil, fmt.Errorf("获取表列表失败: %v", err)
	}

	log.Printf("[DEBUG] SchemaService.ListTables - PgMetaClient返回了 %d 个表", len(tables))

	tableNames := make([]string, 0, len(tables))
	for i, table := range tables {
		log.Printf("[DEBUG] SchemaService.ListTables - 表[%d]: ID=%d, Name=%s, Schema=%s", i, table.ID, table.Name, table.Schema)
		tableNames = append(tableNames, table.Name)
	}

	log.Printf("[DEBUG] SchemaService.ListTables - 成功返回表名列表: %v", tableNames)
	return tableNames, nil
}

// ValidateTableName 验证表名
func (s *SchemaService) ValidateTableName(tableName string) error {
	// 表名验证规则
	if len(tableName) == 0 {
		return fmt.Errorf("表名不能为空")
	}

	if len(tableName) > 63 {
		return fmt.Errorf("表名长度不能超过63个字符")
	}

	// 检查是否以字母开头
	if !((tableName[0] >= 'a' && tableName[0] <= 'z') || (tableName[0] >= 'A' && tableName[0] <= 'Z')) {
		return fmt.Errorf("表名必须以字母开头")
	}

	// 检查是否只包含字母、数字和下划线
	for i := 1; i < len(tableName); i++ {
		c := tableName[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("表名只能包含字母、数字和下划线")
		}
	}

	return nil
}

// CreateSchema 创建Schema
func (s *SchemaService) CreateSchema(schemaName, owner string) error {
	req := client.CreateSchemaRequest{
		Name:  schemaName,
		Owner: owner,
	}

	_, err := s.pgClient.CreateSchema(req)
	return err
}

// DropSchema 删除Schema
func (s *SchemaService) DropSchema(schemaName string, cascade bool) error {
	// 首先获取schema信息
	schemas, err := s.pgClient.ListSchemas(nil, nil, nil)
	if err != nil {
		return fmt.Errorf("获取schema列表失败: %v", err)
	}

	var schemaID int
	for _, schema := range schemas {
		if schema.Name == schemaName {
			schemaID = schema.ID
			break
		}
	}

	if schemaID == 0 {
		return fmt.Errorf("schema %s 不存在", schemaName)
	}

	_, err = s.pgClient.DeleteSchema(schemaID, &cascade)
	return err
}

// ListSchemas 列出所有Schema
func (s *SchemaService) ListSchemas() ([]string, error) {
	schemas, err := s.pgClient.ListSchemas(&[]bool{false}[0], nil, nil) // 不包含系统schema
	if err != nil {
		return nil, err
	}

	schemaNames := make([]string, 0, len(schemas))
	for _, schema := range schemas {
		schemaNames = append(schemaNames, schema.Name)
	}

	return schemaNames, nil
}

// CheckTableExists 检查表是否存在
func (s *SchemaService) CheckTableExists(schemaName, tableName string) (bool, error) {
	log.Printf("[DEBUG] SchemaService.CheckTableExists - 检查表是否存在，schemaName: %s, tableName: %s", schemaName, tableName)

	// 调用PgMetaClient获取表列表
	tables, err := s.pgClient.ListTables(nil, schemaName, "", nil, nil, nil)
	if err != nil {
		log.Printf("[ERROR] SchemaService.CheckTableExists - PgMetaClient调用失败: %v", err)
		return false, fmt.Errorf("获取表列表失败: %v", err)
	}

	// 检查指定表是否存在
	for _, table := range tables {
		if table.Name == tableName && table.Schema == schemaName {
			log.Printf("[DEBUG] SchemaService.CheckTableExists - 表存在: %s.%s", schemaName, tableName)
			return true, nil
		}
	}

	log.Printf("[DEBUG] SchemaService.CheckTableExists - 表不存在: %s.%s", schemaName, tableName)
	return false, nil
}

// GetTableData 获取表数据
func (s *SchemaService) GetTableData(fullTableName string, limit, offset int) ([]map[string]interface{}, int, error) {
	// 分离schema和表名
	parts := strings.Split(fullTableName, ".")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("无效的表名格式，应为 schema.table")
	}

	schemaName := parts[0]
	tableName := parts[1]

	// 验证表是否存在
	var exists bool
	checkSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = $1 AND table_name = $2
		)`
	err := s.db.Raw(checkSQL, schemaName, tableName).Scan(&exists).Error
	if err != nil {
		return nil, 0, fmt.Errorf("检查表存在性失败: %v", err)
	}
	if !exists {
		return nil, 0, fmt.Errorf("表 %s 不存在", fullTableName)
	}

	// 获取总行数
	var totalCount int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s",
		s.db.Statement.Quote(schemaName),
		s.db.Statement.Quote(tableName))
	err = s.db.Raw(countSQL).Scan(&totalCount).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取总行数失败: %v", err)
	}

	// 获取数据
	dataSQL := fmt.Sprintf("SELECT * FROM %s.%s ORDER BY 1 LIMIT %d OFFSET %d",
		s.db.Statement.Quote(schemaName),
		s.db.Statement.Quote(tableName),
		limit, offset)

	rows, err := s.db.Raw(dataSQL).Rows()
	if err != nil {
		return nil, 0, fmt.Errorf("查询数据失败: %v", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, fmt.Errorf("获取列名失败: %v", err)
	}

	// 读取数据
	var data []map[string]interface{}
	for rows.Next() {
		// 创建存储值的切片
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, fmt.Errorf("扫描行数据失败: %v", err)
		}

		// 构建行数据映射
		rowData := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// 将字节数组转换为字符串
				rowData[col] = string(b)
			} else {
				rowData[col] = val
			}
		}
		data = append(data, rowData)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历数据失败: %v", err)
	}

	return data, int(totalCount), nil
}

// createView 创建视图
func (s *SchemaService) createView(schemaName, viewName, viewSQL string) error {
	// 验证SQL语句
	if err := s.validateViewSQL(viewSQL, schemaName, viewName); err != nil {
		return fmt.Errorf("视图SQL验证失败: %v", err)
	}

	// 构造完整的CREATE OR REPLACE VIEW语句
	fullSQL := fmt.Sprintf("CREATE OR REPLACE VIEW %s.%s AS %s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(viewName),
		s.extractSelectFromViewSQL(viewSQL))

	// 执行SQL
	if err := s.db.Exec(fullSQL).Error; err != nil {
		return fmt.Errorf("创建视图失败: %v", err)
	}

	return nil
}

// updateView 更新视图
func (s *SchemaService) updateView(schemaName, viewName, viewSQL string) error {
	// 更新视图实际上就是重新创建视图
	return s.createView(schemaName, viewName, viewSQL)
}

// dropView 删除视图
func (s *SchemaService) dropView(schemaName, viewName string) error {
	dropSQL := fmt.Sprintf("DROP VIEW IF EXISTS %s.%s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(viewName))

	if err := s.db.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("删除视图失败: %v", err)
	}

	return nil
}

// CheckViewExists 检查视图是否存在
func (s *SchemaService) CheckViewExists(schemaName, viewName string) (bool, error) {
	var count int64
	query := `
		SELECT COUNT(*) 
		FROM information_schema.views 
		WHERE table_schema = ? AND table_name = ?
	`

	if err := s.db.Raw(query, schemaName, viewName).Count(&count).Error; err != nil {
		return false, fmt.Errorf("检查视图存在性失败: %v", err)
	}

	return count > 0, nil
}

// validateViewSQL 验证视图SQL语句
func (s *SchemaService) validateViewSQL(viewSQL, schemaName, viewName string) error {
	// 基本验证：检查SQL是否包含SELECT语句
	if viewSQL == "" {
		return fmt.Errorf("视图SQL不能为空")
	}

	// 移除首尾空白字符并转换为大写进行检查
	trimmedSQL := strings.TrimSpace(strings.ToUpper(viewSQL))

	// 检查是否是SELECT语句或CREATE VIEW语句
	if !strings.HasPrefix(trimmedSQL, "SELECT") && !strings.HasPrefix(trimmedSQL, "CREATE") {
		return fmt.Errorf("视图SQL必须是SELECT语句或CREATE VIEW语句")
	}

	// 检查是否包含危险操作
	dangerousKeywords := []string{"DROP", "DELETE", "UPDATE", "INSERT", "TRUNCATE", "ALTER"}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(trimmedSQL, keyword) {
			return fmt.Errorf("视图SQL不能包含危险操作: %s", keyword)
		}
	}

	return nil
}

// extractSelectFromViewSQL 从视图SQL中提取SELECT部分
func (s *SchemaService) extractSelectFromViewSQL(viewSQL string) string {
	trimmedSQL := strings.TrimSpace(viewSQL)
	upperSQL := strings.ToUpper(trimmedSQL)

	// 如果已经是CREATE VIEW语句，提取SELECT部分
	if strings.HasPrefix(upperSQL, "CREATE") {
		// 查找AS关键字后的SELECT部分
		asIndex := strings.Index(upperSQL, " AS ")
		if asIndex != -1 {
			return strings.TrimSpace(trimmedSQL[asIndex+4:])
		}
	}

	// 如果是SELECT语句，直接返回
	return trimmedSQL
}

// quoteIdentifier 给标识符添加引号
func (s *SchemaService) quoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}
