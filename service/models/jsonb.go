package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// 通用 JSON 类型
type JSONB map[string]interface{}

type JSONBArray []JSONB

func (j *JSONBArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言失败: 不是 []byte")
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONBArray) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// 实现 Scanner 接口
func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言失败: 不是 []byte")
	}
	return json.Unmarshal(bytes, j)
}

// 实现 Valuer 接口
func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}
