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
	"datahub-service/service/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncExecutionPhase 同步执行阶段
type SyncExecutionPhase string

const (
	PhaseInitialize   SyncExecutionPhase = "initialize"    // 初始化
	PhaseDataFetch    SyncExecutionPhase = "data_fetch"    // 数据获取
	PhaseAggregation  SyncExecutionPhase = "aggregation"   // 数据汇聚
	PhaseCleansing    SyncExecutionPhase = "cleansing"     // 数据清洗
	PhasePrivacy      SyncExecutionPhase = "privacy"       // 隐私处理
	PhaseQualityCheck SyncExecutionPhase = "quality_check" // 质量检查
	PhaseDataWrite    SyncExecutionPhase = "data_write"    // 数据写入
	PhaseLineage      SyncExecutionPhase = "lineage"       // 血缘记录
	PhaseComplete     SyncExecutionPhase = "complete"      // 完成
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

// ThematicSyncEngine 主题同步引擎
type ThematicSyncEngine struct {
	db                     *gorm.DB
	aggregationEngine      *AggregationEngine
	cleansingEngine        *CleansingEngine
	privacyEngine          *PrivacyEngine
	basicLibraryService    BasicLibraryServiceInterface
	thematicLibraryService ThematicLibraryServiceInterface
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
	thematicLibraryService ThematicLibraryServiceInterface) *ThematicSyncEngine {

	return &ThematicSyncEngine{
		db:                     db,
		aggregationEngine:      NewAggregationEngine(db),
		cleansingEngine:        NewCleansingEngine(),
		privacyEngine:          NewPrivacyEngine(),
		basicLibraryService:    basicLibraryService,
		thematicLibraryService: thematicLibraryService,
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

	// 4. 数据清洗阶段
	var cleansingResults []CleansingResult
	if err := tse.executePhase(PhaseCleansing, progress, func() error {
		var err error
		cleansingResults, err = tse.performCleansing(aggregationResult.AggregatedRecords, request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 5. 隐私处理阶段
	var maskingResults []MaskingResult
	if err := tse.executePhase(PhasePrivacy, progress, func() error {
		var err error
		maskingResults, err = tse.performPrivacyProcessing(cleansingResults, request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 6. 质量检查阶段
	if err := tse.executePhase(PhaseQualityCheck, progress, func() error {
		return tse.performQualityCheck(maskingResults, request, result)
	}); err != nil {
		return nil, err
	}

	// 7. 数据写入阶段
	if err := tse.executePhase(PhaseDataWrite, progress, func() error {
		return tse.writeData(maskingResults, request, result)
	}); err != nil {
		return nil, err
	}

	// 8. 血缘记录阶段
	if err := tse.executePhase(PhaseLineage, progress, func() error {
		return tse.recordLineage(aggregationResult, request, result)
	}); err != nil {
		return nil, err
	}

	// 9. 完成阶段
	if err := tse.executePhase(PhaseComplete, progress, func() error {
		return tse.completeSync(request, result)
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

	var sourceRecords []SourceRecordInfo

	for i, interfaceID := range request.SourceInterfaces {
		libraryID := ""
		if i < len(request.SourceLibraries) {
			libraryID = request.SourceLibraries[i]
		}

		// 获取接口数据
		records, err := tse.basicLibraryService.GetDataByInterface(request.Context, libraryID, interfaceID)
		if err != nil {
			return nil, fmt.Errorf("获取接口 %s 数据失败: %w", interfaceID, err)
		}

		// 转换为源记录信息
		for j, record := range records {
			sourceRecord := SourceRecordInfo{
				LibraryID:   libraryID,
				InterfaceID: interfaceID,
				RecordID:    fmt.Sprintf("%s_%d", interfaceID, j),
				Record:      record,
				Quality:     1.0, // 默认质量评分
				LastUpdated: time.Now(),
			}
			sourceRecords = append(sourceRecords, sourceRecord)
		}
	}

	result.SourceRecordCount = int64(len(sourceRecords))
	return sourceRecords, nil
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

// performCleansing 执行数据清洗
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
func (tse *ThematicSyncEngine) writeData(maskingResults []MaskingResult,
	request *SyncRequest, result *SyncExecutionResult) error {

	// 提取脱敏后的记录
	var maskedRecords []map[string]interface{}
	for _, maskingResult := range maskingResults {
		maskedRecords = append(maskedRecords, maskingResult.MaskedRecord)
	}

	// 写入目标接口
	err := tse.thematicLibraryService.WriteData(request.Context, request.TargetInterfaceID, maskedRecords)
	if err != nil {
		return fmt.Errorf("写入数据失败: %w", err)
	}

	result.ProcessedRecordCount = int64(len(maskedRecords))
	result.InsertedRecordCount = int64(len(maskedRecords)) // 简化处理，假设都是新增

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
	result *SyncExecutionResult) error {

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
