package iputils

// IPClassResult IP分类结果
type IPClassResult struct {
	IPv4s []string // IPv4地址列表
	IPv6s []string // IPv6地址列表
	Other []string // 其他（无效IP）列表
}

// ClassifyIPs 将输入的IP列表分类为IPv4、IPv6和其他（无效）三类
// 参数:
//
//	ips - 待分类的IP字符串切片
//
// 返回:
//
//	*IPClassResult - 分类结果
func ClassifyIPs(ips []string) *IPClassResult {
	result := &IPClassResult{
		IPv4s: make([]string, 0),
		IPv6s: make([]string, 0),
		Other: make([]string, 0),
	}

	for _, ip := range ips {
		parsed := GetIpVersion(ip)
		if parsed == 4 {
			result.IPv4s = append(result.IPv4s, ip)
		} else if parsed == 6 {
			result.IPv6s = append(result.IPv6s, ip)
		} else {
			result.Other = append(result.Other, ip)
		}
	}
	return result
}
