package webapi

import "net/http"

// HandleIP 处理IP定位查询请求
func (h *APIHandler) HandleIP(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("q")

	if ip == "" {
		ip = GetClientIP(r)
		if ip == "" {
			WriteError(w, http.StatusBadRequest, CodeInvalidParam, "缺少参数q且无法获取客户端IP")
			return
		}
	}

	validIP, ok := ValidateIP(ip)
	if !ok {
		WriteError(w, http.StatusBadRequest, CodeInvalidParam, "IP地址格式无效")
		return
	}

	result := h.engine.QueryIP(validIP)
	if result.IPLocate == nil {
		WriteError(w, http.StatusInternalServerError, CodeQueryFailed, "查询IP位置信息失败")
		return
	}

	data := ConvertIPLocate(result.IPLocate)
	WriteSuccess(w, data)
}
