/*
 * @module service/governance/tests/quality_task_integration_test
 * @description 数据质量检测任务集成测试
 * @architecture 测试层
 * @documentReference ai_docs/quality_task_integration_test_guide.md
 * @stateFlow 创建测试表 -> 插入测试数据 -> 创建任务 -> 执行任务 -> 验证结果 -> 清理数据
 * @rules 使用内置规则模板，确保所有功能正常工作
 * @dependencies testing, datahub-service/service/governance, datahub-service/service/models
 */

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"datahub-service/service/database"
	"datahub-service/service/governance"
	"datahub-service/service/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	baseURL     = "http://localhost:8080/swagger/datahub-service"
	testSchema  = "test_quality"
	testTable   = "users"
	testTimeout = 30 * time.Second
)

// TestQualityTaskIntegration 完整的质量检测任务集成测试
func TestQualityTaskIntegration(t *testing.T) {
	// 跳过集成测试（除非设置了环境变量）
	// if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
	// 	t.Skip("跳过集成测试，设置 RUN_INTEGRATION_TESTS=true 来运行")
	// }

	ctx := context.Background()

	// 1. 连接数据库
	db := setupDatabase(t)
	defer cleanupDatabase(t, db)

	// 2. 创建测试表和数据
	setupTestTable(t, db)
	defer cleanupTestTable(t, db)

	// 3. 获取内置规则模板ID
	templateIDs := getBuiltinTemplateIDs(t, db)
	require.NotEmpty(t, templateIDs, "应该有内置规则模板")

	// 4. 创建质量检测任务
	taskID := createQualityTask(t, templateIDs)
	require.NotEmpty(t, taskID, "任务ID不能为空")

	// 5. 执行任务
	executionID := executeTask(t, taskID)
	require.NotEmpty(t, executionID, "执行ID不能为空")

	// 6. 等待任务完成
	waitForTaskCompletion(t, ctx, db, taskID, testTimeout)

	// 7. 验证执行结果
	verifyExecutionResults(t, db, taskID)

	// 8. 验证问题记录
	verifyIssueRecords(t, taskID)

	// 9. 测试问题记录查询过滤
	testIssueRecordFilters(t, taskID)

	// 10. 测试任务更新
	testTaskUpdate(t, taskID, templateIDs)

	// 11. 清理任务
	deleteTask(t, taskID)
}

// setupDatabase 设置数据库连接
func setupDatabase(t *testing.T) *gorm.DB {
	dsn := "host=localhost port=5432 user=supabase_admin password=things2024 dbname=postgres sslmode=disable search_path=public TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "数据库连接失败")

	// 运行迁移
	err = database.AutoMigrate(db)
	require.NoError(t, err, "数据库迁移失败")

	return db
}

// cleanupDatabase 清理数据库连接
func cleanupDatabase(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// setupTestTable 创建测试表和数据
func setupTestTable(t *testing.T, db *gorm.DB) {
	// 创建测试schema
	err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", testSchema)).Error
	require.NoError(t, err, "创建测试schema失败")

	// 创建测试表
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) NOT NULL,
			email VARCHAR(100),
			mobile VARCHAR(20),
			age INTEGER,
			status VARCHAR(20),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, testSchema, testTable)
	err = db.Exec(createTableSQL).Error
	require.NoError(t, err, "创建测试表失败")

	// 插入测试数据
	insertDataSQL := fmt.Sprintf(`
		INSERT INTO %s.%s (username, email, mobile, age, status) VALUES
			('user1', 'user1@example.com', '13812345678', 25, 'active'),
			('user2', 'user2@example.com', '13987654321', 30, 'active'),
			('user3', NULL, NULL, 28, 'inactive'),
			('user4', 'invalid-email', '12345678901', 35, 'active'),
			('user5', 'user5@example.com', '', 15, 'active'),
			('user6', 'user6@example.com', '19900001111', 200, 'active'),
			('user7', 'user7@example.com', '+8613800138000', 40, 'suspended'),
			('', 'user8@example.com', '13611112222', 32, 'active'),
			('user9', '', '13711113333', 27, 'active'),
			('user10', 'user10@example.com', 'abcdefghijk', 29, 'active')
	`, testSchema, testTable)
	err = db.Exec(insertDataSQL).Error
	require.NoError(t, err, "插入测试数据失败")

	t.Logf("测试表创建成功: %s.%s", testSchema, testTable)
}

// cleanupTestTable 清理测试表
func cleanupTestTable(t *testing.T, db *gorm.DB) {
	// 删除测试表
	dropTableSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", testSchema, testTable)
	err := db.Exec(dropTableSQL).Error
	if err != nil {
		t.Logf("删除测试表失败: %v", err)
	}

	// 删除测试schema
	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", testSchema)
	err = db.Exec(dropSchemaSQL).Error
	if err != nil {
		t.Logf("删除测试schema失败: %v", err)
	}

	t.Log("测试表清理完成")
}

// getBuiltinTemplateIDs 获取内置规则模板ID
func getBuiltinTemplateIDs(t *testing.T, db *gorm.DB) map[string]string {
	templates := make(map[string]string)

	// 查询内置规则模板
	var qualityTemplates []models.QualityRuleTemplate
	err := db.Where("is_built_in = ?", true).Find(&qualityTemplates).Error
	require.NoError(t, err, "查询内置规则模板失败")

	for _, template := range qualityTemplates {
		templates[template.Name] = template.ID
		t.Logf("内置规则模板: %s -> %s", template.Name, template.ID)
	}

	return templates
}

// createQualityTask 创建质量检测任务
func createQualityTask(t *testing.T, templateIDs map[string]string) string {
	// 构建任务创建请求
	requestBody := map[string]interface{}{
		"name":               "用户表质量检测-集成测试",
		"description":        "测试用户表的数据质量（包含username、email、mobile字段检测）",
		"task_type":          "scheduled",
		"target_object_id":   "test-users-table",
		"target_object_type": "table",
		"target_schema":      testSchema,
		"target_table":       testTable,
		"field_rules": []map[string]interface{}{
			{
				"field_name":       "username",
				"rule_template_id": templateIDs["字段完整性检查模板"],
				"runtime_config": map[string]interface{}{
					"check_nullable":  false,
					"trim_whitespace": true,
					"case_sensitive":  false,
				},
				"threshold":  map[string]interface{}{},
				"is_enabled": true,
				"priority":   100,
			},
			{
				"field_name":       "email",
				"rule_template_id": templateIDs["字段完整性检查模板"],
				"runtime_config": map[string]interface{}{
					"check_nullable":  false,
					"trim_whitespace": true,
				},
				"threshold":  map[string]interface{}{},
				"is_enabled": true,
				"priority":   90,
			},
			{
				"field_name":       "email",
				"rule_template_id": templateIDs["邮箱格式准确性检查"],
				"runtime_config":   map[string]interface{}{},
				"threshold": map[string]interface{}{
					"pattern": "@",
				},
				"is_enabled": true,
				"priority":   85,
			},
			{
				"field_name":       "mobile",
				"rule_template_id": templateIDs["手机号有效性检查"],
				"runtime_config": map[string]interface{}{
					"allow_international": false,
				},
				"threshold":  map[string]interface{}{},
				"is_enabled": true,
				"priority":   80,
			},
			{
				"field_name":       "mobile",
				"rule_template_id": templateIDs["字段完整性检查模板"],
				"runtime_config": map[string]interface{}{
					"check_nullable":  false,
					"trim_whitespace": true,
				},
				"threshold":  map[string]interface{}{},
				"is_enabled": true,
				"priority":   75,
			},
		},
		"schedule_config": map[string]interface{}{
			"type": "manual",
		},
		"notification_config": map[string]interface{}{
			"enabled": false,
		},
		"priority":   50,
		"is_enabled": true,
	}

	// 发送HTTP请求
	jsonData, err := json.Marshal(requestBody)
	require.NoError(t, err, "JSON序列化失败")

	url := fmt.Sprintf("%s/data-quality/tasks", baseURL)
	fmt.Println("url", url)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err, "创建任务请求失败")
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应失败")

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err, "解析响应失败")

	t.Logf("创建任务响应: %s", string(body))

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.StatusCode, "HTTP状态码应为200")
	assert.Equal(t, float64(0), response["status"], "响应status应为0（成功）")

	// 提取任务ID
	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "响应data应该是对象")

	taskID, ok := data["id"].(string)
	require.True(t, ok && taskID != "", "任务ID不能为空")

	t.Logf("任务创建成功: %s", taskID)
	return taskID
}

// executeTask 执行任务
func executeTask(t *testing.T, taskID string) string {
	url := fmt.Sprintf("%s/data-quality/tasks/%s/start", baseURL, taskID)
	resp, err := http.Post(url, "application/json", nil)
	require.NoError(t, err, "启动任务请求失败")
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应失败")

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err, "解析响应失败")

	t.Logf("执行任务响应: %s", string(body))

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.StatusCode, "HTTP状态码应为200")

	// 提取执行ID
	data, ok := response["data"].(map[string]interface{})
	if ok {
		if executionID, ok := data["id"].(string); ok {
			t.Logf("任务执行已启动: %s", executionID)
			return executionID
		}
	}

	return ""
}

// waitForTaskCompletion 等待任务完成
func waitForTaskCompletion(t *testing.T, ctx context.Context, db *gorm.DB, taskID string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("等待任务完成超时")
			return
		case <-ticker.C:
			var task models.QualityTask
			err := db.First(&task, "id = ?", taskID).Error
			require.NoError(t, err, "查询任务失败")

			t.Logf("任务状态: %s", task.Status)

			if task.Status == "completed" || task.Status == "completed_with_issues" || task.Status == "failed" {
				t.Logf("任务完成，状态: %s", task.Status)
				return
			}
		}
	}
}

// verifyExecutionResults 验证执行结果
func verifyExecutionResults(t *testing.T, db *gorm.DB, taskID string) {
	// 查询任务
	var task models.QualityTask
	err := db.First(&task, "id = ?", taskID).Error
	require.NoError(t, err, "查询任务失败")

	t.Logf("任务执行统计:")
	t.Logf("  执行次数: %d", task.ExecutionCount)
	t.Logf("  成功次数: %d", task.SuccessCount)
	t.Logf("  失败次数: %d", task.FailureCount)

	// 验证执行次数
	assert.Equal(t, int64(1), task.ExecutionCount, "执行次数应为1")

	// 查询执行记录
	var executions []models.QualityTaskExecution
	err = db.Where("task_id = ?", taskID).Order("created_at DESC").Find(&executions).Error
	require.NoError(t, err, "查询执行记录失败")

	require.NotEmpty(t, executions, "应该有执行记录")

	execution := executions[0]
	t.Logf("执行记录详情:")
	t.Logf("  执行ID: %s", execution.ID)
	t.Logf("  状态: %s", execution.Status)
	t.Logf("  总检查数: %d", execution.TotalRulesExecuted)
	t.Logf("  通过数: %d", execution.PassedRules)
	t.Logf("  失败数: %d", execution.FailedRules)
	t.Logf("  总体得分: %.2f", execution.OverallScore)
	t.Logf("  持续时间: %dms", execution.Duration)

	// 验证执行状态
	assert.Contains(t, []string{"completed", "completed_with_issues"}, execution.Status, "执行状态应该是完成")

	// 验证有检查执行
	assert.Greater(t, execution.TotalRulesExecuted, 0, "应该有规则执行")

	// 验证有问题数据（测试数据中故意包含问题）
	assert.Greater(t, execution.FailedRules, 0, "应该检测到问题数据")

	// 验证得分在合理范围
	assert.Greater(t, execution.OverallScore, 0.0, "总体得分应该大于0")
	assert.LessOrEqual(t, execution.OverallScore, 1.0, "总体得分应该小于等于1")
}

// verifyIssueRecords 验证问题记录
func verifyIssueRecords(t *testing.T, taskID string) {
	// 查询问题记录
	url := fmt.Sprintf("%s/data-quality/tasks/%s/issue-records", baseURL, taskID)
	resp, err := http.Get(url)
	require.NoError(t, err, "查询问题记录请求失败")
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应失败")

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err, "解析响应失败")

	t.Logf("问题记录响应: %s", string(body))

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.StatusCode, "HTTP状态码应为200")

	// 提取问题记录列表
	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "响应data应该是对象")

	list, ok := data["list"].([]interface{})
	require.True(t, ok, "应该有问题记录列表")

	t.Logf("问题记录总数: %d", len(list))

	// 验证有问题记录
	assert.Greater(t, len(list), 0, "应该有问题记录")

	// 打印前几条问题记录
	for i, item := range list {
		if i >= 5 {
			break
		}
		record, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		t.Logf("问题记录 %d:", i+1)
		t.Logf("  字段名: %v", record["field_name"])
		t.Logf("  记录标识: %v", record["record_identifier"])
		t.Logf("  问题类型: %v", record["issue_type"])
		t.Logf("  问题描述: %v", record["issue_description"])
		t.Logf("  字段值: %v", record["field_value"])
		t.Logf("  严重程度: %v", record["severity"])
	}
}

// testIssueRecordFilters 测试问题记录过滤
func testIssueRecordFilters(t *testing.T, taskID string) {
	t.Run("按字段名过滤-username", func(t *testing.T) {
		url := fmt.Sprintf("%s/data-quality/tasks/%s/issue-records?field_name=username", baseURL, taskID)
		resp, err := http.Get(url)
		require.NoError(t, err, "查询问题记录请求失败")
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		data, _ := response["data"].(map[string]interface{})
		list, _ := data["list"].([]interface{})

		t.Logf("username字段问题记录数: %d", len(list))

		// 验证所有记录都是username字段
		for _, item := range list {
			record, ok := item.(map[string]interface{})
			if ok {
				assert.Equal(t, "username", record["field_name"], "应该都是username字段")
			}
		}
	})

	t.Run("按字段名过滤-mobile", func(t *testing.T) {
		url := fmt.Sprintf("%s/data-quality/tasks/%s/issue-records?field_name=mobile", baseURL, taskID)
		resp, err := http.Get(url)
		require.NoError(t, err, "查询问题记录请求失败")
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		data, _ := response["data"].(map[string]interface{})
		list, _ := data["list"].([]interface{})

		t.Logf("mobile字段问题记录数: %d", len(list))

		// 打印mobile字段的问题记录
		for i, item := range list {
			if i >= 5 {
				break
			}
			record, ok := item.(map[string]interface{})
			if ok {
				t.Logf("  mobile问题 %d: 值='%v', 描述='%v'", i+1, record["field_value"], record["issue_description"])
			}
		}
	})

	t.Run("按字段名过滤-email", func(t *testing.T) {
		url := fmt.Sprintf("%s/data-quality/tasks/%s/issue-records?field_name=email", baseURL, taskID)
		resp, err := http.Get(url)
		require.NoError(t, err, "查询问题记录请求失败")
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		data, _ := response["data"].(map[string]interface{})
		list, _ := data["list"].([]interface{})

		t.Logf("email字段问题记录数: %d", len(list))
	})

	t.Run("按严重程度过滤", func(t *testing.T) {
		url := fmt.Sprintf("%s/data-quality/tasks/%s/issue-records?severity=critical", baseURL, taskID)
		resp, err := http.Get(url)
		require.NoError(t, err, "查询问题记录请求失败")
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		data, _ := response["data"].(map[string]interface{})
		list, _ := data["list"].([]interface{})

		t.Logf("critical级别问题记录数: %d", len(list))
	})
}

// testTaskUpdate 测试任务更新
func testTaskUpdate(t *testing.T, taskID string, templateIDs map[string]string) {
	// 构建更新请求
	requestBody := map[string]interface{}{
		"name":        "用户表质量检测-已更新",
		"description": "更新后的任务描述",
		"priority":    60,
	}

	jsonData, err := json.Marshal(requestBody)
	require.NoError(t, err, "JSON序列化失败")

	// 发送更新请求
	url := fmt.Sprintf("%s/data-quality/tasks/%s", baseURL, taskID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	require.NoError(t, err, "创建请求失败")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "更新任务请求失败")
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应失败")

	t.Logf("更新任务响应: %s", string(body))

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.StatusCode, "HTTP状态码应为200")

	// 验证任务已更新
	getURL := fmt.Sprintf("%s/data-quality/tasks/%s", baseURL, taskID)
	getResp, err := http.Get(getURL)
	require.NoError(t, err, "查询任务请求失败")
	defer getResp.Body.Close()

	var getResponse map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&getResponse)

	data, _ := getResponse["data"].(map[string]interface{})
	assert.Equal(t, "用户表质量检测-已更新", data["name"], "任务名称应该已更新")
}

// deleteTask 删除任务
func deleteTask(t *testing.T, taskID string) {
	url := fmt.Sprintf("%s/data-quality/tasks/%s", baseURL, taskID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err, "创建删除请求失败")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "删除任务请求失败")
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应失败")

	t.Logf("删除任务响应: %s", string(body))

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.StatusCode, "HTTP状态码应为200")

	t.Logf("任务删除成功: %s", taskID)
}

// TestQualityScheduler 测试质量检测调度器
func TestQualityScheduler(t *testing.T) {
	// 跳过调度器测试（需要长时间运行）
	t.Skip("跳过调度器测试，需要长时间运行")

	db := setupDatabase(t)
	defer cleanupDatabase(t, db)

	setupTestTable(t, db)
	defer cleanupTestTable(t, db)

	templateIDs := getBuiltinTemplateIDs(t, db)

	// 创建间隔调度任务（每2分钟执行一次）
	requestBody := map[string]interface{}{
		"name":               "用户表定时质量检测",
		"description":        "每2分钟执行一次",
		"task_type":          "scheduled",
		"target_object_id":   "test-users-table",
		"target_object_type": "table",
		"target_schema":      testSchema,
		"target_table":       testTable,
		"field_rules": []map[string]interface{}{
			{
				"field_name":       "username",
				"rule_template_id": templateIDs["字段完整性检查模板"],
				"runtime_config":   map[string]interface{}{},
				"threshold":        map[string]interface{}{},
				"is_enabled":       true,
				"priority":         100,
			},
		},
		"schedule_config": map[string]interface{}{
			"type":     "interval",
			"interval": 120, // 2分钟
		},
		"notification_config": map[string]interface{}{
			"enabled": false,
		},
		"priority":   50,
		"is_enabled": true,
	}

	jsonData, _ := json.Marshal(requestBody)
	url := fmt.Sprintf("%s/data-quality/tasks", baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err, "创建调度任务失败")
	defer resp.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	data, _ := response["data"].(map[string]interface{})
	taskID, _ := data["id"].(string)

	t.Logf("调度任务创建成功: %s", taskID)

	// 等待5分钟，验证任务是否被调度执行
	time.Sleep(5 * time.Minute)

	// 查询执行记录
	var task models.QualityTask
	db.First(&task, "id = ?", taskID)

	t.Logf("调度任务执行统计:")
	t.Logf("  执行次数: %d", task.ExecutionCount)

	// 清理
	deleteTask(t, taskID)
}

// TestInternationalPhoneNumber 测试国际号码支持
func TestInternationalPhoneNumber(t *testing.T) {
	ctx := context.Background()

	// 1. 连接数据库
	db := setupDatabase(t)
	defer cleanupDatabase(t, db)

	// 2. 创建测试表和数据（包含国际号码）
	setupTestTable(t, db)
	defer cleanupTestTable(t, db)

	// 3. 获取内置规则模板ID
	templateIDs := getBuiltinTemplateIDs(t, db)

	// 4. 创建支持国际号码的质量检测任务
	requestBody := map[string]interface{}{
		"name":               "国际手机号质量检测",
		"description":        "测试国际手机号验证",
		"task_type":          "scheduled",
		"target_object_id":   "test-users-table",
		"target_object_type": "table",
		"target_schema":      testSchema,
		"target_table":       testTable,
		"field_rules": []map[string]interface{}{
			{
				"field_name":       "mobile",
				"rule_template_id": templateIDs["手机号有效性检查"],
				"runtime_config": map[string]interface{}{
					"check_nullable":  true,
					"trim_whitespace": true,
					"case_sensitive":  false,
					"custom_params": map[string]any{
						"allow_international": true, // 允许国际号码
					},
				},
				"threshold":  map[string]interface{}{},
				"is_enabled": true,
				"priority":   100,
			},
		},
		"schedule_config": map[string]interface{}{
			"type": "manual",
		},
		"notification_config": map[string]interface{}{
			"enabled": false,
		},
		"priority":   50,
		"is_enabled": true,
	}

	jsonData, _ := json.Marshal(requestBody)
	url := fmt.Sprintf("%s/data-quality/tasks", baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err, "创建任务请求失败")
	defer resp.Body.Close()

	var response map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &response)

	data, _ := response["data"].(map[string]interface{})
	taskID, _ := data["id"].(string)

	t.Logf("国际号码任务创建成功: %s", taskID)

	// 5. 执行任务
	executeTask(t, taskID)

	// 6. 等待任务完成
	waitForTaskCompletion(t, ctx, db, taskID, testTimeout)

	// 7. 验证执行结果
	verifyExecutionResults(t, db, taskID)

	// 8. 查看问题记录（国际号码应该被接受）
	issueURL := fmt.Sprintf("%s/data-quality/tasks/%s/issue-records?field_name=mobile", baseURL, taskID)
	issueResp, err := http.Get(issueURL)
	require.NoError(t, err, "查询问题记录请求失败")
	defer issueResp.Body.Close()

	var issueResponse map[string]interface{}
	issueBody, _ := io.ReadAll(issueResp.Body)
	json.Unmarshal(issueBody, &issueResponse)

	issueData, _ := issueResponse["data"].(map[string]interface{})
	issueList, _ := issueData["list"].([]interface{})

	t.Logf("国际号码任务-mobile字段问题记录数: %d", len(issueList))
	for i, item := range issueList {
		if i >= 3 {
			break
		}
		record, _ := item.(map[string]interface{})
		t.Logf("  问题 %d: 值='%v', 描述='%v'", i+1, record["field_value"], record["issue_description"])
	}

	// 9. 清理任务
	deleteTask(t, taskID)
}

// TestGovernanceService 测试治理服务直接调用
func TestGovernanceService(t *testing.T) {
	db := setupDatabase(t)
	defer cleanupDatabase(t, db)

	// 创建治理服务
	govService := governance.NewGovernanceService(db)
	_ = govService // 避免未使用警告

	// 测试获取规则模板
	t.Run("GetQualityRuleTemplates", func(t *testing.T) {
		templates, total, err := govService.GetTemplateService().GetQualityRuleTemplates(1, 10, "", "", nil)
		require.NoError(t, err, "获取规则模板失败")
		assert.Greater(t, total, int64(0), "应该有规则模板")
		assert.NotEmpty(t, templates, "模板列表不应为空")

		t.Logf("规则模板总数: %d", total)
		for _, template := range templates {
			t.Logf("  - %s (%s)", template.Name, template.Type)
		}
	})

	// 测试计算下次执行时间
	t.Run("CalculateNextExecution", func(t *testing.T) {
		// Cron调度
		cronConfig := governance.ScheduleConfigRequest{
			Type:     "cron",
			CronExpr: "0 0 * * * *", // 每小时
		}
		nextTime, err := govService.CalculateNextExecution(cronConfig, nil)
		require.NoError(t, err, "计算Cron下次执行时间失败")
		require.NotNil(t, nextTime, "下次执行时间不应为空")
		t.Logf("Cron下次执行时间: %s", nextTime.Format("2006-01-02 15:04:05"))

		// 间隔调度
		intervalConfig := governance.ScheduleConfigRequest{
			Type:     "interval",
			Interval: 300, // 5分钟
		}
		nextTime, err = govService.CalculateNextExecution(intervalConfig, nil)
		require.NoError(t, err, "计算Interval下次执行时间失败")
		require.NotNil(t, nextTime, "下次执行时间不应为空")
		t.Logf("Interval下次执行时间: %s", nextTime.Format("2006-01-02 15:04:05"))

		// 手动调度
		manualConfig := governance.ScheduleConfigRequest{
			Type: "manual",
		}
		nextTime, err = govService.CalculateNextExecution(manualConfig, nil)
		require.NoError(t, err, "计算Manual下次执行时间失败")
		require.Nil(t, nextTime, "手动调度不应有下次执行时间")
	})
}
