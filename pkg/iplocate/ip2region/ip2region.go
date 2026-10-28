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
	Version   int
	Searcher  *xdb.Searcher
	mu        sync.RWMutex
	DbPath    string
	poolSize  int
	searchers []*xdb.Searcher
	pool      chan *xdb.Searcher
	serial    sync.Mutex
}

// MaxPoolSize limits the number of open database handles per engine.
const MaxPoolSize = 64

// 确保 IP2Region 实现了 ipinfo.IPInfo 接口
var _ iplocate.IPInfo = (*IP2Region)(nil)

// NewIP2Region 创建一个新的IP2Region实例
// version: iplocate.IPv4VersionNo 或 iplocate.IPv6VersionNo
// dbPath: 对应的xdb文件路径
func NewIP2Region(version int, dbPath string) (*IP2Region, error) {
	return NewIP2RegionWithPool(version, dbPath, 1)
}

// NewIP2RegionWithPool shares one immutable VectorIndex across independent Searchers.
func NewIP2RegionWithPool(version int, dbPath string, size int) (*IP2Region, error) {
	if size < 1 || size > MaxPoolSize {
		return nil, fmt.Errorf("XDB pool size must be between 1 and %d", MaxPoolSize)
	}
	ipr := &IP2Region{
		Version:  version,
		DbPath:   dbPath,
		poolSize: size,
	}

	if err := ipr.Init(); err != nil {
		return nil, err
	}

	return ipr, nil
}

// Init 初始化数据库连接
func (ipr *IP2Region) Init() error {
	return ipr.initWithVector(nil, xdb.NewWithVectorIndex)
}

// initWithVector publishes the pool only after every handle opens successfully.
func (ipr *IP2Region) initWithVector(vectorIndex []byte, open func(*xdb.Version, string, []byte) (*xdb.Searcher, error)) error {
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

	if vectorIndex == nil {
		vectorIndex, err = xdb.LoadVectorIndexFromFile(ipr.DbPath)
		if err != nil {
			return fmt.Errorf("加载VectorIndex失败: %w", err)
		}
	}
	if len(vectorIndex) != xdb.VectorIndexRows*xdb.VectorIndexCols*xdb.VectorIndexSize {
		return fmt.Errorf("invalid VectorIndex length: %d", len(vectorIndex))
	}
	size := ipr.poolSize
	if size == 0 {
		size = 1
	}
	if size < 1 || size > MaxPoolSize {
		return fmt.Errorf("invalid XDB pool size: %d", size)
	}
	searchers := make([]*xdb.Searcher, 0, size)
	for i := 0; i < size; i++ {
		searcher, err := open(xdbVersion, ipr.DbPath, vectorIndex)
		if err != nil {
			for _, opened := range searchers {
				opened.Close()
			}
			return fmt.Errorf("创建searcher %d/%d 失败: %w", i+1, size, err)
		}
		searchers = append(searchers, searcher)
	}
	ipr.searchers = searchers
	ipr.Searcher = searchers[0]
	if size > 1 {
		ipr.pool = make(chan *xdb.Searcher, size)
		for _, searcher := range searchers {
			ipr.pool <- searcher
		}
	}
	return nil
}

// NewIP2RegionAuto 从XDB文件头识别IP版本并创建查询器。
func NewIP2RegionAuto(dbPath string) (*IP2Region, error) {
	return NewIP2RegionAutoWithPool(dbPath, 1)
}

func NewIP2RegionAutoWithPool(dbPath string, size int) (*IP2Region, error) {
	header, err := xdb.LoadHeaderFromFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("读取XDB文件头失败: %w", err)
	}
	version, err := xdb.VersionFromHeader(header)
	if err != nil {
		return nil, err
	}
	return NewIP2RegionWithPool(version.Id, dbPath, size)
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
	ipr := &IP2Region{Version: version, DbPath: dbPath, poolSize: 1}
	if err := ipr.initWithVector(vectorIndex, xdb.NewWithVectorIndex); err != nil {
		return nil, err
	}
	return ipr, nil
}

// PoolSize returns the number of currently open Searchers. Searcher is retained
// for source compatibility; callers must not use it concurrently with this API.
func (ipr *IP2Region) PoolSize() int {
	ipr.mu.RLock()
	defer ipr.mu.RUnlock()
	return len(ipr.searchers)
}

// acquire/release require a lifecycle read lock, held until the lease is returned.
func (ipr *IP2Region) acquire() *xdb.Searcher {
	if ipr.pool != nil {
		return <-ipr.pool
	}
	ipr.serial.Lock()
	return ipr.Searcher
}

func (ipr *IP2Region) release(searcher *xdb.Searcher) {
	if ipr.pool != nil {
		ipr.pool <- searcher
		return
	}
	ipr.serial.Unlock()
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
	searcher := ipr.acquire()
	defer ipr.release(searcher)
	return findFull(searcher, query)
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (ipr *IP2Region) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	ipr.mu.RLock()
	defer ipr.mu.RUnlock()
	searcher := ipr.acquire()
	defer ipr.release(searcher)

	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = findFull(searcher, query)
	}

	return results
}

func findFull(searcher *xdb.Searcher, query string) *iplocate.IPLocate {
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

	if searcher == nil {
		return result
	}

	region, err := searcher.SearchByStr(query)
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

	if len(ipr.searchers) > 0 {
		for _, searcher := range ipr.searchers {
			searcher.Close()
		}
	} else if ipr.Searcher != nil {
		ipr.Searcher.Close()
	}
	ipr.Searcher = nil
	ipr.searchers = nil
	ipr.pool = nil
}

// LoadVectorIndexFromFile 从文件加载VectorIndex缓存
func LoadVectorIndexFromFile(dbPath string) ([]byte, error) {
	return xdb.LoadVectorIndexFromFile(dbPath)
}
