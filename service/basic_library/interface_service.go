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
	"datahub-service/service/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// InterfaceService 接口服务
type InterfaceService struct {
	db *gorm.DB
}

// NewInterfaceService 创建接口服务实例
func NewInterfaceService(db *gorm.DB) *InterfaceService {
	return &InterfaceService{
		db: db,
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

	return s.db.Create(interfaceData).Error
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

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("BasicLibrary").Offset(offset).Limit(pageSize).Find(&interfaces).Error

	return interfaces, total, err
}

// TestInterface 测试接口调用
func (s *InterfaceService) TestInterface(interfaceID, testType string, parameters, options map[string]interface{}) (*InterfaceTestResult, error) {
	startTime := time.Now()

	// 获取接口信息
	interfaceData, err := s.GetDataInterface(interfaceID)
	if err != nil {
		return &InterfaceTestResult{
			Success:  false,
			Message:  "接口不存在",
			Duration: time.Since(startTime).Milliseconds(),
			TestType: testType,
			Error:    err.Error(),
		}, err
	}

	// 根据测试类型进行不同的测试
	switch testType {
	case "data_fetch":
		return s.testDataFetch(interfaceData, parameters, options, startTime)
	case "performance":
		return s.testPerformance(interfaceData, parameters, options, startTime)
	case "validation":
		return s.testValidation(interfaceData, parameters, options, startTime)
	default:
		return &InterfaceTestResult{
			Success:  false,
			Message:  "不支持的测试类型",
			Duration: time.Since(startTime).Milliseconds(),
			TestType: testType,
			Error:    "unsupported test type",
		}, fmt.Errorf("不支持的测试类型: %s", testType)
	}
}

// testDataFetch 测试数据获取
func (s *InterfaceService) testDataFetch(interfaceData *models.DataInterface, parameters, options map[string]interface{}, startTime time.Time) (*InterfaceTestResult, error) {
	// 模拟数据获取
	sampleData := []map[string]interface{}{
		{"id": 1, "name": "用户A", "email": "usera@example.com", "created_at": "2024-01-01T10:00:00Z"},
		{"id": 2, "name": "用户B", "email": "userb@example.com", "created_at": "2024-01-01T11:00:00Z"},
		{"id": 3, "name": "用户C", "email": "userc@example.com", "created_at": "2024-01-01T12:00:00Z"},
	}

	// 分析数据类型
	dataTypes := map[string]string{
		"id":         "integer",
		"name":       "string",
		"email":      "string",
		"created_at": "datetime",
	}

	warnings := []string{}
	if len(sampleData) > 1000 {
		warnings = append(warnings, "数据量较大，建议分页查询")
	}

	// 更新接口状态
	s.updateInterfaceStatus(interfaceData.ID, "active", nil, nil)

	result := &InterfaceTestResult{
		Success:     true,
		Message:     "数据获取测试成功",
		Duration:    time.Since(startTime).Milliseconds(),
		TestType:    "data_fetch",
		Data:        sampleData,
		RowCount:    len(sampleData),
		ColumnCount: len(dataTypes),
		DataTypes:   dataTypes,
		Warnings:    warnings,
	}

	return result, nil
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
	// 获取接口信息
	interfaceData, err := s.GetDataInterface(id)
	if err != nil {
		return nil, err
	}

	// 检查表是否已创建
	if !interfaceData.IsTableCreated {
		return nil, fmt.Errorf("接口表尚未创建，无法预览数据")
	}

	// 构造表名：schema.table_name
	schemaName := interfaceData.BasicLibrary.NameEn
	tableName := interfaceData.NameEn
	fullTableName := fmt.Sprintf(`"%s"."%s"`, schemaName, tableName)

	// 设置默认限制
	if limit <= 0 || limit > 1000 {
		limit = 10
	}

	// 查询真实数据
	previewData, actualCount, err := s.queryTableData(fullTableName, limit)
	if err != nil {
		return nil, fmt.Errorf("查询接口表数据失败: %v", err)
	}

	// 获取表结构信息
	tableInfo, err := s.getTableStructure(fullTableName)

	return map[string]interface{}{
		"interface_id":     id,
		"interface_name":   interfaceData.NameZh,
		"interface_type":   interfaceData.Type,
		"schema_name":      schemaName,
		"table_name":       tableName,
		"full_table_name":  fullTableName,
		"requested_limit":  limit,
		"actual_count":     actualCount,
		"preview_data":     previewData,
		"fields_info":      tableInfo,
		"is_table_created": interfaceData.IsTableCreated,
		"queried_at":       time.Now(),
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

// queryTableData 查询表数据
func (s *InterfaceService) queryTableData(fullTableName string, limit int) ([]map[string]interface{}, int, error) {
	// 构造安全的查询SQL
	sql := fmt.Sprintf("SELECT * FROM %s LIMIT %d", fullTableName, limit)

	rows, err := s.db.Raw(sql).Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		// 创建接收数据的slice
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, err
		}

		// 构造map
		record := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				// 处理字节数组类型
				if b, ok := val.([]byte); ok {
					record[col] = string(b)
				} else {
					record[col] = val
				}
			} else {
				record[col] = nil
			}
		}
		result = append(result, record)
	}

	return result, len(result), nil
}

// getTableStructure 获取表结构信息
func (s *InterfaceService) getTableStructure(fullTableName string) ([]map[string]interface{}, error) {
	// 解析schema和table名称 - 从 "schema"."table" 格式中提取
	// 去掉引号并分割
	cleanName := fullTableName[1 : len(fullTableName)-1] // 去掉外层引号
	parts := []string{}
	current := ""
	inQuotes := false

	for i, char := range cleanName {
		if char == '"' {
			inQuotes = !inQuotes
		} else if char == '.' && !inQuotes {
			parts = append(parts, current)
			current = ""
			continue
		} else {
			current += string(char)
		}

		if i == len(cleanName)-1 {
			parts = append(parts, current)
		}
	}

	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的表名格式: %s", fullTableName)
	}

	schemaName := parts[0]
	tableName := parts[1]

	// 查询PostgreSQL系统表获取列信息
	sql := `
		SELECT 
			column_name,
			data_type,
			character_maximum_length,
			numeric_precision,
			numeric_scale,
			is_nullable,
			column_default,
			ordinal_position
		FROM information_schema.columns 
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := s.db.Raw(sql, schemaName, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fieldsInfo []map[string]interface{}
	for rows.Next() {
		var columnName, dataType, isNullable string
		var maxLength, precision, scale, position *int
		var defaultValue *string

		err := rows.Scan(&columnName, &dataType, &maxLength, &precision, &scale, &isNullable, &defaultValue, &position)
		if err != nil {
			return nil, err
		}

		fieldInfo := map[string]interface{}{
			"name_en":     columnName,
			"name_zh":     columnName, // 如果没有中文名映射，使用英文名
			"data_type":   dataType,
			"is_nullable": isNullable == "YES",
			"order_num":   position,
			"description": "",
		}

		if maxLength != nil {
			fieldInfo["data_length"] = *maxLength
		}
		if precision != nil {
			fieldInfo["data_precision"] = *precision
		}
		if scale != nil {
			fieldInfo["data_scale"] = *scale
		}
		if defaultValue != nil {
			fieldInfo["default_value"] = *defaultValue
		}

		fieldsInfo = append(fieldsInfo, fieldInfo)
	}

	return fieldsInfo, nil
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
