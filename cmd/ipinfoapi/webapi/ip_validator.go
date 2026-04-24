package webapi

import (
	"net"
	"strings"
)

// ValidateIP 校验IP地址格式，支持IPv4和IPv6
func ValidateIP(ip string) (string, bool) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", false
	}

	return parsedIP.String(), true
}
