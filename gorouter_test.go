package gorouter

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testConfig 测试用的配置实现
type testConfig struct {
	address      string
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	secretKey    string
	expiration   time.Duration
}

func (c *testConfig) GetAddress() string {
	return c.address
}

func (c *testConfig) GetReadTimeout() time.Duration {
	return c.readTimeout
}

func (c *testConfig) GetWriteTimeout() time.Duration {
	return c.writeTimeout
}

func (c *testConfig) GetIdleTimeout() time.Duration {
	return c.idleTimeout
}

func (c *testConfig) GetSecretKey() string {
	return c.secretKey
}

func (c *testConfig) GetExpiration() time.Duration {
	return c.expiration
}

// 测试用的请求和响应结构
type TestRequest struct {
	Name  string `json:"name" query:"name"`
	Value int    `json:"value" query:"value"`
}

type TestResponse struct {
	Result string `json:"result"`
}

// 创建测试配置
func newTestConfig() *testConfig {
	return &testConfig{
		address:      ":8080",
		readTimeout:  10 * time.Second,
		writeTimeout: 10 * time.Second,
		idleTimeout:  120 * time.Second,
		secretKey:    "test-secret-key",
		expiration:   24 * time.Hour,
	}
}

// TestPost 测试 POST 方法
func TestPost(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Post(router, "/test/post", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "POST: " + req.Name}, nil
	})

	// 测试正常请求
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test/post", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("期望响应码 0, 得到 %d", resp.Code)
	}

	// 测试错误方法
	req = httptest.NewRequest(http.MethodGet, "/test/post", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestPostWithAuth 测试带认证的 POST 方法
func TestPostWithAuth(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	PostWithAuth(router, "/test/post-auth", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "POST Auth: " + claims.Username}, nil
	})

	// 测试无认证请求
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test/post-auth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusUnauthorized, w.Code)
	}

	// 测试有效认证请求
	token, err := GenerateToken(cfg.GetSecretKey(), cfg.GetExpiration(), "testuser", 1)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/test/post-auth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("期望响应码 0, 得到 %d", resp.Code)
	}
}

// TestGet 测试 GET 方法
func TestGet(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Get(router, "/test/get", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "GET: " + req.Name}, nil
	})

	// 测试正常请求
	req := httptest.NewRequest(http.MethodGet, "/test/get?name=test&value=123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("期望响应码 0, 得到 %d", resp.Code)
	}

	// 测试错误方法
	req = httptest.NewRequest(http.MethodPost, "/test/get", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestGetWithAuth 测试带认证的 GET 方法
func TestGetWithAuth(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	GetWithAuth(router, "/test/get-auth", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "GET Auth: " + claims.Username}, nil
	})

	// 测试无认证请求
	req := httptest.NewRequest(http.MethodGet, "/test/get-auth?name=test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusUnauthorized, w.Code)
	}

	// 测试有效认证请求
	token, err := GenerateToken(cfg.GetSecretKey(), cfg.GetExpiration(), "testuser", 1)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/test/get-auth?name=test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}
}

// TestPut 测试 PUT 方法
func TestPut(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Put(router, "/test/put", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "PUT: " + req.Name}, nil
	})

	// 测试正常请求
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/test/put", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("期望响应码 0, 得到 %d", resp.Code)
	}

	// 测试错误方法
	req = httptest.NewRequest(http.MethodGet, "/test/put", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestPutWithAuth 测试带认证的 PUT 方法
func TestPutWithAuth(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	PutWithAuth(router, "/test/put-auth", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "PUT Auth: " + claims.Username}, nil
	})

	// 测试无认证请求
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/test/put-auth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusUnauthorized, w.Code)
	}

	// 测试有效认证请求
	token, err := GenerateToken(cfg.GetSecretKey(), cfg.GetExpiration(), "testuser", 1)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	req = httptest.NewRequest(http.MethodPut, "/test/put-auth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}
}

// TestDelete 测试 DELETE 方法
func TestDelete(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Delete(router, "/test/delete", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "DELETE: " + req.Name}, nil
	})

	// 测试正常请求
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodDelete, "/test/delete", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("期望响应码 0, 得到 %d", resp.Code)
	}

	// 测试错误方法
	req = httptest.NewRequest(http.MethodGet, "/test/delete", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestDeleteWithAuth 测试带认证的 DELETE 方法
func TestDeleteWithAuth(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	DeleteWithAuth(router, "/test/delete-auth", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "DELETE Auth: " + claims.Username}, nil
	})

	// 测试无认证请求
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodDelete, "/test/delete-auth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusUnauthorized, w.Code)
	}

	// 测试有效认证请求
	token, err := GenerateToken(cfg.GetSecretKey(), cfg.GetExpiration(), "testuser", 1)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/test/delete-auth", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}
}

// TestInvalidJSON 测试无效的 JSON 请求
func TestInvalidJSON(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Post(router, "/test/invalid", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "OK"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/test/invalid", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
	}
}

// TestEmptyBody 测试空请求体
func TestEmptyBody(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Post(router, "/test/empty", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "OK"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/test/empty", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}
}

// TestHandlerError 测试处理器返回错误
func TestHandlerError(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Post(router, "/test/error", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return nil, errors.New("test error")
	})

	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/test/error", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusInternalServerError, w.Code)
	}
}

// TestRoutesRegistration 测试路由注册
func TestRoutesRegistration(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	Get(router, "/test/get", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "OK"}, nil
	})
	Post(router, "/test/post", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "OK"}, nil
	})
	Put(router, "/test/put", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "OK"}, nil
	})
	Delete(router, "/test/delete", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "OK"}, nil
	})

	if len(router.routes) != 4 {
		t.Errorf("期望注册 4 个路由, 得到 %d", len(router.routes))
	}

	// 验证路由方法
	expectedMethods := map[string]string{
		"/test/get":    "GET",
		"/test/post":   "POST",
		"/test/put":    "PUT",
		"/test/delete": "DELETE",
	}

	for _, route := range router.routes {
		if expectedMethods[route.Path] != route.Method {
			t.Errorf("路由 %s 的方法期望 %s, 得到 %s", route.Path, expectedMethods[route.Path], route.Method)
		}
	}
}

// TestSamePathDifferentMethods 测试同一路径的不同 HTTP 方法
func TestSamePathDifferentMethods(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	// 为同一路径注册不同的 HTTP 方法
	Get(router, "/api/user", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "GET user"}, nil
	})

	Post(router, "/api/user", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "POST user"}, nil
	})

	Put(router, "/api/user", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "PUT user"}, nil
	})

	Delete(router, "/api/user", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "DELETE user"}, nil
	})

	// 测试 GET 方法
	req := httptest.NewRequest(http.MethodGet, "/api/user?name=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ := json.Marshal(resp.Data)
	var testResp TestResponse
	json.Unmarshal(data, &testResp)
	if testResp.Result != "GET user" {
		t.Errorf("GET 方法期望结果 'GET user', 得到 '%s'", testResp.Result)
	}

	// 测试 POST 方法
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/api/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	if testResp.Result != "POST user" {
		t.Errorf("POST 方法期望结果 'POST user', 得到 '%s'", testResp.Result)
	}

	// 测试 PUT 方法
	req = httptest.NewRequest(http.MethodPut, "/api/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PUT 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	if testResp.Result != "PUT user" {
		t.Errorf("PUT 方法期望结果 'PUT user', 得到 '%s'", testResp.Result)
	}

	// 测试不支持的 HTTP 方法
	req = httptest.NewRequest(http.MethodPatch, "/api/user", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH 方法期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}

	// 验证 Allow 头包含支持的方法
	allowHeader := w.Header().Get("Allow")
	if allowHeader == "" {
		t.Error("期望 Allow 头存在")
	}

	// 测试 DELETE 方法
	req = httptest.NewRequest(http.MethodDelete, "/api/user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DELETE 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	if testResp.Result != "DELETE user" {
		t.Errorf("DELETE 方法期望结果 'DELETE user', 得到 '%s'", testResp.Result)
	}
}

// TestSamePathWithAuth 测试同一路径的不同 HTTP 方法（带认证）
func TestSamePathWithAuth(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	// 为同一路径注册不同的 HTTP 方法（带认证）
	GetWithAuth(router, "/api/resource", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "GET resource: " + claims.Username}, nil
	})

	PostWithAuth(router, "/api/resource", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "POST resource: " + claims.Username}, nil
	})

	PutWithAuth(router, "/api/resource", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "PUT resource: " + claims.Username}, nil
	})

	DeleteWithAuth(router, "/api/resource", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "DELETE resource: " + claims.Username}, nil
	})

	// 生成 token
	token, err := GenerateToken(cfg.GetSecretKey(), cfg.GetExpiration(), "testuser", 1)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	// 测试 GET 方法（带认证）
	req := httptest.NewRequest(http.MethodGet, "/api/resource?name=test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ := json.Marshal(resp.Data)
	var testResp TestResponse
	json.Unmarshal(data, &testResp)
	expected := "GET resource: testuser"
	if testResp.Result != expected {
		t.Errorf("GET 方法期望结果 '%s', 得到 '%s'", expected, testResp.Result)
	}

	// 测试 POST 方法（带认证）
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/api/resource", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	expected = "POST resource: testuser"
	if testResp.Result != expected {
		t.Errorf("POST 方法期望结果 '%s', 得到 '%s'", expected, testResp.Result)
	}

	// 测试 PUT 方法（带认证）
	req = httptest.NewRequest(http.MethodPut, "/api/resource", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PUT 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	expected = "PUT resource: testuser"
	if testResp.Result != expected {
		t.Errorf("PUT 方法期望结果 '%s', 得到 '%s'", expected, testResp.Result)
	}

	// 测试 DELETE 方法（带认证）
	req = httptest.NewRequest(http.MethodDelete, "/api/resource", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DELETE 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	expected = "DELETE resource: testuser"
	if testResp.Result != expected {
		t.Errorf("DELETE 方法期望结果 '%s', 得到 '%s'", expected, testResp.Result)
	}
}

// TestSamePathMixedAuth 测试同一路径混合认证和非认证的路由
func TestSamePathMixedAuth(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	// 同一路径：GET 不需要认证，POST 需要认证
	Get(router, "/api/mixed", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "GET public"}, nil
	})

	PostWithAuth(router, "/api/mixed", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		claims := ctx.GetClaims()
		return &TestResponse{Result: "POST private: " + claims.Username}, nil
	})

	// 测试 GET 方法（不需要认证）
	req := httptest.NewRequest(http.MethodGet, "/api/mixed?name=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ := json.Marshal(resp.Data)
	var testResp TestResponse
	json.Unmarshal(data, &testResp)
	if testResp.Result != "GET public" {
		t.Errorf("GET 方法期望结果 'GET public', 得到 '%s'", testResp.Result)
	}

	// 测试 POST 方法（需要认证，但没有提供 token）
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/api/mixed", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("POST 方法（无认证）期望状态码 %d, 得到 %d", http.StatusUnauthorized, w.Code)
	}

	// 测试 POST 方法（需要认证，提供 token）
	token, err := GenerateToken(cfg.GetSecretKey(), cfg.GetExpiration(), "testuser", 1)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/mixed", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST 方法（有认证）期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, _ = json.Marshal(resp.Data)
	json.Unmarshal(data, &testResp)
	expected := "POST private: testuser"
	if testResp.Result != expected {
		t.Errorf("POST 方法期望结果 '%s', 得到 '%s'", expected, testResp.Result)
	}
}

// TestSamePathNotFound 测试同一路径不存在的情况
func TestSamePathNotFound(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	// 注册一个路径
	Get(router, "/api/exists", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "exists"}, nil
	})

	// 测试不存在的路径
	req := httptest.NewRequest(http.MethodGet, "/api/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 %d, 得到 %d", http.StatusNotFound, w.Code)
	}

	var resp CommonResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Message != "path not found" {
		t.Errorf("期望错误消息 'path not found', 得到 '%s'", resp.Message)
	}
}

// TestSamePathMethodNotAllowed 测试同一路径不支持的方法
func TestSamePathMethodNotAllowed(t *testing.T) {
	cfg := newTestConfig()
	router := NewRouter(cfg)

	// 只注册 GET 和 POST 方法
	Get(router, "/api/limited", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "GET"}, nil
	})

	Post(router, "/api/limited", func(ctx Context, req *TestRequest) (*TestResponse, error) {
		return &TestResponse{Result: "POST"}, nil
	})

	// 测试支持的 GET 方法
	req := httptest.NewRequest(http.MethodGet, "/api/limited?name=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	// 测试支持的 POST 方法
	reqBody := TestRequest{Name: "test", Value: 123}
	body, _ := json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/api/limited", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST 方法期望状态码 %d, 得到 %d", http.StatusOK, w.Code)
	}

	// 测试不支持的 PUT 方法
	req = httptest.NewRequest(http.MethodPut, "/api/limited", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT 方法期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}

	// 验证 Allow 头包含支持的方法
	allowHeader := w.Header().Get("Allow")
	if allowHeader == "" {
		t.Error("期望 Allow 头存在")
	}

	// 验证 Allow 头包含 GET 和 POST
	if !strings.Contains(allowHeader, "GET") || !strings.Contains(allowHeader, "POST") {
		t.Errorf("Allow 头应该包含 GET 和 POST, 得到 '%s'", allowHeader)
	}

	// 测试不支持的 DELETE 方法
	req = httptest.NewRequest(http.MethodDelete, "/api/limited", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE 方法期望状态码 %d, 得到 %d", http.StatusMethodNotAllowed, w.Code)
	}
}
