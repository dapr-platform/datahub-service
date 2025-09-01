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
		render.JSON(w, r, BadRequestResponse( fmt.Sprintf("请求参数格式错误:%s", err.Error()), nil))
		return
	}

	if err := c.service.CreateThematicLibrary(&req); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	library, err := c.service.GetThematicLibrary(id)
	if err != nil {
		render.JSON(w, r, NotFoundResponse( "数据主题库不存在", nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	var req models.ThematicLibrary
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse( "请求参数格式错误", err))
		return
	}

	if err := c.service.UpdateThematicLibrary(id, &req); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	if err := c.service.DeleteThematicLibrary(id); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	if err := c.service.PublishThematicLibrary(id); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
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
		render.JSON(w, r, BadRequestResponse( "请求参数格式错误", err))
		return
	}

	if err := c.service.CreateThematicInterface(&req); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	thematicInterface, err := c.service.GetThematicInterface(id)
	if err != nil {
		render.JSON(w, r, NotFoundResponse( "主题接口不存在", nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	var req models.ThematicInterface
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, BadRequestResponse( "请求参数格式错误", err))
		return
	}

	if err := c.service.UpdateThematicInterface(id, &req); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("更新成功", nil))
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
		render.JSON(w, r, BadRequestResponse( "ID参数不能为空", nil))
		return
	}

	if err := c.service.DeleteThematicInterface(id); err != nil {
		render.JSON(w, r, BadRequestResponse( err.Error(), nil))
		return
	}

	render.JSON(w, r, SuccessResponse("删除成功", nil))
}
