package asninfo

import "errors"

// 错误类型定义
var (
	ErrDatabaseNotInitialized = errors.New("数据库未初始化")
	ErrEmptyDatabasePath      = errors.New("数据库路径为空")
	ErrDatabaseFileNotFound   = errors.New("数据库文件不存在")
	ErrFailedToOpenDatabase   = errors.New("打开数据库失败")
	ErrFailedToCloseDatabase  = errors.New("关闭数据库失败")
	ErrFailedToReadDatabase   = errors.New("读取数据库失败")
	ErrInvalidIPAddress       = errors.New("无效的IP地址")
	ErrFailedToParseNetwork   = errors.New("解析网络段失败")
	ErrFailedToIterateNetwork = errors.New("遍历数据库时发生错误")
)

// ASNInfo ASN查询结果结构体
type ASNInfo struct {
	IP                 string `json:"ip"`
	IPVersion          int    `json:"ip_version"`
	FoundASN           bool   `json:"found_asn"`
	OrganisationNumber uint64 `json:"as_number"`
	OrganisationName   string `json:"as_organisation"`
}

// NewASNInfo 创建新的ASNInfo实例
func NewASNInfo(ipString string, ipVersion int) *ASNInfo {
	return &ASNInfo{
		ipString,
		ipVersion,
		false,
		0,
		""}
}

// DBType 数据库类型
type DBType string

const (
	DBTypeMMDB DBType = "mmdb"
	DBTypeCSV  DBType = "csv"
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
