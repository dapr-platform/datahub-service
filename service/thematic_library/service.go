/*
 * @module service/thematic_library_service
 * @description 数据主题库业务逻辑服务，提供主题库的CRUD操作和业务处理
 * @architecture 分层架构 - 业务服务层
 * @documentReference dev_docs/requirements.md
 * @stateFlow 数据主题库管理流程
 * @rules 确保数据完整性，支持事务操作，权限控制
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs dev_docs/model.md
 */

package thematic_library

import (
	"datahub-service/service/database"
	"datahub-service/service/models"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ThematicLibraryService 数据主题库服务
type Service struct {
	db            *gorm.DB
	schemaService *database.SchemaService
}

// NewThematicLibraryService 创建数据主题库服务实例
func NewService(db *gorm.DB) *Service {
	schemaService := database.NewSchemaService(db)
	service := &Service{
		db:            db,
		schemaService: schemaService,
	}
	return service
}

// GetSchemaService 获取SchemaService实例
func (s *Service) GetSchemaService() *database.SchemaService {
	return s.schemaService
}

// CreateThematicLibrary 创建数据主题库
func (s *Service) CreateThematicLibrary(library *models.ThematicLibrary) error {
	// 检查编码是否已存在
	var existing models.ThematicLibrary
	if err := s.db.Where("name_en = ?", library.NameEn).First(&existing).Error; err == nil {
		return errors.New("主题库名称已存在")
	}

	// 验证编码格式（数据库schema命名规范）
	if !isValidSchemaName(library.NameEn) {
		return errors.New("主题库名称格式不符合数据库schema命名规范")
	}

	// 验证主题分类
	validCategories := []string{"business", "technical", "analysis", "report"}
	if !contains(validCategories, library.Category) {
		return errors.New("无效的主题分类")
	}

	// 验证数据域
	validDomains := []string{"user", "order", "product", "finance", "marketing", "asset", "supply_chain", "park_operation", "park_management", "emergency_safety", "energy", "environment", "security", "service"}
	if !contains(validDomains, library.Domain) {
		return errors.New("无效的数据域")
	}

	// 验证访问权限
	validAccessLevels := []string{"public", "internal", "private"}
	if !contains(validAccessLevels, library.AccessLevel) {
		return errors.New("无效的访问权限级别")
	}

	// 验证更新频率
	validFrequencies := []string{"realtime", "hourly", "daily", "weekly", "monthly"}
	if !contains(validFrequencies, library.UpdateFrequency) {
		return errors.New("无效的更新频率")
	}
	if database.CheckSchemaExists(s.db, library.NameEn) {
		return errors.New("主题库英文名称已存在")
	}
	err := database.CreateSchema(s.db, library.NameEn)
	if err != nil {
		return err
	}

	return s.db.Create(library).Error
}

// GetThematicLibrary 根据ID获取数据主题库
func (s *Service) GetThematicLibrary(id string) (*models.ThematicLibrary, error) {
	var library models.ThematicLibrary
	err := s.db.Preload("Interfaces").First(&library, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("获取主题库失败: %w", err)
	}
	return &library, nil
}

// GetThematicLibraries 获取数据主题库列表
func (s *Service) GetThematicLibraries(page, pageSize int, category, domain, status string) ([]models.ThematicLibrary, int64, error) {
	var libraries []models.ThematicLibrary
	var total int64

	query := s.db.Model(&models.ThematicLibrary{})

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if domain != "" {
		query = query.Where("domain = ?", domain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&libraries).Error

	return libraries, total, err
}

// GetThematicLibraryList 获取数据主题库列表（支持名称搜索）
func (s *Service) GetThematicLibraryList(page, pageSize int, category, domain, status, name string) ([]models.ThematicLibrary, int64, error) {
	var libraries []models.ThematicLibrary
	var total int64

	query := s.db.Model(&models.ThematicLibrary{})

	// 添加过滤条件
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if domain != "" {
		query = query.Where("domain = ?", domain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name_zh ILIKE ? OR name_en ILIKE ?", "%"+name+"%", "%"+name+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，预加载关联数据
	offset := (page - 1) * pageSize
	err := query.Preload("Interfaces").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).Find(&libraries).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取主题库列表失败: %w", err)
	}

	return libraries, total, nil
}

// GetThematicInterfaceList 获取主题接口列表（支持名称搜索）
func (s *Service) GetThematicInterfaceList(page, pageSize int, libraryID, interfaceType, status, name string) ([]models.ThematicInterface, int64, error) {
	var interfaces []models.ThematicInterface
	var total int64

	query := s.db.Model(&models.ThematicInterface{})

	// 添加过滤条件
	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}
	if interfaceType != "" {
		query = query.Where("type = ?", interfaceType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name_zh ILIKE ? OR name_en ILIKE ?", "%"+name+"%", "%"+name+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，预加载主题库信息
	offset := (page - 1) * pageSize
	err := query.Preload("ThematicLibrary").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).Find(&interfaces).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取主题接口列表失败: %w", err)
	}

	return interfaces, total, nil
}

// UpdateThematicLibrary 更新数据主题库
func (s *Service) UpdateThematicLibrary(id string, updates *models.ThematicLibrary) error {
	if updates.NameEn != "" {
		var existing models.ThematicLibrary
		if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
			return errors.New("主题库不存在")
		}
		if !isValidSchemaName(updates.NameEn) {
			return errors.New("主题库英文名称格式不符合数据库schema命名规范")
		}
		if existing.NameEn != updates.NameEn {
			if database.CheckSchemaExists(s.db, updates.NameEn) {
				return errors.New("主题库英文名称已存在")
			}
			err := database.DeleteSchema(s.db, existing.NameEn)
			if err != nil {
				return err
			}
			err = database.CreateSchema(s.db, updates.NameEn)
			if err != nil {
				return err
			}

		}

	}
	return s.db.Model(&models.ThematicLibrary{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteThematicLibrary 删除数据主题库
func (s *Service) DeleteThematicLibrary(id string) error {
	var existing models.ThematicLibrary
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		return errors.New("主题库不存在")
	}
	interfaces, _, err := s.GetThematicInterfaces(1, 10000, id, "", "")
	if err != nil {
		return err
	}
	if len(interfaces) > 0 {
		return errors.New("存在关联的主题接口，无法删除主题库")
	}
	err = database.DeleteSchema(s.db, existing.NameEn)
	if err != nil {
		return err
	}
	return s.db.Delete(&models.ThematicLibrary{}, "id = ?", id).Error
}

// CreateDataFlowGraph 创建数据流程图
func (s *Service) CreateDataFlowGraph(flowGraph *models.DataFlowGraph) error {

	// 验证流程图定义
	if flowGraph.Definition == nil {
		return errors.New("流程图定义不能为空")
	}

	return s.db.Create(flowGraph).Error
}

// GetDataFlowGraph 获取数据流程图详情
func (s *Service) GetDataFlowGraph(id string) (*models.DataFlowGraph, error) {
	var flowGraph models.DataFlowGraph
	err := s.db.Preload("ThematicInterface").
		Preload("Nodes").
		First(&flowGraph, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &flowGraph, nil
}

// UpdateDataFlowGraph 更新数据流程图
func (s *Service) UpdateDataFlowGraph(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataFlowGraph{}).Where("id = ?", id).Updates(updates).Error
}

// PublishThematicLibrary 发布主题库
func (s *Service) PublishThematicLibrary(id string) error {
	// 检查主题库是否存在
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", id).Error; err != nil {
		return errors.New("主题库不存在")
	}

	// 更新发布状态
	return s.db.Model(&models.ThematicLibrary{}).Where("id = ?", id).Update("publish_status", "published").Error
}

// CreateThematicInterface 创建主题接口
func (s *Service) CreateThematicInterface(thematicInterface *models.ThematicInterface) error {
	// 验证主题库是否存在
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", thematicInterface.LibraryID).Error; err != nil {
		return errors.New("主题库不存在")
	}

	// 验证英文名称格式
	if !isValidSchemaName(thematicInterface.NameEn) {
		return errors.New("接口英文名称格式不符合数据库表名命名规范")
	}

	// 检查同一主题库下接口英文名称是否重复
	var existing models.ThematicInterface
	if err := s.db.Where("library_id = ? AND name_en = ?", thematicInterface.LibraryID, thematicInterface.NameEn).First(&existing).Error; err == nil {
		return errors.New("同一主题库下接口英文名称已存在")
	}

	return s.db.Create(thematicInterface).Error
}

// GetThematicInterface 根据ID获取主题接口详情
func (s *Service) GetThematicInterface(id string) (*models.ThematicInterface, error) {
	var thematicInterface models.ThematicInterface
	err := s.db.Preload("ThematicLibrary").First(&thematicInterface, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &thematicInterface, nil
}

// GetThematicInterfaceWithSync 获取主题接口详情并同步字段配置
func (s *Service) GetThematicInterfaceWithSync(id string) (*models.ThematicInterface, error) {
	// 先获取接口基本信息
	interfaceData, err := s.GetThematicInterface(id)
	if err != nil {
		return nil, err
	}

	// 如果表已创建，检查并同步字段配置
	if interfaceData.IsTableCreated || interfaceData.IsViewCreated {
		synced, err := s.syncTableFieldsConfig(interfaceData)
		if err != nil {
			slog.Warn("同步表字段配置失败", "interface_id", id, "error", err)
			// 同步失败不影响返回，只记录警告
		} else if synced {
			// 如果有同步，重新加载接口数据
			interfaceData, err = s.GetThematicInterface(id)
			if err != nil {
				return nil, err
			}
		}
	}

	return interfaceData, nil
}

// syncTableFieldsConfig 同步表字段配置
func (s *Service) syncTableFieldsConfig(interfaceData *models.ThematicInterface) (bool, error) {
	schemaName := interfaceData.ThematicLibrary.NameEn
	tableName := interfaceData.NameEn

	// 检查表或视图是否存在
	var exists bool
	var err error
	if interfaceData.Type == "view" {
		exists, err = s.schemaService.CheckViewExists(schemaName, tableName)
	} else {
		exists, err = s.schemaService.CheckTableExists(schemaName, tableName)
	}
	if err != nil {
		return false, fmt.Errorf("检查表/视图存在性失败: %w", err)
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

	slog.Info("检测到主题接口表字段配置不一致，开始同步",
		"interface_id", interfaceData.ID,
		"schema", schemaName,
		"table", tableName,
		"type", interfaceData.Type,
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

	if err := s.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceData.ID).Updates(updates).Error; err != nil {
		return false, fmt.Errorf("更新字段配置失败: %w", err)
	}

	slog.Info("主题接口表字段配置同步完成", "interface_id", interfaceData.ID, "fields_count", len(mergedFields))

	return true, nil
}

// extractFieldsFromConfig 从配置中提取字段
func (s *Service) extractFieldsFromConfig(configJSON models.JSONB) []models.TableField {
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
func (s *Service) mergeFieldsWithConfig(actualColumns []database.ColumnDefinition, existingFieldMap map[string]models.TableField) []models.TableField {
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

	// 按 OrderNum 排序，如果 OrderNum 相同或为0，则按字段名排序
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
func (s *Service) convertColumnsToTableFields(columns []database.ColumnDefinition) []models.TableField {
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
func (s *Service) extractChineseFromComment(comment string) string {
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
func (s *Service) normalizeDataType(dataType string) string {
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
func (s *Service) isFieldsConfigSame(configJSON models.JSONB, actualFields []models.TableField) bool {
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

// GetThematicInterfaces 获取主题接口列表
func (s *Service) GetThematicInterfaces(page, pageSize int, libraryID, interfaceType, status string) ([]models.ThematicInterface, int64, error) {
	var interfaces []models.ThematicInterface
	var total int64

	query := s.db.Model(&models.ThematicInterface{})

	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}
	if interfaceType != "" {
		query = query.Where("type = ?", interfaceType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，预加载主题库信息
	offset := (page - 1) * pageSize
	err := query.Preload("ThematicLibrary").Offset(offset).Limit(pageSize).Find(&interfaces).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取主题接口列表失败: %w", err)
	}

	return interfaces, total, nil
}

// UpdateThematicInterface 更新主题接口
func (s *Service) UpdateThematicInterface(id string, updates *models.ThematicInterface) error {
	// 检查接口是否存在
	var existing models.ThematicInterface
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		return errors.New("主题接口不存在")
	}

	// 如果更新英文名称，需要验证格式和唯一性
	if updates.NameEn != "" && updates.NameEn != existing.NameEn {
		if !isValidSchemaName(updates.NameEn) {
			return errors.New("接口英文名称格式不符合数据库表名命名规范")
		}

		// 检查同一主题库下接口英文名称是否重复
		var duplicate models.ThematicInterface
		if err := s.db.Where("library_id = ? AND name_en = ? AND id != ?", existing.LibraryID, updates.NameEn, id).First(&duplicate).Error; err == nil {
			return errors.New("同一主题库下接口英文名称已存在")
		}
	}

	// 如果更新接口类型，需要验证
	if updates.Type != "" {
		validTypes := []string{"realtime", "batch", "view"}
		if !contains(validTypes, updates.Type) {
			return errors.New("无效的接口类型")
		}
	}

	return s.db.Model(&models.ThematicInterface{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteThematicInterface 删除主题接口
func (s *Service) DeleteThematicInterface(id string) error {
	// 检查接口是否存在
	var existing models.ThematicInterface
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		return errors.New("主题接口不存在")
	}

	// 开启事务删除接口和相关记录
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否有关联的数据流程图
		var flowGraphCount int64
		tx.Model(&models.DataFlowGraph{}).Where("thematic_interface_id = ? AND status != 'inactive'", id).Count(&flowGraphCount)
		if flowGraphCount > 0 {
			return errors.New("存在关联的数据流程图，无法删除接口")
		}

		// 检查是否有关联的API接口
		var apiInterfaceCount int64
		tx.Model(&models.ApiInterface{}).Where("thematic_interface_id = ?", id).Count(&apiInterfaceCount)
		if apiInterfaceCount > 0 {
			return errors.New("存在关联的API接口，无法删除主题接口")
		}

		// 检查是否有关联的主题同步任务
		var syncTaskCount int64
		tx.Model(&models.ThematicSyncTask{}).Where("thematic_interface_id = ?", id).Count(&syncTaskCount)
		if syncTaskCount > 0 {
			return errors.New("存在关联的主题同步任务，无法删除接口")
		}

		// 删除主题接口
		if err := tx.Delete(&models.ThematicInterface{}, "id = ?", id).Error; err != nil {
			return fmt.Errorf("删除主题接口失败: %w", err)
		}

		return nil
	})
}

// contains 检查字符串切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isValidSchemaName 验证schema名称是否符合数据库命名规范
func isValidSchemaName(name string) bool {
	// 基本验证逻辑
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	// 检查首字符是否为字母
	if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) {
		return false
	}

	// 检查其他字符是否为字母、数字或下划线
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// UpdateThematicInterfaceFields 更新主题接口字段配置
func (s *Service) UpdateThematicInterfaceFields(interfaceID string, fields []models.TableField) error {
	// 获取主题接口信息
	interfaceData, err := s.GetThematicInterface(interfaceID)
	if err != nil {
		return fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 获取主题库信息
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", interfaceData.LibraryID).Error; err != nil {
		return fmt.Errorf("获取主题库信息失败: %w", err)
	}

	schemaName := library.NameEn
	tableName := interfaceData.NameEn

	// 转换字段为JSONB格式
	fieldsData := make(models.JSONB)
	for i, field := range fields {
		fieldsData[fmt.Sprintf("field_%d", i)] = field
	}

	// 更新主题接口字段配置
	updates := map[string]interface{}{
		"table_fields_config": fieldsData,
	}

	if err := s.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新主题接口字段配置失败: %w", err)
	}

	// 检查表是否存在
	tableExists, err := s.schemaService.CheckTableExists(schemaName, tableName)
	if err != nil {
		return fmt.Errorf("检查表存在性失败: %w", err)
	}

	// 根据表是否存在决定操作类型
	var operation string
	if tableExists {
		operation = "alter_table"
	} else {
		operation = "create_table"
	}

	// 调用SchemaService管理表结构
	err = s.schemaService.ManageTableSchema(interfaceID, operation, schemaName, tableName, fields)
	if err != nil {
		return fmt.Errorf("管理表结构失败: %w", err)
	}
	updates["is_table_created"] = true
	if err := s.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新主题接口表创建状态失败: %w", err)
	}
	return nil
}

// CreateThematicInterfaceView 创建主题接口视图
func (s *Service) CreateThematicInterfaceView(interfaceID, viewSQL string) error {
	// 获取主题接口信息
	interfaceData, err := s.GetThematicInterface(interfaceID)
	if err != nil {
		return fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证接口类型是否为视图类型
	if interfaceData.Type != "view" {
		return errors.New("只有视图类型的接口才能创建视图")
	}

	// 获取主题库信息
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", interfaceData.LibraryID).Error; err != nil {
		return fmt.Errorf("获取主题库信息失败: %w", err)
	}

	schemaName := library.NameEn
	viewName := interfaceData.NameEn

	// 检查视图是否已存在
	viewExists, err := s.schemaService.CheckViewExists(schemaName, viewName)
	if err != nil {
		return fmt.Errorf("检查视图存在性失败: %w", err)
	}

	// 根据视图是否存在决定操作类型
	var operation string
	if viewExists {
		operation = "update_view"
	} else {
		operation = "create_view"
	}

	// 调用SchemaService管理视图结构
	err = s.schemaService.ManageViewSchema(interfaceID, operation, schemaName, viewName, viewSQL)
	if err != nil {
		return fmt.Errorf("管理视图结构失败: %w", err)
	}

	// 更新主题接口的视图配置
	updates := map[string]interface{}{
		"view_sql":        viewSQL,
		"is_view_created": true,
		"view_config": map[string]interface{}{
			"created_at":   fmt.Sprintf("%d", time.Now().Unix()),
			"last_updated": fmt.Sprintf("%d", time.Now().Unix()),
			"operation":    operation,
		},
	}

	if err := s.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新主题接口视图配置失败: %w", err)
	}

	return nil
}

// UpdateThematicInterfaceView 更新主题接口视图
func (s *Service) UpdateThematicInterfaceView(interfaceID, viewSQL string) error {
	// 获取主题接口信息
	interfaceData, err := s.GetThematicInterface(interfaceID)
	if err != nil {
		return fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证接口类型是否为视图类型
	if interfaceData.Type != "view" {
		return errors.New("只有视图类型的接口才能更新视图")
	}

	// 获取主题库信息
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", interfaceData.LibraryID).Error; err != nil {
		return fmt.Errorf("获取主题库信息失败: %w", err)
	}

	schemaName := library.NameEn
	viewName := interfaceData.NameEn

	// 调用SchemaService更新视图
	err = s.schemaService.ManageViewSchema(interfaceID, "update_view", schemaName, viewName, viewSQL)
	if err != nil {
		return fmt.Errorf("更新视图失败: %w", err)
	}

	// 更新主题接口的视图配置
	updates := map[string]interface{}{
		"view_sql": viewSQL,
		"view_config": map[string]interface{}{
			"last_updated": fmt.Sprintf("%d", time.Now().Unix()),
			"operation":    "update_view",
		},
	}

	if err := s.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新主题接口视图配置失败: %w", err)
	}

	return nil
}

// DeleteThematicInterfaceView 删除主题接口视图
func (s *Service) DeleteThematicInterfaceView(interfaceID string) error {
	// 获取主题接口信息
	interfaceData, err := s.GetThematicInterface(interfaceID)
	if err != nil {
		return fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证接口类型是否为视图类型
	if interfaceData.Type != "view" {
		return errors.New("只有视图类型的接口才能删除视图")
	}

	// 获取主题库信息
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", interfaceData.LibraryID).Error; err != nil {
		return fmt.Errorf("获取主题库信息失败: %w", err)
	}

	schemaName := library.NameEn
	viewName := interfaceData.NameEn

	// 调用SchemaService删除视图
	err = s.schemaService.ManageViewSchema(interfaceID, "drop_view", schemaName, viewName, "")
	if err != nil {
		return fmt.Errorf("删除视图失败: %w", err)
	}

	// 更新主题接口的视图配置
	updates := map[string]interface{}{
		"view_sql":        "",
		"is_view_created": false,
		"view_config":     nil,
	}

	if err := s.db.Model(&models.ThematicInterface{}).Where("id = ?", interfaceID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新主题接口视图配置失败: %w", err)
	}

	return nil
}

// GetThematicInterfaceViewSQL 获取主题接口的视图SQL
func (s *Service) GetThematicInterfaceViewSQL(interfaceID string) (string, error) {
	// 获取主题接口信息
	interfaceData, err := s.GetThematicInterface(interfaceID)
	if err != nil {
		return "", fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证接口类型是否为视图类型
	if interfaceData.Type != "view" {
		return "", errors.New("只有视图类型的接口才有视图SQL")
	}

	return interfaceData.ViewSQL, nil
}
