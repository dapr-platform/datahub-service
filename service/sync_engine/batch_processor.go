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

	// 获取数据源信息
	var dataSource models.DataSource
	if err := p.db.First(&dataSource, "id = ?", task.DataSourceID).Error; err != nil {
		return nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	// 获取接口信息（如果有）
	var dataInterface *models.DataInterface
	if task.InterfaceID != nil && *task.InterfaceID != "" {
		dataInterface = &models.DataInterface{}
		if err := p.db.First(dataInterface, "id = ?", *task.InterfaceID).Error; err != nil {
			return nil, fmt.Errorf("获取接口信息失败: %w", err)
		}
	}

	// 使用datasource框架处理数据
	return p.processWithDataSource(ctx, &dataSource, dataInterface, task, progress)
}

// processWithDataSource 使用datasource框架处理数据
func (p *BatchProcessor) processWithDataSource(ctx context.Context, dataSource *models.DataSource, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	progress.CurrentPhase = "初始化数据源"
	progress.UpdatedAt = time.Now()

	// 注册数据源到管理器
	err := p.datasourceManager.Register(ctx, dataSource)
	if err != nil {
		return nil, fmt.Errorf("注册数据源失败: %w", err)
	}

	// 获取数据源实例
	dsInstance, err := p.datasourceManager.Get(dataSource.ID)
	if err != nil {
		return nil, fmt.Errorf("获取数据源实例失败: %w", err)
	}

	// 启动数据源（如果需要）
	if dsInstance.IsResident() && !dsInstance.IsStarted() {
		progress.CurrentPhase = "启动数据源"
		progress.UpdatedAt = time.Now()

		if err := dsInstance.Start(ctx); err != nil {
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

// executeFullSync 执行全量同步
func (p *BatchProcessor) executeFullSync(ctx context.Context, dsInstance datasource.DataSourceInterface, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (int64, int, error) {
	var processedRows int64
	var errorCount int

	// 构建查询请求
	request := &datasource.ExecuteRequest{
		Operation: "query",
		Params: map[string]interface{}{
			"sync_type":  "full",
			"batch_size": p.batchSize,
		},
	}

	// 如果有接口配置，使用接口的查询参数
	if dataInterface != nil {
		// 从接口配置中获取查询参数
		if dataInterface.InterfaceConfig != nil {
			if queryParams, ok := dataInterface.InterfaceConfig["query_params"].(map[string]interface{}); ok {
				for key, value := range queryParams {
					request.Params[key] = value
				}
			}
			if query, ok := dataInterface.InterfaceConfig["query"].(string); ok && query != "" {
				request.Query = query
			}
		}
	}

	// 分批次执行查询
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return processedRows, errorCount, fmt.Errorf("任务被取消")
		default:
		}

		// 设置分页参数
		request.Params["offset"] = offset
		request.Params["limit"] = p.batchSize

		// 执行查询
		response, err := dsInstance.Execute(ctx, request)
		if err != nil {
			errorCount++
			if errorCount > 10 {
				return processedRows, errorCount, fmt.Errorf("连续错误次数过多: %w", err)
			}
			continue
		}

		if !response.Success {
			errorCount++
			continue
		}

		// 处理返回的数据
		if response.Data == nil {
			break // 没有更多数据
		}

		// 根据数据类型处理
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
		case map[string]interface{}:
			// 如果返回的是单个对象，包装成数组
			batchData = []map[string]interface{}{data}
		default:
			// 其他类型的数据，尝试转换
			batchData = []map[string]interface{}{
				{"data": data},
			}
		}

		if len(batchData) == 0 {
			break
		}

		// 处理数据批次
		if err := p.processBatch(ctx, batchData, dataInterface, task); err != nil {
			errorCount++
			continue
		}

		processedRows += int64(len(batchData))
		offset += len(batchData)

		// 更新进度
		progress.ProcessedRows = processedRows
		progress.ErrorCount = errorCount
		progress.Speed = p.calculateSpeed(processedRows, time.Since(time.Now().Add(-time.Minute)))
		progress.UpdatedAt = time.Now()

		// 如果返回的数据少于批次大小，说明已经到最后一批
		if len(batchData) < p.batchSize {
			break
		}
	}

	return processedRows, errorCount, nil
}

// executeIncrementalSync 执行增量同步
func (p *BatchProcessor) executeIncrementalSync(ctx context.Context, dsInstance datasource.DataSourceInterface, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (int64, int, error) {
	var processedRows int64
	var errorCount int

	// 获取上次同步时间
	lastSyncTime := p.getLastSyncTime(task)

	// 构建增量查询请求
	request := &datasource.ExecuteRequest{
		Operation: "query",
		Params: map[string]interface{}{
			"sync_type":      "incremental",
			"last_sync_time": lastSyncTime,
			"batch_size":     p.batchSize,
		},
	}

	// 如果有接口配置，使用接口的查询参数
	if dataInterface != nil {
		// 从接口配置中获取查询参数
		if dataInterface.InterfaceConfig != nil {
			if queryParams, ok := dataInterface.InterfaceConfig["query_params"].(map[string]interface{}); ok {
				for key, value := range queryParams {
					request.Params[key] = value
				}
			}
			if query, ok := dataInterface.InterfaceConfig["query"].(string); ok && query != "" {
				request.Query = query
			}
		}
	}

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
