/*
 * @module service/datasource/realtime_processor
 * @description 实时数据处理器，负责将实时数据源接收的数据自动写入关联的数据接口表
 * @architecture 观察者模式 - 实时数据源推送数据，处理器负责分发和写入
 * @documentReference ai_docs/datasource_req1.md
 * @stateFlow 注册接口 -> 接收数据 -> 应用字段映射 -> 批量写入表
 * @rules 支持多接口绑定、批量优化、字段映射、错误容错
 * @dependencies gorm.io/gorm, sync
 * @refs interface_executor/field_mapping.go, interface_executor/interface_info.go
 */

package datasource

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
)

// InterfaceInfo 接口信息抽象（避免循环依赖）
type InterfaceInfo interface {
	GetID() string
	GetSchemaName() string
	GetTableName() string
	GetParseConfig() map[string]interface{}
}

// DataWriter 数据写入器接口（避免循环依赖）
type DataWriter interface {
	// WriteData 写入数据到接口表
	WriteData(ctx context.Context, interfaceID string, data []map[string]interface{}) (int64, error)
}

// InterfaceLoader 接口加载器接口（避免循环依赖）
type InterfaceLoader interface {
	// LoadInterface 加载接口信息
	LoadInterface(ctx context.Context, interfaceID string) (InterfaceInfo, error)
}

// RealtimeDataProcessor 实时数据处理器接口
type RealtimeDataProcessor interface {
	// RegisterInterface 注册数据接口到处理器
	RegisterInterface(ctx context.Context, interfaceID string, dataSourceID string) error

	// UnregisterInterface 注销数据接口
	UnregisterInterface(interfaceID string) error

	// ProcessRealtimeData 处理实时接收的数据
	ProcessRealtimeData(ctx context.Context, dataSourceID string, data map[string]interface{}) error

	// GetProcessorStats 获取处理器统计信息
	GetProcessorStats() map[string]interface{}

	// SetDB 设置数据库连接
	SetDB(db *gorm.DB)

	// SetDataWriter 设置数据写入器
	SetDataWriter(writer DataWriter)

	// SetInterfaceLoader 设置接口加载器
	SetInterfaceLoader(loader InterfaceLoader)
}

// DefaultRealtimeDataProcessor 默认实时数据处理器实现
type DefaultRealtimeDataProcessor struct {
	mu sync.RWMutex

	// 数据源ID -> 接口ID列表的映射
	dataSourceInterfaces map[string][]string

	// 接口ID -> 接口信息的缓存
	interfaceCache map[string]InterfaceInfo

	// 批量写入缓冲
	dataBatches      map[string][]map[string]interface{} // interfaceID -> data batch
	batchMu          sync.RWMutex
	batchSize        int
	batchTimeout     time.Duration
	lastFlushTime    map[string]time.Time
	flushTimerCancel map[string]context.CancelFunc

	// 依赖
	db              *gorm.DB
	dataWriter      DataWriter
	interfaceLoader InterfaceLoader

	// 统计信息
	stats struct {
		sync.RWMutex
		totalProcessed  int64
		totalWritten    int64
		totalFailed     int64
		lastProcessedAt time.Time
		interfaceCount  int
		dataSourceCount int
	}
}

// NewDefaultRealtimeDataProcessor 创建默认实时数据处理器
func NewDefaultRealtimeDataProcessor() *DefaultRealtimeDataProcessor {
	return &DefaultRealtimeDataProcessor{
		dataSourceInterfaces: make(map[string][]string),
		interfaceCache:       make(map[string]InterfaceInfo),
		dataBatches:          make(map[string][]map[string]interface{}),
		lastFlushTime:        make(map[string]time.Time),
		flushTimerCancel:     make(map[string]context.CancelFunc),
		batchSize:            100,
		batchTimeout:         100 * time.Millisecond,
	}
}

// SetDB 设置数据库连接
func (p *DefaultRealtimeDataProcessor) SetDB(db *gorm.DB) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.db = db
}

// SetDataWriter 设置数据写入器
func (p *DefaultRealtimeDataProcessor) SetDataWriter(writer DataWriter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dataWriter = writer
}

// SetInterfaceLoader 设置接口加载器
func (p *DefaultRealtimeDataProcessor) SetInterfaceLoader(loader InterfaceLoader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interfaceLoader = loader
}

// RegisterInterface 注册数据接口到处理器
func (p *DefaultRealtimeDataProcessor) RegisterInterface(ctx context.Context, interfaceID string, dataSourceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.interfaceLoader == nil {
		return fmt.Errorf("接口加载器未初始化")
	}

	// 加载接口信息
	interfaceInfo, err := p.interfaceLoader.LoadInterface(ctx, interfaceID)
	if err != nil {
		return fmt.Errorf("加载接口信息失败: %w", err)
	}

	// 缓存接口信息
	p.interfaceCache[interfaceID] = interfaceInfo

	// 添加到数据源-接口映射
	interfaces := p.dataSourceInterfaces[dataSourceID]
	for _, id := range interfaces {
		if id == interfaceID {
			slog.Info("接口已注册，跳过", "interface_id", interfaceID, "datasource_id", dataSourceID)
			return nil // 已存在
		}
	}
	p.dataSourceInterfaces[dataSourceID] = append(interfaces, interfaceID)

	// 初始化批量缓冲
	p.batchMu.Lock()
	p.dataBatches[interfaceID] = make([]map[string]interface{}, 0, p.batchSize)
	p.lastFlushTime[interfaceID] = time.Now()
	p.batchMu.Unlock()

	// 更新统计
	p.updateStats()

	slog.Info("注册实时接口成功", "interface_id", interfaceID, "datasource_id", dataSourceID)
	return nil
}

// UnregisterInterface 注销数据接口
func (p *DefaultRealtimeDataProcessor) UnregisterInterface(interfaceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 从数据源-接口映射中移除
	for dataSourceID, interfaces := range p.dataSourceInterfaces {
		newInterfaces := make([]string, 0)
		for _, id := range interfaces {
			if id != interfaceID {
				newInterfaces = append(newInterfaces, id)
			}
		}
		p.dataSourceInterfaces[dataSourceID] = newInterfaces
	}

	// 刷新并清理批量缓冲
	p.batchMu.Lock()
	if batch, exists := p.dataBatches[interfaceID]; exists && len(batch) > 0 {
		// 刷新剩余数据
		go p.flushBatch(interfaceID, batch)
	}
	delete(p.dataBatches, interfaceID)
	delete(p.lastFlushTime, interfaceID)
	if cancel, exists := p.flushTimerCancel[interfaceID]; exists {
		cancel()
		delete(p.flushTimerCancel, interfaceID)
	}
	p.batchMu.Unlock()

	// 清理缓存
	delete(p.interfaceCache, interfaceID)

	// 更新统计
	p.updateStats()

	slog.Info("注销实时接口成功", "interface_id", interfaceID)
	return nil
}

// ProcessRealtimeData 处理实时接收的数据
func (p *DefaultRealtimeDataProcessor) ProcessRealtimeData(ctx context.Context, dataSourceID string, data map[string]interface{}) error {
	p.mu.RLock()
	interfaces := p.dataSourceInterfaces[dataSourceID]
	p.mu.RUnlock()

	if len(interfaces) == 0 {
		slog.Debug("数据源没有关联的接口，跳过处理", "datasource_id", dataSourceID)
		return nil
	}

	// 更新统计
	p.stats.Lock()
	p.stats.totalProcessed++
	p.stats.lastProcessedAt = time.Now()
	p.stats.Unlock()

	// 为每个关联的接口处理数据
	for _, interfaceID := range interfaces {
		if err := p.processDataForInterface(ctx, interfaceID, data); err != nil {
			slog.Error("处理接口数据失败", "interface_id", interfaceID, "error", err)
			p.stats.Lock()
			p.stats.totalFailed++
			p.stats.Unlock()
			// 继续处理其他接口，不中断
		}
	}

	return nil
}

// processDataForInterface 为特定接口处理数据
func (p *DefaultRealtimeDataProcessor) processDataForInterface(ctx context.Context, interfaceID string, data map[string]interface{}) error {
	p.mu.RLock()
	interfaceInfo, exists := p.interfaceCache[interfaceID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("接口信息未缓存: %s", interfaceID)
	}

	// 应用字段映射
	parseConfig := interfaceInfo.GetParseConfig()
	mappedData := applyFieldMapping(data, parseConfig)

	// 添加到批量缓冲
	p.batchMu.Lock()
	batch := p.dataBatches[interfaceID]
	batch = append(batch, mappedData)
	p.dataBatches[interfaceID] = batch

	shouldFlush := len(batch) >= p.batchSize
	timeoutReached := time.Since(p.lastFlushTime[interfaceID]) >= p.batchTimeout
	p.batchMu.Unlock()

	// 判断是否需要刷新
	if shouldFlush || timeoutReached {
		p.flushInterfaceBatch(interfaceID)
	} else {
		// 启动或重置超时定时器
		p.resetFlushTimer(interfaceID)
	}

	return nil
}

// applyFieldMapping 应用字段映射（简化版本，避免循环依赖）
func applyFieldMapping(data map[string]interface{}, parseConfig map[string]interface{}) map[string]interface{} {
	if parseConfig == nil {
		return data
	}

	fieldMappingInterface, exists := parseConfig["fieldMapping"]
	if !exists {
		return data
	}

	// 支持数组格式的字段映射
	fieldMappingArray, ok := fieldMappingInterface.([]interface{})
	if !ok {
		return data
	}

	// 构建源字段到目标字段的映射表
	sourceToTargetMap := make(map[string]string)
	for _, mappingItem := range fieldMappingArray {
		if mappingObj, ok := mappingItem.(map[string]interface{}); ok {
			if source, ok1 := mappingObj["source"].(string); ok1 {
				if target, ok2 := mappingObj["target"].(string); ok2 {
					sourceToTargetMap[source] = target
				}
			}
		}
	}

	// 应用映射
	mappedData := make(map[string]interface{})
	for sourceField, value := range data {
		targetField := sourceField
		if target, exists := sourceToTargetMap[sourceField]; exists {
			targetField = target
		}
		mappedData[targetField] = value
	}

	return mappedData
}

// flushInterfaceBatch 刷新特定接口的批量数据
func (p *DefaultRealtimeDataProcessor) flushInterfaceBatch(interfaceID string) {
	p.batchMu.Lock()
	batch := p.dataBatches[interfaceID]
	if len(batch) == 0 {
		p.batchMu.Unlock()
		return
	}

	// 复制批次数据并清空缓冲
	batchCopy := make([]map[string]interface{}, len(batch))
	copy(batchCopy, batch)
	p.dataBatches[interfaceID] = make([]map[string]interface{}, 0, p.batchSize)
	p.lastFlushTime[interfaceID] = time.Now()
	p.batchMu.Unlock()

	// 异步刷新
	go p.flushBatch(interfaceID, batchCopy)
}

// flushBatch 执行批量写入
func (p *DefaultRealtimeDataProcessor) flushBatch(interfaceID string, batch []map[string]interface{}) {
	if len(batch) == 0 {
		return
	}

	p.mu.RLock()
	dataWriter := p.dataWriter
	p.mu.RUnlock()

	if dataWriter == nil {
		slog.Error("无法刷新批次：数据写入器未初始化", "interface_id", interfaceID)
		return
	}

	ctx := context.Background()
	startTime := time.Now()

	// 使用数据写入器批量插入
	insertedRows, err := dataWriter.WriteData(ctx, interfaceID, batch)
	if err != nil {
		slog.Error("批量写入失败",
			"interface_id", interfaceID,
			"batch_size", len(batch),
			"error", err)
		p.stats.Lock()
		p.stats.totalFailed += int64(len(batch))
		p.stats.Unlock()
		return
	}

	duration := time.Since(startTime)
	slog.Info("批量写入成功",
		"interface_id", interfaceID,
		"inserted_rows", insertedRows,
		"batch_size", len(batch),
		"duration_ms", duration.Milliseconds())

	p.stats.Lock()
	p.stats.totalWritten += insertedRows
	p.stats.Unlock()
}

// resetFlushTimer 重置刷新定时器
func (p *DefaultRealtimeDataProcessor) resetFlushTimer(interfaceID string) {
	p.batchMu.Lock()
	defer p.batchMu.Unlock()

	// 取消旧定时器
	if cancel, exists := p.flushTimerCancel[interfaceID]; exists {
		cancel()
	}

	// 创建新定时器
	ctx, cancel := context.WithCancel(context.Background())
	p.flushTimerCancel[interfaceID] = cancel

	go func() {
		timer := time.NewTimer(p.batchTimeout)
		defer timer.Stop()

		select {
		case <-timer.C:
			p.flushInterfaceBatch(interfaceID)
		case <-ctx.Done():
			return
		}
	}()
}

// GetProcessorStats 获取处理器统计信息
func (p *DefaultRealtimeDataProcessor) GetProcessorStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.RLock()
	defer p.stats.RUnlock()

	return map[string]interface{}{
		"total_processed":   p.stats.totalProcessed,
		"total_written":     p.stats.totalWritten,
		"total_failed":      p.stats.totalFailed,
		"last_processed_at": p.stats.lastProcessedAt,
		"interface_count":   p.stats.interfaceCount,
		"datasource_count":  p.stats.dataSourceCount,
		"batch_size":        p.batchSize,
		"batch_timeout_ms":  p.batchTimeout.Milliseconds(),
		"pending_batches":   p.getPendingBatchesCount(),
	}
}

// getPendingBatchesCount 获取待刷新的批次数量
func (p *DefaultRealtimeDataProcessor) getPendingBatchesCount() int {
	p.batchMu.RLock()
	defer p.batchMu.RUnlock()

	count := 0
	for _, batch := range p.dataBatches {
		if len(batch) > 0 {
			count++
		}
	}
	return count
}

// updateStats 更新统计信息
func (p *DefaultRealtimeDataProcessor) updateStats() {
	p.stats.Lock()
	defer p.stats.Unlock()

	p.stats.interfaceCount = len(p.interfaceCache)
	p.stats.dataSourceCount = len(p.dataSourceInterfaces)
}

// 全局实时处理器实例
var (
	globalRealtimeProcessor RealtimeDataProcessor
	processorOnce           sync.Once
)

// GetGlobalRealtimeProcessor 获取全局实时处理器实例
func GetGlobalRealtimeProcessor() RealtimeDataProcessor {
	processorOnce.Do(func() {
		globalRealtimeProcessor = NewDefaultRealtimeDataProcessor()
	})
	return globalRealtimeProcessor
}

// InitGlobalRealtimeProcessor 初始化全局实时处理器(在服务启动时调用)
func InitGlobalRealtimeProcessor(db *gorm.DB, writer DataWriter, loader InterfaceLoader) {
	processorOnce.Do(func() {
		processor := NewDefaultRealtimeDataProcessor()
		processor.SetDB(db)
		processor.SetDataWriter(writer)
		processor.SetInterfaceLoader(loader)
		globalRealtimeProcessor = processor
		slog.Info("全局实时处理器初始化完成")
	})
}
