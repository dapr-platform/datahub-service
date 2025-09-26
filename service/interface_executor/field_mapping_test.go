/*
 * @module service/interface_executor/field_mapping_test
 * @description 字段映射和数据转换逻辑的单元测试
 * @architecture 单元测试 - 验证字段类型转换功能的正确性
 * @documentReference ai_docs/interface_executor.md
 * @stateFlow 测试数据准备 -> 类型转换测试 -> 结果验证
 * @rules 确保各种数据类型转换的正确性和边界情况处理
 * @dependencies testing, github.com/stretchr/testify/assert
 * @refs field_mapping.go
 */

package interface_executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFieldMapper_ProcessValueForDatabase_Timestamp(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回字段配置
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{
		map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{
					"field_name": "created_at",
					"field_type": "timestamp",
				},
			},
		},
	})

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "time.Time转换",
			input:    time.Date(2023, 12, 25, 10, 30, 45, 0, time.UTC),
			expected: "2023-12-25 10:30:45.000",
		},
		{
			name:     "RFC3339字符串转换",
			input:    "2023-12-25T10:30:45Z",
			expected: "2023-12-25 10:30:45.000",
		},
		{
			name:     "标准时间字符串转换",
			input:    "2023-12-25 10:30:45",
			expected: "2023-12-25 10:30:45.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.ProcessValueForDatabase("created_at", tt.input, mockInterface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldMapper_ProcessValueForDatabase_Integer(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回字段配置
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{
		map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{
					"field_name": "user_id",
					"field_type": "int",
				},
			},
		},
	})

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "整数不变",
			input:    123,
			expected: 123,
		},
		{
			name:     "浮点数转整数",
			input:    123.45,
			expected: 123,
		},
		{
			name:     "字符串转整数",
			input:    "456",
			expected: 456,
		},
		{
			name:     "浮点字符串转整数",
			input:    "789.12",
			expected: 789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.ProcessValueForDatabase("user_id", tt.input, mockInterface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldMapper_ProcessValueForDatabase_Boolean(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回字段配置
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{
		map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{
					"field_name": "is_active",
					"field_type": "boolean",
				},
			},
		},
	})

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "布尔值true不变",
			input:    true,
			expected: true,
		},
		{
			name:     "布尔值false不变",
			input:    false,
			expected: false,
		},
		{
			name:     "字符串true转布尔",
			input:    "true",
			expected: true,
		},
		{
			name:     "字符串false转布尔",
			input:    "false",
			expected: false,
		},
		{
			name:     "字符串1转布尔",
			input:    "1",
			expected: true,
		},
		{
			name:     "字符串0转布尔",
			input:    "0",
			expected: false,
		},
		{
			name:     "整数1转布尔",
			input:    1,
			expected: true,
		},
		{
			name:     "整数0转布尔",
			input:    0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.ProcessValueForDatabase("is_active", tt.input, mockInterface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldMapper_ProcessValueForDatabase_String(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回字段配置
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{
		map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{
					"field_name": "username",
					"field_type": "varchar",
				},
			},
		},
	})

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "字符串不变",
			input:    "test_user",
			expected: "test_user",
		},
		{
			name:     "整数转字符串",
			input:    123,
			expected: "123",
		},
		{
			name:     "浮点数转字符串",
			input:    123.45,
			expected: "123.45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.ProcessValueForDatabase("username", tt.input, mockInterface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldMapper_ProcessValueForDatabase_UnknownType(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回空字段配置，应该回退到字段名推断
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{})

	// 测试时间字段名推断
	result := fm.ProcessValueForDatabase("created_at", "2023-12-25T10:30:45Z", mockInterface)
	expected := "2023-12-25 10:30:45.000"
	assert.Equal(t, expected, result)

	// 测试普通字段名，应该作为字符串处理
	result2 := fm.ProcessValueForDatabase("some_field", 123, mockInterface)
	expected2 := "123"
	assert.Equal(t, expected2, result2)
}

func TestFieldMapper_buildFieldTypeMapping(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回字段配置
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{
		map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{
					"field_name": "id",
					"field_type": "varchar",
				},
				map[string]interface{}{
					"field_name": "age",
					"field_type": "int",
				},
				map[string]interface{}{
					"field_name": "created_at",
					"field_type": "timestamp",
				},
			},
		},
	})

	// 构建字段类型映射
	fieldTypeMap := fm.buildFieldTypeMapping(mockInterface)

	// 验证映射结果
	assert.Equal(t, "varchar", fieldTypeMap["id"])
	assert.Equal(t, "int", fieldTypeMap["age"])
	assert.Equal(t, "timestamp", fieldTypeMap["created_at"])

	// 验证缓存功能
	fieldTypeMap2 := fm.buildFieldTypeMapping(mockInterface)
	assert.Equal(t, fieldTypeMap, fieldTypeMap2)
}

func TestFieldMapper_getFieldDataType(t *testing.T) {
	fm := NewFieldMapper()
	mockInterface := &MockInterfaceInfo{}

	// 设置模拟接口返回字段配置
	mockInterface.On("GetID").Return("test-interface-id")
	mockInterface.On("GetTableFieldsConfig").Return([]interface{}{
		map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{
					"field_name": "user_id",
					"field_type": "VARCHAR",
				},
			},
		},
	})

	// 测试已配置字段
	dataType := fm.getFieldDataType("user_id", mockInterface)
	assert.Equal(t, "varchar", dataType) // 应该转换为小写

	// 测试未配置的时间字段（回退到字段名推断）
	dataType2 := fm.getFieldDataType("created_at", mockInterface)
	assert.Equal(t, "timestamp", dataType2)

	// 测试未配置的ID字段（回退到字段名推断）
	dataType3 := fm.getFieldDataType("some_id", mockInterface)
	assert.Equal(t, "varchar", dataType3)

	// 测试未配置的普通字段（默认为varchar）
	dataType4 := fm.getFieldDataType("unknown_field", mockInterface)
	assert.Equal(t, "varchar", dataType4)
}
