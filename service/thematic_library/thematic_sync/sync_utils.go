/*
 * @module service/thematic_sync/sync_utils
 * @description 同步引擎工具函数集合
 * @architecture 工具类模式 - 提供通用的辅助函数
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 工具函数调用 -> 数据处理 -> 结果返回
 * @rules 确保工具函数的可重用性和稳定性
 * @dependencies fmt, strconv, strings
 * @refs sync_types.go
 */

package thematic_sync

import (
	"datahub-service/service/models"
	"fmt"
	"strconv"
	"strings"
)

// SyncUtils 同步工具类
type SyncUtils struct{}

// NewSyncUtils 创建同步工具实例
func NewSyncUtils() *SyncUtils {
	return &SyncUtils{}
}

// GetStringFromMap 从map中获取字符串值
func (su *SyncUtils) GetStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// CompareValues 比较两个值
func (su *SyncUtils) CompareValues(a, b interface{}) int {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// 尝试数值比较
	if aNum, aErr := strconv.ParseFloat(aStr, 64); aErr == nil {
		if bNum, bErr := strconv.ParseFloat(bStr, 64); bErr == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// 字符串比较
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// IsValidFieldValue 检查字段值是否有效
func (su *SyncUtils) IsValidFieldValue(value interface{}) bool {
	if value == nil {
		return false
	}

	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	return str != "" && str != "null" && str != "NULL" && str != "nil"
}

// CalculateRecordQuality 计算记录质量
func (su *SyncUtils) CalculateRecordQuality(record map[string]interface{}) float64 {
	if len(record) == 0 {
		return 0.0
	}

	nonNullCount := 0
	for _, value := range record {
		if value != nil && fmt.Sprintf("%v", value) != "" {
			nonNullCount++
		}
	}

	return float64(nonNullCount) / float64(len(record)) * 100
}

// ParseSQLDataSourceConfigs 解析SQL数据源配置
func (su *SyncUtils) ParseSQLDataSourceConfigs(request *SyncRequest) ([]SQLDataSourceConfig, bool) {
	// 检查请求配置中是否有SQL数据源配置
	if sqlConfigRaw, exists := request.Config["data_source_sql"]; exists {
		var sqlConfigs []SQLDataSourceConfig

		// 尝试直接转换
		if configSlice, ok := sqlConfigRaw.([]SQLDataSourceConfig); ok {
			return configSlice, true
		}

		// 尝试从接口数组转换
		if configSlice, ok := sqlConfigRaw.([]interface{}); ok {
			for _, configRaw := range configSlice {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := SQLDataSourceConfig{
						SQLQuery: su.GetStringFromMap(configMap, "sql_query"),
						Timeout:  30,    // 默认30秒
						MaxRows:  10000, // 默认1万行
					}

					// 解析参数
					if params, exists := configMap["parameters"]; exists {
						if paramsMap, ok := params.(map[string]interface{}); ok {
							config.Parameters = paramsMap
						}
					}

					// 解析超时时间
					if timeout, exists := configMap["timeout"]; exists {
						if timeoutInt, ok := timeout.(int); ok {
							config.Timeout = timeoutInt
						}
					}

					// 解析最大行数
					if maxRows, exists := configMap["max_rows"]; exists {
						if maxRowsInt, ok := maxRows.(int); ok {
							config.MaxRows = maxRowsInt
						}
					}

					sqlConfigs = append(sqlConfigs, config)
				}
			}

			if len(sqlConfigs) > 0 {
				return sqlConfigs, true
			}
		}
	}

	return nil, false
}

// GetThematicPrimaryKeyFields 获取主题接口的主键字段列表 - 通用工具方法
func GetThematicPrimaryKeyFields(thematicInterface *models.ThematicInterface) []string {
	var primaryKeys []string

	// 从TableFieldsConfig中解析主键字段
	if len(thematicInterface.TableFieldsConfig) > 0 {
		// 遍历字段配置，查找主键字段
		for fieldKey, fieldValue := range thematicInterface.TableFieldsConfig {
			if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
				// 检查是否为主键字段
				if isPrimary, exists := fieldMap["is_primary_key"]; exists {
					if isPrimaryBool, ok := isPrimary.(bool); ok && isPrimaryBool {
						// 优先使用name_en字段作为字段名
						if nameEn, exists := fieldMap["name_en"]; exists {
							if nameEnStr, ok := nameEn.(string); ok && nameEnStr != "" {
								primaryKeys = append(primaryKeys, nameEnStr)
							}
						} else {
							// 如果没有name_en，使用字段键名
							primaryKeys = append(primaryKeys, fieldKey)
						}
					}
				}
			}
		}
	}

	// 如果没有找到主键，尝试查找常见的主键字段名
	if len(primaryKeys) == 0 {
		commonPrimaryKeys := []string{"id", "uuid", "primary_key", "pk"}
		for _, pkField := range commonPrimaryKeys {
			if _, exists := thematicInterface.TableFieldsConfig[pkField]; exists {
				primaryKeys = []string{pkField}
				break
			}
		}
	}

	// 如果仍然没有找到主键，返回空切片
	return primaryKeys
}

// GetDataInterfacePrimaryKeyFields 获取数据接口的主键字段列表 - 通用工具方法
func GetDataInterfacePrimaryKeyFields(dataInterface *models.DataInterface) []string {
	var primaryKeys []string

	// 从TableFieldsConfig中解析主键字段
	if len(dataInterface.TableFieldsConfig) > 0 {
		// 遍历字段配置，查找主键字段
		for fieldKey, fieldValue := range dataInterface.TableFieldsConfig {
			if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
				// 检查是否为主键字段
				if isPrimary, exists := fieldMap["is_primary_key"]; exists {
					if isPrimaryBool, ok := isPrimary.(bool); ok && isPrimaryBool {
						// 优先使用name_en字段作为字段名
						if nameEn, exists := fieldMap["name_en"]; exists {
							if nameEnStr, ok := nameEn.(string); ok && nameEnStr != "" {
								primaryKeys = append(primaryKeys, nameEnStr)
							}
						} else {
							// 如果没有name_en，使用字段键名
							primaryKeys = append(primaryKeys, fieldKey)
						}
					}
				}
			}
		}
	}

	// 如果没有找到主键，尝试查找常见的主键字段名
	if len(primaryKeys) == 0 {
		commonPrimaryKeys := []string{"id", "uuid", "primary_key", "pk"}
		for _, pkField := range commonPrimaryKeys {
			if _, exists := dataInterface.TableFieldsConfig[pkField]; exists {
				primaryKeys = []string{pkField}
				break
			}
		}
	}

	// 如果仍然没有找到主键，返回空切片
	return primaryKeys
}
