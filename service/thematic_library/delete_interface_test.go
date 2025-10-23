/*
 * @module service/thematic_library/delete_interface_test
 * @description 测试删除主题接口时自动删除对应的表或视图
 */

package thematic_library

import (
	"datahub-service/service/models"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteTableInterface 测试删除table类型接口时删除对应的表
func TestDeleteTableInterface(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试删除表接口库",
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
		NameZh:    "待删除的表",
		NameEn:    "test_type_table_to_delete",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(tableInterface)
	require.NoError(t, err)

	// 创建表
	fields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
		{NameZh: "名称", NameEn: "name", DataType: "varchar", IsNullable: false, OrderNum: 2},
	}
	err = service.UpdateThematicInterfaceFields(tableInterface.ID, fields)
	require.NoError(t, err)

	// 验证表已创建
	tableExists, err := service.schemaService.CheckTableExists(library.NameEn, tableInterface.NameEn)
	require.NoError(t, err)
	require.True(t, tableExists, "表应该已创建")

	// 删除接口
	err = service.DeleteThematicInterface(tableInterface.ID)
	assert.NoError(t, err, "删除table接口应该成功")

	// 验证接口记录已删除
	_, err = service.GetThematicInterface(tableInterface.ID)
	assert.Error(t, err, "接口记录应该已删除")

	// 验证表已删除
	tableExists, err = service.schemaService.CheckTableExists(library.NameEn, tableInterface.NameEn)
	assert.NoError(t, err)
	assert.False(t, tableExists, "表应该已被删除")
}

// TestDeleteViewInterface 测试删除view类型接口时删除对应的视图
func TestDeleteViewInterface(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试删除视图接口库",
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

	// 创建view类型接口
	viewSQL := fmt.Sprintf("SELECT id, name FROM %s.%s", library.NameEn, baseTable.NameEn)
	viewInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "待删除的视图",
		NameEn:    "test_type_view_to_delete",
		Type:      "view",
		ViewSQL:   viewSQL,
		Status:    "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 验证视图已创建
	viewExists, err := service.schemaService.CheckViewExists(library.NameEn, viewInterface.NameEn)
	require.NoError(t, err)
	require.True(t, viewExists, "视图应该已创建")

	// 删除视图接口
	err = service.DeleteThematicInterface(viewInterface.ID)
	assert.NoError(t, err, "删除view接口应该成功")

	// 验证接口记录已删除
	_, err = service.GetThematicInterface(viewInterface.ID)
	assert.Error(t, err, "接口记录应该已删除")

	// 验证视图已删除
	viewExists, err = service.schemaService.CheckViewExists(library.NameEn, viewInterface.NameEn)
	assert.NoError(t, err)
	assert.False(t, viewExists, "视图应该已被删除")

	// 验证基础表未被删除（只删除视图，不删除依赖的表）
	baseTableExists, err := service.schemaService.CheckTableExists(library.NameEn, baseTable.NameEn)
	assert.NoError(t, err)
	assert.True(t, baseTableExists, "基础表不应该被删除")
}

// TestDeleteInterfaceWithoutDatabaseObject 测试删除未创建表/视图的接口
func TestDeleteInterfaceWithoutDatabaseObject(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试删除无数据库对象接口库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建table接口但不创建表
	tableInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "未创建表的接口",
		NameEn:    "test_type_no_table",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(tableInterface)
	require.NoError(t, err)

	// 验证is_table_created为false
	savedInterface, _ := service.GetThematicInterface(tableInterface.ID)
	assert.False(t, savedInterface.IsTableCreated)

	// 删除接口应该成功（即使没有对应的表）
	err = service.DeleteThematicInterface(tableInterface.ID)
	assert.NoError(t, err, "删除未创建表的接口应该成功")

	// 创建view接口但不创建视图
	viewInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "未创建视图的接口",
		NameEn:    "test_type_no_view",
		Type:      "view",
		Status:    "active",
	}
	err = service.CreateThematicInterface(viewInterface)
	require.NoError(t, err)

	// 验证is_view_created为false
	savedInterface, _ = service.GetThematicInterface(viewInterface.ID)
	assert.False(t, savedInterface.IsViewCreated)

	// 删除接口应该成功（即使没有对应的视图）
	err = service.DeleteThematicInterface(viewInterface.ID)
	assert.NoError(t, err, "删除未创建视图的接口应该成功")
}

// TestDeleteInterfaceWithRelations 测试删除有关联的接口应该失败
func TestDeleteInterfaceWithRelations(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试删除有关联接口库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建接口
	tableInterface := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "有关联的表",
		NameEn:    "test_type_related_table",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(tableInterface)
	require.NoError(t, err)

	// 创建关联的数据流程图
	flowGraph := &models.DataFlowGraph{
		ThematicInterfaceID: tableInterface.ID,
		Name:                "测试流程图",
		Description:         "测试用流程图",
		Definition: map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		},
		Status: "active",
	}
	err = db.Create(flowGraph).Error
	require.NoError(t, err)

	// 尝试删除接口应该失败
	err = service.DeleteThematicInterface(tableInterface.ID)
	assert.Error(t, err, "删除有关联数据流程图的接口应该失败")
	assert.Contains(t, err.Error(), "存在关联的数据流程图")

	// 验证接口和表仍然存在
	savedInterface, err := service.GetThematicInterface(tableInterface.ID)
	assert.NoError(t, err, "接口应该仍然存在")
	assert.NotNil(t, savedInterface)
}

// TestDeleteMultipleInterfaces 测试批量删除接口
func TestDeleteMultipleInterfaces(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试批量删除库",
		NameEn:          "test_type_library",
		Category:        "business",
		Domain:          "user",
		PublishStatus:   "draft",
		AccessLevel:     "internal",
		UpdateFrequency: "daily",
	}
	err := service.CreateThematicLibrary(library)
	require.NoError(t, err)

	// 创建多个table接口
	var tableInterfaces []*models.ThematicInterface
	for i := 1; i <= 3; i++ {
		iface := &models.ThematicInterface{
			LibraryID: library.ID,
			NameZh:    fmt.Sprintf("测试表%d", i),
			NameEn:    fmt.Sprintf("test_type_table_%d", i),
			Type:      "table",
			Status:    "active",
		}
		err = service.CreateThematicInterface(iface)
		require.NoError(t, err)

		fields := []models.TableField{
			{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
		}
		err = service.UpdateThematicInterfaceFields(iface.ID, fields)
		require.NoError(t, err)

		tableInterfaces = append(tableInterfaces, iface)
	}

	// 创建多个view接口
	baseTable := tableInterfaces[0]
	var viewInterfaces []*models.ThematicInterface
	for i := 1; i <= 2; i++ {
		viewSQL := fmt.Sprintf("SELECT id FROM %s.%s", library.NameEn, baseTable.NameEn)
		iface := &models.ThematicInterface{
			LibraryID: library.ID,
			NameZh:    fmt.Sprintf("测试视图%d", i),
			NameEn:    fmt.Sprintf("test_type_view_%d", i),
			Type:      "view",
			ViewSQL:   viewSQL,
			Status:    "active",
		}
		err = service.CreateThematicInterface(iface)
		require.NoError(t, err)

		viewInterfaces = append(viewInterfaces, iface)
	}

	// 删除所有view接口
	for _, iface := range viewInterfaces {
		err = service.DeleteThematicInterface(iface.ID)
		assert.NoError(t, err, "删除视图接口应该成功")

		// 验证视图已删除
		viewExists, _ := service.schemaService.CheckViewExists(library.NameEn, iface.NameEn)
		assert.False(t, viewExists, "视图应该已删除")
	}

	// 删除所有table接口
	for _, iface := range tableInterfaces {
		err = service.DeleteThematicInterface(iface.ID)
		assert.NoError(t, err, "删除表接口应该成功")

		// 验证表已删除
		tableExists, _ := service.schemaService.CheckTableExists(library.NameEn, iface.NameEn)
		assert.False(t, tableExists, "表应该已删除")
	}
}

// TestDeleteInterfaceCascade 测试删除接口的级联效果
func TestDeleteInterfaceCascade(t *testing.T) {
	db := getTestDB(t)
	service := NewService(db)
	defer cleanupTestData(t, db)

	// 创建主题库
	library := &models.ThematicLibrary{
		NameZh:          "测试级联删除库",
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
		NameEn:    "test_type_base",
		Type:      "table",
		Status:    "active",
	}
	err = service.CreateThematicInterface(baseTable)
	require.NoError(t, err)

	fields := []models.TableField{
		{NameZh: "ID", NameEn: "id", DataType: "varchar", IsPrimaryKey: true, IsNullable: false, OrderNum: 1},
		{NameZh: "数据", NameEn: "data", DataType: "varchar", IsNullable: false, OrderNum: 2},
	}
	err = service.UpdateThematicInterfaceFields(baseTable.ID, fields)
	require.NoError(t, err)

	// 基于基础表创建多个视图
	viewSQL1 := fmt.Sprintf("SELECT id FROM %s.%s", library.NameEn, baseTable.NameEn)
	view1 := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "视图1",
		NameEn:    "test_type_view_1",
		Type:      "view",
		ViewSQL:   viewSQL1,
		Status:    "active",
	}
	err = service.CreateThematicInterface(view1)
	require.NoError(t, err)

	viewSQL2 := fmt.Sprintf("SELECT data FROM %s.%s", library.NameEn, baseTable.NameEn)
	view2 := &models.ThematicInterface{
		LibraryID: library.ID,
		NameZh:    "视图2",
		NameEn:    "test_type_view_2",
		Type:      "view",
		ViewSQL:   viewSQL2,
		Status:    "active",
	}
	err = service.CreateThematicInterface(view2)
	require.NoError(t, err)

	// 先删除视图
	err = service.DeleteThematicInterface(view1.ID)
	assert.NoError(t, err)
	err = service.DeleteThematicInterface(view2.ID)
	assert.NoError(t, err)

	// 验证视图已删除
	view1Exists, _ := service.schemaService.CheckViewExists(library.NameEn, view1.NameEn)
	assert.False(t, view1Exists)
	view2Exists, _ := service.schemaService.CheckViewExists(library.NameEn, view2.NameEn)
	assert.False(t, view2Exists)

	// 删除基础表
	err = service.DeleteThematicInterface(baseTable.ID)
	assert.NoError(t, err)

	// 验证基础表已删除
	baseTableExists, _ := service.schemaService.CheckTableExists(library.NameEn, baseTable.NameEn)
	assert.False(t, baseTableExists)
}
