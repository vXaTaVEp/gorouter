package gorouter

// SwaggerConfig Swagger 配置信息
type SwaggerConfig struct {
	Title       string // API 标题
	Version     string // API 版本
	Description string // API 描述
	Host        string // API 主机地址（可选，如 "localhost:8080"）
	BasePath    string // API 基础路径（可选，如 "/api/v1"）
}
