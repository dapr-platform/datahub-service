/*
 * @module service/interface_executor/interface_info
 * @description 接口信息相关的结构体和方法，提供统一的接口信息抽象
 * @architecture 适配器模式 - 为不同类型的接口提供统一的访问接口
 * @documentReference ai_docs/interface_executor.md
 * @stateFlow 接口信息获取 -> 信息封装 -> 统一访问接口
 * @rules 提供统一的接口信息访问方式，支持基础库和主题库接口
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs executor.go
 */

package interface_executor

import (
	"datahub-service/service/models"

	"gorm.io/gorm"
)

// InterfaceInfo 接口信息接口
type InterfaceInfo interface {
	GetID() string
	GetName() string
	GetType() string
	GetDataSourceID() string
	GetSchemaName() string
	GetTableName() string
	GetInterfaceConfig() map[string]interface{}
	GetParseConfig() map[string]interface{}
	GetTableFieldsConfig() []interface{}
	IsTableCreated() bool
}

// BasicLibraryInterfaceInfo 基础库接口信息
type BasicLibraryInterfaceInfo struct {
	*models.DataInterface
}

func (b *BasicLibraryInterfaceInfo) GetID() string           { return b.ID }
func (b *BasicLibraryInterfaceInfo) GetName() string         { return b.NameZh }
func (b *BasicLibraryInterfaceInfo) GetType() string         { return b.Type }
func (b *BasicLibraryInterfaceInfo) GetDataSourceID() string { return b.DataSourceID }
func (b *BasicLibraryInterfaceInfo) GetSchemaName() string   { return b.BasicLibrary.NameEn }
func (b *BasicLibraryInterfaceInfo) GetTableName() string    { return b.NameEn }
func (b *BasicLibraryInterfaceInfo) GetInterfaceConfig() map[string]interface{} {
	return b.InterfaceConfig
}
func (b *BasicLibraryInterfaceInfo) GetParseConfig() map[string]interface{} { return b.ParseConfig }
func (b *BasicLibraryInterfaceInfo) GetTableFieldsConfig() []interface{} {
	if b.TableFieldsConfig == nil {
		return []interface{}{}
	}
	// 将JSONB转换为[]interface{}
	result := make([]interface{}, 0)
	for _, v := range b.TableFieldsConfig {
		result = append(result, v)
	}
	return result
}
func (b *BasicLibraryInterfaceInfo) IsTableCreated() bool { return b.DataInterface.IsTableCreated }

// ThematicLibraryInterfaceInfo 主题库接口信息
type ThematicLibraryInterfaceInfo struct {
	*models.ThematicInterface
}

func (t *ThematicLibraryInterfaceInfo) GetID() string           { return t.ID }
func (t *ThematicLibraryInterfaceInfo) GetName() string         { return t.NameZh }
func (t *ThematicLibraryInterfaceInfo) GetType() string         { return t.Type }
func (t *ThematicLibraryInterfaceInfo) GetDataSourceID() string { return "" } // 主题接口不关联数据源
func (t *ThematicLibraryInterfaceInfo) GetSchemaName() string   { return t.ThematicLibrary.NameEn }
func (t *ThematicLibraryInterfaceInfo) GetTableName() string    { return t.NameEn }
func (t *ThematicLibraryInterfaceInfo) GetInterfaceConfig() map[string]interface{} {
	return t.InterfaceConfig
}
func (t *ThematicLibraryInterfaceInfo) GetParseConfig() map[string]interface{} { return t.ParseConfig }
func (t *ThematicLibraryInterfaceInfo) GetTableFieldsConfig() []interface{} {
	if t.TableFieldsConfig == nil {
		return []interface{}{}
	}
	// 将JSONB转换为[]interface{}
	result := make([]interface{}, 0)
	for _, v := range t.TableFieldsConfig {
		result = append(result, v)
	}
	return result
}
func (t *ThematicLibraryInterfaceInfo) IsTableCreated() bool {
	return t.ThematicInterface.IsTableCreated
}

// InterfaceInfoProviderInterface 接口信息提供者接口
type InterfaceInfoProviderInterface interface {
	GetBasicLibraryInterface(interfaceID string) (InterfaceInfo, error)
	GetThematicLibraryInterface(interfaceID string) (InterfaceInfo, error)
}

// InterfaceInfoProvider 接口信息提供者
type InterfaceInfoProvider struct {
	db *gorm.DB
}

// NewInterfaceInfoProvider 创建接口信息提供者
func NewInterfaceInfoProvider(db *gorm.DB) *InterfaceInfoProvider {
	return &InterfaceInfoProvider{db: db}
}

// GetBasicLibraryInterface 获取基础库接口信息
func (p *InterfaceInfoProvider) GetBasicLibraryInterface(interfaceID string) (InterfaceInfo, error) {
	var dataInterface models.DataInterface
	err := p.db.Preload("BasicLibrary").
		Preload("DataSource").Preload("Fields").Preload("CleanRules").
		First(&dataInterface, "id = ?", interfaceID).Error
	if err != nil {
		return nil, err
	}
	return &BasicLibraryInterfaceInfo{&dataInterface}, nil
}

// GetThematicLibraryInterface 获取主题库接口信息
func (p *InterfaceInfoProvider) GetThematicLibraryInterface(interfaceID string) (InterfaceInfo, error) {
	var thematicInterface models.ThematicInterface
	err := p.db.Preload("ThematicLibrary").
		First(&thematicInterface, "id = ?", interfaceID).Error
	if err != nil {
		return nil, err
	}
	return &ThematicLibraryInterfaceInfo{&thematicInterface}, nil
}
