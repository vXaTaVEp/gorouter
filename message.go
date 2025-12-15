package gorouter

// EmptyRequest 表示空请求结构
type EmptyRequest struct{}

// CommonResponse 表示通用响应结构
type CommonResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewCommonResponse 创建通用响应
func NewCommonResponse(code int, message string, data interface{}) *CommonResponse {
	return &CommonResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewSuccessCommonResponse 创建成功的通用响应
func NewSuccessCommonResponse(data interface{}) *CommonResponse {
	return &CommonResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// NewErrorCommonResponse 创建错误的通用响应
func NewErrorCommonResponse(code int, message string) *CommonResponse {
	return &CommonResponse{
		Code:    code,
		Message: message,
	}
}
