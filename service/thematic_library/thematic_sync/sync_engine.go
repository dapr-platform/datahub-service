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
	"hash/fnv"
	"sort"
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
	ApplyGovernanceRules(ctx context.Context, records []map[string]interface{}, task *models.ThematicSyncTask, config *GovernanceExecutionConfig) ([]map[string]interface{}, *GovernanceExecutionResult, error)
}

// GovernanceExecutionConfig 数据治理执行配置
type GovernanceExecutionConfig struct {
	EnableQualityCheck   bool                   `json:"enable_quality_check"`
	EnableCleansing      bool                   `json:"enable_cleansing"`
	EnableMasking        bool                   `json:"enable_masking"`
	StopOnQualityFailure bool                   `json:"stop_on_quality_failure"`
	QualityThreshold     float64                `json:"quality_threshold"`
	BatchSize            int                    `json:"batch_size"`
	MaxRetries           int                    `json:"max_retries"`
	TimeoutSeconds       int                    `json:"timeout_seconds"`
	CustomConfig         map[string]interface{} `json:"custom_config,omitempty"`
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
	db                    *gorm.DB
	governanceIntegration GovernanceIntegrationServiceInterface
	progressCallback      func(*SyncProgress)
}

// 移除不必要的接口抽象，直接使用数据库操作

// NewThematicSyncEngine 创建主题同步引擎
func NewThematicSyncEngine(db *gorm.DB,
	governanceIntegration GovernanceIntegrationServiceInterface) *ThematicSyncEngine {

	return &ThematicSyncEngine{
		db:                    db,
		governanceIntegration: governanceIntegration,
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

	// 3. 数据合并阶段（基于主键ID简单合并）
	var mergedRecords []map[string]interface{}
	if err := tse.executePhase(PhaseAggregation, progress, func() error {
		var err error
		mergedRecords, err = tse.performSimpleDataMerge(sourceRecords, request, result)
		return err
	}); err != nil {
		return nil, err
	}

	// 4. 数据治理处理阶段 - 统一处理数据质量、清洗、脱敏、转换、校验
	var processedRecords []map[string]interface{}
	var governanceResult *GovernanceExecutionResult
	if err := tse.executePhase(PhaseGovernance, progress, func() error {
		var err error
		processedRecords, governanceResult, err = tse.performGovernanceProcessing(mergedRecords, request, result)
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
		return tse.recordSimpleLineage(sourceRecords, processedRecords, request, result)
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
	if len(request.SourceInterfaces) == 0 && len(request.SourceLibraries) == 0 {
		return fmt.Errorf("源库或源接口列表不能为空")
	}

	if request.TargetInterfaceID == "" {
		return fmt.Errorf("目标接口ID不能为空")
	}

	// 验证目标接口是否存在
	var targetInterface models.ThematicInterface
	if err := tse.db.First(&targetInterface, "id = ?", request.TargetInterfaceID).Error; err != nil {
		return fmt.Errorf("获取目标接口信息失败: %w", err)
	}

	return nil
}

// fetchSourceData 获取源数据 - 简化为直接数据库查询
func (tse *ThematicSyncEngine) fetchSourceData(request *SyncRequest,
	result *SyncExecutionResult) ([]SourceRecordInfo, error) {

	var sourceRecords []SourceRecordInfo

	// 从配置中获取源库配置
	sourceConfigs, err := tse.parseSourceConfigs(request)
	if err != nil {
		return nil, fmt.Errorf("解析源库配置失败: %w", err)
	}

	// 直接查询每个源接口的数据
	for _, config := range sourceConfigs {
		records, err := tse.fetchDataFromInterface(config.LibraryID, config.InterfaceID)
		if err != nil {
			return nil, fmt.Errorf("获取接口数据失败 [%s/%s]: %w",
				config.LibraryID, config.InterfaceID, err)
		}

		// 应用过滤器和转换
		if len(config.Filters) > 0 {
			records = tse.applyFilters(records, config.Filters)
		}

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
					"data_source_type": "direct_query",
					"fetch_time":       time.Now(),
				},
			}
			sourceRecords = append(sourceRecords, sourceRecord)
		}
	}

	result.SourceRecordCount = int64(len(sourceRecords))
	return sourceRecords, nil
}

// fetchDataFromInterface 直接从接口查询数据
func (tse *ThematicSyncEngine) fetchDataFromInterface(libraryID, interfaceID string) ([]map[string]interface{}, error) {
	// 获取接口配置信息
	var dataInterface models.DataInterface
	if err := tse.db.Preload("BasicLibrary").First(&dataInterface, "id = ?", interfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取接口信息失败: %w", err)
	}

	// 验证基础库信息
	if dataInterface.BasicLibrary.NameEn == "" {
		return nil, fmt.Errorf("基础库英文名为空")
	}
	if dataInterface.NameEn == "" {
		return nil, fmt.Errorf("基础接口英文名为空")
	}

	// 构建表名：基础库的name_en作为schema，基础接口的name_en作为表名
	schema := dataInterface.BasicLibrary.NameEn
	tableName := dataInterface.NameEn
	fullTableName := fmt.Sprintf("%s.%s", schema, tableName)

	var records []map[string]interface{}
	rows, err := tse.db.Raw(fmt.Sprintf("SELECT * FROM %s LIMIT 10000", fullTableName)).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询数据失败: %w", err)
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %w", err)
	}

	// 扫描数据
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("扫描数据失败: %w", err)
		}

		record := make(map[string]interface{})
		for i, column := range columns {
			record[column] = tse.convertDatabaseValue(values[i])
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历数据失败: %w", err)
	}

	return records, nil
}

// convertDatabaseValue 转换数据库值
func (tse *ThematicSyncEngine) convertDatabaseValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return v
	}
}

// performSimpleDataMerge 执行简单的数据合并（基于主键ID）
func (tse *ThematicSyncEngine) performSimpleDataMerge(sourceRecords []SourceRecordInfo,
	request *SyncRequest, result *SyncExecutionResult) ([]map[string]interface{}, error) {

	// 调试：打印源记录数量
	fmt.Printf("[DEBUG] 源记录数量: %d\n", len(sourceRecords))

	// 获取目标主题接口的主键字段
	targetPrimaryKeys, err := tse.getThematicPrimaryKeyFields(request.TargetInterfaceID)
	if err != nil {
		fmt.Printf("[DEBUG] 获取目标主键字段失败: %v, 使用默认主键\n", err)
		targetPrimaryKeys = []string{"id"}
	}
	fmt.Printf("[DEBUG] 目标主键字段: %v\n", targetPrimaryKeys)

	// 使用map按ID合并数据
	recordMap := make(map[string]map[string]interface{})

	for _, sourceRecord := range sourceRecords {
		// 根据目标接口的主键字段提取记录ID
		id := tse.extractPrimaryKeyByFields(sourceRecord.Record, targetPrimaryKeys)
		if id == "" {
			// 如果没有主键，使用记录的哈希值作为ID
			id = tse.generateRecordHash(sourceRecord.Record)
			fmt.Printf("[DEBUG] 记录缺少主键字段，使用哈希ID: %s\n", id)
		}

		// 如果已存在相同ID的记录，合并字段
		if existingRecord, exists := recordMap[id]; exists {
			// 合并字段，新数据覆盖旧数据
			for key, value := range sourceRecord.Record {
				existingRecord[key] = value
			}
		} else {
			// 复制记录数据
			recordData := make(map[string]interface{})
			for key, value := range sourceRecord.Record {
				recordData[key] = value
			}
			recordMap[id] = recordData
		}
	}

	// 将map转换为切片
	mergedRecords := make([]map[string]interface{}, 0, len(recordMap))
	for _, record := range recordMap {
		mergedRecords = append(mergedRecords, record)
	}

	// 调试：打印合并结果
	fmt.Printf("[DEBUG] 合并结果记录数量: %d\n", len(mergedRecords))
	for i, record := range mergedRecords {
		fmt.Printf("[DEBUG] 合并记录 %d: %v\n", i, record)
		if i >= 2 { // 只打印前3条记录，避免日志太长
			break
		}
	}

	// 更新处理记录数
	result.ProcessedRecordCount = int64(len(mergedRecords))

	return mergedRecords, nil
}

// extractPrimaryKey 提取记录的主键ID
func (tse *ThematicSyncEngine) extractPrimaryKey(record map[string]interface{}) string {
	// 尝试常见的主键字段名
	primaryKeyFields := []string{"id", "ID", "_id", "uuid", "UUID", "primary_key"}

	for _, field := range primaryKeyFields {
		if value, exists := record[field]; exists && value != nil {
			return fmt.Sprintf("%v", value)
		}
	}

	return ""
}

// getPrimaryKeyFields 获取接口的主键字段列表
func (tse *ThematicSyncEngine) getPrimaryKeyFields(interfaceID string) ([]string, error) {
	// 获取接口字段信息
	var interfaceFields []models.InterfaceField
	if err := tse.db.Where("interface_id = ? AND is_primary_key = ?", interfaceID, true).
		Find(&interfaceFields).Error; err != nil {
		return nil, fmt.Errorf("获取接口主键字段失败: %w", err)
	}

	var primaryKeys []string
	for _, field := range interfaceFields {
		primaryKeys = append(primaryKeys, field.NameEn)
	}

	// 如果没有找到主键字段，尝试从TableFieldsConfig中获取
	if len(primaryKeys) == 0 {
		var dataInterface models.DataInterface
		if err := tse.db.First(&dataInterface, "id = ?", interfaceID).Error; err != nil {
			return nil, fmt.Errorf("获取接口信息失败: %w", err)
		}

		// 从TableFieldsConfig中解析主键字段
		if len(dataInterface.TableFieldsConfig) > 0 {
			var tableFields []models.TableField
			if err := json.Unmarshal([]byte(fmt.Sprintf("%s", dataInterface.TableFieldsConfig)), &tableFields); err == nil {
				for _, field := range tableFields {
					if field.IsPrimaryKey {
						primaryKeys = append(primaryKeys, field.NameEn)
					}
				}
			}
		}
	}

	// 如果还是没有主键，使用默认的id字段
	if len(primaryKeys) == 0 {
		primaryKeys = []string{"id"}
	}

	return primaryKeys, nil
}

// getThematicPrimaryKeyFields 获取主题接口的主键字段列表
func (tse *ThematicSyncEngine) getThematicPrimaryKeyFields(thematicInterfaceID string) ([]string, error) {
	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := tse.db.First(&thematicInterface, "id = ?", thematicInterfaceID).Error; err != nil {
		return nil, fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	var primaryKeys []string

	// 从TableFieldsConfig中解析主键字段
	if len(thematicInterface.TableFieldsConfig) > 0 {
		var tableFields []models.TableField
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", thematicInterface.TableFieldsConfig)), &tableFields); err == nil {
			for _, field := range tableFields {
				if field.IsPrimaryKey {
					primaryKeys = append(primaryKeys, field.NameEn)
				}
			}
		}
	}

	// 如果没有主键，使用默认的id字段
	if len(primaryKeys) == 0 {
		primaryKeys = []string{"id"}
	}

	return primaryKeys, nil
}

// extractPrimaryKeyByFields 根据指定字段提取主键值
func (tse *ThematicSyncEngine) extractPrimaryKeyByFields(record map[string]interface{}, primaryKeyFields []string) string {
	var keyParts []string

	for _, field := range primaryKeyFields {
		if value, exists := record[field]; exists && value != nil {
			keyParts = append(keyParts, fmt.Sprintf("%v", value))
		} else {
			// 如果任一主键字段缺失，返回空字符串
			return ""
		}
	}

	// 如果是复合主键，用下划线连接
	if len(keyParts) > 1 {
		return strings.Join(keyParts, "_")
	} else if len(keyParts) == 1 {
		return keyParts[0]
	}

	return ""
}

// generateRecordHash 生成记录的哈希值作为ID
func (tse *ThematicSyncEngine) generateRecordHash(record map[string]interface{}) string {
	// 将记录转换为字符串并生成哈希
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}

	// 排序确保一致性
	sort.Strings(keys)

	var builder strings.Builder
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(":")
		builder.WriteString(fmt.Sprintf("%v", record[k]))
		builder.WriteString(";")
	}

	// 使用简单的哈希算法
	h := fnv.New32a()
	h.Write([]byte(builder.String()))
	return fmt.Sprintf("hash_%x", h.Sum32())
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
				EnableQualityCheck:   true,
				EnableCleansing:      true,
				EnableMasking:        false,
				StopOnQualityFailure: false,
				QualityThreshold:     0.8,
				BatchSize:            1000,
				MaxRetries:           3,
				TimeoutSeconds:       300,
			}
		}
	} else {
		// 使用默认配置
		governanceConfig = &GovernanceExecutionConfig{
			EnableQualityCheck:   true,
			EnableCleansing:      true,
			EnableMasking:        false,
			StopOnQualityFailure: false,
			QualityThreshold:     0.8,
			BatchSize:            1000,
			MaxRetries:           3,
			TimeoutSeconds:       300,
		}
	}

	// 使用数据治理集成服务处理数据
	processedRecords, governanceResult, err := tse.governanceIntegration.ApplyGovernanceRules(
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

	return processedRecords, governanceResult, nil
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

	// 构建清洗规则（已简化，仅作示例）
	_ = []CleansingRule{
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

	// 已简化，不再使用独立的清洗引擎
	cleansingResults := make([]CleansingResult, len(records))
	for i, record := range records {
		cleansingResults[i] = CleansingResult{
			OriginalRecord:   record,
			CleanedRecord:    record,
			AppliedRules:     []string{},
			ValidationErrors: []ValidationError{},
			QualityScore:     100.0,
			ProcessingTime:   0,
		}
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

	// 构建隐私规则（已简化，仅作示例）
	_ = []PrivacyRule{
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

	// 已简化，不再使用独立的脱敏引擎
	maskingResults := make([]MaskingResult, len(cleanedRecords))
	for i, record := range cleanedRecords {
		maskingResults[i] = MaskingResult{
			OriginalRecord: record,
			MaskedRecord:   record,
			AppliedRules:   []string{},
			MaskingLog:     []MaskingLogEntry{},
			ProcessingTime: 0,
		}
	}
	return maskingResults, nil
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

// writeData 写入数据 - 直接写入主题表
func (tse *ThematicSyncEngine) writeData(processedRecords []map[string]interface{},
	request *SyncRequest, result *SyncExecutionResult, governanceResult *GovernanceExecutionResult) error {

	// 获取主题接口信息
	var thematicInterface models.ThematicInterface
	if err := tse.db.Preload("ThematicLibrary").First(&thematicInterface, "id = ?", request.TargetInterfaceID).Error; err != nil {
		return fmt.Errorf("获取主题接口信息失败: %w", err)
	}

	// 验证主题库信息
	if thematicInterface.ThematicLibrary.NameEn == "" {
		return fmt.Errorf("主题库英文名为空")
	}
	if thematicInterface.NameEn == "" {
		return fmt.Errorf("主题接口英文名为空")
	}

	if len(processedRecords) == 0 {
		return nil // 没有数据需要写入
	}

	// 构建表名：主题库的name_en作为schema，主题接口的name_en作为表名
	schema := thematicInterface.ThematicLibrary.NameEn
	tableName := thematicInterface.NameEn
	fullTableName := fmt.Sprintf("%s.%s", schema, tableName)

	// 获取主题接口的主键字段
	primaryKeyFields, err := tse.getThematicPrimaryKeyFields(request.TargetInterfaceID)
	if err != nil {
		fmt.Printf("[DEBUG] 获取主题接口主键字段失败: %v, 使用默认主键\n", err)
		primaryKeyFields = []string{"id"}
	}
	fmt.Printf("[DEBUG] 主题接口主键字段: %v\n", primaryKeyFields)

	// 批量写入数据
	insertedCount := int64(0)
	for _, record := range processedRecords {
		if len(record) == 0 {
			continue
		}

		// 确保为NOT NULL字段提供默认值
		record = tse.ensureRequiredFields(record)

		// 构建插入SQL
		columns := make([]string, 0, len(record))
		placeholders := make([]string, 0, len(record))
		values := make([]interface{}, 0, len(record))

		paramIndex := 1
		for k, v := range record {
			if k != "" { // 过滤空列名
				columns = append(columns, k)
				placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
				values = append(values, tse.convertValueForDatabase(v))
				paramIndex++
			}
		}

		if len(columns) == 0 {
			continue
		}

		updateClause := tse.generateUpdateClauseWithPrimaryKeys(columns, primaryKeyFields)
		conflictColumns := tse.buildConflictColumns(primaryKeyFields)
		var sql string
		if updateClause != "" {
			sql = fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
				fullTableName,
				strings.Join(columns, "\", \""),
				strings.Join(placeholders, ", "),
				conflictColumns,
				updateClause)
		} else {
			// 如果没有可更新的列，使用DO NOTHING
			sql = fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s) ON CONFLICT (%s) DO NOTHING",
				fullTableName,
				strings.Join(columns, "\", \""),
				strings.Join(placeholders, ", "),
				conflictColumns)
		}

		// 调试：打印SQL语句和值
		fmt.Printf("[DEBUG] 写入SQL: %s\n", sql)
		fmt.Printf("[DEBUG] 写入值: %v\n", values)
		fmt.Printf("[DEBUG] Update clause: %s\n", updateClause)

		result := tse.db.Exec(sql, values...)
		if result.Error != nil {
			return fmt.Errorf("写入数据到表 %s 失败: %w", fullTableName, result.Error)
		}
		fmt.Printf("[DEBUG] 执行结果: 影响行数 %d\n", result.RowsAffected)
		insertedCount++
	}

	result.ProcessedRecordCount = int64(len(processedRecords))
	result.InsertedRecordCount = insertedCount

	return nil
}

// generateUpdateClause 生成UPDATE子句 (保持向后兼容)
func (tse *ThematicSyncEngine) generateUpdateClause(columns []string) string {
	return tse.generateUpdateClauseWithPrimaryKeys(columns, []string{"id"})
}

// generateUpdateClauseWithPrimaryKeys 生成UPDATE子句，跳过指定的主键字段
func (tse *ThematicSyncEngine) generateUpdateClauseWithPrimaryKeys(columns []string, primaryKeyFields []string) string {
	// 创建主键字段映射，用于快速查找
	primaryKeyMap := make(map[string]bool)
	for _, pk := range primaryKeyFields {
		primaryKeyMap[pk] = true
	}

	var updateParts []string
	for _, column := range columns {
		if !primaryKeyMap[column] { // 跳过主键字段
			updateParts = append(updateParts, fmt.Sprintf("\"%s\" = EXCLUDED.\"%s\"", column, column))
		}
	}
	return strings.Join(updateParts, ", ")
}

// buildConflictColumns 构建ON CONFLICT子句中的列名部分
func (tse *ThematicSyncEngine) buildConflictColumns(primaryKeyFields []string) string {
	var quotedFields []string
	for _, field := range primaryKeyFields {
		quotedFields = append(quotedFields, fmt.Sprintf("\"%s\"", field))
	}
	return strings.Join(quotedFields, ", ")
}

// convertValueForDatabase 转换值用于数据库写入
func (tse *ThematicSyncEngine) convertValueForDatabase(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// 处理布尔值转换
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return v
	case float64:
		return v
	case string:
		return v
	case bool:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ensureRequiredFields 确保为NOT NULL字段提供默认值
func (tse *ThematicSyncEngine) ensureRequiredFields(record map[string]interface{}) map[string]interface{} {
	// 创建记录副本
	result := make(map[string]interface{})
	for k, v := range record {
		result[k] = v
	}

	// 确保必需字段有值
	if result["created_by"] == nil || result["created_by"] == "" {
		result["created_by"] = "system"
	}
	if result["updated_by"] == nil || result["updated_by"] == "" {
		result["updated_by"] = "system"
	}
	if result["created_time"] == nil {
		result["created_time"] = time.Now()
	}
	if result["updated_time"] == nil {
		result["updated_time"] = time.Now()
	}
	if result["group_id"] == nil {
		result["group_id"] = ""
	}
	if result["parent_id"] == nil {
		result["parent_id"] = ""
	}
	if result["product_id"] == nil {
		result["product_id"] = ""
	}
	if result["protocol_config"] == nil {
		result["protocol_config"] = ""
	}

	// 处理布尔类型字段的转换
	if result["enabled"] != nil {
		if v, ok := result["enabled"].(int); ok {
			result["enabled"] = v != 0
		} else if v, ok := result["enabled"].(int64); ok {
			result["enabled"] = v != 0
		}
	} else {
		result["enabled"] = false
	}

	if result["status"] != nil {
		if v, ok := result["status"].(int); ok {
			result["status"] = v != 0
		} else if v, ok := result["status"].(int64); ok {
			result["status"] = v != 0
		}
	} else {
		result["status"] = false
	}

	if result["type"] != nil {
		if v, ok := result["type"].(int); ok {
			result["type"] = v != 0
		} else if v, ok := result["type"].(int64); ok {
			result["type"] = v != 0
		}
	} else {
		result["type"] = false
	}

	return result
}

// recordSimpleLineage 记录简单的血缘关系
func (tse *ThematicSyncEngine) recordSimpleLineage(sourceRecords []SourceRecordInfo,
	processedRecords []map[string]interface{}, request *SyncRequest, result *SyncExecutionResult) error {

	// 获取目标主题接口的主键字段
	targetPrimaryKeys, err := tse.getThematicPrimaryKeyFields(request.TargetInterfaceID)
	if err != nil {
		fmt.Printf("[DEBUG] 获取目标主键字段失败: %v, 使用默认主键\n", err)
		targetPrimaryKeys = []string{"id"}
	}

	// 为每个处理后的记录创建血缘记录
	for i, processedRecord := range processedRecords {
		// 根据目标接口的主键字段提取记录ID
		targetRecordID := tse.extractPrimaryKeyByFields(processedRecord, targetPrimaryKeys)
		if targetRecordID == "" {
			targetRecordID = fmt.Sprintf("record_%d", i)
		}

		lineage := &models.ThematicDataLineage{
			ID:                    uuid.New().String(),
			ThematicInterfaceID:   request.TargetInterfaceID,
			ThematicRecordID:      targetRecordID,
			ProcessingRules:       models.JSONB{},
			TransformationDetails: models.JSONB{},
			QualityScore:          result.QualityScore,
			QualityIssues:         models.JSONB{},
			SourceDataTime:        time.Now(),
			ProcessedTime:         time.Now(),
			CreatedAt:             time.Now(),
		}

		// 设置源数据信息（使用第一个源记录作为代表）
		if len(sourceRecords) > 0 {
			source := sourceRecords[0]
			lineage.SourceLibraryID = source.LibraryID
			lineage.SourceInterfaceID = source.InterfaceID
			lineage.SourceRecordID = source.RecordID
		}

		if err := tse.db.Create(lineage).Error; err != nil {
			return fmt.Errorf("创建血缘记录失败: %w", err)
		}
	}

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

	// 兜底：如果没有源库配置但有接口列表，构建简单配置
	if len(configs) == 0 && len(request.SourceInterfaces) > 0 {
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

// 已移除复杂的数据源抽象方法，直接使用数据库查询

// getStringFromMap 从map中获取字符串值
func (tse *ThematicSyncEngine) getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}
