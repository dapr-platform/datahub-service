/*
 * @module service/interface_executor/data_sync_engine_test
 * @description DataSyncEngine的单元测试
 * @architecture 测试驱动开发 - 确保数据同步引擎的各种策略正常工作
 * @documentReference design.md
 * @stateFlow 测试准备 -> 数据构造 -> 执行测试 -> 结果验证 -> 清理资源
 * @rules 测试用例需要覆盖正常流程、边界条件和异常情况
 * @dependencies testing, testify, gorm, sqlite
 * @refs data_sync_engine.go
 */

package interface_executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DataSyncEngineTestSuite 数据同步引擎测试套件
type DataSyncEngineTestSuite struct {
	suite.Suite
	db         *gorm.DB
	syncEngine *DataSyncEngine
	ctx        context.Context
}

// SetupSuite 设置测试套件
func (suite *DataSyncEngineTestSuite) SetupSuite() {
	// 使用内存SQLite数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	suite.db = db
	suite.syncEngine = NewDataSyncEngine(db)
	suite.ctx = context.Background()

	// 创建测试表
	err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	suite.Require().NoError(err)
}

// TearDownSuite 清理测试套件
func (suite *DataSyncEngineTestSuite) TearDownSuite() {
	// 清理测试数据
	suite.db.Exec("DROP TABLE IF EXISTS test_table")
}

// SetupTest 设置每个测试
func (suite *DataSyncEngineTestSuite) SetupTest() {
	// 清空测试表
	suite.db.Exec("DELETE FROM test_table")
}

// TestNewDataSyncEngine 测试创建数据同步引擎
func (suite *DataSyncEngineTestSuite) TestNewDataSyncEngine() {
	engine := NewDataSyncEngine(suite.db)

	assert.NotNil(suite.T(), engine)
	assert.NotNil(suite.T(), engine.db)
	assert.NotNil(suite.T(), engine.schemaService)
}

// TestValidateTableTarget 测试表目标验证
func (suite *DataSyncEngineTestSuite) TestValidateTableTarget() {
	testCases := []struct {
		name        string
		target      TableTarget
		expectError bool
	}{
		{
			name: "有效的表目标",
			target: TableTarget{
				TableName:   "test_table",
				PrimaryKeys: []string{"id"},
			},
			expectError: false,
		},
		{
			name: "空表名",
			target: TableTarget{
				TableName:   "",
				PrimaryKeys: []string{"id"},
			},
			expectError: true,
		},
		{
			name: "空主键",
			target: TableTarget{
				TableName:   "test_table",
				PrimaryKeys: []string{},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.syncEngine.validateTableTarget(tc.target)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExecuteFullSync 测试全量同步
func (suite *DataSyncEngineTestSuite) TestExecuteFullSync() {
	// 准备测试数据
	testData := []map[string]interface{}{
		{"id": 1, "name": "test1", "value": 100},
		{"id": 2, "name": "test2", "value": 200},
		{"id": 3, "name": "test3", "value": 300},
	}

	target := TableTarget{
		TableName:   "test_table",
		PrimaryKeys: []string{"id"},
		Columns:     []string{"id", "name", "value"},
	}

	// 执行全量同步
	result, err := suite.syncEngine.ExecuteFullSync(suite.ctx, testData, target)

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), DataFullSync, result.Strategy)
	assert.Equal(suite.T(), int64(3), result.RecordsCount)
	assert.True(suite.T(), result.Duration > 0)
	assert.NotNil(suite.T(), result.LastSyncTime)

	// 验证数据库中的数据
	var count int64
	err = suite.db.Table("test_table").Count(&count).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), count)
}

// TestExecuteIncrementalSync 测试增量同步
func (suite *DataSyncEngineTestSuite) TestExecuteIncrementalSync() {
	// 先插入一些基础数据
	suite.db.Exec("INSERT INTO test_table (id, name, value) VALUES (1, 'existing', 50)")

	// 准备增量数据（包含更新和新增）
	testData := []map[string]interface{}{
		{"id": 1, "name": "updated", "value": 150}, // 更新现有记录
		{"id": 4, "name": "new", "value": 400},     // 新增记录
	}

	target := TableTarget{
		TableName:   "test_table",
		PrimaryKeys: []string{"id"},
		Columns:     []string{"id", "name", "value"},
	}

	// 执行增量同步
	result, err := suite.syncEngine.ExecuteIncrementalSync(suite.ctx, testData, target)

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), DataIncrementalSync, result.Strategy)
	assert.Equal(suite.T(), int64(2), result.RecordsCount)
	assert.True(suite.T(), result.Duration > 0)
}

// TestExecuteRealtimeSync 测试实时同步
func (suite *DataSyncEngineTestSuite) TestExecuteRealtimeSync() {
	// 准备实时数据
	testData := []map[string]interface{}{
		{"id": 5, "name": "realtime", "value": 500},
	}

	target := TableTarget{
		TableName:   "test_table",
		PrimaryKeys: []string{"id"},
		Columns:     []string{"id", "name", "value"},
	}

	// 执行实时同步
	result, err := suite.syncEngine.ExecuteRealtimeSync(suite.ctx, testData, target)

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), DataRealtimeSync, result.Strategy)
	assert.Equal(suite.T(), int64(1), result.RecordsCount)
	assert.True(suite.T(), result.Duration > 0)
}

// TestExecuteSyncWithInvalidStrategy 测试无效的同步策略
func (suite *DataSyncEngineTestSuite) TestExecuteSyncWithInvalidStrategy() {
	testData := []map[string]interface{}{
		{"id": 1, "name": "test", "value": 100},
	}

	target := TableTarget{
		TableName:   "test_table",
		PrimaryKeys: []string{"id"},
	}

	// 使用无效的同步策略
	result, err := suite.syncEngine.ExecuteSync(suite.ctx, "invalid", testData, target)

	// 验证错误
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "不支持的同步策略")
	assert.NotNil(suite.T(), result)
}

// TestExecuteSyncWithEmptyData 测试空数据同步
func (suite *DataSyncEngineTestSuite) TestExecuteSyncWithEmptyData() {
	var testData []map[string]interface{}

	target := TableTarget{
		TableName:   "test_table",
		PrimaryKeys: []string{"id"},
	}

	// 执行全量同步
	result, err := suite.syncEngine.ExecuteFullSync(suite.ctx, testData, target)

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), int64(0), result.RecordsCount)
}

// TestGetColumnsFromData 测试从数据中提取列名
func (suite *DataSyncEngineTestSuite) TestGetColumnsFromData() {
	testCases := []struct {
		name     string
		record   map[string]interface{}
		target   TableTarget
		expected []string
	}{
		{
			name:   "使用目标表指定的列",
			record: map[string]interface{}{"id": 1, "name": "test", "extra": "ignore"},
			target: TableTarget{
				Columns: []string{"id", "name"},
			},
			expected: []string{"id", "name"},
		},
		{
			name:     "从记录中提取所有列",
			record:   map[string]interface{}{"id": 1, "name": "test"},
			target:   TableTarget{},
			expected: []string{"id", "name"}, // 注意：map遍历顺序不保证
		},
		{
			name:   "使用字段映射",
			record: map[string]interface{}{"user_id": 1, "user_name": "test"},
			target: TableTarget{
				Mapping: map[string]string{
					"user_id":   "id",
					"user_name": "name",
				},
			},
			expected: []string{"id", "name"},
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			columns := suite.syncEngine.getColumnsFromData(tc.record, tc.target)

			if len(tc.target.Columns) > 0 {
				// 如果指定了列，应该完全匹配
				assert.Equal(t, tc.expected, columns)
			} else {
				// 如果没有指定列，检查是否包含所有期望的列
				assert.Len(t, columns, len(tc.expected))
				for _, expectedCol := range tc.expected {
					assert.Contains(t, columns, expectedCol)
				}
			}
		})
	}
}

// TestTruncateTable 测试清空表
func (suite *DataSyncEngineTestSuite) TestTruncateTable() {
	// 先插入一些数据
	suite.db.Exec("INSERT INTO test_table (id, name, value) VALUES (1, 'test', 100)")

	target := TableTarget{
		TableName: "test_table",
	}

	// 在事务中清空表
	tx := suite.db.Begin()
	err := suite.syncEngine.truncateTable(tx, target)
	tx.Commit()

	// 验证结果
	assert.NoError(suite.T(), err)

	// 检查表是否为空
	var count int64
	err = suite.db.Table("test_table").Count(&count).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), count)
}

// TestBatchInsertData 测试批量插入数据
func (suite *DataSyncEngineTestSuite) TestBatchInsertData() {
	// 准备大量测试数据
	testData := make([]map[string]interface{}, 2500) // 超过批次大小1000
	for i := 0; i < 2500; i++ {
		testData[i] = map[string]interface{}{
			"id":    i + 1,
			"name":  fmt.Sprintf("test%d", i+1),
			"value": (i + 1) * 10,
		}
	}

	target := TableTarget{
		TableName: "test_table",
		Columns:   []string{"id", "name", "value"},
	}

	// 在事务中批量插入
	tx := suite.db.Begin()
	insertedCount, errors := suite.syncEngine.batchInsertData(tx, testData, target)
	tx.Commit()

	// 验证结果
	assert.Equal(suite.T(), int64(2500), insertedCount)
	assert.Empty(suite.T(), errors)

	// 验证数据库中的数据
	var count int64
	err := suite.db.Table("test_table").Count(&count).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2500), count)
}

// 运行测试套件
func TestDataSyncEngineTestSuite(t *testing.T) {
	suite.Run(t, new(DataSyncEngineTestSuite))
}

// 基准测试
func BenchmarkFullSync(b *testing.B) {
	// 设置测试环境
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.Exec(`CREATE TABLE bench_table (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)`)

	syncEngine := NewDataSyncEngine(db)
	ctx := context.Background()

	// 准备测试数据
	testData := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		testData[i] = map[string]interface{}{
			"id":    i + 1,
			"name":  fmt.Sprintf("bench%d", i+1),
			"value": i * 10,
		}
	}

	target := TableTarget{
		TableName:   "bench_table",
		PrimaryKeys: []string{"id"},
		Columns:     []string{"id", "name", "value"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 清空表
		db.Exec("DELETE FROM bench_table")

		// 执行全量同步
		_, err := syncEngine.ExecuteFullSync(ctx, testData, target)
		if err != nil {
			b.Fatal(err)
		}
	}
}
