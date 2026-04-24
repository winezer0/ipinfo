package ip2region

import (
	"fmt"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"net"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// IP2Region 实现了统一的IP信息查询接口
type IP2Region struct {
	Version  int
	Searcher *xdb.Searcher
	mu       sync.RWMutex
	DbPath   string
}

// 确保 IP2Region 实现了 ipinfo.IPInfo 接口
var _ iplocate.IPInfo = (*IP2Region)(nil)

// NewIP2Region 创建一个新的IP2Region实例
// version: iplocate.IPv4VersionNo 或 iplocate.IPv6VersionNo
// dbPath: 对应的xdb文件路径
func NewIP2Region(version int, dbPath string) (*IP2Region, error) {
	ipr := &IP2Region{
		Version: version,
		DbPath:  dbPath,
	}

	if err := ipr.Init(); err != nil {
		return nil, err
	}

	return ipr, nil
}

// Init 初始化数据库连接
func (ipr *IP2Region) Init() error {
	ipr.mu.Lock()
	defer ipr.mu.Unlock()

	if ipr.Searcher != nil {
		return nil
	}

	if err := xdb.VerifyFromFile(ipr.DbPath); err != nil {
		return fmt.Errorf("xdb文件验证失败: %w", err)
	}

	xdbVersion, err := getXdbVersion(ipr.Version)
	if err != nil {
		return err
	}

	searcher, err := xdb.NewWithFileOnly(xdbVersion, ipr.DbPath)
	if err != nil {
		return fmt.Errorf("创建searcher失败: %w", err)
	}

	ipr.Searcher = searcher
	return nil
}

// IsInitialized 检查数据库是否已初始化
func (ipr *IP2Region) IsInitialized() bool {
	ipr.mu.RLock()
	defer ipr.mu.RUnlock()

	return ipr.Searcher != nil
}

// NewIP2RegionWithVectorIndex 创建一个使用VectorIndex缓存的IP2Region实例
// version: iplocate.IPv4VersionNo 或 iplocate.IPv6VersionNo
// dbPath: 对应的xdb文件路径
// vectorIndex: 预加载的VectorIndex缓存
func NewIP2RegionWithVectorIndex(version int, dbPath string, vectorIndex []byte) (*IP2Region, error) {
	// 验证xdb文件
	if err := xdb.VerifyFromFile(dbPath); err != nil {
		return nil, fmt.Errorf("xdb文件验证失败: %w", err)
	}

	// 根据版本号获取对应的Version对象
	xdbVersion, err := getXdbVersion(version)
	if err != nil {
		return nil, err
	}

	// 创建searcher
	searcher, err := xdb.NewWithVectorIndex(xdbVersion, dbPath, vectorIndex)
	if err != nil {
		return nil, fmt.Errorf("创建searcher失败: %w", err)
	}

	return &IP2Region{
		Version:  version,
		DbPath:   dbPath,
		Searcher: searcher,
	}, nil
}

// getXdbVersion 根据整型版本号获取xdb.Version对象
func getXdbVersion(version int) (*xdb.Version, error) {
	switch version {
	case iplocate.IPv4VersionNo:
		return xdb.IPv4, nil
	case iplocate.IPv6VersionNo:
		return xdb.IPv6, nil
	default:
		return nil, fmt.Errorf("不支持的IP版本: %d", version)
	}
}

// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
func (ipr *IP2Region) FindFull(query string) *iplocate.IPLocate {
	ipr.mu.RLock()
	defer ipr.mu.RUnlock()

	result := &iplocate.IPLocate{
		IP: query,
	}

	ip := net.ParseIP(query)
	if ip == nil {
		return result
	}

	if ip.To4() != nil {
		result.Version = 4
	} else {
		result.Version = 6
	}

	if ipr.Searcher == nil {
		return result
	}

	region, err := ipr.Searcher.SearchByStr(query)
	if err != nil {
		return result
	}

	return parseRegionToIPLocate(result, region)
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (ipr *IP2Region) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	ipr.mu.RLock()
	defer ipr.mu.RUnlock()

	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = ipr.findFullInternal(query)
	}

	return results
}

// findFullInternal 内部查询方法，不加锁
func (ipr *IP2Region) findFullInternal(query string) *iplocate.IPLocate {
	result := &iplocate.IPLocate{
		IP: query,
	}

	ip := net.ParseIP(query)
	if ip == nil {
		return result
	}

	if ip.To4() != nil {
		result.Version = 4
	} else {
		result.Version = 6
	}

	if ipr.Searcher == nil {
		return result
	}

	region, err := ipr.Searcher.SearchByStr(query)
	if err != nil {
		return result
	}

	return parseRegionToIPLocate(result, region)
}

// parseRegionToIPLocate 解析ip2region的格式为IPLocate
// ip2region格式: 国家|区域|省份|城市|ISP
func parseRegionToIPLocate(result *iplocate.IPLocate, region string) *iplocate.IPLocate {
	if region == "" {
		return result
	}

	result.Location = region

	parts := strings.Split(region, "|")

	if len(parts) > 0 && parts[0] != "" {
		result.Country = parts[0]
	}
	if len(parts) > 1 && parts[1] != "" {
		result.Province = parts[1]
	}
	if len(parts) > 2 && parts[2] != "" && parts[2] != "0" {
		result.City = parts[2]
	}
	if len(parts) > 3 && parts[3] != "" && parts[3] != "0" {
		result.ISP = parts[3]
	}

	return result
}

// GetDatabaseInfo 获取数据库信息
func (ipr *IP2Region) GetDatabaseInfo() *iplocate.DBInfo {
	ipr.mu.RLock()
	defer ipr.mu.RUnlock()

	return &iplocate.DBInfo{
		Type:   iplocate.DBTypeIP2Region,
		DbPath: ipr.DbPath,
		IsIPv4: ipr.Version == iplocate.IPv4VersionNo,
		IsIPv6: ipr.Version == iplocate.IPv6VersionNo,
	}
}

// Close 关闭数据库连接（清理资源）
func (ipr *IP2Region) Close() {
	ipr.mu.Lock()
	defer ipr.mu.Unlock()

	if ipr.Searcher != nil {
		ipr.Searcher.Close()
		ipr.Searcher = nil
	}
}

// LoadVectorIndexFromFile 从文件加载VectorIndex缓存
func LoadVectorIndexFromFile(dbPath string) ([]byte, error) {
	return xdb.LoadVectorIndexFromFile(dbPath)
}
