package gorouter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery 恢复中间件，捕获panic并返回友好的错误响应
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// 记录panic信息
				stack := debug.Stack()
				log.Printf("Panic recovered: %v\nStack: %s", err, string(stack))

				// 设置响应头
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				// 返回错误响应
				errorResponse := map[string]interface{}{
					"code":    500,
					"message": "服务器内部错误",
					"error":   fmt.Sprintf("%v", err),
				}

				if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
					log.Printf("Failed to encode error response: %v", err)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
