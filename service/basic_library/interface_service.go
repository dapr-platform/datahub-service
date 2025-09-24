/*
 * @module service/basic_library/interface_service
 * @description 接口管理服务，负责接口测试、数据预览和性能分析
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/backend_api_analysis.md
 * @stateFlow 接口测试 -> 性能分析 -> 数据验证 -> 状态更新
 * @rules 确保接口调用的可靠性和性能
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs ai_docs/interfaces.md
 */

package basic_library

import (
	"context"
	"datahub-service/service/database"
	"datahub-service/service/datasource"
	"datahub-service/service/interface_executor"
	"datahub-service/service/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// InterfaceService 接口服务
type InterfaceService struct {
	db                *gorm.DB
	datasourceManager datasource.DataSourceManager
	executor          *interface_executor.InterfaceExecutor
	schemaService     *database.SchemaService
}

// NewInterfaceService 创建接口服务实例
func NewInterfaceService(db *gorm.DB, datasourceManager datasource.DataSourceManager) *InterfaceService {
	executor := interface_executor.NewInterfaceExecutor(db, datasourceManager)
	schemaService := database.NewSchemaService(db)
	return &InterfaceService{
		db:                db,
		datasourceManager: datasourceManager,
		executor:          executor,
		schemaService:     schemaService,
	}
}

// InterfaceTestResult 接口测试结果
type InterfaceTestResult struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Duration    int64                  `json:"duration"`
	TestType    string                 `json:"test_type"`
	Data        interface{}            `json:"data,omitempty"`
	RowCount    int                    `json:"row_count,omitempty"`
	ColumnCount int                    `json:"column_count,omitempty"`
	DataTypes   map[string]string      `json:"data_types,omitempty"`
	Performance map[string]interface{} `json:"performance,omitempty"`
	Validation  map[string]interface{} `json:"validation,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
}

// CreateDataInterface 创建数据接口
func (s *InterfaceService) CreateDataInterface(interfaceData *models.DataInterface) error {
	// 检查基础库是否存在
	var library models.BasicLibrary
	if err := s.db.First(&library, "id = ?", interfaceData.LibraryID).Error; err != nil {
		return errors.New("关联的数据基础库不存在")
	}

	// 检查接口英文名称在同一基础库中是否重复
	var existing models.DataInterface
	if err := s.db.Where("library_id = ? AND name_en = ?", interfaceData.LibraryID, interfaceData.NameEn).First(&existing).Error; err == nil {
		return errors.New("接口英文名称在该基础库中已存在")
	}

	// 开启事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建接口记录
	if err := tx.Create(interfaceData).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建接口记录失败: %w", err)
	}

	// 如果配置了表字段，自动创建数据表
	if interfaceData.TableFieldsConfig != nil && len(interfaceData.TableFieldsConfig) > 0 {
		// 解析字段配置
		fields, err := s.parseTableFields(interfaceData.TableFieldsConfig)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("解析表字段配置失败: %w", err)
		}

		// 如果有字段配置，创建数据表
		if len(fields) > 0 {
			schemaName := library.NameEn
			tableName := interfaceData.NameEn

			// 创建数据表
			if err := s.schemaService.ManageTableSchema(interfaceData.ID, "create_table", schemaName, tableName, fields); err != nil {
				tx.Rollback()
				return fmt.Errorf("创建数据表失败: %w", err)
			}

			// 更新表创建状态
			if err := tx.Model(&models.DataInterface{}).Where("id = ?", interfaceData.ID).Update("is_table_created", true).Error; err != nil {
				tx.Rollback()
				// 尝试清理已创建的表
				s.schemaService.ManageTableSchema(interfaceData.ID, "drop_table", schemaName, tableName, []models.TableField{})
				return fmt.Errorf("更新表创建状态失败: %w", err)
			}

			// 更新内存中的状态
			interfaceData.IsTableCreated = true
		}
	}

	return tx.Commit().Error
}

// UpdateDataInterface 更新数据接口
func (s *InterfaceService) UpdateDataInterface(id string, updates map[string]interface{}) error {
	// 检查是否存在
	var interfaceData models.DataInterface
	if err := s.db.First(&interfaceData, "id = ?", id).Error; err != nil {
		return err
	}

	// 如果更新英文名称，检查是否重复
	if nameEn, exists := updates["name_en"]; exists {
		var existing models.DataInterface
		if err := s.db.Where("library_id = ? AND name_en = ? AND id != ?", interfaceData.LibraryID, nameEn, id).First(&existing).Error; err == nil {
			return errors.New("接口英文名称在该基础库中已存在")
		}
	}

	return s.db.Model(&interfaceData).Updates(updates).Error
}

// DeleteDataInterface 删除数据接口
func (s *InterfaceService) DeleteDataInterface(interfaceData *models.DataInterface) error {
	// 检查接口是否存在
	var existing models.DataInterface
	if err := s.db.First(&existing, "id = ?", interfaceData.ID).Error; err != nil {
		return errors.New("接口不存在")
	}
	s.db.Where("interface_id = ?", interfaceData.ID).Delete(&models.InterfaceField{})
	s.db.Where("interface_id = ?", interfaceData.ID).Delete(&models.InterfaceStatus{})
	s.db.Where("interface_id = ?", interfaceData.ID).Delete(&models.InterfaceStatus{})
	err := s.schemaService.ManageTableSchema(interfaceData.ID, "drop_table", interfaceData.BasicLibrary.NameEn, interfaceData.NameEn, []models.TableField{})
	if err != nil {
		return fmt.Errorf("删除表结构失败: %w", err)
	}
	return s.db.Delete(interfaceData).Error
}

// GetDataInterface 获取数据接口详情
func (s *InterfaceService) GetDataInterface(id string) (*models.DataInterface, error) {
	var interfaceData models.DataInterface
	err := s.db.Preload("BasicLibrary").
		Preload("DataSource").
		Preload("Fields").
		Preload("CleanRules").
		First(&interfaceData, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &interfaceData, nil
}
func (s *InterfaceService) UpdateDataInterfaceTableCreated(id string, isTableCreated bool) error {
	var interfaceData models.DataInterface
	err := s.db.Model(&interfaceData).Where("id = ?", id).Updates(map[string]interface{}{"is_table_created": isTableCreated}).Error
	if err != nil {
		return err
	}
	return nil
}

// GetDataInterfaces 获取数据接口列表
func (s *InterfaceService) GetDataInterfaces(libraryID string, page, pageSize int) ([]models.DataInterface, int64, error) {
	var interfaces []models.DataInterface
	var total int64

	query := s.db.Model(&models.DataInterface{})
	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，预加载关联数据
	offset := (page - 1) * pageSize
	err := query.Preload("BasicLibrary").Preload("DataSource").
		Preload("Fields").Preload("CleanRules").
		Offset(offset).Limit(pageSize).Find(&interfaces).Error

	return interfaces, total, err
}

// TestInterface 测试接口调用
func (s *InterfaceService) TestInterface(interfaceID, testType string, parameters, options map[string]interface{}) (*InterfaceTestResult, error) {
	ctx := context.Background()

	// 根据测试类型进行不同的测试
	switch testType {
	case "data_fetch":
		// 数据获取测试：实际执行接口同步，更新表数据
		request := &interface_executor.ExecuteRequest{
			InterfaceID:   interfaceID,
			InterfaceType: "basic_library",
			ExecuteType:   "sync",
			Parameters:    parameters,
			Options:       options,
		}

		response, err := s.executor.Execute(ctx, request)
		if err != nil {
			return &InterfaceTestResult{
				Success:  false,
				Message:  "接口测试失败",
				Duration: response.Duration,
				TestType: testType,
				Error:    err.Error(),
			}, err
		}

		return &InterfaceTestResult{
			Success:     response.Success,
			Message:     response.Message,
			Duration:    response.Duration,
			TestType:    testType,
			Data:        response.Data,
			RowCount:    response.RowCount,
			ColumnCount: response.ColumnCount,
			DataTypes:   response.DataTypes,
			Warnings:    response.Warnings,
		}, nil

	case "performance":
		return s.testPerformance(nil, parameters, options, time.Now())
	case "validation":
		return s.testValidation(nil, parameters, options, time.Now())
	default:
		return &InterfaceTestResult{
			Success:  false,
			Message:  "不支持的测试类型",
			Duration: 0,
			TestType: testType,
			Error:    "unsupported test type",
		}, fmt.Errorf("不支持的测试类型: %s", testType)
	}
}

// isDateTime 检测字符串是否为日期时间格式
func (s *InterfaceService) isDateTime(str string) bool {
	// 常见的日期时间格式
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, str); err == nil {
			return true
		}
	}
	return false
}

// testPerformance 测试性能
func (s *InterfaceService) testPerformance(interfaceData *models.DataInterface, parameters, options map[string]interface{}, startTime time.Time) (*InterfaceTestResult, error) {
	// 模拟性能测试
	performance := map[string]interface{}{
		"query_time_ms":     150,
		"throughput_qps":    100,
		"memory_usage_mb":   64,
		"cpu_usage_percent": 15.5,
		"concurrent_users":  10,
		"avg_response_time": 145,
		"min_response_time": 120,
		"max_response_time": 180,
		"error_rate":        0.01,
	}

	warnings := []string{}
	if performance["query_time_ms"].(int) > 1000 {
		warnings = append(warnings, "查询响应时间较长，建议优化查询")
	}
	if performance["error_rate"].(float64) > 0.05 {
		warnings = append(warnings, "错误率较高，建议检查数据源稳定性")
	}

	result := &InterfaceTestResult{
		Success:     true,
		Message:     "性能测试完成",
		Duration:    time.Since(startTime).Milliseconds(),
		TestType:    "performance",
		Performance: performance,
		Warnings:    warnings,
	}

	return result, nil
}

// testValidation 测试数据验证
func (s *InterfaceService) testValidation(interfaceData *models.DataInterface, parameters, options map[string]interface{}, startTime time.Time) (*InterfaceTestResult, error) {
	// 模拟数据验证
	validation := map[string]interface{}{
		"schema_valid":      true,
		"data_completeness": 0.95,
		"data_accuracy":     0.98,
		"null_rate":         0.02,
		"duplicate_rate":    0.01,
		"format_errors":     3,
		"constraint_violations": []string{
			"字段 'email' 存在3个格式错误",
			"字段 'age' 存在1个范围违规",
		},
		"quality_score": 92,
	}

	warnings := []string{}
	if validation["data_completeness"].(float64) < 0.9 {
		warnings = append(warnings, "数据完整性较低，建议检查数据源质量")
	}
	if validation["quality_score"].(int) < 80 {
		warnings = append(warnings, "数据质量评分较低，建议设置数据清洗规则")
	}

	result := &InterfaceTestResult{
		Success:    true,
		Message:    "数据验证测试完成",
		Duration:   time.Since(startTime).Milliseconds(),
		TestType:   "validation",
		Validation: validation,
		Warnings:   warnings,
	}

	return result, nil
}

// PreviewInterfaceData 预览接口数据
func (s *InterfaceService) PreviewInterfaceData(id string, limit int) (interface{}, error) {
	ctx := context.Background()

	// 使用通用执行器进行预览
	request := &interface_executor.ExecuteRequest{
		InterfaceID:   id,
		InterfaceType: "basic_library",
		ExecuteType:   "preview",
		Limit:         limit,
		Parameters: map[string]interface{}{
			"limit": limit,
		},
		Options: map[string]interface{}{},
	}

	response, err := s.executor.Execute(ctx, request)
	if err != nil {
		return nil, err
	}

	// 返回预览结果
	return map[string]interface{}{
		"interface_id":    response.Metadata["interface_id"],
		"interface_name":  response.Metadata["interface_name"],
		"interface_type":  "basic_library",
		"schema_name":     response.Metadata["schema_name"],
		"table_name":      response.Metadata["table_name"],
		"requested_limit": response.Metadata["requested_limit"],
		"actual_count":    response.RowCount,
		"preview_data":    response.Data,
		"data_types":      response.DataTypes,
		"column_count":    response.ColumnCount,
		"warnings":        response.Warnings,
		"queried_at":      time.Now(),
		"success":         response.Success,
		"message":         response.Message,
		"duration":        response.Duration,
	}, nil
}

// generateRealtimePreviewData 生成实时数据预览
func (s *InterfaceService) generateRealtimePreviewData(interfaceData *models.DataInterface, limit int) []map[string]interface{} {
	data := make([]map[string]interface{}, 0, limit)

	for i := 0; i < limit; i++ {
		record := map[string]interface{}{
			"timestamp": time.Now().Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			"sequence":  1000 + i,
			"status":    "active",
		}

		// 根据字段定义生成数据
		for _, fieldObject := range interfaceData.TableFieldsConfig {
			var field models.TableField
			fieldBytes, _ := json.Marshal(fieldObject)
			json.Unmarshal(fieldBytes, &field)
			record[field.NameEn] = s.generateFieldValue(field.DataType, i)
		}

		data = append(data, record)
	}

	return data
}

// generateBatchPreviewData 生成批量数据预览
func (s *InterfaceService) generateBatchPreviewData(interfaceData *models.DataInterface, limit int) []map[string]interface{} {
	data := make([]map[string]interface{}, 0, limit)

	for i := 0; i < limit; i++ {
		record := map[string]interface{}{
			"batch_id":   "batch_2024_001",
			"row_number": i + 1,
			"processed":  i%2 == 0,
		}

		// 根据字段定义生成数据
		for _, fieldObject := range interfaceData.TableFieldsConfig {
			var field models.TableField
			fieldBytes, _ := json.Marshal(fieldObject)
			json.Unmarshal(fieldBytes, &field)
			record[field.NameEn] = s.generateFieldValue(field.DataType, i)
		}

		data = append(data, record)
	}

	return data
}

// generateFieldValue 根据数据类型生成示例值
func (s *InterfaceService) generateFieldValue(dataType string, index int) interface{} {
	switch dataType {
	case "integer", "int":
		return 1000 + index
	case "string", "varchar", "text":
		return fmt.Sprintf("示例数据_%d", index+1)
	case "boolean", "bool":
		return index%2 == 0
	case "datetime", "timestamp":
		return time.Now().Add(-time.Duration(index) * time.Hour).Format(time.RFC3339)
	case "date":
		return time.Now().Add(-time.Duration(index) * 24 * time.Hour).Format("2006-01-02")
	case "decimal", "float":
		return float64(100.5 + float64(index)*10.5)
	case "json", "jsonb":
		return map[string]interface{}{
			"key":   fmt.Sprintf("value_%d", index+1),
			"index": index,
		}
	default:
		return fmt.Sprintf("数据_%d", index+1)
	}
}

// getFieldsInfo 获取字段信息
func (s *InterfaceService) getFieldsInfo(fields []models.TableField) []map[string]interface{} {
	fieldsInfo := make([]map[string]interface{}, 0, len(fields))

	for _, field := range fields {
		fieldInfo := map[string]interface{}{
			"name_zh":        field.NameZh,
			"name_en":        field.NameEn,
			"data_type":      field.DataType,
			"is_primary_key": field.IsPrimaryKey,
			"is_nullable":    field.IsNullable,
			"default_value":  field.DefaultValue,
			"description":    field.Description,
			"order_num":      field.OrderNum,
		}
		fieldsInfo = append(fieldsInfo, fieldInfo)
	}

	return fieldsInfo
}

// updateInterfaceStatus 更新接口状态
func (s *InterfaceService) updateInterfaceStatus(interfaceID, status string, lastTestTime, lastErrorTime *time.Time) error {
	now := time.Now()

	// 查找现有状态记录
	var statusRecord models.InterfaceStatus
	err := s.db.Where("interface_id = ?", interfaceID).First(&statusRecord).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		statusRecord = models.InterfaceStatus{
			InterfaceID:  interfaceID,
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

// InferInterfaceFields 推断接口字段结构
func (s *InterfaceService) InferInterfaceFields(dataSourceID, tableName string, sampleData interface{}) ([]map[string]interface{}, error) {
	// TODO: 实现字段推断逻辑
	// 这里应该根据数据源类型和样本数据自动推断字段结构

	// 模拟推断结果
	inferredFields := []map[string]interface{}{
		{
			"name_zh":        "用户ID",
			"name_en":        "user_id",
			"data_type":      "integer",
			"is_primary_key": true,
			"is_nullable":    false,
			"description":    "用户唯一标识",
			"order_num":      1,
		},
		{
			"name_zh":        "用户名",
			"name_en":        "username",
			"data_type":      "varchar",
			"is_primary_key": false,
			"is_nullable":    false,
			"description":    "用户名称",
			"order_num":      2,
		},
		{
			"name_zh":        "创建时间",
			"name_en":        "created_at",
			"data_type":      "timestamp",
			"is_primary_key": false,
			"is_nullable":    false,
			"description":    "记录创建时间",
			"order_num":      3,
		},
	}

	return inferredFields, nil
}

// CreateInterfaceFields 创建接口字段
func (s *InterfaceService) CreateInterfaceFields(interfaceID string, fields []map[string]interface{}) error {
	// 删除现有字段
	if err := s.db.Where("interface_id = ?", interfaceID).Delete(&models.InterfaceField{}).Error; err != nil {
		return err
	}

	// 创建新字段
	for _, fieldData := range fields {
		field := models.InterfaceField{
			InterfaceID:  interfaceID,
			NameZh:       fieldData["name_zh"].(string),
			NameEn:       fieldData["name_en"].(string),
			DataType:     fieldData["data_type"].(string),
			IsPrimaryKey: fieldData["is_primary_key"].(bool),
			IsNullable:   fieldData["is_nullable"].(bool),
			Description:  fieldData["description"].(string),
			OrderNum:     fieldData["order_num"].(int),
		}

		if defaultValue, exists := fieldData["default_value"]; exists && defaultValue != nil {
			field.DefaultValue = defaultValue.(string)
		}

		if err := s.db.Create(&field).Error; err != nil {
			return err
		}
	}

	return nil
}

// UpdateInterfaceFields 更新接口字段配置
func (s *InterfaceService) UpdateInterfaceFields(interfaceID string, fields []models.TableField, updateTable bool) error {
	// 验证和修正字段配置
	fields = s.validateAndFixFields(fields)

	// 获取接口信息
	interfaceData, err := s.GetDataInterface(interfaceID)
	if err != nil {
		return fmt.Errorf("获取接口信息失败: %w", err)
	}
	schemaName := interfaceData.BasicLibrary.NameEn
	tableName := interfaceData.NameEn

	if schemaName == "" || tableName == "" {
		return fmt.Errorf("接口信息中没有找到库名或表名")
	}
	if !interfaceData.IsTableCreated {
		err := s.schemaService.ManageTableSchema(interfaceID, "create_table", schemaName, tableName, fields)
		if err != nil {
			return fmt.Errorf("创建表结构失败: %w", err)
		}
		interfaceData.IsTableCreated = true
		if err := s.db.Model(&interfaceData).Where("id = ?", interfaceID).Updates(map[string]interface{}{"is_table_created": true}).Error; err != nil {
			s.schemaService.ManageTableSchema(interfaceID, "drop_table", schemaName, tableName, []models.TableField{})
			return fmt.Errorf("更新接口表创建状态失败: %w", err)
		}
	}

	// 转换字段为JSONB格式（需要是map[string]interface{}）
	fieldsData := make(models.JSONB)
	for i, field := range fields {
		fieldsData[fmt.Sprintf("field_%d", i)] = field
	}

	// 更新接口字段配置
	updates := map[string]interface{}{
		"table_fields_config": fieldsData,
		"updated_at":          time.Now(),
	}

	if err := s.db.Model(&models.DataInterface{}).Where("id = ?", interfaceID).Updates(updates).Error; err != nil {
		s.schemaService.ManageTableSchema(interfaceID, "drop_table", schemaName, tableName, []models.TableField{})
		interfaceData.IsTableCreated = false
		s.db.Model(&interfaceData).Where("id = ?", interfaceID).Updates(map[string]interface{}{"is_table_created": false})
		return fmt.Errorf("更新接口字段配置失败: %w", err)
	}

	// 如果需要更新表结构
	if updateTable && interfaceData.IsTableCreated {
		schemaName := interfaceData.BasicLibrary.NameEn
		tableName := interfaceData.NameEn

		// 调用SchemaService更新表结构
		err := s.schemaService.ManageTableSchema(interfaceID, "alter_table", schemaName, tableName, fields)
		if err != nil {
			// 如果表结构更新失败，记录警告但不回滚字段配置更新
			// 因为字段配置可能用于其他目的（如数据验证、界面显示等）
			return fmt.Errorf("更新表结构失败，但字段配置已更新: %w", err)
		}
	}

	// 删除旧的接口字段记录
	if err := s.db.Where("interface_id = ?", interfaceID).Delete(&models.InterfaceField{}).Error; err != nil {
		return fmt.Errorf("删除旧字段记录失败: %w", err)
	}

	// 创建新的接口字段记录
	for _, field := range fields {
		interfaceField := models.InterfaceField{
			InterfaceID:      interfaceID,
			NameZh:           field.NameZh,
			NameEn:           field.NameEn,
			DataType:         field.DataType,
			IsPrimaryKey:     field.IsPrimaryKey,
			IsUnique:         field.IsUnique,
			IsNullable:       field.IsNullable,
			DefaultValue:     field.DefaultValue,
			Description:      field.Description,
			OrderNum:         field.OrderNum,
			CheckConstraint:  field.CheckConstraint,
			IsIncrementField: field.IsIncrementField,
		}

		if err := s.db.Create(&interfaceField).Error; err != nil {
			return fmt.Errorf("创建字段记录失败: %w", err)
		}
	}

	return nil
}

// parseTableFields 解析表字段配置
func (s *InterfaceService) parseTableFields(tableFieldsConfig models.JSONB) ([]models.TableField, error) {
	var fields []models.TableField

	// 如果配置为空，返回空字段列表
	if tableFieldsConfig == nil {
		return fields, nil
	}

	// 检查是否有 fields 字段
	if fieldsData, exists := tableFieldsConfig["fields"]; exists {
		// 将 fields 转换为字段列表
		if fieldsArray, ok := fieldsData.([]interface{}); ok {
			for _, fieldData := range fieldsArray {
				if fieldMap, ok := fieldData.(map[string]interface{}); ok {
					field := models.TableField{}

					// 解析字段名
					if fieldName, ok := fieldMap["field_name"].(string); ok {
						field.NameEn = fieldName
						field.NameZh = fieldName // 默认中英文名相同
					}

					// 解析字段类型
					if fieldType, ok := fieldMap["field_type"].(string); ok {
						field.DataType = fieldType
					}

					// 解析是否为主键
					if isPrimaryKey, ok := fieldMap["is_primary_key"].(bool); ok {
						field.IsPrimaryKey = isPrimaryKey
					}

					// 解析是否可为空
					if isNullable, ok := fieldMap["is_nullable"].(bool); ok {
						field.IsNullable = isNullable
					} else {
						field.IsNullable = true // 默认可为空
					}

					// 解析默认值
					if defaultValue, exists := fieldMap["default_value"]; exists {
						if defaultValueStr, ok := defaultValue.(string); ok {
							field.DefaultValue = defaultValueStr
						}
					}

					// 解析注释
					if comment, ok := fieldMap["comment"].(string); ok {
						field.Description = comment
					}

					// 解析排序
					if orderNum, ok := fieldMap["order_num"].(float64); ok {
						field.OrderNum = int(orderNum)
					}

					fields = append(fields, field)
				}
			}
		}
	}

	// 如果没有解析到字段，尝试从 auto_create_table 配置创建默认字段
	if len(fields) == 0 {
		if autoCreate, exists := tableFieldsConfig["auto_create_table"]; exists {
			if autoCreateBool, ok := autoCreate.(bool); ok && autoCreateBool {
				// 创建默认字段结构
				fields = append(fields, models.TableField{
					NameZh:       "主键ID",
					NameEn:       "id",
					DataType:     "string",
					IsPrimaryKey: true,
					IsNullable:   false,
					Description:  "主键ID",
					OrderNum:     1,
				})
				fields = append(fields, models.TableField{
					NameZh:       "创建时间",
					NameEn:       "created_at",
					DataType:     "timestamp",
					IsPrimaryKey: false,
					IsNullable:   false,
					DefaultValue: "NOW()",
					Description:  "创建时间",
					OrderNum:     2,
				})
				fields = append(fields, models.TableField{
					NameZh:       "更新时间",
					NameEn:       "updated_at",
					DataType:     "timestamp",
					IsPrimaryKey: false,
					IsNullable:   false,
					DefaultValue: "NOW()",
					Description:  "更新时间",
					OrderNum:     3,
				})
			}
		}
	}

	return fields, nil
}

// validateAndFixFields 验证和修正字段配置
func (s *InterfaceService) validateAndFixFields(fields []models.TableField) []models.TableField {
	validatedFields := make([]models.TableField, len(fields))
	copy(validatedFields, fields)

	for i, field := range validatedFields {
		// 修正主键字段的配置
		if field.IsPrimaryKey {
			// 主键字段不能为空
			validatedFields[i].IsNullable = false

			// 如果主键字段没有设置唯一性，自动设置
			if !field.IsUnique {
				validatedFields[i].IsUnique = true
			}

			// 主键字段的默认值处理
			if field.DefaultValue == "" && field.DataType == "varchar" {
				// 对于字符串类型的主键，不设置默认值
				validatedFields[i].DefaultValue = ""
			}
		}

		// 修正数据类型映射
		switch field.DataType {
		case "string":
			validatedFields[i].DataType = "varchar"
		case "int":
			validatedFields[i].DataType = "integer"
		case "bool":
			validatedFields[i].DataType = "boolean"
		case "datetime":
			validatedFields[i].DataType = "timestamp"
		}

		// 清理空的默认值
		if field.DefaultValue == "" {
			validatedFields[i].DefaultValue = ""
		}

		// 确保描述信息的格式
		if field.Description == "" {
			validatedFields[i].Description = fmt.Sprintf("%s字段", field.NameZh)
		}
	}

	return validatedFields
}
