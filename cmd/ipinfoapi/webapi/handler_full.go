package webapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// HandleFull 处理批量IP查询请求
func (h *APIHandler) HandleFull(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if r.Context().Err() != nil {
		return
	}
	limit := h.maxBatchSize
	if limit <= 0 {
		limit = 100
	}
	if q != "" && strings.Count(q, ",")+1 > limit {
		WriteError(w, http.StatusBadRequest, CodeInvalidParam, fmt.Sprintf("批量查询最多允许%d个IP", limit))
		return
	}

	var ips []string
	var clientIPs *ClientIPInfo

	if q == "" {
		// q为空时，自动获取客户端所有IP
		clientIPsInfo := GetClientIpList(r)
		clientIPs = &clientIPsInfo

		// 收集所有有效IP
		if clientIPsInfo.XRealIP != "" {
			ips = append(ips, clientIPsInfo.XRealIP)
		}
		if clientIPsInfo.CFConnectingIP != "" {
			ips = append(ips, clientIPsInfo.CFConnectingIP)
		}
		ips = append(ips, clientIPsInfo.XForwardedFor...)
		if clientIPsInfo.RemoteAddr != "" {
			ips = append(ips, clientIPsInfo.RemoteAddr)
		}
	} else {
		// 解析逗号分隔的IP列表
		parts := strings.Split(q, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip != "" {
				ips = append(ips, ip)
			}
		}
	}

	if len(ips) == 0 {
		WriteError(w, http.StatusBadRequest, CodeInvalidParam, "未提供有效的IP地址")
		return
	}
	if len(ips) > limit {
		WriteError(w, http.StatusBadRequest, CodeInvalidParam, fmt.Sprintf("批量查询最多允许%d个IP", limit))
		return
	}

	// 去重处理
	ips = uniqueStrings(ips)

	// 批量查询IP信息
	results, successCount, failedCount, err := h.batchQueryIPs(r.Context(), ips)
	if err != nil {
		return
	}

	data := FullData{
		Total:   len(ips),
		Success: successCount,
		Failed:  failedCount,
		Results: results,
	}

	// 如果是自动获取客户端IP，附加客户端IP信息
	if clientIPs != nil {
		data.ClientIPs = clientIPs
	}

	WriteSuccess(w, data)
}

// batchQueryIPs 批量查询IP信息
func (h *APIHandler) batchQueryIPs(ctx context.Context, ips []string) ([]FullResultItem, int, int, error) {
	var results []FullResultItem
	successCount := 0
	failedCount := 0

	for _, ip := range ips {
		if err := ctx.Err(); err != nil {
			return results, successCount, failedCount, err
		}
		item := FullResultItem{IP: ip}

		validIP, ok := ValidateIP(ip)
		if !ok {
			item.Error = "IP地址格式无效"
			failedCount++
			results = append(results, item)
			continue
		}

		result, err := h.engine.QueryIPContext(ctx, validIP)
		if ctx.Err() != nil {
			return results, successCount, failedCount, ctx.Err()
		}
		if err != nil {
			item.Error = "查询数据库失败"
			failedCount++
			results = append(results, item)
			continue
		}
		item.IP = validIP
		item.Location = ConvertIPLocate(result.IPLocate)
		item.ASN = ConvertASN(result.ASNInfo)

		if result.IPLocate.HasData() || (result.ASNInfo != nil && result.ASNInfo.FoundASN) {
			successCount++
		} else {
			failedCount++
			item.Error = "未找到该IP的数据"
		}

		results = append(results, item)
	}

	return results, successCount, failedCount, nil
}

// uniqueStrings 字符串切片去重
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
