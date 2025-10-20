/*
 * @module service/database/schema_service_test
 * @description SchemaService 单元测试
 * @architecture 测试层 - 单元测试
 */

package database

import (
	"datahub-service/service/models"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB      *gorm.DB
	testService *SchemaService
	testSchema  = "test_schema_service"
)

// setupTestDB 初始化测试数据库连接
func setupTestDB(t *testing.T) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "postgres")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "things2024")
	sslMode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslMode)

	var err error
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "连接测试数据库失败")

	testService = NewSchemaService(testDB)

	// 创建测试 schema
	err = testDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", testSchema)).Error
	require.NoError(t, err, "创建测试schema失败")
}

// teardownTestDB 清理测试数据库
func teardownTestDB(t *testing.T) {
	if testDB != nil {
		// 删除测试 schema 及其所有对象
		err := testDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", testSchema)).Error
		assert.NoError(t, err, "删除测试schema失败")
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestMain 测试入口
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// TestNewSchemaService 测试创建服务实例
func TestNewSchemaService(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	assert.NotNil(t, testService)
	assert.NotNil(t, testService.db)
}

// TestCreateTable 测试创建表
func TestCreateTable(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_users"
	fields := []models.TableField{
		{
			NameZh:       "用户ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			Description:  "主键ID",
			OrderNum:     1,
		},
		{
			NameZh:      "用户名",
			NameEn:      "username",
			DataType:    "varchar",
			IsUnique:    true,
			IsNullable:  false,
			Description: "用户名",
			OrderNum:    2,
		},
		{
			NameZh:      "邮箱",
			NameEn:      "email",
			DataType:    "varchar",
			IsNullable:  true,
			Description: "电子邮箱",
			OrderNum:    3,
		},
		{
			NameZh:       "年龄",
			NameEn:       "age",
			DataType:     "integer",
			IsNullable:   true,
			DefaultValue: "0",
			Description:  "年龄",
			OrderNum:     4,
		},
		{
			NameZh:       "激活状态",
			NameEn:       "is_active",
			DataType:     "boolean",
			IsNullable:   false,
			DefaultValue: "true",
			Description:  "是否激活",
			OrderNum:     5,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	assert.NoError(t, err)

	// 验证表是否创建成功
	exists, err := testService.CheckTableExists(testSchema, tableName)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 验证列信息
	columns, err := testService.GetTableColumns(testSchema, tableName)
	assert.NoError(t, err)
	assert.Len(t, columns, 5)

	// 验证主键
	assert.True(t, columns[0].IsPrimaryKey)
	assert.Equal(t, "id", columns[0].Name)
}

// TestCreateTableWithCheckConstraint 测试创建带检查约束的表
func TestCreateTableWithCheckConstraint(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_products"
	fields := []models.TableField{
		{
			NameZh:       "产品ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:          "价格",
			NameEn:          "price",
			DataType:        "numeric",
			IsNullable:      false,
			CheckConstraint: "price > 0",
			OrderNum:        2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	assert.NoError(t, err)

	// 验证表创建成功
	exists, err := testService.CheckTableExists(testSchema, tableName)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestAlterTable 测试修改表结构
func TestAlterTable(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_alter_table"

	// 先创建表
	initialFields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, initialFields)
	require.NoError(t, err)

	// 修改表结构：添加新列、修改现有列
	modifiedFields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "text", // 修改数据类型
			IsNullable: true,   // 修改可空性
			OrderNum:   2,
		},
		{
			NameZh:     "描述",
			NameEn:     "description",
			DataType:   "text",
			IsNullable: true,
			OrderNum:   3,
		},
	}

	err = testService.alterTable(testSchema, tableName, modifiedFields)
	assert.NoError(t, err)

	// 验证修改后的列信息
	columns, err := testService.GetTableColumns(testSchema, tableName)
	assert.NoError(t, err)
	assert.Len(t, columns, 3)

	// 验证新增的列
	var descCol *ColumnDefinition
	for _, col := range columns {
		if col.Name == "description" {
			descCol = &col
			break
		}
	}
	assert.NotNil(t, descCol)
	assert.Equal(t, "text", descCol.DataType)
}

// TestAddColumn 测试添加列
func TestAddColumn(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_add_column"

	// 创建初始表
	initialFields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
	}

	err := testService.createTable(testSchema, tableName, initialFields)
	require.NoError(t, err)

	// 添加新列
	newField := models.TableField{
		NameZh:       "新字段",
		NameEn:       "new_field",
		DataType:     "varchar",
		IsNullable:   true,
		DefaultValue: "default_value",
		Description:  "新增字段",
		OrderNum:     2,
	}

	err = testService.addColumn(testSchema, tableName, newField)
	assert.NoError(t, err)

	// 验证列是否添加成功
	columns, err := testService.GetTableColumns(testSchema, tableName)
	assert.NoError(t, err)
	assert.Len(t, columns, 2)

	// 验证新列的属性
	var newCol *ColumnDefinition
	for _, col := range columns {
		if col.Name == "new_field" {
			newCol = &col
			break
		}
	}
	assert.NotNil(t, newCol)
	assert.True(t, newCol.IsNullable)
}

// TestDropColumn 测试删除列
func TestDropColumn(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_drop_column"

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "待删除字段",
			NameEn:     "to_delete",
			DataType:   "varchar",
			IsNullable: true,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 删除列
	err = testService.dropColumn(testSchema, tableName, "to_delete")
	assert.NoError(t, err)

	// 验证列是否已删除
	columns, err := testService.GetTableColumns(testSchema, tableName)
	assert.NoError(t, err)
	assert.Len(t, columns, 1)
	assert.Equal(t, "id", columns[0].Name)
}

// TestDropTable 测试删除表
func TestDropTable(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_drop_table"

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 验证表存在
	exists, err := testService.CheckTableExists(testSchema, tableName)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 删除表
	err = testService.dropTable(testSchema, tableName)
	assert.NoError(t, err)

	// 验证表已删除
	exists, err = testService.CheckTableExists(testSchema, tableName)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestUpdatePrimaryKey 测试更新主键约束
func TestUpdatePrimaryKey(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_update_pk"

	// 创建带单一主键的表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "代码",
			NameEn:     "code",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 修改为复合主键
	modifiedFields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:       "代码",
			NameEn:       "code",
			DataType:     "varchar",
			IsPrimaryKey: true, // 变为主键
			IsNullable:   false,
			OrderNum:     2,
		},
	}

	err = testService.alterTable(testSchema, tableName, modifiedFields)
	assert.NoError(t, err)

	// 验证复合主键
	primaryKeys, err := testService.getPrimaryKeys(testSchema, tableName)
	assert.NoError(t, err)
	assert.Len(t, primaryKeys, 2)
	assert.Contains(t, primaryKeys, "id")
	assert.Contains(t, primaryKeys, "code")
}

// TestCreateIndex 测试创建索引
func TestCreateIndex(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_index_table"
	indexName := "idx_test_username"

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "用户名",
			NameEn:     "username",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 创建索引
	err = testService.CreateIndex(testSchema, tableName, indexName, []string{"username"}, false, "btree")
	assert.NoError(t, err)

	// 验证索引创建成功
	indexes, err := testService.GetTableIndexes(testSchema, tableName)
	assert.NoError(t, err)
	assert.True(t, len(indexes) > 0)

	// 查找创建的索引
	var found bool
	for _, idx := range indexes {
		if idx.Name == indexName {
			found = true
			assert.Equal(t, "btree", idx.IndexType)
			assert.False(t, idx.IsUnique)
			break
		}
	}
	assert.True(t, found, "索引未找到")
}

// TestCreateUniqueIndex 测试创建唯一索引
func TestCreateUniqueIndex(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_unique_index"
	indexName := "idx_unique_email"

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "邮箱",
			NameEn:     "email",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 创建唯一索引
	err = testService.CreateIndex(testSchema, tableName, indexName, []string{"email"}, true, "btree")
	assert.NoError(t, err)

	// 验证唯一索引
	indexes, err := testService.GetTableIndexes(testSchema, tableName)
	assert.NoError(t, err)

	var found bool
	for _, idx := range indexes {
		if idx.Name == indexName {
			found = true
			assert.True(t, idx.IsUnique)
			break
		}
	}
	assert.True(t, found)
}

// TestDropIndex 测试删除索引
func TestDropIndex(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_drop_index"
	indexName := "idx_drop_test"

	// 创建表和索引
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "字段",
			NameEn:     "field",
			DataType:   "varchar",
			IsNullable: true,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	err = testService.CreateIndex(testSchema, tableName, indexName, []string{"field"}, false, "btree")
	require.NoError(t, err)

	// 删除索引
	err = testService.DropIndex(testSchema, indexName)
	assert.NoError(t, err)

	// 验证索引已删除
	indexes, err := testService.GetTableIndexes(testSchema, tableName)
	assert.NoError(t, err)

	for _, idx := range indexes {
		assert.NotEqual(t, indexName, idx.Name)
	}
}

// TestCreateView 测试创建视图
func TestCreateView(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_view_source"
	viewName := "test_view"

	// 创建源表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 创建视图
	viewSQL := fmt.Sprintf("SELECT id, name FROM %s.%s WHERE id > 0", testSchema, tableName)
	err = testService.createView(testSchema, viewName, viewSQL)
	assert.NoError(t, err)

	// 验证视图创建成功
	exists, err := testService.CheckViewExists(testSchema, viewName)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestUpdateView 测试更新视图
func TestUpdateView(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_modify_view_source"
	viewName := "test_modify_view"

	// 创建源表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 创建初始视图
	initialSQL := fmt.Sprintf("SELECT id FROM %s.%s", testSchema, tableName)
	err = testService.createView(testSchema, viewName, initialSQL)
	require.NoError(t, err)

	// 更新视图
	updatedSQL := fmt.Sprintf("SELECT id, name FROM %s.%s", testSchema, tableName)
	err = testService.updateView(testSchema, viewName, updatedSQL)
	assert.NoError(t, err)

	// 验证视图仍然存在
	exists, err := testService.CheckViewExists(testSchema, viewName)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestDropView 测试删除视图
func TestDropView(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_remove_view_source"
	viewName := "test_remove_view"

	// 创建源表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 创建视图
	viewSQL := fmt.Sprintf("SELECT * FROM %s.%s", testSchema, tableName)
	err = testService.createView(testSchema, viewName, viewSQL)
	require.NoError(t, err)

	// 删除视图
	err = testService.dropView(testSchema, viewName)
	assert.NoError(t, err)

	// 验证视图已删除
	exists, err := testService.CheckViewExists(testSchema, viewName)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestGetTableInfo 测试获取表信息
func TestGetTableInfo(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_table_info"

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 获取表信息
	tableInfo, err := testService.GetTableInfo(testSchema, tableName)
	assert.NoError(t, err)
	assert.NotNil(t, tableInfo)
	assert.Equal(t, tableName, tableInfo.Name)
	assert.Equal(t, testSchema, tableInfo.Schema)
	assert.Len(t, tableInfo.Columns, 2)
	assert.True(t, len(tableInfo.Constraints) > 0)
}

// TestListTables 测试列出所有表
func TestListTables(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	// 创建多个测试表
	table1 := "test_list_table_1"
	table2 := "test_list_table_2"

	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
	}

	err := testService.createTable(testSchema, table1, fields)
	require.NoError(t, err)

	err = testService.createTable(testSchema, table2, fields)
	require.NoError(t, err)

	// 列出所有表
	tables, err := testService.ListTables(testSchema)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tables), 2)
	assert.Contains(t, tables, table1)
	assert.Contains(t, tables, table2)
}

// TestGetTableData 测试获取表数据
func TestGetTableData(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_table_data"
	fullTableName := fmt.Sprintf("%s.%s", testSchema, tableName)

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 插入测试数据
	insertSQL := fmt.Sprintf("INSERT INTO %s.%s (id, name) VALUES (1, 'test1'), (2, 'test2'), (3, 'test3')",
		testSchema, tableName)
	err = testDB.Exec(insertSQL).Error
	require.NoError(t, err)

	// 获取表数据
	data, total, err := testService.GetTableData(fullTableName, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, data, 3)
	assert.Equal(t, int64(1), data[0]["id"])
	assert.Equal(t, "test1", data[0]["name"])
}

// TestGetTableDataWithPagination 测试分页获取表数据
func TestGetTableDataWithPagination(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tableName := "test_table_pagination"
	fullTableName := fmt.Sprintf("%s.%s", testSchema, tableName)

	// 创建表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 插入10条测试数据
	for i := 1; i <= 10; i++ {
		insertSQL := fmt.Sprintf("INSERT INTO %s.%s (id) VALUES (%d)", testSchema, tableName, i)
		err = testDB.Exec(insertSQL).Error
		require.NoError(t, err)
	}

	// 测试分页
	data, total, err := testService.GetTableData(fullTableName, 3, 0)
	assert.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, data, 3)

	// 测试第二页
	data, total, err = testService.GetTableData(fullTableName, 3, 3)
	assert.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, data, 3)
}

// TestValidateTableName 测试表名验证
func TestValidateTableName(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tests := []struct {
		name      string
		tableName string
		wantErr   bool
	}{
		{"有效的表名", "valid_table_name", false},
		{"有效的大写表名", "ValidTableName", false},
		{"空表名", "", true},
		{"过长的表名", "a_very_long_table_name_that_exceeds_the_maximum_allowed_length_limit", true},
		{"以数字开头", "123table", true},
		{"包含特殊字符", "table@name", true},
		{"包含空格", "table name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testService.ValidateTableName(tt.tableName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMapDataType 测试数据类型映射
func TestMapDataType(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tests := []struct {
		inputType  string
		expectType string
	}{
		{"integer", "integer"},
		{"int", "integer"},
		{"bigint", "bigint"},
		{"string", "varchar(255)"},
		{"varchar", "varchar(255)"},
		{"text", "text"},
		{"boolean", "boolean"},
		{"bool", "boolean"},
		{"datetime", "timestamp"},
		{"timestamp", "timestamp"},
		{"date", "date"},
		{"json", "json"},
		{"jsonb", "jsonb"},
		{"uuid", "uuid"},
		{"unknown_type", "varchar(255)"}, // 未知类型默认为varchar
	}

	for _, tt := range tests {
		t.Run(tt.inputType, func(t *testing.T) {
			result := testService.mapDataType(tt.inputType)
			assert.Equal(t, tt.expectType, result)
		})
	}
}

// TestManageTableSchema 测试表结构管理（集成测试）
func TestManageTableSchema(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	interfaceID := "test-interface-001"
	tableName := "test_manage_schema"

	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "名称",
			NameEn:     "name",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	// 测试创建表
	err := testService.ManageTableSchema(interfaceID, "create_table", testSchema, tableName, fields)
	assert.NoError(t, err)

	exists, err := testService.CheckTableExists(testSchema, tableName)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试修改表
	modifiedFields := append(fields, models.TableField{
		NameZh:     "描述",
		NameEn:     "description",
		DataType:   "text",
		IsNullable: true,
		OrderNum:   3,
	})

	err = testService.ManageTableSchema(interfaceID, "alter_table", testSchema, tableName, modifiedFields)
	assert.NoError(t, err)

	columns, err := testService.GetTableColumns(testSchema, tableName)
	assert.NoError(t, err)
	assert.Len(t, columns, 3)

	// 测试删除表
	err = testService.ManageTableSchema(interfaceID, "drop_table", testSchema, tableName, nil)
	assert.NoError(t, err)

	exists, err = testService.CheckTableExists(testSchema, tableName)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestManageViewSchema 测试视图管理（集成测试）
func TestManageViewSchema(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	interfaceID := "test-interface-002"
	tableName := "test_view_manage_source"
	viewName := "test_view_manage"

	// 先创建源表
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "integer",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
	}

	err := testService.createTable(testSchema, tableName, fields)
	require.NoError(t, err)

	// 测试创建视图
	viewSQL := fmt.Sprintf("SELECT * FROM %s.%s", testSchema, tableName)
	err = testService.ManageViewSchema(interfaceID, "create_view", testSchema, viewName, viewSQL)
	assert.NoError(t, err)

	exists, err := testService.CheckViewExists(testSchema, viewName)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试更新视图
	updatedSQL := fmt.Sprintf("SELECT id FROM %s.%s WHERE id > 0", testSchema, tableName)
	err = testService.ManageViewSchema(interfaceID, "update_view", testSchema, viewName, updatedSQL)
	assert.NoError(t, err)

	// 测试删除视图
	err = testService.ManageViewSchema(interfaceID, "drop_view", testSchema, viewName, "")
	assert.NoError(t, err)

	exists, err = testService.CheckViewExists(testSchema, viewName)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestValidateViewSQL 测试视图SQL验证
func TestValidateViewSQL(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{"有效的SELECT语句", "SELECT * FROM table", false},
		{"有效的CREATE VIEW语句", "CREATE VIEW v AS SELECT * FROM table", false},
		{"空SQL", "", true},
		{"包含DROP", "SELECT * FROM table; DROP TABLE users", true},
		{"包含DELETE", "SELECT * FROM table WHERE id IN (DELETE FROM users)", true},
		{"包含UPDATE", "SELECT * FROM table; UPDATE users SET name='test'", true},
		{"包含INSERT", "SELECT * FROM table; INSERT INTO users VALUES (1)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testService.validateViewSQL(tt.sql, testSchema, "test_view")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestListSchemas 测试列出所有schema
func TestListSchemas(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	schemas, err := testService.ListSchemas()
	assert.NoError(t, err)
	assert.True(t, len(schemas) > 0)
	assert.Contains(t, schemas, testSchema)
	// 系统schema不应该在列表中
	assert.NotContains(t, schemas, "pg_catalog")
	assert.NotContains(t, schemas, "information_schema")
}
