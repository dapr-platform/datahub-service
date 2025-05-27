/*
 * @module service/basic_library_service
 * @description 数据基础库业务逻辑服务，提供基础库的CRUD操作和业务处理
 * @architecture 分层架构 - 业务服务层
 * @documentReference dev_docs/requirements.md
 * @stateFlow 数据基础库管理流程
 * @rules 确保数据完整性，支持事务操作
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs dev_docs/model.md
 */

package service

import (
	"datahub-service/service/models"
	"errors"

	"gorm.io/gorm"
)

// BasicLibraryService 数据基础库服务
type BasicLibraryService struct {
	db *gorm.DB
}

// NewBasicLibraryService 创建数据基础库服务实例
func NewBasicLibraryService() *BasicLibraryService {
	return &BasicLibraryService{db: DB}
}

// CreateBasicLibrary 创建数据基础库
func (s *BasicLibraryService) CreateBasicLibrary(library *models.BasicLibrary) error {
	// 检查英文名称是否已存在
	var existing models.BasicLibrary
	if err := s.db.Where("name_en = ?", library.NameEn).First(&existing).Error; err == nil {
		return errors.New("英文名称已存在")
	}

	// 验证英文名称格式（数据库schema命名规范）
	if !isValidSchemaName(library.NameEn) {
		return errors.New("英文名称格式不符合数据库schema命名规范")
	}

	return s.db.Create(library).Error
}

// GetBasicLibrary 根据ID获取数据基础库
func (s *BasicLibraryService) GetBasicLibrary(id string) (*models.BasicLibrary, error) {
	var library models.BasicLibrary
	err := s.db.Preload("Interfaces").First(&library, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &library, nil
}

// GetBasicLibraries 获取数据基础库列表
func (s *BasicLibraryService) GetBasicLibraries(page, pageSize int, status string) ([]models.BasicLibrary, int64, error) {
	var libraries []models.BasicLibrary
	var total int64

	query := s.db.Model(&models.BasicLibrary{})

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

// UpdateBasicLibrary 更新数据基础库
func (s *BasicLibraryService) UpdateBasicLibrary(id string, updates map[string]interface{}) error {
	// 如果更新英文名称，需要检查是否重复
	if nameEn, exists := updates["name_en"]; exists {
		var existing models.BasicLibrary
		if err := s.db.Where("name_en = ? AND id != ?", nameEn, id).First(&existing).Error; err == nil {
			return errors.New("英文名称已存在")
		}

		// 验证英文名称格式
		if !isValidSchemaName(nameEn.(string)) {
			return errors.New("英文名称格式不符合数据库schema命名规范")
		}
	}

	return s.db.Model(&models.BasicLibrary{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteBasicLibrary 删除数据基础库（软删除，更新状态为archived）
func (s *BasicLibraryService) DeleteBasicLibrary(id string) error {
	// 检查是否有关联的接口
	var interfaceCount int64
	if err := s.db.Model(&models.DataInterface{}).Where("library_id = ? AND status != 'archived'", id).Count(&interfaceCount).Error; err != nil {
		return err
	}

	if interfaceCount > 0 {
		return errors.New("存在关联的数据接口，无法删除")
	}

	return s.db.Model(&models.BasicLibrary{}).Where("id = ?", id).Update("status", "archived").Error
}

// CreateDataInterface 创建数据接口
func (s *BasicLibraryService) CreateDataInterface(interfaceData *models.DataInterface) error {
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

	// 验证接口类型
	if interfaceData.Type != "realtime" && interfaceData.Type != "batch" {
		return errors.New("接口类型必须是realtime或batch")
	}

	return s.db.Create(interfaceData).Error
}

// GetDataInterface 获取数据接口详情
func (s *BasicLibraryService) GetDataInterface(id string) (*models.DataInterface, error) {
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

// GetDataInterfaces 获取数据接口列表
func (s *BasicLibraryService) GetDataInterfaces(libraryID string, page, pageSize int) ([]models.DataInterface, int64, error) {
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

// isValidSchemaName 验证数据库schema名称格式
func isValidSchemaName(name string) bool {
	// 简单的验证规则：只允许字母、数字和下划线，且以字母开头
	if len(name) == 0 {
		return false
	}

	// 首字符必须是字母
	if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) {
		return false
	}

	// 其他字符只能是字母、数字或下划线
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}
