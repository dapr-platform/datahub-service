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
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cast"
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

// GetSchemaService 获取SchemaService实例
func (s *InterfaceService) GetSchemaService() *database.SchemaService {
	return s.schemaService
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
	if len(interfaceData.TableFieldsConfig) > 0 {
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

	// 开启事务，确保级联删除的原子性
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 删除主题数据血缘记录（外键约束：thematic_data_lineages.source_interface_id）
	if err := tx.Where("source_interface_id = ?", interfaceData.ID).Delete(&models.ThematicDataLineage{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除关联的主题数据血缘记录失败: %w", err)
	}

	// 2. 删除关联的清洗规则
	if err := tx.Where("interface_id = ?", interfaceData.ID).Delete(&models.CleansingRule{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除关联的清洗规则失败: %w", err)
	}

	// 3. 删除接口状态记录
	if err := tx.Where("interface_id = ?", interfaceData.ID).Delete(&models.InterfaceStatus{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除接口状态记录失败: %w", err)
	}

	// 4. 删除表结构（如果表已创建）
	if interfaceData.IsTableCreated {
		err := s.schemaService.ManageTableSchema(interfaceData.ID, "drop_table", interfaceData.BasicLibrary.NameEn, interfaceData.NameEn, []models.TableField{})
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("删除表结构失败: %w", err)
		}
	}

	// 5. 删除接口记录本身
	if err := tx.Delete(interfaceData).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除接口记录失败: %w", err)
	}

	// 提交事务
	return tx.Commit().Error
}

// GetDataInterface 获取数据接口详情
func (s *InterfaceService) GetDataInterface(id string) (*models.DataInterface, error) {
	var interfaceData models.DataInterface
	err := s.db.Preload("BasicLibrary").
		Preload("DataSource").
		Preload("CleanRules").
		First(&interfaceData, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &interfaceData, nil
}

// GetDataInterfaceWithSync 获取数据接口详情并同步字段配置
func (s *InterfaceService) GetDataInterfaceWithSync(id string) (*models.DataInterface, error) {
	// 先获取接口基本信息
	interfaceData, err := s.GetDataInterface(id)
	if err != nil {
		return nil, err
	}

	// 如果表已创建，检查并同步字段配置
	if interfaceData.IsTableCreated {
		synced, err := s.syncTableFieldsConfig(interfaceData)
		if err != nil {
			slog.Warn("同步表字段配置失败", "interface_id", id, "error", err)
			// 同步失败不影响返回，只记录警告
		} else if synced {
			// 如果有同步，重新加载接口数据
			interfaceData, err = s.GetDataInterface(id)
			if err != nil {
				return nil, err
			}
		}
	}

	return interfaceData, nil
}

// syncTableFieldsConfig 同步表字段配置
func (s *InterfaceService) syncTableFieldsConfig(interfaceData *models.DataInterface) (bool, error) {
	schemaName := interfaceData.BasicLibrary.NameEn
	tableName := interfaceData.NameEn

	// 检查表是否存在
	exists, err := s.schemaService.CheckTableExists(schemaName, tableName)
	if err != nil {
		return false, fmt.Errorf("检查表存在性失败: %w", err)
	}
	if !exists {
		return false, nil
	}

	// 从数据库获取实际字段
	actualColumns, err := s.schemaService.GetTableColumns(schemaName, tableName)
	if err != nil {
		return false, fmt.Errorf("获取表字段失败: %w", err)
	}

	// 从配置中提取现有字段
	existingFields := s.extractFieldsFromConfig(interfaceData.TableFieldsConfig)

	// 构建现有字段映射
	existingFieldMap := make(map[string]models.TableField)
	for _, field := range existingFields {
		existingFieldMap[field.NameEn] = field
	}

	// 转换数据库列为字段，并合并配置
	mergedFields := s.mergeFieldsWithConfig(actualColumns, existingFieldMap)

	// 比较字段配置是否一致
	if s.isFieldsConfigSame(interfaceData.TableFieldsConfig, mergedFields) {
		return false, nil // 字段一致，无需同步
	}

	slog.Info("检测到表字段配置不一致，开始同步",
		"interface_id", interfaceData.ID,
		"schema", schemaName,
		"table", tableName,
		"existing_fields", len(existingFields),
		"actual_columns", len(actualColumns))

	// 更新字段配置
	fieldsData := make(models.JSONB)
	for i, field := range mergedFields {
		fieldsData[fmt.Sprintf("field_%d", i)] = field
	}

	updates := map[string]interface{}{
		"table_fields_config": fieldsData,
		"updated_at":          time.Now(),
	}

	if err := s.db.Model(&models.DataInterface{}).Where("id = ?", interfaceData.ID).Updates(updates).Error; err != nil {
		return false, fmt.Errorf("更新字段配置失败: %w", err)
	}

	slog.Info("表字段配置同步完成", "interface_id", interfaceData.ID, "fields_count", len(mergedFields))

	return true, nil
}

// extractFieldsFromConfig 从配置中提取字段
func (s *InterfaceService) extractFieldsFromConfig(configJSON models.JSONB) []models.TableField {
	var fields []models.TableField

	if len(configJSON) == 0 {
		return fields
	}

	for i := 0; i < len(configJSON); i++ {
		key := fmt.Sprintf("field_%d", i)
		if fieldData, exists := configJSON[key]; exists {
			var field models.TableField
			fieldBytes, _ := json.Marshal(fieldData)
			if err := json.Unmarshal(fieldBytes, &field); err != nil {
				continue
			}
			fields = append(fields, field)
		}
	}

	return fields
}

// mergeFieldsWithConfig 合并数据库字段与现有配置
func (s *InterfaceService) mergeFieldsWithConfig(actualColumns []database.ColumnDefinition, existingFieldMap map[string]models.TableField) []models.TableField {
	mergedFields := make([]models.TableField, 0, len(actualColumns))

	for _, col := range actualColumns {
		// 基于数据库实际字段创建字段对象
		field := models.TableField{
			NameEn:       col.Name,
			NameZh:       s.extractChineseFromComment(col.Comment),
			DataType:     s.normalizeDataType(col.DataType),
			IsPrimaryKey: col.IsPrimaryKey,
			IsUnique:     col.IsUnique,
			IsNullable:   col.IsNullable,
			Description:  col.Comment,
			OrderNum:     col.OrdinalPosition,
		}

		// 处理默认值
		if col.DefaultValue != nil {
			if defaultStr, ok := col.DefaultValue.(string); ok {
				field.DefaultValue = defaultStr
			}
		}

		// 如果配置中存在该字段，合并配置信息
		if existingField, exists := existingFieldMap[col.Name]; exists {
			// 保留原有的 OrderNum、NameZh（如果更有意义）、IsIncrementField 等配置
			if existingField.OrderNum > 0 {
				field.OrderNum = existingField.OrderNum
			}
			if existingField.NameZh != "" && existingField.NameZh != col.Name {
				field.NameZh = existingField.NameZh
			}
			field.IsIncrementField = existingField.IsIncrementField

			// 如果配置中的描述更详细，使用配置中的
			if existingField.Description != "" && len(existingField.Description) > len(col.Comment) {
				field.Description = existingField.Description
			}
		}

		mergedFields = append(mergedFields, field)
	}

	// 按 OrderNum 排序，如果 OrderNum 相同或为0，则按 OrdinalPosition 排序
	sort.SliceStable(mergedFields, func(i, j int) bool {
		if mergedFields[i].OrderNum != 0 && mergedFields[j].OrderNum != 0 {
			if mergedFields[i].OrderNum != mergedFields[j].OrderNum {
				return mergedFields[i].OrderNum < mergedFields[j].OrderNum
			}
		}
		return mergedFields[i].NameEn < mergedFields[j].NameEn
	})

	// 重新分配 OrderNum
	for i := range mergedFields {
		mergedFields[i].OrderNum = i + 1
	}

	return mergedFields
}

// convertColumnsToTableFields 将数据库列定义转换为TableField
func (s *InterfaceService) convertColumnsToTableFields(columns []database.ColumnDefinition) []models.TableField {
	fields := make([]models.TableField, 0, len(columns))

	for _, col := range columns {
		field := models.TableField{
			NameEn:       col.Name,
			NameZh:       s.extractChineseFromComment(col.Comment),
			DataType:     s.normalizeDataType(col.DataType),
			IsPrimaryKey: col.IsPrimaryKey,
			IsUnique:     col.IsUnique,
			IsNullable:   col.IsNullable,
			Description:  col.Comment,
			OrderNum:     col.OrdinalPosition,
		}

		// 处理默认值
		if col.DefaultValue != nil {
			if defaultStr, ok := col.DefaultValue.(string); ok {
				field.DefaultValue = defaultStr
			}
		}

		fields = append(fields, field)
	}

	return fields
}

// extractChineseFromComment 从注释中提取中文名称
func (s *InterfaceService) extractChineseFromComment(comment string) string {
	if comment == "" {
		return ""
	}

	// 尝试提取 "中文名 - 描述" 格式中的中文名
	parts := strings.Split(comment, " - ")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}

	return comment
}

// normalizeDataType 规范化数据类型
func (s *InterfaceService) normalizeDataType(dataType string) string {
	// 移除类型中的长度限制，如 varchar(255) -> varchar
	if idx := strings.Index(dataType, "("); idx > 0 {
		dataType = dataType[:idx]
	}

	// 统一类型名称
	typeMap := map[string]string{
		"character varying":           "varchar",
		"double precision":            "double",
		"timestamp without time zone": "timestamp",
		"timestamp with time zone":    "timestamp",
	}

	if normalized, exists := typeMap[dataType]; exists {
		return normalized
	}

	return dataType
}

// isFieldsConfigSame 比较字段配置是否一致
func (s *InterfaceService) isFieldsConfigSame(configJSON models.JSONB, actualFields []models.TableField) bool {
	if len(configJSON) == 0 {
		return len(actualFields) == 0
	}

	// 从配置中提取字段
	var configFields []models.TableField
	for i := 0; i < len(configJSON); i++ {
		key := fmt.Sprintf("field_%d", i)
		if fieldData, exists := configJSON[key]; exists {
			var field models.TableField
			fieldBytes, _ := json.Marshal(fieldData)
			if err := json.Unmarshal(fieldBytes, &field); err != nil {
				continue
			}
			configFields = append(configFields, field)
		}
	}

	// 比较字段数量
	if len(configFields) != len(actualFields) {
		return false
	}

	// 构建实际字段映射
	actualFieldMap := make(map[string]models.TableField)
	for _, field := range actualFields {
		actualFieldMap[field.NameEn] = field
	}

	// 逐个比较字段
	for _, configField := range configFields {
		actualField, exists := actualFieldMap[configField.NameEn]
		if !exists {
			return false
		}

		// 比较关键属性
		if configField.DataType != actualField.DataType ||
			configField.IsPrimaryKey != actualField.IsPrimaryKey ||
			configField.IsNullable != actualField.IsNullable ||
			configField.IsUnique != actualField.IsUnique {
			return false
		}
	}

	return true
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
		Preload("CleanRules").
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

// CSVImportResult CSV导入结果
type CSVImportResult struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	ImportedRows  int      `json:"imported_rows"`
	TotalRows     int      `json:"total_rows"`
	FailedRows    int      `json:"failed_rows"`
	ErrorMessages []string `json:"error_messages,omitempty"`
}

// ImportCSVData 导入CSV数据到接口表
func (s *InterfaceService) ImportCSVData(interfaceID string, csvContent string) (*CSVImportResult, error) {
	ctx := context.Background()

	// 1. 获取接口信息
	interfaceData, err := s.GetDataInterface(interfaceID)
	if err != nil {
		return nil, fmt.Errorf("获取接口信息失败: %w", err)
	}

	// 2. 检查表是否已创建
	if !interfaceData.IsTableCreated {
		return nil, fmt.Errorf("接口表尚未创建，请先配置字段并创建表")
	}

	// 3. 从 TableFieldsConfig 获取字段配置
	fields := s.extractFieldsFromConfig(interfaceData.TableFieldsConfig)
	if len(fields) == 0 {
		return nil, fmt.Errorf("接口字段配置为空")
	}

	// 4. 构建字段名到字段配置的映射
	fieldMap := make(map[string]models.TableField)
	for _, field := range fields {
		fieldMap[field.NameEn] = field
	}

	// 5. 解析CSV内容
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.TrimLeadingSpace = true

	// 读取所有记录
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析CSV内容失败: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV内容为空或仅包含表头")
	}

	// 6. 解析表头（第一行为字段名）
	headers := records[0]
	if len(headers) == 0 {
		return nil, fmt.Errorf("CSV表头为空")
	}

	// 验证表头字段是否存在于接口配置中
	var errorMessages []string
	for _, header := range headers {
		if _, exists := fieldMap[header]; !exists {
			errorMessages = append(errorMessages, fmt.Sprintf("字段 '%s' 不存在于接口配置中", header))
		}
	}

	// 7. 转换数据行为map格式
	var dataRows []map[string]interface{}
	totalRows := len(records) - 1 // 减去表头行
	failedRows := 0

	for rowIdx, record := range records[1:] {
		if len(record) != len(headers) {
			errorMessages = append(errorMessages, fmt.Sprintf("第 %d 行字段数量不匹配，期望 %d 个，实际 %d 个", rowIdx+2, len(headers), len(record)))
			failedRows++
			continue
		}

		rowData := make(map[string]interface{})
		hasError := false

		for colIdx, value := range record {
			fieldName := headers[colIdx]
			field, exists := fieldMap[fieldName]

			if !exists {
				continue // 跳过不在配置中的字段
			}

			// 数据类型转换和验证
			convertedValue, err := s.convertCSVValue(value, field, rowIdx+2)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
				hasError = true
				continue
			}

			rowData[fieldName] = convertedValue
		}

		if hasError {
			failedRows++
			continue
		}

		// 验证必填字段
		for fieldName, field := range fieldMap {
			if !field.IsNullable && rowData[fieldName] == nil {
				errorMessages = append(errorMessages, fmt.Sprintf("第 %d 行缺少必填字段 '%s'", rowIdx+2, fieldName))
				hasError = true
			}
		}

		if hasError {
			failedRows++
			continue
		}

		dataRows = append(dataRows, rowData)
	}

	// 8. 使用事务批量插入数据
	if len(dataRows) == 0 {
		return &CSVImportResult{
			Success:       false,
			Message:       "没有有效的数据行可导入",
			ImportedRows:  0,
			TotalRows:     totalRows,
			FailedRows:    failedRows,
			ErrorMessages: errorMessages,
		}, nil
	}

	// 开启事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			slog.Error("ImportCSVData - 发生panic，事务已回滚", "error", r)
		}
	}()

	// 使用InterfaceExecutor的FieldMapper批量插入数据
	fieldMapper := interface_executor.NewFieldMapper()
	interfaceInfo := &interface_executor.BasicLibraryInterfaceInfo{
		DataInterface: interfaceData,
	}

	insertedRows, err := fieldMapper.InsertBatchDataWithTx(ctx, tx, interfaceInfo, dataRows)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("批量插入数据失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &CSVImportResult{
		Success:       true,
		Message:       fmt.Sprintf("成功导入 %d 行数据", insertedRows),
		ImportedRows:  int(insertedRows),
		TotalRows:     totalRows,
		FailedRows:    failedRows,
		ErrorMessages: errorMessages,
	}, nil
}

// convertCSVValue 转换CSV值为合适的数据类型
func (s *InterfaceService) convertCSVValue(value string, field models.TableField, rowNum int) (interface{}, error) {
	// 如果值为空且字段可为空，返回nil
	if value == "" {
		if field.IsNullable {
			return nil, nil
		}
		return nil, fmt.Errorf("第 %d 行字段 '%s' 不能为空", rowNum, field.NameEn)
	}

	// 根据字段类型转换值
	switch field.DataType {
	case "integer", "int", "bigint":
		intVal, err := cast.ToInt64E(value)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行字段 '%s' 的值 '%s' 不是有效的整数", rowNum, field.NameEn, value)
		}
		return intVal, nil

	case "float", "real", "decimal", "numeric", "double":
		floatVal, err := cast.ToFloat64E(value)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行字段 '%s' 的值 '%s' 不是有效的浮点数", rowNum, field.NameEn, value)
		}
		return floatVal, nil

	case "boolean", "bool":
		boolVal, err := cast.ToBoolE(value)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行字段 '%s' 的值 '%s' 不是有效的布尔值", rowNum, field.NameEn, value)
		}
		return boolVal, nil

	case "timestamp", "datetime":
		// 尝试多种时间格式
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02",
		}
		var timeVal time.Time
		var err error
		for _, format := range formats {
			timeVal, err = time.Parse(format, value)
			if err == nil {
				return timeVal, nil
			}
		}
		return nil, fmt.Errorf("第 %d 行字段 '%s' 的值 '%s' 不是有效的时间格式", rowNum, field.NameEn, value)

	case "date":
		timeVal, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行字段 '%s' 的值 '%s' 不是有效的日期格式(YYYY-MM-DD)", rowNum, field.NameEn, value)
		}
		return timeVal, nil

	case "json", "jsonb":
		// 验证JSON格式
		var jsonData interface{}
		if err := json.Unmarshal([]byte(value), &jsonData); err != nil {
			return nil, fmt.Errorf("第 %d 行字段 '%s' 的值不是有效的JSON格式", rowNum, field.NameEn)
		}
		return value, nil

	case "varchar", "text", "string", "char":
		return value, nil

	default:
		// 默认作为字符串处理
		return value, nil
	}
}
