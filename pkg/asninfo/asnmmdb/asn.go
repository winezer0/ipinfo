package asnmmdb

import (
	"fmt"
	"github.com/winezer0/ipinfo/pkg/asninfo"
	"github.com/winezer0/ipinfo/pkg/iputils"
	"net"
	"os"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

// MMDBConfig 数据库配置结构
type MMDBConfig struct {
	AsnIpvxDb            string
	MaxConcurrentQueries int
}

// MMDBManager 数据库管理器
type MMDBManager struct {
	config         *MMDBConfig
	mmDb           *maxminddb.Reader
	queryChan      chan struct{}
	mu             sync.RWMutex
	asnIndex       map[uint64][]int // ASN -> 网络索引缓存（懒加载）
	asnIndexLoaded bool             // ASN索引是否已加载
}

// 确保 MMDBManager 实现了 asninfo.ASNQuerier 接口
var _ asninfo.ASNQuerier = (*MMDBManager)(nil)

// NewMMDBManager 创建新的数据库管理器
func NewMMDBManager(config *MMDBConfig) (*MMDBManager, error) {
	if config == nil {
		return nil, fmt.Errorf("%w", asninfo.ErrEmptyDatabasePath)
	}
	if config.AsnIpvxDb == "" {
		return nil, fmt.Errorf("%w", asninfo.ErrEmptyDatabasePath)
	}
	if config.MaxConcurrentQueries <= 0 {
		config.MaxConcurrentQueries = 100
	}
	return &MMDBManager{
		config:    config,
		queryChan: make(chan struct{}, config.MaxConcurrentQueries),
	}, nil
}

// ASNRecord 定义结构体映射MMDB中的ASN数据结构
type ASNRecord struct {
	AutonomousSystemNumber       uint64 `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// InitMMDBConn 初始化 MaxMind ASN 数据库连接
func (m *MMDBManager) InitMMDBConn() error {
	return m.Init()
}

// Init 初始化数据库连接（实现ASNQuerier接口）
func (m *MMDBManager) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mmDb != nil {
		return nil
	}

	if _, err := os.Stat(m.config.AsnIpvxDb); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", asninfo.ErrDatabaseFileNotFound, m.config.AsnIpvxDb)
	}

	conn, err := maxminddb.Open(m.config.AsnIpvxDb)
	if err != nil {
		return fmt.Errorf("%w [%s]: %w", asninfo.ErrFailedToOpenDatabase, m.config.AsnIpvxDb, err)
	}

	m.mmDb = conn
	return nil
}

// Close 关闭数据库连接
func (m *MMDBManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mmDb != nil {
		if err := m.mmDb.Close(); err != nil {
			return fmt.Errorf("%w: %w", asninfo.ErrFailedToCloseDatabase, err)
		}
		m.mmDb = nil
	}
	return nil
}

// IsInitialized 检查数据库是否已初始化
func (m *MMDBManager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mmDb != nil
}

// FindASN 查询单个IP的ASN信息
func (m *MMDBManager) FindASN(ipStr string) *asninfo.ASNInfo {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &asninfo.ASNInfo{
			IP:        ipStr,
			IPVersion: iputils.GetIpVersion(ipStr),
			FoundASN:  false,
		}
	}

	// 获取查询许可
	select {
	case m.queryChan <- struct{}{}:
		defer func() { <-m.queryChan }()
	default:
		return &asninfo.ASNInfo{
			IP:        ipStr,
			IPVersion: iputils.GetIpVersion(ipStr),
			FoundASN:  false,
		}
	}

	ipVersion := iputils.GetIpVersion(ipStr)
	asnInfo := asninfo.NewASNInfo(ipStr, ipVersion)

	m.mu.RLock()
	reader := m.mmDb
	m.mu.RUnlock()

	// 如果数据库未初始化，返回空结果
	if reader == nil {
		return asnInfo
	}

	var asnRecord ASNRecord
	if err := reader.Lookup(ip, &asnRecord); err != nil {
		return asnInfo
	}

	if asnRecord.AutonomousSystemNumber > 0 {
		asnInfo.OrganisationNumber = asnRecord.AutonomousSystemNumber
		asnInfo.OrganisationName = asnRecord.AutonomousSystemOrganization
		asnInfo.FoundASN = true
	}

	return asnInfo
}

// ASNToIPRanges 通过ASN号反查所有IP段
func (m *MMDBManager) ASNToIPRanges(targetASN uint64) ([]*net.IPNet, error) {
	m.ensureASNIndexLoaded()

	m.mu.RLock()
	defer m.mu.RUnlock()

	indices, exists := m.asnIndex[targetASN]
	if !exists {
		return nil, nil
	}

	var findIPs []*net.IPNet
	networks := m.mmDb.Networks()
	currentIdx := 0
	targetIdx := 0
	for networks.Next() {
		if targetIdx >= len(indices) {
			break
		}
		if indices[targetIdx] == currentIdx {
			var record ASNRecord
			ipNet, err := networks.Network(&record)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", asninfo.ErrFailedToParseNetwork, err)
			}
			findIPs = append(findIPs, ipNet)
			targetIdx++
		} else {
			networks.Network(&ASNRecord{})
		}
		currentIdx++
	}

	if err := networks.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", asninfo.ErrFailedToIterateNetwork, err)
	}

	return findIPs, nil
}

// ensureASNIndexLoaded 懒加载ASN索引
func (m *MMDBManager) ensureASNIndexLoaded() {
	m.mu.RLock()
	if m.asnIndexLoaded {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.asnIndexLoaded {
		return
	}

	m.asnIndex = make(map[uint64][]int)
	idx := 0
	networks := m.mmDb.Networks()
	for networks.Next() {
		var record ASNRecord
		_, err := networks.Network(&record)
		if err != nil {
			idx++
			continue
		}
		if record.AutonomousSystemNumber > 0 {
			m.asnIndex[record.AutonomousSystemNumber] = append(
				m.asnIndex[record.AutonomousSystemNumber], idx)
		}
		idx++
	}
	m.asnIndexLoaded = true
}

// BatchFindASN 批量查询多个IP的ASN信息
func (m *MMDBManager) BatchFindASN(ips []string) map[string]*asninfo.ASNInfo {
	results := make(map[string]*asninfo.ASNInfo, len(ips))

	for _, ip := range ips {
		results[ip] = m.FindASN(ip)
	}

	return results
}

// GetDatabaseInfo 获取数据库信息
func (m *MMDBManager) GetDatabaseInfo() *asninfo.DBInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := &asninfo.DBInfo{
		Type:   asninfo.DBTypeMMDB,
		DbPath: m.config.AsnIpvxDb,
	}

	if m.mmDb != nil {
		info.IsIPv4 = m.mmDb.Metadata.IPVersion == 4 || m.mmDb.Metadata.IPVersion == 0
		info.IsIPv6 = m.mmDb.Metadata.IPVersion == 6 || m.mmDb.Metadata.IPVersion == 0
	}

	return info
}
