package webapi

import (
	"net/http"
)

// HandlePing 处理健康检查请求
func HandlePing(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, map[string]string{"status": "ok"})
}
