/*
 * @module api/controllers/basic_library_controller
 * @description 数据基础库API控制器，处理HTTP请求和响应
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

// BasicLibraryController 数据基础库控制器
type BasicLibraryController struct {
	service *service.BasicLibraryService
}

// NewBasicLibraryController 创建数据基础库控制器实例
func NewBasicLibraryController() *BasicLibraryController {
	return &BasicLibraryController{
		service: service.NewBasicLibraryService(),
	}
}

// CreateBasicLibraryRequest 创建数据基础库请求结构
type CreateBasicLibraryRequest struct {
	NameZh      string `json:"name_zh" validate:"required" example:"用户数据基础库"`
	NameEn      string `json:"name_en" validate:"required" example:"user_basic_library"`
	Description string `json:"description" example:"存储用户基础信息的数据库"`
}

// UpdateBasicLibraryRequest 更新数据基础库请求结构
type UpdateBasicLibraryRequest struct {
	NameZh      string `json:"name_zh,omitempty" example:"用户数据基础库"`
	Description string `json:"description,omitempty" example:"存储用户基础信息的数据库"`
	Status      string `json:"status,omitempty" example:"active"`
}


// CreateBasicLibrary 创建数据基础库
// @Summary 创建数据基础库
// @Description 创建新的数据基础库
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param library body CreateBasicLibraryRequest true "数据基础库信息"
// @Success 201 {object} APIResponse{data=models.BasicLibrary}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries [post]
func (c *BasicLibraryController) CreateBasicLibrary(w http.ResponseWriter, r *http.Request) {
	var req CreateBasicLibraryRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	library := &models.BasicLibrary{
		NameZh:      req.NameZh,
		NameEn:      req.NameEn,
		Description: req.Description,
	}

	if err := c.service.CreateBasicLibrary(library); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: err.Error(),
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, APIResponse{
		Status:    http.StatusCreated,
		Msg: "创建成功",
		Data:    library,
	})
}

// GetBasicLibrary 获取数据基础库详情
// @Summary 获取数据基础库详情
// @Description 根据ID获取数据基础库详细信息
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据基础库ID"
// @Success 200 {object} APIResponse{data=models.BasicLibrary}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/{id} [get]
func (c *BasicLibraryController) GetBasicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "ID参数不能为空",
		})
		return
	}

	library, err := c.service.GetBasicLibrary(id)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusNotFound,
			Msg: "数据基础库不存在",
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "查询成功",
		Data:    library,
	})
}

// GetBasicLibraries 获取数据基础库列表
// @Summary 获取数据基础库列表
// @Description 分页获取数据基础库列表
// @Tags 数据基础库
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "状态筛选" Enums(active,inactive,archived)
// @Success 200 {object} PaginatedResponse{data=[]models.BasicLibrary}
// @Failure 500 {object} APIResponse
// @Router /basic-libraries [get]
func (c *BasicLibraryController) GetBasicLibraries(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 10
	}

	status := r.URL.Query().Get("status")

	libraries, total, err := c.service.GetBasicLibraries(page, size, status)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusInternalServerError,
			Msg: "查询失败",
		})
		return
	}

	render.JSON(w, r, PaginatedResponse{
		Status:    http.StatusOK,
		Msg: "查询成功",
		Data:    libraries,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// UpdateBasicLibrary 更新数据基础库
// @Summary 更新数据基础库
// @Description 更新数据基础库信息
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param id path string true "数据基础库ID"
// @Param library body UpdateBasicLibraryRequest true "更新信息"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/{id} [put]
func (c *BasicLibraryController) UpdateBasicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "ID参数不能为空",
		})
		return
	}

	var req UpdateBasicLibraryRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "请求参数格式错误",
		})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.NameZh != "" {
		updates["name_zh"] = req.NameZh
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if len(updates) == 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "没有需要更新的字段",
		})
		return
	}

	if err := c.service.UpdateBasicLibrary(id, updates); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "更新成功",
	})
}

// DeleteBasicLibrary 删除数据基础库
// @Summary 删除数据基础库
// @Description 软删除数据基础库（更新状态为archived）
// @Tags 数据基础库
// @Produce json
// @Param id path string true "数据基础库ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/{id} [delete]
func (c *BasicLibraryController) DeleteBasicLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: "ID参数不能为空",
		})
		return
	}

	if err := c.service.DeleteBasicLibrary(id); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, APIResponse{
			Status:    http.StatusBadRequest,
			Msg: err.Error(),
		})
		return
	}

	render.JSON(w, r, APIResponse{
		Status:    http.StatusOK,
		Msg: "删除成功",
	})
}
