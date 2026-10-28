package webapi

import "net/http"

// HandleASN 处理ASN查询请求
func (h *APIHandler) HandleASN(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.engine.QueryASNContext(r.Context(), validIP)
	if r.Context().Err() != nil {
		return
	}
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, CodeQueryFailed, "查询ASN数据库失败")
		return
	}
	if result == nil {
		WriteError(w, http.StatusInternalServerError, CodeQueryFailed, "查询ASN信息失败")
		return
	}

	data := ConvertASN(result)
	WriteSuccess(w, data)
}
