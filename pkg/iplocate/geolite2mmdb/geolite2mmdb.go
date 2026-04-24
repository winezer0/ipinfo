// Package geolite2mmdb 提供对 GeoLite2 格式 MMDB 数据库的支持
package geolite2mmdb

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/winezer0/ipinfo/pkg/iplocate"

	"github.com/oschwald/maxminddb-golang"
)

// GeoLite2MMDB GeoLite2格式MMDB数据库结构体
type GeoLite2MMDB struct {
	db     *maxminddb.Reader
	mu     sync.RWMutex
	dbPath string
}

var _ iplocate.IPInfo = (*GeoLite2MMDB)(nil)

// NewGeoLite2MMDB 创建GeoLite2格式MMDB数据库实例
func NewGeoLite2MMDB(dbPath string) (*GeoLite2MMDB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("GeoLite2 MMDB数据库文件路径为空")
	}

	m := &GeoLite2MMDB{
		dbPath: dbPath,
	}

	if err := m.Init(); err != nil {
		return nil, err
	}

	return m, nil
}

// Init 初始化数据库连接
func (m *GeoLite2MMDB) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		return nil
	}

	db, err := maxminddb.Open(m.dbPath)
	if err != nil {
		return fmt.Errorf("打开GeoLite2 MMDB数据库失败: %w", err)
	}

	m.db = db
	return nil
}

// IsInitialized 检查数据库是否已初始化
func (m *GeoLite2MMDB) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.db != nil
}

// GeoRecord 地理位置记录结构体，适用于GeoLite2格式
type GeoRecord struct {
	CountryCode string  `maxminddb:"country_code"`
	City        string  `maxminddb:"city"`
	State1      string  `maxminddb:"state1"`
	State2      string  `maxminddb:"state2"`
	Latitude    float32 `maxminddb:"latitude"`
	Longitude   float32 `maxminddb:"longitude"`
	Timezone    string  `maxminddb:"timezone"`
	Postcode    string  `maxminddb:"postcode"`
}

// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
func (m *GeoLite2MMDB) FindFull(query string) *iplocate.IPLocate {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

	var record GeoRecord
	err := m.db.Lookup(ip, &record)
	if err != nil {
		return result
	}

	result.Country = record.CountryCode
	result.Location = formatGeoRecord(record)

	if record.State1 != "" {
		result.Province = record.State1
	}

	if record.City != "" {
		result.City = record.City
	}

	return result
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (m *GeoLite2MMDB) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = m.findFullInternal(query)
	}

	return results
}

// findFullInternal 内部查询方法，不加锁
func (m *GeoLite2MMDB) findFullInternal(query string) *iplocate.IPLocate {
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

	var record GeoRecord
	err := m.db.Lookup(ip, &record)
	if err != nil {
		return result
	}

	result.Country = record.CountryCode

	if record.State1 != "" {
		result.City = record.State1
	}

	if record.City != "" {
		if result.City != "" {
			result.Area = record.City
		} else {
			result.City = record.City
		}
	}

	return result
}

// GetDatabaseInfo 获取数据库信息
func (m *GeoLite2MMDB) GetDatabaseInfo() *iplocate.DBInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := &iplocate.DBInfo{
		Type:   iplocate.DBTypeGeoLiteMMDB,
		DbPath: m.dbPath,
	}

	if m.db != nil {
		info.IsIPv4 = m.db.Metadata.IPVersion == 4
		info.IsIPv6 = m.db.Metadata.IPVersion == 6
	}

	return info
}

// Close 关闭数据库连接
func (m *GeoLite2MMDB) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		m.db.Close()
		m.db = nil
	}
}

// formatGeoRecord 格式化地理位置记录
func formatGeoRecord(record GeoRecord) string {
	var parts []string

	if record.CountryCode != "" {
		parts = append(parts, record.CountryCode)
	}

	if record.State1 != "" {
		parts = append(parts, record.State1)
	}

	if record.State2 != "" {
		parts = append(parts, record.State2)
	}

	if record.City != "" {
		parts = append(parts, record.City)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}
