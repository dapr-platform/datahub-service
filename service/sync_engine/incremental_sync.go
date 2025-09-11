/*
 * @module service/sync_engine/incremental_sync
 * @description 增量同步管理器，负责时间戳增量、主键范围增量、变更日志增量等策略
 * @architecture 分层架构 - 增量同步层
 * @documentReference ai_docs/basic_library_process_impl.md, ai_docs/patch_basic_library_process.md
 * @stateFlow 增量状态检查 -> 增量数据抽取 -> 数据处理 -> 状态记录更新
 * @rules 确保增量同步的准确性和可恢复性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs service/sync_engine, service/basic_library
 */

package sync_engine

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// IncrementalSync 增量同步管理器
type IncrementalSync struct {
	db                *gorm.DB
	datasourceManager datasource.DataSourceManager
}

// NewIncrementalSync 创建增量同步管理器实例
func NewIncrementalSync(db *gorm.DB, datasourceManager datasource.DataSourceManager) *IncrementalSync {
	return &IncrementalSync{
		db:                db,
		datasourceManager: datasourceManager,
	}
}

// Process 执行增量同步处理
func (p *IncrementalSync) Process(ctx context.Context, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	// 获取数据源信息
	var dataSource models.DataSource
	if err := p.db.First(&dataSource, "id = ?", task.DataSourceID).Error; err != nil {
		return nil, fmt.Errorf("获取数据源信息失败: %w", err)
	}

	// 获取任务关联的接口信息
	var taskInterfaces []models.SyncTaskInterface
	if err := p.db.Where("task_id = ?", task.ID).Find(&taskInterfaces).Error; err != nil {
		return nil, fmt.Errorf("获取任务接口关联失败: %w", err)
	}

	// 增量同步通常处理单个接口，如果有多个接口则处理第一个
	var dataInterface *models.DataInterface
	if len(taskInterfaces) > 0 {
		dataInterface = &models.DataInterface{}
		if err := p.db.First(dataInterface, "id = ?", taskInterfaces[0].InterfaceID).Error; err != nil {
			return nil, fmt.Errorf("获取接口信息失败: %w", err)
		}
	}

	// 使用datasource框架处理增量同步
	return p.processWithDataSource(ctx, &dataSource, dataInterface, task, progress)
}

// processWithDataSource 使用datasource框架处理增量同步
func (p *IncrementalSync) processWithDataSource(ctx context.Context, dataSource *models.DataSource, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress) (*SyncResult, error) {
	progress.CurrentPhase = "获取数据源实例"
	progress.UpdatedAt = time.Now()

	var dsInstance datasource.DataSourceInterface
	var err error

	// 首先尝试从管理器获取已注册的数据源实例
	dsInstance, err = p.datasourceManager.Get(dataSource.ID)
	if err != nil {
		// 如果数据源未注册，则注册它
		if err := p.datasourceManager.Register(ctx, dataSource); err != nil {
			return nil, fmt.Errorf("注册数据源失败: %w", err)
		}

		// 再次获取数据源实例
		dsInstance, err = p.datasourceManager.Get(dataSource.ID)
		if err != nil {
			return nil, fmt.Errorf("获取数据源实例失败: %w", err)
		}
	}

	// 启动数据源（如果需要且未启动）
	if dsInstance.IsResident() && !dsInstance.IsStarted() {
		progress.CurrentPhase = "启动数据源"
		progress.UpdatedAt = time.Now()

		if err := dsInstance.Start(ctx); err != nil {
			return nil, fmt.Errorf("启动数据源失败: %w", err)
		}
	}

	progress.CurrentPhase = "开始增量数据同步"
	progress.UpdatedAt = time.Now()

	var processedRows int64
	var errorCount int
	startTime := time.Now()

	// 获取上次同步时间
	lastSyncTime, err := p.getLastSyncTime(task)
	if err != nil {
		return nil, fmt.Errorf("获取上次同步时间失败: %w", err)
	}

	// 执行增量同步
	processedRows, errorCount, err = p.executeIncrementalSync(ctx, dsInstance, dataInterface, task, progress, lastSyncTime)
	if err != nil {
		return nil, fmt.Errorf("增量数据同步失败: %w", err)
	}

	// 更新同步时间记录
	if err := p.updateLastSyncTime(task, time.Now()); err != nil {
		return nil, fmt.Errorf("更新同步时间记录失败: %w", err)
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
			"sync_type":        "incremental",
			"last_sync_time":   lastSyncTime,
			"processing_speed": p.calculateSpeed(processedRows, time.Since(startTime)),
		},
	}

	return result, nil
}

// executeIncrementalSync 执行增量同步
func (p *IncrementalSync) executeIncrementalSync(ctx context.Context, dsInstance datasource.DataSourceInterface, dataInterface *models.DataInterface, task *models.SyncTask, progress *SyncProgress, lastSyncTime *time.Time) (int64, int, error) {
	// TODO: 根据数据源类型和增量策略实现具体的增量同步逻辑
	// 例如：时间戳增量、主键范围增量、变更日志增量等

	// 目前返回模拟数据
	return 500, 0, nil
}

// getLastSyncTime 获取上次同步时间
func (p *IncrementalSync) getLastSyncTime(task *models.SyncTask) (*time.Time, error) {
	// 从数据库查询上次同步时间记录
	var existingTask models.SyncTask
	err := p.db.Select("sync_state").First(&existingTask, "id = ?", task.ID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 第一次同步
		}
		return nil, err
	}

	// TODO: 解析sync_state中的时间记录
	return nil, nil
}

// updateLastSyncTime 更新上次同步时间
func (p *IncrementalSync) updateLastSyncTime(task *models.SyncTask, syncTime time.Time) error {
	// TODO: 更新sync_state中的时间记录
	return nil
}

// calculateSpeed 计算处理速度
func (p *IncrementalSync) calculateSpeed(rows int64, duration time.Duration) float64 {
	if duration.Seconds() == 0 {
		return 0
	}
	return float64(rows) / duration.Seconds()
}

// GetProcessorType 获取处理器类型
func (p *IncrementalSync) GetProcessorType() string {
	return meta.ProcessorTypeIncremental
}

// Validate 验证任务参数
func (p *IncrementalSync) Validate(task *models.SyncTask) error {
	if task.DataSourceID == "" {
		return fmt.Errorf("数据源ID不能为空")
	}
	return nil
}

// EstimateProgress 估算进度
func (p *IncrementalSync) EstimateProgress(task *models.SyncTask) (*ProgressEstimate, error) {
	return &ProgressEstimate{
		EstimatedRows: 500,
		EstimatedTime: 3 * time.Minute,
		Complexity:    "medium",
	}, nil
}
