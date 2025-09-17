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
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	log.Println("开始数据库迁移...")

	// 数据基础库相关表
	log.Println("正在迁移数据基础库相关表...")
	err := db.AutoMigrate(
		&models.BasicLibrary{},
		&models.DataInterface{},
		&models.InterfaceField{},
		&models.DataSource{},
		&models.CleansingRule{},
		&models.DataSourceStatus{},
		&models.InterfaceStatus{},
		&models.SyncTask{},
	)
	if err != nil {
		log.Printf("数据基础库表迁移失败: %v", err)
		return err
	}
	log.Println("数据基础库表迁移完成")

	// 数据主题库相关表
	log.Println("正在迁移数据主题库相关表...")
	log.Printf("迁移表: ThematicLibrary, ThematicInterface, DataFlowGraph, FlowNode")
	err = db.AutoMigrate(
		&models.ThematicLibrary{},
		&models.ThematicInterface{},
		&models.DataFlowGraph{},
		&models.FlowNode{},
	)
	if err != nil {
		log.Printf("数据主题库表迁移失败: %v", err)
		return err
	}
	log.Println("数据主题库表迁移完成")

	// 验证主题库表是否创建成功
	if db.Migrator().HasTable(&models.ThematicLibrary{}) {
		log.Println("✅ thematic_libraries 表创建成功")
	} else {
		log.Println("❌ thematic_libraries 表创建失败")
	}

	if db.Migrator().HasTable(&models.ThematicInterface{}) {
		log.Println("✅ thematic_interfaces 表创建成功")
	} else {
		log.Println("❌ thematic_interfaces 表创建失败")
	}

	// 访问控制相关表已移除，改为使用PostgREST RBAC

	// 数据治理相关表
	log.Println("正在迁移数据治理相关表...")
	err = db.AutoMigrate(
		&models.QualityRule{},
		&models.Metadata{},
		&models.DataMaskingRule{},
		&models.SystemLog{},
		&models.BackupConfig{},
		&models.BackupRecord{},
		&models.DataQualityReport{},
		&models.SystemConfig{},
	)
	if err != nil {
		log.Printf("数据治理表迁移失败: %v", err)
		return err
	}
	log.Println("数据治理表迁移完成")

	// 数据共享服务相关表
	log.Println("正在迁移数据共享服务相关表...")
	err = db.AutoMigrate(
		&models.ApiApplication{},
		&models.ApiKey{},
		&models.ApiInterface{},
		&models.ApiRateLimit{},
		&models.DataSubscription{},
		&models.DataAccessRequest{},
		&models.ApiUsageLog{},
	)
	if err != nil {
		log.Printf("数据共享服务表迁移失败: %v", err)
		return err
	}
	log.Println("数据共享服务表迁移完成")

	// 事件管理相关表
	log.Println("正在迁移事件管理相关表...")
	err = db.AutoMigrate(
		&models.SSEEvent{},
		&models.DBEventListener{},
		&models.DBChangeEvent{},
		&models.SSEConnection{},
	)
	if err != nil {
		log.Printf("事件管理表迁移失败: %v", err)
		return err
	}
	log.Println("事件管理表迁移完成")

	// 数据同步相关表
	log.Println("正在迁移数据同步相关表...")
	err = db.AutoMigrate(
		&models.SyncTask{},
		&models.SyncTaskInterface{},
		&models.SyncTaskExecution{},
		&models.SyncConfig{},
		&models.IncrementalState{},
		&models.SyncStatistics{},
	)
	if err != nil {
		log.Printf("数据同步表迁移失败: %v", err)
		return err
	}
	log.Println("数据同步表迁移完成")

	// 数据质量相关表
	log.Println("正在迁移数据质量相关表...")
	err = db.AutoMigrate(
		&models.QualityCheckExecution{},
		&models.QualityMetricRecord{},
		&models.QualityIssueTracker{},
	)
	if err != nil {
		log.Printf("数据质量表迁移失败: %v", err)
		return err
	}
	log.Println("数据质量表迁移完成")

	// 监控和告警相关表
	log.Println("正在迁移监控和告警相关表...")
	err = db.AutoMigrate(
		&models.AlertRule{},
		&models.MonitoringMetric{},
	)
	if err != nil {
		log.Printf("监控和告警表迁移失败: %v", err)
		return err
	}
	log.Println("监控和告警表迁移完成")

	// 创建同步相关索引
	if err := CreateSyncIndexes(db); err != nil {
		log.Printf("创建同步索引失败: %v", err)
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

	// 初始化同步相关基础数据
	if err := InitializeSyncData(db); err != nil {
		log.Printf("初始化同步基础数据失败: %v", err)
		return err
	}

	log.Println("基础数据初始化完成")
	return nil
}

// CreateSyncIndexes 创建同步相关表的索引
func CreateSyncIndexes(db *gorm.DB) error {
	log.Println("开始创建数据同步相关索引...")

	// 同步配置表索引
	if err := createSyncConfigurationIndexes(db); err != nil {
		return err
	}

	// 同步执行表索引
	if err := createSyncExecutionIndexes(db); err != nil {
		return err
	}

	// 增量状态表索引
	if err := createIncrementalStateIndexes(db); err != nil {
		return err
	}

	// 质量检查表索引
	if err := createQualityIndexes(db); err != nil {
		return err
	}

	// 监控表索引
	if err := createMonitoringIndexes(db); err != nil {
		return err
	}

	log.Println("数据同步相关索引创建完成")
	return nil
}

// createSyncConfigurationIndexes 创建同步配置表索引
func createSyncConfigurationIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_sync_config_data_source_id ON sync_configs(data_source_id)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_interface_id ON sync_configs(interface_id)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_status ON sync_configs(status)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_created_at ON sync_configs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_updated_at ON sync_configs(updated_at)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建同步配置表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// createSyncExecutionIndexes 创建同步执行表索引
func createSyncExecutionIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_config_id ON sync_executions(sync_config_id)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_status ON sync_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_start_time ON sync_executions(start_time)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_end_time ON sync_executions(end_time)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_execution_type ON sync_executions(execution_type)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_records_total ON sync_executions(records_total)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建同步执行表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// createIncrementalStateIndexes 创建增量状态表索引
func createIncrementalStateIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_incremental_config_id ON incremental_states(sync_config_id)",
		"CREATE INDEX IF NOT EXISTS idx_incremental_type ON incremental_states(incremental_type)",
		"CREATE INDEX IF NOT EXISTS idx_incremental_last_sync_time ON incremental_states(last_sync_time)",
		"CREATE INDEX IF NOT EXISTS idx_incremental_updated_at ON incremental_states(updated_at)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建增量状态表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// createErrorLogIndexes 创建错误日志表索引
func createErrorLogIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_error_log_execution_id ON sync_error_logs(execution_id)",
		"CREATE INDEX IF NOT EXISTS idx_error_log_error_type ON sync_error_logs(error_type)",
		"CREATE INDEX IF NOT EXISTS idx_error_log_occurred_at ON sync_error_logs(occurred_at)",
		"CREATE INDEX IF NOT EXISTS idx_error_log_resolved ON sync_error_logs(resolved)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建错误日志表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// createScheduledTaskIndexes 创建调度任务表索引
func createScheduledTaskIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_scheduled_task_type ON scheduled_tasks(task_type)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_task_status ON scheduled_tasks(status)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_task_enabled ON scheduled_tasks(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_task_next_run ON scheduled_tasks(next_run_time)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_task_created_at ON scheduled_tasks(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_task_exec_task_id ON task_executions(task_id)",
		"CREATE INDEX IF NOT EXISTS idx_task_exec_status ON task_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_task_exec_started_at ON task_executions(started_at)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建调度任务表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// createQualityIndexes 创建质量相关表索引
func createQualityIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_quality_check_sync_config_id ON quality_check_executions(sync_config_id)",
		"CREATE INDEX IF NOT EXISTS idx_quality_check_status ON quality_check_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_quality_check_start_time ON quality_check_executions(start_time)",
		"CREATE INDEX IF NOT EXISTS idx_quality_metric_target_table ON quality_metric_records(target_table)",
		"CREATE INDEX IF NOT EXISTS idx_quality_metric_type ON quality_metric_records(metric_type)",
		"CREATE INDEX IF NOT EXISTS idx_quality_metric_date ON quality_metric_records(metric_date)",
		"CREATE INDEX IF NOT EXISTS idx_quality_issue_quality_check_id ON quality_issue_trackers(quality_check_id)",
		"CREATE INDEX IF NOT EXISTS idx_quality_issue_severity ON quality_issue_trackers(severity)",
		"CREATE INDEX IF NOT EXISTS idx_quality_issue_status ON quality_issue_trackers(status)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建质量相关表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// createMonitoringIndexes 创建监控相关表索引
func createMonitoringIndexes(db *gorm.DB) error {
	indexQueries := []string{
		"CREATE INDEX IF NOT EXISTS idx_alert_rule_metric_name ON alert_rules(metric_name)",
		"CREATE INDEX IF NOT EXISTS idx_alert_rule_enabled ON alert_rules(is_enabled)",
		"CREATE INDEX IF NOT EXISTS idx_monitoring_metric_type ON monitoring_metrics(metric_type)",
		"CREATE INDEX IF NOT EXISTS idx_monitoring_metric_timestamp ON monitoring_metrics(timestamp)",
	}

	for _, query := range indexQueries {
		if err := db.Exec(query).Error; err != nil {
			log.Printf("创建监控相关表索引失败: %v", err)
			return err
		}
	}

	return nil
}

// InitializeSyncData 初始化同步相关基础数据
func InitializeSyncData(db *gorm.DB) error {
	log.Println("开始初始化数据同步相关基础数据...")

	// 初始化默认同步策略类型
	syncStrategies := []string{
		"full_sync",   // 全量同步
		"incremental", // 增量同步
		"timestamp",   // 时间戳增量
		"primary_key", // 主键增量
		"change_log",  // 变更日志
		"realtime",    // 实时同步
	}

	// 初始化默认调度类型
	scheduleTypes := []string{
		"cron",     // Cron表达式
		"interval", // 固定间隔
		"once",     // 一次性
		"manual",   // 手动触发
	}

	// 初始化默认数据源状态
	dataSourceStatuses := []string{
		"active",      // 活跃
		"inactive",    // 非活跃
		"error",       // 错误
		"maintenance", // 维护中
		"testing",     // 测试中
	}

	// 初始化默认质量规则类型
	qualityRuleTypes := []string{
		"completeness", // 完整性检查
		"accuracy",     // 准确性检查
		"consistency",  // 一致性检查
		"validity",     // 有效性检查
		"uniqueness",   // 唯一性检查
		"timeliness",   // 时效性检查
		"referential",  // 参照完整性
		"format",       // 格式检查
		"range",        // 范围检查
		"pattern",      // 模式检查
	}

	// 初始化默认告警类型
	alertTypes := []string{
		"sync_failure",     // 同步失败
		"quality_degraded", // 质量下降
		"performance_slow", // 性能缓慢
		"connection_error", // 连接错误
		"resource_usage",   // 资源使用
		"threshold_breach", // 阈值突破
		"system_error",     // 系统错误
		"data_anomaly",     // 数据异常
	}

	log.Printf("支持的同步策略类型: %v", syncStrategies)
	log.Printf("支持的调度类型: %v", scheduleTypes)
	log.Printf("支持的数据源状态: %v", dataSourceStatuses)
	log.Printf("支持的质量规则类型: %v", qualityRuleTypes)
	log.Printf("支持的告警类型: %v", alertTypes)

	// 创建系统默认配置记录
	if err := createDefaultSyncConfigurations(db); err != nil {
		return err
	}

	log.Println("数据同步相关基础数据初始化完成")
	return nil
}

// createDefaultSyncConfigurations 创建默认同步配置
func createDefaultSyncConfigurations(db *gorm.DB) error {
	// 创建默认的系统级配置
	defaultConfigs := []models.SystemConfig{
		{
			ID:          uuid.New().String(),
			Key:         "sync.default_batch_size",
			Value:       "1000",
			Environment: "default",
			Description: "默认的数据同步批量大小",
		},
		{
			ID:          uuid.New().String(),
			Key:         "sync.default_timeout",
			Value:       "300",
			Environment: "default",
			Description: "默认的同步超时时间（秒）",
		},
		{
			ID:          uuid.New().String(),
			Key:         "sync.default_retry_count",
			Value:       "3",
			Environment: "default",
			Description: "默认的同步失败重试次数",
		},
		{
			ID:          uuid.New().String(),
			Key:         "sync.default_concurrency",
			Value:       "5",
			Environment: "default",
			Description: "默认的并发同步任务数",
		},
	}

	for _, config := range defaultConfigs {
		var count int64
		if err := db.Model(&models.SystemConfig{}).Where("key = ?", config.Key).Count(&count).Error; err != nil {
			return fmt.Errorf("检查默认配置失败: %v", err)
		}

		if count == 0 {
			if err := db.Create(&config).Error; err != nil {
				log.Printf("创建默认配置失败: %v", err)
				// 继续执行，不中断整个初始化过程
			} else {
				log.Printf("创建默认配置: %s", config.Key)
			}
		}
	}

	return nil
}
