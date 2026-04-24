package ipdb

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/winezer0/ipinfo/pkg/iplocate"

	"github.com/ipipdotnet/ipdb-go"
)

// IPDB 实现了 IPDB 格式的 IP 信息查询接口
type IPDB struct {
	city   *ipdb.City
	mu     sync.RWMutex
	dbPath string
}

// 确保 IPDB 实现了 iplocate.IPInfo 接口
var _ iplocate.IPInfo = (*IPDB)(nil)

// NewIPDB 创建一个新的 IPDB 实例
func NewIPDB(dbPath string) (*IPDB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("IPDB数据库文件路径为空")
	}

	db := &IPDB{
		dbPath: dbPath,
	}

	if err := db.Init(); err != nil {
		return nil, err
	}

	return db, nil
}

// Init 初始化数据库连接
func (db *IPDB) Init() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.city != nil {
		return nil
	}

	city, err := ipdb.NewCity(db.dbPath)
	if err != nil {
		return fmt.Errorf("打开IPDB数据库失败: %w", err)
	}

	db.city = city
	return nil
}

// IsInitialized 检查数据库是否已初始化
func (db *IPDB) IsInitialized() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.city != nil
}

// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
func (db *IPDB) FindFull(query string) *iplocate.IPLocate {
	db.mu.RLock()
	defer db.mu.RUnlock()

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

	info, err := db.city.FindInfo(query, "CN")
	if err != nil {
		return result
	}

	if info == nil {
		return result
	}

	result.Country = info.CountryName
	result.Province = info.RegionName
	result.City = info.CityName
	result.ISP = info.IspDomain
	result.Location = formatCityInfo(info)

	return result
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (db *IPDB) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	db.mu.RLock()
	defer db.mu.RUnlock()

	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = db.findFullInternal(query)
	}

	return results
}

// findFullInternal 内部查询方法，不加锁
func (db *IPDB) findFullInternal(query string) *iplocate.IPLocate {
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

	info, err := db.city.FindInfo(query, "CN")
	if err != nil {
		return result
	}

	if info == nil {
		return result
	}

	result.Country = info.CountryName
	result.City = info.RegionName
	result.Area = info.CityName
	result.ISP = info.IspDomain

	return result
}

// GetDatabaseInfo 获取数据库信息
func (db *IPDB) GetDatabaseInfo() *iplocate.DBInfo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	info := &iplocate.DBInfo{
		Type:   iplocate.DBTypeIPDB,
		DbPath: db.dbPath,
	}

	if db.city != nil {
		info.IsIPv4 = db.city.IsIPv4()
		info.IsIPv6 = db.city.IsIPv6()
	}

	return info
}

// Close 关闭数据库连接
func (db *IPDB) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.city = nil
}

// formatCityInfo 格式化城市信息
func formatCityInfo(info *ipdb.CityInfo) string {
	if info == nil {
		return ""
	}

	var parts []string

	if info.CountryName != "" {
		parts = append(parts, info.CountryName)
	}
	if info.RegionName != "" {
		parts = append(parts, info.RegionName)
	}
	if info.CityName != "" {
		parts = append(parts, info.CityName)
	}
	if info.IspDomain != "" {
		parts = append(parts, info.IspDomain)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}
