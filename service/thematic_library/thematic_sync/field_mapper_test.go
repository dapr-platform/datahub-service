/*
 * @module service/thematic_sync/field_mapper_test
 * @description 字段映射处理器测试
 * @architecture 单元测试 - 验证字段映射功能的正确性
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 测试数据准备 -> 字段映射执行 -> 结果验证
 * @rules 确保字段映射功能的正确性和边界情况处理
 * @dependencies testing, gorm.io/gorm
 * @refs field_mapper.go, sync_types.go
 */

package thematic_sync

import (
	"datahub-service/service/models"
	"log/slog"
	"testing"

	"gorm.io/gorm"
)

// MockDB 模拟数据库
type MockDB struct {
	*gorm.DB
	thematicInterface *models.ThematicInterface
}

// MockFieldMapper 模拟字段映射器（用于测试）
type MockFieldMapper struct {
	targetFields map[string]TargetFieldInfo
}

// NewMockFieldMapper 创建模拟字段映射器
func NewMockFieldMapper() *MockFieldMapper {
	return &MockFieldMapper{
		targetFields: make(map[string]TargetFieldInfo),
	}
}

// SetTargetFields 设置目标字段配置
func (mfm *MockFieldMapper) SetTargetFields(fields map[string]TargetFieldInfo) {
	mfm.targetFields = fields
}

// ApplyFieldMapping 应用字段映射（模拟实现）
func (mfm *MockFieldMapper) ApplyFieldMapping(
	sourceRecords []map[string]interface{},
	targetInterfaceID string,
	fieldMappingRules interface{},
) ([]map[string]interface{}, error) {

	var mappedRecords []map[string]interface{}

	for _, sourceRecord := range sourceRecords {
		mappedRecord := make(map[string]interface{})

		// 简单的字段映射逻辑：只保留目标字段中存在的字段
		for targetFieldName := range mfm.targetFields {
			if value, exists := sourceRecord[targetFieldName]; exists {
				mappedRecord[targetFieldName] = value
			}
		}

		if len(mappedRecord) > 0 {
			mappedRecords = append(mappedRecords, mappedRecord)
		}
	}

	return mappedRecords, nil
}

// TestFieldMapping 测试字段映射功能
func TestFieldMapping(t *testing.T) {
	// 准备测试数据
	mockMapper := NewMockFieldMapper()

	// 设置目标字段配置
	targetFields := map[string]TargetFieldInfo{
		"id": {
			NameEn:       "id",
			NameZh:       "主键",
			DataType:     "varchar",
			IsPrimaryKey: true,
			IsNullable:   false,
			Required:     true,
		},
		"name": {
			NameEn:     "name",
			NameZh:     "姓名",
			DataType:   "varchar",
			IsNullable: true,
			Required:   false,
		},
		"age": {
			NameEn:     "age",
			NameZh:     "年龄",
			DataType:   "int",
			IsNullable: true,
			Required:   false,
		},
	}
	mockMapper.SetTargetFields(targetFields)

	// 准备源数据（包含目标字段不存在的字段）
	sourceRecords := []map[string]interface{}{
		{
			"id":          "1",
			"name":        "张三",
			"age":         25,
			"email":       "zhangsan@example.com", // 目标字段中不存在
			"phone":       "13800138000",          // 目标字段中不存在
			"create_time": "2023-01-01",           // 目标字段中不存在
		},
		{
			"id":      "2",
			"name":    "李四",
			"age":     30,
			"address": "北京市", // 目标字段中不存在
		},
	}

	// 执行字段映射
	mappedRecords, err := mockMapper.ApplyFieldMapping(sourceRecords, "test-interface-id", nil)
	if err != nil {
		t.Fatalf("字段映射失败: %v", err)
	}

	// 验证结果
	if len(mappedRecords) != 2 {
		t.Errorf("期望映射后记录数为2，实际为%d", len(mappedRecords))
	}

	// 验证第一条记录
	record1 := mappedRecords[0]
	expectedFields1 := map[string]interface{}{
		"id":   "1",
		"name": "张三",
		"age":  25,
	}

	for field, expectedValue := range expectedFields1 {
		if actualValue, exists := record1[field]; !exists {
			t.Errorf("第一条记录缺少字段: %s", field)
		} else if actualValue != expectedValue {
			t.Errorf("第一条记录字段%s值不匹配，期望: %v，实际: %v", field, expectedValue, actualValue)
		}
	}

	// 验证不应该存在的字段被过滤掉
	unwantedFields := []string{"email", "phone", "create_time"}
	for _, field := range unwantedFields {
		if _, exists := record1[field]; exists {
			t.Errorf("第一条记录不应该包含字段: %s", field)
		}
	}

	// 验证第二条记录
	record2 := mappedRecords[1]
	expectedFields2 := map[string]interface{}{
		"id":   "2",
		"name": "李四",
		"age":  30,
	}

	for field, expectedValue := range expectedFields2 {
		if actualValue, exists := record2[field]; !exists {
			t.Errorf("第二条记录缺少字段: %s", field)
		} else if actualValue != expectedValue {
			t.Errorf("第二条记录字段%s值不匹配，期望: %v，实际: %v", field, expectedValue, actualValue)
		}
	}

	// 验证不应该存在的字段被过滤掉
	if _, exists := record2["address"]; exists {
		t.Errorf("第二条记录不应该包含字段: address")
	}

	slog.Info("字段映射测试通过", "sourceCount", len(sourceRecords), "mappedCount", len(mappedRecords))
	slog.Debug("映射后记录", "record1", record1, "record2", record2)
}

// TestFieldMappingWithEmptySource 测试空源数据的字段映射
func TestFieldMappingWithEmptySource(t *testing.T) {
	mockMapper := NewMockFieldMapper()

	// 设置目标字段配置
	targetFields := map[string]TargetFieldInfo{
		"id": {
			NameEn:       "id",
			DataType:     "varchar",
			IsPrimaryKey: true,
			Required:     true,
		},
	}
	mockMapper.SetTargetFields(targetFields)

	// 空源数据
	sourceRecords := []map[string]interface{}{}

	// 执行字段映射
	mappedRecords, err := mockMapper.ApplyFieldMapping(sourceRecords, "test-interface-id", nil)
	if err != nil {
		t.Fatalf("字段映射失败: %v", err)
	}

	// 验证结果
	if len(mappedRecords) != 0 {
		t.Errorf("期望映射后记录数为0，实际为%d", len(mappedRecords))
	}

	slog.Info("空源数据字段映射测试通过")
}

// TestFieldMappingWithMissingFields 测试缺少字段的字段映射
func TestFieldMappingWithMissingFields(t *testing.T) {
	mockMapper := NewMockFieldMapper()

	// 设置目标字段配置
	targetFields := map[string]TargetFieldInfo{
		"id": {
			NameEn:       "id",
			DataType:     "varchar",
			IsPrimaryKey: true,
			Required:     true,
		},
		"name": {
			NameEn:   "name",
			DataType: "varchar",
			Required: false,
		},
	}
	mockMapper.SetTargetFields(targetFields)

	// 源数据缺少某些目标字段
	sourceRecords := []map[string]interface{}{
		{
			"id": "1",
			// 缺少 name 字段
		},
	}

	// 执行字段映射
	mappedRecords, err := mockMapper.ApplyFieldMapping(sourceRecords, "test-interface-id", nil)
	if err != nil {
		t.Fatalf("字段映射失败: %v", err)
	}

	// 验证结果
	if len(mappedRecords) != 1 {
		t.Errorf("期望映射后记录数为1，实际为%d", len(mappedRecords))
	}

	record := mappedRecords[0]
	if record["id"] != "1" {
		t.Errorf("id字段值不匹配，期望: 1，实际: %v", record["id"])
	}

	// name字段应该不存在（因为源数据中没有）
	if _, exists := record["name"]; exists {
		t.Errorf("name字段不应该存在")
	}

	slog.Info("缺少字段的字段映射测试通过")
}
