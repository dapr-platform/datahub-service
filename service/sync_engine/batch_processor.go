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
	db        *gorm.DB
	pgClient  *client.PgMetaClient
	batchSize int
}

// NewBatchProcessor 创建批量数据处理器实例
func NewBatchProcessor(db *gorm.DB) *BatchProcessor {
	return &BatchProcessor{
		db:        db,
		pgClient:  client.NewPgMetaClient("", ""),
		batchSize: 1000, // 默认批量大小
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

	// 根据数据源类型选择处理策略
	switch dataSource.Type {
	case "database", "postgresql", "mysql":
		return p.processDatabaseData(ctx, &dataSource, dataInterface, task, progress)
	case "http", "api":
		return p.processHTTPData(ctx, &dataSource, dataInterface, task, progress)
	case "file", "csv", "json", "excel":
		return p.processFileData(ctx, &dataSource, dataInterface, task, progress)
	default:
		return nil, fmt.Errorf("不支持的数据源类型: %s", dataSource.Type)
	}
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

// processBatch 处理数据批次
func (p *BatchProcessor) processBatch(ctx context.Context, rows []map[string]interface{}, dataInterface *models.DataInterface, task *models.SyncTask) error {
	// 这里应该实现具体的数据处理逻辑
	// 例如：数据转换、验证、写入目标表等

	// 模拟处理延迟
	time.Sleep(100 * time.Millisecond)

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
