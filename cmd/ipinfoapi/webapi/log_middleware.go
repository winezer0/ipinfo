package webapi

import (
	"github.com/winezer0/xutils/logging"
	"net/http"
	"time"
)

// LoggingMiddleware request logging middleware
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		clientIP := GetClientIP(r)
		method := r.Method
		path := r.URL.Path

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		logging.Infof("Request completed ip:%v method:%v path:%v status:%v duration_ms:%v", clientIP, method, path, rw.statusCode, duration.Milliseconds())
	})
}
