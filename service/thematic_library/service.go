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
	"errors"

	"gorm.io/gorm"
)

// ThematicLibraryService 数据主题库服务
type Service struct {
	db *gorm.DB
}

// NewThematicLibraryService 创建数据主题库服务实例
func NewService(db *gorm.DB) *Service {
	service := &Service{db: db}
	return service
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
		return nil, err
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

	return interfaces, total, err
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
		validTypes := []string{"realtime", "batch"}
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

	// 检查是否有关联的数据流程图
	var flowGraphCount int64
	s.db.Model(&models.DataFlowGraph{}).Where("thematic_interface_id = ? AND status != 'inactive'", id).Count(&flowGraphCount)
	if flowGraphCount > 0 {
		return errors.New("存在关联的数据流程图，无法删除接口")
	}
	return s.db.Delete(&models.ThematicInterface{}, "id = ?", id).Error
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
