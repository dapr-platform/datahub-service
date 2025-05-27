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

package service

import (
	"datahub-service/service/models"
	"errors"

	"gorm.io/gorm"
)

// ThematicLibraryService 数据主题库服务
type ThematicLibraryService struct {
	db *gorm.DB
}

// NewThematicLibraryService 创建数据主题库服务实例
func NewThematicLibraryService() *ThematicLibraryService {
	return &ThematicLibraryService{db: DB}
}

// CreateThematicLibrary 创建数据主题库
func (s *ThematicLibraryService) CreateThematicLibrary(library *models.ThematicLibrary) error {
	// 检查编码是否已存在
	var existing models.ThematicLibrary
	if err := s.db.Where("code = ?", library.Code).First(&existing).Error; err == nil {
		return errors.New("主题库编码已存在")
	}

	// 验证编码格式（数据库schema命名规范）
	if !isValidSchemaName(library.Code) {
		return errors.New("主题库编码格式不符合数据库schema命名规范")
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

	return s.db.Create(library).Error
}

// GetThematicLibrary 根据ID获取数据主题库
func (s *ThematicLibraryService) GetThematicLibrary(id string) (*models.ThematicLibrary, error) {
	var library models.ThematicLibrary
	err := s.db.Preload("Interfaces").First(&library, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &library, nil
}

// GetThematicLibraries 获取数据主题库列表
func (s *ThematicLibraryService) GetThematicLibraries(page, pageSize int, category, domain, status string) ([]models.ThematicLibrary, int64, error) {
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

// UpdateThematicLibrary 更新数据主题库
func (s *ThematicLibraryService) UpdateThematicLibrary(id string, updates map[string]interface{}) error {
	// 如果更新编码，需要检查是否重复
	if code, exists := updates["code"]; exists {
		var existing models.ThematicLibrary
		if err := s.db.Where("code = ? AND id != ?", code, id).First(&existing).Error; err == nil {
			return errors.New("主题库编码已存在")
		}

		// 验证编码格式
		if !isValidSchemaName(code.(string)) {
			return errors.New("主题库编码格式不符合数据库schema命名规范")
		}
	}

	// 验证其他字段
	if category, exists := updates["category"]; exists {
		validCategories := []string{"business", "technical", "analysis", "report"}
		if !contains(validCategories, category.(string)) {
			return errors.New("无效的主题分类")
		}
	}

	if domain, exists := updates["domain"]; exists {
		validDomains := []string{"user", "order", "product", "finance", "marketing", "asset", "supply_chain", "park_operation", "park_management", "emergency_safety", "energy", "environment", "security", "service"}
		if !contains(validDomains, domain.(string)) {
			return errors.New("无效的数据域")
		}
	}

	return s.db.Model(&models.ThematicLibrary{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteThematicLibrary 删除数据主题库（软删除，更新状态为archived）
func (s *ThematicLibraryService) DeleteThematicLibrary(id string) error {
	// 检查是否有关联的接口
	var interfaceCount int64
	if err := s.db.Model(&models.ThematicInterface{}).Where("library_id = ? AND status != 'archived'", id).Count(&interfaceCount).Error; err != nil {
		return err
	}

	if interfaceCount > 0 {
		return errors.New("存在关联的主题接口，无法删除")
	}

	return s.db.Model(&models.ThematicLibrary{}).Where("id = ?", id).Update("status", "archived").Error
}

// CreateThematicInterface 创建主题库接口
func (s *ThematicLibraryService) CreateThematicInterface(interfaceData *models.ThematicInterface) error {
	// 检查主题库是否存在
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", interfaceData.LibraryID).Error; err != nil {
		return errors.New("关联的数据主题库不存在")
	}

	// 检查接口英文名称在同一主题库中是否重复
	var existing models.ThematicInterface
	if err := s.db.Where("library_id = ? AND name_en = ?", interfaceData.LibraryID, interfaceData.NameEn).First(&existing).Error; err == nil {
		return errors.New("接口英文名称在该主题库中已存在")
	}

	// 验证接口类型
	if interfaceData.Type != "realtime" && interfaceData.Type != "http" {
		return errors.New("接口类型必须是realtime或http")
	}

	return s.db.Create(interfaceData).Error
}

// GetThematicInterface 获取主题库接口详情
func (s *ThematicLibraryService) GetThematicInterface(id string) (*models.ThematicInterface, error) {
	var interfaceData models.ThematicInterface
	err := s.db.Preload("ThematicLibrary").
		Preload("FlowGraphs").
		Preload("FlowGraphs.Nodes").
		First(&interfaceData, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &interfaceData, nil
}

// GetThematicInterfaces 获取主题库接口列表
func (s *ThematicLibraryService) GetThematicInterfaces(libraryID string, page, pageSize int) ([]models.ThematicInterface, int64, error) {
	var interfaces []models.ThematicInterface
	var total int64

	query := s.db.Model(&models.ThematicInterface{})
	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("ThematicLibrary").Offset(offset).Limit(pageSize).Find(&interfaces).Error

	return interfaces, total, err
}

// CreateDataFlowGraph 创建数据流程图
func (s *ThematicLibraryService) CreateDataFlowGraph(flowGraph *models.DataFlowGraph) error {
	// 检查主题接口是否存在
	var thematicInterface models.ThematicInterface
	if err := s.db.First(&thematicInterface, "id = ?", flowGraph.ThematicInterfaceID).Error; err != nil {
		return errors.New("关联的主题接口不存在")
	}

	// 验证流程图定义
	if flowGraph.Definition == nil {
		return errors.New("流程图定义不能为空")
	}

	return s.db.Create(flowGraph).Error
}

// GetDataFlowGraph 获取数据流程图详情
func (s *ThematicLibraryService) GetDataFlowGraph(id string) (*models.DataFlowGraph, error) {
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
func (s *ThematicLibraryService) UpdateDataFlowGraph(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataFlowGraph{}).Where("id = ?", id).Updates(updates).Error
}

// PublishThematicLibrary 发布主题库
func (s *ThematicLibraryService) PublishThematicLibrary(id string) error {
	// 检查主题库是否存在
	var library models.ThematicLibrary
	if err := s.db.First(&library, "id = ?", id).Error; err != nil {
		return errors.New("主题库不存在")
	}

	// 检查是否有可用的接口
	var interfaceCount int64
	if err := s.db.Model(&models.ThematicInterface{}).Where("library_id = ? AND status = 'active'", id).Count(&interfaceCount).Error; err != nil {
		return err
	}

	if interfaceCount == 0 {
		return errors.New("主题库必须至少包含一个活跃的接口才能发布")
	}

	// 更新发布状态
	return s.db.Model(&models.ThematicLibrary{}).Where("id = ?", id).Update("publish_status", "published").Error
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
