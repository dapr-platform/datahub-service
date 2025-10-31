/*
 * @module service/governance/quality_task_service
 * @description 数据质量检测任务服务，负责任务的创建、执行、查询等
 * @architecture 分层架构 - 服务层
 * @documentReference ai_docs/data_governance_task_req.md
 * @stateFlow 任务创建 -> 任务激活 -> 调度执行 -> 结果记录
 * @rules 每个任务针对单个表，支持字段级规则配置
 * @dependencies gorm.io/gorm, service/models, github.com/robfig/cron/v3
 * @refs quality_scheduler.go, governance_service.go
 */

package governance

import (
	"datahub-service/service/models"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// === 数据质量检测任务管理 ===

// CreateQualityTask 创建质量检测任务
func (s *GovernanceService) CreateQualityTask(req *CreateQualityTaskRequest) (*QualityTaskResponse, error) {
	// 验证字段规则
	if len(req.FieldRules) == 0 {
		return nil, errors.New("至少需要配置一个字段规则")
	}

	// 构建通知配置 JSONB (将数组包装为map以匹配JSONB类型)
	var recipients, channels models.JSONB
	if len(req.NotificationConfig.Recipients) > 0 {
		recipients = models.JSONB{"list": req.NotificationConfig.Recipients}
	}
	if len(req.NotificationConfig.Channels) > 0 {
		channels = models.JSONB{"list": req.NotificationConfig.Channels}
	}

	// 创建任务
	task := &models.QualityTask{
		Name:        req.Name,
		Description: req.Description,
		// 库和接口信息
		LibraryType: req.LibraryType,
		LibraryID:   req.LibraryID,
		InterfaceID: req.InterfaceID,
		// 目标表信息
		TargetSchema: req.TargetSchema,
		TargetTable:  req.TargetTable,
		Status:       "pending",
		Priority:     req.Priority,
		IsEnabled:    req.IsEnabled,
		// 调度配置
		ScheduleType:    req.ScheduleConfig.Type,
		CronExpression:  req.ScheduleConfig.CronExpr,
		IntervalSeconds: req.ScheduleConfig.Interval,
		ScheduledTime:   req.ScheduleConfig.StartTime,
		// 通知配置
		NotifyEnabled:   req.NotificationConfig.Enabled,
		NotifyOnSuccess: req.NotificationConfig.NotifyOnSuccess,
		NotifyOnFailure: req.NotificationConfig.NotifyOnFailure,
		Recipients:      recipients,
		NotifyChannels:  channels,
	}

	// 计算下次执行时间
	nextExec, err := s.CalculateNextExecution(req.ScheduleConfig, nil)
	if err == nil && nextExec != nil {
		task.NextExecution = nextExec
	}

	// 使用事务创建任务和字段规则
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 创建任务
		if err := tx.Create(task).Error; err != nil {
			return fmt.Errorf("创建任务失败: %w", err)
		}

		// 创建字段规则
		for _, fieldRule := range req.FieldRules {
			// 验证规则模板是否存在
			var ruleTemplate models.QualityRuleTemplate
			if err := tx.First(&ruleTemplate, "id = ?", fieldRule.RuleTemplateID).Error; err != nil {
				return fmt.Errorf("规则模板 %s 不存在: %w", fieldRule.RuleTemplateID, err)
			}

			// 构建运行时配置和阈值的JSONB
			runtimeConfigMap := map[string]interface{}{
				"check_nullable":  fieldRule.RuntimeConfig.CheckNullable,
				"trim_whitespace": fieldRule.RuntimeConfig.TrimWhitespace,
				"case_sensitive":  fieldRule.RuntimeConfig.CaseSensitive,
				"custom_params":   fieldRule.RuntimeConfig.CustomParams,
			}
			thresholdMap := make(map[string]interface{})
			if fieldRule.Threshold.MinValue != nil {
				thresholdMap["min_value"] = *fieldRule.Threshold.MinValue
			}
			if fieldRule.Threshold.MaxValue != nil {
				thresholdMap["max_value"] = *fieldRule.Threshold.MaxValue
			}
			if fieldRule.Threshold.MinLength != nil {
				thresholdMap["min_length"] = *fieldRule.Threshold.MinLength
			}
			if fieldRule.Threshold.MaxLength != nil {
				thresholdMap["max_length"] = *fieldRule.Threshold.MaxLength
			}
			if len(fieldRule.Threshold.AllowedValues) > 0 {
				thresholdMap["allowed_values"] = fieldRule.Threshold.AllowedValues
			}
			if fieldRule.Threshold.Pattern != "" {
				thresholdMap["pattern"] = fieldRule.Threshold.Pattern
			}
			if fieldRule.Threshold.CustomThreshold != nil {
				thresholdMap["custom_threshold"] = fieldRule.Threshold.CustomThreshold
			}

			taskFieldRule := &models.QualityTaskFieldRule{
				TaskID:         task.ID,
				FieldName:      fieldRule.FieldName,
				RuleTemplateID: fieldRule.RuleTemplateID,
				RuntimeConfig:  models.JSONB(runtimeConfigMap),
				Threshold:      models.JSONB(thresholdMap),
				IsEnabled:      fieldRule.IsEnabled,
				Priority:       fieldRule.Priority,
			}

			if err := tx.Create(taskFieldRule).Error; err != nil {
				return fmt.Errorf("创建字段规则失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 返回任务详情
	return s.GetQualityTaskByID(task.ID)
}

// buildTaskResponse 构建任务响应
func (s *GovernanceService) buildTaskResponse(task *models.QualityTask) (*QualityTaskResponse, error) {
	// 查询字段规则
	var fieldRules []models.QualityTaskFieldRule
	if err := s.db.Where("task_id = ?", task.ID).Order("priority DESC, created_at ASC").Find(&fieldRules).Error; err != nil {
		return nil, fmt.Errorf("查询字段规则失败: %w", err)
	}

	// 构建字段规则响应
	fieldRuleResponses := make([]FieldRuleResponse, 0, len(fieldRules))
	for _, fieldRule := range fieldRules {
		// 获取规则模板名称
		var ruleTemplate models.QualityRuleTemplate
		ruleName := "未知规则"
		if err := s.db.First(&ruleTemplate, "id = ?", fieldRule.RuleTemplateID).Error; err == nil {
			ruleName = ruleTemplate.Name
		}

		// 从JSONB map中提取运行时配置和阈值
		var runtimeConfig RuntimeConfig
		var threshold ThresholdConfig
		if fieldRule.RuntimeConfig != nil {
			if checkNullable, ok := fieldRule.RuntimeConfig["check_nullable"].(bool); ok {
				runtimeConfig.CheckNullable = checkNullable
			}
			if trimWhitespace, ok := fieldRule.RuntimeConfig["trim_whitespace"].(bool); ok {
				runtimeConfig.TrimWhitespace = trimWhitespace
			}
			if caseSensitive, ok := fieldRule.RuntimeConfig["case_sensitive"].(bool); ok {
				runtimeConfig.CaseSensitive = caseSensitive
			}
			if customParams, ok := fieldRule.RuntimeConfig["custom_params"].(map[string]any); ok {
				runtimeConfig.CustomParams = customParams
			}
		}
		if fieldRule.Threshold != nil {
			if minValue, ok := fieldRule.Threshold["min_value"].(float64); ok {
				threshold.MinValue = &minValue
			}
			if maxValue, ok := fieldRule.Threshold["max_value"].(float64); ok {
				threshold.MaxValue = &maxValue
			}
			if minLength, ok := fieldRule.Threshold["min_length"].(float64); ok {
				ml := int(minLength)
				threshold.MinLength = &ml
			}
			if maxLength, ok := fieldRule.Threshold["max_length"].(float64); ok {
				ml := int(maxLength)
				threshold.MaxLength = &ml
			}
			if allowedValues, ok := fieldRule.Threshold["allowed_values"].([]interface{}); ok {
				threshold.AllowedValues = make([]string, len(allowedValues))
				for i, v := range allowedValues {
					if str, ok := v.(string); ok {
						threshold.AllowedValues[i] = str
					}
				}
			}
			if pattern, ok := fieldRule.Threshold["pattern"].(string); ok {
				threshold.Pattern = pattern
			}
			if customThreshold, ok := fieldRule.Threshold["custom_threshold"].(map[string]interface{}); ok {
				threshold.CustomThreshold = customThreshold
			}
		}

		fieldRuleResponses = append(fieldRuleResponses, FieldRuleResponse{
			ID:             fieldRule.ID,
			FieldName:      fieldRule.FieldName,
			RuleTemplateID: fieldRule.RuleTemplateID,
			RuleName:       ruleName,
			RuntimeConfig:  runtimeConfig,
			Threshold:      threshold,
			IsEnabled:      fieldRule.IsEnabled,
			Priority:       fieldRule.Priority,
		})
	}

	// 构建调度配置响应
	scheduleConfigResp := ScheduleConfigResponse{
		Type:      task.ScheduleType,
		CronExpr:  task.CronExpression,
		Interval:  task.IntervalSeconds,
		StartTime: task.ScheduledTime,
	}

	// 构建通知配置响应
	var recipients, channels []string
	if task.Recipients != nil {
		if list, ok := task.Recipients["list"].([]interface{}); ok {
			for _, v := range list {
				if str, ok := v.(string); ok {
					recipients = append(recipients, str)
				}
			}
		}
	}
	if task.NotifyChannels != nil {
		if list, ok := task.NotifyChannels["list"].([]interface{}); ok {
			for _, v := range list {
				if str, ok := v.(string); ok {
					channels = append(channels, str)
				}
			}
		}
	}

	notificationConfigResp := NotificationConfigResponse{
		Enabled:         task.NotifyEnabled,
		NotifyOnSuccess: task.NotifyOnSuccess,
		NotifyOnFailure: task.NotifyOnFailure,
		Recipients:      recipients,
		Channels:        channels,
	}

	return &QualityTaskResponse{
		ID:                 task.ID,
		Name:               task.Name,
		Description:        task.Description,
		LibraryType:        task.LibraryType,
		LibraryID:          task.LibraryID,
		InterfaceID:        task.InterfaceID,
		TargetSchema:       task.TargetSchema,
		TargetTable:        task.TargetTable,
		Status:             task.Status,
		ExecutionCount:     task.ExecutionCount,
		SuccessCount:       task.SuccessCount,
		FailureCount:       task.FailureCount,
		LastExecuted:       task.LastExecuted,
		NextExecution:      task.NextExecution,
		FieldRules:         fieldRuleResponses,
		ScheduleConfig:     scheduleConfigResp,
		NotificationConfig: notificationConfigResp,
		Priority:           task.Priority,
		IsEnabled:          task.IsEnabled,
		CreatedAt:          task.CreatedAt,
		CreatedBy:          task.CreatedBy,
		UpdatedAt:          task.UpdatedAt,
		UpdatedBy:          task.UpdatedBy,
	}, nil
}

// GetQualityTasks 获取质量检测任务列表
func (s *GovernanceService) GetQualityTasks(page, pageSize int, status, libraryType, libraryID, interfaceID string) ([]QualityTaskResponse, int64, error) {
	query := s.db.Model(&models.QualityTask{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if libraryType != "" {
		query = query.Where("library_type = ?", libraryType)
	}
	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}
	if interfaceID != "" {
		query = query.Where("interface_id = ?", interfaceID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []models.QualityTask
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]QualityTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		resp, err := s.buildTaskResponse(&task)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, *resp)
	}

	return responses, total, nil
}

// GetQualityTaskByID 根据ID获取质量检测任务
func (s *GovernanceService) GetQualityTaskByID(id string) (*QualityTaskResponse, error) {
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return s.buildTaskResponse(&task)
}

// StartQualityTask 启动质量检测任务
func (s *GovernanceService) StartQualityTask(id string) (*QualityTaskExecutionResponse, error) {
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 检查任务状态
	if task.Status == "running" {
		return nil, errors.New("任务正在运行中")
	}

	// 创建执行记录
	execution := &models.QualityTaskExecution{
		TaskID:    id,
		StartTime: time.Now(),
		Status:    "running",
	}

	if err := s.db.Create(execution).Error; err != nil {
		return nil, err
	}

	// 更新任务状态
	if err := s.db.Model(&task).Updates(map[string]interface{}{
		"status":        "running",
		"last_executed": time.Now(),
	}).Error; err != nil {
		return nil, err
	}

	// 异步执行任务
	go s.executeQualityTask(execution)

	return &QualityTaskExecutionResponse{
		ID:        execution.ID,
		TaskID:    execution.TaskID,
		StartTime: execution.StartTime,
		Status:    execution.Status,
	}, nil
}

// StopQualityTask 停止质量检测任务
func (s *GovernanceService) StopQualityTask(id string) error {
	return s.db.Model(&models.QualityTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": "cancelled",
	}).Error
}

// UpdateQualityTask 更新质量检测任务
func (s *GovernanceService) UpdateQualityTask(id string, req *UpdateQualityTaskRequest) error {
	// 检查任务是否存在
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return err
	}

	// 检查任务是否正在运行
	if task.Status == "running" {
		return errors.New("正在运行的任务不能修改")
	}

	// 使用事务更新
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 构建任务更新数据
		updates := make(map[string]interface{})

		if req.Name != "" {
			updates["name"] = req.Name
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if req.Priority != nil {
			updates["priority"] = *req.Priority
		}
		if req.IsEnabled != nil {
			updates["is_enabled"] = *req.IsEnabled
		}

		// 更新调度配置
		if req.ScheduleConfig != nil {
			updates["schedule_type"] = req.ScheduleConfig.Type
			updates["cron_expression"] = req.ScheduleConfig.CronExpr
			updates["interval_seconds"] = req.ScheduleConfig.Interval
			updates["scheduled_time"] = req.ScheduleConfig.StartTime

			// 重新计算下次执行时间
			nextExec, err := s.CalculateNextExecution(*req.ScheduleConfig, nil)
			if err == nil && nextExec != nil {
				updates["next_execution"] = nextExec
			}
		}

		// 更新通知配置
		if req.NotificationConfig != nil {
			updates["notify_enabled"] = req.NotificationConfig.Enabled
			updates["notify_on_success"] = req.NotificationConfig.NotifyOnSuccess
			updates["notify_on_failure"] = req.NotificationConfig.NotifyOnFailure

			if len(req.NotificationConfig.Recipients) > 0 {
				updates["recipients"] = models.JSONB{"list": req.NotificationConfig.Recipients}
			}
			if len(req.NotificationConfig.Channels) > 0 {
				updates["notify_channels"] = models.JSONB{"list": req.NotificationConfig.Channels}
			}
		}

		// 更新任务
		if len(updates) > 0 {
			if err := tx.Model(&models.QualityTask{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}

		// 更新字段规则
		if len(req.FieldRules) > 0 {
			// 删除旧的字段规则
			if err := tx.Where("task_id = ?", id).Delete(&models.QualityTaskFieldRule{}).Error; err != nil {
				return fmt.Errorf("删除旧字段规则失败: %w", err)
			}

			// 创建新的字段规则
			for _, fieldRule := range req.FieldRules {
				// 验证规则模板是否存在
				var ruleTemplate models.QualityRuleTemplate
				if err := tx.First(&ruleTemplate, "id = ?", fieldRule.RuleTemplateID).Error; err != nil {
					return fmt.Errorf("规则模板 %s 不存在: %w", fieldRule.RuleTemplateID, err)
				}

				// 构建运行时配置和阈值的JSONB
				runtimeConfigMap := map[string]interface{}{
					"check_nullable":  fieldRule.RuntimeConfig.CheckNullable,
					"trim_whitespace": fieldRule.RuntimeConfig.TrimWhitespace,
					"case_sensitive":  fieldRule.RuntimeConfig.CaseSensitive,
					"custom_params":   fieldRule.RuntimeConfig.CustomParams,
				}
				thresholdMap := make(map[string]interface{})
				if fieldRule.Threshold.MinValue != nil {
					thresholdMap["min_value"] = *fieldRule.Threshold.MinValue
				}
				if fieldRule.Threshold.MaxValue != nil {
					thresholdMap["max_value"] = *fieldRule.Threshold.MaxValue
				}
				if fieldRule.Threshold.MinLength != nil {
					thresholdMap["min_length"] = *fieldRule.Threshold.MinLength
				}
				if fieldRule.Threshold.MaxLength != nil {
					thresholdMap["max_length"] = *fieldRule.Threshold.MaxLength
				}
				if len(fieldRule.Threshold.AllowedValues) > 0 {
					thresholdMap["allowed_values"] = fieldRule.Threshold.AllowedValues
				}
				if fieldRule.Threshold.Pattern != "" {
					thresholdMap["pattern"] = fieldRule.Threshold.Pattern
				}
				if fieldRule.Threshold.CustomThreshold != nil {
					thresholdMap["custom_threshold"] = fieldRule.Threshold.CustomThreshold
				}

				taskFieldRule := &models.QualityTaskFieldRule{
					TaskID:         id,
					FieldName:      fieldRule.FieldName,
					RuleTemplateID: fieldRule.RuleTemplateID,
					RuntimeConfig:  models.JSONB(runtimeConfigMap),
					Threshold:      models.JSONB(thresholdMap),
					IsEnabled:      fieldRule.IsEnabled,
					Priority:       fieldRule.Priority,
				}

				if err := tx.Create(taskFieldRule).Error; err != nil {
					return fmt.Errorf("创建字段规则失败: %w", err)
				}
			}
		}

		return nil
	})
}

// DeleteQualityTask 删除质量检测任务
func (s *GovernanceService) DeleteQualityTask(id string) error {
	// 检查任务是否存在
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return err
	}

	// 检查任务是否正在运行
	if task.Status == "running" {
		return errors.New("正在运行的任务不能删除")
	}

	// 使用事务删除任务和相关数据
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除问题记录
		if err := tx.Delete(&models.QualityIssueRecord{}, "task_id = ?", id).Error; err != nil {
			return fmt.Errorf("删除问题记录失败: %w", err)
		}

		// 删除执行记录
		if err := tx.Delete(&models.QualityTaskExecution{}, "task_id = ?", id).Error; err != nil {
			return fmt.Errorf("删除执行记录失败: %w", err)
		}

		// 删除字段规则
		if err := tx.Delete(&models.QualityTaskFieldRule{}, "task_id = ?", id).Error; err != nil {
			return fmt.Errorf("删除字段规则失败: %w", err)
		}

		// 删除任务
		if err := tx.Delete(&models.QualityTask{}, "id = ?", id).Error; err != nil {
			return fmt.Errorf("删除任务失败: %w", err)
		}

		return nil
	})
}

// GetQualityTaskExecutions 获取质量检测任务执行记录
func (s *GovernanceService) GetQualityTaskExecutions(taskID string, page, pageSize int) ([]QualityTaskExecutionResponse, int64, error) {
	query := s.db.Model(&models.QualityTaskExecution{}).Where("task_id = ?", taskID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var executions []models.QualityTaskExecution
	offset := (page - 1) * pageSize
	if err := query.Order("start_time DESC").Offset(offset).Limit(pageSize).Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]QualityTaskExecutionResponse, len(executions))
	for i, execution := range executions {
		responses[i] = QualityTaskExecutionResponse{
			ID:                 execution.ID,
			TaskID:             execution.TaskID,
			StartTime:          execution.StartTime,
			EndTime:            execution.EndTime,
			Duration:           execution.Duration,
			Status:             execution.Status,
			TotalRulesExecuted: execution.TotalRulesExecuted,
			PassedRules:        execution.PassedRules,
			FailedRules:        execution.FailedRules,
			OverallScore:       execution.OverallScore,
			IssueCount:         execution.IssueCount,
			ErrorMessage:       execution.ErrorMessage,
		}
	}

	return responses, total, nil
}

// executeQualityTask 执行质量检测任务（实际实现版本）
func (s *GovernanceService) executeQualityTask(execution *models.QualityTaskExecution) {
	// 获取任务详情
	var task models.QualityTask
	if err := s.db.First(&task, "id = ?", execution.TaskID).Error; err != nil {
		s.finishExecution(execution.ID, "failed", 0, 0, 0, 0, 0, fmt.Sprintf("获取任务失败: %v", err))
		return
	}

	// 获取字段规则
	var fieldRules []models.QualityTaskFieldRule
	if err := s.db.Where("task_id = ? AND is_enabled = ?", task.ID, true).
		Order("priority DESC").Find(&fieldRules).Error; err != nil {
		s.finishExecution(execution.ID, "failed", 0, 0, 0, 0, 0, fmt.Sprintf("获取字段规则失败: %v", err))
		return
	}

	if len(fieldRules) == 0 {
		s.finishExecution(execution.ID, "completed", 0, 0, 0, 1.0, 0, "没有启用的规则")
		return
	}

	// 构建查询SQL：SELECT * FROM schema.table
	tableName := fmt.Sprintf("%s.%s", task.TargetSchema, task.TargetTable)

	// 查询目标表的所有数据
	rows, err := s.db.Table(tableName).Rows()
	if err != nil {
		s.finishExecution(execution.ID, "failed", 0, 0, 0, 0, 0, fmt.Sprintf("查询目标表失败: %v", err))
		return
	}
	defer rows.Close()

	// 获取列名
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		s.finishExecution(execution.ID, "failed", 0, 0, 0, 0, 0, fmt.Sprintf("获取列信息失败: %v", err))
		return
	}

	// 创建列名到索引的映射
	columnMap := make(map[string]int)
	for i, col := range columnTypes {
		columnMap[col.Name()] = i
	}

	// 统计变量
	var totalChecks, passedChecks, failedChecks int64
	var issueCount int64

	// 遍历每一行数据
	rowNum := 0
	for rows.Next() {
		rowNum++

		// 创建值容器
		values := make([]interface{}, len(columnTypes))
		valuePtrs := make([]interface{}, len(columnTypes))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// 构建记录标识（使用第一个列作为标识，或行号）
		recordID := fmt.Sprintf("row_%d", rowNum)
		if len(values) > 0 && values[0] != nil {
			recordID = fmt.Sprintf("%v", values[0])
		}

		// 对每个字段规则进行检查
		for _, fieldRule := range fieldRules {
			totalChecks++

			// 获取字段索引
			colIndex, exists := columnMap[fieldRule.FieldName]
			if !exists {
				continue
			}

			fieldValue := values[colIndex]

			// 执行规则检查
			passed, issueDesc := s.checkFieldRule(&fieldRule, fieldValue)
			if passed {
				passedChecks++
			} else {
				failedChecks++
				issueCount++

				// 记录问题数据
				s.recordIssue(execution.ID, task.ID, &fieldRule, recordID, fieldValue, issueDesc)
			}
		}
	}

	// 计算总体得分
	var overallScore float64
	if totalChecks > 0 {
		overallScore = float64(passedChecks) / float64(totalChecks)
	} else {
		overallScore = 1.0
	}

	// 更新执行结果
	status := "completed"
	if failedChecks > 0 {
		status = "completed_with_issues"
	}

	s.finishExecution(execution.ID, status, totalChecks, passedChecks, failedChecks, overallScore, issueCount, "")
}

// checkFieldRule 检查字段规则
func (s *GovernanceService) checkFieldRule(rule *models.QualityTaskFieldRule, value interface{}) (bool, string) {
	// 获取规则模板
	var template models.QualityRuleTemplate
	if err := s.db.First(&template, "id = ?", rule.RuleTemplateID).Error; err != nil {
		return false, "规则模板不存在"
	}

	// 基于规则类型执行不同的检查逻辑
	switch template.Type {
	case "completeness":
		return s.checkCompleteness(rule, value)
	case "accuracy":
		return s.checkAccuracyRule(rule, &template, value)
	case "consistency":
		return s.checkConsistency(rule, value)
	case "validity":
		return s.checkValidityRule(rule, &template, value)
	case "uniqueness":
		return s.checkUniqueness(rule, value)
	default:
		// 兼容旧的基于名称的判断
		ruleName := strings.ToLower(template.Name)
		if strings.Contains(ruleName, "completeness") || strings.Contains(ruleName, "非空") || strings.Contains(ruleName, "完整") {
			return s.checkCompleteness(rule, value)
		} else if strings.Contains(ruleName, "accuracy") || strings.Contains(ruleName, "格式") || strings.Contains(ruleName, "准确") {
			return s.checkAccuracyRule(rule, &template, value)
		} else if strings.Contains(ruleName, "consistency") || strings.Contains(ruleName, "一致") {
			return s.checkConsistency(rule, value)
		} else if strings.Contains(ruleName, "validity") || strings.Contains(ruleName, "有效") || strings.Contains(ruleName, "范围") {
			return s.checkValidityRule(rule, &template, value)
		} else if strings.Contains(ruleName, "uniqueness") || strings.Contains(ruleName, "唯一") {
			return s.checkUniqueness(rule, value)
		}
	}
	return true, ""
}

// checkCompleteness 检查完整性（非空检查）
func (s *GovernanceService) checkCompleteness(rule *models.QualityTaskFieldRule, value interface{}) (bool, string) {
	// 检查是否需要检查null值（根据runtime_config）
	checkNullable := true
	if val, exists := rule.RuntimeConfig["check_nullable"]; exists {
		if b, ok := val.(bool); ok {
			checkNullable = b
		}
	}

	// 检查null值
	if value == nil {
		if checkNullable {
			return false, "字段值为空"
		}
		return true, "" // 不检查null，跳过
	}

	strValue := fmt.Sprintf("%v", value)

	// 检查是否需要trim空白字符
	trimWhitespace := true
	if val, exists := rule.RuntimeConfig["trim_whitespace"]; exists {
		if b, ok := val.(bool); ok {
			trimWhitespace = b
		}
	}

	if trimWhitespace {
		strValue = strings.TrimSpace(strValue)
	}

	if strValue == "" {
		if checkNullable {
			return false, "字段值为空字符串"
		}
		return true, "" // 不检查空字符串，跳过
	}

	return true, ""
}

// checkAccuracy 检查准确性（格式/模式检查）- 废弃，使用checkAccuracyRule
func (s *GovernanceService) checkAccuracy(rule *models.QualityTaskFieldRule, value interface{}) (bool, string) {
	if value == nil {
		return false, "字段值为空"
	}

	strValue := fmt.Sprintf("%v", value)
	if strings.TrimSpace(strValue) == "" {
		return false, "字段值为空字符串"
	}

	// 检查Pattern
	if pattern, ok := rule.Threshold["pattern"].(string); ok && pattern != "" {
		// 简单的模式匹配（实际应使用regexp）
		if !strings.Contains(strValue, strings.TrimSuffix(strings.TrimPrefix(pattern, "%"), "%")) {
			return false, fmt.Sprintf("不匹配模式: %s", pattern)
		}
	}

	return true, ""
}

// checkAccuracyRule 检查准确性（使用模板和正则表达式）
func (s *GovernanceService) checkAccuracyRule(rule *models.QualityTaskFieldRule, template *models.QualityRuleTemplate, value interface{}) (bool, string) {
	// 检查是否需要检查null值（根据runtime_config）
	checkNullable := true
	if val, exists := rule.RuntimeConfig["check_nullable"]; exists {
		if b, ok := val.(bool); ok {
			checkNullable = b
		}
	}

	// 检查null值
	if value == nil {
		if checkNullable {
			return false, "字段值为空"
		}
		return true, "" // 不检查null，跳过
	}

	strValue := fmt.Sprintf("%v", value)

	// 检查是否需要trim空白字符
	trimWhitespace := true
	if val, exists := rule.RuntimeConfig["trim_whitespace"]; exists {
		if b, ok := val.(bool); ok {
			trimWhitespace = b
		}
	}

	if trimWhitespace {
		strValue = strings.TrimSpace(strValue)
	}

	if strValue == "" {
		if checkNullable {
			return false, "字段值为空字符串"
		}
		return true, "" // 不检查空字符串，跳过
	}

	// 从模板的RuleLogic中获取正则表达式
	var regexPattern string
	if pattern, ok := template.RuleLogic["regex_pattern"].(string); ok {
		regexPattern = pattern
	}

	// 获取custom_params中的配置（模板特定参数）
	var customParams map[string]interface{}
	if cp, exists := rule.RuntimeConfig["custom_params"]; exists {
		if params, ok := cp.(map[string]interface{}); ok {
			customParams = params
		}
	}

	// 检查strict_mode（邮箱验证专用）
	strictMode := false
	if customParams != nil {
		if sm, ok := customParams["strict_mode"].(bool); ok {
			strictMode = sm
		}
	}

	// 如果有正则表达式，进行匹配
	if regexPattern != "" {
		// 处理转义字符（JSON中的双反斜杠）
		regexPattern = strings.ReplaceAll(regexPattern, "\\\\", "\\")

		matched, err := regexp.MatchString(regexPattern, strValue)
		if err != nil {
			return false, fmt.Sprintf("正则表达式错误: %v", err)
		}
		if !matched {
			if strictMode {
				return false, fmt.Sprintf("格式不正确（严格模式），不匹配模式: %s", regexPattern)
			}
			return false, fmt.Sprintf("格式不正确，不匹配模式: %s", regexPattern)
		}
	}

	// 检查Threshold中的pattern（简单模式匹配）
	if pattern, ok := rule.Threshold["pattern"].(string); ok && pattern != "" {
		if !strings.Contains(strValue, strings.TrimSuffix(strings.TrimPrefix(pattern, "%"), "%")) {
			return false, fmt.Sprintf("不包含必需的字符: %s", pattern)
		}
	}

	return true, ""
}

// checkConsistency 检查一致性（值域检查）
func (s *GovernanceService) checkConsistency(rule *models.QualityTaskFieldRule, value interface{}) (bool, string) {
	if value == nil {
		return true, ""
	}

	// 检查AllowedValues
	if allowedValues, ok := rule.Threshold["allowed_values"].([]interface{}); ok && len(allowedValues) > 0 {
		strValue := fmt.Sprintf("%v", value)
		found := false
		for _, av := range allowedValues {
			if fmt.Sprintf("%v", av) == strValue {
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Sprintf("值不在允许的范围内: %v", allowedValues)
		}
	}

	return true, ""
}

// checkValidity 检查有效性（范围检查）- 废弃，使用checkValidityRule
func (s *GovernanceService) checkValidity(rule *models.QualityTaskFieldRule, value interface{}) (bool, string) {
	if value == nil {
		return false, "字段值为空"
	}

	strValue := fmt.Sprintf("%v", value)
	if strings.TrimSpace(strValue) == "" {
		return false, "字段值为空字符串"
	}

	// 数值范围检查
	if minValue, ok := rule.Threshold["min_value"].(float64); ok {
		if numValue, ok := value.(float64); ok {
			if numValue < minValue {
				return false, fmt.Sprintf("值 %v 小于最小值 %v", numValue, minValue)
			}
		}
	}

	if maxValue, ok := rule.Threshold["max_value"].(float64); ok {
		if numValue, ok := value.(float64); ok {
			if numValue > maxValue {
				return false, fmt.Sprintf("值 %v 大于最大值 %v", numValue, maxValue)
			}
		}
	}

	// 长度检查
	if minLength, ok := rule.Threshold["min_length"].(float64); ok {
		if len(strValue) < int(minLength) {
			return false, fmt.Sprintf("长度 %d 小于最小长度 %d", len(strValue), int(minLength))
		}
	}

	if maxLength, ok := rule.Threshold["max_length"].(float64); ok {
		if len(strValue) > int(maxLength) {
			return false, fmt.Sprintf("长度 %d 大于最大长度 %d", len(strValue), int(maxLength))
		}
	}

	return true, ""
}

// checkValidityRule 检查有效性（使用模板和正则表达式）
func (s *GovernanceService) checkValidityRule(rule *models.QualityTaskFieldRule, template *models.QualityRuleTemplate, value interface{}) (bool, string) {
	// 检查是否需要检查null值（根据runtime_config）
	checkNullable := true
	if val, exists := rule.RuntimeConfig["check_nullable"]; exists {
		if b, ok := val.(bool); ok {
			checkNullable = b
		}
	}

	// 检查null值
	if value == nil {
		if checkNullable {
			return false, "字段值为空"
		}
		return true, "" // 不检查null，跳过
	}

	strValue := fmt.Sprintf("%v", value)

	// 检查是否需要trim空白字符
	trimWhitespace := true
	if val, exists := rule.RuntimeConfig["trim_whitespace"]; exists {
		if b, ok := val.(bool); ok {
			trimWhitespace = b
		}
	}

	if trimWhitespace {
		strValue = strings.TrimSpace(strValue)
	}

	if strValue == "" {
		if checkNullable {
			return false, "字段值为空字符串"
		}
		return true, "" // 不检查空字符串，跳过
	}

	// 从模板的RuleLogic中获取正则表达式和验证类型
	var regexPattern string
	var validationType string
	if pattern, ok := template.RuleLogic["regex_pattern"].(string); ok {
		regexPattern = pattern
	}
	if vtype, ok := template.RuleLogic["validation_type"].(string); ok {
		validationType = vtype
	}

	// 获取custom_params中的配置（模板特定参数）
	var customParams map[string]interface{}
	if cp, exists := rule.RuntimeConfig["custom_params"]; exists {
		if params, ok := cp.(map[string]interface{}); ok {
			customParams = params
		}
	}

	// 处理手机号验证
	if validationType == "phone" {
		// 检查是否允许国际号码（从custom_params中获取）
		allowInternational := false
		if customParams != nil {
			if allow, ok := customParams["allow_international"].(bool); ok {
				allowInternational = allow
			}
		}

		// 处理转义字符
		regexPattern = strings.ReplaceAll(regexPattern, "\\\\", "\\")

		// 如果允许国际号码，添加额外的匹配规则
		if allowInternational {
			// 中国手机号 或 国际号码（+开头，后跟国家代码和号码）
			chinaPattern := regexPattern
			intlPattern := `^\+\d{1,3}\d{7,14}$`

			// 先尝试中国号码
			if matched, _ := regexp.MatchString(chinaPattern, strValue); matched {
				return true, ""
			}
			// 再尝试国际号码
			if matched, _ := regexp.MatchString(intlPattern, strValue); matched {
				return true, ""
			}
			return false, "手机号格式不正确（支持中国号码或国际号码）"
		} else {
			// 只验证中国手机号
			matched, err := regexp.MatchString(regexPattern, strValue)
			if err != nil {
				return false, fmt.Sprintf("正则表达式错误: %v", err)
			}
			if !matched {
				return false, "手机号格式不正确（仅支持中国大陆手机号）"
			}
		}
		return true, ""
	}

	// 如果有正则表达式，进行匹配
	if regexPattern != "" {
		// 处理转义字符（JSON中的双反斜杠）
		regexPattern = strings.ReplaceAll(regexPattern, "\\\\", "\\")

		matched, err := regexp.MatchString(regexPattern, strValue)
		if err != nil {
			return false, fmt.Sprintf("正则表达式错误: %v", err)
		}
		if !matched {
			return false, fmt.Sprintf("格式不正确，不匹配模式: %s", regexPattern)
		}
	}

	return true, ""
}

// checkUniqueness 检查唯一性（需要查询数据库）
func (s *GovernanceService) checkUniqueness(rule *models.QualityTaskFieldRule, value interface{}) (bool, string) {
	// 唯一性检查需要查询数据库，这里简化实现
	// 实际应该查询目标表，检查该字段值是否重复
	return true, ""
}

// recordIssue 记录质量问题
func (s *GovernanceService) recordIssue(executionID, taskID string, rule *models.QualityTaskFieldRule, recordID string, fieldValue interface{}, issueDesc string) {
	issue := &models.QualityIssueRecord{
		ExecutionID:      executionID,
		TaskID:           taskID,
		FieldName:        rule.FieldName,
		RuleTemplateID:   rule.RuleTemplateID,
		RecordIdentifier: recordID,
		IssueType:        "validation_failed",
		IssueDescription: issueDesc,
		FieldValue:       fmt.Sprintf("%v", fieldValue),
		Severity:         s.determineSeverity(rule),
	}

	if err := s.db.Create(issue).Error; err != nil {
		// 记录错误但不中断执行
		fmt.Printf("记录问题失败: %v\n", err)
	}
}

// determineSeverity 确定问题严重程度
func (s *GovernanceService) determineSeverity(rule *models.QualityTaskFieldRule) string {
	// 根据规则优先级确定严重程度
	if rule.Priority >= 80 {
		return "critical"
	} else if rule.Priority >= 60 {
		return "high"
	} else if rule.Priority >= 40 {
		return "medium"
	}
	return "low"
}

// finishExecution 完成执行并更新状态
func (s *GovernanceService) finishExecution(executionID, status string, totalRules, passedRules, failedRules int64, overallScore float64, issueCount int64, errorMessage string) {
	endTime := time.Now()

	var execution models.QualityTaskExecution
	s.db.First(&execution, "id = ?", executionID)

	duration := endTime.Sub(execution.StartTime).Milliseconds()

	updates := map[string]interface{}{
		"end_time":             &endTime,
		"duration":             duration,
		"status":               status,
		"total_rules_executed": totalRules,
		"passed_rules":         passedRules,
		"failed_rules":         failedRules,
		"overall_score":        overallScore,
		"issue_count":          issueCount,
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	s.db.Model(&models.QualityTaskExecution{}).Where("id = ?", executionID).Updates(updates)

	// 更新任务状态
	taskUpdates := map[string]interface{}{
		"status":          status,
		"last_executed":   &endTime,
		"execution_count": gorm.Expr("execution_count + 1"),
	}

	if status == "completed" || status == "completed_with_issues" {
		taskUpdates["success_count"] = gorm.Expr("success_count + 1")
	} else {
		taskUpdates["failure_count"] = gorm.Expr("failure_count + 1")
	}

	s.db.Model(&models.QualityTask{}).Where("id = ?", execution.TaskID).Updates(taskUpdates)
}

// === 调度和执行相关方法 ===

// CalculateNextExecution 计算下次执行时间
func (s *GovernanceService) CalculateNextExecution(config ScheduleConfigRequest, lastExecution *time.Time) (*time.Time, error) {
	now := time.Now()
	if lastExecution != nil {
		now = *lastExecution
	}

	switch config.Type {
	case "manual":
		// 手动触发，不设置下次执行时间
		return nil, nil

	case "once":
		// 单次执行
		if config.StartTime != nil && config.StartTime.After(time.Now()) {
			return config.StartTime, nil
		}
		return nil, nil

	case "interval":
		// 间隔执行
		if config.Interval <= 0 {
			return nil, errors.New("间隔时间必须大于0")
		}
		nextTime := now.Add(time.Duration(config.Interval) * time.Second)
		return &nextTime, nil

	case "cron":
		// Cron表达式
		if config.CronExpr == "" {
			return nil, errors.New("Cron表达式不能为空")
		}
		parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedule, err := parser.Parse(config.CronExpr)
		if err != nil {
			return nil, fmt.Errorf("解析Cron表达式失败: %w", err)
		}
		nextTime := schedule.Next(now)
		return &nextTime, nil

	default:
		return nil, fmt.Errorf("不支持的调度类型: %s", config.Type)
	}
}

// GetQualityIssueRecords 获取质量问题记录
func (s *GovernanceService) GetQualityIssueRecords(taskID, executionID string, page, pageSize int, fieldName, severity string) ([]QualityIssueRecordResponse, int64, error) {
	query := s.db.Model(&models.QualityIssueRecord{})

	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	if executionID != "" {
		query = query.Where("execution_id = ?", executionID)
	}
	if fieldName != "" {
		query = query.Where("field_name = ?", fieldName)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []models.QualityIssueRecord
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]QualityIssueRecordResponse, len(records))
	for i, record := range records {
		// 获取规则模板名称
		var template models.QualityRuleTemplate
		ruleName := "未知规则"
		if err := s.db.First(&template, "id = ?", record.RuleTemplateID).Error; err == nil {
			ruleName = template.Name
		}

		responses[i] = QualityIssueRecordResponse{
			ID:               record.ID,
			ExecutionID:      record.ExecutionID,
			TaskID:           record.TaskID,
			FieldName:        record.FieldName,
			RuleTemplateID:   record.RuleTemplateID,
			RuleTemplateName: ruleName,
			RecordIdentifier: record.RecordIdentifier,
			IssueType:        record.IssueType,
			IssueDescription: record.IssueDescription,
			FieldValue:       record.FieldValue,
			ExpectedValue:    record.ExpectedValue,
			Severity:         record.Severity,
			CreatedAt:        record.CreatedAt,
		}
	}

	return responses, total, nil
}
