package iplocate

import (
	"errors"
	"strings"
)

// IP版本常量
const (
	IPv4VersionNo = 4
	IPv6VersionNo = 6
)

// 统一错误类型
var (
	ErrInvalidIP       = errors.New("无效的IP地址")
	ErrDatabaseNotInit = errors.New("数据库未初始化")
	ErrQueryFailed     = errors.New("查询失败")
	ErrInvalidDBFile   = errors.New("数据库文件无效")
)

// DBType 数据库类型
type DBType string

const (
	DBTypeIP2Region   DBType = "ip2region"
	DBTypeDBIPMMDB    DBType = "dbip_mmdb"
	DBTypeGeoLiteMMDB DBType = "geolite2_mmdb"
	DBTypeIPDB        DBType = "ipdb"
	DBTypeQQWry       DBType = "qqwry"
	DBTypeZXWry       DBType = "zxwry"
)

// DBInfo 数据库信息结构体
type DBInfo struct {
	// 数据库类型
	Type DBType `json:"type" yaml:"type"`
	// 数据库文件路径
	DbPath string `json:"db_path" yaml:"db_path"`
	// 是否支持IPv4
	IsIPv4 bool `json:"is_ipv4,omitempty" yaml:"is_ipv4,omitempty"`
	// 是否支持IPv6
	IsIPv6 bool `json:"is_ipv6,omitempty" yaml:"is_ipv6,omitempty"`
}

// IPLocate 轻量IP定位结构体
type IPLocate struct {
	IP       string `json:"ip"`       //原始IP
	Version  int    `json:"version"`  //IP版本
	Location string `json:"location"` //原始定位结果字符串
	Country  string `json:"country"`  //国家
	Province string `json:"province"` //省份
	City     string `json:"city"`     //城市
	Area     string `json:"area"`     //区县
	Street   string `json:"street"`   //街道
	ISP      string `json:"isp"`      //网络服务提供者
}

// HasData 判断是否包含实际定位信息，不把只有IP和版本的空结果视为命中。
func (r *IPLocate) HasData() bool {
	if r == nil {
		return false
	}
	for _, value := range []string{r.Location, r.Country, r.Province, r.City, r.Area, r.Street, r.ISP} {
		if strings.Trim(value, " |0\t\r\n") != "" {
			return true
		}
	}
	return false
}
