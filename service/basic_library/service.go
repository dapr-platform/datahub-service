/*
 * @module service/basic_library/service
 * @description 数据基础库主服务，提供基础库的核心业务逻辑
 * @architecture 分层架构 - 业务服务层
 * @documentReference dev_docs/requirements.md, ai_docs/backend_api_analysis.md
 * @stateFlow 数据基础库管理流程
 * @rules 确保数据完整性，支持事务操作
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs dev_docs/model.md
 */

package basic_library

import (
	"datahub-service/service/database"
	"datahub-service/service/models"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Service 数据基础库服务
type Service struct {
	db                *gorm.DB
	datasourceService *DatasourceService
	interfaceService  *InterfaceService
	scheduleService   *ScheduleService
	statusService     *StatusService
}

// NewService 创建数据基础库服务实例
func NewService(db *gorm.DB, eventListener models.EventListener) *Service {
	serviceInstance := &Service{
		db: db,
	}

	// 初始化子服务
	serviceInstance.datasourceService = NewDatasourceService(db)
	serviceInstance.interfaceService = NewInterfaceService(db)
	serviceInstance.scheduleService = NewScheduleService(db)
	serviceInstance.statusService = NewStatusService(db)

	// 如果提供了事件处理器，则注册DB事件处理器,不使用事件通知方式。代码保留备查
	// eventListener.RegisterDBEventProcessor(serviceInstance)

	return serviceInstance
}

// ProcessDBChangeEvent 处理数据库变更事件
func (s *Service) ProcessDBChangeEvent(changeData map[string]interface{}) error {
	log.Println("ProcessDBChangeEvent", changeData)
	// 根据事件类型获取 schema 名称
	switch changeData["type"] {
	case "INSERT":
		// 从新数据中获取名称
		if newData, ok := changeData["new_data"].(map[string]interface{}); ok {
			if nameEn, exists := newData["name_en"]; exists {
				schemaName := nameEn.(string)
				err := database.CreateSchema(s.db, schemaName)
				if err != nil {
					log.Println("CreateSchema error", err)
					return err
				}
			} else if code, exists := newData["code"]; exists {
				schemaName := code.(string)
				err := database.CreateSchema(s.db, schemaName)
				if err != nil {
					log.Println("CreateSchema error", err)
					return err
				}
			}
		}

	case "DELETE":
		// 从旧数据中获取名称
		if oldData, ok := changeData["old_data"].(map[string]interface{}); ok {
			if nameEn, exists := oldData["name_en"]; exists {
				schemaName := nameEn.(string)
				err := database.DeleteSchema(s.db, schemaName)
				if err != nil {
					log.Println("DeleteSchema error", err)
					return err
				}
			} else if code, exists := oldData["code"]; exists {
				schemaName := code.(string)
				err := database.DeleteSchema(s.db, schemaName)
				if err != nil {
					log.Println("DeleteSchema error", err)
					return err
				}
			}
		}
	}
	return nil
}

// TableName 返回表名
func (s *Service) TableName() string {
	return "basic_libraries"
}

// === 基础CRUD操作 ===

// CreateBasicLibrary 创建数据基础库
func (s *Service) CreateBasicLibrary(library *models.BasicLibrary) error {
	// 检查英文名称是否重复
	var existing models.BasicLibrary
	if err := s.db.Where("name_en = ?", library.NameEn).First(&existing).Error; err == nil {
		return errors.New("基础库英文名称已存在")
	}
	if database.CheckSchemaExists(s.db, library.NameEn) {
		return errors.New("基础库英文名称已存在")
	}
	err := database.CreateSchema(s.db, library.NameEn)
	if err != nil {
		return err
	}

	return s.db.Create(library).Error
}

// GetBasicLibrary 获取数据基础库详情
func (s *Service) GetBasicLibrary(id string) (*models.BasicLibrary, error) {
	var library models.BasicLibrary
	err := s.db.Preload("DataSources").Preload("Interfaces").First(&library, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &library, nil
}

// GetBasicLibraries 获取数据基础库列表
func (s *Service) GetBasicLibraries(page, pageSize int) ([]models.BasicLibrary, int64, error) {
	var libraries []models.BasicLibrary
	var total int64

	// 获取总数
	if err := s.db.Model(&models.BasicLibrary{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.Offset(offset).Limit(pageSize).Find(&libraries).Error

	return libraries, total, err
}

// UpdateBasicLibrary 更新数据基础库
func (s *Service) UpdateBasicLibrary(id string, updates map[string]interface{}) error {
	// 检查是否存在
	var library models.BasicLibrary
	if err := s.db.First(&library, "id = ?", id).Error; err != nil {
		return err
	}

	// 如果更新英文名称，检查是否重复
	if nameEn, exists := updates["name_en"]; exists {
		var existing models.BasicLibrary
		if err := s.db.Where("name_en = ? AND id != ?", nameEn, id).First(&existing).Error; err == nil {
			return errors.New("基础库英文名称已存在")
		}
	}

	return s.db.Model(&library).Updates(updates).Error
}

// DeleteBasicLibrary 删除数据基础库
func (s *Service) DeleteBasicLibrary(library *models.BasicLibrary) error {
	// 检查是否存在关联的数据源或接口
	var dataSourceCount, interfaceCount int64

	s.db.Model(&models.DataSource{}).Where("library_id = ?", library.ID).Count(&dataSourceCount)
	s.db.Model(&models.DataInterface{}).Where("library_id = ?", library.ID).Count(&interfaceCount)

	if dataSourceCount > 0 || interfaceCount > 0 {
		return errors.New("无法删除：存在关联的数据源或接口")
	}
	err := database.DeleteSchema(s.db, library.NameEn)
	if err != nil {
		return fmt.Errorf("删除数据基础库schema失败: %v", err)
	}

	return s.db.Delete(&models.BasicLibrary{}, "id = ?", library.ID).Error
}

// === 数据接口操作 ===

// CreateDataInterface 创建数据接口
func (s *Service) CreateDataInterface(interfaceData *models.DataInterface) error {
	return s.interfaceService.CreateDataInterface(interfaceData)
}

// DeleteDataInterface 删除数据接口
func (s *Service) DeleteDataInterface(interfaceData *models.DataInterface) error {
	return s.interfaceService.DeleteDataInterface(interfaceData)
}

// GetDataInterface 获取数据接口详情
func (s *Service) GetDataInterface(id string) (*models.DataInterface, error) {
	return s.interfaceService.GetDataInterface(id)
}

// GetDataInterfaces 获取数据接口列表
func (s *Service) GetDataInterfaces(libraryID string, page, pageSize int) ([]models.DataInterface, int64, error) {
	return s.interfaceService.GetDataInterfaces(libraryID, page, pageSize)
}

// === 控制器所需的接口方法 ===

// TestDataSource 测试数据源
func (s *Service) TestDataSource(dataSourceID, testType string, config map[string]interface{}) (*DataSourceTestResult, error) {
	return s.datasourceService.TestDataSource(dataSourceID, testType, config)
}

// TestInterface 测试接口
func (s *Service) TestInterface(interfaceID, testType string, parameters, options map[string]interface{}) (*InterfaceTestResult, error) {
	return s.interfaceService.TestInterface(interfaceID, testType, parameters, options)
}

// ConfigureSchedule 配置调度
func (s *Service) ConfigureSchedule(scheduleConfig *models.ScheduleConfig) error {
	_, err := s.scheduleService.ConfigureSchedule(scheduleConfig)
	return err
}

// GetDataSourceStatus 获取数据源状态
func (s *Service) GetDataSourceStatus(id string) (*models.DataSourceStatus, error) {
	return s.statusService.GetDataSourceStatus(id)
}

// PreviewInterfaceData 预览接口数据
func (s *Service) PreviewInterfaceData(id string, limit int) (interface{}, error) {
	return s.interfaceService.PreviewInterfaceData(id, limit)
}

// === 工具函数 ===

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

// GetDatasourceService 获取数据源服务
func (s *Service) GetDatasourceService() *DatasourceService {
	return s.datasourceService
}

// GetInterfaceService 获取接口服务
func (s *Service) GetInterfaceService() *InterfaceService {
	return s.interfaceService
}

// GetScheduleService 获取调度服务
func (s *Service) GetScheduleService() *ScheduleService {
	return s.scheduleService
}

// GetStatusService 获取状态服务
func (s *Service) GetStatusService() *StatusService {
	return s.statusService
}

// CreateDataSource 创建数据源
func (s *Service) CreateDataSource(dataSource *models.DataSource) error {
	return s.datasourceService.CreateDataSource(dataSource)
}

// UpdateDataSource 更新数据源
func (s *Service) UpdateDataSource(id string, updates map[string]interface{}) error {
	return s.datasourceService.UpdateDataSource(id, updates)
}

// DeleteDataSource 删除数据源
func (s *Service) DeleteDataSource(dataSource *models.DataSource) error {
	return s.datasourceService.DeleteDataSource(dataSource)
}
