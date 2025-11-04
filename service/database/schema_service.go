/*
 * @module service/database/schema_service
 * @description 表结构管理服务，直接通过SQL操作数据库表结构、索引等
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 表结构操作请求 -> SQL执行 -> 结果验证 -> 状态更新
 * @rules 确保表结构操作的安全性和一致性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/interfaces.md
 */

package database

import (
	"datahub-service/service/models"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SchemaService 表结构管理服务
type SchemaService struct {
	db *gorm.DB
}

// NewSchemaService 创建表结构管理服务实例
func NewSchemaService(db *gorm.DB) *SchemaService {
	return &SchemaService{
		db: db,
	}
}

// TableDefinition 表定义结构
type TableDefinition struct {
	Name        string                 `json:"name"`
	Schema      string                 `json:"schema"`
	Comment     string                 `json:"comment,omitempty"`
	Columns     []ColumnDefinition     `json:"columns"`
	Constraints []ConstraintDefinition `json:"constraints,omitempty"`
	Indexes     []IndexDefinition      `json:"indexes,omitempty"`
}

// ColumnDefinition 列定义结构
type ColumnDefinition struct {
	Name             string      `json:"name"`
	DataType         string      `json:"data_type"`
	IsNullable       bool        `json:"is_nullable"`
	DefaultValue     interface{} `json:"default_value,omitempty"`
	Comment          string      `json:"comment,omitempty"`
	IsPrimaryKey     bool        `json:"is_primary_key,omitempty"`
	IsUnique         bool        `json:"is_unique,omitempty"`
	MaxLength        *int        `json:"max_length,omitempty"`
	NumericPrecision *int        `json:"numeric_precision,omitempty"`
	NumericScale     *int        `json:"numeric_scale,omitempty"`
	OrdinalPosition  int         `json:"ordinal_position"`
}

// ConstraintDefinition 约束定义结构
type ConstraintDefinition struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // primary_key, foreign_key, unique, check
	Columns []string `json:"columns"`
	Check   string   `json:"check,omitempty"`
}

// IndexDefinition 索引定义结构
type IndexDefinition struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	IsUnique   bool     `json:"is_unique"`
	IsPrimary  bool     `json:"is_primary"`
	IndexType  string   `json:"index_type"` // btree, hash, gist, gin, etc.
	Definition string   `json:"definition"` // 索引定义SQL
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
	// 构建CREATE TABLE语句
	var columnDefs []string
	var primaryKeys []string

	for _, field := range fields {
		columnDef := s.buildColumnDefinition(field)
		columnDefs = append(columnDefs, columnDef)

		if field.IsPrimaryKey {
			primaryKeys = append(primaryKeys, s.quoteIdentifier(field.NameEn))
		}
	}

	// 添加主键约束
	if len(primaryKeys) > 0 {
		columnDefs = append(columnDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	createSQL := fmt.Sprintf(
		"CREATE TABLE %s.%s (\n  %s\n)",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
		strings.Join(columnDefs, ",\n  "),
	)

	slog.Debug("SchemaService.createTable", "sql", createSQL)

	if err := s.db.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	// 添加表注释
	commentSQL := fmt.Sprintf(
		"COMMENT ON TABLE %s.%s IS '数据接口表: %s'",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
		tableName,
	)
	if err := s.db.Exec(commentSQL).Error; err != nil {
		slog.Warn("添加表注释失败", "error", err)
	}

	// 添加列注释
	for _, field := range fields {
		if field.NameZh != "" || field.Description != "" {
			comment := field.NameZh
			if field.Description != "" {
				comment = fmt.Sprintf("%s - %s", field.NameZh, field.Description)
			}
			columnCommentSQL := fmt.Sprintf(
				"COMMENT ON COLUMN %s.%s.%s IS %s",
				s.quoteIdentifier(schemaName),
				s.quoteIdentifier(tableName),
				s.quoteIdentifier(field.NameEn),
				s.quoteLiteral(comment),
			)
			if err := s.db.Exec(columnCommentSQL).Error; err != nil {
				slog.Warn("添加列注释失败", "column", field.NameEn, "error", err)
			}
		}
	}

	return nil
}

// buildColumnDefinition 构建列定义SQL
func (s *SchemaService) buildColumnDefinition(field models.TableField) string {
	parts := []string{s.quoteIdentifier(field.NameEn)}

	// 数据类型
	dataType := s.mapDataType(field.DataType)
	parts = append(parts, dataType)

	// NOT NULL约束
	if !field.IsNullable {
		parts = append(parts, "NOT NULL")
	}

	// 默认值
	if field.DefaultValue != "" {
		parts = append(parts, "DEFAULT", s.formatDefaultValue(field.DefaultValue, dataType))
	}

	// UNIQUE约束
	if field.IsUnique && !field.IsPrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	// CHECK约束
	if field.CheckConstraint != "" {
		parts = append(parts, "CHECK", "("+field.CheckConstraint+")")
	}

	return strings.Join(parts, " ")
}

// alterTable 修改表结构
func (s *SchemaService) alterTable(schemaName, tableName string, fields []models.TableField) error {
	// 获取当前表结构
	currentColumns, err := s.GetTableColumns(schemaName, tableName)
	if err != nil {
		return fmt.Errorf("获取表结构失败: %v", err)
	}

	// 构建当前列映射
	currentColMap := make(map[string]ColumnDefinition)
	for _, col := range currentColumns {
		currentColMap[col.Name] = col
	}

	// 构建新列映射
	newColMap := make(map[string]models.TableField)
	newPrimaryKeys := []string{}
	for _, field := range fields {
		newColMap[field.NameEn] = field
		if field.IsPrimaryKey {
			newPrimaryKeys = append(newPrimaryKeys, field.NameEn)
		}
	}

	// 获取当前主键
	currentPrimaryKeys, err := s.GetPrimaryKeys(schemaName, tableName)
	if err != nil {
		return fmt.Errorf("获取主键失败: %v", err)
	}

	// 执行列的添加、删除、修改
	for _, field := range fields {
		if _, exists := currentColMap[field.NameEn]; !exists {
			// 添加新列
			if err := s.addColumn(schemaName, tableName, field); err != nil {
				return fmt.Errorf("添加列 %s 失败: %v", field.NameEn, err)
			}
		} else {
			// 修改现有列
			if err := s.modifyColumn(schemaName, tableName, field, currentColMap[field.NameEn]); err != nil {
				return fmt.Errorf("修改列 %s 失败: %v", field.NameEn, err)
			}
		}
	}

	// 删除不再需要的列
	for colName := range currentColMap {
		if _, exists := newColMap[colName]; !exists {
			if err := s.dropColumn(schemaName, tableName, colName); err != nil {
				return fmt.Errorf("删除列 %s 失败: %v", colName, err)
			}
		}
	}

	// 更新主键约束（如果有变化）
	if !stringSliceEqual(currentPrimaryKeys, newPrimaryKeys) {
		if err := s.updatePrimaryKey(schemaName, tableName, currentPrimaryKeys, newPrimaryKeys); err != nil {
			return fmt.Errorf("更新主键约束失败: %v", err)
		}
	}

	return nil
}

// addColumn 添加列
func (s *SchemaService) addColumn(schemaName, tableName string, field models.TableField) error {
	alterSQL := fmt.Sprintf(
		"ALTER TABLE %s.%s ADD COLUMN %s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
		s.buildColumnDefinition(field),
	)

	slog.Debug("SchemaService.addColumn", "sql", alterSQL)

	if err := s.db.Exec(alterSQL).Error; err != nil {
		return err
	}

	// 添加列注释
	if field.NameZh != "" || field.Description != "" {
		comment := field.NameZh
		if field.Description != "" {
			comment = fmt.Sprintf("%s - %s", field.NameZh, field.Description)
		}
		commentSQL := fmt.Sprintf(
			"COMMENT ON COLUMN %s.%s.%s IS %s",
			s.quoteIdentifier(schemaName),
			s.quoteIdentifier(tableName),
			s.quoteIdentifier(field.NameEn),
			s.quoteLiteral(comment),
		)
		if err := s.db.Exec(commentSQL).Error; err != nil {
			slog.Warn("添加列注释失败", "error", err)
		}
	}

	return nil
}

// modifyColumn 修改列
func (s *SchemaService) modifyColumn(schemaName, tableName string, newField models.TableField, oldCol ColumnDefinition) error {
	newDataType := s.mapDataType(newField.DataType)

	// 检查数据类型是否需要修改
	if oldCol.DataType != newDataType {
		alterSQL := fmt.Sprintf(
			"ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s",
			s.quoteIdentifier(schemaName),
			s.quoteIdentifier(tableName),
			s.quoteIdentifier(newField.NameEn),
			newDataType,
		)
		slog.Debug("SchemaService.modifyColumn - 修改类型", "sql", alterSQL)
		if err := s.db.Exec(alterSQL).Error; err != nil {
			return fmt.Errorf("修改列类型失败: %v", err)
		}
	}

	// 检查可空性是否需要修改
	if oldCol.IsNullable != newField.IsNullable {
		var nullConstraint string
		if newField.IsNullable {
			nullConstraint = "DROP NOT NULL"
		} else {
			nullConstraint = "SET NOT NULL"
		}
		alterSQL := fmt.Sprintf(
			"ALTER TABLE %s.%s ALTER COLUMN %s %s",
			s.quoteIdentifier(schemaName),
			s.quoteIdentifier(tableName),
			s.quoteIdentifier(newField.NameEn),
			nullConstraint,
		)
		slog.Debug("SchemaService.modifyColumn - 修改可空性", "sql", alterSQL)
		if err := s.db.Exec(alterSQL).Error; err != nil {
			return fmt.Errorf("修改列可空性失败: %v", err)
		}
	}

	// 更新默认值
	if newField.DefaultValue != "" {
		alterSQL := fmt.Sprintf(
			"ALTER TABLE %s.%s ALTER COLUMN %s SET DEFAULT %s",
			s.quoteIdentifier(schemaName),
			s.quoteIdentifier(tableName),
			s.quoteIdentifier(newField.NameEn),
			s.formatDefaultValue(newField.DefaultValue, newDataType),
		)
		slog.Debug("SchemaService.modifyColumn - 设置默认值", "sql", alterSQL)
		if err := s.db.Exec(alterSQL).Error; err != nil {
			return fmt.Errorf("设置默认值失败: %v", err)
		}
	}

	// 更新列注释
	if newField.NameZh != "" || newField.Description != "" {
		comment := newField.NameZh
		if newField.Description != "" {
			comment = fmt.Sprintf("%s - %s", newField.NameZh, newField.Description)
		}
		commentSQL := fmt.Sprintf(
			"COMMENT ON COLUMN %s.%s.%s IS %s",
			s.quoteIdentifier(schemaName),
			s.quoteIdentifier(tableName),
			s.quoteIdentifier(newField.NameEn),
			s.quoteLiteral(comment),
		)
		if err := s.db.Exec(commentSQL).Error; err != nil {
			slog.Warn("更新列注释失败", "error", err)
		}
	}

	return nil
}

// dropColumn 删除列
func (s *SchemaService) dropColumn(schemaName, tableName, columnName string) error {
	alterSQL := fmt.Sprintf(
		"ALTER TABLE %s.%s DROP COLUMN %s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
		s.quoteIdentifier(columnName),
	)

	slog.Debug("SchemaService.dropColumn", "sql", alterSQL)
	return s.db.Exec(alterSQL).Error
}

// updatePrimaryKey 更新主键约束
func (s *SchemaService) updatePrimaryKey(schemaName, tableName string, oldKeys, newKeys []string) error {
	// 删除旧主键约束
	if len(oldKeys) > 0 {
		constraintName, err := s.getPrimaryKeyConstraintName(schemaName, tableName)
		if err != nil {
			return fmt.Errorf("获取主键约束名失败: %v", err)
		}

		if constraintName != "" {
			dropSQL := fmt.Sprintf(
				"ALTER TABLE %s.%s DROP CONSTRAINT %s",
				s.quoteIdentifier(schemaName),
				s.quoteIdentifier(tableName),
				s.quoteIdentifier(constraintName),
			)
			slog.Debug("SchemaService.updatePrimaryKey - 删除旧主键", "sql", dropSQL)
			if err := s.db.Exec(dropSQL).Error; err != nil {
				return fmt.Errorf("删除主键约束失败: %v", err)
			}
		}
	}

	// 添加新主键约束
	if len(newKeys) > 0 {
		// 确保主键列不为空
		for _, key := range newKeys {
			setNotNullSQL := fmt.Sprintf(
				"ALTER TABLE %s.%s ALTER COLUMN %s SET NOT NULL",
				s.quoteIdentifier(schemaName),
				s.quoteIdentifier(tableName),
				s.quoteIdentifier(key),
			)
			if err := s.db.Exec(setNotNullSQL).Error; err != nil {
				slog.Warn("设置列为非空失败（可能已经是非空）", "error", err)
			}
		}

		quotedKeys := make([]string, len(newKeys))
		for i, key := range newKeys {
			quotedKeys[i] = s.quoteIdentifier(key)
		}

		addSQL := fmt.Sprintf(
			"ALTER TABLE %s.%s ADD PRIMARY KEY (%s)",
			s.quoteIdentifier(schemaName),
			s.quoteIdentifier(tableName),
			strings.Join(quotedKeys, ", "),
		)
		slog.Debug("SchemaService.updatePrimaryKey - 添加新主键", "sql", addSQL)
		if err := s.db.Exec(addSQL).Error; err != nil {
			return fmt.Errorf("添加主键约束失败: %v", err)
		}
	}

	return nil
}

// dropTable 删除表
func (s *SchemaService) dropTable(schemaName, tableName string) error {
	dropSQL := fmt.Sprintf(
		"DROP TABLE IF EXISTS %s.%s CASCADE",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
	)

	slog.Debug("SchemaService.dropTable", "sql", dropSQL)
	return s.db.Exec(dropSQL).Error
}

// GetTableColumns 获取表的列信息
func (s *SchemaService) GetTableColumns(schemaName, tableName string) ([]ColumnDefinition, error) {
	query := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			c.ordinal_position,
			pgd.description as comment,
			EXISTS(
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu 
					ON tc.constraint_name = kcu.constraint_name
					AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'PRIMARY KEY'
					AND tc.table_schema = c.table_schema
					AND tc.table_name = c.table_name
					AND kcu.column_name = c.column_name
			) as is_primary_key,
			EXISTS(
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu 
					ON tc.constraint_name = kcu.constraint_name
					AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'UNIQUE'
					AND tc.table_schema = c.table_schema
					AND tc.table_name = c.table_name
					AND kcu.column_name = c.column_name
			) as is_unique
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st 
			ON c.table_schema = st.schemaname 
			AND c.table_name = st.relname
		LEFT JOIN pg_catalog.pg_description pgd 
			ON pgd.objoid = st.relid 
			AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	var columns []ColumnDefinition
	rows, err := s.db.Raw(query, schemaName, tableName).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询列信息失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnDefinition
		var defaultValue, comment *string
		var maxLength, numericPrecision, numericScale *int

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&defaultValue,
			&maxLength,
			&numericPrecision,
			&numericScale,
			&col.OrdinalPosition,
			&comment,
			&col.IsPrimaryKey,
			&col.IsUnique,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描列信息失败: %v", err)
		}

		if defaultValue != nil {
			col.DefaultValue = *defaultValue
		}
		if comment != nil {
			col.Comment = *comment
		}
		col.MaxLength = maxLength
		col.NumericPrecision = numericPrecision
		col.NumericScale = numericScale

		columns = append(columns, col)
	}

	return columns, nil
}

// GetTableInfo 获取表信息（包含列、约束、索引）
func (s *SchemaService) GetTableInfo(schemaName, tableName string) (*TableDefinition, error) {
	tableDef := &TableDefinition{
		Schema: schemaName,
		Name:   tableName,
	}

	// 获取表注释
	var comment *string
	commentQuery := `
		SELECT pgd.description
		FROM pg_catalog.pg_statio_all_tables st
		LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = st.relid AND pgd.objsubid = 0
		WHERE st.schemaname = $1 AND st.relname = $2
	`
	s.db.Raw(commentQuery, schemaName, tableName).Scan(&comment)
	if comment != nil {
		tableDef.Comment = *comment
	}

	// 获取列信息
	columns, err := s.GetTableColumns(schemaName, tableName)
	if err != nil {
		return nil, err
	}
	tableDef.Columns = columns

	// 获取约束信息
	constraints, err := s.getTableConstraints(schemaName, tableName)
	if err != nil {
		return nil, err
	}
	tableDef.Constraints = constraints

	// 获取索引信息
	indexes, err := s.GetTableIndexes(schemaName, tableName)
	if err != nil {
		return nil, err
	}
	tableDef.Indexes = indexes

	return tableDef, nil
}

// getTableConstraints 获取表的约束信息
func (s *SchemaService) getTableConstraints(schemaName, tableName string) ([]ConstraintDefinition, error) {
	query := `
		SELECT 
			tc.constraint_name,
			tc.constraint_type,
			string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) as columns,
			cc.check_clause
		FROM information_schema.table_constraints tc
		LEFT JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		LEFT JOIN information_schema.check_constraints cc
			ON tc.constraint_name = cc.constraint_name
			AND tc.table_schema = cc.constraint_schema
		WHERE tc.table_schema = $1 AND tc.table_name = $2
		GROUP BY tc.constraint_name, tc.constraint_type, cc.check_clause
	`

	var constraints []ConstraintDefinition
	rows, err := s.db.Raw(query, schemaName, tableName).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询约束信息失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var constraint ConstraintDefinition
		var constraintType string
		var columnsStr *string
		var checkClause *string

		err := rows.Scan(&constraint.Name, &constraintType, &columnsStr, &checkClause)
		if err != nil {
			return nil, fmt.Errorf("扫描约束信息失败: %v", err)
		}

		// 转换约束类型
		switch constraintType {
		case "PRIMARY KEY":
			constraint.Type = "primary_key"
		case "FOREIGN KEY":
			constraint.Type = "foreign_key"
		case "UNIQUE":
			constraint.Type = "unique"
		case "CHECK":
			constraint.Type = "check"
		}

		// 处理列名(可能为NULL)
		if columnsStr != nil && *columnsStr != "" {
			constraint.Columns = strings.Split(*columnsStr, ",")
		} else {
			constraint.Columns = []string{}
		}

		if checkClause != nil {
			constraint.Check = *checkClause
		}

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

// GetTableIndexes 获取表的索引信息
func (s *SchemaService) GetTableIndexes(schemaName, tableName string) ([]IndexDefinition, error) {
	query := `
		SELECT 
			i.indexname as name,
			string_agg(a.attname, ',' ORDER BY array_position(ix.indkey, a.attnum)) as columns,
			ix.indisunique as is_unique,
			ix.indisprimary as is_primary,
			am.amname as index_type,
			pg_get_indexdef(c.oid) as definition
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.indexname
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = i.schemaname
		JOIN pg_index ix ON ix.indexrelid = c.oid
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_am am ON am.oid = c.relam
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE i.schemaname = $1 AND i.tablename = $2
		GROUP BY i.indexname, ix.indisunique, ix.indisprimary, am.amname, c.oid
		ORDER BY i.indexname
	`

	var indexes []IndexDefinition
	rows, err := s.db.Raw(query, schemaName, tableName).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询索引信息失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var index IndexDefinition
		var columnsStr string

		err := rows.Scan(
			&index.Name,
			&columnsStr,
			&index.IsUnique,
			&index.IsPrimary,
			&index.IndexType,
			&index.Definition,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描索引信息失败: %v", err)
		}

		index.Columns = strings.Split(columnsStr, ",")
		indexes = append(indexes, index)
	}

	return indexes, nil
}

// CreateIndex 创建索引
func (s *SchemaService) CreateIndex(schemaName, tableName, indexName string, columns []string, isUnique bool, indexType string) error {
	if indexType == "" {
		indexType = "btree"
	}

	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = s.quoteIdentifier(col)
	}

	uniqueStr := ""
	if isUnique {
		uniqueStr = "UNIQUE "
	}

	createSQL := fmt.Sprintf(
		"CREATE %sINDEX %s ON %s.%s USING %s (%s)",
		uniqueStr,
		s.quoteIdentifier(indexName),
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
		indexType,
		strings.Join(quotedColumns, ", "),
	)

	slog.Debug("SchemaService.CreateIndex", "sql", createSQL)
	if err := s.db.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建索引失败: %v", err)
	}

	return nil
}

// DropIndex 删除索引
func (s *SchemaService) DropIndex(schemaName, indexName string) error {
	dropSQL := fmt.Sprintf(
		"DROP INDEX IF EXISTS %s.%s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(indexName),
	)

	slog.Debug("SchemaService.DropIndex", "sql", dropSQL)
	return s.db.Exec(dropSQL).Error
}

// GetPrimaryKeys 获取表的主键列
func (s *SchemaService) GetPrimaryKeys(schemaName, tableName string) ([]string, error) {
	query := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
		ORDER BY kcu.ordinal_position
	`

	var primaryKeys []string
	rows, err := s.db.Raw(query, schemaName, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	return primaryKeys, nil
}

// getPrimaryKeyConstraintName 获取主键约束名
func (s *SchemaService) getPrimaryKeyConstraintName(schemaName, tableName string) (string, error) {
	query := `
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE constraint_type = 'PRIMARY KEY'
			AND table_schema = $1
			AND table_name = $2
	`

	var constraintName string
	err := s.db.Raw(query, schemaName, tableName).Scan(&constraintName).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return constraintName, err
}

// createView 创建视图
func (s *SchemaService) createView(schemaName, viewName, viewSQL string) error {
	if err := s.validateViewSQL(viewSQL, schemaName, viewName); err != nil {
		return fmt.Errorf("视图SQL验证失败: %v", err)
	}

	// 检查视图是否已存在
	viewExists, err := s.CheckViewExists(schemaName, viewName)
	if err != nil {
		return fmt.Errorf("检查视图存在性失败: %v", err)
	}

	// 尝试使用 CREATE OR REPLACE VIEW
	fullSQL := fmt.Sprintf("CREATE OR REPLACE VIEW %s.%s AS %s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(viewName),
		s.extractSelectFromViewSQL(viewSQL))

	slog.Debug("SchemaService.createView", "sql", fullSQL, "view_exists", viewExists)

	err = s.db.Exec(fullSQL).Error
	if err != nil {
		// 如果 CREATE OR REPLACE 失败（可能是不兼容的列变更），尝试先删除再创建
		if strings.Contains(err.Error(), "cannot change") ||
			strings.Contains(err.Error(), "cannot drop") ||
			strings.Contains(err.Error(), "cannot be replaced") {

			slog.Warn("视图定义不兼容，将先删除后重建",
				"schema", schemaName,
				"view", viewName,
				"error", err.Error())

			// 先删除视图（会丢失权限）
			if dropErr := s.forceDropView(schemaName, viewName); dropErr != nil {
				return fmt.Errorf("删除不兼容视图失败: %v (原始错误: %v)", dropErr, err)
			}

			// 重新创建视图
			if createErr := s.db.Exec(fullSQL).Error; createErr != nil {
				return fmt.Errorf("重建视图失败: %v", createErr)
			}

			slog.Info("视图已成功重建", "schema", schemaName, "view", viewName)
		} else {
			return fmt.Errorf("创建视图失败: %v", err)
		}
	}

	// 授予视图访问权限
	// 注意：
	// - 如果是 CREATE OR REPLACE 成功且列兼容，原有权限会保留，这里重新授权是幂等的
	// - 如果是先 DROP 后 CREATE，权限已丢失，这里必须重新授权
	if err := s.grantViewPermissions(schemaName, viewName); err != nil {
		slog.Warn("授予视图权限失败", "schema", schemaName, "view", viewName, "error", err)
		// 权限授予失败不应该阻止视图创建，只记录警告
	}

	return nil
}

// updateView 更新视图
func (s *SchemaService) updateView(schemaName, viewName, viewSQL string) error {
	return s.createView(schemaName, viewName, viewSQL)
}

// dropView 删除视图
// 注意：PostgreSQL 在执行 DROP VIEW 时会自动删除所有相关的权限授予(GRANT)
func (s *SchemaService) dropView(schemaName, viewName string) error {
	dropSQL := fmt.Sprintf("DROP VIEW IF EXISTS %s.%s CASCADE",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(viewName))

	slog.Debug("SchemaService.dropView", "sql", dropSQL)
	if err := s.db.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("删除视图失败: %v", err)
	}

	slog.Info("视图已删除（包括所有依赖和权限）", "schema", schemaName, "view", viewName)
	return nil
}

// forceDropView 强制删除视图（内部使用，用于处理 REPLACE 失败的情况）
func (s *SchemaService) forceDropView(schemaName, viewName string) error {
	dropSQL := fmt.Sprintf("DROP VIEW IF EXISTS %s.%s CASCADE",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(viewName))

	slog.Debug("SchemaService.forceDropView", "sql", dropSQL)
	if err := s.db.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("强制删除视图失败: %v", err)
	}

	return nil
}

// grantViewPermissions 授予视图访问权限
func (s *SchemaService) grantViewPermissions(schemaName, viewName string) error {
	// 构建完整的视图名称
	fullViewName := fmt.Sprintf("%s.%s", s.quoteIdentifier(schemaName), s.quoteIdentifier(viewName))

	// 需要授予权限的角色列表
	// 1. authenticator - PostgREST 使用的角色
	// 2. admin, user, readonly, guest - 应用级别的角色
	roles := []string{"authenticator", "admin", `"user"`, "readonly", "guest"}

	slog.Info("开始授予视图访问权限", "schema", schemaName, "view", viewName, "roles", roles)

	for _, role := range roles {
		// 授予 SELECT 权限（视图通常只需要 SELECT）
		grantSQL := fmt.Sprintf("GRANT SELECT ON %s TO %s", fullViewName, role)

		slog.Debug("授予视图权限", "sql", grantSQL)

		if err := s.db.Exec(grantSQL).Error; err != nil {
			// 如果角色不存在，记录警告但继续
			if strings.Contains(err.Error(), "role") && strings.Contains(err.Error(), "does not exist") {
				slog.Warn("角色不存在，跳过权限授予", "role", role, "error", err)
				continue
			}
			return fmt.Errorf("授予视图权限失败 (role=%s): %v", role, err)
		}
	}

	// 同时需要授予该 schema 下所有现有用户的权限
	if err := s.grantViewPermissionsToSchemaUsers(schemaName, viewName); err != nil {
		slog.Warn("授予 schema 用户视图权限失败", "error", err)
		// 不返回错误，因为这只是额外的权限授予
	}

	slog.Info("视图访问权限授予完成", "schema", schemaName, "view", viewName)
	return nil
}

// grantViewPermissionsToSchemaUsers 授予 schema 下所有现有用户视图权限
func (s *SchemaService) grantViewPermissionsToSchemaUsers(schemaName, viewName string) error {
	// 查询有该 schema 访问权限的所有用户
	query := `
		SELECT DISTINCT grantee
		FROM information_schema.role_table_grants
		WHERE table_schema = $1
			AND privilege_type = 'SELECT'
			AND grantee NOT IN ('authenticator', 'admin', 'user', 'readonly', 'guest', 'postgres', 'supabase_admin')
	`

	rows, err := s.db.Raw(query, schemaName).Rows()
	if err != nil {
		return fmt.Errorf("查询 schema 用户失败: %v", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			slog.Warn("扫描用户名失败", "error", err)
			continue
		}
		users = append(users, username)
	}

	if len(users) == 0 {
		slog.Debug("没有需要授予视图权限的额外用户", "schema", schemaName)
		return nil
	}

	fullViewName := fmt.Sprintf("%s.%s", s.quoteIdentifier(schemaName), s.quoteIdentifier(viewName))

	for _, username := range users {
		grantSQL := fmt.Sprintf("GRANT SELECT ON %s TO %s", fullViewName, s.quoteIdentifier(username))

		slog.Debug("授予用户视图权限", "user", username, "sql", grantSQL)

		if err := s.db.Exec(grantSQL).Error; err != nil {
			slog.Warn("授予用户视图权限失败", "user", username, "error", err)
			continue
		}
	}

	slog.Info("已授予额外用户视图权限", "schema", schemaName, "view", viewName, "user_count", len(users))
	return nil
}

// CheckTableExists 检查表是否存在
func (s *SchemaService) CheckTableExists(schemaName, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = $1 AND table_name = $2
		)`

	var exists bool
	err := s.db.Raw(query, schemaName, tableName).Scan(&exists).Error
	return exists, err
}

// CheckViewExists 检查视图是否存在
func (s *SchemaService) CheckViewExists(schemaName, viewName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.views 
			WHERE table_schema = $1 AND table_name = $2
		)`

	var exists bool
	err := s.db.Raw(query, schemaName, viewName).Scan(&exists).Error
	return exists, err
}

// ListTables 列出schema中的所有表
func (s *SchemaService) ListTables(schemaName string) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	var tableNames []string
	rows, err := s.db.Raw(query, schemaName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, tableName)
	}

	return tableNames, nil
}

// ListSchemas 列出所有Schema
func (s *SchemaService) ListSchemas() ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			AND schema_name NOT LIKE 'pg_temp%'
			AND schema_name NOT LIKE 'pg_toast_temp%'
		ORDER BY schema_name
	`

	var schemaNames []string
	rows, err := s.db.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		schemaNames = append(schemaNames, schemaName)
	}

	return schemaNames, nil
}

// GetTableData 获取表数据
func (s *SchemaService) GetTableData(fullTableName string, limit, offset int) ([]map[string]interface{}, int, error) {
	parts := strings.Split(fullTableName, ".")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("无效的表名格式，应为 schema.table")
	}

	schemaName := parts[0]
	tableName := parts[1]

	exists, err := s.CheckTableExists(schemaName, tableName)
	if err != nil {
		return nil, 0, fmt.Errorf("检查表存在性失败: %v", err)
	}
	if !exists {
		return nil, 0, fmt.Errorf("表 %s 不存在", fullTableName)
	}

	// 获取总行数
	var totalCount int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName))
	err = s.db.Raw(countSQL).Scan(&totalCount).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取总行数失败: %v", err)
	}

	// 获取数据
	dataSQL := fmt.Sprintf("SELECT * FROM %s.%s ORDER BY 1 LIMIT %d OFFSET %d",
		s.quoteIdentifier(schemaName),
		s.quoteIdentifier(tableName),
		limit, offset)

	rows, err := s.db.Raw(dataSQL).Rows()
	if err != nil {
		return nil, 0, fmt.Errorf("查询数据失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, fmt.Errorf("获取列名失败: %v", err)
	}

	var data []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, fmt.Errorf("扫描行数据失败: %v", err)
		}

		rowData := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowData[col] = string(b)
			} else if b, ok := val.(time.Time); ok {
				rowData[col] = b.Format("2006-01-02 15:04:05")
			} else if b, ok := val.(*time.Time); ok {
				rowData[col] = b.Format("2006-01-02 15:04:05")
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

// ValidateTableName 验证表名
func (s *SchemaService) ValidateTableName(tableName string) error {
	if len(tableName) == 0 {
		return fmt.Errorf("表名不能为空")
	}
	if len(tableName) > 63 {
		return fmt.Errorf("表名长度不能超过63个字符")
	}
	if !((tableName[0] >= 'a' && tableName[0] <= 'z') || (tableName[0] >= 'A' && tableName[0] <= 'Z')) {
		return fmt.Errorf("表名必须以字母开头")
	}
	for i := 1; i < len(tableName); i++ {
		c := tableName[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("表名只能包含字母、数字和下划线")
		}
	}
	return nil
}

// validateViewSQL 验证视图SQL语句
func (s *SchemaService) validateViewSQL(viewSQL, schemaName, viewName string) error {
	if viewSQL == "" {
		return fmt.Errorf("视图SQL不能为空")
	}

	trimmedSQL := strings.TrimSpace(strings.ToUpper(viewSQL))

	if !strings.HasPrefix(trimmedSQL, "SELECT") && !strings.HasPrefix(trimmedSQL, "CREATE") {
		return fmt.Errorf("视图SQL必须是SELECT语句或CREATE VIEW语句")
	}

	dangerousKeywords := []string{"DROP ", "DELETE ", "UPDATE ", "INSERT ", "TRUNCATE ", "ALTER ", "GRANT ", "REVOKE "}
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

	if strings.HasPrefix(upperSQL, "CREATE") {
		asIndex := strings.Index(upperSQL, " AS ")
		if asIndex != -1 {
			return strings.TrimSpace(trimmedSQL[asIndex+4:])
		}
	}

	return trimmedSQL
}

// mapDataType 映射数据类型到PostgreSQL类型
func (s *SchemaService) mapDataType(dataType string) string {
	typeMap := map[string]string{
		"integer":   "integer",
		"int":       "integer",
		"bigint":    "bigint",
		"smallint":  "smallint",
		"string":    "varchar(255)",
		"varchar":   "varchar(255)",
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
		"double":    "double precision",
		"json":      "json",
		"jsonb":     "jsonb",
		"uuid":      "uuid",
		"inet":      "inet",
		"cidr":      "cidr",
		"macaddr":   "macaddr",
		"bytea":     "bytea",
	}

	if pgType, exists := typeMap[dataType]; exists {
		return pgType
	}

	return "varchar(255)"
}

// formatDefaultValue 格式化默认值
func (s *SchemaService) formatDefaultValue(defaultValue, dataType string) string {
	if defaultValue == "" {
		return "NULL"
	}

	// 特殊函数值
	if strings.Contains(strings.ToUpper(defaultValue), "NOW()") ||
		strings.Contains(strings.ToUpper(defaultValue), "CURRENT_TIMESTAMP") ||
		strings.Contains(strings.ToUpper(defaultValue), "CURRENT_DATE") ||
		strings.Contains(strings.ToUpper(defaultValue), "UUID_GENERATE") ||
		strings.Contains(strings.ToUpper(defaultValue), "GEN_RANDOM_UUID") {
		return defaultValue
	}

	// 根据数据类型格式化
	switch {
	case strings.Contains(dataType, "char") || strings.Contains(dataType, "text"):
		return s.quoteLiteral(defaultValue)
	case strings.Contains(dataType, "boolean"):
		switch strings.ToLower(defaultValue) {
		case "true", "1", "yes":
			return "true"
		case "false", "0", "no":
			return "false"
		}
	case strings.Contains(dataType, "timestamp") || strings.Contains(dataType, "date"):
		if _, err := time.Parse("2006-01-02", defaultValue); err == nil {
			return s.quoteLiteral(defaultValue)
		}
		if _, err := time.Parse("2006-01-02 15:04:05", defaultValue); err == nil {
			return s.quoteLiteral(defaultValue)
		}
	}

	return defaultValue
}

// quoteIdentifier 给标识符添加引号
func (s *SchemaService) quoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}

// quoteLiteral 给字符串字面量添加引号并转义
func (s *SchemaService) quoteLiteral(literal string) string {
	escaped := strings.ReplaceAll(literal, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

// stringSliceEqual 比较两个字符串切片是否相等
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
