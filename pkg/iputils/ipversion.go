package iputils

import "net"

// GetIpVersion 获取IP版本号
func GetIpVersion(ipString string) int {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return 0
	}
	if ip.To4() != nil {
		return 4
	}
	return 6
}

// IsIPv4 判断是否为IPv4地址
func IsIPv4(ip string) bool {
	return GetIpVersion(ip) == 4
}
