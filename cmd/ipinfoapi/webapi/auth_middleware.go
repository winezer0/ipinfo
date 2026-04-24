package webapi

import (
	"net/http"
	"strings"
)

// AuthMiddleware Token认证中间件
type AuthMiddleware struct {
	token     string
	enable    bool
	whiteList []string
}

// NewAuthMiddleware 创建认证中间件实例
func NewAuthMiddleware(token string, enable bool, whiteList ...string) *AuthMiddleware {
	return &AuthMiddleware{
		token:     token,
		enable:    enable,
		whiteList: whiteList,
	}
}

// Middleware 认证中间件处理函数
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enable {
			next.ServeHTTP(w, r)
			return
		}

		// 白名单路径跳过认证
		for _, path := range m.whiteList {
			if r.URL.Path == path {
				next.ServeHTTP(w, r)
				return
			}
		}

		token := extractToken(r)
		if token == "" {
			WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "需要认证")
			return
		}

		if token != m.token {
			WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "认证失败")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractToken 从请求中提取Token
// 优先从Authorization头获取，其次从URL参数token获取
func extractToken(r *http.Request) string {
	// 1. 优先检查 Authorization 头
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// 2. 从 URL 参数 token 获取
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}
