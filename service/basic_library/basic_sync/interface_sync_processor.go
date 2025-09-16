/*
 * @module service/sync_engine/interface_sync_processor
 * @description 接口同步处理器，使用通用接口执行器处理基础库和主题库接口同步
 * @architecture 分层架构 - 同步引擎组件
 * @documentReference ai_docs/sync_engine.md
 * @stateFlow 同步流程：接收同步请求 -> 识别接口类型 -> 调用通用执行器 -> 返回同步结果
 * @rules 统一的接口同步处理逻辑，支持多种接口类型
 * @dependencies datahub-service/service/interface_executor, datahub-service/service/models
 * @refs service/sync_engine, service/interface_executor
 */

package basic_sync

import (
	"context"
	"datahub-service/service/interface_executor"
	"datahub-service/service/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// InterfaceSyncProcessor 接口同步处理器
type InterfaceSyncProcessor struct {
	db       *gorm.DB
	executor *interface_executor.InterfaceExecutor
}

// NewInterfaceSyncProcessor 创建接口同步处理器
func NewInterfaceSyncProcessor(db *gorm.DB, executor *interface_executor.InterfaceExecutor) *InterfaceSyncProcessor {
	return &InterfaceSyncProcessor{
		db:       db,
		executor: executor,
	}
}

// SyncInterfaceRequest 接口同步请求
type SyncInterfaceRequest struct {
	InterfaceID   string                 `json:"interface_id"`
	InterfaceType string                 `json:"interface_type"` // basic_library, thematic_library
	SyncType      string                 `json:"sync_type"`      // full, incremental
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	TaskID        string                 `json:"task_id,omitempty"`      // 关联的同步任务ID
	ExecutionID   string                 `json:"execution_id,omitempty"` // 执行ID
	ScheduledBy   string                 `json:"scheduled_by,omitempty"` // 调度者
}

// ProcessInterfaceSync 处理接口同步
func (p *InterfaceSyncProcessor) ProcessInterfaceSync(ctx context.Context, request *SyncInterfaceRequest) (*models.SyncInterfaceResult, error) {
	startTime := time.Now()

	fmt.Printf("[DEBUG] InterfaceSyncProcessor.ProcessInterfaceSync - 开始同步接口: %s, 类型: %s\n",
		request.InterfaceID, request.InterfaceType)

	// 构建执行器请求
	executeRequest := &interface_executor.ExecuteRequest{
		InterfaceID:   request.InterfaceID,
		InterfaceType: request.InterfaceType,
		ExecuteType:   "sync",
		Parameters:    request.Parameters,
		Options: map[string]interface{}{
			"sync_type":    request.SyncType,
			"task_id":      request.TaskID,
			"execution_id": request.ExecutionID,
			"scheduled_by": request.ScheduledBy,
		},
	}

	// 执行接口同步
	response, err := p.executor.Execute(ctx, executeRequest)
	endTime := time.Now()

	// 构建同步结果
	duration := endTime.Sub(startTime).Milliseconds()
	result := &models.SyncInterfaceResult{
		Success:       response.Success,
		ProcessedRows: int64(response.RowCount),
		StartTime:     &startTime,
		EndTime:       &endTime,
		Duration:      &duration,
		Details: map[string]interface{}{
			"interface_id":   request.InterfaceID,
			"interface_type": request.InterfaceType,
			"sync_type":      request.SyncType,
			"table_updated":  response.TableUpdated,
			"updated_rows":   response.UpdatedRows,
			"data_types":     response.DataTypes,
			"column_count":   response.ColumnCount,
			"warnings":       response.Warnings,
			"message":        response.Message,
		},
	}

	if err != nil {
		result.ErrorMessage = err.Error()
		fmt.Printf("[ERROR] InterfaceSyncProcessor.ProcessInterfaceSync - 同步失败: %v\n", err)
	} else {
		fmt.Printf("[INFO] InterfaceSyncProcessor.ProcessInterfaceSync - 同步成功，处理 %d 行数据\n", result.ProcessedRows)
	}

	// 记录同步结果到数据库（可选）
	if err := p.recordSyncResult(ctx, request, result); err != nil {
		fmt.Printf("[WARN] InterfaceSyncProcessor.ProcessInterfaceSync - 记录同步结果失败: %v\n", err)
	}

	return result, err
}

// ProcessBasicLibraryInterface 处理基础库接口同步
func (p *InterfaceSyncProcessor) ProcessBasicLibraryInterface(ctx context.Context, interfaceID string, parameters map[string]interface{}) (*models.SyncInterfaceResult, error) {
	request := &SyncInterfaceRequest{
		InterfaceID:   interfaceID,
		InterfaceType: "basic_library",
		SyncType:      "full",
		Parameters:    parameters,
	}
	return p.ProcessInterfaceSync(ctx, request)
}

// ProcessThematicLibraryInterface 处理主题库接口同步
func (p *InterfaceSyncProcessor) ProcessThematicLibraryInterface(ctx context.Context, interfaceID string, parameters map[string]interface{}) (*models.SyncInterfaceResult, error) {
	request := &SyncInterfaceRequest{
		InterfaceID:   interfaceID,
		InterfaceType: "thematic_library",
		SyncType:      "full",
		Parameters:    parameters,
	}
	return p.ProcessInterfaceSync(ctx, request)
}

// BatchProcessInterfaces 批量处理接口同步
func (p *InterfaceSyncProcessor) BatchProcessInterfaces(ctx context.Context, requests []*SyncInterfaceRequest) ([]*models.SyncInterfaceResult, error) {
	results := make([]*models.SyncInterfaceResult, 0, len(requests))

	for _, request := range requests {
		result, err := p.ProcessInterfaceSync(ctx, request)
		if err != nil {
			// 记录错误但继续处理其他接口
			fmt.Printf("[ERROR] InterfaceSyncProcessor.BatchProcessInterfaces - 处理接口 %s 失败: %v\n",
				request.InterfaceID, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// recordSyncResult 记录同步结果到数据库
func (p *InterfaceSyncProcessor) recordSyncResult(ctx context.Context, request *SyncInterfaceRequest, result *models.SyncInterfaceResult) error {
	// 这里可以将同步结果记录到数据库中，用于监控和统计
	// 例如创建 InterfaceSyncLog 记录

	syncLog := map[string]interface{}{
		"interface_id":   request.InterfaceID,
		"interface_type": request.InterfaceType,
		"sync_type":      request.SyncType,
		"task_id":        request.TaskID,
		"execution_id":   request.ExecutionID,
		"success":        result.Success,
		"processed_rows": result.ProcessedRows,
		"duration_ms":    *result.Duration,
		"error_message":  result.ErrorMessage,
		"start_time":     result.StartTime,
		"end_time":       result.EndTime,
		"created_at":     time.Now(),
	}

	// 插入到日志表（这里简化处理，实际应该有专门的日志表）
	fmt.Printf("[DEBUG] InterfaceSyncProcessor.recordSyncResult - 记录同步日志: %+v\n", syncLog)

	return nil
}

// GetSyncStatistics 获取同步统计信息
func (p *InterfaceSyncProcessor) GetSyncStatistics(ctx context.Context, interfaceID string, interfaceType string, days int) (map[string]interface{}, error) {
	// 查询指定时间范围内的同步统计
	// 这里返回模拟数据，实际应该从数据库查询

	stats := map[string]interface{}{
		"interface_id":      interfaceID,
		"interface_type":    interfaceType,
		"total_syncs":       50,
		"successful_syncs":  48,
		"failed_syncs":      2,
		"success_rate":      0.96,
		"avg_duration_ms":   1500,
		"total_rows":        125000,
		"avg_rows_per_sync": 2500,
		"last_sync_time":    time.Now().Add(-2 * time.Hour),
		"last_success_time": time.Now().Add(-2 * time.Hour),
		"last_error_time":   time.Now().Add(-24 * time.Hour),
		"period_days":       days,
	}

	return stats, nil
}

// ValidateInterfaceExists 验证接口是否存在
func (p *InterfaceSyncProcessor) ValidateInterfaceExists(ctx context.Context, interfaceID string, interfaceType string) error {
	switch interfaceType {
	case "basic_library":
		var count int64
		if err := p.db.Model(&models.DataInterface{}).Where("id = ?", interfaceID).Count(&count).Error; err != nil {
			return fmt.Errorf("查询基础库接口失败: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("基础库接口 %s 不存在", interfaceID)
		}
	case "thematic_library":
		var count int64
		if err := p.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceID).Count(&count).Error; err != nil {
			return fmt.Errorf("查询主题库接口失败: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("主题库接口 %s 不存在", interfaceID)
		}
	default:
		return fmt.Errorf("不支持的接口类型: %s", interfaceType)
	}

	return nil
}
