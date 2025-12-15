package gorouter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger"
)

// swaggerDoc 存储生成的 Swagger 文档
type swaggerDoc struct {
	OpenAPI    string                 `json:"openapi"`
	Info       swaggerInfo            `json:"info"`
	Servers    []swaggerServer        `json:"servers,omitempty"`
	Paths      map[string]swaggerPath `json:"paths"`
	Components swaggerComponents      `json:"components"`
}

type swaggerInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type swaggerServer struct {
	URL string `json:"url"`
}

type swaggerPath map[string]swaggerOperation

type swaggerOperation struct {
	Tags        []string                   `json:"tags,omitempty"`
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId"`
	Parameters  []swaggerParameter         `json:"parameters,omitempty"`
	RequestBody *swaggerRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]swaggerResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type swaggerParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
}

type swaggerRequestBody struct {
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required,omitempty"`
	Content     map[string]interface{} `json:"content"`
}

type swaggerResponse struct {
	Description string                 `json:"description"`
	Content     map[string]interface{} `json:"content,omitempty"`
}

type swaggerComponents struct {
	Schemas         map[string]interface{} `json:"schemas,omitempty"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes,omitempty"`
}

// EnableSwagger 启用 Swagger 文档功能
func (r *Router) EnableSwagger(config *SwaggerConfig) {
	if config == nil {
		config = &SwaggerConfig{
			Title:   "API Documentation",
			Version: "1.0.0",
		}
	}

	// 生成 Swagger 文档
	doc := r.generateSwaggerDoc(config)

	// 注册 Swagger JSON 端点
	r.registerHandler("/swagger/doc.json", http.MethodGet, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	})

	// 注册 Swagger UI
	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	)
	r.registerHandler("/swagger/", http.MethodGet, swaggerHandler)
	r.registerHandler("/swagger", http.MethodGet, func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/", http.StatusMovedPermanently)
	})
}

// generateSwaggerDoc 生成 Swagger 文档
func (r *Router) generateSwaggerDoc(config *SwaggerConfig) *swaggerDoc {
	doc := &swaggerDoc{
		OpenAPI: "3.0.0",
		Info: swaggerInfo{
			Title:       config.Title,
			Version:     config.Version,
			Description: config.Description,
		},
		Paths: make(map[string]swaggerPath),
		Components: swaggerComponents{
			Schemas:         make(map[string]interface{}),
			SecuritySchemes: make(map[string]interface{}),
		},
	}

	// 添加服务器信息
	if config.Host != "" {
		url := config.Host
		if config.BasePath != "" {
			url += config.BasePath
		}
		doc.Servers = []swaggerServer{
			{URL: url},
		}
	}

	// 添加 JWT 安全方案
	doc.Components.SecuritySchemes = map[string]interface{}{
		"bearerAuth": map[string]string{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		},
	}

	// 处理所有路由
	for _, route := range r.routes {
		if doc.Paths[route.Path] == nil {
			doc.Paths[route.Path] = make(swaggerPath)
		}

		operation := r.generateOperation(route)
		doc.Paths[route.Path][strings.ToLower(route.Method)] = operation

		// 生成请求和响应的 Schema
		if route.RequestType != nil {
			schema := r.generateSchema(route.RequestType)
			schemaName := getTypeName(route.RequestType)
			doc.Components.Schemas[schemaName] = schema
		}

		if route.ResponseType != nil {
			// 响应被包装在 CommonResponse 中
			commonRespSchema := r.generateCommonResponseSchema(route.ResponseType)
			doc.Components.Schemas["CommonResponse"] = commonRespSchema

			// 也生成响应类型本身的 Schema
			respSchema := r.generateSchema(route.ResponseType)
			respSchemaName := getTypeName(route.ResponseType)
			doc.Components.Schemas[respSchemaName] = respSchema
		}
	}

	return doc
}

// generateOperation 生成操作信息
func (r *Router) generateOperation(route RouteInfo) swaggerOperation {
	operation := swaggerOperation{
		Tags:        route.Tags,
		Summary:     route.Description,
		Description: route.Description,
		OperationID: fmt.Sprintf("%s_%s", strings.ToLower(route.Method), sanitizePath(route.Path)),
		Responses: map[string]swaggerResponse{
			"200": {
				Description: "成功响应",
				Content: map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": "#/components/schemas/CommonResponse",
						},
					},
				},
			},
			"400": {
				Description: "请求错误",
			},
			"500": {
				Description: "服务器错误",
			},
		},
	}

	// 如果需要认证，添加安全要求
	if route.RequireAuth {
		operation.Security = []map[string][]string{
			{"bearerAuth": {}},
		}
	}

	// 根据 HTTP 方法设置参数
	method := strings.ToUpper(route.Method)
	switch method {
	case "GET":
		// GET 请求使用查询参数
		if route.RequestType != nil {
			operation.Parameters = r.generateQueryParameters(route.RequestType)
		}
	case "POST", "PUT", "DELETE":
		// POST/PUT/DELETE 请求使用请求体
		if route.RequestType != nil {
			operation.RequestBody = r.generateRequestBody(route.RequestType)
		}
	}

	return operation
}

// generateQueryParameters 生成查询参数
func (r *Router) generateQueryParameters(reqType reflect.Type) []swaggerParameter {
	var params []swaggerParameter

	if reqType == nil {
		return params
	}

	// 处理指针类型
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}

	// 再次检查 nil（处理 any 类型的情况）
	if reqType == nil || reqType.Kind() != reflect.Struct {
		return params
	}

	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)
		queryTag := field.Tag.Get("query")
		if queryTag == "" || queryTag == "-" {
			continue
		}

		// 获取描述：优先使用 description tag，如果没有则留空
		description := field.Tag.Get("description")
		if description == "" {
			description = field.Tag.Get("desc")
		}

		param := swaggerParameter{
			Name:        queryTag,
			In:          "query",
			Description: description, // 使用 description tag 或留空
			Required:    false,       // 查询参数默认非必需
			Schema:      r.getFieldSchema(field.Type),
		}

		// 检查是否有 required tag
		if field.Tag.Get("validate") != "" && strings.Contains(field.Tag.Get("validate"), "required") {
			param.Required = true
		}

		params = append(params, param)
	}

	return params
}

// generateRequestBody 生成请求体
func (r *Router) generateRequestBody(reqType reflect.Type) *swaggerRequestBody {
	if reqType == nil {
		return nil
	}

	// 处理指针类型
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}

	// 再次检查 nil（处理 any 类型的情况）
	if reqType == nil {
		return &swaggerRequestBody{
			Required: true,
			Content: map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{"type": "object"},
				},
			},
		}
	}

	schemaName := getTypeName(reqType)
	if reqType.Kind() == reflect.Struct {
		return &swaggerRequestBody{
			Required: true,
			Content: map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": fmt.Sprintf("#/components/schemas/%s", schemaName),
					},
				},
			},
		}
	}

	return &swaggerRequestBody{
		Required: true,
		Content: map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": r.getFieldSchema(reqType),
			},
		},
	}
}

// generateCommonResponseSchema 生成通用响应 Schema
func (r *Router) generateCommonResponseSchema(dataType reflect.Type) map[string]interface{} {
	var dataSchema interface{}

	if dataType != nil {
		// 处理指针类型
		if dataType.Kind() == reflect.Ptr {
			dataType = dataType.Elem()
		}

		// 检查 nil（处理 any 类型的情况）
		if dataType != nil && dataType.Kind() == reflect.Struct {
			schemaName := getTypeName(dataType)
			dataSchema = map[string]interface{}{
				"$ref": fmt.Sprintf("#/components/schemas/%s", schemaName),
			}
		} else {
			dataSchema = r.getFieldSchema(dataType)
		}
	} else {
		dataSchema = map[string]interface{}{"type": "object"}
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code": map[string]interface{}{
				"type":    "integer",
				"example": 0,
			},
			"message": map[string]interface{}{
				"type":    "string",
				"example": "success",
			},
			"data": dataSchema,
		},
		"required": []string{"code", "message"},
	}
}

// generateSchema 生成 Schema
func (r *Router) generateSchema(typ reflect.Type) map[string]interface{} {
	// 检查 nil 类型（当使用 any 类型时可能为 nil）
	if typ == nil {
		return map[string]interface{}{"type": "object"}
	}

	// 处理指针类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		// 再次检查 nil（处理指向 any 的指针）
		if typ == nil {
			return map[string]interface{}{"type": "object"}
		}
	}

	switch typ.Kind() {
	case reflect.Struct:
		return r.generateStructSchema(typ)
	case reflect.Slice, reflect.Array:
		return r.generateArraySchema(typ)
	case reflect.Map:
		return r.generateMapSchema(typ)
	default:
		schema := r.getFieldSchema(typ)
		if schemaMap, ok := schema.(map[string]interface{}); ok {
			return schemaMap
		}
		return map[string]interface{}{"type": "object"}
	}
}

// generateStructSchema 生成结构体 Schema
func (r *Router) generateStructSchema(typ reflect.Type) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// 解析 json tag
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		fieldSchema := r.getFieldSchema(field.Type)
		properties[jsonName] = fieldSchema

		// 检查是否必需
		if !strings.Contains(jsonTag, "omitempty") {
			required = append(required, jsonName)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// generateArraySchema 生成数组 Schema
func (r *Router) generateArraySchema(typ reflect.Type) map[string]interface{} {
	elemType := typ.Elem()
	itemsSchema := r.getFieldSchema(elemType)

	return map[string]interface{}{
		"type":  "array",
		"items": itemsSchema,
	}
}

// generateMapSchema 生成 Map Schema
func (r *Router) generateMapSchema(typ reflect.Type) map[string]interface{} {
	valueType := typ.Elem()
	valueSchema := r.getFieldSchema(valueType)

	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": valueSchema,
	}
}

// getFieldSchema 获取字段的 Schema
func (r *Router) getFieldSchema(typ reflect.Type) interface{} {
	// 检查 nil 类型（当使用 any 类型时可能为 nil）
	if typ == nil {
		return map[string]interface{}{"type": "object"}
	}

	// 处理指针类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		// 再次检查 nil（处理指向 any 的指针）
		if typ == nil {
			return map[string]interface{}{"type": "object"}
		}
	}

	switch typ.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]interface{}{"type": "integer", "format": "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer", "format": "int64"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number", "format": "double"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Struct:
		schemaName := getTypeName(typ)
		return map[string]interface{}{
			"$ref": fmt.Sprintf("#/components/schemas/%s", schemaName),
		}
	case reflect.Slice, reflect.Array:
		return r.generateArraySchema(typ)
	case reflect.Map:
		return r.generateMapSchema(typ)
	default:
		return map[string]interface{}{"type": "object"}
	}
}

// getTypeName 获取类型名称
func getTypeName(typ reflect.Type) string {
	if typ == nil {
		return "Unknown"
	}

	// 处理指针类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	name := typ.Name()
	if name == "" {
		// 对于匿名类型，使用包路径和类型字符串
		name = strings.ReplaceAll(typ.String(), ".", "_")
		name = strings.ReplaceAll(name, "*", "")
		name = strings.ReplaceAll(name, "[", "")
		name = strings.ReplaceAll(name, "]", "")
	}

	return name
}

// sanitizePath 清理路径用于 OperationID
func sanitizePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	return path
}
