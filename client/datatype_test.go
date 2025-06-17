/*
 * @module client/datatype_test
 * @description PostgreSQL数据类型兼容性专项测试
 * @architecture 测试架构 - 数据类型验证
 * @documentReference client/pgmeta.go
 * @stateFlow 创建测试表 -> 测试各种数据类型 -> 清理
 * @rules 使用public schema进行测试，避免权限问题
 * @dependencies testing, datahub-service/client
 * @refs PostgreSQL数据类型文档
 */

package client

import (
	"fmt"
	"testing"
	"time"
)

// TestDataTypeCompatibility 测试PostgreSQL数据类型兼容性
func TestDataTypeCompatibility(t *testing.T) {
	client := NewPgMetaClient("http://localhost:3001", "default")

	// 创建测试表
	tableName := fmt.Sprintf("datatype_test_%d", time.Now().Unix())
	createReq := CreateTableRequest{
		Name:    tableName,
		Schema:  "public", // 使用public schema
		Comment: "数据类型兼容性测试表",
	}

	table, err := client.CreateTable(createReq)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	t.Logf("✅ 成功创建测试表: %s (ID: %d)", table.Name, table.ID)

	// 确保测试结束后清理
	defer func() {
		_, err := client.DeleteTable(table.ID, &[]bool{true}[0])
		if err != nil {
			t.Logf("⚠️ 清理测试表失败: %v", err)
		} else {
			t.Logf("🧹 成功清理测试表: %s", table.Name)
		}
	}()

	// 测试各种数据类型
	dataTypeTests := []struct {
		name         string
		pgType       string
		expectError  bool
		description  string
		defaultValue interface{}
	}{
		// 字符串类型
		{"varchar_standard", "character varying(100)", false, "标准varchar写法", nil},
		{"varchar_short", "varchar(50)", true, "短varchar写法(可能不支持)", nil},
		{"text_type", "text", false, "文本类型", nil},

		// 数值类型
		{"integer_type", "integer", false, "整数类型", nil},
		{"bigint_type", "bigint", false, "大整数类型", nil},
		{"smallint_type", "smallint", false, "小整数类型", nil},
		{"decimal_type", "numeric(10,2)", false, "数值类型", nil},
		{"money_type", "money", false, "货币类型", nil},
		{"real_type", "real", false, "实数类型", nil},
		{"double_type", "double precision", false, "双精度类型", nil},

		// 布尔类型
		{"boolean_type", "boolean", false, "布尔类型", nil},

		// 时间类型
		{"timestamp_type", "timestamp without time zone", false, "时间戳类型", nil},
		{"timestamptz_type", "timestamp with time zone", false, "带时区时间戳", nil},
		{"date_type", "date", false, "日期类型", nil},
		{"time_type", "time without time zone", false, "时间类型", nil},
		{"timetz_type", "time with time zone", false, "带时区时间", nil},
		{"interval_type", "interval", false, "时间间隔类型", nil},

		// 特殊类型
		{"uuid_type", "uuid", false, "UUID类型", "gen_random_uuid()"},
		{"inet_type", "inet", false, "网络地址类型", nil},
		{"cidr_type", "cidr", false, "CIDR类型", nil},
		{"macaddr_type", "macaddr", false, "MAC地址类型", nil},

		// JSON类型
		{"json_type", "json", false, "JSON类型", nil},
		{"jsonb_type", "jsonb", false, "JSONB类型", nil},

		// 数组类型
		{"text_array", "text[]", false, "文本数组", nil},
		{"integer_array", "integer[]", false, "整数数组", nil},

		// 二进制类型
		{"bytea_type", "bytea", false, "二进制数据", nil},

		// 几何类型
		{"point_type", "point", false, "点类型", nil},
		{"line_type", "line", false, "线类型", nil},
		{"box_type", "box", false, "矩形类型", nil},
		{"circle_type", "circle", false, "圆类型", nil},
	}

	successCount := 0
	failCount := 0

	for _, tt := range dataTypeTests {
		t.Run(fmt.Sprintf("DataType_%s", tt.name), func(t *testing.T) {
			columnReq := CreateColumnRequest{
				TableID:    table.ID,
				Name:       tt.name,
				Type:       tt.pgType,
				IsNullable: &[]bool{true}[0],
				Comment:    tt.description,
			}

			// 设置默认值
			if tt.defaultValue != nil {
				columnReq.DefaultValue = tt.defaultValue
			}

			column, err := client.CreateColumn(columnReq)
			if tt.expectError {
				if err != nil {
					t.Logf("✅ 预期错误: %s 类型 '%s' 失败: %v", tt.name, tt.pgType, err)
					failCount++
				} else {
					t.Logf("⚠️ 意外成功: %s 类型 '%s' 创建成功，实际类型: %s", tt.name, tt.pgType, column.DataType)
					successCount++
				}
			} else {
				if err != nil {
					t.Errorf("❌ 意外错误: %s 类型 '%s' 失败: %v", tt.name, tt.pgType, err)
					failCount++
				} else {
					t.Logf("✅ 成功创建: %s -> 实际类型: '%s', 默认值: %v",
						tt.pgType, column.DataType, column.DefaultValue)
					successCount++
				}
			}
		})
	}

	t.Logf("📊 测试结果: 成功 %d 个, 失败 %d 个, 总计 %d 个",
		successCount, failCount, len(dataTypeTests))
}

// TestVarcharTypes 专门测试varchar类型的不同写法
func TestVarcharTypes(t *testing.T) {
	client := NewPgMetaClient("http://localhost:3001", "default")

	// 创建测试表
	tableName := fmt.Sprintf("varchar_test_%d", time.Now().Unix())
	createReq := CreateTableRequest{
		Name:   tableName,
		Schema: "public",
	}

	table, err := client.CreateTable(createReq)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	defer func() {
		client.DeleteTable(table.ID, &[]bool{true}[0])
	}()

	varcharTests := []struct {
		name   string
		pgType string
		expect string // 期望的实际类型
	}{
		{"varchar_100", "varchar(100)", "character varying"},
		{"character_varying_100", "character varying(100)", "character varying"},
		{"varchar_no_length", "varchar", "character varying"},
		{"character_varying_no_length", "character varying", "character varying"},
		{"text", "text", "text"},
	}

	for _, tt := range varcharTests {
		t.Run(tt.name, func(t *testing.T) {
			columnReq := CreateColumnRequest{
				TableID:    table.ID,
				Name:       tt.name,
				Type:       tt.pgType,
				IsNullable: &[]bool{true}[0],
				Comment:    fmt.Sprintf("测试类型: %s", tt.pgType),
			}

			column, err := client.CreateColumn(columnReq)
			if err != nil {
				t.Logf("❌ %s 类型创建失败: %v", tt.pgType, err)
			} else {
				t.Logf("✅ %s -> %s", tt.pgType, column.DataType)
				// 验证类型是否符合预期
				if column.DataType != tt.expect && tt.expect != "" {
					t.Logf("⚠️ 类型不匹配: 期望 %s, 实际 %s", tt.expect, column.DataType)
				}
			}
		})
	}
}

// TestNumericTypes 专门测试数值类型
func TestNumericTypes(t *testing.T) {
	client := NewPgMetaClient("http://localhost:3001", "default")

	// 创建测试表
	tableName := fmt.Sprintf("numeric_test_%d", time.Now().Unix())
	createReq := CreateTableRequest{
		Name:   tableName,
		Schema: "public",
	}

	table, err := client.CreateTable(createReq)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	defer func() {
		client.DeleteTable(table.ID, &[]bool{true}[0])
	}()

	numericTests := []struct {
		name   string
		pgType string
	}{
		{"decimal_10_2", "decimal(10,2)"},
		{"numeric_10_2", "numeric(10,2)"},
		{"decimal_no_params", "decimal"},
		{"numeric_no_params", "numeric"},
		{"float4", "float4"},
		{"float8", "float8"},
		{"real", "real"},
		{"double_precision", "double precision"},
	}

	for _, tt := range numericTests {
		t.Run(tt.name, func(t *testing.T) {
			columnReq := CreateColumnRequest{
				TableID:    table.ID,
				Name:       tt.name,
				Type:       tt.pgType,
				IsNullable: &[]bool{true}[0],
				Comment:    fmt.Sprintf("测试类型: %s", tt.pgType),
			}

			column, err := client.CreateColumn(columnReq)
			if err != nil {
				t.Logf("❌ %s 类型创建失败: %v", tt.pgType, err)
			} else {
				t.Logf("✅ %s -> %s", tt.pgType, column.DataType)
			}
		})
	}
}
