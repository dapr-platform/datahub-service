/*
 * @module service/sharing_service
 * @description 数据共享服务，提供API应用管理、数据订阅、数据使用申请等功能
 * @architecture 分层架构 - 业务服务层
 * @documentReference ai_docs/requirements.md
 * @stateFlow 数据共享服务生命周期管理
 * @rules 确保数据安全共享和访问控制
 * @dependencies datahub-service/service/models, gorm.io/gorm, golang.org/x/crypto/bcrypt
 * @refs ai_docs/model.md
 */

package sharing

import (
	"crypto/rand"
	"datahub-service/service/database"
	"datahub-service/service/models"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SharingService 数据共享服务
type SharingService struct {
	db *gorm.DB
}

// NewSharingService 创建数据共享服务实例
func NewSharingService(db *gorm.DB) *SharingService {
	return &SharingService{db: db}
}

// === API应用管理 ===

// CreateApiApplication 创建API应用
func (s *SharingService) CreateApiApplication(app *models.ApiApplication) error {
	// 验证主题库是否存在
	var thematicLibrary models.ThematicLibrary
	if err := s.db.First(&thematicLibrary, "id = ?", app.ThematicLibraryID).Error; err != nil {
		return errors.New("主题库不存在")
	}

	// 验证应用路径唯一性
	var count int64
	if err := s.db.Model(&models.ApiApplication{}).Where("path = ?", app.Path).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("应用路径已存在")
	}

	return s.db.Create(app).Error
}

// GetApiApplications 获取API应用列表
func (s *SharingService) GetApiApplications(page, pageSize int, status string) ([]models.ApiApplication, int64, error) {
	var apps []models.ApiApplication
	var total int64

	query := s.db.Model(&models.ApiApplication{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}

// GetApiApplicationByID 根据ID获取API应用
func (s *SharingService) GetApiApplicationByID(id string) (*models.ApiApplication, error) {
	var app models.ApiApplication
	if err := s.db.Preload("ThematicLibrary").Preload("ApiInterfaces").Preload("ApiInterfaces.ThematicInterface").First(&app, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// GetApiApplicationByPath 根据路径获取API应用及其接口信息（包括主题接口字段定义）
func (s *SharingService) GetApiApplicationByPath(path string) (*models.ApiApplication, error) {
	var app models.ApiApplication

	// 预加载所有相关信息：主题库、API接口、主题接口（包含字段配置）
	err := s.db.
		Preload("ThematicLibrary").
		Preload("ApiInterfaces", "status = 'active'").
		Preload("ApiInterfaces.ThematicInterface").
		Where("path = ? AND status = 'active'", path).
		First(&app).Error

	if err != nil {
		return nil, err
	}

	return &app, nil
}

// GetApiApplicationsByApiKey 根据API Key ID获取该Key可访问的所有API应用及其接口信息
func (s *SharingService) GetApiApplicationsByApiKey(apiKeyID string) ([]models.ApiApplication, error) {
	var apps []models.ApiApplication

	// 通过API Key ID和关联表查找对应的应用，并预加载所有相关信息
	err := s.db.
		Joins("JOIN api_key_applications ON api_applications.id = api_key_applications.api_application_id").
		Joins("JOIN api_keys ON api_key_applications.api_key_id = api_keys.id").
		Where("api_keys.id = ? AND api_keys.status = 'active' AND api_applications.status = 'active'", apiKeyID).
		Preload("ThematicLibrary").
		Preload("ApiInterfaces", "status = 'active'").
		Preload("ApiInterfaces.ThematicInterface").
		Find(&apps).Error

	if err != nil {
		return nil, err
	}

	return apps, nil
}

// GetApiApplicationByApiKeyAndPath 根据API Key ID和应用路径获取特定的API应用
func (s *SharingService) GetApiApplicationByApiKeyAndPath(apiKeyID, appPath string) (*models.ApiApplication, error) {
	var app models.ApiApplication

	// 通过API Key ID和应用路径查找对应的应用
	err := s.db.
		Joins("JOIN api_key_applications ON api_applications.id = api_key_applications.api_application_id").
		Joins("JOIN api_keys ON api_key_applications.api_key_id = api_keys.id").
		Where("api_keys.id = ? AND api_applications.path = ? AND api_keys.status = 'active' AND api_applications.status = 'active'", apiKeyID, appPath).
		Preload("ThematicLibrary").
		Preload("ApiInterfaces", "status = 'active'").
		Preload("ApiInterfaces.ThematicInterface").
		First(&app).Error

	if err != nil {
		return nil, err
	}

	return &app, nil
}

// UpdateApiApplication 更新API应用
func (s *SharingService) UpdateApiApplication(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.ApiApplication{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteApiApplication 删除API应用（级联删除关联数据）
func (s *SharingService) DeleteApiApplication(id string) error {
	// 检查应用是否存在
	var existing models.ApiApplication
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		return errors.New("API应用不存在")
	}

	// 开启事务删除应用和相关记录
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除关联的API接口
		if err := tx.Where("api_application_id = ?", id).Delete(&models.ApiInterface{}).Error; err != nil {
			return fmt.Errorf("删除关联的API接口失败: %w", err)
		}

		// 2. 删除关联的API限流规则
		if err := tx.Where("application_id = ?", id).Delete(&models.ApiRateLimit{}).Error; err != nil {
			return fmt.Errorf("删除关联的API限流规则失败: %w", err)
		}

		// 3. 删除关联的API Key关联关系
		if err := tx.Where("api_application_id = ?", id).Delete(&models.ApiKeyApplication{}).Error; err != nil {
			return fmt.Errorf("删除关联的API密钥关系失败: %w", err)
		}

		// 4. 删除关联的API使用日志
		if err := tx.Model(&models.ApiUsageLog{}).Where("application_id = ?", id).Delete(&models.ApiUsageLog{}).Error; err != nil {
			return fmt.Errorf("删除关联的API使用日志失败: %w", err)
		}

		// 5. 删除API应用
		if err := tx.Delete(&models.ApiApplication{}, "id = ?", id).Error; err != nil {
			return fmt.Errorf("删除API应用失败: %w", err)
		}

		return nil
	})
}

// === ApiKey管理 ===

// CreateApiKey 创建一个新的ApiKey并关联到指定的应用
func (s *SharingService) CreateApiKey(name, description string, appIDs []string, expiresAt *time.Time) (*models.ApiKey, string, error) {
	// 验证应用是否存在
	if len(appIDs) == 0 {
		return nil, "", errors.New("至少需要关联一个应用")
	}

	var apps []models.ApiApplication
	if err := s.db.Where("id IN ?", appIDs).Find(&apps).Error; err != nil {
		return nil, "", err
	}

	if len(apps) != len(appIDs) {
		return nil, "", errors.New("部分应用不存在")
	}

	// 生成API Key
	fullKey, err := generateRandomString(64) // 生成32字节的随机字符串，转为64字符的hex
	if err != nil {
		return nil, "", err
	}

	// 生成前缀（取前8个字符）
	keyPrefix := fullKey[:8]

	// 对完整Key进行哈希
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	apiKey := &models.ApiKey{
		Name:         name,
		KeyPrefix:    keyPrefix,
		KeyValueHash: string(hashedKey),
		Description:  description,
		ExpiresAt:    expiresAt,
		Status:       "active",
	}

	// 开始数据库事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, "", tx.Error
	}

	// 创建API Key记录
	if err := tx.Create(apiKey).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	// 关联应用
	for _, appID := range appIDs {
		keyApp := &models.ApiKeyApplication{
			ApiKeyID:         apiKey.ID,
			ApiApplicationID: appID,
		}
		if err := tx.Create(keyApp).Error; err != nil {
			tx.Rollback()
			return nil, "", err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, "", err
	}

	// 加载关联的应用信息
	if err := s.db.Preload("Applications").First(apiKey, "id = ?", apiKey.ID).Error; err != nil {
		return nil, "", err
	}

	// 返回完整的Key值（仅此一次），数据库存储其Hash
	return apiKey, fullKey, nil
}

// GetApiKeys 获取所有ApiKey信息（不包含Key本身），可选择按应用过滤
func (s *SharingService) GetApiKeys(appID string) ([]models.ApiKey, error) {
	var keys []models.ApiKey
	query := s.db.Preload("Applications")

	if appID != "" {
		// 通过关联表查询特定应用的ApiKey
		query = query.Joins("JOIN api_key_applications ON api_keys.id = api_key_applications.api_key_id").
			Where("api_key_applications.api_application_id = ?", appID)
	}

	if err := query.Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// GetApiKeyByID 根据ID获取ApiKey
func (s *SharingService) GetApiKeyByID(keyID string) (*models.ApiKey, error) {
	var key models.ApiKey
	if err := s.db.Preload("Applications").First(&key, "id = ?", keyID).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// UpdateApiKey 更新ApiKey信息（如描述、状态）
func (s *SharingService) UpdateApiKey(keyID string, updates map[string]interface{}) error {
	return s.db.Model(&models.ApiKey{}).Where("id = ?", keyID).Updates(updates).Error
}

// UpdateApiKeyApplications 更新ApiKey关联的应用
func (s *SharingService) UpdateApiKeyApplications(keyID string, appIDs []string) error {
	// 验证ApiKey是否存在
	var key models.ApiKey
	if err := s.db.First(&key, "id = ?", keyID).Error; err != nil {
		return errors.New("ApiKey不存在")
	}

	// 验证应用是否存在
	if len(appIDs) == 0 {
		return errors.New("至少需要关联一个应用")
	}

	var apps []models.ApiApplication
	if err := s.db.Where("id IN ?", appIDs).Find(&apps).Error; err != nil {
		return err
	}

	if len(apps) != len(appIDs) {
		return errors.New("部分应用不存在")
	}

	// 开始数据库事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 删除现有关联
	if err := tx.Where("api_key_id = ?", keyID).Delete(&models.ApiKeyApplication{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建新关联
	for _, appID := range appIDs {
		keyApp := &models.ApiKeyApplication{
			ApiKeyID:         keyID,
			ApiApplicationID: appID,
		}
		if err := tx.Create(keyApp).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 提交事务
	return tx.Commit().Error
}

// DeleteApiKey 吊销（删除）一个ApiKey
func (s *SharingService) DeleteApiKey(keyID string) error {
	// 开始数据库事务
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 删除关联关系
	if err := tx.Where("api_key_id = ?", keyID).Delete(&models.ApiKeyApplication{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除API Key记录
	if err := tx.Delete(&models.ApiKey{}, "id = ?", keyID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除对应的PostgREST用户（使用keyID作为用户名）
	if err := database.DeletePostgRESTUser(tx, keyID); err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// VerifyApiKey 验证API Key
func (s *SharingService) VerifyApiKey(keyValue string) (*models.ApiKey, error) {
	if len(keyValue) < 8 {
		return nil, errors.New("无效的API Key格式")
	}

	keyPrefix := keyValue[:8]

	var keys []models.ApiKey
	if err := s.db.Where("key_prefix = ? AND status = 'active'", keyPrefix).Find(&keys).Error; err != nil {
		return nil, err
	}

	// 遍历所有匹配前缀的Key，验证完整Key
	for _, key := range keys {
		if err := bcrypt.CompareHashAndPassword([]byte(key.KeyValueHash), []byte(keyValue)); err == nil {
			// 检查是否过期
			if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
				return nil, errors.New("API Key已过期")
			}

			// 更新最后使用时间和使用次数
			s.db.Model(&key).Updates(map[string]interface{}{
				"last_used_at": time.Now(),
				"usage_count":  key.UsageCount + 1,
			})

			return &key, nil
		}
	}

	return nil, errors.New("无效的API Key")
}

// GetPostgRESTTokenByApiKey 通过API Key获取PostgREST Token
func (s *SharingService) GetPostgRESTTokenByApiKey(keyValue string) (string, error) {
	// 首先验证API Key
	apiKey, err := s.VerifyApiKey(keyValue)
	if err != nil {
		return "", err
	}

	// 使用API Key的ID作为用户名和密码调用PostgREST的get_token函数
	userName := apiKey.ID
	password := apiKey.ID

	sql := `SELECT postgrest.get_token($1, $2)`

	var result string
	if err := s.db.Raw(sql, userName, password).Scan(&result).Error; err != nil {
		return "", err
	}

	return result, nil
}

// === ApiInterface管理 ===

// CreateApiInterface 创建一个共享接口
func (s *SharingService) CreateApiInterface(apiInterface *models.ApiInterface) error {
	// 验证应用是否存在
	var app models.ApiApplication
	if err := s.db.First(&app, "id = ?", apiInterface.ApiApplicationID).Error; err != nil {
		return errors.New("应用不存在")
	}

	// 验证主题接口是否存在
	var thematicInterface models.ThematicInterface
	if err := s.db.First(&thematicInterface, "id = ?", apiInterface.ThematicInterfaceID).Error; err != nil {
		return errors.New("主题接口不存在")
	}

	// 验证路径唯一性
	var count int64
	if err := s.db.Model(&models.ApiInterface{}).Where("path = ?", apiInterface.Path).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("接口路径已存在")
	}

	return s.db.Create(apiInterface).Error
}

// GetApiInterfaces 查询共享接口列表，可按 api_application_id 过滤
func (s *SharingService) GetApiInterfaces(appID string) ([]models.ApiInterface, error) {
	var interfaces []models.ApiInterface
	query := s.db.Preload("ApiApplication").Preload("ThematicInterface")

	if appID != "" {
		query = query.Where("api_application_id = ?", appID)
	}

	if err := query.Find(&interfaces).Error; err != nil {
		return nil, err
	}
	return interfaces, nil
}

// GetApiInterfaceByID 根据ID获取ApiInterface
func (s *SharingService) GetApiInterfaceByID(id string) (*models.ApiInterface, error) {
	var apiInterface models.ApiInterface
	if err := s.db.Preload("ApiApplication").Preload("ThematicInterface").First(&apiInterface, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &apiInterface, nil
}

// GetApiInterfaceByAppPathAndInterfacePath 根据应用路径和接口路径获取ApiInterface
func (s *SharingService) GetApiInterfaceByAppPathAndInterfacePath(appPath, interfacePath string) (*models.ApiInterface, error) {
	var apiInterface models.ApiInterface
	if err := s.db.Joins("JOIN api_applications ON api_interfaces.api_application_id = api_applications.id").
		Where("api_applications.path = ? AND api_interfaces.path = ? AND api_interfaces.status = 'active' AND api_applications.status = 'active'", appPath, interfacePath).
		Preload("ApiApplication").Preload("ApiApplication.ThematicLibrary").Preload("ThematicInterface").
		First(&apiInterface).Error; err != nil {
		return nil, err
	}
	return &apiInterface, nil
}

// GetApiInterfaceBySchemaAndPath 根据主题库Schema和路径获取ApiInterface（保留向后兼容）
func (s *SharingService) GetApiInterfaceBySchemaAndPath(schema, path string) (*models.ApiInterface, error) {
	var apiInterface models.ApiInterface
	if err := s.db.Joins("JOIN api_applications ON api_interfaces.api_application_id = api_applications.id").
		Joins("JOIN thematic_libraries ON api_applications.thematic_library_id = thematic_libraries.id").
		Where("thematic_libraries.name_en = ? AND api_interfaces.path = ? AND api_interfaces.status = 'active'", schema, path).
		Preload("ApiApplication").Preload("ApiApplication.ThematicLibrary").Preload("ThematicInterface").
		First(&apiInterface).Error; err != nil {
		return nil, err
	}
	return &apiInterface, nil
}

// DeleteApiInterface 删除一个共享接口
func (s *SharingService) DeleteApiInterface(id string) error {
	return s.db.Delete(&models.ApiInterface{}, "id = ?", id).Error
}

// === API限流管理 ===

// CreateApiRateLimit 创建API限流规则
func (s *SharingService) CreateApiRateLimit(limit *models.ApiRateLimit) error {
	// 验证限流类型
	validTypes := []string{"global", "api_key", "application"}
	isValidType := false
	for _, validType := range validTypes {
		if limit.RateLimitType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的限流类型，必须是global、api_key或application")
	}

	// 如果是全局限流，检查是否已存在全局限流规则
	if limit.RateLimitType == "global" {
		var count int64
		if err := s.db.Model(&models.ApiRateLimit{}).
			Where("rate_limit_type = ? AND is_enabled = true", "global").
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("全局限流规则已存在，一个系统只能有一个全局限流规则")
		}
		// 全局限流不需要TargetID
		limit.TargetID = nil
	}

	// 验证密钥或应用是否存在
	if limit.RateLimitType == "api_key" {
		if limit.TargetID == nil {
			return errors.New("API密钥限流必须指定target_id")
		}
		var key models.ApiKey
		if err := s.db.First(&key, "id = ?", *limit.TargetID).Error; err != nil {
			return errors.New("API密钥不存在")
		}
	} else if limit.RateLimitType == "application" {
		if limit.TargetID == nil {
			return errors.New("应用限流必须指定target_id")
		}
		var app models.ApiApplication
		if err := s.db.First(&app, "id = ?", *limit.TargetID).Error; err != nil {
			return errors.New("应用不存在")
		}
	}

	// 验证时间窗口和最大请求数
	if limit.TimeWindow <= 0 {
		return errors.New("时间窗口必须大于0")
	}
	if limit.MaxRequests <= 0 {
		return errors.New("最大请求数必须大于0")
	}

	return s.db.Create(limit).Error
}

// GetApiRateLimits 获取API限流规则列表
func (s *SharingService) GetApiRateLimits(page, pageSize int, rateLimitType, targetID string) ([]models.ApiRateLimit, int64, error) {
	var limits []models.ApiRateLimit
	var total int64

	query := s.db.Model(&models.ApiRateLimit{}).
		Preload("Application").
		Preload("ApiKey")

	if rateLimitType != "" {
		query = query.Where("rate_limit_type = ?", rateLimitType)
	}

	if targetID != "" {
		query = query.Where("target_id = ?", targetID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&limits).Error; err != nil {
		return nil, 0, err
	}

	return limits, total, nil
}

// GetApplicableRateLimits 获取适用于特定请求的所有限流规则（全局、密钥、应用）
func (s *SharingService) GetApplicableRateLimits(apiKeyID, applicationID string) ([]models.ApiRateLimit, error) {
	var limits []models.ApiRateLimit

	// 查询全局限流
	var globalLimit models.ApiRateLimit
	if err := s.db.Where("rate_limit_type = ? AND is_enabled = true", "global").First(&globalLimit).Error; err == nil {
		limits = append(limits, globalLimit)
	}

	// 查询API密钥限流
	if apiKeyID != "" {
		var keyLimit models.ApiRateLimit
		if err := s.db.Where("rate_limit_type = ? AND target_id = ? AND is_enabled = true", "api_key", apiKeyID).First(&keyLimit).Error; err == nil {
			limits = append(limits, keyLimit)
		}
	}

	// 查询应用限流
	if applicationID != "" {
		var appLimit models.ApiRateLimit
		if err := s.db.Where("rate_limit_type = ? AND target_id = ? AND is_enabled = true", "application", applicationID).First(&appLimit).Error; err == nil {
			limits = append(limits, appLimit)
		}
	}

	return limits, nil
}

// UpdateApiRateLimit 更新API限流规则
func (s *SharingService) UpdateApiRateLimit(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.ApiRateLimit{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteApiRateLimit 删除API限流规则
func (s *SharingService) DeleteApiRateLimit(id string) error {
	return s.db.Delete(&models.ApiRateLimit{}, "id = ?", id).Error
}

// === 数据订阅管理 ===

// CreateDataSubscription 创建数据订阅
func (s *SharingService) CreateDataSubscription(subscription *models.DataSubscription) error {
	// 验证订阅者类型
	validSubscriberTypes := []string{"user", "application"}
	isValidType := false
	for _, validType := range validSubscriberTypes {
		if subscription.SubscriberType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的订阅者类型")
	}

	// 验证资源类型
	validResourceTypes := []string{"thematic_interface", "basic_interface"}
	isValidResourceType := false
	for _, validType := range validResourceTypes {
		if subscription.ResourceType == validType {
			isValidResourceType = true
			break
		}
	}
	if !isValidResourceType {
		return errors.New("无效的资源类型")
	}

	// 验证通知方式
	validMethods := []string{"webhook", "message_queue", "email"}
	isValidMethod := false
	for _, validMethod := range validMethods {
		if subscription.NotificationMethod == validMethod {
			isValidMethod = true
			break
		}
	}
	if !isValidMethod {
		return errors.New("无效的通知方式")
	}

	return s.db.Create(subscription).Error
}

// GetDataSubscriptions 获取数据订阅列表
func (s *SharingService) GetDataSubscriptions(page, pageSize int, subscriberID, resourceType, status string) ([]models.DataSubscription, int64, error) {
	var subscriptions []models.DataSubscription
	var total int64

	query := s.db.Model(&models.DataSubscription{})

	if subscriberID != "" {
		query = query.Where("subscriber_id = ?", subscriberID)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&subscriptions).Error; err != nil {
		return nil, 0, err
	}

	return subscriptions, total, nil
}

// GetDataSubscriptionByID 根据ID获取数据订阅
func (s *SharingService) GetDataSubscriptionByID(id string) (*models.DataSubscription, error) {
	var subscription models.DataSubscription
	if err := s.db.First(&subscription, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// UpdateDataSubscription 更新数据订阅
func (s *SharingService) UpdateDataSubscription(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.DataSubscription{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDataSubscription 删除数据订阅
func (s *SharingService) DeleteDataSubscription(id string) error {
	return s.db.Delete(&models.DataSubscription{}, "id = ?", id).Error
}

// === 数据使用申请管理 ===

// CreateDataAccessRequest 创建数据使用申请
func (s *SharingService) CreateDataAccessRequest(request *models.DataAccessRequest) error {
	// 验证资源类型
	validResourceTypes := []string{"thematic_library", "basic_library", "interface"}
	isValidType := false
	for _, validType := range validResourceTypes {
		if request.ResourceType == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("无效的资源类型")
	}

	// 验证访问权限
	validPermissions := []string{"read", "write"}
	isValidPermission := false
	for _, validPermission := range validPermissions {
		if request.AccessPermission == validPermission {
			isValidPermission = true
			break
		}
	}
	if !isValidPermission {
		return errors.New("无效的访问权限")
	}

	return s.db.Create(request).Error
}

// GetDataAccessRequests 获取数据使用申请列表
func (s *SharingService) GetDataAccessRequests(page, pageSize int, requesterID, resourceType, status string) ([]models.DataAccessRequest, int64, error) {
	var requests []models.DataAccessRequest
	var total int64

	query := s.db.Model(&models.DataAccessRequest{})

	if requesterID != "" {
		query = query.Where("requester_id = ?", requesterID)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("requested_at DESC").Find(&requests).Error; err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// GetDataAccessRequestByID 根据ID获取数据使用申请
func (s *SharingService) GetDataAccessRequestByID(id string) (*models.DataAccessRequest, error) {
	var request models.DataAccessRequest
	if err := s.db.First(&request, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

// ApproveDataAccessRequest 审批数据使用申请
func (s *SharingService) ApproveDataAccessRequest(id, approverID string, approved bool, comment string) error {
	updates := map[string]interface{}{
		"approver_id":      approverID,
		"approval_comment": comment,
		"approved_at":      time.Now(),
	}

	if approved {
		updates["status"] = "approved"
	} else {
		updates["status"] = "rejected"
	}

	return s.db.Model(&models.DataAccessRequest{}).Where("id = ?", id).Updates(updates).Error
}

// === API使用日志管理 ===

// CreateApiUsageLog 创建API使用日志
func (s *SharingService) CreateApiUsageLog(log *models.ApiUsageLog) error {
	return s.db.Create(log).Error
}

// GetApiUsageLogs 获取API使用日志列表
func (s *SharingService) GetApiUsageLogs(page, pageSize int, applicationID, userID string, startTime, endTime *time.Time) ([]models.ApiUsageLog, int64, error) {
	var logs []models.ApiUsageLog
	var total int64

	query := s.db.Model(&models.ApiUsageLog{}).Preload("Application")

	if applicationID != "" {
		query = query.Where("application_id = ?", applicationID)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if startTime != nil {
		query = query.Where("request_time >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("request_time <= ?", endTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("request_time DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetApiRateLimitStatistics 获取API限流统计信息
func (s *SharingService) GetApiRateLimitStatistics() (*models.ApiRateLimitStatistics, error) {
	stats := &models.ApiRateLimitStatistics{}

	// 总限流规则数
	if err := s.db.Model(&models.ApiRateLimit{}).Count(&stats.TotalRules).Error; err != nil {
		return nil, err
	}

	// 启用的限流规则数
	if err := s.db.Model(&models.ApiRateLimit{}).Where("is_enabled = ?", true).Count(&stats.EnabledRules).Error; err != nil {
		return nil, err
	}

	// 按类型统计
	if err := s.db.Model(&models.ApiRateLimit{}).
		Select("rate_limit_type, COUNT(*) as count").
		Where("is_enabled = ?", true).
		Group("rate_limit_type").
		Find(&stats.TypeDistribution).Error; err != nil {
		return nil, err
	}

	// 最近7天创建的规则数
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	if err := s.db.Model(&models.ApiRateLimit{}).
		Where("created_at >= ?", sevenDaysAgo).
		Count(&stats.RecentRules).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetApiUsageStatistics 获取API使用统计信息
func (s *SharingService) GetApiUsageStatistics(startTime, endTime *time.Time) (*models.ApiUsageStatistics, error) {
	stats := &models.ApiUsageStatistics{
		TopApplications:    make([]models.TopApplication, 0),
		StatusDistribution: make([]models.StatusDistribution, 0),
	}

	// 辅助函数：应用时间过滤
	applyTimeFilter := func(query *gorm.DB) *gorm.DB {
		if startTime != nil {
			query = query.Where("request_time >= ?", startTime)
		}
		if endTime != nil {
			query = query.Where("request_time <= ?", endTime)
		}
		return query
	}

	// 总请求数
	query := applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Count(&stats.TotalRequests).Error; err != nil {
		return nil, err
	}

	// 成功请求数（2xx状态码）
	query = applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Where("status_code >= ? AND status_code < ?", 200, 300).
		Count(&stats.SuccessRequests).Error; err != nil {
		return nil, err
	}

	// 失败请求数（4xx和5xx状态码）
	query = applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Where("status_code >= ?", 400).
		Count(&stats.FailedRequests).Error; err != nil {
		return nil, err
	}

	// 限流拒绝数（429状态码）
	query = applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Where("status_code = ?", 429).
		Count(&stats.RateLimitedRequests).Error; err != nil {
		return nil, err
	}

	// 平均响应时间
	var avgResponseTime float64
	query = applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Select("AVG(response_time)").
		Row().Scan(&avgResponseTime); err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	stats.AvgResponseTime = int(avgResponseTime)

	// 按应用统计TOP5
	query = applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Select("api_usage_logs.application_id, api_applications.name as app_name, COUNT(*) as count").
		Joins("LEFT JOIN api_applications ON api_usage_logs.application_id = api_applications.id").
		Where("api_usage_logs.application_id IS NOT NULL").
		Group("api_usage_logs.application_id, api_applications.name").
		Order("count DESC").
		Limit(5).
		Find(&stats.TopApplications).Error; err != nil {
		return nil, err
	}

	// 按状态码统计
	query = applyTimeFilter(s.db.Model(&models.ApiUsageLog{}))
	if err := query.Select("status_code, COUNT(*) as count").
		Group("status_code").
		Order("count DESC").
		Find(&stats.StatusDistribution).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// === 工具函数 ===

// generateRandomString 生成随机字符串
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
