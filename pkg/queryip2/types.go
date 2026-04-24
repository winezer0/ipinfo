package queryip2

import (
	"github.com/winezer0/ipinfo/pkg/iplocate"
)

// Source 数据库来源标识（使用数据库文件名）
type Source string

// IPQueryResult 单个IP的完整查询结果（多数据库）
type IPQueryResult struct {
	IP             string                        `json:"ip"`             // 原始IP
	IPLocateResult map[Source]*iplocate.IPLocate `json:"ipLocateResult"` // 所有数据库的定位结果
}

// IPDbInfo 存储IP解析的中间结果（多数据库）
type IPDbInfo struct {
	IPv4Results []IPQueryResult `json:"ipv4Results"` // IPv4查询结果
	IPv6Results []IPQueryResult `json:"ipv6Results"` // IPv6查询结果
}
