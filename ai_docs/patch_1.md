# 修改一下Response定义，改为
type APIResponse struct {
	Status    int         `json:"status" example:"0"`
	Msg string      `json:"msg" example:"操作成功"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Status    int         `json:"status" example:"0"`
	Msg string      `json:"msg" example:"操作成功"`
	Data    interface{} `json:"data"`
	Total   int64       `json:"total" example:"100"`
	Page    int         `json:"page" example:"1"`
	Size    int         `json:"size" example:"10"`
}

# 