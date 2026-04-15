// Package middleware 提供 JSON 格式结构化日志中间件。
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// requestIDKey 是请求 ID 在 context 中的 key。
type requestIDKeyType struct{}

var requestIDKey = requestIDKeyType{}

// GetRequestID 从 context 中获取请求 ID。
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// generateRequestID 生成 8 字节随机请求 ID。
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// logResponseWriter 包装 http.ResponseWriter，捕获状态码和响应大小。
type logResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *logResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *logResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// StructuredLogMiddleware 返回 JSON 格式结构化日志中间件。
// 记录每个请求的方法、路径、状态码、耗时、请求 ID 等信息。
func StructuredLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := generateRequestID()

		// 将请求 ID 注入 context
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)

		// 设置响应头中的请求 ID
		w.Header().Set("X-Request-ID", requestID)

		// 包装 ResponseWriter 以捕获状态码
		lw := &logResponseWriter{ResponseWriter: w, status: http.StatusOK}

		// 执行下一个处理器
		next.ServeHTTP(lw, r)

		// 计算耗时
		duration := time.Since(start)

		// 根据状态码选择日志级别
		level := slog.LevelInfo
		if lw.status >= 500 {
			level = slog.LevelError
		} else if lw.status >= 400 {
			level = slog.LevelWarn
		}

		// 输出结构化日志
		slog.Log(r.Context(), level, "HTTP 请求",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", lw.status,
			"size", lw.size,
			"duration_ms", duration.Milliseconds(),
			"remote", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"module", "api",
		)
	})
}
