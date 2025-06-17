package controllers

import (
	"net/http"

	"github.com/go-chi/render"
)

// APIResponse 统一API响应结构
type APIResponse struct {
	Status int         `json:"status" example:"0"`
	Msg    string      `json:"msg" example:"操作成功"`
	Data   interface{} `json:"data,omitempty"`
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Status int         `json:"status" example:"0"`
	Msg    string      `json:"msg" example:"操作成功"`
	Data   interface{} `json:"data"`
	Total  int64       `json:"total" example:"100"`
	Page   int         `json:"page" example:"1"`
	Size   int         `json:"size" example:"10"`
}

// Response 实现render.Renderer接口
func (a *APIResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// SuccessResponse 创建成功响应
func SuccessResponse(msg string, data interface{}) render.Renderer {
	return &APIResponse{
		Status: 0,
		Msg:    msg,
		Data:   data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(httpStatus int, msg string, err error) render.Renderer {
	response := &APIResponse{
		Status: httpStatus,
		Msg:    msg,
	}

	if err != nil {
		response.Data = map[string]string{"error": err.Error()}
	}

	return response
}
