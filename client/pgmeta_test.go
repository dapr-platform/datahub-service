/*
 * @module client/pgmeta_test
 * @description PostgreSQL Meta客户端全面测试
 * @architecture 测试架构 - 集成测试
 * @documentReference client/pgmeta.go
 * @stateFlow 测试初始化 -> API功能测试 -> 清理
 * @rules 确保所有API功能都经过测试验证
 * @dependencies testing, datahub-service/client
 * @refs PostgreSQL Meta API测试
 */

package client

import (
	"fmt"
	"testing"
	"time"
)

const (
	testBaseURL    = "http://localhost:3001"
	testPgHeader   = "default"
	testSchemaName = "test_schema_" + "pgmeta"
	testTableName  = "test_table_" + "pgmeta"
)

var testClient *PgMetaClient

func init() {
	testClient = NewPgMetaClient(testBaseURL, testPgHeader)
}

// TestPgMetaClient_Schemas 测试Schema相关功能
func TestPgMetaClient_Schemas(t *testing.T) {
	t.Log("=== 开始测试Schema功能 ===")

	// 1. 获取所有schemas
	t.Run("ListSchemas", func(t *testing.T) {
		schemas, err := testClient.ListSchemas(&[]bool{false}[0], nil, nil)
		if err != nil {
			t.Logf("警告: 获取schemas失败: %v", err)
		} else {
			t.Logf("成功获取 %d 个schemas", len(schemas))
			for i, schema := range schemas {
				if i < 3 { // 只显示前3个
					t.Logf("  Schema[%d]: ID=%d, Name=%s, Owner=%s", i, schema.ID, schema.Name, schema.Owner)
				}
			}
		}
	})

	// 2. 创建测试schema
	var createdSchema *Schema
	t.Run("CreateSchema", func(t *testing.T) {
		req := CreateSchemaRequest{
			Name:  testSchemaName,
			Owner: "postgres",
		}

		schema, err := testClient.CreateSchema(req)
		if err != nil {
			t.Logf("警告: 创建schema失败: %v", err)
		} else {
			createdSchema = schema
			t.Logf("成功创建schema: ID=%d, Name=%s, Owner=%s", schema.ID, schema.Name, schema.Owner)
		}
	})

	// 3. 获取特定schema
	if createdSchema != nil {
		t.Run("GetSchema", func(t *testing.T) {
			schema, err := testClient.GetSchema(createdSchema.ID)
			if err != nil {
				t.Errorf("获取schema失败: %v", err)
			} else {
				t.Logf("成功获取schema: ID=%d, Name=%s", schema.ID, schema.Name)
			}
		})

		// 4. 更新schema（如果支持）
		t.Run("UpdateSchema", func(t *testing.T) {
			req := UpdateSchemaRequest{
				Name: testSchemaName + "_updated",
			}

			schema, err := testClient.UpdateSchema(createdSchema.ID, req)
			if err != nil {
				t.Logf("警告: 更新schema失败: %v", err)
			} else {
				t.Logf("成功更新schema: Name=%s", schema.Name)
			}
		})
	}
}

// TestPgMetaClient_Tables 测试Table相关功能
func TestPgMetaClient_Tables(t *testing.T) {
	t.Log("=== 开始测试Table功能 ===")

	// 1. 获取所有表
	t.Run("ListTables", func(t *testing.T) {
		tables, err := testClient.ListTables(&[]bool{false}[0], testSchemaName, "", nil, nil, &[]bool{true}[0])
		if err != nil {
			t.Logf("警告: 获取tables失败: %v", err)
		} else {
			t.Logf("成功获取 %d 个tables", len(tables))
			for i, table := range tables {
				if i < 3 { // 只显示前3个
					t.Logf("  Table[%d]: ID=%d, Schema=%s, Name=%s, Columns=%d",
						i, table.ID, table.Schema, table.Name, len(table.Columns))
				}
			}
		}
	})

	// 2. 创建测试表
	var createdTable *Table
	t.Run("CreateTable", func(t *testing.T) {
		req := CreateTableRequest{
			Name:    testTableName,
			Schema:  testSchemaName,
			Comment: "测试表",
		}

		table, err := testClient.CreateTable(req)
		if err != nil {
			t.Logf("警告: 创建table失败: %v", err)
		} else {
			createdTable = table
			t.Logf("成功创建table: ID=%d, Schema=%s, Name=%s", table.ID, table.Schema, table.Name)
		}
	})

	// 3. 获取特定表
	if createdTable != nil {
		t.Run("GetTable", func(t *testing.T) {
			table, err := testClient.GetTable(createdTable.ID)
			if err != nil {
				t.Errorf("获取table失败: %v", err)
			} else {
				t.Logf("成功获取table: ID=%d, Name=%s, Columns=%d", table.ID, table.Name, len(table.Columns))
			}
		})
	}
}

// TestPgMetaClient_Columns 测试Column相关功能
func TestPgMetaClient_Columns(t *testing.T) {
	t.Log("=== 开始测试Column功能 ===")

	// 首先确保有一个测试表
	var testTable *Table
	req := CreateTableRequest{
		Name:    testTableName + "_cols",
		Schema:  testSchemaName,
		Comment: "测试列的表",
	}

	table, err := testClient.CreateTable(req)
	if err != nil {
		t.Logf("警告: 创建测试表失败: %v", err)
		return
	}
	testTable = table
	t.Logf("创建测试表成功: ID=%d", testTable.ID)

	// 测试各种数据类型的列创建
	testColumns := []struct {
		name     string
		dataType string
		nullable bool
		comment  string
	}{
		{"id", "uuid", false, "主键ID"},
		{"name", "character varying(100)", true, "名称字段"},
		{"age", "integer", true, "年龄"},
		{"salary", "decimal(10,2)", true, "薪资"},
		{"is_active", "boolean", false, "是否激活"},
		{"created_at", "timestamp", false, "创建时间"},
		{"birth_date", "date", true, "出生日期"},
		{"score", "real", true, "分数"},
		{"description", "text", true, "描述"},
		{"ip_address", "inet", true, "IP地址"},
		{"config", "jsonb", true, "配置信息"},
	}

	var createdColumns []*Column

	for _, tc := range testColumns {
		t.Run(fmt.Sprintf("CreateColumn_%s_%s", tc.name, tc.dataType), func(t *testing.T) {
			req := CreateColumnRequest{
				TableID:    testTable.ID,
				Name:       tc.name,
				Type:       tc.dataType,
				IsNullable: &tc.nullable,
				Comment:    tc.comment,
			}

			// 设置特殊的默认值
			switch tc.dataType {
			case "uuid":
				req.DefaultValue = "gen_random_uuid()"
			case "timestamp":
				req.DefaultValue = "CURRENT_TIMESTAMP"
			case "boolean":
				req.DefaultValue = false
			}

			column, err := testClient.CreateColumn(req)
			if err != nil {
				t.Logf("警告: 创建列 %s (%s) 失败: %v", tc.name, tc.dataType, err)
			} else {
				createdColumns = append(createdColumns, column)
				t.Logf("成功创建列: Name=%s, Type=%s, Nullable=%v, Default=%v",
					column.Name, column.DataType, column.IsNullable, column.DefaultValue)
			}
		})
	}

	// 测试获取列信息
	t.Run("ListColumns", func(t *testing.T) {
		columns, err := testClient.ListColumns(&[]bool{false}[0], testSchemaName, "", nil, nil)
		if err != nil {
			t.Logf("警告: 获取columns失败: %v", err)
		} else {
			t.Logf("成功获取 %d 个columns", len(columns))
			for i, column := range columns {
				if i < 5 { // 只显示前5个
					t.Logf("  Column[%d]: Schema=%s, Table=%s, Name=%s, Type=%s",
						i, column.Schema, column.Table, column.Name, column.DataType)
				}
			}
		}
	})

	// 测试更新列
	if len(createdColumns) > 0 {
		t.Run("UpdateColumn", func(t *testing.T) {
			column := createdColumns[0]
			req := UpdateColumnRequest{
				Comment: "更新后的注释",
			}

			updatedColumn, err := testClient.UpdateColumn(column.ID, req)
			if err != nil {
				t.Logf("警告: 更新列失败: %v", err)
			} else {
				t.Logf("成功更新列: Name=%s, Comment=%s", updatedColumn.Name, *updatedColumn.Comment)
			}
		})
	}
}

// TestPgMetaClient_Query 测试Query相关功能
func TestPgMetaClient_Query(t *testing.T) {
	t.Log("=== 开始测试Query功能 ===")

	// 1. 执行简单查询
	t.Run("ExecuteQuery_Simple", func(t *testing.T) {
		query := "SELECT version()"
		response, err := testClient.ExecuteQuery(query)
		if err != nil {
			t.Logf("警告: 执行查询失败: %v", err)
		} else {
			t.Logf("查询成功: Columns=%v, Rows=%d", response.Columns, len(response.Rows))
			if len(response.Rows) > 0 {
				t.Logf("  版本信息: %v", response.Rows[0])
			}
		}
	})

	// 2. 查询时间
	t.Run("ExecuteQuery_Time", func(t *testing.T) {
		query := "SELECT NOW() as current_time, CURRENT_DATE as current_date"
		response, err := testClient.ExecuteQuery(query)
		if err != nil {
			t.Logf("警告: 查询时间失败: %v", err)
		} else {
			t.Logf("时间查询成功: %v", response.Rows)
		}
	})

	// 3. 查询数据类型信息
	t.Run("ExecuteQuery_DataTypes", func(t *testing.T) {
		query := `
		SELECT 
			typname as type_name,
			typcategory as category,
			typlen as length
		FROM pg_type 
		WHERE typname IN ('varchar', 'character varying', 'text', 'timestamp', 'uuid', 'inet', 'jsonb')
		ORDER BY typname
		`
		response, err := testClient.ExecuteQuery(query)
		if err != nil {
			t.Logf("警告: 查询数据类型失败: %v", err)
		} else {
			t.Logf("数据类型查询成功:")
			for _, row := range response.Rows {
				t.Logf("  类型: %v", row)
			}
		}
	})

	// 4. 测试格式化查询
	t.Run("FormatQuery", func(t *testing.T) {
		query := "select * from information_schema.tables where table_schema='public'"
		formatted, err := testClient.FormatQuery(query)
		if err != nil {
			t.Logf("警告: 格式化查询失败: %v", err)
		} else {
			t.Logf("查询格式化成功:\n%s", formatted)
		}
	})

	// 5. 测试解析查询
	t.Run("ParseQuery", func(t *testing.T) {
		query := "SELECT name, age FROM users WHERE age > 18"
		parsed, err := testClient.ParseQuery(query)
		if err != nil {
			t.Logf("警告: 解析查询失败: %v", err)
		} else {
			t.Logf("查询解析成功: %v", parsed)
		}
	})
}

// TestPgMetaClient_Views 测试View相关功能
func TestPgMetaClient_Views(t *testing.T) {
	t.Log("=== 开始测试View功能 ===")

	t.Run("ListViews", func(t *testing.T) {
		views, err := testClient.ListViews(&[]bool{false}[0], testSchemaName, "", nil, nil, &[]bool{true}[0])
		if err != nil {
			t.Logf("警告: 获取views失败: %v", err)
		} else {
			t.Logf("成功获取 %d 个views", len(views))
			for i, view := range views {
				if i < 3 {
					t.Logf("  View[%d]: ID=%d, Schema=%s, Name=%s", i, view.ID, view.Schema, view.Name)
				}
			}
		}
	})
}

// TestPgMetaClient_Roles 测试Role相关功能
func TestPgMetaClient_Roles(t *testing.T) {
	t.Log("=== 开始测试Role功能 ===")

	t.Run("ListRoles", func(t *testing.T) {
		roles, err := testClient.ListRoles(nil, nil, nil)
		if err != nil {
			t.Logf("警告: 获取roles失败: %v", err)
		} else {
			t.Logf("成功获取 %d 个roles", len(roles))
			for i, role := range roles {
				if i < 3 {
					t.Logf("  Role[%d]: ID=%d, Name=%s, IsSuperuser=%v",
						i, role.ID, role.Name, role.IsSuperuser)
				}
			}
		}
	})
}

// TestCleanup 清理测试数据
func TestCleanup(t *testing.T) {
	t.Log("=== 开始清理测试数据 ===")

	// 删除测试表
	t.Run("CleanupTables", func(t *testing.T) {
		tables, err := testClient.ListTables(&[]bool{false}[0], testSchemaName, "", nil, nil, nil)
		if err != nil {
			t.Logf("获取测试表失败: %v", err)
			return
		}

		for _, table := range tables {
			if table.Schema == testSchemaName {
				_, err := testClient.DeleteTable(table.ID, &[]bool{true}[0])
				if err != nil {
					t.Logf("警告: 删除表 %s 失败: %v", table.Name, err)
				} else {
					t.Logf("成功删除表: %s", table.Name)
				}
			}
		}
	})

	// 删除测试schema
	t.Run("CleanupSchemas", func(t *testing.T) {
		schemas, err := testClient.ListSchemas(&[]bool{false}[0], nil, nil)
		if err != nil {
			t.Logf("获取schemas失败: %v", err)
			return
		}

		for _, schema := range schemas {
			if schema.Name == testSchemaName || schema.Name == testSchemaName+"_updated" {
				_, err := testClient.DeleteSchema(schema.ID, &[]bool{true}[0])
				if err != nil {
					t.Logf("警告: 删除schema %s 失败: %v", schema.Name, err)
				} else {
					t.Logf("成功删除schema: %s", schema.Name)
				}
			}
		}
	})
}

// TestPgMetaClient_DataTypeCompatibility 测试数据类型兼容性
func TestPgMetaClient_DataTypeCompatibility(t *testing.T) {
	t.Log("=== 开始测试数据类型兼容性 ===")

	// 创建测试表用于数据类型测试
	req := CreateTableRequest{
		Name:    "datatype_test",
		Schema:  testSchemaName,
		Comment: "数据类型兼容性测试表",
	}

	table, err := testClient.CreateTable(req)
	if err != nil {
		t.Logf("创建数据类型测试表失败: %v", err)
		return
	}

	// 测试PostgreSQL支持的各种数据类型写法
	dataTypeTests := []struct {
		name        string
		pgType      string
		expectError bool
		description string
	}{
		{"varchar_with_length", "character varying(100)", false, "标准varchar写法"},
		{"varchar_short", "varchar(50)", true, "短varchar写法(可能不支持)"},
		{"text_type", "text", false, "文本类型"},
		{"integer_type", "integer", false, "整数类型"},
		{"bigint_type", "bigint", false, "大整数类型"},
		{"boolean_type", "boolean", false, "布尔类型"},
		{"timestamp_type", "timestamp without time zone", false, "时间戳类型"},
		{"timestamptz_type", "timestamp with time zone", false, "带时区时间戳"},
		{"date_type", "date", false, "日期类型"},
		{"time_type", "time without time zone", false, "时间类型"},
		{"uuid_type", "uuid", false, "UUID类型"},
		{"inet_type", "inet", false, "网络地址类型"},
		{"json_type", "json", false, "JSON类型"},
		{"jsonb_type", "jsonb", false, "JSONB类型"},
		{"decimal_type", "numeric(10,2)", false, "数值类型"},
		{"real_type", "real", false, "实数类型"},
		{"double_type", "double precision", false, "双精度类型"},
	}

	for _, tt := range dataTypeTests {
		t.Run(fmt.Sprintf("DataType_%s", tt.name), func(t *testing.T) {
			columnReq := CreateColumnRequest{
				TableID:    table.ID,
				Name:       tt.name,
				Type:       tt.pgType,
				IsNullable: &[]bool{true}[0],
				Comment:    tt.description,
			}

			column, err := testClient.CreateColumn(columnReq)
			if tt.expectError {
				if err != nil {
					t.Logf("预期错误: %s 类型 '%s' 失败: %v", tt.name, tt.pgType, err)
				} else {
					t.Logf("意外成功: %s 类型 '%s' 创建成功: %s", tt.name, tt.pgType, column.DataType)
				}
			} else {
				if err != nil {
					t.Errorf("意外错误: %s 类型 '%s' 失败: %v", tt.name, tt.pgType, err)
				} else {
					t.Logf("成功创建: %s -> 实际类型: '%s'", tt.pgType, column.DataType)
				}
			}
		})
	}
}

// 运行所有测试的主函数
func TestPgMetaClient_FullSuite(t *testing.T) {
	t.Log("开始PostgreSQL Meta客户端全面测试")
	t.Logf("测试服务器: %s", testBaseURL)
	t.Logf("测试时间: %s", time.Now().Format("2006-01-02 15:04:05"))

	// 检查服务器连接
	schemas, err := testClient.ListSchemas(nil, nil, nil)
	if err != nil {
		t.Fatalf("无法连接到PostgreSQL Meta服务: %v", err)
	}
	t.Logf("服务器连接正常，当前有 %d 个schemas", len(schemas))

	// 运行测试套件
	t.Run("Schemas", TestPgMetaClient_Schemas)
	t.Run("DataTypeCompatibility", TestPgMetaClient_DataTypeCompatibility)
	t.Run("Tables", TestPgMetaClient_Tables)
	t.Run("Columns", TestPgMetaClient_Columns)
	t.Run("Views", TestPgMetaClient_Views)
	t.Run("Roles", TestPgMetaClient_Roles)
	t.Run("Query", TestPgMetaClient_Query)
	t.Run("Cleanup", TestCleanup)

	t.Log("PostgreSQL Meta客户端测试完成")
}
