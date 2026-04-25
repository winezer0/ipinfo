package webapi

import (
	"net/http"
	"runtime/debug"

	"github.com/winezer0/xutils/logging"
)

// RecoveryMiddleware 异常恢复中间件，防止panic导致服务崩溃
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				logging.Errorf("Panic recovered error:%v stack:%v path:%v method:%v", err, string(stack), r.URL.Path, r.Method)
				WriteError(w, http.StatusInternalServerError, CodeInternalErr, "服务器内部错误")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
