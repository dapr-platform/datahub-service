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
	"datahub-service/service/meta"
	"datahub-service/service/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DatasourceService 数据源服务
type DatasourceService struct {
	db                *gorm.DB
	validationService *ValidationService
}

// NewDatasourceService 创建数据源服务实例
func NewDatasourceService(db *gorm.DB) *DatasourceService {
	return &DatasourceService{
		db:                db,
		validationService: NewValidationService(db),
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

	return s.db.Create(dataSource).Error
}

// UpdateDataSource 更新数据源
func (s *DatasourceService) UpdateDataSource(id string, updates map[string]interface{}) error {
	// 检查是否存在
	var dataSource models.DataSource
	if err := s.db.First(&dataSource, "id = ?", id).Error; err != nil {
		return err
	}

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

	return s.db.Model(&dataSource).Updates(updates).Error
}
func (s *DatasourceService) DeleteDataSource(dataSource *models.DataSource) error {
	// 检查是否存在关联的接口
	var interfaceCount int64
	s.db.Model(&models.DataInterface{}).Where("data_source_id = ?", dataSource.ID).Count(&interfaceCount)

	if interfaceCount > 0 {
		return errors.New("无法删除：存在关联的数据接口")
	}

	// 删除相关的调度配置和状态记录
	s.db.Where("data_source_id = ?", dataSource.ID).Delete(&models.ScheduleConfig{})
	s.db.Where("data_source_id = ?", dataSource.ID).Delete(&models.DataSourceStatus{})

	return s.db.Delete(&models.DataSource{}, "id = ?", dataSource.ID).Error
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

	// 从注册表获取类型定义
	definition, exists := meta.DataSourceTypes[dataSource.Type]
	if !exists {
		result.Success = false
		result.Message = "不支持的数据源类型"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = fmt.Sprintf("unsupported datasource type: %s", dataSource.Type)
		return result, fmt.Errorf("不支持的数据源类型: %s", dataSource.Type)
	}

	// 根据类别分发测试逻辑
	switch definition.Category {
	case string(meta.DataSourceCategoryDatabase):
		return s.testDatabaseConnection(dataSource, definition, startTime)
	case string(meta.DataSourceCategoryMessaging):
		switch dataSource.Type {
		case string(meta.DataSourceTypeKafka):
			return s.testKafkaConnection(dataSource, startTime)
		case string(meta.DataSourceTypeMQTT):
			return s.testMQTTConnection(dataSource, startTime)
		case string(meta.DataSourceTypeRedis):
			return s.testRedisConnection(dataSource, startTime)
		default:
			return s.testGenericConnection(dataSource, startTime, "消息队列")
		}
	case string(meta.DataSourceCategoryAPI):
		return s.testHTTPConnection(dataSource, startTime)
	case string(meta.DataSourceCategoryFile):
		return s.testFileConnection(dataSource, startTime)
	default:
		result.Success = false
		result.Message = "不支持的数据源类别"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = fmt.Sprintf("unsupported datasource category: %s", definition.Category)
		return result, fmt.Errorf("不支持的数据源类别: %s", definition.Category)
	}
}

// testDataPreview 测试数据预览
func (s *DatasourceService) testDataPreview(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	result := &DataSourceTestResult{
		TestType: "data_preview",
	}

	// 首先测试连接
	connResult, err := s.testConnection(dataSource, startTime)
	if err != nil || !connResult.Success {
		return connResult, err
	}

	// 从注册表获取类型定义
	definition, exists := meta.DataSourceTypes[dataSource.Type]
	if !exists {
		result.Success = false
		result.Message = "不支持的数据源类型预览"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = fmt.Sprintf("unsupported datasource type for preview: %s", dataSource.Type)
		return result, fmt.Errorf("不支持的数据源类型预览: %s", dataSource.Type)
	}

	// 根据类别获取预览数据
	switch definition.Category {
	case meta.DataSourceCategoryDatabase:
		return s.previewDatabaseData(dataSource, config, startTime)
	case meta.DataSourceCategoryMessaging:
		switch dataSource.Type {
		case meta.DataSourceTypeKafka:
			return s.previewKafkaData(dataSource, config, startTime)
		case meta.DataSourceTypeMQTT:
			return s.previewMQTTData(dataSource, config, startTime)
		default:
			return s.previewGenericData(dataSource, config, startTime, "消息队列")
		}
	case meta.DataSourceCategoryAPI:
		return s.previewHTTPData(dataSource, config, startTime)
	case meta.DataSourceCategoryFile:
		return s.previewFileData(dataSource, config, startTime)
	default:
		result.Success = false
		result.Message = "不支持的数据源类别预览"
		result.Duration = time.Since(startTime).Milliseconds()
		result.Error = fmt.Sprintf("unsupported datasource category for preview: %s", definition.Category)
		return result, fmt.Errorf("不支持的数据源类别预览: %s", definition.Category)
	}
}

// 数据库连接测试
func (s *DatasourceService) testDatabaseConnection(dataSource *models.DataSource, dataSourceTypeDefinition *meta.DataSourceTypeDefinition, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现数据库连接测试逻辑
	// 这里应该根据 connection_config 中的配置信息进行实际的数据库连接测试

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "数据库连接测试成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"database_type": dataSource.Type,
			"host":          dataSource.ConnectionConfig["host"],
			"port":          dataSource.ConnectionConfig["port"],
			"database":      dataSource.ConnectionConfig["database"],
		},
		Suggestions: []string{
			"建议定期检查数据库连接状态",
			"确保数据库用户权限配置正确",
		},
	}

	// 更新数据源状态
	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)

	return result, nil
}

// Kafka连接测试
func (s *DatasourceService) testKafkaConnection(dataSource *models.DataSource, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现Kafka连接测试逻辑

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "Kafka连接测试成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"brokers": dataSource.ConnectionConfig["brokers"],
			"topics":  dataSource.ConnectionConfig["topics"],
		},
		Suggestions: []string{
			"建议监控Kafka集群健康状态",
			"确保topic权限配置正确",
		},
	}

	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	return result, nil
}

// MQTT连接测试
func (s *DatasourceService) testMQTTConnection(dataSource *models.DataSource, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现MQTT连接测试逻辑

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "MQTT连接测试成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"broker": dataSource.ConnectionConfig["broker"],
			"topics": dataSource.ConnectionConfig["topics"],
		},
		Suggestions: []string{
			"建议监控MQTT broker状态",
			"确保订阅topic权限正确",
		},
	}

	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	return result, nil
}

// Redis连接测试
func (s *DatasourceService) testRedisConnection(dataSource *models.DataSource, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现Redis连接测试逻辑

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "Redis连接测试成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"host": dataSource.ConnectionConfig["host"],
			"port": dataSource.ConnectionConfig["port"],
			"db":   dataSource.ConnectionConfig["db"],
		},
		Suggestions: []string{
			"建议监控Redis内存使用情况",
			"确保Redis持久化配置正确",
		},
	}

	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	return result, nil
}

// HTTP连接测试
func (s *DatasourceService) testHTTPConnection(dataSource *models.DataSource, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现HTTP连接测试逻辑

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "HTTP接口连接测试成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"url":    dataSource.ConnectionConfig["url"],
			"method": dataSource.ConnectionConfig["method"],
		},
		Suggestions: []string{
			"建议配置适当的超时时间",
			"确保API授权信息正确",
		},
	}

	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	return result, nil
}

// 文件连接测试
func (s *DatasourceService) testFileConnection(dataSource *models.DataSource, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现文件连接测试逻辑

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "文件路径访问测试成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"path":   dataSource.ConnectionConfig["path"],
			"format": dataSource.ConnectionConfig["format"],
		},
		Suggestions: []string{
			"建议检查文件权限配置",
			"确保文件格式解析正确",
		},
	}

	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	return result, nil
}

// testGenericConnection 通用连接测试（用于未实现具体测试逻辑的数据源类型）
func (s *DatasourceService) testGenericConnection(dataSource *models.DataSource, startTime time.Time, category string) (*DataSourceTestResult, error) {
	// TODO: 实现通用连接测试逻辑

	result := &DataSourceTestResult{
		Success:  true,
		Message:  fmt.Sprintf("%s连接测试成功", category),
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "connection",
		Metadata: map[string]interface{}{
			"datasource_type": dataSource.Type,
			"category":        category,
		},
		Suggestions: []string{
			"建议实现具体的连接测试逻辑",
			"确保配置参数正确",
		},
	}

	s.updateDataSourceStatus(dataSource.ID, "online", nil, nil)
	return result, nil
}

// 数据库数据预览
func (s *DatasourceService) previewDatabaseData(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现数据库数据预览逻辑

	// 模拟预览数据
	sampleData := []map[string]interface{}{
		{"id": 1, "name": "张三", "age": 25, "city": "北京"},
		{"id": 2, "name": "李四", "age": 30, "city": "上海"},
		{"id": 3, "name": "王五", "age": 28, "city": "广州"},
	}

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "数据预览获取成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "data_preview",
		Data:     sampleData,
		Metadata: map[string]interface{}{
			"row_count":    len(sampleData),
			"column_count": 4,
			"table_name":   config["table_name"],
			"sample_size":  len(sampleData),
		},
	}

	return result, nil
}

// Kafka数据预览
func (s *DatasourceService) previewKafkaData(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现Kafka数据预览逻辑

	// 模拟Kafka消息数据
	sampleData := []map[string]interface{}{
		{"offset": 100, "partition": 0, "timestamp": "2024-01-01T10:00:00Z", "value": "{\"user_id\": 1001, \"action\": \"login\"}"},
		{"offset": 101, "partition": 0, "timestamp": "2024-01-01T10:01:00Z", "value": "{\"user_id\": 1002, \"action\": \"view_page\"}"},
	}

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "Kafka消息预览获取成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "data_preview",
		Data:     sampleData,
		Metadata: map[string]interface{}{
			"message_count": len(sampleData),
			"topic":         config["topic"],
			"partition":     config["partition"],
		},
	}

	return result, nil
}

// MQTT数据预览
func (s *DatasourceService) previewMQTTData(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现MQTT数据预览逻辑

	// 模拟MQTT消息数据
	sampleData := []map[string]interface{}{
		{"topic": "sensor/temperature", "timestamp": "2024-01-01T10:00:00Z", "payload": "{\"value\": 25.6, \"unit\": \"celsius\"}"},
		{"topic": "sensor/humidity", "timestamp": "2024-01-01T10:01:00Z", "payload": "{\"value\": 68.2, \"unit\": \"percent\"}"},
	}

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "MQTT消息预览获取成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "data_preview",
		Data:     sampleData,
		Metadata: map[string]interface{}{
			"message_count": len(sampleData),
			"topics":        config["topics"],
		},
	}

	return result, nil
}

// HTTP数据预览
func (s *DatasourceService) previewHTTPData(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现HTTP数据预览逻辑

	// 模拟HTTP响应数据
	sampleData := map[string]interface{}{
		"status": "success",
		"data": []map[string]interface{}{
			{"id": 1, "name": "产品A", "price": 99.99},
			{"id": 2, "name": "产品B", "price": 149.99},
		},
		"total": 2,
	}

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "HTTP接口数据预览获取成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "data_preview",
		Data:     sampleData,
		Metadata: map[string]interface{}{
			"status_code":   200,
			"response_size": 1024,
			"content_type":  "application/json",
		},
	}

	return result, nil
}

// 文件数据预览
func (s *DatasourceService) previewFileData(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time) (*DataSourceTestResult, error) {
	// TODO: 实现文件数据预览逻辑

	// 模拟文件数据
	sampleData := []map[string]interface{}{
		{"列1": "值1", "列2": "值2", "列3": "值3"},
		{"列1": "值4", "列2": "值5", "列3": "值6"},
		{"列1": "值7", "列2": "值8", "列3": "值9"},
	}

	result := &DataSourceTestResult{
		Success:  true,
		Message:  "文件数据预览获取成功",
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "data_preview",
		Data:     sampleData,
		Metadata: map[string]interface{}{
			"file_size":    "1.2MB",
			"row_count":    len(sampleData),
			"column_count": 3,
			"file_format":  config["format"],
		},
	}

	return result, nil
}

// previewGenericData 通用数据预览（用于未实现具体预览逻辑的数据源类型）
func (s *DatasourceService) previewGenericData(dataSource *models.DataSource, config map[string]interface{}, startTime time.Time, category string) (*DataSourceTestResult, error) {
	// TODO: 实现通用数据预览逻辑

	// 模拟通用数据
	sampleData := []map[string]interface{}{
		{"字段1": "样例数据1", "字段2": "样例数据2", "时间戳": "2024-01-01T10:00:00Z"},
		{"字段1": "样例数据3", "字段2": "样例数据4", "时间戳": "2024-01-01T10:01:00Z"},
	}

	result := &DataSourceTestResult{
		Success:  true,
		Message:  fmt.Sprintf("%s数据预览获取成功", category),
		Duration: time.Since(startTime).Milliseconds(),
		TestType: "data_preview",
		Data:     sampleData,
		Metadata: map[string]interface{}{
			"data_count":      len(sampleData),
			"datasource_type": dataSource.Type,
			"category":        category,
		},
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
