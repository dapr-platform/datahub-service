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

// ScheduleConfigRequest 调度配置请求
type ScheduleConfigRequest struct {
	DataSourceID   string                 `json:"data_source_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	ScheduleType   string                 `json:"schedule_type" validate:"required" example:"cron"` // cron, interval, manual
	ScheduleConfig map[string]interface{} `json:"schedule_config" validate:"required"`              // 调度配置
	IsEnabled      bool                   `json:"is_enabled" example:"true"`                        // 是否启用
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.CreateBasicLibrary(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "添加数据基础库失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("添加数据基础库成功", nil))
}

// @Summary 删除数据基础库
// @Description 删除数据基础库
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.BasicLibrary true "数据基础库请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/delete-basic-library [post]
func (c *BasicLibraryController) DeleteBasicLibrary(w http.ResponseWriter, r *http.Request) {
	var req models.BasicLibrary
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.DeleteBasicLibrary(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "删除数据基础库失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("删除数据基础库成功", nil))
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.CreateDataSource(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "添加数据源失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("添加数据源成功", nil))
}

// @Summary 删除数据源
// @Description 删除数据源
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.DataSource true "数据源请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/delete-datasource [post]
func (c *BasicLibraryController) DeleteDataSource(w http.ResponseWriter, r *http.Request) {
	var req models.DataSource
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.DeleteDataSource(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "删除数据源失败", err))
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.CreateDataInterface(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "添加数据接口失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("添加数据接口成功", nil))
}

// @Summary 删除数据接口
// @Description 删除数据接口
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.DataInterface true "数据接口请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/delete-interface [post]
func (c *BasicLibraryController) DeleteInterface(w http.ResponseWriter, r *http.Request) {
	var req models.DataInterface
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.DeleteDataInterface(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "删除数据接口失败", err))
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	result, err := c.service.TestDataSource(req.DataSourceID, req.TestType, req.Config)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "数据源测试失败", err))
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	result, err := c.service.TestInterface(req.InterfaceID, req.TestType, req.Parameters, req.Options)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "接口测试失败", err))
		return
	}

	render.JSON(w, r, SuccessResponse("测试完成", result))
}

// ConfigureSchedule 配置数据源调度
// @Summary 配置数据源调度
// @Description 为批量数据源配置调度任务
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.ScheduleConfig true "调度配置请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /basic-libraries/configure-schedule [post]
func (c *BasicLibraryController) ConfigureSchedule(w http.ResponseWriter, r *http.Request) {
	var req models.ScheduleConfig
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.ConfigureSchedule(&req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "调度配置失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("调度配置成功", nil))
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "数据源ID参数不能为空", nil))
		return
	}

	status, err := c.service.GetDataSourceStatus(id)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "获取数据源状态失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("获取状态成功", status))
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "接口ID参数不能为空", nil))
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
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "数据预览失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("数据预览成功", data))
}
