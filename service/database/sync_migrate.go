/**
 * @module sync_migrate
 * @description 数据同步相关表迁移模块，负责创建和管理数据同步功能的数据库表结构
 * @architecture 数据访问层 - 同步迁移管理
 * @documentReference 参考 ai_docs/basic_library_process_impl.md 第10.1节
 * @stateFlow 应用启动时执行同步相关表的数据库迁移
 * @rules
 *   - 确保同步相关表结构与模型定义保持一致
 *   - 迁移操作需要支持版本控制
 *   - 数据迁移需要保证数据完整性
 *   - 表创建需要考虑性能和索引优化
 * @dependencies
 *   - service/models: 数据模型
 *   - gorm.io/gorm: ORM框架
 *   - database/sql: 数据库驱动
 * @refs
 *   - service/database/migrate.go: 主迁移文件
 *   - service/models/sync_models.go: 同步模型
 */

package database

import (
	"datahub-service/service/models"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// AutoMigrateSyncTables 自动迁移数据同步相关表
func AutoMigrateSyncTables(db *gorm.DB) error {
	log.Println("开始数据同步相关表迁移...")

	// 核心同步配置和状态表
	err := db.AutoMigrate(
		&models.SyncConfig{},
		&models.SyncExecution{},
		&models.IncrementalState{},
		&models.SyncEngineConfig{},
		&models.ScheduleTask{},
		&models.TaskExecution{},
		&models.SyncStatistics{},
	)
	if err != nil {
		log.Printf("迁移核心同步表失败: %v", err)
		return err
	}

	// 数据质量相关表
	err = db.AutoMigrate(
		&models.QualityCheckExecution{},
		&models.QualityMetricRecord{},
		&models.CleansingRule{},
		&models.DataQualityReport{},
		&models.QualityIssue{},
	)
	if err != nil {
		log.Printf("迁移数据质量表失败: %v", err)
		return err
	}

	// 监控和告警相关表
	err = db.AutoMigrate(
		&models.AlertRule{},
		&models.MonitoringMetric{},
		&models.HealthCheck{},
		&models.SystemMetrics{},
		&models.PerformanceSnapshot{},
	)
	if err != nil {
		log.Printf("迁移监控告警表失败: %v", err)
		return err
	}

	log.Println("数据同步相关表迁移完成")
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

	// 错误日志表索引
	if err := createErrorLogIndexes(db); err != nil {
		return err
	}

	// 调度任务表索引
	if err := createScheduledTaskIndexes(db); err != nil {
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
		"CREATE INDEX IF NOT EXISTS idx_sync_config_data_source_id ON sync_configurations(data_source_id)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_interface_id ON sync_configurations(interface_id)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_status ON sync_configurations(status)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_enabled ON sync_configurations(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_created_at ON sync_configurations(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_sync_config_updated_at ON sync_configurations(updated_at)",
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
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_config_id ON sync_executions(config_id)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_status ON sync_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_started_at ON sync_executions(started_at)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_finished_at ON sync_executions(finished_at)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_trigger_type ON sync_executions(trigger_type)",
		"CREATE INDEX IF NOT EXISTS idx_sync_exec_records_processed ON sync_executions(records_processed)",
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
		"CREATE INDEX IF NOT EXISTS idx_incremental_config_id ON incremental_states(config_id)",
		"CREATE INDEX IF NOT EXISTS idx_incremental_strategy ON incremental_states(strategy)",
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
		"CREATE INDEX IF NOT EXISTS idx_quality_check_config_id ON quality_check_executions(config_id)",
		"CREATE INDEX IF NOT EXISTS idx_quality_check_status ON quality_check_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_quality_check_started_at ON quality_check_executions(started_at)",
		"CREATE INDEX IF NOT EXISTS idx_quality_metric_execution_id ON quality_metric_records(execution_id)",
		"CREATE INDEX IF NOT EXISTS idx_quality_metric_type ON quality_metric_records(metric_type)",
		"CREATE INDEX IF NOT EXISTS idx_quality_metric_recorded_at ON quality_metric_records(recorded_at)",
		"CREATE INDEX IF NOT EXISTS idx_quality_issue_execution_id ON quality_issue_trackings(execution_id)",
		"CREATE INDEX IF NOT EXISTS idx_quality_issue_severity ON quality_issue_trackings(severity)",
		"CREATE INDEX IF NOT EXISTS idx_quality_issue_status ON quality_issue_trackings(status)",
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
		"CREATE INDEX IF NOT EXISTS idx_alert_rule_type ON alert_rules(rule_type)",
		"CREATE INDEX IF NOT EXISTS idx_alert_rule_enabled ON alert_rules(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_monitoring_metric_type ON monitoring_metrics(metric_type)",
		"CREATE INDEX IF NOT EXISTS idx_monitoring_metric_recorded_at ON monitoring_metrics(recorded_at)",
		"CREATE INDEX IF NOT EXISTS idx_health_check_service_name ON health_checks(service_name)",
		"CREATE INDEX IF NOT EXISTS idx_health_check_status ON health_checks(status)",
		"CREATE INDEX IF NOT EXISTS idx_health_check_checked_at ON health_checks(checked_at)",
		"CREATE INDEX IF NOT EXISTS idx_system_metric_collected_at ON system_metrics(collected_at)",
		"CREATE INDEX IF NOT EXISTS idx_performance_snapshot_created_at ON performance_snapshots(created_at)",
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
	defaultConfigs := []map[string]interface{}{
		{
			"name":         "默认批量大小配置",
			"config_key":   "sync.default_batch_size",
			"config_value": "1000",
			"description":  "默认的数据同步批量大小",
		},
		{
			"name":         "默认超时配置",
			"config_key":   "sync.default_timeout",
			"config_value": "300",
			"description":  "默认的同步超时时间（秒）",
		},
		{
			"name":         "默认重试次数",
			"config_key":   "sync.default_retry_count",
			"config_value": "3",
			"description":  "默认的同步失败重试次数",
		},
		{
			"name":         "默认并发数",
			"config_key":   "sync.default_concurrency",
			"config_value": "5",
			"description":  "默认的并发同步任务数",
		},
	}

	for _, config := range defaultConfigs {
		var count int64
		if err := db.Table("system_configs").Where("config_key = ?", config["config_key"]).Count(&count).Error; err != nil {
			return fmt.Errorf("检查默认配置失败: %v", err)
		}

		if count == 0 {
			if err := db.Table("system_configs").Create(config).Error; err != nil {
				log.Printf("创建默认配置失败: %v", err)
				// 继续执行，不中断整个初始化过程
			} else {
				log.Printf("创建默认配置: %s", config["name"])
			}
		}
	}

	return nil
}
