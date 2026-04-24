package webapi

import "net/http"

// HandleAll 处理完整查询请求
func (h *APIHandler) HandleAll(w http.ResponseWriter, r *http.Request) {
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

	data := AllData{
		IP:       validIP,
		Location: ConvertIPLocate(result.IPLocate),
		ASN:      ConvertASN(result.ASNInfo),
	}

	WriteSuccess(w, data)
}
