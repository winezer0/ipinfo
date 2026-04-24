package asninfo

import "net"

// ASNQuerier ASN数据库查询接口
// 实现此接口可以支持不同的ASN数据库格式（如MMDB、CSV等）
type ASNQuerier interface {
	// Init 初始化数据库连接
	Init() error

	// Close 关闭数据库连接
	Close() error

	// IsInitialized 检查数据库是否已初始化
	IsInitialized() bool

	// FindASN 查询单个IP的ASN信息
	FindASN(ipStr string) *ASNInfo

	// BatchFindASN 批量查询多个IP的ASN信息
	BatchFindASN(ips []string) map[string]*ASNInfo

	// ASNToIPRanges 通过ASN号反查所有IP段
	ASNToIPRanges(targetASN uint64) ([]*net.IPNet, error)

	// GetDatabaseInfo 获取数据库信息
	GetDatabaseInfo() *DBInfo
}
