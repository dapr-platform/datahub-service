/*
 * @module service/basic_library/datasource_service
 * @description 数据源管理服务，负责数据源测试、验证和配置管理
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 数据源连接测试 -> 数据预览 -> 状态更新
 * @rules 确保数据源连接的可靠性和安全性
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/interfaces.md
 */

package basic_library

import (
	"context"
	"datahub-service/service/datasource"
	"datahub-service/service/models"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// DatasourceService 数据源服务
type DatasourceService struct {
	db                *gorm.DB
	validationService *ValidationService
	datasourceManager datasource.DataSourceManager
}

// NewDatasourceService 创建数据源服务实例
func NewDatasourceService(db *gorm.DB) *DatasourceService {
	// 使用全局数据源注册中心，确保内置类型已注册
	registry := datasource.GetGlobalRegistry()
	datasourceManager := registry.GetManager()

	return &DatasourceService{
		db:                db,
		validationService: NewValidationService(db),
		datasourceManager: datasourceManager,
	}
}

// DataSourceTestResult 数据源测试结果
type DataSourceTestResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Duration    int64                  `json:"duration"`
	TestType    string                 `json:"test_type"`
	Data        interface{}            `json:"data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// CreateDataSource 创建数据源
func (s *DatasourceService) CreateDataSource(dataSource *models.DataSource) error {
	// 检查基础库是否存在
	var library models.BasicLibrary
	if err := s.db.First(&library, "id = ?", dataSource.LibraryID).Error; err != nil {
		return errors.New("关联的数据基础库不存在")
	}

	// 验证数据源配置
	validationResult, err := s.validationService.ValidateDataSourceConfig(dataSource.Type, dataSource.ConnectionConfig, dataSource.ParamsConfig)
	if err != nil {
		return err
	}
	if !validationResult.IsValid {
		return fmt.Errorf("数据源配置验证失败: %v", validationResult.Errors)
	}

	// 保存到数据库
	if err := s.db.Create(dataSource).Error; err != nil {
		return err
	}

	// 如果数据源状态为激活，立即注册到管理器
	if dataSource.Status == "active" {
		ctx := context.Background()
		if err := s.datasourceManager.Register(ctx, dataSource); err != nil {
			log.Printf("警告：数据源 %s 创建成功但注册到管理器失败: %v", dataSource.ID, err)
		} else {
			log.Printf("数据源 %s 创建并注册到管理器成功", dataSource.ID)
		}
	}

	return nil
}

// UpdateDataSource 更新数据源
func (s *DatasourceService) UpdateDataSource(id string, updates map[string]interface{}) error {
	// 检查是否存在
	var dataSource models.DataSource
	if err := s.db.First(&dataSource, "id = ?", id).Error; err != nil {
		return err
	}

	// 记录原状态
	originalStatus := dataSource.Status

	// 如果更新配置，验证配置
	if connectionConfig, exists := updates["connection_config"]; exists {
		dataType := dataSource.Type
		if newType, exists3 := updates["type"]; exists3 {
			dataType = newType.(string)
		}

		var paramsConfig map[string]interface{}
		if paramsConfigUpdate, exists2 := updates["params_config"]; exists2 {
			paramsConfig = paramsConfigUpdate.(map[string]interface{})
		}

		validationResult, err := s.validationService.ValidateDataSourceConfig(
			dataType,
			connectionConfig.(map[string]interface{}),
			paramsConfig,
		)
		if err != nil {
			return err
		}
		if !validationResult.IsValid {
			return fmt.Errorf("数据源配置验证失败: %v", validationResult.Errors)
		}
	}

	// 更新数据库
	if err := s.db.Model(&dataSource).Updates(updates).Error; err != nil {
		return err
	}

	// 重新加载更新后的数据源
	if err := s.db.Preload("BasicLibrary").First(&dataSource, "id = ?", id).Error; err != nil {
		return err
	}

	// 处理管理器中的数据源
	ctx := context.Background()
	newStatus := dataSource.Status

	// 如果状态从激活变为非激活，从管理器移除
	if originalStatus == "active" && newStatus != "active" {
		if err := s.datasourceManager.Remove(id); err != nil {
			log.Printf("警告：从管理器移除数据源 %s 失败: %v", id, err)
		} else {
			log.Printf("数据源 %s 已从管理器移除", id)
		}
	} else if newStatus == "active" {
		// 如果状态为激活，重新注册到管理器（先移除再注册）
		if originalStatus == "active" {
			// 先移除现有的
			if err := s.datasourceManager.Remove(id); err != nil {
				log.Printf("警告：移除旧数据源实例 %s 失败: %v", id, err)
			}
		}

		// 注册新的实例
		if err := s.datasourceManager.Register(ctx, &dataSource); err != nil {
			log.Printf("警告：数据源 %s 更新成功但重新注册到管理器失败: %v", id, err)
		} else {
			log.Printf("数据源 %s 更新并重新注册到管理器成功", id)
		}
	}

	return nil
}

// DeleteDataSource 删除数据源
func (s *DatasourceService) DeleteDataSource(dataSource *models.DataSource) error {
	// 检查是否存在关联的接口
	var interfaceCount int64
	s.db.Model(&models.DataInterface{}).Where("data_source_id = ?", dataSource.ID).Count(&interfaceCount)

	if interfaceCount > 0 {
		return errors.New("无法删除：存在关联的数据接口")
	}

	// 先从管理器移除数据源
	if err := s.datasourceManager.Remove(dataSource.ID); err != nil {
		log.Printf("警告：从管理器移除数据源 %s 失败: %v", dataSource.ID, err)
	} else {
		log.Printf("数据源 %s 已从管理器移除", dataSource.ID)
	}

	// 删除相关的状态记录
	s.db.Where("data_source_id = ?", dataSource.ID).Delete(&models.DataSourceStatus{})

	// 删除数据源记录
	if err := s.db.Delete(&models.DataSource{}, "id = ?", dataSource.ID).Error; err != nil {
		return err
	}

	log.Printf("数据源 %s 删除成功", dataSource.ID)
	return nil
}

// TestDataSource 测试数据源连接
func (s *DatasourceService) TestDataSource(dataSourceID, testType string, config map[string]interface{}) (*DataSourceTestResult, error) {
	startTime := time.Now()

	// 获取数据源信息
	var dataSource models.DataSource
	if err := s.db.First(&dataSource, "id = ?", dataSourceID).Error; err != nil {
		return &DataSourceTestResult{
			Success:  false,
			Message:  "数据源不存在",
			Duration: time.Since(startTime).Milliseconds(),
			TestType: testType,
			Error:    err.Error(),
		}, err
	}

	// 根据数据源类型和测试类型进行测试
	switch testType {
	case "connection":
		return s.testConnection(&dataSource, startTime)
	case "data_preview":
		return s.testDataPreview(&dataSource, config, startTime)
	default:
		return &DataSourceTestResult{
			Success:  false,
			Message:  "不支持的测试类型",
			Duration: time.Since(startTime).Milliseconds(),
			TestType: testType,
			Error:    "unsupported test type",
		}, fmt.Errorf("不支持的测试类型: %s", testType)
	}
}

// testConnection 测试数据源连接
func (s *DatasourceService) testConnection(dataSource *models.DataSource, startTime time.Time) (*DataSourceTestResult, error) {
	result := &DataSourceTestResult{
		TestType: "connection",
	}

	// 使用datasource框架测试连接
	ctx := context.Background()

	// 创建测试数据源实例（非常驻模式），不注册到管理器中
	instance, err := s.datasourceManager.CreateTestInstance(dataSource.Type)
	if err != nil {
		result.Success = false
		result.Message = "创建数据源实例失败"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = err.Error()
		return result, err
	}

	// 初始化数据源
	err = instance.Init(ctx, dataSource)
	if err != nil {
		result.Success = false
		result.Message = "初始化数据源失败"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = err.Error()
		return result, err
	}

	// 执行健康检查
	healthStatus, err := instance.HealthCheck(ctx)
	if err != nil {
		result.Success = false
		result.Message = "数据源连接测试失败"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = err.Error()
		return result, err
	}

	// 构建成功结果
	isHealthy := healthStatus.Status == "online"
	result.Success = isHealthy
	result.Message = healthStatus.Message
	result.Duration = time.Since(startTime).Milliseconds()
	result.Metadata = map[string]interface{}{
		"data_source_type": dataSource.Type,
		"health_status":    healthStatus.Status,
		"last_check_time":  healthStatus.LastCheck,
	}

	if isHealthy {
		result.Suggestions = []string{
			"数据源连接正常",
			"建议定期检查数据源状态",
		}
		// 更新数据源状态
		s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	} else {
		result.Suggestions = []string{
			"检查数据源配置",
			"验证网络连接",
			"确认认证信息正确",
		}
		now := time.Now()
		s.updateDataSourceStatus(dataSource.ID, "error", nil, &now)
	}

	return result, nil
}

// testDataPreview 测试数据预览
func (s *DatasourceService) testDataPreview(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	result := &DataSourceTestResult{
		TestType: "data_preview",
	}

	// 使用datasource框架获取预览数据
	ctx := context.Background()

	// 确保数据源已注册
	err := s.datasourceManager.Register(ctx, dataSource)
	if err != nil {
		result.Success = false
		result.Message = "注册数据源失败"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = err.Error()
		return result, err
	}

	// 获取数据源实例
	dsInstance, err := s.datasourceManager.Get(dataSource.ID)
	if err != nil {
		result.Success = false
		result.Message = "获取数据源实例失败"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = err.Error()
		return result, err
	}

	// 构建预览请求
	executeRequest := &datasource.ExecuteRequest{
		Operation: "query",
		Query:     "SELECT * FROM sample_table LIMIT 10", // 示例查询，实际应从config获取
		Params: map[string]interface{}{
			"limit": 10,
		},
	}

	// 如果config中有特定参数，使用它们
	if config != nil {
		if query, exists := config["query"]; exists {
			executeRequest.Query = query.(string)
		}
		if limit, exists := config["limit"]; exists {
			executeRequest.Params["limit"] = limit
		}
		if tableName, exists := config["table_name"]; exists {
			executeRequest.Params["table_name"] = tableName
		}
	}

	// 执行数据预览
	executeResponse, err := dsInstance.Execute(ctx, executeRequest)
	if err != nil {
		result.Success = false
		result.Message = "数据预览失败"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = err.Error()
		return result, err
	}

	// 构建成功结果
	result.Success = executeResponse.Success
	result.Message = "数据预览获取成功"
	result.Duration = time.Since(startTime).Milliseconds()
	result.Data = executeResponse.Data
	result.Metadata = map[string]interface{}{
		"row_count":        executeResponse.RowCount,
		"data_source_type": dataSource.Type,
		"execution_time":   executeResponse.Duration,
	}

	// 如果有元数据，包含更多信息
	if executeResponse.Metadata != nil {
		for k, v := range executeResponse.Metadata {
			result.Metadata[k] = v
		}
	}

	if executeResponse.Message != "" {
		result.Message = executeResponse.Message
	}

	return result, nil
}

// updateDataSourceStatus 更新数据源状态
func (s *DatasourceService) updateDataSourceStatus(dataSourceID, status string, lastTestTime, lastErrorTime *time.Time) error {
	now := time.Now()

	// 查找现有状态记录
	var statusRecord models.DataSourceStatus
	err := s.db.Where("data_source_id = ?", dataSourceID).First(&statusRecord).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		statusRecord = models.DataSourceStatus{
			DataSourceID: dataSourceID,
			Status:       status,
			LastTestTime: &now,
			UpdatedAt:    now,
		}
		if lastErrorTime != nil {
			statusRecord.LastErrorTime = lastErrorTime
		}
		return s.db.Create(&statusRecord).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	updates := map[string]interface{}{
		"status":         status,
		"last_test_time": &now,
		"updated_at":     now,
	}

	if lastErrorTime != nil {
		updates["last_error_time"] = lastErrorTime
	}

	return s.db.Model(&statusRecord).Updates(updates).Error
}

// ValidateDataSourceConfig 验证数据源配置（委托给统一验证服务）
func (s *DatasourceService) ValidateDataSourceConfig(dataSourceType string, connectionConfig, paramsConfig map[string]interface{}) error {
	validationResult, err := s.validationService.ValidateDataSourceConfig(dataSourceType, connectionConfig, paramsConfig)
	if err != nil {
		return err
	}
	if !validationResult.IsValid {
		return fmt.Errorf("数据源配置验证失败: %v", validationResult.Errors)
	}
	return nil
}
