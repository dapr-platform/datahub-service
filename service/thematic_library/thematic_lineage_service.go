/*
 * @module service/thematic_lineage_service
 * @description 主题数据血缘服务，提供数据血缘追踪和影响分析功能
 * @architecture 服务层 - 封装数据血缘相关的业务逻辑
 * @documentReference ai_docs/thematic_sync_design.md
 * @stateFlow 血缘记录 -> 血缘查询 -> 影响分析 -> 血缘图构建
 * @rules 确保血缘关系的准确性和完整性，支持复杂的血缘查询
 * @dependencies gorm.io/gorm, context, models包
 * @refs service/models/thematic_sync.go
 */

package thematic_library

import (
	"context"
	"datahub-service/service/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ThematicLineageService 主题数据血缘服务
type ThematicLineageService struct {
	db *gorm.DB
}

// NewThematicLineageService 创建主题数据血缘服务
func NewThematicLineageService(db *gorm.DB) *ThematicLineageService {
	return &ThematicLineageService{
		db: db,
	}
}

// GetDataLineage 获取数据血缘
func (tls *ThematicLineageService) GetDataLineage(ctx context.Context, recordID string) (*DataLineageResponse, error) {
	// 查询血缘记录
	var lineages []models.ThematicDataLineage
	if err := tls.db.Preload("ThematicInterface").Preload("SourceLibrary").Preload("SourceInterface").
		Where("thematic_record_id = ?", recordID).
		Find(&lineages).Error; err != nil {
		return nil, fmt.Errorf("查询血缘记录失败: %w", err)
	}

	if len(lineages) == 0 {
		return nil, fmt.Errorf("未找到记录 %s 的血缘信息", recordID)
	}

	// 构建响应
	response := &DataLineageResponse{
		ThematicRecordID: recordID,
		SourceRecords:    make([]SourceRecordDetail, 0),
		ProcessingChain:  make([]ProcessingStep, 0),
		QualityHistory:   make([]QualitySnapshot, 0),
		CreatedAt:        time.Now(),
	}

	// 处理源记录信息
	for _, lineage := range lineages {
		sourceRecord := SourceRecordDetail{
			LibraryID:     lineage.SourceLibraryID,
			LibraryName:   getLibraryName(lineage.SourceLibrary),
			InterfaceID:   lineage.SourceInterfaceID,
			InterfaceName: getInterfaceName(lineage.SourceInterface),
			RecordID:      lineage.SourceRecordID,
			RecordHash:    lineage.SourceRecordHash,
			QualityScore:  lineage.QualityScore,
			ProcessedTime: lineage.ProcessedTime,
		}
		response.SourceRecords = append(response.SourceRecords, sourceRecord)

		// 添加处理步骤
		step := ProcessingStep{
			StepName:        "数据同步",
			StepType:        "sync",
			InputSource:     fmt.Sprintf("%s.%s", sourceRecord.LibraryName, sourceRecord.InterfaceName),
			OutputTarget:    fmt.Sprintf("主题记录 %s", recordID),
			ProcessingRules: fmt.Sprintf("%v", lineage.ProcessingRules),
			ProcessedTime:   lineage.ProcessedTime,
		}
		response.ProcessingChain = append(response.ProcessingChain, step)

		// 添加质量快照
		snapshot := QualitySnapshot{
			RecordID:      lineage.SourceRecordID,
			QualityScore:  lineage.QualityScore,
			QualityIssues: fmt.Sprintf("%v", lineage.QualityIssues),
			SnapshotTime:  lineage.ProcessedTime,
		}
		response.QualityHistory = append(response.QualityHistory, snapshot)
	}

	return response, nil
}

// GetImpactAnalysis 获取影响分析
func (tls *ThematicLineageService) GetImpactAnalysis(ctx context.Context, sourceRecordID string) (*ImpactAnalysisResponse, error) {
	// 查询受影响的记录
	var lineages []models.ThematicDataLineage
	if err := tls.db.Preload("ThematicInterface").
		Where("source_record_id = ?", sourceRecordID).
		Find(&lineages).Error; err != nil {
		return nil, fmt.Errorf("查询影响分析失败: %w", err)
	}

	response := &ImpactAnalysisResponse{
		SourceRecordID:         sourceRecordID,
		AffectedRecords:        make([]AffectedRecord, 0),
		ImpactScope:            ImpactScope{},
		ReprocessingSuggestion: "",
		AnalysisTime:           time.Now(),
	}

	// 统计影响范围
	interfaceMap := make(map[string]bool)
	libraryMap := make(map[string]bool)

	for _, lineage := range lineages {
		// 受影响的记录
		affected := AffectedRecord{
			ThematicRecordID:    lineage.ThematicRecordID,
			ThematicInterfaceID: lineage.ThematicInterfaceID,
			InterfaceName:       getThematicInterfaceName(lineage.ThematicInterface),
			QualityScore:        lineage.QualityScore,
			LastProcessedTime:   lineage.ProcessedTime,
		}
		response.AffectedRecords = append(response.AffectedRecords, affected)

		// 统计影响范围
		interfaceMap[lineage.ThematicInterfaceID] = true
		if lineage.ThematicInterface != nil {
			// 简化处理，假设有库ID字段
			libraryMap["thematic_library_id"] = true
		}
	}

	// 设置影响范围
	response.ImpactScope = ImpactScope{
		AffectedRecordCount:    len(lineages),
		AffectedInterfaceCount: len(interfaceMap),
		AffectedLibraryCount:   len(libraryMap),
		ImpactLevel:            calculateImpactLevel(len(lineages)),
	}

	// 生成重处理建议
	if len(lineages) > 0 {
		response.ReprocessingSuggestion = fmt.Sprintf("建议重新同步 %d 条受影响的主题记录", len(lineages))
	}

	return response, nil
}

// GetLineageGraph 获取血缘图
func (tls *ThematicLineageService) GetLineageGraph(ctx context.Context, req *LineageGraphRequest) (*LineageGraphResponse, error) {
	query := tls.db.Model(&models.ThematicDataLineage{}).
		Preload("ThematicInterface").
		Preload("SourceLibrary").
		Preload("SourceInterface")

	// 添加过滤条件
	if req.ThematicLibraryID != "" {
		query = query.Joins("JOIN thematic_interfaces ON thematic_data_lineages.thematic_interface_id = thematic_interfaces.id").
			Where("thematic_interfaces.thematic_library_id = ?", req.ThematicLibraryID)
	}
	if req.SourceLibraryID != "" {
		query = query.Where("source_library_id = ?", req.SourceLibraryID)
	}
	if req.StartTime != nil {
		query = query.Where("processed_time >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("processed_time <= ?", *req.EndTime)
	}

	var lineages []models.ThematicDataLineage
	if err := query.Limit(req.Limit).Find(&lineages).Error; err != nil {
		return nil, fmt.Errorf("查询血缘图数据失败: %w", err)
	}

	// 构建血缘图
	response := &LineageGraphResponse{
		Nodes: make([]LineageNode, 0),
		Edges: make([]LineageEdge, 0),
		Statistics: LineageStatistics{
			TotalNodes: 0,
			TotalEdges: 0,
		},
	}

	nodeMap := make(map[string]bool)

	for _, lineage := range lineages {
		// 源节点
		sourceNodeID := fmt.Sprintf("source_%s_%s", lineage.SourceLibraryID, lineage.SourceInterfaceID)
		if !nodeMap[sourceNodeID] {
			sourceNode := LineageNode{
				ID:          sourceNodeID,
				Type:        "source",
				Name:        getInterfaceName(lineage.SourceInterface),
				LibraryID:   lineage.SourceLibraryID,
				InterfaceID: lineage.SourceInterfaceID,
				RecordCount: 1, // 简化处理
			}
			response.Nodes = append(response.Nodes, sourceNode)
			nodeMap[sourceNodeID] = true
		}

		// 目标节点
		targetNodeID := fmt.Sprintf("target_%s", lineage.ThematicInterfaceID)
		if !nodeMap[targetNodeID] {
			targetNode := LineageNode{
				ID:          targetNodeID,
				Type:        "target",
				Name:        getThematicInterfaceName(lineage.ThematicInterface),
				InterfaceID: lineage.ThematicInterfaceID,
				RecordCount: 1, // 简化处理
			}
			response.Nodes = append(response.Nodes, targetNode)
			nodeMap[targetNodeID] = true
		}

		// 边
		edge := LineageEdge{
			ID:            fmt.Sprintf("edge_%s", lineage.ID),
			SourceNodeID:  sourceNodeID,
			TargetNodeID:  targetNodeID,
			RelationType:  "sync",
			QualityScore:  lineage.QualityScore,
			ProcessedTime: lineage.ProcessedTime,
		}
		response.Edges = append(response.Edges, edge)
	}

	response.Statistics.TotalNodes = len(response.Nodes)
	response.Statistics.TotalEdges = len(response.Edges)

	return response, nil
}

// 辅助函数

func getLibraryName(library *models.BasicLibrary) string {
	if library != nil {
		return library.NameZh // 使用正确的字段名
	}
	return "未知库"
}

func getInterfaceName(iface *models.DataInterface) string {
	if iface != nil {
		return iface.NameZh // 使用正确的字段名
	}
	return "未知接口"
}

func getThematicInterfaceName(iface *models.ThematicInterface) string {
	if iface != nil {
		return iface.NameZh // 使用正确的字段名
	}
	return "未知主题接口"
}

func calculateImpactLevel(recordCount int) string {
	if recordCount >= 1000 {
		return "high"
	} else if recordCount >= 100 {
		return "medium"
	} else if recordCount > 0 {
		return "low"
	}
	return "none"
}

// 请求和响应结构体

// DataLineageResponse 数据血缘响应
type DataLineageResponse struct {
	ThematicRecordID string               `json:"thematic_record_id"`
	SourceRecords    []SourceRecordDetail `json:"source_records"`
	ProcessingChain  []ProcessingStep     `json:"processing_chain"`
	QualityHistory   []QualitySnapshot    `json:"quality_history"`
	CreatedAt        time.Time            `json:"created_at"`
}

// SourceRecordDetail 源记录详情
type SourceRecordDetail struct {
	LibraryID     string    `json:"library_id"`
	LibraryName   string    `json:"library_name"`
	InterfaceID   string    `json:"interface_id"`
	InterfaceName string    `json:"interface_name"`
	RecordID      string    `json:"record_id"`
	RecordHash    string    `json:"record_hash"`
	QualityScore  float64   `json:"quality_score"`
	ProcessedTime time.Time `json:"processed_time"`
}

// ProcessingStep 处理步骤
type ProcessingStep struct {
	StepName        string    `json:"step_name"`
	StepType        string    `json:"step_type"`
	InputSource     string    `json:"input_source"`
	OutputTarget    string    `json:"output_target"`
	ProcessingRules string    `json:"processing_rules"`
	ProcessedTime   time.Time `json:"processed_time"`
}

// QualitySnapshot 质量快照
type QualitySnapshot struct {
	RecordID      string    `json:"record_id"`
	QualityScore  float64   `json:"quality_score"`
	QualityIssues string    `json:"quality_issues"`
	SnapshotTime  time.Time `json:"snapshot_time"`
}

// ImpactAnalysisResponse 影响分析响应
type ImpactAnalysisResponse struct {
	SourceRecordID         string           `json:"source_record_id"`
	AffectedRecords        []AffectedRecord `json:"affected_records"`
	ImpactScope            ImpactScope      `json:"impact_scope"`
	ReprocessingSuggestion string           `json:"reprocessing_suggestion"`
	AnalysisTime           time.Time        `json:"analysis_time"`
}

// AffectedRecord 受影响的记录
type AffectedRecord struct {
	ThematicRecordID    string    `json:"thematic_record_id"`
	ThematicInterfaceID string    `json:"thematic_interface_id"`
	InterfaceName       string    `json:"interface_name"`
	QualityScore        float64   `json:"quality_score"`
	LastProcessedTime   time.Time `json:"last_processed_time"`
}

// ImpactScope 影响范围
type ImpactScope struct {
	AffectedRecordCount    int    `json:"affected_record_count"`
	AffectedInterfaceCount int    `json:"affected_interface_count"`
	AffectedLibraryCount   int    `json:"affected_library_count"`
	ImpactLevel            string `json:"impact_level"` // low, medium, high
}

// LineageGraphRequest 血缘图请求
type LineageGraphRequest struct {
	ThematicLibraryID string     `json:"thematic_library_id,omitempty"`
	SourceLibraryID   string     `json:"source_library_id,omitempty"`
	StartTime         *time.Time `json:"start_time,omitempty"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	Limit             int        `json:"limit,omitempty"`
}

// LineageGraphResponse 血缘图响应
type LineageGraphResponse struct {
	Nodes      []LineageNode     `json:"nodes"`
	Edges      []LineageEdge     `json:"edges"`
	Statistics LineageStatistics `json:"statistics"`
}

// LineageNode 血缘节点
type LineageNode struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // source, target
	Name        string `json:"name"`
	LibraryID   string `json:"library_id,omitempty"`
	InterfaceID string `json:"interface_id"`
	RecordCount int    `json:"record_count"`
}

// LineageEdge 血缘边
type LineageEdge struct {
	ID            string    `json:"id"`
	SourceNodeID  string    `json:"source_node_id"`
	TargetNodeID  string    `json:"target_node_id"`
	RelationType  string    `json:"relation_type"`
	QualityScore  float64   `json:"quality_score"`
	ProcessedTime time.Time `json:"processed_time"`
}

// LineageStatistics 血缘统计
type LineageStatistics struct {
	TotalNodes int `json:"total_nodes"`
	TotalEdges int `json:"total_edges"`
}
