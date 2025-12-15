package gorouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/vXaTaVEp/l"
)

// RouteInfo 存储路由信息
type RouteInfo struct {
	Path   string
	Method string
}

// Router 管理HTTP路由
type Router struct {
	mux    *http.ServeMux
	server *http.Server
	routes []RouteInfo
	cfg    Config
}

// NewRouter 创建新的路由器实例
func NewRouter(cfg Config) *Router {
	router := &Router{
		mux: http.NewServeMux(),
		cfg: cfg,
	}

	router.server = &http.Server{
		Addr:         cfg.GetAddress(),
		Handler:      router,
		ReadTimeout:  cfg.GetReadTimeout(),
		WriteTimeout: cfg.GetWriteTimeout(),
		IdleTimeout:  cfg.GetIdleTimeout(),
	}

	return router
}

// Start 启动HTTP服务器
func (r *Router) Start() error {
	return r.server.ListenAndServe()
}

// Stop 优雅关闭HTTP服务器
func (r *Router) Stop(ctx context.Context) error {
	fmt.Println("正在关闭HTTP服务器...")
	return r.server.Shutdown(ctx)
}

// AddRoute 添加POST路由
func (r *Router) AddRoute(path string, handlerFunc http.HandlerFunc) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handlerFunc(w, req)
	})
}

// Post 添加路由，使用postHandler包装处理函数
func Post[Req any, Resp any](
	r *Router, path string, handlerFunc postHandlerFunc[Req, Resp],
) {
	r.mux.HandleFunc(
		path,
		postHandler(r.cfg, handlerFunc, false),
	)
	r.routes = append(r.routes, RouteInfo{Path: path, Method: "POST"})
}

// PostWithAuth 添加需要JWT认证的路由
func PostWithAuth[Req any, Resp any](
	r *Router, path string, handlerFunc postHandlerFunc[Req, Resp],
) {
	r.mux.HandleFunc(
		path,
		postHandler(r.cfg, handlerFunc, true),
	)
	r.routes = append(r.routes, RouteInfo{Path: path, Method: "POST"})
}

// Get 添加路由，使用getHandler包装处理函数
func Get[Req any, Resp any](
	r *Router, path string, handlerFunc getHandlerFunc[Req, Resp],
) {
	r.mux.HandleFunc(
		path,
		getHandler(r.cfg, handlerFunc, false),
	)
	r.routes = append(r.routes, RouteInfo{Path: path, Method: "GET"})
}

// GetWithAuth 添加需要JWT认证的路由
func GetWithAuth[Req any, Resp any](
	r *Router, path string, handlerFunc getHandlerFunc[Req, Resp],
) {
	r.mux.HandleFunc(
		path,
		getHandler(r.cfg, handlerFunc, true),
	)
	r.routes = append(r.routes, RouteInfo{Path: path, Method: "GET"})
}

// ServeHTTP 实现http.Handler接口，应用中间件
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 应用中间件链：Recovery -> CORS -> Logger -> 业务处理
	handler := Recovery(CORS(Logger(r.mux)))
	handler.ServeHTTP(w, req)
}

type Context interface {
	GetContext() context.Context
	GetConfig() Config
	GetClaims() Claims
}

// handlerContext 处理器上下文，包含配置等信息
type handlerContext struct {
	ctx    context.Context
	config Config
	claims Claims
}

func (h handlerContext) GetContext() context.Context {
	return h.ctx
}

func (h handlerContext) GetConfig() Config {
	return h.config
}

func (h handlerContext) GetClaims() Claims {
	return h.claims
}

// postHandlerFunc 定义业务处理函数的通用类型
// 接收一个上下文、请求结构体指针，返回一个响应结构体和错误
type postHandlerFunc[Req any, Resp any] func(Context, *Req) (*Resp, error)

// getHandlerFunc 定义业务处理函数的通用类型
// 接收一个上下文、请求结构体指针，返回一个响应结构体和错误
type getHandlerFunc[Req any, Resp any] func(Context, *Req) (*Resp, error)

// postHandler 封装业务处理函数为HTTP处理器
// 处理请求解析和响应生成，确保所有接口使用POST方法
func postHandler[Req any, Resp any](cfg Config, handler postHandlerFunc[Req, Resp], requireAuth bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只允许POST方法
		if r.Method != http.MethodPost {
			sendErrorResponse(w, http.StatusMethodNotAllowed, "only POST method is supported")
			return
		}

		// 创建处理器上下文
		ctx := &handlerContext{
			ctx:    r.Context(),
			config: cfg,
		}

		// JWT认证中间件
		if requireAuth {
			claims, err := GetUserFromRequest(cfg.GetSecretKey(), r)
			if err != nil {
				sendErrorResponse(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
				return
			}
			// 将用户信息存储到请求上下文中，供后续使用
			ctx.claims = *claims
		}

		// 记录请求开始时间
		start := time.Now()

		// 解析请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		defer r.Body.Close()

		// 创建请求类型的实例
		var req Req
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				sendErrorResponse(w, http.StatusBadRequest, "invalid JSON data")
				return
			}

			// 记录请求参数，使用JSON格式
			requestName := reflect.TypeOf(req).Name()
			l.Infof("%s %s", requestName, PrettyJSON(req))
		} else {
			l.Infof("Request: %s {}")
		}

		// 调用业务处理函数
		resp, err := handler(ctx, &req)
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		fullResp := NewSuccessCommonResponse(resp)

		// 发送响应
		sendJSONResponse(w, fullResp)

		duration := time.Since(start)

		respType := reflect.TypeOf(resp)
		respName := ""
		if respType != nil {
			if respType.Kind() == reflect.Ptr {
				respName = respType.Elem().Name()
			} else {
				respName = respType.Name()
			}
		} else {
			respName = "nil"
		}

		// 使用JSON格式打印响应
		l.Infof("%s %s, cost: %v", respName, PrettyJSON(fullResp), duration)
	}
}

// 辅助函数：发送JSON响应
func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// 辅助函数：发送错误响应
func sendErrorResponse(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := NewErrorCommonResponse(statusCode, msg)
	json.NewEncoder(w).Encode(response)
}

// getHandler 封装业务处理函数为HTTP处理器
// 处理请求解析和响应生成，确保所有接口使用GET方法
func getHandler[Req any, Resp any](cfg Config, handler getHandlerFunc[Req, Resp], requireAuth bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只允许GET方法
		if r.Method != http.MethodGet {
			sendErrorResponse(w, http.StatusMethodNotAllowed, "only GET method is supported")
			return
		}

		// 创建处理器上下文
		ctx := &handlerContext{
			ctx:    r.Context(),
			config: cfg,
		}

		// JWT认证中间件
		if requireAuth {
			claims, err := GetUserFromRequest(cfg.GetSecretKey(), r)
			if err != nil {
				sendErrorResponse(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
				return
			}
			// 将用户信息存储到请求上下文中，供后续使用
			ctx.claims = *claims
		}

		// 记录请求开始时间
		start := time.Now()

		// 创建请求类型的实例
		var req Req

		// 从查询参数解析到结构体
		if err := parseQueryToStruct(r, &req); err != nil {
			sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to parse query parameters: %v", err))
			return
		}

		requestName := reflect.TypeOf(req).Name()

		// 记录请求参数，使用JSON格式
		l.Infof("%s %s", requestName, PrettyJSON(req))

		// 调用业务处理函数
		resp, err := handler(ctx, &req)
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		fullResp := NewSuccessCommonResponse(resp)

		// 发送响应
		sendJSONResponse(w, fullResp)

		duration := time.Since(start)

		respType := reflect.TypeOf(resp)
		respName := ""
		if respType != nil {
			if respType.Kind() == reflect.Ptr {
				respName = respType.Elem().Name()
			} else {
				respName = respType.Name()
			}
		} else {
			respName = "nil"
		}

		// 使用JSON格式打印响应
		l.Infof("%s %s, cost: %v", respName, PrettyJSON(fullResp), duration)
	}
}

// parseQueryToStruct 使用反射将查询参数解析到结构体
// 支持通过 query tag 指定查询参数名
func parseQueryToStruct(r *http.Request, target interface{}) error {
	query := r.URL.Query()
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetType := targetValue.Type()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := targetValue.Field(i)

		// 获取 query tag
		queryTag := field.Tag.Get("query")
		if queryTag == "" || queryTag == "-" {
			continue
		}

		// 从查询参数中获取值
		values, exists := query[queryTag]
		if !exists || len(values) == 0 {
			continue
		}

		// 取第一个值
		valueStr := values[0]

		// 根据字段类型进行转换
		if err := setFieldValue(fieldValue, valueStr); err != nil {
			return fmt.Errorf("failed to parse field %s: %v", field.Name, err)
		}
	}

	return nil
}

// setFieldValue 根据字段类型设置值
func setFieldValue(fieldValue reflect.Value, valueStr string) error {
	if !fieldValue.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	kind := fieldValue.Kind()

	switch kind {
	case reflect.String:
		fieldValue.SetString(valueStr)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return err
		}
		fieldValue.SetInt(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return err
		}
		fieldValue.SetUint(val)

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return err
		}
		fieldValue.SetFloat(val)

	case reflect.Bool:
		val, err := strconv.ParseBool(valueStr)
		if err != nil {
			return err
		}
		fieldValue.SetBool(val)

	default:
		return fmt.Errorf("unsupported field type: %v", kind)
	}

	return nil
}

// PrintRoutes 打印router注册的全部路由
func (r *Router) PrintRoutes() {
	fmt.Println("registered routes:")
	for _, route := range r.routes {
		fmt.Printf(" |-> %-6s %s\n", route.Method, route.Path)
	}
}
