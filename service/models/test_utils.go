/*
 * @module service/models/test_utils
 * @description 模型测试辅助工具
 * @architecture 测试基础设施 - 专门为模型测试提供工具
 * @documentReference .specify/memory/test_plan.md
 * @stateFlow 测试环境初始化 -> 测试数据创建 -> 测试执行 -> 清理资源
 * @rules 避免循环导入，专门为模型层测试提供工具
 * @dependencies gorm, sqlite, time
 */

package models

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ModelTestDB 模型测试数据库配置
type ModelTestDB struct {
	DB *gorm.DB
}

// NewModelTestDB 创建模型测试数据库
func NewModelTestDB() *ModelTestDB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("failed to connect test database: %v", err))
	}

	// 自动迁移所有模型
	err = db.AutoMigrate(
		&BasicLibrary{},
		&DataSource{},
		&DataInterface{},
		&SyncTask{},
		&ThematicLibrary{},
		&QualityRuleTemplate{},
		&ApiApplication{},
		&ApiKey{},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to migrate test database: %v", err))
	}

	return &ModelTestDB{DB: db}
}

// CleanDB 清理数据库
func (tdb *ModelTestDB) CleanDB() {
	// 清空所有表的数据
	tables := []string{
		"basic_libraries",
		"data_sources",
		"data_interfaces",
		"sync_tasks",
		"thematic_libraries",
		"quality_rule_templates",
		"api_applications",
		"api_keys",
	}

	for _, table := range tables {
		tdb.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
}

// Close 关闭数据库连接
func (tdb *ModelTestDB) Close() {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		fmt.Printf("Error getting underlying DB: %v\n", err)
		return
	}
	sqlDB.Close()
}

// ModelTestDataFactory 模型测试数据工厂
type ModelTestDataFactory struct {
	DB *gorm.DB
}

// NewModelTestDataFactory 创建新的模型测试数据工厂
func NewModelTestDataFactory(db *gorm.DB) *ModelTestDataFactory {
	return &ModelTestDataFactory{DB: db}
}

// CreateBasicLibrary 创建测试基础库
func (f *ModelTestDataFactory) CreateBasicLibrary() *BasicLibrary {
	library := &BasicLibrary{
		ID:          generateID("bl"),
		NameZh:      "测试基础库",
		NameEn:      "test_basic_library_" + generateSuffix(),
		Description: "这是一个测试基础库",
		Status:      "active",
		CreatedBy:   "test",
		UpdatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := f.DB.Create(library).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test basic library: %v", err))
	}

	return library
}

// CreateDataSource 创建测试数据源
func (f *ModelTestDataFactory) CreateDataSource(libraryID string) *DataSource {
	dataSource := &DataSource{
		ID:               generateID("ds"),
		LibraryID:        libraryID,
		Name:             "测试数据源",
		Category:         "api",
		Type:             "http_no_auth",
		Status:           "active",
		ConnectionConfig: JSONB{"url": "http://example.com/api"},
		ParamsConfig:     JSONB{},
		Script:           "",
		ScriptEnabled:    false,
		CreatedBy:        "test",
		UpdatedBy:        "test",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := f.DB.Create(dataSource).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test data source: %v", err))
	}

	return dataSource
}

// CreateDataInterface 创建测试数据接口
func (f *ModelTestDataFactory) CreateDataInterface(libraryID, dataSourceID string) *DataInterface {
	dataInterface := &DataInterface{
		ID:                generateID("di"),
		LibraryID:         libraryID,
		DataSourceID:      dataSourceID,
		NameZh:            "测试数据接口",
		NameEn:            "test_interface_" + generateSuffix(),
		Type:              "api",
		Description:       "这是一个测试数据接口",
		Status:            "active",
		IsTableCreated:    false,
		InterfaceConfig:   JSONB{},
		ParseConfig:       JSONB{},
		TableFieldsConfig: JSONB{},
		CreatedBy:         "test",
		UpdatedBy:         "test",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := f.DB.Create(dataInterface).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test data interface: %v", err))
	}

	return dataInterface
}

// CreateSyncTask 创建测试同步任务
func (f *ModelTestDataFactory) CreateSyncTask(libraryID, dataSourceID string) *SyncTask {
	now := time.Now()
	syncTask := &SyncTask{
		ID:              generateID("st"),
		LibraryID:       libraryID,
		LibraryType:     "basic_library",
		DataSourceID:    dataSourceID,
		TaskType:        "batch_sync",
		Status:          "pending",
		TriggerType:     "manual",
		CronExpression:  "0 */1 * * *", // 每小时执行一次
		IntervalSeconds: 3600,
		NextRunTime:     &now,
		LastRunTime:     &now,
		CreatedBy:       "test",
		UpdatedBy:       "test",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := f.DB.Create(syncTask).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test sync task: %v", err))
	}

	return syncTask
}

// CreateThematicLibrary 创建测试主题库
func (f *ModelTestDataFactory) CreateThematicLibrary() *ThematicLibrary {
	library := &ThematicLibrary{
		ID:          generateID("tl"),
		NameZh:      "测试主题库",
		NameEn:      "test_thematic_library_" + generateSuffix(),
		Description: "这是一个测试主题库",
		Status:      "active",
		CreatedBy:   "test",
		UpdatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := f.DB.Create(library).Error
	if err != nil {
		panic(fmt.Sprintf("failed to create test thematic library: %v", err))
	}

	return library
}

// 辅助函数
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), generateSuffix())
}

func generateSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%100000)
}
