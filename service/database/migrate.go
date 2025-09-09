/*
 * @module service/database/migrate
 * @description 数据库迁移模块，负责创建和更新数据库表结构
 * @architecture 数据访问层 - 迁移管理
 * @documentReference dev_docs/model.md
 * @stateFlow 应用启动时执行数据库迁移
 * @rules 确保数据库结构与模型定义保持一致
 * @dependencies datahub-service/service/models, gorm.io/gorm
 * @refs dev_docs/backend_requirements.md, service/models/datasource_types.go
 */

package database

import (
	"datahub-service/service/models"
	"log"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	log.Println("开始数据库迁移...")

	// 数据基础库相关表
	err := db.AutoMigrate(
		&models.BasicLibrary{},
		&models.DataInterface{},
		&models.DataSource{},
		&models.CleansingRule{},
		&models.DataSourceStatus{},
		&models.InterfaceStatus{},
		&models.SyncTask{},
	)
	if err != nil {
		return err
	}

	// 数据主题库相关表
	err = db.AutoMigrate(
		&models.ThematicLibrary{},
		&models.ThematicInterface{},
		&models.DataFlowGraph{},
		&models.FlowNode{},
	)
	if err != nil {
		return err
	}

	// 访问控制相关表已移除，改为使用PostgREST RBAC

	// 数据治理相关表
	err = db.AutoMigrate(
		&models.QualityRule{},
		&models.Metadata{},
		&models.DataMaskingRule{},
		&models.SystemLog{},
		&models.BackupConfig{},
		&models.BackupRecord{},
		&models.DataQualityReport{},
	)
	if err != nil {
		return err
	}

	// 数据共享服务相关表
	err = db.AutoMigrate(
		&models.ApiApplication{},
		&models.ApiRateLimit{},
		&models.DataSubscription{},
		&models.DataAccessRequest{},
		&models.DataSyncTask{},
		&models.DataSyncLog{},
		&models.ApiUsageLog{},
	)
	if err != nil {
		return err
	}

	// 事件管理相关表
	err = db.AutoMigrate(
		&models.SSEEvent{},
		&models.DBEventListener{},
		&models.DBChangeEvent{},
		&models.SSEConnection{},
	)
	if err != nil {
		return err
	}

	log.Println("数据库迁移完成")
	return nil
}

// InitializeData 初始化基础数据
func InitializeData(db *gorm.DB) error {
	log.Println("开始初始化基础数据...")

	// 数据源类型元数据现在由动态注册表提供，无需数据库存储
	// err := initDataSourceTypeMeta(db)
	// if err != nil {
	// 	log.Printf("初始化数据源类型元数据失败: %v", err)
	// 	return err
	// }

	// 权限和角色管理已移除，改为使用PostgREST RBAC

	// 初始化默认数据质量规则类型
	qualityRuleTypes := []string{
		"completeness",    // 完整性
		"standardization", // 规范性
		"consistency",     // 一致性
		"accuracy",        // 准确性
		"uniqueness",      // 唯一性
		"timeliness",      // 时效性
	}

	// 初始化默认脱敏类型
	maskingTypes := []string{
		"mask",         // 掩码
		"replace",      // 替换
		"encrypt",      // 加密
		"pseudonymize", // 假名化
	}

	// 初始化默认事件类型
	eventTypes := []string{
		"data_change",         // 数据变更
		"system_notification", // 系统通知
		"user_message",        // 用户消息
		"alert",               // 告警
		"status_update",       // 状态更新
	}

	log.Printf("支持的数据质量规则类型: %v", qualityRuleTypes)
	log.Printf("支持的数据脱敏类型: %v", maskingTypes)
	log.Printf("支持的事件类型: %v", eventTypes)

	log.Println("基础数据初始化完成")
	return nil
}
