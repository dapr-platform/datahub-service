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
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ThematicLibraryController 数据主题库控制器
type ThematicLibraryController struct {
	service *service.ThematicLibraryService
}

// NewThematicLibraryController 创建数据主题库控制器实例
func NewThematicLibraryController() *ThematicLibraryController {
	return &ThematicLibraryController{
		service: service.NewThematicLibraryService(),
	}
}

// CreateThematicLibraryRequest 创建数据主题库请求结构
type CreateThematicLibraryRequest struct {
	Name            string   `json:"name" validate:"required" example:"用户行为分析主题库"`
	Code            string   `json:"code" validate:"required" example:"user_behavior_analysis"`
	Category        string   `json:"category" validate:"required" example:"analysis"`
	Domain          string   `json:"domain" validate:"required" example:"user"`
	Description     string   `json:"description" example:"用户行为数据分析和挖掘"`
	Tags            []string `json:"tags" example:"用户,行为,分析"`
	SourceLibraries []string `json:"source_libraries" example:"user_basic_library,event_basic_library"`
	AccessLevel     string   `json:"access_level" example:"internal"`
	UpdateFrequency string   `json:"update_frequency" example:"daily"`
	RetentionPeriod int      `json:"retention_period" example:"365"`
}

// UpdateThematicLibraryRequest 更新数据主题库请求结构
type UpdateThematicLibraryRequest struct {
	Name            string   `json:"name,omitempty" example:"用户行为分析主题库"`
	Description     string   `json:"description,omitempty" example:"用户行为数据分析和挖掘"`
	Tags            []string `json:"tags,omitempty" example:"用户,行为,分析"`
	AccessLevel     string   `json:"access_level,omitempty" example:"internal"`
	UpdateFrequency string   `json:"update_frequency,omitempty" example:"daily"`
	RetentionPeriod int      `json:"retention_period,omitempty" example:"365"`
	Status          string   `json:"status,omitempty" example:"active"`
}

// CreateThematicInterfaceRequest 创建主题库接口请求结构
type CreateThematicInterfaceRequest struct {
	LibraryID   string                 `json:"library_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	NameZh      string                 `json:"name_zh" validate:"required" example:"用户行为事件接口"`
	NameEn      string                 `json:"name_en" validate:"required" example:"user_behavior_events"`
	Type        string                 `json:"type" validate:"required" example:"realtime"`
	Config      map[string]interface{} `json:"config" validate:"required"`
	Description string                 `json:"description" example:"实时用户行为事件数据接口"`
}

// CreateThematicLibrary 创建数据主题库
// @Summary 创建数据主题库
// @Description 创建新的数据主题库
// @Tags 数据主题库
// @Accept json
// @Produce json
// @Param library body CreateThematicLibraryRequest true "数据主题库信息"
// @Success 201 {object} APIResponse{data=models.ThematicLibrary}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries [post]
func (c *ThematicLibraryController) CreateThematicLibrary(w http.ResponseWriter, r *http.Request) {
	var req CreateThematicLibraryRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	library := &models.ThematicLibrary{
		Name:            req.Name,
		Code:            req.Code,
		Category:        req.Category,
		Domain:          req.Domain,
		Description:     req.Description,
		Tags:            req.Tags,
		SourceLibraries: req.SourceLibraries,
		AccessLevel:     req.AccessLevel,
		UpdateFrequency: req.UpdateFrequency,
		RetentionPeriod: req.RetentionPeriod,
	}

	if err := c.service.CreateThematicLibrary(library); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, APIResponse{
		Status: http.StatusCreated,
		Msg:    "创建成功",
		Data:   library,
	})
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "ID参数不能为空",
		})
		return
	}

	library, err := c.service.GetThematicLibrary(id)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "数据主题库不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "查询成功",
		Data:   library,
	})
}

// GetThematicLibraries 获取数据主题库列表
// @Summary 获取数据主题库列表
// @Description 分页获取数据主题库列表
// @Tags 数据主题库
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param category query string false "主题分类筛选" Enums(business,technical,analysis,report)
// @Param domain query string false "数据域筛选" Enums(user,order,product,finance,marketing)
// @Param status query string false "状态筛选" Enums(active,inactive,archived)
// @Success 200 {object} PaginatedResponse{data=[]models.ThematicLibrary}
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries [get]
func (c *ThematicLibraryController) GetThematicLibraries(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	category := r.URL.Query().Get("category")
	domain := r.URL.Query().Get("domain")
	status := r.URL.Query().Get("status")

	libraries, total, err := c.service.GetThematicLibraries(page, size, category, domain, status)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "查询失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "查询成功",
		Data:   libraries,
		Total:  total,
		Page:   page,
		Size:   size,
	})
}

// UpdateThematicLibrary 更新数据主题库
// @Summary 更新数据主题库
// @Description 更新数据主题库信息
// @Tags 数据主题库
// @Accept json
// @Produce json
// @Param id path string true "数据主题库ID"
// @Param library body UpdateThematicLibraryRequest true "更新信息"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-libraries/{id} [put]
func (c *ThematicLibraryController) UpdateThematicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "ID参数不能为空",
		})
		return
	}

	var req UpdateThematicLibraryRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.AccessLevel != "" {
		updates["access_level"] = req.AccessLevel
	}
	if req.UpdateFrequency != "" {
		updates["update_frequency"] = req.UpdateFrequency
	}
	if req.RetentionPeriod > 0 {
		updates["retention_period"] = req.RetentionPeriod
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if len(updates) == 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "没有需要更新的字段",
		})
		return
	}

	if err := c.service.UpdateThematicLibrary(id, updates); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "更新成功",
	})
}

// DeleteThematicLibrary 删除数据主题库
// @Summary 删除数据主题库
// @Description 软删除数据主题库（更新状态为archived）
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "ID参数不能为空",
		})
		return
	}

	if err := c.service.DeleteThematicLibrary(id); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "删除成功",
	})
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "ID参数不能为空",
		})
		return
	}

	if err := c.service.PublishThematicLibrary(id); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "发布成功",
	})
}

// CreateThematicInterface 创建主题库接口
// @Summary 创建主题库接口
// @Description 为主题库创建新的数据接口
// @Tags 数据主题库
// @Accept json
// @Produce json
// @Param interface body CreateThematicInterfaceRequest true "主题库接口信息"
// @Success 201 {object} APIResponse{data=models.ThematicInterface}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces [post]
func (c *ThematicLibraryController) CreateThematicInterface(w http.ResponseWriter, r *http.Request) {
	var req CreateThematicInterfaceRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "请求参数格式错误",
		})
		return
	}

	interfaceData := &models.ThematicInterface{
		LibraryID:   req.LibraryID,
		NameZh:      req.NameZh,
		NameEn:      req.NameEn,
		Type:        req.Type,
		Config:      req.Config,
		Description: req.Description,
	}

	if err := c.service.CreateThematicInterface(interfaceData); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, APIResponse{
		Status: http.StatusCreated,
		Msg:    "创建成功",
		Data:   interfaceData,
	})
}

// GetThematicInterface 获取主题库接口详情
// @Summary 获取主题库接口详情
// @Description 根据ID获取主题库接口详细信息
// @Tags 数据主题库
// @Produce json
// @Param id path string true "主题库接口ID"
// @Success 200 {object} APIResponse{data=models.ThematicInterface}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces/{id} [get]
func (c *ThematicLibraryController) GetThematicInterface(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status: http.StatusBadRequest,
			Msg:    "ID参数不能为空",
		})
		return
	}

	interfaceData, err := c.service.GetThematicInterface(id)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, APIResponse{
			Status: http.StatusNotFound,
			Msg:    "主题库接口不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status: http.StatusOK,
		Msg:    "查询成功",
		Data:   interfaceData,
	})
}

// GetThematicInterfaces 获取主题库接口列表
// @Summary 获取主题库接口列表
// @Description 分页获取主题库接口列表
// @Tags 数据主题库
// @Produce json
// @Param library_id query string false "主题库ID筛选"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} PaginatedResponse{data=[]models.ThematicInterface}
// @Failure 500 {object} APIResponse
// @Router /thematic-interfaces [get]
func (c *ThematicLibraryController) GetThematicInterfaces(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	libraryID := r.URL.Query().Get("library_id")

	interfaces, total, err := c.service.GetThematicInterfaces(libraryID, page, size)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, APIResponse{
			Status: http.StatusInternalServerError,
			Msg:    "查询失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status: http.StatusOK,
		Msg:    "查询成功",
		Data:   interfaces,
		Total:  total,
		Page:   page,
		Size:   size,
	})
}
