package controllers

import (
	"net/http"

	"github.com/go-chi/render"
)

// 业务状态码定义
const (
	StatusSuccess            = 0   // 成功
	StatusBadRequest         = 400 // 请求参数错误
	StatusUnauthorized       = 401 // 未授权
	StatusForbidden          = 403 // 禁止访问
	StatusNotFound           = 404 // 资源不存在
	StatusConflict           = 409 // 冲突（如资源状态不允许操作）
	StatusInternalError      = 500 // 服务器内部错误
	StatusServiceUnavailable = 503 // 服务不可用
)

// APIResponse 统一API响应结构
type APIResponse struct {
	Status int         `json:"status" example:"0"`
	Msg    string      `json:"msg" example:"操作成功"`
	Data   interface{} `json:"data,omitempty"`
}

// Response 实现render.Renderer接口
func (a *APIResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// 统一设置HTTP状态码为200
	w.WriteHeader(http.StatusOK)
	return nil
}

// SuccessResponse 创建成功响应
func SuccessResponse(msg string, data interface{}) render.Renderer {
	return &APIResponse{
		Status: StatusSuccess,
		Msg:    msg,
		Data:   data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(businessStatus int, msg string, err error) render.Renderer {
	response := &APIResponse{
		Status: businessStatus,
		Msg:    msg,
	}

	if err != nil {
		response.Data = map[string]string{"error": err.Error()}
	}

	return response
}

// BadRequestResponse 创建参数错误响应
func BadRequestResponse(msg string, err error) render.Renderer {
	return ErrorResponse(StatusBadRequest, msg, err)
}

// NotFoundResponse 创建资源不存在响应
func NotFoundResponse(msg string, err error) render.Renderer {
	return ErrorResponse(StatusNotFound, msg, err)
}

// ConflictResponse 创建冲突响应（状态不允许操作）
func ConflictResponse(msg string, err error) render.Renderer {
	return ErrorResponse(StatusConflict, msg, err)
}

// InternalErrorResponse 创建服务器内部错误响应
func InternalErrorResponse(msg string, err error) render.Renderer {
	return ErrorResponse(StatusInternalError, msg, err)
}
