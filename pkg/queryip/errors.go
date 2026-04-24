package queryip

import "errors"

var (
	ErrInitASNFailed        = errors.New("初始化ASN数据库失败")
	ErrInitIPFailed         = errors.New("初始化IP数据库失败")
	ErrInitIP2RegionFailed  = errors.New("初始化IP2Region数据库失败")
	ErrOpenMMDBFailed       = errors.New("打开MMDB数据库失败")
	ErrInitDBIPFailed       = errors.New("初始化DBIP数据库失败")
	ErrInitGeoLite2Failed   = errors.New("初始化GeoLite2数据库失败")
	ErrInitIPDBFailed       = errors.New("初始化IPDB数据库失败")
	ErrInitQQWryFailed      = errors.New("初始化QQWry数据库失败")
	ErrInitZXWryFailed      = errors.New("初始化ZXWry数据库失败")
	ErrUnsupportedASNFormat = errors.New("不支持的ASN数据库格式")
	ErrUnsupportedIPFormat  = errors.New("不支持的IP数据库格式")
	ErrMissingIPv4Config    = errors.New("未配置IPv4地理位置数据库")
	ErrMissingIPv6Config    = errors.New("未配置IPv6地理位置数据库")
	ErrCloseDBFailed        = errors.New("关闭数据库连接失败")
	ErrInvalidIPAddress     = errors.New("无效的IP地址")
)
