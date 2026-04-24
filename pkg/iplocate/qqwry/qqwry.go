package qqwry

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

// Ipv4Location IPv4地理位置数据库管理器
type Ipv4Location struct {
	wry.IPDB[uint32]
	mu     sync.RWMutex // 添加读写锁保护并发访问
	dbPath string       // 数据库文件路径
}

// 确保 Ipv4Location 实现了 ipinfo.IPInfo 接口
var _ iplocate.IPInfo = (*Ipv4Location)(nil)

// NewQQWryDB 从文件路径创建新的IPv4地理位置数据库管理器
func NewQQWryDB(filePath string) (*Ipv4Location, error) {
	if filePath == "" {
		return nil, fmt.Errorf("IP数据库[%v]文件路径为空", filePath)
	}

	db := &Ipv4Location{
		dbPath: filePath,
	}

	if err := db.Init(); err != nil {
		return nil, err
	}

	return db, nil
}

// Init 初始化数据库连接
func (db *Ipv4Location) Init() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data != nil {
		return nil
	}

	fileData, err := utils.ReadFileBytes(db.dbPath)
	if err != nil {
		return err
	}

	if !checkIPv4File(fileData) {
		return fmt.Errorf("IP数据库[%v]内容存在错误", db.dbPath)
	}

	header := fileData[0:8]
	start := binary.LittleEndian.Uint32(header[:4])
	end := binary.LittleEndian.Uint32(header[4:])

	db.Data = fileData
	db.OffLen = 3
	db.IPLen = 4
	db.IPCnt = (end-start)/7 + 1
	db.IdxStart = start
	db.IdxEnd = end

	return nil
}

// IsInitialized 检查数据库是否已初始化
func (db *Ipv4Location) IsInitialized() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.Data != nil
}

// find 内部查询方法，返回详细结果
func (db *Ipv4Location) find(query string) (result *wry.Result, err error) {
	// 验证IP地址
	ip := net.ParseIP(query)
	if ip == nil {
		return nil, errors.New("无效的IPv4地址")
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return nil, errors.New("无效的IPv4地址")
	}

	// 转换为uint32进行查询
	ip4uint := binary.BigEndian.Uint32(ip4)

	// 搜索索引
	offset := db.SearchIndexV4(ip4uint)
	if offset <= 0 {
		return nil, errors.New("查询无效")
	}

	// 解析结果
	reader := wry.NewReader(db.Data)
	reader.Parse(offset + 4)
	return reader.Result.DecodeGBK(), nil
}

// FindFull 查询单个IP地址的地理位置信息,返回更结构化的地址信息
func (db *Ipv4Location) FindFull(query string) *iplocate.IPLocate {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := &iplocate.IPLocate{
		IP:      query,
		Version: 4,
	}

	wryResult, err := db.find(query)
	if err != nil || wryResult == nil {
		return result
	}

	return parseWryResultToIPLocate(result, wryResult)
}

// BatchFindFull 批量查询多个IP地址,返回更结构化的地址信息
func (db *Ipv4Location) BatchFindFull(queries []string) map[string]*iplocate.IPLocate {
	results := make(map[string]*iplocate.IPLocate, len(queries))

	for _, query := range queries {
		results[query] = db.FindFull(query)
	}

	return results
}

// findFullInternal 内部查询方法，不加锁
func (db *Ipv4Location) findFullInternal(query string) *iplocate.IPLocate {
	result := &iplocate.IPLocate{
		IP:      query,
		Version: 4,
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

// checkIPv4File 检查IPv4数据库文件的有效性
func checkIPv4File(data []byte) bool {
	// 检查最小长度
	if len(data) < 8 {
		return false
	}

	// 解析头部信息
	header := data[0:8]
	start := binary.LittleEndian.Uint32(header[:4])
	end := binary.LittleEndian.Uint32(header[4:])

	// 验证索引范围
	if start >= end {
		return false
	}

	// 验证数据完整性
	if uint32(len(data)) < end+7 {
		return false
	}

	return true
}

// GetDatabaseInfo 获取数据库信息
func (db *Ipv4Location) GetDatabaseInfo() *iplocate.DBInfo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return &iplocate.DBInfo{
		Type:   iplocate.DBTypeQQWry,
		DbPath: db.dbPath,
		IsIPv4: true,
		IsIPv6: false,
	}
}

// Close 关闭数据库连接（清理资源）
func (db *Ipv4Location) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()

	// 防止重复关闭
	if db.Data == nil {
		return
	}

	// 清理数据
	db.Data = nil
}
