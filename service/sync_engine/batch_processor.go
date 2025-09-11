/*
 * @module service/sync_engine/batch_processor
 * @description 批量数据处理器，负责数据库、HTTP API、文件等批量数据抽取和处理
 * @architecture 分层架构 - 数据处理层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 数据抽取 -> 数据转换 -> 数据分片 -> 批量写入 -> 状态更新
 * @rules 确保批量数据处理的高效性和可靠性，支持大数据量处理
 * @dependencies datahub-service/service/models, datahub-service/client
 * @refs service/sync_engine, service/basic_library
 */

package sync_engine

import (
	"context"
	"datahub-service/client"
	"datahub-service/service/datasource"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BatchProcessor 批量数据处理器
type BatchProcessor struct {
	db                *gorm.DB
	pgClient          *client.PgMetaClient
	datasourceManager datasource.DataSourceManager
	batchSize         int
}

// NewBatchProcessor 创建批量数据处理器实例
func NewBatchProcessor(db *gorm.DB, datasourceManager datasource.DataSourceManager) *BatchProcessor {
	return &BatchProcessor{
		db:                db,
		pgClient:          client.NewPgMetaClient("", ""),
		datasourceManager: datasourceManager,
		batchSize:         1000, // 默认批量大小
	}
}

// Process 执行批量数据处理
func (p *BatchProcessor) Process(ctx context.Context, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	fmt.Printf("[DEBUG] BatchProcessor.Process - 开始处理任务: %s, 数据源ID: %s\n", task.ID, task.DataSourceID)

	// 获取数据源信息
	var dataSource models.DataSource
	if err := p.db.First(&dataSource, "id = ?", task.DataSourceID).Error; err != nil {
		fmt.Printf("[ERROR] BatchProcessor.Process - 获取数据源信息失败: %v\n", err)
		return nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	fmt.Printf("[DEBUG] BatchProcessor.Process - 数据源信息: ID=%s, 类型=%s, 名称=%s\n",
		dataSource.ID, dataSource.Type, dataSource.Name)

	// 加载任务接口关联信息
	if err := p.db.Preload("TaskInterfaces").
		Preload("TaskInterfaces.DataInterface").
		First(task, "id = ?", task.ID).Error; err != nil {
		fmt.Printf("[ERROR] BatchProcessor.Process - 加载任务接口信息失败: %v\n", err)
		return nil, fmt.Errorf("加载任务接口信息失败: %w", err)
	}

	if len(task.TaskInterfaces) == 0 {
		return nil, fmt.Errorf("任务没有关联的接口")
	}

	fmt.Printf("[DEBUG] BatchProcessor.Process - 任务关联接口数量: %d\n", len(task.TaskInterfaces))

	// 使用多接口处理数据
	return p.processMultipleInterfaces(ctx, &dataSource, task, progress)
}

// processWithDataSource 使用datasource框架处理数据
func (p *BatchProcessor) processWithDataSource(ctx context.Context, dataSource *models.DataSource, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	fmt.Printf("[DEBUG] BatchProcessor.processWithDataSource - 开始处理数据源: %s\n", dataSource.ID)
	progress.CurrentPhase = "获取数据源实例"
	progress.UpdatedAt = time.Now()

	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的数据源实例
	dsInstance, err = p.datasourceManager.Get(dataSource.ID)
	if err != nil {
		fmt.Printf("[DEBUG] BatchProcessor.processWithDataSource - 数据源未注册，注册新实例: %s\n", dataSource.ID)

		// 如果数据源未注册，则注册它
		if err := p.datasourceManager.Register(ctx, dataSource); err != nil {
			fmt.Printf("[ERROR] BatchProcessor.processWithDataSource - 注册数据源失败: %v\n", err)
			return nil, fmt.Errorf("注册数据源失败: %w", err)
		}

		// 再次获取数据源实例
		dsInstance, err = p.datasourceManager.Get(dataSource.ID)
		if err != nil {
			fmt.Printf("[ERROR] BatchProcessor.processWithDataSource - 获取注册后的数据源实例失败: %v\n", err)
			return nil, fmt.Errorf("获取数据源实例失败: %w", err)
		}

		fmt.Printf("[DEBUG] BatchProcessor.processWithDataSource - 数据源注册成功: %s\n", dataSource.ID)
	} else {
		fmt.Printf("[DEBUG] BatchProcessor.processWithDataSource - 使用已缓存的数据源实例: %s\n", dataSource.ID)
	}

	// 启动数据源（如果需要且未启动）
	if dsInstance.IsResident() && !dsInstance.IsStarted() {
		progress.CurrentPhase = "启动数据源"
		progress.UpdatedAt = time.Now()

		fmt.Printf("[DEBUG] BatchProcessor.processWithDataSource - 启动常驻数据源: %s\n", dataSource.ID)
		if err := dsInstance.Start(ctx); err != nil {
			fmt.Printf("[ERROR] BatchProcessor.processWithDataSource - 启动数据源失败: %v\n", err)
			return nil, fmt.Errorf("启动数据源失败: %w", err)
		}
	}

	progress.CurrentPhase = "开始数据同步"
	progress.UpdatedAt = time.Now()

	var processedRows int64
	var errorCount int
	startTime := time.Now()

	// 根据同步类型执行不同的操作
	switch task.TaskType {
	case string(SyncTypeFull):
		processedRows, errorCount, err = p.executeFullSync(ctx, dsInstance, dataInterface, task, progress)
	case string(SyncTypeIncremental):
		processedRows, errorCount, err = p.executeIncrementalSync(ctx, dsInstance, dataInterface, task, progress)
	default:
		return nil, fmt.Errorf("不支持的同步类型: %s", task.TaskType)
	}

	if err != nil {
		return nil, fmt.Errorf("数据同步失败: %w", err)
	}

	// 构建结果
	result := &SyncResult{
		TaskID:        task.ID,
		Status:        TaskStatusSuccess,
		ProcessedRows: processedRows,
		SuccessRows:   processedRows - int64(errorCount),
		ErrorRows:     int64(errorCount),
		StartTime:     startTime,
		EndTime:       time.Now(),
		Duration:      time.Since(startTime),
		Statistics: map[string]interface{}{
			"data_source_type": dataSource.Type,
			"data_source_id":   dataSource.ID,
			"sync_type":        task.TaskType,
			"processing_speed": p.calculateSpeed(processedRows, time.Since(startTime)),
		},
	}

	return result, nil
}

// processMultipleInterfaces 处理多个接口的同步
func (p *BatchProcessor) processMultipleInterfaces(ctx context.Context, dataSource *models.DataSource, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	fmt.Printf("[DEBUG] BatchProcessor.processMultipleInterfaces - 开始处理多接口同步\n")

	// 获取数据源实例
	dsInstance, err := p.datasourceManager.Get(dataSource.ID)
	if err != nil {
		fmt.Printf("[ERROR] BatchProcessor.processMultipleInterfaces - 获取数据源实例失败: %v\n", err)
		return nil, fmt.Errorf("获取数据源实例失败: %w", err)
	}

	// 初始化结果
	overallResult := &SyncResult{
		Success:       true,
		ProcessedRows: 0,
		ErrorCount:    0,
		Details:       make(map[string]interface{}),
		Interfaces:    make(map[string]*SyncInterfaceResult),
	}

	// 并发处理各个接口
	interfaceResults := make(chan *InterfaceProcessResult, len(task.TaskInterfaces))

	for i, taskInterface := range task.TaskInterfaces {
		go func(idx int, ti models.SyncTaskInterface) {
			fmt.Printf("[DEBUG] 开始处理接口 %d/%d: %s\n", idx+1, len(task.TaskInterfaces), ti.InterfaceID)

			// 更新接口状态为运行中
			p.updateTaskInterfaceStatus(ti.ID, "running", nil, "")

			// 处理单个接口
			result, err := p.processSingleInterface(ctx, dsInstance, &ti, task, progress)

			interfaceResults <- &InterfaceProcessResult{
				InterfaceID:     ti.InterfaceID,
				TaskInterfaceID: ti.ID,
				Result:          result,
				Error:           err,
			}
		}(i, taskInterface)
	}

	// 收集所有接口的处理结果
	for i := 0; i < len(task.TaskInterfaces); i++ {
		select {
		case result := <-interfaceResults:
			if result.Error != nil {
				fmt.Printf("[ERROR] 接口 %s 处理失败: %v\n", result.InterfaceID, result.Error)
				overallResult.Success = false
				overallResult.ErrorCount++

				// 更新接口状态为失败
				p.updateTaskInterfaceStatus(result.TaskInterfaceID, "failed", nil, result.Error.Error())

				// 记录错误详情
				overallResult.Interfaces[result.InterfaceID] = &SyncInterfaceResult{
					Success:       false,
					ProcessedRows: 0,
					ErrorMessage:  result.Error.Error(),
				}
			} else {
				fmt.Printf("[INFO] 接口 %s 处理成功，处理行数: %d\n", result.InterfaceID, result.Result.ProcessedRows)
				overallResult.ProcessedRows += result.Result.ProcessedRows

				// 更新接口状态为成功
				resultData := map[string]interface{}{
					"processed_rows": result.Result.ProcessedRows,
					"details":        result.Result.Details,
				}
				p.updateTaskInterfaceStatus(result.TaskInterfaceID, "success", resultData, "")

				// 记录成功详情
				overallResult.Interfaces[result.InterfaceID] = &SyncInterfaceResult{
					Success:       true,
					ProcessedRows: result.Result.ProcessedRows,
					Details:       result.Result.Details,
				}
			}
		case <-ctx.Done():
			fmt.Printf("[WARN] 上下文取消，停止等待接口处理结果\n")
			return nil, ctx.Err()
		}
	}

	// 设置整体结果详情
	overallResult.Details["total_interfaces"] = len(task.TaskInterfaces)
	overallResult.Details["success_interfaces"] = len(task.TaskInterfaces) - int(overallResult.ErrorCount)
	overallResult.Details["failed_interfaces"] = overallResult.ErrorCount

	fmt.Printf("[DEBUG] BatchProcessor.processMultipleInterfaces - 处理完成，总处理行数: %d, 错误数: %d\n",
		overallResult.ProcessedRows, overallResult.ErrorCount)

	return overallResult, nil
}

// processSingleInterface 处理单个接口的同步
func (p *BatchProcessor) processSingleInterface(ctx context.Context, dsInstance datasource.DataSourceInterface, taskInterface *models.SyncTaskInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	fmt.Printf("[DEBUG] BatchProcessor.processSingleInterface - 处理接口: %s\n", taskInterface.InterfaceID)

	// 这里调用原有的单接口处理逻辑
	return p.processWithDataSource(ctx, &models.DataSource{}, &taskInterface.DataInterface, task, progress)
}

// updateTaskInterfaceStatus 更新任务接口状态
func (p *BatchProcessor) updateTaskInterfaceStatus(taskInterfaceID, status string, result map[string]interface{}, errorMessage string) {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "running" {
		now := time.Now()
		updates["start_time"] = &now
	} else if status == "success" || status == "failed" {
		now := time.Now()
		updates["end_time"] = &now
	}

	if result != nil {
		updates["result"] = result
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if err := p.db.Model(&models.SyncTaskInterface{}).Where("id = ?", taskInterfaceID).Updates(updates).Error; err != nil {
		fmt.Printf("[ERROR] 更新任务接口状态失败: %v\n", err)
	}
}

// InterfaceProcessResult 接口处理结果
type InterfaceProcessResult struct {
	InterfaceID     string
	TaskInterfaceID string
	Result          *SyncResult
	Error           error
}

// 使用models包中定义的SyncInterfaceResult类型
type SyncInterfaceResult = models.SyncInterfaceResult

// executeFullSync 执行全量同步
func (p *BatchProcessor) executeFullSync(ctx context.Context, dsInstance datasource.DataSourceInterface, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (int64, int, error) {
	// 获取数据源信息
	var dataSource models.DataSource
	if err := p.db.First(&dataSource, "id = ?", task.DataSourceID).Error; err != nil {
		return 0, 1, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	// 创建查询构建器
	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, dataInterface)
	if err != nil {
		return 0, 1, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	// 构建同步请求参数
	syncParams := map[string]interface{}{
		"batch_size": p.batchSize,
	}

	// 构建全量同步请求
	request, err := queryBuilder.BuildSyncRequest("full", syncParams)
	if err != nil {
		return 0, 1, fmt.Errorf("构建全量同步请求失败: %w", err)
	}

	fmt.Printf("[DEBUG] BatchProcessor.executeFullSync - 同步请求构建完成: operation=%s, query=%s\n",
		request.Operation, request.Query)
	fmt.Printf("[DEBUG] BatchProcessor.executeFullSync - 请求参数: %+v\n", request.Params)
	fmt.Printf("[DEBUG] BatchProcessor.executeFullSync - 请求数据: %+v\n", request.Data)

	// 检查是否需要分页
	isPaginationEnabled := queryBuilder.IsPaginationEnabled()
	fmt.Printf("[DEBUG] BatchProcessor.executeFullSync - 分页支持: %t\n", isPaginationEnabled)

	if isPaginationEnabled {
		// 使用分页方式进行同步
		return p.executeFullSyncWithPagination(ctx, dsInstance, queryBuilder, syncParams, task, progress)
	} else {
		// 使用单次请求方式进行同步
		return p.executeFullSyncSingleRequest(ctx, dsInstance, request, task, progress)
	}
}

// executeFullSyncWithPagination 使用分页方式执行全量同步
func (p *BatchProcessor) executeFullSyncWithPagination(ctx context.Context, dsInstance datasource.DataSourceInterface, queryBuilder *datasource.QueryBuilder, syncParams map[string]interface{}, task *models.SyncTask, progress *SyncProgress) (int64, int, error) {
	var processedRows int64
	var errorCount int

	currentPage := 1
	hasMoreData := true

	for hasMoreData {
		select {
		case <-ctx.Done():
			fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncWithPagination - 任务被取消\n")
			return processedRows, errorCount, fmt.Errorf("任务被取消")
		default:
		}

		// 使用查询构建器构建当前页的参数
		pageParams := queryBuilder.BuildNextPageParams(currentPage, p.batchSize)

		// 使用查询构建器构建分页请求
		pageRequest, err := queryBuilder.BuildSyncRequestWithPagination("full", syncParams, pageParams)
		if err != nil {
			fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - 构建分页请求失败: %v\n", err)
			return processedRows, errorCount, fmt.Errorf("构建分页请求失败: %w", err)
		}

		fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncWithPagination - 开始执行查询, 页码=%d, 页大小=%d\n", currentPage, p.batchSize)
		fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncWithPagination - 当前请求参数: %+v\n", pageRequest.Params)

		// 执行查询
		fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncWithPagination - 调用数据源实例执行查询\n")
		response, err := dsInstance.Execute(ctx, pageRequest)
		if err != nil {
			fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - 查询执行失败: %v\n", err)
			errorCount++
			if errorCount > 10 {
				fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - 连续错误次数过多，终止执行\n")
				return processedRows, errorCount, fmt.Errorf("连续错误次数过多: %w", err)
			}
			// 发生错误时，跳出循环，避免死循环
			fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - 由于错误终止同步\n")
			break
		}

		fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncWithPagination - 查询执行完成, success=%t\n", response.Success)

		if !response.Success {
			fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - 响应失败: %s\n", response.Error)
			errorCount++
			// API返回失败时，也应该终止同步，避免死循环
			fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - API响应失败，终止同步\n")
			break
		}

		// 处理批次数据
		batchProcessedRows, batchHasMore, err := p.processBatchData(ctx, response, task, progress)
		if err != nil {
			fmt.Printf("[ERROR] BatchProcessor.executeFullSyncWithPagination - 处理批次数据失败: %v\n", err)
			errorCount++
			// 处理失败时继续下一页，但记录错误
		}

		processedRows += batchProcessedRows
		currentPage++

		fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncWithPagination - 批次处理完成, 已处理: %d 行, 当前页: %d\n", processedRows, currentPage)

		// 更新进度
		progress.ProcessedRows = processedRows
		progress.ErrorCount = errorCount
		progress.Speed = p.calculateSpeed(processedRows, time.Since(time.Now().Add(-time.Minute)))
		progress.UpdatedAt = time.Now()

		// 检查是否还有更多数据
		if !batchHasMore {
			hasMoreData = false
		}
	}

	return processedRows, errorCount, nil
}

// executeFullSyncSingleRequest 使用单次请求方式执行全量同步
func (p *BatchProcessor) executeFullSyncSingleRequest(ctx context.Context, dsInstance datasource.DataSourceInterface, request *datasource.ExecuteRequest, task *models.SyncTask, progress *SyncProgress) (int64, int, error) {
	var processedRows int64
	var errorCount int

	fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncSingleRequest - 开始执行单次请求同步\n")

	// 执行查询
	response, err := dsInstance.Execute(ctx, request)
	if err != nil {
		fmt.Printf("[ERROR] BatchProcessor.executeFullSyncSingleRequest - 查询执行失败: %v\n", err)
		return 0, 1, fmt.Errorf("查询执行失败: %w", err)
	}

	fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncSingleRequest - 查询执行完成, success=%t\n", response.Success)

	if !response.Success {
		fmt.Printf("[ERROR] BatchProcessor.executeFullSyncSingleRequest - 响应失败: %s\n", response.Error)
		return 0, 1, fmt.Errorf("响应失败: %s", response.Error)
	}

	// 处理批次数据
	batchProcessedRows, _, err := p.processBatchData(ctx, response, task, progress)
	if err != nil {
		fmt.Printf("[ERROR] BatchProcessor.executeFullSyncSingleRequest - 处理批次数据失败: %v\n", err)
		return 0, 1, fmt.Errorf("处理批次数据失败: %w", err)
	}

	processedRows += batchProcessedRows

	// 更新进度
	progress.ProcessedRows = processedRows
	progress.ErrorCount = errorCount
	progress.Speed = p.calculateSpeed(processedRows, time.Since(time.Now().Add(-time.Minute)))
	progress.UpdatedAt = time.Now()

	fmt.Printf("[DEBUG] BatchProcessor.executeFullSyncSingleRequest - 单次同步完成, 已处理: %d 行\n", processedRows)

	return processedRows, errorCount, nil
}

// processBatchData 处理批次数据的通用方法
func (p *BatchProcessor) processBatchData(ctx context.Context, response *datasource.ExecuteResponse, task *models.SyncTask, progress *SyncProgress) (int64, bool, error) {
	// 处理返回的数据
	if response.Data == nil {
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 响应数据为空，结束处理\n")
		return 0, false, nil // 没有更多数据
	}

	fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 响应数据类型: %T\n", response.Data)
	fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 响应元数据: %+v\n", response.Metadata)

	// 根据数据类型处理
	var batchData []map[string]interface{}
	switch data := response.Data.(type) {
	case []map[string]interface{}:
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 数据类型: []map[string]interface{}, 长度: %d\n", len(data))
		batchData = data
	case []interface{}:
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 数据类型: []interface{}, 长度: %d\n", len(data))
		batchData = make([]map[string]interface{}, len(data))
		for i, item := range data {
			if itemMap, ok := item.(map[string]interface{}); ok {
				batchData[i] = itemMap
			} else {
				fmt.Printf("[WARN] BatchProcessor.processBatchData - 数组元素 %d 不是 map[string]interface{} 类型: %T\n", i, item)
			}
		}
	case map[string]interface{}:
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 数据类型: map[string]interface{}, 包装为数组\n")
		// 如果返回的是单个对象，包装成数组
		batchData = []map[string]interface{}{data}
	default:
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 数据类型: %T, 转换为通用格式\n", data)
		// 其他类型的数据，尝试转换
		batchData = []map[string]interface{}{
			{"data": data},
		}
	}

	if len(batchData) == 0 {
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 批次数据为空，结束处理\n")
		return 0, false, nil
	}

	fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 准备处理批次数据，数量: %d\n", len(batchData))

	// 获取数据接口信息 - 向后兼容处理
	var dataInterface *models.DataInterface

	// 首先尝试从TaskInterfaces中获取第一个接口（新架构）
	if len(task.TaskInterfaces) > 0 {
		dataInterface = &task.TaskInterfaces[0].DataInterface
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 使用新架构获取接口: %s\n", dataInterface.ID)
	} else {
		fmt.Printf("[WARN] BatchProcessor.processBatchData - 任务没有关联的接口\n")
	}

	// 处理数据批次
	if err := p.processBatch(ctx, batchData, dataInterface, task); err != nil {
		return 0, false, fmt.Errorf("处理批次数据失败: %w", err)
	}

	processedRows := int64(len(batchData))

	// 判断是否还有更多数据
	hasMoreData := true

	// 检查返回数据量是否少于批次大小
	if len(batchData) < p.batchSize {
		fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 返回数据少于批次大小 (%d < %d)，可能没有更多数据\n", len(batchData), p.batchSize)
		hasMoreData = false
	}

	// 检查响应元数据中的分页信息
	if total, exists := response.Metadata["total"]; exists {
		if totalInt, ok := total.(int); ok {
			// 如果有总数信息，检查当前进度
			currentTotal := progress.ProcessedRows + processedRows
			if currentTotal >= int64(totalInt) {
				fmt.Printf("[DEBUG] BatchProcessor.processBatchData - 已处理完所有数据 (%d >= %d)\n", currentTotal, totalInt)
				hasMoreData = false
			}
		}
	}

	return processedRows, hasMoreData, nil
}

// executeIncrementalSync 执行增量同步
func (p *BatchProcessor) executeIncrementalSync(ctx context.Context, dsInstance datasource.DataSourceInterface, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (int64, int, error) {
	var processedRows int64
	var errorCount int

	// 获取数据源信息
	var dataSource models.DataSource
	if err := p.db.First(&dataSource, "id = ?", task.DataSourceID).Error; err != nil {
		return 0, 1, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	// 创建查询构建器
	queryBuilder, err := datasource.GetQueryBuilder(&dataSource, dataInterface)
	if err != nil {
		return 0, 1, fmt.Errorf("创建查询构建器失败: %w", err)
	}

	// 获取上次同步时间
	lastSyncTime := p.getLastSyncTime(task)

	// 构建增量同步请求参数
	syncParams := map[string]interface{}{
		"last_sync_time": lastSyncTime,
		"batch_size":     p.batchSize,
	}

	// 构建增量同步请求
	request, err := queryBuilder.BuildSyncRequest("incremental", syncParams)
	if err != nil {
		return 0, 1, fmt.Errorf("构建增量同步请求失败: %w", err)
	}

	fmt.Printf("[DEBUG] BatchProcessor.executeIncrementalSync - 增量同步请求构建完成: operation=%s, query=%s\n",
		request.Operation, request.Query)

	// 执行增量查询
	response, err := dsInstance.Execute(ctx, request)
	if err != nil {
		return 0, 1, fmt.Errorf("增量查询失败: %w", err)
	}

	if !response.Success {
		return 0, 1, fmt.Errorf("增量查询失败: %s", response.Error)
	}

	// 处理返回的数据
	if response.Data != nil {
		var batchData []map[string]interface{}
		switch data := response.Data.(type) {
		case []map[string]interface{}:
			batchData = data
		case []interface{}:
			batchData = make([]map[string]interface{}, len(data))
			for i, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					batchData[i] = itemMap
				}
			}
		}

		if len(batchData) > 0 {
			// 处理数据批次
			if err := p.processBatch(ctx, batchData, dataInterface, task); err != nil {
				errorCount++
			} else {
				processedRows = int64(len(batchData))
			}
		}
	}

	// 更新最后同步时间
	p.updateLastSyncTime(task, time.Now())

	return processedRows, errorCount, nil
}

// getLastSyncTime 获取上次同步时间
func (p *BatchProcessor) getLastSyncTime(task *models.SyncTask) time.Time {
	if lastSync, exists := task.Config["last_sync_time"]; exists {
		if lastSyncStr, ok := lastSync.(string); ok {
			if t, err := time.Parse(time.RFC3339, lastSyncStr); err == nil {
				return t
			}
		}
	}
	return time.Time{} // 返回零时间，表示首次同步
}

// updateLastSyncTime 更新最后同步时间
func (p *BatchProcessor) updateLastSyncTime(task *models.SyncTask, syncTime time.Time) {
	if task.Config == nil {
		task.Config = make(map[string]interface{})
	}
	task.Config["last_sync_time"] = syncTime.Format(time.RFC3339)

	// 更新数据库中的配置
	p.db.Model(task).Update("config", task.Config)
}

// processDatabaseData 处理数据库数据
func (p *BatchProcessor) processDatabaseData(ctx context.Context, dataSource *models.DataSource, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	progress.CurrentPhase = "连接数据库"
	progress.UpdatedAt = time.Now()

	// 从连接配置中获取数据库信息
	host, _ := dataSource.ConnectionConfig["host"].(string)
	port, _ := dataSource.ConnectionConfig["port"].(float64)
	database, _ := dataSource.ConnectionConfig["database"].(string)
	username, _ := dataSource.ConnectionConfig["username"].(string)
	password, _ := dataSource.ConnectionConfig["password"].(string)

	// 构建连接URL
	dbURL := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", username, password, host, int(port), database)

	// 初始化PgMeta客户端
	pgClient := client.NewPgMetaClient(dbURL, "")

	// 获取表名
	tableName := ""
	if dataInterface != nil {
		tableName = dataInterface.NameEn
	} else if table, exists := task.Config["table_name"]; exists {
		tableName = table.(string)
	}

	if tableName == "" {
		return nil, fmt.Errorf("未指定要同步的表名")
	}

	progress.CurrentPhase = "获取数据统计"
	progress.UpdatedAt = time.Now()

	// 获取总行数
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	totalRows, err := p.executeCountQuery(pgClient, countSQL)
	if err != nil {
		return nil, fmt.Errorf("获取数据总数失败: %w", err)
	}

	progress.TotalRows = totalRows
	progress.CurrentPhase = "开始数据抽取"
	progress.UpdatedAt = time.Now()

	// 分批读取数据
	var processedRows int64
	var errorCount int
	offset := 0

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("任务被取消")
		default:
		}

		// 构建分页查询SQL
		selectSQL := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, p.batchSize, offset)

		// 执行查询
		rows, err := p.executeSelectQuery(pgClient, selectSQL)
		if err != nil {
			errorCount++
			if errorCount > 10 { // 连续错误超过10次则失败
				return nil, fmt.Errorf("批量查询失败次数过多: %w", err)
			}
			continue
		}

		// 如果没有更多数据，结束处理
		if len(rows) == 0 {
			break
		}

		// 处理数据批次
		if err := p.processBatch(ctx, rows, dataInterface, task); err != nil {
			errorCount++
			continue
		}

		processedRows += int64(len(rows))
		offset += p.batchSize

		// 更新进度
		progress.ProcessedRows = processedRows
		progress.ErrorCount = errorCount
		if totalRows > 0 {
			progress.ProgressPercent = int((processedRows * 100) / totalRows)
		}
		progress.Speed = p.calculateSpeed(processedRows, time.Since(time.Now().Add(-time.Minute)))
		progress.UpdatedAt = time.Now()
	}

	// 构建结果
	result := &SyncResult{
		TaskID:        task.ID,
		Status:        TaskStatusSuccess,
		ProcessedRows: processedRows,
		SuccessRows:   processedRows - int64(errorCount),
		ErrorRows:     int64(errorCount),
		StartTime:     time.Now(),
		EndTime:       time.Now(),
		Duration:      time.Since(time.Now()),
		Statistics: map[string]interface{}{
			"total_batches":    (processedRows + int64(p.batchSize) - 1) / int64(p.batchSize),
			"batch_size":       p.batchSize,
			"source_table":     tableName,
			"processing_speed": progress.Speed,
		},
	}

	return result, nil
}

// processHTTPData 处理HTTP API数据
func (p *BatchProcessor) processHTTPData(ctx context.Context, dataSource *models.DataSource, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	progress.CurrentPhase = "初始化HTTP客户端"
	progress.UpdatedAt = time.Now()

	// 从连接配置中获取API信息
	url, _ := dataSource.ConnectionConfig["url"].(string)
	method, _ := dataSource.ConnectionConfig["method"].(string)
	if method == "" {
		method = "GET"
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	progress.CurrentPhase = "开始API数据拉取"
	progress.UpdatedAt = time.Now()

	var processedRows int64
	var errorCount int
	page := 1
	pageSize := p.batchSize

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("任务被取消")
		default:
		}

		// 构建分页请求URL
		requestURL := fmt.Sprintf("%s?page=%d&size=%d", url, page, pageSize)

		// 创建HTTP请求
		req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
		}

		// 添加认证头等
		if headers, exists := dataSource.ConnectionConfig["headers"].(map[string]interface{}); exists {
			for key, value := range headers {
				req.Header.Set(key, fmt.Sprintf("%v", value))
			}
		}

		// 执行请求
		resp, err := client.Do(req)
		if err != nil {
			errorCount++
			if errorCount > 5 {
				return nil, fmt.Errorf("HTTP请求失败次数过多: %w", err)
			}
			continue
		}

		// 解析响应
		var apiResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			resp.Body.Close()
			errorCount++
			continue
		}
		resp.Body.Close()

		// 提取数据
		data, exists := apiResponse["data"]
		if !exists {
			break
		}

		dataList, ok := data.([]interface{})
		if !ok || len(dataList) == 0 {
			break
		}

		// 处理数据批次
		if err := p.processAPIBatch(ctx, dataList, dataInterface, task); err != nil {
			errorCount++
			continue
		}

		processedRows += int64(len(dataList))
		page++

		// 更新进度
		progress.ProcessedRows = processedRows
		progress.ErrorCount = errorCount
		progress.Speed = p.calculateSpeed(processedRows, time.Since(time.Now().Add(-time.Minute)))
		progress.UpdatedAt = time.Now()

		// 如果返回的数据少于页大小，说明已经到最后一页
		if len(dataList) < pageSize {
			break
		}
	}

	// 构建结果
	result := &SyncResult{
		TaskID:        task.ID,
		Status:        TaskStatusSuccess,
		ProcessedRows: processedRows,
		SuccessRows:   processedRows - int64(errorCount),
		ErrorRows:     int64(errorCount),
		StartTime:     time.Now(),
		EndTime:       time.Now(),
		Duration:      time.Since(time.Now()),
		Statistics: map[string]interface{}{
			"total_pages":      page - 1,
			"page_size":        pageSize,
			"source_url":       url,
			"processing_speed": progress.Speed,
		},
	}

	return result, nil
}

// processFileData 处理文件数据
func (p *BatchProcessor) processFileData(ctx context.Context, dataSource *models.DataSource, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	progress.CurrentPhase = "读取文件信息"
	progress.UpdatedAt = time.Now()

	// 从连接配置中获取文件信息
	filePath, _ := dataSource.ConnectionConfig["path"].(string)
	fileFormat, _ := dataSource.ConnectionConfig["format"].(string)

	if filePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}

	progress.CurrentPhase = "开始文件数据处理"
	progress.UpdatedAt = time.Now()

	// 根据文件格式选择处理方式
	switch strings.ToLower(fileFormat) {
	case "csv":
		return p.processCSVFile(ctx, filePath, dataInterface, task, progress)
	case "json":
		return p.processJSONFile(ctx, filePath, dataInterface, task, progress)
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", fileFormat)
	}
}

// processBatch 处理数据批次，根据库类型选择不同的写入策略
func (p *BatchProcessor) processBatch(ctx context.Context, rows []map[string]interface{}, dataInterface *models.DataInterface, task *models.SyncTask) error {
	// 根据库类型选择处理策略
	switch task.LibraryType {
	case meta.LibraryTypeBasic:
		return p.processBasicLibraryBatch(ctx, rows, dataInterface, task)
	case meta.LibraryTypeThematic:
		return p.processThematicLibraryBatch(ctx, rows, dataInterface, task)
	default:
		// 向后兼容，对于未指定库类型的任务，使用基础库处理
		return p.processBasicLibraryBatch(ctx, rows, dataInterface, task)
	}
}

// processBasicLibraryBatch 处理基础库数据批次
func (p *BatchProcessor) processBasicLibraryBatch(ctx context.Context, rows []map[string]interface{}, dataInterface *models.DataInterface, task *models.SyncTask) error {
	// 基础库的数据写入逻辑
	// 通常写入到基础库相关的表中

	// 构造基础库数据记录
	var basicLibraries []models.BasicLibrary
	for _, row := range rows {
		basicLibrary := models.BasicLibrary{
			// 从row中映射其他字段
		}

		// 从原始数据映射字段
		if nameZh, exists := row["name_zh"]; exists {
			if nameStr, ok := nameZh.(string); ok {
				basicLibrary.NameZh = nameStr
			}
		}

		if nameEn, exists := row["name_en"]; exists {
			if nameStr, ok := nameEn.(string); ok {
				basicLibrary.NameEn = nameStr
			}
		}

		if description, exists := row["description"]; exists {
			if descStr, ok := description.(string); ok {
				basicLibrary.Description = descStr
			}
		}

		// 设置默认值
		if basicLibrary.NameZh == "" {
			basicLibrary.NameZh = "默认基础库名称"
		}
		if basicLibrary.NameEn == "" {
			basicLibrary.NameEn = fmt.Sprintf("basic_library_%d", time.Now().Unix())
		}

		basicLibraries = append(basicLibraries, basicLibrary)
	}

	// 批量插入基础库数据
	if len(basicLibraries) > 0 {
		if err := p.db.CreateInBatches(basicLibraries, p.batchSize).Error; err != nil {
			return fmt.Errorf("批量写入基础库数据失败: %w", err)
		}
	}

	return nil
}

// processThematicLibraryBatch 处理专题库数据批次
func (p *BatchProcessor) processThematicLibraryBatch(ctx context.Context, rows []map[string]interface{}, dataInterface *models.DataInterface, task *models.SyncTask) error {
	// 专题库的数据写入逻辑
	// 通常写入到专题库相关的表中

	// 构造专题库数据记录
	var thematicLibraries []models.ThematicLibrary
	for _, row := range rows {
		thematicLibrary := models.ThematicLibrary{
			// 从row中映射其他字段
		}

		// 从原始数据映射字段
		if nameZh, exists := row["name_zh"]; exists {
			if nameStr, ok := nameZh.(string); ok {
				thematicLibrary.NameZh = nameStr
			}
		}

		if nameEn, exists := row["name_en"]; exists {
			if nameStr, ok := nameEn.(string); ok {
				thematicLibrary.NameEn = nameStr
			}
		}

		if description, exists := row["description"]; exists {
			if descStr, ok := description.(string); ok {
				thematicLibrary.Description = descStr
			}
		}

		// 专题库特有字段
		if category, exists := row["category"]; exists {
			if catStr, ok := category.(string); ok {
				thematicLibrary.Category = catStr
			}
		}

		if domain, exists := row["domain"]; exists {
			if domainStr, ok := domain.(string); ok {
				thematicLibrary.Domain = domainStr
			}
		}

		// 设置默认值
		if thematicLibrary.NameZh == "" {
			thematicLibrary.NameZh = "默认专题库名称"
		}
		if thematicLibrary.NameEn == "" {
			thematicLibrary.NameEn = fmt.Sprintf("thematic_library_%d", time.Now().Unix())
		}
		if thematicLibrary.Category == "" {
			thematicLibrary.Category = "business"
		}
		if thematicLibrary.Domain == "" {
			thematicLibrary.Domain = "general"
		}

		thematicLibraries = append(thematicLibraries, thematicLibrary)
	}

	// 批量插入专题库数据
	if len(thematicLibraries) > 0 {
		if err := p.db.CreateInBatches(thematicLibraries, p.batchSize).Error; err != nil {
			return fmt.Errorf("批量写入专题库数据失败: %w", err)
		}
	}

	return nil
}

// processAPIBatch 处理API数据批次
func (p *BatchProcessor) processAPIBatch(ctx context.Context, data []interface{}, dataInterface *models.DataInterface, task *models.SyncTask) error {
	// 这里应该实现具体的API数据处理逻辑

	// 模拟处理延迟
	time.Sleep(100 * time.Millisecond)

	return nil
}

// processCSVFile 处理CSV文件
func (p *BatchProcessor) processCSVFile(ctx context.Context, filePath string, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	// TODO: 实现CSV文件处理逻辑
	return &SyncResult{
		TaskID:        task.ID,
		Status:        TaskStatusSuccess,
		ProcessedRows: 0,
		SuccessRows:   0,
		ErrorRows:     0,
		StartTime:     time.Now(),
		EndTime:       time.Now(),
		Duration:      0,
	}, nil
}

// processJSONFile 处理JSON文件
func (p *BatchProcessor) processJSONFile(ctx context.Context, filePath string, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	// TODO: 实现JSON文件处理逻辑
	return &SyncResult{
		TaskID:        task.ID,
		Status:        TaskStatusSuccess,
		ProcessedRows: 0,
		SuccessRows:   0,
		ErrorRows:     0,
		StartTime:     time.Now(),
		EndTime:       time.Now(),
		Duration:      0,
	}, nil
}

// executeCountQuery 执行计数查询
func (p *BatchProcessor) executeCountQuery(pgClient *client.PgMetaClient, sql string) (int64, error) {
	// TODO: 使用PgMetaClient执行查询
	// 这里返回模拟数据
	return 1000, nil
}

// executeSelectQuery 执行查询
func (p *BatchProcessor) executeSelectQuery(pgClient *client.PgMetaClient, sql string) ([]map[string]interface{}, error) {
	// TODO: 使用PgMetaClient执行查询
	// 这里返回模拟数据
	var results []map[string]interface{}
	for i := 0; i < p.batchSize; i++ {
		results = append(results, map[string]interface{}{
			"id":   i + 1,
			"name": fmt.Sprintf("user_%d", i+1),
		})
	}
	return results, nil
}

// calculateSpeed 计算处理速度
func (p *BatchProcessor) calculateSpeed(processedRows int64, duration time.Duration) int64 {
	if duration.Seconds() == 0 {
		return 0
	}
	return int64(float64(processedRows) / duration.Seconds())
}

// GetProcessorType 获取处理器类型
func (p *BatchProcessor) GetProcessorType() string {
	return meta.ProcessorTypeBatch
}

// Validate 验证任务参数
func (p *BatchProcessor) Validate(task *models.SyncTask) error {
	if task.DataSourceID == "" {
		return fmt.Errorf("数据源ID不能为空")
	}
	return nil
}

// EstimateProgress 估算进度
func (p *BatchProcessor) EstimateProgress(task *models.SyncTask) (*ProgressEstimate, error) {
	// TODO: 实现进度估算逻辑
	return &ProgressEstimate{
		EstimatedRows: 1000,
		EstimatedTime: 5 * time.Minute,
		Complexity:    "medium",
		RequiredResources: map[string]interface{}{
			"memory": "512MB",
			"cpu":    "1 core",
		},
	}, nil
}
