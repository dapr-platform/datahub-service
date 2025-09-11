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

package service

import (
	"crypto/rand"
	"datahub-service/service/models"
	"encoding/hex"
	"errors"
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
func (s *SharingService) CreateApiApplication(app *models.ApiApplication, appSecret string) error {
	// 生成应用密钥
	if app.AppKey == "" {
		appKey, err := generateRandomString(32)
		if err != nil {
			return err
		}
		app.AppKey = appKey
	}

	// 加密应用密钥
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(appSecret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	app.AppSecretHash = string(hashedSecret)

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
	if err := s.db.First(&app, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// GetApiApplicationByAppKey 根据AppKey获取API应用
func (s *SharingService) GetApiApplicationByAppKey(appKey string) (*models.ApiApplication, error) {
	var app models.ApiApplication
	if err := s.db.First(&app, "app_key = ?", appKey).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// UpdateApiApplication 更新API应用
func (s *SharingService) UpdateApiApplication(id string, updates map[string]interface{}) error {
	return s.db.Model(&models.ApiApplication{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteApiApplication 删除API应用
func (s *SharingService) DeleteApiApplication(id string) error {
	return s.db.Delete(&models.ApiApplication{}, "id = ?", id).Error
}

// VerifyApiApplication 验证API应用
func (s *SharingService) VerifyApiApplication(appKey, appSecret string) (*models.ApiApplication, error) {
	app, err := s.GetApiApplicationByAppKey(appKey)
	if err != nil {
		return nil, err
	}

	if app.Status != "active" {
		return nil, errors.New("应用已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(app.AppSecretHash), []byte(appSecret)); err != nil {
		return nil, errors.New("应用密钥验证失败")
	}

	return app, nil
}

// === API限流管理 ===

// CreateApiRateLimit 创建API限流规则
func (s *SharingService) CreateApiRateLimit(limit *models.ApiRateLimit) error {
	return s.db.Create(limit).Error
}

// GetApiRateLimits 获取API限流规则列表
func (s *SharingService) GetApiRateLimits(page, pageSize int, applicationID string) ([]models.ApiRateLimit, int64, error) {
	var limits []models.ApiRateLimit
	var total int64

	query := s.db.Model(&models.ApiRateLimit{}).Preload("Application")

	if applicationID != "" {
		query = query.Where("application_id = ?", applicationID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&limits).Error; err != nil {
		return nil, 0, err
	}

	return limits, total, nil
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

	query := s.db.Model(&models.DataAccessRequest{}).Preload("Requester").Preload("Approver")

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
	if err := s.db.Preload("Requester").Preload("Approver").First(&request, "id = ?", id).Error; err != nil {
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

	query := s.db.Model(&models.ApiUsageLog{}).Preload("Application").Preload("User")

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

// === 工具函数 ===

// generateRandomString 生成随机字符串
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
