/*
 * @module service/thematic_library/interface_type_test
 * @description 主题接口类型验证测试
 */

package thematic_library

import (
	"datahub-service/service/models"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// getTestDB 获取测试数据库连接
func getTestDB(t *testing.T) *gorm.DB {
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbName := getEnvOrDefault("DB_NAME", "postgres")
	dbUser := getEnvOrDefault("DB_USER", "supabase_admin")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "things2024")
	dbSSLMode := getEnvOrDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "连接测试数据库失败")

	// 自动迁移表结构
	err = db.AutoMigrate(
		&models.ThematicLibrary{},
		&models.ThematicInterface{},
	)
	require.NoError(t, err, "迁移数据库表结构失败")

	return db
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// cleanupTestData 清理测试数据
func cleanupTestData(t *testing.T, db *gorm.DB) {
	db.Exec("DELETE FROM thematic_interfaces WHERE name_en LIKE 'test_type_%'")
	db.Exec("DELETE FROM thematic_libraries WHERE name_en LIKE 'test_type_%'")
	db.Exec("DROP SCHEMA IF EXISTS test_type_library CASCADE")
}

// TestInterfaceTypeValidation 测试接口类型验证
func TestInterfaceTypeValidation(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建测试主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试类型验证库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err, "创建主题库应该成功")

	// 测试创建table类型接口
	tableInterface := &models.ThematicInterface{
		LibraryID:   library.ID,
		NameZh:      "表类型接口",
		NameEn:      "test_type_table",
		Type:        "table",
		Description: "表类型接口测试",
		Status:      "active",
	}
	err = service.CreateThematicInterface(tableInterface)
	assert.NoError(t, err, "创建table类型接口应该成功")

	// 测试创建view类型接口
	viewInterface := &models.ThematicInterface{
		LibraryID:   library.ID,
		NameZh:      "视图类型接口",
		NameEn:      "test_type_view",
		Type:        "view",
		Description: "视图类型接口测试",
		Status:      "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	assert.NoError(t, err, "创建view类型接口应该成功")

	// 测试创建无效类型接口（应该在UpdateThematicInterface时验证）
	invalidInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "无效类型接口",
		NameEn:    "test_type_invalid",
		Type:      "table", // 先创建为有效类型
		Status:    "active",
	}
	err = service.CreateThematicInterface(invalidInterface)
	require.NoError(t, err)

	// 尝试更新为无效类型
	updates := &models.ThematicInterface{
		Type: "realtime", // 旧的无效类型
	}
	err = service.UpdateThematicInterface(invalidInterface.ID, updates)
	assert.Error(t, err, "更新为无效类型应该失败")
	assert.Contains(t, err.Error(), "无效的接口类型")
}

// TestTableInterfaceOperations 测试table类型接口操作
func TestTableInterfaceOperations(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试表操作库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建table类型接口
	tableInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "用户表",
		NameEn:    "test_type_users",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(tableInterface)
	require.NoError(t, err)

	// 定义字段
	fields := []models.TableField{
		{
			NameZh:       "ID",
			NameEn:       "id",
			DataType:     "varchar",
			IsPrimaryKey: true,
			IsNullable:   false,
			OrderNum:     1,
		},
		{
			NameZh:     "用户名",
			NameEn:     "username",
			DataType:   "varchar",
			IsNullable: false,
			OrderNum:   2,
		},
	}

	// 更新字段配置（应该成功）
	err = service.UpdateThematicInterfaceFields(tableInterface.ID, fields)
	assert.NoError(t, err, "table类型接口应该可以更新字段配置")

	// 验证表是否创建
	tableExists, err := service.schemaService.CheckTableExists(library.NameEn, tableInterface.NameEn)
	assert.NoError(t, err)
	assert.True(t, tableExists, "表应该被创建")
}

// TestViewInterfaceOperations 测试view类型接口操作
func TestViewInterfaceOperations(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试视图操作库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 先创建基础表
	baseTable := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "基础表",
		NameEn:    "test_type_base",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(baseTable)
	require.NoError(t, err)

	baseFields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
		{NameZh: "名称", NameEn: "name", DataType: "varchar", IsNullable: false, OrderNum: 2},
	}
	err = service.UpdateThematicInterfaceFields(baseTable.ID, baseFields)
	require.NoError(t, err)

	// 创建view类型接口
	viewInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "用户视图",
		NameEn:    "test_type_view",
		Type:      "view",
		Status:    "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 创建视图（应该成功）
	viewSQL := fmt.Sprintf("SELECT id, name FROM %s.%s", library.NameEn, baseTable.NameEn)
	err = service.CreateThematicInterfaceView(viewInterface.ID, viewSQL)
	assert.NoError(t, err, "view类型接口应该可以创建视图")

	// 验证视图是否创建
	viewExists, err := service.schemaService.CheckViewExists(library.NameEn, viewInterface.NameEn)
	assert.NoError(t, err)
	assert.True(t, viewExists, "视图应该被创建")

	// 测试view类型接口不能更新字段配置
	fields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
	}
	err = service.UpdateThematicInterfaceFields(viewInterface.ID, fields)
	assert.Error(t, err, "view类型接口不应该能更新字段配置")
	assert.Contains(t, err.Error(), "只有table类型的接口才能更新字段配置")
}

// TestCrossTypeOperations 测试跨类型操作（应该失败）
func TestCrossTypeOperations(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试跨类型操作库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建table类型接口
	tableInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "表接口",
		NameEn:    "test_type_table",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(tableInterface)
	require.NoError(t, err)

	// 尝试对table类型接口创建视图（应该失败）
	err = service.CreateThematicInterfaceView(tableInterface.ID, "SELECT 1")
	assert.Error(t, err, "table类型接口不应该能创建视图")
	assert.Contains(t, err.Error(), "只有视图类型的接口才能创建视图")

	// 创建view类型接口
	viewInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "视图接口",
		NameEn:    "test_type_view",
		Type:      "view",
		Status:    "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 尝试对view类型接口更新字段配置（应该失败）
	fields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
	}
	err = service.UpdateThematicInterfaceFields(viewInterface.ID, fields)
	assert.Error(t, err, "view类型接口不应该能更新字段配置")
	assert.Contains(t, err.Error(), "只有table类型的接口才能更新字段配置")
}
