// Package asncsv 提供对 CSV 格式 ASN 数据库的支持
package asncsv

import (
	"encoding/csv"
	"fmt"
	"github.com/winezer0/ipinfo/pkg/iputils"
	"io"
	"math/big"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/winezer0/ipinfo/pkg/asninfo"
)

// ASNRecord CSV中的ASN记录结构体
type ASNRecord struct {
	StartIP                net.IP
	EndIP                  net.IP
	AutonomousSystemNumber uint64
	AutonomousSystemOrg    string
	IsIPv6                 bool
}

// ASNCsvQuerier CSV格式ASN数据库查询器
type ASNCsvQuerier struct {
	mu             sync.RWMutex
	records        []ASNRecord
	records6       []ASNRecord
	loaded         bool
	dbPath         string
	asnIndex       map[uint64][]int // ASN -> 记录索引缓存（懒加载）
	asnIndex6      map[uint64][]int // ASN -> IPv6记录索引缓存（懒加载）
	asnIndexLoaded bool             // ASN索引是否已加载
}

// 确保 ASNCsvQuerier 实现了 asninfo.ASNQuerier 接口
var _ asninfo.ASNQuerier = (*ASNCsvQuerier)(nil)

// NewASNCsvQuerier 创建CSV格式ASN数据库查询器实例
func NewASNCsvQuerier(dbPath string) *ASNCsvQuerier {
	return &ASNCsvQuerier{
		dbPath: dbPath,
	}
}

// Init 初始化数据库连接
func (c *ASNCsvQuerier) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loaded {
		return nil
	}

	if c.dbPath == "" {
		return fmt.Errorf("%w: %s", asninfo.ErrEmptyDatabasePath, c.dbPath)
	}

	if _, err := os.Stat(c.dbPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", asninfo.ErrDatabaseFileNotFound, c.dbPath)
	}

	file, err := os.Open(c.dbPath)
	if err != nil {
		return fmt.Errorf("%w: %w", asninfo.ErrFailedToOpenDatabase, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	records := make([]ASNRecord, 0)
	records6 := make([]ASNRecord, 0)

	if err := loadCSVRecords(reader, &records, &records6); err != nil {
		return err
	}

	sort.Slice(records, func(i, j int) bool {
		return compareIP(records[i].StartIP, records[j].StartIP) < 0
	})

	sort.Slice(records6, func(i, j int) bool {
		return compareIP(records6[i].StartIP, records6[j].StartIP) < 0
	})

	c.records = records
	c.records6 = records6
	c.loaded = true

	return nil
}

// loadCSVRecords 加载CSV记录
func loadCSVRecords(reader *csv.Reader, records, records6 *[]ASNRecord) error {
	firstLine, err := reader.Read()
	if err != nil && err != io.EOF {
		return fmt.Errorf("%w: %w", asninfo.ErrFailedToReadDatabase, err)
	}

	isHeader := false
	if len(firstLine) >= 4 {
		firstField := strings.ToLower(strings.TrimSpace(firstLine[0]))
		if firstField == "start_ip" || firstField == "startip" || firstField == "ip_start" {
			isHeader = true
		}
	}

	if !isHeader && len(firstLine) >= 4 {
		if asnRecord := parseCSVRecord(firstLine); asnRecord != nil {
			if asnRecord.IsIPv6 {
				*records6 = append(*records6, *asnRecord)
			} else {
				*records = append(*records, *asnRecord)
			}
		}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: %w", asninfo.ErrFailedToReadDatabase, err)
		}

		if len(record) < 4 {
			continue
		}

		if asnRecord := parseCSVRecord(record); asnRecord != nil {
			if asnRecord.IsIPv6 {
				*records6 = append(*records6, *asnRecord)
			} else {
				*records = append(*records, *asnRecord)
			}
		}
	}

	return nil
}

// Close 关闭数据库连接
func (c *ASNCsvQuerier) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.records = nil
	c.records6 = nil
	c.asnIndex = nil
	c.asnIndex6 = nil
	c.loaded = false
	c.asnIndexLoaded = false

	return nil
}

// IsInitialized 检查数据库是否已初始化
func (c *ASNCsvQuerier) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.loaded
}

// FindASN 查询单个IP的ASN信息
func (c *ASNCsvQuerier) FindASN(ipStr string) *asninfo.ASNInfo {
	ipVersion := iputils.GetIpVersion(ipStr)
	asnInfo := asninfo.NewASNInfo(ipStr, ipVersion)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return asnInfo
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.loaded {
		return asnInfo
	}

	isIPv6 := ip.To4() == nil
	var records []ASNRecord
	if isIPv6 {
		records = c.records6
	} else {
		records = c.records
	}

	idx := sort.Search(len(records), func(i int) bool {
		return compareIP(records[i].EndIP, ip) >= 0
	})

	if idx < len(records) && compareIP(records[idx].StartIP, ip) <= 0 {
		asnInfo.OrganisationNumber = records[idx].AutonomousSystemNumber
		asnInfo.OrganisationName = records[idx].AutonomousSystemOrg
		asnInfo.FoundASN = true
	}

	return asnInfo
}

// BatchFindASN 批量查询多个IP的ASN信息
func (c *ASNCsvQuerier) BatchFindASN(ips []string) map[string]*asninfo.ASNInfo {
	results := make(map[string]*asninfo.ASNInfo, len(ips))

	for _, ip := range ips {
		results[ip] = c.FindASN(ip)
	}

	return results
}

// ASNToIPRanges 通过ASN号反查所有IP段
func (c *ASNCsvQuerier) ASNToIPRanges(targetASN uint64) ([]*net.IPNet, error) {
	// 懒加载ASN索引
	c.ensureASNIndexLoaded()

	c.mu.RLock()
	defer c.mu.RUnlock()

	var findIPs []*net.IPNet

	indices, exists := c.asnIndex[targetASN]
	if exists {
		for _, idx := range indices {
			record := c.records[idx]
			mask := getIPv4MaskString(record.StartIP, record.EndIP)
			if mask == "" {
				continue
			}
			_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%s", record.StartIP.String(), mask))
			if err == nil {
				findIPs = append(findIPs, ipNet)
			}
		}
	}

	indices6, exists6 := c.asnIndex6[targetASN]
	if exists6 {
		for _, idx := range indices6 {
			record := c.records6[idx]
			mask := getIPv6MaskString(record.StartIP, record.EndIP)
			if mask == "" {
				continue
			}
			_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%s", record.StartIP.String(), mask))
			if err == nil {
				findIPs = append(findIPs, ipNet)
			}
		}
	}

	return findIPs, nil
}

// ensureASNIndexLoaded 懒加载ASN索引
func (c *ASNCsvQuerier) ensureASNIndexLoaded() {
	c.mu.RLock()
	if c.asnIndexLoaded {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.asnIndexLoaded {
		return
	}

	c.asnIndex = make(map[uint64][]int)
	for i, record := range c.records {
		c.asnIndex[record.AutonomousSystemNumber] = append(
			c.asnIndex[record.AutonomousSystemNumber], i)
	}

	c.asnIndex6 = make(map[uint64][]int)
	for i, record := range c.records6 {
		c.asnIndex6[record.AutonomousSystemNumber] = append(
			c.asnIndex6[record.AutonomousSystemNumber], i)
	}

	c.asnIndexLoaded = true
}

// GetDatabaseInfo 获取数据库信息
func (c *ASNCsvQuerier) GetDatabaseInfo() *asninfo.DBInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := &asninfo.DBInfo{
		Type:   asninfo.DBTypeCSV,
		DbPath: c.dbPath,
		IsIPv4: len(c.records) > 0,
		IsIPv6: len(c.records6) > 0,
	}

	return info
}

// parseCSVRecord 解析CSV记录行
func parseCSVRecord(record []string) *ASNRecord {
	if len(record) < 4 {
		return nil
	}

	startIPStr := strings.TrimSpace(record[0])
	endIPStr := strings.TrimSpace(record[1])
	asnStr := strings.TrimSpace(record[2])
	org := strings.TrimSpace(record[3])
	org = strings.Trim(org, "\"")

	asn, err := strconv.ParseUint(asnStr, 10, 64)
	if err != nil {
		return nil
	}

	startIP := net.ParseIP(startIPStr)
	endIP := net.ParseIP(endIPStr)

	if startIP == nil || endIP == nil {
		return nil
	}

	isIPv6 := startIP.To4() == nil

	return &ASNRecord{
		StartIP:                startIP,
		EndIP:                  endIP,
		AutonomousSystemNumber: asn,
		AutonomousSystemOrg:    org,
		IsIPv6:                 isIPv6,
	}
}

// ipToBigInt 将IP地址转换为大整数
func ipToBigInt(ip net.IP) *big.Int {
	if ip.To4() != nil {
		ip = ip.To4()
	} else {
		ip = ip.To16()
	}
	return new(big.Int).SetBytes(ip)
}

// getIPv4MaskString 计算IPv4子网掩码
func getIPv4MaskString(startIP, endIP net.IP) string {
	start := ipToBigInt(startIP)
	end := ipToBigInt(endIP)

	diff := new(big.Int).Sub(end, start)
	diff.Add(diff, big.NewInt(1))

	mask := 32
	for diff.Cmp(big.NewInt(1)) > 0 {
		if diff.Bit(0) == 1 {
			return ""
		}
		diff.Rsh(diff, 1)
		mask--
	}

	return strconv.Itoa(mask)
}

// getIPv6MaskString 计算IPv6子网掩码
func getIPv6MaskString(startIP, endIP net.IP) string {
	start := ipToBigInt(startIP)
	end := ipToBigInt(endIP)

	diff := new(big.Int).Sub(end, start)
	diff.Add(diff, big.NewInt(1))

	mask := 128
	for diff.Cmp(big.NewInt(1)) > 0 {
		if diff.Bit(0) == 1 {
			return ""
		}
		diff.Rsh(diff, 1)
		mask--
	}

	return strconv.Itoa(mask)
}

// compareIP 比较两个IP地址
func compareIP(ip1, ip2 net.IP) int {
	ip1Big := ipToBigInt(ip1)
	ip2Big := ipToBigInt(ip2)
	return ip1Big.Cmp(ip2Big)
}
