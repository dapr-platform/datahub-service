package controllers

import (
	"datahub-service/service"
	"datahub-service/service/database"
	"datahub-service/service/models"
	"net/http"

	"github.com/go-chi/render"
)

type TableController struct {
	service *database.SchemaService
}

func NewTableController() *TableController {
	return &TableController{
		service: service.GlobalSchemaService,
	}
}

// ManageTableSchema 管理表结构
// @Summary 管理数据库表结构
// @Description 通过PgMetaApi动态创建、修改、删除数据库表结构
// @Tags 数据基础库
// @Accept json
// @Produce json
// @Param request body models.TableSchemaRequest true "表结构操作请求"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /tables/manage-schema [post]
func (c *TableController) ManageTableSchema(w http.ResponseWriter, r *http.Request) {
	var req models.TableSchemaRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse(http.StatusBadRequest, "请求参数格式错误", err))
		return
	}

	err := c.service.ManageTableSchema(req.InterfaceID, req.Operation, req.SchemaName, req.TableName, req.Fields)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse(http.StatusInternalServerError, "表结构操作失败: "+err.Error(), err))
		return
	}

	render.JSON(w, r, SuccessResponse("表结构操作成功", nil))
}
