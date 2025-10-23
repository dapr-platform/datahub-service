/*
 * @module service/basic_library/realtime_adapter
 * @description 实时数据处理适配器，连接datasource包和interface_executor包
 * @architecture 适配器模式 - 避免循环依赖
 * @documentReference ai_docs/datasource_req1.md
 * @stateFlow 实现DataWriter和InterfaceLoader接口，桥接两个包
 * @rules 单一职责，只负责适配
 * @dependencies gorm.io/gorm
 * @refs service/datasource/realtime_processor.go, service/interface_executor
 */

package basic_library

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/interface_executor"
	"fmt"

	"gorm.io/gorm"
)

// RealtimeDataWriter 实时数据写入器适配器
type RealtimeDataWriter struct {
	db           *gorm.DB
	fieldMapper  *interface_executor.FieldMapper
	infoProvider *interface_executor.InterfaceInfoProvider
}

// NewRealtimeDataWriter 创建实时数据写入器
func NewRealtimeDataWriter(db *gorm.DB) *RealtimeDataWriter {
	return &RealtimeDataWriter{
		db:           db,
		fieldMapper:  interface_executor.NewFieldMapper(),
		infoProvider: interface_executor.NewInterfaceInfoProvider(db),
	}
}

// WriteData 写入数据到接口表
func (w *RealtimeDataWriter) WriteData(ctx context.Context, interfaceID string, data []map[string]interface{}) (int64, error) {
	// 加载接口信息
	interfaceInfo, err := w.infoProvider.GetBasicLibraryInterface(interfaceID)
	if err != nil {
		return 0, fmt.Errorf("加载接口信息失败: %w", err)
	}

	// 使用FieldMapper写入数据
	return w.fieldMapper.InsertBatchData(ctx, w.db, interfaceInfo, data)
}

// RealtimeInterfaceLoader 实时接口加载器适配器
type RealtimeInterfaceLoader struct {
	infoProvider *interface_executor.InterfaceInfoProvider
}

// NewRealtimeInterfaceLoader 创建实时接口加载器
func NewRealtimeInterfaceLoader(db *gorm.DB) *RealtimeInterfaceLoader {
	return &RealtimeInterfaceLoader{
		infoProvider: interface_executor.NewInterfaceInfoProvider(db),
	}
}

// LoadInterface 加载接口信息
func (l *RealtimeInterfaceLoader) LoadInterface(ctx context.Context, interfaceID string) (datasource.InterfaceInfo, error) {
	// 加载基础库接口信息
	interfaceInfo, err := l.infoProvider.GetBasicLibraryInterface(interfaceID)
	if err != nil {
		return nil, err
	}

	// 转换为datasource.InterfaceInfo接口
	return &InterfaceInfoAdapter{info: interfaceInfo}, nil
}

// InterfaceInfoAdapter 接口信息适配器
type InterfaceInfoAdapter struct {
	info interface_executor.InterfaceInfo
}

func (a *InterfaceInfoAdapter) GetID() string         { return a.info.GetID() }
func (a *InterfaceInfoAdapter) GetSchemaName() string { return a.info.GetSchemaName() }
func (a *InterfaceInfoAdapter) GetTableName() string  { return a.info.GetTableName() }
func (a *InterfaceInfoAdapter) GetParseConfig() map[string]interface{} {
	return a.info.GetParseConfig()
}
