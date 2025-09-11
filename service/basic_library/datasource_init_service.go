/*
 * @module service/basic_library/datasource_init_service
 * @description 数据源初始化服务，负责程序启动时从数据库加载数据源并注册到管理器
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/datasource_req.md
 * @stateFlow 服务启动 -> 加载数据源 -> 注册到管理器 -> 启动常驻数据源
 * @rules 确保数据源配置与管理器状态同步，支持批量操作和错误恢复
 * @dependencies datahub-service/service/models, datahub-service/service/datasource, gorm.io/gorm
 * @refs datasource_service.go, manager.go
 */

package basic_library

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/models"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// DatasourceInitService 数据源初始化服务
type DatasourceInitService struct {
	db                *gorm.DB
	datasourceManager datasource.DataSourceManager
	logger            *log.Logger
}

// NewDatasourceInitService 创建数据源初始化服务实例
func NewDatasourceInitService(db *gorm.DB) *DatasourceInitService {
	// 使用全局数据源注册中心
	registry := datasource.GetGlobalRegistry()
	datasourceManager := registry.GetManager()

	return &DatasourceInitService{
		db:                db,
		datasourceManager: datasourceManager,
		logger:            log.Default(),
	}
}

// InitializationResult 初始化结果
type InitializationResult struct {
	TotalCount    int      `json:"total_count"`
	SuccessCount  int      `json:"success_count"`
	FailedCount   int      `json:"failed_count"`
	SkippedCount  int      `json:"skipped_count"`
	FailedSources []string `json:"failed_sources,omitempty"`
	Duration      int64    `json:"duration"` // 毫秒
}

// InitializeAllDataSources 初始化所有数据源
func (s *DatasourceInitService) InitializeAllDataSources(ctx context.Context) (*InitializationResult, error) {
	startTime := time.Now()
	s.logger.Println("开始初始化数据源...")

	result := &InitializationResult{
		FailedSources: make([]string, 0),
	}

	// 从数据库加载所有激活状态的数据源
	dataSources, err := s.loadActiveDataSources()
	if err != nil {
		return nil, fmt.Errorf("加载数据源失败: %v", err)
	}

	result.TotalCount = len(dataSources)
	s.logger.Printf("找到 %d 个数据源需要初始化", result.TotalCount)

	if result.TotalCount == 0 {
		s.logger.Println("没有找到需要初始化的数据源")
		result.Duration = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// 批量注册数据源
	for _, ds := range dataSources {
		if err := s.registerDataSource(ctx, ds); err != nil {
			s.logger.Printf("注册数据源 %s (%s) 失败: %v", ds.ID, ds.Name, err)
			result.FailedCount++
			result.FailedSources = append(result.FailedSources, fmt.Sprintf("%s (%s): %v", ds.ID, ds.Name, err))
		} else {
			s.logger.Printf("数据源 %s (%s) 注册成功", ds.ID, ds.Name)
			result.SuccessCount++
		}
	}

	result.Duration = time.Since(startTime).Milliseconds()
	s.logger.Printf("数据源初始化完成: 总计=%d, 成功=%d, 失败=%d, 耗时=%dms",
		result.TotalCount, result.SuccessCount, result.FailedCount, result.Duration)

	return result, nil
}

// loadActiveDataSources 从数据库加载所有激活状态的数据源
func (s *DatasourceInitService) loadActiveDataSources() ([]*models.DataSource, error) {
	var dataSources []*models.DataSource

	// 查询所有激活状态的数据源
	err := s.db.Where("status = ?", "active").
		Preload("BasicLibrary").
		Find(&dataSources).Error

	if err != nil {
		return nil, fmt.Errorf("查询数据源失败: %v", err)
	}

	return dataSources, nil
}

// registerDataSource 注册单个数据源到管理器
func (s *DatasourceInitService) registerDataSource(ctx context.Context, ds *models.DataSource) error {
	// 检查数据源是否已经存在于管理器中
	if _, err := s.datasourceManager.Get(ds.ID); err == nil {
		s.logger.Printf("数据源 %s 已存在于管理器中，跳过注册", ds.ID)
		return nil
	}

	// 注册数据源到管理器
	err := s.datasourceManager.Register(ctx, ds)
	if err != nil {
		return fmt.Errorf("注册数据源到管理器失败: %v", err)
	}

	return nil
}

// ReloadDataSource 重新加载指定数据源
func (s *DatasourceInitService) ReloadDataSource(ctx context.Context, dataSourceID string) error {
	s.logger.Printf("重新加载数据源: %s", dataSourceID)

	// 从数据库加载数据源
	var ds models.DataSource
	err := s.db.Preload("BasicLibrary").First(&ds, "id = ?", dataSourceID).Error
	if err != nil {
		return fmt.Errorf("查询数据源失败: %v", err)
	}

	// 先从管理器移除（如果存在）
	if err := s.datasourceManager.Remove(dataSourceID); err != nil {
		s.logger.Printf("移除数据源 %s 时出现错误（可能不存在）: %v", dataSourceID, err)
	}

	// 重新注册
	if ds.Status == "active" {
		err = s.registerDataSource(ctx, &ds)
		if err != nil {
			return fmt.Errorf("重新注册数据源失败: %v", err)
		}
		s.logger.Printf("数据源 %s 重新加载成功", dataSourceID)
	} else {
		s.logger.Printf("数据源 %s 状态为非激活状态，跳过注册", dataSourceID)
	}

	return nil
}

// RemoveDataSourceFromManager 从管理器中移除数据源
func (s *DatasourceInitService) RemoveDataSourceFromManager(dataSourceID string) error {
	s.logger.Printf("从管理器移除数据源: %s", dataSourceID)

	err := s.datasourceManager.Remove(dataSourceID)
	if err != nil {
		return fmt.Errorf("从管理器移除数据源失败: %v", err)
	}

	s.logger.Printf("数据源 %s 已从管理器移除", dataSourceID)
	return nil
}

// GetManagerStatistics 获取管理器统计信息
func (s *DatasourceInitService) GetManagerStatistics() map[string]interface{} {
	return s.datasourceManager.GetStatistics()
}

// HealthCheckAllDataSources 对所有数据源进行健康检查
func (s *DatasourceInitService) HealthCheckAllDataSources(ctx context.Context) map[string]*datasource.HealthStatus {
	s.logger.Println("开始对所有数据源进行健康检查...")
	results := s.datasourceManager.HealthCheckAll(ctx)
	s.logger.Printf("健康检查完成，检查了 %d 个数据源", len(results))
	return results
}

// RestartResidentDataSource 重启常驻数据源
func (s *DatasourceInitService) RestartResidentDataSource(ctx context.Context, dataSourceID string) error {
	s.logger.Printf("重启常驻数据源: %s", dataSourceID)

	err := s.datasourceManager.RestartResidentDataSource(ctx, dataSourceID)
	if err != nil {
		return fmt.Errorf("重启常驻数据源失败: %v", err)
	}

	s.logger.Printf("常驻数据源 %s 重启成功", dataSourceID)
	return nil
}

// GetResidentDataSources 获取所有常驻数据源状态
func (s *DatasourceInitService) GetResidentDataSources() map[string]*datasource.DataSourceStatus {
	return s.datasourceManager.GetResidentDataSources()
}

// InitializeAndStartAllDataSources 初始化并启动所有数据源（合并操作，避免重复启动）
func (s *DatasourceInitService) InitializeAndStartAllDataSources(ctx context.Context) (*InitializationResult, error) {
	startTime := time.Now()
	s.logger.Println("开始初始化并启动数据源...")

	result := &InitializationResult{
		FailedSources: make([]string, 0),
	}

	// 从数据库加载所有激活状态的数据源
	dataSources, err := s.loadActiveDataSources()
	if err != nil {
		return nil, fmt.Errorf("加载数据源失败: %v", err)
	}

	result.TotalCount = len(dataSources)
	s.logger.Printf("找到 %d 个数据源需要初始化", result.TotalCount)

	if result.TotalCount == 0 {
		s.logger.Println("没有找到需要初始化的数据源")
		result.Duration = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// 批量注册数据源
	residentCount := 0
	for _, ds := range dataSources {
		if err := s.registerDataSource(ctx, ds); err != nil {
			s.logger.Printf("注册数据源 %s (%s) 失败: %v", ds.ID, ds.Name, err)
			result.FailedCount++
			result.FailedSources = append(result.FailedSources, fmt.Sprintf("%s (%s): %v", ds.ID, ds.Name, err))
		} else {
			s.logger.Printf("数据源 %s (%s) 注册成功", ds.ID, ds.Name)
			result.SuccessCount++
			// 统计常驻数据源数量（从管理器中获取实例来检查）
			if instance, err := s.datasourceManager.Get(ds.ID); err == nil && instance.IsResident() {
				residentCount++
			}
		}
	}

	// 如果有成功注册的常驻数据源，则启动它们
	if result.SuccessCount > 0 && residentCount > 0 {
		s.logger.Printf("开始启动 %d 个常驻数据源...", residentCount)
		if err := s.datasourceManager.StartAll(ctx); err != nil {
			s.logger.Printf("启动常驻数据源失败: %v", err)
			// 不返回错误，因为初始化已经完成，只是启动失败
		} else {
			s.logger.Println("常驻数据源启动完成")
		}
	}

	result.Duration = time.Since(startTime).Milliseconds()
	s.logger.Printf("数据源初始化和启动完成: 总计=%d, 成功=%d, 失败=%d, 常驻=%d, 耗时=%dms",
		result.TotalCount, result.SuccessCount, result.FailedCount, residentCount, result.Duration)

	return result, nil
}

// StartAllResidentDataSources 启动所有常驻数据源（保留此方法以兼容其他调用）
func (s *DatasourceInitService) StartAllResidentDataSources(ctx context.Context) error {
	s.logger.Println("启动所有常驻数据源...")

	err := s.datasourceManager.StartAll(ctx)
	if err != nil {
		return fmt.Errorf("启动常驻数据源失败: %v", err)
	}

	s.logger.Println("所有常驻数据源启动完成")
	return nil
}

// StopAllDataSources 停止所有数据源
func (s *DatasourceInitService) StopAllDataSources(ctx context.Context) error {
	s.logger.Println("停止所有数据源...")

	err := s.datasourceManager.StopAll(ctx)
	if err != nil {
		return fmt.Errorf("停止数据源失败: %v", err)
	}

	s.logger.Println("所有数据源已停止")
	return nil
}

// SyncDataSourceStatus 同步数据源状态到数据库
func (s *DatasourceInitService) SyncDataSourceStatus(ctx context.Context) error {
	s.logger.Println("同步数据源状态到数据库...")

	// 获取管理器中所有数据源的状态
	allStatus := s.datasourceManager.GetAllDataSourceStatus()

	for dsID, status := range allStatus {
		// 更新数据库中的状态记录
		err := s.updateDataSourceStatusInDB(dsID, status)
		if err != nil {
			s.logger.Printf("更新数据源 %s 状态失败: %v", dsID, err)
			continue
		}
	}

	s.logger.Printf("数据源状态同步完成，同步了 %d 个数据源", len(allStatus))
	return nil
}

// updateDataSourceStatusInDB 更新数据库中的数据源状态
func (s *DatasourceInitService) updateDataSourceStatusInDB(dataSourceID string, status *datasource.DataSourceStatus) error {
	now := time.Now()

	// 查找现有状态记录
	var statusRecord models.DataSourceStatus
	err := s.db.Where("data_source_id = ?", dataSourceID).First(&statusRecord).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		statusRecord = models.DataSourceStatus{
			DataSourceID:  dataSourceID,
			Status:        status.HealthStatus,
			LastTestTime:  &status.LastHealthCheck,
			LastErrorTime: nil,
			UpdatedAt:     now,
		}
		if status.HealthStatus == "error" && status.ErrorMessage != "" {
			statusRecord.LastErrorTime = &now
		}
		return s.db.Create(&statusRecord).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	updates := map[string]interface{}{
		"status":         status.HealthStatus,
		"last_test_time": &status.LastHealthCheck,
		"updated_at":     now,
	}

	if status.HealthStatus == "error" && status.ErrorMessage != "" {
		updates["last_error_time"] = &now
	}

	return s.db.Model(&statusRecord).Updates(updates).Error
}
