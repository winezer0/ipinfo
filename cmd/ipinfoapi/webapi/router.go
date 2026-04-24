package webapi

import (
	"net/http"

	"github.com/winezer0/ipinfo/pkg/queryip"
)

// RegisterRoutes 注册所有路由和中间件
func RegisterRoutes(engine *queryip.DBEngine, authToken string, authEnable bool) http.Handler {
	apiHandler := NewAPIHandler(engine)

	// 白名单路径不需要认证
	authMiddleware := NewAuthMiddleware(authToken, authEnable, "/ping")

	mux := http.NewServeMux()

	// 健康检查端点（不需要认证）
	mux.HandleFunc("/ping", HandlePing)

	// 业务路由（需要认证）
	mux.HandleFunc("/asn", apiHandler.HandleASN)
	mux.HandleFunc("/ip", apiHandler.HandleIP)
	mux.HandleFunc("/all", apiHandler.HandleAll)
	mux.HandleFunc("/full", apiHandler.HandleFull)

	// 组装中间件链：Recovery -> Logging -> Auth -> Handler
	var handler http.Handler = mux

	// 认证中间件（白名单路径跳过认证）
	if authEnable {
		handler = authMiddleware.Middleware(mux)
	}

	// 日志中间件
	handler = LoggingMiddleware(handler)

	// 异常恢复中间件（最外层）
	handler = RecoveryMiddleware(handler)

	return handler
}
