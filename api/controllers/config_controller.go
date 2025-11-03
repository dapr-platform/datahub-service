/*
 * @module api/controllers/config_controller
 * @description 配置管理控制器，提供系统配置的HTTP接口
 * @architecture RESTful API架构
 * @documentReference dev_docs/backend_requirements.md
 * @stateFlow HTTP请求 -> 控制器 -> 配置服务 -> 数据库
 * @rules 遵循RESTful API设计规范
 * @dependencies github.com/go-chi/chi/v5, github.com/go-chi/render
 * @refs service/config
 */

package controllers

import (
	"datahub-service/service"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ConfigController 配置控制器
type ConfigController struct {
}

// NewConfigController 创建配置控制器实例
func NewConfigController() *ConfigController {
	return &ConfigController{}
}

// GetAllConfigs 获取所有配置
// @Summary 获取所有系统配置
// @Description 获取系统所有配置项
// @Tags 系统配置
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /config [get]
func (c *ConfigController) GetAllConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := service.GlobalConfigService.GetAllSystemConfigs()
	if err != nil {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"msg":    "获取配置失败: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"status": http.StatusOK,
		"msg":    "获取配置成功",
		"data":   configs,
	})
}

// GetConfig 获取单个配置
// @Summary 获取单个配置
// @Description 根据键名获取配置值
// @Tags 系统配置
// @Accept json
// @Produce json
// @Param key path string true "配置键"
// @Success 200 {object} map[string]interface{}
// @Router /config/{key} [get]
func (c *ConfigController) GetConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusBadRequest,
			"msg":    "配置键不能为空",
		})
		return
	}

	value, err := service.GlobalConfigService.GetSystemConfig(key)
	if err != nil {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusNotFound,
			"msg":    "配置项不存在: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"status": http.StatusOK,
		"msg":    "获取配置成功",
		"data": map[string]interface{}{
			"key":   key,
			"value": value,
		},
	})
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
}

// UpdateConfig 更新配置
// @Summary 更新配置
// @Description 更新指定键的配置值
// @Tags 系统配置
// @Accept json
// @Produce json
// @Param key path string true "配置键"
// @Param request body UpdateConfigRequest true "更新配置请求"
// @Success 200 {object} map[string]interface{}
// @Router /config/{key} [put]
func (c *ConfigController) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusBadRequest,
			"msg":    "配置键不能为空",
		})
		return
	}

	var req UpdateConfigRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusBadRequest,
			"msg":    "请求参数错误: " + err.Error(),
		})
		return
	}

	err := service.GlobalConfigService.SetSystemConfig(key, req.Value, req.Description)
	if err != nil {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"msg":    "更新配置失败: " + err.Error(),
		})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"status": http.StatusOK,
		"msg":    "更新配置成功",
		"data": map[string]interface{}{
			"key":   key,
			"value": req.Value,
		},
	})
}

// BatchUpdateConfigsRequest 批量更新配置请求
type BatchUpdateConfigsRequest struct {
	Configs []struct {
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value" binding:"required"`
		Description string `json:"description"`
	} `json:"configs" binding:"required"`
}

// BatchUpdateConfigs 批量更新配置
// @Summary 批量更新配置
// @Description 批量更新多个配置项
// @Tags 系统配置
// @Accept json
// @Produce json
// @Param request body BatchUpdateConfigsRequest true "批量更新配置请求"
// @Success 200 {object} map[string]interface{}
// @Router /config/batch [post]
func (c *ConfigController) BatchUpdateConfigs(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateConfigsRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, map[string]interface{}{
			"status": http.StatusBadRequest,
			"msg":    "请求参数错误: " + err.Error(),
		})
		return
	}

	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, config := range req.Configs {
		err := service.GlobalConfigService.SetSystemConfig(config.Key, config.Value, config.Description)
		if err != nil {
			failedCount++
			errors = append(errors, config.Key+": "+err.Error())
		} else {
			successCount++
		}
	}

	render.JSON(w, r, map[string]interface{}{
		"status": http.StatusOK,
		"msg":    "批量更新完成",
		"data": map[string]interface{}{
			"success_count": successCount,
			"failed_count":  failedCount,
			"errors":        errors,
		},
	})
}

