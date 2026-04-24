package qqwry

import (
	"encoding/binary"
	"github.com/winezer0/ipinfo/pkg/utils"
	"net"
	"path/filepath"
	"runtime"
	"testing"
)

func getTestDBPath(dbname string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..", "..")
	return filepath.Join(projectRoot, "assets", dbname)
}

func TestIpv4Location_FindFull(t *testing.T) {
	// 集成测试：测试完整的查询流程
	dbpath := getTestDBPath("qqwry.dat")
	db, err := NewQQWryDB(dbpath)
	if err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer db.Close()

	// 测试一些常见的IP地址
	testIPs := []string{
		"8.8.8.8",
		"119.29.29.52",
		"114.114.114.114",
		"223.5.5.5",
		"1.1.1.1",
		"208.67.222.222",
		"266.67.222.222",
	}

	for _, ip := range testIPs {
		t.Run(ip, func(t *testing.T) {
			result := db.FindFull(ip)

			// 记录结果用于调试
			t.Logf("查询IP: %s -> 位置: %s, 结构化: Country=%s, Province=%s, City=%s, ISP=%s",
				ip, result.Location, result.Country, result.Province, result.City, result.ISP)
		})
	}
}

func TestIpv4Location_BatchFindFull(t *testing.T) {
	dbpath := getTestDBPath("qqwry.dat")
	db, err := NewQQWryDB(dbpath)
	if err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"119.29.29.52",
		"114.114.114.114",
		"invalid_ip",
		"2001:db8::1",
	}

	results := db.BatchFindFull(testIPs)

	if len(results) != len(testIPs) {
		t.Errorf("期望结果数量为 %d，但得到了 %d", len(testIPs), len(results))
	}

	for ip, result := range results {
		t.Logf("批量查询 - IP: %s -> 位置: %s, Country=%s", ip, result.Location, result.Country)
	}
}

func TestIpv4Location_GetDatabaseInfo(t *testing.T) {
	dbpath := getTestDBPath("qqwry.dat")
	db, err := NewQQWryDB(dbpath)
	if err != nil {
		t.Skipf("跳过测试，因为无法加载数据库: %v", err)
	}
	defer db.Close()

	info := db.GetDatabaseInfo()

	if info.Type != "qqwry" {
		t.Errorf("数据库类型错误: %v, 期望 qqwry", info.Type)
	}

	if !info.IsIPv4 {
		t.Error("数据库应该支持 IPv4")
	}

	if info.IsIPv6 {
		t.Error("数据库不应该支持 IPv6")
	}

	t.Logf("数据库信息: %+v", info)
}

// TestAnalyzeQQWryIPv6Support 分析qqwry.dat数据库是否支持IPv6
func TestAnalyzeQQWryIPv6Support(t *testing.T) {
	dbpath := getTestDBPath("qqwry.dat")

	// 读取数据库文件
	fileData, err := utils.ReadFileBytes(dbpath)
	if err != nil {
		t.Fatalf("读取数据库文件失败: %v", err)
	}

	t.Logf("数据库文件大小: %d 字节 (%.2f MB)", len(fileData), float64(len(fileData))/1024/1024)

	// 分析数据库头部信息
	analyzeDatabaseHeader(t, fileData)

	// 测试IPv6地址查询
	testIPv6Queries(t, fileData)

	// 测试数据库格式验证
	testDatabaseFormatValidation(t, fileData)
}

// analyzeDatabaseHeader 分析数据库头部信息
func analyzeDatabaseHeader(t *testing.T, data []byte) {
	t.Run("分析数据库头部", func(t *testing.T) {
		if len(data) < 8 {
			t.Fatalf("数据库文件太小，至少需要8字节头部")
		}

		// 解析IPv4头部信息
		header := data[0:8]
		start := binary.LittleEndian.Uint32(header[:4])
		end := binary.LittleEndian.Uint32(header[4:])
		ipCount := (end - start) / 7

		t.Logf("数据库头部信息:")
		t.Logf("  - 索引起始位置: %d (0x%X)", start, start)
		t.Logf("  - 索引结束位置: %d (0x%X)", end, end)
		t.Logf("  - IP记录数量: %d", ipCount)
		t.Logf("  - 每条索引长度: 7 字节")

		// 检查是否有IPv6标识
		hasIPv6Signature := false
		if len(data) >= 24 {
			// 检查前4字节是否为"IPDB"标识（IPv6数据库通常使用此标识）
			signature := string(data[:4])
			if signature == "IPDB" {
				hasIPv6Signature = true
				t.Logf("  - 检测到IPDB标识，可能支持IPv6")
			} else {
				t.Logf("  - 未检测到IPDB标识，可能仅支持IPv4")
			}
		}

		// 分析索引结构
		t.Logf("数据库结构分析:")
		t.Logf("  - 使用32位IP地址 (uint32)，仅支持IPv4")
		t.Logf("  - 不支持128位IPv6地址 (uint64)")

		// 检查数据库类型
		if !hasIPv6Signature {
			t.Logf("  - 数据库类型: QQWry IPv4格式")
			t.Logf("  - 结论: 此数据库不支持IPv6")
		}
	})
}

// testIPv6Queries 测试IPv6地址查询
func testIPv6Queries(t *testing.T, data []byte) {
	t.Run("测试IPv6地址查询", func(t *testing.T) {
		db, err := NewQQWryDB(getTestDBPath("qqwry.dat"))
		if err != nil {
			t.Skipf("跳过测试，因为无法加载数据库: %v", err)
		}
		defer db.Close()

		// 测试常见的IPv6地址
		ipv6Addresses := []string{
			"::1",                  // localhost
			"2001:db8::1",          // 文档示例
			"2001:4860:4860::8888", // Google DNS
			"2606:4700:4700::1111", // Cloudflare DNS
			"fe80::1",              // 链路本地地址
			"2400:3200::1",         // 阿里云DNS (IPv6)
			"240C::6666",           // 中国电信DNS (IPv6)
		}

		t.Logf("测试IPv6地址查询:")
		for _, ipv6 := range ipv6Addresses {
			result := db.FindFull(ipv6)
			ip := net.ParseIP(ipv6)

			if ip == nil {
				t.Logf("  - %s: 无效的IPv6地址", ipv6)
			} else if ip.To4() != nil {
				t.Logf("  - %s: 这是IPv4地址", ipv6)
			} else {
				if result.Location == "" {
					t.Logf("  - %s: 未找到结果 (数据库不支持IPv6)", ipv6)
				} else {
					t.Logf("  - %s: %s", ipv6, result.Location)
				}
			}
		}

		// 验证IPv6地址会被正确识别但不支持
		t.Logf("\n验证IPv6地址处理:")
		for _, ipv6 := range []string{"2001:db8::1", "::1"} {
			ip := net.ParseIP(ipv6)
			if ip != nil {
				isIPv4 := ip.To4() != nil
				t.Logf("  - %s: To4()=%v (false表示纯IPv6)", ipv6, isIPv4)
			}
		}
	})
}

// testDatabaseFormatValidation 测试数据库格式验证
func testDatabaseFormatValidation(t *testing.T, data []byte) {
	t.Run("数据库格式验证", func(t *testing.T) {
		// 测试checkIPv4File函数
		isValid := checkIPv4File(data)
		t.Logf("checkIPv4File验证结果: %v", isValid)

		// 分析数据库特征
		t.Logf("数据库特征分析:")
		t.Logf("  - 文件大小: %d 字节", len(data))
		t.Logf("  - 最小要求: 8 字节")
		t.Logf("  - 满足最小大小要求: %v", len(data) >= 8)

		if len(data) >= 8 {
			header := data[0:8]
			start := binary.LittleEndian.Uint32(header[:4])
			end := binary.LittleEndian.Uint32(header[4:])

			t.Logf("  - 索引范围有效: %v", start < end)
			t.Logf("  - 数据完整性: %v", uint32(len(data)) >= end+7)

			// 判断是否支持IPv6的关键因素
			t.Logf("\nIPv6支持性判断:")
			t.Logf("  - 使用32位索引 (uint32): 仅支持IPv4")
			t.Logf("  - IPv6需要64位索引 (uint64): 不支持")
			t.Logf("  - 结论: qqwry.dat 是纯IPv4数据库，不支持IPv6")
		}
	})
}

// TestQQWryFindAndFindFull 测试Find和FindFull方法对比
func TestQQWryFindAndFindFull(t *testing.T) {
	dbPath := getTestDBPath("qqwry.dat")

	db, err := NewQQWryDB(dbPath)
	if err != nil {
		t.Skipf("跳过测试，数据库文件加载失败: %v", err)
	}
	defer db.Close()

	testIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
		"223.5.5.5",
		"202.108.22.5",
		"220.181.38.148",
	}

	for _, ip := range testIPs {
		findFullResult := db.FindFull(ip)

		t.Logf("=== IP: %s ===", ip)
		t.Logf("FindFull 结果: Location=%s, Country=%s, Province=%s, City=%s, Area=%s, ISP=%s",
			findFullResult.Location, findFullResult.Country, findFullResult.Province, findFullResult.City, findFullResult.Area, findFullResult.ISP)
	}
}
