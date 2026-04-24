// Package dbipmmdb 提供对 DBIP 格式 MMDB 数据库的支持
package dbipmmdb

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/winezer0/ipinfo/pkg/iplocate"

	"github.com/oschwald/maxminddb-golang"
)

// DBIPMMDB DBIP格式MMDB数据库结构体
type DBIPMMDB struct {
	db     *maxminddb.Reader
	mu     sync.RWMutex
	dbPath string
}

var _ iplocate.IPInfo = (*DBIPMMDB)(nil)

// NewDBIPMMDB 创建DBIP格式MMDB数据库实例
func NewDBIPMMDB(dbPath string) (*DBIPMMDB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("DBIP MMDB数据库文件路径为空")
	}

	m := &DBIPMMDB{
		dbPath: dbPath,
	}

	if err := m.Init(); err != nil {
		return nil, err
	}

	return m, nil
}

// Init 初始化数据库连接
func (m *DBIPMMDB) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		return nil
	}

	db, err := maxminddb.Open(m.dbPath)
	if err != nil {
		return fmt.Errorf("打开DBIP MMDB数据库失败: %w", err)
	}

	m.db = db
	return nil
}

// IsInitialized 检查数据库是否已初始化
func (m *DBIPMMDB) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.db != nil
}

// CountryRecord 国家记录结构体
type CountryRecord struct {
	IsoCode   string            `maxminddb:"iso_code"`
	Names     map[string]string `maxminddb:"names"`
	GeoNameID uint              `maxminddb:"geoname_id"`
}

// CityRecord 城市记录结构体
type CityRecord struct {
	Names map[string]string `maxminddb:"names"`
	Name  string            `maxminddb:"name"`
}

// SubRecord 行政区划记录结构体
type SubRecord struct {
	IsoCode   string            `maxminddb:"iso_code"`
	Names     map[string]string `maxminddb:"names"`
	GeoNameID uint              `maxminddb:"geoname_id"`
}

// GeoRecord 地理位置记录结构体，适用于DBIP格式
type GeoRecord struct {
	GeoNameID uint `maxminddb:"geoname_id"`
	Continent struct {
		Code      string            `maxminddb:"code"`
		Names     map[string]string `maxminddb:"names"`
		GeoNameID uint              `maxminddb:"geoname_id"`
	} `maxminddb:"continent"`
	Country      CountryRecord `maxminddb:"country"`
	City         interface{}   `maxminddb:"city"`
	CityName     string        `maxminddb:"city_name"`
	Subdivisions []SubRecord   `maxminddb:"subdivisions"`
	Location     struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
	AutonomousSystemNumber uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrg    string `maxminddb:"autonomous_system_organization"`
}

// getCityName 获取城市名称
func (r *GeoRecord) getCityName() string {
	if r.City != nil {
		city := parseCity(r.City)
		if name := city.getName(); name != "" {
			return name
		}
	}
	if r.CityName != "" {
		return r.CityName
	}
	return ""
}

// parseCity 解析城市记录为结构体
func parseCity(city interface{}) CityRecord {
	switch v := city.(type) {
	case CityRecord:
		return v
	case map[string]interface{}:
		return cityFromMap(v)
	case string:
		return CityRecord{Name: v}
	default:
		return CityRecord{}
	}
}

// cityFromMap 从map转换为CityRecord结构体
func cityFromMap(m map[string]interface{}) CityRecord {
	var city CityRecord

	if namesRaw, ok := m["names"]; ok {
		if namesMap, ok := namesRaw.(map[string]interface{}); ok {
			city.Names = make(map[string]string)
			for k, val := range namesMap {
				if s, ok := val.(string); ok {
					city.Names[k] = s
				}
			}
		}
	}

	if name, ok := m["name"].(string); ok {
		city.Name = name
	}

	return city
}

// getName 获取城市名称
func (c CityRecord) getName() string {
	if c.Names != nil {
		if name, ok := c.Names["zh-CN"]; ok {
			return name
		}
		if name, ok := c.Names["en"]; ok {
			return name
		}
	}
	if c.Name != "" {
		return c.Name
	}
	return ""
}

// getCountryName 获取国家名称
func (r *GeoRecord) getCountryName() string {
	if r.Country.Names != nil {
		if name, ok := r.Country.Names["zh-CN"]; ok {
			return name
		}
		if name, ok := r.Country.Names["en"]; ok {
			return name
		}
	}
	if r.Country.IsoCode != "" {
		return r.Country.IsoCode
	}
	return ""
}

// getSubdivisionNames 获取行政区划名称列表
func (r *GeoRecord) getSubdivisionNames() []string {
	var names []string

	for _, sub := range r.Subdivisions {
		if sub.Names != nil {
			if name, ok := sub.Names["zh-CN"]; ok {
				names = append(names, name)
			} else if name, ok := sub.Names["en"]; ok {
				names = append(names, name)
			}
		}
	}

	return names
}

// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
func (m *DBIPMMDB) FindFull(query string) *iplocate.IPLocate {
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

	result.Country = record.getCountryName()
	result.Location = formatGeoRecord(record)

	subNames := record.getSubdivisionNames()
	if len(subNames) > 0 {
		result.Province = subNames[0]
	}

	cityName := record.getCityName()
	if cityName != "" {
		result.City = cityName
	}

	if record.AutonomousSystemOrg != "" {
		result.ISP = record.AutonomousSystemOrg
	}

	return result
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (m *DBIPMMDB) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = m.findFullInternal(query)
	}

	return results
}

// findFullInternal 内部查询方法，不加锁
func (m *DBIPMMDB) findFullInternal(query string) *iplocate.IPLocate {
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

	result.Country = record.getCountryName()

	subNames := record.getSubdivisionNames()
	if len(subNames) > 0 {
		result.City = subNames[0]
	}

	cityName := record.getCityName()
	if cityName != "" {
		if result.City != "" {
			result.Area = cityName
		} else {
			result.City = cityName
		}
	}

	if record.AutonomousSystemOrg != "" {
		result.ISP = record.AutonomousSystemOrg
	}

	return result
}

// GetDatabaseInfo 获取数据库信息
func (m *DBIPMMDB) GetDatabaseInfo() *iplocate.DBInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := &iplocate.DBInfo{
		Type:   iplocate.DBTypeDBIPMMDB,
		DbPath: m.dbPath,
	}

	if m.db != nil {
		info.IsIPv4 = m.db.Metadata.IPVersion == 4
		info.IsIPv6 = m.db.Metadata.IPVersion == 6
	}

	return info
}

// Close 关闭数据库连接
func (m *DBIPMMDB) Close() {
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

	countryName := record.getCountryName()
	if countryName != "" {
		parts = append(parts, countryName)
	}

	subNames := record.getSubdivisionNames()
	parts = append(parts, subNames...)

	cityName := record.getCityName()
	if cityName != "" {
		parts = append(parts, cityName)
	}

	if record.AutonomousSystemOrg != "" {
		parts = append(parts, record.AutonomousSystemOrg)
	}

	if record.AutonomousSystemNumber > 0 {
		parts = append(parts, fmt.Sprintf("AS%d", record.AutonomousSystemNumber))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}
