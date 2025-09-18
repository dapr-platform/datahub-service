/*
 * @module service/thematic_sync/sync_engine
 * @description 主题同步引擎，协调整个数据同步流程
 * @architecture 管道模式 - 通过多个处理阶段完成数据同步
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 任务接收 -> 数据获取 -> 汇聚处理 -> 清洗脱敏 -> 质量检查 -> 数据写入 -> 血缘记录
 * @rules 确保同步流程的完整性和一致性，支持事务性操作和错误恢复
 * @dependencies gorm.io/gorm, context, time
 * @refs models/thematic_sync.go, aggregation_engine.go, cleansing_engine.go
 */

package thematic_sync

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"datahub-service/service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncExecutionPhase 同步执行阶段
type SyncExecutionPhase string

const (
	PhaseInitialize  SyncExecutionPhase = "initialize"  // 初始化
	PhaseDataFetch   SyncExecutionPhase = "data_fetch"  // 数据获取
	PhaseAggregation SyncExecutionPhase = "aggregation" // 数据汇聚
	PhaseGovernance  SyncExecutionPhase = "governance"  // 数据治理处理
	PhaseDataWrite   SyncExecutionPhase = "data_write"  // 数据写入
	PhaseLineage     SyncExecutionPhase = "lineage"     // 血缘记录
	PhaseComplete    SyncExecutionPhase = "complete"    // 完成
	// 保留旧阶段常量用于向后兼容
	PhaseCleansing    SyncExecutionPhase = "cleansing"     // 数据清洗 (已废弃，使用 PhaseGovernance)
	PhasePrivacy      SyncExecutionPhase = "privacy"       // 隐私处理 (已废弃，使用 PhaseGovernance)
	PhaseQualityCheck SyncExecutionPhase = "quality_check" // 质量检查 (已废弃，使用 PhaseGovernance)
)

// SyncRequest 同步请求
type SyncRequest struct {
	TaskID            string                 `json:"task_id"`
	ExecutionType     string                 `json:"execution_type"` // manual, scheduled, retry
	SourceLibraries   []string               `json:"source_libraries"`
	SourceInterfaces  []string               `json:"source_interfaces"`
	TargetLibraryID   string                 `json:"target_library_id"`
	TargetInterfaceID string                 `json:"target_interface_id"`
	Config            map[string]interface{} `json:"config"`
	Context           context.Context        `json:"-"`
}

// SyncProgress 同步进度
type SyncProgress struct {
	ExecutionID    string             `json:"execution_id"`
	CurrentPhase   SyncExecutionPhase `json:"current_phase"`
	Progress       float64            `json:"progress"` // 0-100
	ProcessedCount int64              `json:"processed_count"`
	TotalCount     int64              `json:"total_count"`
	ErrorCount     int64              `json:"error_count"`
	Message        string             `json:"message"`
	StartTime      time.Time          `json:"start_time"`
	LastUpdateTime time.Time          `json:"last_update_time"`
}

// SyncResponse 同步响应
type SyncResponse struct {
	ExecutionID    string               `json:"execution_id"`
	Status         string               `json:"status"`
	Result         *SyncExecutionResult `json:"result,omitempty"`
	Error          string               `json:"error,omitempty"`
	Progress       *SyncProgress        `json:"progress,omitempty"`
	ProcessingTime time.Duration        `json:"processing_time"`
}

// SyncExecutionResult 同步执行结果
type SyncExecutionResult struct {
	SourceRecordCount    int64                `json:"source_record_count"`
	ProcessedRecordCount int64                `json:"processed_record_count"`
	InsertedRecordCount  int64                `json:"inserted_record_count"`
	UpdatedRecordCount   int64                `json:"updated_record_count"`
	ErrorRecordCount     int64                `json:"error_record_count"`
	QualityScore         float64              `json:"quality_score"`
	ProcessingSteps      []ProcessingStepInfo `json:"processing_steps"`
	ValidationErrors     []ValidationError    `json:"validation_errors"`
	LineageRecords       []LineageRecord      `json:"lineage_records"`
}

// ProcessingStepInfo 处理步骤信息
type ProcessingStepInfo struct {
	Phase       SyncExecutionPhase `json:"phase"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	Duration    time.Duration      `json:"duration"`
	RecordCount int64              `json:"record_count"`
	ErrorCount  int64              `json:"error_count"`
	Status      string             `json:"status"`
	Message     string             `json:"message"`
}

// GovernanceIntegrationServiceInterface 数据治理集成服务接口
type GovernanceIntegrationServiceInterface interface {
	ApplyGovernanceRules(ctx context.Context, records []map[string]interface{}, task *models.ThematicSyncTask, config *GovernanceExecutionConfig) (*GovernanceExecutionResult, error)
}

// GovernanceExecutionConfig 数据治理执行配置
type GovernanceExecutionConfig struct {
	EnableQualityCheck      bool                   `json:"enable_quality_check"`
	EnableCleansing         bool                   `json:"enable_cleansing"`
	EnableMasking           bool                   `json:"enable_masking"`
	EnableTransformation    bool                   `json:"enable_transformation"`
	EnableValidation        bool                   `json:"enable_validation"`
	StopOnQualityFailure    bool                   `json:"stop_on_quality_failure"`
	StopOnValidationFailure bool                   `json:"stop_on_validation_failure"`
	QualityThreshold        float64                `json:"quality_threshold"`
	BatchSize               int                    `json:"batch_size"`
	MaxRetries              int                    `json:"max_retries"`
	TimeoutSeconds          int                    `json:"timeout_seconds"`
	CustomConfig            map[string]interface{} `json:"custom_config,omitempty"`
}

// GovernanceExecutionResult 数据治理执行结果
type GovernanceExecutionResult struct {
	OverallQualityScore   float64       `json:"overall_quality_score"`
	TotalProcessedRecords int64         `json:"total_processed_records"`
	TotalCleansingApplied int64         `json:"total_cleansing_applied"`
	TotalMaskingApplied   int64         `json:"total_masking_applied"`
	TotalValidationErrors int64         `json:"total_validation_errors"`
	ExecutionTime         time.Duration `json:"execution_time"`
	ComplianceStatus      string        `json:"compliance_status"`
}

// ThematicSyncEngine 主题同步引擎
type ThematicSyncEngine struct {
	db                     *gorm.DB
	aggregationEngine      *AggregationEngine
	cleansingEngine        *CleansingEngine
	privacyEngine          *PrivacyEngine
	basicLibraryService    BasicLibraryServiceInterface
	thematicLibraryService ThematicLibraryServiceInterface
	governanceIntegration  GovernanceIntegrationServiceInterface
	progressCallback       func(*SyncProgress)
}

// BasicLibraryServiceInterface 基础库服务接口
type BasicLibraryServiceInterface interface {
	GetDataByInterface(ctx context.Context, libraryID, interfaceID string) ([]map[string]interface{}, error)
	GetInterfaceInfo(ctx context.Context, interfaceID string) (*models.DataInterface, error)
}

// ThematicLibraryServiceInterface 主题库服务接口
type ThematicLibraryServiceInterface interface {
	WriteData(ctx context.Context, interfaceID string, records []map[string]interface{}) error
	GetInterfaceInfo(ctx context.Context, interfaceID string) (*models.ThematicInterface, error)
}

// NewThematicSyncEngine 创建主题同步引擎
func NewThematicSyncEngine(db *gorm.DB,
	basicLibraryService BasicLibraryServiceInterface,
	thematicLibraryService ThematicLibraryServiceInterface,
	governanceIntegration GovernanceIntegrationServiceInterface) *ThematicSyncEngine {

	return &ThematicSyncEngine{
		db:                     db,
		aggregationEngine:      NewAggregationEngine(db),
		cleansingEngine:        NewCleansingEngine(),
		privacyEngine:          NewPrivacyEngine(),
		basicLibraryService:    basicLibraryService,
		thematicLibraryService: thematicLibraryService,
		governanceIntegration:  governanceIntegration,
	}
}

// SetProgressCallback 设置进度回调
func (tse *ThematicSyncEngine) SetProgressCallback(callback func(*SyncProgress)) {
	tse.progressCallback = callback
}

// ExecuteSync 执行同步
func (tse *ThematicSyncEngine) ExecuteSync(request *SyncRequest) (*SyncResponse, error) {
	startTime := time.Now()
	executionID := uuid.New().String()

	// 创建执行记录
	execution := &models.ThematicSyncExecution{
		ID:            executionID,
		TaskID:        request.TaskID,
		ExecutionType: request.ExecutionType,
		Status:        "running",
		StartTime:     &startTime,
		CreatedBy:     "system",
	}

	if err := tse.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// 初始化进度
	progress := &SyncProgress{
		ExecutionID:    executionID,
		CurrentPhase:   PhaseInitialize,
		Progress:       0,
		StartTime:      startTime,
		LastUpdateTime: startTime,
	}

	// 执行同步流程
	result, err := tse.executeSyncPipeline(request, progress)

	// 更新执行记录
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = int64(endTime.Sub(startTime).Seconds())

	if err != nil {
		execution.Status = "failed"
		execution.ErrorDetails = models.JSONB{
			"error": err.Error(),
		}
	} else {
		execution.Status = "success"
		execution.SourceRecordCount = result.SourceRecordCount
		execution.ProcessedRecordCount = result.ProcessedRecordCount
		execution.InsertedRecordCount = result.InsertedRecordCount
		execution.UpdatedRecordCount = result.UpdatedRecordCount
		execution.ErrorRecordCount = result.ErrorRecordCount
	}

	tse.db.Save(execution)

	// 构建响应
	response := &SyncResponse{
		ExecutionID:    executionID,
		ProcessingTime: time.Since(startTime),
		Progress:       progress,
	}

	if err != nil {
		response.Status = "failed"
		response.Error = err.Error()
	} else {
		response.Status = "success"
		response.Result = result
	}

	return response, err
}

// executeSyncPipeline 执行同步管道
func (tse *ThematicSyncEngine) executeSyncPipeline(request *SyncRequest,
	progress *SyncProgress) (*SyncExecutionResult, error) {

	result := &SyncExecutionResult{
		ProcessingSteps: make([]ProcessingStepInfo, 0),
	}

	// 1. 初始化阶段
	if err := tse.executePhase(PhaseInitialize, progress, func() error {
		return tse.initializeSync(request, result)
	}); err != nil {
		return nil, err
	}

	// 2. 数据获取阶段
	var sourceRecords []SourceRecordInfo
	if err := tse.executePhase(PhaseDataFetch, progress, func() error {
		var err error
		sourceRecords, err = tse.fetchSourceData(request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 3. 数据汇聚阶段
	var aggregationResult *AggregationResult
	if err := tse.executePhase(PhaseAggregation, progress, func() error {
		var err error
		aggregationResult, err = tse.performAggregation(sourceRecords, request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 4. 数据治理处理阶段 - 统一处理数据质量、清洗、脱敏、转换、校验
	var processedRecords []map[string]interface{}
	var governanceResult *GovernanceExecutionResult
	if err := tse.executePhase(PhaseGovernance, progress, func() error {
		var err error
		processedRecords, governanceResult, err = tse.performGovernanceProcessing(aggregationResult.AggregatedRecords, request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 5. 数据写入阶段
	if err := tse.executePhase(PhaseDataWrite, progress, func() error {
		return tse.writeData(processedRecords, request, result, governanceResult)
	}); err != nil {
		return nil, err
	}

	// 6. 血缘记录阶段
	if err := tse.executePhase(PhaseLineage, progress, func() error {
		return tse.recordLineage(aggregationResult, request, result)
	}); err != nil {
		return nil, err
	}

	// 7. 完成阶段
	if err := tse.executePhase(PhaseComplete, progress, func() error {
		return tse.completeSync(request, result, governanceResult)
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// executePhase 执行阶段
func (tse *ThematicSyncEngine) executePhase(phase SyncExecutionPhase, progress *SyncProgress,
	fn func() error) error {

	stepInfo := ProcessingStepInfo{
		Phase:     phase,
		StartTime: time.Now(),
		Status:    "running",
	}

	// 更新进度
	progress.CurrentPhase = phase
	progress.LastUpdateTime = time.Now()
	if tse.progressCallback != nil {
		tse.progressCallback(progress)
	}

	// 执行阶段逻辑
	err := fn()

	// 更新步骤信息
	stepInfo.EndTime = time.Now()
	stepInfo.Duration = stepInfo.EndTime.Sub(stepInfo.StartTime)

	if err != nil {
		stepInfo.Status = "failed"
		stepInfo.Message = err.Error()
	} else {
		stepInfo.Status = "success"
	}

	return err
}

// initializeSync 初始化同步
func (tse *ThematicSyncEngine) initializeSync(request *SyncRequest,
	result *SyncExecutionResult) error {

	// 验证请求参数
	if len(request.SourceInterfaces) == 0 {
		return fmt.Errorf("源接口列表不能为空")
	}

	if request.TargetInterfaceID == "" {
		return fmt.Errorf("目标接口ID不能为空")
	}

	// 验证目标接口是否存在
	_, err := tse.thematicLibraryService.GetInterfaceInfo(request.Context, request.TargetInterfaceID)
	if err != nil {
		return fmt.Errorf("获取目标接口信息失败: %w", err)
	}

	return nil
}

// fetchSourceData 获取源数据
func (tse *ThematicSyncEngine) fetchSourceData(request *SyncRequest,
	result *SyncExecutionResult) ([]SourceRecordInfo, error) {

	// 优先检查是否配置了SQL数据源
	if sqlConfigs, hasSQLConfig := tse.parseSQLDataSourceConfigs(request); hasSQLConfig {
		// 使用SQL数据源获取数据
		return tse.fetchDataFromSQL(sqlConfigs, request, result)
	}

	// 兜底：使用传统的基础库接口获取数据
	return tse.fetchDataFromBasicLibrary(request, result)
}

// performAggregation 执行数据汇聚
func (tse *ThematicSyncEngine) performAggregation(sourceRecords []SourceRecordInfo,
	request *SyncRequest, result *SyncExecutionResult) (*AggregationResult, error) {

	// 构建汇聚配置
	config := AggregationConfig{
		Strategy: MergeStrategy, // 默认使用合并策略
		KeyMatchingRules: []KeyMatchingRule{
			{
				Strategy:       ExactMatch,
				MatchFields:    []string{"id", "name", "email"}, // 默认匹配字段
				WeightConfig:   map[string]float64{"id": 1.0, "name": 0.8, "email": 0.9},
				ThresholdScore: 0.8,
				ConflictPolicy: "keep_latest",
			},
		},
		ConflictPolicy: KeepLatest,
		DeduplicationConfig: DeduplicationConfig{
			Enabled:   true,
			KeyFields: []string{"id"},
			Strategy:  "best_quality",
		},
	}

	// 从请求配置中覆盖默认配置
	if aggConfig, exists := request.Config["aggregation"]; exists {
		// 这里可以解析配置并覆盖默认值
		_ = aggConfig
	}

	return tse.aggregationEngine.AggregateData(sourceRecords, config)
}

// performGovernanceProcessing 执行数据治理处理
func (tse *ThematicSyncEngine) performGovernanceProcessing(records []map[string]interface{},
	request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, *GovernanceExecutionResult, error) {

	// 获取任务信息以获取数据治理配置
	task, err := tse.getTaskInfo(request.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取任务信息失败: %w", err)
	}

	// 从请求配置中获取数据治理配置
	var governanceConfig *GovernanceExecutionConfig
	if configInterface, exists := request.Config["governance_config"]; exists {
		if config, ok := configInterface.(GovernanceExecutionConfig); ok {
			governanceConfig = &config
		} else {
			// 如果类型不匹配，使用默认配置
			governanceConfig = &GovernanceExecutionConfig{
				EnableQualityCheck:      true,
				EnableCleansing:         true,
				EnableMasking:           true,
				EnableTransformation:    true,
				EnableValidation:        true,
				StopOnQualityFailure:    false,
				StopOnValidationFailure: false,
				QualityThreshold:        0.8,
				BatchSize:               1000,
				MaxRetries:              3,
				TimeoutSeconds:          300,
			}
		}
	} else {
		// 使用默认配置
		governanceConfig = &GovernanceExecutionConfig{
			EnableQualityCheck:      true,
			EnableCleansing:         true,
			EnableMasking:           true,
			EnableTransformation:    true,
			EnableValidation:        true,
			StopOnQualityFailure:    false,
			StopOnValidationFailure: false,
			QualityThreshold:        0.8,
			BatchSize:               1000,
			MaxRetries:              3,
			TimeoutSeconds:          300,
		}
	}

	// 使用数据治理集成服务处理数据
	governanceResult, err := tse.governanceIntegration.ApplyGovernanceRules(
		request.Context,
		records,
		task,
		governanceConfig,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("数据治理处理失败: %w", err)
	}

	// 更新结果统计
	result.QualityScore = governanceResult.OverallQualityScore

	// 添加处理步骤信息
	stepInfo := ProcessingStepInfo{
		Phase:       PhaseGovernance,
		StartTime:   time.Now().Add(-governanceResult.ExecutionTime),
		EndTime:     time.Now(),
		Duration:    governanceResult.ExecutionTime,
		RecordCount: governanceResult.TotalProcessedRecords,
		ErrorCount:  governanceResult.TotalValidationErrors,
		Status:      governanceResult.ComplianceStatus,
		Message:     fmt.Sprintf("数据治理处理完成，质量评分: %.2f", governanceResult.OverallQualityScore),
	}
	result.ProcessingSteps = append(result.ProcessingSteps, stepInfo)

	return records, governanceResult, nil
}

// getTaskInfo 获取任务信息
func (tse *ThematicSyncEngine) getTaskInfo(taskID string) (*models.ThematicSyncTask, error) {
	var task models.ThematicSyncTask
	if err := tse.db.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("获取任务信息失败: %w", err)
	}
	return &task, nil
}

// performCleansing 执行数据清洗 (已废弃，保留用于向后兼容)
func (tse *ThematicSyncEngine) performCleansing(records []map[string]interface{},
	request *SyncRequest, result *SyncExecutionResult) ([]CleansingResult, error) {

	// 构建清洗规则
	rules := []CleansingRule{
		{
			ID:           "trim_strings",
			Name:         "字符串去空格",
			Type:         DataNormalization,
			TargetFields: []string{"name", "address", "email"},
			Actions: []RuleAction{
				{
					Type:      "transform",
					Transform: "trim",
				},
			},
			Priority:  1,
			IsEnabled: true,
		},
		{
			ID:           "validate_email",
			Name:         "邮箱格式验证",
			Type:         DataValidation,
			TargetFields: []string{"email"},
			Actions: []RuleAction{
				{
					Type: "validate",
					Config: map[string]interface{}{
						"pattern": `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
					},
				},
			},
			Priority:  2,
			IsEnabled: true,
		},
	}

	cleansingResults, err := tse.cleansingEngine.CleanseRecords(records, rules)
	if err != nil {
		return nil, err
	}

	// 更新结果统计
	errorCount := int64(0)
	for _, cleansingResult := range cleansingResults {
		errorCount += int64(len(cleansingResult.ValidationErrors))
	}
	result.ErrorRecordCount += errorCount

	return cleansingResults, nil
}

// performPrivacyProcessing 执行隐私处理
func (tse *ThematicSyncEngine) performPrivacyProcessing(cleansingResults []CleansingResult,
	request *SyncRequest, result *SyncExecutionResult) ([]MaskingResult, error) {

	// 构建隐私规则
	rules := []PrivacyRule{
		{
			ID:               "mask_phone",
			FieldPattern:     "phone|mobile",
			DataType:         "phone",
			SensitivityLevel: Restricted,
			MaskingStrategy:  PartialMasking,
			MaskingConfig: map[string]interface{}{
				"prefix_length": 3,
				"suffix_length": 4,
				"mask_char":     "*",
			},
			IsEnabled: true,
		},
		{
			ID:               "mask_id_card",
			FieldPattern:     "id_card|identity_card",
			DataType:         "id_card",
			SensitivityLevel: Confidential,
			MaskingStrategy:  PartialMasking,
			MaskingConfig: map[string]interface{}{
				"prefix_length": 6,
				"suffix_length": 4,
				"mask_char":     "*",
			},
			IsEnabled: true,
		},
	}

	// 提取清洗后的记录
	var cleanedRecords []map[string]interface{}
	for _, cleansingResult := range cleansingResults {
		cleanedRecords = append(cleanedRecords, cleansingResult.CleanedRecord)
	}

	return tse.privacyEngine.MaskRecords(cleanedRecords, rules)
}

// performQualityCheck 执行质量检查
func (tse *ThematicSyncEngine) performQualityCheck(maskingResults []MaskingResult,
	request *SyncRequest, result *SyncExecutionResult) error {

	totalScore := 0.0
	recordCount := len(maskingResults)

	for _, maskingResult := range maskingResults {
		// 计算记录质量评分
		score := tse.calculateRecordQuality(maskingResult.MaskedRecord)
		totalScore += score
	}

	if recordCount > 0 {
		result.QualityScore = totalScore / float64(recordCount)
	}

	return nil
}

// writeData 写入数据
func (tse *ThematicSyncEngine) writeData(processedRecords []map[string]interface{},
	request *SyncRequest, result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error {

	// 使用经过数据治理处理的记录
	recordsToWrite := processedRecords

	// 写入目标接口
	err := tse.thematicLibraryService.WriteData(request.Context, request.TargetInterfaceID, recordsToWrite)
	if err != nil {
		return fmt.Errorf("写入数据失败: %w", err)
	}

	result.ProcessedRecordCount = int64(len(recordsToWrite))
	result.InsertedRecordCount = int64(len(recordsToWrite)) // 简化处理，假设都是新增

	return nil
}

// recordLineage 记录血缘
func (tse *ThematicSyncEngine) recordLineage(aggregationResult *AggregationResult,
	request *SyncRequest, result *SyncExecutionResult) error {

	// 创建血缘记录
	for _, lineageRecord := range aggregationResult.LineageRecords {
		lineage := &models.ThematicDataLineage{
			ID:                    uuid.New().String(),
			ThematicInterfaceID:   request.TargetInterfaceID,
			ThematicRecordID:      lineageRecord.TargetRecordID,
			ProcessingRules:       models.JSONB{},
			TransformationDetails: models.JSONB{},
			QualityScore:          result.QualityScore,
			QualityIssues:         models.JSONB{},
			SourceDataTime:        time.Now(),
			ProcessedTime:         time.Now(),
			CreatedAt:             time.Now(),
		}

		// 设置源数据信息（简化处理，使用第一个源）
		if len(lineageRecord.SourceRecords) > 0 {
			source := lineageRecord.SourceRecords[0]
			lineage.SourceLibraryID = source.LibraryID
			lineage.SourceInterfaceID = source.InterfaceID
			lineage.SourceRecordID = source.RecordID
		}

		if err := tse.db.Create(lineage).Error; err != nil {
			return fmt.Errorf("创建血缘记录失败: %w", err)
		}
	}

	result.LineageRecords = aggregationResult.LineageRecords
	return nil
}

// completeSync 完成同步
func (tse *ThematicSyncEngine) completeSync(request *SyncRequest,
	result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error {

	// 更新任务统计信息
	var task models.ThematicSyncTask
	if err := tse.db.First(&task, "id = ?", request.TaskID).Error; err != nil {
		return fmt.Errorf("获取任务信息失败: %w", err)
	}

	now := time.Now()
	task.LastSyncTime = &now
	task.LastSyncStatus = "success"
	task.TotalSyncCount++
	task.SuccessfulSyncCount++
	task.UpdateNextRunTime()

	if err := tse.db.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务信息失败: %w", err)
	}

	// 更新执行记录中的数据治理结果
	if governanceResult != nil {
		var execution models.ThematicSyncExecution
		// 这里应该从 request 或其他地方获取 executionID，为了简化暂时跳过
		// 在实际实现中，应该在 ExecuteSync 方法中传递 executionID

		// 如果有执行记录，更新数据治理相关字段
		var governanceResultMap map[string]interface{}
		governanceResultBytes, _ := json.Marshal(governanceResult)
		json.Unmarshal(governanceResultBytes, &governanceResultMap)
		execution.GovernanceResult = models.JSONB(governanceResultMap)
		execution.QualityScore = governanceResult.OverallQualityScore
		execution.CleansingCount = governanceResult.TotalCleansingApplied
		execution.MaskingCount = governanceResult.TotalMaskingApplied
		execution.ValidationErrors = governanceResult.TotalValidationErrors

		// 注意：在实际实现中，应该根据 executionID 更新特定的执行记录
		// tse.db.Model(&execution).Where("id = ?", executionID).Updates(&execution)
	}

	return nil
}

// calculateRecordQuality 计算记录质量
func (tse *ThematicSyncEngine) calculateRecordQuality(record map[string]interface{}) float64 {
	if len(record) == 0 {
		return 0.0
	}

	nonNullCount := 0
	for _, value := range record {
		if value != nil && fmt.Sprintf("%v", value) != "" {
			nonNullCount++
		}
	}

	return float64(nonNullCount) / float64(len(record)) * 100
}

// parseSQLDataSourceConfigs 解析SQL数据源配置
func (tse *ThematicSyncEngine) parseSQLDataSourceConfigs(request *SyncRequest) ([]SQLDataSourceConfig, bool) {
	// 检查请求配置中是否有SQL数据源配置
	if sqlConfigRaw, exists := request.Config["data_source_sql"]; exists {
		var sqlConfigs []SQLDataSourceConfig

		// 尝试直接转换
		if configSlice, ok := sqlConfigRaw.([]SQLDataSourceConfig); ok {
			return configSlice, true
		}

		// 尝试从接口数组转换
		if configSlice, ok := sqlConfigRaw.([]interface{}); ok {
			for _, configRaw := range configSlice {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := SQLDataSourceConfig{
						LibraryID:   tse.getStringFromMap(configMap, "library_id"),
						InterfaceID: tse.getStringFromMap(configMap, "interface_id"),
						SQLQuery:    tse.getStringFromMap(configMap, "sql_query"),
						Timeout:     30,    // 默认30秒
						MaxRows:     10000, // 默认1万行
					}

					// 解析参数
					if params, exists := configMap["parameters"]; exists {
						if paramsMap, ok := params.(map[string]interface{}); ok {
							config.Parameters = paramsMap
						}
					}

					// 解析超时时间
					if timeout, exists := configMap["timeout"]; exists {
						if timeoutInt, ok := timeout.(int); ok {
							config.Timeout = timeoutInt
						}
					}

					// 解析最大行数
					if maxRows, exists := configMap["max_rows"]; exists {
						if maxRowsInt, ok := maxRows.(int); ok {
							config.MaxRows = maxRowsInt
						}
					}

					sqlConfigs = append(sqlConfigs, config)
				}
			}

			if len(sqlConfigs) > 0 {
				return sqlConfigs, true
			}
		}
	}

	return nil, false
}

// parseSourceConfigs 解析源库配置
func (tse *ThematicSyncEngine) parseSourceConfigs(request *SyncRequest) ([]SourceLibraryConfig, error) {
	var configs []SourceLibraryConfig

	// 从请求配置中解析源库配置
	if sourceConfigsRaw, exists := request.Config["source_libraries"]; exists {
		// 尝试直接转换
		if configSlice, ok := sourceConfigsRaw.([]SourceLibraryConfig); ok {
			return configSlice, nil
		}

		// 尝试从接口数组转换
		if configSlice, ok := sourceConfigsRaw.([]interface{}); ok {
			for _, configRaw := range configSlice {
				if configMap, ok := configRaw.(map[string]interface{}); ok {
					config := SourceLibraryConfig{
						LibraryID:   tse.getStringFromMap(configMap, "library_id"),
						InterfaceID: tse.getStringFromMap(configMap, "interface_id"),
						SQLQuery:    tse.getStringFromMap(configMap, "sql_query"),
					}

					// 解析参数
					if params, exists := configMap["parameters"]; exists {
						if paramsMap, ok := params.(map[string]interface{}); ok {
							config.Parameters = paramsMap
						}
					}

					configs = append(configs, config)
				}
			}
			return configs, nil
		}
	}

	// 兜底：使用传统的库和接口列表
	for i, interfaceID := range request.SourceInterfaces {
		libraryID := ""
		if i < len(request.SourceLibraries) {
			libraryID = request.SourceLibraries[i]
		}

		configs = append(configs, SourceLibraryConfig{
			LibraryID:   libraryID,
			InterfaceID: interfaceID,
		})
	}

	return configs, nil
}

// applyFilters 应用过滤器
func (tse *ThematicSyncEngine) applyFilters(records []map[string]interface{}, filters []FilterConfig) []map[string]interface{} {
	if len(filters) == 0 {
		return records
	}

	var filtered []map[string]interface{}

	for _, record := range records {
		if tse.matchesFilters(record, filters) {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

// matchesFilters 检查记录是否匹配过滤条件
func (tse *ThematicSyncEngine) matchesFilters(record map[string]interface{}, filters []FilterConfig) bool {
	for _, filter := range filters {
		if !tse.matchesFilter(record, filter) {
			return false // 任意条件不匹配则过滤掉
		}
	}
	return true
}

// matchesFilter 检查单个过滤条件
func (tse *ThematicSyncEngine) matchesFilter(record map[string]interface{}, filter FilterConfig) bool {
	value, exists := record[filter.Field]
	if !exists {
		return false
	}

	valueStr := fmt.Sprintf("%v", value)
	filterValueStr := fmt.Sprintf("%v", filter.Value)

	switch filter.Operator {
	case "eq", "=":
		return valueStr == filterValueStr
	case "ne", "!=":
		return valueStr != filterValueStr
	case "gt", ">":
		return tse.compareValues(value, filter.Value) > 0
	case "lt", "<":
		return tse.compareValues(value, filter.Value) < 0
	case "gte", ">=":
		return tse.compareValues(value, filter.Value) >= 0
	case "lte", "<=":
		return tse.compareValues(value, filter.Value) <= 0
	case "contains":
		return strings.Contains(valueStr, filterValueStr)
	case "not_contains":
		return !strings.Contains(valueStr, filterValueStr)
	case "starts_with":
		return strings.HasPrefix(valueStr, filterValueStr)
	case "ends_with":
		return strings.HasSuffix(valueStr, filterValueStr)
	case "in":
		if filterSlice, ok := filter.Value.([]interface{}); ok {
			for _, filterVal := range filterSlice {
				if fmt.Sprintf("%v", filterVal) == valueStr {
					return true
				}
			}
		}
		return false
	case "not_in":
		if filterSlice, ok := filter.Value.([]interface{}); ok {
			for _, filterVal := range filterSlice {
				if fmt.Sprintf("%v", filterVal) == valueStr {
					return false
				}
			}
		}
		return true
	default:
		return true // 未知操作符默认通过
	}
}

// applyTransforms 应用数据转换
func (tse *ThematicSyncEngine) applyTransforms(records []map[string]interface{}, transforms []TransformConfig) ([]map[string]interface{}, error) {
	if len(transforms) == 0 {
		return records, nil
	}

	transformer := NewDataTransformer()

	for i, record := range records {
		for _, transform := range transforms {
			if sourceValue, exists := record[transform.SourceField]; exists {
				// 执行转换
				transformedValue, err := transformer.Transform(sourceValue, transform.Transform, transform.Config)
				if err != nil {
					return nil, fmt.Errorf("记录 %d 字段 %s 转换失败: %w", i, transform.SourceField, err)
				}

				// 设置目标字段值
				record[transform.TargetField] = transformedValue
			}
		}
	}

	return records, nil
}

// generateRecordID 生成记录ID
func (tse *ThematicSyncEngine) generateRecordID(libraryID, interfaceID string, index int, record map[string]interface{}) string {
	// 尝试使用记录中的主键字段
	keyFields := []string{"id", "uuid", "primary_key", "pk"}

	for _, keyField := range keyFields {
		if value, exists := record[keyField]; exists && value != nil {
			return fmt.Sprintf("%s_%s_%v", libraryID, interfaceID, value)
		}
	}

	// 使用索引生成ID
	return fmt.Sprintf("%s_%s_%d", libraryID, interfaceID, index)
}

// calculateInitialQuality 计算初始质量评分
func (tse *ThematicSyncEngine) calculateInitialQuality(record map[string]interface{}) float64 {
	if len(record) == 0 {
		return 0.0
	}

	validFieldCount := 0
	for _, value := range record {
		if tse.isValidFieldValue(value) {
			validFieldCount++
		}
	}

	return float64(validFieldCount) / float64(len(record))
}

// isValidFieldValue 检查字段值是否有效
func (tse *ThematicSyncEngine) isValidFieldValue(value interface{}) bool {
	if value == nil {
		return false
	}

	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	return str != "" && str != "null" && str != "NULL" && str != "nil"
}

// compareValues 比较两个值
func (tse *ThematicSyncEngine) compareValues(a, b interface{}) int {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// 尝试数值比较
	if aNum, aErr := strconv.ParseFloat(aStr, 64); aErr == nil {
		if bNum, bErr := strconv.ParseFloat(bStr, 64); bErr == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// 字符串比较
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// fetchDataFromSQL 从SQL数据源获取数据
func (tse *ThematicSyncEngine) fetchDataFromSQL(sqlConfigs []SQLDataSourceConfig,
	request *SyncRequest, result *SyncExecutionResult) ([]SourceRecordInfo, error) {

	var sourceRecords []SourceRecordInfo

	// 创建SQL数据源
	sqlDataSource, err := NewSQLDataSource(tse.db)
	if err != nil {
		return nil, fmt.Errorf("创建SQL数据源失败: %w", err)
	}

	// 执行SQL查询获取数据
	for _, config := range sqlConfigs {
		records, err := sqlDataSource.ExecuteQuery(request.Context, config)
		if err != nil {
			return nil, fmt.Errorf("执行SQL查询失败 [%s/%s]: %w",
				config.LibraryID, config.InterfaceID, err)
		}

		// 转换为源记录信息
		for j, record := range records {
			sourceRecord := SourceRecordInfo{
				LibraryID:   config.LibraryID,
				InterfaceID: config.InterfaceID,
				RecordID:    tse.generateRecordID(config.LibraryID, config.InterfaceID, j, record),
				Record:      record,
				Quality:     tse.calculateInitialQuality(record),
				LastUpdated: time.Now(),
				Metadata: map[string]interface{}{
					"data_source_type": "sql",
					"sql_config":       config,
					"fetch_time":       time.Now(),
				},
			}
			sourceRecords = append(sourceRecords, sourceRecord)
		}
	}

	result.SourceRecordCount = int64(len(sourceRecords))
	return sourceRecords, nil
}

// fetchDataFromBasicLibrary 从基础库接口获取数据
func (tse *ThematicSyncEngine) fetchDataFromBasicLibrary(request *SyncRequest,
	result *SyncExecutionResult) ([]SourceRecordInfo, error) {

	var sourceRecords []SourceRecordInfo

	// 从配置中获取源库配置
	sourceConfigs, err := tse.parseSourceConfigs(request)
	if err != nil {
		return nil, fmt.Errorf("解析源库配置失败: %w", err)
	}

	// 创建SQL数据源（用于配置了SQL的源库）
	sqlDataSource, err := NewSQLDataSource(tse.db)
	if err != nil {
		return nil, fmt.Errorf("创建SQL数据源失败: %w", err)
	}

	// 批量获取数据
	for _, config := range sourceConfigs {
		var records []map[string]interface{}

		if config.SQLQuery != "" {
			// 使用SQL查询获取数据
			sqlConfig := SQLDataSourceConfig{
				LibraryID:   config.LibraryID,
				InterfaceID: config.InterfaceID,
				SQLQuery:    config.SQLQuery,
				Parameters:  config.Parameters,
				Timeout:     30,    // 默认30秒超时
				MaxRows:     10000, // 默认最大1万行
			}

			records, err = sqlDataSource.ExecuteQuery(request.Context, sqlConfig)
			if err != nil {
				return nil, fmt.Errorf("执行SQL查询失败 [%s]: %w", config.LibraryID, err)
			}
		} else {
			// 使用基础库服务获取数据
			records, err = tse.basicLibraryService.GetDataByInterface(
				request.Context, config.LibraryID, config.InterfaceID)
			if err != nil {
				return nil, fmt.Errorf("获取接口数据失败 [%s/%s]: %w",
					config.LibraryID, config.InterfaceID, err)
			}
		}

		// 应用过滤器
		if len(config.Filters) > 0 {
			records = tse.applyFilters(records, config.Filters)
		}

		// 应用转换
		if len(config.Transforms) > 0 {
			records, err = tse.applyTransforms(records, config.Transforms)
			if err != nil {
				return nil, fmt.Errorf("应用数据转换失败: %w", err)
			}
		}

		// 转换为源记录信息
		for j, record := range records {
			sourceRecord := SourceRecordInfo{
				LibraryID:   config.LibraryID,
				InterfaceID: config.InterfaceID,
				RecordID:    tse.generateRecordID(config.LibraryID, config.InterfaceID, j, record),
				Record:      record,
				Quality:     tse.calculateInitialQuality(record),
				LastUpdated: time.Now(),
				Metadata: map[string]interface{}{
					"data_source_type": "basic_library",
					"source_config":    config,
					"fetch_time":       time.Now(),
				},
			}
			sourceRecords = append(sourceRecords, sourceRecord)
		}
	}

	result.SourceRecordCount = int64(len(sourceRecords))
	return sourceRecords, nil
}

// getStringFromMap 从map中获取字符串值
func (tse *ThematicSyncEngine) getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}
