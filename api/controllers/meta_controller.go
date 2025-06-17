package controllers

import (
	"datahub-service/service/meta"
	"net/http"

	"github.com/go-chi/render"
)

type MetaController struct {
}

func NewMetaController() *MetaController {
	return &MetaController{}
}

// @Summary 获取所有数据源类型元数据
// @Description 获取所有数据源类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]meta.DataSourceTypeDefinition}
// @Failure 500 {object} APIResponse
// @Router /meta/basic-libraries/data-sources [get]
func (c *MetaController) GetDataSourceTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据源类型元数据成功", meta.DataSourceTypes))
}

// @Summary 获取所有数据源配置元数据
// @Description 获取所有数据源配置
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]meta.DataInterfaceConfigDefinition}
// @Failure 500 {object} APIResponse
// @Router /meta/basic-libraries/data-interface-configs [get]
func (c *MetaController) GetDataInterfaceConfigs(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据源配置元数据成功", meta.DataInterfaceConfigDefinitions))
}

// @Summary 获取所有同步任务类型元数据
// @Description 获取所有同步任务类型元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]meta.SyncTaskScheduleDefinition}
// @Failure 500 {object} APIResponse
// @Router /meta/basic-libraries/sync-task-types [get]
func (c *MetaController) GetSyncTaskTypes(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取同步任务类型元数据成功", meta.SyncTaskScheduleDefinitions))
}

// @Summary 获取所有同步任务相关元数据
// @Description 获取所有同步任务相关元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]map[string]string}
// @Failure 500 {object} APIResponse
// @Router /meta/basic-libraries/sync-task-meta [get]
func (c *MetaController) GetSyncTaskRelated(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取同步任务相关元数据成功", meta.SyncTaskMetas))
}

// @Summary 获取所有数据主题库分类元数据
// @Description 获取所有数据主题库分类元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryCategory}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/categories [get]
func (c *MetaController) GetThematicLibraryCategories(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据主题库分类元数据成功", meta.ThematicLibraryCategories))
}

// @Summary 获取所有数据主题库域元数据
// @Description 获取所有数据主题库域元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryDomain}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/domains [get]
func (c *MetaController) GetThematicLibraryDomains(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据主题库域元数据成功", meta.ThematicLibraryDomains))
}

// @Summary 获取所有数据主题库访问级别元数据
// @Description 获取所有数据主题库访问级别元数据
// @Tags 元数据
// @Produce json
// @Success 200 {object} APIResponse{data=[]meta.ThematicLibraryAccessLevel}
// @Failure 500 {object} APIResponse
// @Router /meta/thematic-libraries/access-levels [get]
func (c *MetaController) GetThematicLibraryAccessLevels(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, SuccessResponse("获取数据主题库访问级别元数据成功", meta.ThematicLibraryAccessLevels))
}
