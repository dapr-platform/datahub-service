/*
 * @module api/controllers/thematic_library_controller
 * @description 数据主题库API控制器，处理HTTP请求和响应
 * @architecture MVC架构 - 控制器层
 * @documentReference dev_docs/requirements.md
 * @stateFlow HTTP请求处理流程
 * @rules 统一的错误处理和响应格式，参数验证
 * @dependencies datahub-service/service, github.com/go-chi/render
 * @refs dev_docs/model.md
 */

package controllers

import (
	"datahub-service/service"
	"datahub-service/service/models"
	"datahub-service/service/thematic_library"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ThematicLibraryController 数据主题库控制器
type ThematicLibraryController struct {
	service *thematic_library.Service
}

// NewThematicLibraryController 创建数据主题库控制器实例
func NewThematicLibraryController() *ThematicLibraryController {
	return &ThematicLibraryController{
		service: service.GlobalThematicLibraryService,
	}
}

// ThematicLibraryListResponse 数据主题库列表响应结构
type ThematicLibraryListResponse struct {
	List  []models.ThematicLibrary `json:"list"`
	Total int64                    `json:"total"`
	Page  int                      `json:"page"`
	Size  int                      `json:"size"`
}

// ThematicInterfaceListResponse 主题接口列表响应结构
type ThematicInterfaceListResponse struct {
	List  []models.ThematicInterface `json:"list"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

// UpdateThematicInterfaceFieldsRequest 更新主题接口字段配置请求结构
type UpdateThematicInterfaceFieldsRequest struct {
	InterfaceID string              `json:"interface_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Fields      []models.TableField `json:"fields" validate:"required"`
}

// CreateThematicInterfaceViewRequest 创建主题接口视图请求结构
type CreateThematicInterfaceViewRequest struct {
	InterfaceID string `json:"interface_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ViewSQL     string `json:"view_sql" validate:"required" example:"SELECT * FROM users WHERE status = 'active'"`
}

// UpdateThematicInterfaceViewRequest 更新主题接口视图请求结构
type UpdateThematicInterfaceViewRequest struct {
	InterfaceID string `json:"interface_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ViewSQL     string `json:"view_sql" validate:"required" example:"SELECT * FROM users WHERE status = 'active'"`
}

// CreateThematicLibrary 创建数据主题库
// @Summary 创建数据主题库
// @Description 创建新的数据主题库
// @Tags 数据主题库
// @Accept json
// @Produce json
// @Param library body models.ThematicLibrary true "数据主题库信息"
// @Success 201 {object} APIResponse{data=models.ThematicLibrary}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries [post]
func (c *ThematicLibraryController) CreateThematicLibrary(w http.ResponseWriter, r *http.Request) {
	var req models.ThematicLibrary
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse(fmt.Sprintf("请求参数格式错误:%s", err.Error()), nil))
		return
	}

	if err := c.service.CreateThematicLibrary(&req); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("创建成功", &req))
}

// GetThematicLibrary 获取数据主题库详情
// @Summary 获取数据主题库详情
// @Description 根据ID获取数据主题库详细信息
// @Tags 数据主题库
// @Produce json
// @Param id path string true "数据主题库ID"
// @Success 200 {object} APIResponse{data=models.ThematicLibrary}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries/{id} [get]
func (c *ThematicLibraryController) GetThematicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	library, err := c.service.GetThematicLibrary(id)
	if err != nil {
		render.JSON(w, r, NotFoundResponse("数据主题库不存在", nil))
		return
	}

	render.JSON(w, r, SuccessResponse("查询成功", library))
}

// UpdateThematicLibrary 更新数据主题库
// @Summary 更新数据主题库
// @Description 更新数据主题库信息
// @Tags 数据主题库
// @Accept json
// @Produce json
// @Param id path string true "数据主题库ID"
// @Param library body models.ThematicLibrary true "更新信息"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries/{id} [put]
func (c *ThematicLibraryController) UpdateThematicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	var req models.ThematicLibrary
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.service.UpdateThematicLibrary(id, &req); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("更新成功", nil))
}

// DeleteThematicLibrary 删除数据主题库
// @Summary 删除数据主题库
// @Description 删除数据主题库
// @Tags 数据主题库
// @Produce json
// @Param id path string true "数据主题库ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries/{id} [delete]
func (c *ThematicLibraryController) DeleteThematicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	if err := c.service.DeleteThematicLibrary(id); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("删除成功", nil))
}

// PublishThematicLibrary 发布数据主题库
// @Summary 发布数据主题库
// @Description 将主题库状态更新为已发布
// @Tags 数据主题库
// @Produce json
// @Param id path string true "数据主题库ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries/{id}/publish [post]
func (c *ThematicLibraryController) PublishThematicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	if err := c.service.PublishThematicLibrary(id); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("发布成功", nil))
}

// CreateThematicInterface 创建主题接口
// @Summary 创建主题接口
// @Description 创建新的主题接口
// @Tags 主题接口
// @Accept json
// @Produce json
// @Param thematic_interface body models.ThematicInterface true "主题接口信息"
// @Success 201 {object} APIResponse{data=models.ThematicInterface}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces [post]
func (c *ThematicLibraryController) CreateThematicInterface(w http.ResponseWriter, r *http.Request) {
	var req models.ThematicInterface
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.service.CreateThematicInterface(&req); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("创建成功", &req))
}

// GetThematicInterface 获取主题接口详情
// @Summary 获取主题接口详情
// @Description 根据ID获取主题接口详细信息
// @Tags 主题接口
// @Produce json
// @Param id path string true "主题接口ID"
// @Success 200 {object} APIResponse{data=models.ThematicInterface}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/{id} [get]
func (c *ThematicLibraryController) GetThematicInterface(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	thematicInterface, err := c.service.GetThematicInterface(id)
	if err != nil {
		render.JSON(w, r, NotFoundResponse("主题接口不存在", nil))
		return
	}

	render.JSON(w, r, SuccessResponse("查询成功", thematicInterface))
}

// UpdateThematicInterface 更新主题接口
// @Summary 更新主题接口
// @Description 更新主题接口信息
// @Tags 主题接口
// @Accept json
// @Produce json
// @Param id path string true "主题接口ID"
// @Param thematic_interface body models.ThematicInterface true "更新信息"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/{id} [put]
func (c *ThematicLibraryController) UpdateThematicInterface(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	var req models.ThematicInterface
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if err := c.service.UpdateThematicInterface(id, &req); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("更新成功", nil))
}

// GetThematicLibraryList 获取数据主题库列表
// @Summary 获取数据主题库列表
// @Description 分页获取数据主题库列表，支持多种过滤条件
// @Tags 数据主题库
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param category query string false "主题分类过滤" Enums(business,technical,analysis,report)
// @Param domain query string false "数据域过滤"
// @Param status query string false "状态过滤" Enums(draft,published,archived)
// @Param name query string false "名称搜索（支持中英文）"
// @Success 200 {object} APIResponse{data=ThematicLibraryListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /thematic-libraries [get]
func (c *ThematicLibraryController) GetThematicLibraryList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	category := r.URL.Query().Get("category")
	domain := r.URL.Query().Get("domain")
	status := r.URL.Query().Get("status")
	name := r.URL.Query().Get("name")

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 调用服务层方法
	libraries, total, err := c.service.GetThematicLibraryList(page, size, category, domain, status, name)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取数据主题库列表失败", err))
		return
	}

	// 构建响应
	response := ThematicLibraryListResponse{
		List:  libraries,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取数据主题库列表成功", response))
}

// GetThematicInterfaceList 获取主题接口列表
// @Summary 获取主题接口列表
// @Description 分页获取主题接口列表，支持多种过滤条件
// @Tags 主题接口
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页大小" default(10)
// @Param library_id query string false "主题库ID过滤"
// @Param interface_type query string false "接口类型过滤" Enums(realtime,batch)
// @Param status query string false "状态过滤" Enums(active,inactive)
// @Param name query string false "名称搜索（支持中英文）"
// @Success 200 {object} APIResponse{data=ThematicInterfaceListResponse} "获取成功"
// @Failure 400 {object} APIResponse "请求参数错误"
// @Failure 500 {object} APIResponse "服务器内部错误"
// @Router /thematic-interfaces [get]
func (c *ThematicLibraryController) GetThematicInterfaceList(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	page := 1
	size := 10
	libraryID := r.URL.Query().Get("library_id")
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
	interfaces, total, err := c.service.GetThematicInterfaceList(page, size, libraryID, interfaceType, status, name)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取主题接口列表失败", err))
		return
	}

	// 构建响应
	response := ThematicInterfaceListResponse{
		List:  interfaces,
		Total: total,
		Page:  page,
		Size:  size,
	}

	render.JSON(w, r, SuccessResponse("获取主题接口列表成功", response))
}

// DeleteThematicInterface 删除主题接口
// @Summary 删除主题接口
// @Description 软删除主题接口（更新状态为inactive）
// @Tags 主题接口
// @Produce json
// @Param id path string true "主题接口ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/{id} [delete]
func (c *ThematicLibraryController) DeleteThematicInterface(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	if err := c.service.DeleteThematicInterface(id); err != nil {
		render.JSON(w, r, BadRequestResponse(err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("删除成功", nil))
}

// UpdateThematicInterfaceFields 更新主题接口字段配置
// @Summary 更新主题接口字段配置
// @Description 更新主题接口的字段配置，自动同步数据库表结构（如表不存在则创建，存在则修改）
// @Tags 主题接口
// @Accept json
// @Produce json
// @Param request body UpdateThematicInterfaceFieldsRequest true "更新字段配置请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/update-fields [post]
func (c *ThematicLibraryController) UpdateThematicInterfaceFields(w http.ResponseWriter, r *http.Request) {
	var req UpdateThematicInterfaceFieldsRequest
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

	err := c.service.UpdateThematicInterfaceFields(req.InterfaceID, req.Fields)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("更新主题接口字段配置失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新主题接口字段配置成功", nil))
}

// CreateThematicInterfaceView 创建主题接口视图
// @Summary 创建主题接口视图
// @Description 为主题接口创建数据库视图，支持CREATE OR REPLACE VIEW语法
// @Tags 主题接口视图
// @Accept json
// @Produce json
// @Param request body CreateThematicInterfaceViewRequest true "创建视图请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/create-view [post]
func (c *ThematicLibraryController) CreateThematicInterfaceView(w http.ResponseWriter, r *http.Request) {
	var req CreateThematicInterfaceViewRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if req.InterfaceID == "" {
		render.JSON(w, r, BadRequestResponse("接口ID不能为空", nil))
		return
	}

	if req.ViewSQL == "" {
		render.JSON(w, r, BadRequestResponse("视图SQL不能为空", nil))
		return
	}

	err := c.service.CreateThematicInterfaceView(req.InterfaceID, req.ViewSQL)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("创建主题接口视图失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("创建主题接口视图成功", nil))
}

// UpdateThematicInterfaceView 更新主题接口视图
// @Summary 更新主题接口视图
// @Description 更新主题接口的数据库视图SQL
// @Tags 主题接口视图
// @Accept json
// @Produce json
// @Param request body UpdateThematicInterfaceViewRequest true "更新视图请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/update-view [post]
func (c *ThematicLibraryController) UpdateThematicInterfaceView(w http.ResponseWriter, r *http.Request) {
	var req UpdateThematicInterfaceViewRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse("请求参数格式错误", err))
		return
	}

	if req.InterfaceID == "" {
		render.JSON(w, r, BadRequestResponse("接口ID不能为空", nil))
		return
	}

	if req.ViewSQL == "" {
		render.JSON(w, r, BadRequestResponse("视图SQL不能为空", nil))
		return
	}

	err := c.service.UpdateThematicInterfaceView(req.InterfaceID, req.ViewSQL)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("更新主题接口视图失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("更新主题接口视图成功", nil))
}

// DeleteThematicInterfaceView 删除主题接口视图
// @Summary 删除主题接口视图
// @Description 删除主题接口的数据库视图
// @Tags 主题接口视图
// @Produce json
// @Param id path string true "主题接口ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/{id}/delete-view [delete]
func (c *ThematicLibraryController) DeleteThematicInterfaceView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	err := c.service.DeleteThematicInterfaceView(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("删除主题接口视图失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除主题接口视图成功", nil))
}

// GetThematicInterfaceViewSQL 获取主题接口视图SQL
// @Summary 获取主题接口视图SQL
// @Description 获取主题接口的视图SQL语句
// @Tags 主题接口视图
// @Produce json
// @Param id path string true "主题接口ID"
// @Success 200 {object} APIResponse{data=string}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/{id}/view-sql [get]
func (c *ThematicLibraryController) GetThematicInterfaceViewSQL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.JSON(w, r, BadRequestResponse("ID参数不能为空", nil))
		return
	}

	viewSQL, err := c.service.GetThematicInterfaceViewSQL(id)
	if err != nil {
		render.JSON(w, r, InternalErrorResponse("获取主题接口视图SQL失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取主题接口视图SQL成功", viewSQL))
}
