package queryip

import (
	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iplocate"
)

// IPDbConfig 存储程序配置
type IPDbConfig struct {
	AsnIpvxDb    string
	AsnIpv4Db    string
	AsnIpv6Db    string
	IpvxLocateDb string
	Ipv4LocateDb string
	Ipv6LocateDb string
}

// DBEngine 存储所有数据库引擎实例
type DBEngine struct {
	AsnIPv4Engine asninfo.ASNQuerier
	AsnIPv6Engine asninfo.ASNQuerier
	IPv4Engine    iplocate.IPInfo
	IPv6Engine    iplocate.IPInfo
}

// IPLocation IP地理位置查询结果
type IPLocation struct {
	IP       string
	IPLocate *iplocate.IPLocate
}

// IPQueryResult 单个IP的完整查询结果
type IPQueryResult struct {
	IP       string
	IPLocate *iplocate.IPLocate
	ASNInfo  *asninfo.ASNInfo
}

// IPDbInfo 存储IP解析的中间结果
type IPDbInfo struct {
	IPv4Locations []IPLocation
	IPv6Locations []IPLocation
	IPv4AsnInfos  []asninfo.ASNInfo
	IPv6AsnInfos  []asninfo.ASNInfo
}
