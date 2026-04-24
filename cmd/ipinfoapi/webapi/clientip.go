package webapi

import (
	"net"
	"net/http"
	"strings"
)

// ClientIPInfo 客户端IP信息结构体
type ClientIPInfo struct {
	RemoteAddr     string   `json:"remote_addr"`
	XForwardedFor  []string `json:"x_forwarded_for,omitempty"`
	XRealIP        string   `json:"x_real_ip,omitempty"`
	Forwarded      string   `json:"forwarded,omitempty"`
	CFConnectingIP string   `json:"cf_connecting_ip,omitempty"`
}

// GetClientIP 从HTTP请求中获取客户端真实IP
func GetClientIP(r *http.Request) string {
	// 优先检查 X-Forwarded-For 头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			firstIP := strings.TrimSpace(ips[0])
			if ip, ok := ValidateIP(firstIP); ok {
				return ip
			}
		}
	}

	// 检查 X-Real-IP 头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip, ok := ValidateIP(xri); ok {
			return ip
		}
	}

	// 从 RemoteAddr 获取
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if ip, ok := ValidateIP(host); ok {
			return ip
		}
	}

	return ""
}

// GetClientIpList 获取客户端所有相关IP信息
func GetClientIpList(r *http.Request) ClientIPInfo {
	info := ClientIPInfo{}

	// 获取 RemoteAddr
	info.RemoteAddr = r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		info.RemoteAddr = host
	}

	// 解析 X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if _, ok := ValidateIP(ip); ok {
				info.XForwardedFor = append(info.XForwardedFor, ip)
			}
		}
	}

	// 获取 X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip, ok := ValidateIP(xri); ok {
			info.XRealIP = ip
		}
	}

	// 获取 Forwarded 头 (RFC 7239)
	info.Forwarded = r.Header.Get("Forwarded")

	// 获取 Cloudflare 的 CF-Connecting-IP
	info.CFConnectingIP = r.Header.Get("CF-Connecting-IP")

	return info
}
