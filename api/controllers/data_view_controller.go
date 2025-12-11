/*
 * @module api/controllers/data_view_controller
 * @description 数据查看控制器，提供基础库和主题库的数据接口查看功能
 * @architecture MVC架构 - 控制器层
 * @documentReference datahub-service/ai_docs/data_view.md
 * @stateFlow 请求验证 -> 服务调用 -> 响应返回
 * @rules 支持基础库和主题库的通用查看，使用pgmeta客户端获取表信息
 * @dependencies chi, render, client, database
 * @refs basic_library_controller.go, thematic_library_controller.go
 */

package controllers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"gorm.io/gorm"

	"datahub-service/service"
	"datahub-service/service/database"
	"datahub-service/service/models"
)

// DataViewController 数据查看控制器
type DataViewController struct {
	db            *gorm.DB
	schemaService *database.SchemaService
}

// NewDataViewController 创建数据查看控制器实例
func NewDataViewController(db *gorm.DB) *DataViewController {
	schemaService := service.GlobalSchemaService

	return &DataViewController{
		db:            db,
		schemaService: schemaService,
	}
}

// LibraryTablesResponse 库表列表响应
type LibraryTablesResponse struct {
	LibraryID   string      `json:"library_id"`
	LibraryType string      `json:"library_type"` // basic_library, thematic_library
	LibraryName string      `json:"library_name"`
	SchemaName  string      `json:"schema_name"`
	Tables      []TableInfo `json:"tables"`
	TotalCount  int         `json:"total_count"`
}

// TableInfo 表信息
type TableInfo struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	Schema           string         `json:"schema"`
	Comment          *string        `json:"comment"`
	Size             string         `json:"size"`
	LiveRowsEstimate int            `json:"live_rows_estimate"`
	Columns          []ColumnInfo   `json:"columns,omitempty"`
	PrimaryKeys      []PrimaryKey   `json:"primary_keys,omitempty"`
	Relationships    []Relationship `json:"relationships,omitempty"`
}

// ColumnInfo 列信息
type ColumnInfo struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	DataType           string      `json:"data_type"`
	Format             string      `json:"format"`
	IsNullable         bool        `json:"is_nullable"`
	DefaultValue       interface{} `json:"default_value"`
	IsIdentity         bool        `json:"is_identity"`
	IdentityGeneration *string     `json:"identity_generation"`
	IsGenerated        bool        `json:"is_generated"`
	IsUpdatable        bool        `json:"is_updatable"`
	Comment            *string     `json:"comment"`
}

// PrimaryKey 主键信息
type PrimaryKey struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// Relationship 关系信息
type Relationship struct {
	ID               string `json:"id"`
	ConstraintName   string `json:"constraint_name"`
	SourceTableName  string `json:"source_table_name"`
	SourceColumnName string `json:"source_column_name"`
	TargetTableName  string `json:"target_table_name"`
	TargetColumnName string `json:"target_column_name"`
}

// GetLibraryTables 获取库的所有数据接口(表)
// @Summary 获取库的所有数据接口
// @Description 获取指定基础库或主题库的所有数据接口(表)信息
// @Tags 数据查看
// @Accept json
// @Produce json
// @Param library_type path string true "库类型" Enums(basic_library,thematic_library)
// @Param library_id path string true "库ID" format(uuid)
// @Param include_columns query bool false "是否包含列信息" default(false)
// @Param include_relationships query bool false "是否包含关系信息" default(false)
// @Success 200 {object} APIResponse{data=LibraryTablesResponse}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /data-view/{library_type}/{library_id}/tables [get]
func (c *DataViewController) GetLibraryTables(w http.ResponseWriter, r *http.Request) {
	libraryType := chi.URLParam(r, "library_type")
	libraryID := chi.URLParam(r, "library_id")

	slog.Debug("GetLibraryTables - 请求参数",
		"library_type", libraryType,
		"library_id", libraryID)

	// 验证库类型
	if libraryType != "basic_library" && libraryType != "thematic_library" {
		slog.Error("GetLibraryTables - 无效的库类型", "library_type", libraryType)
		render.JSON(w, r, BadRequestResponse("无效的库类型，支持: basic_library, thematic_library", nil))
		return
	}

	// 解析查询参数
	includeColumns := r.URL.Query().Get("include_columns") == "true"
	includeRelationships := r.URL.Query().Get("include_relationships") == "true"

	slog.Debug("GetLibraryTables - 查询参数",
		"include_columns", includeColumns,
		"include_relationships", includeRelationships)

	// 获取库信息和对应的schema名称
	libraryInfo, err := c.getLibraryInfo(libraryType, libraryID)
	if err != nil {
		slog.Error("GetLibraryTables - 获取库信息失败", "error", err)
		render.JSON(w, r, NotFoundResponse("库不存在: "+err.Error(), err))
		return
	}

	slog.Debug("GetLibraryTables - 库信息",
		"id", libraryInfo.ID,
		"name", libraryInfo.Name,
		"schema_name", libraryInfo.SchemaName)

	// 获取schema下的所有表名
	tableNames, err := c.schemaService.ListTables(libraryInfo.SchemaName)
	if err != nil {
		slog.Error("GetLibraryTables - 获取表信息失败",
			"schema", libraryInfo.SchemaName,
			"error", err)
		render.JSON(w, r, InternalErrorResponse("获取表信息失败: "+err.Error(), err))
		return
	}

	slog.Debug("GetLibraryTables - 获取到表数量",
		"count", len(tableNames),
		"table_names", tableNames)

	// 转换数据格式
	tableInfos := make([]TableInfo, 0, len(tableNames))
	for i, tableName := range tableNames {
		tableInfo := TableInfo{
			ID:               i + 1, // 使用索引作为ID
			Name:             tableName,
			Schema:           libraryInfo.SchemaName,
			Comment:          nil,
			Size:             "",
			LiveRowsEstimate: 0,
		}

		// 如果需要详细信息，获取表结构
		if includeColumns {
			structure, err := c.schemaService.GetTableInfo(libraryInfo.SchemaName, tableName)
			if err == nil && structure != nil {
				// 转换列信息
				columns := make([]ColumnInfo, 0, len(structure.Columns))
				for j, col := range structure.Columns {
					columns = append(columns, ColumnInfo{
						ID:           fmt.Sprintf("%d", j+1), // 使用索引作为ID
						Name:         col.Name,
						DataType:     col.DataType,
						Format:       "",
						IsNullable:   col.IsNullable,
						DefaultValue: col.DefaultValue,
						IsIdentity:   false,
						IsGenerated:  false,
						IsUpdatable:  true,
						Comment:      &col.Comment,
					})
				}
				tableInfo.Columns = columns
				if structure.Comment != "" {
					tableInfo.Comment = &structure.Comment
				}
			}
		}

		// 暂时不支持关系信息，因为SchemaService没有提供相关方法
		if includeRelationships {
			tableInfo.PrimaryKeys = []PrimaryKey{}
			tableInfo.Relationships = []Relationship{}
		}

		tableInfos = append(tableInfos, tableInfo)
	}

	response := LibraryTablesResponse{
		LibraryID:   libraryID,
		LibraryType: libraryType,
		LibraryName: libraryInfo.Name,
		SchemaName:  libraryInfo.SchemaName,
		Tables:      tableInfos,
		TotalCount:  len(tableInfos),
	}

	render.JSON(w, r, SuccessResponse("获取库表信息成功", response))
}

// GetTableData 获取表数据
// @Summary 获取表数据
// @Description 获取指定表的数据内容
// @Tags 数据查看
// @Accept json
// @Produce json
// @Param library_type path string true "库类型" Enums(basic_library,thematic_library)
// @Param library_id path string true "库ID" format(uuid)
// @Param table_name path string true "表名"
// @Param limit query int false "限制返回行数" default(100) minimum(1) maximum(1000)
// @Param offset query int false "偏移量" default(0) minimum(0)
// @Param where query string false "WHERE条件(不包含WHERE关键字，由前端拼好并转义)" example("age > 18 AND status = 'active'")
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /data-view/{library_type}/{library_id}/tables/{table_name}/data [get]
func (c *DataViewController) GetTableData(w http.ResponseWriter, r *http.Request) {
	libraryType := chi.URLParam(r, "library_type")
	libraryID := chi.URLParam(r, "library_id")
	tableName := chi.URLParam(r, "table_name")

	// 验证库类型
	if libraryType != "basic_library" && libraryType != "thematic_library" {
		render.JSON(w, r, BadRequestResponse("无效的库类型", nil))
		return
	}

	// 解析查询参数
	limit := 100
	offset := 0
	whereCondition := strings.TrimSpace(r.URL.Query().Get("where"))

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	slog.Debug("GetTableData - 请求参数",
		"library_type", libraryType,
		"library_id", libraryID,
		"table_name", tableName,
		"limit", limit,
		"offset", offset,
		"where", whereCondition)

	// 获取库信息
	libraryInfo, err := c.getLibraryInfo(libraryType, libraryID)
	if err != nil {
		render.JSON(w, r, NotFoundResponse("库不存在: "+err.Error(), err))
		return
	}

	// 使用schema服务获取表数据
	fullTableName := libraryInfo.SchemaName + "." + tableName
	data, totalCount, err := c.schemaService.GetTableData(fullTableName, limit, offset, whereCondition)
	if err != nil {
		slog.Error("GetTableData - 获取表数据失败",
			"table", fullTableName,
			"where", whereCondition,
			"error", err)
		render.JSON(w, r, InternalErrorResponse("获取表数据失败: "+err.Error(), err))
		return
	}

	response := map[string]interface{}{
		"library_id":      libraryID,
		"library_type":    libraryType,
		"library_name":    libraryInfo.Name,
		"schema_name":     libraryInfo.SchemaName,
		"table_name":      tableName,
		"data":            data,
		"total_count":     totalCount,
		"limit":           limit,
		"offset":          offset,
		"where_condition": whereCondition,
	}

	render.JSON(w, r, SuccessResponse("获取表数据成功", response))
}

// GetTableStructure 获取表结构
// @Summary 获取表结构
// @Description 获取指定表的结构信息
// @Tags 数据查看
// @Accept json
// @Produce json
// @Param library_type path string true "库类型" Enums(basic_library,thematic_library)
// @Param library_id path string true "库ID" format(uuid)
// @Param table_name path string true "表名"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /data-view/{library_type}/{library_id}/tables/{table_name}/structure [get]
func (c *DataViewController) GetTableStructure(w http.ResponseWriter, r *http.Request) {
	libraryType := chi.URLParam(r, "library_type")
	libraryID := chi.URLParam(r, "library_id")
	tableName := chi.URLParam(r, "table_name")

	// 验证库类型
	if libraryType != "basic_library" && libraryType != "thematic_library" {
		render.JSON(w, r, BadRequestResponse("无效的库类型", nil))
		return
	}

	// 获取库信息
	libraryInfo, err := c.getLibraryInfo(libraryType, libraryID)
	if err != nil {
		render.JSON(w, r, NotFoundResponse("库不存在: "+err.Error(), err))
		return
	}

	// 获取表结构
	structure, err := c.schemaService.GetTableInfo(libraryInfo.SchemaName, tableName)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取表结构失败: "+err.Error(), err))
		return
	}

	response := map[string]interface{}{
		"library_id":   libraryID,
		"library_type": libraryType,
		"library_name": libraryInfo.Name,
		"schema_name":  libraryInfo.SchemaName,
		"table_name":   tableName,
		"structure":    structure,
	}

	render.JSON(w, r, SuccessResponse("获取表结构成功", response))
}

// GetRecordByPrimaryKey 根据主键值获取单条记录
// @Summary 根据主键值获取单条记录
// @Description 根据schema、table和主键值查询单条记录（用于查看质量问题的原始数据）
// @Tags 数据查看
// @Accept json
// @Produce json
// @Param schema_name query string true "Schema名称"
// @Param table_name query string true "表名"
// @Param record_identifier query string true "记录标识符" example("id=123" or "key1=val1&key2=val2")
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /data-view/record-by-pk [get]
func (c *DataViewController) GetRecordByPrimaryKey(w http.ResponseWriter, r *http.Request) {
	schemaName := r.URL.Query().Get("schema_name")
	tableName := r.URL.Query().Get("table_name")
	recordIdentifier := r.URL.Query().Get("record_identifier")

	// 验证参数
	if schemaName == "" || tableName == "" || recordIdentifier == "" {
		render.JSON(w, r, BadRequestResponse("缺少必要参数: schema_name, table_name, record_identifier", nil))
		return
	}

	slog.Debug("GetRecordByPrimaryKey - 请求参数",
		"schema", schemaName,
		"table", tableName,
		"identifier", recordIdentifier)

	// 解析 recordIdentifier (格式: key1=value1&key2=value2 或 row_123)
	whereConditions, whereValues, err := c.parseRecordIdentifier(recordIdentifier, schemaName, tableName)
	if err != nil {
		render.JSON(w, r, BadRequestResponse("无效的记录标识符: "+err.Error(), err))
		return
	}

	// 构建查询SQL
	fullTableName := fmt.Sprintf("%s.%s", schemaName, tableName)

	// 查询记录
	var record map[string]interface{}
	result := c.db.Table(fullTableName).Where(whereConditions, whereValues...).Take(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			render.JSON(w, r, NotFoundResponse("记录不存在", result.Error))
			return
		}
		slog.Error("GetRecordByPrimaryKey - 查询记录失败", "error", result.Error)
		render.JSON(w, r, InternalErrorResponse("查询记录失败: "+result.Error.Error(), result.Error))
		return
	}

	response := map[string]interface{}{
		"schema_name":       schemaName,
		"table_name":        tableName,
		"record_identifier": recordIdentifier,
		"record":            record,
	}

	render.JSON(w, r, SuccessResponse("获取记录成功", response))
}

// parseRecordIdentifier 解析记录标识符
// 支持格式: "id=123" 或 "key1=val1&key2=val2" 或 "row_123"
func (c *DataViewController) parseRecordIdentifier(identifier, schemaName, tableName string) (string, []interface{}, error) {
	// 特殊处理: 如果是 row_N 格式，说明表没有主键，无法查询
	if strings.HasPrefix(identifier, "row_") {
		return "", nil, fmt.Errorf("表没有主键，无法通过行号查询原始数据")
	}

	// 解析 key=value 格式
	pairs := strings.Split(identifier, "&")
	if len(pairs) == 0 {
		return "", nil, fmt.Errorf("记录标识符格式错误")
	}

	var whereParts []string
	var whereValues []interface{}

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("记录标识符格式错误: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 处理 NULL 值
		if value == "NULL" {
			whereParts = append(whereParts, fmt.Sprintf("%s IS NULL", key))
		} else {
			whereParts = append(whereParts, fmt.Sprintf("%s = ?", key))
			whereValues = append(whereValues, value)
		}
	}

	whereConditions := strings.Join(whereParts, " AND ")
	return whereConditions, whereValues, nil
}

// LibraryInfo 库信息
type LibraryInfo struct {
	ID         string
	Name       string
	SchemaName string
}

// getLibraryInfo 获取库信息
func (c *DataViewController) getLibraryInfo(libraryType, libraryID string) (*LibraryInfo, error) {
	// 这里需要根据库类型查询对应的库信息
	// 基础库使用 name_en 作为 schema 名称
	// 主题库使用 name_en 作为 schema 名称

	switch libraryType {
	case "basic_library":
		return c.getBasicLibraryInfo(libraryID)
	case "thematic_library":
		return c.getThematicLibraryInfo(libraryID)
	default:
		return nil, fmt.Errorf("不支持的库类型: %s", libraryType)
	}
}

// getBasicLibraryInfo 获取基础库信息
func (c *DataViewController) getBasicLibraryInfo(libraryID string) (*LibraryInfo, error) {
	slog.Debug("getBasicLibraryInfo - 查询基础库", "id", libraryID)

	var basicLibrary models.BasicLibrary
	err := c.db.First(&basicLibrary, "id = ?", libraryID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Error("getBasicLibraryInfo - 基础库不存在", "id", libraryID)
			return nil, fmt.Errorf("基础库不存在: %s", libraryID)
		}
		slog.Error("getBasicLibraryInfo - 查询基础库失败", "error", err)
		return nil, fmt.Errorf("查询基础库失败: %v", err)
	}

	slog.Debug("getBasicLibraryInfo - 找到基础库",
		"id", basicLibrary.ID,
		"name_zh", basicLibrary.NameZh,
		"name_en", basicLibrary.NameEn)

	return &LibraryInfo{
		ID:         basicLibrary.ID,
		Name:       basicLibrary.NameZh,
		SchemaName: basicLibrary.NameEn, // 使用英文名作为schema名称
	}, nil
}

// getThematicLibraryInfo 获取主题库信息
func (c *DataViewController) getThematicLibraryInfo(libraryID string) (*LibraryInfo, error) {
	slog.Debug("getThematicLibraryInfo - 查询主题库", "id", libraryID)

	var thematicLibrary models.ThematicLibrary
	err := c.db.First(&thematicLibrary, "id = ?", libraryID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Error("getThematicLibraryInfo - 主题库不存在", "id", libraryID)
			return nil, fmt.Errorf("主题库不存在: %s", libraryID)
		}
		slog.Error("getThematicLibraryInfo - 查询主题库失败", "error", err)
		return nil, fmt.Errorf("查询主题库失败: %v", err)
	}

	slog.Debug("getThematicLibraryInfo - 找到主题库",
		"id", thematicLibrary.ID,
		"name_zh", thematicLibrary.NameZh,
		"name_en", thematicLibrary.NameEn)

	return &LibraryInfo{
		ID:         thematicLibrary.ID,
		Name:       thematicLibrary.NameZh,
		SchemaName: thematicLibrary.NameEn, // 使用英文名作为schema名称
	}, nil
}
