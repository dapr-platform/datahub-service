/**
 * @module data_converter
 * @description 数据转换工具模块，负责类型转换、编码转换、格式转换、时间处理等功能
 * @architecture 工具函数模式，提供静态转换方法集合
 * @documentReference 参考 ai_docs/basic_library_process_impl.md 第8.2节
 * @stateFlow 无状态转换：输入 -> 转换逻辑 -> 输出
 * @rules
 *   - 转换操作需要处理异常情况
 *   - 类型转换需要保证数据精度
 *   - 编码转换需要支持多种字符集
 *   - 时间转换需要考虑时区问题
 * @dependencies
 *   - reflect: 反射支持
 *   - encoding/*: 编码转换
 *   - time: 时间处理
 *   - strconv: 字符串转换
 * @refs
 *   - service/sync_engine/*: 数据同步引擎
 *   - service/data_quality/*: 数据质量模块
 */

package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DataConverter 数据转换器
type DataConverter struct{}

// NewDataConverter 创建新的数据转换器实例
func NewDataConverter() *DataConverter {
	return &DataConverter{}
}

// 类型转换功能

// ToString 转换为字符串
func (dc *DataConverter) ToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return strconv.FormatBool(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		// 尝试JSON序列化
		if data, err := json.Marshal(value); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", value)
	}
}

// ToInt 转换为整数
func (dc *DataConverter) ToInt(value interface{}) (int, error) {
	if value == nil {
		return 0, fmt.Errorf("nil值无法转换为整数")
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("无法将类型 %T 转换为整数", value)
	}
}

// ToFloat 转换为浮点数
func (dc *DataConverter) ToFloat(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("nil值无法转换为浮点数")
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(v).Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Uint()), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("无法将类型 %T 转换为浮点数", value)
	}
}

// ToBool 转换为布尔值
func (dc *DataConverter) ToBool(value interface{}) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("nil值无法转换为布尔值")
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() != 0, nil
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0, nil
	default:
		return false, fmt.Errorf("无法将类型 %T 转换为布尔值", value)
	}
}

// ToByteArray 转换为字节数组
func (dc *DataConverter) ToByteArray(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("nil值无法转换为字节数组")
	}

	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		// 尝试JSON序列化
		return json.Marshal(value)
	}
}

// ConvertType 通用类型转换
func (dc *DataConverter) ConvertType(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch strings.ToLower(targetType) {
	case "string", "varchar", "text":
		return dc.ToString(value), nil
	case "int", "integer", "int32":
		return dc.ToInt(value)
	case "int64", "bigint":
		if intVal, err := dc.ToInt(value); err != nil {
			return nil, err
		} else {
			return int64(intVal), nil
		}
	case "float", "float32":
		if floatVal, err := dc.ToFloat(value); err != nil {
			return nil, err
		} else {
			return float32(floatVal), nil
		}
	case "float64", "double":
		return dc.ToFloat(value)
	case "bool", "boolean":
		return dc.ToBool(value)
	case "bytes", "blob":
		return dc.ToByteArray(value)
	case "json":
		return json.Marshal(value)
	default:
		return value, nil // 不支持的类型保持原样
	}
}

// 编码转换功能

// ConvertEncoding 编码转换
func (dc *DataConverter) ConvertEncoding(data []byte, fromEncoding, toEncoding string) ([]byte, error) {
	switch strings.ToLower(fromEncoding) {
	case "gbk", "gb2312":
		// GBK/GB2312 到 UTF-8
		if strings.ToLower(toEncoding) == "utf-8" {
			decoder := simplifiedchinese.GBK.NewDecoder()
			result, _, err := transform.Bytes(decoder, data)
			return result, err
		}
	case "utf-8":
		// UTF-8 到 GBK/GB2312
		if strings.ToLower(toEncoding) == "gbk" || strings.ToLower(toEncoding) == "gb2312" {
			encoder := simplifiedchinese.GBK.NewEncoder()
			result, _, err := transform.Bytes(encoder, data)
			return result, err
		}
	}

	// 默认情况下，如果不需要转换或不支持的编码，返回原数据
	return data, nil
}

// Base64Encode Base64编码
func (dc *DataConverter) Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode Base64解码
func (dc *DataConverter) Base64Decode(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// 格式转换功能

// FormatJSON 格式化JSON
func (dc *DataConverter) FormatJSON(data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// ParseJSON 解析JSON
func (dc *DataConverter) ParseJSON(jsonStr string) (interface{}, error) {
	var result interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

// FormatNumber 格式化数字
func (dc *DataConverter) FormatNumber(value interface{}, precision int) (string, error) {
	floatVal, err := dc.ToFloat(value)
	if err != nil {
		return "", err
	}

	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, floatVal), nil
}

// NormalizeString 标准化字符串
func (dc *DataConverter) NormalizeString(str string) string {
	// 去除首尾空格
	str = strings.TrimSpace(str)

	// 将多个连续空格替换为单个空格
	str = strings.Join(strings.Fields(str), " ")

	return str
}

// 时间处理功能

// ParseTime 解析时间字符串
func (dc *DataConverter) ParseTime(timeStr string, layouts []string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("时间字符串为空")
	}

	// 默认时间格式
	defaultLayouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"15:04:05",
		"2006/01/02 15:04:05",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"01/02/2006",
	}

	// 合并用户提供的格式和默认格式
	allLayouts := append(layouts, defaultLayouts...)

	for _, layout := range allLayouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间字符串: %s", timeStr)
}

// FormatTime 格式化时间
func (dc *DataConverter) FormatTime(t time.Time, layout string) string {
	if layout == "" {
		layout = time.RFC3339
	}
	return t.Format(layout)
}

// ConvertTimezone 转换时区
func (dc *DataConverter) ConvertTimezone(t time.Time, timezone string) (time.Time, error) {
	if timezone == "" {
		return t, nil
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return t, fmt.Errorf("无效的时区: %s, %v", timezone, err)
	}

	return t.In(loc), nil
}

// TimeToUnix 时间转Unix时间戳
func (dc *DataConverter) TimeToUnix(t time.Time) int64 {
	return t.Unix()
}

// UnixToTime Unix时间戳转时间
func (dc *DataConverter) UnixToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// 数据验证和清洗功能

// ValidateAndConvert 验证并转换数据
func (dc *DataConverter) ValidateAndConvert(value interface{}, rules map[string]interface{}) (interface{}, error) {
	result := value

	// 处理空值
	if nullValue, exists := rules["null_value"]; exists && value == nullValue {
		if defaultValue, exists := rules["default"]; exists {
			result = defaultValue
		} else {
			return nil, nil
		}
	}

	// 类型转换
	if targetType, exists := rules["type"]; exists {
		if typeStr, ok := targetType.(string); ok {
			converted, err := dc.ConvertType(result, typeStr)
			if err != nil {
				return nil, fmt.Errorf("类型转换失败: %v", err)
			}
			result = converted
		}
	}

	// 字符串处理
	if strVal, ok := result.(string); ok {
		// 去除空格
		if trim, exists := rules["trim"]; exists && trim.(bool) {
			strVal = strings.TrimSpace(strVal)
		}

		// 大小写转换
		if caseRule, exists := rules["case"]; exists {
			switch caseRule.(string) {
			case "upper":
				strVal = strings.ToUpper(strVal)
			case "lower":
				strVal = strings.ToLower(strVal)
			case "title":
				strVal = strings.Title(strVal)
			}
		}

		result = strVal
	}

	// 数值范围检查
	if min, exists := rules["min"]; exists {
		if numVal, err := dc.ToFloat(result); err == nil {
			if minVal, err := dc.ToFloat(min); err == nil {
				if numVal < minVal {
					return nil, fmt.Errorf("值 %v 小于最小值 %v", numVal, minVal)
				}
			}
		}
	}

	if max, exists := rules["max"]; exists {
		if numVal, err := dc.ToFloat(result); err == nil {
			if maxVal, err := dc.ToFloat(max); err == nil {
				if numVal > maxVal {
					return nil, fmt.Errorf("值 %v 大于最大值 %v", numVal, maxVal)
				}
			}
		}
	}

	return result, nil
}

// BatchConvert 批量转换
func (dc *DataConverter) BatchConvert(data []map[string]interface{}, rules map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, len(data))

	for i, record := range data {
		convertedRecord := make(map[string]interface{})

		for field, value := range record {
			if fieldRules, exists := rules[field]; exists {
				converted, err := dc.ValidateAndConvert(value, fieldRules)
				if err != nil {
					return nil, fmt.Errorf("记录 %d 字段 %s 转换失败: %v", i, field, err)
				}
				convertedRecord[field] = converted
			} else {
				convertedRecord[field] = value
			}
		}

		result[i] = convertedRecord
	}

	return result, nil
}
