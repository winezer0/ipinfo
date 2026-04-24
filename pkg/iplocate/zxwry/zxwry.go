package zxwry

// forked from https://github.com/zu1k/nali
// ipv6db数据使用http://ip.zxinc.org的免费离线数据（更新到2021年）

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/winezer0/ipinfo/pkg/iplocate"
	"github.com/winezer0/ipinfo/pkg/iplocate/wry"
	"github.com/winezer0/ipinfo/pkg/utils"
	"net"
	"sync"
)

// Ipv6Location IPv6地理位置数据库管理器
type Ipv6Location struct {
	wry.IPDB[uint64]
	mu     sync.RWMutex // 添加读写锁保护并发访问
	dbPath string       // 数据库文件路径
}

// 确保 Ipv6Location 实现了 ipinfo.IPInfo 接口
var _ iplocate.IPInfo = (*Ipv6Location)(nil)

// NewZXWryDB 从文件路径创建新的IPv6地理位置数据库管理器
func NewZXWryDB(filePath string) (*Ipv6Location, error) {
	if filePath == "" {
		return nil, fmt.Errorf("IP数据库[%v]文件路径为空", filePath)
	}

	db := &Ipv6Location{
		dbPath: filePath,
	}

	if err := db.Init(); err != nil {
		return nil, err
	}

	return db, nil
}

// Init 初始化数据库连接
func (db *Ipv6Location) Init() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data != nil {
		return nil
	}

	fileData, err := utils.ReadFileBytes(db.dbPath)
	if err != nil {
		return err
	}

	if !checkIPv6File(fileData) {
		return fmt.Errorf("IP数据库[%v]内容存在错误", db.dbPath)
	}

	header := fileData[:24]
	offLen := header[6]
	ipLen := header[7]

	start := binary.LittleEndian.Uint64(header[16:24])
	counts := binary.LittleEndian.Uint64(header[8:16])
	end := start + counts*11

	db.Data = fileData
	db.OffLen = offLen
	db.IPLen = ipLen
	db.IPCnt = counts
	db.IdxStart = start
	db.IdxEnd = end

	return nil
}

// IsInitialized 检查数据库是否已初始化
func (db *Ipv6Location) IsInitialized() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.Data != nil
}

// find 内部查询方法，返回详细结果
func (db *Ipv6Location) find(query string) (result *wry.Result, err error) {
	// 验证IP地址
	ip := net.ParseIP(query)
	if ip == nil {
		return nil, errors.New("无效的IPv6地址")
	}

	ip6 := ip.To16()
	if ip6 == nil {
		return nil, errors.New("无效的IPv6地址")
	}

	// 取前8字节进行查询
	ip6 = ip6[:8]
	ipu64 := binary.BigEndian.Uint64(ip6)

	// 搜索索引
	offset := db.SearchIndexV6(ipu64)
	if offset <= 0 {
		return nil, errors.New("查询无效")
	}

	// 解析结果
	reader := wry.NewReader(db.Data)
	reader.Parse(offset)
	return &reader.Result, nil
}

// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
func (db *Ipv6Location) FindFull(query string) *iplocate.IPLocate {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := &iplocate.IPLocate{
		IP:      query,
		Version: 6,
	}

	wryResult, err := db.find(query)
	if err != nil || wryResult == nil {
		return result
	}

	return parseWryResultToIPLocate(result, wryResult)
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (db *Ipv6Location) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = db.FindFull(query)
	}

	return results
}

// findFullInternal 内部查询方法，不加锁
func (db *Ipv6Location) findFullInternal(query string) *iplocate.IPLocate {
	result := &iplocate.IPLocate{
		IP:      query,
		Version: 6,
	}

	wryResult, err := db.find(query)
	if err != nil || wryResult == nil {
		return result
	}

	return parseWryResultToIPLocate(result, wryResult)
}

// parseWryResultToIPLocate 解析wry.Result为IPLocate
func parseWryResultToIPLocate(result *iplocate.IPLocate, wryResult *wry.Result) *iplocate.IPLocate {
	return wry.ParseWryResultToIPLocate(result, wryResult)
}

// formatLocationResult 清理和格式化地理位置结果
func formatLocationResult(country string) string {
	return wry.FormatLocationResult(country)
}

// GetDatabaseInfo 获取数据库信息
func (db *Ipv6Location) GetDatabaseInfo() *iplocate.DBInfo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return &iplocate.DBInfo{
		Type:   iplocate.DBTypeZXWry,
		DbPath: db.dbPath,
		IsIPv4: false,
		IsIPv6: true,
	}
}

// Close 关闭数据库连接（清理资源）
func (db *Ipv6Location) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()

	// 防止重复关闭
	if db.Data == nil {
		return
	}

	// 清理数据
	db.Data = nil
}

// checkIPv6File 检查IPv6数据库文件的有效性
func checkIPv6File(data []byte) bool {
	// 检查最小长度
	if len(data) < 4 {
		return false
	}

	// 检查文件标识
	if string(data[:4]) != "IPDB" {
		return false
	}

	// 检查头部长度
	if len(data) < 24 {
		return false
	}

	// 解析头部信息
	header := data[:24]
	start := binary.LittleEndian.Uint64(header[16:24])
	counts := binary.LittleEndian.Uint64(header[8:16])
	end := start + counts*11

	// 验证索引范围
	if start >= end {
		return false
	}

	// 验证数据完整性
	if uint64(len(data)) < end {
		return false
	}

	return true
}
