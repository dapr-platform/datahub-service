/*
 * @module client/fix_test
 * @description 验证PostgreSQL Meta API修复效果
 * @architecture 测试架构 - 修复验证
 * @documentReference service/basic_library/schema_service.go
 * @stateFlow 模拟前端参数 -> 应用修复逻辑 -> 验证结果
 * @rules 使用真实的前端参数测试
 * @dependencies testing, datahub-service/client
 * @refs PostgreSQL Meta API兼容性
 */

package client

import (
	"fmt"
	"testing"
	"time"
)

// TestSchemaServiceFix 测试修复后的效果
func TestSchemaServiceFix(t *testing.T) {
	client := NewPgMetaClient("http://localhost:3001", "default")

	// 模拟前端实际参数 (就是用户遇到问题的参数)
	tableName := fmt.Sprintf("fix_test_%d", time.Now().Unix())
	createReq := CreateTableRequest{
		Name:    tableName,
		Schema:  "public",
		Comment: "修复测试表",
	}

	table, err := client.CreateTable(createReq)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	defer func() {
		client.DeleteTable(table.ID, &[]bool{true}[0])
	}()

	// 前端传递的真实参数
	testFields := []struct {
		nameEn       string
		dataType     string
		isNullable   bool
		isPrimaryKey bool
		defaultValue string
		dataLength   float64
		description  string
	}{
		{"build_time_str", "varchar", false, false, "", 100, "构建时间字符串"},
		{"created_by", "varchar", false, false, "", 20, "创建者"},
		{"created_time", "timestamp", false, false, "", 0, "创建时间"},
		{"id", "uuid", false, true, "", 0, "主键ID"},
		{"ip", "inet", false, false, "", 0, "IP地址"},
		{"status", "boolean", false, false, "", 0, "状态"},
	}

	for _, field := range testFields {
		t.Run(fmt.Sprintf("Column_%s", field.nameEn), func(t *testing.T) {
			// 应用修复后的逻辑
			var finalType string
			switch field.dataType {
			case "varchar":
				finalType = "varchar" // 不带长度
			case "timestamp":
				finalType = "timestamp" // 不带时区
			default:
				finalType = field.dataType
			}

			// 处理默认值
			var defaultValue interface{}
			if field.defaultValue != "" {
				switch field.dataType {
				case "varchar":
					defaultValue = fmt.Sprintf("'%s'", field.defaultValue)
				case "uuid":
					if field.defaultValue == "gen_random_uuid()" {
						defaultValue = nil // 不设置函数默认值
					}
				default:
					defaultValue = field.defaultValue
				}
			}

			columnReq := CreateColumnRequest{
				TableID:      table.ID,
				Name:         field.nameEn,
				Type:         finalType,
				IsNullable:   &field.isNullable,
				IsPrimaryKey: &field.isPrimaryKey,
				Comment:      field.description,
			}

			if defaultValue != nil {
				columnReq.DefaultValue = defaultValue
			}

			column, err := client.CreateColumn(columnReq)
			if err != nil {
				t.Errorf("❌ 创建列 %s (%s) 失败: %v", field.nameEn, finalType, err)
			} else {
				t.Logf("✅ 成功创建列: %s -> %s", field.nameEn, column.DataType)
			}
		})
	}

	t.Log("🎉 修复测试完成！")
}
