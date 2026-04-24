package queryip2

import "errors"

var (
	ErrInitIPFailed        = errors.New("初始化IP数据库失败")
	ErrInitIP2RegionFailed = errors.New("初始化IP2Region数据库失败")
	ErrOpenMMDBFailed      = errors.New("打开MMDB数据库失败")
	ErrInitDBIPFailed      = errors.New("初始化DBIP数据库失败")
	ErrInitGeoLite2Failed  = errors.New("初始化GeoLite2数据库失败")
	ErrInitIPDBFailed      = errors.New("初始化IPDB数据库失败")
	ErrInitQQWryFailed     = errors.New("初始化QQWry数据库失败")
	ErrUnsupportedIPFormat = errors.New("不支持的IP数据库格式")
	ErrMissingIPConfig     = errors.New("未配置IP地理位置数据库")
)
