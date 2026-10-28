package webapi

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type trustedProxyKey struct{}

// TrustedProxyMiddleware 为当前路由配置可信代理；默认不信任转发头。
func TrustedProxyMiddleware(cidrs []string) func(http.Handler) http.Handler {
	var networks []*net.IPNet
	for _, cidr := range cidrs {
		if _, network, err := net.ParseCIDR(cidr); err == nil {
			networks = append(networks, network)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), trustedProxyKey{}, networks)))
		})
	}
}

func isTrustedProxy(r *http.Request, ip string) bool {
	networks, _ := r.Context().Value(trustedProxyKey{}).([]*net.IPNet)
	for _, network := range networks {
		if network.Contains(net.ParseIP(ip)) {
			return true
		}
	}
	return false
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, _ := ValidateIP(host)
	return ip
}

// forwardedChain 只保留从可信代理向左追溯到的第一个非可信地址及其右侧代理。
func forwardedChain(r *http.Request) []string {
	if !isTrustedProxy(r, remoteIP(r)) {
		return nil
	}
	raw := r.Header.Get("X-Forwarded-For")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	start := len(parts) - 1
	for i := len(parts) - 1; i >= 0; i-- {
		ip, ok := ValidateIP(parts[i])
		if !ok {
			return nil
		}
		parts[i] = ip
		start = i
		if !isTrustedProxy(r, ip) {
			break
		}
	}
	return parts[start:]
}

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
	remote := remoteIP(r)
	if !isTrustedProxy(r, remote) {
		return remote
	}
	if r.Header.Get("X-Forwarded-For") != "" {
		if chain := forwardedChain(r); len(chain) > 0 {
			return chain[0]
		}
		return remote // 有损坏的代理链时，不回退到其它可伪造的头。
	}
	for _, header := range []string{"X-Real-IP", "CF-Connecting-IP"} {
		if ip, ok := ValidateIP(r.Header.Get(header)); ok {
			return ip
		}
	}
	return remote
}

// GetClientIpList 获取客户端所有相关IP信息
func GetClientIpList(r *http.Request) ClientIPInfo {
	info := ClientIPInfo{RemoteAddr: remoteIP(r)}
	if !isTrustedProxy(r, info.RemoteAddr) {
		return info
	}
	if r.Header.Get("X-Forwarded-For") != "" {
		info.XForwardedFor = forwardedChain(r)
		return info
	}
	if ip, ok := ValidateIP(r.Header.Get("X-Real-IP")); ok {
		info.XRealIP = ip
	} else if ip, ok := ValidateIP(r.Header.Get("CF-Connecting-IP")); ok {
		info.CFConnectingIP = ip
	}
	return info
}
