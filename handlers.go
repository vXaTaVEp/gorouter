package gorouter

import (
	"encoding/json"
	"io"
	"net/http"
)

// EchoRequest 定义Echo接口的请求结构
type EchoRequest struct {
	Data map[string]interface{} `json:"data"`
}

// EchoResponse 定义Echo接口的响应结构
type EchoResponse struct {
	Data map[string]interface{} `json:"data"`
}

// Echo 处理echo业务逻辑
func Echo(request *EchoRequest) (*EchoResponse, error) {
	return &EchoResponse{
		Data: request.Data,
	}, nil
}

// 辅助函数：解码JSON请求体
func decodeJSONBody(r *http.Request, target interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.Unmarshal(body, target)
}
