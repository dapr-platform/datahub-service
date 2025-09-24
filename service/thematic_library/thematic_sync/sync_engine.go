/*
 * @module service/thematic_sync/sync_engine
 * @description 主题同步引擎，协调整个数据同步流程
 * @architecture 管道模式 - 通过多个处理阶段完成数据同步
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 任务接收 -> 数据获取 -> 汇聚处理 -> 清洗脱敏 -> 质量检查 -> 数据写入 -> 血缘记录
 * @rules 确保同步流程的完整性和一致性，支持事务性操作和错误恢复
 * @dependencies gorm.io/gorm, context, time
 * @refs models/thematic_sync.go, data_fetcher.go, data_processor.go
 */

package thematic_sync

import (
	"context"
	"datahub-service/service/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 类型定义已移动到 types.go 文件

// GovernanceIntegrationServiceInterface 数据治理集成服务接口
type GovernanceIntegrationServiceInterface interface {
	ApplyGovernanceRules(ctx context.Context, records []map[string]interface{}, task *models.ThematicSyncTask, config *GovernanceExecutionConfig) ([]map[string]interface{}, *GovernanceExecutionResult, error)
}

// DataFetcherInterface 数据获取接口
type DataFetcherInterface interface {
	FetchSourceData(request *SyncRequest, result *SyncExecutionResult) ([]SourceRecordInfo, error)
	FetchDataFromInterface(libraryID, interfaceID string) ([]map[string]interface{}, error)
}

// DataProcessorInterface 数据处理接口
type DataProcessorInterface interface {
	ProcessData(sourceRecords []SourceRecordInfo, request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, *GovernanceExecutionResult, error)
	MergeData(sourceRecords []SourceRecordInfo, request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, error)
}

// DataWriterInterface 数据写入接口
type DataWriterInterface interface {
	WriteData(processedRecords []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error
}

// LineageRecorderInterface 血缘记录接口
type LineageRecorderInterface interface {
	RecordLineage(sourceRecords []SourceRecordInfo, processedRecords []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult) error
}

// ThematicSyncEngine 主题同步引擎
type ThematicSyncEngine struct {
	db                    *gorm.DB
	governanceIntegration GovernanceIntegrationServiceInterface
	progressCallback      func(*SyncProgress)

	// 组件实例
	dataFetcher     DataFetcherInterface
	dataProcessor   DataProcessorInterface
	dataWriter      DataWriterInterface
	lineageRecorder LineageRecorderInterface
	utils           *SyncUtils
}

// NewThematicSyncEngine 创建主题同步引擎
func NewThematicSyncEngine(db *gorm.DB, governanceIntegration GovernanceIntegrationServiceInterface) *ThematicSyncEngine {
	// 创建各个组件实例
	dataFetcher := NewDataFetcher(db)
	dataProcessor := NewDataProcessor(db, governanceIntegration)
	dataWriter := NewDataWriter(db)
	lineageRecorder := NewLineageRecorder(db)
	utils := NewSyncUtils()

	return &ThematicSyncEngine{
		db:                    db,
		governanceIntegration: governanceIntegration,
		dataFetcher:           dataFetcher,
		dataProcessor:         dataProcessor,
		dataWriter:            dataWriter,
		lineageRecorder:       lineageRecorder,
		utils:                 utils,
	}
}

// SetProgressCallback 设置进度回调
func (tse *ThematicSyncEngine) SetProgressCallback(callback func(*SyncProgress)) {
	tse.progressCallback = callback
}

// ExecuteSync 执行同步
func (tse *ThematicSyncEngine) ExecuteSync(request *SyncRequest) (*SyncResponse, error) {
	startTime := time.Now()
	executionID := uuid.New().String()

	// 创建执行记录
	execution := &models.ThematicSyncExecution{
		ID:            executionID,
		TaskID:        request.TaskID,
		ExecutionType: request.ExecutionType,
		Status:        "running",
		StartTime:     &startTime,
		CreatedAt:     startTime, // 修复：添加CreatedAt字段
		CreatedBy:     "system",
	}

	if err := tse.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// 初始化进度
	progress := &SyncProgress{
		ExecutionID:    executionID,
		CurrentPhase:   PhaseInitialize,
		Progress:       0,
		StartTime:      startTime,
		LastUpdateTime: startTime,
	}

	// 执行同步流程
	result, err := tse.executeSyncPipeline(request, progress)

	// 更新执行记录
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = int64(endTime.Sub(startTime).Seconds())

	if err != nil {
		execution.Status = "failed"
		execution.ErrorDetails = models.JSONB{
			"error": err.Error(),
		}
	} else {
		execution.Status = "success"
		execution.SourceRecordCount = result.SourceRecordCount
		execution.ProcessedRecordCount = result.ProcessedRecordCount
		execution.InsertedRecordCount = result.InsertedRecordCount
		execution.UpdatedRecordCount = result.UpdatedRecordCount
		execution.ErrorRecordCount = result.ErrorRecordCount
	}

	tse.db.Save(execution)

	// 构建响应
	response := &SyncResponse{
		ExecutionID:    executionID,
		ProcessingTime: time.Since(startTime),
		Progress:       progress,
	}

	if err != nil {
		response.Status = "failed"
		response.Error = err.Error()
	} else {
		response.Status = "success"
		response.Result = result
	}

	return response, err
}

// executeSyncPipeline 执行同步管道
func (tse *ThematicSyncEngine) executeSyncPipeline(request *SyncRequest, progress *SyncProgress) (*SyncExecutionResult, error) {
	result := &SyncExecutionResult{
		ProcessingSteps: make([]ProcessingStepInfo, 0),
	}

	// 1. 初始化阶段
	if err := tse.executePhase(PhaseInitialize, progress, func() error {
		return tse.initializeSync(request, result)
	}); err != nil {
		return nil, err
	}

	// 2. 数据获取阶段
	var sourceRecords []SourceRecordInfo
	if err := tse.executePhase(PhaseDataFetch, progress, func() error {
		var err error
		sourceRecords, err = tse.dataFetcher.FetchSourceData(request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 3. 数据处理阶段（合并+治理）
	var processedRecords []map[string]interface{}
	var governanceResult *GovernanceExecutionResult
	if err := tse.executePhase(PhaseGovernance, progress, func() error {
		var err error
		processedRecords, governanceResult, err = tse.dataProcessor.ProcessData(sourceRecords, request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 4. 数据写入阶段
	if err := tse.executePhase(PhaseDataWrite, progress, func() error {
		return tse.dataWriter.WriteData(processedRecords, request, result, governanceResult)
	}); err != nil {
		return nil, err
	}

	// 5. 血缘记录阶段
	if err := tse.executePhase(PhaseLineage, progress, func() error {
		return tse.lineageRecorder.RecordLineage(sourceRecords, processedRecords, request, result)
	}); err != nil {
		return nil, err
	}

	// 6. 完成阶段
	if err := tse.executePhase(PhaseComplete, progress, func() error {
		return tse.completeSync(request, result, governanceResult)
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// executePhase 执行阶段
func (tse *ThematicSyncEngine) executePhase(phase SyncExecutionPhase, progress *SyncProgress, fn func() error) error {
	stepInfo := ProcessingStepInfo{
		Phase:     phase,
		StartTime: time.Now(),
		Status:    "running",
	}

	// 更新进度
	progress.CurrentPhase = phase
	progress.LastUpdateTime = time.Now()
	if tse.progressCallback != nil {
		tse.progressCallback(progress)
	}

	// 执行阶段逻辑
	err := fn()

	// 更新步骤信息
	stepInfo.EndTime = time.Now()
	stepInfo.Duration = stepInfo.EndTime.Sub(stepInfo.StartTime)

	if err != nil {
		stepInfo.Status = "failed"
		stepInfo.Message = err.Error()
	} else {
		stepInfo.Status = "success"
	}

	return err
}

// initializeSync 初始化同步
func (tse *ThematicSyncEngine) initializeSync(request *SyncRequest, result *SyncExecutionResult) error {
	// 验证请求参数
	if len(request.SourceInterfaces) == 0 && len(request.SourceLibraries) == 0 {
		return fmt.Errorf("源库或源接口列表不能为空")
	}

	if request.TargetInterfaceID == "" {
		return fmt.Errorf("目标接口ID不能为空")
	}

	// 验证目标接口是否存在
	var targetInterface models.ThematicInterface
	if err := tse.db.First(&targetInterface, "id = ?", request.TargetInterfaceID).Error; err != nil {
		return fmt.Errorf("获取目标接口信息失败: %w", err)
	}

	return nil
}

// completeSync 完成同步
func (tse *ThematicSyncEngine) completeSync(request *SyncRequest, result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error {
	// 更新任务统计信息
	var task models.ThematicSyncTask
	if err := tse.db.First(&task, "id = ?", request.TaskID).Error; err != nil {
		return fmt.Errorf("获取任务信息失败: %w", err)
	}

	now := time.Now()
	task.LastSyncTime = &now
	task.LastSyncStatus = "success"
	task.TotalSyncCount++
	task.SuccessfulSyncCount++
	task.UpdateNextRunTime()

	if err := tse.db.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务信息失败: %w", err)
	}

	return nil
}
