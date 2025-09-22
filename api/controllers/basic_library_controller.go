/*
 * @module api/controllers/basic_library_controller
 * @description 数据基础库控制器，处理数据源测试和接口调用测试
 * @architecture MVC架构 - 控制器层
 * @documentReference dev_docs/requirements.md, ai_docs/interfaces.md
 * @stateFlow HTTP请求处理流程，数据源连接测试流程
 * @rules 统一的错误处理和响应格式，参数验证
 * @dependencies datahub-service/service, github.com/go-chi/render
 * @refs dev_docs/model.md, ai_docs/interfaces.md
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/basic_library"
	"datahub-service/service/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// BasicLibraryController 数据基础库控制器
type BasicLibraryController struct {
	service *basic_library.Service
}

// NewBasicLibraryController 创建数据基础库控制器实例
func NewBasicLibraryController() *BasicLibraryController {
	return &BasicLibraryController{
		service: service.GlobalBasicLibraryService,
	}
}

// DataSourceTestRequest 数据源测试请求结构
type DataSourceTestRequest struct {
	DataSourceID string                 `json:"data_source_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	TestType     string                 `json:"test_type" validate:"required" example:"connection"` // connection, data_preview
	Config       map[string]interface{} `json:"config,omitempty"`                                   // 额外测试配置
}

// DataSourceTestResponse 数据源测试响应结构
type DataSourceTestResponse struct {
	Success     bool                   `json:"success" example:"true"`
	Message     string                 `json:"message" example:"连接成功"`
	Duration    int64                  `json:"duration" example:"250"` // 测试耗时（毫秒）
	TestType    string                 `json:"test_type" example:"connection"`
	Data        interface{}            `json:"data,omitempty"`                                 // 预览数据
	Metadata    map[string]interface{} `json:"metadata,omitempty"`                             // 元数据信息
	Error       string                 `json:"error,omitempty" example:""`                     // 错误信息
	Suggestions []string               `json:"suggestions,omitempty" example:"检查网络连接,验证数据库权限"` // 优化建议
}

// InterfaceTestRequest 接口调用测试请求结构
type InterfaceTestRequest struct {
	InterfaceID string                 `json:"interface_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	TestType    string                 `json:"test_type" validate:"required" example:"data_fetch"` // data_fetch, performance, validation
	Parameters  map[string]interface{} `json:"parameters,omitempty"`                               // 测试参数
	Options     map[string]interface{} `json:"options,omitempty"`                                  // 测试选项
}

// InterfaceTestResponse 接口调用测试响应结构
type InterfaceTestResponse struct {
	Success     bool                   `json:"success" example:"true"`
	Message     string                 `json:"message" example:"接口调用成功"`
	Duration    int64                  `json:"duration" example:"150"` // 调用耗时（毫秒）
	TestType    string                 `json:"test_type" example:"data_fetch"`
	Data        interface{}            `json:"data,omitempty"`                            // 返回数据
	RowCount    int                    `json:"row_count,omitempty" example:"25"`          // 数据行数
	ColumnCount int                    `json:"column_count,omitempty" example:"8"`        // 字段数
	DataTypes   map[string]string      `json:"data_types,omitempty"`                      // 字段类型
	Performance map[string]interface{} `json:"performance,omitempty"`                     // 性能指标
	Validation  map[string]interface{} `json:"validation,omitempty"`                      // 数据验证结果
	Error       string                 `json:"error,omitempty" example:""`                // 错误信息
	Warnings    []string               `json:"warnings,omitempty" example:"数据量较大，建议分页查询"` // 警告信息
}

// UpdateBasicLibraryRequest 修改数据基础库请求结构
type UpdateBasicLibraryRequest struct {
	ID          string `json:"id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	NameZh      string `json:"name_zh,omitempty" example:"用户数据库"`
	NameEn      string `json:"name_en,omitempty" example:"user_database"`
	Description string `json:"description,omitempty" example:"存储用户相关数据的基础库"`
	Status      string `json:"status,omitempty" example:"active"`
}

// UpdateDataSourceRequest 修改数据源请求结构
type UpdateDataSourceRequest struct {
	ID               string                 `json:"id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string                 `json:"name,omitempty" example:"用户数据源"`
	Category         string                 `json:"category,omitempty" example:"db"`
	Type             string                 `json:"type,omitempty" example:"mysql"`
	ConnectionConfig map[string]interface{} `json:"connection_config,omitempty"`
	ParamsConfig     map[string]interface{} `json:"params_config,omitempty"`
	Script           string                 `json:"script,omitempty"`
	ScriptEnabled    bool                   `json:"script_enabled,omitempty"`
}

// UpdateDataInterfaceRequest 修改数据接口请求结构
type UpdateDataInterfaceRequest struct {
	ID                string                 `json:"id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	NameZh            string                 `json:"name_zh,omitempty" example:"用户接口"`
	NameEn            string                 `json:"name_en,omitempty" example:"user_interface"`
	Type              string                 `json:"type,omitempty" example:"realtime"` // realtime, batch
	Description       string                 `json:"description,omitempty" example:"用户数据查询接口"`
	Status            string                 `json:"status,omitempty" example:"active"`
	DataSourceID      string                 `json:"data_source_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	InterfaceConfig   map[string]interface{} `json:"interface_config,omitempty"`
	ParseConfig       map[string]interface{} `json:"parse_config,omitempty"`
	TableFieldsConfig map[string]interface{} `json:"table_fields_config,omitempty"`
}

// UpdateInterfaceFieldsRequest 更新接口字段配置请求结构
type UpdateInterfaceFieldsRequest struct {
	InterfaceID string              `json:"interface_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Fields      []models.TableField `json:"fields" validate:"required"`
	UpdateTable bool                `json:"update_table" example:"true"`  // 是否同时更新数据库表结构
	BackupTable bool                `json:"backup_table" example:"false"` // 是否备份原表
}

// BasicLibraryListResponse 数据基础库列表响应结构
type BasicLibraryListResponse struct {
	List  []models.BasicLibrary `json:"list"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// DataSourceListResponse 数据源列表响应结构
type DataSourceListResponse struct {
	List  []models.DataSource `json:"list"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// DataInterfaceListResponse 数据接口列表响应结构
type DataInterfaceListResponse struct {
	List  []models.DataInterface `json:"list"`
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Size  int                    `json:"size"`
}

// @Summary 添加数据基础库
// @Description 添加数据基础库
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.BasicLibrary true "数据基础库请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/add-basic-library [post]
func (c *BasicLibraryController) AddBasicLibrary(w http.ResponseWriter, r *http.Request) {
	var req models.BasicLibrary
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	err := c.service.CreateBasicLibrary(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("添加数据基础库失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("添加数据基础库成功", nil))
}

// @Summary 删除数据基础库
// @Description 删除数据基础库
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据基础库ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/{id} [delete]
func (c *BasicLibraryController) DeleteBasicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("数据基础库ID不能为空", nil))
		return
	}

	// 先根据ID查询基础库信息
	library, err := c.service.GetBasicLibrary(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("查询数据基础库失败", err))
		return
	}

	// 调用删除方法
	err = c.service.DeleteBasicLibrary(library)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据基础库失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据基础库成功", nil))
}

// @Summary 修改数据基础库
// @Description 修改数据基础库信息
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body UpdateBasicLibraryRequest true "修改数据基础库请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/update-basic-library [post]
func (c *BasicLibraryController) UpdateBasicLibrary(w http.ResponseWriter, r *http.Request) {
	var req UpdateBasicLibraryRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if req.ID == "" {
		render.JSON(w, r, BadRequestResponse("基础库ID不能为空", nil))
		return
	}

	// 构建更新字段map
	updates := make(map[string]interface{})
	if req.NameZh != "" {
		updates["name_zh"] = req.NameZh
	}
	if req.NameEn != "" {
		updates["name_en"] = req.NameEn
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if len(updates) == 0 {
		render.JSON(w, r, BadRequestResponse("没有要更新的字段", nil))
		return
	}

	err := c.service.UpdateBasicLibrary(req.ID, updates)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("修改数据基础库失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("修改数据基础库成功", nil))
}

// @Summary 添加数据源
// @Description 添加数据源
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.DataSource true "数据源请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/add-datasource [post]
func (c *BasicLibraryController) AddDataSource(w http.ResponseWriter, r *http.Request) {
	var req models.DataSource
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	err := c.service.CreateDataSource(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("添加数据源失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("添加数据源成功", nil))
}

// @Summary 修改数据源
// @Description 修改数据源信息
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body UpdateDataSourceRequest true "修改数据源请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/update-datasource [post]
func (c *BasicLibraryController) UpdateDataSource(w http.ResponseWriter, r *http.Request) {
	var req UpdateDataSourceRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if req.ID == "" {
		render.JSON(w, r, BadRequestResponse("数据源ID不能为空", nil))
		return
	}

	// 构建更新字段map
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.ConnectionConfig != nil {
		updates["connection_config"] = req.ConnectionConfig
	}
	if req.ParamsConfig != nil {
		updates["params_config"] = req.ParamsConfig
	}
	if req.Script != "" {
		updates["script"] = req.Script
	}
	updates["script_enabled"] = req.ScriptEnabled

	if len(updates) == 0 {
		render.JSON(w, r, BadRequestResponse("没有要更新的字段", nil))
		return
	}

	err := c.service.UpdateDataSource(req.ID, updates)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("修改数据源失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("修改数据源成功", nil))
}

// @Summary 删除数据源
// @Description 删除数据源
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据源ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/datasources/{id} [delete]
func (c *BasicLibraryController) DeleteDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("数据源ID不能为空", nil))
		return
	}

	// 先根据ID查询数据源信息
	dataSource, err := c.service.GetDataSource(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("查询数据源失败", err))
		return
	}

	// 调用删除方法
	err = c.service.DeleteDataSource(dataSource)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据源失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据源成功", nil))
}

// @Summary 添加数据接口
// @Description 添加数据接口
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.DataInterface true "数据接口请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/add-interface [post]
func (c *BasicLibraryController) AddInterface(w http.ResponseWriter, r *http.Request) {
	var req models.DataInterface
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	err := c.service.CreateDataInterface(&req)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("添加数据接口失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("添加数据接口成功", nil))
}

// @Summary 修改数据接口
// @Description 修改数据接口信息
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body UpdateDataInterfaceRequest true "修改数据接口请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/update-interface [post]
func (c *BasicLibraryController) UpdateInterface(w http.ResponseWriter, r *http.Request) {
	var req UpdateDataInterfaceRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if req.ID == "" {
		render.JSON(w, r, BadRequestResponse("数据接口ID不能为空", nil))
		return
	}

	// 构建更新字段map
	updates := make(map[string]interface{})
	if req.NameZh != "" {
		updates["name_zh"] = req.NameZh
	}
	if req.NameEn != "" {
		updates["name_en"] = req.NameEn
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.DataSourceID != "" {
		updates["data_source_id"] = req.DataSourceID
	}
	if req.InterfaceConfig != nil {
		updates["interface_config"] = req.InterfaceConfig
	}
	if req.ParseConfig != nil {
		updates["parse_config"] = req.ParseConfig
	}
	if req.TableFieldsConfig != nil {
		updates["table_fields_config"] = req.TableFieldsConfig
	}

	if len(updates) == 0 {
		render.JSON(w, r, BadRequestResponse("没有要更新的字段", nil))
		return
	}

	err := c.service.UpdateDataInterface(req.ID, updates)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("修改数据接口失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("修改数据接口成功", nil))
}

// @Summary 删除数据接口
// @Description 删除数据接口
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据接口ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/interfaces/{id} [delete]
func (c *BasicLibraryController) DeleteInterface(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("数据接口ID不能为空", nil))
		return
	}

	// 先根据ID查询数据接口信息
	dataInterface, err := c.service.GetDataInterface(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("查询数据接口失败", err))
		return
	}

	// 调用删除方法
	err = c.service.DeleteDataInterface(dataInterface)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("删除数据接口失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据接口成功", nil))
}

// TestDataSource 测试数据源连接
// @Summary 测试数据源连接
// @Description 测试数据源连接和数据获取能力
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body DataSourceTestRequest true "测试请求"
// @Success 200 {object} APIResponse{data=DataSourceTestResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/test-datasource [post]
func (c *BasicLibraryController) TestDataSource(w http.ResponseWriter, r *http.Request) {
	var req DataSourceTestRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.service.TestDataSource(req.DataSourceID, req.TestType, req.Config)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据源测试失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("测试完成", result))
}

// TestInterface 测试接口调用
// @Summary 测试接口调用
// @Description 测试数据接口的调用和数据获取能力
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body InterfaceTestRequest true "测试请求"
// @Success 200 {object} APIResponse{data=InterfaceTestResponse}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/test-interface [post]
func (c *BasicLibraryController) TestInterface(w http.ResponseWriter, r *http.Request) {
	var req InterfaceTestRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	result, err := c.service.TestInterface(req.InterfaceID, req.TestType, req.Parameters, req.Options)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("接口测试失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("测试完成", result))
}

// GetDataSourceStatus 获取数据源状态
// @Summary 获取数据源运行状态
// @Description 获取数据源的连接状态、最近同步时间等信息
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据源ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/datasource-status/{id} [get]
func (c *BasicLibraryController) GetDataSourceStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("数据源ID参数不能为空", nil))
		return
	}

	status, err := c.service.GetDataSourceStatus(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据源状态失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取状态成功", status))
}

// GetBasicLibraryList 获取数据基础库列表
// @Summary 获取数据基础库列表
// @Description 分页获取数据基础库列表，支持多种过滤条件
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param name query string false "名称搜索（支持中英文）"
// @Param status query string false "状态过滤" Enums(active,inactive)
// @Param created_by query string false "创建者过滤"
// @Success 200 {object} APIResponse{data=BasicLibraryListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /basic-libraries [get]
func (c *BasicLibraryController) GetBasicLibraryList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	name := r.URL.Query().Get("name")
	status := r.URL.Query().Get("status")
	createdBy := r.URL.Query().Get("created_by")

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	libraries, total, err := c.service.GetBasicLibraryList(page, size, name, status, createdBy)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据基础库列表失败", err))
		return
	}

	// 构建响应
	response := BasicLibraryListResponse{
		List:  libraries,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据基础库列表成功", response))
}

// GetDataSourceList 获取数据源列表
// @Summary 获取数据源列表
// @Description 分页获取数据源列表，支持多种过滤条件
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param library_id query string false "基础库ID过滤"
// @Param category query string false "数据源分类过滤（如：stream, http, db, file等）"
// @Param type query string false "数据源类型过滤（如：mysql, postgresql, http等）"
// @Param status query string false "状态过滤" Enums(active,inactive)
// @Param name query string false "名称搜索"
// @Success 200 {object} APIResponse{data=DataSourceListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /basic-libraries/datasources [get]
func (c *BasicLibraryController) GetDataSourceList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	libraryID := r.URL.Query().Get("library_id")
	category := r.URL.Query().Get("category")
	source_type := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	name := r.URL.Query().Get("name")

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	dataSources, total, err := c.service.GetDataSourceList(page, size, libraryID, category, source_type, status, name)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据源列表失败", err))
		return
	}

	// 构建响应
	response := DataSourceListResponse{
		List:  dataSources,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据源列表成功", response))
}

// GetDataInterfaceList 获取数据接口列表
// @Summary 获取数据接口列表
// @Description 分页获取数据接口列表，支持多种过滤条件
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param library_id query string false "基础库ID过滤"
// @Param data_source_id query string false "数据源ID过滤"
// @Param interface_type query string false "接口类型过滤（如：realtime, batch）"
// @Param status query string false "状态过滤" Enums(active,inactive)
// @Param name query string false "名称搜索"
// @Success 200 {object} APIResponse{data=DataInterfaceListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /basic-libraries/interfaces [get]
func (c *BasicLibraryController) GetDataInterfaceList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	libraryID := r.URL.Query().Get("library_id")
	dataSourceID := r.URL.Query().Get("data_source_id")
	interfaceType := r.URL.Query().Get("interface_type")
	status := r.URL.Query().Get("status")
	name := r.URL.Query().Get("name")

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	interfaces, total, err := c.service.GetDataInterfaceList(page, size, libraryID, dataSourceID, interfaceType, status, name)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据接口列表失败", err))
		return
	}

	// 构建响应
	response := DataInterfaceListResponse{
		List:  interfaces,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据接口列表成功", response))
}

// PreviewInterfaceData 预览接口数据
// @Summary 预览接口数据
// @Description 获取接口的样例数据用于预览
// @Tags 数据基础库
// @Produce json
// @Param id path string true "接口ID"
// @Param limit query int false "数据条数" default(10)
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/interface-preview/{id} [get]
func (c *BasicLibraryController) PreviewInterfaceData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("接口ID参数不能为空", nil))
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	data, err := c.service.PreviewInterfaceData(id, limit)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("数据预览失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("数据预览成功", data))
}

// UpdateInterfaceFields 更新接口字段配置
// @Summary 更新接口字段配置
// @Description 更新数据接口的字段配置，并可选择同时更新数据库表结构
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body UpdateInterfaceFieldsRequest true "更新字段配置请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/update-interface-fields [post]
func (c *BasicLibraryController) UpdateInterfaceFields(w http.ResponseWriter, r *http.Request) {
	var req UpdateInterfaceFieldsRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if req.InterfaceID == "" {
		render.JSON(w, r, BadRequestResponse("接口ID不能为空", nil))
		return
	}

	if len(req.Fields) == 0 {
		render.JSON(w, r, BadRequestResponse("字段配置不能为空", nil))
		return
	}

	// 验证字段配置的基本规则
	primaryKeyCount := 0
	for _, field := range req.Fields {
		if field.IsPrimaryKey {
			primaryKeyCount++
		}
		if field.NameEn == "" {
			render.JSON(w, r, BadRequestResponse("字段英文名不能为空", nil))
			return
		}
	}

	if primaryKeyCount == 0 {
		render.JSON(w, r, BadRequestResponse("至少需要一个主键字段", nil))
		return
	}

	if primaryKeyCount > 1 {
		render.JSON(w, r, BadRequestResponse("目前只支持单一主键", nil))
		return
	}

	err := c.service.UpdateInterfaceFields(req.InterfaceID, req.Fields, req.UpdateTable)
	if err != nil {
		// 根据错误类型提供更具体的错误信息
		if strings.Contains(err.Error(), "primary key") {
			render.JSON(w, r, BadRequestResponse("主键字段配置错误：主键字段不能为空且必须唯一", err))
			return
		}
		if strings.Contains(err.Error(), "创建表结构失败") {
			render.JSON(w, r, InternalErrorResponse("创建数据库表结构失败", err))
			return
		}
		if strings.Contains(err.Error(), "更新表结构失败") {
			render.JSON(w, r, InternalErrorResponse("更新数据库表结构失败，但字段配置已保存", err))
			return
		}
		render.JSON(w, r, InternalErrorResponse("更新接口字段配置失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新接口字段配置成功", nil))
}

// GetDataSourceManagerStats 获取数据源管理器统计信息
// @Summary 获取数据源管理器统计信息
// @Description 获取数据源管理器的运行统计信息，包括总数、类型分布、在线状态等
// @Tags 数据基础库
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/datasource-manager-stats [get]
func (c *BasicLibraryController) GetDataSourceManagerStats(w http.ResponseWriter, r *http.Request) {
	datasourceInitService := c.service.GetDatasourceInitService()
	stats := datasourceInitService.GetManagerStatistics()

	render.JSON(w, r, SuccessResponse("获取统计信息成功", stats))
}

// GetResidentDataSources 获取所有常驻数据源状态
// @Summary 获取常驻数据源状态
// @Description 获取所有常驻数据源的运行状态和统计信息
// @Tags 数据基础库
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/resident-datasources [get]
func (c *BasicLibraryController) GetResidentDataSources(w http.ResponseWriter, r *http.Request) {
	datasourceInitService := c.service.GetDatasourceInitService()
	residentSources := datasourceInitService.GetResidentDataSources()

	render.JSON(w, r, SuccessResponse("获取常驻数据源状态成功", residentSources))
}

// RestartResidentDataSource 重启常驻数据源
// @Summary 重启常驻数据源
// @Description 重启指定的常驻数据源
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据源ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/restart-resident-datasource/{id} [post]
func (c *BasicLibraryController) RestartResidentDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("数据源ID参数不能为空", nil))
		return
	}

	datasourceInitService := c.service.GetDatasourceInitService()
	ctx := r.Context()

	err := datasourceInitService.RestartResidentDataSource(ctx, id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("重启常驻数据源失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("常驻数据源重启成功", nil))
}

// ReloadDataSource 重新加载数据源
// @Summary 重新加载数据源
// @Description 重新从数据库加载数据源配置并更新管理器中的实例
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据源ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/reload-datasource/{id} [post]
func (c *BasicLibraryController) ReloadDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("数据源ID参数不能为空", nil))
		return
	}

	datasourceInitService := c.service.GetDatasourceInitService()
	ctx := r.Context()

	err := datasourceInitService.ReloadDataSource(ctx, id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("重新加载数据源失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("数据源重新加载成功", nil))
}

// HealthCheckAllDataSources 对所有数据源进行健康检查
// @Summary 健康检查所有数据源
// @Description 对管理器中的所有数据源进行健康检查
// @Tags 数据基础库
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/health-check-all [post]
func (c *BasicLibraryController) HealthCheckAllDataSources(w http.ResponseWriter, r *http.Request) {
	datasourceInitService := c.service.GetDatasourceInitService()
	ctx := r.Context()

	results := datasourceInitService.HealthCheckAllDataSources(ctx)

	render.JSON(w, r, SuccessResponse("健康检查完成", results))
}
