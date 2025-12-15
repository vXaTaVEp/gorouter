package gorouter

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
