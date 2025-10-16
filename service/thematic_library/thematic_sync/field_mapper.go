/*
 * @module service/thematic_sync/field_mapper
 * @description 字段映射处理器，负责源字段到目标字段的映射和过滤
 * @architecture 策略模式 - 支持多种字段映射策略
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 字段配置获取 -> 映射规则应用 -> 字段过滤 -> 数据转换
 * @rules 确保字段映射的准确性和完整性，支持字段转换和默认值
 * @dependencies gorm.io/gorm, fmt, encoding/json
 * @refs sync_types.go, models/thematic_sync.go
 */

package thematic_sync

import (
	"log/slog"
	"datahub-service/service/models"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// FieldMapper 字段映射处理器
type FieldMapper struct {
	db *gorm.DB
}

// NewFieldMapper 创建字段映射处理器
func NewFieldMapper(db *gorm.DB) *FieldMapper {
	return &FieldMapper{db: db}
}

// TargetFieldInfo 目标字段信息
type TargetFieldInfo struct {
	NameEn       string      `json:"name_en"`
	NameZh       string      `json:"name_zh"`
	DataType     string      `json:"data_type"`
	IsPrimaryKey bool        `json:"is_primary_key"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue interface{} `json:"default_value"`
	Required     bool        `json:"required"`
}

// FieldMappingResult 字段映射结果
type FieldMappingResult struct {
	MappedRecord    map[string]interface{} `json:"mapped_record"`
	MissingFields   []string               `json:"missing_fields"`
	ExtraFields     []string               `json:"extra_fields"`
	AppliedMappings []string               `json:"applied_mappings"`
}

// ApplyFieldMapping 应用字段映射
func (fm *FieldMapper) ApplyFieldMapping(
	sourceRecords []map[string]interface{},
	targetInterfaceID string,
	fieldMappingRules interface{},
) ([]map[string]interface{}, error) {

	// 获取目标接口的字段配置
	targetFields, err := fm.getTargetFieldsConfig(targetInterfaceID)
	if err != nil {
		return nil, fmt.Errorf("获取目标字段配置失败: %w", err)
	}

	// 解析字段映射规则
	mappingRules, err := fm.parseFieldMappingRules(fieldMappingRules)
	if err != nil {
		fmt.Printf("[DEBUG] 解析字段映射规则失败: %v，使用默认映射\n", err)
		mappingRules = nil
	}

	var mappedRecords []map[string]interface{}

	for i, sourceRecord := range sourceRecords {
		mappedRecord, err := fm.mapSingleRecord(sourceRecord, targetFields, mappingRules)
		if err != nil {
			fmt.Printf("[WARNING] 映射记录 %d 失败: %v\n", i, err)
			continue
		}
		mappedRecords = append(mappedRecords, mappedRecord)
	}

	fmt.Printf("[DEBUG] 字段映射完成，源记录数: %d，映射后记录数: %d\n",
		len(sourceRecords), len(mappedRecords))

	return mappedRecords, nil
}

// getTargetFieldsConfig 获取目标接口的字段配置
func (fm *FieldMapper) getTargetFieldsConfig(targetInterfaceID string) (map[string]TargetFieldInfo, error) {
	var thematicInterface models.ThematicInterface
	if err := fm.db.First(&thematicInterface, "id = ?", targetInterfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	targetFields := make(map[string]TargetFieldInfo)

	// 从TableFieldsConfig中解析字段配置
	if len(thematicInterface.TableFieldsConfig) > 0 {
		for fieldKey, fieldValue := range thematicInterface.TableFieldsConfig {
			if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
				fieldInfo := TargetFieldInfo{
					NameEn:       fm.getStringFromMap(fieldMap, "name_en"),
					NameZh:       fm.getStringFromMap(fieldMap, "name_zh"),
					DataType:     fm.getStringFromMap(fieldMap, "data_type"),
					IsPrimaryKey: fm.getBoolFromMap(fieldMap, "is_primary_key"),
					IsNullable:   fm.getBoolFromMap(fieldMap, "is_nullable"),
					DefaultValue: fieldMap["default_value"],
					Required:     !fm.getBoolFromMap(fieldMap, "is_nullable"), // 非空字段视为必需
				}

				// 使用name_en作为字段名，如果没有则使用fieldKey
				fieldName := fieldInfo.NameEn
				if fieldName == "" {
					fieldName = fieldKey
				}

				targetFields[fieldName] = fieldInfo
			}
		}
	}

	slog.Debug("获取目标字段配置，字段数", "count", len(targetFields))
	for fieldName, fieldInfo := range targetFields {
		fmt.Printf("[DEBUG] 目标字段: %s (类型: %s, 主键: %v, 可空: %v)\n",
			fieldName, fieldInfo.DataType, fieldInfo.IsPrimaryKey, fieldInfo.IsNullable)
	}

	return targetFields, nil
}

// parseFieldMappingRules 解析字段映射规则
func (fm *FieldMapper) parseFieldMappingRules(fieldMappingRules interface{}) (map[string]string, error) {
	if fieldMappingRules == nil {
		return nil, nil
	}

	mappingRules := make(map[string]string)

	// 尝试解析不同格式的映射规则
	switch rules := fieldMappingRules.(type) {
	case map[string]interface{}:
		// 处理 FieldMappingRules 结构
		if mappings, exists := rules["mappings"]; exists {
			if mappingSlice, ok := mappings.([]interface{}); ok {
				for _, mappingRaw := range mappingSlice {
					if mappingMap, ok := mappingRaw.(map[string]interface{}); ok {
						sourceField := fm.getStringFromMap(mappingMap, "source_field")
						targetField := fm.getStringFromMap(mappingMap, "target_field")
						if sourceField != "" && targetField != "" {
							mappingRules[sourceField] = targetField
						}
					}
				}
			}
		}
	case []interface{}:
		// 处理简单的映射数组
		for _, mappingRaw := range rules {
			if mappingMap, ok := mappingRaw.(map[string]interface{}); ok {
				sourceField := fm.getStringFromMap(mappingMap, "source_field")
				targetField := fm.getStringFromMap(mappingMap, "target_field")
				if sourceField != "" && targetField != "" {
					mappingRules[sourceField] = targetField
				}
			}
		}
	}

	slog.Debug("解析字段映射规则，规则数", "count", len(mappingRules))
	for source, target := range mappingRules {
		fmt.Printf("[DEBUG] 映射规则: %s -> %s\n", source, target)
	}

	return mappingRules, nil
}

// mapSingleRecord 映射单条记录
func (fm *FieldMapper) mapSingleRecord(
	sourceRecord map[string]interface{},
	targetFields map[string]TargetFieldInfo,
	mappingRules map[string]string,
) (map[string]interface{}, error) {

	mappedRecord := make(map[string]interface{})
	var missingFields []string

	// 遍历目标字段，进行映射
	for targetFieldName, targetFieldInfo := range targetFields {
		var value interface{}
		var found bool

		// 1. 首先检查是否有显式的映射规则
		for sourceField, mappedTarget := range mappingRules {
			if mappedTarget == targetFieldName {
				if sourceValue, exists := sourceRecord[sourceField]; exists {
					value = sourceValue
					found = true
					break
				}
			}
		}

		// 2. 如果没有找到映射规则，尝试直接匹配字段名
		if !found {
			if sourceValue, exists := sourceRecord[targetFieldName]; exists {
				value = sourceValue
				found = true
			}
		}

		// 3. 如果仍然没有找到，尝试使用中文名匹配
		if !found && targetFieldInfo.NameZh != "" {
			if sourceValue, exists := sourceRecord[targetFieldInfo.NameZh]; exists {
				value = sourceValue
				found = true
			}
		}

		// 4. 处理未找到的字段
		if !found {
			if targetFieldInfo.DefaultValue != nil {
				// 使用默认值
				value = targetFieldInfo.DefaultValue
				found = true
			} else if targetFieldInfo.Required && !targetFieldInfo.IsNullable {
				// 必需字段但没有找到值，尝试提供系统默认值
				defaultValue := fm.getSystemDefaultValue(targetFieldName, targetFieldInfo.DataType)
				if defaultValue != nil {
					value = defaultValue
					found = true
					slog.Debug("为必需字段 %s 使用系统默认值", "value", targetFieldName, defaultValue)
				} else {
					// 无法提供默认值的必需字段
					missingFields = append(missingFields, targetFieldName)
					continue
				}
			} else {
				// 可空字段，设置为nil
				value = nil
			}
		}

		// 5. 数据类型转换
		convertedValue, err := fm.convertFieldValue(value, targetFieldInfo.DataType)
		if err != nil {
			fmt.Printf("[WARNING] 字段 %s 类型转换失败: %v，使用原值\n", targetFieldName, err)
			convertedValue = value
		}

		mappedRecord[targetFieldName] = convertedValue
	}

	// 检查是否有缺失的必需字段
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("缺失必需字段: %s", strings.Join(missingFields, ", "))
	}

	return mappedRecord, nil
}

// convertFieldValue 转换字段值类型
func (fm *FieldMapper) convertFieldValue(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	// 根据目标类型进行转换
	switch strings.ToLower(targetType) {
	case "varchar", "text", "char", "string":
		return fmt.Sprintf("%v", value), nil
	case "int", "integer", "int4":
		switch v := value.(type) {
		case int:
			return v, nil
		case int64:
			return int(v), nil
		case float64:
			return int(v), nil
		case string:
			// 尝试解析字符串为整数
			if v == "" {
				return 0, nil
			}
			return value, nil // 让数据库处理转换
		default:
			return value, nil
		}
	case "bigint", "int8":
		switch v := value.(type) {
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		default:
			return value, nil
		}
	case "float", "float4", "float8", "decimal", "numeric":
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return value, nil
		}
	case "bool", "boolean":
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return strings.ToLower(v) == "true" || v == "1", nil
		case int:
			return v != 0, nil
		default:
			return value, nil
		}
	case "timestamp", "timestamptz", "datetime":
		// 时间类型保持原值，让数据库处理
		return value, nil
	case "jsonb", "json":
		// JSON类型保持原值
		return value, nil
	default:
		// 未知类型，保持原值
		return value, nil
	}
}

// getStringFromMap 从map中获取字符串值
func (fm *FieldMapper) getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// getBoolFromMap 从map中获取布尔值
func (fm *FieldMapper) getBoolFromMap(m map[string]interface{}, key string) bool {
	if value, exists := m[key]; exists {
		if boolVal, ok := value.(bool); ok {
			return boolVal
		}
		// 尝试从字符串转换
		strVal := fmt.Sprintf("%v", value)
		return strVal == "true" || strVal == "1" || strVal == "yes"
	}
	return false
}

// GetTargetFieldNames 获取目标字段名列表
func (fm *FieldMapper) GetTargetFieldNames(targetInterfaceID string) ([]string, error) {
	targetFields, err := fm.getTargetFieldsConfig(targetInterfaceID)
	if err != nil {
		return nil, err
	}

	var fieldNames []string
	for fieldName := range targetFields {
		fieldNames = append(fieldNames, fieldName)
	}

	return fieldNames, nil
}

// ValidateFieldMapping 验证字段映射配置
func (fm *FieldMapper) ValidateFieldMapping(
	targetInterfaceID string,
	fieldMappingRules interface{},
) error {
	targetFields, err := fm.getTargetFieldsConfig(targetInterfaceID)
	if err != nil {
		return fmt.Errorf("获取目标字段配置失败: %w", err)
	}

	mappingRules, err := fm.parseFieldMappingRules(fieldMappingRules)
	if err != nil {
		return fmt.Errorf("解析字段映射规则失败: %w", err)
	}

	// 检查映射规则中的目标字段是否存在
	for sourceField, targetField := range mappingRules {
		if _, exists := targetFields[targetField]; !exists {
			return fmt.Errorf("映射规则中的目标字段 '%s' (来源: %s) 在目标接口中不存在",
				targetField, sourceField)
		}
	}

	return nil
}

// getSystemDefaultValue 获取系统默认值
func (fm *FieldMapper) getSystemDefaultValue(fieldName, dataType string) interface{} {
	// 根据字段名和数据类型提供系统默认值
	fieldNameLower := strings.ToLower(fieldName)

	// 常见的系统字段默认值
	switch fieldNameLower {
	case "created_by", "updated_by":
		return "system"
	case "created_at", "updated_at":
		return time.Now()
	case "status":
		return "active"
	case "version":
		return 1
	case "is_deleted", "deleted":
		return false
	case "is_active", "active":
		return true
	case "sort_order", "order_num":
		return 0
	}

	// 根据数据类型提供默认值
	switch strings.ToLower(dataType) {
	case "varchar", "text", "char", "string":
		// 字符串类型，如果是必需字段，提供空字符串或系统值
		if strings.Contains(fieldNameLower, "name") {
			return "未命名"
		}
		if strings.Contains(fieldNameLower, "code") {
			return "AUTO_" + fmt.Sprintf("%d", time.Now().Unix())
		}
		return ""
	case "int", "integer", "int4", "bigint", "int8":
		return 0
	case "float", "float4", "float8", "decimal", "numeric":
		return 0.0
	case "bool", "boolean":
		return false
	case "timestamp", "timestamptz", "datetime":
		return time.Now()
	case "jsonb", "json":
		return "{}"
	default:
		return nil
	}
}
