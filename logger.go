package gorouter

import (
	"log"
	"net/http"
	"time"
)

// responseWriter 是一个自定义的响应写入器，用于捕获状态码和响应大小
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader 重写WriteHeader方法，捕获状态码
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write 重写Write方法，捕获响应大小
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logger 是一个HTTP中间件，记录请求信息和耗时
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建自定义响应写入器
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // 默认状态码
		}

		// 调用下一个处理器
		next.ServeHTTP(rw, r)

		// 计算请求耗时
		duration := time.Since(start)

		// 格式化并记录请求信息
		log.Printf(
			"[%s] %s %s - %d %s - %d bytes - %s - %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			rw.statusCode,
			http.StatusText(rw.statusCode),
			rw.size,
			r.UserAgent(),
			duration,
		)

		// 如果请求耗时超过一定阈值，可以单独记录
		if duration > 200*time.Millisecond {
			log.Printf("慢请求警告: %s %s - 耗时: %v", r.Method, r.RequestURI, duration)
		}
	})
}
