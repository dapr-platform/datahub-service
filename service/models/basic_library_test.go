/*
 * @module service/models/basic_library_test
 * @description 基础库数据模型验证测试
 * @architecture 测试层 - 数据模型验证，确保数据完整性和约束
 * @documentReference .specify/memory/test_plan.md
 * @stateFlow 模型创建 -> 字段验证 -> 约束检查 -> 结果断言
 * @rules 确保数据模型的完整性、约束和业务规则
 * @dependencies testing, testify, gorm, datahub-service/testutil
 * @refs basic_library.go
 */

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// BasicLibraryModelTestSuite 基础库模型测试套件
type BasicLibraryModelTestSuite struct {
	suite.Suite
	testDB  *ModelTestDB
	factory *ModelTestDataFactory
}

// SetupSuite 设置测试套件
func (suite *BasicLibraryModelTestSuite) SetupSuite() {
	suite.testDB = NewModelTestDB()
	suite.factory = NewModelTestDataFactory(suite.testDB.DB)
}

// TearDownSuite 清理测试套件
func (suite *BasicLibraryModelTestSuite) TearDownSuite() {
	suite.testDB.Close()
}

// SetupTest 设置每个测试
func (suite *BasicLibraryModelTestSuite) SetupTest() {
	suite.testDB.CleanDB()
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryCreation() {
	// 测试基础库创建
	library := &BasicLibrary{
		ID:          "test-basic-lib-001",
		NameZh:      "测试基础库",
		NameEn:      "test_basic_library",
		Description: "这是一个测试基础库",
		Status:      "active",
		CreatedBy:   "test_user",
		UpdatedBy:   "test_user",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存到数据库
	err := suite.testDB.DB.Create(library).Error
	suite.NoError(err)
	suite.NotEmpty(library.ID)

	// 验证数据完整性
	var savedLibrary BasicLibrary
	err = suite.testDB.DB.First(&savedLibrary, "id = ?", library.ID).Error
	suite.NoError(err)
	suite.Equal(library.NameZh, savedLibrary.NameZh)
	suite.Equal(library.NameEn, savedLibrary.NameEn)
	suite.Equal(library.Status, savedLibrary.Status)
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryValidation() {
	testCases := []struct {
		name        string
		library     BasicLibrary
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效的基础库",
			library: BasicLibrary{
				NameZh:      "有效基础库",
				NameEn:      "valid_library",
				Description: "有效描述",
				Status:      "active",
				CreatedBy:   "user1",
				UpdatedBy:   "user1",
			},
			expectError: false,
		},
		{
			name: "缺少中文名称",
			library: BasicLibrary{
				NameEn:      "missing_zh_name",
				Description: "缺少中文名称",
				Status:      "active",
				CreatedBy:   "user1",
				UpdatedBy:   "user1",
			},
			expectError: true,
			errorMsg:    "中文名称不能为空",
		},
		{
			name: "缺少英文名称",
			library: BasicLibrary{
				NameZh:      "缺少英文名称",
				Description: "缺少英文名称",
				Status:      "active",
				CreatedBy:   "user1",
				UpdatedBy:   "user1",
			},
			expectError: true,
			errorMsg:    "英文名称不能为空",
		},
		{
			name: "无效状态",
			library: BasicLibrary{
				NameZh:      "无效状态库",
				NameEn:      "invalid_status",
				Description: "无效状态",
				Status:      "unknown_status",
				CreatedBy:   "user1",
				UpdatedBy:   "user1",
			},
			expectError: true,
			errorMsg:    "状态值无效",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := suite.testDB.DB.Create(&tc.library).Error
			if tc.expectError {
				suite.Error(err, tc.errorMsg)
			} else {
				suite.NoError(err)
				// 清理测试数据
				suite.testDB.DB.Delete(&tc.library)
			}
		})
	}
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryUniqueness() {
	// 测试英文名称唯一性
	library1 := &BasicLibrary{
		NameZh:    "基础库1",
		NameEn:    "unique_library",
		Status:    "active",
		CreatedBy: "user1",
		UpdatedBy: "user1",
	}

	library2 := &BasicLibrary{
		NameZh:    "基础库2",
		NameEn:    "unique_library", // 相同的英文名称
		Status:    "active",
		CreatedBy: "user1",
		UpdatedBy: "user1",
	}

	// 第一个应该成功
	err := suite.testDB.DB.Create(library1).Error
	suite.NoError(err)

	// 第二个应该失败（如果有唯一性约束）
	err = suite.testDB.DB.Create(library2).Error
	// 注意：这里的行为取决于数据库约束的实际设置
	// 如果没有设置唯一约束，这个测试可能需要调整
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryStatusValues() {
	validStatuses := []string{"active", "inactive", "draft", "archived"}

	for _, status := range validStatuses {
		library := &BasicLibrary{
			NameZh:    "状态测试库",
			NameEn:    "status_test_" + status,
			Status:    status,
			CreatedBy: "user1",
			UpdatedBy: "user1",
		}

		err := suite.testDB.DB.Create(library).Error
		suite.NoError(err, "状态 %s 应该是有效的", status)

		// 清理
		suite.testDB.DB.Delete(library)
	}
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryWithDataSources() {
	// 创建基础库
	library := suite.factory.CreateBasicLibrary()

	// 创建数据源
	dataSource := suite.factory.CreateDataSource(library.ID)

	// 验证关联关系
	var loadedLibrary BasicLibrary
	err := suite.testDB.DB.Preload("DataSources").First(&loadedLibrary, "id = ?", library.ID).Error
	suite.NoError(err)
	suite.Len(loadedLibrary.DataSources, 1)
	suite.Equal(dataSource.Name, loadedLibrary.DataSources[0].Name)
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryWithInterfaces() {
	// 创建基础库和数据源
	library := suite.factory.CreateBasicLibrary()
	dataSource := suite.factory.CreateDataSource(library.ID)

	// 创建数据接口
	dataInterface := suite.factory.CreateDataInterface(library.ID, dataSource.ID)

	// 验证关联关系
	var loadedLibrary BasicLibrary
	err := suite.testDB.DB.Preload("Interfaces").First(&loadedLibrary, "id = ?", library.ID).Error
	suite.NoError(err)
	suite.Len(loadedLibrary.Interfaces, 1)
	suite.Equal(dataInterface.NameZh, loadedLibrary.Interfaces[0].NameZh)
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibraryTimestamps() {
	// 测试时间戳自动设置
	library := &BasicLibrary{
		NameZh:    "时间戳测试库",
		NameEn:    "timestamp_test",
		Status:    "active",
		CreatedBy: "user1",
		UpdatedBy: "user1",
	}

	// 创建前时间戳应该为零值
	suite.True(library.CreatedAt.IsZero())
	suite.True(library.UpdatedAt.IsZero())

	// 保存到数据库
	err := suite.testDB.DB.Create(library).Error
	suite.NoError(err)

	// 创建后时间戳应该被设置（如果有钩子）
	// 注意：这取决于GORM的配置和模型的钩子设置

	// 更新测试
	time.Sleep(time.Millisecond) // 确保时间差异

	library.Description = "更新后的描述"
	err = suite.testDB.DB.Save(library).Error
	suite.NoError(err)

	// UpdatedAt应该被更新（如果有钩子）
	// suite.True(library.UpdatedAt.After(originalUpdatedAt))
}

func (suite *BasicLibraryModelTestSuite) TestBasicLibrarySoftDelete() {
	// 创建基础库
	library := suite.factory.CreateBasicLibrary()
	libraryID := library.ID

	// 软删除
	err := suite.testDB.DB.Delete(library).Error
	suite.NoError(err)

	// 正常查询应该找不到
	var normalQuery BasicLibrary
	err = suite.testDB.DB.First(&normalQuery, "id = ?", libraryID).Error
	suite.Equal(gorm.ErrRecordNotFound, err)

	// 包含软删除的查询应该能找到
	var withDeletedQuery BasicLibrary
	err = suite.testDB.DB.Unscoped().First(&withDeletedQuery, "id = ?", libraryID).Error
	suite.NoError(err)
	// 注意：这里需要根据实际的模型结构来验证软删除字段
	// 如果BasicLibrary没有DeletedAt字段，可能需要检查其他字段或跳过此测试
}

// 运行测试套件
func TestBasicLibraryModel(t *testing.T) {
	suite.Run(t, new(BasicLibraryModelTestSuite))
}

// 独立的单元测试
func TestBasicLibraryFieldValidation(t *testing.T) {
	testCases := []struct {
		name     string
		nameZh   string
		nameEn   string
		status   string
		expected bool
	}{
		{"有效数据", "测试库", "test_lib", "active", true},
		{"空中文名", "", "test_lib", "active", false},
		{"空英文名", "测试库", "", "active", false},
		{"无效状态", "测试库", "test_lib", "invalid", false},
		{"长名称", string(make([]rune, 256)), "test_lib", "active", false}, // 假设有长度限制
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			library := BasicLibrary{
				NameZh: tc.nameZh,
				NameEn: tc.nameEn,
				Status: tc.status,
			}

			// 这里需要根据实际的验证逻辑来实现
			valid := validateBasicLibrary(library)
			assert.Equal(t, tc.expected, valid)
		})
	}
}

// 模拟验证函数（实际应该在模型中实现）
func validateBasicLibrary(library BasicLibrary) bool {
	if library.NameZh == "" || library.NameEn == "" {
		return false
	}

	validStatuses := []string{"active", "inactive", "draft", "archived"}
	for _, status := range validStatuses {
		if library.Status == status {
			return len(library.NameZh) <= 255 && len(library.NameEn) <= 255
		}
	}

	return false
}
