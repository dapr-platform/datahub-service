/*
 * @module service/thematic_library/view_sql_test
 * @description 测试view_sql字段的创建、更新和读取
 */

package thematic_library

import (
	"datahub-service/service/models"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateInterfaceWithViewSQL 测试创建接口时提供view_sql
func TestCreateInterfaceWithViewSQL(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试ViewSQL库",
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
		NameEn:    "test_type_base_table",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(baseTable)
	require.NoError(t, err)

	// 创建基础表字段
	baseFields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
		{NameZh: "名称", NameEn: "name", DataType: "varchar", IsNullable: false, OrderNum: 2},
		{NameZh: "状态", NameEn: "status", DataType: "varchar", IsNullable: false, OrderNum: 3},
	}
	err = service.UpdateThematicInterfaceFields(baseTable.ID, baseFields)
	require.NoError(t, err)

	// 创建view类型接口，同时提供view_sql
	viewSQL := fmt.Sprintf("SELECT id, name FROM %s.%s WHERE status = 'active'", library.NameEn, baseTable.NameEn)
	viewInterface := &models.ThematicInterface{
		LibraryID:   library.ID,
		NameZh:      "活跃记录视图",
		NameEn:      "test_type_active_view",
		Type:        "view",
		ViewSQL:     viewSQL,
		Description: "自动创建的视图",
		Status:      "active",
	}

	err = service.CreateThematicInterface(viewInterface)
	assert.NoError(t, err, "创建view接口时提供view_sql应该成功")
	assert.NotEmpty(t, viewInterface.ID, "接口ID应该被生成")

	// 验证视图是否自动创建
	assert.True(t, viewInterface.IsViewCreated, "IsViewCreated应该为true")

	// 从数据库读取接口，验证view_sql是否被保存
	savedInterface, err := service.GetThematicInterface(viewInterface.ID)
	assert.NoError(t, err)
	assert.Equal(t, viewSQL, savedInterface.ViewSQL, "ViewSQL应该被正确保存")
	assert.True(t, savedInterface.IsViewCreated, "IsViewCreated应该为true")

	// 验证视图在数据库中存在
	viewExists, err := service.schemaService.CheckViewExists(library.NameEn, viewInterface.NameEn)
	assert.NoError(t, err)
	assert.True(t, viewExists, "视图应该在数据库中存在")
}

// TestUpdateInterfaceWithViewSQL 测试更新接口时提供view_sql
func TestUpdateInterfaceWithViewSQL(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试ViewSQL更新库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建基础表
	baseTable := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "基础表",
		NameEn:    "test_type_base_table",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(baseTable)
	require.NoError(t, err)

	baseFields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
		{NameZh: "名称", NameEn: "name", DataType: "varchar", IsNullable: false, OrderNum: 2},
		{NameZh: "状态", NameEn: "status", DataType: "varchar", IsNullable: false, OrderNum: 3},
	}
	err = service.UpdateThematicInterfaceFields(baseTable.ID, baseFields)
	require.NoError(t, err)

	// 创建view类型接口，但不提供view_sql
	viewInterface := &models.ThematicInterface{
		LibraryID:   library.ID,
		NameZh:      "测试视图",
		NameEn:      "test_type_my_view",
		Type:        "view",
		Description: "待配置的视图",
		Status:      "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 验证视图未创建
	savedInterface, err := service.GetThematicInterface(viewInterface.ID)
	require.NoError(t, err)
	assert.False(t, savedInterface.IsViewCreated, "初始时IsViewCreated应该为false")
	assert.Empty(t, savedInterface.ViewSQL, "初始时ViewSQL应该为空")

	// 通过更新接口提供view_sql
	viewSQL := fmt.Sprintf("SELECT id, name FROM %s.%s WHERE status = 'active'", library.NameEn, baseTable.NameEn)
	updates := &models.ThematicInterface{
		ViewSQL:     viewSQL,
		Description: "已配置视图SQL",
	}

	err = service.UpdateThematicInterface(viewInterface.ID, updates)
	assert.NoError(t, err, "更新接口时提供view_sql应该成功")

	// 从数据库读取接口，验证view_sql是否被保存和视图是否创建
	updatedInterface, err := service.GetThematicInterface(viewInterface.ID)
	assert.NoError(t, err)
	assert.Equal(t, viewSQL, updatedInterface.ViewSQL, "ViewSQL应该被正确保存")
	assert.True(t, updatedInterface.IsViewCreated, "IsViewCreated应该为true")

	// 验证视图在数据库中存在
	viewExists, err := service.schemaService.CheckViewExists(library.NameEn, viewInterface.NameEn)
	assert.NoError(t, err)
	assert.True(t, viewExists, "视图应该在数据库中存在")
}

// TestListInterfacesIncludesViewSQL 测试列表接口是否包含view_sql
func TestListInterfacesIncludesViewSQL(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试列表ViewSQL库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建基础表
	baseTable := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "基础表",
		NameEn:    "test_type_base_table",
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

	// 创建带view_sql的view接口
	viewSQL := fmt.Sprintf("SELECT id, name FROM %s.%s", library.NameEn, baseTable.NameEn)
	viewInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "列表测试视图",
		NameEn:    "test_type_list_view",
		Type:      "view",
		ViewSQL:   viewSQL,
		Status:    "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 获取接口列表
	interfaces, total, err := service.GetThematicInterfaceList(1, 10, library.ID, "view", "", "")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1), "应该至少有一个view接口")
	assert.GreaterOrEqual(t, len(interfaces), 1, "列表应该至少有一个接口")

	// 验证列表中包含view_sql
	found := false
	for _, iface := range interfaces {
		if iface.ID == viewInterface.ID {
			found = true
			assert.Equal(t, viewSQL, iface.ViewSQL, "列表中的ViewSQL应该被正确返回")
			assert.True(t, iface.IsViewCreated, "列表中的IsViewCreated应该为true")
			break
		}
	}
	assert.True(t, found, "应该能在列表中找到创建的视图接口")
}

// TestUpdateViewSQLMultipleTimes 测试多次更新view_sql
func TestUpdateViewSQLMultipleTimes(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库和基础表
	library := &models.ThematicLibrary{
		NameZh:          "测试多次更新ViewSQL库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

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
		{NameZh: "值", NameEn: "value", DataType: "integer", IsNullable: true, OrderNum: 3},
	}
	err = service.UpdateThematicInterfaceFields(baseTable.ID, baseFields)
	require.NoError(t, err)

	// 创建view接口
	viewInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "多次更新测试视图",
		NameEn:    "test_type_multi_update_view",
		Type:      "view",
		Status:    "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 第一次设置view_sql
	viewSQL1 := fmt.Sprintf("SELECT id, name FROM %s.%s", library.NameEn, baseTable.NameEn)
	err = service.UpdateThematicInterface(viewInterface.ID, &models.ThematicInterface{ViewSQL: viewSQL1})
	assert.NoError(t, err)

	savedInterface, _ := service.GetThematicInterface(viewInterface.ID)
	assert.Equal(t, viewSQL1, savedInterface.ViewSQL)

	// 第二次更新view_sql
	viewSQL2 := fmt.Sprintf("SELECT id, name, value FROM %s.%s WHERE value > 0", library.NameEn, baseTable.NameEn)
	err = service.UpdateThematicInterface(viewInterface.ID, &models.ThematicInterface{ViewSQL: viewSQL2})
	assert.NoError(t, err)

	savedInterface, _ = service.GetThematicInterface(viewInterface.ID)
	assert.Equal(t, viewSQL2, savedInterface.ViewSQL, "ViewSQL应该被更新为新值")

	// 验证视图在数据库中存在且是最新的
	viewExists, err := service.schemaService.CheckViewExists(library.NameEn, viewInterface.NameEn)
	assert.NoError(t, err)
	assert.True(t, viewExists, "视图应该在数据库中存在")
}
