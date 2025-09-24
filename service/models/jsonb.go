package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// 通用 JSON 类型
type JSONB map[string]interface{}

type JSONBArray []JSONB

// JSONBStringArray 用于存储字符串数组的 JSONB 类型
type JSONBStringArray []string

// JSONBGenericArray 用于存储任意类型数组的 JSONB 类型
type JSONBGenericArray []interface{}

func (j *JSONBArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败: 不是 []byte 或 string")
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONBArray) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// 实现 Scanner 接口
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败: 不是 []byte 或 string")
	}
	return json.Unmarshal(bytes, j)
}

// 实现 Valuer 接口
func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// JSONBStringArray 的 Scanner 接口实现
func (j *JSONBStringArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败: 不是 []byte 或 string")
	}
	return json.Unmarshal(bytes, j)
}

// JSONBStringArray 的 Valuer 接口实现
func (j JSONBStringArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// JSONBGenericArray 的 Scanner 接口实现
func (j *JSONBGenericArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败: 不是 []byte 或 string")
	}
	return json.Unmarshal(bytes, j)
}

// JSONBGenericArray 的 Valuer 接口实现
func (j JSONBGenericArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
